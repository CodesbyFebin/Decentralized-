# Evidence index

Every validation attempt, newest last. Attempts are never deleted or
overwritten; a failed attempt stays next to the one that superseded it.
Check any record with `dh evidence verify --dir evidence/<dir>`.
Rules and schema: `docs/evidence/README.md`.

| Evidence id | Stage | Environment | Source digest | Outcome | Verification | Notes |
|---|---|---|---|---|---|---|
| PV1-S1-A01 | PV1-S1 | Colima VM on macOS | `b3:5bde0c09…` (digest/0) | INFRA_FAILURE | no record | the VM stopped; nothing ran |
| PV1-S1-A02 | PV1-S1 | Linux container in a 2-vCPU Colima VM | `b3:5bde0c09…` (digest/0) | FAIL (legacy record) | VERIFIED | **diagnostic only**; found D1 and D2; see `PV1-S1-HISTORY.md` |
| REF-MAC-A01 | REF-MAC | macOS reference (Mac mini, i5-8500B, 8 GB) | `b3:73d7ee46…` (digest/1) | PASS | VERIFIED | all 13 steps, 10/10 integration, 17/17 chaos with no skips. **Limitation:** digest/1 omitted `pkg/evidence` from the source identity, so the tree it names is incomplete (the binaries are pinned by hash). Superseded by REF-MAC-A02. |
| REF-MAC-A02 | REF-MAC | macOS reference, same machine | `b3:2832989a…` (digest/2) | **FAIL** | VERIFIED | parent REF-MAC-A01. Integration failed: `TestArtifactFetchSkipsDeadHolder` read the holder list before holders were recorded (a test race), and M2 "interrupted write" timed out on reads after recovery again, **which refutes the D4 diagnosis**. Chaos 17/17 with no skips. |
| REF-MAC-A03 | REF-MAC | macOS reference, same machine | `b3:2a23e1da…` (digest/2) | **PASS** | VERIFIED | parent REF-MAC-A02. All 13 steps on the first attempt: 10/10 integration (including 4 regression tests), 17/17 chaos with no skips, gossip 20/20, conformance including Python, TLS install. Includes the D4 fix (a draining fallback needs recent proof of life). **This is the current reference baseline.** |
| CC-RC1-A01 | CC-RC1 | Linux container (cloud sandbox), `dh dev up` on loopback | `b3:749c3011…` (digest/2) + TS manifest `777670d8…` | **FAIL** | VERIFIED | Command Centre qualification. `cc-disconnect` failed: the harness probed the edge with `fetch()`, which drops the `Host` header. All console checks passed. |
| CC-RC1-A02 | CC-RC1 | same environment | `b3:73335966…` (digest/2) + TS manifest `e64d7798…` | **PASS** | VERIFIED | parent CC-RC1-A01. Typecheck, 24 unit/failure tests, no-mock gate, build, 19 live-cluster integration tests, 12/12 disconnect checks. Qualifies the console only; promotes nothing on the platform. |
| NODE-A01-A01 | NODE-A01 | Linux container (cloud sandbox), single machine, loopback | `b3:c55703b2…` (digest/2, commit `70403c4`) | **FAIL** | VERIFIED | Sovereign node foundation. Every NODE-A01 check PASS (enrolment refusals, key custody, measured facts, freshness, tamper, revoked key). Failed on `gossip-repeat` (2/20) and M3 (gossip membership): a pre-existing join defect that also reproduces on the base commit. See `docs/qualification/NODE-A01.md`. |
| PV1-S1-A03 | PV1-S1 | independent Linux VM (≥ 4 vCPU, 8 GB) | must be `b3:2a23e1da…` (bundle `dh-src-b3-2a23e1da.tgz`) | *pending* | — | needs a Linux machine; procedure in `validation/pv1/README.md` |

The macOS reference promotes nothing on its own. It is the baseline Linux
and multi-machine runs are compared against.
