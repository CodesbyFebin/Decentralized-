#!/bin/bash
# P1-LOCAL-VM-A01: Destroy Cluster
# Forcefully terminates VMs and removes cluster state
# Preserves evidence directory
# Usage: ./destroy-cluster.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
CLUSTER_JSON="$STATE_DIR/cluster.json"
EVIDENCE_DIR="$REPO_ROOT/validation/local-vm/evidence"

if [ ! -f "$CLUSTER_JSON" ]; then
  echo "No cluster configured."
  exit 0
fi

HYPERVISOR=$(jq -r '.hypervisor' "$CLUSTER_JSON")
NODES=$(jq -r '.nodes' "$CLUSTER_JSON")

echo "=== P1-LOCAL-VM-A01: Destroy Cluster ==="
echo "Hypervisor: $HYPERVISOR"
echo "Nodes: $NODES"
echo "State Directory: $STATE_DIR"
echo ""
echo "⚠️  WARNING: This will destroy all VMs and cluster state!"
echo "⚠️  Evidence directory ($EVIDENCE_DIR) will be PRESERVED"
echo ""
read -p "Type 'yes' to confirm destruction: " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
  echo "Cancelled."
  exit 0
fi

echo ""

if [ "$HYPERVISOR" != "qemu" ]; then
  echo "ERROR: Cluster backend is '$HYPERVISOR', expected QEMU"
  echo "Cannot safely destroy unknown backend"
  exit 1
fi

# Destroy each VM (SIGKILL if necessary)
echo "Destroying VMs..."
for ((i=1; i<=NODES; i++)); do
  NODE_NAME=$(jq -r ".node_details[$((i-1))].name" "$CLUSTER_JSON")
  PID_FILE=$(jq -r ".node_details[$((i-1))].qemu_pid_file" "$CLUSTER_JSON")

  if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if kill -0 "$PID" 2>/dev/null; then
      echo "  Terminating $NODE_NAME (PID: $PID)..."
      kill -9 "$PID" 2>/dev/null || true
      sleep 1
    fi
    rm -f "$PID_FILE"
  fi
done

echo ""
echo "Removing cluster state..."
rm -rf "$STATE_DIR" || true

echo ""
echo "=== Destruction Complete ==="
echo ""
echo "Preserved:"
echo "  - Evidence/qualification records: $EVIDENCE_DIR"
echo "  - Configuration templates: $REPO_ROOT/validation/local-vm/config/"
echo ""
echo "To run qualification again:"
echo "  bash $SCRIPT_DIR/create-vm-cluster.sh qemu"
