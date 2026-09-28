# P1-MESH-A01: Distributed Mesh Networking & Consensus

**Phase:** P1-MESH (follows P1-FAILURE-A01)  
**Objective:** Verify distributed consensus and mesh network resilience  
**Status:** DESIGN COMPLETE, awaiting P1-FAILURE-A01 completion

---

## Overview

P1-MESH-A01 tests the system's mesh networking layer, distributed consensus, and Byzantine fault tolerance. Tests verify:

1. **Mesh Connectivity**: All nodes can communicate via the mesh (not just TCP)
2. **Gossip Protocol**: Node state propagates across mesh
3. **Consensus**: Nodes reach agreement on canonical state
4. **Byzantine Tolerance**: System tolerates 1 dishonest node (N≥4 assumed)
5. **Network Partition Recovery**: Mesh heals after partition

---

## Gates 49-64 (16 gates)

### Gates 49-52: Mesh Connectivity
**Gate 49:** Mesh network initialized on all nodes  
- **Metric:** WireGuard (or equivalent) interface active on all 3 nodes
- **Evidence:** `ip link show | grep wg0` returns UP on all nodes
- **Failure scenario:** Interface DOWN on any node

**Gate 50:** Mesh encryption working end-to-end  
- **Metric:** Packets encrypted with node's private key, decrypted with peer's public key
- **Evidence:** Packet capture shows encrypted IP protocol, UDP 51820+ (WireGuard)
- **Failure scenario:** Traffic in plaintext

**Gate 51:** Mesh routing functional  
- **Metric:** Nodes can ping each other via mesh IPs (10.0.X.0/24 range)
- **Evidence:** ICMP echo request/reply via mesh interface, RTT < 100ms
- **Failure scenario:** Ping timeout, nodes unreachable

**Gate 52:** Mesh peer discovery automatic  
- **Metric:** New node joins mesh and auto-discovers peers without manual config
- **Evidence:** `wg show` lists all peer public keys automatically
- **Failure scenario:** New node isolated, manual intervention required

### Gates 53-56: Gossip Protocol
**Gate 53:** Gossip protocol active (heartbeats)  
- **Metric:** Nodes exchange heartbeat/state messages every 10s (configurable)
- **Evidence:** Mesh packet capture shows regular gossip traffic
- **Failure scenario:** No gossip traffic observed

**Gate 54:** Node state propagates via gossip  
- **Metric:** When one node changes state, other nodes receive update within 30s
- **Evidence:** Ledger update on node A appears on node B's copy
- **Failure scenario:** State change visible only on originating node

**Gate 55:** Gossip protocol tolerates message loss  
- **Metric:** With 20% packet loss, state still propagates correctly
- **Evidence:** State converges even with dropped messages
- **Failure scenario:** State diverges permanently with packet loss

**Gate 56:** Gossip acknowledges quorum delivery  
- **Metric:** Sender confirms message delivered to majority (2/3) of nodes
- **Evidence:** ACK count in gossip message ≥ 2
- **Failure scenario:** Sends without waiting for quorum ACK

### Gates 57-60: Consensus (Raft/consensus protocol)
**Gate 57:** Leader election successful on init  
- **Metric:** Within 30s of mesh startup, one node elected leader
- **Evidence:** One node has `role: "leader"`, others have `role: "follower"`
- **Failure scenario:** No leader elected (multiple leaders or no leader)

**Gate 58:** Followers replicate leader's log  
- **Metric:** Follower nodes maintain identical copies of leader's ledger
- **Evidence:** Ledger entries and sequence numbers match across all nodes
- **Failure scenario:** Log divergence (followers have different entries)

**Gate 59:** Consensus tolerates follower crash  
- **Metric:** After follower crashes, leader continues, follower catches up on restart
- **Evidence:** Leader can commit entries while follower is down
- **Failure scenario:** System blocks waiting for dead follower

**Gate 60:** Leader re-election after leader crash  
- **Metric:** If leader crashes, new leader elected within 60s
- **Evidence:** Different node becomes leader, new term number incremented
- **Failure scenario:** No new leader elected, system frozen

### Gates 61-64: Byzantine Fault Tolerance
**Gate 61:** Minority Byzantine node cannot disrupt consensus  
- **Metric:** 1 dishonest node (out of 3) cannot force bad state
- **Evidence:** Canonical ledger agreed by 2+ nodes despite 1 node lying
- **Failure scenario:** False data committed because Byzantine node won

**Gate 62:** Byzantine node cannot forge signatures  
- **Metric:** Signatures require node's private key, cannot be forged
- **Evidence:** Signature verification with node's public key always succeeds/fails correctly
- **Failure scenario:** Forged signature accepted as valid

**Gate 63:** Byzantine node cannot rewrite history  
- **Evidence:** Cannot modify log entries from past (immutable ledger)
- **Evidence:** Log entry hash cannot match if content modified
- **Failure scenario:** Byzantine node edits old log entry

**Gate 64:** Byzantine node detected and isolated  
- **Metric:** System detects misbehavior and excludes node from consensus
- **Evidence:** Byzantine node's consensus votes ignored, node marked "untrusted"
- **Failure scenario:** Byzantine node continues participating

### Gates 65-68: Network Partition Recovery (continuation for 4-gate wraparound)
**Gate 65:** Network partition splits mesh into islands  
- **Metric:** Partition injected, nodes in partition A cannot reach partition B
- **Evidence:** Network ACLs block traffic between partitions
- **Failure scenario:** Nodes can still communicate across partition

**Gate 66:** Partitioned nodes detect split  
- **Metric:** Nodes detect loss of peer heartbeats within 60s
- **Evidence:** Peer marked "unreachable" in mesh status
- **Failure scenario:** Partition goes undetected, stale state persists

**Gate 67:** Partition heals automatically  
- **Metric:** When partition is removed, nodes re-connect within 120s
- **Evidence:** Mesh topology restored, peers marked "connected"
- **Failure scenario:** Manual intervention required to heal mesh

**Gate 68:** State converges after partition heals  
- **Metric:** Conflicting state from partition resolved via quorum consensus
- **Evidence:** Canonical state from partition with 2+ nodes wins
- **Failure scenario:** Divergent state persists, split-brain

---

## Execution Flow

```
[P1-FAILURE-A01 PASS]
        ↓
[Initialize mesh network on 3 nodes]
        ↓
[Baseline: 3 workloads, 120s via mesh]
        ↓
[Gates 49-52: Mesh connectivity verification]
        ↓
[Inject load: 10Mbps traffic via mesh]
        ↓
[Gates 53-56: Gossip protocol under load]
        ↓
[Elect leader, verify log replication]
        ↓
[Gates 57-60: Consensus and leader election]
        ↓
[Inject Byzantine node: Send conflicting updates]
        ↓
[Gates 61-64: Byzantine tolerance, signature verification]
        ↓
[Inject network partition: Block partition A ↔ B]
        ↓
[Gates 65-68: Partition detection, healing, convergence]
        ↓
[Generate P1-MESH-A01 report (gates 49-68 → combined with P1-CLOSE + P1-FAILURE)]
        ↓
[Proceed to P1-EVIDENCE-A01]
```

---

## Key Metrics

| Metric | Target | Method |
|--------|--------|--------|
| Mesh startup time | <30s | Time to all nodes UP in `wg show` |
| Gossip propagation delay | <30s | Time from one node change to all nodes converge |
| Leader election time | <30s | Time from startup to single leader elected |
| Partition detection | <60s | Time to detect lost peer after network split |
| Partition healing | <120s | Time to re-establish connectivity after repair |
| Byzantine node isolation | <120s | Time to exclude misbehaving node |
| Consensus commit latency | <5s | Time from leader commit to log replication |

---

## Mesh Protocol Assumptions

- **Mesh Type:** WireGuard (or equivalent VPN mesh layer)
- **Consensus Type:** Raft (or Raft-variant)
- **Gossip Type:** Epidemic/push-based gossip with ACKs
- **Byzantine Assumption:** Crash failures + Byzantine (1-of-N dishonest)
- **Latency Model:** LAN-like (<100ms), TCG-limited throughput

---

## Repository Integration

```
validation/local-vm/scripts/
├── init-mesh-network.sh          ← NEW: Initialize WireGuard mesh
├── verify-mesh-connectivity.sh   ← NEW: Test mesh reachability
├── inject-byzantine-node.sh      ← NEW: Simulate Byzantine node
├── inject-partition.sh           ← REUSE: Network partition injection
├── measure-gossip-latency.sh     ← NEW: Measure gossip propagation
└── verify-consensus.sh           ← NEW: Verify Raft state + leader election
```

---

## Success Criteria

- ✅ Mesh topology established on all nodes
- ✅ Gossip protocol propagates state within 30s
- ✅ Consensus leader elected, followers replicate log
- ✅ Byzantine node cannot corrupt state
- ✅ Network partition detected and healed
- ✅ State convergence after partition heal
- ✅ All gates achieve PASS status

---

## Blocking Conditions

- ❌ P1-CLOSE-A01 gates 1-32 must PASS
- ❌ P1-FAILURE-A01 gates 33-48 must PASS
- ❌ External blockers documented if encountered

---

## Next Phase: P1-EVIDENCE-A01

Upon P1-MESH-A01 completion, proceed to final phase:
- Cryptographic evidence sealing
- External verifier compatibility
- Archive and release readiness

---

## Rollup: Combined Gate Report (P1-A01)

Final report will roll up all phases:
- **P1-CLOSE (Gates 1-32):** Baseline execution, failure detection, immediate recovery
- **P1-FAILURE (Gates 33-48):** Persistence, recovery, evidence immutability
- **P1-MESH (Gates 49-68):** Distributed consensus, Byzantine tolerance, partition recovery

**Total:** 68 gates across 3 integrated phases = **P1-A01 QUALIFICATION**

