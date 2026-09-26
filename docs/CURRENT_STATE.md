# Current state (2026-09-26)

Snapshot of what the repository implements, measured against the Production
Master Build Prompt v1. Evidence status for platform milestones is taken from
`docs/BLUEPRINT.md` and `evidence/INDEX.md`, which remain authoritative; this
file does not promote anything.

Legend: **LIVE** implemented and exercised by tests · **LIMITED** implemented,
narrower than the spec or only single-machine evidence · **PARTIAL** some of it
exists · **MISSING** · **SIMULATED** · **SECURITY-SENSITIVE**.

## Implemented

| Area | State | Where | Evidence |
|---|---|---|---|
| Node identity (Ed25519, generated on the host) | LIVE | `pkg/identity`, `dh-noded` | `m1_test.go`, `m3_test.go` |
| Enrolment: single-use invite → join → approval | LIVE | `dh node invite/invite-revoke/approve`, `pkg/control/hostapi.go` | M1/M3 tests, NODE-A01 gate |
| Signed assignments, per-assignment capabilities | LIVE | `pkg/control/reconcile.go`, `pkg/capability` | 29 conformance vectors |
| Host sovereignty: local `policy.yaml`, 18 ordered admission checks, hold semantics | LIVE | `pkg/policy`, `pkg/node` | `sovereignty_test.go` |
| Signed observations with freshness (FRESH / STALE / lost) | LIVE | `pkg/control/views.go` | M1, chaos |
| Hardware facts (measured, signed) | PARTIAL | `api.Facts`, `pkg/node/hw.go` | memory, CPU model/cores, swap, disks, GPUs, data filesystem, uptime, runtimes, probes; `facts.unknown` names what was not measured. **No NICs, no NAT type.** |
| Deployment spec | LIVE (as `dh/v1` manifest) | `pkg/manifest` | digest pinning required (`b3:` / `sha256:`) |
| Runtimes | LIMITED | `pkg/runtime` | `process` (no enforcement) and `docker`; **`sandbox`** (RUNTIME-P0-A01): namespaces, seccomp, no capabilities, read-only root, cgroup limits, for PRIVATE/RESTRICTED. UNTRUSTED refused (no microVM/gVisor). |
| Scheduler / placement with failure-domain spread | LIVE | `pkg/scheduler` | plans with reasons in `view.apps[].plan` |
| Content-addressed artifacts + root attestation | LIVE | `pkg/storage`, `dh artifact push/sign` | M2 |
| Replicated volumes, snapshots at quorum 2, repair | LIVE | `pkg/storage` | M2 (erasure coding MISSING) |
| Userspace WireGuard mesh + SWIM gossip | LIMITED | `pkg/mesh` | single machine; no cross-NAT evidence (P0-1) |
| Edge: health-gated L7, draining, ejection | LIVE | `pkg/edge` | M4 |
| TLS: ACME HTTP-01/DNS-01/wildcard, local CA | LIMITED | `pkg/edge/certs.go` | Pebble only, no public CA (P0-3) |
| HA control plane (Raft, mTLS), backups, restore | LIVE | `pkg/control` | M5 |
| Audit ledger (hash chain + signed checkpoints) | LIVE | `pkg/audit` | verified in every view |
| Federation between clusters | LIVE (single machine) | `pkg/control/federation.go` | M7 |
| Chaos (17 scenarios) + soak | LIMITED | `pkg/chaos` | 17/17 on the macOS reference |
| Validation evidence records | LIVE | `pkg/evidence`, `evidence/` | REF-MAC-A03 PASS/VERIFIED |
| Operator CLI | LIVE | `pkg/cli` | `dh help` |
| Embedded console (no build) | LIVE | `web/dist` | ADR 0007 |
| Command Centre (React + BFF) | LIVE against the control plane | `command-centre/` | `command-centre/docs/qualification/RC1.md` |

## Simulated

| Item | Where | Status |
|---|---|---|
| Demo replay adapter | `command-centre/src/server/adapters/demo.ts` | Opt-in only (`PLATFORM_ADAPTER=demo`), refused in production, labelled SIMULATED, mutations refused. |
| Prior synthetic store (hard-coded nodes, metrics, invoices, evidence) | removed | Deleted in RC1; see the qualification report. |

## Missing

Marketplace (offers, bids, leases), usage metering, settlement, provider
console, external DePIN adapters, GPU discovery and scheduling, contribution
policy per workload origin (beyond federation grants), secrets subsystem, Git
build service, SBOM / signed releases (P0-9), notifications, mobile,
visitor analytics, time-series metrics (P1-6), DID identities (P1-5),
ZK / TEE (research).

## Workload isolation (RUNTIME-P0-A01)

`pkg/runtime/sandbox` runs verified artifacts under PRIVATE and RESTRICTED
profiles (ADR 0010): user/mount/pid/ipc/uts (+network for RESTRICTED)
namespaces, in-house seccomp (amd64/arm64), empty capability set with
no_new_privs, pivoted read-only root, cgroup memory/PID/CPU limits, and a
setns port relay for RESTRICTED. UNTRUSTED is refused. Admission fails
closed: isolation on a host that cannot sandbox is denied. Evidence:
CC/P0 gate `RUNTIME-P0-A01`.

## Broken / known defects

- Fixed after NODE-A01-A01: gossip joins over the userspace mesh could stall past memberlist's 10 s TCP timeout. The cause was crossed first handshakes: the lower-key side's startup initiation raced the other side's data-triggered initiation, wireguard-go consumed both concurrently, and one direction dropped every packet until the 15 s rekey. Now only the lower-key side starts a pair's first handshake; the other side gets the endpoint after `firstContactGrace` (2 s), or learns it from the handshake (`pkg/mesh/device.go`, `pkg/mesh/handshake_test.go`).
- Fixed in NODE-A01: `facts.memBytes` reported the declared `--mem` capacity as if measured, and the Command Centre labelled it "Memory (measured)". Hosts now measure it; the console treats memory from agents that do not send `facts.unknown` as not measured.
- Fixed in NODE-A01: a validly signed older `enroll` envelope replayed for a known host overwrote its newer enrolment (facts summary and policy).

- `dh-src-digest/2` does not include `.ts`/`.tsx`, so validation records cannot bind the Command Centre source on their own (the RC1 record adds a hash manifest as a workaround).
- ADR 0007 (no-build console) conflicts with the Command Centre; decision pending in ADR 0009.
- Process runtime does not enforce resource limits (documented; P0-6).

## Security-sensitive

Root key custody (P0-8), capability issuance (`dh token`), TLS bootstrap
pinning (ADR 0008), host admission policy, `POST /nodes/{id}/exec` (off by
default via `allowExec: false`), backup/restore, federation agreements, the
Command Centre session cookie and CSRF guard.
