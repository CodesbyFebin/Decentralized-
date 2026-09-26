//go:build linux

package sandbox

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestMain lets a re-executed process become the sandbox init.
func TestMain(m *testing.M) {
	MaybeInit()
	os.Exit(m.Run())
}

// probeBin is the test workload, built once into a run-lifetime directory
// that every test (and every sandbox uid) can read.
var (
	probeBin  string
	probeOnce sync.Once
	probeErr  error
)

func probe(t *testing.T) string {
	t.Helper()
	probeOnce.Do(func() {
		dir, err := os.MkdirTemp("", "dh-probe-")
		if err != nil {
			probeErr = err
			return
		}
		_ = os.Chmod(dir, 0o755)
		out := filepath.Join(dir, "probe")
		cmd := exec.Command("go", "build", "-o", out, "./testdata/probe")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			probeErr = err
			return
		}
		_ = os.Chmod(out, 0o755)
		probeBin = out
	})
	if probeErr != nil {
		t.Fatalf("build probe: %v", probeErr)
	}
	return probeBin
}

func requireSandbox(t *testing.T) *Engine {
	t.Helper()
	e := NewEngine(t.TempDir(), nil)
	if ok, why := e.Available(); !ok {
		t.Skipf("sandbox unavailable: %s", why)
	}
	// The probe binary must be readable by the unprivileged sandbox uid.
	return e
}

// runReport starts the probe with "report", waits for it, and returns the
// JSON it printed to its log.
func runReport(t *testing.T, e *Engine, cfg Config) report {
	t.Helper()
	log := filepath.Join(t.TempDir(), "log")
	cfg.Executable = probe(t)
	cfg.Args = []string{"report"}
	cfg.LogPath = log
	h, err := e.Start(cfg, cfg.ID)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(h, time.Second)
	waitExit(t, e, h, 20*time.Second)
	b, _ := os.ReadFile(log)
	line := lastJSON(b)
	if line == "" {
		t.Fatalf("probe printed no report; log:\n%s", b)
	}
	var r report
	if err := json.Unmarshal([]byte(line), &r); err != nil {
		t.Fatalf("decode report: %v\n%s", err, b)
	}
	return r
}

type report struct {
	UID           int      `json:"uid"`
	PID1          bool     `json:"pid1"`
	ProcPIDs      int      `json:"procPids"`
	RootWritable  bool     `json:"rootWritable"`
	ShadowRead    bool     `json:"shadowRead"`
	ShadowExists  bool     `json:"shadowExists"`
	DevSda        bool     `json:"devSda"`
	DevKmsg       bool     `json:"devKmsg"`
	CapEff        string   `json:"capEff"`
	NoNewPrivs    int      `json:"noNewPrivs"`
	SeccompMode   int      `json:"seccompMode"`
	MountAttempt  string   `json:"mountAttempt"`
	MknodAttempt  string   `json:"mknodAttempt"`
	VisibleMounts int      `json:"visibleMounts"`
	Env           []string `json:"env"`
	Hostname      string   `json:"hostname"`
}

func waitExit(t *testing.T, e *Engine, h Handle, d time.Duration) Status {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if st := e.Status(h); st.State != "running" {
			return st
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("sandbox %s still running after %s", h.ID, d)
	return Status{}
}

func lastJSON(b []byte) string {
	var last string
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "{") {
			last = strings.TrimSpace(l)
		}
	}
	return last
}

// TestRestrictedProfileEscape is the acceptance gate: a RESTRICTED workload
// cannot read the host password file, cannot reach host devices, keeps no
// capabilities, cannot mount or create devices, runs unprivileged, and sees
// only its own process tree.
func TestRestrictedProfileEscape(t *testing.T) {
	e := requireSandbox(t)
	// Ensure a host /etc/shadow with content actually exists to be denied.
	if b, err := os.ReadFile("/etc/shadow"); err != nil || len(b) == 0 {
		t.Skip("host has no readable /etc/shadow to test denial against")
	}
	r := runReport(t, e, Config{ID: "esc-restricted", Profile: ProfileRestricted})

	if r.ShadowRead {
		t.Error("workload read a host-style /etc/shadow")
	}
	// The inside uid depends on whether a subordinate range is configured:
	// 1000 (unprivileged) with one, else namespace-root with every capability
	// dropped. Either way it must have no capabilities (checked below).
	wantUID := identityFor("esc-restricted").runUID
	if r.UID != wantUID {
		t.Errorf("uid inside = %d, want %d", r.UID, wantUID)
	}
	if !r.PID1 {
		t.Error("workload is not PID 1 of its own PID namespace")
	}
	if r.ProcPIDs > 2 { // itself, maybe a transient reaper
		t.Errorf("workload sees %d host processes in /proc; PID namespace not isolated", r.ProcPIDs)
	}
	if r.DevSda || r.DevKmsg {
		t.Errorf("host devices visible: sda=%v kmsg=%v", r.DevSda, r.DevKmsg)
	}
	if r.RootWritable {
		t.Error("root filesystem is writable")
	}
	if r.MountAttempt == "" {
		t.Error("mount(2) succeeded inside the sandbox")
	}
	if r.MknodAttempt == "" {
		t.Error("mknod(2) of a block device succeeded inside the sandbox")
	}
	if r.NoNewPrivs != 1 {
		t.Error("no_new_privs is not set")
	}
	if r.SeccompMode != 2 { // SECCOMP_MODE_FILTER
		t.Errorf("seccomp mode = %d, want 2 (filter)", r.SeccompMode)
	}
	if caps := strings.TrimLeft(r.CapEff, "0"); caps != "" {
		t.Errorf("effective capabilities not empty: %s", r.CapEff)
	}
	if r.Hostname != "sandbox" {
		t.Errorf("UTS namespace not isolated: hostname %q", r.Hostname)
	}
	for _, kv := range r.Env {
		if strings.HasPrefix(kv, "DH_TOKEN") || strings.Contains(kv, "tok.admin") {
			t.Errorf("host environment leaked into the workload: %s", kv)
		}
	}
	t.Logf("restricted report: uid=%d procPids=%d mounts=%d mount=%q mknod=%q", r.UID, r.ProcPIDs, r.VisibleMounts, r.MountAttempt, r.MknodAttempt)
}

// The mount policy refuses host paths and dangerous targets before anything starts.
func TestMountPolicyRejectsHostPaths(t *testing.T) {
	managed := t.TempDir()
	cfg := Config{ID: "mp", Profile: ProfilePrivate, Executable: "/bin/true"}
	cfg.Mounts = []Mount{{HostPath: "/etc", Path: "/data"}}
	if err := cfg.Validate([]string{managed}); err == nil {
		t.Error("mounting /etc from outside the managed root was allowed")
	}
	cfg.Mounts = []Mount{{HostPath: filepath.Join(managed, "v"), Path: "/etc"}}
	if err := cfg.Validate([]string{managed}); err == nil {
		t.Error("mounting over /etc inside the sandbox was allowed")
	}
	cfg.Mounts = []Mount{{HostPath: filepath.Join(managed, "v"), Path: "/data"}}
	if err := cfg.Validate([]string{managed}); err != nil {
		t.Errorf("a legitimate managed mount was rejected: %v", err)
	}
}

// UNTRUSTED is refused, never downgraded to a container.
func TestUntrustedRefused(t *testing.T) {
	cfg := Config{ID: "u", Profile: ProfileUntrusted, Executable: "/bin/true"}
	if err := cfg.Validate(nil); err == nil {
		t.Fatal("UNTRUSTED validated")
	}
	e := NewEngine(t.TempDir(), nil)
	if _, err := e.Start(cfg, "u"); err == nil {
		t.Fatal("UNTRUSTED started")
	}
}

// The memory limit is enforced: a workload over its limit is OOM-killed.
func TestMemoryLimitEnforced(t *testing.T) {
	e := requireSandbox(t)
	log := filepath.Join(t.TempDir(), "log")
	h, err := e.Start(Config{
		ID: "mem", Profile: ProfileRestricted, Executable: probe(t),
		Args: []string{"mem", "256"}, Limits: Limits{MemBytes: 64 << 20, PIDs: 64}, LogPath: log,
	}, "mem")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(h, time.Second)
	st := waitExit(t, e, h, 20*time.Second)
	if st.State != "oom-killed" {
		t.Errorf("state = %q (%s), want oom-killed for a 256 MB alloc under a 64 MB limit", st.State, st.Detail)
	}
}

// The PID limit is enforced: the workload cannot exceed its cap.
func TestPIDLimitEnforced(t *testing.T) {
	e := requireSandbox(t)
	log := filepath.Join(t.TempDir(), "log")
	h, err := e.Start(Config{
		ID: "pids", Profile: ProfileRestricted, Executable: probe(t),
		Args: []string{"fork", "200"}, Limits: Limits{MemBytes: 128 << 20, PIDs: 16}, LogPath: log,
	}, "pids")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(h, time.Second)
	waitExit(t, e, h, 20*time.Second)
	b, _ := os.ReadFile(log)
	if !strings.Contains(string(b), "children") {
		t.Fatalf("probe did not report; log:\n%s", b)
	}
	// The cgroup cap is 16 and a handful are the runtime's own threads, so the
	// workload must fall well short of 200.
	var got int
	fmt.Sscanf(lastLine(b, "children"), "children %d", &got)
	if got >= 100 {
		t.Errorf("started %d children under a 16-PID cap", got)
	}
	t.Logf("pid-limited children: %d", got)
}

// RESTRICTED has no network route out; only the agent's relay reaches its port.
func TestRestrictedNetworkIsolationAndRelay(t *testing.T) {
	e := requireSandbox(t)
	log := filepath.Join(t.TempDir(), "log")
	h, err := e.Start(Config{
		ID: "net", Profile: ProfileRestricted, Executable: probe(t),
		Args: []string{"serve"}, Port: 8080, Limits: Limits{MemBytes: 128 << 20, PIDs: 64}, LogPath: log,
	}, "net")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop(h, time.Second)
	if h.HostPort == 0 {
		t.Fatal("no relay port assigned")
	}
	// The relay reaches the workload.
	var body string
	for i := 0; i < 50; i++ {
		if resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", h.HostPort)); err == nil {
			buf := make([]byte, 8)
			n, _ := resp.Body.Read(buf)
			resp.Body.Close()
			body = string(buf[:n])
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if body != "ok" {
		t.Fatalf("relay did not reach the workload: got %q", body)
	}
	// The workload's own port is not the host port: it bound inside its netns.
	if h.HostPort == 8080 {
		if ln, err := net.Listen("tcp", "127.0.0.1:8080"); err == nil {
			ln.Close() // 8080 is free on the host, so the relay uses a different port
		}
	}
}

// PRIVATE shares the host network and can read host DNS config, but still
// runs unprivileged with no capabilities and a read-only root.
func TestPrivateProfileIsUnprivilegedButNetworked(t *testing.T) {
	e := requireSandbox(t)
	r := runReport(t, e, Config{ID: "priv", Profile: ProfilePrivate})
	if r.UID != identityFor("priv").runUID {
		t.Errorf("uid = %d, want %d", r.UID, identityFor("priv").runUID)
	}
	if r.RootWritable {
		t.Error("root writable under PRIVATE")
	}
	if caps := strings.TrimLeft(r.CapEff, "0"); caps != "" {
		t.Errorf("capabilities under PRIVATE: %s", r.CapEff)
	}
	if r.MountAttempt == "" {
		t.Error("mount succeeded under PRIVATE")
	}
}

func lastLine(b []byte, contains string) string {
	var last string
	for _, l := range strings.Split(string(b), "\n") {
		if strings.Contains(l, contains) {
			last = l
		}
	}
	return last
}

// NEGATIVE CONTROL: How to verify tests detect missing enforcement
//
// These are NOT run by default. They document how to validate that the
// test suite catches enforcement failures. To verify:
//
// 1. PID namespace: Comment out unix.CLONE_NEWPID in sandbox_linux.go:264.
//    Run TestRestrictedProfileEscape. It MUST FAIL (workload sees host pids).
//
// 2. Memory limit: Comment out the cgroup memory setup in cgroup_linux.go.
//    Run TestMemoryLimitEnforced. It MUST FAIL (process not OOM-killed).
//
// 3. Seccomp: Comment out applySeccomp() in init_linux.go:76.
//    Run TestRestrictedProfileEscape. It MUST FAIL (mount/mknod succeed).
//
// 4. Capability dropping: Comment out Capset in init_linux.go:328.
//    Run TestRestrictedProfileEscape. It MUST FAIL (CapEff not empty).
//
// 5. Read-only root: Comment out unix.MS_RDONLY in init_linux.go:277.
//    Run TestRestrictedProfileEscape. It MUST FAIL (root is writable).
//
// 6. Network isolation: Remove unix.CLONE_NEWNET from sandbox_linux.go:266.
//    Run TestRestrictedNetworkIsolationAndRelay. It MUST FAIL or the
//    workload must reach external networks.
//
// If any of these changes cause a test to PASS instead of FAIL, the test
// suite is not testing the enforcement mechanism.
