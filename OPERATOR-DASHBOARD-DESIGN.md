# Operator Dashboard Design
## Phase 6 Launch Dashboard & Phase 7 Analytics Roadmap

**Prepared:** October 4, 2026  
**Purpose:** Define operator-facing dashboard for real-time SLA monitoring, cost tracking, and placement diagnostics  
**Timeline:** Phase 6 launch (Oct 12) minimal viable dashboard + Phase 7 enhanced analytics  
**Design Principles:** Real-time visibility, self-service diagnostics, actionable alerts  

---

## Phase 6 Launch Dashboard (MVP - Oct 12)

### Overall Dashboard Layout

```
┌─────────────────────────────────────────────────────────────────┐
│ DECENTRALIZED.HOST OPERATOR DASHBOARD                           │
│ Operator: Acme Corp (BOOTSTRAP) | 42 Nodes | 850 uWork Stake   │
└─────────────────────────────────────────────────────────────────┘

┌──────────────────────┬──────────────────────┬──────────────────────┐
│  SLA STATUS          │  REVENUE SUMMARY     │  NODE STATUS         │
├──────────────────────┼──────────────────────┼──────────────────────┤
│ ✅ Uptime: 99.95%    │ 💰 Commission:       │ 📊 Nodes: 42/42      │
│ ⚡ Latency: 12ms p95 │    $8,427 this month │ ⚙️  Healthy: 40      │
│ 📦 Error Rate: 0.02% │ 📈 Est. Revenue:     │ ⚠️  Warning: 2       │
│ 🎯 Placements: 8,542 │    $12,500 this Q    │ 🔴 Offline: 0        │
└──────────────────────┴──────────────────────┴──────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│ ALERTS & INCIDENTS                                               │
├──────────────────────────────────────────────────────────────────┤
│ 🟡 [1h ago] Scheduler latency elevated: 18ms (target <15ms)     │
│ 🔵 [4h ago] Node backup-node-12 offline, recovered              │
│ ℹ️  [6h ago] Policy cache hit rate: 89% (excellent)             │
└──────────────────────────────────────────────────────────────────┘

TABS: [Overview] [Metrics] [Placements] [Nodes] [Cost] [Audit] [Settings]
```

### Tab 1: Overview (Default View)

**Purpose:** 15-second SLA status check  
**Refresh:** Every 30 seconds

**Section 1A: SLA Status Cards (Top Row)**
```
┌─────────────────────┬─────────────────────┬─────────────────────┐
│ UPTIME              │ LATENCY (p95)       │ ERROR RATE          │
├─────────────────────┼─────────────────────┼─────────────────────┤
│ 99.95%              │ 12ms                │ 0.02%               │
│ Target: ≥99.0%      │ Target: <15ms       │ Target: <0.1%       │
│ ✅ PASSING          │ ✅ PASSING          │ ✅ PASSING          │
│                     │                     │                     │
│ Trend: ↗ +0.1% ↑   │ Trend: → +1ms       │ Trend: ↘ -0.01% ↓  │
└─────────────────────┴─────────────────────┴─────────────────────┘
```

**Section 1B: Revenue & Stake (Middle Row)**
```
┌─────────────────────────────┬─────────────────────────────┐
│ COMMISSION (This Month)     │ STAKE & TIER STATUS         │
├─────────────────────────────┼─────────────────────────────┤
│ $8,427 earned               │ 42,000 uWork locked         │
│ Base commission: $6,500     │ Tier: BOOTSTRAP (≥100M)     │
│ SLA bonus (5%): $1,100      │ Next tier: TRUSTED          │
│ Tier multiplier (1x): $827  │ Unlock at: 200M uWork       │
│                             │ Months to unlock: ~18       │
│ 📈 Trend: +$527 vs Sept     │                             │
└─────────────────────────────┴─────────────────────────────┘
```

**Section 1C: Recent Alerts (Bottom)**
```
┌──────────────────────────────────────────────────────────┐
│ ACTIVE ALERTS & INCIDENTS                                │
├──────────────────────────────────────────────────────────┤
│ [1h ago] 🟡 Scheduler latency elevated                   │
│          Latency: 18ms (target <15ms) | 5min duration   │
│          [View Details] [Acknowledge]                    │
│                                                          │
│ [4h ago] 🔵 Node recovery complete                       │
│          Node: backup-node-12 | Status: ✅ Online       │
│          [View Timeline]                                 │
└──────────────────────────────────────────────────────────┘
```

---

### Tab 2: Metrics (Detailed Performance)

**Purpose:** Deep-dive into 26 core metrics  
**Refresh:** Every 60 seconds  

**2A: Performance Metrics**
```
AVAILABILITY
├─ Uptime (% operating as expected)
│  └─ Last 7d: 99.94%, Last 30d: 99.92%, Last 90d: 99.88%
│  └─ SLA: ≥99.0% | ✅ PASSING | Trend: → stable
│
├─ Node Availability (TCP health OK)
│  └─ Healthy nodes: 40/42 (95.2%)
│  └─ SLA: ≥95% | ✅ PASSING | Offline: backup-node-09 (2h)
│
└─ Workload Error Rate (errors/total)
   └─ Last 1h: 0.01%, Last 24h: 0.02%, Last 30d: 0.015%
   └─ SLA: <0.1% | ✅ PASSING | Trend: ↘ improving

PERFORMANCE
├─ Placement Latency (time to decision)
│  └─ p50: 8ms | p95: 12ms | p99: 16ms | Max: 34ms
│  └─ SLA: <15ms p95 | ✅ PASSING | Histogram: [shows dist]
│
├─ Scheduler Throughput (placements/sec)
│  └─ Current: 87 placements/sec (peak 95, min 42)
│  └─ Target: 100+ p/sec | ⚠️  NEEDS MONITORING | Trend: →
│
└─ Control Plane Latency (Raft consensus)
   └─ p50: 2ms | p95: 5ms | p99: 12ms
   └─ SLA: <20ms p95 | ✅ PASSING | Trend: → stable

RESOURCE UTILIZATION
├─ Cluster CPU: 52% (21.8 cores of 42 available)
├─ Cluster Memory: 68% (6.2 GB of 9.2 GB)
├─ Node Density: 12 workloads/node (optimal 10-20)
└─ Storage: 34% (340 GB of 1 TB)
```

**2B: Financial Metrics**
```
COMMISSION
├─ Today: $142 earned
├─ This week: $987 earned
├─ This month: $8,427 earned (on pace for $12,500 Q4)
├─ Breakdown:
│  ├─ Base commission: $6,500 (monthly flat)
│  ├─ SLA bonus (5%): $1,100 (5% of base for 99%+ uptime)
│  └─ Tier multiplier (1x): $827 (1x for BOOTSTRAP tier)
│
└─ Projected Q4: $12,500

COSTS & NET INCOME
├─ Est. Infrastructure Cost: $4,200/month (node provisioning)
├─ Net Income: +$4,227/month (commission - costs)
├─ ROI: 101% (earning 2x infrastructure cost)
└─ Stake Investment: 42,000 uWork ($4,200 at launch price)
```

---

### Tab 3: Placements (Diagnostic Drill-Down)

**Purpose:** Troubleshoot placement failures, analyze placement patterns  
**Refresh:** Every 30 seconds (real-time)

**3A: Placement Summary**
```
PLACEMENT ACTIVITY (Last 24 Hours)
├─ Total placements: 8,542
├─ Successful: 8,523 (99.78%)
├─ Failed (resource unavailable): 15 (0.18%)
├─ Failed (policy rejected): 2 (0.02%)
├─ Rejected (no qualifying nodes): 2 (0.02%)
│
FAILURE ANALYSIS
├─ Resource unavailable (15 failures)
│  ├─ Reason: Node ran out of CPU/memory
│  ├─ Nodes: node-25 (7x), node-38 (5x), node-41 (3x)
│  ├─ Recommendation: "Consider adding nodes or consolidating"
│  └─ [View Details]
│
├─ Policy rejected (2 failures)
│  ├─ Workload: ml-training-v3 (2x rejection)
│  ├─ Policy: "GPU quota exceeded for ML team"
│  ├─ Recommendation: "Request GPU quota increase or redistribute load"
│  └─ [View Details]
│
└─ No qualifying nodes (2 failures)
   ├─ Workload: data-processor (2x rejection)
   ├─ Requirement: "Intel CPU (AVX-512)" | Available: "AMD nodes only"
   ├─ Recommendation: "Add Intel nodes to cluster or change requirement"
   └─ [View Details]

PLACEMENT TIME DISTRIBUTION
├─ <5ms (fast): 4,230 placements (49.5%)
├─ 5-10ms (optimal): 3,120 placements (36.5%)
├─ 10-15ms (acceptable): 950 placements (11.1%)
├─ 15-30ms (slow): 200 placements (2.3%)
├─ >30ms (very slow): 42 placements (0.5%)
│
└─ Outliers (>30ms): [Shows list of 5 slowest placements with details]
```

**3B: Slow Placement Drill-Down**
```
SLOWEST PLACEMENTS (Last 24 Hours)
┌──────────────────────────────────────────────────────┐
│ 1. Workload: data-processor-28 | Time: 34ms | ⚠️      │
│    Node: node-41 | Decision tree depth: 38 steps    │
│    CPU check: ✅ (8 cores available)                 │
│    Memory check: ✅ (2.1 GB available)               │
│    Policy eval: ✅ (8ms - complex multi-rule check)  │
│    → Root cause: Policy evaluation slow              │
│    → Recommendation: "Cache policy result"           │
│    [Drill into decision log]                         │
│                                                      │
│ 2. Workload: ml-training-v1 | Time: 31ms | ⚠️        │
│    Node: node-09 | Decision tree depth: 42 steps    │
│    CPU check: ✅ (4 cores available)                 │
│    Memory check: ✅ (1.8 GB available)               │
│    GPU check: ✅ (1 GPU available)                   │
│    Topology check: ❌ (anti-affinity failed on 35)   │
│    → Queried 35 nodes before finding match           │
│    → Recommendation: "Improve topology search"       │
│    [Drill into decision log]                         │
└──────────────────────────────────────────────────────┘
```

---

### Tab 4: Nodes (Cluster Health)

**Purpose:** Monitor individual node health and capacity  
**Refresh:** Every 30 seconds

**4A: Node List (with Quick Status)**
```
CLUSTER NODES (42 Total)
┌─────┬────────────────┬────────────┬──────────┬──────────┬─────────┐
│ ID  │ STATUS         │ CPU Usage  │ Memory   │ Workload │ Issues  │
├─────┼────────────────┼────────────┼──────────┼──────────┼─────────┤
│ n01 │ ✅ Online      │ 42/42 (1%) │ 0.8/9.2  │ 2        │ None    │
│ n02 │ ✅ Online      │ 8/42 (19%) │ 1.2/9.2  │ 3        │ None    │
│ n03 │ ✅ Online      │ 22/42(52%) │ 4.1/9.2  │ 6        │ None    │
│ ... │ ... (37 more)  │            │          │          │         │
│ n25 │ ⚠️  Degraded    │ 41/42(98%) │ 8.9/9.2  │ 12       │ CPU hot │
│ n38 │ ⚠️  Degraded    │ 39/42(93%) │ 8.5/9.2  │ 11       │ Mem hot │
│ n41 │ ✅ Online      │ 28/42(67%) │ 6.2/9.2  │ 8        │ None    │
│ n09 │ 🔴 Offline     │ —          │ —        │ —        │ No beat │
└─────┴────────────────┴────────────┴──────────┴──────────┴─────────┘

Filters: [All] [Healthy] [Degraded] [Offline] [High CPU] [High Memory]
```

**4B: Node Detail (Click node-41)**
```
NODE: node-41
├─ Status: ✅ Online (Last heartbeat: 2 sec ago)
├─ Uptime: 24d 12h 34m (no restarts since launch)
│
├─ CAPACITY
│  ├─ CPU: 42 cores (28 used, 14 available, 67% utilization)
│  ├─ Memory: 9.2 GB (6.2 used, 3.0 available, 67% utilization)
│  ├─ Storage: 1 TB (340 GB used, 660 GB available, 34% utilization)
│  └─ Network: 10Gbps (avg 200 Mbps out, 150 Mbps in)
│
├─ WORKLOADS (8 currently running)
│  ├─ data-processor-28 (2 cores, 1.2 GB, 10h running)
│  ├─ cache-worker-15 (1 core, 512 MB, 14h running)
│  ├─ ... (6 more workloads)
│
├─ ALERTS & ISSUES
│  ├─ [1h ago] 🟡 Latency spike: One placement took 34ms
│  │           Likely cause: Policy evaluation complexity
│  │           Action: Monitor for recurrence
│  │
│  └─ [2d ago] 🔵 Disk warning: 25% full
│             Current: 34% full (recovering)
│
└─ ACTIONS
   ├─ [Drain & Reboot]
   ├─ [Force Offline]
   ├─ [View Logs]
   ├─ [SSH Access]
   └─ [Certificate Info]
```

---

### Tab 5: Cost (Financial Breakdown)

**Purpose:** Understand commission structure and cost drivers  
**Refresh:** Daily (cost data updates once/day)

**5A: Commission Breakdown**
```
COMMISSION STRUCTURE (This Month)

Base Commission: $6,500
├─ Monthly flat: $6,500 (all BOOTSTRAP operators)
│
SLA Bonus: $1,100 (5% of base for ≥99% uptime)
├─ Your uptime: 99.95% (exceeds 99%)
├─ Bonus: 5% × $6,500 = $325
├─ + Bonus history adjustment: +$775 (from prior months)
│  
Tier Multiplier: $827 (1x for BOOTSTRAP, 1.5x for TRUSTED)
├─ Your tier: BOOTSTRAP
├─ Multiplier: 1x
├─ Next tier: TRUSTED (unlock at 200M uWork)
├─ Timeline to unlock: ~18 months at current stake rate
│
Estimated Q4 Commission: $12,500
├─ Oct: $8,427 (10 days so far)
├─ Nov: +$4,073 (projected)
├─ Dec: +$0 (TBD)
│
Total Year 1 Projected: $50,000+
└─ (Based on current tier and SLA performance)

HISTORICAL COMMISSION
├─ Oct 12-31 (11 days): $2,800
├─ Sept (full month): $5,927
├─ Aug (full month): $5,412
└─ Q3 2026 total: $11,339
```

**5B: Cost Attribution**
```
ESTIMATED INFRASTRUCTURE COST (Your Cluster)

Monthly Cost Breakdown:
├─ Node Provisioning: $2,800/month (42 nodes @ $67/node)
├─ Bandwidth: $300/month (average 3-5 TB/month egress)
├─ Storage: $140/month (340 GB persistent @ $0.41/GB/month)
├─ Support: $50/month (included in platform fee)
│
Total Monthly Cost: $3,290
├─ Commission: $8,427 (this month)
├─ Minus Cost: -$3,290
├─ Net Income: +$5,137
│
ROI Ratio: 2.56x (earning 2.56x your infrastructure cost)

TIER COST COMPARISON
├─ BOOTSTRAP: $3,290/month + 100M uWork lock-in
├─ TRUSTED: $2,900/month + 200M uWork lock-in (12% cheaper nodes)
└─ MASTER: $2,400/month + 500M uWork lock-in (27% cheaper nodes)
   → Recommendation: Consider moving to TRUSTED tier in 18 months
```

---

### Tab 6: Audit (Compliance & Security)

**Purpose:** Security & compliance verification  
**Refresh:** Every 6 hours (compliance state changes infrequently)

**6A: Audit Trail & Compliance**
```
AUDIT & COMPLIANCE STATUS

POLICY ENFORCEMENT
├─ RBAC Policies Active: 4 (Admin, Finance, Operations, Viewer)
├─ Last Policy Update: 3 days ago
├─ Policy Violations: 0 (this month)
├─ Audit Log Size: 450 MB (for 30 days)
│
SECURITY CERTIFICATES
├─ Node Identity Certificate: ✅ Valid until Oct 12, 2027 (1 year)
├─ TLS Certificate: ✅ Valid until Oct 12, 2027 (1 year)
├─ Ed25519 Signing Key: ✅ Last rotated 0 days ago (at launch)
├─ Next Rotation Due: 90 days (Jan 12, 2027)
│
AUDIT LOG INTEGRITY (BLAKE3 Chain)
├─ Total Log Entries: 45,230
├─ Chain Hash: 0x7f4c2a89... (current)
├─ Integrity Status: ✅ Valid
├─ Last Verification: 6 hours ago
│
KEY ROTATION SCHEDULE
├─ Ed25519 Key Rotation: Every 90 days (enforced)
├─ Next Rotation: Jan 12, 2027 (100 days away)
├─ Action Required: None (automatic)
└─ Audit Trail: [View rotation history]
```

**6B: Audit Log Query**
```
AUDIT LOG QUERY (Last 7 Days)

[Filter] [All events] [Date range] [Severity] [Actor]

Recent Events:
├─ Oct 4, 12:30 UTC - Policy updated: "ML team GPU quota"
│  Actor: admin@acme.corp | Severity: INFO | [View diff]
│
├─ Oct 3, 08:15 UTC - Node certificate rotated: node-41
│  Actor: system (auto-rotation) | Severity: INFO | [View]
│
├─ Oct 2, 16:45 UTC - API key created: "monitoring-agent-v2"
│  Actor: ops@acme.corp | Severity: INFO | [View]
│
├─ Oct 1, 09:00 UTC - New operator onboarded: backup admin
│  Actor: admin@acme.corp | Severity: INFO | [View]
│
└─ [Load more] [Export to CSV] [Subscribe to alerts]
```

---

### Tab 7: Settings (Admin Controls)

**Purpose:** Cluster configuration and admin controls  
**Refresh:** On-demand

**7A: Basic Settings**
```
CLUSTER SETTINGS

Operator Information
├─ Name: Acme Corp
├─ Email: operators@acme.corp
├─ Support Contact: support@acme.corp
├─ Time Zone: America/New_York
│
API Keys
├─ Monitoring Agent: monitoring-agent-v2 (active)
├─ Billing System: stripe-integration-v1 (active)
├─ [Generate New Key] [Revoke Key] [View Key History]
│
Notifications
├─ Critical Alerts: Slack + Email + SMS
├─ High Alerts: Slack + Email
├─ Medium Alerts: Email only
├─ Low Alerts: Dashboard only
│
Thresholds & Alarms
├─ CPU Warning Threshold: >85% on any node
├─ Memory Warning Threshold: >80% on any node
├─ Placement Latency Alert: >20ms p95 for 5min
├─ [Customize thresholds]
```

---

## Phase 7 Enhanced Analytics (Roadmap)

### Timeline: Q1 2027 (3 months after Phase 6 launch)

**Phase 7 Dashboard Enhancements:**

1. **Placement Failure Analysis** (Feb 2027)
   - Root cause diagnosis: Why did placement fail?
   - Topology visualization: See which nodes match, which don't
   - Recommendation engine: Suggest fixes (add nodes, adjust policies)

2. **Cross-Region Analytics** (Feb 2027)
   - Multi-region SLA dashboard (US-WEST, EU, ASIA)
   - Replication lag monitoring
   - Failover event timeline

3. **Advanced Cost Analytics** (Mar 2027)
   - Cost breakdown by workload type (GPU vs. CPU)
   - Cost per team or department
   - Tier upgrade ROI calculator (when should I move to TRUSTED?)

4. **Predictive Alerts** (Mar 2027)
   - ML-based anomaly detection
   - Forecast capacity exhaustion (predict 48h before full)
   - Estimated SLA miss warning (if current trends continue)

5. **Operator Benchmarking** (Mar 2027)
   - Compare your metrics to industry benchmarks
   - See how you rank: "Your uptime: 99.95% (top 5%)"
   - Peer recommendations: "Other operators in your tier achieve X"

---

## Design Specifications

### Technical Stack
- **Frontend:** React 18 + TypeScript
- **Data:** GraphQL API to metrics backend
- **Charting:** Apache ECharts (performance optimized)
- **Real-time updates:** WebSocket push (30s refresh for most metrics)
- **Caching:** Browser cache + CDN for dashboards

### Accessibility & Usability
- **Dark mode:** Toggle in settings (default light)
- **Responsive:** Mobile-friendly on tablet + mobile
- **Keyboard navigation:** Tab through all controls
- **Export:** All tables exportable to CSV/JSON
- **Print:** Printable SLA report (for compliance)

### Performance Targets
- **Page load:** <2 seconds
- **Chart render:** <500ms
- **Search/filter:** <100ms
- **Drill-down:** <500ms (click node → detail view)

---

**Operator Dashboard Design - Decentralized.Host**  
**Prepared:** October 4, 2026  
**Status:** Ready for Phase 6 launch (Oct 12)  
**Next Step:** Design mockups + frontend implementation (Oct 5-12)
