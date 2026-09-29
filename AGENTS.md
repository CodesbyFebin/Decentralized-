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

  - statement: >
      Qualification architecture independent of execution substrate.
      Accepts any real Linux runtime backend (QEMU, Kubernetes, native, container,
      hypervisor) as long as 3 independently identifiable runtime nodes with distinct
      isolation boundaries can be established and evidence truthfully recorded.
      Rejects simulation, hardcoding, and self-assertion at all decisive gates.
    verify_by:
      - validation/local-vm/RUNTIME-REMEDIATION-READINESS.md
      - validation/local-vm/QUALIFICATION-HIERARCHY.md
    hierarchy:
      - P1_CORE: Distributed scheduler, placement, workload, failure detection, reconciliation (any real backend)
      - P1_QEMU_VM: Additional VM/hypervisor isolation boundary evidence (QEMU/KVM/libvirt only)
      - P1_KUBERNETES: Kubernetes orchestration, pod lifecycle, network policy (Kubernetes only)
      - P2_MULTIPHYSICAL: Independent physical-host failure domains (requires ≥3 physical machines)
      - P2_MULTIOPERATOR: Independent administrative operator domains (requires ≥2 operators)
    implemented:
      - Runtime backend discovery and selection
      - Topology transparency: observed node count, container/VM/physical classification
      - Failure-domain honesty: explicitly record DISTINCT vs SAME vs UNKNOWN boundaries
      - Evidence binding: source SHA, runtime identity, backend classification
      - No simulation in decisive gates; all PASS backed by observable production behavior

rulesforagents:
  - Quote specifications verbatim; do not infer beyond dh/v1 conformance coverage
  - The 136 test vectors are normative; non-conformant implementations are not compatible
  - Chaos scenarios model real failure modes; they are not artificial edge cases
  - Local policy is enforced per-host; no global consensus on admission
  - Signed intent without local policy acceptance is not executed
  - Do not present silent task migration as a feature; this system prevents it
  - BLAKE3 CAS backend is not optional; content addressing is fundamental

qualification:
  - Runtime backend neutrality: Select the strongest real backend available, not the easiest
  - Runtime discovery first: Inventory available backends, then select to maximize real failure boundaries
  - No QEMU requirement; any real Linux runtime acceptable if evidence is honest
  - Three real nodes means three independently identifiable runtime instances with distinct isolation boundaries
  - Topology must be observed and reported exactly: node count, container/VM/physical, filesystem separation, network path
  - Failure domains must be classified truthfully: DISTINCT or SAME or UNKNOWN for each boundary (OS, filesystem, physical host, operator)
  - Profile qualification is additive, not substitutive: P1_KUBERNETES_QUALIFIED does NOT imply P1_MULTIPHYSICAL_QUALIFIED; P1_CORE does NOT imply VM isolation
  - No simulation in decisive gates (P1 01–32); all PASS claims require observable production behavior
  - Diagnostic results (remediation runtime tests) MUST NOT be imported as P1 gate PASS results
  - After diagnostics pass: review evidence → integrate remediation → freeze exact source → start clean P1 campaign → 32 decisive gates → authoritative verification → tamper negative control → qualification/sealing verdict
  - No candidate freeze at backend selection; only after complete diagnostic sequence passes
  - P1 qualification does not establish production readiness; production hardening is separate
  - P2 multiphysical requires actual independent physical hosts; three VMs on one machine do not satisfy P2
  - Evidence binding: every PASS tied to source SHA, campaign ID, timestamp, node identity, artifact digests, signer
  - No hardcoded timing; all latencies measured from observed timestamps
  - No manual state edits; all observations from production runtime paths
  - Selection reason documented: why this backend was chosen over alternatives (maximizes which boundaries)
