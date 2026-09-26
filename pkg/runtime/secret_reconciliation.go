package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ReconciliationResult records the outcome of restart reconciliation
type ReconciliationResult struct {
	Timestamp       int64  // Unix nanoseconds
	Found           int    // Active materializations discovered
	Valid           int    // Still authorized
	Revoked         int    // Found to be revoked
	Removed         int    // Cleaned up
	CompletionTime  int64
}

// ReconciliationManager handles agent restart state reconciliation
type ReconciliationManager struct {
	basePath    string
	revocation  *RevocationManager
	generation  *GenerationStore
	materializer *Materializer
	mu          sync.RWMutex
	lastResult  *ReconciliationResult
}

// NewReconciliationManager creates a new reconciliation manager
func NewReconciliationManager(basePath string, revocation *RevocationManager, generation *GenerationStore, materializer *Materializer) *ReconciliationManager {
	return &ReconciliationManager{
		basePath:     basePath,
		revocation:   revocation,
		generation:   generation,
		materializer: materializer,
	}
}

// DiscoverActiveMaterializations finds all secret files in the basePath
// This is called on agent restart to determine what was active before crash
func (rm *ReconciliationManager) DiscoverActiveMaterializations(ctx context.Context) ([]string, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	discovered := []string{}

	// Walk the basePath looking for ephemeral secret files
	err := filepath.Walk(rm.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on read errors
		}

		// Look for files named ephemeral_*
		if !info.IsDir() && len(info.Name()) > 9 {
			if info.Name()[:9] == "ephemeral" {
				discovered = append(discovered, path)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("discover materializations: %w", err)
	}

	return discovered, nil
}

// ValidateMaterialization checks if a discovered materialization is still authorized
// Returns error if secret is revoked or authorization has changed
func (rm *ReconciliationManager) ValidateMaterialization(ctx context.Context, hostPath string) error {
	// Extract secret ID from path (assumes path format: /var/run/secrets/{assignmentId}/ephemeral_{secretId}_{ts})
	// For now, we'll use a simplified check

	// In production:
	// 1. Query Raft state to check if secret still exists and is authorized
	// 2. Check if secret has been revoked
	// 3. Check if generation has changed (rotation occurred)
	// 4. Check if lease (if applicable) is still valid

	// Placeholder: assume valid if file still exists
	_, err := os.Stat(hostPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("materialization not found: %s", hostPath)
		}
		return fmt.Errorf("stat materialization: %w", err)
	}

	return nil
}

// RemoveUnauthorizedMaterializations cleans up revoked or expired materializations
func (rm *ReconciliationManager) RemoveUnauthorizedMaterializations(ctx context.Context, found []string) (int, error) {
	removed := 0

	for _, hostPath := range found {
		// Validate it
		if err := rm.ValidateMaterialization(ctx, hostPath); err != nil {
			// This materialization is not valid anymore - remove it
			if err := rm.materializer.ReleaseEphemeral(ctx, hostPath); err == nil {
				removed++
			}
		}
	}

	return removed, nil
}

// ReconcileOnRestart performs full reconciliation after agent restart
func (rm *ReconciliationManager) ReconcileOnRestart(ctx context.Context) (*ReconciliationResult, error) {
	startTime := time.Now()
	result := &ReconciliationResult{
		Timestamp: startTime.UnixNano(),
	}

	// Step 1: Discover what was active
	discovered, err := rm.DiscoverActiveMaterializations(ctx)
	if err != nil {
		return nil, fmt.Errorf("discover active: %w", err)
	}

	result.Found = len(discovered)

	// Step 2: Validate each materialization
	valid := []string{}
	revoked := 0

	for _, hostPath := range discovered {
		if err := rm.ValidateMaterialization(ctx, hostPath); err != nil {
			// Could not validate - might be revoked or corrupted
			revoked++
		} else {
			valid = append(valid, hostPath)
		}
	}

	result.Valid = len(valid)
	result.Revoked = revoked

	// Step 3: Remove unauthorized materializations (revoked, expired, or corrupted)
	removed, err := rm.RemoveUnauthorizedMaterializations(ctx, discovered)
	if err != nil {
		return nil, fmt.Errorf("cleanup unauthorized: %w", err)
	}

	result.Removed = removed
	result.CompletionTime = time.Now().UnixNano()

	rm.mu.Lock()
	rm.lastResult = result
	rm.mu.Unlock()

	return result, nil
}

// ReconcilePartitionRecovery handles the case where agent reconnects after network partition
func (rm *ReconciliationManager) ReconcilePartitionRecovery(ctx context.Context) (*ReconciliationResult, error) {
	// On reconnection:
	// 1. Validate all active materializations against current control-plane state
	// 2. Check if any have been revoked during partition
	// 3. Check if any have been rotated (generation changed)
	// 4. Update lease validity windows
	// 5. Remove any that are no longer authorized

	result, err := rm.ReconcileOnRestart(ctx)
	if err != nil {
		return nil, fmt.Errorf("partition recovery: %w", err)
	}

	return result, nil
}

// LastReconciliationResult returns the result of the last reconciliation
func (rm *ReconciliationManager) LastReconciliationResult() *ReconciliationResult {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.lastResult
}

// VerifyNoOrphans checks that no orphaned materializations exist
// Used in tests to verify cleanup is complete
func (rm *ReconciliationManager) VerifyNoOrphans(ctx context.Context) (int, error) {
	discovered, err := rm.DiscoverActiveMaterializations(ctx)
	if err != nil {
		return 0, err
	}

	return len(discovered), nil
}

// DiscoverOrphanMounts returns a list of orphaned mount points (materializations without valid authorization)
func (rm *ReconciliationManager) DiscoverOrphanMounts(ctx context.Context) ([]string, error) {
	discovered, err := rm.DiscoverActiveMaterializations(ctx)
	if err != nil {
		return nil, err
	}

	orphans := []string{}
	for _, hostPath := range discovered {
		if err := rm.ValidateMaterialization(ctx, hostPath); err != nil {
			orphans = append(orphans, hostPath)
		}
	}

	return orphans, nil
}
