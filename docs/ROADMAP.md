# Decentralized.Host Roadmap

**Vision**: Connect your cloud. Own it over time.

Start by observing all your existing infrastructure (GitHub, Vercel, Supabase, Docker, Kubernetes, etc.) in one unified graph. Then progressively migrate workloads to owned infrastructure as you choose, with cryptographic proof every step of the way.

---

## v0.1 Alpha: Provider Connection & Observation (Current)

**Status**: In development  
**Focus**: Viral loop (connect → graph → doctor → migrate → prove)

### Features

✅ **dh connect**
- Discover and authenticate to GitHub, GitLab, Vercel, Railway, Supabase, Cloudflare, Docker, Kubernetes, AWS, GCP, Azure, Ollama
- Store credentials securely (encrypted at-rest)
- Import resource metadata into unified graph

✅ **dh graph**
- Visualize all connected resources (repos, deployments, databases, containers, ML models)
- Show dependencies (repo → deployment → database)
- Highlight risk zones (vendor lock-in, privacy exposure, cost outliers)

✅ **dh doctor**
- Score each resource on:
  - **Sovereignty**: Can you migrate this? (locked vs portable)
  - **Privacy**: Where does data live? (local vs cloud)
  - **Portability**: How much work to move it? (1 day vs 3 months)
  - **Cost**: What are you paying? (per-resource breakdown)

⏳ **dh migrate** (planning only)
- Show before/after topology for migration
- Estimate downtime, data transfer, cost
- NOT YET: Automatic migration execution

⏳ **dh prove** (planning only)
- Cryptographic proof of state before/after
- NOT YET: Signed intent or policy enforcement

### Not In v0.1

- Local infrastructure bootstrap (no Raft, no WireGuard, no policy enforcement)
- Workload execution on owned infrastructure
- Privacy-boundary scheduling
- Incident replay or chaos testing
- Settlement or billing integration

### Testing

- Unit tests: 136/136 dh/v1 conformance vectors PASS
- Dev-cluster validation: Functionality tested on 3+3 loopback topology
- Real multi-machine validation: Pending (requires independent Linux hosts)

---

## v0.2: Own — Local Sovereign Infrastructure (Q1 2027)

**Focus**: First owned machine, progressive workload migration

### Features

✅ **dh up**
- Bootstrap 1-node control plane on Linux with Ed25519 identity
- WireGuard mesh preparation (for multi-node in v0.3)
- TLS/mTLS on all APIs

✅ **dh init**
- Define local policy (what workloads can run, where, with what permissions)
- Generate root-CA and operator identities
- Bootstrap ACME (Pebble for test, production ACME for prod)

✅ **dh migrate <resource>** (execution)
- Execute migrations with rollback capability
- Real workload execution on owned infrastructure
- Continuous state observation and validation

✅ **dh prove** (real)
- Cryptographic binding of all state transitions
- Ed25519 signatures on all operations
- Immutable audit trail (Raft-backed)

### Foundations

- Explicit state machine: DESIRED → ADMITTED → EXECUTING → OBSERVED → VERIFIED
- Per-host local policy enforcement
- BLAKE3 content-addressed storage
- Raft consensus + mTLS control plane
- Signed intent validation (no silent task migration)

### Testing

- Real single-machine qualification (P1_SINGLE)
- Chaos testing on 1-node topology
- Privacy-boundary scheduling validation

### NOT In v0.2

- Multi-node clustering (requires v0.3)
- Settlement or usage billing
- Agent capability passports
- Post-quantum cryptography

---

## v0.3: Mesh — Multi-Node Infrastructure (Q2 2027)

**Focus**: Cluster orchestration, failure isolation, GPU scheduling

### Features

✅ **dh up --nodes 3+**
- Bootstrap 3+ node Raft control plane
- Automated failover and leader election
- WireGuard mesh with signed key bindings
- Network partition detection and halt logic

✅ **dh schedule**
- Affinity rules (data locality, GPU placement)
- Privacy-boundary scheduling (data classification → physical boundaries)
- Failure domain awareness (separate nodes, separate subnets, separate operators)

✅ **dh logs / dh metrics / dh audit**
- Full observability stack
- Prometheus metrics
- Distributed tracing
- Immutable audit trail with cryptographic proof

### Foundations

- Multi-node Raft consensus with 3+ members
- Userspace WireGuard with signed identities
- Network partition detection
- Storage anti-entropy (Merkle proofs)
- Automatic workload reconciliation on node failure

### Testing

- Real multi-machine qualification (P1_MULTIPHYSICAL)
- Chaos testing: node failures, network partitions, storage corruption
- Independent operator domain validation

### NOT In v0.3

- Settlement or blockchain integration
- Kubernetes-style orchestration (k8s compatibility comes v1.0+)
- Agent autonomous scheduling

---

## v1.0: Share — Marketplace & Settlement (Q3 2027)

**Focus**: Monetization, federation, sovereign developer community

### Features

✅ **dh marketplace**
- Publish surplus capacity (CPU, GPU, storage, bandwidth)
- Browse available infrastructure
- Subscribe to managed services

✅ **dh settle**
- Usage billing (compute hours, storage GB, bandwidth Mbps)
- Settlement backends: Stripe credits, fiat, blockchain (optional)
- Transparent pricing (no hidden vendor lock-in)

✅ **dh agent**
- Agent capability passports (signed, expiring credentials)
- Sandboxed AI agent execution on owned infrastructure
- Policy-checked tool access

✅ **dh federation**
- Multi-cluster coordination
- Cross-boundary resource scheduling
- Trust domains and delegation

### Production Hardening

- P1_PRODUCTION qualification on real multi-machine, multi-operator infrastructure
- Security audit with independent firm
- SLA guarantees and incident response procedures
- High-availability control plane (5+ nodes, quorum)

### Completeness

- Full sovereign developer cloud feature set
- 100+ capabilities across infrastructure, AI, privacy, compliance
- Optional blockchain settlement (not required)
- Community marketplace (DAO governance optional)

---

## Quarterly Milestones

| Quarter | Milestone | Qualification Target |
|---------|-----------|---------------------|
| Q4 2026 | v0.1 Alpha | Dev-cluster (3+3 loopback) |
| Q1 2027 | v0.2 Own | Real single-machine (P1_SINGLE) |
| Q2 2027 | v0.3 Mesh | Real multi-machine (P1_MULTIPHYSICAL) |
| Q3 2027 | v1.0 Share | Production (P1_PRODUCTION) |

---

## Qualification Levels

Each release has a target qualification level:

**P1_CORE** (Foundation)
- Signed intent validation
- Local policy enforcement
- State machine correctness
- Immutable audit trail
- **Substrate**: Any Linux runtime (container, VM, physical)

**P1_SINGLE**
- Single-machine reliability and observability
- Chaos testing on 1-node topology
- Ed25519 identity validation
- ACME TLS integration
- **Substrate**: Single Linux host

**P1_MULTIPHYSICAL**
- Multi-machine Raft consensus
- Network partition detection
- Storage anti-entropy with Merkle proofs
- Failure domain isolation (OS, filesystem, physical host)
- **Substrate**: ≥2 independent Linux hosts with distinct isolation boundaries

**P1_PRODUCTION**
- High-availability control plane (5+ nodes)
- Independent security audit
- SLA guarantees
- Incident response procedures
- **Substrate**: Real production infrastructure (multiple data centers or regions)

---

## Feature Freeze Philosophy

**v0.1 carries no foundation work beyond provider discovery and graphing.** No Raft, no WireGuard, no policy enforcement, no migrations.

This forces early focus on the viral loop: If `dh connect`, `dh graph`, and `dh doctor` don't create immediate value for observability, the entire strategy fails. We build the foundation only after proving the market exists.

**v0.2 adds the foundations,** but only single-machine (no clustering complexity).

**v0.3 scales to clusters** once v0.2 proves the single-machine model works in production.

**v1.0 adds settlement** only after multi-machine infrastructure is proven, battle-tested, and boring.

This prevents feature sprawl, maintains implementation focus, and ensures each release ships what matters most first.
