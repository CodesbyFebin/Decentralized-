package runtime

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// DeliveryGeneration tracks secret version and generation for replay prevention
type DeliveryGeneration struct {
	SecretID     string
	Version      int
	GenerationID string // Changes on each rotation or revocation
	IssuedAt     int64  // Unix nanoseconds
	CreatedBy    string // Control plane identity
}

// DeliveryConsumption tracks a delivered envelope consumption (for replay prevention)
type DeliveryConsumption struct {
	DeliveryID   string
	GenerationID string
	Nonce        []byte
	Timestamp    int64 // Unix nanoseconds
}

// GenerationStore maintains current generation state and tracks consumption
type GenerationStore struct {
	// Current generation per secret
	currentGenerations map[string]DeliveryGeneration

	// Consumed delivery records (generationId + nonce -> timestamp)
	consumed map[string]int64 // key = sha256(generationId + nonce)

	mu sync.RWMutex

	// TTL for consumption tracking (prevent memory leak)
	consumptionTTL time.Duration
}

// NewGenerationStore creates a new generation store
func NewGenerationStore() *GenerationStore {
	return &GenerationStore{
		currentGenerations: make(map[string]DeliveryGeneration),
		consumed:           make(map[string]int64),
		consumptionTTL:     5 * time.Minute, // Consumption records expire after 5 minutes
	}
}

// SetCurrentGeneration records the current active generation for a secret
func (gs *GenerationStore) SetCurrentGeneration(ctx context.Context, secretID string, gen DeliveryGeneration) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.currentGenerations[secretID] = gen
	return nil
}

// GetCurrentGeneration retrieves the current generation for a secret
func (gs *GenerationStore) GetCurrentGeneration(ctx context.Context, secretID string) (DeliveryGeneration, error) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	gen, exists := gs.currentGenerations[secretID]
	if !exists {
		return DeliveryGeneration{}, fmt.Errorf("no generation for secret %s", secretID)
	}

	return gen, nil
}

// IsCurrentGeneration returns true if the provided generation is current for the secret
func (gs *GenerationStore) IsCurrentGeneration(ctx context.Context, secretID string, generationID string) bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	gen, exists := gs.currentGenerations[secretID]
	if !exists {
		return false
	}

	return gen.GenerationID == generationID
}

// RecordConsumption records that a delivery was consumed
// Returns error if generation is stale or nonce was already consumed
func (gs *GenerationStore) RecordConsumption(ctx context.Context, secretID string, generationID string, nonce []byte) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	// Check if generation is current
	current, exists := gs.currentGenerations[secretID]
	if !exists {
		return fmt.Errorf("generation not found for secret %s", secretID)
	}

	if current.GenerationID != generationID {
		return fmt.Errorf("generation %s is stale (current: %s)", generationID, current.GenerationID)
	}

	// Check if this nonce was already consumed
	key := consumptionKey(generationID, nonce)
	if consumedAt, exists := gs.consumed[key]; exists {
		// Allow if consumed recently (same request), reject if old consumption
		age := time.Since(time.Unix(0, consumedAt))
		if age < 100*time.Millisecond {
			// Likely same request being retried
			return nil
		}
		return fmt.Errorf("nonce already consumed at %v ago", age)
	}

	// Record consumption
	gs.consumed[key] = time.Now().UnixNano()
	return nil
}

// IsConsumed returns true if the delivery (generation + nonce) has been consumed
func (gs *GenerationStore) IsConsumed(ctx context.Context, generationID string, nonce []byte) bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	key := consumptionKey(generationID, nonce)
	_, exists := gs.consumed[key]
	return exists
}

// CacheCurrentGeneration returns current generation if still valid, error if stale/revoked
// Used during partition to determine if cached authorization can be used
func (gs *GenerationStore) IsCachedGenerationValid(ctx context.Context, secretID string, cachedGeneration DeliveryGeneration) bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	current, exists := gs.currentGenerations[secretID]
	if !exists {
		return false
	}

	// Cached generation is valid if it matches current (no rotation/revocation)
	return current.GenerationID == cachedGeneration.GenerationID
}

// Cleanup removes old consumption records
func (gs *GenerationStore) Cleanup(ctx context.Context) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-gs.consumptionTTL)

	for key, consumedAt := range gs.consumed {
		if time.Unix(0, consumedAt).Before(cutoff) {
			delete(gs.consumed, key)
		}
	}

	return nil
}

// consumptionKey creates a unique key for a generation + nonce pair
func consumptionKey(generationID string, nonce []byte) string {
	h := sha256.New()
	h.Write([]byte(generationID))
	h.Write(nonce)
	return fmt.Sprintf("%x", h.Sum(nil))
}
