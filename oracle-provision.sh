#!/bin/bash
# Oracle Cloud Provisioning Script for P1-ENDTOEND-A01
# Uses OCI CLI to provision infrastructure on Oracle Cloud Free Tier
# Usage: ./oracle-provision.sh <compartment-id> <availability-domain>
# Example: ./oracle-provision.sh ocid1.compartment... us-phoenix-1

set -e

COMPARTMENT_ID="${1}"
AVAILABILITY_DOMAIN="${2:-us-phoenix-1}"
REGION="us-phoenix-1"
SUBNET_CIDR="192.168.100.0/24"
VCN_CIDR="192.168.0.0/16"

if [ -z "$COMPARTMENT_ID" ]; then
  echo "ERROR: Compartment ID required"
  echo "Usage: $0 <compartment-id> [availability-domain]"
  echo ""
  echo "Find compartment ID:"
  echo "  oci iam compartment list --all"
  exit 1
fi

echo "=== P1-ENDTOEND-A01: Oracle Cloud Provisioning ==="
echo "Compartment: $COMPARTMENT_ID"
echo "Availability Domain: $AVAILABILITY_DOMAIN"
echo "Region: $REGION"
echo ""

# Verify OCI CLI is installed
if ! command -v oci &> /dev/null; then
    echo "ERROR: OCI CLI not found. Install with: pip install oci-cli"
    exit 1
fi

# Verify OCI authentication
if ! oci iam compartment get --compartment-id "$COMPARTMENT_ID" &>/dev/null; then
    echo "ERROR: Cannot access compartment. Verify OCI credentials:"
    echo "  oci setup config"
    exit 1
fi

echo "✓ OCI credentials verified"
echo ""

# 1. Create VCN (Virtual Cloud Network)
echo "Creating VCN..."
VCN_OUTPUT=$(oci network vcn create \
  --compartment-id "$COMPARTMENT_ID" \
  --cidr-block "$VCN_CIDR" \
  --display-name "p1-e2e-test" \
  --wait-for-state AVAILABLE \
  --query 'data.id' \
  --output text)
VCN_ID="$VCN_OUTPUT"
echo "✓ VCN created: $VCN_ID"

# 2. Create Internet Gateway
echo "Creating Internet Gateway..."
IGW_OUTPUT=$(oci network internet-gateway create \
  --compartment-id "$COMPARTMENT_ID" \
  --vcn-id "$VCN_ID" \
  --is-enabled true \
  --display-name "p1-e2e-igw" \
  --wait-for-state AVAILABLE \
  --query 'data.id' \
  --output text)
IGW_ID="$IGW_OUTPUT"
echo "✓ IGW created: $IGW_ID"

# 3. Create subnet
echo "Creating subnet..."
SUBNET_OUTPUT=$(oci network subnet create \
  --compartment-id "$COMPARTMENT_ID" \
  --vcn-id "$VCN_ID" \
  --cidr-block "$SUBNET_CIDR" \
  --availability-domain "$AVAILABILITY_DOMAIN" \
  --display-name "p1-e2e-subnet" \
  --wait-for-state AVAILABLE \
  --query 'data.id' \
  --output text)
SUBNET_ID="$SUBNET_OUTPUT"
echo "✓ Subnet created: $SUBNET_ID"

# 4. Update route table to use IGW
echo "Updating route table..."
RT_OUTPUT=$(oci network vcn list --compartment-id "$COMPARTMENT_ID" --vcn-id "$VCN_ID" --query 'data[0].default-route-table-id' --output text 2>/dev/null || echo "")

if [ -n "$RT_OUTPUT" ] && [ "$RT_OUTPUT" != "None" ]; then
  oci network route-table update \
    --rt-id "$RT_OUTPUT" \
    --route-rules "[{\"destination\":\"0.0.0.0/0\",\"destinationCidrBlock\":\"0.0.0.0/0\",\"networkEntityId\":\"$IGW_ID\"}]" \
    --force --wait-for-state AVAILABLE 2>/dev/null || true
fi

# 5. Create security group
echo "Creating security group..."
NSG_OUTPUT=$(oci network network-security-group create \
  --compartment-id "$COMPARTMENT_ID" \
  --vcn-id "$VCN_ID" \
  --display-name "p1-e2e-nsg" \
  --wait-for-state AVAILABLE \
  --query 'data.id' \
  --output text)
NSG_ID="$NSG_OUTPUT"
echo "✓ Security group created: $NSG_ID"

# Add ingress rules: SSH (port 22)
echo "Adding SSH rule..."
oci network network-security-group-security-rule add \
  --network-security-group-id "$NSG_ID" \
  --protocol 6 \
  --source 0.0.0.0/0 \
  --source-type CIDR_BLOCK \
  --tcp-options '{"destinationPortRange":{"min":22,"max":22}}' \
  --direction INGRESS \
  --wait-for-state AVAILABLE 2>/dev/null || true

# Add ingress rules: All traffic within subnet
echo "Adding intra-subnet rules..."
oci network network-security-group-security-rule add \
  --network-security-group-id "$NSG_ID" \
  --protocol all \
  --source "$SUBNET_CIDR" \
  --source-type CIDR_BLOCK \
  --direction INGRESS \
  --wait-for-state AVAILABLE 2>/dev/null || true

# 6. Get available images
echo ""
echo "Fetching Ubuntu 22.04 LTS image..."
IMAGE_ID=$(oci compute image list \
  --compartment-id "$COMPARTMENT_ID" \
  --shape-filter OS=Ubuntu \
  --query "data[?contains(\"display-name\",'Ubuntu-22.04')] | [0].id" \
  --output text)

if [ -z "$IMAGE_ID" ] || [ "$IMAGE_ID" = "None" ]; then
  # Fallback to first available Ubuntu image
  IMAGE_ID=$(oci compute image list \
    --compartment-id "$COMPARTMENT_ID" \
    --query "data[?contains(\"display-name\",'Ubuntu')] | [0].id" \
    --output text)
fi

echo "Using image: $IMAGE_ID"

# 7. Create instances
echo ""
echo "Launching instances..."

# Define instances with specs
declare -A INSTANCES=(
  ["cp-0"]="VM.Standard.A1.Flex"
  ["provider-a"]="VM.Standard.A1.Flex"
  ["provider-b"]="VM.Standard.A1.Flex"
  ["provider-c"]="VM.Standard.A1.Flex"
  ["observer-0"]="VM.Standard.A1.Flex"
)

declare -A PRIVATE_IPS=(
  ["cp-0"]="192.168.100.10"
  ["provider-a"]="192.168.100.20"
  ["provider-b"]="192.168.100.21"
  ["provider-c"]="192.168.100.22"
  ["observer-0"]="192.168.100.30"
)

declare -A CPU_SPECS=(
  ["cp-0"]="4"
  ["provider-a"]="2"
  ["provider-b"]="2"
  ["provider-c"]="2"
  ["observer-0"]="2"
)

declare -A MEM_SPECS=(
  ["cp-0"]="24"
  ["provider-a"]="12"
  ["provider-b"]="12"
  ["provider-c"]="12"
  ["observer-0"]="8"
)

INSTANCE_IDS=()

for NODE in "${!INSTANCES[@]}"; do
  SHAPE="${INSTANCES[$NODE]}"
  PRIVATE_IP="${PRIVATE_IPS[$NODE]}"
  OCPUS="${CPU_SPECS[$NODE]}"
  MEMORY="${MEM_SPECS[$NODE]}"

  echo "  Launching $NODE (${OCPUS} OCPUs, ${MEMORY}GB RAM)..."

  # Create VNIC details
  VNIC_JSON=$(cat <<EOF
{
  "subnetId": "$SUBNET_ID",
  "privateIp": "$PRIVATE_IP",
  "displayName": "$NODE-vnic",
  "nsgIds": ["$NSG_ID"],
  "assignPublicIp": true
}
EOF
)

  # Launch instance
  INSTANCE_OUTPUT=$(oci compute instance launch \
    --availability-domain "$AVAILABILITY_DOMAIN" \
    --compartment-id "$COMPARTMENT_ID" \
    --shape "$SHAPE" \
    --shape-config "{\"ocpus\":$OCPUS,\"memory\":$MEMORY}" \
    --image-id "$IMAGE_ID" \
    --display-name "$NODE" \
    --vnic-details "$VNIC_JSON" \
    --metadata "{\"ssh_authorized_keys\":\"$(cat ~/.ssh/id_rsa.pub 2>/dev/null || echo 'ADD_YOUR_PUBLIC_KEY')\"}" \
    --wait-for-state RUNNING \
    --query 'data.id' \
    --output text 2>/dev/null || echo "")

  if [ -n "$INSTANCE_OUTPUT" ] && [ "$INSTANCE_OUTPUT" != "None" ]; then
    INSTANCE_IDS+=("$INSTANCE_OUTPUT")
    echo "    Instance ID: $INSTANCE_OUTPUT"
  else
    echo "    WARNING: Instance creation may have failed or is pending"
  fi
done

# 8. Wait for instances and collect details
echo ""
echo "=== Provisioning Complete ==="
echo ""
echo "Instance Details:"
echo "---"

for INSTANCE_ID in "${INSTANCE_IDS[@]}"; do
  if [ -n "$INSTANCE_ID" ] && [ "$INSTANCE_ID" != "None" ]; then
    INSTANCE_JSON=$(oci compute instance get --instance-id "$INSTANCE_ID" --query 'data' 2>/dev/null || echo "{}")

    DISPLAY_NAME=$(echo "$INSTANCE_JSON" | jq -r '.display_name // "unknown"')
    STATE=$(echo "$INSTANCE_JSON" | jq -r '.lifecycle_state // "unknown"')

    # Get public IP from VNIC
    VNIC_ID=$(echo "$INSTANCE_JSON" | jq -r '.primary_vnic_id // ""')
    if [ -n "$VNIC_ID" ]; then
      VNIC_JSON=$(oci network vnic get --vnic-id "$VNIC_ID" 2>/dev/null || echo "{}")
      PUBLIC_IP=$(echo "$VNIC_JSON" | jq -r '.public_ip // "pending"')
      PRIVATE_IP=$(echo "$VNIC_JSON" | jq -r '.private_ip // "unknown"')
    else
      PUBLIC_IP="pending"
      PRIVATE_IP="unknown"
    fi

    echo "  $DISPLAY_NAME:"
    echo "    Instance ID: $INSTANCE_ID"
    echo "    Public IP: $PUBLIC_IP"
    echo "    Private IP: $PRIVATE_IP"
    echo "    Status: $STATE"
  fi
done

echo ""
echo "Next Steps:"
echo "1. Wait 2-3 minutes for instances to fully boot"
echo "2. Add public key to instances (if not auto-injected):"
echo "   oci compute instance launch --metadata '{\"ssh_authorized_keys\":\"<pubkey>\"}'"
echo "3. Test SSH access: ssh -i ~/.ssh/id_rsa ubuntu@<PUBLIC_IP>"
echo "4. Run baseline validation:"
echo "   cd Decentralized-"
echo "   bash baseline-validation.sh"
echo ""
echo "VCN ID: $VCN_ID"
echo "Subnet ID: $SUBNET_ID"
echo "NSG ID: $NSG_ID"
echo ""
echo "To clean up:"
echo "  oci network vcn delete --vcn-id $VCN_ID --force"
