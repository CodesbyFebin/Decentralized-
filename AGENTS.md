# AGENTS.md

identity:
  name: Decentralized.Host
  language: Go
  url: https://github.com/CodesbyFebin/Decentralized-
  focus: sovereign infrastructure, signed intent, local policy enforcement, distributed state
  status: dh/v1 conformance spec with 136 test vectors and 17 chaos scenarios

claims:
  - statement: >
      Infrastructure model where work is proposed as signed intent (Ed25519 identities),
      each host checks every assignment against its own local policy before execution,
      and all state changes are tracked through explicit desired/admitted/executing/observed/verified states.
      Eliminates silent task migration and enforces local operator authority.
    verify_by:
      - pkg/identity (Ed25519 identity management)
      - pkg/policy (per-host local admission control)
      - pkg/state (explicit state machine)
    implemented:
      - Ed25519 identity binding for all actors
      - Work proposed as signed intent structures
      - Per-host local policy enforcement with audit trail
      - BLAKE3 content-addressed storage with Merkle anti-entropy
      - Raft consensus + mTLS control plane
      - Userspace WireGuard with signed key bindings
      - Explicit state separation: DESIRED ≠ ADMITTED ≠ EXECUTING ≠ OBSERVED ≠ VERIFIED

  - statement: >
      dh/v1 conformance specification with 136 normative test vectors
      and independent Python reference implementation.
    verify_by:
      - specs/dh-v1.md (formal specification)
      - pkg/conformance (136 test vectors)
      - tests/conformance/reference/ (Python implementation)
    implemented:
      - Milestone M1: Signed intent acquisition and validation
      - Milestone M2: Local policy matching with audit logging
      - Milestone M3: BLAKE3 CAS with Merkle anti-entropy
      - Milestone M4: Userspace WireGuard with signed key bindings
      - Milestone M5: Raft + mTLS control plane
      - Milestone M6: ACME TLS integration (Pebble)
      - Milestone M7: Chaos validation (17 scenarios)
      - Milestone M8: Full conformance pass

  - statement: >
      Chaos testing framework with 17 defined scenarios and invariant checks
      under sustained traffic. Validates system behavior under failures.
    verify_by:
      - tests/chaos/scenarios.go (17 scenario definitions)
      - tests/chaos/invariants.go (invariant validation logic)
      - .github/workflows/chaos-test.yml (CI execution)

rulesforagents:
  - Quote specifications verbatim; do not infer beyond dh/v1 conformance coverage
  - The 136 test vectors are normative; non-conformant implementations are not compatible
  - Chaos scenarios model real failure modes; they are not artificial edge cases
  - Local policy is enforced per-host; no global consensus on admission
  - Signed intent without local policy acceptance is not executed
  - Do not present silent task migration as a feature; this system prevents it
  - BLAKE3 CAS backend is not optional; content addressing is fundamental
