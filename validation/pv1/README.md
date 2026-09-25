# PV-1 — Real Infrastructure Validation

| Stage | Environment | Status |
|---|---|---|
| Reference | this development Mac (macOS, amd64): build, race tests, integration, chaos, conformance | reference only; promotes nothing |
| **PV1-S1** Linux parity | one independent Linux VM or machine: Ubuntu or Debian, ≥ 4 vCPU, ≥ 8 GB RAM, ≥ 40 GB SSD | A03 pending (history: `evidence/PV1-S1-HISTORY.md`) |
| PV1-S2 LAN multi-machine | 3 separate Linux VMs or machines on one LAN | after S1 passes |
| PV1-S3 partition and failure | same cluster; real partitions with firewall rules, power-off, NIC down | after S2 |
| PV1-S4 WAN and NAT | machines in ≥ 2 networks, at least one behind NAT | after S3 |

## Running PV1-S1 (A03) on a Linux VM

1. **Provision the VM.** Ubuntu 24.04 or Debian 12, ≥ 4 vCPU, 8 GB RAM, 40 GB
   SSD. Install:
   ```bash
   sudo apt-get install -y build-essential python3 python3-cryptography curl
   ```
   Install Go 1.26 from go.dev/dl. Docker is optional; without it, the `oom`
   and `postgres-outage` chaos scenarios SKIP and the record says so.
2. **Package the exact source tree** on the Mac:
   ```bash
   ./bin/dh evidence pack --out dh-src.tgz
   ```
   It prints the digest. Copy `dh-src.tgz` to the VM. Don't write `tar`
   excludes by hand: bsdtar's `--exclude=./evidence` also drops
   `docs/evidence` and `pkg/evidence`. That was found the hard way.
3. **On the VM:**
   ```bash
   mkdir dh && cd dh && tar xzf ../dh-src.tgz
   go run ./cmd/dh evidence digest
   ```
   This must print the same digest. If it doesn't, stop: you are not testing
   the same tree.
4. **Run the stage:**
   ```bash
   sh validation/pv1/stage-parity.sh ~/pv1-s1-a03 PV1-S1-A03 "independent Linux VM: <provider>, <vCPU>/<RAM>, <disk>" PV1-S1-A02
   ```
   Use your own words for the scope, and state exactly what the machine is.
   The run takes about 20–30 minutes. Every step is recorded, including
   failures and retries.
5. **Verify and bring the evidence back:**
   ```bash
   ./dh evidence verify --dir ~/pv1-s1-a03
   ```
   (the binary is in `~/pv1-s1-a03/bin/`). Then copy `~/pv1-s1-a03` into
   `evidence/` next to the history file and add an A03 entry there.

A record promotes a claim only if it PASSED, its digest matches the tree
the claim refers to, and its scope names the environment the claim implies.
