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
| Enrolment: single-use invite → join → approval | LIVE | `dh node invite/approve`, `pkg/control/hostapi.go` | M1/M3 tests |
| Signed assignments, per-assignment capabilities | LIVE | `pkg/control/reconcile.go`, `pkg/capability` | 29 conformance vectors |
| Host sovereignty: local `policy.yaml`, 18 ordered admission checks, hold semantics | LIVE | `pkg/policy`, `pkg/node` | `sovereignty_test.go` |
| Signed observations with freshness (FRESH / STALE / lost) | LIVE | `pkg/control/views.go` | M1, chaos |
| Hardware facts | PARTIAL | `api.Facts` | OS, kernel, arch, CPUs, memory, runtimes, probes; **no GPU, disks, NAT type** |
| Deployment spec | LIVE (as `dh/v1` manifest) | `pkg/manifest` | digest pinning required (`b3:` / `sha256:`) |
| Runtimes | LIMITED | `pkg/runtime` | `process` (no CPU/mem enforcement) and `docker`; gVisor / Firecracker detected, not wired |
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

## Broken / known defects

- `dh-src-digest/2` does not include `.ts`/`.tsx`, so validation records cannot bind the Command Centre source on their own (the RC1 record adds a hash manifest as a workaround).
- ADR 0007 (no-build console) conflicts with the Command Centre; decision pending in ADR 0009.
- Process runtime does not enforce resource limits (documented; P0-6).

## Security-sensitive

Root key custody (P0-8), capability issuance (`dh token`), TLS bootstrap
pinning (ADR 0008), host admission policy, `POST /nodes/{id}/exec` (off by
default via `allowExec: false`), backup/restore, federation agreements, the
Command Centre session cookie and CSRF guard.
