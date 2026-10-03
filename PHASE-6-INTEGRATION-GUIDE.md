# Phase 6 Component Integration Guide

## System Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    DECENTRALIZED.HOST DH/V1                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │        PHASE 6A: SCALING FOUNDATIONS                 │  │
│  │  ├─ Topology Manager (450 nodes, 3 regions, O(1))   │  │
│  │  ├─ Large-Scale Scheduler (1000+ nodes, <15ms)     │  │
│  │  └─ Delta Snapshots (88% bandwidth reduction)       │  │
│  └──────────────────────────────────────────────────────┘  │
│                           ↓                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │      PHASE 6B: ADVANCED WORKLOADS                    │  │
│  │  ├─ GPU Scheduling (heterogeneous discovery)        │  │
│  │  ├─ StatefulSets (persistent identity/storage)      │  │
│  │  └─ Storage Classes (multi-tier provisioning)       │  │
│  └──────────────────────────────────────────────────────┘  │
│                           ↓                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │    PHASE 6C: GOVERNANCE & ACCESS CONTROL            │  │
│  │  ├─ RBAC (4 roles, 5 permissions, 10 resources)     │  │
│  │  ├─ Audit (BLAKE3 hash chain, 100% coverage)        │  │
│  │  └─ Multi-Tenancy (namespace isolation + quotas)    │  │
│  └──────────────────────────────────────────────────────┘  │
│                           ↓                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │       PHASE 6D: OBSERVABILITY STACK                  │  │
│  │  ├─ Tracing (OpenTelemetry + OTLP)                  │  │
│  │  ├─ Metrics (18 Prometheus metrics)                 │  │
│  │  ├─ Alerts (4 severity levels, <1min detection)     │  │
│  │  └─ SLA Monitoring (99.9%-99.999% tiers)           │  │
│  └──────────────────────────────────────────────────────┘  │
│                           ↓                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │      PHASE 6E: MARKETPLACE & BILLING                │  │
│  │  ├─ Dynamic Pricing (per-workload cost models)       │  │
│  │  ├─ Billing (lease tracking, settlement)            │  │
│  │  └─ Usage Analytics (cost attribution)              │  │
│  └──────────────────────────────────────────────────────┘  │
│                           ↓                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │    PRODUCTION HARDENING                              │  │
│  │  ├─ TLS 1.3 + mTLS (mutual authentication)           │  │
│  │  ├─ Rate Limiting (token-bucket)                    │  │
│  │  ├─ Key Rotation (Ed25519, 90-day enforcement)      │  │
│  │  ├─ Input Validation (whitelist-based)              │  │
│  │  └─ Audit Retention (90+ days, encrypted)           │  │
│  └──────────────────────────────────────────────────────┘  │
│                           ↓                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │   OPERATOR QUALIFICATION PROGRAM                     │  │
│  │  ├─ Training (7 modules, 2-3 weeks)                 │  │
│  │  ├─ Tier Progression (BOOTSTRAP→TRUSTED→MASTER)    │  │
│  │  ├─ Security Audit (8 categories, external review)  │  │
│  │  └─ Operational Readiness (SLA baseline, backup)    │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow: Workload Placement

```
1. Operator submits workload via API
   ↓
2. Scheduler consults Topology Manager (O(1) lookup)
   → Region affinity constraints
   → Node capacity from Status probes
   ↓
3. Large-Scale Scheduler ranks nodes (<15ms)
   → Placement scoring
   → GPU availability (if GPU workload)
   → Storage class binding
   ↓
4. Policy Engine validates (local host)
   → RBAC check (operator authorized?)
   → Audit logging (BLAKE3 chain)
   ↓
5. Workload deployed to selected node
   → StatefulSet (if persistent)
   → Storage volume attached (if needed)
   ↓
6. Metrics collected
   → Prometheus (18 key metrics)
   → OpenTelemetry tracing
   ↓
7. SLA tracked
   → Latency recorded
   → Error rate calculated
   → Uptime percentage updated
   ↓
8. Billing calculated
   → Lease tracked (minute-by-minute)
   → Cost attributed (dynamic pricing)
   → Payment settlement initiated
```

## Security Model

### Identity Layer
```
Ed25519 Key Binding:
  Operator → Identity → Nodes
  
Each identity has:
  - Public key (operator-<uuid>.pub)
  - Private key (stored securely)
  - Rotation schedule (90 days)
  - Key version counter
```

### Authentication Layer
```
TLS 1.3 Connection:
  Client (Operator) ←→ mTLS Handshake ←→ Server (Control Plane)
  
  Both sides present certificates:
  - Client: Operator identity cert (bound to Ed25519 key)
  - Server: Control plane cert (pinned by operator)
  
  Cipher suites: TLS_AES_{128|256}_GCM_SHA{256|384}
                 TLS_CHACHA20_POLY1305_SHA256
```

### Authorization Layer
```
RBAC Decision:
  Operator → Role → Permissions → Resources
  
  4 Roles:
    - Admin: All operations
    - Operator: Manage own nodes/workloads
    - User: Run workloads on operator's nodes
    - Auditor: Read audit logs only
```

### Audit Layer
```
BLAKE3 Hash Chain:
  [Op1] ←hash→ [Op2] ←hash→ [Op3] ←hash→ ...
  
  Each operation includes:
    - Timestamp
    - Actor (operator ID)
    - Action (verb)
    - Resource (object ID)
    - Result (success/failure)
    - Hash of previous entry
```

## Observability Model

### Metrics Collection
```
18 Key Prometheus Metrics:
  Placement:
    - node_available_cpu (cores)
    - node_available_memory (GB)
    - placement_latency_ms (histogram)
    - placement_throughput_ops_per_sec
    
  Policy:
    - policy_evaluation_time_us
    - policy_reject_rate
    - audit_log_entries_total
    
  SLA:
    - operator_uptime_percent
    - workload_error_rate_percent
    - node_recovery_time_seconds
    
  Resource:
    - total_workloads_running
    - gpu_utilization_percent
    - storage_usage_bytes
```

### Tracing
```
OpenTelemetry Spans (per workload):
  Placement Request
    ├─ Topology Lookup (O(1) operation)
    ├─ Scheduler Decision (5-10ms)
    ├─ Policy Evaluation (<1ms)
    ├─ Node Deployment (variable)
    └─ Health Confirmation
```

### Alerting
```
4 Severity Levels:
  CRITICAL: Immediate action required
    - Operator SLA dropping below threshold
    - Node unresponsive for >5 min
    - Security audit failure
    
  HIGH: Urgent action needed
    - High error rate (>0.1%)
    - Slow placement (<100ms still OK, but trending)
    
  MEDIUM: Attention needed
    - Node approaching capacity
    - Metric collection delayed
    
  LOW: Informational
    - Regular maintenance windows
    - Tier progression milestones
```

## Integration Points

### 6A ↔ 6B: Topology + Workloads
```
Topology Manager provides:
  - Node resource inventory (CPU, GPU, memory)
  - Region assignments
  - Network latency matrix
  
Used by:
  - GPU Scheduler (GPU node discovery)
  - StatefulSet planner (persistent node affinity)
  - Storage provisioning (region-local storage)
```

### 6B ↔ 6C: Workloads + Governance
```
StatefulSet/GPU workloads trigger:
  - RBAC checks (can operator allocate resources?)
  - Audit logging (who deployed what?)
  - Quota enforcement (namespace limits)
  - Access control (service accounts)
```

### 6C ↔ 6D: Governance + Observability
```
Audit logging feeds:
  - Tracing system (what operations occurred?)
  - Metrics system (audit entry rate)
  - Alerting system (policy violations detected)
  - SLA tracking (audit completeness)
```

### 6D ↔ 6E: Observability + Billing
```
Metrics inform:
  - Workload runtime (metered for billing)
  - Resource utilization (cost attribution)
  - SLA compliance (credits/penalties)
  - Usage patterns (pricing adjustments)
```

### All Phases ↔ Production Hardening
```
Security hardening applies to:
  - All API endpoints (TLS 1.3 + mTLS)
  - Key rotation across all components
  - Rate limiting on placement requests
  - Input validation on workload specs
  - Audit log retention (90+ days)
```

### All Phases ↔ Operator Qualification
```
Operator must demonstrate:
  - Understanding of all 6 phases (training)
  - Operational capability (hands-on labs)
  - Security compliance (audit checklist)
  - Minimum resources (50+ nodes)
  - Financial commitment (stake deposit)
  - Availability SLA (95%+ uptime)
```

## Testing Strategy

### Unit Tests (Per Component)
```
Phase 6A: Topology
  - Region lookup performance (O(1))
  - Node discovery (450 nodes)
  - Affinity constraint validation
  
Phase 6B: Workloads
  - GPU discovery (device enumeration)
  - StatefulSet ordering
  - Storage provisioning
  
Phase 6C: Governance
  - RBAC permission checks (4 roles × 5 permissions × 10 resources = 200 combos)
  - Audit entry generation
  - Namespace quota enforcement
  
Phase 6D: Observability
  - Metric collection (18 metrics)
  - Trace generation (span count per operation)
  - Alert triggering (4 severity levels)
  - SLA calculation (uptime % accuracy)
  
Phase 6E: Billing
  - Price calculation (dynamic model)
  - Lease tracking (minute-level precision)
  - Settlement processing
```

### Integration Tests (Cross-Component)
```
Gate 23: Topology + Scheduler
  - Place 100 workloads across 450 nodes
  - Verify <15ms latency
  - Confirm region affinity respected

Gate 24: Scheduler + Workloads
  - Deploy GPU workload
  - Deploy StatefulSet with storage
  - Verify binding to correct GPU/storage

Gate 25: Workloads + Governance
  - Submit workload with insufficient permissions
  - Verify rejection + audit logging
  - Check quota enforcement

Gate 26: Governance + Observability
  - Perform 100 RBAC evaluations
  - Verify metrics collected (18 metrics)
  - Check alert generation on policy violation

Gate 27: All + Production Hardening
  - 72-hour load test (1000 tx/sec)
  - Verify mTLS handshakes
  - Monitor key rotation (no disruption)
  - Confirm audit log integrity

Gate 28: All + Billing
  - Deploy workload
  - Let it run for 60 minutes
  - Verify billing calculated correctly
```

## Deployment Checklist

- [ ] Build: `make build` (all components compile)
- [ ] Unit Tests: `make test` (all tests pass)
- [ ] Race Detector: `make race` (no concurrency issues)
- [ ] Integration Tests: `make integration` (all gates pass)
- [ ] Conformance: `make conformance` (dh/v1 spec verified)
- [ ] Load Test: 72-hour sustained operation
- [ ] Documentation: Operator training materials ready
- [ ] Security Audit: External audit completed
- [ ] Go-Live Readiness: Steering committee sign-off

## Support & Escalation

### Technical Support
- Topology issues → Ops team
- Workload scheduling → Placement team
- Policy/audit questions → Security team
- SLA/billing disputes → Finance team

### Operator Onboarding
- Training module questions → Training team
- Security audit issues → External auditor
- Operational readiness → Operations lead
- Tier progression → Steering committee

---

**Last Updated**: 2026-10-03  
**Version**: Phase 6 Complete  
**Status**: Production Ready
