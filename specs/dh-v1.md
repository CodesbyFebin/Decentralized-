# dh/v1 Conformance Specification

**Version:** v1  
**Status:** NORMATIVE  
**Conformance Test Vectors:** 136  
**Reference Implementation:** Python (tests/conformance/reference/)

## Core Invariants

### 1. Signed Intent (§1)
- All work is proposed as Ed25519-signed intent structures
- Signature verification required before admission
- Unsigned work is rejected with `ErrUnsigned`

**Test Vectors (T001-T012):**
- T001: Valid Ed25519 signature acceptance
- T002: Invalid signature rejection
- T003: Signature verification latency ≤ 1ms
- T004: Batch signature verification (10 intents)
- T005: Concurrent signature verification (100 goroutines)
- T006: Key rotation during verification
- T007: Zero-copy verification path
- T008: Replay attack rejection
- T009: Truncated signature rejection
- T010: Algorithm confusion rejection
- T011: Key binding verification
- T012: Multi-signature aggregation

### 2. Local Policy Enforcement (§2)
- Each host enforces local admission control independent of others
- No global consensus on work admission
- Policy decisions logged to audit trail
- Denial does not require consensus

**Test Vectors (T013-T028):**
- T013: Policy match success path
- T014: Policy match failure path
- T015: Resource verification (CPU, memory, disk)
- T016: Capability matching (runtime, network)
- T017: Identity binding check
- T018: Audit logging completeness
- T019: Latency of policy evaluation ≤ 10ms
- T020: Concurrent policy evaluation (50 requests)
- T021: Policy update atomicity
- T022: Rollback of denied work
- T023: Quarantine of corrupt artifacts
- T024: Remediation state marking
- T025: Policy version tracking
- T026: Signer authorization check
- T027: Reserved resource enforcement
- T028: Capacity deduction correctness

### 3. BLAKE3 Content Addressing (§3)
- All artifacts addressed by BLAKE3-256 hash
- Merkle anti-entropy for replica detection
- Corrupt objects quarantined without consensus
- Content addressing decouples from source

**Test Vectors (T029-T044):**
- T029: BLAKE3 digest consistency
- T030: Chunk boundary independence
- T031: FastCDC gear table correctness
- T032: Merkle tree construction (empty set)
- T033: Merkle tree construction (2+ objects)
- T034: Anti-entropy detection of missing chunks
- T035: Replica comparison by Merkle root
- T036: Corruption detection and quarantine
- T037: Concurrent replica verification
- T038: Chunk ordering independence
- T039: Partial object verification
- T040: Content addressing stability across restarts
- T041: Gear table derivation (domains 0-255)
- T042: Leaf hash (0x00 || id) correctness
- T043: Node hash (0x01 || left || right) correctness
- T044: Empty set hash (0x02) correctness

### 4. Userspace WireGuard (§4)
- Mesh control plane via WireGuard
- Keys signed by node identity
- No unsecured paths for state sync
- Public key binding to node IDs

**Test Vectors (T045-T060):**
- T045: WireGuard interface creation
- T046: Peer key binding to node ID
- T047: Signed key exchange
- T048: Mesh route discovery
- T049: Asymmetric path recovery
- T050: MTU enforcement (1280 minimum)
- T051: Keepalive interval (25s default)
- T052: Concurrent peer management
- T053: Key rotation without service loss
- T054: Peer removal cleanup
- T055: Interface state persistence
- T056: Cross-peer latency measurement
- T057: Bandwidth profiling
- T058: Packet reordering tolerance
- T059: Loss injection tolerance
- T060: Route table convergence

### 5. Raft Consensus + mTLS (§5)
- Leadership election via Raft
- mTLS 1.3 for all control plane traffic
- Snapshot isolation for state machine
- Signed append-entries log

**Test Vectors (T061-T080):**
- T061: Leader election timing ≤ 5s
- T062: Follower commit guarantee
- T063: Log persistence across restarts
- T064: Snapshot creation atomicity
- T065: Snapshot restore correctness
- T066: Concurrent read/write isolation
- T067: mTLS handshake latency ≤ 200ms
- T068: Certificate verification
- T069: TLS session reuse
- T070: Peer identity from certificate
- T071: Clock skew tolerance (30s)
- T072: Partition recovery
- T073: Cascading failure tolerance
- T074: Quorum check correctness
- T075: Leader step-down on partition
- T076: Append-entries log signing
- T077: Log entry deduplication
- T078: Stale read detection
- T079: Linearizable read guarantee
- T080: Transparent follower redirect

### 6. ACME Certificate Automation (§6)
- Automatic TLS certificate provisioning
- HTTP-01, DNS-01, and wildcard support
- Renewal before expiration
- Pebble staging environment support

**Test Vectors (T081-T096):**
- T081: HTTP-01 challenge completion
- T082: DNS-01 challenge completion
- T083: Certificate issuance
- T084: Certificate installation
- T085: Renewal triggering (7 days before expiry)
- T086: Wildcard certificate handling
- T087: SAN certificate handling
- T088: Challenge retry on failure
- T089: Backoff on rate limit
- T090: Concurrent challenge handling
- T091: ACME account persistence
- T092: Certificate chain verification
- T093: Pebble integration test
- T094: Production CA integration readiness
- T095: Certificate revocation support
- T096: OCSP stapling support

### 7. Explicit State Machine (§7)
- Five distinct states: DESIRED → ADMITTED → EXECUTING → OBSERVED → VERIFIED
- State transitions are explicit, not implicit
- Audit trail records every transition
- No silent state changes

**Test Vectors (T097-T110):**
- T097: Work proposed (DESIRED)
- T098: Policy match (ADMITTED)
- T099: Execution start (EXECUTING)
- T100: Completion observed (OBSERVED)
- T101: Hash verification (VERIFIED)
- T102: Rollback from EXECUTING
- T103: Retry on OBSERVED failure
- T104: Timeout from EXECUTING (30s)
- T105: State persistence across restarts
- T106: Concurrent state updates
- T107: Atomicity of state transitions
- T108: Audit entry for each transition
- T109: Recovery from partial transition
- T110: State query consistency

### 8. Failure Detection & Recovery (§8)
- Health probes on fixed intervals (default: 15s)
- Failure detection latency ≤ 30s
- Workload redistribution automatic
- Recovery without coordination

**Test Vectors (T111-T122):**
- T111: Health probe latency ≤ 100ms
- T112: Failure detection latency ≤ 30s
- T113: False positive rate < 0.1%
- T114: Workload redistribution latency ≤ 10s
- T115: No silent task migration
- T116: Recovery with partition tolerance
- T117: Cascading failure containment
- T118: Replica rebalancing
- T119: Storage recovery atomicity
- T120: Audit trail of all migrations
- T121: Reconciliation convergence
- T122: Observer-driven state verification

### 9. Chaos & Invariant Validation (§9)
- 17 defined failure scenarios
- Invariants verified under sustained load (1000+ ops/sec)
- No data loss, corruption, or silent failures
- Observable behavior matches specification

**Test Vectors (T123-T136):**
- T123: Node crash recovery
- T124: Network partition (2-way)
- T125: Network partition (1-way)
- T126: Clock skew injection
- T127: Storage corruption (chunk loss)
- T128: Storage corruption (bit flip)
- T129: Concurrent updates to same key
- T130: Cascade failure (sequential node loss)
- T131: Disk full condition
- T132: Memory pressure
- T133: High latency paths (>1000ms)
- T134: Packet loss (10-30%)
- T135: Certificate expiration during operation
- T136: Graceful shutdown

## Conformance Claim

A system claims dh/v1 conformance when:
1. All 136 test vectors PASS
2. No simulation or hardcoding in evidence
3. Observable production behavior under real failure modes
4. Audit trail is complete and verifiable
5. No silent task migration
6. Local policy enforced per-host

## Implementation Notes

- Reference implementation in Python covers all vectors
- Go implementation at pkg/conformance/
- Test harness validates both reference and implementations
- CI/CD runs full vector suite on every commit
- Evidence binding includes source SHA and campaign timestamp

---

*Last updated: 2026-09-29 by dh/v1 conformance working group*
