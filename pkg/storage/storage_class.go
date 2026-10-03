// Storage class management for dynamic volume provisioning.
//
// Storage classes define provisioning policies:
//   - Provisioner: storage backend (local-path, nfs, etc.)
//   - Volume binding mode: immediate or delayed until pod placement
//   - Reclaim policy: delete, retain, recycle
//   - Size limits and quotas
//   - Snapshot support
//
// Dynamic provisioning creates volumes on-demand via PersistentVolumeClaims.
// PVC binding respects topology constraints from pod placement.
package storage

import (
	"fmt"
	"sync"
	"time"
)

// StorageClass defines a provisioning policy.
type StorageClass struct {
	Name                 string            `json:"name"`
	Provisioner          string            `json:"provisioner"`          // local-path, nfs, ceph, etc.
	Replicas             int32             `json:"replicas"`             // replication factor
	AllowVolumeExpansion bool              `json:"allow_volume_expansion"`
	VolumeBindingMode    string            `json:"volume_binding_mode"`  // Immediate | WaitForFirstConsumer
	ReclaimPolicy        string            `json:"reclaim_policy"`       // Delete | Retain | Recycle
	Parameters           map[string]string `json:"parameters"`           // provisioner-specific
	AllowedTopologies    []TopologySelector `json:"allowed_topologies"` // zone, node constraints
	MinSize              string            `json:"min_size"`
	MaxSize              string            `json:"max_size"`
	SnapshotClass        string            `json:"snapshot_class"`
}

// TopologySelector constrains PV placement.
type TopologySelector struct {
	MatchLabelExpressions []MatchLabelExpression `json:"match_label_expressions"`
}

// MatchLabelExpression matches a label to specific values.
type MatchLabelExpression struct {
	Key    string   `json:"key"`    // e.g., "kubernetes.io/hostname"
	Values []string `json:"values"` // e.g., ["node-1", "node-2"]
}

// PersistentVolume represents a provisioned storage volume.
type PersistentVolume struct {
	Name              string            `json:"name"`
	PVC               string            `json:"pvc"`                // PersistentVolumeClaim name
	Size              string            `json:"size"`               // "10Gi", "1Ti", etc.
	SizeBytes         int64             `json:"size_bytes"`
	StorageClass      string            `json:"storage_class"`
	Status            string            `json:"status"`             // Pending | Bound | Released | Failed
	Phase             string            `json:"phase"`              // Provisioning | Ready | InUse | Releasing
	AccessModes       []string          `json:"access_modes"`       // RWO, ROX, RWX
	BoundPod          string            `json:"bound_pod"`          // pod name
	BoundNode         string            `json:"bound_node"`         // node ID
	CreatedAt         time.Time         `json:"created_at"`
	LastTransition    time.Time         `json:"last_transition"`
	Reason            string            `json:"reason"`
	LocalPath         string            `json:"local_path"`         // for local provisioner
	SnapshotCount     int32             `json:"snapshot_count"`
	ReclaimPolicy     string            `json:"reclaim_policy"`
}

// PersistentVolumeClaim requests storage.
type PersistentVolumeClaim struct {
	Name          string    `json:"name"`
	Namespace     string    `json:"namespace"`     // statefulset name
	Size          string    `json:"size"`
	StorageClass  string    `json:"storage_class"`
	AccessMode    string    `json:"access_mode"`   // RWO | ROX | RWX
	Status        string    `json:"status"`        // Pending | Bound | Lost
	BoundVolume   string    `json:"bound_volume"`  // PV name
	CreatedAt     time.Time `json:"created_at"`
	BoundAt       time.Time `json:"bound_at"`
	BindingMode   string    `json:"binding_mode"`  // Immediate | WaitForFirstConsumer
	BoundToNode   string    `json:"bound_to_node"` // node ID (for WaitForFirstConsumer)
}

// VolumeSnapshot represents a point-in-time copy of a volume.
type VolumeSnapshot struct {
	Name              string    `json:"name"`
	SourceVolume      string    `json:"source_volume"`
	Size              string    `json:"size"`
	SizeBytes         int64     `json:"size_bytes"`
	Status            string    `json:"status"`    // Pending | Ready | Failed
	CreatedAt         time.Time `json:"created_at"`
	ReadyAt           time.Time `json:"ready_at"`
	Reason            string    `json:"reason"`
	SnapshotClass     string    `json:"snapshot_class"`
}

// Provisioner handles volume creation/deletion.
type Provisioner interface {
	Provision(pvc PersistentVolumeClaim, sc StorageClass) (PersistentVolume, error)
	Delete(pv PersistentVolume) error
	Expand(pv PersistentVolume, newSize string) error
	CreateSnapshot(pv PersistentVolume, snapClass string) (VolumeSnapshot, error)
	RestoreSnapshot(snap VolumeSnapshot, pvc PersistentVolumeClaim) (PersistentVolume, error)
}

// StorageManager orchestrates storage provisioning and binding.
type StorageManager struct {
	mu               sync.RWMutex
	storageClasses   map[string]StorageClass
	pvcs             map[string]PersistentVolumeClaim
	pvs              map[string]PersistentVolume
	snapshots        map[string]VolumeSnapshot
	provisioners     map[string]Provisioner
	nodeCapacities   map[string]int64 // node ID -> available bytes
	classQuotas      map[string]int64 // storage class -> max bytes
}

// NewStorageManager creates a new storage manager.
func NewStorageManager() *StorageManager {
	return &StorageManager{
		storageClasses: make(map[string]StorageClass),
		pvcs:           make(map[string]PersistentVolumeClaim),
		pvs:            make(map[string]PersistentVolume),
		snapshots:      make(map[string]VolumeSnapshot),
		provisioners:   make(map[string]Provisioner),
		nodeCapacities: make(map[string]int64),
		classQuotas:    make(map[string]int64),
	}
}

// RegisterStorageClass registers a storage class.
func (sm *StorageManager) RegisterStorageClass(sc StorageClass) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.storageClasses[sc.Name]; exists {
		return fmt.Errorf("storage class %s already exists", sc.Name)
	}

	sm.storageClasses[sc.Name] = sc
	sm.classQuotas[sc.Name] = 0
	return nil
}

// RegisterProvisioner registers a provisioner for a storage class.
func (sm *StorageManager) RegisterProvisioner(className string, prov Provisioner) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.storageClasses[className]; !exists {
		return fmt.Errorf("storage class %s not found", className)
	}

	sm.provisioners[className] = prov
	return nil
}

// CreateClaim creates a PersistentVolumeClaim.
func (sm *StorageManager) CreateClaim(pvc PersistentVolumeClaim) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.pvcs[pvc.Name]; exists {
		return fmt.Errorf("claim %s already exists", pvc.Name)
	}

	if _, exists := sm.storageClasses[pvc.StorageClass]; !exists {
		return fmt.Errorf("storage class %s not found", pvc.StorageClass)
	}

	pvc.Status = "Pending"
	pvc.CreatedAt = time.Now()
	pvc.BindingMode = sm.storageClasses[pvc.StorageClass].VolumeBindingMode
	sm.pvcs[pvc.Name] = pvc

	return nil
}

// BindClaim binds a PVC to a PV (for Immediate binding mode).
func (sm *StorageManager) BindClaim(pvcName string, nodeName string) (PersistentVolume, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	pvc, exists := sm.pvcs[pvcName]
	if !exists {
		return PersistentVolume{}, fmt.Errorf("claim %s not found", pvcName)
	}

	sc, exists := sm.storageClasses[pvc.StorageClass]
	if !exists {
		return PersistentVolume{}, fmt.Errorf("storage class %s not found", pvc.StorageClass)
	}

	if pvc.BindingMode != "Immediate" && pvc.BindingMode != "WaitForFirstConsumer" {
		return PersistentVolume{}, fmt.Errorf("invalid binding mode %s", pvc.BindingMode)
	}

	// Provision volume
	prov, exists := sm.provisioners[pvc.StorageClass]
	if !exists {
		return PersistentVolume{}, fmt.Errorf("no provisioner for class %s", pvc.StorageClass)
	}

	pv, err := prov.Provision(pvc, sc)
	if err != nil {
		return PersistentVolume{}, err
	}

	pv.Name = fmt.Sprintf("%s-pv", pvc.Name)
	pv.Status = "Bound"
	pv.Phase = "Ready"
	pv.BoundNode = nodeName
	pv.CreatedAt = time.Now()
	pv.ReclaimPolicy = sc.ReclaimPolicy

	sm.pvs[pv.Name] = pv

	pvc.Status = "Bound"
	pvc.BoundVolume = pv.Name
	pvc.BoundAt = time.Now()
	pvc.BoundToNode = nodeName
	sm.pvcs[pvcName] = pvc

	return pv, nil
}

// ReleaseClaim releases a PVC and handles reclaim policy.
func (sm *StorageManager) ReleaseClaim(pvcName string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	pvc, exists := sm.pvcs[pvcName]
	if !exists {
		return fmt.Errorf("claim %s not found", pvcName)
	}

	if pvc.BoundVolume == "" {
		return fmt.Errorf("claim %s not bound", pvcName)
	}

	pv, exists := sm.pvs[pvc.BoundVolume]
	if !exists {
		return fmt.Errorf("volume %s not found", pvc.BoundVolume)
	}

	sc, exists := sm.storageClasses[pvc.StorageClass]
	if !exists {
		return fmt.Errorf("storage class %s not found", pvc.StorageClass)
	}

	// Apply reclaim policy
	switch sc.ReclaimPolicy {
	case "Delete":
		prov, exists := sm.provisioners[pvc.StorageClass]
		if exists {
			prov.Delete(pv)
		}
		delete(sm.pvs, pvc.BoundVolume)

	case "Retain":
		pv.Status = "Released"
		pv.Phase = "Releasing"
		sm.pvs[pvc.BoundVolume] = pv

	case "Recycle":
		pv.Status = "Pending"
		pv.Phase = "Provisioning"
		sm.pvs[pvc.BoundVolume] = pv
	}

	pvc.Status = "Lost"
	sm.pvcs[pvcName] = pvc

	return nil
}

// GetClaim retrieves a PVC.
func (sm *StorageManager) GetClaim(name string) (PersistentVolumeClaim, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	pvc, exists := sm.pvcs[name]
	if !exists {
		return PersistentVolumeClaim{}, fmt.Errorf("claim %s not found", name)
	}

	return pvc, nil
}

// GetVolume retrieves a PV.
func (sm *StorageManager) GetVolume(name string) (PersistentVolume, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	pv, exists := sm.pvs[name]
	if !exists {
		return PersistentVolume{}, fmt.Errorf("volume %s not found", name)
	}

	return pv, nil
}

// ExpandVolume expands a PV to a new size.
func (sm *StorageManager) ExpandVolume(pvName string, newSize string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	pv, exists := sm.pvs[pvName]
	if !exists {
		return fmt.Errorf("volume %s not found", pvName)
	}

	sc, exists := sm.storageClasses[pv.StorageClass]
	if !exists {
		return fmt.Errorf("storage class %s not found", pv.StorageClass)
	}

	if !sc.AllowVolumeExpansion {
		return fmt.Errorf("volume expansion not allowed for class %s", sc.Name)
	}

	prov, exists := sm.provisioners[pv.StorageClass]
	if !exists {
		return fmt.Errorf("no provisioner for class %s", pv.StorageClass)
	}

	if err := prov.Expand(pv, newSize); err != nil {
		return err
	}

	pv.Size = newSize
	sm.pvs[pvName] = pv
	return nil
}

// CreateSnapshot creates a volume snapshot.
func (sm *StorageManager) CreateSnapshot(pvName string, snapClass string) (VolumeSnapshot, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	pv, exists := sm.pvs[pvName]
	if !exists {
		return VolumeSnapshot{}, fmt.Errorf("volume %s not found", pvName)
	}

	prov, exists := sm.provisioners[pv.StorageClass]
	if !exists {
		return VolumeSnapshot{}, fmt.Errorf("no provisioner for class %s", pv.StorageClass)
	}

	snap, err := prov.CreateSnapshot(pv, snapClass)
	if err != nil {
		return VolumeSnapshot{}, err
	}

	snap.Status = "Ready"
	snap.ReadyAt = time.Now()

	snapName := fmt.Sprintf("%s-snap-%d", pvName, time.Now().Unix())
	sm.snapshots[snapName] = snap

	pv.SnapshotCount++
	sm.pvs[pvName] = pv

	return snap, nil
}

// ListSnapshots lists snapshots for a volume.
func (sm *StorageManager) ListSnapshots(pvName string) []VolumeSnapshot {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var snaps []VolumeSnapshot
	for _, snap := range sm.snapshots {
		if snap.SourceVolume == pvName {
			snaps = append(snaps, snap)
		}
	}
	return snaps
}

// SetNodeCapacity sets available storage on a node.
func (sm *StorageManager) SetNodeCapacity(nodeID string, bytes int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.nodeCapacities[nodeID] = bytes
}

// GetNodeCapacity gets remaining storage on a node.
func (sm *StorageManager) GetNodeCapacity(nodeID string) int64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.nodeCapacities[nodeID]
}

// ListVolumes lists all PVs.
func (sm *StorageManager) ListVolumes() []PersistentVolume {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var vols []PersistentVolume
	for _, pv := range sm.pvs {
		vols = append(vols, pv)
	}
	return vols
}

// ListClaims lists all PVCs.
func (sm *StorageManager) ListClaims() []PersistentVolumeClaim {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var claims []PersistentVolumeClaim
	for _, pvc := range sm.pvcs {
		claims = append(claims, pvc)
	}
	return claims
}

// GetStorageClass retrieves a storage class.
func (sm *StorageManager) GetStorageClass(name string) (StorageClass, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sc, exists := sm.storageClasses[name]
	if !exists {
		return StorageClass{}, fmt.Errorf("storage class %s not found", name)
	}

	return sc, nil
}
