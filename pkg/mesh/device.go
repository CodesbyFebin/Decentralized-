// Package mesh runs the dh-mesh: a userspace WireGuard device
// (wireguard-go over a gVisor netstack, no root or kernel module needed)
// plus SWIM gossip (hashicorp/memberlist) carried inside the tunnel.
//
// WireGuard provides authenticated, encrypted transport and cryptokey
// routing: a packet from mesh address X was sent by the holder of the
// WireGuard key bound to X. Bindings between dh1 identities and WireGuard
// keys are signed by the host's own identity key, so the control plane
// cannot substitute a key for a host it does not control. Gossip provides
// membership and liveness observations. Neither is an authorization
// authority.
package mesh

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/curve25519"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

// Key is a WireGuard X25519 key pair.
type Key struct {
	Private [32]byte
	Public  [32]byte
}

// PublicString is the standard base64 WireGuard encoding.
func (k Key) PublicString() string { return base64.StdEncoding.EncodeToString(k.Public[:]) }

// LoadOrCreateKey keeps the WireGuard private key in path (0600).
func LoadOrCreateKey(path string) (Key, error) {
	var k Key
	if b, err := os.ReadFile(path); err == nil {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
		if err != nil || len(raw) != 32 {
			return k, errors.New("mesh: bad wireguard key file")
		}
		copy(k.Private[:], raw)
	} else {
		if _, err := rand.Read(k.Private[:]); err != nil {
			return k, err
		}
		k.Private[0] &= 248
		k.Private[31] = (k.Private[31] & 127) | 64
		if err := os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(k.Private[:])+"\n"), 0o600); err != nil {
			return k, err
		}
	}
	pub, err := curve25519.X25519(k.Private[:], curve25519.Basepoint)
	if err != nil {
		return k, err
	}
	copy(k.Public[:], pub)
	return k, nil
}

// PeerConfig is one desired WireGuard peer.
type PeerConfig struct {
	Node     string
	WGPub    string // base64
	Endpoint string // host:port (UDP)
	MeshIP   string
}

// PeerStats is measured device state for one peer.
type PeerStats struct {
	Node          string
	WGPub         string
	Endpoint      string
	LastHandshake time.Time
	RxBytes       int64
	TxBytes       int64
}

// Device is a running userspace WireGuard interface.
type Device struct {
	key    Key
	ip     netip.Addr
	port   int
	dev    *device.Device
	tnet   *netstack.Net
	mu     sync.Mutex
	peers  map[string]PeerConfig // by WGPub
	closed bool
}

// Start creates the device, binds UDP listenPort on all interfaces and
// brings it up with address meshIP.
func Start(key Key, meshIP string, listenPort int) (*Device, error) {
	ip, err := netip.ParseAddr(meshIP)
	if err != nil {
		return nil, fmt.Errorf("mesh: address %q: %w", meshIP, err)
	}
	tun, tnet, err := netstack.CreateNetTUN([]netip.Addr{ip}, nil, 1420)
	if err != nil {
		return nil, err
	}
	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(logLevel(), "wg "))
	cfg := fmt.Sprintf("private_key=%s\nlisten_port=%d\n", hex.EncodeToString(key.Private[:]), listenPort)
	if err := dev.IpcSet(cfg); err != nil {
		dev.Close()
		return nil, fmt.Errorf("mesh: configure device: %w", err)
	}
	if err := dev.Up(); err != nil {
		dev.Close()
		return nil, err
	}
	return &Device{key: key, ip: ip, port: listenPort, dev: dev, tnet: tnet, peers: map[string]PeerConfig{}}, nil
}

// IP returns the mesh address.
func (d *Device) IP() string { return d.ip.String() }

// ListenPort returns the UDP port.
func (d *Device) ListenPort() int { return d.port }

// PublicKey returns the base64 WireGuard public key.
func (d *Device) PublicKey() string { return d.key.PublicString() }

// SetPeers reconciles the peer set incrementally so existing sessions are
// not torn down.
func (d *Device) SetPeers(want []PeerConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return errors.New("mesh: device closed")
	}
	desired := map[string]PeerConfig{}
	for _, p := range want {
		if p.WGPub == "" || p.WGPub == d.key.PublicString() {
			continue
		}
		desired[p.WGPub] = p
	}
	var b strings.Builder
	for pub := range d.peers {
		if _, keep := desired[pub]; !keep {
			hexPub, err := b64ToHex(pub)
			if err != nil {
				continue
			}
			fmt.Fprintf(&b, "public_key=%s\nremove=true\n", hexPub)
		}
	}
	keys := make([]string, 0, len(desired))
	for k := range desired {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, pub := range keys {
		p := desired[pub]
		cur, had := d.peers[pub]
		if had && cur.Endpoint == p.Endpoint && cur.MeshIP == p.MeshIP {
			continue
		}
		hexPub, err := b64ToHex(pub)
		if err != nil {
			return fmt.Errorf("mesh: peer %s key: %w", p.Node, err)
		}
		fmt.Fprintf(&b, "public_key=%s\n", hexPub)
		if p.Endpoint != "" {
			addr, err := net.ResolveUDPAddr("udp", p.Endpoint)
			if err != nil {
				return fmt.Errorf("mesh: peer %s endpoint %q: %w", p.Node, p.Endpoint, err)
			}
			fmt.Fprintf(&b, "endpoint=%s\n", addr.String())
		}
		// Only one side of each pair sends the startup keepalive, so two
		// devices configured at the same instant do not cross handshake
		// initiations. Data traffic still triggers handshakes both ways.
		keepalive := 0
		if d.key.PublicString() < pub {
			keepalive = 5
		}
		fmt.Fprintf(&b, "replace_allowed_ips=true\nallowed_ip=%s/32\npersistent_keepalive_interval=%d\n", p.MeshIP, keepalive)
	}
	if b.Len() > 0 {
		if err := d.dev.IpcSet(b.String()); err != nil {
			return fmt.Errorf("mesh: set peers: %w", err)
		}
	}
	d.peers = desired
	return nil
}

// Stats reads measured peer state from the device.
func (d *Device) Stats() []PeerStats {
	d.mu.Lock()
	byPub := map[string]string{}
	for pub, p := range d.peers {
		byPub[pub] = p.Node
	}
	d.mu.Unlock()
	out, err := d.dev.IpcGet()
	if err != nil {
		return nil
	}
	var res []PeerStats
	var cur *PeerStats
	var sec, nsec int64
	flush := func() {
		if cur != nil {
			if sec > 0 {
				cur.LastHandshake = time.Unix(sec, nsec)
			}
			res = append(res, *cur)
		}
		sec, nsec = 0, 0
	}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		k, v, found := strings.Cut(sc.Text(), "=")
		if !found {
			continue
		}
		switch k {
		case "public_key":
			flush()
			raw, _ := hex.DecodeString(v)
			pub := base64.StdEncoding.EncodeToString(raw)
			cur = &PeerStats{WGPub: pub, Node: byPub[pub]}
		case "endpoint":
			if cur != nil {
				cur.Endpoint = v
			}
		case "last_handshake_time_sec":
			sec, _ = strconv.ParseInt(v, 10, 64)
		case "last_handshake_time_nsec":
			nsec, _ = strconv.ParseInt(v, 10, 64)
		case "rx_bytes":
			if cur != nil {
				cur.RxBytes, _ = strconv.ParseInt(v, 10, 64)
			}
		case "tx_bytes":
			if cur != nil {
				cur.TxBytes, _ = strconv.ParseInt(v, 10, 64)
			}
		}
	}
	flush()
	return res
}

// DialContext dials through the tunnel.
func (d *Device) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return d.tnet.DialContext(ctx, network, addr)
}

// ListenTCP listens on the mesh address.
func (d *Device) ListenTCP(port int) (net.Listener, error) {
	return d.tnet.ListenTCP(&net.TCPAddr{IP: net.IP(d.ip.AsSlice()), Port: port})
}

// HTTPClient returns a client whose connections travel inside the tunnel.
func (d *Device) HTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:         d.DialContext,
			MaxIdleConnsPerHost: 4,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  true,
		},
	}
}

// Net exposes the netstack (gossip transport).
func (d *Device) Net() *netstack.Net { return d.tnet }

// Close tears the device down.
func (d *Device) Close() {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true
	d.mu.Unlock()
	d.dev.Close()
}

func b64ToHex(s string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(raw) != 32 {
		return "", errors.New("bad wireguard public key")
	}
	return hex.EncodeToString(raw), nil
}

func logLevel() int {
	if os.Getenv("DH_WG_DEBUG") != "" {
		return device.LogLevelVerbose
	}
	return device.LogLevelSilent
}
