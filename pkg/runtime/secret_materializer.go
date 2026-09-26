package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Materializer handles ephemeral tmpfs allocation and lifecycle for secrets.
type Materializer struct {
	basePath string
	timeout  time.Duration
}

// NewMaterializer creates a new secret materializer.
// basePath should be /var/run/secrets or similar (must exist and be writable).
func NewMaterializer(basePath string) *Materializer {
	return &Materializer{
		basePath: basePath,
		timeout:  5 * time.Minute,
	}
}

// AllocateEphemeral creates an ephemeral secret file on the host.
// Returns the host path where the secret was written.
//
// The file is created with restrictive permissions (0400) and is intended
// to be mounted into a workload namespace as read-only.
func (m *Materializer) AllocateEphemeral(ctx context.Context,
	assignmentID, ephemeralID string,
	plaintext []byte) (hostPath string, err error) {

	// Create assignment directory if needed
	assignDir := filepath.Join(m.basePath, assignmentID)
	if err := os.MkdirAll(assignDir, 0755); err != nil {
		return "", fmt.Errorf("create assignment directory: %w", err)
	}

	// Create ephemeral file path (timestamp ensures uniqueness)
	ts := time.Now().Unix()
	ephemeralFile := filepath.Join(assignDir,
		fmt.Sprintf("ephemeral_%s_%d", ephemeralID, ts))

	// Verify path is within base (prevent directory traversal)
	if err := m.validatePath(ephemeralFile); err != nil {
		return "", fmt.Errorf("path validation failed: %w", err)
	}

	// Write secret to file with restrictive permissions
	if err := os.WriteFile(ephemeralFile, plaintext, 0400); err != nil {
		return "", fmt.Errorf("write ephemeral file: %w", err)
	}

	return ephemeralFile, nil
}

// ReleaseEphemeral removes an ephemeral secret file and verifies cleanup.
// Returns error if the file still exists after removal attempt.
func (m *Materializer) ReleaseEphemeral(ctx context.Context,
	hostPath string) error {

	// Verify path is within base (prevent directory traversal)
	if err := m.validatePath(hostPath); err != nil {
		return fmt.Errorf("path validation failed: %w", err)
	}

	// Remove the file
	if err := os.Remove(hostPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove ephemeral file: %w", err)
	}

	// Verify removal (stat should return ENOENT)
	if _, err := os.Stat(hostPath); !os.IsNotExist(err) {
		return fmt.Errorf("ephemeral file still exists after removal: %s", hostPath)
	}

	return nil
}

// validatePath ensures the target path is within the base directory
// (prevents directory traversal attacks).
func (m *Materializer) validatePath(targetPath string) error {
	absBase, err := filepath.Abs(m.basePath)
	if err != nil {
		return fmt.Errorf("resolve base path: %w", err)
	}

	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}

	// Check if target is within base
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return fmt.Errorf("compute relative path: %w", err)
	}

	// If rel starts with "..", it's outside base
	if filepath.IsAbs(rel) || rel == ".." || len(rel) > 2 && rel[:3] == "../" {
		return fmt.Errorf("path escapes base directory: %s", targetPath)
	}

	return nil
}

// Cleanup removes stale ephemeral files older than the timeout.
// Called by control plane background goroutine to handle agent crashes.
// Returns count of cleaned files and any error.
func (m *Materializer) Cleanup(ctx context.Context) (int, error) {
	cleaned := 0
	cutoff := time.Now().Add(-m.timeout)

	entries, err := os.ReadDir(m.basePath)
	if err != nil {
		return 0, fmt.Errorf("read base directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		assignDir := filepath.Join(m.basePath, entry.Name())
		ephemeralEntries, err := os.ReadDir(assignDir)
		if err != nil {
			continue // Skip on read error, will retry next cycle
		}

		for _, ephemeralEntry := range ephemeralEntries {
			if !ephemeralEntry.Type().IsRegular() {
				continue
			}

			// Only process ephemeral_ files
			if ephemeralEntry.Name()[:min(9, len(ephemeralEntry.Name()))] != "ephemeral" {
				continue
			}

			info, err := ephemeralEntry.Info()
			if err != nil {
				continue
			}

			// Remove if older than timeout
			if info.ModTime().Before(cutoff) {
				filePath := filepath.Join(assignDir, ephemeralEntry.Name())
				if err := os.Remove(filePath); err == nil {
					cleaned++
				}
			}
		}
	}

	return cleaned, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
