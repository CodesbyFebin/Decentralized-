"""Pure-Python BLAKE3 (hash mode, 32-byte output).

Written from the BLAKE3 specification for the dh/v1 conformance suite, so
this implementation shares no code with the Go reference. It favours
clarity over speed but inlines the round function, which is enough for the
suite's ~2 MB of input.
"""

import struct

_IV = (0x6A09E667, 0xBB67AE85, 0x3C6EF372, 0xA54FF53A,
       0x510E527F, 0x9B05688C, 0x1F83D9AB, 0x5BE0CD19)
_PERM = (2, 6, 3, 10, 7, 0, 4, 13, 1, 11, 12, 5, 9, 14, 15, 8)
CHUNK_START, CHUNK_END, PARENT, ROOT = 1, 2, 4, 8
BLOCK_LEN, CHUNK_LEN = 64, 1024
_M = 0xFFFFFFFF

# Message word order for each of the 7 rounds.
_SCHEDULE = []
_order = list(range(16))
for _ in range(7):
    _SCHEDULE.append(tuple(_order))
    _order = [_order[p] for p in _PERM]


def _compress(cv, m, counter, block_len, flags):
    """Returns the 16-word output of the compression function."""
    s0, s1, s2, s3, s4, s5, s6, s7 = cv
    s8, s9, s10, s11 = _IV[0], _IV[1], _IV[2], _IV[3]
    s12 = counter & _M
    s13 = (counter >> 32) & _M
    s14 = block_len
    s15 = flags
    for o in _SCHEDULE:
        # columns
        s0 = (s0 + s4 + m[o[0]]) & _M; s12 ^= s0; s12 = ((s12 >> 16) | (s12 << 16)) & _M
        s8 = (s8 + s12) & _M; s4 ^= s8; s4 = ((s4 >> 12) | (s4 << 20)) & _M
        s0 = (s0 + s4 + m[o[1]]) & _M; s12 ^= s0; s12 = ((s12 >> 8) | (s12 << 24)) & _M
        s8 = (s8 + s12) & _M; s4 ^= s8; s4 = ((s4 >> 7) | (s4 << 25)) & _M

        s1 = (s1 + s5 + m[o[2]]) & _M; s13 ^= s1; s13 = ((s13 >> 16) | (s13 << 16)) & _M
        s9 = (s9 + s13) & _M; s5 ^= s9; s5 = ((s5 >> 12) | (s5 << 20)) & _M
        s1 = (s1 + s5 + m[o[3]]) & _M; s13 ^= s1; s13 = ((s13 >> 8) | (s13 << 24)) & _M
        s9 = (s9 + s13) & _M; s5 ^= s9; s5 = ((s5 >> 7) | (s5 << 25)) & _M

        s2 = (s2 + s6 + m[o[4]]) & _M; s14 ^= s2; s14 = ((s14 >> 16) | (s14 << 16)) & _M
        s10 = (s10 + s14) & _M; s6 ^= s10; s6 = ((s6 >> 12) | (s6 << 20)) & _M
        s2 = (s2 + s6 + m[o[5]]) & _M; s14 ^= s2; s14 = ((s14 >> 8) | (s14 << 24)) & _M
        s10 = (s10 + s14) & _M; s6 ^= s10; s6 = ((s6 >> 7) | (s6 << 25)) & _M

        s3 = (s3 + s7 + m[o[6]]) & _M; s15 ^= s3; s15 = ((s15 >> 16) | (s15 << 16)) & _M
        s11 = (s11 + s15) & _M; s7 ^= s11; s7 = ((s7 >> 12) | (s7 << 20)) & _M
        s3 = (s3 + s7 + m[o[7]]) & _M; s15 ^= s3; s15 = ((s15 >> 8) | (s15 << 24)) & _M
        s11 = (s11 + s15) & _M; s7 ^= s11; s7 = ((s7 >> 7) | (s7 << 25)) & _M
        # diagonals
        s0 = (s0 + s5 + m[o[8]]) & _M; s15 ^= s0; s15 = ((s15 >> 16) | (s15 << 16)) & _M
        s10 = (s10 + s15) & _M; s5 ^= s10; s5 = ((s5 >> 12) | (s5 << 20)) & _M
        s0 = (s0 + s5 + m[o[9]]) & _M; s15 ^= s0; s15 = ((s15 >> 8) | (s15 << 24)) & _M
        s10 = (s10 + s15) & _M; s5 ^= s10; s5 = ((s5 >> 7) | (s5 << 25)) & _M

        s1 = (s1 + s6 + m[o[10]]) & _M; s12 ^= s1; s12 = ((s12 >> 16) | (s12 << 16)) & _M
        s11 = (s11 + s12) & _M; s6 ^= s11; s6 = ((s6 >> 12) | (s6 << 20)) & _M
        s1 = (s1 + s6 + m[o[11]]) & _M; s12 ^= s1; s12 = ((s12 >> 8) | (s12 << 24)) & _M
        s11 = (s11 + s12) & _M; s6 ^= s11; s6 = ((s6 >> 7) | (s6 << 25)) & _M

        s2 = (s2 + s7 + m[o[12]]) & _M; s13 ^= s2; s13 = ((s13 >> 16) | (s13 << 16)) & _M
        s8 = (s8 + s13) & _M; s7 ^= s8; s7 = ((s7 >> 12) | (s7 << 20)) & _M
        s2 = (s2 + s7 + m[o[13]]) & _M; s13 ^= s2; s13 = ((s13 >> 8) | (s13 << 24)) & _M
        s8 = (s8 + s13) & _M; s7 ^= s8; s7 = ((s7 >> 7) | (s7 << 25)) & _M

        s3 = (s3 + s4 + m[o[14]]) & _M; s14 ^= s3; s14 = ((s14 >> 16) | (s14 << 16)) & _M
        s9 = (s9 + s14) & _M; s4 ^= s9; s4 = ((s4 >> 12) | (s4 << 20)) & _M
        s3 = (s3 + s4 + m[o[15]]) & _M; s14 ^= s3; s14 = ((s14 >> 8) | (s14 << 24)) & _M
        s9 = (s9 + s14) & _M; s4 ^= s9; s4 = ((s4 >> 7) | (s4 << 25)) & _M
    return (s0 ^ s8, s1 ^ s9, s2 ^ s10, s3 ^ s11, s4 ^ s12, s5 ^ s13, s6 ^ s14, s7 ^ s15,
            s8 ^ cv[0], s9 ^ cv[1], s10 ^ cv[2], s11 ^ cv[3], s12 ^ cv[4], s13 ^ cv[5], s14 ^ cv[6], s15 ^ cv[7])


_WORDS = struct.Struct("<16I")


def _words(block):
    if len(block) < BLOCK_LEN:
        block = block + b"\x00" * (BLOCK_LEN - len(block))
    return _WORDS.unpack(block)


class _Output:
    __slots__ = ("cv", "words", "counter", "block_len", "flags")

    def __init__(self, cv, words, counter, block_len, flags):
        self.cv, self.words, self.counter, self.block_len, self.flags = cv, words, counter, block_len, flags

    def chaining_value(self):
        return _compress(self.cv, self.words, self.counter, self.block_len, self.flags)[:8]

    def root_bytes(self):
        out = _compress(self.cv, self.words, 0, self.block_len, self.flags | ROOT)
        return struct.pack("<8I", *out[:8])


def _chunk_output(chunk, counter):
    cv = _IV
    n = len(chunk)
    nblocks = max(1, (n + BLOCK_LEN - 1) // BLOCK_LEN)
    for i in range(nblocks - 1):
        flags = CHUNK_START if i == 0 else 0
        cv = _compress(cv, _WORDS.unpack_from(chunk, i * BLOCK_LEN), counter, BLOCK_LEN, flags)[:8]
    last = chunk[(nblocks - 1) * BLOCK_LEN:]
    flags = CHUNK_END | (CHUNK_START if nblocks == 1 else 0)
    return _Output(cv, _words(last), counter, len(last), flags)


def _parent_output(left, right):
    return _Output(_IV, tuple(left) + tuple(right), 0, BLOCK_LEN, PARENT)


def digest(data: bytes) -> bytes:
    """BLAKE3-256 of data."""
    data = bytes(data)
    n = len(data)
    nchunks = max(1, (n + CHUNK_LEN - 1) // CHUNK_LEN)
    stack = []
    for i in range(nchunks - 1):
        cv = _chunk_output(data[i * CHUNK_LEN:(i + 1) * CHUNK_LEN], i).chaining_value()
        total = i + 1
        while total & 1 == 0:
            cv = _parent_output(stack.pop(), cv).chaining_value()
            total >>= 1
        stack.append(cv)
    out = _chunk_output(data[(nchunks - 1) * CHUNK_LEN:], nchunks - 1)
    while stack:
        out = _parent_output(stack.pop(), out.chaining_value())
    return out.root_bytes()


def hexdigest(data: bytes) -> str:
    return digest(data).hex()


if __name__ == "__main__":
    # Published BLAKE3 test values.
    assert hexdigest(b"") == "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262"
    assert hexdigest(b"abc") == "6437b3ac38465133ffb63b75273a8db548c558465d79db03fd359c6cd5bd9d85"
    print("blake3 self-test ok")
