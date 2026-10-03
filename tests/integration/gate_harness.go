package integration

import (
	"fmt"
	"log"
	"testing"
	"time"
)

// TestHarness provides consistent test reporting for mainnet launch gates
type TestHarness struct {
	Name         string
	ChecksPassed int
	ChecksFailed int
	Errors       []string
	StartTime    time.Time
	EndTime      time.Time
}

// NewTestHarness creates a new test harness
func NewTestHarness(name string) *TestHarness {
	return &TestHarness{
		Name:   name,
		Errors: make([]string, 0),
	}
}

// Start marks the beginning of a test
func (h *TestHarness) Start() {
	h.StartTime = time.Now()
	log.Printf("[%s] Test started at %v", h.Name, h.StartTime)
}

// ReportPass records a passing check
func (h *TestHarness) ReportPass(checkName, details string) {
	h.ChecksPassed++
	log.Printf("[%s] ✓ PASS: %s - %s", h.Name, checkName, details)
}

// ReportFail records a failing check
func (h *TestHarness) ReportFail(checkName, details string) {
	h.ChecksFailed++
	h.Errors = append(h.Errors, fmt.Sprintf("%s: %s", checkName, details))
	log.Printf("[%s] ✗ FAIL: %s - %s", h.Name, checkName, details)
}

// Finalize completes the test and returns true if all checks passed
func (h *TestHarness) Finalize(t *testing.T) bool {
	h.EndTime = time.Now()
	duration := h.EndTime.Sub(h.StartTime).Seconds()

	t.Logf(`
        [%s] Test Results:
          Duration: %.0f seconds
          Passed: %d
          Failed: %d
        `, h.Name, duration, h.ChecksPassed, h.ChecksFailed)

	if len(h.Errors) > 0 {
		t.Logf(`          Errors:`)
		for _, err := range h.Errors {
			t.Logf(`            - %s`, err)
		}
	}

	return h.ChecksFailed == 0
}
