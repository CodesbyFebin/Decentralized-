#!/usr/bin/env bash
set -euo pipefail
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
fixture=$(mktemp -d)
export DH_STATE_DIR="$fixture"
export DH_RESOURCELEDGER_JSON="$fixture/resourceledger.json"
cleanup() {
  if [[ -n "${scheduler_pid:-}" ]]; then kill "$scheduler_pid" 2>/dev/null || true; wait "$scheduler_pid" 2>/dev/null || true; fi
  rm -rf "$fixture"
}
trap cleanup EXIT
jq -n '{node_capacity:[range(1;4) | {
  node:("dh-node-" + tostring),cpu_cores:1,cpu_owner_reserve:0,cpu_reserved:0,cpu_allocated:0,
  memory_mb:1024,memory_owner_reserve_mb:0,memory_reserved_mb:0,memory_allocated_mb:0,
  disk_gb:8,disk_owner_reserve_gb:0,disk_reserved_gb:0,disk_allocated_gb:0,allocations:[]
}],summary:{cpu_allocated:0,memory_allocated_mb:0,disk_allocated_gb:0}}' > "$DH_RESOURCELEDGER_JSON"
bash "$repo_root/validation/local-vm/scripts/run-scheduler.sh" first-fit "$fixture/scheduler.log" &
scheduler_pid=$!
for i in $(seq 1 50); do [[ -f "$fixture/scheduler-ready.json" ]] && break; sleep 0.1; done
[[ -f "$fixture/scheduler-ready.json" ]] || { echo "Scheduler not ready" >&2; exit 1; }

pids=()
for i in 1 2 3; do
  bash "$repo_root/validation/local-vm/scripts/request-placement.sh" "workload-$i" 0.6 256 2 15 "$fixture/result-$i.json" > "$fixture/client-$i.log" 2>&1 &
  pids+=("$!")
done
for pid in "${pids[@]}"; do wait "$pid"; done
for i in 1 2 3; do
  jq -e --arg workload "workload-$i" '.result == "PASS" and .workload_id == $workload and .allocation_id != ""' "$fixture/result-$i.json" >/dev/null
done
[[ $(jq -r '[.node_capacity[].allocations[] | select(.state == "ACTIVE")] | length' "$DH_RESOURCELEDGER_JSON") -eq 3 ]]
jq -e '([.node_capacity[].cpu_allocated] | add) as $n | $n > 1.79999 and $n < 1.80001' "$DH_RESOURCELEDGER_JSON" >/dev/null
if bash "$repo_root/validation/local-vm/scripts/request-placement.sh" workload-1 0.1 64 1 5 "$fixture/duplicate.json" > "$fixture/duplicate.log" 2>&1; then
  echo 'Duplicate active workload was placed' >&2; exit 1
fi
[[ $(jq -r '[.node_capacity[].allocations[] | select(.state == "ACTIVE")] | length' "$DH_RESOURCELEDGER_JSON") -eq 3 ]]
echo 'Three simultaneous placements are correlated and capacity remains consistent'
