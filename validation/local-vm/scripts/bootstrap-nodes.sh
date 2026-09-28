#!/bin/bash
# P1-LOCAL-VM-A01: Bootstrap Nodes
# Verifies SSH access to all VMs and collects baseline evidence
# Usage: ./bootstrap-nodes.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
CLUSTER_JSON="$STATE_DIR/cluster.json"
SSH_KEY="$HOME/.ssh/p1-local-vm"
SSH_USER="ubuntu"
MAX_RETRIES=20
RETRY_DELAY=2

if [ ! -f "$CLUSTER_JSON" ]; then
  echo "ERROR: Cluster not configured. Run create-vm-cluster.sh first."
  exit 1
fi

if [ ! -f "$SSH_KEY" ]; then
  echo "ERROR: SSH key not found: $SSH_KEY"
  exit 1
fi

NODES=$(jq -r '.nodes' "$CLUSTER_JSON")

echo "=== P1-LOCAL-VM-A01: Bootstrap Nodes ==="
echo "SSH Key: $SSH_KEY"
echo "Nodes: $NODES"
echo ""

ssh_exec() {
  local port=$1
  shift

  ssh -i "$SSH_KEY" \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o ConnectTimeout=2 \
    -p "$port" \
    "$SSH_USER@localhost" "$@" 2>/dev/null || return 1
}

# Wait for all nodes to be SSH-accessible
echo "Waiting for all nodes to boot..."
all_ready=false
attempt=0

while [ "$all_ready" = false ] && [ $attempt -lt $MAX_RETRIES ]; do
  all_ready=true

  for ((i=1; i<=NODES; i++)); do
    NODE_NAME=$(jq -r ".node_details[$((i-1))].name" "$CLUSTER_JSON")
    SSH_PORT=$(jq -r ".node_details[$((i-1))].ssh_port" "$CLUSTER_JSON")

    if ssh_exec "$SSH_PORT" "echo ready" > /dev/null 2>&1; then
      echo "  ✓ $NODE_NAME ($SSH_PORT)"
    else
      all_ready=false
      echo "  ✗ $NODE_NAME ($SSH_PORT) not ready"
    fi
  done

  if [ "$all_ready" = false ]; then
    attempt=$((attempt + 1))
    if [ $attempt -lt $MAX_RETRIES ]; then
      echo "  Attempt $attempt/$MAX_RETRIES, retrying in ${RETRY_DELAY}s..."
      sleep $RETRY_DELAY
    fi
  fi
done

if [ "$all_ready" = false ]; then
  echo ""
  echo "ERROR: Not all nodes became SSH-accessible"
  exit 1
fi

echo ""
echo "✓ All nodes SSH-accessible"
echo ""

# Collect baseline evidence from each node
echo "Collecting baseline evidence..."
for ((i=1; i<=NODES; i++)); do
  NODE_NAME=$(jq -r ".node_details[$((i-1))].name" "$CLUSTER_JSON")
  SSH_PORT=$(jq -r ".node_details[$((i-1))].ssh_port" "$CLUSTER_JSON")

  echo ""
  echo "Node $i: $NODE_NAME (port $SSH_PORT)"

  # Hostname
  HOSTNAME=$(ssh_exec "$SSH_PORT" "hostname" 2>/dev/null || echo "UNKNOWN")
  echo "  Hostname: $HOSTNAME"

  # Machine ID
  MACHINE_ID=$(ssh_exec "$SSH_PORT" "cat /etc/machine-id" 2>/dev/null || echo "UNKNOWN")
  echo "  Machine ID: $MACHINE_ID"

  # Kernel
  KERNEL=$(ssh_exec "$SSH_PORT" "uname -r" 2>/dev/null || echo "UNKNOWN")
  echo "  Kernel: $KERNEL"

  # Architecture
  ARCH=$(ssh_exec "$SSH_PORT" "uname -m" 2>/dev/null || echo "UNKNOWN")
  echo "  Architecture: $ARCH"

  # CPU count
  CPUS=$(ssh_exec "$SSH_PORT" "grep -c ^processor /proc/cpuinfo" 2>/dev/null || echo "UNKNOWN")
  echo "  CPUs: $CPUS"

  # Memory
  MEMORY=$(ssh_exec "$SSH_PORT" "free -h | grep Mem | awk '{print \$2}'" 2>/dev/null || echo "UNKNOWN")
  echo "  Memory: $MEMORY"

  # Uptime
  UPTIME=$(ssh_exec "$SSH_PORT" "uptime -p" 2>/dev/null || echo "UNKNOWN")
  echo "  Uptime: $UPTIME"

  # Disk
  DISK=$(ssh_exec "$SSH_PORT" "df -h / | tail -1 | awk '{print \$4}'" 2>/dev/null || echo "UNKNOWN")
  echo "  Disk Free: $DISK"
done

echo ""
echo "=== Bootstrap Complete ==="
echo "All nodes ready for P1-LOCAL-VM-A01 campaign."
echo ""
echo "Verify 3 distinct VMs:"
for ((i=1; i<=NODES; i++)); do
  SSH_PORT=$(jq -r ".node_details[$((i-1))].ssh_port" "$CLUSTER_JSON")
  echo "  ssh -i ~/.ssh/p1-local-vm -p $SSH_PORT ubuntu@localhost 'hostname; cat /etc/machine-id; uname -m'"
done
echo ""
echo "Expected: 3 different hostnames, 3 different machine IDs, same architecture"
