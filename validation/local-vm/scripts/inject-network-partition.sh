#!/usr/bin/env bash
# Retired injector: the previous implementation only polled forwarded SSH
# without isolating guest-to-guest traffic, then reported injection as PASS.
#
# The new diagnostic is observe-network-fault.py, which schedules actual tc netem
# fault injection via guest systemd timers and captures real traffic loss.
# Use that script instead.
set -euo pipefail
echo 'BLOCKED: legacy network partition injector is retired.' >&2
echo 'Use: python3 validation/local-vm/scripts/observe-network-fault.py' >&2
echo '  --node <name> --workload-id <id> --http-port <port> --inject' >&2
exit 2
