#!/usr/bin/env bash
set -euo pipefail
scripts="$(cd "$(dirname "${BASH_SOURCE[0]}")/../scripts" && pwd)"
for name in verify-p1-close-gates inject-network-partition inject-process-crash; do
  output="$(bash "$scripts/$name.sh" 2>&1)" && {
    echo "FAIL: $name returned success without a runtime campaign" >&2
    exit 1
  }
  case "$output" in
    BLOCKED:*) ;;
    *) echo "FAIL: $name did not report BLOCKED" >&2; exit 1 ;;
  esac
  if printf '%s\n' "$output" | grep -Eq '(^|[[:space:]])PASS([[:space:]]|$)'; then
    echo "FAIL: $name printed PASS" >&2; exit 1
  fi
done
output="$(bash "$scripts/../../../validate-p1-close-gates.sh" 2>&1)" && {
  echo 'FAIL: root validator returned success without a runtime campaign' >&2
  exit 1
}
case "$output" in
  BLOCKED:*) ;;
  *) echo 'FAIL: root validator did not report BLOCKED' >&2; exit 1 ;;
esac
echo 'PASS: retired gate scripts fail closed without simulated evidence'
