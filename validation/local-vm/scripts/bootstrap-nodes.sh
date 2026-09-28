#!/bin/bash
# P1-LOCAL-VM-A01: Bootstrap Local Nodes
# Verify SSH access to all 3 VMs and collect baseline evidence
# Usage: ./bootstrap-nodes.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CLUSTER_STATE_DIR="$REPO_ROOT/.p1-local-vm-state"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/p1-local-vm}"
SSH_USER="ubuntu"
MAX_RETRIES=15
RETRY_DELAY=2

if [ ! -f "$CLUSTER_STATE_DIR/manifest.json" ]; then
  echo "ERROR: Cluster not configured. Run create-vm-cluster.sh first."
  exit 1
fi

NODES=$(jq -r '.nodes' "$CLUSTER_STATE_DIR/manifest.json")

echo "=== P1-LOCAL-VM-A01: Bootstrap Nodes ==="
echo "SSH Key: $SSH_KEY"
echo "Cluster State: $CLUSTER_STATE_DIR"
echo ""

if [ ! -f "$SSH_KEY" ]; then
  echo "ERROR: SSH key not found: $SSH_KEY"
  exit 1
fi

ssh_exec() {
  local port=$1
  shift

  ssh -i "$SSH_KEY" \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o ConnectTimeout=3 \
    -p "$port" \
    "$SSH_USER@localhost" "$@" 2>/dev/null || return 1
}

# Wait for SSH availability on all nodes
echo "Waiting for all nodes to boot..."
all_ready=false
attempts=0

while [ "$all_ready" = false ] && [ $attempts -lt $MAX_RETRIES ]; do
  all_ready=true

  for ((i=1; i<=NODES; i++)); do
    NODE_NAME=$(jq -r ".nodes[$((i-1))].name" "$CLUSTER_STATE_DIR/manifest.json")
    SSH_PORT=$(jq -r ".nodes[$((i-1))].ssh_port" "$CLUSTER_STATE_DIR/manifest.json")

    if ! ssh_exec "$SSH_PORT" "echo ok" > /dev/null 2>&1; then
      all_ready=false
      echo "  ✗ $NODE_NAME not ready"
    else
      echo "  ✓ $NODE_NAME SSH OK"
    fi
  done

  if [ "$all_ready" = false ]; then
    attempts=$((attempts + 1))
    if [ $attempts -lt $MAX_RETRIES ]; then
      echo "  Waiting... (attempt $attempts/$MAX_RETRIES)"
      sleep $RETRY_DELAY
    fi
  fi
done

if [ "$all_ready" = false ]; then
  echo ""
  echo "ERROR: Not all nodes reached SSH availability after $((MAX_RETRIES * RETRY_DELAY)) seconds."
  exit 1
fi

echo ""
echo "✓ All nodes SSH accessible"
echo ""

# Collect baseline evidence from each node
echo "Collecting baseline evidence..."
for ((i=1; i<=NODES; i++)); do
  NODE_NAME=$(jq -r ".nodes[$((i-1))].name" "$CLUSTER_STATE_DIR/manifest.json")
  SSH_PORT=$(jq -r ".nodes[$((i-1))].ssh_port" "$CLUSTER_STATE_DIR/manifest.json")

  echo "Node $i: $NODE_NAME (port $SSH_PORT)"

  # Check system info
  HOSTNAME=$(ssh_exec "$SSH_PORT" "hostname" 2>/dev/null || echo "UNKNOWN")
  echo "  Hostname: $HOSTNAME"

  KERNEL=$(ssh_exec "$SSH_PORT" "uname -r" 2>/dev/null || echo "UNKNOWN")
  echo "  Kernel: $KERNEL"

  ARCH=$(ssh_exec "$SSH_PORT" "uname -m" 2>/dev/null || echo "UNKNOWN")
  echo "  Architecture: $ARCH"

  CPUS=$(ssh_exec "$SSH_PORT" "grep -c ^processor /proc/cpuinfo" 2>/dev/null || echo "UNKNOWN")
  echo "  CPUs: $CPUS"

  MEMORY=$(ssh_exec "$SSH_PORT" "free -h | grep Mem | awk '{print \$2}'" 2>/dev/null || echo "UNKNOWN")
  echo "  Memory: $MEMORY"

  UPTIME=$(ssh_exec "$SSH_PORT" "uptime -p" 2>/dev/null || echo "UNKNOWN")
  echo "  Uptime: $UPTIME"

  DISK_FREE=$(ssh_exec "$SSH_PORT" "df -h / | tail -1 | awk '{print \$4}'" 2>/dev/null || echo "UNKNOWN")
  echo "  Disk Free (/): $DISK_FREE"

  echo ""
done

echo "=== Bootstrap Complete ==="
echo "All nodes are ready for P1-LOCAL-VM-A01 qualification campaign."
echo ""
echo "To SSH into a node directly:"
echo "  ssh -i ~/.ssh/p1-local-vm -p 2201 ubuntu@localhost  # Node 1"
echo "  ssh -i ~/.ssh/p1-local-vm -p 2202 ubuntu@localhost  # Node 2"
echo "  ssh -i ~/.ssh/p1-local-vm -p 2203 ubuntu@localhost  # Node 3"
echo ""
echo "Next steps:"
echo "  1. Deploy Decentralized Host agent to each node"
echo "  2. Establish distributed consensus"
echo "  3. Inject failures and measure detection/recovery"
echo "  4. Collect and seal evidence"
