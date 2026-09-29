#!/usr/bin/env bash
set -euo pipefail
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
export DH_STATE_DIR="$fixture"
sha=$(git -C "$repo_root" rev-parse HEAD)
jq -n --arg sha "$sha" '{nodes:2,source_sha:$sha,cpu_per_node:3,
  memory_per_node_mb:2048,disk_per_node_gb:12,
  node_details:[{name:"dh-node-1"},{name:"dh-node-2"}]}' > "$fixture/cluster.json"
bash "$repo_root/validation/local-vm/scripts/init-resourceledger.sh" > "$fixture/init.log"
jq -e '.summary.cpu_total == 6 and .summary.memory_total_mb == 4096 and
  .summary.disk_total_gb == 24 and
  ([.node_capacity[] | select(.cpu_cores == 3 and .memory_mb == 2048 and .disk_gb == 12 and .reservations == [])] | length == 2)' \
  "$fixture/resourceledger.json" >/dev/null
rm "$fixture/resourceledger.json"
jq '.source_sha = "stale"' "$fixture/cluster.json" > "$fixture/changed.json"
mv "$fixture/changed.json" "$fixture/cluster.json"
if bash "$repo_root/validation/local-vm/scripts/init-resourceledger.sh" > "$fixture/stale.log" 2>&1; then
  echo 'Ledger accepted a stale cluster source SHA' >&2; exit 1
fi
[[ ! -f "$fixture/resourceledger.json" ]]
echo 'Ledger initializes declared capacity and rejects stale source SHA'
