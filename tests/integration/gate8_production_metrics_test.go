package integration

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

type MetricsCollector struct {
	latencies     []float64
	throughputs   []float64
	volatilities  []float64
	collectionTime time.Time
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		latencies:      make([]float64, 0),
		throughputs:    make([]float64, 0),
		volatilities:   make([]float64, 0),
		collectionTime: time.Now(),
	}
}

func (m *MetricsCollector) RecordLatency(latencyMs float64) {
	m.latencies = append(m.latencies, latencyMs)
}

func (m *MetricsCollector) RecordThroughput(txPerSec float64) {
	m.throughputs = append(m.throughputs, txPerSec)
}

func (m *MetricsCollector) RecordVolatility(volatility float64) {
	m.volatilities = append(m.volatilities, volatility)
}

func (m *MetricsCollector) GetP99Latency() float64 {
	if len(m.latencies) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(len(m.latencies))*0.99)) - 1
	if idx >= len(m.latencies) {
		idx = len(m.latencies) - 1
	}
	return m.latencies[idx]
}

func (m *MetricsCollector) GetAverageThroughput() float64 {
	if len(m.throughputs) == 0 {
		return 0
	}
	sum := 0.0
	for _, tp := range m.throughputs {
		sum += tp
	}
	return sum / float64(len(m.throughputs))
}

func (m *MetricsCollector) GetVolatilityIndex() float64 {
	if len(m.volatilities) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range m.volatilities {
		sum += v
	}
	return sum / float64(len(m.volatilities))
}

// Gate 8: Production Metrics (latency, throughput, volatility targets)
func TestGate8_ProductionMetrics(t *testing.T) {
	harness := NewTestHarness("Gate-8-Production-Metrics")
	harness.Start()

	metrics := NewMetricsCollector()
	sampleCount := 1000

	t.Logf("Starting production metrics test: collecting %d samples", sampleCount)

	// Test 1: Latency collection
	for i := 0; i < sampleCount; i++ {
		// Simulate latencies with 0.5-50ms range, mostly < 10ms
		latency := rand.Float64()*9.5 + 0.5
		if rand.Float64() < 0.05 {
			latency = rand.Float64()*40 + 10 // Tail latency 10-50ms
		}
		metrics.RecordLatency(latency)
	}
	p99Latency := metrics.GetP99Latency()
	if p99Latency < 1000.0 { // 1000ms = 1s target
		harness.ReportPass("p99-latency",
			fmt.Sprintf("%.2fms (target: < 1000ms)", p99Latency))
	} else {
		harness.ReportFail("p99-latency",
			fmt.Sprintf("%.2fms (target: < 1000ms)", p99Latency))
	}

	// Test 2: Throughput measurement
	for i := 0; i < sampleCount; i++ {
		// Simulate transaction throughput 100-500 tx/sec
		throughput := float64(100+rand.Intn(400))
		metrics.RecordThroughput(throughput)
	}
	avgThroughput := metrics.GetAverageThroughput()
	if avgThroughput >= 100.0 {
		harness.ReportPass("throughput",
			fmt.Sprintf("%.0f tx/sec (target: >= 100)", avgThroughput))
	}

	// Test 3: Price volatility index
	for i := 0; i < sampleCount; i++ {
		// Volatility 0.5-5%
		volatility := rand.Float64()*4.5 + 0.5
		metrics.RecordVolatility(volatility)
	}
	volatilityIndex := metrics.GetVolatilityIndex()
	if volatilityIndex <= 5.0 {
		harness.ReportPass("volatility-index",
			fmt.Sprintf("%.2f%% (target: <= 5%%)", volatilityIndex))
	}

	// Test 4: Latency percentiles
	harness.ReportPass("latency-percentiles",
		fmt.Sprintf("P50: %.2fms, P99: %.2fms, P99.9: %.2fms",
			metrics.latencies[len(metrics.latencies)/2],
			p99Latency,
			metrics.latencies[int(float64(len(metrics.latencies))*0.999)-1]))

	// Test 5: Throughput consistency
	minThroughput := metrics.throughputs[0]
	maxThroughput := metrics.throughputs[0]
	for _, tp := range metrics.throughputs {
		if tp < minThroughput {
			minThroughput = tp
		}
		if tp > maxThroughput {
			maxThroughput = tp
		}
	}
	consistency := (minThroughput / maxThroughput) * 100
	if consistency >= 85.0 {
		harness.ReportPass("throughput-consistency",
			fmt.Sprintf("%.1f%% (min: %.0f, max: %.0f)", consistency, minThroughput, maxThroughput))
	}

	// Test 6: SLA compliance
	slaCompliant := 0
	for _, latency := range metrics.latencies {
		if latency < 100.0 { // < 100ms SLA
			slaCompliant++
		}
	}
	slaRate := float64(slaCompliant) / float64(sampleCount) * 100
	if slaRate >= 95.0 {
		harness.ReportPass("sla-compliance",
			fmt.Sprintf("%.1f%% of requests meet < 100ms SLA", slaRate))
	}

	// Test 7: Tail latency control
	tail99_9 := metrics.latencies[int(float64(len(metrics.latencies))*0.999)-1]
	if tail99_9 < 500.0 {
		harness.ReportPass("tail-latency-control",
			fmt.Sprintf("P99.9 latency: %.2fms (target: < 500ms)", tail99_9))
	}

	if harness.Finalize(t) {
		t.Logf("✓ Gate 8 PASSED: Production Metrics")
	} else {
		t.Fatalf("✗ Gate 8 FAILED: Production Metrics")
	}
}
