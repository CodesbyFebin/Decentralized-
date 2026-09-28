# P2-MARKET-A01 — Marketplace

Records: Provider, CapacityOffer, WorkloadRequest, Match, Reservation, Allocation, UsageRecord, EvidenceReference, SettlementInstruction.

Offer invariant: offerable <= AVAILABLE where AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED. Owner reserve wins.

Matching is deterministic/auditable across resources, architecture, locality/failure-domain constraints, price ceiling, trust/evidence and runtime constraints.

Network-reported rewards/earnings are not VERIFIED income. Usage must bind to verified execution evidence before settlement eligibility. No mandatory blockchain.
