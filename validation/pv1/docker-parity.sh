#!/bin/sh
# DIAGNOSTIC ONLY: runs the PV1-S1 script inside a Linux container on a
# workstation (for example Colima on macOS). A run here never establishes
# Linux parity (see evidence/PV1-S1-HISTORY.md). Use it to reproduce a
# finding, not to promote a claim.
set -eu
cd "$(dirname "$0")/../.."
TS=$(date -u +%Y%m%dT%H%M%SZ)
EVID="evidence/pv1-s1-diagnostic-docker-$TS"
mkdir -p "$EVID"
IMAGE=${IMAGE:-golang:1.26-bookworm}
docker run --rm --init --shm-size=1g \
  -v "$PWD":/src:ro -v "$PWD/$EVID":/evidence \
  -v "$(go env GOMODCACHE)":/go/pkg/mod \
  -v dh-gocache:/root/.cache/go-build \
  "$IMAGE" sh -c '
    set -e
    mkdir -p /work && cd /src && tar --exclude=./bin --exclude=./tools/bin --exclude=./evidence --exclude=./chaos-reports -cf - . | tar -C /work -xf -
    cd /work && sh validation/pv1/stage-parity.sh /evidence "one Linux container (Docker, image '"$IMAGE"') on a Linux VM hosted by this macOS machine; multi-process over loopback; no Docker-in-Docker"'
echo "evidence: $EVID"
