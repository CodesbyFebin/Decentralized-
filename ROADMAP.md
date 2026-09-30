# Roadmap

## Vision

Decentralized.Host aims to give operators sovereign control over their infrastructure through signed intent, local policy enforcement, and verifiable evidence. We're building the infrastructure layer that lets you take back ownership of your workloads.

---

## Current Status: v0.1 Alpha (2026-09-30)

### ✅ Completed
- Provider discovery and observation (GitHub, Vercel, Supabase, Docker, Kubernetes, etc.)
- Universal resource graph for all cloud providers
- Migration planning and cost analysis
- Evidence-gated promotion gates
- 136/136 dh/v1 conformance test vectors
- Dev-cluster validation (3+3 topology)
- Comprehensive documentation

### 🚀 In Progress (Phases 4r, 4s, 4t)
- **Phase 4r**: Performance Optimization
  - In-memory caching with LRU/LFU eviction
  - Query optimization and slow query detection
  - Connection pooling and latency tracking
  - Status: ✅ Complete

- **Phase 4s**: GraphQL API Layer
  - Schema-driven queries, mutations, subscriptions
  - Schema introspection and discovery
  - GraphQL metrics and monitoring
  - Status: ✅ Complete

- **Phase 4t**: Advanced Monitoring
  - Distributed tracing with trace spans
  - Memory and CPU profiling
  - Health checking with configurable thresholds
  - Status: ✅ Complete

---

## v0.2 (Q1-Q2 2027): Local Control Plane Foundation

### Infrastructure
- [ ] **Raft Consensus**: 3+ node control plane with automatic failover
- [ ] **Local Policy Enforcement**: Per-host admission control before execution
- [ ] **Signed Intent**: Ed25519 cryptographic binding for workload proposals
- [ ] **Immutable Audit Trail**: Kafka-backed evidence of all state changes
- [ ] **WireGuard Mesh**: Encrypted node-to-node communication with signed identities
- [ ] **mTLS**: Mandatory encryption on all APIs and inter-node paths

### Features
- [ ] Bootstrap local infrastructure with `dh init`
- [ ] Local host management with `dh host add/list`
- [ ] Workload deployment to local infrastructure
- [ ] Local policy definition and enforcement
- [ ] Audit trail querying and export

### Testing
- [ ] Real multi-machine qualification (≥3 independent hosts)
- [ ] Network partition injection and recovery
- [ ] Storage failure scenarios
- [ ] Operator domain separation testing

### Estimated: 6-8 weeks

---

## v0.3 (Q2-Q3 2027): Multi-Node Orchestration

### Features
- [ ] Distributed scheduler with placement constraints
- [ ] Privacy-boundary scheduling (data locality)
- [ ] Failure domain awareness and isolation
- [ ] Multi-node workload rebalancing
- [ ] Disaster recovery and failover

### Chaos Testing
- [ ] 17 defined chaos scenarios
- [ ] Invariant validation under sustained traffic
- [ ] Automated chaos test suite with CI integration
- [ ] Performance profiling and benchmarking

### Estimated: 6-8 weeks

---

## v0.4 (Q3-Q4 2027): Cloud Migration & Interop

### Features
- [ ] Workload migration execution (not just planning)
- [ ] Multi-cloud orchestration
- [ ] Hybrid cloud support (local + cloud)
- [ ] Cross-cloud failover
- [ ] Provider interoperability layer

### Integrations
- [ ] AWS native integrations
- [ ] GCP native integrations
- [ ] Azure native integrations
- [ ] Additional provider adapters

### Estimated: 8-10 weeks

---

## v1.0 (Q4 2027): Production Ready

### Completion Criteria
- [ ] Full dh/v1 conformance with chaos validation
- [ ] Production qualification on real multi-machine infrastructure
- [ ] Comprehensive security audit
- [ ] Performance benchmarks and optimization
- [ ] Complete documentation and runbooks

### Stability
- [ ] API stability guarantee (semver)
- [ ] Upgrade path documentation
- [ ] Data migration guides
- [ ] Long-term support plan

### Estimated: 12+ weeks

---

## v1.x (2028+): Advanced Features

### High Priority
- [ ] Kubernetes Operator for Decentralized.Host
- [ ] ACME TLS integration (Let's Encrypt)
- [ ] Advanced monitoring integrations (Prometheus, Grafana)
- [ ] Settlement and billing system
- [ ] Multi-operator domains

### Medium Priority
- [ ] Hardware security module (HSM) integration
- [ ] Quantum-resistant cryptography
- [ ] Advanced network policies
- [ ] Resource quota management
- [ ] Custom scheduler plugins

### Community Features (with contributor support)
- [ ] Terraform provider
- [ ] Helm charts
- [ ] ArgoCD integration
- [ ] Flux CD integration
- [ ] Additional cloud providers

---

## How to Get Involved

### Contributors Welcome
- **Code**: Pick an item from this roadmap and open a PR
- **Testing**: Help validate features on your infrastructure
- **Documentation**: Improve guides and examples
- **Community**: Help review PRs, answer questions, create content

### Feedback & Requests
- Feature requests: [GitHub Discussions](https://github.com/CodesbyFebin/Decentralized-/discussions)
- Bug reports: [GitHub Issues](https://github.com/CodesbyFebin/Decentralized-/issues)
- General feedback: codesbyfebin@gmail.com

### Support the Project
- ⭐ Star the repository
- 🍴 Fork and contribute
- 💰 [Sponsor development](https://github.com/sponsors/CodesbyFebin)
- 📢 Share with your network

---

## Notes

- This roadmap is subject to change based on community feedback and resource availability
- Timelines are estimates and may shift
- Priority may change based on community needs
- All major milestones will be tracked and communicated

---

**Last Updated**: 2026-09-30
**Next Review**: 2026-12-31
