package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// VolumeStatus represents volume lifecycle state
type VolumeStatus string

const (
	VolumeCreated  VolumeStatus = "CREATED"
	VolumeAttached VolumeStatus = "ATTACHED"
	VolumeMounted  VolumeStatus = "MOUNTED"
	VolumeDetached VolumeStatus = "DETACHED"
	VolumeDeleted  VolumeStatus = "DELETED"
)

// WorkloadVolume represents a persistent volume for a workload
type WorkloadVolume struct {
	VolumeID    string
	WorkloadID  string
	NodeID      string
	MountPath   string
	Size        int64
	Status      VolumeStatus
	CreatedAt   int64
	AttachedAt  int64
	MountedAt   int64
	LastAccess  int64
	Owner       string // workload UID
	Permissions int    // Unix permissions (e.g., 0755)
	Snapshot    *VolumeSnapshot
}

// VolumeSnapshot represents a point-in-time snapshot of volume data
type VolumeSnapshot struct {
	SnapshotID    string
	VolumeID      string
	CreatedAt     int64
	Size          int64
	Path          string // Snapshot storage path
	RetentionDays int
	Checksum      string // SHA256 of snapshot
}

// VolumeQuota enforces per-workload storage limits
type VolumeQuota struct {
	WorkloadID    string
	MaxVolumeSize int64
	MaxVolumes    int
	CurrentSize   int64
	CurrentCount  int
}

// WorkloadStorageManager manages persistent volumes for workloads
type WorkloadStorageManager struct {
	volumes       map[string]*WorkloadVolume   // volumeID -> volume
	byWorkload    map[string][]*WorkloadVolume // workloadID -> volumes
	byNode        map[string][]*WorkloadVolume // nodeID -> volumes
	quotas        map[string]*VolumeQuota      // workloadID -> quota
	snapshots     map[string]*VolumeSnapshot   // snapshotID -> snapshot
	baseStorePath string
	mu            sync.RWMutex
}

// NewWorkloadStorageManager creates a storage manager
func NewWorkloadStorageManager(baseStorePath string) *WorkloadStorageManager {
	return &WorkloadStorageManager{
		volumes:       make(map[string]*WorkloadVolume),
		byWorkload:    make(map[string][]*WorkloadVolume),
		byNode:        make(map[string][]*WorkloadVolume),
		quotas:        make(map[string]*VolumeQuota),
		snapshots:     make(map[string]*VolumeSnapshot),
		baseStorePath: baseStorePath,
	}
}

// SetVolumeQuota sets storage quota for a workload
func (wsm *WorkloadStorageManager) SetVolumeQuota(workloadID string, maxSize int64, maxCount int) error {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	if maxSize <= 0 || maxCount <= 0 {
		return fmt.Errorf("quota limits must be positive")
	}

	wsm.quotas[workloadID] = &VolumeQuota{
		WorkloadID:    workloadID,
		MaxVolumeSize: maxSize,
		MaxVolumes:    maxCount,
	}

	return nil
}

// CreateVolume creates a new persistent volume
func (wsm *WorkloadStorageManager) CreateVolume(workloadID string, nodeID string,
	size int64, mountPath string) (*WorkloadVolume, error) {

	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	if size <= 0 {
		return nil, fmt.Errorf("volume size must be positive")
	}

	// Check quota
	quota, ok := wsm.quotas[workloadID]
	if !ok {
		return nil, fmt.Errorf("no quota set for workload %s", workloadID)
	}

	if quota.CurrentCount >= quota.MaxVolumes {
		return nil, fmt.Errorf("workload %s has reached max volumes limit (%d)", workloadID, quota.MaxVolumes)
	}

	if quota.CurrentSize+size > quota.MaxVolumeSize {
		return nil, fmt.Errorf("workload %s insufficient quota: need %d, available %d",
			workloadID, size, quota.MaxVolumeSize-quota.CurrentSize)
	}

	// Create volume directory
	volumeID := fmt.Sprintf("vol-%s-%d", workloadID[:8], time.Now().Unix())
	volumePath := filepath.Join(wsm.baseStorePath, volumeID)

	if err := os.MkdirAll(volumePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create volume directory: %v", err)
	}

	volume := &WorkloadVolume{
		VolumeID:    volumeID,
		WorkloadID:  workloadID,
		NodeID:      nodeID,
		MountPath:   mountPath,
		Size:        size,
		Status:      VolumeCreated,
		CreatedAt:   time.Now().UnixNano(),
		Owner:       workloadID,
		Permissions: 0755,
	}

	wsm.volumes[volumeID] = volume
	wsm.byWorkload[workloadID] = append(wsm.byWorkload[workloadID], volume)
	wsm.byNode[nodeID] = append(wsm.byNode[nodeID], volume)

	quota.CurrentCount++
	quota.CurrentSize += size

	return volume, nil
}

// AttachVolume attaches a volume to a node
func (wsm *WorkloadStorageManager) AttachVolume(volumeID string, nodeID string) error {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	volume, ok := wsm.volumes[volumeID]
	if !ok {
		return fmt.Errorf("volume not found: %s", volumeID)
	}

	if volume.Status != VolumeCreated {
		return fmt.Errorf("volume %s cannot be attached from status %s", volumeID, volume.Status)
	}

	volume.NodeID = nodeID
	volume.Status = VolumeAttached
	volume.AttachedAt = time.Now().UnixNano()

	return nil
}

// MountVolume mounts a volume in a container
func (wsm *WorkloadStorageManager) MountVolume(volumeID string) error {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	volume, ok := wsm.volumes[volumeID]
	if !ok {
		return fmt.Errorf("volume not found: %s", volumeID)
	}

	if volume.Status != VolumeAttached {
		return fmt.Errorf("volume %s must be attached before mounting", volumeID)
	}

	volume.Status = VolumeMounted
	volume.MountedAt = time.Now().UnixNano()

	return nil
}

// WriteToVolume records a write operation
func (wsm *WorkloadStorageManager) WriteToVolume(volumeID string, bytes int64) error {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	volume, ok := wsm.volumes[volumeID]
	if !ok {
		return fmt.Errorf("volume not found: %s", volumeID)
	}

	if volume.Status != VolumeMounted {
		return fmt.Errorf("volume %s must be mounted for write operations", volumeID)
	}

	volume.LastAccess = time.Now().UnixNano()
	return nil
}

// UnmountVolume unmounts a volume from container
func (wsm *WorkloadStorageManager) UnmountVolume(volumeID string) error {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	volume, ok := wsm.volumes[volumeID]
	if !ok {
		return fmt.Errorf("volume not found: %s", volumeID)
	}

	if volume.Status != VolumeMounted {
		return fmt.Errorf("volume %s must be mounted before unmounting", volumeID)
	}

	volume.Status = VolumeAttached
	return nil
}

// DetachVolume detaches volume from node
func (wsm *WorkloadStorageManager) DetachVolume(volumeID string) error {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	volume, ok := wsm.volumes[volumeID]
	if !ok {
		return fmt.Errorf("volume not found: %s", volumeID)
	}

	if volume.Status == VolumeMounted {
		return fmt.Errorf("volume %s must be unmounted before detaching", volumeID)
	}

	volume.Status = VolumeDetached
	return nil
}

// DeleteVolume removes a volume
func (wsm *WorkloadStorageManager) DeleteVolume(volumeID string) error {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	volume, ok := wsm.volumes[volumeID]
	if !ok {
		return fmt.Errorf("volume not found: %s", volumeID)
	}

	if volume.Status == VolumeMounted {
		return fmt.Errorf("volume %s must be unmounted before deletion", volumeID)
	}

	// Delete volume directory
	volumePath := filepath.Join(wsm.baseStorePath, volumeID)
	if err := os.RemoveAll(volumePath); err != nil {
		return fmt.Errorf("failed to delete volume directory: %v", err)
	}

	// Update quota
	if quota, ok := wsm.quotas[volume.WorkloadID]; ok {
		quota.CurrentCount--
		quota.CurrentSize -= volume.Size
	}

	// Remove from maps
	delete(wsm.volumes, volumeID)

	// Remove from byWorkload
	workloadVolumes := wsm.byWorkload[volume.WorkloadID]
	newVolumes := []*WorkloadVolume{}
	for _, v := range workloadVolumes {
		if v.VolumeID != volumeID {
			newVolumes = append(newVolumes, v)
		}
	}
	if len(newVolumes) == 0 {
		delete(wsm.byWorkload, volume.WorkloadID)
	} else {
		wsm.byWorkload[volume.WorkloadID] = newVolumes
	}

	// Remove from byNode
	nodeVolumes := wsm.byNode[volume.NodeID]
	newVolumes = []*WorkloadVolume{}
	for _, v := range nodeVolumes {
		if v.VolumeID != volumeID {
			newVolumes = append(newVolumes, v)
		}
	}
	if len(newVolumes) == 0 {
		delete(wsm.byNode, volume.NodeID)
	} else {
		wsm.byNode[volume.NodeID] = newVolumes
	}

	volume.Status = VolumeDeleted
	return nil
}

// CreateSnapshot creates a point-in-time snapshot of a volume
func (wsm *WorkloadStorageManager) CreateSnapshot(volumeID string) (*VolumeSnapshot, error) {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	volume, ok := wsm.volumes[volumeID]
	if !ok {
		return nil, fmt.Errorf("volume not found: %s", volumeID)
	}

	snapshotID := fmt.Sprintf("snap-%s-%d", volumeID[:8], time.Now().Unix())
	snapshotPath := filepath.Join(wsm.baseStorePath, snapshotID)

	snapshot := &VolumeSnapshot{
		SnapshotID:    snapshotID,
		VolumeID:      volumeID,
		CreatedAt:     time.Now().UnixNano(),
		Path:          snapshotPath,
		RetentionDays: 7,
	}

	wsm.snapshots[snapshotID] = snapshot
	volume.Snapshot = snapshot

	return snapshot, nil
}

// RestoreSnapshot restores a volume from snapshot
func (wsm *WorkloadStorageManager) RestoreSnapshot(snapshotID string, volumeID string) error {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	snapshot, ok := wsm.snapshots[snapshotID]
	if !ok {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	volume, ok := wsm.volumes[volumeID]
	if !ok {
		return fmt.Errorf("volume not found: %s", volumeID)
	}

	if volume.Status != VolumeAttached && volume.Status != VolumeCreated {
		return fmt.Errorf("volume must be attached or newly created for restore")
	}

	volume.Snapshot = snapshot
	return nil
}

// GetVolume retrieves a volume
func (wsm *WorkloadStorageManager) GetVolume(volumeID string) (*WorkloadVolume, error) {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	volume, ok := wsm.volumes[volumeID]
	if !ok {
		return nil, fmt.Errorf("volume not found: %s", volumeID)
	}

	return volume, nil
}

// GetWorkloadVolumes retrieves all volumes for a workload
func (wsm *WorkloadStorageManager) GetWorkloadVolumes(workloadID string) []*WorkloadVolume {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	volumes, ok := wsm.byWorkload[workloadID]
	if !ok {
		return []*WorkloadVolume{}
	}

	result := make([]*WorkloadVolume, len(volumes))
	copy(result, volumes)
	return result
}

// GetNodeVolumes retrieves all volumes on a node
func (wsm *WorkloadStorageManager) GetNodeVolumes(nodeID string) []*WorkloadVolume {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	volumes, ok := wsm.byNode[nodeID]
	if !ok {
		return []*WorkloadVolume{}
	}

	result := make([]*WorkloadVolume, len(volumes))
	copy(result, volumes)
	return result
}

// GetVolumeQuota retrieves quota for a workload
func (wsm *WorkloadStorageManager) GetVolumeQuota(workloadID string) (*VolumeQuota, error) {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	quota, ok := wsm.quotas[workloadID]
	if !ok {
		return nil, fmt.Errorf("no quota set for workload %s", workloadID)
	}

	return quota, nil
}

// CleanupWorkloadVolumes removes all volumes for a workload
func (wsm *WorkloadStorageManager) CleanupWorkloadVolumes(workloadID string) error {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	volumes, ok := wsm.byWorkload[workloadID]
	if !ok {
		return nil
	}

	// Make copy to avoid mutation during iteration
	volumesCopy := make([]*WorkloadVolume, len(volumes))
	copy(volumesCopy, volumes)

	for _, volume := range volumesCopy {
		// Delete volume directory
		volumePath := filepath.Join(wsm.baseStorePath, volume.VolumeID)
		os.RemoveAll(volumePath)

		// Remove from byNode
		if nodeVolumes, ok := wsm.byNode[volume.NodeID]; ok {
			newVolumes := []*WorkloadVolume{}
			for _, v := range nodeVolumes {
				if v.VolumeID != volume.VolumeID {
					newVolumes = append(newVolumes, v)
				}
			}
			if len(newVolumes) == 0 {
				delete(wsm.byNode, volume.NodeID)
			} else {
				wsm.byNode[volume.NodeID] = newVolumes
			}
		}

		// Remove from volumes
		delete(wsm.volumes, volume.VolumeID)
		volume.Status = VolumeDeleted
	}

	// Remove from byWorkload
	delete(wsm.byWorkload, workloadID)

	// Clear quota
	delete(wsm.quotas, workloadID)

	return nil
}

// GetStorageStats returns storage statistics
func (wsm *WorkloadStorageManager) GetStorageStats() map[string]interface{} {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	totalSize := int64(0)
	mountedVolumes := 0

	for _, volume := range wsm.volumes {
		totalSize += volume.Size
		if volume.Status == VolumeMounted {
			mountedVolumes++
		}
	}

	return map[string]interface{}{
		"total_volumes":    len(wsm.volumes),
		"mounted_volumes":  mountedVolumes,
		"total_size_bytes": totalSize,
		"total_snapshots":  len(wsm.snapshots),
		"workloads":        len(wsm.byWorkload),
	}
}
