package tenant

import (
	"testing"

	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/rbac"
)

func TestTenant_CreateAndGet(t *testing.T) {
	ledger := &audit.Ledger{}
	rbacMgr := rbac.NewManager(ledger)
	manager := NewManager(rbacMgr, ledger)

	tenant := &Tenant{
		ID:    "tenant-1",
		Name:  "Test Tenant",
		Owner: "dh1owner123456789abcdefgh",
	}

	if err := manager.CreateTenant(tenant); err != nil {
		t.Fatalf("create tenant failed: %v", err)
	}

	retrieved, err := manager.GetTenant("tenant-1")
	if err != nil {
		t.Fatalf("get tenant failed: %v", err)
	}

	if retrieved.ID != tenant.ID || retrieved.Name != tenant.Name {
		t.Fatalf("tenant mismatch: got %+v, want %+v", retrieved, tenant)
	}

	if retrieved.Status != "active" {
		t.Fatalf("expected status 'active', got %q", retrieved.Status)
	}
}

func TestTenant_QuotaEnforcement(t *testing.T) {
	ledger := &audit.Ledger{}
	rbacMgr := rbac.NewManager(ledger)
	manager := NewManager(rbacMgr, ledger)

	tenant := &Tenant{
		ID:   "tenant-1",
		Name: "Test Tenant",
		Quotas: &Quotas{
			MaxApplications: 5,
			MaxCPUMilli:     10000,
			MaxMemBytes:     100 << 30, // 100GB
		},
	}

	if err := manager.CreateTenant(tenant); err != nil {
		t.Fatalf("create tenant failed: %v", err)
	}

	// Record usage within quota
	delta := &Usage{
		Applications: 3,
		CPUMilli:     5000,
		MemBytes:     50 << 30,
	}

	if err := manager.RecordUsage("tenant-1", delta); err != nil {
		t.Fatalf("record usage failed: %v", err)
	}

	// Verify usage was recorded
	usage, err := manager.GetUsage("tenant-1")
	if err != nil {
		t.Fatalf("get usage failed: %v", err)
	}

	if usage.Applications != 3 || usage.CPUMilli != 5000 {
		t.Fatalf("usage mismatch: got %+v", usage)
	}

	// Try to exceed quota
	overQuota := &Usage{
		Applications: 3,
		CPUMilli:     6000,
	}

	if err := manager.RecordUsage("tenant-1", overQuota); err == nil {
		t.Fatal("expected error when exceeding CPU quota")
	}
}

func TestTenant_CheckQuota(t *testing.T) {
	ledger := &audit.Ledger{}
	rbacMgr := rbac.NewManager(ledger)
	manager := NewManager(rbacMgr, ledger)

	tenant := &Tenant{
		ID:   "tenant-1",
		Name: "Test Tenant",
		Quotas: &Quotas{
			MaxApplications: 10,
		},
	}

	if err := manager.CreateTenant(tenant); err != nil {
		t.Fatalf("create tenant failed: %v", err)
	}

	// Record some usage
	delta := &Usage{Applications: 8}
	if err := manager.RecordUsage("tenant-1", delta); err != nil {
		t.Fatalf("record usage failed: %v", err)
	}

	// Check if we can add 2 more (should be ok)
	checkDelta := &Usage{Applications: 2}
	if err := manager.CheckQuota("tenant-1", checkDelta); err != nil {
		t.Fatalf("check quota should pass: %v", err)
	}

	// Check if we can add 3 more (should fail)
	checkDelta = &Usage{Applications: 3}
	if err := manager.CheckQuota("tenant-1", checkDelta); err == nil {
		t.Fatal("check quota should fail when exceeding limit")
	}
}

func TestTenant_ListTenants(t *testing.T) {
	ledger := &audit.Ledger{}
	rbacMgr := rbac.NewManager(ledger)
	manager := NewManager(rbacMgr, ledger)

	tenantIDs := []string{"tenant-1", "tenant-2", "tenant-3"}

	for _, id := range tenantIDs {
		tenant := &Tenant{ID: id, Name: id}
		if err := manager.CreateTenant(tenant); err != nil {
			t.Fatalf("create tenant failed: %v", err)
		}
	}

	tenants := manager.ListTenants()
	if len(tenants) != len(tenantIDs) {
		t.Fatalf("expected %d tenants, got %d", len(tenantIDs), len(tenants))
	}
}

func TestTenant_DeleteTenant(t *testing.T) {
	ledger := &audit.Ledger{}
	rbacMgr := rbac.NewManager(ledger)
	manager := NewManager(rbacMgr, ledger)

	tenant := &Tenant{ID: "tenant-1", Name: "Test Tenant"}
	if err := manager.CreateTenant(tenant); err != nil {
		t.Fatalf("create tenant failed: %v", err)
	}

	// Delete should work if empty
	if err := manager.DeleteTenant("tenant-1"); err != nil {
		t.Fatalf("delete tenant failed: %v", err)
	}

	// Get should fail after delete
	_, err := manager.GetTenant("tenant-1")
	if err == nil {
		t.Fatal("get should fail after delete")
	}
}

func TestTenant_DeleteNonEmptyTenant(t *testing.T) {
	ledger := &audit.Ledger{}
	rbacMgr := rbac.NewManager(ledger)
	manager := NewManager(rbacMgr, ledger)

	tenant := &Tenant{ID: "tenant-1", Name: "Test Tenant"}
	if err := manager.CreateTenant(tenant); err != nil {
		t.Fatalf("create tenant failed: %v", err)
	}

	// Add some usage
	delta := &Usage{Applications: 1}
	if err := manager.RecordUsage("tenant-1", delta); err != nil {
		t.Fatalf("record usage failed: %v", err)
	}

	// Delete should fail because tenant is not empty
	if err := manager.DeleteTenant("tenant-1"); err == nil {
		t.Fatal("delete should fail for non-empty tenant")
	}
}

func TestTenant_NetworkPolicy(t *testing.T) {
	ledger := &audit.Ledger{}
	rbacMgr := rbac.NewManager(ledger)
	manager := NewManager(rbacMgr, ledger)

	tenant := &Tenant{ID: "tenant-1", Name: "Test Tenant"}
	if err := manager.CreateTenant(tenant); err != nil {
		t.Fatalf("create tenant failed: %v", err)
	}

	policy := &NetworkPolicy{
		TenantID:         "tenant-1",
		Name:             "default",
		AllowCrossTenant: true,
	}

	if err := manager.SetNetworkPolicy(policy); err != nil {
		t.Fatalf("set network policy failed: %v", err)
	}

	retrieved, err := manager.GetNetworkPolicy("tenant-1")
	if err != nil {
		t.Fatalf("get network policy failed: %v", err)
	}

	if !retrieved.AllowCrossTenant {
		t.Fatal("network policy not updated correctly")
	}
}

func TestTenant_CrossTenantEnforcement(t *testing.T) {
	ledger := &audit.Ledger{}
	rbacMgr := rbac.NewManager(ledger)
	manager := NewManager(rbacMgr, ledger)

	// Create two tenants
	t1 := &Tenant{ID: "tenant-1"}
	t2 := &Tenant{ID: "tenant-2"}

	if err := manager.CreateTenant(t1); err != nil {
		t.Fatalf("create tenant 1 failed: %v", err)
	}
	if err := manager.CreateTenant(t2); err != nil {
		t.Fatalf("create tenant 2 failed: %v", err)
	}

	// By default, cross-tenant should be denied
	err := manager.EnforceTenantIsolation("dh1actor123456789abcdefgh", "tenant-1", "tenant-2")
	if err == nil {
		t.Fatal("cross-tenant operation should be denied by default")
	}

	// Allow cross-tenant in tenant-1
	policy := &NetworkPolicy{
		TenantID:         "tenant-1",
		Name:             "default",
		AllowCrossTenant: true,
	}
	if err := manager.SetNetworkPolicy(policy); err != nil {
		t.Fatalf("set network policy failed: %v", err)
	}

	// Now should be allowed
	err = manager.EnforceTenantIsolation("dh1actor123456789abcdefgh", "tenant-1", "tenant-2")
	if err != nil {
		t.Fatalf("cross-tenant operation should be allowed: %v", err)
	}

	// Same tenant should always be allowed
	err = manager.EnforceTenantIsolation("dh1actor123456789abcdefgh", "tenant-1", "tenant-1")
	if err != nil {
		t.Fatalf("same-tenant operation should always be allowed: %v", err)
	}
}

func TestTenant_Suspend(t *testing.T) {
	ledger := &audit.Ledger{}
	rbacMgr := rbac.NewManager(ledger)
	manager := NewManager(rbacMgr, ledger)

	tenant := &Tenant{ID: "tenant-1", Name: "Test Tenant"}
	if err := manager.CreateTenant(tenant); err != nil {
		t.Fatalf("create tenant failed: %v", err)
	}

	if err := manager.SuspendTenant("tenant-1"); err != nil {
		t.Fatalf("suspend tenant failed: %v", err)
	}

	retrieved, err := manager.GetTenant("tenant-1")
	if err != nil {
		t.Fatalf("get tenant failed: %v", err)
	}

	if retrieved.Status != "suspended" {
		t.Fatalf("expected status 'suspended', got %q", retrieved.Status)
	}

	if err := manager.ResumeTenant("tenant-1"); err != nil {
		t.Fatalf("resume tenant failed: %v", err)
	}

	retrieved, err = manager.GetTenant("tenant-1")
	if err != nil {
		t.Fatalf("get tenant failed: %v", err)
	}

	if retrieved.Status != "active" {
		t.Fatalf("expected status 'active', got %q", retrieved.Status)
	}
}

func TestTenant_DefaultQuotas(t *testing.T) {
	ledger := &audit.Ledger{}
	rbacMgr := rbac.NewManager(ledger)
	manager := NewManager(rbacMgr, ledger)

	tenant := &Tenant{ID: "tenant-1", Name: "Test Tenant"}
	// Don't set quotas; should use defaults
	if err := manager.CreateTenant(tenant); err != nil {
		t.Fatalf("create tenant failed: %v", err)
	}

	retrieved, err := manager.GetTenant("tenant-1")
	if err != nil {
		t.Fatalf("get tenant failed: %v", err)
	}

	if retrieved.Quotas == nil {
		t.Fatal("quotas should be set to default")
	}

	if retrieved.Quotas.MaxApplications == 0 {
		t.Fatal("default quotas not applied")
	}
}

func TestTenant_RBACIntegration(t *testing.T) {
	ledger := &audit.Ledger{}
	rbacMgr := rbac.NewManager(ledger)
	manager := NewManager(rbacMgr, ledger)

	owner := "dh1owner123456789abcdefgh"
	tenant := &Tenant{
		ID:    "tenant-1",
		Name:  "Test Tenant",
		Owner: owner,
	}

	if err := manager.CreateTenant(tenant); err != nil {
		t.Fatalf("create tenant failed: %v", err)
	}

	// Owner should be bound as admin
	binding, err := rbacMgr.GetBinding(owner)
	if err != nil {
		t.Fatalf("get binding failed: %v", err)
	}

	if binding.Tenant != "tenant-1" {
		t.Fatalf("binding not scoped to tenant: got %q, want 'tenant-1'", binding.Tenant)
	}

	// Owner should have admin permissions within tenant
	allowed, _, err := rbacMgr.Check(owner, rbac.ResourceApplication, "app", rbac.PermCreate)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}

	if !allowed {
		t.Fatal("tenant owner should have admin permissions")
	}
}

func TestTenant_UsageTracking(t *testing.T) {
	ledger := &audit.Ledger{}
	rbacMgr := rbac.NewManager(ledger)
	manager := NewManager(rbacMgr, ledger)

	tenant := &Tenant{
		ID:   "tenant-1",
		Name: "Test Tenant",
		Quotas: &Quotas{
			MaxApplications: 100,
			MaxCPUMilli:     100000,
			MaxMemBytes:     1 << 40,
		},
	}

	if err := manager.CreateTenant(tenant); err != nil {
		t.Fatalf("create tenant failed: %v", err)
	}

	// Record multiple usage updates
	updates := []*Usage{
		{Applications: 2, CPUMilli: 1000},
		{Applications: 3, CPUMilli: 2000},
		{Applications: 1, CPUMilli: 500},
	}

	for _, delta := range updates {
		if err := manager.RecordUsage("tenant-1", delta); err != nil {
			t.Fatalf("record usage failed: %v", err)
		}
	}

	usage, err := manager.GetUsage("tenant-1")
	if err != nil {
		t.Fatalf("get usage failed: %v", err)
	}

	if usage.Applications != 6 || usage.CPUMilli != 3500 {
		t.Fatalf("usage not accumulated correctly: got %+v", usage)
	}
}
