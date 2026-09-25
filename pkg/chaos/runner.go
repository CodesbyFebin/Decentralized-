package chaos

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/control"
	"decentralized.host/pkg/devcluster"
	"decentralized.host/pkg/envelope"
	"decentralized.host/pkg/identity"
)

// Verdicts.
const (
	Pass = "PASS"
	Fail = "FAIL"
	Skip = "SKIP"
)

// Step is one timestamped event in a scenario's timeline.
type Step struct {
	AtMs  int64  `json:"atMs"` // since scenario start
	Event string `json:"event"`
}

// Report is the signed result of one scenario (envelope kind "chaos-report").
type Report struct {
	ID           string            `json:"id"`
	Scenario     string            `json:"scenario"`
	Title        string            `json:"title"`
	Injection    string            `json:"injection"`
	Invariant    string            `json:"invariant"`
	Observed     string            `json:"observed"`
	Verdict      string            `json:"verdict"`
	Measurements map[string]string `json:"measurements"`
	Recovery     []Step            `json:"recovery"`
	Evidence     []string          `json:"evidence"`
	Started      int64             `json:"started"`
	DurationMs   int64             `json:"durationMs"`
	Environment  map[string]string `json:"environment"`
}

// ErrSkip marks a scenario whose prerequisites are missing (never a PASS).
var ErrSkip = errors.New("skipped")

// Scenario is one failure experiment.
type Scenario struct {
	ID        string
	Title     string
	Injection string
	Invariant string
	CPs       int
	Hosts     int
	Edges     int
	Proxy     bool // route WireGuard through fault proxies
	LostAfter string
	HostArgs  []string
	Needs     []string // "docker", "postgres"
	Postgres  bool     // start a Postgres container and wire it as the evidence mirror
	App       string   // manifest template (%s = beacon image); "" = default web app
	Run       func(x *Ctx) error
}

// Ctx is handed to a scenario.
type Ctx struct {
	C       *devcluster.Cluster
	Ctl     *Controller
	Image   string
	PG      string // postgres container name (Scenario.Postgres)
	rep     *Report
	start   time.Time
	mu      sync.Mutex
	Opt     RunOptions
	proxies []*Proxy
}

// Step records a timeline event.
func (x *Ctx) Step(format string, args ...any) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.rep.Recovery = append(x.rep.Recovery, Step{AtMs: time.Since(x.start).Milliseconds(), Event: fmt.Sprintf(format, args...)})
	if x.Opt.Verbose {
		fmt.Printf("    %6dms  %s\n", time.Since(x.start).Milliseconds(), fmt.Sprintf(format, args...))
	}
}

// Measure records a measurement.
func (x *Ctx) Measure(k string, format string, args ...any) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.rep.Measurements[k] = fmt.Sprintf(format, args...)
}

// Evidence records an evidence reference.
func (x *Ctx) Evidence(format string, args ...any) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.rep.Evidence = append(x.rep.Evidence, fmt.Sprintf(format, args...))
}

// Observed sets the observed-behavior summary.
func (x *Ctx) Observed(format string, args ...any) { x.rep.Observed = fmt.Sprintf(format, args...) }

// RunOptions configures a run.
type RunOptions struct {
	Dir     string
	Bin     string
	Base    int
	Verbose bool
	Submit  bool // submit reports to each scenario's own cluster
	Keep    bool // keep cluster directories
}

// Runner executes scenarios and signs reports.
type Runner struct {
	opt RunOptions
	id  *identity.Identity
}

// NewRunner loads or creates the runner identity (reports are signed by it).
func NewRunner(opt RunOptions) (*Runner, error) {
	if err := os.MkdirAll(opt.Dir, 0o700); err != nil {
		return nil, err
	}
	id, err := identity.LoadOrCreate(filepath.Join(opt.Dir, "runner"))
	if err != nil {
		return nil, err
	}
	if opt.Base == 0 {
		opt.Base = 30000
	}
	return &Runner{opt: opt, id: id}, nil
}

// ID returns the runner identity.
func (r *Runner) ID() *identity.Identity { return r.id }

var runSeq atomic.Int64

// Run executes one scenario end to end and returns its signed report.
func (r *Runner) Run(s Scenario) (*Report, *envelope.Envelope) {
	n := runSeq.Add(1)
	rep := &Report{ID: fmt.Sprintf("%s-%d", s.ID, time.Now().UnixMilli()), Scenario: s.ID, Title: s.Title, Injection: s.Injection, Invariant: s.Invariant,
		Measurements: map[string]string{}, Started: time.Now().UnixMilli(),
		Environment: map[string]string{"os": goruntime.GOOS, "arch": goruntime.GOARCH, "go": goruntime.Version(), "cpus": fmt.Sprint(goruntime.NumCPU()),
			"topology": fmt.Sprintf("%d control-plane member(s), %d host(s), %d edge(s)", max1(s.CPs), s.Hosts, s.Edges)}}
	x := &Ctx{rep: rep, start: time.Now(), Opt: r.opt, Ctl: NewController()}
	finish := func(verdict string, err error) (*Report, *envelope.Envelope) {
		rep.Verdict = verdict
		if err != nil && rep.Observed == "" {
			rep.Observed = err.Error()
		} else if err != nil {
			rep.Observed += " — " + err.Error()
		}
		rep.DurationMs = time.Since(x.start).Milliseconds()
		if x.C != nil {
			r.collectAudit(x)
			env, _ := envelope.Sign(r.id, "", envelope.KindChaosReport, rep)
			if r.opt.Submit && env != nil {
				var res map[string]any
				if err := x.C.Op.Do("POST", "/api/v1/chaos/report", env, &res); err != nil {
					rep.Evidence = append(rep.Evidence, "report submission failed: "+err.Error())
				}
			}
			for _, p := range x.proxies {
				p.Close()
			}
			x.C.Down()
			if !r.opt.Keep {
				os.RemoveAll(x.C.Opt.Dir)
			}
		}
		env, _ := envelope.Sign(r.id, "", envelope.KindChaosReport, rep)
		b, _ := json.MarshalIndent(env, "", "  ")
		_ = os.WriteFile(filepath.Join(r.opt.Dir, rep.ID+".json"), b, 0o644)
		return rep, env
	}
	for _, need := range s.Needs {
		if why := prerequisite(need); why != "" {
			return finish(Skip, fmt.Errorf("prerequisite %s unavailable: %s", need, why))
		}
	}
	base := r.opt.Base + int(n%20)*1000
	opt := devcluster.Options{Dir: filepath.Join(r.opt.Dir, rep.ID), Bin: r.opt.Bin, Cluster: "chaos", CPs: max1(s.CPs), Hosts: s.Hosts, Edges: s.Edges,
		Base: base, Chaos: true, LostAfter: s.LostAfter, Quiet: true}
	if len(s.HostArgs) > 0 {
		opt.HostArgs = func(int, string) []string { return s.HostArgs }
	}
	if s.Postgres {
		name := fmt.Sprintf("dh-chaos-pg-%d", base)
		url, err := startPostgres(name, base+600)
		if err != nil {
			return finish(Skip, fmt.Errorf("%w: postgres: %v", ErrSkip, err))
		}
		x.PG, opt.Postgres = name, url
		x.Step("postgres container %s ready", name)
		defer exec.Command("docker", "rm", "-f", name).Run()
	}
	if s.Proxy {
		opt.MeshAdvertise = func(i int, name, listen string) string {
			addr := fmt.Sprintf("127.0.0.1:%d", base+500+i)
			p, err := x.Ctl.NewProxy(name, addr, listen)
			if err != nil {
				return listen
			}
			x.proxies = append(x.proxies, p)
			return addr
		}
	}
	x.Step("starting %d control-plane member(s), %d host(s), %d edge(s)", opt.CPs, s.Hosts, s.Edges)
	c, err := devcluster.Up(opt)
	x.C = c
	if err != nil {
		return finish(Fail, fmt.Errorf("cluster did not start: %w", err))
	}
	x.Step("cluster ready")
	if s.App != "-" {
		image, _, err := c.Op.PushArtifact(filepath.Join(r.opt.Bin, "dh-beacon"), "dh-beacon", true)
		if err != nil {
			return finish(Fail, err)
		}
		x.Image = image
		tmpl := s.App
		if tmpl == "" {
			tmpl = defaultApp
		}
		if strings.Contains(tmpl, "%s") {
			res, err := c.Op.Apply([]byte(fmt.Sprintf(tmpl, image)))
			if err != nil || !res.OK {
				return finish(Fail, fmt.Errorf("apply: %v %s", err, res.Message))
			}
		}
	}
	err = s.Run(x)
	switch {
	case errors.Is(err, ErrSkip):
		return finish(Skip, err)
	case err != nil:
		return finish(Fail, err)
	}
	return finish(Pass, nil)
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// collectAudit attaches the audit entries produced during the scenario and
// a fresh verification of the whole ledger.
func (r *Runner) collectAudit(x *Ctx) {
	var out struct {
		Entries []audit.Entry `json:"entries"`
	}
	if err := x.C.Op.Do("GET", "/api/v1/audit?limit=5000", nil, &out); err != nil {
		x.Evidence("audit unavailable: %v", err)
		return
	}
	since := x.rep.Started
	interesting := map[string]bool{"node-lost": true, "node-recovered": true, "workload-start": true, "workload-stop": true, "workload-oom": true,
		"workload-fail": true, "node-refuse": true, "host-mode": true, "assignment-sign": true, "assignment-stop": true, "snapshot-commit": true,
		"storage-repair": true, "storage-corruption": true, "route-ejected": true, "route-routing": true, "route-draining": true, "node-revoke": true,
		"cp-restore": true, "roster-update": true, "health-fail": true, "health-pass": true, "key-revoke": true}
	count := 0
	for _, e := range out.Entries {
		if e.TS < since || !interesting[e.Action] {
			continue
		}
		if count < 40 {
			x.Evidence("audit #%d %s %s %s: %s (%s)", e.Seq, time.UnixMilli(e.TS).UTC().Format("15:04:05.000"), e.Action, e.Resource, trunc(e.Detail, 120), short(e.Hash))
		}
		count++
	}
	if count > 40 {
		x.Evidence("… %d more audit entries in the window", count-40)
	}
	if b := audit.Verify(out.Entries, 0, audit.Genesis); b != nil {
		x.Evidence("AUDIT CHAIN BROKEN: %s at %d", b.Reason, b.Seq)
		if x.rep.Verdict == Pass {
			x.rep.Verdict = Fail
		}
	} else if len(out.Entries) > 0 {
		x.Evidence("audit chain verified: %d entries, head %s", len(out.Entries), short(out.Entries[len(out.Entries)-1].Hash))
	}
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func short(s string) string {
	s = strings.TrimPrefix(s, "b3:")
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// startPostgres runs a disposable Postgres and waits until it accepts queries.
func startPostgres(name string, port int) (string, error) {
	_ = exec.Command("docker", "rm", "-f", name).Run()
	out, err := exec.Command("docker", "run", "-d", "--name", name, "-e", "POSTGRES_PASSWORD=dh", "-p", fmt.Sprintf("127.0.0.1:%d:5432", port), "postgres:16-alpine").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker run: %s", strings.TrimSpace(string(out)))
	}
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if exec.Command("docker", "exec", name, "pg_isready", "-U", "postgres", "-h", "127.0.0.1").Run() == nil {
			time.Sleep(time.Second)
			return fmt.Sprintf("postgres://postgres:dh@127.0.0.1:%d/postgres?sslmode=disable", port), nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", errors.New("postgres did not become ready")
}

func prerequisite(need string) string {
	switch need {
	case "docker":
		if dockerVersion() == "" {
			return "docker daemon not reachable"
		}
	}
	return ""
}

// ------------------------------------------------------------- helpers

const defaultApp = `apiVersion: dh/v1
kind: Application
metadata: {name: web}
spec:
  replicas: 3
  image: %s
  resources: {cpu: 100m, mem: 32Mi}
  placement: {tiers: [trusted], spread: failure-domain, antiAffinity: hard}
  ports: [{name: http}]
  ingress: [{host: web.chaos.test, port: http, tls: none}]
  health: {http: /healthz, interval: 1s}
`

// App returns the view of an app.
func App(v *control.View, name string) *control.AppView {
	for i := range v.Apps {
		if v.Apps[i].Name == name {
			return &v.Apps[i]
		}
	}
	return nil
}

// Node returns a node view by name.
func Node(v *control.View, name string) *control.NodeView {
	for i := range v.Nodes {
		if v.Nodes[i].Name == name {
			return &v.Nodes[i]
		}
	}
	return nil
}

// WaitRunning waits for n observed replicas.
func (x *Ctx) WaitRunning(app string, n int, d time.Duration) error {
	return x.C.WaitFor(d, fmt.Sprintf("%d replica(s) of %s observed running", n, app), func(v *control.View) bool {
		a := App(v, app)
		return a != nil && a.Observed >= n
	})
}

// WaitRouting waits until the edge routes n endpoints for host.
func (x *Ctx) WaitRouting(host string, n int, d time.Duration) error {
	return x.C.WaitFor(d, fmt.Sprintf("edge routing %d endpoint(s) for %s", n, host), func(v *control.View) bool {
		count := 0
		for _, e := range v.Edge.Edges {
			if e.Obs == nil {
				continue
			}
			for _, r := range e.Obs.Routes {
				if r.Host == host {
					for _, ep := range r.Endpoints {
						if ep.State == "routing" {
							count++
						}
					}
				}
			}
		}
		return count >= n
	})
}

// Traffic continuously requests host through the edge.
type Traffic struct {
	ok, fail atomic.Int64
	upstream sync.Map // upstream -> count
	stop     chan struct{}
	wg       sync.WaitGroup
	lat      []time.Duration
	errs     []string
	x        *Ctx
	mu       sync.Mutex
}

// Errors returns a sample of failed requests (first 10).
func (tr *Traffic) Errors() []string {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return append([]string(nil), tr.errs...)
}

// StartTraffic begins sending requests every interval.
func (x *Ctx) StartTraffic(host string, every time.Duration) *Traffic {
	tr := &Traffic{stop: make(chan struct{}), x: x}
	tr.wg.Add(1)
	go func() {
		defer tr.wg.Done()
		cl := &http.Client{Timeout: 5 * time.Second}
		for {
			select {
			case <-tr.stop:
				return
			default:
			}
			req, _ := http.NewRequest("GET", "http://"+x.C.Edge+"/", nil)
			req.Host = host
			start := time.Now()
			resp, err := cl.Do(req)
			if err != nil || resp.StatusCode != 200 {
				why := ""
				if err != nil {
					why = err.Error()
				} else {
					b, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
					why = fmt.Sprintf("HTTP %d %s (upstream %s)", resp.StatusCode, strings.TrimSpace(string(b)), resp.Header.Get("X-DH-Upstream"))
				}
				tr.mu.Lock()
				first := len(tr.errs) == 0
				if len(tr.errs) < 10 {
					tr.errs = append(tr.errs, fmt.Sprintf("%s: %s", time.Now().Format("15:04:05.000"), why))
				}
				tr.mu.Unlock()
				if first && len(x.C.Edges) > 0 {
					// Capture the edge's own routing table at the first failure.
					if st, err := x.C.HostStatus(x.C.Edges[0].Name); err == nil {
						b, _ := json.Marshal(st.Edge)
						tr.mu.Lock()
						tr.errs = append(tr.errs, "edge routing table at first failure: "+string(b))
						tr.mu.Unlock()
					}
				}
			}
			if err == nil && resp.StatusCode == 200 {
				tr.ok.Add(1)
				up := resp.Header.Get("X-DH-Upstream")
				v, _ := tr.upstream.LoadOrStore(up, new(atomic.Int64))
				v.(*atomic.Int64).Add(1)
				tr.mu.Lock()
				tr.lat = append(tr.lat, time.Since(start))
				tr.mu.Unlock()
			} else {
				tr.fail.Add(1)
			}
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
			time.Sleep(every)
		}
	}()
	return tr
}

// Stop ends traffic and returns ok/fail counts.
func (tr *Traffic) Stop() (int64, int64) {
	close(tr.stop)
	tr.wg.Wait()
	if tr.x != nil {
		for _, e := range tr.Errors() {
			tr.x.Evidence("failed request %s", e)
		}
	}
	return tr.ok.Load(), tr.fail.Load()
}

// Counts returns current counts.
func (tr *Traffic) Counts() (int64, int64) { return tr.ok.Load(), tr.fail.Load() }

// Latency returns p50 and p99 of successful requests.
func (tr *Traffic) Latency() (time.Duration, time.Duration) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if len(tr.lat) == 0 {
		return 0, 0
	}
	l := append([]time.Duration(nil), tr.lat...)
	sort.Slice(l, func(i, j int) bool { return l[i] < l[j] })
	return l[len(l)/2], l[len(l)*99/100]
}

// ServedBy reports whether any successful request was served by an upstream prefix.
func (tr *Traffic) ServedBy(prefix string) int64 {
	var n int64
	tr.upstream.Range(func(k, v any) bool {
		if strings.HasPrefix(k.(string), prefix) {
			n += v.(*atomic.Int64).Load()
		}
		return true
	})
	return n
}
