# Command Centre API contract requirements

Operational responses should expose value, source, observedAt, freshness and state. Mutations return an operation ID and accepted/committed state; the UI must refresh canonical state before confirming completion.

## Node admission
POST /api/v1/nodes/:id/cordon
POST /api/v1/nodes/:id/uncordon
GET  /api/v1/operations/:operationId

Required path: authenticated client -> authz/policy -> Raft proposal -> quorum commit -> FSM Apply -> canonical state -> FleetInventory/scheduler -> UI refresh.

## Resource ledger
GET /api/v1/nodes/:id/resources
Fields per supported dimension: total, ownerReserve, reserved, allocated, available, unit, generation/version, truth metadata. Missing dimensions are UNKNOWN/UNAVAILABLE, never 0.

## Capability registry
GET /api/v1/capabilities
Return capability + maturity + evidence reference. UI surfaces use this to gate Marketplace/Web3/Federation features.

## Evidence
GET /api/v1/evidence/:id
Return digest, signer, source SHA, manifest/signature verification state and gate statuses. Never infer VERIFIED from RUNNING.
