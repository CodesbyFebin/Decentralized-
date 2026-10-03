package storage

import (
	"testing"
	"time"
)

// MockProvisioner is a test provisioner.
type MockProvisioner struct{}

func (m *MockProvisioner) Provision(pvc PersistentVolumeClaim, sc StorageClass) (PersistentVolume, error) {
	return PersistentVolume{
		PVC:          pvc.Name,
		Size:         pvc.Size,
		StorageClass: pvc.StorageClass,
		Status:       "Pending",
		Phase:        "Provisioning",
		AccessModes:  []string{pvc.AccessMode},
	}, nil
}

func (m *MockProvisioner) Delete(pv PersistentVolume) error {
	return nil
}

func (m *MockProvisioner) Expand(pv PersistentVolume, newSize string) error {
	return nil
}

func (m *MockProvisioner) CreateSnapshot(pv PersistentVolume, snapClass string) (VolumeSnapshot, error) {
	return VolumeSnapshot{
		SourceVolume:  pv.Name,
		Size:          pv.Size,
		Status:        "Pending",
		SnapshotClass: snapClass,
	}, nil
}

func (m *MockProvisioner) RestoreSnapshot(snap VolumeSnapshot, pvc PersistentVolumeClaim) (PersistentVolume, error) {
	return PersistentVolume{
		PVC:          pvc.Name,
		Size:         snap.Size,
		StorageClass: pvc.StorageClass,
		Status:       "Bound",
		Phase:        "Ready",
	}, nil
}

func TestStorageClassRegistration(t *testing.T) {
	sm := NewStorageManager()

	sc := StorageClass{
		Name:              "fast-ssd",
		Provisioner:       "local-path",
		AllowVolumeExpansion: true,
		VolumeBindingMode: "Immediate",
		ReclaimPolicy:     "Delete",
		Parameters: map[string]string{
			"path": "/mnt/fast-ssd",
		},
	}

	err := sm.RegisterStorageClass(sc)
	if err != nil {
		t.Fatalf("RegisterStorageClass failed: %v", err)
	}

	// Register provisioner
	prov := &MockProvisioner{}
	err = sm.RegisterProvisioner("fast-ssd", prov)
	if err != nil {
		t.Fatalf("RegisterProvisioner failed: %v", err)
	}

	// Retrieve and verify
	retrieved, err := sm.GetStorageClass("fast-ssd")
	if err != nil {
		t.Fatalf("GetStorageClass failed: %v", err)
	}

	if retrieved.Name != "fast-ssd" {
		t.Errorf("expected name fast-ssd, got %s", retrieved.Name)
	}
	if retrieved.VolumeBindingMode != "Immediate" {
		t.Errorf("expected Immediate binding, got %s", retrieved.VolumeBindingMode)
	}
}

func TestPVCCreation(t *testing.T) {
	sm := NewStorageManager()

	// Register storage class
	sc := StorageClass{
		Name:        "standard",
		Provisioner: "local-path",
		VolumeBindingMode: "Immediate",
		ReclaimPolicy: "Delete",
	}
	sm.RegisterStorageClass(sc)
	sm.RegisterProvisioner("standard", &MockProvisioner{})

	// Create PVC
	pvc := PersistentVolumeClaim{
		Name:         "test-claim",
		Namespace:    "default",
		Size:         "10Gi",
		StorageClass: "standard",
		AccessMode:   "ReadWriteOnce",
	}

	err := sm.CreateClaim(pvc)
	if err != nil {
		t.Fatalf("CreateClaim failed: %v", err)
	}

	// Verify
	retrieved, err := sm.GetClaim("test-claim")
	if err != nil {
		t.Fatalf("GetClaim failed: %v", err)
	}

	if retrieved.Status != "Pending" {
		t.Errorf("expected Pending status, got %s", retrieved.Status)
	}
	if retrieved.BindingMode != "Immediate" {
		t.Errorf("expected Immediate binding, got %s", retrieved.BindingMode)
	}
}

func TestPVCBinding(t *testing.T) {
	sm := NewStorageManager()

	// Setup
	sc := StorageClass{
		Name:        "fast",
		Provisioner: "local-path",
		VolumeBindingMode: "Immediate",
		ReclaimPolicy: "Retain",
	}
	sm.RegisterStorageClass(sc)
	sm.RegisterProvisioner("fast", &MockProvisioner{})

	// Create PVC
	pvc := PersistentVolumeClaim{
		Name:         "data-pvc",
		Namespace:    "default",
		Size:         "50Gi",
		StorageClass: "fast",
		AccessMode:   "ReadWriteOnce",
	}
	sm.CreateClaim(pvc)

	// Bind to node
	pv, err := sm.BindClaim("data-pvc", "node-1")
	if err != nil {
		t.Fatalf("BindClaim failed: %v", err)
	}

	// Verify PV
	if pv.Status != "Bound" {
		t.Errorf("expected Bound status, got %s", pv.Status)
	}
	if pv.Phase != "Ready" {
		t.Errorf("expected Ready phase, got %s", pv.Phase)
	}
	if pv.BoundNode != "node-1" {
		t.Errorf("expected bound to node-1, got %s", pv.BoundNode)
	}

	// Verify PVC
	retrievedPVC, _ := sm.GetClaim("data-pvc")
	if retrievedPVC.Status != "Bound" {
		t.Errorf("PVC: expected Bound status, got %s", retrievedPVC.Status)
	}
	if retrievedPVC.BoundVolume == "" {
		t.Error("PVC: expected bound volume")
	}
}

func TestVolumeRelease(t *testing.T) {
	sm := NewStorageManager()

	// Setup
	sc := StorageClass{
		Name:        "deletable",
		Provisioner: "local-path",
		VolumeBindingMode: "Immediate",
		ReclaimPolicy: "Delete",
	}
	sm.RegisterStorageClass(sc)
	sm.RegisterProvisioner("deletable", &MockProvisioner{})

	// Create and bind
	pvc := PersistentVolumeClaim{
		Name:         "temp-pvc",
		Namespace:    "default",
		Size:         "5Gi",
		StorageClass: "deletable",
		AccessMode:   "ReadWriteOnce",
	}
	sm.CreateClaim(pvc)
	pv, _ := sm.BindClaim("temp-pvc", "node-1")

	// Release
	err := sm.ReleaseClaim("temp-pvc")
	if err != nil {
		t.Fatalf("ReleaseClaim failed: %v", err)
	}

	// Verify PVC is lost
	retrieved, _ := sm.GetClaim("temp-pvc")
	if retrieved.Status != "Lost" {
		t.Errorf("expected Lost status, got %s", retrieved.Status)
	}

	// Verify PV is deleted (for Delete reclaim policy)
	_, err = sm.GetVolume(pv.Name)
	if err == nil {
		t.Error("expected volume to be deleted")
	}
}

func TestVolumeRetention(t *testing.T) {
	sm := NewStorageManager()

	// Setup with Retain policy
	sc := StorageClass{
		Name:        "retained",
		Provisioner: "local-path",
		VolumeBindingMode: "Immediate",
		ReclaimPolicy: "Retain",
	}
	sm.RegisterStorageClass(sc)
	sm.RegisterProvisioner("retained", &MockProvisioner{})

	// Create and bind
	pvc := PersistentVolumeClaim{
		Name:         "keep-pvc",
		Namespace:    "default",
		Size:         "20Gi",
		StorageClass: "retained",
		AccessMode:   "ReadWriteOnce",
	}
	sm.CreateClaim(pvc)
	pv, _ := sm.BindClaim("keep-pvc", "node-2")

	// Release
	sm.ReleaseClaim("keep-pvc")

	// Verify PV still exists with Released status
	retrieved, err := sm.GetVolume(pv.Name)
	if err != nil {
		t.Fatalf("GetVolume failed: %v", err)
	}
	if retrieved.Status != "Released" {
		t.Errorf("expected Released status, got %s", retrieved.Status)
	}
}

func TestVolumeExpansion(t *testing.T) {
	sm := NewStorageManager()

	// Setup with expansion enabled
	sc := StorageClass{
		Name:                 "expandable",
		Provisioner:          "local-path",
		AllowVolumeExpansion: true,
		VolumeBindingMode:    "Immediate",
		ReclaimPolicy:        "Delete",
	}
	sm.RegisterStorageClass(sc)
	sm.RegisterProvisioner("expandable", &MockProvisioner{})

	// Create and bind
	pvc := PersistentVolumeClaim{
		Name:         "grow-pvc",
		Namespace:    "default",
		Size:         "10Gi",
		StorageClass: "expandable",
		AccessMode:   "ReadWriteOnce",
	}
	sm.CreateClaim(pvc)
	pv, _ := sm.BindClaim("grow-pvc", "node-1")

	// Expand volume
	err := sm.ExpandVolume(pv.Name, "20Gi")
	if err != nil {
		t.Fatalf("ExpandVolume failed: %v", err)
	}

	// Verify
	retrieved, _ := sm.GetVolume(pv.Name)
	if retrieved.Size != "20Gi" {
		t.Errorf("expected size 20Gi, got %s", retrieved.Size)
	}
}

func TestVolumeSnapshot(t *testing.T) {
	sm := NewStorageManager()

	// Setup
	sc := StorageClass{
		Name:        "snappable",
		Provisioner: "local-path",
		VolumeBindingMode: "Immediate",
		ReclaimPolicy: "Delete",
	}
	sm.RegisterStorageClass(sc)
	sm.RegisterProvisioner("snappable", &MockProvisioner{})

	// Create and bind
	pvc := PersistentVolumeClaim{
		Name:         "snap-pvc",
		Namespace:    "default",
		Size:         "15Gi",
		StorageClass: "snappable",
		AccessMode:   "ReadWriteOnce",
	}
	sm.CreateClaim(pvc)
	pv, _ := sm.BindClaim("snap-pvc", "node-1")

	// Create snapshot
	snap, err := sm.CreateSnapshot(pv.Name, "default")
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	if snap.Status != "Ready" {
		t.Errorf("expected Ready status, got %s", snap.Status)
	}

	// Verify snapshot count increased
	retrieved, _ := sm.GetVolume(pv.Name)
	if retrieved.SnapshotCount != 1 {
		t.Errorf("expected 1 snapshot, got %d", retrieved.SnapshotCount)
	}
}

func TestNodeCapacity(t *testing.T) {
	sm := NewStorageManager()

	// Set node capacities
	sm.SetNodeCapacity("node-1", 1000 * 1024 * 1024 * 1024) // 1TB
	sm.SetNodeCapacity("node-2", 500 * 1024 * 1024 * 1024)   // 500GB

	// Verify
	cap1 := sm.GetNodeCapacity("node-1")
	if cap1 != 1000 * 1024 * 1024 * 1024 {
		t.Errorf("expected 1TB, got %d bytes", cap1)
	}

	cap2 := sm.GetNodeCapacity("node-2")
	if cap2 != 500 * 1024 * 1024 * 1024 {
		t.Errorf("expected 500GB, got %d bytes", cap2)
	}
}

func TestListOperations(t *testing.T) {
	sm := NewStorageManager()

	// Setup
	sc := StorageClass{
		Name:        "test",
		Provisioner: "local-path",
		VolumeBindingMode: "Immediate",
		ReclaimPolicy: "Delete",
	}
	sm.RegisterStorageClass(sc)
	sm.RegisterProvisioner("test", &MockProvisioner{})

	// Create multiple PVCs
	for i := 0; i < 3; i++ {
		pvc := PersistentVolumeClaim{
			Name:         "pvc-" + string(rune('0'+i)),
			Namespace:    "default",
			Size:         "10Gi",
			StorageClass: "test",
			AccessMode:   "ReadWriteOnce",
		}
		sm.CreateClaim(pvc)
	}

	// List
	claims := sm.ListClaims()
	if len(claims) != 3 {
		t.Errorf("expected 3 claims, got %d", len(claims))
	}

	// Bind them
	for _, claim := range claims {
		sm.BindClaim(claim.Name, "node-1")
	}

	// List volumes
	vols := sm.ListVolumes()
	if len(vols) != 3 {
		t.Errorf("expected 3 volumes, got %d", len(vols))
	}
}

func TestWaitForFirstConsumerBinding(t *testing.T) {
	sm := NewStorageManager()

	// Setup with WaitForFirstConsumer
	sc := StorageClass{
		Name:              "delayed",
		Provisioner:       "local-path",
		VolumeBindingMode: "WaitForFirstConsumer",
		ReclaimPolicy:     "Delete",
	}
	sm.RegisterStorageClass(sc)
	sm.RegisterProvisioner("delayed", &MockProvisioner{})

	// Create PVC
	pvc := PersistentVolumeClaim{
		Name:         "delayed-pvc",
		Namespace:    "default",
		Size:         "10Gi",
		StorageClass: "delayed",
		AccessMode:   "ReadWriteOnce",
	}
	sm.CreateClaim(pvc)

	// Verify BindingMode is set correctly
	retrieved, _ := sm.GetClaim("delayed-pvc")
	if retrieved.BindingMode != "WaitForFirstConsumer" {
		t.Errorf("expected WaitForFirstConsumer, got %s", retrieved.BindingMode)
	}

	// Now bind when pod is scheduled
	pv, err := sm.BindClaim("delayed-pvc", "node-3")
	if err != nil {
		t.Fatalf("BindClaim failed: %v", err)
	}

	if pv.BoundNode != "node-3" {
		t.Errorf("expected node-3, got %s", pv.BoundNode)
	}
}

func TestTimingTracking(t *testing.T) {
	sm := NewStorageManager()

	// Setup
	sc := StorageClass{
		Name:        "timed",
		Provisioner: "local-path",
		VolumeBindingMode: "Immediate",
		ReclaimPolicy: "Delete",
	}
	sm.RegisterStorageClass(sc)
	sm.RegisterProvisioner("timed", &MockProvisioner{})

	before := time.Now()

	// Create PVC
	pvc := PersistentVolumeClaim{
		Name:         "timed-pvc",
		Namespace:    "default",
		Size:         "10Gi",
		StorageClass: "timed",
		AccessMode:   "ReadWriteOnce",
	}
	sm.CreateClaim(pvc)

	retrieved, _ := sm.GetClaim("timed-pvc")
	if retrieved.CreatedAt.Before(before) {
		t.Error("CreatedAt should be after test start")
	}

	// Bind
	_, _ = sm.BindClaim("timed-pvc", "node-1")
	retrieved, _ = sm.GetClaim("timed-pvc")

	if retrieved.BoundAt.IsZero() {
		t.Error("BoundAt should be set")
	}
	if retrieved.BoundAt.Before(retrieved.CreatedAt) {
		t.Error("BoundAt should be after CreatedAt")
	}
}
