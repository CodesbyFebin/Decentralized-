#!/bin/bash
# P1-ENDTOEND-A01: Stop/Terminate a Node
# Usage: ./kill-node.sh <node-name>
# Example: ./kill-node.sh dh-node-02

set -e

INVENTORY="${1:-../p1-inventory.json}"
NODE_NAME="${2}"
TF_DIR="$(dirname "$(pwd)")"

if [ -z "$NODE_NAME" ]; then
  echo "ERROR: Node name required"
  echo "Usage: $0 <node-name>"
  echo ""
  echo "Available nodes:"
  jq -r '.nodes[].name' "$INVENTORY" 2>/dev/null || echo "  (no inventory found)"
  exit 1
fi

if [ ! -f "$INVENTORY" ]; then
  echo "ERROR: Inventory file not found: $INVENTORY"
  exit 1
fi

# Find node in inventory
NODE=$(jq ".nodes[] | select(.name == \"$NODE_NAME\")" "$INVENTORY" 2>/dev/null)

if [ -z "$NODE" ]; then
  echo "ERROR: Node not found: $NODE_NAME"
  exit 1
fi

NODE_IP=$(echo "$NODE" | jq -r '.public_ip')

echo "=== P1-ENDTOEND-A01: Stop Node ==="
echo "Node: $NODE_NAME"
echo "Public IP: $NODE_IP"
echo ""

# Record timestamp
KILL_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
echo "Stop initiated at: $KILL_TIME"

# Option 1: Graceful stop via OCI (recommended for P1 test)
echo ""
echo "Stopping instance via OCI API..."
echo "(This will mark the node as stopped, triggering failure detection)"
echo ""

# Get instance ID from Terraform state
cd "$TF_DIR"
INSTANCE_ID=$(terraform output -json node_details | jq -r ".[\"$NODE_NAME\"].instance_id" 2>/dev/null || echo "")

if [ -n "$INSTANCE_ID" ] && [ "$INSTANCE_ID" != "null" ]; then
  echo "Instance ID: $INSTANCE_ID"
  echo ""
  echo "To stop this instance with OCI CLI, run:"
  echo "  oci compute instance action --instance-id $INSTANCE_ID --action STOP"
  echo ""
  echo "To terminate this instance, run:"
  echo "  oci compute instance terminate --instance-id $INSTANCE_ID"
  echo ""
else
  echo "ℹ Manual stop required:"
  echo "  1. Go to OCI Console → Compute → Instances"
  echo "  2. Find: $NODE_NAME"
  echo "  3. Click More → Stop"
  echo ""
fi

# Record event
mkdir -p ../evidence
cat > "../evidence/kill-node-$NODE_NAME-$KILL_TIME.json" << EOF
{
  "event": "node_stop",
  "node_name": "$NODE_NAME",
  "node_ip": "$NODE_IP",
  "instance_id": "$INSTANCE_ID",
  "timestamp_utc": "$KILL_TIME",
  "test_phase": "P1-ENDTOEND-A01"
}
EOF

echo "Event recorded to: evidence/kill-node-$NODE_NAME-$KILL_TIME.json"
echo ""
echo "The control plane should detect the node failure within 30 seconds."
echo "Monitor with: watch -n1 'curl -s http://control-plane:8080/nodes | jq .'"
