// Package node implements dh-noded, the sovereign host agent.
//
// The control plane proposes; this agent decides. It pins the cluster root
// from its join token, verifies every bundle, roster, assignment and
// capability against that root, applies its own local policy, runs what it
// admits, and reports what it actually observes in signed, sequenced
// observations. It keeps its own hash-chained ledger and keeps admitted work
// running when the control plane is unreachable.
package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
	"decentralized.host/pkg/mesh"
	"decentralized.host/pkg/policy"
	"decentralized.host/pkg/runtime"
	"decentralized.host/pkg/storage"
)

// Config configures the agent.
type Config struct {
	DataDir          string
	JoinToken        string
	Name             string
	Region           string
	Zone             string
	Host             string
	Tiers            []string
	Roles            []string
	Features         []string
	CPUMilli         int64
	MemBytes         int64
	MeshListen       string // UDP bind for WireGuard, e.g. 0.0.0.0:51821
	MeshAdvertise    string // endpoint peers dial, e.g. 127.0.0.1:51821
	StatusListen     string // local debug/chaos API (loopback)
	Chaos            bool   // enable fault-injection endpoints on the status API
	Tick             time.Duration
	AntiEntropyEvery time.Duration // how often each retained snapshot is re-verified (default 60s)
	Edge             EdgeConfig
	Logger           *log.Logger
}

// EdgeConfig configures the edge role.
type EdgeConfig struct {
	HTTPListen    string
	HTTPSListen   string
	ACMEDirectory string
	ACMEEmail     string
	ACMECACert    string // PEM file trusted for the ACME directory (e.g. Pebble)
	ACMEDNS       string // DNS-01 provider, e.g. "challtestsrv=http://127.0.0.1:8055"
}

// Admitted is a workload this host has accepted and started.
type Admitted struct {
	ID         string               `json:"id"`
	App        string               `json:"app"`
	Replica    int64                `json:"replica"`
	Generation int64                `json:"generation"`
	Runtime    string               `json:"runtime"`
	Image      string               `json:"image"`
	Digest     string               `json:"digest"`
	CPUMilli   int64                `json:"cpuMilli"`
	MemBytes   int64                `json:"memBytes"`
	Inst       runtime.Instance     `json:"inst"`
	Health     api.Health           `json:"health"`
	Ports      []api.Port           `json:"ports"`
	Volumes    []api.AssignedVolume `json:"volumes"`
	Restarts   int64                `json:"restarts"`
	LastState  string               `json:"lastState"`
	LastExit   int64                `json:"lastExit"`
	LastDetail string               `json:"lastDetail"`
	Backoff    int64                `json:"backoff"`
	AdmittedAt int64                `json:"admittedAt"`
	Assignment json.RawMessage      `json:"assignment"`
	Stopped    bool                 `json:"stopped"`
}

// VolState tracks one volume duty.
type VolState struct {
	LastSnapshot   string           `json:"lastSnapshot"`
	LastRoot       string           `json:"lastRoot"`
	LastSnapshotAt int64            `json:"lastSnapshotAt"`
	Restored       string           `json:"restored"`
	Evidence       map[string]int64 `json:"evidence"` // snapshot -> last evidence sent
}

// State is persisted after every tick.
type State struct {
	NodeID     string               `json:"nodeId"` // genesis dh1 id; stable across key rotation
	Cluster    string               `json:"cluster"`
	Root       string               `json:"root"`
	RootCACert string               `json:"rootCaCert"`
	Endpoints  []string             `json:"endpoints"`
	TLS        bool                 `json:"tls"`
	JoinCap    string               `json:"joinCap"`
	Enrolled   bool                 `json:"enrolled"`
	LastIndex  int64                `json:"lastIndex"`
	LastIssued int64                `json:"lastIssued"`
	Seq        int64                `json:"seq"`
	Admitted   map[string]*Admitted `json:"admitted"`
	MeshPorts  map[string]int64     `json:"meshPorts"`
	NextMesh   int64                `json:"nextMesh"`
	Volumes    map[string]*VolState `json:"volumes"`
	Decisions  map[string]string    `json:"decisions"` // last journaled decision key per assignment
	Mode       string               `json:"mode"`
	// Retiring holds old-generation instances still serving while their
	// replacement warms up (make-before-break). An agent that restarts
	// mid-handover stops them instead of orphaning them.
	Retiring map[string]*Retiring `json:"retiring"`
}

// Retiring is an old instance kept alive during a handover.
type Retiring struct {
	Runtime    string           `json:"runtime"`
	Inst       runtime.Instance `json:"inst"`
	Generation int64            `json:"generation"`
}

// decision is the latest admission outcome for an assignment.
type decision struct {
	Gen     int64
	Allowed bool
	Hold    bool
	Code    string
	Reason  string
	Checks  []api.Check
	At      int64
}

// Agent is a running host agent.
type Agent struct {
	cfg       Config
	log       *log.Logger
	id        *identity.Identity
	pol       policy.Policy
	journal   *audit.Journal
	cas       *storage.CAS
	cp        *cpClient
	proc      *runtime.Process
	docker    *runtime.Docker
	dockerOK  atomic.Bool
	dockerVer atomic.Value

	mu               sync.RWMutex
	st               State
	bundle           *api.Bundle
	bundleEnv        *envelope.Envelope
	trusted          bool
	trustWhy         string
	fresh            bool
	skewMs           int64
	lastBundleAt     time.Time
	decisions        map[string]*decision
	health           map[string]*api.HealthObs
	healthAt         map[string]time.Time
	lastErr          string
	mode, modeDetail string
	lastBuffered     string
	fetching         map[string]time.Time // artifact digest -> fetch started
	idCheck          time.Time

	meshMu        sync.RWMutex
	dev           *mesh.Device
	gossip        *mesh.Gossip
	peerIPs       map[string]string // mesh ip -> node id (bindings verified)
	peerInfo      map[string]api.Peer
	bindOK        map[string]bool
	rtt           map[string][2]int64 // node -> rtt us, measured at
	forwards      map[string]*forwarder
	cutover       map[string]bool // assignment -> forwarder stays on the old instance until handover
	joining       atomic.Bool     // a gossip join is in flight
	badHolder     map[string]time.Time
	peerSrv       *peerServer
	sentBinding   string
	sentEnv       *envelope.Envelope
	sentBindingAt time.Time
	refused       string // last refused service connection (diagnostics)

	edge    edgeRunner
	storeMu sync.Mutex

	outbox   *outbox
	facts    atomic.Value // api.Facts
	factsAt  time.Time
	clockOff atomic.Int64 // chaos: milliseconds added to the host clock
	chaos    chaosFlags
	stopc    chan struct{}
}

type chaosFlags struct {
	dropCP     atomic.Bool
	failHealth sync.Map // assignment -> bool
}

// New prepares the agent (identity, policy, journal, storage).
func New(cfg Config) (*Agent, error) {
	if cfg.Tick == 0 {
		cfg.Tick = time.Second
	}
	if cfg.AntiEntropyEvery == 0 {
		cfg.AntiEntropyEvery = time.Minute
	}
	if cfg.Logger == nil {
		cfg.Logger = log.New(os.Stderr, "dh-noded ", log.LstdFlags|log.Lmsgprefix)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, err
	}
	a := &Agent{cfg: cfg, log: cfg.Logger, proc: runtime.NewProcess(), docker: runtime.NewDocker(),
		decisions: map[string]*decision{}, health: map[string]*api.HealthObs{}, healthAt: map[string]time.Time{},
		peerIPs: map[string]string{}, peerInfo: map[string]api.Peer{}, bindOK: map[string]bool{}, rtt: map[string][2]int64{},
		forwards: map[string]*forwarder{}, cutover: map[string]bool{}, badHolder: map[string]time.Time{}, stopc: make(chan struct{})}
	var err error
	if a.id, err = identity.LoadOrCreate(filepath.Join(cfg.DataDir, "identity")); err != nil {
		return nil, err
	}
	if a.pol, err = policy.LoadOrCreate(filepath.Join(cfg.DataDir, "policy.yaml")); err != nil {
		return nil, err
	}
	if a.journal, err = audit.OpenJournal(filepath.Join(cfg.DataDir, "journal.jsonl")); err != nil {
		return nil, err
	}
	if a.cas, err = storage.OpenCAS(filepath.Join(cfg.DataDir, "cas"), a.pol.StorageQuotaBytes); err != nil {
		return nil, err
	}
	if a.outbox, err = openOutbox(filepath.Join(cfg.DataDir, "outbox.jsonl")); err != nil {
		return nil, err
	}
	if err := a.loadState(); err != nil {
		return nil, err
	}
	if a.st.NodeID == "" {
		a.st.NodeID = a.id.ID
	}
	if cfg.JoinToken != "" {
		t, err := DecodeJoinToken(cfg.JoinToken)
		if err != nil {
			return nil, err
		}
		switch {
		case a.st.Root == "" || !a.st.Enrolled:
			// Not enrolled yet (for example the first token was already used):
			// a new token may replace the pending one.
			prev := a.st.Root
			a.st.Cluster, a.st.Root, a.st.RootCACert, a.st.Endpoints, a.st.TLS, a.st.JoinCap = t.Cluster, t.Root, t.RootCACert, t.Endpoints, t.TLS, t.Capability
			detail := fmt.Sprintf("pinned root %s from join token; control plane %v", short(t.Root), t.Endpoints)
			if prev != "" && prev != t.Root {
				detail += fmt.Sprintf(" (replaces root %s pinned by an earlier token that never enrolled)", short(prev))
			}
			a.record("join-token", "cluster/"+t.Cluster, 0, detail)
		case t.Root != a.st.Root:
			return nil, fmt.Errorf("this host is enrolled in cluster %s under root %s; refusing a join token for root %s (to move the host, stop it and start from an empty --data directory)", a.st.Cluster, short(a.st.Root), short(t.Root))
		default:
			a.log.Printf("already enrolled in cluster %s; ignoring --join", a.st.Cluster)
		}
	}
	if a.st.Root == "" {
		return nil, errors.New("this host has not joined a cluster: pass --join <dhjoin1 token>")
	}
	if a.cp, err = newCPClient(a.st.Endpoints, a.st.TLS, a.st.RootCACert); err != nil {
		return nil, err
	}
	if b := a.journal.Corrupt(); b != nil {
		a.log.Printf("HOST LEDGER CORRUPT: %s at seq %d (expected %s, actual %s); refusing new work", b.Reason, b.Seq, b.Expected, b.Actual)
	}
	return a, nil
}

// ID returns the stable host id.
func (a *Agent) ID() string { return a.st.NodeID }

func (a *Agent) now() int64 { return time.Now().UnixMilli() + a.clockOff.Load() }

func (a *Agent) statePath() string { return filepath.Join(a.cfg.DataDir, "state.json") }

func (a *Agent) loadState() error {
	a.st = State{Admitted: map[string]*Admitted{}, MeshPorts: map[string]int64{}, Volumes: map[string]*VolState{}, Decisions: map[string]string{}, NextMesh: 20000}
	b, err := os.ReadFile(a.statePath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &a.st); err != nil {
		return fmt.Errorf("host state: %w", err)
	}
	if a.st.Admitted == nil {
		a.st.Admitted = map[string]*Admitted{}
	}
	if a.st.MeshPorts == nil {
		a.st.MeshPorts = map[string]int64{}
	}
	if a.st.Volumes == nil {
		a.st.Volumes = map[string]*VolState{}
	}
	if a.st.Decisions == nil {
		a.st.Decisions = map[string]string{}
	}
	if a.st.NextMesh == 0 {
		a.st.NextMesh = 20000
	}
	return nil
}

func (a *Agent) saveState() {
	a.mu.RLock()
	b, err := json.MarshalIndent(&a.st, "", "  ")
	a.mu.RUnlock()
	if err != nil {
		return
	}
	tmp := a.statePath() + ".tmp"
	if f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600); err == nil {
		_, _ = f.Write(b)
		_ = f.Sync()
		f.Close()
		_ = os.Rename(tmp, a.statePath())
	}
}

// record appends to the host ledger (the authoritative host-side evidence).
func (a *Agent) record(action, resource string, gen int64, detail string) {
	_, err := a.journal.Append(audit.Entry{TS: a.now(), Actor: a.st.NodeID, Source: audit.SourceHost, Action: action, Resource: resource, Generation: gen, Detail: detail})
	if err != nil {
		a.log.Printf("ledger: %v", err)
	}
}

// sign signs as the stable host id with the current key. It must not be
// called while holding a.mu.
func (a *Agent) sign(kind string, payload any) (*envelope.Envelope, error) {
	return envelope.Sign(a.key(), a.st.NodeID, kind, payload)
}

func (a *Agent) key() *identity.Identity {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.id
}

// Run executes the agent loop until ctx ends.
func (a *Agent) Run(ctx context.Context) error {
	a.probeFacts()
	a.readopt()
	if a.cfg.StatusListen != "" {
		if err := a.startStatus(); err != nil {
			return err
		}
	}
	go a.storageLoop(ctx)
	go a.meshLoop(ctx)
	if contains(a.cfg.Roles, "edge") {
		if err := a.startEdge(ctx); err != nil {
			return fmt.Errorf("edge: %w", err)
		}
	}
	t := time.NewTicker(a.cfg.Tick)
	defer t.Stop()
	a.log.Printf("host %s (%s) running; root %s; control plane %v", a.st.NodeID, a.cfg.Name, short(a.st.Root), a.st.Endpoints)
	for {
		a.tick()
		select {
		case <-ctx.Done():
			close(a.stopc)
			a.shutdown()
			return nil
		case <-t.C:
		}
	}
}

func (a *Agent) shutdown() {
	a.saveState()
	a.meshMu.Lock()
	for _, f := range a.forwards {
		f.close()
	}
	if a.gossip != nil {
		_ = a.gossip.Close()
	}
	if a.peerSrv != nil {
		a.peerSrv.close()
	}
	if a.dev != nil {
		a.dev.Close()
	}
	a.meshMu.Unlock()
	// Admitted workloads keep running: the agent stopping is not the
	// workload stopping. They are re-adopted on the next start.
}

// readopt re-attaches to workloads that survived an agent restart, and
// stops old instances left over from an interrupted handover.
func (a *Agent) readopt() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, r := range a.st.Retiring {
		_ = a.runtimeFor(r.Runtime).Stop(r.Inst, 5*time.Second)
		delete(a.st.Retiring, id)
		go a.record("workload-stop", "app/"+id, r.Generation, "old generation left running by an interrupted handover; stopped at agent start")
	}
	for id, ad := range a.st.Admitted {
		if ad.Stopped {
			continue
		}
		st := a.runtimeFor(ad.Runtime).Status(ad.Inst)
		detail := fmt.Sprintf("re-adopted %s generation %d after agent restart: %s", id, ad.Generation, st.State)
		if ad.Runtime == "process" {
			detail += fmt.Sprintf(" (pid %d, start time verified)", ad.Inst.PID)
		}
		ad.LastState = st.State
		go a.record("workload-readopt", "app/"+id, ad.Generation, detail)
	}
}

func (a *Agent) runtimeFor(name string) runtime.Runtime {
	if name == "docker" {
		return a.docker
	}
	return a.proc
}

// reloadIdentity picks up a key installed by `dh-noded rotate-key` without
// a restart. The stable host id never changes.
func (a *Agent) reloadIdentity() {
	if time.Since(a.idCheck) < 3*time.Second {
		return
	}
	a.idCheck = time.Now()
	id, err := identity.Load(filepath.Join(a.cfg.DataDir, "identity"))
	if err != nil || id.PubString() == a.id.PubString() {
		return
	}
	a.mu.Lock()
	a.id = id
	a.mu.Unlock()
	a.record("key-reload", "node/"+a.st.NodeID, 0, "now signing with rotated key "+short(id.PubString()))
}

// tick is one reconciliation pass.
func (a *Agent) tick() {
	a.reloadIdentity()
	a.syncBundle()
	a.reconcileWorkloads()
	a.checkHealth()
	a.observe()
	a.saveState()
}

func (a *Agent) probeFacts() {
	f := api.Facts{OS: goruntime.GOOS, Arch: arch(), CPUs: int64(goruntime.NumCPU()), MemBytes: a.cfg.MemBytes}
	f.Kernel = kernelVersion()
	f.UDP443 = udpListening(443)
	f.Runtimes = []string{"process"}
	if ok, ver := a.docker.Available(); ok {
		a.dockerOK.Store(true)
		f.Docker = ver
		f.Runtimes = append(f.Runtimes, "docker")
	} else {
		a.dockerOK.Store(false)
	}
	f.Probes = probeTools()
	a.facts.Store(f)
	a.factsAt = time.Now()
}

func arch() string {
	switch goruntime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	}
	return goruntime.GOARCH
}

func udpListening(port int) bool {
	c, err := net.ListenPacket("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		// Either something already listens there, or we lack permission.
		// Only report LISTENING when a probe packet is answered by a socket.
		return udpAnswered(port)
	}
	c.Close()
	return false
}

func udpAnswered(port int) bool {
	c, err := net.DialTimeout("udp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
	if err != nil {
		return false
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(300 * time.Millisecond))
	if _, err := c.Write([]byte{0}); err != nil {
		return false
	}
	buf := make([]byte, 16)
	_, err = c.Read(buf)
	// A connected UDP socket reports ICMP port-unreachable as an error, so a
	// timeout (no error surfaced) means some process holds the port.
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func short(s string) string {
	if len(s) > 3 && s[:3] == "b3:" {
		s = s[3:]
	}
	if len(s) > 12 {
		return s[:12]
	}
	return s
}
