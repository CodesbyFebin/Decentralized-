// Package identity key rotation implements 90-day key rotation enforcement.
package identity

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// KeyMetadata tracks expiry and rotation status.
type KeyMetadata struct {
	PublicKey  string    `json:"public_key"`  // Wire-encoded public key
	CreatedAt  time.Time `json:"created_at"`  // When this key was created
	ExpiresAt  time.Time `json:"expires_at"`  // When this key expires (90 days after creation)
	RotatedAt  time.Time `json:"rotated_at"`  // When rotation was performed (zero if not rotated yet)
	PreviousID string    `json:"previous_id"` // ID of previous key in rotation chain
	Generation int64     `json:"generation"`  // Monotonic counter for key versions
	Status     string    `json:"status"`      // "active", "rotating", "revoked", "archived"
}

// KeyRotationManager handles key lifecycle and rotation.
type KeyRotationManager struct {
	dir              string
	keyExpiryDays    int           // Default 90 days
	gracePeriodDays  int           // Default 7 days before expiry
	rotationInterval time.Duration // How often to check for rotation needed
}

// NewKeyRotationManager creates a new key rotation manager.
func NewKeyRotationManager(dir string) *KeyRotationManager {
	return &KeyRotationManager{
		dir:              dir,
		keyExpiryDays:    90,
		gracePeriodDays:  7,
		rotationInterval: 24 * time.Hour,
	}
}

// LoadWithRotation loads an identity and checks if rotation is needed.
func (m *KeyRotationManager) LoadWithRotation(dir string) (*Identity, *KeyMetadata, error) {
	// Load current identity
	id, err := Load(dir)
	if err != nil {
		return nil, nil, err
	}

	// Load metadata
	metadata, err := m.LoadMetadata(dir)
	if err != nil {
		// First time loading, create metadata
		metadata = &KeyMetadata{
			PublicKey:  EncodePub(id.Pub),
			CreatedAt:  time.Now(),
			ExpiresAt:  time.Now().AddDate(0, 0, m.keyExpiryDays),
			Generation: 1,
			Status:     "active",
		}

		if err := m.SaveMetadata(dir, metadata); err != nil {
			return nil, metadata, fmt.Errorf("failed to save metadata: %w", err)
		}
	}

	return id, metadata, nil
}

// CheckRotationNeeded returns true if key rotation is needed.
func (m *KeyRotationManager) CheckRotationNeeded(metadata *KeyMetadata) bool {
	if metadata.Status == "revoked" || metadata.Status == "archived" {
		return false
	}

	now := time.Now()
	gracePeriodStart := metadata.ExpiresAt.AddDate(0, 0, -m.gracePeriodDays)

	// Rotation needed if in grace period or expired
	return now.After(gracePeriodStart)
}

// RotateKey generates a new key and tracks the old one.
func (m *KeyRotationManager) RotateKey(dir string, oldMetadata *KeyMetadata) (*Identity, *KeyMetadata, error) {
	// Generate new key
	newID, err := Generate()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate new key: %w", err)
	}

	// Archive old metadata
	if oldMetadata != nil {
		oldMetadata.RotatedAt = time.Now()
		oldMetadata.Status = "archived"

		// Save old metadata with timestamp
		archivedPath := filepath.Join(dir, fmt.Sprintf("identity.archived-%d.json", oldMetadata.Generation))
		if err := m.saveMetadataToFile(archivedPath, oldMetadata); err != nil {
			return nil, nil, fmt.Errorf("failed to archive old metadata: %w", err)
		}
	}

	// Create new metadata
	newMetadata := &KeyMetadata{
		PublicKey:  EncodePub(newID.Pub),
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().AddDate(0, 0, m.keyExpiryDays),
		Generation: 1,
		Status:     "active",
	}

	if oldMetadata != nil {
		newMetadata.PreviousID = oldMetadata.PublicKey
		newMetadata.Generation = oldMetadata.Generation + 1
	}

	// Save new identity and metadata
	if err := Save(dir, newID); err != nil {
		return nil, nil, fmt.Errorf("failed to save new identity: %w", err)
	}

	if err := m.SaveMetadata(dir, newMetadata); err != nil {
		return nil, nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	return newID, newMetadata, nil
}

// LoadMetadata loads key metadata from disk.
func (m *KeyRotationManager) LoadMetadata(dir string) (*KeyMetadata, error) {
	path := filepath.Join(dir, "identity.metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var metadata KeyMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// SaveMetadata saves key metadata to disk.
func (m *KeyRotationManager) SaveMetadata(dir string, metadata *KeyMetadata) error {
	path := filepath.Join(dir, "identity.metadata.json")
	return m.saveMetadataToFile(path, metadata)
}

// saveMetadataToFile saves metadata to a specific file path.
func (m *KeyRotationManager) saveMetadataToFile(path string, metadata *KeyMetadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return writeAtomic(path, data, 0o644)
}

// VerifyKeyChain verifies a rotation chain starting from a trusted key.
func (m *KeyRotationManager) VerifyKeyChain(dir string, trustedKey ed25519.PublicKey) error {
	id, metadata, err := m.LoadWithRotation(dir)
	if err != nil {
		return err
	}

	// Verify current key matches metadata
	if EncodePub(id.Pub) != metadata.PublicKey {
		return errors.New("key mismatch: current key does not match metadata")
	}

	// If previous key was provided, verify the chain
	if metadata.PreviousID != "" && trustedKey != nil {
		if EncodePub(trustedKey) != metadata.PreviousID {
			return fmt.Errorf("key chain broken: previous key does not match (expected %s, got %s)",
				metadata.PreviousID, EncodePub(trustedKey))
		}
	}

	// Verify expiry is in the future (unless explicitly testing)
	if time.Now().After(metadata.ExpiresAt) && metadata.Status == "active" {
		return fmt.Errorf("key is expired as of %s", metadata.ExpiresAt)
	}

	return nil
}

// ListArchivedKeys returns all archived key metadata.
func (m *KeyRotationManager) ListArchivedKeys(dir string) ([]*KeyMetadata, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var archived []*KeyMetadata

	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > len("identity.archived-") {
			if entry.Name()[:len("identity.archived-")] == "identity.archived-" {
				path := filepath.Join(dir, entry.Name())
				metadata, err := m.loadMetadataFromFile(path)
				if err != nil {
					// Log error but continue
					continue
				}
				archived = append(archived, metadata)
			}
		}
	}

	return archived, nil
}

// loadMetadataFromFile loads metadata from a specific file path.
func (m *KeyRotationManager) loadMetadataFromFile(path string) (*KeyMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var metadata KeyMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// CleanupOldKeys removes archived keys older than specified duration.
func (m *KeyRotationManager) CleanupOldKeys(dir string, keepDuration time.Duration) error {
	archived, err := m.ListArchivedKeys(dir)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-keepDuration)

	for _, metadata := range archived {
		if metadata.RotatedAt.Before(cutoff) && metadata.Status == "archived" {
			// Remove the file
			path := filepath.Join(dir, fmt.Sprintf("identity.archived-%d.json", metadata.Generation))
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove archived key: %w", err)
			}
		}
	}

	return nil
}

// GetRotationStatus returns current rotation status.
func (m *KeyRotationManager) GetRotationStatus(metadata *KeyMetadata) map[string]interface{} {
	now := time.Now()
	gracePeriodStart := metadata.ExpiresAt.AddDate(0, 0, -m.gracePeriodDays)

	daysUntilExpiry := int(metadata.ExpiresAt.Sub(now).Hours() / 24)
	isInGracePeriod := now.After(gracePeriodStart)
	isExpired := now.After(metadata.ExpiresAt)

	return map[string]interface{}{
		"generation":          metadata.Generation,
		"created_at":          metadata.CreatedAt,
		"expires_at":          metadata.ExpiresAt,
		"days_until_expiry":   daysUntilExpiry,
		"in_grace_period":     isInGracePeriod,
		"is_expired":          isExpired,
		"status":              metadata.Status,
		"grace_period_days":   m.gracePeriodDays,
	}
}
