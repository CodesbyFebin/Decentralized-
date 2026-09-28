#!/bin/bash
# P1-LOCAL-VM-A01: Master Qualification Script
# End-to-end qualification workflow for local QEMU cluster
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="/tmp/p1-qualification-$(date +%s)"
mkdir -p "$LOG_DIR"

log_step() {
  local step="$1"
  local msg="$2"
  echo "[STEP $step] $msg" | tee -a "$LOG_DIR/master.log"
}

log_result() {
  local result="$1"
  local msg="$2"
  echo "[$result] $msg" | tee -a "$LOG_DIR/master.log"
}

die() {
  local step="$1"
  local msg="$2"
  log_result "FAIL" "Step $step: $msg"
  echo "=== DIAGNOSTICS ===" >> "$LOG_DIR/master.log"
  if [ -f "$SCRIPT_DIR/validation/local-vm/state/cluster.json" ]; then
    echo "Cluster status:" >> "$LOG_DIR/master.log"
    jq '.' "$SCRIPT_DIR/validation/local-vm/state/cluster.json" >> "$LOG_DIR/master.log" 2>&1 || true
  fi
  echo "Log directory: $LOG_DIR"
  exit 1
}

cd "$SCRIPT_DIR"

# Step 1: Environment verify
log_step 1 "Environment verification"
if ! command -v qemu-system-x86_64 >/dev/null 2>&1; then
  die 1 "qemu-system-x86_64 not found"
fi
if ! command -v jq >/dev/null 2>&1; then
  die 1 "jq not found"
fi
log_result "PASS" "Step 1: Environment verified"

# Step 2: Cluster create
log_step 2 "Create VM cluster"
if ! ./validation/local-vm/scripts/create-vm-cluster.sh >"$LOG_DIR/create-vm-cluster.log" 2>&1; then
  die 2 "Failed to create VM cluster"
fi
log_result "PASS" "Step 2: Cluster created"

# Step 3: Start VMs
log_step 3 "Start VM cluster"
if ! ./validation/local-vm/scripts/start-cluster.sh >"$LOG_DIR/start-cluster.log" 2>&1; then
  die 3 "Failed to start cluster"
fi
log_result "PASS" "Step 3: Cluster started"

# Step 4: Bootstrap with retries
log_step 4 "Bootstrap nodes (3 retries)"
BOOTSTRAP_ATTEMPTS=3
for attempt in $(seq 1 $BOOTSTRAP_ATTEMPTS); do
  log_result "ATTEMPT" "Bootstrap attempt $attempt/$BOOTSTRAP_ATTEMPTS"
  if ./validation/local-vm/scripts/bootstrap-nodes.sh >"$LOG_DIR/bootstrap-attempt-$attempt.log" 2>&1; then
    log_result "PASS" "Step 4: Bootstrap succeeded on attempt $attempt"
    break
  elif [ $attempt -lt $BOOTSTRAP_ATTEMPTS ]; then
    log_result "RETRY" "Bootstrap failed; waiting 30s before retry..."
    sleep 30
  else
    die 4 "Bootstrap failed after $BOOTSTRAP_ATTEMPTS attempts"
  fi
done

# Step 5: Network qualify
log_step 5 "Qualify network connectivity"
if ! ./validation/local-vm/scripts/qualify-network.sh >"$LOG_DIR/qualify-network.log" 2>&1; then
  die 5 "Network qualification failed"
fi
log_result "PASS" "Step 5: Network qualified"

# Step 6: Install socat on all nodes
log_step 6 "Install socat on all nodes"
CLUSTER_JSON="./validation/local-vm/state/cluster.json"
NODES="$(jq -r '.nodes' "$CLUSTER_JSON")"
SSH_KEY="$HOME/.ssh/p1-local-vm"
SSH_USER="ubuntu"
for ((i=1;i<=NODES;i++)); do
  idx=$((i-1))
  port="$(jq -r ".node_details[$idx].ssh_port" "$CLUSTER_JSON")"
  node_name="$(jq -r ".node_details[$idx].name" "$CLUSTER_JSON")"
  if ! ssh -i "$SSH_KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$port" "$SSH_USER@localhost" "sudo apt-get update -qq && sudo apt-get install -y socat 2>&1 | tail -5" >"$LOG_DIR/socat-install-$i.log" 2>&1; then
    log_result "WARN" "socat install on $node_name may have partial failure; continuing"
  fi
done
log_result "PASS" "Step 6: socat installation attempted on all nodes"

# Step 7: Initialize resource ledger
log_step 7 "Initialize resource ledger"
if ! ./validation/local-vm/scripts/init-resourceledger.sh >"$LOG_DIR/init-resourceledger.log" 2>&1; then
  die 7 "Resource ledger initialization failed"
fi
log_result "PASS" "Step 7: Resource ledger initialized"

# Step 8: Smoke test (15s)
log_step 8 "Smoke test (15s traffic)"
if ! ./validation/local-vm/scripts/launch-workload.sh workload-api-01 >"$LOG_DIR/smoke-test.log" 2>&1; then
  log_result "WARN" "Smoke test workload failed; inspecting logs"
  if [ -f "./validation/local-vm/state/workload-logs/workload-api-01-traffic.log" ]; then
    echo "Traffic log:" >> "$LOG_DIR/master.log"
    tail -20 "./validation/local-vm/state/workload-logs/workload-api-01-traffic.log" >> "$LOG_DIR/master.log"
  fi
  die 8 "Smoke test traffic failed"
fi
log_result "PASS" "Step 8: Smoke test passed"

# Step 9: Baseline run (120s)
log_step 9 "Baseline (3 workloads, 120s)"
if ! ./validation/local-vm/scripts/launch-baseline-workload.sh >"$LOG_DIR/baseline.log" 2>&1; then
  die 9 "Baseline run failed"
fi
log_result "PASS" "Step 9: Baseline completed"

# Final result
echo "=== P1-LOCAL-VM-A01 QUALIFICATION COMPLETE ===" | tee -a "$LOG_DIR/master.log"
echo "Status: PASS"
echo "Log directory: $LOG_DIR"
exit 0
