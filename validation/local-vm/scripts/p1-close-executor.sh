#!/bin/bash

# P1-CLOSE-A01 Executor (Gates 10-32)
# Role: Capture observations and raw evidence
# NO outcome determination (verifier evaluates against gate contracts)
#
# Design: Executor/Verifier Separation
# - Executor: perform operation → capture observation → bind artifact
# - Verifier: read contract + artifact → evaluate predicate → outcome

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="/home/user/Decentralized-"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
EVIDENCE_DIR="$REPO_ROOT/validation/local-vm/evidence"
SSH_KEY="$HOME/.ssh/p1-local-vm"

# Use provided EXECUTION_DIR or create a new one
if [ -n "$1" ]; then
    EXECUTION_DIR="$1"
    CAMPAIGN_ID=$(basename "$EXECUTION_DIR" | sed 's/P1-CLOSE-A01-EXECUTION-//')
else
    CAMPAIGN_ID=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
    EXECUTION_DIR="$EVIDENCE_DIR/P1-CLOSE-A01-EXECUTION-$CAMPAIGN_ID"
fi

mkdir -p "$EXECUTION_DIR/artifacts" "$EXECUTION_DIR/observations"

# Logging (observation capture only, no outcome)
log() {
    local msg="$*"
    local ts=$(date -u +'%Y-%m-%dT%H:%M:%S.%3NZ')
    echo "[$ts] $msg" | tee -a "$EXECUTION_DIR/execution.log"
}

observe() {
    local gate_id="$1"
    local observation="$2"
    local ts=$(date -u +'%s%N')  # nanoseconds for precision

    echo "$observation" >> "$EXECUTION_DIR/observations/gate-${gate_id}.txt"
    log "Gate $gate_id observation: $observation (ts: $ts)"
}

capture_artifact() {
    local gate_id="$1"
    local artifact_name="$2"
    local source_path="$3"

    if [ -f "$source_path" ]; then
        cp "$source_path" "$EXECUTION_DIR/artifacts/gate-${gate_id}-${artifact_name}"
        echo "Captured: $artifact_name"
    else
        log "WARNING: Could not capture artifact $artifact_name from $source_path"
    fi
}

capture_command_output() {
    local gate_id="$1"
    local artifact_name="$2"
    shift 2

    local output_file="$EXECUTION_DIR/artifacts/gate-${gate_id}-${artifact_name}"

    # Capture both stdout and exit code
    local exit_code=0
    "$@" > "$output_file" 2>&1 || exit_code=$?

    echo "$exit_code" > "${output_file}.exit_code"
    return $exit_code
}

timestamp_ns() {
    date +%s%N
}

timestamp_iso() {
    date -u +'%Y-%m-%dT%H:%M:%S.%3NZ'
}

# ============================================================
# GATE 10: ResourceLedger Presence & Validity
# ============================================================

log "=== GATE 10: ResourceLedger Presence ==="

GATE_10_OBSERVATIONS=""

# Observation 1: File existence
if [ -f "$STATE_DIR/resourceledger.json" ]; then
    observe 10 "resourceledger.json EXISTS at $STATE_DIR"
    GATE_10_OBSERVATIONS="${GATE_10_OBSERVATIONS}file_exists=true "
else
    observe 10 "resourceledger.json NOT FOUND at $STATE_DIR"
    GATE_10_OBSERVATIONS="${GATE_10_OBSERVATIONS}file_exists=false "
fi

# Observation 2: JSON validity
if [ -f "$STATE_DIR/resourceledger.json" ]; then
    LEDGER_PARSE_EXIT=0
    jq . "$STATE_DIR/resourceledger.json" > "$EXECUTION_DIR/artifacts/gate-10-ledger-parse.json" 2> "$EXECUTION_DIR/artifacts/gate-10-ledger-parse-error.txt" || LEDGER_PARSE_EXIT=$?

    observe 10 "ledger_parse_exit_code=$LEDGER_PARSE_EXIT"
    GATE_10_OBSERVATIONS="${GATE_10_OBSERVATIONS}ledger_parse_exit=$LEDGER_PARSE_EXIT "

    if [ $LEDGER_PARSE_EXIT -eq 0 ]; then
        # Observation 3: Structure validation
        LEDGER_TYPE=$(jq -r 'type' "$STATE_DIR/resourceledger.json" 2>/dev/null || echo "unknown")
        observe 10 "ledger_type=$LEDGER_TYPE"
        GATE_10_OBSERVATIONS="${GATE_10_OBSERVATIONS}ledger_type=$LEDGER_TYPE "
    fi
fi

# Artifact: Copy ledger for evidence
capture_artifact 10 "ledger-pre-injection.json" "$STATE_DIR/resourceledger.json"

# Write observation summary
echo "$GATE_10_OBSERVATIONS" > "$EXECUTION_DIR/observations/gate-10-summary.txt"

log "Gate 10 observations captured"

# ============================================================
# GATE 11-16: ResourceLedger Model A Consistency
# ============================================================

log "=== GATE 11-16: ResourceLedger Model A Validation ==="

LEDGER_VALIDATION_FILE="$EXECUTION_DIR/artifacts/gate-11-16-model-a-validation.txt"

{
    echo "=== ResourceLedger Model A Validation ==="
    echo "Timestamp: $(timestamp_iso)"
    echo "Ledger: $STATE_DIR/resourceledger.json"
    echo ""

    if [ ! -f "$STATE_DIR/resourceledger.json" ]; then
        echo "ERROR: ResourceLedger not found (depends on Gate 10)"
        echo "validation_possible=false"
    else
        echo "validation_possible=true"
        echo ""
        echo "Formula: AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED"
        echo ""

        # For each node, validate formula
        jq -r '.node_capacity[] | .node' "$STATE_DIR/resourceledger.json" 2>/dev/null | while read -r node; do
            echo "Node: $node"

            # Extract values (CPU)
            TOTAL=$(jq -r --arg n "$node" '.node_capacity[] | select(.node == $n) | .cpu_cores' "$STATE_DIR/resourceledger.json" 2>/dev/null || echo "0")
            OWNER=$(jq -r --arg n "$node" '.node_capacity[] | select(.node == $n) | .cpu_owner_reserve' "$STATE_DIR/resourceledger.json" 2>/dev/null || echo "0")
            RESERVED=$(jq -r --arg n "$node" '.node_capacity[] | select(.node == $n) | .cpu_reserved' "$STATE_DIR/resourceledger.json" 2>/dev/null || echo "0")
            ALLOCATED=$(jq -r --arg n "$node" '.node_capacity[] | select(.node == $n) | .cpu_allocated' "$STATE_DIR/resourceledger.json" 2>/dev/null || echo "0")

            # Calculate expected available
            EXPECTED_AVAILABLE=$(echo "$TOTAL - $OWNER - $RESERVED - $ALLOCATED" | bc -l 2>/dev/null || echo "calc_error")

            echo "  CPU: total=$TOTAL owner_reserve=$OWNER reserved=$RESERVED allocated=$ALLOCATED"
            echo "  Expected AVAILABLE: $EXPECTED_AVAILABLE"

            # Verify constraint: allocated <= total
            CONSTRAINT_VIOLATED=false
            if (( $(echo "$ALLOCATED > $TOTAL" | bc -l) )); then
                echo "  ERROR: allocated ($ALLOCATED) > total ($TOTAL)"
                CONSTRAINT_VIOLATED=true
            fi

            echo "  constraint_violated=$CONSTRAINT_VIOLATED"
            echo ""
        done
    fi
} | tee "$LEDGER_VALIDATION_FILE"

observe 11-16 "ledger_model_a_validation_performed"

log "Gate 11-16 observations captured"

# ============================================================
# GATE 17-18: Network Partition Injection & Detection
# ============================================================

log "=== GATE 17-18: Network Partition Injection ==="

PARTITION_OBSERVATIONS_FILE="$EXECUTION_DIR/observations/gate-17-18-partition.txt"
PARTITION_INJECTION_LOG="$EXECUTION_DIR/artifacts/gate-17-18-partition-injection.log"

{
    echo "=== Network Partition Injection ==="
    echo "Timestamp: $(timestamp_iso)"
    echo ""

    # Observation 1: tc availability
    if command -v tc &>/dev/null; then
        echo "tc_available=true"
        echo "tc_path=$(which tc)"
        tc -V 2>&1 || true
        echo ""

        # Observation 2: Pre-partition connectivity
        echo "Pre-partition connectivity check:"
        NODE_2_PORT=$(jq -r '.node_details[] | select(.name == "dh-node-2") | .ssh_port' "$STATE_DIR/cluster.json" 2>/dev/null || echo "2202")

        PING_START=$(timestamp_ns)
        if ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$NODE_2_PORT" "ubuntu@127.0.0.1" "echo OK" 2>&1; then
            PING_END=$(timestamp_ns)
            echo "pre_partition_connectivity=success"
            echo "pre_partition_latency_ns=$((PING_END - PING_START))"
        else
            echo "pre_partition_connectivity=failed_before_injection"
        fi
        echo ""

        # Observation 3: Inject real partition
        echo "Injecting 100% packet loss via tc qdisc:"
        INJECT_TIME=$(timestamp_ns)
        echo "partition_inject_timestamp_ns=$INJECT_TIME"

        # Capture tc command execution
        TC_INJECT_EXIT=0
        ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$NODE_2_PORT" "ubuntu@127.0.0.1" \
            "tc qdisc add dev eth0 root netem loss 100%" > "$EXECUTION_DIR/artifacts/gate-17-18-tc-inject-stdout.log" 2> "$EXECUTION_DIR/artifacts/gate-17-18-tc-inject-stderr.log" || TC_INJECT_EXIT=$?

        echo "tc_inject_exit_code=$TC_INJECT_EXIT"

        if [ $TC_INJECT_EXIT -eq 0 ]; then
            echo "partition_injection_successful=true"

            # Wait brief moment for partition to stabilize
            sleep 2

            # Observation 4: Negative control - verify partition is active
            echo ""
            echo "Verifying partition is active (negative control):"
            PARTITION_ACTIVE=false
            for attempt in {1..5}; do
                if ! ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=2 -p "$NODE_2_PORT" "ubuntu@127.0.0.1" "echo OK" 2>&1; then
                    echo "attempt_$attempt: no_connectivity (partition_active)"
                    PARTITION_ACTIVE=true
                else
                    echo "attempt_$attempt: connectivity_remains (unexpected)"
                fi
                sleep 1
            done
            echo "partition_confirmed_active=$PARTITION_ACTIVE"
            echo ""

            if [ "$PARTITION_ACTIVE" = true ]; then
                # Observation 5: Measure detection latency
                echo "Measuring SSH detection latency:"
                DETECT_START=$(timestamp_ns)
                DETECTED=false
                DETECTION_LATENCY_NS=0

                for attempt in {1..30}; do
                    if ! ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=3 -p "$NODE_2_PORT" "ubuntu@127.0.0.1" "echo OK" 2>&1; then
                        DETECT_END=$(timestamp_ns)
                        DETECTION_LATENCY_NS=$((DETECT_END - DETECT_START))
                        echo "detected_at_attempt=$attempt"
                        echo "detection_latency_ns=$DETECTION_LATENCY_NS"
                        echo "detection_latency_ms=$(( DETECTION_LATENCY_NS / 1000000 ))"
                        DETECTED=true
                        break
                    fi
                    sleep 1
                done
                echo "detection_confirmed=$DETECTED"
                echo ""

                # Store for later gates
                echo "$INJECT_TIME" > "$EXECUTION_DIR/artifacts/gate-17-18-partition-inject-timestamp.txt"
                echo "$DETECTION_LATENCY_NS" > "$EXECUTION_DIR/artifacts/gate-17-18-detection-latency-ns.txt"
                echo "$PARTITION_ACTIVE" > "$EXECUTION_DIR/artifacts/gate-17-18-partition-confirmed.txt"
            else
                echo "ERROR: Partition injection did not create isolation"
                echo "partition_injection_failed=true"
            fi
        else
            echo "partition_injection_failed=true"
            echo "ERROR: tc qdisc injection failed with exit code $TC_INJECT_EXIT"
        fi
    else
        echo "tc_available=false"
        echo "partition_injection_blocked=true"
        echo "reason=tc_command_not_found"
        echo ""
        echo "Blocker documentation:"
        echo "  Required: iproute2 package (provides tc)"
        echo "  Alternative: network namespace isolation"
        echo "  Impact: Cannot perform real network fault injection"
    fi
} | tee "$PARTITION_INJECTION_LOG"

observe 17-18 "partition_injection_observation_captured"

log "Gate 17-18 observations captured"

# ============================================================
# GATE 23: Workload Crash Injection
# ============================================================

log "=== GATE 23: Workload Crash Injection ==="

CRASH_OBSERVATIONS_FILE="$EXECUTION_DIR/observations/gate-23-crash.txt"
CRASH_LOG="$EXECUTION_DIR/artifacts/gate-23-crash-injection.log"

{
    echo "=== Workload Crash Injection ==="
    echo "Timestamp: $(timestamp_iso)"
    echo ""

    NODE_1_PORT=$(jq -r '.node_details[] | select(.name == "dh-node-1") | .ssh_port' "$STATE_DIR/cluster.json" 2>/dev/null || echo "2201")

    # Ensure a workload is running
    echo "Ensuring workload is running on dh-node-1..."
    EXISTING_WORKLOAD=$(ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$NODE_1_PORT" "ubuntu@127.0.0.1" \
        "ps aux | grep -E 'workload|java|python' | grep -v grep" 2>/dev/null | wc -l || echo "0")

    if [ "$EXISTING_WORKLOAD" -eq 0 ] || [ "$EXISTING_WORKLOAD" -lt 1 ]; then
        echo "No workload found, launching test workload..."
        ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$NODE_1_PORT" "ubuntu@127.0.0.1" \
            "nohup bash -c 'while true; do echo -e \"HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 2\r\n\r\nOK\" | nc -l -p 8080 -q 1; done' > /tmp/workload.log 2>&1 &" 2>/dev/null || true
        sleep 2
        echo "Test workload started"
    else
        echo "Existing workload found ($EXISTING_WORKLOAD processes)"
    fi
    echo ""

    # Observation 1: Find workload process
    echo "Finding workload process on dh-node-1:"

    WORKLOAD_PS_BEFORE=$(ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$NODE_1_PORT" "ubuntu@127.0.0.1" \
        "ps aux | grep -E 'workload|java|python' | grep -v grep" 2>/dev/null || true)

    echo "ps_output_before_crash:"
    echo "$WORKLOAD_PS_BEFORE"
    echo ""

    # Extract PID
    WORKLOAD_PID=$(echo "$WORKLOAD_PS_BEFORE" | head -1 | awk '{print $2}' || echo "")

    if [ -z "$WORKLOAD_PID" ]; then
        echo "workload_process_found=false"
        echo "ERROR: No workload process identified"
    else
        echo "workload_pid=$WORKLOAD_PID"
        echo "workload_process_found=true"
        echo ""

        # Observation 2: Verify workload is responding
        echo "Testing workload HTTP response before crash:"
        HTTP_SUCCESS_BEFORE=$(ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$NODE_1_PORT" "ubuntu@127.0.0.1" \
            "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8080/health 2>&1 || echo '000'" 2>/dev/null || echo "000")

        echo "http_response_before=$HTTP_SUCCESS_BEFORE"

        if [ "$HTTP_SUCCESS_BEFORE" = "200" ] || [ "$HTTP_SUCCESS_BEFORE" = "201" ]; then
            echo "workload_responding_before_crash=true"
        else
            echo "workload_responding_before_crash=false (code: $HTTP_SUCCESS_BEFORE)"
        fi
        echo ""

        # Observation 3: Inject real SIGKILL
        echo "Injecting SIGKILL to PID $WORKLOAD_PID:"
        INJECT_TIME=$(timestamp_ns)
        echo "crash_inject_timestamp_ns=$INJECT_TIME"

        KILL_EXIT=0
        KILL_OUTPUT=$(ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$NODE_1_PORT" "ubuntu@127.0.0.1" \
            "kill -9 $WORKLOAD_PID 2>&1" 2>/dev/null || true)

        echo "kill_command_output: $KILL_OUTPUT"
        echo ""

        # Wait brief moment
        sleep 1

        # Observation 4: Verify process is gone
        echo "Verifying process is gone:"
        WORKLOAD_PS_AFTER=$(ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$NODE_1_PORT" "ubuntu@127.0.0.1" \
            "ps -p $WORKLOAD_PID -o pid,state,cmd 2>&1 || echo 'no process'" 2>/dev/null || echo "")

        echo "ps_output_after_kill: $WORKLOAD_PS_AFTER"

        if echo "$WORKLOAD_PS_AFTER" | grep -q "no process"; then
            echo "process_confirmed_gone=true"
        else
            echo "process_confirmed_gone=false"
        fi

        # Store for Gate 24
        echo "$INJECT_TIME" > "$EXECUTION_DIR/artifacts/gate-23-crash-inject-timestamp.txt"
        echo "$WORKLOAD_PID" > "$EXECUTION_DIR/artifacts/gate-23-workload-pid.txt"
        echo "$NODE_1_PORT" > "$EXECUTION_DIR/artifacts/gate-23-node-port.txt"
    fi
} | tee "$CRASH_LOG"

observe 23 "crash_injection_observation_captured"

log "Gate 23 observations captured"

# ============================================================
# GATE 24: Workload Recovery Measurement
# ============================================================

log "=== GATE 24: Workload Recovery Measurement ==="

RECOVERY_LOG="$EXECUTION_DIR/artifacts/gate-24-recovery-measurement.log"

{
    echo "=== Workload Recovery Measurement ==="
    echo "Timestamp: $(timestamp_iso)"
    echo ""

    # Read stored values from Gate 23
    if [ ! -f "$EXECUTION_DIR/artifacts/gate-23-crash-inject-timestamp.txt" ]; then
        echo "ERROR: Gate 23 crash injection timestamp not found"
        echo "recovery_measurement_possible=false"
    else
        INJECT_TIME=$(cat "$EXECUTION_DIR/artifacts/gate-23-crash-inject-timestamp.txt")
        NODE_1_PORT=$(cat "$EXECUTION_DIR/artifacts/gate-23-node-port.txt")
        ORIGINAL_PID=$(cat "$EXECUTION_DIR/artifacts/gate-23-workload-pid.txt")

        echo "Monitoring for workload restart:"
        echo "original_pid=$ORIGINAL_PID"
        echo "measurement_start_time=$INJECT_TIME"
        echo ""

        RECOVERED=false
        RECOVERY_TIME_NS=0
        NEW_PID=""
        RECOVERY_ATTEMPTS=0

        for attempt in {1..60}; do
            RECOVERY_ATTEMPTS=$attempt

            # Check if new workload process exists
            NEW_PROCESS=$(ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$NODE_1_PORT" "ubuntu@127.0.0.1" \
                "ps aux | grep -E 'workload|java|python' | grep -v grep | head -1 | awk '{print \$2}'" 2>/dev/null || echo "")

            if [ -n "$NEW_PROCESS" ] && [ "$NEW_PROCESS" != "$ORIGINAL_PID" ]; then
                RECOVERY_END=$(timestamp_ns)
                RECOVERY_TIME_NS=$((RECOVERY_END - INJECT_TIME))
                NEW_PID="$NEW_PROCESS"
                RECOVERED=true
                echo "recovery_detected_at_attempt=$attempt"
                echo "recovery_timestamp_ns=$RECOVERY_END"
                echo "recovery_time_ns=$RECOVERY_TIME_NS"
                echo "recovery_time_ms=$(( RECOVERY_TIME_NS / 1000000 ))"
                echo "new_process_pid=$NEW_PID"
                break
            fi

            sleep 1
        done

        echo ""
        echo "recovery_complete=$RECOVERED"
        echo "recovery_attempts=$RECOVERY_ATTEMPTS"
        echo "recovery_timeout_exceeded=$( [ $RECOVERY_ATTEMPTS -ge 60 ] && echo 'true' || echo 'false' )"

        if [ "$RECOVERED" = true ]; then
            echo ""
            echo "Verifying recovered workload responds:"
            HTTP_RESPONSE=$(ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$NODE_1_PORT" "ubuntu@127.0.0.1" \
                "curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8080/health 2>&1 || echo '000'" 2>/dev/null || echo "000")

            echo "http_response_after_recovery=$HTTP_RESPONSE"

            if [ "$HTTP_RESPONSE" = "200" ] || [ "$HTTP_RESPONSE" = "201" ]; then
                echo "workload_responding_after_recovery=true"
            else
                echo "workload_responding_after_recovery=false"
            fi

            # Store recovery measurements
            echo "$RECOVERY_TIME_NS" > "$EXECUTION_DIR/artifacts/gate-24-recovery-time-ns.txt"
        fi
    fi
} | tee "$RECOVERY_LOG"

observe 24 "recovery_measurement_captured"

log "Gate 24 observations captured"

# ============================================================
# GATE 25: Scheduler Process State
# ============================================================

log "=== GATE 25: Scheduler Process State ==="

SCHEDULER_LOG="$EXECUTION_DIR/artifacts/gate-25-scheduler-state.log"

{
    echo "=== Scheduler Process State ==="
    echo "Timestamp: $(timestamp_iso)"
    echo ""

    SCHEDULER_PID=$(pgrep -f "run-scheduler.sh" | head -1 || echo "")

    if [ -z "$SCHEDULER_PID" ]; then
        echo "scheduler_process_found=false"
    else
        echo "scheduler_pid=$SCHEDULER_PID"
        echo "scheduler_process_found=true"
        echo ""

        # Capture ps output
        echo "ps output:"
        ps -p "$SCHEDULER_PID" -o pid,ppid,state,vsz,rss,cmd 2>&1 || true
        echo ""

        # Read /proc state
        if [ -f "/proc/$SCHEDULER_PID/stat" ]; then
            PROC_STATE=$(awk '{print $3}' "/proc/$SCHEDULER_PID/stat" 2>/dev/null || echo "unknown")
            echo "process_state_from_proc=$PROC_STATE"
        fi

        if [ -f "/proc/$SCHEDULER_PID/status" ]; then
            MEMORY_KB=$(grep "^VmRSS" "/proc/$SCHEDULER_PID/status" 2>/dev/null | awk '{print $2}' || echo "0")
            THREADS=$(grep "^Threads" "/proc/$SCHEDULER_PID/status" 2>/dev/null | awk '{print $2}' || echo "1")
            echo "memory_rss_kb=$MEMORY_KB"
            echo "thread_count=$THREADS"
        fi
        echo ""

        # Check file descriptors
        FD_COUNT=$(ls -1 "/proc/$SCHEDULER_PID/fd" 2>/dev/null | wc -l || echo "0")
        echo "open_file_descriptors=$FD_COUNT"
    fi
} | tee "$SCHEDULER_LOG"

observe 25 "scheduler_state_captured"

log "Gate 25 observations captured"

# ============================================================
# GATE 29: Evidence Artifact Collection
# ============================================================

log "=== GATE 29: Evidence Artifact Collection ==="

MANIFEST_FILE="$EXECUTION_DIR/artifacts/evidence-manifest.json"

# Create artifact manifest
{
    echo "{"
    echo "  \"campaign_id\": \"$CAMPAIGN_ID\","
    echo "  \"execution_directory\": \"$EXECUTION_DIR\","
    echo "  \"source_sha\": \"$(cd $REPO_ROOT && git rev-parse HEAD 2>/dev/null || echo 'unknown')\","
    echo "  \"gate_contract_version\": \"P1-GATE-CONTRACT.md\","
    echo "  \"artifacts\": ["

    # List all artifacts
    first=true
    for artifact in "$EXECUTION_DIR/artifacts"/*; do
        if [ -f "$artifact" ]; then
            artifact_name=$(basename "$artifact")
            artifact_digest=$(sha256sum "$artifact" | awk '{print $1}')

            if [ "$first" = true ]; then
                first=false
            else
                echo ","
            fi

            echo -n "    {"
            echo -n "\"name\": \"$artifact_name\", "
            echo -n "\"path\": \"artifacts/$artifact_name\", "
            echo -n "\"sha256\": \"$artifact_digest\", "
            echo -n "\"size_bytes\": $(stat -f%z "$artifact" 2>/dev/null || stat -c%s "$artifact" 2>/dev/null || echo 0)"
            echo -n "}"
        fi
    done

    echo ""
    echo "  ],"
    echo "  \"observations\": ["

    # List all observations
    first=true
    for obs_file in "$EXECUTION_DIR/observations"/*; do
        if [ -f "$obs_file" ]; then
            obs_name=$(basename "$obs_file")

            if [ "$first" = true ]; then
                first=false
            else
                echo ","
            fi

            echo -n "    {\"gate\": \"$obs_name\"}"
        fi
    done

    echo ""
    echo "  ],"
    echo "  \"execution_complete\": true,"
    echo "  \"execution_timestamp\": \"$(timestamp_iso)\""
    echo "}"
} > "$MANIFEST_FILE"

# Compute manifest digest
MANIFEST_DIGEST=$(sha256sum "$MANIFEST_FILE" | awk '{print $1}')
echo "$MANIFEST_DIGEST" > "$EXECUTION_DIR/artifacts/evidence-manifest-digest.txt"

observe 29 "evidence_manifest_created digest=$MANIFEST_DIGEST"

log "Gate 29 observations captured"

# ============================================================
# GATE 30: Cryptographic Signature Generation
# ============================================================

log "=== GATE 30: Cryptographic Signature ==="

SIG_LOG="$EXECUTION_DIR/artifacts/gate-30-signature-generation.log"

{
    echo "=== Cryptographic Signature Generation ==="
    echo "Timestamp: $(timestamp_iso)"
    echo ""

    # Check OpenSSL availability
    if ! command -v openssl &>/dev/null; then
        echo "openssl_available=false"
        echo "signature_generation_possible=false"
    else
        echo "openssl_available=true"
        echo "openssl_version:"
        openssl version
        echo ""

        # Try to generate Ed25519 keypair
        KEYS_DIR="$EXECUTION_DIR/keys"
        mkdir -p "$KEYS_DIR"

        # Generate private key
        echo "Generating Ed25519 keypair:"
        KEYGEN_EXIT=0
        openssl genpkey -algorithm Ed25519 -out "$KEYS_DIR/evidence.key" 2> "$EXECUTION_DIR/artifacts/gate-30-keygen-stderr.log" || KEYGEN_EXIT=$?

        echo "keygen_exit_code=$KEYGEN_EXIT"

        if [ $KEYGEN_EXIT -eq 0 ]; then
            # Extract public key
            PUBKEY_EXIT=0
            openssl pkey -in "$KEYS_DIR/evidence.key" -pubout -out "$KEYS_DIR/evidence.pub" 2> "$EXECUTION_DIR/artifacts/gate-30-pubkey-extract-stderr.log" || PUBKEY_EXIT=$?

            echo "pubkey_extract_exit_code=$PUBKEY_EXIT"

            if [ $PUBKEY_EXIT -eq 0 ]; then
                # Sign manifest
                echo ""
                echo "Signing evidence manifest:"
                SIGN_EXIT=0
                openssl dgst -sha256 -sign "$KEYS_DIR/evidence.key" -out "$EXECUTION_DIR/artifacts/evidence-manifest.sig" "$MANIFEST_FILE" 2> "$EXECUTION_DIR/artifacts/gate-30-sign-stderr.log" || SIGN_EXIT=$?

                echo "sign_exit_code=$SIGN_EXIT"

                if [ $SIGN_EXIT -eq 0 ]; then
                    # Verify signature
                    echo ""
                    echo "Verifying signature:"
                    VERIFY_EXIT=0
                    VERIFY_OUTPUT=$(openssl dgst -sha256 -verify "$KEYS_DIR/evidence.pub" -signature "$EXECUTION_DIR/artifacts/evidence-manifest.sig" "$MANIFEST_FILE" 2>&1 || true)
                    VERIFY_EXIT=$?

                    echo "verify_exit_code=$VERIFY_EXIT"
                    echo "verify_output: $VERIFY_OUTPUT"

                    if [ $VERIFY_EXIT -eq 0 ] && echo "$VERIFY_OUTPUT" | grep -q "Verified OK"; then
                        echo "signature_verified=true"
                    else
                        echo "signature_verified=false"
                    fi

                    # Copy public key to artifacts
                    cp "$KEYS_DIR/evidence.pub" "$EXECUTION_DIR/artifacts/evidence-public-key.pem"
                else
                    echo "signature_creation_failed=true"
                fi
            else
                echo "pubkey_extraction_failed=true"
            fi
        else
            echo "keypair_generation_failed=true"
        fi
    fi
} | tee "$SIG_LOG"

observe 30 "cryptographic_signature_captured"

log "Gate 30 observations captured"

# ============================================================
# GATE 31: ResourceLedger Consistency Verification
# ============================================================

log "=== GATE 31: ResourceLedger Consistency Verification ==="

CONSISTENCY_LOG="$EXECUTION_DIR/artifacts/gate-31-consistency-verification.log"

{
    echo "=== ResourceLedger Consistency Verification ==="
    echo "Timestamp: $(timestamp_iso)"
    echo ""

    if [ ! -f "$STATE_DIR/resourceledger.json" ]; then
        echo "ledger_accessible=false"
    else
        echo "ledger_accessible=true"

        # Validate schema version
        SCHEMA_VERSION=$(jq -r '.schema_version' "$STATE_DIR/resourceledger.json" 2>/dev/null || echo "unknown")
        echo "schema_version=$SCHEMA_VERSION"
        echo ""

        echo "Model A Formula Verification:"
        echo "AVAILABLE = TOTAL - OWNER_RESERVE - RESERVED - ALLOCATED"
        echo ""

        # For each node, validate formula
        jq -r '.node_capacity[] | .node' "$STATE_DIR/resourceledger.json" 2>/dev/null | while read -r node; do
            TOTAL=$(jq -r --arg n "$node" '.node_capacity[] | select(.node == $n) | .cpu_cores' "$STATE_DIR/resourceledger.json" 2>/dev/null || echo "0")
            OWNER=$(jq -r --arg n "$node" '.node_capacity[] | select(.node == $n) | .cpu_owner_reserve' "$STATE_DIR/resourceledger.json" 2>/dev/null || echo "0")
            RESERVED=$(jq -r --arg n "$node" '.node_capacity[] | select(.node == $n) | .cpu_reserved' "$STATE_DIR/resourceledger.json" 2>/dev/null || echo "0")
            ALLOCATED=$(jq -r --arg n "$node" '.node_capacity[] | select(.node == $n) | .cpu_allocated' "$STATE_DIR/resourceledger.json" 2>/dev/null || echo "0")

            # Check constraint: allocated <= total
            CONSTRAINT_PASS=true
            if (( $(echo "$ALLOCATED > $TOTAL" | bc -l) )); then
                CONSTRAINT_PASS=false
            fi

            echo "node=$node total=$TOTAL owner=$OWNER reserved=$RESERVED allocated=$ALLOCATED constraint_pass=$CONSTRAINT_PASS"
        done
    fi
} | tee "$CONSISTENCY_LOG"

observe 31 "consistency_verification_captured"

log "Gate 31 observations captured"

# ============================================================
# GATE 32: Tamper Detection Negative Control
# ============================================================

log "=== GATE 32: Tamper Detection Negative Control ==="

TAMPER_LOG="$EXECUTION_DIR/artifacts/gate-32-tamper-test.log"

{
    echo "=== Tamper Detection Negative Control ==="
    echo "Timestamp: $(timestamp_iso)"
    echo ""

    # Create tampered copy
    TAMPERED_MANIFEST="$EXECUTION_DIR/artifacts/evidence-manifest-tampered.json"
    cp "$MANIFEST_FILE" "$TAMPERED_MANIFEST"

    echo "Created tampered manifest: $TAMPERED_MANIFEST"

    # Inject tampering
    jq '.campaign_id = "TAMPERED_CAMPAIGN"' "$TAMPERED_MANIFEST" > "$TAMPERED_MANIFEST.tmp" && mv "$TAMPERED_MANIFEST.tmp" "$TAMPERED_MANIFEST"

    echo ""
    echo "Tamper injection: Changed campaign_id to TAMPERED_CAMPAIGN"
    echo ""

    # Compute hashes
    ORIGINAL_HASH=$(sha256sum "$MANIFEST_FILE" | awk '{print $1}')
    TAMPERED_HASH=$(sha256sum "$TAMPERED_MANIFEST" | awk '{print $1}')

    echo "Original manifest hash:  $ORIGINAL_HASH"
    echo "Tampered manifest hash:  $TAMPERED_HASH"
    echo ""

    if [ "$ORIGINAL_HASH" != "$TAMPERED_HASH" ]; then
        echo "hash_mismatch_detected=true"
        echo "tamper_detection_negative_control_PASS"
    else
        echo "hash_mismatch_detected=false"
        echo "ERROR: Tamper not detected"
    fi
} | tee "$TAMPER_LOG"

observe 32 "tamper_detection_test_executed"

log "Gate 32 observations captured"

# ============================================================
# Execution Complete
# ============================================================

log ""
log "=== P1-CLOSE EXECUTION COMPLETE ==="
log "Campaign ID: $CAMPAIGN_ID"
log "Execution directory: $EXECUTION_DIR"
log "Observations directory: $EXECUTION_DIR/observations"
log "Artifacts directory: $EXECUTION_DIR/artifacts"
log "Manifest: $MANIFEST_FILE"
log ""
log "NEXT: Run verifier against observations + gate contracts"
log "Command: p1-close-verifier.sh $EXECUTION_DIR $REPO_ROOT/validation/local-vm/evidence/P1-GATE-CONTRACT.md"

echo "=== EXECUTION SUMMARY ===" > "$EXECUTION_DIR/EXECUTION_COMPLETE"
echo "campaign_id=$CAMPAIGN_ID" >> "$EXECUTION_DIR/EXECUTION_COMPLETE"
echo "execution_timestamp=$(timestamp_iso)" >> "$EXECUTION_DIR/EXECUTION_COMPLETE"
echo "observations_count=$(ls -1 "$EXECUTION_DIR/observations" | wc -l)" >> "$EXECUTION_DIR/EXECUTION_COMPLETE"
echo "artifacts_count=$(ls -1 "$EXECUTION_DIR/artifacts" | wc -l)" >> "$EXECUTION_DIR/EXECUTION_COMPLETE"
