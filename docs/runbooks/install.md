# Install a cluster

This is the production shape: three control-plane members serving TLS, and
hosts joining with single-use tokens. Every command below was run end to end
against the binaries in this repository.

## 0. Build

```bash
make build        # bin/dh, bin/dh-control, bin/dh-noded, bin/dh-conformance, bin/dh-beacon
```

Copy `dh-control` to control-plane machines, `dh-noded` to hosts, and keep
`dh` on the operator machine.

## 1. Create the trust root (operator machine)

```bash
export DH_HOME=~/.dh
dh init --cluster prod           # TLS is on by default
```

This creates `~/.dh/prod/root/identity.key`, the cluster root key, and
`root-ca.pem`. **The root key is the trust anchor every host pins.** Keep it
offline or on a hardware-protected machine. You need it to:
- bootstrap members and add or remove them;
- create invites;
- sign rosters, federation agreements and root rotations.

Day-to-day operations use short-lived capabilities derived from it.

## 2. Start control-plane members

On each of three machines:

```bash
dh-control --data /var/lib/dh-control \
  --api 0.0.0.0:7700 --api-advertise cp1.example.net:7700 \
  --raft 0.0.0.0:7800 --raft-advertise cp1.example.net:7800 \
  --mesh 0.0.0.0:51900 --mesh-advertise cp1.example.net:51900 \
  --tls
```

Before a member has credentials, it serves a throwaway self-signed
certificate. It prints that certificate's fingerprint and writes it to
`bootstrap.fingerprint`, next to the one-time `bootstrap.code`, in its data
directory. Copy both files to the operator machine over a channel you trust,
such as scp or your configuration management.

## 3. Bootstrap and add members (operator machine)

```bash
dh cp bootstrap  --api cp1.example.net:7700 --code-file ./cp1/bootstrap.code   # fingerprint read from ./cp1/bootstrap.fingerprint
dh cp add-member --api cp2.example.net:7700 --code-file ./cp2/bootstrap.code
dh cp add-member --api cp3.example.net:7700 --code-file ./cp3/bootstrap.code
dh cp status
```

The CLI pins each member's throwaway certificate, so the code and the
credentials never travel in cleartext. Once a member holds credentials, it
serves a certificate issued by the root CA for its advertised names, with no
restart. From then on the CLI verifies members against `root-ca.pem`. Pass
`--fingerprint sha256:…` if the fingerprint file is not next to the code file.

Expected `dh cp status`: one `leader`, two `follower`s, and all three voters.

## 4. Join hosts

**Invites are single-use and short-lived** (15 minutes unless you pass
`--ttl`). Create one per host, just before you start it:

```bash
dh node invite --out host-1.token          # add --roles edge for an edge host, --auto to skip approval
dh node invite-revoke host-1.token         # withdraw it if it leaked or is no longer needed
```

On the host:

```bash
dh-noded --data /var/lib/dh-noded --join-file host-1.token \
  --name host-1 --region eu-west --zone a --host rack1-u12 \
  --mesh 0.0.0.0:51820 --mesh-advertise host-1.example.net:51820
```

On first start, the host writes `policy.yaml` to its data directory: its
**sovereign policy**, covering accepted tiers, runtimes, resource caps,
digest pinning, artifact signatures, federated work, remote exec and
offline behaviour. Edit it and restart the agent to change it. The control
plane can read the policy but never change it.

The token pins the root key and root CA, so the host speaks HTTPS to the
control plane from its first request. If a token was already used, the host
reports `join token already used by …`. Give it a new token and restart it
with `--join-file`; a host that never enrolled accepts the replacement.

Approve pending hosts:

```bash
dh get nodes
dh node approve host-1
```

## 5. Deploy

```bash
dh artifact push ./myservice --name myservice --sign     # prints myservice@b3:<digest>
cat > web.yaml <<'EOF'
apiVersion: dh/v1
kind: Application
metadata: {name: web}
spec:
  replicas: 2
  image: myservice@b3:<digest>
  resources: {cpu: 100m, mem: 32Mi}
  placement: {tiers: [trusted], spread: failure-domain, antiAffinity: hard}
  ports: [{name: http}]
  health: {http: /healthz, interval: 1s}
EOF
dh apply -f web.yaml
dh rollout status app web
dh describe app web
```

Process workloads receive their port in `$PORT`. For ingress, add an edge
host and `ingress: [{host: web.example.net, port: http, tls: acme}]`. Edge
hosts take `--edge-http :80 --edge-https :443 --acme-directory … --acme-email …`.

## 6. Console

```bash
dh console              # or --read-only; prints https://<member>/#token=…
```

The console is served by every member. The session capability travels in
the URL fragment, which browsers never send to servers. Browsers need
`root-ca.pem` trusted to open the TLS console, or you can front it with
your own TLS terminator.

## 7. Verify

```bash
dh audit verify                     # ledger chain and signed checkpoints, verified locally
dh mesh doctor                      # WireGuard bindings, handshakes and gossip
dh-conformance run -self            # protocol vectors against this build
dh cp backup --out cp-$(date +%F).json && dh verify backup cp-$(date +%F).json
```
