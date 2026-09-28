# P1-A01 Qualification Framework

**Complete specification and implementation for P1-LOCAL-VM-A01 qualification**

**Status:** Phase 1 (Baseline) - IN PROGRESS  
**Target Completion:** 2026-09-28 (estimated 21:00-21:30 UTC)  
**Documentation:** Comprehensive (all 4 phases designed)

---

## Quick Start

### Run Full Qualification (Autonomous)
```bash
cd /home/user/Decentralized-
timeout 1800 ./p1-qualification-master.sh 2>&1 | tee /tmp/master-run.log &
```

### Monitor Progress
```bash
tail -f /tmp/master-run.log
```

### Validate All 80 Gates
```bash
./validate-p1-close-gates.sh
# Will show pass/fail status for gates 1-80 as they complete
```

---

## Overview

P1-A01 is a 4-phase, 80-gate qualification framework that comprehensively tests the Decentralized.Host P1 reference implementation:

| Phase | Gates | Focus | Status |
|-------|-------|-------|--------|
| **P1-CLOSE** | 1-32 | Baseline, immediate recovery, failure detection | 🔄 IN PROGRESS |
| **P1-FAILURE** | 33-48 | Persistence, data recovery, evidence immutability | ⏳ PENDING |
| **P1-MESH** | 49-68 | Distributed consensus, Byzantine tolerance | ⏳ PENDING |
| **P1-EVIDENCE** | 69-80 | Cryptographic verification, release readiness | ⏳ PENDING |

**Total:** 80 gates across 4 integrated phases

---

## Phase 1: P1-CLOSE (Gates 1-32)

### What It Tests
- Baseline workload execution (120s with 3 concurrent workloads)
- Immediate failure recovery (workload restart <10s)
- Network partition detection (<45s)
- Process crash injection and detection
- Evidence collection and cryptographic integrity

### Gates 1-8: Baseline Execution
1. **Environment verification** - Tools present, platform compatible
2. **Cluster creation** - 3-node QEMU cluster configured
3. **VM startup** - QEMU TCG emulation (no /dev/kvm available)
4. **Bootstrap nodes** - SSH readiness, 3 retries, 30s between retries
5. **Network qualification** - Mesh connectivity tested
6. **socat installation** - Networking tools on all nodes
7. **ResourceLedger initialization** - Capacity management system ready
8. **Smoke test (15s)** - Simple workload placement and traffic

### Gates 9-16: Workload Placement & Scheduling
9. **Single workload placement** - First-fit scheduler assigns node
10. **ResourceLedger consistency** - Allocations tracked correctly
11. **Three concurrent workloads** - Multiple workloads coexist
12. **Traffic distribution** - Requests balanced across workloads
13. **Baseline duration (120s)** - Full workload execution
14. **Request count aggregation** - Total requests measurable
15. **Workload exit codes (0)** - Clean shutdown, no errors
16. **Workload cleanup/deallocation** - Resources released

### Gates 17-22: Network Partition Failure
17. **Partition injection** - Network isolation simulated
18. **Detection latency (≤45s)** - System detects failure quickly
19. **Workload migration** - Workload moves to healthy node
20. **Restart time (<10s)** - Workload recovers quickly
21. **Traffic continuity (≤5s gap)** - Requests resume smoothly
22. **Node recovery** - Failed node rejoins cluster

### Gates 23-28: Process Crash Injection
23. **Workload crash detection** - System detects process death
24. **Restart time (<10s)** - Workload restarts automatically
25. **Scheduler crash detection** - Scheduler failure detected
26. **Scheduler recovery (<60s)** - Scheduler restarts and recovers state
27. **Simultaneous crashes** - Multiple failures handled
28. **Restart order independence** - Recovery deterministic

### Gates 29-32: Evidence & Verification
29. **Evidence collection** - Logs and state captured for all gates
30. **Cryptographic integrity (Ed25519)** - Evidence signatures verified
31. **Consistency cross-check** - Evidence contradictions detected
32. **Tamper detection** - Modified evidence detected

### Implementation Files
- `p1-qualification-master.sh` - Main orchestrator (9 steps)
- `scripts/bootstrap-nodes.sh` - Node initialization with retries
- `scripts/launch-workload.sh` - Workload placement and execution
- `scripts/run-scheduler.sh` - Scheduler daemon (NEWLY INTEGRATED)
- `scripts/inject-network-partition.sh` - Failure injection
- `scripts/inject-process-crash.sh` - Crash simulation
- `validate-p1-close-gates.sh` - Gate verification

---

## Phase 2: P1-FAILURE (Gates 33-48)

### What It Tests
- Workload data persistence across restarts
- Ledger state consistency after failures
- Evidence log immutability
- Cascading failure handling

### Gates by Category
- **33-36:** Workload persistence & recovery
- **37-40:** Ledger persistence & quorum consensus
- **41-44:** Evidence immutability & tamper detection
- **45-48:** Cascading failures & system stability

### Design Reference
See `P1-FAILURE-A01.md` for complete specification

---

## Phase 3: P1-MESH (Gates 49-68)

### What It Tests
- Distributed mesh networking
- Gossip protocol and state propagation
- Raft consensus and leader election
- Byzantine fault tolerance (1-of-3 dishonest node)
- Network partition healing

### Gates by Category
- **49-52:** Mesh connectivity verification
- **53-56:** Gossip protocol under load
- **57-60:** Consensus and leader election
- **61-64:** Byzantine fault tolerance
- **65-68:** Network partition recovery

### Design Reference
See `P1-MESH-A01.md` for complete specification

---

## Phase 4: P1-EVIDENCE (Gates 69-80)

### What It Tests
- Evidence collection and archival
- Cryptographic verification (Ed25519)
- External verifier compatibility
- Release certificate generation

### Gates by Category
- **69-72:** Evidence collection & archival
- **73-76:** Cryptographic verification
- **77-80:** External verification & release

### Design Reference
See `P1-EVIDENCE-A01.md` for complete specification

---

## Key Concepts

### Semantic Gate Status
- **PASS:** Condition met, evidence supports result, metric within threshold
- **FAIL:** Condition not met, evidence contradicts requirement
- **BLOCKED:** Cannot evaluate because prerequisite gate failed
- **UNKNOWN:** Result indeterminate, evidence missing/ambiguous

### Critical Definitions
- **SMOKE_PASS ≠ P1_QUALIFIED:** Smoke test is gate 8 only; qualification is all 80 gates
- **BASELINE_PASS ≠ P1_QUALIFIED:** Baseline is gates 1-9 only; phases 2-4 also required
- **NO GATE WEAKENING:** Gates never weakened to make tests pass
- **REAL FAILURES:** No simulated/stubbed dependencies; actual crashes/partitions

### Evidence Requirements
- **Immutable:** Once committed to git, cannot be modified
- **Signed:** All artifacts signed with Ed25519 private key
- **Portable:** Self-contained, no system-specific paths
- **Auditable:** Third parties can verify with public key only

---

## Execution Timeline

### Phase 1 (P1-CLOSE): ~60-75 minutes
- Cluster setup: 10-15 min (QEMU TCG slowness)
- Bootstrap: 25-30 min (with retries)
- Baseline: 120s
- Remaining gates: 10 min

### Phase 2 (P1-FAILURE): ~45-60 minutes
- Baseline with persistence: 120s
- Single node failure recovery: 30 min
- Ledger consensus recovery: 15 min
- Evidence verification: 5-10 min

### Phase 3 (P1-MESH): ~30-45 minutes
- Mesh initialization: 5 min
- Consensus testing: 15 min
- Byzantine tolerance: 10 min
- Partition healing: 10 min

### Phase 4 (P1-EVIDENCE): ~15-20 minutes
- Evidence collection: 5 min
- Signature verification: 5 min
- Release certificate: 5 min

**Total Estimated Time:** 150-200 minutes (2.5-3.3 hours)

---

## Repository Structure

```
validation/local-vm/
├── README-P1-A01-QUALIFICATION.md     ← This file
├── P1-CLOSE-GATES.md                  ← Phase 1 spec
├── P1-FAILURE-A01.md                  ← Phase 2 design
├── P1-MESH-A01.md                     ← Phase 3 design
├── P1-EVIDENCE-A01.md                 ← Phase 4 design
│
├── scripts/
│   ├── p1-qualification-master.sh      ← Main orchestrator
│   ├── create-vm-cluster.sh            ← Cluster creation
│   ├── start-cluster.sh                ← VM startup (TCG support)
│   ├── bootstrap-nodes.sh              ← Node initialization
│   ├── qualify-network.sh              ← Network verification
│   ├── init-resourceledger.sh          ← ResourceLedger init
│   ├── run-scheduler.sh                ← Scheduler daemon (NEW)
│   ├── launch-workload.sh              ← Workload placement
│   ├── request-placement.sh            ← Placement API client
│   ├── inject-network-partition.sh     ← Network failure
│   ├── inject-process-crash.sh         ← Crash simulation
│   └── verify-p1-close-gates.sh        ← Gate analysis
│
├── state/
│   ├── cluster.json                    ← Node configuration
│   ├── resourceledger.json             ← Capacity state
│   ├── placement-result.json           ← Scheduler response
│   ├── placement-manifest.json         ← All placements
│   ├── workload-logs/                  ← Workload outputs
│   └── network-qualification.json      ← Network evidence
│
└── evidence/
    ├── P1-CLOSE-A01/                   ← Phase 1 evidence
    ├── P1-FAILURE-A01/                 ← Phase 2 evidence
    ├── P1-MESH-A01/                    ← Phase 3 evidence
    └── P1-A01-QUALIFICATION-REPORT.md  ← Final report
```

---

## Running Qualification Phases

### Automatic (Recommended)
```bash
cd /home/user/Decentralized-
# Runs all phases autonomously, no manual intervention
timeout 3600 ./p1-qualification-master.sh 2>&1 | tee /tmp/master-run.log &
```

### Manual (Debug/Testing)
```bash
# Run individual steps
./validation/local-vm/scripts/create-vm-cluster.sh
./validation/local-vm/scripts/start-cluster.sh
./validation/local-vm/scripts/bootstrap-nodes.sh
# ... etc for each step

# Verify gates
./validate-p1-close-gates.sh
```

---

## Troubleshooting

### Slow Bootstrap
- **Cause:** QEMU TCG emulation (10-100x slower than KVM)
- **Mitigation:** Bootstrap retry logic (3 attempts, 30s between)
- **Timeout:** 120s per attempt, 30s wait = 3.5 min total for 3 attempts
- **Symptom:** Bootstrap attempt 1 fails, attempt 2 succeeds (normal)

### Placement Timeout
- **Cause:** Scheduler daemon not running (FIXED)
- **Error:** "Placement request timed out after 30s"
- **Fix:** Master script Step 7.5 now starts scheduler
- **Status:** ✅ Deployed in current version

### Missing Evidence
- **Cause:** Gate failed, preventing dependent gates
- **Action:** Check logs in `/tmp/p1-qualification-*/`
- **Recovery:** Fix root cause, restart from clean state
- **Data:** Previous evidence preserved in `/evidence/`

### Network Issues
- **Cause:** Cloud environment with restricted network
- **Symptom:** SSH timeouts, can't reach nodes
- **Action:** Check SSH key permissions, firewall rules
- **Fallback:** Use TCG emulation (already enabled)

---

## Success Criteria

- ✅ All 32 gates in P1-CLOSE achieve PASS status
- ✅ All 16 gates in P1-FAILURE achieve PASS status
- ✅ All 20 gates in P1-MESH achieve PASS status
- ✅ All 12 gates in P1-EVIDENCE achieve PASS status
- ✅ Evidence archived with cryptographic signatures
- ✅ External verifier can validate evidence with public key
- ✅ Release certificate generated and signed

---

## Next Steps After P1-A01

1. **Tag Release:** `git tag v1.0-P1-QUALIFIED`
2. **Create Release Note:** Document all 80 gates, metrics, results
3. **External Audit:** Third parties verify evidence independently
4. **Begin P2-MULTIHOST:** Multi-machine deployment (5+ nodes)
5. **Production Planning:** Deployment strategy, monitoring, updates

---

## Key Contacts & References

- **Mandate:** See `/root/.claude/projects/.../mandate.txt`
- **Branch:** `claude/friendly-gauss-kfxoc2`
- **Status:** `QUALIFICATION-STATUS-P1-CLOSE-A01.md`
- **Implementation:** All scripts in `/validation/local-vm/scripts/`

---

## Notes

- **Autonomous Execution:** All phases run without manual intervention
- **Real Environment:** QEMU TCG emulation (cloud container, no /dev/kvm)
- **Comprehensive Testing:** 80 gates covering baseline, failure, consensus, evidence
- **Cryptographic Verification:** Ed25519 signatures on all evidence
- **Permanent Record:** Evidence committed to git, cannot be modified post-hoc

---

**Last Updated:** 2026-09-28 20:35 UTC  
**Version:** 1.0 (P1-A01 Framework)  
**Status:** Ready for autonomous execution through all 4 phases

