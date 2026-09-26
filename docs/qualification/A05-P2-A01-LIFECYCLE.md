# A05-P2-A01: Secret Lifecycle & Recovery

**Status**: Architecture design and test matrix definition  
**Gate**: A05-P2-A01  
**Baseline**: Main branch after A05-P1-R1-A01 (6524ac0)  
**Scope**: Live revocation, version rotation, agent crash recovery, partition handling, host reboot reconciliation, replay durability

## Overview

A05-P1 established the delivery trust boundary: control plane signs, agent validates and materializes to local tmpfs.

A05-P2 extends this to cover secret lifecycle after materialization:
- **Revocation**: Emergency denial of secret access while workload is running
- **Rotation**: Replace old secret version with new without workload restart
- **Crash recovery**: Handle agent crashes before/during/after materialization
- **Partition handling**: Define authorized state during control-plane unavailability
- **Reboot reconciliation**: Restore valid secrets after host reboot
- **Replay durability**: Prevent restart from re-accepting old delivery envelopes

## Part 1: Live Revocation

### Semantic Model

A secret can be revoked while a workload holds an active materialization:

```
Secret ACTIVE
    ↓
revocation committed (Raft consensus)
    ↓
revocation propagated to agent
    ↓
agent marks materialization REVOKED
    ↓
secret mount unmounted from workload namespace
    ↓
workload future accesses DENIED
    ↓
audit/evidence records transition ACTIVE → REVOKED
```

### Critical Invariants

1. **No in-memory erasure**: Revocation cannot erase bytes the workload already copied into its own memory. This is beyond kernel enforcement.

2. **Managed materialization removal**: The enforceable guarantee is:
   - Managed secret file is removed from filesystem
   - Mount is removed from workload namespace
   - New delivery/retrieval requests are DENIED
   - Audit records show revocation timestamp and operator

3. **Auditable revocation**: Every revocation must record:
   - secretId, workload context, revocation timestamp
   - who authorized revocation
   - how many live materializations were affected
   - completion timestamp (mount unmounted, file cleaned)

### Implementation: RevocationManager

```go
// RevocationManager handles live secret revocation
type RevocationManager interface {
    RevokeSecret(ctx context.Context, secretId string) error
    IsRevoked(ctx context.Context, secretId string) bool
    RegisterMaterialization(hostPath string, secretId string) error
    UnregisterMaterialization(hostPath string) error
    RevocationObserver() RevocationObserver
}

// RevocationEvent records state transitions
type RevocationEvent struct {
    SecretID         string
    Timestamp        int64
    Status           string // "REVOKED", "UNMOUNTED", "CLEANED"
    AffectedMounts   []string
    AuthorizedBy     string
    CompletionTime   int64
}
```

### Revocation Flow

1. **Revocation request**: Operator calls control-plane revocation API
2. **Raft commit**: Revocation committed to cluster state
3. **Agent notification**: Agent learns revocation (polling or subscription)
4. **Mount removal**: SecretDeliveryReceiver unmounts from affected workload namespaces
5. **File cleanup**: ReleaseEphemeral is called for affected materializations
6. **Audit recording**: RevocationEvent recorded to evidence

## Part 2: Version Rotation

### Semantic Model

Rotation updates a secret from version 1 to version 2 without workload restart:

```
Secret v1 attached
      ↓
v2 authorized (Raft consensus)
      ↓
v2 delivery offered
      ↓
v2 materialized
      ↓
workload-visible mount switched to v2 path
      ↓
v1 mount removed
      ↓
v1 cleaned up
      ↓
audit records v1→v2 transition
```

### Critical Property: Transition Visibility

**Do NOT claim atomic replacement until tested.** Observable state during rotation:

```
Time T0:   v1 mounted at /secrets/secret-id/current
Time T1:   v2 materialized at /secrets/secret-id/v2
Time T2:   symlink /secrets/secret-id/current updated v1 → v2
Time T3:   v1 mount point removed
```

Between T1 and T2, workload may see:
- Old v1 via current symlink (valid)
- New v2 directly (also readable)

This is **not atomic**. Document observable window and define acceptable behavior:
- (a) Atomic switch required: test proves it
- (b) Brief window tolerated: application must handle short overlap
- (c) Read latest only: application must follow new path

### Implementation: SecretRotationManager

```go
type SecretRotationManager interface {
    RotateSecret(ctx context.Context, secretId string, newVersion int, newPayload []byte) error
    GetActiveVersion(ctx context.Context, secretId string) (int, error)
    RotationObserver() RotationObserver
}

type RotationEvent struct {
    SecretID         string
    FromVersion      int
    ToVersion        int
    Timestamp        int64
    Status           string // "V2_AUTHORIZED", "V2_MATERIALIZED", "SWITCHED", "CLEANED"
    V1MountPaths     []string
    V2MountPath      string
    CompletionTime   int64
    TransitionWindow time.Duration // Observable window if not atomic
}
```

### Rotation Constraints

- Old version must remain valid until new version is fully materialized
- Cannot start v2 materialization until v1 is still accessible (no double-failure window)
- Cleanup of v1 only after v2 is confirmed live in all workloads

## Part 3: Crash Recovery

### Agent Crash Scenarios

#### Scenario 1: Crash after authorization, before delivery

```
Control plane: authorization committed
Agent: hasn't received yet
Agent crashes
Agent restarts

Question: Is the authorization still valid?
Answer: YES - authorization is Raft state, survives all agent restarts
Question: Can the restarted agent delivery the secret?
Answer: YES - agent queries current authorization state, finds v1 active, delivers v1
Question: Is this replay (reusing old delivery envelope)?
Answer: NO - agent fetches fresh authorization on restart
```

#### Scenario 2: Crash during materialization (after delivery, before mount)

```
Control plane: signed envelope delivered
Agent: received envelope, validated it
Agent crashes mid-write to /secrets/secret-id/ephemeral
Agent restarts

Question: Is /secrets/secret-id/ephemeral partially written?
Answer: Likely - filesystem may have partial file
Question: Who cleans it up?
Answer: SecretMaterializer.Cleanup() (background goroutine, timeout-based)
Question: Can delivery be retried?
Answer: YES - old envelope is no longer accepted if nonce is tracked
```

#### Scenario 3: Crash after materialization, before state update

```
Control plane: signed envelope delivered
Agent: received, validated, materialized to /secrets/secret-id/ephemeral_xyz
Agent crashes before recording in memory that this materialization is active
Agent restarts

Question: Does the secret file still exist?
Answer: YES - tmpfs file persists across tmpfs mount lifetime (until workload exit or explicit cleanup)
Question: Does the agent know this materialization is active?
Answer: NO - this is the key challenge
Question: How does the workload access the secret?
Answer: Depends on how mounting was done - if static mount, it's still there; if dynamic, agent must remount
```

### Reconciliation on Restart

```go
// ReconciliationManager handles restart state reconciliation
type ReconciliationManager interface {
    // On agent restart, discover active materializations
    DiscoverActiveMaterializations(ctx context.Context) ([]string, error)
    
    // Determine which are still authorized
    ValidateMaterialization(ctx context.Context, hostPath string) error
    
    // Clean up orphaned/revoked materializations
    RemoveUnauthorizedMaterializations(ctx context.Context) error
    
    // Record reconciliation result to evidence
    RecordReconciliation(ctx context.Context, found, valid, removed int) error
}
```

## Part 4: Network Partition Semantics

### Critical: Fail Closed

**Absence of control-plane authorization information is NOT the same as current authorization.**

```
Cannot reach control plane
  ├─ Case A: Maybe revoked (unknown)
  │   Action: DENY (fail closed)
  │
  ├─ Case B: Maybe network partition, auth still valid
  │   Action: Need lease/validity window
  │
  └─ Case C: Maybe authorization removed from state
      Action: DENY (fail closed)
```

### Lease-Based Validity

Define explicit lease windows:

```go
type AuthorizationLease struct {
    SecretID      string
    Version       int
    IssuedAt      int64
    ExpiresAt     int64 // Lease window, not delivery expiry
    LeaseId       string
    
    // If partition occurs during lease validity:
    // - Agent may continue using secret
    // - Agent must validate upon reconnection
    // - If revoked during partition, revocation is discovered on reconnect
}
```

### Partition Failure Mode

1. **Authorization committed**: Agent receives authorization with lease validity window
2. **Network partition**: Agent loses connection to control plane
3. **During lease validity**: Agent may continue using secret (explicit policy)
4. **After lease expires**: Agent must reconnect or deny new retrievals
5. **On reconnection**: Agent validates current state:
   - If revoked during partition: revocation takes effect immediately
   - If rotated during partition: new version is now active
   - If still active: continues (may have different version)

## Part 5: Host Reboot

### Three Separate Questions

#### Question 1: Are tmpfs contents still there?

**Answer**: No - tmpfs is volatile. After reboot, /var/run/secrets is empty.

```
OBSERVED: secret files disappeared
```

#### Question 2: Is the authorization still valid?

**Answer**: Yes - authorization state is in Raft, survives node reboots.

```
SEPARATE: authorization remains active in control-plane state
```

#### Question 3: Can the workload rematerialize the secret?

**Answer**: Depends on policy. Requires **explicit re-authorization** or defined auto-rematerialization.

```
REQUIRES POLICY: 
- Option A: Workload requests secret on startup (new delivery envelope required)
- Option B: Auto-rematerialize if authorization is still active (needs test)
- Option C: Deny until explicit re-request (explicit lease/generation required)
```

**Do not infer re-materialization from authorization persistence.**

### Reboot Workflow

```
Host reboot
    ↓
tmpfs mount recreated (empty)
    ↓
Agent starts
    ↓
ReconciliationManager.DiscoverActiveMaterializations() → empty list
    ↓
Agent learns no materializations are active (correct state)
    ↓
Workload starts
    ↓
Workload requests secret
    ↓
New authorization query (control plane, may have rotated/revoked/expired)
    ↓
New delivery envelope generated
    ↓
New materialization created
    ↓
Workload mounts new path
```

## Part 6: Replay Durability

### Critical Vulnerability: Generation Tracking

If restarting the agent only empties an in-memory cache and allows an old signed delivery envelope to be accepted again, A05 is vulnerable to replay.

### Prevention: Generation-Based Revocation

```go
type DeliveryGeneration struct {
    SecretID      string
    Version       int
    GenerationId  string  // Changes on each rotation or revocation
    IssuedAt      int64
    CreatedBy     string  // Control plane identity
}

type DeliveryEnvelope struct {
    // ... existing fields from P1 ...
    GenerationId  string  // Cryptographically bound by signature
}

type GenerationStore interface {
    // Store current generation (survives agent restart via Raft or disk)
    SetCurrentGeneration(ctx context.Context, secretId string, gen DeliveryGeneration) error
    
    // Query current generation
    GetCurrentGeneration(ctx context.Context, secretId string) (DeliveryGeneration, error)
    
    // Track which generations have been consumed
    RecordConsumption(ctx context.Context, deliveryId string, genId string) error
    IsConsumed(ctx context.Context, genId string) bool
}
```

### Replay Prevention Flow

```
Envelope arrives with GenerationId G1
    ↓
Check: Is G1 current?
    ├─ If NO (revoked/rotated): DENY
    └─ If YES: Continue to signature verification
    ↓
Signature valid
    ↓
Check: Has this delivery (by nonce) been materialized before?
    ├─ If YES: DENY (exact duplicate)
    └─ If NO: Proceed
    ↓
Materialize
    ↓
Record: GenerationId G1, nonce, delivery timestamp
    ↓
Agent restarts
    ↓
GenerationStore persists: current gen is G2 (rotated after G1)
    ↓
Old envelope with G1 arrives (somehow, after restart)
    ↓
Check: Is G1 == current?
    ├─ NO: DENY "generation superseded"
```

### Generation Storage: Raft vs. Local Disk

**Choice 1: Raft-backed** (all nodes have current generation)
- Pros: Survives full cluster failure, visible to control plane
- Cons: Generation change is consensus operation, latency

**Choice 2: Local disk** (agent-specific, survives agent restart only)
- Pros: Fast, no consensus needed
- Cons: Cannot prevent replay if node is cloned/forensically restored

**For P2**: Implement local disk + Raft advisory state. Agent trusts local disk; control plane records generations in Raft for audit.

## Plaintext Canary for Lifecycle

### Surfaces Scanned (extending P1)

P1 verified: control plane, BoltDB, logs, Raft, filesystem (excluding tmpfs)

P2 adds:

1. **Revocation audit trail**: No plaintext in revocation records
2. **Rotation audit trail**: No plaintext in v1→v2 records
3. **Crash dumps**: No plaintext in /var/crash, /var/lib/systemd/coredump
4. **Reboot evidence**: No plaintext in /var/log post-reboot
5. **Orphan cleanup logs**: No plaintext in orphan discovery/cleanup
6. **Reconciliation state**: No plaintext in restart reconciliation records
7. **Partition recovery**: No plaintext in partition-detected logs

### Canary Test Suite

```
TestPlaintextCanary_RevocationRecordsNoPayload
TestPlaintextCanary_RotationRecordsNoPayload
TestPlaintextCanary_CrashDumpNoPayload
TestPlaintextCanary_RebootLogsNoPayload
TestPlaintextCanary_ReconciliationNoPayload
```

## Adversarial Test Matrix (A05-P2-A01)

### Revocation Tests

- `TestRevokeRunningSecret`: Revoke while workload is accessing secret → mount removed, new accesses denied
- `TestRetrievalAfterRevocationDenied`: Revocation committed → retrieval API returns "REVOKED"
- `TestRevokeDuringDelivery`: Revocation committed while envelope in flight → agent receives revocation before materialization
- `TestRevokeDuringMaterialization`: Revocation while file is being written → partial file cleaned up

### Rotation Tests

- `TestRotateV1ToV2`: Live rotation with running workload → v2 becomes active
- `TestOldVersionDeniedAfterRotation`: Old delivery envelopes with v1 GenerationId rejected
- `TestRotationFailurePreservesDefinedState`: Failed v2 materialization leaves v1 intact and usable
- `TestConcurrentRotationAndRetrieval`: Simultaneous rotation and retrieval requests → defined outcome

### Crash Recovery Tests

- `TestAgentCrashAfterReceiveBeforeMaterialize`: Crash after delivery received → orphan cleanup removes partial file
- `TestAgentCrashDuringMaterialize`: Crash mid-write → file is incomplete/corrupted, cleaned up
- `TestAgentCrashAfterMaterializeBeforeStateUpdate`: Crash after file written → restart finds file, validates authorization, remounts
- `TestAgentRestartReconcilesActiveMount`: Restarted agent discovers active mount, validates, records reconciliation
- `TestAgentRestartRemovesOrphanMount`: Restarted agent finds revoked mount, removes it, records cleanup

### Partition Tests

- `TestControlPlaneUnavailableWithExistingSecret`: Partition occurs → agent uses current generation until lease expires
- `TestPartitionBeforeAuthorization`: Partition before authorization query → DENY (no lease)
- `TestPartitionAfterAuthorizationBeforeDelivery`: Authorization received, then partition, then delivery request → use cached authorization within lease window
- `TestPartitionAfterMaterialization`: Secret materialized, then partition → workload can still access (tmpfs)
- `TestReconnectReconciliation`: On reconnection after partition, agent validates current state against control plane

### Reboot Tests

- `TestHostReboot`: Full reboot → tmpfs is empty, auth state in Raft is intact, fresh authorization required
- `TestRebootRequiresFreshAuthorizationWhereRequired`: Workload cannot use old path after reboot without new delivery

### Replay Durability Tests

- `TestDuplicateDeliveryAfterRestart`: Same envelope (same nonce, same generationId) delivered twice across restart → second delivery DENIED
- `TestStaleGenerationRejected`: Envelope with old GenerationId (superseded by rotation) → DENIED
- `TestReplayAfterAgentRestartDenied`: Delivery envelope replayed after agent restart using same nonce → DENIED

### Plaintext Canary Tests (Lifecycle)

- `TestPlaintextCanary_RevocationRecordsNoPayload`
- `TestPlaintextCanary_RotationRecordsNoPayload`
- `TestPlaintextCanary_CrashDumpNoPayload`
- `TestPlaintextCanary_RebootLogsNoPayload`
- `TestPlaintextCanary_ReconciliationNoPayload`

### Negative Controls

- `TestNegativeControl_DisableGenerationTracking_ReplayAccepted`: Without generation tracking, old envelopes accepted (proves necessity)
- `TestNegativeControl_SkipNonceTracking_DuplicateAccepted`: Without nonce tracking, exact duplicate accepted (proves necessity)
- `TestNegativeControl_IgnoreLeaseExpiry_ExpiredAuthUsed`: Without lease validation, expired auth used (proves necessity)
- `TestNegativeControl_NoPartitionFailClosed_AuthAssumedValid`: Without fail-closed semantics, partition results in continued use (proves necessity)

## Acceptance Criteria: A05-P2-A01 PASS/VERIFIED

```
Revocation               PASS (4 tests)
Rotation                PASS (4 tests)
Concurrent lifecycle    PASS (1 test)
Agent crash recovery    PASS (5 tests)
Agent restart reconciliation (implicit in crash tests)
Orphan cleanup          PASS (1 test)
Partition semantics     PASS (5 tests)
Reconnect reconciliation PASS (1 test)
Host reboot             PASS (2 tests)
Replay across restart   DENIED (3 tests)
Stale generation        DENIED (1 test)
Plaintext persistence   0 on SCANNED surfaces (5 tests)
Negative controls       PASS (4 tests)
Race detector           PASS (clean run)
Regression              PASS/reconciled (all A05-P1 tests still pass)

Total: 39 new tests + regression suite
```

## Evidence Collection

Results stored in: `evidence/A05-P2-A01-<timestamp>/`

- `record.json`: Execution timeline, baseline, test summary
- `environment.json`: Runtime environment facts
- `qualification.md`: This document (rendered)
- `tests.json`: All 39 test execution log and results
- `lifecycle-results.json`: Revocation, rotation, crash recovery test details
- `partition-results.json`: Partition semantics and reconnection validation
- `reboot-results.json`: Reboot reconciliation test details
- `replay-durability.json`: Generation and nonce tracking verification
- `canary-results.json`: Plaintext persistence across lifecycle
- `negative-controls.json`: Proof that each check is necessary
- `race-detector.json`: Race detector run results
- `signature`: SHA256-HMAC of record.json

## Next Phase

After A05-P2-A01 passes:

**A05-P0-A01**: Final integrated qualification combining P1 (delivery/isolation) and P2 (lifecycle/recovery) in a fresh environment with full adversarial campaign, plaintext canary, and immutable evidence seal.

Then: **LIFECYCLE-P0-A01** (node/workload lifecycle coordination)
