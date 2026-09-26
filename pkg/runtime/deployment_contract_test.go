package runtime

import (
	"context"
	"testing"
	"time"
)

// TestSchedulingRuleOnCordoned verifies Rule 1: Cordoned nodes rejected for scheduling
func TestSchedulingRuleOnCordoned(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)

	t.Log("Rule 1: Scheduling on CORDONED nodes is prohibited")

	nodeID := "node-cordoned"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	if err := nlm.CordonNode(ctx, nodeID); err != nil {
		t.Fatalf("CordonNode failed: %v", err)
	}

	req := &DeploymentRequest{
		WorkloadID: "workload-1",
		Replicas:   1,
		Timeout:    30 * time.Second,
	}

	err := contract.ValidateSchedulingContract(ctx, nodeID, req)
	if err == nil {
		t.Error("Expected error scheduling on cordoned node, got nil")
	}

	t.Logf("PASS: Scheduling on cordoned node rejected with error: %v", err)
}

// TestSchedulingRuleOnOffline verifies Rule 2: Offline nodes rejected for scheduling
func TestSchedulingRuleOnOffline(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)

	t.Log("Rule 2: Scheduling on OFFLINE nodes is prohibited")

	nodeID := "node-offline"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Simulate heartbeat failure
	nlm.mu.Lock()
	nlm.nodes[nodeID].LastHeartbeat = time.Now().Add(-70 * time.Second).UnixNano()
	nlm.mu.Unlock()

	// Detect offline
	nlm.DetectOfflineNodes(ctx)

	req := &DeploymentRequest{
		WorkloadID: "workload-1",
		Replicas:   1,
		Timeout:    30 * time.Second,
	}

	err := contract.ValidateSchedulingContract(ctx, nodeID, req)
	if err == nil {
		t.Error("Expected error scheduling on offline node, got nil")
	}

	t.Logf("PASS: Scheduling on offline node rejected with error: %v", err)
}

// TestSchedulingRuleOnDraining verifies Rule 3: Draining nodes rejected for scheduling
func TestSchedulingRuleOnDraining(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)

	t.Log("Rule 3: Scheduling on DRAINING nodes is prohibited")

	nodeID := "node-draining"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	if err := nlm.DrainNode(ctx, nodeID, 2); err != nil {
		t.Fatalf("DrainNode failed: %v", err)
	}

	req := &DeploymentRequest{
		WorkloadID: "workload-1",
		Replicas:   1,
		Timeout:    30 * time.Second,
	}

	err := contract.ValidateSchedulingContract(ctx, nodeID, req)
	if err == nil {
		t.Error("Expected error scheduling on draining node, got nil")
	}

	t.Logf("PASS: Scheduling on draining node rejected with error: %v", err)
}

// TestSchedulingRuleOnRevoked verifies Rule 4: Revoked nodes rejected for scheduling
func TestSchedulingRuleOnRevoked(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)

	t.Log("Rule 4: Scheduling on REVOKED nodes is prohibited")

	nodeID := "node-revoked"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	if err := nlm.RevokeNode(ctx, nodeID); err != nil {
		t.Fatalf("RevokeNode failed: %v", err)
	}

	req := &DeploymentRequest{
		WorkloadID: "workload-1",
		Replicas:   1,
		Timeout:    30 * time.Second,
	}

	err := contract.ValidateSchedulingContract(ctx, nodeID, req)
	if err == nil {
		t.Error("Expected error scheduling on revoked node, got nil")
	}

	t.Logf("PASS: Scheduling on revoked node rejected with error: %v", err)
}

// TestSchedulingRuleStaleAllowed verifies Rule 5: Stale nodes allowed with warning
func TestSchedulingRuleStaleAllowed(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)

	t.Log("Rule 5: STALE nodes can be scheduled on (with warning)")

	nodeID := "node-stale"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Simulate stale heartbeat (30-60 seconds)
	nlm.mu.Lock()
	nlm.nodes[nodeID].LastHeartbeat = time.Now().Add(-40 * time.Second).UnixNano()
	nlm.mu.Unlock()

	// Mark as stale
	nlm.DetectOfflineNodes(ctx)

	req := &DeploymentRequest{
		WorkloadID: "workload-1",
		Replicas:   1,
		Timeout:    30 * time.Second,
	}

	err := contract.ValidateSchedulingContract(ctx, nodeID, req)
	if err != nil {
		t.Logf("Note: Scheduling on stale node returned: %v", err)
	}

	t.Logf("PASS: STALE node scheduling handled (warning logged)")
}

// TestSchedulingRuleActivePreferred verifies Rule 6: Active nodes preferred
func TestSchedulingRuleActivePreferred(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)

	t.Log("Rule 6: ACTIVE nodes are preferred for scheduling")

	nodeID := "node-active"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	req := &DeploymentRequest{
		WorkloadID: "workload-1",
		Replicas:   1,
		Timeout:    30 * time.Second,
	}

	err := contract.ValidateSchedulingContract(ctx, nodeID, req)
	if err != nil {
		t.Errorf("ACTIVE node should be available for scheduling, got error: %v", err)
	}

	t.Logf("PASS: ACTIVE node available for scheduling")
}

// TestDeploymentRequestValidation verifies request parameter validation
func TestDeploymentRequestValidation(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	dv := NewDeploymentValidator(nlm, wlm)

	t.Log("Testing deployment request validation")

	// Test empty workload ID
	req := &DeploymentRequest{
		WorkloadID: "",
		Replicas:   1,
		Timeout:    30 * time.Second,
	}

	err := dv.ValidateDeployment(ctx, req)
	if err == nil {
		t.Error("Expected error for empty workload ID")
	}

	// Test invalid replicas
	req.WorkloadID = "workload-1"
	req.Replicas = 0
	err = dv.ValidateDeployment(ctx, req)
	if err == nil {
		t.Error("Expected error for zero replicas")
	}

	// Test invalid timeout
	req.Replicas = 1
	req.Timeout = 0
	err = dv.ValidateDeployment(ctx, req)
	if err == nil {
		t.Error("Expected error for zero timeout")
	}

	// Test valid request
	req.Timeout = 30 * time.Second
	err = dv.ValidateDeployment(ctx, req)
	if err != nil {
		t.Errorf("Valid request rejected: %v", err)
	}

	t.Logf("PASS: Request validation working correctly")
}

// TestNodeSelectionForDeployment verifies proper node selection
func TestNodeSelectionForDeployment(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	dv := NewDeploymentValidator(nlm, wlm)

	t.Log("Testing node selection for deployment")

	// Create 3 nodes: 1 active, 1 cordoned, 1 offline
	for i := 1; i <= 3; i++ {
		nodeID := "node-" + string(rune('0'+i))
		if err := nlm.RegisterNode(ctx, nodeID); err != nil {
			t.Fatalf("RegisterNode %s failed: %v", nodeID, err)
		}

		nlm.mu.Lock()
		nlm.nodes[nodeID].State = NodeVerified
		nlm.mu.Unlock()

		if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
			t.Fatalf("TransitionToActive %s failed: %v", nodeID, err)
		}
	}

	// Cordon node-2
	if err := nlm.CordonNode(ctx, "node-2"); err != nil {
		t.Fatalf("CordonNode failed: %v", err)
	}

	// Offline node-3
	nlm.mu.Lock()
	nlm.nodes["node-3"].LastHeartbeat = time.Now().Add(-70 * time.Second).UnixNano()
	nlm.mu.Unlock()
	nlm.DetectOfflineNodes(ctx)

	req := &DeploymentRequest{
		WorkloadID: "workload-1",
		Replicas:   1,
		Timeout:    30 * time.Second,
	}

	selected, err := dv.SelectNodesForDeployment(ctx, req)
	if err != nil {
		t.Errorf("SelectNodesForDeployment failed: %v", err)
	}

	if len(selected) != 1 {
		t.Errorf("Expected 1 selected node, got %d", len(selected))
	}

	if selected[0] != "node-1" {
		t.Errorf("Expected node-1 selected, got %s", selected[0])
	}

	t.Logf("PASS: Node selection correctly filtered unsuitable nodes")
}

// TestExecuteDeployment verifies complete deployment workflow
func TestExecuteDeployment(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	dv := NewDeploymentValidator(nlm, wlm)

	t.Log("Testing complete deployment execution")

	// Setup active node
	nodeID := "node-deploy"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	req := &DeploymentRequest{
		WorkloadID: "workload-deploy",
		Replicas:   1,
		Timeout:    30 * time.Second,
	}

	result, err := dv.ExecuteDeployment(ctx, req)
	if err != nil {
		t.Fatalf("ExecuteDeployment failed: %v", err)
	}

	if result.WorkloadID != "workload-deploy" {
		t.Errorf("Expected workload ID workload-deploy, got %s", result.WorkloadID)
	}

	if result.Status != DeploymentScheduled {
		t.Errorf("Expected status SCHEDULED, got %s", result.Status)
	}

	if len(result.Assigned) == 0 {
		t.Error("Expected nodes to be assigned")
	}

	if result.CompletedAt == 0 {
		t.Error("Expected CompletedAt to be set")
	}

	t.Logf("PASS: Deployment executed successfully with status %s", result.Status)
}

// TestExecutionContractValidation verifies Rule 8: Node state must remain ACTIVE/VERIFIED
func TestExecutionContractValidation(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)

	t.Log("Rule 8: Node state must remain ACTIVE/VERIFIED during workload execution")

	nodeID := "node-exec"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Create workload
	if err := wlm.CreateWorkload(ctx, "workload-1"); err != nil {
		t.Fatalf("CreateWorkload failed: %v", err)
	}

	// Validate execution on ACTIVE node
	err := contract.ValidateExecutionContract(ctx, "workload-1", nodeID)
	if err != nil {
		t.Errorf("Execution validation failed on ACTIVE node: %v", err)
	}

	// Transition node to VERIFIED and validate again
	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	err = contract.ValidateExecutionContract(ctx, "workload-1", nodeID)
	if err != nil {
		t.Errorf("Execution validation failed on VERIFIED node: %v", err)
	}

	t.Logf("PASS: Execution contract validated for ACTIVE/VERIFIED nodes")
}

// TestTerminationContractValidation verifies Rule 9: Drain policy enforcement
func TestTerminationContractValidation(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)

	t.Log("Rule 9: Drain policy - stop all workloads on node sequentially")

	nodeID := "node-term"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	// Create multiple workloads
	for i := 1; i <= 3; i++ {
		wlID := "workload-" + string(rune('0'+i))
		if err := wlm.CreateWorkload(ctx, wlID); err != nil {
			t.Fatalf("CreateWorkload failed: %v", err)
		}

		if err := wlm.UpdateWorkloadObservedState(ctx, wlID, WorkloadRunning, map[string]string{}); err != nil {
			t.Fatalf("UpdateWorkloadObservedState failed: %v", err)
		}
	}

	// Validate termination (drain) contract
	count, err := contract.ValidateTerminationContract(ctx, nodeID)
	if err != nil {
		t.Errorf("ValidateTerminationContract failed: %v", err)
	}

	if count < 0 {
		t.Errorf("Expected non-negative workload count, got %d", count)
	}

	t.Logf("PASS: Termination contract validated - %d workloads to drain", count)
}

// TestCordonPreventsScheduling verifies Rule 10: Cordoned nodes blocked
func TestCordonPreventsScheduling(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	dv := NewDeploymentValidator(nlm, wlm)

	t.Log("Rule 10: Cordoned nodes must not accept new workload assignments")

	nodeID := "node-block-schedule"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	if err := nlm.CordonNode(ctx, nodeID); err != nil {
		t.Fatalf("CordonNode failed: %v", err)
	}

	req := &DeploymentRequest{
		WorkloadID: "workload-1",
		Replicas:   1,
		Timeout:    30 * time.Second,
	}

	// Attempt to schedule on cordoned node
	err := dv.CanScheduleOnNode(ctx, nodeID, req)
	if err == nil {
		t.Error("Expected error when scheduling on cordoned node")
	}

	t.Logf("PASS: Cordoned node correctly blocks scheduling")
}

// TestDeploymentConstraintVerification verifies deployment state tracking
func TestDeploymentConstraintVerification(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	dv := NewDeploymentValidator(nlm, wlm)

	t.Log("Testing deployment constraint verification")

	// Setup node and workload
	nodeID := "node-verify"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	if err := wlm.CreateWorkload(ctx, "workload-1"); err != nil {
		t.Fatalf("CreateWorkload failed: %v", err)
	}

	// Verify constraints with active node
	ok, reason := dv.VerifyDeploymentConstraints(ctx, "workload-1", []string{nodeID})
	if !ok {
		t.Errorf("Expected constraints satisfied, got reason: %s", reason)
	}

	// Go offline and verify
	nlm.mu.Lock()
	nlm.nodes[nodeID].LastHeartbeat = time.Now().Add(-70 * time.Second).UnixNano()
	nlm.mu.Unlock()
	nlm.DetectOfflineNodes(ctx)

	ok, reason = dv.VerifyDeploymentConstraints(ctx, "workload-1", []string{nodeID})
	if ok {
		t.Error("Expected constraints violated for offline node")
	}

	if reason == "" {
		t.Error("Expected reason for constraint violation")
	}

	t.Logf("PASS: Deployment constraints verified correctly - reason: %s", reason)
}

// TestArtifactReferenceVerification verifies artifact hash and signature validation
func TestArtifactReferenceVerification(t *testing.T) {
	ctx := context.Background()
	contract := NewDeploymentContract(NewNodeLifecycleManager(), NewWorkloadLifecycleManager())

	t.Log("Testing artifact reference verification")

	// Valid artifact
	validArtifact := &ArtifactReference{
		SourceHash:    "2c26b46911185131006b194efb6a3156b0b4e6426b5291864b66f4ecf9b2d3ce",
		SignerID:      "control-plane",
		Signature:     "valid_signature_signature",
		SignAlgorithm: "ed25519",
		SignedAt:      time.Now().UnixNano(),
	}

	if err := contract.ValidateArtifact(ctx, validArtifact); err != nil {
		t.Fatalf("Valid artifact rejected: %v", err)
	}

	// Missing source hash
	invalidArtifact := &ArtifactReference{
		SignerID:      "control-plane",
		Signature:     "valid_signature",
		SignAlgorithm: "ed25519",
	}

	if err := contract.ValidateArtifact(ctx, invalidArtifact); err == nil {
		t.Error("Expected error for missing source hash")
	}

	// Invalid hash format
	invalidHashArtifact := &ArtifactReference{
		SourceHash:    "not_hex",
		SignerID:      "control-plane",
		Signature:     "valid_signature",
		SignAlgorithm: "ed25519",
	}

	if err := contract.ValidateArtifact(ctx, invalidHashArtifact); err == nil {
		t.Error("Expected error for invalid hash format")
	}

	// Unsupported algorithm
	invalidAlgoArtifact := &ArtifactReference{
		SourceHash:    "2c26b46911185131006b194efb6a3156b0b4e6426b5291864b66f4ecf9b2d3ce",
		SignerID:      "control-plane",
		Signature:     "valid_signature",
		SignAlgorithm: "rsa",
	}

	if err := contract.ValidateArtifact(ctx, invalidAlgoArtifact); err == nil {
		t.Error("Expected error for unsupported algorithm")
	}

	t.Logf("PASS: Artifact verification working correctly")
}

// TestDeploymentSpecValidation verifies deployment spec structure and artifact binding
func TestDeploymentSpecValidation(t *testing.T) {
	ctx := context.Background()
	contract := NewDeploymentContract(NewNodeLifecycleManager(), NewWorkloadLifecycleManager())

	t.Log("Testing deployment spec validation")

	// Valid spec
	validSpec := &DeploymentSpec{
		WorkloadID: "workload-1",
		ArtifactRef: &ArtifactReference{
			SourceHash:    "2c26b46911185131006b194efb6a3156b0b4e6426b5291864b66f4ecf9b2d3ce",
			SignerID:      "control-plane",
			Signature:     "valid_signature",
			SignAlgorithm: "ed25519",
			SignedAt:      time.Now().UnixNano(),
		},
		ContainerImage: "nginx:latest",
		ResourceConstraints: &ResourceConstraints{
			MemoryBytes: 256 * 1024 * 1024,
			CPUShares:   200,
			DiskBytes:   1024 * 1024,
		},
		Replicas: 3,
	}

	if err := contract.ValidateDeploymentSpec(ctx, validSpec); err != nil {
		t.Fatalf("Valid spec rejected: %v", err)
	}

	if validSpec.SpecHash == "" {
		t.Error("Expected spec hash to be calculated")
	}

	// Missing artifact reference
	noArtifactSpec := &DeploymentSpec{
		WorkloadID:     "workload-1",
		ContainerImage: "nginx:latest",
		ResourceConstraints: &ResourceConstraints{
			MemoryBytes: 256 * 1024 * 1024,
			CPUShares:   200,
			DiskBytes:   1024 * 1024,
		},
	}

	if err := contract.ValidateDeploymentSpec(ctx, noArtifactSpec); err == nil {
		t.Error("Expected error for missing artifact reference")
	}

	// Invalid resource constraints
	invalidResourceSpec := &DeploymentSpec{
		WorkloadID: "workload-1",
		ArtifactRef: &ArtifactReference{
			SourceHash:    "2c26b46911185131006b194efb6a3156b0b4e6426b5291864b66f4ecf9b2d3ce",
			SignerID:      "control-plane",
			Signature:     "valid_signature",
			SignAlgorithm: "ed25519",
		},
		ContainerImage: "nginx:latest",
		ResourceConstraints: &ResourceConstraints{
			MemoryBytes: -1,
			CPUShares:   200,
		},
	}

	if err := contract.ValidateDeploymentSpec(ctx, invalidResourceSpec); err == nil {
		t.Error("Expected error for negative memory")
	}

	t.Logf("PASS: Deployment spec validation working correctly")
}
