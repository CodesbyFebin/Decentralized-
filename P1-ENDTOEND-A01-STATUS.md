# P1-ENDTOEND-A01: Qualification Test — Status & Dependencies

**Current Phase**: P1 UI Complete → P1-E2E Qualification Planning

---

## What We Have

### ✅ P1 UI Surfaces (Completed)

All 6 P1 command-centre surfaces shipped in PR #25:

1. **Evidence View** — Resource/node facts with verification
2. **Deploy View** — Workload placement lifecycle
3. **Storage View** — Volume replication with proof
4. **Failure Domains & Placement** — Hierarchy and constraints
5. **Distributed Ingress & Service Discovery** — Service IP and LB
6. **Copilot** — Audit trails and safety gates

**Status**: Merged to main (PR #25 ready for final review)

### ✅ Test Planning Documents (Completed)

1. **P1-ENDTOEND-A01-TESTPLAN.md** — 5 scenarios, evidence collection, success criteria
2. **P1-ENDTOEND-A01-INFRASTRUCTURE.md** — Node setup, failure injection, helpers
3. **P1-ENDTOEND-A01-STATUS.md** — This document

**Status**: Documented and locked

---

## What We Need

### 1. Backend Implementation (Control Plane & Node Agent)

**Required before test execution**:

| Component | Status | Dependency |
|-----------|--------|------------|
| DHP control plane core | ? | Raft consensus, node management |
| Workload scheduling | ? | Placement algorithm, constraints |
| Evidence signing | ? | Ed25519 key management, record format |
| Failure detection | ? | Gossip/membership protocol |
| Reconciliation engine | ? | Reschedule logic on node failure |
| Node agent | ? | Resource reporting, evidence signing |
| Storage subsystem | ? | Volume replication, replica tracking |
| Service discovery | ? | IP allocation, endpoint health |

**Blocking Question**: Which of these are implemented vs. still being developed?

### 2. Infrastructure

**Required before test**:

- [ ] 5 real Linux nodes (or VMs with real networking)
- [ ] Network connectivity (1-50ms latency, <0.1% packet loss)
- [ ] NTP configured (±100ms clock sync)
- [ ] Centralized logging (syslog to observer node)
- [ ] Failure injection tools (iptables, disk fill, etc.)

**Timeline**: 2-4 hours to set up (if nodes available)

### 3. Test Tooling

**Required before test**:

- [ ] `dhp-ctl` CLI for control plane operations
- [ ] `dhp-ctl evidence list/dump` for evidence collection
- [ ] `dhp-validator` for offline signature verification
- [ ] Test orchestrator for scenario automation
- [ ] Metrics aggregation and reporting

**Timeline**: 1-2 hours (most can be shell scripts initially)

### 4. Upstream Research Alignment

For P1 infrastructure insights, study:

- **Nomad** — scheduling concepts (placement, constraints, reconciliation)
- **memberlist** — gossip membership and failure detection
- **Raft** — consensus (already likely in use)
- **CoreDNS** — service discovery patterns
- **Prometheus** — metrics collection patterns

These are reference-only for P1 (don't adopt wholesale).

---

## Gate: P1-ENDTOEND-A01 Completion

### Success Criteria

```
PASS: All 5 scenarios execute without workload loss
      All evidence signatures verify (100%)
      Failure detection <30 seconds
      Workload reschedule <120 seconds
      No-mock gate passes (zero mock data)
      Signed report generated

FAIL: Any scenario fails any criterion
      Any workload crashes/lost
      Evidence validation fails
      Timings exceed thresholds
      Mock data detected
```

### Expected Outcome

**If PASS**:
- P1 (Distributed Private Cloud) sealed as production-ready
- PR merged to main
- P2 (First-Party DePIN Marketplace) research and implementation begins
- Upstream research corpus (Akash, Golem, Bacalhau, etc.) unlocked

**If FAIL**:
- Root cause analysis
- Fixes implemented
- Re-test required (1-3 day iteration)

---

## Dependency Chain: What Blocks What

```
P1-E2E TEST EXECUTION
  ├─ Backend Implementation Ready
  │   ├─ Control plane (Raft, placement, reconciliation)
  │   ├─ Node agent (resources, signing, failure detection)
  │   ├─ Evidence subsystem (signing, storage, validation)
  │   └─ Storage subsystem (replication, recovery)
  │
  ├─ Infrastructure Available
  │   ├─ 5 Linux nodes (real network)
  │   ├─ Failure injection mechanisms
  │   ├─ Centralized logging
  │   └─ Monitoring/metrics
  │
  └─ Test Tooling
      ├─ CLI tools (dhp-ctl, dhp-node)
      ├─ Evidence validator
      ├─ Test orchestrator
      └─ Report generator

↓

P1-ENDTOEND-A01 REPORT SIGNED
  → P1 SEALED
    → P2 Research Begins
      → P2-E2E (later, independent operators)
```

---

## Next Actions (Decision Points)

### Immediate (This Week)

**Question 1**: What's the status of backend implementation?
- [ ] Core control plane ready for test
- [ ] Node agent ready for test
- [ ] Evidence system implemented
- [ ] Failure detection working

**Action**: Code review/verification of control plane readiness

### Near-Term (1-2 Weeks)

**Question 2**: Can we provision 5 test nodes?
- [ ] Cloud infrastructure available (AWS/GCP/bare metal)
- [ ] Budget approved for test environment
- [ ] Network configured (isolated test VLAN)

**Action**: Infrastructure provisioning request

**Question 3**: Can we build test tooling?
- [ ] CLI tools already exist for control plane
- [ ] Evidence validator can be built from evidence schema
- [ ] Test orchestrator can be shell scripts + Python

**Action**: Assign tooling development (1-2 days)

### Test Execution (3-4 Weeks)

Once all dependencies cleared:

1. Provision infrastructure (2 hours)
2. Install and configure (2 hours)
3. Baseline validation (1 hour)
4. Run Scenarios 1-5 (3 hours)
5. Evidence verification (2 hours)
6. Report generation (1 hour)

**Total**: 8 hours of active test time + 1-2 hours analysis

---

## Critical Success Factors

### 1. Evidence System Must Be Complete

If evidence signing isn't implemented, P1-E2E cannot pass the no-mock gate.

**Verify**:
```bash
curl http://cp-0:8080/api/v1/evidence | jq '.[] | {id, signature, signer}'
```

Expected: Signed EvidenceRecords with valid Ed25519 signatures.

### 2. Failure Detection Must Work

If node failure isn't detected within 30 seconds, reconciliation can't trigger.

**Verify** (manually):
- Stop node agent on provider-b
- Wait and check: does control plane detect DEAD state?
- Latency <30 seconds required

### 3. No-Mock Data

If production code contains mock/hardcoded data, test doesn't prove real distributed operation.

**Verify**:
```bash
grep -r "mock\|Mock\|MOCK\|hardcoded\|placeholder" \
  control-plane/src node-agent/src \
  --include="*.go" \
  --exclude-dir=test
```

Expected: Zero matches.

---

## Risk Register

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Backend not ready | Test blocked | Code review now, identify gaps early |
| Node failure detection >30s | Test fails | Tune gossip timeout, pre-test measurement |
| Evidence signing errors | Test fails | Unit tests on signature verification |
| Network instability | Test unreliable | Use dedicated isolated test network |
| Mock data leaks through | Test fails validation | Grep enforcement in CI |
| Clock skew >100ms | Timing measurements wrong | NTP setup, pre-test verification |

---

## Success Metrics Dashboard

```
P1 COMPLETION ROADMAP:

┌─────────────────────────────────────────────────┐
│ P1 UI Surfaces         [████████████] 100%     │  ✅ DONE
│ Test Plan              [████████████] 100%     │  ✅ DONE
│ Infrastructure Plan    [████████████] 100%     │  ✅ DONE
│ Backend Implementation [??????????  ] 30-80%?  │  ? VERIFY
│ Test Tooling           [???????     ] 0-50%?   │  ? ASSESS
│ Infrastructure Ready   [           ] 0%        │  ⏳ PENDING
│ Test Execution        [           ] 0%        │  ⏳ PENDING
│ Report & Seal         [           ] 0%        │  ⏳ PENDING
└─────────────────────────────────────────────────┘

Blocker: Backend Implementation Status (UNKNOWN)
Next: Code review + infrastructure assessment
Target: P1-ENDTOEND-A01 execution within 3-4 weeks
```

---

## Documentation Reference

| Document | Purpose | Status |
|----------|---------|--------|
| **PHASE-BOUNDARIES.md** | Architecture constraints, phase gates | ✅ Source of truth |
| **P1-ENDTOEND-A01-TESTPLAN.md** | Test scenarios, evidence collection, criteria | ✅ Complete |
| **P1-ENDTOEND-A01-INFRASTRUCTURE.md** | Node setup, installation, helpers | ✅ Complete |
| **UI_MERGE_P1_STRATEGY.md** | P1 surfaces, UI roadmap | ✅ Earlier session |
| **PR #25** | P1 UI surfaces code | ✅ Ready for merge |

---

## Handoff Notes

This qualification test plan is **locked and ready for execution** pending:

1. **Backend implementation verification** — Code review of control plane readiness
2. **Infrastructure provisioning** — Allocation of 5 test nodes
3. **Test tooling buildout** — 1-2 days to assemble helpers

Once those are cleared, P1-ENDTOEND-A01 can run.

**Timeline to P1 Seal**: 3-4 weeks from now (test + analysis + report).

**Then**: P2 (First-Party DePIN Marketplace) research and implementation begins.

