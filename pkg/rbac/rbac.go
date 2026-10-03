// Package rbac implements role-based access control for dh/v1 with audit coverage.
//
// The RBAC system defines roles (admin, operator, user, auditor), permissions
// (create, read, update, delete, execute), and enforces resource-scoped access
// decisions. All decisions are logged to the audit trail.
package rbac

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"decentralized.host/pkg/audit"
)

// Role names defined by dh/v1 RBAC.
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleUser     = "user"
	RoleAuditor  = "auditor"
)

// Permissions are the atomic actions that can be allowed or denied.
const (
	PermCreate = "create"
	PermRead   = "read"
	PermUpdate = "update"
	PermDelete = "delete"
	PermExecute = "execute"
)

// ResourceTypes define the scope of access control.
const (
	ResourceApplication = "application"
	ResourceNode        = "node"
	ResourceCluster     = "cluster"
	ResourceNamespace   = "namespace"
	ResourceAuditLog    = "auditLog"
	ResourcePolicy      = "policy"
	ResourceRole        = "role"
	ResourceUser        = "user"
	ResourceSecret      = "secret"
	ResourceVolume      = "volume"
)

// Binding associates an identity with a set of role grants and restrictions.
type Binding struct {
	Identity  string        `json:"identity"`  // dh1 identity
	Roles     []string      `json:"roles"`     // assigned roles
	Grants    []Grant       `json:"grants"`    // explicit grants
	Denials   []Denial      `json:"denials"`   // explicit denials (override grants)
	Tenant    string        `json:"tenant"`    // namespace for multi-tenancy
	CreatedAt int64         `json:"createdAt"` // unix milliseconds
	UpdatedAt int64         `json:"updatedAt"` // unix milliseconds
}

// Grant is an explicit permission grant.
type Grant struct {
	Resource string   `json:"resource"`       // e.g., "application:myapp"
	Perms    []string `json:"permissions"`    // PermCreate, PermRead, ...
}

// Denial is an explicit permission denial (overrides grants).
type Denial struct {
	Resource string   `json:"resource"`
	Perms    []string `json:"permissions"`
}

// Decision records the result of an access control decision.
type Decision struct {
	Actor      string    `json:"actor"`      // dh1 identity
	Resource   string    `json:"resource"`   // resource identifier
	Permission string    `json:"permission"` // requested permission
	Allowed    bool      `json:"allowed"`
	Reason     string    `json:"reason"`
	Timestamp  int64     `json:"timestamp"`
}

// RoleDefinition defines the permissions granted by a role.
type RoleDefinition struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Permissions map[string][]string `json:"permissions"` // resource type -> []permissions
}

// Manager implements RBAC enforcement with audit integration.
type Manager struct {
	mu       sync.RWMutex
	bindings map[string]*Binding // identity -> Binding
	roles    map[string]*RoleDefinition
	ledger   *audit.Ledger
}

// NewManager creates a new RBAC manager with default role definitions.
func NewManager(ledger *audit.Ledger) *Manager {
	m := &Manager{
		bindings: make(map[string]*Binding),
		roles:    make(map[string]*RoleDefinition),
		ledger:   ledger,
	}
	m.registerDefaultRoles()
	return m
}

// registerDefaultRoles sets up the built-in roles.
func (m *Manager) registerDefaultRoles() {
	m.roles[RoleAdmin] = &RoleDefinition{
		Name:        RoleAdmin,
		Description: "Full cluster access",
		Permissions: map[string][]string{
			ResourceApplication: {PermCreate, PermRead, PermUpdate, PermDelete, PermExecute},
			ResourceNode:        {PermCreate, PermRead, PermUpdate, PermDelete},
			ResourceCluster:     {PermCreate, PermRead, PermUpdate, PermDelete},
			ResourceNamespace:   {PermCreate, PermRead, PermUpdate, PermDelete},
			ResourceAuditLog:    {PermRead},
			ResourcePolicy:      {PermCreate, PermRead, PermUpdate, PermDelete},
			ResourceRole:        {PermCreate, PermRead, PermUpdate, PermDelete},
			ResourceUser:        {PermCreate, PermRead, PermUpdate, PermDelete},
			ResourceSecret:      {PermCreate, PermRead, PermUpdate, PermDelete},
			ResourceVolume:      {PermCreate, PermRead, PermUpdate, PermDelete},
		},
	}

	m.roles[RoleOperator] = &RoleDefinition{
		Name:        RoleOperator,
		Description: "Deploy and manage applications",
		Permissions: map[string][]string{
			ResourceApplication: {PermCreate, PermRead, PermUpdate, PermExecute},
			ResourceNode:        {PermRead},
			ResourceCluster:     {PermRead},
			ResourceNamespace:   {PermRead},
			ResourceAuditLog:    {PermRead},
			ResourceVolume:      {PermCreate, PermRead, PermUpdate},
		},
	}

	m.roles[RoleUser] = &RoleDefinition{
		Name:        RoleUser,
		Description: "View and execute applications",
		Permissions: map[string][]string{
			ResourceApplication: {PermRead, PermExecute},
			ResourceNode:        {PermRead},
			ResourceCluster:     {PermRead},
		},
	}

	m.roles[RoleAuditor] = &RoleDefinition{
		Name:        RoleAuditor,
		Description: "Read-only access to audit logs and policies",
		Permissions: map[string][]string{
			ResourceAuditLog: {PermRead},
			ResourcePolicy:   {PermRead},
			ResourceRole:     {PermRead},
		},
	}
}

// Bind assigns roles and explicit grants to an identity.
func (m *Manager) Bind(binding Binding) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate roles exist
	for _, role := range binding.Roles {
		if _, ok := m.roles[role]; !ok {
			return fmt.Errorf("rbac: unknown role %q", role)
		}
	}

	now := time.Now().UnixMilli()
	binding.CreatedAt = now
	binding.UpdatedAt = now
	m.bindings[binding.Identity] = &binding

	if m.ledger != nil {
		m.ledger.Append(audit.Entry{
			Action:   "bind",
			Actor:    "system",
			Source:   audit.SourceHost,
			Resource: fmt.Sprintf("rbac:binding:%s", binding.Identity),
			Detail:   fmt.Sprintf("roles=%v tenant=%s", binding.Roles, binding.Tenant),
		})
	}

	return nil
}

// Unbind removes all roles from an identity.
func (m *Manager) Unbind(identity string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.bindings[identity]; !ok {
		return fmt.Errorf("rbac: identity %q not bound", identity)
	}

	delete(m.bindings, identity)

	if m.ledger != nil {
		m.ledger.Append(audit.Entry{
			Action:   "unbind",
			Actor:    "system",
			Source:   audit.SourceHost,
			Resource: fmt.Sprintf("rbac:binding:%s", identity),
		})
	}

	return nil
}

// Check evaluates whether an actor can perform an action on a resource.
// All decisions are logged to the audit trail.
func (m *Manager) Check(actor, resourceType, resourceID, permission string) (bool, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	binding, ok := m.bindings[actor]
	if !ok {
		reason := fmt.Sprintf("identity %q has no binding", actor)
		m.logDecision(actor, resourceType+":"+resourceID, permission, false, reason)
		return false, reason, nil
	}

	// Check explicit denials first (deny always wins)
	for _, denial := range binding.Denials {
		if m.resourceMatches(denial.Resource, resourceType, resourceID) {
			if slices.Contains(denial.Perms, permission) {
				reason := fmt.Sprintf("explicitly denied by policy")
				m.logDecision(actor, resourceType+":"+resourceID, permission, false, reason)
				return false, reason, nil
			}
		}
	}

	// Check explicit grants
	for _, grant := range binding.Grants {
		if m.resourceMatches(grant.Resource, resourceType, resourceID) {
			if slices.Contains(grant.Perms, permission) {
				m.logDecision(actor, resourceType+":"+resourceID, permission, true, "explicit grant")
				return true, "explicit grant", nil
			}
		}
	}

	// Check role permissions
	for _, roleName := range binding.Roles {
		role, ok := m.roles[roleName]
		if !ok {
			continue
		}

		perms, ok := role.Permissions[resourceType]
		if ok && slices.Contains(perms, permission) {
			m.logDecision(actor, resourceType+":"+resourceID, permission, true, fmt.Sprintf("role %q", roleName))
			return true, fmt.Sprintf("role %q", roleName), nil
		}
	}

	reason := fmt.Sprintf("no permission from roles or grants")
	m.logDecision(actor, resourceType+":"+resourceID, permission, false, reason)
	return false, reason, nil
}

// resourceMatches checks if a policy resource selector matches a specific resource.
// Supports wildcards: "application:*" matches all applications, "*" matches all.
func (m *Manager) resourceMatches(selector, resourceType, resourceID string) bool {
	if selector == "*" {
		return true
	}
	if selector == resourceType+":*" {
		return true
	}
	return selector == resourceType+":"+resourceID
}

// logDecision logs an access control decision to the audit trail.
func (m *Manager) logDecision(actor, resource, permission string, allowed bool, reason string) {
	if m.ledger == nil {
		return
	}

	action := "access_denied"
	if allowed {
		action = "access_granted"
	}

	m.ledger.Append(audit.Entry{
		Action:   action,
		Actor:    actor,
		Source:   audit.SourceHost,
		Resource: resource,
		Detail:   fmt.Sprintf("permission=%s reason=%s", permission, reason),
	})
}

// GetBinding retrieves the role binding for an identity.
func (m *Manager) GetBinding(identity string) (*Binding, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	binding, ok := m.bindings[identity]
	if !ok {
		return nil, fmt.Errorf("rbac: binding not found for %q", identity)
	}
	return binding, nil
}

// ListBindings returns all role bindings.
func (m *Manager) ListBindings() []*Binding {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bindings := make([]*Binding, 0, len(m.bindings))
	for _, b := range m.bindings {
		bindings = append(bindings, b)
	}
	return bindings
}

// GetRole returns the definition of a role.
func (m *Manager) GetRole(name string) (*RoleDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	role, ok := m.roles[name]
	if !ok {
		return nil, fmt.Errorf("rbac: role %q not found", name)
	}
	return role, nil
}

// ListRoles returns all role definitions.
func (m *Manager) ListRoles() []*RoleDefinition {
	m.mu.RLock()
	defer m.mu.RUnlock()

	roles := make([]*RoleDefinition, 0, len(m.roles))
	for _, r := range m.roles {
		roles = append(roles, r)
	}
	return roles
}

// DefineRole registers a custom role.
func (m *Manager) DefineRole(role *RoleDefinition) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if role.Name == "" {
		return fmt.Errorf("rbac: role name cannot be empty")
	}

	m.roles[role.Name] = role

	if m.ledger != nil {
		m.ledger.Append(audit.Entry{
			Action:   "define_role",
			Actor:    "system",
			Source:   audit.SourceHost,
			Resource: fmt.Sprintf("rbac:role:%s", role.Name),
			Detail:   role.Description,
		})
	}

	return nil
}

// Grant adds an explicit permission grant to a binding.
func (m *Manager) Grant(identity, resource string, perms []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	binding, ok := m.bindings[identity]
	if !ok {
		return fmt.Errorf("rbac: identity %q not bound", identity)
	}

	binding.Grants = append(binding.Grants, Grant{Resource: resource, Perms: perms})
	binding.UpdatedAt = time.Now().UnixMilli()

	if m.ledger != nil {
		m.ledger.Append(audit.Entry{
			Action:   "grant",
			Actor:    "system",
			Source:   audit.SourceHost,
			Resource: fmt.Sprintf("rbac:binding:%s", identity),
			Detail:   fmt.Sprintf("resource=%s perms=%v", resource, perms),
		})
	}

	return nil
}

// Deny adds an explicit permission denial to a binding.
func (m *Manager) Deny(identity, resource string, perms []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	binding, ok := m.bindings[identity]
	if !ok {
		return fmt.Errorf("rbac: identity %q not bound", identity)
	}

	binding.Denials = append(binding.Denials, Denial{Resource: resource, Perms: perms})
	binding.UpdatedAt = time.Now().UnixMilli()

	if m.ledger != nil {
		m.ledger.Append(audit.Entry{
			Action:   "deny",
			Actor:    "system",
			Source:   audit.SourceHost,
			Resource: fmt.Sprintf("rbac:binding:%s", identity),
			Detail:   fmt.Sprintf("resource=%s perms=%v", resource, perms),
		})
	}

	return nil
}
