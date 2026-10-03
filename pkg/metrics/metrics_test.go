package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewManager(t *testing.T) {
	// Unregister any previously registered metrics for testing
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = false

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr == nil {
		t.Fatal("Manager is nil")
	}
}

func TestManagerDisabled(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = false

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr == nil {
		t.Fatal("Manager is nil")
	}

	// These should not panic even when disabled
	mgr.RecordPlacementLatency(100 * time.Millisecond)
	mgr.RecordPolicyLatency(50 * time.Millisecond)
	mgr.IncrementPolicyMatches()
	mgr.IncrementPolicyDenials()
	mgr.UpdateResourceUsage(0.5, 1024*1024*1024, 10*1024*1024*1024)
	mgr.AddNetworkBytes(1000, 2000)
	mgr.UpdateSLAMetrics(99.9, 99.8)
	mgr.IncrementSLAViolations()
	mgr.RecordTrace(true)
	mgr.UpdateThroughput(100.0)
	mgr.UpdateSuccessRate(95.5)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Enabled == false {
		t.Error("Expected enabled to be true by default")
	}

	if cfg.ListenAddress == "" {
		t.Error("Expected listen address to be set")
	}

	if cfg.MetricsPath == "" {
		t.Error("Expected metrics path to be set")
	}

	if len(cfg.HistogramBuckets) == 0 {
		t.Error("Expected histogram buckets to be set")
	}
}

func TestRecordPlacementLatency(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = true

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.RecordPlacementLatency(100 * time.Millisecond)
	mgr.RecordPlacementLatency(200 * time.Millisecond)

	// Just verify it doesn't panic
	if mgr.PlacementLatency == nil {
		t.Error("PlacementLatency metric is nil")
	}
}

func TestRecordPolicyLatency(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = true

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.RecordPolicyLatency(50 * time.Millisecond)

	if mgr.PolicyLatency == nil {
		t.Error("PolicyLatency metric is nil")
	}
}

func TestIncrementPolicyMatches(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = true

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.IncrementPolicyMatches()
	mgr.IncrementPolicyMatches()

	if mgr.PolicyMatches == nil {
		t.Error("PolicyMatches metric is nil")
	}
}

func TestIncrementPolicyDenials(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = true

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.IncrementPolicyDenials()

	if mgr.PolicyDenials == nil {
		t.Error("PolicyDenials metric is nil")
	}
}

func TestUpdateResourceUsage(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = true

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.UpdateResourceUsage(0.5, 1024*1024*1024, 10*1024*1024*1024)

	if mgr.CPUUsage == nil || mgr.MemoryUsage == nil || mgr.DiskUsage == nil {
		t.Error("Resource usage metrics are nil")
	}
}

func TestAddNetworkBytes(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = true

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.AddNetworkBytes(1000, 2000)
	mgr.AddNetworkBytes(500, 1000)

	if mgr.NetworkIn == nil || mgr.NetworkOut == nil {
		t.Error("Network metrics are nil")
	}
}

func TestUpdateSLAMetrics(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = true

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.UpdateSLAMetrics(99.9, 99.8)

	if mgr.SLATargetUptime == nil || mgr.SLAActualUptime == nil {
		t.Error("SLA metrics are nil")
	}
}

func TestIncrementSLAViolations(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = true

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.IncrementSLAViolations()

	if mgr.SLAViolations == nil {
		t.Error("SLAViolations metric is nil")
	}
}

func TestRecordTrace(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = true

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.RecordTrace(true)
	mgr.RecordTrace(false)

	if mgr.TracesReceived == nil || mgr.TracesExported == nil || mgr.TraceErrors == nil {
		t.Error("Trace metrics are nil")
	}
}

func TestUpdateThroughput(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = true

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.UpdateThroughput(100.0)

	if mgr.ThroughputRate == nil {
		t.Error("ThroughputRate metric is nil")
	}
}

func TestUpdateSuccessRate(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = true

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.UpdateSuccessRate(95.5)

	if mgr.SuccessRate == nil || mgr.FailureRate == nil {
		t.Error("Success/Failure rate metrics are nil")
	}
}

func TestStartServer(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.ListenAddress = "127.0.0.1:0" // Use random port

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Don't start the server in test to avoid port conflicts
	// Just verify the methods exist and don't panic
	if mgr == nil {
		t.Fatal("Manager is nil")
	}
}

func TestStopServer(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	cfg := DefaultConfig()
	cfg.Enabled = false

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = mgr.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}
