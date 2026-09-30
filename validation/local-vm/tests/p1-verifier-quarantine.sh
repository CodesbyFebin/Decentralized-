#!/usr/bin/env bash
set -euo pipefail
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
mkdir -p "$fixture/artifacts" "$fixture/observations"
printf 'partition_confirmed_active=true\ndetection_latency_ns=1\n' > "$fixture/artifacts/gate-17-18-partition-injection.log"
printf 'signature_verified=true\n' > "$fixture/artifacts/gate-30-signature-generation.log"
printf 'hash_mismatch_detected=true\n' > "$fixture/artifacts/gate-32-tamper-test.log"
printf 'file_exists=true ledger_parse_exit=0 ledger_type=object\n' > "$fixture/observations/gate-10-summary.txt"
if bash "$repo_root/validation/local-vm/scripts/p1-close-verifier.sh" "$fixture" > "$fixture/output" 2>&1; then
  echo 'Synthetic observations were accepted' >&2
  exit 1
fi
[[ $(find "$fixture/verdicts" -name 'gate-*-verdict.txt' | wc -l) -eq 32 ]]
! grep -R -q '^OUTCOME=PASS$' "$fixture/verdicts"
grep -q '^UNKNOWN=32$' "$fixture/verdicts/verification-summary.txt"
echo 'Legacy P1 verifier fails closed on synthetic observations'
