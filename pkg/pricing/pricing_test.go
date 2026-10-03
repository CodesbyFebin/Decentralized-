package pricing

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestPricingEngineBasic(t *testing.T) {
	engine := NewEngine(ModeDynamic)

	engine.SetBasePricing(BasePricing{
		CPUPerMillicorePerHour: 0.001,
		MemoryPerGBPerHour:     0.01,
		StoragePerGBPerMonth:   0.1,
		BandwidthPerGBPerHour:  0.005,
		MinimumHourlyCharge:    0.5,
		LeaseSetupFee:          1.0,
	})

	ctx := context.Background()

	req := QuoteRequest{
		ClusterID:          "cluster-1",
		CPUMillicores:      1000, // 1 core
		MemoryGB:           4.0,
		StorageGB:          100.0,
		BandwidthGBPerHour: 1.0,
		DurationHours:      100,
		Region:             RegionDefault,
		TimeWindow:         WindowDefault,
	}

	q, err := engine.GetQuote(ctx, req)
	if err != nil {
		t.Fatalf("GetQuote failed: %v", err)
	}

	// Check calculations
	expectedCPURate := 0.001 // per millicore
	expectedMemRate := 0.01  // per GB
	expectedTotalHourly := (0.001 * 1000.0) + (0.01 * 4.0) + (0.005 * 1.0) // 1.045

	if math.Abs(q.CPUCostPerHour-expectedCPURate) > 0.0001 {
		t.Errorf("CPU cost rate: %.6f vs %.6f", q.CPUCostPerHour, expectedCPURate)
	}

	if math.Abs(q.MemoryCostPerHour-expectedMemRate) > 0.0001 {
		t.Errorf("Memory cost rate: %.6f vs %.6f", q.MemoryCostPerHour, expectedMemRate)
	}

	if math.Abs(q.TotalHourlyCharge-expectedTotalHourly) > 0.01 {
		t.Errorf("Hourly charge: %.6f vs %.6f", q.TotalHourlyCharge, expectedTotalHourly)
	}

	if q.ValidUntil.Before(time.Now()) {
		t.Error("Quote expired immediately")
	}

	if q.BulkDiscountApplied {
		t.Error("Should not have bulk discount without qualifier")
	}
}

func TestDemandMultiplier(t *testing.T) {
	cases := []struct {
		utilization float64
		minMult     float64
		maxMult     float64
	}{
		{0.0, 0.99, 1.01},
		{0.25, 1.08, 1.15},
		{0.5, 1.35, 1.45},
		{0.75, 1.80, 1.90},
		{0.95, 2.30, 2.50},
		{1.0, 2.49, 2.51},
	}

	for _, tc := range cases {
		m := calculateDemandMultiplier(tc.utilization)
		if m < tc.minMult || m > tc.maxMult {
			t.Errorf("Util %.2f: multiplier %.6f not in [%.6f, %.6f]", tc.utilization, m, tc.minMult, tc.maxMult)
		}
	}
}

func TestRegionalAdjustment(t *testing.T) {
	engine := NewEngine(ModeDynamic)

	engine.SetBasePricing(BasePricing{
		CPUPerMillicorePerHour: 0.001,
		MemoryPerGBPerHour:     0.01,
		MinimumHourlyCharge:    0.1,
	})

	// Set regional adjustments
	engine.SetRegionalAdjustment(RegionUS, 1.0)
	engine.SetRegionalAdjustment(RegionEU, 1.2)
	engine.SetRegionalAdjustment(RegionASIA, 0.8)

	ctx := context.Background()

	base := QuoteRequest{
		CPUMillicores: 1000,
		MemoryGB:      4.0,
		DurationHours: 100,
	}

	// Get quote for each region
	usReq := base
	usReq.Region = RegionUS
	q1, _ := engine.GetQuote(ctx, usReq)

	euReq := base
	euReq.Region = RegionEU
	q2, _ := engine.GetQuote(ctx, euReq)

	asiaReq := base
	asiaReq.Region = RegionASIA
	q3, _ := engine.GetQuote(ctx, asiaReq)

	// EU should be more expensive than US
	if q2.TotalHourlyCharge <= q1.TotalHourlyCharge {
		t.Errorf("EU (%.6f) should be > US (%.6f)", q2.TotalHourlyCharge, q1.TotalHourlyCharge)
	}

	// ASIA should be cheaper than US
	if q3.TotalHourlyCharge >= q1.TotalHourlyCharge {
		t.Errorf("ASIA (%.6f) should be < US (%.6f)", q3.TotalHourlyCharge, q1.TotalHourlyCharge)
	}
}

func TestTimeOfDayAdjustment(t *testing.T) {
	engine := NewEngine(ModeDynamic)

	engine.SetBasePricing(BasePricing{
		CPUPerMillicorePerHour: 0.001,
		MemoryPerGBPerHour:     0.01,
		MinimumHourlyCharge:    0.1,
	})

	engine.SetTimeOfDayAdjustment(WindowPeak, 1.5)
	engine.SetTimeOfDayAdjustment(WindowOffPeak, 0.7)

	ctx := context.Background()

	base := QuoteRequest{
		CPUMillicores: 1000,
		MemoryGB:      4.0,
		DurationHours: 100,
		Region:        RegionDefault,
	}

	peakReq := base
	peakReq.TimeWindow = WindowPeak
	q1, _ := engine.GetQuote(ctx, peakReq)

	offPeakReq := base
	offPeakReq.TimeWindow = WindowOffPeak
	q2, _ := engine.GetQuote(ctx, offPeakReq)

	// Peak should be more expensive
	if q1.TotalHourlyCharge <= q2.TotalHourlyCharge {
		t.Errorf("Peak (%.6f) should be > OffPeak (%.6f)", q1.TotalHourlyCharge, q2.TotalHourlyCharge)
	}
}

func TestBulkDiscount(t *testing.T) {
	engine := NewEngine(ModeDynamic)

	engine.SetBasePricing(BasePricing{
		CPUPerMillicorePerHour: 0.001,
		MemoryPerGBPerHour:     0.01,
		MinimumHourlyCharge:    0.1,
	})

	engine.AddBulkDiscount(100, 0.10)  // 10% off at 100 hours
	engine.AddBulkDiscount(500, 0.20)  // 20% off at 500 hours
	engine.AddBulkDiscount(1000, 0.30) // 30% off at 1000 hours

	ctx := context.Background()

	base := QuoteRequest{
		CPUMillicores: 1000,
		MemoryGB:      4.0,
		Region:        RegionDefault,
		TimeWindow:    WindowDefault,
	}

	// No discount
	q0 := base
	q0.DurationHours = 50
	quote0, _ := engine.GetQuote(ctx, q0)
	if quote0.BulkDiscountApplied {
		t.Error("Should not have discount for 50 hours")
	}

	// 10% discount
	q1 := base
	q1.DurationHours = 150
	quote1, _ := engine.GetQuote(ctx, q1)
	if !quote1.BulkDiscountApplied || math.Abs(quote1.BulkDiscountPercent-10.0) > 0.01 {
		t.Errorf("Should have 10 percent discount, got %v %.2f", quote1.BulkDiscountApplied, quote1.BulkDiscountPercent)
	}

	// 30% discount
	q2 := base
	q2.DurationHours = 1500
	quote2, _ := engine.GetQuote(ctx, q2)
	if !quote2.BulkDiscountApplied || math.Abs(quote2.BulkDiscountPercent-30.0) > 0.01 {
		t.Errorf("Should have 30 percent discount, got %v %.2f", quote2.BulkDiscountApplied, quote2.BulkDiscountPercent)
	}

	// Verify discounts are applied
	ratio := quote1.TotalHourlyCharge / quote0.TotalHourlyCharge
	if ratio > 0.92 || ratio < 0.88 {
		t.Errorf("10 percent discount ratio %.6f not in [0.88, 0.92]", ratio)
	}
}

func TestDemandFactor(t *testing.T) {
	engine := NewEngine(ModeDynamic)

	engine.SetBasePricing(BasePricing{
		CPUPerMillicorePerHour: 0.001,
		MemoryPerGBPerHour:     0.01,
		MinimumHourlyCharge:    0.1,
	})

	ctx := context.Background()
	base := QuoteRequest{
		ClusterID:     "cluster-1",
		CPUMillicores: 1000,
		MemoryGB:      4.0,
		DurationHours: 100,
		Region:        RegionDefault,
		TimeWindow:    WindowDefault,
	}

	// Get baseline quote
	engine.SetDemandFactor("cluster-1", 0.0, 0.0)
	q0, _ := engine.GetQuote(ctx, base)

	// Get quote with high demand
	engine.SetDemandFactor("cluster-1", 0.95, 0.95)
	q1, _ := engine.GetQuote(ctx, base)

	// High demand should be more expensive
	if q1.TotalHourlyCharge <= q0.TotalHourlyCharge {
		t.Errorf("High demand (%.6f) should be > low demand (%.6f)", q1.TotalHourlyCharge, q0.TotalHourlyCharge)
	}

	ratio := q1.TotalHourlyCharge / q0.TotalHourlyCharge
	if ratio < 2.0 {
		t.Errorf("High demand multiplier %.2f should be >= 2.0", ratio)
	}
}

func TestPriceSnapshot(t *testing.T) {
	engine := NewEngine(ModeDynamic)

	engine.SetBasePricing(BasePricing{
		CPUPerMillicorePerHour: 0.001,
		MemoryPerGBPerHour:     0.01,
		MinimumHourlyCharge:    0.1,
	})

	engine.SetRegionalAdjustment(RegionUS, 1.0)
	engine.SetDemandFactor("cluster-1", 0.5, 0.6)
	engine.snapshotIntervalSecs = 0 // allow immediate snapshots for testing

	ctx := context.Background()

	ps1, err := engine.TakeSnapshot(ctx)
	if err != nil {
		t.Fatalf("TakeSnapshot failed: %v", err)
	}

	if ps1.Hash == "" {
		t.Error("Snapshot hash is empty")
	}

	if ps1.Mode != ModeDynamic {
		t.Errorf("Wrong mode: %v", ps1.Mode)
	}

	if ps1.EffectiveMultiplier <= 0 {
		t.Errorf("Invalid effective multiplier: %.6f", ps1.EffectiveMultiplier)
	}

	// Get history
	hist := engine.GetPriceHistory(ctx, time.Now().Add(-1*time.Hour), 10)
	if len(hist) == 0 {
		t.Error("No history returned")
	}

	if hist[0].Hash != ps1.Hash {
		t.Error("Snapshot not in history")
	}
}

func TestMinimumHourlyCharge(t *testing.T) {
	engine := NewEngine(ModeDynamic)

	engine.SetBasePricing(BasePricing{
		CPUPerMillicorePerHour: 0.0001,
		MemoryPerGBPerHour:     0.001,
		MinimumHourlyCharge:    1.0,
	})

	ctx := context.Background()

	req := QuoteRequest{
		CPUMillicores: 10,  // very small
		MemoryGB:      0.5, // very small
		DurationHours: 100,
		Region:        RegionDefault,
		TimeWindow:    WindowDefault,
	}

	q, _ := engine.GetQuote(ctx, req)

	// Should be at least the minimum
	if q.TotalHourlyCharge < 1.0 {
		t.Errorf("Hourly charge %.6f below minimum 1.0", q.TotalHourlyCharge)
	}
}
