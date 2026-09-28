#!/bin/bash
# P1-LOCAL-VM-A01: Request Placement
# Client API for workloads to request scheduler placement
# Usage: ./request-placement.sh <workload-id> <cpu> <memory-mb> <disk-gb> [timeout-seconds]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"

die() { echo "ERROR: $*" >&2; exit 1; }

WORKLOAD_ID="${1:-}"
CPU="${2:-}"
MEMORY="${3:-}"
DISK="${4:-}"
TIMEOUT="${5:-30}"

[ -n "$WORKLOAD_ID" ] || die "Usage: $0 <workload-id> <cpu> <memory-mb> <disk-gb> [timeout-seconds]"
[ -n "$CPU" ] || die "Usage: $0 <workload-id> <cpu> <memory-mb> <disk-gb> [timeout-seconds]"
[ -n "$MEMORY" ] || die "Usage: $0 <workload-id> <cpu> <memory-mb> <disk-gb> [timeout-seconds]"
[ -n "$DISK" ] || die "Usage: $0 <workload-id> <cpu> <memory-mb> <disk-gb> [timeout-seconds]"

PLACEMENT_REQUEST_FILE="$STATE_DIR/placement-request.json"
PLACEMENT_RESULT_FILE="$STATE_DIR/placement-result.json"

# Submit request
cat > "$PLACEMENT_REQUEST_FILE" << EOF
{
  "workload_id": "$WORKLOAD_ID",
  "cpu": $CPU,
  "memory_mb": $MEMORY,
  "disk_gb": $DISK,
  "requested_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

echo "Placement request submitted for $WORKLOAD_ID"
echo "Waiting up to ${TIMEOUT}s for scheduler response..."

# Wait for result
deadline=$((SECONDS + TIMEOUT))
while (( SECONDS < deadline )); do
  if [ -f "$PLACEMENT_RESULT_FILE" ]; then
    # Check if this is our result
    result_workload=$(jq -r '.workload_id' "$PLACEMENT_RESULT_FILE")
    if [ "$result_workload" = "$WORKLOAD_ID" ]; then
      result=$(jq -r '.result' "$PLACEMENT_RESULT_FILE")
      node=$(jq -r '.target_node' "$PLACEMENT_RESULT_FILE")
      alloc_id=$(jq -r '.allocation_id' "$PLACEMENT_RESULT_FILE")

      echo ""
      echo "Placement result: $result"

      if [ "$result" = "PASS" ]; then
        echo "  Node: $node"
        echo "  Allocation ID: $alloc_id"
        echo ""
        echo "Workload $WORKLOAD_ID scheduled on $node"
        exit 0
      else
        echo "  Reason: No node with sufficient capacity"
        exit 1
      fi
    fi
  fi

  sleep 0.5
done

die "Placement request timed out after ${TIMEOUT}s"
