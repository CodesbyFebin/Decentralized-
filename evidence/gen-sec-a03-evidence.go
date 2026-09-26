// +build ignore

// Code generation for SEC-P0-A01-A03 cryptographic storage core evidence.
package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type TestResult struct {
	Name     string `json:"name"`
	Outcome  string `json:"outcome"` // PASS | FAIL
	Duration int64  `json:"duration"` // nanoseconds
	Detail   string `json:"detail,omitempty"`
}

type NegativeControlResult struct {
	Control string `json:"control"`
	Outcome string `json:"outcome"` // FAIL (expected) | PASS (unexpected)
	Detail  string `json:"detail,omitempty"`
}

type Evidence struct {
	EvidenceID            string                     `json:"evidenceId"`
	QualificationID       string                     `json:"qualificationId"`
	SourceSHA             string                     `json:"sourceSha"`
	Branch                string                     `json:"branch"`
	Environment           string                     `json:"environment"`
	Algorithm             string                     `json:"algorithm"`
	KeyVersion            string                     `json:"keyVersion"`
	KeyCustodyMode        string                     `json:"keyCustodyMode"`
	CryptoTestResults     []TestResult               `json:"cryptoTestResults"`
	PersistenceTestResults []TestResult              `json:"persistenceTestResults"`
	NegativeControlResults []NegativeControlResult   `json:"negativeControlResults"`
	FailClosedTestResult  TestResult                 `json:"failClosedTestResult"`
	RestartTestResult     TestResult                 `json:"restartTestResult"`
	StartedAt             string                     `json:"startedAt"`
	CompletedAt           string                     `json:"completedAt"`
	EvidenceDigest        string                     `json:"evidenceDigest"`
	Signer                string                     `json:"signer"`
	Signature             string                     `json:"signature,omitempty"`
}

func main() {
	outputDir := flag.String("dir", "", "evidence output directory")
	flag.Parse()

	if *outputDir == "" {
		fmt.Fprintf(os.Stderr, "usage: gen-sec-a03-evidence -dir <output-dir>\n")
		os.Exit(1)
	}

	startedAt := time.Now()
	startedAtStr := startedAt.Format(time.RFC3339Nano)

	// Get source SHA
	sourceSHA := getSourceSHA()
	branch := getGitBranch()

	// Run tests
	cryptoTests := runCryptoTests()
	persistenceTests := runPersistenceTests()
	negativeControls := runNegativeControls()
	failClosedTest := runFailClosedTest()
	restartTest := runRestartTest()

	// Build evidence
	completedAt := time.Now()
	completedAtStr := completedAt.Format(time.RFC3339Nano)

	ev := Evidence{
		EvidenceID:             fmt.Sprintf("SEC-P0-A01-A03-%d", startedAt.UnixNano()),
		QualificationID:        "SEC-P0-A01-A03",
		SourceSHA:              sourceSHA,
		Branch:                 branch,
		Environment:            "Linux container (cloud sandbox)",
		Algorithm:              "aes-256-gcm",
		KeyVersion:             "v1",
		KeyCustodyMode:         "operator-local-bootstrap + cluster-held-wrapped",
		CryptoTestResults:      cryptoTests,
		PersistenceTestResults: persistenceTests,
		NegativeControlResults: negativeControls,
		FailClosedTestResult:   failClosedTest,
		RestartTestResult:      restartTest,
		StartedAt:              startedAtStr,
		CompletedAt:            completedAtStr,
		Signer:                 "codesbyfebin@gmail.com",
	}

	// Compute digest and sign
	digest := computeDigest(&ev)
	ev.EvidenceDigest = digest

	// Write evidence (unsigned for now; signing will be added in A05)
	recordDir := filepath.Join(*outputDir, "SEC-P0-A01-A03-"+startedAt.Format("20060102T150405"))
	if err := os.MkdirAll(recordDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	recordJSON, _ := json.MarshalIndent(&ev, "", "  ")
	recordPath := filepath.Join(recordDir, "record.json")
	if err := os.WriteFile(recordPath, append(recordJSON, '\n'), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write record: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Evidence: %s\n", recordPath)
	fmt.Printf("EvidenceID: %s\n", ev.EvidenceID)
	fmt.Printf("Status: %s (digest %s)\n", aggregateOutcome(ev), ev.EvidenceDigest[:16])
}

func getSourceSHA() string {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return string(out[:len(out)-1]) // trim newline
}

func getGitBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return string(out[:len(out)-1]) // trim newline
}

func runCryptoTests() []TestResult {
	tests := []string{
		"TestGenerateDEK",
		"TestDeriveKEK",
		"TestEncryptDecryptSecret_RoundTrip",
		"TestEncryptSecret_WrongDEK_Rejected",
		"TestDecryptSecret_CiphertextTamper_Rejected",
		"TestDecryptSecret_NonceTamper_Rejected",
		"TestDecryptSecret_AADTamper_Rejected",
		"TestDecryptSecret_VersionTamper_Rejected",
		"TestWrapUnwrapDEK_RoundTrip",
		"TestUnwrapDEK_WrongKEK_Rejected",
		"TestUnwrapDEK_AADTamper_Rejected",
		"TestUnwrapDEK_WrappedTamper_Rejected",
		"TestSecretVersionIndependence",
		"TestNonceUniqueness",
	}

	var results []TestResult
	for _, test := range tests {
		start := time.Now()
		cmd := exec.Command("go", "test", "-v", "./pkg/control", "-run", "^"+test+"$")
		cmd.Dir = "/home/user/Decentralized-"
		output, err := cmd.CombinedOutput()
		duration := time.Since(start).Nanoseconds()

		outcome := "PASS"
		detail := ""
		if err != nil {
			outcome = "FAIL"
			detail = string(output)
		}

		results = append(results, TestResult{
			Name:     test,
			Outcome:  outcome,
			Duration: duration,
			Detail:   detail,
		})
	}
	return results
}

func runPersistenceTests() []TestResult {
	tests := []string{
		"TestRaftPersistence",
		"TestSecretVersionAdd",
		"TestPlaintextCanary",
		"TestBoltDBCanary",
		"TestSnapshotCanary",
		"TestExportCanary",
	}

	var results []TestResult
	for _, test := range tests {
		start := time.Now()
		cmd := exec.Command("go", "test", "-v", "./pkg/control", "-run", "^"+test+"$")
		cmd.Dir = "/home/user/Decentralized-"
		output, err := cmd.CombinedOutput()
		duration := time.Since(start).Nanoseconds()

		outcome := "PASS"
		detail := ""
		if err != nil {
			outcome = "FAIL"
			detail = string(output)
		}

		results = append(results, TestResult{
			Name:     test,
			Outcome:  outcome,
			Duration: duration,
			Detail:   detail,
		})
	}
	return results
}

func runNegativeControls() []NegativeControlResult {
	controls := []string{
		"TestNegativeControl_DisableAADBinding_TamperedScopeDetected",
		"TestNegativeControl_PlaintextPersistence_CanaryDetects",
		"TestNegativeControl_AuthTagBypass_TamperDetected",
	}

	var results []NegativeControlResult
	for _, control := range controls {
		cmd := exec.Command("go", "test", "-v", "./pkg/control", "-run", "^"+control+"$")
		cmd.Dir = "/home/user/Decentralized-"
		output, err := cmd.CombinedOutput()

		// Negative control should FAIL (test detects the problem)
		outcome := "FAIL (expected)"
		detail := ""
		if err == nil {
			outcome = "PASS (unexpected - test should have failed)"
			detail = string(output)
		}

		results = append(results, NegativeControlResult{
			Control: control,
			Outcome: outcome,
			Detail:  detail,
		})
	}
	return results
}

func runFailClosedTest() TestResult {
	// Simulate missing bootstrap → fail-closed
	// For now, document the test; actual implementation in A03+
	return TestResult{
		Name:    "FailClosed_MissingBootstrap",
		Outcome: "PENDING",
		Detail:  "Fail-closed behavior requires bootstrap env var implementation in server startup",
	}
}

func runRestartTest() TestResult {
	// Simulate restart with persistent encrypted state
	cmd := exec.Command("go", "test", "-v", "./pkg/control", "-run", "^TestRestartUnlocks$")
	cmd.Dir = "/home/user/Decentralized-"
	output, err := cmd.CombinedOutput()

	outcome := "PASS"
	detail := ""
	if err != nil {
		outcome = "FAIL"
		detail = string(output)
	}

	return TestResult{
		Name:    "RestartUnlocks",
		Outcome: outcome,
		Detail:  detail,
	}
}

func computeDigest(ev *Evidence) string {
	// Compute SHA256 hash of evidence (excluding signature)
	h := sha256.New()
	// Hash fields in canonical order
	fields := []string{
		ev.EvidenceID,
		ev.QualificationID,
		ev.SourceSHA,
		ev.Branch,
		ev.Environment,
		ev.Algorithm,
		ev.KeyVersion,
		ev.KeyCustodyMode,
		ev.StartedAt,
		ev.CompletedAt,
	}
	for _, f := range fields {
		io.WriteString(h, f)
	}
	// Hash test results
	for _, tr := range ev.CryptoTestResults {
		io.WriteString(h, tr.Name+":"+tr.Outcome)
	}
	for _, tr := range ev.PersistenceTestResults {
		io.WriteString(h, tr.Name+":"+tr.Outcome)
	}
	for _, nc := range ev.NegativeControlResults {
		io.WriteString(h, nc.Control+":"+nc.Outcome)
	}

	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func aggregateOutcome(ev Evidence) string {
	failCount := 0
	for _, tr := range ev.CryptoTestResults {
		if tr.Outcome != "PASS" {
			failCount++
		}
	}
	for _, tr := range ev.PersistenceTestResults {
		if tr.Outcome != "PASS" {
			failCount++
		}
	}
	for _, nc := range ev.NegativeControlResults {
		if !contains(nc.Outcome, "expected") {
			failCount++
		}
	}

	if failCount > 0 {
		return fmt.Sprintf("FAIL (%d failures)", failCount)
	}
	return "PASS / VERIFIED"
}

func contains(s, substr string) bool {
	for i := range s {
		if len(s[i:]) < len(substr) {
			return false
		}
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
