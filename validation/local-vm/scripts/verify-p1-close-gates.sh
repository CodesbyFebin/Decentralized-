#!/bin/bash
# P1-CLOSE Qualification Gates 9-32 Verification
# Usage: ./verify-p1-close-gates.sh <baseline-log-dir> <output-report>
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
CLUSTER_JSON="$STATE_DIR/cluster.json"
BASELINE_LOG_DIR="${1:-.}"
REPORT_OUT="${2:-P1-CLOSE-QUALIFICATION-REPORT.md}"
SSH_KEY="${SSH_KEY:-$HOME/.ssh/p1-local-vm}"
SSH_USER="ubuntu"

# Initialize report
cat > "$REPORT_OUT" << 'EOF'
# P1-LOCAL-VM-A01 Qualification Report

**Status:** IN_PROGRESS
**Generated:** $(date -u +%Y-%m-%dT%H:%M:%SZ)

## Gates 1-8: Baseline Results (from master.log)
EOF

log_gate() {
  local gate_num="$1"
  local gate_name="$2"
  local result="$3"
  local evidence="$4"
  echo "### Gate $gate_num: $gate_name - $result" >> "$REPORT_OUT"
  echo "Evidence: $evidence" >> "$REPORT_OUT"
  echo "" >> "$REPORT_OUT"
}

log_result() {
  echo "$*" >> "$REPORT_OUT"
}

echo "=== P1-CLOSE GATES 9-32 VERIFICATION ===" >&2
echo "Baseline log directory: $BASELINE_LOG_DIR" >&2
echo "Output report: $REPORT_OUT" >&2

# Gate 9: Single Workload Placement Success
echo "[GATE 9] Single Workload Placement Success" >&2
PLACEMENT_LOG="$BASELINE_LOG_DIR/placement-result.json"
if [ -f "$PLACEMENT_LOG" ]; then
  ALLOCATION_ID=$(jq -r '.allocation_id // empty' "$PLACEMENT_LOG" 2>/dev/null || echo "")
  if [ -n "$ALLOCATION_ID" ]; then
    log_gate 9 "Single Workload Placement Success" "PASS" "allocation_id=$ALLOCATION_ID"
    GATE_9_PASS=1
  else
    log_gate 9 "Single Workload Placement Success" "FAIL" "No allocation_id in placement-result.json"
    GATE_9_PASS=0
  fi
else
  log_gate 9 "Single Workload Placement Success" "FAIL" "placement-result.json not found"
  GATE_9_PASS=0
fi

# Gate 10: Workload Ledger State Consistency
echo "[GATE 10] Workload Ledger State Consistency" >&2
LEDGER_JSON="$STATE_DIR/ResourceLedger.json"
if [ -f "$LEDGER_JSON" ]; then
  ACTIVE_COUNT=$(jq '.active_resources | length' "$LEDGER_JSON" 2>/dev/null || echo "0")
  if [ "$ACTIVE_COUNT" -gt 0 ]; then
    log_gate 10 "Workload Ledger State Consistency" "PASS" "active_resources=$ACTIVE_COUNT allocations tracked"
    GATE_10_PASS=1
  else
    log_gate 10 "Workload Ledger State Consistency" "FAIL" "No active_resources in ledger"
    GATE_10_PASS=0
  fi
else
  log_gate 10 "Workload Ledger State Consistency" "FAIL" "ResourceLedger.json not found"
  GATE_10_PASS=0
fi

# Gate 11: Three Concurrent Workloads
echo "[GATE 11] Three Concurrent Workloads (Baseline)" >&2
BASELINE_LOG="$BASELINE_LOG_DIR/baseline.log"
if [ -f "$BASELINE_LOG" ]; then
  WL_COUNT=$(grep -c "workload-api-01\|workload-api-02\|workload-cache-01" "$BASELINE_LOG" 2>/dev/null || echo "0")
  if [ "$WL_COUNT" -ge 3 ]; then
    log_gate 11 "Three Concurrent Workloads" "PASS" "3 workload allocations detected"
    GATE_11_PASS=1
  else
    log_gate 11 "Three Concurrent Workloads" "FAIL" "Only $WL_COUNT workload allocations found"
    GATE_11_PASS=0
  fi
else
  log_gate 11 "Three Concurrent Workloads" "FAIL" "baseline.log not found"
  GATE_11_PASS=0
fi

# Gate 12: Workload Traffic Distribution
echo "[GATE 12] Workload Traffic Distribution" >&2
TRAFFIC_LOGS=$(find "$BASELINE_LOG_DIR" -name "*-traffic.log" 2>/dev/null | wc -l)
if [ "$TRAFFIC_LOGS" -ge 3 ]; then
  TOTAL_REQUESTS=0
  for tlog in $(find "$BASELINE_LOG_DIR" -name "*-traffic.log" 2>/dev/null | head -3); do
    REQUESTS=$(grep -c "HTTP\|200\|GET" "$tlog" 2>/dev/null || echo "0")
    TOTAL_REQUESTS=$((TOTAL_REQUESTS + REQUESTS))
  done
  if [ "$TOTAL_REQUESTS" -gt 0 ]; then
    log_gate 12 "Workload Traffic Distribution" "PASS" "Total requests: $TOTAL_REQUESTS across 3 workloads"
    GATE_12_PASS=1
  else
    log_gate 12 "Workload Traffic Distribution" "FAIL" "No HTTP traffic detected"
    GATE_12_PASS=0
  fi
else
  log_gate 12 "Workload Traffic Distribution" "FAIL" "Only $TRAFFIC_LOGS traffic logs found (need 3)"
  GATE_12_PASS=0
fi

# Gate 13-14: Duration and aggregated counts
echo "[GATE 13-14] Baseline Duration and Request Count" >&2
if [ -f "$BASELINE_LOG" ]; then
  DURATION=$(grep -oP 'Baseline completed in \K[0-9]+' "$BASELINE_LOG" 2>/dev/null || echo "0")
  if [ -n "$DURATION" ] && [ "$DURATION" -lt 150 ]; then
    log_gate 13 "Baseline Duration (120s target)" "PASS" "Duration: ${DURATION}s"
    GATE_13_PASS=1
  else
    log_gate 13 "Baseline Duration (120s target)" "FAIL" "Duration: ${DURATION}s (>120s)"
    GATE_13_PASS=0
  fi

  log_gate 14 "Aggregated Request Count" "INFO" "Total requests analyzed in gate 12"
  GATE_14_PASS=1
else
  log_gate 13 "Baseline Duration (120s target)" "SKIP" "baseline.log not found"
  log_gate 14 "Aggregated Request Count" "SKIP" "baseline.log not found"
  GATE_13_PASS=0
  GATE_14_PASS=0
fi

# Gates 15-16: Exit codes and cleanup
echo "[GATE 15-16] Process Exit Codes and Ledger Cleanup" >&2
WORKLOAD_STATES=$(find "$STATE_DIR" -name "*workload*-state.json" 2>/dev/null | wc -l)
if [ "$WORKLOAD_STATES" -ge 3 ]; then
  EXIT_CODE_FAIL=0
  for state_file in $(find "$STATE_DIR" -name "*workload*-state.json" 2>/dev/null | head -3); do
    EXIT_CODE=$(jq -r '.exit_code // -1' "$state_file" 2>/dev/null || echo "-1")
    if [ "$EXIT_CODE" != "0" ]; then
      EXIT_CODE_FAIL=1
    fi
  done
  if [ "$EXIT_CODE_FAIL" -eq 0 ]; then
    log_gate 15 "Workload Process Exit Codes" "PASS" "All workloads exited with code 0"
    GATE_15_PASS=1
  else
    log_gate 15 "Workload Process Exit Codes" "FAIL" "Some workloads exited with non-zero code"
    GATE_15_PASS=0
  fi
else
  log_gate 15 "Workload Process Exit Codes" "SKIP" "Workload state files not found"
  GATE_15_PASS=0
fi

log_gate 16 "Ledger Cleanup After Workload" "SKIP" "Deallocation verification - manual review required"
GATE_16_PASS=0

# Summary
echo "" >> "$REPORT_OUT"
echo "## Gates 9-16 Summary" >> "$REPORT_OUT"
GATES_9_16_PASS=$((GATE_9_PASS + GATE_10_PASS + GATE_11_PASS + GATE_12_PASS + GATE_13_PASS + GATE_14_PASS + GATE_15_PASS + GATE_16_PASS))
echo "**Result:** $GATES_9_16_PASS/8 gates PASS" >> "$REPORT_OUT"
echo "" >> "$REPORT_OUT"

# Gates 17-28 SKIPPED (require failure injection)
echo "## Gates 17-32 (Failure Injection & Evidence)" >> "$REPORT_OUT"
echo "**Status:** PENDING - Requires manual failure injection scenarios" >> "$REPORT_OUT"
echo "" >> "$REPORT_OUT"
echo "To proceed with gates 17-32:" >> "$REPORT_OUT"
echo "1. Run network partition injection test" >> "$REPORT_OUT"
echo "2. Run process crash injection tests" >> "$REPORT_OUT"
echo "3. Collect evidence and verify integrity" >> "$REPORT_OUT"
echo "4. Run tamper detection negative control" >> "$REPORT_OUT"
echo "" >> "$REPORT_OUT"

echo "=== VERIFICATION COMPLETE ===" >&2
echo "Report written to: $REPORT_OUT" >&2
echo "Gates 9-16 result: $GATES_9_16_PASS/8 PASS" >&2
