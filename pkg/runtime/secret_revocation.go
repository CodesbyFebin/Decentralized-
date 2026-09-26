package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RevocationEvent records when a secret is revoked
type RevocationEvent struct {
	SecretID       string
	Timestamp      int64
	Status         string // "REVOKED", "UNMOUNTED", "CLEANED"
	AffectedMounts []string
	AuthorizedBy   string
	CompletionTime int64
}

// RevocationManager handles live secret revocation
type RevocationManager struct {
	revokedSecrets map[string]int64 // secretId -> revocation time (Unix ns)
	materializations map[string]string // hostPath -> secretId
	mu             sync.RWMutex
	observers      []RevocationObserver
}

// RevocationObserver is notified of revocation events
type RevocationObserver interface {
	OnSecretRevoked(ctx context.Context, event RevocationEvent) error
}

// NewRevocationManager creates a new revocation manager
func NewRevocationManager() *RevocationManager {
	return &RevocationManager{
		revokedSecrets:   make(map[string]int64),
		materializations: make(map[string]string),
	}
}

// RevokeSecret marks a secret as revoked
func (rm *RevocationManager) RevokeSecret(ctx context.Context, secretID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Record revocation timestamp
	revokedAt := time.Now().UnixNano()
	rm.revokedSecrets[secretID] = revokedAt

	// Find all materializations for this secret
	affectedMounts := []string{}
	for hostPath, sid := range rm.materializations {
		if sid == secretID {
			affectedMounts = append(affectedMounts, hostPath)
		}
	}

	// Notify observers
	event := RevocationEvent{
		SecretID:       secretID,
		Timestamp:      revokedAt,
		Status:         "REVOKED",
		AffectedMounts: affectedMounts,
	}

	for _, obs := range rm.observers {
		if err := obs.OnSecretRevoked(ctx, event); err != nil {
			return fmt.Errorf("revocation observer failed: %w", err)
		}
	}

	// Record completion
	event.Status = "CLEANED"
	event.CompletionTime = time.Now().UnixNano()

	for _, obs := range rm.observers {
		obs.OnSecretRevoked(ctx, event)
	}

	return nil
}

// IsRevoked returns true if the secret has been revoked
func (rm *RevocationManager) IsRevoked(ctx context.Context, secretID string) bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	_, exists := rm.revokedSecrets[secretID]
	return exists
}

// RegisterMaterialization records a materialized secret
func (rm *RevocationManager) RegisterMaterialization(hostPath, secretID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Check if secret is already revoked
	if _, revoked := rm.revokedSecrets[secretID]; revoked {
		return fmt.Errorf("secret %s is revoked, cannot materialize", secretID)
	}

	rm.materializations[hostPath] = secretID
	return nil
}

// UnregisterMaterialization removes a materialized secret
func (rm *RevocationManager) UnregisterMaterialization(hostPath string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.materializations, hostPath)
	return nil
}

// RegisterObserver adds a revocation observer
func (rm *RevocationManager) RegisterObserver(observer RevocationObserver) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.observers = append(rm.observers, observer)
}

// GetAffectedMaterializations returns all materialized paths for a secret
func (rm *RevocationManager) GetAffectedMaterializations(secretID string) []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	affected := []string{}
	for hostPath, sid := range rm.materializations {
		if sid == secretID {
			affected = append(affected, hostPath)
		}
	}
	return affected
}
