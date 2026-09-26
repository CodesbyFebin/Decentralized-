# ADR-0012: SEC-P0-A01-A03 Cryptographic Storage Core Implementation

**Status:** DRAFT  
**Date:** 2026-09-26  
**Stage:** SEC-P0-A01-A03  
**Scope:** Cryptographic storage core only (no retrieval, rotation, delivery, nonce ledger)

## Objective

Prove application secrets can enter the control plane, survive Raft replication/snapshot/restart, and remain recoverable only with authorized key material—without plaintext entering persistent replicated state.

## Design Decisions

### 1. AEAD Algorithm: AES-256-GCM

**Choice:** `crypto/aes` + `crypto/cipher.NewGCM` (golang.org/x/crypto v0.57.0)

**Rationale:**
- Hardware-accelerated on modern CPUs (AES-NI)
- Audited standard (NIST, FIPS)
- 256-bit key strength matches system's Ed25519/Curve25519 hierarchy
- 96-bit random nonce per DEK (sufficient for per-version independence)
- Provided by Go stdlib `crypto/cipher`

**Alternative rejected:** ChaCha20-Poly1305 (deferred to future optimization; GCM chosen for broader hardware support)

### 2. KEK Custody: Operator-Provided Bootstrap + Cluster-Held Wrapper

**Concrete mechanism:**

```
Startup (A03):
1. Member reads DECENTRALIZED_KEK_BOOTSTRAP from environment
   - Format: base64url-encoded 32-byte random material
   - Must be identical across all members
   - Never persisted in Raft/BoltDB
   - Fail-closed if absent or corrupted

2. Member derives KEK from bootstrap:
   KEK = BLAKE3(bootstrap || "v1" || clusterID)[:32]

3. Member stores wrapped DEK in Raft state (not KEK itself):
   wrappedDEK = AES-256-GCM(KEK, randomNonce, DEK, clusterID)

4. On restart:
   - Read bootstrap from environment (must match prior startup)
   - Derive identical KEK
   - Decrypt wrappedDEK to recover DEK
   - Fail-closed if bootstrap missing or different

**Invariant:** Theft of Raft/BoltDB alone cannot recover secrets:
- wrappedDEK is authenticated (AAD = clusterID)
- KEK cannot be derived without bootstrap material
- Tampering with wrappedDEK detected (auth tag fails)
- Member crashes → must re-read bootstrap to unlock
```

### 3. SecretRecord Persistence Model

```go
// Secret identity (immutable per secret name)
type Secret struct {
    ID            string // e.g., "secret-abc123def456"
    Name          string // e.g., "app/web/db-password"
    CreatedAt     int64  // Unix ns
}

// Versioned plaintext data (only exists in memory during ops)
type SecretVersion struct {
    SecretID      string
    Version       int32  // 1-indexed, increments on each write
    Data          []byte // plaintext (never persisted)
    
    // Metadata for AAD binding
    DeploymentID  string
    WorkloadID    string
    Environment   string // "prod", "staging", etc.
    
    CreatedAt     int64  // Unix ns
    EncryptedAt   int64  // Unix ns (when encrypted)
}

// Encrypted storage record (what persists in Raft/BoltDB)
type SecretRecord struct {
    SecretID      string    // references Secret.ID
    Version       int32     // matches SecretVersion.Version
    
    // Authenticated encryption
    Algorithm     string    // "aes-256-gcm"
    KeyID         string    // identifies which DEK was used (deployment-specific)
    EncryptedData []byte    // AES-256-GCM(DEK, nonce, plaintext, AAD)
    Nonce         []byte    // 96-bit random nonce (persisted alongside ciphertext)
    AuthTag       []byte    // AEAD authentication tag (embedded in EncryptedData)
    
    // Canonical AAD (scope binding)
    ClusterID     string
    DeploymentID  string
    WorkloadID    string
    Environment   string
    
    CreatedAt     int64
    EncryptedAt   int64
}

// Replicated state addition
type State struct {
    // ... existing fields ...
    
    // Secrets storage (indexed by SecretID + Version)
    Secrets map[string]map[int32]*SecretRecord // [secretID][version]record
}
```

### 4. Cryptographic Operations

```go
// GenerateDEK creates cryptographically random 32-byte DEK per secret version
func GenerateDEK() [32]byte { ... }

// EncryptSecret encrypts plaintext using DEK-per-version with AEAD
// Returns SecretRecord with ciphertext, nonce, and authenticated metadata
func EncryptSecret(plaintext []byte, version *SecretVersion, dek [32]byte, kekID string) (*SecretRecord, error)

// DecryptSecret recovers plaintext from SecretRecord using DEK
// Verifies AAD scope binding; fails if tampering detected
func DecryptSecret(record *SecretRecord, dek [32]byte) ([]byte, error)

// WrapDEK encrypts DEK using KEK for cluster replication
// Returns wrappedDEK with authentication tag
func WrapDEK(dek [32]byte, kek [32]byte, clusterID string) ([]byte, error)

// UnwrapDEK decrypts wrapped DEK using KEK
// Fails if tampering detected or KEK cannot be derived
func UnwrapDEK(wrappedDEK []byte, kek [32]byte, clusterID string) ([32]byte, error)

// DeriveKEK derives KEK from bootstrap material (never called if bootstrap missing)
func DeriveKEK(bootstrap string, clusterID string) [32]byte
```

### 5. Raft Integration

**New FSM commands:**

```go
type SecretCommand struct {
    Type      string      // "secret-create", "secret-version-add"
    Timestamp int64       // Unix ns
    Actor     string      // who requested (for audit)
    Data      SecretRecord // already encrypted before proposing
}

// Command processing in FSM.Apply:
// 1. Receive encrypted SecretRecord (plaintext already encrypted before Raft)
// 2. Validate structure and AAD scope
// 3. Store in State.Secrets[secretID][version] = record
// 4. Return success (never decrypt in FSM)
```

**Fail-closed behavior:**

```go
// If KEK unavailable (bootstrap missing/corrupted):
// - Member enters locked state
// - Rejects all secret operations
// - Logs "LOCKED: cannot derive KEK from bootstrap"
// - Requires operator intervention (provide bootstrap, restart)
```

### 6. Canonical Authenticated Metadata (AAD)

All AEAD operations use consistent AAD:

```go
aad := fmt.Sprintf("%s|%s|%s|%s|%d",
    record.ClusterID,
    record.DeploymentID,
    record.WorkloadID,
    record.Environment,
    record.Version,
)
```

Tampering with any of these values causes decryption to fail (auth tag mismatch).

### 7. Tests (Implementation Phases)

**Phase 1: Cryptographic primitives**
- AEAD round-trip (plaintext → encrypted → decrypted)
- Wrong DEK rejection (decrypt with wrong key → auth tag fails)
- Ciphertext tamper detection (modify bytes → auth tag fails)
- Nonce tamper detection (modify nonce → decryption fails)
- AAD tamper detection (modify scope fields → auth tag fails)

**Phase 2: Key custody**
- DEK generation independence (each version gets unique DEK)
- KEK derivation from bootstrap
- Wrapped DEK tamper rejection
- Missing bootstrap → fail-closed
- Different bootstrap per member → fail-closed (leader cannot decrypt)

**Phase 3: Raft persistence**
- Propose encrypted SecretRecord (plaintext never enters Raft)
- FSM Apply stores record
- Snapshot includes encrypted records
- Restart: read bootstrap → derive KEK → unwrap DEK → decrypt test secret

**Phase 4: Persistence scanning (canaries)**
- BoltDB scan: verify no plaintext secrets in raft.db
- Snapshot scan: verify no plaintext secrets in snapshots
- Export scan (if touched): verify no plaintext in backup
- Memory scan: verify plaintext only in volatile working memory

**Phase 5: Negative controls**
- Disable AAD scope binding in one test → scope-tamper test MUST fail
- Introduce test-only plaintext persistence → canary MUST detect
- Disable auth tag verification → tamper test MUST fail
- Restore production implementation after each mutation

### 8. Evidence Format

```json
{
  "evidenceId": "SEC-P0-A01-A03-<timestamp>",
  "qualificationId": "SEC-P0-A01-A03",
  "sourceSHA": "git commit hash",
  "branch": "claude/friendly-gauss-kfxoc2",
  "environment": "Linux container (cloud sandbox)",
  "algorithm": "aes-256-gcm",
  "keyVersion": "v1",
  "keyCustodyMode": "operator-local-bootstrap + cluster-held-wrapped",
  "testResults": {
    "cryptoRoundTrip": "PASS",
    "wrongKeyRejection": "PASS",
    "ciphertextTamper": "PASS",
    "nonceTamper": "PASS",
    "aadTamper": "PASS",
    "wrappedDEKTamper": "PASS",
    "missingKeystoreFail": "PASS",
    "restart": "PASS",
    "secretVersionIndependence": "PASS",
    "raftPersistence": "PASS",
    "boltdbCanary": "PASS",
    "snapshotCanary": "PASS",
    "exportCanary": "PASS"
  },
  "negativeControls": {
    "aadBindingDisable": "FAIL (expected)",
    "plaintextPersistence": "FAIL (expected)",
    "authTagBypass": "FAIL (expected)"
  },
  "startedAt": "2026-09-26T00:00:00Z",
  "completedAt": "2026-09-26T01:30:00Z",
  "signer": "codesbyfebin@gmail.com",
  "signature": "base64url(signature)"
}
```

## Non-Scope (Deferred)

- `/run/secrets` delivery → A05
- Workload retrieval authorization → A04
- Nonce consumption ledger → A04 (Raft-backed)
- Automatic rotation → A06
- LocalCAKey migration → separate track
- DePIN, marketplace, settlement → future phases

## Security Invariants

1. **Plaintext encrypted before Raft:** Application secret is encrypted with DEK before constructing raft.Command
2. **No plaintext in persistent state:** Raft logs, BoltDB, snapshots, exports must contain only EncryptedData (ciphertext)
3. **Fail-closed on key loss:** Missing/corrupted bootstrap → all secret operations reject with "LOCKED" status
4. **Scope binding enforced:** AAD includes deploymentID, workloadID, environment; tampering detected by AEAD tag verification

## Implementation Order

1. `pkg/control/secrets.go` - SecretRecord, SecretVersion, state.Secrets field
2. `pkg/control/crypto.go` - EncryptSecret, DecryptSecret, WrapDEK, UnwrapDEK, DeriveKEK
3. `pkg/control/state.go` - Add Secrets field to State struct
4. `pkg/control/fsm.go` - Add secret-create, secret-version-add commands
5. `pkg/control/server.go` - Add bootstrap check on startup, KEK derivation, fail-closed init
6. `pkg/control/secrets_test.go` - Unit tests (crypto, tampering, key custody)
7. `pkg/control/integration_test.go` - Raft, snapshot, restart tests
8. `evidence/` - Evidence record generation and verification
9. Documentation updates to ADR-0011 reflecting concrete choices

## Next Gate: SEC-P0-A01-A03 = PASS

Proceed to A04 only when:
- All tests pass
- No mandatory tests skipped
- Plaintext canary absent from persistent surfaces
- Negative controls fail as expected
- Evidence signature verifies
- Missing key material fails closed
