#!/bin/bash
# P1-LOCAL-VM-A01: Launch Workload with Scheduler Placement
# Requests placement, starts workload on assigned node, generates traffic
# Usage: ./launch-workload.sh <workload-id> <cpu> <memory-mb> <disk-gb> [duration-seconds]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
CLUSTER_JSON="$STATE_DIR/cluster.json"
WORKLOAD_LOG_DIR="$STATE_DIR/workload-logs"

die() { echo "ERROR: $*" >&2; exit 1; }

WORKLOAD_ID="${1:-}"
CPU_REQ="${2:-}"
MEMORY_REQ="${3:-}"
DISK_REQ="${4:-}"
DURATION="${5:-60}"

[ -n "$WORKLOAD_ID" ] || die "Usage: $0 <workload-id> <cpu> <memory-mb> <disk-gb> [duration-seconds]"
[ -n "$CPU_REQ" ] || die "Usage: $0 <workload-id> <cpu> <memory-mb> <disk-gb> [duration-seconds]"
[ -n "$MEMORY_REQ" ] || die "Usage: $0 <workload-id> <cpu> <memory-mb> <disk-gb> [duration-seconds]"
[ -n "$DISK_REQ" ] || die "Usage: $0 <workload-id> <cpu> <memory-mb> <disk-gb> [duration-seconds]"

[ -f "$CLUSTER_JSON" ] || die "Cluster not configured"

mkdir -p "$WORKLOAD_LOG_DIR"

echo "=== P1-LOCAL-VM-A01: Launch Workload ==="
echo "Workload ID: $WORKLOAD_ID"
echo "Requested resources: CPU=$CPU_REQ, Memory=${MEMORY_REQ}M, Disk=${DISK_REQ}G"
echo "Duration: ${DURATION}s"
echo ""

# Request placement from scheduler
echo "Requesting placement from scheduler..."
PLACEMENT_RESULT_FILE=$(mktemp "$WORKLOAD_LOG_DIR/.placement-${WORKLOAD_ID}.XXXXXX")
trap 'rm -f "$PLACEMENT_RESULT_FILE"' EXIT
bash "$SCRIPT_DIR/request-placement.sh" "$WORKLOAD_ID" "$CPU_REQ" "$MEMORY_REQ" "$DISK_REQ" 30 "$PLACEMENT_RESULT_FILE" || die "Placement failed"
[[ $(jq -r '.workload_id' "$PLACEMENT_RESULT_FILE") == "$WORKLOAD_ID" ]] || die "Placement identity mismatch"
allocation_id="$(jq -r '.allocation_id' "$PLACEMENT_RESULT_FILE")"
[ -n "$allocation_id" ] && [ "$allocation_id" != "null" ] || die "Placement result missing allocation_id"

echo "Allocation ID: $allocation_id"
echo ""

# Get target node from scheduler's placement result
target_node=$(jq -r '.target_node' "$PLACEMENT_RESULT_FILE")
[ -n "$target_node" ] || die "Could not determine target node from scheduler"

# Find SSH port for this node
ssh_port=$(jq -r ".node_details[] | select(.name == \"$target_node\") | .ssh_port" "$CLUSTER_JSON")
[ -n "$ssh_port" ] || die "Could not find SSH port for $target_node"

echo "Target node: $target_node (port $ssh_port)"
echo ""

# Store workload metadata
WORKLOAD_STATE_FILE="$STATE_DIR/workload-$WORKLOAD_ID.json"
cat > "$WORKLOAD_STATE_FILE" << EOF
{
  "workload_id": "$WORKLOAD_ID",
  "allocation_id": "$allocation_id",
  "target_node": "$target_node",
  "ssh_port": $ssh_port,
  "cpu_requested": $CPU_REQ,
  "memory_requested_mb": $MEMORY_REQ,
  "disk_requested_gb": $DISK_REQ,
  "started_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "duration_seconds": $DURATION,
  "status": "RUNNING"
}
EOF

WORKLOAD_LOG="$WORKLOAD_LOG_DIR/${WORKLOAD_ID}.log"
TRAFFIC_LOG="$WORKLOAD_LOG_DIR/${WORKLOAD_ID}-traffic.log"
RESOURCE_LOG="$WORKLOAD_LOG_DIR/${WORKLOAD_ID}-resources.log"

echo "Logs:"
echo "  State: $WORKLOAD_STATE_FILE"
echo "  Server: $WORKLOAD_LOG"
echo "  Traffic: $TRAFFIC_LOG"
echo "  Resources: $RESOURCE_LOG"
echo ""

# SSH key configuration
SSH_KEY="${SSH_KEY:-$HOME/.ssh/p1-local-vm}"
SSH_USER="ubuntu"

# Start workload server on the target node
echo "Starting workload server on $target_node..."
REMOTE_LOG_FILE="/tmp/workload-${WORKLOAD_ID}.log"
REMOTE_PORT=18080
case "$WORKLOAD_ID" in
  workload-api-01) REMOTE_PORT=18081 ;;
  workload-api-02) REMOTE_PORT=18082 ;;
  workload-cache-01) REMOTE_PORT=18083 ;;
esac
ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$ssh_port" "$SSH_USER@localhost" \
  bash -s "$WORKLOAD_ID" "$REMOTE_PORT" "$REMOTE_LOG_FILE" << 'REMOTE_SCRIPT' &

#!/bin/bash
set -euo pipefail

WORKLOAD_ID="$1"
PORT="$2"
LOG_FILE="$3"

mkdir -p "$(dirname "$LOG_FILE")"

{
  echo "=== P1-LOCAL-VM-A01 Workload Server ==="
  echo "Started at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "Workload ID: $WORKLOAD_ID"
  echo "Port: $PORT"
  echo "PID: $$"
  echo ""
} | tee "$LOG_FILE"

serve_requests() {
  request_count=0
  startup_seconds=$SECONDS

  # Use Python HTTP server (reliable JSON encoding, proper Content-Length)
  if command -v python3 >/dev/null 2>&1; then
    python3 -c "
import http.server
import socketserver
import json
import sys

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        response = json.dumps({
            'workload_id': '$WORKLOAD_ID',
            'timestamp': '$(date -u +%Y-%m-%dT%H:%M:%SZ)',
            'pid': $$,
            'uptime_seconds': $((SECONDS - startup_seconds)),
            'requests_served': 1,
            'status': 'running'
        })
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(response)))
        self.end_headers()
        self.wfile.write(response.encode())

    def log_message(self, format, *args):
        pass  # Suppress request logging

with socketserver.TCPServer(('127.0.0.1', $PORT), Handler) as server:
    while True:
        server.handle_request()
" 2>/dev/null
  else
    # Fallback: try netcat (limited but better than invalid JSON from socat)
    while true; do
      response="{\"workload_id\":\"$WORKLOAD_ID\",\"status\":\"running\"}"
      (echo 'HTTP/1.1 200 OK'; echo 'Content-Type: application/json'; echo "Content-Length: ${#response}"; echo 'Connection: close'; echo ''; echo "$response") | nc -l 127.0.0.1 $PORT 2>/dev/null || sleep 0.1
    done
  fi
}

trap 'echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) Shutting down" | tee -a "$LOG_FILE"; exit 0' SIGTERM

serve_requests
REMOTE_SCRIPT

SERVER_PID=$!
sleep 1

echo "Server launcher PID $SERVER_PID; remote port $REMOTE_PORT"
echo ""

# Generate continuous traffic
echo "Generating traffic to $target_node:$REMOTE_PORT..."
TRAFFIC_JSONL="$WORKLOAD_LOG_DIR/${WORKLOAD_ID}-traffic.jsonl"
: > "$TRAFFIC_JSONL"
{
  deadline=$((SECONDS + DURATION))
  request_count=0
  failed_count=0
  request_number=0

  while (( SECONDS < deadline )); do
    request_number=$((request_number + 1))
    ts=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    started_ns=$(python3 -c 'import time; print(time.monotonic_ns())')
    stderr_file="$WORKLOAD_LOG_DIR/${WORKLOAD_ID}-request-${request_number}.stderr"
    ssh_exit=0
    response="$(ssh -i "$SSH_KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$ssh_port" "$SSH_USER@localhost" \
      "python3 -c 'import json,urllib.request; r=urllib.request.urlopen(\"http://127.0.0.1:$REMOTE_PORT/\",timeout=2); print(json.dumps({\"status\":r.status,\"body\":json.loads(r.read().decode())}))'" 2>"$stderr_file")" || ssh_exit=$?
    ended_ns=$(python3 -c 'import time; print(time.monotonic_ns())')
    latency_ms=$(( (ended_ns - started_ns) / 1000000 ))
    status=null
    identity=""
    error=""
    if (( ssh_exit != 0 )); then
      error="SSH_OR_HTTP_EXIT_${ssh_exit}"
    elif ! printf '%s\n' "$response" | jq -e 'type == "object" and (.status | type == "number") and (.body | type == "object")' >/dev/null 2>&1; then
      error="INVALID_HTTP_JSON"
    else
      status=$(printf '%s\n' "$response" | jq -r '.status')
      identity=$(printf '%s\n' "$response" | jq -r '.body.workload_id // empty')
      if [[ "$status" == 200 && "$identity" == "$WORKLOAD_ID" ]]; then
        request_count=$((request_count + 1))
      else
        error="HTTP_STATUS_OR_IDENTITY_MISMATCH"
      fi
    fi
    if [[ -n "$error" ]]; then
      failed_count=$((failed_count + 1))
      echo "$ts Request $request_number FAILED: $error" >> "$TRAFFIC_LOG"
    else
      echo "$ts Request $request_number OK: $response" >> "$TRAFFIC_LOG"
    fi
    jq -cn --arg timestamp "$ts" --arg request_id "$WORKLOAD_ID-$request_number" \
      --arg workload_id "$WORKLOAD_ID" --arg node_id "$target_node" \
      --argjson http_status "$status" --argjson latency_ms "$latency_ms" \
      --arg response_identity "$identity" --arg error "$error" --arg raw_response "$response" \
      '{timestamp:$timestamp,request_id:$request_id,workload_id:$workload_id,node_id:$node_id,
        http_status:$http_status,latency_ms:$latency_ms,response_identity:$response_identity,
        error:$error,raw_response:$raw_response}' >> "$TRAFFIC_JSONL"

    sleep 1
  done

  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) Traffic complete: attempts=$request_number success=$request_count failed=$failed_count" | tee -a "$TRAFFIC_LOG"
  (( request_count > 0 && failed_count == 0 )) || exit 42
} &
TRAFFIC_PID=$!

# Monitor resource utilization
echo "Monitoring resource utilization..."
{
  deadline=$((SECONDS + DURATION))
  sample_count=0
  sample_failures=0

  while (( SECONDS < deadline )); do
    # Get system resource usage on remote node
    if ssh -i "$SSH_KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$ssh_port" "$SSH_USER@localhost" >> "$RESOURCE_LOG" 2>&1 <<'REMOTE_MONITOR'
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) CPU: $(grep -c ^processor /proc/cpuinfo) cores, Usage: $(top -bn1 | grep Cpu | awk '{print $2}')";
echo "Memory: $(free -m | awk 'NR==2 {printf "Used: %dM / %dM (%.1f%%)", $3, $2, $3/$2*100}')";
echo "Disk: $(df -h / | awk 'NR==2 {printf "Used: %s / %s (%s)", $3, $2, $5}')";
REMOTE_MONITOR
    then
      sample_count=$((sample_count + 1))
    else
      sample_failures=$((sample_failures + 1))
      echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) RESOURCE_SAMPLE_FAILED" >> "$RESOURCE_LOG"
    fi
    sleep 5
  done

  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) Resource monitoring complete: success=$sample_count failed=$sample_failures" | tee -a "$RESOURCE_LOG"
  (( sample_count > 0 && sample_failures == 0 )) || exit 43
} &
MONITOR_PID=$!

# Wait for workload to complete
if ! wait $TRAFFIC_PID; then
  wait "$MONITOR_PID" || true
  bash "$SCRIPT_DIR/deallocate-resource.sh" "$allocation_id" >/dev/null 2>&1 || true
  jq --arg completed_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" '.status = "FAILED" | .failure = "TRAFFIC_INTEGRITY_FAILURE" | .completed_at = $completed_at' "$WORKLOAD_STATE_FILE" > "$WORKLOAD_STATE_FILE.tmp"
  mv "$WORKLOAD_STATE_FILE.tmp" "$WORKLOAD_STATE_FILE"
  die "Traffic verification failed for $WORKLOAD_ID"
fi
sleep 1

# Terminate server
ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$ssh_port" "$SSH_USER@localhost" "pkill -f 'workload-$WORKLOAD_ID' || true" >/dev/null 2>&1 || true

# Wait for monitor; an incomplete observation cannot become a clean baseline.
if ! wait "$MONITOR_PID"; then
  jq --arg completed_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" '.status = "FAILED" | .failure = "RESOURCE_OBSERVATION_FAILURE" | .completed_at = $completed_at' "$WORKLOAD_STATE_FILE" > "$WORKLOAD_STATE_FILE.tmp"
  mv "$WORKLOAD_STATE_FILE.tmp" "$WORKLOAD_STATE_FILE"
  die "Resource observations incomplete for $WORKLOAD_ID"
fi

# Update workload state
jq --arg completed_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" '.status = "COMPLETED" | .completed_at = $completed_at' "$WORKLOAD_STATE_FILE" > "$WORKLOAD_STATE_FILE.tmp"
mv "$WORKLOAD_STATE_FILE.tmp" "$WORKLOAD_STATE_FILE"

echo ""
echo "=== Workload Complete ==="
echo "Workload: $WORKLOAD_ID"
echo "Allocation ID: $allocation_id"
echo "Duration: ${DURATION}s"
echo ""
echo "Next: deallocate with: bash deallocate-resource.sh $allocation_id"
