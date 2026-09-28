#!/bin/bash
# P1-LOCAL-VM-A01: Stop Cluster (graceful)
# Reads cluster.json and stops all VMs
# Usage: ./stop-cluster.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
CLUSTER_JSON="$STATE_DIR/cluster.json"

if [ ! -f "$CLUSTER_JSON" ]; then
  echo "No cluster configured."
  exit 0
fi

HYPERVISOR=$(jq -r '.hypervisor' "$CLUSTER_JSON")
NODES=$(jq -r '.nodes' "$CLUSTER_JSON")

echo "=== P1-LOCAL-VM-A01: Stop Cluster ==="
echo "Hypervisor: $HYPERVISOR"
echo "Nodes: $NODES"
echo ""

if [ "$HYPERVISOR" != "qemu" ]; then
  echo "ERROR: Cluster backend is '$HYPERVISOR', expected QEMU"
  exit 1
fi

# Stop each VM gracefully with SIGTERM
echo "Stopping VMs gracefully (SIGTERM)..."
for ((i=1; i<=NODES; i++)); do
  NODE_NAME=$(jq -r ".node_details[$((i-1))].name" "$CLUSTER_JSON")
  PID_FILE=$(jq -r ".node_details[$((i-1))].qemu_pid_file" "$CLUSTER_JSON")

  if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if kill -0 "$PID" 2>/dev/null; then
      echo "  Stopping $NODE_NAME (PID: $PID)..."
      kill "$PID" 2>/dev/null || true

      # Wait up to 10 seconds for graceful shutdown
      for ((j=0; j<10; j++)); do
        if ! kill -0 "$PID" 2>/dev/null; then
          echo "    ✓ Stopped"
          rm -f "$PID_FILE"
          break
        fi
        sleep 1
      done

      if kill -0 "$PID" 2>/dev/null; then
        echo "    ⚠ Still running after 10 seconds"
      fi
    fi
  fi
done

echo ""
echo "=== Cluster Stopped ==="
echo "To restart:"
echo "  bash $SCRIPT_DIR/start-cluster.sh"
echo ""
echo "To destroy:"
echo "  bash $SCRIPT_DIR/destroy-cluster.sh"
