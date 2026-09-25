package runtime

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	goruntime "runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Process runs a verified artifact as a direct child process: no shell, a
// minimal environment, its own process group (so it survives a host agent
// restart and can be re-adopted by pid + kernel start time).
type Process struct {
	mu    sync.Mutex
	exits map[int64]int64 // pid -> exit code, for children this agent started
	waits map[int64]bool
}

// NewProcess returns the subprocess runtime.
func NewProcess() *Process { return &Process{exits: map[int64]int64{}, waits: map[int64]bool{}} }

func (p *Process) Name() string              { return "process" }
func (p *Process) Available() (bool, string) { return true, "direct exec of BLAKE3-verified artifacts" }
func (p *Process) Limits() string {
	return "not enforced: the process runtime does not apply cgroup or rlimit memory/CPU limits on " + goruntime.GOOS + "; use the docker runtime for enforced limits"
}

func freePort() (int64, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return int64(l.Addr().(*net.TCPAddr).Port), nil
}

func (p *Process) Start(s Spec) (Instance, error) {
	if s.Executable == "" {
		return Instance{}, errors.New("process runtime: no executable")
	}
	env := map[string]string{
		"PATH": "/usr/bin:/bin", "HOME": s.WorkDir, "TMPDIR": s.WorkDir,
		"DH_ASSIGNMENT": s.Assignment, "DH_APP": s.App, "DH_NODE": s.Node,
		"DH_GENERATION": fmt.Sprint(s.Generation), "DH_REPLICA": fmt.Sprint(s.Replica),
	}
	for k, v := range s.Env {
		env[k] = v
	}
	var port int64
	if s.WantPort {
		var err error
		if port, err = freePort(); err != nil {
			return Instance{}, err
		}
		env["PORT"] = fmt.Sprint(port)
		env["DH_LISTEN"] = fmt.Sprintf("127.0.0.1:%d", port)
	}
	for _, m := range s.Mounts {
		env["DH_VOLUME_"+strings.ToUpper(strings.ReplaceAll(m.Name, "-", "_"))] = m.HostPath
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
	if err := os.MkdirAll(s.WorkDir, 0o700); err != nil {
		return Instance{}, err
	}
	logf, err := os.OpenFile(s.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return Instance{}, err
	}
	defer logf.Close()
	fmt.Fprintf(logf, "--- dh-noded: start %s generation %d at %s ---\n", s.Assignment, s.Generation, time.Now().UTC().Format(time.RFC3339))
	cmd := exec.Command(s.Executable, s.Args...)
	cmd.Env = envList
	cmd.Dir = s.WorkDir
	cmd.Stdout, cmd.Stderr, cmd.Stdin = logf, logf, nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return Instance{}, fmt.Errorf("process runtime: %w", err)
	}
	pid := int64(cmd.Process.Pid)
	p.mu.Lock()
	p.waits[pid] = true
	p.mu.Unlock()
	go func() {
		err := cmd.Wait()
		code := int64(0)
		if err != nil {
			code = -1
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				code = int64(ee.ExitCode())
			}
		}
		p.mu.Lock()
		p.exits[pid] = code
		delete(p.waits, pid)
		p.mu.Unlock()
	}()
	inst := Instance{Runtime: "process", PID: pid, Port: port, StartedAt: time.Now().UnixMilli(), StartToken: ProcStartToken(int(pid))}
	// Give the process a moment to fail fast (bad binary, port conflict).
	time.Sleep(150 * time.Millisecond)
	if st := p.Status(inst); st.State != "running" {
		return inst, fmt.Errorf("process exited immediately (code %d); see %s", st.ExitCode, s.LogPath)
	}
	return inst, nil
}

func (p *Process) Status(i Instance) Status {
	p.mu.Lock()
	code, exited := p.exits[i.PID]
	p.mu.Unlock()
	if exited {
		return Status{State: "exited", ExitCode: code, Detail: fmt.Sprintf("exit code %d", code)}
	}
	if i.PID <= 0 || syscall.Kill(int(i.PID), 0) != nil {
		return Status{State: "exited", ExitCode: -1, Detail: "process no longer exists"}
	}
	if i.StartToken != "" {
		if tok := ProcStartToken(int(i.PID)); tok != i.StartToken {
			return Status{State: "exited", ExitCode: -1, Detail: "pid reused by another process (start time differs)"}
		}
	}
	return Status{State: "running"}
}

func (p *Process) Stop(i Instance, grace time.Duration) error {
	if p.Status(i).State != "running" {
		return nil
	}
	_ = syscall.Kill(-int(i.PID), syscall.SIGTERM)
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if p.Status(i).State != "running" {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = syscall.Kill(-int(i.PID), syscall.SIGKILL)
	return nil
}

func (p *Process) Exec(Instance, []string, time.Duration) (int, string, error) {
	return -1, "", ErrExecUnsupported
}
