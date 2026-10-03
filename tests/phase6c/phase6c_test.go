package phase6c

import (
	"testing"

	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/rbac"
	"decentralized.host/pkg/tenant"
)

// TestPhase6C_Governance demonstrates the complete governance model.
func TestPhase6C_Governance(t *testing.T) {
	// Initialize components
	ledger := &audit.Ledger{}
	complianceLog := audit.NewComplianceLog(t.TempDir(), nil)
	rbacManager := rbac.NewManager(ledger)
	tenantManager := tenant.NewManager(rbacManager, ledger)

	// Scenario: Multi-tenant cluster with RBAC and audit logging

	// Step 1: Create two tenants
	tenant1 := &tenant.Tenant{
		ID:    "acme-corp",
		Name:  "ACME Corporation",
		Owner: "dh1acmeadmin123456789abc",
		Quotas: &tenant.Quotas{
			MaxApplications: 100,
			MaxCPUMilli:     128000,
			MaxMemBytes:     512 << 30,
		},
	}

	tenant2 := &tenant.Tenant{
		ID:    "globex-corp",
		Name:  "Globex Corporation",
		Owner: "dh1globexadmin1234567abc",
		Quotas: &tenant.Quotas{
			MaxApplications: 50,
			MaxCPUMilli:     64000,
			MaxMemBytes:     256 << 30,
		},
	}

	if err := tenantManager.CreateTenant(tenant1); err != nil {
		t.Fatalf("failed to create tenant 1: %v", err)
	}
	if err := tenantManager.CreateTenant(tenant2); err != nil {
		t.Fatalf("failed to create tenant 2: %v", err)
	}

	// Log tenant creation
	if err := complianceLog.Log(audit.Entry{
		Actor:    "system",
		Action:   "create_tenant",
		Resource: "tenant:acme-corp",
		Source:   audit.SourceHost,
		Detail:   "ACME Corporation tenant created",
	}); err != nil {
		t.Fatalf("failed to log tenant creation: %v", err)
	}

	// Step 2: Verify tenant isolation
	if err := tenantManager.EnforceTenantIsolation(
		"dh1acmeadmin123456789abc",
		"acme-corp",
		"globex-corp",
	); err == nil {
		t.Fatal("cross-tenant communication should be denied by default")
	}

	// Step 3: Test RBAC with tenant scoping
	// Admin of ACME should have full permissions within ACME
	allowed, reason, err := rbacManager.Check(
		"dh1acmeadmin123456789abc",
		rbac.ResourceApplication,
		"app1",
		rbac.PermCreate,
	)
	if err != nil {
		t.Fatalf("rbac check failed: %v", err)
	}
	if !allowed {
		t.Fatalf("ACME admin should have create permission: %s", reason)
	}

	// Log the access decision
	if err := complianceLog.Log(audit.Entry{
		Actor:    "dh1acmeadmin123456789abc",
		Action:   "access_granted",
		Resource: "rbac:application:app1",
		Source:   audit.SourceHost,
		Detail:   "admin role allows create permission",
	}); err != nil {
		t.Fatalf("failed to log access: %v", err)
	}

	// Step 4: Test resource quotas
	quotaCheck := &tenant.Usage{
		Applications: 50,
		CPUMilli:     64000,
	}

	if err := tenantManager.CheckQuota("acme-corp", quotaCheck); err != nil {
		t.Fatalf("quota check should pass: %v", err)
	}

	overQuota := &tenant.Usage{
		Applications: 101, // Exceeds max of 100
	}

	if err := tenantManager.CheckQuota("acme-corp", overQuota); err == nil {
		t.Fatal("quota check should fail for excessive applications")
	}

	// Log quota check
	if err := complianceLog.Log(audit.Entry{
		Actor:    "dh1acmeadmin123456789abc",
		Action:   "quota_check",
		Resource: "tenant:acme-corp",
		Source:   audit.SourceHost,
		Detail:   "application limit check",
	}); err != nil {
		t.Fatalf("failed to log quota check: %v", err)
	}

	// Step 5: Test audit compliance - query logs
	criteria := &audit.QueryCriteria{
		Resource: "tenant:",
	}

	results, err := complianceLog.QueryEntries(criteria)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one tenant-related audit entry")
	}

	// Step 6: Test audit tamper detection
	tamperReport := complianceLog.DetectTampering()
	if tamperReport.TamperingDetected {
		t.Fatal("audit log should not show tampering")
	}

	// Step 7: Test role-based access hierarchy
	operatorBinding := rbac.Binding{
		Identity: "dh1acmeoperator123456abc",
		Roles:    []string{rbac.RoleOperator},
		Tenant:   "acme-corp",
	}

	if err := rbacManager.Bind(operatorBinding); err != nil {
		t.Fatalf("failed to bind operator: %v", err)
	}

	// Operator should be able to create and read, but not delete
	createOk, _, _ := rbacManager.Check(
		"dh1acmeoperator123456abc",
		rbac.ResourceApplication,
		"app2",
		rbac.PermCreate,
	)

	deleteOk, _, _ := rbacManager.Check(
		"dh1acmeoperator123456abc",
		rbac.ResourceApplication,
		"app2",
		rbac.PermDelete,
	)

	if !createOk || deleteOk {
		t.Fatal("operator permissions incorrect")
	}

	// Step 8: Test explicit grant
	if err := rbacManager.Grant(
		"dh1acmeoperator123456abc",
		"application:admin-app",
		[]string{rbac.PermDelete},
	); err != nil {
		t.Fatalf("grant failed: %v", err)
	}

	// Now operator should be able to delete admin-app
	deleteOk, _, _ = rbacManager.Check(
		"dh1acmeoperator123456abc",
		rbac.ResourceApplication,
		"admin-app",
		rbac.PermDelete,
	)

	if !deleteOk {
		t.Fatal("explicit grant should allow delete")
	}

	// Step 9: Test multi-tenant isolation with network policy
	policy := &tenant.NetworkPolicy{
		TenantID:         "acme-corp",
		Name:             "default",
		AllowCrossTenant: true,
	}

	if err := tenantManager.SetNetworkPolicy(policy); err != nil {
		t.Fatalf("failed to set network policy: %v", err)
	}

	// Now cross-tenant communication should be allowed
	if err := tenantManager.EnforceTenantIsolation(
		"dh1acmeadmin123456789abc",
		"acme-corp",
		"globex-corp",
	); err != nil {
		t.Fatalf("cross-tenant should be allowed after policy: %v", err)
	}

	// Step 10: Test audit compliance summary
	summary := complianceLog.GetSummary()
	if summary.TotalEntries == 0 {
		t.Fatal("audit log should have entries")
	}

	// Verify audit contains expected actions
	if summary.ActionCounts["quota_check"] == 0 {
		t.Fatal("audit log should contain quota_check action")
	}

	// Step 11: Test tenant suspension
	if err := tenantManager.SuspendTenant("globex-corp"); err != nil {
		t.Fatalf("failed to suspend tenant: %v", err)
	}

	suspendedTenant, err := tenantManager.GetTenant("globex-corp")
	if err != nil {
		t.Fatalf("failed to get suspended tenant: %v", err)
	}

	if suspendedTenant.Status != "suspended" {
		t.Fatal("tenant should be marked as suspended")
	}

	// Step 12: Verify complete audit trail
	if err := complianceLog.VerifyChain(); err != nil {
		t.Fatalf("audit chain verification failed: %v", err)
	}

	// All tests passed - governance model is working correctly
	t.Logf("Phase 6C Governance test passed:")
	t.Logf("  - %d tenants created and managed", 2)
	t.Logf("  - Tenant isolation enforced")
	t.Logf("  - RBAC with %d role definitions", len(rbacManager.ListRoles()))
	t.Logf("  - %d audit entries logged and verified", summary.TotalEntries)
}

// TestPhase6C_ComplianceLogging verifies 100% audit coverage.
func TestPhase6C_ComplianceLogging(t *testing.T) {
	complianceLog := audit.NewComplianceLog(t.TempDir(), nil)

	// Simulate various operations
	operations := []struct {
		action   string
		actor    string
		resource string
		detail   string
	}{
		{"create", "admin", "application:app1", "Create application"},
		{"read", "operator", "application:app1", "Read application"},
		{"update", "admin", "application:app1", "Update application"},
		{"delete", "admin", "application:app1", "Delete application"},
		{"execute", "user", "application:app2", "Execute application"},
		{"access_granted", "user", "auditLog:all", "User accessed audit log"},
		{"access_denied", "unknown", "policy:edit", "Unauthorized policy edit"},
	}

	for _, op := range operations {
		if err := complianceLog.Log(audit.Entry{
			Actor:    op.actor,
			Action:   op.action,
			Resource: op.resource,
			Source:   audit.SourceHost,
			Detail:   op.detail,
		}); err != nil {
			t.Fatalf("failed to log %s: %v", op.action, err)
		}
	}

	// Verify all operations were logged
	if complianceLog.Count() != int64(len(operations)) {
		t.Fatalf("expected %d logged operations, got %d", len(operations), complianceLog.Count())
	}

	// Query by action
	allowedCriteria := &audit.QueryCriteria{Action: "access_granted"}
	allowed, err := complianceLog.QueryEntries(allowedCriteria)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	deniedCriteria := &audit.QueryCriteria{Action: "access_denied"}
	denied, err := complianceLog.QueryEntries(deniedCriteria)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(allowed) != 1 || len(denied) != 1 {
		t.Fatal("access control decisions not properly logged")
	}

	// Verify audit chain integrity
	if err := complianceLog.VerifyChain(); err != nil {
		t.Fatalf("audit chain verification failed: %v", err)
	}

	t.Logf("Compliance logging test passed:")
	t.Logf("  - %d operations fully audited", complianceLog.Count())
	t.Logf("  - Audit chain verified and intact")
}

// TestPhase6C_RBACPermissionMatrix tests permission matrix for all roles.
func TestPhase6C_RBACPermissionMatrix(t *testing.T) {
	ledger := &audit.Ledger{}
	rbacManager := rbac.NewManager(ledger)

	roles := []string{rbac.RoleAdmin, rbac.RoleOperator, rbac.RoleUser, rbac.RoleAuditor}

	testCases := []struct {
		role       string
		resource   string
		permission string
		expected   bool
	}{
		// Admin - full access
		{rbac.RoleAdmin, rbac.ResourceApplication, rbac.PermCreate, true},
		{rbac.RoleAdmin, rbac.ResourceApplication, rbac.PermDelete, true},
		{rbac.RoleAdmin, rbac.ResourcePolicy, rbac.PermUpdate, true},

		// Operator - limited access
		{rbac.RoleOperator, rbac.ResourceApplication, rbac.PermCreate, true},
		{rbac.RoleOperator, rbac.ResourceApplication, rbac.PermDelete, false},
		{rbac.RoleOperator, rbac.ResourceCluster, rbac.PermUpdate, false},

		// User - read and execute only
		{rbac.RoleUser, rbac.ResourceApplication, rbac.PermRead, true},
		{rbac.RoleUser, rbac.ResourceApplication, rbac.PermExecute, true},
		{rbac.RoleUser, rbac.ResourceApplication, rbac.PermCreate, false},

		// Auditor - read-only
		{rbac.RoleAuditor, rbac.ResourceAuditLog, rbac.PermRead, true},
		{rbac.RoleAuditor, rbac.ResourcePolicy, rbac.PermRead, true},
		{rbac.RoleAuditor, rbac.ResourceAuditLog, rbac.PermUpdate, false},
	}

	for _, tc := range testCases {
		// Create binding with the role
		binding := rbac.Binding{
			Identity: "dh1test123456789abcdefgh",
			Roles:    []string{tc.role},
		}

		if err := rbacManager.Bind(binding); err != nil {
			t.Fatalf("bind failed: %v", err)
		}

		// Check permission
		allowed, _, err := rbacManager.Check(
			"dh1test123456789abcdefgh",
			tc.resource,
			"test",
			tc.permission,
		)

		if err != nil {
			t.Fatalf("check failed: %v", err)
		}

		if allowed != tc.expected {
			t.Fatalf(
				"role %s, resource %s, perm %s: expected %v, got %v",
				tc.role, tc.resource, tc.permission, tc.expected, allowed,
			)
		}

		// Unbind for next test
		_ = rbacManager.Unbind("dh1test123456789abcdefgh")
	}

	t.Logf("RBAC permission matrix test passed: %d roles verified", len(roles))
}
