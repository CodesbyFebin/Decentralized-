package runtime

import (
	"context"
	"testing"
)

// TestLeaseValidationDuringPartition verifies cached generation can be used during partition
func TestLeaseValidationDuringPartition(t *testing.T) {
	gs := NewGenerationStore()
	ctx := context.Background()

	secretID := "partitioned-secret"
	gen := DeliveryGeneration{
		SecretID:     secretID,
		Version:      1,
		GenerationID: "gen-001",
		CreatedBy:    "control-plane",
	}

	// Before partition: authorization established
	if err := gs.SetCurrentGeneration(ctx, secretID, gen); err != nil {
		t.Fatalf("SetCurrentGeneration failed: %v", err)
	}

	// During partition: agent is disconnected from control-plane
	// Cached generation can be used if it's still valid (matches current)
	if !gs.IsCachedGenerationValid(ctx, secretID, gen) {
		t.Fatal("Cached generation should be valid if no rotation/revocation occurred")
	}

	// Agent can deliver using cached generation
	nonce := []byte("nonce-during-partition")
	if err := gs.RecordConsumption(ctx, secretID, gen.GenerationID, nonce); err != nil {
		t.Fatalf("RecordConsumption during partition failed: %v", err)
	}

	t.Logf("PASS: Cached generation valid during partition for lease-based use")
}

// TestGenerationInvalidatedAfterRotationDuringPartition verifies rotation invalidates cache
func TestGenerationInvalidatedAfterRotationDuringPartition(t *testing.T) {
	gs := NewGenerationStore()
	ctx := context.Background()

	secretID := "rotated-during-partition"

	// Before partition: gen1 active
	gen1 := DeliveryGeneration{
		SecretID:     secretID,
		Version:      1,
		GenerationID: "gen-v1-001",
		CreatedBy:    "control-plane",
	}

	if err := gs.SetCurrentGeneration(ctx, secretID, gen1); err != nil {
		t.Fatalf("SetCurrentGeneration failed: %v", err)
	}

	// Partition begins: agent is disconnected
	// While agent is disconnected, rotation occurs on control-plane
	gen2 := DeliveryGeneration{
		SecretID:     secretID,
		Version:      2,
		GenerationID: "gen-v2-001",
		CreatedBy:    "control-plane",
	}

	// Control-plane rotates (this update reaches agent when reconnecting)
	if err := gs.SetCurrentGeneration(ctx, secretID, gen2); err != nil {
		t.Fatalf("SetCurrentGeneration for gen2 failed: %v", err)
	}

	// After reconnection: agent's cached gen1 is now stale
	if gs.IsCachedGenerationValid(ctx, secretID, gen1) {
		t.Fatal("Cached gen1 should be invalid after rotation to gen2")
	}

	// New deliveries must use gen2
	if !gs.IsCachedGenerationValid(ctx, secretID, gen2) {
		t.Fatal("Cached gen2 should be valid")
	}

	// gen1 envelopes are rejected
	nonce := []byte("nonce-gen1")
	err := gs.RecordConsumption(ctx, secretID, gen1.GenerationID, nonce)
	if err == nil {
		t.Fatal("Should reject stale gen1 after partition recovery")
	}

	// gen2 envelopes are accepted
	err = gs.RecordConsumption(ctx, secretID, gen2.GenerationID, nonce)
	if err != nil {
		t.Fatalf("Should accept gen2 after partition recovery: %v", err)
	}

	t.Logf("PASS: Rotation during partition invalidates cached generation")
}

// TestFailClosedWithoutCachedGeneration verifies unknown authorization is denied
func TestFailClosedWithoutCachedGeneration(t *testing.T) {
	gs := NewGenerationStore()
	ctx := context.Background()

	secretID := "unknown-secret"
	cachedGen := DeliveryGeneration{
		SecretID:     secretID,
		Version:      1,
		GenerationID: "gen-cached-001",
		CreatedBy:    "control-plane",
	}

	// Agent has no information about this secret (first delivery, no cache)
	// During partition, agent cannot confirm it's still authorized

	// IsCachedGenerationValid must return false because no current generation exists
	if gs.IsCachedGenerationValid(ctx, secretID, cachedGen) {
		t.Fatal("Should fail-closed: unknown generation cannot be assumed valid")
	}

	// RecordConsumption also fails because current generation not found
	nonce := []byte("nonce-unknown")
	err := gs.RecordConsumption(ctx, secretID, cachedGen.GenerationID, nonce)
	if err == nil {
		t.Fatal("Should fail-closed: unknown generation should be rejected")
	}

	t.Logf("PASS: Fail-closed semantics enforced for unknown generation")
}

// TestCachedGenerationBecomesStaleAfterRevocation verifies revocation invalidates cache
func TestCachedGenerationBecomesStaleAfterRevocation(t *testing.T) {
	gs := NewGenerationStore()
	ctx := context.Background()

	secretID := "revoked-during-partition"
	gen := DeliveryGeneration{
		SecretID:     secretID,
		Version:      1,
		GenerationID: "gen-001",
		CreatedBy:    "control-plane",
	}

	// Before partition: generation active
	if err := gs.SetCurrentGeneration(ctx, secretID, gen); err != nil {
		t.Fatalf("SetCurrentGeneration failed: %v", err)
	}

	// Partition begins: agent is disconnected
	// While disconnected, control-plane revokes the secret
	// Revocation is represented as clearing the current generation or setting a revoked marker

	// Simulate revocation by setting a new "revoked" generation
	// (in production, could also be a separate revocation state)
	revokedGen := DeliveryGeneration{
		SecretID:     secretID,
		Version:      1,
		GenerationID: "gen-revoked-001", // Different ID = revocation
		CreatedBy:    "control-plane",
	}

	if err := gs.SetCurrentGeneration(ctx, secretID, revokedGen); err != nil {
		t.Fatalf("SetCurrentGeneration for revoked failed: %v", err)
	}

	// After reconnection: cached gen is now invalid
	if gs.IsCachedGenerationValid(ctx, secretID, gen) {
		t.Fatal("Cached generation should be invalid after revocation")
	}

	// New deliveries using old generation are rejected
	nonce := []byte("nonce-revoked-secret")
	err := gs.RecordConsumption(ctx, secretID, gen.GenerationID, nonce)
	if err == nil {
		t.Fatal("Should reject delivery for revoked generation")
	}

	t.Logf("PASS: Revocation during partition invalidates cached generation")
}

// TestMultipleSecretsPartitionRecovery verifies recovery with multiple secrets
func TestMultipleSecretsPartitionRecovery(t *testing.T) {
	gs := NewGenerationStore()
	ctx := context.Background()

	// Setup 3 secrets before partition
	secrets := []DeliveryGeneration{
		{SecretID: "secret-1", Version: 1, GenerationID: "gen-1-001", CreatedBy: "control-plane"},
		{SecretID: "secret-2", Version: 1, GenerationID: "gen-2-001", CreatedBy: "control-plane"},
		{SecretID: "secret-3", Version: 1, GenerationID: "gen-3-001", CreatedBy: "control-plane"},
	}

	for _, gen := range secrets {
		if err := gs.SetCurrentGeneration(ctx, gen.SecretID, gen); err != nil {
			t.Fatalf("SetCurrentGeneration failed: %v", err)
		}
	}

	// During partition: only secret-1 and secret-3 are rotated
	secrets[0].GenerationID = "gen-1-v2-001"
	secrets[2].GenerationID = "gen-3-v2-001"

	for i, gen := range secrets {
		if i == 1 {
			continue // secret-2 not rotated
		}
		if err := gs.SetCurrentGeneration(ctx, gen.SecretID, gen); err != nil {
			t.Fatalf("SetCurrentGeneration failed: %v", err)
		}
	}

	// After partition recovery: each secret's cache is evaluated independently
	tests := []struct {
		secretID      string
		genID         string
		shouldBeValid bool
	}{
		{"secret-1", "gen-1-v2-001", true},  // Rotated
		{"secret-1", "gen-1-001", false},    // Stale
		{"secret-2", "gen-2-001", true},     // Not rotated
		{"secret-3", "gen-3-v2-001", true},  // Rotated
		{"secret-3", "gen-3-001", false},    // Stale
	}

	for _, test := range tests {
		cached := DeliveryGeneration{
			SecretID:     test.secretID,
			GenerationID: test.genID,
		}

		valid := gs.IsCachedGenerationValid(ctx, test.secretID, cached)
		if valid != test.shouldBeValid {
			t.Errorf("Secret %s gen %s: expected valid=%v, got %v",
				test.secretID, test.genID, test.shouldBeValid, valid)
		}
	}

	t.Logf("PASS: Multiple secrets partition recovery handled independently")
}
