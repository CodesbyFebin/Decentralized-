#!/bin/bash
# P1-LOCAL-VM-A01: Deallocate Resource
# Release a resource allocation from a node
# Usage: ./deallocate-resource.sh <allocation-id>

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
RESOURCELEDGER_JSON="$STATE_DIR/resourceledger.json"

die() { echo "ERROR: $*" >&2; exit 1; }
lock_ledger() { flock 9 || die "Failed to acquire lock"; }
unlock_ledger() { true; }

[ -f "$RESOURCELEDGER_JSON" ] || die "ResourceLedger not initialized."

ALLOCATION_ID="${1:-}"
[ -n "$ALLOCATION_ID" ] || die "Usage: $0 <allocation-id>"

exec 9< "$RESOURCELEDGER_JSON"
lock_ledger

# Find allocation and extract details
alloc_data=$(jq ".node_capacity | .[] | .allocations[] | select(.allocation_id == \"$ALLOCATION_ID\")" "$RESOURCELEDGER_JSON" 2>/dev/null || true)
[ -n "$alloc_data" ] || { unlock_ledger; die "Allocation '$ALLOCATION_ID' not found"; }

node_with_alloc=$(jq ".node_capacity | .[] | select(.allocations[] | select(.allocation_id == \"$ALLOCATION_ID\")) | .node" "$RESOURCELEDGER_JSON" 2>/dev/null | head -1)
cpu_to_free=$(echo "$alloc_data" | jq '.cpu')
mem_to_free=$(echo "$alloc_data" | jq '.memory_mb')
disk_to_free=$(echo "$alloc_data" | jq '.disk_gb')

echo "Deallocating $ALLOCATION_ID from $node_with_alloc"
echo "Releasing: $cpu_to_free CPU, ${mem_to_free}M RAM, ${disk_to_free}G disk"

# Update ledger
NEW_LEDGER=$(jq \
  --arg node "$node_with_alloc" \
  --arg id "$ALLOCATION_ID" \
  '.node_capacity |= map(
    if .node == $node then
      .cpu_allocated -= (.allocations[] | select(.allocation_id == $id) | .cpu) |
      .memory_allocated_mb -= (.allocations[] | select(.allocation_id == $id) | .memory_mb) |
      .disk_allocated_gb -= (.allocations[] | select(.allocation_id == $id) | .disk_gb) |
      .allocations |= map(
        if .allocation_id == $id then
          .state = "RELEASED"
        else . end
      )
    else . end
  ) |
  .summary |= (
    .cpu_allocated -= '$cpu_to_free' |
    .memory_allocated_mb -= '$mem_to_free' |
    .disk_allocated_gb -= '$disk_to_free'
  )' "$RESOURCELEDGER_JSON")

# Write atomically
tmp_file=$(mktemp)
echo "$NEW_LEDGER" > "$tmp_file"
jq empty "$tmp_file" || die "Invalid JSON generated"
mv "$tmp_file" "$RESOURCELEDGER_JSON"

unlock_ledger

echo "STATUS: DEALLOCATED"
echo ""
echo "Remaining capacity on $node_with_alloc:"
jq ".node_capacity[] | select(.node == \"$node_with_alloc\") | \"CPU: \(.cpu_allocated)/\(.cpu_cores), Memory: \(.memory_allocated_mb)/\(.memory_mb)M, Disk: \(.disk_allocated_gb)/\(.disk_gb)G\"" -r "$RESOURCELEDGER_JSON"
