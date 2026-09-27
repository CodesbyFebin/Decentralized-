# Decentralized.Host — Command Centre UI Merge Pack

Purpose: selectively transplant the useful visual/product work from `decentralized.host.zip` into the current production repository `CodesbyFebin/Decentralized-` without importing prototype truth violations.

## Current production target used for comparison

- Repository: `CodesbyFebin/Decentralized-`
- Branch: `main`
- Observed main SHA during pack creation: `9738aa5f699b14cdc92881da894dbb377edaafa1`
- Target application: `command-centre/`

This pack is **reference/adaptation source**, not a drop-in replacement for `command-centre/`.

## Intentionally excluded from the source ZIP

Do NOT recover or merge these from the original prototype:

- `src/data/mockData.ts`
- `src/context/NetworkContext.tsx`
- `server.ts`
- `src/App.tsx`
- `src/main.tsx`
- prototype `package.json`, Vite config, tsconfig, metadata, env files

Reason: these files contain or enable simulated operational state, local-only infrastructure mutations, fake node/deployment/DePIN state, or prototype routing/server behavior. The production repository already has a real router, adapters, no-mock gate, tests and control-plane integration.

## How to use this pack

1. Keep the existing `command-centre` shell, React Router, session layer, platform/control-plane adapters, truth envelopes, tests and `scripts/no-mock-gate.mjs`.
2. Use files under `reference-ui/` as visual and interaction references.
3. Port JSX/layout/styles into the corresponding existing production components. Do not overwrite production files wholesale.
4. Replace every `useNetwork()` dependency with current production data hooks/adapters.
5. Replace all local mutations with authenticated production API/control-plane operations.
6. Preserve UNKNOWN / UNAVAILABLE / STALE / LIVE truth semantics. Missing data must never become zero, healthy, online or verified.
7. DePIN marketplace surfaces remain PLANNED/UNAVAILABLE until P2 backend qualification exists.
8. Copilot may describe infrastructure only from canonical platform context/evidence; no invented fallback diagnostics.
9. Cordon must remain orthogonal to lifecycle: e.g. ACTIVE + cordoned=true. Existing workloads continue; new placements are rejected.
10. Owner reserve must come from the authoritative ResourceLedger projection. Under MODEL A: AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED, with allocation consuming reservation.

## Suggested transplant priority

P0 / current work:
- `components/NodeDetailModal.tsx`
- `pages/NodesCompute.tsx`
- `components/AddNodeModal.tsx`
- `components/CommandPalette.tsx`
- `pages/Dashboard.tsx`

P0/P1 supporting surfaces:
- `pages/Storage.tsx`
- `pages/Deploy.tsx`
- `components/NewDeploymentModal.tsx`
- `components/DeploymentDetailModal.tsx`
- `components/GlobeMap.tsx`
- `pages/EvidenceAudit.tsx`
- `pages/RagCopilot.tsx`

Later/future surfaces:
- `pages/DePINMarketplace.tsx` — P2 visual reference only
- remaining domain/security/analytics/billing/team/settings/apps pages — selectively transplant visual improvements after backend truth binding.

## Acceptance gate for every transplanted surface

A surface is mergeable only when:

- no mock data is imported;
- no prototype NetworkContext is imported;
- no fake success response exists;
- operational mutations use the canonical backend path;
- live values carry source/freshness/truth state;
- missing data renders UNKNOWN/UNAVAILABLE rather than `0`;
- backend disconnect removes LIVE presentation;
- loading/empty/error/unknown/unavailable/stale states exist;
- keyboard/focus/reduced-motion/responsive behavior remains valid;
- existing no-mock gate, typecheck, build and relevant tests pass.


## Additional production features in this final pack

The `additions/` directory was added after the first merge pack. It contains truth-aware, adaptation-ready UI primitives plus API/merge contracts for features the current repository needs as it moves from P0 into P1/P2. These additions intentionally contain no mock infrastructure and no fake server implementation. See `additions/docs/CURRENT_REPO_FEATURE_ADDITIONS.md`.
