# Operator Certification Exam

**Exam Format:** Written (50 questions) + Practical (90-minute deployment test)  
**Passing Score:** 80% overall (40/50 questions + practical demonstration)  
**Duration:** 2 hours total (1.5 hours written + 0.5 hours practical)  
**Prerequisite:** Completion of all 7 training modules  
**Validity:** Valid for 3 years; recertification required for tier progression

---

## Part 1: Written Exam (50 Questions, 90 minutes)

**Scoring:** 1 point per question | Passing: 40/50 (80%)

### Section A: Architecture & Core Concepts (10 questions)

**A1. Workload Placement Basics**
Q: What is the primary advantage of Decentralized.Host's O(1) topology lookup?
A) Eliminates need for caching in the scheduler
B) Allows placement of workloads in <15ms p95 latency
C) Reduces operator requirements to 10 nodes minimum
D) Prevents GPU scheduling on distributed nodes
*Answer: B*

**A2. Operator Responsibilities**
Q: As a BOOTSTRAP tier operator, which of the following is NOT required?
A) 95% uptime SLA commitment
B) 50+ node minimum infrastructure
C) Governance rights over operator policies
D) Weekly check-ins with onboarding POC
*Answer: C*

**A3. Tier Progression**
Q: After 90 days of ≥95% uptime and 100+ nodes, what tier can a BOOTSTRAP operator progress to?
A) MASTER (directly)
B) TRUSTED
C) Requires steering committee approval (cannot auto-progress)
D) Depends on GPU node count
*Answer: B*

**A4. Multi-Tenancy**
Q: How does Decentralized.Host ensure operator A cannot see operator B's workloads?
A) Firewall rules only
B) RBAC permission model + namespace isolation
C) Operator B doesn't exist (single-tenant architecture)
D) Network segmentation at BGP level
*Answer: B*

**A5. Replication Model**
Q: In a 3-region deployment, how many replicas of a workload are typically maintained?
A) 1 (primary only)
B) 2 (primary + 1 backup)
C) 3 (primary + 2 replicas)
D) Depends on operator preference
*Answer: C*

**A6. SLA Definition (BOOTSTRAP)**
Q: What uptime percentage must a BOOTSTRAP tier operator achieve to maintain their tier?
A) 90% over rolling 30 days
B) 95% over rolling 30 days
C) 98% over rolling 90 days
D) 99.5% over rolling 1 year
*Answer: B*

**A7. Key Rotation**
Q: How often are operator Ed25519 keys rotated in Decentralized.Host?
A) Daily
B) Weekly
C) 90-day enforcement (with overlap period)
D) On-demand only
*Answer: C*

**A8. Consensus & Control Plane**
Q: How many control plane nodes form a quorum for decision-making in Decentralized.Host?
A) 1 (single point of decision)
B) 2-of-3 regions
C) 3-of-5 nodes per region
D) Simple majority (N/2 + 1)
*Answer: B*

**A9. Failover Detection**
Q: What is the typical detection latency if a primary workload region fails?
A) <1 second
B) <10 seconds
C) <30 seconds (after automatic failover)
D) 2-3 minutes (manual investigation time)
*Answer: B*

**A10. Cost Model**
Q: If a workload costs $138.70/month in a single region, what is the approximate cost for 3-region replication with secondary (80%) and tertiary (60%) multipliers?
A) $138.70 (same cost)
B) $208.05 (1.5x)
C) $332.88 (2.4x)
D) $415.95 (3x)
*Answer: C*

---

### Section B: Operations & Support (10 questions)

**B1. Node Monitoring**
Q: What is the minimum network connectivity requirement for operator nodes?
A) 100 Mbps
B) 500 Mbps
C) 1-2 Gbps
D) 10 Gbps
*Answer: C*

**B2. Incident Response SLA**
Q: What is the expected response time for Decentralized.Host to acknowledge a critical support incident?
A) 24 hours
B) <4 hours
C) <1 hour
D) Immediate (real-time)
*Answer: B*

**B3. Infrastructure Failures**
Q: If a node fails, how long does the operator have to replace it to maintain SLA?
A) 24 hours
B) 72 hours
C) 30 days
D) No requirement (Decentralized.Host will automatically migrate workloads)
*Answer: B*

**B4. Stake Deposit**
Q: A BOOTSTRAP operator deposits 100M uWork. For how long is it locked?
A) 30 days
B) 90 days minimum (can request release after)
C) 1 year
D) Until demotion from BOOTSTRAP
*Answer: B*

**B5. Support Channels**
Q: Which of the following is NOT a supported way to contact Decentralized.Host operations?
A) Email: operator-support@decentralized.host
B) Slack: #operator-support (private channel)
C) Direct phone call (no published number)
D) Weekly sync with onboarding POC
*Answer: C*

**B6. Billing Settlement**
Q: How often are operator earnings calculated and settled?
A) Real-time (live billing)
B) Daily
C) Weekly
D) Monthly (net 30 days)
*Answer: D*

**B7. Downtime Reporting**
Q: If a workload is down for 4 hours due to operator maintenance, who files the incident report?
A) Operator files their own report
B) Decentralized.Host automatically detects and logs
C) Workload owner (customer) reports to Decentralized.Host
D) Steering committee investigates
*Answer: B*

**B8. SLA Credits**
Q: If an operator achieves 92% uptime (below 95% BOOTSTRAP target), what automatic credit is issued?
A) 0% (no credit, but warning issued)
B) 5% of monthly bill
C) 10% of monthly bill
D) 100% (full month free)
*Answer: C*

**B9. Security Audit**
Q: When does an operator undergo their first external security audit?
A) Before BOOTSTRAP tier activation (part of qualification)
B) After 90 days in BOOTSTRAP
C) When progressing to TRUSTED
D) Annually (no relation to tier)
*Answer: A*

**B10. Escalation Path**
Q: A workload owner has an SLA dispute with your operator node behavior. What is the escalation path?
A) Workload owner → Your operator (direct negotiation)
B) Workload owner → Decentralized.Host (investigation)
C) Workload owner → MASTER operator (dispute resolution)
D) Workload owner → Steering committee (final arbiter)
*Answer: C*

---

### Section C: Security & Compliance (10 questions)

**C1. TLS Configuration**
Q: What is the minimum TLS version enforced for control plane connections in Phase 6+?
A) TLS 1.0 (legacy support)
B) TLS 1.2 (industry standard)
C) TLS 1.3 (modern, required)
D) No minimum (any version acceptable)
*Answer: C*

**C2. Certificate Pinning**
Q: Why does Decentralized.Host require operators to pin control plane certificates?
A) To reduce certificate authority load
B) To prevent man-in-the-middle attacks
C) To simplify certificate renewal
D) No particular reason (standard practice)
*Answer: B*

**C3. mTLS Mutual Authentication**
Q: In mTLS, who must authenticate? Choose all that apply.
A) Only the client (operator) authenticates to the server
B) Only the server (control plane) authenticates to the client
C) Both client and server present certificates
D) Neither side authenticates (encrypted only)
*Answer: C*

**C4. Audit Logging**
Q: How long must operator activity audit logs be retained?
A) 7 days
B) 30 days
C) 90+ days
D) Forever
*Answer: C*

**C5. BLAKE3 Hash Chain**
Q: What security property does the BLAKE3 hash chain in audit logs provide?
A) Compression (reduces storage size)
B) Tamper detection (changes are detectable)
C) Encryption (unreadable without key)
D) Compression and encryption
*Answer: B*

**C6. Workload Signing**
Q: How must a workload be signed before deployment on an operator's nodes?
A) No signature required (trust control plane)
B) Workload owner's Ed25519 private key signature
C) Operator's Ed25519 private key signature
D) Control plane's RSA-2048 signature
*Answer: B*

**C7. Policy Enforcement**
Q: Where is the workload placement policy enforced: global policy engine or local per-node?
A) Global policy engine (centralized decision)
B) Local per-node (each node enforces independently)
C) Both (global then local)
D) Depends on workload type
*Answer: C*

**C8. Rate Limiting**
Q: What type of rate limiting protects against brute-force placement requests?
A) Token-bucket algorithm (per-operator quota)
B) Fixed rate (all operators same limit)
C) No rate limiting (unlimited placement)
D) Depends on operator tier
*Answer: A*

**C9. Key Rotation Security**
Q: During Ed25519 key rotation (90-day enforcement), how long are OLD keys still accepted?
A) Not at all (hard cut-over)
B) 7 days (grace period)
C) 30 days (overlap period)
D) Until next rotation
*Answer: C*

**C10. Input Validation**
Q: What validation approach does Decentralized.Host use for workload specifications?
A) Blacklist known bad values (block obvious attacks)
B) Whitelist allowed values (only known good inputs)
C) No validation (trust operator)
D) Depends on workload type
*Answer: B*

---

### Section D: Workload Management (10 questions)

**D1. StatefulSets**
Q: What is the primary purpose of StatefulSets in Decentralized.Host?
A) Automatic load balancing across nodes
B) Persistent identity and storage ordering
C) GPU-only workload scheduling
D) Cross-region failover management
*Answer: B*

**D2. GPU Scheduling**
Q: How does the scheduler discover available GPUs across your nodes?
A) Manual GPU registration per operator
B) Automatic heterogeneous device discovery
C) Hard-coded GPU requirements
D) No GPU support in Phase 6
*Answer: B*

**D3. Storage Classes**
Q: Which storage class would be appropriate for a high-performance database workload?
A) Standard (HDD, slow)
B) Fast (SSD, local)
C) Cloud (remote S3-like)
D) In-memory (ephemeral)
*Answer: B*

**D4. Workload Placement Latency**
Q: What is the target placement latency for workload scheduling decisions?
A) <100ms
B) <15ms p95
C) <1 second
D) No target (on-demand)
*Answer: B*

**D5. Workload Replication Cost**
Q: A workload owner chooses 3-region replication. Who pays for the replication overhead (bandwidth, storage)?
A) Decentralized.Host absorbs cost
B) Operator absorbs cost
C) Workload owner pays higher fee for replication
D) Shared 50/50 between operator and owner
*Answer: C*

**D6. Workload Affinity**
Q: Can a workload owner request their workload run only on your operator's nodes (affinity)?
A) No (workload must be distributed globally)
B) Yes (operator-specific affinity allowed)
C) Only for TRUSTED+ tier operators
D) Only if stake is >200M uWork
*Answer: B*

**D7. Workload Failure**
Q: If a workload crashes, what happens automatically?
A) Decentralized.Host restarts it on any healthy node
B) Workload owner must manually restart
C) Operator must manually restart
D) Workload stays down (no auto-restart)
*Answer: A*

**D8. Cross-Region Workload**
Q: For a 3-region workload, where do write requests go?
A) Nearest region (lowest latency)
B) Primary region (primary replica)
C) All regions equally (load-balanced)
D) Operator's choice (configurable)
*Answer: B*

**D9. Workload Dashboard**
Q: What metrics should an operator monitor for each of their nodes? (Select primary metric)
A) Only node uptime
B) Only network throughput
C) CPU, memory, disk, network utilization
D) SLA compliance only
*Answer: C*

**D10. Workload Removal**
Q: If a workload is deleted by its owner, how long until your nodes' storage is freed?
A) Immediately
B) 24 hours (grace period for recovery)
C) 30 days (data retention)
D) Permanent (data never deleted)
*Answer: C*

---

### Section E: Advanced Topics (10 questions)

**E1. Eventual Consistency**
Q: In Phase 7 (multi-region), what is the maximum time for all regions to converge to the same global state?
A) Instant (strong consistency)
B) <1 second
C) <5 minutes (eventual consistency)
D) Unbounded (may never converge)
*Answer: C*

**E2. Network Partition**
Q: If your region is partitioned from the control plane (network down), what happens?
A) Workloads continue running (local autonomy)
B) All workloads immediately stop
C) New workloads cannot be placed (existing continue)
D) Unknown (depends on partition direction)
*Answer: C*

**E3. Sponsorship Model (TRUSTED Only)**
Q: If you are a TRUSTED operator sponsoring a BOOTSTRAP operator, what commission do you earn?
A) 5% of their revenue
B) 10% of their revenue
C) 50% of their revenue
D) No commission (just governance rights)
*Answer: A*

**E4. Tier Demotion**
Q: How many consecutive days of <95% uptime triggers BOOTSTRAP demotion?
A) 3 days
B) 7 days
C) 30 days
D) Immediate (automatic)
*Answer: C*

**E5. SLA Verification**
Q: Who independently verifies operator SLA claims each month?
A) Operator self-reports (honor system)
B) Decentralized.Host internal team
C) External third-party auditor (10% random sample)
D) Steering committee (manual review)
*Answer: C*

**E6. Merkle Anti-Entropy**
Q: If a workload replica diverges from primary, within how long is divergence detected?
A) Immediately (real-time sync)
B) <1 minute
C) <1 hour
D) Unknown (eventual detection)
*Answer: C*

**E7. Control Plane Quorum**
Q: In Phase 7, what constitutes a quorum for global decisions?
A) Any 1 region's control plane
B) 2-of-3 regions + local majority
C) All 3 regions unanimous
D) Simple majority (N/2 + 1 of all nodes)
*Answer: B*

**E8. Failover Latency**
Q: If primary workload region fails, what is target latency to failover to replica region?
A) <5 seconds
B) <10 seconds
C) <30 seconds
D) <1 minute
*Answer: C*

**E9. Data Loss Scenarios**
Q: In a 3-replica setup with quorum writes, how many replicas can fail without data loss?
A) 0 (any failure loses data)
B) 1 (2 must survive for quorum)
C) 2 (1 replica sufficient for recovery)
D) 3 (can lose all replicas)
*Answer: B*

**E10. Future Roadmap**
Q: What is the primary goal of Phase 8 (beyond Phase 7)?
A) Support for 50,000+ nodes
B) Byzantine fault tolerance (malicious operators)
C) Blockchain integration
D) Machine learning workload optimization
*Answer: B*

---

## Part 2: Practical Certification Test (60 minutes)

**Scenario:** Deploy a sample multi-region workload, respond to simulated incidents

### Practical Test Setup

**Provided Environment:**
- 3 test regions (US-WEST, EU-CENTRAL, ASIA-EAST)
- 10 test nodes per region (30 total)
- Control plane access with test credentials
- Sample workload specifications (web app, database, cache)

### Test Components

**1. Workload Deployment (20 minutes)**

**Task 1.1: Single-Region Deployment**
- [ ] Deploy provided web app workload in US-WEST region
- [ ] Specify 2 vCPU, 4 GB memory, 100 GB storage
- [ ] Verify placement on 2 different nodes (no colocation)
- [ ] Confirm <15ms placement latency
- [ ] Success Criteria: Workload running, health check passing

**Task 1.2: Cross-Region Replication**
- [ ] Configure 3-region replication (primary + 2 replicas)
- [ ] Specify: primary in US-WEST, replica in EU-CENTRAL, replica in ASIA-EAST
- [ ] Confirm replication status (all replicas receiving updates)
- [ ] Verify delta snapshot replication (not full syncs)
- [ ] Success Criteria: All replicas show "healthy" status

**Task 1.3: Cost Verification**
- [ ] Calculate expected monthly cost for 3-region deployment
- [ ] Base: $138.70 single-region
- [ ] Replica 1 (80%): $110.96
- [ ] Replica 2 (60%): $83.22
- [ ] Total: $332.88
- [ ] Success Criteria: Calculation within 5% accuracy

---

**2. Incident Response (25 minutes)**

**Scenario A: Node Failure (10 minutes)**
- [ ] Simulate primary workload node failure
- [ ] Observe: Workload remains available (health check continues)
- [ ] Verify: Workload migrated to different node
- [ ] Confirm: <30s downtime (acceptable)
- [ ] Document: What happened, why workload recovered
- Success Criteria: Workload recovery and documentation

**Scenario B: Replica Divergence (10 minutes)**
- [ ] Simulate replica lag (EU-CENTRAL replica 2 minutes behind)
- [ ] Detect divergence using monitoring tools
- [ ] Initiate Merkle anti-entropy sync
- [ ] Verify: Replica converges to primary state
- [ ] Confirm: <5 minute convergence time
- Success Criteria: Replica recovery and verification

**Scenario C: Regional Failover (5 minutes)**
- [ ] Simulate US-WEST region failure (network down)
- [ ] Observe: BGP routes change (EU-CENTRAL becomes primary)
- [ ] Verify: Workload still accessible (now from EU-CENTRAL replica)
- [ ] Confirm: Client transparency (no manual reconnect)
- [ ] Measure: Failover latency
- Success Criteria: Failover completion and measurement

---

**3. Monitoring & Dashboards (15 minutes)**

**Task 3.1: SLA Tracking**
- [ ] View operator SLA dashboard (simulated data)
- [ ] Calculate uptime: 718/730 hours = 98.4%
- [ ] Identify SLA tier: BOOTSTRAP (≥95%) ✓ Passing
- [ ] Find any issues affecting uptime
- [ ] Success Criteria: Correct calculation and tier status

**Task 3.2: Metrics Review**
- [ ] Review 18 key Prometheus metrics
- [ ] Identify any concerning trends (latency increasing? Error rate high?)
- [ ] Document: 3 metrics to monitor closely
- [ ] Success Criteria: Reasonable metric selections with justification

**Task 3.3: Audit Log Review**
- [ ] View audit logs for deployment actions
- [ ] Verify: Every action logged with timestamp
- [ ] Check: BLAKE3 hash chain integrity (no tampering)
- [ ] Find: Who deployed the workload (audit entry)
- [ ] Success Criteria: Correct audit trail interpretation

---

## Passing Criteria

### Written Exam
- **Passing Score:** 40/50 questions (80%)
- **Scoring:** 1 point per correct answer
- **Time Limit:** 90 minutes (can finish early)
- **Retakes:** Allowed after 7 days (max 3 attempts)

### Practical Test
- **Passing Criteria:** All 3 sections (deployment, incidents, monitoring) marked "success"
- **Time Limit:** 60 minutes total
- **Retakes:** Allowed after 7 days (max 2 attempts)

### Overall Certification
- **Requirement:** 80% on written exam AND all practical sections passing
- **Valid for:** 3 years
- **Recertification:** Required for tier progression (TRUSTED requires recert)

---

## Exam Conduct & Integrity

**Allowed Resources:**
- Official Decentralized.Host documentation (on screen)
- TRAINING.md and ONBOARDING.md materials (reference only)
- No external web search during exam
- No collaboration (individual assessment)

**Not Allowed:**
- Copying from other operators
- Using scripts written before exam
- Consulting external ChatGPT/AI tools
- Taking screenshots/recording exam content

**Violation Consequences:**
- First violation: Exam invalidated, retest in 30 days
- Second violation: Certification revoked, 90-day waiting period before retesting

---

## Sample Question Answers (For Study)

**A1:** B (O(1) topology lookup enables <15ms placement latency)  
**A10:** C ($332.88 for 3-region with replication)  
**B1:** C (1-2 Gbps minimum connectivity)  
**C5:** B (BLAKE3 provides tamper detection)  
**D2:** B (Automatic heterogeneous device discovery)  
**E7:** B (2-of-3 regions + local majority)

---

## Retake Policy

| Attempt | Passing Score | Wait Time | Notes |
|---------|---------------|-----------|-------|
| 1st | 80% | N/A | Initial certification |
| 2nd | 80% | 7 days | After failing 1st |
| 3rd | 80% | 7 days | Final attempt (must pass) |
| 4th+ | N/A | 30 days | After exhausting retakes |

---

## Certification Card

Upon passing, operator receives digital certification card with:
- Operator name & company
- Certification date (valid 3 years)
- Tier at certification (BOOTSTRAP, TRUSTED, or MASTER)
- Unique certification ID (fraud detection)
- QR code (verification on decentralized.host)

---

**Operator Certification Exam - Decentralized.Host**  
**Prepared:** 2026-10-04  
**Status:** Ready for Deployment  
**Target Use:** Oct 5-6 (onboarding cohort 1)

