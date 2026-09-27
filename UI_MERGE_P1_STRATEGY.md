# UI Merge Phase P1 — Distributed Private Cloud

**Date**: 2026-09-27  
**Branch**: `claude/ui-merge-wave2-p1-evidence`  
**Phase**: P1 BEGINNING  
**Scope**: Distributed private cloud primitives, evidence-first truth contracts, no marketplace/blockchain

---

## P1 Objectives

Build the Command Centre surfaces for managing a **distributed private cloud** (multiple independently controlled Linux nodes forming a single logical cloud).

**What P1 IS**:
- Distributed scheduling across owned/controlled nodes
- Resource allocation and workload placement
- Real node failure detection and recovery
- Signed evidence of resource allocation and usage
- Desired/observed/evidence separation in all surfaces
- Freshness semantics (LIVE/STALE/UNKNOWN/UNAVAILABLE)
- No mock data, no blockchain, no tokens, no marketplace

**What P1 IS NOT**:
- Provider/consumer marketplace (that's P2)
- Token settlement or metering-for-payment (that's P3)
- External DePIN integration (that's P4)
- Open federation (that's P5)

---

## P1 UI Order

### 1. Evidence View (IN PROGRESS)
**File**: `src/components/pages/EvidenceView.tsx`  
**Focus**: Signed proof of state, resource allocation, and usage

Enhancements over P0:
- **Resource Evidence**: CPU/Memory/Storage consumption with TruthValue freshness
- **Node Evidence**: Per-node health facts, capacity (LIVE vs STALE)
- **Usage Records**: Signed metering data (provider's digital signature)
- **Evidence Chain**: Desired/observed/evidence matrix for all measurements

**Acceptance Criteria**:
- [ ] TruthValue components show freshness for all measurements
- [ ] UNKNOWN never renders as zero
- [ ] STALE data marked explicitly
- [ ] Resource evidence includes CPU, memory, storage
- [ ] Node evidence shows health and capacity
- [ ] Usage records are cryptographically signed
- [ ] No mock data (all resources come from backend)
- [ ] Responsive: mobile, tablet, desktop
- [ ] No-mock gate passes

---

### 2. Deploy View (NEXT)
**File**: `src/components/pages/DeployView.tsx`  
**Focus**: Workload placement and lifecycle management

Will implement:
- **Desired state**: Workload specification (image, resources, replicas)
- **Observed state**: Current placement and status
- **Evidence**: Signed placement proof, execution evidence
- **Lifecycle**: Deploy → Staging → Running → Stopping → Stopped
- **Failure handling**: Automatic replacement on node failure

**Key patterns**:
- Desired/observed/evidence matrix for each placement
- TruthValue for placement status freshness
- Real workload spec (OCI/container), not mock

---

### 3. Storage View (AFTER DEPLOY)
**File**: `src/components/pages/StorageView.tsx`  
**Focus**: Volume management and replication state

Will implement:
- **Volumes**: Persistent storage across nodes
- **Replication policy**: How many copies, where
- **Replication state**: Current replica distribution (observed)
- **Evidence**: Signed storage proofs from providers
- **Capacity tracking**: Used/available storage per node

---

### 4. Failure Domains & Placement (AFTER STORAGE)
**File**: `src/components/pages/NodeDetailView.tsx` (enhanced) + new placement UI  
**Focus**: Logical grouping and affinity/anti-affinity

Will implement:
- **Failure domains**: Rack, zone, region grouping (from node metadata)
- **Affinity rules**: "Place these replicas together" or "separate them"
- **Placement constraints**: "No two on same failure domain"
- **Evidence**: Why placement was approved/rejected

---

### 5. Distributed Ingress & Service Discovery (AFTER PLACEMENT)
**File**: New component  
**Focus**: Network connectivity across nodes

Will implement:
- **Service IP allocation**: Virtual IPs for services
- **DNS records**: How workloads discover each other
- **Load balancing**: Multiple replicas behind one endpoint
- **Evidence**: DNS resolution proofs, traffic routing logs

---

### 6. Copilot (LAST)
**File**: `src/components/pages/CopilotView.tsx`  
**Focus**: AI-assisted operations with audit trails

Will implement:
- **Operation suggestions**: Based on resource/failure state
- **Safety gates**: Confirmation before destructive changes
- **Audit trail**: Who/what/when for all operations
- **Evidence**: Signed audit records

---

## Truth Contracts (Foundation)

All P1 surfaces use the **desired/observed/evidence** triple:

```typescript
interface DesiredObservedEvidence<T> {
  desired: TruthEnvelope<T>;      // intended state (control plane)
  observed: TruthEnvelope<T>;     // actual state (provider measurement)
  evidence: TruthEnvelope<T>;     // signed proof (provider signature)
  discrepancy?: string;            // if they don't match
}
```

**Freshness rules**:
- LIVE: Measured within SLA window (e.g., <60s)
- STALE: Older than SLA window but not lost
- UNKNOWN: Not measured yet
- UNAVAILABLE: Measurement impossible (e.g., node offline)

**Rendering rules**:
- UNKNOWN never renders as zero/false/healthy
- STALE always marked explicitly (amber warning)
- LIVE shown with confidence (emerald)
- UNAVAILABLE shown as "—" with reason

---

## DHP Primitives (P1-P2 Boundary)

P1 establishes these entities (normalized, no external code imports):

```go
// P1 establishes
Provider           // node identity + capabilities
Reservation        // resource allocated to a workload
Placement          // workload instance on a node
UsageRecord        // signed metering (provider signature)
EvidenceRecord     // signed proof of state/allocation/execution

// P2 adds
Offer              // provider publishes available resources
ResourceRequest    // consumer wants resources
Bid                // provider responds with price
Lease              // agreement between consumer and provider

// P3 adds (optional)
SettlementRecord   // finalized accounting with payment
ChainAdapter       // blockchain commitment (optional)
```

---

## Testing & Qualification

**P1-ENDTOEND-A01**: First distributed test
- [ ] 3+ independently controlled Linux nodes
- [ ] Actual workload placement across nodes
- [ ] Real node failure (kill service/crash node)
- [ ] Automatic failure detection
- [ ] Workload reconciliation (move replicas)
- [ ] Service continuity or measured restoration
- [ ] Signed evidence for all state changes
- [ ] No-mock gate passes
- [ ] Multiple processes on one host NOT sufficient

---

## Branch & Commit Strategy

**Base**: `main` (P0 merged)  
**Branch**: `claude/ui-merge-wave2-p1-evidence`  
**Commits**: One per P1 surface
- `P1: Evidence View - resource/node/usage evidence with TruthValue`
- `P1: Deploy View - desired/observed/evidence for workloads`
- `P1: Storage View - volume replication evidence`
- `P1: Failure Domains - affinity/anti-affinity constraints`
- `P1: Distributed Ingress - service discovery and load balancing`
- `P1: Copilot - AI operations with audit trails`
- `P1: P1-ENDTOEND-A01 - distributed test qualification`

---

## DePIN Research (Parallel, Read-Only)

**Location**: `~/dh-depin-research/`  
**Status**: Non-invasive, architecture documentation only

Contents:
- `MANIFEST.md`: 11 upstream projects (Akash, Golem, Bacalhau, Filecoin, Sia, etc.)
- `DHP-PRIMITIVES.md`: Normalized entities (Provider, Offer, Lease, etc.)
- (No code imports until P2)

---

## Acceptance Criteria (P1 Complete)

- [ ] Evidence View: TruthValue for all measurements, no-mock gate pass
- [ ] Deploy View: Workload placement, desired/observed/evidence visible
- [ ] Storage View: Volume replication, evidence of copies
- [ ] Failure Domains: Affinity rules, placement constraints
- [ ] Distributed Ingress: Service discovery, load balancing
- [ ] Copilot: Operation suggestions, audit trails
- [ ] P1-ENDTOEND-A01: Real distributed test with node failure
- [ ] All surfaces: UNKNOWN never renders as zero, STALE marked
- [ ] All surfaces: No-mock gate passes (zero mock imports)
- [ ] All surfaces: TypeScript lint clean
- [ ] PR ready for review on `claude/ui-merge-wave2-p1-evidence`

---

## Next Phase Trigger

P1 complete when:
1. All 6 surfaces shipped
2. P1-ENDTOEND-A01 passes (real distributed test)
3. No-mock gate confirms zero mock data
4. PR merged to main

Then **P2 (First-Party DePIN Marketplace)** begins:
- Provider/consumer marketplace without blockchain
- Offer/ResourceRequest/Bid/Lease protocol
- P2-E2E test: independent operators offering resources
