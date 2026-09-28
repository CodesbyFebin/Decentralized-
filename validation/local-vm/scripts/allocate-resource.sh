#!/bin/bash
# P1-LOCAL-VM-A01: Allocate Resource
# Request and record a resource allocation on a node
# Usage: ./allocate-resource.sh <node-name> <cpu> <memory-mb> <disk-gb> <workload-id>

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
RESOURCELEDGER_JSON="$STATE_DIR/resourceledger.json"

die() { echo "ERROR: $*" >&2; exit 1; }

[ -f "$RESOURCELEDGER_JSON" ] || die "ResourceLedger not initialized. Run init-resourceledger.sh first."

NODE_NAME="${1:-}"
CPU_REQ="${2:-}"
MEMORY_REQ="${3:-}"
DISK_REQ="${4:-}"
WORKLOAD_ID="${5:-}"

[ -n "$NODE_NAME" ] || die "Usage: $0 <node-name> <cpu> <memory-mb> <disk-gb> <workload-id>"
[ -n "$CPU_REQ" ] || die "Usage: $0 <node-name> <cpu> <memory-mb> <disk-gb> <workload-id>"
[ -n "$MEMORY_REQ" ] || die "Usage: $0 <node-name> <cpu> <memory-mb> <disk-gb> <workload-id>"
[ -n "$DISK_REQ" ] || die "Usage: $0 <node-name> <cpu> <memory-mb> <disk-gb> <workload-id>"
[ -n "$WORKLOAD_ID" ] || die "Usage: $0 <node-name> <cpu> <memory-mb> <disk-gb> <workload-id>"

# Validate numbers
for val in "$CPU_REQ" "$MEMORY_REQ" "$DISK_REQ"; do
  [[ "$val" =~ ^[0-9]+(\.[0-9]+)?$ ]] || die "Invalid numeric value: $val"
done

ALLOCATION_ID="alloc-$(date -u +%s)-$(od -An -N4 -tu4 /dev/urandom | tr -d ' ')"

# Read current state and check availability
node_data="$(jq ".node_capacity[] | select(.node == \"$NODE_NAME\")" "$RESOURCELEDGER_JSON")"
[ -n "$node_data" ] || die "Node '$NODE_NAME' not found in ResourceLedger"

# Calculate allocated from ACTIVE allocations only
active_allocated=$(echo "$node_data" | jq '[.allocations[] | select(.state == "ACTIVE") | {cpu: .cpu, mem: .memory_mb, disk: .disk_gb}] | {cpu: map(.cpu) | add // 0, mem: map(.mem) | add // 0, disk: map(.disk) | add // 0}')
cpu_active=$(echo "$active_allocated" | jq '.cpu')
mem_active=$(echo "$active_allocated" | jq '.mem')
disk_active=$(echo "$active_allocated" | jq '.disk')

cpu_available="$(echo "$node_data" | jq -n --argjson total "$($node_data | jq '.cpu_cores')" --argjson active "$cpu_active" '$total - $active')"
memory_available="$(echo "$node_data" | jq -n --argjson total "$($node_data | jq '.memory_mb')" --argjson active "$mem_active" '$total - $active')"
disk_available="$(echo "$node_data" | jq -n --argjson total "$($node_data | jq '.disk_gb')" --argjson active "$disk_active" '$total - $active')"

# Check allocation feasibility using awk for floating-point comparison
if awk -v req="$CPU_REQ" -v avail="$cpu_available" 'BEGIN { exit !(req > avail) }'; then
  die "Insufficient CPU: requested $CPU_REQ, available $cpu_available on $NODE_NAME"
fi
if awk -v req="$MEMORY_REQ" -v avail="$memory_available" 'BEGIN { exit !(req > avail) }'; then
  die "Insufficient memory: requested $MEMORY_REQ MB, available $memory_available MB on $NODE_NAME"
fi
if awk -v req="$DISK_REQ" -v avail="$disk_available" 'BEGIN { exit !(req > avail) }'; then
  die "Insufficient disk: requested $DISK_REQ GB, available $disk_available GB on $NODE_NAME"
fi

# Update ledger with new allocation
NEW_LEDGER=$(jq \
  --arg node "$NODE_NAME" \
  --arg id "$ALLOCATION_ID" \
  --arg workload "$WORKLOAD_ID" \
  --argjson cpu "$CPU_REQ" \
  --argjson mem "$MEMORY_REQ" \
  --argjson disk "$DISK_REQ" \
  '.node_capacity |= map(
    if .node == $node then
      .cpu_allocated += $cpu |
      .memory_allocated_mb += $mem |
      .disk_allocated_gb += $disk |
      .allocations += [{
        "allocation_id": $id,
        "workload_id": $workload,
        "cpu": $cpu,
        "memory_mb": $mem,
        "disk_gb": $disk,
        "state": "ACTIVE",
        "created_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
      }]
    else . end
  ) |
  .summary |= (
    .cpu_allocated += $cpu |
    .memory_allocated_mb += $mem |
    .disk_allocated_gb += $disk
  )' "$RESOURCELEDGER_JSON")

# Write atomically
tmp_file=$(mktemp)
echo "$NEW_LEDGER" > "$tmp_file"
jq empty "$tmp_file" || die "Invalid JSON generated"
mv "$tmp_file" "$RESOURCELEDGER_JSON"

echo "STATUS: ALLOCATED"
echo "Allocation ID: $ALLOCATION_ID"
echo "Node: $NODE_NAME"
echo "Resources: $CPU_REQ CPU, ${MEMORY_REQ}M RAM, ${DISK_REQ}G disk"
echo "Workload: $WORKLOAD_ID"
echo ""
echo "Remaining capacity on $NODE_NAME:"
jq ".node_capacity[] | select(.node == \"$NODE_NAME\") | \"CPU: \(.cpu_allocated)/\(.cpu_cores), Memory: \(.memory_allocated_mb)/\(.memory_mb)M, Disk: \(.disk_allocated_gb)/\(.disk_gb)G\"" -r "$RESOURCELEDGER_JSON"
