package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// LocalPathProvisioner creates volumes on local host filesystems.
// Used for development/testing and single-node deployments.
// For multi-node, prefer distributed storage (Ceph, NFS, etc.).
type LocalPathProvisioner struct {
	mu            sync.RWMutex
	basePath      string
	volumeCounter int64
}

// NewLocalPathProvisioner creates a local path provisioner.
func NewLocalPathProvisioner(basePath string) *LocalPathProvisioner {
	return &LocalPathProvisioner{
		basePath:      basePath,
		volumeCounter: 0,
	}
}

// Provision creates a new local volume.
func (lp *LocalPathProvisioner) Provision(pvc PersistentVolumeClaim, sc StorageClass) (PersistentVolume, error) {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	// Create unique directory
	lp.volumeCounter++
	volID := lp.volumeCounter
	volPath := filepath.Join(lp.basePath, fmt.Sprintf("pvc-%s-%d", pvc.Name, volID))

	// Create directory
	if err := os.MkdirAll(volPath, 0755); err != nil {
		return PersistentVolume{}, fmt.Errorf("failed to create volume directory: %v", err)
	}

	pv := PersistentVolume{
		Name:         fmt.Sprintf("local-%d", volID),
		PVC:          pvc.Name,
		Size:         pvc.Size,
		SizeBytes:    parseSize(pvc.Size),
		StorageClass: pvc.StorageClass,
		Status:       "Pending",
		Phase:        "Provisioning",
		AccessModes:  []string{pvc.AccessMode},
		CreatedAt:    time.Now(),
		LocalPath:    volPath,
	}

	return pv, nil
}

// Delete removes a local volume.
func (lp *LocalPathProvisioner) Delete(pv PersistentVolume) error {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	if pv.LocalPath == "" {
		return fmt.Errorf("no local path for volume %s", pv.Name)
	}

	if err := os.RemoveAll(pv.LocalPath); err != nil {
		return fmt.Errorf("failed to remove volume: %v", err)
	}

	return nil
}

// Expand extends a volume.
func (lp *LocalPathProvisioner) Expand(pv PersistentVolume, newSize string) error {
	// Local path volumes don't have actual size limits,
	// just track the requested size.
	return nil
}

// CreateSnapshot creates a point-in-time copy.
func (lp *LocalPathProvisioner) CreateSnapshot(pv PersistentVolume, snapClass string) (VolumeSnapshot, error) {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	if pv.LocalPath == "" {
		return VolumeSnapshot{}, fmt.Errorf("no local path for volume %s", pv.Name)
	}

	// Create snapshot directory
	snapPath := pv.LocalPath + "-snap-" + strconv.FormatInt(time.Now().Unix(), 10)
	if err := os.Mkdir(snapPath, 0755); err != nil {
		return VolumeSnapshot{}, fmt.Errorf("failed to create snapshot: %v", err)
	}

	snap := VolumeSnapshot{
		Name:          fmt.Sprintf("%s-snapshot", pv.Name),
		SourceVolume:  pv.Name,
		Size:          pv.Size,
		SizeBytes:     pv.SizeBytes,
		Status:        "Ready",
		CreatedAt:     time.Now(),
		ReadyAt:       time.Now(),
		SnapshotClass: snapClass,
	}

	return snap, nil
}

// RestoreSnapshot creates a volume from a snapshot.
func (lp *LocalPathProvisioner) RestoreSnapshot(snap VolumeSnapshot, pvc PersistentVolumeClaim) (PersistentVolume, error) {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	// Provision a new volume
	pv := PersistentVolume{
		Name:         fmt.Sprintf("restored-%d", time.Now().Unix()),
		PVC:          pvc.Name,
		Size:         snap.Size,
		SizeBytes:    snap.SizeBytes,
		StorageClass: pvc.StorageClass,
		Status:       "Bound",
		Phase:        "Ready",
		AccessModes:  []string{pvc.AccessMode},
		CreatedAt:    time.Now(),
		LocalPath:    snap.SourceVolume + "-restored",
	}

	return pv, nil
}

// parseSize converts a size string to bytes.
// Supports: 1Gi, 1G, 1Mi, 1M, 1Ki, 1K, 1B
func parseSize(size string) int64 {
	multipliers := map[string]int64{
		"G":  1000 * 1000 * 1000,
		"Gi": 1024 * 1024 * 1024,
		"M":  1000 * 1000,
		"Mi": 1024 * 1024,
		"K":  1000,
		"Ki": 1024,
		"B":  1,
	}

	for suffix, mult := range multipliers {
		if len(size) > len(suffix) && size[len(size)-len(suffix):] == suffix {
			numStr := size[:len(size)-len(suffix)]
			num, _ := strconv.ParseInt(numStr, 10, 64)
			return num * mult
		}
	}

	num, _ := strconv.ParseInt(size, 10, 64)
	return num
}
