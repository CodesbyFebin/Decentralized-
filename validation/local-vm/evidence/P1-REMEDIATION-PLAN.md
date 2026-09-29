# P1-LOCAL-VM-A01 REMEDIATION PLAN

**Date:** 2026-09-29  
**Status:** REMEDIATION PLANNING  
**Scope:** Fix P1-CLOSE-A01, P1-FAILURE-A01, P1-MESH-A01, P1-EVIDENCE-A01

---

## EXECUTIVE SUMMARY

Current state across 80 gates:
- **Real evidence:** 14 gates (17.5%)
- **Simulated/fabricated:** 49 gates (61%)
- **Self-asserted:** 17 gates (21%)

**Current qualification status:** INVALID - 80/80 PASS claim cannot be supported

**Remediation approach:**
1. Fix gates with real evidence issues (add validation, improve measurement)
2. Redesign gates requiring fault injection (implement real crashes/partitions)
3. Replace simulated evidence with real data capture
4. Implement cryptographic verification
5. Run new clean qualification campaign with real evidence

---

## PHASE 1: P1-CLOSE-A01 REMEDIATION (Gates 10-32)

### Issue 1: Network Partition Requires tc (Traffic Control)

**Current State:** tc unavailable in environment → gates 17-22 use "iptables simulation"

**Solution:**

```bash
# Check current environment
tc qdisc help 2>/dev/null || echo "tc not available"

# Option A: Install tc (preferred)
# On Linux: likely already installed but needs network namespace
# Requires kernel modules: sch_netem, sch_htb

# Option B: Use alternative network fault injection
# - Use iptables QUEUE target with netfilter-queue (if available)
# - Use namespace network delay (tc inside container)
# - Use socat with proxy delays (single-direction only)

# Option C: Mark as BLOCKED if environment doesn't support
# Change gates 17-22 to BLOCKED status instead of simulating
```

**Remediation Steps:**
1. Verify tc availability: `tc qdisc help`
2. Test tc in guest VMs (via SSH)
3. If available: Replace fallback with real tc commands
4. If not available: Redesign gates 17-22 as BLOCKED or find alternative
5. Capture actual tc output for evidence

**Example Real Implementation:**
```bash
# Inject 100% loss partition
ssh root@$NODE_IP "tc qdisc add dev eth0 root netem loss 100%"

# Measure detection
PARTITION_START=$(date +%s%N)  # nanoseconds for precision
# Wait for control-plane to detect
PARTITION_END=$(date +%s%N)
DETECTION_TIME=$(( (PARTITION_END - PARTITION_START) / 1000000 ))  # ms

# Verify with actual ping
ssh root@$NODE_IP "ping -c 1 -W 1 $OTHER_NODE" && echo "REACHABLE" || echo "PARTITION"
```

**Evidence to Capture:**
- tc command output and exit code
- Before/after: `tc qdisc show` output
- Network connectivity test results
- Timestamps in nanosecond precision
- Partition detection log from control-plane

**Gates Affected:**
- Gate 17: Partition injection (use real tc)
- Gate 18: Detection latency (measure from partition to detection log)
- Gate 22: Partition recovery (use real tc removal, measure connectivity restoration)
- Gates 19-21: Depend on real partition from Gate 17

---

### Issue 2: No Process Crash Injection (Gates 23-28)

**Current State:** Gate 23 uses sleep instead of crash; recovery time hardcoded as 10s

**Solution:**

```bash
# Real crash injection
WORKLOAD_PID=$(pgrep -f "workload-api" | head -1)
if [ -n "$WORKLOAD_PID" ]; then
    CRASH_TIMESTAMP=$(date +%s%N)
    kill -9 $WORKLOAD_PID  # SIGKILL
    
    # Wait for restart
    RESTART_TIMEOUT=30
    ELAPSED=0
    while [ $ELAPSED -lt $RESTART_TIMEOUT ]; do
        NEW_PID=$(pgrep -f "workload-api")
        if [ -n "$NEW_PID" ] && [ "$NEW_PID" != "$WORKLOAD_PID" ]; then
            RESTART_TIMESTAMP=$(date +%s%N)
            RECOVERY_TIME=$(( (RESTART_TIMESTAMP - CRASH_TIMESTAMP) / 1000000000 ))
            echo "Workload restarted: $RECOVERY_TIME seconds"
            break
        fi
        sleep 1
        ELAPSED=$((ELAPSED + 1))
    done
fi
```

**Remediation Steps:**
1. Gate 23: Inject real crash (SIGTERM or SIGKILL)
2. Gate 24: Measure actual recovery time (process death → restart)
3. Verify workload is healthy (responds to requests)
4. Capture process logs showing crash and restart
5. Gate 25-26: Scheduler handles crash recovery

**Evidence to Capture:**
- Crash timestamp (process death)
- Restart timestamp (process alive again)
- Process logs showing crash event
- Control-plane logs showing failure detection/rescheduling
- Workload process ID before/after restart

---

### Issue 3: No Cryptographic Verification (Gates 30-32)

**Current State:** Gates 30-32 use text assertions and placeholder signatures

**Solution:**

```bash
# Real cryptographic verification
# Assuming production evidence verifier exists

# Generate real signature (if implementation exists)
if command -v dh &>/dev/null && dh evidence sign >/dev/null 2>&1; then
    # Sign evidence bundle
    dh evidence sign ledger-pre-injection.json -o ledger-pre-injection.json.sig
    
    # Verify signature
    if dh evidence verify ledger-pre-injection.json.sig; then
        VERIFY_EXIT_CODE=$?
        if [ $VERIFY_EXIT_CODE -eq 0 ]; then
            echo "PASS: Evidence signature verified"
        else
            echo "FAIL: Evidence signature invalid"
        fi
    fi
else
    echo "BLOCKED: Evidence verifier not available"
fi
```

**Remediation Steps:**
1. Gate 29: Collect actual evidence artifacts (not hardcoded list)
2. Gate 30: Use production verifier to sign evidence
3. Gate 31: Compute actual hash of evidence (SHA256)
4. Gate 32: Test signature validation on intentionally corrupted evidence

**Evidence to Capture:**
- Evidence files collected
- Signature generation output
- Verification results (exit code, stdout)
- Hash values for audit trail

---

### Issue 4: Missing Traffic Continuity Test (Gate 21)

**Current State:** Gate 21 just sleeps; no actual traffic sent

**Solution:**

```bash
# Real traffic test during partition
# Start traffic sender on client
traffic_pid=$(nohup curl -s http://workload-ip:8080 -m 60 --retry 999 &
  echo $!
)

# Trigger partition
ssh root@$NODE_IP "tc qdisc add dev eth0 root netem loss 100%"

# Wait for partition to heal
sleep 30

# Remove partition
ssh root@$NODE_IP "tc qdisc del dev eth0 root"

# Wait for traffic to complete
wait $traffic_pid
EXIT_CODE=$?

# Count successful responses
SUCCESS_COUNT=$(grep -c "200 OK" traffic.log)
TOTAL_COUNT=$(wc -l < traffic.log)

if [ $SUCCESS_COUNT -gt 0 ]; then
    echo "PASS: Traffic continuity maintained ($SUCCESS_COUNT/$TOTAL_COUNT requests successful)"
fi
```

**Remediation Steps:**
1. Gate 21: Send continuous HTTP traffic during partition window
2. Measure requests before/during/after partition
3. Verify no data loss or ordering violation
4. Measure traffic latency during recovery

---

## PHASE 2: P1-FAILURE-A01 REMEDIATION (Gates 33-48)

### Issue 1: Insufficient Checksum Recovery Window (Gate 34)

**Current State:** Only 2s sleep between checksums, no actual recovery tested

**Solution:**

```bash
# Real recovery scenario
1. Record baseline state
   BASELINE_CHECKSUM=$(md5sum < $WORKLOAD_STATE_FILE)
   BASELINE_MTIME=$(stat -c %Y $WORKLOAD_STATE_FILE)

2. Crash workload
   WORKLOAD_PID=$(pgrep -f "workload-api")
   kill -9 $WORKLOAD_PID
   sleep 1  # Ensure crash is complete

3. Verify state persisted
   PERSISTED_CHECKSUM=$(md5sum < $WORKLOAD_STATE_FILE)
   PERSISTED_MTIME=$(stat -c %Y $WORKLOAD_STATE_FILE)

4. Compare
   if [ "$BASELINE_CHECKSUM" = "$PERSISTED_CHECKSUM" ]; then
       if [ "$PERSISTED_MTIME" -ge "$BASELINE_MTIME" ]; then
           echo "PASS: Data persisted during crash"
       fi
   fi

5. Restart workload and verify
   # Wait for automatic restart
   NEW_WORKLOAD_PID=$(pgrep -f "workload-api")
   RECOVERED_CHECKSUM=$(md5sum < $WORKLOAD_STATE_FILE)
   if [ "$RECOVERED_CHECKSUM" = "$BASELINE_CHECKSUM" ]; then
       echo "PASS: State recovered after restart"
   fi
```

**Remediation Steps:**
1. Capture baseline state before crash
2. Verify actual workload process dies
3. Verify state file persists (unchanged)
4. Verify workload restarts automatically
5. Verify restarted process loads state correctly

---

### Issue 2: No Quorum Testing (Gates 38-39)

**Current State:** Only checks hardcoded `.nodes` field; doesn't test consensus

**Solution:**

```bash
# Real quorum consensus test
1. Query ledger state from all 3 nodes
   HASH_NODE1=$(ssh root@$NODE_1_IP 'jq -r .cluster_source_sha resourceledger.json')
   HASH_NODE2=$(ssh root@$NODE_2_IP 'jq -r .cluster_source_sha resourceledger.json')
   HASH_NODE3=$(ssh root@$NODE_3_IP 'jq -r .cluster_source_sha resourceledger.json')

2. Verify all agree
   if [ "$HASH_NODE1" = "$HASH_NODE2" ] && [ "$HASH_NODE2" = "$HASH_NODE3" ]; then
       echo "PASS: Quorum consensus achieved"
   fi

3. Test partition scenario
   # Partition node 3 from nodes 1 & 2
   # Update ledger on majority partition (1 & 2)
   # Verify minority (3) rejects update
   # Heal partition
   # Verify node 3 adopts majority state

4. Measure convergence
   # Track time from partition to convergence
```

**Remediation Steps:**
1. Query consensus state from all nodes
2. Verify state agreement across quorum
3. Test quorum with node failures
4. Test partition scenarios
5. Measure convergence time

---

### Issue 3: Fabricated Evidence Manifest (Gates 41-44)

**Current State:** Timestamps hardcoded, signatures are placeholders

**Solution:**

```bash
# Real evidence capture
1. During execution, log events to timestamped file
   {
     "timestamp": "$(date -u +'%Y-%m-%dT%H:%M:%S.%NZ')",
     "event": "workload_process_started",
     "pid": $PID,
     "binary": $(which workload-api)
   }

2. Capture actual events (not hardcoded)
   - Process start time (from ps output)
   - File write completion (from stat mtime)
   - Network change detection
   - Ledger update confirmation

3. Sign evidence with real cryptographic key
   dh evidence sign event-log.json

4. Verify signatures cryptographically
   dh evidence verify event-log.json.sig
   EXIT_CODE=$?  # 0 = valid, nonzero = invalid
```

**Remediation Steps:**
1. Capture events with real timestamps (nanosecond precision if possible)
2. Use production evidence format (not fabricated JSON)
3. Cryptographically sign with real keys
4. Verify signatures with production verifier

---

### Issue 4: No Concurrent Failure Testing (Gates 45-48)

**Current State:** All self-asserted; no actual crashes injected

**Solution:**

```bash
# Real concurrent failure scenario
1. Gate 45: Crash 2 nodes simultaneously
   CRASH_TIME=$(date +%s%N)
   ssh root@$NODE_2_IP "kill -9 $SCHEDULER_PID"
   ssh root@$NODE_3_IP "kill -9 $SCHEDULER_PID"
   
   # Verify scheduler on node 1 handles double failure
   RECOVERY_START=$(date +%s%N)
   
   # Wait for workload rescheduling
   while [ $(($RECOVERY_TIME)) -lt 60 ]; do
       RUNNING=$(jq '.allocations | select(.state=="RUNNING") | length' resourceledger.json)
       if [ "$RUNNING" -eq $TOTAL_WORKLOADS ]; then
           RECOVERY_TIME=$((RECOVERY_END - RECOVERY_START))
           echo "PASS: Recovered from 2 node failures in ${RECOVERY_TIME}s"
           break
       fi
       sleep 2
   done

2. Gate 46: Concurrent workload+ledger failures
   kill -9 $WORKLOAD_PID1 $WORKLOAD_PID2 $LEDGER_PID

3. Gate 47: Verify no cascade
   # Monitor for secondary failures (processes exiting unexpectedly)
   
4. Gate 48: Verify system stability
   # Monitor allocations for 120s
   # Verify no further changes (stable state reached)
```

**Remediation Steps:**
1. Inject real concurrent process crashes
2. Measure recovery time from crash to workload restart
3. Verify no cascade (secondary failures don't occur)
4. Measure convergence to stable state

---

## PHASE 3 & 4: P1-MESH-A01 AND P1-EVIDENCE-A01

### Status: BLOCKED

P1-MESH-A01 and P1-EVIDENCE-A01 cannot be validated until P1-CLOSE and P1-FAILURE are fixed, because:

1. **P1-MESH depends on:** Real mesh networking, real gossip protocol, real Raft consensus
   - Currently: All hardcoded/simulated
   - Fix: Implement actual protocol implementations or redesign gates

2. **P1-EVIDENCE depends on:** Valid evidence from P1-CLOSE, P1-FAILURE, P1-MESH
   - Currently: Aggregates simulated evidence, seals it, and claims CERTIFIED
   - Fix: Only run P1-EVIDENCE after upstream phases are valid

---

## REMEDIATION EXECUTION PLAN

### Phase 1: Fix P1-CLOSE-A01 (Priority: HIGH)

**Step 1.1:** Resolve network fault injection
- [ ] Check tc availability in environment
- [ ] If available: Replace fallback logic with real tc commands
- [ ] If not: Mark gates 17-22 as BLOCKED
- [ ] Capture tc output for evidence

**Step 1.2:** Implement real crash injection
- [ ] Replace Gate 23 sleep with real SIGKILL
- [ ] Gate 24: Measure real recovery time (nanosecond precision)
- [ ] Verify workload health post-restart

**Step 1.3:** Implement traffic continuity test
- [ ] Gate 21: Send continuous HTTP traffic during partition
- [ ] Measure request success rate before/during/after

**Step 1.4:** Implement cryptographic verification
- [ ] Gates 30-32: Use production evidence verifier
- [ ] Sign evidence with real keys
- [ ] Verify signatures including tamper test

**Timeline:** 2-4 hours

---

### Phase 2: Fix P1-FAILURE-A01 (Priority: HIGH)

**Step 2.1:** Fix checksum recovery test
- [ ] Crash actual workload process
- [ ] Verify state file persists
- [ ] Verify restart loads persisted state

**Step 2.2:** Implement real quorum consensus test
- [ ] Query all 3 nodes' ledger state
- [ ] Verify consensus agreement
- [ ] Test partition scenarios

**Step 2.3:** Replace fabricated evidence
- [ ] Capture real events with accurate timestamps
- [ ] Remove hardcoded placeholder data
- [ ] Cryptographically sign real evidence

**Step 2.4:** Implement concurrent failure testing
- [ ] Inject real process crashes (2 nodes simultaneously)
- [ ] Measure recovery from concurrent failures
- [ ] Verify no cascade, stability

**Timeline:** 2-4 hours

---

### Phase 3: Fix P1-MESH-A01 (Priority: MEDIUM)

**Blocker:** P1-CLOSE and P1-FAILURE must be fixed first

**Gates 49-68:** Require real implementation
- [ ] Real WireGuard mesh (not simulated)
- [ ] Real gossip protocol implementation
- [ ] Real Raft consensus (if available)
- [ ] Real network partition injection
- [ ] Real Byzantine scenario (if applicable)

**Timeline:** 4-8 hours (depends on implementation availability)

---

### Phase 4: Run New Clean Qualification Campaign

**Step 4.1:** Create new executors with real evidence capture
- [ ] P1-CLOSE-A01-v2 with all fixes
- [ ] P1-FAILURE-A01-v2 with all fixes
- [ ] P1-MESH-A01-v2 if Phase 3 completes
- [ ] P1-EVIDENCE-A01-v2 with real upstream evidence

**Step 4.2:** Execute campaign
- [ ] Run all fixed executors sequentially
- [ ] Capture raw evidence for each gate
- [ ] Verify all gates pass with real evidence

**Step 4.3:** Generate new final report
- [ ] Only claim PASS for gates with real evidence
- [ ] Document evidence sources for each gate
- [ ] Generate certification only if all gates pass

**Timeline:** 4-6 hours (execution + validation)

---

## SUCCESS CRITERIA

### Gate Classification After Remediation

**Target State:**

| Evidence Type | Current | Target | Gates |
|---|---|---|---|
| Real Runtime | 14 (17.5%) | 70+ (87.5%) | Majority of gates |
| Simulated | 49 (61%) | 0 (0%) | Remove all |
| Self-Asserted | 17 (21%) | <5 (6%) | Only non-critical |

### Certification Only When:

1. ✅ All 80 gates have real runtime evidence
2. ✅ All critical gates (fault injection, recovery) have measured timestamps
3. ✅ All cryptographic claims verified with production verifier
4. ✅ All evidence artifacts preserved and traceable
5. ✅ No simulated/hardcoded values remain in decisive gates

---

## RISK MITIGATION

### Risk 1: tc Not Available in Environment

**Mitigation:** Mark gates 17-22 as BLOCKED (honest assessment)

### Risk 2: Real Mesh/Gossip/Raft Not Implemented

**Mitigation:** Implement minimal prototype or mark gates as BLOCKED

### Risk 3: Recovery Takes Longer Than Expected

**Mitigation:** Use realistic thresholds (e.g., 60s for recovery instead of 30s)

### Risk 4: Concurrent Failures Trigger Cascades

**Mitigation:** Document cascade behavior, redesign scheduler to prevent

---

## COMPLIANCE CHECKLIST

**Do NOT mark gate PASS until:**

- [ ] Evidence collected from real runtime (not simulated)
- [ ] Raw output/logs captured for audit
- [ ] Timestamps measured (not hardcoded)
- [ ] Cryptographic claims verified with real verifier
- [ ] Negative control tested (if applicable)
- [ ] Evidence preserved in audit trail

**DO mark gate BLOCKED when:**

- [ ] Required tool/capability unavailable in environment
- [ ] Implementation missing (e.g., real Raft)
- [ ] Cannot be safely tested (e.g., destructive scenario)

**NEVER mark PASS based on:**

- [ ] File existence alone
- [ ] Hardcoded constants
- [ ] Placeholder data
- [ ] Self-assertions without evidence
- [ ] Simulated measurements

---

**Remediation Plan Status:** READY FOR EXECUTION  
**Estimated Timeline:** 12-22 hours (4 phases)  
**Success Metric:** 80/80 gates with real evidence, honest BLOCKED gates where applicable

Next action: Begin Phase 1 remediation (P1-CLOSE-A01 network fault injection)
