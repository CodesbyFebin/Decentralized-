#!/bin/bash
# P1-LOCAL-VM-A01: ResourceLedger Reconciliation
# Detect orphaned allocations (workloads no longer running) and recover capacity
# Usage: ./reconcile-resourceledger.sh [--auto-release]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
RESOURCELEDGER_JSON="$STATE_DIR/resourceledger.json"
CLUSTER_JSON="$STATE_DIR/cluster.json"

die() { echo "ERROR: $*" >&2; exit 1; }

[ -f "$RESOURCELEDGER_JSON" ] || die "ResourceLedger not initialized"
[ -f "$CLUSTER_JSON" ] || die "Cluster not configured"

AUTO_RELEASE="${1:-}"
[ "$AUTO_RELEASE" = "--auto-release" ] || AUTO_RELEASE=""

echo "=== P1-LOCAL-VM-A01: ResourceLedger Reconciliation ==="
echo ""

# Track reconciliation results
orphaned_ids=()
active_count=0

# Process each node and collect orphaned allocation IDs using temp files
tmpfile=$(mktemp)
nodes=$(jq -r '.node_capacity[].node' "$RESOURCELEDGER_JSON")

for node in $nodes; do
  node_ssh_port=$(jq -r ".node_details[] | select(.name == \"$node\") | .ssh_port" "$CLUSTER_JSON")
  echo "Checking node: $node (port $node_ssh_port)"

  # Use jq -c to output compact JSON objects, one per line
  jq -c ".node_capacity[] | select(.node == \"$node\") | .allocations[] | select(.state != \"RELEASED\" and .state != \"STALE\")" "$RESOURCELEDGER_JSON" 2>/dev/null | while IFS= read -r alloc_json; do
    [ -z "$alloc_json" ] && continue

    alloc_id=$(echo "$alloc_json" | jq -r '.allocation_id')
    workload_id=$(echo "$alloc_json" | jq -r '.workload_id')

    # Check if workload is actually running on the node
    workload_running=$(ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
      -i "${SSH_KEY:-$HOME/.ssh/p1-local-vm}" -p "$node_ssh_port" ubuntu@localhost \
      "ps aux | grep -E 'nc -l 127.0.0.1 8080' | grep -v grep | wc -l" 2>/dev/null || echo "0")

    if [ "$workload_running" -gt 0 ]; then
      echo "  ✓ $alloc_id ($workload_id): workload running"
      echo "ACTIVE" >> "$tmpfile"
    else
      echo "  ✗ $alloc_id ($workload_id): NO RUNNING WORKLOAD (orphaned)"
      echo "$alloc_id" >> "$tmpfile"
    fi
  done
done

# Read results back (macOS/Bash 3 compatible; no mapfile)
active_count="$(grep -c '^ACTIVE
if [ ${#orphaned_ids[@]} -gt 0 ] && [ -n "$AUTO_RELEASE" ]; then
  echo ""
  echo "Releasing ${#orphaned_ids[@]} orphaned allocations..."

  # Mark all orphaned allocations as STALE and recalculate totals
  LEDGER=$(cat "$RESOURCELEDGER_JSON")
  for alloc_id in "${orphaned_ids[@]}"; do
    LEDGER=$(echo "$LEDGER" | jq ".node_capacity[].allocations[] |= if .allocation_id == \"$alloc_id\" then .state = \"STALE\" else . end")
  done

  # Recalculate totals based on ACTIVE allocations only
  LEDGER=$(echo "$LEDGER" | jq '
    ([.node_capacity[].allocations[] | select(.state == "ACTIVE") | .cpu] | add // 0) as $total_cpu |
    ([.node_capacity[].allocations[] | select(.state == "ACTIVE") | .memory_mb] | add // 0) as $total_mem |
    ([.node_capacity[].allocations[] | select(.state == "ACTIVE") | .disk_gb] | add // 0) as $total_disk |
    .node_capacity |= map(
      (([.allocations[] | select(.state == "ACTIVE") | .cpu] | add) // 0) as $cpu_sum |
      (([.allocations[] | select(.state == "ACTIVE") | .memory_mb] | add) // 0) as $mem_sum |
      (([.allocations[] | select(.state == "ACTIVE") | .disk_gb] | add) // 0) as $disk_sum |
      .cpu_allocated = $cpu_sum |
      .memory_allocated_mb = $mem_sum |
      .disk_allocated_gb = $disk_sum
    ) |
    .summary |= (
      .cpu_allocated = $total_cpu |
      .memory_allocated_mb = $total_mem |
      .disk_allocated_gb = $total_disk
    )
  ')

  # Write atomically
  tmp_file=$(mktemp)
  echo "$LEDGER" > "$tmp_file"
  jq empty "$tmp_file" || die "Invalid JSON generated"
  mv "$tmp_file" "$RESOURCELEDGER_JSON"

  echo "Marked ${#orphaned_ids[@]} allocations as STALE"
  for aid in "${orphaned_ids[@]}"; do
    echo "  - $aid"
  done
fi

echo ""
echo "=== Reconciliation Summary ==="
echo "Active allocations:   $active_count"
echo "Orphaned allocations: ${#orphaned_ids[@]}"
if [ -n "$AUTO_RELEASE" ]; then
  echo "Released allocations: ${#orphaned_ids[@]}"
  echo ""
  echo "Updated capacity:"
  bash "$SCRIPT_DIR/query-resourceledger.sh" 2>/dev/null | head -15
else
  echo ""
  echo "To automatically release orphaned allocations, run:"
  echo "  bash $0 --auto-release"
fi
 "$tmpfile" 2>/dev/null || true)"
[ -n "$active_count" ] || active_count=0
orphaned_ids=()
while IFS= read -r aid; do
  [ -n "$aid" ] && orphaned_ids+=("$aid")
done < <(grep -v '^ACTIVE
if [ ${#orphaned_ids[@]} -gt 0 ] && [ -n "$AUTO_RELEASE" ]; then
  echo ""
  echo "Releasing ${#orphaned_ids[@]} orphaned allocations..."

  # Mark all orphaned allocations as STALE and recalculate totals
  LEDGER=$(cat "$RESOURCELEDGER_JSON")
  for alloc_id in "${orphaned_ids[@]}"; do
    LEDGER=$(echo "$LEDGER" | jq ".node_capacity[].allocations[] |= if .allocation_id == \"$alloc_id\" then .state = \"STALE\" else . end")
  done

  # Recalculate totals based on ACTIVE allocations only
  LEDGER=$(echo "$LEDGER" | jq '
    ([.node_capacity[].allocations[] | select(.state == "ACTIVE") | .cpu] | add // 0) as $total_cpu |
    ([.node_capacity[].allocations[] | select(.state == "ACTIVE") | .memory_mb] | add // 0) as $total_mem |
    ([.node_capacity[].allocations[] | select(.state == "ACTIVE") | .disk_gb] | add // 0) as $total_disk |
    .node_capacity |= map(
      (([.allocations[] | select(.state == "ACTIVE") | .cpu] | add) // 0) as $cpu_sum |
      (([.allocations[] | select(.state == "ACTIVE") | .memory_mb] | add) // 0) as $mem_sum |
      (([.allocations[] | select(.state == "ACTIVE") | .disk_gb] | add) // 0) as $disk_sum |
      .cpu_allocated = $cpu_sum |
      .memory_allocated_mb = $mem_sum |
      .disk_allocated_gb = $disk_sum
    ) |
    .summary |= (
      .cpu_allocated = $total_cpu |
      .memory_allocated_mb = $total_mem |
      .disk_allocated_gb = $total_disk
    )
  ')

  # Write atomically
  tmp_file=$(mktemp)
  echo "$LEDGER" > "$tmp_file"
  jq empty "$tmp_file" || die "Invalid JSON generated"
  mv "$tmp_file" "$RESOURCELEDGER_JSON"

  echo "Marked ${#orphaned_ids[@]} allocations as STALE"
  for aid in "${orphaned_ids[@]}"; do
    echo "  - $aid"
  done
fi

echo ""
echo "=== Reconciliation Summary ==="
echo "Active allocations:   $active_count"
echo "Orphaned allocations: ${#orphaned_ids[@]}"
if [ -n "$AUTO_RELEASE" ]; then
  echo "Released allocations: ${#orphaned_ids[@]}"
  echo ""
  echo "Updated capacity:"
  bash "$SCRIPT_DIR/query-resourceledger.sh" 2>/dev/null | head -15
else
  echo ""
  echo "To automatically release orphaned allocations, run:"
  echo "  bash $0 --auto-release"
fi
 "$tmpfile" 2>/dev/null || true)
rm -f "$tmpfile"

# If orphaned allocations found, optionally mark as STALE and recover capacity
if [ ${#orphaned_ids[@]} -gt 0 ] && [ -n "$AUTO_RELEASE" ]; then
  echo ""
  echo "Releasing ${#orphaned_ids[@]} orphaned allocations..."

  # Mark all orphaned allocations as STALE and recalculate totals
  LEDGER=$(cat "$RESOURCELEDGER_JSON")
  for alloc_id in "${orphaned_ids[@]}"; do
    LEDGER=$(echo "$LEDGER" | jq ".node_capacity[].allocations[] |= if .allocation_id == \"$alloc_id\" then .state = \"STALE\" else . end")
  done

  # Recalculate totals based on ACTIVE allocations only
  LEDGER=$(echo "$LEDGER" | jq '
    ([.node_capacity[].allocations[] | select(.state == "ACTIVE") | .cpu] | add // 0) as $total_cpu |
    ([.node_capacity[].allocations[] | select(.state == "ACTIVE") | .memory_mb] | add // 0) as $total_mem |
    ([.node_capacity[].allocations[] | select(.state == "ACTIVE") | .disk_gb] | add // 0) as $total_disk |
    .node_capacity |= map(
      (([.allocations[] | select(.state == "ACTIVE") | .cpu] | add) // 0) as $cpu_sum |
      (([.allocations[] | select(.state == "ACTIVE") | .memory_mb] | add) // 0) as $mem_sum |
      (([.allocations[] | select(.state == "ACTIVE") | .disk_gb] | add) // 0) as $disk_sum |
      .cpu_allocated = $cpu_sum |
      .memory_allocated_mb = $mem_sum |
      .disk_allocated_gb = $disk_sum
    ) |
    .summary |= (
      .cpu_allocated = $total_cpu |
      .memory_allocated_mb = $total_mem |
      .disk_allocated_gb = $total_disk
    )
  ')

  # Write atomically
  tmp_file=$(mktemp)
  echo "$LEDGER" > "$tmp_file"
  jq empty "$tmp_file" || die "Invalid JSON generated"
  mv "$tmp_file" "$RESOURCELEDGER_JSON"

  echo "Marked ${#orphaned_ids[@]} allocations as STALE"
  for aid in "${orphaned_ids[@]}"; do
    echo "  - $aid"
  done
fi

echo ""
echo "=== Reconciliation Summary ==="
echo "Active allocations:   $active_count"
echo "Orphaned allocations: ${#orphaned_ids[@]}"
if [ -n "$AUTO_RELEASE" ]; then
  echo "Released allocations: ${#orphaned_ids[@]}"
  echo ""
  echo "Updated capacity:"
  bash "$SCRIPT_DIR/query-resourceledger.sh" 2>/dev/null | head -15
else
  echo ""
  echo "To automatically release orphaned allocations, run:"
  echo "  bash $0 --auto-release"
fi
