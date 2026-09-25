# 0002 — Canonical JSON with integers only

**Status:** accepted

Signed and hashed objects use a canonical JSON encoding (spec §2): sorted
keys, integers within ±(2^53−1), and no floats. Duplicate keys, lone
surrogates and invalid UTF-8 are rejected. All quantities are integers in
fixed units (millicores, bytes, milliseconds).

**Why.** Two parties must derive identical bytes to verify a signature.
Floats and duplicate keys are the classic sources of disagreement between
JSON implementations. Integer units remove them.

**Consequences.** Verifiers canonicalize received payloads before
verifying, so transport re-encoding (for example `\u003c`) is harmless. The
Go decoder silently repairs invalid UTF-8 and lone surrogates, so the
canonicalizer checks for them explicitly. Conformance vectors pin every rule.
