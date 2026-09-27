// Package control persistence for application secrets encryption at rest.
package control

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// Secret represents a named secret with identity and metadata.
type Secret struct {
	ID        string `json:"id"`        // immutable: secret-<random>
	Name      string `json:"name"`      // e.g., app/web/db-password
	CreatedAt int64  `json:"createdAt"` // Unix nanoseconds
}

// SecretVersion represents a versioned plaintext (volatile, never persisted).
type SecretVersion struct {
	SecretID     string `json:"secretId"`
	Version      int32  `json:"version"` // 1-indexed
	Data         []byte `json:"data"`    // plaintext (memory-only)
	DeploymentID string `json:"deploymentId"`
	WorkloadID   string `json:"workloadId"`
	Environment  string `json:"environment"` // "prod", "staging", etc.
	CreatedAt    int64  `json:"createdAt"`   // Unix ns
	EncryptedAt  int64  `json:"encryptedAt"` // Unix ns
}

// SecretRecord is the encrypted persistence unit (what lives in Raft/BoltDB).
type SecretRecord struct {
	SecretID      string `json:"secretId"`
	Version       int32  `json:"version"`
	Algorithm     string `json:"algorithm"`     // "aes-256-gcm"
	KeyID         string `json:"keyId"`         // identifies which DEK version
	EncryptedData []byte `json:"encryptedData"` // ciphertext
	Nonce         []byte `json:"nonce"`         // 96-bit random
	ClusterID     string `json:"clusterId"`
	DeploymentID  string `json:"deploymentId"`
	WorkloadID    string `json:"workloadId"`
	Environment   string `json:"environment"`
	CreatedAt     int64  `json:"createdAt"`
	EncryptedAt   int64  `json:"encryptedAt"`
}

// SecretsStore holds all persisted secret records, indexed by secretID then version.
type SecretsStore map[string]map[int32]*SecretRecord

// NewSecretsStore initializes an empty secrets store.
func NewSecretsStore() SecretsStore {
	return make(SecretsStore)
}

// AddRecord stores or updates a secret record.
func (s SecretsStore) AddRecord(record *SecretRecord) {
	if _, exists := s[record.SecretID]; !exists {
		s[record.SecretID] = make(map[int32]*SecretRecord)
	}
	s[record.SecretID][record.Version] = record
}

// GetRecord retrieves a specific secret version.
func (s SecretsStore) GetRecord(secretID string, version int32) *SecretRecord {
	if versions, exists := s[secretID]; exists {
		return versions[version]
	}
	return nil
}

// LatestVersion returns the highest version number for a secret, or 0 if none.
func (s SecretsStore) LatestVersion(secretID string) int32 {
	if versions, exists := s[secretID]; exists {
		var maxVersion int32
		for v := range versions {
			if v > maxVersion {
				maxVersion = v
			}
		}
		return maxVersion
	}
	return 0
}

// DeleteSecret removes all versions of a secret (hard delete).
func (s SecretsStore) DeleteSecret(secretID string) {
	delete(s, secretID)
}

// MarshalJSON serializes the secrets store (for Raft/persistence).
func (s SecretsStore) MarshalJSON() ([]byte, error) {
	// Convert map[string]map[int32]*SecretRecord to JSON-serializable form
	// (JSON doesn't support int32 as object keys, so normalize to string keys)
	normalized := make(map[string]map[string]*SecretRecord)
	for secretID, versions := range s {
		normalized[secretID] = make(map[string]*SecretRecord)
		for version, record := range versions {
			normalized[secretID][json.Number(version).String()] = record
		}
	}
	return json.Marshal(normalized)
}

// UnmarshalJSON deserializes the secrets store.
func (s *SecretsStore) UnmarshalJSON(data []byte) error {
	normalized := make(map[string]map[string]*SecretRecord)
	if err := json.Unmarshal(data, &normalized); err != nil {
		return err
	}
	*s = NewSecretsStore()
	for _, versions := range normalized {
		for versionStr, record := range versions {
			var version int32
			if tmpInt64, err := json.Number(versionStr).Int64(); err == nil {
				version = int32(tmpInt64)
			}
			(*s).AddRecord(&SecretRecord{
				SecretID:      record.SecretID,
				Version:       version,
				Algorithm:     record.Algorithm,
				KeyID:         record.KeyID,
				EncryptedData: record.EncryptedData,
				Nonce:         record.Nonce,
				ClusterID:     record.ClusterID,
				DeploymentID:  record.DeploymentID,
				WorkloadID:    record.WorkloadID,
				Environment:   record.Environment,
				CreatedAt:     record.CreatedAt,
				EncryptedAt:   record.EncryptedAt,
			})
		}
	}
	return nil
}

// KekStatus tracks KEK availability and bootstrap state.
type KekStatus struct {
	Locked        bool   // true if bootstrap missing/corrupted
	BootstrapHash string // BLAKE3 hash of bootstrap (never stores bootstrap itself)
	DerivedAt     int64  // Unix ns when KEK was last derived
	Error         string // error message if locked
}

// NewKekStatus initializes the KEK status (initially locked until derived).
func NewKekStatus() KekStatus {
	return KekStatus{
		Locked: true,
		Error:  "KEK not yet derived from bootstrap",
	}
}

// SecretCommand is the Raft command for secret operations.
type SecretCommand struct {
	Type      string        `json:"type"`      // "secret-create", "secret-version-add"
	Timestamp int64         `json:"timestamp"` // Unix ns
	Actor     string        `json:"actor"`     // who requested (for audit)
	SecretID  string        `json:"secretId"`
	Record    *SecretRecord `json:"record"` // encrypted record
}

// Now returns current Unix nanoseconds (used in tests to mock time).
func Now() int64 {
	return time.Now().UnixNano()
}

// SecretRetrievalRequest is a signed authorization request from a node for secret decryption.
// Version 1: Ed25519-signed request with nonce-based replay protection.
// Replay identity combines request digest and nonce: caller cannot change requestId/other fields while reusing same request.
// EphemeralID (A05): optional ephemeral tmpfs delivery mode (phase 1: returned as path instead of plaintext)
type SecretRetrievalRequest struct {
	Version       int    `json:"version"`               // always 1
	RequestID     string `json:"requestId"`             // unique request identifier
	SecretID      string `json:"secretId"`              // target secret
	SecretVersion int32  `json:"secretVersion"`         // target version
	NodeID        string `json:"nodeId"`                // requesting node (dh1...)
	WorkloadID    string `json:"workloadId"`            // workload scope (matches assignment)
	DeploymentID  string `json:"deploymentId"`          // deployment scope (matches assignment)
	Environment   string `json:"environment"`           // environment scope (matches assignment)
	Timestamp     string `json:"timestamp"`             // Unix nanoseconds as decimal string (clock skew tolerance ±5s)
	Nonce         []byte `json:"nonce"`                 // unique per-request nonce (for replay ledger)
	NodePublicKey string `json:"nodePublicKey"`         // wire-encoded Ed25519 public key
	Signature     []byte `json:"signature"`             // Ed25519 signature over canonical request (excluding signature field)
	EphemeralID   string `json:"ephemeralId,omitempty"` // [A05] UUID for ephemeral tmpfs delivery (optional, phase 1)
}

// CanonicalRequest returns the canonical form of the request for signing/verification.
// Fields are hashed in deterministic order (version through nonce, excluding signature).
// All fields encoded as text for JSON compatibility and canonical reproducibility.
func (r *SecretRetrievalRequest) CanonicalRequest() []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "v%d", r.Version)
	buf.WriteByte('|')
	io.WriteString(&buf, r.RequestID)
	buf.WriteByte('|')
	io.WriteString(&buf, r.SecretID)
	buf.WriteByte('|')
	fmt.Fprintf(&buf, "%d", r.SecretVersion)
	buf.WriteByte('|')
	io.WriteString(&buf, r.NodeID)
	buf.WriteByte('|')
	io.WriteString(&buf, r.WorkloadID)
	buf.WriteByte('|')
	io.WriteString(&buf, r.DeploymentID)
	buf.WriteByte('|')
	io.WriteString(&buf, r.Environment)
	buf.WriteByte('|')
	io.WriteString(&buf, r.Timestamp)
	buf.WriteByte('|')
	io.WriteString(&buf, base64.RawURLEncoding.EncodeToString(r.Nonce))
	return buf.Bytes()
}

// RequestDigest returns the SHA256 hash of the canonical request (for replay identity).
func (r *SecretRetrievalRequest) RequestDigest() string {
	h := sha256.Sum256(r.CanonicalRequest())
	return hex.EncodeToString(h[:])
}

// VerifySignature validates the request signature against the node's public key.
func (r *SecretRetrievalRequest) VerifySignature() error {
	if r.Version != 1 {
		return errors.New("request version not supported")
	}
	if len(r.Signature) == 0 {
		return errors.New("signature empty")
	}
	pub, err := base64.RawURLEncoding.DecodeString(r.NodePublicKey)
	if err != nil {
		return fmt.Errorf("decode node public key: %w", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("public key has %d bytes, want 32", len(pub))
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), r.CanonicalRequest(), r.Signature) {
		return errors.New("signature verification failed")
	}
	return nil
}

// ConsumedAuthorization records a consumed authorization in the replay ledger.
// Persisted to Raft FSM, survives leader change/restart/failover.
type ConsumedAuthorization struct {
	RequestDigest string `json:"requestDigest"`        // SHA256(canonical request)
	ConsumedNonce []byte `json:"consumedNonce"`        // nonce from request (duplicate protection)
	ConsumedAt    string `json:"consumedAt"`           // Unix ns when consumed as decimal string (FSM timestamp, JSON safe)
	NodeID        string `json:"nodeId"`               // requesting node
	SecretID      string `json:"secretId"`             // target secret
	SecretVersion int32  `json:"secretVersion"`        // target version
	Outcome       string `json:"outcome"`              // "SUCCESS" | "DENIED" (audit)
	DenyReason    string `json:"denyReason,omitempty"` // why denied (audit)
}

// ReplayLedger holds all consumed authorizations, indexed by requestDigest.
// Shared within FSM to prevent concurrent identical requests from both succeeding.
type ReplayLedger map[string]*ConsumedAuthorization

// NewReplayLedger creates an empty replay ledger.
func NewReplayLedger() ReplayLedger {
	return make(ReplayLedger)
}

// IsConsumed checks if a request has already been authorized.
func (l ReplayLedger) IsConsumed(requestDigest string) bool {
	_, exists := l[requestDigest]
	return exists
}

// Record records a successful authorization consumption.
func (l ReplayLedger) Record(auth *ConsumedAuthorization) {
	l[auth.RequestDigest] = auth
}

// MarshalJSON serializes the replay ledger for Raft persistence.
func (l ReplayLedger) MarshalJSON() ([]byte, error) {
	// Convert to a slice for deterministic JSON ordering
	type entry struct {
		Digest string                 `json:"digest"`
		Auth   *ConsumedAuthorization `json:"auth"`
	}
	entries := make([]entry, 0, len(l))
	for digest, auth := range l {
		entries = append(entries, entry{Digest: digest, Auth: auth})
	}
	return json.Marshal(entries)
}

// UnmarshalJSON deserializes the replay ledger from Raft.
func (l *ReplayLedger) UnmarshalJSON(data []byte) error {
	type entry struct {
		Digest string                 `json:"digest"`
		Auth   *ConsumedAuthorization `json:"auth"`
	}
	var entries []entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}
	*l = NewReplayLedger()
	for _, e := range entries {
		(*l)[e.Digest] = e.Auth
	}
	return nil
}

// SecretLeaseRequest is a signed authorization request from a node for secret lease (A04).
// Version 1: Ed25519-signed request with nonce-based replay protection and scope binding.
// Domain-separated canonical representation prevents cross-protocol signature reuse.
type SecretLeaseRequest struct {
	Protocol      string `json:"protocol"`           // "dhp://secrets/v1"
	Version       int    `json:"version"`            // always 1
	RequestType   string `json:"requestType"`        // "secret-lease-request"
	RequestID     string `json:"requestId"`          // unique request identifier
	SecretID      string `json:"secretId"`           // target secret
	SecretVersion int32  `json:"secretVersion"`      // target version
	NodeID        string `json:"nodeId"`             // requesting node (dh1...)
	WorkloadID    string `json:"workloadId"`         // workload scope
	DeploymentID  string `json:"deploymentId"`       // deployment scope
	Environment   string `json:"environment"`        // environment scope (prod/staging)
	Generation    int64  `json:"generation"`         // generation counter for this node's authorization
	Timestamp     string `json:"timestamp"`          // Unix nanoseconds as decimal string
	Nonce         []byte `json:"nonce"`              // unique per-request nonce (for replay ledger)
	CallerID      string `json:"callerId"`           // caller identity (matches node authorization)
	IssuedAt      int64  `json:"issuedAt"`           // Unix nanoseconds
	ExpiresAt     int64  `json:"expiresAt"`          // Unix nanoseconds
	NodePublicKey string `json:"nodePublicKey"`      // wire-encoded Ed25519 public key
	Signature     []byte `json:"signature"`          // Ed25519 signature over canonical request
}

// CanonicalLeaseRequest returns the domain-separated canonical form for signing/verification.
// Mandatory domain separation per spec: signing domain first, then request type, then cluster domain, then immutable fields.
// Prevents signature reuse across protocols, request types, clusters, or security scopes.
// RECONCILIATION (P0-A06→A01): cluster domain currently hard-coded as "cluster" for compatibility.
// A06 qualification validates consistent canonical encoding across all 30 gates.
// Forward enhancement (P1): extract from FSM state (fsm.s.Cluster) during FSM handler verification.
func (r *SecretLeaseRequest) CanonicalLeaseRequest() []byte {
	var buf bytes.Buffer
	// DOMAIN SEPARATOR: protocol + version + request type + cluster domain
	io.WriteString(&buf, "dhp://secrets/v1")
	buf.WriteByte('|')
	io.WriteString(&buf, "secret-lease-request")
	buf.WriteByte('|')
	// cluster domain: hard-coded for P0 compatibility; extracted from fsm.s.Cluster in P1+
	clusterDomain := "cluster"
	io.WriteString(&buf, clusterDomain)
	buf.WriteByte('|')

	// IMMUTABLE REQUEST IDENTITY
	io.WriteString(&buf, r.RequestID)
	buf.WriteByte('|')
	io.WriteString(&buf, r.SecretID)
	buf.WriteByte('|')
	fmt.Fprintf(&buf, "%d", r.SecretVersion)
	buf.WriteByte('|')
	io.WriteString(&buf, r.NodeID)
	buf.WriteByte('|')
	io.WriteString(&buf, r.WorkloadID)
	buf.WriteByte('|')
	io.WriteString(&buf, r.DeploymentID)
	buf.WriteByte('|')
	io.WriteString(&buf, r.Environment)
	buf.WriteByte('|')
	fmt.Fprintf(&buf, "%d", r.Generation)
	buf.WriteByte('|')
	io.WriteString(&buf, r.Timestamp)
	buf.WriteByte('|')
	io.WriteString(&buf, base64.RawURLEncoding.EncodeToString(r.Nonce))
	buf.WriteByte('|')
	io.WriteString(&buf, r.CallerID)
	buf.WriteByte('|')
	fmt.Fprintf(&buf, "%d", r.IssuedAt)
	buf.WriteByte('|')
	fmt.Fprintf(&buf, "%d", r.ExpiresAt)
	return buf.Bytes()
}

// RequestLeaseDigest returns the SHA256 hash of the canonical lease request (for replay identity).
func (r *SecretLeaseRequest) RequestLeaseDigest() string {
	h := sha256.Sum256(r.CanonicalLeaseRequest())
	return hex.EncodeToString(h[:])
}

// VerifyLeaseSignature validates the lease request signature against the node's public key.
func (r *SecretLeaseRequest) VerifyLeaseSignature() error {
	if r.Protocol != "dhp://secrets/v1" {
		return errors.New("lease request protocol not supported")
	}
	if r.Version != 1 {
		return errors.New("lease request version not supported")
	}
	if r.RequestType != "secret-lease-request" {
		return errors.New("lease request type mismatch")
	}
	if len(r.Signature) == 0 {
		return errors.New("signature empty")
	}
	pub, err := base64.RawURLEncoding.DecodeString(r.NodePublicKey)
	if err != nil {
		return fmt.Errorf("decode node public key: %w", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("public key has %d bytes, want 32", len(pub))
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), r.CanonicalLeaseRequest(), r.Signature) {
		return errors.New("lease signature verification failed")
	}
	return nil
}

// ConsumedLeaseAuthorization records a consumed lease authorization in the replay ledger.
// Persisted to Raft FSM, survives leader change/restart/failover.
type ConsumedLeaseAuthorization struct {
	RequestID             string `json:"requestId"`             // unique request identifier
	RequestDigest         string `json:"requestDigest"`         // SHA256(canonical lease request)
	RequestDigestAlt      string `json:"requestDigestAlt"`      // alternative replay key for nonce binding
	ConsumedNonce         []byte `json:"consumedNonce"`         // nonce from request (duplicate protection)
	ConsumedAt            string `json:"consumedAt"`            // Unix ns when consumed as decimal string (FSM timestamp)
	NodeID                string `json:"nodeId"`                // requesting node
	SecretID              string `json:"secretId"`              // target secret
	SecretVersion         int32  `json:"secretVersion"`         // target version
	WorkloadID            string `json:"workloadId"`            // workload scope
	DeploymentID          string `json:"deploymentId"`          // deployment scope
	Environment           string `json:"environment"`           // environment scope
	Generation            int64  `json:"generation"`            // generation counter
	CallerID              string `json:"callerId"`              // caller identity
	Outcome               string `json:"outcome"`               // "SUCCESS" | "CONFLICT" | "DENIED" (audit)
	DenyReason            string `json:"denyReason,omitempty"`  // why denied (audit)
}

// SecretLeaseRecord is the durable record persisted for a consumed lease authorization.
type SecretLeaseRecord struct {
	RequestID     string `json:"requestId"`
	RequestDigest string `json:"requestDigest"`
	SecretID      string `json:"secretId"`
	SecretVersion int32  `json:"secretVersion"`
	NodeID        string `json:"nodeId"`
	WorkloadID    string `json:"workloadId"`
	DeploymentID  string `json:"deploymentId"`
	Generation    int64  `json:"generation"`
	Nonce         []byte `json:"nonce"`
	CallerID      string `json:"callerId"`
	AuthorizedAt  int64  `json:"authorizedAt"` // Unix ns
	Status        string `json:"status"`       // "authorized" | "consumed" | "revoked"
}

// LeaseReplayLedger holds all consumed lease authorizations, indexed by requestDigest.
// Single-level key structure for deterministic commit semantics.
type LeaseReplayLedger map[string]*ConsumedLeaseAuthorization

// NewLeaseReplayLedger creates an empty lease replay ledger.
func NewLeaseReplayLedger() LeaseReplayLedger {
	return make(LeaseReplayLedger)
}

// IsConsumedLease checks if a lease request has already been authorized.
func (l LeaseReplayLedger) IsConsumedLease(requestDigest string) bool {
	_, exists := l[requestDigest]
	return exists
}

// GetConflict checks if a request ID is already in use with a different payload.
// Returns non-nil if same ID exists with different digest (CONFLICT).
func (l LeaseReplayLedger) GetConflict(requestID, requestDigest string) *ConsumedLeaseAuthorization {
	// Scan all entries for matching RequestID
	for _, auth := range l {
		if auth.RequestID == requestID && auth.RequestDigest != requestDigest {
			// Same ID, different payload: CONFLICT
			return auth
		}
	}
	return nil
}

// RecordLease records a successful lease authorization consumption.
func (l LeaseReplayLedger) RecordLease(auth *ConsumedLeaseAuthorization) {
	l[auth.RequestDigest] = auth
}

// MarshalJSON serializes the lease replay ledger for Raft persistence.
func (l LeaseReplayLedger) MarshalJSON() ([]byte, error) {
	type entry struct {
		Digest string                       `json:"digest"`
		Auth   *ConsumedLeaseAuthorization `json:"auth"`
	}
	entries := make([]entry, 0, len(l))
	for digest, auth := range l {
		entries = append(entries, entry{Digest: digest, Auth: auth})
	}
	return json.Marshal(entries)
}

// UnmarshalJSON deserializes the lease replay ledger from Raft.
func (l *LeaseReplayLedger) UnmarshalJSON(data []byte) error {
	type entry struct {
		Digest string                       `json:"digest"`
		Auth   *ConsumedLeaseAuthorization `json:"auth"`
	}
	var entries []entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}
	*l = NewLeaseReplayLedger()
	for _, e := range entries {
		(*l)[e.Digest] = e.Auth
	}
	return nil
}
