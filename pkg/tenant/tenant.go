// Package tenant implements namespace-based multi-tenant isolation for dh/v1.
//
// Tenants are isolated namespaces with their own resource quotas, network policies,
// and audit separation. All cross-tenant operations are audited and blocked.
package tenant

import (
	"fmt"
	"sync"
	"time"

	"decentralized.host/pkg/audit"
	"decentralized.host/pkg/rbac"
)

// Tenant represents an isolated namespace with quotas and policies.
type Tenant struct {
	ID          string             `json:"id"`          // unique tenant identifier
	Name        string             `json:"name"`        // human-readable name
	Owner       string             `json:"owner"`       // owning operator (dh1 identity)
	Description string             `json:"description"`
	Quotas      *Quotas            `json:"quotas"`
	Status      string             `json:"status"`      // active | suspended | archived
	CreatedAt   int64              `json:"createdAt"`   // unix milliseconds
	UpdatedAt   int64              `json:"updatedAt"`   // unix milliseconds
	Labels      map[string]string  `json:"labels"`      // arbitrary key-value tags
}

// Quotas enforces resource limits per tenant.
type Quotas struct {
	MaxApplications int64 `json:"maxApplications"`
	MaxCPUMilli     int64 `json:"maxCPUMilli"`
	MaxMemBytes     int64 `json:"maxMemBytes"`
	MaxStorageBytes int64 `json:"maxStorageBytes"`
	MaxNetworkMbps  int64 `json:"maxNetworkMbps"`
}

// Usage tracks resource consumption for a tenant.
type Usage struct {
	Applications int64 `json:"applications"`
	CPUMilli     int64 `json:"cpuMilli"`
	MemBytes     int64 `json:"memBytes"`
	StorageBytes int64 `json:"storageBytes"`
	NetworkMbps  int64 `json:"networkMbps"`
}

// NetworkPolicy controls inter-tenant and intra-tenant network connectivity.
type NetworkPolicy struct {
	TenantID       string              `json:"tenantId"`
	Name           string              `json:"name"`
	EgressRules    []EgressRule        `json:"egressRules"`
	IngressRules   []IngressRule       `json:"ingressRules"`
	AllowCrossTenant bool              `json:"allowCrossTenant"`
}

// EgressRule permits outbound traffic.
type EgressRule struct {
	Destinations []string `json:"destinations"` // CIDR blocks or "external"
	Ports        []int64  `json:"ports"`
	Protocol     string   `json:"protocol"`     // tcp | udp | icmp
}

// IngressRule permits inbound traffic.
type IngressRule struct {
	Sources []string `json:"sources"` // CIDR blocks, tenant IDs, or "external"
	Ports   []int64  `json:"ports"`
	Protocol string  `json:"protocol"`
}

// Manager implements multi-tenant isolation and enforcement.
type Manager struct {
	mu             sync.RWMutex
	tenants        map[string]*Tenant            // id -> Tenant
	usage          map[string]*Usage              // id -> Usage
	policies       map[string]*NetworkPolicy      // id -> policy
	rbacManager    *rbac.Manager
	ledger         *audit.Ledger
	defaultQuotas  *Quotas
}

// NewManager creates a new tenant manager.
func NewManager(rbacMgr *rbac.Manager, ledger *audit.Ledger) *Manager {
	return &Manager{
		tenants:   make(map[string]*Tenant),
		usage:     make(map[string]*Usage),
		policies:  make(map[string]*NetworkPolicy),
		rbacManager: rbacMgr,
		ledger:    ledger,
		defaultQuotas: &Quotas{
			MaxApplications: 100,
			MaxCPUMilli:     64000,
			MaxMemBytes:     512 << 30,
			MaxStorageBytes: 100 << 30,
			MaxNetworkMbps:  10000,
		},
	}
}

// CreateTenant creates a new tenant namespace.
func (m *Manager) CreateTenant(tenant *Tenant) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if tenant.ID == "" {
		return fmt.Errorf("tenant: ID cannot be empty")
	}

	if _, exists := m.tenants[tenant.ID]; exists {
		return fmt.Errorf("tenant: %q already exists", tenant.ID)
	}

	now := time.Now().UnixMilli()
	tenant.CreatedAt = now
	tenant.UpdatedAt = now
	tenant.Status = "active"

	if tenant.Quotas == nil {
		tenant.Quotas = m.defaultQuotas
	}

	m.tenants[tenant.ID] = tenant
	m.usage[tenant.ID] = &Usage{}

	// Create default network policy (no cross-tenant by default)
	policy := &NetworkPolicy{
		TenantID:         tenant.ID,
		Name:             "default",
		AllowCrossTenant: false,
	}
	m.policies[tenant.ID] = policy

	// Bind tenant owner to admin role within tenant
	if tenant.Owner != "" {
		binding := rbac.Binding{
			Identity: tenant.Owner,
			Roles:    []string{rbac.RoleAdmin},
			Tenant:   tenant.ID,
		}
		_ = m.rbacManager.Bind(binding)
	}

	if m.ledger != nil {
		m.ledger.Append(audit.Entry{
			Action:   "create_tenant",
			Actor:    tenant.Owner,
			Source:   audit.SourceHost,
			Resource: fmt.Sprintf("tenant:%s", tenant.ID),
			Detail:   fmt.Sprintf("name=%s owner=%s", tenant.Name, tenant.Owner),
		})
	}

	return nil
}

// GetTenant retrieves a tenant by ID.
func (m *Manager) GetTenant(id string) (*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenant, ok := m.tenants[id]
	if !ok {
		return nil, fmt.Errorf("tenant: %q not found", id)
	}
	return tenant, nil
}

// ListTenants returns all tenants.
func (m *Manager) ListTenants() []*Tenant {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenants := make([]*Tenant, 0, len(m.tenants))
	for _, t := range m.tenants {
		tenants = append(tenants, t)
	}
	return tenants
}

// DeleteTenant removes a tenant (only if empty and not in use).
func (m *Manager) DeleteTenant(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, ok := m.tenants[id]
	if !ok {
		return fmt.Errorf("tenant: %q not found", id)
	}

	usage, ok := m.usage[id]
	if !ok || usage.Applications > 0 || usage.StorageBytes > 0 {
		return fmt.Errorf("tenant: %q not empty (applications=%d, storage=%d bytes)", id, usage.Applications, usage.StorageBytes)
	}

	delete(m.tenants, id)
	delete(m.usage, id)
	delete(m.policies, id)

	if m.ledger != nil {
		m.ledger.Append(audit.Entry{
			Action:   "delete_tenant",
			Actor:    tenant.Owner,
			Source:   audit.SourceHost,
			Resource: fmt.Sprintf("tenant:%s", id),
		})
	}

	return nil
}

// RecordUsage updates resource usage for a tenant.
func (m *Manager) RecordUsage(tenantID string, delta *Usage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, ok := m.tenants[tenantID]
	if !ok {
		return fmt.Errorf("tenant: %q not found", tenantID)
	}

	usage, ok := m.usage[tenantID]
	if !ok {
		usage = &Usage{}
		m.usage[tenantID] = usage
	}

	// Apply deltas
	usage.Applications += delta.Applications
	usage.CPUMilli += delta.CPUMilli
	usage.MemBytes += delta.MemBytes
	usage.StorageBytes += delta.StorageBytes
	usage.NetworkMbps += delta.NetworkMbps

	// Verify quotas
	if tenant.Quotas.MaxApplications > 0 && usage.Applications > tenant.Quotas.MaxApplications {
		return fmt.Errorf("tenant: %q exceeds application quota: %d > %d", tenantID, usage.Applications, tenant.Quotas.MaxApplications)
	}
	if tenant.Quotas.MaxCPUMilli > 0 && usage.CPUMilli > tenant.Quotas.MaxCPUMilli {
		return fmt.Errorf("tenant: %q exceeds CPU quota: %d > %d", tenantID, usage.CPUMilli, tenant.Quotas.MaxCPUMilli)
	}
	if tenant.Quotas.MaxMemBytes > 0 && usage.MemBytes > tenant.Quotas.MaxMemBytes {
		return fmt.Errorf("tenant: %q exceeds memory quota: %d > %d", tenantID, usage.MemBytes, tenant.Quotas.MaxMemBytes)
	}
	if tenant.Quotas.MaxStorageBytes > 0 && usage.StorageBytes > tenant.Quotas.MaxStorageBytes {
		return fmt.Errorf("tenant: %q exceeds storage quota: %d > %d", tenantID, usage.StorageBytes, tenant.Quotas.MaxStorageBytes)
	}

	if m.ledger != nil {
		m.ledger.Append(audit.Entry{
			Action:   "record_usage",
			Actor:    "system",
			Source:   audit.SourceHost,
			Resource: fmt.Sprintf("tenant:%s", tenantID),
			Detail:   fmt.Sprintf("apps=%d cpu=%d mem=%d storage=%d", usage.Applications, usage.CPUMilli, usage.MemBytes, usage.StorageBytes),
		})
	}

	return nil
}

// GetUsage returns resource usage for a tenant.
func (m *Manager) GetUsage(tenantID string) (*Usage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	usage, ok := m.usage[tenantID]
	if !ok {
		return nil, fmt.Errorf("tenant: %q not found", tenantID)
	}
	return usage, nil
}

// CheckQuota verifies if an operation would exceed tenant quotas.
func (m *Manager) CheckQuota(tenantID string, delta *Usage) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenant, ok := m.tenants[tenantID]
	if !ok {
		return fmt.Errorf("tenant: %q not found", tenantID)
	}

	usage, ok := m.usage[tenantID]
	if !ok {
		usage = &Usage{}
	}

	projected := &Usage{
		Applications: usage.Applications + delta.Applications,
		CPUMilli:     usage.CPUMilli + delta.CPUMilli,
		MemBytes:     usage.MemBytes + delta.MemBytes,
		StorageBytes: usage.StorageBytes + delta.StorageBytes,
		NetworkMbps:  usage.NetworkMbps + delta.NetworkMbps,
	}

	if tenant.Quotas.MaxApplications > 0 && projected.Applications > tenant.Quotas.MaxApplications {
		return fmt.Errorf("quota: would exceed application limit: %d > %d", projected.Applications, tenant.Quotas.MaxApplications)
	}
	if tenant.Quotas.MaxCPUMilli > 0 && projected.CPUMilli > tenant.Quotas.MaxCPUMilli {
		return fmt.Errorf("quota: would exceed CPU limit: %d > %d", projected.CPUMilli, tenant.Quotas.MaxCPUMilli)
	}
	if tenant.Quotas.MaxMemBytes > 0 && projected.MemBytes > tenant.Quotas.MaxMemBytes {
		return fmt.Errorf("quota: would exceed memory limit: %d > %d", projected.MemBytes, tenant.Quotas.MaxMemBytes)
	}
	if tenant.Quotas.MaxStorageBytes > 0 && projected.StorageBytes > tenant.Quotas.MaxStorageBytes {
		return fmt.Errorf("quota: would exceed storage limit: %d > %d", projected.StorageBytes, tenant.Quotas.MaxStorageBytes)
	}

	return nil
}

// SetNetworkPolicy sets the network policy for a tenant.
func (m *Manager) SetNetworkPolicy(policy *NetworkPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tenants[policy.TenantID]; !ok {
		return fmt.Errorf("tenant: %q not found", policy.TenantID)
	}

	m.policies[policy.TenantID] = policy

	if m.ledger != nil {
		m.ledger.Append(audit.Entry{
			Action:   "set_network_policy",
			Actor:    "system",
			Source:   audit.SourceHost,
			Resource: fmt.Sprintf("tenant:%s:policy:%s", policy.TenantID, policy.Name),
			Detail:   fmt.Sprintf("crossTenant=%v", policy.AllowCrossTenant),
		})
	}

	return nil
}

// GetNetworkPolicy retrieves the network policy for a tenant.
func (m *Manager) GetNetworkPolicy(tenantID string) (*NetworkPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policy, ok := m.policies[tenantID]
	if !ok {
		return nil, fmt.Errorf("tenant: %q policy not found", tenantID)
	}
	return policy, nil
}

// EnforceTenantIsolation checks if an operation is allowed between tenants.
func (m *Manager) EnforceTenantIsolation(actor, sourceTenant, destTenant string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if sourceTenant == destTenant {
		return nil // Same tenant, always allowed
	}

	policy, ok := m.policies[sourceTenant]
	if !ok {
		return fmt.Errorf("tenant: isolation policy not found for %q", sourceTenant)
	}

	if !policy.AllowCrossTenant {
		if m.ledger != nil {
			m.ledger.Append(audit.Entry{
				Action:   "cross_tenant_denied",
				Actor:    actor,
				Source:   audit.SourceHost,
				Resource: fmt.Sprintf("tenant:%s->%s", sourceTenant, destTenant),
				Detail:   "cross-tenant communication not allowed",
			})
		}
		return fmt.Errorf("tenant: cross-tenant operation denied from %q to %q", sourceTenant, destTenant)
	}

	return nil
}

// SuspendTenant marks a tenant as suspended (no new operations allowed).
func (m *Manager) SuspendTenant(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, ok := m.tenants[id]
	if !ok {
		return fmt.Errorf("tenant: %q not found", id)
	}

	tenant.Status = "suspended"
	tenant.UpdatedAt = time.Now().UnixMilli()

	if m.ledger != nil {
		m.ledger.Append(audit.Entry{
			Action:   "suspend_tenant",
			Actor:    "system",
			Source:   audit.SourceHost,
			Resource: fmt.Sprintf("tenant:%s", id),
		})
	}

	return nil
}

// ResumeTenant marks a tenant as active again.
func (m *Manager) ResumeTenant(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenant, ok := m.tenants[id]
	if !ok {
		return fmt.Errorf("tenant: %q not found", id)
	}

	tenant.Status = "active"
	tenant.UpdatedAt = time.Now().UnixMilli()

	if m.ledger != nil {
		m.ledger.Append(audit.Entry{
			Action:   "resume_tenant",
			Actor:    "system",
			Source:   audit.SourceHost,
			Resource: fmt.Sprintf("tenant:%s", id),
		})
	}

	return nil
}
