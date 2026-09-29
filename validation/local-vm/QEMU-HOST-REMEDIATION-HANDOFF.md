# QEMU Host Remediation Diagnostic Handoff

**Source SHA**: 19681d3
**Date**: 2026-09-29
**Status**: Ready for host execution

## Overview

This document provides the QEMU host (with actual VM guest cluster support) with the
exact diagnostic workflow needed to verify PR #30 remediation before source freeze.

This is NOT P1 qualification. This is remediation diagnostics that must pass before
the P1 candidate SHA is frozen.

## Environment Requirements

### Required Tools

```
qemu-system-x86_64     (QEMU hypervisor, version 5.0+)
qemu-img              (QEMU disk image management)
jq                    (JSON query tool)
ssh                   (OpenSSH, for guest access)
ip                    (iproute2, network interface management)
tc                    (Linux traffic control, for fault injection)
nft                   (nftables, firewall rules - optional alternative to iptables)
iptables              (legacy iptables - optional alternative to nft)
python3               (Python 3.8+, for network diagnostics)
git                   (for source checkout verification)
```

### Minimum Resources

- **CPU cores**: ≥4 (1 per QEMU guest + 1 host overhead)
- **RAM**: ≥3 GB (1 GB per guest + margin)
- **Disk**: ≥20 GB free (Ubuntu images + snapshots)
- **Virtualization**: KVM or equivalent (tcg slower but functional)
- **Network**: Multicast capable (QEMU socket multicast on localhost)

### Host Network Configuration

- Multicast address `239.192.42.17:12347` must be routable on localhost
- No existing QEMU processes on ports 2201-2203 (SSH forwarding)
- No existing persistent cluster state in `validation/local-vm/state/`

## Remediation Diagnostic Workflow

### Phase 1: Environment Setup

```bash
bash validation/local-vm/scripts/preflight.sh qemu
```

Validates:
- QEMU binary is available and runnable
- Host has sufficient resources (CPU, RAM, disk)
- Git source SHA is accessible
- SSH key generation possible

**Exit code 0**: Proceed to Phase 2
**Exit code 2**: Missing tools or insufficient resources (BLOCKED)

### Phase 2: Cluster Creation

```bash
bash validation/local-vm/scripts/create-vm-cluster.sh --nodes 3 --cpu 1 --memory 1024 --disk 8
```

Creates:
- 3 QEMU guests (dh-node-1, dh-node-2, dh-node-3)
- Ubuntu cloud-init seed ISOs
- QEMU disk images (qcow2 snapshots)
- `cluster.json` with full topology metadata

**Output**: `validation/local-vm/state/cluster.json`

Validates:
- Mesh backend: `qemu-socket-multicast`
- Mesh bus: `239.192.42.17:12347`
- Mesh IP addresses: 172.30.10.1, 172.30.10.2, 172.30.10.3
- Primary MAC addresses: 52:54:00:10:00:01/02/03
- Mesh MAC addresses: 52:54:00:20:00:01/02/03
- Source SHA binding

### Phase 3: Cluster Start

```bash
bash validation/local-vm/scripts/start-cluster.sh
```

Launches:
- 3 QEMU processes with virtio-net devices on both primary and mesh networks
- Primary network: user-mode NAT with SSH forwarding (127.0.0.1:2201-2203)
- Mesh network: socket multicast for guest-to-guest L2/L3 connectivity

**Validates**:
- QEMU process running
- PID files created and valid
- Serial logs accessible

Exit code 0: Proceed to Phase 4
Exit code 2: QEMU launch failed (BLOCKED)

### Phase 4: Bootstrap and SSH Readiness

```bash
bash validation/local-vm/scripts/bootstrap-nodes.sh
```

Verifies:
- Cloud-init completed on all 3 guests
- SSH daemon running and accepting connections
- Machine IDs unique and persistent
- Network interfaces up (eth0 primary, eth1 mesh)

**Required**: 3/3 SSH authenticated connections before proceeding

If any guest fails:
1. Inspect serial log: `cat validation/local-vm/state/node-N/serial.log`
2. Check cloud-init: `ssh ubuntu@127.0.0.1 -p 220N "cloud-init status"`
3. Verify network: `ssh ubuntu@127.0.0.1 -p 220N "ip addr; ip route"`

### Phase 5: Mesh Network Verification

```bash
bash validation/local-vm/scripts/qualify-interguest-network.sh
```

Tests:
- 6 directed guest-to-guest paths (N×(N-1) for 3 guests)
- Each path: `ping -I mesh0` from source to destination mesh IP
- Captures: `ip -4 addr show dev mesh0`, `ip -4 route get`, ping results

**Required**: 6/6 paths PASS

Output: `validation/local-vm/evidence/interguest-network/observations.jsonl`

If any path fails:
1. SSH to source guest: `ssh -p 220N ubuntu@127.0.0.1`
2. Check mesh interface: `ip addr show dev mesh0` (expect 172.30.10.X/24)
3. Check route: `ip route` (expect default via multicast socket)
4. Debug multicast: `netstat -an | grep 239.192.42.17:12347`

### Phase 6: Mesh HTTP Service

```bash
bash validation/local-vm/scripts/launch-workload.sh workload-api-01 0.5 256 1 60
```

Starts:
- HTTP server on guest returning JSON with `workload_id`
- Continuous traffic from host to guest over forwarded SSH port
- Resource monitoring (CPU, memory, disk)
- Structured traffic logging (JSONL format)

**Required**: 
- HTTP 200 responses
- Workload ID matches request
- No invalid JSON
- Successful completion (traffic_log shows success count > 0)

Output: 
- `validation/local-vm/state/workload-api-01.json` (state)
- `validation/local-vm/state/workload-logs/workload-api-01-traffic.jsonl` (detailed traffic)

### Phase 7: Real Network Partition (Diagnostic)

```bash
python3 validation/local-vm/scripts/observe-network-fault.py \
  --node dh-node-2 \
  --workload-id workload-api-02 \
  --http-port 18082 \
  --duration 20 \
  --output validation/local-vm/evidence/network-fault-diagnostic-001 \
  --inject
```

Executes:
- Pre-fault connectivity check (HTTP and route)
- Schedules heal timer (independent systemd-run, runs after fault)
- Schedules injection timer (100% loss on mesh0, via tc netem)
- Continuous HTTP traffic for 45 seconds (covers fault window)
- Post-fault observation (network state, service journal)

**Note**: This is a DIAGNOSTIC only. It tests host-forwarded SSH traffic loss, not
mesh partition detection. Mesh partition detection requires implementation of actual
distributed heartbeat/membership checking in the runtime.

Exit code 0: Diagnostic captured traffic loss and restoration
Exit code 2: Preconditions failed (tc unavailable, QEMU PID invalid, etc.)

Output: `validation/local-vm/evidence/network-fault-diagnostic-001/`
- `events.jsonl` (timeline of events)
- `traffic.jsonl` (individual HTTP requests with latency, error)
- `network-before.txt`, `network-after.txt` (tc and route state)
- `guest-fault-journal.txt` (systemd service execution logs)
- `traffic-summary.json` (aggregate: success, failed, downtime_seconds)

### Phase 8: Workload Failure Injection

```bash
# Deploy concurrent workloads
bash validation/local-vm/scripts/launch-workload.sh workload-api-01 0.5 256 1 120
bash validation/local-vm/scripts/launch-workload.sh workload-api-02 0.5 256 1 120
bash validation/local-vm/scripts/launch-workload.sh workload-cache-01 0.5 256 1 120

# Kill one workload process and observe reconciliation
ssh ubuntu@127.0.0.1 -p 2201 "pkill -9 -f python3"

# Wait for reconciliation and verify traffic resumes
sleep 10
bash validation/local-vm/scripts/launch-workload.sh workload-api-01 0.5 256 1 30
```

**Required**:
- 3 concurrent workloads active
- Post-kill traffic resumes after reconciliation
- New workload has valid JSON responses

### Phase 9: ResourceLedger Validation

```bash
# Initialize ledger from cluster
bash validation/local-vm/scripts/init-resourceledger.sh

# Test atomic operations
bash validation/local-vm/tests/resourceledger-model-a.sh
bash validation/local-vm/tests/resourceledger-concurrency.sh
bash validation/local-vm/tests/placement-concurrency.sh
```

**Required**:
- Atomic reservation→allocation transfer (RESERVED decreases, ALLOCATED increases)
- Concurrent requests produce no double-allocation
- Duplicate workload IDs rejected
- Negative availability prevented

### Phase 10: Evidence Signing Verification

```bash
go test ./pkg/evidence -v
```

**Validates**:
- Ed25519 signature generation and verification
- Artifact hash preservation (BLAKE3)
- Tampered artifact rejection
- Source digest reproducibility

## Cleanup and Artifact Collection

### Preserve Diagnostic Evidence

```bash
mkdir -p $HOME/p1-remediation-evidence-$(date +%s)
cp -r validation/local-vm/evidence/* $HOME/p1-remediation-evidence-*/
```

### Destroy Cluster

```bash
bash validation/local-vm/scripts/destroy-cluster.sh
rm -f validation/local-vm/state/*.json validation/local-vm/state/*.lock
```

## Failure Interpretation

| Exit Code | Meaning | Action |
|-----------|---------|--------|
| 0 | Diagnostic passed | Proceed to next phase |
| 2 | Precondition failed (BLOCKED) | Fix blocker, retry phase |
| 1 | Unexpected error | Check logs, diagnose root cause |

## Critical Points

1. **Source SHA Binding**: Every cluster is bound to a specific source SHA. If source
   changes between phases, cluster must be recreated.

2. **No Simulation**: Every test involves actual QEMU processes, real guest networking,
   and genuine traffic. No mock objects or hardcoded results.

3. **Fail-Closed**: Legacy verifiers are quarantined. Scripts exit non-zero on
   precondition failure. No PASS values are emitted that haven't been verified.

4. **Mesh is New**: The socket-multicast mesh (qemu-socket-multicast backend) is the
   remediation being verified. Previous clusters used user-mode NAT forwarding only.

5. **Diagnostic ≠ Qualification**: Network fault diagnostics are educational tests,
   not P1 gates. They validate fault injection mechanics but do not assess production
   detection or reconciliation logic.

## Post-Run Verification

If all phases complete with exit code 0:

1. Gather evidence directory with structured JSON
2. Verify no hardcoded test values in evidence
3. Run `dh evidence verify` on evidence bundle
4. Check source SHA is bound in all records
5. Confirm Ed25519 signatures are valid

Then:

- Document the exact SHA tested
- Report which gates were exercised (not passed)
- Update qualification candidate only after independent review

## Next Steps After Remediation Passes

1. **Code Review**: Is the implementation sound? Are there any defects to fix?
2. **Candidate Freeze**: Pick exact SHA for P1-LOCAL-VM-A01 candidate
3. **Decisive Campaign**: Run full P1-CLOSE workflow on frozen candidate
4. **Final Verification**: Independent signature and artifact verification
5. **P1 Verdict**: QUALIFIED only when all 32 gates pass and evidence verifies

## Support

If a phase fails:
1. Consult the phase description above
2. Check the specific exit code and logged reason
3. Inspect relevant state files (cluster.json, serial logs, network state)
4. Fix the specific defect (usually a precondition or tool availability issue)
5. Do NOT force a PASS; fail-closed and diagnose

Simulation, mocking, or hardcoding results invalidates the entire qualification.
