#!/bin/bash

# P1-LOCAL-VM-A01 Executor (Gates 1-10)
# Initial Qualification: Preflight → Topology → Bootstrap → Mesh → Baseline → State Consistency
#
# Design: Executor/Verifier Separation
# - Executor: perform operation → capture observation → bind artifact
# - Verifier: read contract + artifact → evaluate predicate → outcome

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="${REPO_ROOT:-/home/user/Decentralized-}"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
EVIDENCE_DIR="$REPO_ROOT/validation/local-vm/evidence"
SCRIPTS_DIR="$REPO_ROOT/validation/local-vm/scripts"
SSH_KEY="$HOME/.ssh/p1-local-vm"

# Use provided EXECUTION_DIR or create a new one
if [ -n "${1:-}" ]; then
    EXECUTION_DIR="$1"
    CAMPAIGN_ID=$(basename "$EXECUTION_DIR" | sed 's/P1-LOCAL-VM-A01-EXECUTION-//')
else
    CAMPAIGN_ID=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
    EXECUTION_DIR="$EVIDENCE_DIR/P1-LOCAL-VM-A01-EXECUTION-$CAMPAIGN_ID"
fi

mkdir -p "$EXECUTION_DIR/artifacts" "$EXECUTION_DIR/observations"

# Logging and observation capture
log() {
    local msg="$*"
    local ts=$(date -u +'%Y-%m-%dT%H:%M:%S.%3NZ')
    echo "[$ts] $msg" | tee -a "$EXECUTION_DIR/execution.log"
}

observe() {
    local gate_id="$1"
    local observation="$2"
    local ts=$(date -u +'%s%N')

    echo "$observation" >> "$EXECUTION_DIR/observations/gate-${gate_id}.txt"
    log "Gate $gate_id observation: $observation (ts: $ts)"
}

capture_artifact() {
    local gate_id="$1"
    local artifact_name="$2"
    local source_path="$3"

    if [ -f "$source_path" ]; then
        cp "$source_path" "$EXECUTION_DIR/artifacts/gate-${gate_id}-${artifact_name}"
        log "Captured artifact: gate-${gate_id}-${artifact_name}"
    else
        log "WARNING: Could not capture artifact $artifact_name from $source_path"
    fi
}

capture_command_output() {
    local gate_id="$1"
    local artifact_name="$2"
    shift 2

    local output_file="$EXECUTION_DIR/artifacts/gate-${gate_id}-${artifact_name}"

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
# GATE 1: Preflight Verification
# ============================================================

log "=== GATE 1: Preflight Verification ==="

GATE_1_OBSERVATIONS=""

# Observation 1: Prerequisites check
observe 1 "preflight_start=$(timestamp_iso)"

# Check commands
COMMANDS_REQUIRED=("bash" "jq" "git" "curl" "ssh-keygen" "lsof")
COMMANDS_MISSING=""
for cmd in "${COMMANDS_REQUIRED[@]}"; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        COMMANDS_MISSING="$COMMANDS_MISSING $cmd"
    fi
done

if [ -z "$COMMANDS_MISSING" ]; then
    observe 1 "required_commands=PRESENT"
    GATE_1_OBSERVATIONS="${GATE_1_OBSERVATIONS}commands_ok=true "
else
    observe 1 "required_commands=MISSING:$COMMANDS_MISSING"
    GATE_1_OBSERVATIONS="${GATE_1_OBSERVATIONS}commands_ok=false "
fi

# Check repository state
CURRENT_SHA=$(cd "$REPO_ROOT" && git rev-parse HEAD 2>/dev/null || echo "error")
observe 1 "repo_source_sha=$CURRENT_SHA"
GATE_1_OBSERVATIONS="${GATE_1_OBSERVATIONS}source_sha=$CURRENT_SHA "

# Check QEMU availability
if command -v qemu-system-x86_64 >/dev/null 2>&1 || command -v qemu-system-aarch64 >/dev/null 2>&1; then
    observe 1 "qemu_status=AVAILABLE"
    GATE_1_OBSERVATIONS="${GATE_1_OBSERVATIONS}qemu_ok=true "
else
    observe 1 "qemu_status=NOT_FOUND"
    GATE_1_OBSERVATIONS="${GATE_1_OBSERVATIONS}qemu_ok=false "
fi

# Check state directory
if [ -d "$STATE_DIR" ]; then
    observe 1 "state_dir_exists=true"
    GATE_1_OBSERVATIONS="${GATE_1_OBSERVATIONS}state_dir_ok=true "
else
    observe 1 "state_dir_exists=false"
    GATE_1_OBSERVATIONS="${GATE_1_OBSERVATIONS}state_dir_ok=false "
fi

echo "$GATE_1_OBSERVATIONS" > "$EXECUTION_DIR/observations/gate-1-summary.txt"
log "Gate 1: Preflight verification complete"

# ============================================================
# GATE 2: Topology Discovery
# ============================================================

log "=== GATE 2: Topology Discovery ==="

TOPOLOGY_FILE="$STATE_DIR/topology.sh"
GATE_2_OBSERVATIONS=""

observe 2 "topology_discovery_start=$(timestamp_iso)"

if [ -f "$TOPOLOGY_FILE" ]; then
    observe 2 "topology_file_found=true"
    GATE_2_OBSERVATIONS="${GATE_2_OBSERVATIONS}topology_present=true "

    # Source topology and extract values
    source "$TOPOLOGY_FILE" 2>/dev/null || true

    observe 2 "preflight_status=${PREFLIGHT_STATUS:-UNKNOWN}"
    observe 2 "node_count=${NODES:-UNKNOWN}"
    observe 2 "cpu_per_node=${CPU_PER_NODE:-UNKNOWN}"
    observe 2 "memory_per_node=${MEMORY_PER_NODE:-UNKNOWN}"
    observe 2 "disk_per_node=${DISK_PER_NODE:-UNKNOWN}"

    GATE_2_OBSERVATIONS="${GATE_2_OBSERVATIONS}preflight=${PREFLIGHT_STATUS:-UNKNOWN} nodes=${NODES:-UNKNOWN} cpu=${CPU_PER_NODE:-UNKNOWN} "

    # Capture topology file
    capture_artifact 2 "topology.sh" "$TOPOLOGY_FILE"
else
    observe 2 "topology_file_found=false"
    GATE_2_OBSERVATIONS="${GATE_2_OBSERVATIONS}topology_present=false "
fi

echo "$GATE_2_OBSERVATIONS" > "$EXECUTION_DIR/observations/gate-2-summary.txt"
log "Gate 2: Topology discovery complete"

# ============================================================
# GATE 3-4: Bootstrap SSH Setup
# ============================================================

log "=== GATE 3-4: Bootstrap SSH Setup ==="

GATE_34_OBSERVATIONS=""

observe 3 "bootstrap_start=$(timestamp_iso)"

# Check SSH key existence and state
if [ -f "$SSH_KEY" ]; then
    observe 3 "ssh_key_exists=true"
    SSH_KEY_SIZE=$(wc -c < "$SSH_KEY")
    observe 3 "ssh_key_size=$SSH_KEY_SIZE"
    GATE_34_OBSERVATIONS="${GATE_34_OBSERVATIONS}ssh_key_ok=true size=$SSH_KEY_SIZE "

    # Capture SSH key fingerprint (public part only)
    if [ -f "${SSH_KEY}.pub" ]; then
        SSH_FINGERPRINT=$(ssh-keygen -l -f "${SSH_KEY}.pub" 2>/dev/null || echo "error")
        observe 3 "ssh_fingerprint=$SSH_FINGERPRINT"
        GATE_34_OBSERVATIONS="${GATE_34_OBSERVATIONS}ssh_fingerprint_ok=true "
    fi
else
    observe 3 "ssh_key_exists=false"
    GATE_34_OBSERVATIONS="${GATE_34_OBSERVATIONS}ssh_key_ok=false "
fi

# Check cluster state
if [ -f "$STATE_DIR/cluster.json" ]; then
    observe 4 "cluster_json_exists=true"
    CLUSTER_STATUS=$(jq -r '.status // "unknown"' "$STATE_DIR/cluster.json" 2>/dev/null || echo "error")
    observe 4 "cluster_status=$CLUSTER_STATUS"

    CLUSTER_NODES=$(jq -r '.node_details | length' "$STATE_DIR/cluster.json" 2>/dev/null || echo "0")
    observe 4 "cluster_nodes=$CLUSTER_NODES"

    GATE_34_OBSERVATIONS="${GATE_34_OBSERVATIONS}cluster_status=$CLUSTER_STATUS nodes=$CLUSTER_NODES "

    # Capture cluster state
    capture_artifact 4 "cluster.json" "$STATE_DIR/cluster.json"
else
    observe 4 "cluster_json_exists=false"
    GATE_34_OBSERVATIONS="${GATE_34_OBSERVATIONS}cluster_status=NOT_CREATED "
fi

echo "$GATE_34_OBSERVATIONS" > "$EXECUTION_DIR/observations/gate-34-summary.txt"
log "Gate 3-4: Bootstrap SSH setup complete"

# ============================================================
# GATE 5: Mesh Network Verification
# ============================================================

log "=== GATE 5: Mesh Network Verification ==="

GATE_5_OBSERVATIONS=""

observe 5 "mesh_verification_start=$(timestamp_iso)"

if [ -f "$STATE_DIR/cluster.json" ]; then
    RUNNING_NODES=0
    HEALTHY_NODES=0
    UNREACHABLE_NODES=0

    # Parse cluster JSON and test connectivity
    NODES_ARRAY=$(jq -r '.node_details[]? | .name' "$STATE_DIR/cluster.json" 2>/dev/null || true)

    while IFS= read -r node_name; do
        if [ -z "$node_name" ]; then continue; fi

        SSH_PORT=$(jq -r ".node_details[] | select(.name==\"$node_name\") | .ssh_port" "$STATE_DIR/cluster.json" 2>/dev/null || echo "0")

        if [ "$SSH_PORT" != "0" ]; then
            RUNNING_NODES=$((RUNNING_NODES + 1))

            # Test SSH connectivity (non-blocking)
            if timeout 5 ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o ConnectTimeout=3 "root@localhost" -p "$SSH_PORT" "echo ok" >/dev/null 2>&1; then
                HEALTHY_NODES=$((HEALTHY_NODES + 1))
                observe 5 "node_ssh_status=$node_name:REACHABLE"
            else
                UNREACHABLE_NODES=$((UNREACHABLE_NODES + 1))
                observe 5 "node_ssh_status=$node_name:UNREACHABLE"
            fi
        fi
    done <<< "$NODES_ARRAY"

    observe 5 "mesh_nodes_running=$RUNNING_NODES"
    observe 5 "mesh_nodes_healthy=$HEALTHY_NODES"
    observe 5 "mesh_nodes_unreachable=$UNREACHABLE_NODES"
    GATE_5_OBSERVATIONS="running=$RUNNING_NODES healthy=$HEALTHY_NODES unreachable=$UNREACHABLE_NODES "
else
    observe 5 "mesh_verification_skipped=no_cluster_state"
    GATE_5_OBSERVATIONS="skipped=true "
fi

echo "$GATE_5_OBSERVATIONS" > "$EXECUTION_DIR/observations/gate-5-summary.txt"
log "Gate 5: Mesh network verification complete"

# ============================================================
# GATE 6-9: Workload Baseline & Metrics
# ============================================================

log "=== GATE 6-9: Workload Baseline & Metrics ==="

GATE_69_OBSERVATIONS=""

observe 6 "baseline_workload_start=$(timestamp_iso)"

# Collect system metrics from nodes
if [ -f "$STATE_DIR/cluster.json" ]; then
    TOTAL_CPU=0
    TOTAL_MEMORY=0
    TOTAL_STORAGE=0

    # Extract resource baseline
    NODES_ARRAY=$(jq -r '.node_details[]? | .name' "$STATE_DIR/cluster.json" 2>/dev/null || true)
    while IFS= read -r node_name; do
        if [ -z "$node_name" ]; then continue; fi

        SSH_PORT=$(jq -r ".node_details[] | select(.name==\"$node_name\") | .ssh_port" "$STATE_DIR/cluster.json" 2>/dev/null || echo "0")

        if [ "$SSH_PORT" != "0" ]; then
            # Attempt to collect metrics (non-blocking)
            {
                CPU_COUNT=$(timeout 5 ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o ConnectTimeout=3 "root@localhost" -p "$SSH_PORT" "nproc 2>/dev/null || echo 0" 2>/dev/null || echo "0")
                MEMORY_KB=$(timeout 5 ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o ConnectTimeout=3 "root@localhost" -p "$SSH_PORT" "grep MemTotal /proc/meminfo 2>/dev/null | awk '{print \$2}' || echo 0" 2>/dev/null || echo "0")

                TOTAL_CPU=$((TOTAL_CPU + ${CPU_COUNT:-0}))
                TOTAL_MEMORY=$((TOTAL_MEMORY + ${MEMORY_KB:-0}))

                observe 7 "baseline_${node_name}_cpu=$CPU_COUNT"
                observe 8 "baseline_${node_name}_memory=$MEMORY_KB"
            } &
        fi
    done <<< "$NODES_ARRAY"

    # Wait for background tasks
    wait

    observe 9 "baseline_total_cpu=$TOTAL_CPU"
    observe 9 "baseline_total_memory=$TOTAL_MEMORY"
    GATE_69_OBSERVATIONS="total_cpu=$TOTAL_CPU total_memory=$TOTAL_MEMORY "
else
    observe 6 "baseline_skipped=no_cluster_state"
    GATE_69_OBSERVATIONS="skipped=true "
fi

echo "$GATE_69_OBSERVATIONS" > "$EXECUTION_DIR/observations/gate-69-summary.txt"
log "Gate 6-9: Workload baseline & metrics complete"

# ============================================================
# GATE 10: Initial State Consistency
# ============================================================

log "=== GATE 10: Initial State Consistency ==="

GATE_10_OBSERVATIONS=""

observe 10 "state_consistency_check=$(timestamp_iso)"

# Check cluster.json validity
if [ -f "$STATE_DIR/cluster.json" ]; then
    CLUSTER_PARSE_EXIT=0
    jq . "$STATE_DIR/cluster.json" > "$EXECUTION_DIR/artifacts/gate-10-cluster-parse.json" 2> "$EXECUTION_DIR/artifacts/gate-10-cluster-parse-error.txt" || CLUSTER_PARSE_EXIT=$?

    observe 10 "cluster_json_parse_exit=$CLUSTER_PARSE_EXIT"
    GATE_10_OBSERVATIONS="${GATE_10_OBSERVATIONS}cluster_json_ok=$([[ $CLUSTER_PARSE_EXIT -eq 0 ]] && echo 'true' || echo 'false') "

    if [ $CLUSTER_PARSE_EXIT -eq 0 ]; then
        CLUSTER_TYPE=$(jq -r 'type' "$STATE_DIR/cluster.json" 2>/dev/null || echo "unknown")
        observe 10 "cluster_json_type=$CLUSTER_TYPE"
        GATE_10_OBSERVATIONS="${GATE_10_OBSERVATIONS}cluster_json_type=$CLUSTER_TYPE "
    fi

    # Capture cluster state for evidence
    capture_artifact 10 "cluster-final.json" "$STATE_DIR/cluster.json"
fi

# Check topology.sh validity
if [ -f "$TOPOLOGY_FILE" ]; then
    TOPOLOGY_PARSE_EXIT=0
    {
        source "$TOPOLOGY_FILE"
        [ "${PREFLIGHT_STATUS:-}" = "PASS" ] && [ -n "${NODES:-}" ] && [ -n "${CPU_PER_NODE:-}" ]
    } 2>/dev/null || TOPOLOGY_PARSE_EXIT=$?

    observe 10 "topology_valid_exit=$TOPOLOGY_PARSE_EXIT"
    GATE_10_OBSERVATIONS="${GATE_10_OBSERVATIONS}topology_ok=$([[ $TOPOLOGY_PARSE_EXIT -eq 0 ]] && echo 'true' || echo 'false') "

    # Capture topology for evidence
    capture_artifact 10 "topology-final.sh" "$TOPOLOGY_FILE"
fi

# Verify execution directory integrity
if [ -d "$EXECUTION_DIR/observations" ]; then
    OBS_COUNT=$(find "$EXECUTION_DIR/observations" -type f | wc -l)
    observe 10 "execution_observations_count=$OBS_COUNT"
    GATE_10_OBSERVATIONS="${GATE_10_OBSERVATIONS}observations_captured=$OBS_COUNT "
fi

if [ -d "$EXECUTION_DIR/artifacts" ]; then
    ART_COUNT=$(find "$EXECUTION_DIR/artifacts" -type f | wc -l)
    observe 10 "execution_artifacts_count=$ART_COUNT"
    GATE_10_OBSERVATIONS="${GATE_10_OBSERVATIONS}artifacts_captured=$ART_COUNT "
fi

# Write final summary
echo "$GATE_10_OBSERVATIONS" > "$EXECUTION_DIR/observations/gate-10-summary.txt"

log "Gate 10: Initial state consistency check complete"

# ============================================================
# Execution Summary
# ============================================================

log ""
log "=== P1-LOCAL-VM-A01 EXECUTION COMPLETE ==="
log "Campaign ID: $CAMPAIGN_ID"
log "Execution Directory: $EXECUTION_DIR"
log "Observations captured: $(find "$EXECUTION_DIR/observations" -type f | wc -l)"
log "Artifacts captured: $(find "$EXECUTION_DIR/artifacts" -type f | wc -l)"
log "Timestamp: $(timestamp_iso)"
log ""

# Create execution summary
{
    echo "=== P1-LOCAL-VM-A01 Execution Summary ==="
    echo "Campaign ID: $CAMPAIGN_ID"
    echo "Start Time: $(grep -oP '\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}' "$EXECUTION_DIR/execution.log" | head -1)"
    echo "End Time: $(timestamp_iso)"
    echo "Repository SHA: $CURRENT_SHA"
    echo "Execution Dir: $EXECUTION_DIR"
    echo ""
    echo "Gates Executed: 1-10 (Preflight, Topology, Bootstrap, Mesh, Baseline, State Consistency)"
    echo "Total Observations: $(find "$EXECUTION_DIR/observations" -type f | wc -l)"
    echo "Total Artifacts: $(find "$EXECUTION_DIR/artifacts" -type f | wc -l)"
    echo ""
    echo "Next Step: Run verifier script to evaluate gate contracts against observations"
    echo "Command: bash validation/local-vm/scripts/verify-p1-local-vm-gates.sh $EXECUTION_DIR"
} | tee "$EXECUTION_DIR/EXECUTION-SUMMARY.txt"

log "Execution summary written to $EXECUTION_DIR/EXECUTION-SUMMARY.txt"
echo "P1-LOCAL-VM-A01 GATES 1-10 EXECUTION COMPLETE: $EXECUTION_DIR"
