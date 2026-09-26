//go:build linux

package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// RuntimeEvidence documents the qualification of kernel isolation enforcement.
//
// This record is generated after all hostile-fixture tests pass.
// It proves that the host can enforce the specified isolation profile
// with the required security properties (namespaces, seccomp, capabilities, cgroups).
type RuntimeEvidence struct {
	// Metadata
	EvidenceID   string    `json:"evidenceId"`   // Unique identifier (e.g., "RUNTIME-P0-A01-<timestamp>")
	ObservedAt   time.Time `json:"observedAt"`   // When the tests ran
	NodeHostname string    `json:"nodeHostname"` // Host where tests ran

	// Environment
	Kernel       string `json:"kernel"`       // uname -sr
	Architecture string `json:"architecture"` // amd64, arm64, etc.
	Glibc        string `json:"glibc"`        // libc version

	// Enforcement capabilities discovered
	RuntimeCapabilities RuntimeCapabilities `json:"runtimeCapabilities"` // What this host can enforce

	// Test Results
	TestCases map[string]TestResult `json:"testCases"` // Test name -> result

	// Qualification
	OverallOutcome string `json:"overallOutcome"` // PASS or FAIL
	Reason         string `json:"reason"`         // Why it passed or failed
}

// RuntimeCapabilities describe what isolation mechanisms the host can enforce.
type RuntimeCapabilities struct {
	UserNamespace    bool `json:"userNamespace"`
	PIDNamespace     bool `json:"pidNamespace"`
	MountNamespace   bool `json:"mountNamespace"`
	NetworkNamespace bool `json:"networkNamespace"`
	IPCNamespace     bool `json:"ipcNamespace"`
	UTSNamespace     bool `json:"utsNamespace"`
	CgroupsV2        bool `json:"cgroupsV2"`
	Seccomp          bool `json:"seccomp"`
	ReadOnlyRoot     bool `json:"readOnlyRoot"`
	CapabilityDrop   bool `json:"capabilityDrop"`
	PortRelay        bool `json:"portRelay"` // For RESTRICTED network isolation
}

// TestResult records the outcome of one test case.
type TestResult struct {
	Name     string `json:"name"`
	Outcome  string `json:"outcome"`  // PASS or FAIL
	Duration int64  `json:"duration"` // milliseconds
	Detail   string `json:"detail"`   // Error message if failed
}

// NewRuntimeEvidence creates an evidence record for the current host.
// It documents the environment and will be populated with test results.
func NewRuntimeEvidence() (*RuntimeEvidence, error) {
	hostname, _ := os.Hostname()

	ev := &RuntimeEvidence{
		EvidenceID:   fmt.Sprintf("RUNTIME-P0-A01-%d", time.Now().UnixNano()),
		ObservedAt:   time.Now().UTC(),
		NodeHostname: hostname,
		TestCases:    make(map[string]TestResult),
	}

	// Capture kernel and architecture info
	if kernel, err := os.ReadFile("/proc/version"); err == nil {
		ev.Kernel = string(kernel)
	}
	ev.Architecture = archName // Set at compile time in sandbox_linux.go

	// Capture libc version if available
	if libc, err := os.ReadFile("/lib/x86_64-linux-gnu/libc.so.6"); err == nil && len(libc) > 0 {
		// Note: A proper version check would use ldd, but this is a placeholder
		ev.Glibc = "detected"
	}

	return ev, nil
}

// Seal completes the evidence record by checking all mandatory test results.
// It returns true only if all required tests passed.
func (e *RuntimeEvidence) Seal() bool {
	mandatoryTests := map[string]bool{
		"escape":               false,
		"memory_limit":         false,
		"pid_limit":            false,
		"network_isolation":    false,
		"filesystem_isolation": false,
		"readonly_root":        false,
		"capability_dropping":  false,
		"seccomp_enforcement":  false,
		"mount_policy":         false,
		"untrusted_refused":    false,
		"cleanup_enforcement":  false,
	}

	passed := 0
	for name, required := range mandatoryTests {
		if result, ok := e.TestCases[name]; ok {
			if result.Outcome == "PASS" {
				passed++
			} else if required {
				e.OverallOutcome = "FAIL"
				e.Reason = fmt.Sprintf("mandatory test %q failed: %s", name, result.Detail)
				return false
			}
		}
	}

	if passed < len(mandatoryTests) {
		missing := 0
		for name := range mandatoryTests {
			if _, ok := e.TestCases[name]; !ok {
				missing++
			}
		}
		e.OverallOutcome = "FAIL"
		e.Reason = fmt.Sprintf("missing %d mandatory tests", missing)
		return false
	}

	e.OverallOutcome = "PASS"
	e.Reason = "all mandatory tests passed; kernel isolation enforcement verified"
	return true
}

// Write marshals the evidence to JSON and writes it to a file.
func (e *RuntimeEvidence) Write(path string) error {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("RUNTIME-P0-A01 evidence sealed: %s\n", path)
	return nil
}
