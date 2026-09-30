package control

import (
	"encoding/json"
	"net/http"
	"time"

	"decentralized.host/pkg/providers"
)

// handleQualificationCampaign starts a qualification campaign for a resource
func (s *Server) handleQualificationCampaign(w http.ResponseWriter, r *http.Request, a *authz) {
	var request map[string]string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeErr(w, 400, "invalid request: %v", err)
		return
	}

	resourceID := request["resource_id"]
	sourceSHA := request["source_sha"]

	if resourceID == "" {
		writeErr(w, 400, "resource_id required")
		return
	}

	if sourceSHA == "" {
		sourceSHA = "unspecified"
	}

	// Create evidence qualifier for signing
	signer, err := providers.NewEvidenceQualifier(a.Actor)
	if err != nil {
		writeErr(w, 500, "failed to create evidence qualifier: %v", err)
		return
	}

	// Run qualification campaign (v0.1: stub, v0.2: actual gates)
	campaign, err := providers.RunQualificationCampaign(resourceID, sourceSHA, signer)
	if err != nil {
		writeErr(w, 500, "qualification campaign failed: %v", err)
		return
	}

	response := map[string]interface{}{
		"campaign_id": campaign.ID,
		"resource_id": campaign.ResourceID,
		"status":      campaign.Status,
		"passed":      campaign.Evidence != nil && campaign.Evidence.P1_CORE_Passed,
		"start_time":  campaign.StartTime,
		"end_time":    campaign.EndTime,
		"gate_count":  len(campaign.GateResults),
		"gates_passed": func() int {
			count := 0
			for _, g := range campaign.GateResults {
				if g.Passed {
					count++
				}
			}
			return count
		}(),
	}

	writeJSON(w, 200, response)
}

// handleChaosTest runs a chaos scenario against provider configuration
func (s *Server) handleChaosTest(w http.ResponseWriter, r *http.Request, a *authz) {
	var request map[string]string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeErr(w, 400, "invalid request: %v", err)
		return
	}

	scenarioID := request["scenario_id"]
	providerName := request["provider"]

	if scenarioID == "" && providerName == "" {
		writeErr(w, 400, "scenario_id or provider required")
		return
	}

	// Find matching chaos scenario
	var scenario *providers.ChaosScenario
	for i := range providers.Chaos17Scenarios {
		if providers.Chaos17Scenarios[i].ID == scenarioID {
			scenario = &providers.Chaos17Scenarios[i]
			break
		}
	}

	if scenario == nil && scenarioID != "" {
		writeErr(w, 404, "scenario not found: %s", scenarioID)
		return
	}

	// If no specific scenario, return full suite results
	if scenario == nil {
		var cfg providers.ProviderConfig
		cfg.Name = providerName

		results, allPassed := providers.RunChaosTestSuite(r.Context(), &cfg)

		response := map[string]interface{}{
			"suite_id":         "chaos-suite-" + string(rune(time.Now().UnixNano())),
			"scenarios_run":    len(results),
			"scenarios_passed": 0,
			"invariants_held":  allPassed,
			"results":          results,
		}

		// Count passed scenarios
		passed := 0
		for _, res := range results {
			if res.Passed {
				passed++
			}
		}
		response["scenarios_passed"] = passed

		writeJSON(w, 200, response)
		return
	}

	// Run single scenario
	var cfg providers.ProviderConfig
	cfg.Name = providerName

	result, err := providers.RunChaosTest(r.Context(), *scenario, &cfg)
	if err != nil {
		writeErr(w, 500, "scenario execution failed: %v", err)
		return
	}

	response := map[string]interface{}{
		"scenario_id": scenario.ID,
		"scenario":    scenario.Name,
		"passed":      result.Passed,
		"start_time":  result.StartTime,
		"end_time":    result.EndTime,
		"evidence":    result.Evidence,
		"error":       result.ErrorMsg,
	}

	writeJSON(w, 200, response)
}

// handleRuntimeDetection discovers the runtime backend topology
func (s *Server) handleRuntimeDetection(w http.ResponseWriter, r *http.Request, _ *authz) {
	topology, err := providers.DetectRuntime(r.Context())
	if err != nil {
		writeErr(w, 500, "runtime detection failed: %v", err)
		return
	}

	response := map[string]interface{}{
		"node_count":       topology.NodeCount,
		"physical_hosts":   topology.PhysicalHosts,
		"vm_hosts":         topology.VMHosts,
		"containers":       topology.Containers,
		"operator_domains": topology.OperatorDomains,
		"detection_method": topology.DetectionMethod,
		"confidence":       topology.Confidence,
		"nodes": func() []map[string]interface{} {
			nodeList := make([]map[string]interface{}, len(topology.Nodes))
			for i, n := range topology.Nodes {
				nodeList[i] = map[string]interface{}{
					"id":                n.ID,
					"hostname":          n.Hostname,
					"backend":           n.Backend,
					"isolation_level":   n.IsolationLevel,
					"os_boundary":       n.OSBoundary,
					"filesystem_bound":  n.FilesystemBound,
					"physical_bound":    n.PhysicalBound,
					"operator_bound":    n.OperatorBound,
				}
			}
			return nodeList
		}(),
	}

	writeJSON(w, 200, response)
}

// handleEvidenceVerification verifies a signed evidence record
func (s *Server) handleEvidenceVerification(w http.ResponseWriter, r *http.Request, _ *authz) {
	var evidence providers.QualifiedEvidence
	if err := json.NewDecoder(r.Body).Decode(&evidence); err != nil {
		writeErr(w, 400, "invalid evidence: %v", err)
		return
	}

	err := providers.VerifyEvidence(&evidence)
	if err != nil {
		response := map[string]interface{}{
			"verified":  false,
			"error":     err.Error(),
			"campaign":  evidence.CampaignID,
			"resource":  evidence.ResourceID,
			"timestamp": evidence.Timestamp,
		}
		writeJSON(w, 200, response)
		return
	}

	response := map[string]interface{}{
		"verified":             true,
		"campaign":             evidence.CampaignID,
		"resource":             evidence.ResourceID,
		"signer_id":            evidence.SignerID,
		"timestamp":            evidence.Timestamp,
		"p1_core_passed":       evidence.P1_CORE_Passed,
		"p1_qemu_passed":       evidence.P1_QEMU_Passed,
		"p1_k8s_passed":        evidence.P1_K8S_Passed,
		"p2_multi_passed":      evidence.P2_Multi_Passed,
		"os_boundary":          evidence.OSBoundary,
		"filesystem_boundary":  evidence.FilesystemBoundary,
		"physical_boundary":    evidence.PhysicalBoundary,
		"operator_boundary":    evidence.OperatorBoundary,
	}

	writeJSON(w, 200, response)
}
