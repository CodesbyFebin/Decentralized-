package mesh

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func freeUDP(t *testing.T) int {
	c, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).Port
}

func pair(t *testing.T) (*Device, *Device) {
	ka, _ := LoadOrCreateKey(filepath.Join(t.TempDir(), "a.key"))
	kb, _ := LoadOrCreateKey(filepath.Join(t.TempDir(), "b.key"))
	pa, pb := freeUDP(t), freeUDP(t)
	a, err := Start(ka, "10.77.0.1", pa)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Start(kb, "10.77.0.2", pb)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close(); b.Close() })
	if err := a.SetPeers([]PeerConfig{{Node: "b", WGPub: kb.PublicString(), Endpoint: fmt.Sprintf("127.0.0.1:%d", pb), MeshIP: "10.77.0.2"}}); err != nil {
		t.Fatal(err)
	}
	if err := b.SetPeers([]PeerConfig{{Node: "a", WGPub: ka.PublicString(), Endpoint: fmt.Sprintf("127.0.0.1:%d", pa), MeshIP: "10.77.0.1"}}); err != nil {
		t.Fatal(err)
	}
	return a, b
}

// Real WireGuard: an HTTP request travels through two userspace devices
// over loopback UDP, and the device reports a completed handshake.
func TestHTTPThroughWireGuard(t *testing.T) {
	a, b := pair(t)
	ln, err := b.ListenTCP(7800)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello from b; you are %s", r.RemoteAddr)
	})}
	go srv.Serve(ln)
	defer srv.Close()
	resp, err := a.HTTPClient(15 * time.Second).Get("http://10.77.0.2:7800/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "you are 10.77.0.1:") {
		t.Fatalf("unexpected body %q", body)
	}
	var hs time.Time
	for _, s := range a.Stats() {
		if s.Node == "b" {
			hs = s.LastHandshake
			if s.TxBytes == 0 || s.RxBytes == 0 {
				t.Fatalf("no traffic counted: %+v", s)
			}
		}
	}
	if hs.IsZero() {
		t.Fatal("no handshake recorded")
	}
}

// A device without the peer configured cannot be reached: no key, no route.
func TestUnknownPeerIsUnreachable(t *testing.T) {
	a, b := pair(t)
	_ = b
	if err := a.SetPeers(nil); err != nil {
		t.Fatal(err)
	}
	c := a.HTTPClient(1500 * time.Millisecond)
	if _, err := c.Get("http://10.77.0.2:7800/"); err == nil {
		t.Fatal("reached a peer that was removed")
	}
}

func TestGossipMembership(t *testing.T) {
	a, b := pair(t)
	ga, err := StartGossip(a, "node-a")
	if err != nil {
		t.Fatal(err)
	}
	defer ga.Close()
	gb, err := StartGossip(b, "node-b")
	if err != nil {
		t.Fatal(err)
	}
	defer gb.Close()
	if n := gb.Join([]string{"10.77.0.1:7946"}); n != 1 {
		t.Fatalf("join contacted %d", n)
	}
	deadline := time.Now().Add(5 * time.Second)
	for ga.NumMembers() < 2 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if ga.NumMembers() != 2 {
		t.Fatal(ga.Members())
	}
}
