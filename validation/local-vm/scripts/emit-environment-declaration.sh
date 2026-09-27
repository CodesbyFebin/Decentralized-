#!/bin/bash
# P1-LOCAL-VM-A01: Emit Environment Declaration
# Outputs machine-readable topology and qualification scope
# Usage: ./emit-environment-declaration.sh [output-format]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../config"
OUTPUT_FORMAT="${1:-json}"

if [ ! -f "$CONFIG_DIR/domains.json" ]; then
  echo "ERROR: domains.json not found"
  exit 1
fi

case "$OUTPUT_FORMAT" in
  json)
    jq '.environment_declaration' "$CONFIG_DIR/domains.json"
    ;;
  
  env)
    # Shell environment variable format
    echo "# P1-LOCAL-VM-A01 Environment Declaration"
    echo "# Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo ""
    jq -r '.environment_declaration | to_entries | .[] | "\(.key)=\(.value)"' "$CONFIG_DIR/domains.json"
    echo ""
    echo "# Failure Domains"
    jq -r '.failure_domains.distinct[] | "DISTINCT_\(.name | ascii_upcase)_COUNT=\(.count)"' "$CONFIG_DIR/domains.json"
    ;;
  
  text)
    # Human-readable format
    echo "=== P1-LOCAL-VM-A01 Environment Declaration ==="
    echo "Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo ""
    echo "Cluster Composition:"
    echo "  Physical Hosts: $(jq -r '.environment_declaration.PHYSICAL_HOSTS' "$CONFIG_DIR/domains.json")"
    echo "  VM Instances: $(jq -r '.environment_declaration.VM_INSTANCES' "$CONFIG_DIR/domains.json")"
    echo "  OS Instances: $(jq -r '.environment_declaration.OS_INSTANCES' "$CONFIG_DIR/domains.json")"
    echo "  Node Identities: $(jq -r '.environment_declaration.NODE_IDENTITIES' "$CONFIG_DIR/domains.json")"
    echo ""
    echo "Distinct Failure Domains:"
    jq -r '.failure_domains.distinct[] | "  - \(.name): \(.count) instances"' "$CONFIG_DIR/domains.json"
    echo ""
    echo "Shared Infrastructure:"
    jq -r '.failure_domains.shared[] | "  - \(.name): \(.value)"' "$CONFIG_DIR/domains.json"
    echo ""
    echo "Qualification Status:"
    echo "  P1 Local VM: $(jq -r '.environment_declaration.P1_DISTRIBUTED_VM_QUALIFICATION' "$CONFIG_DIR/domains.json")"
    echo "  P1 Physical Host: $(jq -r '.environment_declaration.P1_INDEPENDENT_PHYSICAL_HOST_QUALIFICATION' "$CONFIG_DIR/domains.json")"
    echo "  P2 Operator: $(jq -r '.environment_declaration.P2_INDEPENDENT_OPERATOR_QUALIFICATION' "$CONFIG_DIR/domains.json")"
    ;;
  
  *)
    echo "ERROR: Unknown format: $OUTPUT_FORMAT"
    echo "Supported: json, env, text"
    exit 1
    ;;
esac
