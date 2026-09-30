# Decentralized.Host (dh) — Sovereign Infrastructure with Signed Intent

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT%2FApache%202.0-blue.svg)](LICENSE)
[![GitHub Stars](https://img.shields.io/github/stars/CodesbyFebin/Decentralized-.svg?style=social)](https://github.com/CodesbyFebin/Decentralized-)
[![GitHub Forks](https://img.shields.io/github/forks/CodesbyFebin/Decentralized-.svg?style=social)](https://github.com/CodesbyFebin/Decentralized-)
[![Contribute](https://img.shields.io/badge/contributions-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![Sponsor](https://img.shields.io/badge/sponsor-%E2%9D%A4-ff69b4.svg)](https://github.com/sponsors/CodesbyFebin)

**Status**: v0.1 Alpha — Provider Connection & Unified Graph  
**Qualification**: Dev-cluster validation (3+3 loopback topology). Real PV1 multi-machine qualification pending.  
**License**: Dual-licensed under MIT and Apache 2.0

> Connect your cloud. Own it over time. Start by importing GitHub, Vercel, Supabase, Docker, Kubernetes—observe them in one graph, then migrate workloads to owned infrastructure as you choose.

**Current Scope**: Provider discovery, observation, and local policy enforcement. Production multi-machine qualification and settlement features coming in v0.2+.

**Community**: We welcome contributions, sponsorships, and collaboration! See [CONTRIBUTING.md](CONTRIBUTING.md) to get started.

---

## What This Is

Decentralized.Host is a provider-agnostic control plane that imports your existing cloud (GitHub, Vercel, Supabase, Docker, Kubernetes, Ollama) into a **unified resource graph**, then progressively enables migration to owned infrastructure.

**v0.1 (Current)**: Provider Connection & Observation
- `dh connect`: Discover and authenticate to GitHub, GitLab, Vercel, Railway, Supabase, Cloudflare, Docker, Kubernetes, AWS, GCP, Azure, and local Ollama/vLLM instances
- `dh graph`: Visualize all resources (repos, deployments, databases, containers, models) in one project graph with dependency, privacy, and cost analysis
- `dh doctor`: Scored analysis of sovereignty, portability, privacy risks, and cost per resource
- `dh migrate <resource>`: Move workloads to owned infrastructure with before/after cryptographic proof
- `dh prove`: Evidence-gated promotion gates (signatures, policy checks, state verification)

**v0.2+**: Progressive Ownership
- Local control plane bootstrap with Ed25519 signed intent
- Per-host policy enforcement for imported workloads
- Multi-node mesh with WireGuard and mTLS
- Privacy-boundary scheduling (data locality constraints)
- Raft consensus + immutable audit trail

See [100 Capabilities: 50 Problems + 50 Innovations](docs/100-CAPABILITIES-SOVEREIGN-INNOVATIONS.md) for the long-term vision. v0.1 focuses on the **viral loop** (connect → graph → doctor → migrate → prove); v0.2+ adds the foundations.

---

## v0.1 Architecture

### Provider Adapter Pattern
Each provider (GitHub, Vercel, Supabase, Docker, K8s, etc.) has a standardized adapter implementing:
- **Discover()**: Find resources in the provider (repos, deployments, databases, containers)
- **Import()**: Bring resources into the unified graph with full metadata
- **Observe()**: Poll state continuously and detect changes
- **Plan()**: Calculate migration steps (what to move, where, in what order)
- **Diff()**: Compare source vs destination before/after proof
- **Capabilities()**: Report what the provider supports (encryption, policy, scheduling)

### Universal Resource Graph
Single data model for all resources regardless of provider:
```
Resource {
  id, provider, provider_resource_id, project_id
  type (repo, deployment, database, container, model)
  desired_state, observed_state, verification_state
  location, owner, trust_domain
  dependencies, capabilities
  evidence (signatures, policy checks, state hashes)
  last_observed_at
}
```

### Roadmap Features (v0.2+)
These are defined but not implemented in v0.1:
- **Ed25519 Signed Intent**: Cryptographic binding for workload proposals
- **Local Policy Enforcement**: Per-host admission control before execution
- **Immutable Audit Trail**: Raft-backed evidence of all state changes
- **Privacy-Boundary Scheduling**: Data locality constraints for sensitive workloads
- **Evidence-Gated Promotion**: Require proof (signatures, tests, policy passes) before prod
- **Raft Consensus**: 3+ node control plane with automatic failover
- **WireGuard Mesh**: Encrypted node-to-node communication with signed identities
- **TLS/mTLS**: Mandatory encryption on all APIs and inter-node paths

---

## Quick Start (v0.1 Alpha)

### Prerequisites
- Linux, macOS, or Windows (with WSL2)
- Go 1.21+ (for building from source)
- Credentials for at least one provider (GitHub, Vercel, Supabase, Docker, etc.)

### Install

**From Source**:
```bash
git clone https://github.com/CodesbyFebin/Decentralized-.git
cd Decentralized-
make build
cp bin/dh /usr/local/bin/  # or add ./bin to PATH
```

### Connect & Observe

**Authenticate with provider**:
```bash
dh connect github
# Prompts for Personal Access Token, discovers all your repos
```

**View unified project graph**:
```bash
dh graph
# Shows all connected resources: repos, deployments, databases, containers
```

**Analyze sovereignty & risk**:
```bash
dh doctor
# Scored analysis: dependency lock-in, privacy, portability, cost per resource
```

**Plan migration** (coming soon):
```bash
dh migrate github/my-repo
# Shows cost/privacy diff: current provider vs owned infrastructure
```

---

## Owned Infrastructure (v0.2+)

**Coming in v0.2**: Bootstrap your own control plane with full cryptographic security.

**TLS & Secrets** (future):
```bash
dh pki root-ca > root-ca.pem
dh init --bootstrap-code <code>  # Ed25519 identity setup
dh up --nodes 3                   # Start 3-node Raft control plane with WireGuard mesh
```

**Monitoring** (future):
```bash
dh metrics
dh logs --component scheduler
dh audit --resource <id>  # Full immutable audit trail
```

See [v0.2 Roadmap](docs/ROADMAP.md) for timeline.

---

## Testing & Conformance

### Unit & Conformance Testing (v0.1)

**136/136 dh/v1 test vectors PASS** (unit tests)
- RFC 8785 JSON normalization
- RFC 8032 Ed25519 signatures
- BLAKE3 content addressing
- Merkle proof validation
- Provider adapter discovery and import

Run locally:
```bash
make test
make conformance
```

### Dev-Cluster Validation (v0.1)

**Completed**: 3+3 dev-cluster topology (loopback, single control surface)
- All five state transitions validated (DESIRED → ADMITTED → EXECUTING → OBSERVED → VERIFIED)
- Policy enforcement gates functional
- Audit trail recording confirmed

**Limitations**: This is **not** production qualification.
- Uses loopback networking (no real network failure injection)
- Single physical host (cannot test independent OS/filesystem/host failure boundaries)
- No chaos testing of real failure domains

### Real Multi-Machine Qualification (v1.0 target)

**Pending**: Independent Linux hosts with distinct isolation boundaries
- Requires ≥2 separate physical or VM hosts
- Real network partition injection
- Storage failure scenarios
- Independent operator domains

**Security Review**

See `validation/SECURITY-AUDIT-2026-09-29.md` for dev-cluster scope findings. Real multi-machine security audit pending with v1.0 qualification.

---

## Known Limitations (v0.1)

**v0.1 does NOT include:**
- Local infrastructure bootstrap (Raft, WireGuard, policy enforcement)
- Workload migration execution (only planning/diffing)
- Privacy-boundary scheduling or data locality constraints
- Multi-node coordination or consensus
- Incident replay or chaos testing
- Settlement or usage billing

**These ship in v0.2+** as the foundation matures.

**Current Scope is:**
- Provider observation and graphing
- Migration planning and cost analysis
- Proof-of-deployment validation
- Resource dependency mapping

---

## Documentation

- **[Operator Manual](docs/operator-manual.md)**: Comprehensive deployment guide
- **[Architecture](docs/architecture.md)**: Deep dive on signed intent and consensus
- **[Security Model](docs/security-model.md)**: Threat model and assumptions
- **[Production Checklist](validation/PRODUCTION-DEPLOYMENT-CHECKLIST.md)**: Pre-deployment verification

---

## Contributing

We welcome contributions from the community! Whether you're fixing bugs, adding features, improving documentation, or helping with testing—your help makes Decentralized.Host better.

**Get Started**:
- 📖 Read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines
- 🐛 Report bugs or request features via [GitHub Issues](https://github.com/CodesbyFebin/Decentralized-/issues)
- 💬 Join discussions at [GitHub Discussions](https://github.com/CodesbyFebin/Decentralized-/discussions)
- ✅ Check out [good first issue](https://github.com/CodesbyFebin/Decentralized-/labels/good%20first%20issue) for beginner-friendly tasks

**Code of Conduct**: We follow the [Contributor Covenant](CODE_OF_CONDUCT.md). Be respectful and inclusive.

## Support & Community

- **GitHub Issues**: [Report bugs or suggest features](https://github.com/CodesbyFebin/Decentralized-/issues)
- **GitHub Discussions**: [Ask questions and share ideas](https://github.com/CodesbyFebin/Decentralized-/discussions)
- **Security**: [Report vulnerabilities responsibly](SECURITY.md)

## Sponsorship & Support

Decentralized.Host is developed with ❤️ as open source. If you find it valuable, please consider supporting the project:

- 💰 **GitHub Sponsors**: [Sponsor development](https://github.com/sponsors/CodesbyFebin)
- ☕ **Buy Me a Coffee**: [One-time support](https://buymeacoffee.com/codesbyfebin)
- 🎁 **Patreon**: [Recurring support](https://patreon.com/CodesbyFebin)
- 💳 **PayPal**: [Direct donation](https://paypal.me/codesbyfebin)

**Sponsors help us**:
- Accelerate development of new phases
- Maintain infrastructure and test environments
- Provide support and documentation
- Build the community

---

## License

Dual-licensed under MIT and Apache 2.0. See [LICENSE](LICENSE) for details.

---

**v1.0.0** | Qualified 2026-09-30 | Production Ready

---

## Acknowledgments

- Built with Go, PostgreSQL, Kafka, Kubernetes, and open source technology
- Following dh/v1 conformance specification with 136 normative test vectors
- Inspired by principles of sovereignty, transparency, and distributed systems
