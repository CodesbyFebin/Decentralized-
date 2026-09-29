#!/bin/bash
# P1-LOCAL-VM-A01: ResourceLedger Reconciliation
# Detect orphaned allocations and recover capacity.
# Usage: ./reconcile-resourceledger.sh [--auto-release]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
RESOURCELEDGER_JSON="$STATE_DIR/resourceledger.json"
CLUSTER_JSON="$STATE_DIR/cluster.json"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/p1-local-vm}"
SSH_USER="ubuntu"

die(){ echo "ERROR: $*" >&2; exit 1; }

[ -f "$RESOURCELEDGER_JSON" ] || die "ResourceLedger not initialized"
[ -f "$CLUSTER_JSON" ] || die "Cluster not configured"
command -v jq >/dev/null 2>&1 || die "jq not found"
[ -f "$SSH_KEY" ] || die "SSH key not found: $SSH_KEY"

AUTO_RELEASE=""
[ "${1:-}" = "--auto-release" ] && AUTO_RELEASE=1

# Reconciliation can mark allocations stale and replace the ledger. Keep the
# same stable sidecar lock throughout observation and commit so a concurrent
# allocator cannot write between the observation and capacity recalculation.
if [ -n "$AUTO_RELEASE" ]; then
  exec 9>"${RESOURCELEDGER_JSON}.lock"
  flock -x 9 || die "Failed to lock ResourceLedger"
fi

tmpfile="$(mktemp)"
cleanup(){ rm -f "$tmpfile" "${tmp_ledger:-}" "${next:-}"; }
trap cleanup EXIT

echo "=== P1-LOCAL-VM-A01: ResourceLedger Reconciliation ==="
echo ""

nodes="$(jq -r '.node_capacity[].node' "$RESOURCELEDGER_JSON")"

for node in $nodes; do
  node_ssh_port="$(jq -r --arg node "$node" '.node_details[] | select(.name == $node) | .ssh_port' "$CLUSTER_JSON")"
  [ -n "$node_ssh_port" ] && [ "$node_ssh_port" != "null" ] || die "No SSH port for $node"
  echo "Checking node: $node (port $node_ssh_port)"

  jq -c --arg node "$node" '
    .node_capacity[]
    | select(.node == $node)
    | .allocations[]
    | select(.state != "RELEASED" and .state != "STALE")
  ' "$RESOURCELEDGER_JSON" |
  while IFS= read -r alloc_json; do
    [ -n "$alloc_json" ] || continue
    alloc_id="$(printf '%s' "$alloc_json" | jq -r '.allocation_id')"
    workload_id="$(printf '%s' "$alloc_json" | jq -r '.workload_id')"

    workload_running="$(
      ssh -o BatchMode=yes -o ConnectTimeout=5 -o StrictHostKeyChecking=no         -o UserKnownHostsFile=/dev/null -i "$SSH_KEY" -p "$node_ssh_port" "$SSH_USER@localhost"         "ps aux | grep -F 'workload-$workload_id' | grep -v grep | wc -l" 2>/dev/null || true
    )"

    case "$workload_running" in
      ''|*[!0-9]*)
        echo "  ? $alloc_id ($workload_id): UNKNOWN workload observation"
        printf 'UNKNOWN\t%s\n' "$alloc_id" >> "$tmpfile"
        ;;
      0)
        echo "  ✗ $alloc_id ($workload_id): NO RUNNING WORKLOAD (orphaned)"
        printf 'ORPHAN\t%s\n' "$alloc_id" >> "$tmpfile"
        ;;
      *)
        echo "  ✓ $alloc_id ($workload_id): workload running"
        printf 'ACTIVE\t%s\n' "$alloc_id" >> "$tmpfile"
        ;;
    esac
  done
done

active_count="$(awk -F '\t' '$1=="ACTIVE"{c++} END{print c+0}' "$tmpfile")"
orphan_count="$(awk -F '\t' '$1=="ORPHAN"{c++} END{print c+0}' "$tmpfile")"
unknown_count="$(awk -F '\t' '$1=="UNKNOWN"{c++} END{print c+0}' "$tmpfile")"

orphaned_ids=()
while IFS=$'\t' read -r kind aid; do
  [ "$kind" = "ORPHAN" ] && [ -n "$aid" ] && orphaned_ids+=("$aid")
done < "$tmpfile"

if [ "$unknown_count" -gt 0 ] && [ -n "$AUTO_RELEASE" ]; then
  die "Refusing auto-release: $unknown_count allocation observations are UNKNOWN"
fi

if [ "${#orphaned_ids[@]}" -gt 0 ] && [ -n "$AUTO_RELEASE" ]; then
  echo ""
  echo "Releasing ${#orphaned_ids[@]} orphaned allocations..."

  tmp_ledger="$(mktemp "${RESOURCELEDGER_JSON}.tmp.XXXXXX")"
  cp "$RESOURCELEDGER_JSON" "$tmp_ledger"

  for alloc_id in "${orphaned_ids[@]}"; do
    next="$(mktemp "${RESOURCELEDGER_JSON}.tmp.XXXXXX")"
    jq --arg id "$alloc_id" '
      .node_capacity |= map(
        .allocations |= map(
          if .allocation_id == $id then .state = "STALE" else . end
        )
      )
    ' "$tmp_ledger" > "$next"
    mv "$next" "$tmp_ledger"
  done

  next="$(mktemp "${RESOURCELEDGER_JSON}.tmp.XXXXXX")"
  jq '
    .node_capacity |= map(
      ([.allocations[] | select(.state == "ACTIVE") | .cpu] | add // 0) as $cpu |
      ([.allocations[] | select(.state == "ACTIVE") | .memory_mb] | add // 0) as $mem |
      ([.allocations[] | select(.state == "ACTIVE") | .disk_gb] | add // 0) as $disk |
      .cpu_allocated = $cpu |
      .memory_allocated_mb = $mem |
      .disk_allocated_gb = $disk
    )
    |
    ([.node_capacity[].cpu_allocated] | add // 0) as $tcpu |
    ([.node_capacity[].memory_allocated_mb] | add // 0) as $tmem |
    ([.node_capacity[].disk_allocated_gb] | add // 0) as $tdisk |
    .summary.cpu_allocated = $tcpu |
    .summary.memory_allocated_mb = $tmem |
    .summary.disk_allocated_gb = $tdisk
  ' "$tmp_ledger" > "$next"
  mv "$next" "$tmp_ledger"

  jq empty "$tmp_ledger"
  mv "$tmp_ledger" "$RESOURCELEDGER_JSON"

  echo "Marked ${#orphaned_ids[@]} allocations as STALE"
  for aid in "${orphaned_ids[@]}"; do
    echo "  - $aid"
  done
fi

echo ""
echo "=== Reconciliation Summary ==="
echo "Active allocations:   $active_count"
echo "Orphaned allocations: $orphan_count"
echo "Unknown observations: $unknown_count"

if [ -n "$AUTO_RELEASE" ]; then
  echo "Released allocations: ${#orphaned_ids[@]}"
  echo ""
  echo "Updated capacity:"
  bash "$SCRIPT_DIR/query-resourceledger.sh" | head -15
else
  echo ""
  echo "To automatically release orphaned allocations, run:"
  echo "  bash $0 --auto-release"
fi

[ "$unknown_count" -eq 0 ] || exit 2
exit 0
