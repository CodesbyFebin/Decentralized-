#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
ledger="${DH_RESOURCELEDGER_JSON:-$REPO_ROOT/validation/local-vm/state/resourceledger.json}"
[[ $# == 5 ]] || { echo "Usage: $0 node cpu memory_mb disk_gb workload_id" >&2; exit 2; }
exec python3 "$SCRIPT_DIR/resourceledger.py" --ledger "$ledger" reserve "$1" "$2" "$3" "$4" "$5"
