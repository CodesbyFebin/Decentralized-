package integration

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"decentralized.host/pkg/devcluster"
)

// A production-shaped cluster: every member API speaks TLS from its first
// start. Bootstrap pins each member's throwaway certificate by fingerprint;
// afterwards members serve root-issued certificates, hosts join over HTTPS
// with the root CA from the join token, and plain HTTP is refused.
func TestTLSEndToEnd(t *testing.T) {
	c := up(t, devcluster.Options{CPs: 3, Hosts: 2, Edges: 0, TLS: true})
	deploy(t, c, beacon(2))
	waitRunning(t, c, "beacon", 2, 60*time.Second)

	caPEM, err := os.ReadFile(c.Op.RootCAPath())
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)
	verified := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}}

	for _, p := range c.Procs {
		if p.Kind != "control" {
			continue
		}
		// Root-issued certificate, verified against the cluster root CA.
		resp, err := verified.Get("https://" + p.API + "/api/v1/health")
		if err != nil {
			t.Fatalf("%s: TLS with root CA verification: %v", p.Name, err)
		}
		var h map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&h)
		resp.Body.Close()
		if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 || resp.TLS.PeerCertificates[0].Subject.CommonName == "dh member awaiting bootstrap" {
			t.Fatalf("%s still serves its bootstrap certificate", p.Name)
		}
		// Plain HTTP must not reach the API.
		if resp, err := (&http.Client{Timeout: 3 * time.Second}).Get("http://" + p.API + "/api/v1/health"); err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				t.Fatalf("%s answered plain HTTP", p.Name)
			}
		}
		// The bootstrap pin is gone once credentials exist.
		if _, err := os.Stat(filepath.Join(p.Data, "bootstrap.fingerprint")); !os.IsNotExist(err) {
			t.Fatalf("%s: bootstrap.fingerprint left behind (%v)", p.Name, err)
		}
		// The console is served over the same verified TLS.
		resp, err = verified.Get("https://" + p.API + "/")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 || resp.Header.Get("Content-Security-Policy") == "" {
			t.Fatalf("%s console over TLS: HTTP %d", p.Name, resp.StatusCode)
		}
	}

	// Hosts reach the control plane over TLS: their evidence is fresh.
	v, err := c.View()
	if err != nil {
		t.Fatal(err)
	}
	if v.Overview.HostsFresh != 2 {
		t.Fatalf("hosts with fresh signed observations over TLS: %d", v.Overview.HostsFresh)
	}

	// Leader loss over TLS: a new leader is elected and a write through a
	// follower is forwarded to it over verified TLS.
	leader, err := c.Leader()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Kill(leader, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		if l, err := c.Leader(); err == nil && l != leader {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no new leader over TLS within 20s")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := c.Op.Do("POST", "/api/v1/apps/beacon/scale", map[string]int64{"replicas": 2}, nil); err != nil {
		t.Fatalf("write after failover over TLS: %v", err)
	}
	waitRunning(t, c, "beacon", 2, 30*time.Second)
}
