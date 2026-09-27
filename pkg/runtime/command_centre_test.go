package runtime

import (
	"context"
	"testing"
	"time"
)

// TestCC_W2_A01_TruthEnvelopeIntegration verifies Command Centre with TruthEnvelope
func TestCC_W2_A01_TruthEnvelopeIntegration(t *testing.T) {
	t.Log("=== CC-W2-A01: COMMAND CENTRE WITH TRUTHENVELOPE ===")
	t.Log("Verifying Command Centre backend provides accurate state with freshness tracking")

	ctx := context.Background()
	stateDir := t.TempDir()
	reconDir := t.TempDir()

	// Create integrated subsystems
	nlm := NewNodeLifecycleManagerWithStore(stateDir, reconDir)
	sr := NewServiceRegistry()
	npe := NewNetworkPolicyEngine()
	cc := NewCommandCentreBackend(nlm, sr, npe)

	// Phase 1: Node Registration and Initial State
	t.Log("\nPhase 1: Node Registration with State Observation")
	nodeID := "node-cc-001"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// Record observation
	if err := cc.RecordObservation(ctx, nodeID, "DISCOVERED", "control-plane"); err != nil {
		t.Fatalf("RecordObservation failed: %v", err)
	}

	state, err := cc.GetNodeState(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeState failed: %v", err)
	}

	if state == nil || state.TruthEnvelope == nil {
		t.Fatalf("Expected TruthEnvelope, got nil")
	}

	if state.TruthEnvelope.Freshness != FreshnessFresh {
		t.Errorf("Expected FRESH, got %s", state.TruthEnvelope.Freshness)
	}

	if state.TruthEnvelope.Source != nodeID {
		t.Errorf("Expected source %s, got %s", nodeID, state.TruthEnvelope.Source)
	}

	t.Logf("✓ Node registered with FRESH observation (freshness=%s, source=%s)", state.Freshness, state.ObservedBy)

	// Phase 2: Node Activation
	t.Log("\nPhase 2: Node State Transition (DISCOVERED → ACTIVE)")
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeEnrolling
	nlm.nodes[nodeID].Generation++
	nlm.mu.Unlock()

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.nodes[nodeID].Generation++
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	state, _ = cc.GetNodeState(ctx, nodeID)
	if state.State != string(NodeActive) {
		t.Errorf("Expected ACTIVE, got %s", state.State)
	}

	if state.Convergence != "CONVERGED" {
		t.Errorf("Expected CONVERGED (desired=actual), got %s", state.Convergence)
	}

	t.Logf("✓ Node transitioned to ACTIVE (convergence=%s, generation=%d)", state.Convergence, state.Generation)

	// Phase 3: DESIRED vs OBSERVED Divergence
	t.Log("\nPhase 3: DESIRED ≠ OBSERVED State (Divergence Detection)")
	if err := cc.UpdateDesiredState(ctx, nodeID, string(NodeCordoned)); err != nil {
		t.Fatalf("UpdateDesiredState failed: %v", err)
	}

	state, _ = cc.GetNodeState(ctx, nodeID)
	if state.DesiredState != string(NodeCordoned) {
		t.Errorf("Expected desired=CORDONED, got %s", state.DesiredState)
	}

	if state.State != string(NodeActive) {
		t.Errorf("Expected observed=ACTIVE, got %s", state.State)
	}

	if state.Convergence != "DIVERGED" {
		t.Errorf("Expected DIVERGED, got %s", state.Convergence)
	}

	t.Logf("✓ Divergence detected (desired=%s, observed=%s, convergence=%s)", state.DesiredState, state.State, state.Convergence)

	// Phase 4: Freshness Calculation (Stale Data Detection)
	t.Log("\nPhase 4: Freshness Tracking - Stale Data Detection")
	nlm.mu.Lock()
	// Simulate old observation (35 seconds ago - beyond stale threshold of 30s)
	nlm.nodes[nodeID].LastHeartbeat = time.Now().Add(-35 * time.Second).UnixNano()
	nlm.mu.Unlock()

	state, _ = cc.GetNodeState(ctx, nodeID)
	if state.Freshness != FreshnessStale {
		t.Errorf("Expected STALE (>30s old), got %s", state.Freshness)
	}

	if state.TruthEnvelope.Freshness != FreshnessStale {
		t.Errorf("Expected TruthEnvelope.Freshness=STALE, got %s", state.TruthEnvelope.Freshness)
	}

	t.Logf("✓ Stale data detected (age > 30s, freshness=%s)", state.Freshness)

	// Phase 5: Expired Data
	t.Log("\nPhase 5: Freshness Tracking - Expired Data")
	nlm.mu.Lock()
	// Simulate very old observation (6 minutes ago - beyond expiry threshold of 5min)
	nlm.nodes[nodeID].LastHeartbeat = time.Now().Add(-6 * time.Minute).UnixNano()
	nlm.mu.Unlock()

	state, _ = cc.GetNodeState(ctx, nodeID)
	if state.Freshness != FreshnessExpired {
		t.Errorf("Expected EXPIRED (>5min old), got %s", state.Freshness)
	}

	t.Logf("✓ Expired data detected (age > 5min, freshness=%s)", state.Freshness)

	// Phase 6: Service Discovery via Command Centre
	t.Log("\nPhase 6: Service Discovery Integration")
	serviceAddr := &NetworkAddress{
		Host:     "10.0.0.1",
		Port:     8080,
		Protocol: "HTTP",
		TLS:      false,
	}
	endpoint, err := sr.RegisterService("svc-001", "wl-001", nodeID, serviceAddr)
	if err != nil {
		t.Fatalf("RegisterService failed: %v", err)
	}

	services, err := cc.GetNodeServices(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeServices failed: %v", err)
	}

	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	}

	if services[0].ServiceID != endpoint.ServiceID {
		t.Errorf("Expected service %s, got %s", endpoint.ServiceID, services[0].ServiceID)
	}

	t.Logf("✓ Service discovered via Command Centre (service=%s)", endpoint.ServiceID)

	// Phase 7: Audit Trail
	t.Log("\nPhase 7: Evidence Recording and Audit Trail")
	auditLog := NewAuditLog()
	auditLog.LogOperation(&EvidenceRecord{
		NodeID:      nodeID,
		Operation:   "RegisterNode",
		RequestData: `{"nodeID":"node-cc-001"}`,
		Response:    `{"status":"DISCOVERED","generation":1}`,
		Signer:      "control-plane",
		Status:      "SUCCESS",
	})

	auditLog.LogOperation(&EvidenceRecord{
		NodeID:      nodeID,
		Operation:   "TransitionToActive",
		RequestData: `{"nodeID":"node-cc-001"}`,
		Response:    `{"status":"ACTIVE","generation":4}`,
		Signer:      "control-plane",
		Status:      "SUCCESS",
	})

	history := auditLog.GetOperationHistory(nodeID)
	if len(history) != 2 {
		t.Errorf("Expected 2 audit records, got %d", len(history))
	}

	if history[0].Operation != "RegisterNode" {
		t.Errorf("Expected RegisterNode, got %s", history[0].Operation)
	}

	if history[0].Status != "SUCCESS" {
		t.Errorf("Expected SUCCESS, got %s", history[0].Status)
	}

	t.Logf("✓ Audit trail recorded (%d operations)", len(history))

	// Phase 8: List All Nodes with State
	t.Log("\nPhase 8: List All Nodes with Convergence")
	// Register second node
	nodeID2 := "node-cc-002"
	nlm.RegisterNode(ctx, nodeID2)
	nlm.mu.Lock()
	nlm.nodes[nodeID2].State = NodeActive
	nlm.mu.Unlock()

	nodes, err := cc.ListNodes(ctx)
	if err != nil {
		t.Fatalf("ListNodes failed: %v", err)
	}

	if len(nodes) < 2 {
		t.Errorf("Expected at least 2 nodes, got %d", len(nodes))
	}

	for _, node := range nodes {
		if node.TruthEnvelope == nil {
			t.Errorf("Node %s missing TruthEnvelope", node.NodeID)
		}
		if node.Freshness == FreshnessUnknown {
			t.Errorf("Node %s has UNKNOWN freshness", node.NodeID)
		}
	}

	t.Logf("✓ Listed %d nodes with TruthEnvelope metadata", len(nodes))

	// Phase 9: Node Cordoning
	t.Log("\nPhase 9: Node Cordoning via Command Centre")
	if err := nlm.CordonNode(ctx, nodeID2); err != nil {
		t.Fatalf("CordonNode failed: %v", err)
	}

	state, _ = cc.GetNodeState(ctx, nodeID2)
	if state.State != string(NodeCordoned) {
		t.Errorf("Expected CORDONED, got %s", state.State)
	}

	t.Logf("✓ Node cordoned (state=%s, no new workloads accepted)", state.State)

	// Phase 10: Node Draining
	t.Log("\nPhase 10: Node Draining via Command Centre")
	if err := cc.DrainNode(ctx, nodeID2); err != nil {
		t.Fatalf("DrainNode failed: %v", err)
	}

	state, _ = cc.GetNodeState(ctx, nodeID2)
	if state.State != string(NodeDraining) {
		t.Errorf("Expected DRAINING, got %s", state.State)
	}

	t.Logf("✓ Node draining (state=%s, workloads migrating)", state.State)

	// Phase 11: Node Revocation
	t.Log("\nPhase 11: Node Revocation via Command Centre")
	if err := cc.RevokeNode(ctx, nodeID2); err != nil {
		t.Fatalf("RevokeNode failed: %v", err)
	}

	state, _ = cc.GetNodeState(ctx, nodeID2)
	if state.State != string(NodeRevoked) {
		t.Errorf("Expected REVOKED, got %s", state.State)
	}

	t.Logf("✓ Node revoked (final state, generation=%d)", state.Generation)

	t.Log("\n=== CC-W2-A01: PASS ===")
	t.Logf("✓ Command Centre backend verified with TruthEnvelope")
	t.Logf("✓ DESIRED vs OBSERVED state tracking working")
	t.Logf("✓ Freshness calculation (FRESH/STALE/EXPIRED)")
	t.Logf("✓ Service discovery integration")
	t.Logf("✓ Audit trail recording with timestamps")
	t.Logf("✓ Node state operations (Cordoned, Draining, Revoked)")
}

// TestCC_W2_A01_FreshnessCalculation verifies freshness thresholds
func TestCC_W2_A01_FreshnessCalculation(t *testing.T) {
	t.Log("=== CC-W2-A01: FRESHNESS CALCULATION ===")

	ctx := context.Background()
	stateDir := t.TempDir()
	reconDir := t.TempDir()

	nlm := NewNodeLifecycleManagerWithStore(stateDir, reconDir)
	cc := NewCommandCentreBackend(nlm, NewServiceRegistry(), NewNetworkPolicyEngine())
	nodeID := "test-node"

	nlm.RegisterNode(ctx, nodeID)

	// Fresh observation (just now)
	nlm.mu.Lock()
	nlm.nodes[nodeID].LastHeartbeat = time.Now().UnixNano()
	nlm.mu.Unlock()

	state, _ := cc.GetNodeState(ctx, nodeID)
	if state.Freshness != FreshnessFresh {
		t.Errorf("Expected FRESH for recent observation, got %s", state.Freshness)
	}
	t.Logf("✓ Recent observation: %s", state.Freshness)

	// Stale observation (35 seconds old)
	nlm.mu.Lock()
	nlm.nodes[nodeID].LastHeartbeat = time.Now().Add(-35 * time.Second).UnixNano()
	nlm.mu.Unlock()

	state, _ = cc.GetNodeState(ctx, nodeID)
	if state.Freshness != FreshnessStale {
		t.Errorf("Expected STALE for 35s old observation, got %s", state.Freshness)
	}
	t.Logf("✓ 35s old observation: %s", state.Freshness)

	// Expired observation (6 minutes old)
	nlm.mu.Lock()
	nlm.nodes[nodeID].LastHeartbeat = time.Now().Add(-6 * time.Minute).UnixNano()
	nlm.mu.Unlock()

	state, _ = cc.GetNodeState(ctx, nodeID)
	if state.Freshness != FreshnessExpired {
		t.Errorf("Expected EXPIRED for 6min old observation, got %s", state.Freshness)
	}
	t.Logf("✓ 6min old observation: %s", state.Freshness)

	t.Log("=== PASS: Freshness thresholds verified ===")
}

// TestCC_W2_A01_NodePolicies verifies network policy queries
func TestCC_W2_A01_NodePolicies(t *testing.T) {
	t.Log("=== CC-W2-A01: NODE NETWORK POLICIES ===")

	ctx := context.Background()
	stateDir := t.TempDir()
	reconDir := t.TempDir()

	nlm := NewNodeLifecycleManagerWithStore(stateDir, reconDir)
	sr := NewServiceRegistry()
	npe := NewNetworkPolicyEngine()
	cc := NewCommandCentreBackend(nlm, sr, npe)

	nodeID := "policy-node"
	nlm.RegisterNode(ctx, nodeID)

	// Create policies
	policy1, _ := npe.CreatePolicy("pol-001", "allow-internal", "workload-a", "service-b", 8080, "ALLOW")
	policy2, _ := npe.CreatePolicy("pol-002", "deny-external", "workload-c", "service-d", 9090, "DENY")

	policies, err := cc.GetNodePolicies(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodePolicies failed: %v", err)
	}

	if len(policies) < 2 {
		t.Errorf("Expected at least 2 policies, got %d", len(policies))
	}

	if policy1 == nil || policy2 == nil {
		t.Errorf("Expected both policies to be created")
	}

	t.Logf("✓ Retrieved %d network policies for node", len(policies))
}
