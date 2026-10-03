// Package sla implements SLA compliance tracking and enforcement
// for the Decentralized.Host operator qualification program.
//
// SLA compliance is tracked against 95%+ uptime targets and P99 latency
// < 1s. Violations trigger automatic penalties and alerting.
package sla

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	// UptimeTarget is the required uptime percentage
	UptimeTarget = 95.0

	// P99LatencyTarget is the maximum acceptable P99 latency
	P99LatencyTarget = 1 * time.Second

	// MetricWindowSize is the size of the metrics window (1 hour)
	MetricWindowSize = 1 * time.Hour

	// AlertThreshold triggers alerts (percentage of UptimeTarget)
	AlertThreshold = 0.90 * UptimeTarget // 85.5%
)

// ComplianceStatus represents SLA compliance state
type ComplianceStatus string

const (
	COMPLIANT   ComplianceStatus = "COMPLIANT"
	AT_RISK     ComplianceStatus = "AT_RISK"
	VIOLATED    ComplianceStatus = "VIOLATED"
	UNKNOWN     ComplianceStatus = "UNKNOWN"
)

// LatencyBucket represents latency measurement data
type LatencyBucket struct {
	Timestamp   time.Time
	Count       int64
	MinLatency  time.Duration
	MaxLatency  time.Duration
	AvgLatency  time.Duration
	P50Latency  time.Duration // Median
	P95Latency  time.Duration
	P99Latency  time.Duration
}

// UptimeWindow tracks uptime within a time window
type UptimeWindow struct {
	StartTime      time.Time
	EndTime        time.Time
	TotalRequests  int64
	SuccessCount   int64
	FailureCount   int64
	UptimePercent  float64
}

// SLAMetrics tracks SLA metrics for an operator
type SLAMetrics struct {
	OperatorID              string                     `json:"operator_id"`
	Status                  ComplianceStatus           `json:"status"`
	CurrentUptime           float64                    `json:"current_uptime"` // percentage
	P99Latency              time.Duration              `json:"p99_latency"`
	P95Latency              time.Duration              `json:"p95_latency"`
	AverageLatency          time.Duration              `json:"average_latency"`
	LastMeasurementTime     time.Time                  `json:"last_measurement_time"`
	ViolationCount          int                        `json:"violation_count"`
	LastViolationTime       *time.Time                 `json:"last_violation_time,omitempty"`
	AlertLevel              string                     `json:"alert_level"` // NONE, WARNING, CRITICAL
	ConsecutiveFailures     int                        `json:"consecutive_failures"`
	LastMTBF                time.Duration              `json:"last_mtbf"` // Mean Time Between Failures
	TrendingUptime          bool                       `json:"trending_uptime"` // improving or degrading
	HistoricalCompliance    map[string]float64         `json:"historical_compliance"` // Date -> uptime%
	LatencyHistory          []LatencyBucket            `json:"latency_history"`
}

// SLAManager manages SLA compliance tracking
type SLAManager struct {
	mu      sync.RWMutex
	metrics map[string]*SLAMetrics
}

// NewSLAManager creates a new SLA manager
func NewSLAManager() *SLAManager {
	return &SLAManager{
		metrics: make(map[string]*SLAMetrics),
	}
}

// RegisterOperator registers an operator for SLA tracking
func (sm *SLAManager) RegisterOperator(operatorID string) (*SLAMetrics, error) {
	if operatorID == "" {
		return nil, errors.New("operator ID cannot be empty")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.metrics[operatorID]; exists {
		return nil, fmt.Errorf("operator %s already registered", operatorID)
	}

	now := time.Now()
	metrics := &SLAMetrics{
		OperatorID:          operatorID,
		Status:              UNKNOWN,
		CurrentUptime:       100.0,
		P99Latency:          0,
		P95Latency:          0,
		AverageLatency:      0,
		LastMeasurementTime: now,
		ViolationCount:      0,
		AlertLevel:          "NONE",
		ConsecutiveFailures: 0,
		TrendingUptime:      true,
		HistoricalCompliance: make(map[string]float64),
		LatencyHistory:      []LatencyBucket{},
	}

	sm.metrics[operatorID] = metrics
	return metrics, nil
}

// GetMetrics retrieves SLA metrics for an operator
func (sm *SLAManager) GetMetrics(operatorID string) (*SLAMetrics, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	metrics, exists := sm.metrics[operatorID]
	if !exists {
		return nil, fmt.Errorf("operator %s not found", operatorID)
	}
	return metrics, nil
}

// RecordMetrics updates SLA metrics based on observed performance
func (sm *SLAManager) RecordMetrics(operatorID string, successCount int64, failureCount int64, latencies []time.Duration) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	metrics, exists := sm.metrics[operatorID]
	if !exists {
		return fmt.Errorf("operator %s not found", operatorID)
	}

	now := time.Now()
	totalCount := successCount + failureCount

	if totalCount == 0 {
		return errors.New("no metrics to record")
	}

	// Update uptime
	oldUptime := metrics.CurrentUptime
	metrics.CurrentUptime = float64(successCount) / float64(totalCount) * 100.0

	// Track uptime trending
	if metrics.CurrentUptime > oldUptime {
		metrics.TrendingUptime = true
	} else {
		metrics.TrendingUptime = false
	}

	// Track consecutive failures
	if failureCount > 0 {
		metrics.ConsecutiveFailures++
	} else {
		metrics.ConsecutiveFailures = 0
	}

	// Record historical compliance
	dateKey := now.Format("2006-01-02")
	metrics.HistoricalCompliance[dateKey] = metrics.CurrentUptime

	// Update latency metrics
	if len(latencies) > 0 {
		bucket := calculateLatencyMetrics(now, latencies)
		metrics.LatencyHistory = append(metrics.LatencyHistory, bucket)

		// Keep only last 24 hours of history
		cutoff := now.Add(-24 * time.Hour)
		var filtered []LatencyBucket
		for _, b := range metrics.LatencyHistory {
			if b.Timestamp.After(cutoff) {
				filtered = append(filtered, b)
			}
		}
		metrics.LatencyHistory = filtered

		metrics.P99Latency = bucket.P99Latency
		metrics.P95Latency = bucket.P95Latency
		metrics.AverageLatency = bucket.AvgLatency
	}

	metrics.LastMeasurementTime = now

	// Update compliance status
	sm.updateComplianceStatus(metrics)

	return nil
}

// updateComplianceStatus evaluates SLA compliance
func (sm *SLAManager) updateComplianceStatus(metrics *SLAMetrics) {
	// Check uptime compliance
	uptimeOK := metrics.CurrentUptime >= UptimeTarget
	latencyOK := metrics.P99Latency <= P99LatencyTarget || metrics.P99Latency == 0

	// Determine overall status and alert level
	if uptimeOK && latencyOK {
		metrics.Status = COMPLIANT
		metrics.AlertLevel = "NONE"
	} else if metrics.CurrentUptime >= AlertThreshold && metrics.P99Latency <= P99LatencyTarget*2 {
		metrics.Status = AT_RISK
		metrics.AlertLevel = "WARNING"
	} else {
		metrics.Status = VIOLATED
		metrics.AlertLevel = "CRITICAL"
		metrics.ViolationCount++

		now := time.Now()
		metrics.LastViolationTime = &now
	}
}

// calculateLatencyMetrics computes latency statistics
func calculateLatencyMetrics(timestamp time.Time, latencies []time.Duration) LatencyBucket {
	bucket := LatencyBucket{
		Timestamp: timestamp,
		Count:     int64(len(latencies)),
	}

	if len(latencies) == 0 {
		return bucket
	}

	// Sort latencies for percentile calculation (simplified)
	// In production, use a proper sorting algorithm
	bucket.MinLatency = latencies[0]
	bucket.MaxLatency = latencies[0]

	var total time.Duration
	for _, lat := range latencies {
		total += lat
		if lat < bucket.MinLatency {
			bucket.MinLatency = lat
		}
		if lat > bucket.MaxLatency {
			bucket.MaxLatency = lat
		}
	}

	bucket.AvgLatency = total / time.Duration(len(latencies))

	// Simple percentile calculation (in production, use exact method)
	sortedLatencies := make([]time.Duration, len(latencies))
	copy(sortedLatencies, latencies)

	// Bubble sort for small sample sizes (not production-grade)
	for i := 0; i < len(sortedLatencies); i++ {
		for j := i + 1; j < len(sortedLatencies); j++ {
			if sortedLatencies[j] < sortedLatencies[i] {
				sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
			}
		}
	}

	// P50 (median)
	p50Idx := len(sortedLatencies) / 2
	bucket.P50Latency = sortedLatencies[p50Idx]

	// P95
	p95Idx := (len(sortedLatencies) * 95) / 100
	if p95Idx < len(sortedLatencies) {
		bucket.P95Latency = sortedLatencies[p95Idx]
	}

	// P99
	p99Idx := (len(sortedLatencies) * 99) / 100
	if p99Idx < len(sortedLatencies) {
		bucket.P99Latency = sortedLatencies[p99Idx]
	}

	return bucket
}

// GetComplianceReport generates a compliance report
type ComplianceReport struct {
	OperatorID        string
	Status            ComplianceStatus
	UptimePercentage  float64
	UptimeTarget      float64
	P99Latency        time.Duration
	LatencyTarget     time.Duration
	ViolationCount    int
	AlertLevel        string
	CompliancePeriod  string
	RecommendedAction string
}

// GenerateReport creates a compliance report for an operator
func (sm *SLAManager) GenerateReport(operatorID string) (*ComplianceReport, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	metrics, exists := sm.metrics[operatorID]
	if !exists {
		return nil, fmt.Errorf("operator %s not found", operatorID)
	}

	report := &ComplianceReport{
		OperatorID:       operatorID,
		Status:           metrics.Status,
		UptimePercentage: metrics.CurrentUptime,
		UptimeTarget:     UptimeTarget,
		P99Latency:       metrics.P99Latency,
		LatencyTarget:    P99LatencyTarget,
		ViolationCount:   metrics.ViolationCount,
		AlertLevel:       metrics.AlertLevel,
		CompliancePeriod: "Last 24 hours",
	}

	// Generate recommendation
	switch metrics.Status {
	case COMPLIANT:
		report.RecommendedAction = "Continue monitoring. No action required."
	case AT_RISK:
		report.RecommendedAction = "Investigate performance degradation. Increase monitoring frequency."
	case VIOLATED:
		report.RecommendedAction = "Critical: Immediate action required. Review infrastructure and implementation. SLA slashing penalty will be applied."
	default:
		report.RecommendedAction = "Insufficient data. Collect more metrics."
	}

	return report, nil
}

// ListMetrics returns all operator metrics
func (sm *SLAManager) ListMetrics() []*SLAMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	metrics := make([]*SLAMetrics, 0, len(sm.metrics))
	for _, m := range sm.metrics {
		metrics = append(metrics, m)
	}
	return metrics
}

// SLAStatusSummary provides aggregate SLA compliance information
type SLAStatusSummary struct {
	TotalOperators        int
	CompliantOperators    int
	AtRiskOperators       int
	ViolatedOperators     int
	AverageUptime         float64
	AverageP99Latency     time.Duration
	CriticalAlertCount    int
	LastUpdateTime        time.Time
}

// GetStatusSummary returns overall SLA compliance statistics
func (sm *SLAManager) GetStatusSummary() SLAStatusSummary {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	summary := SLAStatusSummary{
		TotalOperators: len(sm.metrics),
		LastUpdateTime: time.Now(),
	}

	var totalUptime float64
	var totalLatency time.Duration

	for _, m := range sm.metrics {
		totalUptime += m.CurrentUptime

		switch m.Status {
		case COMPLIANT:
			summary.CompliantOperators++
		case AT_RISK:
			summary.AtRiskOperators++
		case VIOLATED:
			summary.ViolatedOperators++
		}

		if m.AlertLevel == "CRITICAL" {
			summary.CriticalAlertCount++
		}

		if m.P99Latency > 0 {
			totalLatency += m.P99Latency
		}
	}

	if summary.TotalOperators > 0 {
		summary.AverageUptime = totalUptime / float64(summary.TotalOperators)
		summary.AverageP99Latency = totalLatency / time.Duration(summary.TotalOperators)
	}

	return summary
}

// ResetMetricsWindow clears metrics for a new measurement window
func (sm *SLAManager) ResetMetricsWindow(operatorID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	metrics, exists := sm.metrics[operatorID]
	if !exists {
		return fmt.Errorf("operator %s not found", operatorID)
	}

	// Keep historical data but reset current measurements
	metrics.CurrentUptime = 100.0
	metrics.ConsecutiveFailures = 0
	metrics.LastMeasurementTime = time.Now()

	return nil
}
