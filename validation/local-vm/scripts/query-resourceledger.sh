#!/bin/bash
# P1-LOCAL-VM-A01: Query ResourceLedger
# Display current resource allocation state
# Usage: ./query-resourceledger.sh [node-name]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
RESOURCELEDGER_JSON="$STATE_DIR/resourceledger.json"

die() { echo "ERROR: $*" >&2; exit 1; }

[ -f "$RESOURCELEDGER_JSON" ] || die "ResourceLedger not initialized. Run init-resourceledger.sh first."

QUERY_NODE="${1:-}"

echo "=== P1-LOCAL-VM-A01: ResourceLedger Status ==="
echo ""

if [ -z "$QUERY_NODE" ]; then
  # Show summary
  echo "Cluster Summary:"
  jq '.summary | to_entries[] | "\(.key): \(.value)"' -r "$RESOURCELEDGER_JSON"
  echo ""

  # Show per-node summary
  echo "Per-Node Status:"
  jq '.node_capacity[] | "  \(.node): \(.cpu_allocated)/\(.cpu_cores) CPU, \(.memory_allocated_mb)/\(.memory_mb)M RAM, \(.disk_allocated_gb)/\(.disk_gb)G disk, \(.allocations | length) allocations"' -r "$RESOURCELEDGER_JSON"
else
  # Show specific node details
  node_data="$(jq ".node_capacity[] | select(.node == \"$QUERY_NODE\")" "$RESOURCELEDGER_JSON")"
  if [ -z "$node_data" ]; then
    die "Node '$QUERY_NODE' not found"
  fi

  echo "Node: $QUERY_NODE"
  echo "$node_data" | jq '.'
  echo ""
  echo "Allocations:"
  echo "$node_data" | jq '.allocations[] | "  \(.allocation_id): \(.workload_id) - \(.cpu)C \(.memory_mb)M \(.disk_gb)G (state=\(.state))"' -r
fi

echo ""
