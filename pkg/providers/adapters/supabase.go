package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"decentralized.host/pkg/providers"
)

// SupabaseAdapter implements the ProviderAdapter interface for Supabase
type SupabaseAdapter struct {
	client *http.Client
	providers.BaseAdapter
}

// NewSupabaseAdapter creates a new Supabase provider adapter
func NewSupabaseAdapter() *SupabaseAdapter {
	return &SupabaseAdapter{
		client: &http.Client{Timeout: 30 * time.Second},
		BaseAdapter: providers.BaseAdapter{
			ProviderName:    "supabase",
			ProviderVersion: "v1",
		},
	}
}

func (a *SupabaseAdapter) Discover(ctx context.Context, config *providers.ProviderConfig) ([]*providers.Resource, error) {
	url := "https://api.supabase.com/api/v1/projects"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.Token))

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("supabase api error: status %d", resp.StatusCode)
	}

	var projects []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var resources []*providers.Resource
	for _, project := range projects {
		resource := a.projectToResource(project, config)
		resources = append(resources, resource)
	}

	return resources, nil
}

func (a *SupabaseAdapter) projectToResource(project map[string]interface{}, config *providers.ProviderConfig) *providers.Resource {
	name := project["name"].(string)
	id := project["id"].(string)

	resource := providers.NewResource("supabase", providers.TypeDatabase, name).
		WithID(fmt.Sprintf("supabase:%s", id)).
		WithProviderID(id).
		WithProjectID(config.ProjectID).
		WithLocation("supabase-cloud").
		WithTrustDomain(providers.DomainPRIVATE).
		WithCapability("database", "full", "PostgreSQL database hosting").
		WithCapability("realtime", "full", "Real-time subscriptions").
		WithCapability("auth", "full", "Authentication and authorization").
		WithLabel("provider", "supabase").
		Build()

	return resource
}

func (a *SupabaseAdapter) Import(ctx context.Context, config *providers.ProviderConfig, resourceID string) (*providers.Resource, error) {
	resource := providers.NewResource("supabase", providers.TypeDatabase, resourceID).
		WithProviderID(resourceID).
		WithProjectID(config.ProjectID).
		WithLocation("supabase-cloud").
		WithTrustDomain(providers.DomainSHARED).
		Build()
	return resource, nil
}

func (a *SupabaseAdapter) Observe(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource) (providers.ObserveResult, error) {
	url := fmt.Sprintf("https://api.supabase.com/api/v1/projects/%s", resource.ProviderResourceID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return providers.ObserveResult{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.Token))

	resp, err := a.client.Do(req)
	if err != nil {
		return providers.ObserveResult{}, fmt.Errorf("failed to fetch project: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return providers.ObserveResult{}, fmt.Errorf("supabase api error: status %d", resp.StatusCode)
	}

	var project map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return providers.ObserveResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	state := "active"
	if status, ok := project["status"].(string); ok && status == "inactive" {
		state = "paused"
	}

	result := providers.ObserveResult{
		State: state,
		Evidence: &providers.Evidence{
			Timestamp:   time.Now(),
			ContentHash: fmt.Sprintf("%v", project["id"]),
			SignedBy:    "supabase-adapter",
		},
	}

	return result, nil
}

func (a *SupabaseAdapter) Plan(ctx context.Context, resource *providers.Resource, targetLocation string) (*providers.MigrationPlan, error) {
	return &providers.MigrationPlan{
		ResourceID:     resource.ID,
		SourceProvider: resource.Provider,
		TargetLocation: targetLocation,
		EstimatedCost:  "$0 (self-hosted PostgreSQL)",
		EstimatedTime:  4 * time.Hour,
		DowntimeWindow: 30 * time.Minute,
		Steps: []providers.MigrationStep{
			{Sequence: 1, Name: "Export database schema", Action: "pg_dump", Timeout: 5 * time.Minute},
			{Sequence: 2, Name: "Backup data", Action: "backup", Timeout: 10 * time.Minute},
			{Sequence: 3, Name: "Initialize owned PostgreSQL", Action: "init", Timeout: 10 * time.Minute},
			{Sequence: 4, Name: "Restore data", Action: "restore", Timeout: 10 * time.Minute},
			{Sequence: 5, Name: "Verify integrity", Action: "verify", Timeout: 5 * time.Minute},
		},
	}, nil
}

func (a *SupabaseAdapter) Apply(ctx context.Context, plan *providers.MigrationPlan) error {
	return fmt.Errorf("Apply not implemented in v0.1")
}

func (a *SupabaseAdapter) Watch(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource, callback func(*providers.Resource)) error {
	return fmt.Errorf("Watch not implemented in v0.1")
}

func (a *SupabaseAdapter) Diff(ctx context.Context, sourceID string, targetID string) (*providers.DiffResult, error) {
	return &providers.DiffResult{}, nil
}

func (a *SupabaseAdapter) Export(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource) (interface{}, error) {
	return map[string]interface{}{"database_id": resource.ProviderResourceID}, nil
}

func (a *SupabaseAdapter) Health(ctx context.Context, config *providers.ProviderConfig) providers.HealthStatus {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.supabase.com/api/v1/projects", nil)
	if err != nil {
		return providers.HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "Failed to create request",
			Error:     err.Error(),
		}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.Token))

	resp, err := a.client.Do(req)
	if err != nil {
		return providers.HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "Failed to connect to Supabase API",
			Error:     err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return providers.HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "Supabase API authentication failed",
			Error:     fmt.Sprintf("status %d: %s", resp.StatusCode, string(body)),
		}
	}

	return providers.HealthStatus{
		Healthy:   true,
		LastCheck: time.Now(),
		Message:   "Connected to Supabase",
	}
}

var _ providers.ProviderAdapter = (*SupabaseAdapter)(nil)
