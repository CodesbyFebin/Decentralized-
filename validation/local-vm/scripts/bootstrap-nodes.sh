#!/bin/bash
# P1-LOCAL-VM-A01: Bootstrap Nodes
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
CLUSTER_JSON="$STATE_DIR/cluster.json"
SSH_KEY="$HOME/.ssh/p1-local-vm"
SSH_USER="ubuntu"
BOOT_TIMEOUT_SECONDS=120
RETRY_DELAY=2

die(){ echo "ERROR: $*" >&2; exit 1; }
set_status(){ local s="$1"; local t; t="$(mktemp)"; jq --arg s "$s" '.status=$s' "$CLUSTER_JSON" > "$t" && mv "$t" "$CLUSTER_JSON"; }
serial_tail(){ local f="$1"; [ -f "$f" ] && { echo "--- serial tail: $f ---"; tail -100 "$f" || true; }; }

[ -f "$CLUSTER_JSON" ] || die "Cluster not configured"
[ -f "$SSH_KEY" ] || die "SSH key not found: $SSH_KEY"
NODES="$(jq -r '.nodes' "$CLUSTER_JSON")"
STATUS="$(jq -r '.status' "$CLUSTER_JSON")"
[ "$STATUS" = "RUNNING" ] || die "Cluster status must be RUNNING before bootstrap; got $STATUS"

ssh_exec(){
  local port="$1"; shift
  ssh -i "$SSH_KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null     -o ConnectTimeout=3 -p "$port" "$SSH_USER@localhost" "$@" 2>/dev/null
}

echo "=== P1-LOCAL-VM-A01: Bootstrap Nodes ==="
echo "Nodes: $NODES"
echo "Boot timeout: ${BOOT_TIMEOUT_SECONDS}s"

# Process liveness is a prerequisite to SSH readiness.
for ((i=1;i<=NODES;i++)); do
  idx=$((i-1))
  name="$(jq -r ".node_details[$idx].name" "$CLUSTER_JSON")"
  pidfile="$(jq -r ".node_details[$idx].qemu_pid_file" "$CLUSTER_JSON")"
  slog="$(jq -r ".node_details[$idx].serial_log" "$CLUSTER_JSON")"
  if [ ! -s "$pidfile" ]; then set_status BOOTSTRAP_FAILED; serial_tail "$slog"; die "$name PROCESS_NOT_RUNNING: PID file missing"; fi
  pid="$(cat "$pidfile")"
  if ! [[ "$pid" =~ ^[0-9]+$ ]] || ! kill -0 "$pid" 2>/dev/null; then set_status BOOTSTRAP_FAILED; serial_tail "$slog"; die "$name PROCESS_NOT_RUNNING: PID $pid is not live"; fi
  echo "✓ $name PROCESS_RUNNING pid=$pid"
done

set_status BOOTSTRAPPING
deadline=$((SECONDS + BOOT_TIMEOUT_SECONDS))
while (( SECONDS < deadline )); do
  ready=0
  for ((i=1;i<=NODES;i++)); do
    idx=$((i-1))
    name="$(jq -r ".node_details[$idx].name" "$CLUSTER_JSON")"
    port="$(jq -r ".node_details[$idx].ssh_port" "$CLUSTER_JSON")"
    pidfile="$(jq -r ".node_details[$idx].qemu_pid_file" "$CLUSTER_JSON")"
    slog="$(jq -r ".node_details[$idx].serial_log" "$CLUSTER_JSON")"
    pid="$(cat "$pidfile" 2>/dev/null || true)"
    if ! [[ "$pid" =~ ^[0-9]+$ ]] || ! kill -0 "$pid" 2>/dev/null; then
      set_status BOOTSTRAP_FAILED; serial_tail "$slog"; die "$name PROCESS_DIED during bootstrap"
    fi
    if ssh_exec "$port" "echo ready" >/dev/null 2>&1; then
      echo "✓ $name SSH_READY ($port)"
      ready=$((ready+1))
    else
      if lsof -nP -iTCP:"$port" -sTCP:LISTEN 2>/dev/null | grep -q LISTEN; then
        echo "… $name PORT_LISTENING_SSH_NOT_READY ($port)"
      else
        echo "… $name SSH_NOT_READY ($port)"
      fi
    fi
  done
  [ "$ready" -eq "$NODES" ] && break
  sleep "$RETRY_DELAY"
done

ready=0
for ((i=1;i<=NODES;i++)); do
  idx=$((i-1)); port="$(jq -r ".node_details[$idx].ssh_port" "$CLUSTER_JSON")"
  ssh_exec "$port" "echo ready" >/dev/null 2>&1 && ready=$((ready+1))
done
if [ "$ready" -ne "$NODES" ]; then
  set_status BOOTSTRAP_FAILED
  echo "ERROR: SSH readiness timeout; ready=$ready/$NODES"
  for ((i=1;i<=NODES;i++)); do idx=$((i-1)); serial_tail "$(jq -r ".node_details[$idx].serial_log" "$CLUSTER_JSON")"; done
  exit 1
fi

echo "Collecting baseline evidence..."
machine_ids=()
for ((i=1;i<=NODES;i++)); do
  idx=$((i-1))
  name="$(jq -r ".node_details[$idx].name" "$CLUSTER_JSON")"
  port="$(jq -r ".node_details[$idx].ssh_port" "$CLUSTER_JSON")"
  hostname="$(ssh_exec "$port" hostname || true)"
  mid="$(ssh_exec "$port" 'cat /etc/machine-id' || true)"
  kernel="$(ssh_exec "$port" 'uname -r' || true)"
  arch="$(ssh_exec "$port" 'uname -m' || true)"
  cpus="$(ssh_exec "$port" 'grep -c ^processor /proc/cpuinfo' || true)"
  memory="$(ssh_exec "$port" 'free -h | awk '"'"'/^Mem:/ {print $2}'"'"'' || true)"
  uptime="$(ssh_exec "$port" 'uptime -p' || true)"
  disk="$(ssh_exec "$port" 'df -h / | awk '"'"'NR==2 {print $4}'"'"'' || true)"
  for pair in "Hostname:$hostname" "Machine ID:$mid" "Kernel:$kernel" "Architecture:$arch" "CPUs:$cpus" "Memory:$memory" "Uptime:$uptime" "Disk Free:$disk"; do
    value="${pair#*:}"; [ -n "$value" ] || { set_status BOOTSTRAP_FAILED; die "$name required evidence field empty: ${pair%%:*}"; }
  done
  machine_ids+=("$mid")
  echo "$name: hostname=$hostname machine_id=$mid arch=$arch cpus=$cpus memory=$memory uptime=$uptime disk_free=$disk"
done

unique="$(printf '%s\n' "${machine_ids[@]}" | sort -u | wc -l | tr -d ' ')"
[ "$unique" -eq "$NODES" ] || { set_status BOOTSTRAP_FAILED; die "Machine IDs are not unique ($unique/$NODES)"; }

set_status READY
echo "=== Bootstrap Complete ==="
echo "STATUS: READY"
echo "All $NODES nodes are PROCESS_RUNNING + SSH_READY with unique machine IDs."
