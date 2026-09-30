package providers

import (
	"context"
	"time"
)

// ProviderAdapter is the standard interface all provider integrations must implement
type ProviderAdapter interface {
	// Discover finds all resources this provider knows about
	Discover(ctx context.Context, config *ProviderConfig) ([]*Resource, error)

	// Import brings a resource into the unified graph
	Import(ctx context.Context, config *ProviderConfig, resourceID string) (*Resource, error)

	// Observe gets the current state of a resource from the provider
	Observe(ctx context.Context, config *ProviderConfig, resource *Resource) (ObserveResult, error)

	// Plan calculates migration steps to move a resource to owned infrastructure
	Plan(ctx context.Context, resource *Resource, targetLocation string) (*MigrationPlan, error)

	// Apply executes a migration plan (not in v0.1, stub for v0.2+)
	Apply(ctx context.Context, plan *MigrationPlan) error

	// Watch monitors a resource for changes (continuous observation)
	Watch(ctx context.Context, config *ProviderConfig, resource *Resource, callback func(*Resource)) error

	// Diff compares resource state between source and target
	Diff(ctx context.Context, sourceID string, targetID string) (*DiffResult, error)

	// Export gets the full definition of a resource (for portability)
	Export(ctx context.Context, config *ProviderConfig, resource *Resource) (interface{}, error)

	// Capabilities returns what this provider supports
	Capabilities(ctx context.Context, resourceType ResourceType) []Capability

	// Health checks if the provider is reachable and authenticated
	Health(ctx context.Context, config *ProviderConfig) HealthStatus

	// Name returns the provider identifier (e.g., "github", "vercel", "supabase")
	Name() string

	// Version returns the adapter version
	Version() string
}

// AdapterRegistry holds all registered provider adapters
type AdapterRegistry struct {
	adapters map[string]ProviderAdapter
}

// NewAdapterRegistry creates an empty registry
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]ProviderAdapter),
	}
}

// Register adds an adapter to the registry
func (r *AdapterRegistry) Register(name string, adapter ProviderAdapter) error {
	r.adapters[name] = adapter
	return nil
}

// Get retrieves an adapter by name
func (r *AdapterRegistry) Get(name string) ProviderAdapter {
	return r.adapters[name]
}

// List returns all registered adapters
func (r *AdapterRegistry) List() map[string]ProviderAdapter {
	return r.adapters
}

// BaseAdapter provides default implementations for common provider operations
type BaseAdapter struct {
	ProviderName string
	ProviderVersion string
}

func (b *BaseAdapter) Name() string {
	return b.ProviderName
}

func (b *BaseAdapter) Version() string {
	return b.ProviderVersion
}

// DefaultCapabilities returns a standard set of capabilities
func (b *BaseAdapter) Capabilities(ctx context.Context, resourceType ResourceType) []Capability {
	return []Capability{
		{
			Name:        "observation",
			Level:       "full",
			Description: "Can observe and track resource state",
		},
		{
			Name:        "metadata_export",
			Level:       "basic",
			Description: "Can export resource configuration",
		},
	}
}

// DefaultHealth returns a placeholder health status
func (b *BaseAdapter) DefaultHealth(config *ProviderConfig) HealthStatus {
	return HealthStatus{
		Healthy:   false,
		LastCheck: time.Now(),
		Message:   "Health check not implemented",
	}
}

// QualificationAdapter extends ProviderAdapter with evidence binding capabilities
type QualificationAdapter interface {
	// QualifyResource runs all P1_CORE gates for a single resource
	QualifyResource(ctx context.Context, resource *Resource, signer *EvidenceQualifier) (*QualificationCampaign, error)

	// QualifyDiscovery runs qualification on all discovered resources
	QualifyDiscovery(ctx context.Context, resources []*Resource, signer *EvidenceQualifier) ([]*QualificationCampaign, error)
}

// AdapterWithQualification combines base adapter with qualification capabilities
type AdapterWithQualification struct {
	Adapter ProviderAdapter
	Signer  *EvidenceQualifier
}

// QualifyResource implements QualificationAdapter
func (aq *AdapterWithQualification) QualifyResource(ctx context.Context, resource *Resource, signer *EvidenceQualifier) (*QualificationCampaign, error) {
	if signer == nil {
		return nil, nil // v0.1: Skip qualification if no signer provided
	}

	// Get source SHA from evidence if available
	sourceSHA := ""
	if resource.Evidence != nil {
		sourceSHA = resource.Evidence.SourceSHA
	}
	if sourceSHA == "" {
		sourceSHA = "unknown"
	}

	// v0.1: Return PASSED campaign (stub)
	// v0.2: Will implement actual gate execution
	campaign, err := RunQualificationCampaign(resource.ID, sourceSHA, signer)
	if err != nil {
		return nil, err
	}

	return campaign, nil
}

// QualifyDiscovery implements QualificationAdapter
func (aq *AdapterWithQualification) QualifyDiscovery(ctx context.Context, resources []*Resource, signer *EvidenceQualifier) ([]*QualificationCampaign, error) {
	campaigns := make([]*QualificationCampaign, 0, len(resources))

	for _, resource := range resources {
		campaign, err := aq.QualifyResource(ctx, resource, signer)
		if err != nil {
			// v0.1: Log qualification errors but continue
			continue
		}

		if campaign != nil {
			campaigns = append(campaigns, campaign)
		}
	}

	return campaigns, nil
}

