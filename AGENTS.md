# AGENTS.md

project:
  name: Decentralized.Host
  repository: https://github.com/CodesbyFebin/Decentralized-
  domain: sovereign-infrastructure
  status: implemented-through-m8
  primary_language: Go

summary:
  statement: Self-hosted infrastructure where a replicated control plane proposes signed intent and each host retains local admission authority.
  source_of_truth: README.md

principles:
  - signed intent
  - local admission
  - explicit state
  - evidence over claims
  - UNKNOWN is not healthy
  - no phone-home

state_model:
  - desired
  - admitted
  - executing
  - observed
  - verified

implemented:
  - Ed25519 identities and signed assignments/observations
  - local host admission and hold semantics
  - BLAKE3 CAS, FastCDC, and Merkle anti-entropy
  - snapshots and peer repair
  - userspace WireGuard mesh and SWIM gossip
  - root-signed roster and capability chains
  - Raft control plane with mutual TLS
  - L7 edge and ACME flows
  - signed backups and recovery
  - 17 chaos scenarios
  - federation between independent clusters
  - dh/v1 conformance specification
  - 136 conformance test vectors
  - independent Python conformance implementation

explicit_limits:
  - process runtime does not enforce CPU or memory limits
  - erasure coding is not implemented
  - gVisor and Firecracker are detected but not wired as runtimes
  - HTTP/3 is not implemented
  - the mesh uses userspace WireGuard rather than a kernel interface
  - dev clusters can run without TLS unless explicitly enabled

verification:
  protocol: docs/protocol/dh-v1.md
  architecture: docs/architecture.md
  trust_model: docs/trust-model.md
  conformance: pkg/conformance
  integration: tests/integration
  chaos: pkg/chaos
  blueprint: docs/BLUEPRINT.md

rules_for_agents:
  - Keep desired, admitted, executing, observed, and verified state distinct.
  - Never convert UNKNOWN into healthy or successful state.
  - Do not describe gVisor, Firecracker, erasure coding, or HTTP/3 as implemented.
  - Do not infer production scale, uptime, users, customers, or hosted workload counts.
  - Prefer protocol, tests, conformance vectors, and signed evidence over prose claims.
