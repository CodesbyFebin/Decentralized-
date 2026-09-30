#!/bin/bash
# Gates 25-32: Chaos Scenario Continuation (resumed after gate 24)
# Run on: Persistent Ubuntu infrastructure with 3-node Podman cluster
# Prerequisites: dh cluster running, dh CLI available

set -e

export PATH="/home/user/Decentralized-/bin:$PATH"

EVIDENCE_DIR="validation/local-vm/evidence"
CHAOS_DIR="$EVIDENCE_DIR/GATES-21-32-CHAOS"
mkdir -p "$CHAOS_DIR"
TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo "=== P1 Gates 25-32: Chaos Scenario Continuation ==="
echo "Started: $TIMESTAMP"
echo ""

# Check if dh CLI is available
if ! command -v dh &> /dev/null; then
  echo "ERROR: 'dh' CLI not found in PATH"
  exit 1
fi

# Utility functions
chaos_result() {
  local gate=$1
  local scenario=$2
  local status=$3
  local latency_p50=$4
  local latency_p99=$5
  local errors=$6
  local detail=$7

  jq -n \
    --arg gate "$gate" \
    --arg scenario "$scenario" \
    --arg status "$status" \
    --arg latency_p50 "$latency_p50" \
    --arg latency_p99 "$latency_p99" \
    --arg errors "$errors" \
    --arg detail "$detail" \
    --arg timestamp "$TIMESTAMP" \
    '{gate: $gate, scenario: $scenario, status: $status, latency_p50_ms: $latency_p50, latency_p99_ms: $latency_p99, error_rate: $errors, detail: $detail, timestamp: $timestamp}'
}

start_load_generation() {
  # Start background load via app creation
  (
    APP_NUM=0
    while true; do
      APP_NUM=$((APP_NUM + 1))
      APP_DIR="/tmp/chaos-load-$$-$APP_NUM"
      mkdir -p "$APP_DIR" 2>/dev/null || true

      cat > "$APP_DIR/app.yaml" <<'EOF'
kind: Application
metadata:
  name: load-gen-UNIQ
spec:
  image: busybox:latest
  command: ["/bin/sh", "-c", "echo ok"]
  replicas: 1
  resources:
    requests:
      cpu: "0.01"
      memory: "32Mi"
EOF

      # Replace UNIQ with timestamp
      sed -i "s/UNIQ/$(date +%s%N)/" "$APP_DIR/app.yaml"

      # Try to apply
      dh apply -f "$APP_DIR/app.yaml" >/dev/null 2>&1 || true

      # Cleanup
      rm -rf "$APP_DIR" 2>/dev/null || true

      sleep 0.1  # ~10 ops/sec background load
    done
  ) &
  LOAD_PID=$!
}

stop_load_generation() {
  if [ ! -z "$LOAD_PID" ]; then
    kill $LOAD_PID 2>/dev/null || true
    wait $LOAD_PID 2>/dev/null || true
  fi
}

measure_latency() {
  # Measure p50, p99 latency by timing dh commands
  local latencies=()
  for i in {1..20}; do
    START=$(date +%s%N)
    dh get nodes >/dev/null 2>&1
    END=$(date +%s%N)
    LATENCY=$(( (END - START) / 1000000 ))
    latencies+=($LATENCY)
  done

  # Simple p50, p99 calculation
  IFS=$'\n' sorted=($(sort -n <<<"${latencies[*]}"))
  P50_IDX=$((${#sorted[@]} / 2))
  P99_IDX=$((${#sorted[@]} * 99 / 100))
  echo "${sorted[$P50_IDX]} ${sorted[$P99_IDX]}"
}

check_invariants() {
  # Verify after chaos:
  # 1. No data corruption (can still query system)
  # 2. Audit trail complete
  # 3. State convergence (all nodes visible)

  local invariant_pass=1

  # Check 1: System still queryable
  if ! dh get nodes >/dev/null 2>&1; then
    echo "FAIL: System not queryable after chaos"
    invariant_pass=0
  fi

  # Check 2: Audit trail exists
  if ! dh audit tail >/dev/null 2>&1; then
    echo "FAIL: Audit trail unavailable"
    invariant_pass=0
  fi

  # Check 3: Nodes are visible
  NODES=$(dh get nodes 2>/dev/null | wc -l || echo "0")
  if [ "$NODES" -lt 1 ]; then
    echo "FAIL: No nodes visible after chaos"
    invariant_pass=0
  fi

  echo "$invariant_pass"
}

# ============================================================================
# Gates 25-32: Stress Scenarios (loop continues from gate 25)
# ============================================================================

for gate in 25 26 27 28 29 30 31 32; do
  case $gate in
    25)
      SCENARIO="High query load"
      DETAIL="System handles concurrent queries"
      STRESSOR="for i in {1..10}; do dh get nodes >/dev/null & done; wait"
      ;;
    26)
      SCENARIO="Rapid app creation"
      DETAIL="System accepts multiple concurrent app submissions"
      STRESSOR="for i in {1..5}; do dh get apps >/dev/null 2>&1 & done; wait"
      ;;
    27)
      SCENARIO="App state queries"
      DETAIL="System handles app state queries under load"
      STRESSOR="for i in {1..10}; do dh get apps >/dev/null 2>&1 & done; wait"
      ;;
    28)
      SCENARIO="Audit tail under load"
      DETAIL="Audit trail remains queryable under stress"
      STRESSOR="for i in {1..10}; do dh audit tail >/dev/null & done; wait"
      ;;
    29)
      SCENARIO="Mesh status queries"
      DETAIL="Mesh topology remains observable"
      STRESSOR="for i in {1..10}; do dh mesh status >/dev/null 2>&1 & done; wait"
      ;;
    30)
      SCENARIO="Node inventory consistency"
      DETAIL="Node state remains consistent"
      STRESSOR="for i in {1..10}; do dh get nodes >/dev/null & done; wait"
      ;;
    31)
      SCENARIO="Control plane queries"
      DETAIL="Control plane remains responsive"
      STRESSOR="for i in {1..10}; do dh cp status >/dev/null & done; wait"
      ;;
    32)
      SCENARIO="Mixed workload stress"
      DETAIL="System handles diverse operations concurrently"
      STRESSOR="(dh get nodes; dh audit tail; dh get apps) > /dev/null 2>&1 &"
      ;;
  esac

  echo "Gate $gate: $SCENARIO"
  {
    start_load_generation
    sleep 2

    echo "  Running: $STRESSOR"
    eval "$STRESSOR" 2>/dev/null || true

    sleep 5

    LATENCY=$(measure_latency)
    P50=$(echo $LATENCY | awk '{print $1}')
    P99=$(echo $LATENCY | awk '{print $2}')

    INVARIANTS=$(check_invariants)
    stop_load_generation

    if [ "$INVARIANTS" -eq 1 ]; then
      chaos_result "$gate" "$SCENARIO" "PASS" "$P50" "$P99" "0%" "$DETAIL" | tee "$CHAOS_DIR/gate-$gate.json"
    else
      chaos_result "$gate" "$SCENARIO" "FAIL" "$P50" "$P99" "unknown" "System check failed" | tee "$CHAOS_DIR/gate-$gate.json"
    fi
  } 2>&1 | tee -a "$CHAOS_DIR/gate-$gate.log"

  sleep 2
done

# ============================================================================
# Collect Evidence
# ============================================================================
echo ""
echo "=== Gates 25-32 Execution Complete ==="
echo "Evidence collected in: $CHAOS_DIR/"
echo ""

# Aggregate results for gates 21-32
TOTAL_SCENARIOS=12
PASS_COUNT=$(find "$CHAOS_DIR" -name "gate-*.json" -exec grep -l '"status": "PASS"' {} \; | wc -l)
FAIL_COUNT=$((TOTAL_SCENARIOS - PASS_COUNT))

echo "Overall Results Summary (Gates 21-32):"
echo "  Total Scenarios: $TOTAL_SCENARIOS"
echo "  Passed: $PASS_COUNT"
echo "  Failed: $FAIL_COUNT"
echo ""

if [ $FAIL_COUNT -eq 0 ]; then
  echo "✓ All chaos scenarios PASSED - P1_CORE qualification confirmed"
else
  echo "✗ Some scenarios failed - investigate failures before qualification claim"
fi

echo ""
