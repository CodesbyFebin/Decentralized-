package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"decentralized.host/pkg/control"
)

// SecretDeliveryValidator validates SecretDeliveryEnvelope against context and agent identity.
// A05-P1-R1: Comprehensive validation before materialization ensures context binding is enforced.
type SecretDeliveryValidator struct {
	nodeID               string // authenticated agent identity (dh1...)
	clusterID            string // cluster scope
	controlPlanePublicKey string // wire-encoded public key of signing control plane
	now                  func() time.Time // for testing time-based checks
}

// NewSecretDeliveryValidator creates a validator for received envelopes.
// nodeID must be the agent's authenticated identity (matches envelope.NodeID for delivery validity).
// controlPlanePublicKey is base64url-encoded Ed25519 public key.
func NewSecretDeliveryValidator(nodeID, clusterID, controlPlanePublicKey string) *SecretDeliveryValidator {
	return &SecretDeliveryValidator{
		nodeID:               nodeID,
		clusterID:            clusterID,
		controlPlanePublicKey: controlPlanePublicKey,
		now:                  time.Now,
	}
}

// ValidateEnvelope performs comprehensive validation of a SecretDeliveryEnvelope.
// Returns error if any check fails; plaintext only if all checks pass.
// Validation order: protocol, signature, expiry, node binding, cluster binding, context match.
func (v *SecretDeliveryValidator) ValidateEnvelope(ctx context.Context, envelope *control.SecretDeliveryEnvelope,
	expectedDeploymentID, expectedWorkloadID, expectedEnvironment string) ([]byte, error) {

	// 1. Verify protocol version
	if envelope.ProtocolVersion != 1 {
		return nil, fmt.Errorf("envelope version %d not supported", envelope.ProtocolVersion)
	}

	// 2. Verify signature (proves envelope came from authorized control plane)
	if err := envelope.VerifySignature(v.controlPlanePublicKey); err != nil {
		return nil, fmt.Errorf("envelope signature verification failed: %w", err)
	}

	// 3. Verify expiry (reject if expired)
	now := v.now().UnixNano()
	if now > envelope.ExpiresAt {
		return nil, fmt.Errorf("envelope expired at %d (current time: %d)", envelope.ExpiresAt, now)
	}

	// 4. Verify issued time is not in future (clock skew tolerance: +5 seconds)
	if envelope.IssuedAt > now+int64(5*time.Second) {
		return nil, fmt.Errorf("envelope issued at future time %d (current time: %d)", envelope.IssuedAt, now)
	}

	// 5. Verify node binding (must match this agent's identity)
	// This prevents transplantation of envelope to wrong node
	if envelope.NodeID != v.nodeID {
		return nil, fmt.Errorf("envelope target node %s does not match agent identity %s", envelope.NodeID, v.nodeID)
	}

	// 6. Verify cluster scope binding
	if envelope.ClusterID != v.clusterID {
		return nil, fmt.Errorf("envelope cluster %s does not match agent cluster %s", envelope.ClusterID, v.clusterID)
	}

	// 7. Verify deployment scope binding
	if envelope.DeploymentID != expectedDeploymentID {
		return nil, fmt.Errorf("envelope deployment %s does not match expected deployment %s", envelope.DeploymentID, expectedDeploymentID)
	}

	// 8. Verify workload scope binding
	if envelope.WorkloadID != expectedWorkloadID {
		return nil, fmt.Errorf("envelope workload %s does not match expected workload %s", envelope.WorkloadID, expectedWorkloadID)
	}

	// 9. Verify environment scope binding
	if envelope.Environment != expectedEnvironment {
		return nil, fmt.Errorf("envelope environment %s does not match expected environment %s", envelope.Environment, expectedEnvironment)
	}

	// All validations passed; return plaintext payload
	return envelope.PlaintextPayload, nil
}

// SecretDeliveryReceiver handles agent-side secret retrieval with envelope validation and materialization.
// A05-P1-R1: Receive envelope from control plane, validate completely, materialize locally.
type SecretDeliveryReceiver struct {
	validator    *SecretDeliveryValidator
	materializer *Materializer
}

// NewSecretDeliveryReceiver creates a receiver for secret delivery.
func NewSecretDeliveryReceiver(validator *SecretDeliveryValidator, materializer *Materializer) *SecretDeliveryReceiver {
	return &SecretDeliveryReceiver{validator: validator, materializer: materializer}
}

// ReceiveAndMaterialize performs the complete secret delivery workflow:
// 1. Receives envelope from control plane (caller supplies this)
// 2. Validates envelope signature, expiry, and all context bindings
// 3. Materializes validated plaintext to tmpfs on this node
// 4. Returns host path for mounting in workload namespace
// Returns error if validation fails; plaintext never written to disk if validation rejects.
func (r *SecretDeliveryReceiver) ReceiveAndMaterialize(ctx context.Context,
	envelope *control.SecretDeliveryEnvelope,
	expectedDeploymentID, expectedWorkloadID, expectedEnvironment string,
	assignmentID string) (hostPath string, err error) {

	// Validate envelope completely before touching filesystem
	plaintext, err := r.validator.ValidateEnvelope(ctx, envelope, expectedDeploymentID, expectedWorkloadID, expectedEnvironment)
	if err != nil {
		return "", fmt.Errorf("envelope validation failed: %w", err)
	}

	// Envelope is valid; materialize plaintext to agent-local tmpfs
	// This ensures plaintext exists only in agent memory, not on control plane or in transit
	hostPath, err = r.materializer.AllocateEphemeral(ctx, assignmentID, envelope.DeliveryID, plaintext)
	if err != nil {
		return "", fmt.Errorf("materialization failed: %w", err)
	}

	return hostPath, nil
}

// ConsumptionRecord tracks when a secret delivery was consumed (for audit and cleanup).
type ConsumptionRecord struct {
	DeliveryID       string    // unique delivery identifier
	EnvelopeDigest   string    // SHA256 of canonical envelope (immutable proof of what was authorized)
	NodeID           string    // which node materialized
	DeploymentID     string    // deployment scope
	WorkloadID       string    // workload scope
	SecretID         string    // which secret
	SecretVersion    int32     // which version
	MaterializedAt   int64     // Unix ns when materialized
	MaterializedPath string    // host path where materialized (for cleanup ownership)
	Status           string    // "MATERIALIZED", "CONSUMED", "ORPHANED", "CLEANED"
	CleanedAt        int64     // Unix ns when cleaned (0 if not yet cleaned)
	Error            string    // error if materialization failed (audit)
}

// DeliveryConsumedObserver allows tracking of secret deliveries (for audit and testing).
type DeliveryConsumedObserver interface {
	// Called after envelope validation succeeds (before materialization)
	ValidatedEnvelope(env *control.SecretDeliveryEnvelope)
	// Called after successful materialization
	MaterializedToPath(hostPath string, rec *ConsumptionRecord)
	// Called on any error
	DeliveryFailed(rec *ConsumptionRecord, err error)
	// Called when cleanup removes materialized secret
	CleanedUp(hostPath string)
}

// NoOpDeliveryObserver provides a default observer that does nothing.
type NoOpDeliveryObserver struct{}

func (o *NoOpDeliveryObserver) ValidatedEnvelope(*control.SecretDeliveryEnvelope) {}
func (o *NoOpDeliveryObserver) MaterializedToPath(string, *ConsumptionRecord)      {}
func (o *NoOpDeliveryObserver) DeliveryFailed(*ConsumptionRecord, error)           {}
func (o *NoOpDeliveryObserver) CleanedUp(string)                                   {}

// ErrEnvelopeSignatureInvalid means signature verification failed (untrusted envelope).
var ErrEnvelopeSignatureInvalid = errors.New("envelope signature invalid")

// ErrEnvelopeExpired means envelope was issued outside acceptable time window.
var ErrEnvelopeExpired = errors.New("envelope expired or issued in future")

// ErrNodeMismatch means envelope target node does not match agent identity.
var ErrNodeMismatch = errors.New("envelope target node does not match agent identity")

// ErrContextMismatch means envelope scope (deployment/workload/environment) does not match expected.
var ErrContextMismatch = errors.New("envelope scope mismatch")

// ErrMaterializationFailed means allocation of ephemeral tmpfs failed.
var ErrMaterializationFailed = errors.New("ephemeral materialization failed")
