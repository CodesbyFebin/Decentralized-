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

// VercelAdapter implements the ProviderAdapter interface for Vercel
type VercelAdapter struct {
	client *http.Client
	providers.BaseAdapter
}

// NewVercelAdapter creates a new Vercel provider adapter
func NewVercelAdapter() *VercelAdapter {
	return &VercelAdapter{
		client: &http.Client{Timeout: 30 * time.Second},
		BaseAdapter: providers.BaseAdapter{
			ProviderName:    "vercel",
			ProviderVersion: "v1",
		},
	}
}

func (a *VercelAdapter) Discover(ctx context.Context, config *providers.ProviderConfig) ([]*providers.Resource, error) {
	url := "https://api.vercel.com/v9/projects?limit=100"

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
		return nil, fmt.Errorf("vercel api error: status %d", resp.StatusCode)
	}

	var result struct {
		Projects []map[string]interface{} `json:"projects"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var resources []*providers.Resource
	for _, project := range result.Projects {
		resource := a.projectToResource(project, config)
		resources = append(resources, resource)
	}

	return resources, nil
}

func (a *VercelAdapter) projectToResource(project map[string]interface{}, config *providers.ProviderConfig) *providers.Resource {
	name := project["name"].(string)
	id := project["id"].(string)

	resource := providers.NewResource("vercel", providers.TypeDeployment, name).
		WithID(fmt.Sprintf("vercel:%s", id)).
		WithProviderID(id).
		WithProjectID(config.ProjectID).
		WithLocation("vercel-edge").
		WithTrustDomain(providers.DomainSHARED).
		WithCapability("deployment", "full", "Deploy to edge network").
		WithCapability("scaling", "full", "Auto-scaling and load balancing").
		WithLabel("provider", "vercel").
		Build()

	return resource
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
	url := fmt.Sprintf("https://api.vercel.com/v9/projects/%s", resource.ProviderResourceID)

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
		return providers.ObserveResult{}, fmt.Errorf("vercel api error: status %d", resp.StatusCode)
	}

	var project map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return providers.ObserveResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	state := "active"
	if archived, ok := project["archived"].(bool); ok && archived {
		state = "archived"
	}

	result := providers.ObserveResult{
		State: state,
		Evidence: &providers.Evidence{
			Timestamp:   time.Now(),
			ContentHash: fmt.Sprintf("%v", project["id"]),
			SignedBy:    "vercel-adapter",
		},
	}

	return result, nil
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
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.vercel.com/v9/user", nil)
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
			Message:   "Failed to connect to Vercel API",
			Error:     err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return providers.HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "Vercel API authentication failed",
			Error:     fmt.Sprintf("status %d: %s", resp.StatusCode, string(body)),
		}
	}

	var user map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return providers.HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "Failed to parse Vercel API response",
			Error:     err.Error(),
		}
	}

	return providers.HealthStatus{
		Healthy:   true,
		LastCheck: time.Now(),
		Message:   fmt.Sprintf("Connected to Vercel"),
	}
}

var _ providers.ProviderAdapter = (*VercelAdapter)(nil)
