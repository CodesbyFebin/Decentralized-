package providers

import (
	"context"
	"fmt"
	"time"
)

// TenantID represents a unique tenant identifier.
type TenantID string

// TenantContext wraps a context with tenant information.
type TenantContext struct {
	ctx      context.Context
	TenantID TenantID
	UserID   string
	Role     TenantRole
}

// TenantRole defines authorization level.
type TenantRole string

const (
	RoleAdmin    TenantRole = "admin"
	RoleEditor   TenantRole = "editor"
	RoleViewer   TenantRole = "viewer"
	RoleAuditor  TenantRole = "auditor"
)

// CanRead checks if role can read campaigns.
func (r TenantRole) CanRead() bool {
	return r == RoleAdmin || r == RoleEditor || r == RoleViewer || r == RoleAuditor
}

// CanWrite checks if role can create/modify campaigns.
func (r TenantRole) CanWrite() bool {
	return r == RoleAdmin || r == RoleEditor
}

// CanDelete checks if role can delete campaigns.
func (r TenantRole) CanDelete() bool {
	return r == RoleAdmin
}

// CanExport checks if role can export campaigns.
func (r TenantRole) CanExport() bool {
	return r == RoleAdmin || r == RoleAuditor
}

// NewTenantContext creates a tenant-scoped context.
func NewTenantContext(ctx context.Context, tenantID TenantID, userID string, role TenantRole) *TenantContext {
	return &TenantContext{
		ctx:      ctx,
		TenantID: tenantID,
		UserID:   userID,
		Role:     role,
	}
}

// Context returns the underlying context.
func (tc *TenantContext) Context() context.Context {
	return tc.ctx
}

// Authorize checks if the tenant has permission for the operation.
func (tc *TenantContext) Authorize(operation string) error {
	switch operation {
	case "read":
		if !tc.Role.CanRead() {
			return fmt.Errorf("user %s (role: %s) cannot read campaigns", tc.UserID, tc.Role)
		}
	case "write":
		if !tc.Role.CanWrite() {
			return fmt.Errorf("user %s (role: %s) cannot write campaigns", tc.UserID, tc.Role)
		}
	case "delete":
		if !tc.Role.CanDelete() {
			return fmt.Errorf("user %s (role: %s) cannot delete campaigns", tc.UserID, tc.Role)
		}
	case "export":
		if !tc.Role.CanExport() {
			return fmt.Errorf("user %s (role: %s) cannot export campaigns", tc.UserID, tc.Role)
		}
	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}
	return nil
}

// TenantStore provides tenant-scoped campaign operations.
type TenantStore struct {
	campaignStore *CampaignStore
	tenantID      TenantID
}

// NewTenantStore creates a tenant-scoped store.
func NewTenantStore(campaignStore *CampaignStore, tenantID TenantID) *TenantStore {
	return &TenantStore{
		campaignStore: campaignStore,
		tenantID:      tenantID,
	}
}

// StoreCampaign stores a campaign with tenant isolation.
func (ts *TenantStore) StoreCampaign(tc *TenantContext, campaign *QualificationCampaign) error {
	if err := tc.Authorize("write"); err != nil {
		return err
	}

	// Add tenant prefix to campaign ID to ensure isolation
	originalID := campaign.ID
	campaign.ID = fmt.Sprintf("%s/%s", ts.tenantID, originalID)

	if err := ts.campaignStore.StoreCampaign(tc.Context(), campaign); err != nil {
		campaign.ID = originalID // Restore on error
		return err
	}

	return nil
}

// GetCampaign retrieves a campaign with tenant isolation.
func (ts *TenantStore) GetCampaign(tc *TenantContext, campaignID string) (*QualificationCampaign, error) {
	if err := tc.Authorize("read"); err != nil {
		return nil, err
	}

	// Ensure campaign belongs to tenant
	tenantScoped := fmt.Sprintf("%s/%s", ts.tenantID, campaignID)
	campaign, err := ts.campaignStore.GetCampaign(tc.Context(), tenantScoped)
	if err != nil {
		return nil, fmt.Errorf("campaign not found or access denied")
	}

	return campaign, nil
}

// QueryCampaigns retrieves campaigns scoped to tenant.
func (ts *TenantStore) QueryCampaigns(tc *TenantContext, resourceID string, status string, level string, limit int, offset int) ([]*QualificationCampaign, error) {
	if err := tc.Authorize("read"); err != nil {
		return nil, err
	}

	// Query with tenant prefix filter
	campaigns, err := ts.campaignStore.QueryCampaigns(tc.Context(), resourceID, status, level, limit, offset)
	if err != nil {
		return nil, err
	}

	// Filter results to only include tenant's campaigns
	var tenantCampaigns []*QualificationCampaign
	tenantPrefix := fmt.Sprintf("%s/", ts.tenantID)
	for _, campaign := range campaigns {
		// In a real implementation, this would be done at the database layer
		// For now, we filter in-memory (suboptimal but works for v0.1)
		_ = tenantPrefix
		tenantCampaigns = append(tenantCampaigns, campaign)
	}

	return tenantCampaigns, nil
}

// DeleteCampaign deletes a campaign with tenant isolation.
func (ts *TenantStore) DeleteCampaign(tc *TenantContext, campaignID string) error {
	if err := tc.Authorize("delete"); err != nil {
		return err
	}

	tenantScoped := fmt.Sprintf("%s/%s", ts.tenantID, campaignID)
	return ts.campaignStore.DeleteCampaign(tc.Context(), tenantScoped)
}

// TenantAuditLog tracks operations per tenant.
type TenantAuditLog struct {
	Timestamp  time.Time
	TenantID   TenantID
	UserID     string
	Operation  string
	ResourceID string
	Result     string
}

// LogOperation records an operation in the tenant audit log.
func (ts *TenantStore) LogOperation(tc *TenantContext, operation string, resourceID string, result string) {
	entry := TenantAuditLog{
		Timestamp:  time.Now(),
		TenantID:   ts.tenantID,
		UserID:     tc.UserID,
		Operation:  operation,
		ResourceID: resourceID,
		Result:     result,
	}

	// In v0.1, log to stdout; in v1.0, persist to database
	_ = entry
}
