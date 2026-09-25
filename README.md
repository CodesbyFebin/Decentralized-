# pv1-offline-kit (branch)

The PV1-S1-A03 offline kit for Linux x86_64, delivered through git for machines that can
reach `github.com` but not GitHub's release download host. This branch shares no history with
`main`, and the kit qualifies source commit `ef849e2` (digest `b3:2a23e1da…`).

The kit is the release asset `dh-pv1-offline-kit-linux-amd64.tar.gz` (tag `ref-mac-a03`), split into
45 MB parts. Published SHA-256:
`fba3108f834ea4974bf660817208784c7e59fb464eb3b9289e5d2954b1517e73`

```bash
git clone --branch pv1-offline-kit --depth 1 https://github.com/CodesbyFebin/Decentralized-.git dh-kit
cd dh-kit
sh reassemble.sh        # checks every part, then the rebuilt archive, and only then extracts
cd dh-pv1-offline-kit-linux-amd64
sha256sum -c KIT-SHA256SUMS
sh verify-kit.sh
sh run-pv1-s1-a03.sh "<exact machine description>" /path/to/ef849e2-clone
```

The kit's own `README.md` covers prerequisites, dry runs and where evidence is written.
