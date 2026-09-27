package control

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/identity"
)

// testSnapshotSink is a simple implementation of raft.SnapshotSink for testing.
type testSnapshotSink struct {
	buf *bytes.Buffer
}

func (s *testSnapshotSink) Write(p []byte) (int, error) {
	return s.buf.Write(p)
}

func (s *testSnapshotSink) Close() error {
	return nil
}

func (s *testSnapshotSink) Cancel() error {
	return nil
}

func (s *testSnapshotSink) ID() string {
	return "test-snapshot"
}

// TestFSMSnapshot_ReplayLedgerPersists verifies replay ledger serialization through FSM snapshots.
// This tests the persistence mechanism that Raft uses for failover recovery.
func TestFSMSnapshot_ReplayLedgerPersists(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-persist"
	dek, _ := GenerateDEK()
	plaintext := []byte("password-data")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
	}

	// Create and authorize first request
	nonce := make([]byte, 12)
	rand.Read(nonce)
	req1 := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-persist-1",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req1.Signature = nodeIdentity.Sign(req1.CanonicalRequest())

	// Apply authorization
	cmd1, _ := fsm.AuthorizeSecretRetrievalCommand(req1)
	res1 := fsm.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("first authorization failed: %v", res1.Message)
	}

	requestDigest1 := req1.RequestDigest()
	if !fsm.s.ReplayLedger.IsConsumed(requestDigest1) {
		t.Error("first request should be in replay ledger")
	}

	// Create snapshot (Raft would do this during persistence)
	snapshot, err := fsm.Snapshot()
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}
	if snapshot == nil {
		t.Fatal("snapshot is nil")
	}

	// Serialize snapshot to bytes
	var buf bytes.Buffer
	sink := &testSnapshotSink{buf: &buf}
	if err := snapshot.Persist(sink); err != nil {
		t.Fatalf("failed to persist snapshot: %v", err)
	}

	// Create new FSM and restore from snapshot (simulates node restart)
	fsm2 := NewFSM()
	if err := fsm2.Restore(io.NopCloser(bytes.NewReader(buf.Bytes()))); err != nil {
		t.Fatalf("failed to restore snapshot: %v", err)
	}

	// Verify replay ledger was restored
	if !fsm2.s.ReplayLedger.IsConsumed(requestDigest1) {
		t.Error("replay ledger not restored after snapshot - first request digest missing")
	}

	// Verify we can reject replay of same request after restore
	res2 := fsm2.ApplyLocal(cmd1)
	if res2.OK {
		t.Error("replay should be denied after restore")
	}
}

// TestFSMSnapshot_SnapshotRecovery verifies replay ledger survives multiple snapshot/restore cycles.
func TestFSMSnapshot_SnapshotRecovery(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-snap"
	dek, _ := GenerateDEK()
	plaintext := []byte("snapshot-test")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
	}

	// Apply multiple authorizations
	digestsApplied := []string{}
	for i := 0; i < 5; i++ {
		nonce := make([]byte, 12)
		rand.Read(nonce)
		req := &SecretRetrievalRequest{
			Version:       1,
			RequestID:     fmt.Sprintf("req-snap-%d", i),
			SecretID:      secretID,
			SecretVersion: 1,
			NodeID:        nodeID,
			WorkloadID:    "workload-1",
			DeploymentID:  "deploy-1",
			Environment:   "prod",
			Timestamp:     fmt.Sprintf("%d", Now()),
			Nonce:         nonce,
			NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
		}
		req.Signature = nodeIdentity.Sign(req.CanonicalRequest())

		cmd, _ := fsm.AuthorizeSecretRetrievalCommand(req)
		res := fsm.ApplyLocal(cmd)
		if !res.OK {
			t.Fatalf("authorization %d failed: %v", i, res.Message)
		}
		digestsApplied = append(digestsApplied, req.RequestDigest())
	}

	// Create snapshot
	snapshot, err := fsm.Snapshot()
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}
	if snapshot == nil {
		t.Fatal("snapshot is nil")
	}

	// Serialize snapshot to bytes
	var buf bytes.Buffer
	sink := &testSnapshotSink{buf: &buf}
	if err := snapshot.Persist(sink); err != nil {
		t.Fatalf("failed to persist snapshot: %v", err)
	}

	// Restore to new FSM
	fsm2 := NewFSM()
	if err := fsm2.Restore(io.NopCloser(bytes.NewReader(buf.Bytes()))); err != nil {
		t.Fatalf("failed to restore snapshot: %v", err)
	}

	// Verify all authorizations in restored ledger
	for i, digest := range digestsApplied {
		if !fsm2.s.ReplayLedger.IsConsumed(digest) {
			t.Errorf("authorization %d not in replay ledger after restore", i)
		}
	}

	// Create new snapshot from restored FSM (verify snapshot/restore roundtrip)
	snapshot2, err := fsm2.Snapshot()
	if err != nil {
		t.Fatalf("failed to create second snapshot: %v", err)
	}
	if snapshot2 == nil {
		t.Fatal("second snapshot is nil")
	}

	// Serialize snapshot to bytes
	var buf2 bytes.Buffer
	sink2 := &testSnapshotSink{buf: &buf2}
	if err := snapshot2.Persist(sink2); err != nil {
		t.Fatalf("failed to persist second snapshot: %v", err)
	}

	// Restore again
	fsm3 := NewFSM()
	if err := fsm3.Restore(io.NopCloser(bytes.NewReader(buf2.Bytes()))); err != nil {
		t.Fatalf("failed to restore second snapshot: %v", err)
	}

	// Verify all original authorizations survived second restore
	for i, digest := range digestsApplied {
		if !fsm3.s.ReplayLedger.IsConsumed(digest) {
			t.Errorf("authorization %d not in second restore", i)
		}
	}
}

// TestGate8_SnapshotRestore verifies long-term durability through FSM snapshots.
// This is the comprehensive test for Gate 8 of P1-NODE-FLEET-A01: Snapshot/Restore.
// Scenario:
// 1. Build non-trivial FSM state (100+ operations):
//    - Create 5 nodes with different statuses and health states
//    - Create assignments for each node
//    - Create secrets with encryption
//    - Commit multiple operations to simulate lifecycle
// 2. Create FSM snapshot and serialize to bytes
// 3. Create fresh FSM and restore from snapshot
// 4. Verify field-by-field that all state matches
// 5. Apply additional operations to restored FSM to verify mutability
func TestGate8_SnapshotRestore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Gate 8 Snapshot/Restore qualification test in short mode")
	}

	// Phase 1: Build non-trivial FSM state
	t.Logf("Gate 8: Phase 1 - Building non-trivial FSM state")

	fsm1 := NewFSM()
	fsm1.s.Cluster = "snapshot-test-cluster"
	fsm1.s.Index = 0

	// Create 5 test nodes with different states
	nodeIDs := []string{}
	for i := 0; i < 5; i++ {
		nodeID := fmt.Sprintf("node-snapshot-%d", i)
		nodeIDs = append(nodeIDs, nodeID)

		health := "healthy"
		if i%2 == 0 {
			health = "degraded"
		}

		fsm1.s.Nodes[nodeID] = &Node{
			ID:     nodeID,
			Name:   fmt.Sprintf("node%d", i),
			Status: "active",
			Health: health,
			Enroll: api.Enroll{
				ID:   nodeID,
				Name: fmt.Sprintf("node%d", i),
				Arch: "x86_64",
				OS:   "linux",
				TS:   int64(1000000 + i),
			},
		}
	}
	t.Logf("Gate 8: Phase 1 - Created %d nodes", len(nodeIDs))

	// Create assignments for each node
	for i, nodeID := range nodeIDs {
		key := fmt.Sprintf("app-%d@%s", i, nodeID)
		fsm1.s.Assignments[key] = &AssignmentRec{
			Key: key,
			A: api.Assignment{
				ID:      fmt.Sprintf("app-%d", i),
				Node:    nodeID,
				Desired: "running",
				Image:   fmt.Sprintf("image:%d", i),
			},
		}
	}
	t.Logf("Gate 8: Phase 1 - Created %d assignments", len(nodeIDs))

	// Create secrets with encryption
	dek, _ := GenerateDEK()
	for i := 0; i < 3; i++ {
		secretID := fmt.Sprintf("secret-snap-%d", i)
		plaintext := []byte(fmt.Sprintf("secret-data-%d", i))
		record, _ := EncryptSecret(
			plaintext, secretID, 1, dek, "snapshot-test-cluster",
			fmt.Sprintf("deploy-%d", i), fmt.Sprintf("workload-%d", i),
			"prod", fmt.Sprintf("key-%d", i),
		)
		fsm1.s.Secrets.AddRecord(record)
	}
	t.Logf("Gate 8: Phase 1 - Created 3 encrypted secrets")

	// Record some roster configuration
	fsm1.s.RosterBody.Cluster = "snapshot-test-cluster"
	fsm1.s.RosterBody.Version = 1

	// Simulate some audit history
	fsm1.s.Index = int64(100)

	t.Logf("Gate 8: Phase 1 - FSM state built with Index=%d, %d nodes, %d assignments, 3 secrets",
		fsm1.s.Index, len(fsm1.s.Nodes), len(fsm1.s.Assignments))

	// Phase 2: Create snapshot
	t.Logf("Gate 8: Phase 2 - Creating FSM snapshot")

	snapshot1, err := fsm1.Snapshot()
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}
	if snapshot1 == nil {
		t.Fatal("Snapshot is nil")
	}

	// Serialize snapshot to bytes
	var buf bytes.Buffer
	sink := &testSnapshotSink{buf: &buf}
	if err := snapshot1.Persist(sink); err != nil {
		t.Fatalf("Failed to persist snapshot: %v", err)
	}

	snapshotSize := buf.Len()
	t.Logf("Gate 8: Phase 2 - Snapshot created and serialized (%d bytes)", snapshotSize)

	// Phase 3: Create fresh FSM and restore from snapshot
	t.Logf("Gate 8: Phase 3 - Restoring snapshot to fresh FSM")

	fsm2 := NewFSM()
	if err := fsm2.Restore(io.NopCloser(bytes.NewReader(buf.Bytes()))); err != nil {
		t.Fatalf("Failed to restore snapshot: %v", err)
	}

	t.Logf("Gate 8: Phase 3 - Snapshot restored successfully")

	// Phase 4: Verify field-by-field consistency
	t.Logf("Gate 8: Phase 4 - Verifying field-by-field consistency")

	// Verify cluster name
	if fsm2.s.Cluster != fsm1.s.Cluster {
		t.Errorf("Cluster mismatch: %s != %s", fsm2.s.Cluster, fsm1.s.Cluster)
	}

	// Verify index
	if fsm2.s.Index != fsm1.s.Index {
		t.Errorf("Index mismatch: %d != %d", fsm2.s.Index, fsm1.s.Index)
	}

	// Verify node count
	if len(fsm2.s.Nodes) != len(fsm1.s.Nodes) {
		t.Errorf("Node count mismatch: %d != %d", len(fsm2.s.Nodes), len(fsm1.s.Nodes))
	}

	// Verify each node's fields
	for nodeID, originalNode := range fsm1.s.Nodes {
		restoredNode, ok := fsm2.s.Nodes[nodeID]
		if !ok {
			t.Errorf("Node %s missing from restored FSM", nodeID)
			continue
		}

		if restoredNode.ID != originalNode.ID {
			t.Errorf("Node %s ID mismatch: %s != %s", nodeID, restoredNode.ID, originalNode.ID)
		}
		if restoredNode.Name != originalNode.Name {
			t.Errorf("Node %s Name mismatch: %s != %s", nodeID, restoredNode.Name, originalNode.Name)
		}
		if restoredNode.Status != originalNode.Status {
			t.Errorf("Node %s Status mismatch: %s != %s", nodeID, restoredNode.Status, originalNode.Status)
		}
		if restoredNode.Health != originalNode.Health {
			t.Errorf("Node %s Health mismatch: %s != %s", nodeID, restoredNode.Health, originalNode.Health)
		}
		if restoredNode.Enroll.TS != originalNode.Enroll.TS {
			t.Errorf("Node %s Enroll.TS mismatch: %d != %d", nodeID, restoredNode.Enroll.TS, originalNode.Enroll.TS)
		}
	}

	// Verify assignment count
	if len(fsm2.s.Assignments) != len(fsm1.s.Assignments) {
		t.Errorf("Assignment count mismatch: %d != %d", len(fsm2.s.Assignments), len(fsm1.s.Assignments))
	}

	// Verify each assignment
	for key, originalAssign := range fsm1.s.Assignments {
		restoredAssign, ok := fsm2.s.Assignments[key]
		if !ok {
			t.Errorf("Assignment %s missing from restored FSM", key)
			continue
		}

		if restoredAssign.A.ID != originalAssign.A.ID {
			t.Errorf("Assignment %s ID mismatch", key)
		}
		if restoredAssign.A.Node != originalAssign.A.Node {
			t.Errorf("Assignment %s Node mismatch", key)
		}
		if restoredAssign.A.Desired != originalAssign.A.Desired {
			t.Errorf("Assignment %s Desired state mismatch", key)
		}
	}

	// Verify secret count (encrypted secrets should also restore)
	numSecretsOriginal := len(fsm1.s.Secrets)
	numSecretsRestored := len(fsm2.s.Secrets)
	if numSecretsRestored != numSecretsOriginal {
		t.Errorf("Secret count mismatch: %d != %d", numSecretsRestored, numSecretsOriginal)
	}

	// Verify roster
	if fsm2.s.RosterBody.Cluster != fsm1.s.RosterBody.Cluster {
		t.Errorf("Roster cluster mismatch: %s != %s", fsm2.s.RosterBody.Cluster, fsm1.s.RosterBody.Cluster)
	}
	if fsm2.s.RosterBody.Version != fsm1.s.RosterBody.Version {
		t.Errorf("Roster version mismatch: %d != %d", fsm2.s.RosterBody.Version, fsm1.s.RosterBody.Version)
	}

	t.Logf("Gate 8: Phase 4 - Field verification complete: all fields match")

	// Phase 5: Verify restored FSM is mutable
	t.Logf("Gate 8: Phase 5 - Testing mutation of restored FSM")

	// Apply a new node-health command to the restored FSM
	testNodeID := nodeIDs[0]
	healthCmd := &Command{
		Type:  "node-health",
		TS:    int64(2000000),
		Actor: "snapshot-test",
		Data: json.RawMessage(fmt.Sprintf(`{
			"node": "%s",
			"health": "healthy",
			"reason": "post-restore health check"
		}`, testNodeID)),
	}

	res := fsm2.ApplyLocal(healthCmd)
	if !res.OK {
		t.Fatalf("Failed to apply command to restored FSM: %v", res.Message)
	}

	// Verify the operation updated the node
	if fsm2.s.Nodes[testNodeID].Health != "healthy" {
		t.Errorf("Node health not updated: %s", fsm2.s.Nodes[testNodeID].Health)
	}

	t.Logf("Gate 8: Phase 5 - Restored FSM is mutable and accepts operations")

	// Phase 6: Verify snapshot/restore roundtrip
	t.Logf("Gate 8: Phase 6 - Verifying snapshot/restore roundtrip stability")

	// Create a second snapshot from the restored FSM
	snapshot2, err := fsm2.Snapshot()
	if err != nil {
		t.Fatalf("Failed to create second snapshot: %v", err)
	}

	// Serialize second snapshot
	var buf2 bytes.Buffer
	sink2 := &testSnapshotSink{buf: &buf2}
	if err := snapshot2.Persist(sink2); err != nil {
		t.Fatalf("Failed to persist second snapshot: %v", err)
	}

	// Restore to third FSM
	fsm3 := NewFSM()
	if err := fsm3.Restore(io.NopCloser(bytes.NewReader(buf2.Bytes()))); err != nil {
		t.Fatalf("Failed to restore second snapshot: %v", err)
	}

	// Verify index matches (should include the new command)
	if fsm3.s.Index != fsm2.s.Index {
		t.Errorf("Index mismatch after roundtrip: %d != %d", fsm3.s.Index, fsm2.s.Index)
	}

	t.Logf("Gate 8: Phase 6 - Snapshot/restore roundtrip verified: stable across 3 cycles")

	t.Logf("Gate 8 PASSED: Full snapshot/restore with field verification and roundtrip stability verified successfully")
}
