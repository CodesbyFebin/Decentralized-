// Package devcluster runs a real multi-process Decentralized.Host cluster on
// one machine: Raft control-plane members, host agents in distinct failure
// domains talking over userspace WireGuard, and edge hosts. It backs
// `dh dev up`, the integration tests and the chaos suite. Nothing here is
// simulated: every component is the same binary an operator would deploy.
package devcluster

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"decentralized.host/pkg/cli"
	"decentralized.host/pkg/control"
	"decentralized.host/pkg/node"
)

// Options configures a cluster.
type Options struct {
	Dir       string
	Bin       string // directory containing dh-control, dh-noded, dh-beacon
	Cluster   string
	CPs       int
	Hosts     int
	Edges     int
	Base      int
	Chaos     bool
	LostAfter string
	Postgres  string
	HostArgs  func(i int, name string) []string `json:"-"` // extra dh-noded flags
	EdgeArgs  []string
	EdgeExtra func(edgeIndex int) []string `json:"-"` // per-edge extra flags
	EdgeHTTP  map[int]int                  // edge index -> fixed HTTP port (ACME HTTP-01 uses 5002 with Pebble)
	Policy    func(name string) string     `json:"-"` // optional policy.yaml content per host
	// MeshAdvertise lets the chaos suite put a fault proxy in front of a
	// host's WireGuard socket: peers dial the returned address instead.
	MeshAdvertise func(i int, name, listen string) string `json:"-"`
	Quiet         bool
	// TLS runs every member API over TLS (root-issued member certificates;
	// bootstrap pins each member's throwaway certificate by fingerprint).
	TLS bool
}

// EdgeAddr lists an edge host's listeners.
type EdgeAddr struct {
	Name  string `json:"name"`
	HTTP  string `json:"http"`
	HTTPS string `json:"https"`
}

// Proc is one managed process.
type Proc struct {
	Name string   `json:"name"`
	Kind string   `json:"kind"` // control | host
	PID  int      `json:"pid"`
	Bin  string   `json:"bin"`
	Args []string `json:"args"`
	Log  string   `json:"log"`
	Data string   `json:"data"`
	API  string   `json:"api,omitempty"`
	Mesh string   `json:"mesh,omitempty"`
	cmd  *exec.Cmd
	done chan struct{}
}

// Cluster is a running cluster.
type Cluster struct {
	Opt     Options       `json:"opt"`
	Home    string        `json:"home"`
	Procs   []*Proc       `json:"procs"`
	Edge    string        `json:"edge"`
	EdgeTLS string        `json:"edgeTls"`
	Edges   []EdgeAddr    `json:"edges"`
	Op      *cli.Operator `json:"-"`
	mu      sync.Mutex
}

func (o *Options) defaults() {
	if o.Cluster == "" {
		o.Cluster = "dev"
	}
	if o.CPs == 0 {
		o.CPs = 3
	}
	if o.Base == 0 {
		o.Base = 17700
	}
	if o.LostAfter == "" {
		o.LostAfter = "15s"
	}
	if o.Bin == "" {
		exe, _ := os.Executable()
		o.Bin = filepath.Dir(exe)
	}
}

func (c *Cluster) logf(format string, args ...any) {
	if !c.Opt.Quiet {
		fmt.Printf(format, args...)
	}
}

// Up starts a cluster and waits until every host is ready and meshed.
func Up(opt Options) (*Cluster, error) {
	opt.defaults()
	abs, err := filepath.Abs(opt.Dir)
	if err != nil {
		return nil, err
	}
	opt.Dir = abs
	if _, err := os.Stat(filepath.Join(abs, "cluster.json")); err == nil {
		return nil, fmt.Errorf("%s already holds a cluster", abs)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, err
	}
	for _, b := range []string{"dh-control", "dh-noded"} {
		if _, err := os.Stat(filepath.Join(opt.Bin, b)); err != nil {
			return nil, fmt.Errorf("%s not found in %s (build with `make build`)", b, opt.Bin)
		}
	}
	c := &Cluster{Opt: opt, Home: filepath.Join(abs, "operator")}
	if opt.Quiet {
		cli.Out = io.Discard
	}
	op, err := cli.Init(c.Home, opt.Cluster, opt.TLS)
	if err != nil {
		return nil, err
	}
	c.Op = op
	c.logf("• starting %d control-plane member(s)\n", opt.CPs)
	for i := 1; i <= opt.CPs; i++ {
		if _, err := c.StartMember(i); err != nil {
			return c, err
		}
	}
	for i := 1; i <= opt.CPs; i++ {
		p := c.Proc(fmt.Sprintf("cp-%d", i))
		if err := waitHealth(c.scheme(), p.API, 20*time.Second); err != nil {
			return c, err
		}
		code, err := waitFile(filepath.Join(p.Data, "bootstrap.code"), 10*time.Second)
		if err != nil {
			return c, err
		}
		fp := ""
		if opt.TLS {
			if fp, err = waitFile(filepath.Join(p.Data, "bootstrap.fingerprint"), 10*time.Second); err != nil {
				return c, err
			}
		}
		if i == 1 {
			err = c.Op.Bootstrap(p.API, code, fp)
		} else {
			err = c.Op.AddMember(p.API, code, fp)
		}
		if err != nil {
			return c, fmt.Errorf("%s: %w", p.Name, err)
		}
	}
	total := opt.Hosts + opt.Edges
	c.logf("• starting %d host(s) (%d edge)\n", total, opt.Edges)
	for i := 0; i < total; i++ {
		if _, err := c.StartHost(i, i >= opt.Hosts); err != nil {
			return c, err
		}
	}
	if err := c.WaitFor(90*time.Second, "all hosts ready and meshed", func(v *control.View) bool {
		ready := 0
		for _, n := range v.Nodes {
			if n.Status == "ready" && n.LastObs.Freshness == "FRESH" && n.Mesh != nil && n.Mesh.Device != "none" {
				ready++
			}
		}
		return ready == total
	}); err != nil {
		return c, err
	}
	return c, c.save()
}

// HostName returns the conventional name of host i.
func (c *Cluster) HostName(i int, edge bool) string {
	if edge {
		return fmt.Sprintf("edge-%d", i-c.Opt.Hosts+1)
	}
	return fmt.Sprintf("host-%c", 'a'+i)
}

// StartMember starts control-plane member i (1-based).
func (c *Cluster) StartMember(i int) (*Proc, error) {
	name := fmt.Sprintf("cp-%d", i)
	data := filepath.Join(c.Opt.Dir, name)
	api := fmt.Sprintf("127.0.0.1:%d", c.Opt.Base+i)
	args := []string{"--data", data, "--api", api, "--raft", fmt.Sprintf("127.0.0.1:%d", c.Opt.Base+100+i),
		"--mesh", fmt.Sprintf("127.0.0.1:%d", c.Opt.Base+200+i), "--fast-raft", "--lost-after", c.Opt.LostAfter}
	if c.Opt.Postgres != "" {
		args = append(args, "--postgres", c.Opt.Postgres)
	}
	if c.Opt.TLS {
		args = append(args, "--tls")
	}
	p := &Proc{Name: name, Kind: "control", Bin: filepath.Join(c.Opt.Bin, "dh-control"), Args: args, Data: data, API: api}
	return p, c.launch(p)
}

// StartHost starts host i.
func (c *Cluster) StartHost(i int, edge bool) (*Proc, error) {
	name := c.HostName(i, edge)
	regions := []string{"cell-a", "cell-b", "cell-c", "cell-d", "cell-e", "cell-f"}
	var roles []string
	if edge {
		roles = []string{"edge"}
	}
	tok, err := c.Op.Invite(cli.InviteOpts{Roles: roles, Auto: true, Note: "dev " + name})
	if err != nil {
		return nil, err
	}
	data := filepath.Join(c.Opt.Dir, name)
	_ = os.MkdirAll(data, 0o700)
	if c.Opt.Policy != nil {
		if pol := c.Opt.Policy(name); pol != "" {
			_ = os.WriteFile(filepath.Join(data, "policy.yaml"), []byte(pol), 0o600)
		}
	}
	tokFile := filepath.Join(c.Opt.Dir, name+".join")
	if err := os.WriteFile(tokFile, []byte(tok+"\n"), 0o600); err != nil {
		return nil, err
	}
	mesh := fmt.Sprintf("127.0.0.1:%d", c.Opt.Base+300+i)
	adv := mesh
	if c.Opt.MeshAdvertise != nil {
		adv = c.Opt.MeshAdvertise(i, name, mesh)
	}
	args := []string{"--data", data, "--join-file", tokFile, "--name", name, "--region", regions[i%len(regions)], "--host", name,
		"--mesh", mesh, "--mesh-advertise", adv, "--status", "127.0.0.1:0", "--mem", "2Gi"}
	if edge {
		ei := i - c.Opt.Hosts
		args = append(args, "--roles", "edge")
		httpAddr := fmt.Sprintf("127.0.0.1:%d", c.Opt.Base+400+i)
		if p, ok := c.Opt.EdgeHTTP[ei]; ok {
			httpAddr = fmt.Sprintf("127.0.0.1:%d", p)
		}
		tlsAddr := fmt.Sprintf("127.0.0.1:%d", c.Opt.Base+450+i)
		if c.Edge == "" {
			c.Edge, c.EdgeTLS = httpAddr, tlsAddr
		}
		c.Edges = append(c.Edges, EdgeAddr{Name: name, HTTP: httpAddr, HTTPS: tlsAddr})
		args = append(args, "--edge-http", httpAddr, "--edge-https", tlsAddr)
		args = append(args, c.Opt.EdgeArgs...)
		if c.Opt.EdgeExtra != nil {
			args = append(args, c.Opt.EdgeExtra(ei)...)
		}
	}
	if c.Opt.Chaos {
		args = append(args, "--chaos")
	}
	if c.Opt.HostArgs != nil {
		args = append(args, c.Opt.HostArgs(i, name)...)
	}
	p := &Proc{Name: name, Kind: "host", Bin: filepath.Join(c.Opt.Bin, "dh-noded"), Args: args, Data: data, Mesh: mesh}
	return p, c.launch(p)
}

func (c *Cluster) launch(p *Proc) error {
	p.Log = filepath.Join(c.Opt.Dir, "logs", p.Name+".log")
	_ = os.MkdirAll(filepath.Dir(p.Log), 0o700)
	logf, err := os.OpenFile(p.Log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer logf.Close()
	cmd := exec.Command(p.Bin, p.Args...)
	cmd.Stdout, cmd.Stderr = logf, logf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	p.cmd, p.PID, p.done = cmd, cmd.Process.Pid, make(chan struct{})
	go func(done chan struct{}) { _ = cmd.Wait(); close(done) }(p.done)
	c.mu.Lock()
	replaced := false
	for i, q := range c.Procs {
		if q.Name == p.Name {
			c.Procs[i] = p
			replaced = true
		}
	}
	if !replaced {
		c.Procs = append(c.Procs, p)
	}
	c.mu.Unlock()
	return c.save()
}

// Proc returns a process by name.
func (c *Cluster) Proc(name string) *Proc {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, p := range c.Procs {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// Kill sends sig to a process (SIGKILL simulates a crash). Workloads the
// host started are separate process groups and are not affected.
func (c *Cluster) Kill(name string, sig syscall.Signal) error {
	p := c.Proc(name)
	if p == nil {
		return fmt.Errorf("no process %s", name)
	}
	if err := syscall.Kill(p.PID, sig); err != nil {
		return err
	}
	if p.done != nil {
		select {
		case <-p.done:
		case <-time.After(10 * time.Second):
			return fmt.Errorf("%s did not exit", name)
		}
	} else {
		for i := 0; i < 100 && syscall.Kill(p.PID, 0) == nil; i++ {
			time.Sleep(100 * time.Millisecond)
		}
	}
	return nil
}

// KillMachine simulates losing a whole host: the agent and every workload
// it started are killed with SIGKILL, and nothing is cleaned up.
func (c *Cluster) KillMachine(name string) (int, error) {
	p := c.Proc(name)
	if p == nil {
		return 0, fmt.Errorf("no host %s", name)
	}
	var st node.State
	if b, err := os.ReadFile(filepath.Join(p.Data, "state.json")); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	if err := c.Kill(name, syscall.SIGKILL); err != nil {
		return 0, err
	}
	killed := 0
	for _, ad := range st.Admitted {
		if ad.Inst.PID > 0 && ad.Runtime == "process" && syscall.Kill(-int(ad.Inst.PID), syscall.SIGKILL) == nil {
			killed++
		}
		if ad.Inst.ContainerID != "" {
			_ = exec.Command("docker", "kill", ad.Inst.ContainerID).Run()
			killed++
		}
	}
	return killed, nil
}

// Restart starts a stopped process again with the same arguments.
func (c *Cluster) Restart(name string) error {
	p := c.Proc(name)
	if p == nil {
		return fmt.Errorf("no process %s", name)
	}
	if syscall.Kill(p.PID, 0) == nil {
		return fmt.Errorf("%s is still running", name)
	}
	np := &Proc{Name: p.Name, Kind: p.Kind, Bin: p.Bin, Args: p.Args, Data: p.Data, API: p.API, Mesh: p.Mesh}
	return c.launch(np)
}

// Alive reports whether a process is running.
func (c *Cluster) Alive(name string) bool {
	p := c.Proc(name)
	return p != nil && syscall.Kill(p.PID, 0) == nil
}

func (c *Cluster) save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("cluster state: %w", err)
	}
	p := filepath.Join(c.Opt.Dir, "cluster.json")
	if err := os.WriteFile(p+".tmp", b, 0o600); err != nil {
		return err
	}
	return os.Rename(p+".tmp", p)
}

// Load reopens a cluster started earlier (for dev down / status).
func Load(dir string) (*Cluster, error) {
	b, err := os.ReadFile(filepath.Join(dir, "cluster.json"))
	if err != nil {
		return nil, err
	}
	var c Cluster
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	op, err := cli.Load(c.Home, c.Opt.Cluster)
	if err == nil {
		c.Op = op
	}
	return &c, nil
}

// Down stops every process and the workloads hosts started.
func (c *Cluster) Down() (int, int) {
	c.mu.Lock()
	procs := append([]*Proc(nil), c.Procs...)
	c.mu.Unlock()
	for _, p := range procs {
		_ = syscall.Kill(-p.PID, syscall.SIGTERM)
	}
	time.Sleep(700 * time.Millisecond)
	for _, p := range procs {
		_ = syscall.Kill(-p.PID, syscall.SIGKILL)
	}
	stopped := 0
	for _, p := range procs {
		if p.Kind != "host" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(p.Data, "state.json"))
		if err != nil {
			continue
		}
		var st node.State
		if json.Unmarshal(b, &st) != nil {
			continue
		}
		for _, ad := range st.Admitted {
			if ad.Inst.PID > 0 && ad.Runtime == "process" && syscall.Kill(-int(ad.Inst.PID), syscall.SIGKILL) == nil {
				stopped++
			}
			if ad.Inst.ContainerID != "" {
				_ = exec.Command("docker", "rm", "-f", ad.Inst.ContainerID).Run()
				stopped++
			}
		}
	}
	return len(procs), stopped
}

// View fetches the console projection.
func (c *Cluster) View() (*control.View, error) { return c.Op.View() }

// WaitFor polls the view until cond holds.
func (c *Cluster) WaitFor(timeout time.Duration, what string, cond func(v *control.View) bool) error {
	deadline := time.Now().Add(timeout)
	var last error
	for {
		v, err := c.View()
		if err == nil && cond(v) {
			return nil
		}
		if err != nil {
			last = err
		}
		if time.Now().After(deadline) {
			if last != nil {
				return fmt.Errorf("timed out after %s waiting for %s (last error: %v)", timeout, what, last)
			}
			return fmt.Errorf("timed out after %s waiting for %s", timeout, what)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// HostLedgerTail returns the last n entries of a host's own journal as
// text lines (diagnostics for failing tests).
func (c *Cluster) HostLedgerTail(name string, n int) []string {
	p := c.Proc(name)
	if p == nil {
		return nil
	}
	addr, err := os.ReadFile(filepath.Join(p.Data, "status.addr"))
	if err != nil {
		return []string{err.Error()}
	}
	resp, err := http.Get("http://" + strings.TrimSpace(string(addr)) + "/ledger")
	if err != nil {
		return []string{err.Error()}
	}
	defer resp.Body.Close()
	var out struct {
		Entries []struct {
			Seq    int64  `json:"seq"`
			Action string `json:"action"`
			Detail string `json:"detail"`
		} `json:"entries"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Entries) > n {
		out.Entries = out.Entries[len(out.Entries)-n:]
	}
	var lines []string
	for _, e := range out.Entries {
		lines = append(lines, fmt.Sprintf("#%d %s: %s", e.Seq, e.Action, e.Detail))
	}
	return lines
}

// HostStatus reads a host agent's local status API.
func (c *Cluster) HostStatus(name string) (*node.Status, error) {
	p := c.Proc(name)
	if p == nil {
		return nil, fmt.Errorf("no host %s", name)
	}
	addr, err := os.ReadFile(filepath.Join(p.Data, "status.addr"))
	if err != nil {
		return nil, err
	}
	resp, err := http.Get("http://" + strings.TrimSpace(string(addr)) + "/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var s node.Status
	return &s, json.NewDecoder(resp.Body).Decode(&s)
}

// Chaos injects a fault through a host's local status API (needs Options.Chaos).
func (c *Cluster) Chaos(name string, req node.ChaosRequest) (map[string]any, error) {
	p := c.Proc(name)
	if p == nil {
		return nil, fmt.Errorf("no host %s", name)
	}
	addr, err := os.ReadFile(filepath.Join(p.Data, "status.addr"))
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(req)
	resp, err := http.Post("http://"+strings.TrimSpace(string(addr))+"/chaos", "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, errors.New(strings.TrimSpace(string(data)))
	}
	var out map[string]any
	_ = json.Unmarshal(data, &out)
	return out, nil
}

// scheme is how the harness reaches member APIs.
func (c *Cluster) scheme() string {
	if c.Opt.TLS {
		return "https"
	}
	return "http"
}

// Leader returns the name of the current raft leader process.
func (c *Cluster) Leader() (string, error) {
	for _, p := range c.Procs {
		if p.Kind != "control" || !c.Alive(p.Name) {
			continue
		}
		var h map[string]any
		if getJSON(c.scheme()+"://"+p.API+"/api/v1/health", &h) != nil {
			continue
		}
		if h["state"] == "leader" {
			return p.Name, nil
		}
	}
	return "", errors.New("no leader")
}

// probeClient reads /api/v1/health, which carries no secrets, from members
// that may still be serving their throwaway bootstrap certificate.
var probeClient = &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

func getJSON(url string, out any) error {
	resp, err := probeClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func waitHealth(scheme, addr string, d time.Duration) error {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		var h map[string]any
		if getJSON(scheme+"://"+addr+"/api/v1/health", &h) == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s", addr)
}

func waitFile(path string, d time.Duration) (string, error) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(b))) > 0 {
			return strings.TrimSpace(string(b)), nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return "", fmt.Errorf("timed out waiting for %s", path)
}

// EdgeGet performs a request through the edge (plain HTTP, no redirect).
func (c *Cluster) EdgeGet(host, path string) (*http.Response, []byte, error) {
	req, _ := http.NewRequest("GET", "http://"+c.Edge+path, nil)
	req.Host = host
	req.Header.Set("X-DH-No-Redirect", "1")
	cl := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := cl.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, b, nil
}

// StartHostWithToken starts one more host that joins with the given token
// (for example an invite without auto-approval) instead of a fresh
// auto-approved invite. Slot picks the port block and region, as in StartHost.
func (c *Cluster) StartHostWithToken(slot int, name, token string, extra ...string) (*Proc, error) {
	data := filepath.Join(c.Opt.Dir, name)
	if err := os.MkdirAll(data, 0o700); err != nil {
		return nil, err
	}
	tokFile := filepath.Join(c.Opt.Dir, name+".join")
	if err := os.WriteFile(tokFile, []byte(token+"\n"), 0o600); err != nil {
		return nil, err
	}
	regions := []string{"cell-a", "cell-b", "cell-c", "cell-d", "cell-e", "cell-f"}
	mesh := fmt.Sprintf("127.0.0.1:%d", c.Opt.Base+300+slot)
	args := append([]string{"--data", data, "--join-file", tokFile, "--name", name, "--region", regions[slot%len(regions)], "--host", name,
		"--mesh", mesh, "--mesh-advertise", mesh, "--status", "127.0.0.1:0"}, extra...)
	if c.Opt.Chaos {
		args = append(args, "--chaos")
	}
	p := &Proc{Name: name, Kind: "host", Bin: filepath.Join(c.Opt.Bin, "dh-noded"), Args: args, Data: data, Mesh: mesh}
	return p, c.launch(p)
}
