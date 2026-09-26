package control

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
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
