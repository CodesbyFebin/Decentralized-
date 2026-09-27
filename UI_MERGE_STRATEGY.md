# Command Centre UI Merge Strategy

**Merge Pack Source**: `UI-MERGE-PACK/` (validated, no mock infrastructure)  
**Production Authority**: `/command-centre/src/` (existing router, adapters, tests, control-plane binding)  
**Date**: 2026-09-27  
**Status**: IN PROGRESS

## Phases

### Phase P0: Node Management + Dashboard (Current Sprint)

#### P0.1: Node Detail Modal + Nodes & Compute Page
- Transplant visual hierarchy and interaction patterns
- Replace operationally with existing production adapters
- Bind lifecycle/cordon/ownership/resource state to canonical backend
- Support UNKNOWN/UNAVAILABLE/STALE freshness states
- Verify ResourceLedger MODEL A constraint: AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED
- Test: no-mock gate, typecheck, build, relevant tests pass

#### P0.2: Dashboard
- Real metrics/topology only (no invented capacity)
- UNKNOWN/UNAVAILABLE/STALE on missing/disconnected
- Workload status from FleetInventory, not mock
- Evidence-backed health signals only
- Test: full suite + no-mock gate

### Phase P1: Storage + Deploy + Evidence (Next Sprint)
- Storage: real backend state, no invented capacity/integrity
- Deploy: real build/deployment state, no fabricated progress
- Evidence/Audit: evidence-backed claims, ACT requires approval path
- Copilot: canonical context only, no fallback diagnostics

### Phase P2+: Future Surfaces (Gated)
- DePIN Marketplace: visual reference only, UNAVAILABLE until P2 backend qualification
- Web3/Federation: UNAVAILABLE until respective qualification phases

## Acceptance Checklist (Per Surface)

- [ ] no mockData/NetworkContext imported
- [ ] TruthEnvelope + freshness used for values
- [ ] UNKNOWN is never rendered as zero/healthy/online
- [ ] backend disconnect removes LIVE state
- [ ] cordon is orthogonal to lifecycle
- [ ] cordon mutation waits for committed-state confirmation
- [ ] ResourceLedger uses authoritative backend values
- [ ] unsupported dimensions remain UNAVAILABLE
- [ ] desired != observed != evidence separation
- [ ] loading/empty/error/unknown/unavailable/stale states present
- [ ] keyboard/focus/reduced-motion/mobile checked
- [ ] no-mock gate passes
- [ ] typecheck/lint/build/tests pass
- [ ] affected Go/control-plane tests pass

## File Mapping

### Additions (production primitives - adapt into existing design system)
- `TruthValue.tsx` → truth envelope presentation
- `ResourceLedgerCard.tsx` → MODEL A constraint display
- `AdmissionCordonCard.tsx` → lifecycle + cordon orthogonal display
- `CapabilityGate.tsx` → qualification-gated surfaces
- `DesiredObservedEvidence.tsx` → separation of concerns
- `FailureDomainPanel.tsx` → network/failure topology
- `QualificationMatrix.tsx` → evidence-backed capability matrix
- truth.ts, resourceLedger.ts → utilities

### Reference UI → Production (Selective Transplant)
- `NodeDetailModal.tsx` → `command-centre/src/components/pages/NodeDetailView.tsx`
- `NodesCompute.tsx` → `command-centre/src/components/pages/NodesView.tsx`
- `Dashboard.tsx` → `command-centre/src/components/pages/DashboardView.tsx`
- `Deploy.tsx` → `command-centre/src/components/pages/DeployView.tsx`
- `Storage.tsx` → `command-centre/src/components/pages/StorageView.tsx`
- `EvidenceAudit.tsx` → `command-centre/src/components/pages/EvidenceView.tsx`
- `RagCopilot.tsx` → `command-centre/src/components/pages/CopilotView.tsx`
- `CommandPalette.tsx` → enhance existing layout/CommandPalette.tsx
- `Sidebar.tsx` → enhance existing layout/Sidebar.tsx

## Success Criteria

**Per Phase**:
- All surfaces in phase pass acceptance checklist
- Full test suite (unit + integration + no-mock) passes
- Control-plane integration validated
- Evidence-backed operations only (no optimistic local state)
- Freshness semantics preserved (STALE/UNKNOWN never become LIVE)

**Overall**:
- P0 surfaces complete by end of current session
- All gates passing (typecheck, build, test, no-mock)
- Ready for P1 surfaces (storage/deploy/evidence/copilot)
- PV1 infrastructure qualification still pending (external blocker)

## Technical Notes

- ResourceLedger MODEL A: `AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED`
  - Allocation consumes reservation (reserved decreases as allocated increases)
  - Do not fabricate unsupported resource dimensions
  
- Cordon Orthogonality: Cordon flag does NOT mutate lifecycle state
  - ACTIVE + cordoned=true: existing workloads run, new placements rejected
  - REVOKED + cordoned=false: ineligible regardless of cordon state
  - Canonical path: client -> API -> authorization -> Raft proposal -> committed FSM -> FleetInventory refresh
  
- Truth Envelope: Every operational value carries source/freshness/truth state
  - LIVE: real-time from backend
  - STALE: known but older than threshold
  - UNAVAILABLE: not supported or missing
  - UNKNOWN: not yet discovered
  - Never render UNKNOWN as zero, UNAVAILABLE as healthy, STALE as LIVE

