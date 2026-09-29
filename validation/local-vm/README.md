# LOCAL-VM-P1-QUALIFICATION-KIT

**P1-LOCAL-VM-A01: Distributed Private Cloud Qualification on Single Physical Host**

Real P1 qualification proving distributed machine/VM/OS/node/network failure boundaries on your hardware.

> **Precisely scoped qualification:** Three distinct Linux VMs on one physical host, demonstrating real failure detection and workload rescheduling. NOT a multi-physical-host or multi-operator qualification (those require additional physical domains for P2).
>
> **Verification surfaces (NOT P1 nodes):**
> - GitHub Actions: Independent evidence verification from fresh environment
> - Xcode Cloud (optional, Apple Developer members): Additional CI verification backend
> - Hugging Face Spaces (optional): External portability/integration tests
>
> **Aligned with Decentralized Host's sovereign, local-first premise.** No cloud provider lock-in. No credit card required. Complete control over evidence collection and network topology.

## What's Inside

- **Hypervisor Support**: QEMU, UTM (macOS), VMware, VirtualBox
- **Network Orchestration**: WireGuard VPN for deterministic latency measurement
- **Cloud-init Templates**: Same baseline/network evidence collection as OCI kit
- **Orchestration Scripts**: VM lifecycle, failure injection, evidence collection
- **CI/CD Integration**: GitHub Actions for automated qualification runs
- **Portability**: Hugging Face Spaces for remote result aggregation
- **Domain Classification**: Local topology with measured regional/network properties

## Quick Start

### 1. Prerequisites

```bash
# Install hypervisor (choose one)
# macOS:
brew install utm qemu  # or install VMware Fusion, VirtualBox

# Linux:
sudo apt-get install -y qemu qemu-kvm libvirt-daemon-system

# Windows:
# Download VirtualBox or Hyper-V

# Install WireGuard
brew install wireguard-tools  # macOS
sudo apt-get install wireguard-tools  # Linux
choco install wireguard  # Windows

# Install Terraform (for optional templating)
brew install terraform
```

### 2. Create Local VM Configuration

```bash
cd validation/local-vm

# For QEMU-based setup
bash scripts/create-vm-cluster.sh qemu \
  --nodes 3 \
  --cpu 2 \
  --memory 4096 \
  --disk 50 \
  --network wireguard

# For UTM (macOS):
bash scripts/create-vm-cluster.sh utm \
  --nodes 3 \
  --network wireguard

# For VirtualBox:
bash scripts/create-vm-cluster.sh virtualbox \
  --nodes 3 \
  --network host-only
```

### 3. Provision VMs

```bash
# Bring up all 3 nodes
bash scripts/start-cluster.sh

# Verify SSH access
bash scripts/bootstrap-nodes.sh

# Qualify network (latency, NTP, connectivity)
bash scripts/qualify-network.sh
```

### 4. Deploy Decentralized Host

```bash
# Build binaries (from repo root)
cd ../..
make clean && make build

# Copy to each node
for node in dh-local-01 dh-local-02 dh-local-03; do
  scp -r build/dhp ubuntu@${node}:/opt/decentralized-host/
done

# Start agents (via cloud-init or manual systemctl)
for node in dh-local-01 dh-local-02 dh-local-03; do
  ssh ubuntu@${node} "sudo systemctl start decentralized-host-agent"
done
```

### 5. Run P1-E2E Scenarios

```bash
# Option A: Local orchestration
bash scripts/run-scenarios.sh local

# Option B: GitHub Actions (push + trigger)
git push -u origin p1-local-vm
# GitHub Actions automatically runs qualification suite

# Option C: Hugging Face Spaces (for result aggregation)
bash scripts/push-results-to-huggingface.sh
```

### 6. Collect & Validate Evidence

```bash
# Collect from all nodes
bash scripts/collect-evidence.sh ./evidence

# Validate signatures
cd tools/p1-e2e/validator
go build -o validator
../../../validator -dir ../../validation/local-vm/evidence -out validation-report.json

# Upload to Hugging Face for sharing (optional)
bash validation/local-vm/scripts/upload-results.sh
```

### 7. Cleanup

```bash
# Stop all VMs
bash scripts/stop-cluster.sh

# Destroy VMs (careful!)
bash scripts/destroy-cluster.sh
```

## Locked Topology: P1-LOCAL-VM-A01

```
┌─────────────────────────────────────────────────┐
│        Physical Host (Your Laptop/Mac/Linux)    │
│                                                 │
│  ┌──────────────────────────────────────────┐  │
│  │  Hypervisor (QEMU/UTM/VirtualBox/KVM)    │  │
│  │                                          │  │
│  │  ┌─────────────┐  ┌─────────────┐       │  │
│  │  │ dh-node-01  │  │ dh-node-02  │       │  │
│  │  │ 2 CPU, 4GB  │  │ 2 CPU, 4GB  │       │  │
│  │  └─────────────┘  └─────────────┘       │  │
│  │         │              │                 │  │
│  │  ┌──────┴──────────────┴────────┐       │  │
│  │  │  192.168.200.0/24 (VM net)   │       │  │
│  │  │  D.H. cluster network        │       │  │
│  │  │  - Membership                │       │  │
│  │  │  - Placement                 │       │  │
│  │  │  - Workload runtime          │       │  │
│  │  │  - Service discovery         │       │  │
│  │  │  - Ingress                   │       │  │
│  │  └──────┬──────────────┬────────┘       │  │
│  │         │              │                 │  │
│  │  ┌─────────────────────────────┐        │  │
│  │  │  dh-node-03 (Control Plane) │        │  │
│  │  │  4 CPU, 8GB, persistent disk│        │  │
│  │  └─────────────────────────────┘        │  │
│  └──────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘

⬇ Evidence Collection (this host)

┌─────────────────────────────────────────────────┐
│  GitHub Actions (Independent Verifier)          │
│  - Verify committed evidence bundle             │
│  - Tamper-copy negative control                 │
│  - Regression tests                             │
│  - Fresh environment validation                 │
│  NOT A P1 NODE                                  │
└─────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────┐
│  Xcode Cloud (Optional, Apple Developer members)│
│  - 25 hours/month included CI compute           │
│  - Build/test/verification workloads            │
│  - Additional independent verifier              │
│  NOT A P1 NODE                                  │
└─────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────┐
│  Hugging Face Spaces (Optional Portability)     │
│  - External integration testing                 │
│  - Shareable results                            │
│  NOT A P1 NODE, NOT IN QUORUM                   │
└─────────────────────────────────────────────────┘
```

### Domain Classification

```json
{
  "PHYSICAL_HOSTS": 1,
  "VM_INSTANCES": 3,
  "OS_INSTANCES": 3,
  "NODE_IDENTITIES": 3,
  
  "P1_FAILURE_DOMAINS": {
    "VM_DOMAIN": "DISTINCT",
    "OS_DOMAIN": "DISTINCT",
    "FILESYSTEM_DOMAIN": "DISTINCT",
    "NODE_IDENTITY_DOMAIN": "DISTINCT"
  },
  
  "SHARED_INFRASTRUCTURE": {
    "PHYSICAL_HOST_DOMAIN": "SAME",
    "OPERATOR_DOMAIN": "SAME",
    "POWER_DOMAIN": "SAME",
    "HYPERVISOR_DOMAIN": "SAME"
  },
  
  "QUALIFICATION_SCOPE": {
    "P1_LOCAL_VM_QUALIFICATION": "ELIGIBLE",
    "P1_INDEPENDENT_PHYSICAL_HOST_QUALIFICATION": "NOT_ESTABLISHED",
    "P2_INDEPENDENT_OPERATOR_QUALIFICATION": "NOT_ESTABLISHED"
  }
}
```

### Resource Requirements

- **Host**: 8+ CPU cores, 16GB+ RAM, 150GB free disk
- **Per VM**: 2-4 CPU, 4-8GB RAM, 50GB persistent disk
- **Network**: Real hypervisor network isolation (not application-level)
- **Total**: ~200GB disk, 20GB active RAM
- **Time to Provision**: 10-15 minutes (download image + create VMs)

### Verification Environments (NOT P1 nodes)

**GitHub Actions (Free Tier):**
- Independent evidence verification from fresh Linux environment
- 2000 free minutes/month
- Tamper-copy negative control (modifies one artifact, expects verification to fail)
- Regression testing on each push

**Xcode Cloud (Optional, Apple Developer Members):**
- 25 included compute hours/month
- Build/test/verification workloads
- Apple's managed CI service
- Separate execution environment from P1 cluster

**Hugging Face Spaces (Optional):**
- External portability and integration testing
- Shareable results via Space UI
- NOT counted in P1 failure domains or quorum

## What P1-LOCAL-VM-A01 Establishes

**Proven (✓):**
- Real VM failure boundaries (3 distinct hypervisor instances)
- Real OS failure boundaries (3 distinct Linux kernels)
- Real node identity boundaries (3 distinct D.H. node IDs)
- Real network failure boundaries (hypervisor-level partitioning)
- Real failure detection (<30s, measured)
- Real workload rescheduling (<120s, measured)
- Evidence signature verification (Ed25519 validation)
- Real recovery and reconciliation
- ResourceLedger accuracy before/after failure
- Service interruption and restoration latency

**NOT Established (Requires P2):**
- Multi-physical-host resilience (requires 3+ distinct physical machines)
- Multi-operator resilience (requires 3+ independent operators)
- Multi-provider resilience (requires AWS + Azure + Oracle + others)
- Geographic/regional resilience (requires physically distant nodes)
- Power domain independence (requires separate PDUs/power supplies)
- Independent network operators

**Qualification Progression:**
1. **P1-LOCAL-VM-A01** (this kit): Distributed VM failure, detection, rescheduling
   - Run campaign → Seal evidence → Document P1 properties established
2. **Additional Physical Domains** (if master spec requires): Only for blocked P1 properties
3. **P1-ENDTOEND-A01**: Multi-physical-host qualification (OCI kit or similar)
   - Run campaign → Seal evidence → Signed metering

## File Structure

```
validation/local-vm/
├── README.md                           # This file
├── config/
│   ├── wireguard-topology.json        # WireGuard peer configuration
│   ├── vm-specs.json                   # VM resource allocation
│   └── domains.json                    # Domain classification for local topology
├── cloud-init/
│   ├── dh-node.yaml                   # Same as OCI kit
│   └── networking-setup.sh            # WireGuard peer setup
├── scripts/
│   ├── create-vm-cluster.sh            # Create 3-node cluster (multi-hypervisor)
│   ├── start-cluster.sh                # Boot all VMs
│   ├── bootstrap-nodes.sh              # SSH access + baseline evidence
│   ├── qualify-network.sh              # Network validation (latency, NTP)
│   ├── run-scenarios.sh                # Execute P1-E2E tests locally
│   ├── kill-node.sh                    # Trigger node failure (shutdown)
│   ├── restore-node.sh                 # Recover failed node
│   ├── collect-evidence.sh             # Gather evidence from all nodes
│   ├── upload-results.sh               # Push to Hugging Face dataset
│   ├── stop-cluster.sh                 # Graceful shutdown
│   ├── destroy-cluster.sh              # Clean up VMs (idempotent)
│   └── github-actions-runner.sh        # Self-hosted runner setup (optional)
├── github-actions/
│   ├── p1-qualification.yml            # Main CI/CD workflow
│   ├── nightly-regression.yml          # Scheduled nightly tests
│   └── evidence-aggregation.yml        # Collect + upload results
├── huggingface/
│   ├── spaces-config.yaml              # HF Spaces environment
│   ├── dataset-schema.json             # Expected evidence format
│   └── results-ui.html                 # Shareable results dashboard
├── evidence/                           # Evidence collection output
│   └── .gitkeep
└── .gitkeep
```

## Machine Domain Classification (Local Topology)

```json
{
  "machineDomain":   "distinct",       // 3 separate OS processes / VMs
  "vmDomain":        "distinct",       // 3 separate hypervisor instances
  "operatorDomain":  "SAME/self",      // Your laptop/desktop
  "cloudDomain":     "NONE/local",     // No cloud provider
  "regionDomain":    "SAME/localhost", // Single physical machine
  "ADDomain":        "SAME/localhost", // Single availability zone
  "networkDomain":   "measured",       // WireGuard latency simulated
  "powerDomain":     "SAME/local"      // Single power supply
}
```

**What This Tests (✓):**
- Real machine failure boundaries (distinct VMs)
- Real failure detection (<30s)
- Real workload rescheduling (<120s)
- Evidence signature verification
- Network partition detection (WireGuard partition)

**What This Does NOT Test (for P2):**
- Multi-provider diversity (AWS + Azure + Oracle)
- Multi-region failures (geographic partition)
- Multi-operator resilience (different cloud operators)
- Power domain failures (PSU/PDU events)

## Orchestration: Local vs. GitHub Actions vs. Hugging Face

### Local Orchestration (Interactive)

**Best For:** Development, quick testing, hands-on qualification

```bash
bash scripts/run-scenarios.sh local
# Real-time console output
# Pause/resume capability
# Inspect node state at any time
```

**Timeline:** ~3-4 hours active (can pause between scenarios)

### GitHub Actions (Automated CI/CD)

**Best For:** Regression testing, proof-of-qualification, automated evidence collection

```yaml
# .github/workflows/p1-qualification.yml
on:
  push:
    branches: [p1-local-vm]
  schedule:
    - cron: '0 2 * * *'  # Nightly at 2 AM UTC

jobs:
  p1-e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup cluster
        run: bash validation/local-vm/scripts/create-vm-cluster.sh docker --nodes 3
      - name: Run scenarios
        run: bash validation/local-vm/scripts/run-scenarios.sh github-actions
      - name: Collect evidence
        run: bash validation/local-vm/scripts/collect-evidence.sh
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: p1-evidence
          path: validation/local-vm/evidence/
      - name: Push to Hugging Face
        run: bash validation/local-vm/scripts/upload-results.sh
        env:
          HF_TOKEN: ${{ secrets.HF_TOKEN }}
```

**Timeline:** ~2 hours (fully automated, runs on GitHub's schedule)

### Hugging Face Spaces (Long-Running Persistent)

**Best For:** Continuous qualification, result aggregation, community collaboration

```bash
# Create Space with persistent storage
huggingface-cli repo create \
  --type space \
  --space-sdk docker \
  p1-local-qualification

# Push orchestration scripts
cd validation/local-vm
git push space main

# Space runs P1 scenarios continuously, aggregates evidence
# Results available via shared Space UI
```

**Timeline:** 12-48 hours continuous (accumulate evidence over time)

## Failure Injection

### Kill a Node (Graceful Stop)

```bash
# This stops the VM, triggering failure detection
bash scripts/kill-node.sh dh-local-02

# Control plane should detect failure within 30 seconds
watch -n1 'curl http://control-plane:8080/nodes | jq .nodes[].status'

# Verify workload reschedule (<120s)
watch -n1 'curl http://control-plane:8080/workloads | jq .workloads[].node_id'
```

### Restore Node (Recovery)

```bash
bash scripts/restore-node.sh dh-local-02

# Node should join cluster within 2-3 minutes
# Workloads re-assigned if any pending
```

### Simulate Network Partition (WireGuard)

```bash
# Block traffic from dh-local-02 to others
bash scripts/partition-network.sh dh-local-02

# Control plane maintains quorum (2 of 3)
# Partition detection triggers workload safety checks

# Heal partition
bash scripts/heal-network.sh dh-local-02
```

## Cost Comparison

| Approach | Setup | Per-Run | Storage | Total/Month |
|----------|-------|---------|---------|-------------|
| **Local (QEMU/UTM/VBox)** | $0 | Electricity (~$5) | Local | $5 |
| **GitHub Actions** | $0 | $0 (free tier) | $0 | $0 |
| **Hugging Face Spaces** | $0 | $0 (free tier) | $0 | $0 |
| **OCI Always Free** | $0 | $0 | 100GB free | $0 |
| **AWS Free Tier** | $0 | ~$1-2/run | $0 | $1-2 |

## Troubleshooting

### QEMU: "KVM not available"

```bash
# Check nested virt (if on VM host)
cat /proc/cpuinfo | grep -i kvm

# Fall back to QEMU without KVM (slower)
bash scripts/create-vm-cluster.sh qemu-no-kvm --nodes 3
```

### UTM (macOS): Network connectivity issues

```bash
# Ensure bridged network mode
# UTM → Settings → Network → Bridged

# Or use WireGuard for explicit routing
bash scripts/qualify-network.sh --use-wireguard
```

### VirtualBox: Port forwarding for SSH

```bash
# If using host-only network
VBoxManage modifyvm "dh-local-01" --natpf1 "SSH,tcp,,2201,,22"
ssh -p 2201 ubuntu@localhost
```

### GitHub Actions: Runner self-hosted setup

```bash
# Register runner on your machine (if using local resources)
bash validation/local-vm/scripts/github-actions-runner.sh register

# Or use GitHub's runners (2000 free minutes/month)
# No registration needed
```

### Evidence Collection: SSH key permissions

```bash
# If SSH fails with "Permission denied (publickey)"
chmod 600 ~/.ssh/p1-local-vm
ssh-add ~/.ssh/p1-local-vm
```

## References

- [QEMU Documentation](https://www.qemu.org/documentation/)
- [UTM for macOS](https://mac.getutm.app/)
- [WireGuard VPN](https://www.wireguard.com/)
- [GitHub Actions (Free Tier)](https://docs.github.com/en/actions/learn-github-actions)
- [Hugging Face Spaces](https://huggingface.co/docs/hub/spaces)
- [Decentralized Host](https://github.com/codesbyfebin/decentralized-)

## License

Same as Decentralized Host project.

---

**P1-ENDTOEND-A01 LOCAL**: Real distributed machine failure qualification on your hardware.
Zero cloud cost. Zero credit card. Sovereign P1 laboratory.
