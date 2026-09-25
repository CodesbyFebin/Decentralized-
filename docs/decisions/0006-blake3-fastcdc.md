# 0006 — BLAKE3 content addressing with FastCDC and Merkle anti-entropy

**Status:** accepted

Objects are addressed by BLAKE3-256. Volumes and artifacts are chunked with
FastCDC (16/64/256 KiB, gear table derived from BLAKE3). Replicas compare 256
Merkle bucket roots. A snapshot commits when two replicas sign evidence that
they hold every chunk.

**Why.** Verifying every read makes corruption detectable and repairable
from any peer. Content-defined chunking deduplicates across snapshots.
Bucketed Merkle roots keep anti-entropy proportional to the difference.

**Consequences.** Replica counts shown to operators come from signed
evidence, never from configuration. Erasure coding is not implemented;
durability comes from replication.
