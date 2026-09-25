#!/bin/sh
# Full reference qualification on one machine, sealed as signed evidence.
#   PV1-S1 (Linux parity): authoritative only on a real Linux VM or machine.
#   REF-MAC (macOS reference): the development reference; promotes nothing.
#
# Usage: sh validation/pv1/stage-parity.sh EVIDENCE_DIR EVIDENCE_ID "SCOPE" [PARENT_ID]
#   e.g. sh validation/pv1/stage-parity.sh ~/pv1-s1-a03 PV1-S1-A03 "Ubuntu 24.04 VM, 4 vCPU/8 GB, KVM" PV1-S1-A02
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
R --name build        -- go build -trimpath ./...
R --name vet          -- go vet ./...
R --name gofmt        -- sh -c 'test -z "$(gofmt -l cmd pkg tests conformance web)"'
R --name unit-race    -- go test -count=1 -race ./pkg/... ./cmd/... ./conformance/...
# Regression candidate from PV1-S1-A02: the original single-join assertion, repeated.
R --name gossip-repeat -- go test -race -count=20 -run TestGossipMembership ./pkg/mesh
R --name tools        -- sh -c 'GOBIN=$PWD/tools/bin go install github.com/letsencrypt/pebble/v2/cmd/pebble@v2.10.1 github.com/letsencrypt/pebble/v2/cmd/pebble-challtestsrv@v2.10.1'
R --name conformance-check  -- "$OUT/dh-conformance" check
R --name conformance-go     -- "$OUT/dh-conformance" run -self
R --name conformance-stdio  -- "$OUT/dh-conformance" run -adapter "$OUT/dh-conformance adapter"
R --name conformance-python -- "$OUT/dh-conformance" run -adapter "python3 conformance/python/adapter.py"
R --name integration  -- go test -count=1 -timeout 40m -v ./tests/integration/...
R --name chaos        -- "$DH" chaos run --scenario all --out "$EVID/chaos"
R --name tls-install  -- sh validation/pv1/tls-install.sh "$OUT" "$EVID/tls-install"

SEAL="--dir $EVID --src $SRC --bin $OUT --id $ID --stage $STAGE"
[ -n "$PARENT" ] && SEAL="$SEAL --parent $PARENT"
# shellcheck disable=SC2086
"$DH" evidence seal $SEAL \
  --require build,vet,gofmt,unit-race,gossip-repeat,tools,conformance-check,conformance-go,conformance-stdio,conformance-python,integration,chaos,tls-install \
  --claim "The M1–M8 gate set (build, format, vet, race unit tests, repeated gossip join, conformance incl. independent Python, all integration and regression tests, 17 chaos scenarios, scripted TLS install runbook) passes in this environment." \
  --scope "$SCOPE" \
  --exclude "multi-machine behaviour (single machine, loopback transport)" \
  --exclude "public ACME (Pebble only)" \
  --limitation "chaos scenarios SKIP when their prerequisites (Docker, Postgres) are absent; SKIPs are listed in the chaos log"
"$DH" evidence verify --dir "$EVID"
