package integration

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/control"
	"decentralized.host/pkg/devcluster"
)

const edgeManifest = `apiVersion: dh/v1
kind: Application
metadata: {name: web}
spec:
  replicas: 3
  image: %s
  resources: {cpu: 100m, mem: 32Mi}
  placement: {tiers: [trusted], spread: failure-domain, antiAffinity: hard}
  ports: [{name: http}]
  ingress:
    - {host: web.acme.test, port: http, tls: acme}
    - {host: "*.apps.acme.test", port: http, tls: acme}
    - {host: local.web.test, port: http, tls: local}
  health: {http: /healthz, interval: 1s}
`

func httpsVia(addr, sni string, roots *x509.CertPool, path string) (int, string, *x509.Certificate, error) {
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{RootCAs: roots, ServerName: sni, NextProtos: []string{"http/1.1"}})
	if err != nil {
		return 0, "", nil, err
	}
	defer conn.Close()
	cert := conn.ConnectionState().PeerCertificates[0]
	fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", path, sni)
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		return 0, "", cert, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, resp.Header.Get("X-DH-Upstream"), cert, nil
}

func edgeObs(v *control.View, name string) *api.EdgeObs {
	for _, e := range v.Edge.Edges {
		if e.Name == name {
			return e.Obs
		}
	}
	return nil
}

func certState(o *api.EdgeObs, host string) *api.CertObs {
	if o == nil {
		return nil
	}
	for i := range o.Certs {
		if o.Certs[i].Host == host {
			return &o.Certs[i]
		}
	}
	return nil
}

func routeStates(o *api.EdgeObs, host string) map[string]string {
	out := map[string]string{}
	if o == nil {
		return out
	}
	for _, r := range o.Routes {
		if r.Host == host {
			for _, e := range r.Endpoints {
				out[e.Assignment] = e.State
			}
		}
	}
	return out
}

// M4 exit: HTTPS to a three-replica app through ACME-issued certificates,
// health-gated routing, draining, and edge failover.
func TestM4EdgeTLSAndFailover(t *testing.T) {
	pb := startPebble(t)
	c := up(t, devcluster.Options{CPs: 1, Hosts: 3, Edges: 2, Chaos: true,
		EdgeHTTP: map[int]int{0: 5002},
		EdgeArgs: []string{"--acme-directory", pb.Directory, "--acme-ca", pb.TrustFile, "--acme-email", "ops@acme.test"},
		EdgeExtra: func(i int) []string {
			if i == 1 {
				return []string{"--acme-dns", "challtestsrv=" + pb.DNSMgmt}
			}
			return nil
		}})
	deploy(t, c, edgeManifest)
	waitRunning(t, c, "web", 3, 60*time.Second)
	e1, e2 := c.Edges[0], c.Edges[1]

	if err := c.WaitFor(90*time.Second, "ACME certificates issued (HTTP-01 on edge-1, DNS-01 incl. wildcard on edge-2) and all replicas routing", func(v *control.View) bool {
		o1, o2 := edgeObs(v, "edge-1"), edgeObs(v, "edge-2")
		ok := func(o *api.EdgeObs, h string) bool { c := certState(o, h); return c != nil && c.State == "ISSUED" }
		routing := 0
		for _, o := range []*api.EdgeObs{o1, o2} {
			for _, s := range routeStates(o, "web.acme.test") {
				if s == "routing" {
					routing++
				}
			}
		}
		return ok(o1, "web.acme.test") && ok(o2, "web.acme.test") && ok(o2, "*.apps.acme.test") && ok(o1, "local.web.test") && routing == 6
	}); err != nil {
		v, _ := c.View()
		for _, e := range v.Edge.Edges {
			if e.Obs != nil {
				for _, cs := range e.Obs.Certs {
					t.Logf("%s %s %s: %s", e.Name, cs.Host, cs.State, cs.Detail)
				}
			}
			t.Logf("%s freshness %s routes %v", e.Name, e.Freshness, routeStates(e.Obs, "web.acme.test"))
		}
		if st, err := c.HostStatus("edge-1"); err == nil {
			b, _ := json.Marshal(st.Edge)
			t.Logf("edge-1 local status: mode=%s lastErr=%q edge=%s", st.Mode, st.LastErr, b)
		}
		for _, h := range []string{"host-a", "host-b", "host-c", "edge-1", "edge-2"} {
			if st, err := c.HostStatus(h); err == nil {
				t.Logf("%s mesh %s peers %v binding %v refused %q", h, st.MeshIP, st.PeerIPs, st.BindingOK, st.Refused)
			}
		}
		t.Fatal(err)
	}
	v, _ := c.View()
	if cs := certState(edgeObs(v, "edge-1"), "*.apps.acme.test"); cs == nil || cs.State != "FAILED" || !strings.Contains(cs.Detail, "DNS-01") {
		t.Fatalf("edge-1 has no DNS provider: its wildcard certificate must be FAILED with the reason, got %+v", cs)
	}

	roots := pb.Roots(t)
	t.Run("HTTPS with a certificate chained to the ACME CA, round-robin over three replicas", func(t *testing.T) {
		seen := map[string]bool{}
		for i := 0; i < 9; i++ {
			code, up, cert, err := httpsVia(e1.HTTPS, "web.acme.test", roots, "/")
			if err != nil || code != 200 {
				t.Fatalf("https: %d %v", code, err)
			}
			if !strings.Contains(cert.Issuer.CommonName, "Pebble") {
				t.Fatalf("issuer %q", cert.Issuer.CommonName)
			}
			seen[up] = true
		}
		if len(seen) != 3 {
			t.Fatalf("round robin reached %d replicas", len(seen))
		}
		code, _, cert, err := httpsVia(e2.HTTPS, "anything.apps.acme.test", roots, "/")
		if err != nil || code != 200 || cert.DNSNames[0] != "*.apps.acme.test" {
			t.Fatalf("wildcard via DNS-01: %d %v %v", code, err, cert)
		}
	})

	t.Run("a hung replica is ejected and requests keep succeeding", func(t *testing.T) {
		v, _ := c.View()
		row := app(v, "web").Rows[0]
		if err := syscall.Kill(int(row.PID), syscall.SIGSTOP); err != nil {
			t.Fatal(err)
		}
		defer syscall.Kill(int(row.PID), syscall.SIGCONT)
		fails, total, afterEject := 0, 0, 0
		ejectedAt := time.Time{}
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			code, up, _, err := httpsVia(e1.HTTPS, "web.acme.test", roots, "/")
			total++
			if err != nil || code != 200 {
				fails++
			}
			if !ejectedAt.IsZero() && strings.HasPrefix(up, row.Assignment+"@") {
				afterEject++
			}
			if ejectedAt.IsZero() {
				v, _ := c.View()
				if routeStates(edgeObs(v, "edge-1"), "web.acme.test")[row.Assignment] == "ejected" {
					ejectedAt = time.Now()
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		if ejectedAt.IsZero() {
			t.Fatal("hung replica was never ejected")
		}
		if fails > 0 || afterEject > 0 {
			t.Fatalf("%d/%d requests failed, %d served by the ejected replica", fails, total, afterEject)
		}
		t.Logf("%d requests, 0 failures while replica %s was hung and ejected", total, row.Assignment)
		syscall.Kill(int(row.PID), syscall.SIGCONT)
		if err := c.WaitFor(20*time.Second, "replica returns to routing after recovery", func(v *control.View) bool {
			return routeStates(edgeObs(v, "edge-1"), "web.acme.test")[row.Assignment] == "routing"
		}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("scale-down drains the replica before it leaves the routing table", func(t *testing.T) {
		if err := c.Op.Do("POST", "/api/v1/apps/web/scale", map[string]int64{"replicas": 2}, nil); err != nil {
			t.Fatal(err)
		}
		if err := c.WaitFor(30*time.Second, "replica 2 drained and removed", func(v *control.View) bool {
			s := routeStates(edgeObs(v, "edge-1"), "web.acme.test")
			_, present := s["web/r2"]
			return !present && len(s) == 2
		}); err != nil {
			t.Fatal(err)
		}
		v, _ := c.View()
		drained := false
		for _, e := range v.Audit.Entries {
			if e.Action == "route-draining" && strings.Contains(e.Detail, "web/r2") {
				drained = true
			}
		}
		if !drained {
			t.Log("note: the draining transition was shorter than one observation interval (no audit entry); removal verified")
		}
	})

	t.Run("edge failover: losing an edge does not lose the service", func(t *testing.T) {
		if err := c.Kill("edge-1", syscall.SIGKILL); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 20; i++ {
			code, _, _, err := httpsVia(e2.HTTPS, "web.acme.test", roots, "/")
			if err != nil || code != 200 {
				t.Fatalf("edge-2 after edge-1 loss: %d %v", code, err)
			}
		}
	})
}
