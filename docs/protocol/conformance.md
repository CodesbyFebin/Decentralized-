# dh/v1 conformance

The conformance suite checks that an implementation produces and accepts
exactly the bytes the protocol specifies. It has three parts:

| part | where |
|---|---|
| test vectors | `conformance/vectors/dh-v1.json` (generated; 136 vectors) |
| runner and Go reference adapter | `cmd/dh-conformance`, `pkg/conformance` |
| independent implementation | `conformance/python` (pure-Python BLAKE3; Ed25519 via `cryptography`, or pure RFC 8032 fallback) |

## Running

```bash
go build -o bin/ ./cmd/dh-conformance
./bin/dh-conformance run -self                                          # Go reference, in process
./bin/dh-conformance run -adapter "./bin/dh-conformance adapter"         # Go reference over stdio
./bin/dh-conformance run -adapter "python3 conformance/python/adapter.py" # independent implementation
DH_PY_PURE_ED25519=1 ./bin/dh-conformance run -adapter "python3 conformance/python/adapter.py"
./bin/dh-conformance check      # vectors still match the reference (drift guard)
```

Useful flags for `run`: `-ops canon,verify` selects a subset,
`-report out.json` writes a machine-readable report, and `-allow-skip` lets an
adapter that does not implement some ops still pass on the rest.
`go test ./pkg/conformance` runs the drift guard, the reference and the
Python implementation.

## Adapter protocol

An adapter is any executable. It reads one JSON request per line on stdin and
writes one JSON response per line on stdout, in order:

```
→ {"id":"verify/wrong-kind","op":"verify","input":{…}}
← {"id":"verify/wrong-kind","ok":true,"output":{"valid":false,"error":"kind-mismatch"}}
← {"id":"canon/reject-float","ok":false,"error":"reject"}
← {"id":"x","ok":false,"error":"unsupported"}        (op not implemented → skipped)
```

The runner compares `canonical({"output": output})` or
`canonical({"error": code})` with the vector's `expect`, so key order and
whitespace in responses do not matter. Binary data is standard base64 with
padding in fields ending `_b64`. Wire keys and signatures are unpadded
base64url, as in the protocol.

| op | input | output |
|---|---|---|
| `canon` | `json_b64` | `canonical_b64`, or error `reject` |
| `identity` | `seed` (hex, 32 bytes) or `pub` | `pub`, `id`; error `bad-key` |
| `digest` | `value` | `hash`; error `reject` |
| `domain-hash` | `domain`, `value` | `hash`; error `reject` |
| `sign` | `seed`, `signer` (`""` = own id), `kind`, `payload` | `envelope`, `signing_input_b64` |
| `verify` | `envelope_b64`, `kind`, `self`, `allowed` | `valid`, `error` (§4 codes or `""`) |
| `audit-hash` | a ledger entry | `hash` |
| `audit-verify` | `entries`, `startSeq`, `startPrev`, `checkpoints`, `keys` | `break`: `null` or `{seq, reason}` |
| `capability-mint` | `blocks: [{seed, caveats, next, note}]` | `token` |
| `capability-verify` | `token`, `roots`, `request` | `ok`, `block` |
| `chunk` | `data_b64` | `chunks: [{offset, length, id}]` |
| `merkle` | `ids` | `root` |

Vectors for `chunk` store `data_gen: {alg, seed, length}` instead of the
bytes, and the runner expands it to `data_b64` before sending:

- `xorshift64*`: `x ^= x>>12; x ^= x<<25; x ^= x>>27`, then emit
  `x·0x2545F4914F6CDD1D mod 2^64` as 8 little-endian bytes, truncated to `length`;
- `zero`: `length` zero bytes;
- `pattern`: byte *i* is `i mod 251`.

Test identities use seed *n* = `BLAKE3("decentralized.host/conformance-seed/v1" || n)`.

## What the vectors cover

- **canon (42):** key order by UTF-8 bytes; every escape rule; raw U+2028
  and `<>&`; surrogate pairs; `-0`; the ±(2^53−1) boundary; and rejection of
  floats, exponents, out-of-range and huge integers, duplicate keys (also via
  escapes and nesting), trailing data, NaN/Infinity, leading zeros, lone and
  reversed surrogates, invalid and overlong UTF-8, a BOM, raw controls and an
  empty document.
- **identity (9):** seeds, arbitrary keys, and malformed, padded or
  standard-alphabet keys.
- **digest / domain-hash (9)**, including rejection of non-canonical values.
- **sign (5)**, including HTML characters, Unicode and a rotated signer.
- **verify (14):**
  - valid envelopes, with self-certifying keys, allowed sets and rotated keys;
  - one of each error code;
  - transport re-encoding (reordered keys, `\u003c`), which must still verify;
  - duplicate and float payloads (`not-canonical`);
  - truncated or padded signatures, and key substitution.
- **audit (14):** hashes, intact chains and suffixes, and every break reason,
  including all four checkpoint failures.
- **capability (29):** minting, and each caveat at its boundary
  (`now == expires` is expired). Also:
  - narrowing across blocks;
  - wildcard, prefix and `path.Match` resource rules;
  - an untrusted root, a wrong delegate and a sealed chain;
  - tampered encodings and blocks, and the 16-block limit.
- **chunk (8):** empty input, 1 byte, exactly and just over the minimum,
  random data, all-zero data (cut at the maximum) and a repeating pattern.
- **merkle (6):** the empty set, odd promotion, and de-duplication with
  sorting.

## Changing the protocol

Vectors are generated from the Go reference implementation, and
`TestVectorsMatchReference` fails if the committed file is stale. A change
that alters vector bytes is a protocol change: regenerate with
`dh-conformance gen`, review the diff, update `dh-v1.md`, update the Python
implementation to match, and bump `SuiteVersion`. Anything that breaks an
existing vector's meaning needs `dh/v2`.
