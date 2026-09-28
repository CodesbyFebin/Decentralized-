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
allocation_id=$(bash "$SCRIPT_DIR/request-placement.sh" "$WORKLOAD_ID" "$CPU_REQ" "$MEMORY_REQ" "$DISK_REQ" 30) || die "Placement failed"

echo "Allocation ID: $allocation_id"
echo ""

# Get target node from scheduler's placement result
PLACEMENT_RESULT_FILE="$STATE_DIR/placement-result.json"
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
SSH_OPTS="-i $SSH_KEY -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5"

# Start workload server on the target node
echo "Starting workload server on $target_node..."
REMOTE_LOG_FILE="/tmp/workload-${WORKLOAD_ID}.log"
ssh $SSH_OPTS -p "$ssh_port" "$SSH_USER@localhost" \
  bash -s "$WORKLOAD_ID" 8080 "$REMOTE_LOG_FILE" << 'REMOTE_SCRIPT' &

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

  # Start simple HTTP server using socat if available, else Python
  if command -v socat >/dev/null 2>&1; then
    # Use socat for listening and request handling
    socat TCP-LISTEN:$PORT,reuseaddr,fork SYSTEM:"bash -c 'read -t 5 line; request_count=$((request_count+1)); echo \"$(date -u +%Y-%m-%dT%H:%M:%SZ) \$line\" >> \"$LOG_FILE\"; echo \"HTTP/1.1 200 OK\"; echo \"Content-Type: application/json\"; echo \"Content-Length: 250\"; echo \"Connection: close\"; echo \"\"; echo \"{\\\"workload_id\\\":\\\"$WORKLOAD_ID\\\",\\\"timestamp\\\":\\\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\\\",\\\"pid\\\":$$,\\\"uptime_seconds\\\":$((SECONDS - startup_seconds)),\\\"requests_served\\\":$request_count,\\\"status\\\":\\\"running\\\"}\"'" 2>/dev/null
  elif command -v python3 >/dev/null 2>&1; then
    # Use Python as fallback
    python3 -c "
import http.server
import json
import time
import sys

class Handler(http.server.BaseRequestHandler):
    def handle(self):
        try:
            self.request.recv(1024)
            response = json.dumps({
                'workload_id': '$WORKLOAD_ID',
                'timestamp': '$(date -u +%Y-%m-%dT%H:%M:%SZ)',
                'pid': $$,
                'uptime_seconds': $((SECONDS - startup_seconds)),
                'requests_served': 1,
                'status': 'running'
            })
            self.request.sendall(b'HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: ' + str(len(response)).encode() + b'\r\nConnection: close\r\n\r\n' + response.encode())
        except:
            pass

server = http.server.TCPServer(('127.0.0.1', $PORT), Handler)
while True:
    server.handle_request()
" 2>/dev/null
  else
    # Fallback: try netcat with simple echo (limited but better than nothing)
    while true; do
      (echo 'HTTP/1.1 200 OK'; echo 'Content-Type: application/json'; echo ''; echo "{\"workload_id\":\"$WORKLOAD_ID\",\"status\":\"running\"}") | nc -l 127.0.0.1 $PORT 2>/dev/null || sleep 0.1
    done
  fi
}

trap 'echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) Shutting down" | tee -a "$LOG_FILE"; exit 0' SIGTERM

serve_requests
REMOTE_SCRIPT

SERVER_PID=$!
sleep 1

echo "Server started (PID $SERVER_PID on remote)"
echo ""

# Generate continuous traffic
echo "Generating traffic to $target_node:8080..."
TRAFFIC_TEMP_LOG="$TRAFFIC_LOG.tmp"
{
  deadline=$((SECONDS + DURATION))

  while (( SECONDS < deadline )); do
    # Send HTTP request to workload server
    {
      echo "GET / HTTP/1.1"
      echo "Host: localhost:8080"
      echo "Connection: close"
      echo ""
    } | ssh $SSH_OPTS -p "$ssh_port" "$SSH_USER@localhost" "nc -q 1 127.0.0.1 8080" 2>/dev/null | grep -E "^{" >> "$TRAFFIC_TEMP_LOG" || true

    sleep 1
  done

  # Count responses and format for final log
  if [ -f "$TRAFFIC_TEMP_LOG" ]; then
    request_count=$(wc -l < "$TRAFFIC_TEMP_LOG")
    while IFS= read -r response; do
      echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) Request $((request_count++)): $response" >> "$TRAFFIC_LOG"
    done < "$TRAFFIC_TEMP_LOG"
    rm -f "$TRAFFIC_TEMP_LOG"
  else
    request_count=0
  fi

  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) Traffic generation complete: $request_count requests" | tee -a "$TRAFFIC_LOG"
} &
TRAFFIC_PID=$!

# Monitor resource utilization
echo "Monitoring resource utilization..."
{
  deadline=$((SECONDS + DURATION))
  sample_count=0

  while (( SECONDS < deadline )); do
    # Get system resource usage on remote node
    ssh $SSH_OPTS -p "$ssh_port" "$SSH_USER@localhost" <<'REMOTE_MONITOR' 2>/dev/null
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) CPU: $(grep -c ^processor /proc/cpuinfo) cores, Usage: $(top -bn1 | grep Cpu | awk '{print $2}')";
echo "Memory: $(free -m | awk 'NR==2 {printf "Used: %dM / %dM (%.1f%%)", $3, $2, $3/$2*100}')";
echo "Disk: $(df -h / | awk 'NR==2 {printf "Used: %s / %s (%s)", $3, $2, $5}')";
REMOTE_MONITOR

    sample_count=$((sample_count + 1))
    sleep 5
  done

  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) Resource monitoring complete: $sample_count samples" | tee -a "$RESOURCE_LOG"
} &
MONITOR_PID=$!

# Wait for workload to complete
wait $TRAFFIC_PID || true
sleep 1

# Terminate server
ssh $SSH_OPTS -p "$ssh_port" "$SSH_USER@localhost" "pkill -f 'nc -l 127.0.0.1 8080'" || true

# Wait for monitor
wait $MONITOR_PID || true

# Update workload state
jq '.status = "COMPLETED" | .completed_at = "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"' "$WORKLOAD_STATE_FILE" > "$WORKLOAD_STATE_FILE.tmp"
mv "$WORKLOAD_STATE_FILE.tmp" "$WORKLOAD_STATE_FILE"

echo ""
echo "=== Workload Complete ==="
echo "Workload: $WORKLOAD_ID"
echo "Allocation ID: $allocation_id"
echo "Duration: ${DURATION}s"
echo ""
echo "Next: deallocate with: bash deallocate-resource.sh $allocation_id"
