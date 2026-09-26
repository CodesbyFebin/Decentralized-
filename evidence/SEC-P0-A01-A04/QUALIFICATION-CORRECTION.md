# SEC-P0-A01-A04 Qualification Correction and Gap Analysis

**Date:** 2026-09-26  
**Status:** QUALIFICATION BLOCKED - Material Gap Identified  
**Correction Basis:** Review of evidence against distributed-systems requirements  

## Issue Summary

The initial qualification record (record.json, QUALIFICATION.md, TEST-SUMMARY.md) contains a material misclassification that oversells the evidence:

**Claimed:** "Failover resilience (snapshot/restore persistence)"  
**Actual:** "Replay-ledger persistence across FSM snapshot/restore"

These are **not equivalent**.

## Technical Gap

### What Snapshot/Restore Tests Prove
```
ConsumedAuthorization entry
  ↓
FSM serialization (JSON marshal)
  ↓
FSM snapshot
  ↓
FSM restore (JSON unmarshal)
  ↓
ConsumedAuthorization entry still present
```

**Conclusion:** Local state persistence is working.

### What Real Raft Leader Failover Must Prove
```
leader A receives authorization request
  ↓
Raft applies command to FSM (all members)
  ↓
consumption committed to replicated log
  ↓
leader A crashes / isolation / network partition
  ↓
leader B or C elected (new term)
  ↓
identical signed request sent to new leader
  ↓
EXPECTED: DENY_ALREADY_CONSUMED
  ↓
ACTUAL: ???
```

**This test was never executed.**

### Additional Untested Scenarios

1. **Ambiguous Response Boundary**
   - Leader A commits consumption
   - Leader A dies before response reaches client
   - Client retries with same request
   - Expected: replay rejected, plaintext not released twice

2. **Pre-Commit Leadership Loss**
   - Authorization proposal starts on leader A
   - Leader A loses quorum before FSM commit
   - No consumption record committed globally
   - Expected: NO decryption, NO plaintext response

3. **Cross-Member Concurrency During Failover**
   - Identical requests concurrently sent to multiple members during leadership transition
   - Expected: exactly one consumption committed globally

## Evidence Classification Correction

### Original Record - Misclassified Sections

**From TEST-SUMMARY.md:**
```
| Test | Focus | Result |
|------|-------|--------|
| `TestFSMSnapshot_ReplayLedgerPersists` | Snapshot/restore cycle | PASS |
| `TestFSMSnapshot_SnapshotRecovery` | Multi-roundtrip persistence | PASS |
```

**Corrected Classification:**
```
Category: PERSISTENCE/RECOVERY (local FSM state)
NOT: Failover/Distributed Consensus
```

**From QUALIFICATION.md:**
```
#### 3. Failover Resilience
**Claim:** Replay ledger survives Raft leader failover and node restart.
```

**Corrected Classification:**
```
#### 3. Snapshot/Restore Persistence
**Claim:** Replay ledger survives FSM serialization and deserialization.
**Scope:** Local state machine only; does not test distributed consensus.
```

**From VERIFICATION-CHECKLIST.md:**
```
### S3: Failover Resilience
- [x] FSM implements raft.FSM interface
- [x] Snapshot() serializes state to JSON
- [x] Restore() deserializes JSON and rebuilds state
- [x] ReplayLedger persisted in FSM.s.ReplayLedger
- [x] Test: `TestFSMSnapshot_ReplayLedgerPersists` - PASS
- [x] Test: `TestFSMSnapshot_SnapshotRecovery` - PASS
```

**Corrected Assessment:**
```
### S3: Snapshot/Restore Persistence
- [x] FSM implements raft.FSM interface
- [x] Snapshot() serializes state to JSON
- [x] Restore() deserializes JSON and rebuilds state
- [x] ReplayLedger persisted in FSM.s.ReplayLedger
- [x] Test: `TestFSMSnapshot_ReplayLedgerPersists` - PASS (local persistence)
- [x] Test: `TestFSMSnapshot_SnapshotRecovery` - PASS (local persistence)
- [ ] Real Raft leader failover - NOT YET TESTED
```

## Current Qualification Status

| Aspect | Status | Evidence |
|--------|--------|----------|
| Code Implementation | ✓ PASS | Merged to main |
| Unit Authorization Tests | ✓ PASS | 15 tests, all pass |
| High-Contention Concurrency | ✓ PASS | 50 goroutines, exactly-once |
| Snapshot/Restore Persistence | ✓ PASS | 2 tests, 2 roundtrips |
| Timestamp Regression | ✓ FIXED | Nanoseconds → milliseconds |
| Race Detector | ✓ PASS | 47 tests, 0 races |
| **Real Raft Leader Failover** | **✗ MISSING** | Not demonstrated |
| **Ambiguous Response Handling** | **✗ MISSING** | Not demonstrated |
| **Pre-Commit Leader Loss** | **✗ MISSING** | Not demonstrated |
| **Cross-Member Replay Contention** | **✗ MISSING** | Not demonstrated |

## Production-Ready Assessment

**Original Claim:** "APPROVED FOR PRODUCTION DEPLOYMENT"  
**Corrected Assessment:** **NOT SUPPORTED BY THIS EVIDENCE**

The implementation is solid and well-tested locally. However, the claim that it is "production-ready" requires proof of distributed-systems properties that have not yet been demonstrated:

1. **Leader failover does not lose authorization state**
2. **Concurrent replay across members yields exactly-once consumption**
3. **Ambiguous responses (response loss) do not cause double-release**
4. **Pre-commit leader failure produces no unauthorized access**

These are not implementation issues. They are qualification gaps.

## Preserved Evidence

All files in `evidence/SEC-P0-A01-A04/` remain unchanged:
- `record.json` - Formal test results (still valid, misclassified claims removed)
- `QUALIFICATION.md` - Original attestation (misclassified sections noted)
- `TEST-SUMMARY.md` - Test analysis (sections corrected by this document)
- `VERIFICATION-CHECKLIST.md` - Verification (failover claims withdrawn)
- `DEPLOYMENT-GUIDE.md` - Procedures (contingent on real failover proof)
- Supporting logs and manifests (unchanged)

This correction document is **supplementary**, not a rewrite.

## Next Phase: SEC-P0-A01-A04-FAILOVER-R1

A new, narrowly-scoped qualification (R1 = Revision 1) is required to prove the missing distributed-systems properties. Specification follows in `FAILOVER-R1-SPEC.md`.

## Summary

- **Implementation Quality:** Good ✓
- **Local Testing:** Thorough ✓
- **Distributed Property Testing:** Incomplete ✗
- **Production-Ready Claim:** Withdrawn
- **Qualification Status:** BLOCKED pending real failover testing
- **A05 Lock Status:** LOCKED (depends on A04 completion)

The corrective gate (R1) is narrow and focused. No new authorization features are required unless failover testing discovers defects.

---

**This document preserves the original evidence while correcting the qualification assessment. The implementation remains merged; the qualification remains incomplete pending real Raft failover verification.**
