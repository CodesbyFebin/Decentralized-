#!/bin/bash
# Observe the guest-to-guest mesh from every guest; host SSH is only the control plane.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
CLUSTER="$REPO_ROOT/validation/local-vm/state/cluster.json"
EVIDENCE="$REPO_ROOT/validation/local-vm/evidence/interguest-network"

[ -f "$CLUSTER" ] || { echo "BLOCKED: cluster.json absent" >&2; exit 2; }
command -v jq >/dev/null || { echo "BLOCKED: jq absent" >&2; exit 2; }
[ "$(jq -r '.status' "$CLUSTER")" = READY ] || { echo "BLOCKED: cluster is not READY" >&2; exit 2; }
[ "$(jq -r '.mesh_backend // empty' "$CLUSTER")" = qemu-socket-multicast ] || { echo "BLOCKED: mesh is not configured" >&2; exit 2; }
SHA="$(git -C "$REPO_ROOT" rev-parse HEAD)"
[ "$(jq -r '.source_sha' "$CLUSTER")" = "$SHA" ] || { echo "BLOCKED: source SHA mismatch" >&2; exit 2; }
KEY="$(jq -r '.ssh_key_path' "$CLUSTER")"
[ -f "$KEY" ] || { echo "BLOCKED: SSH key absent" >&2; exit 2; }
N="$(jq -r '.nodes' "$CLUSTER")"
[ "$N" -ge 2 ] || { echo "BLOCKED: need at least two guests" >&2; exit 2; }
mkdir -p "$EVIDENCE"
OUT="$EVIDENCE/observations.jsonl"
: > "$OUT"
failure=0
for ((i=0;i<N;i++)); do
  name="$(jq -r ".node_details[$i].name" "$CLUSTER")"
  port="$(jq -r ".node_details[$i].ssh_port" "$CLUSTER")"
  own_ip="$(jq -r ".node_details[$i].mesh_ip" "$CLUSTER")"
  for ((j=0;j<N;j++)); do
    [ "$i" -eq "$j" ] && continue
    peer="$(jq -r ".node_details[$j].name" "$CLUSTER")"
    ip="$(jq -r ".node_details[$j].mesh_ip" "$CLUSTER")"
    raw="$EVIDENCE/${name}-to-${peer}.txt"
    if ssh -i "$KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5 -p "$port" ubuntu@127.0.0.1 \
      "ip -4 addr show dev mesh0; ip -4 route get $ip; ping -I mesh0 -c 3 -W 2 $ip" > "$raw" 2>&1 \
      && grep -Fq "$own_ip/24" "$raw"; then status=PASS; else status=FAIL; failure=1; fi
    jq -nc --arg from "$name" --arg to "$peer" --arg target "$ip" --arg status "$status" --arg raw "$(basename "$raw")" --arg sha "$SHA" \
      '{from:$from,to:$to,target:$target,status:$status,raw:$raw,source_sha:$sha}' >> "$OUT"
  done
done
if [ "$failure" -eq 0 ]; then echo "PASS: directed mesh reachability observed ($((N*(N-1))) paths)"; else echo "FAIL: at least one mesh path failed" >&2; fi
echo "Evidence: $OUT"
exit "$failure"
