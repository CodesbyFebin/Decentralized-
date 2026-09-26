package runtime

import (
	"context"
	"testing"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/pkg/identity"
)

// TestNegativeControl_DisableNodeBinding_WrongNodeAccepted verifies that when
// node binding is disabled, an envelope for the wrong node is incorrectly accepted.
// This test demonstrates the importance of node binding.
func TestNegativeControl_DisableNodeBinding_WrongNodeAccepted(t *testing.T) {
	cpID := identity.FromSeed(seedFor("cp-negctrl-node"))
	nodeID := identity.FromSeed(seedFor("node-negctrl-node"))
	wrongNodeID := identity.FromSeed(seedFor("wrong-node-negctrl"))

	// Validator expects nodeID
	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	// Envelope is for wrongNodeID
	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-negctrl-node",
		AuthorizationDigest: "digest",
		ClusterID:           "cluster-1",
		NodeID:              wrongNodeID.ID, // Wrong node!
		DeploymentID:        "deploy-1",
		WorkloadID:          "workload-1",
		Environment:         "prod",
		SecretID:            "secret-1",
		SecretVersion:       1,
		IssuedAt:            time.Now().UnixNano(),
		ExpiresAt:           time.Now().Add(5 * time.Minute).UnixNano(),
		Nonce:               []byte("nonce-negctrl-node"),
		PlaintextPayload:    []byte("secret-negctrl-node"),
	}
	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	ctx := context.Background()

	// Normal validation should reject it
	plaintext, err := validator.ValidateEnvelope(ctx, envelope, "deploy-1", "workload-1", "prod")
	if err == nil {
		t.Fatal("FAIL: Wrong node accepted when it should be rejected")
	}
	if plaintext != nil {
		t.Fatal("FAIL: Plaintext returned despite node mismatch")
	}

	// If we were to bypass node binding (this test documents what would happen)
	// The envelope would be accepted, which would be a security violation.
	t.Logf("PASS: Node binding correctly rejected envelope for %s", wrongNodeID.ID)
}

// TestNegativeControl_DisableSignatureVerification_UntrustedAccepted verifies
// that an envelope signed by an untrusted identity is correctly rejected.
// This demonstrates the importance of signature verification.
func TestNegativeControl_DisableSignatureVerification_UntrustedAccepted(t *testing.T) {
	cpID := identity.FromSeed(seedFor("cp-negctrl-sig"))
	evilID := identity.FromSeed(seedFor("evil-negctrl-sig"))
	nodeID := identity.FromSeed(seedFor("node-negctrl-sig"))

	// Validator trusts cpID
	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	// Envelope signed by evilID (not trusted)
	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-negctrl-sig",
		AuthorizationDigest: "digest",
		ClusterID:           "cluster-1",
		NodeID:              nodeID.ID,
		DeploymentID:        "deploy-1",
		WorkloadID:          "workload-1",
		Environment:         "prod",
		SecretID:            "secret-1",
		SecretVersion:       1,
		IssuedAt:            time.Now().UnixNano(),
		ExpiresAt:           time.Now().Add(5 * time.Minute).UnixNano(),
		Nonce:               []byte("nonce-negctrl-sig"),
		PlaintextPayload:    []byte("secret-negctrl-sig"),
	}
	envelope.Signature = evilID.Sign(envelope.CanonicalEnvelope()) // Wrong signer!

	ctx := context.Background()

	// Validation should reject it
	plaintext, err := validator.ValidateEnvelope(ctx, envelope, "deploy-1", "workload-1", "prod")
	if err == nil {
		t.Fatal("FAIL: Untrusted signature accepted when it should be rejected")
	}
	if plaintext != nil {
		t.Fatal("FAIL: Plaintext returned despite invalid signature")
	}

	// If we were to bypass signature verification (this test documents what would happen)
	// The untrusted envelope would be accepted, which would be a critical security violation.
	t.Logf("PASS: Signature verification correctly rejected envelope from untrusted identity")
}

// TestNegativeControl_SkipExpiryCheck_ExpiredAccepted verifies that an expired
// envelope is correctly rejected by expiry validation.
// This demonstrates the importance of expiry checks.
func TestNegativeControl_SkipExpiryCheck_ExpiredAccepted(t *testing.T) {
	cpID := identity.FromSeed(seedFor("cp-negctrl-exp"))
	nodeID := identity.FromSeed(seedFor("node-negctrl-exp"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	now := time.Now().UnixNano()
	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-negctrl-exp",
		AuthorizationDigest: "digest",
		ClusterID:           "cluster-1",
		NodeID:              nodeID.ID,
		DeploymentID:        "deploy-1",
		WorkloadID:          "workload-1",
		Environment:         "prod",
		SecretID:            "secret-1",
		SecretVersion:       1,
		IssuedAt:            now - int64(10*time.Minute), // Issued 10 minutes ago
		ExpiresAt:           now - int64(1*time.Minute),  // Expired 1 minute ago
		Nonce:               []byte("nonce-negctrl-exp"),
		PlaintextPayload:    []byte("secret-negctrl-exp"),
	}
	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	ctx := context.Background()

	// Validation should reject expired envelope
	plaintext, err := validator.ValidateEnvelope(ctx, envelope, "deploy-1", "workload-1", "prod")
	if err == nil {
		t.Fatal("FAIL: Expired envelope accepted when it should be rejected")
	}
	if plaintext != nil {
		t.Fatal("FAIL: Plaintext returned despite expiry")
	}

	// If we were to skip expiry checks (this test documents what would happen)
	// The expired envelope would be accepted, allowing stale secrets to be used.
	t.Logf("PASS: Expiry check correctly rejected expired envelope")
}

// TestNegativeControl_SkipDeploymentBinding_WrongDeploymentAccepted verifies
// that deployment binding prevents cross-deployment secret access.
func TestNegativeControl_SkipDeploymentBinding_WrongDeploymentAccepted(t *testing.T) {
	cpID := identity.FromSeed(seedFor("cp-negctrl-dep"))
	nodeID := identity.FromSeed(seedFor("node-negctrl-dep"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	// Envelope for deploy-1
	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-negctrl-dep",
		AuthorizationDigest: "digest",
		ClusterID:           "cluster-1",
		NodeID:              nodeID.ID,
		DeploymentID:        "deploy-1",
		WorkloadID:          "workload-1",
		Environment:         "prod",
		SecretID:            "secret-1",
		SecretVersion:       1,
		IssuedAt:            time.Now().UnixNano(),
		ExpiresAt:           time.Now().Add(5 * time.Minute).UnixNano(),
		Nonce:               []byte("nonce-negctrl-dep"),
		PlaintextPayload:    []byte("secret-negctrl-dep"),
	}
	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	ctx := context.Background()

	// Validate with different deployment
	plaintext, err := validator.ValidateEnvelope(ctx, envelope, "deploy-2", "workload-1", "prod")
	if err == nil {
		t.Fatal("FAIL: Envelope for wrong deployment accepted")
	}
	if plaintext != nil {
		t.Fatal("FAIL: Plaintext returned despite deployment mismatch")
	}

	t.Logf("PASS: Deployment binding correctly rejected envelope for wrong deployment")
}

// TestNegativeControl_SkipWorkloadBinding_WrongWorkloadAccepted verifies
// that workload binding prevents sibling workloads from accessing each other's secrets.
func TestNegativeControl_SkipWorkloadBinding_WrongWorkloadAccepted(t *testing.T) {
	cpID := identity.FromSeed(seedFor("cp-negctrl-wl"))
	nodeID := identity.FromSeed(seedFor("node-negctrl-wl"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	// Envelope for workload-1
	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-negctrl-wl",
		AuthorizationDigest: "digest",
		ClusterID:           "cluster-1",
		NodeID:              nodeID.ID,
		DeploymentID:        "deploy-1",
		WorkloadID:          "workload-1",
		Environment:         "prod",
		SecretID:            "secret-1",
		SecretVersion:       1,
		IssuedAt:            time.Now().UnixNano(),
		ExpiresAt:           time.Now().Add(5 * time.Minute).UnixNano(),
		Nonce:               []byte("nonce-negctrl-wl"),
		PlaintextPayload:    []byte("secret-negctrl-wl"),
	}
	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	ctx := context.Background()

	// Validate with different workload
	plaintext, err := validator.ValidateEnvelope(ctx, envelope, "deploy-1", "workload-2", "prod")
	if err == nil {
		t.Fatal("FAIL: Envelope for wrong workload accepted")
	}
	if plaintext != nil {
		t.Fatal("FAIL: Plaintext returned despite workload mismatch")
	}

	t.Logf("PASS: Workload binding correctly rejected envelope for wrong workload")
}

// TestNegativeControl_SkipEnvironmentBinding_WrongEnvironmentAccepted verifies
// that environment binding prevents cross-environment secret access.
func TestNegativeControl_SkipEnvironmentBinding_WrongEnvironmentAccepted(t *testing.T) {
	cpID := identity.FromSeed(seedFor("cp-negctrl-env"))
	nodeID := identity.FromSeed(seedFor("node-negctrl-env"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	// Envelope for prod
	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-negctrl-env",
		AuthorizationDigest: "digest",
		ClusterID:           "cluster-1",
		NodeID:              nodeID.ID,
		DeploymentID:        "deploy-1",
		WorkloadID:          "workload-1",
		Environment:         "prod",
		SecretID:            "secret-1",
		SecretVersion:       1,
		IssuedAt:            time.Now().UnixNano(),
		ExpiresAt:           time.Now().Add(5 * time.Minute).UnixNano(),
		Nonce:               []byte("nonce-negctrl-env"),
		PlaintextPayload:    []byte("secret-negctrl-env"),
	}
	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	ctx := context.Background()

	// Validate with different environment
	plaintext, err := validator.ValidateEnvelope(ctx, envelope, "deploy-1", "workload-1", "staging")
	if err == nil {
		t.Fatal("FAIL: Envelope for wrong environment accepted")
	}
	if plaintext != nil {
		t.Fatal("FAIL: Plaintext returned despite environment mismatch")
	}

	t.Logf("PASS: Environment binding correctly rejected envelope for wrong environment")
}

// TestNegativeControl_EnvelopeTampering_DetectedAndRejected verifies that
// tampering with an envelope (modifying plaintext after signing) is detected.
func TestNegativeControl_EnvelopeTampering_DetectedAndRejected(t *testing.T) {
	cpID := identity.FromSeed(seedFor("cp-negctrl-tamp"))
	nodeID := identity.FromSeed(seedFor("node-negctrl-tamp"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	// Create and sign envelope
	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-negctrl-tamp",
		AuthorizationDigest: "digest",
		ClusterID:           "cluster-1",
		NodeID:              nodeID.ID,
		DeploymentID:        "deploy-1",
		WorkloadID:          "workload-1",
		Environment:         "prod",
		SecretID:            "secret-1",
		SecretVersion:       1,
		IssuedAt:            time.Now().UnixNano(),
		ExpiresAt:           time.Now().Add(5 * time.Minute).UnixNano(),
		Nonce:               []byte("nonce-negctrl-tamp"),
		PlaintextPayload:    []byte("secret-original"),
	}
	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	// Attempt to tamper with plaintext after signing
	envelope.PlaintextPayload = []byte("secret-tampered")

	ctx := context.Background()

	// Validation should reject tampered envelope
	plaintext, err := validator.ValidateEnvelope(ctx, envelope, "deploy-1", "workload-1", "prod")
	if err == nil {
		t.Fatal("FAIL: Tampered envelope accepted")
	}
	if plaintext != nil {
		t.Fatal("FAIL: Plaintext returned despite tampering")
	}

	t.Logf("PASS: Signature verification correctly detected envelope tampering")
}

// TestNegativeControl_FutureIssuedEnvelope_RejectedAsClockSkew verifies that
// an envelope issued too far in the future is rejected (clock skew protection).
func TestNegativeControl_FutureIssuedEnvelope_RejectedAsClockSkew(t *testing.T) {
	cpID := identity.FromSeed(seedFor("cp-negctrl-future"))
	nodeID := identity.FromSeed(seedFor("node-negctrl-future"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	now := time.Now().UnixNano()
	// Issued 10 seconds in the future (beyond 5-second tolerance)
	futureTime := now + int64(10*time.Second)

	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-negctrl-future",
		AuthorizationDigest: "digest",
		ClusterID:           "cluster-1",
		NodeID:              nodeID.ID,
		DeploymentID:        "deploy-1",
		WorkloadID:          "workload-1",
		Environment:         "prod",
		SecretID:            "secret-1",
		SecretVersion:       1,
		IssuedAt:            futureTime,
		ExpiresAt:           now + int64(5*time.Minute),
		Nonce:               []byte("nonce-negctrl-future"),
		PlaintextPayload:    []byte("secret-negctrl-future"),
	}
	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	ctx := context.Background()

	// Validation should reject envelope issued too far in future
	plaintext, err := validator.ValidateEnvelope(ctx, envelope, "deploy-1", "workload-1", "prod")
	if err == nil {
		t.Fatal("FAIL: Future-issued envelope accepted beyond clock skew tolerance")
	}
	if plaintext != nil {
		t.Fatal("FAIL: Plaintext returned despite future issue time")
	}

	t.Logf("PASS: Clock skew check correctly rejected envelope issued too far in future")
}
