package mesh

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// orderedKeys returns two keys with lo.PublicString() < hi.PublicString():
// lo is the side that starts the first handshake of the pair.
func orderedKeys(t *testing.T) (lo, hi Key) {
	t.Helper()
	a, _ := LoadOrCreateKey(filepath.Join(t.TempDir(), "a.key"))
	b, _ := LoadOrCreateKey(filepath.Join(t.TempDir(), "b.key"))
	if a.PublicString() > b.PublicString() {
		a, b = b, a
	}
	return a, b
}

func peerTx(d *Device) int64 {
	var n int64
	for _, s := range d.Stats() {
		n += s.TxBytes
	}
	return n
}

// Regression: crossed first handshakes. The lower-key side of a pair sends a
// handshake initiation as soon as the peer is configured (startup
// keepalive). If the higher-key side sends data at the same moment, its
// data-triggered initiation crosses that one; wireguard-go processes the two
// messages on different goroutines, and one direction of the tunnel can then
// drop every packet until the 15 s rekey timer (REKEY_TIMEOUT +
// KEEPALIVE_TIMEOUT). Gossip joins gave up at memberlist's 10 s TCP timeout.
//
// The higher-key side must therefore not initiate the first handshake while
// the lower side's initiation may be in flight: it sends nothing until the
// lower side's initiation arrives, and its data flows once that handshake
// completes.
func TestFirstHandshakeIsStartedByTheLowerKeyOnly(t *testing.T) {
	loKey, hiKey := orderedKeys(t)
	pl, ph := freeUDP(t), freeUDP(t)
	lo, err := Start(loKey, "10.77.0.1", pl)
	if err != nil {
		t.Fatal(err)
	}
	hi, err := Start(hiKey, "10.77.0.2", ph)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lo.Close(); hi.Close() })
	ln, err := lo.ListenTCP(7800)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	// The higher-key side is configured first and immediately has data.
	if err := hi.SetPeers([]PeerConfig{{Node: "lo", WGPub: loKey.PublicString(), Endpoint: fmt.Sprintf("127.0.0.1:%d", pl), MeshIP: "10.77.0.1"}}); err != nil {
		t.Fatal(err)
	}
	dialed := make(chan time.Time, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if c, err := hi.DialContext(ctx, "tcp", "10.77.0.1:7800"); err == nil {
			c.Close()
			dialed <- time.Now()
		}
	}()
	time.Sleep(300 * time.Millisecond)
	if tx := peerTx(hi); tx != 0 {
		t.Fatalf("higher-key side sent %d bytes (a handshake initiation) before the lower side's initiation could arrive", tx)
	}

	// The lower-key side comes up; its startup initiation completes the
	// handshake and releases the staged connection attempt.
	start := time.Now()
	if err := lo.SetPeers([]PeerConfig{{Node: "hi", WGPub: hiKey.PublicString(), Endpoint: fmt.Sprintf("127.0.0.1:%d", ph), MeshIP: "10.77.0.2"}}); err != nil {
		t.Fatal(err)
	}
	select {
	case at := <-dialed:
		// TCP retransmits the SYN at 1 s, 3 s, …; the connection must be up
		// long before the grace period would have let hi initiate itself.
		if d := at.Sub(start); d >= firstContactGrace {
			t.Fatalf("connection took %s: it waited for the fallback, not the lower side's handshake", d)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no connection through the tunnel")
	}
}

// If the lower-key side cannot reach the higher one (for example the higher
// side is behind NAT and only it can open the path), the higher side starts
// the handshake itself once the grace period has passed.
func TestHigherKeyInitiatesAfterGraceWhenLowerCannotReachIt(t *testing.T) {
	loKey, hiKey := orderedKeys(t)
	pl, ph := freeUDP(t), freeUDP(t)
	lo, err := Start(loKey, "10.77.0.1", pl)
	if err != nil {
		t.Fatal(err)
	}
	hi, err := Start(hiKey, "10.77.0.2", ph)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lo.Close(); hi.Close() })
	ln, err := lo.ListenTCP(7801)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	// lo knows hi's key but not its endpoint: it can answer, never initiate.
	if err := lo.SetPeers([]PeerConfig{{Node: "hi", WGPub: hiKey.PublicString(), MeshIP: "10.77.0.2"}}); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if err := hi.SetPeers([]PeerConfig{{Node: "lo", WGPub: loKey.PublicString(), Endpoint: fmt.Sprintf("127.0.0.1:%d", pl), MeshIP: "10.77.0.1"}}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), firstContactGrace+8*time.Second)
	defer cancel()
	c, err := hi.DialContext(ctx, "tcp", "10.77.0.1:7801")
	if err != nil {
		t.Fatalf("no connection after the grace period: %v", err)
	}
	c.Close()
	if d := time.Since(start); d < firstContactGrace {
		t.Fatalf("connected after %s, before the grace period; the lower side cannot have initiated", d)
	}
}
