package runtime

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// PlacementRecord stores a single workload placement decision
type PlacementRecord struct {
	RecordID       string
	WorkloadID     string
	SelectedNodes  []string
	Strategy       SchedulingStrategy
	Constraints    *ResourceConstraints
	Timestamp      int64
	Status         string // SCHEDULED, RUNNING, TERMINATED, FAILED
	StatusReason   string
	DecisionTime   time.Duration
}

// AuditEntry tracks state changes in placement
type AuditEntry struct {
	EntryID     string
	RecordID    string
	WorkloadID  string
	OldStatus   string
	NewStatus   string
	Reason      string
	Timestamp   int64
}

// SchedulingStore provides persistent storage for scheduling decisions
type SchedulingStore struct {
	placements map[string]*PlacementRecord
	auditLog   map[string]*AuditEntry
	indices    map[string][]string // workloadID -> recordIDs
	mu         sync.RWMutex
	nextID     int64
}

// NewSchedulingStore creates a new scheduling store
func NewSchedulingStore() *SchedulingStore {
	return &SchedulingStore{
		placements: make(map[string]*PlacementRecord),
		auditLog:   make(map[string]*AuditEntry),
		indices:    make(map[string][]string),
		nextID:     1,
	}
}

// generateID generates a unique record ID
func (ss *SchedulingStore) generateID() string {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	id := ss.nextID
	ss.nextID++
	return fmt.Sprintf("RECORD-%d", id)
}

// generateAuditID generates a unique audit entry ID
func (ss *SchedulingStore) generateAuditID() string {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	id := ss.nextID
	ss.nextID++
	return fmt.Sprintf("AUDIT-%d", id)
}

// StorePlacement records a new scheduling decision
func (ss *SchedulingStore) StorePlacement(ctx context.Context,
	workloadID string, decision *SchedulingDecision, constraints *ResourceConstraints) (string, error) {

	if workloadID == "" {
		return "", fmt.Errorf("workload ID cannot be empty")
	}

	if decision == nil {
		return "", fmt.Errorf("decision cannot be nil")
	}

	recordID := ss.generateID()

	record := &PlacementRecord{
		RecordID:      recordID,
		WorkloadID:    workloadID,
		SelectedNodes: decision.SelectedNodes,
		Strategy:      decision.Strategy,
		Constraints:   constraints,
		Timestamp:     time.Now().UnixNano(),
		Status:        "SCHEDULED",
		StatusReason:  "Placement decision recorded",
		DecisionTime:  time.Since(time.Unix(0, decision.DecisionTime)),
	}

	ss.mu.Lock()
	ss.placements[recordID] = record
	ss.indices[workloadID] = append(ss.indices[workloadID], recordID)
	ss.mu.Unlock()

	// Add audit entry
	auditID := ss.generateAuditID()
	auditEntry := &AuditEntry{
		EntryID:    auditID,
		RecordID:   recordID,
		WorkloadID: workloadID,
		OldStatus:  "",
		NewStatus:  "SCHEDULED",
		Reason:     "Initial placement recorded",
		Timestamp:  record.Timestamp,
	}

	ss.mu.Lock()
	ss.auditLog[auditID] = auditEntry
	ss.mu.Unlock()

	return recordID, nil
}

// UpdatePlacementStatus updates the status of a placement record
func (ss *SchedulingStore) UpdatePlacementStatus(ctx context.Context,
	recordID string, newStatus string, reason string) error {

	ss.mu.Lock()
	record, ok := ss.placements[recordID]
	if !ok {
		ss.mu.Unlock()
		return fmt.Errorf("placement record not found: %s", recordID)
	}

	oldStatus := record.Status
	record.Status = newStatus
	record.StatusReason = reason
	ss.mu.Unlock()

	// Add audit entry
	auditID := ss.generateAuditID()
	auditEntry := &AuditEntry{
		EntryID:    auditID,
		RecordID:   recordID,
		WorkloadID: record.WorkloadID,
		OldStatus:  oldStatus,
		NewStatus:  newStatus,
		Reason:     reason,
		Timestamp:  time.Now().UnixNano(),
	}

	ss.mu.Lock()
	ss.auditLog[auditID] = auditEntry
	ss.mu.Unlock()

	return nil
}

// GetPlacementRecord retrieves a specific placement record
func (ss *SchedulingStore) GetPlacementRecord(ctx context.Context, recordID string) (*PlacementRecord, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	record, ok := ss.placements[recordID]
	if !ok {
		return nil, fmt.Errorf("placement record not found: %s", recordID)
	}

	return record, nil
}

// GetWorkloadPlacements retrieves all placement records for a workload
func (ss *SchedulingStore) GetWorkloadPlacements(ctx context.Context, workloadID string) ([]*PlacementRecord, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	recordIDs, ok := ss.indices[workloadID]
	if !ok {
		return []*PlacementRecord{}, nil
	}

	records := []*PlacementRecord{}
	for _, recordID := range recordIDs {
		if record, ok := ss.placements[recordID]; ok {
			records = append(records, record)
		}
	}

	return records, nil
}

// GetPlacementsByStrategy retrieves placements using a specific strategy
func (ss *SchedulingStore) GetPlacementsByStrategy(ctx context.Context,
	strategy SchedulingStrategy) ([]*PlacementRecord, error) {

	ss.mu.RLock()
	defer ss.mu.RUnlock()

	records := []*PlacementRecord{}
	for _, record := range ss.placements {
		if record.Strategy == strategy {
			records = append(records, record)
		}
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp > records[j].Timestamp
	})

	return records, nil
}

// GetPlacementsByStatus retrieves placements in a specific status
func (ss *SchedulingStore) GetPlacementsByStatus(ctx context.Context,
	status string) ([]*PlacementRecord, error) {

	ss.mu.RLock()
	defer ss.mu.RUnlock()

	records := []*PlacementRecord{}
	for _, record := range ss.placements {
		if record.Status == status {
			records = append(records, record)
		}
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp > records[j].Timestamp
	})

	return records, nil
}

// GetAllPlacements retrieves all placement records
func (ss *SchedulingStore) GetAllPlacements(ctx context.Context) ([]*PlacementRecord, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	records := []*PlacementRecord{}
	for _, record := range ss.placements {
		records = append(records, record)
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp > records[j].Timestamp
	})

	return records, nil
}

// GetAuditTrail retrieves audit entries for a placement record
func (ss *SchedulingStore) GetAuditTrail(ctx context.Context, recordID string) ([]*AuditEntry, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	entries := []*AuditEntry{}
	for _, entry := range ss.auditLog {
		if entry.RecordID == recordID {
			entries = append(entries, entry)
		}
	}

	// Sort by timestamp ascending (chronological order)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp < entries[j].Timestamp
	})

	return entries, nil
}

// GetAuditHistory retrieves all audit entries for a workload
func (ss *SchedulingStore) GetAuditHistory(ctx context.Context, workloadID string) ([]*AuditEntry, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	entries := []*AuditEntry{}
	for _, entry := range ss.auditLog {
		if entry.WorkloadID == workloadID {
			entries = append(entries, entry)
		}
	}

	// Sort by timestamp ascending (chronological order)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp < entries[j].Timestamp
	})

	return entries, nil
}

// ListAllAuditEntries retrieves all audit entries
func (ss *SchedulingStore) ListAllAuditEntries(ctx context.Context) ([]*AuditEntry, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	entries := []*AuditEntry{}
	for _, entry := range ss.auditLog {
		entries = append(entries, entry)
	}

	// Sort by timestamp ascending (chronological order)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp < entries[j].Timestamp
	})

	return entries, nil
}

// GetPlacementStats returns statistics about placements
func (ss *SchedulingStore) GetPlacementStats(ctx context.Context) map[string]interface{} {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	stats := map[string]interface{}{
		"total_placements": len(ss.placements),
		"total_workloads":  len(ss.indices),
		"audit_entries":    len(ss.auditLog),
	}

	// Count by status
	statusCounts := make(map[string]int)
	for _, record := range ss.placements {
		statusCounts[record.Status]++
	}
	stats["by_status"] = statusCounts

	// Count by strategy
	strategyCounts := make(map[string]int)
	for _, record := range ss.placements {
		strategyCounts[string(record.Strategy)]++
	}
	stats["by_strategy"] = strategyCounts

	return stats
}

// DeletePlacementRecord removes a placement record and its audit trail
func (ss *SchedulingStore) DeletePlacementRecord(ctx context.Context, recordID string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	record, ok := ss.placements[recordID]
	if !ok {
		return fmt.Errorf("placement record not found: %s", recordID)
	}

	// Remove from placements
	delete(ss.placements, recordID)

	// Remove from indices
	workloadID := record.WorkloadID
	recordIDs := ss.indices[workloadID]
	newIDs := []string{}
	for _, id := range recordIDs {
		if id != recordID {
			newIDs = append(newIDs, id)
		}
	}
	if len(newIDs) == 0 {
		delete(ss.indices, workloadID)
	} else {
		ss.indices[workloadID] = newIDs
	}

	// Remove audit entries
	auditToDelete := []string{}
	for auditID, entry := range ss.auditLog {
		if entry.RecordID == recordID {
			auditToDelete = append(auditToDelete, auditID)
		}
	}
	for _, auditID := range auditToDelete {
		delete(ss.auditLog, auditID)
	}

	return nil
}

// ClearAllRecords removes all placements and audit entries
func (ss *SchedulingStore) ClearAllRecords(ctx context.Context) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.placements = make(map[string]*PlacementRecord)
	ss.auditLog = make(map[string]*AuditEntry)
	ss.indices = make(map[string][]string)
	ss.nextID = 1

	return nil
}
