#!/bin/sh
# NODE-A01 — sovereign node foundation, sealed as signed evidence.
#
# Scope: host identity and key custody, enrolment and its refusals, measured
# hardware facts, owner approval, authenticated heartbeats, freshness under
# control-plane and host disconnects, command rejection on the host, revoked
# keys, and audit / host-ledger tamper detection. Out of scope: marketplace,
# leases, settlement, tokens, external DePIN, ZK, coordinator federation.
#
# Usage: sh validation/node-a01.sh EVIDENCE_DIR EVIDENCE_ID "SCOPE" [PARENT_ID]
#   e.g. sh validation/node-a01.sh evidence/NODE-A01-A01 NODE-A01-A01 "Linux container, loopback"
set -u
[ $# -ge 3 ] || { echo "usage: $0 EVIDENCE_DIR EVIDENCE_ID SCOPE [PARENT_ID]" >&2; exit 2; }
mkdir -p "$1" && EVID=$(cd "$1" && pwd)
ID=$2; SCOPE=$3; PARENT=${4:-}
STAGE=${ID%-A*}
if [ -e "$EVID/record.json" ] || [ -e "$EVID/steps.jsonl" ]; then
  echo "$EVID already holds an attempt; use a new directory for a new attempt" >&2; exit 2
fi
SRC=$(pwd)
OUT="$EVID/bin"
mkdir -p "$OUT"
go build -trimpath -o "$OUT/" ./cmd/... || exit 1
DH="$OUT/dh"
R() { "$DH" evidence run --dir "$EVID" "$@"; }

"$DH" evidence env --data "$EVID" > "$EVID/env.json"
"$DH" evidence digest --src "$SRC" > "$EVID/source-digest.txt"
R --name build         -- go build -trimpath ./...
R --name vet           -- go vet ./...
R --name gofmt         -- sh -c 'test -z "$(gofmt -l cmd pkg tests conformance web)"'
R --name unit-race     -- go test -count=1 -race ./pkg/... ./cmd/...
R --name gossip-repeat -- go test -race -count=20 -run TestGossipMembership ./pkg/mesh
R --name conformance-go    -- "$OUT/dh-conformance" run -self
R --name conformance-stdio -- "$OUT/dh-conformance" run -adapter "$OUT/dh-conformance adapter"
R --name conformance-python -- "$OUT/dh-conformance" run -adapter "python3 conformance/python/adapter.py"
R --name host-commands -- go test -count=1 -v -run 'TestCompromisedControlPlaneCannotCommandHost|TestHardwareProbe' ./pkg/node
R --name node-a01      -- go test -count=1 -v -run TestNodeA01 ./tests/integration
R --name integration   -- go test -count=1 -timeout 40m -v ./tests/integration/...
# The console reads the facts this checkpoint changed; its checks are re-run.
# dh-src-digest/2 does not hash .ts/.tsx, so the TypeScript sources are bound
# by a hash manifest stored in the record.
R --name cc-source-manifest -- sh -c "node command-centre/scripts/source-manifest.mjs '$EVID/cc-source-manifest.txt'"
R --name cc-typecheck  -- sh -c 'cd command-centre && npx tsc --noEmit'
R --name cc-unit       -- sh -c 'cd command-centre && npm test'
R --name cc-no-mock-gate -- sh -c 'cd command-centre && npm run gate'

SEAL="--dir $EVID --src $SRC --bin $OUT --id $ID --stage $STAGE"
[ -n "$PARENT" ] && SEAL="$SEAL --parent $PARENT"
# shellcheck disable=SC2086
"$DH" evidence seal $SEAL \
  --require build,vet,gofmt,unit-race,gossip-repeat,conformance-go,conformance-stdio,conformance-python,host-commands,node-a01,integration,cc-source-manifest,cc-typecheck,cc-unit,cc-no-mock-gate \
  --claim "NODE-A01: a host generates and keeps its own Ed25519 key; enrolment refuses reused, forged, expired, revoked and replayed attempts; facts are measured, not declared; approval, heartbeats, disconnects, command rejection, key revocation and tamper detection behave as specified; the existing integration suite still passes." \
  --scope "$SCOPE" \
  --exclude "multi-machine behaviour and NAT (single machine, loopback transport)" \
  --exclude "chaos scenarios and the TLS install runbook (covered by the REF-MAC / PV1 gate set, not re-run here)" \
  --exclude "marketplace, leases, settlement, tokens, external DePIN, ZK, coordinator federation" \
  --limitation "${PY_NOTE:-python3 is the first one on PATH}" \
  --limitation "hardware facts were measured in the environment named in env.json; GPU, NIC and NAT discovery were not exercised against real devices"
"$DH" evidence verify --dir "$EVID"
