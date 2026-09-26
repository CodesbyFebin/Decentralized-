# NODE-A01 — sovereign node foundation

Commit under test: `70403c4` · branch `claude/friendly-gauss-kfxoc2`
Evidence: `evidence/NODE-A01-A01` (verify with `dh evidence verify --dir evidence/NODE-A01-A01`)
Script: `validation/node-a01.sh`

**Current outcome: PASS (NODE-A01-A02, VERIFIED).** First attempt, NODE-A01-A01: **FAIL (VERIFIED)**. Every NODE-A01 check passed. The record fails
because two regression steps hit a mesh defect that predates this checkpoint
and reproduces on the base commit (below). No second attempt was run: running
again until it passes would not tell anyone anything new.

## Name

The proposed id was `PV1-NODE-A01`. In this repository, `PV1` is already the
real-infrastructure validation series (`validation/pv1`, PV1-S1…S4:
independent Linux, LAN, partitions, WAN/NAT). A single-machine run cannot
promote a PV1 claim, so this checkpoint is named `NODE-A01`.

## What the gate checks

`tests/integration/node_a01_test.go` runs a real control plane and a real
host process. The host joins with an invite that needs the owner's approval.

| Check | Result |
|---|---|
| Key generated on the host, mode 0600; host waits as `pending` until the owner approves | PASS |
| Private key absent from every control-plane file, `/state`, `/view`, `/audit` and the log (7 encodings: raw, hex, base64, base64url, PEM body) | PASS |
| Used join token presented by another identity | refused: "already used" |
| Join token not signed by the cluster root | refused |
| Expired join token | refused: "expired" |
| Join token revoked by the owner (`dh node invite-revoke`); revoking a consumed invite is refused | refused: "revoked" |
| Enrolment signed by a key other than the enrolling identity | refused: "mismatch" |
| Replay of the host's own accepted enrolment | refused: "not newer" (**fixed**; it used to be accepted and overwrote the record) |
| Facts: `memBytes` equals `/proc/meminfo` MemTotal and differs from the declared `--mem 1Gi`; NAT type listed as unknown | PASS (**fixed**; it used to be the declared value) |
| Approval: `ready`, `ACTIVE`, `FRESH`, `live`; host's own status trusted, fresh, normal | PASS |
| Heartbeat replay; heartbeat signed by a foreign key | refused |
| Control plane frozen (SIGSTOP): host goes `offline-hold`, not fresh; recovers after SIGCONT | PASS |
| Host frozen: plane shows STALE, then `lost`; never FRESH beyond the 10 s window; recovers | PASS |
| Audit chain verifies; one edited entry is named by the verifier; invite, enrol, approve and invite-revoke are all audited | PASS |
| Host ledger edited on disk: the host detects it on restart (`ledger-corrupt`) | PASS |
| Key revoked: heartbeat and bundle request signed with it are refused | PASS |

Host-side command rejection (`pkg/node` `TestCompromisedControlPlaneCannotCommandHost`,
a compromised member signing bundles): unattested artifact, wrong target
host, capability not anchored in the root, swapped image, resources above
the capability, disallowed runtime, disallowed tier, non-member signer,
**expired capability (added)**, stale generation, bundle rollback, impostor
member key, forged roster, revocation. All refused. Unit tests for the
hardware probe run on recorded `/proc` and `/sys` trees.

Mutation check: with the enrolment-replay fix disabled, the replay subtest
fails (`200 re-enrolled`).

## Steps in the record

| Step | Result |
|---|---|
| build, vet, gofmt, unit-race | PASS |
| gossip-repeat (`TestGossipMembership` ×20, `-race`) | **FAIL**: 2 of 20 "join contacted 0" |
| conformance: Go, stdio, Python | PASS (Python from a virtualenv; this sandbox's system `cryptography` is broken) |
| host-commands, node-a01 | PASS |
| integration | **FAIL**: `TestM3TrustAndMesh` timed out waiting for full gossip membership. M1, M2, M5, M7, NODE-A01, rollout, TLS and the D1/D3 regression tests PASS. M4 SKIP: Pebble not installed. |
| Command Centre: source manifest, typecheck, unit (25), no-mock gate | PASS |

## The failing steps: a pre-existing gossip-join defect

PV1-S1-A02 recorded this `TestGossipMembership` failure as a regression
candidate, with the rule "if it reproduces on stable Linux, it is a product
defect". Findings from this session:

- It reproduces on the **base commit** `5bb3780`, without NODE-A01's
  changes: 3 of 20 with `-race`.
- A timing probe of 25 joins with `-race` gave 3 timeouts at 10.002 s, with
  the WireGuard handshake already done within about 120 ms. 3 more joins
  waited about 5 s for a handshake retry.
- Without `-race`: 0 timeouts in 25. 3 joins still waited about 5 s for the
  handshake retry.
- The stall is in memberlist's TCP push-pull over the userspace netstack,
  after the tunnel is up. The occasional lost first handshake initiation
  adds 5 s. Under `-race` overhead, the total exceeds memberlist's 10 s TCP
  timeout.
- `TestM3TrustAndMesh` waits for full gossip membership, and passed in a
  separate full integration run in this session. It is consistent with the
  same defect but was not separately root-caused.

This is a Linux container, not the independent VM that PV1-S1 requires, and
a second 7-process dev cluster was running during the evidence run (stopped
before the timing probe). The defect needs a fix in `pkg/mesh` and its own
regression test before NODE-A01-A02 is attempted.

## Not covered

- A distinct CORDONED state.
- Owner-reserved capacity as a first-class record.
- NIC and NAT discovery.
- Real GPUs.
- Multiple machines.
- Chaos scenarios and the TLS runbook: not re-run.
- A server-issued enrolment challenge. The root-signed single-use join nonce
  is the challenge; see `docs/protocol/dh-v1.md` §9.1.

## Follow-up: the gossip defect is root-caused and fixed

**Cause.** Crossed first handshakes.
- The lower-key side of a pair sends a handshake initiation as soon as the peer is configured (startup keepalive).
- In the failing joins, the higher-key side had data (the join's TCP SYN) and initiated at the same moment.
- wireguard-go consumed the peer's initiation and the response to its own initiation on different goroutines. The resulting session no longer matched the index the other side addressed.
- Every packet in that direction was then dropped until the rekey timer fired at 15 s (REKEY_TIMEOUT + KEEPALIVE_TIMEOUT). A trace with per-device WireGuard logs showed it: `Received invalid response message` on one side, one side's rx counter frozen while the other's tx grew, and recovery at 15.07 s.

**Fix** (`pkg/mesh/device.go`). For a newly added peer, the higher-key side withholds the endpoint for `firstContactGrace` (2 s). Until then it can answer an initiation but not send one, and its data is staged until the handshake completes. After the grace period it gets the endpoint (unless a handshake already taught it one) and initiates at its next retry, so a pair where only it can open the path still connects.

**Regression tests** (`pkg/mesh/handshake_test.go`):
- the higher-key side sends nothing before the lower side's initiation. Without the fix it sent a 148-byte initiation immediately;
- the higher-key side does initiate after the grace period when the lower side cannot reach it.

**Measurements after the fix**
- `TestGossipMembership` ×50 with `-race`: 50/50.
- Join-latency probe, 60 joins with `-race`: 59 under 1 s, 1 at about 5 s, 0 timeouts.
  - The remaining 5 s case is a startup initiation that arrived before the other side had configured the peer, and was resent at REKEY_TIMEOUT.
  - Before the fix, 3 of 25 joins took about 5 s and 3 of 25 timed out.
- Full integration suite: PASS, including M3.

**Second attempt: `evidence/NODE-A01-A02`: PASS (VERIFIED)** at commit `775e1dd`, parent NODE-A01-A01. All 15 steps passed on the first try, including `gossip-repeat` (×20, `-race`) and the full integration suite (M3 and NODE-A01 included). The scope is still a single machine: this promotes nothing for PV1.
