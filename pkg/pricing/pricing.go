// Package pricing implements dynamic pricing for Decentralized.Host marketplace.
//
// Pricing model includes:
// - Base pricing per CPU/memory/storage unit
// - Demand/supply multiplier based on capacity utilization
// - Regional pricing adjustments
// - Time-of-day pricing (peak/off-peak)
// - Bulk discount tiers
// - Price history tracking
package pricing

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// PricingMode defines the pricing strategy used.
type PricingMode string

const (
	ModeDynamic    PricingMode = "dynamic"
	ModeFixed      PricingMode = "fixed"
	ModeAuction    PricingMode = "auction"
	ModeSpotMarket PricingMode = "spot"
)

// Region defines geographic pricing zones.
type Region string

const (
	RegionUS      Region = "us"
	RegionEU      Region = "eu"
	RegionASIA    Region = "asia"
	RegionDefault Region = "default"
)

// TimeWindow represents pricing periods throughout the day.
type TimeWindow string

const (
	WindowPeak    TimeWindow = "peak"
	WindowOffPeak TimeWindow = "offpeak"
	WindowDefault TimeWindow = "default"
)

// BasePricing holds the fundamental unit prices.
type BasePricing struct {
	CPUPerMillicorePerHour    float64 // USD per milli-core per hour
	MemoryPerGBPerHour        float64 // USD per GB per hour
	StoragePerGBPerMonth      float64 // USD per GB per month
	BandwidthPerGBPerHour     float64 // USD per GB per hour
	MinimumHourlyCharge       float64 // USD minimum per lease per hour
	LeaseSetupFee             float64 // USD one-time
}

// DemandFactor represents supply/demand multiplier.
type DemandFactor struct {
	Timestamp       time.Time
	ClusterID       string
	UtilizationPct  float64 // 0.0 to 1.0
	Multiplier      float64 // price multiplier (1.0 = base price)
	PredictedPctFull float64 // predicted utilization in 24h
}

// RegionalAdjustment defines per-region pricing multipliers.
type RegionalAdjustment struct {
	Region     Region
	Multiplier float64 // 1.0 = no adjustment
}

// TimeOfDayAdjustment defines pricing based on time windows.
type TimeOfDayAdjustment struct {
	Window     TimeWindow
	Multiplier float64 // 1.0 = no adjustment
}

// BulkDiscount defines quantity discount tiers.
type BulkDiscount struct {
	MinHours   int64   // minimum lease hours to qualify
	Discount   float64 // discount as decimal (0.10 = 10% off)
	Multiplier float64 // 1.0 - discount
}

// PriceSnapshot captures pricing at a point in time.
type PriceSnapshot struct {
	Timestamp           time.Time
	Mode                PricingMode
	Base                BasePricing
	DemandFactor        DemandFactor
	RegionalAdjustment  RegionalAdjustment
	TimeOfDayAdjustment TimeOfDayAdjustment
	EffectiveMultiplier float64 // total multiplicative factor
	Hash                string  // content hash for verification
}

// Engine calculates prices based on current market conditions.
type Engine struct {
	mu                     sync.RWMutex
	mode                   PricingMode
	base                   BasePricing
	demandFactors          map[string]*DemandFactor
	regionalAdjustments    map[Region]float64
	timeOfDayAdjustments   map[TimeWindow]float64
	bulkDiscounts          []BulkDiscount
	priceHistory           []*PriceSnapshot
	maxHistoryEntries      int
	lastSnapshotTime       time.Time
	snapshotIntervalSecs   int64
}

// NewEngine creates a new pricing engine with default settings.
func NewEngine(mode PricingMode) *Engine {
	return &Engine{
		mode:              mode,
		demandFactors:     make(map[string]*DemandFactor),
		regionalAdjustments: make(map[Region]float64),
		timeOfDayAdjustments: make(map[TimeWindow]float64),
		bulkDiscounts:     []BulkDiscount{},
		priceHistory:      []*PriceSnapshot{},
		maxHistoryEntries: 10000,
		snapshotIntervalSecs: 300, // 5 minutes
	}
}

// SetBasePricing updates the base pricing model.
func (e *Engine) SetBasePricing(bp BasePricing) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.base = bp
}

// SetDemandFactor updates the demand multiplier for a cluster.
func (e *Engine) SetDemandFactor(clusterID string, utilization float64, predicted float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	multiplier := calculateDemandMultiplier(utilization)
	e.demandFactors[clusterID] = &DemandFactor{
		Timestamp:        time.Now(),
		ClusterID:        clusterID,
		UtilizationPct:   utilization,
		Multiplier:       multiplier,
		PredictedPctFull: predicted,
	}
}

// SetRegionalAdjustment sets per-region pricing multiplier.
func (e *Engine) SetRegionalAdjustment(region Region, multiplier float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if multiplier <= 0 {
		multiplier = 1.0
	}
	e.regionalAdjustments[region] = multiplier
}

// SetTimeOfDayAdjustment sets pricing adjustment for time windows.
func (e *Engine) SetTimeOfDayAdjustment(window TimeWindow, multiplier float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if multiplier <= 0 {
		multiplier = 1.0
	}
	e.timeOfDayAdjustments[window] = multiplier
}

// AddBulkDiscount adds a quantity discount tier.
func (e *Engine) AddBulkDiscount(minHours int64, discount float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if discount < 0 || discount >= 1.0 {
		discount = 0 // no discount
	}

	bd := BulkDiscount{
		MinHours:   minHours,
		Discount:   discount,
		Multiplier: 1.0 - discount,
	}

	e.bulkDiscounts = append(e.bulkDiscounts, bd)
	sort.Slice(e.bulkDiscounts, func(i, j int) bool {
		return e.bulkDiscounts[i].MinHours < e.bulkDiscounts[j].MinHours
	})
}

// Quote calculates the price for a resource allocation.
type Quote struct {
	CPUCostPerHour        float64 // per milli-core
	MemoryCostPerHour     float64 // per GB
	StorageCostPerMonth   float64 // per GB
	BandwidthCostPerHour  float64 // per GB
	TotalHourlyCharge     float64
	BulkDiscountApplied   bool
	BulkDiscountPercent   float64
	EstimatedMonthlyCost  float64
	ValidUntil            time.Time
	Timestamp             time.Time
}

// GetQuote calculates a price quote for the given parameters.
func (e *Engine) GetQuote(ctx context.Context, req QuoteRequest) (*Quote, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get adjustments
	demand := e.getDemandFactorLocked(req.ClusterID)
	demandMult := 1.0
	if demand != nil {
		demandMult = demand.Multiplier
	}

	regional := e.regionalAdjustments[req.Region]
	if regional == 0 {
		regional = e.regionalAdjustments[RegionDefault]
		if regional == 0 {
			regional = 1.0
		}
	}

	timeOfDay := e.timeOfDayAdjustments[req.TimeWindow]
	if timeOfDay == 0 {
		timeOfDay = e.timeOfDayAdjustments[WindowDefault]
		if timeOfDay == 0 {
			timeOfDay = 1.0
		}
	}

	// Calculate per-unit costs
	cpuCost := e.base.CPUPerMillicorePerHour * demandMult * regional * timeOfDay
	memCost := e.base.MemoryPerGBPerHour * demandMult * regional * timeOfDay
	storageCost := e.base.StoragePerGBPerMonth * regional
	bwCost := e.base.BandwidthPerGBPerHour * demandMult * regional * timeOfDay

	// Calculate total hourly charge
	cpuTotal := cpuCost * float64(req.CPUMillicores)
	memTotal := memCost * float64(req.MemoryGB)
	bwTotal := bwCost * float64(req.BandwidthGBPerHour)

	hourlyCharge := cpuTotal + memTotal + bwTotal
	if hourlyCharge < e.base.MinimumHourlyCharge {
		hourlyCharge = e.base.MinimumHourlyCharge
	}

	// Apply bulk discount
	bulkMult := 1.0
	bulkDiscount := 0.0
	if req.DurationHours > 0 {
		for i := len(e.bulkDiscounts) - 1; i >= 0; i-- {
			if req.DurationHours >= e.bulkDiscounts[i].MinHours {
				bulkMult = e.bulkDiscounts[i].Multiplier
				bulkDiscount = e.bulkDiscounts[i].Discount
				break
			}
		}
	}

	hourlyCharge *= bulkMult

	// Storage cost is separate (monthly)
	storageCostPerMonth := storageCost * float64(req.StorageGB)

	// Estimated monthly cost
	estimatedMonthly := (hourlyCharge * 730) + storageCostPerMonth + e.base.LeaseSetupFee

	q := &Quote{
		CPUCostPerHour:       cpuCost,
		MemoryCostPerHour:    memCost,
		StorageCostPerMonth:  storageCost,
		BandwidthCostPerHour: bwCost,
		TotalHourlyCharge:    hourlyCharge,
		BulkDiscountApplied:  bulkDiscount > 0,
		BulkDiscountPercent:  bulkDiscount * 100,
		EstimatedMonthlyCost: estimatedMonthly,
		ValidUntil:           time.Now().Add(15 * time.Minute),
		Timestamp:            time.Now(),
	}

	return q, nil
}

// QuoteRequest is a pricing inquiry.
type QuoteRequest struct {
	ClusterID           string
	CPUMillicores       int64  // millicores
	MemoryGB            float64
	StorageGB           float64
	BandwidthGBPerHour  float64
	DurationHours       int64
	Region              Region
	TimeWindow          TimeWindow
}

// TakeSnapshot captures current pricing state for audit trail.
func (e *Engine) TakeSnapshot(ctx context.Context) (*PriceSnapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()

	// Only snapshot if interval has passed
	if !e.lastSnapshotTime.IsZero() {
		elapsed := now.Sub(e.lastSnapshotTime).Seconds()
		if elapsed < float64(e.snapshotIntervalSecs) {
			return nil, fmt.Errorf("snapshot interval not met: %.0f < %d seconds", elapsed, e.snapshotIntervalSecs)
		}
	}

	// Build snapshot from current state
	ps := &PriceSnapshot{
		Timestamp:    now,
		Mode:         e.mode,
		Base:         e.base,
		EffectiveMultiplier: 1.0,
	}

	// Aggregate demand factor
	if len(e.demandFactors) > 0 {
		avgUtil := 0.0
		avgMult := 0.0
		for _, df := range e.demandFactors {
			avgUtil += df.UtilizationPct
			avgMult += df.Multiplier
		}
		n := float64(len(e.demandFactors))
		ps.DemandFactor = DemandFactor{
			Timestamp:        now,
			UtilizationPct:   avgUtil / n,
			Multiplier:       avgMult / n,
		}
	}

	// Get average regional adjustment
	ps.RegionalAdjustment = RegionalAdjustment{Multiplier: 1.0}
	if len(e.regionalAdjustments) > 0 {
		sum := 0.0
		for _, m := range e.regionalAdjustments {
			sum += m
		}
		ps.RegionalAdjustment = RegionalAdjustment{
			Multiplier: sum / float64(len(e.regionalAdjustments)),
		}
	}

	// Get average time-of-day adjustment
	ps.TimeOfDayAdjustment = TimeOfDayAdjustment{Multiplier: 1.0}
	if len(e.timeOfDayAdjustments) > 0 {
		sum := 0.0
		for _, m := range e.timeOfDayAdjustments {
			sum += m
		}
		ps.TimeOfDayAdjustment = TimeOfDayAdjustment{
			Multiplier: sum / float64(len(e.timeOfDayAdjustments)),
		}
	}

	demandMult := ps.DemandFactor.Multiplier
	if demandMult == 0 {
		demandMult = 1.0
	}
	ps.EffectiveMultiplier = demandMult * ps.RegionalAdjustment.Multiplier * ps.TimeOfDayAdjustment.Multiplier

	// Compute hash
	ps.Hash = e.hashSnapshot(ps)

	// Store in history
	e.priceHistory = append(e.priceHistory, ps)
	if len(e.priceHistory) > e.maxHistoryEntries {
		e.priceHistory = e.priceHistory[1:]
	}

	e.lastSnapshotTime = now

	return ps, nil
}

// GetPriceHistory returns pricing snapshots within a time window.
func (e *Engine) GetPriceHistory(ctx context.Context, since time.Time, limit int) []*PriceSnapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	var result []*PriceSnapshot
	for i := len(e.priceHistory) - 1; i >= 0 && len(result) < limit; i-- {
		if e.priceHistory[i].Timestamp.After(since) {
			result = append(result, e.priceHistory[i])
		}
	}

	return result
}

// Helper functions

func calculateDemandMultiplier(utilization float64) float64 {
	// Sigmoid-like curve: starts at 1.0 at 0%, rises to 2.0+ at 95%+
	// Uses formula: 1 + (1.5 * utilization^2) to create non-linear scaling

	if utilization <= 0 {
		return 1.0
	}
	if utilization >= 1.0 {
		return 2.5
	}

	// Non-linear scaling: exponential increase as utilization approaches 100%
	return 1.0 + (1.5 * math.Pow(utilization, 2))
}

func (e *Engine) getDemandFactorLocked(clusterID string) *DemandFactor {
	if clusterID == "" {
		// Return aggregated demand across all clusters
		if len(e.demandFactors) == 0 {
			return nil
		}
		avgUtil := 0.0
		avgMult := 0.0
		for _, df := range e.demandFactors {
			avgUtil += df.UtilizationPct
			avgMult += df.Multiplier
		}
		n := float64(len(e.demandFactors))
		return &DemandFactor{
			Timestamp:      time.Now(),
			UtilizationPct: avgUtil / n,
			Multiplier:     avgMult / n,
		}
	}
	return e.demandFactors[clusterID]
}

func (e *Engine) hashSnapshot(ps *PriceSnapshot) string {
	data := fmt.Sprintf(
		"%.6f:%.6f:%.6f:%.6f:%.4f:%.4f:%.4f:%d",
		ps.Base.CPUPerMillicorePerHour,
		ps.Base.MemoryPerGBPerHour,
		ps.Base.StoragePerGBPerMonth,
		ps.DemandFactor.Multiplier,
		ps.RegionalAdjustment.Multiplier,
		ps.TimeOfDayAdjustment.Multiplier,
		ps.EffectiveMultiplier,
		ps.Timestamp.Unix(),
	)

	h := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", h)[:16]
}
