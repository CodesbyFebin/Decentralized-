#!/usr/bin/env bash
# Retired injector: the previous implementation only polled forwarded SSH
# without isolating guest-to-guest traffic, then reported injection as PASS.
set -euo pipefail
echo 'BLOCKED: no network partition was injected.' >&2
echo 'Use a reviewed hypervisor or network isolation mechanism, capture the actual' >&2
echo 'runtime effect, and verify detection, reconciliation, heal, and traffic.' >&2
exit 2
