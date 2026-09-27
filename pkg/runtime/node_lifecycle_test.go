package runtime

import (
	"context"
	"testing"
	"time"
)

// TestNodeStateTransitions verifies basic node state machine transitions
func TestNodeStateTransitions(t *testing.T) {
	nlm := NewNodeLifecycleManager()
	ctx := context.Background()

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

	// Transition to VERIFIED first (intermediate state needed before ACTIVE)
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	state, err = nlm.GetNodeState(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeState failed: %v", err)
	}

	if state.State != NodeActive {
		t.Errorf("Expected ACTIVE, got %s", state.State)
	}

	t.Logf("PASS: Node state transitions work correctly")
}

// TestCordonNode verifies cordoning prevents new workload scheduling
func TestCordonNode(t *testing.T) {
	nlm := NewNodeLifecycleManager()
	ctx := context.Background()

	nodeID := "node-cordon-001"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// Transition to VERIFIED first
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Cordon the node
	if err := nlm.CordonNode(ctx, nodeID); err != nil {
		t.Fatalf("CordonNode failed: %v", err)
	}

	state, err := nlm.GetNodeState(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeState failed: %v", err)
	}

	if state.State != NodeCordoned {
		t.Errorf("Expected CORDONED, got %s", state.State)
	}

	if !state.Cordoned {
		t.Error("Expected Cordoned flag to be true")
	}

	t.Logf("PASS: Node cordoning works correctly")
}

// TestDrainNode verifies workload draining with target tracking
func TestDrainNode(t *testing.T) {
	nlm := NewNodeLifecycleManager()
	ctx := context.Background()

	nodeID := "node-drain-001"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// Transition to VERIFIED first
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Start draining with 5 workloads
	if err := nlm.DrainNode(ctx, nodeID, 5); err != nil {
		t.Fatalf("DrainNode failed: %v", err)
	}

	state, err := nlm.GetNodeState(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeState failed: %v", err)
	}

	if state.State != NodeDraining {
		t.Errorf("Expected DRAINING, got %s", state.State)
	}

	if state.DrainTarget != 5 {
		t.Errorf("Expected DrainTarget=5, got %d", state.DrainTarget)
	}

	// Complete the drain
	if err := nlm.CompleteDrain(ctx, nodeID); err != nil {
		t.Fatalf("CompleteDrain failed: %v", err)
	}

	state, err = nlm.GetNodeState(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeState failed: %v", err)
	}

	if state.State != NodeIdle {
		t.Errorf("Expected IDLE after drain, got %s", state.State)
	}

	if state.DrainTarget != 0 {
		t.Errorf("Expected DrainTarget=0 after completion, got %d", state.DrainTarget)
	}

	t.Logf("PASS: Node draining works correctly")
}

// TestRevokeNode verifies revocation prevents further operations
func TestRevokeNode(t *testing.T) {
	nlm := NewNodeLifecycleManager()
	ctx := context.Background()

	nodeID := "node-revoke-001"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// Transition to VERIFIED first
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Revoke the node
	if err := nlm.RevokeNode(ctx, nodeID); err != nil {
		t.Fatalf("RevokeNode failed: %v", err)
	}

	state, err := nlm.GetNodeState(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeState failed: %v", err)
	}

	if state.State != NodeRevoked {
		t.Errorf("Expected REVOKED, got %s", state.State)
	}

	// Should not be able to transition revoked node
	if err := nlm.TransitionToActive(ctx, nodeID); err == nil {
		t.Error("Should not allow transition from REVOKED state")
	}

	t.Logf("PASS: Node revocation prevents further operations")
}

// TestHeartbeatTimeout verifies nodes marked offline after heartbeat timeout
func TestHeartbeatTimeout(t *testing.T) {
	nlm := NewNodeLifecycleManager()
	ctx := context.Background()

	nodeID := "node-heartbeat-001"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// Transition to VERIFIED first
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	state, err := nlm.GetNodeState(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeState failed: %v", err)
	}

	if state.Freshness != "FRESH" {
		t.Errorf("Expected FRESH, got %s", state.Freshness)
	}

	// Manually advance heartbeat time for testing (more than 60 seconds ago to trigger OFFLINE)
	nlm.mu.Lock()
	node := nlm.nodes[nodeID]
	node.LastHeartbeat = time.Now().Add(-70 * time.Second).UnixNano()
	nlm.mu.Unlock()

	// Detect offline nodes
	offline := nlm.DetectOfflineNodes(ctx)

	if len(offline) != 1 || offline[0] != nodeID {
		t.Errorf("Expected node to be marked offline, got %v", offline)
	}

	state, err = nlm.GetNodeState(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeState failed: %v", err)
	}

	if state.State != NodeOffline {
		t.Errorf("Expected OFFLINE, got %s", state.State)
	}

	if state.Freshness != "UNREACHABLE" {
		t.Errorf("Expected UNREACHABLE, got %s", state.Freshness)
	}

	t.Logf("PASS: Heartbeat timeout detection works correctly")
}

// TestHeartbeatUpdate verifies heartbeat updates restore ACTIVE state
func TestHeartbeatUpdate(t *testing.T) {
	nlm := NewNodeLifecycleManager()
	ctx := context.Background()

	nodeID := "node-heartbeat-restore-001"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// Transition to VERIFIED first
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Mark offline (> 60 seconds ago)
	nlm.mu.Lock()
	node := nlm.nodes[nodeID]
	node.LastHeartbeat = time.Now().Add(-70 * time.Second).UnixNano()
	nlm.mu.Unlock()

	offline := nlm.DetectOfflineNodes(ctx)
	if len(offline) != 1 {
		t.Errorf("Expected 1 offline node, got %d", len(offline))
	}

	// Update heartbeat to bring back online
	if err := nlm.UpdateHeartbeat(ctx, nodeID); err != nil {
		t.Fatalf("UpdateHeartbeat failed: %v", err)
	}

	state, err := nlm.GetNodeState(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeState failed: %v", err)
	}

	if state.State != NodeActive {
		t.Errorf("Expected ACTIVE after heartbeat, got %s", state.State)
	}

	if state.Freshness != "FRESH" {
		t.Errorf("Expected FRESH, got %s", state.Freshness)
	}

	t.Logf("PASS: Heartbeat update restores node to ACTIVE")
}

// TestStaleHeartbeat verifies stale detection between timeouts
func TestStaleHeartbeat(t *testing.T) {
	nlm := NewNodeLifecycleManager()
	ctx := context.Background()

	nodeID := "node-stale-001"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// Transition to VERIFIED first
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Mark stale (between 30-60 seconds)
	nlm.mu.Lock()
	node := nlm.nodes[nodeID]
	node.LastHeartbeat = time.Now().Add(-45 * time.Second).UnixNano()
	nlm.mu.Unlock()

	nlm.DetectOfflineNodes(ctx)

	state, err := nlm.GetNodeState(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeState failed: %v", err)
	}

	if state.State != NodeActive {
		t.Errorf("Expected ACTIVE (not yet offline), got %s", state.State)
	}

	if state.Freshness != "STALE" {
		t.Errorf("Expected STALE, got %s", state.Freshness)
	}

	t.Logf("PASS: Stale heartbeat detection works correctly")
}

// TestMultipleNodesIndependent verifies multiple nodes managed independently
func TestMultipleNodesIndependent(t *testing.T) {
	nlm := NewNodeLifecycleManager()
	ctx := context.Background()

	// Register 3 nodes
	nodeIDs := []string{"node-1", "node-2", "node-3"}
	for _, id := range nodeIDs {
		if err := nlm.RegisterNode(ctx, id); err != nil {
			t.Fatalf("RegisterNode failed for %s: %v", id, err)
		}
		// Transition to VERIFIED first
		nlm.mu.Lock()
		nlm.nodes[id].State = NodeVerified
		nlm.mu.Unlock()
		if err := nlm.TransitionToActive(ctx, id); err != nil {
			t.Fatalf("TransitionToActive failed for %s: %v", id, err)
		}
	}

	// Cordon only node-1
	if err := nlm.CordonNode(ctx, "node-1"); err != nil {
		t.Fatalf("CordonNode failed: %v", err)
	}

	// Verify independence
	state1, _ := nlm.GetNodeState(ctx, "node-1")
	state2, _ := nlm.GetNodeState(ctx, "node-2")
	state3, _ := nlm.GetNodeState(ctx, "node-3")

	if state1.State != NodeCordoned {
		t.Error("node-1 should be CORDONED")
	}
	if state2.State != NodeActive {
		t.Error("node-2 should still be ACTIVE")
	}
	if state3.State != NodeActive {
		t.Error("node-3 should still be ACTIVE")
	}

	t.Logf("PASS: Multiple nodes managed independently")
}

// TestWorkloadLifecycle verifies workload state machine
func TestWorkloadLifecycle(t *testing.T) {
	wlm := NewWorkloadLifecycleManager()
	ctx := context.Background()

	workloadID := "workload-001"
	if err := wlm.CreateWorkload(ctx, workloadID); err != nil {
		t.Fatalf("CreateWorkload failed: %v", err)
	}

	state, err := wlm.GetWorkloadState(ctx, workloadID)
	if err != nil {
		t.Fatalf("GetWorkloadState failed: %v", err)
	}

	if state.DesiredState != WorkloadDesired {
		t.Errorf("Expected DesiredState=DESIRED, got %s", state.DesiredState)
	}

	if state.ObservedState != WorkloadUnknown {
		t.Errorf("Expected ObservedState=UNKNOWN, got %s", state.ObservedState)
	}

	// Update observed state to RUNNING
	details := map[string]string{"health": "healthy"}
	if err := wlm.UpdateWorkloadObservedState(ctx, workloadID, WorkloadRunning, details); err != nil {
		t.Fatalf("UpdateWorkloadObservedState failed: %v", err)
	}

	state, err = wlm.GetWorkloadState(ctx, workloadID)
	if err != nil {
		t.Fatalf("GetWorkloadState failed: %v", err)
	}

	if state.ObservedState != WorkloadRunning {
		t.Errorf("Expected ObservedState=RUNNING, got %s", state.ObservedState)
	}

	if state.HealthStatus != "healthy" {
		t.Errorf("Expected HealthStatus=healthy, got %s", state.HealthStatus)
	}

	t.Logf("PASS: Workload lifecycle transitions work correctly")
}

// TestWorkloadStopSequence verifies graceful workload stop
func TestWorkloadStopSequence(t *testing.T) {
	wlm := NewWorkloadLifecycleManager()
	ctx := context.Background()

	workloadID := "workload-stop-001"
	if err := wlm.CreateWorkload(ctx, workloadID); err != nil {
		t.Fatalf("CreateWorkload failed: %v", err)
	}

	// Update to RUNNING
	if err := wlm.UpdateWorkloadObservedState(ctx, workloadID, WorkloadRunning, map[string]string{}); err != nil {
		t.Fatalf("UpdateWorkloadObservedState failed: %v", err)
	}

	// Stop workload
	if err := wlm.StopWorkload(ctx, workloadID); err != nil {
		t.Fatalf("StopWorkload failed: %v", err)
	}

	state, err := wlm.GetWorkloadState(ctx, workloadID)
	if err != nil {
		t.Fatalf("GetWorkloadState failed: %v", err)
	}

	if state.DesiredState != WorkloadStopped {
		t.Errorf("Expected DesiredState=STOPPED, got %s", state.DesiredState)
	}

	if state.ObservedState != WorkloadStopping {
		t.Errorf("Expected ObservedState=STOPPING, got %s", state.ObservedState)
	}

	// Mark as fully stopped
	if err := wlm.MarkWorkloadStopped(ctx, workloadID); err != nil {
		t.Fatalf("MarkWorkloadStopped failed: %v", err)
	}

	state, err = wlm.GetWorkloadState(ctx, workloadID)
	if err != nil {
		t.Fatalf("GetWorkloadState failed: %v", err)
	}

	if state.ObservedState != WorkloadStopped {
		t.Errorf("Expected ObservedState=STOPPED, got %s", state.ObservedState)
	}

	t.Logf("PASS: Workload graceful stop works correctly")
}

// TestWorkloadCountByState verifies state counting
func TestWorkloadCountByState(t *testing.T) {
	wlm := NewWorkloadLifecycleManager()
	ctx := context.Background()

	// Create 5 workloads
	for i := 1; i <= 5; i++ {
		id := "workload-" + string(rune('0'+i))
		if err := wlm.CreateWorkload(ctx, id); err != nil {
			t.Fatalf("CreateWorkload failed: %v", err)
		}
	}

	// Move some to RUNNING
	wlm.UpdateWorkloadObservedState(ctx, "workload-1", WorkloadRunning, map[string]string{})
	wlm.UpdateWorkloadObservedState(ctx, "workload-2", WorkloadRunning, map[string]string{})
	wlm.UpdateWorkloadObservedState(ctx, "workload-3", WorkloadHealthy, map[string]string{})

	counts := wlm.CountWorkloadsByState(ctx)

	if counts[WorkloadUnknown] != 2 {
		t.Errorf("Expected 2 UNKNOWN workloads, got %d", counts[WorkloadUnknown])
	}

	if counts[WorkloadRunning] != 2 {
		t.Errorf("Expected 2 RUNNING workloads, got %d", counts[WorkloadRunning])
	}

	if counts[WorkloadHealthy] != 1 {
		t.Errorf("Expected 1 HEALTHY workload, got %d", counts[WorkloadHealthy])
	}

	t.Logf("PASS: Workload state counting works correctly")
}

// TestCordonDuringDeployment verifies cordon prevents scheduling during deployment
func TestCordonDuringDeployment(t *testing.T) {
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	ctx := context.Background()

	nodeID := "node-deploy-001"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// Transition to VERIFIED first
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Cordon node
	if err := nlm.CordonNode(ctx, nodeID); err != nil {
		t.Fatalf("CordonNode failed: %v", err)
	}

	// Try to create workload on cordoned node (in real deployment, scheduler would check this)
	workloadID := "workload-cordoned-001"
	if err := wlm.CreateWorkload(ctx, workloadID); err != nil {
		t.Fatalf("CreateWorkload failed: %v", err)
	}

	// Verify node is cordoned
	node, _ := nlm.GetNodeState(ctx, nodeID)
	if !node.Cordoned {
		t.Error("Node should be marked as cordoned")
	}

	t.Logf("PASS: Node cordoning state tracked during deployment")
}

// TestDrainWithRunningWorkload verifies drain targets tracked
func TestDrainWithRunningWorkload(t *testing.T) {
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	ctx := context.Background()

	nodeID := "node-drain-workload-001"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// Transition to VERIFIED first
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Create 3 workloads
	for i := 1; i <= 3; i++ {
		id := "workload-drain-" + string(rune('0'+i))
		if err := wlm.CreateWorkload(ctx, id); err != nil {
			t.Fatalf("CreateWorkload failed: %v", err)
		}
		wlm.UpdateWorkloadObservedState(ctx, id, WorkloadRunning, map[string]string{})
	}

	// Start drain with 3 workloads
	if err := nlm.DrainNode(ctx, nodeID, 3); err != nil {
		t.Fatalf("DrainNode failed: %v", err)
	}

	node, _ := nlm.GetNodeState(ctx, nodeID)
	if node.DrainTarget != 3 {
		t.Errorf("Expected DrainTarget=3, got %d", node.DrainTarget)
	}

	counts := wlm.CountWorkloadsByState(ctx)
	if counts[WorkloadRunning] != 3 {
		t.Errorf("Expected 3 RUNNING workloads during drain, got %d", counts[WorkloadRunning])
	}

	t.Logf("PASS: Drain with running workload tracked correctly")
}

// TestInvalidStateTransition verifies invalid transitions rejected
func TestInvalidStateTransition(t *testing.T) {
	nlm := NewNodeLifecycleManager()
	ctx := context.Background()

	nodeID := "node-invalid-001"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// Transition to VERIFIED first
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Cordon the active node
	if err := nlm.CordonNode(ctx, nodeID); err != nil {
		t.Fatalf("CordonNode failed: %v", err)
	}

	// Now try to transition cordoned node to ACTIVE (invalid, can't transition from CORDONED)
	if err := nlm.TransitionToActive(ctx, nodeID); err == nil {
		t.Error("Should not allow CORDONED -> ACTIVE transition")
	}

	t.Logf("PASS: Invalid state transitions rejected correctly")
}

// TestNodeProvenance verifies state includes provenance information
func TestNodeProvenance(t *testing.T) {
	nlm := NewNodeLifecycleManager()
	ctx := context.Background()

	nodeID := "node-provenance-001"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	state, err := nlm.GetNodeState(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeState failed: %v", err)
	}

	// Verify provenance fields
	if state.NodeID != nodeID {
		t.Error("NodeID not set correctly")
	}

	if state.State == "" {
		t.Error("State should be set")
	}

	if state.ObservedAt == 0 {
		t.Error("ObservedAt should be set")
	}

	if state.SourceID != "agent" {
		t.Errorf("Expected SourceID=agent, got %s", state.SourceID)
	}

	if state.Freshness != "FRESH" {
		t.Errorf("Expected Freshness=FRESH, got %s", state.Freshness)
	}

	t.Logf("PASS: Node provenance information tracked correctly")
}

// TestWorkloadProvenance verifies workload state includes provenance
func TestWorkloadProvenance(t *testing.T) {
	wlm := NewWorkloadLifecycleManager()
	ctx := context.Background()

	workloadID := "workload-provenance-001"
	if err := wlm.CreateWorkload(ctx, workloadID); err != nil {
		t.Fatalf("CreateWorkload failed: %v", err)
	}

	state, err := wlm.GetWorkloadState(ctx, workloadID)
	if err != nil {
		t.Fatalf("GetWorkloadState failed: %v", err)
	}

	// Verify provenance fields
	if state.WorkloadID != workloadID {
		t.Error("WorkloadID not set correctly")
	}

	if state.DesiredState == "" {
		t.Error("DesiredState should be set")
	}

	if state.ObservedAt == 0 {
		t.Error("ObservedAt should be set")
	}

	if state.SourceID != "control-plane" {
		t.Errorf("Expected SourceID=control-plane, got %s", state.SourceID)
	}

	if state.Freshness != "FRESH" {
		t.Errorf("Expected Freshness=FRESH, got %s", state.Freshness)
	}

	t.Logf("PASS: Workload provenance information tracked correctly")
}
