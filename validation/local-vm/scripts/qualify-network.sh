#!/bin/bash
# P1-LOCAL-VM-A01: Baseline Network / Node Qualification
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"
CLUSTER_JSON="$STATE_DIR/cluster.json"
RESULTS_DIR="$REPO_ROOT/validation/local-vm/evidence"
RESULT="$RESULTS_DIR/network-qualification.json"
SSH_USER="ubuntu"

die(){ echo "ERROR: $*" >&2; exit 1; }
[ -f "$CLUSTER_JSON" ] || die "Cluster not configured"
command -v jq >/dev/null || die "jq not found"

QUAL="$(jq -r '.qualification' "$CLUSTER_JSON")"
STATUS="$(jq -r '.status' "$CLUSTER_JSON")"
NODES="$(jq -r '.nodes' "$CLUSTER_JSON")"
CLUSTER_SHA="$(jq -r '.source_sha' "$CLUSTER_JSON")"
SSH_KEY="$(jq -r '.ssh_key_path' "$CLUSTER_JSON")"
VERIFIER_SHA="$(git -C "$REPO_ROOT" rev-parse HEAD)"
[ "$QUAL" = "P1-LOCAL-VM-A01" ] || die "Wrong qualification: $QUAL"
[ "$STATUS" = "READY" ] || die "Cluster must be READY; got $STATUS"
[ -f "$SSH_KEY" ] || die "SSH key missing: $SSH_KEY"
mkdir -p "$RESULTS_DIR"
tmp="$(mktemp)"
checks="$(mktemp)"
printf '[]' > "$checks"
trap 'rm -f "$tmp" "$checks"' EXIT

add_check(){
  local node="$1" check="$2" result="$3" detail="$4" t
  t="$(mktemp)"
  jq --arg node "$node" --arg check "$check" --arg result "$result" --arg detail "$detail"     '. + [{node:$node,check:$check,result:$result,detail:$detail}]' "$checks" > "$t"
  mv "$t" "$checks"
}

ssh_run(){
  local port="$1"; shift
  ssh -i "$SSH_KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null     -o ConnectTimeout=5 -p "$port" "$SSH_USER@localhost" "$@"
}

echo "=== P1-LOCAL-VM-A01: Baseline Network Qualification ==="
machine_ids=()
blocking=0

for ((i=1;i<=NODES;i++)); do
  idx=$((i-1))
  name="$(jq -r ".node_details[$idx].name" "$CLUSTER_JSON")"
  port="$(jq -r ".node_details[$idx].ssh_port" "$CLUSTER_JSON")"
  pidfile="$(jq -r ".node_details[$idx].qemu_pid_file" "$CLUSTER_JSON")"

  if [ ! -s "$pidfile" ]; then
    add_check "$name" process_liveness FAIL "PID file missing"; blocking=$((blocking+1)); continue
  fi
  pid="$(cat "$pidfile" 2>/dev/null || true)"
  if ! [[ "$pid" =~ ^[0-9]+$ ]] || ! kill -0 "$pid" 2>/dev/null; then
    add_check "$name" process_liveness FAIL "QEMU process not live"; blocking=$((blocking+1)); continue
  fi
  add_check "$name" process_liveness PASS "pid=$pid"

  if ! ssh_run "$port" true >/dev/null 2>&1; then
    add_check "$name" ssh_connectivity FAIL "localhost:$port unreachable/authentication failed"; blocking=$((blocking+1)); continue
  fi
  add_check "$name" ssh_connectivity PASS "localhost:$port"

  hostname="$(ssh_run "$port" hostname 2>/dev/null || true)"
  mid="$(ssh_run "$port" cat /etc/machine-id 2>/dev/null || true)"
  arch="$(ssh_run "$port" uname -m 2>/dev/null || true)"
  uptime="$(ssh_run "$port" uptime -p 2>/dev/null || true)"
  disk="$(ssh_run "$port" df -Pk / 2>/dev/null | sed -n '2p' | tr -s ' ' | cut -d' ' -f4 || true)"
  ips="$(ssh_run "$port" hostname -I 2>/dev/null || true)"

  for spec in "hostname:$hostname" "machine_id:$mid" "architecture:$arch" "uptime:$uptime" "disk_available_kb:$disk"; do
    key="${spec%%:*}"; value="${spec#*:}"
    if [ -n "$value" ]; then add_check "$name" "$key" PASS "$value"; else add_check "$name" "$key" UNKNOWN "empty observation"; blocking=$((blocking+1)); fi
  done
  if [ -n "$ips" ]; then add_check "$name" guest_ip_observation PASS "$ips"; else add_check "$name" guest_ip_observation UNKNOWN "no guest IP reported"; blocking=$((blocking+1)); fi
  [ -n "$mid" ] && machine_ids+=("$mid")
done

if [ "${#machine_ids[@]}" -eq "$NODES" ]; then
  unique="$(printf '%s\n' "${machine_ids[@]}" | sort -u | wc -l | tr -d ' ')"
  if [ "$unique" -eq "$NODES" ]; then add_check cluster unique_machine_ids PASS "$unique/$NODES"; else add_check cluster unique_machine_ids FAIL "$unique/$NODES"; blocking=$((blocking+1)); fi
else
  add_check cluster unique_machine_ids UNKNOWN "${#machine_ids[@]}/$NODES observed"; blocking=$((blocking+1))
fi

timestamp="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
overall=PASS
[ "$blocking" -eq 0 ] || overall=FAIL
jq -n   --arg qualification "P1-LOCAL-VM-A01"   --arg timestamp_utc "$timestamp"   --arg cluster_source_sha "$CLUSTER_SHA"   --arg verifier_source_sha "$VERIFIER_SHA"   --arg cluster_status "$STATUS"   --arg overall "$overall"   --argjson nodes "$NODES"   --slurpfile checks "$checks"   '{qualification:$qualification,timestamp_utc:$timestamp_utc,cluster_source_sha:$cluster_source_sha,verifier_source_sha:$verifier_source_sha,cluster_status:$cluster_status,nodes:$nodes,scope:"baseline host-forwarded SSH and guest observations; does not prove VM-to-VM partition semantics",overall:$overall,checks:$checks[0]}' > "$tmp"
jq empty "$tmp"
mv "$tmp" "$RESULT"
sha256="$(shasum -a 256 "$RESULT" | awk '{print $1}')"
printf '%s  %s\n' "$sha256" "$(basename "$RESULT")" > "$RESULT.sha256"

echo "Result: $overall"
echo "Evidence: $RESULT"
echo "SHA256: $sha256"
[ "$overall" = PASS ] || exit 1
