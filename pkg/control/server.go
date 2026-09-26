// Package control implements dh-control: the replicated desired-state
// authority. It proposes; hosts decide.
package control

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/raft"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/capability"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
	"decentralized.host/pkg/pki"
	"decentralized.host/pkg/runtime"
	"decentralized.host/pkg/storage"
)

// Config configures one control-plane member.
type Config struct {
	DataDir       string
	APIListen     string // e.g. 127.0.0.1:7700
	APIAdvertise  string // address hosts and peers use; defaults to APIListen
	RaftListen    string
	RaftAdvertise string
	TLS           bool   // serve the API over TLS with the root-issued member certificate
	MeshListen    string // UDP address for this member's WireGuard device ("" = no mesh)
	MeshAdvertise string
	LostAfter     time.Duration
	PostgresURL   string
	Web           fs.FS
	FastRaft      bool
	Logger        *log.Logger
}

// Credentials are written when the member is bootstrapped or joined.
type Credentials struct {
	Cluster    string `json:"cluster"`
	Root       string `json:"root"`
	RootCACert string `json:"rootCaCert"`
	MemberCert string `json:"memberCert"`
	Bootstrap  bool   `json:"bootstrap"`
}

// Server is one dh-control member.
type Server struct {
	cfg Config
	id  *identity.Identity
	fsm *FSM
	log *log.Logger
	cas *storage.CAS

	mu     sync.RWMutex
	bootMu sync.Mutex // serializes bootstrap/join (single-use code)
	rn     *raftNode
	creds  *Credentials
	code   string

	httpSrv *http.Server
	kick    chan struct{}
	stop    chan struct{}
	stopped chan struct{}

	obs       *obsCache
	plans     *planCache
	local     *localRejections
	leaderVer struct {
		sync.Mutex
		at time.Time
	}
	mirror       *mirror
	meshMu       sync.RWMutex
	meshCli      meshClient
	started      time.Time
	lastCP       time.Time             // only touched by the housekeeping goroutine
	materializer *runtime.Materializer // [A05] for ephemeral secret delivery
}

// New creates a member; call Run to serve.
func New(cfg Config) (*Server, error) {
	if cfg.LostAfter == 0 {
		cfg.LostAfter = 30 * time.Second
	}
	if cfg.APIAdvertise == "" {
		cfg.APIAdvertise = cfg.APIListen
	}
	if cfg.RaftAdvertise == "" {
		cfg.RaftAdvertise = cfg.RaftListen
	}
	if cfg.Logger == nil {
		cfg.Logger = log.New(os.Stderr, "dh-control ", log.LstdFlags|log.Lmsgprefix)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, err
	}
	id, err := identity.LoadOrCreate(filepath.Join(cfg.DataDir, "identity"))
	if err != nil {
		return nil, err
	}
	cas, err := storage.OpenCAS(filepath.Join(cfg.DataDir, "cas"), 0)
	if err != nil {
		return nil, err
	}
	// Initialize ephemeral secrets materializer for A05 delivery
	ephemeralDir := filepath.Join(cfg.DataDir, "ephemeral-secrets")
	if err := os.MkdirAll(ephemeralDir, 0o700); err != nil {
		return nil, fmt.Errorf("create ephemeral secrets dir: %w", err)
	}
	s := &Server{cfg: cfg, id: id, fsm: NewFSM(), log: cfg.Logger, cas: cas, kick: make(chan struct{}, 1),
		stop: make(chan struct{}), stopped: make(chan struct{}), obs: newObsCache(), plans: newPlanCache(), local: &localRejections{}, started: time.Now(),
		materializer: runtime.NewMaterializer(ephemeralDir)}
	s.fsm.onApply = func(*Command, *Result) { s.wake() }
	if b, err := os.ReadFile(s.credsPath()); err == nil {
		var c Credentials
		if err := json.Unmarshal(b, &c); err != nil {
			return nil, fmt.Errorf("member credentials: %w", err)
		}
		s.creds = &c
	} else {
		s.code, err = s.bootstrapCode()
		if err != nil {
			return nil, err
		}
	}
	if cfg.PostgresURL != "" {
		s.mirror = newMirror(cfg.PostgresURL, s.log)
	}
	return s, nil
}

// ID returns the member identity.
func (s *Server) ID() *identity.Identity { return s.id }

// BootstrapCode returns the one-time code required to bootstrap or join
// this member ("" once it has credentials).
func (s *Server) BootstrapCode() string { return s.code }

// credentials returns this member's credentials (nil before bootstrap).
func (s *Server) credentials() *Credentials {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.creds
}

func (s *Server) credsPath() string { return filepath.Join(s.cfg.DataDir, "member.json") }

func (s *Server) bootstrapCode() (string, error) {
	p := filepath.Join(s.cfg.DataDir, "bootstrap.code")
	if b, err := os.ReadFile(p); err == nil {
		return strings.TrimSpace(string(b)), nil
	}
	code := randomHex(16)
	return code, os.WriteFile(p, []byte(code+"\n"), 0o600)
}

// Run serves until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	if s.creds != nil {
		if err := s.startRaft(s.creds.Bootstrap); err != nil {
			return err
		}
	}
	ln, err := net.Listen("tcp", s.cfg.APIListen)
	if err != nil {
		return err
	}
	s.httpSrv = &http.Server{Handler: s.routes(), ReadHeaderTimeout: 10 * time.Second}
	if s.cfg.TLS {
		conf, err := s.apiTLS()
		if err != nil {
			ln.Close()
			return err
		}
		ln = tls.NewListener(ln, conf)
	}
	go s.reconcileLoop()
	go s.backgroundLoop()
	if s.mirror != nil {
		go s.mirror.run(s)
	}
	errc := make(chan error, 1)
	go func() { errc <- s.httpSrv.Serve(ln) }()
	s.log.Printf("member %s serving API on %s (raft %s)", s.id.ID, s.cfg.APIListen, s.cfg.RaftListen)
	select {
	case <-ctx.Done():
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	close(s.stop)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.httpSrv.Shutdown(shutdownCtx)
	s.stopMesh()
	s.mu.Lock()
	s.rn.shutdown()
	s.rn = nil
	s.mu.Unlock()
	_ = s.cas.Close()
	return nil
}

// apiTLS serves the root-issued member certificate once this member has
// credentials. Before that it serves a throwaway self-signed certificate
// whose fingerprint is written next to the bootstrap code, so `dh cp
// bootstrap` / `dh cp add-member` can pin it. The switch needs no restart.
func (s *Server) apiTLS() (*tls.Config, error) {
	boot, fp, err := pki.BootstrapCert()
	if err != nil {
		return nil, err
	}
	if s.creds == nil {
		if err := os.WriteFile(filepath.Join(s.cfg.DataDir, "bootstrap.fingerprint"), []byte(fp+"\n"), 0o600); err != nil {
			return nil, err
		}
		s.log.Printf("awaiting bootstrap over TLS; certificate fingerprint %s (also in %s/bootstrap.fingerprint)", fp, s.cfg.DataDir)
	}
	var (
		mu     sync.Mutex
		member *tls.Certificate
		forPEM string
	)
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			s.mu.RLock()
			c := s.creds
			s.mu.RUnlock()
			if c == nil {
				return &boot, nil
			}
			mu.Lock()
			defer mu.Unlock()
			if member == nil || forPEM != c.MemberCert {
				conf, err := pki.MemberTLS(s.id, []byte(c.MemberCert), []byte(c.RootCACert), nil)
				if err != nil {
					return nil, err
				}
				member, forPEM = &conf.Certificates[0], c.MemberCert
			}
			return member, nil
		},
	}, nil
}

func (s *Server) startRaft(bootstrap bool) error {
	c := s.credentials()
	conf, err := pki.MemberTLS(s.id, []byte(c.MemberCert), []byte(c.RootCACert), func(pub string) bool {
		allowed := false
		s.fsm.Read(func(st *State) {
			if len(st.RosterBody.Members) == 0 {
				allowed = true // before the first roster replicates, the root-issued certificate is the credential
				return
			}
			allowed = st.memberForKey(pub) != nil
		})
		return allowed
	})
	if err != nil {
		return err
	}
	logf, _ := os.OpenFile(filepath.Join(s.cfg.DataDir, "raft.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	rn, err := startRaft(raftOptions{Dir: filepath.Join(s.cfg.DataDir, "raft"), ID: s.id.ID, Bind: s.cfg.RaftListen, Advertise: s.cfg.RaftAdvertise,
		TLS: conf, FSM: s.fsm, Bootstrap: bootstrap, LogOutput: logf, FastTimeouts: s.cfg.FastRaft})
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.rn = rn
	s.mu.Unlock()
	return nil
}

func (s *Server) raft() *raftNode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rn
}

// IsLeader reports raft leadership.
func (s *Server) IsLeader() bool {
	rn := s.raft()
	return rn != nil && rn.r.State() == raft.Leader
}

// verifyLeader confirms leadership with a quorum round trip, cached briefly.
// Bundles are only issued by a verified leader so a partitioned member can
// never make hosts believe the control plane is fresh.
func (s *Server) verifyLeader() error {
	rn := s.raft()
	if rn == nil {
		return errors.New("not bootstrapped")
	}
	s.leaderVer.Lock()
	defer s.leaderVer.Unlock()
	if time.Since(s.leaderVer.at) < 300*time.Millisecond && rn.r.State() == raft.Leader {
		return nil
	}
	if err := rn.r.VerifyLeader().Error(); err != nil {
		return err
	}
	s.leaderVer.at = time.Now()
	return nil
}

// leaderAPI returns the API address of the current leader.
func (s *Server) leaderAPI() (string, string) {
	rn := s.raft()
	if rn == nil {
		return "", ""
	}
	_, id := rn.r.LeaderWithID()
	if id == "" {
		return "", ""
	}
	var addr string
	s.fsm.Read(func(st *State) {
		if m := st.member(string(id)); m != nil {
			addr = m.APIAddr
		}
	})
	return addr, string(id)
}

// propose runs a command on the leader.
func (s *Server) propose(typ, actor string, data any) (*Result, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return s.raft().propose(&Command{Type: typ, TS: nowMs(), Actor: actor, Data: raw}, 10*time.Second)
}

func (s *Server) wake() {
	select {
	case s.kick <- struct{}{}:
	default:
	}
}

// ------------------------------------------------------------------ http

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	// Unauthenticated: health, member info, bootstrap/join (code-protected).
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/member", s.handleMemberInfo)
	mux.HandleFunc("POST /api/v1/bootstrap", s.handleBootstrap)
	mux.HandleFunc("POST /api/v1/join-cluster", s.handleJoinCluster)

	// Host channel: every request is a signed envelope.
	mux.HandleFunc("POST /v1/enroll", s.leaderOnly(s.handleEnroll))
	mux.HandleFunc("POST /v1/bundle", s.leaderOnly(s.handleBundle))
	mux.HandleFunc("POST /v1/observe", s.leaderOnly(s.handleObserve))
	mux.HandleFunc("POST /v1/evidence", s.leaderOnly(s.handleEvidence))
	mux.HandleFunc("POST /v1/binding", s.leaderOnly(s.handleBinding))
	mux.HandleFunc("POST /v1/rotate", s.leaderOnly(s.handleRotate))
	mux.HandleFunc("POST /v1/retrieve-secret", s.leaderOnly(s.handleRetrieveSecret))
	mux.HandleFunc("POST /v1/chunk", s.handleChunk)
	mux.HandleFunc("POST /v1/cert", s.leaderOnly(s.handleCertSign))

	// Federation channel: signed by the peer cluster.
	s.fedRoutes(mux)
	for _, add := range extraRoutes {
		add(s, mux)
	}

	// Operator API.
	s.opRoutes(mux)

	if s.cfg.Web != nil {
		files := http.FileServer(http.FS(s.cfg.Web))
		mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				if _, err := fs.Stat(s.cfg.Web, strings.TrimPrefix(r.URL.Path, "/")); err != nil {
					r.URL.Path = "/"
				}
			}
			// No third-party origins: the console works offline and phones nowhere.
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "no-referrer")
			// Always revalidate: a console upgrade must never mix old and new files.
			w.Header().Set("Cache-Control", "no-cache")
			files.ServeHTTP(w, r)
		}))
	}
	return mux
}

// extraRoutes lets feature files register handlers.
var extraRoutes []func(*Server, *http.ServeMux)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeErr(w http.ResponseWriter, code int, format string, args ...any) {
	writeJSON(w, code, map[string]any{"ok": false, "message": fmt.Sprintf(format, args...)})
}

func readBody(r *http.Request, limit int64, v any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		return err
	}
	if int64(len(body)) > limit {
		return errors.New("request body too large")
	}
	return json.Unmarshal(body, v)
}

// leaderOnly forwards to the leader when this member is a follower.
func (s *Server) leaderOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rn := s.raft()
		if rn == nil {
			writeErr(w, http.StatusServiceUnavailable, "control-plane member %s is not bootstrapped", s.id.ID)
			return
		}
		if rn.r.State() == raft.Leader {
			h(w, r)
			return
		}
		s.forward(w, r)
	}
}

// leaderGet performs an authenticated GET against the leader on behalf of
// the caller (same Authorization header). ok is false on any failure.
func (s *Server) leaderGet(r *http.Request, path string, timeout time.Duration) ([]byte, bool) {
	addr, _ := s.leaderAPI()
	if addr == "" {
		return nil, false
	}
	scheme := "http"
	tr := &http.Transport{}
	if s.cfg.TLS {
		conf, err := pki.ClientTLS([]byte(s.credentials().RootCACert))
		if err != nil {
			return nil, false
		}
		scheme, tr.TLSClientConfig = "https", conf
	}
	defer tr.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, scheme+"://"+addr+path, nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("Authorization", r.Header.Get("Authorization"))
	req.Header.Set("X-DH-Forwarded", s.id.ID)
	resp, err := (&http.Client{Transport: tr}).Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, false
	}
	return body, true
}

func (s *Server) forward(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-DH-Forwarded") != "" {
		writeErr(w, http.StatusServiceUnavailable, "no leader (forwarding loop avoided)")
		return
	}
	addr, lid := s.leaderAPI()
	if addr == "" {
		writeErr(w, http.StatusServiceUnavailable, "no control-plane leader: quorum unavailable")
		return
	}
	scheme := "http"
	var transport http.RoundTripper = http.DefaultTransport
	if s.cfg.TLS {
		scheme = "https"
		conf, err := pki.ClientTLS([]byte(s.credentials().RootCACert))
		if err != nil {
			writeErr(w, 500, "%v", err)
			return
		}
		transport = &http.Transport{TLSClientConfig: conf}
	}
	target, _ := url.Parse(scheme + "://" + addr)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		writeErr(w, http.StatusBadGateway, "forward to leader %s: %v", lid, err)
	}
	r.Header.Set("X-DH-Forwarded", s.id.ID)
	proxy.ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	out := map[string]any{"member": s.id.ID, "pub": s.id.PubString(), "protocol": api.ProtocolVersion, "api": s.cfg.APIAdvertise, "raft": s.cfg.RaftAdvertise}
	rn := s.raft()
	if rn == nil {
		out["state"] = "awaiting-bootstrap"
		writeJSON(w, 200, out)
		return
	}
	addr, lid := s.leaderAPI()
	out["state"] = strings.ToLower(rn.r.State().String())
	out["leader"] = lid
	out["leaderApi"] = addr
	s.fsm.Read(func(st *State) {
		out["cluster"] = st.Cluster
		out["index"] = st.Index
		out["frozen"] = st.Frozen
	})
	out["appliedIndex"] = rn.r.AppliedIndex()
	out["lastContact"] = unixMilliOrZero(rn.r.LastContact())
	writeJSON(w, 200, out)
}

func (s *Server) handleMemberInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{
		"id": s.id.ID, "pub": s.id.PubString(), "apiAddr": s.cfg.APIAdvertise, "raftAddr": s.cfg.RaftAdvertise,
		"bootstrapped": s.raft() != nil,
	})
}

type bootstrapReq struct {
	Code       string             `json:"code"`
	Cluster    string             `json:"cluster"`
	Root       string             `json:"root"`
	RootCACert string             `json:"rootCaCert"`
	MemberCert string             `json:"memberCert"`
	Roster     *envelope.Envelope `json:"roster"`
}

func (s *Server) acceptCredentials(req bootstrapReq, bootstrap bool) error {
	// One bootstrap at a time: the code is single-use.
	s.bootMu.Lock()
	defer s.bootMu.Unlock()
	if s.raft() != nil {
		return errors.New("member already has credentials")
	}
	if s.code == "" || subtle.ConstantTimeCompare([]byte(req.Code), []byte(s.code)) != 1 {
		return errors.New("bootstrap code does not match")
	}
	cert, err := pki.ParseCert([]byte(req.MemberCert))
	if err != nil {
		return err
	}
	if cert.Subject.CommonName != s.id.ID {
		return errors.New("member certificate is for a different member")
	}
	if _, err := pki.MemberTLS(s.id, []byte(req.MemberCert), []byte(req.RootCACert), nil); err != nil {
		return err
	}
	creds := &Credentials{Cluster: req.Cluster, Root: req.Root, RootCACert: req.RootCACert, MemberCert: req.MemberCert, Bootstrap: bootstrap}
	b, _ := json.MarshalIndent(creds, "", "  ")
	if err := os.WriteFile(s.credsPath(), b, 0o600); err != nil {
		return err
	}
	s.mu.Lock()
	s.creds = creds
	s.mu.Unlock()
	_ = os.Remove(filepath.Join(s.cfg.DataDir, "bootstrap.fingerprint"))
	_ = os.Remove(filepath.Join(s.cfg.DataDir, "bootstrap.code"))
	s.code = ""
	return s.startRaft(bootstrap)
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	var req bootstrapReq
	if err := readBody(r, 1<<20, &req); err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	if _, err := verifyRoster(req.Roster, req.Root); err != nil {
		writeErr(w, 400, "roster: %v", err)
		return
	}
	if err := s.acceptCredentials(req, true); err != nil {
		writeErr(w, 403, "%v", err)
		return
	}
	deadline := time.Now().Add(15 * time.Second)
	for !s.IsLeader() {
		if time.Now().After(deadline) {
			writeErr(w, 503, "bootstrap: no leadership after 15s")
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	res, err := s.propose("init", "operator", initData{Cluster: req.Cluster, Root: req.Root, RootCACert: req.RootCACert, Roster: req.Roster})
	if err != nil {
		writeErr(w, 500, "%v", err)
		return
	}
	writeJSON(w, 200, res)
}

func (s *Server) handleJoinCluster(w http.ResponseWriter, r *http.Request) {
	var req bootstrapReq
	if err := readBody(r, 1<<20, &req); err != nil {
		writeErr(w, 400, "%v", err)
		return
	}
	if err := s.acceptCredentials(req, false); err != nil {
		writeErr(w, 403, "%v", err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "message": "member credentials stored; waiting for the leader to add this member"})
}

// ------------------------------------------------------------ operator auth

type authz struct {
	Actor  string
	Signer string
}

// authorize verifies the bearer capability against the cluster root.
func (s *Server) authorize(r *http.Request, action string) (*authz, error) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return nil, errors.New("missing bearer capability")
	}
	tok, err := capability.Decode(strings.TrimPrefix(h, "Bearer "))
	if err != nil {
		return nil, err
	}
	var root, cluster string
	s.fsm.Read(func(st *State) { root, cluster = st.Root, st.Cluster })
	if root == "" {
		return nil, errors.New("cluster is not initialized")
	}
	res := capability.Verify(tok, []string{root}, capability.Request{Action: action, Resource: "cluster/" + cluster, Now: nowMs()})
	if !res.OK {
		return nil, fmt.Errorf("capability: %s", res.Reason)
	}
	actor := "operator"
	if n := res.Last.Note; n != "" {
		actor = "operator:" + n
	}
	return &authz{Actor: actor, Signer: res.Signers[len(res.Signers)-1]}, nil
}

func (s *Server) op(action string, h func(w http.ResponseWriter, r *http.Request, a *authz)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, err := s.authorize(r, action)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "%v", err)
			return
		}
		h(w, r, a)
	}
}

// opWrite is op + leader forwarding.
func (s *Server) opWrite(action string, h func(w http.ResponseWriter, r *http.Request, a *authz)) http.HandlerFunc {
	return s.leaderOnly(s.op(action, h))
}

// -------------------------------------------------------------- background

func (s *Server) backgroundLoop() {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
		}
		s.ensureMesh()
		if !s.IsLeader() {
			continue
		}
		s.leaderHousekeeping()
	}
}
