# Trust model

## Anchors

| Key | Held by | Can | Cannot |
|---|---|---|---|
| **Cluster root** (Ed25519, also the X.509 root CA) | operator, offline | sign rosters, delegate to members, create invites, sign root rotations, grant federation, attest artifacts | run anything on a host whose policy refuses it |
| **Member key** | each control-plane member | sign bundles and assignments within its root delegation (`workload.*` on `app/*`) | enlarge the roster, change host policy, roll a host back, issue work for a revoked host |
| **Host key** | each host | sign enrollment, observations, `wg-binding`, storage evidence, its own ledger | speak for another host; its signed `seq` cannot be replayed |
| **Session capability** | operator browser or CLI | attenuated bearer token for `api.read` / `api.write` / `api.admin`, with expiry | exceed the caveats of any block in its chain |

## What a host verifies before it runs anything

1. The bundle chains to the **pinned root**, meaning the root from the join
   token or from a rotation that the pinned root signed.
2. The bundle signer is in the **root-signed roster**, and the bundle's state
   index is not older than one already accepted.
3. The assignment is signed by a roster member, names this host, and has a
   generation at least as high as any already admitted.
4. The **capability chain** in the assignment authorizes `workload.admit` for
   this assignment, host, generation, artifact digest and resources, down
   from the root.
5. Its own **policy.yaml**: tiers, runtimes, digest pinning, artifact
   attestation by trusted publishers, federated work, and caps.
6. The artifact bytes hash to the assigned digest.

If any of these fails, nothing new starts. If the same generation is already
running, it is **held**, not stopped. Every decision is written to the
host's own hash-chained ledger.

## Failure and attack cases

| Case | Outcome | Where it is tested |
|---|---|---|
| Control plane down or partitioned | `offline-hold`: admitted work continues, new work refused, observations queued | chaos `cp-total-outage`, `network-partition`; M1 |
| Compromised member (valid member key) | can only sign within its delegation, and every host still applies its own policy. Stale generations, rollbacks, impostor keys and unsigned rosters are refused. | `pkg/node/sovereignty_test.go` |
| Replayed, forged or tampered observation | rejected with a specific reason, recorded, and state unchanged | chaos `replay-forgery`; M1 |
| Stolen host key | `dh node revoke-key` revokes that key; the host's other keys keep working | M3 |
| Revoked host | cut from mesh and routing, replicas rescheduled, new admission blocked | chaos `revoked-host`; M3 |
| Host clock skew | skew detected from bundle timestamps, `CLOCK_SKEW`, admitted work held | chaos `clock-skew` |
| Host ledger tampering | break named, `LEDGER_CORRUPT`, no appends to the broken chain; an operator seal recovers with the break recorded | chaos `journal-corruption` |
| Audit ledger truncation or rewrite | hash chain plus signed checkpoints, verified offline (`dh audit verify`) | `pkg/audit`, M1 |
| Storage corruption | reads verify BLAKE3, bad objects are quarantined and repaired from peers with evidence | M2, chaos `storage-replica-loss`, `disk-full` |
| Root key rotation | hosts re-pin only when the old root signed the rotation | M3 |
| Network observer | member APIs use TLS with root-issued certificates, bootstrap pins a fingerprint, Raft is mutual TLS, host-to-host traffic runs over WireGuard, and federation verifies the grantor CA in the agreement | `tls_test.go`, M7 |

## Deliberate limits

- The root key is the single anchor. Protect it offline. Rotating it is
  supported, but a stolen root can sign a rotation.
- A member with a valid delegation can sign harmful-but-permitted work (for
  example, scaling an app to zero). Host policy bounds *what* runs; it cannot
  judge operator intent. Use `dh freeze` when compromise is suspected.
- The process runtime does not enforce CPU or memory limits (reported in
  admission details). Use Docker for enforcement.
- Session capabilities are bearer tokens. The console keeps them in
  `sessionStorage` and the URL fragment, never in server logs, and they
  expire (`dh console --ttl`).
