#!/usr/bin/env python3
"""dh/v1 conformance adapter for the Python implementation.

Speaks the line protocol in docs/protocol/conformance.md:

    dh-conformance run -adapter "python3 conformance/python/adapter.py"
"""

import base64
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from dhv1 import core  # noqa: E402


class Fail(Exception):
    def __init__(self, code):
        super().__init__(code)
        self.code = code


def _b64(b):
    return base64.b64encode(b).decode()


def _unb64(s):
    return base64.b64decode(s, validate=True)


def _ident(seed_hex):
    try:
        return core.Identity(bytes.fromhex(seed_hex))
    except (ValueError, TypeError):
        raise Fail("bad-input")


def op_canon(i):
    try:
        return {"canonical_b64": _b64(core.canonicalize(_unb64(i["json_b64"])))}
    except core.CanonError:
        raise Fail("reject")


def op_identity(i):
    if i.get("seed"):
        ident = _ident(i["seed"])
        return {"pub": ident.pub_str, "id": ident.id}
    try:
        pub = core.decode_pub(i.get("pub"))
    except ValueError:
        raise Fail("bad-key")
    return {"pub": i["pub"], "id": core.node_id(pub)}


def op_digest(i):
    try:
        return {"hash": core.b3(core.canonical(i["value"]))}
    except core.CanonError:
        raise Fail("reject")


def op_domain_hash(i):
    try:
        return {"hash": core.domain_hash(i["domain"], i["value"])}
    except core.CanonError:
        raise Fail("reject")


def op_sign(i):
    ident = _ident(i["seed"])
    try:
        body = core.canonical(i["payload"])
    except core.CanonError:
        raise Fail("reject")
    return {"envelope": core.sign(ident, i["kind"], i["payload"], i.get("signer", "")),
            "signing_input_b64": _b64(core.signing_input(i["kind"], body))}


def op_verify(i):
    try:
        env = core.parse(_unb64(i["envelope_b64"]))
    except (ValueError, core.CanonError):
        raise Fail("bad-input")
    if not isinstance(env, dict):
        raise Fail("bad-input")
    try:
        if i.get("self"):
            core.verify_self(env, i["kind"])
        else:
            core.verify_key(env, i["kind"], i.get("allowed") or [])
    except core.VerifyError as e:
        return {"valid": False, "error": e.code}
    return {"valid": True, "error": ""}


def op_audit_hash(i):
    return {"hash": core.audit_hash(i)}


def op_audit_verify(i):
    entries = i.get("entries") or []
    br = core.audit_verify(entries, i.get("startSeq", 0), i.get("startPrev", ""))
    if br is None and i.get("checkpoints"):
        keys = i.get("keys") or {}
        br = core.verify_checkpoints(entries, i["checkpoints"], lambda s: keys.get(s) or [])
    return {"break": br}


def op_capability_mint(i):
    blocks = []
    for b in i["blocks"]:
        payload = {"caveats": core.caveats(b.get("caveats") or {}), "next": b.get("next", ""), "note": b.get("note", "")}
        blocks.append(core.sign(_ident(b["seed"]), "capability", payload))
    return {"token": core.cap_encode(blocks)}


def op_capability_verify(i):
    ok, block = core.cap_verify(i["token"], i.get("roots") or [], i.get("request") or {})
    return {"ok": ok, "block": block}


def op_chunk(i):
    return {"chunks": core.chunk_all(_unb64(i["data_b64"]))}


def op_merkle(i):
    return {"root": core.merkle_root(i.get("ids") or [])}


OPS = {
    "canon": op_canon, "identity": op_identity, "digest": op_digest, "domain-hash": op_domain_hash,
    "sign": op_sign, "verify": op_verify, "audit-hash": op_audit_hash, "audit-verify": op_audit_verify,
    "capability-mint": op_capability_mint, "capability-verify": op_capability_verify,
    "chunk": op_chunk, "merkle": op_merkle,
}


def handle(line):
    try:
        req = core.parse(line)
    except (ValueError, core.CanonError):
        return {"id": "", "ok": False, "error": "bad-input"}
    rid = req.get("id", "")
    fn = OPS.get(req.get("op"))
    if fn is None:
        return {"id": rid, "ok": False, "error": "unsupported"}
    try:
        return {"id": rid, "ok": True, "output": fn(req.get("input") or {})}
    except Fail as f:
        return {"id": rid, "ok": False, "error": f.code}
    except (KeyError, TypeError, ValueError) as e:
        return {"id": rid, "ok": False, "error": "bad-input: %s" % e}


def main():
    for line in sys.stdin.buffer:
        if not line.strip():
            continue
        resp = handle(line.rstrip(b"\r\n"))
        sys.stdout.write(json.dumps(resp, ensure_ascii=False, separators=(",", ":")) + "\n")
        sys.stdout.flush()


if __name__ == "__main__":
    main()
