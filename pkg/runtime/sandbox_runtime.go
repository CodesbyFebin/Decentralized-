package runtime

import (
	"encoding/json"
	"fmt"
	"time"

	"decentralized.host/pkg/runtime/sandbox"
)

// Sandboxed runs verified artifacts inside kernel isolation (see
// pkg/runtime/sandbox). It is selected when an assignment sets an isolation
// profile. Its Instance carries the sandbox Handle in ContainerID.
type Sandboxed struct {
	eng *sandbox.Engine
}

// NewSandboxed returns the sandbox runtime. stateDir holds per-instance mount
// points; allowedRoots are the host directories a workload may mount from
// (the agent's work and volume trees).
func NewSandboxed(stateDir string, allowedRoots []string) *Sandboxed {
	return &Sandboxed{eng: sandbox.NewEngine(stateDir, allowedRoots)}
}

func (s *Sandboxed) Name() string { return "sandbox" }

func (s *Sandboxed) Available() (bool, string) {
	ok, why := s.eng.Available()
	if !ok {
		return false, why
	}
	return true, why
}

func (s *Sandboxed) Limits() string {
	return "enforced: user/mount/pid/ipc/uts namespaces (network too for RESTRICTED), seccomp, no capabilities, read-only root, cgroup memory/PID/CPU limits"
}

func (s *Sandboxed) Start(spec Spec) (Instance, error) {
	profile, err := sandbox.ParseProfile(spec.Isolation)
	if err != nil {
		return Instance{}, err
	}
	if spec.Executable == "" {
		return Instance{}, fmt.Errorf("sandbox runtime: no verified artifact for %s", spec.Assignment)
	}
	cfg := sandbox.Config{
		ID:         sandboxID(spec),
		Profile:    profile,
		Executable: spec.Executable,
		Args:       spec.Args,
		Env:        spec.Env,
		Limits:     sandbox.Limits{MemBytes: spec.MemBytes, CPUMilli: spec.CPUMilli},
		LogPath:    spec.LogPath,
	}
	if spec.WantPort {
		cfg.Port = spec.ContainerPort
		if cfg.Port == 0 {
			cfg.Port = 8080
		}
	}
	ephemeralPath := ""
	for _, m := range spec.Mounts {
		cfg.Mounts = append(cfg.Mounts, sandbox.Mount{HostPath: m.HostPath, Path: m.Path, ReadOnly: m.ReadOnly})
	}
	// Ephemeral mounts are read-only secret material
	for _, m := range spec.EphemeralMounts {
		cfg.Mounts = append(cfg.Mounts, sandbox.Mount{HostPath: m.HostPath, Path: m.Path, ReadOnly: true})
		if ephemeralPath == "" {
			ephemeralPath = m.HostPath
		}
	}
	h, err := s.eng.Start(cfg, spec.App)
	if err != nil {
		return Instance{}, err
	}
	blob, _ := json.Marshal(h)
	return Instance{
		Runtime:         "sandbox",
		PID:             int64(h.PID),
		ContainerID:     string(blob),
		Port:            h.HostPort,
		StartedAt:       time.Now().UnixMilli(),
		StartToken:      h.StartToken,
		EphemeralID:     spec.EphemeralID,
		EphemeralPath:   ephemeralPath,
	}, nil
}

func (s *Sandboxed) handle(i Instance) sandbox.Handle {
	var h sandbox.Handle
	if i.ContainerID != "" {
		_ = json.Unmarshal([]byte(i.ContainerID), &h)
	}
	if h.PID == 0 {
		h.PID = int(i.PID)
		h.HostPort = i.Port
		h.StartToken = i.StartToken
	}
	return h
}

func (s *Sandboxed) Status(i Instance) Status {
	st := s.eng.Status(s.handle(i))
	return Status{State: st.State, ExitCode: st.ExitCode, Detail: st.Detail}
}

func (s *Sandboxed) Stop(i Instance, grace time.Duration) error {
	return s.eng.Stop(s.handle(i), grace)
}

func (s *Sandboxed) Exec(Instance, []string, time.Duration) (int, string, error) {
	// Exec into a sandbox would need to enter its namespaces and re-apply the
	// profile; not supported. Logs are the supported way to observe a workload.
	return -1, "", ErrExecUnsupported
}

// sandboxID is a filesystem- and cgroup-safe per-instance id.
func sandboxID(spec Spec) string {
	id := fmt.Sprintf("%s-g%d", spec.Assignment, spec.Generation)
	safe := make([]byte, 0, len(id))
	for i := 0; i < len(id); i++ {
		c := id[i]
		if c == '/' || c == ' ' {
			c = '_'
		}
		safe = append(safe, c)
	}
	return string(safe)
}
