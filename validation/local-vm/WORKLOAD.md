# Workload: Phase 3 Baseline Traffic Generation

## Overview

The Workload subsystem implements a distributed test workload that:
1. Requests placement from the Scheduler using ResourceLedger
2. Runs on assigned nodes via SSH
3. Generates continuous HTTP traffic for baseline measurement
4. Monitors resource utilization before failure injection
5. Preserves allocation IDs for recovery tracking

## Architecture

```
Control Machine                          Cluster (3 VMs)
┌─────────────────────────────┐         ┌──────────────┐
│ launch-workload.sh          │         │ dh-node-1    │
│ 1. request-placement.sh     │────────>│ workload-api │
│ 2. SSH to assigned node     │         │ (HTTP 8080)  │
│ 3. Start workload server    │         └──────────────┘
│ 4. Generate traffic         │
│ 5. Monitor resources        │         ┌──────────────┐
│ 6. Log allocation_id        │────────>│ dh-node-2    │
└─────────────────────────────┘         │ workload-db  │
                                        │ (HTTP 8080)  │
                                        └──────────────┘
```

## Components

### 1. workload-server.sh
Simple HTTP server that runs on each node:
- Listens on 127.0.0.1:8080 (or configurable port)
- Responds to GET requests with JSON workload metadata
- Logs each request with timestamp
- Gracefully shuts down on SIGTERM

Response payload:
```json
{
  "workload_id": "workload-api-01",
  "timestamp": "2026-09-28T10:40:00Z",
  "pid": 12345,
  "uptime_seconds": 120,
  "requests_served": 450,
  "status": "running"
}
```

### 2. launch-workload.sh
Main workload launcher that orchestrates:

**Phase 1: Placement Request**
```bash
bash request-placement.sh workload-api-01 0.3 256 2
→ allocation_id: alloc-1726053600-12345
```

**Phase 2: Remote Server Start**
- SSHes to assigned node
- Starts workload-server.sh on port 8080
- Captures server logs

**Phase 3: Traffic Generation**
- Sends continuous HTTP GET requests to workload server
- Logs each request to traffic log
- Runs for specified duration (default 60s)

**Phase 4: Resource Monitoring**
- Periodically queries node for resource usage
- Logs CPU, memory, disk utilization
- Samples every 5 seconds

**Phase 5: Cleanup**
- Terminates workload server via SIGTERM
- Updates workload state file with completion time
- Returns allocation_id for deallocation

### 3. launch-baseline-workload.sh
Batch launcher for Phase 3 baseline:
- Starts multiple workloads in parallel
- Staggered starts (2s between each)
- Waits for all to complete
- Reports aggregated logs

## Lifecycle

### Phase 3.3.1: Initialize Baseline

After Scheduler is running:

```bash
cd /Users/cyberteck/Desktop/Decentralized/Decentralized-
bash validation/local-vm/scripts/launch-baseline-workload.sh 120
```

This launches 3 test workloads:
- `workload-api-01`: 0.3 CPU, 256M RAM, 2G disk
- `workload-api-02`: 0.3 CPU, 256M RAM, 2G disk  
- `workload-cache-01`: 0.2 CPU, 128M RAM, 1G disk

### Phase 3.3.2: Monitor Placement

While workloads run, in another terminal:

```bash
bash validation/local-vm/scripts/query-resourceledger.sh
```

Output shows:
```
Cluster Summary:
cpu_total: 3
cpu_allocated: 0.8
memory_total_mb: 3072
memory_allocated_mb: 640
disk_total_gb: 24
disk_allocated_gb: 5

Per-Node Status:
  dh-node-1: 0.3/1 CPU, 256/1024M RAM, 2/8G disk, 1 allocations
  dh-node-2: 0.3/1 CPU, 256/1024M RAM, 2/8G disk, 1 allocations
  dh-node-3: 0.2/1 CPU, 128/1024M RAM, 1/8G disk, 1 allocations
```

### Phase 3.3.3: Inspect Workload Logs

During/after workload execution:

```bash
# Server logs (HTTP requests received)
cat validation/local-vm/state/workload-logs/workload-api-01.log

# Traffic logs (HTTP responses captured)
cat validation/local-vm/state/workload-logs/workload-api-01-traffic.log

# Resource usage
cat validation/local-vm/state/workload-logs/workload-api-01-resources.log
```

### Phase 3.3.4: Workload State

Persistent state file for each workload:

```bash
cat validation/local-vm/state/workload-api-01.json
```

Output:
```json
{
  "workload_id": "workload-api-01",
  "allocation_id": "alloc-1726053600-12345",
  "target_node": "dh-node-1",
  "ssh_port": 2201,
  "cpu_requested": 0.3,
  "memory_requested_mb": 256,
  "disk_requested_gb": 2,
  "started_at": "2026-09-28T10:40:00Z",
  "completed_at": "2026-09-28T10:42:00Z",
  "duration_seconds": 120,
  "status": "COMPLETED"
}
```

## Integration with Failure Injection

### Scenario 1: Workload Relocation

1. Workload runs on dh-node-1 with `allocation_id: alloc-001`
2. Deallocate: `bash deallocate-resource.sh alloc-001`
3. Node 1 capacity restored
4. Relaunch: `bash launch-workload.sh workload-api-01-relocated ...`
5. Scheduler may place on different node
6. Measure placement latency and resource overhead

### Scenario 2: Node Failure → Cascade

1. Three workloads running: alloc-001, alloc-002, alloc-003
2. dh-node-2 goes offline
3. Workloads on node-2 must deallocate and reschedule
4. Scheduler places on available nodes
5. Measure recovery time and final placement state

### Scenario 3: Resource Contention

1. RequestCPU=0.5 + memory=768M on 1-CPU node (normal load)
2. Add workload requesting CPU=0.6
3. Scheduler: FAIL (insufficient capacity)
4. Queue retry with backoff
5. Measure scheduler response time under constraint

## Traffic Patterns

Current workload generates **request every 1 second** to generate:
- 1 request/sec × 120 sec = ~120 requests per workload
- 3 workloads × 120 requests = ~360 total requests in baseline period
- HTTP overhead: ~200 bytes per request
- ~72 KB total traffic in baseline

Traffic log shows:
```
2026-09-28T10:40:00Z Request 1: {"workload_id":"workload-api-01",...}
2026-09-28T10:40:01Z Request 2: {"workload_id":"workload-api-01",...}
2026-09-28T10:40:02Z Request 3: {"workload_id":"workload-api-01",...}
...
```

## Resource Monitoring

Resource logs captured every 5 seconds show:
```
2026-09-28T10:40:00Z CPU: 1 cores, Usage: 5.2%
Memory: Used: 256M / 1024M (25.0%)
Disk: Used: 2.0G / 8.0G (25.0%)

2026-09-28T10:40:05Z CPU: 1 cores, Usage: 4.8%
Memory: Used: 256M / 1024M (25.0%)
Disk: Used: 2.0G / 8.0G (25.0%)
```

## Deallocation and Cleanup

After workload completes, deallocate resources:

```bash
# Extract allocation_id from workload state
allocation_id=$(jq -r '.allocation_id' validation/local-vm/state/workload-api-01.json)

# Deallocate
bash validation/local-vm/scripts/deallocate-resource.sh "$allocation_id"

# Verify
bash validation/local-vm/scripts/query-resourceledger.sh
```

ResourceLedger now shows allocation as RELEASED (preserved in history for audit).

## Commands Reference

| Command | Purpose |
|---------|---------|
| `launch-workload.sh <id> <cpu> <mem> <disk> [duration]` | Launch single workload |
| `launch-baseline-workload.sh [duration]` | Launch 3-workload baseline |
| `query-resourceledger.sh` | View current allocations |
| `deallocate-resource.sh <id>` | Release allocation |

## Next Steps

1. Start Scheduler: `run-scheduler.sh first-fit &`
2. Launch baseline: `launch-baseline-workload.sh 120`
3. Monitor during run: `query-resourceledger.sh` in another terminal
4. Inspect logs: `cat validation/local-vm/state/workload-logs/*`
5. After completion, begin Phase 3 failure injection:
   - Agent kill scenarios
   - Node shutdown recovery
   - Resource contention tests

Phase 3.3 baseline establishes the healthy operational state before measuring failure behavior.
# Traffic observation integrity

The workload launcher writes every HTTP attempt to
`state/workload-logs/<workload-id>-traffic.jsonl`, including request ID,
timestamp, node, HTTP status when observed, latency, response identity, raw
response, and error. It parses JSON identity instead of matching a log string.
Any failed attempt or incomplete remote resource sample fails the local
workload run; the raw per-request stderr and resource log remain available.
This local result is not a decisive P1 qualification verdict. An independent
verifier must evaluate the 15-second smoke and 120-second baseline against
source-bound raw artifacts from the QEMU host.
