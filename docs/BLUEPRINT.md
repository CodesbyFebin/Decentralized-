# DECENTRALIZED.HOST
## Production Blueprint — implementation-aware revision
### Host Anywhere. Run Everywhere. Own the Infrastructure.

**Revision:** 2 (replaces the "M2+ Master Build Blueprint", which described M2–M8 as planned work)
**Evidence date:** 2026-09-24 · **Protocol:** `dh/v1` · **Build:** this repository, Go 1.26.4

The first blueprint was a roadmap. This revision is a statement of fact plus a
plan. Every capability is filed under exactly one of these labels:

| Label | Meaning |
|---|---|
| **VERIFIED** | Implemented, and exercised by an automated test or chaos scenario that runs real processes, sockets and failures. The evidence is named. |
| **IMPLEMENTED — LIMITED** | Works and is used, but has a stated limitation, or has only been exercised manually or in a narrower setting than production. |
| **NOT IMPLEMENTED** | Absent. Where the system can detect the gap, it reports it (for example `NOT AVAILABLE`, or `not enforced` in admission details). |
| **HARDENING** | Required before a production release; tracked in §12. |

The labels obey the blueprint's own truthfulness rule: nothing is filed
higher than its evidence supports. **Evidence scope applies to every row:**
all runs so far are **multi-process on a single macOS (darwin/amd64)
machine over loopback.** No test has crossed physical machines, a real WAN,
NAT, or a Linux host. That single fact is the largest open item (§12, P0-1).

---

# 1. Mission and principles (unchanged)

Decentralized.Host lets independent operators run applications on hardware
they control, coordinated by a control plane that can **propose** but never
**command**. The product principles still stand, and each one now has an
enforcement point:

| Principle | Enforced by | Status |
|---|---|---|
| Self-hosted by default | single-binary control plane and host agent | VERIFIED |
| No mandatory SaaS dependency | no external calls; ACME only if configured | VERIFIED |
| No telemetry, no phone-home | no outbound calls except those the operator configures; console CSP `default-src 'self'` | VERIFIED (by construction and header tests; no traffic audit yet) |
| No fake runtime state | truth basis on every value; stale views marked | VERIFIED (console tested manually; see §9) |
| No browser-controlled truth | the console only calls signed, authorized APIs, never updates state optimistically, and hosts decide admission | VERIFIED |
| Hosts remain sovereign | host-local `policy.Admit`, pinned root, host journal | VERIFIED |
| Every consequential transition is observable | control-plane audit ledger and host journals, both hash-chained | VERIFIED |
| Cryptographic identity is first-class | Ed25519 and dh1 ids on every message | VERIFIED |
| Desired / admitted / observed distinct | separate fields end to end (API, CLI, console) | VERIFIED |
| Offline operation is expected | hold semantics, outbox, buffered observations | VERIFIED |
| Failure is part of the product model | 17 chaos scenarios | VERIFIED |
| Exportability is mandatory | `dh export` / `import` / `volume export` | IMPLEMENTED — LIMITED (no automated round-trip test) |
| Protocol over UI convenience | `dh/v1` spec and conformance suite | VERIFIED for the protocol primitives |
| UNKNOWN remains UNKNOWN | `fact()`/`unknown()` rendering; negative diagnostics | VERIFIED |

---

# 2. Evidence baseline

Results are **not** written into this document. They live in signed
validation records under `evidence/`, indexed in **`evidence/INDEX.md`**.
Both sit outside the source digest, so recording a result never changes the
identity of the tree it qualifies. Each record states its evidence id,
source digest, environment, scope, exclusions, outcome, and whether it
verifies (`dh evidence verify --dir …`). Schema and rules:
`docs/evidence/README.md`.

A gate result is current only when its record's source digest matches the
tree it is claimed for. The gate set run by every reference and PV1-S1
attempt (`validation/pv1/stage-parity.sh`):
- build, gofmt, vet;
- unit tests with `-race`;
- the gossip join repeated 20 times;
- conformance: drift check, Go in process, Go over stdio, and the
  independent Python implementation;
- all integration and regression tests;
- 17 chaos scenarios;
- the scripted TLS install runbook.

Still outside every record: browser automation (manual only), the
long-duration soak (not run), and public ACME (Pebble only).

---

# 3. System as built

```
operator ── dh CLI / web console (served by every member, no third parties)
                │  bearer capability (attenuable, expiring) over TLS
                ▼
control plane: 3 or 5 × dh-control
  Raft (mutual TLS, root-issued certs) · deterministic FSM · audit ledger + signed checkpoints
  scheduler (plan) → reconciler (signed assignments + per-assignment capabilities)
  views with truth basis · optional Postgres mirror (never the source of truth)
                │  signed bundles (stateIndex, rollback-protected)
                ▼
hosts: dh-noded  (each one sovereign)
  pinned root · local policy.yaml · ordered admission with hold semantics
  runtimes: process | docker · host journal (hash-chained, fsync) · outbox
  CAS (BLAKE3, FastCDC) · snapshots · anti-entropy · edge role (L7 + ACME)
  userspace WireGuard + SWIM gossip · peer API on mesh
                │  signed observations (monotonic seq), storage/repair/cert evidence
                ▼
federation: root-signed agreements between independent clusters
```

Binaries: `dh`, `dh-control`, `dh-noded`, `dh-conformance`, `dh-beacon` (sample
workload). Details: `docs/architecture.md`, `docs/trust-model.md`,
`docs/protocol/dh-v1.md`.

---

# 4. Milestones against their original exit criteria

## M1 — Sovereign runtime · **VERIFIED**

| Exit criterion | Status | Evidence |
|---|---|---|
| Two hosts | VERIFIED | `m1_test.go` (plus every other test) |
| Signed observations | VERIFIED | `m1_test.go`; chaos `replay-forgery` |
| Generation protection | VERIFIED | `sovereignty_test.go` (stale generation); chaos `stale-generation` |
| Replay protection | VERIFIED | `m1_test.go` (sequence numbers named); chaos `replay-forgery` |
| Revocation | VERIFIED | `m1_test.go`; chaos `revoked-host` |
| Freeze | VERIFIED | `m1_test.go`; manual console freeze/unfreeze |
| Durable journal | VERIFIED | fsync per entry; chaos `journal-corruption` including operator `ledger-seal` recovery |
| Verified audit | VERIFIED | `dh audit verify` (offline chain and checkpoints); `pkg/audit` tests |
| Operator projection | VERIFIED | `/api/v1/view`, CLI, console |

## M2 — Sovereign storage · **VERIFIED**, with limitations

| Exit criterion | Status | Evidence / gap |
|---|---|---|
| CAS | VERIFIED | BLAKE3, verified on read, corrupt objects quarantined |
| Chunking | VERIFIED | FastCDC 16/64/256 KiB; conformance vectors |
| Replicas | VERIFIED | snapshot committed at quorum 2 via signed `replica-evidence` |
| Integrity verification | VERIFIED | chaos `storage-replica-loss` (objects repaired with evidence) |
| Anti-entropy | VERIFIED | 256-bucket Merkle roots; repair evidence per object |
| Snapshot | VERIFIED | `m2_test.go`; chaos `interrupted-storage-write` (rollback to committed) |
| Node-failure recovery | VERIFIED | `m2_test.go` (restore after host loss, identical bytes) |
| Erasure coding | NOT IMPLEMENTED | `durability.erasure` accepts only `none` |
| Volume snapshot/restore CLI | NOT IMPLEMENTED | `dh volume snapshot` and `dh volume restore` do not exist. Snapshots are automatic, and restore happens through rescheduling. `dh volume export` exists. |
| Quota / disk-full behaviour | VERIFIED | chaos `disk-full` (injected `ENOSPC`; DEGRADED → HEALTHY) |

## M3 — Trust, capabilities, real mesh · **IMPLEMENTED — LIMITED**

| Exit criterion | Status | Evidence / gap |
|---|---|---|
| Multiple physical/virtual hosts | **LIMITED** | Many real host processes, each with its own identity, WireGuard device and data, but **all on one machine**. No cross-machine, NAT or WAN run (§12, P0-1). |
| WireGuard | VERIFIED | wireguard-go on netstack; signed `wg-binding`; `m3_test.go` |
| Gossip | VERIFIED | memberlist inside the tunnel; chaos `network-partition` (suspect → lost → heal) |
| Capability authorization | VERIFIED | chained capabilities checked by hosts; 29 conformance vectors |
| Sovereign policy | VERIFIED | `policy.Admit`, 18 ordered checks; `sovereignty_test.go` |
| Key rotation | VERIFIED | host key rotation with grace period, emergency key revocation, root rotation (`m3_test.go`) |

## M4 — Edge and TLS · **IMPLEMENTED — LIMITED**

| Exit criterion | Status | Evidence / gap |
|---|---|---|
| Public HTTPS | **LIMITED** | ACME HTTP-01, DNS-01 and wildcard issuance verified against **Pebble**, locally. Never exercised against a public CA or public DNS. |
| 3 replicas behind the edge | VERIFIED | `m4_test.go`, `m5_test.go` |
| Health-gated routing | VERIFIED | probes over the mesh; routing/draining/ejected/pending states |
| Edge failover | VERIFIED | `m4_test.go` (hung-replica ejection with 0 failures; edge failover) |
| TLS state observable | VERIFIED | certificate states REQUESTED … UNKNOWN in observations and console |
| HTTP/3 | NOT IMPLEMENTED | `UDP 443: NOT LISTENING` is reported |

## M5 — HA control plane · **VERIFIED**

| Exit criterion | Status | Evidence |
|---|---|---|
| Three members, Raft | VERIFIED | `m5_test.go`, `tls_test.go` |
| Leader failover | VERIFIED | 1.2–2.2 s measured; 0 failed requests (chaos `leader-crash`) |
| Snapshot and restore | VERIFIED | signed backup, offline verify, restore after total loss (`m5_test.go`) |
| No workload interruption on leader loss | VERIFIED | `m5_test.go`; chaos `interrupted-deployment` |
| Rollback protection | VERIFIED | `stateIndex`; `sovereignty_test.go` (bundle rollback refused) |
| Membership changes | VERIFIED | add-member, remove-member, transfer-leadership |

## M6 — Chaos and hardening · **IMPLEMENTED — LIMITED**

| Exit criterion | Status | Evidence / gap |
|---|---|---|
| Chaos suite | VERIFIED | 17 scenarios on disposable clusters, under traffic, with signed reports |
| Long-duration soak | **NOT RUN** | `dh chaos soak` exists but has not been run on the final build |
| Failure evidence | VERIFIED | per-scenario timelines and signed reports; routing snapshot at the first failure |
| No silent corruption | VERIFIED within scope | verify-on-read, journal break detection, audit chain |
| No unhandled panic in tested paths | LIMITED | no panic in the final integration run's logs. Chaos runs verify invariants, but per-process logs are not retained, so panics there are not independently checked. Not fuzzed (§12). |

Scenarios: leader crash, total control-plane outage, host crash, agent restart,
journal corruption, network partition, packet loss/duplication/reordering/delay,
clock skew, stale generation, replay/forgery, revoked host, storage replica
loss, disk full, OOM (Docker), interrupted deployment, interrupted storage
write, Postgres outage.

## M7 — Federation · **VERIFIED** (single-machine scope)

| Exit criterion | Status | Evidence |
|---|---|---|
| Two independent meshes | VERIFIED | two clusters with separate roots (`m7_test.go`) |
| Signed peering | VERIFIED | root-signed agreements; the grantor CA is carried for TLS |
| Cross-mesh placement | VERIFIED | delegated placements re-signed by the grantor |
| Local policy enforcement | VERIFIED | quota refusal, `allowFederated`, revocation stops the work |

## M8 — Protocol conformance · **IMPLEMENTED — LIMITED**

| Exit criterion | Status | Evidence / gap |
|---|---|---|
| Published specification | VERIFIED | `docs/protocol/dh-v1.md` |
| Conformance suite | VERIFIED | 136 vectors, adapter protocol, drift guard in CI tests |
| Independent implementation passes | VERIFIED **for §2–§8** | Python covers canonical JSON, identity, envelopes, audit, capabilities, chunking and Merkle roots. It does **not** cover manifests, admission decisions, bundle verification or mesh bindings. |
| Frozen protocol | **LIMITED** | §2–§8 are frozen by vectors (`SuiteVersion 1`). Message schemas in §9–§10 are specified but not yet pinned by vectors. |

The blueprint's conformance domains, one by one: **identity** VERIFIED ·
**audit** VERIFIED · **storage** VERIFIED · **assignment/observation signing**
VERIFIED (envelope level) · **manifest** NOT COVERED · **admission** NOT COVERED ·
**mesh** NOT COVERED.

---

# 5. Sovereignty, capabilities, keys and outages

**Host sovereignty.** A host admits an assignment only if every one of 18
ordered checks passes (spec §9.3):
- ledger, plane trust, assignment signature, audience, generation, clock;
- capability chain, revocation, freeze, freshness;
- tier, runtime, digest, attestation, federation;
- workload, CPU and memory caps.

For plane, clock, freeze, freshness and revocation failures, work already
admitted at the same generation is **held, not stopped**. **VERIFIED**
(`sovereignty_test.go` runs a host against a control plane holding a genuine
member key).

**Capabilities.** These are attenuable chains of Ed25519 blocks, in four uses:
- root → member delegation;
- member → per-assignment capability, bound to host, generation, digest and
  resource ceilings;
- operator sessions (`api.read` / `write` / `admin`, expiring);
- single-use join tokens.

The console has a capability lab signed by an ephemeral root that no host
trusts. **VERIFIED**.

**Key rotation.** Host keys rotate with dual signatures and a grace window,
single keys can be revoked in an emergency, and root rotation is signed by
the old root, with assignments re-signed. **VERIFIED**. Root key custody
(HSM or offline signer) is **HARDENING**.

**Control-plane outage semantics.** **VERIFIED** by chaos `cp-total-outage`
(854/854 requests OK):

| Aspect | Behaviour |
|---|---|
| Existing admitted workloads | continue |
| New admission | refused (`CONTROL_PLANE_STALE`); `offlineAdmission: stop` is opt-in per host |
| New generations | impossible (no signed bundle) |
| Observations | queued in the outbox and flushed later marked `buffered` |
| Recovery | automatic; rollback-protected by `stateIndex` |

**Offline-first matrix.** Every row is **VERIFIED**:

| Subsystem | Offline behaviour | Evidence |
|---|---|---|
| Host runtime | admitted work continues | chaos `cp-total-outage` |
| Host journal | local, fsynced, inspectable | `dh-noded status`; peer ledger API |
| Observations | queued locally | outbox flush in `cp-total-outage` |
| Control plane | desired-state changes unavailable without quorum | M5 |
| New admission | policy-dependent (`deny` / `stop`) | `sovereignty_test.go` |
| Storage | local reads verify; repair resumes on reconnection | M2, chaos |
| Mesh | peer-dependent; gossip marks suspect/dead | `network-partition` |
| UI | view age shown; unreachable or stalled views marked NOT CURRENT | console, manual |
| Audit | ledger verifiable offline from export or backup | `dh verify backup` / `export` |

---

# 6. Web console (14 screens) · **VERIFIED manually**

Overview · Runtime · Hosts · Workloads · Storage · Mesh · Edge · Policy · Audit ·
Diagnostics · Capabilities · Federation · Conformance · Settings. The console
also has:
- a command palette (⌘K or `/`);
- `g` + letter navigation;
- reduced motion;
- a mobile layout;
- a truth-basis badge on every value.

What "no fake state" means in practice, and was checked in a live browser:
- Desired, admitted and observed state are separate counters and never one
  health number. Drift is DERIVED.
- A revoked host shows `Identity: REVOKED / New admission: BLOCKED /
  Existing runtime: MAY CONTINUE`.
- Latency shows only where it was measured; elsewhere it says `NOT MEASURED`.
- Mesh topology lines are drawn only from host-reported handshakes, with no
  animation.
- Stale evidence is labelled. A hung or unreachable control plane turns the
  whole view into **LAST KNOWN STATE — NOT CURRENT**.
- Followers relay the leader's view. A former leader's stale cache cannot
  override newer replicated evidence.
- The Conformance screen runs the vectors on the serving member when asked,
  and shows the result only after the run.
- The M8 milestone is derived from an actual run.

**Gaps (HARDENING):**
- no automated browser test suite;
- no accessibility audit beyond contrast and keyboard checks;
- over TLS, browsers must trust the cluster root CA.

---

# 7. CLI · **VERIFIED**, with two commands missing

| Blueprint command family | Status |
|---|---|
| `init`, `node invite/join/approve/drain/revoke/revoke-key` | present |
| `apply`, `get apps/nodes/volumes`, `describe`, `explain`, `scale`, `rollout status`, `delete` | present |
| `logs app`, `exec app` (policy-gated) | present |
| `mesh status/peers/ping/doctor` | present |
| `cp bootstrap/add-member/remove-member/status/backup/restore/snapshot/transfer-leadership/rotate-root` | present |
| `audit tail/verify/host`, `verify backup/export` | present |
| `export`, `import`, `volume ls`, `volume export` | present, **no automated round-trip test** |
| `volume snapshot`, `volume restore` | **NOT IMPLEMENTED** |
| `federation root/grant/accept/ls/revoke/withdraw`, `chaos list/run/soak`, `console`, `dev up/down/status` | present |

The CLI works without the browser. `-h` works without a configured cluster.

---

# 8. Engine subsystems

| Subsystem | Status | Notes |
|---|---|---|
| Manifest (`dh/v1`) | VERIFIED | strict YAML (unknown fields rejected), normalized and hashed. **Gap:** `metadata.owner` is a free string, and `did:dh` identities are not validated. |
| Scheduler | VERIFIED | deterministic plan; filters for tier, resources, arch, features, anti-affinity and failure domain; stability-preserving; `dh explain`. |
| Reconciler | VERIFIED | plan → signed assignments; rolling budget; artifact-ready gating; volume restore hints; re-signing after root rotation. |
| Observation model | VERIFIED | host-signed, monotonic seq, replay-rejected, buffered offline; leader cache plus periodic commit. |
| Database and journal | VERIFIED, **by design different from the original** | Raft is the source of truth (ADR 0004). Postgres is an optional mirror, and its outage degrades durability reporting to `JOURNAL ONLY` (chaos `postgres-outage`). |
| Runtime isolation | LIMITED | `process` (**no CPU/memory enforcement**; admission details say so) and `docker` (enforced `--memory`/`--cpus`, OOM detected). containerd, gVisor and Firecracker are **NOT IMPLEMENTED** (gVisor and KVM are detected and reported). |
| Transport security | VERIFIED | TLS from first start; fingerprint-pinned bootstrap; root-issued member certs; hosts on HTTPS; mutual-TLS Raft; WireGuard between hosts; federation verifies the grantor CA (`tls_test.go`, M7). `dh init` defaults to TLS. |
| Export and exit | LIMITED | `dh export` is signed and secret-free. It contains manifests, rosters, trust configuration, host records and policies, volume and snapshot metadata, artifact references, audit and checkpoints, and federation. Secrets are excluded and listed. Snapshot *data* leaves through `dh volume export`. **Gap:** no automated export → import → equivalence test. |
| Truthfulness engine | VERIFIED | OBSERVED / DERIVED / CONFIGURED / PLANNED / UNKNOWN on every console value. |

---

# 9. Security evidence

| Required test (original §36) | Status | Evidence |
|---|---|---|
| Signature tampering | VERIFIED | M1; chaos `replay-forgery`; conformance `verify/tampered-payload` |
| Wrong key | VERIFIED | M1; conformance `key-substitution`, `key-not-allowed` |
| Replay | VERIFIED | M1; chaos `replay-forgery` |
| Generation rollback | VERIFIED | `sovereignty_test.go`; chaos `stale-generation` |
| Revocation | VERIFIED | M1, M3; chaos `revoked-host` |
| Forged shell / command | VERIFIED | exec is policy-gated (`allowExec`); commands are argv, never shell; unsigned or forged assignments refused |
| Audit tampering | VERIFIED | `pkg/audit` (every break reason); 14 conformance vectors; offline `dh audit verify` |
| Compromised control plane (valid member key) | VERIFIED | `sovereignty_test.go` |
| Canonicalization attacks | VERIFIED | 42 canon vectors (duplicate keys, lone surrogates, invalid UTF-8, floats, BOM …) |
| Transport interception | VERIFIED | `tls_test.go` (plain HTTP refused; certificates verify against the root; bootstrap pinned) |
| Fuzzing | NOT DONE | HARDENING |
| External security review | NOT DONE | HARDENING |

Findings fixed during this cycle, which a production review should re-check:
- TLS existed but was never enabled end to end.
- Invalid UTF-8 and lone surrogates were silently repaired by the Go decoder.
- Credentials were published without a lock, racing with the TLS handshake.
- A host that lost a join-token race could not re-join.
- Corrupt host ledgers had no auditable recovery path; `ledger-seal` now
  provides one.
- Found by PV1-S1-A02 and reproduced on macOS, each with a regression test:
  - **D1:** an edge host refused its own connections to a replica it ran itself.
  - **D2:** rolling updates dropped requests (stop-before-start). The fix is
    make-before-break for stateless replicas, plus an edge fallback to live
    draining endpoints.

---

# 10. Performance · measured values only

The following values were measured on the development machine (darwin/amd64,
loopback, Go 1.26.4). They are **indicative, not benchmarks**: the sample
sizes are small and no dedicated hardware was used.

| Measurement | Result | Method |
|---|---|---|
| Control-plane failover | 1.2–2.2 s to a new leader | M5 test and chaos `leader-crash` (several runs) |
| Requests failed during leader loss | 0 of 204–240 | edge traffic every 20 ms during failover |
| Userspace WireGuard throughput | ~73 MiB/s | `pkg/mesh` benchmark, loopback |
| Mesh RTT | ~1–2 ms | host-measured HTTP round trips over WireGuard |
| Artifact fetch (9 MiB, 125 chunks) | ~45 ms | host journal `artifact-verified` timing |
| Anti-entropy repair (26–28 objects) | ~3.7–4.0 s | chaos `storage-replica-loss` |
| Conformance run (136 vectors) | ~60 ms in process | `dh-conformance` |

Everything else is **NOT MEASURED**: scheduling rate, storage IOPS,
replication throughput, CPU and memory overhead of the agent, edge
requests/s, and any multi-machine latency. Formal measurements must record
hardware, version, configuration, sample size, method and timestamp (§12, P0-4).

---

# 11. Invariants, release gates and Definition of Done

## Master invariants

| # | Invariant | Status | Evidence |
|---|---|---|---|
| 1 | Control-plane compromise ≠ host compromise | VERIFIED | `sovereignty_test.go` |
| 2 | A host can reject a control-plane instruction | VERIFIED | 18 refusal codes; `describe app` shows the checks |
| 3 | Desired ≠ observed | VERIFIED | separate fields in the API, CLI and console |
| 4 | Revocation blocks new work without necessarily killing existing work | VERIFIED | M1, chaos `revoked-host` |
| 5 | A frozen plane does not imply workload failure | VERIFIED | M1, console freeze test |
| 6 | Replay and stale generations cannot move state backward | VERIFIED | M1, `stale-generation`, `stateIndex` |
| 7 | Audit tampering is detectable offline | VERIFIED | `dh audit verify`, `dh verify backup/export` |
| 8 | Browser state cannot create infrastructure truth | VERIFIED | no optimistic updates; every write is an authorized API call that hosts may still refuse |
| 9 | Unknown measurements remain unknown | VERIFIED | `unknown()` rendering, `NOT MEASURED`, negative diagnostics |
| 10 | The complete system remains exportable | LIMITED | export exists; no automated round trip |

## Release gates

| Gate | Status |
|---|---|
| A Compile · B Vet · C Tests · D Race | PASS |
| E Security (adversarial tests) | PASS (fuzzing and external review outstanding) |
| F Integration (real runtime, database and network paths) | PASS, single machine |
| G Browser | PASS, **manual** |
| H Evidence (every claim has evidence) | PASS for this document's claims |
| I Documentation matches implementation | PASS: install runbook executed end to end over TLS; operations runbook commands checked |
| J Truthfulness | PASS: limitations are published here and in the README |

## Definition of Done

```text
[x] M1 runtime remains green                [x] Edge routing is real
[x] Host sovereignty is preserved           [x] TLS state is observable
[x] Desired/admitted/observed separate      [x] HA control plane is real
[x] Signatures verified                     [x] Failover is tested
[x] Replay protection verified              [x] Chaos scenarios are executable
[x] Generation protection verified          [x] Federation boundaries are enforced
[x] Revocation verified                     [x] Protocol conformance exists
[x] Audit integrity verified                [x] CLI works independently
[x] Storage integrity verified              [x] Browser reflects real state (manual)
[x] Replica recovery verified               [x] Offline behavior is explicit
[~] Real mesh transport exists              [~] Export/restore works (no round-trip test)
    (real WireGuard; single machine only)   [x] Documentation matches implementation
[x] Capability authorization enforced       [x] No fake metrics / state / certifications
[x] Key rotation works                      [x] No hidden telemetry
                                            [x] No mandatory vendor dependency
                                            [~] Security evidence reproducible (yes; no external review)
```

**Project status:**

> Decentralized.Host M1–M8 is functionally implemented and verified in a
> single-machine, multi-process environment. Production readiness has not
> yet been demonstrated across real multi-machine infrastructure.

**Not yet a production release.** Production requires the Production
Validation Release in §12.

---

# 12. Production Validation Release

The next phase is **not M9**, and it is not feature expansion. It proves
that the architecture already built survives the environments
Decentralized.Host claims to orchestrate:
- real Linux machines in independent failure domains, over WAN and through NAT;
- long-running operation;
- public PKI;
- upgrades, backup and restore;
- resource exhaustion and adversarial testing.

**Release progression:**

```text
M1–M8 implementation ──► Production Validation ──► Release Candidate ──► externally reproducible evidence ──► Production Release
        (done)                  (this phase)
```

**Governing rule (Production Candidate gate):**

> No capability moves from *Implemented — limited* to *Verified* merely
> because its code exists. Verification requires reproducible evidence from
> the environment the claim implies.

The rule applies the project's own state model to its release process:

```text
desired capability ≠ implemented capability ≠ observed capability ≠ verified capability
```

In practice:
- A row changes label only in the same change that adds its evidence: a
  test, a signed chaos or soak report, or a published measurement with its
  method.
- Limitations are never removed from this document to make a release look
  complete. They are moved only when evidence retires them.
- Each validation run must be reproducible by someone else from the
  published harness, configuration and signed reports. That is the
  "externally reproducible evidence" stage.

This phase is expected to find defects, not just confirm results. The last
cycle showed why: running the console and runbooks against live clusters
exposed four problems that feature testing had missed:
- the TLS bootstrap gap;
- stale follower views;
- no recovery path for corrupt ledgers;
- the mesh-forwarder lifecycle bug.

## PV-1 — Real Infrastructure Validation (first promotion gate)

> Reproduce the existing verified M1–M8 behaviours on independent Linux
> machines, and keep reproducible evidence for every promoted claim.

PV-1 changes no semantics and adds no capability. Its purpose is to find
assumptions that stayed invisible while everything shared one macOS kernel
and a loopback network. The stages run in order, and each must pass before
the next starts. WAN comes last.

**Environments.** The development Mac is the **reference environment only**:
builds, race tests, integration, chaos, conformance and development of the
evidence tooling run there, and it promotes nothing. Linux containers on the
Mac (Colima) are **diagnostic only** after attempt A01 was lost to the VM and
A02 ran on 2 starved vCPUs. PV-1 verification runs on real Linux
infrastructure.

| Stage | Environment | What it establishes | Promotes |
|---|---|---|---|
| **PV1-S1** Linux parity | one independent Linux VM or machine (Ubuntu or Debian, ≥ 4 vCPU, ≥ 8 GB, ≥ 40 GB SSD) | moving from macOS to Linux changes no protocol or runtime behaviour: build, race tests, conformance (Go and Python), all integration tests, 17 chaos scenarios, scripted TLS install runbook, repeated gossip join | "runs on Linux", nothing multi-machine |
| **PV1-S2** LAN multi-machine | 3 separate Linux VMs or machines on one LAN. Each is a host, and control-plane members are spread across them. | identity, TLS bootstrap, WireGuard, gossip, reconciliation, storage replication, edge routing, freeze and recovery, revocation, leader failover, audit, all across real machine boundaries | M3 "multiple hosts"; M5 failover across machines |
| **PV1-S3** Partition and failure | same cluster | machine power-off, NIC down, real network partitions (firewall rules, not the loopback fault proxy), disk-full on a real filesystem, latency, loss, reordering | M6 scenarios on real infrastructure |
| **PV1-S4** WAN and NAT | machines in ≥ 2 networks, at least one behind NAT, real latency | mesh endpoints and keepalives through NAT, edge over WAN, failover under WAN latency | the WAN claims in M3, M4 and M7 |

**Current state:** kept outside this document, in `evidence/INDEX.md` and
`evidence/PV1-S1-HISTORY.md`. That covers every attempt (including failed and
infrastructure-failed ones), the defects each one found, and their regression
tests. The source digest lets reference and Linux runs prove they tested the
same tree without claiming the same environment.

**Evidence record** (produced by `dh evidence run` / `seal` and checked by
`dh evidence verify`). The record contains:
- the environment of every machine: OS, kernel, distribution, architecture,
  CPU model and count, RAM, the filesystem of the data directory, disk,
  network addresses, and whether it is a container or VM;
- the exact source: a tree digest (`dh evidence digest`), plus the commit
  once the tree is under version control;
- digests of the binaries under test;
- every command, with its start and end time, exit code, attempt count
  (retries are recorded, never hidden) and log hash;
- chaos reports, and the claim and **scope** in plain words;
- a signature (kind `validation-record`).

**A passing result that someone else cannot reproduce from its record does
not promote anything.**

**Regression rule.** Every PV-1 failure becomes an automated regression test
(integration test, chaos scenario or unit test) in the fix, before the next
stage starts.

Scripts and procedure: `validation/pv1/README.md`. The scripts are:
- `stage-parity.sh`: PV1-S1, authoritative on a real Linux machine;
- `docker-parity.sh`: diagnostic only;
- `tls-install.sh`: the install runbook, scripted.

**After PV-1**, in order:
1. 24-hour soak
2. public ACME validation
3. performance baseline
4. export/import disaster recovery
5. runtime resource enforcement
6. browser automation
7. security, fuzzing and root-key hardening
8. signed releases and rolling upgrades

These are P0-2 … P0-9 below.

## P0 — Production Validation (required for Release Candidate)

| ID | Work | Exit criterion |
|---|---|---|
| P0-1 | **Multi-machine and Linux validation** | M1–M7 integration tests and all 17 chaos scenarios pass on ≥ 5 Linux machines across ≥ 2 networks, including NAT between hosts, with real latency. `proc_linux.go` paths exercised. |
| P0-2 | **Long-duration soak** | `dh chaos soak` for ≥ 24 h under traffic on the multi-machine cluster: zero invariant violations, bounded memory and file descriptors, signed report. |
| P0-3 | **Public ACME** | Let's Encrypt **staging**, then production, on a public domain: HTTP-01, DNS-01 with a real DNS provider, and renewal observed through RENEWING → ISSUED. |
| P0-4 | **Measured performance** | A benchmark harness recording hardware, version, configuration, sample size, method and timestamp for the §10 metrics. Unmeasured values stay `NOT MEASURED`. |
| P0-5 | **Export and restore round trip** | Automated test: export → fresh cluster → import → `volume export` → byte equality and audit continuity. Plus `dh volume snapshot` / `dh volume restore`. |
| P0-6 | **Process-runtime enforcement** | cgroup v2 limits for the process runtime on Linux, or host policy that refuses `process` with resource limits it cannot enforce. |
| P0-7 | **Automated browser tests** | Headless tests for all 14 screens: truth-basis presence, the stale view, freeze and scale flows, and accessibility (axe) checks. |
| P0-8 | **Security hardening** | Fuzz canonical JSON, envelope, capability and manifest parsers. External review of the trust model and TLS bootstrap. Root key custody (offline signer or HSM). |
| P0-9 | **Packaging and upgrades** | Signed release artifacts, systemd units, and rolling upgrades of members and hosts across versions, with a protocol-version check between mixed versions. |

## P1 — planned capability

| ID | Work |
|---|---|
| P1-1 | containerd runtime; gVisor and Firecracker runtimes behind the existing runtime interface |
| P1-2 | Erasure-coded volumes (`durability.erasure`) |
| P1-3 | HTTP/3 at the edge |
| P1-4 | Conformance breadth: vectors for manifest normalization, admission decisions, bundle verification and `wg-binding`; the independent implementation extended to them; §9–§10 frozen |
| P1-5 | `did:dh` owner identities, validated in manifests |
| P1-6 | Opt-in, self-hosted metrics export (never telemetry) |
| P1-7 | Console TLS trust UX (for example ACME-issued console names), plus finer-grained operator roles beyond read/write/admin |

## Known design decisions that differ from the original blueprint

- **Raft, not Postgres, is the source of truth.** Postgres is an optional
  evidence mirror (ADR 0004).
- **Mesh is userspace WireGuard.** Workloads are reached through
  per-assignment forwarders, not routed per-workload IPs (ADR 0005).
- **The console has no build step and no framework** (ADR 0007).
- **TLS by default**, with fingerprint-pinned bootstrap (ADR 0008).

---

# 13. The standard (unchanged)

Do not rewrite M1 to build M2. Do not call a single control plane HA. Do not
call a simulated failure chaos. Do not call a browser visualization mesh
state. Do not call planned capabilities implemented. Do not publish numbers
that were not measured.

This revision applies that standard to itself. Everything marked VERIFIED
has a named test or scenario. Everything else is named as limited, absent or
planned.
