#!/bin/bash
# P1-ENDTOEND-A01: Bootstrap Local Nodes
# Verify SSH access and baseline evidence collection
# Usage: ./bootstrap-nodes.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../config"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/p1-local-vm}"
SSH_USER="ubuntu"
MAX_RETRIES=10
RETRY_DELAY=3

if [ ! -f "$CONFIG_DIR/vm-specs.json" ]; then
  echo "ERROR: VM cluster not configured. Run create-vm-cluster.sh first."
  exit 1
fi

NODES=$(jq -r '.nodes[] | .node_id' "$CONFIG_DIR/vm-specs.json" 2>/dev/null || seq 1 3)

echo "=== P1-ENDTOEND-A01: Bootstrap Nodes ==="
echo "SSH Key: $SSH_KEY"
echo ""

if [ ! -f "$SSH_KEY" ]; then
  echo "ERROR: SSH key not found: $SSH_KEY"
  exit 1
fi

ssh_exec() {
  local node_id=$1
  local port=$((2220 + node_id))
  shift

  ssh -i "$SSH_KEY" \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o ConnectTimeout=5 \
    "$SSH_USER@localhost" -p "$port" "$@" 2>/dev/null || echo ""
}

# Wait for SSH availability
for node_id in $NODES; do
  NODE_NAME="dh-local-$(printf '%02d' $node_id)"
  PORT=$((2220 + node_id))

  echo "Waiting for $NODE_NAME (port $PORT) to boot..."

  for retry in $(seq 1 $MAX_RETRIES); do
    if ssh_exec "$node_id" "echo ok" > /dev/null 2>&1; then
      echo "✓ $NODE_NAME SSH access OK"
      break
    fi

    if [ $retry -lt $MAX_RETRIES ]; then
      echo "  Attempt $retry/$MAX_RETRIES (waiting ${RETRY_DELAY}s)..."
      sleep $RETRY_DELAY
    else
      echo "✗ $NODE_NAME failed to boot"
      exit 1
    fi
  done

  echo ""
done

# Verify baseline evidence
echo "Verifying baseline evidence collection..."
for node_id in $NODES; do
  NODE_NAME="dh-local-$(printf '%02d' $node_id)"

  echo "Checking $NODE_NAME..."

  # Check baseline evidence
  if ssh_exec "$node_id" "test -f /var/log/decentralized-host/baseline-evidence.json"; then
    echo "  ✓ Baseline evidence collected"
  else
    echo "  ✗ Baseline evidence NOT found"
  fi

  # Check network evidence
  if ssh_exec "$node_id" "test -f /var/log/decentralized-host/network-evidence.json"; then
    echo "  ✓ Network evidence collected"
  else
    echo "  ✗ Network evidence NOT found"
  fi

  # Check system status
  UPTIME=$(ssh_exec "$node_id" "uptime -p")
  echo "  Uptime: $UPTIME"

  # Check hostname
  HOSTNAME=$(ssh_exec "$node_id" "hostname")
  echo "  Hostname: $HOSTNAME"

  # Check disk space
  DISK=$(ssh_exec "$node_id" "df -h / | tail -1 | awk '{print \$4}' ")
  echo "  Free Disk: $DISK"

  echo ""
done

echo "=== Bootstrap Complete ==="
echo "All nodes are ready for Decentralized Host deployment."
echo ""
echo "Next steps:"
echo "1. Copy DHP binaries to each node"
echo "2. Start agents: sudo systemctl start decentralized-host-agent"
echo "3. Run scenarios: bash $SCRIPT_DIR/run-scenarios.sh"
