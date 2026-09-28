# P1-CLOSE Qualification: 32 Decisive Gates

**Framework:** P1-LOCAL-VM-A01 (Single Physical Host, 3 VM Nodes)  
**Scope:** Distributed failure detection, workload rescheduling, evidence collection  
**Status:** Gates 1-8 (Baseline) + Gates 9-32 (Failure Injection)

---

## Gates 1-8: Baseline Workload Execution (Smoke + Extended)

These gates verify basic cluster functionality without failure injection.

### Gate 1: Environment Readiness
**Condition:** All prerequisites met (qemu, jq, SSH keys, network)  
**Metric:** Environment verification step succeeds  
**Evidence:** master.log [STEP 1] PASS  
**Blocker Exit:** qemu-system-x86_64 missing → BLOCKED

### Gate 2: Cluster Creation
**Condition:** 3 VM definitions created with unique disk, cloud-init, SSH ports  
**Metric:** cluster.json exists with valid node_details array  
**Evidence:** master.log [STEP 2] PASS + cluster.json structure  
**Blocker Exit:** Disk allocation fails → BLOCKED

### Gate 3: VM Startup
**Condition:** All 3 QEMU processes start and PID files created  
**Metric:** All 3 VMs running, SERIAL_LOG files growing  
**Evidence:** master.log [STEP 3] PASS + ps aux shows 3 qemu-system-x86_64 processes  
**Blocker Exit:** QEMU kernel module error (no /dev/kvm, no TCG) → BLOCKED

### Gate 4: SSH Bootstrap Readiness
**Condition:** All 3 nodes respond to SSH within retry window (30s × 3 retries)  
**Metric:** bootstrap-nodes.sh exits 0 on attempt N ≤ 3  
**Evidence:** master.log [STEP 4] PASS (attempt M/3)  
**Blocker Exit:** No SSH readiness within 90s → BLOCKED (TCG too slow for this environment)

### Gate 5: Network Qualification
**Condition:** Connectivity matrix (3×3 ping, latency, DNS) succeeds  
**Metric:** qualify-network.sh returns 0  
**Evidence:** master.log [STEP 5] PASS + network-qualification.json timestamp  
**Blocker Exit:** Network partition, DNS not available → BLOCKED

### Gate 6: socat Installation
**Condition:** socat installed on all 3 nodes (used for workload traffic relay)  
**Metric:** launch-workload.sh can execute workload-server via socat  
**Evidence:** master.log [STEP 6] PASS  
**Blocker Exit:** Package manager unavailable → WARN (continue)

### Gate 7: Resource Ledger Initialization
**Condition:** ResourceLedger.json created with empty allocation records  
**Metric:** init-resourceledger.sh exits 0  
**Evidence:** master.log [STEP 7] PASS + ResourceLedger.json valid  
**Blocker Exit:** Disk space exhausted → BLOCKED

### Gate 8: Smoke Test Traffic (15s)
**Condition:** Single workload runs for 15s, generates >0 HTTP requests  
**Metric:** REQUEST_COUNT > 0 in smoke-test log  
**Evidence:** master.log [STEP 8] PASS + smoke-test.log with traffic count  
**Failure Detection:** REQUEST_COUNT = 0 → FAIL (workload server never started or network broken)

---

## Gates 9-16: Workload Placement & Scheduling

**Objective:** Verify that ResourceLedger-driven placement correctly reserves and allocates resources.

### Gate 9: Single Workload Placement Success
**Condition:** request-placement.sh returns allocation_id for workload-api-01  
**Metric:** placement-result.json has valid allocation_id and assigned_node  
**Evidence:** Launch logs with `allocation_id: alloc-<timestamp>-<node>`  
**Failure Scenario:** Placement fails if all nodes exhausted → FAIL

### Gate 10: Workload Ledger State Consistency
**Condition:** ResourceLedger active-resources sum equals workload CPU + memory allocation  
**Metric:** `jq '.active_resources[].allocation_id' ResourceLedger.json` includes workload-api-01  
**Evidence:** reconcile-resourceledger.sh shows allocation persisted  
**Failure Scenario:** Ledger not updated → FAIL (undetected double-booking)

### Gate 11: Three Concurrent Workloads (Baseline)
**Condition:** launch-baseline-workload.sh starts 3 workloads, all reach RUNNING state  
**Metric:** All 3 workload processes visible on their assigned nodes  
**Evidence:** workload-api-01, workload-api-02, workload-cache-01 all have allocation_ids  
**Failure Scenario:** Only 2/3 placement succeeds (3rd exhausted) → FAIL

### Gate 12: Workload Traffic Distribution
**Condition:** HTTP requests directed to correct workload server on assigned node  
**Metric:** Each workload logs >0 requests with correct node hostname in server response  
**Evidence:** traffic logs show 3 distinct response patterns per workload  
**Failure Scenario:** Traffic misrouted (server on wrong node) → FAIL

### Gate 13: Baseline Duration (120s)
**Condition:** All 3 baseline workloads run to completion without timeout  
**Metric:** Elapsed time from start to all COMPLETE ≤ 130s (baseline + 10s margin)  
**Evidence:** master.log [STEP 9] PASS  
**Failure Scenario:** Any workload timeout or kill → FAIL

### Gate 14: Aggregated Request Count
**Condition:** Total requests across 3 workloads > 100 (reasonable baseline)  
**Metric:** REQUEST_COUNT = workload-api-01 + workload-api-02 + workload-cache-01  
**Evidence:** baseline.log with aggregated count  
**Failure Scenario:** REQUEST_COUNT < 100 → WARN (performance issue, not blocking)

### Gate 15: Workload Process Exit Codes
**Condition:** All 3 workload-server.sh processes exit with code 0 (clean shutdown)  
**Metric:** Server exit code logged in workload state file  
**Evidence:** workload-[id]-state.json has `"exit_code": 0`  
**Failure Scenario:** Exit code ≠ 0 → FAIL (crash or unhandled signal)

### Gate 16: Ledger Cleanup After Workload
**Condition:** deallocate-resource.sh removes allocation from ResourceLedger  
**Metric:** Post-deallocate, `jq '.allocations[] | select(.allocation_id == "alloc-...")' ResourceLedger.json` returns empty  
**Evidence:** reconcile-resourceledger.sh shows allocation marked dealloc  
**Failure Scenario:** Allocation persists after dealloc → FAIL (orphaned resource leak)

---

## Gates 17-22: Failure Detection & Recovery (Network Partition)

**Objective:** Verify detection and reconciliation when a node becomes unreachable.

### Gate 17: Network Partition Injection (Single Node)
**Condition:** Simulate network loss on dh-node-2 (block all traffic in/out via iptables on host)  
**Metric:** Node responds to direct ping but SSH commands fail; control-plane observes timeout  
**Evidence:** Bootstrap script logs node-2 SSH_TIMEOUT within 30s  
**Failure Scenario:** Partition not detected within 60s → FAIL (heartbeat timeout too long)

### Gate 18: Control-Plane Detection Latency
**Condition:** Measure time from partition injection to heartbeat-timeout + state-change  
**Metric:** Detection latency ≤ 45s (3× heartbeat timeout + grace)  
**Evidence:** Timestamp in control-plane log when node marked UNREACHABLE  
**Failure Scenario:** Latency > 60s → WARN (acceptable but suboptimal)

### Gate 19: Workload Migration Trigger
**Condition:** If workload running on partitioned node, control-plane initiates migration to healthy node  
**Metric:** New allocation_id created for same workload on different node  
**Evidence:** placement-result.json shows new assignment with different node  
**Failure Scenario:** No migration initiated → FAIL (workload remains stranded)

### Gate 20: Migrated Workload Startup
**Condition:** Workload server starts on new node within 30s of migration decision  
**Metric:** SSH command to new node succeeds, workload-server process running  
**Evidence:** Remote workload-state.json shows process_start_time ≤ migration_trigger + 30s  
**Failure Scenario:** Workload fails to start → FAIL (infrastructure issue on healthy node)

### Gate 21: Traffic Continuity After Migration
**Condition:** HTTP requests to migrated workload succeed without interruption after migration  
**Metric:** Request sequence unbroken; latency spike ≤ 5s during migration  
**Evidence:** traffic.log shows continuous GET responses before/after migration  
**Failure Scenario:** Gap > 5s in traffic log → WARN (acceptable brief interruption)

### Gate 22: Partition Recovery (Node Heals)
**Condition:** Remove iptables rule, restore network connectivity to node-2  
**Metric:** SSH commands succeed; node responds to membership gossip  
**Evidence:** Node state transitions from UNREACHABLE → HEALTHY within 30s  
**Failure Scenario:** Node remains in UNREACHABLE state → FAIL (gossip join failed)

---

## Gates 23-28: Failure Injection - Process Crashes

**Objective:** Verify behavior when critical processes crash (agent, control-plane components).

### Gate 23: Workload Server Crash (While Running)
**Condition:** Kill workload-server.sh process with SIGKILL while processing traffic  
**Metric:** Control-plane detects process exit via process monitor  
**Evidence:** Node observes PID no longer running; logs attempt to restart  
**Failure Scenario:** Process death not detected → FAIL (monitoring broken)

### Gate 24: Workload Restart After Crash
**Condition:** Control-plane automatically restarts workload-server on same node  
**Metric:** New PID appears in workload-state.json; server responds to new requests  
**Evidence:** traffic.log shows request gap < 10s, then recovery  
**Failure Scenario:** Workload remains down > 30s → FAIL (restart timeout exceeded)

### Gate 25: Scheduler Process Crash
**Condition:** Kill run-scheduler.sh with SIGKILL  
**Metric:** Control-plane detects Scheduler unavailability  
**Evidence:** Subsequent placement requests timeout or return error within 30s  
**Failure Scenario:** Placement requests still route to dead Scheduler → FAIL (no failover)

### Gate 26: Scheduler Recovery
**Condition:** Manual restart of scheduler (or wait for auto-restart if implemented)  
**Metric:** New placement request succeeds within 60s of restart  
**Evidence:** placement-result.json shows valid allocation_id  
**Failure Scenario:** Placement still fails → FAIL (state recovery incomplete)

### Gate 27: Multiple Simultaneous Crashes
**Condition:** Kill 2 workload servers + Scheduler simultaneously  
**Metric:** Control-plane remains responsive; surviving workload continues  
**Evidence:** Surviving workload traffic uninterrupted; control-plane logs multi-crash recovery  
**Failure Scenario:** Control-plane becomes unresponsive → FAIL (cascade failure)

### Gate 28: Crash Recovery Order Independence
**Condition:** Restart workloads in random order (not same as crash order)  
**Metric:** All workloads eventually reach RUNNING regardless of restart sequence  
**Evidence:** All allocation_ids active; all traffic resuming  
**Failure Scenario:** Startup order dependent on crash order → FAIL (hidden ordering assumption)

---

## Gates 29-32: Evidence & Verification

**Objective:** Collect comprehensive evidence and verify integrity of qualification.

### Gate 29: Evidence Collection (All Workloads)
**Condition:** collect-evidence.sh gathers baseline-evidence.json, dhp-evidence.jsonl, systemd-logs from all nodes  
**Metric:** Evidence files exist and are non-empty  
**Evidence:** find p1-evidence -type f -size +0  
**Failure Scenario:** Evidence collection fails → FAIL (cannot verify claims)

### Gate 30: Evidence Cryptographic Integrity
**Condition:** Each evidence record (dhp-evidence.jsonl) has valid signature (Ed25519)  
**Metric:** `jq '.signature' p1-evidence/*/dhp-evidence.jsonl | xargs -I {} echo {} | wc -l` > 0  
**Evidence:** All records signed by node's identity key  
**Failure Scenario:** Unsigned or invalid signature → FAIL (evidence tamperable)

### Gate 31: Consistency Cross-Check
**Condition:** ResourceLedger final state matches evidence from all nodes  
**Metric:** Compute hash(final-ledger) + hash(aggregated-evidence) → consistent hash  
**Evidence:** Verification report shows consistency PASS  
**Failure Scenario:** Ledger contradicts node observations → FAIL (data corruption or Byzantine bug)

### Gate 32: Tamper Detection (Negative Control)
**Condition:** Deliberately corrupt one evidence artifact (e.g., modify REQUEST_COUNT in workload-api-01 traffic log)  
**Metric:** Verification script rejects evidence bundle; reports FAIL with corruption description  
**Evidence:** Verification report explicitly names the tampered file and hash mismatch  
**Failure Scenario:** Tampered evidence accepted as valid → FAIL (no integrity checking)

---

## Failure Mode Summary

| Gate Range | Component | Failure Mode | Impact |
|-----------|-----------|--------------|--------|
| 1-8 | Baseline | Environment missing, SSH fail, network down | BLOCKED - cannot proceed |
| 9-16 | Placement | Ledger inconsistency, resource exhaustion | FAIL - scheduling broken |
| 17-22 | Network partition | Detection latency, migration failure, gossip recovery | FAIL - resilience broken |
| 23-28 | Process crashes | Restart failure, cascade failures, ordering assumptions | FAIL - reliability broken |
| 29-32 | Evidence | Collection failure, signature invalid, tampering undetected | FAIL - verification broken |

---

## Execution Flow

1. Run p1-qualification-master.sh (gates 1-8)
2. After baseline PASS:
   - Implement gate 9-16 verification (placement, ledger)
   - Run fault injection for gates 17-22 (network partition)
   - Run crash injection for gates 23-28 (process crashes)
   - Collect evidence for gates 29-32
3. Generate P1-CLOSE qualification report with PASS/FAIL per gate
4. Commit evidence artifacts to /evidence/P1-CLOSE-A01/
5. Create signed VerificationRecord using pkg/evidence contract

---

## Result Semantics

- **PASS**: Gate condition met, evidence captured, blocker detected if applicable
- **FAIL**: Gate condition not met, root cause logged, qualification stops
- **BLOCKED**: External dependency unavailable (no action taken on this gate)
- **WARN**: Suboptimal but non-blocking (e.g., latency high but acceptable)

P1-CLOSE qualification is **COMPLETE** when all 32 gates reach PASS status.
