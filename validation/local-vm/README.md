# LOCAL-VM-P1-QUALIFICATION-KIT

**P1-ENDTOEND-A01 Distributed Private Cloud Qualification on Your Own Hardware**

Zero-cost, card-free P1 qualification using local VMs on your machine or cheap cloud starter tiers (GitHub Actions, Hugging Face Spaces, Google Cloud Starter).

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

## Topology

### Option A: Local QEMU/UTM/VirtualBox

```
Your Laptop/Desktop
├── dh-local-01: 2 CPU, 4GB RAM
├── dh-local-02: 2 CPU, 4GB RAM
└── dh-local-03: 4 CPU, 8GB RAM (control plane)

Network: 192.168.200.0/24 (host-only or WireGuard)
Latency: <1ms (localhost) to ~50ms (WireGuard simulated)
```

**Resource Requirements:**
- Host: 8+ CPU cores, 16GB+ RAM, 150GB free disk
- Per VM: 2-4 CPU, 4-8GB RAM, 50GB disk
- Total: ~200GB disk, 20GB active RAM

**Time to Provision:** 10-15 minutes (download Ubuntu image + create VMs)

### Option B: GitHub Actions (Free Tier)

```
GitHub Actions Runner
├── 2 CPU, 7GB RAM
├── 150GB disk
└── 3 containers (via Docker)

Network: GitHub's datacenter (<5ms inter-container)
Cost: $0 (free tier 2000 minutes/month)
```

**Constraints:**
- Only 60-minute max job runtime
- Limited to sequential scenarios (no 2-hour baseline test)
- Useful for quick regression testing, not full P1 qualification

### Option C: Hugging Face Spaces (Free Tier + Docker)

```
Hugging Face Spaces
└── 16GB CPU RAM container
    ├── Orchestrate 3x Docker containers
    ├── Collect evidence to HF dataset
    └── Stream results to shared Space

Cost: $0 (free tier)
Runtime: 12-48 hours continuous
```

**Advantages:**
- Persistent storage (HF dataset integration)
- Shareable results via Space UI
- Natural Hugging Face integration for AI workloads

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
