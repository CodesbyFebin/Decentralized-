# Scheduler: Phase 3 Resource-Aware Placement

## Overview

The Scheduler reads ResourceLedger to make resource-aware placement decisions for workloads across the P1-LOCAL-VM-A01 cluster. It implements two placement strategies (first-fit and best-fit) and records all placements in a manifest for audit and failure injection scenarios.

## Architecture

The Scheduler operates as a standalone daemon that:
1. Monitors placement requests from workloads
2. Queries ResourceLedger for available node capacity
3. Makes placement decisions using a configurable strategy
4. Calls `allocate-resource.sh` to record the placement
5. Returns placement results to the requesting workload
6. Logs all decisions and allocations to a placement manifest

Communication with workloads uses JSON files in the state directory:
- **Inbound**: `$STATE_DIR/placement-request.json` — workload requests
- **Outbound**: `$STATE_DIR/placement-result.json` — scheduler responses

## Strategies

### First-Fit
Returns the first node with sufficient available capacity. Fast and simple; useful for baseline measurements.

```bash
run-scheduler.sh first-fit
```

### Best-Fit
Returns the node that would leave the most remaining capacity after placement. Better resource utilization and suitable for complex failure scenarios.

```bash
run-scheduler.sh best-fit
```

## Lifecycle

### Phase 3.2.1: Start Scheduler

After ResourceLedger is initialized:

```bash
cd /Users/cyberteck/Desktop/Decentralized/Decentralized-
bash validation/local-vm/scripts/run-scheduler.sh first-fit &
SCHEDULER_PID=$!
```

The scheduler logs to `validation/local-vm/state/scheduler.log` and listens for requests.

### Phase 3.2.2: Request Placement

Workloads call `request-placement.sh` to ask the scheduler where to run:

```bash
bash validation/local-vm/scripts/request-placement.sh \
  workload-api-01 \
  0.5 \
  512 \
  4
```

This:
1. Writes a placement request to `placement-request.json`
2. Waits up to 30 seconds for the scheduler to respond
3. Returns placement result or times out

### Phase 3.2.3: Scheduler Decision Flow

```
request-placement.sh
  └─> placement-request.json (workload: api-01, cpu: 0.5, mem: 512M, disk: 4G)
        └─> run-scheduler.sh (listening)
              ├─ query-resourceledger.sh (find available capacity)
              ├─ best-fit selection (dh-node-1 chosen)
              └─ allocate-resource.sh dh-node-1 0.5 512 4 workload-api-01
                    └─ ResourceLedger updated, allocation-id returned
      └─> placement-result.json (workload: api-01, result: PASS, node: dh-node-1, allocation-id: alloc-...)
            └─ request-placement.sh returns allocation-id to workload
```

### Phase 3.2.4: Placement Manifest

The scheduler maintains `placements.json` with a complete audit trail:

```json
{
  "schema_version": 1,
  "qualification": "P1-LOCAL-VM-A01",
  "created_at": "2026-09-28T10:35:00Z",
  "strategy": "best-fit",
  "placements": [
    {
      "workload_id": "workload-api-01",
      "result": "PASS",
      "target_node": "dh-node-1",
      "allocation_id": "alloc-1726053300-12345",
      "timestamp": "2026-09-28T10:35:15Z"
    },
    {
      "workload_id": "workload-db-01",
      "result": "PASS",
      "target_node": "dh-node-2",
      "allocation_id": "alloc-1726053325-67890",
      "timestamp": "2026-09-28T10:35:40Z"
    },
    {
      "workload_id": "workload-cache-01",
      "result": "FAIL",
      "target_node": "",
      "allocation_id": "",
      "timestamp": "2026-09-28T10:35:50Z"
    }
  ]
}
```

This manifest enables:
- **Audit**: See all placement decisions and their outcomes
- **Failure Recovery**: Extract allocation_ids before node shutdown
- **Rescheduling**: Identify failed placements to retry with different strategy
- **Evidence**: Compare placement state before/after failure injection

## Integration with Workload

Workloads call `request-placement.sh` as part of startup:

```bash
# Example workload component startup
workload_name="workload-scheduler-01"
allocation_id=$(bash request-placement.sh "$workload_name" 0.5 512 4) || exit 1

# Now run workload with knowledge of allocation
export WORKLOAD_ID="$workload_name"
export ALLOCATION_ID="$allocation_id"
./run-workload.sh
```

The workload must preserve `allocation_id` for later deallocation during failure recovery.

## Failure Injection Scenarios

### Scenario 1: Agent Kill → Rescheduling

1. Agent dies (or is killed)
2. Deallocate resources via `deallocate-resource.sh $allocation_id`
3. Retry placement via `request-placement.sh` with new workload_id
4. Measure rescheduling latency from placement request to new allocation

### Scenario 2: Node Shutdown → Capacity Loss

1. Node goes offline
2. All allocations on that node become invalid
3. Identify affected workloads from `placements.json`
4. Deallocate and reschedule to remaining nodes
5. Measure recovery: time from shutdown to all workloads rescheduled

### Scenario 3: Oversubscription → Placement Failures

1. Request more resources than cluster capacity
2. Observe scheduler FAIL response
3. Trigger backoff/retry logic in workload
4. Measure scheduler response time under constraint

## Commands Reference

| Command | Purpose |
|---------|---------|
| `run-scheduler.sh <strategy>` | Start scheduler daemon |
| `request-placement.sh <workload> <cpu> <mem> <disk>` | Request placement |
| `query-resourceledger.sh` | View current allocations |
| `allocate-resource.sh` | Direct allocation (called by scheduler) |
| `deallocate-resource.sh <id>` | Release allocation |

## Limitations and Design Notes

- **Single-threaded**: The current implementation processes one placement request at a time. For concurrent workloads, queue-based scheduling may be needed in Phase 4.
- **File-based IPC**: Placement requests use JSON files instead of network RPC. This is sufficient for Phase 3 baseline and failure injection on a single physical host.
- **No preemption**: Once allocated, resources are reserved until explicitly deallocated. No eviction or overcommit logic.
- **Synchronous placement**: `request-placement.sh` blocks waiting for scheduler response (30s timeout). Async placement with queues could be added if Phase 3.3 needs higher throughput.

## Next Steps

1. Run `init-resourceledger.sh` to create ResourceLedger
2. Start scheduler: `run-scheduler.sh first-fit &`
3. Implement test workload that calls `request-placement.sh`
4. Verify placements appear in `placements.json`
5. Test failure injection: deallocate and rescheduling scenarios

Phase 3.3 will implement the Workload component that generates continuous traffic with active allocations.
