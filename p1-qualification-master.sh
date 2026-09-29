#!/bin/bash
# Project entry point: observe the Kubernetes deployment without asserting P1 qualification.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ "${1:-}" = "--local-vm" ]; then
  shift
  exec "$ROOT/validation/local-vm/scripts/legacy-qemu-baseline.sh" "$@"
fi
if [ "$#" -ne 0 ]; then
  echo "Usage: $0 [--local-vm]" >&2
  exit 2
fi
exec "$ROOT/deploy/kubernetes/qualify-readiness.sh"
