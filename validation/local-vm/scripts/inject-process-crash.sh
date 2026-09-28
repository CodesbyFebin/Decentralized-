#!/usr/bin/env bash
# Retired injector: the previous implementation printed simulated process
# kills and hardcoded recovery times as decisive PASS outcomes.
set -euo pipefail
echo 'BLOCKED: no process crash was injected.' >&2
echo 'Target an observed PID, perform a real injection, and measure recovery from' >&2
echo 'the runtime and traffic evidence before evaluating any decisive gate.' >&2
exit 2
