#!/bin/bash
# P1-LOCAL-VM-A01: Run Scheduler
# Simple placement controller that reads ResourceLedger and schedules workloads
# Usage: ./run-scheduler.sh <strategy> <log-file>

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
RESOURCELEDGER_JSON="$STATE_DIR/resourceledger.json"
SCHEDULER_LOG="${2:-$STATE_DIR/scheduler.log}"
PLACEMENT_MANIFEST="$STATE_DIR/placements.json"

die() { echo "ERROR: $*" >&2; exit 1; }

STRATEGY="${1:-first-fit}"
[ "$STRATEGY" = "first-fit" ] || [ "$STRATEGY" = "best-fit" ] || die "Unknown strategy: $STRATEGY (use first-fit or best-fit)"

[ -f "$RESOURCELEDGER_JSON" ] || die "ResourceLedger not initialized"

mkdir -p "$(dirname "$SCHEDULER_LOG")"

echo "=== P1-LOCAL-VM-A01: Scheduler ($STRATEGY) ===" | tee "$SCHEDULER_LOG"
echo "Started at $(date -u +%Y-%m-%dT%H:%M:%SZ)" | tee -a "$SCHEDULER_LOG"
echo ""

# Initialize placement manifest if it doesn't exist
if [ ! -f "$PLACEMENT_MANIFEST" ]; then
  cat > "$PLACEMENT_MANIFEST" << 'EOF'
{
  "schema_version": 1,
  "qualification": "P1-LOCAL-VM-A01",
  "created_at": "",
  "strategy": "",
  "placements": []
}
EOF
fi

# Update creation timestamp and strategy
tmp_file=$(mktemp)
jq \
  --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg strat "$STRATEGY" \
  '.created_at = $ts | .strategy = $strat' \
  "$PLACEMENT_MANIFEST" > "$tmp_file"
mv "$tmp_file" "$PLACEMENT_MANIFEST"

echo "Strategy: $STRATEGY" | tee -a "$SCHEDULER_LOG"
echo "Placement manifest: $PLACEMENT_MANIFEST" | tee -a "$SCHEDULER_LOG"
echo "" | tee -a "$SCHEDULER_LOG"

# Function to find best node for placement
find_placement_node() {
  local cpu="$1"
  local mem="$2"
  local disk="$3"
  local strategy="$4"

  if [ "$strategy" = "first-fit" ]; then
    # Return first node with sufficient capacity
    jq -r ".node_capacity[] | select(
      (.cpu_cores - .cpu_allocated) >= $cpu and
      (.memory_mb - .memory_allocated_mb) >= $mem and
      (.disk_gb - .disk_allocated_gb) >= $disk
    ) | .node" "$RESOURCELEDGER_JSON" | head -1
  elif [ "$strategy" = "best-fit" ]; then
    # Return node with smallest remaining capacity after allocation
    jq -r ".node_capacity[] | select(
      (.cpu_cores - .cpu_allocated) >= $cpu and
      (.memory_mb - .memory_allocated_mb) >= $mem and
      (.disk_gb - .disk_allocated_gb) >= $disk
    ) | {
      node: .node,
      remaining_cpu: (.cpu_cores - .cpu_allocated - $cpu),
      remaining_mem: (.memory_mb - .memory_allocated_mb - $mem),
      remaining_disk: (.disk_gb - .disk_allocated_gb - $disk)
    }" "$RESOURCELEDGER_JSON" | jq -s "sort_by(.remaining_cpu + .remaining_mem/1024 + .remaining_disk) | .[0].node" -r
  fi
}

# Main scheduler loop
# Listens for placement requests via a request file
PLACEMENT_REQUEST_FILE="$STATE_DIR/placement-request.json"
PLACEMENT_RESULT_FILE="$STATE_DIR/placement-result.json"

echo "Scheduler ready, listening for placement requests at: $PLACEMENT_REQUEST_FILE" | tee -a "$SCHEDULER_LOG"
echo "" | tee -a "$SCHEDULER_LOG"

while true; do
  if [ -f "$PLACEMENT_REQUEST_FILE" ]; then
    # Read placement request
    workload_id=$(jq -r '.workload_id' "$PLACEMENT_REQUEST_FILE")
    cpu_req=$(jq -r '.cpu' "$PLACEMENT_REQUEST_FILE")
    mem_req=$(jq -r '.memory_mb' "$PLACEMENT_REQUEST_FILE")
    disk_req=$(jq -r '.disk_gb' "$PLACEMENT_REQUEST_FILE")

    echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) Placement request: $workload_id (CPU=$cpu_req, MEM=${mem_req}M, DISK=${disk_req}G)" | tee -a "$SCHEDULER_LOG"

    # Find best node
    target_node=$(find_placement_node "$cpu_req" "$mem_req" "$disk_req" "$STRATEGY")

    if [ -z "$target_node" ]; then
      echo "  ✗ No available node with capacity" | tee -a "$SCHEDULER_LOG"
      result="FAIL"
      allocation_id=""
    else
      echo "  → Attempting placement on $target_node" | tee -a "$SCHEDULER_LOG"

      # Attempt allocation
      if allocation_id=$(bash "$SCRIPT_DIR/allocate-resource.sh" "$target_node" "$cpu_req" "$mem_req" "$disk_req" "$workload_id" 2>&1 | grep "Allocation ID:" | awk '{print $NF}'); then
        echo "  ✓ Placement succeeded, allocation_id=$allocation_id" | tee -a "$SCHEDULER_LOG"
        result="PASS"
      else
        echo "  ✗ Allocation failed" | tee -a "$SCHEDULER_LOG"
        result="FAIL"
        allocation_id=""
      fi
    fi

    # Write result
    cat > "$PLACEMENT_RESULT_FILE" << EOF
{
  "workload_id": "$workload_id",
  "result": "$result",
  "target_node": "${target_node:-}",
  "allocation_id": "$allocation_id",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

    # Add to placement manifest
    tmp_file=$(mktemp)
    jq \
      --arg wl "$workload_id" \
      --arg res "$result" \
      --arg node "${target_node:-}" \
      --arg alloc "$allocation_id" \
      '.placements += [{
        "workload_id": $wl,
        "result": $res,
        "target_node": $node,
        "allocation_id": $alloc,
        "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
      }]' \
      "$PLACEMENT_MANIFEST" > "$tmp_file"
    mv "$tmp_file" "$PLACEMENT_MANIFEST"

    # Clean up request
    rm "$PLACEMENT_REQUEST_FILE"

    echo "" | tee -a "$SCHEDULER_LOG"
  fi

  sleep 1
done
