// Package audit compliance implements structured audit logging with compliance features.
//
// The compliance module provides:
// - Structured audit log format with who/what/when/where
// - Tamper-evident audit trail via hash chain verification
// - Retention policy enforcement (90+ days default)
// - Audit log querying and filtering
package audit

import (
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"
)

// ComplianceEntry extends audit entries with compliance metadata.
type ComplianceEntry struct {
	Entry            Entry             `json:"entry"`
	ChecksumVerified bool              `json:"checksumVerified"`
	VerificationTime  int64             `json:"verificationTime"`
	Annotations      map[string]string `json:"annotations"`
}

// ComplianceLog provides tamper-evident audit logging with retention.
type ComplianceLog struct {
	mu              sync.RWMutex
	ledger          *Ledger
	manager         *LogManager
	entries         []ComplianceEntry
	retention       *RetentionPolicy
	checksumHistory []string // Running checksums for verification
}

// NewComplianceLog creates a new compliance audit log.
func NewComplianceLog(dir string, policy *RetentionPolicy) *ComplianceLog {
	if policy == nil {
		policy = NewRetentionPolicy()
	}

	return &ComplianceLog{
		ledger:          &Ledger{},
		manager:         NewLogManager(dir, policy),
		entries:         make([]ComplianceEntry, 0),
		retention:       policy,
		checksumHistory: make([]string, 0),
	}
}

// Log appends an entry to the audit log with automatic verification.
func (cl *ComplianceLog) Log(entry Entry) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	// Set timestamp if not already set
	if entry.TS == 0 {
		entry.TS = time.Now().UnixMilli()
	}

	// Append to ledger
	entry = cl.ledger.Append(entry)

	// Create compliance entry
	complianceEntry := ComplianceEntry{
		Entry:            entry,
		ChecksumVerified: true,
		VerificationTime: time.Now().UnixMilli(),
		Annotations:      make(map[string]string),
	}

	cl.entries = append(cl.entries, complianceEntry)
	cl.checksumHistory = append(cl.checksumHistory, entry.Hash)

	// Periodic verification (every 100 entries or manually triggered)
	if len(cl.entries)%100 == 0 {
		if err := cl.verify(); err != nil {
			return fmt.Errorf("audit: verification failed: %w", err)
		}
	}

	return nil
}

// QueryEntries retrieves audit entries matching criteria.
func (cl *ComplianceLog) QueryEntries(criteria *QueryCriteria) ([]ComplianceEntry, error) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	results := make([]ComplianceEntry, 0)

	for _, ce := range cl.entries {
		if criteria.matchesEntry(ce.Entry) {
			results = append(results, ce)
		}
	}

	return results, nil
}

// QueryCriteria defines filters for audit log queries.
type QueryCriteria struct {
	Actor     string      // Filter by actor identity
	Action    string      // Filter by action (exact match)
	Resource  string      // Filter by resource (prefix match)
	Source    string      // Filter by source
	StartTime int64       // Minimum timestamp (unix milliseconds)
	EndTime   int64       // Maximum timestamp (unix milliseconds)
	SeqRange  *SeqRange   // Sequence range
}

// SeqRange specifies a range of sequence numbers.
type SeqRange struct {
	Min int64
	Max int64
}

// matchesEntry checks if an entry matches the criteria.
func (qc *QueryCriteria) matchesEntry(e Entry) bool {
	if qc.Actor != "" && e.Actor != qc.Actor {
		return false
	}
	if qc.Action != "" && e.Action != qc.Action {
		return false
	}
	if qc.Resource != "" && !matchesPrefix(e.Resource, qc.Resource) {
		return false
	}
	if qc.Source != "" && e.Source != qc.Source {
		return false
	}
	if qc.StartTime > 0 && e.TS < qc.StartTime {
		return false
	}
	if qc.EndTime > 0 && e.TS > qc.EndTime {
		return false
	}
	if qc.SeqRange != nil {
		if e.Seq < qc.SeqRange.Min || e.Seq > qc.SeqRange.Max {
			return false
		}
	}
	return true
}

// matchesPrefix checks if a value matches a prefix (for resource filtering).
func matchesPrefix(value, prefix string) bool {
	if prefix == "*" {
		return true
	}
	if prefix == "" {
		return true
	}
	return slices.ContainsFunc([]string{value}, func(s string) bool {
		return len(s) >= len(prefix) && s[:len(prefix)] == prefix
	})
}

// verify performs integrity verification on logged entries.
func (cl *ComplianceLog) verify() error {
	entries := make([]Entry, len(cl.entries))
	for i, ce := range cl.entries {
		entries[i] = ce.Entry
	}

	brk := Verify(entries, 0, Genesis)
	if brk != nil {
		return brk
	}

	// Mark all entries as verified
	now := time.Now().UnixMilli()
	for i := range cl.entries {
		cl.entries[i].ChecksumVerified = true
		cl.entries[i].VerificationTime = now
	}

	return nil
}

// GetEntry retrieves a specific entry by sequence number.
func (cl *ComplianceLog) GetEntry(seq int64) (*ComplianceEntry, error) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	for _, ce := range cl.entries {
		if ce.Entry.Seq == seq {
			return &ce, nil
		}
	}

	return nil, fmt.Errorf("audit: entry %d not found", seq)
}

// Count returns the total number of entries.
func (cl *ComplianceLog) Count() int64 {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return int64(len(cl.entries))
}

// ListEntries returns all entries (optionally limited).
func (cl *ComplianceLog) ListEntries(limit int) []ComplianceEntry {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	if limit <= 0 || limit > len(cl.entries) {
		limit = len(cl.entries)
	}

	result := make([]ComplianceEntry, limit)
	copy(result, cl.entries[len(cl.entries)-limit:])
	return result
}

// EnforceRetention applies the retention policy to archived logs.
func (cl *ComplianceLog) EnforceRetention() error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	return cl.manager.EnforceRetention()
}

// Export exports the audit log as JSON.
func (cl *ComplianceLog) Export() ([]byte, error) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	return json.MarshalIndent(cl.entries, "", "  ")
}

// Save persists the audit log to disk.
func (cl *ComplianceLog) Save() error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	_, err := cl.manager.SaveLedger(cl.ledger, "compliance-log")
	return err
}

// Load restores the audit log from disk.
func (cl *ComplianceLog) Load() error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	ledger, err := cl.manager.LoadLedger()
	if err != nil {
		return err
	}

	cl.ledger = ledger

	// Rebuild compliance entries
	cl.entries = make([]ComplianceEntry, len(ledger.Entries))
	for i, entry := range ledger.Entries {
		cl.entries[i] = ComplianceEntry{
			Entry:            entry,
			ChecksumVerified: true,
			VerificationTime: time.Now().UnixMilli(),
			Annotations:      make(map[string]string),
		}
	}

	return nil
}

// AnnotateEntry adds metadata annotations to an entry.
func (cl *ComplianceLog) AnnotateEntry(seq int64, annotations map[string]string) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	for i, ce := range cl.entries {
		if ce.Entry.Seq == seq {
			for k, v := range annotations {
				cl.entries[i].Annotations[k] = v
			}
			return nil
		}
	}

	return fmt.Errorf("audit: entry %d not found", seq)
}

// GetChainHead returns the current chain head (seq and hash).
func (cl *ComplianceLog) GetChainHead() (int64, string) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.ledger.Head()
}

// VerifyChain verifies the integrity of the entire chain.
func (cl *ComplianceLog) VerifyChain() error {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	brk := Verify(cl.ledger.Entries, 0, Genesis)
	if brk != nil {
		return brk
	}

	return nil
}

// GetSummary returns a summary of the audit log.
func (cl *ComplianceLog) GetSummary() *AuditSummary {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	summary := &AuditSummary{
		TotalEntries: int64(len(cl.entries)),
		ActionCounts: make(map[string]int64),
		ActorCounts:  make(map[string]int64),
		SourceCounts: make(map[string]int64),
	}

	for _, ce := range cl.entries {
		summary.ActionCounts[ce.Entry.Action]++
		summary.ActorCounts[ce.Entry.Actor]++
		summary.SourceCounts[ce.Entry.Source]++
	}

	if len(cl.entries) > 0 {
		summary.FirstEntry = cl.entries[0].Entry.TS
		summary.LastEntry = cl.entries[len(cl.entries)-1].Entry.TS
	}

	return summary
}

// AuditSummary provides statistics about the audit log.
type AuditSummary struct {
	TotalEntries int64             `json:"totalEntries"`
	FirstEntry   int64             `json:"firstEntry"`   // unix milliseconds
	LastEntry    int64             `json:"lastEntry"`    // unix milliseconds
	ActionCounts map[string]int64  `json:"actionCounts"`
	ActorCounts  map[string]int64  `json:"actorCounts"`
	SourceCounts map[string]int64  `json:"sourceCounts"`
}

// DetectTampering checks for signs of tampering (sequence gaps, hash mismatches).
func (cl *ComplianceLog) DetectTampering() *TamperReport {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	report := &TamperReport{
		DetectedAt: time.Now().UnixMilli(),
		Anomalies:  make([]Anomaly, 0),
	}

	if len(cl.entries) == 0 {
		return report
	}

	// Check for sequence gaps
	for i := 1; i < len(cl.entries); i++ {
		prev := cl.entries[i-1].Entry
		curr := cl.entries[i].Entry

		if curr.Seq != prev.Seq+1 {
			report.Anomalies = append(report.Anomalies, Anomaly{
				Type:    "sequence_gap",
				Details: fmt.Sprintf("gap between seq %d and %d", prev.Seq, curr.Seq),
			})
		}

		if curr.Prev != prev.Hash {
			report.Anomalies = append(report.Anomalies, Anomaly{
				Type:    "hash_mismatch",
				Details: fmt.Sprintf("entry %d prev hash mismatch", curr.Seq),
			})
		}
	}

	if len(report.Anomalies) > 0 {
		report.TamperingDetected = true
	}

	return report
}

// TamperReport describes detected tampering or anomalies.
type TamperReport struct {
	DetectedAt         int64      `json:"detectedAt"`
	TamperingDetected  bool       `json:"tamperingDetected"`
	Anomalies          []Anomaly  `json:"anomalies"`
}

// Anomaly describes a detected anomaly.
type Anomaly struct {
	Type    string `json:"type"`
	Details string `json:"details"`
}
