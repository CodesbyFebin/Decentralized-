#!/bin/bash
# P1-LOCAL-VM-A01: Preflight Checks
# Validates prerequisites before cluster creation
# Usage: ./preflight.sh qemu

set -e

HYPERVISOR="${1:-qemu}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== P1-LOCAL-VM-A01: Preflight Checks ==="
echo "Hypervisor: $HYPERVISOR"
echo ""

PASS=0
FAIL=0
BLOCKED=0

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

# Host architecture
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
    echo "Checking disk and memory..."

    # Check available disk space (need ~150GB for 3x50GB VMs + base image)
    DISK_AVAILABLE=$(df -B1 "$HOME" 2>/dev/null | tail -1 | awk '{print $4}')
    DISK_NEEDED=$((150 * 1024 * 1024 * 1024))

    if [ "$DISK_AVAILABLE" -gt "$DISK_NEEDED" ]; then
      DISK_GB=$((DISK_AVAILABLE / 1024 / 1024 / 1024))
      echo "✓ Available Disk: ${DISK_GB}GB (need ~150GB)"
      ((PASS++))
    else
      DISK_GB=$((DISK_AVAILABLE / 1024 / 1024 / 1024))
      echo "⚠ Available Disk: ${DISK_GB}GB (need ~150GB)"
      ((FAIL++))
    fi

    # Check available RAM (need ~12GB for 3x4GB VMs)
    MEMORY_AVAILABLE=$(vm_stat 2>/dev/null | grep "Pages free" | awk '{print $3}' | sed 's/\.//' | awk '{print $1 * 4096}' || echo 0)
    if [ "$MEMORY_AVAILABLE" -gt 0 ]; then
      MEMORY_GB=$((MEMORY_AVAILABLE / 1024 / 1024 / 1024))
      echo "✓ Available RAM: ${MEMORY_GB}GB"
      ((PASS++))
    else
      echo "⚠ Could not determine available RAM (be sure you have ~12GB)"
      ((FAIL++))
    fi

    echo ""
    echo "Checking port availability..."

    # Check SSH ports 2201-2203
    for port in 2201 2202 2203; do
      if ! lsof -nP -iTCP:$port -sTCP:LISTEN 2>/dev/null | grep -q LISTEN; then
        echo "✓ Port $port available"
        ((PASS++))
      else
        echo "✗ Port $port already in use"
        ((FAIL++))
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
          echo "⚠ No KVM support (will use slower QEMU emulation)"
          ((FAIL++))
        fi
      fi
    fi

    ;;
  *)
    echo "ERROR: Unsupported hypervisor: $HYPERVISOR"
    ((BLOCKED++))
    ;;
esac

echo ""
echo "=== Preflight Summary ==="
echo "PASS:    $PASS"
echo "FAIL:    $FAIL"
echo "BLOCKED: $BLOCKED"
echo ""

if [ $BLOCKED -gt 0 ]; then
  echo "STATUS: BLOCKED"
  exit 1
elif [ $FAIL -gt 0 ]; then
  echo "STATUS: ISSUES FOUND (may proceed with caution)"
  exit 0
else
  echo "STATUS: PASS"
  exit 0
fi
