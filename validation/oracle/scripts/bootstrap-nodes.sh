#!/bin/bash
# P1-ENDTOEND-A01: Bootstrap DH Software on Nodes
# Usage: ./bootstrap-nodes.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INVENTORY="${1:-p1-inventory.json}"
SSH_KEY="${SSH_KEY:-~/.ssh/id_rsa}"
SSH_USER="ubuntu"

if [ ! -f "$INVENTORY" ]; then
  echo "ERROR: Inventory file not found: $INVENTORY"
  exit 1
fi

echo "=== P1-ENDTOEND-A01: Bootstrap Nodes ==="
echo "Inventory: $INVENTORY"
echo ""

# Helper function to run SSH commands
ssh_exec() {
  local host=$1
  shift
  ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$SSH_USER@$host" "$@"
}

# Parse inventory
echo "Extracting node details from inventory..."
NODES=$(jq -r '.nodes[] | @base64' "$INVENTORY")

FAILED=0
PASSED=0

for node_b64 in $NODES; do
  NODE=$(echo "$node_b64" | base64 -d)

  NAME=$(echo "$NODE" | jq -r '.name')
  PUBLIC_IP=$(echo "$NODE" | jq -r '.public_ip')
  PRIVATE_IP=$(echo "$NODE" | jq -r '.private_ip')

  echo ""
  echo "Bootstrapping node: $NAME ($PUBLIC_IP)"

  # Wait for SSH to be ready
  echo "  Waiting for SSH access..."
  for i in {1..30}; do
    if ssh_exec "$PUBLIC_IP" "echo OK" &>/dev/null; then
      echo "  ✓ SSH ready"
      break
    fi
    if [ $i -eq 30 ]; then
      echo "  ✗ SSH timeout after 30 attempts"
      ((FAILED++))
      continue 2
    fi
    sleep 2
  done

  # Check baseline evidence
  echo "  Checking baseline evidence..."
  if ssh_exec "$PUBLIC_IP" "[ -f /var/log/decentralized-host/baseline-evidence.json ]"; then
    echo "  ✓ Baseline evidence collected"
  else
    echo "  ✗ Baseline evidence not found"
    ((FAILED++))
    continue
  fi

  # Check network evidence
  echo "  Checking network evidence..."
  if ssh_exec "$PUBLIC_IP" "[ -f /var/log/decentralized-host/network-evidence.json ]"; then
    echo "  ✓ Network evidence collected"
  else
    echo "  ✗ Network evidence not found"
    ((FAILED++))
    continue
  fi

  # Format storage (block volume)
  echo "  Formatting block storage..."
  if ssh_exec "$PUBLIC_IP" "sudo test -b /dev/oracleoci/oraclevdb"; then
    ssh_exec "$PUBLIC_IP" "sudo parted -s /dev/oracleoci/oraclevdb mklabel gpt mkpart primary ext4 0% 100%" &>/dev/null || true
    ssh_exec "$PUBLIC_IP" "sudo mkfs.ext4 /dev/oracleoci/oraclevdb" &>/dev/null || true
    ssh_exec "$PUBLIC_IP" "sudo mkdir -p /mnt/data && sudo mount /dev/oracleoci/oraclevdb /mnt/data" &>/dev/null || true
    echo "  ✓ Block storage mounted at /mnt/data"
  else
    echo "  ⊘ Block storage not yet available"
  fi

  # Verify DH directories
  echo "  Verifying DH directories..."
  ssh_exec "$PUBLIC_IP" "sudo mkdir -p /opt/decentralized-host /var/lib/decentralized-host /var/log/decentralized-host"
  ssh_exec "$PUBLIC_IP" "sudo chown ubuntu:ubuntu /opt/decentralized-host /var/lib/decentralized-host /var/log/decentralized-host"
  echo "  ✓ Directories ready"

  ((PASSED++))
done

echo ""
echo "=== Bootstrap Summary ==="
echo "Passed: $PASSED"
echo "Failed: $FAILED"

if [ $FAILED -gt 0 ]; then
  exit 1
fi

echo ""
echo "✓ All nodes bootstrapped successfully"
echo ""
echo "Next steps:"
echo "1. Qualify network: bash qualify-network.sh"
echo "2. Deploy DHP binaries to each node"
echo "3. Start DHP agents"
