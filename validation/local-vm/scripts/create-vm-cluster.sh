#!/bin/bash
# P1-ENDTOEND-A01: Create Local VM Cluster
# Supports: QEMU, UTM (macOS), VirtualBox, Docker
# Usage: ./create-vm-cluster.sh <hypervisor> [options]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../config"
CLOUD_INIT_DIR="$SCRIPT_DIR/../cloud-init"

# Defaults
HYPERVISOR="qemu"
NODES=3
CPU_PER_NODE=2
MEMORY_PER_NODE=4096  # MB
DISK_PER_NODE=50      # GB
NETWORK_MODE="host-only"
SSH_KEY_PATH="$HOME/.ssh/p1-local-vm"
UBUNTU_IMAGE_URL="https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"

# Parse arguments
parse_args() {
  HYPERVISOR="${1:-qemu}"
  shift || true

  while [[ $# -gt 0 ]]; do
    case $1 in
      --nodes) NODES="$2"; shift 2 ;;
      --cpu) CPU_PER_NODE="$2"; shift 2 ;;
      --memory) MEMORY_PER_NODE="$2"; shift 2 ;;
      --disk) DISK_PER_NODE="$2"; shift 2 ;;
      --network) NETWORK_MODE="$2"; shift 2 ;;
      --ssh-key) SSH_KEY_PATH="$2"; shift 2 ;;
      *) echo "Unknown option: $1"; exit 1 ;;
    esac
  done
}

echo "=== P1-ENDTOEND-A01: Local VM Cluster Setup ==="
echo "Hypervisor: $HYPERVISOR"
echo "Nodes: $NODES"
echo "CPU/Node: $CPU_PER_NODE cores"
echo "Memory/Node: $MEMORY_PER_NODE MB"
echo "Disk/Node: $DISK_PER_NODE GB"
echo "Network Mode: $NETWORK_MODE"
echo ""

# Create SSH key if needed
if [ ! -f "$SSH_KEY_PATH" ]; then
  echo "Creating SSH key: $SSH_KEY_PATH"
  mkdir -p "$(dirname "$SSH_KEY_PATH")"
  ssh-keygen -t rsa -b 4096 -f "$SSH_KEY_PATH" -N ""
  chmod 600 "$SSH_KEY_PATH"
  chmod 644 "$SSH_KEY_PATH.pub"
fi

SSH_PUBLIC_KEY=$(cat "$SSH_KEY_PATH.pub")
echo "SSH Public Key: $SSH_PUBLIC_KEY"
echo ""

# Create config directory
mkdir -p "$CONFIG_DIR"

case "$HYPERVISOR" in
  qemu)
    echo "Setting up QEMU cluster..."
    setup_qemu_cluster
    ;;
  qemu-no-kvm)
    echo "Setting up QEMU cluster (without KVM)..."
    QEMU_ENABLE_KVM=false
    setup_qemu_cluster
    ;;
  utm)
    echo "Setting up UTM cluster (macOS)..."
    setup_utm_cluster
    ;;
  virtualbox)
    echo "Setting up VirtualBox cluster..."
    setup_virtualbox_cluster
    ;;
  docker)
    echo "Setting up Docker-based cluster..."
    setup_docker_cluster
    ;;
  *)
    echo "ERROR: Unsupported hypervisor: $HYPERVISOR"
    exit 1
    ;;
esac

echo ""
echo "=== Cluster Setup Complete ==="
echo "To start cluster: bash $SCRIPT_DIR/start-cluster.sh"
echo "To bootstrap nodes: bash $SCRIPT_DIR/bootstrap-nodes.sh"
echo "To destroy cluster: bash $SCRIPT_DIR/destroy-cluster.sh"
