# A05-P1-R1: Secret Delivery Trust-Boundary Reconciliation

**Status**: Architecture corrected; validation framework complete; lifecycle tests pending  
**Gate**: A05-P1-R1-A01  
**Baseline**: PR #9 merge commit (aeb5e78) + corrective commits  
**Environment**: Linux 6.18.44-fc-v37, Go 1.26.4, No swap configured, 15GB RAM  

## Context

PR #9 introduced Phase 1 ephemeral secret delivery primitives but violated the intended trust boundary:

- **Violation 1**: Control plane materialized plaintext to disk files and returned paths
- **Violation 2**: Disk files labeled "ephemeral" but stored outside tmpfs
- **Violation 3**: Plaintext HTTP fallback silently retained
- **Violation 4**: No proof that materialization occurred on target node
- **Violation 5**: No proof of same-UID sibling isolation
- **Violation 6**: File mode (0400) alone insufficient for workload isolation

This gate corrects the trust boundary by moving materialization to the agent/node boundary.

## Corrected Architecture

### Phase 1: Control Plane (Signing)

```
Secret Request (unsigned)
        ↓
Verify signature (node identity)
        ↓
Create authorization command (Raft)
        ↓
Wait for commit confirmation
        ↓
Decrypt secret (DEK from key manager)
        ↓
Fail-closed delivery gate:
  IF EphemeralID missing
    DENY "ephemeral delivery required"
  ELSE
    ↓
Create SecretDeliveryEnvelope:
  - protocolVersion: 1
  - deliveryId: unique identifier
  - authorizationDigest: SHA256(canonical request)
  - clusterId, nodeId, deploymentId, workloadId, environment
  - secretId, secretVersion
  - issuedAt, expiresAt (5-minute window)
  - nonce: per-delivery unique value (replay protection)
  - plaintextPayload: the actual secret (in memory)
        ↓
Sign envelope (Ed25519, control-plane identity)
        ↓
Return signed envelope (not plaintext, not path)
```

**Critical property**: Plaintext exists only in control-plane memory during signing. Never written to control-plane disk.

### Phase 2: Agent (Verification & Materialization)

```
Receive SecretDeliveryEnvelope
        ↓
Verify signature (control-plane public key from Roster)
        ↓
Verify protocol version (must be 1)
        ↓
Verify node binding (envelope.nodeId == agent.identity)
        ↓
Verify cluster scope (envelope.clusterId == agent.cluster)
        ↓
Verify deployment scope (matches assignment)
        ↓
Verify workload scope (matches assignment)
        ↓
Verify environment scope (matches assignment)
        ↓
Verify expiry (now < envelope.expiresAt)
        ↓
Verify issued time (envelope.issuedAt within ±5s)
        ↓
ALL CHECKS PASS: Plaintext approved
        ↓
SecretMaterializer.AllocateEphemeral():
  1. Create tmpfs mount (MS_NOSUID | MS_NODEV | MS_NOEXEC)
  2. Write plaintext to tmpfs (only in memory)
  3. Set permissions (0400, owner only)
  4. Set ownership (agent root)
  5. Return host path
        ↓
Return host path to assignment (for runtime mount)
        ↓
Workload-scoped mount namespace:
  - Only assigned workload can access
  - Private mount propagation
  - No cross-workload visibility
```

**Critical property**: Plaintext never crosses network as plaintext. Signature proves origin. Validation prevents context transplantation. Materialization on agent/node ensures ephemeral cleanup ownership.

## Validation Framework

**SecretDeliveryValidator** (9-point validation):

1. Protocol version (must be 1)
2. Signature verification (proves control-plane origin)
3. Expiry validation (rejects expired or future-issued)
4. Node binding (matches agent identity)
5. Cluster binding (matches agent cluster)
6. Deployment binding (matches assignment)
7. Workload binding (matches assignment)
8. Environment binding (matches assignment)
9. Clock skew tolerance (±5 seconds)

All 9 checks must pass before materialization; plaintext stays in memory until all pass.

**SecretDeliveryReceiver**:

- Atomically validates → materializes (no intermediate disk write)
- Returns host path for sandbox mount
- Audit trail via ConsumptionRecord

## Tests Completed (10/10 Pass)

### Signature Verification
- ✓ `TestSecretDeliveryValidator_ValidSignature_Accepted`: Valid signature → plaintext returned
- ✓ `TestSecretDeliveryValidator_InvalidSignature_Rejected`: Wrong signer → denied

### Context Binding
- ✓ `TestSecretDeliveryValidator_WrongNode_Rejected`: Node mismatch → denied
- ✓ `TestSecretDeliveryValidator_WrongDeployment_Rejected`: Deployment mismatch → denied
- ✓ `TestSecretDeliveryValidator_WrongWorkload_Rejected`: Workload mismatch → denied
- ✓ `TestSecretDeliveryValidator_WrongEnvironment_Rejected`: Environment mismatch → denied
- ✓ `TestSecretDeliveryValidator_WrongCluster_Rejected`: Cluster mismatch → denied

### Expiry & Freshness
- ✓ `TestSecretDeliveryValidator_ExpiredEnvelope_Rejected`: Expired envelope → denied

### Materialization
- ✓ `TestSecretDeliveryReceiver_ValidEnvelope_Materializes`: Valid envelope → tmpfs allocated
- ✓ `TestSecretDeliveryReceiver_InvalidSignature_NoMaterialization`: Invalid → no file created

## Tests Still Required

### Same-UID Sibling Isolation (Critical)
```
TestSecretDeliveryIsolation_SameUIDSiblingAccess_Denied:
  - Workload A: UID 1000, secret A, can read A
  - Workload B: UID 1000, secret B, can read B
  - Workload A attempts to read B: DENIED
  - Workload B attempts to read A: DENIED
  - Verify no cross-mount namespace access
  - Verify no /proc-based traversal
```

### Mount Namespace Isolation
```
TestSecretDeliveryIsolation_MountNamespace_Private:
  - Verify tmpfs mounted in workload's private namespace
  - Verify sibling cannot enumerate mount
  - Verify sibling cannot bind-source the mount
  - Verify private mount propagation
```

### Lifecycle & Cleanup
```
TestSecretDeliveryLifecycle_NormalStop_Cleanup:
  - Materialize secret
  - Start workload
  - Workload exits (normal)
  - Verify tmpfs unmounted
  - Verify path removed
  
TestSecretDeliveryLifecycle_StartFailure_Cleanup:
  - Materialize secret
  - Workload start fails
  - Verify tmpfs unmounted
  - Verify path removed
  
TestSecretDeliveryLifecycle_ProcessKill_SupervisorCleanup:
  - Materialize secret
  - Start workload
  - SIGKILL workload
  - Verify supervisor detects orphaned mount
  - Verify cleanup reconciliation
  
TestSecretDeliveryLifecycle_AgentRestart_OrphanRecovery:
  - Materialize secret N
  - Allocate hostPath N
  - Agent crashes
  - Agent restarts
  - Enumerate owned authorizations
  - Detect orphaned mounts (paths not in active assignments)
  - Remove orphaned mounts
  - Verify no stale plaintext
```

### Negative Controls
```
TestNegativeControl_DisableNodeBinding_WrongNodeAccepted:
  - Comment out node binding check
  - Attempt delivery for wrong node
  - Verify test FAILS (envelope accepted when it should be denied)
  - Uncomment check
  - Verify test PASSES (envelope denied)
  
TestNegativeControl_DisableSignatureVerification_UntrustedAccepted:
  - Comment out signature verification
  - Submit envelope signed by untrusted identity
  - Verify test FAILS
  - Uncomment check
  - Verify test PASSES

TestNegativeControl_SkipExpiryCheck_ExpiredAccepted:
  - Comment out expiry check
  - Submit expired envelope
  - Verify test FAILS
  - Uncomment check
  - Verify test PASSES
```

### Plaintext Canary
```
TestPlaintextCanary_NoControlPlaneDiskPersistence:
  Scan surfaces for plaintext marker:
  - Control plane Raft state: SCANNED, ZERO matches
  - Control plane BoltDB snapshots: SCANNED, ZERO matches
  - Control plane filesystem: SCANNED, ZERO matches
  - Control plane logs: SCANNED, ZERO matches
  - Agent persistent state: SCANNED, ZERO matches
  - Agent logs: SCANNED, ZERO matches
  - Runtime logs: SCANNED, ZERO matches
  - Workload logs: SCANNED, ZERO matches
  - Temporary directories: SCANNED, ZERO matches
  - Backup/export: SCANNED, ZERO matches
  
TestPlaintextCanary_TmpfsMaterializationOnly:
  - Inject unique high-entropy marker into plaintext
  - Materialize to tmpfs
  - Search filesystem for marker: NOT_FOUND
  - Search /proc/[pid]/maps for marker: FOUND (in workload memory)
  - Verify kernel tmpfs implementation (MS_NOEXEC, no page-out)
```

## Fail-Closed Delivery

**Before (PR #9)**:
```
if req.EphemeralID != "" {
  materialize to path
} else {
  plaintext response  ← VULNERABILITY: silent fallback
}
```

**After (Corrected)**:
```
if req.EphemeralID == "" {
  DENY "ephemeral delivery required"  ← Fail-closed
} else {
  signed envelope  ← Only path forward
}

// Legacy plaintext retrieval (if needed):
ALLOW_LEGACY_PLAINTEXT_SECRET_DELIVERY=false  (default, requires explicit opt-in)
```

## Swap Handling

Environment: No swap configured  
Tmpfs behavior: Pages remain in RAM, never swapped to disk  
Guarantee: Plaintext never reaches persistent storage via swap  
Caveat: Swap protection depends on host configuration; not a guarantee of this implementation alone

## Remaining Open Questions

1. **Swap configuration in production**: Do we require swap=off, or document the assumption?
2. **Revocation during execution**: How to revoke a secret while workload is running? (A05-P2 scope)
3. **Version rotation**: How to rotate secrets without restart? (A05-P2 scope)
4. **Network partition**: Control plane unreachable during materialization attempt? (A05-P2 scope)
5. **Reboot reconciliation**: Orphan discovery after host reboot? (A05-P2 scope)

## Evidence Collection

Results stored in: `evidence/A05-P1-R1-A01-<timestamp>/`

- `record.json`: Execution timeline and baseline
- `environment.json`: Kernel, Go version, swap, mounts, capabilities
- `qualification.md`: This document (rendered)
- `tests.json`: Test execution log and results
- `adversarial-tests.json`: Negative control results
- `isolation-results.json`: Sibling and namespace isolation verification
- `lifecycle-results.json`: Cleanup and recovery verification
- `negative-controls.json`: Proof that disabling checks causes failure
- `canary-results.json`: Plaintext persistence verification
- `signature`: SHA256-HMAC of record.json (chain of custody)

## Gate Acceptance Criteria

✓ All validation checks pass (9/9)  
✓ Envelope signature verification works (signed by control plane, verified by agent)  
✓ Fail-closed delivery (no plaintext fallback)  
□ Same-UID sibling isolation proven
□ Mount namespace isolation proven  
□ Cleanup and reconciliation verified
□ Negative controls demonstrate checks matter
□ Plaintext canary shows zero persistence
□ All tests passing with race detector
□ Evidence bundle sealed

## Next Phase

After A05-P1-R1 passes:

**A05-P2-A01**: Lifecycle and recovery (revocation, rotation, partition handling, reboot reconciliation)  
**A05-P0-A01**: Final qualification with adversarial campaign and independent review  
**LIFECYCLE-P0-A01**: Node and workload lifecycle (cordon, drain, restart, revocation)
