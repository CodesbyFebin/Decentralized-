#!/usr/bin/env bash
# Usage: allocate-resource.sh node cpu memory_mb disk_gb workload_id [reservation_id]
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
ledger="${DH_RESOURCELEDGER_JSON:-$REPO_ROOT/validation/local-vm/state/resourceledger.json}"
[[ $# == 5 || $# == 6 ]] || { echo "Usage: $0 node cpu memory_mb disk_gb workload_id [reservation_id]" >&2; exit 2; }
if [[ $# == 6 ]]; then
  exec python3 "$SCRIPT_DIR/resourceledger.py" --ledger "$ledger" allocate "$1" "$2" "$3" "$4" "$5" --reservation-id "$6"
fi
exec python3 "$SCRIPT_DIR/resourceledger.py" --ledger "$ledger" allocate "$1" "$2" "$3" "$4" "$5"
