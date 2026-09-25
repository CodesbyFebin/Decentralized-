# 0001 — One Go module, static binaries

**Status:** accepted

The earlier prototype was a Node.js app. The rebuild is one Go module that
produces static binaries (`dh`, `dh-control`, `dh-noded`, `dh-conformance`).
Operators install files, not a runtime.
- Go gives the protocol code, hashicorp/raft, wireguard-go with the gVisor
  netstack, and x/crypto/acme in one memory-safe toolchain.
- The race detector covers the concurrency-heavy agent.

**Consequences.** The Node prototype is kept only as reference. The protocol
is specified independently of Go (`docs/protocol/dh-v1.md`), and a
separate Python implementation keeps it honest.
