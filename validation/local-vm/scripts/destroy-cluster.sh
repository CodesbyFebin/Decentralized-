#!/bin/bash
# P1-LOCAL-VM-A01: Destroy Local VM Cluster
# Forcefully terminates all QEMU processes and removes cluster state
# Usage: ./destroy-cluster.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CLUSTER_STATE_DIR="$REPO_ROOT/.p1-local-vm-state"

if [ ! -f "$CLUSTER_STATE_DIR/manifest.json" ]; then
  echo "No cluster configured."
  exit 0
fi

NODES=$(jq -r '.nodes' "$CLUSTER_STATE_DIR/manifest.json")

echo "=== P1-LOCAL-VM-A01: Destroy Cluster ==="
echo "Nodes: $NODES"
echo "Cluster State: $CLUSTER_STATE_DIR"
echo ""
echo "⚠️  WARNING: This will destroy all VMs and local data!"
echo "⚠️  Evidence files in $REPO_ROOT/validation/local-vm/evidence/ will NOT be deleted."
echo "⚠️  This action cannot be undone."
echo ""
read -p "Type 'yes' to confirm destruction: " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
  echo "Cancelled."
  exit 0
fi

echo ""
echo "Destroying VMs..."

# Kill all QEMU processes
for ((i=1; i<=NODES; i++)); do
  NODE_NAME=$(jq -r ".nodes[$((i-1))].name" "$CLUSTER_STATE_DIR/manifest.json")
  DISK=$(jq -r ".nodes[$((i-1))].disk" "$CLUSTER_STATE_DIR/manifest.json")
  NODE_DIR="$(dirname "$DISK")"
  PID_FILE="$NODE_DIR/qemu.pid"

  if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if kill -0 "$PID" 2>/dev/null; then
      echo "Terminating $NODE_NAME (PID: $PID)..."
      kill -9 "$PID" 2>/dev/null || true
    fi
    rm -f "$PID_FILE"
  fi
done

sleep 1

echo ""
echo "Removing cluster state directory..."
rm -rf "$CLUSTER_STATE_DIR" || true

echo ""
echo "=== Cleanup Complete ==="
echo "Cluster has been destroyed."
echo ""
echo "Preserved:"
echo "  - Evidence/qualification records: $REPO_ROOT/validation/local-vm/evidence/"
echo "  - Configuration templates: $REPO_ROOT/validation/local-vm/config/"
echo "  - Scripts: $REPO_ROOT/validation/local-vm/scripts/"
echo ""
echo "To run qualification again:"
echo "  bash $SCRIPT_DIR/create-vm-cluster.sh qemu"
