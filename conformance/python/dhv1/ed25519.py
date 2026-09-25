"""Ed25519 for the Python dh/v1 implementation.

Uses the `cryptography` package when it is installed. Otherwise, or when
DH_PY_PURE_ED25519=1, falls back to the RFC 8032 §6 reference algorithm
(slow and not constant-time: fine for conformance, not for production keys).
"""

import hashlib
import os

try:
    if os.environ.get("DH_PY_PURE_ED25519") == "1":
        raise ImportError("pure Ed25519 forced")
    from cryptography.exceptions import InvalidSignature
    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey, Ed25519PublicKey
    from cryptography.hazmat.primitives.serialization import Encoding, PublicFormat

    BACKEND = "cryptography"

    def public_key(seed: bytes) -> bytes:
        return Ed25519PrivateKey.from_private_bytes(seed).public_key().public_bytes(Encoding.Raw, PublicFormat.Raw)

    def sign(seed: bytes, msg: bytes) -> bytes:
        return Ed25519PrivateKey.from_private_bytes(seed).sign(msg)

    def verify(pub: bytes, msg: bytes, sig: bytes) -> bool:
        try:
            Ed25519PublicKey.from_public_bytes(pub).verify(sig, msg)
            return True
        except (InvalidSignature, ValueError):
            return False

except ImportError:
    BACKEND = "pure-python (RFC 8032)"

    _p = 2 ** 255 - 19
    _L = 2 ** 252 + 27742317777372353535851937790883648493
    _d = -121665 * pow(121666, _p - 2, _p) % _p
    _I = pow(2, (_p - 1) // 4, _p)

    def _sha512_int(b):
        return int.from_bytes(hashlib.sha512(b).digest(), "little")

    def _add(P, Q):
        x1, y1, z1, t1 = P
        x2, y2, z2, t2 = Q
        A = (y1 - x1) * (y2 - x2) % _p
        B = (y1 + x1) * (y2 + x2) % _p
        C = t1 * 2 * _d * t2 % _p
        D = z1 * 2 * z2 % _p
        E, F, G, H = B - A, D - C, D + C, B + A
        return (E * F % _p, G * H % _p, F * G % _p, E * H % _p)

    def _mul(s, P):
        Q = (0, 1, 1, 0)
        while s > 0:
            if s & 1:
                Q = _add(Q, P)
            P = _add(P, P)
            s >>= 1
        return Q

    def _equal(P, Q):
        x1, y1, z1, _ = P
        x2, y2, z2, _ = Q
        return (x1 * z2 - x2 * z1) % _p == 0 and (y1 * z2 - y2 * z1) % _p == 0

    def _recover_x(y, sign_bit):
        if y >= _p:
            return None
        x2 = (y * y - 1) * pow(_d * y * y + 1, _p - 2, _p)
        if x2 == 0:
            return None if sign_bit else 0
        x = pow(x2, (_p + 3) // 8, _p)
        if (x * x - x2) % _p != 0:
            x = x * _I % _p
        if (x * x - x2) % _p != 0:
            return None
        if (x & 1) != sign_bit:
            x = _p - x
        return x

    _gy = 4 * pow(5, _p - 2, _p) % _p
    _gx = _recover_x(_gy, 0)
    _G = (_gx, _gy, 1, _gx * _gy % _p)

    def _compress(P):
        x, y, z, _ = P
        zi = pow(z, _p - 2, _p)
        x, y = x * zi % _p, y * zi % _p
        return int.to_bytes(y | ((x & 1) << 255), 32, "little")

    def _decompress(b):
        if len(b) != 32:
            return None
        y = int.from_bytes(b, "little")
        sign_bit = y >> 255
        y &= (1 << 255) - 1
        x = _recover_x(y, sign_bit)
        if x is None:
            return None
        return (x, y, 1, x * y % _p)

    def _expand(seed):
        h = hashlib.sha512(seed).digest()
        a = int.from_bytes(h[:32], "little")
        a &= (1 << 254) - 8
        a |= 1 << 254
        return a, h[32:]

    def public_key(seed: bytes) -> bytes:
        a, _ = _expand(seed)
        return _compress(_mul(a, _G))

    def sign(seed: bytes, msg: bytes) -> bytes:
        a, prefix = _expand(seed)
        A = _compress(_mul(a, _G))
        r = _sha512_int(prefix + msg) % _L
        R = _compress(_mul(r, _G))
        h = _sha512_int(R + A + msg) % _L
        s = (r + h * a) % _L
        return R + int.to_bytes(s, 32, "little")

    def verify(pub: bytes, msg: bytes, sig: bytes) -> bool:
        if len(pub) != 32 or len(sig) != 64:
            return False
        A = _decompress(pub)
        R = _decompress(sig[:32])
        if A is None or R is None:
            return False
        s = int.from_bytes(sig[32:], "little")
        if s >= _L:
            return False
        h = _sha512_int(sig[:32] + pub + msg) % _L
        return _equal(_mul(s, _G), _add(R, _mul(h, A)))
