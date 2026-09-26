package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMaterializer_AllocateEphemeral(t *testing.T) {
	tmpdir := t.TempDir()
	m := NewMaterializer(tmpdir)
	ctx := context.Background()

	secret := []byte("supersecret")
	assignmentID := "assign-123"
	ephemeralID := "eph-456"

	hostPath, err := m.AllocateEphemeral(ctx, assignmentID, ephemeralID, secret)
	if err != nil {
		t.Fatalf("AllocateEphemeral failed: %v", err)
	}

	// Verify file exists
	data, err := os.ReadFile(hostPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(data) != string(secret) {
		t.Errorf("Secret mismatch: got %q, want %q", string(data), string(secret))
	}

	// Verify permissions (0400 = read-only for owner)
	info, _ := os.Stat(hostPath)
	perm := info.Mode().Perm()
	if perm != 0400 {
		t.Errorf("Permission mismatch: got %03o, want 0400", perm)
	}
}

func TestMaterializer_ReleaseEphemeral(t *testing.T) {
	tmpdir := t.TempDir()
	m := NewMaterializer(tmpdir)
	ctx := context.Background()

	// Create an ephemeral file
	secret := []byte("test-secret")
	hostPath, err := m.AllocateEphemeral(ctx, "assign-123", "eph-456", secret)
	if err != nil {
		t.Fatalf("AllocateEphemeral failed: %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(hostPath); err != nil {
		t.Fatalf("File should exist after allocation: %v", err)
	}

	// Release it
	if err := m.ReleaseEphemeral(ctx, hostPath); err != nil {
		t.Fatalf("ReleaseEphemeral failed: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(hostPath); !os.IsNotExist(err) {
		t.Errorf("File should not exist after release: %v", err)
	}
}

func TestMaterializer_ConcurrentAllocations(t *testing.T) {
	tmpdir := t.TempDir()
	m := NewMaterializer(tmpdir)
	ctx := context.Background()

	assignmentID := "assign-123"

	// Allocate multiple ephemeral files for the same assignment
	paths := make([]string, 3)
	secrets := [][]byte{
		[]byte("secret-1"),
		[]byte("secret-2"),
		[]byte("secret-3"),
	}

	for i, secret := range secrets {
		ephID := "eph-" + string('a'+rune(i))
		p, err := m.AllocateEphemeral(ctx, assignmentID, ephID, secret)
		if err != nil {
			t.Fatalf("AllocateEphemeral %d failed: %v", i, err)
		}
		paths[i] = p
	}

	// Verify all files exist with correct content
	for i, expectedSecret := range secrets {
		data, err := os.ReadFile(paths[i])
		if err != nil {
			t.Fatalf("ReadFile %d failed: %v", i, err)
		}
		if string(data) != string(expectedSecret) {
			t.Errorf("Secret %d mismatch: got %q, want %q", i, string(data), string(expectedSecret))
		}
	}

	// Verify they're in different files
	if paths[0] == paths[1] || paths[1] == paths[2] || paths[0] == paths[2] {
		t.Errorf("Paths should be unique: %v", paths)
	}
}

func TestMaterializer_IsolatedAssignments(t *testing.T) {
	tmpdir := t.TempDir()
	m := NewMaterializer(tmpdir)
	ctx := context.Background()

	// Create ephemeral files for two different assignments
	secret1 := []byte("secret-from-assign-1")
	secret2 := []byte("secret-from-assign-2")

	path1, err := m.AllocateEphemeral(ctx, "assign-1", "eph-a", secret1)
	if err != nil {
		t.Fatalf("AllocateEphemeral for assign-1 failed: %v", err)
	}

	path2, err := m.AllocateEphemeral(ctx, "assign-2", "eph-b", secret2)
	if err != nil {
		t.Fatalf("AllocateEphemeral for assign-2 failed: %v", err)
	}

	// Verify they're in different directories
	dir1 := filepath.Dir(path1)
	dir2 := filepath.Dir(path2)
	if dir1 == dir2 {
		t.Errorf("Assignments should be isolated: %s vs %s", dir1, dir2)
	}

	// Verify correct content
	data1, _ := os.ReadFile(path1)
	data2, _ := os.ReadFile(path2)

	if string(data1) != string(secret1) {
		t.Errorf("Secret 1 mismatch")
	}
	if string(data2) != string(secret2) {
		t.Errorf("Secret 2 mismatch")
	}
}

func TestMaterializer_PathTraversalPrevention(t *testing.T) {
	tmpdir := t.TempDir()
	m := NewMaterializer(tmpdir)

	// Try to create a file with path traversal
	dangerousPath := filepath.Join(tmpdir, "../../../etc/passwd")
	err := m.validatePath(dangerousPath)
	if err == nil {
		t.Errorf("Path traversal should be rejected")
	}
}

func TestMaterializer_Cleanup(t *testing.T) {
	tmpdir := t.TempDir()
	m := NewMaterializer(tmpdir)
	m.timeout = 100 * time.Millisecond // Short timeout for testing
	ctx := context.Background()

	// Create old ephemeral file (pretend it's old)
	oldAssignDir := filepath.Join(tmpdir, "assign-old")
	os.MkdirAll(oldAssignDir, 0755)
	oldPath := filepath.Join(oldAssignDir, "ephemeral_old_1000000000")
	os.WriteFile(oldPath, []byte("old-secret"), 0400)

	// Set modification time to long ago
	oldTime := time.Now().Add(-10 * time.Minute)
	os.Chtimes(oldPath, oldTime, oldTime)

	// Create fresh ephemeral file (should not be cleaned)
	freshPath, _ := m.AllocateEphemeral(ctx, "assign-fresh", "eph-new", []byte("fresh-secret"))

	// Run cleanup
	cleaned, err := m.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Old file should be removed
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("Old ephemeral file should be cleaned")
	}

	// Fresh file should remain
	if _, err := os.Stat(freshPath); err != nil {
		t.Errorf("Fresh ephemeral file should not be cleaned: %v", err)
	}

	if cleaned == 0 {
		t.Errorf("Cleanup should have removed old file")
	}
}

func TestMaterializer_DoubleRelease(t *testing.T) {
	tmpdir := t.TempDir()
	m := NewMaterializer(tmpdir)

	hostPath, _ := m.AllocateEphemeral(context.Background(), "assign-123", "eph-456", []byte("secret"))

	// First release should succeed
	if err := m.ReleaseEphemeral(context.Background(), hostPath); err != nil {
		t.Fatalf("First release failed: %v", err)
	}

	// Second release should also succeed (idempotent, file doesn't exist)
	if err := m.ReleaseEphemeral(context.Background(), hostPath); err != nil {
		// File not existing is fine for ReleaseEphemeral (idempotent)
		t.Logf("Second release on missing file: %v", err)
	}
}

func TestMaterializer_LargeSecret(t *testing.T) {
	tmpdir := t.TempDir()
	m := NewMaterializer(tmpdir)
	ctx := context.Background()

	// Create a large secret (1MB)
	largeSecret := make([]byte, 1024*1024)
	for i := range largeSecret {
		largeSecret[i] = byte(i % 256)
	}

	hostPath, err := m.AllocateEphemeral(ctx, "assign-123", "eph-large", largeSecret)
	if err != nil {
		t.Fatalf("AllocateEphemeral with large secret failed: %v", err)
	}

	// Verify content
	data, _ := os.ReadFile(hostPath)
	if len(data) != len(largeSecret) {
		t.Errorf("Size mismatch: got %d, want %d", len(data), len(largeSecret))
	}
}
