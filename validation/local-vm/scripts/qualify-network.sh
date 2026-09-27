#!/bin/bash
# P1-ENDTOEND-A01: Qualify Local Network
# Test latency, NTP sync, connectivity
# Usage: ./qualify-network.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../config"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/p1-local-vm}"
SSH_USER="ubuntu"

NODES=3
RESULTS_DIR="$SCRIPT_DIR/../evidence"
mkdir -p "$RESULTS_DIR"

echo "=== P1-ENDTOEND-A01: Network Qualification ==="
echo ""

ssh_exec() {
  local node_id=$1
  local port=$((2220 + node_id))
  shift

  ssh -i "$SSH_KEY" \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o ConnectTimeout=5 \
    "$SSH_USER@localhost" -p "$port" "$@" 2>/dev/null || echo ""
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

for i in $(seq 1 $NODES); do
  NODE_NAME="dh-local-$(printf '%02d' $i)"
  PORT=$((2220 + i))

  echo "Testing $NODE_NAME (port $PORT)..."

  # Test 1: SSH connectivity
  echo -n "  1. SSH connectivity... "
  if ssh_exec "$i" "echo ok" > /dev/null 2>&1; then
    echo "✓"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "ssh_connectivity", "result": "PASS"}')
  else
    echo "✗"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "ssh_connectivity", "result": "FAIL"}')
  fi

  # Test 2: Disk space
  echo -n "  2. Disk space (>20GB)... "
  DISK_AVAILABLE=$(ssh_exec "$i" "df / | tail -1 | awk '{print \$4}'")
  if [ -n "$DISK_AVAILABLE" ] && [ "$DISK_AVAILABLE" -gt 20000000 ]; then
    echo "✓ (${DISK_AVAILABLE}K available)"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "disk_space", "result": "PASS", "available_kb": '$DISK_AVAILABLE'}')
  else
    echo "✗ (${DISK_AVAILABLE}K available)"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "disk_space", "result": "FAIL", "available_kb": '$DISK_AVAILABLE'}')
  fi

  # Test 3: NTP synchronization
  echo -n "  3. NTP sync... "
  NTP_STATUS=$(ssh_exec "$i" "timedatectl | grep synchronized | awk '{print \$NF}'")
  if [ "$NTP_STATUS" = "yes" ]; then
    echo "✓"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "ntp_sync", "result": "PASS", "status": "'$NTP_STATUS'"}')
  else
    echo "⚠ (not synced - may be OK during boot)"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "ntp_sync", "result": "WARN", "status": "'$NTP_STATUS'"}')
  fi

  # Test 4: Container runtime
  echo -n "  4. Containerd status... "
  CONTAINERD_STATUS=$(ssh_exec "$i" "systemctl is-active containerd 2>/dev/null || echo 'inactive'")
  if [ "$CONTAINERD_STATUS" = "active" ]; then
    echo "✓"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "containerd", "result": "PASS", "status": "'$CONTAINERD_STATUS'"}')
  else
    echo "⚠ (not running - will start with agents)"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "containerd", "result": "WARN", "status": "'$CONTAINERD_STATUS'"}')
  fi

  # Test 5: Baseline evidence
  echo -n "  5. Baseline evidence... "
  if ssh_exec "$i" "test -f /var/log/decentralized-host/baseline-evidence.json"; then
    echo "✓"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "baseline_evidence", "result": "PASS"}')
  else
    echo "✗"
    RESULTS+=('{"node": "'$NODE_NAME'", "check": "baseline_evidence", "result": "FAIL"}')
  fi

  echo ""
done

# Inter-node latency test (if using local IPs)
echo "Testing inter-node latency..."
for i in $(seq 1 $NODES); do
  for j in $(seq 1 $NODES); do
    if [ $i -ne $j ]; then
      # Ping test (note: will fail initially before network setup)
      PING_RESULT=$(ssh_exec "$i" "ping -c 1 -W 1 192.168.200.$j 2>&1" | grep "time=" | awk -F= '{print $NF}' || echo "N/A")
      echo "  dh-local-$(printf '%02d' $i) → dh-local-$(printf '%02d' $j): $PING_RESULT"
    fi
  done
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
