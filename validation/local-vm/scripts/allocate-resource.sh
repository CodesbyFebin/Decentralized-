#!/bin/bash
# P1-LOCAL-VM-A01: Allocate Resource
# Request and record a resource allocation on a node
# Usage: ./allocate-resource.sh <node-name> <cpu> <memory-mb> <disk-gb> <workload-id>

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
RESOURCELEDGER_JSON="${DH_RESOURCELEDGER_JSON:-$STATE_DIR/resourceledger.json}"

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

# Lock a stable sidecar inode for the entire read/check/write transaction.
# Locking resourceledger.json itself is ineffective after atomic rename.
exec 9>"${RESOURCELEDGER_JSON}.lock"
flock -x 9 || die "Failed to lock ResourceLedger"

# Validate numbers
for val in "$CPU_REQ" "$MEMORY_REQ" "$DISK_REQ"; do
  [[ "$val" =~ ^[0-9]+(\.[0-9]+)?$ ]] || die "Invalid numeric value: $val"
  awk -v req="$val" 'BEGIN { exit !(req > 0) }' || die "Resource request must be positive: $val"
done

ALLOCATION_ID="alloc-$(date -u +%s)-$(od -An -N4 -tu4 /dev/urandom | tr -d ' ')"

# Read current state and check availability
node_data="$(jq --arg node "$NODE_NAME" '.node_capacity[] | select(.node == $node)' "$RESOURCELEDGER_JSON")"
[ -n "$node_data" ] || die "Node '$NODE_NAME' not found in ResourceLedger"
if jq -e --arg workload "$WORKLOAD_ID" '[.node_capacity[].allocations[] | select(.workload_id == $workload and .state == "ACTIVE")] | length > 0' "$RESOURCELEDGER_JSON" >/dev/null; then
  die "Active allocation already exists for workload '$WORKLOAD_ID'"
fi

# Calculate allocated from ACTIVE allocations only
active_allocated=$(echo "$node_data" | jq -c '
  [.allocations[] | select(.state == "ACTIVE") | {cpu: .cpu, mem: .memory_mb, disk: .disk_gb}] |
  {cpu: (map(.cpu) | add // 0), mem: (map(.mem) | add // 0), disk: (map(.disk) | add // 0)}
')
cpu_active=$(echo "$active_allocated" | jq '.cpu')
mem_active=$(echo "$active_allocated" | jq '.mem')
disk_active=$(echo "$active_allocated" | jq '.disk')

cpu_total=$(echo "$node_data" | jq '.cpu_cores')
memory_total=$(echo "$node_data" | jq '.memory_mb')
disk_total=$(echo "$node_data" | jq '.disk_gb')

cpu_owner_reserve=$(echo "$node_data" | jq '.cpu_owner_reserve')
memory_owner_reserve=$(echo "$node_data" | jq '.memory_owner_reserve_mb')
disk_owner_reserve=$(echo "$node_data" | jq '.disk_owner_reserve_gb')

cpu_reserved=$(echo "$node_data" | jq '.cpu_reserved')
memory_reserved=$(echo "$node_data" | jq '.memory_reserved_mb')
disk_reserved=$(echo "$node_data" | jq '.disk_reserved_gb')

# Model A: AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED
cpu_available=$(echo "$cpu_total $cpu_owner_reserve $cpu_reserved $cpu_active" | awk '{print $1 - $2 - $3 - $4}')
memory_available=$(echo "$memory_total $memory_owner_reserve $memory_reserved $mem_active" | awk '{print $1 - $2 - $3 - $4}')
disk_available=$(echo "$disk_total $disk_owner_reserve $disk_reserved $disk_active" | awk '{print $1 - $2 - $3 - $4}')

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

for available in "$cpu_available" "$memory_available" "$disk_available"; do
  awk -v amount="$available" 'BEGIN { exit !(amount < 0) }' && die "Negative ledger availability on $NODE_NAME"
done

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
tmp_file=$(mktemp "${RESOURCELEDGER_JSON}.tmp.XXXXXX")
trap 'rm -f "$tmp_file"' EXIT
echo "$NEW_LEDGER" > "$tmp_file"
jq empty "$tmp_file" || die "Invalid JSON generated"
mv "$tmp_file" "$RESOURCELEDGER_JSON"
trap - EXIT

echo "STATUS: ALLOCATED"
echo "Allocation ID: $ALLOCATION_ID"
echo "Node: $NODE_NAME"
echo "Resources: $CPU_REQ CPU, ${MEMORY_REQ}M RAM, ${DISK_REQ}G disk"
echo "Workload: $WORKLOAD_ID"
echo ""
echo "Remaining capacity on $NODE_NAME:"
jq --arg node "$NODE_NAME" -r '.node_capacity[] | select(.node == $node) | "CPU: \(.cpu_allocated)/\(.cpu_cores), Memory: \(.memory_allocated_mb)/\(.memory_mb)M, Disk: \(.disk_allocated_gb)/\(.disk_gb)G"' "$RESOURCELEDGER_JSON"
