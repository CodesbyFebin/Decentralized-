# Metrics Definition & Dashboard Schema

**Prepared:** 2026-10-04  
**Scope:** Operational metrics for Phase 6 go-live monitoring and Phase 7 optimization  
**Target Dashboard:** Real-time operator dashboard with 15+ core metrics  
**Metrics Architecture:** Prometheus scrape + OpenTelemetry tracing + custom aggregations

---

## Overview

Phase 6 go-live requires comprehensive operational metrics to track system health, operator performance, workload placement efficiency, and financial metrics. Dashboard displays real-time and historical metrics for operators, engineers, and steering committee oversight.

**Metrics Categories:**
1. **Availability Metrics** (SLA tracking): Uptime %, error rate, incident count
2. **Performance Metrics** (latency/throughput): Placement latency, scheduler throughput, replication lag
3. **Resource Metrics** (capacity): Node utilization, workload density, operator capacity
4. **Financial Metrics** (revenue/cost): Commission earned, stake locked, operator cost
5. **Compliance Metrics** (audit/security): Key rotation, audit findings, policy violations
6. **Operational Metrics** (incidents/support): MTTR, incident response SLA, escalation count

---

## Metrics Definitions

### 1. Availability Metrics

#### 1.1 Operator Uptime (Global)
**Definition:** Percentage of time operator's nodes are available (not down)  
**Formula:** `available_hours / total_hours_per_month`  
**Total Hours per Month:** 730 (365 days × 24 hours / 12 months)  
**Calculation Frequency:** Every 5 minutes (rolling window)  
**SLA Threshold:** BOOTSTRAP ≥95%, TRUSTED ≥98%, MASTER ≥99.5%  
**Granularity:** Per operator, per region, global

**Example:**
- Month 1: July 2026 (31 days = 744 hours)
- Available hours: 707 (37 hours down)
- Uptime %: 707 / 730 = 96.8%
- SLA status: PASS (>95% BOOTSTRAP target)

**Metric Name (Prometheus):** `operator_uptime_percent{operator_id, region}`  
**Data Type:** Gauge (0-100)  
**Retention:** 13 months (for tier progression verification)

#### 1.2 Node Availability (Per-Node)
**Definition:** Node up/down status (binary)  
**Detection Method:** TCP health check every 5 seconds to node port 8080  
**Timeout:** 15 seconds (3 failed probes = down)  
**Recovery Confirmation:** 2 successful probes = back up  
**Metric Name:** `node_available{operator_id, node_id, region}`  
**Data Type:** Binary (0=down, 1=up)  
**Resolution:** 5-second granularity

#### 1.3 Workload Error Rate
**Definition:** Failed workload placements / total placement attempts  
**Formula:** `failed_placements / total_placements`  
**Failed Placement Causes:**
- Insufficient capacity (CPU/memory/storage exhausted)
- Node policy rejection (local admission control denied)
- Network timeout (unreachable node)
- Invalid workload specification
**SLA Target:** <0.1% (99.9% successful placements)  
**Metric Name:** `workload_placement_error_rate{operator_id}`  
**Data Type:** Gauge (0-100)  
**Calculation:** Per 5-minute rolling window

#### 1.4 Incident Count (Per Operator)
**Definition:** Number of customer-impacting incidents  
**Incident Classification:**
- Critical: Complete operator downtime (all nodes down)
- High: >50% nodes down
- Medium: 10-50% nodes down
- Low: <10% nodes down, isolated workload failure
**Detection:** Automated alerts when nodes go down  
**Metric Name:** `incidents_total{operator_id, severity}`  
**Data Type:** Counter (only increases)  
**Aggregation:** Daily, weekly, monthly counts

---

### 2. Performance Metrics

#### 2.1 Placement Latency (p50/p95/p99)
**Definition:** Time from workload request to placement decision  
**Measurement Point:** Control plane processes placement request to node admission decision  
**SLA Targets:**
- p50: <10ms (typical case)
- p95: <15ms (95% of requests)
- p99: <30ms (99% of requests)
**Metric Name:** `workload_placement_latency_seconds{percentile, operator_id}`  
**Data Type:** Histogram buckets  
**Calculation Frequency:** Per 5-minute rolling window  
**Example:** Placement latency p95 = 12ms (2 ms away from SLA)

#### 2.2 Scheduler Throughput
**Definition:** Workload placements processed per second  
**Formula:** `placements_count / 60` (converted to per-second)  
**Target:** 100+ placements/second sustained  
**Metric Name:** `workload_placements_per_second{operator_id}`  
**Data Type:** Gauge  
**Calculation:** Per 1-minute window  
**Example:** Peak throughput = 150 placements/sec (single operator)

#### 2.3 Replication Lag (Per Region Pair)
**Definition:** Delay between primary write and replica acknowledgment  
**SLA Target:** <1 second p99 (Phase 7B requirement)  
**Measurement Point:** From control plane ack to replica state machine applied  
**Metric Name:** `replication_lag_seconds{percentile, source_region, target_region}`  
**Data Type:** Histogram  
**Calculation Frequency:** Per 5-minute rolling window  
**Regions:** US-WEST → EU-CENTRAL, US-WEST → ASIA-EAST, etc.  
**Example:**  
- US-WEST → EU-CENTRAL: p99 lag = 450ms
- US-WEST → ASIA-EAST: p99 lag = 550ms

#### 2.4 Control Plane Latency (Multi-Region)
**Definition:** Raft consensus latency for topology updates  
**SLA Target:** <100ms p95 intra-region, <500ms p99 inter-region  
**Measurement Point:** Quorum decision time (2/3 regions must acknowledge)  
**Metric Name:** `raft_consensus_latency_seconds{percentile, operation_type}`  
**Data Type:** Histogram  
**Example:** Operator registration (typical): p95 = 45ms

---

### 3. Resource Metrics

#### 3.1 Node Utilization (Compute)
**Definition:** Percentage of node CPU/memory/storage in use  
**Formula per Resource:**
- CPU: `used_vcpu / total_vcpu × 100`
- Memory: `used_memory / total_memory × 100`
- Storage: `used_storage / total_storage × 100`
**SLA Consideration:** >90% utilization may impact SLA (indicate capacity constraint)  
**Metric Name:** `node_utilization_percent{operator_id, node_id, resource_type}`  
**Data Type:** Gauge (0-100)  
**Calculation Frequency:** Per 30 seconds  
**Example:** Node utilization breakdown:
- CPU: 65%
- Memory: 55%
- Storage: 42%
- Average: 54%

#### 3.2 Workload Density
**Definition:** Number of running workloads per node  
**Formula:** `count(workloads) / count(nodes)`  
**High Density Issues:** May impact latency, incident response time  
**Target:** 10-20 workloads per node (optimal balance)  
**Metric Name:** `workload_density{operator_id, node_id}`  
**Data Type:** Gauge  
**Calculation Frequency:** Real-time  
**Example:** 100 workloads / 5 nodes = 20 workloads/node (target)

#### 3.3 Operator Capacity Tracking
**Definition:** Used vs. available capacity across all operator nodes  
**Metrics:**
- Total capacity provisioned (vCPU, GB RAM, GB storage)
- Currently allocated (sum of all running workloads)
- Available capacity (total - allocated)
- Percent utilization (allocated / total)
**Metric Name:** `operator_capacity{operator_id, resource_type, state}`  
**Data Type:** Gauge  
**Example (BOOTSTRAP operator with 50 nodes):**
- Total: 5,000 vCPU, 100 GB RAM, 500 GB storage
- Allocated: 3,250 vCPU, 65 GB RAM, 200 GB storage
- Available: 1,750 vCPU, 35 GB RAM, 300 GB storage
- Utilization: 65% CPU, 65% memory, 40% storage

#### 3.4 Multi-Region Resource Distribution
**Definition:** How capacity is distributed across regions  
**Metrics per Region:**
- Nodes deployed in region
- Total capacity in region
- Current utilization in region
**Used for:** Load balancing decisions, failure impact analysis  
**Metric Name:** `operator_capacity_by_region{operator_id, region, resource_type}`  
**Data Type:** Gauge  
**Example:**
- US-WEST: 2,500 vCPU, 85% utilized
- EU-CENTRAL: 1,500 vCPU, 60% utilized
- ASIA-EAST: 1,000 vCPU, 45% utilized

---

### 4. Financial Metrics

#### 4.1 Monthly Commission Earned (Operator)
**Definition:** Total commission earned by operator in current calendar month  
**Formula:** `sum(base_commission + SLA_bonus + tier_multiplier) for current month`  
**Update Frequency:** Every hour (aggregates workload usage data)  
**Metric Name:** `operator_commission_usd{operator_id, tier}`  
**Data Type:** Gauge (accumulated per month)  
**Example:** Operator "bootstrap-1" earned $2,847 in July 2026  
**Breakdown:**
- Base commission: $2,700
- SLA bonus (5%): +$135
- Tier multiplier (BOOTSTRAP=1.0x): $2,835
- Sponsor cut (0% for BOOTSTRAP): $0
- Net commission: $2,835

#### 4.2 Estimated Monthly Revenue (Projection)
**Definition:** Projected monthly revenue based on current utilization  
**Formula:** `(ytd_commission / ytd_days) × days_in_month`  
**Purpose:** Help operators forecast earnings  
**Update Frequency:** Daily (recalculate as utilization changes)  
**Metric Name:** `operator_projected_revenue_usd{operator_id}`  
**Data Type:** Gauge  
**Caveats:** Assumes similar utilization for rest of month (volatile)  
**Example:** If operator earned $1,400 in 15 days, projected = $2,800 for month

#### 4.3 Stake Locked (Operator)
**Definition:** Total uWork stake locked by operator  
**Metric Name:** `operator_stake_locked_uwork{operator_id, tier}`  
**Data Type:** Gauge  
**Example:** Operator "bootstrap-1" has 100M uWork locked (BOOTSTRAP tier)

#### 4.4 Operator Cost (Estimated)
**Definition:** Estimated infrastructure cost to run operator nodes  
**Calculation:** Cost varies by region, node type (physical vs VM)  
**Estimation Formula:** `per_node_cost × count(nodes)`  
**Per-Node Cost Examples:**
- Physical server (colo): $200-300/month
- VM on public cloud: $50-150/month
- Custom hardware: $100-250/month
**Metric Name:** `operator_estimated_cost_usd{operator_id, node_type}`  
**Data Type:** Gauge  
**Used for:** ROI calculation, profitability tracking  
**Example:** 50 nodes × $200/month (colo) = $10,000/month cost

#### 4.5 Operator Profitability (Net)
**Definition:** Estimated profit after costs  
**Formula:** `commission_earned - estimated_cost`  
**Metric Name:** `operator_net_profit_usd{operator_id}`  
**Data Type:** Gauge  
**Example:** $2,835 commission - $10,000 cost = -$7,165 (month 1, expected)  
**Note:** ROI takes 18-24 months as utilization ramps up

#### 4.6 SLA Credit Earned (Bonus)
**Definition:** Additional commission from SLA bonus (95%+ uptime)  
**Formula:** `base_commission × SLA_bonus_percent`  
**Metric Name:** `operator_sla_bonus_usd{operator_id}`  
**Data Type:** Gauge  
**Update Frequency:** Monthly (on billing date)  
**Example:** Base $2,700 × 5% (97% uptime) = $135 bonus

---

### 5. Compliance Metrics

#### 5.1 Key Rotation Status
**Definition:** Ed25519 key age and rotation compliance  
**Metrics:**
- Current key age (days since generation)
- Days until next rotation required (90-day enforcement)
- Overlap period remaining (if rotation in progress)
**Metric Name:** `operator_key_rotation_days_until_next{operator_id}`  
**Data Type:** Gauge  
**Alert Trigger:** <7 days until rotation due  
**Example:** Key generated 83 days ago → 7 days until rotation required

#### 5.2 Audit Findings Count
**Definition:** Count of security audit findings (per severity)  
**Categories:** Critical, High, Medium, Low  
**Metric Name:** `security_audit_findings_total{operator_id, severity, status}`  
**Data Type:** Counter  
**Status Values:** Open, In-Progress, Resolved  
**SLA for Remediation:**
- Critical: 48 hours
- High: 1 week
- Medium: 2 weeks
- Low: 30 days  
**Example:**
- Critical: 0 findings
- High: 2 findings (1 resolved, 1 in-progress)
- Medium: 3 findings (1 resolved, 2 open)

#### 5.3 Compliance Item Completion
**Definition:** Operator progress on 40+ compliance checklist items  
**Metric Name:** `compliance_items_completed{operator_id}`  
**Data Type:** Gauge (0-40)  
**SLA for Completion:** Before go-live (100% required)  
**Example:** 38/40 items completed (2 pending: firewall rules, backup testing)

#### 5.4 Policy Violations (Per Node)
**Definition:** Admission policy rejections (workload denied by local policy)  
**Metric Name:** `policy_violations_total{operator_id, node_id, reason}`  
**Data Type:** Counter  
**Violation Reasons:**
- Unauthorized operator
- Insufficient resources
- Policy constraint violation
- Invalid workload signature
**Alert Trigger:** >10 violations in 5-minute window  
**Example:** Node rejects workload because operator not authorized (should not happen in normal operation)

#### 5.5 Audit Log Health
**Definition:** BLAKE3 audit chain integrity verification  
**Metrics:**
- Log entries total (count)
- Hash chain verification status (pass/fail)
- Gaps detected (missing entry ranges)
- Retention period (days stored)
**Metric Name:** `audit_log_health{operator_id, status}`  
**Data Type:** Gauge + metadata  
**Alert Trigger:** Hash chain verification fails  
**Example:** 50,000 audit entries, chain verified OK, 91-day retention

---

### 6. Operational Metrics

#### 6.1 Mean Time to Recovery (MTTR)
**Definition:** Average time to recover from incident (from detection to resolution)  
**Formula:** `sum(recovery_time) / count(incidents)`  
**SLA Target:** <1 hour (typical incident)  
**Metric Name:** `operator_mttr_minutes{operator_id, severity}`  
**Data Type:** Gauge  
**Calculation Frequency:** Rolling 30-day window  
**Example:**
- Critical incidents: 35 minutes average MTTR
- High severity: 25 minutes average MTTR
- Medium: 15 minutes average MTTR

#### 6.2 Incident Response SLA Compliance
**Definition:** Percentage of incidents responded to within SLA  
**SLA Response Time:**
- Critical: 30 minutes
- High: 1 hour
- Medium: 4 hours  
**Formula:** `incidents_within_sla / total_incidents × 100`  
**Metric Name:** `incident_response_sla_compliance_percent{operator_id}`  
**Data Type:** Gauge (0-100)  
**Target:** 100% SLA compliance  
**Example:** 18 incidents / 20 total = 90% SLA compliance (2 late responses)

#### 6.3 Support Ticket Volume
**Definition:** Count of support tickets per operator per month  
**Metric Name:** `support_tickets_total{operator_id, status}`  
**Data Type:** Counter  
**Status Values:** Open, In-Progress, Resolved, Closed  
**Used for:** Operator success tracking (spike = unhappy operator)  
**Example:** 12 tickets total (8 resolved, 3 in-progress, 1 open)

#### 6.4 Escalation Count
**Definition:** Tickets escalated to engineering team  
**Metric Name:** `support_escalations_total{operator_id, reason}`  
**Data Type:** Counter  
**Escalation Reasons:**
- Bug in control plane
- Unusual SLA behavior
- Security incident
- Billing dispute  
**Alert Trigger:** >5 escalations per operator in 1 month  
**Example:** 2 escalations this month (1 SLA bug, 1 billing dispute)

#### 6.5 Operator Satisfaction (NPS)
**Definition:** Net Promoter Score (monthly survey)  
**Scale:** -100 to +100  
**Calculation:** `(Promoters% - Detractors%) × 100`  
**Target:** >50 NPS (healthy satisfaction)  
**Metric Name:** `operator_nps_score{operator_id}`  
**Data Type:** Gauge  
**Collection Method:** Monthly email survey  
**Example:** 15 responses: 10 promoters (67%), 2 neutral, 3 detractors (20%) → NPS = 47

---

## Metric Collection & Storage

### Prometheus Configuration
```yaml
# prometheus.yml
global:
  scrape_interval: 30s  # Scrape every 30 seconds
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'decentralized-host-metrics'
    static_configs:
      - targets: ['localhost:8080', 'localhost:8081']
    relabel_configs:
      - source_labels: [__address__]
        target_label: instance

# Retention
retention: 13 months  # Keep SLA-relevant data for tier progression
```

### OpenTelemetry Tracing
- Trace placement requests (end-to-end latency)
- Trace replication writes (lag measurement)
- Trace control plane operations (consensus latency)
- Sampling: 100% for critical paths, 10% for high-volume paths

### Data Storage
- **Real-time:** Prometheus in-memory (last 13 months)
- **Long-term archive:** BigQuery or S3 (7+ years for compliance)
- **Historical queries:** Grafana with Prometheus datasource

---

## Dashboard Layout

### Operator Dashboard (Main View)
**4-section layout (visible above fold):**

1. **SLA Status** (top-left)
   - Current month uptime %
   - Target % (95/98/99.5 depending on tier)
   - Trend (↑↓ compared to last month)
   - Days remaining in month

2. **Revenue Summary** (top-right)
   - Commission earned this month
   - Projected month total (if trend continues)
   - Last month comparison (% change)
   - YTD total

3. **Node Status** (bottom-left)
   - Nodes up / total nodes
   - Resource utilization (CPU, memory, storage)
   - Most loaded node
   - Least loaded node

4. **Alerts & Incidents** (bottom-right)
   - Active incidents (count + severity)
   - SLA warnings (days to demotion if <95%)
   - Key rotation status (days until due)
   - Upcoming maintenance windows

### Detailed Metrics Tabs

**Tab 1: Availability**
- Uptime % timeline (last 30 days)
- SLA achievement vs target
- Incident timeline (when downtime occurred)
- Error rate trend (workload placement failures)

**Tab 2: Performance**
- Placement latency (p50/p95/p99 over 30 days)
- Scheduler throughput (placements/sec)
- Replication lag (per region pair, if multi-region)
- Control plane latency (consensus time)

**Tab 3: Capacity**
- Node utilization distribution (CPU, memory, storage)
- Workload density per node (scatter plot)
- Capacity by region (if multi-region)
- Projection: capacity exhaustion date (if trend continues)

**Tab 4: Financial**
- Commission breakdown (base + bonus + multiplier)
- Monthly trend (commission over 12 months)
- Cost estimate vs revenue (profitability chart)
- Stake locked status and withdrawal options

**Tab 5: Compliance**
- Key rotation countdown (days until due)
- Audit findings status (by severity)
- Compliance checklist progress (X/40 items)
- Audit log integrity status

**Tab 6: Incidents**
- Incident log (date, severity, duration, MTTR)
- MTTR trend (improving or declining?)
- SLA impact per incident
- Support ticket summary (open/resolved)

**Tab 7: Financial Details**
- Workload breakdown (top 10 workloads by revenue)
- Sponsor relationship (if TRUSTED, shows who it sponsors)
- Billing history (last 12 months, PDF download)
- Settlement ledger (detailed monthly breakdown)

---

## Alerting Rules

### Critical Alerts (Trigger Page-On-Call)
- Operator uptime <95% (for 1+ hour)
- Node unavailability >50% of nodes
- Control plane unavailability (can't reach Raft quorum)
- Security audit failure (critical findings)
- Key rotation overdue (>90 days old)

### High-Severity Alerts (Notify Slack #operator-support)
- Workload placement error rate >0.5%
- Operator uptime <98% (for 4+ hours)
- Replication lag >2 seconds (p99)
- Incident response SLA miss (>30 min for critical)
- Support escalation ratio >20% (too many escalations)

### Medium-Severity Alerts (Notify Operator via Email)
- Key rotation due within 7 days
- Audit findings remain unresolved >SLA
- Capacity utilization >80%
- SLA demotion warning (day 15 if <95%)
- Revenue below trend (utilization declining)

### Low-Severity Alerts (Dashboard Widget Only)
- Node maintenance window scheduled
- New feature available for upgrade
- Operator tier progression eligible
- Sponsorship opportunity (TRUSTED needs sponsors)

---

## Metrics API (For Programmatic Access)

### REST Endpoints

```
GET /api/v1/operators/{operator_id}/metrics/sla
Returns: { uptime_percent, incidents, error_rate, placement_latency }

GET /api/v1/operators/{operator_id}/metrics/financial
Returns: { commission_earned, estimated_cost, net_profit, sla_bonus }

GET /api/v1/operators/{operator_id}/metrics/capacity
Returns: { total_capacity, allocated, available, utilization_percent }

GET /api/v1/operators/{operator_id}/metrics/incidents
Returns: [ { incident_id, severity, duration, start_time, resolution_time } ]

GET /api/v1/operators/{operator_id}/audit-log
Returns: [ { timestamp, event, data, blake3_hash, status } ]
```

### GraphQL Query Example
```graphql
query OperatorMetrics($id: ID!) {
  operator(id: $id) {
    id
    tier
    sla {
      uptimePercent
      targetPercent
      incidents { id severity duration }
    }
    financial {
      commissionEarned
      estimatedCost
      netProfit
    }
    capacity {
      nodes { id utilization region }
      totalCapacity { cpu memory storage }
    }
  }
}
```

---

## Success Criteria

### Metrics Accuracy
- [ ] Uptime calculation audited against third-party (±0.5% tolerance)
- [ ] Commission billing accurate to cent (0% variance)
- [ ] Latency measurements within ±5ms of actual (spot-check sample)

### Dashboard Responsiveness
- [ ] Load time <2 seconds (90th percentile)
- [ ] Metric update latency <5 minutes (most recent data)
- [ ] No stale data older than 1 hour

### Alert Accuracy
- [ ] <5% false-positive rate (alerts that don't need response)
- [ ] <1% false-negative rate (real incidents that don't alert)
- [ ] Alert message clearly states problem and recommended action

### Operator Adoption
- [ ] >80% of operators accessing dashboard weekly
- [ ] >50% using programmatic API (for integration with their systems)
- [ ] >90% satisfaction with metric accuracy and dashboard usefulness

---

**Metrics Definition - Decentralized.Host**  
**Prepared:** 2026-10-04  
**Status:** Ready for Oct 6 dashboard prototype development  
**Launch Date:** Oct 12, 2026 (with Phase 6 go-live)
