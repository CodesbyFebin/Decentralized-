#!/bin/bash
# P1-LOCAL-VM-A01: Create Local VM Cluster
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
TOPOLOGY_FILE="$STATE_DIR/topology.sh"

HYPERVISOR="${1:-qemu}"
shift || true
NODES=""
CPU_PER_NODE=""
MEMORY_PER_NODE=""
DISK_PER_NODE=""
SSH_KEY_PATH="$HOME/.ssh/p1-local-vm"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --nodes) NODES="$2"; shift 2 ;;
    --cpu) CPU_PER_NODE="$2"; shift 2 ;;
    --memory) MEMORY_PER_NODE="$2"; shift 2 ;;
    --disk) DISK_PER_NODE="$2"; shift 2 ;;
    --ssh-key) SSH_KEY_PATH="$2"; shift 2 ;;
    *) echo "ERROR: Unknown option: $1" >&2; exit 1 ;;
  esac
done

[ "$HYPERVISOR" = "qemu" ] || { echo "ERROR: P1-LOCAL-VM-A01 requires qemu" >&2; exit 1; }
[ -f "$TOPOLOGY_FILE" ] || { echo "ERROR: Missing topology.sh. Run preflight first." >&2; exit 1; }

# shellcheck disable=SC1090
source "$TOPOLOGY_FILE"
[ "${PREFLIGHT_STATUS:-}" = "PASS" ] || { echo "ERROR: Topology has no successful preflight marker" >&2; exit 1; }
[ "${PREFLIGHT_SOURCE_SHA:-}" = "$(git -C "$REPO_ROOT" rev-parse HEAD)" ] || { echo "ERROR: Preflight source SHA differs from current source" >&2; exit 1; }

REQ_NODES="${NODES:-}"
REQ_CPU="${CPU_PER_NODE:-}"
REQ_MEM="${MEMORY_PER_NODE:-}"
REQ_DISK="${DISK_PER_NODE:-}"

for v in REQ_NODES REQ_CPU REQ_MEM REQ_DISK; do
  eval "val=\${$v}"
  case "$val" in ''|*[!0-9]*|0) echo "ERROR: Invalid persisted topology: $v=$val" >&2; exit 1 ;; esac
done
[ "$REQ_NODES" -le 254 ] || { echo "ERROR: Mesh address plan supports at most 254 nodes" >&2; exit 1; }

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) QEMU_ARCH="x86_64"; QEMU_BINARY="qemu-system-x86_64"; UBUNTU_ARCH="amd64" ;;
  arm64|aarch64) QEMU_ARCH="aarch64"; QEMU_BINARY="qemu-system-aarch64"; UBUNTU_ARCH="arm64" ;;
  *) echo "ERROR: Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

command -v "$QEMU_BINARY" >/dev/null || { echo "ERROR: $QEMU_BINARY not found" >&2; exit 1; }
command -v qemu-img >/dev/null || { echo "ERROR: qemu-img not found" >&2; exit 1; }
command -v jq >/dev/null || { echo "ERROR: jq not found" >&2; exit 1; }
command -v curl >/dev/null || { echo "ERROR: curl not found" >&2; exit 1; }

# refuse existing live or stale cluster state
if [ -f "$STATE_DIR/cluster.json" ]; then
  status="$(jq -r '.status // "UNKNOWN"' "$STATE_DIR/cluster.json" 2>/dev/null || echo UNKNOWN)"
  echo "ERROR: Existing cluster state detected (status=$status). Destroy it first." >&2
  exit 1
fi

for ((i=1;i<=REQ_NODES;i++)); do
  port=$((2200+i))
  if lsof -nP -iTCP:"$port" -sTCP:LISTEN 2>/dev/null | grep -q LISTEN; then
    echo "ERROR: Port $port already in use" >&2
    exit 1
  fi
done

mkdir -p "$STATE_DIR"
if [ ! -f "$SSH_KEY_PATH" ]; then
  mkdir -p "$(dirname "$SSH_KEY_PATH")"
  ssh-keygen -t rsa -b 4096 -f "$SSH_KEY_PATH" -N ""
fi
SSH_PUBLIC_KEY="$(cat "$SSH_KEY_PATH.pub")"

UBUNTU_IMAGE_URL="https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-${UBUNTU_ARCH}.img"
UBUNTU_IMAGE="$STATE_DIR/ubuntu-${UBUNTU_ARCH}.img"
if [ ! -f "$UBUNTU_IMAGE" ]; then
  curl -fL --progress-bar -o "$UBUNTU_IMAGE.tmp" "$UBUNTU_IMAGE_URL"
  mv "$UBUNTU_IMAGE.tmp" "$UBUNTU_IMAGE"
fi

echo "=== P1-LOCAL-VM-A01: Create Local VM Cluster ==="
echo "Topology source: $TOPOLOGY_FILE"
echo "Nodes: $REQ_NODES"
echo "CPU/Node: $REQ_CPU"
echo "Memory/Node: $REQ_MEM MiB"
echo "Disk/Node: $REQ_DISK GiB"

for ((i=1;i<=REQ_NODES;i++)); do
  NODE_DIR="$STATE_DIR/node-$i"
  NODE_NAME="dh-node-$i"
  printf -v PRIMARY_MAC '52:54:00:10:00:%02x' "$i"
  printf -v MESH_MAC '52:54:00:20:00:%02x' "$i"
  MESH_IP="172.30.10.$i"
  mkdir -p "$NODE_DIR"
  DISK="$NODE_DIR/disk.qcow2"
  qemu-img create -f qcow2 -b "$UBUNTU_IMAGE" -F qcow2 "$DISK" "${REQ_DISK}G"

  # Write user-data with proper cloud-init YAML format and safe key injection
  {
    printf '#cloud-config\n'
    printf 'hostname: %s\n' "$NODE_NAME"
    printf 'fqdn: %s.local\n' "$NODE_NAME"
    printf 'preserve_hostname: true\n'
    printf 'ssh_pwauth: false\n'
    printf 'users:\n'
    printf '  - default\n'
    printf '  - name: ubuntu\n'
    printf '    sudo: ALL=(ALL) NOPASSWD:ALL\n'
    printf '    groups: [adm, sudo]\n'
    printf '    shell: /bin/bash\n'
    printf '    lock_passwd: true\n'
    printf '    ssh_authorized_keys:\n'
    printf '      - %s\n' "$SSH_PUBLIC_KEY"
    printf 'package_update: false\n'
    printf 'packages:\n'
    printf '  - openssh-server\n'
    printf '  - iproute2\n'
    printf 'runcmd:\n'
    printf '  - systemctl enable ssh\n'
    printf '  - systemctl restart ssh\n'
    printf '  - mkdir -p /opt/decentralized-host /var/lib/decentralized-host /var/log/decentralized-host\n'
    printf '  - echo "P1-LOCAL-VM-A01 Node: %s ready" > /var/log/cloud-init-success.log\n' "$NODE_NAME"
  } > "$NODE_DIR/user-data"

  cat > "$NODE_DIR/meta-data" <<EOF
instance-id: p1-local-vm-a01-node-$i
local-hostname: $NODE_NAME
EOF
  cat > "$NODE_DIR/network-config" <<EOF
version: 2
ethernets:
  primary:
    match:
      macaddress: "$PRIMARY_MAC"
    dhcp4: true
  mesh:
    match:
      macaddress: "$MESH_MAC"
    set-name: mesh0
    dhcp4: false
    addresses:
      - $MESH_IP/24
EOF
done

CLUSTER_JSON="$STATE_DIR/cluster.json"
CURRENT_SHA="$(git -C "$REPO_ROOT" rev-parse HEAD)"
{
  cat <<EOF
{
  "schema_version": 2,
  "qualification": "P1-LOCAL-VM-A01",
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "source_sha": "$CURRENT_SHA",
  "hypervisor": "qemu",
  "host_architecture": "$ARCH",
  "guest_architecture": "$QEMU_ARCH",
  "qemu_binary": "$QEMU_BINARY",
  "nodes": $REQ_NODES,
  "cpu_per_node": $REQ_CPU,
  "memory_per_node_mb": $REQ_MEM,
  "disk_per_node_gb": $REQ_DISK,
  "base_image": "$UBUNTU_IMAGE",
  "mesh_backend": "qemu-socket-multicast",
  "mesh_bus": "239.192.42.17:12347",
  "ssh_key_path": "$SSH_KEY_PATH",
  "state_directory": "$STATE_DIR",
  "status": "CREATED",
  "node_details": [
EOF
  for ((i=1;i<=REQ_NODES;i++)); do
    comma=","; [ "$i" -eq "$REQ_NODES" ] && comma=""
    printf -v PRIMARY_MAC '52:54:00:10:00:%02x' "$i"
    printf -v MESH_MAC '52:54:00:20:00:%02x' "$i"
    cat <<EOF
    {
      "id": $i,
      "name": "dh-node-$i",
      "disk": "$STATE_DIR/node-$i/disk.qcow2",
      "cloud_init_dir": "$STATE_DIR/node-$i",
      "ssh_port": $((2200+i)),
      "primary_mac": "$PRIMARY_MAC",
      "mesh_mac": "$MESH_MAC",
      "mesh_ip": "172.30.10.$i",
      "qemu_pid_file": "$STATE_DIR/node-$i/qemu.pid",
      "qemu_socket": "$STATE_DIR/node-$i/qemu.sock",
      "serial_log": "$STATE_DIR/node-$i/serial.log"
    }$comma
EOF
  done
  echo "  ]"
  echo "}"
} > "$CLUSTER_JSON"

jq empty "$CLUSTER_JSON"
echo "STATUS: CREATED"
echo "Cluster configuration: $CLUSTER_JSON"
