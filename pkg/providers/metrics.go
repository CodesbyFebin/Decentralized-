package providers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MetricsCollector collects and aggregates system metrics for monitoring.
type MetricsCollector struct {
	campaignMetrics  *CampaignMetrics
	systemMetrics    *SystemMetrics
	gateMetrics      *GateExecutionMetrics
	backupMetrics    *BackupMetrics
	migrationMetrics *MigrationMetrics
	collectorMutex   sync.RWMutex
	lastCollection   time.Time
	collectionErrors []string
}

// CampaignMetrics tracks campaign execution and qualification metrics.
type CampaignMetrics struct {
	TotalCampaigns      int64
	ActiveCampaigns     int64
	CompletedCampaigns  int64
	FailedCampaigns     int64
	PassedCampaigns     int64
	FailedQualification int64
	AverageDuration     float64 // seconds
	P50Duration         float64 // 50th percentile
	P95Duration         float64 // 95th percentile
	P99Duration         float64 // 99th percentile
	SuccessRate         float64 // percentage
	Throughput          float64 // campaigns per second
	LastUpdated         time.Time
}

// SystemMetrics tracks database and infrastructure metrics.
type SystemMetrics struct {
	DatabaseConnections     int64
	DatabaseQueryLatency    float64 // milliseconds average
	DatabaseQueryErrors     int64
	StorageUsage            int64 // bytes
	MemoryUsage             int64 // bytes
	CPUUsage                float64 // percentage
	CacheHitRate            float64 // percentage
	ConnectionPoolSize      int64
	ConnectionPoolUsage     int64
	ActiveTransactions      int64
	FailedTransactions      int64
	SchemaVersion           string
	LastUpdated             time.Time
}

// GateExecutionMetrics tracks qualification gate execution metrics for monitoring.
type GateExecutionMetrics struct {
	TotalGates          int64
	ExecutedGates       int64
	PassedGates         int64
	FailedGates         int64
	SkippedGates        int64
	AverageDuration     float64 // milliseconds
	P95Duration         float64
	P99Duration         float64
	MostFailedGate      string
	MostFailedCount     int64
	GateCategoryStats   map[string]*GateCategoryMetric
	LastUpdated         time.Time
}

// GateCategoryMetric tracks metrics per gate category.
type GateCategoryMetric struct {
	Category    string
	Total       int64
	Passed      int64
	Failed      int64
	SuccessRate float64
}

// BackupMetrics tracks backup and replication metrics.
type BackupMetrics struct {
	TotalBackups        int64
	SuccessfulBackups   int64
	FailedBackups       int64
	TotalReplicas       int64
	HealthyReplicas     int64
	DegradedReplicas    int64
	OfflineReplicas     int64
	ReplicationHealth   float64 // percentage
	AverageBakupSize    int64   // bytes
	TotalBackupStorage  int64   // bytes
	LastBackupTime      time.Time
	BackupRetentionDays int
	RecoveryRPO         float64 // minutes
	RecoveryRTO         float64 // minutes
	LastUpdated         time.Time
}

// MigrationMetrics tracks database migration metrics.
type MigrationMetrics struct {
	TotalMigrations      int64
	SuccessfulMigrations int64
	FailedMigrations     int64
	RolledBackCount      int64
	AverageDuration      float64 // seconds
	CampaignsMigrated    int64
	MigrationThroughput  float64 // campaigns per second
	LastMigrationTime    time.Time
	LastMigrationStatus  string // "success", "failed", "rolled_back"
	DataIntegrityScore   float64 // percentage (0-100)
	HashMismatchCount    int64
	LastUpdated          time.Time
}

// MetricsSnapshot represents a point-in-time view of all metrics.
type MetricsSnapshot struct {
	Timestamp            time.Time
	CampaignMetrics      *CampaignMetrics
	SystemMetrics        *SystemMetrics
	GateMetrics          *GateExecutionMetrics
	BackupMetrics        *BackupMetrics
	MigrationMetrics     *MigrationMetrics
	CollectionDuration   time.Duration
	CollectionErrors     []string
}

// NewMetricsCollector creates a metrics collector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		campaignMetrics:  &CampaignMetrics{},
		systemMetrics:    &SystemMetrics{},
		gateMetrics:      &GateExecutionMetrics{GateCategoryStats: make(map[string]*GateCategoryMetric)},
		backupMetrics:    &BackupMetrics{},
		migrationMetrics: &MigrationMetrics{},
		collectionErrors: make([]string, 0),
	}
}

// CollectMetrics gathers all system metrics from runtime state.
func (mc *MetricsCollector) CollectMetrics(ctx context.Context) (*MetricsSnapshot, error) {
	startTime := time.Now()

	mc.collectorMutex.Lock()
	defer mc.collectorMutex.Unlock()

	snapshot := &MetricsSnapshot{
		Timestamp:           startTime,
		CampaignMetrics:     mc.campaignMetrics,
		SystemMetrics:       mc.systemMetrics,
		GateMetrics:         mc.gateMetrics,
		BackupMetrics:       mc.backupMetrics,
		MigrationMetrics:    mc.migrationMetrics,
		CollectionErrors:    mc.collectionErrors,
	}

	snapshot.CollectionDuration = time.Since(startTime)
	mc.lastCollection = startTime

	return snapshot, nil
}

// UpdateCampaignMetrics updates campaign-related metrics.
func (mc *MetricsCollector) UpdateCampaignMetrics(
	total, active, completed, failed, passed int64,
	avgDuration, p50, p95, p99 float64,
	throughput float64) {

	mc.collectorMutex.Lock()
	defer mc.collectorMutex.Unlock()

	mc.campaignMetrics.TotalCampaigns = total
	mc.campaignMetrics.ActiveCampaigns = active
	mc.campaignMetrics.CompletedCampaigns = completed
	mc.campaignMetrics.FailedCampaigns = failed
	mc.campaignMetrics.PassedCampaigns = passed
	mc.campaignMetrics.FailedQualification = failed
	mc.campaignMetrics.AverageDuration = avgDuration
	mc.campaignMetrics.P50Duration = p50
	mc.campaignMetrics.P95Duration = p95
	mc.campaignMetrics.P99Duration = p99

	if total > 0 {
		mc.campaignMetrics.SuccessRate = (float64(passed) / float64(total)) * 100
	}
	mc.campaignMetrics.Throughput = throughput
	mc.campaignMetrics.LastUpdated = time.Now()
}

// UpdateSystemMetrics updates database and system metrics.
func (mc *MetricsCollector) UpdateSystemMetrics(
	dbConnections, dbQueryErrors, storageUsage, memoryUsage int64,
	dbQueryLatency, cpuUsage, cacheHitRate float64,
	poolSize, poolUsage, activeTransactions, failedTransactions int64,
	schemaVersion string) {

	mc.collectorMutex.Lock()
	defer mc.collectorMutex.Unlock()

	mc.systemMetrics.DatabaseConnections = dbConnections
	mc.systemMetrics.DatabaseQueryLatency = dbQueryLatency
	mc.systemMetrics.DatabaseQueryErrors = dbQueryErrors
	mc.systemMetrics.StorageUsage = storageUsage
	mc.systemMetrics.MemoryUsage = memoryUsage
	mc.systemMetrics.CPUUsage = cpuUsage
	mc.systemMetrics.CacheHitRate = cacheHitRate
	mc.systemMetrics.ConnectionPoolSize = poolSize
	mc.systemMetrics.ConnectionPoolUsage = poolUsage
	mc.systemMetrics.ActiveTransactions = activeTransactions
	mc.systemMetrics.FailedTransactions = failedTransactions
	mc.systemMetrics.SchemaVersion = schemaVersion
	mc.systemMetrics.LastUpdated = time.Now()
}

// UpdateGateMetrics updates qualification gate execution metrics.
func (mc *MetricsCollector) UpdateGateMetrics(
	total, executed, passed, failed, skipped int64,
	avgDuration, p95, p99 float64,
	mostFailedGate string, mostFailedCount int64) {

	mc.collectorMutex.Lock()
	defer mc.collectorMutex.Unlock()

	mc.gateMetrics.TotalGates = total
	mc.gateMetrics.ExecutedGates = executed
	mc.gateMetrics.PassedGates = passed
	mc.gateMetrics.FailedGates = failed
	mc.gateMetrics.SkippedGates = skipped
	mc.gateMetrics.AverageDuration = avgDuration
	mc.gateMetrics.P95Duration = p95
	mc.gateMetrics.P99Duration = p99
	mc.gateMetrics.MostFailedGate = mostFailedGate
	mc.gateMetrics.MostFailedCount = mostFailedCount
	mc.gateMetrics.LastUpdated = time.Now()
}

// UpdateGateCategoryMetric updates metrics for a specific gate category.
func (mc *MetricsCollector) UpdateGateCategoryMetric(category string, total, passed, failed int64) {
	mc.collectorMutex.Lock()
	defer mc.collectorMutex.Unlock()

	metric := &GateCategoryMetric{
		Category: category,
		Total:    total,
		Passed:   passed,
		Failed:   failed,
	}
	if total > 0 {
		metric.SuccessRate = (float64(passed) / float64(total)) * 100
	}
	mc.gateMetrics.GateCategoryStats[category] = metric
}

// UpdateBackupMetrics updates backup and replication metrics.
func (mc *MetricsCollector) UpdateBackupMetrics(
	totalBackups, successfulBackups, failedBackups,
	totalReplicas, healthyReplicas, degradedReplicas, offlineReplicas int64,
	avgBackupSize, totalStorage int64,
	retentionDays int,
	rpo, rto float64,
	lastBackup time.Time) {

	mc.collectorMutex.Lock()
	defer mc.collectorMutex.Unlock()

	mc.backupMetrics.TotalBackups = totalBackups
	mc.backupMetrics.SuccessfulBackups = successfulBackups
	mc.backupMetrics.FailedBackups = failedBackups
	mc.backupMetrics.TotalReplicas = totalReplicas
	mc.backupMetrics.HealthyReplicas = healthyReplicas
	mc.backupMetrics.DegradedReplicas = degradedReplicas
	mc.backupMetrics.OfflineReplicas = offlineReplicas

	if totalReplicas > 0 {
		mc.backupMetrics.ReplicationHealth = (float64(healthyReplicas) / float64(totalReplicas)) * 100
	}
	mc.backupMetrics.AverageBakupSize = avgBackupSize
	mc.backupMetrics.TotalBackupStorage = totalStorage
	mc.backupMetrics.BackupRetentionDays = retentionDays
	mc.backupMetrics.RecoveryRPO = rpo
	mc.backupMetrics.RecoveryRTO = rto
	mc.backupMetrics.LastBackupTime = lastBackup
	mc.backupMetrics.LastUpdated = time.Now()
}

// UpdateMigrationMetrics updates database migration metrics.
func (mc *MetricsCollector) UpdateMigrationMetrics(
	totalMigrations, successfulMigrations, failedMigrations, rolledBack int64,
	avgDuration float64,
	campaignsMigrated int64,
	migrationThroughput float64,
	integrityScore float64,
	hashMismatchCount int64,
	lastMigrationTime time.Time,
	lastStatus string) {

	mc.collectorMutex.Lock()
	defer mc.collectorMutex.Unlock()

	mc.migrationMetrics.TotalMigrations = totalMigrations
	mc.migrationMetrics.SuccessfulMigrations = successfulMigrations
	mc.migrationMetrics.FailedMigrations = failedMigrations
	mc.migrationMetrics.RolledBackCount = rolledBack
	mc.migrationMetrics.AverageDuration = avgDuration
	mc.migrationMetrics.CampaignsMigrated = campaignsMigrated
	mc.migrationMetrics.MigrationThroughput = migrationThroughput
	mc.migrationMetrics.DataIntegrityScore = integrityScore
	mc.migrationMetrics.HashMismatchCount = hashMismatchCount
	mc.migrationMetrics.LastMigrationTime = lastMigrationTime
	mc.migrationMetrics.LastMigrationStatus = lastStatus
	mc.migrationMetrics.LastUpdated = time.Now()
}

// RecordError records a collection error.
func (mc *MetricsCollector) RecordError(errMsg string) {
	mc.collectorMutex.Lock()
	defer mc.collectorMutex.Unlock()
	mc.collectionErrors = append(mc.collectionErrors, fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), errMsg))
}

// GetLastCollection returns the last collection timestamp.
func (mc *MetricsCollector) GetLastCollection() time.Time {
	mc.collectorMutex.RLock()
	defer mc.collectorMutex.RUnlock()
	return mc.lastCollection
}

// ExportPrometheus exports metrics in Prometheus text format.
func (snapshot *MetricsSnapshot) ExportPrometheus() string {
	output := fmt.Sprintf("# HELP qualification_campaigns_total Total campaigns in system\n")
	output += fmt.Sprintf("# TYPE qualification_campaigns_total gauge\n")
	output += fmt.Sprintf("qualification_campaigns_total %d\n\n", snapshot.CampaignMetrics.TotalCampaigns)

	output += fmt.Sprintf("# HELP qualification_campaigns_active Active campaigns\n")
	output += fmt.Sprintf("# TYPE qualification_campaigns_active gauge\n")
	output += fmt.Sprintf("qualification_campaigns_active %d\n\n", snapshot.CampaignMetrics.ActiveCampaigns)

	output += fmt.Sprintf("# HELP qualification_campaigns_completed Completed campaigns\n")
	output += fmt.Sprintf("# TYPE qualification_campaigns_completed gauge\n")
	output += fmt.Sprintf("qualification_campaigns_completed %d\n\n", snapshot.CampaignMetrics.CompletedCampaigns)

	output += fmt.Sprintf("# HELP qualification_campaigns_passed Passed campaigns\n")
	output += fmt.Sprintf("# TYPE qualification_campaigns_passed gauge\n")
	output += fmt.Sprintf("qualification_campaigns_passed %d\n\n", snapshot.CampaignMetrics.PassedCampaigns)

	output += fmt.Sprintf("# HELP qualification_campaigns_failed Failed campaigns\n")
	output += fmt.Sprintf("# TYPE qualification_campaigns_failed gauge\n")
	output += fmt.Sprintf("qualification_campaigns_failed %d\n\n", snapshot.CampaignMetrics.FailedCampaigns)

	output += fmt.Sprintf("# HELP qualification_success_rate Success rate percentage\n")
	output += fmt.Sprintf("# TYPE qualification_success_rate gauge\n")
	output += fmt.Sprintf("qualification_success_rate %.2f\n\n", snapshot.CampaignMetrics.SuccessRate)

	output += fmt.Sprintf("# HELP qualification_campaign_duration_seconds Campaign duration quantiles\n")
	output += fmt.Sprintf("# TYPE qualification_campaign_duration_seconds gauge\n")
	output += fmt.Sprintf("qualification_campaign_duration_seconds{quantile=\"p50\"} %.2f\n", snapshot.CampaignMetrics.P50Duration)
	output += fmt.Sprintf("qualification_campaign_duration_seconds{quantile=\"p95\"} %.2f\n", snapshot.CampaignMetrics.P95Duration)
	output += fmt.Sprintf("qualification_campaign_duration_seconds{quantile=\"p99\"} %.2f\n\n", snapshot.CampaignMetrics.P99Duration)

	output += fmt.Sprintf("# HELP qualification_campaign_throughput Campaigns per second\n")
	output += fmt.Sprintf("# TYPE qualification_campaign_throughput gauge\n")
	output += fmt.Sprintf("qualification_campaign_throughput %.4f\n\n", snapshot.CampaignMetrics.Throughput)

	output += fmt.Sprintf("# HELP qualification_gates_total Total gates\n")
	output += fmt.Sprintf("# TYPE qualification_gates_total gauge\n")
	output += fmt.Sprintf("qualification_gates_total %d\n\n", snapshot.GateMetrics.TotalGates)

	output += fmt.Sprintf("# HELP qualification_gates_executed Executed gates\n")
	output += fmt.Sprintf("# TYPE qualification_gates_executed gauge\n")
	output += fmt.Sprintf("qualification_gates_executed %d\n\n", snapshot.GateMetrics.ExecutedGates)

	output += fmt.Sprintf("# HELP qualification_gates_passed Passed gates\n")
	output += fmt.Sprintf("# TYPE qualification_gates_passed gauge\n")
	output += fmt.Sprintf("qualification_gates_passed %d\n\n", snapshot.GateMetrics.PassedGates)

	output += fmt.Sprintf("# HELP qualification_gates_failed Failed gates\n")
	output += fmt.Sprintf("# TYPE qualification_gates_failed gauge\n")
	output += fmt.Sprintf("qualification_gates_failed %d\n\n", snapshot.GateMetrics.FailedGates)

	output += fmt.Sprintf("# HELP qualification_database_connections Active database connections\n")
	output += fmt.Sprintf("# TYPE qualification_database_connections gauge\n")
	output += fmt.Sprintf("qualification_database_connections %d\n\n", snapshot.SystemMetrics.DatabaseConnections)

	output += fmt.Sprintf("# HELP qualification_database_query_latency_ms Query latency milliseconds\n")
	output += fmt.Sprintf("# TYPE qualification_database_query_latency_ms gauge\n")
	output += fmt.Sprintf("qualification_database_query_latency_ms %.2f\n\n", snapshot.SystemMetrics.DatabaseQueryLatency)

	output += fmt.Sprintf("# HELP qualification_database_query_errors Query errors\n")
	output += fmt.Sprintf("# TYPE qualification_database_query_errors gauge\n")
	output += fmt.Sprintf("qualification_database_query_errors %d\n\n", snapshot.SystemMetrics.DatabaseQueryErrors)

	output += fmt.Sprintf("# HELP qualification_backup_health Backup health percentage\n")
	output += fmt.Sprintf("# TYPE qualification_backup_health gauge\n")
	output += fmt.Sprintf("qualification_backup_health %.2f\n\n", snapshot.BackupMetrics.ReplicationHealth)

	output += fmt.Sprintf("# HELP qualification_backup_replicas_total Total replicas\n")
	output += fmt.Sprintf("# TYPE qualification_backup_replicas_total gauge\n")
	output += fmt.Sprintf("qualification_backup_replicas_total %d\n\n", snapshot.BackupMetrics.TotalReplicas)

	output += fmt.Sprintf("# HELP qualification_backup_replicas_healthy Healthy replicas\n")
	output += fmt.Sprintf("# TYPE qualification_backup_replicas_healthy gauge\n")
	output += fmt.Sprintf("qualification_backup_replicas_healthy %d\n\n", snapshot.BackupMetrics.HealthyReplicas)

	output += fmt.Sprintf("# HELP qualification_backup_replicas_degraded Degraded replicas\n")
	output += fmt.Sprintf("# TYPE qualification_backup_replicas_degraded gauge\n")
	output += fmt.Sprintf("qualification_backup_replicas_degraded %d\n\n", snapshot.BackupMetrics.DegradedReplicas)

	output += fmt.Sprintf("# HELP qualification_migration_integrity_score Data integrity score\n")
	output += fmt.Sprintf("# TYPE qualification_migration_integrity_score gauge\n")
	output += fmt.Sprintf("qualification_migration_integrity_score %.2f\n\n", snapshot.MigrationMetrics.DataIntegrityScore)

	output += fmt.Sprintf("# HELP qualification_migration_throughput Migrations per second\n")
	output += fmt.Sprintf("# TYPE qualification_migration_throughput gauge\n")
	output += fmt.Sprintf("qualification_migration_throughput %.4f\n\n", snapshot.MigrationMetrics.MigrationThroughput)

	return output
}

// AlertRule represents a Prometheus alerting rule.
type AlertRule struct {
	Name        string
	Description string
	Expr        string
	For         string
	Severity    string // "critical", "warning", "info"
	Annotations map[string]string
}

// GetAlertRules returns standard alerting rules for the qualification system.
func GetAlertRules() []AlertRule {
	return []AlertRule{
		{
			Name:        "QualificationSuccessRateLow",
			Description: "Qualification success rate below 80%",
			Expr:        "qualification_success_rate < 80",
			For:         "5m",
			Severity:    "warning",
			Annotations: map[string]string{
				"summary":     "Low qualification success rate",
				"description": "Success rate {{ $value }}% is below threshold",
			},
		},
		{
			Name:        "CampaignExecutionSlow",
			Description: "Campaign execution P95 latency above 30 seconds",
			Expr:        "qualification_campaign_duration_seconds{quantile=\"p95\"} > 30",
			For:         "10m",
			Severity:    "warning",
			Annotations: map[string]string{
				"summary":     "Slow campaign execution",
				"description": "P95 latency {{ $value }}s exceeds threshold",
			},
		},
		{
			Name:        "GateCategoryHighFailure",
			Description: "Gate category with >50% failure rate",
			Expr:        "qualification_gate_failure_rate > 50",
			For:         "5m",
			Severity:    "warning",
			Annotations: map[string]string{
				"summary":     "High gate failure rate",
				"description": "Gate failure rate {{ $value }}% is high",
			},
		},
		{
			Name:        "DatabaseConnectionPoolExhausted",
			Description: "Database connection pool usage above 90%",
			Expr:        "(qualification_database_connections / qualification_database_pool_size) > 0.9",
			For:         "2m",
			Severity:    "critical",
			Annotations: map[string]string{
				"summary":     "Database connection pool critical",
				"description": "Connection pool usage {{ $value }}%",
			},
		},
		{
			Name:        "DatabaseQueryLatencyHigh",
			Description: "Database query latency above 100ms",
			Expr:        "qualification_database_query_latency_ms > 100",
			For:         "5m",
			Severity:    "warning",
			Annotations: map[string]string{
				"summary":     "High database query latency",
				"description": "Query latency {{ $value }}ms exceeds threshold",
			},
		},
		{
			Name:        "DatabaseQueryErrors",
			Description: "Database query errors detected",
			Expr:        "increase(qualification_database_query_errors[5m]) > 10",
			For:         "2m",
			Severity:    "critical",
			Annotations: map[string]string{
				"summary":     "Database query errors",
				"description": "{{ $value }} query errors in 5 minutes",
			},
		},
		{
			Name:        "BackupHealthDegraded",
			Description: "Backup replication health below 80%",
			Expr:        "qualification_backup_health < 80",
			For:         "5m",
			Severity:    "critical",
			Annotations: map[string]string{
				"summary":     "Backup health degraded",
				"description": "Replication health {{ $value }}%",
			},
		},
		{
			Name:        "ReplicaOffline",
			Description: "More than 1 replica offline",
			Expr:        "qualification_backup_replicas_offline > 1",
			For:         "2m",
			Severity:    "critical",
			Annotations: map[string]string{
				"summary":     "Multiple replicas offline",
				"description": "{{ $value }} replicas offline",
			},
		},
		{
			Name:        "MigrationFailed",
			Description: "Database migration failed",
			Expr:        "increase(qualification_migration_failed[1h]) > 0",
			For:         "1m",
			Severity:    "critical",
			Annotations: map[string]string{
				"summary":     "Database migration failed",
				"description": "Migration operation failed",
			},
		},
		{
			Name:        "DataIntegrityIssues",
			Description: "Data integrity score below 95%",
			Expr:        "qualification_migration_integrity_score < 95",
			For:         "5m",
			Severity:    "critical",
			Annotations: map[string]string{
				"summary":     "Data integrity issues detected",
				"description": "Integrity score {{ $value }}%",
			},
		},
		{
			Name:        "StorageCapacityWarning",
			Description: "Storage usage above 80%",
			Expr:        "(qualification_storage_used / qualification_storage_total) > 0.8",
			For:         "10m",
			Severity:    "warning",
			Annotations: map[string]string{
				"summary":     "Storage capacity warning",
				"description": "Storage usage {{ $value }}%",
			},
		},
		{
			Name:        "CacheMissRateHigh",
			Description: "Cache miss rate above 20%",
			Expr:        "(1 - qualification_cache_hit_rate) > 0.2",
			For:         "5m",
			Severity:    "info",
			Annotations: map[string]string{
				"summary":     "High cache miss rate",
				"description": "Cache miss rate {{ $value }}%",
			},
		},
	}
}

// GrafanaDashboard represents a Grafana dashboard configuration.
type GrafanaDashboard struct {
	Title       string
	Description string
	Panels      []*GrafanaPanel
	Refresh     string
	TimeRange   string
}

// GrafanaPanel represents a panel in a Grafana dashboard.
type GrafanaPanel struct {
	Title       string
	Description string
	Type        string // "graph", "stat", "gauge", "table", "heatmap"
	Datasource  string
	Targets     []map[string]string
	Unit        string
	Thresholds  []float64
}

// GetGrafanaDashboard returns the main qualification dashboard configuration.
func GetGrafanaDashboard() *GrafanaDashboard {
	return &GrafanaDashboard{
		Title:       "Qualification System Metrics",
		Description: "Real-time monitoring for the qualification system",
		Refresh:     "30s",
		TimeRange:   "now-1h",
		Panels: []*GrafanaPanel{
			{
				Title:       "Campaigns - Success Rate",
				Description: "Overall qualification success rate",
				Type:        "gauge",
				Datasource:  "Prometheus",
				Unit:        "percent",
				Targets: []map[string]string{
					{"expr": "qualification_success_rate"},
				},
				Thresholds: []float64{50, 80},
			},
			{
				Title:       "Campaigns - Total Count",
				Description: "Total campaigns in system",
				Type:        "stat",
				Datasource:  "Prometheus",
				Unit:        "short",
				Targets: []map[string]string{
					{"expr": "qualification_campaigns_total"},
				},
			},
			{
				Title:       "Campaigns - Active",
				Description: "Currently active campaigns",
				Type:        "stat",
				Datasource:  "Prometheus",
				Unit:        "short",
				Targets: []map[string]string{
					{"expr": "qualification_campaigns_active"},
				},
			},
			{
				Title:       "Campaign Duration Quantiles",
				Description: "P50, P95, P99 campaign execution time",
				Type:        "graph",
				Datasource:  "Prometheus",
				Unit:        "s",
				Targets: []map[string]string{
					{"expr": "qualification_campaign_duration_seconds"},
				},
			},
			{
				Title:       "Campaign Throughput",
				Description: "Campaigns executed per second",
				Type:        "graph",
				Datasource:  "Prometheus",
				Unit:        "ops",
				Targets: []map[string]string{
					{"expr": "qualification_campaign_throughput"},
				},
			},
			{
				Title:       "Gates - Success by Category",
				Description: "Gate execution success rate by category",
				Type:        "graph",
				Datasource:  "Prometheus",
				Unit:        "percent",
				Targets: []map[string]string{
					{"expr": "qualification_gate_success_rate"},
				},
			},
			{
				Title:       "Database - Query Latency",
				Description: "Database query latency in milliseconds",
				Type:        "graph",
				Datasource:  "Prometheus",
				Unit:        "ms",
				Targets: []map[string]string{
					{"expr": "qualification_database_query_latency_ms"},
				},
				Thresholds: []float64{50, 100},
			},
			{
				Title:       "Database - Connection Pool",
				Description: "Database connection pool usage",
				Type:        "gauge",
				Datasource:  "Prometheus",
				Unit:        "percent",
				Targets: []map[string]string{
					{"expr": "(qualification_database_connections / qualification_database_pool_size) * 100"},
				},
				Thresholds: []float64{70, 90},
			},
			{
				Title:       "Database - Query Errors",
				Description: "Database query errors over time",
				Type:        "graph",
				Datasource:  "Prometheus",
				Unit:        "short",
				Targets: []map[string]string{
					{"expr": "rate(qualification_database_query_errors[5m])"},
				},
			},
			{
				Title:       "Backup - Replication Health",
				Description: "Percentage of healthy replicas",
				Type:        "gauge",
				Datasource:  "Prometheus",
				Unit:        "percent",
				Targets: []map[string]string{
					{"expr": "qualification_backup_health"},
				},
				Thresholds: []float64{60, 80},
			},
			{
				Title:       "Backup - Replica Status",
				Description: "Health status of backup replicas",
				Type:        "graph",
				Datasource:  "Prometheus",
				Unit:        "short",
				Targets: []map[string]string{
					{"expr": "qualification_backup_replicas_healthy"},
					{"expr": "qualification_backup_replicas_degraded"},
					{"expr": "qualification_backup_replicas_offline"},
				},
			},
			{
				Title:       "Migration - Data Integrity",
				Description: "Data integrity score after migration",
				Type:        "gauge",
				Datasource:  "Prometheus",
				Unit:        "percent",
				Targets: []map[string]string{
					{"expr": "qualification_migration_integrity_score"},
				},
				Thresholds: []float64{90, 95},
			},
			{
				Title:       "Migration - Throughput",
				Description: "Migrations processed per second",
				Type:        "graph",
				Datasource:  "Prometheus",
				Unit:        "ops",
				Targets: []map[string]string{
					{"expr": "qualification_migration_throughput"},
				},
			},
			{
				Title:       "System - Storage Usage",
				Description: "Overall system storage utilization",
				Type:        "gauge",
				Datasource:  "Prometheus",
				Unit:        "percent",
				Targets: []map[string]string{
					{"expr": "(qualification_storage_used / qualification_storage_total) * 100"},
				},
				Thresholds: []float64{70, 80},
			},
			{
				Title:       "System - Memory Usage",
				Description: "System memory utilization",
				Type:        "gauge",
				Datasource:  "Prometheus",
				Unit:        "percent",
				Targets: []map[string]string{
					{"expr": "qualification_memory_usage_percent"},
				},
				Thresholds: []float64{70, 85},
			},
			{
				Title:       "System - Cache Hit Rate",
				Description: "Cache hit rate percentage",
				Type:        "gauge",
				Datasource:  "Prometheus",
				Unit:        "percent",
				Targets: []map[string]string{
					{"expr": "qualification_cache_hit_rate * 100"},
				},
				Thresholds: []float64{60, 80},
			},
		},
	}
}
