// Package backup implements backup configuration validation and failover testing
// for the Decentralized.Host operator qualification program.
//
// Operators must maintain minimum 2 failover nodes with geographic distribution,
// and achieve RTO <5 minutes and RPO <1 minute.
package backup

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	// MinimumFailoverNodes is the minimum number of failover nodes required
	MinimumFailoverNodes = 2

	// RTOTarget is the maximum Recovery Time Objective
	RTOTarget = 5 * time.Minute

	// RPOTarget is the maximum Recovery Point Objective
	RPOTarget = 1 * time.Minute

	// MinimumGeographicSeparation is the minimum distance in regions
	MinimumGeographicSeparation = 1 // Different regions
)

// BackupStatus represents the state of backup configuration
type BackupStatus string

const (
	CONFIGURED      BackupStatus = "CONFIGURED"
	NOT_CONFIGURED  BackupStatus = "NOT_CONFIGURED"
	DEGRADED        BackupStatus = "DEGRADED"
	UNTESTED        BackupStatus = "UNTESTED"
	TESTED_PASSING  BackupStatus = "TESTED_PASSING"
	TESTED_FAILING  BackupStatus = "TESTED_FAILING"
)

// FailoverNode represents a backup/failover node
type FailoverNode struct {
	NodeID              string    `json:"node_id"`
	Region              string    `json:"region"`
	IsPrimary           bool      `json:"is_primary"`
	SyncedAt            time.Time `json:"synced_at"`
	ReplicationLag      time.Duration `json:"replication_lag"` // RPO indicator
	LastHealthCheckAt   time.Time `json:"last_health_check_at"`
	IsHealthy           bool      `json:"is_healthy"`
	BackupSize          int64     `json:"backup_size"` // bytes
	LastFullBackupAt    time.Time `json:"last_full_backup_at"`
	LastIncrementalAt   time.Time `json:"last_incremental_at"`
}

// FailoverTest records a failover test execution
type FailoverTest struct {
	TestID              string        `json:"test_id"`
	StartTime           time.Time     `json:"start_time"`
	EndTime             time.Time     `json:"end_time"`
	Duration            time.Duration `json:"duration"`
	Success             bool          `json:"success"`
	RTO                 time.Duration `json:"rto"` // Actual recovery time
	RPO                 time.Duration `json:"rpo"` // Actual recovery point
	DataLoss            bool          `json:"data_loss"`
	FailedComponent     string        `json:"failed_component,omitempty"`
	ErrorMessage        string        `json:"error_message,omitempty"`
	Notes               string        `json:"notes,omitempty"`
}

// BackupConfiguration tracks backup and failover setup
type BackupConfiguration struct {
	OperatorID            string          `json:"operator_id"`
	Status                BackupStatus    `json:"status"`
	FailoverNodes         []FailoverNode  `json:"failover_nodes"`
	HealthyFailoverCount  int             `json:"healthy_failover_count"`
	RTOMinutes            int             `json:"rto_minutes"`
	RPOMinutes            int             `json:"rpo_minutes"`
	IsGeographicallyDiversified bool      `json:"is_geographically_diversified"`
	LastTestTime          *time.Time      `json:"last_test_time,omitempty"`
	LastTestResult        *FailoverTest   `json:"last_test_result,omitempty"`
	FailoverTestHistory   []FailoverTest  `json:"failover_test_history"`
	BackupSchedule        string          `json:"backup_schedule"` // e.g., "daily", "hourly"
	RetentionDays         int             `json:"retention_days"`
	EncryptionEnabled     bool            `json:"encryption_enabled"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

// BackupManager manages backup configurations
type BackupManager struct {
	mu      sync.RWMutex
	configs map[string]*BackupConfiguration
}

// NewBackupManager creates a new backup manager
func NewBackupManager() *BackupManager {
	return &BackupManager{
		configs: make(map[string]*BackupConfiguration),
	}
}

// CreateConfiguration initializes backup configuration for an operator
func (bm *BackupManager) CreateConfiguration(operatorID string, backupSchedule string, retentionDays int) (*BackupConfiguration, error) {
	if operatorID == "" {
		return nil, errors.New("operator ID cannot be empty")
	}

	if retentionDays < 7 {
		return nil, errors.New("retention period must be at least 7 days")
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	if _, exists := bm.configs[operatorID]; exists {
		return nil, fmt.Errorf("configuration already exists for operator %s", operatorID)
	}

	now := time.Now()
	config := &BackupConfiguration{
		OperatorID:          operatorID,
		Status:              NOT_CONFIGURED,
		FailoverNodes:       []FailoverNode{},
		RTOMinutes:          int(RTOTarget.Minutes()),
		RPOMinutes:          int(RPOTarget.Minutes()),
		BackupSchedule:      backupSchedule,
		RetentionDays:       retentionDays,
		EncryptionEnabled:   true,
		FailoverTestHistory: []FailoverTest{},
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	bm.configs[operatorID] = config
	return config, nil
}

// GetConfiguration retrieves backup configuration
func (bm *BackupManager) GetConfiguration(operatorID string) (*BackupConfiguration, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	config, exists := bm.configs[operatorID]
	if !exists {
		return nil, fmt.Errorf("no configuration found for operator %s", operatorID)
	}
	return config, nil
}

// AddFailoverNode adds a failover node to the configuration
func (bm *BackupManager) AddFailoverNode(operatorID string, nodeID string, region string, isPrimary bool) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	config, exists := bm.configs[operatorID]
	if !exists {
		return fmt.Errorf("no configuration found for operator %s", operatorID)
	}

	// Check for duplicate
	for _, node := range config.FailoverNodes {
		if node.NodeID == nodeID {
			return fmt.Errorf("node %s already in configuration", nodeID)
		}
	}

	now := time.Now()
	node := FailoverNode{
		NodeID:            nodeID,
		Region:            region,
		IsPrimary:         isPrimary,
		SyncedAt:          now,
		LastHealthCheckAt: now,
		IsHealthy:         true,
		LastFullBackupAt:  now,
	}

	config.FailoverNodes = append(config.FailoverNodes, node)
	config.UpdatedAt = now

	// Update status and geographic diversity
	bm.updateConfigurationStatus(config)

	return nil
}

// updateConfigurationStatus evaluates configuration readiness
func (bm *BackupManager) updateConfigurationStatus(config *BackupConfiguration) {
	// Check minimum node count
	if len(config.FailoverNodes) < MinimumFailoverNodes {
		config.Status = NOT_CONFIGURED
		config.HealthyFailoverCount = 0
		config.IsGeographicallyDiversified = false
		return
	}

	// Count healthy nodes
	healthyCount := 0
	regions := make(map[string]bool)

	for _, node := range config.FailoverNodes {
		regions[node.Region] = true
		if node.IsHealthy {
			healthyCount++
		}
	}

	config.HealthyFailoverCount = healthyCount
	config.IsGeographicallyDiversified = len(regions) >= (MinimumGeographicSeparation + 1)

	// Update status
	if healthyCount < MinimumFailoverNodes {
		config.Status = DEGRADED
	} else if config.LastTestResult != nil && config.LastTestResult.Success {
		config.Status = TESTED_PASSING
	} else if config.LastTestResult != nil {
		config.Status = TESTED_FAILING
	} else {
		config.Status = UNTESTED
	}
}

// RecordSynchronization updates replication status for a node
func (bm *BackupManager) RecordSynchronization(operatorID string, nodeID string, replicationLag time.Duration, isHealthy bool) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	config, exists := bm.configs[operatorID]
	if !exists {
		return fmt.Errorf("no configuration found for operator %s", operatorID)
	}

	now := time.Now()
	for i, node := range config.FailoverNodes {
		if node.NodeID == nodeID {
			config.FailoverNodes[i].ReplicationLag = replicationLag
			config.FailoverNodes[i].IsHealthy = isHealthy
			config.FailoverNodes[i].SyncedAt = now
			config.FailoverNodes[i].LastHealthCheckAt = now

			// Update replication lag on last incremental
			if replicationLag <= RPOTarget {
				config.FailoverNodes[i].LastIncrementalAt = now
			}

			config.UpdatedAt = now
			bm.updateConfigurationStatus(config)
			return nil
		}
	}

	return fmt.Errorf("node %s not found in configuration", nodeID)
}

// ExecuteFailoverTest simulates a failover to test readiness
func (bm *BackupManager) ExecuteFailoverTest(operatorID string, failingNode string) (*FailoverTest, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	config, exists := bm.configs[operatorID]
	if !exists {
		return nil, fmt.Errorf("no configuration found for operator %s", operatorID)
	}

	if len(config.FailoverNodes) < MinimumFailoverNodes {
		return nil, errors.New("insufficient failover nodes for test")
	}

	startTime := time.Now()

	// Simulate failover process
	// In production, this would initiate actual failover procedures
	recoveryTime := 2 * time.Minute // Simulated recovery time
	recoveryPointAge := 30 * time.Second // Simulated RPO

	test := FailoverTest{
		TestID:          fmt.Sprintf("test_%d", startTime.Unix()),
		StartTime:       startTime,
		EndTime:         startTime.Add(recoveryTime),
		Duration:        recoveryTime,
		FailedComponent: failingNode,
		RTO:             recoveryTime,
		RPO:             recoveryPointAge,
		Success:         recoveryTime <= RTOTarget && recoveryPointAge <= RPOTarget,
		DataLoss:        recoveryPointAge > RPOTarget,
	}

	if !test.Success {
		test.ErrorMessage = fmt.Sprintf("RTO %v exceeds target %v or RPO %v exceeds target %v",
			test.RTO, RTOTarget, test.RPO, RPOTarget)
	}

	config.FailoverTestHistory = append(config.FailoverTestHistory, test)
	now := time.Now()
	config.LastTestTime = &now
	config.LastTestResult = &test
	config.UpdatedAt = now

	bm.updateConfigurationStatus(config)

	return &test, nil
}

// GetBackupStatus returns comprehensive backup status
type BackupStatusReport struct {
	OperatorID                  string
	Status                      BackupStatus
	FailoverNodeCount           int
	HealthyFailoverCount        int
	IsGeographicallyDiversified bool
	LastReplicationLag          time.Duration
	RTOTarget                   time.Duration
	ActualRTO                   time.Duration
	RPOTarget                   time.Duration
	ActualRPO                   time.Duration
	LastTestResult              *FailoverTest
	IsReadyForProduction        bool
	Recommendations             []string
}

// GetStatus returns detailed backup status report
func (bm *BackupManager) GetStatus(operatorID string) (*BackupStatusReport, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	config, exists := bm.configs[operatorID]
	if !exists {
		return nil, fmt.Errorf("no configuration found for operator %s", operatorID)
	}

	report := &BackupStatusReport{
		OperatorID:                  operatorID,
		Status:                      config.Status,
		FailoverNodeCount:           len(config.FailoverNodes),
		HealthyFailoverCount:        config.HealthyFailoverCount,
		IsGeographicallyDiversified: config.IsGeographicallyDiversified,
		RTOTarget:                   RTOTarget,
		RPOTarget:                   RPOTarget,
		LastTestResult:              config.LastTestResult,
	}

	// Calculate actual RTO/RPO from last test
	if config.LastTestResult != nil {
		report.ActualRTO = config.LastTestResult.RTO
		report.ActualRPO = config.LastTestResult.RPO
	}

	// Calculate last replication lag
	for _, node := range config.FailoverNodes {
		if node.ReplicationLag > 0 {
			report.LastReplicationLag = node.ReplicationLag
			break
		}
	}

	// Determine production readiness
	report.IsReadyForProduction = config.Status == TESTED_PASSING &&
		report.FailoverNodeCount >= MinimumFailoverNodes &&
		report.HealthyFailoverCount >= MinimumFailoverNodes &&
		report.IsGeographicallyDiversified

	// Generate recommendations
	report.Recommendations = generateRecommendations(config)

	return report, nil
}

// generateRecommendations provides actionable guidance
func generateRecommendations(config *BackupConfiguration) []string {
	var recommendations []string

	if len(config.FailoverNodes) < MinimumFailoverNodes {
		recommendations = append(recommendations,
			fmt.Sprintf("Add at least %d failover nodes", MinimumFailoverNodes))
	}

	if config.HealthyFailoverCount < MinimumFailoverNodes {
		recommendations = append(recommendations, "Investigate and fix unhealthy failover nodes")
	}

	if !config.IsGeographicallyDiversified {
		recommendations = append(recommendations, "Distribute failover nodes across at least 2 geographic regions")
	}

	if config.LastTestResult == nil {
		recommendations = append(recommendations, "Execute initial failover test")
	} else if !config.LastTestResult.Success {
		recommendations = append(recommendations, "Review and improve failover procedures")
	}

	if config.LastTestResult != nil && config.LastTestResult.RTO > RTOTarget {
		recommendations = append(recommendations, fmt.Sprintf("Optimize recovery process to meet RTO target of %v", RTOTarget))
	}

	if config.LastTestResult != nil && config.LastTestResult.RPO > RPOTarget {
		recommendations = append(recommendations, fmt.Sprintf("Increase replication frequency to meet RPO target of %v", RPOTarget))
	}

	return recommendations
}

// ListConfigurations returns all backup configurations
func (bm *BackupManager) ListConfigurations() []*BackupConfiguration {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	configs := make([]*BackupConfiguration, 0, len(bm.configs))
	for _, config := range bm.configs {
		configs = append(configs, config)
	}
	return configs
}

// BackupStatusSummary provides aggregate backup information
type BackupStatusSummary struct {
	TotalOperators          int
	ConfiguredOperators     int
	ReadyForProductionCount int
	AverageFailoverNodes    float64
	AverageHealthyNodes     float64
	AverageRTO              time.Duration
	AverageRPO              time.Duration
}

// GetStatusSummary returns overall backup statistics
func (bm *BackupManager) GetStatusSummary() BackupStatusSummary {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	summary := BackupStatusSummary{
		TotalOperators: len(bm.configs),
	}

	var totalNodes, totalHealthy, totalRTO, totalRPO int64
	for _, config := range bm.configs {
		if config.Status != NOT_CONFIGURED {
			summary.ConfiguredOperators++
		}

		if config.Status == TESTED_PASSING &&
			len(config.FailoverNodes) >= MinimumFailoverNodes &&
			config.IsGeographicallyDiversified {
			summary.ReadyForProductionCount++
		}

		totalNodes += int64(len(config.FailoverNodes))
		totalHealthy += int64(config.HealthyFailoverCount)

		if config.LastTestResult != nil {
			totalRTO += int64(config.LastTestResult.RTO)
			totalRPO += int64(config.LastTestResult.RPO)
		}
	}

	if summary.TotalOperators > 0 {
		summary.AverageFailoverNodes = float64(totalNodes) / float64(summary.TotalOperators)
		summary.AverageHealthyNodes = float64(totalHealthy) / float64(summary.TotalOperators)
	}

	if summary.ConfiguredOperators > 0 {
		summary.AverageRTO = time.Duration(totalRTO / int64(summary.ConfiguredOperators))
		summary.AverageRPO = time.Duration(totalRPO / int64(summary.ConfiguredOperators))
	}

	return summary
}
