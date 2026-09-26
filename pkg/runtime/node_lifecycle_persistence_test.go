package runtime

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestNodeStatePersistence verifies node state persists across restarts
func TestNodeStatePersistence(t *testing.T) {
	t.Log("Testing node state persistence")

	// Create temporary directories for test
	stateDir := t.TempDir()
	reconDir := t.TempDir()

	ctx := context.Background()

	// Phase 1: Create manager and register node
	t.Log("Phase 1: Register node and persist state")
	nlm1 := NewNodeLifecycleManagerWithStore(stateDir, reconDir)

	if err := nlm1.RegisterNode(ctx, "persistent-node"); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// Verify state file was created
	stateFile := stateDir + "/persistent-node.json"
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Errorf("State file not created: %s", stateFile)
	}

	// Phase 2: Simulate restart - create new manager and recover
	t.Log("Phase 2: Recover state from disk after restart")
	nlm2 := NewNodeLifecycleManagerWithStore(stateDir, reconDir)

	if err := nlm2.RecoverNodeStatesFromDisk(ctx); err != nil {
		t.Fatalf("RecoverNodeStatesFromDisk failed: %v", err)
	}

	// Verify recovered state matches
	state, err := nlm2.GetNodeState(ctx, "persistent-node")
	if err != nil {
		t.Fatalf("GetNodeState failed: %v", err)
	}

	if state.State != NodeDiscovered {
		t.Errorf("Expected DISCOVERED state after recovery, got %s", state.State)
	}

	if state.DesiredState != NodeDiscovered {
		t.Errorf("Expected DISCOVERED desired state after recovery, got %s", state.DesiredState)
	}

	// Phase 3: Transition to verified
	t.Log("Phase 3: Transition to verified")
	nlm2.mu.Lock()
	node, _ := nlm2.nodes["persistent-node"]
	node.State = NodeVerified
	nlm2.mu.Unlock()

	if err := nlm2.TransitionToActive(ctx, "persistent-node"); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Verify new state was persisted
	state, _ = nlm2.GetNodeState(ctx, "persistent-node")
	if state.State != NodeActive {
		t.Errorf("Expected ACTIVE state, got %s", state.State)
	}

	if state.Generation != 2 {
		t.Errorf("Expected generation 2, got %d", state.Generation)
	}

	// Phase 4: Verify state survives another restart
	t.Log("Phase 4: Verify state persists through second restart")
	nlm3 := NewNodeLifecycleManagerWithStore(stateDir, reconDir)

	if err := nlm3.RecoverNodeStatesFromDisk(ctx); err != nil {
		t.Fatalf("RecoverNodeStatesFromDisk failed: %v", err)
	}

	state, _ = nlm3.GetNodeState(ctx, "persistent-node")
	if state.State != NodeActive {
		t.Errorf("Expected ACTIVE state after second recovery, got %s", state.State)
	}

	t.Logf("PASS: Node state persisted across 2 restarts")
}

// TestDesiredVsObservedState verifies reconciliation tracking
func TestDesiredVsObservedState(t *testing.T) {
	t.Log("Testing DESIRED vs OBSERVED state tracking")

	stateDir := t.TempDir()
	reconDir := t.TempDir()
	ctx := context.Background()

	nlm := NewNodeLifecycleManagerWithStore(stateDir, reconDir)

	if err := nlm.RegisterNode(ctx, "reconcile-node"); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// Initially DESIRED == OBSERVED
	state, _ := nlm.GetNodeState(ctx, "reconcile-node")
	if state.DesiredState != state.State {
		t.Errorf("Initial state: DESIRED != OBSERVED")
	}

	// Set desired state to ACTIVE but leave observed at DISCOVERED
	nlm.mu.Lock()
	node := nlm.nodes["reconcile-node"]
	node.DesiredState = NodeActive
	nlm.mu.Unlock()

	// Record reconciliation divergence
	if err := nlm.ReconcileNodeState(ctx, "reconcile-node"); err != nil {
		t.Fatalf("ReconcileNodeState failed: %v", err)
	}

	// Verify reconciliation was recorded
	recon, err := nlm.reconStore.GetReconciliationStatus("reconcile-node")
	if err != nil {
		t.Fatalf("GetReconciliationStatus failed: %v", err)
	}

	if !recon.Divergent {
		t.Error("Expected Divergent=true for mismatched DESIRED/OBSERVED")
	}

	if recon.DesiredState != "ACTIVE" {
		t.Errorf("Expected ACTIVE desired state, got %s", recon.DesiredState)
	}

	if recon.ObservedState != "DISCOVERED" {
		t.Errorf("Expected DISCOVERED observed state, got %s", recon.ObservedState)
	}

	t.Logf("PASS: DESIRED vs OBSERVED reconciliation tracked")
}

// TestGenerationCounter verifies optimistic update counter
func TestGenerationCounter(t *testing.T) {
	t.Log("Testing generation counter for optimistic updates")

	stateDir := t.TempDir()
	reconDir := t.TempDir()
	ctx := context.Background()

	nlm := NewNodeLifecycleManagerWithStore(stateDir, reconDir)

	if err := nlm.RegisterNode(ctx, "versioned-node"); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	state, _ := nlm.GetNodeState(ctx, "versioned-node")
	initialGen := state.Generation

	if initialGen != 1 {
		t.Errorf("Expected initial generation 1, got %d", initialGen)
	}

	// Transition changes generation
	nlm.mu.Lock()
	node := nlm.nodes["versioned-node"]
	node.State = NodeVerified
	nlm.mu.Unlock()

	nlm.TransitionToActive(ctx, "versioned-node")

	state, _ = nlm.GetNodeState(ctx, "versioned-node")
	if state.Generation != initialGen+1 {
		t.Errorf("Expected generation %d after transition, got %d", initialGen+1, state.Generation)
	}

	// Multiple transitions increment generation
	nlm.CordonNode(ctx, "versioned-node")
	state, _ = nlm.GetNodeState(ctx, "versioned-node")
	if state.Generation != initialGen+2 {
		t.Errorf("Expected generation %d after cordoning, got %d", initialGen+2, state.Generation)
	}

	t.Logf("PASS: Generation counter increments correctly")
}

// TestRecoverySequenceOnNodeRestart verifies node restart handling
func TestRecoverySequenceOnNodeRestart(t *testing.T) {
	t.Log("Testing recovery sequence on node restart (OFFLINE → DISCOVERED)")

	stateDir := t.TempDir()
	reconDir := t.TempDir()
	ctx := context.Background()

	// Create and persist an ACTIVE node
	nlm1 := NewNodeLifecycleManagerWithStore(stateDir, reconDir)
	nlm1.RegisterNode(ctx, "restart-node")

	nlm1.mu.Lock()
	node := nlm1.nodes["restart-node"]
	node.State = NodeVerified
	nlm1.mu.Unlock()

	nlm1.TransitionToActive(ctx, "restart-node")

	// Simulate heartbeat timeout marking node OFFLINE
	nlm1.mu.Lock()
	node = nlm1.nodes["restart-node"]
	node.LastHeartbeat = time.Now().Add(-90 * time.Second).UnixNano()
	nlm1.mu.Unlock()

	offlineNodes := nlm1.DetectOfflineNodes(ctx)
	if len(offlineNodes) != 1 {
		t.Fatalf("Expected 1 offline node, got %d", len(offlineNodes))
	}

	state, _ := nlm1.GetNodeState(ctx, "restart-node")
	if state.State != NodeOffline {
		t.Errorf("Expected OFFLINE after timeout, got %s", state.State)
	}

	// Restart: new manager instance, recover, and reset to DISCOVERED
	nlm2 := NewNodeLifecycleManagerWithStore(stateDir, reconDir)
	nlm2.RecoverNodeStatesFromDisk(ctx)

	// Simulate recovery heartbeat resetting to DISCOVERED
	nlm2.mu.Lock()
	node = nlm2.nodes["restart-node"]
	node.State = NodeDiscovered
	node.DesiredState = NodeDiscovered
	node.LastHeartbeat = time.Now().UnixNano()
	node.Freshness = "FRESH"
	node.Generation++
	nlm2.mu.Unlock()

	nlm2.persistNodeState(nlm2.nodes["restart-node"])

	state, _ = nlm2.GetNodeState(ctx, "restart-node")
	if state.State != NodeDiscovered {
		t.Errorf("Expected DISCOVERED after recovery, got %s", state.State)
	}

	t.Logf("PASS: Recovery sequence handled correctly")
}
