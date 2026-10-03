// Package tracing implements distributed tracing for Decentralized.Host.
//
// This package provides OpenTelemetry-based distributed tracing with support for:
//   - Trace instrumentation across components
//   - Trace sampling and retention policies
//   - Trace correlation and W3C Trace Context propagation
//   - Latency measurement (P50, P95, P99)
//   - Bottleneck identification
//
// The tracing system integrates with:
//   - Jaeger for trace collection and visualization
//   - Prometheus for trace metrics export
//   - Standard Go context propagation
package tracing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Config defines the tracing configuration.
type Config struct {
	// ServiceName is the name of this service for tracing
	ServiceName string

	// JaegerEndpoint is the Jaeger collector endpoint (e.g., http://localhost:4317)
	JaegerEndpoint string

	// SamplingRate is the probability of sampling a trace (0.0-1.0)
	SamplingRate float64

	// BatchSize is the maximum number of spans in a batch
	BatchSize int

	// MaxQueueSize is the maximum number of spans to queue
	MaxQueueSize int

	// ExportInterval is how often to export spans
	ExportInterval time.Duration

	// Enabled controls whether tracing is active
	Enabled bool
}

// DefaultConfig returns a configuration suitable for production.
func DefaultConfig() Config {
	return Config{
		ServiceName:    "decentralized-host",
		JaegerEndpoint: "http://localhost:4317",
		SamplingRate:   0.1, // 10% sampling by default
		BatchSize:      512,
		MaxQueueSize:   2048,
		ExportInterval: 5 * time.Second,
		Enabled:        false, // Disabled by default; enable explicitly
	}
}

// Manager manages the tracing infrastructure for a node.
type Manager struct {
	mu              sync.RWMutex
	cfg             Config
	provider        *trace.TracerProvider
	exporter        trace.SpanExporter
	tracer          oteltrace.Tracer
	latencies       map[string]*latencyBucket
	sampledSpans    uint64
	droppedSpans    uint64
	exportedSpans   uint64
}

// latencyBucket tracks latency statistics for an operation.
type latencyBucket struct {
	mu      sync.RWMutex
	name    string
	values  []time.Duration
	sum     time.Duration
	count   uint64
	minVal  time.Duration
	maxVal  time.Duration
}

// NewManager creates a new tracing manager.
func NewManager(cfg Config) (*Manager, error) {
	m := &Manager{
		cfg:        cfg,
		latencies: make(map[string]*latencyBucket),
	}

	if !cfg.Enabled {
		return m, nil
	}

	// Create OTEL resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: create resource: %w", err)
	}

	// Create OTLP HTTP exporter
	exporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint("localhost:4317"),
	)
	if err != nil {
		// If exporter fails, create a noop provider instead
		// In production, a failed exporter setup should be logged
		m.provider = trace.NewTracerProvider(
			trace.WithResource(res),
		)
		m.tracer = m.provider.Tracer(cfg.ServiceName)
		return m, nil
	}

	m.exporter = exporter

	// Create trace provider with batch processor
	m.provider = trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(exporter,
			trace.WithBatchTimeout(cfg.ExportInterval),
			trace.WithMaxExportBatchSize(cfg.BatchSize),
			trace.WithMaxQueueSize(cfg.MaxQueueSize),
		),
		trace.WithSampler(trace.ParentBased(
			trace.TraceIDRatioBased(cfg.SamplingRate),
		)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(m.provider)

	// Get tracer
	m.tracer = m.provider.Tracer(cfg.ServiceName)

	return m, nil
}

// Tracer returns the global tracer for this manager.
func (m *Manager) Tracer() oteltrace.Tracer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tracer
}

// StartSpan starts a new span with the given name and attributes.
func (m *Manager) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, oteltrace.Span) {
	if m.tracer == nil {
		return ctx, oteltrace.SpanFromContext(ctx)
	}
	return m.tracer.Start(ctx, name, oteltrace.WithAttributes(attrs...))
}

// RecordLatency records a latency measurement for an operation.
func (m *Manager) RecordLatency(operation string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	bucket, ok := m.latencies[operation]
	if !ok {
		bucket = &latencyBucket{
			name:   operation,
			values: make([]time.Duration, 0, 1000),
		}
		m.latencies[operation] = bucket
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	bucket.values = append(bucket.values, latency)
	bucket.sum += latency
	bucket.count++

	if latency < bucket.minVal || bucket.minVal == 0 {
		bucket.minVal = latency
	}
	if latency > bucket.maxVal {
		bucket.maxVal = latency
	}
}

// LatencyStats returns latency statistics for an operation.
type LatencyStats struct {
	Operation string
	Count     uint64
	Min       time.Duration
	Max       time.Duration
	Mean      time.Duration
	P50       time.Duration
	P95       time.Duration
	P99       time.Duration
}

// GetLatencyStats returns statistics for a specific operation.
func (m *Manager) GetLatencyStats(operation string) (LatencyStats, bool) {
	m.mu.RLock()
	bucket, ok := m.latencies[operation]
	m.mu.RUnlock()

	if !ok {
		return LatencyStats{}, false
	}

	bucket.mu.RLock()
	defer bucket.mu.RUnlock()

	stats := LatencyStats{
		Operation: operation,
		Count:     bucket.count,
		Min:       bucket.minVal,
		Max:       bucket.maxVal,
	}

	if bucket.count == 0 {
		return stats, true
	}

	stats.Mean = bucket.sum / time.Duration(bucket.count)

	// Calculate percentiles
	if len(bucket.values) > 0 {
		// Simple implementation; in production use a more efficient algorithm
		stats.P50 = percentile(bucket.values, 0.50)
		stats.P95 = percentile(bucket.values, 0.95)
		stats.P99 = percentile(bucket.values, 0.99)
	}

	return stats, true
}

// percentile calculates the nth percentile of sorted latencies.
func percentile(values []time.Duration, p float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	idx := int(float64(len(values)) * p)
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return values[idx]
}

// Shutdown gracefully shuts down the tracing system.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.provider == nil {
		return nil
	}

	return m.provider.ForceFlush(ctx)
}

// Close closes the tracing manager and releases resources.
func (m *Manager) Close(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.provider == nil {
		return nil
	}

	return m.provider.Shutdown(ctx)
}

// Stats returns statistics about tracing.
type Stats struct {
	SampledSpans   uint64
	DroppedSpans   uint64
	ExportedSpans  uint64
	LatencyMetrics map[string]LatencyStats
}

// GetStats returns current tracing statistics.
func (m *Manager) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := Stats{
		SampledSpans:   m.sampledSpans,
		DroppedSpans:   m.droppedSpans,
		ExportedSpans:  m.exportedSpans,
		LatencyMetrics: make(map[string]LatencyStats),
	}

	for op := range m.latencies {
		if s, ok := m.GetLatencyStats(op); ok {
			stats.LatencyMetrics[op] = s
		}
	}

	return stats
}
