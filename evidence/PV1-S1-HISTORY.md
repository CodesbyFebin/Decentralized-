# PV1-S1 — Linux single-machine parity: attempt history

Source tree for every attempt below: `b3:5bde0c097b71493bf7a5c625f4ba3a8171c82639e04159cc22dc484de2ec524d`
(`dh evidence digest`, 140 files). This covers A01 and A02. The tree has changed
since then (the D1–D3 fixes and their tests), so A03 pins its own digest.

Classification rules:
- **INFRA_FAILURE**: the environment failed; the result says nothing about Decentralized.Host.
- **DIAGNOSTIC**: ran, but in an environment that cannot establish the claim.
  Findings are kept and followed up; nothing is promoted.
- **AUTHORITATIVE**: ran in the environment the claim implies; can promote.

---

## PV1-S1-A01 · INFRA_FAILURE

- **Environment:** Colima Linux VM (2 vCPU, 4 GiB) on this macOS machine, Docker 29.5.2
- **When:** 2026-09-24, about 14:29Z
- **What happened:** `docker run` failed immediately. The Colima VM had stopped
  and its socket was gone. No step ran and no record was sealed.
- **Evidence:** none (the empty evidence directory was removed).

## PV1-S1-A02 · DIAGNOSTIC (sealed record: FAILED)

- **Environment:** `golang:1.26-bookworm` container (Debian 12, Go 1.26.8, Linux
  6.8.0-117-generic) in the Colima VM (**2 vCPU, 4 GiB**) on this macOS machine.
  No Docker-in-Docker.
- **Record:** `pv1.1-linux-parity-docker-20260924T143020Z/record.json`
  (signed; check it with `dh evidence verify --dir …`). All 12 steps ran to the end.
  This record uses the pre-release schema: `format: dh-validation/1`, a
  `passed` flag instead of `outcome`, and `dh-src-digest/0`. Its signature and
  file hashes still verify. The outcome under today's rules is FAIL.
- **Scope:** diagnostic only. **Does not establish Linux parity.** The VM is
  resource-starved (2 vCPU for a race-enabled parallel suite), and the same
  Colima instance had just failed in A01.

| Step | Result | Follow-up |
|---|---|---|
| build, vet, gofmt | PASS | none |
| unit tests with `-race` | FAIL: `TestGossipMembership`, "join contacted 0" (one join over the tunnel exceeded memberlist's 10 s TCP timeout) | **Regression candidate, not fixed.** It passed 10 of 10 isolated runs in the same VM (5 with race, 5 without). The original assertion is kept and must be re-run on A03. If it reproduces on stable Linux, it is a product defect; if not, repeat it under stress before calling it environment-specific. |
| conformance (check, Go, stdio, Python) | PASS | none |
| integration | FAIL: M5 setup timed out waiting for the edge to route all replicas; the other 6 passed | Explained by defect D1 below |
| chaos | 11 PASS / 4 FAIL / 2 SKIP | host-crash, packet-chaos and stale-generation failed at setup (D1). interrupted-deployment: 2 of 224 requests failed (D2). SKIPs: oom and postgres-outage (no Docker inside the container) |
| tls-install runbook | PASS | none |

### Product defects found by A02 (environment-independent)

Both reproduce **on macOS**. Each has a regression test that fails without its
fix and passes with it, so they are classified as product defects whatever the
A02 environment.

- **D1 — an edge host could not route to a replica it ran itself.** The mesh
  forwarder admitted only *peer* edge hosts. An edge dialling its own mesh
  address was refused, so the replica was ejected. Random placement hid this on
  earlier runs.
  - Fix: `pkg/node/support.go` (`allowServiceClient`).
  - Test: `tests/integration/edge_self_test.go`. Without the fix, the replica
    ends `ejected`; with it, the route is `routing` and returns HTTP 200.
- **D2 — rolling updates dropped requests.** The host stopped the old generation
  before starting the new one. The edge learned "draining" only afterwards, and
  the rollout budget counted host readiness rather than edge routing.
  - Fixes:
    - make-before-break on the host for stateless replicas (`pkg/node/workloads.go`, `handover`);
    - the edge falls back to live draining endpoints instead of returning 503 (`pkg/edge/edge.go`).
  - Test: `tests/integration/rollout_test.go`, one replica under traffic.
    Without the fix, 12 of 126 requests failed; with it, 0 of 139.
  - Remaining: replicas **with volumes** still stop before they start (two
    writers must never share a volume). They depend on other replicas and the
    edge fallback.

### Product defect found by the macOS reference run after the D1/D2 fixes

- **D3 — a rescheduled replica could wait minutes for its artifact.** The
  fetch tried chunk holders in order, and a dead or partitioned holder cost
  the full 20 s timeout on each of the artifact's 125 chunks. Chaos
  host-crash and network-partition failed intermittently (2 of 17
  scenarios in one run, 1 in 3 host-crash repetitions), depending on
  whether the dead host came first in the holder list. The bug predates D1
  and D2, and random holder order had hidden it.
  - Fix: holders that fail are skipped for 30 s by every fetch, and the
    per-chunk timeout is 8 s (`pkg/node/support.go`).
  - Test: `tests/integration/fetch_dead_holder_test.go` kills the first-listed
    holder, then scales. Without the fix, the new host is still "fetching and
    verifying artifact" after 45 s. With it, all 3 replicas run 9.9 s after
    scaling.

### Found by repeated full-suite runs in the macOS reference environment

- **D4 — a regression introduced by the D2 fix: the edge could hang a request on a dead host.**
  - **First diagnosis, wrong:** "storage recovery stalls on dead chunk
    holders". I made a fix for that (skip holders gossip reports dead) without
    reproducing the failure. It passed 6 suites and then failed again in
    REF-MAC-A02, so the diagnosis was refuted.
  - **Reproduction:** M2 run under concurrent load, with new diagnostics
    (edge route states, last reads, recovery-host journal). The recovered
    replica on host-b was routing and a `beta` read returned 404 at once, but
    the first `alpha` read hung for the whole 30 s client timeout.
  - **Root cause:** D2 added an edge fallback to *draining* endpoints. For
    a moment after the primary died, the only endpoints were draining ones on
    dead hosts. The edge sent the request over a pooled connection to a dead
    host, and it waited for headers instead of failing fast. Every M2 failure
    came after D2; none came before it.
  - **Fix:** a draining endpoint is a last resort only if the edge saw it
    alive (a probe or a response) within the last 3 s (`pkg/edge/edge.go`).
  - **Test:** `pkg/edge/pick_test.go` (deterministic). With the old rule it
    fails ("picked a draining endpoint with no proof of life"); with the
    fix it passes. The single-replica rollout still dropped 0 of 135 requests.
  - **Reverted:** the gossip-based holder skip, because its rationale was
    disproven. D3's failed-holder backoff stays; it has its own deterministic
    test.
- **D5 — a revoking control-plane member kept the revoked host as a mesh
  peer for up to 2 s.** Its own mesh refreshed only on a periodic pass. In
  M3, a race with hosts dropping the peer let the control plane still ping
  the revoked host. Fix: every committed change refreshes the member's own
  mesh at once (`pkg/control/reconcile.go`).
  - Test: M3 now pings 300 ms after the committed revocation. Without the
    fix, 2 of 2 runs failed; with it, 3 of 3 passed.
  - Side fix: mesh ping stops after the first failed sample. An unreachable
    host took 15 s (5 × 3 s), which is longer than the console's 8 s request
    timeout.
- **Open — M1 "control-plane outage is not workload outage".** The
  `host-mode offline-hold` entry was missing from the ledger after the
  outage in 2 of 3 full-suite runs on 2026-09-24. The subtest has not
  reproduced in the 6 full suites since then, and the cause is **not
  established**. Diagnostics were added so the next failure explains itself:
  the host journals the per-observation outcome of every outbox flush, and
  the test prints both hosts' journals. This stays open until it is
  reproduced and explained. It is not written off as noise.

### Product change made from code review (not from A02)

- **Gossip joins are single-flight** (`pkg/node/meshrun.go`). The agent started a
  new join every second, and each can block for up to 10 s, so they piled up on
  a slow host. This change is justified by the code alone. It does **not** settle
  the gossip regression candidate above.

## PV1-S1-A03 · PENDING (AUTHORITATIVE when run)

- **Environment:** an independent Linux VM or machine (Ubuntu or Debian), at
  least 4 vCPU, 8 GB RAM and 40 GB SSD. Not a container on this Mac.
- **Procedure:** `validation/pv1/README.md`. Copy the tree, check
  `dh evidence digest` matches, then run `sh validation/pv1/stage-parity.sh <evidence-dir>`.
- **Must include:** `go test -race -count=20 -run TestGossipMembership ./pkg/mesh`
  as a separate recorded step, to settle the regression candidate.
