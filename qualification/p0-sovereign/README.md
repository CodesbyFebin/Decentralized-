# P0-SOVEREIGN-A01 Qualification Kit

## Overview

This directory contains the complete qualification campaign kit for P0-SOVEREIGN-A01: Clean-Linux operational qualification of Decentralized.Host in a single-node production scenario.

**Purpose:** Prove that the product can be installed on a clean Linux machine, enrolled, and run a complete workload lifecycle with real isolation, secrets, storage, and network integration—without external dependencies (wallet, blockchain, marketplace, DePIN).

**Duration:** 10-14 days of continuous testing on a single real node

**Outcome:** evidence/ directory with complete audit trail, independent verification, and final qualification tag

## Quick Start

### Prerequisites

1. **Clean Linux Machine**
   - Ubuntu 22.04 LTS, AlmaLinux 9, Debian 12, or similar
   - Never used for development or testing
   - 4+ CPU cores, 8+ GB RAM, 100+ GB free disk
   - Network connectivity to control plane (TCP 25001 by default)
   - No development tools pre-installed (Go, Node, Docker, etc.)

2. **Control Plane**
   - Running and accessible on network
   - Bootstrap token or join token available
   - Operator approval process ready

3. **Fixture Application**
   - Deterministic app that tests full lifecycle
   - Exposes /health and /state endpoints
   - Reads a delivered secret (without returning it)
   - Writes marker to persistent storage
   - Listens for network ingress

### Installation

```bash
# On clean Linux node:

# 1. Download qualification kit (or clone decentralized.host repo)
git clone https://github.com/codesbyfebin/decentralized-.git
cd decentralized-/qualification/p0-sovereign

# 2. Run preflight checks
./preflight.sh --collect > environment.json

# 3. Review preflight output
cat environment.json
# Verify: Linux distribution, kernel, CPU, RAM, storage, network

# 4. Run installation
./install.sh \
  --name "node-qual-01" \
  --region "local" \
  --join "dhjoin1..." \
  --control-plane "control-plane-ip:25001"

# 5. Begin qualification campaign
./run.sh --campaign full --output ./evidence/

# 6. Monitor progress
tail -f ./evidence/campaign.log

# 7. Run chaos scenarios
./chaos.sh --scenario restart-workload
./chaos.sh --scenario restart-agent
./chaos.sh --scenario restart-control-plane
./chaos.sh --scenario reboot-host

# 8. Test recovery
./recovery.sh --verify-state

# 9. Export state
./export-restore.sh --export --output ./state-backup.tar.gz

# 10. Collect evidence
./collect-evidence.sh --output evidence/

# 11. Verify evidence
./verify-evidence.sh --input evidence/

```

## Scripts

### preflight.sh
**Purpose:** Pre-qualification environment checks

**Usage:**
```bash
./preflight.sh [--check all|distribution|kernel|hardware|network|permissions]
./preflight.sh --collect  # Collect all data and output JSON
./preflight.sh --quiet    # Only exit code, no output
```

**Output:** environment.json
```json
{
  "distribution": "Ubuntu 22.04 LTS",
  "kernel": "6.1.0-1234-generic",
  "architecture": "x86_64",
  "cpu": {"cores": 4, "threads": 8, "model": "..."},
  "ram": {"total_bytes": 17179869184, "swap_bytes": 0},
  "storage": {"/": {"total": 1099511627776, "free": 549755813888}},
  "network": {"interfaces": [...], "connectivity": "ok"},
  "cgroups": "v2",
  "namespaces": ["pid", "network", "mount", "ipc", "uts", "user"],
  "wireguard": "kernel_module" | "userspace",
  "selinux": "enforcing" | "permissive" | "disabled",
  "apparmor": "enabled" | "disabled",
  "seccomp": "supported",
  "timestamp": "2026-09-26T13:30:00Z"
}
```

**Exit Codes:**
- 0: All checks passed
- 1: Warning (soft failure, likely okay)
- 2: Failure (qualification cannot proceed)

### install.sh
**Purpose:** Install decentralized.host node agent on clean system

**Usage:**
```bash
./install.sh \
  --name "node-name" \
  --region "region" \
  --zone "zone" \
  --join "dhjoin1..." \
  --control-plane "ip:port" \
  --data-dir "/data/node" \
  --mesh-advertise "node-external-ip:51820"
```

**Actions:**
1. Download or build binaries (dh-noded, dh-chaos, etc.)
2. Verify signatures/checksums
3. Create data directory
4. Generate local identity
5. Create systemd service or init script
6. Start agent

**Output:** installation.json
```json
{
  "binaries": {
    "dh-noded": {"path": "...", "digest": "b3:..."},
    ...
  },
  "config": {"data_dir": "/data/node", "join_token": "dhjoin1...", ...},
  "identity": {"node_id": "dh1...", "public_key": "..."},
  "service": {"type": "systemd", "name": "dh-noded"},
  "start_time": "2026-09-26T13:35:00Z",
  "first_heartbeat": "2026-09-26T13:35:15Z"
}
```

### run.sh
**Purpose:** Execute qualification campaign phases

**Usage:**
```bash
./run.sh --campaign full
./run.sh --campaign phase-1-4  # Phases 1-4 only (faster testing)
./run.sh --campaign phase-enrollment
./run.sh --campaign phase-deployment
```

**Campaign Phases:**
1. Enrollment & approval
2. Hardware discovery
3. Application deployment
4. Secrets delivery
5. Persistent storage
6. Network & ingress
7. Chaos & recovery
8. Isolation verification
9. Export & restore
10. Zero-dependency check

**Output:** campaign.log, campaign.json

### chaos.sh
**Purpose:** Inject faults during campaign

**Usage:**
```bash
./chaos.sh --scenario restart-workload
./chaos.sh --scenario restart-agent
./chaos.sh --scenario restart-control-plane
./chaos.sh --scenario network-partition --duration 30s
./chaos.sh --scenario cpu-throttle --limit 50%
./chaos.sh --scenario memory-pressure --fill 80%
./chaos.sh --scenario reboot-host
```

**Scenarios:**
- restart-workload: Stop and restart application
- restart-agent: Stop and restart dh-noded
- restart-control-plane: Disconnect and reconnect control plane
- network-partition: Block network for N seconds
- cpu-throttle: Limit CPU usage to X%
- memory-pressure: Fill X% of available RAM
- disk-pressure: Fill X% of disk
- reboot-host: Full system reboot

**Output:** chaos.log

### recovery.sh
**Purpose:** Verify recovery after chaos/faults

**Usage:**
```bash
./recovery.sh --verify-state
./recovery.sh --verify-workload
./recovery.sh --verify-network
./recovery.sh --all
```

**Checks:**
- Agent identity preserved
- Agent reconnects and reconciles
- Workload desired state recovered
- Persistent volume data intact
- Secret lifecycle behavior correct
- Network & ingress operational
- Audit trail coherent

**Output:** recovery.log, recovery-results.json

### export-restore.sh
**Purpose:** Export and restore node state

**Usage:**
```bash
./export-restore.sh --export --output state.tar.gz

# Later:
# On new clean node:
./export-restore.sh --restore --input state.tar.gz
```

**Export Contents:**
- Node identity
- Policy
- Workload manifests (not applications)
- Volume snapshots
- Network configuration
- Audit ledger
- NO plaintext secrets
- NO plaintext application code

**Restrictions:**
- Workload code must be re-pulled from source
- Secrets must be re-delivered per policy
- External network must be reconfigured

**Output:** state.tar.gz, export-manifest.json

### collect-evidence.sh
**Purpose:** Gather all evidence artifacts

**Usage:**
```bash
./collect-evidence.sh --output evidence/
./collect-evidence.sh --phases 1-4   # Only specific phases
./collect-evidence.sh --from-date "2026-09-26"
```

**Collected Evidence:**
- Logs (control plane, agent, workload)
- Manifests (node, app, volumes, policies)
- Signatures (enrollments, observations, operations)
- Metrics (timings, resource usage)
- Audit trail (all operations with evidence)
- Test results (pass/fail for each scenario)
- Hardware facts (preflight output)

**Output:** evidence/ directory with evidence.json manifest

### verify-evidence.sh
**Purpose:** Verify evidence integrity and completeness

**Usage:**
```bash
./verify-evidence.sh --input evidence/
./verify-evidence.sh --input evidence/ --strict
./verify-evidence.sh --input evidence/ --verify-signatures
```

**Verifications:**
- Manifest integrity (all files present)
- File hashes (SHA256 of all content)
- Signatures (if cryptographic evidence present)
- No plaintext secrets (grep scan)
- Completeness (all phases represented)
- Coherence (no timestamp gaps, sequence numbers)
- Determinism (reproducible digests)

**Output:** verification-report.json
```json
{
  "status": "PASS" | "FAIL" | "WARNINGS",
  "manifest_integrity": true,
  "file_hashes_verified": 123,
  "signatures_verified": 45,
  "secrets_scan": {"plaintext_found": false},
  "completeness": {"phases": [1,2,3,4,5,6,7,8,9,10], "all_present": true},
  "coherence": {"gaps": [], "sequence_errors": []},
  "timestamp": "2026-09-26T14:00:00Z",
  "warnings": []
}
```

## Campaign Structure

### Directory Layout

```
qualification/p0-sovereign/
├── README.md                 # This file
├── preflight.sh             # Environment checks
├── install.sh               # Installation
├── run.sh                   # Main campaign
├── chaos.sh                 # Fault injection
├── recovery.sh              # Recovery verification
├── export-restore.sh        # State management
├── collect-evidence.sh      # Evidence gathering
├── verify-evidence.sh       # Evidence verification
├── fixtures/                # Test applications
│   ├── fixture-app/         # Sample application manifest
│   ├── Dockerfile
│   └── main.go
├── policies/                # Resource & network policies
│   ├── default-policy.json
│   └── isolation-policy.json
└── evidence/                # Qualification evidence (generated)
    ├── environment.json
    ├── installation.json
    ├── campaign.json
    ├── chaos.log
    ├── recovery.json
    ├── export-manifest.json
    ├── evidence-manifest.json
    └── verification-report.json
```

### Timeline

**Day 1: Preflight & Installation**
```
Morning: preflight.sh → environment.json
        install.sh → installation.json
        Start agent, wait for heartbeat
Afternoon: Enroll and get operator approval
```

**Day 2: Deployment & Discovery**
```
Morning: Deploy fixture application
         Verify health checks passing
Afternoon: Collect hardware discovery facts
           Verify facts vs. preflight
```

**Day 3-4: Features & Recovery**
```
Day 3: Secrets delivery → verify unreadability
       Persistent storage → write marker → restart → verify
Day 4: Network & ingress → connections tested
       Chaos scenario 1: restart workload → verify recovery
```

**Day 5-7: Advanced Scenarios**
```
Day 5: Chaos scenario 2: restart agent → verify reconciliation
       Chaos scenario 3: network partition → offline detection
Day 6: Chaos scenario 4: full reboot → verify identity preserved
Day 7: Isolation testing → verify all denials logged
```

**Day 8-9: Export & Verification**
```
Day 8: export-restore.sh --export
       Verify export manifest
       No plaintext secrets in export
Day 9: Import state documentation
       Verify all constraints documented
```

**Day 10+: Evidence & Review**
```
Day 10: collect-evidence.sh → complete evidence directory
        verify-evidence.sh → verification report
Day 11+: Independent reviewer verifies
         Addresss findings (if any)
         Final approval
```

## Important Notes

### Real Node Requirement

This qualification **must** run on a real, clean Linux machine—not a simulator, mock, or development environment. The point is to prove real-world operations, not test coverage.

### Separation from Development

The qualification node must be:
- Separate from development/test environments
- Never used for development purposes
- Network-isolated if possible (or at least not sharing the same cluster)
- Treated as a production deployment

### Evidence Immutability

Once qualification begins, all evidence is collected and must remain unchanged:
- Logs are write-only (append-only)
- Timestamps are immutable (recorded at time of event)
- Audit trail is cryptographically signed (if applicable)
- Any change to evidence invalidates qualification

### Plaintext Secrets Protection

Qualification kit automatically:
- Scans evidence for plaintext secrets (fails if found)
- Records policies for secret delivery (not the secrets themselves)
- Verifies secrets are never logged or stored unencrypted
- Documents secret lifecycle (creation, delivery, rotation, revocation)

## Troubleshooting

### preflight.sh fails with FAILURE

Common causes:
- Kernel too old (need 6.x+)
- cgroups v2 not available (add `cgroup_no_v2` to boot params, reboot)
- WireGuard not available (install linux-headers, rebuild module)
- Insufficient resources (need 4 CPU, 8 GB RAM, 100 GB disk)

### install.sh cannot reach control plane

- Verify control plane IP/hostname and port
- Check firewall rules (allow 25001/tcp)
- Verify network connectivity: `nc -zv control-plane 25001`
- Check control plane logs for enrollment endpoint errors

### Operator approval never arrives

- Verify control plane has enrollment pending approval
- Check operator interface for new enrollment requests
- If manual approval required, accept in UI
- If timeout in agent, may need to re-enroll

### Workload health checks fail

- Verify fixture application deployed correctly
- Check workload logs: `dh-noded status`
- Verify resource constraints not too restrictive
- Check network connectivity from workload

### Chaos scenarios hang

- May indicate actual system issue (not intentional failure)
- Check system logs: `journalctl -u dh-noded -n 100`
- Verify control plane still responding
- If needed, manually intervene and document in recovery.sh

### Evidence verification fails

- Check file hashes are present for all files
- Verify no plaintext secrets in logs
- Check timestamp ordering (no gaps or reversals)
- Review warnings in verification-report.json

## Support

For issues or questions:
1. Check qualification/p0-sovereign/README.md (this file)
2. Review evidence/campaign.log and evidence/recovery.log
3. Consult control plane logs on operator machine
4. Report findings with full evidence/ directory

## References

- Specification: https://github.com/codesbyfebin/decentralized-/docs/P0-QUALIFICATION.md
- Product Documentation: https://github.com/codesbyfebin/decentralized-/docs/
- Control Plane API: https://github.com/codesbyfebin/decentralized-/pkg/api/

---

**Status:** Kit complete and ready for deployment  
**Next Step:** Provision clean Linux node and run `./preflight.sh --collect`
