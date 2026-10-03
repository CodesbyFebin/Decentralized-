// Package billing implements lease settlement and usage-based billing.
//
// A lease represents a resource allocation period with:
// - Agreed pricing
// - Resource allocations (CPU, memory, storage, bandwidth)
// - Settlement periods (daily, weekly, monthly)
// - Usage tracking and accumulation
// - Billing and invoice generation
package billing

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"decentralized.host/pkg/pricing"
)

// LeaseState represents the lifecycle state of a lease.
type LeaseState string

const (
	StateProposed   LeaseState = "proposed"
	StateActive     LeaseState = "active"
	StateExpiring   LeaseState = "expiring"
	StateExpired    LeaseState = "expired"
	StateCancelled  LeaseState = "cancelled"
	StateDisputed   LeaseState = "disputed"
)

// SettlementPeriod defines billing cycle.
type SettlementPeriod string

const (
	PeriodHourly  SettlementPeriod = "hourly"
	PeriodDaily   SettlementPeriod = "daily"
	PeriodWeekly  SettlementPeriod = "weekly"
	PeriodMonthly SettlementPeriod = "monthly"
)

// Lease represents a resource allocation agreement.
type Lease struct {
	ID                    string
	TenantID              string
	OperatorID            string
	StartTime             time.Time
	EndTime               time.Time
	InitialQuote          *pricing.Quote
	SettlementPeriod      SettlementPeriod
	State                 LeaseState
	ResourceAllocation    ResourceAllocation
	UsageAccumulator      UsageAccumulator
	SettlementRecords     []*Settlement
	Refundable            bool
	RefundPercentIfEarly  float64
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ResourceAllocation specifies the guaranteed resources.
type ResourceAllocation struct {
	CPUMillicores      int64
	MemoryGB           float64
	StorageGB          float64
	BandwidthGBPerHour float64
	BandwidthGBPerDay  float64
	BandwidthGBPerMonth float64
}

// UsageAccumulator tracks actual resource consumption.
type UsageAccumulator struct {
	StartTime             time.Time
	CPUMillicoreHours     float64 // cumulative
	MemoryGBHours        float64 // cumulative
	StorageGBMonths      float64 // cumulative
	BandwidthGBUsed      float64 // cumulative
	BandwidthOverages    float64 // above agreed cap
	LastUpdateTime       time.Time
	LastPeriodStartTime  time.Time
	LastPeriodEndTime    time.Time
}

// Settlement represents a billing period settlement.
type Settlement struct {
	ID                  string
	LeaseID             string
	PeriodStart         time.Time
	PeriodEnd           time.Time
	UsageSnapshot       UsageAccumulator
	CalculatedCost      float64
	UsageOverageCost    float64
	DiscountApplied     float64
	TotalDue            float64
	PaidAmount          float64
	OutstandingBalance  float64
	Status              string // pending, calculated, billed, paid, disputed, refunded
	PaymentDeadline     time.Time
	DisputeReason       string
	DisputeResolvedAt   *time.Time
	SettledAt           time.Time
	Hash                string // content hash for verification
}

// Manager handles lease lifecycle and billing.
type Manager struct {
	mu                        sync.RWMutex
	leases                    map[string]*Lease
	settlements               map[string]*Settlement
	pricingEngine             *pricing.Engine
	overtimeMultiplier        float64
	settlementRetentionDays   int
	autoSettleOnExpire        bool
	maxConcurrentLeases       int64
}

// NewManager creates a new billing manager.
func NewManager(engine *pricing.Engine) *Manager {
	return &Manager{
		leases:                  make(map[string]*Lease),
		settlements:             make(map[string]*Settlement),
		pricingEngine:           engine,
		overtimeMultiplier:      1.5,
		settlementRetentionDays: 365,
		autoSettleOnExpire:      true,
		maxConcurrentLeases:     1000000, // 1M default
	}
}

// CreateLease creates a new lease from a pricing quote.
func (m *Manager) CreateLease(ctx context.Context, req CreateLeaseRequest) (*Lease, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check concurrent lease limit
	active := 0
	for _, l := range m.leases {
		if l.State == StateActive || l.State == StateExpiring {
			active++
		}
	}
	if int64(active) >= m.maxConcurrentLeases {
		return nil, fmt.Errorf("lease limit reached: %d active leases", active)
	}

	lease := &Lease{
		ID:                 generateLeaseID(),
		TenantID:           req.TenantID,
		OperatorID:         req.OperatorID,
		StartTime:          req.StartTime,
		EndTime:            req.EndTime,
		InitialQuote:       req.Quote,
		SettlementPeriod:   req.SettlementPeriod,
		State:              StateProposed,
		ResourceAllocation: req.ResourceAllocation,
		UsageAccumulator: UsageAccumulator{
			StartTime:          req.StartTime,
			LastUpdateTime:     req.StartTime,
			LastPeriodStartTime: req.StartTime,
		},
		SettlementRecords: []*Settlement{},
		Refundable:        req.Refundable,
		RefundPercentIfEarly: req.RefundPercentIfEarly,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	m.leases[lease.ID] = lease
	return lease, nil
}

// CreateLeaseRequest specifies lease creation parameters.
type CreateLeaseRequest struct {
	TenantID           string
	OperatorID         string
	StartTime          time.Time
	EndTime            time.Time
	Quote              *pricing.Quote
	SettlementPeriod   SettlementPeriod
	ResourceAllocation ResourceAllocation
	Refundable         bool
	RefundPercentIfEarly float64
}

// ActivateLease transitions a lease to active state.
func (m *Manager) ActivateLease(ctx context.Context, leaseID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lease, ok := m.leases[leaseID]
	if !ok {
		return fmt.Errorf("lease not found: %s", leaseID)
	}

	if lease.State != StateProposed {
		return fmt.Errorf("lease not in proposed state: %s", lease.State)
	}

	lease.State = StateActive
	lease.UpdatedAt = time.Now()

	return nil
}

// RecordUsage updates usage accumulator with actual consumption.
func (m *Manager) RecordUsage(ctx context.Context, leaseID string, usage ResourceUsage) error {
	m.mu.Lock()
	lease, ok := m.leases[leaseID]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("lease not found: %s", leaseID)
	}

	if lease.State != StateActive && lease.State != StateExpiring {
		return fmt.Errorf("lease not active: %s", lease.State)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Accumulate usage
	lease.UsageAccumulator.CPUMillicoreHours += usage.CPUMillicoreHours
	lease.UsageAccumulator.MemoryGBHours += usage.MemoryGBHours
	lease.UsageAccumulator.StorageGBMonths += usage.StorageGBMonths
	lease.UsageAccumulator.BandwidthGBUsed += usage.BandwidthGBUsed

	// Check bandwidth overage
	if usage.BandwidthGBUsed > lease.ResourceAllocation.BandwidthGBPerDay {
		overage := usage.BandwidthGBUsed - lease.ResourceAllocation.BandwidthGBPerDay
		lease.UsageAccumulator.BandwidthOverages += overage
	}

	lease.UsageAccumulator.LastUpdateTime = now
	lease.UpdatedAt = now

	return nil
}

// ResourceUsage captures consumption metrics.
type ResourceUsage struct {
	CPUMillicoreHours float64
	MemoryGBHours    float64
	StorageGBMonths  float64
	BandwidthGBUsed  float64
	Timestamp        time.Time
}

// SettleNow generates a settlement for a lease's current period.
func (m *Manager) SettleNow(ctx context.Context, leaseID string) (*Settlement, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	lease, ok := m.leases[leaseID]
	if !ok {
		return nil, fmt.Errorf("lease not found: %s", leaseID)
	}

	now := time.Now()

	// Determine settlement period boundaries
	periodStart := lease.UsageAccumulator.LastPeriodStartTime
	periodEnd := now

	if lease.State == StateExpired {
		periodEnd = lease.EndTime
	}

	settlement := &Settlement{
		ID:            generateSettlementID(),
		LeaseID:       leaseID,
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		UsageSnapshot: lease.UsageAccumulator,
		Status:        "calculated",
		PaymentDeadline: now.AddDate(0, 0, 30), // 30 days
		SettledAt:     time.Now(),
	}

	// Calculate cost
	cost := m.calculateSettlementCost(lease, settlement)

	settlement.CalculatedCost = cost
	settlement.TotalDue = cost
	settlement.OutstandingBalance = cost

	// Generate hash
	settlement.Hash = m.hashSettlement(settlement)

	m.settlements[settlement.ID] = settlement
	lease.SettlementRecords = append(lease.SettlementRecords, settlement)

	// Reset usage accumulator for next period
	lease.UsageAccumulator.LastPeriodStartTime = periodEnd
	lease.UsageAccumulator.LastPeriodEndTime = periodEnd

	return settlement, nil
}

// ExpireLease moves a lease to expired state and optionally settles final charges.
func (m *Manager) ExpireLease(ctx context.Context, leaseID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lease, ok := m.leases[leaseID]
	if !ok {
		return fmt.Errorf("lease not found: %s", leaseID)
	}

	if lease.State == StateExpired || lease.State == StateCancelled {
		return fmt.Errorf("lease already terminated: %s", lease.State)
	}

	lease.State = StateExpired
	lease.EndTime = time.Now()
	lease.UpdatedAt = time.Now()

	if m.autoSettleOnExpire {
		// Generate final settlement
		m.mu.Unlock()
		_, err := m.SettleNow(ctx, leaseID)
		m.mu.Lock()
		if err != nil {
			return fmt.Errorf("failed to settle on expiry: %w", err)
		}
	}

	return nil
}

// CancelLease terminates a lease early with refund calculation.
func (m *Manager) CancelLease(ctx context.Context, leaseID string) (*Refund, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	lease, ok := m.leases[leaseID]
	if !ok {
		return nil, fmt.Errorf("lease not found: %s", leaseID)
	}

	if lease.State == StateExpired || lease.State == StateCancelled {
		return nil, fmt.Errorf("lease already terminated: %s", lease.State)
	}

	lease.State = StateCancelled
	lease.UpdatedAt = time.Now()

	// Calculate refund if applicable
	refund := &Refund{
		LeaseID:        leaseID,
		RequestedAt:    time.Now(),
		EligibleAmount: 0,
	}

	if lease.Refundable {
		// Calculate pro-rata refund
		now := time.Now()
		duration := lease.EndTime.Sub(lease.StartTime)
		used := now.Sub(lease.StartTime)
		remaining := duration - used

		if remaining > 0 {
			pct := float64(remaining) / float64(duration)
			refund.EligibleAmount = lease.InitialQuote.EstimatedMonthlyCost * pct * (lease.RefundPercentIfEarly / 100.0)
		}
	}

	return refund, nil
}

// Refund represents a refund transaction.
type Refund struct {
	LeaseID        string
	RequestedAt    time.Time
	EligibleAmount float64
	ApprovedAmount float64
	Status         string // pending, approved, rejected, processed
	ProcessedAt    *time.Time
	ApprovalNote   string
}

// RecordPayment records a payment against a settlement.
func (m *Manager) RecordPayment(ctx context.Context, settlementID string, amount float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	settlement, ok := m.settlements[settlementID]
	if !ok {
		return fmt.Errorf("settlement not found: %s", settlementID)
	}

	if amount <= 0 {
		return fmt.Errorf("invalid payment amount: %.2f", amount)
	}

	settlement.PaidAmount += amount
	settlement.OutstandingBalance = settlement.TotalDue - settlement.PaidAmount

	if settlement.OutstandingBalance <= 0 {
		settlement.Status = "paid"
	}

	return nil
}

// GetLease returns lease details.
func (m *Manager) GetLease(ctx context.Context, leaseID string) (*Lease, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lease, ok := m.leases[leaseID]
	if !ok {
		return nil, fmt.Errorf("lease not found: %s", leaseID)
	}

	return lease, nil
}

// ListLeases returns leases for a tenant.
func (m *Manager) ListLeases(ctx context.Context, tenantID string, states ...LeaseState) []*Lease {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Lease
	stateMap := make(map[LeaseState]bool)
	for _, s := range states {
		stateMap[s] = true
	}

	for _, lease := range m.leases {
		if lease.TenantID == tenantID {
			if len(stateMap) == 0 || stateMap[lease.State] {
				result = append(result, lease)
			}
		}
	}

	return result
}

// GetSettlement returns settlement details.
func (m *Manager) GetSettlement(ctx context.Context, settlementID string) (*Settlement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	settlement, ok := m.settlements[settlementID]
	if !ok {
		return nil, fmt.Errorf("settlement not found: %s", settlementID)
	}

	return settlement, nil
}

// Helper functions

func (m *Manager) calculateSettlementCost(lease *Lease, settlement *Settlement) float64 {
	// Calculate cost based on usage and pricing
	// For now: simple proportional pricing

	if lease.InitialQuote == nil {
		return 0
	}

	usage := settlement.UsageSnapshot

	// CPU cost
	cpuCost := usage.CPUMillicoreHours * lease.InitialQuote.CPUCostPerHour

	// Memory cost
	memoryCost := usage.MemoryGBHours * lease.InitialQuote.MemoryCostPerHour

	// Storage cost (pro-rata monthly)
	duration := settlement.PeriodEnd.Sub(settlement.PeriodStart)
	monthFraction := duration.Hours() / 730.0 // 730 hours per month
	storageCost := usage.StorageGBMonths * lease.InitialQuote.StorageCostPerMonth * monthFraction

	// Bandwidth cost
	bwCost := usage.BandwidthGBUsed * lease.InitialQuote.BandwidthCostPerHour

	// Overage cost (1.5x multiplier)
	overageCost := usage.BandwidthOverages * lease.InitialQuote.BandwidthCostPerHour * m.overtimeMultiplier

	totalCost := cpuCost + memoryCost + storageCost + bwCost + overageCost

	settlement.UsageOverageCost = overageCost

	return totalCost
}

func (m *Manager) hashSettlement(s *Settlement) string {
	data := fmt.Sprintf(
		"%s:%s:%.2f:%.2f:%d",
		s.LeaseID,
		s.PeriodStart.Format(time.RFC3339),
		s.CalculatedCost,
		s.TotalDue,
		s.PeriodEnd.Unix(),
	)

	h := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", h)[:16]
}

func generateLeaseID() string {
	h := sha256.Sum256([]byte(time.Now().String()))
	return fmt.Sprintf("lease-%x", h)[:18]
}

func generateSettlementID() string {
	h := sha256.Sum256([]byte(time.Now().String()))
	return fmt.Sprintf("settle-%x", h)[:20]
}
