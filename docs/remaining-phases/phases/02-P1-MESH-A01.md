# P1-MESH-A01 — Membership/Gossip

Persistent cryptographic node identity. Gossip envelope: node_id, generation, monotonic sequence, observed timestamp, capability digest, allocation digest, workload digest, signature.

Lifecycle: JOINING -> ACTIVE -> SUSPECT -> DOWN; RECOVERING on return. CORDONED is an orthogonal scheduler flag.

Reject replay, stale generation and invalid signatures. Test 3-node membership, duplicate identity rejection, silence detection, partitioned views, heal convergence and cordon orthogonality.
