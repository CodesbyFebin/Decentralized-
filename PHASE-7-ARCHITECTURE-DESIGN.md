# Phase 7 Architecture Design - Multi-Region Expansion

**Document Version:** 1.0  
**Date:** 2026-10-04  
**Status:** Ready for Technical Review & Approval  
**Target Approval:** Oct 8, 2026

---

## Executive Architecture Summary

Phase 7 transforms Decentralized.Host from single-region (450 nodes, 1-3 operators) to multi-region federation (2000+ nodes, 5-20 operators) through five integrated subsystems:

```
┌─────────────────────────────────────────────────────────┐
│        PHASE 7: MULTI-REGION FEDERATION                 │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │  7A: Multi-Region Control Plane                  │  │
│  │  ├─ 5-node Raft cluster per region               │  │
│  │  ├─ Regional quorum + global quorum              │  │
│  │  ├─ Sub-100ms global operation latency           │  │
│  │  └─ Automatic leader election + failover        │  │
│  └──────────────────────────────────────────────────┘  │
│                           ↓                              │
│  ┌──────────────────────────────────────────────────┐  │
│  │  7B: Cross-Region Workload Replication           │  │
│  │  ├─ Primary + 2 replicas (3-region minimum)      │  │
│  │  ├─ Delta snapshot replication (88% reduction)   │  │
│  │  ├─ Automatic failover (<10s detection)          │  │
│  │  └─ Merkle anti-entropy (background sync)       │  │
│  └──────────────────────────────────────────────────┘  │
│                           ↓                              │
│  ┌──────────────────────────────────────────────────┐  │
│  │  7C: Operator Federation                         │  │
│  │  ├─ Tier progression (BOOTSTRAP→TRUSTED→MASTER) │  │
│  │  ├─ Sponsorship model (TRUSTED sponsor)          │  │
│  │  ├─ Revenue sharing (5% commission)              │  │
│  │  └─ Federation policies (MASTER governance)      │  │
│  └──────────────────────────────────────────────────┘  │
│                           ↓                              │
│  ┌──────────────────────────────────────────────────┐  │
│  │  7D: Global Load Balancing                       │  │
│  │  ├─ Anycast IP per workload (same in all regions)  │
│  │  ├─ Read routing (nearest replica)               │  │
│  │  ├─ Write routing (to primary)                   │  │
│  │  └─ Failover routing (automatic on region down)  │  │
│  └──────────────────────────────────────────────────┘  │
│                           ↓                              │
│  ┌──────────────────────────────────────────────────┐  │
│  │  7E: Advanced SLA Management                     │  │
│  │  ├─ Global uptime calculation (rolling 30d)      │  │
│  │  ├─ Tier-based thresholds (95%/98%/99.5%)       │  │
│  │  ├─ SLA credits (auto on failure)                │  │
│  │  └─ Tier progression automation                  │  │
│  └──────────────────────────────────────────────────┘  │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## System Architecture

### Multi-Region Topology (Phase 7A)

```
┌──────────────────────────────────────────────────────────────┐
│                    GLOBAL CONTROL PLANE                      │
│                                                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐ │
│  │   US-WEST       │  │   EU-CENTRAL    │  │  ASIA-EAST  │ │
│  │ (Primary Region)│  │  (Secondary)    │  │  (Tertiary) │ │
│  │                 │  │                 │  │             │ │
│  │ Raft: 5 nodes   │  │ Raft: 5 nodes   │  │ Raft: 3 nodes│ │
│  │ Quorum: 3       │  │ Quorum: 3       │  │ Quorum: 2   │ │
│  │ State: 100GB    │  │ State: 100GB    │  │ State: 100GB│ │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────┘ │
│           │                    │                    │       │
│           └────────────────────┼────────────────────┘       │
│                      Raft Consensus Log                     │
│                (Global operator state replicated)          │
└──────────────────────────────────────────────────────────────┘
                               ↓
        ┌──────────────────────┬──────────────────────┐
        │                      │                      │
   ┌────────────┐        ┌─────────────┐        ┌──────────┐
   │ US-WEST    │        │ EU-CENTRAL  │        │ASIA-EAST │
   │            │        │             │        │          │
   │ Nodes:250  │        │ Nodes: 250  │        │Nodes:250 │
   │ Operators:5│        │ Operators:5 │        │Op:5      │
   └────────────┘        └─────────────┘        └──────────┘
```

### Control Plane Design (Phase 7A)

**Raft Consensus Architecture:**
```
┌────────────────────────────────────────────────────────┐
│              Raft Cluster per Region                   │
│                                                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐             │
│  │  Node 1  │  │  Node 2  │  │  Node 3  │ ← Quorum    │
│  │ (Leader) │  │(Follower)│  │(Follower)│   (3-of-5) │
│  └──────────┘  └──────────┘  └──────────┘             │
│       ↓             ↓             ↓                    │
│   [Raft Log] [Raft Log] [Raft Log]                    │
│                                                        │
│  ┌──────────┐  ┌──────────┐                           │
│  │  Node 4  │  │  Node 5  │ ← Additional nodes       │
│  │(Follower)│  │(Follower)│   (for redundancy)      │
│  └──────────┘  └──────────┘                           │
└────────────────────────────────────────────────────────┘
     ↓              ↓              ↓
  [Persistence] [Persistence] [Persistence]
  (etcd-like)   (etcd-like)   (etcd-like)
```

**Global State Replication:**
- Regional state (node registrations): Replicated via Raft per-region
- Global state (operator profiles, policies): Replicated via Raft globally (cross-region)
- Consistency model: Strong consistency for global state (quorum writes)
- Conflict resolution: Last-write-wins with timestamp binding + operator ID

**Consensus Mechanics:**
```
Write to Operator Profile (Global State):
  1. Client sends WRITE to any control plane node
  2. Node elected as leader (or forwards to leader)
  3. Leader appends entry to local Raft log
  4. Leader replicates to followers (5-node cluster)
  5. Followers acknowledge receipt
  6. Leader waits for quorum (3-of-5 acknowledgments)
  7. Leader commits entry → applies to state machine
  8. Leader responds to client: SUCCESS
  9. Followers commit entry asynchronously
  10. Cross-region: Regional leaders gossip consensus complete
  
Latency Target: <100ms p95 (Raft replication + gossip)
Failure Handling: If leader fails, followers elect new leader (<5s)
Network Partition: Minority partition stops accepting writes (split-brain prevention)
```

---

### Cross-Region Replication (Phase 7B)

**Replica Placement Strategy:**

```
Workload Deployment (Primary + Replicas):

Primary Workload (User's home region)
  ↓ Replicate to Secondary (2nd closest region)
  ↓ Replicate to Tertiary (3rd closest region)

Example:
  Workload created in US-WEST
    → Primary: US-WEST (node-42, operator-alice)
    → Replica 1: EU-CENTRAL (node-85, operator-bob)
    → Replica 2: ASIA-EAST (node-203, operator-charlie)

Placement Constraints:
  ✓ No two replicas on same host (failure independence)
  ✓ No two replicas same operator (geographic diversity)
  ✓ Primary + replicas in different regions (disaster recovery)
  ✓ User-configurable distribution (all in primary region allowed)
```

**Replication Protocol:**

```
Write Path (Read-Your-Writes Consistency):
  1. Client: WRITE to Primary (US-WEST)
  2. Primary: Append to local log + filesystem
  3. Primary: Send delta snapshot to Replica 1 (EU-CENTRAL)
  4. Primary: Send delta snapshot to Replica 2 (ASIA-EAST)
  5. Replica 1 & 2: Acknowledge receipt
  6. Primary: Wait for 2-of-2 replicas (quorum: majority)
  7. If timeout >5s: Write to fewer replicas (availability trade-off)
  8. Primary: Return SUCCESS to client
  
Latency: <1s p99 (replication latency)

Read Path (Quorum Reads):
  1. Client: READ from any replica
  2. Contact 2-of-3 replicas (quorum)
  3. Compare versions: Return majority version
  4. Consistency guarantee: Always sees most recent committed write
```

**Synchronization Mechanics:**

```
Delta Snapshots (Incremental Updates):
  - Send only changes since last snapshot (88% bandwidth reduction)
  - Merkle tree hashing: Detect divergence within 1 hour
  - Periodic full snapshot: Weekly baseline (500MB per workload)
  - Incremental updates: Daily (50MB per workload)

Merkle Anti-Entropy (Background Sync):
  1. Periodically hash replicas' state (Merkle tree)
  2. Compare hashes across regions
  3. If divergent: Identify divergent subtrees (binary search)
  4. Fetch missing data from primary
  5. Repair replica in background (no workload disruption)
  
Detection Latency: <1 hour
Repair Latency: <30 minutes

Failover Detection:
  1. Health probe every 5 seconds (HTTP GET to primary)
  2. If no response: Increase probe frequency (every 1s)
  3. After 10s of failed probes: Declare primary dead
  4. Replicas elect new primary (quorum among remaining)
  5. New primary routes all traffic to itself
  6. Clients fail over to new primary (sub-30s total)
```

**Failover & Recovery:**

```
Primary Failure Scenario:

T=0:00  Primary (US-WEST) stops responding
T=0:10  Failover detection: Replicas declare primary dead
T=0:15  New primary election: Replicas vote (quorum: 2-of-3)
T=0:20  New primary elected (Replica 1 from EU-CENTRAL)
T=0:25  Replica 1 becomes primary, accepts new writes
T=0:30  Clients redirected to new primary
T=1:00  Old primary comes back online → discovers it's behind
T=2:00  Old primary syncs missed writes from new primary (Merkle anti-entropy)

Recovery Time Objective (RTO): <30 seconds
Recovery Point Objective (RPO): <1 second (quorum writes)
Data Loss: Zero (all replicas have majority committed writes)
```

---

### Operator Federation (Phase 7C)

**Tier Progression Model:**

```
BOOTSTRAP Tier (Month 1-3):
  ├─ Entry point for new operators
  ├─ Requirements: 50+ nodes, 100M uWork stake
  ├─ SLA target: 95% uptime (baseline)
  ├─ Progression: After 90 days ≥95% SLA + 100+ nodes
  ├─ Sponsor: None (steering committee sponsors all BOOTSTRAP)
  └─ Commission: 1.0x base rate
      
        ↓ After 90 days ≥95% SLA + 100+ nodes ↓

TRUSTED Tier (Months 3+):
  ├─ Advanced operator tier
  ├─ Requirements: 100+ nodes, 200M uWork stake
  ├─ SLA target: 98% uptime (higher standard)
  ├─ Progression: After 1 year ≥98% SLA + 250+ nodes → MASTER
  ├─ Sponsor: Can sponsor BOOTSTRAP operators (1:1 ratio)
  ├─ Sponsorship model:
  │   ├─ TRUSTED operator sponsors BOOTSTRAP operator
  │   ├─ BOOTSTRAP operator's stake held in TRUSTED wallet
  │   ├─ TRUSTED earns 5% of BOOTSTRAP's revenue
  │   ├─ Sponsorship revocable: BOOTSTRAP has 30 days to find new sponsor
  │   └─ Default sponsor: Steering committee (until TRUSTED finds sponsor)
  └─ Commission: 1.5x base rate

        ↓ After 1 year ≥98% SLA + 250+ nodes ↓

MASTER Tier (Year 2+):
  ├─ Leadership tier (2-3 operators)
  ├─ Requirements: 250+ nodes, 500M uWork stake
  ├─ SLA target: 99.5% uptime (strict)
  ├─ Authority: Govern operator disputes, set regional policies
  ├─ Sponsorship: Can sponsor multiple TRUSTED operators
  └─ Commission: 2.0x base rate
```

**Federation Policies:**

```
Policy Governance:
  - BOOTSTRAP: No governance rights (steering committee decides)
  - TRUSTED: Can propose regional policies (GPU pricing, storage tiers)
  - MASTER: Approve policies, arbitrate disputes

Dispute Resolution:
  1. Operator A claims unfair SLA measurement by Operator B
  2. Escalate to MASTER operators (3-of-3 vote)
  3. MASTER operators review evidence (Decentralized.Host audit logs)
  4. Decision: 2-of-3 majority vote (B pays credits to A or not)
  5. Decision published on operator dashboard

Operator Demotion:
  - TRUSTED → BOOTSTRAP: 30 consecutive days <98% SLA
  - BOOTSTRAP → Suspended: 30 consecutive days <95% SLA
  - Recovery: Cure SLA breach within 14 days to restore tier
  - Permanent suspension: 90 consecutive days below tier target
```

**Inter-Operator Accounting:**

```
Sponsorship Revenue Model:

Month 1 (BOOTSTRAP Operator earning):
  BOOTSTRAP-A (Alice's company):
    - Node revenue: 1,000 vCPU-hours × $0.05 = $50
    - SLA bonus (98% uptime): 1.05x multiplier = +$2.50
    - Gross: $52.50
    
  TRUSTED-B (Bob's company - sponsors Alice):
    - Alice's revenue: $52.50
    - Sponsor commission: 5% of $52.50 = $2.625
    - Bob receives: $2.625 monthly
    - Alice net: $52.50 - $2.625 = $49.875

Sponsorship Settlement:
  - Alice's stake (100M uWork) held in Bob's wallet
  - Monthly: Transfer commission to Bob's account
  - Annual: Alice can request full sponsorship release (finds new sponsor)
  - Default: Steering committee can revoke sponsorship if dispute unresolved
```

---

### Global Load Balancing (Phase 7D)

**Anycast Routing Architecture:**

```
┌──────────────────────────────────────────────────────────┐
│             Global Load Balancer (Anycast IP)            │
│                                                          │
│  Workload IP: 203.0.113.42 (same in all 3 regions)      │
│                                                          │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────┐ │
│  │  US-WEST LB    │  │  EU-CENTRAL LB │  │ ASIA-EAST │ │
│  │  203.0.113.42  │  │  203.0.113.42  │  │203.0.113.42│ │
│  │                │  │                │  │            │ │
│  │ Primary:       │  │ Replica 1:     │  │ Replica 2: │ │
│  │ node-42        │  │ node-85        │  │ node-203   │ │
│  └────────────────┘  └────────────────┘  └────────────┘ │
│         ↑                    ↑                    ↑       │
│         │ Client from       │ Client from       │        │
│         │ US routes here    │ EU routes here    │ Client │
│         │                   │                   │ from   │
│         └───────────────────┴───────────────────┘ ASIA  │
│                                                          │
│  BGP Anycast: Lowest latency path (nearest replica)      │
└──────────────────────────────────────────────────────────┘
```

**Read/Write Routing:**

```
Read Request (from client):
  1. BGP routing: Client traffic → nearest LB (lowest latency)
  2. LB: Check local replica health
  3. If healthy: Serve from local replica (read-only)
  4. If unhealthy: Fail over to next-nearest replica
  5. Latency: <10ms (local) + <100ms (cross-region)

Write Request (from client):
  1. BGP routing: Client traffic → nearest LB
  2. LB: Route write to primary (US-WEST)
  3. Primary: Accept write, replicate to replicas
  4. LB: Return SUCCESS to client
  5. Latency: <1s (replication time)
  
Consistency: Read-your-writes guaranteed
```

**Failover Routing:**

```
Primary Region Failure (US-WEST down):

T=0:00  Client writes to 203.0.113.42 → routes to US-WEST LB
T=0:10  US-WEST LB doesn't respond
T=0:20  BGP updates: Route 203.0.113.42 → EU-CENTRAL LB
T=0:30  Client traffic redirected to EU-CENTRAL
T=0:35  EU-CENTRAL LB becomes primary for writes
T=0:40  Clients in EU write to local LB (fast)
T=0:45  Clients in US/ASIA: Write still works (cross-region, slower)

Automatic Recovery:
  1. US-WEST comes back online
  2. New primary (EU-CENTRAL) detected by old primary
  3. Old primary fetches missed writes (Merkle sync)
  4. After sync: BGP weight rebalancing (re-prefer US-WEST if better SLA)
  5. No client-side changes needed
```

**Cost Calculation:**

```
Single-Region Workload:
  - Compute: 2 vCPU × 730 hours × $0.05 = $73
  - Memory: 4 GB × 730 hours × $0.02 = $58.40
  - Storage: 100 GB × 730 hours × $0.001 = $7.30
  - Total: $138.70 / month

Multi-Region Workload (3-region):
  - Primary (US-WEST): 100% cost = $138.70
  - Replica 1 (EU-CENTRAL, secondary): 80% cost = $110.96
  - Replica 2 (ASIA-EAST, tertiary): 60% cost = $83.22
  - Total: $332.88 / month (2.4x base, pays for availability)

Cost Optimization:
  - User can select: 1 region (single), 2 regions, or 3+ regions
  - Pricing shown upfront before placement
  - Higher availability cost justified by uptime guarantees
```

---

### Advanced SLA Management (Phase 7E)

**Multi-Region SLA Calculation:**

```
Per-Region SLA:
  - US-WEST uptime: 98.5%
  - EU-CENTRAL uptime: 99.2%
  - ASIA-EAST uptime: 97.8%

Global SLA (any region down = outage):
  - Global uptime: 98.5% × 99.2% × 97.8% = 95.0%
  - BUT: Replicas survive regional failure
  - Adjusted: 99.5% (replica continuity prevents global outage)

Measurement Window: Rolling 30 days
  - Day 1-30: Initial measurement
  - Day 31: Drop day 1, add day 31
  - Continue daily rolling calculation

SLA Tiers:
  - BOOTSTRAP: ≥95% global uptime → keep tier (else demotion)
  - TRUSTED: ≥98% global uptime → maintain tier, progress to MASTER
  - MASTER: ≥99.5% global uptime → maintain tier

SLA Credits (auto-issued):
  - If 90% < actual < 95%: 10% credit
  - If 85% < actual ≤ 90%: 25% credit
  - If 80% < actual ≤ 85%: 50% credit
  - If actual ≤ 80%: 100% credit (full month free)
```

**Verification & Attestation:**

```
Monthly SLA Attestation:

Week 1: Decentralized.Host measures SLA
  - Aggregate uptime from monitoring data
  - Compare to tier targets

Week 2: Third-party auditor verifies
  - External firm randomly audits 10% of measurements
  - Re-measure sample of operators
  - Issue audit report

Week 3: Results published
  - SLA dashboard shows per-operator uptime
  - Credits calculated and issued
  - Dispute window opens (14 days)

Week 4: Dispute resolution (if needed)
  - Operator claims measurement error
  - Independent re-measurement by auditor
  - Decision published within 7 days
```

**Tier Progression Automation:**

```
Monthly Tier Check:

BOOTSTRAP Operator (Alice):
  1. Measure: 95.2% uptime over last 90 days → ≥95% ✓
  2. Check: 120 nodes operated → ≥100+ nodes ✓
  3. Conditions: Passed all checks
  4. Automatic action: Promote to TRUSTED
  5. New tier: TRUSTED effective immediately
  6. Notification: Email + dashboard update
  7. Commission rate: Increase from 1.0x to 1.5x (effective next month)

TRUSTED Operator (Bob):
  1. Measure: 98.1% uptime over last 1 year → ≥98% ✓
  2. Check: 280 nodes operated → ≥250+ nodes ✓
  3. Conditions: Passed all checks
  4. Automatic action: Promote to MASTER
  5. New tier: MASTER effective immediately
  6. Authority granted: Can now approve policies, arbitrate disputes
  7. Commission rate: Increase from 1.5x to 2.0x

Demotion Check (TRUSTED → BOOTSTRAP):
  1. Measure: 97.2% uptime over last 30 days → <98% ✓
  2. Degradation: 14 consecutive days <98%
  3. Warning issued: "Below TRUSTED tier SLA for 14 days"
  4. Recovery window: 14 days to cure (reach ≥98% again)
  5. If not cured by day 28: Demoted to BOOTSTRAP
  6. Recovery: Must achieve ≥98% for 90 consecutive days to re-promote
```

---

## Integration with Phase 6

| Phase 6 Component | Phase 7 Extension | Implementation |
|------------------|------------------|----------------|
| Topology Manager | Regional topology service | Add region field, region-aware lookups |
| Large-Scale Scheduler | Global load balancer | Route to nearest replica |
| Delta Snapshots | Cross-region replication | Sync replicas via delta snapshots |
| GPU Scheduling | Regional GPU pools | Tag GPUs by region, respect affinity |
| StatefulSets | Cross-region persistence | Bind replicas by region order |
| RBAC | Federation policies | Add tier-based permissions (can sponsor, can arbitrate) |
| Audit Logging | Cross-region consolidation | Aggregate logs by region, chain hashes |
| Tracing | Cross-region span collection | Propagate trace ID across regions |
| Metrics | Global aggregation | Sum latencies per region, aggregate uptime |
| Alerting | Multi-region correlation | Detect region-wide vs operator-specific issues |
| Pricing | Region-aware multipliers | Base × regional multiplier × replication factor |
| Billing | Inter-operator settlements | Track sponsor commission, tier-based rates |

---

## Implementation Roadmap

### Phase 7A: Multi-Region Control Plane (Weeks 1-4)

**Week 1-2: Raft Infrastructure**
- [ ] Deploy 5-node Raft cluster per region
- [ ] Persistent volume per node (etcd-like)
- [ ] Leader election mechanism
- [ ] Consensus log replication
- [ ] Tests: 100+ Raft-specific tests

**Week 3: Global State Replication**
- [ ] Global operator state replication
- [ ] Cross-region log gossip
- [ ] Conflict resolution (last-write-wins + timestamp)
- [ ] Tests: Cross-region replication tests

**Week 4: Failure Handling**
- [ ] Leader failure detection + election
- [ ] Network partition handling
- [ ] Recovery procedures
- [ ] Tests: Chaos scenarios (kill leader, partition regions)

**Deliverables:**
- [ ] 5-node Raft clusters deployed (3+ regions)
- [ ] 99.9% control plane uptime
- [ ] <100ms p95 global operation latency
- [ ] Zero data loss on leader failure

---

### Phase 7B: Cross-Region Replication (Weeks 5-10)

**Week 5-6: Replica Placement**
- [ ] Replica placement algorithm (no colocation)
- [ ] Replica selection based on latency
- [ ] User-configurable distribution
- [ ] Tests: Placement tests (1000+ workloads)

**Week 7-8: Replication Protocol**
- [ ] Delta snapshot mechanism
- [ ] Quorum read implementation
- [ ] Replica acknowledgment tracking
- [ ] Tests: Replication protocol tests

**Week 9-10: Failover & Recovery**
- [ ] Failover detection (<10s)
- [ ] New primary election
- [ ] Merkle anti-entropy
- [ ] Tests: Failover chaos (kill primary, measure recovery)

**Deliverables:**
- [ ] Primary + 2 replicas working
- [ ] <30s failover latency
- [ ] <1s replication latency p99
- [ ] Zero data loss on replica failure

---

### Phase 7C: Operator Federation (Weeks 11-16)

**Week 11-12: Tier Progression Gates**
- [ ] 90-day timer for BOOTSTRAP→TRUSTED
- [ ] 1-year timer for TRUSTED→MASTER
- [ ] Automatic promotion/demotion
- [ ] SLA requirement checks
- [ ] Tests: Tier progression tests (simulate 90-day period)

**Week 13-14: Sponsorship Model**
- [ ] Sponsorship wallet mechanics
- [ ] Revenue sharing calculation
- [ ] Sponsor revocation handling
- [ ] Tests: Sponsorship tests (transfer, revocation)

**Week 15-16: Federation Policies**
- [ ] MASTER policy approval workflow
- [ ] Dispute resolution mechanism
- [ ] Policy enforcement per-region
- [ ] Tests: Policy tests (e.g., GPU pricing regional variation)

**Deliverables:**
- [ ] Tier progression working automatically
- [ ] Sponsorship revenue flowing
- [ ] Federation policies enforced
- [ ] Dispute resolution operable

---

### Phase 7D: Global Load Balancing (Weeks 17-24)

**Week 17-18: Anycast Infrastructure**
- [ ] Anycast IP assignment per workload
- [ ] BGP route announcement
- [ ] Load balancer per region
- [ ] Tests: Anycast routing tests (clients from 3 regions)

**Week 19-20: Read/Write Routing**
- [ ] Read routing to nearest replica
- [ ] Write routing to primary
- [ ] Health probe per replica
- [ ] Tests: Routing tests (read latency, write consistency)

**Week 21-22: Failover Routing**
- [ ] Automatic BGP failover on primary region down
- [ ] Client transparency (no reconnect needed)
- [ ] Gradual weight shifting (avoid storm)
- [ ] Tests: Failover tests (kill primary region, measure client impact)

**Week 23-24: Performance & Cost**
- [ ] Latency optimization (reduce cross-region RTT)
- [ ] Bandwidth optimization (delta snapshots)
- [ ] Cost attribution per region
- [ ] Tests: Load tests (1000 cross-region workloads)

**Deliverables:**
- [ ] Anycast IP routing working
- [ ] <10ms local reads, <1s writes
- [ ] Automatic failover on region down
- [ ] Cost accurately calculated per region

---

### Phase 7E: Advanced SLA Management (Weeks 25-30)

**Week 25-26: SLA Calculation**
- [ ] Global uptime calculation
- [ ] Per-region uptime tracking
- [ ] Tier-based thresholds
- [ ] Tests: SLA calculation tests (96%, 98%, 99.5%)

**Week 27: SLA Credits**
- [ ] Auto-credit issuance on SLA miss
- [ ] Credit amount calculation
- [ ] Credit application to next month's bill
- [ ] Tests: Credit tests (various SLA scenarios)

**Week 28: Tier Progression Automation**
- [ ] Automatic BOOTSTRAP→TRUSTED promotion
- [ ] Automatic TRUSTED→MASTER promotion
- [ ] Automatic demotion (SLA miss)
- [ ] Tests: Promotion/demotion tests

**Week 29-30: Verification & Attestation**
- [ ] Third-party SLA auditor integration
- [ ] SLA dashboard publication
- [ ] Dispute resolution workflow
- [ ] Tests: End-to-end SLA verification

**Deliverables:**
- [ ] SLA calculated accurately (rolling 30-day)
- [ ] Credits issued automatically
- [ ] Tier progression automatic (no manual approval)
- [ ] Third-party attestation operable

---

## Deployment Architecture

### Kubernetes-Native Deployment

```yaml
# Phase 7A: Multi-Region Control Plane
apiVersion: v1
kind: Namespace
metadata:
  name: dh-control-plane
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  namespace: dh-control-plane
  name: raft-cluster-us-west
spec:
  serviceName: raft-us-west
  replicas: 5
  selector:
    matchLabels:
      app: raft-us-west
  template:
    metadata:
      labels:
        app: raft-us-west
        region: us-west
    spec:
      containers:
      - name: raft
        image: dh-control-plane:7.0
        ports:
        - containerPort: 2379 (Raft consensus)
        - containerPort: 2380 (Raft replication)
        volumeMounts:
        - name: raft-state
          mountPath: /var/lib/dh-raft
  volumeClaimTemplates:
  - metadata:
      name: raft-state
    spec:
      accessModes: [ "ReadWriteOnce" ]
      storageClassName: "fast-ssd"
      resources:
        requests:
          storage: 100Gi

---
# Phase 7B: Cross-Region Replication
apiVersion: v1
kind: Service
metadata:
  namespace: dh-control-plane
  name: workload-replicator
spec:
  type: LoadBalancer
  selector:
    app: workload-replicator
  ports:
  - port: 5000
    targetPort: 5000

---
# Phase 7D: Global Load Balancer
apiVersion: v1
kind: Service
metadata:
  name: workload-anycast
  annotations:
    # Anycast IP: 203.0.113.42 (same in all regions)
    external-ip: "203.0.113.42"
spec:
  type: LoadBalancer
  selector:
    app: workload-lb
  ports:
  - port: 443
    targetPort: 443
    protocol: TCP
```

---

## Testing Strategy

### Unit Tests (Per Component)
- [ ] Raft consensus: 50+ tests (leader election, log replication)
- [ ] Replica placement: 30+ tests (no colocation, latency optimization)
- [ ] SLA calculation: 40+ tests (various uptime scenarios)
- [ ] Tier progression: 20+ tests (90-day window, promotion/demotion)

### Integration Tests (Cross-Component)
- [ ] End-to-end workload placement across 3 regions (100 workloads)
- [ ] Write to primary, read from replicas (consistency verification)
- [ ] Failover detection and recovery (<30s latency)
- [ ] Tier progression over 90-day simulation

### Chaos Tests (Failure Scenarios)
- [ ] Kill primary region control plane (measure failover latency)
- [ ] Kill replica node (measure data loss with quorum reads)
- [ ] Network partition (region A isolated from B & C)
- [ ] Delayed replication (>1s latency on delta snapshots)
- [ ] BGP route flapping (simulate provider network issues)

### Load Tests
- [ ] 1,000 workloads across 3 regions (measure SLA impact)
- [ ] 10,000 cross-region reads/sec (verify latency targets)
- [ ] 1,000 cross-region writes/sec (verify replication throughput)
- [ ] 72-hour sustained operation (stability)

---

## Success Criteria

### Functional
- [ ] All 5 phases (7A-7E) implemented and integrated
- [ ] 500+ integration tests passing
- [ ] 90% code coverage (security-critical paths 100%)
- [ ] No critical security findings in external audit
- [ ] Zero data loss incidents

### Performance
- [ ] <100ms p95 global operation latency (control plane)
- [ ] <10ms local read latency (same region)
- [ ] <1s write latency p99 (cross-region replication)
- [ ] <30s failover latency (primary region failure)
- [ ] <1 hour Merkle anti-entropy detection

### Operational
- [ ] 99.9% control plane uptime over 30-day measurement
- [ ] 99.95% workload availability (with replication)
- [ ] 5-20 operators successfully operating
- [ ] 2000+ nodes operational across 3+ regions
- [ ] Zero unplanned data loss

### Business
- [ ] $500K+ annual operational revenue (conservative)
- [ ] 5+ BOOTSTRAP tier operators
- [ ] 1-2 TRUSTED tier operators (graduated)
- [ ] Cost per workload-hour competitive vs centralized platforms

---

## Risk Mitigation

| Risk | Mitigation | Residual Risk |
|------|-----------|---------------|
| Raft consensus bugs | Formal verification, extensive testing | Low |
| Cross-region latency > target | Deploy in low-latency zones (AWS AZs) | Low |
| Replica divergence | Merkle anti-entropy + hourly detection | Very Low |
| Operator adoption slow | Phased rollout (BOOTSTRAP only initially) | Medium |
| BGP failover glitches | Pre-production testing, fallback manual failover | Low |
| SLA calculation disputes | Third-party auditor verification, public dashboard | Medium |

---

## Glossary

- **Raft:** Consensus algorithm for distributed state machine replication
- **Quorum:** Majority subset (e.g., 3-of-5 nodes)
- **Replica:** Workload copy in secondary region
- **Failover:** Transition to secondary primary on failure
- **RPO:** Recovery Point Objective (max data loss allowed)
- **RTO:** Recovery Time Objective (max time to restore service)
- **Anycast:** Single IP advertised from multiple regions (BGP)
- **Merkle Anti-Entropy:** Background synchronization using hash trees
- **Delta Snapshot:** Incremental update (only changes since last snapshot)

---

**Phase 7 Architecture Design - Decentralized.Host**  
**Prepared:** 2026-10-04  
**Status:** Ready for Technical Review & Approval  
**Next Step:** Approval by Oct 8, Implementation begins Week of Oct 17

