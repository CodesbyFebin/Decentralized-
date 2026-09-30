#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
ledger="${DH_RESOURCELEDGER_JSON:-$REPO_ROOT/validation/local-vm/state/resourceledger.json}"
[[ $# == 1 ]] || { echo "Usage: $0 reservation_id" >&2; exit 2; }
exec python3 "$SCRIPT_DIR/resourceledger.py" --ledger "$ledger" release-reservation "$1"
