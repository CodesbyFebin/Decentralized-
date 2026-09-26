package runtime

import (
	"context"
	"testing"
	"time"
)

// MockRevocationObserver tracks revocation events for testing
type MockRevocationObserver struct {
	events []RevocationEvent
}

func (mro *MockRevocationObserver) OnSecretRevoked(ctx context.Context, event RevocationEvent) error {
	mro.events = append(mro.events, event)
	return nil
}

// TestRevokeRunningSecret verifies revocation while workload is running
func TestRevokeRunningSecret(t *testing.T) {
	rm := NewRevocationManager()
	observer := &MockRevocationObserver{}
	rm.RegisterObserver(observer)

	ctx := context.Background()
	secretID := "secret-1"

	// Register an active materialization
	hostPath := "/var/run/secrets/secret-1/ephemeral_abc_123"
	if err := rm.RegisterMaterialization(hostPath, secretID); err != nil {
		t.Fatalf("RegisterMaterialization failed: %v", err)
	}

	// Verify secret is not revoked
	if rm.IsRevoked(ctx, secretID) {
		t.Fatal("Secret should not be revoked yet")
	}

	// Revoke the secret
	if err := rm.RevokeSecret(ctx, secretID); err != nil {
		t.Fatalf("RevokeSecret failed: %v", err)
	}

	// Verify secret is now revoked
	if !rm.IsRevoked(ctx, secretID) {
		t.Fatal("Secret should be revoked")
	}

	// Verify observer was notified
	if len(observer.events) < 1 {
		t.Fatal("Observer should have been notified")
	}

	if observer.events[0].Status != "REVOKED" {
		t.Errorf("First event should be REVOKED, got %s", observer.events[0].Status)
	}

	// Verify affected mounts were identified
	if len(observer.events[0].AffectedMounts) != 1 {
		t.Errorf("Should identify 1 affected mount, got %d", len(observer.events[0].AffectedMounts))
	}

	t.Logf("PASS: Revocation of running secret correctly revoked materialization")
}

// TestRetrievalAfterRevocationDenied verifies new retrievals are denied after revocation
func TestRetrievalAfterRevocationDenied(t *testing.T) {
	rm := NewRevocationManager()
	ctx := context.Background()

	secretID := "secret-revoked"

	// Register materialization
	hostPath := "/var/run/secrets/secret-revoked/ephemeral_xyz_789"
	if err := rm.RegisterMaterialization(hostPath, secretID); err != nil {
		t.Fatalf("RegisterMaterialization failed: %v", err)
	}

	// Revoke
	if err := rm.RevokeSecret(ctx, secretID); err != nil {
		t.Fatalf("RevokeSecret failed: %v", err)
	}

	// Try to register a new materialization for this secret
	// Should fail because secret is revoked
	newPath := "/var/run/secrets/secret-revoked/ephemeral_new_999"
	err := rm.RegisterMaterialization(newPath, secretID)
	if err == nil {
		t.Fatal("Should not allow materialization of revoked secret")
	}

	t.Logf("PASS: Retrieval correctly denied after revocation: %v", err)
}

// TestRevokeDuringDelivery verifies revocation while delivery is in flight
func TestRevokeDuringDelivery(t *testing.T) {
	rm := NewRevocationManager()
	observer := &MockRevocationObserver{}
	rm.RegisterObserver(observer)

	ctx := context.Background()
	secretID := "secret-in-flight"

	// Simulate: delivery has started but not yet materialized
	// Agent receives envelope, begins validation

	// Revocation happens concurrently
	if err := rm.RevokeSecret(ctx, secretID); err != nil {
		t.Fatalf("RevokeSecret failed: %v", err)
	}

	// Now try to materialize
	hostPath := "/var/run/secrets/secret-in-flight/ephemeral_concurrent_111"
	err := rm.RegisterMaterialization(hostPath, secretID)
	if err == nil {
		t.Fatal("Should not allow materialization after revocation")
	}

	t.Logf("PASS: Concurrent revocation prevented materialization: %v", err)
}

// TestRevokeDuringMaterialization verifies cleanup when revocation occurs mid-write
func TestRevokeDuringMaterialization(t *testing.T) {
	rm := NewRevocationManager()
	observer := &MockRevocationObserver{}
	rm.RegisterObserver(observer)

	ctx := context.Background()
	secretID := "secret-midwrite"

	// Register a materialization (simulates mid-write state)
	hostPath := "/var/run/secrets/secret-midwrite/ephemeral_partial_222"
	if err := rm.RegisterMaterialization(hostPath, secretID); err != nil {
		t.Fatalf("RegisterMaterialization failed: %v", err)
	}

	// Revoke during materialization
	if err := rm.RevokeSecret(ctx, secretID); err != nil {
		t.Fatalf("RevokeSecret failed: %v", err)
	}

	// Verify secret is revoked and affected mount is recorded
	if !rm.IsRevoked(ctx, secretID) {
		t.Fatal("Secret should be revoked")
	}

	affected := rm.GetAffectedMaterializations(secretID)
	if len(affected) == 0 {
		t.Fatal("Should identify affected materializations")
	}

	if affected[0] != hostPath {
		t.Errorf("Should identify correct mount path, got %s", affected[0])
	}

	t.Logf("PASS: Revocation during materialization correctly identified affected mounts")
}

// TestRevokeMultipleWorkloads verifies revocation affects all workloads using a secret
func TestRevokeMultipleWorkloads(t *testing.T) {
	rm := NewRevocationManager()
	observer := &MockRevocationObserver{}
	rm.RegisterObserver(observer)

	ctx := context.Background()
	secretID := "shared-secret"

	// Register materializations for multiple workloads
	workloads := []string{
		"/var/run/secrets/assign-1/ephemeral_shared_1",
		"/var/run/secrets/assign-2/ephemeral_shared_2",
		"/var/run/secrets/assign-3/ephemeral_shared_3",
	}

	for _, path := range workloads {
		if err := rm.RegisterMaterialization(path, secretID); err != nil {
			t.Fatalf("RegisterMaterialization failed: %v", err)
		}
	}

	// Revoke the shared secret
	if err := rm.RevokeSecret(ctx, secretID); err != nil {
		t.Fatalf("RevokeSecret failed: %v", err)
	}

	// Verify all affected mounts are identified
	affected := rm.GetAffectedMaterializations(secretID)
	if len(affected) != 3 {
		t.Errorf("Should identify 3 affected mounts, got %d", len(affected))
	}

	// Verify observer reports all affected mounts
	found := 0
	for _, event := range observer.events {
		if event.Status == "REVOKED" {
			found = len(event.AffectedMounts)
			break
		}
	}

	if found != 3 {
		t.Errorf("Observer should report 3 affected mounts, got %d", found)
	}

	t.Logf("PASS: Revocation correctly affected all %d workloads", len(workloads))
}

// TestRevocationTimestamp verifies revocation timestamps are recorded
func TestRevocationTimestamp(t *testing.T) {
	rm := NewRevocationManager()
	observer := &MockRevocationObserver{}
	rm.RegisterObserver(observer)

	ctx := context.Background()
	secretID := "secret-timestamp"

	beforeRevoke := time.Now().UnixNano()
	time.Sleep(10 * time.Millisecond)

	if err := rm.RevokeSecret(ctx, secretID); err != nil {
		t.Fatalf("RevokeSecret failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	afterRevoke := time.Now().UnixNano()

	if len(observer.events) == 0 {
		t.Fatal("Observer should have recorded events")
	}

	revokedEvent := observer.events[0]
	if revokedEvent.Timestamp <= beforeRevoke || revokedEvent.Timestamp >= afterRevoke {
		t.Errorf("Revocation timestamp not in expected range: %d (expected between %d and %d)",
			revokedEvent.Timestamp, beforeRevoke, afterRevoke)
	}

	t.Logf("PASS: Revocation timestamp recorded correctly")
}
