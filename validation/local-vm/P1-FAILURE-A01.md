# P1-FAILURE-A01: Persistence Testing & Failure Recovery

**Phase:** P1-FAILURE (follows P1-CLOSE-A01 qualification)  
**Objective:** Test system persistence and failure recovery with real persistence  
**Status:** DESIGN COMPLETE, awaiting P1-CLOSE qualification

---

## Overview

P1-FAILURE-A01 tests the system's ability to persist state across failures and recover to consistent states. Unlike P1-CLOSE (which measures failure detection and immediate recovery), P1-FAILURE tests:

1. **State Persistence**: Workload data, allocations, ledger state survive node restarts
2. **Failure Recovery**: System reaches consistent state after forced failures
3. **Evidence Immutability**: Evidence logs cannot be modified post-failure
4. **Cascading Failures**: Multiple simultaneous failures handled correctly

---

## Gates 33-48 (16 gates)

### Gates 33-36: Workload Persistence
**Gate 33:** Workload data written to persistent storage before failure  
- **Metric:** Workload process writes data to node's persistent store
- **Evidence:** File timestamps in persistent storage
- **Failure scenario:** Data lost on crash (would FAIL this gate)

**Gate 34:** Data survives node restart  
- **Metric:** Written data recoverable after node reboot
- **Evidence:** Read data checksums match written checksums
- **Failure scenario:** Data corrupted (checksum mismatch)

**Gate 35:** Workload re-scheduled after restart  
- **Metric:** Scheduler detects unavailable allocation, reassigns workload
- **Evidence:** New allocation_id in placement log
- **Failure scenario:** Workload orphaned (never reassigned)

**Gate 36:** Workload resumes from saved checkpoint  
- **Metric:** Application continues from last checkpoint, not from start
- **Evidence:** Log shows "resumed from checkpoint X" not "started from beginning"
- **Failure scenario:** Restarts from beginning (loses progress)

### Gates 37-40: Ledger Persistence
**Gate 37:** Ledger state written before any node shutdown  
- **Metric:** Ledger JSON flushed to disk on all nodes before failure
- **Evidence:** Ledger file modification time <= failure timestamp
- **Failure scenario:** Ledger modified after node stops (phantom writes)

**Gate 38:** Ledger quorum consensus maintained  
- **Metric:** At least 2/3 nodes agree on canonical ledger state after restart
- **Evidence:** Ledger hash on 2+ nodes matches canonical
- **Failure scenario:** Split-brain (nodes disagree on state)

**Gate 39:** Lost updates resolved via quorum  
- **Metric:** Allocations from stopped node are re-assigned to available nodes
- **Evidence:** Old allocations marked "lost", new ones assigned
- **Failure scenario:** Old allocations remain as "phantom" (never cleaned up)

**Gate 40:** Ledger consistency cross-check after failure  
- **Metric:** Ledger invariants hold (sum of allocations ≤ capacity)
- **Evidence:** Invariant check runs, all assertions pass
- **Failure scenario:** Inconsistent state (allocation sum > capacity)

### Gates 41-44: Evidence Immutability
**Gate 41:** Evidence log flushed before failure  
- **Metric:** Evidence entries written to persistent storage before crash
- **Evidence:** Evidence file's mtime before or at failure time
- **Failure scenario:** Missing evidence entries (written but not persisted)

**Gate 42:** Evidence signatures valid after restart  
- **Metric:** All evidence records pass Ed25519 verification
- **Evidence:** signature_verify() returns true for all records
- **Failure scenario:** Signatures invalid (evidence tampered)

**Gate 43:** Evidence log is append-only  
- **Metric:** No evidence entries deleted or reordered post-failure
- **Evidence:** Entry sequence numbers are contiguous and monotonic
- **Failure scenario:** Missing or reordered entries

**Gate 44:** Tamper detection: modified evidence detected  
- **Metric:** System detects if any evidence record was modified
- **Evidence:** Signature verification fails on tampered record
- **Failure scenario:** Tamper goes undetected (signature still valid)

### Gates 45-48: Cascading Failures
**Gate 45:** Two simultaneous node failures handled  
- **Metric:** System recovers after 2/3 nodes fail simultaneously
- **Evidence:** Remaining node reaches quorum (1 node minimum) and continues
- **Failure scenario:** System deadlocks waiting for quorum

**Gate 46:** Multiple workload failures + ledger failure = recovery  
- **Metric:** System recovers after N workload crashes + node crash
- **Evidence:** All workloads eventually re-scheduled, ledger consistent
- **Failure scenario:** Workloads remain orphaned or ledger inconsistent

**Gate 47:** Failure cascade doesn't propagate  
- **Metric:** Crash of one node doesn't trigger crashes on other nodes
- **Evidence:** Sibling nodes remain healthy and running after peer fails
- **Failure scenario:** Cascading crashes (one failure triggers others)

**Gate 48:** System reaches quiescent state after cascade  
- **Metric:** All recovery operations complete, no pending work
- **Evidence:** Ledger stabilizes, no more reallocations, all workloads running
- **Failure scenario:** System in active recovery loop (never quiescent)

---

## Execution Flow

```
[P1-CLOSE-A01 PASS]
        ↓
[Initialize P1-FAILURE environment]
        ↓
[Baseline: 3 workloads, 120s]
        ↓
[Inject node failure: dh-node-2 crashes]
        ↓
[Gates 33-36: Workload persistence, recovery, reschedule]
        ↓
[Inject ledger failure: Force quorum mismatch]
        ↓
[Gates 37-40: Ledger persistence, quorum recovery, invariant check]
        ↓
[Collect evidence, verify signatures]
        ↓
[Gates 41-44: Evidence immutability, tamper detection]
        ↓
[Inject cascading failures: Node 1 + 3 fail simultaneously]
        ↓
[Gates 45-48: Cascade handling, recovery, quiescence]
        ↓
[Generate P1-FAILURE-A01 report (gates 33-48)]
        ↓
[Proceed to P1-MESH-A01]
```

---

## Key Metrics

| Metric | Target | Method |
|--------|--------|--------|
| Workload data recovery time | <30s | Time from node restart to data readable |
| Ledger quorum consensus time | <60s | Time to reach 2/3 node agreement |
| Workload rescheduling latency | <15s | Time from node failure to new allocation |
| Evidence log flushing | Synchronous | Evidence written to disk before return |
| Evidence signature verification | 100% | All records pass cryptographic check |
| Cascading failure propagation | 0 | No secondary failures triggered |
| System quiescence time | <120s | Time to stable state after failures |

---

## Testing Strategy

### 1. Single Node Failure (Gates 33-36)
- **Setup:** 3 workloads running for 30s
- **Failure:** Force one node to fail (SIGKILL to scheduler + workload processes)
- **Observation:** Workload detection, rescheduling, persistence
- **Expected:** Workload resumes on different node within 15s

### 2. Ledger Failure (Gates 37-40)
- **Setup:** Ledger quorum operational (all 3 nodes agree)
- **Failure:** Corrupt ledger on one node, restart node
- **Observation:** Quorum resolution, lost update detection
- **Expected:** System reaches consensus on canonical ledger (2 votes for correct version)

### 3. Evidence Verification (Gates 41-44)
- **Setup:** Evidence collected across all failure scenarios
- **Failure:** Attempt to modify one evidence entry
- **Observation:** Signature verification, tamper detection
- **Expected:** Modified entry fails signature check, system raises alarm

### 4. Cascading Failures (Gates 45-48)
- **Setup:** Baseline running on all 3 nodes
- **Failure:** Crash nodes 1 and 3 simultaneously
- **Observation:** Recovery on remaining node, workload redistribution
- **Expected:** Node 2 survives, workloads eventually resume (may be degraded)

---

## Evidence Artifacts

### Workload Persistence Evidence
```
/evidence/P1-FAILURE-A01/workload-data-before-crash.json
/evidence/P1-FAILURE-A01/workload-data-after-restart.json
/evidence/P1-FAILURE-A01/workload-persistence.log
```

### Ledger Persistence Evidence
```
/evidence/P1-FAILURE-A01/ledger-state-pre-failure.json
/evidence/P1-FAILURE-A01/ledger-state-post-recovery.json
/evidence/P1-FAILURE-A01/ledger-consensus-log.log
```

### Evidence Immutability Records
```
/evidence/P1-FAILURE-A01/evidence-manifest.json (Ed25519 signatures)
/evidence/P1-FAILURE-A01/evidence-signatures.log
/evidence/P1-FAILURE-A01/tamper-detection-results.log
```

### Failure Logs
```
/evidence/P1-FAILURE-A01/node-failure-timeline.json
/evidence/P1-FAILURE-A01/recovery-actions.log
/evidence/P1-FAILURE-A01/cascade-analysis.json
```

---

## Success Criteria

- ✅ All 16 gates (33-48) achieve PASS or justified BLOCKED status
- ✅ Evidence artifacts signed and archived
- ✅ No gates weakened to make tests pass
- ✅ Failure scenarios are real (not simulated)
- ✅ Recovery is autonomous (no manual intervention)
- ✅ System reaches stable state within defined timeouts

---

## Blocking Conditions

- ❌ P1-CLOSE-A01 gates 1-32 must all PASS
- ❌ If any gate FAIL, investigation required before proceeding
- ❌ External blockers documented in `/evidence/P1-FAILURE-A01/blockers.md`

---

## Next Phase: P1-MESH-A01

Upon P1-FAILURE-A01 completion, the qualification proceeds to P1-MESH-A01:
- Multi-node mesh networking
- Distributed consensus
- Byzantine fault tolerance
- Network partition recovery

---

## Repository Structure

```
validation/local-vm/
├── P1-CLOSE-GATES.md              ← Phase 1 spec (32 gates)
├── P1-FAILURE-A01.md              ← This file (Phase 2 design)
├── P1-MESH-A01.md                 ← Phase 3 design (TODO)
├── P1-EVIDENCE-A01.md             ← Phase 4 design (TODO)
├── scripts/
│   ├── p1-qualification-master.sh
│   ├── run-scheduler.sh
│   ├── inject-network-partition.sh
│   ├── inject-process-crash.sh
│   ├── inject-workload-crash.sh    ← NEW for P1-FAILURE
│   └── ...
├── state/
│   └── [cluster, workloads, evidence]
└── evidence/
    ├── P1-CLOSE-A01/              ← Phase 1 evidence
    └── P1-FAILURE-A01/            ← Phase 2 evidence (created after PASS)
```

