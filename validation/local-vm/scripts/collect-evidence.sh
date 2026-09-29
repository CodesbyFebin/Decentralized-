#!/bin/bash
# P1-ENDTOEND-A01: Collect Evidence from Local Nodes
# Usage: ./collect-evidence.sh [output-dir]

OUTPUT_DIR="${1:-../evidence}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/p1-local-vm}"
SSH_USER="ubuntu"
NODES=3

mkdir -p "$OUTPUT_DIR"

echo "=== P1-ENDTOEND-A01: Evidence Collection ==="
echo "Output: $OUTPUT_DIR"
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

for i in $(seq 1 $NODES); do
  NODE_NAME="dh-local-$(printf '%02d' $i)"
  PORT=$((2220 + i))

  echo "Collecting evidence from $NODE_NAME (port $PORT)..."

  NODE_DIR="$OUTPUT_DIR/$NODE_NAME"
  mkdir -p "$NODE_DIR"

  # Baseline evidence
  if ssh_exec "$i" "test -f /var/log/decentralized-host/baseline-evidence.json"; then
    ssh_exec "$i" "cat /var/log/decentralized-host/baseline-evidence.json" > "$NODE_DIR/baseline-evidence.json" 2>/dev/null || echo "{}" > "$NODE_DIR/baseline-evidence.json"
    echo "  ✓ Baseline evidence"
  fi

  # Network evidence
  if ssh_exec "$i" "test -f /var/log/decentralized-host/network-evidence.json"; then
    ssh_exec "$i" "cat /var/log/decentralized-host/network-evidence.json" > "$NODE_DIR/network-evidence.json" 2>/dev/null || echo "{}" > "$NODE_DIR/network-evidence.json"
    echo "  ✓ Network evidence"
  fi

  # DHP evidence (if running)
  if ssh_exec "$i" "curl -s http://localhost:8080/evidence 2>/dev/null" > "$NODE_DIR/dhp-evidence.jsonl" 2>/dev/null; then
    if [ -s "$NODE_DIR/dhp-evidence.jsonl" ]; then
      echo "  ✓ DHP evidence records"
    else
      rm -f "$NODE_DIR/dhp-evidence.jsonl"
    fi
  fi

  # System logs
  ssh_exec "$i" "journalctl -n 100 --no-pager" > "$NODE_DIR/systemd-logs.txt" 2>/dev/null || echo "" > "$NODE_DIR/systemd-logs.txt"
  echo "  ✓ System logs"

  # DHP logs (if available)
  if ssh_exec "$i" "sudo test -f /var/log/decentralized-host/agent.log"; then
    ssh_exec "$i" "sudo cat /var/log/decentralized-host/agent.log" > "$NODE_DIR/dhp-agent.log" 2>/dev/null || echo "" > "$NODE_DIR/dhp-agent.log"
    echo "  ✓ DHP agent logs"
  fi

  # Collection metadata
  cat > "$NODE_DIR/collection-metadata.json" << EOF
{
  "node_name": "$NODE_NAME",
  "node_ssh_port": $PORT,
  "collection_time_utc": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "test_phase": "P1-ENDTOEND-A01",
  "hypervisor": "local-vm"
}
EOF
  echo "  ✓ Collection metadata"

  echo ""
done

echo "=== Evidence Collection Complete ==="
echo "Evidence saved to: $OUTPUT_DIR"
echo ""
echo "Evidence files per node:"
echo "  - baseline-evidence.json (hardware facts)"
echo "  - network-evidence.json (interfaces, routes, DNS)"
echo "  - dhp-evidence.jsonl (agent observations)"
echo "  - systemd-logs.txt (kernel logs)"
echo "  - dhp-agent.log (DHP agent logs)"
echo "  - collection-metadata.json (collection context)"
echo ""
echo "To validate signatures:"
echo "  cd $OUTPUT_DIR"
echo "  find . -name 'dhp-evidence.jsonl' -exec jq '.signature' {} \;"
