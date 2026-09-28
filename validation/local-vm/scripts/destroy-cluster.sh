#!/bin/bash
# P1-LOCAL-VM-A01: Destroy Cluster
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
CLUSTER_JSON="$STATE_DIR/cluster.json"
EVIDENCE_DIR="$REPO_ROOT/validation/local-vm/evidence"
FORCE=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --yes|-y) FORCE=true; shift ;;
    *) echo "ERROR: Unknown option: $1" >&2; exit 1 ;;
  esac
done

if [ ! -f "$CLUSTER_JSON" ]; then
  echo "No cluster configured."
  exit 0
fi

HYPERVISOR="$(jq -r '.hypervisor' "$CLUSTER_JSON")"
NODES="$(jq -r '.nodes' "$CLUSTER_JSON")"
[ "$HYPERVISOR" = "qemu" ] || { echo "ERROR: Refusing to destroy non-QEMU backend: $HYPERVISOR" >&2; exit 1; }

echo "=== P1-LOCAL-VM-A01: Destroy Cluster ==="
echo "Nodes: $NODES"
echo "State Directory: $STATE_DIR"
echo "Evidence preserved: $EVIDENCE_DIR"

if [ "$FORCE" != true ]; then
  printf "Type 'yes' to confirm destruction: "
  read -r CONFIRM
  [ "$CONFIRM" = "yes" ] || { echo "Cancelled."; exit 2; }
fi

for ((i=1;i<=NODES;i++)); do
  pidfile="$(jq -r ".node_details[$((i-1))].qemu_pid_file" "$CLUSTER_JSON")"
  name="$(jq -r ".node_details[$((i-1))].name" "$CLUSTER_JSON")"
  if [ -s "$pidfile" ]; then
    pid="$(cat "$pidfile" 2>/dev/null || true)"
    if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
      echo "Stopping $name (PID $pid)"
      kill "$pid" 2>/dev/null || true
      for _ in {1..10}; do kill -0 "$pid" 2>/dev/null || break; sleep 0.2; done
      kill -0 "$pid" 2>/dev/null && kill -9 "$pid" 2>/dev/null || true
    fi
  fi
done

rm -rf "$STATE_DIR"
echo "STATUS: DESTROYED"
