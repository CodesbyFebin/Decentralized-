package chaos

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestM7ChaosValidation executes all 17 chaos scenarios and validates invariants.
// This is the main entry point for M7 (Chaos Framework) qualification.
func TestM7ChaosValidation(t *testing.T) {
	// Initialize test cluster
	tc := NewLocalTestCluster("podman", "localhost:7700")

	scenarios := AllScenarios()
	if len(scenarios) != 17 {
		t.Fatalf("expected 17 scenarios, got %d", len(scenarios))
	}

	// Track results
	results := make([]*ScenarioResult, 0, len(scenarios))
	passCount := 0
	failCount := 0

	t.Logf("M7 Chaos Framework: Executing %d scenarios", len(scenarios))
	t.Logf("==================================================================================")

	// Execute each scenario
	for _, scenario := range scenarios {
		result := executeScenario(t, tc, scenario)
		results = append(results, result)

		if result.Passed {
			passCount++
			t.Logf("✓ [%s] %s (%.2fs)", scenario.Category, scenario.Title, result.DurationSec)
		} else {
			failCount++
			t.Logf("✗ [%s] %s (%.2fs)", scenario.Category, scenario.Title, result.DurationSec)
			t.Logf("  Error: %s", result.Error)
			if result.Invariants != nil && !result.Invariants.IsValid() {
				t.Logf("  %s", result.Invariants.Summary())
			}
		}
	}

	// Print summary
	t.Logf("==================================================================================")
	t.Logf("M7 Results: %d PASS, %d FAIL", passCount, failCount)
	t.Logf("==================================================================================")

	// Detailed results for debugging
	if failCount > 0 {
		t.Logf("\nFailed Scenarios:")
		for _, result := range results {
			if !result.Passed {
				t.Logf("\n[%s] %s", result.Scenario.Category, result.Scenario.Title)
				t.Logf("  Error: %s", result.Error)
				if result.Invariants != nil && !result.Invariants.IsValid() {
					t.Logf("  Violations:\n%s", result.Invariants.Summary())
				}
			}
		}
	}

	// All scenarios must pass
	if failCount > 0 {
		t.Fatalf("M7 Chaos Framework validation FAILED: %d scenarios failed", failCount)
	}

	t.Logf("✓ M7 Chaos Framework validation PASSED")
}

// ScenarioResult records the outcome of a single scenario execution.
type ScenarioResult struct {
	Scenario     *Scenario
	Passed       bool
	Error        string
	DurationSec  float64
	Invariants   *Invariants
	StartTime    time.Time
	EndTime      time.Time
}

// executeScenario runs a single chaos scenario and validates invariants.
func executeScenario(t *testing.T, tc TestCluster, scenario *Scenario) *ScenarioResult {
	result := &ScenarioResult{
		Scenario:  scenario,
		StartTime: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), scenario.Duration+30*time.Second)
	defer cancel()

	// Phase 1: Setup
	if err := scenario.Setup(ctx, tc); err != nil {
		result.Error = fmt.Sprintf("setup failed: %v", err)
		result.EndTime = time.Now()
		result.DurationSec = result.EndTime.Sub(result.StartTime).Seconds()
		return result
	}

	// Phase 2: Inject failure
	if err := scenario.Inject(ctx, tc); err != nil {
		result.Error = fmt.Sprintf("injection failed: %v", err)
		result.EndTime = time.Now()
		result.DurationSec = result.EndTime.Sub(result.StartTime).Seconds()
		return result
	}

	// Let scenario run for its duration
	select {
	case <-time.After(scenario.Duration):
	case <-ctx.Done():
	}

	// Phase 3: Verify behavior and invariants
	if err := scenario.Verify(ctx, tc); err != nil {
		result.Error = fmt.Sprintf("verification failed: %v", err)
		result.EndTime = time.Now()
		result.DurationSec = result.EndTime.Sub(result.StartTime).Seconds()
		result.Invariants = CheckInvariants(ctx, tc, scenario)
		return result
	}

	// Phase 4: Cleanup and check invariants
	if err := scenario.Cleanup(ctx, tc); err != nil {
		result.Error = fmt.Sprintf("cleanup failed: %v", err)
		result.EndTime = time.Now()
		result.DurationSec = result.EndTime.Sub(result.StartTime).Seconds()
		result.Invariants = CheckInvariants(ctx, tc, scenario)
		return result
	}

	// Final invariant check
	result.Invariants = CheckInvariants(ctx, tc, scenario)
	if !result.Invariants.IsValid() {
		result.Error = "invariant violations detected"
	} else {
		result.Passed = true
	}

	result.EndTime = time.Now()
	result.DurationSec = result.EndTime.Sub(result.StartTime).Seconds()
	return result
}

// TestM7Individual allows testing individual scenarios for debugging.
// Usage: go test -run TestM7Individual -args scenario-id
func TestM7Individual(t *testing.T) {
	// This would be enhanced to support individual scenario execution
	// For now, it's a placeholder for debugging specific scenarios
	t.Skip("Use TestM7ChaosValidation for full validation")
}

// BenchmarkM7Throughput measures system throughput under sustained load.
func BenchmarkM7Throughput(b *testing.B) {
	tc := NewLocalTestCluster("podman", "localhost:7700")
	ctx := context.Background()

	// Start load generator
	if err := tc.StartLoadGenerator(ctx, 1000); err != nil {
		b.Fatalf("failed to start load: %v", err)
	}
	defer tc.StopLoadGenerator(ctx)

	b.ResetTimer()

	// Verify API remains responsive under load
	for i := 0; i < b.N; i++ {
		if err := tc.VerifyAPIResponsive(ctx); err != nil {
			b.Fatalf("API unresponsive under load: %v", err)
		}
	}
}

// BenchmarkM7Recovery measures recovery time after failures.
func BenchmarkM7Recovery(b *testing.B) {
	tc := NewLocalTestCluster("podman", "localhost:7700")
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer() // Setup phase

		// Kill a node
		if err := tc.KillNode(ctx, "provider-1"); err != nil {
			b.Fatalf("failed to kill node: %v", err)
		}

		b.StartTimer() // Measure recovery

		// Time until cluster recovers
		startRecovery := time.Now()
		recoveryTimeout := time.After(10 * time.Second)
		for {
			select {
			case <-recoveryTimeout:
				b.Fatalf("recovery timeout")
			default:
				if err := tc.VerifyAPIResponsive(ctx); err == nil {
					if err := tc.VerifyRaftLeader(ctx); err == nil {
						elapsed := time.Since(startRecovery)
						b.ReportMetric(elapsed.Seconds(), "recovery_time_s")
						break
					}
				}
				time.Sleep(100 * time.Millisecond)
			}
		}

		b.StopTimer() // Cleanup phase
		tc.RestartNode(ctx, "provider-1")
	}
}

// M7ScenarioStats provides statistics for a single scenario.
type M7ScenarioStats struct {
	ScenarioID       string
	Category         string
	ExecutionCount   int
	PassCount        int
	FailCount        int
	AverageDuration  float64
	MaxDuration      float64
	MinDuration      float64
	LastError        string
}

// TestM7Statistics tracks scenario execution statistics over multiple runs.
// This would be integrated with CI/CD to track trend analysis.
func TestM7Statistics(t *testing.T) {
	t.Skip("Statistics tracking requires persistent storage (database)")
	// In a real implementation:
	// 1. Connect to statistics database
	// 2. Record each scenario execution
	// 3. Calculate rolling averages and trends
	// 4. Alert on regressions (e.g., recovery time increases)
}
