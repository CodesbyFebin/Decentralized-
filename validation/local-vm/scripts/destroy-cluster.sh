#!/bin/bash
# P1-ENDTOEND-A01: Destroy Local VM Cluster
# Usage: ./destroy-cluster.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../config"

if [ ! -f "$CONFIG_DIR/vm-specs.json" ]; then
  echo "No cluster configured."
  exit 0
fi

CLUSTER_DIR=$(jq -r '.cluster_dir' "$CONFIG_DIR/vm-specs.json")
HYPERVISOR=$(jq -r '.hypervisor' "$CONFIG_DIR/vm-specs.json")
NODES=$(jq -r '.nodes | length' "$CONFIG_DIR/vm-specs.json")

echo "=== P1-ENDTOEND-A01: Destroy Cluster ==="
echo "Hypervisor: $HYPERVISOR"
echo "Nodes: $NODES"
echo "Cluster Directory: $CLUSTER_DIR"
echo ""
echo "⚠️  WARNING: This will destroy all VMs and local data!"
echo "⚠️  This action cannot be undone."
echo ""
read -p "Type 'yes' to confirm destruction: " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
  echo "Cancelled."
  exit 0
fi

echo ""
echo "Destroying VMs..."

case "$HYPERVISOR" in
  qemu)
    for i in $(seq 1 $NODES); do
      VM_NAME="dh-local-$(printf '%02d' $i)"
      QEMU_PID_FILE="$CLUSTER_DIR/$VM_NAME/qemu.pid"

      if [ -f "$QEMU_PID_FILE" ]; then
        PID=$(cat "$QEMU_PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
          echo "Stopping $VM_NAME (PID: $PID)..."
          kill -9 "$PID" 2>/dev/null || true
          sleep 1
        fi
        rm -f "$QEMU_PID_FILE"
      fi
    done
    ;;

  docker)
    for i in $(seq 1 $NODES); do
      VM_NAME="dh-local-$(printf '%02d' $i)"
      echo "Removing $VM_NAME..."
      docker rm -f "$VM_NAME" 2>/dev/null || true
    done
    ;;

  *)
    echo "WARNING: Unsupported hypervisor. Manual cleanup may be needed."
    ;;
esac

echo ""
echo "Removing cluster directory: $CLUSTER_DIR"
rm -rf "$CLUSTER_DIR" 2>/dev/null || true

echo ""
echo "=== Cleanup Complete ==="
echo "Cluster has been destroyed."
echo "Evidence files (if backed up) are still available in: $SCRIPT_DIR/../evidence/"
