package runtime

import (
	"context"
	"testing"
	"time"
)

// TestFullPipelineExecution verifies complete 16-stage pipeline
func TestFullPipelineExecution(t *testing.T) {
	t.Log("Testing full 16-stage deployment pipeline execution")

	stateDir := t.TempDir()
	reconDir := t.TempDir()
	ctx := context.Background()

	// Setup
	nlm := NewNodeLifecycleManagerWithStore(stateDir, reconDir)
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)
	pipeline := NewDeploymentPipeline(contract)

	// Create workload spec with artifact reference
	spec := &DeploymentSpec{
		WorkloadID:     "test-workload",
		ContainerImage: "myregistry.io/myapp:v1.0",
		ResourceConstraints: &ResourceConstraints{
			MemoryBytes: 512 * 1024 * 1024,
			CPUShares:   500,
			DiskBytes:   1024 * 1024 * 1024,
		},
		Environment: map[string]string{
			"ENV": "test",
		},
		Replicas: 2,
	}

	// Create artifact reference with hash and signature
	spec.ArtifactRef = &ArtifactReference{
		SourceHash:    "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234",
		SignerID:      "control-plane-1",
		Signature:     "sig1234567890abcdef1234567890abcdef",
		SignAlgorithm: "ed25519",
		SignedAt:      time.Now().UnixNano(),
	}

	if err := contract.ValidateDeploymentSpec(ctx, spec); err != nil {
		t.Fatalf("spec validation failed: %v", err)
	}

	// Phase 1: Create pipeline execution
	t.Log("Phase 1: Create pipeline execution")
	pe, err := pipeline.CreatePipelineExecution(ctx, "test-workload", spec)
	if err != nil {
		t.Fatalf("CreatePipelineExecution failed: %v", err)
	}

	if pe.WorkloadID != "test-workload" {
		t.Errorf("Expected workload ID test-workload, got %s", pe.WorkloadID)
	}

	if pe.Status != "PENDING" {
		t.Errorf("Expected PENDING status, got %s", pe.Status)
	}

	// Phase 2: Execute all 16 stages
	t.Log("Phase 2: Execute all 16 stages")
	stages := []PipelineStage{
		StageSource, StageResolve, StageInstall, StageBuild, StageTest, StagePackage,
		StageHash, StageSign, StageArtifactReady, StageMatch, StagePlace,
		StageStart, StageHealth, StageRoute, StageObserve, StageEvidence,
	}

	for _, stage := range stages {
		if err := pe.StartStage(ctx, stage); err != nil {
			t.Fatalf("StartStage(%s) failed: %v", stage, err)
		}

		if stage == StageHash {
			// Record artifact hash
			if err := pe.RecordArtifactHash(ctx, "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"); err != nil {
				t.Fatalf("RecordArtifactHash failed: %v", err)
			}
		}

		if stage == StageSign {
			// Record signature
			if err := pe.RecordArtifactSignature(ctx, "sig1234567890abcdef1234567890abcdef"); err != nil {
				t.Fatalf("RecordArtifactSignature failed: %v", err)
			}
		}

		output := "output-" + string(stage)
		if err := pe.CompleteStage(ctx, stage, output); err != nil {
			t.Fatalf("CompleteStage(%s) failed: %v", stage, err)
		}

		// Record node execution
		nodeID := "node-1"
		if stage == StagePlace {
			nodeID = "node-1"
		}
		if err := pe.RecordNodeExecution(ctx, stage, nodeID); err != nil {
			t.Fatalf("RecordNodeExecution failed: %v", err)
		}
	}

	// Phase 3: Verify all stages completed
	t.Log("Phase 3: Verify all stages completed")
	for _, stage := range stages {
		progress, err := pe.GetStageProgress(ctx, stage)
		if err != nil {
			t.Fatalf("GetStageProgress(%s) failed: %v", stage, err)
		}

		if progress.Status != "SUCCESS" {
			t.Errorf("Stage %s not SUCCESS, got %s", stage, progress.Status)
		}

		if progress.Duration == 0 {
			t.Errorf("Stage %s duration is 0", stage)
		}
	}

	// Phase 4: Finalize pipeline
	t.Log("Phase 4: Finalize pipeline")
	if err := pe.FinalizePipeline(ctx); err != nil {
		t.Fatalf("FinalizePipeline failed: %v", err)
	}

	if pe.Status != "SUCCESS" {
		t.Errorf("Expected SUCCESS status, got %s", pe.Status)
	}

	if pe.Duration == 0 {
		t.Errorf("Expected non-zero pipeline duration")
	}

	// Phase 5: Retrieve from pipeline manager
	t.Log("Phase 5: Retrieve from pipeline manager")
	retrieved, err := pipeline.GetPipelineExecution(ctx, "test-workload")
	if err != nil {
		t.Fatalf("GetPipelineExecution failed: %v", err)
	}

	if retrieved.Status != "SUCCESS" {
		t.Errorf("Retrieved pipeline status should be SUCCESS, got %s", retrieved.Status)
	}

	t.Logf("PASS: Full 16-stage pipeline completed successfully in %d ns", pe.Duration)
}

// TestPipelineStageFailure verifies failure handling
func TestPipelineStageFailure(t *testing.T) {
	t.Log("Testing pipeline stage failure handling")

	stateDir := t.TempDir()
	reconDir := t.TempDir()
	ctx := context.Background()

	nlm := NewNodeLifecycleManagerWithStore(stateDir, reconDir)
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)
	pipeline := NewDeploymentPipeline(contract)

	spec := &DeploymentSpec{
		WorkloadID:     "fail-test",
		ContainerImage: "myregistry.io/app:v1",
		ResourceConstraints: &ResourceConstraints{
			MemoryBytes: 256 * 1024 * 1024,
			CPUShares:   250,
			DiskBytes:   512 * 1024 * 1024,
		},
		Replicas: 1,
		ArtifactRef: &ArtifactReference{
			SourceHash:    "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			SignerID:      "control-plane",
			Signature:     "sig123",
			SignAlgorithm: "ed25519",
			SignedAt:      time.Now().UnixNano(),
		},
	}

	if err := contract.ValidateDeploymentSpec(ctx, spec); err != nil {
		t.Fatalf("spec validation failed: %v", err)
	}

	pe, err := pipeline.CreatePipelineExecution(ctx, "fail-test", spec)
	if err != nil {
		t.Fatalf("CreatePipelineExecution failed: %v", err)
	}

	// Start stages and fail at TEST
	if err := pe.StartStage(ctx, StageSource); err != nil {
		t.Fatalf("StartStage(SOURCE) failed: %v", err)
	}
	if err := pe.CompleteStage(ctx, StageSource, "output"); err != nil {
		t.Fatalf("CompleteStage(SOURCE) failed: %v", err)
	}

	if err := pe.StartStage(ctx, StageBuild); err != nil {
		t.Fatalf("StartStage(BUILD) failed: %v", err)
	}

	if err := pe.StartStage(ctx, StageTest); err != nil {
		t.Fatalf("StartStage(TEST) failed: %v", err)
	}

	// Fail at TEST stage
	if err := pe.FailStage(ctx, StageTest, "test_failure: unit tests failed"); err != nil {
		t.Fatalf("FailStage failed: %v", err)
	}

	if pe.Status != "FAILED" {
		t.Errorf("Expected FAILED status, got %s", pe.Status)
	}

	progress, err := pe.GetStageProgress(ctx, StageTest)
	if err != nil {
		t.Fatalf("GetStageProgress failed: %v", err)
	}

	if progress.Status != "FAILED" {
		t.Errorf("Expected FAILED stage status, got %s", progress.Status)
	}

	if progress.ErrorMsg != "test_failure: unit tests failed" {
		t.Errorf("Expected error message, got %s", progress.ErrorMsg)
	}

	t.Logf("PASS: Pipeline failure handled correctly")
}

// TestPipelineArtifactVerification verifies artifact hash/signature recording
func TestPipelineArtifactVerification(t *testing.T) {
	t.Log("Testing pipeline artifact verification")

	stateDir := t.TempDir()
	reconDir := t.TempDir()
	ctx := context.Background()

	nlm := NewNodeLifecycleManagerWithStore(stateDir, reconDir)
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)
	pipeline := NewDeploymentPipeline(contract)

	spec := &DeploymentSpec{
		WorkloadID:     "verify-test",
		ContainerImage: "registry/app:v1",
		ResourceConstraints: &ResourceConstraints{
			MemoryBytes: 128 * 1024 * 1024,
			CPUShares:   100,
			DiskBytes:   256 * 1024 * 1024,
		},
		Replicas: 1,
		ArtifactRef: &ArtifactReference{
			SourceHash:    "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
			SignerID:      "node-agent-1",
			Signature:     "signode123",
			SignAlgorithm: "ed25519",
			SignedAt:      time.Now().UnixNano(),
		},
	}

	if err := contract.ValidateDeploymentSpec(ctx, spec); err != nil {
		t.Fatalf("spec validation failed: %v", err)
	}

	pe, err := pipeline.CreatePipelineExecution(ctx, "verify-test", spec)
	if err != nil {
		t.Fatalf("CreatePipelineExecution failed: %v", err)
	}

	// Record hash at HASH stage
	if err := pe.StartStage(ctx, StageHash); err != nil {
		t.Fatalf("StartStage(HASH) failed: %v", err)
	}

	expectedHash := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	if err := pe.RecordArtifactHash(ctx, expectedHash); err != nil {
		t.Fatalf("RecordArtifactHash failed: %v", err)
	}

	if pe.SourceHash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, pe.SourceHash)
	}

	if err := pe.CompleteStage(ctx, StageHash, expectedHash); err != nil {
		t.Fatalf("CompleteStage(HASH) failed: %v", err)
	}

	// Record signature at SIGN stage
	if err := pe.StartStage(ctx, StageSign); err != nil {
		t.Fatalf("StartStage(SIGN) failed: %v", err)
	}

	expectedSig := "signode123sig123"
	if err := pe.RecordArtifactSignature(ctx, expectedSig); err != nil {
		t.Fatalf("RecordArtifactSignature failed: %v", err)
	}

	if pe.SourceSignature != expectedSig {
		t.Errorf("Expected signature %s, got %s", expectedSig, pe.SourceSignature)
	}

	if err := pe.CompleteStage(ctx, StageSign, expectedSig); err != nil {
		t.Fatalf("CompleteStage(SIGN) failed: %v", err)
	}

	t.Logf("PASS: Artifact hash and signature recorded correctly")
}

// TestPipelineNodeExecution verifies node tracking across stages
func TestPipelineNodeExecution(t *testing.T) {
	t.Log("Testing pipeline node execution tracking")

	stateDir := t.TempDir()
	reconDir := t.TempDir()
	ctx := context.Background()

	nlm := NewNodeLifecycleManagerWithStore(stateDir, reconDir)
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)
	pipeline := NewDeploymentPipeline(contract)

	spec := &DeploymentSpec{
		WorkloadID:     "node-track",
		ContainerImage: "registry/app:latest",
		ResourceConstraints: &ResourceConstraints{
			MemoryBytes: 256 * 1024 * 1024,
			CPUShares:   250,
			DiskBytes:   512 * 1024 * 1024,
		},
		Replicas: 3,
		ArtifactRef: &ArtifactReference{
			SourceHash:    "1111111111111111111111111111111111111111111111111111111111111111",
			SignerID:      "signer",
			Signature:     "sig",
			SignAlgorithm: "ed25519",
			SignedAt:      time.Now().UnixNano(),
		},
	}

	if err := contract.ValidateDeploymentSpec(ctx, spec); err != nil {
		t.Fatalf("spec validation failed: %v", err)
	}

	pe, err := pipeline.CreatePipelineExecution(ctx, "node-track", spec)
	if err != nil {
		t.Fatalf("CreatePipelineExecution failed: %v", err)
	}

	// Track nodes across stages
	nodesByStage := map[PipelineStage]string{
		StagePlace:  "node-1",
		StageStart:  "node-2",
		StageHealth: "node-3",
	}

	for stage, nodeID := range nodesByStage {
		if err := pe.StartStage(ctx, stage); err != nil {
			t.Fatalf("StartStage(%s) failed: %v", stage, err)
		}

		if err := pe.RecordNodeExecution(ctx, stage, nodeID); err != nil {
			t.Fatalf("RecordNodeExecution failed: %v", err)
		}

		if err := pe.CompleteStage(ctx, stage, ""); err != nil {
			t.Fatalf("CompleteStage(%s) failed: %v", stage, err)
		}
	}

	// Verify nodes were tracked
	if len(pe.NodesExecuted) != 3 {
		t.Errorf("Expected 3 unique nodes, got %d", len(pe.NodesExecuted))
	}

	expectedNodes := map[string]bool{"node-1": true, "node-2": true, "node-3": true}
	for _, n := range pe.NodesExecuted {
		if !expectedNodes[n] {
			t.Errorf("Unexpected node %s in execution", n)
		}
	}

	t.Logf("PASS: Node execution tracked across %d stages", len(pe.NodesExecuted))
}

// TestPipelineStatus verifies status reporting
func TestPipelineStatus(t *testing.T) {
	t.Log("Testing pipeline status reporting")

	stateDir := t.TempDir()
	reconDir := t.TempDir()
	ctx := context.Background()

	nlm := NewNodeLifecycleManagerWithStore(stateDir, reconDir)
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)
	pipeline := NewDeploymentPipeline(contract)

	spec := &DeploymentSpec{
		WorkloadID:     "status-test",
		ContainerImage: "registry/app:v1",
		ResourceConstraints: &ResourceConstraints{
			MemoryBytes: 128 * 1024 * 1024,
			CPUShares:   100,
			DiskBytes:   256 * 1024 * 1024,
		},
		Replicas: 1,
		ArtifactRef: &ArtifactReference{
			SourceHash:    "2222222222222222222222222222222222222222222222222222222222222222",
			SignerID:      "signer",
			Signature:     "sig",
			SignAlgorithm: "ed25519",
			SignedAt:      time.Now().UnixNano(),
		},
	}

	if err := contract.ValidateDeploymentSpec(ctx, spec); err != nil {
		t.Fatalf("spec validation failed: %v", err)
	}

	pe, err := pipeline.CreatePipelineExecution(ctx, "status-test", spec)
	if err != nil {
		t.Fatalf("CreatePipelineExecution failed: %v", err)
	}

	// Check initial status
	status := pe.GetPipelineStatus(ctx)
	if status["overall_status"] != "PENDING" {
		t.Errorf("Expected PENDING status, got %v", status["overall_status"])
	}

	// Execute a few stages
	stages := []PipelineStage{StageHash, StageSign, StageEvidence}
	for _, stage := range stages {
		if err := pe.StartStage(ctx, stage); err != nil {
			t.Fatalf("StartStage(%s) failed: %v", stage, err)
		}

		if stage == StageHash {
			pe.RecordArtifactHash(ctx, "2222222222222222222222222222222222222222222222222222222222222222")
		}

		if stage == StageSign {
			pe.RecordArtifactSignature(ctx, "sig")
		}

		if err := pe.CompleteStage(ctx, stage, ""); err != nil {
			t.Fatalf("CompleteStage(%s) failed: %v", stage, err)
		}
	}

	// Check status update
	status = pe.GetPipelineStatus(ctx)
	if status["overall_status"] != "IN_PROGRESS" {
		t.Errorf("Expected IN_PROGRESS status, got %v", status["overall_status"])
	}

	stageMap := status["stages"].(map[string]string)
	if stageMap["HASH"] != "SUCCESS" {
		t.Errorf("Expected HASH stage SUCCESS, got %s", stageMap["HASH"])
	}

	t.Logf("PASS: Pipeline status reporting works correctly")
}
