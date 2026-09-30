package providers

import (
	"time"
)

// ResourceState represents the lifecycle state of a resource
type ResourceState string

const (
	StateDESIRED    ResourceState = "DESIRED"    // Resource imported, waiting for observation
	StateOBSERVED   ResourceState = "OBSERVED"   // Current state verified from provider
	StateVERIFIED   ResourceState = "VERIFIED"   // State recorded in audit trail
	StateMIGRATED   ResourceState = "MIGRATED"   // Workload moved to owned infrastructure
	StateMIGRATING  ResourceState = "MIGRATING"  // Migration in progress
	StateMIGRATION_FAILED ResourceState = "MIGRATION_FAILED" // Migration failed
)

// ResourceType categorizes different infrastructure objects
type ResourceType string

const (
	TypeRepository    ResourceType = "repository"
	TypeDeployment    ResourceType = "deployment"
	TypeDatabase      ResourceType = "database"
	TypeContainer     ResourceType = "container"
	TypeCluster       ResourceType = "cluster"
	TypeModel         ResourceType = "model"
	TypeSecret        ResourceType = "secret"
	TypeNetwork       ResourceType = "network"
	TypeStorage       ResourceType = "storage"
	TypeFunction      ResourceType = "function"
)

// TrustDomain represents a security/trust boundary for resource classification
type TrustDomain string

const (
	DomainLOCAL      TrustDomain = "LOCAL"       // Data must stay on-premise
	DomainORG_ONLY   TrustDomain = "ORG_ONLY"    // Data within organization only
	DomainPRIVATE    TrustDomain = "PRIVATE"     // Encrypted end-to-end
	DomainSHARED     TrustDomain = "SHARED"      // Can be in public cloud
	DomainUNCLASSIFIED TrustDomain = "UNCLASSIFIED" // No special restrictions
)

// Evidence represents cryptographic proof of resource state
type Evidence struct {
	Signature    string            `json:"signature"`    // Ed25519 signature
	SignedBy     string            `json:"signed_by"`    // Identity that signed
	Timestamp    time.Time         `json:"timestamp"`    // When this was observed
	ContentHash  string            `json:"content_hash"` // BLAKE3 hash of state
	SourceSHA    string            `json:"source_sha"`   // Git commit SHA of source
	Metadata     map[string]string `json:"metadata"`     // Provider-specific details
}

// Capability describes what a provider or resource can do
type Capability struct {
	Name        string            `json:"name"`        // e.g., "encryption", "scaling"
	Level       string            `json:"level"`       // "none", "basic", "full"
	Description string            `json:"description"` // Human explanation
	Evidence    *Evidence         `json:"evidence"`    // Proof this capability exists
	Metadata    map[string]string `json:"metadata"`    // Provider details
}

// Dependency tracks relationships between resources
type Dependency struct {
	TargetID     string    `json:"target_id"`     // ID of dependent resource
	TargetType   ResourceType `json:"target_type"` // Type of dependent resource
	DependencyType string  `json:"type"`         // "deploys", "depends_on", "reads", etc.
	Required     bool      `json:"required"`     // Is this dependency required?
	LastChecked  time.Time `json:"last_checked"` // When dependency was last validated
}

// MigrationPlan describes how to move a resource to owned infrastructure
type MigrationPlan struct {
	ResourceID      string        `json:"resource_id"`
	SourceProvider  string        `json:"source_provider"`
	TargetLocation  string        `json:"target_location"`
	EstimatedCost   string        `json:"estimated_cost"`       // e.g., "$50/month"
	EstimatedTime   time.Duration `json:"estimated_time"`       // How long to migrate
	DowntimeWindow  time.Duration `json:"downtime_window"`      // Expected downtime
	PrerequisiteIDs []string      `json:"prerequisite_ids"`     // Must migrate these first
	RollbackPlan    string        `json:"rollback_plan"`        // How to revert if needed
	Steps           []MigrationStep `json:"steps"`              // Ordered steps to execute
}

// MigrationStep is one unit of work in a migration
type MigrationStep struct {
	Sequence    int       `json:"sequence"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Action      string    `json:"action"`    // e.g., "backup", "sync", "cutover", "verify"
	Timeout     time.Duration `json:"timeout"`
	Rollback    string    `json:"rollback"`  // How to undo this step
}

// Resource represents any infrastructure object that can be imported into the graph
type Resource struct {
	// Identity
	ID                string      `json:"id"`                  // Unique ID in graph (deterministic)
	Provider          string      `json:"provider"`           // Provider name (github, vercel, supabase, etc.)
	ProviderResourceID string     `json:"provider_resource_id"` // ID in the source provider
	ProjectID         string      `json:"project_id"`         // Which project owns this

	// Classification
	Type              ResourceType `json:"type"`              // What is this resource?
	Name              string      `json:"name"`              // Human name
	Description       string      `json:"description"`       // What it does

	// State
	DesiredState      string      `json:"desired_state"`     // What state we want
	ObservedState     string      `json:"observed_state"`    // Current state from provider
	VerificationState ResourceState `json:"verification_state"` // Is observed state verified?
	MigrationState    ResourceState `json:"migration_state"`    // Migration progress

	// Location & Trust
	Location          string      `json:"location"`          // Where data lives (us-west, eu, etc.)
	Owner             string      `json:"owner"`             // User who owns this
	TrustDomain       TrustDomain `json:"trust_domain"`      // Data classification

	// Relationships
	Dependencies      []Dependency `json:"dependencies"`     // What this resource depends on
	Dependents        []string     `json:"dependents"`       // Resources that depend on this

	// Capabilities
	Capabilities      map[string]Capability `json:"capabilities"` // What this can do

	// Evidence & Audit
	Evidence          *Evidence   `json:"evidence"`          // Proof of current state
	LastObservedAt    time.Time   `json:"last_observed_at"`  // When we last checked state
	LastModifiedAt    time.Time   `json:"last_modified_at"`  // When provider last changed this
	CreatedAt         time.Time   `json:"created_at"`        // When created

	// Migration
	MigrationPlan     *MigrationPlan `json:"migration_plan"`  // How to move this resource
	MigrationHistory  []Evidence  `json:"migration_history"` // Past migration attempts

	// Metadata
	Labels            map[string]string `json:"labels"`      // Arbitrary tags
	Annotations       map[string]string `json:"annotations"` // Provider-specific notes
}

// Graph represents the unified view of all resources
type Graph struct {
	ProjectID    string              `json:"project_id"`
	Resources    map[string]*Resource `json:"resources"`     // ID → Resource
	CreatedAt    time.Time           `json:"created_at"`
	LastUpdatedAt time.Time          `json:"last_updated_at"`
}

// ProviderConfig holds authentication and connection details for a provider
type ProviderConfig struct {
	Name       string            `json:"name"`       // Provider identifier
	Provider   string            `json:"provider"`   // Full provider name
	Endpoint   string            `json:"endpoint"`   // API endpoint
	Token      string            `json:"-"`          // Secret (never serialize)
	ProjectID  string            `json:"project_id"` // Project or org in provider
	Metadata   map[string]string `json:"metadata"`   // Extra config options
}

// DiscoverResult is what a provider returns when discovering resources
type DiscoverResult struct {
	Resources []*Resource
	Error     error
}

// ObserveResult is what a provider returns when checking resource state
type ObserveResult struct {
	State    string
	Evidence *Evidence
	Error    error
}

// DiffResult shows changes between source and target
type DiffResult struct {
	Added    []string          `json:"added"`    // New in target
	Removed  []string          `json:"removed"`  // Removed from target
	Modified map[string]string `json:"modified"` // Changed fields
	Error    error             `json:"-"`
}

// HealthStatus represents provider connectivity status
type HealthStatus struct {
	Healthy   bool      `json:"healthy"`
	LastCheck time.Time `json:"last_check"`
	Message   string    `json:"message"`
	Error     string    `json:"error,omitempty"`
}
