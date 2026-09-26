//go:build !linux

package sandbox

import (
	"errors"
	"time"
)

// MaybeInit is a no-op off Linux.
func MaybeInit() {}

// Engine is unavailable off Linux.
type Engine struct {
	StateDir     string
	AllowedRoots []string
}

func NewEngine(stateDir string, allowedRoots []string) *Engine {
	return &Engine{StateDir: stateDir, AllowedRoots: allowedRoots}
}

type Handle struct {
	ID         string  `json:"id"`
	Profile    Profile `json:"profile"`
	PID        int     `json:"pid"`
	StartToken string  `json:"startToken"`
	HostPort   int64   `json:"hostPort"`
	InnerPort  int64   `json:"innerPort"`
	HostUID    int     `json:"hostUid"`
}

type Status struct {
	State    string
	ExitCode int64
	Detail   string
}

var errUnsupported = errors.New("sandbox: requires Linux namespaces, seccomp and cgroups")

func (e *Engine) Available() (bool, string)            { return false, errUnsupported.Error() }
func (e *Engine) Start(Config, string) (Handle, error) { return Handle{}, errUnsupported }
func (e *Engine) Status(Handle) Status {
	return Status{State: "exited", ExitCode: -1, Detail: errUnsupported.Error()}
}
func (e *Engine) Stop(Handle, time.Duration) error { return nil }
func (e *Engine) Release(Handle)                   {}
