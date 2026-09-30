package adapters

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"decentralized.host/pkg/providers"
)

// KubernetesAdapter implements the ProviderAdapter interface for Kubernetes
type KubernetesAdapter struct {
	client *http.Client
	providers.BaseAdapter
}

// NewKubernetesAdapter creates a new Kubernetes provider adapter
func NewKubernetesAdapter() *KubernetesAdapter {
	return &KubernetesAdapter{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // For self-signed certs
			},
		},
		BaseAdapter: providers.BaseAdapter{
			ProviderName:    "kubernetes",
			ProviderVersion: "v1",
		},
	}
}

func (a *KubernetesAdapter) Discover(ctx context.Context, config *providers.ProviderConfig) ([]*providers.Resource, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		return nil, fmt.Errorf("kubernetes endpoint not configured")
	}

	url := fmt.Sprintf("%s/api/v1/namespaces", endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.Token))

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch namespaces: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kubernetes api error: status %d", resp.StatusCode)
	}

	var result struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var resources []*providers.Resource
	for _, ns := range result.Items {
		resource := a.namespaceToResource(ns, config)
		resources = append(resources, resource)
	}

	return resources, nil
}

func (a *KubernetesAdapter) namespaceToResource(ns map[string]interface{}, config *providers.ProviderConfig) *providers.Resource {
	metadata := ns["metadata"].(map[string]interface{})
	name := metadata["name"].(string)

	resource := providers.NewResource("kubernetes", providers.TypeCluster, name).
		WithID(fmt.Sprintf("kubernetes:%s", name)).
		WithProviderID(name).
		WithProjectID(config.ProjectID).
		WithLocation("k8s-cluster").
		WithTrustDomain(providers.DomainORG_ONLY).
		WithCapability("orchestration", "full", "Kubernetes workload orchestration").
		WithCapability("scaling", "full", "Auto-scaling and load balancing").
		WithLabel("provider", "kubernetes").
		Build()

	return resource
}

func (a *KubernetesAdapter) Import(ctx context.Context, config *providers.ProviderConfig, resourceID string) (*providers.Resource, error) {
	resource := providers.NewResource("kubernetes", providers.TypeCluster, resourceID).
		WithProviderID(resourceID).
		WithProjectID(config.ProjectID).
		WithLocation("k8s-cluster").
		WithTrustDomain(providers.DomainORG_ONLY).
		Build()
	return resource, nil
}

func (a *KubernetesAdapter) Observe(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource) (providers.ObserveResult, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		return providers.ObserveResult{}, fmt.Errorf("kubernetes endpoint not configured")
	}

	url := fmt.Sprintf("%s/api/v1/namespaces/%s", endpoint, resource.ProviderResourceID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return providers.ObserveResult{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.Token))

	resp, err := a.client.Do(req)
	if err != nil {
		return providers.ObserveResult{}, fmt.Errorf("failed to fetch namespace: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return providers.ObserveResult{}, fmt.Errorf("kubernetes api error: status %d", resp.StatusCode)
	}

	var ns map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ns); err != nil {
		return providers.ObserveResult{}, fmt.Errorf("failed to parse response: %w", err)
	}

	status := "active"
	if phase, ok := ns["status"].(map[string]interface{}); ok {
		if p, ok := phase["phase"].(string); ok && p == "Terminating" {
			status = "terminating"
		}
	}

	result := providers.ObserveResult{
		State: status,
		Evidence: &providers.Evidence{
			Timestamp:   time.Now(),
			ContentHash: fmt.Sprintf("%v", ns["metadata"].(map[string]interface{})["uid"]),
			SignedBy:    "kubernetes-adapter",
		},
	}

	return result, nil
}

func (a *KubernetesAdapter) Plan(ctx context.Context, resource *providers.Resource, targetLocation string) (*providers.MigrationPlan, error) {
	return &providers.MigrationPlan{
		ResourceID:     resource.ID,
		SourceProvider: resource.Provider,
		TargetLocation: targetLocation,
		EstimatedCost:  "$0 (cluster reuse)",
		EstimatedTime:  2 * time.Hour,
		DowntimeWindow: 0 * time.Minute, // Can be zero-downtime via gradual rollover
		Steps: []providers.MigrationStep{
			{Sequence: 1, Name: "Drain old workloads", Action: "drain", Timeout: 10 * time.Minute},
			{Sequence: 2, Name: "Scale up new cluster", Action: "scale", Timeout: 10 * time.Minute},
			{Sequence: 3, Name: "Route traffic", Action: "route", Timeout: 5 * time.Minute},
			{Sequence: 4, Name: "Verify health", Action: "verify", Timeout: 5 * time.Minute},
		},
	}, nil
}

func (a *KubernetesAdapter) Apply(ctx context.Context, plan *providers.MigrationPlan) error {
	return fmt.Errorf("Apply not implemented in v0.1")
}

func (a *KubernetesAdapter) Watch(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource, callback func(*providers.Resource)) error {
	return fmt.Errorf("Watch not implemented in v0.1")
}

func (a *KubernetesAdapter) Diff(ctx context.Context, sourceID string, targetID string) (*providers.DiffResult, error) {
	return &providers.DiffResult{}, nil
}

func (a *KubernetesAdapter) Export(ctx context.Context, config *providers.ProviderConfig, resource *providers.Resource) (interface{}, error) {
	return map[string]interface{}{"cluster_id": resource.ProviderResourceID}, nil
}

func (a *KubernetesAdapter) Health(ctx context.Context, config *providers.ProviderConfig) providers.HealthStatus {
	endpoint := config.Endpoint
	if endpoint == "" {
		return providers.HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "Kubernetes endpoint not configured",
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1", endpoint), nil)
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
			Message:   "Failed to connect to Kubernetes API",
			Error:     err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return providers.HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "Kubernetes API authentication failed",
			Error:     fmt.Sprintf("status %d: %s", resp.StatusCode, string(body)),
		}
	}

	return providers.HealthStatus{
		Healthy:   true,
		LastCheck: time.Now(),
		Message:   "Connected to Kubernetes cluster",
	}
}

var _ providers.ProviderAdapter = (*KubernetesAdapter)(nil)
