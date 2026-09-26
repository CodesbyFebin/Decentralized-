#!/bin/sh
# CC-W1 — Command Centre Wave 1, sealed as signed evidence.
#
# Needs a running `dh dev up` cluster and the console served from THIS tree
# (npm run build; npm start) with DH_EVIDENCE_DIR and DH_CLI set.
#
#   DH_CONTROL_URL=http://127.0.0.1:17701,… DH_TOKEN_ADMIN=… DH_TOKEN_READ=… \
#   DEV_CLUSTER_DIR=… CC_URL=http://127.0.0.1:3100 EDGE_URL=http://127.0.0.1:18103 \
#   PLAYWRIGHT_MODULE=… sh validation/cc-wave1.sh EVIDENCE_DIR EVIDENCE_ID "SCOPE" [PARENT_ID]
set -u
[ $# -ge 3 ] || { echo "usage: $0 EVIDENCE_DIR EVIDENCE_ID SCOPE [PARENT_ID]" >&2; exit 2; }
for v in DH_CONTROL_URL DH_TOKEN_ADMIN DH_TOKEN_READ DEV_CLUSTER_DIR CC_URL; do
  eval "[ -n \"\${$v:-}\" ]" || { echo "$v is required" >&2; exit 2; }
done
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
R --name cc-source-manifest -- sh -c "node command-centre/scripts/source-manifest.mjs '$EVID/cc-source-manifest.txt'"
R --name go-vet            -- go vet ./...
R --name go-unit-race      -- go test -count=1 -race ./pkg/control/... ./pkg/cli/... ./pkg/node/... ./pkg/mesh/...
R --name node-a01          -- go test -count=1 -v -run TestNodeA01 ./tests/integration
R --name cc-typecheck      -- sh -c 'cd command-centre && npx tsc --noEmit'
R --name cc-unit           -- sh -c 'cd command-centre && npm test'
R --name cc-no-mock-gate   -- sh -c 'cd command-centre && npm run gate'
R --name cc-build          -- sh -c 'cd command-centre && npx vite build'
R --name cc-integration    -- sh -c 'cd command-centre && npm run test:integration'
R --name cc-disconnect     -- sh -c "cd command-centre && OUT_DIR='$EVID/disconnect' node tests/e2e/disconnect.mjs"
R --name cc-w1-disconnect  -- sh -c "cd command-centre && OUT_DIR='$EVID/wave1-disconnect' node tests/e2e/wave1-disconnect.mjs"

SEAL="--dir $EVID --src $SRC --bin $OUT --id $ID --stage $STAGE"
[ -n "$PARENT" ] && SEAL="$SEAL --parent $PARENT"
# shellcheck disable=SC2086
"$DH" evidence seal $SEAL \
  --require cc-source-manifest,go-vet,go-unit-race,node-a01,cc-typecheck,cc-unit,cc-no-mock-gate,cc-build,cc-integration,cc-disconnect,cc-w1-disconnect \
  --claim "Command Centre Wave 1 (dashboard, apps, app detail, nodes, node detail, add node, evidence, evidence detail, activity, operations, settings): strict typecheck, unit tests, no-mock gate, build, live-cluster integration including invites, validation records and derived operations, and real backend-disconnect tests on every Wave 1 page." \
  --scope "$SCOPE" \
  --exclude "Wave 2+ pages (deploy wizard, domains, storage, security, analytics, marketplace, billing, DePIN)" \
  --exclude "automated WCAG audit (axe not available in this environment)" \
  --exclude "multi-machine behaviour (single machine, loopback transport)" \
  --limitation "dh-src-digest/2 does not hash .ts/.tsx files; the Command Centre source is bound by cc-source-manifest.txt, a hashed artifact of this record" \
  --limitation "signed by an ephemeral validator key created in the sandbox; the dev cluster and its keys are disposable" \
  --limitation "the console under test is the one served at CC_URL; the operator must serve it from this tree"
"$DH" evidence verify --dir "$EVID"
