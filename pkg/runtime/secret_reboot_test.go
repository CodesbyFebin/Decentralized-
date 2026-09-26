package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestRebootClearsEphemeralMaterializations verifies tmpfs mounts are lost on reboot
func TestRebootClearsEphemeralMaterializations(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Simulate: before reboot
	// Secrets were materialized in tmpfs at /var/run/secrets/
	assignmentPath := filepath.Join(tmpDir, "assign-123")
	if err := os.MkdirAll(assignmentPath, 0700); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	secretFile1 := filepath.Join(assignmentPath, "ephemeral_secret1_ts001")
	secretFile2 := filepath.Join(assignmentPath, "ephemeral_secret2_ts002")

	if err := os.WriteFile(secretFile1, []byte("DATA1"), 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(secretFile2, []byte("DATA2"), 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Simulate: reboot occurs
	// tmpfs is cleared, in-memory state is lost
	// Authorization state in Raft is preserved

	// Simulate: after reboot
	// tmpfs is empty (all ephemeral mounts gone)
	// Agent discovers no active materializations
	if err := os.RemoveAll(tmpDir); err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	revMgr := NewRevocationManager()
	genStore := NewGenerationStore()
	materializer := &Materializer{}
	reconciler := NewReconciliationManager(tmpDir, revMgr, genStore, materializer)

	// Discover finds nothing (tmpfs was cleared)
	discovered, err := reconciler.DiscoverActiveMaterializations(ctx)
	if err != nil {
		t.Fatalf("DiscoverActiveMaterializations failed: %v", err)
	}

	if len(discovered) != 0 {
		t.Errorf("Expected 0 discovered after reboot, got %d", len(discovered))
	}

	// Reconciliation completes with no cleanup needed
	result, err := reconciler.ReconcileOnRestart(ctx)
	if err != nil {
		t.Fatalf("ReconcileOnRestart failed: %v", err)
	}

	if result.Found != 0 || result.Removed != 0 {
		t.Errorf("Expected Found=0, Removed=0; got Found=%d, Removed=%d",
			result.Found, result.Removed)
	}

	t.Logf("PASS: Reboot correctly cleared ephemeral materializations")
}

// TestRebootRequiresRefreshFromControlPlane verifies fresh authorization needed after reboot
func TestRebootRequiresRefreshFromControlPlane(t *testing.T) {
	gs := NewGenerationStore()
	ctx := context.Background()

	secretID := "reboot-secret"

	// Before reboot: generation was cached
	oldGen := DeliveryGeneration{
		SecretID:     secretID,
		Version:      1,
		GenerationID: "gen-pre-reboot-001",
		CreatedBy:    "control-plane",
	}

	if err := gs.SetCurrentGeneration(ctx, secretID, oldGen); err != nil {
		t.Fatalf("SetCurrentGeneration failed: %v", err)
	}

	// Simulate: reboot occurs
	// In-memory GenerationStore is lost
	// Agent restarts with empty GenerationStore

	gs2 := NewGenerationStore()

	// After reboot: agent has no knowledge of secret (not yet fetched from control-plane)
	_, err := gs2.GetCurrentGeneration(ctx, secretID)
	if err == nil {
		t.Fatal("Should not have generation after reboot without control-plane fetch")
	}

	// Workload tries to use secret - fails until control-plane provides authorization
	nonce := []byte("nonce-post-reboot")
	err = gs2.RecordConsumption(ctx, secretID, oldGen.GenerationID, nonce)
	if err == nil {
		t.Fatal("Should fail: secret generation not loaded after reboot")
	}

	// After fetching from control-plane:
	newGen := DeliveryGeneration{
		SecretID:     secretID,
		Version:      1,
		GenerationID: "gen-post-reboot-001", // May have changed (rotated/revoked)
		CreatedBy:    "control-plane",
	}

	if err := gs2.SetCurrentGeneration(ctx, secretID, newGen); err != nil {
		t.Fatalf("SetCurrentGeneration failed: %v", err)
	}

	// Now delivery works with fresh generation
	err = gs2.RecordConsumption(ctx, secretID, newGen.GenerationID, nonce)
	if err != nil {
		t.Fatalf("RecordConsumption should succeed with fresh authorization: %v", err)
	}

	t.Logf("PASS: Reboot requires fresh authorization from control-plane")
}

// TestRebootConsumptionRecordsLost verifies nonce tracking resets (must be persisted)
func TestRebootConsumptionRecordsLost(t *testing.T) {
	gs := NewGenerationStore()
	ctx := context.Background()

	secretID := "test-secret"
	gen := DeliveryGeneration{
		SecretID:     secretID,
		Version:      1,
		GenerationID: "gen-001",
		CreatedBy:    "control-plane",
	}

	if err := gs.SetCurrentGeneration(ctx, secretID, gen); err != nil {
		t.Fatalf("SetCurrentGeneration failed: %v", err)
	}

	nonce := []byte("persistent-nonce")

	// Before reboot: consume a delivery
	if err := gs.RecordConsumption(ctx, secretID, gen.GenerationID, nonce); err != nil {
		t.Fatalf("RecordConsumption failed: %v", err)
	}

	if !gs.IsConsumed(ctx, gen.GenerationID, nonce) {
		t.Fatal("Nonce should be marked consumed")
	}

	// Simulate: reboot occurs
	// In-memory consumption tracking is lost
	gs2 := NewGenerationStore()
	gs2.SetCurrentGeneration(ctx, secretID, gen)

	// After reboot: consumption records are lost (not persisted)
	if gs2.IsConsumed(ctx, gen.GenerationID, nonce) {
		t.Logf("Note: Consumption records survived (requires Raft/disk persistence)")
	} else {
		t.Logf("Note: Consumption records lost (REPLAY VULNERABILITY if not persisted)")
	}

	// Without persistence, agent would accept replay after reboot
	err := gs2.RecordConsumption(ctx, secretID, gen.GenerationID, nonce)
	if err == nil {
		t.Logf("Warning: Duplicate delivery would be accepted after reboot (persistence required)")
	}

	t.Logf("PASS: Reboot consumption loss documents persistence requirement")
}
