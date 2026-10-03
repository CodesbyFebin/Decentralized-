# Extended Integration Test Suite for dh/v1

This document describes the advanced testing scenarios that extend the 10 mainnet launch gates with comprehensive edge case, load, stability, and multi-operator coordination tests.

## Overview

The extended test suite adds four major test files to the existing integration test infrastructure:

1. **Edge Case Tests** (`edge_cases_gates_4_10_test.go`)
2. **Load & Stress Testing** (`load_stress_test.go`)
3. **Long-Running Stability Tests** (`stability_test.go`)
4. **Multi-Operator Coordination Tests** (`multioperator_coordination_test.go`)

All tests follow the existing `TestHarness` pattern and integrate seamlessly with Gates 1-10 qualification tests.

---

## 1. Edge Case Test Suite (`edge_cases_gates_4_10_test.go`)

### Purpose
Test boundary conditions and failure scenarios beyond normal operation for Gates 4-10.

### Test Coverage

#### Gate 4: Workload Baseline (Placement) - Edge Cases
- **Placement Tie-Breaking**: 30 concurrent workloads with identical size on identical-capacity nodes
  - Verifies deterministic, reproducible tie-breaking
  - Tests concurrent placement without race conditions
  
- **Resource Exhaustion**: Test resource ledger model enforcement
  - Exact capacity matching
  - Over-capacity rejection
  - Reserve protection (system reserve never allocated)

#### Gate 5: Operator Onboarding - Edge Cases
- **Concurrent Registrations**: 100 parallel operator registrations
  - Verifies no race conditions in concurrent access
  - Tests duplicate ID rejection
  - Validates stake boundary conditions
  
- **Stake Boundaries**: Test minimum (100M) and maximum (10B) stake limits
  - Below/at/above minimum
  - Below/at/above maximum
  - Zero-stake rejection

#### Gate 6: Chaos Recovery - Edge Cases
- **Correlated Failures**: Cascade failure detection with 5 nodes
  - Injects 3 sequential failures
  - Verifies remaining nodes survive
  - Tests recovery attempt triggering
  
- **Byzantine Consensus**: 5-node consensus with 2 Byzantine nodes
  - Validates 3-node majority threshold
  - Tests single witness rejection
  - Orphan witness cannot override consensus

#### Gate 7: Observer-Driven Recovery - Edge Cases
- **Partition Recovery**: 4-node mesh network partitioning
  - Detects 1-way partition (single failed link)
  - Detects 2-way partition (bidirectional failure)
  - Tests partition healing and recovery
  
- **Multiple Observer Authority**: Independent observer action
  - 3 concurrent observers act independently
  - Unauthorized observer rejection
  - Action isolation and independence

#### Gate 8 & 9: Audit Trail & Certificates - Edge Cases
- **Audit Completeness**: Rapid state transitions with concurrent operations
  - All 5 state transitions logged (DESIRED→ADMITTED→EXECUTING→OBSERVED→VERIFIED)
  - 10 workers × 5 operations = 50 concurrent operations + state logs
  - Audit immutability verification
  
- **Certificate Expiry**: Certificate rotation under stress
  - Detect expiring certificates (< 2 hours remaining)
  - Rotate certificates for 5 nodes
  - Verify zero-downtime rotation
  - GC event tracking

#### Gate 10: Persistent State - Edge Cases
- **Rapid Restart Resilience**: 10 rapid restarts preserve identity
  - Node identity persists across restarts
  - Version numbers increment correctly
  - Work logs survive restarts
  - Policy configuration re-loaded

- **BLAKE3 Corruption Detection**: Data corruption detection
  - Detect bit-flip corruption via hash mismatch
  - Quarantine corrupted artifacts
  - Selective quarantine in multi-artifact scenarios

---

## 2. Load & Stress Testing (`load_stress_test.go`)

### Purpose
Exercise the system under sustained high throughput and resource constraints.

### Test Components

#### LoadTestMetrics Structure
Tracks:
- Total/successful/failed operations
- Latency statistics (min, max, mean)
- Memory usage (before/after)
- Throughput (ops/sec)
- Goroutine count

#### Test Scenarios

##### 1. 10x Baseline Throughput (`TestLoad_10xBaselineThroughput`)
- **50 workers** × **1000 ops each** = **50,000 total operations**
- **Throughput requirement**: ≥1000 ops/sec
- **Success rate**: ≥99%
- **Tests**: Policy evaluation + state update under high concurrency
- **Metrics**: Mean latency, throughput, memory growth

##### 2. 1000+ Concurrent Operators (`TestLoad_1000ConcurrentOperators`)
- **1000 operators** with **100 ops each** = **100,000 total ops**
- **Random operator selection** per operation
- **Activity distribution** across 80% of operators
- **Throughput requirement**: ≥5000 ops/sec
- **Tests**: Concurrent access patterns without lock contention

##### 3. Memory Leak Detection (`TestMemory_LeakDetection24Hour`)
- **30-second run** (production: 24 hours)
- **20 concurrent workers** spawning operations
- **10 memory samples** at regular intervals
- **Leak threshold**: <20% growth in test duration
- **GC tracking**: Verify cleanup cycles occurring
- **Baseline tracking**: Before and after memory states

##### 4. Network Bandwidth Saturation (`TestLoad_NetworkBandwidthSaturation`)
- **100 concurrent streams**
- **1MB per stream** in 8KB chunks
- **Throughput requirement**: ≥90 streams completed
- **Tests**: Network I/O under sustained load
- **Metrics**: Total MB transferred, per-stream latency

##### 5. Resource-Constrained Operation (`TestLoad_ResourceConstrainedOperation`)
- **Fixed 100MB memory budget**
- **10 workers** × **1000 ops** each
- **10KB blocks** allocated per operation
- **Graceful degradation** when budget exceeded
- **Success rate tracking** with memory constraints
- **Over-allocation prevention**

##### 6. Sustained Operation (`TestLoad_SustainedOperation`)
- **2-minute sustained load** (test mode)
- **20 continuous workers** without stopping
- **Throughput sampling** every 10 seconds
- **Stability threshold**: <30% variation between samples
- **Tests**: Long-running steady-state behavior

---

## 3. Long-Running Stability Tests (`stability_test.go`)

### Purpose
Validate system behavior under extended operation (7+ days in production).

### StabilityMetrics Structure
Tracks:
- Transaction counts (total, successful, failed)
- Audit entries logged
- Anomalies detected
- Recovery statistics
- Memory leak indicators
- State consistency

### Test Scenarios

##### 1. 7-Day Continuous Operation (`TestStability_7DaysContinuous`)
- **2 minutes** test duration (production: 7 days)
- **~1000 tx/sec** throughput
- **30 concurrent workers** generating transactions
- **Metrics**:
  - 95% success rate target
  - 100+ tx/sec minimum throughput
  - 99% state consistency for verified transactions
  - Audit trail completeness

##### 2. Periodic Chaos Injection (`TestStability_ChaosInjection`)
- **90-second test** duration
- **Chaos events every 10 seconds** (4 types):
  - Node crash
  - Network partition
  - Resource exhaustion
  - Clock skew
- **Event duration**: 0-5 seconds
- **Recovery rate**: ≥95%
- **Operation continuity** during chaos events

##### 3. Memory & Resource Trending (`TestStability_MemoryTrending`)
- **60-second continuous operation**
- **5 workers** with memory allocation
- **Memory samples** every 5 seconds
- **Leak detection**:
  - Track allocation growth rate
  - <1% growth per sample = normal operation
  - GC frequency monitoring
  - Goroutine count stability

##### 4. Event Log Anomaly Analysis (`TestStability_EventLogAnalysis`)
- **30-second test** with 5 components
- **Error rate target**: <2%
- **Warning rate target**: <5%
- **Anomaly detection**:
  - Component-wise error distribution
  - Outlier identification
  - One component >50% of errors = anomaly
- **Audit logging**: All operations recorded

##### 5. Distributed State Convergence (`TestStability_StateConvergence`)
- **5 nodes** with gossip protocol
- **30-second test**
- **State updates** with version tracking
- **Convergence metrics**:
  - >80% of nodes at max version (within 1 version lag)
  - Gossip-based propagation
  - Version number monotonicity

---

## 4. Multi-Operator Coordination Tests (`multioperator_coordination_test.go`)

### Purpose
Validate system behavior when 50+ operators interact with conflicting intentions.

### OperatorCoordinator Infrastructure
Manages:
- Operator registry with stakes and tiers
- Proposal creation and voting
- Consensus mechanics (supermajority thresholds)
- Stake slashing and disputes
- Comprehensive audit trail

### Test Scenarios

##### 1. 50+ Concurrent Operators (`TestMultiOperator_50ConcurrentOperators`)
- **50 operator concurrent registration**
- **Varying stake amounts** (100M + index×1M)
- **Error tracking** for race conditions
- **Registration verification**

##### 2. Consensus Under Disagreement (`TestMultiOperator_ConsensusDisagreement`)
- **30 operators** with split voting
  - 40% vote YES (smaller stakes)
  - 40% vote NO (larger stakes)
  - 20% abstain
- **Supermajority threshold**: 67%
- **Proposal failure** expected (no supermajority)
- **Vote aggregation** with stake weighting
- **Yes-vote percentage** calculation

##### 3. Stake Redistribution (`TestMultiOperator_StakeRedistribution`)
- **20 operators** × **100M stake**
- **Scenario 1**: Slash single operator 10%
- **Scenario 2**: Slash 4 operators 5% each
- **Scenario 3**: Verify stake conservation
  - Active stake + Disputed pool = Initial total
  - No loss or creation of stake

##### 4. Dispute Resolution (`TestMultiOperator_DisputeResolution`)
- **40 operators** with jury voting
- **Dispute proposal** against operator-1
- **70% vote threshold** for guilt
- **Guilty verdict**: 50% stake slash
- **Audit trail** for dispute tracking
- **Dispute closure** counting

##### 5. Cross-Operator Actions (`TestMultiOperator_CrossOperatorActions`)
- **25 operators** concurrently voting
- **5 proposals** created simultaneously
- **Vote distribution**:
  - 60% YES
  - 20% NO
  - 20% ABSTAIN
- **Concurrent proposal+voting** operations
- **Expected votes**: 125 total (5 proposals × 25 operators)
- **Audit entry** generation

##### 6. Operator Churn (`TestMultiOperator_OperatorChurn`)
- **30 initial operators**
- **30-second churn period**
- **Random operator joins** every 100ms
- **10 worker threads** maintain normal operations
- **Operator activity** during churn
- **Final operator count** > initial count

---

## Running the Tests

### Run all new tests:
```bash
go test -v ./tests/integration -run "EdgeCases|Load_|Stability_|MultiOperator_"
```

### Run specific test categories:
```bash
# Edge cases only
go test -v ./tests/integration -run "EdgeCases"

# Load tests only
go test -v ./tests/integration -run "Load_"

# Stability tests (excluding long-running)
go test -short -v ./tests/integration -run "Stability_"

# Multi-operator tests
go test -v ./tests/integration -run "MultiOperator_"
```

### Run with longer timeout for stability tests:
```bash
go test -v -timeout 5m ./tests/integration -run "Stability_"
```

---

## Integration with Gates 1-10

These tests complement rather than replace the mainnet launch gates:

| Gates    | Focus                          | New Tests Provided           |
|----------|--------------------------------|------------------------------|
| 1-3      | Bootstrap & topology           | (no new tests)               |
| 4        | Workload placement             | Edge case tie-breaking, exhaustion |
| 5        | Operator onboarding            | Concurrent registration, stake boundaries |
| 6        | Chaos recovery                 | Cascading failures, Byzantine consensus |
| 7        | Observer-driven recovery       | Partition recovery, multi-observer auth |
| 8-9      | Audit trail, recovery scenarios| State transitions, certificate expiry |
| 10       | Persistent state               | Rapid restarts, corruption detection |
| 11-20    | Edge cases (design phase)      | All 4 comprehensive test files |
| 21-32    | Chaos validation               | Integrated chaos injection test |

---

## Performance Baselines

### Load Testing
- **10x baseline**: 1000+ ops/sec, 99%+ success
- **1000 operators**: 5000+ ops/sec
- **Network saturation**: 90+ streams, MB/s tracking
- **Resource constrained**: Memory budget enforcement

### Stability Testing
- **7-day continuous**: 95%+ success rate, 100+ tx/sec
- **Chaos recovery**: 95%+ recovery rate
- **Memory leak**: <20% growth in test duration
- **State convergence**: >80% of nodes converged

### Multi-Operator Coordination
- **50+ concurrent operators**: All registered without error
- **Consensus voting**: Correct aggregate calculation
- **Stake conservation**: Active + Disputed = Total
- **Dispute resolution**: Automated slashing execution

---

## Future Enhancements

1. **Extended Duration**: Run 7-day stability tests in production
2. **Network Conditions**: Add latency, packet loss injection
3. **Byzantine Operator Testing**: Malicious operator actions
4. **Cross-Shard Coordination**: Multi-shard consensus tests
5. **Persistent Storage**: State recovery from disk/WAL
6. **TLS Certificate Expiry**: Production ACME integration
7. **Stake Redistribution Logic**: Dispute resolution economics
8. **Validator Rotation**: Dynamic validator set changes

---

## Test Harness Pattern

All tests follow the standard pattern:

```go
func TestCategory_Description(t *testing.T) {
    harness := NewTestHarness("Category-Description")
    harness.Start()
    
    // Test execution
    
    harness.ReportPass("check-name", "evidence")
    harness.ReportFail("check-name", "reason")
    
    if !harness.Finalize(t) {
        t.FailNow()
    }
}
```

This ensures:
- Consistent metrics collection
- Pass/fail tracking
- Duration measurement
- Error aggregation
- Integration with CI/CD

---

*Last Updated: 2026-10-03*
*Conformance: dh/v1 specification*
