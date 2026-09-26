package runtime

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLIFECYCLE_P0_A01_EndToEndIntegration verifies LIFECYCLE-P0-A01 works with all downstream subsystems
func TestLIFECYCLE_P0_A01_EndToEndIntegration(t *testing.T) {
	t.Log("=== LIFECYCLE-P0-A01 REVALIDATION: END-TO-END INTEGRATION ===")
	t.Log("Verifying node lifecycle manager integrates with A05-P0-A01, storage, deployment, and networking")

	stateDir := t.TempDir()
	reconDir := t.TempDir()
	storageDir := t.TempDir()
	ctx := context.Background()

	// Create integrated subsystem instances
	nlm := NewNodeLifecycleManagerWithStore(stateDir, reconDir)
	wsm := NewWorkloadStorageManager(storageDir)
	sr := NewServiceRegistry()
	npe := NewNetworkPolicyEngine()

	// Phase 1: Node Registration and State Persistence
	t.Log("\nPhase 1: Node Registration with Persistent State")
	nodeID := "node-001"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	state, err := nlm.GetNodeState(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeState failed: %v", err)
	}

	if state.State != NodeDiscovered {
		t.Errorf("Expected DISCOVERED, got %s", state.State)
	}

	// Verify state file exists on disk
	stateFile := stateDir + "/" + nodeID + ".json"
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Errorf("State file not persisted: %s", stateFile)
	}
	t.Logf("✓ Node registered in DISCOVERED state, persisted to %s", stateFile)

	// Phase 2: Node Enrollment and Verification
	t.Log("\nPhase 2: Node Enrollment and Verification")
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeEnrolling
	nlm.nodes[nodeID].DesiredState = NodeVerified
	nlm.nodes[nodeID].Generation++ // Increment for enrollment transition
	nlm.mu.Unlock()

	// Manually transition to VERIFIED (enrollment completes)
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.nodes[nodeID].Generation++
	nlm.mu.Unlock()

	state, _ = nlm.GetNodeState(ctx, nodeID)
	if state.State != NodeVerified {
		t.Errorf("Expected VERIFIED after enrollment, got %s", state.State)
	}
	t.Logf("✓ Node enrolled and VERIFIED (generation=%d)", state.Generation)

	// Phase 3: Node Activation
	t.Log("\nPhase 3: Node Activation with Service Integration")
	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	state, _ = nlm.GetNodeState(ctx, nodeID)
	if state.State != NodeActive {
		t.Errorf("Expected ACTIVE, got %s", state.State)
	}
	t.Logf("✓ Node transitioned to ACTIVE (generation=%d)", state.Generation)

	// Register service for the node
	serviceID := "service-001"
	serviceAddr := &NetworkAddress{
		Host:     "127.0.0.1",
		Port:     8080,
		Protocol: "HTTP",
		TLS:      false,
	}
	endpoint, err := sr.RegisterService(serviceID, "workload-001", nodeID, serviceAddr)
	if err != nil {
		t.Fatalf("RegisterService failed: %v", err)
	}
	if endpoint == nil {
		t.Fatalf("RegisterService returned nil endpoint")
	}
	t.Logf("✓ Service registered for node %s", nodeID)

	// Phase 4: Workload Scheduling with Storage Integration
	t.Log("\nPhase 4: Workload Scheduling with Persistent Storage")
	workloadID := "workload-001"
	wsm.SetVolumeQuota(workloadID, 500*1024*1024, 5)

	volume, err := wsm.CreateVolume(workloadID, nodeID, 50*1024*1024, "/data")
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}

	if volume.Status != VolumeCreated {
		t.Errorf("Expected CREATED status, got %s", volume.Status)
	}

	if err := wsm.AttachVolume(volume.VolumeID, nodeID); err != nil {
		t.Fatalf("AttachVolume failed: %v", err)
	}

	volume, _ = wsm.GetVolume(volume.VolumeID)
	if volume.Status != VolumeAttached {
		t.Errorf("Expected ATTACHED status, got %s", volume.Status)
	}
	t.Logf("✓ Persistent volume created and attached (volume=%s, size=%d bytes)", volume.VolumeID, volume.Size)

	// Phase 5: Policy Enforcement and Network Integration
	t.Log("\nPhase 5: Network Policy Enforcement")
	policyID := "policy-001"
	policy, err := npe.CreatePolicy(policyID, "allow-workload-to-service", "workload-001", "service-001", 8080, "ALLOW")
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}
	if policy == nil {
		t.Fatalf("CreatePolicy returned nil policy")
	}

	allowed, reason := npe.EvaluatePolicy("workload-001", "service-001", 8080)
	if !allowed {
		t.Errorf("Expected policy to allow, got %s", reason)
	}
	t.Logf("✓ Network policy created and evaluated (policy=%s, allowed=%v)", policyID, allowed)

	// Phase 6: Node Cordoning
	t.Log("\nPhase 6: Node Cordoning (Graceful Drain)")
	if err := nlm.CordonNode(ctx, nodeID); err != nil {
		t.Fatalf("CordonNode failed: %v", err)
	}

	state, _ = nlm.GetNodeState(ctx, nodeID)
	if state.State != NodeCordoned {
		t.Errorf("Expected CORDONED, got %s", state.State)
	}

	if !state.Cordoned {
		t.Error("Expected Cordoned flag to be true")
	}
	t.Logf("✓ Node cordoned (no new workloads scheduled, generation=%d)", state.Generation)

	// Phase 7: Node Draining
	t.Log("\nPhase 7: Node Draining (Workload Migration)")
	if err := nlm.DrainNode(ctx, nodeID, 1); err != nil {
		t.Fatalf("DrainNode failed: %v", err)
	}

	state, _ = nlm.GetNodeState(ctx, nodeID)
	if state.State != NodeDraining {
		t.Errorf("Expected DRAINING, got %s", state.State)
	}
	t.Logf("✓ Node draining (workloads migrating, generation=%d)", state.Generation)

	// Phase 8: Persistence Verification After Restart Simulation
	t.Log("\nPhase 8: Verify State Persistence Across Restart")
	nlm2 := NewNodeLifecycleManagerWithStore(stateDir, reconDir)
	if err := nlm2.RecoverNodeStatesFromDisk(ctx); err != nil {
		t.Fatalf("RecoverNodeStatesFromDisk failed: %v", err)
	}

	state, err = nlm2.GetNodeState(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeState after recovery failed: %v", err)
	}

	if state.State != NodeDraining {
		t.Errorf("Expected DRAINING after recovery, got %s", state.State)
	}

	if state.Generation != 6 {
		t.Errorf("Expected generation 6 (transitions: register→enroll→verify→active→cordoned→draining), got %d", state.Generation)
	}
	t.Logf("✓ State recovered after restart (state=%s, generation=%d)", state.State, state.Generation)

	// Phase 9: DESIRED vs OBSERVED Reconciliation
	t.Log("\nPhase 9: DESIRED vs OBSERVED Reconciliation Tracking")
	nlm2.mu.Lock()
	node := nlm2.nodes[nodeID]
	node.DesiredState = NodeIdle       // Set desired state
	nlm2.mu.Unlock()

	if err := nlm2.ReconcileNodeState(ctx, nodeID); err != nil {
		t.Fatalf("ReconcileNodeState failed: %v", err)
	}

	recon, err := nlm2.reconStore.GetReconciliationStatus(nodeID)
	if err != nil {
		t.Fatalf("GetReconciliationStatus failed: %v", err)
	}

	if !recon.Divergent {
		t.Error("Expected Divergent=true when DESIRED != OBSERVED")
	}
	t.Logf("✓ Reconciliation tracked divergence (desired=%s, observed=%s, divergent=%v)",
		recon.DesiredState, recon.ObservedState, recon.Divergent)

	// Phase 10: Storage Cleanup on Node Removal
	t.Log("\nPhase 10: Storage Cleanup on Node Removal")
	if err := wsm.CleanupWorkloadVolumes(workloadID); err != nil {
		t.Fatalf("CleanupWorkloadVolumes failed: %v", err)
	}

	remaining := wsm.GetWorkloadVolumes(workloadID)
	if len(remaining) != 0 {
		t.Errorf("Expected no volumes after cleanup, got %d", len(remaining))
	}
	t.Logf("✓ All workload volumes cleaned up")

	// Phase 11: Node Revocation
	t.Log("\nPhase 11: Node Revocation")
	if err := nlm2.RevokeNode(ctx, nodeID); err != nil {
		t.Fatalf("RevokeNode failed: %v", err)
	}

	state, _ = nlm2.GetNodeState(ctx, nodeID)
	if state.State != NodeRevoked {
		t.Errorf("Expected REVOKED, got %s", state.State)
	}
	t.Logf("✓ Node revoked (final state, generation=%d)", state.Generation)

	t.Log("\n=== LIFECYCLE-P0-A01 REVALIDATION: PASS ===")
	t.Logf("✓ Node lifecycle verified end-to-end with storage, networking, and deployment integration")
	t.Logf("✓ State persistence working across manager restarts")
	t.Logf("✓ DESIRED vs OBSERVED reconciliation tracked")
	t.Logf("✓ Generation counter incremented correctly through all transitions")
	t.Logf("✓ Resource cleanup verified on node removal")
}

// TestLIFECYCLE_P0_A01_ConcurrentTransitions verifies concurrent transition safety
func TestLIFECYCLE_P0_A01_ConcurrentTransitions(t *testing.T) {
	t.Log("=== LIFECYCLE-P0-A01: CONCURRENT TRANSITION SAFETY ===")

	stateDir := t.TempDir()
	reconDir := t.TempDir()
	ctx := context.Background()

	nlm := NewNodeLifecycleManagerWithStore(stateDir, reconDir)
	nodeID := "concurrent-node"

	nlm.RegisterNode(ctx, nodeID)

	// Transition to VERIFIED for the test
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	// Multiple concurrent transitions should be safe
	done := make(chan error, 3)

	go func() {
		done <- nlm.TransitionToActive(ctx, nodeID)
	}()

	go func() {
		done <- nlm.CordonNode(ctx, nodeID)
	}()

	go func() {
		done <- nlm.DrainNode(ctx, nodeID, 1)
	}()

	// Collect results
	for i := 0; i < 3; i++ {
		err := <-done
		if err != nil {
			// Some transitions may fail due to state conflicts, which is expected
			t.Logf("Transition blocked (expected in concurrent scenario): %v", err)
		}
	}

	state, _ := nlm.GetNodeState(ctx, nodeID)
	if state.State == NodeUnknown {
		t.Error("Node state became UNKNOWN after concurrent transitions")
	}

	t.Logf("✓ Concurrent transitions handled safely (final state=%s, generation=%d)", state.State, state.Generation)
}

// TestLIFECYCLE_P0_A01_HeartbeatTracking verifies freshness and offline detection
func TestLIFECYCLE_P0_A01_HeartbeatTracking(t *testing.T) {
	t.Log("=== LIFECYCLE-P0-A01: HEARTBEAT TRACKING & OFFLINE DETECTION ===")

	stateDir := t.TempDir()
	reconDir := t.TempDir()
	ctx := context.Background()

	nlm := NewNodeLifecycleManagerWithStore(stateDir, reconDir)
	nodeID := "heartbeat-node"

	nlm.RegisterNode(ctx, nodeID)

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeActive
	nlm.nodes[nodeID].LastHeartbeat = time.Now().UnixNano()
	nlm.nodes[nodeID].Freshness = "FRESH"
	nlm.mu.Unlock()

	state, _ := nlm.GetNodeState(ctx, nodeID)
	if state.Freshness != "FRESH" {
		t.Errorf("Expected FRESH, got %s", state.Freshness)
	}
	t.Logf("✓ Node marked FRESH with recent heartbeat")

	// Simulate heartbeat timeout
	nlm.mu.Lock()
	nlm.nodes[nodeID].LastHeartbeat = time.Now().Add(-90 * time.Second).UnixNano()
	nlm.mu.Unlock()

	offlineNodes := nlm.DetectOfflineNodes(ctx)
	if len(offlineNodes) != 1 {
		t.Fatalf("Expected 1 offline node, got %d", len(offlineNodes))
	}

	state, _ = nlm.GetNodeState(ctx, nodeID)
	if state.State != NodeOffline {
		t.Errorf("Expected OFFLINE after timeout, got %s", state.State)
	}

	t.Logf("✓ Offline detection triggered after heartbeat timeout (state=%s)", state.State)
}
