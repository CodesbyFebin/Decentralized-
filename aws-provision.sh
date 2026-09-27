#!/bin/bash
# AWS EC2 Provisioning Script for P1-ENDTOEND-A01
# Usage: ./aws-provision.sh <region> <key-name>
# Example: ./aws-provision.sh us-east-1 p1-e2e-test

set -e

REGION="${1:-us-east-1}"
KEY_NAME="${2:-p1-e2e-test}"
VPC_CIDR="192.168.100.0/24"
SECURITY_GROUP="p1-e2e-test-sg"

echo "=== P1-ENDTOEND-A01: AWS EC2 Provisioning ==="
echo "Region: $REGION"
echo "Key Pair: $KEY_NAME"
echo "VPC CIDR: $VPC_CIDR"
echo ""

# Verify AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo "ERROR: AWS CLI not found. Install with: pip install awscli"
    exit 1
fi

# Verify key pair exists
echo "Verifying key pair..."
if ! aws ec2 describe-key-pairs --key-names $KEY_NAME --region $REGION &>/dev/null; then
    echo "ERROR: Key pair '$KEY_NAME' not found in $REGION"
    echo "Create with: aws ec2 create-key-pair --key-name $KEY_NAME --region $REGION"
    exit 1
fi
echo "✓ Key pair verified"

# 1. Create VPC
echo ""
echo "Creating VPC..."
VPC_OUTPUT=$(aws ec2 create-vpc \
  --cidr-block $VPC_CIDR \
  --region $REGION \
  --tag-specifications "ResourceType=vpc,Tags=[{Key=Name,Value=p1-e2e-test}]")
VPC_ID=$(echo $VPC_OUTPUT | jq -r '.Vpc.VpcId')
echo "✓ VPC created: $VPC_ID"

# Enable DNS hostnames
aws ec2 modify-vpc-attribute \
  --vpc-id $VPC_ID \
  --enable-dns-hostnames \
  --region $REGION

# 2. Create subnet
echo "Creating subnet..."
SUBNET_OUTPUT=$(aws ec2 create-subnet \
  --vpc-id $VPC_ID \
  --cidr-block $VPC_CIDR \
  --region $REGION \
  --tag-specifications "ResourceType=subnet,Tags=[{Key=Name,Value=p1-e2e-test}]")
SUBNET_ID=$(echo $SUBNET_OUTPUT | jq -r '.Subnet.SubnetId')
echo "✓ Subnet created: $SUBNET_ID"

# Enable public IP auto-assign
aws ec2 modify-subnet-attribute \
  --subnet-id $SUBNET_ID \
  --map-public-ip-on-launch \
  --region $REGION

# 3. Create Internet Gateway
echo "Creating Internet Gateway..."
IGW_OUTPUT=$(aws ec2 create-internet-gateway \
  --region $REGION \
  --tag-specifications "ResourceType=internet-gateway,Tags=[{Key=Name,Value=p1-e2e-test}]")
IGW_ID=$(echo $IGW_OUTPUT | jq -r '.InternetGateway.InternetGatewayId')
echo "✓ IGW created: $IGW_ID"

aws ec2 attach-internet-gateway \
  --vpc-id $VPC_ID \
  --internet-gateway-id $IGW_ID \
  --region $REGION

# 4. Create security group
echo "Creating security group..."
SG_OUTPUT=$(aws ec2 create-security-group \
  --group-name $SECURITY_GROUP \
  --description "P1-E2E Test Network" \
  --vpc-id $VPC_ID \
  --region $REGION)
SG_ID=$(echo $SG_OUTPUT | jq -r '.GroupId')
echo "✓ Security group created: $SG_ID"

# Allow SSH from anywhere (restrict to your IP in production)
aws ec2 authorize-security-group-ingress \
  --group-id $SG_ID \
  --protocol tcp \
  --port 22 \
  --cidr 0.0.0.0/0 \
  --region $REGION

# Allow all intra-group traffic
aws ec2 authorize-security-group-ingress \
  --group-id $SG_ID \
  --protocol all \
  --port 0-65535 \
  --source-security-group-id $SG_ID \
  --region $REGION

# 5. Launch instances
echo ""
echo "Launching instances..."

declare -A INSTANCES=(
  ["cp-0"]="t3.large"
  ["provider-a"]="t3.medium"
  ["provider-b"]="t3.medium"
  ["provider-c"]="t3.medium"
  ["observer-0"]="t3.small"
)

declare -A PRIVATE_IPS=(
  ["cp-0"]="192.168.100.10"
  ["provider-a"]="192.168.100.20"
  ["provider-b"]="192.168.100.21"
  ["provider-c"]="192.168.100.22"
  ["observer-0"]="192.168.100.30"
)

# Get latest Ubuntu 22.04 LTS AMI
AMI_ID=$(aws ec2 describe-images \
  --owners 099720109477 \
  --filters "Name=name,Values=ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*" \
  --query "Images[0].ImageId" \
  --output text \
  --region $REGION)

echo "Using AMI: $AMI_ID"

INSTANCE_IDS=()

for NODE in "${!INSTANCES[@]}"; do
  INSTANCE_TYPE="${INSTANCES[$NODE]}"
  PRIVATE_IP="${PRIVATE_IPS[$NODE]}"

  echo "  Launching $NODE ($INSTANCE_TYPE)..."

  INSTANCE_OUTPUT=$(aws ec2 run-instances \
    --image-id $AMI_ID \
    --instance-type $INSTANCE_TYPE \
    --key-name $KEY_NAME \
    --security-group-ids $SG_ID \
    --subnet-id $SUBNET_ID \
    --private-ip-address $PRIVATE_IP \
    --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=$NODE}]" \
    --region $REGION)

  INSTANCE_ID=$(echo $INSTANCE_OUTPUT | jq -r '.Instances[0].InstanceId')
  INSTANCE_IDS+=($INSTANCE_ID)

  echo "    Instance ID: $INSTANCE_ID"
done

# 6. Wait for instances to be running
echo ""
echo "Waiting for instances to be running..."
aws ec2 wait instance-running \
  --instance-ids "${INSTANCE_IDS[@]}" \
  --region $REGION

# Get public IPs and hostnames
echo ""
echo "=== Provisioning Complete ==="
echo ""
echo "Instance Details:"
echo "---"

INSTANCES_JSON=$(aws ec2 describe-instances \
  --instance-ids "${INSTANCE_IDS[@]}" \
  --region $REGION)

echo $INSTANCES_JSON | jq -r '.Reservations[].Instances[] |
  "\(.Tags[] | select(.Key=="Name").Value):
    Instance ID: \(.InstanceId)
    Public IP: \(.PublicIpAddress)
    Private IP: \(.PrivateIpAddress)
    Status: \(.State.Name)"' | sed 's/^/  /'

echo ""
echo "Next Steps:"
echo "1. Wait 2-3 minutes for instances to fully boot"
echo "2. Test SSH access: ssh -i ~/.ssh/$KEY_NAME.pem ubuntu@<PUBLIC_IP>"
echo "3. Run baseline validation:"
echo "   cd Decentralized-"
echo "   bash scripts/baseline-validation.sh"
echo ""
echo "To clean up:"
echo "  bash scripts/cleanup-aws.sh $VPC_ID $REGION"
