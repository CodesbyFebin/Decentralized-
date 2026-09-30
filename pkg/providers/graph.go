package providers

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ProjectGraph manages the unified view of all imported resources
type ProjectGraph struct {
	projectID       string
	resources       map[string]*Resource
	mu              sync.RWMutex
	createdAt       time.Time
	lastUpdatedAt   time.Time
	dependencyIndex map[string][]string // resourceID → list of dependent IDs
}

// NewProjectGraph creates a new unified resource graph for a project
func NewProjectGraph(projectID string) *ProjectGraph {
	return &ProjectGraph{
		projectID:       projectID,
		resources:       make(map[string]*Resource),
		createdAt:       time.Now(),
		lastUpdatedAt:   time.Now(),
		dependencyIndex: make(map[string][]string),
	}
}

// AddResource adds a resource to the graph
func (g *ProjectGraph) AddResource(resource *Resource) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if resource.ID == "" {
		return fmt.Errorf("resource must have an ID")
	}

	if _, exists := g.resources[resource.ID]; exists {
		return fmt.Errorf("resource %s already exists", resource.ID)
	}

	g.resources[resource.ID] = resource
	g.lastUpdatedAt = time.Now()

	// Update dependency index
	for _, dep := range resource.Dependencies {
		g.dependencyIndex[dep.TargetID] = append(g.dependencyIndex[dep.TargetID], resource.ID)
	}

	return nil
}

// UpdateResource updates an existing resource in the graph
func (g *ProjectGraph) UpdateResource(resource *Resource) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.resources[resource.ID]; !exists {
		return fmt.Errorf("resource %s not found", resource.ID)
	}

	g.resources[resource.ID] = resource
	g.lastUpdatedAt = time.Now()
	return nil
}

// GetResource retrieves a resource by ID
func (g *ProjectGraph) GetResource(id string) (*Resource, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	resource, exists := g.resources[id]
	if !exists {
		return nil, fmt.Errorf("resource %s not found", id)
	}
	return resource, nil
}

// ListResources returns all resources in the graph
func (g *ProjectGraph) ListResources() []*Resource {
	g.mu.RLock()
	defer g.mu.RUnlock()

	resources := make([]*Resource, 0, len(g.resources))
	for _, r := range g.resources {
		resources = append(resources, r)
	}
	return resources
}

// ListResourcesByType returns resources of a specific type
func (g *ProjectGraph) ListResourcesByType(resourceType ResourceType) []*Resource {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var resources []*Resource
	for _, r := range g.resources {
		if r.Type == resourceType {
			resources = append(resources, r)
		}
	}
	return resources
}

// ListResourcesByProvider returns resources from a specific provider
func (g *ProjectGraph) ListResourcesByProvider(provider string) []*Resource {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var resources []*Resource
	for _, r := range g.resources {
		if r.Provider == provider {
			resources = append(resources, r)
		}
	}
	return resources
}

// ListResourcesByTrustDomain returns resources with a specific trust classification
func (g *ProjectGraph) ListResourcesByTrustDomain(domain TrustDomain) []*Resource {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var resources []*Resource
	for _, r := range g.resources {
		if r.TrustDomain == domain {
			resources = append(resources, r)
		}
	}
	return resources
}

// GetDependents returns resources that depend on the given resource
func (g *ProjectGraph) GetDependents(resourceID string) []*Resource {
	g.mu.RLock()
	defer g.mu.RUnlock()

	dependentIDs := g.dependencyIndex[resourceID]
	resources := make([]*Resource, 0, len(dependentIDs))

	for _, depID := range dependentIDs {
		if r, exists := g.resources[depID]; exists {
			resources = append(resources, r)
		}
	}
	return resources
}

// GetDependencies returns resources that the given resource depends on
func (g *ProjectGraph) GetDependencies(resourceID string) ([]*Resource, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	resource, exists := g.resources[resourceID]
	if !exists {
		return nil, fmt.Errorf("resource %s not found", resourceID)
	}

	dependencies := make([]*Resource, 0, len(resource.Dependencies))
	for _, dep := range resource.Dependencies {
		if r, exists := g.resources[dep.TargetID]; exists {
			dependencies = append(dependencies, r)
		}
	}
	return dependencies, nil
}

// AnalyzeSovereignty calculates sovereignty score for resources
// Returns a map of resourceID -> sovereignty score (0.0 to 1.0)
// Higher score = more sovereign (can be migrated)
func (g *ProjectGraph) AnalyzeSovereignty() map[string]float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	scores := make(map[string]float64)

	for id, resource := range g.resources {
		score := 1.0

		// Locked-in providers (Vercel, Heroku) = lower sovereignty
		lockedInProviders := map[string]float64{
			"vercel":  0.3,
			"heroku":  0.3,
			"railway": 0.4,
		}
		if penaltyFactor, isLocked := lockedInProviders[resource.Provider]; isLocked {
			score *= penaltyFactor
		}

		// Proprietary data formats = lower sovereignty
		proprietaryTypes := map[ResourceType]float64{
			TypeCluster: 0.5, // Kubernetes clusters harder to migrate
			TypeModel:   0.6, // ML models need retraining
		}
		if penaltyFactor, isProprietary := proprietaryTypes[resource.Type]; isProprietary {
			score *= penaltyFactor
		}

		// Many dependencies = lower sovereignty (harder to migrate)
		if len(resource.Dependencies) > 5 {
			score *= 0.7
		} else if len(resource.Dependencies) > 0 {
			score *= 0.85
		}

		// Trust domain affects sovereignty
		trustPenalties := map[TrustDomain]float64{
			DomainLOCAL:      1.0, // Already sovereign
			DomainORG_ONLY:   0.8,
			DomainPRIVATE:    0.7,
			DomainSHARED:     0.6,
			DomainUNCLASSIFIED: 0.5,
		}
		if penalty, exists := trustPenalties[resource.TrustDomain]; exists {
			score *= penalty
		}

		scores[id] = score
	}

	return scores
}

// AnalyzePrivacy returns resources with sensitive data locations
func (g *ProjectGraph) AnalyzePrivacy() map[string]PrivacyAnalysis {
	g.mu.RLock()
	defer g.mu.RUnlock()

	analysis := make(map[string]PrivacyAnalysis)

	for id, resource := range g.resources {
		pa := PrivacyAnalysis{
			ResourceID:   id,
			Location:     resource.Location,
			TrustDomain:  resource.TrustDomain,
			RiskLevel:    "low",
		}

		// Assess risk
		if resource.Location != "" && resource.Location != "on-premise" {
			pa.RiskLevel = "medium"
		}
		if resource.TrustDomain == DomainSHARED || resource.TrustDomain == DomainUNCLASSIFIED {
			pa.RiskLevel = "high"
		}
		if resource.TrustDomain == DomainLOCAL {
			pa.RiskLevel = "none"
		}

		// Flag if data crosses trust boundaries
		for _, dep := range resource.Dependencies {
			if dep.TargetID != "" {
				if target, exists := g.resources[dep.TargetID]; exists {
					if target.TrustDomain != resource.TrustDomain {
						pa.CrossesTrustBoundary = true
					}
				}
			}
		}

		analysis[id] = pa
	}

	return analysis
}

// EstimateCostSavings calculates potential savings from migration
func (g *ProjectGraph) EstimateCostSavings() map[string]CostAnalysis {
	g.mu.RLock()
	defer g.mu.RUnlock()

	analysis := make(map[string]CostAnalysis)

	// This is a placeholder for v0.1
	// Real cost analysis requires provider-specific pricing models
	for id, resource := range g.resources {
		ca := CostAnalysis{
			ResourceID:      id,
			CurrentProvider: resource.Provider,
			CurrentCost:     "$?/month", // Requires provider API
			OwnedCost:       "$?/month",
			MonthlySavings:  "$?",
			BreakEvenMonths: 0,
		}
		analysis[id] = ca
	}

	return analysis
}

// Serialize exports the graph to JSON
func (g *ProjectGraph) Serialize() ([]byte, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	graph := &Graph{
		ProjectID:     g.projectID,
		Resources:     g.resources,
		CreatedAt:     g.createdAt,
		LastUpdatedAt: g.lastUpdatedAt,
	}

	return json.MarshalIndent(graph, "", "  ")
}

// Deserialize imports a graph from JSON
func DeserializeGraph(data []byte) (*ProjectGraph, error) {
	var graph Graph
	if err := json.Unmarshal(data, &graph); err != nil {
		return nil, err
	}

	pg := NewProjectGraph(graph.ProjectID)
	pg.createdAt = graph.CreatedAt
	pg.lastUpdatedAt = graph.LastUpdatedAt
	pg.resources = graph.Resources

	// Rebuild dependency index
	for id, resource := range graph.Resources {
		for _, dep := range resource.Dependencies {
			pg.dependencyIndex[dep.TargetID] = append(pg.dependencyIndex[dep.TargetID], id)
		}
	}

	return pg, nil
}

// PrivacyAnalysis describes data handling and risk for a resource
type PrivacyAnalysis struct {
	ResourceID           string
	Location             string
	TrustDomain          TrustDomain
	RiskLevel            string // "none", "low", "medium", "high"
	CrossesTrustBoundary bool
}

// CostAnalysis shows economic impact of migration
type CostAnalysis struct {
	ResourceID      string
	CurrentProvider string
	CurrentCost     string // e.g., "$500/month"
	OwnedCost       string // e.g., "$200/month"
	MonthlySavings  string // e.g., "$300/month"
	BreakEvenMonths int    // Months to recoup migration cost
}
