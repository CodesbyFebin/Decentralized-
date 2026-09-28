#!/bin/bash
# P1-LOCAL-VM-A01: Lightweight HTTP Workload Server
# Runs on assigned node and generates measurable traffic
# Usage: ./workload-server.sh <workload-id> <port> <log-file>

set -euo pipefail

WORKLOAD_ID="${1:-workload-test}"
PORT="${2:-8080}"
LOG_FILE="${3:-/tmp/workload.log}"

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"

{
  echo "=== P1-LOCAL-VM-A01 Workload Server ==="
  echo "Started at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "Workload ID: $WORKLOAD_ID"
  echo "Port: $PORT"
  echo "PID: $$"
  echo ""
} | tee "$LOG_FILE"

# Simple HTTP server using nc (netcat) in a loop
# Handles multiple concurrent requests and generates measurable traffic
serve_requests() {
  while true; do
    # Accept incoming connection and serve response
    {
      read -t 5 method path protocol || true

      if [ -n "$method" ]; then
        # Log request
        echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $method $path" >> "$LOG_FILE"

        # Serve response
        echo "HTTP/1.1 200 OK"
        echo "Content-Type: application/json"
        echo "Content-Length: 200"
        echo "Connection: close"
        echo ""

        # Generate response body with workload metadata
        cat <<EOF
{
  "workload_id": "$WORKLOAD_ID",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "pid": $$,
  "uptime_seconds": $((SECONDS - startup_seconds)),
  "requests_served": $((requests_served + 1))
}
EOF
      fi
    } | nc -q 1 -l 127.0.0.1 "$PORT" 2>/dev/null || true

    requests_served=$((requests_served + 1))
  done
}

startup_seconds=$SECONDS
requests_served=0

# Trap SIGTERM for graceful shutdown
trap 'echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) Shutting down, served $requests_served requests" | tee -a "$LOG_FILE"; exit 0' SIGTERM

echo "Listening on 127.0.0.1:$PORT" | tee -a "$LOG_FILE"

# Start serving requests
serve_requests
