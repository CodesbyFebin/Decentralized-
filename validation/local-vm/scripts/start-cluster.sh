#!/bin/bash
# P1-LOCAL-VM-A01: Start Local VM Cluster
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
CLUSTER_JSON="$STATE_DIR/cluster.json"

die(){ echo "ERROR: $*" >&2; exit 1; }
set_status(){ local s="$1"; local t; t="$(mktemp)"; jq --arg s "$s" '.status=$s' "$CLUSTER_JSON" > "$t" && mv "$t" "$CLUSTER_JSON"; }
serial_tail(){ local f="$1"; [ -f "$f" ] && { echo "--- serial tail: $f ---"; tail -100 "$f" || true; }; }
stop_started(){
  local p pid
  for p in "$@"; do
    [ -f "$p" ] || continue
    pid="$(cat "$p" 2>/dev/null || true)"
    case "$pid" in ''|*[!0-9]*) ;; *) kill "$pid" 2>/dev/null || true; sleep 1; kill -9 "$pid" 2>/dev/null || true ;; esac
    rm -f "$p"
  done
}
trap 'rc=$?; if [ $rc -ne 0 ] && [ -f "$CLUSTER_JSON" ]; then set_status START_FAILED || true; fi' EXIT

[ -f "$CLUSTER_JSON" ] || die "Cluster not configured. Run create-vm-cluster.sh first."
command -v jq >/dev/null || die "jq not found"

HYPERVISOR="$(jq -r '.hypervisor' "$CLUSTER_JSON")"
NODES="$(jq -r '.nodes' "$CLUSTER_JSON")"
QEMU_BINARY="$(jq -r '.qemu_binary' "$CLUSTER_JSON")"
CLUSTER_SHA="$(jq -r '.source_sha' "$CLUSTER_JSON")"
CLUSTER_STATE_DIR="$(jq -r '.state_directory' "$CLUSTER_JSON")"
CURRENT_SHA="$(git -C "$REPO_ROOT" rev-parse HEAD)"
HOST_OS="$(uname -s)"

echo "=== P1-LOCAL-VM-A01: Start Cluster ==="
echo "Hypervisor: $HYPERVISOR"
echo "Nodes: $NODES"
echo "QEMU Binary: $QEMU_BINARY"

[ "$HYPERVISOR" = "qemu" ] || die "Backend mismatch: cluster=$HYPERVISOR expected=qemu"
[ "$CLUSTER_SHA" = "$CURRENT_SHA" ] || die "Source SHA mismatch: cluster=$CLUSTER_SHA current=$CURRENT_SHA. Recreate cluster from current source."
[ "$CLUSTER_STATE_DIR" = "$STATE_DIR" ] || die "State root mismatch: cluster=$CLUSTER_STATE_DIR canonical=$STATE_DIR"
command -v "$QEMU_BINARY" >/dev/null || die "$QEMU_BINARY not found"

if [ "$HOST_OS" = "Darwin" ]; then MACHINE_TYPE="pc"; ACCEL="hvf"; else MACHINE_TYPE="pc"; ACCEL="kvm"; fi

# Validate all inputs before mutating runtime state.
for ((i=1;i<=NODES;i++)); do
  idx=$((i-1))
  disk="$(jq -r ".node_details[$idx].disk" "$CLUSTER_JSON")"
  cidir="$(jq -r ".node_details[$idx].cloud_init_dir" "$CLUSTER_JSON")"
  port="$(jq -r ".node_details[$idx].ssh_port" "$CLUSTER_JSON")"
  [ -f "$disk" ] || die "Missing disk for node $i: $disk"
  [ -f "$cidir/user-data" ] || die "Missing user-data for node $i"
  [ -f "$cidir/meta-data" ] || die "Missing meta-data for node $i"
  if lsof -nP -iTCP:"$port" -sTCP:LISTEN 2>/dev/null | grep -q LISTEN; then die "SSH port $port already in use"; fi
done

set_status STARTING
started_pids=()

for ((i=1;i<=NODES;i++)); do
  idx=$((i-1))
  NODE_NAME="$(jq -r ".node_details[$idx].name" "$CLUSTER_JSON")"
  DISK="$(jq -r ".node_details[$idx].disk" "$CLUSTER_JSON")"
  CLOUD_INIT_DIR="$(jq -r ".node_details[$idx].cloud_init_dir" "$CLUSTER_JSON")"
  SSH_PORT="$(jq -r ".node_details[$idx].ssh_port" "$CLUSTER_JSON")"
  PID_FILE="$(jq -r ".node_details[$idx].qemu_pid_file" "$CLUSTER_JSON")"
  SERIAL_LOG="$(jq -r ".node_details[$idx].serial_log" "$CLUSTER_JSON")"
  NODE_DIR="$(dirname "$DISK")"
  SEED_ISO="$NODE_DIR/seed.iso"

  if [ -f "$PID_FILE" ]; then
    oldpid="$(cat "$PID_FILE" 2>/dev/null || true)"
    if [[ "$oldpid" =~ ^[0-9]+$ ]] && kill -0 "$oldpid" 2>/dev/null; then die "$NODE_NAME already running (PID $oldpid)"; fi
    rm -f "$PID_FILE"
  fi

  if [ ! -f "$SEED_ISO" ]; then
    echo "Creating cloud-init seed for $NODE_NAME..."
    if command -v cloud-localds >/dev/null 2>&1; then
      cloud-localds "$SEED_ISO" "$CLOUD_INIT_DIR/user-data" "$CLOUD_INIT_DIR/meta-data"
    elif command -v mkisofs >/dev/null 2>&1; then
      mkisofs -quiet -output "$SEED_ISO" -volid cidata -joliet -rock "$CLOUD_INIT_DIR/user-data" "$CLOUD_INIT_DIR/meta-data"
    elif [ "$HOST_OS" = "Darwin" ] && command -v hdiutil >/dev/null 2>&1; then
      seedtmp="$(mktemp -d)"
      cp "$CLOUD_INIT_DIR/user-data" "$seedtmp/user-data"
      cp "$CLOUD_INIT_DIR/meta-data" "$seedtmp/meta-data"
      hdiutil makehybrid -quiet -iso -joliet -default-volume-name cidata -o "$SEED_ISO" "$seedtmp"
      rm -rf "$seedtmp"
    else
      die "Cannot create cloud-init seed: install cloud-localds or mkisofs"
    fi
  fi
  [ -s "$SEED_ISO" ] || die "Cloud-init seed missing/empty for $NODE_NAME"

  : > "$SERIAL_LOG"
  echo "Starting $NODE_NAME (SSH port $SSH_PORT)..."
  "$QEMU_BINARY"     -name "$NODE_NAME"     -machine "type=$MACHINE_TYPE,accel=$ACCEL"     -cpu host     -smp "$(jq -r '.cpu_per_node' "$CLUSTER_JSON")"     -m "$(jq -r '.memory_per_node_mb' "$CLUSTER_JSON")"     -drive "file=$DISK,format=qcow2,cache=writeback"     -drive "file=$SEED_ISO,format=raw,media=cdrom,readonly=on"     -net "user,hostfwd=tcp::${SSH_PORT}-:22"     -net "nic,model=virtio"     -display none     -daemonize     -pidfile "$PID_FILE"     -serial "file:$SERIAL_LOG" || {
      serial_tail "$SERIAL_LOG"
      stop_started "${started_pids[@]}"
      die "$NODE_NAME QEMU launch failed"
    }

  sleep 1
  [ -s "$PID_FILE" ] || { serial_tail "$SERIAL_LOG"; stop_started "${started_pids[@]}"; die "$NODE_NAME did not create PID file"; }
  pid="$(cat "$PID_FILE")"
  [[ "$pid" =~ ^[0-9]+$ ]] || { stop_started "${started_pids[@]}"; die "$NODE_NAME PID is invalid: $pid"; }
  kill -0 "$pid" 2>/dev/null || { serial_tail "$SERIAL_LOG"; rm -f "$PID_FILE"; stop_started "${started_pids[@]}"; die "$NODE_NAME process exited after launch"; }
  started_pids+=("$PID_FILE")
  echo "  ✓ PROCESS_RUNNING pid=$pid"
done

set_status RUNNING
trap - EXIT
echo "=== VMs RUNNING ==="
echo "Cluster process state verified. Run:"
echo "  bash $SCRIPT_DIR/bootstrap-nodes.sh"
