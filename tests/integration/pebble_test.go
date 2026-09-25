package integration

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"decentralized.host/pkg/pki"
)

// pebbleEnv is a running Let's Encrypt Pebble ACME test CA plus its
// challenge test server (DNS for HTTP-01 resolution and DNS-01 records).
type pebbleEnv struct {
	Directory string
	TrustFile string // CA that signed Pebble's own TLS certificate
	Mgmt      string
	DNSMgmt   string
}

func toolsBin(name string) string {
	root, _ := filepath.Abs("../..")
	return filepath.Join(root, "tools", "bin", name)
}

func freePort(addr string) bool {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	l.Close()
	return true
}

func startPebble(t *testing.T) *pebbleEnv {
	t.Helper()
	for _, b := range []string{"pebble", "pebble-challtestsrv"} {
		if _, err := os.Stat(toolsBin(b)); err != nil {
			t.Skipf("%s not installed (run `make tools`); ACME issuance cannot be exercised without an ACME server", b)
		}
	}
	for _, a := range []string{"127.0.0.1:14000", "127.0.0.1:15000", "127.0.0.1:5002", "127.0.0.1:8055"} {
		if !freePort(a) {
			t.Skipf("%s is in use; Pebble needs it", a)
		}
	}
	dir := t.TempDir()
	caCert, caKey, err := pki.NewLocalCA("pebble-tls")
	if err != nil {
		t.Fatal(err)
	}
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	leaf, err := pki.SignLocal(caCert, caKey, &key.PublicKey, []string{"localhost", "127.0.0.1"}, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	kder, _ := x509.MarshalECPrivateKey(key)
	os.WriteFile(filepath.Join(dir, "cert.pem"), leaf, 0o600)
	os.WriteFile(filepath.Join(dir, "key.pem"), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kder}), 0o600)
	os.WriteFile(filepath.Join(dir, "ca.pem"), caCert, 0o600)
	cfg := fmt.Sprintf(`{"pebble":{"listenAddress":"127.0.0.1:14000","managementListenAddress":"127.0.0.1:15000","certificate":%q,"privateKey":%q,
"httpPort":5002,"tlsPort":5001,"ocspResponderURL":"","externalAccountBindingRequired":false,"domainBlocklist":[],
"retryAfter":{"authz":1,"order":1},"keyAlgorithm":"ecdsa","profiles":{"default":{"description":"test","validityPeriod":7776000}}}}`,
		filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"))
	os.WriteFile(filepath.Join(dir, "pebble.json"), []byte(cfg), 0o600)

	start := func(name string, args ...string) {
		logf, _ := os.Create(filepath.Join(dir, name+".log"))
		cmd := exec.Command(toolsBin(name), args...)
		cmd.Stdout, cmd.Stderr = logf, logf
		cmd.Env = append(os.Environ(), "PEBBLE_VA_NOSLEEP=1", "PEBBLE_WFE_NONCEREJECT=0", "PEBBLE_AUTHZREUSE=0")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			if t.Failed() {
				b, _ := os.ReadFile(filepath.Join(dir, name+".log"))
				t.Logf("---- %s ----\n%s", name, b)
			}
		})
	}
	start("pebble-challtestsrv", "-defaultIPv4", "127.0.0.1", "-defaultIPv6", "", "-dnsserver", "127.0.0.1:8053",
		"-http01", "", "-https01", "", "-tlsalpn01", "", "-doh", "", "-management", "127.0.0.1:8055")
	start("pebble", "-config", filepath.Join(dir, "pebble.json"), "-dnsserver", "127.0.0.1:8053", "-strict=false")
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)
	cl := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}}
	deadline := time.Now().Add(15 * time.Second)
	for {
		resp, err := cl.Get("https://127.0.0.1:14000/dir")
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("pebble did not start: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	return &pebbleEnv{Directory: "https://127.0.0.1:14000/dir", TrustFile: filepath.Join(dir, "ca.pem"), Mgmt: "https://127.0.0.1:15000", DNSMgmt: "http://127.0.0.1:8055"}
}

// Roots returns the pool of Pebble's issuing roots (for verifying issued certificates).
func (p *pebbleEnv) Roots(t *testing.T) *x509.CertPool {
	b, _ := os.ReadFile(p.TrustFile)
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(b)
	cl := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}}
	resp, err := cl.Get(p.Mgmt + "/roots/0")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	root, _ := io.ReadAll(resp.Body)
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(root) {
		t.Fatalf("pebble root: %s", root)
	}
	return roots
}
