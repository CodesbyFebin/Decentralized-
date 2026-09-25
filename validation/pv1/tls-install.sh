#!/bin/sh
# The production install runbook (docs/runbooks/install.md), scripted:
# 3 TLS members, fingerprint-pinned bootstrap, 2 hosts with one token each,
# approval, signed artifact, deploy, rollout, audit verification.
set -eu
B=$1; T=$2; rm -rf "$T"; mkdir -p "$T"
export DH_HOME="$T/op"
trap 'pkill -f "$T/" 2>/dev/null || true' EXIT
"$B/dh" init --cluster pv
for i in 1 2 3; do
  "$B/dh-control" --data "$T/cp$i" --api 127.0.0.1:3770$i --raft 127.0.0.1:3780$i --mesh 127.0.0.1:3790$i --tls > "$T/cp$i.log" 2>&1 &
done
for i in 1 2 3; do until [ -s "$T/cp$i/bootstrap.fingerprint" ]; do sleep 0.2; done; done
"$B/dh" cp bootstrap  --api 127.0.0.1:37701 --code-file "$T/cp1/bootstrap.code"
"$B/dh" cp add-member --api 127.0.0.1:37702 --code-file "$T/cp2/bootstrap.code"
"$B/dh" cp add-member --api 127.0.0.1:37703 --code-file "$T/cp3/bootstrap.code"
"$B/dh" cp status
for h in 1 2; do
  "$B/dh" node invite --out "$T/join$h.token"
  "$B/dh-noded" --data "$T/h$h" --join-file "$T/join$h.token" --name host-$h --host phys-$h \
    --mesh 127.0.0.1:3795$h --status 127.0.0.1:3796$h > "$T/h$h.log" 2>&1 &
done
for h in 1 2; do until "$B/dh" node approve host-$h 2>/dev/null; do sleep 0.5; done; done
IMG=$("$B/dh" artifact push "$B/dh-beacon" --name dh-beacon --sign | grep -o 'dh-beacon@b3:[0-9a-f]*' | head -1)
cat > "$T/web.yaml" <<YAML
apiVersion: dh/v1
kind: Application
metadata: {name: web}
spec:
  replicas: 2
  image: $IMG
  resources: {cpu: 100m, mem: 32Mi}
  placement: {tiers: [trusted], spread: failure-domain, antiAffinity: hard}
  ports: [{name: http}]
  health: {http: /healthz, interval: 1s}
YAML
"$B/dh" apply -f "$T/web.yaml"
"$B/dh" rollout status app web --timeout 90s
"$B/dh" get nodes
"$B/dh" audit verify
