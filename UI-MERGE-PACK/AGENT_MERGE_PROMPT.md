# Autonomous merge directive

Use the current repository as engineering authority and this pack only as UI/product reference.

Do not replace `command-centre/`. Do not restore excluded prototype files. For each reference component/page, compare it with the existing production equivalent, transplant only visual hierarchy and useful interaction patterns, then bind every operational value/action to the existing production adapter/API/control-plane path.

Priority order:
1. Node Detail + Nodes & Compute: lifecycle, cordon/admission, freshness, authoritative ResourceLedger totals/ownerReserve/reserved/allocated/available, workloads, network, events, evidence.
2. Dashboard: real metrics/topology only; UNKNOWN/UNAVAILABLE/STALE on missing/disconnected sources.
3. Storage and Deploy: real backend state only; no invented capacity, integrity, build/deploy progress or verification.
4. Evidence/Audit and Copilot: evidence-backed claims only; ACT requires existing approval/control path.
5. DePIN Marketplace: retain as gated future surface; do not expose as LIVE until P2 qualification.

For cordon/uncordon, require the canonical path: authenticated client -> API -> authorization/policy -> Raft proposal -> committed FSM -> canonical node state -> FleetInventory/scheduler -> Command Centre refresh. Never confirm from optimistic local state alone.

For owner reserve, consume the canonical backend projection. MODEL A invariant:
AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED.
Allocation consumes reservation, so reserved decreases as allocated increases. Do not fabricate unsupported resource dimensions.

Run the repository's canonical no-mock gate, typecheck/lint, build, unit/integration tests, accessibility checks and affected control-plane tests. Report CHANGED / TESTED / PASSED / FAILED / BLOCKED / SIMULATED / REMAINING / EVIDENCE / SOURCE_SHA. Do not label a surface VERIFIED or SEALED from UI tests alone.


## Additional production hardening supplied in this updated pack

Inspect `additions/` before adapting the reference UI. These files are deliberately backend-neutral primitives and contract requirements. Prefer adapting them into the repository's existing design system rather than creating a parallel component library.

Required additions from this pack:
- canonical TruthEnvelope/freshness presentation;
- ResourceLedger MODEL A invariant surface;
- admission/cordon surface separate from lifecycle;
- capability maturity gates for P2/P3/P4/P5 surfaces;
- desired/observed/evidence separation;
- failure-domain UNKNOWN preservation;
- qualification matrix without fake scores;
- operation-status confirmation for mutations;
- evidence-backed Copilot context only.

Do not implement missing backend endpoints as fake frontend adapters. If the current repository lacks a required endpoint, mark the UI capability UNAVAILABLE/BLOCKED and implement the backend path in its proper Go/control-plane package before enabling the surface.
