#!/bin/bash
# P1-ENDTOEND-A01: Oracle Cloud Terraform Provisioning
# Usage: ./provision.sh <terraform-vars-file>

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TF_DIR="$(dirname "$SCRIPT_DIR")"
VARS_FILE="${1:-terraform.tfvars}"

if [ ! -f "$VARS_FILE" ]; then
  echo "ERROR: Variables file not found: $VARS_FILE"
  echo "Usage: $0 <terraform-vars-file>"
  echo ""
  echo "Example terraform.tfvars:"
  echo "  tenancy_ocid       = \"ocid1.tenancy...\""
  echo "  user_ocid          = \"ocid1.user...\""
  echo "  fingerprint        = \"...\""
  echo "  private_key        = file(\"~/.oci/id_rsa\")"
  echo "  compartment_ocid   = \"ocid1.compartment...\""
  echo "  availability_domain = \"us-phoenix-1-AD-1\""
  echo "  ssh_public_key     = file(\"~/.ssh/id_rsa.pub\")"
  exit 1
fi

echo "=== P1-ENDTOEND-A01: Oracle Cloud Terraform Provisioning ==="
echo "Terraform directory: $TF_DIR"
echo "Variables file: $VARS_FILE"
echo ""

cd "$TF_DIR"

# Initialize Terraform
echo "Initializing Terraform..."
terraform init

# Validate configuration
echo ""
echo "Validating Terraform configuration..."
terraform validate

# Plan provisioning
echo ""
echo "Planning infrastructure..."
terraform plan -var-file="$VARS_FILE" -out=tfplan

# Apply provisioning
echo ""
echo "Applying infrastructure (provisioning ~5-10 minutes)..."
terraform apply tfplan

# Collect outputs
echo ""
echo "=== Provisioning Complete ==="
terraform output -json > inventory.json

echo ""
echo "Node details written to: inventory.json"
echo ""
terraform output node_details

# Save inventory for other scripts
echo ""
echo "Generating inventory for operational scripts..."
terraform output -json p1_test_inventory | jq . > p1-inventory.json
echo "Inventory saved to: p1-inventory.json"

echo ""
echo "Next steps:"
echo "1. Bootstrap nodes: bash bootstrap-nodes.sh"
echo "2. Qualify network: bash qualify-network.sh"
echo "3. Run P1 scenarios: (from repository root)"
