# P1-ENDTOEND-A01: Distributed Private Cloud Qualification Test

**Objective**: Validate P1 (Distributed Private Cloud) readiness by executing real distributed workload placement, failure detection, and reconciliation across multiple independently controlled Linux nodes with signed evidence for all state changes.

**Gate Criteria** (from PHASE-BOUNDARIES.md):
- ✓ Real distributed test with real node failure/recovery
- ✓ Multiple independently controlled Linux nodes (3+)
- ✓ Signed evidence for all state changes
- ✓ No-mock gate passes (zero mock data)
- ✓ Measured restoration time on failure

---

## Test Infrastructure

### Node Requirements

| Role | Count | Specs | Purpose |
|------|-------|-------|---------|
| Control Plane | 1 | 4+ CPU, 8GB RAM, 50GB disk | DHP control plane, Raft consensus |
| Provider Nodes | 3 | 2+ CPU, 4GB RAM, 30GB disk | Workload execution, evidence signing |
| Observer Node | 1 | 2+ CPU, 2GB RAM, 20GB disk | Independent evidence verification |
| **Total** | **5** | | Real network, not localhost |

### Environment Constraints
- **Network**: Real TCP/IP, NOT loopback (127.0.0.1)
- **Isolation**: Independent Linux instances (KVM/bare metal/cloud VMs acceptable)
- **Latency**: 1-50ms inter-node (realistic LAN)
- **Failure induction**: Controllable via `iptables`, process kill, disk full, CPU throttle
- **Logging**: Centralized collection (syslog/journal to observer node)
- **Clock**: NTP synchronized (±100ms max skew)

### Node Configuration

Each provider node runs:
```
control-plane node daemon
  ├─ DHP agent (resource declaration, evidence signing)
  ├─ containerd + runc (workload execution)
  ├─ Gossip membership (memberlist or DHP native)
  ├─ Evidence ledger (local append-only log)
  └─ Metrics collector (CPU, memory, network, uptime)

Observer node:
  ├─ Syslog receiver (all node logs)
  ├─ Evidence validator (offline verification of all signatures)
  ├─ Metrics aggregator
  └─ Test orchestrator (failure injection, scenario control)
```

---

## Test Scenarios

### Scenario 1: Baseline — Normal Operation (30 min)

**Goal**: Establish healthy cluster state, verify all subsystems operational.

**Setup**:
1. Start control plane, join 3 provider nodes to cluster
2. Verify Raft consensus (3+ nodes, leader elected)
3. Each provider node publishes resource capacity:
   - CPU: millicores
   - Memory: bytes
   - Storage: bytes
   - Evidence: signed by node key, timestamped

**Workloads**:
- Deploy 5 simple workloads (nginx, echo-server, sleep-daemon, cpu-burner, mem-consumer)
- 2 workloads on node-a, 2 on node-b, 1 on node-c (spread placement)
- Verify placement with `GET /apps/{id}/deployments` on each node

**Evidence Collected**:
1. **Placement Evidence**: Control plane signs placement decision for each workload
   - workload ID, desired node, placement timestamp, control-plane key
2. **Execution Evidence**: Provider node signs execution proof
   - workload ID, container runtime (containerd) proof, process ID, start timestamp, node key
3. **Resource Usage Evidence**: Provider node signs resource measurements
   - CPU: millicores × seconds
   - Memory: bytes × seconds (peak observed)
   - Timestamp, measurement interval, node key

**Success Criteria**:
- All 5 workloads reach RUNNING state within 60 seconds
- All 3 provider nodes report LIVE resource state (freshness LIVE)
- Evidence ledger contains ≥15 signed records (3 evidence types × 5 workloads)
- No errors in control plane logs (filtered for INFO and above)
- Each evidence record verifies correctly (signature check passes)

---

### Scenario 2: Node Failure — Provider Goes Offline (60 min)

**Goal**: Detect node failure, trigger workload reconciliation, measure recovery time.

**Setup**:
- Use baseline state from Scenario 1 (5 workloads running)
- Establish baseline metrics:
  - Cluster consensus latency (Raft heartbeat RTT)
  - Node failure detection latency (gossip timeout)
  - Workload reconciliation time

**Failure Injection**:
1. **T=0s**: Stop all network traffic to node-b (iptables DROP all)
   - Simulate hard failure (not graceful shutdown)
   - Workloads on node-b become unreachable
2. **T=5s**: Gossip detects node-b offline (expected: <5s)
3. **T=30s**: Control plane marks node-b as DEAD (reconciliation trigger)
4. **T=60s**: Workloads originally on node-b should be rescheduled to remaining nodes

**Expected State Transitions**:

```
PLACEMENT (Desired)           OBSERVED                        EVIDENCE
node-b: workload-1       →    UNKNOWN (10s)      →    FAILED (signed by node-b)
                         →    UNREACHABLE (30s)  →    FAILURE_RECOVERY (control plane)
node-a: rescheduled-1    →    SCHEDULED (40s)    →    PLACEMENT (new)
                         →    RUNNING (50s)      →    EXECUTION (node-a signs)
```

**Evidence Collected**:
1. **Failure Detection Evidence**:
   - node-b last heartbeat timestamp
   - Detection time (gossip timeout or RPC fail)
   - Signed by control plane
2. **Reconciliation Evidence**:
   - Original placement (workload → node-b)
   - Failure trigger (node-b DEAD)
   - New placement (workload → node-a)
   - All signed by control plane with sequence numbers
3. **Resource Release Evidence**:
   - node-b resource reservation revocation
   - Timestamp, control-plane signature
4. **Recovery Execution Evidence**:
   - New workload execution on node-a
   - Start timestamp, node-a signature

**Measurement Points**:
- `failure_detection_latency` = T(gossip detects) - T(iptables DROP)
  - Target: <5 seconds
  - Acceptable: <10 seconds
  - Failure: >30 seconds
- `reconciliation_start_latency` = T(control plane marks DEAD) - T(detection)
  - Target: <5 seconds
  - Acceptable: <15 seconds
- `workload_reschedule_latency` = T(new workload RUNNING) - T(detection)
  - Target: <60 seconds
  - Acceptable: <120 seconds
  - Failure: >300 seconds (unacceptable delay)

**Success Criteria**:
- Failure detected within 30 seconds ✓
- Workloads rescheduled within 120 seconds ✓
- New placements execute successfully on remaining nodes ✓
- All state changes signed with valid cryptographic proof ✓
- Evidence chain unbroken (no gaps in sequence numbers) ✓
- Observer node validates all signatures (offline verification passes) ✓

---

### Scenario 3: Partial Network Partition (45 min)

**Goal**: Test Byzantine resilience — partial connectivity loss (not complete isolation).

**Setup**:
- Restore node-b to cluster (recovery from Scenario 2)
- Allow cluster to stabilize (5 min)

**Failure Injection**:
1. **T=0s**: Degrade network between control plane ↔ node-c
   - Simulate lossy WAN: 50% packet loss, 500ms latency
   - node-c can still reach node-a and node-b (local network)
   - Control plane loses RPC connectivity to node-c
2. **Expected**: Cluster detects node-c as STALE or UNKNOWN (not DEAD, since other nodes see it)

**Expected Behavior**:
- Control plane cannot reach node-c directly → state becomes STALE
- node-a and node-b report node-c LIVE (they can reach it)
- Decision: Hold placement, don't reschedule yet (quorum sees it alive)
- If workload crashes on node-c, new placement waits for partition heal

**Evidence Collected**:
1. **State Divergence Evidence**:
   - Control plane: node-c UNKNOWN/STALE
   - node-a/node-b: node-c LIVE
   - All signed by respective signers with timestamp
2. **Partition Healing Evidence**:
   - Time partition healed
   - State reconciliation (if diverged)
   - Final consensus state

**Success Criteria**:
- Cluster remains operational (majority quorum holds) ✓
- No workloads incorrectly rescheduled during partition ✓
- State divergence properly logged with evidence ✓
- Partition healing completes within 5 min ✓
- Evidence correctly reflects partition state ✓

---

### Scenario 4: Storage Failure — Volume Replication (40 min)

**Goal**: Verify storage subsystem detects replica failures and triggers replication recovery.

**Setup**:
- Deploy workload with persistent volume (3-replica commitment)
- Volume replicas on nodes: a, b, c (one per node)
- Baseline: All 3 replicas LIVE, committed, verified

**Failure Injection**:
1. **T=0s**: Fail replica on node-b
   - Fill node-b disk to capacity (triggers replica eviction or failure)
2. **Expected**: node-b reports replica FAILED, observed replica count becomes 2
3. **T=30s**: Recovery should trigger replication rebuild
   - New replica scheduled on different node
   - Data copied from existing replicas
   - New replica reaches COMMITTED state

**Evidence Collected**:
1. **Replica Failure Evidence**:
   - Desired replicas: 3
   - Observed replicas: 2 (node-b gone)
   - Failure timestamp, node-b signature
2. **Replication Recovery Evidence**:
   - New replica scheduled (node-d or other)
   - Replication source (node-a or node-c)
   - Completion timestamp
   - All signed with sequence numbers

**Measurement Points**:
- `replica_failure_detection_latency` = T(observed count changes) - T(disk full)
  - Target: <10 seconds
- `replica_recovery_latency` = T(new replica COMMITTED) - T(failure detected)
  - Target: <60 seconds (depends on data size; measure actual)

**Success Criteria**:
- Replica failure detected within 30 seconds ✓
- New replica scheduled and recovery starts ✓
- Recovery completes within 120 seconds (or per data size benchmark) ✓
- Desired/observed replica counts match after recovery ✓
- All replication evidence properly signed ✓

---

### Scenario 5: Control Plane Restart (30 min)

**Goal**: Verify state persistence across control plane restart, no workload disruption.

**Setup**:
- Cluster stable with 5 workloads running
- All state committed to Raft log on all nodes

**Failure Injection**:
1. **T=0s**: Restart control plane process (not OS, just service)
2. **Expected**: Brief leadership election, new leader within 5 seconds
3. **No workloads should be killed or rescheduled**

**Evidence Collected**:
1. **Shutdown Evidence**:
   - Control plane shutdown timestamp, graceful flag
2. **Restart Evidence**:
   - Startup timestamp, recovered state size
3. **Leadership Election Evidence**:
   - Old leader key, new leader key, election timestamp
4. **State Validation Evidence**:
   - Workload count before/after (must match)
   - Placement evidence reloaded from log

**Success Criteria**:
- Control plane restart completes within 30 seconds ✓
- Leadership election occurs (new leader elected) ✓
- All 5 workloads remain RUNNING (no disruption) ✓
- State recovered from Raft log matches pre-restart ✓
- Evidence ledger consistent (no gaps) ✓

---

## Evidence Collection & Validation

### Evidence Record Structure

Every state change (placement, execution, resource update, failure) generates a signed `EvidenceRecord`:

```protobuf
message EvidenceRecord {
  string id = 1;                    // UUID
  string type = 2;                  // PLACEMENT | EXECUTION | USAGE | FAILURE | RECOVERY
  string workload_id = 3;           // if applicable
  string node_id = 4;               // if applicable
  int64 timestamp_unix_ms = 5;      // measurement time
  
  // State at measurement time
  string desired_state = 6;         // what we want
  string observed_state = 7;        // what we measured
  string source = 8;                // "control-plane" | "provider-{nodeId}"
  
  // Proof
  bytes signature = 9;              // Ed25519 or similar
  string signer_public_key = 10;    // Ed25519 public key
  int64 sequence_number = 11;       // per-signer sequence
  bytes previous_hash = 12;         // hash(previous record from same signer)
  
  // Context
  map<string, string> details = 13; // additional context
}
```

### Validation Procedure

**On Observer Node**:
1. Collect all EvidenceRecords from all sources (control plane, provider nodes)
2. Verify each record:
   - Signature valid (public key matches signer)
   - Sequence numbers continuous (no gaps)
   - Previous hash matches (chain integrity)
   - Timestamp within acceptable bounds (≤5 min from now)
3. Verify state transitions:
   - desired → observed → evidence flow makes sense
   - No impossible state jumps
4. Verify evidence consistency:
   - If control-plane says workload placed on node-a
   - Then node-a should have execution evidence
   - And metrics should show activity on node-a

### Evidence Output

After each scenario, generate:
```
P1-ENDTOEND-A01-SCENARIO-{N}-EVIDENCE.jsonl
  (newline-delimited JSON records, one per evidence item)

P1-ENDTOEND-A01-SCENARIO-{N}-AUDIT.md
  (human-readable audit trail with verification status)

P1-ENDTOEND-A01-SCENARIO-{N}-METRICS.json
  (latency measurements, throughput, resource usage)
```

---

## No-Mock Gate Verification

Before test execution, verify:

```bash
# 1. No mock data in control plane
grep -r "mock\|Mock\|MOCK" \
  /path/to/control-plane/src \
  --include="*.go" \
  --exclude-dir=test \
  --exclude-dir=vendor
# Expected: zero matches

# 2. No mock data in provider node agent
grep -r "mock\|Mock\|MOCK" \
  /path/to/node-agent/src \
  --include="*.go" \
  --exclude-dir=test \
  --exclude-dir=vendor
# Expected: zero matches

# 3. All state comes from real sources
grep -r "hardcoded\|placeholder" \
  /path/to/{control-plane,node-agent}/src \
  --include="*.go" | grep -v "comment\|test"
# Expected: zero production uses

# 4. Backend API returns real data
curl -H "Authorization: Bearer $TOKEN" \
  http://control-plane:8080/api/v1/nodes | jq .
# Expected: actual node data, timestamps reflect current time
```

---

## Success Criteria & Pass/Fail Thresholds

| Criterion | Threshold | Measurement | Pass/Fail |
|-----------|-----------|-------------|-----------|
| **Failure detection** | <30 sec | T(detection) - T(failure) | FAIL >30s |
| **Workload reschedule** | <120 sec | T(running) - T(detection) | FAIL >120s |
| **Evidence signing** | 100% | (signed records) / (total records) | FAIL <100% |
| **Signature verification** | 100% | (verified) / (total) | FAIL <100% |
| **Sequence continuity** | 100% | unbroken sequence per signer | FAIL gaps |
| **State transitions** | valid | no impossible state jumps | FAIL invalid |
| **No-mock gate** | 0 matches | grep for mock data | FAIL >0 |
| **Workload availability** | ≥95% | uptime during test | FAIL <95% |
| **Cluster quorum** | held | Raft consensus maintained | FAIL lost |

**Test Result**: PASS only if ALL criteria pass.

---

## Test Execution Checklist

### Pre-Test (Day 0)
- [ ] Provision 5 nodes (control, 3 providers, observer)
- [ ] Install DHP, containerd, measurement tools
- [ ] Configure NTP, verify clock sync
- [ ] Enable centralized logging (observer collects all logs)
- [ ] No-mock gate verification (grep passes)
- [ ] Test failure injection mechanisms (iptables, disk fill, etc.)
- [ ] Baseline performance measurement (empty cluster)
- [ ] Document expected times for each scenario

### Test Execution (Day 1)
- [ ] Scenario 1: Normal Operation — 30 min
  - [ ] Evidence collection completes
  - [ ] Verification passes
  - [ ] Metrics recorded
- [ ] Scenario 2: Node Failure — 60 min
  - [ ] Failure injection triggers
  - [ ] Reconciliation completes
  - [ ] Latency measurements within threshold
- [ ] Scenario 3: Partial Partition — 45 min
  - [ ] Partition detection
  - [ ] Healing verification
  - [ ] State consistency
- [ ] Scenario 4: Storage Failure — 40 min
  - [ ] Replica failure detected
  - [ ] Recovery triggers
  - [ ] Committed state reached
- [ ] Scenario 5: Control Plane Restart — 30 min
  - [ ] Restart completes
  - [ ] State recovered
  - [ ] Workloads unaffected

### Post-Test (Day 2)
- [ ] Collect all evidence records
- [ ] Run offline verification on all evidence
- [ ] Generate audit trails
- [ ] Measure latencies and throughput
- [ ] Analyze logs for errors
- [ ] Document failure root causes (if any)
- [ ] Sign final report with test evidence

### Report Output
```
P1-ENDTOEND-A01-REPORT.md
├─ Executive summary (PASS/FAIL)
├─ Test infrastructure (nodes, network)
├─ Scenario results (5 scenarios × 5 criteria each)
├─ Evidence validation (signature verification, state consistency)
├─ Latency analysis (with charts)
├─ Failure analysis (if any)
├─ Signed evidence bundle (tarball of all records)
└─ Recommendations (blockers, improvements)
```

---

## Timeline & Resource Allocation

| Phase | Duration | Owner | Output |
|-------|----------|-------|--------|
| **Setup** | 2 hours | DevOps | 5 nodes provisioned, logged in |
| **Scenario 1-5** | 3 hours | QA | Evidence logs, metrics |
| **Verification** | 2 hours | QA + Code | Signature checks, state audit |
| **Report** | 1 hour | QA | PASS/FAIL + recommendations |
| **Total** | 8 hours | | Signed qualification report |

---

## Gate Exit Criteria

**P1-ENDTOEND-A01 is SEALED** when:
1. All 5 scenarios execute without data loss or workload crashes
2. Evidence validation passes 100% (all signatures verify)
3. Failure detection and recovery times meet thresholds
4. No-mock gate passes (zero mock data in production paths)
5. All evidence records form valid chain (sequence + hash continuity)
6. Signed report endorsed by test team

**Then**: P1 is production-ready. Proceed to P2 (First-Party DePIN Marketplace) research and implementation.

