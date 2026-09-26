# ADR 0011: Sovereign Secrets Architecture

**Status:** DESIGN (SEC-P0-A01-A02, not yet implemented)

**Date:** 2026-09-26

**Decision Drivers:**
- Secrets must never be stored in plaintext in Raft/BoltDB
- Workload isolation (RUNTIME-P0-A01) provides process boundary; secrets require cryptographic boundary
- Key custody must survive node replacement, restarts, and multi-member Raft
- Backup/restore must not leak plaintext without explicit operator recovery material
- Authorization must bind node identity, workload assignment, and scope; cluster membership alone is insufficient

---

## Context

### Existing Trust Boundaries (Preserved)

1. **Node Identity (Ed25519):** Node-local private key, never replicated. Control plane holds public key only.
2. **WireGuard Keys (X25519):** Node-local private key, never replicated. Used for mesh transport authentication (cryptokey routing).
3. **Capability Root:** Operator-held key used to sign API bearer tokens (capability.Verify). Already in place for coarse-grained (api.read/api.write/api.admin) authorization.
4. **Local CA Key:** Currently stored plaintext in replicated `State.LocalCAKey`. Used for TLS certificate issuance.
5. **Raft Consensus:** BoltDB logs all State mutations. 3 snapshots retained. Backups export State.
6. **Audit Ledger:** Immutable log in State; never encrypted historically.

### Key Naming (Disambiguated)

To prevent overloading "root key," we name each trust root explicitly:

| Name | Purpose | Holder | Rotation |
|------|---------|--------|----------|
| **Cluster Identity Root** | Signs rosters, member certs, capabilities | Operator | Annual (ADR 0008) |
| **Secrets Master Key** | Encrypts KEK; per-cluster; survives bootstrap/join | Operator (custody model TBD) | On-demand or annual |
| **KEK (Key Encryption Key)** | Encrypts per-secret DEKs; replicable across members | Cluster (protected in memory) | On operator request |
| **DEK (Data Encryption Key)** | Per-secret or per-version; never replicated plaintext | Generated per secret/version | On secret rotate |
| **Node Identity Key** | Signs observations, mesh bindings | Node (local-only) | On-demand (rare) |
| **WireGuard Key** | Mesh transport; X25519 Curve25519 | Node (local-only) | On-demand (rare) |
| **Local CA Key** | TLS CA private key; currently plaintext in State | Cluster (to be migrated) | On-demand |

---

## Security Objective

The secrets subsystem must guarantee all of the following. Do not weaken any.

| # | Guarantee | Mechanism |
|---|-----------|-----------|
| A | Raft never receives application-secret plaintext | Encrypt before proposing command |
| B | BoltDB never contains application-secret plaintext | Same; encrypted-at-rest |
| C | Raft snapshots never contain secret plaintext | FSM never stores plaintext; snapshots contain State only |
| D | PostgreSQL mirror never receives secret plaintext | Mirror layer explicitly excludes secrets |
| E | Ordinary backup/export never contains secret plaintext | Export format excludes wrapped keys; ciphertext only |
| F | dh-agent never persists delivered secret plaintext | Deliver to tmpfs `/run/secrets`; OS cleanup on process exit |
| G | Logs/audit/evidence never contain secret plaintext | Audit entries log only metadata: secretId, scope, actor, result, not value |
| H | One workload cannot read another workload's secret | Authorization binding: `(nodeId, workloadId, deploymentId, secretId)` |
| I | One cluster node cannot read a secret merely by membership | Require workload assignment AND valid authorization token |
| J | Replayed authorization cannot retrieve a secret | Nonce consumed on use; request signed with timestamp; TTL enforced |
| K | Ciphertext modification detected | AEAD (AES-256-GCM or ChaCha20-Poly1305) |
| L | Key rotation does not silently destroy secrets | Version tracking; old DEKs retained for decryption; explicit supersession |
| M | Loss of required root material fails closed | Unavailable KEK → secrets unavailable, not downgraded to plaintext |

---

## Key Custody Model: Operator-Local + Cluster-Held KEK

### Rationale

- **Sovereign P0:** No external KMS/Vault required.
- **Operational:** Operator can recover after node loss without external backup of key material.
- **Restart:** No operator interaction needed for normal restart (KEK held in memory or encrypted in cluster).
- **Multi-member:** KEK must be available to all members; Raft replication works if KEK is protected.

### Model

```
Operator
   ├─ Secrets Master Key
   │  (offline or on-demand; not replicated)
   │
   ├──> generates KEK (on cluster init or operator request)
   │
   └──> supplies KEK to cluster (via authenticated command or deployment)

Cluster
   ├─ KEK
   │  (encrypted in State, or held in memory during operation)
   │  (each member can decrypt using KEK to recover or process secrets)
   │
   ├──> derives DEK per secret/version (using KDF, stored wrapped)
   │
   └──> stores (wrappedDEK, ciphertext, metadata) in Raft
```

### Bootstrap

1. Operator generates `secrets_master_key` (e.g., `dh secret init --master-key-file`)
2. Operator supplies KEK to first cluster member (via secure channel or bootstrapped root auth)
3. Member stores wrapped KEK in State: `State.WrappedKEK`
4. On join, new members receive wrapped KEK via Raft replication
5. Each member can unwrap KEK using locally-derived decryption key (from bootstrap material)

### Restart

- KEK remains in `State.WrappedKEK` (Raft persists it)
- On boot, member loads wrapped KEK, unwraps with bootstrap material
- No operator action needed

### Disaster Recovery

If cluster loses all copies of wrapped KEK:

1. Secrets become unavailable (fail-closed)
2. Operator can re-supply KEK (from backup or re-derive)
3. Member applies re-keying command
4. Secrets become accessible with new KEK version

**Trade-off:** Requires operator to retain `secrets_master_key` offline or in hardware storage. Achieves sovereignt without external KMS.

---

## Cryptographic Envelope

### AEAD Primitive

**Decision:** AES-256-GCM or ChaCha20-Poly1305 (decided at milestone A03)

**Authenticated Associated Data (AAD):**

```
AAD = canonicalJSON({
  "secretId": string,
  "version": int64,
  "deploymentId": string,
  "workloadId": string,
  "environment": string,
  "keyId": string,
  "algorithm": string
})
```

Changing any field makes decryption fail (AEAD integrity check).

### SecretRecord (Raft State)

```json
{
  "secretId": "string (unique per cluster)",
  "version": 1,
  "deploymentId": "string",
  "workloadId": "string",
  "environment": "string (e.g., 'prod', 'staging')",
  
  "algorithm": "AES-256-GCM",
  "keyId": "KEK-1 (or DEK-1 if unwrapped DEK stored)",
  
  "wrappedDEK": "base64(KEK_encrypt(DEK))",
  "nonce": "base64(random 96 bits for GCM)",
  "ciphertext": "base64(AES-GCM(plaintext, AAD, nonce))",
  
  "createdAt": 1695123456000,
  "rotatedAt": 0,
  "lifecycle": "ACTIVE",
  
  "metadata": {
    "source": "deployment manifest",
    "tags": {}
  }
}
```

**In Raft Command:**

```json
{
  "type": "secret-create",
  "ts": 1695123456000,
  "actor": "deployment-controller",
  "data": {
    "secretRecord": { ... as above ... }
  }
}
```

**Never in Raft:**
- Plaintext secret value
- Unwrapped DEK
- Unwrapped KEK
- Secrets Master Key

---

## Secret Identities and Versions

### Lifecycle States

```
CREATED
  ↓
ACTIVE (can be retrieved)
  ↓
SUPERSEDED (next version is ACTIVE; old version still decryptable)
  ↓
REVOKED (cannot be decrypted; marked for deletion)
  ↓
DELETED (tombstone remains for audit)
```

### Version Semantics

- Each secret rotation creates a new version
- Old versions remain decryptable (for rollback/audit)
- Workload must explicitly request a version or get ACTIVE
- Deleting ACTIVE is an error; must supersede first

### Secret ID Format

Suggested: `sec_{deploymentId}_{name}_{hash}` (namespace collision avoidance)

---

## Authorization Model

### Separation of Concerns

**Management Authorization** (api.admin capability):
- Create/delete secrets
- Rotate secrets
- Update scope/metadata
- Revoke versions

**Retrieval Authorization** (new mechanism, per secret):
- Workload on node requests secret
- Control plane verifies workload assignment and scope

### Retrieval Authorization Token

```json
{
  "secretId": "string",
  "deploymentId": "string",
  "workloadId": "string",
  "nodeId": "string",
  "environment": "string",
  "version": 1,
  "nonce": "random-uuid (single-use)",
  "issuedAt": 1695123456000,
  "expiresAt": 1695123456000 + 300000,
  "signature": "Ed25519(requestDigest, control-plane-key)"
}
```

Signed by control-plane member identity (Ed25519).

### Verification Steps

Agent requests secret with token:

```json
{
  "request": {
    "secretId": "sec_...",
    "nodeId": "dh1...",
    "workloadId": "w_...",
    "nonce": "uuid"
  },
  "token": { ... as above ... },
  "requestSignature": "Ed25519(digest, nodeIdentityKey)"
}
```

Control plane verifies:

1. ✓ Token signature valid (control-plane member key)
2. ✓ Request signature valid (node identity key)
3. ✓ Token fields match request fields
4. ✓ Token not expired
5. ✓ Nonce not consumed (check nonce ledger)
6. ✓ Node not revoked (check `Node.Status != "revoked"`)
7. ✓ Workload assigned to node (check `Assignment.Node == nodeId`)
8. ✓ Deployment owns secret (check `Secret.DeploymentId == deployment.Id`)
9. ✓ Environment matches (check `Token.Environment == workload.Environment`)

**Failure → deny (fail-closed).** Never downgrade to plaintext or weaker check.

---

## Transport Authentication

### Assumption from Discovery

Assume agent→control API uses application-layer authentication (TLS + capability bearer token, or mTLS + identity verification). If not, design must add it.

### Secret Retrieval Channel

- Agent sends token + request signature over authenticated channel
- Control plane verifies both signatures and metadata
- Response is plaintext secret (in memory only; never persisted)
- Delivered to agent memory, agent writes to tmpfs (not control plane)

---

## Ephemeral Delivery

### Path

```
Control Plane
   ├─ (verify authorization)
   ├─ (decrypt ciphertext from Raft)
   ├─ (hold plaintext in memory)
   │
   └──> respond with plaintext (single response, not persisted at server)

dh-agent
   ├─ (receive plaintext)
   ├─ (hold in memory)
   │
   └──> write to /run/secrets/{workload_id}/{secret_name}
        ├─ mode 0600 (rw-------)
        ├─ owner: workload UID/GID (from runtime profile)
        ├─ no other acl
        │
        └──> hand fd to runtime
             └──> open(/run/secrets/...) → read secret
```

### Cleanup

| Trigger | Action |
|---------|--------|
| Normal stop | rm /run/secrets/{workload_id} |
| Force stop | rm -rf /run/secrets/{workload_id} |
| Runtime crash | OS tmpfs cleanup on process exit |
| Agent crash | OS tmpfs cleanup; if orphaned, cleanup on next agent start |
| Node reboot | tmpfs cleared (not persistent) |

### Restrictions

- Never write to image layers
- Never add to artifact bundles
- Never include in logs/evidence
- Never persist to disk
- Single-fd delivery (not multiplexed)

---

## Secret Lease Semantics

### Nonce Consumption

Each retrieval authorization includes a `nonce` (UUIDv4).

**Nonce Ledger** (in-memory or persisted, operator-defined):

```
{
  "nonce": "uuid",
  "secret": "secretId",
  "consumed": true,
  "consumedAt": 1695123456000,
  "expiresAt": 1695123456000 + 86400000
}
```

**Constraint:** Nonce can be consumed only once. Second use → deny.

**Expiry:** Old nonce entries discarded after TTL (e.g., 24h).

### Request TTL

Default: 5 minutes (300s). Configurable per deployment.

Rationale: Allows agent to retry on transient network issues; prevents unlimited replay.

---

## Rotation

### Secret Rotation

Operator issues `SecretVersion` with new plaintext.

- New version created with version+1
- New DEK derived for this version
- Old versions remain accessible
- Workload fetches new version on next lease request

### DEK Rotation

Operator requests DEK re-encryption without changing plaintext.

- Same plaintext
- New DEK derived
- Ciphertext re-encrypted with new DEK
- Old DEK discarded (or kept for backward compat, operator decides)

### KEK Rotation

Operator supplies new KEK.

- All wrapped DEKs are re-wrapped with new KEK
- State.WrappedKEK updated
- Old KEK is discarded (or kept for recovery, operator decides)

### Interrupted Rotation

If rotation command fails mid-apply:

- Raft re-applies on restart
- Idempotent: re-wrapping same DEK produces same result
- Or: explicit recovery state (e.g., "ROTATION_PENDING")

---

## Local CA Key Migration

### Status Quo

- `State.LocalCAKey` stored plaintext
- Already in Raft/BoltDB/snapshots/backups

### Migration Design (Explicit, Auditable)

Phase 1: **PLAINTEXT_LEGACY**
- Current state (no change)
- Audit: all commands that read/write LocalCAKey logged with "LEGACY" tag

Phase 2: **MIGRATION_PENDING** (operator initiates migration)
- Command: `{"type": "local-ca-migrate", ...}`
- New field: `State.LocalCAKeyWrapped` created
- Old field: `State.LocalCAKey` remains (for now)
- Marker: `State.LocalCAKeyMigrationState = "PENDING"`
- Audit: "starting LocalCA migration"

Phase 3: **ENCRYPTED** (migration completes)
- Command applies successfully
- `State.LocalCAKey` cleared to ""
- `State.LocalCAKeyWrapped` contains (wrappedDEK, nonce, ciphertext)
- `State.LocalCAKeyMigrationState = "ENCRYPTED"`
- Audit: "LocalCA migrated to encrypted storage"

Phase 4: **LEGACY_REMOVED** (after confidence interval, e.g., 1 week)
- Operator confirms: "LocalCA still accessible, no issues"
- Command: `{"type": "local-ca-legacy-cleanup", ...}`
- Raft entries tagged "LEGACY" can be excluded from new exports

Phase 5: **VERIFIED** (after full backup cycle)
- All snapshots taken after Phase 3 confirmed to NOT contain plaintext
- Audit ledger confirms no "LEGACY" access since Phase 4
- Evidence record documents successful migration

### Historical Media Sanitization

**Critical distinction:**

- **Current State confidentiality:** Phase 3 achieves it (plaintext removed)
- **Historical media sanitization:** Separate, not automatic

**Operator responsibility:**

- Snapshots created before Phase 3 may contain plaintext
- Backups created before Phase 3 may contain plaintext
- Raft logs may contain plaintext for days/weeks

**Policy:**
- Retain old snapshots/backups only as long as recovery is needed
- Explicit rotation: old snapshot → archive → overwrite/destroy
- Or: don't migrate to encrypted storage; accept plaintext-in-Raft risk

**Does NOT automatically sanitize:**
- Deleted Raft log entries (if compaction not enabled)
- Swapped memory (swap should be encrypted or disabled)
- Crash dumps
- Forensic copies

---

## Backup/Restore Model

### Backup Structure (No Change to Existing)

```json
{
  "format": "dh-export/v1",
  "cluster": "string",
  "state": { ... State with secrets ciphertext ... },
  "secrets": false
}
```

**Secrets field:** Always false for automatic backups. Plaintext secrets never included.

### Recovery Scenario 1: Restore to Same Cluster

1. Backup file contains ciphertext + wrapped DEK
2. Restore command loads State
3. KEK is present in cluster (in memory or wrapped in new State)
4. Secrets automatically decryptable
5. No additional recovery material needed

### Recovery Scenario 2: Restore to New Cluster

1. Backup file contains ciphertext + wrapped DEK (old KEK version)
2. Operator supplies backup + original Secrets Master Key
3. New cluster initializes with same Secrets Master Key
4. Old wrapped DEKs are decryptable with old KEK
5. Secrets accessible

**Requirement:** Operator retains Secrets Master Key backup offline.

### Recovery Scenario 3: Keys Lost

1. Backup file contains ciphertext + wrapped DEK (old KEK version)
2. KEK is not recoverable
3. Secrets are **permanently inaccessible**
4. Fail-closed: no attempt to downgrade or guess

**Operator action:** Re-issue secrets using new KEK. Workloads receive new versions.

### Operator Recovery Material

Operator must retain:

- Secrets Master Key (offline, HSM, or secure backup)
- Bootstrap secret material (to unwrap KEK)

If lost:
- Cluster remains accessible (no key material needed for existing ops)
- New secrets can be created (with new Master Key)
- Old secrets become inaccessible (permanent loss)

---

## Logging / Audit / Evidence

### Allowed in Audit Ledger

```json
{
  "ts": 1695123456000,
  "operation": "secret-retrieve",
  "secretId": "sec_...",
  "version": 1,
  "deploymentId": "dep_...",
  "workloadId": "w_...",
  "nodeId": "dh1...",
  "actor": "agent",
  "authorized": true,
  "reason": "nonce valid, workload assigned"
}
```

### Forbidden in Audit / Evidence / Logs

- Secret plaintext
- DEK plaintext
- KEK plaintext
- Secrets Master Key
- Token signatures (redact for privacy)
- Any decrypted value

### Canary Test

- Generate `canary_secret` = high-entropy random value
- Create secret with canary value
- Retrieve secret (valid authorization)
- Rotate secret (new version)
- Restart cluster
- Backup/export
- Restore
- Revoke canary secret

**Scan for canary plaintext:**
- Raft logs (dh-control data dir)
- Snapshots (dh-control data dir)
- Backups (export dir)
- PostgreSQL mirror
- dh-agent persistent data
- /tmp, /var/tmp
- Evidence records
- Logs (dh-control, dh-agent)

**Expected:** Zero plaintext occurrences outside authorized ephemeral delivery to `/run/secrets`.

---

## Threat Model

### Threats and Controls

| # | Threat | Asset | Attacker | Boundary | Control | Residual | Test |
|---|--------|-------|----------|----------|---------|----------|------|
| T1 | Stolen Raft/BoltDB file | Secrets | Disk thief | File on disk | AEAD + wrapped DEK | Ciphertext only (no plaintext) | Read stolen DB; verify no plaintext |
| T2 | Stolen snapshot | Secrets | Disk thief | Snapshot file | Same as T1 | Same | Extract snapshot; scan plaintext |
| T3 | Stolen backup | Secrets | Admin | Backup export | Same as T1 | Same | Steal backup; attempt decrypt without KEK |
| T4 | Compromised workload | Own secret | Malicious workload | Process/tmpfs | Only deliver to requesting workload | Workload can read its own secret (intended); cannot read others | Hostile fixture: compromised workload requests peer's secret; verify deny |
| T5 | Malicious workload on same node | Peer secret | Malicious workload | Node OS isolation | Authorization binding (workloadId) | Runtime isolates processes; OS must prevent tmpfs escape | Hostile fixture: workload breaks out of runtime; still cannot access `/run/secrets/{peer}` |
| T6 | Compromised node identity | All node secrets | Machine compromise | Ed25519 key security | Node key stored locally; operator revokes key | Attacker can impersonate node | Generate fake token with stolen identity; verify node revocation blocks retrieval |
| T7 | Revoked node replay | Old secret | Network attacker | Token TTL + nonce | Consumed nonce + revocation check | Consumed nonces stored briefly; old tokens cannot be replayed after revocation | Revoke node; attempt token replay; verify deny |
| T8 | Stolen retrieval token | Secret | Network attacker | TLS + request signature | Token has TTL (5min); request must be re-signed by node | Token alone useless without node private key | Steal token; attempt use without signing; verify deny |
| T9 | Malicious cluster member | All secrets | Insider | Raft membership + key material | KEK not held plaintext by members; only wrapped in State | Member can decrypt within authorized operations but cannot modify DEK directly | Member modifies AAD to change scope; verify AEAD integrity check fails |
| T10 | Compromised control-plane process | All secrets | Process compromise | dh-control isolation | Operator monitors; secrets in memory only during response | Process can decrypt and leak during operation | Honeypot secret; compromise process; scan /tmp for leaked plaintext |
| T11 | Operator credential compromise | Secrets Master Key | Credential theft | Key storage (offline/HSM) | Operator-dependent; model TBD at A03 | Depends on custody model | Depends on model |
| T12 | Disk theft (node) | Agent-side secrets | Disk thief | Tmpfs only (volatile) | No persistent storage | Tmpfs is lost on power loss or unmount | Power off node; try to recover secrets from disk; verify none found |
| T13 | Logs/evidence exfiltrated | Secrets | Log reader | Audit ledger + evidence format | Never log plaintext; canary test | Log only metadata (secretId, result, actor) | Exfiltrate logs; scan for canary plaintext; expect zero |
| T14 | Ciphertext modified | Ciphertext integrity | Network MITM | AEAD (GCM) | Authentication tag fails on corrupt ciphertext | Decryption fails; secret unavailable | Modify ciphertext in transit; verify decryption fails |
| T15 | Metadata/scope modified | Scope binding | Network MITM | AAD in AEAD | Modifying scope changes AAD; auth tag fails | Attempted scope escalation fails | Modify AAD (scope, deploymentId); verify AEAD fails |
| T16 | Rollback to old version | Version liveness | Attacker on Raft | Version supersession | Old version marked SUPERSEDED; metadata audit trail | Operator must choose: keep old versions or delete | Attempt to force old version; verify current version enforced |
| T17 | Interrupted rotation | Consistency | Operator error | Idempotent commands + tombstones | Re-apply rotation command; same result | Partial state OK; retry succeeds | Stop cluster mid-rotation; restart; verify state consistent |
| T18 | Key loss (KEK) | All secrets | Operator | Key backup | Operator retains Secrets Master Key offline | Secrets permanently inaccessible if KEK + backup lost | Delete KEK from all members; attempt secret retrieval; verify unavailable |
| T19 | Node reboot during secret use | Secret cleanup | Node crash | Tmpfs not persistent | tmpfs cleared on reboot; no secrets left on disk | Workload must request secret again after reboot | Reboot node; scan disk; verify no secrets in /run/secrets |
| T20 | Agent crash before cleanup | Secret cleanup | Process crash | Tmpfs ownership + TTL | OS cleans up /run/secrets on agent exit | Old /run/secrets entries manually cleaned by admin or TTL | Kill agent without cleanup; verify old entries remain until TTL or manual delete |

### Fail-Closed Rules

| Condition | Action |
|-----------|--------|
| KEK unavailable | Secrets unavailable; no fallback |
| Authorization invalid | Deny retrieval |
| Nonce consumed | Deny (one-use) |
| Token expired | Deny |
| Node revoked | Deny; all node requests rejected |
| Workload not assigned to node | Deny |
| Scope mismatch (environment, deploymentId) | Deny |
| AAD mismatch | Decryption fails (AEAD integrity) |
| Ciphertext corrupt | Decryption fails |
| Signature invalid | Request rejected |
| Secret in REVOKED state | Deny retrieval |

---

## Qualification Test Plan (SEC-P0-A01-A03 onwards)

### Unit Tests (Cryptography)

- [ ] AEAD encrypt/decrypt round-trip
- [ ] DEK wrapping/unwrapping with KEK
- [ ] AAD affects AEAD tag (modify AAD → tag fails)
- [ ] Nonce uniqueness (same plaintext + key + nonce → same ciphertext)
- [ ] Tag verification fails on tampered ciphertext

### Integration Tests (Persistence)

- [ ] Create secret → stored in Raft with wrapped DEK, ciphertext
- [ ] Raft log contains no plaintext
- [ ] Retrieve secret → plaintext never stored on disk
- [ ] Backup contains ciphertext + wrapped DEK, no plaintext
- [ ] Snapshot contains ciphertext only
- [ ] Export format excludes plaintext

### Authorization Tests

- [ ] Valid token + valid request signature → secret retrieved
- [ ] Invalid token → deny
- [ ] Expired token → deny
- [ ] Consumed nonce → deny on re-use
- [ ] Revoked node → deny
- [ ] Workload not assigned → deny
- [ ] Wrong scope (deploymentId mismatch) → deny
- [ ] Cross-workload retrieval → deny

### Rotation Tests

- [ ] Rotate secret → new version created
- [ ] Old version still accessible for rollback
- [ ] DEK rotation → ciphertext re-encrypted, plaintext unchanged
- [ ] KEK rotation → all wrapped DEKs re-wrapped
- [ ] Interrupted rotation → idempotent retry succeeds

### Restart/Recovery Tests

- [ ] Restart cluster → KEK recoverable, secrets accessible
- [ ] Restore from backup → secrets accessible with same KEK
- [ ] KEK lost → secrets unavailable (fail-closed)
- [ ] Nonce ledger persists across restart

### Delivery Tests

- [ ] Secret written to `/run/secrets` with correct permissions (0600)
- [ ] Secret owned by workload UID/GID
- [ ] Secret deleted on workload stop
- [ ] Secret deleted on agent stop
- [ ] Secret not persistent after reboot

### Hostile Fixture Tests (Negative Controls)

- [ ] Workload breaks out of runtime; cannot read `/run/secrets/{peer}`
- [ ] Compromised workload requests peer secret; verify authorization denies
- [ ] Modify ciphertext in Raft; verify AEAD fails on retrieval
- [ ] Modify AAD (scope); verify AEAD tag fails
- [ ] Replay old token after node revocation; verify deny
- [ ] Cross-cluster secret retrieval attempt; verify deny
- [ ] Disable nonce consumption check; test MUST FAIL (verify regression caught)
- [ ] Disable authorization verification; test MUST FAIL
- [ ] Disable revocation check; test MUST FAIL

### Leakage Tests (Canary)

- [ ] Create canary secret
- [ ] Retrieve, rotate, restart, backup, restore
- [ ] Scan logs, evidence, Raft, snapshots, backups, agent storage for plaintext canary
- [ ] Expected: zero occurrences outside `/run/secrets`

---

## Unresolved Items for A03+

1. **AEAD Algorithm Choice:** AES-256-GCM or ChaCha20-Poly1305? (crypto decision at A03)
2. **KEK Custody Implementation:** How exactly is wrapped KEK stored/unwrapped? (A03)
3. **Nonce Ledger Persistence:** In-memory bounded cache or Raft-backed? (A03)
4. **Local CA Migration Timeline:** When to execute Phases 2–5? (product decision)
5. **Secret Rotation Semantics:** Backward compat (keep old versions) or not? (product decision)
6. **Scope Granularity:** Just (deploymentId, workloadId, environment) or finer? (product decision)

---

## Summary

**Current Phase (A02):** Design only. No code.

**Next Phase (A03):** Cryptographic core (AEAD envelope, KEK wrapping, DEK derivation).

**A04:** Authorization + delivery mechanism.

**A05+:** Rotation, backup/restore, qualification.

**LocalCAKey Migration:** Separate decision path; can execute during A03–A05 with confidence.

---

## Related Documents

- ADR 0008: TLS Bootstrap Pinning (Cluster Identity Root model)
- docs/CURRENT_STATE.md: (existing system inventory)
- docs/qualification/ (evidence format and verification)
