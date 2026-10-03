# Operator Onboarding Modules

**Prepared:** 2026-10-04  
**Target Audience:** BOOTSTRAP tier operators  
**Program Duration:** 7-10 weeks (2-3 weeks training, 2-3 weeks audit, 1-2 weeks operations)  
**Modules:** 7 self-paced online courses + 5 hands-on labs + 1 certification exam

---

## Program Overview

The BOOTSTRAP Operator Onboarding Program qualifies infrastructure operators through 7 comprehensive modules, 5 hands-on labs, and a 50-question certification exam. Upon completion, operators are eligible for BOOTSTRAP tier activation, managing 50+ nodes in a single region with 95% SLA commitment and earning commission-based revenue.

**Module Completion Timeline:**
- **Weeks 1-2**: Modules 1-2 (pre-registration, qualification overview)
- **Weeks 3-4**: Modules 3-4 (technical operations, security baseline)
- **Weeks 5-7**: Modules 5-6 (economics, hands-on labs)
- **Week 8**: Module 7 (certification, sign-off)

---

## Module 1: Qualification Requirements & Program Overview

**Duration:** 4-5 hours (self-paced, 2-day window minimum)  
**Delivery:** Online video course + interactive quiz + reading materials  
**Passing Score:** 80% (40/50 questions)

### Learning Objectives
- Understand BOOTSTRAP tier requirements and milestones
- Explain the 7-10 week qualification program phases
- Know operator obligations and Decentralized.Host guarantees
- Understand stake commitment and financial implications

### Content Outline

**Section 1: Decentralized.Host Overview (30 min)**
- What is Decentralized.Host?
  - Distributed scheduling platform for sovereign infrastructure
  - Work proposed as signed intent (Ed25519 identities)
  - Each host enforces its own local policy
  - No silent task migration; explicit state transitions
- How does it differ from hyperscalers?
  - Local policy enforcement (no centralized control)
  - Operator sovereignty over infrastructure
  - Revenue sharing model (earn commission on workloads)
  - Multi-tenant, multi-region, multi-operator
- Multi-region architecture
  - 3-region Raft clusters (US-WEST, EU-CENTRAL, ASIA-EAST)
  - Quorum-based consensus for topology changes
  - Cross-region replication (primary + 2 replicas)
  - Automatic failover (<30s latency)

**Section 2: Tier Progression Path (45 min)**
- BOOTSTRAP Tier (months 1-3)
  - 50+ nodes in single region
  - 95% uptime SLA target (achievable baseline)
  - 100M uWork stake (locked 90 days minimum)
  - Base commission + SLA bonus (95-97% → +2%, 97-99% → +5%, 99%+ → +10%)
  - Estimated revenue: $50-100/month (month 1, variable by utilization)
  - Time to ROI: 18-24 months
- TRUSTED Tier (months 3+)
  - 100+ nodes across 2+ regions
  - 98% uptime SLA target (higher expectation)
  - 200M uWork stake (additional 100M on progression)
  - 1.5x commission multiplier (50% revenue increase vs BOOTSTRAP)
  - Automatic progression after 90 days sustained 95%+ SLA
- MASTER Tier (year 2+)
  - 250+ nodes across 3+ regions
  - 99.5% uptime SLA target (production-grade)
  - 500M uWork stake (additional 300M)
  - 2.0x commission multiplier (100% revenue increase vs BOOTSTRAP)
  - 1-year sustained 98%+ SLA requirement
  - Federated operators (govern federation disputes)
  - Replication authorities (manage cross-region consistency)
  - Policy arbiters (dispute resolution)

**Section 3: Operator Obligations (45 min)**
- SLA Commitment
  - 95% uptime minimum (BOOTSTRAP)
  - 4-hour critical incident response
  - Monthly SLA reporting (automated)
  - Measurement methodology (available_hours / 730_hours_per_month)
- Security & Compliance
  - Annual third-party security audit
  - 40+ security audit items compliance
  - Ed25519 key rotation (90-day enforcement with overlap)
  - mTLS connection to control plane (mandatory)
  - Compliance with regional data residency regulations
- Operational Diligence
  - Weekly check-ins with onboarding POC
  - Documented incident response procedures
  - Quarterly backup/recovery testing
  - 24/7 on-call monitoring and alerting
  - Monthly SLA reports and operational metrics
- Financial Commitment
  - 100M uWork stake locked 90 days minimum
  - Operational cost (~$1,500-2,000/month for 50 nodes)
  - No guaranteed revenue (market-driven workload placement)
  - Monthly settlement with 30-day net payment terms

**Section 4: Decentralized.Host Guarantees (45 min)**
- Technical Support
  - Dedicated onboarding POC (assigned to each operator)
  - 24/7 support channel (Slack #operator-support)
  - <4-hour SLA for critical issues
  - Weekly operational reviews and optimization recommendations
  - Quarterly architecture briefings
- Operational Stability
  - Control plane uptime 99.9% (committed SLA)
  - 72-hour workload recovery SLA
  - Zero unplanned node terminations (unless hardware failure)
  - 30-day advance notice for breaking changes
- Revenue Protection
  - Transparent billing (minute-level precision)
  - Fair pricing (no hidden fees)
  - Monthly revenue settlement (net 30 days)
  - Third-party dispute resolution available
- Community & Growth
  - Featured on decentralized.host operator map
  - Joint marketing opportunities (webinars, case studies)
  - Peer learning community forum
  - Priority access to new features

**Section 5: Pre-Qualification Self-Assessment (20 min)**
Operators assess their readiness:
- Do we have 5+ years infrastructure ops experience?
- Can we commit to 95% SLA with confidence?
- Do we have 100M uWork capital available?
- Can we provision and operate 50+ nodes reliably?
- Are we willing to undergo security audit?
- Do we understand the 7-10 week qualification timeline?
- Can we commit to weekly check-ins + incident response?
- (6+ yes answers = good fit for BOOTSTRAP)

### Module 1 Quiz Format
- 50 multiple-choice questions (90 minutes)
- Topics: Tier progression, operator obligations, SLA commitment, financial models
- Passing score: 80% (40/50)
- Retakes: Unlimited, 1-week wait between attempts

### Deliverables
- ✅ Module 1 quiz score (80%+)
- ✅ Signed commitment to operator obligations
- ✅ Financial confirmation of 100M uWork stake

---

## Module 2: Tier Progression Path & Sponsorship Model

**Duration:** 5-6 hours (self-paced, 2-day window minimum)  
**Delivery:** Online video course + diagrams + financial spreadsheets + quiz  
**Passing Score:** 80% (40/50 questions)

### Learning Objectives
- Understand tier progression metrics and timelines
- Know automatic vs. manual progression triggers
- Understand sponsorship model and revenue sharing
- Know demotion triggers and remediation procedures

### Content Outline

**Section 1: BOOTSTRAP Tier Deep Dive (1.5 hours)**
- Tier Mechanics
  - 50+ dedicated compute nodes
  - Single-region operation (US-WEST, EU-CENTRAL, or ASIA-EAST)
  - 1-2 Gbps network minimum
  - 100M uWork stake locked 90 days minimum
  - 95% uptime SLA commitment
- Commission Model
  - Base commission: (workload_cpu_hours + workload_memory_hours + workload_storage_hours) × base_price
  - Example: 1000 vCPU-hours/month @ $0.05/hour = $50/month base
  - SLA bonus: 95-97% → +2%, 97-99% → +5%, 99%+ → +10%
  - No tier multiplier (1.0x base rate)
  - Monthly settlement in arrears
- Revenue Calculation (Example Month 1)
  - 50 nodes × 100 vCPUs = 5,000 vCPU available
  - 30% utilization average = 1,500 vCPU-hours/day
  - At $0.05/vCPU + $0.02/GB-hour (100GB storage):
    - Daily revenue: $75-100
    - Monthly revenue: $2,250-3,000
    - Infrastructure cost: $1,500-2,000/month
    - Net monthly: $250-1,500
  - Plus 95% SLA bonus (+5%): $2,362-3,150
  - ROI timeline: 18-24 months (conservative estimate)
- Key Metrics Tracked
  - Node availability (uptime %)
  - Error rate (failed placements / total)
  - Placement latency (p95 <15ms)
  - Workload density (utilization %)
  - Incident response time

**Section 2: TRUSTED Tier & Progression (1.5 hours)**
- Tier Requirements
  - 100+ nodes across 2+ regions
  - 98% uptime SLA target
  - 200M uWork stake (additional 100M deposit)
  - Automatic progression after 90 days sustained 95%+ SLA + capacity expansion
- Sponsorship Model
  - BOOTSTRAP operators cannot directly progress to TRUSTED without sponsor
  - TRUSTED operator sponsors BOOTSTRAP operator
  - Sponsorship agreement: TRUSTED guarantees BOOTSTRAP's SLA
  - Revenue sharing: BOOTSTRAP pays TRUSTED 5% commission (on top of Decentralized.Host commission)
  - Example: BOOTSTRAP earning $3,000/month
    - Decentralized.Host commission: $3,000
    - TRUSTED sponsor cut: 5% = $150
    - BOOTSTRAP net: $2,850
- Progression Timeline
  - Day 0: BOOTSTRAP operator enrollment
  - Day 90: SLA check (must be >95% across all 90 days)
  - Day 91: Capacity expansion to 100+ nodes
  - Day 92: Additional 100M uWork stake deposit
  - Day 93: Steering committee approval (automatic if criteria met)
  - Day 94: Tier badge updated, new commission rates active
- Commission Multiplier
  - TRUSTED: 1.5x base rate (50% revenue increase)
  - Example: TRUSTED earning $4,500/month
    - Base commission: $4,500
    - Tier multiplier: 1.5x
    - Total commission: $6,750
    - Sponsor cut: 5% = $337.50
    - TRUSTED net: $6,412.50

**Section 3: MASTER Tier (1 hour)**
- Tier Requirements
  - 250+ nodes across 3+ regions
  - 99.5% uptime SLA target
  - 500M uWork stake (additional 300M)
  - 1-year sustained 98%+ SLA
- Federation Roles
  - Federated operators (serve other operators' workloads)
  - Replication authorities (manage cross-region consistency)
  - Policy arbiters (govern operator disputes)
- Commission Multiplier
  - MASTER: 2.0x base rate (100% revenue increase)
  - Dispute arbitration fees (resolution of operator disputes)
- Progression Path
  - TRUSTED with 1-year sustained 98%+ SLA → MASTER eligible
  - Steering committee approval + operator nomination (3+ other operators)

**Section 4: Sponsorship Model Mechanics (1 hour)**
- Who Sponsors Whom?
  - TRUSTED sponsors BOOTSTRAP (1-2 per TRUSTED typical)
  - MASTER can sponsor BOOTSTRAP or TRUSTED
  - Max 3 sponsorships per operator (capacity limit)
  - Sponsorship can be revoked (with 90-day notice)
- Revenue Sharing
  - Base model: 5% commission share to sponsor
  - Sponsor is liable for sponsored operator's SLA gaps
  - If sponsored operator misses SLA, sponsor also loses that month's bonus
- Dispute Resolution in Sponsorship
  - Sponsor and sponsored operator disagree on SLA calculation
  - Escalate to MASTER operator arbiters
  - MASTER votes on dispute (simple majority)
  - Binding decision, payment adjusted accordingly
- Incentive Programs
  - Early adopter bonus: First 5 BOOTSTRAP cohort → +10% commission month 1
  - Referral commission: Refer operator who enrolls → +2% on referred operator's commission
  - Tier loyalty bonus: Maintain tier for 1 year → +5% commission next year
  - Sponsorship incentive: Sponsor BOOTSTRAP → +3% commission on sponsored operator's earnings

**Section 5: SLA Demotion & Remediation (45 min)**
- Demotion Triggers
  - 30 consecutive days below SLA threshold
  - Critical security audit findings
  - Regulatory compliance breach
  - Repeated incident response SLA misses
- Demotion Timeline
  - Day 1-15: Warning notice, operator notified
  - Day 15-25: Remediation period (operator can improve SLA)
  - Day 25: Second warning (last 5 days)
  - Day 30: Automatic demotion if still below threshold
  - Audit log entry for governance record
- Remediation Procedures
  - Operator can request SLA re-audit
  - Provide evidence of infrastructure improvements
  - Third-party audit verifies improvements
  - If improvements verified, demotion reversed
  - Re-audit cost: $5,000 (operator pays if fails re-audit)

### Module 2 Quiz Format
- 50 multiple-choice + 5 short-answer questions (120 minutes)
- Topics: Tier progression, sponsorship model, commission calculation, demotion rules
- Passing score: 80% (44/55)
- Retakes: Unlimited, 1-week wait

### Deliverables
- ✅ Module 2 quiz score (80%+)
- ✅ Commission calculation worksheet (verified by trainer)
- ✅ Sponsorship agreement template (if pursuing TRUSTED progression)

---

## Module 3: Technical Operations & Control Plane API

**Duration:** 6-7 hours (self-paced + 1 live lab session)  
**Delivery:** Online video course + API documentation + code examples + hands-on lab  
**Passing Score:** 80% (40/50 questions)

### Learning Objectives
- Understand control plane API architecture and endpoints
- Know workload registration and scheduling flow
- Understand local policy enforcement and admission control
- Know node identity management and signed operations

### Content Outline

**Section 1: Control Plane Architecture (1.5 hours)**
- Architecture Overview
  - Multi-region Raft-based control plane
  - 3-region consensus (US-WEST, EU-CENTRAL, ASIA-EAST)
  - Sub-100ms intra-region latency
  - Quorum-based decisions (2/3 regions required)
- Node Registration
  - Each node gets Ed25519 identity (public + private key)
  - Node registers with control plane (mTLS mutual auth)
  - Node identity bound to operator account
  - Node capabilities declared (CPU, memory, storage, network)
- Control Plane Endpoints
  - Topology API (list nodes, get node status, register node)
  - Workload API (deploy workload, list workloads, delete workload)
  - Policy API (query node policy, submit policy update)
  - Metrics API (get node metrics, operator metrics, SLA metrics)

**Section 2: Workload Registration & Scheduling (2 hours)**
- Workload Proposal
  - Operator submits workload as signed intent
  - Format: TOML/JSON with workload requirements
  - Signature: Ed25519 signed by operator key
  - Control plane validates signature
- Scheduling Decision
  - Control plane evaluates placement constraints
  - Geographic diversity (no colocation in same region)
  - Node capacity (available CPU/memory/storage)
  - Cost optimization (prefer cheaper regions)
  - Placement decision sent to selected node
- Node Admission Control
  - Node receives placement decision (signed by control plane)
  - Node runs local policy evaluation (Rego policy file)
  - Policy checks: operator authorization, workload signature, resource availability
  - Node admits (execute) or rejects placement
  - All decisions logged to audit trail (BLAKE3 chain)
- Workload Execution
  - Node executes workload in isolated container
  - Node monitors workload health and metrics
  - Node reports back to control plane (status updates, metrics)
  - Control plane tracks workload state (DESIRED → ADMITTED → EXECUTING → OBSERVED → VERIFIED)

**Section 3: Local Policy Enforcement (1.5 hours)**
- Policy Language (Rego/OPA)
  - Open Policy Agent (OPA) policy language
  - Example policies:
    ```
    package policy
    default allow = false
    allow {
      input.operator.tier == "TRUSTED"
      input.workload.cpu <= 8
      input.node.available_memory >= input.workload.memory
    }
    ```
  - Per-node policy customization
  - Operator controls own policy (local sovereignty)
- Common Policy Rules
  - Resource quotas (max CPU/memory per operator)
  - Security constraints (only signed workloads)
  - Locality constraints (enforce data residency)
  - Tier-based policies (BOOTSTRAP vs TRUSTED vs MASTER)
- Policy Audit Trail
  - All policy decisions logged
  - BLAKE3 hash chain (tamper-evident)
  - 90+ day retention minimum
  - Searchable by workload, operator, decision

**Section 4: Node Identity & Cryptography (1.5 hours)**
- Ed25519 Key Management
  - Private key: Never leave node, protected by file permissions
  - Public key: Shared with control plane during registration
  - Key rotation: 90-day enforcement with overlap period
  - Old key still valid during overlap (seamless rotation)
- Signed Operations
  - All workload proposals signed by operator
  - All node operations signed by node identity
  - All policy updates signed by operator
  - Signature format: Ed25519, compact binary encoding
- mTLS Mutual Authentication
  - Node → Control Plane: mTLS mutual auth
  - Control Plane → Node: mTLS mutual auth
  - TLS 1.3 minimum
  - Certificate rotation: 30-day enforcement
  - Hostname verification: enabled

**Section 5: Hands-On Lab Session (1 hour live + 2 hours self-paced)**
- Lab 1: Node Registration (hands-on)
  - Generate Ed25519 key pair
  - Register node with control plane (mTLS)
  - Verify node identity
  - Query node status from API
- Lab 2: Workload Deployment (hands-on)
  - Create workload definition (TOML)
  - Sign workload with operator key
  - Submit to control plane
  - Monitor placement and execution
- Lab 3: Local Policy (hands-on)
  - Create Rego policy file
  - Deploy policy to node
  - Test policy with various workloads
  - Verify audit logs

### Module 3 Quiz Format
- 50 multiple-choice questions (90 minutes)
- Topics: Control plane API, workload scheduling, policy enforcement, cryptography
- Passing score: 80% (40/50)
- Retakes: Unlimited, 1-week wait

### Deliverables
- ✅ Module 3 quiz score (80%+)
- ✅ Lab 1-3 completion certificates
- ✅ Sample policy file (reviewed by trainer)

---

## Module 4: Security & Compliance Baseline

**Duration:** 5-6 hours (self-paced)  
**Delivery:** Online video course + compliance checklist + security hardening guide  
**Passing Score:** 80% (40/50 questions)

### Learning Objectives
- Understand TLS/mTLS mutual authentication requirements
- Know key rotation enforcement and overlap periods
- Understand audit logging and BLAKE3 hash chain
- Know compliance requirements and audit process

### Content Outline

**Section 1: TLS & mTLS Configuration (1.5 hours)**
- TLS 1.3 Minimum Enforcement
  - No TLS 1.2 or older allowed
  - Ciphers: TLS_AES_256_GCM_SHA384, TLS_CHACHA20_POLY1305_SHA256
  - Session resumption: disabled (no session tickets)
  - Forward secrecy: enabled (ephemeral key exchange)
- mTLS Mutual Authentication
  - Both client and server verify identity
  - Client certificate: node identity
  - Server certificate: control plane identity
  - Certificate pinning: enabled (no TOFU)
- Certificate Management
  - Issue certificates via ACME (Let's Encrypt or Pebble)
  - Certificate lifetime: 90 days
  - Renewal: automatic at day 30
  - Revocation: CRL checking enabled

**Section 2: Key Rotation & Crypto Practices (1.5 hours)**
- Ed25519 Key Rotation
  - Rotation frequency: 90 days (enforced)
  - Overlap period: 30 days (old and new key valid simultaneously)
  - Zero-downtime rotation (stateless services)
  - Rotation timeline:
    - Day 0: New key generated, shared with control plane
    - Day 1-30: Both old and new keys accepted
    - Day 31: Old key deactivated, removed from control plane
- Cryptographic Practices
  - No MD5, SHA1, or weak hashes
  - BLAKE3 for content-addressed storage
  - SHA256/SHA512 for general hashing
  - Random number generation: /dev/urandom only
  - No hardcoded secrets or keys in code

**Section 3: Audit Logging & BLAKE3 Chain (1.5 hours)**
- Audit Trail Requirements
  - Every policy decision logged
  - Every workload placement logged
  - Every key rotation logged
  - Every configuration change logged
  - Timestamps: UTC, high precision (nanoseconds)
- BLAKE3 Hash Chain
  - Entry format: {timestamp, event, data, previous_hash}
  - Hash: BLAKE3(entry_json) → 256-bit digest
  - Tamper detection: Any hash mismatch breaks chain
  - Anti-entropy: Merkle tree for efficient verification
- Retention & Archival
  - Minimum 90-day local retention
  - Monthly archive to immutable storage (S3, GCS)
  - Audit retention: 7 years (regulatory requirement)
  - Searchable index (operator, date, event type)

**Section 4: Compliance & Audit Process (1.5 hours)**
- 40+ Compliance Items
  - Operating system: Latest stable LTS (Ubuntu 22.04+, CentOS 8+)
  - Kernel patches: Applied monthly, security patches within 48 hours
  - User access: RBAC-based, principle of least privilege
  - Network: Firewall rules, no default allow-all
  - SSH: Key-based auth only, no password auth
  - Logging: Centralized syslog, retention 30+ days
  - Backup: Daily snapshots, recovery tested quarterly
  - Monitoring: Prometheus metrics, alerting on anomalies
  - Updates: Patch management process documented
  - Passwords: Min 12 chars, complexity requirements, 90-day rotation
- Third-Party Audit Process
  - Auditor assigned (independent firm, 5+ years experience)
  - Pre-audit questionnaire (40+ items, 2-week response window)
  - On-site audit (1-2 weeks, can be remote)
  - Testing: Code review, configuration audit, penetration testing
  - Findings: Categorized as critical/high/medium/low
  - Remediation: Critical items within 48 hours, high within 1 week
  - Sign-off: Auditor certifies operator meets baseline
- Post-Audit Remediation
  - Audit findings documented
  - Remediation plan created (critical first)
  - Proof of remediation provided to auditor
  - Auditor verifies fixes
  - Final audit report issued

### Module 4 Quiz Format
- 50 multiple-choice questions (90 minutes)
- Topics: TLS/mTLS, key rotation, audit logging, compliance
- Passing score: 80% (40/50)
- Retakes: Unlimited, 1-week wait

### Deliverables
- ✅ Module 4 quiz score (80%+)
- ✅ Pre-audit questionnaire completed (40+ items)
- ✅ Network security diagram (reviewed by trainer)

---

## Module 5: Economics & Settlement

**Duration:** 4-5 hours (self-paced + financial models)  
**Delivery:** Online video course + spreadsheets + settlement examples  
**Passing Score:** 80% (40/50 questions)

### Learning Objectives
- Understand commission calculation and SLA bonuses
- Know stake mechanics and withdrawal process
- Understand monthly settlement and dispute resolution
- Know financial projections and ROI analysis

### Content Outline

**Section 1: Commission Model & Calculation (1.5 hours)**
- Base Commission
  - Formula: (workload_cpu_hours + workload_memory_hours + workload_storage_hours) × base_price
  - CPU: $0.05/hour per vCPU
  - Memory: $0.02/hour per GB
  - Storage: $0.01/hour per GB
  - Example: 4vCPU, 8GB RAM, 100GB storage for 1 hour
    - CPU: 4 × $0.05 = $0.20
    - Memory: 8 × $0.02 = $0.16
    - Storage: 100 × $0.01 = $1.00
    - Total: $1.36 per hour
- SLA Bonus
  - 95-97% uptime: +2% commission
  - 97-99% uptime: +5% commission
  - 99%+ uptime: +10% commission
  - Example: Workload at 98% uptime
    - Base commission: $1.36
    - SLA bonus: 5%
    - Total: $1.36 × 1.05 = $1.4288
- Tier Multiplier
  - BOOTSTRAP: 1.0x (no multiplier)
  - TRUSTED: 1.5x (50% increase)
  - MASTER: 2.0x (100% increase)
  - Example: TRUSTED at 98% uptime
    - Base: $1.36 × 1.05 = $1.4288
    - Tier multiplier: $1.4288 × 1.5 = $2.1432

**Section 2: Stake Mechanics (1.5 hours)**
- Initial Stake Deposit
  - BOOTSTRAP: 100M uWork (locked 90 days minimum)
  - TRUSTED: Additional 100M uWork (200M total)
  - MASTER: Additional 300M uWork (500M total)
- Stake Lock-Up & Withdrawal
  - Minimum lock-up: 90 days (BOOTSTRAP), 6 months (TRUSTED), 1 year (MASTER)
  - Early withdrawal penalty: 10% (applied to withdrawing amount)
  - Withdrawal timeline: Request → 30-day notice → release
  - No interest paid on stake (pure collateral)
- Slash Conditions
  - Severe SLA breach (>50% downtime): 5% slash
  - Security audit failure: 10% slash
  - Regulatory breach: Full stake forfeit (rare)
  - No slash for operational incidents if SLA met

**Section 3: Monthly Settlement & Billing (1 hour)**
- Billing Cycle
  - Billing month: Calendar month (1st - last day)
  - Calculation: Minute-level granularity (workload start/stop times)
  - Billing: Released on 1st of next month
  - Payment: Net 30 days (payment due by 30th of next month)
- Settlement Process
  - Control plane calculates usage (workload × duration)
  - Decentralized.Host verifies usage (audit logs)
  - Commission calculated (base + SLA bonus + tier multiplier)
  - Settlement ledger generated
  - Operator notified (dashboard + email)
- Dispute Resolution
  - Operator disputes commission calculation
  - Provide evidence (workload logs, metrics)
  - Third-party audit (if disagreement persists)
  - Binding decision within 30 days
  - Payment adjustment in following month

**Section 4: Financial Projections & ROI (1.5 hours)**
- BOOTSTRAP Month 1 Projection
  - 50 nodes, 100 vCPU per node = 5,000 vCPU available
  - 30% utilization = 1,500 vCPU-hours/day
  - At $0.05/vCPU: $75/day CPU commission
  - Storage (100GB per workload): $0.01/hour/GB
  - Daily revenue: ~$100-150
  - Monthly revenue: $3,000-4,500
  - Infrastructure cost: $1,500-2,000/month
  - Net monthly: $1,000-3,000
  - Plus SLA bonus (95%): +5% = $50-150/month net
- TRUSTED Projection (Month 4)
  - 100 nodes, 200 vCPU available
  - 40% utilization (improved): 1,920 vCPU-hours/day
  - Tier multiplier: 1.5x base rate
  - Monthly revenue: $10,000-12,000
  - Infrastructure cost: $3,500-4,000/month
  - Net monthly: $6,000-8,500
  - Plus sponsorship revenue (5 BOOTSTRAP sponsored):
    - $100 × 5 operators × 5% = ~$25-50/month
  - Total net: $6,025-8,550
- ROI Analysis
  - Initial investment: 100M uWork ($50,000-100,000)
  - Monthly net (BOOTSTRAP): $1,500 average
  - Payback period: 33-67 months (~3-5 years)
  - Year 1 revenue: $12,000-18,000 net (higher than cost)
  - Year 2 progression to TRUSTED: 1.5x multiplier boost
  - Long-term (5+ years): Cumulative $100,000+ earnings potential

### Module 5 Quiz Format
- 40 multiple-choice + 5 calculation questions (90 minutes)
- Topics: Commission models, SLA bonuses, stake mechanics, settlement
- Passing score: 80% (36/45)
- Retakes: Unlimited, 1-week wait

### Deliverables
- ✅ Module 5 quiz score (80%+)
- ✅ Commission calculation worksheet (verified)
- ✅ Financial projection spreadsheet (12-month forecast)

---

## Module 6: Hands-On Labs (All 5 Labs)

**Duration:** 10-12 hours total (self-paced, 4-week delivery window)  
**Delivery:** Real multi-region test environment + code templates + detailed instructions  
**Completion Criteria:** All 5 labs operational, metrics verified

### Learning Objectives
- Deploy workloads across single and multiple regions
- Verify replication and failover mechanics
- Inject failures and verify recovery
- Monitor metrics and SLA tracking

### Lab Outline

**Lab 1: Single-Node Bootstrap (2 hours)**
- Provision single node in US-WEST region
- Generate Ed25519 identity
- Register node with control plane
- Deploy sample workload
- Verify workload execution and metrics
- Deliverable: Node registered, 1 workload running

**Lab 2: Multi-Node Federation (2.5 hours)**
- Provision 3 nodes in single region
- Form Raft cluster
- Deploy workload to cluster
- Verify placement and replication
- Test leader election (kill leader, verify recovery)
- Deliverable: Cluster stable, replication verified

**Lab 3: Failure Injection (2.5 hours)**
- Deploy 3-region workload (primary + 2 replicas)
- Inject node failure (simulate crash)
- Measure failover latency (<30s target)
- Verify zero data loss (quorum writes)
- Observe replica promotion
- Deliverable: Failover <30s, zero data loss

**Lab 4: Cross-Region Failover (2.5 hours)**
- Simulate regional partition (network latency increase to 500ms+)
- Observe quorum behavior (can minority survive?)
- Test read routing (reads from nearest replica)
- Test write routing (writes to primary with quorum ACK)
- Verify eventual consistency (anti-entropy repair)
- Deliverable: Consistency maintained under partition

**Lab 5: Production Validation (2.5 hours)**
- Deploy 50-node single-region cluster
- Sustained load (1000 workloads, 100 writes/sec)
- Measure end-to-end latency (p95, p99)
- Verify 95% uptime SLA (run for 24 hours)
- Review audit logs and metrics dashboard
- Deliverable: 24-hour production run verified, metrics collected

### Lab Completion Certificate
Upon completion of all 5 labs with passing metrics:
- ✅ Lab completion certificate issued
- ✅ Eligible to proceed to certification exam
- ✅ Operator readiness verified

---

## Module 7: Certification & Sign-Off

**Duration:** 3-4 hours (proctored exam)  
**Delivery:** Remote proctored exam + practical tasks  
**Passing Score:** 40/50 (80%) written + all practical sections pass

### Exam Format
See CERTIFICATION-EXAM.md for complete details:
- Part 1: 50-question written exam (90 minutes)
- Part 2: 3-part practical test (60 minutes)

### Certification Validity
- Valid for 3 years
- Renewal: Ongoing operational compliance
- Retakes: Allowed, 1-week wait between attempts

### Sign-Off Process
- Trainer reviews exam results
- Operations team confirms production readiness
- Security team confirms audit completion
- Finance team confirms stake deposit
- Steering committee approves for go-live
- Operator notified: BOOTSTRAP tier activated

### Deliverables
- ✅ Certification exam score (40/50+ passing)
- ✅ Operations sign-off
- ✅ Security audit completion
- ✅ Go-live approval

---

## Support & Continuous Education

### Support Channels
- **Slack #operator-support**: 24/7 monitored by on-call engineer
- **Email operator-support@decentralized.host**: General questions
- **Weekly office hours**: Tuesday 2pm UTC, Thursday 5pm UTC
- **Monthly webinars**: Deep dives into new features, operator best practices

### Ongoing Learning
- Monthly operator webinars (operator success stories, technical deep dives)
- Quarterly architecture briefings (what's new, what's coming)
- Peer learning forum (operator-to-operator knowledge sharing)
- Certification renewal (3-year cycle)

### Common Questions FAQ
- How long until TRUSTED tier? (90 days sustained >95% SLA + capacity expansion)
- What happens if SLA drops below 95%? (Warning at day 15, demotion at day 30, appeal available)
- Can I change regions? (BOOTSTRAP single-region only; TRUSTED can expand)
- What if I need to exit? (30-day notice, workloads migrate off, stake released)

---

## Program Metrics & Success Criteria

### Completion Metrics
- Module completion rate: 100% (all 7 modules)
- Quiz passing rate: 80%+ (each module)
- Lab completion rate: 100% (all 5 labs)
- Exam passing rate: 40/50 (80%)
- Time to completion: 7-10 weeks target

### Operator Readiness Metrics
- Pre-audit questionnaire: 40/40 items completed
- Security audit: No critical findings
- Operational readiness: 95% uptime achievable (demonstrated in labs)
- Financial readiness: Stake deposited, invoice accepted
- Compliance: All signatures collected

### Post-Launch Metrics (First 90 Days)
- SLA achievement: Track weekly uptime %
- Workload placement: Track daily/weekly placement count
- Revenue generation: Track monthly commission (baseline for future tiers)
- Incident response: Track MTTR (mean time to recovery)
- Operator satisfaction: Monthly NPS survey

---

**Operator Onboarding Modules - Decentralized.Host**  
**Prepared:** 2026-10-04  
**Status:** Approved for distribution to BOOTSTRAP cohort 1  
**Launch Date:** Oct 5, 2026
