# Decentralized.Host (dh) — Sovereign Infrastructure with Signed Intent

**Status**: v1.0.0 Production Release  
**Qualification**: P1_CORE Approved (32/32 gates PASS)  
**License**: MIT/Apache 2.0

> Infrastructure where work is proposed as signed cryptographic intent, every host checks it against local policy before execution, and all state changes are immutably recorded. No silent task migration. Your cluster, your rules.

---

## What This Is

Decentralized.Host is a distributed workload orchestrator built on three core principles:

1. **Signed Intent**: All work is proposed as cryptographically signed requests using Ed25519 identities. No anonymous or forgeable instructions.
2. **Local Policy Enforcement**: Every host independently evaluates incoming work against its own policy before admission. No global consensus on what runs where.
3. **Immutable Evidence**: All state transitions are recorded in a Raft-backed audit trail with cryptographic signatures. Full deterministic replay capability.

This eliminates silent task migration, enforces operator authority at each machine, and provides irrefutable evidence of what ran and when.

---

## Key Features

### Cryptography First
- **Ed25519**: Identity binding for all actors (operators, hosts, workloads)
- **TLS 1.3**: Enforced on all operator APIs and inter-node communication
- **AES-256-GCM**: Secrets at-rest encryption with DEK/KEK separation
- **mTLS**: Mutual authentication on all mesh communication
- **BLAKE3**: Content-addressed storage with Merkle anti-entropy

### Explicit State Machine
Five observable states for every workload:
- **DESIRED** → Work proposed with signed intent
- **ADMITTED** → Local policy approved execution
- **EXECUTING** → Container running on host
- **OBSERVED** → State verified by host observation
- **VERIFIED** → State recorded in audit trail

### Failure Domain Awareness
- **Node Failures**: Detects and responds to unreachable nodes
- **Network Partitions**: Identifies split-brain scenarios and halts execution
- **Storage Corruption**: Validates Merkle proofs, rejects corrupted state
- **Automatic Recovery**: Restarts failed workloads with audit trail preservation

### Production Infrastructure
- **Raft Consensus**: 3+ member control plane with leader election
- **Mesh Networking**: Userspace WireGuard with signed key bindings
- **TLS/mTLS**: Mandatory mutual authentication on all paths
- **ACME Integration**: Pebble for test, standard ACME for production

---

## Quick Start

### Prerequisites
- Linux (Kernel 5.10+) with cgroups v2
- Go 1.21+ (for building from source)

### Install

**From Source**:
```bash
git clone https://github.com/CodesbyFebin/Decentralized-.git
cd Decentralized-
make build
sudo cp bin/dh* /usr/local/bin/
```

### Bootstrap Cluster

**Start a local 4-node cluster** (for testing):
```bash
dh dev up
```

**Check cluster health**:
```bash
dh get nodes
dh cp status
```

**Submit work**:
```bash
dh apply -f workload.yaml
dh get apps
```

---

## Production Deployment

### TLS Configuration

**Generate Bootstrap Material**:
```bash
dh pki root-ca > root-ca.pem
dh pki bootstrap > bootstrap-code.txt
```

**Bootstrap Cluster**:
```bash
export DECENTRALIZED_KEK_BOOTSTRAP="<bootstrap-material>"
dh-control -data /var/lib/dh/member-0
```

### Monitoring

**Prometheus Metrics** (port 19090):
```bash
curl http://localhost:19090/metrics | grep dh_raft_leader_known
```

See `validation/MONITORING-ALERTING-CONFIG.md` for complete setup.

---

## Qualification & Verification

### P1_CORE Qualification (v1.0.0)

All 32 production gates passed on live 4-node cluster:

| Category | Gates | Status |
|----------|-------|--------|
| Signed Intent & Policy | 01-08 | ✅ PASS |
| State Machine | 09-16 | ✅ PASS |
| Failure Detection | 17-24 | ✅ PASS |
| Evidence & Verification | 25-32 | ✅ PASS |

**Campaign ID**: `P1_CORE_OFFICIAL_20260930_001133`

### Conformance Testing

**136/136 dh/v1 test vectors PASS**
- RFC 8785 JSON normalization
- RFC 8032 Ed25519 signatures
- BLAKE3 content addressing
- Merkle proof validation

Run locally:
```bash
make conformance
```

### Security Audit

**Zero critical findings**  
See `validation/SECURITY-AUDIT-2026-09-29.md`

---

## Known Limitations (v1.0.0)

- Single-region, single-cluster deployment
- 4-node tested topology (larger clusters supported operationally)
- Local storage backend (distributed storage post-v1.0)
- Post-quantum cryptography is Phase 2

---

## Documentation

- **[Operator Manual](docs/operator-manual.md)**: Comprehensive deployment guide
- **[Architecture](docs/architecture.md)**: Deep dive on signed intent and consensus
- **[Security Model](docs/security-model.md)**: Threat model and assumptions
- **[Production Checklist](validation/PRODUCTION-DEPLOYMENT-CHECKLIST.md)**: Pre-deployment verification

---

## Support & Community

- **GitHub Issues**: https://github.com/CodesbyFebin/Decentralized-/issues
- **Discussions**: https://github.com/CodesbyFebin/Decentralized-/discussions

---

## License

Dual-licensed under MIT and Apache 2.0.

---

**v1.0.0** | Qualified 2026-09-30 | Production Ready
