#!/bin/bash

# P1-CLOSE-A01 Verifier
# Role: Evaluate observations against gate contracts
# Input: Executor output (observations + artifacts)
# Output: Gate outcomes (PASS, FAIL, BLOCKED, UNKNOWN) with evidence binding

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <execution_directory> [gate_contract_path]"
    echo ""
    echo "execution_directory: Path to executor output (contains observations/ and artifacts/)"
    echo "gate_contract_path: Path to P1-GATE-CONTRACT.md (optional, defaults to repo)"
    exit 1
fi

EXECUTION_DIR="$1"
GATE_CONTRACT_PATH="${2:-/home/user/Decentralized-/validation/local-vm/evidence/P1-GATE-CONTRACT.md}"
REPO_ROOT="/home/user/Decentralized-"
VERDICT_DIR="$EXECUTION_DIR/verdicts"

mkdir -p "$VERDICT_DIR"

log() {
    local msg="$*"
    local ts=$(date -u +'%Y-%m-%dT%H:%M:%S.%3NZ')
    echo "[$ts] $msg" | tee -a "$VERDICT_DIR/verification.log"
}

verdict() {
    local gate_id="$1"
    local outcome="$2"
    local reason="$3"

    {
        echo "gate_id=\"$gate_id\""
        echo "outcome=\"$outcome\""
        echo "reason=\"$reason\""
        echo "verified_at=\"$(date -u +'%Y-%m-%dT%H:%M:%S.%3NZ')\""
    } > "$VERDICT_DIR/gate-${gate_id}-verdict.txt"

    log "Gate $gate_id: $outcome ($reason)"
}

# ============================================================
# GATE 10: ResourceLedger Presence & Validity
# ============================================================

log "=== Verifying Gate 10: ResourceLedger Presence ==="

GATE_10_OBSERVATIONS="$EXECUTION_DIR/observations/gate-10-summary.txt"
GATE_10_LEDGER="$EXECUTION_DIR/artifacts/gate-10-ledger-pre-injection.json"

if [ ! -f "$GATE_10_OBSERVATIONS" ]; then
    verdict 10 UNKNOWN "Observations missing from executor"
else
    source "$GATE_10_OBSERVATIONS"

    # Evaluate pass condition:
    # file_exists AND ledger_parse_exit == 0 AND ledger_type == "object"

    if [ "$file_exists" = "true" ] && [ "$ledger_parse_exit" = "0" ] && [ "$ledger_type" = "object" ]; then
        verdict 10 PASS "ResourceLedger exists, valid JSON, type=object"
    elif [ "$file_exists" = "false" ]; then
        verdict 10 FAIL "ResourceLedger file not found"
    elif [ "$ledger_parse_exit" != "0" ]; then
        verdict 10 FAIL "ResourceLedger JSON parse failed (exit $ledger_parse_exit)"
    else
        verdict 10 FAIL "ResourceLedger type is not object (got: $ledger_type)"
    fi
fi

# ============================================================
# GATE 11-16: ResourceLedger Model A Validation
# ============================================================

log "=== Verifying Gate 11-16: ResourceLedger Model A ==="

GATE_11_16_VALIDATION="$EXECUTION_DIR/artifacts/gate-11-16-model-a-validation.txt"

if [ ! -f "$GATE_11_16_VALIDATION" ]; then
    verdict 11-16 UNKNOWN "Model A validation file missing"
else
    # Parse validation results
    CONSTRAINT_VIOLATIONS=$(grep "ERROR: allocated" "$GATE_11_16_VALIDATION" 2>/dev/null | wc -l || echo 0)

    if grep -q "validation_possible=false" "$GATE_11_16_VALIDATION" 2>/dev/null; then
        verdict 11-16 BLOCKED "Ledger not available (depends on Gate 10)"
    elif [ "$CONSTRAINT_VIOLATIONS" -eq 0 ]; then
        verdict 11-16 PASS "Model A formula validated; no capacity violations"
    else
        verdict 11-16 FAIL "Model A violations detected: $CONSTRAINT_VIOLATIONS"
    fi
fi

# ============================================================
# GATE 17-18: Network Partition Injection & Detection
# ============================================================

log "=== Verifying Gate 17-18: Network Partition ==="

GATE_17_18_LOG="$EXECUTION_DIR/artifacts/gate-17-18-partition-injection.log"
GATE_17_18_TIMESTAMP="$EXECUTION_DIR/artifacts/gate-17-18-partition-inject-timestamp.txt"
GATE_17_18_DETECTION_LAT="$EXECUTION_DIR/artifacts/gate-17-18-detection-latency-ns.txt"
GATE_17_18_CONFIRMED="$EXECUTION_DIR/artifacts/gate-17-18-partition-confirmed.txt"

if [ ! -f "$GATE_17_18_LOG" ]; then
    verdict 17-18 UNKNOWN "Partition injection log missing"
else
    # Check if tc is available
    if grep -q "tc_available=false" "$GATE_17_18_LOG" 2>/dev/null; then
        verdict 17-18 BLOCKED "tc command not found in environment"
    elif grep -q "partition_injection_failed=true" "$GATE_17_18_LOG" 2>/dev/null; then
        verdict 17-18 FAIL "Partition injection failed (tc exit code non-zero)"
    elif grep -q "partition_confirmed_active=true" "$GATE_17_18_LOG" 2>/dev/null; then
        # Read detection latency
        if [ -f "$GATE_17_18_DETECTION_LAT" ]; then
            DETECTION_LAT_NS=$(cat "$GATE_17_18_DETECTION_LAT")
            DETECTION_LAT_MS=$((DETECTION_LAT_NS / 1000000))

            if [ "$DETECTION_LAT_MS" -le 45000 ]; then
                verdict 17-18 PASS "Partition confirmed active; detection latency $DETECTION_LAT_MS ms"
            else
                verdict 17-18 FAIL "Detection latency $DETECTION_LAT_MS ms exceeds 45s threshold"
            fi
        else
            verdict 17-18 FAIL "Partition active but detection latency not measured"
        fi
    else
        verdict 17-18 FAIL "Partition injection did not create isolation"
    fi
fi

# ============================================================
# GATE 19-20: Workload Migration Detection (depends on Gate 17-18)
# ============================================================

log "=== Verifying Gate 19-20: Workload Migration ==="

GATE_17_18_VERDICT="$VERDICT_DIR/gate-17-18-verdict.txt"

if [ ! -f "$GATE_17_18_VERDICT" ]; then
    verdict 19-20 UNKNOWN "Gate 17-18 verdict not found"
else
    source "$GATE_17_18_VERDICT"

    case "$outcome" in
        BLOCKED|FAIL|UNKNOWN)
            verdict 19-20 BLOCKED "Cannot test without real partition (Gate 17-18: $outcome)"
            ;;
        PASS)
            # Partition is real, check for migration
            # Note: Full implementation would compare ledger snapshots
            verdict 19-20 UNKNOWN "Requires ledger snapshot analysis (not implemented in this version)"
            ;;
    esac
fi

# ============================================================
# GATE 23: Workload Crash Injection
# ============================================================

log "=== Verifying Gate 23: Workload Crash Injection ==="

GATE_23_LOG="$EXECUTION_DIR/artifacts/gate-23-crash-injection.log"

if [ ! -f "$GATE_23_LOG" ]; then
    verdict 23 UNKNOWN "Crash injection log missing"
else
    # Evaluate pass condition:
    # workload_process_found AND workload_responding_before_crash AND process_confirmed_gone

    PROCESS_FOUND=$(grep "workload_process_found=" "$GATE_23_LOG" | tail -1 | cut -d= -f2 || echo "unknown")
    RESPONDING=$(grep "workload_responding_before_crash=" "$GATE_23_LOG" | tail -1 | cut -d= -f2 || echo "unknown")
    GONE=$(grep "process_confirmed_gone=" "$GATE_23_LOG" | tail -1 | cut -d= -f2 || echo "unknown")

    if [ "$PROCESS_FOUND" = "false" ]; then
        verdict 23 FAIL "No workload process identified"
    elif [ "$RESPONDING" = "false" ]; then
        verdict 23 FAIL "Workload not responding before crash"
    elif [ "$GONE" = "false" ]; then
        verdict 23 FAIL "Process did not disappear after SIGKILL"
    elif [ "$PROCESS_FOUND" = "true" ] && [ "$RESPONDING" = "true" ] && [ "$GONE" = "true" ]; then
        verdict 23 PASS "Workload crashed (SIGKILL injected and confirmed)"
    else
        verdict 23 UNKNOWN "Inconclusive crash injection state"
    fi
fi

# ============================================================
# GATE 24: Workload Recovery Measurement (depends on Gate 23)
# ============================================================

log "=== Verifying Gate 24: Workload Recovery ==="

GATE_23_VERDICT="$VERDICT_DIR/gate-23-verdict.txt"
GATE_24_LOG="$EXECUTION_DIR/artifacts/gate-24-recovery-measurement.log"
GATE_24_RECOVERY_NS="$EXECUTION_DIR/artifacts/gate-24-recovery-time-ns.txt"

if [ ! -f "$GATE_23_VERDICT" ]; then
    verdict 24 UNKNOWN "Gate 23 verdict not found"
else
    source "$GATE_23_VERDICT"

    if [ "$outcome" != "PASS" ]; then
        verdict 24 BLOCKED "Cannot measure recovery without successful crash (Gate 23: $outcome)"
    else
        # Crash was successful, check recovery
        if [ ! -f "$GATE_24_LOG" ]; then
            verdict 24 UNKNOWN "Recovery measurement log missing"
        else
            RECOVERY=$(grep "recovery_complete=" "$GATE_24_LOG" | tail -1 | cut -d= -f2 || echo "unknown")
            RESPONDING=$(grep "workload_responding_after_recovery=" "$GATE_24_LOG" | tail -1 | cut -d= -f2 || echo "unknown")

            if [ "$RECOVERY" = "false" ]; then
                verdict 24 FAIL "Workload did not restart within 60s"
            elif [ "$RESPONDING" = "false" ]; then
                verdict 24 FAIL "Workload recovered but not responding to HTTP"
            elif [ "$RECOVERY" = "true" ] && [ "$RESPONDING" = "true" ]; then
                if [ -f "$GATE_24_RECOVERY_NS" ]; then
                    RECOVERY_NS=$(cat "$GATE_24_RECOVERY_NS")
                    RECOVERY_MS=$((RECOVERY_NS / 1000000))

                    if [ "$RECOVERY_MS" -le 30000 ]; then
                        verdict 24 PASS "Workload recovered in $RECOVERY_MS ms (≤30s threshold)"
                    else
                        verdict 24 FAIL "Recovery time $RECOVERY_MS ms exceeds 30s threshold"
                    fi
                else
                    verdict 24 UNKNOWN "Recovery detected but time not measured"
                fi
            else
                verdict 24 UNKNOWN "Inconclusive recovery state"
            fi
        fi
    fi
fi

# ============================================================
# GATE 25: Scheduler Process State
# ============================================================

log "=== Verifying Gate 25: Scheduler Process State ==="

GATE_25_LOG="$EXECUTION_DIR/artifacts/gate-25-scheduler-state.log"

if [ ! -f "$GATE_25_LOG" ]; then
    verdict 25 UNKNOWN "Scheduler state log missing"
else
    FOUND=$(grep "scheduler_process_found=" "$GATE_25_LOG" | tail -1 | cut -d= -f2 || echo "unknown")
    STATE=$(grep "process_state_from_proc=" "$GATE_25_LOG" | tail -1 | cut -d= -f2 || echo "unknown")

    if [ "$FOUND" = "false" ]; then
        verdict 25 FAIL "Scheduler process not found"
    elif [ "$STATE" = "Z" ] || [ "$STATE" = "D" ]; then
        verdict 25 FAIL "Scheduler in zombie/uninterruptible state (state=$STATE)"
    elif [ "$FOUND" = "true" ] && ([ "$STATE" = "R" ] || [ "$STATE" = "S" ]); then
        verdict 25 PASS "Scheduler process operational (state=$STATE)"
    else
        verdict 25 UNKNOWN "Scheduler state inconclusive (state=$STATE)"
    fi
fi

# ============================================================
# GATE 26: Scheduler Responsiveness (depends on Gate 24)
# ============================================================

log "=== Verifying Gate 26: Scheduler Responsiveness ==="

GATE_24_VERDICT="$VERDICT_DIR/gate-24-verdict.txt"
GATE_26_LOG="$EXECUTION_DIR/artifacts/gate-25-scheduler-state.log"

if [ ! -f "$GATE_24_VERDICT" ]; then
    verdict 26 UNKNOWN "Gate 24 verdict not found"
else
    source "$GATE_24_VERDICT"

    if [ "$outcome" != "PASS" ]; then
        verdict 26 BLOCKED "Cannot test responsiveness without successful recovery (Gate 24: $outcome)"
    else
        # Recovery successful, scheduler should be responsive
        # Note: Full implementation would query scheduler API
        verdict 26 UNKNOWN "Requires scheduler API query (not implemented in this version)"
    fi
fi

# ============================================================
# GATE 27-28: Control-Plane Consistency
# ============================================================

log "=== Verifying Gate 27-28: Control-Plane Consistency ==="

# Note: Full implementation would check ledger state post-recovery
verdict 27-28 UNKNOWN "Requires ledger state comparison post-recovery (not implemented in this version)"

# ============================================================
# GATE 29: Evidence Artifact Collection
# ============================================================

log "=== Verifying Gate 29: Evidence Artifact Collection ==="

MANIFEST_FILE="$EXECUTION_DIR/artifacts/evidence-manifest.json"

if [ ! -f "$MANIFEST_FILE" ]; then
    verdict 29 FAIL "Evidence manifest not created"
else
    # Verify manifest is valid JSON
    if jq . "$MANIFEST_FILE" > /dev/null 2>&1; then
        ARTIFACT_COUNT=$(jq '.artifacts | length' "$MANIFEST_FILE" 2>/dev/null || echo 0)
        verdict 29 PASS "Evidence manifest created and valid ($ARTIFACT_COUNT artifacts)"
    else
        verdict 29 FAIL "Evidence manifest is not valid JSON"
    fi
fi

# ============================================================
# GATE 30: Cryptographic Signature Verification
# ============================================================

log "=== Verifying Gate 30: Cryptographic Signature ==="

SIG_LOG="$EXECUTION_DIR/artifacts/gate-30-signature-generation.log"
SIGNATURE_FILE="$EXECUTION_DIR/artifacts/evidence-manifest.sig"
PUBKEY_FILE="$EXECUTION_DIR/artifacts/evidence-public-key.pem"

if [ ! -f "$SIG_LOG" ]; then
    verdict 30 UNKNOWN "Signature generation log missing"
else
    # Check if signature was generated
    if grep -q "signature_verified=true" "$SIG_LOG" 2>/dev/null; then
        verdict 30 PASS "Evidence manifest cryptographically signed and verified"
    elif grep -q "signature_verified=false" "$SIG_LOG" 2>/dev/null; then
        verdict 30 FAIL "Signature verification failed"
    elif grep -q "openssl_available=false" "$SIG_LOG" 2>/dev/null; then
        verdict 30 BLOCKED "OpenSSL not available for cryptographic operations"
    elif grep -q "keypair_generation_failed=true" "$SIG_LOG" 2>/dev/null; then
        verdict 30 FAIL "Keypair generation failed"
    else
        verdict 30 UNKNOWN "Signature generation state inconclusive"
    fi
fi

# ============================================================
# GATE 31: ResourceLedger Consistency Verification
# ============================================================

log "=== Verifying Gate 31: ResourceLedger Consistency ==="

CONSISTENCY_LOG="$EXECUTION_DIR/artifacts/gate-31-consistency-verification.log"

if [ ! -f "$CONSISTENCY_LOG" ]; then
    verdict 31 UNKNOWN "Consistency verification log missing"
else
    # Check if ledger is accessible
    if grep -q "ledger_accessible=false" "$CONSISTENCY_LOG" 2>/dev/null; then
        verdict 31 FAIL "ResourceLedger not accessible"
    else
        # Check for constraint violations
        VIOLATIONS=$(grep "constraint_pass=false" "$CONSISTENCY_LOG" 2>/dev/null | wc -l || echo 0)

        if [ "$VIOLATIONS" -eq 0 ]; then
            verdict 31 PASS "ResourceLedger Model A formula verified; no violations"
        else
            verdict 31 FAIL "Model A formula violations: $VIOLATIONS nodes"
        fi
    fi
fi

# ============================================================
# GATE 32: Tamper Detection Negative Control
# ============================================================

log "=== Verifying Gate 32: Tamper Detection ==="

TAMPER_LOG="$EXECUTION_DIR/artifacts/gate-32-tamper-test.log"

if [ ! -f "$TAMPER_LOG" ]; then
    verdict 32 UNKNOWN "Tamper test log missing"
else
    # Check if hash mismatch was detected
    if grep -q "hash_mismatch_detected=true" "$TAMPER_LOG" 2>/dev/null; then
        verdict 32 PASS "Tamper detection negative control: tampered evidence detected via hash mismatch"
    elif grep -q "hash_mismatch_detected=false" "$TAMPER_LOG" 2>/dev/null; then
        verdict 32 FAIL "Tamper detection failed: tampered evidence not detected"
    else
        verdict 32 UNKNOWN "Tamper test state inconclusive"
    fi
fi

# ============================================================
# Verification Summary
# ============================================================

log ""
log "=== VERIFICATION COMPLETE ==="
log "Verdict directory: $VERDICT_DIR"

# Summarize outcomes
{
    echo "=== P1-CLOSE Verification Summary ==="
    echo "Execution: $EXECUTION_DIR"
    echo "Verified: $(date -u +'%Y-%m-%dT%H:%M:%S.%3NZ')"
    echo ""

    PASS_COUNT=0
    FAIL_COUNT=0
    BLOCKED_COUNT=0
    UNKNOWN_COUNT=0

    for verdict_file in "$VERDICT_DIR"/gate-*-verdict.txt; do
        if [ -f "$verdict_file" ]; then
            source "$verdict_file"
            echo "$gate_id: $outcome"

            case "$outcome" in
                PASS) PASS_COUNT=$((PASS_COUNT + 1)) ;;
                FAIL) FAIL_COUNT=$((FAIL_COUNT + 1)) ;;
                BLOCKED) BLOCKED_COUNT=$((BLOCKED_COUNT + 1)) ;;
                UNKNOWN) UNKNOWN_COUNT=$((UNKNOWN_COUNT + 1)) ;;
            esac
        fi
    done

    echo ""
    echo "Summary:"
    echo "  PASS=$PASS_COUNT"
    echo "  FAIL=$FAIL_COUNT"
    echo "  BLOCKED=$BLOCKED_COUNT"
    echo "  UNKNOWN=$UNKNOWN_COUNT"
} | tee "$VERDICT_DIR/verification-summary.txt"

log "Verification complete"
