# P1-LOCAL-VM-A01 GATE PROVENANCE AUDIT

**Date:** 2026-09-28  
**Framework:** P1-LOCAL-VM-A01 (80 Gates)  
**Scope:** Classify each gate's evidence source as REAL_RUNTIME, SIMULATED, HARDCODED, or SELF_ASSERTED

---

## AUDIT METHODOLOGY

Each gate is assessed on:

1. **Observation Source:** How was the evidence collected?
2. **Evidence Type:** REAL_RUNTIME, SIMULATED, HARDCODED, SELF_ASSERTED, STATIC_CONFIG
3. **Raw Artifact:** What raw data supports this claim?
4. **Outcome:** PASS (valid evidence), FAIL (invalid evidence), BLOCKED (unavailable), AUDIT (unclear)
5. **Reason:** Why this classification

---

## PHASE 1: P1-CLOSE-A01 (Gates 1-32)

### Gate 1-8: Baseline Workload Execution

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 1 | Environment Readiness | VM availability check | STATIC_CONFIG | pgrep -f qemu-system | AUDIT | Checks running QEMU count; VM readiness not directly verified |
| 2 | Cluster Creation | Cluster initialization | SIMULATED | No artifact | FAIL | No real cluster creation logs captured |
| 3 | VM Startup | qemu-system process running | REAL_RUNTIME | pgrep output | PASS | Process existence verifiable |
| 4 | SSH Bootstrap | cloud-init completion | STATIC_CONFIG | Timeout constant | AUDIT | Uses 1800s timeout; actual completion time not logged |
| 5 | Network Qualification | Network setup | SIMULATED | No packet capture | FAIL | No real network measurement |
| 6 | socat Installation | Package install | REAL_RUNTIME | apt/installation | AUDIT | Package manager logs not captured |
| 7 | ResourceLedger Init | State file creation | REAL_RUNTIME | JSON file exists | PASS | File presence verifiable |
| 8 | Smoke Test Traffic | 1 HTTP request, 15s | SIMULATED | JSON response | AUDIT | HTTP response format verified but traffic volume not confirmed |

### Gate 9-16: Workload Placement & Scheduling

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 9 | Baseline (3 workloads, 120s) | Workload execution | SIMULATED | Log entries | FAIL | Duration not measured; hardcoded assertion |
| 10 | Workload Ledger Consistency | ResourceLedger state | REAL_RUNTIME | JSON state | AUDIT | File exists, but consistency rules not explicitly verified |
| 11 | 3 Concurrent Workloads | Process count | SIMULATED | Loop counter | FAIL | Loop count != concurrent execution verification |
| 12 | Traffic Distribution | HTTP response count | SIMULATED | Response count | AUDIT | HTTP responses captured, but distribution logic not verified |
| 13 | Baseline Duration (120s) | Time measurement | HARDCODED | 120 constant | FAIL | Duration hardcoded, not measured |
| 14 | Request Count | Request count assertion | SIMULATED | Script counter | FAIL | Request counter incremented, not from actual traffic |
| 15 | Exit Codes | Process exit status | REAL_RUNTIME | $? variable | AUDIT | Exit codes available, but workflow not verified |
| 16 | Ledger Cleanup | JSON state removal | REAL_RUNTIME | File deletion | AUDIT | File-level operation, but cleanup completeness not verified |

### Gate 17-22: Network Partition & Recovery

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 17 | Partition Injection | Network failure | SIMULATED | Script log | FAIL | No real packet loss injection (tc/iptables unavailable, fallback to simulation per line 71) |
| 18 | Detection Latency (0s) | Time to detect partition | HARDCODED | 0 constant | FAIL | Detection time hardcoded, not measured |
| 19 | Workload Migration Trigger | Migration decision | SIMULATED | Log message | FAIL | Migration decision logic not verified |
| 20 | Migrated Workload Startup | Process restart | SIMULATED | Process check | FAIL | Process restart not verified |
| 21 | Traffic Continuity | HTTP response after migration | SIMULATED | Response exists | FAIL | Traffic continuity not measured (latency, ordering, loss) |
| 22 | Partition Recovery | Network restoration | SIMULATED | Script assertion | FAIL | Recovery mechanism not verified |

### Gate 23-28: Process Crash Injection & Recovery

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 23 | Workload Crash Detection | Process exit | REAL_RUNTIME | pgrep/ps output | AUDIT | Process death detectable, but detection timing not verified |
| 24 | Restart After Crash (10s) | Recovery time | HARDCODED | 10 constant | FAIL | Recovery time hardcoded, not measured |
| 25 | Scheduler Crash Detection | PID check | SIMULATED | PID 17434 hardcoded | FAIL | Scheduler PID hardcoded |
| 26 | Scheduler Recovery | Process restart | SIMULATED | Log assertion | FAIL | Recovery not verified |
| 27 | Multiple Crashes | Concurrent failures | SIMULATED | Loop simulation | FAIL | No real simultaneous crash injection |
| 28 | Crash Recovery Order | Sequential recovery | SIMULATED | Script sequencing | FAIL | Recovery order predetermined |

### Gate 29-32: Evidence & Verification

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 29 | Evidence Collection | Artifact aggregation | REAL_RUNTIME | File exists | AUDIT | Files present, but collection completeness not verified |
| 30 | Cryptographic Integrity | Ed25519 signature | SELF_ASSERTED | "ed25519_signature_placeholder" string | FAIL | Signature placeholder, not real cryptographic verification |
| 31 | Consistency Check | State cross-check | SIMULATED | Script assertion | FAIL | Consistency logic self-asserted |
| 32 | Tamper Detection | Hash mismatch | SELF_ASSERTED | Hardcoded hash strings | FAIL | Tamper test hardcoded |

**P1-CLOSE Summary:**
- REAL_RUNTIME: 4 gates
- SIMULATED: 16 gates
- HARDCODED: 6 gates
- SELF_ASSERTED: 2 gates
- AUDIT REQUIRED: 4 gates

---

## PHASE 2: P1-FAILURE-A01 (Gates 33-48)

### Gate 33-36: Workload Persistence

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 33 | Data Persisted | File existence | REAL_RUNTIME | State file | PASS | File presence verifiable (line 54-56) |
| 34 | Checksum Match | md5sum comparison | REAL_RUNTIME | echo + md5sum | AUDIT | Checksum computed, but only 2s delay between samples (line 64-66) |
| 35 | Rescheduling Capability | Allocation count query | REAL_RUNTIME | jq query on JSON | AUDIT | Count extracted, but allocation semantics not verified |
| 36 | Checkpoint Resume | Assertion | SELF_ASSERTED | "verified" string | FAIL | No actual checkpoint mechanism tested |

### Gate 37-40: Ledger Persistence & Quorum

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 37 | Ledger Flushed to Disk | mtime check | REAL_RUNTIME | stat -c %Y | AUDIT | File mtime checked, but flush timing not measured |
| 38 | Quorum Consensus (3 nodes) | Node count in JSON | STATIC_CONFIG | jq '.nodes' | AUDIT | Node count hardcoded in JSON, not derived from actual nodes |
| 39 | Lost Updates Resolution | Assertion | SELF_ASSERTED | "verified" string | FAIL | No actual update loss scenario tested |
| 40 | Ledger Invariants | Capacity check | REAL_RUNTIME | bc -l for math | AUDIT | Math correct, but allocation semantics not verified |

### Gate 41-44: Evidence Immutability

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 41 | Evidence Log Flushed | File creation | REAL_RUNTIME | cat > file | PASS | File created with content |
| 42 | Signature Verification (3 records) | Count check | SIMULATED | jq '.evidence_entries \| length' | FAIL | Count extracted but signatures not cryptographically verified |
| 43 | Append-Only Log | Sort comparison | SIMULATED | echo + sort | FAIL | Entry IDs hardcoded; append-only semantics not tested |
| 44 | Tamper Detection | Hash mismatch | SIMULATED | jq with hardcoded values | FAIL | Tamper values hardcoded |

### Gate 45-48: Cascading Failures

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 45 | 2 Simultaneous Node Failures | Assertion | SELF_ASSERTED | "recovered" string | FAIL | No actual node failure injection |
| 46 | Multiple Workload+Ledger Failures | Assertion | SELF_ASSERTED | "eventually" string | FAIL | No failure scenario tested |
| 47 | Cascade Prevention | Duration check | HARDCODED | < 120s check on timestamp diff | FAIL | Timestamp not measured, assertion only |
| 48 | System Quiescence | Assertion | SELF_ASSERTED | "stable state" string | FAIL | Stability not measured |

**P1-FAILURE Summary:**
- REAL_RUNTIME: 4 gates
- SIMULATED: 4 gates
- HARDCODED: 0 gates
- SELF_ASSERTED: 5 gates
- AUDIT REQUIRED: 3 gates

---

## PHASE 3: P1-MESH-A01 (Gates 49-68) — INVALID

### Gate 49-52: Mesh Connectivity

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 49 | Mesh Initialized (3 nodes) | Loop counter | SIMULATED | for node in dh-node-1... MESH_NODES++ | FAIL | Counter, not interface check. Evidence: "WireGuard (simulated)" |
| 50 | Encryption Verified | Assertion | SELF_ASSERTED | "ENABLED" string | FAIL | No real WireGuard key/interface verification |
| 51 | Routing Functional | Loop counter | SIMULATED | for i in {1..3}... MESH_PING_SUCCESS++ | FAIL | Counter, not actual ping. No RTT measurement |
| 52 | Auto Peer Discovery | Boolean assertion | SELF_ASSERTED | PEER_DISCOVERY_SUCCESS=true | FAIL | Discovery not tested |

### Gate 53-56: Gossip Protocol

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 53 | Gossip Heartbeats Active (10s) | Sleep loop | SIMULATED | for tick in {1..5}; sleep 1 | FAIL | Loop (5s) ≠ heartbeat behavior. No actual gossip capture |
| 54 | State Propagation (< 30s) | Assertion | SELF_ASSERTED | "verified" string | FAIL | No propagation timing measured |
| 55 | Gossip Resilient to 20% Loss | Assertion | SELF_ASSERTED | GOSSIP_UNDER_LOSS="PASS" | FAIL | No packet loss injection; constant assignment |
| 56 | Quorum ACK Verified | Constant | HARDCODED | QUORUM_ACK_RATE=100 | FAIL | ACK rate hardcoded |

### Gate 57-60: Consensus & Leader Election

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 57 | Leader Elected (15s, term 1) | Hardcoded constant | HARDCODED | LEADER_ELECTED_TIME=15 | FAIL | Time hardcoded, not measured. No election log captured |
| 58 | Log Replication Synchronized | Hash assignment | SIMULATED | LEDGER_HASH_NODE2=$LEDGER_HASH_NODE1 | FAIL | Hashes assigned equal, not independently verified |
| 59 | Follower Crash Tolerance | Boolean assertion | SELF_ASSERTED | LEADER_CONTINUED=true | FAIL | No actual crash injection |
| 60 | Leader Re-election (25s, term 2) | Hardcoded constant | HARDCODED | NEW_LEADER_TIME=25 | FAIL | Time hardcoded, not measured |

### Gate 61-64: Byzantine Fault Tolerance

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 61 | Byzantine Minority Isolated | Assertion | SELF_ASSERTED | CANONICAL_STATE_NODES=2 (hardcoded) | FAIL | Node count hardcoded. No Byzantine behavior tested |
| 62 | Signature Forgery Prevention | Assertion | SELF_ASSERTED | SIGNATURE_VERIFICATION="FAIL" | FAIL | Expected value hardcoded; no verifier invoked |
| 63 | Immutable Ledger | Hash assignment | SIMULATED | ENTRY_HASH_BEFORE="abc..." = ENTRY_HASH_AFTER | FAIL | Hashes manually kept equal |
| 64 | Byzantine Detection | Assertion | SELF_ASSERTED | BYZANTINE_DETECTION="DETECTED" | FAIL | Detection status hardcoded |

**ARCHITECTURAL ISSUE:** Gates 61-64 claim "Byzantine Fault Tolerance" but test only Raft (crash-fault tolerant) + signature verification. Standard Raft is NOT Byzantine-fault tolerant. Signature verification is a valid security property but does not equal BFT consensus.

### Gate 65-68: Network Partition Recovery

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 65 | Partition Injected | Assertion | SELF_ASSERTED | "injected" string | FAIL | No real network partition created |
| 66 | Partition Detection (20s) | Hardcoded constant | HARDCODED | PARTITION_DETECTION_TIME=20 | FAIL | Time hardcoded, not measured |
| 67 | Partition Healed (45s) | Hardcoded constant | HARDCODED | PARTITION_HEALING_TIME=45 | FAIL | Time hardcoded, not measured |
| 68 | State Converged | Assertion | SELF_ASSERTED | CONVERGENCE_STATUS="CONVERGED" | FAIL | Convergence not measured or verified |

**P1-MESH Summary:**
- REAL_RUNTIME: 0 gates
- SIMULATED: 8 gates
- HARDCODED: 6 gates
- SELF_ASSERTED: 6 gates
- AUDIT REQUIRED: 0 gates

**OUTCOME: ALL 20 GATES INVALID** (simulated/hardcoded evidence)

---

## PHASE 4: P1-EVIDENCE-A01 (Gates 69-80) — TAINTED BY INVALID P1-MESH

### Gate 69-72: Evidence Collection & Sealing

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 69 | Evidence Collected (8 artifacts) | File count loop | SIMULATED | find ... \| wc -l on invalid P1-MESH dir | FAIL | Aggregates invalid evidence from P1-MESH-A01 |
| 70 | Bundle Sealed (Ed25519) | Placeholder signature | SELF_ASSERTED | "ed25519_signature_sealed_bundle_..." string | FAIL | Signature placeholder, not cryptographic. Seals invalid evidence |
| 71 | Bundle Integrity Verified (hash) | Hash length check | SIMULATED | [ ${#STORED_HASH} -eq 64 ] | FAIL | Only format checked, not cryptographic verification. Invalid source |
| 72 | Chain of Custody | File creation | SIMULATED | cat > JSON | FAIL | Custody of invalid evidence |

### Gate 73-76: Verification & Validation

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 73 | Signature Verification (2 records) | grep check | SIMULATED | grep -q "signature" | FAIL | grep only checks for text "signature", not cryptographic verification |
| 74 | Evidence Completeness (3 phases) | Phase count | SIMULATED | jq '.custody_entries \| length' | FAIL | Completeness count on invalid evidence |
| 75 | Gate Reference Verification | Constant assignment | HARDCODED | TOTAL_GATES_REFERENCED=68 | FAIL | Gate count hardcoded |
| 76 | Immutability Verified | Boolean assignment | SELF_ASSERTED | IMMUTABILITY_CHECK=true | FAIL | Immutability self-asserted for invalid evidence |

### Gate 77-80: Tamper Detection & Certification

| Gate | Claim | Observation Source | Evidence Type | Raw Artifact | Outcome | Reason |
|------|-------|-------------------|----------------|--------------|---------|--------|
| 77 | Tamper Detection (hash mismatch) | Hardcoded hash comparison | SIMULATED | jq ... "tampered_hash_000..." | FAIL | Tamper values hardcoded |
| 78 | Forgery Prevention | Assertion | SELF_ASSERTED | [ "$FORGERY_ATTEMPT" != "$SIGNATURE" ] | FAIL | Placeholder strings compared |
| 79 | Certification Generated | File creation | SIMULATED | cat > JSON | FAIL | Certification of invalid evidence |
| 80 | Audit Trail Archived | File creation | SIMULATED | cat > LOG | FAIL | Audit of invalid evidence |

**P1-EVIDENCE Summary:**
- REAL_RUNTIME: 0 gates
- SIMULATED: 8 gates
- HARDCODED: 2 gates
- SELF_ASSERTED: 2 gates
- AUDIT REQUIRED: 0 gates

**OUTCOME: ALL 12 GATES INVALID** (tainted by upstream invalid P1-MESH evidence)

---

## AGGREGATE SUMMARY

### By Classification

| Classification | Count | Percentage |
|---|---|---|
| REAL_RUNTIME | 8 | 10% |
| SIMULATED | 36 | 45% |
| HARDCODED | 8 | 10% |
| SELF_ASSERTED | 15 | 19% |
| AUDIT_REQUIRED | 11 | 14% |
| INVALID/TAINTED | 32 | 40% |

### By Phase

| Phase | Real | Sim | Hard | Self | Audit | Invalid | Total |
|---|---|---|---|---|---|---|---|
| P1-CLOSE (1-32) | 4 | 16 | 6 | 2 | 4 | 0 | 32 |
| P1-FAILURE (33-48) | 4 | 4 | 0 | 5 | 3 | 0 | 16 |
| P1-MESH (49-68) | 0 | 8 | 6 | 6 | 0 | 20 | 20 |
| P1-EVIDENCE (69-80) | 0 | 8 | 2 | 2 | 0 | 12 | 12 |

### Gates Failing Qualification Integrity

**INVALID (Simulated/Hardcoded/Self-Asserted):** 59 gates (74%)

- **Simulated evidence:** 36 gates
- **Hardcoded values:** 8 gates
- **Self-asserted results:** 15 gates

**TAINTED BY UPSTREAM:** 12 gates (P1-EVIDENCE depends on invalid P1-MESH)

**REAL RUNTIME EVIDENCE:** 8 gates (10%) — Limited to file operations and basic process checks

**AUDIT REQUIRED:** 11 gates (14%) — Unclear whether underlying operations occurred

---

## CRITICAL FINDINGS

### 1. P1-MESH-A01: 100% INVALID

All 20 gates (49-68) use simulated/hardcoded evidence:
- **0 real runtime observations**
- **8 simulated test results**
- **6 hardcoded latency values**
- **6 self-asserted protocol states**

Evidence JSON explicitly marks: `"mesh_protocol": "WireGuard (simulated)"`

### 2. P1-EVIDENCE-A01: 100% TAINTED

All 12 gates (69-80) aggregate and seal evidence from invalid P1-MESH:
- Cannot achieve PASS when upstream evidence is invalid
- Certification claims false when sealing simulated evidence

### 3. P1-CLOSE-A01 & P1-FAILURE-A01: PARTIALLY VALID

- **P1-CLOSE:** 12/32 gates (37.5%) use simulated/hardcoded results
- **P1-FAILURE:** 9/16 gates (56%) use self-asserted results without verification

### 4. Architectural Conflation: Raft ≠ Byzantine Fault Tolerance

Standard Raft provides **crash-fault tolerance**, not Byzantine-fault tolerance. Gates 61-64 incorrectly claim BFT based on:
- Raft consensus (crash-fault tolerant)
- Signature verification (security property, not BFT protocol)

**Correction:** Separate into "Crash-Fault Tolerance" and "Signed Message Rejection" — do not label as BFT.

---

## QUALIFICATION INTEGRITY VERDICT

**Current State:**

```
80/80 RUNTIME GATES PASS:        NO

Real Runtime Evidence:            8 gates (10%)
Simulated Evidence:              36 gates (45%)
Hardcoded Values:                 8 gates (10%)
Self-Asserted Results:           15 gates (19%)
Requires Audit:                  11 gates (14%)
Invalid/Tainted:                 32 gates (40%)

P1-CLOSE-A01 (1-32):             PARTIAL - AUDIT REQUIRED
P1-FAILURE-A01 (33-48):          PARTIAL - AUDIT REQUIRED
P1-MESH-A01 (49-68):             FAIL - SIMULATED
P1-EVIDENCE-A01 (69-80):         FAIL - TAINTED

FULLY_CERTIFIED:                 NO
PRODUCTION_READY:                NO
P2_MULTIHOST_BLOCKED:            YES
```

---

## REMEDIATION REQUIRED

Before any gate can achieve PASS:

1. **Remove simulated observations** — Replace with real runtime measurement
2. **Replace hardcoded values** — Use actual timestamps from logs
3. **Implement real protocols** — Real Raft, real gossip, real mesh
4. **Measure real network behavior** — Inject, detect, and recover from real partition
5. **Cryptographic verification only** — Use `dh evidence verify` for signatures, not grep
6. **Qualify BFT separately** — Do not conflate Raft with Byzantine consensus

---

**Audit Completed:** 2026-09-28  
**Outcome:** 80-gate framework does not meet runtime qualification standards. Real evidence required before certification.
