#!/bin/bash
# P1-LOCAL-VM-A01: Create Local VM Cluster
# Single physical host, 3 distinct QEMU VMs
# Creates persistent cluster.json state file
# Usage: ./create-vm-cluster.sh qemu [--nodes N] [--cpu C] [--memory M] [--disk D]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
CLOUD_INIT_DIR="$REPO_ROOT/validation/local-vm/cloud-init"

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  arm64|aarch64)
    QEMU_ARCH="aarch64"
    QEMU_BINARY="qemu-system-aarch64"
    UBUNTU_ARCH="arm64"
    ;;
  x86_64|amd64)
    QEMU_ARCH="x86_64"
    QEMU_BINARY="qemu-system-x86_64"
    UBUNTU_ARCH="amd64"
    ;;
  *)
    echo "ERROR: Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Defaults
HYPERVISOR="${1:-qemu}"
NODES=3
CPU_PER_NODE=2
MEMORY_PER_NODE=4096
DISK_PER_NODE=50
SSH_KEY_PATH="$HOME/.ssh/p1-local-vm"
UBUNTU_IMAGE_URL="https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-${UBUNTU_ARCH}.img"

# Parse arguments
shift || true
while [[ $# -gt 0 ]]; do
  case $1 in
    --nodes) NODES="$2"; shift 2 ;;
    --cpu) CPU_PER_NODE="$2"; shift 2 ;;
    --memory) MEMORY_PER_NODE="$2"; shift 2 ;;
    --disk) DISK_PER_NODE="$2"; shift 2 ;;
    --ssh-key) SSH_KEY_PATH="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

echo "=== P1-LOCAL-VM-A01: Create Local VM Cluster ==="
echo "Host Architecture: $ARCH ($QEMU_ARCH)"
echo "QEMU Binary: $QEMU_BINARY"
echo "Hypervisor: $HYPERVISOR"
echo "Nodes: $NODES"
echo "CPU/Node: $CPU_PER_NODE cores"
echo "Memory/Node: $MEMORY_PER_NODE MB"
echo "Disk/Node: $DISK_PER_NODE GB"
echo ""

# Validate hypervisor
if [ "$HYPERVISOR" != "qemu" ] && [ "$HYPERVISOR" != "qemu-no-kvm" ]; then
  echo "ERROR: Only 'qemu' hypervisor is supported for P1-LOCAL-VM-A01"
  exit 1
fi

# Check prerequisites
if ! command -v "$QEMU_BINARY" &> /dev/null; then
  echo "ERROR: $QEMU_BINARY not found"
  echo "Cannot proceed. Install QEMU first."
  exit 1
fi

if ! command -v qemu-img &> /dev/null; then
  echo "ERROR: qemu-img not found"
  exit 1
fi

if ! command -v ssh-keygen &> /dev/null; then
  echo "ERROR: ssh-keygen not found"
  exit 1
fi

if ! command -v curl &> /dev/null; then
  echo "ERROR: curl not found"
  exit 1
fi

echo "✓ All prerequisites available"
echo ""

# Create/verify SSH key
if [ ! -f "$SSH_KEY_PATH" ]; then
  echo "Creating SSH key: $SSH_KEY_PATH"
  mkdir -p "$(dirname "$SSH_KEY_PATH")"
  ssh-keygen -t rsa -b 4096 -f "$SSH_KEY_PATH" -N ""
  chmod 600 "$SSH_KEY_PATH"
  chmod 644 "$SSH_KEY_PATH.pub"
fi

SSH_PUBLIC_KEY=$(cat "$SSH_KEY_PATH.pub")
echo "✓ SSH key ready"
echo ""

# Create state directory
mkdir -p "$STATE_DIR"

# Check port availability before creating anything
echo "Checking SSH port availability..."
for ((i=1; i<=NODES; i++)); do
  PORT=$((2200 + i))
  if lsof -nP -iTCP:$PORT -sTCP:LISTEN 2>/dev/null | grep -q LISTEN; then
    echo "ERROR: Port $PORT already in use"
    exit 1
  fi
done
echo "✓ All SSH ports available"
echo ""

# Download base Ubuntu image
echo "Setting up base Ubuntu image..."
UBUNTU_IMAGE="$STATE_DIR/ubuntu-${UBUNTU_ARCH}.img"

if [ ! -f "$UBUNTU_IMAGE" ]; then
  echo "Downloading Ubuntu cloud image from:"
  echo "  $UBUNTU_IMAGE_URL"

  if ! curl -L --progress-bar -o "$UBUNTU_IMAGE" "$UBUNTU_IMAGE_URL"; then
    echo "ERROR: Failed to download Ubuntu image"
    rm -f "$UBUNTU_IMAGE"
    exit 1
  fi
fi
echo "✓ Base image ready: $UBUNTU_IMAGE"
echo ""

# Create VM disks and cloud-init
echo "Creating $NODES VM disks and cloud-init configurations..."
SSH_PORTS=()

for ((i=1; i<=NODES; i++)); do
  NODE_DIR="$STATE_DIR/node-${i}"
  NODE_NAME="dh-node-${i}"
  SSH_PORT=$((2200 + i))
  SSH_PORTS+=("$SSH_PORT")

  mkdir -p "$NODE_DIR"

  echo "  Node $i: $NODE_NAME (SSH port $SSH_PORT)"

  # Create qcow2 disk overlay (copy-on-write, sparse)
  DISK="$NODE_DIR/disk.qcow2"
  if [ ! -f "$DISK" ]; then
    qemu-img create -f qcow2 -b "$UBUNTU_IMAGE" "$DISK" "${DISK_PER_NODE}G"
  fi

  # Generate cloud-init user-data
  cat > "$NODE_DIR/user-data" << CLOUD_INIT_EOF
#cloud-config
hostname: $NODE_NAME
fqdn: $NODE_NAME.local
preserve_hostname: true

users:
  - name: ubuntu
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh_authorized_keys:
      - $SSH_PUBLIC_KEY
    home: /home/ubuntu
    shell: /bin/bash

packages:
  - openssh-server
  - curl
  - jq
  - ca-certificates
  - systemd

runcmd:
  - mkdir -p /opt/decentralized-host /var/lib/decentralized-host /var/log/decentralized-host
  - echo "$NODE_NAME" > /etc/hostname
  - hostnamectl set-hostname "$NODE_NAME"
  - systemctl enable ssh
  - systemctl start ssh
  - echo "P1-LOCAL-VM-A01 Node: $NODE_NAME ready" > /var/log/cloud-init-success.log

power_state:
  mode: poweroff
  timeout: 0
  condition: False
CLOUD_INIT_EOF

  # Generate cloud-init meta-data
  cat > "$NODE_DIR/meta-data" << CLOUD_META_EOF
instance-id: $i
local-hostname: $NODE_NAME
CLOUD_META_EOF

  echo "    ✓ Disk: $DISK"
  echo "    ✓ Cloud-init: $NODE_DIR"
done

echo ""

# Create cluster.json state file (single source of truth)
CLUSTER_JSON="$STATE_DIR/cluster.json"

cat > "$CLUSTER_JSON" << JSON_EOF
{
  "qualification": "P1-LOCAL-VM-A01",
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "source_sha": "$(cd "$REPO_ROOT" && git rev-parse HEAD)",
  "hypervisor": "qemu",
  "host_architecture": "$ARCH",
  "guest_architecture": "$QEMU_ARCH",
  "qemu_binary": "$QEMU_BINARY",
  "nodes": $NODES,
  "cpu_per_node": $CPU_PER_NODE,
  "memory_per_node_mb": $MEMORY_PER_NODE,
  "disk_per_node_gb": $DISK_PER_NODE,
  "base_image": "$UBUNTU_IMAGE",
  "ssh_key_path": "$SSH_KEY_PATH",
  "state_directory": "$STATE_DIR",
  "status": "CREATED",
  "node_details": [
JSON_EOF

for ((i=1; i<=NODES; i++)); do
  NODE_DIR="$STATE_DIR/node-${i}"
  SSH_PORT=$((2200 + i))
  COMMA=$([[ $i -lt $NODES ]] && echo "," || echo "")

  cat >> "$CLUSTER_JSON" << JSON_NODE_EOF
    {
      "id": $i,
      "name": "dh-node-${i}",
      "disk": "$NODE_DIR/disk.qcow2",
      "cloud_init_dir": "$NODE_DIR",
      "ssh_port": $SSH_PORT,
      "qemu_pid_file": "$NODE_DIR/qemu.pid",
      "qemu_socket": "$NODE_DIR/qemu.sock",
      "serial_log": "$NODE_DIR/serial.log"
    }${COMMA}
JSON_NODE_EOF
done

cat >> "$CLUSTER_JSON" << JSON_END_EOF
  ]
}
JSON_END_EOF

echo "✓ Cluster configuration persisted: $CLUSTER_JSON"
echo ""

cat "$CLUSTER_JSON" | jq '.' 2>/dev/null || cat "$CLUSTER_JSON"

echo ""
echo "=== Cluster Creation Complete ==="
echo ""
echo "Next steps:"
echo "  1. Verify prerequisites:"
echo "     bash $SCRIPT_DIR/preflight.sh qemu"
echo ""
echo "  2. Start cluster:"
echo "     bash $SCRIPT_DIR/start-cluster.sh"
echo ""
echo "  3. Bootstrap nodes:"
echo "     bash $SCRIPT_DIR/bootstrap-nodes.sh"
echo ""
echo "State directory: $STATE_DIR"
