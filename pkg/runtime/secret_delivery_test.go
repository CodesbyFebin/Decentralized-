package runtime

import (
	"context"
	"os"
	"testing"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/pkg/identity"
)

// seedFor generates a deterministic 32-byte seed from a name (for testing)
func seedFor(name string) []byte {
	seed := make([]byte, 32)
	for i := 0; i < len(name) && i < 32; i++ {
		seed[i] = name[i]
	}
	return seed
}

func TestSecretDeliveryValidator_ValidSignature_Accepted(t *testing.T) {
	// Create control plane identity and node identity
	cpID := identity.FromSeed(seedFor("control-plane-1"))
	nodeID := identity.FromSeed(seedFor("node-identity-1"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	// Create valid envelope
	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-123",
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
		Nonce:               []byte("nonce-data"),
		PlaintextPayload:    []byte("secret-plaintext"),
	}

	// Sign envelope with control plane
	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	// Validate should succeed
	plaintext, err := validator.ValidateEnvelope(context.Background(), envelope, "deploy-1", "workload-1", "prod")
	if err != nil {
		t.Fatalf("Valid envelope rejected: %v", err)
	}
	if string(plaintext) != "secret-plaintext" {
		t.Errorf("Plaintext mismatch: got %q, want %q", plaintext, "secret-plaintext")
	}
}

func TestSecretDeliveryValidator_InvalidSignature_Rejected(t *testing.T) {
	cpID := identity.FromSeed(seedFor("control-plane-2"))
	evilID := identity.FromSeed(seedFor("evil-identity"))
	nodeID := identity.FromSeed(seedFor("node-identity-2"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-123",
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
		Nonce:               []byte("nonce-data"),
		PlaintextPayload:    []byte("secret-plaintext"),
	}

	// Sign with wrong identity (evil)
	envelope.Signature = evilID.Sign(envelope.CanonicalEnvelope())

	// Validate should fail
	_, err := validator.ValidateEnvelope(context.Background(), envelope, "deploy-1", "workload-1", "prod")
	if err == nil {
		t.Fatal("Invalid signature was accepted")
	}
}

func TestSecretDeliveryValidator_ExpiredEnvelope_Rejected(t *testing.T) {
	cpID := identity.FromSeed(seedFor("control-plane-3"))
	nodeID := identity.FromSeed(seedFor("node-identity-3"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	now := time.Now().UnixNano()
	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-123",
		AuthorizationDigest: "digest",
		ClusterID:           "cluster-1",
		NodeID:              nodeID.ID,
		DeploymentID:        "deploy-1",
		WorkloadID:          "workload-1",
		Environment:         "prod",
		SecretID:            "secret-1",
		SecretVersion:       1,
		IssuedAt:            now - int64(10*time.Minute), // issued 10 minutes ago
		ExpiresAt:           now - int64(1*time.Minute),  // expired 1 minute ago
		Nonce:               []byte("nonce-data"),
		PlaintextPayload:    []byte("secret-plaintext"),
	}

	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	_, err := validator.ValidateEnvelope(context.Background(), envelope, "deploy-1", "workload-1", "prod")
	if err == nil {
		t.Fatal("Expired envelope was accepted")
	}
}

func TestSecretDeliveryValidator_WrongNode_Rejected(t *testing.T) {
	cpID := identity.FromSeed(seedFor("control-plane-3"))
	nodeID := identity.FromSeed(seedFor("node-identity-3"))
	otherNodeID := identity.FromSeed(seedFor("other-identity"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-123",
		AuthorizationDigest: "digest",
		ClusterID:           "cluster-1",
		NodeID:              otherNodeID.ID, // Wrong node ID
		DeploymentID:        "deploy-1",
		WorkloadID:          "workload-1",
		Environment:         "prod",
		SecretID:            "secret-1",
		SecretVersion:       1,
		IssuedAt:            time.Now().UnixNano(),
		ExpiresAt:           time.Now().Add(5 * time.Minute).UnixNano(),
		Nonce:               []byte("nonce-data"),
		PlaintextPayload:    []byte("secret-plaintext"),
	}

	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	_, err := validator.ValidateEnvelope(context.Background(), envelope, "deploy-1", "workload-1", "prod")
	if err == nil {
		t.Fatal("Envelope for wrong node was accepted")
	}
}

func TestSecretDeliveryValidator_WrongDeployment_Rejected(t *testing.T) {
	cpID := identity.FromSeed(seedFor("control-plane-3"))
	nodeID := identity.FromSeed(seedFor("node-identity-3"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-123",
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
		Nonce:               []byte("nonce-data"),
		PlaintextPayload:    []byte("secret-plaintext"),
	}

	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	// Validate with different deployment ID
	_, err := validator.ValidateEnvelope(context.Background(), envelope, "deploy-2", "workload-1", "prod")
	if err == nil {
		t.Fatal("Envelope for wrong deployment was accepted")
	}
}

func TestSecretDeliveryValidator_WrongWorkload_Rejected(t *testing.T) {
	cpID := identity.FromSeed(seedFor("control-plane-3"))
	nodeID := identity.FromSeed(seedFor("node-identity-3"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-123",
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
		Nonce:               []byte("nonce-data"),
		PlaintextPayload:    []byte("secret-plaintext"),
	}

	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	// Validate with different workload ID
	_, err := validator.ValidateEnvelope(context.Background(), envelope, "deploy-1", "workload-2", "prod")
	if err == nil {
		t.Fatal("Envelope for wrong workload was accepted")
	}
}

func TestSecretDeliveryValidator_WrongEnvironment_Rejected(t *testing.T) {
	cpID := identity.FromSeed(seedFor("control-plane-3"))
	nodeID := identity.FromSeed(seedFor("node-identity-3"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-123",
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
		Nonce:               []byte("nonce-data"),
		PlaintextPayload:    []byte("secret-plaintext"),
	}

	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	// Validate with different environment
	_, err := validator.ValidateEnvelope(context.Background(), envelope, "deploy-1", "workload-1", "staging")
	if err == nil {
		t.Fatal("Envelope for wrong environment was accepted")
	}
}

func TestSecretDeliveryValidator_WrongCluster_Rejected(t *testing.T) {
	cpID := identity.FromSeed(seedFor("control-plane-3"))
	nodeID := identity.FromSeed(seedFor("node-identity-3"))

	// Validator expects cluster-1
	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())

	// But envelope is for cluster-2
	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-123",
		AuthorizationDigest: "digest",
		ClusterID:           "cluster-2",
		NodeID:              nodeID.ID,
		DeploymentID:        "deploy-1",
		WorkloadID:          "workload-1",
		Environment:         "prod",
		SecretID:            "secret-1",
		SecretVersion:       1,
		IssuedAt:            time.Now().UnixNano(),
		ExpiresAt:           time.Now().Add(5 * time.Minute).UnixNano(),
		Nonce:               []byte("nonce-data"),
		PlaintextPayload:    []byte("secret-plaintext"),
	}

	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	_, err := validator.ValidateEnvelope(context.Background(), envelope, "deploy-1", "workload-1", "prod")
	if err == nil {
		t.Fatal("Envelope for wrong cluster was accepted")
	}
}

func TestSecretDeliveryReceiver_ValidEnvelope_Materializes(t *testing.T) {
	tmpdir := t.TempDir()
	cpID := identity.FromSeed(seedFor("control-plane-3"))
	nodeID := identity.FromSeed(seedFor("node-identity-3"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())
	materializer := NewMaterializer(tmpdir)
	receiver := NewSecretDeliveryReceiver(validator, materializer)

	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-123",
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
		Nonce:               []byte("nonce-data"),
		PlaintextPayload:    []byte("secret-plaintext"),
	}

	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	ctx := context.Background()
	hostPath, err := receiver.ReceiveAndMaterialize(ctx, envelope, "deploy-1", "workload-1", "prod", "assign-123")
	if err != nil {
		t.Fatalf("ReceiveAndMaterialize failed: %v", err)
	}

	// Verify file was created with correct content
	data, err := readFileSecurely(hostPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != "secret-plaintext" {
		t.Errorf("Content mismatch: got %q, want %q", data, "secret-plaintext")
	}
}

func TestSecretDeliveryReceiver_InvalidSignature_NoMaterialization(t *testing.T) {
	tmpdir := t.TempDir()
	cpID := identity.FromSeed(seedFor("control-plane-3"))
	evilID := identity.FromSeed(seedFor("evil-identity"))
	nodeID := identity.FromSeed(seedFor("node-identity-3"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())
	materializer := NewMaterializer(tmpdir)
	receiver := NewSecretDeliveryReceiver(validator, materializer)

	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-123",
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
		Nonce:               []byte("nonce-data"),
		PlaintextPayload:    []byte("secret-plaintext"),
	}

	// Sign with wrong identity
	envelope.Signature = evilID.Sign(envelope.CanonicalEnvelope())

	ctx := context.Background()
	_, err := receiver.ReceiveAndMaterialize(ctx, envelope, "deploy-1", "workload-1", "prod", "assign-123")
	if err == nil {
		t.Fatal("Invalid envelope was accepted")
	}

	// Verify no files were created in tmpdir
	entries, _ := readDirCount(tmpdir)
	if entries > 0 {
		t.Errorf("Plaintext was materialized despite failed validation")
	}
}

// Helper functions for testing

func readFileSecurely(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func readDirCount(path string) (int, error) {
	entries, err := os.ReadDir(path)
	return len(entries), err
}
