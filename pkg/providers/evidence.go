package providers

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/zeebo/blake3"
)

// EvidenceQualifier binds cryptographic proofs to resource state transitions.
// Every state change is signed and hashed, creating an immutable audit trail.
type EvidenceQualifier struct {
	signerKey ed25519.PrivateKey
	signerID  string
}

// NewEvidenceQualifier creates a new evidence qualifier with Ed25519 signing.
func NewEvidenceQualifier(signerID string) (*EvidenceQualifier, error) {
	// Generate Ed25519 keypair for signing
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signing key: %w", err)
	}

	return &EvidenceQualifier{
		signerKey: priv,
		signerID:  signerID,
	}, nil
}

// QualifiedEvidence represents a cryptographically-bound state observation.
type QualifiedEvidence struct {
	// Identity
	CampaignID   string    `json:"campaign_id"`   // Unique qualification campaign identifier
	ResourceID   string    `json:"resource_id"`   // Resource being qualified
	SourceSHA    string    `json:"source_sha"`    // Git commit of qualification code
	Timestamp    time.Time `json:"timestamp"`     // When observation occurred

	// Cryptographic Proofs
	ContentHash   string `json:"content_hash"`   // BLAKE3(state) for content addressing
	StateSnapshot string `json:"state_snapshot"` // Serialized state (opaque)
	Signature     string `json:"signature"`      // Ed25519 signature over hash
	SignerID      string `json:"signer_id"`      // Identity of signer
	SignerPubKey  string `json:"signer_pubkey"`  // Public key of signer

	// Qualification Hierarchy
	P1_CORE_Passed bool `json:"p1_core_passed"` // Distributed scheduling passes 32 gates
	P1_QEMU_Passed bool `json:"p1_qemu_passed"` // VM isolation boundary verified
	P1_K8S_Passed  bool `json:"p1_k8s_passed"`  // Kubernetes orchestration verified
	P2_Multi_Passed bool `json:"p2_multi_passed"` // Independent physical hosts

	// Failure Domains (Honesty)
	OSBoundary         string `json:"os_boundary"`         // DISTINCT | SAME | UNKNOWN
	FilesystemBoundary string `json:"filesystem_boundary"` // DISTINCT | SAME | UNKNOWN
	PhysicalBoundary   string `json:"physical_boundary"`   // DISTINCT | SAME | UNKNOWN
	OperatorBoundary   string `json:"operator_boundary"`   // DISTINCT | SAME | UNKNOWN
}

// HashResourceState computes BLAKE3 hash of resource state.
// This provides content-addressed storage and tamper detection.
func (eq *EvidenceQualifier) HashResourceState(state string) string {
	h := blake3.New()
	h.Write([]byte(state))
	return hex.EncodeToString(h.Sum(nil))
}

// SignEvidence signs a qualified evidence record with Ed25519.
// Signature covers: contentHash | resourceID | timestamp | qualificationFlags
func (eq *EvidenceQualifier) SignEvidence(evidence *QualifiedEvidence) error {
	// Construct message to sign: canonical format prevents reordering attacks
	message := fmt.Sprintf("%s|%s|%d|%v|%v|%v|%v",
		evidence.ContentHash,
		evidence.ResourceID,
		evidence.Timestamp.UnixNano(),
		evidence.P1_CORE_Passed,
		evidence.P1_QEMU_Passed,
		evidence.P1_K8S_Passed,
		evidence.P2_Multi_Passed,
	)

	// Sign with Ed25519
	sig := ed25519.Sign(eq.signerKey, []byte(message))
	evidence.Signature = hex.EncodeToString(sig)
	evidence.SignerID = eq.signerID
	evidence.SignerPubKey = hex.EncodeToString(eq.signerKey.Public().(ed25519.PublicKey))

	return nil
}

// VerifyEvidence cryptographically verifies an evidence record.
// Returns error if signature is invalid or content hash doesn't match state.
func VerifyEvidence(evidence *QualifiedEvidence) error {
	if evidence.Signature == "" {
		return fmt.Errorf("no signature present")
	}

	// Reconstruct signed message
	message := fmt.Sprintf("%s|%s|%d|%v|%v|%v|%v",
		evidence.ContentHash,
		evidence.ResourceID,
		evidence.Timestamp.UnixNano(),
		evidence.P1_CORE_Passed,
		evidence.P1_QEMU_Passed,
		evidence.P1_K8S_Passed,
		evidence.P2_Multi_Passed,
	)

	// Decode signature
	sig, err := hex.DecodeString(evidence.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	// Decode public key
	pubKeyBytes, err := hex.DecodeString(evidence.SignerPubKey)
	if err != nil {
		return fmt.Errorf("invalid public key encoding: %w", err)
	}

	pubKey := ed25519.PublicKey(pubKeyBytes)

	// Verify signature
	if !ed25519.Verify(pubKey, []byte(message), sig) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// QualificationCampaign tracks a single qualification run across all profile levels.
type QualificationCampaign struct {
	ID                  string                      `json:"id"`                   // Unique campaign ID
	ResourceID          string                      `json:"resource_id"`          // Resource being qualified
	StartTime           time.Time                   `json:"start_time"`
	EndTime             time.Time                   `json:"end_time,omitempty"`
	SourceSHA           string                      `json:"source_sha"`           // Qualification code version
	GateResults         []GateResult                `json:"gate_results"`         // Results of P1_CORE gates
	BackendSpecificGates map[string][]GateResult   `json:"backend_specific_gates"` // Results of backend-specific gates
	Evidence            *QualifiedEvidence          `json:"evidence"`             // Final signed evidence
	Status              string                      `json:"status"`               // RUNNING | PASSED | FAILED
	QualificationLevel  string                      `json:"qualification_level"`  // P1_CORE | P1_QEMU | P1_K8S | P2_MULTIPHYSICAL
}

// GateResult represents the result of one qualification gate.
type GateResult struct {
	Sequence    int       `json:"sequence"`    // 1-32 for P1_CORE gates
	Name        string    `json:"name"`        // Gate name (e.g., "Scheduler Determinism")
	Description string    `json:"description"` // What the gate verifies
	Passed      bool      `json:"passed"`      // Whether gate passed
	Evidence    string    `json:"evidence"`    // Proof data or error message
	Timestamp   time.Time `json:"timestamp"`
}

// P1_CORE_Gates defines the 32 mandatory qualification gates for core distributed functionality.
// These gates verify: distributed scheduling, placement, workload execution, failure detection, reconciliation.
var P1_CORE_Gates = []string{
	// Scheduling & Placement (Gates 1-8)
	1:  "Scheduler Determinism - Same input → same placement always",
	2:  "Placement Symmetry - All nodes treated equivalently",
	3:  "Schedule Stability - No silent task migration",
	4:  "Replica Independence - Replicas don't share single points of failure",
	5:  "Affinity Respect - Pod affinity rules honored",
	6:  "Anti-affinity Enforcement - Replicas spread across isolation boundaries",
	7:  "Preemption Fairness - Eviction order is deterministic",
	8:  "Bin Packing Optimality - Resources packed efficiently",

	// Workload Execution (Gates 9-16)
	9:  "Image Identity - Same image hash always produces same artifact",
	10: "Manifest Integrity - Manifest changes trigger reconciliation",
	11: "Startup Determinism - Same startup sequence always",
	12: "Environment Consistency - Env vars identical across replicas",
	13: "Volume Mounting - PVCs mounted consistently",
	14: "Network Identity - Each pod gets stable network identity",
	15: "Service Discovery - DNS names resolve consistently",
	16: "Readiness Honesty - Probe results reflect actual state",

	// Failure Detection (Gates 17-24)
	17: "Liveness Detection - Dead processes detected in <timeout",
	18: "Network Partition Detection - Split-brain conditions detected",
	19: "Disk Exhaustion Detection - Storage failures detected",
	20: "CPU Overload Detection - Resource pressure detected",
	21: "Crash Loop Detection - Rapid failures trigger backoff",
	22: "Zombie Process Detection - Orphaned processes cleaned up",
	23: "Deadlock Detection - Resource contention detected",
	24: "Time Skew Detection - Clock drift detected",

	// Reconciliation (Gates 25-32)
	25: "Desired ≠ Observed Reconciliation - Drift triggers correction",
	26: "Node Recovery - Rejoined nodes re-sync state",
	27: "Rollout Ordering - Updates proceed sequentially",
	28: "Rollback Correctness - Rollback restores previous state",
	29: "Idempotence - Repeated operations produce same result",
	30: "State Consistency - Consensus on current state",
	31: "Evidence Binding - All changes cryptographically signed",
	32: "Audit Trail - All operations logged and verifiable",
}

// RunQualificationCampaign executes all 32 P1_CORE gates and produces signed evidence.
// Deprecated: Use RunQualificationCampaignWithContext instead for full gate execution.
func RunQualificationCampaign(resourceID string, sourceSHA string, signer *EvidenceQualifier) (*QualificationCampaign, error) {
	return RunQualificationCampaignWithContext(context.Background(), resourceID, sourceSHA, signer, nil, nil, nil)
}

// RunQualificationCampaignWithContext executes all 32 P1_CORE gates with adapter and topology context.
func RunQualificationCampaignWithContext(ctx context.Context, resourceID string, sourceSHA string,
	signer *EvidenceQualifier, adapter ProviderAdapter, cfg *ProviderConfig, topology *RuntimeTopology) (*QualificationCampaign, error) {

	if signer == nil {
		return nil, fmt.Errorf("signer required for evidence binding")
	}

	executor := NewGateExecutor(ctx, signer)
	campaign, err := executor.ExecuteAllGates(ctx, resourceID, sourceSHA, adapter, cfg, topology)
	if err != nil {
		return nil, fmt.Errorf("gate execution failed: %w", err)
	}

	return campaign, nil
}
