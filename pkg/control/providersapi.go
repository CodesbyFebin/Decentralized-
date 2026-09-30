package control

import (
	"encoding/json"
	"net/http"
	"time"

	"decentralized.host/pkg/providers"
	"decentralized.host/pkg/providers/adapters"
)

// handleConnectProvider processes provider credential registration
func (s *Server) handleConnectProvider(w http.ResponseWriter, r *http.Request, a *authz) {
	var config providers.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeErr(w, 400, "invalid request: %v", err)
		return
	}

	if config.Name == "" {
		writeErr(w, 400, "provider name required")
		return
	}

	// Propose provider connection to FSM
	res, err := s.propose("connect-provider", a.Actor, map[string]any{
		"provider": config.Name,
		"token":    config.Token,
		"endpoint": config.Endpoint,
	})
	if err != nil {
		writeErr(w, 503, "%v", err)
		return
	}

	code := 200
	if !res.OK {
		code = http.StatusConflict
	}
	writeJSON(w, code, res)
}

// handleProviderGraph returns the unified resource graph
func (s *Server) handleProviderGraph(w http.ResponseWriter, _ *http.Request, _ *authz) {
	var graph providers.Graph
	s.fsm.Read(func(st *State) {
		// Build graph from current state
		graph.ProjectID = "current"
		graph.Resources = make(map[string]*providers.Resource)
		graph.CreatedAt = time.Now()
		graph.LastUpdatedAt = time.Now()

		// In a real implementation, this would iterate through state to build resources
		// For now, return empty graph structure
	})

	writeJSON(w, 200, graph)
}

// handleProviderHealth checks health of all configured providers
func (s *Server) handleProviderHealth(w http.ResponseWriter, r *http.Request, _ *authz) {
	response := map[string]interface{}{
		"providers": make(map[string]interface{}),
	}

	registry := providers.NewAdapterRegistry()
	registry.Register("github", adapters.NewGitHubAdapter())
	registry.Register("vercel", adapters.NewVercelAdapter())
	registry.Register("supabase", adapters.NewSupabaseAdapter())
	registry.Register("docker", adapters.NewDockerAdapter())
	registry.Register("kubernetes", adapters.NewKubernetesAdapter())

	// Placeholder: would iterate through configured providers
	providerStatus := response["providers"].(map[string]interface{})
	providerStatus["github"] = map[string]interface{}{
		"healthy": false,
		"message": "provider credentials not configured",
		"lastCheck": time.Now(),
	}

	writeJSON(w, 200, response)
}

// handleMigrationPlan generates a migration plan for a resource
func (s *Server) handleMigrationPlan(w http.ResponseWriter, r *http.Request, a *authz) {
	var request map[string]string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeErr(w, 400, "invalid request: %v", err)
		return
	}

	resourceID := request["resource_id"]
	targetLocation := request["target_location"]

	if resourceID == "" {
		writeErr(w, 400, "resource_id required")
		return
	}

	if targetLocation == "" {
		targetLocation = "owned-infrastructure"
	}

	// Placeholder migration plan
	plan := &providers.MigrationPlan{
		ResourceID:     resourceID,
		SourceProvider: "unknown",
		TargetLocation: targetLocation,
		EstimatedCost:  "$0 (owned infrastructure)",
		EstimatedTime:  1 * time.Hour,
		DowntimeWindow: 5 * time.Minute,
		Steps: []providers.MigrationStep{
			{
				Sequence:    1,
				Name:        "Export resource",
				Description: "Export resource configuration and data",
				Action:      "export",
				Timeout:     10 * time.Minute,
			},
			{
				Sequence:    2,
				Name:        "Transfer data",
				Description: "Transfer to owned infrastructure",
				Action:      "transfer",
				Timeout:     15 * time.Minute,
			},
			{
				Sequence:    3,
				Name:        "Verify integrity",
				Description: "Verify data integrity post-migration",
				Action:      "verify",
				Timeout:     5 * time.Minute,
			},
		},
	}

	writeJSON(w, 200, plan)
}
