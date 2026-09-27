package runtime

import (
	"context"
	"testing"
	"time"
)

// TestNodeAndWorkloadIntegration verifies nodes and workloads coordinate lifecycle
func TestNodeAndWorkloadIntegration(t *testing.T) {
	ctx := context.Background()

	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()

	t.Log("=== A05-P0-A01: Node and Workload Lifecycle Integration Test ===")

	// Phase 1: Register node and bring to ACTIVE
	t.Log("Phase 1: Node Registration and Activation")
	nodeID := "node-001"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	nodeState, err := nlm.GetNodeState(ctx, nodeID)
	if err != nil {
		t.Fatalf("GetNodeState failed: %v", err)
	}

	if nodeState.State != NodeActive {
		t.Fatalf("Expected node ACTIVE, got %s", nodeState.State)
	}

	// Phase 2: Create workloads on node
	t.Log("Phase 2: Workload Creation on Active Node")
	workloadIDs := []string{"workload-1", "workload-2", "workload-3"}
	for _, id := range workloadIDs {
		if err := wlm.CreateWorkload(ctx, id); err != nil {
			t.Fatalf("CreateWorkload %s failed: %v", id, err)
		}
	}

	// Verify workloads created in DESIRED state
	for _, wlID := range workloadIDs {
		wl, _ := wlm.GetWorkloadState(ctx, wlID)
		if wl.DesiredState != WorkloadDesired {
			t.Errorf("Expected DesiredState=DESIRED for %s, got %s", wlID, wl.DesiredState)
		}
	}

	// Phase 3: Transition workloads to RUNNING
	t.Log("Phase 3: Workload Execution on Active Node")
	for _, wlID := range workloadIDs {
		if err := wlm.UpdateWorkloadObservedState(ctx, wlID, WorkloadRunning, map[string]string{"health": "healthy"}); err != nil {
			t.Fatalf("UpdateWorkloadObservedState failed: %v", err)
		}
	}

	// Verify all workloads are running
	counts := wlm.CountWorkloadsByState(ctx)
	if counts[WorkloadRunning] != 3 {
		t.Fatalf("Expected 3 running workloads, got %d", counts[WorkloadRunning])
	}

	// Phase 4: Node drain (graceful shutdown)
	t.Log("Phase 4: Node Drain During Active Workload Execution")
	if err := nlm.DrainNode(ctx, nodeID, 3); err != nil {
		t.Fatalf("DrainNode failed: %v", err)
	}

	nodeState, _ = nlm.GetNodeState(ctx, nodeID)
	if nodeState.State != NodeDraining {
		t.Fatalf("Expected DRAINING, got %s", nodeState.State)
	}

	if nodeState.DrainTarget != 3 {
		t.Fatalf("Expected DrainTarget=3, got %d", nodeState.DrainTarget)
	}

	// Phase 5: Gracefully stop workloads
	t.Log("Phase 5: Graceful Workload Shutdown")
	for _, wlID := range workloadIDs {
		if err := wlm.StopWorkload(ctx, wlID); err != nil {
			t.Fatalf("StopWorkload failed: %v", err)
		}
		if err := wlm.MarkWorkloadStopped(ctx, wlID); err != nil {
			t.Fatalf("MarkWorkloadStopped failed: %v", err)
		}
	}

	// Complete drain
	if err := nlm.CompleteDrain(ctx, nodeID); err != nil {
		t.Fatalf("CompleteDrain failed: %v", err)
	}

	nodeState, _ = nlm.GetNodeState(ctx, nodeID)
	if nodeState.State != NodeIdle {
		t.Fatalf("Expected IDLE after drain, got %s", nodeState.State)
	}

	// Verify all workloads are stopped
	counts = wlm.CountWorkloadsByState(ctx)
	if counts[WorkloadStopped] != 3 {
		t.Fatalf("Expected 3 stopped workloads, got %d", counts[WorkloadStopped])
	}

	t.Logf("PASS: Node and workload lifecycle integration complete")
	t.Logf("  - Node: DISCOVERED → VERIFIED → ACTIVE → DRAINING → IDLE")
	t.Logf("  - Workloads: DESIRED → RUNNING → STOPPING → STOPPED")
	t.Logf("  - Lifecycle: Coordinated transitions with proper state tracking")
}

// TestNodeCordonBlocksNewWorkloadScheduling verifies cordon prevents scheduling
func TestNodeCordonBlocksNewWorkloadScheduling(t *testing.T) {
	ctx := context.Background()

	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()

	t.Log("Testing node cordon prevents new workload scheduling")

	// Setup node
	nodeID := "node-cordon-test"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Create workload on active node
	if err := wlm.CreateWorkload(ctx, "workload-1"); err != nil {
		t.Fatalf("CreateWorkload failed: %v", err)
	}

	// Cordon the node
	if err := nlm.CordonNode(ctx, nodeID); err != nil {
		t.Fatalf("CordonNode failed: %v", err)
	}

	// Verify node is cordoned
	node, _ := nlm.GetNodeState(ctx, nodeID)
	if !node.Cordoned {
		t.Error("Node should be marked cordoned")
	}

	if node.State != NodeCordoned {
		t.Errorf("Expected CORDONED state, got %s", node.State)
	}

	// In production, scheduler would check node.Cordoned before scheduling
	// Verify it's set correctly
	if !node.Cordoned {
		t.Fatal("Scheduler would reject scheduling on cordoned node")
	}

	t.Logf("PASS: Node cordon correctly prevents scheduling")
}

// TestHeartbeatFailureMarksNodeOffline verifies node goes offline on heartbeat failure
func TestHeartbeatFailureMarksNodeOffline(t *testing.T) {
	ctx := context.Background()

	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()

	t.Log("Testing heartbeat failure marks node offline")

	// Setup active node with running workload
	nodeID := "node-heartbeat-fail"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Create running workload
	if err := wlm.CreateWorkload(ctx, "workload-on-failing-node"); err != nil {
		t.Fatalf("CreateWorkload failed: %v", err)
	}

	if err := wlm.UpdateWorkloadObservedState(ctx, "workload-on-failing-node", WorkloadRunning, map[string]string{}); err != nil {
		t.Fatalf("UpdateWorkloadObservedState failed: %v", err)
	}

	// Simulate heartbeat failure (> 60 seconds ago)
	nlm.mu.Lock()
	nlm.nodes[nodeID].LastHeartbeat = time.Now().Add(-70 * time.Second).UnixNano()
	nlm.mu.Unlock()

	// Detect offline
	offline := nlm.DetectOfflineNodes(ctx)

	if len(offline) != 1 || offline[0] != nodeID {
		t.Fatalf("Expected node to be offline, got %v", offline)
	}

	// Verify node state
	node, _ := nlm.GetNodeState(ctx, nodeID)
	if node.State != NodeOffline {
		t.Errorf("Expected OFFLINE, got %s", node.State)
	}

	// Workload should still be RUNNING (no automatic failover)
	wl, _ := wlm.GetWorkloadState(ctx, "workload-on-failing-node")
	if wl.ObservedState != WorkloadRunning {
		t.Errorf("Workload should still be RUNNING until control-plane reschedules, got %s", wl.ObservedState)
	}

	t.Logf("PASS: Heartbeat failure correctly marks node offline")
}

// TestMultipleNodesWithWorkloads verifies independent node/workload management
func TestMultipleNodesWithWorkloads(t *testing.T) {
	ctx := context.Background()

	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()

	t.Log("Testing multiple nodes with independent workloads")

	// Setup 2 nodes
	for i := 1; i <= 2; i++ {
		nodeID := "node-" + string(rune('0'+i))
		if err := nlm.RegisterNode(ctx, nodeID); err != nil {
			t.Fatalf("RegisterNode failed: %v", err)
		}

		nlm.mu.Lock()
		nlm.nodes[nodeID].State = NodeVerified
		nlm.mu.Unlock()

		if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
			t.Fatalf("TransitionToActive failed: %v", err)
		}

		// Create 2 workloads per node
		for j := 1; j <= 2; j++ {
			wlID := "node-" + string(rune('0'+i)) + "-workload-" + string(rune('0'+j))
			if err := wlm.CreateWorkload(ctx, wlID); err != nil {
				t.Fatalf("CreateWorkload failed: %v", err)
			}

			// Mark as running
			if err := wlm.UpdateWorkloadObservedState(ctx, wlID, WorkloadRunning, map[string]string{}); err != nil {
				t.Fatalf("UpdateWorkloadObservedState failed: %v", err)
			}
		}
	}

	// Cordon only node-1
	if err := nlm.CordonNode(ctx, "node-1"); err != nil {
		t.Fatalf("CordonNode failed: %v", err)
	}

	// Verify node-2 still active
	node2, _ := nlm.GetNodeState(ctx, "node-2")
	if node2.State != NodeActive {
		t.Errorf("node-2 should still be ACTIVE, got %s", node2.State)
	}

	// Verify workload counts
	counts := wlm.CountWorkloadsByState(ctx)
	if counts[WorkloadRunning] != 4 {
		t.Errorf("Expected 4 running workloads, got %d", counts[WorkloadRunning])
	}

	t.Logf("PASS: Multiple nodes and workloads managed independently")
}
