# Decentralized.Host Operator Qualification Program (dh/v1)

**Status**: Implemented  
**Version**: 1.0  
**Last Updated**: 2026-10-03

## Executive Summary

The Operator Qualification Program establishes a comprehensive framework for onboarding, validating, and certifying operators who contribute infrastructure resources to the Decentralized.Host network. The program implements a multi-stage qualification pipeline that ensures operators meet security, reliability, and operational standards before being certified for mainnet deployment.

## Program Overview

### Tier Progression System

Operators progress through four certification tiers based on accumulated work and reputation:

```
BOOTSTRAP (0 uWork)
    ↓
NOVICE (100K uWork, reputation ≥400)
    ↓
JOURNEYMAN (1M uWork, reputation ≥600)
    ↓
MASTER (10M uWork, reputation ≥800) → CERTIFICATION (CERTIFIED)
```

### Core Requirements

**Stake Validation**
- Minimum 100M uWork stake required (locked and enforced)
- 14-day unbonding period after stake withdrawal requested
- Automatic slashing for SLA violations (5% default, 20% for critical)
- Slashing penalties reduce operator reputation and can trigger compliance reviews

**Node Registration**
- Minimum 50 nodes for readiness verification
- Hardware validation: ≥2 CPUs, ≥4GB RAM, ≥100GB disk
- Network validation: ≥10 Mbps bandwidth, <5% packet loss
- Geographic diversity: nodes in ≥3 distinct regions preferred
- Health monitoring: continuous uptime tracking, automated alerting

**SLA Compliance Enforcement**
- 95%+ uptime target (measured per 24-hour window)
- P99 latency <1 second (hard target, critical violations at >2s)
- Violation penalties: 5% stake slash + reputation reduction
- At-risk status triggers: uptime 80-95% with warning level alerts
- Historical compliance tracking for trend analysis

**Security Audit Framework**
- Five core security checks (required for MASTER tier):
  1. Operator identity verification (Ed25519 signature validation)
  2. Key management validation (secure key storage and rotation)
  3. Network security assessment (TLS, firewall, port validation)
  4. Data isolation verification (filesystem, network, crypto separation)
  5. Encryption enforcement (at-rest and in-transit)
  
- Additional checks for enhanced compliance:
  6. TLS certificate validation (ACME/Pebble integration)
  7. Firewall rules validation
  8. Encryption at rest verification
  9. Encryption in transit verification
  10. Access control verification
  11. Audit logging verification

- Compliance levels:
  - UNVERIFIED: No checks passed
  - BASIC: 1/4 core checks passed
  - STANDARD: 2/4 core checks passed
  - ADVANCED: 3/4 core checks passed
  - CERTIFIED: All core checks + ≥4 additional checks passed

**Backup Configuration Validation**
- Minimum 2 failover nodes (mandatory)
- Geographic distribution: nodes in ≥2 regions (preferred)
- RTO target: <5 minutes (Recovery Time Objective)
- RPO target: <1 minute (Recovery Point Objective)
- Failover testing: quarterly validation with pass/fail results
- Data retention: minimum 7 days backup history

## Implementation Architecture

### Package Structure

```
pkg/operator/
├── qualification/    # Tier progression, reputation scoring
├── stake/           # Stake validation, bonding, slashing
├── nodes/           # Node registration, health monitoring
├── sla/             # SLA compliance tracking
├── security/        # Security audit framework
└── backup/          # Backup and failover validation
```

### Core Interfaces and Types

**Qualification Manager**
```go
type QualificationManager struct {
    operators map[string]*OperatorProfile
}

type OperatorProfile struct {
    OperatorID string           // Unique operator identifier
    NodeID string               // dh1 identity
    CurrentTier Tier            // BOOTSTRAP|NOVICE|JOURNEYMAN|MASTER
    AccumulatedWork uint64      // Total work units completed
    ReputationScore int16       // 0-1000 points
    Certified bool              // Mainnet readiness
    CertifiedAt *time.Time      // Certification timestamp
    TierHistory []TierAdvance   // Audit trail of tier changes
}
```

**Staking Manager**
```go
type StakingManager struct {
    stake map[string]*StakeAccount
}

type StakeAccount struct {
    OperatorID string          // Operator identifier
    StakedAmount uint64        // Current stake (uWork)
    Status StakeStatus         // ACTIVE|UNBONDING|RELEASED|SLASHED
    SlashHistory []SlashEvent  // Audit trail of penalties
    TotalSlashed uint64        // Cumulative slashing amount
}
```

**Node Registry**
```go
type NodeRegistry struct {
    nodes map[string]*NodeInfo
    ops map[string][]string  // operator -> node IDs
}

type NodeInfo struct {
    NodeID string                      // dh1 identifier
    OperatorID string                  // Operator owner
    Status NodeStatus                  // HEALTHY|DEGRADED|UNHEALTHY|OFFLINE
    HardwareProfile HardwareProfile    // CPU, memory, disk specs
    NetworkProfile NetworkProfile      // IPv4, port, bandwidth, latency
    GeographicLocation GeographicLocation
    UptimePercentage float64          // 0-100%
}
```

**SLA Manager**
```go
type SLAManager struct {
    metrics map[string]*SLAMetrics
}

type SLAMetrics struct {
    OperatorID string                   // Operator identifier
    Status ComplianceStatus             // COMPLIANT|AT_RISK|VIOLATED|UNKNOWN
    CurrentUptime float64               // Percentage
    P99Latency time.Duration           // 99th percentile
    ViolationCount int                 // Cumulative violations
    AlertLevel string                   // NONE|WARNING|CRITICAL
}
```

**Security Manager**
```go
type SecurityManager struct {
    audits map[string]*OperatorAudit
}

type OperatorAudit struct {
    OperatorID string              // Operator identifier
    ComplianceLevel ComplianceLevel // UNVERIFIED|BASIC|STANDARD|ADVANCED|CERTIFIED
    OverallScore int16             // 0-100 points
    AuditResults []AuditResult    // Individual check results
    CertificationTime *time.Time   // When audit was certified
    ExpirationTime *time.Time      // When certification expires (1 year)
}
```

**Backup Manager**
```go
type BackupManager struct {
    configs map[string]*BackupConfiguration
}

type BackupConfiguration struct {
    OperatorID string               // Operator identifier
    Status BackupStatus             // CONFIGURED|NOT_CONFIGURED|DEGRADED|TESTED_PASSING|TESTED_FAILING
    FailoverNodes []FailoverNode   // Backup node list
    LastTestResult *FailoverTest   // Last failover test results
}
```

## Qualification Flow

### 1. Operator Registration

```
Register Operator
├─ Create identity (Ed25519 key pair)
├─ Initialize profile (BOOTSTRAP tier, 500 reputation)
└─ Begin qualification tracking
```

### 2. Stake Deposit and Validation

```
Deposit Stake (100M uWork minimum)
├─ Validate amount ≥ minimum
├─ Lock stake (ACTIVE status)
├─ Record lock timestamp
└─ Enable work tracking
```

### 3. Node Registration and Health

```
Register Nodes (50+ minimum)
├─ For each node:
│  ├─ Validate hardware (CPU, memory, disk)
│  ├─ Validate network (bandwidth, latency, packet loss)
│  ├─ Record geographic location
│  ├─ Initialize health checks
│  └─ Track uptime percentage
├─ Verify geographic diversity (3+ regions)
└─ Start continuous health monitoring
```

### 4. Work Accumulation and Tier Progression

```
For each work unit completed:
├─ Record work unit (latency measurement)
├─ Update accumulated work count
├─ Update average latency
├─ Check tier advancement conditions:
│  ├─ BOOTSTRAP→NOVICE: 100K work + 400 reputation
│  ├─ NOVICE→JOURNEYMAN: 1M work + 600 reputation
│  └─ JOURNEYMAN→MASTER: 10M work + 800 reputation
└─ Record tier change in history
```

### 5. SLA Monitoring and Compliance

```
Every monitoring window (hourly):
├─ Collect success/failure counts
├─ Calculate uptime percentage
├─ Measure P99 latency
├─ Evaluate compliance:
│  ├─ COMPLIANT: uptime ≥95% AND P99 ≤1s
│  ├─ AT_RISK: uptime 80-95% OR P99 1-2s
│  └─ VIOLATED: uptime <80% OR P99 >2s
├─ If violated:
│  ├─ Apply stake slash (5% default, 20% critical)
│  ├─ Reduce reputation (-5 to -20 points)
│  ├─ Record violation
│  └─ Trigger alert
└─ Update historical compliance
```

### 6. Security Audit

```
Initiate Audit
├─ Create audit record
└─ For each check type:
   ├─ Identity verification
   ├─ Key management validation
   ├─ Network security assessment
   ├─ Data isolation verification
   ├─ Optional additional checks
   └─ Record results with evidence
```

### 7. Backup Configuration

```
Configure Backups
├─ Create backup configuration
├─ Add failover nodes
│  └─ For each node:
│     ├─ Record replication lag (RPO indicator)
│     ├─ Monitor health status
│     └─ Track synchronization
├─ Execute failover tests
│  └─ Measure RTO (recovery time)
│  └─ Measure actual RPO (data loss window)
└─ Verify geographic distribution
```

### 8. Certification

```
Certify Operator (MASTER tier only)
├─ Verify tier = MASTER
├─ Verify reputation ≥800
├─ Verify all SLA targets met
├─ Verify security = CERTIFIED
├─ Verify backup status = TESTED_PASSING
├─ Issue certification
├─ Set expiration (1 year)
└─ Record timestamp
```

## Compliance Monitoring and Enforcement

### Reputation System

**Starting Value**: 500/1000 (neutral)  
**Maximum**: 1000  
**Minimum**: 0

**Adjustments**:
- Work completion: +1-5 per unit (based on quality metrics)
- SLA violation: -5 to -20 (based on severity)
- Security audit failure: -10 to -50 (based on severity)
- Successful tier advancement: +20
- Certification issued: +50

### Stake Slashing

**Default Violation** (SLA target miss):
- Slash percentage: 5%
- Trigger: Single SLA violation
- Recovery: No automatic recovery; requires manual deposit

**Critical Violation** (cascading failures):
- Slash percentage: 20%
- Trigger: 3+ consecutive SLA violations within 7 days
- Recovery: No automatic recovery; severe reputation penalty

**Insufficient Stake**:
- Status change: INSUFFICIENT
- Effect: Operator cannot participate in work
- Recovery: Must deposit new stake to resume

### Alert Levels

| Alert Level | Condition | Action |
|-------------|-----------|--------|
| NONE | All targets met | Routine monitoring |
| WARNING | 1-2 targets approaching limits | Escalated monitoring |
| CRITICAL | 3+ targets violated OR violations trending | Immediate escalation |

## Compliance Verification

### Evidence Binding

All compliance claims are backed by:
1. **Source SHA**: Git commit hash of implementation
2. **Campaign ID**: Qualification campaign identifier
3. **Timestamp**: UTC time of measurement
4. **Node Identity**: dh1 identity of reporting node
5. **Artifact Digest**: BLAKE3 hash of evidence data
6. **Signer**: Ed25519 public key of signing operator

### No Simulation Policy

- All P1 qualification gates reject simulation
- All measurements must be from production infrastructure
- Manual state edits not permitted
- Hardcoded timing not permitted
- All latencies measured from observed timestamps
- All observations from production runtime paths

## Testing and Validation

### Test Coverage

- Unit tests: 100+ test cases across all packages
- Integration tests: Multi-component qualification flows
- Chaos tests: 17 failure scenarios
- Compliance tests: 136 normative test vectors

### Test Vectors

#### Qualification Tests
- Tier progression validation
- Reputation score clamping (0-1000)
- Certification eligibility checks
- Tier history audit trail

#### Stake Tests
- Minimum stake enforcement (100M uWork)
- Unbonding period validation (14 days)
- Slashing calculations and balance updates
- Insufficient stake detection

#### Node Tests
- Hardware validation (CPU, memory, disk)
- Network validation (bandwidth, latency, packet loss)
- Geographic diversity tracking
- Health check uptime calculations

#### SLA Tests
- Uptime percentage calculations
- P99 latency tracking and sorting
- Compliance status determination
- Violation detection and alerting
- Trending analysis (improving vs degrading)

#### Security Tests
- Compliance level progression
- Audit result recording
- Signature verification
- Certification lifecycle (issue, renew, revoke)

#### Backup Tests
- Failover node registration
- RTO measurement and validation
- RPO measurement and validation
- Failover test execution

## CLI Tools

### operator-qualification CLI

```bash
# Operator management
operator-qualification register-operator -id <id> -node <nodeID>

# Staking
operator-qualification stake deposit -id <id> -amount <uwork>
operator-qualification stake validate -id <id>

# Node management
operator-qualification nodes info -id <id>

# SLA tracking
operator-qualification sla register -id <id>
operator-qualification sla status -id <id>

# Security audits
operator-qualification security audit -id <id> -node <nodeID>

# Backup configuration
operator-qualification backup configure -id <id> -schedule daily -retention 30

# Status reporting
operator-qualification status
```

## Metrics and Reporting

### Qualification Status Summary

```
Total Operators: N
├─ BOOTSTRAP tier: N
├─ NOVICE tier: N
├─ JOURNEYMAN tier: N
└─ MASTER tier: N (of which CERTIFIED: M)

Certified Operators: M
Ready for Mainnet: K
Average Reputation: R/1000
```

### Staking Status

```
Total Staked: X uWork
├─ Active Operators: N
├─ Slashed Operators: M
└─ Average Stake: X/N uWork
```

### SLA Compliance

```
Compliant Operators: N
├─ At Risk: M
└─ Violated: K

Average Uptime: Y%
Average P99 Latency: Z ms
Critical Alerts: K
```

### Security Compliance

```
Security Certified: N
├─ Advanced: M
├─ Standard: K
├─ Basic: J
└─ Unverified: I

Critical Findings: X
High Findings: Y
Average Compliance Score: Z/100
```

### Backup Readiness

```
Backup Configured: N/Total
├─ Ready for Production: M
└─ Needs Remediation: K

Average RTO: X minutes
Average RPO: Y seconds
```

## Integration Points

### dh/v1 Conformance Spec

This operator qualification program implements Milestone M8 of the dh/v1 conformance specification:
- M1: Signed intent acquisition and validation ✓
- M2: Local policy matching with audit logging ✓
- M3: BLAKE3 CAS with Merkle anti-entropy ✓
- M4: Userspace WireGuard with signed key bindings ✓
- M5: Raft + mTLS control plane ✓
- M6: ACME TLS integration (Pebble) ✓
- M7: Chaos validation (17 scenarios) ✓
- M8: **Operator Qualification Program** ✓

### Integration with Identity Package

All operator and node identities use the dh/v1 Ed25519 identity system:
- Node IDs follow dh1[a-z2-7]{26} pattern
- Public key encoding via base64url
- Private key storage via PKCS#8 PEM
- Signature verification for identity proofs

### Integration with Policy Package

Operator stake and capability levels map to local host policy:
- BOOTSTRAP tier: restricted work types only
- NOVICE tier: trusted work types
- JOURNEYMAN tier: federation-capable work
- MASTER tier: full capability scope

### Integration with Evidence Package

All qualification measurements are backed by cryptographic evidence:
- Attestation signing via operator Ed25519 keys
- Evidence binding to source code SHA
- Campaign and timestamp recording
- BLAKE3 content hashing of artifacts

## Future Enhancements

### Phase 2: Operator Economics

- Dynamic fee structures based on operator tier
- Work allocation algorithms (reputation-weighted)
- Revenue sharing and payment settlement
- Penalty escrow and bonding

### Phase 3: Governance

- Operator community voting on policy changes
- Dispute resolution mechanisms
- Policy evolution tracking
- Democratic tier advancement

### Phase 3: Geographic Enforcement

- Physical-host failure domain verification
- Multi-operator administrative domains
- Independent operator licensing
- Cross-operator audit requirements

## References

- `pkg/operator/qualification/` - Tier progression implementation
- `pkg/operator/stake/` - Stake management
- `pkg/operator/nodes/` - Node registration and monitoring
- `pkg/operator/sla/` - SLA compliance tracking
- `pkg/operator/security/` - Security audit framework
- `pkg/operator/backup/` - Backup and failover validation
- `cmd/operator-qualification/` - CLI tool
- `tests/operator/` - Comprehensive test suite
