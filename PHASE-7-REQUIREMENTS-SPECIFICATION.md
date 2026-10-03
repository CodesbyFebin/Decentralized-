# Phase 7 Requirements Specification

**Document Version:** 1.0  
**Date:** 2026-10-03  
**Status:** Ready for Architecture Team Review  
**Timeline:** Oct 4-8, 2026 (5 days)  
**Target Approval:** Oct 8, 2026

---

## Executive Summary

Phase 7 extends Phase 6 capabilities for production-scale multi-region operation and strategic operator growth. Phase 7 focuses on:

1. **Multi-Region Control Plane** - Distributed consensus across 3+ regions
2. **Cross-Region Workload Replication** - Automatic failover and redundancy
3. **Operator Federation** - Tier progression and inter-operator policies
4. **Global Load Balancing** - Intelligent workload distribution across regions
5. **Advanced SLA Management** - Multi-region compliance tracking

**Phase 6 Scope:** Single-region, 450 nodes, 1-3 operators  
**Phase 7 Scope:** Multi-region (3-8 regions), 2000+ nodes, 5-20 operators  
**Timeline:** 6-12 month implementation roadmap

---

## Dependency Analysis: Phase 6 → Phase 7

### Phase 6 Capabilities Required

**Scaling Foundations (6A):**
- [x] Topology Manager (450 node baseline) - extends to 2000+ nodes
- [x] Large-Scale Scheduler (<15ms latency) - must hold under cross-region load
- [x] Delta Snapshots (88% reduction) - basis for cross-region replication

**Advanced Workloads (6B):**
- [x] GPU Scheduling - extends to heterogeneous regions (GPU availability varies)
- [x] StatefulSets - requires cross-region persistence guarantees
- [x] Storage Classes - multi-region tier provisioning

**Governance (6C):**
- [x] RBAC (4 roles) - extends to federation policies
- [x] Audit Logging (BLAKE3 chain) - cross-region log consolidation
- [x] Multi-Tenancy - isolation across regions

**Observability (6D):**
- [x] Distributed Tracing - cross-region span collection
- [x] Metrics (18 Prometheus) - multi-region aggregation
- [x] Alerting - cross-region anomaly detection
- [x] SLA Monitoring - global SLA calculation

**Marketplace & Billing (6E):**
- [x] Dynamic Pricing - region-aware pricing multipliers
- [x] Lease Tracking - precise cross-region timing
- [x] Settlement - inter-operator payment clearance

**Production Hardening:**
- [x] TLS 1.3 + mTLS - regional mesh
- [x] Ed25519 Key Rotation - synchronized across regions
- [x] Rate Limiting - global quota allocation
- [x] Input Validation - uniform enforcement
- [x] Audit Trail - immutable cross-region logs

---

## Functional Requirements

### FR-1: Multi-Region Control Plane

**FR-1.1: Distributed Raft Consensus**
- [ ] 5-node minimum Raft cluster per region (for fault tolerance)
- [ ] Cross-region consensus for global state (operator registrations, policies)
- [ ] Regional quorum + global quorum (2 of 3 regions + majority within region)
- [ ] Sub-100ms consensus latency (p99) for global operations
- [ ] Automatic leader election on regional failure

**FR-1.2: Regional State Partitioning**
- [ ] Region-local state (node registrations, workload placements)
- [ ] Global state (operator profiles, policy rules, SLA definitions)
- [ ] Eventual consistency guarantee (all regions converge to same global state within 5 minutes)
- [ ] Conflict resolution (last-write-wins with timestamp binding)

**FR-1.3: Control Plane Deployment**
- [ ] Kubernetes StatefulSet deployment (headless service for Raft)
- [ ] Persistent volume for Raft state (etcd-like guarantees)
- [ ] Multi-AZ deployment (failure domain isolation)
- [ ] Health probes (liveness: reachable, readiness: quorum membership)

**FR-1.4: Control Plane Security**
- [ ] mTLS between control plane nodes (encrypted cross-region)
- [ ] Ed25519 key binding per control plane node
- [ ] Rate limiting on consensus messages (prevent DDoS)
- [ ] Audit log of all consensus operations

### FR-2: Cross-Region Workload Replication

**FR-2.1: Replica Placement Strategy**
- [ ] Primary + 2 replicas by default (3-region minimum deployment)
- [ ] User-configurable replica count (1-5 replicas)
- [ ] No colocation on same host (ensures failure independence)
- [ ] User-selectable replica distribution (e.g., all in primary region, or 1 per region)

**FR-2.2: Replication Protocol**
- [ ] Write goes to primary (blocking until replicated)
- [ ] Replication timeout: 5 seconds (fallback: write to fewer replicas if needed)
- [ ] Quorum reads (read from 2 of 3 replicas, get majority)
- [ ] Read-your-writes consistency (single client always sees their writes)

**FR-2.3: Replica Synchronization**
- [ ] Delta snapshots (88% bandwidth reduction from Phase 6A)
- [ ] Merkle tree anti-entropy (background sync if replicas diverge)
- [ ] Periodic full snapshot (weekly baseline)
- [ ] Incremental updates between snapshots (daily)

**FR-2.4: Failover & Recovery**
- [ ] Automatic failover (if primary unresponsive for 10 seconds)
- [ ] New primary elected from replica set (quorum among remaining)
- [ ] Data loss detection (compare data versions, alert if divergent)
- [ ] Manual recovery (operator can force-sync replica from primary)

### FR-3: Operator Federation

**FR-3.1: Tier Progression Gates**
- [ ] BOOTSTRAP → TRUSTED (90+ days, 95% SLA, 100+ nodes)
- [ ] TRUSTED → MASTER (1+ year, 98% SLA, 250+ nodes)
- [ ] Automatic progression (no steering committee approval needed if gates met)
- [ ] Tier demotion (if SLA drops below threshold for 30 consecutive days)

**FR-3.2: Federation Policies**
- [ ] TRUSTED operators can sponsor BOOTSTRAP operators (1:1 ratio limit)
- [ ] MASTER operators govern operator disputes (3-of-5 majority)
- [ ] MASTER operators set regional policies (e.g., GPU pricing, storage tiers)
- [ ] Cross-operator workload placement (workload owner can specify operator affinity)

**FR-3.3: Inter-Operator Accounting**
- [ ] Operator A sponsors Operator B (B's stake locked in A's wallet)
- [ ] Revenue sharing (A gets 5% commission on B's earnings during sponsorship)
- [ ] Sponsorship revocation (A can revoke, B has 30 days to find new sponsor)
- [ ] Default sponsorship pool (steering committee sponsors unreferred operators)

**FR-3.4: Operator Dashboard Enhancements**
- [ ] Regional breakdown (revenue per region, workload count per region)
- [ ] Tier status (current tier, time to next tier, SLA progress)
- [ ] Sponsorship tree (if TRUSTED: who they sponsor, who sponsors them)
- [ ] Competitive leaderboard (top operators by SLA, revenue, node count)

### FR-4: Global Load Balancing

**FR-4.1: Workload Placement Across Regions**
- [ ] Operator preference (user can specify preferred operator/region)
- [ ] Cost optimization (algorithm minimizes multi-region replication cost)
- [ ] Latency optimization (place near user-specified "primary" region)
- [ ] Availability optimization (distribute replicas across failure domains)

**FR-4.2: Cross-Region Routing**
- [ ] Workload entry point (load balancer in primary region)
- [ ] Read routing (send reads to nearest replica)
- [ ] Write routing (send writes to primary, wait for replication)
- [ ] Failover routing (if primary region fails, route to secondary)

**FR-4.3: Global Load Balancer Mechanics**
- [ ] Anycast IP per workload (same IP in all regions, routed to nearest)
- [ ] Health probes per replica (continuous uptime monitoring)
- [ ] Traffic shaping (gradually shift load during region failover)
- [ ] Client connection migration (persistent connections survive region failover)

**FR-4.4: Cost Calculation**
- [ ] Replication cost = (replica_count - 1) × storage_cost × per_region_multiplier
- [ ] Primary region = 1.0x, secondary = 0.8x, tertiary = 0.6x (incentivize consolidation)
- [ ] Total cost = (workload_compute_cost × instances) + (replication_cost)
- [ ] Pricing shown upfront before placement

### FR-5: Advanced SLA Management

**FR-5.1: Multi-Region SLA Calculation**
- [ ] Regional uptime (measure per-region availability independently)
- [ ] Global uptime (measure end-to-end availability across regions)
- [ ] SLA tier thresholds:
  - BOOTSTRAP: 95% global uptime
  - TRUSTED: 98% global uptime
  - MASTER: 99.5% global uptime
- [ ] SLA measurement window (rolling 30-day calculation)

**FR-5.2: SLA Compliance Tracking**
- [ ] Per-operator regional breakdown (uptime % by region)
- [ ] Workload-level SLA tracking (per workload uptime)
- [ ] SLA credits (if uptime < threshold, customer gets credit)
- [ ] Credit calculation: (100% - achieved%) × workload_cost × 30%

**FR-5.3: SLA Verification**
- [ ] Independent measurement (Decentralized.Host measures, not operator)
- [ ] Third-party attestation (monthly audit report from external firm)
- [ ] Dispute resolution (if operator disputes SLA, independent re-measurement)
- [ ] Public SLA dashboard (operators' SLA metrics published)

**FR-5.4: SLA-Based Tier Progression**
- [ ] BOOTSTRAP to TRUSTED: 90 days of ≥95% uptime
- [ ] TRUSTED to MASTER: 1 year of ≥98% uptime + 250+ nodes
- [ ] Automatic demotion: 30 consecutive days of <95% (BOOTSTRAP) / <98% (TRUSTED)
- [ ] Remediation period: 14 days to cure SLA breach before demotion

---

## Non-Functional Requirements

### NFR-1: Performance

**NFR-1.1: Control Plane Latency**
- [ ] Global operation latency: <100ms p95 (e.g., operator registration)
- [ ] Regional operation latency: <15ms p95 (e.g., workload placement)
- [ ] No degradation under 10K operators, 1M workloads

**NFR-1.2: Replication Latency**
- [ ] Replica synchronization: <1s p99 (Delta snapshot)
- [ ] Failover detection: <10s (health probe + consensus)
- [ ] Failover completion: <30s (new primary elected, client redirected)

**NFR-1.3: Global Load Balancer Throughput**
- [ ] 100,000+ workloads per second (read requests)
- [ ] 10,000+ workloads per second (write requests)
- [ ] Sub-millisecond routing decision latency

### NFR-2: Availability

**NFR-2.1: Control Plane Availability**
- [ ] 99.9% availability (max 8.76 hours/year downtime)
- [ ] Automatic failover (no manual intervention needed)
- [ ] Regional recovery: auto-heal within 1 hour of regional failure

**NFR-2.2: Workload Availability**
- [ ] 99.95% baseline (with 3-replica configuration)
- [ ] SLA tiers: 95% (BOOTSTRAP), 98% (TRUSTED), 99.5% (MASTER)
- [ ] Cross-region failover: max 30s outage during region failure

**NFR-2.3: Data Availability**
- [ ] RPO (Recovery Point Objective): <1s (all replicas sync within 1 second)
- [ ] RTO (Recovery Time Objective): <30s (workload operational on new primary)

### NFR-3: Scalability

**NFR-3.1: Topology Scaling**
- [ ] 2000+ nodes (8 regions × 250 nodes/region minimum)
- [ ] 100+ operators (independent entities)
- [ ] O(1) lookup for operator info, O(log n) for node discovery

**NFR-3.2: Raft Consensus Scaling**
- [ ] 5-node clusters per region (not per operator)
- [ ] 3+ regions (global quorum: 2-of-3 + local majority)
- [ ] No operator-specific control plane nodes (shared infrastructure)

**NFR-3.3: State Storage**
- [ ] 1TB total state (operator profiles, workload metadata, SLA history)
- [ ] 100GB per region (local state)
- [ ] Snapshot size: <10GB per region (allows restore in <5 min)

### NFR-4: Security

**NFR-4.1: Cryptography**
- [ ] TLS 1.3 minimum (control plane <> region communication)
- [ ] Ed25519 per-region keys (90-day rotation)
- [ ] BLAKE3 cross-region log chain (detect tampering)

**NFR-4.2: Access Control**
- [ ] mTLS authentication (every region node presents cert)
- [ ] Cross-region RBAC (operator can only access own workloads across regions)
- [ ] Audit logging (every cross-region operation logged)

**NFR-4.3: Isolation**
- [ ] Regional isolation (region A cannot interfere with region B)
- [ ] Operator isolation (operator A cannot see operator B's workloads)
- [ ] Network isolation (firewall rules enforce segmentation)

### NFR-5: Observability

**NFR-5.1: Distributed Tracing**
- [ ] Cross-region span collection (trace a workload placement across 3 regions)
- [ ] Trace ID propagation (consistent trace ID through all regions)
- [ ] Span sampling (1% of traces sampled, all production incidents traced)

**NFR-5.2: Metrics**
- [ ] Per-region metrics (throughput, latency, errors per region)
- [ ] Global aggregation (sum/average across regions)
- [ ] Cardinality: <100M time series (8 regions × 10K operators × 1000 metrics)

**NFR-5.3: Alerting**
- [ ] Multi-region anomaly detection (detect region-specific issues)
- [ ] Correlation rules (if region A + region B both degrade, it's a network issue)
- [ ] Escalation rules (critical alerts to on-call engineer)

### NFR-6: Reliability

**NFR-6.1: Data Integrity**
- [ ] BLAKE3 hash chain ensures audit logs tamper-evident
- [ ] Merkle tree anti-entropy (detect data divergence within 1 hour)
- [ ] Quorum reads (always consistent)

**NFR-6.2: Consensus Robustness**
- [ ] Split-brain prevention (quorum-based write)
- [ ] Network partition handling (minority partition stops accepting writes)
- [ ] Byzantine fault detection (not required for Phase 7; Phase 8+ only)

---

## Integration Points with Phase 6

| Phase 6 Component | Phase 7 Extension | New Requirements |
|------------------|------------------|------------------|
| Topology Manager | Regional topology service | Multi-region node discovery, region-aware affinity |
| Scheduler | Global load balancer | Cross-region placement cost optimization |
| Delta Snapshots | Cross-region replication | Replica synchronization, failover |
| GPU Scheduling | Regional GPU pools | GPU availability per region, preference routing |
| StatefulSets | Cross-region persistence | Primary + replica binding, failover ordering |
| RBAC | Federation policies | Operator sponsorship, tier-based permissions |
| Audit Logging | Cross-region log consolidation | Global audit chain, per-region segment signing |
| Tracing | Cross-region span collection | Trace ID propagation, regional sampling |
| Metrics | Global metrics aggregation | Per-region cardinality limits, correlation analysis |
| Alerting | Multi-region alerting | Cross-region anomaly detection, escalation rules |
| Pricing | Region-aware pricing | Multi-region replication multiplier, regional rates |
| Billing | Cross-region settlement | Inter-operator payments, tier-based commission |

---

## Functional Dependencies (Implementation Order)

### Phase 7A: Multi-Region Control Plane (Weeks 1-4)
**Deliverables:**
- [ ] Raft cluster per region (5-node minimum)
- [ ] Global state replication (operator profiles, policies)
- [ ] Consensus failure handling (leader election, partition tolerance)
- [ ] 99.9% control plane uptime verified

**Dependencies:** Phase 6C (RBAC), Phase 6D (audit logging)

### Phase 7B: Cross-Region Workload Replication (Weeks 5-10)
**Deliverables:**
- [ ] Replica placement algorithm (no colocation)
- [ ] Delta snapshot replication (88% bandwidth reduction)
- [ ] Automatic failover (primary election from replicas)
- [ ] Merkle anti-entropy (background sync)

**Dependencies:** Phase 7A (multi-region control plane), Phase 6A (delta snapshots)

### Phase 7C: Operator Federation (Weeks 11-16)
**Deliverables:**
- [ ] Tier progression gates (90-day, 1-year timelines)
- [ ] Sponsorship model (TRUSTED sponsor BOOTSTRAP)
- [ ] Revenue sharing (5% commission for sponsor)
- [ ] Federation policies (MASTER operator governance)

**Dependencies:** Phase 7A, Phase 7B, Phase 6E (billing)

### Phase 7D: Global Load Balancing (Weeks 17-24)
**Deliverables:**
- [ ] Anycast IP per workload (same IP in all regions)
- [ ] Read routing (send to nearest replica)
- [ ] Write routing (send to primary)
- [ ] Failover routing (automatic on primary failure)

**Dependencies:** Phase 7A, Phase 7B

### Phase 7E: Advanced SLA Management (Weeks 25-30)
**Deliverables:**
- [ ] Multi-region SLA calculation (global + regional breakdown)
- [ ] Compliance tracking (rolling 30-day window)
- [ ] SLA credits (automatic credit on failure)
- [ ] Tier progression automation (no manual approval needed)

**Dependencies:** Phase 7A-D, Phase 6D (observability)

---

## Success Criteria

### Technical Acceptance
- [ ] All 50+ requirements implemented and tested
- [ ] 500+ integration tests (covering Phase 7A-E)
- [ ] 90% code coverage (security-critical paths 100%)
- [ ] No critical security findings in external audit
- [ ] 99.9% control plane uptime over 30-day measurement
- [ ] <100ms p95 global operation latency
- [ ] <30s failover completion latency

### Operational Acceptance
- [ ] 5-20 operators successfully operating
- [ ] Multi-region deployment in 3+ regions
- [ ] 2000+ nodes operational
- [ ] 95%+ operator satisfaction score
- [ ] Zero unplanned data loss incidents

### Business Acceptance
- [ ] $100K+ monthly operational revenue (conservative estimate)
- [ ] 5+ BOOTSTRAP tier operators
- [ ] 1-2 TRUSTED tier operators (graduated from BOOTSTRAP)
- [ ] Competitive advantage vs. centralized platforms (cost + sovereignty)

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Control Plane Uptime | 99.9% | Automated health check, SLA tracking |
| Global Operation Latency p95 | <100ms | OpenTelemetry tracing + Prometheus |
| Failover Latency | <30s | Synthetic workload + timer |
| Replica Synchronization | <1s p99 | Delta snapshot throughput test |
| Operator Count | 5-20 | Manual headcount + billing system |
| Region Count | 3+ | Topology service |
| Global Node Count | 2000+ | Topology service |
| SLA Compliance Rate | 95%+ | Rolling 30-day calculation |
| Zero Data Loss Incidents | 100% | Incident log audit |
| External Audit Result | No critical issues | Third-party audit sign-off |

---

## Timeline & Effort Estimates

| Phase | Duration | Team Size | Effort (weeks) | Critical Path |
|-------|----------|-----------|----------------|---------------|
| 7A: Control Plane | 4 weeks | 3 engineers | 12 weeks | Raft implementation |
| 7B: Replication | 6 weeks | 4 engineers | 24 weeks | Failover testing |
| 7C: Federation | 6 weeks | 3 engineers | 18 weeks | Policy rules |
| 7D: Load Balancing | 8 weeks | 4 engineers | 32 weeks | Anycast + routing |
| 7E: SLA Management | 6 weeks | 3 engineers | 18 weeks | Compliance tracking |
| **Total** | **6-12 months** | **4-5 engineers** | **~50-60 weeks** | **Parallel execution** |

**Parallelization:** 7A (weeks 1-4) → 7B+7C start (week 4) → 7D (week 10) → 7E (week 16)

**Staffing:** 2-3 core engineers (Raft, scheduling) + 2 specialists (observability, RBAC) + 1 QA

---

## Risks & Mitigations

| Risk | Probability | Mitigation |
|------|-------------|-----------|
| Raft consensus bugs | Medium | Formal verification, extensive testing, external review |
| Cross-region latency > target | Low | Regional deployment in low-latency locations (AWS Availability Zones) |
| Operator adoption slower than expected | Medium | Phased rollout (BOOTSTRAP tier first, TRUSTED month 3) |
| Data loss during failover | Low | Quorum reads, Merkle anti-entropy, 72-hour data loss detection |
| SLA calculation disputes | Medium | Third-party verification, public SLA dashboard, dispute resolution process |

---

## Glossary

- **BOOTSTRAP Tier:** Initial operator tier, 95% SLA, 50+ nodes
- **TRUSTED Tier:** Advanced tier, 98% SLA, 100+ nodes, can sponsor BOOTSTRAP operators
- **MASTER Tier:** Expert tier, 99.5% SLA, 250+ nodes, governs operator disputes
- **Replica:** Workload copy in secondary region
- **Failover:** Automatic transition to secondary primary when primary fails
- **RPO:** Recovery Point Objective (max data loss)
- **RTO:** Recovery Time Objective (max time to restore)
- **Anycast:** Single IP address advertised from multiple regions
- **Quorum:** Majority subset (e.g., 2-of-3 replicas)

---

**Phase 7 Requirements Specification - Decentralized.Host**  
**Prepared:** 2026-10-03  
**Status:** Ready for Architecture Team Review  
**Next Step:** Architecture Design (Oct 5-6), Approval (Oct 8)

