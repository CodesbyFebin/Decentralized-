# Additional production features recommended for current repository

This folder adds adaptation-ready primitives, not a replacement app and not fake backend functionality.

## Added now
1. TruthEnvelope + Freshness types: prevents missing values becoming zero/healthy/live.
2. ResourceLedgerCard: canonical MODEL A projection with invariant check.
3. AdmissionCordonCard: keeps lifecycle separate from admission/cordon and refuses action when truth is unknown.
4. CapabilityGate: prevents P2/P3/P4 UI from appearing operational before qualification maturity.
5. DesiredObservedEvidence: makes desired, observed and evidence explicitly separate.
6. FailureDomainPanel: preserves UNKNOWN rather than inventing operator/ASN/region/power diversity.
7. QualificationMatrix: exposes PASS/FAIL/BLOCKED/UNKNOWN without percentage theater.

## Backend/API work the merge agent must add, not fake in UI
- canonical ResourceLedger projection: total/ownerReserve/reserved/allocated/available + truth metadata;
- authenticated cordon/uncordon operation with operation ID and committed-state confirmation;
- node detail projection: identity, lifecycle, admission, freshness, resources, workloads, network, events, evidence;
- capability registry/maturity endpoint for P0/P1/P2/P3 feature gating;
- evidence qualification endpoint exposing signed evidence metadata, not invented status;
- failure-domain projection with UNKNOWN preserved;
- operation status endpoint so UI never treats optimistic submission as completion;
- Copilot context endpoint containing only canonical truth/evidence with source and freshness.

## Later roadmap additions
- P1 scheduler rejection explanation UI and reconciliation timeline;
- P1 replica/failure-domain topology with real observations;
- P2 provider/offers/requests/bids/leases/metering/evidence/settlement;
- P3 wallet/chain/evidence-anchor surfaces only after P2 end-to-end qualification;
- P5 federation coordinator/peer status only after open protocol implementation.
