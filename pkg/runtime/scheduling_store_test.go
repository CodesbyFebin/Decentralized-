package runtime

import (
	"context"
	"testing"
)

// TestStorePlacementRecord verifies recording placement decisions
func TestStorePlacementRecord(t *testing.T) {
	ctx := context.Background()
	store := NewSchedulingStore()

	t.Log("Testing placement record storage")

	decision := &SchedulingDecision{
		WorkloadID:    "workload-1",
		SelectedNodes: []string{"node-1", "node-2"},
		Strategy:      StrategyFirstFit,
	}

	constraints := &ResourceConstraints{
		MemoryBytes: 256 * 1024 * 1024,
		CPUShares:   200,
		DiskBytes:   2 * 1024 * 1024,
	}

	recordID, err := store.StorePlacement(ctx, "workload-1", decision, constraints)
	if err != nil {
		t.Fatalf("StorePlacement failed: %v", err)
	}

	if recordID == "" {
		t.Error("Expected non-empty record ID")
	}

	record, err := store.GetPlacementRecord(ctx, recordID)
	if err != nil {
		t.Fatalf("GetPlacementRecord failed: %v", err)
	}

	if record.WorkloadID != "workload-1" {
		t.Errorf("Expected workload-1, got %s", record.WorkloadID)
	}

	if record.Status != "SCHEDULED" {
		t.Errorf("Expected status SCHEDULED, got %s", record.Status)
	}

	if len(record.SelectedNodes) != 2 {
		t.Errorf("Expected 2 selected nodes, got %d", len(record.SelectedNodes))
	}

	t.Logf("PASS: Placement record stored successfully with ID %s", recordID)
}

// TestUpdatePlacementStatus verifies status updates
func TestUpdatePlacementStatus(t *testing.T) {
	ctx := context.Background()
	store := NewSchedulingStore()

	t.Log("Testing placement status updates")

	decision := &SchedulingDecision{
		WorkloadID:    "workload-1",
		SelectedNodes: []string{"node-1"},
		Strategy:      StrategyFirstFit,
	}

	recordID, err := store.StorePlacement(ctx, "workload-1", decision, nil)
	if err != nil {
		t.Fatalf("StorePlacement failed: %v", err)
	}

	// Verify initial status
	record, _ := store.GetPlacementRecord(ctx, recordID)
	if record.Status != "SCHEDULED" {
		t.Fatalf("Expected initial status SCHEDULED, got %s", record.Status)
	}

	// Update to RUNNING
	if err := store.UpdatePlacementStatus(ctx, recordID, "RUNNING", "Workload started"); err != nil {
		t.Fatalf("UpdatePlacementStatus failed: %v", err)
	}

	// Verify updated status
	record, _ = store.GetPlacementRecord(ctx, recordID)
	if record.Status != "RUNNING" {
		t.Errorf("Expected updated status RUNNING, got %s", record.Status)
	}

	// Update to TERMINATED
	if err := store.UpdatePlacementStatus(ctx, recordID, "TERMINATED", "Workload stopped"); err != nil {
		t.Fatalf("UpdatePlacementStatus failed: %v", err)
	}

	record, _ = store.GetPlacementRecord(ctx, recordID)
	if record.Status != "TERMINATED" {
		t.Errorf("Expected final status TERMINATED, got %s", record.Status)
	}

	t.Logf("PASS: Placement status updates working correctly")
}

// TestWorkloadPlacementHistory retrieves all placements for a workload
func TestWorkloadPlacementHistory(t *testing.T) {
	ctx := context.Background()
	store := NewSchedulingStore()

	t.Log("Testing workload placement history retrieval")

	// Store multiple placements for same workload
	for i := 1; i <= 3; i++ {
		decision := &SchedulingDecision{
			WorkloadID:    "workload-1",
			SelectedNodes: []string{"node-1"},
			Strategy:      StrategyFirstFit,
		}

		_, err := store.StorePlacement(ctx, "workload-1", decision, nil)
		if err != nil {
			t.Fatalf("StorePlacement %d failed: %v", i, err)
		}
	}

	// Retrieve history
	placements, err := store.GetWorkloadPlacements(ctx, "workload-1")
	if err != nil {
		t.Fatalf("GetWorkloadPlacements failed: %v", err)
	}

	if len(placements) != 3 {
		t.Errorf("Expected 3 placements, got %d", len(placements))
	}

	t.Logf("PASS: Retrieved %d placement records for workload", len(placements))
}

// TestPlacementsByStrategy filters placements by strategy
func TestPlacementsByStrategy(t *testing.T) {
	ctx := context.Background()
	store := NewSchedulingStore()

	t.Log("Testing placement filtering by strategy")

	strategies := []SchedulingStrategy{StrategyFirstFit, StrategyBestFit, StrategySpreadOut}
	for i, strategy := range strategies {
		wlID := "workload-" + string(rune('0'+i+1))
		decision := &SchedulingDecision{
			WorkloadID:    wlID,
			SelectedNodes: []string{"node-1"},
			Strategy:      strategy,
		}

		_, err := store.StorePlacement(ctx, wlID, decision, nil)
		if err != nil {
			t.Fatalf("StorePlacement failed: %v", err)
		}
	}

	// Query by strategy
	firstFitPlacements, err := store.GetPlacementsByStrategy(ctx, StrategyFirstFit)
	if err != nil {
		t.Fatalf("GetPlacementsByStrategy failed: %v", err)
	}

	if len(firstFitPlacements) != 1 {
		t.Errorf("Expected 1 FIRST_FIT placement, got %d", len(firstFitPlacements))
	}

	if firstFitPlacements[0].Strategy != StrategyFirstFit {
		t.Errorf("Expected strategy FIRST_FIT, got %s", firstFitPlacements[0].Strategy)
	}

	t.Logf("PASS: Strategy filtering working correctly")
}

// TestPlacementsByStatus filters placements by status
func TestPlacementsByStatus(t *testing.T) {
	ctx := context.Background()
	store := NewSchedulingStore()

	t.Log("Testing placement filtering by status")

	// Create 3 placements with different statuses
	recordIDs := []string{}
	for i := 1; i <= 3; i++ {
		wlID := "workload-" + string(rune('0'+i))
		decision := &SchedulingDecision{
			WorkloadID:    wlID,
			SelectedNodes: []string{"node-1"},
			Strategy:      StrategyFirstFit,
		}

		recordID, _ := store.StorePlacement(ctx, wlID, decision, nil)
		recordIDs = append(recordIDs, recordID)
	}

	// Update statuses
	store.UpdatePlacementStatus(ctx, recordIDs[0], "RUNNING", "Started")
	store.UpdatePlacementStatus(ctx, recordIDs[1], "RUNNING", "Started")
	store.UpdatePlacementStatus(ctx, recordIDs[2], "TERMINATED", "Stopped")

	// Query by status
	runningPlacements, err := store.GetPlacementsByStatus(ctx, "RUNNING")
	if err != nil {
		t.Fatalf("GetPlacementsByStatus failed: %v", err)
	}

	if len(runningPlacements) != 2 {
		t.Errorf("Expected 2 RUNNING placements, got %d", len(runningPlacements))
	}

	for _, p := range runningPlacements {
		if p.Status != "RUNNING" {
			t.Errorf("Expected status RUNNING, got %s", p.Status)
		}
	}

	t.Logf("PASS: Status filtering working correctly")
}

// TestAuditTrail tracks status transitions
func TestAuditTrail(t *testing.T) {
	ctx := context.Background()
	store := NewSchedulingStore()

	t.Log("Testing audit trail tracking")

	decision := &SchedulingDecision{
		WorkloadID:    "workload-1",
		SelectedNodes: []string{"node-1"},
		Strategy:      StrategyFirstFit,
	}

	recordID, _ := store.StorePlacement(ctx, "workload-1", decision, nil)

	// Make multiple status changes
	store.UpdatePlacementStatus(ctx, recordID, "RUNNING", "Started")
	store.UpdatePlacementStatus(ctx, recordID, "TERMINATED", "Stopped")

	// Retrieve audit trail
	auditTrail, err := store.GetAuditTrail(ctx, recordID)
	if err != nil {
		t.Fatalf("GetAuditTrail failed: %v", err)
	}

	// Should have 3 entries: creation + 2 updates
	if len(auditTrail) != 3 {
		t.Errorf("Expected 3 audit entries, got %d", len(auditTrail))
	}

	// Verify chronological order
	if auditTrail[0].NewStatus != "SCHEDULED" {
		t.Errorf("Expected first entry status SCHEDULED, got %s", auditTrail[0].NewStatus)
	}

	if auditTrail[1].NewStatus != "RUNNING" {
		t.Errorf("Expected second entry status RUNNING, got %s", auditTrail[1].NewStatus)
	}

	if auditTrail[2].NewStatus != "TERMINATED" {
		t.Errorf("Expected third entry status TERMINATED, got %s", auditTrail[2].NewStatus)
	}

	t.Logf("PASS: Audit trail tracking %d entries in chronological order", len(auditTrail))
}

// TestAllPlacements retrieves all records
func TestAllPlacements(t *testing.T) {
	ctx := context.Background()
	store := NewSchedulingStore()

	t.Log("Testing retrieval of all placements")

	// Store multiple placements
	for i := 1; i <= 5; i++ {
		wlID := "workload-" + string(rune('0'+i))
		decision := &SchedulingDecision{
			WorkloadID:    wlID,
			SelectedNodes: []string{"node-1"},
			Strategy:      StrategyFirstFit,
		}

		_, err := store.StorePlacement(ctx, wlID, decision, nil)
		if err != nil {
			t.Fatalf("StorePlacement failed: %v", err)
		}
	}

	// Retrieve all
	allPlacements, err := store.GetAllPlacements(ctx)
	if err != nil {
		t.Fatalf("GetAllPlacements failed: %v", err)
	}

	if len(allPlacements) != 5 {
		t.Errorf("Expected 5 placements, got %d", len(allPlacements))
	}

	t.Logf("PASS: Retrieved all %d placement records", len(allPlacements))
}

// TestPlacementStats provides statistics
func TestPlacementStats(t *testing.T) {
	ctx := context.Background()
	store := NewSchedulingStore()

	t.Log("Testing placement statistics")

	// Create placements with different strategies and statuses
	for i := 1; i <= 4; i++ {
		wlID := "workload-" + string(rune('0'+i))
		strategy := StrategyFirstFit
		if i == 2 {
			strategy = StrategyBestFit
		}

		decision := &SchedulingDecision{
			WorkloadID:    wlID,
			SelectedNodes: []string{"node-1"},
			Strategy:      strategy,
		}

		recordID, _ := store.StorePlacement(ctx, wlID, decision, nil)

		// Update some to RUNNING
		if i > 2 {
			store.UpdatePlacementStatus(ctx, recordID, "RUNNING", "Started")
		}
	}

	stats := store.GetPlacementStats(ctx)

	if stats["total_placements"] != 4 {
		t.Errorf("Expected 4 placements in stats, got %d", stats["total_placements"])
	}

	if stats["total_workloads"] != 4 {
		t.Errorf("Expected 4 workloads in stats, got %d", stats["total_workloads"])
	}

	statusCounts := stats["by_status"].(map[string]int)
	if statusCounts["SCHEDULED"] != 2 {
		t.Errorf("Expected 2 SCHEDULED, got %d", statusCounts["SCHEDULED"])
	}

	if statusCounts["RUNNING"] != 2 {
		t.Errorf("Expected 2 RUNNING, got %d", statusCounts["RUNNING"])
	}

	t.Logf("PASS: Statistics: %v", stats)
}

// TestDeletePlacementRecord removes records and audit trail
func TestDeletePlacementRecord(t *testing.T) {
	ctx := context.Background()
	store := NewSchedulingStore()

	t.Log("Testing placement record deletion")

	decision := &SchedulingDecision{
		WorkloadID:    "workload-1",
		SelectedNodes: []string{"node-1"},
		Strategy:      StrategyFirstFit,
	}

	recordID, _ := store.StorePlacement(ctx, "workload-1", decision, nil)

	// Update status to create audit entries
	store.UpdatePlacementStatus(ctx, recordID, "RUNNING", "Started")

	// Verify exists
	_, err := store.GetPlacementRecord(ctx, recordID)
	if err != nil {
		t.Fatalf("Record should exist before deletion")
	}

	// Delete
	if err := store.DeletePlacementRecord(ctx, recordID); err != nil {
		t.Fatalf("DeletePlacementRecord failed: %v", err)
	}

	// Verify deleted
	_, err = store.GetPlacementRecord(ctx, recordID)
	if err == nil {
		t.Error("Record should not exist after deletion")
	}

	// Verify audit trail also deleted
	auditTrail, _ := store.GetAuditTrail(ctx, recordID)
	if len(auditTrail) != 0 {
		t.Errorf("Expected no audit entries after deletion, got %d", len(auditTrail))
	}

	t.Logf("PASS: Placement record and audit trail deleted successfully")
}

// TestClearAllRecords removes all data
func TestClearAllRecords(t *testing.T) {
	ctx := context.Background()
	store := NewSchedulingStore()

	t.Log("Testing clear all records")

	// Store multiple records
	for i := 1; i <= 3; i++ {
		wlID := "workload-" + string(rune('0'+i))
		decision := &SchedulingDecision{
			WorkloadID:    wlID,
			SelectedNodes: []string{"node-1"},
			Strategy:      StrategyFirstFit,
		}

		store.StorePlacement(ctx, wlID, decision, nil)
	}

	// Verify data exists
	allPlacements, _ := store.GetAllPlacements(ctx)
	if len(allPlacements) != 3 {
		t.Fatalf("Expected 3 placements before clear")
	}

	// Clear
	if err := store.ClearAllRecords(ctx); err != nil {
		t.Fatalf("ClearAllRecords failed: %v", err)
	}

	// Verify cleared
	allPlacements, _ = store.GetAllPlacements(ctx)
	if len(allPlacements) != 0 {
		t.Errorf("Expected 0 placements after clear, got %d", len(allPlacements))
	}

	stats := store.GetPlacementStats(ctx)
	if stats["total_placements"] != 0 {
		t.Errorf("Expected 0 in stats, got %d", stats["total_placements"])
	}

	t.Logf("PASS: All records cleared successfully")
}

// TestWorkloadAuditHistory retrieves all audit entries for workload
func TestWorkloadAuditHistory(t *testing.T) {
	ctx := context.Background()
	store := NewSchedulingStore()

	t.Log("Testing workload audit history")

	recordID, _ := store.StorePlacement(ctx, "workload-1",
		&SchedulingDecision{
			WorkloadID:    "workload-1",
			SelectedNodes: []string{"node-1"},
			Strategy:      StrategyFirstFit,
		}, nil)

	store.UpdatePlacementStatus(ctx, recordID, "RUNNING", "Started")
	store.UpdatePlacementStatus(ctx, recordID, "TERMINATED", "Stopped")

	// Get audit history
	history, err := store.GetAuditHistory(ctx, "workload-1")
	if err != nil {
		t.Fatalf("GetAuditHistory failed: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("Expected 3 audit entries, got %d", len(history))
	}

	// Verify chronological order
	for i := 0; i < len(history)-1; i++ {
		if history[i].Timestamp > history[i+1].Timestamp {
			t.Error("Audit history not in chronological order")
		}
	}

	t.Logf("PASS: Workload audit history with %d entries in order", len(history))
}
