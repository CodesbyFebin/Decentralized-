package tracing

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

func TestNewManager(t *testing.T) {
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

func TestTracerWithDisabledTracing(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx, span := mgr.StartSpan(context.Background(), "test-op")
	if ctx == nil {
		t.Fatal("Context is nil")
	}
	if span == nil {
		t.Fatal("Span is nil")
	}
}

func TestRecordLatency(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Record some latencies
	mgr.RecordLatency("test-op", 10*time.Millisecond)
	mgr.RecordLatency("test-op", 20*time.Millisecond)
	mgr.RecordLatency("test-op", 30*time.Millisecond)

	stats, ok := mgr.GetLatencyStats("test-op")
	if !ok {
		t.Fatal("Stats not found")
	}

	if stats.Count != 3 {
		t.Errorf("Expected count 3, got %d", stats.Count)
	}

	if stats.Min != 10*time.Millisecond {
		t.Errorf("Expected min 10ms, got %v", stats.Min)
	}

	if stats.Max != 30*time.Millisecond {
		t.Errorf("Expected max 30ms, got %v", stats.Max)
	}

	expectedMean := 20 * time.Millisecond
	if stats.Mean != expectedMean {
		t.Errorf("Expected mean 20ms, got %v", stats.Mean)
	}
}

func TestGetLatencyStatsNonexistent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, ok := mgr.GetLatencyStats("nonexistent")
	if ok {
		t.Fatal("Expected stats not found")
	}
}

func TestStartSpanWithAttributes(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx := context.Background()
	ctx, span := mgr.StartSpan(ctx, "test-op",
		attribute.String("request_id", "123"),
		attribute.Int("retry_count", 1),
	)

	if ctx == nil {
		t.Fatal("Context is nil")
	}

	if span == nil {
		t.Fatal("Span is nil")
	}
}

func TestGetStats(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	mgr.RecordLatency("op1", 10*time.Millisecond)
	mgr.RecordLatency("op2", 20*time.Millisecond)

	stats := mgr.GetStats()
	if len(stats.LatencyMetrics) != 2 {
		t.Errorf("Expected 2 operations, got %d", len(stats.LatencyMetrics))
	}
}

func TestShutdown(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = mgr.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestClose(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = mgr.Close(ctx)
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestMultipleOperations(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	operations := []string{"placement", "policy", "storage"}
	for _, op := range operations {
		for i := 0; i < 10; i++ {
			duration := time.Duration(i*10+1) * time.Millisecond
			mgr.RecordLatency(op, duration)
		}
	}

	stats := mgr.GetStats()
	if len(stats.LatencyMetrics) != 3 {
		t.Errorf("Expected 3 operations, got %d", len(stats.LatencyMetrics))
	}

	for _, op := range operations {
		s, ok := mgr.GetLatencyStats(op)
		if !ok {
			t.Errorf("Stats not found for operation %s", op)
		}
		if s.Count != 10 {
			t.Errorf("Expected count 10 for %s, got %d", op, s.Count)
		}
	}
}

func TestLatencyPercentiles(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Record 100 latencies
	for i := 1; i <= 100; i++ {
		mgr.RecordLatency("test", time.Duration(i)*time.Millisecond)
	}

	stats, ok := mgr.GetLatencyStats("test")
	if !ok {
		t.Fatal("Stats not found")
	}

	// Percentiles should be within expected ranges
	if stats.P50 == 0 || stats.P95 == 0 || stats.P99 == 0 {
		t.Errorf("Percentiles not calculated: P50=%v, P95=%v, P99=%v", stats.P50, stats.P95, stats.P99)
	}

	if stats.P50 > stats.P95 {
		t.Error("P50 should be less than P95")
	}

	if stats.P95 > stats.P99 {
		t.Error("P95 should be less than P99")
	}
}
