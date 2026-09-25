# Architecture

## Processes

| Binary | Runs on | Holds |
|---|---|---|
| `dh-control` | 1, 3 or 5 control-plane machines | Raft log and snapshots (bbolt), member identity and root-issued certificate, CAS of artifacts, optional Postgres mirror |
| `dh-noded` | every host | host identity, **policy.yaml**, **journal.jsonl** (hash-chained), admitted-state file, CAS, volume data, WireGuard key |
| `dh` | operator machines | cluster root key, root CA, config (`$DH_HOME`) |

## Control plane (`pkg/control`)

- **Replicated FSM.** All desired state is changed by commands that go
  through Raft (hashicorp/raft on a mutual-TLS stream layer with root-issued
  member certificates). The FSM is deterministic: timestamps travel inside
  commands, and nothing reads the clock during apply. The same FSM holds the
  control-plane audit ledger, so every committed change has a ledger entry.
- **Leader duties.** Only a verified leader issues bundles, signs assignments
  and reconciles. Followers forward writes and relay the leader's console view.
  Every bundle carries `stateIndex`, and hosts reject lower indexes, so a
  deposed leader or restored backup cannot roll a host back.
- **Reconciler** (`reconcile.go`). This is where desired state becomes
  scheduler plans (`pkg/scheduler`, deterministic and stability-preserving),
  then signed assignments with per-assignment capabilities, rolling updates
  within `maxUnavailable`, volume placement and restores, edge routing tables,
  and certificate issuance for `tls: local`.
- **Observations** are verified (host key, monotonic `seq`) and cached by the
  leader. They are committed to the log periodically, so followers and backups
  carry them without a Raft write per heartbeat.
- **Views** (`views.go`) project state and evidence for the console and CLI.
  Every value carries its truth basis (§11 of the spec).

## Host agent (`pkg/node`)

A reconcile loop runs every second (`--tick`):

1. **Fetch and verify the bundle.** This checks the pinned root, the root-signed
   roster, the member signature and rollback protection. The bundle is judged
   *trusted*, *fresh* and *skew-free* independently.
2. **Admit.** `policy.Admit` runs for every assignment, with its checks in a fixed
   order and hold semantics (spec §9.3). The decision and its checks are journaled
   when they change.
3. **Converge the runtime.** Artifacts are fetched from mesh peers first and
   the control plane last, and each chunk is BLAKE3-verified. Admitted
   generations then start. Nothing is stopped unless a trusted, fresh bundle
   no longer lists it, a signed stop arrives, or local policy says
   `offlineAdmission: stop`.
4. **Observe and sign.** The agent measures processes and containers, health,
   the mesh, storage and edge, then signs an observation. If the control plane
   is unreachable, observations go to a local outbox and are flushed later
   marked `buffered`.

Runtimes (`pkg/runtime`):
- **process:** executables started with `$PORT`. Adoption across agent restarts
  uses a start token, so PIDs are verified, not assumed.
- **docker:** pinned `sha256:` images, with enforced `--memory`/`--cpus` and
  OOM detection.

## Mesh (`pkg/mesh`, `pkg/peer`)

wireguard-go runs on a gVisor netstack in userspace: no root and no kernel
module. Each host signs a `wg-binding` tying its WireGuard key and endpoint to its
dh1 id, and hosts configure only peers whose bindings verify. memberlist
(SWIM) gossip runs *inside* the tunnel. The peer API (`/peer/v1/*`, port 7800
on mesh addresses) serves verified chunks, bucket roots, ledgers and logs, and
exec only when policy allows it. Each assignment's port is exposed on the mesh
by a forwarder that lives as long as the assignment; restarts only retarget it.

## Storage (`pkg/storage`)

- **CAS.** Content-addressed BLAKE3 objects, verified on every read, with
  corrupt objects quarantined. Writes are durable per object (fsync), with
  full barriers (`F_FULLFSYNC` on macOS) at commit points.
- **Volumes.** Each volume is chunked with FastCDC into snapshots (manifests
  of file chunk lists), taken every `snapshotEveryMs`. A snapshot is
  **committed** once two replicas send signed `replica-evidence` proving they
  hold every chunk. A moved replica restores the last committed snapshot, so
  an interrupted write rolls back cleanly.
- **Anti-entropy.** Replicas compare 256 Merkle bucket roots, fetch only
  differing buckets, and record every repaired object in `repair-evidence`.

## Edge (`pkg/edge`)

An L7 reverse proxy on edge hosts, over the mesh, fed by the bundle's
routing table and by its own probes:
- endpoints are `routing`, `draining`, `ejected` or `pending`;
- hung replicas are ejected on first-byte timeout;
- connections of a replaced generation drain.

TLS certificates come from the cluster-local CA or from ACME (HTTP-01,
DNS-01, wildcards), and their states are reported verbatim.

## Console (`web/dist`)

The console is plain ES modules and CSS with no build step and no
dependencies. It is embedded into `dh-control`, served by every member under
a strict CSP (`default-src 'self'`), and revalidated on every load.
- **Data.** It polls `/api/v1/view` (followers relay the leader's view).
- **Staleness.** It never updates state optimistically. It marks the whole
  view stale when polls fail or stall.
- **Rendering.** Every interpolated string is escaped, because host names and
  reasons come from hosts.
- **Controls.** Keyboard-first, with a command palette and reduced-motion
  support.
- **Development.** Set `DH_CONSOLE_DIR=web/dist` to serve the console from
  disk while working on it.

## Chaos and development clusters (`pkg/devcluster`, `pkg/chaos`)

`devcluster` starts real processes: members, hosts, edges, and optionally
Postgres or TLS. It can kill, restart, pause, partition (through UDP fault
proxies in front of WireGuard sockets) and inject host faults (clock offset,
`ENOSPC`, dropped control-plane connectivity). `chaos` runs 17 scenarios on
disposable clusters under traffic and signs a report for each.

## Wire and storage formats

Everything signed or hashed uses canonical JSON (spec §2): integers only,
sorted keys, and duplicates, lone surrogates and invalid UTF-8 rejected.
Transport uses JSON without HTML escaping, and verifiers canonicalize again
anyway. See [protocol/dh-v1.md](protocol/dh-v1.md).
