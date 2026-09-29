#!/usr/bin/env bash
# Single scheduler for per-request placement files. The ledger allocator is
# the transaction authority and rechecks capacity under its stable lock.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="${DH_STATE_DIR:-$REPO_ROOT/validation/local-vm/state}"
RESOURCELEDGER_JSON="${DH_RESOURCELEDGER_JSON:-$STATE_DIR/resourceledger.json}"
export DH_RESOURCELEDGER_JSON
STRATEGY="${1:-first-fit}"
SCHEDULER_LOG="${2:-$STATE_DIR/scheduler.log}"
[[ "$STRATEGY" == first-fit || "$STRATEGY" == best-fit ]] || { echo "Invalid strategy" >&2; exit 2; }
[[ -f "$RESOURCELEDGER_JSON" ]] || { echo "ResourceLedger not initialized" >&2; exit 2; }
mkdir -p "$STATE_DIR/placement-requests" "$STATE_DIR/placement-results" "$(dirname "$SCHEDULER_LOG")"

exec 8>"$STATE_DIR/scheduler.lock"
flock -n 8 || { echo "Another scheduler is running" >&2; exit 2; }
ready="$STATE_DIR/scheduler-ready.json"
cleanup() { rm -f "$ready"; }
trap cleanup EXIT
trap 'exit 0' INT TERM
ready_tmp=$(mktemp "$STATE_DIR/.scheduler-ready.XXXXXX")
jq -n --argjson pid "$$" --arg sha "$(git -C "$REPO_ROOT" rev-parse HEAD)" \
  --arg strategy "$STRATEGY" '{pid:$pid,source_sha:$sha,strategy:$strategy}' > "$ready_tmp"
mv "$ready_tmp" "$ready"
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) Scheduler ready pid=$$ strategy=$STRATEGY" >> "$SCHEDULER_LOG"

while true; do
  for request in "$STATE_DIR"/placement-requests/*.json; do
    [[ -f "$request" ]] || continue
    id=$(basename "$request" .json)
    processing="${request%.json}.processing"
    mv "$request" "$processing" 2>/dev/null || continue
    request="$processing"
    # The filename and JSON identity must agree. The client publishes by
    # atomic rename so the scheduler never reads a partial request.
    if ! jq -e --arg id "$id" '
      .request_id == $id and
      (.workload_id | type == "string" and test("^[A-Za-z0-9_-]{1,64}$")) and
      (.cpu | type == "number" and . > 0) and
      (.memory_mb | type == "number" and . > 0) and
      (.disk_gb | type == "number" and . > 0)
    ' "$request" >/dev/null 2>&1; then
      echo "Rejected malformed request: $request" >> "$SCHEDULER_LOG"
      mv "$request" "$request.invalid"
      continue
    fi
    workload=$(jq -r '.workload_id' "$request")
    cpu=$(jq -r '.cpu' "$request")
    mem=$(jq -r '.memory_mb' "$request")
    disk=$(jq -r '.disk_gb' "$request")

    # Candidate selection is advisory. allocate-resource.sh locks, rechecks,
    # and atomically commits; a stale candidate cannot overbook the ledger.
    candidates=$(jq -r --argjson cpu "$cpu" --argjson mem "$mem" --argjson disk "$disk" \
      --arg strategy "$STRATEGY" '
      [.node_capacity[] | select(
        (.cpu_cores - .cpu_owner_reserve - .cpu_reserved - .cpu_allocated) >= $cpu and
        (.memory_mb - .memory_owner_reserve_mb - .memory_reserved_mb - .memory_allocated_mb) >= $mem and
        (.disk_gb - .disk_owner_reserve_gb - .disk_reserved_gb - .disk_allocated_gb) >= $disk
      ) | {
        node:.node,
        score:((.cpu_cores - .cpu_owner_reserve - .cpu_reserved - .cpu_allocated - $cpu) +
              (.memory_mb - .memory_owner_reserve_mb - .memory_reserved_mb - .memory_allocated_mb - $mem)/1024 +
              (.disk_gb - .disk_owner_reserve_gb - .disk_reserved_gb - .disk_allocated_gb - $disk))
      }] | if $strategy == "best-fit" then sort_by(.score) else . end | .[].node
    ' "$RESOURCELEDGER_JSON")
    result=FAIL
    target=""
    allocation_id=""
    for node in $candidates; do
      if allocation_output=$(bash "$SCRIPT_DIR/allocate-resource.sh" "$node" "$cpu" "$mem" "$disk" "$workload" 2>&1); then
        allocation_id=$(printf '%s\n' "$allocation_output" | sed -n 's/^Allocation ID: //p' | head -1)
        if [[ -n "$allocation_id" ]]; then
          target="$node"
          result=PASS
          break
        fi
      else
        printf '%s\n' "$allocation_output" >> "$SCHEDULER_LOG"
      fi
    done
    result_tmp=$(mktemp "$STATE_DIR/placement-results/.result.XXXXXX")
    jq -n --arg id "$id" --arg workload "$workload" --arg result "$result" \
      --arg node "$target" --arg allocation "$allocation_id" \
      --arg sha "$(git -C "$REPO_ROOT" rev-parse HEAD)" \
      '{request_id:$id,workload_id:$workload,result:$result,target_node:$node,
        allocation_id:$allocation,source_sha:$sha}' > "$result_tmp"
    mv "$result_tmp" "$STATE_DIR/placement-results/$id.json"
    rm -f "$request"
    echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $id $workload $result $target $allocation_id" >> "$SCHEDULER_LOG"
  done
  sleep 0.2
done
