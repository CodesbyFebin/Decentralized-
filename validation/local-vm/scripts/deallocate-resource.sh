#!/usr/bin/env bash
# Release one ACTIVE allocation under the same stable lock as allocation.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
RESOURCELEDGER_JSON="${DH_RESOURCELEDGER_JSON:-$REPO_ROOT/validation/local-vm/state/resourceledger.json}"
ALLOCATION_ID="${1:-}"
die() { echo "ERROR: $*" >&2; exit 1; }
[[ -n "$ALLOCATION_ID" ]] || die "Usage: $0 <allocation-id>"
[[ -f "$RESOURCELEDGER_JSON" ]] || die "ResourceLedger not initialized"
exec 9>"${RESOURCELEDGER_JSON}.lock"
flock -x 9 || die "Failed to lock ResourceLedger"

matches=$(jq --arg id "$ALLOCATION_ID" '[.node_capacity[].allocations[] | select(.allocation_id == $id)] | length' "$RESOURCELEDGER_JSON")
[[ "$matches" == 1 ]] || die "Expected one allocation for '$ALLOCATION_ID', found $matches"
state=$(jq -r --arg id "$ALLOCATION_ID" '.node_capacity[].allocations[] | select(.allocation_id == $id) | .state' "$RESOURCELEDGER_JSON")
[[ "$state" == ACTIVE ]] || die "Allocation '$ALLOCATION_ID' is $state, not ACTIVE"

tmp_file=$(mktemp "${RESOURCELEDGER_JSON}.tmp.XXXXXX")
trap 'rm -f "$tmp_file"' EXIT
jq --arg id "$ALLOCATION_ID" '
  .node_capacity |= map(
    ([.allocations[] | select(.allocation_id == $id and .state == "ACTIVE")] | first) as $a |
    if $a == null then . else
      .cpu_allocated -= $a.cpu |
      .memory_allocated_mb -= $a.memory_mb |
      .disk_allocated_gb -= $a.disk_gb |
      .allocations |= map(if .allocation_id == $id then .state = "RELEASED" else . end)
    end
  ) |
  .summary.cpu_allocated = ([.node_capacity[].cpu_allocated] | add) |
  .summary.memory_allocated_mb = ([.node_capacity[].memory_allocated_mb] | add) |
  .summary.disk_allocated_gb = ([.node_capacity[].disk_allocated_gb] | add)
' "$RESOURCELEDGER_JSON" > "$tmp_file"
jq -e 'all(.node_capacity[]; .cpu_allocated >= 0 and .memory_allocated_mb >= 0 and .disk_allocated_gb >= 0)' "$tmp_file" >/dev/null || die "Deallocation violates ledger invariants"
mv "$tmp_file" "$RESOURCELEDGER_JSON"
trap - EXIT
echo "STATUS: DEALLOCATED"
echo "Allocation ID: $ALLOCATION_ID"
