// Package metrics implements Prometheus metrics for Decentralized.Host.
//
// This package provides metrics collection and export for:
//   - Placement latency and throughput
//   - Work success rate and failure analysis
//   - Resource utilization (CPU, memory, network)
//   - Custom business logic metrics
//   - SLA tracking and reporting
//
// Metrics are exported in Prometheus format and can be scraped by:
//   - Prometheus server
//   - Grafana dashboards
//   - Alertmanager for threshold-based alerts
package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config defines metrics configuration.
type Config struct {
	// Enabled controls whether metrics are active
	Enabled bool

	// ListenAddress is the HTTP address for metrics export
	ListenAddress string

	// MetricsPath is the HTTP path for /metrics endpoint
	MetricsPath string

	// HistogramBuckets defines histogram bucket boundaries
	HistogramBuckets []float64

	// RetentionDays is how long to retain metric data
	RetentionDays int
}

// DefaultConfig returns a configuration suitable for production.
func DefaultConfig() Config {
	return Config{
		Enabled:       true,
		ListenAddress: "0.0.0.0:9090",
		MetricsPath:   "/metrics",
		HistogramBuckets: []float64{
			0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
		},
		RetentionDays: 7,
	}
}

// Manager manages Prometheus metrics for a node.
type Manager struct {
	mu      sync.RWMutex
	cfg     Config
	server  *http.Server
	active  bool

	// Core metrics
	PlacementLatency prometheus.Histogram
	ThroughputRate   prometheus.Gauge
	SuccessRate      prometheus.Gauge
	FailureRate      prometheus.Gauge

	// Resource metrics
	CPUUsage    prometheus.Gauge
	MemoryUsage prometheus.Gauge
	DiskUsage   prometheus.Gauge
	NetworkIn   prometheus.Counter
	NetworkOut  prometheus.Counter

	// Policy metrics
	PolicyMatches  prometheus.Counter
	PolicyDenials  prometheus.Counter
	PolicyLatency  prometheus.Histogram

	// Trace metrics
	TracesReceived prometheus.Counter
	TracesExported prometheus.Counter
	TraceErrors    prometheus.Counter

	// SLA metrics
	SLATargetUptime prometheus.Gauge
	SLAActualUptime prometheus.Gauge
	SLAViolations   prometheus.Counter
}

// NewManager creates a new metrics manager.
func NewManager(cfg Config) (*Manager, error) {
	m := &Manager{
		cfg:    cfg,
		active: false,
	}

	if !cfg.Enabled {
		return m, nil
	}

	// Create metrics
	m.PlacementLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "dh_placement_latency_seconds",
		Help:    "Placement operation latency in seconds",
		Buckets: cfg.HistogramBuckets,
	})

	m.ThroughputRate = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dh_throughput_ops_per_second",
		Help: "Current throughput in operations per second",
	})

	m.SuccessRate = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dh_success_rate",
		Help: "Success rate as a percentage (0-100)",
	})

	m.FailureRate = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dh_failure_rate",
		Help: "Failure rate as a percentage (0-100)",
	})

	m.CPUUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dh_cpu_usage_cores",
		Help: "CPU usage in cores",
	})

	m.MemoryUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dh_memory_usage_bytes",
		Help: "Memory usage in bytes",
	})

	m.DiskUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dh_disk_usage_bytes",
		Help: "Disk usage in bytes",
	})

	m.NetworkIn = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dh_network_in_bytes_total",
		Help: "Total bytes received over network",
	})

	m.NetworkOut = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dh_network_out_bytes_total",
		Help: "Total bytes sent over network",
	})

	m.PolicyMatches = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dh_policy_matches_total",
		Help: "Total policy matches",
	})

	m.PolicyDenials = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dh_policy_denials_total",
		Help: "Total policy denials",
	})

	m.PolicyLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "dh_policy_latency_seconds",
		Help:    "Policy evaluation latency in seconds",
		Buckets: cfg.HistogramBuckets,
	})

	m.TracesReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dh_traces_received_total",
		Help: "Total traces received",
	})

	m.TracesExported = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dh_traces_exported_total",
		Help: "Total traces exported",
	})

	m.TraceErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dh_trace_errors_total",
		Help: "Total trace errors",
	})

	m.SLATargetUptime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dh_sla_target_uptime",
		Help: "Target SLA uptime as a percentage (0-100)",
	})

	m.SLAActualUptime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dh_sla_actual_uptime",
		Help: "Actual SLA uptime as a percentage (0-100)",
	})

	m.SLAViolations = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dh_sla_violations_total",
		Help: "Total SLA violations",
	})

	// Register all metrics
	for _, metric := range []prometheus.Collector{
		m.PlacementLatency,
		m.ThroughputRate,
		m.SuccessRate,
		m.FailureRate,
		m.CPUUsage,
		m.MemoryUsage,
		m.DiskUsage,
		m.NetworkIn,
		m.NetworkOut,
		m.PolicyMatches,
		m.PolicyDenials,
		m.PolicyLatency,
		m.TracesReceived,
		m.TracesExported,
		m.TraceErrors,
		m.SLATargetUptime,
		m.SLAActualUptime,
		m.SLAViolations,
	} {
		if err := prometheus.Register(metric); err != nil {
			// Already registered (e.g., in tests)
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				return nil, fmt.Errorf("metrics: register: %w", err)
			}
		}
	}

	m.active = true
	return m, nil
}

// Start starts the HTTP metrics server.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.active {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle(m.cfg.MetricsPath, promhttp.Handler())

	m.server = &http.Server{
		Addr:    m.cfg.ListenAddress,
		Handler: mux,
	}

	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("metrics: server error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the metrics HTTP server.
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.server == nil {
		return nil
	}

	return m.server.Shutdown(ctx)
}

// RecordPlacementLatency records a placement operation latency.
func (m *Manager) RecordPlacementLatency(duration time.Duration) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.active {
		return
	}

	m.PlacementLatency.Observe(duration.Seconds())
}

// RecordPolicyLatency records a policy evaluation latency.
func (m *Manager) RecordPolicyLatency(duration time.Duration) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.active {
		return
	}

	m.PolicyLatency.Observe(duration.Seconds())
}

// IncrementPolicyMatches increments the policy match counter.
func (m *Manager) IncrementPolicyMatches() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.active {
		return
	}

	m.PolicyMatches.Inc()
}

// IncrementPolicyDenials increments the policy denial counter.
func (m *Manager) IncrementPolicyDenials() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.active {
		return
	}

	m.PolicyDenials.Inc()
}

// UpdateResourceUsage updates resource utilization metrics.
func (m *Manager) UpdateResourceUsage(cpu, memory, disk float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.active {
		return
	}

	m.CPUUsage.Set(cpu)
	m.MemoryUsage.Set(memory)
	m.DiskUsage.Set(disk)
}

// AddNetworkBytes adds bytes to network counters.
func (m *Manager) AddNetworkBytes(in, out uint64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.active {
		return
	}

	m.NetworkIn.Add(float64(in))
	m.NetworkOut.Add(float64(out))
}

// UpdateSLAMetrics updates SLA-related metrics.
func (m *Manager) UpdateSLAMetrics(targetUptime, actualUptime float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.active {
		return
	}

	m.SLATargetUptime.Set(targetUptime)
	m.SLAActualUptime.Set(actualUptime)
}

// IncrementSLAViolations increments the SLA violation counter.
func (m *Manager) IncrementSLAViolations() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.active {
		return
	}

	m.SLAViolations.Inc()
}

// RecordTrace records a trace reception.
func (m *Manager) RecordTrace(success bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.active {
		return
	}

	m.TracesReceived.Inc()
	if success {
		m.TracesExported.Inc()
	} else {
		m.TraceErrors.Inc()
	}
}

// UpdateThroughput updates the throughput metric.
func (m *Manager) UpdateThroughput(opsPerSecond float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.active {
		return
	}

	m.ThroughputRate.Set(opsPerSecond)
}

// UpdateSuccessRate updates the success rate metric.
func (m *Manager) UpdateSuccessRate(percentage float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.active {
		return
	}

	m.SuccessRate.Set(percentage)
	m.FailureRate.Set(100.0 - percentage)
}
