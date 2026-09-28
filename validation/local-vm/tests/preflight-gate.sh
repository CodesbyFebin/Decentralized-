#!/usr/bin/env bash
# Regression: a failed preflight must not leave usable topology state.
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/validation/local-vm/scripts" "$tmp/validation/local-vm/state"
cp "$repo/validation/local-vm/scripts/preflight.sh" "$tmp/validation/local-vm/scripts/"
cp "$repo/validation/local-vm/scripts/create-vm-cluster.sh" "$tmp/validation/local-vm/scripts/"
printf 'PREFLIGHT_STATUS=PASS\n' > "$tmp/validation/local-vm/state/topology.sh"

if bash "$tmp/validation/local-vm/scripts/preflight.sh" invalid-backend >"$tmp/preflight.log" 2>&1; then
  echo 'FAIL: unsupported backend passed preflight' >&2; exit 1
fi
if [ -e "$tmp/validation/local-vm/state/topology.sh" ]; then
  echo 'FAIL: failed preflight left topology state' >&2; exit 1
fi
if ! grep -q 'STATUS: BLOCKED' "$tmp/preflight.log"; then
  echo 'FAIL: failed preflight did not report BLOCKED' >&2; exit 1
fi
if bash "$tmp/validation/local-vm/scripts/create-vm-cluster.sh" qemu >"$tmp/create.log" 2>&1; then
  echo 'FAIL: create accepted missing preflight' >&2; exit 1
fi
printf 'NODES=3\n' > "$tmp/validation/local-vm/state/topology.sh"
if bash "$tmp/validation/local-vm/scripts/create-vm-cluster.sh" qemu >"$tmp/create.log" 2>&1; then
  echo 'FAIL: create accepted an unsigned legacy preflight state' >&2; exit 1
fi
grep -q 'no successful preflight marker' "$tmp/create.log"
echo 'PASS: failed and unmarked preflight cannot authorize cluster creation'
