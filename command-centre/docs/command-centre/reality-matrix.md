# Command Centre — Reality Matrix (RC1)

What every surface shows, where it comes from, and the truth state it is served
under. Derived from code, not intent. The authoritative backend is the
Decentralized.Host **Go control plane** (`dh-control`, `pkg/control`); there is
no Python platform in this repository (the only Python is the independent
conformance implementation in `conformance/python`, which has no API).

## Architecture

```
Browser ──(HttpOnly cookie: dh_cap)──► BFF (server.ts, src/server/bff.ts)
                                         │  authn passthrough · validation · normalization
                                         │  request ids · sanitized errors · CSRF · confirmations
                                         ▼
                                 PlatformAdapter
                         ┌───────────────┴────────────────┐
              ControlPlaneAdapter (production)    DemoPlatformAdapter (opt-in)
              Bearer = caller's capability        replay of a recorded real view,
              failover across members             every value SIMULATED,
              one deadline + 3 s breaker          all mutations refused
                         ▼
     dh-control  GET /api/v1/view · /audit · /audit/verify · /nodes/{id}/logs · /health
                 POST /apply · /apps/{n}/scale · /apps/{n}/delete · /nodes/{id}/{approve,drain,undrain,revoke}
```

`PLATFORM_ADAPTER` must be set explicitly. Production refuses `demo` unless
`ALLOW_DEMO_IN_PRODUCTION=1`. There is **no code path** from the control-plane
adapter to the demo adapter.

## Truth states

`LIVE` direct observation · `DERIVED` computed from observations · `CONFIGURED`
operator intent / configuration · `UNAVAILABLE` subsystem does not exist ·
`PLANNED` roadmap · `SIMULATED` demo replay · `UNKNOWN` no trustworthy
evidence. Unmeasured values are `null` and render as `—`, never `0`.

## Matrix

| UI capability | UI | BFF route | Control-plane source | Truth state | Notes |
|---|---|---|---|---|---|
| Hosts / nodes | `NodesView`, `NodeDetailView` | `/nodes`, `/nodes/:id` | `view.nodes` (signed host observations) | **LIVE** | Health mirrors the control plane: `FRESH`→HEALTHY, `STALE` (outside the 10 s fresh window)→DEGRADED, `health=lost` (LostAfter, default 30 s)→OFFLINE, no observation→UNKNOWN. |
| Host hardware | `NodeDetailView` → Hardware, `NodesView` memory tile | `/nodes/:id` | `view.nodes[].facts` (signed observation) | **LIVE** | Measured by the host: memory, CPU model and cores, swap, disks, GPUs, data filesystem. Fields in `facts.unknown` show "not measured". Agents that do not send `unknown` put declared memory in `facts.memBytes`, so memory from them shows "not measured". Declared capacity is shown separately as **CONFIGURED**. |
| Node operations | `NodeDetailView`, Copilot | `POST /nodes/:id/operations` | `/nodes/{id}/approve·drain·undrain·revoke` | **LIVE** | Result is the Raft-committed `Result`; REVOKE needs the node id typed as confirmation. Enforced by capability (`api.write` / `api.admin`). |
| Node logs | `NodeDetailView` → Logs | `/nodes/:id/logs` | `/nodes/{id}/logs?assignment=` over the WireGuard mesh | **LIVE** | Error text sanitized (paths stripped). |
| Applications | `AppsView`, `AppDetailView` | `/apps`, `/apps/:name` | `view.apps` (+ replica rows) | **LIVE** | Desired / admitted / observed kept separate; only fresh, currently-desired RUNNING replicas count as observed. Env values never leave the BFF. |
| Deployments | `DeployView`, `DeployNewView`, `DeploymentView` | `/deployments`, `/deployments/:app/:gen`, `POST /deployments` | `POST /apply` (returns committed generation) + replica rows | **LIVE / DERIVED** | Stages derived per generation from replica rows; initial state is never READY/VERIFIED. Scaling keeps the generation (control-plane semantics) and appears as a revision. |
| Artifacts | Deploy rail, Storage → Objects | `/deployments`, `/storage` | `view.artifacts` (CAS, attestation) | **LIVE** | Real b3 digests; attestation flag from the control plane. |
| Storage | `StorageView` | `/storage` | `view.volumes`, `nodes[].storage` | **LIVE / DERIVED** | Volume state + replica verification of committed snapshots; quotas summed from hosts. No S3 / DePIN backends (UNAVAILABLE). |
| Domains | `DomainsView` | `/domains` | manifest ingress + `edge.edges[].obs.routes` | **CONFIGURED / LIVE** | Desired routing vs observed routing vs TLS shown separately. DNS is **UNAVAILABLE** (not observed). |
| TLS certificates | `SecurityView`, `DomainsView` | `/domains`, `/security` | `edge.edges[].obs.certs` (X.509 read by edge hosts) | **LIVE** | VALID / EXPIRING (< 14 days) / EXPIRED computed from `notAfter`; FAILED / PENDING from edge state. |
| Security controls | `SecurityView` | `/security` | audit verification, host policies, facts, cluster, probe | **DERIVED** | Concrete PASS/WARN/FAIL/UNKNOWN controls; no aggregate score. WAF and DDoS telemetry **UNAVAILABLE**. |
| Audit ledger | `EvidenceView`, dashboard | `/audit` (paged), `/evidence` | `/audit`, `view.audit` | **LIVE** | Newest-first paging; immutable, read-only. |
| Ledger verification | `EvidenceView` | `POST /evidence/verify` | `/audit/verify` | **LIVE** | Re-verified by the control plane on demand. The demo replay returns UNKNOWN. |
| Milestone evidence | `EvidenceView` | `/evidence` | `view.milestones` | **DERIVED** (by the control plane) | Gaps listed as reported. |
| Edge traffic | `AnalyticsView` | `/analytics` | `edge.obs.requests/errors`, replica health probes | **LIVE** | Counters since edge start; no history, so no trend lines. |
| Visitor analytics | `AnalyticsView` | — | — | **UNAVAILABLE** | No visitor analytics subsystem. |
| Bandwidth | Dashboard, Analytics | — | — | **UNAVAILABLE** | Edges do not report byte counters. |
| Resource utilisation over time | Dashboard | — | — | **UNAVAILABLE** | Hosts report facts, not time series. |
| Map positions | globes | config | `DH_REGION_LOCATIONS` | **CONFIGURED** / UNAVAILABLE | Hosts report regions, not coordinates; unconfigured regions are listed, not placed. |
| Federation | Nodes rail, Deploy | `/capabilities` | `view.federation` | **LIVE** (capability) | Agreements managed with `dh federation`. |
| DePIN networks | Nodes / Storage tabs | — | — | **UNAVAILABLE** | No adapter; nothing installed, no rewards shown. |
| Marketplace / leases / metering | Deploy card, Billing | — | — | **PLANNED** | See `docs/GAP_MATRIX.md`. |
| Billing | `BillingView` | — | — | **UNAVAILABLE** | "Self-hosted deployment — external billing is not configured." No invoices, cards or charges. |
| Team | `TeamView` | `/session` | capability caveats | **LIVE** (session) | Identity is the signed capability; no user directory (UNAVAILABLE). |
| Settings | `SettingsView` | `/capabilities`, `/bff/metrics` | health + discovery | **LIVE** | Read-only; configuration is server environment. |
| RAG Copilot | `CopilotView` | `/copilot/*` | snapshot fetched **with the caller's capability** + repo docs | **DERIVED** | Every factual line cited; UNKNOWN when the backend is unavailable; actions are proposals approved by the same principal and executed via the control plane. Model rephrasing is opt-in (`COPILOT_MODEL=gemini`) because it sends data to a third party. |

## Security properties

- Authentication: the control plane verifies the capability chain against the cluster root on every call; the BFF keeps the token in an `HttpOnly; SameSite=Strict` cookie scoped to `/api`, never in page storage.
- Authorization: enforced server-side by the control plane (`api.read`/`api.write`/`api.admin`); a read-only capability attempting a write gets `403 PERMISSION_DENIED`.
- Actor identity comes from the capability note (e.g. `operator:console`) and is recorded in the audit ledger by the control plane. Nothing in the console hard-codes a person.
- CSRF: every mutation needs `X-DH-Console: 1`; the BFF never answers CORS.
- Destructive operations (revoke, delete, scale to 0, CRITICAL Copilot actions) need typed confirmation (`428 CONFIRMATION_REQUIRED`).
- Errors: `{ error: { code, message, request_id, details } }`; 5xx text is replaced, filesystem paths are stripped from 4xx text.
- Request ids: `crypto.randomUUID()`, or a trusted incoming UUID; forwarded to the control plane as `X-Request-Id`.
- Headers: CSP `default-src 'self'` (production), `X-Frame-Options: DENY`, `nosniff`, `no-referrer`. Fonts are bundled.

## Product invariant

DESIRED STATE ≠ OBSERVED STATE ≠ VERIFIED EVIDENCE. A deployment reaching
READY is an observation; it is not evidence. Evidence is the audit ledger
verification and signed records, shown separately.
