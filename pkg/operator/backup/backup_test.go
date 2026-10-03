package backup

import (
	"testing"
	"time"
)

func TestCreateConfiguration(t *testing.T) {
	bm := NewBackupManager()

	config, err := bm.CreateConfiguration("op_001", "daily", 30)
	if err != nil {
		t.Fatalf("CreateConfiguration() error = %v", err)
	}

	if config.Status != NOT_CONFIGURED {
		t.Errorf("expected NOT_CONFIGURED status, got %s", config.Status)
	}

	if config.RetentionDays != 30 {
		t.Errorf("expected retention 30 days, got %d", config.RetentionDays)
	}
}

func TestCreateConfigurationInvalidRetention(t *testing.T) {
	bm := NewBackupManager()

	_, err := bm.CreateConfiguration("op_001", "daily", 3)
	if err == nil {
		t.Error("expected error for invalid retention period")
	}
}

func TestAddFailoverNode(t *testing.T) {
	bm := NewBackupManager()
	bm.CreateConfiguration("op_001", "daily", 30)

	err := bm.AddFailoverNode("op_001", "node_001", "us-east", true)
	if err != nil {
		t.Fatalf("AddFailoverNode() error = %v", err)
	}

	config, _ := bm.GetConfiguration("op_001")
	if len(config.FailoverNodes) != 1 {
		t.Errorf("expected 1 failover node, got %d", len(config.FailoverNodes))
	}
}

func TestAddDuplicateNode(t *testing.T) {
	bm := NewBackupManager()
	bm.CreateConfiguration("op_001", "daily", 30)

	bm.AddFailoverNode("op_001", "node_001", "us-east", true)

	// Try to add same node again
	err := bm.AddFailoverNode("op_001", "node_001", "us-east", true)
	if err == nil {
		t.Error("expected error adding duplicate node")
	}
}

func TestConfigurationStatus(t *testing.T) {
	bm := NewBackupManager()
	bm.CreateConfiguration("op_001", "daily", 30)

	// Initially not configured
	config, _ := bm.GetConfiguration("op_001")
	if config.Status != NOT_CONFIGURED {
		t.Errorf("expected NOT_CONFIGURED, got %s", config.Status)
	}

	// Add minimum failover nodes
	bm.AddFailoverNode("op_001", "node_001", "us-east", true)
	bm.AddFailoverNode("op_001", "node_002", "us-west", false)

	config, _ = bm.GetConfiguration("op_001")
	if config.Status == NOT_CONFIGURED {
		t.Error("expected status to change from NOT_CONFIGURED")
	}

	if !config.IsGeographicallyDiversified {
		t.Error("expected geographic diversity with 2 regions")
	}
}

func TestRecordSynchronization(t *testing.T) {
	bm := NewBackupManager()
	bm.CreateConfiguration("op_001", "daily", 30)
	bm.AddFailoverNode("op_001", "node_001", "us-east", true)

	err := bm.RecordSynchronization("op_001", "node_001", 500*time.Millisecond, true)
	if err != nil {
		t.Fatalf("RecordSynchronization() error = %v", err)
	}

	config, _ := bm.GetConfiguration("op_001")
	node := config.FailoverNodes[0]

	if node.ReplicationLag != 500*time.Millisecond {
		t.Errorf("expected replication lag 500ms, got %v", node.ReplicationLag)
	}

	if !node.IsHealthy {
		t.Error("expected node to be healthy")
	}
}

func TestExecuteFailoverTest(t *testing.T) {
	bm := NewBackupManager()
	bm.CreateConfiguration("op_001", "daily", 30)

	bm.AddFailoverNode("op_001", "node_001", "us-east", true)
	bm.AddFailoverNode("op_001", "node_002", "us-west", false)

	// Record healthy synchronization
	bm.RecordSynchronization("op_001", "node_001", 500*time.Millisecond, true)
	bm.RecordSynchronization("op_001", "node_002", 500*time.Millisecond, true)

	test, err := bm.ExecuteFailoverTest("op_001", "node_001")
	if err != nil {
		t.Fatalf("ExecuteFailoverTest() error = %v", err)
	}

	if test.TestID == "" {
		t.Error("expected test ID to be set")
	}

	config, _ := bm.GetConfiguration("op_001")
	if len(config.FailoverTestHistory) != 1 {
		t.Error("expected test to be recorded in history")
	}
}

func TestFailoverTestSuccess(t *testing.T) {
	bm := NewBackupManager()
	bm.CreateConfiguration("op_001", "daily", 30)

	bm.AddFailoverNode("op_001", "node_001", "us-east", true)
	bm.AddFailoverNode("op_001", "node_002", "us-west", false)

	test, _ := bm.ExecuteFailoverTest("op_001", "node_001")

	if test.RTO > RTOTarget {
		t.Errorf("expected RTO within target %v, got %v", RTOTarget, test.RTO)
	}

	if test.RPO > RPOTarget {
		t.Errorf("expected RPO within target %v, got %v", RPOTarget, test.RPO)
	}
}

func TestGetBackupStatus(t *testing.T) {
	bm := NewBackupManager()
	bm.CreateConfiguration("op_001", "daily", 30)

	bm.AddFailoverNode("op_001", "node_001", "us-east", true)
	bm.AddFailoverNode("op_001", "node_002", "us-west", false)

	bm.RecordSynchronization("op_001", "node_001", 500*time.Millisecond, true)
	bm.RecordSynchronization("op_001", "node_002", 500*time.Millisecond, true)

	bm.ExecuteFailoverTest("op_001", "node_001")

	status, err := bm.GetStatus("op_001")
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if status.FailoverNodeCount != 2 {
		t.Errorf("expected 2 failover nodes, got %d", status.FailoverNodeCount)
	}

	if status.HealthyFailoverCount != 2 {
		t.Errorf("expected 2 healthy nodes, got %d", status.HealthyFailoverCount)
	}
}

func TestProductionReadiness(t *testing.T) {
	bm := NewBackupManager()
	bm.CreateConfiguration("op_001", "daily", 30)

	// Add minimum failover nodes
	bm.AddFailoverNode("op_001", "node_001", "us-east", true)
	bm.AddFailoverNode("op_001", "node_002", "us-west", false)

	// Record healthy state
	bm.RecordSynchronization("op_001", "node_001", 500*time.Millisecond, true)
	bm.RecordSynchronization("op_001", "node_002", 500*time.Millisecond, true)

	// Execute successful test
	bm.ExecuteFailoverTest("op_001", "node_001")

	status, _ := bm.GetStatus("op_001")

	if !status.IsReadyForProduction {
		t.Error("expected to be ready for production")
	}
}

func TestRecommendations(t *testing.T) {
	bm := NewBackupManager()
	bm.CreateConfiguration("op_001", "daily", 30)

	status, _ := bm.GetStatus("op_001")

	if len(status.Recommendations) == 0 {
		t.Error("expected recommendations for unconfigured backup")
	}

	// Check that we get recommendation to add nodes
	hasAddNodesRec := false
	for _, rec := range status.Recommendations {
		if len(rec) > 0 {
			hasAddNodesRec = true
			break
		}
	}

	if !hasAddNodesRec {
		t.Error("expected recommendation to add failover nodes")
	}
}

func TestBackupStatusSummary(t *testing.T) {
	bm := NewBackupManager()

	for i := 0; i < 5; i++ {
		operatorID := "op_00" + string(rune('1'+i))
		bm.CreateConfiguration(operatorID, "daily", 30)

		if i < 2 {
			// First 2 ready for production
			bm.AddFailoverNode(operatorID, "node_1", "us-east", true)
			bm.AddFailoverNode(operatorID, "node_2", "us-west", false)
			bm.RecordSynchronization(operatorID, "node_1", 500*time.Millisecond, true)
			bm.RecordSynchronization(operatorID, "node_2", 500*time.Millisecond, true)
			bm.ExecuteFailoverTest(operatorID, "node_1")
		}
	}

	summary := bm.GetStatusSummary()
	if summary.TotalOperators != 5 {
		t.Errorf("expected 5 operators, got %d", summary.TotalOperators)
	}

	if summary.ConfiguredOperators != 2 {
		t.Errorf("expected 2 configured (with nodes), got %d", summary.ConfiguredOperators)
	}

	if summary.ReadyForProductionCount != 2 {
		t.Errorf("expected 2 production ready, got %d", summary.ReadyForProductionCount)
	}
}

func TestListConfigurations(t *testing.T) {
	bm := NewBackupManager()

	count := 10
	for i := 0; i < count; i++ {
		operatorID := "op_" + string(rune('0'+i))
		bm.CreateConfiguration(operatorID, "daily", 30)
	}

	configs := bm.ListConfigurations()
	if len(configs) != count {
		t.Errorf("expected %d configurations, got %d", count, len(configs))
	}
}

func TestDegradedStatus(t *testing.T) {
	bm := NewBackupManager()
	bm.CreateConfiguration("op_001", "daily", 30)

	bm.AddFailoverNode("op_001", "node_001", "us-east", true)
	bm.AddFailoverNode("op_001", "node_002", "us-west", false)

	// Mark one as unhealthy
	bm.RecordSynchronization("op_001", "node_001", 500*time.Millisecond, true)
	bm.RecordSynchronization("op_001", "node_002", 500*time.Millisecond, false)

	config, _ := bm.GetConfiguration("op_001")
	if config.Status != DEGRADED {
		t.Errorf("expected DEGRADED status, got %s", config.Status)
	}

	if config.HealthyFailoverCount != 1 {
		t.Errorf("expected 1 healthy node, got %d", config.HealthyFailoverCount)
	}
}
