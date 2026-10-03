// Package analytics implements usage analytics and reporting for the marketplace.
//
// Features include:
// - Usage data collection and aggregation
// - Cost analysis per operator/tenant
// - Revenue tracking
// - Capacity planning analytics
// - Operator profitability analysis
package analytics

import (
	"context"
	"sort"
	"sync"
	"time"
)

// MetricType represents a measurable quantity.
type MetricType string

const (
	MetricCPUHours        MetricType = "cpu_hours"
	MetricMemoryGBHours   MetricType = "memory_gb_hours"
	MetricStorageGBMonths MetricType = "storage_gb_months"
	MetricBandwidthGB     MetricType = "bandwidth_gb"
	MetricLeaseCount      MetricType = "lease_count"
	MetricRevenueUSD      MetricType = "revenue_usd"
	MetricCostUSD         MetricType = "cost_usd"
)

// UsageMetric captures resource consumption.
type UsageMetric struct {
	Timestamp    time.Time
	EntityID     string // tenant or operator ID
	EntityType   string // "tenant" or "operator"
	MetricType   MetricType
	Value        float64
	Unit         string
	ClusterID    string
	Region       string
}

// AggregateMetric represents aggregated usage data.
type AggregateMetric struct {
	EntityID     string
	EntityType   string
	MetricType   MetricType
	StartTime    time.Time
	EndTime      time.Time
	Count        int64
	TotalValue   float64
	AvgValue     float64
	MinValue     float64
	MaxValue     float64
}

// CostAnalysis breaks down costs by category.
type CostAnalysis struct {
	EntityID           string
	EntityType         string
	PeriodStart        time.Time
	PeriodEnd          time.Time
	ComputeCost        float64
	MemoryCost         float64
	StorageCost        float64
	BandwidthCost      float64
	TotalCost          float64
	BudgetAllocated    float64
	BudgetRemaining    float64
	BudgetUtilization  float64 // percentage
	TopCostDrivers     []string
}

// RevenueMetrics tracks operator earnings.
type RevenueMetrics struct {
	OperatorID         string
	PeriodStart        time.Time
	PeriodEnd          time.Time
	TotalRevenue       float64
	TotalLeases        int64
	AvgLeaseValue      float64
	TenantCount        int64
	RepeatTenants      int64
	UtilizationAvg     float64 // capacity utilization %
	ProfitMargin       float64 // net profit / revenue
	GrossProfit        float64 // revenue - direct costs
}

// CapacityMetrics tracks resource availability and planning.
type CapacityMetrics struct {
	EntityID              string
	PeriodStart           time.Time
	PeriodEnd            time.Time
	CPUMillicoresTotal    int64
	CPUMillicoresUsed     int64
	CPUUtilization        float64 // percentage
	MemoryGBTotal         float64
	MemoryGBUsed          float64
	MemoryUtilization     float64
	StorageGBTotal        float64
	StorageGBUsed         float64
	StorageUtilization    float64
	BandwidthGBPerDayTotal float64
	BandwidthGBPerDayUsed float64
	ProjectedFullDate     time.Time // when capacity will be full
}

// TenantAnalytics tracks individual tenant behavior.
type TenantAnalytics struct {
	TenantID           string
	FirstLeaseDate     time.Time
	LastLeaseDate      time.Time
	TotalLeases        int64
	TotalSpent         float64
	AvgLeaseValue      float64
	AvgLeaseDuration   time.Duration
	ChurnRisk          float64 // 0-1 score
	Lifetime           time.Duration
	Engagement         string // high, medium, low
	PreferredRegions   []string
	PreferredOperators []string
}

// Collector gathers usage metrics from the system.
type Collector struct {
	mu           sync.RWMutex
	metrics      []*UsageMetric
	maxMetrics   int
	aggregates   map[string]*AggregateMetric
	costAnalysis map[string]*CostAnalysis
	revenueData  map[string]*RevenueMetrics
	capacityData map[string]*CapacityMetrics
	tenantData   map[string]*TenantAnalytics
}

// NewCollector creates a new analytics collector.
func NewCollector() *Collector {
	return &Collector{
		metrics:      []*UsageMetric{},
		maxMetrics:   1000000,
		aggregates:   make(map[string]*AggregateMetric),
		costAnalysis: make(map[string]*CostAnalysis),
		revenueData:  make(map[string]*RevenueMetrics),
		capacityData: make(map[string]*CapacityMetrics),
		tenantData:   make(map[string]*TenantAnalytics),
	}
}

// RecordMetric records a usage event.
func (c *Collector) RecordMetric(ctx context.Context, metric UsageMetric) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	c.metrics = append(c.metrics, &metric)

	// Maintain max size
	if len(c.metrics) > c.maxMetrics {
		c.metrics = c.metrics[len(c.metrics)-c.maxMetrics:]
	}

	return nil
}

// AggregateByEntity aggregates metrics by entity over a time period.
func (c *Collector) AggregateByEntity(ctx context.Context, entityID string, start, end time.Time) []AggregateMetric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Group metrics by type
	grouped := make(map[MetricType][]*UsageMetric)

	for _, m := range c.metrics {
		if m.EntityID == entityID && m.Timestamp.After(start) && m.Timestamp.Before(end) {
			grouped[m.MetricType] = append(grouped[m.MetricType], m)
		}
	}

	var result []AggregateMetric

	for metricType, metrics := range grouped {
		if len(metrics) == 0 {
			continue
		}

		total := 0.0
		min := metrics[0].Value
		max := metrics[0].Value

		for _, m := range metrics {
			total += m.Value
			if m.Value < min {
				min = m.Value
			}
			if m.Value > max {
				max = m.Value
			}
		}

		agg := AggregateMetric{
			EntityID:   entityID,
			MetricType: metricType,
			StartTime:  start,
			EndTime:    end,
			Count:      int64(len(metrics)),
			TotalValue: total,
			AvgValue:   total / float64(len(metrics)),
			MinValue:   min,
			MaxValue:   max,
		}

		result = append(result, agg)
	}

	return result
}

// CalculateCostAnalysis generates cost breakdown for an entity.
func (c *Collector) CalculateCostAnalysis(ctx context.Context, entityID string, entityType string, start, end time.Time) (*CostAnalysis, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	analysis := &CostAnalysis{
		EntityID:       entityID,
		EntityType:     entityType,
		PeriodStart:    start,
		PeriodEnd:      end,
		TopCostDrivers: []string{},
	}

	costByType := make(map[MetricType]float64)

	// Aggregate metrics and calculate costs
	for _, m := range c.metrics {
		if m.EntityID != entityID || m.Timestamp.Before(start) || m.Timestamp.After(end) {
			continue
		}

		// Simple cost model based on metric type
		var cost float64
		switch m.MetricType {
		case MetricCPUHours:
			cost = m.Value * 0.05 // $0.05 per CPU-hour
		case MetricMemoryGBHours:
			cost = m.Value * 0.01 // $0.01 per GB-hour
		case MetricStorageGBMonths:
			cost = m.Value * 0.1 // $0.10 per GB-month
		case MetricBandwidthGB:
			cost = m.Value * 0.01 // $0.01 per GB
		}

		costByType[m.MetricType] += cost
	}

	// Assign to categories
	analysis.ComputeCost = costByType[MetricCPUHours]
	analysis.MemoryCost = costByType[MetricMemoryGBHours]
	analysis.StorageCost = costByType[MetricStorageGBMonths]
	analysis.BandwidthCost = costByType[MetricBandwidthGB]
	analysis.TotalCost = analysis.ComputeCost + analysis.MemoryCost + analysis.StorageCost + analysis.BandwidthCost

	// Identify top cost drivers
	type costItem struct {
		name  string
		value float64
	}
	items := []costItem{
		{"compute", analysis.ComputeCost},
		{"memory", analysis.MemoryCost},
		{"storage", analysis.StorageCost},
		{"bandwidth", analysis.BandwidthCost},
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].value > items[j].value
	})

	for i := 0; i < 3 && i < len(items); i++ {
		if items[i].value > 0 {
			analysis.TopCostDrivers = append(analysis.TopCostDrivers, items[i].name)
		}
	}

	return analysis, nil
}

// CalculateRevenueMetrics generates revenue report for an operator.
func (c *Collector) CalculateRevenueMetrics(ctx context.Context, operatorID string, start, end time.Time) (*RevenueMetrics, error) {
	c.mu.RLock()

	metrics := &RevenueMetrics{
		OperatorID:   operatorID,
		PeriodStart:  start,
		PeriodEnd:    end,
		TotalLeases:  0,
		TenantCount:  0,
	}

	// Count transactions and sum revenue
	revenueSum := 0.0
	leaseCount := int64(0)
	tenants := make(map[string]bool)

	for _, m := range c.metrics {
		if m.EntityID != operatorID || m.Timestamp.Before(start) || m.Timestamp.After(end) {
			continue
		}

		if m.MetricType == MetricRevenueUSD {
			revenueSum += m.Value
		}
		if m.MetricType == MetricLeaseCount {
			leaseCount += int64(m.Value)
		}
	}

	c.mu.RUnlock()

	metrics.TotalRevenue = revenueSum
	metrics.TotalLeases = leaseCount
	if leaseCount > 0 {
		metrics.AvgLeaseValue = revenueSum / float64(leaseCount)
	}
	metrics.TenantCount = int64(len(tenants))

	return metrics, nil
}

// CalculateCapacityMetrics generates capacity report.
func (c *Collector) CalculateCapacityMetrics(ctx context.Context, entityID string, start, end time.Time, totalCPU int64, totalMem float64, totalStorage float64) (*CapacityMetrics, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := &CapacityMetrics{
		EntityID:            entityID,
		PeriodStart:         start,
		PeriodEnd:           end,
		CPUMillicoresTotal:  totalCPU,
		MemoryGBTotal:       totalMem,
		StorageGBTotal:      totalStorage,
	}

	// Aggregate usage
	for _, m := range c.metrics {
		if m.EntityID != entityID || m.Timestamp.Before(start) || m.Timestamp.After(end) {
			continue
		}

		switch m.MetricType {
		case MetricCPUHours:
			// Average CPU usage over period duration
			hours := end.Sub(start).Hours()
			if hours > 0 {
				metrics.CPUMillicoresUsed += int64(m.Value / hours)
			}
		case MetricMemoryGBHours:
			hours := end.Sub(start).Hours()
			if hours > 0 {
				metrics.MemoryGBUsed += m.Value / hours
			}
		case MetricStorageGBMonths:
			days := float64(end.Sub(start).Hours() / 24)
			if days > 0 {
				metrics.StorageGBUsed += (m.Value / 30) * days
			}
		}
	}

	// Calculate utilization percentages
	if metrics.CPUMillicoresTotal > 0 {
		metrics.CPUUtilization = (float64(metrics.CPUMillicoresUsed) / float64(metrics.CPUMillicoresTotal)) * 100
	}
	if metrics.MemoryGBTotal > 0 {
		metrics.MemoryUtilization = (metrics.MemoryGBUsed / metrics.MemoryGBTotal) * 100
	}
	if metrics.StorageGBTotal > 0 {
		metrics.StorageUtilization = (metrics.StorageGBUsed / metrics.StorageGBTotal) * 100
	}

	// Predict when full (simple linear extrapolation)
	if metrics.CPUUtilization > 0 {
		daysToFull := float64(end.Sub(start).Hours()/24) * (100 / metrics.CPUUtilization)
		metrics.ProjectedFullDate = end.Add(time.Duration(daysToFull*24) * time.Hour)
	}

	return metrics, nil
}

// AnalyzeTenant generates behavior report for a tenant.
func (c *Collector) AnalyzeTenant(ctx context.Context, tenantID string) (*TenantAnalytics, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	analytics := &TenantAnalytics{
		TenantID:           tenantID,
		PreferredRegions:   []string{},
		PreferredOperators: []string{},
	}

	var leases int64
	var totalSpent float64
	var durations []time.Duration
	regions := make(map[string]int)

	firstTime := time.Now()
	lastTime := time.Time{}

	for _, m := range c.metrics {
		if m.EntityID != tenantID {
			continue
		}

		if m.Timestamp.Before(firstTime) {
			firstTime = m.Timestamp
		}
		if m.Timestamp.After(lastTime) {
			lastTime = m.Timestamp
		}

		if m.MetricType == MetricLeaseCount {
			leases += int64(m.Value)
		}
		if m.MetricType == MetricRevenueUSD {
			totalSpent += m.Value
		}

		if m.Region != "" {
			regions[m.Region]++
		}
	}

	analytics.FirstLeaseDate = firstTime
	analytics.LastLeaseDate = lastTime
	analytics.TotalLeases = leases
	analytics.TotalSpent = totalSpent
	analytics.Lifetime = lastTime.Sub(firstTime)

	if leases > 0 {
		analytics.AvgLeaseValue = totalSpent / float64(leases)
		if len(durations) > 0 {
			totalDuration := time.Duration(0)
			for _, d := range durations {
				totalDuration += d
			}
			analytics.AvgLeaseDuration = time.Duration(int64(totalDuration) / int64(len(durations)))
		}
	}

	// Determine engagement level
	if analytics.TotalLeases > 10 {
		analytics.Engagement = "high"
	} else if analytics.TotalLeases > 3 {
		analytics.Engagement = "medium"
	} else {
		analytics.Engagement = "low"
	}

	// Extract top regions
	for region, count := range regions {
		if count > 0 {
			analytics.PreferredRegions = append(analytics.PreferredRegions, region)
		}
	}
	sort.SliceStable(analytics.PreferredRegions, func(i, j int) bool {
		return regions[analytics.PreferredRegions[i]] > regions[analytics.PreferredRegions[j]]
	})
	if len(analytics.PreferredRegions) > 3 {
		analytics.PreferredRegions = analytics.PreferredRegions[:3]
	}

	return analytics, nil
}

// GetMetricsTimeSeries returns metrics over time for visualization.
func (c *Collector) GetMetricsTimeSeries(ctx context.Context, entityID string, metricType MetricType, start, end time.Time, bucketSize time.Duration) map[time.Time]float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[time.Time]float64)

	if bucketSize == 0 {
		bucketSize = time.Hour
	}

	for ts := start; ts.Before(end); ts = ts.Add(bucketSize) {
		result[ts] = 0
	}

	for _, m := range c.metrics {
		if m.EntityID != entityID || m.MetricType != metricType {
			continue
		}
		if m.Timestamp.Before(start) || m.Timestamp.After(end) {
			continue
		}

		bucket := m.Timestamp.Truncate(bucketSize)
		result[bucket] += m.Value
	}

	return result
}
