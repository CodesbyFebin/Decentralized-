package alerts

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
	if stats.TotalRules != 0 || stats.ActiveAlerts != 0 {
		t.Error("Expected empty manager")
	}
}

func TestAddRule(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	rule := &Rule{
		ID:                 "test-rule-1",
		Name:               "High CPU",
		Description:        "CPU usage is high",
		Severity:           SeverityHigh,
		MetricName:         "cpu_usage",
		Condition:          "> 0.9",
		Threshold:          0.9,
		EvaluationInterval: time.Minute,
		Enabled:            true,
	}

	err := mgr.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	retrieved, ok := mgr.GetRule("test-rule-1")
	if !ok {
		t.Fatal("Rule not found")
	}

	if retrieved.Name != "High CPU" {
		t.Errorf("Expected rule name 'High CPU', got '%s'", retrieved.Name)
	}
}

func TestAddRuleNoID(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	rule := &Rule{
		Name:     "No ID Rule",
		Severity: SeverityLow,
	}

	err := mgr.AddRule(rule)
	if err == nil {
		t.Fatal("Expected error when adding rule without ID")
	}
}

func TestRemoveRule(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	rule := &Rule{
		ID:       "test-rule-1",
		Name:     "Test Rule",
		Severity: SeverityLow,
		Enabled:  true,
	}

	mgr.AddRule(rule)
	mgr.RemoveRule("test-rule-1")

	_, ok := mgr.GetRule("test-rule-1")
	if ok {
		t.Fatal("Rule should have been removed")
	}
}

func TestFireAlert(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	rule := &Rule{
		ID:       "test-rule-1",
		Name:     "High CPU",
		Severity: SeverityCritical,
		Enabled:  true,
		Routes:   []string{},
	}

	mgr.AddRule(rule)

	alert := &Alert{
		Summary:     "CPU usage is 95%",
		Description: "CPU usage exceeded threshold",
		Labels: map[string]string{
			"host": "node-1",
		},
	}

	ctx := context.Background()
	err := mgr.FireAlert(ctx, "test-rule-1", alert)
	if err != nil {
		t.Fatalf("FireAlert failed: %v", err)
	}

	stats := mgr.GetStats()
	if stats.ActiveAlerts != 1 {
		t.Errorf("Expected 1 active alert, got %d", stats.ActiveAlerts)
	}
}

func TestFireAlertNonexistentRule(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	alert := &Alert{
		Summary: "Test alert",
	}

	ctx := context.Background()
	err := mgr.FireAlert(ctx, "nonexistent-rule", alert)
	if err == nil {
		t.Fatal("Expected error when firing alert with nonexistent rule")
	}
}

func TestResolveAlert(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	rule := &Rule{
		ID:       "test-rule-1",
		Name:     "Test Rule",
		Severity: SeverityHigh,
		Enabled:  true,
	}

	mgr.AddRule(rule)

	alert := &Alert{
		Summary: "Test alert",
	}

	ctx := context.Background()
	mgr.FireAlert(ctx, "test-rule-1", alert)

	// Get the alert ID
	activeAlerts := mgr.ListActiveAlerts()
	if len(activeAlerts) == 0 {
		t.Fatal("No active alerts found")
	}

	alertID := activeAlerts[0].ID
	err := mgr.ResolveAlert(alertID)
	if err != nil {
		t.Fatalf("ResolveAlert failed: %v", err)
	}

	alert, ok := mgr.GetAlert(alertID)
	if !ok {
		t.Fatal("Alert not found after resolution")
	}

	if alert.State != StateResolved {
		t.Errorf("Expected alert state RESOLVED, got %s", alert.State)
	}

	if alert.ResolvedAt == nil {
		t.Error("Expected ResolvedAt to be set")
	}
}

func TestSuppressAlert(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	rule := &Rule{
		ID:       "test-rule-1",
		Name:     "Test Rule",
		Severity: SeverityMedium,
		Enabled:  true,
	}

	mgr.AddRule(rule)

	alert := &Alert{
		Summary: "Test alert",
	}

	ctx := context.Background()
	mgr.FireAlert(ctx, "test-rule-1", alert)

	activeAlerts := mgr.ListActiveAlerts()
	if len(activeAlerts) == 0 {
		t.Fatal("No active alerts found")
	}

	alertID := activeAlerts[0].ID
	err := mgr.SuppressAlert(alertID)
	if err != nil {
		t.Fatalf("SuppressAlert failed: %v", err)
	}

	alert, ok := mgr.GetAlert(alertID)
	if !ok {
		t.Fatal("Alert not found after suppression")
	}

	if alert.State != StateSuppressed {
		t.Errorf("Expected alert state SUPPRESSED, got %s", alert.State)
	}
}

func TestAlertDeduplication(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DeduplicationWindow = 100 * time.Millisecond
	mgr := NewManager(cfg)

	rule := &Rule{
		ID:       "test-rule-1",
		Name:     "Test Rule",
		Severity: SeverityLow,
		Enabled:  true,
	}

	mgr.AddRule(rule)

	alert := &Alert{
		Summary: "Test alert",
		Labels: map[string]string{
			"host": "node-1",
		},
	}

	ctx := context.Background()

	// Fire the same alert twice quickly
	err1 := mgr.FireAlert(ctx, "test-rule-1", alert)
	err2 := mgr.FireAlert(ctx, "test-rule-1", alert)

	if err1 != nil || err2 != nil {
		t.Fatal("FireAlert failed")
	}

	// Second alert should be deduplicated
	stats := mgr.GetStats()
	if stats.ActiveAlerts != 1 {
		t.Errorf("Expected 1 active alert after deduplication, got %d", stats.ActiveAlerts)
	}

	// Wait for deduplication window to pass
	time.Sleep(150 * time.Millisecond)

	// Fire again - should not be deduplicated
	err3 := mgr.FireAlert(ctx, "test-rule-1", alert)
	if err3 != nil {
		t.Fatal("Third FireAlert failed")
	}

	stats = mgr.GetStats()
	if stats.ActiveAlerts != 2 {
		t.Errorf("Expected 2 active alerts after deduplication window, got %d", stats.ActiveAlerts)
	}
}

func TestListActiveAlerts(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	for i := 0; i < 3; i++ {
		rule := &Rule{
			ID:       fmt.Sprintf("rule-%d", i),
			Name:     fmt.Sprintf("Rule %d", i),
			Severity: SeverityLow,
			Enabled:  true,
		}
		mgr.AddRule(rule)

		alert := &Alert{
			Summary: fmt.Sprintf("Alert %d", i),
		}

		ctx := context.Background()
		mgr.FireAlert(ctx, rule.ID, alert)
	}

	activeAlerts := mgr.ListActiveAlerts()
	if len(activeAlerts) != 3 {
		t.Errorf("Expected 3 active alerts, got %d", len(activeAlerts))
	}

	// Resolve one alert
	mgr.ResolveAlert(activeAlerts[0].ID)

	activeAlerts = mgr.ListActiveAlerts()
	if len(activeAlerts) != 2 {
		t.Errorf("Expected 2 active alerts after resolution, got %d", len(activeAlerts))
	}
}

func TestListAlerts(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	rule := &Rule{
		ID:       "test-rule-1",
		Name:     "Test Rule",
		Severity: SeverityLow,
		Enabled:  true,
	}

	mgr.AddRule(rule)

	for i := 0; i < 2; i++ {
		alert := &Alert{
			Summary: fmt.Sprintf("Alert %d", i),
			Labels: map[string]string{
				"index": fmt.Sprintf("%d", i),
			},
		}
		ctx := context.Background()
		mgr.FireAlert(ctx, "test-rule-1", alert)
	}

	allAlerts := mgr.ListAlerts()
	if len(allAlerts) != 2 {
		t.Errorf("Expected 2 total alerts, got %d", len(allAlerts))
	}
}

func TestRegisterRoute(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	callCount := 0
	handler := func(ctx context.Context, alert *Alert) error {
		callCount++
		return nil
	}

	mgr.RegisterRoute("slack", handler)

	rule := &Rule{
		ID:       "test-rule-1",
		Name:     "Test Rule",
		Severity: SeverityHigh,
		Enabled:  true,
		Routes:   []string{"slack"},
	}

	mgr.AddRule(rule)

	alert := &Alert{
		Summary: "Test alert",
	}

	ctx := context.Background()
	mgr.FireAlert(ctx, "test-rule-1", alert)

	if callCount != 1 {
		t.Errorf("Expected route handler to be called once, got %d", callCount)
	}
}

func TestStartStop(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	mgr.Start()

	// Add a rule
	rule := &Rule{
		ID:       "test-rule-1",
		Name:     "Test Rule",
		Severity: SeverityLow,
		Enabled:  true,
	}

	mgr.AddRule(rule)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := mgr.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestAlertSeverities(t *testing.T) {
	severities := []Severity{
		SeverityCritical,
		SeverityHigh,
		SeverityMedium,
		SeverityLow,
	}

	if len(severities) != 4 {
		t.Errorf("Expected 4 severities, got %d", len(severities))
	}
}

func TestAlertStates(t *testing.T) {
	states := []AlertState{
		StateActive,
		StatePending,
		StateResolved,
		StateSuppressed,
	}

	if len(states) != 4 {
		t.Errorf("Expected 4 states, got %d", len(states))
	}
}

func TestGetStats(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	rule := &Rule{
		ID:       "test-rule-1",
		Name:     "Test Rule",
		Severity: SeverityLow,
		Enabled:  true,
	}

	mgr.AddRule(rule)

	stats := mgr.GetStats()
	if stats.TotalRules != 1 {
		t.Errorf("Expected 1 rule, got %d", stats.TotalRules)
	}

	if stats.ActiveAlerts != 0 {
		t.Errorf("Expected 0 active alerts, got %d", stats.ActiveAlerts)
	}

	// Fire an alert
	alert := &Alert{
		Summary: "Test alert",
	}

	ctx := context.Background()
	mgr.FireAlert(ctx, "test-rule-1", alert)

	stats = mgr.GetStats()
	if stats.ActiveAlerts != 1 {
		t.Errorf("Expected 1 active alert, got %d", stats.ActiveAlerts)
	}

	if stats.FiredCount != 1 {
		t.Errorf("Expected fired count 1, got %d", stats.FiredCount)
	}
}
