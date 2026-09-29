#!/usr/bin/env bash
# Retired: the previous validator inferred decisive PASS from log substrings,
# mixed observations from the latest /tmp run with mutable cluster state, and
# did not verify a source-bound signed campaign.
set -euo pipefail
echo 'BLOCKED: validate-p1-close-gates.sh cannot qualify P1.' >&2
echo 'Freeze one campaign source SHA, collect raw runtime observations for each' >&2
echo 'required gate, and verify the signed record with dh evidence verify.' >&2
echo 'No decisive gate was evaluated by this command.' >&2
exit 2
