package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"decentralized/pkg/providers"
)

// GitHubAdapter implements the ProviderAdapter interface for GitHub
type GitHubAdapter struct {
	client *http.Client
	providers.BaseAdapter
}

// NewGitHubAdapter creates a new GitHub provider adapter
func NewGitHubAdapter() *GitHubAdapter {
	return &GitHubAdapter{
		client: &http.Client{Timeout: 30 * time.Second},
		BaseAdapter: providers.BaseAdapter{
			ProviderName:    "github",
			ProviderVersion: "v1",
		},
	}
}

// Discover finds all repositories the authenticated user has access to
func (a *GitHubAdapter) Discover(ctx context.Context, config *providers.ProviderConfig) ([]*providers.Resource, error) {
	// v0.1: Simple stub that returns empty list
	// v0.2: Will implement full GitHub API discovery (repos, workflows, packages, etc.)
	resources := []*providers.Resource{}

	// Placeholder for GitHub API call
	// In production:
	// 1. Paginate through GitHub API: GET /user/repos
	// 2. For each repo, fetch additional metadata
	// 3. Convert to Resource objects with proper ID, dependencies, etc.

	return resources, nil
}

// Import brings a single GitHub repository into the graph
func (a *GitHubAdapter) Import(ctx context.Context, config *providers.ProviderConfig, resourceID string) (*providers.Resource, error) {
	// resourceID format: "owner/repo"
	// v0.1: Returns a mock resource
	// v0.2: Will fetch actual data from GitHub API

	resource := providers.NewResource("github", providers.TypeRepository, resourceID).
		WithProviderID(resourceID).
		WithProjectID(config.ProjectID).
		WithLocation("github.com").
		WithTrustDomain(providers.DomainSHARED).
		Build()

	return resource, nil
}

// Observe gets the current state of a GitHub repository
func (a *GitHubAdapter) Observe(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource) (providers.ObserveResult, error) {
	// v0.1: Returns mock state
	// v0.2: Will fetch actual repo state from GitHub API

	result := providers.ObserveResult{
		State: "active",
		Evidence: &providers.Evidence{
			Timestamp:  time.Now(),
			ContentHash: "mock-hash",
			SignedBy:   "github-adapter",
		},
	}

	return result, nil
}

// Plan calculates migration steps to move a repo to owned infrastructure
func (a *GitHubAdapter) Plan(ctx context.Context, resource *providers.Resource, targetLocation string) (*providers.MigrationPlan, error) {
	// v0.1: Returns basic migration plan
	// v0.2: Will include detailed cost/time estimates

	plan := &providers.MigrationPlan{
		ResourceID:     resource.ID,
		SourceProvider: resource.Provider,
		TargetLocation: targetLocation,
		EstimatedCost:  "$0 (owned infrastructure)",
		EstimatedTime:  1 * time.Hour,
		DowntimeWindow: 5 * time.Minute,
		Steps: []providers.MigrationStep{
			{
				Sequence:    1,
				Name:        "Export repository",
				Description: "Clone repository with full history",
				Action:      "clone",
				Timeout:     10 * time.Minute,
			},
			{
				Sequence:    2,
				Name:        "Create mirror",
				Description: "Push to owned Git server",
				Action:      "push",
				Timeout:     5 * time.Minute,
			},
			{
				Sequence:    3,
				Name:        "Verify integrity",
				Description: "Compare commit hashes before/after",
				Action:      "verify",
				Timeout:     2 * time.Minute,
			},
		},
	}

	return plan, nil
}

// Apply executes a migration plan (not in v0.1)
func (a *GitHubAdapter) Apply(ctx context.Context, plan *providers.MigrationPlan) error {
	return fmt.Errorf("Apply not implemented in v0.1 (coming in v0.2)")
}

// Watch monitors a GitHub repository for changes
func (a *GitHubAdapter) Watch(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource, callback func(*providers.Resource)) error {
	// v0.1: Stub for future webhook/polling implementation
	// v0.2: Will implement polling or webhook observation
	return fmt.Errorf("Watch not implemented in v0.1")
}

// Diff compares a GitHub repo against a migrated version
func (a *GitHubAdapter) Diff(ctx context.Context, sourceID string, targetID string) (*providers.DiffResult, error) {
	// v0.1: Returns empty diff
	// v0.2: Will compare commit hashes, file lists, etc.

	return &providers.DiffResult{
		Added:    []string{},
		Removed:  []string{},
		Modified: make(map[string]string),
	}, nil
}

// Export gets the full repository definition
func (a *GitHubAdapter) Export(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource) (interface{}, error) {
	// v0.1: Returns basic metadata
	// v0.2: Will export full repo backup

	export := map[string]interface{}{
		"repository_id": resource.ProviderResourceID,
		"url":           fmt.Sprintf("https://github.com/%s", resource.ProviderResourceID),
		"exported_at":   time.Now().ISO8601(),
	}

	return export, nil
}

// Capabilities returns what GitHub adapter supports
func (a *GitHubAdapter) Capabilities(ctx context.Context, resourceType providers.ResourceType) []providers.Capability {
	return []providers.Capability{
		{
			Name:        "repository_discovery",
			Level:       "full",
			Description: "Can discover all accessible repositories",
		},
		{
			Name:        "repo_export",
			Level:       "full",
			Description: "Can export repository with full history",
		},
		{
			Name:        "migration_planning",
			Level:       "full",
			Description: "Can plan repository migration",
		},
		{
			Name:        "webhook_observation",
			Level:       "basic",
			Description: "Can observe changes via GitHub webhooks (future)",
		},
	}
}

// Health checks GitHub API connectivity
func (a *GitHubAdapter) Health(ctx context.Context, config *providers.ProviderConfig) providers.HealthStatus {
	// v0.1: Returns placeholder
	// v0.2: Will actually call GitHub API to verify token

	status := providers.HealthStatus{
		Healthy:   true, // Placeholder
		LastCheck: time.Now(),
		Message:   "GitHub adapter initialized",
	}

	return status
}

// GitHubDiscoveryResponse matches GitHub API response structure
type GitHubDiscoveryResponse struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	HTMLURL     string    `json:"html_url"`
	Private     bool      `json:"private"`
	UpdatedAt   time.Time `json:"updated_at"`
	Language    string    `json:"language"`
}

// makeGitHubRequest is a helper for GitHub API calls
func (a *GitHubAdapter) makeGitHubRequest(ctx context.Context, token string, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com"+endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

var _ providers.ProviderAdapter = (*GitHubAdapter)(nil)
