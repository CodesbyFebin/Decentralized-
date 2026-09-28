#!/bin/bash
# P1-LOCAL-VM-A01: Initialize ResourceLedger
# Persistent state store for resource allocations and claims
# Usage: ./init-resourceledger.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
CLUSTER_JSON="$STATE_DIR/cluster.json"
RESOURCELEDGER_JSON="$STATE_DIR/resourceledger.json"

die() { echo "ERROR: $*" >&2; exit 1; }

[ -f "$CLUSTER_JSON" ] || die "Cluster not configured. Run bootstrap first."
[ ! -f "$RESOURCELEDGER_JSON" ] || die "ResourceLedger already exists at $RESOURCELEDGER_JSON. Destroy cluster to reinitialize."

NODES="$(jq -r '.nodes' "$CLUSTER_JSON")"
CLUSTER_SOURCE_SHA="$(jq -r '.source_sha' "$CLUSTER_JSON")"
CURRENT_SHA="$(git -C "$REPO_ROOT" rev-parse HEAD)"

echo "=== P1-LOCAL-VM-A01: Initialize ResourceLedger ==="
echo "Nodes: $NODES"
echo "Cluster source SHA: $CLUSTER_SOURCE_SHA"
echo "Ledger source SHA: $CURRENT_SHA"
echo ""

{
  echo "{"
  echo '  "schema_version": 1,'
  echo '  "qualification": "P1-LOCAL-VM-A01",'
  echo "  \"created_at\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\","
  echo "  \"cluster_source_sha\": \"$CLUSTER_SOURCE_SHA\","
  echo "  \"ledger_source_sha\": \"$CURRENT_SHA\","
  echo "  \"nodes\": $NODES,"
  echo '  "node_capacity": ['

  for ((i=1;i<=NODES;i++)); do
    idx=$((i-1))
    node_name="$(jq -r ".node_details[$idx].name" "$CLUSTER_JSON")"

    # Read specs from vm-specs.json if available, else use fixed defaults
    # VM specs: 1 CPU, 1024 MiB RAM, 8 GiB disk per node
    cpu_capacity=1
    memory_mb=1024
    disk_gb=8

    comma=","; [ "$i" -eq "$NODES" ] && comma=""
    echo "    {"
    echo "      \"node\": \"$node_name\","
    echo "      \"cpu_cores\": $cpu_capacity,"
    echo "      \"memory_mb\": $memory_mb,"
    echo "      \"disk_gb\": $disk_gb,"
    echo "      \"cpu_allocated\": 0,"
    echo "      \"memory_allocated_mb\": 0,"
    echo "      \"disk_allocated_gb\": 0,"
    echo "      \"allocations\": []"
    echo "    }$comma"
  done

  echo "  ],"
  echo '  "summary": {'
  echo "    \"cpu_total\": $NODES,"
  echo "    \"cpu_allocated\": 0,"
  echo "    \"memory_total_mb\": $((NODES * 1024)),"
  echo "    \"memory_allocated_mb\": 0,"
  echo "    \"disk_total_gb\": $((NODES * 8)),"
  echo "    \"disk_allocated_gb\": 0"
  echo "  }"
  echo "}"
} > "$RESOURCELEDGER_JSON"

jq empty "$RESOURCELEDGER_JSON" || die "Invalid JSON generated"

echo "STATUS: INITIALIZED"
echo "ResourceLedger: $RESOURCELEDGER_JSON"
echo ""
echo "Capacity:"
jq '.node_capacity[] | "\(.node): \(.cpu_cores) CPU, \(.memory_mb)M RAM, \(.disk_gb)G disk"' -r "$RESOURCELEDGER_JSON"
