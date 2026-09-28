#!/bin/bash
# P1-LOCAL-VM-A01: Launch Baseline Workload
# Starts multiple workloads for Phase 3 baseline measurement
# Usage: ./launch-baseline-workload.sh [duration-seconds]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"

die() { echo "ERROR: $*" >&2; exit 1; }

DURATION="${1:-120}"

[ -f "$STATE_DIR/cluster.json" ] || die "Cluster not configured"
[ -f "$STATE_DIR/resourceledger.json" ] || die "ResourceLedger not initialized"

echo "=== P1-LOCAL-VM-A01: Launch Baseline Workload ==="
echo "Duration: ${DURATION}s"
echo ""

# Define baseline workloads
# Each workload gets a small allocation to demonstrate placement diversity
workloads=(
  "workload-api-01:0.3:256:2"
  "workload-api-02:0.3:256:2"
  "workload-cache-01:0.2:128:1"
)

WORKLOAD_PIDS=()

# Start each workload
for workload_def in "${workloads[@]}"; do
  IFS=':' read -r workload_id cpu mem disk <<< "$workload_def"

  echo "Launching $workload_id (CPU=$cpu, Memory=${mem}M, Disk=${disk}G)..."
  bash "$SCRIPT_DIR/launch-workload.sh" "$workload_id" "$cpu" "$mem" "$disk" "$DURATION" &
  WORKLOAD_PIDS+=($!)

  sleep 2  # Stagger workload starts
done

echo ""
echo "Started ${#WORKLOAD_PIDS[@]} workloads"
echo "Waiting for completion..."
echo ""

# Wait for all workloads to complete and track failures
failed_count=0
for i in "${!WORKLOAD_PIDS[@]}"; do
  pid=${WORKLOAD_PIDS[$i]}
  if ! wait $pid; then
    failed_count=$((failed_count + 1))
    echo "ERROR: Workload ${workloads[$i]} (PID $pid) failed"
  fi
done

if [ $failed_count -gt 0 ]; then
  echo ""
  echo "=== Baseline Workload Failed ===" >&2
  echo "$failed_count of ${#WORKLOAD_PIDS[@]} workloads failed" >&2
  exit 1
fi

echo ""
echo "=== Baseline Workload Complete ==="
echo ""
echo "Workload state files:"
ls -lh "$STATE_DIR"/workload-*.json 2>/dev/null || echo "  (none)"
echo ""
echo "Workload logs:"
ls -lh "$STATE_DIR"/workload-logs/ 2>/dev/null || echo "  (none)"
echo ""
echo "Next steps:"
echo "  1. Review workload placement in: query-resourceledger.sh"
echo "  2. Check traffic logs in: validation/local-vm/state/workload-logs/"
echo "  3. Deallocate workloads when ready"
echo "  4. Begin Phase 3 failure injection scenarios"
