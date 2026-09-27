# Decentralized.Host — Stream 0 Reconciliation Report
**Date:** 2026-09-27  
**Base Branch:** main @ bbd9004 (merged PR #17)  
**Current Session Branch:** claude/friendly-gauss-kfxoc2 @ 2692b26 (24 commits ahead)

---

## Executive Summary

The repository has successfully verified the **sovereign node foundation** (NODE-A01) and the **HA control-plane core** (Raft consensus via P1-NODE-FLEET-A01 gates 1-22). However:

1. **Gates 1-22 are unmerged** — they exist on a feature branch and have not been integrated to main
2. **Main is at Wave 3 Phase 3** (resource reservations, hardware discovery, secrets) but evidence drift exists between claimed and verified
3. **Critical P0 blockers remain unresolved** — secrets subsystem, PV1 multi-machine, UNTRUSTED isolation, public ACME, root key custody, SBOM/signed releases
4. **No marketplace/metering infrastructure exists** — P2 is blocked waiting for foundation

**Status:** P0 is **PARTIAL** (not production-ready for multi-node private cloud or any marketplace work).

---

## Current Branch State

### Main Branch (bbd9004)
```
Merge commit: PR #17 - Wave 3 Phase 3: Resource Reservations
Latest features:
  - NODE-A01 verified (sovereign node foundation)
  - HA control-plane (Raft, mTLS)
  - Fleet inventory and capacity aggregation
  - Resource reservations (claimed in Wave 3 PR #16, #17)
  - Hardware discovery enhancements
  - Secrets subsystem (claimed in Wave 3 PR #15)
  - Command Centre Wave 1 (live)
  - 17 chaos scenarios
  - Audit ledger + evidence chain
```

### Feature Branch (claude/friendly-gauss-kfxoc2)
```
Latest commit: 2692b26 - Gate 22: Comprehensive Integration and Advanced Scenarios
Content: P1-NODE-FLEET-A01 gates 1-22
  - Raft consensus verification (leader election, log replication, snapshots, membership changes)
  - 10-phase comprehensive integration test
  - All tests passing with -race flag (101.6s full suite)
  - Comprehensive documentation for each gate
Diverge point: d8d562d0825b48a84f05cbcbb8faa943ea1e3537
Commits ahead of main: 24
Status: UNMERGED (waiting for decision on where these tests live)
```

---

## Evidence Reconciliation vs GAP_MATRIX

### What Main Claims vs What Is Verified

| Area | Main Claim | Verified | Status | Gap |
|------|-----------|----------|--------|-----|
| **§39 Secrets** | Implemented (PR #15) | Manifest env in control-plane state | **GAP** | Needs subsystem before production secrets; no encrypted DEK, no materialization control |
| **§8 Hardware Discovery** | Enhanced (PR #16) | Memory, CPU, cores, disks, GPUs, swap, uptime measured; Docker runtime detected | **PARTIAL** | Missing: NICs, NAT type, public reachability (P0-1 blocker), container runtimes beyond Docker |
| **§9 Resource Model** | Implemented (PR #17) | MODEL A spec exists (`resource-semantics.md`); reservation/allocation semantics defined | **PARTIAL** | Code not found on main; spec written but implementation unverified |
| **§10-11 Owner Kill Switch** | Claimed in resource reserve | `freeze`, `drain`, policy exist | **PARTIAL** | No CORDONED state; no owner-reserved capacity tracking in Ledger |
| **§12 Node Lifecycle** | CORDONED state | pending/ready/draining/revoked/lost exist | **MISSING** | CORDONED not distinct state; needed for owner reserve |
| **§14 Workload Security** | RUNTIME-P0-A01 qualified | sandbox + PRIVATE/RESTRICTED verified | **VERIFIED** | UNTRUSTED still refused (no microVM/gVisor); not production-ready for hostile/marketplace workloads |
| **§16 Build Service** | — | Prebuilt artifacts only | **MISSING** | No Git/Dockerfile build; no SBOM; no builder identity binding |
| **§17 Provenance** | — | Digests + root attestation | **PARTIAL** | No SBOM, no builder identity, no source commit binding, no signed releases |
| **HA Control Plane** | Raft, mTLS, M5 | Raft verified in 22 gates | **VERIFIED** | Gates on unmerged branch; need decision on merge into main |
| **Audit Ledger** | Hash chain + signed checkpoints | Evidence records exist | **VERIFIED** | Works as designed; no integration with metering yet |

### Evidence Drift Details

1. **Secrets (§39) — CRITICAL BLOCKER**
   - **Claim:** Implemented in PR #15  
   - **Reality:** Manifest env values stored in control-plane state (plaintext in Raft replicated snapshots)
   - **Issue:** No encryption, no DEK derivation, no materialization control
   - **Evidence:** `CURRENT_STATE.md` line 85 lists as MISSING
   - **Action Required:** Implement sovereign root key → KEK → per-secret DEK architecture before any non-owner workload

2. **Resource Reservations (§9, §10) — CRITICAL BLOCKER**
   - **Claim:** Implemented in Wave 3 PR #17  
   - **Reality:** Specification written (`resource-semantics.md` MODEL A); implementation unverified  
   - **Code Status:** Not confirmed on main; no test evidence of reserve/allocate operations
   - **Action Required:** Verify implementation on main; implement owner-reserved capacity as first-class Ledger record

3. **Hardware Discovery (§8) — P0-1 BLOCKER**
   - **Claim:** Enhanced in PR #16  
   - **Reality:** CPU/memory/storage measured; NICs missing, NAT type missing
   - **Impact:** Cannot do multi-machine validation (PV1) without NIC facts and NAT detection
   - **Action Required:** Implement NIC discovery, NAT type detection via external observer

4. **UNTRUSTED Isolation (§14) — MARKETPLACE BLOCKER**
   - **Claim:** RUNTIME-P0-A01 qualified  
   - **Reality:** Only PRIVATE/RESTRICTED profiles supported; UNTRUSTED (microVM/gVisor) not implemented
   - **Impact:** Cannot run hostile/marketplace workloads; isolation only sufficient for owner workloads
   - **Action Required:** Defer until P2 or implement gVisor/Firecracker

5. **Raft Gates 1-22 (P1-NODE-FLEET-A01)**
   - **Claim:** None yet (on unmerged branch)  
   - **Reality:** 22 comprehensive integration gates, all passing with -race flag
   - **Impact:** Control-plane infrastructure is verified but not integrated
   - **Decision Required:** Merge to main or keep as reference qualification suite?

---

## Unmerged Branches

```
Branch: claude/p1-scheduler-a01
Commits: Fleet inventory, placement, scheduler logic
Status: Unmerged (needs P1 foundation)

Branch: claude/p1-node-fleet-a01  
Commits: Scheduler/placement enhancements
Status: Unmerged (needs P1 foundation)

Branch: claude/friendly-gauss-kfxoc2 (current session)
Commits: P1-NODE-FLEET-A01 gates 1-22 (Raft consensus verification)
Status: Unmerged (needs decision on integration)
```

---

## P0 Production Blockers — Actual Status

### Must-Close Before P1/P2

| Blocker | Status | Blocker | Evidence | Next Step |
|---------|--------|---------|----------|-----------|
| **Secrets (§39)** | GAP | 🔴 CRITICAL | `CURRENT_STATE.md` line 85 | Implement sovereign root → DEK subsystem |
| **Owner Reserve (§9-11)** | PARTIAL | 🔴 CRITICAL | `resource-semantics.md` spec exists, code unverified | Verify/implement reserve/allocate on Ledger |
| **CORDONED State (§12)** | MISSING | 🔴 CRITICAL | No distinct state in `pkg/control` | Add CORDONED to node lifecycle FSM |
| **PV1 Multi-machine (§8, P0-1)** | LIMITED | 🔴 CRITICAL | WireGuard mesh single-machine; no cross-NAT evidence | Real >=2-node topology, NAT hole-punch or relay |
| **UNTRUSTED Isolation (§14)** | MISSING | 🟠 BLOCKS MARKETPLACE | RUNTIME-P0-A01 only PRIVATE/RESTRICTED | Defer to P2 or implement gVisor/Firecracker |
| **Public ACME (§18)** | LIMITED | 🟠 PRODUCTION | Pebble only; no public CA | Integrate production ACME endpoint |
| **Root Key Custody (§5, P0-8)** | GAP | 🟠 PRODUCTION | Root material not documented | Specify bootstrap, unlock, rotation, backup, restore |
| **SBOM & Signed Releases (§17, P0-9)** | PARTIAL | 🟠 SECURITY | No SBOM, no build provenance | Generate SBOM, sign builds, version attestation |
| **Rate Limiting (§84)** | LIMITED | 🟡 OPERABILITY | No BFF rate limiting | Add rate limiting to API |

### Can Defer to P1

| Feature | Status | Notes |
|---------|--------|-------|
| Time-series metrics (§40, P1-6) | PARTIAL | Point-in-time facts work; no time series |
| DID identities (P1-5) | MISSING | Needed for federation, not critical for local P0 |
| Contribution policy per workload (§10) | PARTIAL | Freeze/drain exist; per-origin limits missing |
| Erasure coding (§35) | MISSING | Replicas work; erasure coding deferred |
| Edge caching (§33) | MISSING | L7 health routing works; caching deferred |

---

## Recommended Immediate Action (Stream 0 Closure)

### Phase 1A: Reconcile Main (1-2 days)
1. Verify resource reservations code on main @ bbd9004
2. Verify secrets claim in Wave 3 PR #15 vs actual implementation
3. Document what is actually LIVE vs SIMULATED vs MISSING on main
4. Decide: Merge P1-NODE-FLEET-A01 gates 1-22 to main, or keep as reference suite

### Phase 1B: Fix Critical P0 Blockers (2-3 weeks)
**In this order:**

1. **Secrets Subsystem (§39)** — 3-4 days
   - Implement sovereign root key (not stored in Raft)
   - DEK derivation per secret/version
   - Ciphertext in control-plane state only
   - Runtime materialization to memory only
   - Test: plaintext never in logs, snapshots, artifacts

2. **Owner Reserve + CORDONED (§9-11) — 2-3 days**
   - Add CORDONED state to node FSM
   - Implement ResourceReservation as Ledger record
   - Verify MODEL A constraints
   - Test: reserve/allocate/release operations
   - Display on Command Centre

3. **Root Key Custody (§5, P0-8) — 2-3 days**
   - Document: bootstrap, unlock, restart, rotation, backup, restore, loss recovery
   - Implement: Bootstrap prompt, unlock on start, key rotation ceremony
   - Test: Restart with unlocked/locked scenarios

4. **PV1 Multi-machine Validation (§8, P0-1) — 3-4 days**
   - Use real >=2 independent nodes (not loopback)
   - Measure: NIC facts, IP routes, NAT type
   - Test: Mesh formation, cross-NAT reachability
   - Qualify: `PV1-MULTI-MACHINE-A01` evidence

### Phase 2: Build Foundation for P1/P2 (Parallel)
5. **Signed Metering (§24, P2 prerequisite) — 2-3 weeks**
   - UsageRecord type (resource, quantity, interval, signer, nonce, signature)
   - Agent → control-plane ingestion
   - Prevention: replay, double-count, rollback
   - Test: 100+ usage records, cross-verify signatures

---

## For This Session

**Immediate Task:**

1. **Merge Decision:** Should P1-NODE-FLEET-A01 gates 1-22 be merged to main?
   - **Option A:** Merge to main as permanent control-plane qualification suite
   - **Option B:** Keep as reference; extract only essential Raft tests to main
   - **Option C:** Archive on a release/reference branch

2. **Evidence Reconciliation:** 
   - Verify resource reservations code on main
   - Verify secrets implementation on main
   - Update GAP_MATRIX with findings

3. **Blocking Decision:**
   - Start with secrets and owner-reserve in parallel
   - Don't start P1/P2 work until P0 blockers are unblocked

---

## Canonical Sources

- `docs/BLUEPRINT.md` — spec v1.2
- `docs/CURRENT_STATE.md` — what's live today (2026-09-26)
- `docs/GAP_MATRIX.md` — spec gaps
- `docs/resource-semantics.md` — resource model (MODEL A)
- `CURRENT_STATE.md` § Broken/Defects — known issues
- Evidence: `evidence/` directory + validation records

No production deployment or P2 marketplace work until P0 blockers close.

---

**Report Status:** READY FOR REVIEW  
**Next Step:** Merge decision + Phase 1A execution plan
