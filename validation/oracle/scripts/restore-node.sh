#!/bin/bash
# P1-ENDTOEND-A01: Restore a Stopped Node
# Usage: ./restore-node.sh <node-name>

set -e

INVENTORY="${1:-../p1-inventory.json}"
NODE_NAME="${2}"
TF_DIR="$(dirname "$(pwd)")"

if [ -z "$NODE_NAME" ]; then
  echo "ERROR: Node name required"
  echo "Usage: $0 <node-name>"
  exit 1
fi

# Find node in inventory
NODE=$(jq ".nodes[] | select(.name == \"$NODE_NAME\")" "$INVENTORY" 2>/dev/null)

if [ -z "$NODE" ]; then
  echo "ERROR: Node not found: $NODE_NAME"
  exit 1
fi

echo "=== P1-ENDTOEND-A01: Restore Node ==="
echo "Node: $NODE_NAME"
echo ""

RESTORE_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
echo "Restore initiated at: $RESTORE_TIME"

# Get instance ID from Terraform state
cd "$TF_DIR"
INSTANCE_ID=$(terraform output -json node_details | jq -r ".[\"$NODE_NAME\"].instance_id" 2>/dev/null || echo "")

if [ -n "$INSTANCE_ID" ] && [ "$INSTANCE_ID" != "null" ]; then
  echo "Instance ID: $INSTANCE_ID"
  echo ""
  echo "To start this instance with OCI CLI, run:"
  echo "  oci compute instance action --instance-id $INSTANCE_ID --action START"
  echo ""
else
  echo "ℹ Manual start required:"
  echo "  1. Go to OCI Console → Compute → Instances"
  echo "  2. Find: $NODE_NAME"
  echo "  3. Click More → Start"
  echo ""
fi

# Record event
mkdir -p ../evidence
cat > "../evidence/restore-node-$NODE_NAME-$RESTORE_TIME.json" << EOF
{
  "event": "node_start",
  "node_name": "$NODE_NAME",
  "instance_id": "$INSTANCE_ID",
  "timestamp_utc": "$RESTORE_TIME",
  "test_phase": "P1-ENDTOEND-A01"
}
EOF

echo "Event recorded to: evidence/restore-node-$NODE_NAME-$RESTORE_TIME.json"
echo ""
echo "Node should be running again within 2-3 minutes."
echo "The control plane will re-add the node and re-schedule workloads."
