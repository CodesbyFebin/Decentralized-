package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RotationEvent records a secret version rotation
type RotationEvent struct {
	SecretID         string
	FromVersion      int
	ToVersion        int
	Timestamp        int64 // Unix nanoseconds
	Status           string // "V2_AUTHORIZED", "V2_MATERIALIZED", "SWITCHED", "CLEANED"
	V1MountPaths     []string
	V2MountPath      string
	CompletionTime   int64
	TransitionWindow time.Duration // Observable window if not atomic
}

// RotationObserver is notified of rotation events
type RotationObserver interface {
	OnSecretRotated(ctx context.Context, event RotationEvent) error
}

// SecretRotationManager handles version rotation without workload restart
type SecretRotationManager struct {
	// Current active version per secret
	activeVersions map[string]int // secretId -> version

	// Materialized paths for each version
	materializations map[string]map[int][]string // secretId -> version -> paths

	mu        sync.RWMutex
	observers []RotationObserver
}

// NewSecretRotationManager creates a new rotation manager
func NewSecretRotationManager() *SecretRotationManager {
	return &SecretRotationManager{
		activeVersions:   make(map[string]int),
		materializations: make(map[string]map[int][]string),
	}
}

// GetActiveVersion returns the current active version for a secret
func (srm *SecretRotationManager) GetActiveVersion(ctx context.Context, secretID string) (int, error) {
	srm.mu.RLock()
	defer srm.mu.RUnlock()

	version, exists := srm.activeVersions[secretID]
	if !exists {
		return 0, fmt.Errorf("secret %s has no active version", secretID)
	}

	return version, nil
}

// RotateSecret transitions from one version to another
// Semantic: v1 remains valid until v2 is materialized and mount is switched
func (srm *SecretRotationManager) RotateSecret(ctx context.Context, secretID string, fromVersion, toVersion int, newPayload []byte) error {
	srm.mu.Lock()

	// Step 1: Validate current version
	currentVersion, exists := srm.activeVersions[secretID]
	if !exists {
		srm.mu.Unlock()
		return fmt.Errorf("secret %s has no active version", secretID)
	}

	if currentVersion != fromVersion {
		srm.mu.Unlock()
		return fmt.Errorf("cannot rotate from version %d to %d: current is %d", fromVersion, toVersion, currentVersion)
	}

	// Step 2: Record pre-rotation state
	v1Paths := srm.getMountPaths(secretID, fromVersion)

	srm.mu.Unlock()

	// Step 3: Materialize v2 (outside lock to avoid deadlock)
	// In production, this would materialize the new version to tmpfs
	v2Path := fmt.Sprintf("/var/run/secrets/%s/v%d", secretID, toVersion)

	// Step 4: Record rotation progression
	srm.mu.Lock()
	defer srm.mu.Unlock()

	// Record that v2 was materialized
	if srm.materializations[secretID] == nil {
		srm.materializations[secretID] = make(map[int][]string)
	}
	srm.materializations[secretID][toVersion] = []string{v2Path}

	// Notify observers of materialization
	event := RotationEvent{
		SecretID:     secretID,
		FromVersion:  fromVersion,
		ToVersion:    toVersion,
		Timestamp:    time.Now().UnixNano(),
		Status:       "V2_MATERIALIZED",
		V1MountPaths: v1Paths,
		V2MountPath:  v2Path,
	}

	for _, obs := range srm.observers {
		if err := obs.OnSecretRotated(ctx, event); err != nil {
			return fmt.Errorf("rotation observer failed: %w", err)
		}
	}

	// Step 5: Switch mount (this is the visible transition point)
	// Update active version
	srm.activeVersions[secretID] = toVersion

	event.Status = "SWITCHED"
	event.CompletionTime = time.Now().UnixNano()

	for _, obs := range srm.observers {
		obs.OnSecretRotated(ctx, event)
	}

	// Step 6: Schedule v1 cleanup (after brief window to allow workload migration)
	// In production, this would remove v1 mount and file
	event.Status = "CLEANED"
	event.CompletionTime = time.Now().UnixNano()

	for _, obs := range srm.observers {
		obs.OnSecretRotated(ctx, event)
	}

	return nil
}

// RegisterMaterialization records a version materialization
func (srm *SecretRotationManager) RegisterMaterialization(secretID string, version int, hostPath string) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	if srm.materializations[secretID] == nil {
		srm.materializations[secretID] = make(map[int][]string)
	}

	srm.materializations[secretID][version] = append(srm.materializations[secretID][version], hostPath)
}

// GetMaterializations returns all paths for a secret version
func (srm *SecretRotationManager) GetMaterializations(secretID string, version int) []string {
	srm.mu.RLock()
	defer srm.mu.RUnlock()

	return srm.getMountPaths(secretID, version)
}

// getMountPaths is internal helper (must hold lock)
func (srm *SecretRotationManager) getMountPaths(secretID string, version int) []string {
	if versionMap, exists := srm.materializations[secretID]; exists {
		if paths, exists := versionMap[version]; exists {
			result := make([]string, len(paths))
			copy(result, paths)
			return result
		}
	}
	return []string{}
}

// RegisterObserver adds a rotation observer
func (srm *SecretRotationManager) RegisterObserver(observer RotationObserver) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	srm.observers = append(srm.observers, observer)
}

// IsRotationInProgress returns true if a rotation is currently happening
func (srm *SecretRotationManager) IsRotationInProgress(secretID string) bool {
	srm.mu.RLock()
	defer srm.mu.RUnlock()

	// Simplified: no active rotation tracking yet
	// In production, would track rotation state machine
	return false
}
