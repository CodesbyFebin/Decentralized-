// Package sandbox runs a workload inside kernel isolation boundaries.
//
// Profiles:
//
//   - PRIVATE: an owner's own workload. New user, mount, PID, IPC and UTS
//     namespaces; host network; unprivileged host uid; read-only root;
//     seccomp; no capabilities; cgroup limits when given.
//   - RESTRICTED: a workload from someone the owner trusts less. PRIVATE
//     plus a private network namespace (loopback only; its one declared port
//     is relayed by the agent) and mandatory memory, PID and CPU limits.
//   - UNTRUSTED: arbitrary marketplace code. Not available: namespaces and
//     seccomp share the host kernel, which is not a hostile-code boundary.
//     It is refused, never silently replaced by a weaker profile.
//
// Nothing here decides whether a workload may run; host admission does.
package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Profile is an isolation profile.
type Profile string

const (
	ProfilePrivate    Profile = "PRIVATE"
	ProfileRestricted Profile = "RESTRICTED"
	ProfileUntrusted  Profile = "UNTRUSTED"
)

// ErrUntrustedUnavailable is returned for the UNTRUSTED profile.
var ErrUntrustedUnavailable = errors.New("sandbox: the UNTRUSTED profile is not available: no qualified hostile-code boundary (microVM or gVisor) is implemented; it is never replaced by a container")

// ParseProfile accepts PRIVATE, RESTRICTED and UNTRUSTED (case-insensitive).
func ParseProfile(s string) (Profile, error) {
	switch p := Profile(strings.ToUpper(strings.TrimSpace(s))); p {
	case ProfilePrivate, ProfileRestricted, ProfileUntrusted:
		return p, nil
	}
	return "", fmt.Errorf("sandbox: unknown isolation profile %q (PRIVATE, RESTRICTED or UNTRUSTED)", s)
}

// Limits are enforced with cgroups. Zero means "profile default".
type Limits struct {
	MemBytes int64 `json:"memBytes"`
	CPUMilli int64 `json:"cpuMilli"`
	PIDs     int64 `json:"pids"`
	TmpBytes int64 `json:"tmpBytes"` // size of the /tmp tmpfs
}

// RESTRICTED defaults when a workload declares nothing.
const (
	DefaultMemBytes = 256 << 20
	DefaultPIDs     = 128
	DefaultTmpBytes = 64 << 20
)

// Mount binds one host directory the agent owns into the sandbox.
type Mount struct {
	HostPath string `json:"hostPath"`
	Path     string `json:"path"`
	ReadOnly bool   `json:"readOnly"`
}

// Config describes one sandboxed workload.
type Config struct {
	ID         string            `json:"id"`         // unique per instance; names the cgroup
	Profile    Profile           `json:"profile"`    //
	Executable string            `json:"executable"` // verified artifact on the host
	Args       []string          `json:"args"`
	Env        map[string]string `json:"env"`
	Limits     Limits            `json:"limits"`
	Mounts     []Mount           `json:"mounts"`
	Port       int64             `json:"port"` // port the workload listens on inside (RESTRICTED); 0 = none
	LogPath    string            `json:"logPath"`
}

// Sandbox-internal paths the workload sees.
const (
	AppDir  = "/app"
	WorkDir = "/tmp"
)

// forbidden are sandbox paths a mount may never cover.
var forbidden = []string{"/", "/proc", "/dev", "/sys", AppDir, "/usr", "/lib", "/lib64", "/bin", "/sbin", "/etc"}

// Validate checks a config against the mount policy: host paths must lie
// under one of allowedRoots, targets must be clean absolute paths outside
// the sandbox's own system directories.
func (c *Config) Validate(allowedRoots []string) error {
	if c.ID == "" || strings.ContainsAny(c.ID, "/\\ \x00") || c.ID == "." || c.ID == ".." {
		return fmt.Errorf("sandbox: invalid id %q", c.ID)
	}
	switch c.Profile {
	case ProfilePrivate, ProfileRestricted:
	case ProfileUntrusted:
		return ErrUntrustedUnavailable
	default:
		return fmt.Errorf("sandbox: unknown profile %q", c.Profile)
	}
	if !filepath.IsAbs(c.Executable) {
		return fmt.Errorf("sandbox: executable must be an absolute host path")
	}
	if c.Limits.MemBytes < 0 || c.Limits.CPUMilli < 0 || c.Limits.PIDs < 0 || c.Limits.TmpBytes < 0 {
		return errors.New("sandbox: negative limit")
	}
	if c.Port < 0 || c.Port > 65535 {
		return errors.New("sandbox: port out of range")
	}
	for _, m := range c.Mounts {
		if err := checkMount(m, allowedRoots); err != nil {
			return err
		}
	}
	for k := range c.Env {
		if k == "" || strings.ContainsAny(k, "=\x00") {
			return fmt.Errorf("sandbox: invalid environment name %q", k)
		}
	}
	return nil
}

func checkMount(m Mount, allowedRoots []string) error {
	if !filepath.IsAbs(m.HostPath) || filepath.Clean(m.HostPath) != m.HostPath {
		return fmt.Errorf("sandbox: mount source %q must be a clean absolute path", m.HostPath)
	}
	// Attempt symlink resolution; if path doesn't exist, check parent recursively.
	// This prevents symlink escapes while allowing paths to be created later.
	realPath := m.HostPath
	if _, err := os.Stat(m.HostPath); err == nil {
		// Path exists; resolve it fully.
		resolved, err := filepath.EvalSymlinks(m.HostPath)
		if err != nil {
			return fmt.Errorf("sandbox: mount source %q cannot be resolved: %w", m.HostPath, err)
		}
		realPath = resolved
	} else {
		// Path doesn't exist; check that we can resolve the parent to prevent traversal.
		parent := filepath.Dir(m.HostPath)
		for parent != "/" && parent != m.HostPath {
			if _, err := os.Stat(parent); err == nil {
				// Found an existing ancestor; resolve it.
				resolved, err := filepath.EvalSymlinks(parent)
				if err != nil {
					return fmt.Errorf("sandbox: mount source %q has an unresolvable parent: %w", m.HostPath, err)
				}
				// Reconstruct the full path from the resolved parent.
				rel := m.HostPath[len(parent):]
				realPath = filepath.Join(resolved, rel)
				break
			}
			parent = filepath.Dir(parent)
		}
	}
	inside := false
	for _, r := range allowedRoots {
		r = filepath.Clean(r)
		realRoot := r
		if _, err := os.Stat(r); err == nil {
			resolved, err := filepath.EvalSymlinks(r)
			if err != nil {
				continue
			}
			realRoot = resolved
		}
		// Check exact match or proper containment (must have a separator boundary).
		if realPath == realRoot {
			inside = true
			break
		}
		if len(realPath) > len(realRoot) && realPath[len(realRoot)] == filepath.Separator && strings.HasPrefix(realPath, realRoot+string(filepath.Separator)) {
			inside = true
			break
		}
	}
	if !inside {
		return fmt.Errorf("sandbox: mount source %q is not under a directory the agent manages", m.HostPath)
	}
	if !filepath.IsAbs(m.Path) || filepath.Clean(m.Path) != m.Path {
		return fmt.Errorf("sandbox: mount target %q must be a clean absolute path", m.Path)
	}
	for _, f := range forbidden {
		if m.Path == f || (f != "/" && strings.HasPrefix(m.Path, f+"/")) {
			return fmt.Errorf("sandbox: mount target %q would cover %s", m.Path, f)
		}
	}
	return nil
}

// Effective returns the limits a profile applies: RESTRICTED always has
// memory, PID and /tmp limits.
func (c *Config) Effective() Limits {
	l := c.Limits
	if c.Profile == ProfileRestricted {
		if l.MemBytes == 0 {
			l.MemBytes = DefaultMemBytes
		}
		if l.PIDs == 0 {
			l.PIDs = DefaultPIDs
		}
	}
	if l.TmpBytes == 0 {
		l.TmpBytes = DefaultTmpBytes
	}
	return l
}
