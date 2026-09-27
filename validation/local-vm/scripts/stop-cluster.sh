#!/bin/bash
# P1-ENDTOEND-A01: Stop Local VM Cluster (graceful)
# Usage: ./stop-cluster.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../config"

if [ ! -f "$CONFIG_DIR/vm-specs.json" ]; then
  echo "No cluster configured."
  exit 0
fi

CLUSTER_DIR=$(jq -r '.cluster_dir' "$CONFIG_DIR/vm-specs.json")
HYPERVISOR=$(jq -r '.hypervisor' "$CONFIG_DIR/vm-specs.json")
NODES=$(jq -r '.nodes | length' "$CONFIG_DIR/vm-specs.json")

echo "=== P1-ENDTOEND-A01: Stop Cluster ==="
echo "Hypervisor: $HYPERVISOR"
echo "Nodes: $NODES"
echo ""

case "$HYPERVISOR" in
  qemu)
    for i in $(seq 1 $NODES); do
      VM_NAME="dh-local-$(printf '%02d' $i)"
      QEMU_PID_FILE="$CLUSTER_DIR/$VM_NAME/qemu.pid"

      if [ -f "$QEMU_PID_FILE" ]; then
        PID=$(cat "$QEMU_PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
          echo "Stopping $VM_NAME (PID: $PID)..."
          kill "$PID" 2>/dev/null || true
          sleep 2
        fi
      fi
    done
    ;;

  docker)
    for i in $(seq 1 $NODES); do
      VM_NAME="dh-local-$(printf '%02d' $i)"
      echo "Stopping $VM_NAME..."
      docker stop "$VM_NAME" 2>/dev/null || true
    done
    ;;

  *)
    echo "WARNING: Unsupported hypervisor."
    ;;
esac

echo ""
echo "=== Cluster Stopped ==="
echo "To restart: bash $SCRIPT_DIR/start-cluster.sh"
echo "To destroy: bash $SCRIPT_DIR/destroy-cluster.sh"
