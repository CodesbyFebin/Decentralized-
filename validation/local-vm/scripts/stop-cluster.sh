#!/bin/bash
# P1-LOCAL-VM-A01: Stop Local VM Cluster (graceful)
# Sends SIGTERM to all QEMU processes and waits for graceful shutdown
# Usage: ./stop-cluster.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CLUSTER_STATE_DIR="$REPO_ROOT/.p1-local-vm-state"

if [ ! -f "$CLUSTER_STATE_DIR/manifest.json" ]; then
  echo "No cluster configured."
  exit 0
fi

NODES=$(jq -r '.nodes' "$CLUSTER_STATE_DIR/manifest.json")

echo "=== P1-LOCAL-VM-A01: Stop Cluster (graceful) ==="
echo "Nodes: $NODES"
echo ""

# Stop each VM gracefully
for ((i=1; i<=NODES; i++)); do
  NODE_NAME=$(jq -r ".nodes[$((i-1))].name" "$CLUSTER_STATE_DIR/manifest.json")
  DISK=$(jq -r ".nodes[$((i-1))].disk" "$CLUSTER_STATE_DIR/manifest.json")
  NODE_DIR="$(dirname "$DISK")"
  PID_FILE="$NODE_DIR/qemu.pid"

  if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if kill -0 "$PID" 2>/dev/null; then
      echo "Stopping $NODE_NAME (PID: $PID)..."
      kill "$PID" 2>/dev/null || true

      # Wait up to 10 seconds for graceful shutdown
      for ((j=0; j<10; j++)); do
        if ! kill -0 "$PID" 2>/dev/null; then
          echo "  ✓ Stopped"
          rm -f "$PID_FILE"
          break
        fi
        sleep 1
      done

      # If still running, check and log
      if kill -0 "$PID" 2>/dev/null; then
        echo "  ⚠ Still running after 10 seconds"
      fi
    fi
  fi
done

echo ""
echo "=== Cluster Stopped ==="
echo "To restart: bash $SCRIPT_DIR/start-cluster.sh"
echo "To destroy: bash $SCRIPT_DIR/destroy-cluster.sh"
