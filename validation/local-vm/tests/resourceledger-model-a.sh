#!/usr/bin/env bash
set -euo pipefail
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
export DH_RESOURCELEDGER_JSON="$fixture/resourceledger.json"
cat > "$DH_RESOURCELEDGER_JSON" <<'JSON'
{
  "node_capacity": [{
    "node":"dh-node-1","cpu_cores":1,"cpu_owner_reserve":0,"cpu_reserved":0,"cpu_allocated":0,
    "memory_mb":1024,"memory_owner_reserve_mb":0,"memory_reserved_mb":0,"memory_allocated_mb":0,
    "disk_gb":8,"disk_owner_reserve_gb":0,"disk_reserved_gb":0,"disk_allocated_gb":0,
    "reservations":[],"allocations":[]
  }],
  "summary":{"cpu_owner_reserve":0,"cpu_reserved":0,"cpu_allocated":0,
    "memory_owner_reserve_mb":0,"memory_reserved_mb":0,"memory_allocated_mb":0,
    "disk_owner_reserve_gb":0,"disk_reserved_gb":0,"disk_allocated_gb":0}
}
JSON
scripts="$repo_root/validation/local-vm/scripts"
bash "$scripts/set-owner-reserve.sh" dh-node-1 0.1 128 1 >/dev/null
reservation=$(bash "$scripts/reserve-resource.sh" dh-node-1 0.5 256 2 reserved-workload | sed -n 's/^Reservation ID: //p')
[[ -n "$reservation" ]]
jq -e '.node_capacity[0] | .cpu_reserved == 0.5 and .cpu_owner_reserve == 0.1 and .cpu_allocated == 0' "$DH_RESOURCELEDGER_JSON" >/dev/null
if bash "$scripts/set-owner-reserve.sh" dh-node-1 0.6 128 1 > "$fixture/overreserve.log" 2>&1; then
  echo 'Owner reserve exceeded capacity' >&2; exit 1
fi
jq -e '.node_capacity[0].cpu_owner_reserve == 0.1' "$DH_RESOURCELEDGER_JSON" >/dev/null

# Exactly one concurrent claimant can consume a reservation. The reserved
# counter falls as allocated rises; available capacity is unchanged.
pids=()
for i in 1 2; do
  bash "$scripts/allocate-resource.sh" dh-node-1 0.5 256 2 reserved-workload "$reservation" > "$fixture/alloc-$i.log" 2>&1 &
  pids+=("$!")
done
success=0
for pid in "${pids[@]}"; do if wait "$pid"; then success=$((success + 1)); fi; done
[[ "$success" == 1 ]] || { echo "Reservation was consumed $success times" >&2; exit 1; }
jq -e '.node_capacity[0] | .cpu_reserved == 0 and .cpu_allocated == 0.5 and
  .reservations[0].state == "CONSUMED" and
  ([.allocations[] | select(.state == "ACTIVE")] | length == 1)' "$DH_RESOURCELEDGER_JSON" >/dev/null
if bash "$scripts/release-reservation.sh" "$reservation" > "$fixture/consumed-release.log" 2>&1; then
  echo 'Consumed reservation was released twice' >&2; exit 1
fi
allocation=$(jq -r '.node_capacity[0].allocations[0].allocation_id' "$DH_RESOURCELEDGER_JSON")
bash "$scripts/deallocate-resource.sh" "$allocation" >/dev/null
unconsumed=$(bash "$scripts/reserve-resource.sh" dh-node-1 0.2 128 1 unconsumed-workload | sed -n 's/^Reservation ID: //p')
bash "$scripts/release-reservation.sh" "$unconsumed" >/dev/null
jq -e '.node_capacity[0] | .cpu_reserved == 0 and .cpu_allocated == 0 and .cpu_owner_reserve == 0.1' "$DH_RESOURCELEDGER_JSON" >/dev/null

# A corrupt summary must be rejected before any mutation or success report.
jq '.summary.cpu_allocated = 0.5' "$DH_RESOURCELEDGER_JSON" > "$fixture/corrupt.json"
export DH_RESOURCELEDGER_JSON="$fixture/corrupt.json"
before=$(sha256sum "$DH_RESOURCELEDGER_JSON" | cut -d' ' -f1)
if bash "$scripts/allocate-resource.sh" dh-node-1 0.1 64 1 corrupt-attempt > "$fixture/corrupt-output.log" 2>&1; then
  echo 'Corrupt summary was accepted' >&2; exit 1
fi
after=$(sha256sum "$DH_RESOURCELEDGER_JSON" | cut -d' ' -f1)
[[ "$before" == "$after" ]]
! grep -q 'STATUS: ALLOCATED' "$fixture/corrupt-output.log"
echo 'ResourceLedger Model A reservation transfer, owner reserve, and rejection checks passed'
