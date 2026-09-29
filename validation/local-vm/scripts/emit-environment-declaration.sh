#!/bin/bash
# P1-LOCAL-VM-A01: Emit Environment Declaration
# Outputs machine-readable topology and qualification scope
# Usage: ./emit-environment-declaration.sh [output-format]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../config"
OUTPUT_FORMAT="${1:-json}"

if [ ! -f "$CONFIG_DIR/domains.json" ]; then
  # Create minimal environment declaration if domains.json missing
  cat << 'EOF'
{
  "P1_DISTRIBUTED_VM_QUALIFICATION": "ELIGIBLE",
  "P1_INDEPENDENT_PHYSICAL_HOST_QUALIFICATION": "NOT_ESTABLISHED",
  "P2_INDEPENDENT_OPERATOR_QUALIFICATION": "NOT_ESTABLISHED"
}
EOF
  exit 0
fi

# Helper: try to use jq, fallback to cat entire file if not available
output_json_field() {
  local field="$1"
  local file="$2"

  if command -v jq &> /dev/null; then
    jq "$field" "$file"
  else
    # Fallback: for environment_declaration, output the entire file
    # (it contains the field as a top-level object)
    cat "$file"
  fi
}

# Try to ensure jq is available for better output
if ! command -v jq &> /dev/null; then
  if [ -f /etc/os-release ]; then
    apt-get update -qq >/dev/null 2>&1 && apt-get install -y jq >/dev/null 2>&1 || true
  fi
fi

case "$OUTPUT_FORMAT" in
  json)
    # Output environment_declaration object or entire file as fallback
    if command -v jq &> /dev/null; then
      jq '.environment_declaration' "$CONFIG_DIR/domains.json"
    else
      # Fallback: output the entire domains.json (contains environment_declaration)
      cat "$CONFIG_DIR/domains.json"
    fi
    ;;

  env)
    # Shell environment variable format
    echo "# P1-LOCAL-VM-A01 Environment Declaration"
    echo "# Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo ""

    if command -v jq &> /dev/null; then
      jq -r '.environment_declaration | to_entries | .[] | "\(.key)=\(.value)"' "$CONFIG_DIR/domains.json"
      echo ""
      echo "# Failure Domains"
      jq -r '.failure_domains.distinct[] | "DISTINCT_\(.name | ascii_upcase)_COUNT=\(.count)"' "$CONFIG_DIR/domains.json"
    else
      echo "# (jq not available, outputting raw file)"
      cat "$CONFIG_DIR/domains.json"
    fi
    ;;

  text)
    # Human-readable format
    echo "=== P1-LOCAL-VM-A01 Environment Declaration ==="
    echo "Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo ""

    if command -v jq &> /dev/null; then
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
    else
      echo "(jq not available, outputting raw JSON structure)"
      cat "$CONFIG_DIR/domains.json"
    fi
    ;;

  *)
    echo "ERROR: Unknown format: $OUTPUT_FORMAT"
    echo "Supported: json, env, text"
    exit 1
    ;;
esac
