#!/bin/sh
# Rebuild the PV1-S1-A03 offline kit from its parts, verify it, then extract.
# Nothing is extracted unless every part and the rebuilt archive verify.
set -eu
cd "$(dirname "$0")"
KIT=dh-pv1-offline-kit-linux-amd64.tar.gz
EXPECT=fba3108f834ea4974bf660817208784c7e59fb464eb3b9289e5d2954b1517e73
echo "== verifying parts"
sha256sum -c PARTS-SHA256SUMS
echo "== reassembling $KIT"
cat parts/$KIT.part-* > "$KIT"
GOT=$(sha256sum "$KIT" | cut -d' ' -f1)
if [ "$GOT" != "$EXPECT" ]; then
  echo "rebuilt archive SHA-256 is $GOT, expected $EXPECT: not extracting" >&2
  rm -f "$KIT"
  exit 1
fi
echo "$KIT: SHA-256 $GOT matches the published value"
tar xzf "$KIT"
echo "== extracted to $(pwd)/dh-pv1-offline-kit-linux-amd64"
echo "next: cd dh-pv1-offline-kit-linux-amd64 && sha256sum -c KIT-SHA256SUMS && sh verify-kit.sh"
