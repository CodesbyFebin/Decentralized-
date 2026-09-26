package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PipelineStage defines deployment pipeline stages
type PipelineStage string

const (
	StageSource        PipelineStage = "SOURCE"         // Fetch source artifact
	StageResolve       PipelineStage = "RESOLVE"        // Resolve dependencies
	StageInstall       PipelineStage = "INSTALL"        // Install dependencies
	StageBuild         PipelineStage = "BUILD"          // Build workload
	StageTest          PipelineStage = "TEST"           // Run tests
	StagePackage       PipelineStage = "PACKAGE"        // Package artifact
	StageHash          PipelineStage = "HASH"           // Compute SHA256 hash
	StageSign          PipelineStage = "SIGN"           // Sign artifact with Ed25519
	StageArtifactReady PipelineStage = "ARTIFACT_READY" // Artifact ready for deployment
	StageMatch         PipelineStage = "MATCH"          // Match nodes to workload spec
	StagePlace         PipelineStage = "PLACE"          // Place workload on nodes
	StageStart         PipelineStage = "START"          // Start workload
	StageHealth        PipelineStage = "HEALTH"         // Health check
	StageRoute         PipelineStage = "ROUTE"          // Configure routing
	StageObserve       PipelineStage = "OBSERVE"        // Observe metrics
	StageEvidence      PipelineStage = "EVIDENCE"       // Record deployment evidence
)

// PipelineStageProgress tracks progress through a single stage
type PipelineStageProgress struct {
	Stage      PipelineStage
	Status     string // "PENDING", "IN_PROGRESS", "SUCCESS", "FAILED", "SKIPPED"
	StartTime  int64  // Nanoseconds since epoch
	EndTime    int64  // Nanoseconds since epoch
	Duration   int64  // Nanoseconds
	ErrorMsg   string // Error message if failed
	Output     string // Stage output (hash, signature, node list, etc.)
	ExecutedOn string // Node/component that executed this stage
}

// PipelineExecution tracks full deployment pipeline execution
type PipelineExecution struct {
	WorkloadID      string
	SpecHash        string // Hash of deployment spec
	SourceHash      string // SHA256 of source artifact
	SourceSignature string // Ed25519 signature of source
	Stages          map[PipelineStage]*PipelineStageProgress
	CurrentStage    PipelineStage
	Status          string // "PENDING", "IN_PROGRESS", "SUCCESS", "FAILED"
	StartTime       int64
	EndTime         int64
	Duration        int64
	NodesExecuted   []string
	Evidence        *DeploymentEvidence
	mu              sync.RWMutex
}

// NewPipelineExecution creates a new pipeline execution
func NewPipelineExecution(workloadID string, spec *DeploymentSpec) *PipelineExecution {
	return &PipelineExecution{
		WorkloadID:    workloadID,
		SpecHash:      spec.SpecHash,
		Stages:        make(map[PipelineStage]*PipelineStageProgress),
		CurrentStage:  StageSource,
		Status:        "PENDING",
		StartTime:     time.Now().UnixNano(),
		NodesExecuted: []string{},
	}
}

// StartStage marks a stage as in progress
func (pe *PipelineExecution) StartStage(ctx context.Context, stage PipelineStage) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if pe.Status == "FAILED" {
		return fmt.Errorf("cannot start stage on failed pipeline")
	}

	progress := &PipelineStageProgress{
		Stage:     stage,
		Status:    "IN_PROGRESS",
		StartTime: time.Now().UnixNano(),
	}

	pe.Stages[stage] = progress
	pe.CurrentStage = stage
	pe.Status = "IN_PROGRESS"

	return nil
}

// CompleteStage marks a stage as completed successfully
func (pe *PipelineExecution) CompleteStage(ctx context.Context, stage PipelineStage, output string) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	progress, ok := pe.Stages[stage]
	if !ok {
		return fmt.Errorf("stage %s not started", stage)
	}

	if progress.Status != "IN_PROGRESS" {
		return fmt.Errorf("stage %s not in progress", stage)
	}

	now := time.Now().UnixNano()
	progress.Status = "SUCCESS"
	progress.EndTime = now
	progress.Duration = now - progress.StartTime
	progress.Output = output

	return nil
}

// FailStage marks a stage as failed
func (pe *PipelineExecution) FailStage(ctx context.Context, stage PipelineStage, errMsg string) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	progress, ok := pe.Stages[stage]
	if !ok {
		return fmt.Errorf("stage %s not started", stage)
	}

	now := time.Now().UnixNano()
	progress.Status = "FAILED"
	progress.EndTime = now
	progress.Duration = now - progress.StartTime
	progress.ErrorMsg = errMsg
	pe.Status = "FAILED"

	return nil
}

// RecordArtifactHash records source artifact hash at HASH stage
func (pe *PipelineExecution) RecordArtifactHash(ctx context.Context, hash string) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if len(hash) != 64 {
		return fmt.Errorf("invalid hash length: expected 64 hex chars, got %d", len(hash))
	}

	pe.SourceHash = hash
	return nil
}

// RecordArtifactSignature records Ed25519 signature at SIGN stage
func (pe *PipelineExecution) RecordArtifactSignature(ctx context.Context, signature string) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if signature == "" {
		return fmt.Errorf("signature cannot be empty")
	}

	pe.SourceSignature = signature
	return nil
}

// RecordNodeExecution records which node executed a stage
func (pe *PipelineExecution) RecordNodeExecution(ctx context.Context, stage PipelineStage, nodeID string) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	progress, ok := pe.Stages[stage]
	if !ok {
		return fmt.Errorf("stage %s not started", stage)
	}

	progress.ExecutedOn = nodeID

	// Track unique nodes
	found := false
	for _, n := range pe.NodesExecuted {
		if n == nodeID {
			found = true
			break
		}
	}
	if !found {
		pe.NodesExecuted = append(pe.NodesExecuted, nodeID)
	}

	return nil
}

// FinalizePipeline marks pipeline execution as complete
func (pe *PipelineExecution) FinalizePipeline(ctx context.Context) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if pe.Status != "IN_PROGRESS" {
		return fmt.Errorf("pipeline not in progress")
	}

	// Check all critical stages completed
	criticalStages := []PipelineStage{
		StageHash, StageSign, StagePlace, StageStart, StageEvidence,
	}

	for _, stage := range criticalStages {
		progress, ok := pe.Stages[stage]
		if !ok || progress.Status != "SUCCESS" {
			pe.Status = "FAILED"
			return fmt.Errorf("critical stage %s not completed successfully", stage)
		}
	}

	now := time.Now().UnixNano()
	pe.EndTime = now
	pe.Duration = now - pe.StartTime
	pe.Status = "SUCCESS"

	return nil
}

// GetStageProgress retrieves progress for a specific stage
func (pe *PipelineExecution) GetStageProgress(ctx context.Context, stage PipelineStage) (*PipelineStageProgress, error) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	progress, ok := pe.Stages[stage]
	if !ok {
		return nil, fmt.Errorf("no progress for stage %s", stage)
	}

	return progress, nil
}

// GetPipelineStatus returns current pipeline execution status
func (pe *PipelineExecution) GetPipelineStatus(ctx context.Context) map[string]interface{} {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	stageStatuses := make(map[string]string)
	for stage, progress := range pe.Stages {
		stageStatuses[string(stage)] = progress.Status
	}

	return map[string]interface{}{
		"workload_id":    pe.WorkloadID,
		"spec_hash":      pe.SpecHash,
		"source_hash":    pe.SourceHash,
		"current_stage":  string(pe.CurrentStage),
		"overall_status": pe.Status,
		"stages":         stageStatuses,
		"nodes_executed": pe.NodesExecuted,
		"duration_ns":    pe.Duration,
	}
}

// DeploymentPipeline manages pipeline executions
type DeploymentPipeline struct {
	pipelines map[string]*PipelineExecution
	contract  *DeploymentContract
	mu        sync.RWMutex
}

// NewDeploymentPipeline creates a new deployment pipeline manager
func NewDeploymentPipeline(contract *DeploymentContract) *DeploymentPipeline {
	return &DeploymentPipeline{
		pipelines: make(map[string]*PipelineExecution),
		contract:  contract,
	}
}

// CreatePipelineExecution creates a new pipeline execution for a workload
func (dp *DeploymentPipeline) CreatePipelineExecution(ctx context.Context, workloadID string, spec *DeploymentSpec) (*PipelineExecution, error) {
	if err := dp.contract.ValidateDeploymentSpec(ctx, spec); err != nil {
		return nil, fmt.Errorf("spec validation failed: %v", err)
	}

	if err := dp.contract.ValidateArtifact(ctx, spec.ArtifactRef); err != nil {
		return nil, fmt.Errorf("artifact verification failed: %v", err)
	}

	dp.mu.Lock()
	defer dp.mu.Unlock()

	pe := NewPipelineExecution(workloadID, spec)
	dp.pipelines[workloadID] = pe

	return pe, nil
}

// GetPipelineExecution retrieves an existing pipeline execution
func (dp *DeploymentPipeline) GetPipelineExecution(ctx context.Context, workloadID string) (*PipelineExecution, error) {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	pe, ok := dp.pipelines[workloadID]
	if !ok {
		return nil, fmt.Errorf("no pipeline execution for workload %s", workloadID)
	}

	return pe, nil
}

// DeletePipelineExecution removes a completed pipeline execution
func (dp *DeploymentPipeline) DeletePipelineExecution(ctx context.Context, workloadID string) error {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	pe, ok := dp.pipelines[workloadID]
	if !ok {
		return fmt.Errorf("no pipeline execution for workload %s", workloadID)
	}

	if pe.Status != "SUCCESS" && pe.Status != "FAILED" {
		return fmt.Errorf("cannot delete pipeline in %s state", pe.Status)
	}

	delete(dp.pipelines, workloadID)
	return nil
}

// GetAllPipelineExecutions returns all pipeline executions
func (dp *DeploymentPipeline) GetAllPipelineExecutions(ctx context.Context) []*PipelineExecution {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	execs := make([]*PipelineExecution, 0, len(dp.pipelines))
	for _, pe := range dp.pipelines {
		execs = append(execs, pe)
	}

	return execs
}
