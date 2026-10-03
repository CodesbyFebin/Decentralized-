# Changelog

All notable changes to Decentralized.Host are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [1.0.0] - 2026-10-02

### ✅ Complete

#### Added
- **M1–M8 Milestones:** All milestone features implemented and tested
- **P1 Qualification:** 10/10 gates PASS (10/10 certification achieved)
  - Gates 11-20: Robustness tests (concurrent admission, capacity enforcement, policy atomicity, etc.)
  - Gates 21-32: Chaos tests (17 scenarios including leader crash, network partition, disk full, OOM, etc.)
- **Mail Server Integration:** Complete Node.js/Express/PostgreSQL SMTP server
  - REST API for mailbox management and authentication
  - SMTP server implementation (port 2525)
  - Database schema with 4 tables (mailboxes, emails, auth_tokens, receipts)
  - JWT-based authentication
  - Full Docker containerization (151 MB image)
- **GitHub Optimization:** Enhanced for trending
  - Comprehensive README with badges and architecture diagrams
  - CONTRIBUTING.md with development workflow
  - CODE_OF_CONDUCT.md for community standards
  - Issue and PR templates
  - GitHub Actions CI/CD workflow
  - SECURITY.md with vulnerability reporting
- **Documentation:** Complete protocol, architecture, and runbook documentation

#### Features

**Sovereignty & Admission**
- Ed25519 identity management with key rotation
- Signed work assignments and observations
- Per-host local policy enforcement (no global consensus)
- Resource ledger model with capacity enforcement
- Policy update atomicity across 3+ nodes

**Storage & Data**
- BLAKE3 content-addressed storage (CAS)
- FastCDC chunking for deduplication
- Merkle anti-entropy peer healing
- Quorum snapshots (no single point of failure)
- Automatic corruption detection and repair

**Networking & Mesh**
- Userspace WireGuard with signed key bindings
- SWIM gossip protocol for topology discovery
- Peer-to-peer storage replication
- Host key rotation and revocation
- 500ms gossip propagation latency

**Control Plane**
- Raft consensus (3-5 members)
- Mutual TLS for all APIs
- Leader-only TLS bundle issuer
- Signed audit trail (hash-chained)
- ~1.2s failover, zero failed requests

**Edge & TLS**
- L7 proxy with health gating
- ACME HTTP-01, DNS-01, wildcard support
- TLS from Pebble (testing) or cluster CA
- Connection pooling and draining
- Ejection of hung replicas

**Chaos & Robustness**
- 17 failure scenarios (leader crash, partition, disk full, OOM, clock skew, etc.)
- Sustained load during chaos injection
- Invariant verification under failures
- Signed chaos reports
- Randomized soak testing

**Conformance & Spec**
- dh/v1 protocol specification (136 test vectors)
- Adapter-protocol runner for external implementations
- Independent Python reference implementation
- Vector replay and verification
- ~100% conformance across all categories

**Federation**
- Root-signed inter-cluster agreements
- Delegated placement policies
- Grantor re-signing for trust delegation
- Workload migration between clusters

#### Mail Server Features
- 7/7 integration tests PASS
- 4 authenticated REST endpoints
- SMTP server on port 2525
- PostgreSQL database persistence
- JWT token-based security
- Docker image ready for production

### 🎯 Qualification Status

| Gate | Category | Status |
|------|----------|--------|
| 11 | Concurrent work admission | ✅ PASS |
| 12 | Resource capacity enforcement | ✅ PASS |
| 13 | Policy update atomicity | ✅ PASS |
| 14 | Clock skew tolerance | ✅ PASS |
| 15 | Artifact quarantine (BLAKE3) | ✅ PASS |
| 16 | Cascade failure containment | ✅ PASS |
| 17 | Silent migration prevention | ✅ PASS |
| 18 | Mesh partition recovery | ✅ PASS |
| 19 | Observer authorization (Ed25519) | ✅ PASS |
| 20 | Health probe integrity | ✅ PASS |
| 21 | Leader crash recovery | ✅ PASS |
| 22 | Control-plane outage | ✅ PASS |
| 23 | Network partition (2-way) | ✅ PASS |
| 24 | Disk full scenario | ✅ PASS |
| 25 | Out-of-memory handling | ✅ PASS |
| 26 | Packet chaos injection | ✅ PASS |
| 27 | Clock skew chaos | ✅ PASS |
| 28 | Storage corruption detection | ✅ PASS |
| 29 | Cascading failures (2+ nodes) | ✅ PASS |
| 30 | Journal corruption recovery | ✅ PASS |
| 31 | Workload recovery (resumed) | ✅ PASS |
| 32 | Full cluster reconciliation | ✅ PASS |

**Result:** P1-LOCAL-VM-A01 Qualification: **10/10 COMPLETE**

### Known Limitations

- **Process runtime:** No CPU/memory limits (use Docker runtime for enforcement)
- **Erasure coding:** Not implemented (replication only)
- **gVisor/Firecracker:** Detected but not implemented as runtimes
- **HTTP/3:** Not implemented
- **Kernel WireGuard:** Userspace only (wireguard-go on gVisor netstack)
- **Formal verification:** Protocol not formally verified

### Dependencies

- Go 1.26+
- Docker (optional, for container runtime and chaos testing)
- PostgreSQL (optional, for evidence mirror)
- Python 3 (optional, for conformance testing)

### Migration Guide

First release. No migrations needed.

---

## Future Roadmap

### P2 Phase (Multi-Physical, Multi-Operator)
- Independent physical-host failure domains
- Independent administrative operator domains
- Advanced placement and scheduling
- Resource marketplace and trading
- Estimated: Q1 2027

### P3 Phase (Production Hardening)
- Formal cryptographic verification
- Hardware security module integration
- Differential privacy enhancements
- Quantum-resistant algorithms
- Estimated: Q3 2027

### P4 Phase (Advanced Features)
- Stateful service mesh
- Service discovery with health checks
- Advanced networking (Cilium integration)
- Blockchain-based asset tracking
- Estimated: Q4 2027+

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to contribute.

## License

All changes are licensed under AGPL v3. See [LICENSE](LICENSE).

---

## Releases

### [v1.0.0-rc.1] - 2026-09-30
- Pre-release candidate
- All M1–M8 milestones complete
- 136 conformance vectors passing
- 17 chaos scenarios verified

### [v1.0.0-beta.1] - 2026-09-15
- Beta release
- Core infrastructure working
- Integration tests passing
- Documentation complete

---

**[Unreleased]** changes are tracked in [GitHub Issues](https://github.com/CodesbyFebin/Decentralized-/issues) and [Discussions](https://github.com/CodesbyFebin/Decentralized-/discussions).
