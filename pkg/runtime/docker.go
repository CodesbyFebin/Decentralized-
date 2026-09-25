package runtime

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Docker runs digest-pinned container images through the docker CLI with
// enforced memory and CPU limits. The daemon verifies the manifest digest
// when pulling by digest; the host additionally checks RepoDigests.
type Docker struct {
	bin string
}

// NewDocker returns the docker runtime.
func NewDocker() *Docker { return &Docker{bin: "docker"} }

func (d *Docker) Name() string { return "docker" }

func (d *Docker) Limits() string {
	return "enforced by the container runtime: --memory and --cpus (cgroups inside the Docker VM/host)"
}

func (d *Docker) run(timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, d.bin, args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return strings.TrimSpace(out.String()), fmt.Errorf("docker %s: %s", args[0], msg)
	}
	return strings.TrimSpace(out.String()), nil
}

func (d *Docker) Available() (bool, string) {
	v, err := d.run(5*time.Second, "version", "--format", "{{.Server.Version}}")
	if err != nil || v == "" {
		return false, "docker daemon not reachable"
	}
	return true, "docker " + v
}

func containerName(s Spec) string {
	safe := strings.NewReplacer("/", "-", "@", "-", ":", "-").Replace(s.Assignment)
	return fmt.Sprintf("dh-%s-g%d-%d", safe, s.Generation, time.Now().UnixNano()%1_000_000)
}

func (d *Docker) Start(s Spec) (Instance, error) {
	if !strings.Contains(s.Image, "@sha256:") {
		return Instance{}, fmt.Errorf("docker runtime: image %q is not digest-pinned", s.Image)
	}
	if _, err := d.run(10*time.Minute, "pull", "--quiet", s.Image); err != nil {
		return Instance{}, err
	}
	args := []string{"run", "-d", "--name", containerName(s),
		"--label", "dh.assignment=" + s.Assignment, "--label", "dh.generation=" + fmt.Sprint(s.Generation), "--label", "dh.node=" + s.Node,
		"--restart", "no", "--security-opt", "no-new-privileges", "--cap-drop", "ALL", "--pids-limit", "512"}
	if s.MemBytes > 0 {
		args = append(args, "--memory", fmt.Sprint(s.MemBytes), "--memory-swap", fmt.Sprint(s.MemBytes))
	}
	if s.CPUMilli > 0 {
		args = append(args, "--cpus", strconv.FormatFloat(float64(s.CPUMilli)/1000, 'f', 3, 64))
	}
	env := map[string]string{"DH_ASSIGNMENT": s.Assignment, "DH_APP": s.App, "DH_NODE": s.Node, "DH_GENERATION": fmt.Sprint(s.Generation), "DH_REPLICA": fmt.Sprint(s.Replica)}
	for k, v := range s.Env {
		env[k] = v
	}
	if s.WantPort && s.ContainerPort > 0 {
		env["PORT"] = fmt.Sprint(s.ContainerPort)
		args = append(args, "-p", fmt.Sprintf("127.0.0.1::%d", s.ContainerPort))
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "-e", k+"="+env[k])
	}
	for _, m := range s.Mounts {
		if err := os.MkdirAll(m.HostPath, 0o755); err != nil {
			return Instance{}, err
		}
		args = append(args, "-v", m.HostPath+":"+m.Path)
	}
	args = append(args, s.Image)
	args = append(args, s.Args...)
	id, err := d.run(2*time.Minute, args...)
	if err != nil {
		return Instance{}, err
	}
	inst := Instance{Runtime: "docker", ContainerID: id, StartedAt: time.Now().UnixMilli(), StartToken: id}
	digests, _ := d.run(10*time.Second, "inspect", "--format", "{{json .Image}}", id)
	_ = digests
	if repo, err := d.run(10*time.Second, "image", "inspect", "--format", "{{join .RepoDigests \" \"}}", s.Image); err == nil {
		want := s.Image[strings.LastIndex(s.Image, "@")+1:]
		if !strings.Contains(repo, want) {
			_ = d.Stop(inst, time.Second)
			return Instance{}, fmt.Errorf("docker runtime: pulled image does not carry digest %s (got %s)", want, repo)
		}
	}
	if s.WantPort && s.ContainerPort > 0 {
		out, err := d.run(10*time.Second, "port", id, fmt.Sprintf("%d/tcp", s.ContainerPort))
		if err != nil {
			_ = d.Stop(inst, time.Second)
			return Instance{}, err
		}
		for _, line := range strings.Split(out, "\n") {
			if _, p, err := net.SplitHostPort(strings.TrimSpace(line)); err == nil {
				inst.Port, _ = strconv.ParseInt(p, 10, 64)
				break
			}
		}
	}
	if pid, err := d.run(10*time.Second, "inspect", "--format", "{{.State.Pid}}", id); err == nil {
		inst.PID, _ = strconv.ParseInt(pid, 10, 64)
	}
	return inst, nil
}

func (d *Docker) Status(i Instance) Status {
	out, err := d.run(10*time.Second, "inspect", "--format", "{{.State.Status}} {{.State.ExitCode}} {{.State.OOMKilled}}", i.ContainerID)
	if err != nil {
		if strings.Contains(err.Error(), "No such") {
			return Status{State: "exited", ExitCode: -1, Detail: "container no longer exists"}
		}
		return Status{State: "unknown", Detail: err.Error()}
	}
	f := strings.Fields(out)
	if len(f) < 3 {
		return Status{State: "unknown", Detail: out}
	}
	code, _ := strconv.ParseInt(f[1], 10, 64)
	switch {
	case f[2] == "true":
		return Status{State: "oom-killed", ExitCode: code, Detail: "container exceeded its memory limit and was killed (OOMKilled=true)"}
	case f[0] == "running":
		return Status{State: "running"}
	default:
		return Status{State: "exited", ExitCode: code, Detail: "container " + f[0]}
	}
}

func (d *Docker) Stop(i Instance, grace time.Duration) error {
	secs := int(grace.Seconds())
	if secs < 1 {
		secs = 1
	}
	_, _ = d.run(grace+10*time.Second, "stop", "-t", fmt.Sprint(secs), i.ContainerID)
	_, err := d.run(20*time.Second, "rm", "-f", i.ContainerID)
	if err != nil && strings.Contains(err.Error(), "No such") {
		return nil
	}
	return err
}

func (d *Docker) Exec(i Instance, argv []string, timeout time.Duration) (int, string, error) {
	args := append([]string{"exec", i.ContainerID}, argv...)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, d.bin, args...)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		code = -1
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
	}
	if len(out) > 64<<10 {
		out = out[len(out)-64<<10:]
	}
	return code, string(out), nil
}

// Logs returns the container's recent output.
func (d *Docker) Logs(i Instance, tail int) string {
	out, _ := d.run(10*time.Second, "logs", "--tail", fmt.Sprint(tail), i.ContainerID)
	return out
}
