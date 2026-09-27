# Resource Semantics Specification

## Overview

This document specifies the canonical capacity accounting model for P1-NODE-FLEET-A01 and its propagation through Scheduler, Placement, P2 Leases, and Marketplace components.

## Capacity Model

### Definitions

Each node in the fleet declares three resource dimensions:

**Capacity:**
```
total_cpu:      total allocatable millicores
total_memory:   total allocatable bytes
total_disk:     total allocatable bytes

owner_cpu:      millicores reserved for cluster operations (0 ≤ owner_cpu ≤ total_cpu)
owner_memory:   bytes reserved for cluster operations (0 ≤ owner_memory ≤ total_memory)
owner_disk:     bytes reserved for cluster operations (0 ≤ owner_disk ≤ total_disk)
```

**Ledger State:**
```
reserved:  sum of all active ResourceReservation amounts
allocated: sum of all active ResourceAllocation amounts
```

Constraint: For each reservation R: `sum(A.amount for A in Allocations where A.reservationID = R.ID) ≤ R.amount`

### Availability Calculation

**MODEL A: CONSUMPTIVE RESERVATION (CANONICAL)**

Allocations are carved from reservations. Available capacity for new reserves depends only on total and existing reservations, not on allocations within those reservations.

```
available = total - owner_reserve - reserved
```

**Invariants:**
1. `available ≥ 0` always (reserves rejected if they would make available < 0)
2. `allocated ≤ reserved` (enforced per-reservation in Allocate)
3. Releasing a reservation requires first releasing all allocations from it
4. Available capacity is independent of allocation utilization rate

**Example:**
```
Node: 100 CPU total, 20 CPU owner reserve

State 1: No reserves
  available = 100 - 20 - 0 = 80 CPU ✓

State 2: Reserve 50 CPU (for app A)
  available = 100 - 20 - 50 = 30 CPU ✓
  (30 CPU still available for new reserves)

State 3: Allocate 40 CPU from app A's reservation
  available = 100 - 20 - 50 = 30 CPU ✓
  (unchanged; allocated is "inside" the reservation)

State 4: Release app A's allocations (40 CPU)
  available = 100 - 20 - 50 = 30 CPU ✓
  (still unchanged; reservation remains)

State 5: Release app A's reservation
  available = 100 - 20 - 0 = 80 CPU ✓
  (reservation released, capacity returns)
```

**Rationale:**
- Reserves represent commitments to hold capacity
- Allocations represent usage within committed capacity
- Available capacity for new reserves is independent of how currently-reserved capacity is utilized
- Aligns with traditional multi-tenant capacity pooling: "reserve a pool, allocate from pool"
- Double-counting is eliminated: allocated ⊂ reserved, so counting only reserved is sufficient

### Resource Reservation Contract

**Reservation (Reserve Operation)**
- Input: `reservationID` (idempotency key), `nodeID`, `cpuMilli`, `memBytes`, `diskBytes`
- Effect: Commits capacity; prevents concurrent Reserve/Allocate operations that would exceed it
- Idempotency: Reserve(same ID, same params) succeeds on retry; Reserve(same ID, different params) returns error
- Constraint: `requested ≤ (total - owner - currently_reserved)`
- Terminal state: Once created, a reservation occupies capacity until ReleaseReservation

**Allocation (Allocate Operation)**
- Input: `allocationID` (idempotency key), `reservationID`, `cpuMilli`, `memBytes`, `diskBytes`
- Effect: Records workload usage against a reservation
- Idempotency: Allocate(same ID, same params) succeeds on retry; Allocate(same ID, different params) returns error
- Constraint: `requested + sum(existing allocations on same reservation) ≤ reservation.amount`
- Terminal state: Once created, an allocation occupies reservation capacity until ReleaseAllocation

**Release Operations**
- ReleaseAllocation: Removes allocation record, freeing reservation capacity for new allocations
- ReleaseReservation: Removes reservation record (only if all allocations released first), freeing node capacity for new reserves

### Semantics Across Subsystems

#### Fleet
- Implements the ledger and enforces availability constraints
- Response to query: "Can I reserve X capacity?" → Check: `X ≤ available`
- Response to query: "How much is allocated?" → Return: sum of allocations (for observability only)
- Capacity Equation: `available = total - owner - reserved`

#### Scheduler
- Consumes `available` from Fleet to determine node eligibility
- Rejection code `INSUFFICIENT_CPU` when node available < request.cpu
- Rejection code `INSUFFICIENT_MEMORY` when node available < request.memory
- Makes placement decisions based on available capacity (not allocated)

#### Placement
- After scheduling decision, issues Allocate() call to Fleet ledger
- Allocate() consumes `available` capacity within the reservation
- If allocation fails, placement fails (exception path)
- Successful allocation records workload identity

#### P2 Leases
- Lease is a multi-month or multi-year reservation
- Lease::Create → Fleet::Reserve with long-lived reservation ID
- Lease::Cancel → Fleet::ReleaseReservation (after releasing all contained workloads)
- Lease::Available = Fleet available capacity within lease reservation

#### Marketplace
- Marketplace price is based on requested reserve amount, not allocated
- Dynamic pricing: price per CPU-month for reserved capacity (consumed whether allocated or not)
- Utilization is orthogonal to pricing (owner pays for reservation, not consumption)

### Failure Scenarios & Resolutions

**Double-Release:**
- Reserve(ID=A), ReleaseReservation(A), ReleaseReservation(A)
- Second release must be idempotent: returns success with no state change
- Mechanism: Persisted terminal operation state (not time-based grace period)

**Overcommit Attempt:**
- Node has 100 CPU total, 20 owner reserve (80 available)
- Reserve(50 CPU) succeeds; available = 30
- Reserve(35 CPU) fails; available remains 30
- Reserve(30 CPU) succeeds; available = 0
- Reserve(1 CPU) fails; capacity exhausted

**Allocation Beyond Reservation:**
- Reserve(ID=R1, 50 CPU)
- Allocate(ID=A1, R1, 40 CPU) succeeds
- Allocate(ID=A2, R1, 20 CPU) fails; exceeds reservation
- Allocate(ID=A2, R1, 10 CPU) succeeds; total allocation = 50 CPU

**Release with Active Allocations:**
- Reserve(ID=R1, 50 CPU)
- Allocate(ID=A1, R1, 40 CPU)
- ReleaseReservation(R1) fails; allocations still active
- ReleaseAllocation(A1)
- ReleaseReservation(R1) succeeds

### Formal Definition

For a node with capacity model C and ledger L:

```
node_total(C)      = C.total_cpu or C.total_memory or C.total_disk
node_owner(C)      = C.owner_cpu or C.owner_memory or C.owner_disk
user_reserved(L,C) = Σ R.amount for all R ∈ L.Reservations where R.nodeID = C.nodeID
allocated(L,C)     = Σ A.amount for all A ∈ L.Allocations where A.nodeID = C.nodeID
available(L,C)     = node_total(C) - node_owner(C) - user_reserved(L,C)

Invariant 1: ∀R ∈ Reservations, R.nodeID = C.nodeID
  ⟹ Σ A.amount for all A ∈ Allocations where A.reservationID = R.ID ≤ R.amount

Invariant 2: ∀C ∈ Nodes
  ⟹ available(L,C) ≥ 0

Invariant 3: ∀R ∈ Reservations
  ⟹ ∃ zero or more A ∈ Allocations where A.reservationID = R.ID
```

### Implementation Notes

**Ledger Structure**
```go
type ResourceLedger struct {
    CapacityByNode   map[string]*NodeCapacityModel    // authoritative node capacity
    Reservations     map[string]*ResourceReservation   // active reservations by ID
    Allocations      map[string]*ResourceAllocation    // active allocations by ID
    Generation       int64                             // for optimistic concurrency
}
```

**Atomicity**
- Reserve, Allocate, ReleaseReservation, ReleaseAllocation must be atomic
- In replicated control plane: atomicity comes from Raft FSM Apply, not process-local mutex
- Each operation is a command: SetCapacityCommand, ReserveCommand, AllocateCommand, ReleaseReservationCommand, ReleaseAllocationCommand
- FSM Apply is deterministic (no time.Now, randomness, network calls)

**Idempotency**
- Each operation has an idempotency key (ID)
- Terminal operations must be durable across:
  - Control plane restart
  - Raft leader change
  - Snapshot and restore
  - Log replay
- Mechanism: Persisted terminal operation record (not time-based grace period)

**Observability**
- Available capacity is the canonical source of truth for scheduler decisions
- Allocated capacity is available for observability/reporting but does not affect availability calculation
- Combined capacity-plus-allocation reports can show "reserved but not yet allocated" for capacity planning

---

## Decision Log

**Date:** 2026-09-27
**Decision:** MODEL A: CONSUMPTIVE RESERVATION
**Rationale:**
- Simpler availability formula: `available = total - owner - reserved`
- Aligns with traditional capacity pooling (reserve pools, allocate from pools)
- Eliminates apparent double-counting of allocated capacity
- Allocation utilization is independent of availability for new reserves
- Clearer semantics across Fleet, Scheduler, Placement, P2 Leases, Marketplace

**Approved By:** P1-NODE-FLEET-A01 Gate 4 §2
**Status:** CANONICAL (no further changes without formal approval)
