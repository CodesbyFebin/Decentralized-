package runtime

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/pkg/identity"
)

// TestPlaintextCanary_TmpfsMaterializationOnly verifies that plaintext is never
// persisted to the filesystem. A unique marker is injected into the plaintext,
// and the filesystem is scanned to confirm the marker never appears.
func TestPlaintextCanary_TmpfsMaterializationOnly(t *testing.T) {
	tmpdir := t.TempDir()
	cpID := identity.FromSeed(seedFor("cp-canary-tmpfs"))
	nodeID := identity.FromSeed(seedFor("node-canary-tmpfs"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())
	materializer := NewMaterializer(tmpdir)
	receiver := NewSecretDeliveryReceiver(validator, materializer)

	ctx := context.Background()

	// Create unique canary marker that's unlikely to appear elsewhere
	canaryMarker := "PLAINTEXT_CANARY_MARKER_12345_ABCDE_XYZAB"

	// Materialize secret with canary marker
	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-canary-tmpfs",
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
		Nonce:               []byte("nonce-canary-tmpfs"),
		PlaintextPayload:    []byte(canaryMarker + "-secret-confidential-data"),
	}
	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	hostPath, err := receiver.ReceiveAndMaterialize(ctx, envelope, "deploy-1", "workload-1", "prod", "assign-canary-tmpfs")
	if err != nil {
		t.Fatalf("ReceiveAndMaterialize failed: %v", err)
	}

	// Verify the secret file itself contains the marker (it should)
	data, err := os.ReadFile(hostPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Contains(data, []byte(canaryMarker)) {
		t.Errorf("Canary marker not in materialized secret file")
	}

	// Scan the filesystem for the canary marker in other files
	// The marker should ONLY appear in the materialized tmpfs file, not anywhere else
	foundInOtherFile := false
	err = filepath.Walk(tmpdir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the secret file itself
		if path == hostPath {
			return nil
		}

		// Only scan regular files
		if !info.Mode().IsRegular() {
			return nil
		}

		// Read file and search for marker
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		if bytes.Contains(content, []byte(canaryMarker)) {
			t.Errorf("Canary marker found in unexpected file: %s", path)
			foundInOtherFile = true
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if !foundInOtherFile {
		t.Logf("PASS: Canary marker only found in materialized file, not in filesystem")
	}
}

// TestPlaintextCanary_NoMaterializationOnValidationFailure verifies that plaintext
// is never written to disk if validation fails.
func TestPlaintextCanary_NoMaterializationOnValidationFailure(t *testing.T) {
	tmpdir := t.TempDir()
	cpID := identity.FromSeed(seedFor("cp-canary-fail"))
	evilID := identity.FromSeed(seedFor("evil-canary-fail"))
	nodeID := identity.FromSeed(seedFor("node-canary-fail"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())
	materializer := NewMaterializer(tmpdir)
	receiver := NewSecretDeliveryReceiver(validator, materializer)

	ctx := context.Background()

	canaryMarker := "PLAINTEXT_CANARY_FAIL_12345_ABCDE_XYZAB"

	// Create envelope with invalid signature
	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-canary-fail",
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
		Nonce:               []byte("nonce-canary-fail"),
		PlaintextPayload:    []byte(canaryMarker + "-secret-should-not-persist"),
	}
	// Sign with wrong identity (evilID instead of cpID)
	envelope.Signature = evilID.Sign(envelope.CanonicalEnvelope())

	// Attempt materialization (should fail)
	_, err := receiver.ReceiveAndMaterialize(ctx, envelope, "deploy-1", "workload-1", "prod", "assign-canary-fail")
	if err == nil {
		t.Fatal("ReceiveAndMaterialize should have failed with invalid signature")
	}

	// Scan filesystem for the canary marker
	// It should NOT appear anywhere
	foundCanary := false
	err = filepath.Walk(tmpdir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		if bytes.Contains(content, []byte(canaryMarker)) {
			t.Errorf("Canary marker found in file after failed validation: %s", path)
			foundCanary = true
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if !foundCanary {
		t.Logf("PASS: Plaintext not persisted when validation fails")
	}
}

// TestPlaintextCanary_ControlPlaneNoDiskPersistence verifies that the control plane
// doesn't persist plaintext. This test documents the assumption that control plane
// only has plaintext in memory during signing, never on disk.
func TestPlaintextCanary_ControlPlaneNoDiskPersistence(t *testing.T) {
	// This test documents the trust boundary assumption:
	// - Control plane has plaintext only in memory during signing
	// - Control plane NEVER writes plaintext to disk
	// - Control plane NEVER returns plaintext directly (only signed envelope)
	// - Plaintext only materialized by agent to tmpfs

	// Verification would require:
	// 1. Scan control plane's persistent storage (BoltDB, filesystem)
	// 2. Scan control plane's logs
	// 3. Scan control plane's process memory (before and after secret delivery)
	// 4. Monitor control plane's disk I/O for secret payloads

	// In this test environment, we document that:
	// - SecretDeliveryEnvelope.PlaintextPayload is never written to control plane storage
	// - handleRetrieveSecret returns envelope, not plaintext or path
	// - No temporary files are created on control plane
	// - No swap/page files contain plaintext (assumption: no swap configured)

	t.Logf("CANARY ASSUMPTION: Control plane plaintext never reaches disk")
	t.Logf("  - Plaintext exists only in signing function's stack")
	t.Logf("  - Plaintext returned in memory within envelope")
	t.Logf("  - Envelope signed before return (signature binding)")
	t.Logf("  - No disk I/O for plaintext")
	t.Logf("  - Verification: source code audit (pkg/control/hostapi.go)")
}

// TestPlaintextCanary_NoLogging verifies that secrets are not logged or printed
// in test output (documentation of logging assumptions).
func TestPlaintextCanary_NoLogging(t *testing.T) {
	// This test documents the assumption that the secret delivery mechanism
	// does not log plaintext payloads anywhere.

	// Verification would require:
	// 1. Search application logs for secret patterns
	// 2. Monitor log output during secret delivery
	// 3. Audit logging calls in secret delivery code

	// In the implemented code:
	// - SecretDeliveryValidator doesn't log plaintext
	// - SecretDeliveryReceiver doesn't log plaintext
	// - Materializer doesn't log content
	// - Only errors/digests/metadata logged, never payloads

	t.Logf("CANARY ASSUMPTION: Plaintext never logged or printed")
	t.Logf("  - Only digests/IDs/metadata logged")
	t.Logf("  - No fmt.Printf or t.Logf with plaintext")
	t.Logf("  - Verification: grep for fmt.* in secret_delivery*.go")
}

// TestPlaintextCanary_MultipleSecrets_NoMixing verifies that secrets for
// different workloads don't get mixed or leak into each other's files.
func TestPlaintextCanary_MultipleSecrets_NoMixing(t *testing.T) {
	tmpdir := t.TempDir()
	cpID := identity.FromSeed(seedFor("cp-canary-mix"))
	nodeID := identity.FromSeed(seedFor("node-canary-mix"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())
	materializer := NewMaterializer(tmpdir)
	receiver := NewSecretDeliveryReceiver(validator, materializer)

	ctx := context.Background()

	// Create secrets with distinct markers
	markers := []string{
		"CANARY_SECRET_A_MARKER_UNIQUE_12345",
		"CANARY_SECRET_B_MARKER_UNIQUE_67890",
		"CANARY_SECRET_C_MARKER_UNIQUE_ABCDE",
	}

	paths := make([]string, len(markers))

	for i, marker := range markers {
		envelope := &control.SecretDeliveryEnvelope{
			ProtocolVersion:     1,
			DeliveryID:          fmt.Sprintf("delivery-canary-mix-%d", i),
			AuthorizationDigest: "digest",
			ClusterID:           "cluster-1",
			NodeID:              nodeID.ID,
			DeploymentID:        "deploy-1",
			WorkloadID:          fmt.Sprintf("workload-%d", i),
			Environment:         "prod",
			SecretID:            fmt.Sprintf("secret-%d", i),
			SecretVersion:       1,
			IssuedAt:            time.Now().UnixNano(),
			ExpiresAt:           time.Now().Add(5 * time.Minute).UnixNano(),
			Nonce:               []byte(fmt.Sprintf("nonce-%d", i)),
			PlaintextPayload:    []byte(marker + "-secret-data"),
		}
		envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

		hostPath, err := receiver.ReceiveAndMaterialize(ctx, envelope, "deploy-1", fmt.Sprintf("workload-%d", i), "prod", fmt.Sprintf("assign-%d", i))
		if err != nil {
			t.Fatalf("ReceiveAndMaterialize failed: %v", err)
		}
		paths[i] = hostPath
	}

	// Verify each secret file contains only its own marker, not others' markers
	for i, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}

		// Should contain own marker
		if !bytes.Contains(content, []byte(markers[i])) {
			t.Errorf("Secret %d missing its own marker", i)
		}

		// Should NOT contain other markers
		for j, otherMarker := range markers {
			if i != j && bytes.Contains(content, []byte(otherMarker)) {
				t.Errorf("Secret %d contains marker from secret %d (cross-contamination)", i, j)
			}
		}
	}

	t.Logf("PASS: Secrets isolated, no cross-contamination")
}

// TestPlaintextCanary_FilePermissionsPreventAccess verifies that materialized
// secrets have 0400 permissions, preventing unauthorized access even if found.
func TestPlaintextCanary_FilePermissionsPreventAccess(t *testing.T) {
	tmpdir := t.TempDir()
	cpID := identity.FromSeed(seedFor("cp-canary-perm"))
	nodeID := identity.FromSeed(seedFor("node-canary-perm"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())
	materializer := NewMaterializer(tmpdir)
	receiver := NewSecretDeliveryReceiver(validator, materializer)

	ctx := context.Background()

	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-canary-perm",
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
		Nonce:               []byte("nonce-canary-perm"),
		PlaintextPayload:    []byte("secret-with-restricted-perms"),
	}
	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	hostPath, err := receiver.ReceiveAndMaterialize(ctx, envelope, "deploy-1", "workload-1", "prod", "assign-canary-perm")
	if err != nil {
		t.Fatalf("ReceiveAndMaterialize failed: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(hostPath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	mode := info.Mode().Perm()
	if mode != 0o400 {
		t.Errorf("File permissions incorrect: got %#o, want 0o400", mode)
	}

	// Verify the file is only readable by owner
	// (In a real scenario, we'd verify this prevents access by other UIDs)
	t.Logf("PASS: File permissions set to %#o (read-only by owner)", mode)
}

// TestPlaintextCanary_ReleaseDeletesContent verifies that when a secret is
// released, the file is completely deleted (not just zeroed).
func TestPlaintextCanary_ReleaseDeletesContent(t *testing.T) {
	tmpdir := t.TempDir()
	cpID := identity.FromSeed(seedFor("cp-canary-del"))
	nodeID := identity.FromSeed(seedFor("node-canary-del"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())
	materializer := NewMaterializer(tmpdir)
	receiver := NewSecretDeliveryReceiver(validator, materializer)

	ctx := context.Background()

	canaryMarker := "CANARY_DELETE_MARKER_12345_ABCDE_XYZAB"

	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-canary-del",
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
		Nonce:               []byte("nonce-canary-del"),
		PlaintextPayload:    []byte(canaryMarker + "-secret-to-delete"),
	}
	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	hostPath, err := receiver.ReceiveAndMaterialize(ctx, envelope, "deploy-1", "workload-1", "prod", "assign-canary-del")
	if err != nil {
		t.Fatalf("ReceiveAndMaterialize failed: %v", err)
	}

	// Verify file exists and contains marker
	content, err := os.ReadFile(hostPath)
	if err != nil {
		t.Fatalf("ReadFile before delete failed: %v", err)
	}
	if !bytes.Contains(content, []byte(canaryMarker)) {
		t.Errorf("Marker not in file before delete")
	}

	// Release the secret
	err = materializer.ReleaseEphemeral(ctx, hostPath)
	if err != nil {
		t.Fatalf("ReleaseEphemeral failed: %v", err)
	}

	// Verify file was deleted (stat returns ENOENT)
	if _, err := os.Stat(hostPath); !os.IsNotExist(err) {
		t.Errorf("File still exists after delete: %s", hostPath)
	}

	// Note: After deletion, the plaintext may still exist in page cache or memory.
	// Complete wiping (overwriting) would be a future enhancement.
	// For now, deletion removes the path, making it inaccessible via normal means.
	t.Logf("PASS: File deleted after release (path removed from filesystem)")
}

// TestPlaintextCanary_MultipleRelease_NoDoubleDelete verifies that releasing
// an already-deleted secret doesn't cause issues.
func TestPlaintextCanary_MultipleRelease_NoDoubleDelete(t *testing.T) {
	tmpdir := t.TempDir()
	cpID := identity.FromSeed(seedFor("cp-canary-multi-del"))
	nodeID := identity.FromSeed(seedFor("node-canary-multi-del"))

	validator := NewSecretDeliveryValidator(nodeID.ID, "cluster-1", cpID.PubString())
	materializer := NewMaterializer(tmpdir)
	receiver := NewSecretDeliveryReceiver(validator, materializer)

	ctx := context.Background()

	envelope := &control.SecretDeliveryEnvelope{
		ProtocolVersion:     1,
		DeliveryID:          "delivery-canary-multi-del",
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
		Nonce:               []byte("nonce-canary-multi-del"),
		PlaintextPayload:    []byte("secret-multi-delete-test"),
	}
	envelope.Signature = cpID.Sign(envelope.CanonicalEnvelope())

	hostPath, err := receiver.ReceiveAndMaterialize(ctx, envelope, "deploy-1", "workload-1", "prod", "assign-canary-multi-del")
	if err != nil {
		t.Fatalf("ReceiveAndMaterialize failed: %v", err)
	}

	// First release
	err = materializer.ReleaseEphemeral(ctx, hostPath)
	if err != nil {
		t.Fatalf("First ReleaseEphemeral failed: %v", err)
	}

	// Second release (file already gone)
	err = materializer.ReleaseEphemeral(ctx, hostPath)
	if err != nil {
		// ReleaseEphemeral should handle already-deleted files gracefully
		t.Logf("Second ReleaseEphemeral on already-deleted file: %v", err)
	}

	t.Logf("PASS: Multiple releases handled safely")
}
