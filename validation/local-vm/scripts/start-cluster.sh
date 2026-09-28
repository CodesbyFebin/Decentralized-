#!/bin/bash
# P1-LOCAL-VM-A01: Start Local VM Cluster
# Boots 3 distinct VMs with cloud-init configuration
# Usage: ./start-cluster.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CLUSTER_STATE_DIR="$REPO_ROOT/.p1-local-vm-state"

if [ ! -f "$CLUSTER_STATE_DIR/manifest.json" ]; then
  echo "ERROR: Cluster not initialized. Run create-vm-cluster.sh first."
  exit 1
fi

# Parse manifest
QEMU_ARCH=$(jq -r '.qemu_arch' "$CLUSTER_STATE_DIR/manifest.json")
NODES=$(jq -r '.nodes' "$CLUSTER_STATE_DIR/manifest.json")
CPU_PER_NODE=$(jq -r '.cpu_per_node' "$CLUSTER_STATE_DIR/manifest.json")
MEMORY_PER_NODE=$(jq -r '.memory_per_node_mb' "$CLUSTER_STATE_DIR/manifest.json")

# Determine QEMU binary
case "$QEMU_ARCH" in
  aarch64)
    QEMU_BINARY="qemu-system-aarch64"
    ;;
  x86_64)
    QEMU_BINARY="qemu-system-x86_64"
    ;;
  *)
    echo "ERROR: Unsupported QEMU architecture: $QEMU_ARCH"
    exit 1
    ;;
esac

echo "=== P1-LOCAL-VM-A01: Starting Cluster ==="
echo "QEMU Architecture: $QEMU_ARCH"
echo "QEMU Binary: $QEMU_BINARY"
echo "Nodes: $NODES"
echo "CPU/Node: $CPU_PER_NODE"
echo "Memory/Node: $MEMORY_PER_NODE MB"
echo "Cluster State: $CLUSTER_STATE_DIR"
echo ""

# Check if QEMU binary is available
if ! command -v "$QEMU_BINARY" &> /dev/null; then
  echo "ERROR: $QEMU_BINARY not found. Cannot start cluster."
  exit 1
fi

# Start each VM
echo "Starting QEMU VMs..."
for ((i=1; i<=NODES; i++)); do
  NODE_NAME=$(jq -r ".nodes[$((i-1))].name" "$CLUSTER_STATE_DIR/manifest.json")
  DISK=$(jq -r ".nodes[$((i-1))].disk" "$CLUSTER_STATE_DIR/manifest.json")
  SEED=$(jq -r ".nodes[$((i-1))].seed" "$CLUSTER_STATE_DIR/manifest.json")
  SSH_PORT=$(jq -r ".nodes[$((i-1))].ssh_port" "$CLUSTER_STATE_DIR/manifest.json")

  NODE_DIR="$(dirname "$DISK")"
  PID_FILE="$NODE_DIR/qemu.pid"

  # Check if already running
  if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if kill -0 "$PID" 2>/dev/null; then
      echo "✓ $NODE_NAME already running (PID: $PID)"
      continue
    fi
  fi

  echo "Starting $NODE_NAME (SSH port: $SSH_PORT)..."

  # Create cloud-init ISO if needed
  SEED_ISO="$NODE_DIR/seed.iso"
  if [ ! -f "$SEED_ISO" ]; then
    if command -v cloud-localds &> /dev/null; then
      cloud-localds -v "$SEED_ISO" "$SEED/user-data" "$SEED/meta-data"
    else
      # Fallback: create minimal ISO without cloud-localds
      mkisofs -output "$SEED_ISO" -volid cidata -joliet -rock "$SEED" 2>/dev/null || {
        echo "WARNING: Could not create cloud-init ISO. VMs will start but may not be configured."
      }
    fi
  fi

  # Build QEMU command based on architecture
  QEMU_CMD=(
    "$QEMU_BINARY"
    "-name" "$NODE_NAME"
    "-machine" "type=virt,accel=kvm"
    "-cpu" "host"
    "-smp" "$CPU_PER_NODE"
    "-m" "${MEMORY_PER_NODE}M"
    "-drive" "file=$DISK,format=qcow2,cache=writeback"
    "-net" "user,hostfwd=tcp::${SSH_PORT}-:22"
    "-net" "nic,model=virtio"
    "-nographic"
    "-daemonize"
    "-pidfile" "$PID_FILE"
  )

  # Add cloud-init seed if available
  if [ -f "$SEED_ISO" ]; then
    QEMU_CMD+=("-drive" "file=$SEED_ISO,format=raw,media=cdrom")
  fi

  # For Intel/x86_64 with KVM, add special boot options
  if [ "$QEMU_ARCH" = "x86_64" ]; then
    QEMU_CMD+=("-boot" "c")
  fi

  "${QEMU_CMD[@]}"

  echo "  SSH: ssh -i $HOME/.ssh/p1-local-vm -p $SSH_PORT ubuntu@localhost"
  echo "  PID file: $PID_FILE"
done

echo ""
echo "=== Cluster Started ==="
echo "Waiting for nodes to boot (15 seconds)..."
sleep 15

echo ""
echo "To check node status:"
echo "  for port in 2201 2202 2203; do ssh -i ~/.ssh/p1-local-vm -p \$port -o ConnectTimeout=2 ubuntu@localhost hostname 2>/dev/null && echo \"✓ Node on port \$port\" || echo \"✗ Node on port \$port not ready\"; done"
echo ""
echo "To verify cluster connectivity:"
echo "  bash $SCRIPT_DIR/bootstrap-nodes.sh"
echo ""
echo "To stop cluster:"
echo "  bash $SCRIPT_DIR/stop-cluster.sh"
echo ""
echo "To destroy cluster completely:"
echo "  bash $SCRIPT_DIR/destroy-cluster.sh"
