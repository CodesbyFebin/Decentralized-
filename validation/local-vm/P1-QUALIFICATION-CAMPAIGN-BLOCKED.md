# P1 Qualification Campaign Structure

**Status**: NOT_STARTED (blocked pending infrastructure)  
**Discovery Source SHA**: `e133fcf0c3908343ee08b7578aac9c3a53adc081`  
**P1_CANDIDATE_SHA**: NONE (pending remediation diagnostics + integration)  
**Campaign ID**: NOT_ASSIGNED  
**Phase**: INITIALIZATION

---

## Prerequisites (ALL MUST COMPLETE FIRST)

```
✓ SOURCE_SHA=e133fcf complete and reviewed
✓ IMPLEMENTED=YES (16-17 commits, full remediation)
✓ LOCAL_TESTED=YES (CI environment)
⏳ RUNTIME_TESTED=NO (BLOCKED: infrastructure unavailable)
⏳ REMEDIATION_DIAGNOSTICS=NOT_STARTED (blocked)
⏳ PR #31=NOT_INTEGRATED (blocked on diagnostics)
⏳ SOURCE_FREEZE=NOT_LOCKED (blocked on integration)
⏳ P1_CANDIDATE_SHA=NOT_ASSIGNED (blocked on freeze)
⏳ CAMPAIGN_INITIALIZED=NOT_READY (blocked on candidate SHA)
```

### Unblock Sequence

```
OPERATOR_PROVISIONS_REAL_INFRASTRUCTURE
    ↓
RUNTIME_DISCOVERY=PASS
BACKEND_SELECTED=<one of: physical, VMs, Kubernetes, persistent-Docker, single-host-containers>
    ↓
REMEDIATION_DIAGNOSTICS_EXECUTED
    ↓
IF ANY_DIAGNOSTIC_FAILS:
    → Root-cause in source
    → Fix implementation
    → New commit & SHA
    → Rerun affected diagnostics
    → Continue until all pass
    ↓
REMEDIATION_DIAGNOSTICS=PASS (all 26 phases)
    ↓
EVIDENCE_REVIEW=PASS (raw artifacts reviewed, no anomalies)
    ↓
PR #31 FINAL REVIEW=PASS
    ↓
PR #31 INTEGRATED INTO MAIN
    ↓
INTEGRATED_SHA_VERIFIED (identical implementation)
    ↓
P1_CANDIDATE_SHA=<exact integrated SHA>
    ↓
SOURCE_FREEZE (no changes during P1 campaign)
    ↓
CAMPAIGN_INITIALIZED (all 32 gates UNKNOWN)
    ↓
EXECUTE 32 DECISIVE GATES
    ↓
AUTHORITATIVE_VERIFICATION
    ↓
QUALIFICATION/SEALING_VERDICT
```

---

## P1 Campaign: 32 Decisive Gates (Backend-Neutral)

All gates initialize as **UNKNOWN** after source freeze.  
All gates will be executed against **P1_CANDIDATE_SHA** (frozen, not advancing).  
No gate execution can begin until infrastructure is available.

### Gate 01–08: Signed Intent & Local Policy

```
GATE_01=UNKNOWN  "Ed25519 identity binding present"
GATE_02=UNKNOWN  "Work proposed as signed intent structures"
GATE_03=UNKNOWN  "Envelope canonical JSON serialization"
GATE_04=UNKNOWN  "Domain-prefix anti-forgery (decentralized.host/{kind}/v1)"
GATE_05=UNKNOWN  "Ed25519 signature verification enforces kind match"
GATE_06=UNKNOWN  "Per-host local policy enforcement with audit trail"
GATE_07=UNKNOWN  "Local policy rejection prevents execution"
GATE_08=UNKNOWN  "No silent task migration observed"
```

### Gate 09–16: State Machine & ResourceLedger

```
GATE_09=UNKNOWN  "State separation: DESIRED ≠ ADMITTED ≠ EXECUTING ≠ OBSERVED ≠ VERIFIED"
GATE_10=UNKNOWN  "ResourceLedger Model A: AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED"
GATE_11=UNKNOWN  "Reservation-to-allocation atomic transfer (no double-count)"
GATE_12=UNKNOWN  "Concurrent placement requests serialize on ledger lock"
GATE_13=UNKNOWN  "Ledger invariant preserved under concurrency"
GATE_14=UNKNOWN  "Insufficient capacity rejection"
GATE_15=UNKNOWN  "Owner reserve enforcement"
GATE_16=UNKNOWN  "Cordon/uncordon lifecycle correct"
```

### Gate 17–24: Failure Detection & Recovery

```
GATE_17=UNKNOWN  "Three real Linux runtime nodes active"
GATE_18=UNKNOWN  "Real runtime-node loss at strongest applicable isolation boundary"
GATE_19=UNKNOWN  "Runtime-node-loss detection/reconciliation"
GATE_20=UNKNOWN  "Workload failure (process/container termination)"
GATE_21=UNKNOWN  "Workload failure detection and rescheduling"
GATE_22=UNKNOWN  "Agent failure (runtime component loss)"
GATE_23=UNKNOWN  "Agent failure detection"
GATE_24=UNKNOWN  "Runtime-node restart/rejoin/recovery"
```

### Gate 25–32: Evidence & Verification

```
GATE_25=UNKNOWN  "Network partition injection proves traffic loss"
GATE_26=UNKNOWN  "Network partition detection by production component"
GATE_27=UNKNOWN  "Network partition healing and convergence"
GATE_28=UNKNOWN  "Detection latency measured (not hardcoded)"
GATE_29=UNKNOWN  "Evidence records generated with source SHA binding"
GATE_30=UNKNOWN  "Evidence records cryptographically signed"
GATE_31=UNKNOWN  "Authoritative verifier accepts valid evidence"
GATE_32=UNKNOWN  "Authoritative verifier rejects tampered evidence"
```

---

## Qualification Profiles

Each profile is a set of gates that must pass. Profiles are additive and independent.

### P1_CORE (Any Real Backend)
```
Status: UNKNOWN (blocked)
Prerequisite: 3 real Linux nodes, distinct runtimes, distinct identities

Required Gates: 01–32 (all)
Additional Requirements:
  - Production scheduler exercise
  - Real workload execution
  - Real network partition + recovery
  - Real component failure + recovery
  - Evidence integrity

If PASS: System has baseline dh/v1 conformance
If FAIL: Indicates defect in core mechanisms
```

### P1_QEMU_VM (QEMU/KVM/libvirt Only)
```
Status: NOT_ELIGIBLE (no hypervisor in discovery)

Additional Gates: (VM-specific)
  - Guest OS isolation
  - QEMU hypervisor failure injection
  - VM state persistence
  - VM networking stack

If PASS: QEMU-specific additional evidence
If not-attempted: No impact on P1_CORE

Note: Not required. Backend-neutral approach preferred.
```

### P1_KUBERNETES (Kubernetes Only)
```
Status: NOT_ELIGIBLE (Kubernetes not available in discovery)

Additional Gates: (Kubernetes-specific)
  - Pod lifecycle management
  - StatefulSet with persistent identity
  - NetworkPolicy enforcement
  - kubelet failure + node recovery

If PASS: Kubernetes-specific additional evidence
If not-attempted: No impact on P1_CORE

Note: Only when Kubernetes backend is selected.
```

### P1_MULTIPHYSICAL (≥3 Physical Hosts)
```
Status: NOT_ELIGIBLE (single physical host in discovery environment)

Prerequisite: 3 independently-physical Linux hosts
Additional Gates: (Multi-physical)
  - Physical host power failure injection
  - Physical host network isolation
  - Cross-DC failure recovery (if applicable)

If PASS: Multi-physical failure domain evidence
If not-attempted: No impact on P1_CORE

Note: Only when 3+ physical hosts available.
```

### P1_MULTIOPERATOR (≥2 Independent Operators)
```
Status: NOT_ELIGIBLE (single operator in discovery environment)

Prerequisite: ≥2 independent administrative operators
Additional Gates: (Multi-operator)
  - Operator A → Operator B trust
  - Operator isolation
  - Cross-operator failure scenarios

If PASS: Multi-operator independence evidence
If not-attempted: No impact on P1_CORE

Note: Only when multiple independent operators control nodes.
```

---

## Campaign Execution (When Unblocked)

### Pre-Campaign Checkpoint

Before executing any gate:

```
P1_CANDIDATE_SHA=<exact integrated SHA from main branch>
CAMPAIGN_ID=<timestamp-based ID>
BACKEND=<selected backend name>
TOPOLOGY_DIGEST=<hash of observed node/domain classification>
EVIDENCE_ROOT=<path to campaign evidence directory>

Verify:
  ✓ Source frozen (no commits after candidate SHA)
  ✓ All 32 gates initialized UNKNOWN
  ✓ Runtime backend active and stable
  ✓ 3 nodes healthy and identifiable
  ✓ 6/6 inter-node paths active
  ✓ Evidence collection infrastructure ready
  ✓ Verifier tools available and tested
```

### Gate Execution Principles

1. **Each gate is independent**
   - GATE_01 failure does not prevent GATE_02 execution
   - But logical prerequisites must pass first (e.g., Ed25519 before verification gates)

2. **Every gate produces evidence**
   - Observation timestamp
   - Node identity
   - Action taken
   - Result observed
   - Artifact digests

3. **No hardcoding**
   - All latencies from observed timestamps
   - All identities from actual runtime
   - All results from production paths

4. **Negative controls**
   - Ensure failures are real, not stage-managed
   - Tampered evidence MUST be rejected
   - Missing evidence MUST be detected

5. **Failure = Stop & Diagnose**
   - If GATE_01 fails: Ed25519 implementation broken
   - Diagnose root cause
   - Report exact failure
   - Do NOT skip to next gate

### Evidence Structure (Per-Campaign)

```
validation/p1-qualification/
├── CAMPAIGN_ID/
│   ├── checkpoint-pre-campaign.json
│   ├── topology-observed.json
│   ├── node-identities.json
│   ├── gate-results.jsonl (one line per gate)
│   ├── signed-intent/
│   │   ├── GATE_01.json
│   │   ├── GATE_02.json
│   │   └── ...
│   ├── state-machine/
│   │   ├── GATE_09.json
│   │   └── ...
│   ├── failure-injection/
│   │   ├── GATE_18.json
│   │   └── ...
│   ├── evidence-verification/
│   │   ├── GATE_31-valid.json
│   │   ├── GATE_32-tampered.json
│   │   └── verification-results.json
│   ├── campaign-log.jsonl
│   └── CAMPAIGN_VERDICT.json
```

---

## Qualification Verdict (Decision Matrix)

After all 32 gates execute:

```
if GATE_01–32 = PASS:
    P1_CORE_QUALIFIED = YES
    QUALIFIED = YES (if P1_CORE only)
    
if P1_KUBERNETES_ELIGIBLE and KUBERNETES_GATES = PASS:
    P1_KUBERNETES_QUALIFIED = YES
    QUALIFIED = YES (additive)
    
if P1_MULTIPHYSICAL_ELIGIBLE and MULTIPHYSICAL_GATES = PASS:
    P1_MULTIPHYSICAL_QUALIFIED = YES
    QUALIFIED = YES (additive)

if P1_MULTIOPERATOR_ELIGIBLE and MULTIOPERATOR_GATES = PASS:
    P1_MULTIOPERATOR_QUALIFIED = YES
    QUALIFIED = YES (additive)

if GATE_31 = PASS and GATE_32 = PASS:
    VERIFIED = YES (evidence integrity confirmed)
    
if QUALIFIED = YES and VERIFIED = YES:
    SEALED = YES (final verdict issued)
```

### Possible Verdicts

```
QUALIFIED=YES, VERIFIED=YES, SEALED=YES
→ Decentralized.Host is P1 qualified on selected backend
→ Ready for production hardening phase

QUALIFIED=YES, VERIFIED=NO, SEALED=NO
→ Gates passed but evidence integrity failed
→ Do not issue qualification

QUALIFIED=NO, VERIFIED=YES, SEALED=NO
→ At least one gate failed
→ Diagnose failure
→ Fix source if defect
→ Rerun campaign on new SHA

QUALIFIED=NO, VERIFIED=NO, SEALED=NO
→ Both gates and evidence failed
→ Do not issue qualification
```

---

## Critical Rules (Locked In)

1. **Source SHA is frozen during campaign**
   - No commits to candidate branch
   - All evidence binds to exact SHA
   - If defect found, campaign ends → new SHA → new campaign

2. **Every gate produces real evidence**
   - No simulated results
   - No grep-based verification
   - No hardcoded responses

3. **Profiles are additive, not substitutive**
   - `P1_KUBERNETES_QUALIFIED` ≠ `P1_MULTIPHYSICAL_QUALIFIED`
   - `P1_CORE` ≠ implies VM isolation
   - `P1_CORE` ≠ implies multi-host

4. **Evidence integrity is verified**
   - Original evidence must pass authoritative verifier
   - Tampered evidence must fail verifier
   - Both conditions must be observed

5. **Backend selection is honored**
   - Only test profiles for selected backend
   - Do not attempt VM profiles on containers
   - Do not attempt Kubernetes gates on single-host
   - Classify topology honestly

---

## Dependency Chain

```
┌─────────────────────────────────────────┐
│ SOURCE_SHA=e133fcf (IMPLEMENTED=YES)    │
│ LOCAL_TESTED=YES                         │
│ RUNTIME_TESTED=NO                        │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ RUNTIME_DISCOVERY (blocked: no infra)   │
│ Search for real infrastructure           │
│ Select strongest backend                 │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ REMEDIATION_DIAGNOSTICS (26 phases)     │
│ Prove implementation on real backend     │
│ Collect evidence per specification      │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ EVIDENCE_REVIEW & PR #31 INTEGRATION    │
│ Review raw artifacts                    │
│ Fix any anomalies (new SHA if needed)   │
│ Merge into main                         │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ SOURCE_FREEZE & P1_CANDIDATE_SHA        │
│ Record exact integrated SHA             │
│ Lock source for P1 campaign             │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ P1 QUALIFICATION CAMPAIGN (32 gates)    │
│ Execute on frozen candidate SHA         │
│ All gates UNKNOWN → tested              │
│ Evidence binding to exact SHA           │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ AUTHORITATIVE VERIFICATION              │
│ Verifier accepts valid evidence         │
│ Verifier rejects tampered evidence      │
│ Verdict issued                          │
└────────────┬────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────┐
│ QUALIFIED=YES, VERIFIED=YES, SEALED=YES│
│ System achieves P1 qualification        │
│ Ready for production hardening          │
└─────────────────────────────────────────┘
```

---

## Current State

```
SOURCE_SHA=e133fcf0c3908343ee08b7578aac9c3a53adc081
PHASE=BACKEND_SELECTION_BLOCKED

RUNTIME_DISCOVERY=PASS (documented in RUNTIME-DISCOVERY-CHECKPOINT.md)
BACKEND_SELECTION=BLOCKED (no infrastructure available in CI)
REMEDIATION_DIAGNOSTICS=NOT_STARTED (blocked)

P1_CANDIDATE_SHA=NONE (waiting for remediation diagnostics + integration)
CAMPAIGN_ID=NONE (waiting for source freeze)

GATE_01-32=UNKNOWN (waiting for campaign initialization)

QUALIFIED=NO
VERIFIED=NO
SEALED=NO

NEXT=OPERATOR_PROVISIONS_REAL_INFRASTRUCTURE → return to remediation diagnostics phase
```

---

**Operator**: When real infrastructure is provisioned and backend is selected, return to remediation diagnostics phase. This campaign structure will activate once the candidate SHA is frozen post-integration.

**Do not execute any P1 gate until:**
1. Remediation diagnostics PASS (all 26 phases)
2. Evidence is reviewed (no anomalies)
3. PR #31 is integrated into main
4. Exact integrated SHA is recorded as P1_CANDIDATE_SHA
5. Source is frozen (no further commits)
