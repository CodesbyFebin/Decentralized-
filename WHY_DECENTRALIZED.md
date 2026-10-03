# Why Decentralized.Host?

## The Problem

You're running infrastructure and you have choices:

- **Kubernetes:** Powerful but centralized scheduling, complex networking, vendor-specific extensions
- **Nomad:** Flexible but still requires central coordination, external databases, agent complexity
- **Cloud APIs (AWS EC2, Lambda, GCP):** Easy but vendor lock-in, cost surprises, data leaves your control, API changes break your code
- **Docker Swarm:** Simple but limited, not suitable for production multi-region scenarios

**What's missing?** A system where:
- ✗ Hosts keep final say (no central override)
- ✗ Work is cryptographically signed (can't be silently reassigned)
- ✗ Storage is P2P (no central database)
- ✗ Everything is audit-verified locally
- ✗ You own the infrastructure completely

## The Decentralized.Host Answer

Decentralized.Host is an **infrastructure control plane designed from first principles for sovereignty**.

### Five Core Principles

**1. Sovereignty Over Consensus**
- Each host has local policy
- No centralized scheduler overrides
- No "blessed" compute tier
- Every host decides what runs

```
Traditional: [Scheduler] → Host (execute or be replaced)
DH:         Host → [Local Policy] → Execute or Reject
```

**2. Signed Intent Over Silent Mutations**
- Every work assignment is Ed25519-signed
- Every observation is cryptographically proven
- No silent task migration
- Full audit trail

```
Traditional: Scheduler decides → Host runs (maybe)
DH:         Operator proposes (signed) → Host admits (signed) → Host executes & reports (signed)
```

**3. Peer-to-Peer Storage Over Central Gateway**
- BLAKE3 content-addressed storage
- Peer healing via Merkle anti-entropy
- No single point of failure
- Works when any 2+ nodes are alive

```
Traditional: App → Database ← All nodes depend on it
DH:         Node1 ←→ Node2 ←→ Node3 (gossip, automatic repair)
```

**4. Local Measurement Over Assertion**
- Observed state is measured (not claimed)
- DESIRED ≠ ADMITTED ≠ EXECUTING ≠ OBSERVED ≠ VERIFIED
- Unknown stays unknown (no guessing)
- Operators can verify locally

```
Traditional: "Status: Running" (take it or leave it)
DH:         OBSERVED: {cpu: 50%, memory: 2.1GB, verified: yes, timestamp: 2026-10-02T14:42Z}
```

**5. Self-Hosted Over Cloud Dependency**
- Operator console runs on the cluster itself
- No external APIs or callbacks
- Works offline completely
- No telemetry, no vendor tracking

```
Traditional: App → Cloud Dashboard (requires internet, login, API key, monthly bill)
DH:         ./bin/dh get apps (runs locally, cryptographically verified)
```

---

## Why This Matters

### Use Case 1: Mission-Critical On-Premises

**Scenario:** You run a factory, hospital, or utility that can't have external dependencies.

**Traditional approach fails:**
- Kubernetes depends on etcd (single cluster of trust)
- AWS goes down → cluster goes down
- Network outage → can't reach cloud console
- Vendor updates break your code

**Decentralized.Host succeeds:**
- Each machine decides independently
- Network partition? Machines keep running
- Offline? Still fully operational
- Self-hosted console works without internet
- Zero vendor updates break your workflow

### Use Case 2: Data Sovereignty

**Scenario:** You have compliance requirements (GDPR, HIPAA, CCPA) that demand data stays local.

**Traditional approach fails:**
- Kubernetes pods might migrate anywhere
- Cloud providers may replicate data
- Compliance team can't audit the network
- Vendor could theoretically access your data

**Decentralized.Host succeeds:**
- Each host enforces local policy
- Data stays on-premises (encrypted at rest)
- Full audit trail (cryptographically verified)
- No vendor can override your policy
- Compliance team can verify the hash chain

### Use Case 3: Cost Control

**Scenario:** You want predictable, auditable infrastructure costs.

**Traditional approach fails:**
- Kubernetes: You pay for the cluster even if idle
- AWS Lambda: 10x cost spike if traffic unexpectedly rises
- GCP: Per-API-call billing model is hard to predict
- Nomad: Still requires external databases, support contracts

**Decentralized.Host succeeds:**
- Your machines, your power bill (no per-request charges)
- No surprise API costs
- No vendor lock-in licensing
- Predictable resource usage (local policy enforces limits)
- Open source (no support contracts required)

### Use Case 4: Offline-First Operations

**Scenario:** You have field offices, remote locations, or need to survive network failures.

**Traditional approach fails:**
- Kubernetes: Needs etcd quorum (usually 3+ members)
- Any cloud system: Requires internet connectivity
- Central scheduler: Unreachable = cluster down

**Decentralized.Host succeeds:**
- Each host runs independently
- Network partition? Keep operating
- Central control plane down? Hosts keep running
- Eventual consistency when network heals
- No dependency on external connectivity

---

## How It's Different

### vs. Kubernetes
| Aspect | Kubernetes | DH |
|--------|-----------|----| 
| Scheduling | Centralized (kube-scheduler) | Per-host policy |
| Storage | Central etcd (required) | P2P Merkle (optional) |
| Networking | CNI plugins (complex) | Userspace WireGuard (simple) |
| Audit trail | Per-node logs | Signed, hash-chained, tamper-proof |
| Offline mode | No (etcd quorum required) | Yes (full operation) |
| Vendor lock-in | High (cloud-specific operators) | Zero (portable) |

### vs. Nomad
| Aspect | Nomad | DH |
|--------|-------|-----| 
| Admission control | Central | Per-host (sovereign) |
| Storage | Requires external DB | Built-in P2P |
| Trust model | Trusts operators | Verifies signatures |
| Failure tolerance | Depends on database | Any 2+ nodes |
| Compliance audit | Via logs (user must verify) | Cryptographically proven |
| Licensing | Open source (free) | Open source (free) |

### vs. Docker Swarm
| Aspect | Swarm | DH |
|--------|-------|-----| 
| Scale | ~1000 nodes | Tested to 100+, designed for 1000+ |
| Failures | Limited chaos testing | 17 verified chaos scenarios |
| Storage | None built-in | BLAKE3 CAS with Merkle healing |
| Audit trail | None | Full signed ledger |
| Production-ready | Limited | 10/10 qualification gates pass |

### vs. Cloud APIs (AWS, GCP, Azure)
| Aspect | Cloud | DH |
|--------|-------|-----| 
| Sovereignty | No (vendor owns it) | Yes (you own hardware) |
| Cost | Per-request (unpredictable) | Hardware only (predictable) |
| Vendor lock-in | Extreme | Zero |
| Data location | Vendor decides | You decide |
| Compliance audit | Trust vendor | Cryptographically verify |
| Offline capable | No | Yes |
| Internet dependent | Yes | No |

---

## Real-World Performance

### Failover Under Load
- **Setup:** 3-node cluster, 1000 ops/sec sustained traffic
- **Failure:** Leader killed (SIGKILL)
- **Result:** ~1.2 seconds to new leader, zero failed requests
- **Proof:** [See tests/integration/m5_test.go](tests/integration/m5_test.go)

### Network Partition Recovery
- **Setup:** 3-node mesh, 2-way partition (node-1 and node-3 isolated)
- **Failures:** Packet chaos, 30+ second partition
- **Result:** Automatic healing, eventual consistency, zero data loss
- **Proof:** [See tests/chaos/network_partition_test.go](tests/chaos/)

### Storage Corruption Detection
- **Setup:** BLAKE3-protected object storage
- **Injection:** Bit-flip corruption in stored object
- **Result:** Detected and repaired from peer within 2 seconds
- **Proof:** [See tests/chaos/storage_corruption_test.go](tests/chaos/)

---

## When to Use Decentralized.Host

### ✅ Perfect For
- On-premises infrastructure (factory, hospital, utility)
- Data sovereignty requirements (GDPR, HIPAA, CCPA)
- Air-gapped or offline-first operations
- Cost-sensitive environments (no per-request billing)
- Compliance-heavy industries
- Organizations that reject vendor lock-in
- Building "infrastructure as code" for controlled environments
- Edge computing and distributed sites

### ⚠️ Consider Alternatives If
- You need extreme scale (1000s+ nodes) — DH is designed for 100s
- You want managed services (let vendor handle operations)
- You require Kubernetes ecosystem (specific operators, tools)
- Your workloads are extremely dynamic (function-based billing models)
- You need CNCF certification (Kubernetes focus)

---

## Getting Started (5 Minutes)

```bash
# Clone
git clone https://github.com/CodesbyFebin/Decentralized-
cd Decentralized-

# Build (requires Go 1.26+)
make build

# Start 3-node dev cluster (real processes, real sockets, real WireGuard)
./bin/dh dev up --dir ./devcluster

# In another terminal
export DH_HOME=./devcluster/operator

# List deployed apps
./bin/dh get apps

# Check mesh health
./bin/dh mesh peers

# Verify audit trail locally
./bin/dh audit verify

# Simulate chaos (kill leader)
./bin/dh chaos run --scenario leader-crash

# Cleanup
./bin/dh dev down --dir ./devcluster
```

**Expected result:** Full cluster running, leader election, failover, audit trail verified. ✅

---

## Community & Support

- **GitHub:** https://github.com/CodesbyFebin/Decentralized-
- **Issues:** [Report bugs, request features](https://github.com/CodesbyFebin/Decentralized-/issues)
- **Discussions:** [Ask questions, share ideas](https://github.com/CodesbyFebin/Decentralized-/discussions)
- **Contributing:** [Join the project](CONTRIBUTING.md)
- **Documentation:** [Full protocol and runbooks](docs/)

---

## The Vision

**Decentralized.Host** is built on a belief: infrastructure should be owned and controlled by the people who run it, not by vendors who rent it.

We're building the operating system for self-hosted, sovereign infrastructure. Where:
- Hosts are trusted to make decisions
- Work is proven, not asserted
- Storage is communal, not siloed
- Operators are in control, not vendors

**This is infrastructure for people who want to own their compute.**

---

<div align="center">

**[Get Started](README.md#quick-start-5-minutes) · [Read the Spec](docs/protocol/dh-v1.md) · [Run Chaos Tests](Makefile) · [Star on GitHub](https://github.com/CodesbyFebin/Decentralized-)**

*Made for self-hosted infrastructure where hosts stay sovereign.*

</div>
