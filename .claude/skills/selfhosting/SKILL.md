# Self-Hosting Decentralized.Host

Deploy Decentralized.Host distributed scheduler in your own infrastructure with local policy enforcement and sovereign control.

## Prerequisites

- **Linux runtime**: QEMU/KVM, Kubernetes, native Linux, containers, or hypervisor (any real backend; simulation not accepted)
- **Container runtime**: Podman v3.4+ or Docker (Podman recommended for rootless operation)
- **3+ isolated runtime nodes**: Distinct failure domains (physical machines, VMs, or containers with separate namespaces)
- **CNI networking**: Bridge-based network with firewall plugin support (cniVersion 0.4.0 compatible)
- **Persistent storage**: For node identities, policy configs, and audit logs across campaigns

## Quick Start (Podman)

### 1. Verify Podman Installation

```bash
podman --version  # v3.4.0+
podman network ls  # Check CNI availability
```

### 2. Create Isolated Network

```bash
cat > dhnet.conflist << 'EOF'
{
  "cniVersion": "0.4.0",
  "name": "dhnet",
  "plugins": [
    {
      "type": "bridge",
      "bridge": "cni-podman1",
      "isGateway": true,
      "ipMasq": true,
      "hairpinMode": true,
      "ipam": {
        "type": "host-local",
        "routes": [{"dst": "0.0.0.0/0"}],
        "ranges": [[{"subnet": "172.30.0.0/24", "gateway": "172.30.0.1"}]]
      }
    },
    {"type": "portmap", "capabilities": {"portMappings": true}},
    {"type": "firewall"}
  ]
}
EOF

mkdir -p ~/.config/cni/net.d
cp dhnet.conflist ~/.config/cni/net.d/
```

### 3. Bootstrap Three Nodes

```bash
for i in 1 2 3; do
  podman run -d \
    --name dh-node-$i \
    --net dhnet \
    --hostname dh-node-$i \
    ubuntu:22.04 \
    sleep infinity
done
```

### 4. Configure Each Node

Inside each container:

```bash
# Install dependencies
apt-get update && apt-get install -y openssh-server python3 curl

# Enable SSH
ssh-keygen -A
echo "PasswordAuthentication yes" >> /etc/ssh/sshd_config
service ssh start

# Start health HTTP server (for liveness checks)
python3 -m http.server 8080 &
```

### 5. Enable Inter-Node Networking

On host (allow container-to-container traffic):

```bash
sudo iptables -I DOCKER-FORWARD -s 172.30.0.0/24 -d 172.30.0.0/24 -j ACCEPT
sudo iptables -I DOCKER-FORWARD -d 172.30.0.0/24 -s 172.30.0.0/24 -j ACCEPT
```

### 6. Verify Cluster Connectivity

```bash
for src in dh-node-1 dh-node-2 dh-node-3; do
  for dst in dh-node-1 dh-node-2 dh-node-3; do
    [ "$src" != "$dst" ] && \
    podman exec "$src" curl -s http://${dst}:8080/ >/dev/null && \
    echo "✓ $src → $dst" || echo "✗ $src → $dst"
  done
done
```

## Deployment Patterns

### Single Physical Host (Development)

- **Nodes**: 3 containers on same machine
- **Failure domains**: DISTINCT (separate namespaces, same kernel, same filesystem host)
- **Use case**: Testing, CI/CD, development
- **Qualification**: P1_CORE (all phases on single backend)

```bash
# Minimal setup: all containers on one host
podman run -d --name dh-node-1 --net dhnet ubuntu:22.04 sleep infinity
# ... repeat for nodes 2, 3
```

### Multi-VM (Staging)

- **Nodes**: VMs on shared hypervisor (QEMU/KVM, ESXi, etc.)
- **Failure domains**: DISTINCT VMs, SAME physical host
- **Use case**: Staging, integration testing
- **Qualification**: P1_CORE + P1_QEMU_VM (if using KVM)

```bash
# Each node runs on separate QEMU/KVM instance
for i in 1 2 3; do
  qemu-system-x86_64 -m 2G -drive file=node-$i.qcow2 ...
done
```

### Multi-Physical (Production)

- **Nodes**: Physical machines
- **Failure domains**: DISTINCT physical hosts, independent operators
- **Use case**: Production deployment
- **Qualification**: P1_CORE + P1_MULTIPHYSICAL + P1_MULTIOPERATOR

```bash
# Install on three independent machines
# Each runs: scheduler, control plane, policy engine
# Connected via mTLS mesh (WireGuard + Raft consensus)
```

## Local Policy Enforcement

Each node enforces admission control before executing work:

```bash
# Example policy (on each node):
cat > /etc/dh/policy.rego << 'EOF'
package policy

default allow = false

allow {
  input.work.signature.verified == true
  input.work.target == input.node.identity
  input.node.capacity.available >= input.work.required
}
EOF
```

Work is proposed as signed intent (Ed25519):

```bash
# Sign work proposal
dh-cli propose-work \
  --target dh-node-1 \
  --workload scheduler:placement \
  --resources cpu:2,memory:4Gi \
  --identity-key node-admin.key \
  --output work.signed.json

# Each node independently verifies signature + local policy
dh-node-1 exec: verify(work.signed) -> admit or reject
```

## Failure Detection & Recovery

The scheduler detects node failures via peer health probes:

```bash
# Automatic detection (HTTP/TCP liveness on port 8080)
# Latency: ~15-30s per probe cycle
# Recovery: Automatic workload redistribution to healthy nodes
```

Test failure injection (Phase 6 diagnostics):

```bash
# Inject node failure
podman stop dh-node-2

# Observe detection latency
# Measure time from stop → detection via peer probes

# Recover
podman start dh-node-2

# Verify all paths restored
podman exec dh-node-1 curl http://dh-node-2:8080/
```

## Persistence & Identity

Node identities survive restarts (persistent storage):

```bash
# Mount persistent volume for node identity
podman run -d \
  --name dh-node-1 \
  --mount type=volume,src=dh-node-1-identity,dst=/var/lib/dh \
  --net dhnet \
  ubuntu:22.04 sleep infinity

# Identity in /var/lib/dh/identity.ed25519 persists across restarts
```

## Qualification & Evidence

After deployment, run the 10-phase diagnostic to qualify:

```bash
cd /path/to/Decentralized-
bash validation/local-vm/QEMU-HOST-REMEDIATION-HANDOFF.md  # Reference runbook

# Phases 1-10 executed:
# 1. Preflight verification
# 2. Topology discovery
# 3. Bootstrap SSH setup
# 4. Mesh network verification
# 5. Workload baseline
# 6. Node failure injection & detection
# 7. Production detection verification
# 8. Heal and convergence
# 9. Recovery verification (multiple scenarios)
# 10. Evidence signing & checkpoint

# Output: validation/local-vm/evidence/DIAGNOSTIC_CHECKPOINT.json
```

## Security Considerations

- **No simulation**: All evidence from real, observable production behavior
- **Signed intent**: All work proposals are Ed25519-signed; unsigned work rejected
- **Local policy**: Each node independently enforces admission; no global consensus bypass
- **Audit trail**: All policy decisions logged per-host with timestamps
- **mTLS mesh**: Control plane uses TLS 1.3; peer identities bound to node keys
- **No hardcoded latency**: All metrics measured from actual system behavior, not assumptions

## Troubleshooting

### Containers Won't Start

```bash
# Check CNI config version
grep cniVersion ~/.config/cni/net.d/dhnet.conflist
# Must be "0.4.0", not "1.0.0" (firewall plugin incompatibility)

# Verify bridge plugin
podman network inspect dhnet
```

### No Inter-Container Connectivity

```bash
# Restore iptables rules
sudo iptables -I DOCKER-FORWARD -s 172.30.0.0/24 -d 172.30.0.0/24 -j ACCEPT
sudo iptables -I DOCKER-FORWARD -d 172.30.0.0/24 -s 172.30.0.0/24 -j ACCEPT

# Test ping
podman exec dh-node-1 ping -c 1 172.30.0.3
```

### SSH Handshake Fails

```bash
# Generate host keys
podman exec dh-node-1 ssh-keygen -A

# Enable password auth
podman exec dh-node-1 sed -i 's/PasswordAuthentication no/PasswordAuthentication yes/' /etc/ssh/sshd_config
podman exec dh-node-1 service ssh restart

# Clear stale known_hosts
rm ~/.ssh/known_hosts
```

## Next Steps

1. **Deploy nodes** using your chosen backend (containers, VMs, or physical)
2. **Run diagnostics** to verify failure detection & recovery
3. **Commit evidence** from 10-phase campaign
4. **Integrate policy** for your workloads and operator requirements
5. **Monitor** via logs and health probes; adjust probe latency if needed

## References

- **AGENTS.md**: Qualification hierarchy, evidence rules, dh/v1 conformance
- **P1_CORE qualification**: Container-based; all failure detection & recovery phases
- **dh/v1 conformance spec**: Signed intent, local policy, explicit state machine
- **Evidence integrity**: No hardcoding; all metrics from observable production behavior

---

*Generated by Claude Code (Decentralized.Host Remediation Diagnostic v10)*
