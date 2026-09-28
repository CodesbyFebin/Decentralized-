#!/usr/bin/env bash
# Retired gate checker: the previous implementation inferred PASS from log
# strings, counted missing observations as zero, and did not bind a campaign.
set -euo pipefail
echo 'BLOCKED: verify-p1-close-gates.sh cannot establish P1 qualification.' >&2
echo 'Use the current campaign raw evidence and the signed dh evidence verify contract.' >&2
echo 'Every unobserved decisive gate remains UNKNOWN; no gate was evaluated.' >&2
exit 2
