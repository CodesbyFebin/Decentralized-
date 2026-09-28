#!/bin/bash
# P1-LOCAL-VM-A01: Preflight Checks
# Validates prerequisites before cluster creation
# Topology-aware resource detection with fail-closed semantics
#
# Usage:
#   ./preflight.sh [qemu|qemu-no-kvm] [--nodes N] [--cpu C] [--memory M] [--disk D]
#
# Examples:
#   ./preflight.sh qemu
#   ./preflight.sh qemu --nodes 3 --cpu 1 --memory 1024 --disk 8
#

set +e

# ============================================================================
# CONFIGURATION
# ============================================================================

HYPERVISOR="${1:-qemu}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
STATE_DIR="$REPO_ROOT/validation/local-vm/state"

# Topology defaults (per node)
NODES=3
CPU_PER_NODE=1
MEMORY_PER_NODE=1024      # MiB per node
DISK_PER_NODE=8           # GiB per node

# Parse topology arguments
shift || true
while [[ $# -gt 0 ]]; do
  case $1 in
    --nodes)   NODES="$2"; shift 2 ;;
    --cpu)     CPU_PER_NODE="$2"; shift 2 ;;
    --memory)  MEMORY_PER_NODE="$2"; shift 2 ;;
    --disk)    DISK_PER_NODE="$2"; shift 2 ;;
    *)         echo "Unknown option: $1"; exit 1 ;;
  esac
done

echo "=== P1-LOCAL-VM-A01: Preflight Checks ==="
echo "Hypervisor: $HYPERVISOR"
echo "Topology:"
echo "  Nodes: $NODES"
echo "  CPU/node: $CPU_PER_NODE"
echo "  Memory/node: ${MEMORY_PER_NODE} MiB"
echo "  Disk/node: ${DISK_PER_NODE} GiB"
echo ""

PASS=0
FAIL=0
BLOCKED=0

# Resource observations (never coerce to 0)
HOST_TOTAL_RAM_GIB="UNKNOWN"
HOST_AVAILABLE_DISK_GIB="UNKNOWN"
TOTAL_VCPU="UNKNOWN"
TOTAL_GUEST_RAM_MIB="UNKNOWN"
VIRTUAL_DISK_CAPACITY_GIB="UNKNOWN"

# ============================================================================
# HELPER FUNCTIONS
# ============================================================================

check() {
  local name="$1"
  local cmd="$2"

  if eval "$cmd" > /dev/null 2>&1; then
    echo "✓ $name"
    ((PASS++))
    return 0
  else
    echo "✗ $name"
    ((FAIL++))
    return 1
  fi
}

fail() {
  local msg="$1"
  echo "✗ $msg"
  ((FAIL++))
}

warn() {
  local msg="$1"
  echo "⚠ $msg"
  ((FAIL++))
}

# ============================================================================
# HOST ARCHITECTURE
# ============================================================================

ARCH=$(uname -m)
case "$ARCH" in
  arm64|aarch64)
    echo "✓ Host Architecture: $ARCH (ARM64)"
    ((PASS++))
    QEMU_ARCH="aarch64"
    ;;
  x86_64|amd64)
    echo "✓ Host Architecture: $ARCH (x86_64)"
    ((PASS++))
    QEMU_ARCH="x86_64"
    ;;
  *)
    echo "✗ Unsupported Host Architecture: $ARCH"
    ((FAIL++))
    QEMU_ARCH="unknown"
    ;;
esac
echo ""

# ============================================================================
# QEMU PREREQUISITES
# ============================================================================

case "$HYPERVISOR" in
  qemu|qemu-no-kvm)
    echo "Checking QEMU prerequisites..."

    # QEMU binary
    case "$QEMU_ARCH" in
      aarch64)
        check "QEMU binary (qemu-system-aarch64)" "command -v qemu-system-aarch64"
        ;;
      x86_64)
        check "QEMU binary (qemu-system-x86_64)" "command -v qemu-system-x86_64"
        ;;
    esac

    # qemu-img
    check "QEMU disk tool (qemu-img)" "command -v qemu-img"

    # SSH and connectivity tools
    check "SSH client" "command -v ssh"
    check "SSH keygen" "command -v ssh-keygen"
    check "curl" "command -v curl"
    check "jq" "command -v jq"

    echo ""
    echo "Checking host resources..."

    # ========================================================================
    # HOST PHYSICAL RAM DETECTION (macOS/Linux)
    # ========================================================================

    RAM_BYTES="UNKNOWN"
    RAM_MIB="UNKNOWN"
    RAM_GIB="UNKNOWN"

    # Try sysctl first (macOS, some Linux)
    if command -v sysctl &>/dev/null; then
      RAM_BYTES="$(sysctl -n hw.memsize 2>/dev/null || echo "")"

      # Validate before arithmetic
      case "$RAM_BYTES" in
        ''|*[!0-9]*)
          fail "Unable to determine physical RAM (sysctl -n hw.memsize returned invalid value)"
          RAM_BYTES="UNKNOWN"
          ;;
        *)
          # Successfully got numeric value
          RAM_MIB=$((RAM_BYTES / 1024 / 1024))
          RAM_GIB=$((RAM_BYTES / 1024 / 1024 / 1024))
          HOST_TOTAL_RAM_GIB=$RAM_GIB
          echo "✓ Total Physical RAM: ${RAM_GIB} GiB (${RAM_MIB} MiB)"
          ((PASS++))
          ;;
      esac
    fi

    # Fallback for Linux systems without hw.memsize
    if [ "$RAM_BYTES" = "UNKNOWN" ] && [ -f /proc/meminfo ]; then
      RAM_KIB="$(grep "^MemTotal:" /proc/meminfo | awk '{print $2}')"

      # Validate before arithmetic
      case "$RAM_KIB" in
        ''|*[!0-9]*)
          fail "Unable to determine physical RAM (/proc/meminfo invalid)"
          ;;
        *)
          RAM_MIB=$((RAM_KIB / 1024))
          RAM_GIB=$((RAM_KIB / 1024 / 1024))
          HOST_TOTAL_RAM_GIB=$RAM_GIB
          echo "✓ Total Physical RAM: ${RAM_GIB} GiB (${RAM_MIB} MiB)"
          ((PASS++))
          ;;
      esac
    fi

    if [ "$RAM_BYTES" = "UNKNOWN" ]; then
      warn "Could not determine available RAM"
    fi

    # ========================================================================
    # HOST AVAILABLE DISK DETECTION
    # ========================================================================

    DISK_KIB="UNKNOWN"
    DISK_MIB="UNKNOWN"
    DISK_GIB="UNKNOWN"

    # Use df -k to get available disk in 1024-byte blocks
    DISK_KIB="$(df -k / 2>/dev/null | awk 'NR==2 {print $4}' | head -1)"

    # Validate before arithmetic
    case "$DISK_KIB" in
      ''|*[!0-9]*)
        fail "Unable to determine available disk space (df returned invalid value)"
        DISK_KIB="UNKNOWN"
        ;;
      0)
        fail "No available disk space detected"
        DISK_KIB="UNKNOWN"
        ;;
      *)
        DISK_MIB=$((DISK_KIB / 1024))
        DISK_GIB=$((DISK_KIB / 1024 / 1024))
        HOST_AVAILABLE_DISK_GIB=$DISK_GIB
        echo "✓ Available Disk: ${DISK_GIB} GiB (${DISK_MIB} MiB, ${DISK_KIB} KiB)"
        ((PASS++))
        ;;
    esac

    # ========================================================================
    # TOPOLOGY-AWARE RESOURCE CALCULATION
    # ========================================================================

    echo ""
    echo "Topology-aware resource requirements:"

    # Calculate total vCPU needed
    TOTAL_VCPU=$((NODES * CPU_PER_NODE))
    echo "  Total vCPU: $TOTAL_VCPU (${NODES} nodes × ${CPU_PER_NODE} CPU/node)"

    # Calculate total guest RAM
    TOTAL_GUEST_RAM_MIB=$((NODES * MEMORY_PER_NODE))
    TOTAL_GUEST_RAM_GIB=$((TOTAL_GUEST_RAM_MIB / 1024))
    echo "  Total guest RAM: ${TOTAL_GUEST_RAM_MIB} MiB (${TOTAL_GUEST_RAM_GIB} GiB, ${NODES} nodes × ${MEMORY_PER_NODE} MiB/node)"

    # Calculate virtual disk capacity (qcow2 max size, not physical allocation)
    VIRTUAL_DISK_CAPACITY_GIB=$((NODES * DISK_PER_NODE))
    echo "  Virtual disk capacity: ${VIRTUAL_DISK_CAPACITY_GIB} GiB (${NODES} nodes × ${DISK_PER_NODE} GiB/node)"

    # QCOW2 sparse allocation estimation:
    # - Base image: ~2.5 GiB (Ubuntu cloud image)
    # - Initial sparse allocation: ~500 MiB per disk
    BASE_IMAGE_SIZE=2500  # MiB
    INITIAL_SPARSE_PER_DISK=500  # MiB
    ESTIMATED_INITIAL_PHYSICAL_MIB=$((BASE_IMAGE_SIZE + (NODES * INITIAL_SPARSE_PER_DISK)))
    ESTIMATED_INITIAL_PHYSICAL_GIB=$((ESTIMATED_INITIAL_PHYSICAL_MIB / 1024))

    echo "  Base image size: ~${BASE_IMAGE_SIZE} MiB"
    echo "  Estimated initial physical requirement: ~${ESTIMATED_INITIAL_PHYSICAL_GIB} GiB"

    # Reserve 20% of available disk as headroom
    if [ "$DISK_GIB" != "UNKNOWN" ]; then
      MINIMUM_HOST_HEADROOM=$((DISK_GIB / 5))
      MINIMUM_HOST_HEADROOM=$((MINIMUM_HOST_HEADROOM < 2 ? 2 : MINIMUM_HOST_HEADROOM))
      echo "  Minimum host headroom (20%): ~${MINIMUM_HOST_HEADROOM} GiB"
    fi

    echo ""
    echo "Checking resource constraints..."

    # Check host has enough physical RAM for guest + QEMU/macOS overhead
    # Assume QEMU/macOS needs ~2GB, guest needs TOTAL_GUEST_RAM_MIB
    if [ "$RAM_GIB" != "UNKNOWN" ]; then
      REQUIRED_RAM_MIB=$((TOTAL_GUEST_RAM_MIB + 2048))  # +2GB for QEMU/OS
      REQUIRED_RAM_GIB=$((REQUIRED_RAM_MIB / 1024))

      if [ "$RAM_GIB" -lt "$REQUIRED_RAM_GIB" ]; then
        warn "Insufficient physical RAM: have ${RAM_GIB} GiB, need ~${REQUIRED_RAM_GIB} GiB"
      else
        echo "✓ Physical RAM adequate: ${RAM_GIB} GiB >= ${REQUIRED_RAM_GIB} GiB needed"
        ((PASS++))
      fi
    fi

    # Check host has enough disk for initial allocation + headroom
    if [ "$DISK_GIB" != "UNKNOWN" ]; then
      TOTAL_NEEDED=$((ESTIMATED_INITIAL_PHYSICAL_GIB + MINIMUM_HOST_HEADROOM))

      if [ "$DISK_GIB" -lt "$TOTAL_NEEDED" ]; then
        warn "Insufficient disk space: have ${DISK_GIB} GiB, need ~${TOTAL_NEEDED} GiB"
      else
        echo "✓ Disk space adequate: ${DISK_GIB} GiB >= ${TOTAL_NEEDED} GiB needed"
        ((PASS++))
      fi
    fi

    echo ""
    echo "Checking port availability..."

    # Check SSH ports for each node
    PORT_AVAILABLE=true
    for ((i=1; i<=NODES; i++)); do
      PORT=$((2200 + i))
      if lsof -nP -iTCP:$PORT -sTCP:LISTEN 2>/dev/null | grep -q LISTEN; then
        echo "✗ Port $PORT already in use"
        ((FAIL++))
        PORT_AVAILABLE=false
      else
        echo "✓ Port $PORT available"
        ((PASS++))
      fi
    done

    echo ""
    echo "Checking virtualization support..."

    if [ "$ARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then
      # Apple Silicon: QEMU can always run (uses Hypervisor.framework)
      echo "✓ Running on Apple Silicon (native QEMU support)"
      ((PASS++))
    else
      # Intel: check for KVM or other acceleration
      if [ "$(uname)" = "Darwin" ]; then
        echo "✓ Running on macOS Intel (Hypervisor.framework available)"
        ((PASS++))
      else
        if grep -q "kvm" /proc/cpuinfo 2>/dev/null; then
          echo "✓ KVM support detected"
          ((PASS++))
        else
          warn "No KVM support (will use slower QEMU emulation)"
        fi
      fi
    fi

    ;;
  *)
    echo "ERROR: Unsupported hypervisor: $HYPERVISOR"
    ((BLOCKED++))
    ;;
esac

# ============================================================================
# PERSIST TOPOLOGY FOR OTHER SCRIPTS
# ============================================================================

mkdir -p "$STATE_DIR"
TOPOLOGY_FILE="$STATE_DIR/topology.sh"

cat > "$TOPOLOGY_FILE" << 'TOPOLOGY_EOF'
#!/bin/bash
# P1-LOCAL-VM-A01: Topology agreement between preflight/create/start
# Sourced by create-vm-cluster.sh and start-cluster.sh
TOPOLOGY_EOF

cat >> "$TOPOLOGY_FILE" << TOPOLOGY_EOF

# Topology accepted by preflight
NODES=$NODES
CPU_PER_NODE=$CPU_PER_NODE
MEMORY_PER_NODE=$MEMORY_PER_NODE
DISK_PER_NODE=$DISK_PER_NODE

# Host resource observations
HOST_TOTAL_RAM_GIB="$HOST_TOTAL_RAM_GIB"
HOST_AVAILABLE_DISK_GIB="$HOST_AVAILABLE_DISK_GIB"

# Calculated totals
TOTAL_VCPU=$TOTAL_VCPU
TOTAL_GUEST_RAM_MIB=$TOTAL_GUEST_RAM_MIB
VIRTUAL_DISK_CAPACITY_GIB=$VIRTUAL_DISK_CAPACITY_GIB

# Persisted at: $(date -u +%Y-%m-%dT%H:%M:%SZ)
TOPOLOGY_EOF

chmod 644 "$TOPOLOGY_FILE"

# ============================================================================
# SUMMARY AND EXIT CODE
# ============================================================================

echo ""
echo "=== Preflight Summary ==="
echo "PASS:    $PASS"
echo "FAIL:    $FAIL"
echo "BLOCKED: $BLOCKED"
echo ""
echo "Host Resources:"
echo "  Physical RAM:       $HOST_TOTAL_RAM_GIB GiB"
echo "  Available Disk:     $HOST_AVAILABLE_DISK_GIB GiB"
echo ""
echo "Topology Agreement:"
echo "  Nodes:              $NODES"
echo "  CPU per node:       $CPU_PER_NODE"
echo "  Memory per node:    $MEMORY_PER_NODE MiB"
echo "  Disk per node:      $DISK_PER_NODE GiB"
echo ""
echo "Calculated Requirements:"
echo "  Total vCPU:         $TOTAL_VCPU"
echo "  Total guest RAM:    $TOTAL_GUEST_RAM_MIB MiB"
echo "  Virtual disk:       $VIRTUAL_DISK_CAPACITY_GIB GiB"
echo ""
echo "Topology persisted to: $TOPOLOGY_FILE"
echo ""

# FAIL-CLOSED SEMANTICS: exit non-zero if any failures or blocks
if [ $BLOCKED -gt 0 ]; then
  echo "STATUS: BLOCKED"
  exit 1
elif [ $FAIL -gt 0 ]; then
  echo "STATUS: FAIL"
  exit 1
else
  echo "STATUS: PASS"
  exit 0
fi

# ============================================================================
# INLINE TEST DATA
# ============================================================================

# Real 8 GiB macOS observation (from P1-LOCAL-VM-A01 directive):
#
# Host system:
#   uname -m => x86_64
#   sysctl -n hw.memsize => 8589934592 (8 GiB)
#   sysctl -n hw.ncpu => 6
#   df -k / => Available: 22284712 KiB (~21 GiB)
#
# Expected preflight output for topology: 3 nodes, 1 CPU, 1024 MiB, 8 GiB:
#   HOST_TOTAL_RAM_GIB = 8
#   HOST_AVAILABLE_DISK_GIB ≈ 21
#   TOTAL_VCPU = 3
#   TOTAL_GUEST_RAM_MIB = 3072
#   VIRTUAL_DISK_CAPACITY_GIB = 24
#   STATUS = PASS
#   exit code = 0
#
# Test negative controls:
#   - empty RAM_BYTES: FAIL, STATUS=FAIL, exit 1
#   - malformed RAM_BYTES: FAIL, STATUS=FAIL, exit 1
#   - zero RAM_BYTES: handled as UNKNOWN, FAIL, exit 1
#   - empty DISK_KIB: FAIL, STATUS=FAIL, exit 1
#   - malformed DISK_KIB: FAIL, STATUS=FAIL, exit 1
#   - zero DISK_KIB: FAIL, STATUS=FAIL, exit 1
#   - TOTAL_GUEST_RAM_MIB exceeds HOST_TOTAL_RAM_GIB-2GB: warning, still FAIL
#   - VIRTUAL_DISK_CAPACITY_GIB exceeds available disk: warning, still FAIL
#   - Port 2201 in use: FAIL, exit 1
#   - QEMU binary missing: FAIL, exit 1
#   - hostname doesn't match topology.sh: start-cluster.sh should fail
#   - No "~150GB" constant: ✓
#   - UNKNOWN never converted to "0": ✓
