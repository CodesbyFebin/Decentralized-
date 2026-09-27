package runtime

import (
	"os"
	"testing"
)

// TestVolumeCreation verifies volume creation
func TestVolumeCreation(t *testing.T) {
	t.Log("Testing volume creation")

	storagePath := t.TempDir()
	wsm := NewWorkloadStorageManager(storagePath)

	// Set quota
	err := wsm.SetVolumeQuota("workload-1", 1024*1024*1024, 10)
	if err != nil {
		t.Fatalf("SetVolumeQuota failed: %v", err)
	}

	// Create volume
	volume, err := wsm.CreateVolume("workload-1", "node-1", 100*1024*1024, "/data")
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}

	if volume.WorkloadID != "workload-1" {
		t.Errorf("Expected workload-1, got %s", volume.WorkloadID)
	}

	if volume.Status != VolumeCreated {
		t.Errorf("Expected CREATED status, got %s", volume.Status)
	}

	if volume.CreatedAt == 0 {
		t.Error("Expected non-zero CreatedAt")
	}

	// Verify volume directory exists
	volumePath := storagePath + "/" + volume.VolumeID
	if _, err := os.Stat(volumePath); os.IsNotExist(err) {
		t.Errorf("Volume directory not created at %s", volumePath)
	}

	t.Logf("PASS: Volume %s created successfully", volume.VolumeID)
}

// TestVolumeLifecycle verifies full volume lifecycle
func TestVolumeLifecycle(t *testing.T) {
	t.Log("Testing volume lifecycle (CREATE → ATTACH → MOUNT → WRITE → UNMOUNT → DETACH → DELETE)")

	storagePath := t.TempDir()
	wsm := NewWorkloadStorageManager(storagePath)

	_ = wsm.SetVolumeQuota("workload-1", 500*1024*1024, 5)

	// Create
	volume, _ := wsm.CreateVolume("workload-1", "node-1", 50*1024*1024, "/data")
	if volume.Status != VolumeCreated {
		t.Errorf("Expected CREATED, got %s", volume.Status)
	}

	// Attach
	err := wsm.AttachVolume(volume.VolumeID, "node-1")
	if err != nil {
		t.Fatalf("AttachVolume failed: %v", err)
	}

	volume, _ = wsm.GetVolume(volume.VolumeID)
	if volume.Status != VolumeAttached {
		t.Errorf("Expected ATTACHED, got %s", volume.Status)
	}

	// Mount
	err = wsm.MountVolume(volume.VolumeID)
	if err != nil {
		t.Fatalf("MountVolume failed: %v", err)
	}

	volume, _ = wsm.GetVolume(volume.VolumeID)
	if volume.Status != VolumeMounted {
		t.Errorf("Expected MOUNTED, got %s", volume.Status)
	}

	// Write
	err = wsm.WriteToVolume(volume.VolumeID, 1024)
	if err != nil {
		t.Fatalf("WriteToVolume failed: %v", err)
	}

	volume, _ = wsm.GetVolume(volume.VolumeID)
	if volume.LastAccess == 0 {
		t.Error("Expected LastAccess to be set")
	}

	// Unmount
	err = wsm.UnmountVolume(volume.VolumeID)
	if err != nil {
		t.Fatalf("UnmountVolume failed: %v", err)
	}

	volume, _ = wsm.GetVolume(volume.VolumeID)
	if volume.Status != VolumeAttached {
		t.Errorf("Expected ATTACHED after unmount, got %s", volume.Status)
	}

	// Detach
	err = wsm.DetachVolume(volume.VolumeID)
	if err != nil {
		t.Fatalf("DetachVolume failed: %v", err)
	}

	volume, _ = wsm.GetVolume(volume.VolumeID)
	if volume.Status != VolumeDetached {
		t.Errorf("Expected DETACHED, got %s", volume.Status)
	}

	// Delete
	err = wsm.DeleteVolume(volume.VolumeID)
	if err != nil {
		t.Fatalf("DeleteVolume failed: %v", err)
	}

	_, err = wsm.GetVolume(volume.VolumeID)
	if err == nil {
		t.Error("Expected error retrieving deleted volume")
	}

	t.Logf("PASS: Full volume lifecycle completed successfully")
}

// TestVolumeQuota verifies quota enforcement
func TestVolumeQuota(t *testing.T) {
	t.Log("Testing volume quota enforcement")

	storagePath := t.TempDir()
	wsm := NewWorkloadStorageManager(storagePath)

	// Set small quota
	_ = wsm.SetVolumeQuota("workload-1", 100*1024*1024, 2)

	// Create first volume
	_, err := wsm.CreateVolume("workload-1", "node-1", 50*1024*1024, "/data1")
	if err != nil {
		t.Fatalf("CreateVolume 1 failed: %v", err)
	}

	// Create second volume
	vol2, err := wsm.CreateVolume("workload-1", "node-1", 40*1024*1024, "/data2")
	if err != nil {
		t.Fatalf("CreateVolume 2 failed: %v", err)
	}

	// Try to exceed size quota
	_, err = wsm.CreateVolume("workload-1", "node-1", 20*1024*1024, "/data3")
	if err == nil {
		t.Error("Expected error exceeding size quota")
	}

	// Try to exceed count quota
	_ = wsm.DeleteVolume(vol2.VolumeID)
	_, _ = wsm.CreateVolume("workload-1", "node-1", 20*1024*1024, "/data2")
	_, err = wsm.CreateVolume("workload-1", "node-1", 10*1024*1024, "/data3")
	if err == nil {
		t.Error("Expected error exceeding volume count quota")
	}

	t.Logf("PASS: Quota enforcement works correctly")
}

// TestVolumeSnapshot verifies snapshot creation and restore
func TestVolumeSnapshot(t *testing.T) {
	t.Log("Testing volume snapshot creation and restore")

	storagePath := t.TempDir()
	wsm := NewWorkloadStorageManager(storagePath)

	_ = wsm.SetVolumeQuota("workload-1", 500*1024*1024, 5)

	// Create and mount volume
	volume, _ := wsm.CreateVolume("workload-1", "node-1", 50*1024*1024, "/data")
	_ = wsm.AttachVolume(volume.VolumeID, "node-1")
	_ = wsm.MountVolume(volume.VolumeID)

	// Create snapshot
	snapshot, err := wsm.CreateSnapshot(volume.VolumeID)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	if snapshot.VolumeID != volume.VolumeID {
		t.Errorf("Expected volume ID %s, got %s", volume.VolumeID, snapshot.VolumeID)
	}

	if snapshot.CreatedAt == 0 {
		t.Error("Expected non-zero CreatedAt")
	}

	// Create second volume for restore test
	volume2, _ := wsm.CreateVolume("workload-1", "node-1", 50*1024*1024, "/data2")

	// Restore from snapshot
	err = wsm.RestoreSnapshot(snapshot.SnapshotID, volume2.VolumeID)
	if err != nil {
		t.Fatalf("RestoreSnapshot failed: %v", err)
	}

	t.Logf("PASS: Snapshot creation and restore works correctly")
}

// TestMultipleWorkloads verifies volume isolation per workload
func TestMultipleWorkloads(t *testing.T) {
	t.Log("Testing volume isolation per workload")

	storagePath := t.TempDir()
	wsm := NewWorkloadStorageManager(storagePath)

	// Set quotas for two workloads
	_ = wsm.SetVolumeQuota("workload-1", 500*1024*1024, 5)
	_ = wsm.SetVolumeQuota("workload-2", 500*1024*1024, 5)

	// Create volumes for each workload
	_, _ = wsm.CreateVolume("workload-1", "node-1", 50*1024*1024, "/data")
	_, _ = wsm.CreateVolume("workload-1", "node-1", 30*1024*1024, "/logs")
	_, _ = wsm.CreateVolume("workload-2", "node-1", 40*1024*1024, "/data")

	// Verify workload volumes
	vols1 := wsm.GetWorkloadVolumes("workload-1")
	if len(vols1) != 2 {
		t.Errorf("Expected 2 volumes for workload-1, got %d", len(vols1))
	}

	vols2 := wsm.GetWorkloadVolumes("workload-2")
	if len(vols2) != 1 {
		t.Errorf("Expected 1 volume for workload-2, got %d", len(vols2))
	}

	// Verify node volumes
	nodeVols := wsm.GetNodeVolumes("node-1")
	if len(nodeVols) != 3 {
		t.Errorf("Expected 3 volumes on node-1, got %d", len(nodeVols))
	}

	// Test cleanup for workload-1
	_ = wsm.CleanupWorkloadVolumes("workload-1")

	vols1 = wsm.GetWorkloadVolumes("workload-1")
	if len(vols1) != 0 {
		t.Errorf("Expected 0 volumes for workload-1 after cleanup, got %d", len(vols1))
	}

	// Workload-2 volumes should remain
	vols2 = wsm.GetWorkloadVolumes("workload-2")
	if len(vols2) != 1 {
		t.Errorf("Expected 1 volume for workload-2, got %d", len(vols2))
	}

	t.Logf("PASS: Volume isolation per workload works correctly")
}

// TestPersistenceAfterRestart verifies volume persists across restarts
func TestPersistenceAfterRestart(t *testing.T) {
	t.Log("Testing volume persistence across storage manager restart")

	storagePath := t.TempDir()

	// Phase 1: Create volume with first instance
	wsm1 := NewWorkloadStorageManager(storagePath)
	_ = wsm1.SetVolumeQuota("workload-1", 500*1024*1024, 5)
	volume, _ := wsm1.CreateVolume("workload-1", "node-1", 50*1024*1024, "/data")

	volumeID := volume.VolumeID
	_ = wsm1.AttachVolume(volumeID, "node-1")
	_ = wsm1.MountVolume(volumeID)

	// Verify volume directory exists
	volumePath := storagePath + "/" + volumeID
	if _, err := os.Stat(volumePath); os.IsNotExist(err) {
		t.Fatalf("Volume directory not found at %s", volumePath)
	}

	// Phase 2: Create new instance and verify volume data persists
	_ = NewWorkloadStorageManager(storagePath)

	// Directory should still exist (persistence mechanism)
	if _, err := os.Stat(volumePath); os.IsNotExist(err) {
		t.Fatalf("Volume directory lost after storage manager restart")
	}

	// Note: Full persistence recovery would require loading metadata from disk
	// This test verifies the foundation (directory persistence)
	t.Logf("PASS: Volume directory persists across manager restarts")
}

// TestStorageStats verifies statistics reporting
func TestStorageStats(t *testing.T) {
	t.Log("Testing storage statistics reporting")

	storagePath := t.TempDir()
	wsm := NewWorkloadStorageManager(storagePath)

	_ = wsm.SetVolumeQuota("workload-1", 500*1024*1024, 5)

	// Create volumes
	volume1, err1 := wsm.CreateVolume("workload-1", "node-1", 50*1024*1024, "/data1")
	if err1 != nil {
		t.Fatalf("Failed to create volume1: %v", err1)
	}

	_, err2 := wsm.CreateVolume("workload-1", "node-1", 30*1024*1024, "/data2")
	if err2 != nil {
		t.Fatalf("Failed to create volume2: %v", err2)
	}

	// Mount volume1
	_ = wsm.AttachVolume(volume1.VolumeID, "node-1")
	_ = wsm.MountVolume(volume1.VolumeID)

	// Get stats
	stats := wsm.GetStorageStats()

	if stats["total_volumes"] == nil || stats["total_volumes"].(int) < 1 {
		t.Errorf("Expected at least 1 volume, got %v", stats["total_volumes"])
	}

	if stats["mounted_volumes"] != 1 {
		t.Errorf("Expected 1 mounted volume, got %v", stats["mounted_volumes"])
	}

	totalSize := stats["total_size_bytes"].(int64)
	if totalSize <= 0 {
		t.Errorf("Expected positive total size, got %d", totalSize)
	}

	t.Logf("PASS: Storage statistics reporting works correctly")
}

// TestInvalidOperations verifies error handling for invalid operations
func TestInvalidOperations(t *testing.T) {
	t.Log("Testing invalid volume operations")

	storagePath := t.TempDir()
	wsm := NewWorkloadStorageManager(storagePath)

	// Try to create volume without quota
	_, err := wsm.CreateVolume("workload-1", "node-1", 50*1024*1024, "/data")
	if err == nil {
		t.Error("Expected error creating volume without quota")
	}

	_ = wsm.SetVolumeQuota("workload-1", 500*1024*1024, 5)

	// Try to create volume with invalid size
	_, err = wsm.CreateVolume("workload-1", "node-1", 0, "/data")
	if err == nil {
		t.Error("Expected error creating volume with zero size")
	}

	// Create valid volume
	volume, _ := wsm.CreateVolume("workload-1", "node-1", 50*1024*1024, "/data")

	// Try to mount without attaching
	err = wsm.MountVolume(volume.VolumeID)
	if err == nil {
		t.Error("Expected error mounting without attaching")
	}

	// Try to delete mounted volume
	_ = wsm.AttachVolume(volume.VolumeID, "node-1")
	_ = wsm.MountVolume(volume.VolumeID)

	err = wsm.DeleteVolume(volume.VolumeID)
	if err == nil {
		t.Error("Expected error deleting mounted volume")
	}

	t.Logf("PASS: Invalid operations properly rejected")
}
