// Integration tests for the marketplace pricing, billing, payment, and analytics pipeline.
package billing

import (
	"context"
	"testing"
	"time"

	"decentralized.host/pkg/analytics"
	"decentralized.host/pkg/payment"
	"decentralized.host/pkg/pricing"
)

// TestE2EMarketplaceFlow tests complete marketplace flow:
// pricing -> lease creation -> usage tracking -> settlement -> payment -> analytics
func TestE2EMarketplaceFlow(t *testing.T) {
	ctx := context.Background()

	// 1. Set up pricing engine
	priceEngine := pricing.NewEngine(pricing.ModeDynamic)
	priceEngine.SetBasePricing(pricing.BasePricing{
		CPUPerMillicorePerHour: 0.001,
		MemoryPerGBPerHour:     0.01,
		StoragePerGBPerMonth:   0.1,
		BandwidthPerGBPerHour:  0.005,
		MinimumHourlyCharge:    0.5,
		LeaseSetupFee:          1.0,
	})

	// Set demand multiplier (high demand = high prices)
	priceEngine.SetDemandFactor("cluster-1", 0.8, 0.9)
	priceEngine.SetRegionalAdjustment(pricing.RegionUS, 1.0)
	priceEngine.SetTimeOfDayAdjustment(pricing.WindowDefault, 1.0)

	// Add bulk discounts
	priceEngine.AddBulkDiscount(100, 0.10)
	priceEngine.AddBulkDiscount(500, 0.25)

	// 2. Get pricing quote
	quoteReq := pricing.QuoteRequest{
		ClusterID:          "cluster-1",
		CPUMillicores:      2000,
		MemoryGB:           8.0,
		StorageGB:          500.0,
		BandwidthGBPerHour: 2.0,
		DurationHours:      720, // 30 days
		Region:             pricing.RegionUS,
		TimeWindow:         pricing.WindowDefault,
	}

	quote, err := priceEngine.GetQuote(ctx, quoteReq)
	if err != nil {
		t.Fatalf("Failed to get quote: %v", err)
	}

	if quote.EstimatedMonthlyCost <= 0 {
		t.Fatalf("Invalid quote cost: %.2f", quote.EstimatedMonthlyCost)
	}

	// 3. Create lease
	billingManager := NewManager(priceEngine)

	leaseReq := CreateLeaseRequest{
		TenantID:           "tenant-1",
		OperatorID:         "operator-1",
		StartTime:          time.Now(),
		EndTime:            time.Now().AddDate(0, 1, 0),
		Quote:              quote,
		SettlementPeriod:   PeriodDaily,
		ResourceAllocation: ResourceAllocation{
			CPUMillicores:      2000,
			MemoryGB:           8.0,
			StorageGB:          500.0,
			BandwidthGBPerDay:  48.0,
		},
		Refundable:           true,
		RefundPercentIfEarly: 80,
	}

	lease, err := billingManager.CreateLease(ctx, leaseReq)
	if err != nil {
		t.Fatalf("Failed to create lease: %v", err)
	}

	if err := billingManager.ActivateLease(ctx, lease.ID); err != nil {
		t.Fatalf("Failed to activate lease: %v", err)
	}

	// 4. Record usage
	usage := ResourceUsage{
		CPUMillicoreHours: 1440.0, // 2 cores for 24h
		MemoryGBHours:     192.0,  // 8GB for 24h
		StorageGBMonths:   500.0,
		BandwidthGBUsed:   24.0,
		Timestamp:         time.Now(),
	}

	if err := billingManager.RecordUsage(ctx, lease.ID, usage); err != nil {
		t.Fatalf("Failed to record usage: %v", err)
	}

	// 5. Settle lease
	settlement, err := billingManager.SettleNow(ctx, lease.ID)
	if err != nil {
		t.Fatalf("Failed to settle lease: %v", err)
	}

	if settlement.CalculatedCost <= 0 {
		t.Fatalf("Invalid settlement cost: %.2f", settlement.CalculatedCost)
	}

	// 6. Process payment
	paymentProc := payment.NewProcessor()
	gateway := &MockPaymentGateway{}
	paymentProc.RegisterGateway(gateway)

	paymentReq := payment.PaymentRequest{
		TenantID:     "tenant-1",
		SettlementID: settlement.ID,
		Amount:       settlement.CalculatedCost,
		Currency:     payment.CurrencyUSD,
		Method:       payment.MethodCreditCard,
	}

	txn, err := paymentProc.ProcessPayment(ctx, paymentReq)
	if err != nil {
		t.Fatalf("Failed to process payment: %v", err)
	}

	if txn.Status != payment.StatusCompleted {
		t.Errorf("Payment status: %s, want completed", txn.Status)
	}

	// Record the payment
	if err := billingManager.RecordPayment(ctx, settlement.ID, settlement.CalculatedCost); err != nil {
		t.Fatalf("Failed to record payment: %v", err)
	}

	// 7. Analytics
	collector := analytics.NewCollector()

	// Record metrics for analytics
	_ = collector.RecordMetric(ctx, analytics.UsageMetric{
		Timestamp:  time.Now(),
		EntityID:   "tenant-1",
		EntityType: "tenant",
		MetricType: analytics.MetricCPUHours,
		Value:      1440.0,
		Unit:       "hours",
		Region:     "us",
	})

	_ = collector.RecordMetric(ctx, analytics.UsageMetric{
		Timestamp:  time.Now(),
		EntityID:   "tenant-1",
		EntityType: "tenant",
		MetricType: analytics.MetricRevenueUSD,
		Value:      settlement.CalculatedCost,
		Region:     "us",
	})

	_ = collector.RecordMetric(ctx, analytics.UsageMetric{
		Timestamp:  time.Now(),
		EntityID:   "operator-1",
		EntityType: "operator",
		MetricType: analytics.MetricRevenueUSD,
		Value:      settlement.CalculatedCost * 0.9, // operator gets 90%
		Region:     "us",
	})

	// Get analytics
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(1 * time.Hour)

	costAnalysis, _ := collector.CalculateCostAnalysis(ctx, "tenant-1", "tenant", start, end)
	if costAnalysis.TotalCost <= 0 {
		t.Error("Cost analysis should have positive cost")
	}

	revenueMetrics, _ := collector.CalculateRevenueMetrics(ctx, "operator-1", start, end)
	if revenueMetrics.TotalRevenue <= 0 {
		t.Error("Revenue metrics should have positive revenue")
	}

	tenantAnalytics, _ := collector.AnalyzeTenant(ctx, "tenant-1")
	if tenantAnalytics.TotalSpent <= 0 {
		t.Error("Tenant analytics should have positive spending")
	}

	t.Logf("E2E Test Summary:")
	t.Logf("  Quote (monthly):    $%.2f", quote.EstimatedMonthlyCost)
	t.Logf("  Settlement cost:    $%.2f", settlement.CalculatedCost)
	t.Logf("  Payment processed:  $%.2f", txn.Amount)
	t.Logf("  Tenant cost:        $%.2f", costAnalysis.TotalCost)
	t.Logf("  Operator revenue:   $%.2f", revenueMetrics.TotalRevenue)
}

// Mock payment gateway for testing
type MockPaymentGateway struct{}

func (m *MockPaymentGateway) Name() string {
	return "mock"
}

func (m *MockPaymentGateway) ProcessPayment(ctx context.Context, req payment.PaymentRequest) (*payment.PaymentResult, error) {
	return &payment.PaymentResult{
		TransactionID:  "mock-" + time.Now().String(),
		Status:         payment.StatusCompleted,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Timestamp:      time.Now(),
		ConfirmationID: "mock-conf",
	}, nil
}

func (m *MockPaymentGateway) RefundPayment(ctx context.Context, transactionID string, amount float64) (*payment.RefundResult, error) {
	return &payment.RefundResult{
		RefundID:      "refund-" + time.Now().String(),
		OriginalTxnID: transactionID,
		Status:        payment.StatusRefunded,
		Amount:        amount,
		Timestamp:     time.Now(),
	}, nil
}

func (m *MockPaymentGateway) GetStatus(ctx context.Context, transactionID string) (payment.PaymentStatus, error) {
	return payment.StatusCompleted, nil
}
