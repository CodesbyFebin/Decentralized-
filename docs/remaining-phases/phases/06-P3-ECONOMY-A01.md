# P3-ECONOMY-A01 — Optional Settlement/Web3

SettlementAdapter: quote, authorize, capture, refund, status, evidence_binding.

Adapters may implement test ledger, regulated payment partner or Web3 network. Web3 is optional: testnet first, chain ID pinned, contract addresses configuration-bound, replay protection, idempotency, explicit finality, no private keys in evidence bundles. Settlement never substitutes for execution evidence.
