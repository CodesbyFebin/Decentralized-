# Decentralized.Host Command Centre — Reality Matrix

This document defines the authoritative capability mapping between the Command Centre UI, the TypeScript API/BFF adapter layer, and the underlying operational control plane.

## Truth Model States

- **LIVE**: Direct authoritative observation from running platform agents/daemons.
- **DERIVED**: Deterministically calculated from authoritative live observations.
- **CONFIGURED**: Desired/configured intent; does not claim runtime observation.
- **UNAVAILABLE**: Backend does not implement this capability; UI must honestly show unavailable/unconfigured.
- **PLANNED**: Intentionally reserved for roadmap; no operational claims.
- **SIMULATED**: Explicitly flagged demo/development adapter state.
- **UNKNOWN**: Insufficient authoritative evidence; never converted to Healthy or Online.

---

## Capability Reality Matrix

| UI Capability | UI Source Component | Platform Source | Production Truth State | Adapter Integration & Guardrails |
| :--- | :--- | :--- | :--- | :--- |
| **Nodes** | `NodesView.tsx`, `WorldMap.tsx`, `NodeTelemetryDashboard.tsx` | Node Agent heartbeat & OS telemetry | **LIVE** | Derived from real node observations. Stale heartbeats (>30s) transition to Degraded/Offline. No hardcoded node counts. |
| **Deployments** | `DeployView.tsx` | Deployment Engine / Container Orchestrator | **LIVE** | Mutations create asynchronous operations (`QUEUED`). No instant fake `VERIFIED` state. Logs stream from real runtime events. |
| **Evidence Ledger**| `EvidenceView.tsx` | Cryptographic Qualification Subsystem | **LIVE** | Sealed SHA-256 digests and signer fingerprints. Immutable history; failed qualification cannot be visually softened. |
| **Applications** | `WebsitesAppsView.tsx` | Workload Registry | **LIVE** | Distinguishes Desired Replicas from Observed Healthy Replicas. |
| **Storage** | `StorageView.tsx` | Distributed Blob / Volume Store | **LIVE / DERIVED** | Real file uploads calculate SHA-256 digests. Only labeled IPFS where IPFS subsystem is active. |
| **Domains & DNS** | `DomainsView.tsx` | Edge DNS Quorum | **CONFIGURED / LIVE** | Separates desired domain target from observed DNS resolution and TLS propagation. |
| **SSL / ACME** | `SecurityView.tsx` | ACME Certificate Manager | **LIVE** | Certificate validity, issuer, and expiration observed from real X.509 metadata. |
| **Security Controls**| `SecurityView.tsx` | Control Plane Policy Checker | **LIVE (Controls) / UNAVAILABLE (WAF Telemetry)** | Concrete controls (TLS, Auth, Audit, MFA). WAF and DDoS threat counters are marked `UNAVAILABLE` when telemetry feed is absent. |
| **Analytics** | `AnalyticsView.tsx` | Ingress Access Logs & Node Metrics | **DERIVED** | Traffic metrics derived from HTTP access logs; zero `Math.random()` in operational analytics. |
| **Billing** | `BillingView.tsx` | Autonomous Self-Hosted Ledger | **CONFIGURED** | Honestly displays: "Self-hosted autonomous mode. External credit card billing is not configured." |
| **Team & RBAC** | `TeamView.tsx` | Auth & Session Principal Store | **LIVE** | Server-side permission enforcement on all destructive mutation endpoints (`app.delete`, `node.drain`, `cert.renew`). |
| **RAG Copilot** | `CopilotView.tsx`, `DashboardView.tsx` | Hybrid RAG Engine (Docs + Live State + Gemini) | **LIVE** | Permission-filtered retrieval. Unknown states are explicitly stated. Destructive actions require explicit operator approval. |

---

## Product Invariant

$$\text{DESIRED STATE} \neq \text{OBSERVED STATE} \neq \text{VERIFIED EVIDENCE}$$

1. **Desired State**: What the operator specified in configuration or manifest.
2. **Observed State**: What distributed node heartbeats and agent probes currently report.
3. **Verified Evidence**: Cryptographically sealed, signed audit proof of qualification gates.
