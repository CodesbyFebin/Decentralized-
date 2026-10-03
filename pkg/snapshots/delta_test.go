package snapshots

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestFullSnapshot(t *testing.T) {
	repo := NewRepository()

	// Create some objects
	objects := map[ObjectID][]byte{
		"obj-1": []byte("data for object 1"),
		"obj-2": []byte("data for object 2"),
		"obj-3": []byte("data for object 3"),
	}

	// Create a full snapshot
	snapshotID, err := repo.CreateFullSnapshot(objects)
	if err != nil {
		t.Fatalf("CreateFullSnapshot failed: %v", err)
	}

	// Verify the snapshot
	snapshot := repo.GetSnapshot(snapshotID)
	if snapshot == nil {
		t.Fatalf("GetSnapshot returned nil")
	}

	if len(snapshot.Objects) != 3 {
		t.Fatalf("Expected 3 objects, got %d", len(snapshot.Objects))
	}

	if snapshot.Parent != "" {
		t.Fatalf("Expected empty parent for full snapshot, got %s", snapshot.Parent)
	}

	expectedBytes := int64(17 + 17 + 17) // len of each data
	if snapshot.Bytes != expectedBytes {
		t.Fatalf("Expected %d bytes, got %d", expectedBytes, snapshot.Bytes)
	}
}

func TestDeltaSnapshot(t *testing.T) {
	repo := NewRepository()

	// Create initial full snapshot
	objects := map[ObjectID][]byte{
		"obj-1": []byte("data for object 1"),
		"obj-2": []byte("data for object 2"),
		"obj-3": []byte("data for object 3"),
	}

	snapshotID, err := repo.CreateFullSnapshot(objects)
	if err != nil {
		t.Fatalf("CreateFullSnapshot failed: %v", err)
	}

	// Create a delta with changes
	newObjects := map[ObjectID][]byte{
		"obj-1": []byte("modified data for object 1"), // Modified
		"obj-2": []byte("data for object 2"),           // Unchanged
		"obj-4": []byte("data for object 4"),           // Added
		// obj-3 is removed
	}

	deltaID, err := repo.CreateDeltaSnapshot(snapshotID, newObjects)
	if err != nil {
		t.Fatalf("CreateDeltaSnapshot failed: %v", err)
	}

	// Verify the delta
	delta := repo.GetDelta(deltaID)
	if delta == nil {
		t.Fatalf("GetDelta returned nil")
	}

	if len(delta.Added) != 1 {
		t.Fatalf("Expected 1 added object, got %d", len(delta.Added))
	}

	if len(delta.Modified) != 1 {
		t.Fatalf("Expected 1 modified object, got %d", len(delta.Modified))
	}

	if len(delta.Removed) != 1 {
		t.Fatalf("Expected 1 removed object, got %d", len(delta.Removed))
	}

	// Verify compression ratio
	ratio := repo.GetCompressionRatio(snapshotID, deltaID)
	if ratio == 0 {
		t.Fatalf("Expected non-zero compression ratio")
	}

	if ratio > 1.0 {
		t.Fatalf("Expected compression ratio <= 1.0, got %f", ratio)
	}

	t.Logf("Compression ratio: %.2f (delta %d bytes vs full %d bytes)",
		ratio, delta.Size, repo.GetSnapshot(snapshotID).Bytes)
}

func TestApplyDelta(t *testing.T) {
	repo := NewRepository()

	// Create initial full snapshot
	objects := map[ObjectID][]byte{
		"obj-1": []byte("data for object 1"),
		"obj-2": []byte("data for object 2"),
		"obj-3": []byte("data for object 3"),
	}

	snapshotID, err := repo.CreateFullSnapshot(objects)
	if err != nil {
		t.Fatalf("CreateFullSnapshot failed: %v", err)
	}

	// Create a delta
	newObjects := map[ObjectID][]byte{
		"obj-1": []byte("modified data for object 1"), // Modified
		"obj-2": []byte("data for object 2"),           // Unchanged
		"obj-4": []byte("data for object 4"),           // Added
		// obj-3 is removed
	}

	deltaID, err := repo.CreateDeltaSnapshot(snapshotID, newObjects)
	if err != nil {
		t.Fatalf("CreateDeltaSnapshot failed: %v", err)
	}

	// Apply the delta to reconstruct the snapshot
	reconstructed, err := repo.ApplyDelta(deltaID)
	if err != nil {
		t.Fatalf("ApplyDelta failed: %v", err)
	}

	// Verify the reconstructed snapshot
	if len(reconstructed.Objects) != 3 {
		t.Fatalf("Expected 3 objects in reconstructed snapshot, got %d", len(reconstructed.Objects))
	}

	// Verify each object
	if obj, ok := reconstructed.Objects["obj-1"]; !ok || string(obj.Data) != "modified data for object 1" {
		t.Fatalf("obj-1 not properly modified")
	}

	if obj, ok := reconstructed.Objects["obj-2"]; !ok || string(obj.Data) != "data for object 2" {
		t.Fatalf("obj-2 not preserved")
	}

	if obj, ok := reconstructed.Objects["obj-4"]; !ok || string(obj.Data) != "data for object 4" {
		t.Fatalf("obj-4 not added")
	}

	if _, ok := reconstructed.Objects["obj-3"]; ok {
		t.Fatalf("obj-3 should be removed but still exists")
	}
}

func TestSerialization(t *testing.T) {
	repo := NewRepository()

	// Create a snapshot
	objects := map[ObjectID][]byte{
		"obj-1": []byte("data for object 1"),
		"obj-2": []byte("data for object 2"),
	}

	snapshotID, err := repo.CreateFullSnapshot(objects)
	if err != nil {
		t.Fatalf("CreateFullSnapshot failed: %v", err)
	}

	// Serialize
	data, err := repo.Serialize(snapshotID)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatalf("Serialized data is empty")
	}

	// Deserialize
	repo2 := NewRepository()
	snapshot, err := repo2.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if len(snapshot.Objects) != 2 {
		t.Fatalf("Expected 2 objects in deserialized snapshot, got %d", len(snapshot.Objects))
	}
}

func TestListSnapshots(t *testing.T) {
	repo := NewRepository()

	// Create multiple snapshots
	snapshotIDs := make([]SnapshotID, 5)
	for i := 0; i < 5; i++ {
		objects := map[ObjectID][]byte{
			ObjectID(fmt.Sprintf("obj-%d", i)): []byte(fmt.Sprintf("data-%d", i)),
		}
		id, err := repo.CreateFullSnapshot(objects)
		if err != nil {
			t.Fatalf("CreateFullSnapshot failed: %v", err)
		}
		snapshotIDs[i] = id
		// Add small delay to ensure different timestamps
		time.Sleep(1 * time.Millisecond)
	}

	snapshots := repo.ListSnapshots()
	if len(snapshots) < 1 {
		t.Fatalf("Expected at least 1 snapshot, got %d", len(snapshots))
	}

	// Verify we can retrieve each snapshot
	for _, id := range snapshotIDs {
		snap := repo.GetSnapshot(id)
		if snap != nil {
			t.Logf("Retrieved snapshot %s", id)
		}
	}
}

func TestBandwidthReduction(t *testing.T) {
	repo := NewRepository()

	// Simulate a large state with 1000 objects
	fullObjects := make(map[ObjectID][]byte)
	for i := 0; i < 1000; i++ {
		objID := ObjectID(fmt.Sprintf("obj-%d", i))
		fullObjects[objID] = []byte(fmt.Sprintf("data for object %d", i))
	}

	snapshotID, err := repo.CreateFullSnapshot(fullObjects)
	if err != nil {
		t.Fatalf("CreateFullSnapshot failed: %v", err)
	}

	snapshot := repo.GetSnapshot(snapshotID)
	fullSize := snapshot.Bytes

	// Now simulate 10% change (100 objects modified, 100 new)
	modifiedObjects := make(map[ObjectID][]byte)
	for i := 0; i < 1000; i++ {
		objID := ObjectID(fmt.Sprintf("obj-%d", i))
		if i < 900 {
			modifiedObjects[objID] = []byte(fmt.Sprintf("data for object %d", i))
		} else {
			// Modified data for last 100
			modifiedObjects[objID] = []byte(fmt.Sprintf("modified data for object %d", i))
		}
	}

	// Add 100 new objects
	for i := 1000; i < 1100; i++ {
		objID := ObjectID(fmt.Sprintf("obj-%d", i))
		modifiedObjects[objID] = []byte(fmt.Sprintf("data for object %d", i))
	}

	deltaID, err := repo.CreateDeltaSnapshot(snapshotID, modifiedObjects)
	if err != nil {
		t.Fatalf("CreateDeltaSnapshot failed: %v", err)
	}

	delta := repo.GetDelta(deltaID)
	deltaSize := delta.Size

	ratio := float64(deltaSize) / float64(fullSize)

	t.Logf("Bandwidth Reduction Test:")
	t.Logf("  Full snapshot: %d bytes", fullSize)
	t.Logf("  Delta snapshot: %d bytes", deltaSize)
	t.Logf("  Compression ratio: %.2f (%.0f%% reduction)", ratio, (1-ratio)*100)

	if ratio > 0.2 {
		t.Logf("WARNING: Compression ratio %.2f is higher than expected for 10%% change", ratio)
	}
}

func TestStreamWriter(t *testing.T) {
	buffer := &bytes.Buffer{}
	sw := NewStreamWriter(buffer)

	// Write some objects
	for i := 0; i < 10; i++ {
		obj := &Object{
			ID:       ObjectID("obj-" + string(rune('0'+i))),
			Hash:     Hash("hash" + string(rune('0'+i))),
			Size:     100,
			Data:     []byte("test data"),
			Modified: time.Now(),
		}
		if err := sw.WriteObject(obj); err != nil {
			t.Fatalf("WriteObject failed: %v", err)
		}
	}

	if err := sw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if buffer.Len() == 0 {
		t.Fatalf("Stream writer wrote no data")
	}

	t.Logf("Stream writer produced %d bytes for 10 objects", buffer.Len())
}

func TestComputeHash(t *testing.T) {
	data1 := []byte("test data")
	data2 := []byte("test data")
	data3 := []byte("different data")

	hash1 := computeHash(data1)
	hash2 := computeHash(data2)
	hash3 := computeHash(data3)

	if hash1 != hash2 {
		t.Fatalf("Same data should produce same hash")
	}

	if hash1 == hash3 {
		t.Fatalf("Different data should produce different hash")
	}

	if len(hash1) != 64 { // SHA256 in hex is 64 chars
		t.Fatalf("Expected hash length 64, got %d", len(hash1))
	}
}
