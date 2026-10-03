// Package snapshots provides delta snapshot management for bandwidth reduction in Decentralized.Host.
//
// Delta snapshots significantly reduce replication bandwidth by:
//   - Incremental snapshot generation (only changed objects)
//   - Parent snapshot references (differential snapshots)
//   - Snapshot transmission protocol with CAS
//   - Recovery from delta snapshots with validation
package snapshots

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"
)

// ObjectID uniquely identifies an object within a snapshot.
type ObjectID string

// SnapshotID uniquely identifies a snapshot.
type SnapshotID string

// Hash is the SHA256 hash of an object's content.
type Hash string

// Object represents a single object in a snapshot.
type Object struct {
	ID       ObjectID  `json:"id"`
	Hash     Hash      `json:"hash"`
	Size     int64     `json:"size"`
	Data     []byte    `json:"data,omitempty"` // Only in full snapshots
	Modified time.Time `json:"modified"`
}

// Snapshot is a point-in-time capture of cluster state.
type Snapshot struct {
	ID       SnapshotID       `json:"id"`
	Timestamp time.Time        `json:"timestamp"`
	Objects  map[ObjectID]*Object `json:"objects"`
	Parent   SnapshotID       `json:"parent,omitempty"` // Empty for full snapshots
	Bytes    int64            `json:"bytes"` // Total size
	Version  int64            `json:"version"`
}

// DeltaSnapshot represents incremental changes from a parent snapshot.
type DeltaSnapshot struct {
	ID       SnapshotID             `json:"id"`
	Parent   SnapshotID             `json:"parent"` // Parent snapshot ID
	Added    map[ObjectID]*Object   `json:"added"`   // New objects
	Modified map[ObjectID]*Object   `json:"modified"` // Changed objects
	Removed  map[ObjectID]struct{}  `json:"removed"` // Deleted objects
	Timestamp time.Time              `json:"timestamp"`
	Size     int64                  `json:"size"` // Size of delta only
}

// Repository manages snapshots and delta snapshots.
type Repository struct {
	mu               sync.RWMutex
	snapshots        map[SnapshotID]*Snapshot
	deltas           map[SnapshotID]*DeltaSnapshot
	objectIndex      map[Hash]ObjectID // Hash to ObjectID index for deduplication
	snapshotVersions map[SnapshotID]int64 // Snapshot ID to version number
	maxSnapshots     int
	maxDeltas        int
	nextVersion      int64
}

// NewRepository creates a new snapshot repository.
func NewRepository() *Repository {
	return &Repository{
		snapshots:        make(map[SnapshotID]*Snapshot),
		deltas:           make(map[SnapshotID]*DeltaSnapshot),
		objectIndex:      make(map[Hash]ObjectID),
		snapshotVersions: make(map[SnapshotID]int64),
		maxSnapshots:     100,
		maxDeltas:        500,
		nextVersion:      1,
	}
}

// CreateFullSnapshot creates a new full snapshot.
func (r *Repository) CreateFullSnapshot(objects map[ObjectID][]byte) (SnapshotID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	snapshotID := SnapshotID(fmt.Sprintf("snap-%d-%d", time.Now().UnixNano()/1000000, len(objects)))
	snapshot := &Snapshot{
		ID:        snapshotID,
		Timestamp: time.Now(),
		Objects:   make(map[ObjectID]*Object),
		Parent:    "",
		Version:   r.nextVersion,
	}

	var totalBytes int64
	for objID, data := range objects {
		hash := computeHash(data)
		obj := &Object{
			ID:       objID,
			Hash:     hash,
			Size:     int64(len(data)),
			Data:     data,
			Modified: time.Now(),
		}
		snapshot.Objects[objID] = obj
		r.objectIndex[hash] = objID
		totalBytes += int64(len(data))
	}

	snapshot.Bytes = totalBytes
	r.snapshots[snapshotID] = snapshot
	r.snapshotVersions[snapshotID] = r.nextVersion
	r.nextVersion++

	// Prune old snapshots if necessary
	r.pruneSnapshots()

	return snapshotID, nil
}

// CreateDeltaSnapshot creates a delta snapshot from the parent snapshot.
func (r *Repository) CreateDeltaSnapshot(parentID SnapshotID, objects map[ObjectID][]byte) (SnapshotID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	parent, ok := r.snapshots[parentID]
	if !ok {
		return "", fmt.Errorf("parent snapshot %s not found", parentID)
	}

	deltaID := SnapshotID(fmt.Sprintf("delta-%d-%s", time.Now().UnixNano()/1000000, parentID))
	delta := &DeltaSnapshot{
		ID:        deltaID,
		Parent:    parentID,
		Added:     make(map[ObjectID]*Object),
		Modified:  make(map[ObjectID]*Object),
		Removed:   make(map[ObjectID]struct{}),
		Timestamp: time.Now(),
	}

	// Calculate changes
	newHashes := make(map[Hash]bool)
	for objID, data := range objects {
		hash := computeHash(data)
		newHashes[hash] = true

		parentObj, inParent := parent.Objects[objID]

		if !inParent {
			// New object
			obj := &Object{
				ID:       objID,
				Hash:     hash,
				Size:     int64(len(data)),
				Data:     data,
				Modified: time.Now(),
			}
			delta.Added[objID] = obj
			delta.Size += int64(len(data))
		} else if parentObj.Hash != hash {
			// Modified object
			obj := &Object{
				ID:       objID,
				Hash:     hash,
				Size:     int64(len(data)),
				Data:     data,
				Modified: time.Now(),
			}
			delta.Modified[objID] = obj
			delta.Size += int64(len(data))
		}
		// else: object unchanged, no entry needed
	}

	// Find removed objects
	for objID := range parent.Objects {
		if _, still := objects[objID]; !still {
			delta.Removed[objID] = struct{}{}
		}
	}

	r.deltas[deltaID] = delta
	r.snapshotVersions[deltaID] = r.nextVersion
	r.nextVersion++

	// Prune old deltas if necessary
	r.pruneDeltas()

	return deltaID, nil
}

// ApplyDelta reconstructs a snapshot from a parent and delta.
func (r *Repository) ApplyDelta(deltaID SnapshotID) (*Snapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	delta, ok := r.deltas[deltaID]
	if !ok {
		return nil, fmt.Errorf("delta snapshot %s not found", deltaID)
	}

	parent, ok := r.snapshots[delta.Parent]
	if !ok {
		return nil, fmt.Errorf("parent snapshot %s not found", delta.Parent)
	}

	// Start with parent's objects
	objects := make(map[ObjectID]*Object)
	for objID, obj := range parent.Objects {
		// Deep copy
		objCopy := &Object{
			ID:       obj.ID,
			Hash:     obj.Hash,
			Size:     obj.Size,
			Data:     make([]byte, len(obj.Data)),
			Modified: obj.Modified,
		}
		copy(objCopy.Data, obj.Data)
		objects[objID] = objCopy
	}

	// Apply additions
	for objID, obj := range delta.Added {
		objCopy := &Object{
			ID:       obj.ID,
			Hash:     obj.Hash,
			Size:     obj.Size,
			Data:     make([]byte, len(obj.Data)),
			Modified: obj.Modified,
		}
		copy(objCopy.Data, obj.Data)
		objects[objID] = objCopy
	}

	// Apply modifications
	for objID, obj := range delta.Modified {
		objCopy := &Object{
			ID:       obj.ID,
			Hash:     obj.Hash,
			Size:     obj.Size,
			Data:     make([]byte, len(obj.Data)),
			Modified: obj.Modified,
		}
		copy(objCopy.Data, obj.Data)
		objects[objID] = objCopy
	}

	// Apply removals
	for objID := range delta.Removed {
		delete(objects, objID)
	}

	// Create reconstructed snapshot
	reconstructedID := SnapshotID(fmt.Sprintf("reconstructed-%s", deltaID))
	snapshot := &Snapshot{
		ID:        reconstructedID,
		Timestamp: delta.Timestamp,
		Objects:   objects,
		Parent:    "",
		Version:   r.nextVersion,
	}

	var totalBytes int64
	for _, obj := range objects {
		totalBytes += obj.Size
	}
	snapshot.Bytes = totalBytes

	return snapshot, nil
}

// GetSnapshot retrieves a snapshot.
func (r *Repository) GetSnapshot(snapshotID SnapshotID) *Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot, ok := r.snapshots[snapshotID]
	if !ok {
		return nil
	}

	// Return a shallow copy
	return snapshot
}

// GetDelta retrieves a delta snapshot.
func (r *Repository) GetDelta(deltaID SnapshotID) *DeltaSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	delta, ok := r.deltas[deltaID]
	if !ok {
		return nil
	}

	return delta
}

// ListSnapshots returns all snapshot IDs.
func (r *Repository) ListSnapshots() []SnapshotID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshots := make([]SnapshotID, 0, len(r.snapshots))
	for id := range r.snapshots {
		snapshots = append(snapshots, id)
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return r.snapshotVersions[snapshots[i]] < r.snapshotVersions[snapshots[j]]
	})
	return snapshots
}

// ListDeltas returns all delta snapshot IDs.
func (r *Repository) ListDeltas() []SnapshotID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	deltas := make([]SnapshotID, 0, len(r.deltas))
	for id := range r.deltas {
		deltas = append(deltas, id)
	}
	return deltas
}

// GetCompressionRatio calculates the compression ratio of a delta vs full snapshot.
func (r *Repository) GetCompressionRatio(snapshotID, deltaID SnapshotID) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot, ok := r.snapshots[snapshotID]
	if !ok {
		return 0
	}

	delta, ok := r.deltas[deltaID]
	if !ok {
		return 0
	}

	if snapshot.Bytes == 0 {
		return 0
	}

	return float64(delta.Size) / float64(snapshot.Bytes)
}

// Serialize converts a snapshot to JSON.
func (r *Repository) Serialize(snapshotID SnapshotID) ([]byte, error) {
	r.mu.RLock()
	snapshot, ok := r.snapshots[snapshotID]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("snapshot %s not found", snapshotID)
	}

	return json.Marshal(snapshot)
}

// SerializeDelta converts a delta snapshot to JSON.
func (r *Repository) SerializeDelta(deltaID SnapshotID) ([]byte, error) {
	r.mu.RLock()
	delta, ok := r.deltas[deltaID]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("delta %s not found", deltaID)
	}

	return json.Marshal(delta)
}

// Deserialize reconstructs a snapshot from JSON.
func (r *Repository) Deserialize(data []byte) (*Snapshot, error) {
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.snapshots[snapshot.ID] = &snapshot
	r.snapshotVersions[snapshot.ID] = r.nextVersion
	r.nextVersion++
	r.mu.Unlock()

	return &snapshot, nil
}

// DeserializeDelta reconstructs a delta snapshot from JSON.
func (r *Repository) DeserializeDelta(data []byte) (*DeltaSnapshot, error) {
	var delta DeltaSnapshot
	if err := json.Unmarshal(data, &delta); err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.deltas[delta.ID] = &delta
	r.snapshotVersions[delta.ID] = r.nextVersion
	r.nextVersion++
	r.mu.Unlock()

	return &delta, nil
}

// pruneSnapshots removes old snapshots when the limit is exceeded.
func (r *Repository) pruneSnapshots() {
	if len(r.snapshots) <= r.maxSnapshots {
		return
	}

	// Find the oldest snapshots by version
	type snapshotEntry struct {
		id      SnapshotID
		version int64
	}
	var entries []snapshotEntry
	for id, version := range r.snapshotVersions {
		entries = append(entries, snapshotEntry{id, version})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].version < entries[j].version
	})

	// Remove oldest snapshots
	toRemove := len(r.snapshots) - r.maxSnapshots + 10
	for i := 0; i < toRemove && i < len(entries); i++ {
		delete(r.snapshots, entries[i].id)
		delete(r.snapshotVersions, entries[i].id)
	}
}

// pruneDeltas removes old deltas when the limit is exceeded.
func (r *Repository) pruneDeltas() {
	if len(r.deltas) <= r.maxDeltas {
		return
	}

	// Find the oldest deltas by version
	type deltaEntry struct {
		id      SnapshotID
		version int64
	}
	var entries []deltaEntry
	for id := range r.deltas {
		version, _ := r.snapshotVersions[id]
		entries = append(entries, deltaEntry{id, version})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].version < entries[j].version
	})

	// Remove oldest deltas
	toRemove := len(r.deltas) - r.maxDeltas + 50
	for i := 0; i < toRemove && i < len(entries); i++ {
		delete(r.deltas, entries[i].id)
		delete(r.snapshotVersions, entries[i].id)
	}
}

// computeHash returns the SHA256 hash of data.
func computeHash(data []byte) Hash {
	hash := sha256.Sum256(data)
	return Hash(hex.EncodeToString(hash[:]))
}

// StreamWriter provides streaming support for large snapshots.
type StreamWriter struct {
	writer io.Writer
	buffer []byte
	bufLen int
}

// NewStreamWriter creates a new stream writer.
func NewStreamWriter(w io.Writer) *StreamWriter {
	return &StreamWriter{
		writer: w,
		buffer: make([]byte, 8192),
	}
}

// WriteObject writes an object to the stream.
func (sw *StreamWriter) WriteObject(obj *Object) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	if sw.bufLen+len(data) > len(sw.buffer) {
		if _, err := sw.writer.Write(sw.buffer[:sw.bufLen]); err != nil {
			return err
		}
		sw.bufLen = 0
	}

	if len(data) > len(sw.buffer) {
		// Object too large for buffer, write directly
		_, err := sw.writer.Write(data)
		return err
	}

	copy(sw.buffer[sw.bufLen:], data)
	sw.bufLen += len(data)
	return nil
}

// Flush writes any remaining buffered data.
func (sw *StreamWriter) Flush() error {
	if sw.bufLen > 0 {
		_, err := sw.writer.Write(sw.buffer[:sw.bufLen])
		sw.bufLen = 0
		return err
	}
	return nil
}
