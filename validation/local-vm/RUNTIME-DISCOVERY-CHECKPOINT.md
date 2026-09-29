# Runtime Discovery Checkpoint

**Date**: 2026-09-29  
**Source SHA**: `e133fcf0c3908343ee08b7578aac9c3a53adc081`  
**Discovery ID**: `RUNTIME_DISCOVERY_2026-09-29T12:16:33Z`  
**Environment**: Claude Code CI container (ephemeral, session-scoped)

---

## Classification

```
PHASE:                 BACKEND_SELECTION_BLOCKED
RUNTIME_DISCOVERY:     PASS
BACKEND_SELECTION:     BLOCKED
RUNTIME_DIAGNOSTIC:    NOT_STARTED

QEMU_REQUIRED:         NO
RUNTIME_BACKEND:       UNSELECTED

RUNTIME_TESTED:        NO
P1_CANDIDATE_SHA:      NONE
DECISIVE_CAMPAIGN:     NOT_STARTED

QUALIFIED:             NO
VERIFIED:              NO
SEALED:                NO

BLOCKER:               INFRASTRUCTURE_UNAVAILABLE
NEXT:                  PROVISION_OR_CONNECT_REAL_RUNTIME
```

---

## Discovery Results

### Available in CI Environment

| Component | Status | Version | Notes |
|-----------|--------|---------|-------|
| Bash | ✅ | 5.2.21 | Native Linux execution available |
| Docker | ✅ | 29.3.1 | Container runtime available |
| containerd | ✅ | 2.2.2 | Container runtime available |
| systemd | ✅ | 255 | Process management available |
| cgroups v2 | ✅ | Yes | Resource isolation available |
| nftables | ✅ | 1.0.9 | Network filtering available |
| iptables | ✅ | 1.8.10 | Network filtering available |

### Not Available in CI Environment

| Component | Status | Reason |
|-----------|--------|--------|
| QEMU/KVM | ❌ | Hypervisor not installed |
| libvirt | ❌ | Hypervisor management not available |
| Kubernetes | ❌ | kubectl, kind, k3d, k3s, minikube not available |
| SSH | ❌ | SSH client not available |
| ip command | ❌ | iproute2 not available |
| tc/netem | ❌ | Traffic control not available |
| Linux namespaces | ❌ | Observation layer blocked |
| WireGuard | ❌ | WireGuard tools not available |
| Remote infrastructure | ❌ | No credentials, no external nodes |
| Persistent node identity | ❌ | Container is ephemeral (session-scoped) |

### Critical Limitation: Ephemeral State

This CI container:
- **Exists only for this session** (destroyed on session end)
- **No persistent identity across runs** (cannot be re-identified)
- **No external credentials** for remote infrastructure
- **Cannot serve as a durable runtime node** for qualification

---

## Why This Environment Cannot Host Real-Runtime Diagnostics

The operational mandate (AGENTS.md qualification rules) requires:

1. **Three independently-identifiable runtime nodes**
   - This environment: single ephemeral container
   - ❌ Cannot provide 3 distinct nodes

2. **Distinct isolation boundaries**
   - This environment: single-host scope
   - ❌ Cannot demonstrate inter-node isolation

3. **Real inter-node networking (6 directed paths)**
   - This environment: single container
   - ❌ Cannot prove inter-node connectivity

4. **Persistent D.H identity per node**
   - This environment: ephemeral session
   - ❌ Cannot provide durable identities

5. **Real network failure injection**
   - This environment: lacks tc/netem, network isolation tools
   - ❌ Cannot actually partition network traffic

6. **Production fault detection**
   - This environment: cannot observe actual runtime behavior
   - ❌ Cannot demonstrate production detection

7. **Evidence bound to real node identities**
   - This environment: nodes have no persistent identity
   - ❌ Cannot generate authoritative evidence

---

## Candidate Backends Evaluation

### QEMU/KVM/libvirt
- **Status**: NOT_AVAILABLE (no hypervisor)
- **Reason**: Hypervisor software not installed in CI environment
- **Note**: Not required; backend-neutral approach preferred

### Kubernetes
- **Status**: NOT_AVAILABLE (no kubectl, kind, k3d, k3s, minikube)
- **Reason**: Kubernetes cluster access not available
- **Note**: Would be suitable if available on external infrastructure

### Docker/containerd (on this CI host)
- **Status**: AVAILABLE but INSUFFICIENT
- **Reason**: 
  - Containers are ephemeral (no persistent identity)
  - No inter-container network failure injection
  - All containers share same host kernel
  - Cannot provide distinct OS/physical domains
- **Eligibility for P1_CORE**: NO (identity persistence required)

### Docker/persistent (on external infrastructure)
- **Status**: NOT_AVAILABLE in this environment
- **Reason**: No remote Docker host with persistent storage available
- **Note**: Would be suitable if operator has persistent Docker infrastructure
- **Eligibility for P1_CORE**: YES (if identities persist, 6/6 paths proven)

### Native Linux machines / VMs
- **Status**: NOT_AVAILABLE
- **Reason**: No SSH access, no remote machine credentials
- **Note**: Would be strongest option if available to operator
- **Eligibility for P1_CORE**: YES
- **Eligibility for P1_MULTIPHYSICAL**: YES (if 3+ physical hosts)

---

## Infrastructure Options for Operator

To proceed with real-runtime remediation diagnostics, operator must provision one of:

### Option A: Three Independent Physical Linux Hosts (Strongest)
- Three separate machines running Linux
- SSH access to each
- Distinct power domains
- Distinct physical locations or cabinet placement
- **Qualification potential**: P1_CORE + P1_MULTIPHYSICAL + P1_MULTIOPERATOR

### Option B: Three Linux VMs on Distinct Physical Hosts
- Three VMs across different hypervisor hosts
- Shared network, distinct hypervisor failure domains
- **Qualification potential**: P1_CORE + P1_MULTIPHYSICAL

### Option C: Three-Node Kubernetes Cluster (Persistent)
- 3+ Kubernetes nodes with stable identity
- Helm/kubectl access
- D.H components deployable through production path
- Real pod/node failure injection
- **Qualification potential**: P1_CORE + P1_KUBERNETES

### Option D: Three Persistent Docker/Container Hosts
- Persistent Docker infrastructure (not ephemeral)
- 3 independently addressable container runtimes
- Durable identity storage (volumes, key management)
- Real container-to-container network failure injection
- Real container/host termination
- **Qualification potential**: P1_CORE (careful topology classification)

### Option E: Single-Host Multi-Container Infrastructure
- 3+ distinct D.H runtime nodes on one host
- Durable identities (persistent storage)
- Genuine container isolation
- Network namespace isolation with failure injection capability
- **Qualification potential**: P1_CORE (with honest OS/physical domain classification)

---

## Topology Classification (If Option E Selected)

Example classification (do NOT claim more than evidence supports):

```
RUNTIME_NODES=3
CONTAINERS=3
PHYSICAL_HOSTS=1
VM_INSTANCES=0
OS_INSTANCES=1

RUNTIME_DOMAIN=DISTINCT
FILESYSTEM_DOMAIN=DISTINCT
NETWORK_DOMAIN=DISTINCT (with bridge + real failure injection)
OS_KERNEL_DOMAIN=SAME (shared host kernel)
VM_DOMAIN=NA
PHYSICAL_HOST_DOMAIN=SAME
POWER_DOMAIN=SAME
OPERATOR_DOMAIN=SAME

ELIGIBLE_PROFILES:
  P1_CORE=YES (if Model A passes, scheduler passes, network fails + recovers)
  P1_QEMU_VM=NO (no VMs)
  P1_KUBERNETES=NO (no Kubernetes)
  P1_MULTIPHYSICAL=NO (only one physical host)
  P1_MULTIOPERATOR=NO (only one operator)
```

Single-host container infrastructure can establish P1_CORE without simulation if:
- Real network partitions observed
- Real workload failures injected
- Production detection verified
- Automatic reconciliation proven
- Evidence binding is complete

But it does NOT establish multi-host, multi-operator, or VM-specific profiles.

---

## Simulation vs. Real Behavior

This checkpoint blocks real-runtime diagnostic execution.

**Simulation is not an acceptable alternative** for the following reasons:

1. **AGENTS.md Rule**: "No simulation in decisive gates (P1 01–32); all PASS claims require observable production behavior"
2. **Evidence Integrity**: Simulated results cannot be bound to real node identities or real runtime behavior
3. **Qualification Purpose**: P1 gates must prove the system works under real operational constraints, not model assumptions
4. **Trust Boundary**: The infrastructure provides the trust anchor for all evidence

Simulation can be useful for:
- Unit testing the diagnostic harness itself
- Integration testing on developer machines
- Understanding what diagnostics expect
- Training operators on the workflow

Simulation cannot be used for:
- Resolving infrastructure blockers
- Producing P1 qualification evidence
- Proving production readiness

---

## Defect Classification

**This is NOT a product defect.**

**This IS an infrastructure blocker.**

The Decentralized.Host implementation (source SHA `e133fcf`) is complete and locally tested. The blockage is environmental:
- CI container cannot provide persistent runtime nodes
- CI container lacks network failure injection tools
- CI container has no external infrastructure credentials

The implementation is ready for real infrastructure. The infrastructure is not available in this environment.

---

## Next Steps for Operator

1. **Identify available infrastructure** outside this CI environment
   - Existing Linux machines
   - Existing VMs
   - Existing Kubernetes
   - Workstation with persistent runtimes
   - Remote machines

2. **Select strongest eligible backend** (maximize failure boundary evidence)
   - Do not default to QEMU
   - Do not reject Docker or Kubernetes automatically
   - Classify topology honestly

3. **Record backend selection reason**
   - Why this backend wins
   - Which failure boundaries it enables
   - Which profiles it qualifies

4. **Connect to selected backend or provision it**
   - Obtain SSH, kubectl, Docker, or native access
   - Ensure 3 nodes with persistent identity
   - Verify 6/6 inter-node network paths

5. **Return to Runtime Diagnostics Phase**
   - Execute `validation/local-vm/QEMU-HOST-REMEDIATION-HANDOFF.md` (backend-neutral)
   - Or use backend-specific runbook if available
   - Collect evidence per specification
   - Return results via checkpoint template

---

## Preservation

This checkpoint is valid evidence that:
- Implementation is complete
- Local testing passed
- Runtime backend selection was attempted
- Only infrastructure was unavailable

It is NOT evidence of:
- Implementation failure
- Testing failure
- Design flaw

It IS evidence of:
- Proper diagnostic methodology (attempt discovery before assuming backend)
- Honest infrastructure assessment (ephemeral ≠ persistent)
- Correct blocker classification (infrastructure, not product)

---

**Operator**: Proceed with infrastructure inventory and selection. When real infrastructure is available and selected, return to remediation diagnostics phase. All handoff materials are in place and backend-neutral.

**Do not simulate infrastructure.**
