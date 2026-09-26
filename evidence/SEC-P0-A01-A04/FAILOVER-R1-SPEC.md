# SEC-P0-A01-A04-FAILOVER-R1 Qualification Specification

**Revision:** 1 (Corrected)  
**Classification:** SEC-P0-A01-A04-FAILOVER-R1  
**Status:** SPECIFICATION (Not Yet Executed)  
**Date:** 2026-09-26  
**Scope:** Distributed leader failover verification for retrieval authorization  

## Qualification Gate

This specification gates A05 unlock pending successful demonstration of four distributed-systems properties:

1. Consumption state survives leader failover (replication through quorum commit)
2. Ambiguous responses do not cause double-release (at-most-once disclosure)
3. Pre-commit leader loss produces no unauthorized access (atomicity before quorum)
4. Cross-member concurrency yields exactly-one consumption (serialization during transition)

## Test Topology

**3-Member Raft Cluster**
- Member A: Initial leader (gossip port 50001, raft port 50101)
- Member B: Follower (gossip port 50002, raft port 50102)
- Member C: Follower (gossip port 50003, raft port 50103)

**Failure Injection Points**
- Network partition (isolate A from B+C, force quorum re-election)
- Process termination (kill A, observe state transfer)
- Process restart (restart A, verify convergence)
- Deterministic log replication control (block entries before quorum)
- Concurrent requests during transition (send identical to A, B, C simultaneously)

## Test Instrumentation

Each scenario requires test-only instrumentation counters:

```
authorizationCommitted     — count of consumptions durable in quorum
decryptInvocations         — count of decrypt calls attempted
plaintextReleaseAttempts   — count of plaintext bytes released to client
responseWrites             — count of successful response sends
```

These counters enable precise verification of fault boundaries and safety invariants.

## Test Scenarios

### Scenario R1-01: Leader Failover with Consumption Persistence

**Invariant:** Consumption committed by quorum before success → new leader rejects replay

**Sequence**

```
A = leader, term T1
Send R1 to A
    ↓
A proposes R1 to Raft
    ↓
R1 replicated to B, C (quorum achieved)
    ↓
authorizationCommitted == 1
    ↓
A returns SUCCESS to client
    ↓
FAULT: Isolate A from B+C (network partition)
    ↓
B or C elected leader (term T2, quorum confirms)
    ↓
Send identical R1 to new leader
    ↓
EXPECTED: DENIED_ALREADY_CONSUMED
    ↓
Verify R1 digest in new leader's replay ledger
    ↓
Verify audit trail: one SUCCESS, one DENIED
```

**Success Criteria**

- New leader's replay ledger contains R1 (proves quorum commitment)
- Duplicate request correctly rejected with specific error code
- No second plaintext release attempted
- No decryption invocation on second request
- Raft log shows R1 reached quorum before A isolation

**Evidence Capture (R1-01)**

```
clusterId
memberA_id, memberB_id, memberC_id
initialLeader = A
initialTerm = T1
replacementLeader ∈ {B, C}
replacementTerm = T2

requestId
requestDigest
nonceDigest

proposalLogIndex
commitIndexBeforeFault
faultInjectionPoint = "after commit, before response"
faultStart = timestamp
faultEnd = timestamp (A rejoins, if applicable)

authorizationCommitted = 1
decryptInvocations = 0 (on second request)
plaintextReleaseAttempts = 0 (on second request)
clientSuccessfulReceipt = 1 (first), 0 (second)

replayLedgerA_postRestart
replayLedgerB_postElection
replayLedgerC_postElection
auditOutcomeA
auditOutcomeB
auditOutcomeC

postHealConvergenceIndex
```

---

### Scenario R1-02: Ambiguous Response Boundary

**Invariant:** Consumption committed → plaintext at-most-once disclosed → even if response lost, retry is denied

**Sequence**

```
A = leader, term T1
Send R2 to A
    ↓
A proposes R2 to Raft
    ↓
R2 replicated to B, C (quorum achieved)
    ↓
Raft Apply(R2) returns SUCCESS
    ↓
authorizationCommitted == 1
    ↓
FAULT POINT: Before DecryptSecret() call, before response write
    ↓
Kill A (process termination, not graceful)
    ↓
No response sent to client (ambiguous)
    ↓
Client detects timeout, retries R2
    ↓
B or C elected leader (term T2)
    ↓
New leader receives retry of R2
    ↓
EXPECTED: DENIED_ALREADY_CONSUMED
    ↓
Verify:
  - decryptInvocations == 1 (first request only)
  - plaintextReleaseAttempts == 1 (first request only)
  - clientSuccessfulReceipt[first] = 0 (killed before send)
  - clientSuccessfulReceipt[retry] = 0 (denied)
```

**Success Criteria**

- At-most-once disclosure: plaintext decrypted at most once
- Plaintext released at most once
- Retry correctly denied without second decryption
- Audit shows one SUCCESS (internally), one DENIED (on retry)
- Convergence confirms R2 applied by quorum, never double-committed

**Evidence Capture (R1-02)**

```
clusterId
memberA_id, memberB_id, memberC_id
initialLeader = A
initialTerm = T1
replacementLeader ∈ {B, C}
replacementTerm = T2

requestId
requestDigest
nonceDigest

proposalLogIndex
commitIndexBeforeFault

faultInjectionPoint = "after Raft Apply, before DecryptSecret"
faultStart = timestamp
faultEnd = timestamp (when new leader elected)

authorizationCommitted = 1
decryptInvocations = 1
plaintextReleaseAttempts = 1
clientSuccessfulReceipt[initial] = 0
clientSuccessfulReceipt[retry] = 0

replayLedgerA (from restart)
replayLedgerB (from election)
replayLedgerC (from quorum)
auditOutcomeA
auditOutcomeB
auditOutcomeC

retryDetectedAsReplay = true
```

---

### Scenario R1-03: Pre-Commit Leader Loss

**Invariant:** Entry not yet committed (not at quorum) → leader dies → entry never becomes committed → no consumption, no decryption

**Sequence**

```
A = leader, term T1
Send R3 to A
    ↓
A validates R3 locally (signature, scope, etc.)
    ↓
A proposes R3 to Raft (sends to B, C)
    ↓
DETERMINISTIC FAULT: Block replication before quorum
    ↓
Prove: commitIndex < R3.logIndex on all members
    ↓
FAULT: A loses leadership (partition or terminate)
    ↓
B or C elected leader (term T2)
    ↓
Proof: R3 entry exists in log, but commitIndex never reaches R3.logIndex
    ↓
Wait for log reconciliation under new term
    ↓
Verify: commitIndex still < R3.logIndex
    ↓
Proof:
  - replayLedger lacks R3 on all members
  - authorizationCommitted == 0
  - decryptInvocations == 0
  - plaintextReleaseAttempts == 0
```

**Success Criteria**

- Raft entry R3 proposed but never committed (quorum proof)
- No consumption recorded in any replay ledger
- No decryption attempt made
- No plaintext released
- Commit index examination proves entry below high-water mark
- Post-heal convergence: R3 still not applied

**Evidence Capture (R1-03)**

```
clusterId
memberA_id, memberB_id, memberC_id
initialLeader = A
initialTerm = T1
replacementLeader ∈ {B, C}
replacementTerm = T2

requestId
requestDigest
nonceDigest

proposalLogIndex
commitIndexBeforeFault (must be < proposalLogIndex)
commitIndexAfterFault (must remain < proposalLogIndex)

faultInjectionPoint = "before quorum commit"
faultStart = timestamp
faultEnd = timestamp (new leader elected, attempted apply)

authorizationCommitted = 0
decryptInvocations = 0
plaintextReleaseAttempts = 0
clientSuccessfulReceipt = 0

replayLedgerA (pre-partition, unchanged post-restart)
replayLedgerB (unchanged, no application)
replayLedgerC (unchanged, no application)
auditOutcomeA = none
auditOutcomeB = none
auditOutcomeC = none

raftLogStateA_proposal = present
raftLogStateB_proposal = absent or present but not applied
raftLogStateC_proposal = absent or present but not applied

postHealCommitIndex (still < proposalLogIndex)
finalAppliedIndex (R3 never reached)
```

---

### Scenario R1-04: Cross-Member Concurrency During Failover

**Invariant:** Identical requests sent to multiple members during leadership transition → exactly one globally committed

**Sequence**

```
A = leader, term T1, logs synchronized
B, C = followers
Send R4 to A (queued, not yet applied)
    ↓
FAULT: Isolate A from B+C (A alone, quorum on B+C side)
    ↓
Simultaneously send identical R4 to B and C
    ↓
Capture observed term and leader state at each member
    ↓
A (alone) may process R4 locally but has no quorum
    ↓
B or C elected leader (term T2, quorum confirms)
    ↓
Quorum-side leader applies R4 via Raft
    ↓
authorizationCommitted == 1 (exactly once)
    ↓
Collect responses from all members:
  - Quorum leader: SUCCESS
  - Non-leader followers: varies (may timeout or DENIED_ALREADY_CONSUMED after convergence)
  - Isolated A: varies (processes independently until reconnection)
    ↓
FAULT END: Heal partition (A rejoins cluster)
    ↓
A converges with quorum
    ↓
Post-heal: all members agree
  - R4 is in replay ledger (committed once)
  - R4.digest appears exactly once
  - authorizationCommitted == 1 (still)
```

**Success Criteria**

- Exactly one authorization committed globally (quorum side only)
- Responses vary but post-heal convergence shows single consumption
- No race condition in digest collision or double-application
- Isolated member's local state does not override quorum state
- Race detector: 0 data races during concurrent applies

**Evidence Capture (R1-04)**

```
clusterId
memberA_id, memberB_id, memberC_id
initialLeader = A
initialTerm = T1
replacementLeader ∈ {B, C}
replacementTerm = T2
isolatedMember = A
quorumMembers = [B, C]

requestId
requestDigest
nonceDigest

faultInjectionPoint = "partition A from quorum"
faultStart = timestamp
faultEnd = timestamp (partition healed)

observedTermA_during_isolation
observedTermB_during_isolation
observedTermC_during_isolation
observedLeaderA
observedLeaderB
observedLeaderC

proposalLogIndexB_or_C (whichever is elected leader)
commitIndexB_or_C

authorizationCommitted = 1 (exactly)
decryptInvocations = 1
plaintextReleaseAttempts = 1
clientSuccessfulReceipt[A] = (unknown, isolated)
clientSuccessfulReceipt[B_or_C_leader] = 1
clientSuccessfulReceipt[B_or_C_follower] = 0

replayLedgerA_during_isolation
replayLedgerB_during_quorum
replayLedgerC_during_quorum
replayLedgerA_postHeal
replayLedgerB_postHeal
replayLedgerC_postHeal

digestCountA_post_heal = 1
digestCountB_post_heal = 1
digestCountC_post_heal = 1

postHealConvergenceIndex
raceDetectorResult = "0 races"
```

---

### Scenario R1-05: Old Leader Rejoins

**Invariant:** Partitioned leader rejoins after new leader elected → converges without replaying authorization

**Sequence**

```
Prerequisite: Complete Scenario R1-01 or R1-04 (leader was isolated)
    ↓
A was leader in term T1, now isolated
B or C elected leader in term T2
R5 committed by new leader (quorum)
    ↓
R5 in replayLedgerB and replayLedgerC
R5 not yet applied to A (was isolated)
    ↓
FAULT END: Heal partition (reconnect A to cluster)
    ↓
A observes term T2, recognizes new leader
    ↓
A snapshots local FSM
    ↓
A receives snapshot from new leader
    ↓
A restores from snapshot (now has R5)
    ↓
Retry R5 to any member
    ↓
EXPECTED: DENIED_ALREADY_CONSUMED (across all members)
    ↓
Verify: replayLedgerA_post_heal == replayLedgerB == replayLedgerC
```

**Success Criteria**

- Snapshot/restore correctly transfers consumed authorization
- No double-application or race during reconciliation
- Replay ledger converges to identical state
- Retry correctly rejected after heal
- No audit duplication for R5 (single consumption record, not per member)

**Evidence Capture (R1-05)**

```
clusterId
oldLeaderA_term = T1
newLeader ∈ {B, C}
newLeader_term = T2

requestId (R5 from Scenario R1-01 or R1-04)
requestDigest

snapshotIndexA_pre_heal
snapshotTimestampA
snapshotMetadataA

replayLedgerA_pre_heal (empty or partial)
replayLedgerB (authoritative)
replayLedgerC (authoritative)
replayLedgerA_post_restore

convergenceProof:
  hash(replayLedgerA) == hash(replayLedgerB) == hash(replayLedgerC)

retryAfterHeal = DENIED_ALREADY_CONSUMED
auditConsistency = single entry per request, not per member
```

---

## Simplified Test Matrix

| Scenario | Fault Boundary | Required Proof | A05 Gate |
|----------|---|---|---|
| **R1-01** | After commit, before response | New leader rejects replay | ✓ Replication |
| **R1-02** | After commit, before decrypt/response | At-most-once disclosure | ✓ Idempotence |
| **R1-03** | Before quorum commit | Zero consumption, zero decryption | ✓ Atomicity |
| **R1-04** | Requests across partition/election | Exactly one globally | ✓ Serialization |
| **R1-05** | Old leader rejoins after heal | Convergence, no replay | ✓ Recovery |

---

## Regression Test Suite (Post-Failover)

**A03 Baseline Tests**
- All A03 tests must pass unchanged (no functionality break)

**A04 Baseline Tests**
- All original A04 local tests must pass (snapshot/restore persistence)
- High-contention test (50 goroutines) must pass
- **NEW:** Timestamp nanoseconds/milliseconds boundary regression test
  - Captures the fix from prior session (canonical timestamp overflow)
  - Ensures JSON marshaling handles large timestamps as strings

**Race Detector Validation**
- All tests (A03, A04, failover R1-01 through R1-05) must pass with `go test -race`
- 0 data races permitted

**Canary Tests**
- Positive path (valid request, new leader, success)
- Negative path (revoked node, new leader, denial)
- Scope enforcement across members
- Signature verification across replication

---

## Evidence Capture and Verification

**Mandatory Evidence Artifacts**

1. **failover-test-log.jsonl** — Step-by-step execution (timestamp, action, fault point, result)
2. **cluster-topology.json** — Member IDs, ports, initial roles
3. **raft-logs-member-*.txt** — Log entries for each scenario, each member
4. **fsm-state-snapshots-*.json** — FSM state at critical points (commit, partition, heal)
5. **audit-trail-member-*.json** — Authorization attempts per member
6. **network-partition-timeline.txt** — Exact timing and sequencing
7. **commit-index-proof-*.json** — Commit index at each fault point (R1-03 proof)
8. **test-environment-details.json** — Go version, Raft library, OS, kernel

**Artifact Integrity**

- All evidence files SHA256-hashed
- Hash manifest recorded with execution timestamp
- Timestamp precision: wall-clock seconds, monotonic sequencing
- Reproducibility: test harness, seed values, failure injection parameters recorded
- No external audit claim: retain "reproducible by independent test run", remove "verified by external auditor"

---

## Success Criteria Summary

| Criterion | Test Scenario(s) | Verification |
|-----------|---|---|
| **Replication** | R1-01 | Quorum-committed entry survives leader death |
| **At-Most-Once Disclosure** | R1-02 | Plaintext decrypted/released ≤1 times |
| **Atomicity** | R1-03 | Pre-commit loss → no consumption, no decryption |
| **Serialization** | R1-04 | Concurrent identical requests → exactly one committed |
| **Recovery** | R1-05 | Partitioned leader rejoins, converges, no replay |
| **No Regression** | Baseline A03/A04 | All tests pass, 0 races |
| **Evidence Reproducibility** | All Scenarios | Independent test run produces same proof |

---

## Unlock Criteria for A05

A05 remains LOCKED until all conditions are met:

1. ✗ Scenario R1-01 passes (leader failover + persistence)
2. ✗ Scenario R1-02 passes (ambiguous response + at-most-once)
3. ✗ Scenario R1-03 passes (pre-commit loss + no consumption)
4. ✗ Scenario R1-04 passes (cross-member concurrency + exactly-one)
5. ✗ Scenario R1-05 passes (old leader rejoins + convergence)
6. ✗ A03 regression suite passes
7. ✗ A04 regression suite passes (including timestamp boundary test)
8. ✗ Race detector: 0 races on all tests
9. ✗ Evidence artifacts generated with reproducible timestamp sequencing
10. ✗ Final qualification record generated with "VERIFIED / SEALED" status

---

## Next Steps

1. **Test Harness Development** — 3-member Raft cluster with deterministic fault injection
2. **Scenario Implementation** — R1-01 through R1-05 with evidence capture
3. **Execution** — Run full suite, collect evidence
4. **Verification** — Confirm reproducibility, no regressions
5. **Seal** — Generate final qualification record

**Implementation can proceed only after these technical corrections are in place.**

---

**This specification gates A05. The corrective path is narrow and focused: five concrete Raft scenarios, proper fault boundaries, evidence reproducibility, no speculative external audits.**

