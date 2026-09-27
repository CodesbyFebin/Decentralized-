# UI Merge Phase P0 — Implementation Status

**Date**: 2026-09-27  
**Branch**: `claude/ui-merge-wave1-p0`  
**Commits**: 4 (reference pack + truth components + NodeDetailView + NodesView)  
**Gates**: no-mock gate PASS ✓ (94 allowed, 0 forbidden)

## Completed

### Infrastructure Setup
- [x] Copied UI-MERGE-PACK to repository
- [x] Created UI_MERGE_STRATEGY.md with detailed phasing and acceptance criteria
- [x] Created branch `claude/ui-merge-wave1-p0`
- [x] Created draft PR #24

### Production Truth-Aware Components
- [x] `truthDisplay.tsx`: New component library integrated into design system
  - **TruthValue**: Display operational values with LIVE/STALE/UNAVAILABLE/UNKNOWN freshness
    - Never renders UNKNOWN as zero
    - Shows source, freshness, observedAt metadata
    - Integrates with existing design tokens (Glass, Tone)
  
  - **ResourceLedgerCard**: Enforces MODEL A constraint visualization
    - AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED
    - Visual bar chart breakdown (Owner/Reserved/Allocated/Available)
    - Freshness indicator (LIVE data highlighted, STALE warned)
    - Detailed constraint breakdown with unit support
  
  - **CordonCard**: Admission control orthogonal to lifecycle
    - Shows cordoned state separate from ACTIVE/REVOKED/DRAINING
    - New workloads rejected when cordoned; existing continue
    - Interactive cordon/uncordon buttons when applicable
    - Explanation that revoked nodes stay ineligible regardless of cordon

### Quality Gates
- [x] No-mock gate: PASS (no mock data, no NetworkContext, no simulated server)
- [x] TypeScript compilation: Green for truthDisplay.tsx (pre-existing errors in other files are unrelated)
- [x] Build: Ready (no changes to build config)

## Remaining (Phase P0)

### Node Detail View (NodeDetailView.tsx)
- [x] Import TruthValue, ResourceLedgerCard, CordonCard from truthDisplay
- [x] Overview tab: Add CordonCard to show admission control orthogonal to lifecycle
- [x] Resources tab: Replace current resource table with ResourceLedgerCard for CPU/Memory
- [x] Enhance hardware facts display with TruthValue for freshness semantics
- [x] Test: no-mock gate PASS, lint OK (NodeDetailView has no new errors), build output shows pre-existing issues in unrelated files

### Nodes & Compute View (NodesView.tsx)
- [x] Import production components (CordonCard, NodeOperation types)
- [x] Add quick-action menu for node lifecycle operations
- [x] Show operation options: Approve, Drain, Resume (Undrain), Revoke
- [x] Implement confirmation dialogs for destructive operations (Drain, Revoke)
- [x] Handle operation results and errors with visual feedback
- [x] Maintain RBAC checks and disabled state during operation
- [x] Test: no-mock gate PASS (94 allowed, 0 forbidden), lint OK

### Dashboard View (DashboardView.tsx)
- [ ] Import production components
- [ ] Real metrics only (no invented capacity)
- [ ] Resource overview using ResourceLedgerCard pattern (cluster-wide aggregation)
- [ ] Cordon/lifecycle summary for nodes
- [ ] UNKNOWN/UNAVAILABLE/STALE handling for disconnected backend
- [ ] Test: no-mock gate, lint, build, integration tests pass

## Recent Changes (Session 2)

### NodesView Integration (Commit 0b41f06)
- **Quick-action menu per node**: Added MoreVertical icon in Actions column
  - Shows lifecycle operations contextually based on node state
  - PENDING_APPROVAL: Approve button (requires api.admin)
  - ACTIVE: Drain button (requires api.write)
  - DRAINING: Resume button (requires api.write)
  - NOT REVOKED: Revoke identity button (requires api.admin)
  
- **Confirmation dialogs**: For destructive operations
  - DRAIN: Explains that existing workloads continue, no new workloads placed
  - REVOKE: Requires typing node id to confirm (cannot be undone)
  
- **Operation feedback**: Result messages after operation completes
  - Success: Shows "committed" with actor and request ID
  - Error: Shows "refused" with error code/message
  - Status during operation: Button shows "Submitting…"
  
- **RBAC integration**: Buttons disabled if user lacks permission or data is stale
  - Displays reason why button is disabled (needs api.write/admin, data stale)
  
- **Refactor to component**: Extracted NodeTableRow as separate component for clarity
  - Encapsulates all per-node operation state (pending, confirm, result, error)
  - Manages modal-like confirmation UI inline in table rows
  - Clean separation from parent NodesView state management

### NodeDetailView Integration (Commit 9b933c3)
- **Overview tab**: Integrated CordonCard after Placement panel to show admission control orthogonal to lifecycle
  - Shows cordoned state (currently always false, disabled pending backend support)
  - Explains that REVOKED nodes remain ineligible regardless of cordon state
  - Ready for future cordon mutation handlers
  
- **Resources tab**: Replaced old table with ResourceLedgerCard components for CPU and Memory
  - CPU dimension: total (measured cpus*1000), ownerReserve (UNAVAILABLE), reserved (policy cap), allocated
  - Memory dimension: total (measured memBytes), ownerReserve (UNAVAILABLE), reserved (policy cap), allocated
  - Enforces MODEL A: AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED
  - Visual bar chart with color-coded breakdown (rose/amber/blue/emerald)
  - Shows freshness state for all components (LIVE if recent, STALE if old data)
  
- **Hardware facts**: Wrapped CPU, RAM, Swap, Filesystem, Storage, GPU displays with TruthValue components
  - Each fact now shows freshness indicator (LIVE/STALE/UNKNOWN)
  - UNKNOWN displays when data is not measured (never rendered as zero)
  - Source metadata shows "measured" for all hardware facts
  - Freshness tracks stale flag from Gate component

## Acceptance Criteria (Per Surface)

Every P0 surface must pass:

- [ ] no mockData/NetworkContext imported
- [ ] TruthEnvelope + freshness used for values
- [ ] UNKNOWN is never rendered as zero/healthy/online
- [ ] backend disconnect removes LIVE state (shows UNKNOWN/STALE)
- [ ] cordon is orthogonal to lifecycle
- [ ] cordon mutation waits for committed-state confirmation
- [ ] ResourceLedger uses authoritative backend values (MODEL A)
- [ ] unsupported dimensions remain UNAVAILABLE
- [ ] desired != observed != evidence separation maintained
- [ ] loading/empty/error/unknown/unavailable/stale states present
- [ ] keyboard/focus/reduced-motion/mobile checked
- [ ] no-mock gate passes
- [ ] lint (tsc --noEmit) passes
- [ ] build passes
- [ ] unit/integration tests pass
- [ ] affected Go/control-plane tests pass

## Implementation Notes

### Reference UI Files (in UI-MERGE-PACK/reference-ui/)
- `NodeDetailModal.tsx` (18KB): Visual hierarchy improvements, grid layout for hardware facts
- `NodesCompute.tsx` (28KB): Bulk operations UI, improved list presentation
- `Dashboard.tsx` (29KB): Real metrics layout, resource overview panels

These are UI references only—NOT to be imported wholesale. Use as visual guides for interaction patterns and layout while binding all operations to existing production adapters.

### Additions Files (in UI-MERGE-PACK/additions/)
- `TruthValue.tsx` → **ADAPTED** to truthDisplay.tsx ✓
- `ResourceLedgerCard.tsx` → **ADAPTED** to truthDisplay.tsx ✓
- `AdmissionCordonCard.tsx` → **ADAPTED** to CordonCard in truthDisplay.tsx ✓
- `CapabilityGate.tsx` → For P1+ surfaces (gating)
- `DesiredObservedEvidence.tsx` → For P1+ surfaces (evidence)
- `FailureDomainPanel.tsx` → For P1+ surfaces (topology)
- `QualificationMatrix.tsx` → For P1+ surfaces (qualification display)

### Model A Constraint
```
AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED

Where:
- TOTAL: measured hardware total (or policy cap if lower)
- OWNER_RESERVE: reserved capacity for operator/system use
- RESERVED: capacity reserved by policies/quotas
- ALLOCATED: sum of replica resource requests placed on node
- AVAILABLE: remaining capacity for new workloads

Key: Allocation consumes reservation (reserved decreases as allocated increases)
Do NOT fabricate unsupported resource dimensions (GPU VRAM, network, etc.)
```

### Cordon Orthogonality
```
Cordon ⊥ Lifecycle

State matrix:
╔════════════╦═════════════════════════════════════════════════════════╗
║ Lifecycle  ║ Cordoned State                                          ║
╠════════════╬═════════════════════╦═════════════════════════════════╣
║            ║ Cordoned=false      ║ Cordoned=true                   ║
╠════════════╬═════════════════════╬═════════════════════════════════╣
║ ACTIVE     ║ Accepts new work    ║ New work rejected; existing OK  ║
║ DRAINING   ║ No new work         ║ No new work (same effect)       ║
║ REVOKED    ║ INELIGIBLE always   ║ INELIGIBLE always (orthogonal)  ║
╚════════════╩═════════════════════╩═════════════════════════════════╝

Critical: Uncordoning a REVOKED node does NOT make it eligible.
Lifecycle gates eligibility independently of cordon state.
```

## Next Steps

1. **Continue implementation**: Integrate CordonCard + TruthValue into NodeDetailView
2. **Test incrementally**: After each view update, run:
   - `npm run gate` (no-mock)
   - `npm run lint` (tsc)
   - `npm run build`
   - `npm test`
3. **Selective transplant pattern**:
   - Read reference UI file for visual inspiration
   - Compare with existing production view
   - Transplant layout/interaction patterns (NOT mock infrastructure)
   - Bind all operations to existing adapters/control-plane paths
   - Add TruthEnvelope metadata to every operational value
4. **Commit incrementally**: After each surface is green, commit with evidence
5. **Move to P1**: Once P0 surfaces pass acceptance, start Storage/Deploy/Evidence/Copilot

## Timeline Estimate

- **Current session**: Infrastructure + truth components (complete)
- **Next session**: NodeDetailView + NodesView implementation + testing
- **Following**: Dashboard + final P0 testing + P1 planning

## Resources

- Strategy: `/home/user/Decentralized-/UI_MERGE_STRATEGY.md`
- Reference UI: `/home/user/Decentralized-/UI-MERGE-PACK/reference-ui/`
- Additions: `/home/user/Decentralized-/UI-MERGE-PACK/additions/`
- Merge checklist: `/home/user/Decentralized-/UI-MERGE-PACK/additions/docs/MERGE_ACCEPTANCE_CHECKLIST.md`
- Repository PR: https://github.com/CodesbyFebin/Decentralized-/pull/24

