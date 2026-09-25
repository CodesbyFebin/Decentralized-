# 0003 — Hosts hold admitted work; a bad plane can never start work

**Status:** accepted

When the control plane is unreachable, stale, frozen, untrusted, skewed or
revoking, a host keeps running work it already admitted **at the same
generation**. It starts nothing new. Stopping on staleness is opt-in per
host (`offlineAdmission: stop`).

**Why.** Sovereignty means a host's availability does not depend on the
control plane's. The asymmetry is deliberate: losing the plane must never
start work, and it should not stop work either.

**Consequences.** Admission is ordered, and the first failing check decides
(spec §9.3). A signed stop or a trusted, fresh bundle that no longer lists
an assignment is the only way work stops. Revoked hosts are cut off at
the mesh and routing instead.
