# P1-FAILURE-A01 — Failure Injection

Scenarios: AGENT_KILL, VM_POWER_OFF, NETWORK_PARTITION, NETWORK_HEAL, RESOURCE_CONTENTION, COLD_RESTART, CORDON.

For every scenario preserve T0 ledger/traffic, actual injection command and exit, detection timestamp, eligibility transition, reconciliation decision, replacement placement, traffic loss/restoration, post-state and duplicate-allocation check.

Decisive gates: FAILURE_REALITY, DETECTION, INELIGIBILITY, RECONCILIATION, REPLACEMENT, TRAFFIC_RECOVERY, NO_DUPLICATE_ALLOCATION, EVIDENCE_COMPLETE.
