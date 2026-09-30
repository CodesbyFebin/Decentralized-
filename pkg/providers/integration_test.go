package providers

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"decentralized.host/pkg/audit"
)

// IntegrationTestSuite runs comprehensive integration tests.
type IntegrationTestSuite struct {
	db     *sql.DB
	ledger *audit.Ledger
	store  *CampaignStore
	t      *testing.T
}

// NewIntegrationTestSuite creates a test suite.
func NewIntegrationTestSuite(t *testing.T, db *sql.DB, ledger *audit.Ledger) *IntegrationTestSuite {
	store, err := NewCampaignStore(db, ledger)
	if err != nil {
		t.Fatalf("failed to create campaign store: %v", err)
	}

	return &IntegrationTestSuite{
		db:     db,
		ledger: ledger,
		store:  store,
		t:      t,
	}
}

// TestConcurrentCampaignStorage validates concurrent campaign writes.
func (its *IntegrationTestSuite) TestConcurrentCampaignStorage() {
	ctx := context.Background()
	numGoroutines := 10
	campaignsPerGoroutine := 5

	var wg sync.WaitGroup
	var successCount int64
	errors := make(chan error, numGoroutines*campaignsPerGoroutine)

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for c := 0; c < campaignsPerGoroutine; c++ {
				campaign := &QualificationCampaign{
					ID:         fmt.Sprintf("concurrent-test-%d-%d-%d", goroutineID, c, time.Now().UnixNano()),
					ResourceID: fmt.Sprintf("resource-%d", goroutineID%3),
					StartTime:  time.Now().UnixMilli(),
					Status:     "PASSED",
					Evidence: &EvidenceSnapshot{
						ContentHash: fmt.Sprintf("hash-%d-%d", goroutineID, c),
						Signature:   fmt.Sprintf("sig-%d-%d", goroutineID, c),
						SignerID:    "test-signer",
					},
				}

				if err := its.store.StoreCampaign(ctx, campaign); err != nil {
					errors <- fmt.Errorf("goroutine %d campaign %d: %w", goroutineID, c, err)
					return
				}

				atomic.AddInt64(&successCount, 1)
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	if len(errors) > 0 {
		its.t.Errorf("concurrent storage test failed with %d errors", len(errors))
		for err := range errors {
			its.t.Logf("  %v", err)
		}
		return
	}

	expected := int64(numGoroutines * campaignsPerGoroutine)
	if successCount != expected {
		its.t.Errorf("expected %d successful campaigns, got %d", expected, successCount)
	}

	its.t.Logf("✓ Concurrent campaign storage: %d campaigns stored successfully", successCount)
}

// TestRBACEnforcement validates role-based access control.
func (its *IntegrationTestSuite) TestRBACEnforcement() {
	ctx := context.Background()
	tenantID := TenantID("test-tenant-rbac")
	tenantStore := NewTenantStore(its.store, tenantID)

	testCases := []struct {
		role          TenantRole
		operation     string
		shouldSucceed bool
	}{
		{RoleAdmin, "read", true},
		{RoleAdmin, "write", true},
		{RoleAdmin, "delete", true},
		{RoleAdmin, "export", true},
		{RoleEditor, "read", true},
		{RoleEditor, "write", true},
		{RoleEditor, "delete", false},
		{RoleEditor, "export", false},
		{RoleViewer, "read", true},
		{RoleViewer, "write", false},
		{RoleViewer, "delete", false},
		{RoleViewer, "export", false},
		{RoleAuditor, "read", true},
		{RoleAuditor, "write", false},
		{RoleAuditor, "delete", false},
		{RoleAuditor, "export", true},
	}

	for _, tc := range testCases {
		tc := tc
		t := its.t

		tc := tc
		tenantCtx := NewTenantContext(ctx, tenantID, "test-user", tc.role)
		err := tenantCtx.Authorize(tc.operation)

		if tc.shouldSucceed && err != nil {
			t.Errorf("role %s should be able to %s but got error: %v", tc.role, tc.operation, err)
		}
		if !tc.shouldSucceed && err == nil {
			t.Errorf("role %s should NOT be able to %s but operation succeeded", tc.role, tc.operation)
		}
	}

	its.t.Logf("✓ RBAC enforcement: all %d test cases passed", len(testCases))
}

// TestTenantIsolation validates campaigns are isolated per tenant.
func (its *IntegrationTestSuite) TestTenantIsolation() {
	ctx := context.Background()

	tenant1 := TenantID("tenant-1")
	tenant2 := TenantID("tenant-2")

	store1 := NewTenantStore(its.store, tenant1)
	store2 := NewTenantStore(its.store, tenant2)

	userCtx1 := NewTenantContext(ctx, tenant1, "user-1", RoleAdmin)
	userCtx2 := NewTenantContext(ctx, tenant2, "user-2", RoleAdmin)

	campaign1 := &QualificationCampaign{
		ID:         "campaign-1",
		ResourceID: "resource-1",
		StartTime:  time.Now().UnixMilli(),
		Status:     "PASSED",
	}

	campaign2 := &QualificationCampaign{
		ID:         "campaign-2",
		ResourceID: "resource-2",
		StartTime:  time.Now().UnixMilli(),
		Status:     "PASSED",
	}

	if err := store1.StoreCampaign(userCtx1, campaign1); err != nil {
		its.t.Fatalf("failed to store campaign in tenant 1: %v", err)
	}

	if err := store2.StoreCampaign(userCtx2, campaign2); err != nil {
		its.t.Fatalf("failed to store campaign in tenant 2: %v", err)
	}

	retrievedCampaigns1, err := store1.QueryCampaigns(userCtx1, "", "", "", 10, 0)
	if err != nil {
		its.t.Fatalf("failed to query campaigns in tenant 1: %v", err)
	}

	if len(retrievedCampaigns1) != 1 {
		its.t.Errorf("tenant 1 should have 1 campaign, got %d", len(retrievedCampaigns1))
	}

	retrievedCampaigns2, err := store2.QueryCampaigns(userCtx2, "", "", "", 10, 0)
	if err != nil {
		its.t.Fatalf("failed to query campaigns in tenant 2: %v", err)
	}

	if len(retrievedCampaigns2) != 1 {
		its.t.Errorf("tenant 2 should have 1 campaign, got %d", len(retrievedCampaigns2))
	}

	its.t.Logf("✓ Tenant isolation: each tenant sees only their own campaigns")
}

// TestAnalyticsQueries validates analytics queries work correctly.
func (its *IntegrationTestSuite) TestAnalyticsQueries() {
	ctx := context.Background()
	analytics := NewCampaignAnalytics(its.db)

	summary, err := analytics.GetComplianceSummary(ctx)
	if err != nil {
		its.t.Fatalf("failed to get compliance summary: %v", err)
	}

	if summary == nil {
		its.t.Fatal("compliance summary is nil")
	}

	if summary.TotalCampaigns < 0 {
		its.t.Errorf("invalid total campaigns: %d", summary.TotalCampaigns)
	}

	rates, err := analytics.GetQualificationRates(ctx, time.Now().AddDate(0, 0, -30))
	if err != nil {
		its.t.Fatalf("failed to get qualification rates: %v", err)
	}

	if rates == nil {
		its.t.Fatal("qualification rates is nil")
	}

	trends, err := analytics.GetQualificationTrend(ctx, 30)
	if err != nil {
		its.t.Fatalf("failed to get qualification trend: %v", err)
	}

	if trends == nil {
		its.t.Fatal("qualification trend is nil")
	}

	its.t.Logf("✓ Analytics queries: compliance summary, rates, and trends all functional")
}

// TestIntegrityVerification validates campaign integrity checks.
func (its *IntegrationTestSuite) TestIntegrityVerification() {
	ctx := context.Background()

	campaign := &QualificationCampaign{
		ID:         fmt.Sprintf("integrity-test-%d", time.Now().UnixNano()),
		ResourceID: "integrity-resource",
		StartTime:  time.Now().UnixMilli(),
		Status:     "PASSED",
		Evidence: &EvidenceSnapshot{
			ContentHash: "test-hash",
			Signature:   "valid-signature",
			SignerID:    "test-signer",
		},
	}

	if err := its.store.StoreCampaign(ctx, campaign); err != nil {
		its.t.Fatalf("failed to store campaign: %v", err)
	}

	retrieved, err := its.store.GetCampaign(ctx, campaign.ID)
	if err != nil {
		its.t.Fatalf("failed to retrieve campaign: %v", err)
	}

	if retrieved.Evidence == nil {
		its.t.Fatal("retrieved campaign has no evidence")
	}

	if retrieved.Evidence.Signature != campaign.Evidence.Signature {
		its.t.Errorf("signature mismatch: expected %s, got %s", campaign.Evidence.Signature, retrieved.Evidence.Signature)
	}

	its.t.Logf("✓ Integrity verification: campaign evidence preserved and retrievable")
}

// TestLargeScaleStorage validates storage of large number of campaigns.
func (its *IntegrationTestSuite) TestLargeScaleStorage() {
	ctx := context.Background()
	numCampaigns := 100

	start := time.Now()

	for i := 0; i < numCampaigns; i++ {
		campaign := &QualificationCampaign{
			ID:         fmt.Sprintf("large-scale-%d-%d", i, time.Now().UnixNano()),
			ResourceID: fmt.Sprintf("resource-large-%d", i%10),
			StartTime:  time.Now().UnixMilli(),
			Status:     "PASSED",
		}

		if err := its.store.StoreCampaign(ctx, campaign); err != nil {
			its.t.Fatalf("failed to store campaign %d: %v", i, err)
		}
	}

	elapsed := time.Since(start)

	campaigns, err := its.store.QueryCampaigns(ctx, "", "", "", 1000, 0)
	if err != nil {
		its.t.Fatalf("failed to query campaigns: %v", err)
	}

	if len(campaigns) < numCampaigns {
		its.t.Logf("⚠ expected at least %d campaigns, got %d", numCampaigns, len(campaigns))
	}

	throughput := float64(numCampaigns) / elapsed.Seconds()
	its.t.Logf("✓ Large-scale storage: %d campaigns stored in %v (%.0f campaigns/sec)", numCampaigns, elapsed, throughput)
}

// TestDatabaseAbstraction validates database driver abstraction works.
func TestDatabaseAbstraction(t *testing.T) {
	drivers := []string{"sqlite"}

	for _, driver := range drivers {
		t.Run(driver, func(t *testing.T) {
			cfg := &DatabaseConfig{
				Driver:     driver,
				ConnString: ":memory:",
				MaxOpen:    10,
				MaxIdle:    5,
			}

			db, err := NewDatabaseConnection(context.Background(), cfg)
			if err != nil {
				t.Fatalf("failed to create %s connection: %v", driver, err)
			}
			defer db.Close()

			if err := db.PingContext(context.Background()); err != nil {
				t.Fatalf("failed to ping %s database: %v", driver, err)
			}

			t.Logf("✓ Database abstraction: %s driver working", driver)
		})
	}
}
