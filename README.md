# Decentralized.Host

Self-hosted infrastructure where **hosts stay sovereign**. A replicated
control plane proposes work as signed intent. Each host checks every
assignment against its own local policy before anything runs, keeps its own
hash-chained ledger, and reports what it actually observed in signed
messages. Desired, admitted and observed state stay separate everywhere, and
anything that was not measured is shown as `UNKNOWN`.

There is no SaaS dependency, no telemetry and no phone-home. The operator
console is served by the control plane itself and loads nothing from third
parties.

```
operator (dh CLI / console) ──► control plane (Raft, 3 or 5 members) ──signed assignments──► hosts
                                         ▲                                                   │ admit under local policy
                                         └──────────────── signed observations ◄─────────────┘ run · observe · journal
                              hosts ◄── WireGuard mesh + gossip, peer-to-peer storage and artifacts ──► hosts
```

## Status

Each milestone is exercised by a real multi-process test: real processes,
real sockets, a real WireGuard mesh, and kill -9 where the test calls for it.

| Milestone | What works | Evidence |
|---|---|---|
| **M1** Sovereign runtime | Ed25519 identities, signed assignments and observations, and local admission with hold semantics. Also replay, forgery and tamper rejection, freeze, revocation, and audit verification. | `tests/integration/m1_test.go`, `pkg/node/sovereignty_test.go` |
| **M2** Sovereign storage | BLAKE3 CAS, FastCDC, Merkle anti-entropy, snapshots committed at quorum 2, and restore after host loss. An interrupted write rolls back to the last committed snapshot; corruption is repaired from peers. | `m2_test.go` |
| **M3** Trust and mesh | Root-signed roster, capability chains, userspace WireGuard with signed key bindings, SWIM gossip, host key rotation and revocation, and root key rotation. | `m3_test.go` |
| **M4** Edge and TLS | Health-gated L7 proxy over the mesh, draining, and ejection of hung replicas. TLS from ACME HTTP-01, DNS-01 and wildcards, tested against Pebble, or from the cluster-local CA. | `m4_test.go`, `pebble_test.go` |
| **M5** HA control plane | Raft with mutual TLS, a leader-only bundle issuer and rollback protection. Leader loss causes no workload interruption. Signed backups, and restore after total control-plane loss. | `m5_test.go` (failover about 1.2 s, 0 failed requests) |
| **M6** Chaos | 17 scenarios on disposable clusters with invariants checked under traffic, signed reports, and a randomized soak. | `dh chaos run` (17/17 PASS) |
| **M7** Federation | Root-signed agreements between independent clusters, delegated placements, grantor re-signing, and revocation. | `m7_test.go` |
| Transport security | Member APIs use TLS from their first start. Bootstrap pins the member's certificate by fingerprint, members then serve root-issued certificates, hosts join over HTTPS, and federation verifies the grantor CA. | `tls_test.go`, `m7_test.go` |
| **M8** Conformance | The `dh/v1` spec, 136 test vectors, an adapter-protocol runner, and an independent Python implementation. | `dh-conformance`, `go test ./pkg/conformance` |

### Known limitations

These are deliberate and are reported by the system itself.

- The **process runtime does not enforce CPU or memory limits**. It says so in
  admission details. Use the `docker` runtime for enforced limits (`--memory`, `--cpus`).
- **Erasure coding** is not implemented. Volumes are replicated.
- **gVisor and Firecracker** are detected and reported but not wired as
  runtimes. **HTTP/3** is not implemented.
- The mesh is userspace WireGuard (wireguard-go on a gVisor netstack): no root
  and no kernel interface. Workloads are reached through per-assignment mesh
  forwarders, not through a routed IP per workload.
- `dh dev up` clusters bind to loopback and serve the API without TLS unless you
  pass `--tls`. `dh init` enables TLS by default for real installations.

## Quick start

Requirements: Go 1.26+. Optional: Docker (container runtime and `oom` chaos),
Python 3 (independent conformance implementation), and Postgres (evidence mirror).

```bash
make build                       # bin/dh, bin/dh-control, bin/dh-noded, bin/dh-conformance, bin/dh-beacon
./bin/dh dev up --dir ./devcluster
```

`dev up` starts 3 control-plane members, 3 hosts and 1 edge as real local
processes. It pushes the `dh-beacon` sample artifact, deploys a 3-replica app,
and prints a console URL with a session capability in the URL fragment:

```bash
export DH_HOME=./devcluster/operator
./bin/dh get apps
./bin/dh describe app web          # desired / admitted / observed, and the admission checks of every replica
./bin/dh mesh peers                # WireGuard handshakes and measured RTT
./bin/dh audit verify              # fetch the ledger and verify the chain and checkpoints locally
./bin/dh chaos run --scenario leader-crash
./bin/dh dev down --dir ./devcluster
```

For a real installation, see [docs/runbooks/install.md](docs/runbooks/install.md).

## Layout

| Path | What |
|---|---|
| `cmd/dh` | operator CLI (cluster init, apply, nodes, control plane, audit, federation, chaos, dev clusters) |
| `cmd/dh-control` | control-plane member (Raft, API, reconciler, console) |
| `cmd/dh-noded` | host agent (admission, runtimes, journal, mesh, storage, edge) |
| `cmd/dh-conformance` | vector generator and conformance runner |
| `cmd/dh-beacon` | sample workload used by tests and `dev up` |
| `pkg/canon`, `envelope`, `identity`, `audit`, `capability` | protocol core (`docs/protocol/dh-v1.md` §2–§6) |
| `pkg/control` | control plane: FSM, API, views, reconciler, federation |
| `pkg/node`, `policy`, `runtime` | host agent, sovereign policy, process and Docker runtimes |
| `pkg/storage`, `mesh`, `edge`, `peer`, `pki` | CAS/FastCDC/Merkle, WireGuard and gossip, L7 edge and ACME, peer API, certificates |
| `pkg/chaos`, `devcluster` | chaos scenarios and the real multi-process cluster harness |
| `web/dist` | operator console (plain ES modules, no build step, embedded into `dh-control`) |
| `conformance/` | vectors and the independent Python implementation |
| `docs/` | protocol, architecture, trust model, runbooks and decisions |

## Tests

```bash
make test          # unit tests, including the Python conformance implementation
make race          # unit tests with the race detector
make integration   # M1–M7 multi-process tests (about 5 minutes; needs Pebble for M4 ACME: make tools)
make conformance   # vectors against Go (in process and over stdio) and Python
make chaos         # all 17 chaos scenarios
```

## Documentation

- [Protocol dh/v1](docs/protocol/dh-v1.md) and [conformance](docs/protocol/conformance.md)
- [Architecture](docs/architecture.md) · [Trust model](docs/trust-model.md)
- [Runbooks](docs/runbooks/) · [Decisions](docs/decisions/)
- [Production blueprint](docs/BLUEPRINT.md): status of every milestone against its exit criteria, and the remaining hardening plan
