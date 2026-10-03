package analytics

import (
	"context"
	"testing"
	"time"
)

func TestRecordMetric(t *testing.T) {
	collector := NewCollector()
	ctx := context.Background()

	metric := UsageMetric{
		Timestamp:  time.Now(),
		EntityID:   "tenant-1",
		EntityType: "tenant",
		MetricType: MetricCPUHours,
		Value:      100.0,
		Unit:       "hours",
		ClusterID:  "cluster-1",
		Region:     "us",
	}

	err := collector.RecordMetric(ctx, metric)
	if err != nil {
		t.Fatalf("RecordMetric failed: %v", err)
	}
}

func TestAggregateByEntity(t *testing.T) {
	collector := NewCollector()
	ctx := context.Background()

	now := time.Now()

	// Record multiple metrics
	metrics := []UsageMetric{
		{Timestamp: now, EntityID: "tenant-1", MetricType: MetricCPUHours, Value: 10},
		{Timestamp: now.Add(1 * time.Hour), EntityID: "tenant-1", MetricType: MetricCPUHours, Value: 20},
		{Timestamp: now.Add(2 * time.Hour), EntityID: "tenant-1", MetricType: MetricCPUHours, Value: 30},
		{Timestamp: now, EntityID: "tenant-1", MetricType: MetricMemoryGBHours, Value: 50},
	}

	for _, m := range metrics {
		collector.RecordMetric(ctx, m)
	}

	// Aggregate
	start := now.Add(-1 * time.Hour)
	end := now.Add(3 * time.Hour)

	aggs := collector.AggregateByEntity(ctx, "tenant-1", start, end)

	if len(aggs) != 2 {
		t.Errorf("Expected 2 aggregates, got %d", len(aggs))
	}

	// Find CPU aggregate
	var cpuAgg *AggregateMetric
	var memAgg *AggregateMetric
	for i := range aggs {
		if aggs[i].MetricType == MetricCPUHours {
			cpuAgg = &aggs[i]
		} else if aggs[i].MetricType == MetricMemoryGBHours {
			memAgg = &aggs[i]
		}
	}

	if cpuAgg == nil {
		t.Fatal("CPU aggregate not found")
	}

	if cpuAgg.Count != 3 {
		t.Errorf("CPU count: %d, want 3", cpuAgg.Count)
	}

	if cpuAgg.TotalValue != 60.0 {
		t.Errorf("CPU total: %.2f, want 60.0", cpuAgg.TotalValue)
	}

	if cpuAgg.AvgValue != 20.0 {
		t.Errorf("CPU avg: %.2f, want 20.0", cpuAgg.AvgValue)
	}

	if cpuAgg.MinValue != 10.0 {
		t.Errorf("CPU min: %.2f, want 10.0", cpuAgg.MinValue)
	}

	if cpuAgg.MaxValue != 30.0 {
		t.Errorf("CPU max: %.2f, want 30.0", cpuAgg.MaxValue)
	}

	if memAgg == nil || memAgg.Count != 1 {
		t.Error("Memory aggregate incorrect")
	}
}

func TestCalculateCostAnalysis(t *testing.T) {
	collector := NewCollector()
	ctx := context.Background()

	now := time.Now()

	// Record diverse metrics
	metrics := []UsageMetric{
		{Timestamp: now, EntityID: "tenant-1", MetricType: MetricCPUHours, Value: 100},
		{Timestamp: now, EntityID: "tenant-1", MetricType: MetricMemoryGBHours, Value: 200},
		{Timestamp: now, EntityID: "tenant-1", MetricType: MetricStorageGBMonths, Value: 500},
		{Timestamp: now, EntityID: "tenant-1", MetricType: MetricBandwidthGB, Value: 50},
	}

	for _, m := range metrics {
		collector.RecordMetric(ctx, m)
	}

	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)

	analysis, err := collector.CalculateCostAnalysis(ctx, "tenant-1", "tenant", start, end)
	if err != nil {
		t.Fatalf("CalculateCostAnalysis failed: %v", err)
	}

	if analysis.EntityID != "tenant-1" {
		t.Errorf("Entity ID: %s", analysis.EntityID)
	}

	if analysis.ComputeCost <= 0 {
		t.Errorf("Compute cost: %.2f", analysis.ComputeCost)
	}

	if analysis.MemoryCost <= 0 {
		t.Errorf("Memory cost: %.2f", analysis.MemoryCost)
	}

	if analysis.StorageCost <= 0 {
		t.Errorf("Storage cost: %.2f", analysis.StorageCost)
	}

	if analysis.BandwidthCost <= 0 {
		t.Errorf("Bandwidth cost: %.2f", analysis.BandwidthCost)
	}

	if analysis.TotalCost <= 0 {
		t.Errorf("Total cost: %.2f", analysis.TotalCost)
	}

	expectedTotal := analysis.ComputeCost + analysis.MemoryCost + analysis.StorageCost + analysis.BandwidthCost
	if analysis.TotalCost != expectedTotal {
		t.Errorf("Total mismatch: %.2f vs %.2f", analysis.TotalCost, expectedTotal)
	}

	if len(analysis.TopCostDrivers) == 0 {
		t.Error("No top cost drivers identified")
	}
}

func TestCalculateRevenueMetrics(t *testing.T) {
	collector := NewCollector()
	ctx := context.Background()

	now := time.Now()

	metrics := []UsageMetric{
		{Timestamp: now, EntityID: "operator-1", MetricType: MetricRevenueUSD, Value: 1000},
		{Timestamp: now, EntityID: "operator-1", MetricType: MetricRevenueUSD, Value: 500},
		{Timestamp: now, EntityID: "operator-1", MetricType: MetricLeaseCount, Value: 5},
	}

	for _, m := range metrics {
		collector.RecordMetric(ctx, m)
	}

	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)

	revenue, err := collector.CalculateRevenueMetrics(ctx, "operator-1", start, end)
	if err != nil {
		t.Fatalf("CalculateRevenueMetrics failed: %v", err)
	}

	if revenue.OperatorID != "operator-1" {
		t.Errorf("Operator ID: %s", revenue.OperatorID)
	}

	if revenue.TotalRevenue != 1500.0 {
		t.Errorf("Total revenue: %.2f, want 1500.0", revenue.TotalRevenue)
	}

	if revenue.TotalLeases != 5 {
		t.Errorf("Total leases: %d, want 5", revenue.TotalLeases)
	}

	if revenue.AvgLeaseValue != 300.0 {
		t.Errorf("Avg lease value: %.2f, want 300.0", revenue.AvgLeaseValue)
	}
}

func TestCalculateCapacityMetrics(t *testing.T) {
	collector := NewCollector()
	ctx := context.Background()

	now := time.Now()

	metrics := []UsageMetric{
		{Timestamp: now, EntityID: "cluster-1", MetricType: MetricCPUHours, Value: 480}, // 20 cores for 24h
		{Timestamp: now, EntityID: "cluster-1", MetricType: MetricMemoryGBHours, Value: 2880}, // 120GB for 24h
		{Timestamp: now, EntityID: "cluster-1", MetricType: MetricStorageGBMonths, Value: 100},
	}

	for _, m := range metrics {
		collector.RecordMetric(ctx, m)
	}

	start := now.Add(-24 * time.Hour)
	end := now.Add(24 * time.Hour)

	// Total capacity: 32 cores, 256GB memory, 1000GB storage
	capacity, err := collector.CalculateCapacityMetrics(ctx, "cluster-1", start, end, 32000, 256.0, 1000.0)
	if err != nil {
		t.Fatalf("CalculateCapacityMetrics failed: %v", err)
	}

	if capacity.EntityID != "cluster-1" {
		t.Errorf("Entity ID: %s", capacity.EntityID)
	}

	if capacity.CPUMillicoresUsed <= 0 {
		t.Errorf("CPU used: %d", capacity.CPUMillicoresUsed)
	}

	if capacity.MemoryGBUsed <= 0 {
		t.Errorf("Memory used: %.2f", capacity.MemoryGBUsed)
	}

	if capacity.CPUUtilization < 0 || capacity.CPUUtilization > 100 {
		t.Errorf("CPU utilization: %.2f%%", capacity.CPUUtilization)
	}

	if capacity.MemoryUtilization < 0 || capacity.MemoryUtilization > 100 {
		t.Errorf("Memory utilization: %.2f%%", capacity.MemoryUtilization)
	}
}

func TestAnalyzeTenant(t *testing.T) {
	collector := NewCollector()
	ctx := context.Background()

	now := time.Now()

	metrics := []UsageMetric{
		{Timestamp: now, EntityID: "tenant-1", MetricType: MetricLeaseCount, Value: 15, Region: "us"},
		{Timestamp: now, EntityID: "tenant-1", MetricType: MetricLeaseCount, Value: 10, Region: "us"},
		{Timestamp: now, EntityID: "tenant-1", MetricType: MetricLeaseCount, Value: 5, Region: "eu"},
		{Timestamp: now, EntityID: "tenant-1", MetricType: MetricRevenueUSD, Value: 500},
	}

	for _, m := range metrics {
		collector.RecordMetric(ctx, m)
	}

	analysis, err := collector.AnalyzeTenant(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("AnalyzeTenant failed: %v", err)
	}

	if analysis.TenantID != "tenant-1" {
		t.Errorf("Tenant ID: %s", analysis.TenantID)
	}

	if analysis.TotalLeases != 30 {
		t.Errorf("Total leases: %d, want 30", analysis.TotalLeases)
	}

	if analysis.TotalSpent != 500.0 {
		t.Errorf("Total spent: %.2f, want 500.0", analysis.TotalSpent)
	}

	if analysis.Engagement != "high" {
		t.Errorf("Engagement: %s, want high", analysis.Engagement)
	}

	if len(analysis.PreferredRegions) == 0 {
		t.Error("No preferred regions identified")
	}
}

func TestGetMetricsTimeSeries(t *testing.T) {
	collector := NewCollector()
	ctx := context.Background()

	now := time.Now().Truncate(time.Hour)

	// Record metrics across hours
	for i := 0; i < 24; i++ {
		metric := UsageMetric{
			Timestamp:  now.Add(time.Duration(i) * time.Hour),
			EntityID:   "tenant-1",
			MetricType: MetricCPUHours,
			Value:      float64(i * 10),
		}
		collector.RecordMetric(ctx, metric)
	}

	start := now
	end := now.Add(25 * time.Hour)

	series := collector.GetMetricsTimeSeries(ctx, "tenant-1", MetricCPUHours, start, end, 1*time.Hour)

	if len(series) == 0 {
		t.Error("No data in time series")
	}

	// Should have data for multiple buckets
	if len(series) < 5 {
		t.Errorf("Expected at least 5 buckets, got %d", len(series))
	}
}

func TestEmptyAnalytics(t *testing.T) {
	collector := NewCollector()
	ctx := context.Background()

	now := time.Now()
	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)

	// Query non-existent entity
	aggs := collector.AggregateByEntity(ctx, "unknown", start, end)
	if len(aggs) != 0 {
		t.Error("Should return empty for non-existent entity")
	}

	// Cost analysis on empty data
	analysis, _ := collector.CalculateCostAnalysis(ctx, "unknown", "tenant", start, end)
	if analysis.TotalCost != 0 {
		t.Errorf("Empty cost should be 0, got %.2f", analysis.TotalCost)
	}
}

func TestMultipleEntities(t *testing.T) {
	collector := NewCollector()
	ctx := context.Background()

	now := time.Now()

	// Record metrics for different entities
	for i := 1; i <= 5; i++ {
		for j := 0; j < 3; j++ {
			metric := UsageMetric{
				Timestamp:  now.Add(time.Duration(j) * time.Hour),
				EntityID:   "tenant-" + string(rune('0'+i)),
				MetricType: MetricCPUHours,
				Value:      float64(i * 10),
			}
			collector.RecordMetric(ctx, metric)
		}
	}

	// Aggregate for each tenant
	for i := 1; i <= 5; i++ {
		tenantID := "tenant-" + string(rune('0'+i))
		start := now.Add(-1 * time.Hour)
		end := now.Add(3 * time.Hour)

		aggs := collector.AggregateByEntity(ctx, tenantID, start, end)
		if len(aggs) == 0 {
			t.Errorf("No aggregates for %s", tenantID)
		}
	}
}

func TestMetricTimeSeries(t *testing.T) {
	collector := NewCollector()
	ctx := context.Background()

	base := time.Now().Truncate(time.Hour)

	// Create hourly data
	for h := 0; h < 48; h++ {
		metric := UsageMetric{
			Timestamp:  base.Add(time.Duration(h) * time.Hour),
			EntityID:   "operator-1",
			MetricType: MetricRevenueUSD,
			Value:      1000.0, // $1000 per hour
		}
		collector.RecordMetric(ctx, metric)
	}

	series := collector.GetMetricsTimeSeries(ctx, "operator-1", MetricRevenueUSD, base, base.Add(48*time.Hour), time.Hour)

	// Sum up total revenue
	total := 0.0
	for _, v := range series {
		total += v
	}

	if total < 48000 { // Should be close to $48,000
		t.Errorf("Total revenue: %.2f, expected ~48000", total)
	}
}
