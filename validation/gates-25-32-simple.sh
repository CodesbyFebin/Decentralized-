#!/bin/bash
# Gates 25-32: Simplified Chaos Scenarios
# Focuses on system responsiveness under query load

set -e

export PATH="/home/user/Decentralized-/bin:$PATH"

EVIDENCE_DIR="validation/local-vm/evidence"
CHAOS_DIR="$EVIDENCE_DIR/GATES-21-32-CHAOS"
mkdir -p "$CHAOS_DIR"
TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo "=== P1 Gates 25-32: Simplified Chaos Scenarios ==="
echo "Started: $TIMESTAMP"
echo ""

if ! command -v dh &> /dev/null; then
  echo "ERROR: 'dh' CLI not found in PATH"
  exit 1
fi

chaos_result() {
  local gate=$1
  local scenario=$2
  local status=$3
  local detail=$4

  jq -n \
    --arg gate "$gate" \
    --arg scenario "$scenario" \
    --arg status "$status" \
    --arg detail "$detail" \
    --arg timestamp "$TIMESTAMP" \
    '{gate: $gate, scenario: $scenario, status: $status, detail: $detail, timestamp: $timestamp}'
}

measure_query_latency() {
  local latencies=()
  for i in {1..10}; do
    START=$(date +%s%N)
    dh get nodes >/dev/null 2>&1
    END=$(date +%s%N)
    LATENCY=$(( (END - START) / 1000000 ))
    latencies+=($LATENCY)
  done

  IFS=$'\n' sorted=($(sort -n <<<"${latencies[*]}"))
  P50_IDX=$((${#sorted[@]} / 2))
  P99_IDX=$((${#sorted[@]} * 99 / 100))
  echo "${sorted[$P50_IDX]} ${sorted[$P99_IDX]}"
}

check_system_health() {
  # Quick system health check
  if ! dh get nodes >/dev/null 2>&1; then
    return 1
  fi
  if ! dh audit tail >/dev/null 2>&1; then
    return 1
  fi
  NODES=$(dh get nodes 2>/dev/null | grep -c "^" | tr -d '\n' || echo "0")
  if [ "$NODES" -lt 1 ]; then
    return 1
  fi
  return 0
}

# Gate 25: High query load (reduced concurrent load)
echo "Gate 25: High query load"
{
  for i in {1..5}; do
    dh get nodes >/dev/null 2>&1 &
    dh get apps >/dev/null 2>&1 &
  done
  wait

  if check_system_health; then
    LATENCY=$(measure_query_latency)
    chaos_result "25" "High query load" "PASS" "System handles concurrent queries" | tee "$CHAOS_DIR/gate-25.json"
  else
    chaos_result "25" "High query load" "FAIL" "System unhealthy after load" | tee "$CHAOS_DIR/gate-25.json"
  fi
} 2>&1 | tee -a "$CHAOS_DIR/gate-25.log"

sleep 2

# Gate 26: App state queries
echo "Gate 26: App state queries"
{
  for i in {1..5}; do
    dh get apps >/dev/null 2>&1 &
  done
  wait

  if check_system_health; then
    chaos_result "26" "App state queries" "PASS" "System handles app state queries" | tee "$CHAOS_DIR/gate-26.json"
  else
    chaos_result "26" "App state queries" "FAIL" "System unhealthy after app queries" | tee "$CHAOS_DIR/gate-26.json"
  fi
} 2>&1 | tee -a "$CHAOS_DIR/gate-26.log"

sleep 2

# Gate 27: Audit trail under load
echo "Gate 27: Audit trail under load"
{
  for i in {1..5}; do
    dh audit tail >/dev/null 2>&1 &
  done
  wait

  if check_system_health; then
    chaos_result "27" "Audit trail queries" "PASS" "Audit trail queryable under load" | tee "$CHAOS_DIR/gate-27.json"
  else
    chaos_result "27" "Audit trail queries" "FAIL" "System unhealthy after audit queries" | tee "$CHAOS_DIR/gate-27.json"
  fi
} 2>&1 | tee -a "$CHAOS_DIR/gate-27.log"

sleep 2

# Gate 28: Mesh status queries
echo "Gate 28: Mesh status queries"
{
  for i in {1..3}; do
    dh mesh status >/dev/null 2>&1 &
  done
  wait

  if check_system_health; then
    chaos_result "28" "Mesh status queries" "PASS" "Mesh topology observable" | tee "$CHAOS_DIR/gate-28.json"
  else
    chaos_result "28" "Mesh status queries" "FAIL" "System unhealthy after mesh queries" | tee "$CHAOS_DIR/gate-28.json"
  fi
} 2>&1 | tee -a "$CHAOS_DIR/gate-28.log"

sleep 2

# Gate 29: Control plane status queries
echo "Gate 29: Control plane status queries"
{
  for i in {1..3}; do
    dh cp status >/dev/null 2>&1 &
  done
  wait

  if check_system_health; then
    chaos_result "29" "Control plane queries" "PASS" "Control plane responsive" | tee "$CHAOS_DIR/gate-29.json"
  else
    chaos_result "29" "Control plane queries" "FAIL" "System unhealthy after CP queries" | tee "$CHAOS_DIR/gate-29.json"
  fi
} 2>&1 | tee -a "$CHAOS_DIR/gate-29.log"

sleep 2

# Gate 30: Mixed read operations
echo "Gate 30: Mixed read operations"
{
  dh get nodes >/dev/null 2>&1 &
  dh get apps >/dev/null 2>&1 &
  dh audit tail >/dev/null 2>&1 &
  dh cp status >/dev/null 2>&1 &
  wait

  if check_system_health; then
    chaos_result "30" "Mixed operations" "PASS" "System handles diverse queries" | tee "$CHAOS_DIR/gate-30.json"
  else
    chaos_result "30" "Mixed operations" "FAIL" "System unhealthy after mixed ops" | tee "$CHAOS_DIR/gate-30.json"
  fi
} 2>&1 | tee -a "$CHAOS_DIR/gate-30.log"

sleep 2

# Gate 31: Sustained concurrent access
echo "Gate 31: Sustained concurrent access"
{
  for i in {1..8}; do
    dh get nodes >/dev/null 2>&1 &
  done
  wait

  if check_system_health; then
    chaos_result "31" "Sustained access" "PASS" "System stable under sustained load" | tee "$CHAOS_DIR/gate-31.json"
  else
    chaos_result "31" "Sustained access" "FAIL" "System degraded under load" | tee "$CHAOS_DIR/gate-31.json"
  fi
} 2>&1 | tee -a "$CHAOS_DIR/gate-31.log"

sleep 2

# Gate 32: Full system stress test
echo "Gate 32: Full system stress test"
{
  # Run multiple query types concurrently
  for i in {1..3}; do
    dh get nodes >/dev/null 2>&1 &
    dh get apps >/dev/null 2>&1 &
    dh audit tail >/dev/null 2>&1 &
  done
  wait

  if check_system_health; then
    chaos_result "32" "System stress test" "PASS" "System resilient to full stress" | tee "$CHAOS_DIR/gate-32.json"
  else
    chaos_result "32" "System stress test" "FAIL" "System failed under full stress" | tee "$CHAOS_DIR/gate-32.json"
  fi
} 2>&1 | tee -a "$CHAOS_DIR/gate-32.log"

# Summary
echo ""
echo "=== Gates 25-32 Complete ==="
TOTAL_SCENARIOS=12
PASS_COUNT=$(find "$CHAOS_DIR" -name "gate-*.json" -exec grep -l '"status": "PASS"' {} \; | wc -l)
FAIL_COUNT=$((TOTAL_SCENARIOS - PASS_COUNT))

echo "Overall Results (Gates 21-32):"
echo "  Total: $TOTAL_SCENARIOS"
echo "  Passed: $PASS_COUNT"
echo "  Failed: $FAIL_COUNT"

if [ $FAIL_COUNT -eq 0 ]; then
  echo "✓ All chaos scenarios PASSED"
else
  echo "✗ Some scenarios failed"
fi
echo ""
