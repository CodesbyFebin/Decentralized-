# dhv1 — an independent Python implementation of dh/v1

Written from `docs/protocol/dh-v1.md`, sharing no code with the Go reference:

- `dhv1/blake3.py`: BLAKE3 in pure Python (self-test: `python3 dhv1/blake3.py`)
- `dhv1/ed25519.py`: Ed25519 via `cryptography`, or the RFC 8032 reference
  algorithm when that package is missing or `DH_PY_PURE_ED25519=1`
- `dhv1/core.py`: canonical JSON, dh1 ids, envelopes, audit chain and
  checkpoints, capabilities (including Go `path.Match` semantics), FastCDC and
  Merkle roots
- `adapter.py`: the conformance adapter (line protocol on stdio)

```bash
dh-conformance run -adapter "python3 conformance/python/adapter.py"
DH_PY_PURE_ED25519=1 dh-conformance run -adapter "python3 conformance/python/adapter.py"
```

Both configurations pass all 136 vectors. The suite catches deliberate bugs:
uppercase `\u00XX` escapes, an off-by-one expiry, and a broken Merkle odd-node
rule each produce named failures.
