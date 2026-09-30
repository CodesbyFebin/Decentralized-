package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"decentralized.host/pkg/providers"
)

// DockerAdapter implements the ProviderAdapter interface for Docker
type DockerAdapter struct {
	providers.BaseAdapter
}

// newDockerClient creates an HTTP client for Docker socket communication
func newDockerClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
	}
}

// NewDockerAdapter creates a new Docker provider adapter
func NewDockerAdapter() *DockerAdapter {
	return &DockerAdapter{
		BaseAdapter: providers.BaseAdapter{
			ProviderName:    "docker",
			ProviderVersion: "v1",
		},
	}
}

func (a *DockerAdapter) Discover(ctx context.Context, config *providers.ProviderConfig) ([]*providers.Resource, error) {
	client := newDockerClient()
	defer client.CloseIdleConnections()

	// List containers
	req, err := http.NewRequestWithContext(ctx, "GET", "http://docker/containers/json?all=true", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker api error: status %d", resp.StatusCode)
	}

	var containers []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var resources []*providers.Resource
	for _, container := range containers {
		resource := a.containerToResource(container, config)
		resources = append(resources, resource)
	}

	return resources, nil
}

func (a *DockerAdapter) containerToResource(container map[string]interface{}, config *providers.ProviderConfig) *providers.Resource {
	id := container["Id"].(string)
	if len(id) > 12 {
		id = id[:12]
	}

	names := container["Names"].([]interface{})
	name := "unknown"
	if len(names) > 0 {
		if n, ok := names[0].(string); ok {
			name = n
			if name[0] == '/' {
				name = name[1:]
			}
		}
	}

	resource := providers.NewResource("docker", providers.TypeContainer, name).
		WithID(fmt.Sprintf("docker:%s", id)).
		WithProviderID(id).
		WithProjectID(config.ProjectID).
		WithLocation("docker-local").
		WithTrustDomain(providers.DomainLOCAL).
		WithCapability("containerization", "full", "Docker containerized workloads").
		WithLabel("provider", "docker").
		Build()

	return resource
}

func (a *DockerAdapter) Import(ctx context.Context, config *providers.ProviderConfig, resourceID string) (*providers.Resource, error) {
	resource := providers.NewResource("docker", providers.TypeContainer, resourceID).
		WithProviderID(resourceID).
		WithProjectID(config.ProjectID).
		WithLocation("docker-local").
		WithTrustDomain(providers.DomainLOCAL).
		Build()
	return resource, nil
}

func (a *DockerAdapter) Observe(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource) (providers.ObserveResult, error) {
	client := newDockerClient()
	defer client.CloseIdleConnections()

	url := fmt.Sprintf("http://docker/containers/%s/json", resource.ProviderResourceID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return providers.ObserveResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return providers.ObserveResult{}, fmt.Errorf("failed to fetch container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return providers.ObserveResult{}, fmt.Errorf("docker api error: status %d", resp.StatusCode)
	}

	var container map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&container); err != nil {
		return providers.ObserveResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	state := "unknown"
	if s, ok := container["State"].(map[string]interface{}); ok {
		if running, ok := s["Running"].(bool); ok {
			if running {
				state = "running"
			} else {
				state = "stopped"
			}
		}
	}

	result := providers.ObserveResult{
		State: state,
		Evidence: &providers.Evidence{
			Timestamp:   time.Now(),
			ContentHash: fmt.Sprintf("%v", container["Id"]),
			SignedBy:    "docker-adapter",
		},
	}

	return result, nil
}

func (a *DockerAdapter) Plan(ctx context.Context, resource *providers.Resource, targetLocation string) (*providers.MigrationPlan, error) {
	return &providers.MigrationPlan{
		ResourceID:     resource.ID,
		SourceProvider: resource.Provider,
		TargetLocation: targetLocation,
		EstimatedCost:  "$0 (already sovereign)",
		EstimatedTime:  1 * time.Hour,
		DowntimeWindow: 2 * time.Minute,
		Steps: []providers.MigrationStep{
			{Sequence: 1, Name: "Export image", Action: "docker-save", Timeout: 5 * time.Minute},
			{Sequence: 2, Name: "Transfer image", Action: "transfer", Timeout: 10 * time.Minute},
			{Sequence: 3, Name: "Start on owned infrastructure", Action: "docker-run", Timeout: 5 * time.Minute},
		},
	}, nil
}

func (a *DockerAdapter) Apply(ctx context.Context, plan *providers.MigrationPlan) error {
	return fmt.Errorf("Apply not implemented in v0.1")
}

func (a *DockerAdapter) Watch(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource, callback func(*providers.Resource)) error {
	return fmt.Errorf("Watch not implemented in v0.1")
}

func (a *DockerAdapter) Diff(ctx context.Context, sourceID string, targetID string) (*providers.DiffResult, error) {
	return &providers.DiffResult{}, nil
}

func (a *DockerAdapter) Export(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource) (interface{}, error) {
	return map[string]interface{}{"image_id": resource.ProviderResourceID}, nil
}

func (a *DockerAdapter) Health(ctx context.Context, config *providers.ProviderConfig) providers.HealthStatus {
	client := newDockerClient()
	defer client.CloseIdleConnections()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://docker/_ping", nil)
	if err != nil {
		return providers.HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "Failed to create request",
			Error:     err.Error(),
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return providers.HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "Failed to connect to Docker daemon",
			Error:     err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return providers.HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "Docker daemon not responding",
			Error:     fmt.Sprintf("status %d: %s", resp.StatusCode, string(body)),
		}
	}

	return providers.HealthStatus{
		Healthy:   true,
		LastCheck: time.Now(),
		Message:   "Docker daemon accessible",
	}
}

var _ providers.ProviderAdapter = (*DockerAdapter)(nil)
