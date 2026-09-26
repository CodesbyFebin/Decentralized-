# dh/v1 — the Decentralized.Host protocol

Status: implemented by this repository (Go reference) and, for §2–§8, by the
independent Python implementation in `conformance/python`. Test vectors:
`conformance/vectors/dh-v1.json` (see [conformance.md](conformance.md)).

The key words MUST, MUST NOT, SHOULD and MAY are used as in RFC 2119.

## 1. Model

A cluster is a set of **sovereign hosts** coordinated by a **control plane**.
Three kinds of state are kept apart everywhere:

| State | Who writes it | Where it lives |
|---|---|---|
| **Desired** | operators, through the control plane | replicated control-plane log, signed assignments |
| **Admitted** | each host, by its own local policy | the host's hash-chained journal |
| **Observed** | each host, by measurement | signed observations |

The control plane proposes; a host decides. A host never runs work that its own
policy refuses, and never treats the absence of information as good news:
anything not measured is `UNKNOWN`.

Trust flows from one **cluster root key**. Hosts pin it when they join. Every
other key a host accepts (control-plane members, capability delegations, key
rotations, root rotations) is reachable from the pinned root by signatures.

## 2. Canonical JSON

Every signed or hashed object is serialized canonically. An implementation
MUST produce exactly these bytes and MUST reject inputs that have no canonical
form.

Input:

- MUST be valid UTF-8. A byte order mark is not whitespace and is rejected.
- MUST be exactly one JSON value (RFC 8259), optionally surrounded by the four
  JSON whitespace characters. `NaN`, `Infinity`, single quotes, trailing
  commas and leading zeros are not JSON.
- Objects MUST NOT repeat a key. Keys are compared after unescaping (`"a"` and
  `"\u0061"` are the same key).
- Numbers MUST be integers: no fraction and no exponent (`1.0` and `1e2` are
  rejected even though they are integral), within ±(2^53−1). `-0` is `0`.
- `\uXXXX` escapes MUST NOT encode an unpaired UTF-16 surrogate. A valid
  surrogate pair decodes to one code point.

Output:

- Objects: members sorted by key in ascending UTF-8 byte order (equivalently,
  code point order), `{"k":v,...}`, no whitespace.
- Arrays keep their order.
- Integers in shortest decimal form.
- Strings: `"` → `\"`, `\` → `\\`, U+0008 → `\b`, U+000C → `\f`,
  U+000A → `\n`, U+000D → `\r`, U+0009 → `\t`, any other code point below
  U+0020 → `\u00xx` with **lowercase** hex. Everything else, including `/`,
  U+007F, U+2028, U+2029 and `<>&`, is emitted as raw UTF-8.
- `true`, `false`, `null`.

Transport MAY re-encode JSON (for example, escaping `<` as `\u003c`).
Verifiers therefore canonicalize a received payload before checking a
signature (§4).

Quantities in signed payloads are integers: CPU in millicores, memory and disk
in bytes, durations in milliseconds, timestamps in Unix milliseconds.

## 3. Identity

- Keys are Ed25519 (RFC 8032). A **wire public key** is the raw 32 bytes
  encoded as unpadded base64url (RFC 4648 §5). Padding, the standard alphabet
  and any other length MUST be rejected.
- A **dh1 identifier** is `"dh1"` followed by the first 26 characters of the
  lowercase, unpadded RFC 4648 base32 encoding of BLAKE3-256(raw public key).
  It matches `^dh1[a-z2-7]{26}$`.
- A host's dh1 id comes from its genesis key and never changes. Later keys are
  bound to it by rotation statements (§9.4), so a signer field names the stable
  id while the envelope's `pub` names the key that actually signed.
- Private keys are stored as PKCS#8 PEM with mode 0600 and never leave the host.

## 4. Signed envelopes

```json
{"kind":"assignment","signer":"dh1…","pub":"<wire key>","payload":{…},"sig":"<base64url>"}
```

The signature is Ed25519 over the **signing input**:

```
"decentralized.host/" + kind + "/v1\n" + canonical(payload)
```

`kind` is the signing domain, so a signature made for one kind never verifies
as another. `sig` is the 64-byte signature as unpadded base64url.

To verify an envelope against an expected kind, in this order:

1. `kind` MUST equal the expected kind → else `kind-mismatch`.
2. The payload MUST have a canonical form → else `not-canonical`.
3. `pub` MUST decode to 32 bytes → else `bad-key`.
4. `sig` MUST decode to 64 bytes and verify over the signing input built from
   the *canonicalized* payload → else `bad-signature`.
5. Key binding, one of:
   - **self-certifying** (enrollment): `signer` MUST equal dh1(`pub`);
   - **allowed set**: `pub` MUST be one of the keys the verifier holds for
     `signer` (for example, a host's unrevoked key records, the roster key of a
     control-plane member, or the pinned root)
   → else `signer-key`.

The error codes are protocol values. The console and the conformance suite
show them verbatim.

Signing domains used by dh/v1:

| kind | signed by | payload |
|---|---|---|
| `enroll` | joining host (self-certifying) | `Enroll` |
| `request` | host | `{node, ts, op, arg, lastIndex}` — authenticated reads |
| `bundle` | control-plane member in the roster | `Bundle` |
| `assignment` | control-plane member in the roster | `Assignment` |
| `observation` | host | `Observation` |
| `roster` | cluster root | `Roster` |
| `capability` | root or delegate | capability block (§6) |
| `rotation` | new key, plus `oldSig` by the old key; or the old root, for root rotation | `Rotation` |
| `wg-binding` | host | `WGBinding` |
| `audit-checkpoint` | control-plane member or host | `{ledger, seq, hash, ts}` |
| `host-ledger`, `facts` | host | ledger heads, facts |
| `replica-evidence`, `repair-evidence` | host | storage evidence (§7) |
| `cert-evidence`, `route-evidence` | edge host | TLS and routing evidence |
| `artifact-attestation` | publisher key | `{digest, name, ts}` |
| `federation-offer`, `federation-agreement`, `federation-placement`, `federation-status`, `federation-revocation` | cluster roots and members | §10 |
| `cp-backup`, `export` | control-plane member | backups and exports |
| `chaos-report` | chaos runner key | signed scenario reports |

**Digests.** `digest(v)` = `"b3:" + hex(BLAKE3-256(canonical(v)))`. An
envelope's digest is the digest of the envelope object itself; ledger entries
use it as evidence references. A **domain hash**
`H_d(v)` = `"b3:" + hex(BLAKE3-256("decentralized.host/" + d + "/v1\n" + canonical(v)))`.

## 5. Audit ledgers

The control plane and every host each keep an append-only ledger.

```
entry = {seq, ts, actor, source, action, resource, generation, detail, evidence, prev, hash}
hash  = H_audit(entry without "hash")
```

- `seq` starts at 1 and increases by exactly 1.
- `prev` is the previous entry's `hash`; the first entry's `prev` is the
  string `genesis`.
- `source` separates authoritative ledgers from projections: `control-plane`,
  `host`, `operator`, `federation`, `chaos`.
- `evidence` is the digest of the signed envelope that caused the entry, or `""`.

Verification walks the chain and reports the **first** break by its seq and one
of these reasons (protocol values):

| reason | condition |
|---|---|
| `sequence gap` | `seq ≠ previous seq + 1` |
| `previous-hash mismatch` | `prev ≠ previous hash` |
| `event-hash mismatch` | recomputed hash ≠ `hash` |
| `checkpoint signature invalid` | a checkpoint does not verify for its signer |
| `ledger truncated below signed checkpoint` | a checkpoint names a seq past the end |
| `checkpoint does not match chain` | the checkpoint hash ≠ the entry hash at that seq |

A suffix can be verified from a trusted `(seq, hash)` predecessor.

**Checkpoints.** The head is signed periodically (kind `audit-checkpoint`).
Checkpoints detect truncation and rewriting by anyone who holds the signer's
public key. After a root or member key rotation, historical checkpoints remain
verifiable with the keys of the roster that was current when they were signed.

A host journal is fsynced per entry. If it fails verification, the host refuses
new work (`LEDGER_CORRUPT`) and keeps running work it had already admitted.

## 6. Capabilities

A capability is a chain of 1–16 signed blocks (kind `capability`):

```json
{"caveats":{"actions":[…],"resources":[…],"audience":"","notBefore":0,"expires":0,
            "generation":0,"digest":"","cpuMaxMilli":0,"memMaxBytes":0,"nonce":""},
 "next":"<wire key allowed to sign the next block, or \"\" = sealed>",
 "note":"…"}
```

Wire form: `"dhcap1." + base64url_nopad(canonical([block envelope, …]))`.

A request `{action, resource, audience, now, generation, digest, cpuMilli, memBytes}`
is authorized iff, for every block *i* in order:

1. Block 0 verifies against a trusted root key. Block *i* > 0 verifies against
   block *i−1*'s `next`, which MUST be non-empty. A block following a sealed
   block fails at the index of that following block.
2. Block *i*'s caveats hold:
   - `actions` contains `"*"`, the exact action, or a `"p.*"` whose prefix
     `"p."` starts the action;
   - `resources` contains `"*"`, the exact resource, a Go `path.Match`
     pattern that matches it (`*` and `?` never match `/`, `[…]` classes with
     `^` negation, and `\` escapes; a malformed pattern matches nothing), or a
     pattern ending in `/*` whose prefix starts the resource;
   - non-zero or non-empty `audience`, `generation` and `digest` values must be
     equal to the request's;
   - `notBefore ≤ now` and `now < expires` when set;
   - `cpuMilli ≤ cpuMaxMilli` and `memBytes ≤ memMaxBytes` when set.

Every block must hold, so a later block can only narrow authority: attenuation
is structural. Verification reports the failing block index, or −1 for a
malformed token. It needs only the root public key, which is why a host can
check an assignment while the control plane is offline.

Uses:

- **Member delegation.** The root delegates `workload.*` on `app/*` to each
  control-plane member key (recorded in the roster).
- **Assignment.** A member attenuates to one host (`audience`), one
  generation, one digest and resource ceilings. It is embedded in the
  assignment.
- **Operator access.** A bearer token for actions `api.read`, `api.write`
  and `api.admin` on `cluster/<name>`.
- **Join.** A single-use `node.join` token with a `nonce`.

## 7. Storage

**Content addressing.** An object id is `"b3:" + hex(BLAKE3-256(bytes))`.
Readers MUST verify the hash of every object they read, locally or from a peer.
Objects that fail are quarantined, never served.

**FastCDC.** Minimum 16 KiB, average 64 KiB, maximum 256 KiB, normalized
chunking level 2 with a Gear rolling hash:

```
gear[i] = uint64_le(BLAKE3("decentralized.host/fastcdc-gear/v1" || byte(i))[0:8])
cut(data):
  n = len(data); if n ≤ MIN: return n
  n = min(n, MAX); normal = min(AVG, n); h = 0
  for i in [MIN, normal):  h = (h<<1) + gear[data[i]]  (mod 2^64); if h & MASK_S == 0: return i+1
  for i in [normal, n):    h = (h<<1) + gear[data[i]]  (mod 2^64); if h & MASK_L == 0: return i+1
  return n
MASK_S = 0xFFFFC00000000000   (top 18 bits: harder to cut before the average)
MASK_L = 0xFFFC000000000000   (top 14 bits: easier to cut after it)
```

The masks test the top bits, which depend on the last 64 input bytes.

**Merkle root** over a set of ids: de-duplicate, sort ascending, then
`leaf = BLAKE3(0x00 || id)` (the id as its ASCII string),
`node = BLAKE3(0x01 || left || right)`. An odd trailing node is promoted
unchanged. The empty set hashes to `BLAKE3(0x02)`. The result is `"b3:" + hex`.
For anti-entropy, ids are split into 256 buckets by the first byte of their
hash, and replicas compare bucket roots before listing contents.

**Snapshots.** A snapshot manifest `{volumeId, parent, ts, files:[{path, mode, size, chunks}]}`
has id `H_snapshot(manifest)`. A snapshot is **committed** only when a quorum
of 2 replicas has sent `replica-evidence` stating it holds every chunk,
verified. A host restores only committed snapshots. Writes are durable per
object (fsync) with a full barrier at commit points.

**Repair evidence** lists every object moved: source host, previous state
(`missing` or `corrupt`), result and whether it verified.

## 8. Artifacts

A process artifact is addressed by the BLAKE3 digest of the whole file. The
control plane stores its chunks and an `ArtifactManifest {digest, name, bytes, chunks}`
in the CAS. Hosts fetch chunks from mesh peers first and the control plane
last, verify every chunk and the assembled file, and only then write it
executable. Docker images MUST be pinned by `sha256:` digest. When
`requireImageSignature` is set, a host runs only artifacts attested
(`artifact-attestation`) by a key in its own trusted publisher set.

## 9. Host ↔ control plane

### 9.1 Joining

`dh node invite` produces `"dhjoin1." + base64url_nopad(json({cluster, root, rootCaCert, endpoints, tls, capability}))`.
The token pins the root key and root CA, and carries a single-use `node.join`
capability. Members serve their API over TLS with root-issued certificates,
so hosts use HTTPS from their first request. A member that has no
credentials yet serves a throwaway certificate, which the operator pins by
fingerprint while delivering its credentials. The host generates its identity locally and sends an `enroll`
envelope (self-certifying) that carries a summary of its own sovereign
policy. An operator approves the host. The control plane cannot change host
policy; it can only read the summary the host signed.

The control plane refuses an `enroll` envelope when: its self-signature does
not verify; its payload `id`/`pub` differ from the envelope signer; the join
capability is not anchored in the cluster root, is expired, or names an
invite nonce that is unknown, already used, or revoked by the operator; or,
for a host it already knows, the envelope's `ts` is not newer than the
enrollment it has accepted (replay). Every refusal is recorded in the
replicated rejection log.

The possession proof is the host's self-signature over a payload that
contains the single-use, root-signed join nonce; there is no separate
server-issued challenge. The nonce is the challenge, issued by the operator
out of band, and it can be answered once.

### 9.1.1 Host facts

Every observation carries `facts`: values the host measured when it built
the observation (`memBytes` from the OS, CPU model and physical cores, swap,
block devices, GPUs found through sysfs or `nvidia-smi`, the filesystem that
holds the agent's data, uptime). A value the host could not measure is left
zero or empty and named in `facts.unknown`. Declared capacity (`--cpu`,
`--mem`) is carried by the `enroll` payload and is never reported as a fact.
Agents built before `facts.unknown` existed put the declared memory into
`facts.memBytes`; consumers must treat `memBytes` as measured only when
`unknown` is present.

### 9.2 Roster and bundles

The **roster** (kind `roster`, signed by the root) lists control-plane
members: `{id, pub, apiAddr, raftAddr, delegation}`. It is the only way a
member key becomes trusted.

A host fetches its **bundle** with a signed `request` (`op: "bundle"`) and
accepts it only if all of these hold:

1. `root` equals the pinned root. Otherwise a `rootRotation` statement signed
   by the pinned root MUST connect the two (`oldPub` = pinned, `newPub` =
   bundle root), and the host re-pins.
2. `cluster` matches.
3. The embedded roster verifies against the root.
4. The bundle signer is a roster member and the bundle verifies with that
   member's key.
5. `node.id` is this host.
6. `stateIndex` ≥ the highest index already accepted (rollback protection
   across leader changes and restores).

A bundle is **fresh** while `now − issued ≤ freshWindow` (host policy,
default 10 s) and **skewed** when `issued − now > maxClockSkew` (default
30 s). Only a verified leader issues bundles.

### 9.3 Admission

Each assignment in a trusted bundle is evaluated by the host's policy. The
checks run in a fixed order and the first failure decides:

| # | check | refusal code |
|---|---|---|
| 1 | host ledger verifies | `LEDGER_CORRUPT` |
| 2 | bundle trusted (§9.2) | `UNTRUSTED_PLANE` |
| 3 | assignment signed by a roster member | `ASSIGNMENT_SIGNATURE` |
| 4 | assignment names this host | `WRONG_HOST` |
| 5 | generation ≥ admitted generation | `STALE_GENERATION` |
| 6 | clock skew within limit | `CLOCK_SKEW` |
| 7 | capability chain authorizes `workload.admit` on `app/<id>` for this host, generation, digest and resources | `CAPABILITY` |
| — | `desired: stopped` → a signed stop is accepted | `STOP` |
| 8 | host approval not revoked | `HOST_REVOKED` |
| 9 | control plane not frozen | `CONTROL_PLANE_FROZEN` |
| 10 | bundle fresh | `CONTROL_PLANE_STALE` |
| 11 | trust tier accepted | `POLICY_TIER` |
| 12 | runtime allowed | `POLICY_RUNTIME` |
| 13 | image digest-pinned | `POLICY_DIGEST` |
| 14 | artifact attested (if required) | `POLICY_ARTIFACT_SIGNATURE` |
| 15 | federated work accepted (if federated) | `POLICY_FEDERATED` |
| 16–18 | workload, CPU and memory caps | `POLICY_WORKLOAD_CAP`, `POLICY_CPU_CAP`, `POLICY_MEMORY_CAP` |

**Hold semantics.** For checks 1–3, 6–10: if the same generation is already
admitted and running, the host **holds** it (keeps it running, starts nothing
new) instead of stopping it. A stale, frozen, untrusted or unreachable control
plane can never start new work and never causes admitted work to stop, unless
the host's own policy says `offlineAdmission: stop`. Every decision, with its
checks, is written to the host journal and reported in observations.

### 9.4 Observations and key rotation

Observations (kind `observation`) carry a per-host `seq`. The control plane
rejects any seq at or below the last accepted one (replay) and any envelope
not signed by a current key of that host. While the control plane is
unreachable, observations are queued and sent later with `buffered: true`.

A **rotation** is signed by the new key, with `oldSig` from the old key over
the same canonical payload. The old key stays valid until `graceUntil`. A key
can be revoked independently of the host.

## 10. Mesh, edge and federation

**Mesh.** WireGuard (userspace, wireguard-go over a gVisor netstack) with keys
bound to dh1 identities by self-signed `wg-binding` statements
`{node, wgPub, endpoint, meshIp, ts}`. A host only configures peers whose
binding verifies with a key in the bundle. SWIM gossip (memberlist) runs
inside the tunnel. Hosts get addresses in `10.77.0.0/16`, and control-plane
members get `10.77.255.x`. The peer API (`/peer/v1/*`, port 7800 on the mesh
address) serves verified chunks, bucket roots, ledgers, logs and, only if the
host's policy allows it, exec. Revoked hosts are removed from peer sets and
routing.

**Edge.** The edge proxies ingress hosts to healthy endpoints over the mesh.
It ejects endpoints on failed probes or hung responses and drains replaced
generations. TLS comes from ACME (HTTP-01 or DNS-01) or the cluster-local
CA. Certificate states: `REQUESTED PENDING ISSUED RENEWING EXPIRED REVOKED FAILED UNKNOWN`.

**Federation.** There is no global authority. Cluster A grants cluster B an
**agreement** (kind `federation-agreement`) signed by A's root. It names B's
root and limits actions, tiers, runtimes, replicas and resources, with a
validity window. B's members sign **placements** under the agreement,
carrying a delegation chain to B's root. A re-signs the resulting assignments
under its own root, so A's hosts only ever trust A's root. Hosts apply their
own `allowFederated` policy. A can revoke an agreement at any time (kind
`federation-revocation`), and the placements under it are stopped.

## 11. Truthfulness

Every value shown by the console or CLI carries a basis:

| basis | meaning |
|---|---|
| `OBSERVED` | measured by a host and reported in a signed observation |
| `DERIVED` | computed from observed values (with the inputs named) |
| `CONFIGURED` | stated by policy or manifest; not a measurement |
| `PLANNED` | desired state that has not been observed yet |
| `UNKNOWN` | not measured, or the measurement is stale |

`UNKNOWN` is never rendered as healthy, zero or green.

## 12. Versioning

The protocol string is `dh/v1`. Signing domains carry `/v1`. A change that
alters any byte covered by §2–§8 requires new vectors, and a change that
breaks existing vectors requires `dh/v2`.
