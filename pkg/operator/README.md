# Operator Qualification Program

Complete implementation of the Decentralized.Host operator qualification framework for dh/v1 conformance.

## Overview

The Operator Qualification Program manages the complete lifecycle of infrastructure operators contributing to the Decentralized.Host network. It implements:

- **Tier Progression**: BOOTSTRAP → NOVICE → JOURNEYMAN → MASTER (with certification)
- **Stake Validation**: 100M uWork minimum with automatic slashing for violations
- **Node Registration**: 50+ node minimum with hardware and network validation
- **SLA Compliance**: 95%+ uptime and <1s P99 latency enforcement
- **Security Audits**: Multi-level compliance verification
- **Backup Validation**: RTO <5min and RPO <1min enforcement

## Packages

### qualification

Operator tier progression and reputation scoring system.

```go
import "decentralized.host/pkg/operator/qualification"

qm := qualification.NewQualificationManager()
profile, _ := qm.RegisterOperator("op_001", "dh1abcdef...")

// Record work and track tier advancement
qm.RecordWork("op_001", 100_000, 50*time.Millisecond)

// Certify for mainnet
qm.Certify("op_001") // Must be MASTER tier with 800+ reputation
```

**Types**:
- `Tier`: BOOTSTRAP, NOVICE, JOURNEYMAN, MASTER
- `OperatorProfile`: Current tier, work count, reputation score
- `TierAdvance`: Audit trail of tier changes
- `QualificationManager`: Central qualification tracking

### stake

Operator stake validation, bonding, and slashing.

```go
import "decentralized.host/pkg/operator/stake"

sm := stake.NewStakingManager()

// Deposit minimum stake
account, _ := sm.DepositStake("op_001", 100_000_000)

// Validate stake
sm.ValidateStake("op_001") // Error if insufficient

// Slash for SLA violation
sm.SlashStake("op_001", "SLA_VIOLATION", 5, "Uptime < 95%")

// Unbond and wait
sm.UnbondStake("op_001")
time.Sleep(14 * 24 * time.Hour)
sm.CompleteUnbonding("op_001")
```

**Constants**:
- `MinimumStake`: 100,000,000 uWork
- `UnbondingPeriod`: 14 days
- `DefaultSlashPercentage`: 5% (20% for CRITICAL)

**Types**:
- `StakeAccount`: Current stake, status, slash history
- `SlashEvent`: Slash records with reason and timestamp
- `StakingManager`: Stake and bonding management

### nodes

Node registration, validation, and health monitoring.

```go
import "decentralized.host/pkg/operator/nodes"

nr := nodes.NewNodeRegistry()

hardware := nodes.HardwareProfile{
    CPUCores: 4, MemoryGB: 16, DiskGB: 500, DiskType: "SSD",
}
network := nodes.NetworkProfile{
    IPv4Address: "192.168.1.100", Port: 8080,
    Bandwidth: 1000, Latency: 50*time.Millisecond, PacketLoss: 0.1,
}
location := nodes.GeographicLocation{
    Region: "us-east", Country: "US",
}

// Register node
nr.RegisterNode("dh1abcdef...", "op_001", hardware, network, location)

// Record health checks
nr.RecordHealthCheck("dh1abcdef...", true, 45*time.Millisecond)
nr.RecordHealthCheck("dh1abcdef...", false, 1*time.Second)

// Check readiness
status := nr.GetOperatorNodeStatus("op_001")
if status.IsReadyNodeCount && status.IsReadyRegions {
    // Operator has sufficient nodes and geographic diversity
}
```

**Constants**:
- `MinimumNodesForReadiness`: 50 nodes
- `MinimumGeographicRegions`: 3 regions

**Types**:
- `NodeInfo`: Node identity, status, hardware, network, location
- `HardwareProfile`: CPU, memory, disk specifications
- `NetworkProfile`: IPv4, bandwidth, latency, packet loss
- `GeographicLocation`: Region, country, coordinates
- `NodeRegistry`: Node registration and health tracking

### sla

SLA compliance tracking and enforcement.

```go
import "decentralized.host/pkg/operator/sla"

sm := sla.NewSLAManager()

// Register operator for SLA tracking
sm.RegisterOperator("op_001")

// Record metrics
latencies := []time.Duration{
    50*time.Millisecond, 100*time.Millisecond, 75*time.Millisecond,
}
sm.RecordMetrics("op_001", 95, 5, latencies) // 95 successes, 5 failures

// Check compliance
metrics, _ := sm.GetMetrics("op_001")
switch metrics.Status {
case sla.COMPLIANT:
    log.Println("SLA target met")
case sla.AT_RISK:
    log.Println("Warning: approaching SLA violation")
case sla.VIOLATED:
    log.Println("ALERT: SLA violation - applying penalties")
}

// Generate report
report, _ := sm.GenerateReport("op_001")
```

**Constants**:
- `UptimeTarget`: 95.0%
- `P99LatencyTarget`: 1 second
- `AlertThreshold`: 85.5% (90% of UptimeTarget)

**Types**:
- `SLAMetrics`: Current uptime, latency, violation count
- `ComplianceStatus`: COMPLIANT, AT_RISK, VIOLATED, UNKNOWN
- `ComplianceReport`: Human-readable compliance summary
- `SLAManager`: SLA tracking and reporting

### security

Security audit framework and compliance verification.

```go
import "decentralized.host/pkg/operator/security"

sm := security.NewSecurityManager()

// Initiate audit
audit, _ := sm.InitiateAudit("op_001", "dh1abcdef...")

// Record security checks
sm.RecordCheck("op_001", security.IDENTITY_VERIFICATION, true, "INFO", "Identity verified", "")
sm.RecordCheck("op_001", security.KEY_MANAGEMENT, true, "INFO", "Keys properly stored", "")
// ... more checks

// Check compliance level
sm.CertifyOperator("op_001") // Requires CERTIFIED level

// Generate audit report
status, _ := sm.GetStatus("op_001")
log.Printf("Compliance Level: %s, Score: %d/100", status.ComplianceLevel, status.OverallScore)
```

**Types**:
- `ComplianceLevel`: UNVERIFIED, BASIC, STANDARD, ADVANCED, CERTIFIED
- `AuditCheckType`: 10 check types (identity, key management, etc.)
- `AuditResult`: Individual check result with evidence
- `OperatorAudit`: Complete audit record with compliance level
- `SecurityManager`: Audit lifecycle management

### backup

Backup configuration validation and failover testing.

```go
import "decentralized.host/pkg/operator/backup"

bm := backup.NewBackupManager()

// Create backup configuration
config, _ := bm.CreateConfiguration("op_001", "daily", 30)

// Add failover nodes
bm.AddFailoverNode("op_001", "node_1", "us-east", true)
bm.AddFailoverNode("op_001", "node_2", "us-west", false)

// Record synchronization
bm.RecordSynchronization("op_001", "node_1", 500*time.Millisecond, true)
bm.RecordSynchronization("op_001", "node_2", 500*time.Millisecond, true)

// Execute failover test
test, _ := bm.ExecuteFailoverTest("op_001", "node_1")
if test.Success && test.RTO < backup.RTOTarget && test.RPO < backup.RPOTarget {
    log.Println("Failover test PASSED")
}

// Check production readiness
status, _ := bm.GetStatus("op_001")
if status.IsReadyForProduction {
    // Approved for mainnet deployment
}
```

**Constants**:
- `MinimumFailoverNodes`: 2 nodes
- `RTOTarget`: 5 minutes
- `RPOTarget`: 1 minute

**Types**:
- `BackupConfiguration`: Failover nodes and test results
- `FailoverNode`: Individual backup node with replication lag
- `FailoverTest`: Test execution record with RTO/RPO measurements
- `BackupManager`: Backup lifecycle and testing

## Integration Example

Complete operator qualification flow:

```go
package main

import (
    "time"
    "log"
    
    "decentralized.host/pkg/operator/backup"
    "decentralized.host/pkg/operator/nodes"
    "decentralized.host/pkg/operator/qualification"
    "decentralized.host/pkg/operator/security"
    "decentralized.host/pkg/operator/sla"
    "decentralized.host/pkg/operator/stake"
)

func main() {
    // Initialize managers
    qm := qualification.NewQualificationManager()
    sm := stake.NewStakingManager()
    nr := nodes.NewNodeRegistry()
    slaM := sla.NewSLAManager()
    secM := security.NewSecurityManager()
    bm := backup.NewBackupManager()

    // Register operator
    qm.RegisterOperator("op_001", "dh1a234567b234567c234567")
    
    // Deposit stake
    sm.DepositStake("op_001", 100_000_000)
    
    // Register 50+ nodes
    for i := 0; i < 50; i++ {
        nodeID := "dh1node" + string(rune('a'+i)) + "bcdef234567890"
        hardware := nodes.HardwareProfile{CPUCores: 4, MemoryGB: 16, DiskGB: 500, DiskType: "SSD"}
        network := nodes.NetworkProfile{IPv4Address: "10.0.0.100", Port: 8080, Bandwidth: 1000, Latency: 50*time.Millisecond, PacketLoss: 0.1}
        location := nodes.GeographicLocation{Region: "us-east", Country: "US"}
        nr.RegisterNode(nodeID, "op_001", hardware, network, location)
    }
    
    // Track work and accumulate reputation
    for i := 0; i < 100; i++ {
        qm.RecordWork("op_001", 100_000, 50*time.Millisecond)
    }
    
    // Register for SLA tracking
    slaM.RegisterOperator("op_001")
    slaM.RecordMetrics("op_001", 99, 1, []time.Duration{100*time.Millisecond})
    
    // Audit security
    secM.InitiateAudit("op_001", "dh1a234567b234567c234567")
    secM.RecordCheck("op_001", security.IDENTITY_VERIFICATION, true, "INFO", "Verified", "")
    secM.RecordCheck("op_001", security.KEY_MANAGEMENT, true, "INFO", "Valid", "")
    
    // Configure backup
    bm.CreateConfiguration("op_001", "daily", 30)
    bm.AddFailoverNode("op_001", "dh1backup1234567890ab", "us-west", false)
    bm.AddFailoverNode("op_001", "dh1backup1234567890ac", "eu-west", false)
    bm.ExecuteFailoverTest("op_001", "dh1backup1234567890ab")
    
    // Certify operator
    qm.Certify("op_001")
    
    // Report status
    status := qm.GetQualificationStatus()
    log.Printf("Total operators: %d, Certified: %d, Ready: %d",
        status.TotalOperators, status.CertifiedOperators, status.OperatorsReadyForMain)
}
```

## Testing

Run all operator package tests:

```bash
go test ./pkg/operator/... -v
```

Run specific package tests:

```bash
go test ./pkg/operator/qualification/... -v
go test ./pkg/operator/stake/... -v
go test ./pkg/operator/nodes/... -v
go test ./pkg/operator/sla/... -v
go test ./pkg/operator/security/... -v
go test ./pkg/operator/backup/... -v
```

## CLI Tool

Command-line interface for operator management:

```bash
# Build the CLI
go build -o operator-qualification ./cmd/operator-qualification

# Register operator
./operator-qualification register-operator -id op_001 -node dh1a234567b234567c234567

# Deposit stake
./operator-qualification stake deposit -id op_001 -amount 100000000

# Check node status
./operator-qualification nodes info -id op_001

# Register for SLA tracking
./operator-qualification sla register -id op_001

# View overall status
./operator-qualification status
```

## Compliance and Verification

All operators must meet these requirements for mainnet certification:

1. **Tier Requirement**: MASTER (10M uWork + 800 reputation)
2. **Stake Requirement**: 100M uWork minimum (ACTIVE status)
3. **Node Requirement**: 50+ nodes in 3+ regions
4. **SLA Requirement**: 95%+ uptime, <1s P99 latency
5. **Security Requirement**: CERTIFIED compliance level
6. **Backup Requirement**: Tested failover with passing results

## Performance Characteristics

- **Registration**: O(1)
- **Work Recording**: O(1) with tier advancement check
- **Compliance Verification**: O(n) for n operators
- **Health Check Recording**: O(1) per node
- **Audit Recording**: O(1) per check
- **Failover Test**: O(n) for n failover nodes

## Thread Safety

All managers use sync.RWMutex for concurrent access:
- Write operations (RegisterOperator, RecordWork, etc.) lock for writing
- Read operations (GetOperator, ListOperators, etc.) lock for reading
- Safe for concurrent use in multi-threaded applications

## Future Enhancements

- Dynamic stake requirements based on network load
- Operator reputation recovery mechanisms
- Work allocation algorithms
- Cross-operator audit requirements
- Physical-host failure domain verification
