# Gap matrix — repository vs Production Master Build Prompt v1

One row per spec area. **Status** is what exists today (see
`docs/CURRENT_STATE.md`). **Phase** is the spec's phase. A row is only
COMPLETE when its backend acceptance test passes; UI presence never counts.

| § | Spec area | Phase | Status | Gap / next step |
|---|---|---|---|---|
| 2.1–2.2 | Local-first, no wallet | P0 | **MET** | Nothing in the platform or console needs a token or wallet. |
| 2.4–2.6 | Reality before UI, fail closed, no mock fallback | P0 | **MET (console)** | RC1: normalized truth model, no fallback, disconnect test PASS. |
| 3 | Truth states | P0 | **MET** | Control plane has `basis`; console adds LIVE/DERIVED/CONFIGURED/UNAVAILABLE/PLANNED/SIMULATED/UNKNOWN. Spec freshness `EXPIRED`/`UNREACHABLE` map to `lost`/backend-unreachable. |
| 4–5 | Service split, monorepo layout | — | **DIVERGES** | Single Go module by design (ADR 0001). Not recommended to restructure without a reason beyond the spec. |
| 6 | First-party protocol family (DHP-*) | P0–P2 | **PARTIAL** | `dh/v1` covers identity, runtime, storage, evidence (`docs/protocol/dh-v1.md`). Discovery, market and settlement protocols do not exist. |
| 7 | Identity + enrolment | P0 | **MET** | NODE-A01 gate (`tests/integration/node_a01_test.go`): key generated on the host (0600) and absent from every control-plane file, API document and log; single-use, 15-minute default, owner-revocable invites; forged, expired, reused and revoked tokens refused; replayed enrolment refused. The challenge is the root-signed single-use join nonce the host signs over (no separate server nonce; `docs/protocol/dh-v1.md` §9.1). Spec states CHALLENGE_SENT / CAPABILITY_PROBED are not distinct records. |
| 8 | Hardware discovery | P0 | **PARTIAL** | Measured and signed: memory, CPU model/cores, swap, disks, GPUs (sysfs, `nvidia-smi`), data filesystem, uptime; unmeasurable values listed in `facts.unknown`. Fixed a defect: `facts.memBytes` used to be the declared `--mem`. Missing: NICs, NAT type / public reachability (needs an outside observer), container runtimes other than Docker, virtualization beyond the `/dev/kvm` probe. |
| 9 | Resource model (capacity/reservation/allocation) | P0–P1 | **PARTIAL** | Declared capacity and policy caps exist; reservations/allocations as first-class records do not. |
| 10–11 | Contribution policy, owner kill switch | P0–P2 | **PARTIAL** | Kill-switch equivalents: `freeze`, `drain`, host `policy.yaml` (refuse tiers / federated work). Missing: schedules, per-origin limits, one-switch "stop external". |
| 12 | Node lifecycle state machine | P0 | **PARTIAL** | pending / ready / draining / revoked / lost exist and are validated server-side; CORDONED, IDLE, DEGRADED are not distinct states. |
| 13 | Workload origin | P1–P2 | **PARTIAL** | Owner vs federated is recorded; community / marketplace / external do not exist. |
| 14 | Workload security profiles | P0 | **GAP** | Docker runtime only; no seccomp/AppArmor profile management, no rootless/gVisor/Firecracker (P1-1). Process runtime unenforced (P0-6). **Security-critical before any untrusted workload.** |
| 15 | Portable deployment spec | P0 | **MET (dh/v1)** | Spec's `decentralized.host.yaml` shape differs; dh/v1 is the implemented contract. |
| 16 | Universal deploy (Git, Dockerfile, upload) | P0 | **GAP** | Only prebuilt artifacts (CAS) and digest-pinned OCI images. No build service. |
| 17 | Immutable artifacts + provenance | P0 | **PARTIAL** | Digests + root attestation. Missing: SBOM, builder identity, source commit binding. |
| 18 | Deployment modes | P0–P4 | **PARTIAL** | SELF_HOST, PRIVATE_MESH, EDGE, federated. COMMUNITY/MARKETPLACE/EXTERNAL_DEPIN/HYBRID missing. |
| 19–20 | Placement engine, failure domains | P1 | **MET (single operator)** | Plans with reasons; spread by failure domain. Operator independence not modelled (all hosts share one root). |
| 21–25 | Marketplace, bids, leases, metering, settlement | P2–P3 | **MISSING** | Start with metering (runtime-sourced usage records, signed) — prerequisite for everything else. |
| 26–28 | Web3, contracts, staking/slashing | P3 | **NOT STARTED (by design)** | Deferred until P2 works. |
| 29–30 | ZK, TEE | P6 | **RESEARCH** | UI must show PLANNED; the console does not claim either. |
| 31–32 | Private mesh, NAT/CGNAT | P0–P1 | **LIMITED** | WireGuard mesh works; no hole-punching/relay; no cross-NAT evidence (P0-1). |
| 33 | Edge | P1 | **LIMITED** | TLS termination, health routing, draining. No caching, rate limiting, HTTP/3. |
| 34–35 | Distributed storage | P1–P2 | **PARTIAL** | Replicated CAS volumes with repair. No object API, encryption at rest, erasure coding. |
| 36 | GPU network | P2 | **MISSING** | Needs GPU discovery (§8) first. |
| 37 | Decentralized CDN | P2 | **MISSING** | Requires independent edge operators. |
| 38 | DNS / naming | P1 | **PARTIAL** | Ingress names + ACME DNS-01. No DNS observation; no private mesh naming. |
| 39 | Secrets | P0 | **GAP** | Manifest env values are stored in control-plane state. **Needs a secrets subsystem before production secrets.** |
| 40 | Observability | P1 | **PARTIAL** | Point-in-time facts and edge counters; no time series (P1-6). BFF has in-process metrics. |
| 41–43 | Evidence ledger, chain, execution evidence | P0 | **MET** | Hash-chained audit + signed validation records. Execution evidence binding usage is missing (no metering). |
| 44–45 | Reputation, decentralization metrics | P2 | **PARTIAL (console)** | Console shows dimensions (distinct hosts, largest share, operators = 1), never a score. |
| 46–49 | External DePIN adapters | P4 | **MISSING** | Console shows every adapter as UNAVAILABLE; no logos claim integration. |
| 50–56 | Onramp, gasless, recovery, DID, KYC, GDPR, green | P3+ | **MISSING / P1-5 for DID** | — |
| 57–60 | Bare metal, DeWi, IoT, federated AI | P6 | **OUT OF SCOPE** | Separate verticals. |
| 61 | API | P0 | **PARTIAL** | Operator API exists with different paths (`/api/v1/view`, `/apply`, `/nodes/{id}/drain`…). Marketplace/usage/adapter APIs missing. |
| 62 | Authorization | P0 | **MET (coarse)** | Capability actions `api.read`/`api.write`/`api.admin`, attenuable. Finer roles are P1-7. |
| 63 | Audit | P0 | **MET** | Every mutation audited with actor from the capability. `before/after` diff fields are not recorded. |
| 64 | Operations | P0 | **PARTIAL** | Mutations are synchronous Raft commits; long-running progress is observed from state (deployments), not an Operation resource. |
| 65 | Durable events | P0 | **MET** | The audit ledger is the event log (replayable, sequenced). |
| 66 | PostgreSQL as source of truth | — | **DIVERGES (by design)** | Raft is authoritative; Postgres is an optional mirror (ADR 0004). |
| 67 | CLI | P0 | **MET** | `dh` covers node, deploy, logs, scale, audit, evidence, mesh, chaos, federation. Missing: marketplace commands. |
| 68–70 | Agent, command security, offline-first | P0 | **MET** | Signed, audience-bound, generation-checked assignments; hosts keep running work when the control plane is gone (verified again by the RC1 disconnect test: edge kept serving). |
| 71 / 107 | Control-plane failure test | P0 | **MET (console)** | `command-centre/tests/e2e/disconnect.mjs`: 12/12 checks. Platform-side covered by chaos `leader-crash`, `cp-total-outage`. |
| 72–74 | Provider/developer UX, console IA | P0–P2 | **PARTIAL** | Real routing and deep links for all P0 surfaces. Provider/marketplace routes intentionally absent. |
| 84–85 | Security baseline, supply chain | P0 | **PARTIAL** | TLS, CSRF, headers, validation, RBAC, audit, key rotation exist. Missing: rate limiting in the BFF, SBOM, signed releases (P0-9), dependency/image scanning in CI. |
| 88 | No-mock production gate | P0 | **MET (console)** | `npm run build` fails on forbidden patterns. |
| 91–94 | Chaos / security / marketplace / storage tests | P0–P2 | **PARTIAL** | 17 chaos scenarios; marketplace tests impossible until P2. |
| 96 | **P0 acceptance** (one Linux machine end to end) | P0 | **PARTIAL** | install→enroll→approve→deploy→serve→observe→logs→restart→cordon/drain→resume pass on the dev cluster. Missing: hardware discovery breadth, resource reservation, rollback command, Git deploy, secrets. |
| 97 | P1 acceptance (3 independent nodes) | P1 | **LIMITED** | 3+ hosts with failover verified on one machine; needs P0-1 multi-machine run. |
| 98–102 | P2–P6 | — | **NOT STARTED** | In order; do not start P3 before P2 acceptance. |

## Checkpoint NODE-A01 (sovereign node foundation)

Gate: `go test ./tests/integration -run TestNodeA01` plus
`go test ./pkg/node -run 'TestCompromised|Hardware'`. It covers local key
generation and custody, enrolment refusals (reused, forged, expired, revoked
token; signer mismatch; replay), measured facts vs declared capacity, owner
approval, authenticated heartbeats (replay, foreign key), control-plane
disconnect (host stops claiming freshness, then reconciles), host disconnect
(STALE then lost, never FRESH), audit and host-ledger tamper detection, and a
revoked key refused on the heartbeat and bundle channels. On the host side,
replayed/rolled-back bundles, wrong signer, wrong target, expired capability,
stale generation and revocation are refused. Out of scope, as proposed:
marketplace, leases, settlement, tokens, external DePIN, ZK, federation of
coordinators. First attempt: `evidence/NODE-A01-A01`, **FAIL** (VERIFIED).
All NODE-A01 checks passed; the record fails on a pre-existing gossip-join
defect (`docs/qualification/NODE-A01.md`). Not covered by NODE-A01: a distinct CORDONED state and
owner-reserved capacity as a first-class record (§9–12).

## Recommended next steps (in order)

1. Secrets subsystem (§39) and runtime isolation profiles (§14): both block any non-owner workload.
2. Hardware discovery breadth (§8) incl. GPU, and resource reservations (§9).
3. Git/Dockerfile build service producing attested artifacts with SBOM (§16–17).
4. PV1 multi-machine validation across NAT (P0-1) — already specified in `validation/pv1`.
5. Runtime-sourced, signed usage metering (§24) — the foundation of any marketplace.
6. Decide ADR 0009 (Command Centre vs ADR 0007).
