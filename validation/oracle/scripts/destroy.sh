#!/bin/bash
# P1-ENDTOEND-A01: Cleanup - Destroy All Infrastructure
# Usage: ./destroy.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TF_DIR="$(dirname "$SCRIPT_DIR")"

echo "=== P1-ENDTOEND-A01: Infrastructure Cleanup ==="
echo "Terraform directory: $TF_DIR"
echo ""
echo "⚠️  WARNING: This will destroy all infrastructure!"
echo "⚠️  This action cannot be undone."
echo ""
read -p "Type 'yes' to confirm destruction: " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
  echo "Cancelled."
  exit 0
fi

cd "$TF_DIR"

echo ""
echo "Destroying infrastructure..."
terraform destroy -auto-approve

echo ""
echo "=== Cleanup Complete ==="
echo "All P1 qualification infrastructure has been destroyed."
echo ""
echo "Evidence files (if backed up) are still available in: evidence/"
