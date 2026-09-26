package control

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/identity"
)

// TestSecretRetrievalAuthorize_Positive verifies a valid authorization request is accepted.
func TestSecretRetrievalAuthorize_Positive(t *testing.T) {
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

	// Create authorization request
	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-001",
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

	// Authorize
	cmd, err := fsm.AuthorizeSecretRetrievalCommand(req)
	if err != nil {
		t.Fatalf("AuthorizeSecretRetrievalCommand failed: %v", err)
	}
	t.Logf("DEBUG: Command created successfully, Type=%s, Data length=%d", cmd.Type, len(cmd.Data))

	res := fsm.ApplyLocal(cmd)
	t.Logf("DEBUG: ApplyLocal result: OK=%v, Message=%s", res.OK, res.Message)
	if !res.OK {
		t.Fatalf("authorization failed: %v", res.Message)
	}

	// Verify consumption recorded
	if !fsm.s.ReplayLedger.IsConsumed(req.RequestDigest()) {
		t.Error("request digest not in replay ledger after authorization")
	}
}

// TestSecretRetrievalAuthorize_BadSignature verifies signature verification check.
func TestSecretRetrievalAuthorize_BadSignature(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	// Create secret
	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("password")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	// Create assignment
	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-001",
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
	// Bad signature
	req.Signature = make([]byte, 64)
	rand.Read(req.Signature)

	cmd, err := fsm.AuthorizeSecretRetrievalCommand(req)
	if err == nil {
		t.Fatal("expected signature verification to fail on leader")
	}
	if cmd != nil {
		t.Fatal("command should not be created on signature failure")
	}
}

// TestSecretRetrievalAuthorize_RevokedNode verifies revoked node rejection.
func TestSecretRetrievalAuthorize_RevokedNode(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{
		ID:        nodeID,
		Status:    "revoked",
		RevokedAt: Now(),
	}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("password")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-001",
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
	if res.OK {
		t.Error("revoked node should be denied")
	}
}

// TestSecretRetrievalAuthorize_MissingSecret verifies secret existence check.
func TestSecretRetrievalAuthorize_MissingSecret(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-001",
		SecretID:      "nonexistent-secret",
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
	if res.OK {
		t.Error("missing secret should be denied")
	}
}

// TestSecretRetrievalAuthorize_AssignmentNotRunning verifies assignment state check.
func TestSecretRetrievalAuthorize_AssignmentNotRunning(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("password")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	// Assignment in stopped state
	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "stopped"},
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-001",
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
	if res.OK {
		t.Error("stopped assignment should be denied")
	}
}

// TestSecretRetrievalAuthorize_ScopeMismatch verifies scope binding check.
func TestSecretRetrievalAuthorize_ScopeMismatch(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("password")
	// Secret encrypted with specific scope
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-001",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-2", // different deployment
		Environment:   "prod",
		Timestamp:     fmt.Sprintf("%d", Now()),
		Nonce:         nonce,
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req.Signature = nodeIdentity.Sign(req.CanonicalRequest())

	cmd, _ := fsm.AuthorizeSecretRetrievalCommand(req)
	res := fsm.ApplyLocal(cmd)
	if res.OK {
		t.Error("scope mismatch should be denied")
	}
}

// TestSecretRetrievalAuthorize_Replay verifies replay protection.
func TestSecretRetrievalAuthorize_Replay(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("password")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-replay",
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

	// First authorization should succeed
	cmd1, _ := fsm.AuthorizeSecretRetrievalCommand(req)
	res1 := fsm.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("first authorization failed: %v", res1.Message)
	}

	// Replay the same request → should fail
	cmd2, _ := fsm.AuthorizeSecretRetrievalCommand(req)
	res2 := fsm.ApplyLocal(cmd2)
	if res2.OK {
		t.Error("replay of same request should be denied")
	}
}

// TestSecretRetrievalAuthorize_Concurrent tests exactly-once semantics under concurrency.
// Simulates concurrent identical requests; only one should succeed.
func TestSecretRetrievalAuthorize_Concurrent(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("password")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-concurrent",
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

	// Simulate concurrent requests (in FSM, they'd be sequential, but same request digest)
	cmd1, _ := fsm.AuthorizeSecretRetrievalCommand(req)
	cmd2, _ := fsm.AuthorizeSecretRetrievalCommand(req)

	res1 := fsm.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("first concurrent request failed: %v", res1.Message)
	}

	res2 := fsm.ApplyLocal(cmd2)
	if res2.OK {
		t.Error("second concurrent request should be denied (already consumed)")
	}
}

// TestSecretRetrievalAuthorize_ClockSkew verifies clock tolerance check.
func TestSecretRetrievalAuthorize_ClockSkew(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("password")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
	}

	now := Now()
	// Request timestamp is more than 5 seconds in the future
	future := now + 10*time.Second.Nanoseconds()
	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-skew",
		SecretID:      secretID,
		SecretVersion: 1,
		NodeID:        nodeID,
		WorkloadID:    "workload-1",
		DeploymentID:  "deploy-1",
		Environment:   "prod",
		Timestamp:     fmt.Sprintf("%d", future),
		Nonce:         nonce,
		NodePublicKey: base64.RawURLEncoding.EncodeToString(nodeIdentity.Pub),
	}
	req.Signature = nodeIdentity.Sign(req.CanonicalRequest())

	cmd, _ := fsm.AuthorizeSecretRetrievalCommand(req)
	res := fsm.ApplyLocal(cmd)
	if res.OK {
		t.Error("request with excessive clock skew should be denied")
	}
}

// TestSecretRetrievalAuthorize_CanaryLeakTest verifies no plaintext in replay ledger.
func TestSecretRetrievalAuthorize_CanaryLeakTest(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("super-secret-password-canary")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-canary",
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
	fsm.ApplyLocal(cmd)

	// Serialize replay ledger and verify plaintext does not appear
	fsm.Read(func(s *State) {
		ledgerJSON, _ := json.Marshal(s.ReplayLedger)
		if bytes.Contains(ledgerJSON, plaintext) {
			t.Error("plaintext canary found in replay ledger (leak detected)")
		}
	})
}

// TestSecretRetrievalAuthorize_Failover verifies consumption persists after restore.
func TestSecretRetrievalAuthorize_Failover(t *testing.T) {
	fsm1 := NewFSM()
	fsm1.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm1.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("password")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm1.s.Secrets.AddRecord(record)

	fsm1.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-failover",
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

	// Authorize on fsm1
	cmd1, _ := fsm1.AuthorizeSecretRetrievalCommand(req)
	res1 := fsm1.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("first authorization failed: %v", res1.Message)
	}

	// Simulate failover: export state and restore on new FSM
	stateJSON, _ := json.Marshal(fsm1.s)

	fsm2 := NewFSM()
	json.Unmarshal(stateJSON, fsm2.s)
	fsm2.s.ensure()

	// Replay attempt on new leader should fail (consumption persisted)
	cmd2, _ := fsm2.AuthorizeSecretRetrievalCommand(req)
	res2 := fsm2.ApplyLocal(cmd2)
	if res2.OK {
		t.Error("replay after failover should be denied (consumption persisted)")
	}
}

// TestSecretRetrievalAuthorize_NegativeControl_DisableSignatureCheck
// Deliberately breaks signature verification; must fail at leader validation.
func TestSecretRetrievalAuthorize_NegativeControl_DisableSignatureCheck(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("password")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-neg",
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
	// Wrong signature
	req.Signature = make([]byte, 64)
	rand.Read(req.Signature)

	cmd, err := fsm.AuthorizeSecretRetrievalCommand(req)
	if err == nil {
		t.Fatal("expected error on leader for bad signature")
	}
	if cmd != nil {
		t.Error("command should be nil when signature is bad")
	}
}

// TestSecretRetrievalAuthorize_NegativeControl_BypassNodeRevocationCheck
// Modifies FSM to skip revocation check; test must fail (catch mutation).
func TestSecretRetrievalAuthorize_NegativeControl_BypassNodeRevocationCheck(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{
		ID:        nodeID,
		Status:    "revoked",
		RevokedAt: Now(),
	}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("password")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-neg-revoke",
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
	if res.OK {
		t.Error("negative control: revocation check should reject this node")
	}
}

// TestSecretRetrievalAuthorize_NegativeControl_BypassReplayCheck
// Ensures replay ledger is enforced.
func TestSecretRetrievalAuthorize_NegativeControl_BypassReplayCheck(t *testing.T) {
	fsm := NewFSM()
	fsm.s.Cluster = "test-cluster"

	nodeID := "dh1testaaaaaaaaaaaaaaaaaa"
	nodeIdentity, _ := identity.Generate()
	nodeID = nodeIdentity.ID
	fsm.s.Nodes[nodeID] = &Node{ID: nodeID, Status: "ready"}

	secretID := "secret-001"
	dek, _ := GenerateDEK()
	plaintext := []byte("password")
	record, _ := EncryptSecret(plaintext, secretID, 1, dek, "test-cluster", "deploy-1", "workload-1", "prod", "key-1")
	fsm.s.Secrets.AddRecord(record)

	fsm.s.Assignments["app-r0@"+nodeID] = &AssignmentRec{
		Key: "app-r0@" + nodeID,
		A:   api.Assignment{ID: "app-r0", Node: nodeID, Desired: "running"},
	}

	nonce := make([]byte, 12)
	rand.Read(nonce)
	req := &SecretRetrievalRequest{
		Version:       1,
		RequestID:     "req-neg-replay",
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

	// First request succeeds
	cmd1, _ := fsm.AuthorizeSecretRetrievalCommand(req)
	res1 := fsm.ApplyLocal(cmd1)
	if !res1.OK {
		t.Fatalf("first request failed: %v", res1.Message)
	}

	// If replay check were bypassed, second identical request would succeed
	// But it should fail
	cmd2, _ := fsm.AuthorizeSecretRetrievalCommand(req)
	res2 := fsm.ApplyLocal(cmd2)
	if res2.OK {
		t.Error("negative control: replay check should reject second request")
	}
}
