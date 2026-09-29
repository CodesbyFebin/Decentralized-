#!/usr/bin/env bash
# Queue one uniquely identified placement request and return its exact result.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="${DH_STATE_DIR:-$REPO_ROOT/validation/local-vm/state}"
WORKLOAD_ID="${1:-}"
CPU="${2:-}"
MEMORY="${3:-}"
DISK="${4:-}"
TIMEOUT="${5:-30}"
OUTPUT="${6:-}"
die() { echo "ERROR: $*" >&2; exit 1; }
[[ "$WORKLOAD_ID" =~ ^[A-Za-z0-9_-]{1,64}$ ]] || die "Invalid workload ID"
for value in "$CPU" "$MEMORY" "$DISK"; do
  [[ "$value" =~ ^[0-9]+(\.[0-9]+)?$ ]] || die "Invalid resource request"
  awk -v n="$value" 'BEGIN { exit !(n > 0) }' || die "Resources must be positive"
done
[[ "$TIMEOUT" =~ ^[1-9][0-9]*$ ]] || die "Invalid timeout"
(( TIMEOUT <= 300 )) || die "Timeout exceeds 300 seconds"
mkdir -p "$STATE_DIR/placement-requests" "$STATE_DIR/placement-results"
ready="$STATE_DIR/scheduler-ready.json"
[[ -f "$ready" ]] || die "Scheduler has no readiness record"
scheduler_pid=$(jq -r '.pid // empty' "$ready")
[[ "$scheduler_pid" =~ ^[0-9]+$ ]] && kill -0 "$scheduler_pid" 2>/dev/null || die "Scheduler readiness record is stale"
expected_sha=$(git -C "$REPO_ROOT" rev-parse HEAD)
[[ $(jq -r '.source_sha // empty' "$ready") == "$expected_sha" ]] || die "Scheduler source SHA mismatch"

request_id=$(python3 -c 'import uuid; print(uuid.uuid4().hex)')
request="$STATE_DIR/placement-requests/$request_id.json"
result="$STATE_DIR/placement-results/$request_id.json"
tmp=$(mktemp "$STATE_DIR/placement-requests/.pending.XXXXXX")
trap 'rm -f "$tmp"' EXIT
jq -n --arg id "$request_id" --arg workload "$WORKLOAD_ID" \
  --argjson cpu "$CPU" --argjson mem "$MEMORY" --argjson disk "$DISK" \
  '{request_id:$id,workload_id:$workload,cpu:$cpu,memory_mb:$mem,disk_gb:$disk}' > "$tmp"
mv "$tmp" "$request"
trap - EXIT

deadline=$((SECONDS + TIMEOUT))
while (( SECONDS < deadline )); do
  if [[ -f "$result" ]]; then
    jq -e --arg id "$request_id" --arg workload "$WORKLOAD_ID" \
      '.request_id == $id and .workload_id == $workload and (.result == "PASS" or .result == "FAIL")' "$result" >/dev/null || die "Mismatched scheduler result"
    [[ $(jq -r '.source_sha' "$result") == "$expected_sha" ]] || die "Placement result source SHA mismatch"
    if [[ -n "$OUTPUT" ]]; then
      mkdir -p "$(dirname "$OUTPUT")"
      cp "$result" "$OUTPUT"
    fi
    jq . "$result"
    [[ $(jq -r '.result' "$result") == PASS ]] || exit 1
    exit 0
  fi
  kill -0 "$scheduler_pid" 2>/dev/null || die "Scheduler exited before responding"
  sleep 0.2
done
if [[ -f "$request" ]]; then
  rm -f "$request"
  die "Placement request $request_id timed out; unclaimed request cancelled"
fi
die "Placement request $request_id timed out while processing; outcome UNKNOWN, inspect $result and the ledger before retry"
