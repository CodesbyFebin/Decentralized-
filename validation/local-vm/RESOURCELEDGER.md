# ResourceLedger: Phase 3 Workload Baseline

## Overview

ResourceLedger is a persistent state store that tracks resource allocations and claims across the P1-LOCAL-VM-A01 cluster. It is required for Phase 3 (workload/control-plane baseline) qualification before failure injection tests can produce meaningful evidence.

## Architecture

ResourceLedger operates as a single source of truth (similar to `cluster.json` in Phase 1) with:
- Per-node capacity declarations (CPU, memory, disk)
- Current allocation state (what is claimed and by whom)
- Allocation history for audit and recovery
- Atomic updates via file locking and transactional writes

State file: `validation/local-vm/state/resourceledger.json`

## Schema

```json
{
  "schema_version": 1,
  "qualification": "P1-LOCAL-VM-A01",
  "created_at": "2026-09-28T10:30:00Z",
  "cluster_source_sha": "a5d0e04...",
  "ledger_source_sha": "4842a6a...",
  "nodes": 3,
  "node_capacity": [
    {
      "node": "dh-node-1",
      "cpu_cores": 1,
      "memory_mb": 1024,
      "disk_gb": 8,
      "cpu_allocated": 0.5,
      "memory_allocated_mb": 512,
      "disk_allocated_gb": 4,
      "allocations": [
        {
          "allocation_id": "alloc-1726053000-12345",
          "workload_id": "workload-scheduler-01",
          "cpu": 0.5,
          "memory_mb": 512,
          "disk_gb": 4,
          "state": "ACTIVE",
          "created_at": "2026-09-28T10:30:15Z"
        }
      ]
    }
  ],
  "summary": {
    "cpu_total": 3,
    "cpu_allocated": 0.5,
    "memory_total_mb": 3072,
    "memory_allocated_mb": 512,
    "disk_total_gb": 24,
    "disk_allocated_gb": 4
  }
}
```

## Lifecycle

### Phase 3.1: Initialize ResourceLedger

After Phase 2 (network qualification) completes with status READY:

```bash
cd /Users/cyberteck/Desktop/Decentralized/Decentralized-
bash validation/local-vm/scripts/init-resourceledger.sh
```

This creates `resourceledger.json` with:
- Cluster topology from `cluster.json`
- Per-node resource capacities (1 CPU, 1024 MB RAM, 8 GB disk per node)
- Source SHA references (cluster and ledger)
- Empty allocations list

### Phase 3.2: Query Resource State

Display current allocation state:

```bash
# Overall cluster summary
bash validation/local-vm/scripts/query-resourceledger.sh

# Specific node details
bash validation/local-vm/scripts/query-resourceledger.sh dh-node-1
```

Output shows:
- Total and allocated resources per node
- Available capacity
- Active allocations and their workload IDs

### Phase 3.3: Allocate Resources

When scheduler or operator wants to place a workload:

```bash
bash validation/local-vm/scripts/allocate-resource.sh \
  dh-node-1 \
  0.5 \
  512 \
  4 \
  workload-scheduler-01
```

This:
1. Validates availability on the target node
2. Generates unique allocation_id
3. Records allocation with workload_id and timestamp
4. Updates node and summary allocated totals
5. Returns allocation_id for future reference

### Phase 3.4: Deallocate Resources

When workload completes or is rescheduled:

```bash
bash validation/local-vm/scripts/deallocate-resource.sh alloc-1726053000-12345
```

This:
1. Finds allocation by ID
2. Returns resources to available pool
3. Marks allocation state as RELEASED
4. Updates node and summary totals
5. Preserves allocation in history for audit

## Integration with Scheduler and Workload

### Scheduler Integration

The scheduler must:
1. Before placement: query ResourceLedger to identify nodes with available capacity
2. On placement decision: call `allocate-resource.sh` with workload_id
3. On workload failure/rescheduling: call `deallocate-resource.sh` with allocation_id

### Workload Tracking

Each workload must:
1. Be assigned a workload_id (e.g., `workload-{component}-{instance}`)
2. Record the allocation_id returned by allocate-resource.sh
3. Report resource utilization during execution (for Phase 3 baseline measurement)
4. Trigger deallocate when work completes or before rescheduling

## Failure Injection Readiness

ResourceLedger enables meaningful failure tests:

- **Agent Kill**: Deallocate resources when agent dies, verify rescheduling
- **VM Shutdown**: Deallocate all node allocations before shutdown, measure recovery time
- **Partition/Heal**: Track allocation changes during network isolation and recovery
- **Resource Contention**: Oversubscribe allocations and measure scheduler behavior

Before-/after measurements compare ResourceLedger state across failure boundaries, providing quantitative evidence for P1 qualification properties.

## Commands Reference

| Command | Purpose |
|---------|---------|
| `init-resourceledger.sh` | Initialize from cluster.json |
| `query-resourceledger.sh [node]` | Display allocation state |
| `allocate-resource.sh N cpu mem disk workload` | Request allocation |
| `deallocate-resource.sh allocation-id` | Release allocation |

## Error Handling

- **Insufficient capacity**: allocate fails with clear availability message
- **Not found**: deallocate fails if allocation_id doesn't exist
- **Lock contention**: File lock prevents concurrent writes (retries handled by flock)
- **JSON corruption**: Atomic writes via tmp file + mv prevent partial writes

## Next Steps

1. Run `init-resourceledger.sh` on Mac after Phase 2 passes
2. Implement Scheduler that queries and uses ResourceLedger
3. Implement Workload components that request allocations
4. Establish continuous traffic baseline with active workload
5. Then begin Phase 3 failure injection scenarios
