#!/bin/bash
# P1-ENDTOEND-A01: Collect Evidence from All Nodes
# Usage: ./collect-evidence.sh [output-dir]

INVENTORY="${1:-p1-inventory.json}"
OUTPUT_DIR="${2:-../evidence}"
SSH_KEY="${SSH_KEY:-~/.ssh/id_rsa}"
SSH_USER="ubuntu"

if [ ! -f "$INVENTORY" ]; then
  echo "ERROR: Inventory file not found: $INVENTORY"
  exit 1
fi

mkdir -p "$OUTPUT_DIR"

echo "=== P1-ENDTOEND-A01: Evidence Collection ==="
echo "Inventory: $INVENTORY"
echo "Output: $OUTPUT_DIR"
echo ""

ssh_exec() {
  local host=$1
  shift
  ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o ConnectTimeout=5 "$SSH_USER@$host" "$@" 2>/dev/null || echo ""
}

NODES=$(jq -r '.nodes[] | @base64' "$INVENTORY")

for node_b64 in $NODES; do
  NODE=$(echo "$node_b64" | base64 -d)
  NAME=$(echo "$NODE" | jq -r '.name')
  IP=$(echo "$NODE" | jq -r '.public_ip')

  echo "Collecting evidence from $NAME ($IP)..."

  NODE_DIR="$OUTPUT_DIR/$NAME"
  mkdir -p "$NODE_DIR"

  # Baseline evidence
  if ssh_exec "$IP" "test -f /var/log/decentralized-host/baseline-evidence.json"; then
    ssh_exec "$IP" "cat /var/log/decentralized-host/baseline-evidence.json" > "$NODE_DIR/baseline-evidence.json" 2>/dev/null || echo "{}"  > "$NODE_DIR/baseline-evidence.json"
    echo "  ✓ Baseline evidence"
  fi

  # Network evidence
  if ssh_exec "$IP" "test -f /var/log/decentralized-host/network-evidence.json"; then
    ssh_exec "$IP" "cat /var/log/decentralized-host/network-evidence.json" > "$NODE_DIR/network-evidence.json" 2>/dev/null || echo "{}" > "$NODE_DIR/network-evidence.json"
    echo "  ✓ Network evidence"
  fi

  # DHP evidence (if running)
  if ssh_exec "$IP" "curl -s http://localhost:8080/evidence 2>/dev/null" > "$NODE_DIR/dhp-evidence.jsonl" 2>/dev/null; then
    if [ -s "$NODE_DIR/dhp-evidence.jsonl" ]; then
      echo "  ✓ DHP evidence records"
    else
      rm "$NODE_DIR/dhp-evidence.jsonl"
    fi
  fi

  # System logs
  ssh_exec "$IP" "sudo journalctl -n 100 --no-pager" > "$NODE_DIR/systemd-logs.txt" 2>/dev/null || echo "" > "$NODE_DIR/systemd-logs.txt"
  echo "  ✓ System logs"

  # DHP logs (if available)
  if ssh_exec "$IP" "sudo test -f /var/log/decentralized-host/agent.log"; then
    ssh_exec "$IP" "sudo cat /var/log/decentralized-host/agent.log" > "$NODE_DIR/dhp-agent.log" 2>/dev/null || echo "" > "$NODE_DIR/dhp-agent.log"
    echo "  ✓ DHP agent logs"
  fi

  # Runtime metrics
  cat > "$NODE_DIR/collection-metadata.json" << EOF
{
  "node_name": "$NAME",
  "node_ip": "$IP",
  "collection_time_utc": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "test_phase": "P1-ENDTOEND-A01"
}
EOF
  echo "  ✓ Collection metadata"

  echo ""
done

echo "=== Evidence Collection Complete ==="
echo "Evidence saved to: $OUTPUT_DIR"
echo ""
echo "To validate signatures:"
echo "  cd $OUTPUT_DIR"
echo "  find . -name 'dhp-evidence.jsonl' -exec jq '.signature' {} \\;"
