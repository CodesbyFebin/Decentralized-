package billing

import (
	"context"
	"math"
	"testing"
	"time"

	"decentralized.host/pkg/pricing"
)

func setupPricingEngine() *pricing.Engine {
	engine := pricing.NewEngine(pricing.ModeDynamic)
	engine.SetBasePricing(pricing.BasePricing{
		CPUPerMillicorePerHour: 0.001,
		MemoryPerGBPerHour:     0.01,
		StoragePerGBPerMonth:   0.1,
		BandwidthPerGBPerHour:  0.005,
		MinimumHourlyCharge:    0.5,
		LeaseSetupFee:          1.0,
	})
	return engine
}

func TestCreateLease(t *testing.T) {
	engine := setupPricingEngine()
	manager := NewManager(engine)

	ctx := context.Background()

	quote := &pricing.Quote{
		CPUCostPerHour:       0.001,
		MemoryCostPerHour:    0.01,
		TotalHourlyCharge:    0.5,
		EstimatedMonthlyCost: 365,
		ValidUntil:           time.Now().Add(15 * time.Minute),
		Timestamp:            time.Now(),
	}

	req := CreateLeaseRequest{
		TenantID:   "tenant-1",
		OperatorID: "operator-1",
		StartTime:  time.Now(),
		EndTime:    time.Now().AddDate(0, 1, 0),
		Quote:      quote,
		SettlementPeriod: PeriodMonthly,
		ResourceAllocation: ResourceAllocation{
			CPUMillicores:      1000,
			MemoryGB:           4.0,
			StorageGB:          100.0,
			BandwidthGBPerHour: 1.0,
		},
		Refundable:       true,
		RefundPercentIfEarly: 80,
	}

	lease, err := manager.CreateLease(ctx, req)
	if err != nil {
		t.Fatalf("CreateLease failed: %v", err)
	}

	if lease.ID == "" {
		t.Error("Lease ID is empty")
	}

	if lease.State != StateProposed {
		t.Errorf("Lease state: %v, want %v", lease.State, StateProposed)
	}

	if lease.TenantID != "tenant-1" {
		t.Errorf("Tenant ID: %s", lease.TenantID)
	}

	if lease.SettlementPeriod != PeriodMonthly {
		t.Errorf("Settlement period: %v", lease.SettlementPeriod)
	}
}

func TestActivateLease(t *testing.T) {
	engine := setupPricingEngine()
	manager := NewManager(engine)

	ctx := context.Background()

	quote := &pricing.Quote{
		TotalHourlyCharge:    0.5,
		EstimatedMonthlyCost: 365,
		ValidUntil:           time.Now().Add(15 * time.Minute),
	}

	req := CreateLeaseRequest{
		TenantID:           "tenant-1",
		OperatorID:         "operator-1",
		StartTime:          time.Now(),
		EndTime:            time.Now().AddDate(0, 1, 0),
		Quote:              quote,
		SettlementPeriod:   PeriodMonthly,
		ResourceAllocation: ResourceAllocation{},
	}

	lease, _ := manager.CreateLease(ctx, req)

	if err := manager.ActivateLease(ctx, lease.ID); err != nil {
		t.Fatalf("ActivateLease failed: %v", err)
	}

	lease, _ = manager.GetLease(ctx, lease.ID)
	if lease.State != StateActive {
		t.Errorf("Lease state: %v, want active", lease.State)
	}
}

func TestRecordUsage(t *testing.T) {
	engine := setupPricingEngine()
	manager := NewManager(engine)

	ctx := context.Background()

	quote := &pricing.Quote{
		TotalHourlyCharge:    0.5,
		EstimatedMonthlyCost: 365,
	}

	req := CreateLeaseRequest{
		TenantID:   "tenant-1",
		OperatorID: "operator-1",
		StartTime:  time.Now(),
		EndTime:    time.Now().AddDate(0, 1, 0),
		Quote:      quote,
		SettlementPeriod: PeriodMonthly,
		ResourceAllocation: ResourceAllocation{
			BandwidthGBPerDay: 10.0,
		},
	}

	lease, _ := manager.CreateLease(ctx, req)
	manager.ActivateLease(ctx, lease.ID)

	usage := ResourceUsage{
		CPUMillicoreHours: 240.0,
		MemoryGBHours:    32.0,
		StorageGBMonths:   100.0,
		BandwidthGBUsed:   5.0,
		Timestamp:         time.Now(),
	}

	if err := manager.RecordUsage(ctx, lease.ID, usage); err != nil {
		t.Fatalf("RecordUsage failed: %v", err)
	}

	lease, _ = manager.GetLease(ctx, lease.ID)
	if lease.UsageAccumulator.CPUMillicoreHours != 240.0 {
		t.Errorf("CPU usage: %.2f, want 240.0", lease.UsageAccumulator.CPUMillicoreHours)
	}
}

func TestSettleLease(t *testing.T) {
	engine := setupPricingEngine()
	manager := NewManager(engine)

	ctx := context.Background()

	quote := &pricing.Quote{
		CPUCostPerHour:       0.001,
		MemoryCostPerHour:    0.01,
		BandwidthCostPerHour: 0.005,
		TotalHourlyCharge:    0.5,
		EstimatedMonthlyCost: 365,
	}

	req := CreateLeaseRequest{
		TenantID:   "tenant-1",
		OperatorID: "operator-1",
		StartTime:  time.Now().Add(-24 * time.Hour),
		EndTime:    time.Now().AddDate(0, 1, 0),
		Quote:      quote,
		SettlementPeriod: PeriodDaily,
		ResourceAllocation: ResourceAllocation{
			StorageGB: 100.0,
		},
	}

	lease, _ := manager.CreateLease(ctx, req)
	manager.ActivateLease(ctx, lease.ID)

	usage := ResourceUsage{
		CPUMillicoreHours:  240.0, // 1 core for 24h
		MemoryGBHours:     96.0,  // 4GB for 24h
		StorageGBMonths:    100.0,
		BandwidthGBUsed:    10.0,
	}

	manager.RecordUsage(ctx, lease.ID, usage)

	settlement, err := manager.SettleNow(ctx, lease.ID)
	if err != nil {
		t.Fatalf("SettleNow failed: %v", err)
	}

	if settlement.ID == "" {
		t.Error("Settlement ID is empty")
	}

	if settlement.Status != "calculated" {
		t.Errorf("Settlement status: %s", settlement.Status)
	}

	if settlement.CalculatedCost <= 0 {
		t.Errorf("Calculated cost: %.2f", settlement.CalculatedCost)
	}

	if settlement.TotalDue != settlement.CalculatedCost {
		t.Errorf("Total due mismatch: %.2f vs %.2f", settlement.TotalDue, settlement.CalculatedCost)
	}
}

func TestExpireLease(t *testing.T) {
	engine := setupPricingEngine()
	manager := NewManager(engine)

	ctx := context.Background()

	quote := &pricing.Quote{
		TotalHourlyCharge:    0.5,
		EstimatedMonthlyCost: 365,
	}

	req := CreateLeaseRequest{
		TenantID:           "tenant-1",
		OperatorID:         "operator-1",
		StartTime:          time.Now(),
		EndTime:            time.Now().AddDate(0, 1, 0),
		Quote:              quote,
		SettlementPeriod:   PeriodMonthly,
		ResourceAllocation: ResourceAllocation{},
	}

	lease, _ := manager.CreateLease(ctx, req)
	manager.ActivateLease(ctx, lease.ID)

	if err := manager.ExpireLease(ctx, lease.ID); err != nil {
		t.Fatalf("ExpireLease failed: %v", err)
	}

	lease, _ = manager.GetLease(ctx, lease.ID)
	if lease.State != StateExpired {
		t.Errorf("Lease state: %v, want expired", lease.State)
	}
}

func TestCancelLease(t *testing.T) {
	engine := setupPricingEngine()
	manager := NewManager(engine)

	ctx := context.Background()

	now := time.Now()
	future := now.AddDate(0, 1, 0)

	quote := &pricing.Quote{
		TotalHourlyCharge:    0.5,
		EstimatedMonthlyCost: 100,
	}

	req := CreateLeaseRequest{
		TenantID:              "tenant-1",
		OperatorID:           "operator-1",
		StartTime:            now,
		EndTime:              future,
		Quote:                quote,
		SettlementPeriod:     PeriodMonthly,
		Refundable:           true,
		RefundPercentIfEarly: 80,
		ResourceAllocation:   ResourceAllocation{},
	}

	lease, _ := manager.CreateLease(ctx, req)
	manager.ActivateLease(ctx, lease.ID)

	// Cancel after half the duration
	time.Sleep(100 * time.Millisecond)

	refund, err := manager.CancelLease(ctx, lease.ID)
	if err != nil {
		t.Fatalf("CancelLease failed: %v", err)
	}

	if refund.EligibleAmount <= 0 {
		t.Errorf("Refund eligible amount: %.2f", refund.EligibleAmount)
	}

	lease, _ = manager.GetLease(ctx, lease.ID)
	if lease.State != StateCancelled {
		t.Errorf("Lease state: %v, want cancelled", lease.State)
	}
}

func TestRecordPayment(t *testing.T) {
	engine := setupPricingEngine()
	manager := NewManager(engine)

	ctx := context.Background()

	quote := &pricing.Quote{
		TotalHourlyCharge:    0.5,
		EstimatedMonthlyCost: 365,
	}

	req := CreateLeaseRequest{
		TenantID:           "tenant-1",
		OperatorID:         "operator-1",
		StartTime:          time.Now().Add(-24 * time.Hour),
		EndTime:            time.Now().AddDate(0, 1, 0),
		Quote:              quote,
		SettlementPeriod:   PeriodDaily,
		ResourceAllocation: ResourceAllocation{},
	}

	lease, _ := manager.CreateLease(ctx, req)
	manager.ActivateLease(ctx, lease.ID)

	// Record usage first so settlement has cost
	usage := ResourceUsage{
		CPUMillicoreHours: 100.0,
		MemoryGBHours:    50.0,
	}
	manager.RecordUsage(ctx, lease.ID, usage)

	settlement, _ := manager.SettleNow(ctx, lease.ID)

	if settlement.CalculatedCost <= 0 {
		t.Fatalf("Settlement cost is 0, cannot test payment")
	}

	payment := settlement.CalculatedCost / 2
	if err := manager.RecordPayment(ctx, settlement.ID, payment); err != nil {
		t.Fatalf("RecordPayment failed: %v", err)
	}

	settlement, _ = manager.GetSettlement(ctx, settlement.ID)
	if settlement.PaidAmount != payment {
		t.Errorf("Paid amount: %.2f, want %.2f", settlement.PaidAmount, payment)
	}

	expectedOutstanding := settlement.TotalDue - payment
	if math.Abs(settlement.OutstandingBalance-expectedOutstanding) > 0.01 {
		t.Errorf("Outstanding: %.2f, want %.2f", settlement.OutstandingBalance, expectedOutstanding)
	}
}

func TestListLeases(t *testing.T) {
	engine := setupPricingEngine()
	manager := NewManager(engine)

	ctx := context.Background()

	quote := &pricing.Quote{
		TotalHourlyCharge:    0.5,
		EstimatedMonthlyCost: 365,
	}

	// Create multiple leases
	for i := 0; i < 5; i++ {
		req := CreateLeaseRequest{
			TenantID:           "tenant-1",
			OperatorID:         "operator-1",
			StartTime:          time.Now(),
			EndTime:            time.Now().AddDate(0, 1, 0),
			Quote:              quote,
			SettlementPeriod:   PeriodMonthly,
			ResourceAllocation: ResourceAllocation{},
		}
		manager.CreateLease(ctx, req)
	}

	leases := manager.ListLeases(ctx, "tenant-1")
	if len(leases) != 5 {
		t.Errorf("Expected 5 leases, got %d", len(leases))
	}

	// Filter by state
	manager.ActivateLease(ctx, leases[0].ID)

	active := manager.ListLeases(ctx, "tenant-1", StateActive)
	if len(active) != 1 {
		t.Errorf("Expected 1 active lease, got %d", len(active))
	}

	proposed := manager.ListLeases(ctx, "tenant-1", StateProposed)
	if len(proposed) != 4 {
		t.Errorf("Expected 4 proposed leases, got %d", len(proposed))
	}
}

func TestBandwidthOverage(t *testing.T) {
	engine := setupPricingEngine()
	manager := NewManager(engine)

	ctx := context.Background()

	quote := &pricing.Quote{
		BandwidthCostPerHour: 0.005,
		TotalHourlyCharge:    0.5,
		EstimatedMonthlyCost: 365,
	}

	req := CreateLeaseRequest{
		TenantID:   "tenant-1",
		OperatorID: "operator-1",
		StartTime:  time.Now(),
		EndTime:    time.Now().AddDate(0, 1, 0),
		Quote:      quote,
		SettlementPeriod: PeriodDaily,
		ResourceAllocation: ResourceAllocation{
			BandwidthGBPerDay: 5.0,
		},
	}

	lease, _ := manager.CreateLease(ctx, req)
	manager.ActivateLease(ctx, lease.ID)

	// Usage exceeds allocated bandwidth
	usage := ResourceUsage{
		BandwidthGBUsed: 10.0, // twice the daily limit
	}

	manager.RecordUsage(ctx, lease.ID, usage)

	settlement, _ := manager.SettleNow(ctx, lease.ID)

	if settlement.UsageOverageCost <= 0 {
		t.Errorf("Overage cost should be > 0, got %.2f", settlement.UsageOverageCost)
	}
}

func TestConcurrentLeaseLimit(t *testing.T) {
	engine := setupPricingEngine()
	manager := NewManager(engine)
	manager.maxConcurrentLeases = 2

	ctx := context.Background()

	quote := &pricing.Quote{
		TotalHourlyCharge:    0.5,
		EstimatedMonthlyCost: 365,
	}

	// Create two active leases at the limit
	for i := 0; i < 2; i++ {
		req := CreateLeaseRequest{
			TenantID:           "tenant-1",
			OperatorID:         "operator-1",
			StartTime:          time.Now(),
			EndTime:            time.Now().AddDate(0, 1, 0),
			Quote:              quote,
			SettlementPeriod:   PeriodMonthly,
			ResourceAllocation: ResourceAllocation{},
		}
		lease, _ := manager.CreateLease(ctx, req)
		manager.ActivateLease(ctx, lease.ID)
	}

	// Third lease should fail
	req := CreateLeaseRequest{
		TenantID:           "tenant-1",
		OperatorID:         "operator-1",
		StartTime:          time.Now(),
		EndTime:            time.Now().AddDate(0, 1, 0),
		Quote:              quote,
		SettlementPeriod:   PeriodMonthly,
		ResourceAllocation: ResourceAllocation{},
	}

	_, err := manager.CreateLease(ctx, req)
	if err == nil {
		t.Error("Expected limit error, got nil")
	}
}
