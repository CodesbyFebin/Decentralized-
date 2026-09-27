package control

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"testing"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/identity"
)

// TestSecretLeaseRequest_SignatureVerification tests signature verification directly.
func TestSecretLeaseRequest_SignatureVerification(t *testing.T) {
	nodeIdentity, _ := identity.Generate()

	nonce := make([]byte, 12)
	rand.Read(nonce)

	req := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "sig-test",
		SecretID:      "secret-001",
		SecretVersion: 1,
		NodeID:        nodeIdentity.ID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeIdentity.ID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}

	canonical := req.CanonicalLeaseRequest()
	t.Logf("Canonical form length: %d bytes", len(canonical))
	t.Logf("Canonical form (first 100 chars): %s", string(canonical[:100]))

	sig := nodeIdentity.Sign(canonical)
	t.Logf("Signature length: %d bytes", len(sig))
	req.Signature = sig

	// Try to verify
	err := req.VerifyLeaseSignature()
	if err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}
	t.Log("Signature verification passed!")

	// Test JSON round-trip
	jsonBytes, _ := json.Marshal(req)
	t.Logf("JSON size: %d bytes", len(jsonBytes))

	req2 := &SecretLeaseRequest{}
	json.Unmarshal(jsonBytes, req2)

	err2 := req2.VerifyLeaseSignature()
	if err2 != nil {
		t.Fatalf("signature verification failed after JSON round-trip: %v", err2)
	}
	t.Log("Signature verification passed after JSON round-trip!")
}

// TestSecretLeaseAuthorize_Positive verifies a valid lease authorization request is accepted.
func TestSecretLeaseAuthorize_Positive(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	// Create node
	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{
		ID:     nodeID,
		Name:   "node-1",
		Status: "ready",
	}

	// Create assignment for the node
	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A: api.Assignment{
			ID:      "app-r0",
			Node:    nodeID,
			Desired: "running",
		},
		Created: Now(),
	}

	// Create secret with encryption
	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("database-password")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	// Create lease authorization request
	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-lease-001",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9), // 1 hour
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req.Signature = nodeIdentity.Sign(req.CanonicalLeaseRequest())

	// Authorize
	t.Logf("Request canonical length: %d", len(req.CanonicalLeaseRequest()))
	t.Logf("Request signature length: %d", len(req.Signature))

	cmd, err := fsm.AuthorizeSecretLeaseCommand(req)
	if err != nil {
		t.Fatalf("AuthorizeSecretLeaseCommand failed (leader-side): %v", err)
	}
	if cmd == nil {
		t.Fatalf("AuthorizeSecretLeaseCommand returned nil command")
	}
	t.Logf("Command created successfully, Type=%s", cmd.Type)

	res := fsm.ApplyLocal(cmd)
	if !res.OK {
		t.Fatalf("lease authorization failed (FSM-side): %v (code: %s)", res.Message, res.Code)
	}

	// Verify consumption recorded
	if !fsm.s.LeaseReplayLedger.IsConsumedLease(req.RequestLeaseDigest()) {
		t.Error("request digest not in lease replay ledger after authorization")
	}
}

// TestSecretLeaseAuthorize_BadSignature verifies signature verification check.
func TestSecretLeaseAuthorize_BadSignature(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	// Create secret
	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("database-password")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)

	req := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-lease-001",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}

	// Bad signature: sign with wrong canonical data
	badCanonical := []byte("wrong-data")
	req.Signature = nodeIdentity.Sign(badCanonical)

	cmd, err := fsm.AuthorizeSecretLeaseCommand(req)
	if err == nil {
		// May not error at command creation; error comes at Apply
		res := fsm.ApplyLocal(cmd)
		if res.OK {
			t.Error("expected bad signature to be denied")
		}
	}
}

// TestSecretLeaseAuthorize_Replay verifies replay protection: same request denied on retry.
func TestSecretLeaseAuthorize_Replay(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("database-password")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	reqID := "req-lease-replay-001"

	// First authorization
	req := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     reqID,
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req.Signature = nodeIdentity.Sign(req.CanonicalLeaseRequest())

	cmd1, _ := fsm.AuthorizeSecretLeaseCommand(req)
	res1 := fsm.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("first authorization failed: %v", res1.Message)
	}

	// Retry identical request
	cmd2, _ := fsm.AuthorizeSecretLeaseCommand(req)
	res2 := fsm.ApplyLocal(cmd2)
	if res2.OK {
		t.Error("expected replay to be denied, but got success")
	}
	if res2.Code != "DENIED" {
		t.Errorf("expected DENIED, got %s: %s", res2.Code, res2.Message)
	}
}

// TestSecretLeaseAuthorize_Concurrent verifies concurrent identical requests: only one succeeds.
func TestSecretLeaseAuthorize_Concurrent(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("database-password")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)

	req := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "concurrent-req",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req.Signature = nodeIdentity.Sign(req.CanonicalLeaseRequest())

	// Submit 50 concurrent identical requests
	numGoroutines := 50
	results := make([]*Result, numGoroutines)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cmd, _ := fsm.AuthorizeSecretLeaseCommand(req)
			res := fsm.ApplyLocal(cmd)
			mu.Lock()
			results[idx] = res
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify exactly one success, rest denied
	successCount := 0
	for _, res := range results {
		if res.OK {
			successCount++
		}
	}

	if successCount != 1 {
		t.Errorf("expected 1 success, got %d", successCount)
	}
}

// TestSecretLeaseAuthorize_SameLeaseDifferentPayload verifies CONFLICT rejection.
func TestSecretLeaseAuthorize_SameLeaseDifferentPayload(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID1 := "secret-001"
	secretID2 := "secret-002"
	dek, _ := GenerateDEK()

	record1, _ := EncryptSecret([]byte("pass1"), secretID1, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	record2, _ := EncryptSecret([]byte("pass2"), secretID2, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record1)
	fsm.s.Secrets.AddRecord(record2)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	// First request for secret 1
	nonce1 := make([]byte, 12)
	rand.Read(nonce1)
	reqID := "conflict-req"

	req1 := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     reqID,
		SecretID:      secretID1,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce1,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req1.Signature = nodeIdentity.Sign(req1.CanonicalLeaseRequest())

	cmd1, _ := fsm.AuthorizeSecretLeaseCommand(req1)
	res1 := fsm.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("first lease authorization failed: %v", res1.Message)
	}

	// Second request with same ID but different secret
	nonce2 := make([]byte, 12)
	rand.Read(nonce2)
	now := Now()

	req2 := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     reqID, // SAME ID
		SecretID:      secretID2, // DIFFERENT SECRET
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", now), // same timestamp window
		Nonce:         nonce2,
		CallerID:      nodeID,
		IssuedAt:      now - int64(100e6), // issued in the past (100ms ago)
		ExpiresAt:     now + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req2.Signature = nodeIdentity.Sign(req2.CanonicalLeaseRequest())

	cmd2, _ := fsm.AuthorizeSecretLeaseCommand(req2)
	res2 := fsm.ApplyLocal(cmd2)
	if res2.OK {
		t.Error("expected CONFLICT, but got success")
	}
	if res2.Code != "CONFLICT" {
		t.Errorf("expected CONFLICT code, got %s: %s", res2.Code, res2.Message)
	}
}

// TestSecretLeaseAuthorize_NonceReuseWithDifferentScope verifies nonce replay protection.
func TestSecretLeaseAuthorize_NonceReuseWithDifferentScope(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	record, _ := EncryptSecret([]byte("pass"), secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)

	// First request
	req1 := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "nonce-test-1",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce, // SAME NONCE
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req1.Signature = nodeIdentity.Sign(req1.CanonicalLeaseRequest())

	cmd1, _ := fsm.AuthorizeSecretLeaseCommand(req1)
	res1 := fsm.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("first authorization failed: %v", res1.Message)
	}

	// Second request with same nonce but different generation
	req2 := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "nonce-test-2",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    2, // DIFFERENT GENERATION
		Timestamp:     fmt.Sprintf("%d", Now()+1e9),
		Nonce:         nonce, // SAME NONCE
		CallerID:      nodeID,
		IssuedAt:      Now() + int64(1e9),
		ExpiresAt:     Now() + int64(3600e9+1e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req2.Signature = nodeIdentity.Sign(req2.CanonicalLeaseRequest())

	cmd2, _ := fsm.AuthorizeSecretLeaseCommand(req2)
	res2 := fsm.ApplyLocal(cmd2)
	if res2.OK {
		t.Error("expected nonce reuse with different scope to be denied")
	}
	if res2.Code != "DENIED" {
		t.Errorf("expected DENIED, got %s: %s", res2.Code, res2.Message)
	}
}

// TestSecretLeaseAuthorize_SnapshotRestore verifies lease replay ledger persists through snapshot.
func TestSecretLeaseAuthorize_SnapshotRestore(t *testing.T) {
	fsm1 := NewFSM()
	fsm1.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm1.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	record, _ := EncryptSecret([]byte("pass"), secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm1.s.Secrets.AddRecord(record)

	fsm1.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	// Authorize a lease
	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "snapshot-test",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req.Signature = nodeIdentity.Sign(req.CanonicalLeaseRequest())

	cmd, _ := fsm1.AuthorizeSecretLeaseCommand(req)
	fsm1.ApplyLocal(cmd)

	requestDigest := req.RequestLeaseDigest()

	// Take snapshot
	snapshot, err := fsm1.Snapshot()
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	// Serialize snapshot
	var buf bytes.Buffer
	if err := snapshot.Persist(&testSnapshotSink{buf: &buf}); err != nil {
		t.Fatalf("persist snapshot failed: %v", err)
	}

	// Restore to new FSM
	fsm2 := NewFSM()
	if err := fsm2.Restore(io.NopCloser(bytes.NewReader(buf.Bytes()))); err != nil {
		t.Fatalf("restore snapshot failed: %v", err)
	}

	// Verify lease replay ledger was restored
	if !fsm2.s.LeaseReplayLedger.IsConsumedLease(requestDigest) {
		t.Error("lease authorization not restored from snapshot")
	}

	// Verify retry is denied
	cmd2, _ := fsm2.AuthorizeSecretLeaseCommand(req)
	res2 := fsm2.ApplyLocal(cmd2)
	if res2.OK {
		t.Error("expected replay to be denied after restore")
	}
}

// ============================================================================
// QUALIFICATION TESTS (Sections 17-31 per SEC-P0-A01-A04)
// ============================================================================

// TestSecretLease_LostResponse: Commit → Discard Response → Retry = ALREADY_CONSUMED
func TestSecretLease_LostResponse(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-lost-response"
	dek, _ := GenerateDEK()
	record, _ := EncryptSecret([]byte("secret-data"), secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	reqID := "lost-response-req"

	req := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     reqID,
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req.Signature = nodeIdentity.Sign(req.CanonicalLeaseRequest())

	// First request commits and succeeds
	cmd1, _ := fsm.AuthorizeSecretLeaseCommand(req)
	res1 := fsm.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("first authorization failed: %v", res1.Message)
	}
	t.Logf("First request succeeded: response would have been sent")

	// Simulate lost response: client doesn't receive, but Raft state is committed
	// Retry identical request
	cmd2, _ := fsm.AuthorizeSecretLeaseCommand(req)
	res2 := fsm.ApplyLocal(cmd2)

	// Retry must be denied, not succeed again
	if res2.OK {
		t.Error("SECURITY VIOLATION: lost response allowed second authorization")
	}
	if res2.Code != "DENIED" && res2.Code != "ALREADY_CONSUMED" {
		t.Errorf("expected DENIED or ALREADY_CONSUMED, got %s: %s", res2.Code, res2.Message)
	}
	t.Logf("Retry correctly denied with code: %s", res2.Code)
}

// TestSecretLease_CrossScope_NodeIDRejection: Nonce reuse across different nodes
func TestSecretLease_CrossScope_NodeIDRejection(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	node1ID := "dh1testnode1aaaaaaaaaaaaa"
	node2ID := "dh1testnode2aaaaaaaaaaaaa"
	identity1, _ := identity.Generate()
	identity2, _ := identity.Generate()
	node1ID = identity1.ID
	node2ID = identity2.ID

	fsm.s.Nodes[node1ID] = &Node{ID: node1ID, Status: "ready"}
	fsm.s.Nodes[node2ID] = &Node{ID: node2ID, Status: "ready"}

	secretID := "secret-cross-scope"
	dek, _ := GenerateDEK()
	record, _ := EncryptSecret([]byte("secret"), secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+node1ID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: node1ID, Desired: "running"},
		Created: Now(),
	}
	fsm.s.Assignments["app-r0@"+node2ID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: node2ID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)

	// Request from node1
	req1 := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-node1",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        node1ID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce, // SHARED NONCE
		CallerID:      node1ID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(identity1.Pub),
	}
	req1.Signature = identity1.Sign(req1.CanonicalLeaseRequest())

	cmd1, _ := fsm.AuthorizeSecretLeaseCommand(req1)
	res1 := fsm.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("node1 authorization failed: %v", res1.Message)
	}

	// Request from node2 with same nonce but different node
	req2 := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-node2",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        node2ID, // DIFFERENT NODE
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now() + 1e9),
		Nonce:         nonce, // SAME NONCE
		CallerID:      node2ID,
		IssuedAt:      Now() + int64(1e9),
		ExpiresAt:     Now() + int64(3600e9+1e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(identity2.Pub),
	}
	req2.Signature = identity2.Sign(req2.CanonicalLeaseRequest())

	cmd2, _ := fsm.AuthorizeSecretLeaseCommand(req2)
	res2 := fsm.ApplyLocal(cmd2)

	// Should be denied due to nonce reuse across scope (different node)
	if res2.OK {
		t.Error("SECURITY VIOLATION: nonce reuse across different nodes allowed")
	}
	if res2.Code != "DENIED" {
		t.Errorf("expected DENIED, got %s: %s", res2.Code, res2.Message)
	}
}

// TestSecretLease_CrossScope_WorkloadIDRejection: Nonce reuse across different workloads
func TestSecretLease_CrossScope_WorkloadIDRejection(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-workload-scope"
	dek, _ := GenerateDEK()
	record, _ := EncryptSecret([]byte("secret"), secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)

	// Request 1: workload-1
	req1 := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-wl1",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req1.Signature = nodeIdentity.Sign(req1.CanonicalLeaseRequest())

	cmd1, _ := fsm.AuthorizeSecretLeaseCommand(req1)
	res1 := fsm.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("workload-1 authorization failed: %v", res1.Message)
	}

	// Request 2: workload-2 with same nonce
	req2 := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-wl2",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-2", // DIFFERENT WORKLOAD
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now() + 1e9),
		Nonce:         nonce, // SAME NONCE
		CallerID:      nodeID,
		IssuedAt:      Now() + int64(1e9),
		ExpiresAt:     Now() + int64(3600e9+1e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req2.Signature = nodeIdentity.Sign(req2.CanonicalLeaseRequest())

	cmd2, _ := fsm.AuthorizeSecretLeaseCommand(req2)
	res2 := fsm.ApplyLocal(cmd2)

	if res2.OK {
		t.Error("SECURITY VIOLATION: nonce reuse across different workloads allowed")
	}
	if res2.Code != "DENIED" {
		t.Errorf("expected DENIED, got %s: %s", res2.Code, res2.Message)
	}
}

// TestSecretLease_CrossScope_DeploymentIDRejection: Nonce reuse across different deployments
func TestSecretLease_CrossScope_DeploymentIDRejection(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-deploy-scope"
	dek, _ := GenerateDEK()
	record, _ := EncryptSecret([]byte("secret"), secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)

	// Request 1: deploy-1
	req1 := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-d1",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req1.Signature = nodeIdentity.Sign(req1.CanonicalLeaseRequest())

	cmd1, _ := fsm.AuthorizeSecretLeaseCommand(req1)
	res1 := fsm.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("deploy-1 authorization failed: %v", res1.Message)
	}

	// Request 2: deploy-2 with same nonce
	req2 := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-d2",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-2", // DIFFERENT DEPLOYMENT
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now() + 1e9),
		Nonce:         nonce, // SAME NONCE
		CallerID:      nodeID,
		IssuedAt:      Now() + int64(1e9),
		ExpiresAt:     Now() + int64(3600e9+1e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req2.Signature = nodeIdentity.Sign(req2.CanonicalLeaseRequest())

	cmd2, _ := fsm.AuthorizeSecretLeaseCommand(req2)
	res2 := fsm.ApplyLocal(cmd2)

	if res2.OK {
		t.Error("SECURITY VIOLATION: nonce reuse across different deployments allowed")
	}
	if res2.Code != "DENIED" {
		t.Errorf("expected DENIED, got %s: %s", res2.Code, res2.Message)
	}
}

// TestSecretLease_CrossScope_SecretIDRejection: Nonce reuse across different secrets
func TestSecretLease_CrossScope_SecretIDRejection(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	dek, _ := GenerateDEK()
	secret1, _ := EncryptSecret([]byte("secret1"), "secret-a", 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	secret2, _ := EncryptSecret([]byte("secret2"), "secret-b", 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(secret1)
	fsm.s.Secrets.AddRecord(secret2)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)

	// Request 1: secret-a
	req1 := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-sa",
		SecretID:      "secret-a",
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req1.Signature = nodeIdentity.Sign(req1.CanonicalLeaseRequest())

	cmd1, _ := fsm.AuthorizeSecretLeaseCommand(req1)
	res1 := fsm.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("secret-a authorization failed: %v", res1.Message)
	}

	// Request 2: secret-b with same nonce
	req2 := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-sb",
		SecretID:      "secret-b", // DIFFERENT SECRET
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now() + 1e9),
		Nonce:         nonce, // SAME NONCE
		CallerID:      nodeID,
		IssuedAt:      Now() + int64(1e9),
		ExpiresAt:     Now() + int64(3600e9+1e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req2.Signature = nodeIdentity.Sign(req2.CanonicalLeaseRequest())

	cmd2, _ := fsm.AuthorizeSecretLeaseCommand(req2)
	res2 := fsm.ApplyLocal(cmd2)

	if res2.OK {
		t.Error("SECURITY VIOLATION: nonce reuse across different secrets allowed")
	}
	if res2.Code != "DENIED" {
		t.Errorf("expected DENIED, got %s: %s", res2.Code, res2.Message)
	}
}

// TestSecretLease_CrossScope_GenerationRejection: Nonce reuse across different generations
func TestSecretLease_CrossScope_GenerationRejection(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-gen-scope"
	dek, _ := GenerateDEK()
	record, _ := EncryptSecret([]byte("secret"), secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)

	// Request 1: generation 1
	req1 := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-g1",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req1.Signature = nodeIdentity.Sign(req1.CanonicalLeaseRequest())

	cmd1, _ := fsm.AuthorizeSecretLeaseCommand(req1)
	res1 := fsm.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("generation 1 authorization failed: %v", res1.Message)
	}

	// Request 2: generation 2 with same nonce
	req2 := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-g2",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    2, // DIFFERENT GENERATION
		Timestamp:     fmt.Sprintf("%d", Now() + 1e9),
		Nonce:         nonce, // SAME NONCE
		CallerID:      nodeID,
		IssuedAt:      Now() + int64(1e9),
		ExpiresAt:     Now() + int64(3600e9+1e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req2.Signature = nodeIdentity.Sign(req2.CanonicalLeaseRequest())

	cmd2, _ := fsm.AuthorizeSecretLeaseCommand(req2)
	res2 := fsm.ApplyLocal(cmd2)

	if res2.OK {
		t.Error("SECURITY VIOLATION: nonce reuse across different generations allowed")
	}
	if res2.Code != "DENIED" {
		t.Errorf("expected DENIED, got %s: %s", res2.Code, res2.Message)
	}
}

// TestSecretLease_NegativeControl_ExpiredCredential: Expired credential should be rejected
func TestSecretLease_NegativeControl_ExpiredCredential(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-expired"
	dek, _ := GenerateDEK()
	record, _ := EncryptSecret([]byte("secret"), secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	now := Now()

	// Request with EXPIRED ExpiresAt (in the past)
	req := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-expired",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", now),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      now - int64(3600e9), // 1 hour ago
		ExpiresAt:     now - int64(1e9),    // EXPIRED 1 second ago
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req.Signature = nodeIdentity.Sign(req.CanonicalLeaseRequest())

	cmd, _ := fsm.AuthorizeSecretLeaseCommand(req)
	res := fsm.ApplyLocal(cmd)

	if res.OK {
		t.Error("SECURITY VIOLATION: expired credential was accepted")
	}
	if res.Code != "DENIED" {
		t.Errorf("expected DENIED, got %s: %s", res.Code, res.Message)
	}
}

// TestSecretLease_NegativeControl_FutureCredential: Not-yet-issued credential should be rejected
func TestSecretLease_NegativeControl_FutureCredential(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-future"
	dek, _ := GenerateDEK()
	record, _ := EncryptSecret([]byte("secret"), secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	now := Now()

	// Request with IssuedAt in the future (not yet issued)
	req := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-future",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", now),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      now + int64(3600e9), // FUTURE: issued in 1 hour
		ExpiresAt:     now + int64(7200e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req.Signature = nodeIdentity.Sign(req.CanonicalLeaseRequest())

	cmd, _ := fsm.AuthorizeSecretLeaseCommand(req)
	res := fsm.ApplyLocal(cmd)

	if res.OK {
		t.Error("SECURITY VIOLATION: not-yet-issued credential was accepted")
	}
	if res.Code != "DENIED" {
		t.Errorf("expected DENIED, got %s: %s", res.Code, res.Message)
	}
}

// TestSecretLease_NegativeControl_ClockSkewTooLarge: Clock skew beyond tolerance should be rejected
func TestSecretLease_NegativeControl_ClockSkewTooLarge(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-skew"
	dek, _ := GenerateDEK()
	record, _ := EncryptSecret([]byte("secret"), secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	now := Now()

	// Request with timestamp far in the future (clock skew > 5s tolerance)
	req := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-skew",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", now + int64(10e9)), // SKEW: 10 seconds in future
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      now + int64(10e9),
		ExpiresAt:     now + int64(3610e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req.Signature = nodeIdentity.Sign(req.CanonicalLeaseRequest())

	cmd, _ := fsm.AuthorizeSecretLeaseCommand(req)
	res := fsm.ApplyLocal(cmd)

	if res.OK {
		t.Error("SECURITY VIOLATION: excessive clock skew was accepted")
	}
	if res.Code != "DENIED" {
		t.Errorf("expected DENIED, got %s: %s", res.Code, res.Message)
	}
}

// TestSecretLease_NegativeControl_RevokedNode: Request from revoked node should be rejected
func TestSecretLease_NegativeControl_RevokedNode(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-revoked"
	dek, _ := GenerateDEK()
	record, _ := EncryptSecret([]byte("secret"), secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)

	req := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-revoked",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req.Signature = nodeIdentity.Sign(req.CanonicalLeaseRequest())

	// Revoke the node (set RevokedAt timestamp)
	fsm.s.Nodes[nodeID].RevokedAt = Now()

	cmd, _ := fsm.AuthorizeSecretLeaseCommand(req)
	res := fsm.ApplyLocal(cmd)

	if res.OK {
		t.Error("SECURITY VIOLATION: revoked node was granted authorization")
	}
	if res.Code != "DENIED" {
		t.Errorf("expected DENIED, got %s: %s", res.Code, res.Message)
	}
}

// TestSecretLease_NegativeControl_WrongSecret: Request for non-existent secret should be rejected
func TestSecretLease_NegativeControl_WrongSecret(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	// Secret-001 exists
	dek, _ := GenerateDEK()
	record, _ := EncryptSecret([]byte("secret"), "secret-001", 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)

	// Request for non-existent secret
	req := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-wrong-secret",
		SecretID:      "secret-nonexistent", // DOES NOT EXIST
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req.Signature = nodeIdentity.Sign(req.CanonicalLeaseRequest())

	cmd, _ := fsm.AuthorizeSecretLeaseCommand(req)
	res := fsm.ApplyLocal(cmd)

	if res.OK {
		t.Error("SECURITY VIOLATION: non-existent secret was granted")
	}
	if res.Code != "DENIED" {
		t.Errorf("expected DENIED, got %s: %s", res.Code, res.Message)
	}
}

// TestSecretLease_NegativeControl_NoAssignment: Node without assignment should be rejected
func TestSecretLease_NegativeControl_NoAssignment(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}
	// NOTE: No assignment created for this node

	secretID := "secret-no-assign"
	dek, _ := GenerateDEK()
	record, _ := EncryptSecret([]byte("secret"), secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	nonce := make([]byte, 12)
	rand.Read(nonce)

	req := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-no-assign",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req.Signature = nodeIdentity.Sign(req.CanonicalLeaseRequest())

	cmd, _ := fsm.AuthorizeSecretLeaseCommand(req)
	res := fsm.ApplyLocal(cmd)

	if res.OK {
		t.Error("SECURITY VIOLATION: node without assignment was granted access")
	}
	if res.Code != "DENIED" {
		t.Errorf("expected DENIED, got %s: %s", res.Code, res.Message)
	}
}

// TestSecretLease_NegativeControl_AssignmentNotRunning: Non-running assignment should be rejected
func TestSecretLease_NegativeControl_AssignmentNotRunning(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-not-running"
	dek, _ := GenerateDEK()
	record, _ := EncryptSecret([]byte("secret"), secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	// Assignment with Desired != "running"
	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		A:       api.Assignment{ID: "app-r0", Node: nodeID, Desired: "stopped"},
		Created: Now(),
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)

	req := &SecretLeaseRequest{
		Protocol:      "dhp://secrets/v1",
		Version:       1,
		RequestType:   "secret-lease-request",
		RequestID:     "req-not-running",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Generation:    1,
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		CallerID:      nodeID,
		IssuedAt:      Now(),
		ExpiresAt:     Now() + int64(3600e9),
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req.Signature = nodeIdentity.Sign(req.CanonicalLeaseRequest())

	cmd, _ := fsm.AuthorizeSecretLeaseCommand(req)
	res := fsm.ApplyLocal(cmd)

	if res.OK {
		t.Error("SECURITY VIOLATION: non-running assignment was granted access")
	}
	if res.Code != "DENIED" {
		t.Errorf("expected DENIED, got %s: %s", res.Code, res.Message)
	}
}
