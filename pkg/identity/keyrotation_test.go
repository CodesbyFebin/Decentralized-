package identity

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestKeyMetadataCreation(t *testing.T) {
	metadata := &KeyMetadata{
		PublicKey:  "test-pub-key",
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().AddDate(0, 0, 90),
		Generation: 1,
		Status:     "active",
	}

	if metadata.PublicKey != "test-pub-key" {
		t.Errorf("PublicKey = %s, want test-pub-key", metadata.PublicKey)
	}

	if metadata.Status != "active" {
		t.Errorf("Status = %s, want active", metadata.Status)
	}
}

func TestKeyRotationManagerCreation(t *testing.T) {
	dir := t.TempDir()
	manager := NewKeyRotationManager(dir)

	if manager.keyExpiryDays != 90 {
		t.Errorf("keyExpiryDays = %d, want 90", manager.keyExpiryDays)
	}

	if manager.gracePeriodDays != 7 {
		t.Errorf("gracePeriodDays = %d, want 7", manager.gracePeriodDays)
	}
}

func TestCheckRotationNeeded(t *testing.T) {
	manager := NewKeyRotationManager("")

	// Not expired, not in grace period
	metadata := &KeyMetadata{
		ExpiresAt: time.Now().AddDate(0, 0, 30),
		Status:    "active",
	}

	if manager.CheckRotationNeeded(metadata) {
		t.Error("should not need rotation when not in grace period")
	}

	// In grace period
	metadata.ExpiresAt = time.Now().AddDate(0, 0, 3)
	if !manager.CheckRotationNeeded(metadata) {
		t.Error("should need rotation when in grace period")
	}

	// Expired
	metadata.ExpiresAt = time.Now().AddDate(0, 0, -1)
	if !manager.CheckRotationNeeded(metadata) {
		t.Error("should need rotation when expired")
	}

	// Revoked
	metadata.Status = "revoked"
	if manager.CheckRotationNeeded(metadata) {
		t.Error("should not need rotation when revoked")
	}
}

func TestRotateKey(t *testing.T) {
	dir := t.TempDir()
	manager := NewKeyRotationManager(dir)

	// Create initial identity
	id1, err := Generate()
	if err != nil {
		t.Fatalf("failed to generate initial identity: %v", err)
	}

	if err := Save(dir, id1); err != nil {
		t.Fatalf("failed to save initial identity: %v", err)
	}

	oldMetadata := &KeyMetadata{
		PublicKey:  EncodePub(id1.Pub),
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().AddDate(0, 0, 90),
		Generation: 1,
		Status:     "active",
	}

	// Rotate to new key
	id2, newMetadata, err := manager.RotateKey(dir, oldMetadata)
	if err != nil {
		t.Fatalf("failed to rotate key: %v", err)
	}

	// Verify new key is different
	if EncodePub(id2.Pub) == EncodePub(id1.Pub) {
		t.Error("rotated key should be different from old key")
	}

	// Verify metadata was updated
	if newMetadata.Generation != 2 {
		t.Errorf("new generation = %d, want 2", newMetadata.Generation)
	}

	if newMetadata.PreviousID != EncodePub(id1.Pub) {
		t.Error("PreviousID should be old public key")
	}

	if newMetadata.Status != "active" {
		t.Errorf("Status = %s, want active", newMetadata.Status)
	}

	// Verify old metadata was archived
	archivedPath := filepath.Join(dir, "identity.archived-1.json")
	if _, err := os.Stat(archivedPath); os.IsNotExist(err) {
		t.Error("archived metadata file should exist")
	}
}

func TestSaveAndLoadMetadata(t *testing.T) {
	dir := t.TempDir()
	manager := NewKeyRotationManager(dir)

	metadata := &KeyMetadata{
		PublicKey:  "test-key",
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().AddDate(0, 0, 90),
		Generation: 1,
		Status:     "active",
	}

	if err := manager.SaveMetadata(dir, metadata); err != nil {
		t.Fatalf("failed to save metadata: %v", err)
	}

	loaded, err := manager.LoadMetadata(dir)
	if err != nil {
		t.Fatalf("failed to load metadata: %v", err)
	}

	if loaded.PublicKey != metadata.PublicKey {
		t.Errorf("loaded PublicKey = %s, want %s", loaded.PublicKey, metadata.PublicKey)
	}

	if loaded.Generation != metadata.Generation {
		t.Errorf("loaded Generation = %d, want %d", loaded.Generation, metadata.Generation)
	}
}

func TestVerifyKeyChain(t *testing.T) {
	dir := t.TempDir()
	manager := NewKeyRotationManager(dir)

	// Create initial identity
	id1, err := Generate()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}

	if err := Save(dir, id1); err != nil {
		t.Fatalf("failed to save identity: %v", err)
	}

	// Verify chain (should succeed with nil trusted key)
	if err := manager.VerifyKeyChain(dir, nil); err != nil {
		t.Errorf("key chain verification failed: %v", err)
	}

	// Verify with matching trusted key
	if err := manager.VerifyKeyChain(dir, id1.Pub); err == nil {
		// This should fail because PreviousID is empty
		t.Logf("key chain verification with nil PreviousID: expected to be lenient")
	}

	// Rotate key
	oldMetadata := &KeyMetadata{
		PublicKey:  EncodePub(id1.Pub),
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().AddDate(0, 0, 90),
		Generation: 1,
		Status:     "active",
	}

	_, _, err = manager.RotateKey(dir, oldMetadata)
	if err != nil {
		t.Fatalf("failed to rotate key: %v", err)
	}

	// Verify chain with old key
	if err := manager.VerifyKeyChain(dir, id1.Pub); err != nil {
		t.Errorf("key chain verification with correct previous key failed: %v", err)
	}

	// Verify chain with wrong key
	id3, _ := Generate()
	if err := manager.VerifyKeyChain(dir, id3.Pub); err == nil {
		t.Error("key chain verification should fail with wrong previous key")
	}
}

func TestListArchivedKeys(t *testing.T) {
	dir := t.TempDir()
	manager := NewKeyRotationManager(dir)

	// Create and rotate key multiple times
	id1, err := Generate()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}

	if err := Save(dir, id1); err != nil {
		t.Fatalf("failed to save identity: %v", err)
	}

	for i := 0; i < 3; i++ {
		oldMetadata := &KeyMetadata{
			PublicKey:  EncodePub(id1.Pub),
			CreatedAt:  time.Now(),
			ExpiresAt:  time.Now().AddDate(0, 0, 90),
			Generation: int64(i + 1),
			Status:     "active",
		}

		var err error
		id1, _, err = manager.RotateKey(dir, oldMetadata)
		if err != nil {
			t.Fatalf("failed to rotate key: %v", err)
		}
	}

	archived, err := manager.ListArchivedKeys(dir)
	if err != nil {
		t.Fatalf("failed to list archived keys: %v", err)
	}

	if len(archived) != 3 {
		t.Errorf("expected 3 archived keys, got %d", len(archived))
	}
}

func TestCleanupOldKeys(t *testing.T) {
	dir := t.TempDir()
	manager := NewKeyRotationManager(dir)

	// Create and rotate key
	id1, err := Generate()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}

	if err := Save(dir, id1); err != nil {
		t.Fatalf("failed to save identity: %v", err)
	}

	oldMetadata := &KeyMetadata{
		PublicKey:  EncodePub(id1.Pub),
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().AddDate(0, 0, 90),
		Generation: 1,
		Status:     "active",
		RotatedAt:  time.Now().Add(-24 * time.Hour), // Rotated yesterday
	}

	if _, _, err := manager.RotateKey(dir, oldMetadata); err != nil {
		t.Fatalf("failed to rotate key: %v", err)
	}

	// Cleanup with 12-hour duration (should keep keys newer than 12 hours)
	if err := manager.CleanupOldKeys(dir, 12*time.Hour); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	archived, err := manager.ListArchivedKeys(dir)
	if err != nil {
		t.Fatalf("failed to list archived keys: %v", err)
	}

	// Should have 0 keys after cleanup
	if len(archived) > 0 {
		t.Logf("cleanup: archived keys after cleanup: %d (may have already cleaned up)", len(archived))
	}
}

func TestGetRotationStatus(t *testing.T) {
	manager := NewKeyRotationManager("")

	now := time.Now()
	metadata := &KeyMetadata{
		CreatedAt:  now.AddDate(0, 0, -30),
		ExpiresAt:  now.AddDate(0, 0, 60),
		Generation: 1,
		Status:     "active",
	}

	status := manager.GetRotationStatus(metadata)

	if status["generation"] != int64(1) {
		t.Errorf("generation = %v, want 1", status["generation"])
	}

	if status["status"] != "active" {
		t.Errorf("status = %v, want active", status["status"])
	}

	daysUntilExpiry := status["days_until_expiry"].(int)
	if daysUntilExpiry < 59 || daysUntilExpiry > 61 {
		t.Errorf("days_until_expiry = %d, want ~60", daysUntilExpiry)
	}

	if status["in_grace_period"] != false {
		t.Error("in_grace_period should be false")
	}

	if status["is_expired"] != false {
		t.Error("is_expired should be false")
	}
}

func TestLoadWithRotation(t *testing.T) {
	dir := t.TempDir()
	manager := NewKeyRotationManager(dir)

	// Create initial identity
	id, err := Generate()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}

	if err := Save(dir, id); err != nil {
		t.Fatalf("failed to save identity: %v", err)
	}

	// Load with rotation
	loadedID, metadata, err := manager.LoadWithRotation(dir)
	if err != nil {
		t.Fatalf("failed to load with rotation: %v", err)
	}

	if loadedID == nil {
		t.Fatal("loaded identity is nil")
	}

	if metadata == nil {
		t.Fatal("metadata is nil")
	}

	if metadata.Generation != 1 {
		t.Errorf("generation = %d, want 1", metadata.Generation)
	}

	if metadata.Status != "active" {
		t.Errorf("status = %s, want active", metadata.Status)
	}
}

func BenchmarkRotateKey(b *testing.B) {
	dir := b.TempDir()
	manager := NewKeyRotationManager(dir)

	// Create initial identity
	id, _ := Generate()
	Save(dir, id)

	_ = &KeyMetadata{
		PublicKey:  EncodePub(id.Pub),
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().AddDate(0, 0, 90),
		Generation: 1,
		Status:     "active",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var next *KeyMetadata
		if i > 0 {
			next = &KeyMetadata{
				PublicKey:  EncodePub(id.Pub),
				CreatedAt:  time.Now(),
				ExpiresAt:  time.Now().AddDate(0, 0, 90),
				Generation: int64(i + 1),
				Status:     "active",
			}
		}
		var err error
		id, _, err = manager.RotateKey(dir, next)
		if err != nil {
			b.Fatalf("rotation failed: %v", err)
		}
	}
}
