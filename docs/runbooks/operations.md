# Day-2 operations

Each procedure states what the system guarantees while it runs. When a
command changes desired state, the change is signed intent: nothing counts
as done until hosts observe it (`dh describe app`, `dh get nodes`).

## Applications

| Task | Command | Notes |
|---|---|---|
| Update | edit the manifest, then `dh apply -f app.yaml` | A new generation rolls out within `maxUnavailable`. The new artifact is fetched and verified on each host before the old generation stops. |
| Roll back | re-apply the previous manifest | Generations only increase; a rollback is a new generation with old content. |
| Scale | `dh scale app web --replicas 5` | |
| Explain placement | `dh explain app web` | Scheduler plan, per host and replica, with reasons. |
| Why is a replica not running? | `dh describe app web` | Every admission check on the host, in order, with the refusal code (dh/v1 §9.3). |
| Logs / exec | `dh logs app web --replica 0` · `dh exec app web -- ls` | Over the mesh. Exec needs `allowExec: true` in that host's policy. |
| Delete | `dh delete app web` | Signed stops go to every replica. Volumes and snapshots are kept. |

## Hosts

| Task | Command |
|---|---|
| Maintenance | `dh node drain host-1` … `dh node undrain host-1` |
| Remove a host you trust less now | `dh node revoke host-1 --reason "…"` |
| Replace a key you suspect | `dh node revoke-key host-1 --pub <key> --reason "…"` |
| Rotate a host key (routine) | on the host: `dh-noded rotate-key --data /var/lib/dh-noded --grace 168h` |
| Inspect a host's own ledger | `dh audit host host-1` (fetched over the mesh and verified locally) |
| Local state | on the host: `dh-noded status --data /var/lib/dh-noded` |

**Revocation is not the same as loss.** A revoked host shows
`Identity: REVOKED`, `New admission: BLOCKED`, `Existing runtime: MAY CONTINUE`.
It is cut from the mesh and routing, and its replicas are rescheduled. Under its
own policy it may keep running work it already admitted. If you need that
work gone, stop it on the host.

## Control plane

| Situation | What happens | What to do |
|---|---|---|
| Leader lost | A new leader is elected in about 1–2 s. Hosts keep running work and fetch bundles from any member. | Nothing. Check `dh cp status`. |
| One member lost for good | The cluster tolerates it (3 voters). | `dh cp remove-member --id <id>`, then start a fresh member and run `dh cp add-member` (install §2–3). |
| Quorum lost | No desired-state changes. Hosts enter `offline-hold`: admitted work continues, new work is refused, and observations queue locally. | Restore a quorum of members. If they are gone, restore from backup (below). |
| Planned maintenance | | `dh cp transfer-leadership`, then restart the old leader. |

**Backups.** `dh cp backup --out FILE` writes a signed backup of replicated
state. `dh verify backup FILE` checks it offline: the signature plus the
full audit chain. `--secrets` also includes the local CA key; store such
backups like the root key.

**Total control-plane loss.** Hosts keep running and queue observations.
1. Start new members and bootstrap them (install §2–3), using the **same
   operator home**, which holds the same root key.
2. Run `dh cp restore FILE`. Indexes continue after the backup's index, so hosts
   accept the restored control plane (rollback protection), and restored
   checkpoints verify against the members that signed them.
3. Hosts reconnect, and running work was never stopped. `m5_test.go`
   exercises this path.

**Root key rotation.** `dh cp rotate-root`. The old root signs the rotation.
Hosts re-pin only if the rotation is signed by the root they already pinned,
and assignments are re-signed under the new root.

## Incidents

**Suspected control-plane compromise.** Run `dh freeze`. Hosts hold admitted
work and refuse all new work. A frozen or untrusted plane can never start
anything on a host. Investigate with `dh audit verify` and `dh audit host`.
Unfreeze with `dh unfreeze` or from the console.
A compromised member can sign only within its root delegation, and every host
still applies its own policy to each assignment. That policy covers the
capability chain, generation, attested digest, tiers, runtimes and caps.
Rosters, roots and bundle rollbacks not signed by the pinned root are
rejected. `pkg/node/sovereignty_test.go` runs a host against a control plane
holding a genuine member key.

**Host reports `ledger-corrupt`.** The host found a break in its own
hash-chained journal (`dh describe node` shows the exact break). It keeps
running admitted work, refuses new work, and never appends to the broken chain.
1. Investigate on the host: disk errors, tampering, restores from an old
   image.
2. Stop the agent, then seal the journal:
   ```bash
   dh-noded ledger-seal --data /var/lib/dh-noded --reason "disk replaced after I/O errors; see ticket 123"
   ```
   The corrupt file is kept byte for byte as `journal.jsonl.sealed-<ts>`. A new
   chain starts whose first entry records the sealed file's BLAKE3 hash, the
   break and your reason.
3. Start the agent. It admits work again. The `journal-corruption` chaos
   scenario exercises this path.

**`CLOCK_SKEW`.** The host clock differs from bundle timestamps by more than
`maxClockSkew`. Admitted work is held and new work refused. Fix NTP; admission
resumes on the next bundle.

**Storage `DEGRADED`.** Fewer replicas than required hold verified evidence
for the committed snapshot. Anti-entropy repairs from peers automatically,
and every repaired object is listed with its source in `repair-evidence`
(console → Storage → Repairs). Persistent degradation means a member host is
full (injected or real `ENOSPC`) or lost. Check `dh get volumes` and host
storage facts.

## Federation

```bash
dh federation root                                   # share with the peer that will grant you capacity
# on the granting cluster:
dh federation grant --to alpha --to-root <key> --tiers federated --max-replicas 3 --ttl 720h --out agreement.json
# on the receiving cluster:
dh federation accept agreement.json
# placing: add `placement: {federation: [beta]}` to the manifest, then dh apply
dh federation ls
dh federation revoke <agreement-digest> --reason "…"  # grantor side; placements under it stop
```

Hosts on the granting side only accept federated work if their own policy
says `allowFederated: true`. With TLS, the agreement carries the grantor's
root CA, and peers verify the grantor's API against it.

## Verifying claims

- `dh audit verify`: control-plane ledger chain and checkpoints.
- `dh chaos run --scenario all --submit`: 17 failure scenarios on disposable
  clusters, with signed reports stored in the cluster (console →
  Conformance).
- `dh-conformance run -adapter "…"`: protocol conformance of any
  implementation.
