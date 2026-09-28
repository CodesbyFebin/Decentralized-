#!/bin/bash
# P1-LOCAL-VM-A01: Start Local VM Cluster
# Reads cluster configuration from cluster.json
# Fails if backend mismatch detected
# Usage: ./start-cluster.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
CLUSTER_JSON="$STATE_DIR/cluster.json"

if [ ! -f "$CLUSTER_JSON" ]; then
  echo "ERROR: Cluster not configured."
  echo "Run create-vm-cluster.sh first:"
  echo "  bash $SCRIPT_DIR/create-vm-cluster.sh qemu"
  exit 1
fi

# Read cluster configuration
HYPERVISOR=$(jq -r '.hypervisor' "$CLUSTER_JSON")
NODES=$(jq -r '.nodes' "$CLUSTER_JSON")
QEMU_BINARY=$(jq -r '.qemu_binary' "$CLUSTER_JSON")
GUEST_ARCH=$(jq -r '.guest_architecture' "$CLUSTER_JSON")

echo "=== P1-LOCAL-VM-A01: Start Cluster ==="
echo "Status: Starting from cluster.json"
echo "Hypervisor: $HYPERVISOR"
echo "Nodes: $NODES"
echo "QEMU Binary: $QEMU_BINARY"
echo ""

# Enforce backend consistency
if [ "$HYPERVISOR" != "qemu" ]; then
  echo "ERROR: Cluster backend is '$HYPERVISOR', not QEMU"
  echo "Backend mismatch => FAIL CLOSED"
  exit 1
fi

# Verify QEMU is available
if ! command -v "$QEMU_BINARY" &> /dev/null; then
  echo "ERROR: $QEMU_BINARY not found"
  exit 1
fi

echo "✓ Backend consistency verified (qemu)"
echo ""

# Start each VM
echo "Starting $NODES QEMU VMs..."
for ((i=1; i<=NODES; i++)); do
  NODE_NAME=$(jq -r ".node_details[$((i-1))].name" "$CLUSTER_JSON")
  DISK=$(jq -r ".node_details[$((i-1))].disk" "$CLUSTER_JSON")
  CLOUD_INIT_DIR=$(jq -r ".node_details[$((i-1))].cloud_init_dir" "$CLUSTER_JSON")
  SSH_PORT=$(jq -r ".node_details[$((i-1))].ssh_port" "$CLUSTER_JSON")
  PID_FILE=$(jq -r ".node_details[$((i-1))].qemu_pid_file" "$CLUSTER_JSON")
  SERIAL_LOG=$(jq -r ".node_details[$((i-1))].serial_log" "$CLUSTER_JSON")

  NODE_DIR=$(dirname "$DISK")

  # Check if already running
  if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if kill -0 "$PID" 2>/dev/null; then
      echo "✓ $NODE_NAME already running (PID: $PID)"
      continue
    else
      # Stale PID file
      rm -f "$PID_FILE"
    fi
  fi

  echo "Starting $NODE_NAME (SSH port $SSH_PORT)..."

  # Create cloud-init seed ISO
  SEED_ISO="$NODE_DIR/seed.iso"
  if [ ! -f "$SEED_ISO" ]; then
    if command -v cloud-localds &> /dev/null; then
      cloud-localds -v "$SEED_ISO" "$CLOUD_INIT_DIR/user-data" "$CLOUD_INIT_DIR/meta-data" 2>/dev/null
    else
      # Fallback: try mkisofs
      if command -v mkisofs &> /dev/null; then
        mkisofs -output "$SEED_ISO" -volid cidata -joliet -rock \
          "$CLOUD_INIT_DIR/user-data" "$CLOUD_INIT_DIR/meta-data" 2>/dev/null || {
          echo "  WARNING: Could not create cloud-init ISO"
        }
      fi
    fi
  fi

  # Build QEMU command
  QEMU_ARGS=(
    "$QEMU_BINARY"
    "-name" "$NODE_NAME"
    "-machine" "type=virt,accel=kvm"
    "-cpu" "host"
    "-smp" "$(jq -r '.cpu_per_node' "$CLUSTER_JSON")"
    "-m" "$(jq -r '.memory_per_node_mb' "$CLUSTER_JSON")"
    "-drive" "file=$DISK,format=qcow2,cache=writeback"
    "-net" "user,hostfwd=tcp::${SSH_PORT}-:22"
    "-net" "nic,model=virtio"
    "-nographic"
    "-daemonize"
    "-pidfile" "$PID_FILE"
    "-serial" "file:$SERIAL_LOG"
  )

  # Add cloud-init seed if available
  if [ -f "$SEED_ISO" ]; then
    QEMU_ARGS+=("-drive" "file=$SEED_ISO,format=raw,media=cdrom")
  fi

  # Execute QEMU
  "${QEMU_ARGS[@]}"

  echo "  ✓ Started (SSH port $SSH_PORT)"
  echo "  ssh -i ~/.ssh/p1-local-vm -p $SSH_PORT ubuntu@localhost"
done

echo ""
echo "=== VMs Started ==="
echo "Waiting for boot (30 seconds)..."
sleep 30

echo ""
echo "To verify SSH access:"
echo "  bash $SCRIPT_DIR/bootstrap-nodes.sh"
echo ""
echo "To stop cluster:"
echo "  bash $SCRIPT_DIR/stop-cluster.sh"
echo ""
echo "To destroy cluster:"
echo "  bash $SCRIPT_DIR/destroy-cluster.sh"
