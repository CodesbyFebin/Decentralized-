# 0005 — Userspace WireGuard with signed key bindings

**Status:** accepted

The mesh is wireguard-go on a gVisor netstack in the agent process, with no
root and no kernel module. WireGuard keys are bound to dh1 identities by
self-signed `wg-binding` statements, and SWIM gossip runs inside the tunnel.

**Why.** Hosts should not need root to join. Binding keys to identities
means the control plane cannot splice a foreign key into a host's peer set.

**Consequences.** Workloads are reached through per-assignment forwarders on
mesh ports, not a routed IP per workload. Throughput is lower than kernel
WireGuard (about 70 MiB/s measured locally), which is enough for control and
storage traffic.
