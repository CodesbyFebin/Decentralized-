# Decentralized.Host Command Centre

A console for the Decentralized.Host control plane (`dh-control`). The
control plane is authoritative; this app is a presentation layer with a thin
BFF that forwards your operator capability, validates input, normalizes the
control plane's view and sanitizes errors. It holds no operational state.

> ADR 0007 makes the embedded, no-build console in `web/dist` the default.
> This app has a build step and npm dependencies; see the proposed ADR 0009.

## Run against a real cluster

```bash
# from the repository root
make build
./bin/dh dev up --dir ./devcluster            # 3 control-plane members, 3 hosts, 1 edge
export DH_HOME=./devcluster/operator

cd command-centre
npm install                                   # or: bun install
PLATFORM_ADAPTER=controlplane \
DH_CONTROL_URL=http://127.0.0.1:17701,http://127.0.0.1:17702,http://127.0.0.1:17703 \
npm run dev                                   # http://127.0.0.1:3000
```

Sign in with a capability minted by your cluster root:

```bash
dh token --ttl 12h               # operator: api.read, api.write, api.admin
dh token --read-only --ttl 12h   # viewer
```

or open `http://127.0.0.1:3000/#token=<capability>` (the fragment is exchanged
for an HttpOnly cookie and removed). The control plane verifies the capability
and enforces what it allows; the console hides nothing it relies on for security.

Production: `npm run build && PLATFORM_ADAPTER=controlplane DH_CONTROL_URL=… npm start`.
With TLS clusters set `NODE_EXTRA_CA_CERTS` to the cluster root CA. All options:
`.env.example`.

`PLATFORM_ADAPTER=demo` replays a recorded dev-cluster view for UI work. Every
value is labelled SIMULATED, every mutation is refused, and production refuses
it unless `ALLOW_DEMO_IN_PRODUCTION=1`. There is no fallback between adapters.

Optional: `DH_EVIDENCE_DIR` points the Evidence page at sealed validation
records (the repository's `evidence/`), and `DH_CLI` at a `dh` binary so the
console can verify a record with `dh evidence verify`.

## What is real

See `docs/command-centre/reality-matrix.md`: hosts, applications, deployments,
volumes, artifacts, routes, certificates, audit and evidence are live from the
control plane; visitor analytics, bandwidth, billing, DePIN and the marketplace
are shown as UNAVAILABLE or PLANNED instead of being filled in.

## Checks

```bash
npm run lint              # tsc --noEmit (strict)
npm test                  # unit tests on recorded real views + failure injection (offline, timeout, malformed, 5xx, config)
npm run gate              # no-mock production gate (also runs as part of npm run build)
DH_CONTROL_URL=… DH_TOKEN_ADMIN=$(dh token --ttl 2h) DH_TOKEN_READ=$(dh token --read-only --ttl 2h) \
  npm run test:integration   # against a live cluster: RBAC, CSRF, deploy lifecycle, drain, Copilot approvals, audit actor
CC_URL=http://127.0.0.1:3000 DH_TOKEN_ADMIN=… DEV_CLUSTER_DIR=../devcluster \
  node tests/e2e/disconnect.mjs   # freezes every control-plane member; the console must lose confidence
CC_URL=… DH_TOKEN_ADMIN=… DEV_CLUSTER_DIR=… \
  node tests/e2e/wave1-disconnect.mjs   # the same, checked on every Wave 1 page at once
```

Page-by-page status: `docs/command-centre/wave1.md`.

Qualification record: `docs/qualification/RC1.md`.
