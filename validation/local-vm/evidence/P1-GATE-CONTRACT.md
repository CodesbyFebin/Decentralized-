# P1-LOCAL-VM-A01 Gate Contract

**Date:** 2026-09-29  
**Purpose:** Define machine-readable gate contracts for all decisive qualification gates  
**Principle:** Executor performs observations; Verifier evaluates against contract

---

## Contract Specification Structure

Every gate contract defines:

```
gate_id: 
qualification_id: P1-LOCAL-VM-A01
campaign_id: (assigned at execution time)
description: 
gate_type: STATIC | RUNTIME | FAILURE_INJECTION | CRYPTOGRAPHIC | RECOVERY | PERFORMANCE
required_observation: (what must be measured)
observation_source: (where measurement comes from)
negative_control: (proof of intended failure)
required_artifacts: (files/logs/records to capture)
pass_condition: (predicate for PASS)
fail_condition: (predicate for FAIL)
blocked_condition: (predicate for BLOCKED)
evidence_binding: (connection to raw artifacts)
```

---

## P1-CLOSE-A01: Gates 10-32

### Gate 10: ResourceLedger Presence

```
gate_id: 10
qualification_id: P1-LOCAL-VM-A01
description: ResourceLedger file exists and is valid JSON
gate_type: STATIC
required_observation: File exists at known path; JSON parses correctly
observation_source: File I/O and jq parsing
negative_control: Corrupt JSON; missing file; empty file
required_artifacts: 
  - ledger-pre-injection.json (copied state)
  - ledger-validation-report.txt (jq parse result)
pass_condition: |
  file_exists AND
  jq_parse_exit_code == 0 AND
  jq_output_type == "object"
fail_condition: |
  file_exists == false OR
  jq_parse_exit_code != 0 OR
  jq_output_type != "object"
blocked_condition: false
evidence_binding: 
  - artifact: ledger-pre-injection.json
  - log: ledger-validation-report.txt
```

### Gate 11-16: ResourceLedger Allocation Consistency

```
gate_id: 11-16
qualification_id: P1-LOCAL-VM-A01
description: ResourceLedger allocations satisfy Model A capacity formula
gate_type: RUNTIME
required_observation: |
  For each node in ledger:
    AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED
  For each allocation:
    allocation.cpu <= node.cpu_cores
    allocation.memory_mb <= node.memory_mb
    allocation.disk_gb <= node.disk_gb
observation_source: ResourceLedger ledger-pre-injection.json
negative_control: |
  Create allocation exceeding node capacity
  Ledger consistency check must reject it
  Verify allocation not recorded
required_artifacts:
  - ledger-pre-injection.json
  - ledger-model-a-verification.txt (formula check results)
  - ledger-capacity-violations.txt (if any found)
pass_condition: |
  FOR ALL nodes:
    available_calculated == (total - owner_reserve - reserved - allocated)
  AND
  FOR ALL allocations:
    (cpu_alloc <= cpu_total) AND
    (memory_alloc <= memory_total) AND
    (disk_alloc <= disk_total)
fail_condition: |
  ANY node: available_calculated != (total - owner_reserve - reserved - allocated)
  OR
  ANY allocation exceeds node capacity
blocked_condition: |
  ResourceLedger file missing (depends on Gate 10)
evidence_binding:
  - artifact: ledger-pre-injection.json
  - log: ledger-model-a-verification.txt
```

### Gates 17-18: Network Partition Injection & Detection

```
gate_id: 17-18
qualification_id: P1-LOCAL-VM-A01
description: Real network partition injected; detection latency measured
gate_type: FAILURE_INJECTION
required_observation: |
  1. Baseline: Node is reachable (SSH/ping succeeds)
  2. Injection: Network packet loss set to 100% (real mechanism)
  3. Negative control: Verify partition is active (TCP timeout, zero throughput)
  4. Detection: Measure time from partition to SSH timeout
  5. Measurement: Capture timestamps with nanosecond precision
observation_source: |
  - tc (traffic control) on guest node
  - SSH connectivity test
  - System timestamps (date +%s%N)
negative_control: |
  Before partition: ping/SSH must succeed
  During partition: ping/SSH must fail after threshold
  After partition removal: ping/SSH must succeed again
required_artifacts:
  - pre-partition-connectivity.log (ping/SSH success)
  - partition-injection-command.txt (tc qdisc output)
  - partition-injection-stderr.log (tc errors if any)
  - ssh-detection-attempts.log (each attempt timestamp + result)
  - post-partition-connectivity.log (recovery verification)
pass_condition: |
  tc_qdisc_exit_code == 0 AND
  partition_active_confirmed == true AND
  detection_latency_ms <= 45000 AND
  partition_removal_exit_code == 0 AND
  post_partition_connectivity == true
fail_condition: |
  tc_qdisc_exit_code != 0 AND blocker_not_documented
fail_if_simulation: |
  iptables_fallback_used AND no_real_packet_loss_confirmed
blocked_condition: |
  tc_command_not_found == true
  → Document blocker in partition-blocker-evidence.txt
  → Do not execute downstream gates 19-22 as PASS
evidence_binding:
  - artifact: pre-partition-connectivity.log
  - artifact: partition-injection-command.txt
  - log: ssh-detection-attempts.log
  - measurement: DETECTION_LATENCY_MS (from timestamps)
  - artifact: post-partition-connectivity.log
```

### Gate 19-20: Workload Migration Detection

```
gate_id: 19-20
qualification_id: P1-LOCAL-VM-A01
description: Workload allocation changes detected during partition
gate_type: RUNTIME
required_observation: |
  1. Capture ledger state before partition (Gate 17)
  2. Partition remains active (verified by negative control)
  3. Capture ledger state after 30s
  4. Measure allocation count/distribution changes
observation_source: ResourceLedger queries before/after partition
negative_control: |
  Ledger timestamps must bracket partition window
  Allocation changes must correlate with partition duration
  Do not accept unchanged ledger as pass
required_artifacts:
  - ledger-before-partition.json (snapshot at T_PARTITION_START)
  - ledger-after-partition.json (snapshot at T_PARTITION_START + 30s)
  - migration-diff.txt (jq diff output)
  - partition-active-confirmation.txt (verified partition still active at T_AFTER)
pass_condition: |
  partition_active_at_T_AFTER == true AND
  (allocation_count_changed OR allocation_distribution_changed) AND
  ledger_timestamps_valid == true
fail_condition: |
  partition_active_at_T_AFTER == false OR
  (allocation_count_unchanged AND allocation_distribution_unchanged)
blocked_condition: |
  Gates 17-18 BLOCKED → This gate cannot execute
evidence_binding:
  - artifact: ledger-before-partition.json
  - artifact: ledger-after-partition.json
  - log: migration-diff.txt
```

### Gate 21: Traffic Continuity During Partition

```
gate_id: 21
qualification_id: P1-LOCAL-VM-A01
description: Continuous HTTP traffic maintains connection state during partition
gate_type: FAILURE_INJECTION
required_observation: |
  1. Start continuous HTTP requests (1/sec) to workload
  2. Wait 5s (baseline traffic success rate)
  3. Inject network partition (from Gate 17)
  4. Continue HTTP requests during partition (30s window)
  5. Remove partition
  6. Continue HTTP requests post-recovery (30s window)
  7. Record every request: timestamp, status, latency, response_id
observation_source: HTTP client test harness + request logs
negative_control: |
  Pre-partition: 100% success rate required
  During partition: 0% success expected (partition is 100% loss)
  Post-recovery: 100% success rate within 10s of partition removal
required_artifacts:
  - http-traffic-continuous.log (every request + response)
  - http-request-summary.txt (counts, latencies, errors)
  - traffic-continuity-analysis.txt (success rates per phase)
pass_condition: |
  pre_partition_success_rate >= 95% AND
  partition_active_confirmed == true AND
  post_recovery_success_rate >= 95% AND
  post_recovery_time_to_100percent <= 10000ms
fail_condition: |
  pre_partition_success_rate < 95% OR
  post_recovery_success_rate < 95% OR
  post_recovery_time_to_100percent > 10000ms
blocked_condition: |
  Gates 17-18 BLOCKED → Cannot test continuity without real partition
evidence_binding:
  - artifact: http-traffic-continuous.log
  - log: http-request-summary.txt
  - log: traffic-continuity-analysis.txt
```

### Gate 22: Network Partition Recovery

```
gate_id: 22
qualification_id: P1-LOCAL-VM-A01
description: Node connectivity restored after partition removal
gate_type: RECOVERY
required_observation: |
  1. Partition is active (verified by Gate 17)
  2. Remove partition via tc qdisc del
  3. Measure time until SSH succeeds
  4. Verify node health/state
observation_source: tc qdisc removal + SSH connectivity test
negative_control: |
  During partition: SSH must fail
  After removal: SSH must eventually succeed
  Measure exact moment of recovery
required_artifacts:
  - partition-removal-command.txt (tc qdisc del output)
  - ssh-recovery-attempts.log (timestamp + result for each attempt)
  - post-recovery-node-health.txt (uptime, services, ledger state)
pass_condition: |
  partition_removal_exit_code == 0 AND
  recovery_latency_ms <= 45000 AND
  post_recovery_ssh_success == true AND
  post_recovery_node_healthy == true
fail_condition: |
  recovery_latency_ms > 45000 OR
  post_recovery_ssh_success == false OR
  post_recovery_node_healthy == false
blocked_condition: |
  Gates 17-18 BLOCKED → Cannot test recovery without real partition
evidence_binding:
  - artifact: partition-removal-command.txt
  - log: ssh-recovery-attempts.log
  - measurement: RECOVERY_LATENCY_MS (from timestamps)
  - log: post-recovery-node-health.txt
```

### Gate 23: Workload Crash Injection

```
gate_id: 23
qualification_id: P1-LOCAL-VM-A01
description: Actual workload process terminated; crash detected by control-plane
gate_type: FAILURE_INJECTION
required_observation: |
  1. Identify workload process (via SSH ps)
  2. Capture process state: PID, command, listening ports, recent successful request
  3. Send SIGKILL to workload process
  4. Verify process is gone (ps shows no process)
  5. Verify control-plane crash-detector reports event
observation_source: SSH process operations; control-plane logs
negative_control: |
  Before crash: workload process must exist and respond to requests
  After SIGKILL: process must not exist
  Control-plane must record crash event
required_artifacts:
  - workload-process-before-crash.txt (ps output + listening ports)
  - workload-http-success-before-crash.log (successful request)
  - crash-injection-command.txt (kill -9 output)
  - workload-process-after-crash.txt (ps confirms gone)
  - control-plane-crash-detection.log (crash event recorded)
pass_condition: |
  workload_process_before_exists == true AND
  workload_responding_before == true AND
  kill_exit_code == 0 AND
  workload_process_after_exists == false AND
  control_plane_crash_detected == true
fail_condition: |
  workload_process_before_exists == false OR
  kill_exit_code != 0 OR
  workload_process_after_exists == true OR
  control_plane_crash_detected == false
blocked_condition: false
evidence_binding:
  - artifact: workload-process-before-crash.txt
  - artifact: workload-http-success-before-crash.log
  - timestamp: T_INJECT (from crash-injection-command.txt)
  - log: workload-process-after-crash.txt
  - log: control-plane-crash-detection.log
```

### Gate 24: Workload Automatic Recovery

```
gate_id: 24
qualification_id: P1-LOCAL-VM-A01
description: Workload process automatically restarted within 30s
gate_type: RECOVERY
required_observation: |
  1. At T_INJECT (from Gate 23): workload killed
  2. Loop: every 1s check if workload process exists
  3. At T_RESTART: new workload process appears
  4. Measure recovery_latency = T_RESTART - T_INJECT
  5. Verify new process responds to HTTP requests
observation_source: SSH process checks; HTTP requests to recovered workload
negative_control: |
  Before recovery: process must be gone
  During recovery loop: capture every check timestamp + result
  After recovery: new process must respond
required_artifacts:
  - crash-inject-timestamp.txt (T_INJECT in nanoseconds)
  - recovery-loop-checks.log (each check: timestamp, process_exists, PID)
  - recovery-complete-timestamp.txt (T_RESTART in nanoseconds)
  - workload-http-success-after-recovery.log (first successful request post-restart)
pass_condition: |
  recovery_latency_ms := (T_RESTART - T_INJECT) / 1000000
  recovery_latency_ms <= 30000 AND
  workload_responding_after_recovery == true
fail_condition: |
  recovery_latency_ms > 30000 OR
  workload_responding_after_recovery == false OR
  recovery_timeout_60s_exceeded == true
blocked_condition: |
  Gate 23 FAIL → Cannot measure recovery if crash injection failed
evidence_binding:
  - artifact: crash-inject-timestamp.txt (T_INJECT)
  - log: recovery-loop-checks.log (all timestamps)
  - artifact: recovery-complete-timestamp.txt (T_RESTART)
  - measurement: RECOVERY_LATENCY_MS
  - artifact: workload-http-success-after-recovery.log
```

### Gate 25: Scheduler Process Operational State

```
gate_id: 25
qualification_id: P1-LOCAL-VM-A01
description: Scheduler process is running and in operational state
gate_type: RUNTIME
required_observation: |
  1. Query ps for scheduler process
  2. Read /proc/[pid]/stat (state: R=running, S=sleeping, etc.)
  3. Read /proc/[pid]/status (memory, threads)
  4. Verify process file descriptors (listening ports)
observation_source: pgrep + /proc filesystem
negative_control: |
  Scheduler process must exist
  State must be R or S (not Z=zombie, D=uninterruptible)
required_artifacts:
  - scheduler-process-info.txt (ps output)
  - scheduler-proc-stat.txt (raw /proc/[pid]/stat)
  - scheduler-proc-status.txt (memory, FDs)
pass_condition: |
  scheduler_pid_found == true AND
  scheduler_state in [R, S] AND
  scheduler_memory_mb > 0 AND
  scheduler_file_descriptors >= 4
fail_condition: |
  scheduler_pid_found == false OR
  scheduler_state in [Z, D, T] OR
  scheduler_memory_mb == 0
blocked_condition: false
evidence_binding:
  - artifact: scheduler-process-info.txt
  - artifact: scheduler-proc-stat.txt
  - measurement: SCHEDULER_STATE (from /proc)
```

### Gate 26: Scheduler Responsiveness Post-Crash

```
gate_id: 26
qualification_id: P1-LOCAL-VM-A01
description: Scheduler responds to requests after workload crash (Gate 23)
gate_type: RECOVERY
required_observation: |
  1. After crash recovery (Gate 24)
  2. Query scheduler health endpoint (if available)
  3. Or verify scheduler accepts new scheduling request
  4. Capture response
observation_source: HTTP query to scheduler API
negative_control: |
  Scheduler must respond (not timeout)
  Response must indicate operational state
required_artifacts:
  - scheduler-health-query.log (HTTP request + response)
  - scheduler-response-time.txt (latency)
pass_condition: |
  scheduler_response_code in [200, 201] AND
  scheduler_response_contains_operational_indicator == true AND
  scheduler_response_time_ms <= 5000
fail_condition: |
  scheduler_response_code != 200 OR
  scheduler_response_time_ms > 5000 OR
  scheduler_no_response == true
blocked_condition: |
  Gate 24 BLOCKED → Cannot test if recovery didn't complete
evidence_binding:
  - artifact: scheduler-health-query.log
  - measurement: SCHEDULER_RESPONSE_TIME_MS
```

### Gates 27-28: Control-Plane Consistency & Workload Recovery

```
gate_id: 27-28
qualification_id: P1-LOCAL-VM-A01
description: Control-plane remains consistent; workloads reach ACTIVE state
gate_type: RECOVERY
required_observation: |
  1. After crash recovery (Gate 24)
  2. Query ResourceLedger for workload allocation states
  3. Count allocations in ACTIVE state
  4. Verify no cascading failures (ledger accessible, no corruption)
observation_source: ResourceLedger queries post-crash
negative_control: |
  Ledger must be parseable
  No allocation states should be FAILED/UNKNOWN
required_artifacts:
  - ledger-post-recovery.json (state after crash+recovery)
  - workload-state-summary.txt (allocation states + counts)
  - control-plane-consistency-check.txt (parse validation)
pass_condition: |
  ledger_parseable == true AND
  workload_active_count >= (workload_count * 0.8) AND
  no_cascading_failures == true
fail_condition: |
  ledger_parseable == false OR
  workload_active_count < (workload_count * 0.8) OR
  cascading_failures_detected == true
blocked_condition: |
  Gate 24 BLOCKED → Cannot verify post-crash state
evidence_binding:
  - artifact: ledger-post-recovery.json
  - log: workload-state-summary.txt
  - measurement: WORKLOAD_ACTIVE_PERCENT
```

### Gate 29: Evidence Artifact Collection

```
gate_id: 29
qualification_id: P1-LOCAL-VM-A01
description: All required evidence artifacts collected and indexed
gate_type: STATIC
required_observation: |
  1. Verify all required artifact files exist
  2. Verify all artifacts are readable
  3. Create evidence manifest listing all artifacts
  4. Compute digest for each artifact
observation_source: File I/O
negative_control: |
  Missing artifact = fail
  Corrupted/unreadable artifact = fail
  Manifest must enumerate all artifacts
required_artifacts:
  - evidence-manifest.json (list of all artifacts + digests)
  - evidence-manifest-digest.txt (SHA256 of manifest itself)
pass_condition: |
  ALL required artifacts exist AND
  ALL required artifacts readable AND
  evidence_manifest_created == true AND
  manifest_digest_valid == true
fail_condition: |
  ANY required artifact missing OR
  ANY required artifact unreadable OR
  evidence_manifest_creation_failed == true
blocked_condition: false
evidence_binding:
  - artifact: evidence-manifest.json
  - artifact: evidence-manifest-digest.txt
```

### Gate 30: Cryptographic Signature Verification

```
gate_id: 30
qualification_id: P1-LOCAL-VM-A01
description: Evidence manifest cryptographically signed and verified
gate_type: CRYPTOGRAPHIC
required_observation: |
  1. Generate/load signing keypair (Ed25519)
  2. Sign evidence manifest
  3. Verify signature against public key
  4. Capture verification result
observation_source: Cryptographic operations (openssl dgst)
negative_control: |
  Tamper manifest and re-verify → verification must fail
negative_control_artifact: |
  evidence-manifest-tampered.json (intentional modification)
  verification-of-tampered-manifest.log (verification FAIL)
required_artifacts:
  - evidence-manifest-signature.sig (binary signature)
  - evidence-public-key.pem (public key)
  - signature-verification-result.txt (verification output + exit code)
  - verification-of-tampered-manifest.log (negative control result)
pass_condition: |
  signature_creation_exit_code == 0 AND
  signature_verification_exit_code == 0 AND
  verification_output contains "Verified OK" AND
  tampered_manifest_verification_exit_code != 0
fail_condition: |
  signature_creation_exit_code != 0 OR
  signature_verification_exit_code != 0 OR
  tampered_manifest_verification_exit_code == 0 (negative control failed)
blocked_condition: false
evidence_binding:
  - artifact: evidence-manifest-signature.sig
  - artifact: evidence-public-key.pem
  - log: signature-verification-result.txt
  - log: verification-of-tampered-manifest.log (negative control)
```

### Gate 31: ResourceLedger Consistency Verification

```
gate_id: 31
qualification_id: P1-LOCAL-VM-A01
description: ResourceLedger Model A formula verified across all nodes
gate_type: RUNTIME
required_observation: |
  1. For each node in ledger-pre-injection.json
  2. Extract: total, owner_reserve, reserved, allocated
  3. Calculate: available = total - owner_reserve - reserved - allocated
  4. Verify available matches ledger value
  5. Verify allocated <= total
observation_source: ResourceLedger mathematical validation
negative_control: |
  Intentionally violate formula (e.g., allocated > total)
  Verification must reject it
required_artifacts:
  - ledger-model-a-validation.txt (formula check for each node)
  - ledger-consistency-errors.txt (any violations found)
pass_condition: |
  FOR ALL nodes:
    (calculated_available == ledger_available) AND
    (allocated <= total) AND
    (allocated >= 0)
fail_condition: |
  ANY node:
    (calculated_available != ledger_available) OR
    (allocated > total) OR
    (allocated < 0)
blocked_condition: |
  Gate 10 FAIL → Cannot validate if ledger unreadable
evidence_binding:
  - artifact: ledger-model-a-validation.txt
  - log: ledger-consistency-errors.txt
```

### Gate 32: Tamper Detection Negative Control

```
gate_id: 32
qualification_id: P1-LOCAL-VM-A01
description: Tampered evidence rejected by authoritative verifier
gate_type: CRYPTOGRAPHIC
required_observation: |
  1. Take evidence-manifest.json (valid)
  2. Create evidence-manifest-tampered.json
  3. Modify: change a gate status (e.g., "PASS" → "FAIL")
  4. Do NOT update signature
  5. Run verifier against tampered manifest
  6. Capture: exit code, output
observation_source: Cryptographic verifier (dh evidence verify or equivalent)
negative_control: |
  Tampered manifest MUST be rejected
  Verification exit code must be nonzero
required_artifacts:
  - evidence-manifest-tampered.json (intentionally corrupted)
  - tamper-verification-output.log (verifier output + exit code)
  - tamper-verification-error.txt (error message if any)
pass_condition: |
  tampered_manifest_verification_exit_code != 0 AND
  verification_output contains "tampered" OR "invalid" OR "rejected" OR "failed"
fail_condition: |
  tampered_manifest_verification_exit_code == 0 (tamper not detected)
blocked_condition: false
evidence_binding:
  - artifact: evidence-manifest-tampered.json
  - log: tamper-verification-output.log
  - log: tamper-verification-error.txt
```

---

## Remediation Success Criteria (Corrected)

**OLD CRITERIA (REJECTED):**
- 70+ real gates
- <5 self-asserted gates

**NEW CRITERIA (AUTHORITATIVE):**

```
DECISIVE_RUNTIME_GATE_SIMULATION=0
DECISIVE_RUNTIME_GATE_HARDCODED_RESULT=0
DECISIVE_RUNTIME_GATE_SELF_ASSERTED=0
```

Every decisive runtime gate must have outcome:

- **PASS** → backed by real, timestamped observation + artifact binding
- **FAIL** → backed by real, timestamped observation demonstrating failure condition
- **BLOCKED** → backed by documented blocker + remediation path
- **UNKNOWN** → backed by documented reason (e.g., dependency BLOCKED)

Static/configuration gates may use static evidence only if the gate itself is explicitly about static state (e.g., schema version, configuration file presence).

**No gate may use:**
- Simulated measurements
- Hardcoded latency values used as measured performance
- Self-assigned result constants (e.g., `PASS=true`)
- Sleep duration as proof of completion

---

## Gate Classification

### Decisive Runtime Gates (must have real evidence)
- Gates 17-22: Network partition injection and recovery
- Gate 23: Workload crash injection
- Gate 24: Workload recovery latency
- Gate 26: Scheduler responsiveness
- Gate 27-28: Control-plane consistency
- Gate 30: Cryptographic verification
- Gate 32: Tamper detection

### Static/Configuration Gates (may use static evidence)
- Gate 10: ResourceLedger presence
- Gate 11-16: Capacity formula validation (mathematical, not temporal)
- Gate 25: Scheduler process state
- Gate 29: Evidence collection
- Gate 31: Consistency formula

### Dependent Gates (outcome depends on predecessor)
- Gate 19-22: Depend on Gate 17 (real partition)
- Gate 24-28: Depend on Gate 23 (real crash)
- Gate 26-28: Depend on Gate 24 (successful recovery)

---

## Executor/Verifier Separation

**Executor Role:**
1. Perform operation
2. Capture observations
3. Record timestamps + measurements
4. Collect artifacts
5. Write observation log

**Executor Must NOT:**
- Decide gate outcome
- Use `pass()` to assert (only report)
- Assign hardcoded result variables
- Sleep as proof

**Verifier Role:**
1. Read gate contract
2. Read evidence artifacts
3. Evaluate pass/fail/blocked predicate
4. Sign/certify outcome
5. Bind to evidence record

**Verifier Runs:**
- After execution completes
- In clean/isolated environment
- Against immutable artifacts
- With cryptographic signature

---

## Evidence Record Binding

Every decision must bind:

```
EvidenceRecord {
  qualification_id: P1-LOCAL-VM-A01
  campaign_id: <UTC-timestamp>
  gate_id: <10-32>
  gate_contract_version: <hash>
  source_sha: <git commit>
  gate_type: <STATIC|RUNTIME|...>
  observation: <what was measured>
  artifacts: [
    {path, digest, description},
    ...
  ]
  pass_condition_predicate: <boolean expression>
  pass_condition_result: true|false
  outcome: PASS|FAIL|BLOCKED|UNKNOWN
  timestamp: <UTC>
  signer: <identity>
  signature: <cryptographic>
}
```

No gate outcome without this binding.

---

## Corrected P1-CLOSE Gate Status (After Remediation)

Will be determined by real evidence execution.

Initial expectations based on audit:

| Gate | Type | Current Status | Required Evidence | Path |
|------|------|---|---|---|
| 10 | STATIC | PASS | File existence | Direct |
| 11-16 | RUNTIME | TBD | Formula validation | Direct |
| 17-18 | FAILURE_INJECTION | BLOCKED if tc unavailable | Real partition + detection | Requires tc |
| 19-22 | RUNTIME | DEPENDENT | Partition evidence | Depends on 17 |
| 23 | FAILURE_INJECTION | TBD | Real SIGKILL | Direct |
| 24 | RECOVERY | TBD | Recovery timestamp diff | Depends on 23 |
| 25 | RUNTIME | TBD | Process state from /proc | Direct |
| 26 | RECOVERY | TBD | Scheduler API response | Depends on 24 |
| 27-28 | RECOVERY | TBD | Ledger state + counts | Depends on 24 |
| 29 | STATIC | PASS | Manifest + file list | Direct |
| 30 | CRYPTOGRAPHIC | TBD | Signature verification | Direct |
| 31 | RUNTIME | TBD | Formula check | Direct |
| 32 | CRYPTOGRAPHIC | TBD | Tamper test | Direct |

---

## Next Action

With gate contracts defined, proceed to implement P1-CLOSE executor that:
1. Respects executor/verifier separation
2. Captures raw observations (not assertions)
3. Binds all evidence to contract predicates
4. Leaves outcome determination to verifier

