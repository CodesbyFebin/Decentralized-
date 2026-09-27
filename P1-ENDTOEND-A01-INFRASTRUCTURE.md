# P1-ENDTOEND-A01: Test Infrastructure Setup

This document specifies the exact infrastructure, configuration, and deployment procedures for the P1 qualification test.

---

## Node Provisioning

### Node Specifications

```yaml
Control Plane:
  hostname: cp-0
  cpu: 4+ cores
  memory: 8GB
  disk: 50GB (ext4)
  network: eth0 (real network, not loopback)
  os: Ubuntu 22.04 LTS or equivalent
  role: DHP control plane, Raft quorum

Provider Node A:
  hostname: provider-a
  cpu: 2+ cores
  memory: 4GB
  disk: 30GB (ext4)
  network: eth0 (real network)
  os: Ubuntu 22.04 LTS
  role: Workload execution, evidence signing

Provider Node B:
  hostname: provider-b
  cpu: 2+ cores
  memory: 4GB
  disk: 30GB (ext4)
  network: eth0 (real network)
  os: Ubuntu 22.04 LTS
  role: Workload execution, evidence signing

Provider Node C:
  hostname: provider-c
  cpu: 2+ cores
  memory: 4GB
  disk: 30GB (ext4)
  network: eth0 (real network)
  os: Ubuntu 22.04 LTS
  role: Workload execution, evidence signing

Observer Node:
  hostname: observer-0
  cpu: 2+ cores
  memory: 2GB
  disk: 20GB (ext4)
  network: eth0 (can reach all nodes)
  os: Ubuntu 22.04 LTS
  role: Log aggregation, evidence validation, test orchestration
```

### Network Configuration

**Cluster Network**: 192.168.100.0/24 (example, adjust to your environment)

```
cp-0         = 192.168.100.10
provider-a   = 192.168.100.20
provider-b   = 192.168.100.21
provider-c   = 192.168.100.22
observer-0   = 192.168.100.30
```

**Network Constraints**:
- All nodes have real, routable IP addresses (NOT localhost/127.0.0.1)
- Latency: 1-50ms inter-node (measure with ping)
- Packet loss: <0.1% baseline (clean network)
- Bandwidth: ≥100 Mbps (for workload transfer)
- NTP synchronized: ±100ms max clock skew

**Verify Network**:
```bash
# From observer-0
for node in cp-0 provider-a provider-b provider-c; do
  echo "=== $node ==="
  ping -c 3 $node
  ssh $node "date" | awk '{print $4}' | sort | uniq -c
done
```

---

## Software Installation

### Prerequisites (All Nodes)

```bash
sudo apt-get update
sudo apt-get install -y \
  curl wget git vim htop \
  openssh-server openssh-client \
  ca-certificates gnupg lsb-release \
  net-tools traceroute ntp \
  jq yq \
  build-essential
```

### Control Plane Setup

```bash
# 1. Install DHP control plane
# (Assume built from /home/user/Decentralized-/control-plane)
scp -r control-plane cp-0:/opt/dhp-control/
ssh cp-0 "cd /opt/dhp-control && ./install.sh"

# 2. Initialize Raft cluster (3-node quorum with provider nodes)
ssh cp-0 "dhp-ctl init-cluster \
  --node-id cp-0 \
  --peers provider-a,provider-b,provider-c"

# 3. Start control plane service
ssh cp-0 "sudo systemctl start dhp-control"
ssh cp-0 "sudo systemctl enable dhp-control"

# 4. Verify running
ssh cp-0 "dhp-ctl status"
# Expected: leader elected, 3 quorum members
```

### Provider Node Setup (All 3)

```bash
# Run for each provider node (provider-a, provider-b, provider-c)
for node in provider-a provider-b provider-c; do
  # 1. Install DHP node agent
  scp -r node-agent $node:/opt/dhp-node/
  ssh $node "cd /opt/dhp-node && ./install.sh"
  
  # 2. Install container runtime
  ssh $node "sudo apt-get install -y containerd docker.io"
  ssh $node "sudo systemctl start containerd"
  ssh $node "sudo systemctl enable containerd"
  
  # 3. Configure node agent to connect to control plane
  ssh $node "cat > /etc/dhp-node/config.yaml <<EOF
control_plane: cp-0:8080
node_id: $node
data_dir: /var/lib/dhp
log_level: info
signing_key_path: /etc/dhp-node/node.key
EOF
"
  
  # 4. Generate node signing key (Ed25519)
  ssh $node "openssl genpkey -algorithm ed25519 -out /etc/dhp-node/node.key"
  ssh $node "chmod 600 /etc/dhp-node/node.key"
  
  # 5. Start node agent
  ssh $node "sudo systemctl start dhp-node"
  ssh $node "sudo systemctl enable dhp-node"
  
  # 6. Verify joined
  sleep 5
done

# Verify all nodes joined
ssh cp-0 "dhp-ctl nodes list"
# Expected: provider-a, provider-b, provider-c all LIVE
```

### Observer Node Setup

```bash
# 1. Install log aggregation (rsyslog or similar)
ssh observer-0 "sudo apt-get install -y rsyslog"

# 2. Configure to receive remote logs
ssh observer-0 "cat >> /etc/rsyslog.conf <<EOF
\$ModLoad imudp
\$UDPServerRun 514
\$ModLoad imtcp
\$InputTCPServerRun 514
EOF
"
ssh observer-0 "sudo systemctl restart rsyslog"

# 3. Configure all nodes to forward logs to observer
for node in cp-0 provider-a provider-b provider-c; do
  ssh $node "cat >> /etc/rsyslog.conf <<EOF
*.* @observer-0:514
EOF
"
  ssh $node "sudo systemctl restart rsyslog"
done

# 4. Install evidence validator (part of DHP)
scp -r tools/evidence-validator observer-0:/opt/dhp-validator/
ssh observer-0 "cd /opt/dhp-validator && ./install.sh"

# 5. Install test orchestrator
scp -r tools/test-orchestrator observer-0:/opt/test-orchestrator/
ssh observer-0 "cd /opt/test-orchestrator && ./install.sh"
```

---

## Baseline Validation (Pre-Test)

Run these checks **before** starting any test scenarios:

### Cluster Health

```bash
# From observer-0
ssh cp-0 "dhp-ctl status" | jq .
# Expected:
# - status: "running"
# - role: "leader"
# - quorum_members: ["cp-0", "provider-a", "provider-b", "provider-c"]
# - committed_index: >0

# Verify all nodes joined
ssh cp-0 "dhp-ctl nodes list" | jq '.[] | {id: .id, state: .state}'
# Expected: all nodes state=LIVE
```

### Network Baseline

```bash
# From observer-0, measure latency
for node in cp-0 provider-a provider-b provider-c; do
  latency=$(ping -c 10 $node | tail -1 | awk -F'/' '{print $5}')
  echo "$node: ${latency}ms"
done
# Expected: <50ms all nodes
```

### No-Mock Gate Verification

```bash
# On each node, verify no mock data in running processes
for node in cp-0 provider-a provider-b provider-c; do
  ssh $node "ps aux | grep dhp"  # Find process
  
  # Extract binary path and verify source
  ssh $node "strings /opt/dhp-*/bin/* | grep -i 'mock\|fake\|dummy'"
  # Expected: zero matches (or only in test helpers)
done

# Verify control plane API returns real data
curl -s "http://cp-0:8080/api/v1/nodes" | jq '.[] | {id, uptime, cpu_cores, memory_bytes}'
# Expected: actual hardware facts, timestamps recent
```

### Evidence System Test

```bash
# Deploy a dummy workload to verify evidence generation
ssh cp-0 "dhp-ctl workload deploy \
  --name test-workload \
  --image nginx:latest \
  --cpus 100 \
  --memory 256Mi"

# Wait for placement
sleep 10

# Verify evidence was generated
ssh cp-0 "dhp-ctl evidence list --workload test-workload" | jq '.[]'
# Expected: PLACEMENT + EXECUTION evidence records

# Clean up
ssh cp-0 "dhp-ctl workload delete --name test-workload"
```

---

## Failure Injection Mechanisms

### Node Network Isolation (for Scenario 2)

```bash
# On observer-0, isolate provider-b from network
ssh provider-b "sudo iptables -A OUTPUT -j DROP"
ssh provider-b "sudo iptables -A INPUT -j DROP"

# Verify isolation
ping -c 1 provider-b  # Should timeout

# Restore connectivity
ssh provider-b "sudo iptables -F"
ssh provider-b "sudo iptables -X"
```

### Disk Full (for Scenario 4)

```bash
# Fill disk on provider-b to trigger storage failure
ssh provider-b "dd if=/dev/zero of=/var/lib/dhp/fillfile bs=1M count=29000"
# Wait for storage subsystem to detect

# Verify disk is full
ssh provider-b "df /var/lib/dhp | tail -1 | awk '{print $5}'"
# Expected: 99% or 100%

# Clean up
ssh provider-b "rm /var/lib/dhp/fillfile"
ssh provider-b "sync"
```

### CPU Throttle (optional, for advanced testing)

```bash
# Limit provider-a CPU to 50%
ssh provider-a "sudo apt-get install -y cpulimit"
ssh provider-a "cpulimit -p \$(pgrep dhp-node) -l 50"

# Remove throttle
ssh provider-a "killall cpulimit"
```

### Control Plane Restart (for Scenario 5)

```bash
ssh cp-0 "sudo systemctl stop dhp-control"
sleep 5
ssh cp-0 "sudo systemctl start dhp-control"

# Verify recovery
sleep 10
ssh cp-0 "dhp-ctl status"
```

---

## Test Execution Helpers

### Workload Deployment Script

```bash
#!/bin/bash
# deploy-test-workloads.sh

CONTROL_PLANE="cp-0"
WORKLOADS=(
  "nginx:latest|100|256"      # image|cpu_millicores|memory_bytes
  "echo-server|50|128"
  "sleep-daemon|25|64"
  "cpu-burner|200|512"
  "mem-consumer|50|1024"
)

for wl in "${WORKLOADS[@]}"; do
  IFS='|' read -r image cpu mem <<< "$wl"
  name="${image%:*}-$(date +%s)"
  
  ssh $CONTROL_PLANE "dhp-ctl workload deploy \
    --name $name \
    --image $image \
    --cpus $cpu \
    --memory ${mem}Mi"
  
  echo "Deployed: $name"
  sleep 5
done
```

### Evidence Collector Script

```bash
#!/bin/bash
# collect-evidence.sh

SCENARIO=$1
OUTPUT_DIR="evidence-${SCENARIO}"
mkdir -p "$OUTPUT_DIR"

# Collect from control plane
ssh cp-0 "dhp-ctl evidence dump" | jq '.' > "$OUTPUT_DIR/control-plane-evidence.jsonl"

# Collect from each provider
for node in provider-a provider-b provider-c; do
  ssh $node "dhp-node evidence dump" | jq '.' > "$OUTPUT_DIR/${node}-evidence.jsonl"
done

# Collect logs
scp observer-0:/var/log/syslog "$OUTPUT_DIR/syslog-combined"

echo "Evidence collected to $OUTPUT_DIR"
```

### Verification Script

```bash
#!/bin/bash
# verify-evidence.sh

EVIDENCE_DIR=$1

# Run offline verification
ssh observer-0 "dhp-validator verify-all \
  --input-dir $EVIDENCE_DIR \
  --output verify-report.json"

# Check for gaps or errors
cat "$EVIDENCE_DIR/verify-report.json" | jq '.summary'
```

---

## Monitoring & Metrics

### Real-Time Dashboard

During test execution, monitor:

```bash
# Terminal 1: Control plane status
watch -n 2 "ssh cp-0 'dhp-ctl status | jq .'"

# Terminal 2: Node status
watch -n 2 "ssh cp-0 'dhp-ctl nodes list | jq .'"

# Terminal 3: Workload status
watch -n 2 "ssh cp-0 'dhp-ctl workloads list | jq .'"

# Terminal 4: Evidence count
watch -n 5 "ssh cp-0 'dhp-ctl evidence count'"
```

### Log Tailing

```bash
# Tail control plane logs
ssh cp-0 "sudo tail -f /var/log/dhp-control.log" | grep -E "ERROR|WARN|reconcile|placement"

# Tail provider node logs
ssh provider-a "sudo tail -f /var/log/dhp-node.log" | grep -E "ERROR|WARN|execute|evidence"

# Tail observer node evidence validator
ssh observer-0 "tail -f /var/log/dhp-validator.log"
```

---

## Teardown

After test completion:

```bash
# Stop all services
for node in cp-0 provider-a provider-b provider-c; do
  ssh $node "sudo systemctl stop dhp-control dhp-node"
done

# Collect final evidence
./collect-evidence.sh final

# Preserve node logs
for node in cp-0 provider-a provider-b provider-c; do
  scp $node:/var/log/dhp-*.log test-results/logs/
done

# Shut down nodes (don't delete; preserve for investigation)
echo "Nodes preserved for investigation. To delete:"
echo "  for node in cp-0 provider-a provider-b provider-c observer-0; do"
echo "    cloud-cli delete-instance \$node"
echo "  done"
```

---

## Quick Start

```bash
# 1. Provision 5 nodes (prerequisites)
# 2. Install software (control-plane, providers, observer)
./install-cluster.sh

# 3. Verify baseline
./verify-baseline.sh

# 4. Run test
./run-test.sh --scenario all

# 5. Collect evidence
./collect-evidence.sh scenario-all

# 6. Verify
./verify-evidence.sh evidence-scenario-all

# 7. Generate report
./generate-report.sh evidence-scenario-all > P1-ENDTOEND-A01-REPORT.md
```

