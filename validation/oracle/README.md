# OCI-P1-QUALIFICATION-KIT

**P1-ENDTOEND-A01 Distributed Private Cloud Qualification on Oracle Cloud Free Tier**

Complete Infrastructure-as-Code kit for provisioning and testing Decentralized Host on Oracle Cloud Always Free resources.

## What's Inside

- **Terraform**: Infrastructure provisioning for 3-node cluster on Always Free instances
- **Cloud-init**: Automatic node bootstrap with baseline/network evidence collection
- **Orchestration Scripts**: Failure injection, evidence collection, network qualification
- **Evidence Collection**: Real machine failure detection and workload reschedule validation

## Quick Start

### 1. Prerequisites

```bash
# Install Terraform
brew install terraform  # macOS
# or: Download from https://www.terraform.io/downloads

# Install OCI CLI
pip install oci-cli
oci setup config

# Get your OCI credentials
# From OCI Console → Identity & Security → Users → Your User
# - User OCID
# - Tenancy OCID
# - API Key (create new → download private key, get fingerprint)

# Set up SSH key
ssh-keygen -t rsa -b 4096 -f ~/.ssh/p1-oracle -N ""
```

### 2. Create terraform.tfvars

```bash
cd validation/oracle

cat > terraform.tfvars << 'EOF'
# OCI Credentials (get from OCI Console)
tenancy_ocid = "ocid1.tenancy.oc1..aaaaaaaaexample"
user_ocid    = "ocid1.user.oc1..aaaaaaaaexample"
fingerprint  = "aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99"
private_key  = file("~/.oci/id_rsa")

# Compartment and Region
compartment_ocid   = "ocid1.compartment.oc1..aaaaaaaaexample"
availability_domain = "us-phoenix-1-AD-1"  # Change to your region
region             = "us-phoenix-1"

# SSH Access
ssh_public_key = file("~/.ssh/p1-oracle.pub")
EOF
```

### 3. Provision Infrastructure

```bash
# Initialize Terraform
terraform init

# Review what will be created
terraform plan -var-file=terraform.tfvars

# Provision 3 nodes (5-10 minutes)
bash scripts/provision.sh terraform.tfvars
```

**Topology (Always Free):**
- `dh-node-01`: 1 OCPU, 6GB RAM
- `dh-node-02`: 1 OCPU, 6GB RAM  
- `dh-node-03`: 2 OCPU, 12GB RAM
- **Total**: 4 OCPU, 24GB RAM (all within Always Free pool)

### 4. Bootstrap Nodes

```bash
# Verify SSH access and baseline state
bash scripts/bootstrap-nodes.sh

# Qualify network (latency, NTP, connectivity)
bash scripts/qualify-network.sh
```

### 5. Deploy DHP Software

```bash
# Build binaries
cd ../..  # Repository root
make clean && make build

# Copy binaries to each node (manual or via Terraform)
# Then start DHP agents
```

### 6. Run P1-E2E Scenarios

From the repository root, run the qualification test:

```bash
bash tools/p1-e2e/orchestrator/run-all-scenarios.sh
```

**Scenarios:**
1. **Baseline** (30m): Healthy cluster, evidence collection
2. **Node Failure** (60m): Failure detection <30s, reschedule <120s
3. **Network Partition** (45m): Byzantine resilience, quorum maintained
4. **Storage Failure** (40m): Replica recovery <120s
5. **Control Plane Restart** (30m): State persistence, workload recovery

### 7. Collect Evidence & Seal P1

```bash
# Collect evidence from all nodes
bash scripts/collect-evidence.sh ../evidence

# Validate signatures
cd tools/p1-e2e/validator
go build -o validator
./validator -dir ../../evidence -out validation-report.json

# Generate report
bash tools/p1-e2e/reporting/generate-report.sh scenario-*
```

### 8. Cleanup

```bash
cd validation/oracle
bash scripts/destroy.sh
```

## File Structure

```
validation/oracle/
├── README.md                    # This file
├── versions.tf                  # Terraform version requirements
├── provider.tf                  # OCI provider config
├── variables.tf                 # Input variables
├── data.tf                      # Data sources (images, domains)
├── network.tf                   # VCN, subnet, routing
├── nsg.tf                       # Network security groups
├── compute.tf                   # 3-node instances
├── storage.tf                   # Block storage volumes
├── outputs.tf                   # Outputs (IPs, inventory)
├── cloud-init/
│   └── dh-node.yaml            # Node bootstrap template
├── scripts/
│   ├── provision.sh            # Deploy infrastructure
│   ├── bootstrap-nodes.sh       # Verify SSH, collect baseline
│   ├── qualify-network.sh       # Latency, NTP, connectivity checks
│   ├── kill-node.sh             # Stop instance (failure injection)
│   ├── restore-node.sh          # Start instance (recovery)
│   ├── collect-evidence.sh      # Gather evidence from all nodes
│   └── destroy.sh               # Cleanup infrastructure
└── evidence/                    # Evidence collection output
    └── .gitkeep
```

## Machine Domain Classification

The P1 qualification on OCI records these domain classifications:

```json
{
  "machineDomain":   "distinct",    // 3 separate OCI VMs
  "vmDomain":        "distinct",    // 3 different VM instances
  "operatorDomain":  "SAME/OCI",    // Same OCI operator
  "cloudDomain":     "SAME/OCI",    // Same OCI cloud
  "regionDomain":    "measured",    // us-phoenix-1 / us-ashburn-1
  "ADDomain":        "measured",    // Availability Domain
  "networkDomain":   "measured",    // VCN latency <50ms
  "powerDomain":     "UNKNOWN"      // Not evidenced in free tier
}
```

**Important**: This is **NOT** a multi-provider test. It qualifies:
- ✓ Real machine failure boundaries (3 distinct VMs)
- ✓ Real node-level failure detection (<30s)
- ✓ Real workload rescheduling (<120s)
- ✗ Multi-provider diversity (use for P2 research)

## Failure Injection

To trigger a real node failure during P1 scenarios:

```bash
# Stop a node (triggers failure detection)
bash scripts/kill-node.sh dh-node-02

# Monitor failure detection (~30 seconds)
watch -n1 'curl http://cp-0:8080/nodes | jq'

# Verify reschedule happens (~120 seconds)
watch -n1 'curl http://cp-0:8080/workloads | jq'

# Restore node
bash scripts/restore-node.sh dh-node-02
```

## Cost

**OCI Free Tier**: $0/month for this topology
- 2 x AMD instances (64 GB/month credit if used)
- Unlimited Arm instances (this kit uses 3)
- 100GB block storage (monthly free)

**Estimated Charges**: $0 for Always Free resources

## Troubleshooting

### Terraform Apply Fails

```bash
# Verify OCI CLI credentials
oci iam compartment list --all

# Check that compartment exists
oci iam compartment get --compartment-id <your-compartment>

# Verify Terraform can authenticate
terraform validate
```

### SSH Access Denied

```bash
# Check SSH key permissions
chmod 600 ~/.ssh/p1-oracle

# Verify key in Terraform
cat terraform.tfvars | grep ssh_public_key

# Test connection
ssh -i ~/.ssh/p1-oracle ubuntu@<public-ip>
```

### Network Latency >50ms

- Ensure all instances are in same availability domain
- Use same region for all nodes
- Verify security group allows all inter-subnet traffic

### Nodes Won't Start DHP Agent

```bash
# Check if binaries are present
ssh ubuntu@<node-ip> ls -la /opt/decentralized-host/

# Check boot logs
ssh ubuntu@<node-ip> journalctl -xe | grep dhp

# Check baseline evidence was collected
ssh ubuntu@<node-ip> cat /var/log/decentralized-host/baseline-evidence.json
```

## References

- [Oracle Free Tier](https://github.com/oracle/free)
- [OCI CLI Documentation](https://github.com/oracle/oci-cli)
- [OCI Terraform Provider](https://registry.terraform.io/providers/oracle/oci/latest)
- [Oracle Quick Start Templates](https://github.com/oracle-quickstart/oci-quickstart-template)

## License

Same as Decentralized Host project.

---

**P1-ENDTOEND-A01**: Real distributed machine failure qualification on Always Free compute.
Real machines. Real network. Real failure detection. Real workload rescheduling.
