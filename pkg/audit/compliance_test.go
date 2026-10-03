package audit

import (
	"os"
	"testing"
	"time"
)

func TestComplianceLog_Log(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	entry := Entry{
		Actor:   "dh1test123456789abcdefgh",
		Action:  "create",
		Resource: "app:test",
		Source:  SourceHost,
	}

	if err := cl.Log(entry); err != nil {
		t.Fatalf("log failed: %v", err)
	}

	if cl.Count() != 1 {
		t.Fatalf("expected 1 entry, got %d", cl.Count())
	}
}

func TestComplianceLog_QueryEntries(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	entries := []Entry{
		{Actor: "actor1", Action: "create", Resource: "app:test1", Source: SourceHost},
		{Actor: "actor2", Action: "delete", Resource: "app:test2", Source: SourceHost},
		{Actor: "actor1", Action: "update", Resource: "app:test1", Source: SourceHost},
	}

	for _, e := range entries {
		if err := cl.Log(e); err != nil {
			t.Fatalf("log failed: %v", err)
		}
	}

	// Query by actor
	criteria := &QueryCriteria{Actor: "actor1"}
	results, err := cl.QueryEntries(criteria)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 entries for actor1, got %d", len(results))
	}

	// Query by action
	criteria = &QueryCriteria{Action: "delete"}
	results, err = cl.QueryEntries(criteria)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 entry for delete action, got %d", len(results))
	}
}

func TestComplianceLog_QueryByResource(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	entries := []Entry{
		{Actor: "actor1", Action: "create", Resource: "application:app1", Source: SourceHost},
		{Actor: "actor1", Action: "create", Resource: "application:app2", Source: SourceHost},
		{Actor: "actor1", Action: "create", Resource: "node:node1", Source: SourceHost},
	}

	for _, e := range entries {
		if err := cl.Log(e); err != nil {
			t.Fatalf("log failed: %v", err)
		}
	}

	// Query by resource prefix
	criteria := &QueryCriteria{Resource: "application:"}
	results, err := cl.QueryEntries(criteria)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 application entries, got %d", len(results))
	}
}

func TestComplianceLog_QueryByTimeRange(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	// Get the time range after logging starts to ensure accurate timestamps
	startTime := time.Now().UnixMilli()

	entries := []Entry{
		{Actor: "actor1", Action: "create", Resource: "app:test1", Source: SourceHost},
		{Actor: "actor1", Action: "create", Resource: "app:test2", Source: SourceHost},
		{Actor: "actor1", Action: "create", Resource: "app:test3", Source: SourceHost},
	}

	for _, e := range entries {
		if err := cl.Log(e); err != nil {
			t.Fatalf("log failed: %v", err)
		}
	}

	endTime := time.Now().UnixMilli()

	// Query within time range - all entries should be within the range
	criteria := &QueryCriteria{StartTime: startTime, EndTime: endTime}
	results, err := cl.QueryEntries(criteria)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) < 1 {
		t.Fatalf("expected at least 1 entry in time range, got %d", len(results))
	}

	// Test with a range that excludes entries
	criteria = &QueryCriteria{StartTime: endTime + 1000}
	results, err = cl.QueryEntries(criteria)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 entries after current time, got %d", len(results))
	}
}

func TestComplianceLog_VerifyChain(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	for i := 0; i < 10; i++ {
		entry := Entry{
			Actor:    "actor1",
			Action:   "create",
			Resource: "app:test",
			Source:   SourceHost,
		}
		if err := cl.Log(entry); err != nil {
			t.Fatalf("log failed: %v", err)
		}
	}

	if err := cl.VerifyChain(); err != nil {
		t.Fatalf("chain verification failed: %v", err)
	}
}

func TestComplianceLog_GetEntry(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	entries := []Entry{
		{Actor: "actor1", Action: "create", Resource: "app:test1", Source: SourceHost},
		{Actor: "actor1", Action: "create", Resource: "app:test2", Source: SourceHost},
		{Actor: "actor1", Action: "create", Resource: "app:test3", Source: SourceHost},
	}

	for _, e := range entries {
		if err := cl.Log(e); err != nil {
			t.Fatalf("log failed: %v", err)
		}
	}

	// Get specific entry
	entry, err := cl.GetEntry(2)
	if err != nil {
		t.Fatalf("get entry failed: %v", err)
	}

	if entry.Entry.Seq != 2 {
		t.Fatalf("expected seq 2, got %d", entry.Entry.Seq)
	}
}

func TestComplianceLog_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	entries := []Entry{
		{Actor: "actor1", Action: "create", Resource: "app:test1", Source: SourceHost},
		{Actor: "actor1", Action: "create", Resource: "app:test2", Source: SourceHost},
	}

	for _, e := range entries {
		if err := cl.Log(e); err != nil {
			t.Fatalf("log failed: %v", err)
		}
	}

	if err := cl.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Load in new instance
	cl2 := NewComplianceLog(tmpDir, nil)
	if err := cl2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cl2.Count() != 2 {
		t.Fatalf("expected 2 entries after load, got %d", cl2.Count())
	}
}

func TestComplianceLog_GetChainHead(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	for i := 0; i < 5; i++ {
		entry := Entry{
			Actor:    "actor1",
			Action:   "create",
			Resource: "app:test",
			Source:   SourceHost,
		}
		if err := cl.Log(entry); err != nil {
			t.Fatalf("log failed: %v", err)
		}
	}

	seq, hash := cl.GetChainHead()
	if seq != 5 {
		t.Fatalf("expected seq 5, got %d", seq)
	}

	if hash == "" {
		t.Fatal("hash should not be empty")
	}
}

func TestComplianceLog_Summary(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	entries := []Entry{
		{Actor: "actor1", Action: "create", Resource: "app:test1", Source: SourceHost},
		{Actor: "actor2", Action: "delete", Resource: "app:test2", Source: SourceHost},
		{Actor: "actor1", Action: "update", Resource: "app:test3", Source: SourceHost},
	}

	for _, e := range entries {
		if err := cl.Log(e); err != nil {
			t.Fatalf("log failed: %v", err)
		}
	}

	summary := cl.GetSummary()

	if summary.TotalEntries != 3 {
		t.Fatalf("expected 3 entries in summary, got %d", summary.TotalEntries)
	}

	if summary.ActionCounts["create"] != 1 {
		t.Fatalf("expected 1 create action, got %d", summary.ActionCounts["create"])
	}

	if summary.ActorCounts["actor1"] != 2 {
		t.Fatalf("expected 2 entries for actor1, got %d", summary.ActorCounts["actor1"])
	}
}

func TestComplianceLog_AnnotateEntry(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	entry := Entry{
		Actor:    "actor1",
		Action:   "create",
		Resource: "app:test",
		Source:   SourceHost,
	}

	if err := cl.Log(entry); err != nil {
		t.Fatalf("log failed: %v", err)
	}

	annotations := map[string]string{
		"severity": "high",
		"category": "security",
	}

	if err := cl.AnnotateEntry(1, annotations); err != nil {
		t.Fatalf("annotate failed: %v", err)
	}

	retrieved, err := cl.GetEntry(1)
	if err != nil {
		t.Fatalf("get entry failed: %v", err)
	}

	if retrieved.Annotations["severity"] != "high" {
		t.Fatalf("annotation not found or incorrect")
	}
}

func TestComplianceLog_DetectTampering(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	for i := 0; i < 5; i++ {
		entry := Entry{
			Actor:    "actor1",
			Action:   "create",
			Resource: "app:test",
			Source:   SourceHost,
		}
		if err := cl.Log(entry); err != nil {
			t.Fatalf("log failed: %v", err)
		}
	}

	report := cl.DetectTampering()

	if report.TamperingDetected {
		t.Fatal("no tampering should be detected in clean log")
	}

	if len(report.Anomalies) > 0 {
		t.Fatalf("expected no anomalies, got %d", len(report.Anomalies))
	}
}

func TestComplianceLog_ListEntries(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	for i := 0; i < 10; i++ {
		entry := Entry{
			Actor:    "actor1",
			Action:   "create",
			Resource: "app:test",
			Source:   SourceHost,
		}
		if err := cl.Log(entry); err != nil {
			t.Fatalf("log failed: %v", err)
		}
	}

	// List last 5 entries
	entries := cl.ListEntries(5)
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(entries))
	}

	// Entries should be in order
	if entries[0].Entry.Seq != 6 {
		t.Fatalf("expected seq 6 for first entry, got %d", entries[0].Entry.Seq)
	}
}

func TestComplianceLog_Export(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	entry := Entry{
		Actor:    "actor1",
		Action:   "create",
		Resource: "app:test",
		Source:   SourceHost,
	}

	if err := cl.Log(entry); err != nil {
		t.Fatalf("log failed: %v", err)
	}

	data, err := cl.Export()
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("exported data should not be empty")
	}
}

func TestComplianceLog_EnforceRetention(t *testing.T) {
	tmpDir := t.TempDir()
	policy := NewRetentionPolicy()
	policy.RetentionDays = 0 // Immediately eligible for archival

	cl := NewComplianceLog(tmpDir, policy)

	entry := Entry{
		Actor:    "actor1",
		Action:   "create",
		Resource: "app:test",
		Source:   SourceHost,
	}

	if err := cl.Log(entry); err != nil {
		t.Fatalf("log failed: %v", err)
	}

	// Wait a bit to ensure file is written
	time.Sleep(100 * time.Millisecond)

	if err := cl.EnforceRetention(); err != nil {
		t.Fatalf("enforce retention failed: %v", err)
	}

	// Verify log files exist in directory
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("read dir failed: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected log files after retention enforcement")
	}
}

func TestComplianceLog_MultipleQueries(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	// Log entries with various combinations
	for _, actor := range []string{"actor1", "actor2"} {
		for _, action := range []string{"create", "update", "delete"} {
			entry := Entry{
				Actor:    actor,
				Action:   action,
				Resource: "app:test",
				Source:   SourceHost,
			}
			if err := cl.Log(entry); err != nil {
				t.Fatalf("log failed: %v", err)
			}
		}
	}

	// Query combining multiple criteria
	criteria := &QueryCriteria{
		Actor:  "actor1",
		Action: "create",
	}

	results, err := cl.QueryEntries(criteria)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Entry.Actor != "actor1" || results[0].Entry.Action != "create" {
		t.Fatal("query result does not match criteria")
	}
}

func TestQueryCriteria_MatchesPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	cl := NewComplianceLog(tmpDir, nil)

	resources := []string{
		"application:app1",
		"application:app2",
		"node:node1",
		"volume:vol1",
	}

	for _, res := range resources {
		entry := Entry{
			Actor:    "actor1",
			Action:   "create",
			Resource: res,
			Source:   SourceHost,
		}
		if err := cl.Log(entry); err != nil {
			t.Fatalf("log failed: %v", err)
		}
	}

	// Test wildcard matching
	criteria := &QueryCriteria{Resource: "*"}
	results, err := cl.QueryEntries(criteria)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("wildcard should match all, got %d", len(results))
	}

	// Test prefix matching
	criteria = &QueryCriteria{Resource: "application:"}
	results, err = cl.QueryEntries(criteria)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 application entries, got %d", len(results))
	}
}
