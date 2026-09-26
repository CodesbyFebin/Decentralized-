// Package runtime executes admitted workloads. The host admission layer
// always sits between an assignment and a runtime: nothing here decides
// whether something may run, only how.
package runtime

import (
	"errors"
	"time"
)

// Spec is what an admitted assignment asks a runtime to start.
type Spec struct {
	Assignment    string
	App           string
	Node          string
	Generation    int64
	Replica       int64
	Executable    string // process: absolute path of the verified artifact
	Image         string // docker: ref@sha256:...
	Isolation     string // "" (bare process) | PRIVATE | RESTRICTED (sandbox)
	Args          []string
	Env           map[string]string
	CPUMilli      int64
	MemBytes      int64
	WantPort      bool
	ContainerPort int64
	WorkDir       string
	Mounts        []Mount
	LogPath       string
}

// Mount attaches a host directory.
type Mount struct {
	Name     string
	HostPath string
	Path     string
}

// Instance identifies a started workload well enough to re-adopt it after
// the host agent restarts.
type Instance struct {
	Runtime     string `json:"runtime"`
	PID         int64  `json:"pid"`
	ContainerID string `json:"containerId"`
	Port        int64  `json:"port"`
	StartedAt   int64  `json:"startedAt"`
	StartToken  string `json:"startToken"` // kernel start time of the pid, or container id
}

// Status is a runtime observation.
type Status struct {
	State    string // running | exited | oom-killed | unknown
	ExitCode int64
	Detail   string
}

// Runtime is one execution backend.
type Runtime interface {
	Name() string
	Available() (bool, string)
	Start(Spec) (Instance, error)
	Status(Instance) Status
	Stop(Instance, time.Duration) error
	Exec(Instance, []string, time.Duration) (int, string, error)
	// Limits describes whether resource limits are enforced (shown verbatim).
	Limits() string
}

// ErrExecUnsupported is returned by runtimes without exec.
var ErrExecUnsupported = errors.New("exec is not supported by this runtime")
