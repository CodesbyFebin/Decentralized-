package mesh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

// GossipPort is the SWIM port inside the mesh.
const GossipPort = 7946

// nsTransport carries memberlist packets and streams over the netstack, so
// gossip only ever travels inside the WireGuard tunnel.
type nsTransport struct {
	tnet     *netstack.Net
	ip       net.IP
	port     int
	udp      net.PacketConn
	tcp      net.Listener
	packetCh chan *memberlist.Packet
	streamCh chan net.Conn
	done     chan struct{}
	once     sync.Once
}

func newNSTransport(tnet *netstack.Net, ip string, port int) (*nsTransport, error) {
	nip := net.ParseIP(ip)
	udp, err := tnet.ListenUDP(&net.UDPAddr{IP: nip, Port: port})
	if err != nil {
		return nil, fmt.Errorf("gossip udp: %w", err)
	}
	tcp, err := tnet.ListenTCP(&net.TCPAddr{IP: nip, Port: port})
	if err != nil {
		udp.Close()
		return nil, fmt.Errorf("gossip tcp: %w", err)
	}
	t := &nsTransport{tnet: tnet, ip: nip, port: port, udp: udp, tcp: tcp,
		packetCh: make(chan *memberlist.Packet, 256), streamCh: make(chan net.Conn, 16), done: make(chan struct{})}
	go t.readPackets()
	go t.acceptStreams()
	return t, nil
}

func (t *nsTransport) readPackets() {
	buf := make([]byte, 65536)
	for {
		n, addr, err := t.udp.ReadFrom(buf)
		if err != nil {
			select {
			case <-t.done:
				return
			default:
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}
		msg := make([]byte, n)
		copy(msg, buf[:n])
		select {
		case t.packetCh <- &memberlist.Packet{Buf: msg, From: addr, Timestamp: time.Now()}:
		case <-t.done:
			return
		}
	}
}

func (t *nsTransport) acceptStreams() {
	for {
		c, err := t.tcp.Accept()
		if err != nil {
			select {
			case <-t.done:
				return
			default:
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}
		select {
		case t.streamCh <- c:
		case <-t.done:
			c.Close()
			return
		}
	}
}

func (t *nsTransport) FinalAdvertiseAddr(string, int) (net.IP, int, error) { return t.ip, t.port, nil }

func (t *nsTransport) WriteTo(b []byte, addr string) (time.Time, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return time.Time{}, err
	}
	port, _ := strconv.Atoi(portStr)
	_, err = t.udp.WriteTo(b, &net.UDPAddr{IP: net.ParseIP(host), Port: port})
	return time.Now(), err
}

func (t *nsTransport) PacketCh() <-chan *memberlist.Packet { return t.packetCh }

func (t *nsTransport) DialTimeout(addr string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return t.tnet.DialContext(ctx, "tcp", addr)
}

func (t *nsTransport) StreamCh() <-chan net.Conn { return t.streamCh }

func (t *nsTransport) Shutdown() error {
	t.once.Do(func() {
		close(t.done)
		t.udp.Close()
		t.tcp.Close()
	})
	return nil
}

// Gossip is SWIM membership inside the mesh.
type Gossip struct {
	ml    *memberlist.Memberlist
	trans *nsTransport
}

// StartGossip starts memberlist named name on the device's mesh address.
func StartGossip(d *Device, name string) (*Gossip, error) {
	t, err := newNSTransport(d.Net(), d.IP(), GossipPort)
	if err != nil {
		return nil, err
	}
	cfg := memberlist.DefaultLANConfig()
	cfg.Name = name
	cfg.Transport = t
	cfg.BindAddr = d.IP()
	cfg.BindPort = GossipPort
	cfg.AdvertiseAddr = d.IP()
	cfg.AdvertisePort = GossipPort
	cfg.LogOutput = io.Discard
	cfg.ProbeInterval = 1 * time.Second
	cfg.ProbeTimeout = 500 * time.Millisecond
	cfg.SuspicionMult = 3
	cfg.GossipInterval = 200 * time.Millisecond
	cfg.PushPullInterval = 15 * time.Second
	cfg.DeadNodeReclaimTime = 30 * time.Second
	ml, err := memberlist.Create(cfg)
	if err != nil {
		t.Shutdown()
		return nil, err
	}
	return &Gossip{ml: ml, trans: t}, nil
}

// Join contacts peers (mesh ip:port). Unreachable peers are not an error.
func (g *Gossip) Join(addrs []string) int {
	if len(addrs) == 0 {
		return 0
	}
	n, _ := g.ml.Join(addrs)
	return n
}

// MemberState is a gossip observation.
type MemberState struct {
	Name  string
	Addr  string
	State string // alive | suspect | dead | left
}

// Members returns the current membership view.
func (g *Gossip) Members() []MemberState {
	var out []MemberState
	for _, m := range g.ml.Members() {
		st := "alive"
		switch m.State {
		case memberlist.StateSuspect:
			st = "suspect"
		case memberlist.StateDead:
			st = "dead"
		case memberlist.StateLeft:
			st = "left"
		}
		out = append(out, MemberState{Name: m.Name, Addr: m.Addr.String(), State: st})
	}
	return out
}

// NumMembers returns the alive member count.
func (g *Gossip) NumMembers() int { return g.ml.NumMembers() }

// Close leaves the cluster.
func (g *Gossip) Close() error {
	if g == nil {
		return errors.New("nil gossip")
	}
	_ = g.ml.Leave(500 * time.Millisecond)
	return g.ml.Shutdown()
}
