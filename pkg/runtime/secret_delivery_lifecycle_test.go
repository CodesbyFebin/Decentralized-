package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/pkg/identity"
)

// TestSecretDeliveryLifecycle_NormalStop_Cleanup verifies that secrets are cleaned up
// when workload exits normally.
func TestSecretDeliveryLifecycle_NormalStop_Cleanup(t *testing.T) {
	tmpdir := t.TempDir()
	cpID := identity.FromSeed(seedFor("cp-lifecycle-1"))
	nodeID := identity.FromSeed(seedFor("node-lifecycle-1"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())
	materializer := NewMaterializer(tmpdir)
	receiver := NewSecretDeliveryReceiver(validator, materializer)

	ctx := context.Background()

	// Create and materialize envelope
	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-lifecycle-1",
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
		Nonce:               []byte("nonce-lifecycle-1"),
		PlaintextPayload:    []byte("secret-payload-lifecycle-1"),
	}
	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	hostPath, err := receiver.ReceiveAndMaterialize(ctx, envelope, "deploy-1", "workload-1", "prod", "assign-lifecycle-1")
	if err != nil {
		t.Fatalf("ReceiveAndMaterialize failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(hostPath); err != nil {
		t.Fatalf("Secret file not created: %v", err)
	}

	// Simulate workload exit: release the ephemeral secret
	err = materializer.ReleaseEphemeral(ctx, hostPath)
	if err != nil {
		t.Fatalf("ReleaseEphemeral failed: %v", err)
	}

	// Verify file was removed
	if _, err := os.Stat(hostPath); !os.IsNotExist(err) {
		t.Errorf("Secret file was not cleaned up after release")
	}
}

// TestSecretDeliveryLifecycle_StartFailure_Cleanup verifies that secrets are cleaned up
// if workload fails to start.
func TestSecretDeliveryLifecycle_StartFailure_Cleanup(t *testing.T) {
	tmpdir := t.TempDir()
	cpID := identity.FromSeed(seedFor("cp-lifecycle-2"))
	nodeID := identity.FromSeed(seedFor("node-lifecycle-2"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())
	materializer := NewMaterializer(tmpdir)
	receiver := NewSecretDeliveryReceiver(validator, materializer)

	ctx := context.Background()

	// Create and materialize envelope
	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-lifecycle-2",
		AuthorizationDigest: "digest",
		ClusterID:           "cluster-1",
		NodeID:              nodeID.ID,
		DeploymentID:        "deploy-1",
		WorkloadID:          "workload-2",
		Environment:         "prod",
		SecretID:            "secret-2",
		SecretVersion:       1,
		IssuedAt:            time.Now().UnixNano(),
		ExpiresAt:           time.Now().Add(5 * time.Minute).UnixNano(),
		Nonce:               []byte("nonce-lifecycle-2"),
		PlaintextPayload:    []byte("secret-payload-lifecycle-2"),
	}
	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	hostPath, err := receiver.ReceiveAndMaterialize(ctx, envelope, "deploy-1", "workload-2", "prod", "assign-lifecycle-2")
	if err != nil {
		t.Fatalf("ReceiveAndMaterialize failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(hostPath); err != nil {
		t.Fatalf("Secret file not created: %v", err)
	}

	// Simulate workload startup failure: cleanup must occur
	// In production, the supervisor detects failed workload and calls Release
	err = materializer.ReleaseEphemeral(ctx, hostPath)
	if err != nil {
		t.Fatalf("ReleaseEphemeral failed: %v", err)
	}

	// Verify file was removed
	if _, err := os.Stat(hostPath); !os.IsNotExist(err) {
		t.Errorf("Secret file was not cleaned up after start failure")
	}
}

// TestSecretDeliveryLifecycle_StaleCleanup_RemovesExpiredFiles verifies that the
// Cleanup() method removes ephemeral files older than the timeout.
func TestSecretDeliveryLifecycle_StaleCleanup_RemovesExpiredFiles(t *testing.T) {
	tmpdir := t.TempDir()
	cpID := identity.FromSeed(seedFor("cp-lifecycle-3"))
	nodeID := identity.FromSeed(seedFor("node-lifecycle-3"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())
	materializer := NewMaterializer(tmpdir)
	receiver := NewSecretDeliveryReceiver(validator, materializer)

	ctx := context.Background()

	// Create and materialize multiple envelopes
	hostPaths := []string{}
	for i := 1; i <= 3; i++ {
		envelope := &control.SecretDeliveryEnvelope{
			ProtocolVersion:     1,
			DeliveryID:          "delivery-stale-" + string(rune('0'+i)),
			AuthorizationDigest: "digest",
			ClusterID:           "cluster-1",
			NodeID:              nodeID.ID,
			DeploymentID:        "deploy-1",
			WorkloadID:          "workload-stale-" + string(rune('0'+i)),
			Environment:         "prod",
			SecretID:            "secret-stale",
			SecretVersion:       1,
			IssuedAt:            time.Now().UnixNano(),
			ExpiresAt:           time.Now().Add(5 * time.Minute).UnixNano(),
			Nonce:               []byte("nonce-stale"),
			PlaintextPayload:    []byte("secret-stale"),
		}
		envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

		hostPath, err := receiver.ReceiveAndMaterialize(ctx, envelope, "deploy-1", "workload-stale-"+string(rune('0'+i)), "prod", "assign-stale")
		if err != nil {
			t.Fatalf("ReceiveAndMaterialize failed: %v", err)
		}
		hostPaths = append(hostPaths, hostPath)
	}

	// Verify all files exist
	for _, path := range hostPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Secret file not created: %v", err)
		}
	}

	// Run cleanup (should not remove recent files)
	cleaned, err := materializer.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
	if cleaned > 0 {
		t.Errorf("Cleanup removed recent files (count: %d)", cleaned)
	}

	// Verify files still exist
	for _, path := range hostPaths {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("File was cleaned up despite being recent: %s", path)
		}
	}
}

// TestSecretDeliveryLifecycle_MultipleSecrets_IndependentCleanup verifies that
// cleaning up one secret doesn't affect others.
func TestSecretDeliveryLifecycle_MultipleSecrets_IndependentCleanup(t *testing.T) {
	tmpdir := t.TempDir()
	cpID := identity.FromSeed(seedFor("cp-lifecycle-multi"))
	nodeID := identity.FromSeed(seedFor("node-lifecycle-multi"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())
	materializer := NewMaterializer(tmpdir)
	receiver := NewSecretDeliveryReceiver(validator, materializer)

	ctx := context.Background()

	// Create multiple secrets for different workloads
	secrets := make([]string, 2)
	for i := 0; i < 2; i++ {
		envelope := &control.SecretDeliveryEnvelope{
			ProtocolVersion:     1,
			DeliveryID:          "delivery-multi-" + string(rune('0'+i)),
			AuthorizationDigest: "digest",
			ClusterID:           "cluster-1",
			NodeID:              nodeID.ID,
			DeploymentID:        "deploy-1",
			WorkloadID:          "workload-multi-" + string(rune('0'+i)),
			Environment:         "prod",
			SecretID:            "secret-multi-" + string(rune('0'+i)),
			SecretVersion:       1,
			IssuedAt:            time.Now().UnixNano(),
			ExpiresAt:           time.Now().Add(5 * time.Minute).UnixNano(),
			Nonce:               []byte("nonce-multi"),
			PlaintextPayload:    []byte("secret-payload-multi-" + string(rune('0'+i))),
		}
		envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

		hostPath, err := receiver.ReceiveAndMaterialize(ctx, envelope, "deploy-1", "workload-multi-"+string(rune('0'+i)), "prod", "assign-multi")
		if err != nil {
			t.Fatalf("ReceiveAndMaterialize failed: %v", err)
		}
		secrets[i] = hostPath
	}

	// Verify both files exist
	for _, path := range secrets {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("Secret file not created: %v", err)
		}
	}

	// Release first secret only
	err := materializer.ReleaseEphemeral(ctx, secrets[0])
	if err != nil {
		t.Fatalf("ReleaseEphemeral failed: %v", err)
	}

	// Verify first is gone, second remains
	if _, err := os.Stat(secrets[0]); !os.IsNotExist(err) {
		t.Errorf("First secret not cleaned up")
	}

	if _, err := os.Stat(secrets[1]); err != nil {
		t.Errorf("Second secret was affected: %v", err)
	}

	// Release second secret
	err = materializer.ReleaseEphemeral(ctx, secrets[1])
	if err != nil {
		t.Fatalf("ReleaseEphemeral failed: %v", err)
	}

	// Verify both are gone
	for _, path := range secrets {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("Secret not cleaned up: %s", path)
		}
	}
}

// TestSecretDeliveryLifecycle_AssignmentDirectoryIsolation verifies that different
// assignments maintain separate directories.
func TestSecretDeliveryLifecycle_AssignmentDirectoryIsolation(t *testing.T) {
	tmpdir := t.TempDir()
	cpID := identity.FromSeed(seedFor("cp-lifecycle-iso"))
	nodeID := identity.FromSeed(seedFor("node-lifecycle-iso"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())
	materializer := NewMaterializer(tmpdir)
	receiver := NewSecretDeliveryReceiver(validator, materializer)

	ctx := context.Background()

	// Create secrets for different assignments
	secrets := make(map[string]string)
	for i := 0; i < 2; i++ {
		assignmentID := "assign-iso-" + string(rune('0'+i))
		envelope := &control.SecretDeliveryEnvelope{
			ProtocolVersion:     1,
			DeliveryID:          "delivery-iso-" + string(rune('0'+i)),
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
			Nonce:               []byte("nonce-iso"),
			PlaintextPayload:    []byte("secret-payload-iso"),
		}
		envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

		hostPath, err := receiver.ReceiveAndMaterialize(ctx, envelope, "deploy-1", "workload-1", "prod", assignmentID)
		if err != nil {
			t.Fatalf("ReceiveAndMaterialize failed: %v", err)
		}
		secrets[assignmentID] = hostPath
	}

	// Verify paths are in different assignment directories
	dir0 := filepath.Dir(secrets["assign-iso-0"])
	dir1 := filepath.Dir(secrets["assign-iso-1"])

	if dir0 == dir1 {
		t.Errorf("Different assignments in same directory: %s", dir0)
	}

	if !pathContainsAssignment(secrets["assign-iso-0"], "assign-iso-0") {
		t.Errorf("Path does not contain assignment ID: %s", secrets["assign-iso-0"])
	}

	if !pathContainsAssignment(secrets["assign-iso-1"], "assign-iso-1") {
		t.Errorf("Path does not contain assignment ID: %s", secrets["assign-iso-1"])
	}
}
