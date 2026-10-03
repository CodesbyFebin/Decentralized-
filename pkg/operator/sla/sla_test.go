package sla

import (
	"testing"
	"time"
)

func TestRegisterOperator(t *testing.T) {
	sm := NewSLAManager()

	_, err := sm.RegisterOperator("op_001")
	if err != nil {
		t.Fatalf("RegisterOperator() error = %v", err)
	}

	// Register same operator again (should fail)
	_, err = sm.RegisterOperator("op_001")
	if err == nil {
		t.Error("expected error registering same operator twice")
	}
}

func TestRecordMetrics(t *testing.T) {
	sm := NewSLAManager()
	operatorID := "op_001"
	sm.RegisterOperator(operatorID)

	latencies := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		75 * time.Millisecond,
		60 * time.Millisecond,
		80 * time.Millisecond,
	}

	err := sm.RecordMetrics(operatorID, 95, 5, latencies)
	if err != nil {
		t.Fatalf("RecordMetrics() error = %v", err)
	}

	metrics, _ := sm.GetMetrics(operatorID)
	if metrics.CurrentUptime != 95.0 {
		t.Errorf("expected uptime 95.0, got %.1f", metrics.CurrentUptime)
	}
}

func TestComplianceStatus(t *testing.T) {
	sm := NewSLAManager()
	operatorID := "op_001"
	sm.RegisterOperator(operatorID)

	// Record compliant metrics
	latencies := make([]time.Duration, 100)
	for i := 0; i < 100; i++ {
		latencies[i] = 100 * time.Millisecond // All under 1s target
	}

	sm.RecordMetrics(operatorID, 99, 1, latencies)
	metrics, _ := sm.GetMetrics(operatorID)

	if metrics.Status != COMPLIANT {
		t.Errorf("expected COMPLIANT status, got %s", metrics.Status)
	}

	// Record non-compliant metrics
	sm.RecordMetrics(operatorID, 80, 20, latencies) // 80% uptime violates 95% target
	metrics, _ = sm.GetMetrics(operatorID)

	if metrics.Status != VIOLATED {
		t.Errorf("expected VIOLATED status, got %s", metrics.Status)
	}
}

func TestComplianceReport(t *testing.T) {
	sm := NewSLAManager()
	operatorID := "op_001"
	sm.RegisterOperator(operatorID)

	latencies := make([]time.Duration, 50)
	for i := 0; i < 50; i++ {
		latencies[i] = 100 * time.Millisecond
	}

	sm.RecordMetrics(operatorID, 100, 0, latencies)

	report, err := sm.GenerateReport(operatorID)
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	if report.Status != COMPLIANT {
		t.Errorf("expected COMPLIANT report, got %s", report.Status)
	}
}

func TestViolationTracking(t *testing.T) {
	sm := NewSLAManager()
	operatorID := "op_001"
	sm.RegisterOperator(operatorID)

	latencies := []time.Duration{100 * time.Millisecond}

	// Record violations
	for i := 0; i < 3; i++ {
		sm.RecordMetrics(operatorID, 80, 20, latencies)
	}

	metrics, _ := sm.GetMetrics(operatorID)
	if metrics.ViolationCount != 3 {
		t.Errorf("expected 3 violations, got %d", metrics.ViolationCount)
	}

	if metrics.LastViolationTime == nil {
		t.Error("expected LastViolationTime to be set")
	}
}

func TestStatusSummary(t *testing.T) {
	sm := NewSLAManager()

	// Register multiple operators
	for i := 0; i < 10; i++ {
		operatorID := "op_00" + string(rune('0'+i))
		sm.RegisterOperator(operatorID)

		latencies := []time.Duration{100 * time.Millisecond}
		if i < 5 {
			// First 5 compliant
			sm.RecordMetrics(operatorID, 99, 1, latencies)
		} else {
			// Last 5 violated
			sm.RecordMetrics(operatorID, 80, 20, latencies)
		}
	}

	summary := sm.GetStatusSummary()
	if summary.TotalOperators != 10 {
		t.Errorf("expected 10 operators, got %d", summary.TotalOperators)
	}

	if summary.CompliantOperators != 5 {
		t.Errorf("expected 5 compliant operators, got %d", summary.CompliantOperators)
	}

	if summary.ViolatedOperators != 5 {
		t.Errorf("expected 5 violated operators, got %d", summary.ViolatedOperators)
	}
}

func TestUpTimeTrending(t *testing.T) {
	sm := NewSLAManager()
	operatorID := "op_001"
	sm.RegisterOperator(operatorID)

	latencies := []time.Duration{100 * time.Millisecond}

	// Record degrading uptime
	sm.RecordMetrics(operatorID, 95, 5, latencies)
	metrics, _ := sm.GetMetrics(operatorID)
	if !metrics.TrendingUptime {
		t.Error("expected uptime to be trending up initially")
	}

	sm.RecordMetrics(operatorID, 80, 20, latencies)
	metrics, _ = sm.GetMetrics(operatorID)
	if metrics.TrendingUptime {
		t.Error("expected uptime to be trending down")
	}
}

func TestConsecutiveFailures(t *testing.T) {
	sm := NewSLAManager()
	operatorID := "op_001"
	sm.RegisterOperator(operatorID)

	latencies := []time.Duration{100 * time.Millisecond}

	// Record multiple failures
	for i := 0; i < 3; i++ {
		sm.RecordMetrics(operatorID, 50, 50, latencies)
	}

	metrics, _ := sm.GetMetrics(operatorID)
	if metrics.ConsecutiveFailures != 3 {
		t.Errorf("expected 3 consecutive failures, got %d", metrics.ConsecutiveFailures)
	}

	// Record success
	sm.RecordMetrics(operatorID, 99, 1, latencies)
	metrics, _ = sm.GetMetrics(operatorID)
	if metrics.ConsecutiveFailures != 0 {
		t.Errorf("expected consecutive failures to reset, got %d", metrics.ConsecutiveFailures)
	}
}

func TestLatencyMetrics(t *testing.T) {
	latencies := []time.Duration{
		10 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		150 * time.Millisecond,
		200 * time.Millisecond,
	}

	bucket := calculateLatencyMetrics(time.Now(), latencies)

	if bucket.MinLatency != 10*time.Millisecond {
		t.Errorf("expected min 10ms, got %v", bucket.MinLatency)
	}

	if bucket.MaxLatency != 200*time.Millisecond {
		t.Errorf("expected max 200ms, got %v", bucket.MaxLatency)
	}

	if bucket.Count != 5 {
		t.Errorf("expected count 5, got %d", bucket.Count)
	}
}

func TestAlertLevels(t *testing.T) {
	sm := NewSLAManager()

	tests := []struct {
		name        string
		uptime      float64
		latency     time.Duration
		expectedStatus  ComplianceStatus
		expectedAlert   string
	}{
		{"compliant", 99.0, 100 * time.Millisecond, COMPLIANT, "NONE"},
		{"at_risk", 90.0, 100 * time.Millisecond, AT_RISK, "WARNING"},
		{"violated", 80.0, 2 * time.Second, VIOLATED, "CRITICAL"},
	}

	for i, tt := range tests {
		operatorID := "op_" + string(rune('0'+i))
		sm.RegisterOperator(operatorID)

		successCount := int64(float64(100) * tt.uptime / 100.0)
		failureCount := 100 - successCount

		latencies := []time.Duration{tt.latency}
		sm.RecordMetrics(operatorID, successCount, failureCount, latencies)

		metrics, _ := sm.GetMetrics(operatorID)
		if metrics.Status != tt.expectedStatus {
			t.Errorf("test %s: expected status %s, got %s", tt.name, tt.expectedStatus, metrics.Status)
		}

		if metrics.AlertLevel != tt.expectedAlert {
			t.Errorf("test %s: expected alert %s, got %s", tt.name, tt.expectedAlert, metrics.AlertLevel)
		}
	}
}
