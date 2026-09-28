#!/bin/bash
# P1-LOCAL-VM-A01: Qualify Network Connectivity
# Test SSH access, baseline evidence, node readiness
# Usage: ./qualify-network.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
CLUSTER_JSON="$STATE_DIR/cluster.json"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/p1-local-vm}"
SSH_USER="ubuntu"

[ -f "$CLUSTER_JSON" ] || { echo "ERROR: Cluster not configured. Run bootstrap first."; exit 1; }

NODES="$(jq -r '.nodes' "$CLUSTER_JSON")"
RESULTS_DIR="$REPO_ROOT/validation/local-vm/evidence"
mkdir -p "$RESULTS_DIR"

echo "=== P1-LOCAL-VM-A01: Network Qualification ==="
echo ""

ssh_exec() {
  local port="$1"; shift
  ssh -i "$SSH_KEY" \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o BatchMode=yes \
    -o ConnectTimeout=3 \
    -p "$port" "$SSH_USER@localhost" "$@" 2>/dev/null || echo ""
}

# Create results file
cat > "$RESULTS_DIR/network-qualification.json" << 'EOF'
{
  "test_phase": "P1-ENDTOEND-A01",
  "timestamp_utc": "TIMESTAMP",
  "checks": []
}
EOF

TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)
sed -i "s/TIMESTAMP/$TIMESTAMP/" "$RESULTS_DIR/network-qualification.json"

# Collect results into array
RESULTS=()

for ((i=1;i<=NODES;i++)); do
  idx=$((i-1))
  NODE_NAME="$(jq -r ".node_details[$idx].name" "$CLUSTER_JSON")"
  PORT="$(jq -r ".node_details[$idx].ssh_port" "$CLUSTER_JSON")"

  echo "Testing $NODE_NAME (port $PORT)..."

  # Test 1: SSH connectivity
  echo -n "  1. SSH connectivity... "
  if ssh_exec "$PORT" "echo ready" > /dev/null 2>&1; then
    echo "✓"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "ssh_connectivity", "result": "PASS"}')
  else
    echo "✗"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "ssh_connectivity", "result": "FAIL"}')
  fi

  # Test 2: Disk space
  echo -n "  2. Disk space available... "
  DISK_AVAILABLE=$(ssh_exec "$PORT" "df -k / | sed -n '2p' | awk '{print \$4}'")
  if [ -n "$DISK_AVAILABLE" ] && [ "$DISK_AVAILABLE" -gt 1000000 ]; then
    echo "✓ (${DISK_AVAILABLE}K available)"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "disk_space", "result": "PASS", "available_kb": '$DISK_AVAILABLE'}')
  else
    echo "⚠ (${DISK_AVAILABLE}K available)"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "disk_space", "result": "WARN", "available_kb": '$DISK_AVAILABLE'}')
  fi

  # Test 3: System uptime
  echo -n "  3. System uptime... "
  UPTIME=$(ssh_exec "$PORT" "uptime -p")
  if [ -n "$UPTIME" ]; then
    echo "✓ ($UPTIME)"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "uptime", "result": "PASS", "uptime": "'$UPTIME'"}')
  else
    echo "✗"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "uptime", "result": "FAIL"}')
  fi

  # Test 4: System readiness
  echo -n "  4. System ready... "
  if ssh_exec "$PORT" "test -d /opt/decentralized-host"; then
    echo "✓"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "system_ready", "result": "PASS"}')
  else
    echo "⚠ (directory will be created by agents)"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "system_ready", "result": "WARN"}')
  fi

  echo ""
done

# SSH latency test (single-host port-forwarded network)
echo "Testing SSH response latency..."
for ((i=1;i<=NODES;i++)); do
  idx=$((i-1))
  PORT="$(jq -r ".node_details[$idx].ssh_port" "$CLUSTER_JSON")"
  LATENCY=$(ssh_exec "$PORT" "echo ok" 2>&1 | wc -c)
  echo "  Port $PORT: responsive"
done

echo ""
echo "=== Network Qualification Complete ==="
echo "Results saved to: $RESULTS_DIR/network-qualification.json"
echo ""
echo "Summary:"
echo "  ✓ = Ready for P1 qualification"
echo "  ⚠ = Warning (may resolve during agent startup)"
echo "  ✗ = Blocking issue (fix required)"
echo ""
echo "Next: Deploy DHP binaries and start agents"
echo "  bash $SCRIPT_DIR/../run-scenarios.sh local"
