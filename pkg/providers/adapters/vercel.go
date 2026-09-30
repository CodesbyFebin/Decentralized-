package adapters

import (
	"context"
	"fmt"
	"time"

	"decentralized/pkg/providers"
)

// VercelAdapter implements the ProviderAdapter interface for Vercel
type VercelAdapter struct {
	providers.BaseAdapter
}

// NewVercelAdapter creates a new Vercel provider adapter
func NewVercelAdapter() *VercelAdapter {
	return &VercelAdapter{
		BaseAdapter: providers.BaseAdapter{
			ProviderName:    "vercel",
			ProviderVersion: "v1",
		},
	}
}

func (a *VercelAdapter) Discover(ctx context.Context, config *providers.ProviderConfig) ([]*providers.Resource, error) {
	// v0.1: Stub
	return []*providers.Resource{}, nil
}

func (a *VercelAdapter) Import(ctx context.Context, config *providers.ProviderConfig, resourceID string) (*providers.Resource, error) {
	resource := providers.NewResource("vercel", providers.TypeDeployment, resourceID).
		WithProviderID(resourceID).
		WithProjectID(config.ProjectID).
		WithLocation("vercel-edge").
		WithTrustDomain(providers.DomainSHARED).
		Build()
	return resource, nil
}

func (a *VercelAdapter) Observe(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource) (providers.ObserveResult, error) {
	return providers.ObserveResult{
		State: "active",
		Evidence: &providers.Evidence{
			Timestamp: time.Now(),
		},
	}, nil
}

func (a *VercelAdapter) Plan(ctx context.Context, resource *providers.Resource, targetLocation string) (*providers.MigrationPlan, error) {
	return &providers.MigrationPlan{
		ResourceID:     resource.ID,
		SourceProvider: resource.Provider,
		TargetLocation: targetLocation,
		EstimatedCost:  "$0 (owned infrastructure)",
		EstimatedTime:  2 * time.Hour,
		DowntimeWindow: 10 * time.Minute,
		Steps: []providers.MigrationStep{
			{Sequence: 1, Name: "Export deployment config", Action: "export", Timeout: 5 * time.Minute},
			{Sequence: 2, Name: "Clone repository", Action: "clone", Timeout: 5 * time.Minute},
			{Sequence: 3, Name: "Deploy to owned infrastructure", Action: "deploy", Timeout: 10 * time.Minute},
		},
	}, nil
}

func (a *VercelAdapter) Apply(ctx context.Context, plan *providers.MigrationPlan) error {
	return fmt.Errorf("Apply not implemented in v0.1")
}

func (a *VercelAdapter) Watch(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource, callback func(*providers.Resource)) error {
	return fmt.Errorf("Watch not implemented in v0.1")
}

func (a *VercelAdapter) Diff(ctx context.Context, sourceID string, targetID string) (*providers.DiffResult, error) {
	return &providers.DiffResult{}, nil
}

func (a *VercelAdapter) Export(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource) (interface{}, error) {
	return map[string]interface{}{"deployment_id": resource.ProviderResourceID}, nil
}

func (a *VercelAdapter) Health(ctx context.Context, config *providers.ProviderConfig) providers.HealthStatus {
	return providers.HealthStatus{Healthy: true, LastCheck: time.Now()}
}

var _ providers.ProviderAdapter = (*VercelAdapter)(nil)
