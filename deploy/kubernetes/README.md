# Kubernetes control plane (optional)

This deploys three `dh-control` members as a StatefulSet with separate persistent
volumes. Kubernetes schedules the control-plane processes; Decentralized.Host
still supplies its own Raft membership, signed host admission, policy, ledger,
and workload scheduler. `dh-noded` runs on independently administered hosts
using the [host installation procedure](../../docs/runbooks/install.md).
This path is the project's default runtime readiness path. QEMU is no longer
a prerequisite for this deployment. `P1-LOCAL-VM-A01` remains a separate VM
campaign and retains its own evidence requirements.

Run `./p1-qualification-master.sh` to collect read-only Kubernetes observations
in `deploy/kubernetes/evidence/readiness.json`. Its identifier is
`K8S-CONTROL-A01`; it exits BLOCKED until authenticated quorum, admitted hosts,
continuous traffic, actual fault/recovery, and signed evidence are verified.
Pod readiness alone cannot qualify the application. The old VM baseline is
available explicitly through `./p1-qualification-master.sh --local-vm`.

## Build and deploy

1. Build `deploy/kubernetes/Dockerfile` from the repository root, push the
   image to a registry accessible to the cluster, and obtain its immutable
   `image@sha256:...` reference. Replace `REPLACE_WITH_IMAGE_AT_SHA256_DIGEST`
   in `control-plane.yaml` with that reference. Pin the base images by digest
   too for a reproducible release build.
2. Ensure a default StorageClass provides persistent `ReadWriteOnce` volumes
   and all three pods can reach one another on TCP 7700/7800 and UDP 51900.
   Provide ingress or routable addresses for hosts outside the cluster before
   enrolling hosts. Pod DNS names in this example are internal only.
3. Apply and inspect each pod and claim:

   ```sh
   kubectl apply -f deploy/kubernetes/control-plane.yaml
   kubectl -n decentralized-host get pods,pvc -l app=dh-control
   kubectl -n decentralized-host get pvc
   ```

The headless Service publishes pod DNS before readiness so members can find
one another. A Kubernetes `Running` or `Ready` pod only proves the API TCP
socket is reachable; it does **not** prove Raft quorum or qualification.

## Establish membership

Use the operator `dh` CLI and the trust-root procedure in
[install.md](../../docs/runbooks/install.md). On the trusted operator machine,
run `dh init --cluster <name>` and keep the root identity there, outside the
cluster. Retrieve each pod's `bootstrap.code` and `bootstrap.fingerprint`
through a trusted channel. They live on that pod's persistent volume and are
one-time bootstrap material; never put them in a ConfigMap, log, ticket, or Git.

Reach each member through an authenticated administrative connection or a
separate `kubectl port-forward pod/dh-control-N LOCAL:7700` session. Use
`dh cp bootstrap` for member 0 and `dh cp add-member` for members 1 and 2,
with each member's own code and fingerprint as described in install.md. Use
the actual reachable API address for each CLI call. Verify the CLI's pinned
fingerprint and root CA behavior, then run `dh cp status` and require three
voters with one leader. Do not treat a TCP probe as this verification.

The advertised names above are Kubernetes pod DNS. External hosts must reach
the advertised control-plane API and mesh endpoints and validate the issued
certificates. If they cannot, supply stable externally routable addresses and
an appropriate network design before joining them. Do not expose Raft or
bootstrap material publicly.

## Operations

Keep all three PVCs across pod restarts. Back up the cluster with the signed
`dh cp backup` and `dh verify backup` workflow. Do not delete or clone a
member's PVC to replace a pod: its signing identity and Raft state belong to
that member. Kubernetes replicas and persistent volumes are infrastructure
configuration; observe quorum, host state, and signed evidence separately.
