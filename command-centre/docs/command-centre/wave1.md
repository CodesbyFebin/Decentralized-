# Command Centre — Wave 1 (real current platform)

Scope: the pages whose backend exists today. Wave 2+ pages (Deploy wizard,
Domains, Storage, Security, Analytics, Marketplace, Billing, DePIN networks)
were not rebuilt; they keep their RC1 state and show future capabilities as
PLANNED / UNAVAILABLE.

## Classification

| # | Page | Route | Before (RC1) | After | Notes |
|---|---|---|---|---|---|
| 01 | Dashboard | `/` (`/dashboard` redirects) | PARTIAL | **LIVE** | Six overview cards with state, source and freshness. Topology globe plus a table equivalent (my nodes, private mesh, applications, storage, edge; community/DePIN PLANNED). Health as counts, not percentages. Recent operations from the audit ledger. Attention required (derived server-side, each item with its source). Evidence snapshot (signed records, FAIL kept red). Quick actions. The always-full "CPUs measured" ring and zero-filled gauges were removed. |
| 02 | Apps | `/apps` | LIVE | **LIVE** | Filters: all, running, converging, degraded, stopped, unknown. Search by name, manifest hash, domain or node. Columns: desired, observed, verified, placement, artifact, endpoint, updated. Empty state reads "No applications deployed." |
| 03 | App detail | `/apps/:name` | LIVE | **LIVE** | Desired / observed / verified side by side with a convergence result. Placement table with per-replica freshness. Twelve tabs. Stop is scale to 0 with typed confirmation. Delete explains its impact and needs typed confirmation. Restart and rollback are shown disabled with the reason: the control plane has no restart operation and keeps manifest hashes, not previous manifests. |
| 06 | Nodes | `/nodes` | LIVE | **LIVE** | Observed CPU, memory, storage and GPU, each with "x/y hosts measured". Unmeasured values are never added in as zero; the sums are per-host reports. Table: lifecycle, freshness (incl. EXPIRED for lost hosts), CPU, memory, storage, GPU, workloads, last seen. |
| 07 | Add node | `/nodes/add` | MISSING | **LIVE** | Node type picker, and the exact `dh node invite` and `dh-noded` commands. Pending hosts with Approve (api.admin). Invite list from the control plane with state, expiry countdown, per-invite enrolment progress and Revoke. The console cannot create join tokens: they must be signed by the root key, which only the operator CLI holds. |
| 08 | Node detail | `/nodes/:id` | LIVE | **LIVE** | Ten tabs. Identity: public key, lifecycle, freshness, uptime. "Agent version: not reported" rather than a guess. Resources compares measured total, host policy limit and allocated (from manifests); available is the smaller of total and policy limit, minus allocated. Owner reserve shows "no record". Network keeps WireGuard, membership and transport as separate rows; interface and NAT discovery UNAVAILABLE. Actions: Approve, Drain, Resume, Revoke (typed confirmation). There is no cordon, so none is offered. |
| 16 | Evidence | `/evidence` | PARTIAL | **LIVE** | Signed validation records are the flagship table: outcome filter, stage filter, source digest, commit, signer, verification state. FAIL rows are highlighted, never hidden. Ledger verification, milestones, attestations and chaos reports are kept. The audit table moved to Activity. |
| 17 | Evidence detail | `/evidence/:id` | MISSING | **LIVE** | Failure banner first. Subject, scope, exclusions, limitations. Binding: source digest, commit, binary digests, signer, signature. Gate results with per-step logs. **Verify** runs `dh evidence verify` on the stored record. **Export** downloads the signed `record.json` unchanged. |
| 27 | Activity | `/activity` | MISSING | **LIVE** | Audit explorer: filters for actor, action, resource, source and date. Paged from the ledger, entry and evidence hashes, links to nodes and apps. **Refused** tab: the control plane's replicated rejection log (bad signatures, replays, invalid join tokens). Request and operation ids are not recorded by the ledger, and the page says so. |
| 28 | Operations | `/operations` | MISSING | **DERIVED** | The platform has no Operation resource. Deployments converging, drains and volume replication are read from current state, each with its source and steps. No percentages. While the control plane is unreachable the last observed status is shown and labelled as not current. |
| 26 | Settings | `/settings` | PARTIAL | **LIVE** | Eleven sections. Every setting shows its current value, source (env var or CLI flag) and impact. The danger zone names the real CLI commands; it does not pretend the browser can rotate keys or revoke sessions. The server route never returns credentials. |
| 29 | Command palette | ⌘K | PARTIAL | **LIVE** | Pages, hosts, apps, deployments, volumes, domains, evidence records. Actions (deploy, add node, open logs) only navigate; nothing destructive runs from search. |

## Shared components added

- `TruthValue`, `MetricValue`: value + truth state + freshness + source + observedAt.
- `FreshnessBadge`: FRESH / STALE / EXPIRED / UNREACHABLE / UNKNOWN.
- `StateComparison` / `ConvergenceBadge`, `EvidenceSeal`, `Digest` (copy).
- `LogViewer`: a polled tail, not a push stream, and it says so. Pause, search, text-match severity filter, download, and a LOG STREAM DISCONNECTED banner that keeps existing lines.
- Every status is icon + text + colour.
- `Gate` now provides a page-stale context. Every freshness badge inside a stale page renders UNREACHABLE, so no page can show FRESH during an outage.

## Backend added for Wave 1

- Control plane: `GET /api/v1/invites` (api.admin; no token material).
- BFF routes:
  - `/invites`, `/invites/:nonce/revoke`;
  - `/evidence/records[/:id[/record.json|/steps/:step/log|/verify]]` (`DH_EVIDENCE_DIR`, `DH_CLI`; ids confined to the directory);
  - `/operations`, `/rejections`, `/settings`;
  - `/overview` now includes attention items and volumes;
  - `/nodes/:id` includes allocation.
- CLI: `dh node invite-revoke` accepts a token, a token file or a nonce.

## Defects found and fixed on the way

- App resources were read as strings (`cpu`/`mem`) but the control plane sends `cpuMilli`/`memBytes`. Every app showed "—" for resources. Fixed, and covered by a unit test on a recorded real view.
- Freshness pills could show FRESH inside a page that had lost the control plane.
- Evidence step durations were rendered with the age formatter ("2s ago").
- Two-column grids overflowed horizontally on phones.

## Definition of done: status

| Criterion | Wave 1 status |
|---|---|
| Real route, real API | yes, all pages |
| Auth / RBAC | control-plane enforced. Read-only gets 403 on invites and on every mutation (live tests). Controls a capability cannot use are hidden or disabled with the reason. |
| Loading / empty / error states | `Gate`, `Empty`, `ErrorState` on every page |
| Stale / unreachable | `tests/e2e/wave1-disconnect.mjs`: **50/50** on a live cluster. Mutation check: with the page-stale badge fix removed, the gate fails on node detail. |
| Actions real and audited | approve, drain, resume, revoke, scale, stop, delete, invite revoke, evidence verify. Invite revoke asserted in the audit ledger (live test). |
| Responsive | screenshots at 1536 and 390 px: no page errors, no horizontal overflow |
| Keyboard | cards and quick actions are links; tabs, filters and log controls are buttons with focus rings |
| Accessibility | status is never colour-only; topology has a table equivalent. **An automated WCAG audit (axe) was not run**: axe is not installed in this environment. |
| No-mock gate | PASS; a planted `'VERIFIED'` assignment still fails it |
| Tests | unit 37, live integration 26 (7 new), e2e disconnect 12 + 50 |

## Evidence

`evidence/CC-W1-A01`: **PASS**, VERIFIED, at commit `f7242f6`. Reproduce with `validation/cc-wave1.sh`.

## Not done in Wave 1

- A real Operation resource.
- Request ids in the audit ledger.
- A cordon lifecycle state.
- An owner-reserve record.
- Interface and NAT discovery.
- Token creation from the browser. This is by design: the root key stays with the CLI.
- An automated accessibility audit.
