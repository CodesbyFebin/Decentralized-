#!/usr/bin/env bash
# The former verifier certified grep matches and sourced executor-controlled text
# as shell code. Quarantine that path until independently captured, source-bound
# runtime observations can be evaluated against the gate contracts.
set -euo pipefail

execution_dir=${1:-}
if [[ -z "$execution_dir" || ! -d "$execution_dir" ]]; then
  echo 'Usage: p1-close-verifier.sh <execution_directory>' >&2
  exit 2
fi

verdict_dir="$execution_dir/verdicts"
mkdir -p "$verdict_dir"
for gate in $(seq 1 32); do
  printf 'GATE=%s\nTYPE=DECISIVE\nOUTCOME=UNKNOWN\nREASON=Legacy observations lack independent runtime and cryptographic verification\n' "$gate" > "$verdict_dir/gate-${gate}-verdict.txt"
done
printf 'PASS=0\nFAIL=0\nBLOCKED=0\nUNKNOWN=32\nQUALIFIED=NO\nVERIFIED=NO\nSEALED=NO\n' > "$verdict_dir/verification-summary.txt"
cat "$verdict_dir/verification-summary.txt"
exit 2
