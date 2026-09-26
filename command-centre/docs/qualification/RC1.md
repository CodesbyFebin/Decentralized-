# Command Centre RC1 Qualification

Starting revision: `ca4d2c0` (visual upgrade, synthetic PlatformStore)
Ending revision: `667e431` (qualified) · this report is committed after it
Branch: `claude/friendly-gauss-kfxoc2` (the session's designated branch; the brief suggested `hardening/command-centre-rc1`)

**Premise correction.** The brief describes an "existing Python platform". There is none: the
authoritative platform is the Go control plane (`dh-control`, `dh-noded`). The only Python
is `conformance/python`, an independent implementation of the `dh/v1` encodings used for
conformance tests; it has no API. Everything below integrates with the Go control plane.

## Build

| | Result |
|---|---|
| Platform (Go) | `make build` OK (go1.26.4). `go test ./pkg/... ./cmd/... ./conformance/...`: all packages pass; `pkg/conformance` first failed because this sandbox's system `cryptography` package is broken (`No module named '_cffi_backend'`), and passes (136 vectors incl. Python) with a clean virtualenv. |
| Command Centre typecheck | `tsc --noEmit` with `strict: true`: 0 errors |
| Production bundle | `npm run build` (no-mock gate, then Vite): OK, main chunk 289 kB (routes lazy-loaded) |
| TS2688 `vite/client` | Reproduced only in a tree without `node_modules`; root cause is a missing install, not source/config. Nothing suppressed; `skipLibCheck` was already present in the original config. |

## Tests

| Suite | Result | Where |
|---|---|---|
| Go platform unit tests | PASS (see Build) | `go test` |
| TypeScript unit | 16/16 | `tests/unit/mapView.test.ts` on three views recorded from a real cluster (healthy, host stale, host lost) |
| Failure injection | 8/8 | `tests/integration/failure.test.ts`: offline, timeout, malformed, 5xx leak, member failover, demo labelling, config refusal, request ids |
| Integration (live cluster) | 19/19, 0 skipped | `tests/integration/controlplane.test.ts` |
| Security | inside the suites above | unauthenticated, malformed token, read-only mutation (403 from the control plane), CSRF, confirmation, sanitization, env values, actor attribution, Copilot principal binding and replay |
| Backend disconnect (e2e) | 12/12 | `tests/e2e/disconnect.mjs`, real SIGSTOP of all control-plane members |
| E2E browser | screenshots of every route, no page errors | manual run; automated browser coverage is the disconnect test only |

## Integration

| Surface | State | Why |
|---|---|---|
| Nodes | **LIVE** | `/api/v1/view` nodes; health from control-plane freshness; approve/drain/undrain/revoke via the operator API |
| Applications | **LIVE** | desired / admitted / observed per replica with admission checks |
| Deployments | **LIVE** | manifests applied with `/api/v1/apply`; generation returned by the control plane; progress derived from replica rows; survives refresh |
| Events | **LIVE** | the audit ledger is the event log (paged) |
| Evidence | **LIVE** | ledger verification on demand, milestones, attestations, chaos reports |
| Storage | **LIVE** | volumes, committed snapshots and replica verification, host storage; S3 / DePIN UNAVAILABLE |
| Domains | **PARTIAL** | desired + observed routing LIVE; DNS **UNAVAILABLE** (not observed) |
| SSL | **LIVE** | X.509 observed by edge hosts |
| Analytics | **PARTIAL** | edge counters and probe latency LIVE; visitor analytics and bandwidth **UNAVAILABLE** |
| Security | **LIVE (derived)** | concrete controls; WAF / DDoS **UNAVAILABLE**; no score |
| Billing | **UNAVAILABLE** | self-hosted; no billing subsystem |
| Team | **PARTIAL** | session principal and capability actions LIVE; no user directory |
| Settings | **LIVE** | capability matrix, connection, BFF metrics (read-only) |
| Copilot | **LIVE (derived)** | retrieval with the caller's capability + repo docs; citations; approvals executed via the control plane |

## Removed synthetic claims

Deleted `src/server/store.ts`, `fleet.ts`, `platformAdapter.ts`, `copilotService.ts`, the old
`lib/api.ts` / `types/platform.ts`, and every page built on them. That removes, among others:
10 fabricated hosts ("Ashburn Core Validator", "London GPU Cluster Rig", …) and 24 fabricated
owned/community/DePIN fleet nodes; `onlineNodesCount`/`totalNodesCount`, app/domain/cert counts,
`storageUsedGb`, visitors, bandwidth, CPU/memory percentages; the static 24 h resource wave;
security score 98 and "threats blocked"; DePIN rewards ("6274.9 DH", "$3,820"); storage totals
(68.4 TB, 12.6 TB-hours, 12,942 integrity checks, 100 % integrity); fleet deployments and
ownership mix; invoices/payment cards; `Math.random()` API keys, CIDs and ids; instant
`VERIFIED` deployments and fictional stage logs; hard-coded actor "Febin Francis"; synthetic
evidence records; the silent `PythonPlatformAdapter` → demo fallback on every method.

## Remaining simulation

**None in production paths** (`npm run gate`: 0 forbidden; report in the evidence record).
The only simulation is the opt-in `DemoPlatformAdapter` (a replay of a recorded real view,
labelled SIMULATED, mutations refused, refused in production without
`ALLOW_DEMO_IN_PRODUCTION=1`).

## Security

| | |
|---|---|
| Authentication | Root-anchored capability verified by the control plane; HttpOnly SameSite=Strict cookie scoped to `/api`; `#token=` fragment exchanged and stripped |
| RBAC | `api.read`/`api.write`/`api.admin`, enforced by the control plane (coarse; finer roles are P1-7) |
| Audit | Every mutation audited by the control plane with the actor from the capability (test asserts it) |
| Secrets | Env values never leave the BFF (test); no secrets subsystem exists in the platform (gap) |
| Copilot authorization | Retrieval uses the caller's capability; proposals bound to the proposing principal; single use; CRITICAL needs typed confirmation |
| Not done | BFF rate limiting; dependency / image scanning in CI; external review |

## Backend disconnect test

**PASS** (CC-RC1-A02). With all three control-plane members frozen: the edge kept serving
the workload (200); `/nodes` returned `504 BACKEND_TIMEOUT` with no data; the Copilot answered
"I cannot verify current platform state … I will not guess"; a deployment failed with 504;
capabilities went UNKNOWN; the console showed CONTROL PLANE UNREACHABLE with last
observations desaturated and the globe frozen. After SIGCONT every host was healthy within
~4 s and the banner cleared.

## Evidence

| Qualification id | Source digest (dh-src-digest/2) | TS source manifest (sha256 root) | Outcome | Verification |
|---|---|---|---|---|
| CC-RC1-A01 | `b3:749c3011…` (commit `cf09efb`) | `777670d8…` | **FAIL** — `cc-disconnect`: the harness's edge probe used `fetch()`, which drops the `Host` header; all console checks passed | VERIFIED |
| CC-RC1-A02 (parent A01) | `b3:73335966…` (commit `667e431`) | `e64d7798…` | **PASS** | VERIFIED |

Check with `dh evidence verify --dir evidence/CC-RC1-A02`.

## Known limitations

1. Single machine, loopback networking, TLS off on the dev control-plane API.
2. Evidence signed by an ephemeral validator key created in the sandbox.
3. `dh-src-digest/2` does not hash `.ts/.tsx`; the records bind the TypeScript source through `cc-source-manifest.txt`.
4. ADR 0007 (no-build console) conflicts with this app; ADR 0009 is proposed, not accepted.
5. RBAC is coarse (read / write / admin); no team directory.
6. No time-series metrics, visitor analytics, DNS observation, billing, DePIN or marketplace: shown as UNAVAILABLE / PLANNED.
7. Map positions exist only where `DH_REGION_LOCATIONS` is configured.
8. Automated browser tests cover the disconnect flow only (axe accessibility checks not run).
9. Copilot model path (`COPILOT_MODEL=gemini`) not exercised.
10. `bun.lock` edited by hand for two font packages; the sandbox's bun could not read the v2 lockfile.

## Final verdict

**COMMAND CENTRE QUALIFIED WITH DOCUMENTED LIMITATIONS**
