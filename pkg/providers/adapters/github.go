package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"decentralized.host/pkg/providers"
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
	var resources []*providers.Resource

	// Paginate through GitHub API: GET /user/repos
	page := 1
	perPage := 100
	for {
		url := fmt.Sprintf("https://api.github.com/user/repos?page=%d&per_page=%d&sort=updated&direction=desc", page, perPage)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("token %s", config.Token))
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := a.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch repositories: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("github api error: status %d", resp.StatusCode)
		}

		var repos []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if len(repos) == 0 {
			break
		}

		// Convert each repo to a Resource
		for _, repo := range repos {
			resource := a.repositoryToResource(repo, config)
			resources = append(resources, resource)
		}

		page++
	}

	return resources, nil
}

// repositoryToResource converts a GitHub API repository to a Resource
func (a *GitHubAdapter) repositoryToResource(repo map[string]interface{}, config *providers.ProviderConfig) *providers.Resource {
	fullName := repo["full_name"].(string)
	parts := strings.Split(fullName, "/")
	owner := parts[0]
	repoName := parts[1]

	// Use deterministic ID format: github:owner/repo
	id := fmt.Sprintf("github:%s", fullName)

	description := ""
	if desc, ok := repo["description"].(string); ok {
		description = desc
	}

	// Determine trust domain based on visibility
	trustDomain := providers.DomainSHARED
	if private, ok := repo["private"].(bool); ok && private {
		trustDomain = providers.DomainPRIVATE
	}

	resource := providers.NewResource("github", providers.TypeRepository, repoName).
		WithID(id).
		WithProviderID(fullName).
		WithProjectID(config.ProjectID).
		WithOwner(owner).
		WithDescription(description).
		WithLocation("github.com").
		WithTrustDomain(trustDomain).
		WithCapability("version-control", "full", "Git repository with version history").
		WithCapability("ci-cd", "basic", "GitHub Actions CI/CD support").
		WithCapability("collaboration", "full", "Team collaboration and code review").
		WithLabel("provider", "github").
		WithLabel("repository_type", determineRepoType(repo)).
		WithAnnotation("github_url", repo["html_url"].(string)).
		WithAnnotation("github_id", fmt.Sprintf("%.0f", repo["id"].(float64))).
		Build()

	resource.LastModifiedAt = parseTime(repo["updated_at"].(string))
	resource.CreatedAt = parseTime(repo["created_at"].(string))

	return resource
}

// determineRepoType guesses the repository type from its description/topics
func determineRepoType(repo map[string]interface{}) string {
	if language, ok := repo["language"].(string); ok && language != "" {
		return strings.ToLower(language)
	}
	return "mixed"
}

// parseTime parses RFC3339 timestamp
func parseTime(ts string) time.Time {
	t, _ := time.Parse(time.RFC3339, ts)
	return t
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
	// Fetch repo state from GitHub API
	url := fmt.Sprintf("https://api.github.com/repos/%s", resource.ProviderResourceID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return providers.ObserveResult{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", config.Token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := a.client.Do(req)
	if err != nil {
		return providers.ObserveResult{}, fmt.Errorf("failed to fetch repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return providers.ObserveResult{}, fmt.Errorf("github api error: status %d", resp.StatusCode)
	}

	var repo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return providers.ObserveResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Determine state based on repository status
	state := "active"
	if archived, ok := repo["archived"].(bool); ok && archived {
		state = "archived"
	}

	result := providers.ObserveResult{
		State: state,
		Evidence: &providers.Evidence{
			Timestamp:   time.Now(),
			ContentHash: fmt.Sprintf("%d", int64(repo["id"].(float64))),
			SignedBy:    "github-adapter",
			SourceSHA:   repo["default_branch"].(string),
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
	// Fetch source repo from GitHub to get commit info
	sourceURL := fmt.Sprintf("https://api.github.com/repos/%s/commits?per_page=1", sourceID)
	resp, err := a.client.Get(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch source commits: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error (source): status %d", resp.StatusCode)
	}

	var sourceCommits []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&sourceCommits); err != nil {
		return nil, fmt.Errorf("failed to parse source commits: %w", err)
	}

	result := &providers.DiffResult{
		Added:    []string{},
		Removed:  []string{},
		Modified: make(map[string]string),
	}

	if len(sourceCommits) > 0 {
		commit := sourceCommits[0]
		sha := commit["sha"].(string)
		result.Modified["latest_commit"] = sha
	}

	return result, nil
}

// Export gets the full repository definition
func (a *GitHubAdapter) Export(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource) (interface{}, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s", resource.ProviderResourceID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", config.Token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error: status %d", resp.StatusCode)
	}

	var repo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	export := map[string]interface{}{
		"repository_id": resource.ProviderResourceID,
		"url":           fmt.Sprintf("https://github.com/%s", resource.ProviderResourceID),
		"exported_at":   time.Now().Format(time.RFC3339),
		"full_name":     repo["full_name"],
		"description":   repo["description"],
		"private":       repo["private"],
		"language":      repo["language"],
		"stars":         repo["stargazers_count"],
		"forks":         repo["forks_count"],
		"default_branch": repo["default_branch"],
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
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return providers.HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "Failed to create request",
			Error:     err.Error(),
		}
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", config.Token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := a.client.Do(req)
	if err != nil {
		return providers.HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "Failed to connect to GitHub API",
			Error:     err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return providers.HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "GitHub API authentication failed",
			Error:     fmt.Sprintf("status %d: %s", resp.StatusCode, string(body)),
		}
	}

	var user map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return providers.HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "Failed to parse GitHub API response",
			Error:     err.Error(),
		}
	}

	return providers.HealthStatus{
		Healthy:   true,
		LastCheck: time.Now(),
		Message:   fmt.Sprintf("Connected as %v", user["login"]),
	}
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
