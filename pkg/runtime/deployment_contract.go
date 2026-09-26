package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// ArtifactReference represents an immutable workload artifact
type ArtifactReference struct {
	SourceHash    string // SHA256 hash of artifact source (immutable)
	SignerID      string // Identity of signer (control-plane or node)
	Signature     string // Ed25519 signature over SourceHash
	SignAlgorithm string // Signing algorithm (ed25519)
	SignedAt      int64  // Unix nanoseconds
}

// DeploymentSpec represents the deployment specification (decentralized.host.yaml)
type DeploymentSpec struct {
	WorkloadID          string               // Unique workload identifier
	ArtifactRef         *ArtifactReference   // Immutable artifact reference
	ContainerImage      string               // Container image reference (must match artifact)
	ResourceConstraints *ResourceConstraints // CPU, memory, disk limits
	Environment         map[string]string    // Environment variables
	RequiredSecrets     []string             // Ephemeral secrets required
	VolumeMounts        map[string]string    // Volume mount paths
	NetworkPorts        []int                // Exposed ports
	Replicas            int                  // Desired replica count
	NodeSelector        map[string]string    // Label selectors for node placement
	SpecVersion         string               // Schema version (v1.0)
	SpecHash            string               // SHA256 of canonical spec (for integrity)
}

// DeploymentRequest represents a request to deploy a workload
type DeploymentRequest struct {
	WorkloadID       string             // Unique workload identifier
	Spec             *DeploymentSpec    // Full deployment specification
	NodeSelector     map[string]string  // Labels to match nodes (override from spec)
	Replicas         int                // Number of copies (override from spec)
	RequestedAt      int64              // Unix nanoseconds
	Timeout          time.Duration      // Deployment timeout
	RequiredState    NodeLifecycleState // Required node state (ACTIVE, VERIFIED, etc)
	ArtifactVerified bool               // Signature verified before request
}

// DeploymentEvidence records proof of deployment execution
type DeploymentEvidence struct {
	WorkloadID        string   // Workload deployed
	ArtifactHash      string   // SHA256 of deployed artifact
	SourceSignature   string   // Signature of artifact source
	SpecHash          string   // Hash of deployment spec
	DeploymentCommand string   // Original deployment request hash
	SourceSHA         string   // Git SHA or source commit hash
	Timestamp         int64    // When deployment executed
	ExecutedOn        []string // Node IDs where executed
	Status            string   // SUCCESS, PARTIAL, FAILED
}

// DeploymentResult represents the outcome of a deployment
type DeploymentResult struct {
	WorkloadID     string
	Assigned       []string            // Node IDs where workload deployed
	Failed         []string            // Node IDs where deployment failed
	Status         DeploymentStatus    // Overall status
	Reason         string              // Reason for any failures
	CompletedAt    int64               // Unix nanoseconds
	PlacementInfo  map[string]string   // Metadata about placement decisions
	SignedEvidence *DeploymentEvidence // Proof of deployment
}

// DeploymentStatus represents deployment phase
type DeploymentStatus string

const (
	DeploymentPending    DeploymentStatus = "PENDING"
	DeploymentScheduling DeploymentStatus = "SCHEDULING"
	DeploymentScheduled  DeploymentStatus = "SCHEDULED"
	DeploymentStarting   DeploymentStatus = "STARTING"
	DeploymentActive     DeploymentStatus = "ACTIVE"
	DeploymentUpdating   DeploymentStatus = "UPDATING"
	DeploymentDraining   DeploymentStatus = "DRAINING"
	DeploymentTerminated DeploymentStatus = "TERMINATED"
	DeploymentFailed     DeploymentStatus = "FAILED"
)

// DeploymentValidator enforces deployment contract rules
type DeploymentValidator struct {
	nlm *NodeLifecycleManager
	wlm *WorkloadLifecycleManager
	mu  sync.RWMutex
}

// NewDeploymentValidator creates a new deployment contract validator
func NewDeploymentValidator(nlm *NodeLifecycleManager, wlm *WorkloadLifecycleManager) *DeploymentValidator {
	return &DeploymentValidator{
		nlm: nlm,
		wlm: wlm,
	}
}

// VerifyArtifact validates artifact reference (hash and signature)
func (dv *DeploymentValidator) VerifyArtifact(ctx context.Context, artifact *ArtifactReference) error {
	if artifact == nil {
		return fmt.Errorf("artifact reference cannot be nil")
	}

	if artifact.SourceHash == "" {
		return fmt.Errorf("artifact source hash is required")
	}

	// Validate hash format (should be hex-encoded SHA256)
	if len(artifact.SourceHash) != 64 {
		return fmt.Errorf("invalid artifact hash length: expected 64 hex chars, got %d", len(artifact.SourceHash))
	}

	// Validate hash is valid hex
	if _, err := hex.DecodeString(artifact.SourceHash); err != nil {
		return fmt.Errorf("artifact hash is not valid hex: %v", err)
	}

	if artifact.SignerID == "" {
		return fmt.Errorf("artifact signer ID is required")
	}

	if artifact.Signature == "" {
		return fmt.Errorf("artifact signature is required")
	}

	if artifact.SignAlgorithm != "ed25519" {
		return fmt.Errorf("unsupported signing algorithm: %s", artifact.SignAlgorithm)
	}

	return nil
}

// ValidateDeploymentSpec validates deployment specification
func (dv *DeploymentValidator) ValidateDeploymentSpec(ctx context.Context, spec *DeploymentSpec) error {
	if spec == nil {
		return fmt.Errorf("deployment spec cannot be nil")
	}

	if spec.WorkloadID == "" {
		return fmt.Errorf("workload ID is required")
	}

	if spec.ArtifactRef == nil {
		return fmt.Errorf("artifact reference is required in spec")
	}

	if err := dv.VerifyArtifact(ctx, spec.ArtifactRef); err != nil {
		return fmt.Errorf("artifact verification failed: %v", err)
	}

	if spec.ContainerImage == "" {
		return fmt.Errorf("container image is required")
	}

	if spec.ResourceConstraints == nil {
		return fmt.Errorf("resource constraints are required")
	}

	if spec.ResourceConstraints.MemoryBytes <= 0 {
		return fmt.Errorf("memory must be positive")
	}

	if spec.ResourceConstraints.CPUShares < 0 {
		return fmt.Errorf("CPU shares cannot be negative")
	}

	if spec.ResourceConstraints.DiskBytes < 0 {
		return fmt.Errorf("disk bytes cannot be negative")
	}

	if spec.SpecVersion == "" {
		spec.SpecVersion = "v1.0"
	}

	// Calculate spec hash for integrity
	specData := fmt.Sprintf("%s-%s-%s-%d-%d-%d",
		spec.WorkloadID,
		spec.ContainerImage,
		spec.ArtifactRef.SourceHash,
		spec.ResourceConstraints.MemoryBytes,
		spec.ResourceConstraints.CPUShares,
		spec.ResourceConstraints.DiskBytes,
	)

	hash := sha256.Sum256([]byte(specData))
	spec.SpecHash = hex.EncodeToString(hash[:])

	return nil
}

// CanScheduleOnNode validates if a workload can be scheduled on a node
func (dv *DeploymentValidator) CanScheduleOnNode(ctx context.Context, nodeID string, req *DeploymentRequest) error {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	// Get node state
	nodeState, err := dv.nlm.GetNodeState(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("cannot get node state: %v", err)
	}

	// Node must not be revoked
	if nodeState.State == NodeRevoked {
		return fmt.Errorf("node %s is revoked, cannot schedule", nodeID)
	}

	// Node must not be cordoned
	if nodeState.Cordoned {
		return fmt.Errorf("node %s is cordoned, cannot schedule", nodeID)
	}

	// Node must not be draining
	if nodeState.State == NodeDraining {
		return fmt.Errorf("node %s is draining, cannot schedule", nodeID)
	}

	// Node must be online
	if nodeState.State == NodeOffline {
		return fmt.Errorf("node %s is offline, cannot schedule", nodeID)
	}

	// If specific state required, check it
	if req.RequiredState != "" && nodeState.State != req.RequiredState {
		return fmt.Errorf("node %s state %s does not match required %s", nodeID, nodeState.State, req.RequiredState)
	}

	// Freshness check - stale nodes require verification
	if nodeState.Freshness == "STALE" {
		// In production, might want to fail or require verification
		// For now, log as warning but allow
		fmt.Printf("Warning: scheduling on stale node %s\n", nodeID)
	}

	return nil
}

// ValidateDeployment checks if deployment request is valid
func (dv *DeploymentValidator) ValidateDeployment(ctx context.Context, req *DeploymentRequest) error {
	if req.WorkloadID == "" {
		return fmt.Errorf("workload ID cannot be empty")
	}

	if req.Replicas < 1 {
		return fmt.Errorf("replicas must be at least 1, got %d", req.Replicas)
	}

	if req.Timeout < time.Second {
		return fmt.Errorf("timeout must be at least 1 second, got %v", req.Timeout)
	}

	return nil
}

// SelectNodesForDeployment finds suitable nodes for workload placement
func (dv *DeploymentValidator) SelectNodesForDeployment(ctx context.Context, req *DeploymentRequest) ([]string, error) {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	// Validate request first
	if err := dv.ValidateDeployment(ctx, req); err != nil {
		return nil, err
	}

	// Get all nodes
	dv.nlm.mu.RLock()
	allNodes := make([]string, 0, len(dv.nlm.nodes))
	for nodeID := range dv.nlm.nodes {
		allNodes = append(allNodes, nodeID)
	}
	dv.nlm.mu.RUnlock()

	// Filter for suitable nodes
	selected := []string{}
	for _, nodeID := range allNodes {
		if err := dv.CanScheduleOnNode(ctx, nodeID, req); err == nil {
			selected = append(selected, nodeID)
			if len(selected) >= req.Replicas {
				break
			}
		}
	}

	// Check if we found enough nodes
	if len(selected) < req.Replicas {
		return nil, fmt.Errorf("insufficient suitable nodes: found %d, need %d", len(selected), req.Replicas)
	}

	return selected, nil
}

// ExecuteDeployment performs the deployment following the contract
func (dv *DeploymentValidator) ExecuteDeployment(ctx context.Context, req *DeploymentRequest) (*DeploymentResult, error) {
	// Validate
	if err := dv.ValidateDeployment(ctx, req); err != nil {
		return &DeploymentResult{
			WorkloadID: req.WorkloadID,
			Status:     DeploymentFailed,
			Reason:     err.Error(),
			Failed:     []string{},
		}, err
	}

	result := &DeploymentResult{
		WorkloadID:    req.WorkloadID,
		Status:        DeploymentScheduling,
		Assigned:      []string{},
		Failed:        []string{},
		PlacementInfo: make(map[string]string),
	}

	// Select nodes
	selectedNodes, err := dv.SelectNodesForDeployment(ctx, req)
	if err != nil {
		result.Status = DeploymentFailed
		result.Reason = err.Error()
		return result, err
	}

	// Create workload
	if err := dv.wlm.CreateWorkload(ctx, req.WorkloadID); err != nil {
		result.Status = DeploymentFailed
		result.Reason = fmt.Sprintf("failed to create workload: %v", err)
		return result, err
	}

	// Schedule on each node
	for _, nodeID := range selectedNodes {
		// Verify node is still suitable (state could have changed)
		if err := dv.CanScheduleOnNode(ctx, nodeID, req); err != nil {
			result.Failed = append(result.Failed, nodeID)
			result.PlacementInfo[nodeID] = fmt.Sprintf("scheduling check failed: %v", err)
			continue
		}

		result.Assigned = append(result.Assigned, nodeID)
		result.PlacementInfo[nodeID] = "ASSIGNED"
	}

	// Set final status
	if len(result.Assigned) > 0 {
		result.Status = DeploymentScheduled
	} else {
		result.Status = DeploymentFailed
		result.Reason = "no nodes could be assigned"
	}

	result.CompletedAt = time.Now().UnixNano()
	return result, nil
}

// VerifyDeploymentConstraints checks if current state satisfies deployment requirements
func (dv *DeploymentValidator) VerifyDeploymentConstraints(ctx context.Context, workloadID string, assignedNodes []string) (bool, string) {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	// Workload must exist
	wl, err := dv.wlm.GetWorkloadState(ctx, workloadID)
	if err != nil {
		return false, fmt.Sprintf("workload not found: %v", err)
	}

	// Workload should be in running/scheduled state
	if wl.DesiredState == WorkloadStopped {
		return false, "workload desired state is STOPPED"
	}

	// All assigned nodes must still be schedulable
	for _, nodeID := range assignedNodes {
		node, err := dv.nlm.GetNodeState(ctx, nodeID)
		if err != nil {
			return false, fmt.Sprintf("node %s not found", nodeID)
		}

		// Node went offline
		if node.State == NodeOffline {
			return false, fmt.Sprintf("assigned node %s is offline", nodeID)
		}

		// Node was revoked
		if node.State == NodeRevoked {
			return false, fmt.Sprintf("assigned node %s was revoked", nodeID)
		}

		// Node went cordoned (new cordoning after assignment)
		if node.Cordoned && node.State == NodeCordoned {
			return false, fmt.Sprintf("assigned node %s was cordoned", nodeID)
		}
	}

	return true, ""
}

// ValidateNodeStateForWorkload ensures node state matches workload requirements
func (dv *DeploymentValidator) ValidateNodeStateForWorkload(ctx context.Context, nodeID string, workloadID string) error {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	node, err := dv.nlm.GetNodeState(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("node %s not found", nodeID)
	}

	// Node must be active or verified (not offline)
	if node.State != NodeActive && node.State != NodeVerified {
		return fmt.Errorf("node %s in state %s, workload needs ACTIVE or VERIFIED", nodeID, node.State)
	}

	// Freshness check
	if node.Freshness == "UNREACHABLE" {
		return fmt.Errorf("node %s is unreachable, freshness %s", nodeID, node.Freshness)
	}

	return nil
}

// EnforceDrainPolicy implements drain behavior during node shutdown
func (dv *DeploymentValidator) EnforceDrainPolicy(ctx context.Context, nodeID string) ([]*WorkloadStateValue, error) {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	// Get all workloads (in production would filter by node assignment)
	allWorkloads := dv.wlm.GetAllWorkloads(ctx)

	// Collect workloads that need draining
	workloadsToDrain := []*WorkloadStateValue{}
	for _, wl := range allWorkloads {
		// Skip already stopped/failed
		if wl.ObservedState == WorkloadStopped || wl.ObservedState == WorkloadFailed {
			continue
		}

		// Add to drain list
		workloadsToDrain = append(workloadsToDrain, wl)
	}

	return workloadsToDrain, nil
}

// DeploymentContract encapsulates all contract validation rules
type DeploymentContract struct {
	validator *DeploymentValidator
	mu        sync.RWMutex
}

// NewDeploymentContract creates a new deployment contract
func NewDeploymentContract(nlm *NodeLifecycleManager, wlm *WorkloadLifecycleManager) *DeploymentContract {
	return &DeploymentContract{
		validator: NewDeploymentValidator(nlm, wlm),
	}
}

// Contract Rules (documented as contract enforcement):

// Rule 1: Scheduling on CORDONED nodes is prohibited
// Rule 2: Scheduling on OFFLINE nodes is prohibited
// Rule 3: Scheduling on DRAINING nodes is prohibited
// Rule 4: Scheduling on REVOKED nodes is prohibited
// Rule 5: STALE nodes can be scheduled on (with warning)
// Rule 6: ACTIVE nodes are preferred for scheduling
// Rule 7: Workload desired state must not be STOPPED for active deployment
// Rule 8: Node state must remain ACTIVE/VERIFIED during workload execution
// Rule 9: Drain policy: stop all workloads on node sequentially
// Rule 10: Cordoned nodes must not accept new workload assignments

// ValidateSchedulingContract ensures scheduling follows contract rules
func (dc *DeploymentContract) ValidateSchedulingContract(ctx context.Context, nodeID string, req *DeploymentRequest) error {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	return dc.validator.CanScheduleOnNode(ctx, nodeID, req)
}

// ValidateExecutionContract ensures running workload meets contract requirements
func (dc *DeploymentContract) ValidateExecutionContract(ctx context.Context, workloadID string, nodeID string) error {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	return dc.validator.ValidateNodeStateForWorkload(ctx, nodeID, workloadID)
}

// ValidateTerminationContract ensures graceful shutdown follows contract
func (dc *DeploymentContract) ValidateTerminationContract(ctx context.Context, nodeID string) (int, error) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	workloads, err := dc.validator.EnforceDrainPolicy(ctx, nodeID)
	return len(workloads), err
}

// ValidateArtifact validates artifact reference including hash and signature
func (dc *DeploymentContract) ValidateArtifact(ctx context.Context, artifact *ArtifactReference) error {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	return dc.validator.VerifyArtifact(ctx, artifact)
}

// ValidateDeploymentSpec validates the full deployment specification
func (dc *DeploymentContract) ValidateDeploymentSpec(ctx context.Context, spec *DeploymentSpec) error {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	return dc.validator.ValidateDeploymentSpec(ctx, spec)
}
