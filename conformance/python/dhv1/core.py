"""An independent Python implementation of the dh/v1 protocol primitives.

Written from docs/protocol/dh-v1.md, not translated from the Go code:
canonical JSON, dh1 identities, signed envelopes, the audit hash chain and
checkpoints, attenuable capabilities, FastCDC chunking and Merkle roots.
"""

import base64
import json
import re

from . import blake3, ed25519

# --------------------------------------------------------------- errors


class CanonError(ValueError):
    """The value has no dh/v1 canonical form."""


# ------------------------------------------------------ strict JSON parse


class _Float:
    """A parsed number with a fraction or exponent (never canonical)."""

    def __init__(self, text):
        self.text = text


class _Bad:
    """A parsed NaN/Infinity constant (not JSON)."""


class _DupObject(dict):
    """An object that repeated a key (never canonical)."""


def _pairs(pairs):
    out = {}
    for k, v in pairs:
        if k in out:
            d = _DupObject(out)
            d[k] = v
            return d
        out[k] = v
    return out


_decoder = json.JSONDecoder(object_pairs_hook=_pairs, parse_float=_Float, parse_constant=lambda c: _Bad())


def parse(data):
    """Parses JSON bytes, keeping markers for non-canonical constructs.

    Invalid UTF-8 survives as surrogate escapes and is rejected when the
    value is encoded, so an envelope with a bad payload still parses and
    fails with not-canonical rather than bad-input.
    """
    if isinstance(data, (bytes, bytearray)):
        text = bytes(data).decode("utf-8", errors="surrogateescape")
    else:
        text = data
    if text.startswith("﻿"):
        raise CanonError("byte order mark")
    value, end = _decoder.raw_decode(text, _skip_ws(text, 0))
    if _skip_ws(text, end) != len(text):
        raise CanonError("trailing data after JSON value")
    return value


def _skip_ws(text, i):
    while i < len(text) and text[i] in " \t\n\r":
        i += 1
    return i


# ------------------------------------------------------ canonical encode

MAX_SAFE = 2 ** 53 - 1
_SHORT = {'"': '\\"', "\\": "\\\\", "\b": "\\b", "\f": "\\f", "\n": "\\n", "\r": "\\r", "\t": "\\t"}


def _string(s):
    try:
        s.encode("utf-8")  # rejects lone surrogates (escaped or from bad UTF-8)
    except UnicodeEncodeError:
        raise CanonError("string is not valid Unicode")
    out = ['"']
    for ch in s:
        if ch in _SHORT:
            out.append(_SHORT[ch])
        elif ord(ch) < 0x20:
            out.append("\\u%04x" % ord(ch))
        else:
            out.append(ch)
    out.append('"')
    return "".join(out)


def _encode(v, out):
    if v is None:
        out.append("null")
    elif v is True:
        out.append("true")
    elif v is False:
        out.append("false")
    elif isinstance(v, int):
        if v > MAX_SAFE or v < -MAX_SAFE:
            raise CanonError("integer outside ±(2^53-1)")
        out.append(str(v))
    elif isinstance(v, str):
        out.append(_string(v))
    elif isinstance(v, list):
        out.append("[")
        for i, item in enumerate(v):
            if i:
                out.append(",")
            _encode(item, out)
        out.append("]")
    elif isinstance(v, _DupObject):
        raise CanonError("duplicate object key")
    elif isinstance(v, dict):
        for k in v:
            _string(k)
        out.append("{")
        for i, k in enumerate(sorted(v)):  # code point order == UTF-8 byte order
            if i:
                out.append(",")
            out.append(_string(k))
            out.append(":")
            _encode(v[k], out)
        out.append("}")
    elif isinstance(v, _Float):
        raise CanonError("non-integer number " + v.text)
    else:
        raise CanonError("value is not JSON: %r" % (v,))


def canonical(v) -> bytes:
    out = []
    _encode(v, out)
    return "".join(out).encode("utf-8")


def canonicalize(data: bytes) -> bytes:
    try:
        return canonical(parse(data))
    except (ValueError, RecursionError) as e:  # json.JSONDecodeError is a ValueError
        raise CanonError(str(e))


# ------------------------------------------------------------- encodings

_B64URL = re.compile(r"^[A-Za-z0-9_-]*$")


def b64url(b: bytes) -> str:
    return base64.urlsafe_b64encode(b).decode().rstrip("=")


def unb64url(s: str) -> bytes:
    """Strict unpadded base64url."""
    if not isinstance(s, str) or not _B64URL.match(s) or len(s) % 4 == 1:
        raise ValueError("not unpadded base64url")
    return base64.urlsafe_b64decode(s + "=" * (-len(s) % 4))


def b3(data: bytes) -> str:
    return "b3:" + blake3.hexdigest(data)


# -------------------------------------------------------------- identity


def node_id(pub: bytes) -> str:
    return "dh1" + base64.b32encode(blake3.digest(pub)).decode().lower().rstrip("=")[:26]


def decode_pub(s) -> bytes:
    raw = unb64url(s)
    if len(raw) != 32:
        raise ValueError("public key must be 32 bytes")
    return raw


class Identity:
    def __init__(self, seed: bytes):
        if len(seed) != 32:
            raise ValueError("seed must be 32 bytes")
        self.seed = seed
        self.pub = ed25519.public_key(seed)
        self.pub_str = b64url(self.pub)
        self.id = node_id(self.pub)


# -------------------------------------------------------------- envelopes


def signing_input(kind: str, canonical_payload: bytes) -> bytes:
    return ("decentralized.host/" + kind + "/v1\n").encode() + canonical_payload


def domain_hash(domain: str, value) -> str:
    return b3(signing_input(domain, canonical(value)))


def sign(ident: Identity, kind: str, payload, signer: str = "") -> dict:
    body = canonical(payload)
    sig = ed25519.sign(ident.seed, signing_input(kind, body))
    return {"kind": kind, "signer": signer or ident.id, "pub": ident.pub_str,
            "payload": json.loads(body), "sig": b64url(sig)}


class VerifyError(Exception):
    def __init__(self, code):
        super().__init__(code)
        self.code = code


def _verify_with(env: dict, kind: str) -> bytes:
    if env.get("kind") != kind:
        raise VerifyError("kind-mismatch")
    if "payload" not in env:
        raise VerifyError("not-canonical")
    try:
        body = canonical(env["payload"])
    except CanonError:
        raise VerifyError("not-canonical")
    try:
        pub = decode_pub(env.get("pub"))
    except ValueError:
        raise VerifyError("bad-key")
    try:
        sig = unb64url(env.get("sig"))
    except ValueError:
        raise VerifyError("bad-signature")
    if len(sig) != 64 or not ed25519.verify(pub, signing_input(kind, body), sig):
        raise VerifyError("bad-signature")
    return pub


def verify_self(env: dict, kind: str):
    pub = _verify_with(env, kind)
    if node_id(pub) != env.get("signer"):
        raise VerifyError("signer-key")


def verify_key(env: dict, kind: str, allowed):
    _verify_with(env, kind)
    if env.get("pub") not in allowed:
        raise VerifyError("signer-key")


# ------------------------------------------------------------------ audit

GENESIS = "genesis"
_ENTRY_FIELDS = (("seq", 0), ("ts", 0), ("actor", ""), ("source", ""), ("action", ""), ("resource", ""),
                 ("generation", 0), ("detail", ""), ("evidence", ""), ("prev", ""))


def audit_hash(entry: dict) -> str:
    return domain_hash("audit", {k: entry.get(k, d) for k, d in _ENTRY_FIELDS})


def audit_verify(entries, start_seq=0, start_prev=GENESIS):
    prev_seq, prev = start_seq, start_prev
    for e in entries:
        seq = e.get("seq", 0)
        if seq != prev_seq + 1:
            return {"seq": seq, "reason": "sequence gap"}
        if e.get("prev", "") != prev:
            return {"seq": seq, "reason": "previous-hash mismatch"}
        if audit_hash(e) != e.get("hash", ""):
            return {"seq": seq, "reason": "event-hash mismatch"}
        prev_seq, prev = seq, e["hash"]
    return None


def verify_checkpoints(entries, checkpoints, keys_for):
    by_seq = {e.get("seq", 0): e.get("hash", "") for e in entries}
    max_seq = max([0] + list(by_seq))
    for env in checkpoints:
        try:
            verify_key(env, "audit-checkpoint", keys_for(env.get("signer", "")))
        except VerifyError:
            p = env.get("payload")
            seq = p.get("seq", 0) if isinstance(p, dict) and isinstance(p.get("seq"), int) else 0
            return {"seq": seq, "reason": "checkpoint signature invalid"}
        p = env["payload"]
        seq = p.get("seq", 0)
        if seq > max_seq:
            return {"seq": seq, "reason": "ledger truncated below signed checkpoint"}
        if seq == 0:
            continue
        if by_seq.get(seq, "") != p.get("hash", ""):
            return {"seq": seq, "reason": "checkpoint does not match chain"}
    return None


# ------------------------------------------------------------ capabilities

CAP_PREFIX = "dhcap1."
_CAVEATS = (("actions", None), ("resources", None), ("audience", ""), ("notBefore", 0), ("expires", 0),
            ("generation", 0), ("digest", ""), ("cpuMaxMilli", 0), ("memMaxBytes", 0), ("nonce", ""))


def caveats(c: dict) -> dict:
    return {k: c.get(k, d) for k, d in _CAVEATS}


def cap_encode(blocks) -> str:
    return CAP_PREFIX + b64url(canonical(blocks))


def cap_decode(token: str):
    if not isinstance(token, str) or not token.startswith(CAP_PREFIX):
        raise ValueError("missing prefix")
    blocks = parse(unb64url(token[len(CAP_PREFIX):]))
    if not isinstance(blocks, list) or not 1 <= len(blocks) <= 16 or not all(isinstance(b, dict) for b in blocks):
        raise ValueError("token must have 1..16 blocks")
    return blocks


def go_path_match(pattern: str, name: str) -> bool:
    """Go path.Match: * and ? never match '/', [..] classes with ^ negation,
    backslash escapes. A malformed pattern matches nothing."""
    rx = []
    i, n = 0, len(pattern)
    while i < n:
        c = pattern[i]
        if c == "*":
            rx.append("[^/]*")
            i += 1
        elif c == "?":
            rx.append("[^/]")
            i += 1
        elif c == "\\":
            if i + 1 >= n:
                return False
            rx.append(re.escape(pattern[i + 1]))
            i += 2
        elif c == "[":
            i += 1
            neg = i < n and pattern[i] == "^"
            if neg:
                i += 1
            ranges = []
            while True:
                if i < n and pattern[i] == "]" and ranges:
                    i += 1
                    break

                def esc(j):
                    if j >= n or pattern[j] in "-]":
                        raise ValueError
                    if pattern[j] == "\\":
                        j += 1
                        if j >= n:
                            raise ValueError
                    if j + 1 >= n:
                        raise ValueError
                    return pattern[j], j + 1

                try:
                    lo, i = esc(i)
                    hi = lo
                    if pattern[i] == "-":
                        hi, i = esc(i + 1)
                except (ValueError, IndexError):
                    return False
                ranges.append((lo, hi))
            body = "".join(re.escape(lo) if lo == hi else re.escape(lo) + "-" + re.escape(hi) for lo, hi in ranges)
            rx.append("[" + ("^" if neg else "") + body + "]")
        else:
            rx.append(re.escape(c))
            i += 1
    try:
        return re.fullmatch("".join(rx), name, re.DOTALL) is not None
    except re.error:
        return False


def _match_action(granted, action):
    for g in granted or []:
        if g == "*" or g == action:
            return True
        if g.endswith(".*") and action.startswith(g[:-1]):
            return True
    return False


def _match_resource(granted, resource):
    for g in granted or []:
        if g == "*" or g == resource or go_path_match(g, resource):
            return True
        if g.endswith("/*") and resource.startswith(g[:-1]):
            return True
    return False


def _caveat_ok(c, r) -> bool:
    if not _match_action(c.get("actions"), r["action"]):
        return False
    if not _match_resource(c.get("resources"), r["resource"]):
        return False
    if c.get("audience") and c["audience"] != r["audience"]:
        return False
    if c.get("notBefore") and r["now"] < c["notBefore"]:
        return False
    if c.get("expires") and r["now"] >= c["expires"]:
        return False
    if c.get("generation") and c["generation"] != r["generation"]:
        return False
    if c.get("digest") and c["digest"] != r["digest"]:
        return False
    if c.get("cpuMaxMilli") and r["cpuMilli"] > c["cpuMaxMilli"]:
        return False
    if c.get("memMaxBytes") and r["memBytes"] > c["memMaxBytes"]:
        return False
    return True


def cap_verify(token: str, roots, request: dict):
    """Returns (ok, failing block index or -1)."""
    try:
        blocks = cap_decode(token)
    except (ValueError, CanonError):
        return False, -1
    req = {"action": "", "resource": "", "audience": "", "now": 0, "generation": 0, "digest": "", "cpuMilli": 0, "memBytes": 0}
    req.update(request)
    expect = ""
    for i, env in enumerate(blocks):
        allowed = list(roots) if i == 0 else [expect]
        try:
            verify_key(env, "capability", allowed)
        except VerifyError:
            return False, i
        block = env["payload"]
        if not isinstance(block, dict):
            return False, i
        if not _caveat_ok(block.get("caveats") or {}, req):
            return False, i
        nxt = block.get("next", "")
        if i < len(blocks) - 1 and nxt == "":
            return False, i + 1
        expect = nxt
    return True, -1


# ---------------------------------------------------------------- storage

MIN_CHUNK, AVG_CHUNK, MAX_CHUNK = 16 << 10, 64 << 10, 256 << 10
MASK_S, MASK_L = 0xFFFFC00000000000, 0xFFFC000000000000
_M64 = (1 << 64) - 1
GEAR = [int.from_bytes(blake3.digest(b"decentralized.host/fastcdc-gear/v1" + bytes([i]))[:8], "little") for i in range(256)]


def cut(data, start, end) -> int:
    n = end - start
    if n <= MIN_CHUNK:
        return n
    if n > MAX_CHUNK:
        n = MAX_CHUNK
    normal = min(AVG_CHUNK, n)
    h, gear = 0, GEAR
    i = MIN_CHUNK
    while i < normal:
        h = ((h << 1) + gear[data[start + i]]) & _M64
        if h & MASK_S == 0:
            return i + 1
        i += 1
    while i < n:
        h = ((h << 1) + gear[data[start + i]]) & _M64
        if h & MASK_L == 0:
            return i + 1
        i += 1
    return n


def chunk_all(data: bytes):
    out, off = [], 0
    while off < len(data):
        n = cut(data, off, len(data))
        out.append({"offset": off, "length": n, "id": b3(data[off:off + n])})
        off += n
    return out


def merkle_root(ids) -> str:
    level = [blake3.digest(b"\x00" + i.encode()) for i in sorted(set(ids))]
    if not level:
        return b3(b"\x02")
    while len(level) > 1:
        nxt = []
        for i in range(0, len(level), 2):
            if i + 1 == len(level):
                nxt.append(level[i])
            else:
                nxt.append(blake3.digest(b"\x01" + level[i] + level[i + 1]))
        level = nxt
    return "b3:" + level[0].hex()
