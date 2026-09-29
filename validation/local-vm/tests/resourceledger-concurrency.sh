#!/usr/bin/env bash
set -euo pipefail
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
export DH_RESOURCELEDGER_JSON="$fixture/resourceledger.json"
cat > "$DH_RESOURCELEDGER_JSON" <<'JSON'
{
  "node_capacity": [{
    "node": "dh-node-1", "cpu_cores": 1, "cpu_owner_reserve": 0.2,
    "cpu_reserved": 0.3, "cpu_allocated": 0,
    "memory_mb": 1024, "memory_owner_reserve_mb": 128,
    "memory_reserved_mb": 128, "memory_allocated_mb": 0,
    "disk_gb": 8, "disk_owner_reserve_gb": 1,
    "disk_reserved_gb": 1, "disk_allocated_gb": 0,
    "reservations": [{"reservation_id":"res-existing","workload_id":"reserved-existing",
      "cpu":0.3,"memory_mb":128,"disk_gb":1,"state":"ACTIVE"}],
    "allocations": []
  }],
  "summary": {"cpu_allocated": 0, "memory_allocated_mb": 0, "disk_allocated_gb": 0}
}
JSON
allocate="$repo_root/validation/local-vm/scripts/allocate-resource.sh"
deallocate="$repo_root/validation/local-vm/scripts/deallocate-resource.sh"

# Ten independent processes compete for the same 0.5 CPU of available
# capacity. Exactly two 0.2 CPU allocations may commit.
pids=()
for i in $(seq 1 10); do
  bash "$allocate" dh-node-1 0.2 128 1 "workload-$i" > "$fixture/$i.log" 2>&1 &
  pids+=("$!")
done
success=0
for pid in "${pids[@]}"; do
  if wait "$pid"; then success=$((success + 1)); fi
done
[[ "$success" -eq 2 ]] || { echo "Expected 2 commits, got $success" >&2; exit 1; }
jq -e '.node_capacity[0] | .cpu_allocated == 0.4 and .memory_allocated_mb == 256 and .disk_allocated_gb == 2 and ([.allocations[] | select(.state == "ACTIVE")] | length == 2)' "$DH_RESOURCELEDGER_JSON" >/dev/null
jq -e '.summary.cpu_allocated == .node_capacity[0].cpu_allocated' "$DH_RESOURCELEDGER_JSON" >/dev/null

workload=$(jq -r '.node_capacity[0].allocations[0].workload_id' "$DH_RESOURCELEDGER_JSON")
if bash "$allocate" dh-node-1 0.1 64 1 "$workload" > "$fixture/duplicate.log" 2>&1; then
  echo 'Duplicate active workload accepted' >&2; exit 1
fi
id=$(jq -r '.node_capacity[0].allocations[0].allocation_id' "$DH_RESOURCELEDGER_JSON")
bash "$deallocate" "$id" >/dev/null
if bash "$deallocate" "$id" > "$fixture/repeat-release.log" 2>&1; then
  echo 'Repeated release accepted' >&2; exit 1
fi
bash "$allocate" dh-node-1 0.2 128 1 workload-after-restart > "$fixture/restart.log"
jq -e '.node_capacity[0] | .cpu_allocated == 0.4 and ([.allocations[] | select(.state == "ACTIVE")] | length == 2)' "$DH_RESOURCELEDGER_JSON" >/dev/null
echo 'ResourceLedger capacity, contention, duplicate, release, and persistence checks passed'
