#!/bin/bash
# P1-ENDTOEND-A01: Network Qualification (latency, NTP, connectivity)
# Usage: ./qualify-network.sh

set -e

INVENTORY="${1:-p1-inventory.json}"
SSH_KEY="${SSH_KEY:-~/.ssh/id_rsa}"
SSH_USER="ubuntu"
LATENCY_TARGET_MS=50
NTP_OFFSET_TARGET_MS=100

if [ ! -f "$INVENTORY" ]; then
  echo "ERROR: Inventory file not found: $INVENTORY"
  exit 1
fi

echo "=== P1-ENDTOEND-A01: Network Qualification ==="
echo ""

ssh_exec() {
  local host=$1
  shift
  ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$SSH_USER@$host" "$@"
}

# Extract nodes
echo "Extracting nodes from inventory..."
NODES=$(jq -r '.nodes[] | @base64' "$INVENTORY")
NODES_ARRAY=()

for node_b64 in $NODES; do
  NODE=$(echo "$node_b64" | base64 -d)
  NAME=$(echo "$NODE" | jq -r '.name')
  IP=$(echo "$NODE" | jq -r '.public_ip')
  NODES_ARRAY+=("{\"name\": \"$NAME\", \"ip\": \"$IP\"}")
done

# 1. SSH Connectivity Check
echo "1. SSH Connectivity"
echo "   ─────────────────"
CONNECTIVITY_PASS=0
for node_b64 in $NODES; do
  NODE=$(echo "$node_b64" | base64 -d)
  NAME=$(echo "$NODE" | jq -r '.name')
  IP=$(echo "$NODE" | jq -r '.public_ip')

  if ssh_exec "$IP" "echo OK" &>/dev/null; then
    echo "   ✓ $NAME ($IP)"
    ((CONNECTIVITY_PASS++))
  else
    echo "   ✗ $NAME ($IP) - UNREACHABLE"
  fi
done
echo ""

# 2. Network Latency Check
echo "2. Network Latency (target: <${LATENCY_TARGET_MS}ms)"
echo "   ─────────────────────────────────"
LATENCY_PASS=0
for node_b64 in $NODES; do
  NODE=$(echo "$node_b64" | base64 -d)
  NAME=$(echo "$NODE" | jq -r '.name')
  IP=$(echo "$NODE" | jq -r '.public_ip')
  PRIVATE_IP=$(echo "$NODE" | jq -r '.private_ip')

  # Ping another node (pick first available)
  TARGET_IP=$(echo "$INVENTORY" | jq -r '.nodes[0].private_ip' 2>/dev/null || echo "")

  if [ -z "$TARGET_IP" ] || [ "$TARGET_IP" = "$PRIVATE_IP" ]; then
    # Use gateway or skip if only one node
    echo "   ⊘ $NAME - latency test requires 2+ nodes"
    continue
  fi

  LATENCY=$(ssh_exec "$IP" "ping -c 3 $TARGET_IP 2>/dev/null | tail -1 | awk -F'/' '{print int(\$5)}'" 2>/dev/null || echo "9999")

  if [ "$LATENCY" -lt "$LATENCY_TARGET_MS" ]; then
    echo "   ✓ $NAME: ${LATENCY}ms"
    ((LATENCY_PASS++))
  else
    echo "   ⚠ $NAME: ${LATENCY}ms (above ${LATENCY_TARGET_MS}ms target)"
  fi
done
echo ""

# 3. NTP Synchronization Check
echo "3. NTP Clock Synchronization (target: ±${NTP_OFFSET_TARGET_MS}ms)"
echo "   ──────────────────────────────────────────"
NTP_PASS=0
for node_b64 in $NODES; do
  NODE=$(echo "$node_b64" | base64 -d)
  NAME=$(echo "$NODE" | jq -r '.name')
  IP=$(echo "$NODE" | jq -r '.public_ip')

  NTP_STATUS=$(ssh_exec "$IP" "ntpstat 2>/dev/null | head -1" || echo "unsynchronised")

  if echo "$NTP_STATUS" | grep -q "synchronised"; then
    echo "   ✓ $NAME: $NTP_STATUS"
    ((NTP_PASS++))
  else
    echo "   ⚠ $NAME: $NTP_STATUS"
  fi
done
echo ""

# 4. Disk Space Check
echo "4. Disk Space (target: >20GB free)"
echo "   ────────────────────────────────"
DISK_PASS=0
for node_b64 in $NODES; do
  NODE=$(echo "$node_b64" | base64 -d)
  NAME=$(echo "$NODE" | jq -r '.name')
  IP=$(echo "$NODE" | jq -r '.public_ip')

  DISK_FREE=$(ssh_exec "$IP" "df / | tail -1 | awk '{print int(\$4/(1024*1024))}'" 2>/dev/null || echo "0")

  if [ "$DISK_FREE" -gt 20 ]; then
    echo "   ✓ $NAME: ${DISK_FREE}GB free"
    ((DISK_PASS++))
  else
    echo "   ✗ $NAME: ${DISK_FREE}GB free (need >20GB)"
  fi
done
echo ""

# 5. Containerd Runtime Check
echo "5. Container Runtime"
echo "   ──────────────────"
RUNTIME_PASS=0
for node_b64 in $NODES; do
  NODE=$(echo "$node_b64" | base64 -d)
  NAME=$(echo "$NODE" | jq -r '.name')
  IP=$(echo "$NODE" | jq -r '.public_ip')

  if ssh_exec "$IP" "sudo systemctl is-active containerd" &>/dev/null; then
    echo "   ✓ $NAME: containerd running"
    ((RUNTIME_PASS++))
  else
    echo "   ⚠ $NAME: containerd not running"
  fi
done
echo ""

# Summary
echo "=== Network Qualification Summary ==="
TOTAL_CHECKS=$((CONNECTIVITY_PASS + LATENCY_PASS + NTP_PASS + DISK_PASS + RUNTIME_PASS))
echo "Checks Passed: $TOTAL_CHECKS"

if [ $TOTAL_CHECKS -gt 0 ]; then
  echo "✓ Network qualifications complete"
else
  echo "✗ Network qualification failed"
  exit 1
fi
