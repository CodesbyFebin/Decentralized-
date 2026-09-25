package edge

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme"

	"decentralized.host/pkg/api"
)

// Certificate states (blueprint §10).
const (
	StateRequested = "REQUESTED"
	StatePending   = "PENDING"
	StateIssued    = "ISSUED"
	StateRenewing  = "RENEWING"
	StateExpired   = "EXPIRED"
	StateRevoked   = "REVOKED"
	StateFailed    = "FAILED"
	StateUnknown   = "UNKNOWN"
)

// LocalSigner asks the control plane's local CA to sign a public key.
type LocalSigner func(names []string, pubPEM string) (certPEM string, err error)

// ACMEConfig configures the ACME issuer.
type ACMEConfig struct {
	Directory string
	Email     string
	CACert    string // PEM file trusted for the directory TLS (Pebble)
	DNS       string // DNS-01 provider "challtestsrv=http://host:port"
}

type certState struct {
	host     string
	names    []string
	issuer   string
	state    string
	detail   string
	since    int64
	cert     *tls.Certificate
	leaf     *x509.Certificate
	nextTry  time.Time
	attempts int
	busy     bool
}

// CertManager keeps private keys on the edge host and drives issuance.
type CertManager struct {
	dir     string
	local   LocalSigner
	acmeCfg ACMEConfig
	log     *log.Logger

	mu     sync.Mutex
	acmeMu sync.Mutex
	certs  map[string]*certState // by ingress host
	tokens map[string]string     // HTTP-01 token -> key authorization
	client *acme.Client
	wake   chan struct{}
}

// NewCertManager creates the manager; keys and certificates live in dir.
func NewCertManager(dir string, local LocalSigner, ac ACMEConfig, l *log.Logger) *CertManager {
	_ = os.MkdirAll(dir, 0o700)
	m := &CertManager{dir: dir, local: local, acmeCfg: ac, log: l, certs: map[string]*certState{}, tokens: map[string]string{}, wake: make(chan struct{}, 1)}
	return m
}

// Run drives issuance and renewal until ctx ends.
func (m *CertManager) Run(ctx context.Context) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case <-m.wake:
		}
		m.step(ctx)
	}
}

// Want registers the ingress hosts that need certificates.
func (m *CertManager) Want(services []api.Service) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UnixMilli()
	for _, s := range services {
		if s.TLS == "" || s.TLS == "none" {
			continue
		}
		cs := m.certs[s.Host]
		if cs == nil {
			cs = &certState{host: s.Host, names: []string{s.Host}, issuer: s.TLS, state: StateRequested, detail: "ingress declares tls: " + s.TLS, since: now}
			m.certs[s.Host] = cs
			m.loadFromDisk(cs)
			select {
			case m.wake <- struct{}{}:
			default:
			}
		}
	}
}

func (m *CertManager) paths(host string) (string, string) {
	safe := strings.ReplaceAll(host, "*", "_wildcard_")
	return filepath.Join(m.dir, safe+".crt"), filepath.Join(m.dir, safe+".key")
}

func (m *CertManager) loadFromDisk(cs *certState) {
	cp, kp := m.paths(cs.host)
	pair, err := tls.LoadX509KeyPair(cp, kp)
	if err != nil {
		return
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return
	}
	pair.Leaf = leaf
	cs.cert, cs.leaf = &pair, leaf
	cs.state, cs.detail = StateIssued, "loaded from the edge host's certificate store"
	if time.Now().After(leaf.NotAfter) {
		cs.state, cs.detail = StateExpired, "stored certificate has expired"
	}
}

func (m *CertManager) step(ctx context.Context) {
	m.mu.Lock()
	var todo []*certState
	for _, cs := range m.certs {
		due := time.Now().After(cs.nextTry)
		switch cs.state {
		case StateRequested, StateFailed, StateExpired:
			if due {
				todo = append(todo, cs)
			}
		case StateIssued:
			if cs.leaf != nil {
				life := cs.leaf.NotAfter.Sub(cs.leaf.NotBefore)
				if time.Until(cs.leaf.NotAfter) < life/3 {
					cs.state, cs.detail, cs.since = StateRenewing, "less than a third of the lifetime remains", time.Now().UnixMilli()
					todo = append(todo, cs)
				}
				if time.Now().After(cs.leaf.NotAfter) {
					cs.state, cs.detail, cs.since = StateExpired, "certificate expired", time.Now().UnixMilli()
				}
			}
		case StateRenewing:
			if due {
				todo = append(todo, cs)
			}
		}
	}
	m.mu.Unlock()
	// Each certificate is issued independently: a slow ACME order must not
	// hold up a local-CA certificate or another host's order.
	for _, cs := range todo {
		m.mu.Lock()
		busy := cs.busy
		cs.busy = true
		m.mu.Unlock()
		if busy {
			continue
		}
		go func(cs *certState) {
			m.issue(ctx, cs)
			m.mu.Lock()
			cs.busy = false
			m.mu.Unlock()
		}(cs)
	}
}

func (m *CertManager) set(cs *certState, state, detail string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cs.state != state || cs.detail != detail {
		cs.state, cs.detail, cs.since = state, detail, time.Now().UnixMilli()
		m.logf("certificate %s: %s (%s)", cs.host, state, detail)
	}
}

func (m *CertManager) logf(format string, args ...any) {
	if m.log != nil {
		m.log.Printf(format, args...)
	}
}

func (m *CertManager) issue(ctx context.Context, cs *certState) {
	m.mu.Lock()
	// A retry after a failure stays FAILED (with its reason) until it
	// succeeds; only a first attempt is shown as PENDING.
	renewing := cs.state == StateRenewing || cs.state == StateFailed
	m.mu.Unlock()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		m.set(cs, StateFailed, err.Error())
		return
	}
	var chain [][]byte
	switch cs.issuer {
	case "local":
		if m.local == nil {
			m.set(cs, StateFailed, "no local CA signer")
			return
		}
		if !renewing {
			m.set(cs, StatePending, "requesting a certificate from the cluster-local CA")
		}
		pubDER, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
		certPEM, err := m.local(cs.names, string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})))
		if err != nil {
			m.fail(cs, err)
			return
		}
		for rest := []byte(certPEM); ; {
			var b *pem.Block
			b, rest = pem.Decode(rest)
			if b == nil {
				break
			}
			chain = append(chain, b.Bytes)
		}
	case "acme":
		if m.acmeCfg.Directory == "" {
			m.fail(cs, errors.New("no ACME directory configured on this edge (--acme-directory)"))
			return
		}
		if !renewing {
			m.set(cs, StatePending, "ACME order in progress at "+m.acmeCfg.Directory)
		}
		chain, err = m.acmeIssue(ctx, cs, key)
		if err != nil {
			m.fail(cs, err)
			return
		}
	default:
		m.fail(cs, fmt.Errorf("unknown issuer %q", cs.issuer))
		return
	}
	if len(chain) == 0 {
		m.fail(cs, errors.New("issuer returned no certificate"))
		return
	}
	leaf, err := x509.ParseCertificate(chain[0])
	if err != nil {
		m.fail(cs, err)
		return
	}
	if err := leaf.VerifyHostname(strings.TrimPrefix(cs.host, "*.")); err != nil && !strings.HasPrefix(cs.host, "*.") {
		m.fail(cs, err)
		return
	}
	var certBuf bytes.Buffer
	for _, c := range chain {
		_ = pem.Encode(&certBuf, &pem.Block{Type: "CERTIFICATE", Bytes: c})
	}
	kder, _ := x509.MarshalECPrivateKey(key)
	cp, kp := m.paths(cs.host)
	_ = os.WriteFile(kp, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kder}), 0o600)
	_ = os.WriteFile(cp, certBuf.Bytes(), 0o644)
	pair := &tls.Certificate{Certificate: chain, PrivateKey: key, Leaf: leaf}
	m.mu.Lock()
	cs.cert, cs.leaf, cs.attempts = pair, leaf, 0
	cs.state, cs.detail, cs.since = StateIssued, fmt.Sprintf("issued by %q, valid until %s", leaf.Issuer.CommonName, leaf.NotAfter.UTC().Format(time.RFC3339)), time.Now().UnixMilli()
	m.logf("certificate %s: ISSUED (%s)", cs.host, cs.detail)
	m.mu.Unlock()
}

func (m *CertManager) fail(cs *certState, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cs.attempts++
	backoff := time.Duration(1<<uint(minInt(cs.attempts, 6))) * 5 * time.Second
	cs.nextTry = time.Now().Add(backoff)
	m.logf("certificate %s: attempt %d failed: %v (retry in %s)", cs.host, cs.attempts, err, backoff)
	if cs.state == StateRenewing && cs.leaf != nil && time.Now().Before(cs.leaf.NotAfter) {
		cs.detail = "renewal failed (" + err.Error() + "); current certificate still valid"
		return
	}
	cs.state, cs.detail, cs.since = StateFailed, err.Error(), time.Now().UnixMilli()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m *CertManager) acmeClient(ctx context.Context) (*acme.Client, error) {
	// One account per edge: concurrent issuances must not each register.
	m.acmeMu.Lock()
	defer m.acmeMu.Unlock()
	m.mu.Lock()
	c := m.client
	m.mu.Unlock()
	if c != nil {
		return c, nil
	}
	keyPath := filepath.Join(m.dir, "acme-account.key")
	var key *ecdsa.PrivateKey
	if b, err := os.ReadFile(keyPath); err == nil {
		blk, _ := pem.Decode(b)
		if blk != nil {
			key, _ = x509.ParseECPrivateKey(blk.Bytes)
		}
	}
	if key == nil {
		var err error
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		der, _ := x509.MarshalECPrivateKey(key)
		_ = os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}), 0o600)
	}
	hc := &http.Client{Timeout: 30 * time.Second}
	if m.acmeCfg.CACert != "" {
		pemBytes, err := os.ReadFile(m.acmeCfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("acme CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(pemBytes)
		hc.Transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}}
	}
	c = &acme.Client{Key: key, DirectoryURL: m.acmeCfg.Directory, HTTPClient: hc, UserAgent: "decentralized.host-edge"}
	acct := &acme.Account{}
	if m.acmeCfg.Email != "" {
		acct.Contact = []string{"mailto:" + m.acmeCfg.Email}
	}
	if _, err := c.Register(ctx, acct, acme.AcceptTOS); err != nil && !errors.Is(err, acme.ErrAccountAlreadyExists) {
		return nil, fmt.Errorf("acme register: %w", err)
	}
	m.mu.Lock()
	m.client = c
	m.mu.Unlock()
	return c, nil
}

func (m *CertManager) acmeIssue(ctx context.Context, cs *certState, key crypto.Signer) ([][]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	c, err := m.acmeClient(ctx)
	if err != nil {
		return nil, err
	}
	order, err := c.AuthorizeOrder(ctx, acme.DomainIDs(cs.names...))
	if err != nil {
		return nil, fmt.Errorf("acme order: %w", err)
	}
	for _, u := range order.AuthzURLs {
		z, err := c.GetAuthorization(ctx, u)
		if err != nil {
			return nil, err
		}
		if z.Status == acme.StatusValid {
			continue
		}
		var chal *acme.Challenge
		// HTTP-01 reaches only the edge the CA happens to resolve to; with a
		// DNS provider every edge can validate independently, so prefer it.
		wantType := "http-01"
		if z.Wildcard || m.acmeCfg.DNS != "" {
			wantType = "dns-01"
		}
		for _, ch := range z.Challenges {
			if ch.Type == wantType {
				chal = ch
			}
		}
		if chal == nil {
			return nil, fmt.Errorf("acme: no %s challenge offered for %s", wantType, z.Identifier.Value)
		}
		switch wantType {
		case "http-01":
			resp, err := c.HTTP01ChallengeResponse(chal.Token)
			if err != nil {
				return nil, err
			}
			m.mu.Lock()
			m.tokens[chal.Token] = resp
			m.mu.Unlock()
		case "dns-01":
			rec, err := c.DNS01ChallengeRecord(chal.Token)
			if err != nil {
				return nil, err
			}
			if err := m.setTXT("_acme-challenge."+z.Identifier.Value+".", rec); err != nil {
				return nil, fmt.Errorf("dns-01: %w", err)
			}
		}
		if _, err := c.Accept(ctx, chal); err != nil {
			return nil, fmt.Errorf("acme accept: %w", err)
		}
		if _, err := c.WaitAuthorization(ctx, z.URI); err != nil {
			return nil, fmt.Errorf("acme %s validation for %s: %w", wantType, z.Identifier.Value, err)
		}
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{Subject: pkix.Name{CommonName: cs.names[0]}, DNSNames: cs.names}, key)
	if err != nil {
		return nil, err
	}
	ready, err := c.WaitOrder(ctx, order.URI)
	if err != nil {
		return nil, fmt.Errorf("acme order wait: %w", err)
	}
	finalize := ready.FinalizeURL
	if finalize == "" {
		finalize = order.FinalizeURL
	}
	der, _, err := c.CreateOrderCert(ctx, finalize, csr, true)
	if err == nil {
		return der, nil
	}
	// Some servers (Pebble) answer finalize with a "processing" order and no
	// Location header; x/crypto then polls an empty URL. Poll the order URL
	// we already know and fetch the certificate ourselves.
	done, werr := c.WaitOrder(ctx, order.URI)
	if werr != nil || done.CertURL == "" {
		return nil, fmt.Errorf("acme finalize: %v (order wait: %v)", err, werr)
	}
	der, err = c.FetchCert(ctx, done.CertURL, true)
	if err != nil {
		return nil, fmt.Errorf("acme fetch certificate: %w", err)
	}
	return der, nil
}

// setTXT publishes a DNS-01 record through the configured provider.
func (m *CertManager) setTXT(fqdn, value string) error {
	name, addr, ok := strings.Cut(m.acmeCfg.DNS, "=")
	if !ok || name != "challtestsrv" {
		return fmt.Errorf("no DNS-01 provider configured (supported: challtestsrv=<url>); wildcard certificates need one")
	}
	body, _ := json.Marshal(map[string]string{"host": fqdn, "value": value})
	resp, err := http.Post(strings.TrimRight(addr, "/")+"/set-txt", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("challtestsrv set-txt: %s", resp.Status)
	}
	return nil
}

// ChallengeResponse serves an HTTP-01 key authorization.
func (m *CertManager) ChallengeResponse(token string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.tokens[token]
	return v, ok
}

// Issued reports whether host has an ISSUED certificate.
func (m *CertManager) Issued(host string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cs := m.certs[host]; cs != nil {
		return cs.state == StateIssued || cs.state == StateRenewing
	}
	return false
}

// GetCertificate selects by SNI (exact, then wildcard).
func (m *CertManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	name := strings.ToLower(hello.ServerName)
	if cs := m.certs[name]; cs != nil && cs.cert != nil {
		return cs.cert, nil
	}
	if i := strings.IndexByte(name, '.'); i > 0 {
		if cs := m.certs["*"+name[i:]]; cs != nil && cs.cert != nil {
			return cs.cert, nil
		}
	}
	return nil, fmt.Errorf("no issued certificate for %q", name)
}

// Observe returns certificate states as the edge measures them.
func (m *CertManager) Observe() []api.CertObs {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []api.CertObs
	for _, cs := range m.certs {
		o := api.CertObs{Host: cs.host, Names: cs.names, State: cs.state, Issuer: cs.issuer, Detail: cs.detail, Since: cs.since}
		if cs.leaf != nil {
			o.Serial = cs.leaf.SerialNumber.Text(16)
			o.NotBefore, o.NotAfter = cs.leaf.NotBefore.UnixMilli(), cs.leaf.NotAfter.UnixMilli()
			sum := sha256.Sum256(cs.leaf.Raw)
			o.Fingerprint = "sha256:" + hex.EncodeToString(sum[:])
			if cs.leaf.Issuer.CommonName != "" {
				o.Issuer = cs.issuer + " (" + cs.leaf.Issuer.CommonName + ")"
			}
		}
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Host < out[j].Host })
	return out
}
