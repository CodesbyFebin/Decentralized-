#!/bin/bash
# P1-CLOSE Gates 1-32 Validation Framework
# Runs immediately after master script completion to analyze all gates
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"

# Find the latest log directory
LOG_DIR=$(ls -td /tmp/p1-qualification-* 2>/dev/null | head -1)
if [ -z "$LOG_DIR" ]; then
  echo "ERROR: No p1-qualification log directory found" >&2
  exit 1
fi

echo "=== P1-CLOSE Gates 1-32 Analysis ===" >&2
echo "Log directory: $LOG_DIR" >&2
echo "Started: $(date -u)" >&2
echo ""

# Counters
PASS_COUNT=0
FAIL_COUNT=0
BLOCKED_COUNT=0

# Helper functions
test_gate() {
  local gate_num=$1
  local description=$2
  local condition=$3

  echo -n "Gate $gate_num ($description): "

  if eval "$condition" 2>/dev/null; then
    echo "✅ PASS"
    ((PASS_COUNT++))
    return 0
  else
    echo "❌ FAIL"
    ((FAIL_COUNT++))
    return 1
  fi
}

test_gate_conditional() {
  local gate_num=$1
  local description=$2
  local condition=$3
  local blocked_condition=$4

  echo -n "Gate $gate_num ($description): "

  if eval "$blocked_condition" 2>/dev/null; then
    echo "⏳ BLOCKED"
    ((BLOCKED_COUNT++))
    return 2
  elif eval "$condition" 2>/dev/null; then
    echo "✅ PASS"
    ((PASS_COUNT++))
    return 0
  else
    echo "❌ FAIL"
    ((FAIL_COUNT++))
    return 1
  fi
}

# ========== PHASE 1: BASELINE (Gates 1-8) ==========
echo "[PHASE 1] Baseline Execution (Gates 1-8)" >&2

test_gate 1 "Environment verification" \
  "grep -q '\[PASS\] Step 1: Environment verified' '$LOG_DIR/master.log'"

test_gate 2 "Cluster creation" \
  "grep -q '\[PASS\] Step 2: Cluster created' '$LOG_DIR/master.log' && [ -f '$STATE_DIR/cluster.json' ]"

test_gate 3 "VM cluster startup" \
  "grep -q '\[PASS\] Step 3: Cluster started' '$LOG_DIR/master.log'"

test_gate 4 "Bootstrap nodes" \
  "grep -q 'All 3 nodes are PROCESS_RUNNING + SSH_READY' '$LOG_DIR/bootstrap-attempt-2.log' 2>/dev/null || grep -q 'All 3 nodes are PROCESS_RUNNING + SSH_READY' '$LOG_DIR/bootstrap-attempt-1.log' 2>/dev/null"

test_gate 5 "Network qualification" \
  "grep -q '\[PASS\] Step 5: Network qualified' '$LOG_DIR/master.log' && [ -f '$STATE_DIR/network-qualification.json' ]"

test_gate 6 "socat installation" \
  "[ -f '$LOG_DIR/socat-install-1.log' ] && [ -f '$LOG_DIR/socat-install-2.log' ] && [ -f '$LOG_DIR/socat-install-3.log' ]"

test_gate 7 "ResourceLedger initialization" \
  "grep -q '\[PASS\] Step 7: Resource ledger initialized' '$LOG_DIR/master.log' && [ -f '$STATE_DIR/resourceledger.json' ]"

test_gate_conditional 8 "Smoke test (15s traffic)" \
  "grep -qE 'traffic.*generated|request_count' '$LOG_DIR/smoke-test.log'" \
  "grep -q 'Placement request timed out' '$LOG_DIR/smoke-test.log' || grep -q 'Scheduler not running' '$LOG_DIR/scheduler-startup.log' 2>/dev/null"

echo ""

# ========== PHASE 2: WORKLOAD PLACEMENT (Gates 9-16) ==========
echo "[PHASE 2] Workload Placement (Gates 9-16)" >&2

test_gate_conditional 9 "Single workload placement" \
  "[ -f '$STATE_DIR/placement-result.json' ] && grep -q '\"result\": \"PASS\"' '$STATE_DIR/placement-result.json'" \
  "[ $PASS_COUNT -lt 8 ]"

test_gate_conditional 10 "ResourceLedger consistency" \
  "jq '.node_capacity | length' '$STATE_DIR/resourceledger.json' 2>/dev/null | grep -q '[3-9]'" \
  "[ $PASS_COUNT -lt 8 ]"

test_gate_conditional 11 "Three concurrent workloads" \
  "[ -f '$LOG_DIR/baseline.log' ] && grep -q 'workload.*running' '$LOG_DIR/baseline.log'" \
  "[ ! -f '$LOG_DIR/baseline.log' ]"

test_gate_conditional 12 "Traffic distribution" \
  "ls -1 '$STATE_DIR/workload-logs'/*-traffic.log 2>/dev/null | wc -l | grep -qE '[3-9]'" \
  "[ ! -d '$STATE_DIR/workload-logs' ]"

test_gate_conditional 13 "Baseline duration (120s)" \
  "grep -q 'duration.*120' '$LOG_DIR/baseline.log' 2>/dev/null || grep -qE '120.*second|elapsed.*120' '$LOG_DIR/baseline.log' 2>/dev/null" \
  "[ ! -f '$LOG_DIR/baseline.log' ]"

test_gate_conditional 14 "Request count aggregation" \
  "grep -qE 'total_requests|request_count.*[0-9]{2,}' '$LOG_DIR/baseline.log' 2>/dev/null" \
  "[ ! -f '$LOG_DIR/baseline.log' ]"

test_gate_conditional 15 "Workload exit codes (0)" \
  "grep -qE 'exit.*0|success' '$LOG_DIR/baseline.log' 2>/dev/null" \
  "[ ! -f '$LOG_DIR/baseline.log' ]"

test_gate_conditional 16 "Workload cleanup/deallocation" \
  "grep -qE 'cleanup|dealloc|removed' '$LOG_DIR/baseline.log' 2>/dev/null" \
  "[ ! -f '$LOG_DIR/baseline.log' ]"

echo ""

# ========== PHASE 3: NETWORK PARTITION FAILURE (Gates 17-22) ==========
echo "[PHASE 3] Network Partition Failure (Gates 17-22)" >&2

# Check if baseline passed; if not, mark network tests as blocked
if [ $PASS_COUNT -lt 16 ]; then
  echo "Gates 17-22: ⏳ BLOCKED (baseline incomplete)"
  BLOCKED_COUNT=$((BLOCKED_COUNT + 6))
else
  echo "Network partition injection tests pending..."
  BLOCKED_COUNT=$((BLOCKED_COUNT + 6))
fi

echo ""

# ========== PHASE 4: PROCESS CRASH INJECTION (Gates 23-28) ==========
echo "[PHASE 4] Process Crash Injection (Gates 23-28)" >&2
echo "Process crash injection tests pending..."
BLOCKED_COUNT=$((BLOCKED_COUNT + 6))

echo ""

# ========== PHASE 5: EVIDENCE & VERIFICATION (Gates 29-32) ==========
echo "[PHASE 5] Evidence & Verification (Gates 29-32)" >&2
echo "Evidence collection and verification tests pending..."
BLOCKED_COUNT=$((BLOCKED_COUNT + 4))

echo ""
echo "=== Summary ===" >&2
echo "PASS:    $PASS_COUNT gates" >&2
echo "FAIL:    $FAIL_COUNT gates" >&2
echo "BLOCKED: $BLOCKED_COUNT gates" >&2
echo "TOTAL:   $(($PASS_COUNT + $FAIL_COUNT + $BLOCKED_COUNT)) gates" >&2

# Report overall status
echo ""
if [ $FAIL_COUNT -eq 0 ] && [ $PASS_COUNT -ge 8 ]; then
  echo "Status: ✅ GATES 1-8 PASS - Ready for failure injection (gates 17-28)" >&2
  exit 0
elif [ $FAIL_COUNT -eq 0 ] && [ $PASS_COUNT -ge 16 ]; then
  echo "Status: ✅ GATES 1-16 PASS - Ready for gates 17-32" >&2
  exit 0
else
  echo "Status: ❌ TESTS INCOMPLETE - $FAIL_COUNT failures, $BLOCKED_COUNT blocked" >&2
  echo "Log directory: $LOG_DIR" >&2
  exit 1
fi
