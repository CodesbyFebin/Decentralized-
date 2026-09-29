#!/bin/bash
# Read-only Kubernetes observations. Never equate pod readiness with dh qualification.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
NAMESPACE="${DH_K8S_NAMESPACE:-decentralized-host}"
OUT="${DH_K8S_EVIDENCE_DIR:-$ROOT/deploy/kubernetes/evidence}"
mkdir -p "$OUT"
timestamp="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
sha="$(git -C "$ROOT" rev-parse HEAD)"
checks="$(mktemp)"
trap 'rm -f "$checks"' EXIT
printf '[]' > "$checks"
add() {
  local name="$1" status="$2" detail="$3" next
  next="$(mktemp)"
  jq --arg name "$name" --arg status "$status" --arg detail "$detail" '. + [{name:$name,status:$status,detail:$detail}]' "$checks" > "$next"
  mv "$next" "$checks"
}
if ! command -v jq >/dev/null; then
  echo 'BLOCKED: jq is required to record observations' >&2
  exit 2
fi
if ! command -v kubectl >/dev/null; then
  add kubernetes_api BLOCKED 'kubectl unavailable'
elif ! kubectl --request-timeout=10s cluster-info > "$OUT/cluster-info.txt" 2>&1; then
  add kubernetes_api BLOCKED 'API unavailable; see cluster-info.txt'
else
  add kubernetes_api PASS 'API responded; see cluster-info.txt'
  if kubectl --request-timeout=10s -n "$NAMESPACE" get statefulset dh-control -o json > "$OUT/statefulset.json" 2> "$OUT/statefulset-error.txt"; then
    replicas="$(jq -r '.spec.replicas // "UNKNOWN"' "$OUT/statefulset.json")"
    ready="$(jq -r '.status.readyReplicas // "UNKNOWN"' "$OUT/statefulset.json")"
    if [ "$replicas" = 3 ] && [ "$ready" = 3 ]; then
      add statefulset_ready PASS '3/3 Kubernetes replicas ready; does not prove Raft quorum'
    elif [ "$replicas" = UNKNOWN ] || [ "$ready" = UNKNOWN ]; then
      add statefulset_ready UNKNOWN "desired=$replicas ready=$ready"
    else
      add statefulset_ready FAIL "desired=$replicas ready=$ready"
    fi
    image="$(jq -r '.spec.template.spec.containers[] | select(.name=="control") | .image' "$OUT/statefulset.json")"
    if [[ "$image" =~ @sha256:[0-9a-f]{64}$ ]]; then
      add pinned_image PASS "$image"
    else
      add pinned_image FAIL 'Control image is not pinned by SHA256 digest'
    fi
  else
    add statefulset_ready BLOCKED 'Cannot observe StatefulSet; see statefulset-error.txt'
    add pinned_image UNKNOWN 'StatefulSet unavailable'
  fi
  if kubectl --request-timeout=10s -n "$NAMESPACE" get pods -l app=dh-control -o json > "$OUT/pods.json" 2> "$OUT/pods-error.txt"; then
    count="$(jq '[.items[] | select(.status.phase == "Running")] | length' "$OUT/pods.json")"
    [ "$count" = 3 ] && add pods_running PASS '3/3 pod phases Running' || add pods_running FAIL "running=$count/3"
  else
    add pods_running BLOCKED 'Cannot observe pods; see pods-error.txt'
  fi
fi
add raft_quorum UNKNOWN 'Requires authenticated dh cp status with three voters and one leader'
add host_admission UNKNOWN 'Requires signed per-host identity and policy observations'
add traffic_recovery UNKNOWN 'Requires continuous traffic, actual fault/heal, and measured restoration'
add signed_evidence UNKNOWN 'Requires manifest, Ed25519 verification, and tamper negative control bound to source SHA'
jq -n --arg ts "$timestamp" --arg sha "$sha" --arg namespace "$NAMESPACE" --slurpfile checks "$checks" \
  '{qualification:"K8S-CONTROL-A01",timestamp_utc:$ts,source_sha:$sha,namespace:$namespace,overall:"BLOCKED",checks:$checks[0],scope:"Kubernetes infrastructure observations only; no P1 VM qualification"}' > "$OUT/readiness.json"
echo 'BLOCKED: Kubernetes readiness observations cannot establish product qualification.'
echo "Evidence: $OUT/readiness.json"
exit 2
