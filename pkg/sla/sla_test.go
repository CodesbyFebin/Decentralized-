package sla

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	if mgr == nil {
		t.Fatal("Manager is nil")
	}

	stats := mgr.GetStats()
	if stats.TotalSLAs != 0 {
		t.Error("Expected empty manager")
	}
}

func TestDefineSLA(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	def := &SLADefinition{
		ID:          "test-sla-1",
		ServiceName: "placement-service",
		Level:       SLALevel99p9,
		Enabled:     true,
	}

	err := mgr.DefineSLA(def)
	if err != nil {
		t.Fatalf("DefineSLA failed: %v", err)
	}

	retrieved, ok := mgr.GetSLA("test-sla-1")
	if !ok {
		t.Fatal("SLA not found")
	}

	if retrieved.UptimeTarget != 99.9 {
		t.Errorf("Expected uptime target 99.9, got %v", retrieved.UptimeTarget)
	}
}

func TestDefineSLANoID(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	def := &SLADefinition{
		ServiceName: "test-service",
		Level:       SLALevel99p99,
	}

	err := mgr.DefineSLA(def)
	if err == nil {
		t.Fatal("Expected error when defining SLA without ID")
	}
}

func TestDefineSLAInvalidLevel(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	def := &SLADefinition{
		ID:          "test-sla-1",
		ServiceName: "test-service",
		Level:       SLALevel("invalid"),
	}

	err := mgr.DefineSLA(def)
	if err == nil {
		t.Fatal("Expected error when defining SLA with invalid level")
	}
}

func TestSLALevels(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	levels := []struct {
		level  SLALevel
		target float64
	}{
		{SLALevel99p9, 99.9},
		{SLALevel99p99, 99.99},
		{SLALevel99p999, 99.999},
	}

	for i, lt := range levels {
		def := &SLADefinition{
			ID:          fmt.Sprintf("sla-%d", i),
			ServiceName: fmt.Sprintf("service-%d", i),
			Level:       lt.level,
			Enabled:     true,
		}

		err := mgr.DefineSLA(def)
		if err != nil {
			t.Fatalf("DefineSLA failed for level %s: %v", lt.level, err)
		}

		retrieved, ok := mgr.GetSLA(def.ID)
		if !ok {
			t.Fatalf("SLA not found for level %s", lt.level)
		}

		if retrieved.UptimeTarget != lt.target {
			t.Errorf("Expected uptime target %v for level %s, got %v", lt.target, lt.level, retrieved.UptimeTarget)
		}
	}
}

func TestUpdateUptime(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	def := &SLADefinition{
		ID:          "test-sla-1",
		ServiceName: "test-service",
		Level:       SLALevel99p9,
		Enabled:     true,
	}

	mgr.DefineSLA(def)

	// Update with good uptime
	err := mgr.UpdateUptime("test-sla-1", 99.95)
	if err != nil {
		t.Fatalf("UpdateUptime failed: %v", err)
	}

	// Update with bad uptime (should trigger violation)
	err = mgr.UpdateUptime("test-sla-1", 99.5)
	if err != nil {
		t.Fatalf("UpdateUptime failed: %v", err)
	}

	stats := mgr.GetStats()
	if stats.ViolationCount != 1 {
		t.Errorf("Expected 1 violation, got %d", stats.ViolationCount)
	}
}

func TestUpdateLatency(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	def := &SLADefinition{
		ID:              "test-sla-1",
		ServiceName:     "test-service",
		Level:           SLALevel99p9,
		Enabled:         true,
		ResponseTimeP99: 100 * time.Millisecond,
	}

	mgr.DefineSLA(def)

	// Update with good latency
	err := mgr.UpdateLatency("test-sla-1", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("UpdateLatency failed: %v", err)
	}

	// Update with bad latency
	err = mgr.UpdateLatency("test-sla-1", 150*time.Millisecond)
	if err != nil {
		t.Fatalf("UpdateLatency failed: %v", err)
	}

	stats := mgr.GetStats()
	if stats.ViolationCount != 1 {
		t.Errorf("Expected 1 violation, got %d", stats.ViolationCount)
	}
}

func TestUpdateErrorRate(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	def := &SLADefinition{
		ID:              "test-sla-1",
		ServiceName:     "test-service",
		Level:           SLALevel99p9,
		Enabled:         true,
		ErrorRateTarget: 0.01,
	}

	mgr.DefineSLA(def)

	// Update with good error rate
	err := mgr.UpdateErrorRate("test-sla-1", 0.005)
	if err != nil {
		t.Fatalf("UpdateErrorRate failed: %v", err)
	}

	// Update with bad error rate
	err = mgr.UpdateErrorRate("test-sla-1", 0.02)
	if err != nil {
		t.Fatalf("UpdateErrorRate failed: %v", err)
	}

	stats := mgr.GetStats()
	if stats.ViolationCount != 1 {
		t.Errorf("Expected 1 violation, got %d", stats.ViolationCount)
	}
}

func TestGetViolations(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	def := &SLADefinition{
		ID:          "test-sla-1",
		ServiceName: "test-service",
		Level:       SLALevel99p9,
		Enabled:     true,
	}

	mgr.DefineSLA(def)

	// Trigger violations
	mgr.UpdateUptime("test-sla-1", 99.5)
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	mgr.UpdateUptime("test-sla-1", 99.4)

	violations := mgr.GetViolations("test-sla-1")
	if len(violations) != 2 {
		t.Errorf("Expected 2 violations, got %d", len(violations))
	}

	for _, v := range violations {
		if v.ViolationType != "UPTIME" {
			t.Errorf("Expected violation type UPTIME, got %s", v.ViolationType)
		}
		if v.SLADefinitionID != "test-sla-1" {
			t.Errorf("Expected SLA ID test-sla-1, got %s", v.SLADefinitionID)
		}
	}
}

func TestGetComplianceReport(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	def := &SLADefinition{
		ID:          "test-sla-1",
		ServiceName: "placement-service",
		Level:       SLALevel99p9,
		Enabled:     true,
	}

	mgr.DefineSLA(def)

	mgr.UpdateUptime("test-sla-1", 99.95)

	report, err := mgr.GetComplianceReport("test-sla-1")
	if err != nil {
		t.Fatalf("GetComplianceReport failed: %v", err)
	}

	if report.SLAID != "test-sla-1" {
		t.Errorf("Expected SLA ID test-sla-1, got %s", report.SLAID)
	}

	if report.ServiceName != "placement-service" {
		t.Errorf("Expected service name placement-service, got %s", report.ServiceName)
	}

	if report.WindowCount != 1 {
		t.Errorf("Expected 1 window, got %d", report.WindowCount)
	}
}

func TestComplianceReportNonexistentSLA(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	_, err := mgr.GetComplianceReport("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent SLA")
	}
}

func TestStartStop(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	mgr.Start()

	def := &SLADefinition{
		ID:          "test-sla-1",
		ServiceName: "test-service",
		Level:       SLALevel99p9,
		Enabled:     true,
	}

	mgr.DefineSLA(def)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := mgr.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestMultipleSLAs(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	for i := 0; i < 3; i++ {
		def := &SLADefinition{
			ID:          fmt.Sprintf("sla-%d", i),
			ServiceName: fmt.Sprintf("service-%d", i),
			Level:       SLALevel99p9,
			Enabled:     true,
		}
		mgr.DefineSLA(def)
	}

	stats := mgr.GetStats()
	if stats.TotalSLAs != 3 {
		t.Errorf("Expected 3 SLAs, got %d", stats.TotalSLAs)
	}
}

func TestRemediationAction(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	remediationCalled := false
	remediationAction := func(ctx context.Context, violation *SLAViolation) error {
		remediationCalled = true
		return nil
	}

	def := &SLADefinition{
		ID:                 "test-sla-1",
		ServiceName:        "test-service",
		Level:              SLALevel99p9,
		Enabled:            true,
		RemediationAction:  remediationAction,
	}

	mgr.DefineSLA(def)

	// Wait a bit for the goroutine to start
	mgr.UpdateUptime("test-sla-1", 99.5)
	time.Sleep(100 * time.Millisecond)

	if !remediationCalled {
		t.Error("Remediation action was not called")
	}
}

func TestDisabledSLA(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	def := &SLADefinition{
		ID:          "test-sla-1",
		ServiceName: "test-service",
		Level:       SLALevel99p9,
		Enabled:     false,
	}

	mgr.DefineSLA(def)

	// Update should not trigger violations
	mgr.UpdateUptime("test-sla-1", 99.5)

	stats := mgr.GetStats()
	if stats.ViolationCount != 0 {
		t.Errorf("Expected 0 violations for disabled SLA, got %d", stats.ViolationCount)
	}
}

func TestGetStats(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	def := &SLADefinition{
		ID:          "test-sla-1",
		ServiceName: "test-service",
		Level:       SLALevel99p9,
		Enabled:     true,
	}

	mgr.DefineSLA(def)

	mgr.UpdateUptime("test-sla-1", 99.5)
	mgr.UpdateUptime("test-sla-1", 99.4)

	stats := mgr.GetStats()
	if stats.TotalSLAs != 1 {
		t.Errorf("Expected 1 SLA, got %d", stats.TotalSLAs)
	}

	if stats.ViolationCount != 2 {
		t.Errorf("Expected 2 violations, got %d", stats.ViolationCount)
	}
}

func TestViolationDetails(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	def := &SLADefinition{
		ID:          "test-sla-1",
		ServiceName: "test-service",
		Level:       SLALevel99p9,
		Enabled:     true,
	}

	mgr.DefineSLA(def)

	mgr.UpdateUptime("test-sla-1", 99.5)

	violations := mgr.GetViolations("test-sla-1")
	if len(violations) == 0 {
		t.Fatal("No violations found")
	}

	v := violations[0]
	if v.Details["target"] != 99.9 {
		t.Errorf("Expected target 99.9, got %v", v.Details["target"])
	}

	if v.Details["actual"] != 99.5 {
		t.Errorf("Expected actual 99.5, got %v", v.Details["actual"])
	}

	deficit := v.Details["deficit"].(float64)
	// Use approximate equality for floating point
	if deficit < 0.39 || deficit > 0.41 {
		t.Errorf("Expected deficit approximately 0.4, got %v", deficit)
	}
}
