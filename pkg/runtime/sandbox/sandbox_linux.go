//go:build linux

package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// initArg0 is argv[0] of the re-executed agent that becomes the sandbox's
// init: it prepares the namespaces, then execs the workload.
const initArg0 = "dh-sandbox-init"

// insideUID is the unprivileged uid a workload runs as inside the user
// namespace when a subordinate uid range is available. Without one it runs
// as namespace-root (uid 0 mapped to the agent's own host uid) with every
// capability dropped, which is still confined by the namespaces, seccomp,
// read-only root and cgroups.
const insideUID = 1000

// Engine starts and supervises sandboxes.
type Engine struct {
	// StateDir holds per-instance mount points. It is made world-traversable
	// (0711) because the sandbox init runs as an unprivileged host uid.
	StateDir string
	// AllowedRoots are the host directories mounts may come from.
	AllowedRoots []string

	mu     sync.Mutex
	exits  map[int]int64
	relays map[string]*relay
}

// NewEngine returns an engine using stateDir.
func NewEngine(stateDir string, allowedRoots []string) *Engine {
	return &Engine{StateDir: stateDir, AllowedRoots: allowedRoots, exits: map[int]int64{}, relays: map[string]*relay{}}
}

// Handle identifies a started sandbox well enough to re-adopt it.
type Handle struct {
	ID         string  `json:"id"`
	Profile    Profile `json:"profile"`
	PID        int     `json:"pid"`
	StartToken string  `json:"startToken"`
	HostPort   int64   `json:"hostPort"` // relay listener (RESTRICTED) or the workload's own port (PRIVATE)
	InnerPort  int64   `json:"innerPort"`
	HostUID    int     `json:"hostUid"`
}

// Status is an observation of a sandbox.
type Status struct {
	State    string // running | exited | oom-killed
	ExitCode int64
	Detail   string
}

// Available reports whether this host can create sandboxes, and why not.
func (e *Engine) Available() (bool, string) {
	if os.Geteuid() != 0 {
		return false, "the sandbox needs dh-noded to run as root (to map an unprivileged uid and manage cgroups)"
	}
	if b, err := os.ReadFile("/proc/sys/user/max_user_namespaces"); err == nil && strings.TrimSpace(string(b)) == "0" {
		return false, "user namespaces are disabled (user.max_user_namespaces = 0)"
	}
	if goruntime.GOARCH != archName {
		return false, "no seccomp filter for " + goruntime.GOARCH
	}
	// Probe for actual namespace support via dry-run unshare.
	namespaceTests := []string{
		"user",  // must work (used for mapping)
		"pid",   // must work (isolation)
		"mount", // must work (isolation)
		"net",   // required for RESTRICTED
	}
	for _, ns := range namespaceTests {
		cmd := exec.Command("unshare", "--"+ns, "true")
		if err := cmd.Run(); err != nil {
			return false, fmt.Sprintf("%s namespace not supported by kernel: %v", ns, err)
		}
	}
	// Check rootless support (subuid availability).
	if _, err := os.Stat("/etc/subuid"); err != nil {
		return false, "no /etc/subuid range configured; workloads will run as namespace-root (less ideal)"
	}
	v := "cgroup v1"
	if cgroupV2() {
		v = "cgroup v2"
	}
	return true, "user/mount/pid/ipc/uts/net namespaces, seccomp, no capabilities, read-only root, " + v + " limits, rootless support"
}

// identity is how one workload is mapped into and out of its user namespace.
type identity struct {
	uidMap  []syscall.SysProcIDMap
	gidMap  []syscall.SysProcIDMap
	runUID  int // uid the workload execs as, inside the namespace
	hostUID int // host uid that runUID maps to (writable mounts are chowned to it)
}

// subRange returns a subordinate uid range for name from /etc/subuid.
func subRange(name string) (base, count int, ok bool) {
	b, err := os.ReadFile("/etc/subuid")
	if err != nil {
		return 0, 0, false
	}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Split(line, ":")
		if len(f) != 3 || f[0] != name {
			continue
		}
		base, _ = strconv.Atoi(f[1])
		count, _ = strconv.Atoi(f[2])
		if base > 0 && count >= 2 {
			return base, count, true
		}
	}
	return 0, 0, false
}

func userName(uid int) string {
	b, err := os.ReadFile("/etc/passwd")
	if err == nil {
		want := ":" + strconv.Itoa(uid) + ":"
		for _, line := range strings.Split(string(b), "\n") {
			if f := strings.Split(line, ":"); len(f) >= 3 && ":"+f[2]+":" == want {
				return f[0]
			}
		}
	}
	if uid == 0 {
		return "root"
	}
	return ""
}

// identityFor picks the mapping for owner. With a subordinate range the
// workload runs unprivileged (uid 1000) mapped to a per-owner host uid; the
// agent's own uid maps to namespace-root for setup. Without one it runs as
// namespace-root mapped to the agent's uid, and drops all capabilities.
func identityFor(owner string) identity {
	euid := os.Geteuid()
	if base, count, ok := subRange(userName(euid)); ok {
		h := fnv.New32a()
		_, _ = h.Write([]byte(owner))
		host := base + int(h.Sum32()%uint32(count))
		m := func() []syscall.SysProcIDMap {
			return []syscall.SysProcIDMap{{ContainerID: 0, HostID: euid, Size: 1}, {ContainerID: insideUID, HostID: host, Size: 1}}
		}
		return identity{uidMap: m(), gidMap: m(), runUID: insideUID, hostUID: host}
	}
	m := func() []syscall.SysProcIDMap { return []syscall.SysProcIDMap{{ContainerID: 0, HostID: euid, Size: 1}} }
	return identity{uidMap: m(), gidMap: m(), runUID: 0, hostUID: euid}
}

// initConfig is what the parent hands the sandbox init on fd 3.
type initConfig struct {
	Config
	Root      string   `json:"root"`
	NewNet    bool     `json:"newNet"`
	Env       []string `json:"envList"`
	Effective Limits   `json:"effective"`
	RunUID    int      `json:"runUid"`
}

// Start creates the namespaces and cgroups and execs the workload.
func (e *Engine) Start(cfg Config, owner string) (Handle, error) {
	if err := cfg.Validate(e.AllowedRoots); err != nil {
		return Handle{}, err
	}
	if ok, why := e.Available(); !ok {
		return Handle{}, fmt.Errorf("sandbox unavailable: %s", why)
	}
	if owner == "" {
		owner = cfg.ID
	}
	lim := cfg.Effective()
	idn := identityFor(owner)
	h := Handle{ID: cfg.ID, Profile: cfg.Profile, HostUID: idn.hostUID}

	if err := os.MkdirAll(e.StateDir, 0o711); err != nil {
		return h, err
	}
	_ = os.Chmod(e.StateDir, 0o711)
	root := filepath.Join(e.StateDir, cfg.ID)
	if err := os.Mkdir(root, 0o755); err != nil {
		return h, fmt.Errorf("sandbox root: %w", err)
	}
	cleanup := func() { _ = os.Remove(root) }

	if _, err := os.Stat(cfg.Executable); err != nil {
		cleanup()
		return h, fmt.Errorf("sandbox executable: %w", err)
	}
	for _, m := range cfg.Mounts {
		if !m.ReadOnly {
			if err := chownTree(m.HostPath, idn.hostUID); err != nil {
				cleanup()
				return h, err
			}
		}
	}

	var port, inner int64
	env := map[string]string{"PATH": "/usr/local/bin:/usr/bin:/bin", "HOME": WorkDir, "TMPDIR": WorkDir}
	for k, v := range cfg.Env {
		env[k] = v
	}
	newNet := cfg.Profile == ProfileRestricted
	if cfg.Port > 0 {
		inner = cfg.Port
		if !newNet {
			// PRIVATE shares the host network: the workload binds the port itself.
			port = cfg.Port
		}
		env["PORT"] = strconv.FormatInt(inner, 10)
		env["DH_LISTEN"] = fmt.Sprintf("127.0.0.1:%d", inner)
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var envList []string
	for _, k := range keys {
		envList = append(envList, k+"="+env[k])
	}

	cg, err := newCgroups(cfg.ID, lim)
	if err != nil {
		cleanup()
		return h, err
	}

	cfgR, cfgW, err := os.Pipe()
	if err != nil {
		cg.remove()
		cleanup()
		return h, err
	}
	syncR, syncW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	// fd 3: config, fd 4: go signal, fd 5: error channel (closes on exec)
	extra := []*os.File{cfgR, syncR, errW}
	ic := initConfig{Config: cfg, Root: root, NewNet: newNet, Env: envList, Effective: lim, RunUID: idn.runUID}

	self, err := os.Executable()
	if err != nil {
		cg.remove()
		cleanup()
		return h, err
	}
	var logf *os.File
	if cfg.LogPath != "" {
		if logf, err = os.OpenFile(cfg.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err != nil {
			cg.remove()
			cleanup()
			return h, err
		}
		defer logf.Close()
		fmt.Fprintf(logf, "--- dh-noded: sandbox %s profile %s at %s ---\n", cfg.ID, cfg.Profile, time.Now().UTC().Format(time.RFC3339))
	}
	cmd := &exec.Cmd{Path: self, Args: []string{initArg0}, Env: []string{}, Stdout: logf, Stderr: logf, ExtraFiles: extra}
	if logf == nil {
		cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	}
	flags := uintptr(unix.CLONE_NEWUSER | unix.CLONE_NEWNS | unix.CLONE_NEWPID | unix.CLONE_NEWIPC | unix.CLONE_NEWUTS)
	if newNet {
		flags |= unix.CLONE_NEWNET
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:                 flags,
		UidMappings:                idn.uidMap,
		GidMappings:                idn.gidMap,
		GidMappingsEnableSetgroups: false,
		Setpgid:                    true,
	}
	if err := cmd.Start(); err != nil {
		cg.remove()
		cleanup()
		return h, fmt.Errorf("sandbox: %w", err)
	}
	cfgR.Close()
	syncR.Close()
	errW.Close()
	pid := cmd.Process.Pid
	fail := func(err error) (Handle, error) {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		cg.remove()
		cleanup()
		return h, err
	}
	// Into the cgroups before the workload can run a single instruction.
	if err := cg.add(pid); err != nil {
		cfgW.Close()
		syncW.Close()
		return fail(err)
	}
	if err := json.NewEncoder(cfgW).Encode(ic); err != nil {
		cfgW.Close()
		syncW.Close()
		return fail(err)
	}
	cfgW.Close()
	_, _ = syncW.Write([]byte{1})
	syncW.Close()
	msg, _ := io.ReadAll(errR) // EOF at exec (close-on-exec) or init exit
	errR.Close()
	if len(msg) > 0 {
		return fail(fmt.Errorf("sandbox init: %s", strings.TrimSpace(string(msg))))
	}
	h.PID = pid
	h.StartToken = startToken(pid)
	e.mu.Lock()
	e.exits[pid] = -999
	e.mu.Unlock()
	go func() {
		err := cmd.Wait()
		code := int64(0)
		if err != nil {
			code = -1
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				code = int64(ee.ExitCode())
				if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
					code = -int64(ws.Signal())
				}
			}
		}
		e.mu.Lock()
		e.exits[pid] = code
		e.mu.Unlock()
	}()
	if newNet && inner > 0 {
		r, err := e.ensureRelay(h, inner, 0)
		if err != nil {
			return fail(err)
		}
		port = r.port
	}
	h.HostPort, h.InnerPort = port, inner
	return h, nil
}

func chownTree(dir string, uid int) error {
	return filepath.Walk(dir, func(p string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Lchown(p, uid, uid)
	})
}

// Status observes a sandbox. After an agent restart it re-establishes the
// port relay for a RESTRICTED workload that is still running.
func (e *Engine) Status(h Handle) Status {
	e.mu.Lock()
	code, known := e.exits[h.PID]
	e.mu.Unlock()
	if known && code != -999 {
		return e.exited(h, code)
	}
	if h.PID <= 0 || syscall.Kill(h.PID, 0) != nil || (h.StartToken != "" && startToken(h.PID) != h.StartToken) {
		return e.exited(h, -1)
	}
	if h.Profile == ProfileRestricted && h.InnerPort > 0 {
		if _, err := e.ensureRelay(h, h.InnerPort, h.HostPort); err != nil {
			return Status{State: "running", Detail: "relay: " + err.Error()}
		}
	}
	return Status{State: "running"}
}

func (e *Engine) exited(h Handle, code int64) Status {
	if n := cgroupsFor(h.ID).oomKills(); n > 0 {
		return Status{State: "oom-killed", ExitCode: code, Detail: fmt.Sprintf("killed by the kernel: memory limit reached (%d OOM kill(s) in the sandbox cgroup)", n)}
	}
	return Status{State: "exited", ExitCode: code, Detail: fmt.Sprintf("exit code %d", code)}
}

// Stop signals the workload (SIGTERM, then SIGKILL) and removes its cgroups
// and mount point. The workload is PID 1 of its namespace, so it only sees
// SIGTERM if it handles it; SIGKILL always applies.
func (e *Engine) Stop(h Handle, grace time.Duration) error {
	if e.Status(h).State == "running" {
		_ = syscall.Kill(h.PID, syscall.SIGTERM)
		deadline := time.Now().Add(grace)
		for time.Now().Before(deadline) && e.Status(h).State == "running" {
			time.Sleep(50 * time.Millisecond)
		}
		if e.Status(h).State == "running" {
			_ = syscall.Kill(h.PID, syscall.SIGKILL)
			for i := 0; i < 100 && e.Status(h).State == "running"; i++ {
				time.Sleep(20 * time.Millisecond)
			}
		}
	}
	e.Release(h)
	return nil
}

// Release removes what a finished sandbox left on the host.
func (e *Engine) Release(h Handle) {
	e.mu.Lock()
	if r := e.relays[h.ID]; r != nil {
		r.close()
		delete(e.relays, h.ID)
	}
	e.mu.Unlock()
	cg := cgroupsFor(h.ID)
	for i := 0; i < 50; i++ {
		if cg.remove() == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = os.Remove(filepath.Join(e.StateDir, h.ID))
}

// startToken is field 22 (starttime) of /proc/<pid>/stat.
func startToken(pid int) string {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return ""
	}
	s := string(b)
	i := strings.LastIndexByte(s, ')')
	if i < 0 {
		return ""
	}
	f := strings.Fields(s[i+1:])
	if len(f) < 20 {
		return ""
	}
	return f[19]
}
