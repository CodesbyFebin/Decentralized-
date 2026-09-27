#!/bin/bash
# P1-ENDTOEND-A01: Start Local VM Cluster
# Usage: ./start-cluster.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../config"

if [ ! -f "$CONFIG_DIR/vm-specs.json" ]; then
  echo "ERROR: VM cluster not configured. Run create-vm-cluster.sh first."
  exit 1
fi

CLUSTER_DIR=$(jq -r '.cluster_dir' "$CONFIG_DIR/vm-specs.json")
HYPERVISOR=$(jq -r '.hypervisor' "$CONFIG_DIR/vm-specs.json")
NODES=$(jq -r '.nodes | length' "$CONFIG_DIR/vm-specs.json")

echo "=== P1-ENDTOEND-A01: Starting Cluster ==="
echo "Hypervisor: $HYPERVISOR"
echo "Nodes: $NODES"
echo "Cluster Directory: $CLUSTER_DIR"
echo ""

case "$HYPERVISOR" in
  qemu)
    echo "Starting QEMU VMs..."
    for i in $(seq 1 $NODES); do
      VM_NAME="dh-local-$(printf '%02d' $i)"
      QEMU_PID_FILE="$CLUSTER_DIR/$VM_NAME/qemu.pid"

      if [ -f "$QEMU_PID_FILE" ]; then
        PID=$(cat "$QEMU_PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
          echo "✓ $VM_NAME already running (PID: $PID)"
          continue
        fi
      fi

      DISK="$CLUSTER_DIR/$VM_NAME/disk.qcow2"
      VNC_PORT=$((5900 + i))

      echo "Starting $VM_NAME..."
      qemu-system-x86_64 \
        -enable-kvm \
        -cpu host \
        -smp $CPU_PER_NODE \
        -m $MEMORY_PER_NODE \
        -drive file=$DISK,format=qcow2 \
        -net user,hostfwd=tcp::$((2220 + i))-:22 \
        -net nic \
        -vnc :$(($i - 1)) \
        -daemonize \
        -pidfile "$QEMU_PID_FILE" \
        -name "$VM_NAME"

      echo "  VNC: localhost:$VNC_PORT"
      echo "  SSH: ssh -p $((2220 + i)) ubuntu@localhost"
    done
    ;;

  docker)
    echo "Starting Docker containers..."
    for i in $(seq 1 $NODES); do
      VM_NAME="dh-local-$(printf '%02d' $i)"
      echo "Starting $VM_NAME..."
      docker run -d \
        --name "$VM_NAME" \
        --hostname "$VM_NAME" \
        -p $((2220 + i)):22 \
        -e NODE_ID=$i \
        -v "$CLUSTER_DIR/$VM_NAME/data:/var/lib/decentralized-host" \
        -v "$CLUSTER_DIR/$VM_NAME/logs:/var/log/decentralized-host" \
        ubuntu:22.04 \
        sleep infinity

      echo "  SSH: ssh -p $((2220 + i)) ubuntu@localhost"
    done
    ;;

  *)
    echo "ERROR: Unsupported hypervisor: $HYPERVISOR"
    exit 1
    ;;
esac

echo ""
echo "=== Cluster Started ==="
echo "Waiting for nodes to boot (30 seconds)..."
sleep 30

echo ""
echo "To verify connectivity:"
echo "  bash $SCRIPT_DIR/bootstrap-nodes.sh"
echo ""
echo "To stop cluster:"
echo "  bash $SCRIPT_DIR/stop-cluster.sh"
