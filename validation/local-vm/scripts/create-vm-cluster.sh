#!/bin/bash
# P1-LOCAL-VM-A01: Create Local VM Cluster
# Single physical host, 3 distinct VMs proving VM/OS/filesystem/node failure boundaries
# Supports: QEMU (Linux, macOS), others explicitly not supported in this phase
# Usage: ./create-vm-cluster.sh qemu [options]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../config"
CLOUD_INIT_DIR="$SCRIPT_DIR/../cloud-init"
CLUSTER_STATE_DIR="$REPO_ROOT/.p1-local-vm-state"

# Defaults
HYPERVISOR="${1:-qemu}"
NODES=3
CPU_PER_NODE=2
MEMORY_PER_NODE=4096  # MB
DISK_PER_NODE=50      # GB
SSH_KEY_PATH="$HOME/.ssh/p1-local-vm"

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

UBUNTU_IMAGE_URL="https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-${UBUNTU_ARCH}.img"

echo "=== P1-LOCAL-VM-A01: Local VM Cluster Setup ==="
echo "Host Architecture: $ARCH"
echo "QEMU Architecture: $QEMU_ARCH"
echo "Hypervisor: $HYPERVISOR"
echo "Nodes: $NODES"
echo "CPU/Node: $CPU_PER_NODE cores"
echo "Memory/Node: $MEMORY_PER_NODE MB"
echo "Disk/Node: $DISK_PER_NODE GB"
echo "Ubuntu Image: $UBUNTU_IMAGE_URL"
echo ""

# Check prerequisites
check_prerequisites() {
  echo "Checking prerequisites..."
  local missing=()

  if ! command -v "$QEMU_BINARY" &> /dev/null; then
    missing+=("$QEMU_BINARY")
  fi

  if ! command -v qemu-img &> /dev/null; then
    missing+=("qemu-img")
  fi

  if ! command -v ssh-keygen &> /dev/null; then
    missing+=("ssh-keygen")
  fi

  if ! command -v curl &> /dev/null; then
    missing+=("curl")
  fi

  if [ ${#missing[@]} -gt 0 ]; then
    echo "ERROR: Missing prerequisites: ${missing[*]}"
    echo ""
    echo "To install on macOS:"
    echo "  brew install qemu"
    echo ""
    echo "To install on Ubuntu/Debian:"
    echo "  sudo apt-get install qemu qemu-system qemu-system-x86-64 qemu-img curl"
    echo ""
    exit 1
  fi

  echo "✓ All prerequisites available"
}

# Create SSH key if needed
setup_ssh_key() {
  if [ ! -f "$SSH_KEY_PATH" ]; then
    echo "Creating SSH key: $SSH_KEY_PATH"
    mkdir -p "$(dirname "$SSH_KEY_PATH")"
    ssh-keygen -t rsa -b 4096 -f "$SSH_KEY_PATH" -N ""
    chmod 600 "$SSH_KEY_PATH"
    chmod 644 "$SSH_KEY_PATH.pub"
  fi

  SSH_PUBLIC_KEY=$(cat "$SSH_KEY_PATH.pub")
  echo "✓ SSH key ready: $SSH_KEY_PATH"
}

# Download Ubuntu cloud image
download_ubuntu_image() {
  local image_path="$CLUSTER_STATE_DIR/ubuntu-${UBUNTU_ARCH}.img"

  if [ -f "$image_path" ]; then
    echo "✓ Ubuntu image already present: $image_path"
    return
  fi

  echo "Downloading Ubuntu cloud image..."
  mkdir -p "$CLUSTER_STATE_DIR"

  if ! curl -L -o "$image_path" "$UBUNTU_IMAGE_URL"; then
    echo "ERROR: Failed to download Ubuntu image from $UBUNTU_IMAGE_URL"
    rm -f "$image_path"
    exit 1
  fi

  echo "✓ Ubuntu image downloaded: $image_path"
}

# Create individual VM disk from base image
create_vm_disk() {
  local node_id=$1
  local disk_path="$CLUSTER_STATE_DIR/node-${node_id}/disk.qcow2"
  local base_image="$CLUSTER_STATE_DIR/ubuntu-${UBUNTU_ARCH}.img"

  mkdir -p "$(dirname "$disk_path")"

  if [ -f "$disk_path" ]; then
    echo "  VM disk already exists: $disk_path"
    return
  fi

  # Create qcow2 overlay from base image (copy-on-write, sparse)
  qemu-img create -f qcow2 -b "$base_image" "$disk_path" "${DISK_PER_NODE}G"
  echo "  ✓ Created disk: $disk_path"
}

# Generate per-node cloud-init config
generate_cloud_init() {
  local node_id=$1
  local node_name="dh-node-${node_id}"
  local seed_dir="$CLUSTER_STATE_DIR/node-${node_id}/seed"

  mkdir -p "$seed_dir"

  # Substitute node-specific values into user-data
  cat > "$seed_dir/user-data" << EOF
#cloud-config
hostname: $node_name
fqdn: $node_name.local

users:
  - name: ubuntu
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh_authorized_keys:
      - $SSH_PUBLIC_KEY

packages:
  - openssh-server
  - curl
  - jq
  - cloud-utils

runcmd:
  - echo "P1-LOCAL-VM-A01 Node: $node_name" > /etc/motd
  - mkdir -p /opt/decentralized-host
  - mkdir -p /var/lib/decentralized-host
  - mkdir -p /var/log/decentralized-host
  - echo "$node_name" > /opt/decentralized-host/node-id
  - systemctl enable ssh
  - systemctl restart ssh
EOF

  # Generate empty meta-data for cloud-init
  cat > "$seed_dir/meta-data" << EOF
instance-id: $node_id
local-hostname: $node_name
EOF

  echo "  ✓ Generated cloud-init for node $node_id: $seed_dir"
}

# Setup QEMU cluster
setup_qemu_cluster() {
  echo ""
  echo "Setting up QEMU-based cluster ($QEMU_ARCH)..."

  check_prerequisites
  echo ""

  setup_ssh_key
  echo ""

  download_ubuntu_image
  echo ""

  echo "Creating VM disks and cloud-init..."
  for ((i=1; i<=NODES; i++)); do
    echo "  Node $i:"
    create_vm_disk "$i"
    generate_cloud_init "$i"
  done
  echo ""

  # Create cluster manifest
  cat > "$CLUSTER_STATE_DIR/manifest.json" << EOF
{
  "qualification": "P1-LOCAL-VM-A01",
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "host_arch": "$ARCH",
  "qemu_arch": "$QEMU_ARCH",
  "nodes": $NODES,
  "cpu_per_node": $CPU_PER_NODE,
  "memory_per_node_mb": $MEMORY_PER_NODE,
  "disk_per_node_gb": $DISK_PER_NODE,
  "base_image": "ubuntu-jammy-cloudimg-${UBUNTU_ARCH}",
  "ssh_key_path": "$SSH_KEY_PATH",
  "state_directory": "$CLUSTER_STATE_DIR",
  "nodes": [
EOF

  for ((i=1; i<=NODES; i++)); do
    local comma=$([[ $i -lt $NODES ]] && echo "," || echo "")
    cat >> "$CLUSTER_STATE_DIR/manifest.json" << EOF
    {
      "id": $i,
      "name": "dh-node-${i}",
      "disk": "$CLUSTER_STATE_DIR/node-${i}/disk.qcow2",
      "seed": "$CLUSTER_STATE_DIR/node-${i}/seed",
      "ssh_port": $((2200 + i))
    }${comma}
EOF
  done

  cat >> "$CLUSTER_STATE_DIR/manifest.json" << EOF
  ]
}
EOF

  echo "✓ Cluster manifest: $CLUSTER_STATE_DIR/manifest.json"
}

# Main dispatch
case "$HYPERVISOR" in
  qemu|qemu-no-kvm)
    setup_qemu_cluster
    ;;
  utm|virtualbox|docker)
    echo "ERROR: Hypervisor '$HYPERVISOR' not yet supported for P1-LOCAL-VM-A01"
    echo "Use 'qemu' for Linux/macOS (Intel/Apple Silicon)"
    exit 1
    ;;
  *)
    echo "ERROR: Unsupported hypervisor: $HYPERVISOR"
    echo "Supported: qemu"
    exit 1
    ;;
esac

echo ""
echo "=== Cluster Setup Complete ==="
echo "Cluster state: $CLUSTER_STATE_DIR"
echo ""
echo "Next steps:"
echo "  1. Review cluster manifest:"
echo "     cat $CLUSTER_STATE_DIR/manifest.json"
echo ""
echo "  2. Start cluster:"
echo "     bash $SCRIPT_DIR/start-cluster.sh"
echo ""
echo "  3. Bootstrap nodes:"
echo "     bash $SCRIPT_DIR/bootstrap-nodes.sh"
echo ""
echo "  4. Destroy cluster (when done):"
echo "     bash $SCRIPT_DIR/destroy-cluster.sh"
