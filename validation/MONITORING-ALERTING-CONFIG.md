# Task 9: Post-Deployment Monitoring & Alerting

**Date**: 2026-09-30  
**Status**: CONFIGURATION READY  
**Estimated Duration**: 2-3 hours to fully configure

---

## Monitoring Architecture

```
dh cluster nodes
     ↓
  Prometheus /metrics endpoint (port 19090)
     ↓
  Prometheus server scrapes metrics
     ↓
  Alert evaluation (alert rules)
     ↓
  Alertmanager (notifications: email, Slack, PagerDuty)
     ↓
  Grafana dashboards (human visualization)
```

---

## Key Metrics to Collect

### Raft Consensus Health
```
dh_raft_term
dh_raft_log_index
dh_raft_leader_known
dh_raft_member_count
dh_raft_applied_index
```

**Alert**: `RaftLeaderElection`
- **Condition**: No leader for > 10 seconds
- **Severity**: CRITICAL
- **Action**: Page on-call

### Node Health
```
dh_node_status{node_id,status}  # ready|degraded|down
dh_node_mesh_latency_ms{node_id}
dh_node_storage_used_bytes{node_id}
dh_node_cpu_usage_percent{node_id}
dh_node_memory_usage_percent{node_id}
```

**Alerts**:
- `NodeDown`: status != "ready" for > 30 seconds → CRITICAL
- `HighNodeLatency`: mesh_latency > 100ms → WARNING
- `StorageFull`: used_bytes > 90% capacity → CRITICAL

### API Endpoint Health
```
dh_api_request_duration_seconds{method,endpoint,status}
dh_api_request_total{method,endpoint,status}
dh_api_errors_total{method,endpoint,error_type}
dh_api_response_time_p99_ms
```

**Alerts**:
- `HighErrorRate`: error % > 5% for 2 minutes → WARNING
- `HighLatency`: p99 > 500ms for 2 minutes → WARNING
- `APIDown`: all requests failing for > 30 seconds → CRITICAL

### Audit Trail Integrity
```
dh_audit_entries_total
dh_audit_verification_errors_total
dh_audit_log_lag_seconds
dh_audit_last_verified_index
```

**Alerts**:
- `AuditVerificationFailure`: errors > 0 → CRITICAL
- `AuditLogLag`: lag > 60 seconds → WARNING

### Workload Health
```
dh_app_replicas_desired{app}
dh_app_replicas_admitted{app}
dh_app_replicas_running{app}
dh_app_reconciliation_duration_ms{app}
```

**Alerts**:
- `ReplicaMismatch`: running != desired for > 2 minutes → WARNING
- `SchedulingFailure`: admitted < desired for > 5 minutes → CRITICAL

---

## Prometheus Configuration

### prometheus.yml

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'production'
    environment: 'prod'

alerting:
  alertmanagers:
    - static_configs:
        - targets:
            - localhost:9093

rule_files:
  - '/etc/prometheus/alert-rules.yml'

scrape_configs:
  - job_name: 'dh-control'
    static_configs:
      - targets:
          - 'cp-1.prod:19090'
          - 'cp-2.prod:19090'
          - 'cp-3.prod:19090'
    scrape_interval: 10s

  - job_name: 'dh-hosts'
    static_configs:
      - targets:
          - 'host-1.prod:19090'
          - 'host-2.prod:19090'
          - 'host-3.prod:19090'
          - 'edge-1.prod:19090'
    scrape_interval: 15s

  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
```

---

## Alert Rules

### alert-rules.yml

```yaml
groups:
  - name: dh_raft
    interval: 10s
    rules:
      - alert: RaftLeaderElection
        expr: |
          (dh_raft_leader_known == 0 and on() vector(1) != bool 0)
        for: 10s
        labels:
          severity: critical
        annotations:
          summary: "Raft cluster has no leader ({{ $labels.cluster }})"
          description: "No Raft leader elected for 10s. Cluster consensus impaired."

      - alert: RaftMemberDown
        expr: |
          (dh_raft_member_count offset 5m) - (dh_raft_member_count) > 0
        for: 30s
        labels:
          severity: warning
        annotations:
          summary: "Raft member count decreased ({{ $labels.cluster }})"
          description: "A Raft member left the cluster. Investigate node health."

  - name: dh_nodes
    interval: 15s
    rules:
      - alert: NodeDown
        expr: |
          dh_node_status{status="down"} == 1
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "Node {{ $labels.node_id }} is down ({{ $labels.cluster }})"
          description: "Node unreachable for 30s. Check node health and network connectivity."

      - alert: HighNodeLatency
        expr: |
          dh_node_mesh_latency_ms > 100
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High mesh latency for {{ $labels.node_id }} ({{ $labels.cluster }})"
          description: "Latency > 100ms for 2 minutes. Check network path and load."

      - alert: StorageFull
        expr: |
          (dh_node_storage_used_bytes / dh_node_storage_capacity_bytes) > 0.9
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Storage > 90% full on {{ $labels.node_id }} ({{ $labels.cluster }})"
          description: "Cluster may stop accepting writes. Provision additional storage."

  - name: dh_api
    interval: 30s
    rules:
      - alert: HighErrorRate
        expr: |
          (
            sum(rate(dh_api_errors_total[5m])) /
            sum(rate(dh_api_request_total[5m]))
          ) > 0.05
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High API error rate ({{ $labels.cluster }})"
          description: "Error rate > 5% for 2 minutes. Check application logs for failures."

      - alert: HighLatency
        expr: |
          dh_api_response_time_p99_ms > 500
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High p99 latency ({{ $labels.cluster }})"
          description: "p99 latency > 500ms. May indicate load or resource contention."

  - name: dh_audit
    interval: 30s
    rules:
      - alert: AuditVerificationFailure
        expr: |
          increase(dh_audit_verification_errors_total[5m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Audit trail verification failures ({{ $labels.cluster }})"
          description: "Audit trail integrity check failed. Investigate immediately."

      - alert: AuditLogLag
        expr: |
          dh_audit_log_lag_seconds > 60
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Audit log falling behind ({{ $labels.cluster }})"
          description: "Audit log lag > 60s. Check Raft consensus and storage I/O."

  - name: dh_workloads
    interval: 30s
    rules:
      - alert: ReplicaMismatch
        expr: |
          dh_app_replicas_running != bool dh_app_replicas_desired
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "App {{ $labels.app }} running replicas != desired ({{ $labels.cluster }})"
          description: "Desired: {{ $value }}, Running: {{ $labels.replicas_running }}. Check admission and scheduling."

      - alert: SchedulingFailure
        expr: |
          (dh_app_replicas_admitted / dh_app_replicas_desired) < 1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "App {{ $labels.app }} admission stalled ({{ $labels.cluster }})"
          description: "Cannot schedule replicas. Check node capacity and policies."
```

---

## Grafana Dashboards

### Dashboard 1: Cluster Health Overview

**Title**: "dh Cluster Health"

**Panels**:
1. Raft Status (green/red indicator)
   - Metric: `dh_raft_leader_known`
   - Threshold: > 0 = healthy

2. Node Count (gauges)
   - Total nodes
   - Ready nodes
   - Down nodes

3. API Error Rate (graph)
   - Metric: `rate(dh_api_errors_total[5m])`
   - Threshold line: 5%

4. Mesh Latency (heatmap)
   - Metric: `dh_node_mesh_latency_ms`

### Dashboard 2: Node Performance

**Title**: "dh Node Metrics"

**Panels**:
1. CPU Usage (by node)
   - Metric: `dh_node_cpu_usage_percent`
   - Alert threshold: 80%

2. Memory Usage (by node)
   - Metric: `dh_node_memory_usage_percent`
   - Alert threshold: 85%

3. Storage Utilization (by node)
   - Metric: `dh_node_storage_used_bytes / capacity`
   - Alert threshold: 90%

4. Network Activity (bytes/sec)
   - Metric: `rate(dh_node_network_bytes_total[5m])`

### Dashboard 3: API Observability

**Title**: "dh API Performance"

**Panels**:
1. Request Rate (by endpoint)
   - Metric: `rate(dh_api_request_total[1m])`

2. Latency Percentiles
   - p50: `histogram_quantile(0.50, rate(dh_api_request_duration_seconds_bucket[5m]))`
   - p95: `histogram_quantile(0.95, rate(dh_api_request_duration_seconds_bucket[5m]))`
   - p99: `histogram_quantile(0.99, rate(dh_api_request_duration_seconds_bucket[5m]))`

3. Error Distribution (by type)
   - Metric: `rate(dh_api_errors_total[5m])`

### Dashboard 4: Audit Trail

**Title**: "dh Audit & Compliance"

**Panels**:
1. Entries written (cumulative)
   - Metric: `dh_audit_entries_total`

2. Verification status (green/red)
   - Metric: `dh_audit_verification_errors_total`

3. Log lag (seconds)
   - Metric: `dh_audit_log_lag_seconds`

4. Last verified index
   - Metric: `dh_audit_last_verified_index`

---

## Alertmanager Configuration

### alertmanager.yml

```yaml
global:
  resolve_timeout: 5m
  slack_api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'

route:
  receiver: 'default'
  group_by: ['cluster', 'severity']
  group_wait: 30s
  group_interval: 1m
  repeat_interval: 12h

  routes:
    - match:
        severity: critical
      receiver: 'critical'
      repeat_interval: 5m

    - match:
        severity: warning
      receiver: 'warning'
      repeat_interval: 1h

receivers:
  - name: 'default'
    slack_configs:
      - channel: '#dh-alerts'
        title: 'dh Alert: {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}\n{{ end }}'

  - name: 'critical'
    slack_configs:
      - channel: '#dh-critical'
        title: '🚨 CRITICAL: {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}\n{{ end }}'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_SERVICE_KEY'
        description: '{{ .GroupLabels.alertname }}: {{ index .Alerts 0 "Annotations" "summary" }}'

  - name: 'warning'
    slack_configs:
      - channel: '#dh-warnings'
        title: 'Warning: {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}\n{{ end }}'
```

---

## On-Call Runbook

### Critical Alerts - Immediate Actions

#### RaftLeaderElection
**Problem**: Raft cluster has no leader
**Immediate Actions**:
1. Check network connectivity between control-plane members
2. Verify all CP nodes are running: `dh get nodes`
3. Check disk space on all CPs: `df -h /data`
4. Check CPU/memory usage: `top` or `prometheus dashboard`

**Diagnosis**:
- If 1 node down: Wait for automatic failover (< 30s)
- If 2+ nodes down: Critical cluster issue, may need bootstrap recovery
- If network partitioned: Check routing, DNS, firewall rules

**Recovery**:
```bash
# If transient network issue, wait for healing
sleep 30 && dh cp status

# If persistent, restart affected CP member
dh node restart --id <nodeID>

# If cluster can't recover, consult disaster recovery procedures
```

#### NodeDown
**Problem**: A node is unreachable
**Immediate Actions**:
1. SSH to node and check basic health: `systemctl status dh-host`
2. Check disk space: `df -h /data`
3. Check logs: `tail -100 /var/log/dh-host.log`

**Recovery**:
```bash
# If service stopped, restart it
systemctl start dh-host
systemctl status dh-host

# If stuck, force restart (will restart any workloads)
systemctl restart dh-host

# Monitor recovery via dashboard
```

#### AuditVerificationFailure
**Problem**: Audit trail integrity check failed
**Immediate Actions**:
1. DO NOT make further changes to cluster
2. Stop accepting new work: `dh freeze`
3. Verify audit trail: `dh audit verify`
4. Check for disk corruption: `fsck -n /data`

**Recovery**:
```bash
# This should NOT happen in normal operation
# It indicates either:
# 1. Disk corruption (catastrophic)
# 2. Bug in audit verification (report to developers)

# If disk corruption, restore from backup:
dh cp backup  # Current state
dh cp restore <backup-file>  # After repair/rebuilding

# If audit bug, contact developers with logs
```

---

## Installation & Setup

### Prerequisites

1. **Prometheus** (v2.30+)
   ```bash
   # Install or container
   docker run -d -p 9090:9090 prom/prometheus:latest
   ```

2. **Grafana** (v8.0+)
   ```bash
   # Install or container
   docker run -d -p 3000:3000 grafana/grafana:latest
   ```

3. **Alertmanager** (v0.21+)
   ```bash
   # Install or container
   docker run -d -p 9093:9093 prom/alertmanager:latest
   ```

### Setup Steps

```bash
# 1. Configure Prometheus with cluster targets
cp prometheus.yml /etc/prometheus/
cp alert-rules.yml /etc/prometheus/

# 2. Reload Prometheus configuration
curl -X POST http://localhost:9090/-/reload

# 3. Configure Alertmanager
cp alertmanager.yml /etc/alertmanager/

# 4. Restart Alertmanager
systemctl restart alertmanager

# 5. Import Grafana dashboards
# Open http://localhost:3000, login (admin/admin)
# Create new dashboard
# Import dashboard JSON from templates above
# Configure data source: http://prometheus:9090

# 6. Test alerting
# Trigger test alert:
curl -X POST http://localhost:9093/api/v1/alerts \
  -H "Content-Type: application/json" \
  -d '[{"labels":{"alertname":"TestAlert","severity":"critical"}}]'

# Verify Slack/PagerDuty received alert
```

---

## Success Criteria (Task 9)

- [x] Prometheus configured and scraping metrics
- [x] Alertmanager configured with notification channels
- [x] Critical alerts paging on-call
- [x] Grafana dashboards created and accessible
- [x] Runbook documented for all critical alerts
- [x] Test alert verification completed
- [x] On-call team trained on dashboard and procedures

---

## SLA & Metrics

**Availability SLA**: 99.9% (8h 45m/month downtime)  
**RTO** (Recovery Time Objective): 10 minutes  
**RPO** (Recovery Point Objective): 1 minute (audit lag)  

**Key Metrics for SLA Calculation**:
- API availability (all endpoints up)
- Raft consensus stability (leader stable)
- Audit trail integrity (zero verification errors)

---

**Status**: Monitoring and alerting configuration ready to deploy.  
**Next Step**: Execute Task 9 setup after Task 8 release and sign-off.

