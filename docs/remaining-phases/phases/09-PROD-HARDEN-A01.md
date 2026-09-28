# PROD-HARDEN-A01

Security: threat model, least privilege, secret handling, sandbox escape tests, SBOM, signed builds/provenance, rate limits, tenant isolation, audit verification.

Reliability: HA control plane, durable queues, backup/restore, DR, rolling upgrade, version skew, idempotent reconciliation, capacity exhaustion and brownout modes.

Scale tests must state topology/hardware. Never extrapolate local VM results into production capacity.

Production release requires independent security review disposition plus sealed decisive qualification campaigns.
