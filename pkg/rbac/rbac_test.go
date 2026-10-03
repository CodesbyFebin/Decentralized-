package rbac

import (
	"testing"

	"decentralized.host/pkg/audit"
)

func TestRBAC_DefaultRoles(t *testing.T) {
	ledger := &audit.Ledger{}
	manager := NewManager(ledger)

	roles := manager.ListRoles()
	if len(roles) != 4 {
		t.Fatalf("expected 4 default roles, got %d", len(roles))
	}

	for _, name := range []string{RoleAdmin, RoleOperator, RoleUser, RoleAuditor} {
		role, err := manager.GetRole(name)
		if err != nil {
			t.Fatalf("failed to get role %q: %v", name, err)
		}
		if role.Name != name {
			t.Fatalf("role name mismatch: got %q, want %q", role.Name, name)
		}
	}
}

func TestRBAC_AdminRole(t *testing.T) {
	ledger := &audit.Ledger{}
	manager := NewManager(ledger)

	// Bind identity as admin
	binding := Binding{
		Identity: "dh1exampleadmin12345678abc",
		Roles:    []string{RoleAdmin},
	}
	if err := manager.Bind(binding); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	// Admin should be able to create, read, update, delete, execute applications
	tests := []struct {
		perm string
		ok   bool
	}{
		{PermCreate, true},
		{PermRead, true},
		{PermUpdate, true},
		{PermDelete, true},
		{PermExecute, true},
	}

	for _, test := range tests {
		allowed, reason, err := manager.Check("dh1exampleadmin12345678abc", ResourceApplication, "myapp", test.perm)
		if err != nil {
			t.Fatalf("check failed: %v", err)
		}
		if allowed != test.ok {
			t.Fatalf("permission %s: got %v (%s), want %v", test.perm, allowed, reason, test.ok)
		}
	}
}

func TestRBAC_OperatorRole(t *testing.T) {
	ledger := &audit.Ledger{}
	manager := NewManager(ledger)

	binding := Binding{
		Identity: "dh1exampleoperator12345abc",
		Roles:    []string{RoleOperator},
	}
	if err := manager.Bind(binding); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	tests := []struct {
		resource string
		perm     string
		ok       bool
	}{
		{ResourceApplication, PermCreate, true},
		{ResourceApplication, PermRead, true},
		{ResourceApplication, PermUpdate, true},
		{ResourceApplication, PermExecute, true},
		{ResourceApplication, PermDelete, false},
		{ResourceNode, PermDelete, false},
		{ResourceCluster, PermUpdate, false},
		{ResourceAuditLog, PermRead, true},
	}

	for _, test := range tests {
		allowed, reason, err := manager.Check("dh1exampleoperator12345abc", test.resource, "test", test.perm)
		if err != nil {
			t.Fatalf("check failed: %v", err)
		}
		if allowed != test.ok {
			t.Fatalf("%s %s: got %v (%s), want %v", test.resource, test.perm, allowed, reason, test.ok)
		}
	}
}

func TestRBAC_UserRole(t *testing.T) {
	ledger := &audit.Ledger{}
	manager := NewManager(ledger)

	binding := Binding{
		Identity: "dh1exampleuser123456789abc",
		Roles:    []string{RoleUser},
	}
	if err := manager.Bind(binding); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	tests := []struct {
		resource string
		perm     string
		ok       bool
	}{
		{ResourceApplication, PermRead, true},
		{ResourceApplication, PermExecute, true},
		{ResourceApplication, PermCreate, false},
		{ResourceApplication, PermDelete, false},
		{ResourceNode, PermRead, true},
		{ResourceNode, PermDelete, false},
	}

	for _, test := range tests {
		allowed, reason, err := manager.Check("dh1exampleuser123456789abc", test.resource, "test", test.perm)
		if err != nil {
			t.Fatalf("check failed: %v", err)
		}
		if allowed != test.ok {
			t.Fatalf("%s %s: got %v (%s), want %v", test.resource, test.perm, allowed, reason, test.ok)
		}
	}
}

func TestRBAC_AuditorRole(t *testing.T) {
	ledger := &audit.Ledger{}
	manager := NewManager(ledger)

	binding := Binding{
		Identity: "dh1exampleauditor1234567abc",
		Roles:    []string{RoleAuditor},
	}
	if err := manager.Bind(binding); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	tests := []struct {
		resource string
		perm     string
		ok       bool
	}{
		{ResourceAuditLog, PermRead, true},
		{ResourcePolicy, PermRead, true},
		{ResourceRole, PermRead, true},
		{ResourceAuditLog, PermWrite, false},
		{ResourceApplication, PermRead, false},
		{ResourceApplication, PermExecute, false},
	}

	for _, test := range tests {
		allowed, reason, err := manager.Check("dh1exampleauditor1234567abc", test.resource, "test", test.perm)
		if err != nil {
			t.Fatalf("check failed: %v", err)
		}
		if allowed != test.ok {
			t.Fatalf("%s %s: got %v (%s), want %v", test.resource, test.perm, allowed, reason, test.ok)
		}
	}
}

func TestRBAC_ExplicitGrant(t *testing.T) {
	ledger := &audit.Ledger{}
	manager := NewManager(ledger)

	// Bind user without admin role
	binding := Binding{
		Identity: "dh1exampleuser123456789abc",
		Roles:    []string{RoleUser},
	}
	if err := manager.Bind(binding); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	// User should not be able to delete applications
	allowed, _, err := manager.Check("dh1exampleuser123456789abc", ResourceApplication, "app1", PermDelete)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if allowed {
		t.Fatal("user should not have delete permission without grant")
	}

	// Grant explicit delete permission for specific app
	if err := manager.Grant("dh1exampleuser123456789abc", ResourceApplication+":app1", []string{PermDelete}); err != nil {
		t.Fatalf("grant failed: %v", err)
	}

	// Now user should be able to delete app1
	allowed, _, err = manager.Check("dh1exampleuser123456789abc", ResourceApplication, "app1", PermDelete)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !allowed {
		t.Fatal("user should have delete permission for app1 after grant")
	}

	// But should not be able to delete app2
	allowed, _, err = manager.Check("dh1exampleuser123456789abc", ResourceApplication, "app2", PermDelete)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if allowed {
		t.Fatal("user should not have delete permission for app2")
	}
}

func TestRBAC_ExplicitDenial(t *testing.T) {
	ledger := &audit.Ledger{}
	manager := NewManager(ledger)

	// Bind as operator (can create apps)
	binding := Binding{
		Identity: "dh1exampleoperator12345abc",
		Roles:    []string{RoleOperator},
	}
	if err := manager.Bind(binding); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	// Operator should be able to create applications
	allowed, _, err := manager.Check("dh1exampleoperator12345abc", ResourceApplication, "app1", PermCreate)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !allowed {
		t.Fatal("operator should have create permission")
	}

	// Deny creation for specific app
	if err := manager.Deny("dh1exampleoperator12345abc", ResourceApplication+":forbidden", []string{PermCreate}); err != nil {
		t.Fatalf("deny failed: %v", err)
	}

	// Now operator should not be able to create "forbidden" app (even though role allows it)
	allowed, _, err = manager.Check("dh1exampleoperator12345abc", ResourceApplication, "forbidden", PermCreate)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if allowed {
		t.Fatal("operator should not have create permission for forbidden app (deny overrides grant)")
	}

	// But should be able to create other apps
	allowed, _, err = manager.Check("dh1exampleoperator12345abc", ResourceApplication, "allowed", PermCreate)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !allowed {
		t.Fatal("operator should have create permission for allowed app")
	}
}

func TestRBAC_UnknownIdentity(t *testing.T) {
	ledger := &audit.Ledger{}
	manager := NewManager(ledger)

	allowed, reason, err := manager.Check("dh1unknownidentity123456abc", ResourceApplication, "app", PermRead)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if allowed {
		t.Fatal("unknown identity should not have any permissions")
	}
	if reason == "" {
		t.Fatal("reason should not be empty for denied access")
	}
}

func TestRBAC_Unbind(t *testing.T) {
	ledger := &audit.Ledger{}
	manager := NewManager(ledger)

	binding := Binding{
		Identity: "dh1exampleadmin12345678abc",
		Roles:    []string{RoleAdmin},
	}
	if err := manager.Bind(binding); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	// Verify admin access works
	allowed, _, err := manager.Check("dh1exampleadmin12345678abc", ResourceApplication, "app", PermDelete)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !allowed {
		t.Fatal("admin should have delete permission")
	}

	// Unbind the identity
	if err := manager.Unbind("dh1exampleadmin12345678abc"); err != nil {
		t.Fatalf("unbind failed: %v", err)
	}

	// Now should not have any permissions
	allowed, _, err = manager.Check("dh1exampleadmin12345678abc", ResourceApplication, "app", PermDelete)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if allowed {
		t.Fatal("unbound identity should not have any permissions")
	}
}

func TestRBAC_ListBindings(t *testing.T) {
	ledger := &audit.Ledger{}
	manager := NewManager(ledger)

	identities := []string{
		"dh1exampleadmin12345678abc",
		"dh1exampleoperator12345abc",
		"dh1exampleuser123456789abc",
	}

	for _, id := range identities {
		binding := Binding{
			Identity: id,
			Roles:    []string{RoleUser},
		}
		if err := manager.Bind(binding); err != nil {
			t.Fatalf("bind failed: %v", err)
		}
	}

	bindings := manager.ListBindings()
	if len(bindings) != len(identities) {
		t.Fatalf("expected %d bindings, got %d", len(identities), len(bindings))
	}
}

func TestRBAC_MultipleRoles(t *testing.T) {
	ledger := &audit.Ledger{}
	manager := NewManager(ledger)

	// Bind with multiple roles
	binding := Binding{
		Identity: "dh1exampleuser123456789abc",
		Roles:    []string{RoleUser, RoleAuditor},
	}
	if err := manager.Bind(binding); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	// Should have permissions from both roles
	tests := []struct {
		resource string
		perm     string
		ok       bool
	}{
		{ResourceApplication, PermRead, true},   // from RoleUser
		{ResourceApplication, PermExecute, true}, // from RoleUser
		{ResourceAuditLog, PermRead, true},       // from RoleAuditor
		{ResourceApplication, PermCreate, false}, // neither role allows
	}

	for _, test := range tests {
		allowed, _, err := manager.Check("dh1exampleuser123456789abc", test.resource, "test", test.perm)
		if err != nil {
			t.Fatalf("check failed: %v", err)
		}
		if allowed != test.ok {
			t.Fatalf("%s %s: got %v, want %v", test.resource, test.perm, allowed, test.ok)
		}
	}
}

func TestRBAC_AuditLogging(t *testing.T) {
	ledger := &audit.Ledger{}
	manager := NewManager(ledger)

	binding := Binding{
		Identity: "dh1exampleadmin12345678abc",
		Roles:    []string{RoleAdmin},
	}
	if err := manager.Bind(binding); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	// Perform some operations
	manager.Check("dh1exampleadmin12345678abc", ResourceApplication, "app1", PermCreate)
	manager.Grant("dh1exampleadmin12345678abc", ResourceApplication+":app2", []string{PermDelete})

	// Verify audit trail has entries
	if len(ledger.Entries) < 3 {
		t.Fatalf("expected at least 3 audit entries (bind, check, grant), got %d", len(ledger.Entries))
	}
}

func TestRBAC_WildcardMatching(t *testing.T) {
	ledger := &audit.Ledger{}
	manager := NewManager(ledger)

	binding := Binding{
		Identity: "dh1exampleuser123456789abc",
		Roles:    []string{RoleUser},
		Grants: []Grant{
			{Resource: "application:*", Perms: []string{PermDelete}},
		},
	}
	if err := manager.Bind(binding); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	// User should be able to delete any application
	for _, app := range []string{"app1", "app2", "app3"} {
		allowed, _, err := manager.Check("dh1exampleuser123456789abc", ResourceApplication, app, PermDelete)
		if err != nil {
			t.Fatalf("check failed: %v", err)
		}
		if !allowed {
			t.Fatalf("user should have delete permission for %s via wildcard grant", app)
		}
	}
}

func TestRBAC_CustomRole(t *testing.T) {
	ledger := &audit.Ledger{}
	manager := NewManager(ledger)

	// Define a custom role
	customRole := &RoleDefinition{
		Name:        "developer",
		Description: "Developer with deployment rights",
		Permissions: map[string][]string{
			ResourceApplication: {PermCreate, PermRead, PermUpdate, PermExecute},
			ResourceVolume:      {PermCreate, PermRead},
		},
	}

	if err := manager.DefineRole(customRole); err != nil {
		t.Fatalf("define role failed: %v", err)
	}

	// Bind identity with custom role
	binding := Binding{
		Identity: "dh1exampledeveloper1234abc",
		Roles:    []string{"developer"},
	}
	if err := manager.Bind(binding); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	// Verify permissions
	tests := []struct {
		resource string
		perm     string
		ok       bool
	}{
		{ResourceApplication, PermCreate, true},
		{ResourceApplication, PermDelete, false},
		{ResourceVolume, PermCreate, true},
		{ResourceVolume, PermDelete, false},
	}

	for _, test := range tests {
		allowed, _, err := manager.Check("dh1exampledeveloper1234abc", test.resource, "test", test.perm)
		if err != nil {
			t.Fatalf("check failed: %v", err)
		}
		if allowed != test.ok {
			t.Fatalf("%s %s: got %v, want %v", test.resource, test.perm, allowed, test.ok)
		}
	}
}

const PermWrite = "write" // for testing invalid permissions
