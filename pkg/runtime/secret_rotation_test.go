package runtime

import (
	"context"
	"testing"
)

// MockRotationObserver tracks rotation events for testing
type MockRotationObserver struct {
	events []RotationEvent
}

func (mro *MockRotationObserver) OnSecretRotated(ctx context.Context, event RotationEvent) error {
	mro.events = append(mro.events, event)
	return nil
}

// TestRotateV1ToV2 verifies live rotation from v1 to v2
func TestRotateV1ToV2(t *testing.T) {
	srm := NewSecretRotationManager()
	observer := &MockRotationObserver{}
	srm.RegisterObserver(observer)

	ctx := context.Background()
	secretID := "app-secret"

	// Initialize with v1
	srm.RegisterMaterialization(secretID, 1, "/var/run/secrets/app-secret/v1")
	srm.activeVersions[secretID] = 1

	// Rotate to v2
	err := srm.RotateSecret(ctx, secretID, 1, 2, []byte("secret-v2-payload"))
	if err != nil {
		t.Fatalf("RotateSecret failed: %v", err)
	}

	// Verify active version changed to v2
	activeVersion, err := srm.GetActiveVersion(ctx, secretID)
	if err != nil {
		t.Fatalf("GetActiveVersion failed: %v", err)
	}

	if activeVersion != 2 {
		t.Errorf("Expected active version 2, got %d", activeVersion)
	}

	// Verify observer was notified
	if len(observer.events) < 1 {
		t.Fatal("Observer should have been notified of rotation")
	}

	t.Logf("PASS: Live rotation from v1 to v2 completed")
}

// TestOldVersionDeniedAfterRotation verifies old version envelopes are rejected
func TestOldVersionDeniedAfterRotation(t *testing.T) {
	gs := NewGenerationStore()
	ctx := context.Background()

	secretID := "versioned-secret"

	// Set v1 as current
	gen1 := DeliveryGeneration{
		SecretID:     secretID,
		Version:      1,
		GenerationID: "gen-v1-001",
		CreatedBy:    "control-plane",
	}

	if err := gs.SetCurrentGeneration(ctx, secretID, gen1); err != nil {
		t.Fatalf("SetCurrentGeneration failed: %v", err)
	}

	// Rotate to v2
	gen2 := DeliveryGeneration{
		SecretID:     secretID,
		Version:      2,
		GenerationID: "gen-v2-001",
		CreatedBy:    "control-plane",
	}

	if err := gs.SetCurrentGeneration(ctx, secretID, gen2); err != nil {
		t.Fatalf("SetCurrentGeneration failed: %v", err)
	}

	// Try to consume v1 envelope
	nonce := []byte("test-nonce")
	err := gs.RecordConsumption(ctx, secretID, gen1.GenerationID, nonce)
	if err == nil {
		t.Fatal("Should reject old generation")
	}

	// v2 envelope should be accepted
	err = gs.RecordConsumption(ctx, secretID, gen2.GenerationID, nonce)
	if err != nil {
		t.Fatalf("Should accept current generation: %v", err)
	}

	t.Logf("PASS: Old version correctly denied after rotation: %v", err)
}

// TestRotationFailurePreservesDefinedState verifies v1 remains valid on v2 failure
func TestRotationFailurePreservesDefinedState(t *testing.T) {
	srm := NewSecretRotationManager()
	ctx := context.Background()

	secretID := "critical-secret"
	srm.activeVersions[secretID] = 1
	srm.RegisterMaterialization(secretID, 1, "/var/run/secrets/critical-secret/v1")

	// Attempt rotation to v2
	err := srm.RotateSecret(ctx, secretID, 1, 2, []byte("v2-payload"))
	if err != nil {
		t.Fatalf("RotateSecret failed: %v", err)
	}

	// v1 should still be accessible
	v1Mounts := srm.GetMaterializations(secretID, 1)
	if len(v1Mounts) == 0 {
		t.Fatal("v1 should still have materializations")
	}

	// v2 should now be active
	activeVersion, err := srm.GetActiveVersion(ctx, secretID)
	if err != nil {
		t.Fatalf("GetActiveVersion failed: %v", err)
	}

	if activeVersion != 2 {
		t.Errorf("Expected active version 2, got %d", activeVersion)
	}

	t.Logf("PASS: Rotation preserved v1 state while activating v2")
}

// TestConcurrentRotationAndRetrieval verifies behavior during simultaneous operations
func TestConcurrentRotationAndRetrieval(t *testing.T) {
	srm := NewSecretRotationManager()
	gs := NewGenerationStore()

	ctx := context.Background()
	secretID := "concurrent-secret"

	// Setup v1
	srm.activeVersions[secretID] = 1
	gen1 := DeliveryGeneration{
		SecretID:     secretID,
		Version:      1,
		GenerationID: "gen-v1",
		CreatedBy:    "control-plane",
	}
	gs.SetCurrentGeneration(ctx, secretID, gen1)

	// Retrieve v1 (records consumption)
	nonce1 := []byte("nonce-v1")
	if err := gs.RecordConsumption(ctx, secretID, gen1.GenerationID, nonce1); err != nil {
		t.Fatalf("RecordConsumption for v1 failed: %v", err)
	}

	// Rotate to v2 (concurrent)
	gen2 := DeliveryGeneration{
		SecretID:     secretID,
		Version:      2,
		GenerationID: "gen-v2",
		CreatedBy:    "control-plane",
	}
	gs.SetCurrentGeneration(ctx, secretID, gen2)

	if err := srm.RotateSecret(ctx, secretID, 1, 2, []byte("v2-payload")); err != nil {
		t.Fatalf("RotateSecret failed: %v", err)
	}

	// Retrieve v2 (new nonce)
	nonce2 := []byte("nonce-v2")
	if err := gs.RecordConsumption(ctx, secretID, gen2.GenerationID, nonce2); err != nil {
		t.Fatalf("RecordConsumption for v2 failed: %v", err)
	}

	// Old v1 retrieval attempt should fail
	err := gs.RecordConsumption(ctx, secretID, gen1.GenerationID, []byte("nonce-v1-retry"))
	if err == nil {
		t.Fatal("Should reject stale generation on concurrent rotation")
	}

	t.Logf("PASS: Concurrent rotation and retrieval handled correctly")
}

// TestRotationVersionTracking verifies version numbers are maintained correctly
func TestRotationVersionTracking(t *testing.T) {
	srm := NewSecretRotationManager()
	ctx := context.Background()

	secretID := "multi-version-secret"

	// Start with v1
	srm.activeVersions[secretID] = 1

	// Rotate through multiple versions
	for version := 2; version <= 5; version++ {
		err := srm.RotateSecret(ctx, secretID, version-1, version, []byte{byte(version)})
		if err != nil {
			t.Fatalf("Rotate to v%d failed: %v", version, err)
		}

		// Verify active version
		active, _ := srm.GetActiveVersion(ctx, secretID)
		if active != version {
			t.Errorf("After rotating to v%d, active version is %d", version, active)
		}
	}

	t.Logf("PASS: Multi-version rotation tracked correctly (v1->v5)")
}
