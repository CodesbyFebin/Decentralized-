// Package audit retention policy implementation.
package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// RetentionPolicy defines how long audit logs are retained.
type RetentionPolicy struct {
	RetentionDays      int           // Keep logs for this many days (default 90)
	ArchiveAfterDays   int           // Archive logs after this many days (default 30)
	MaxEntriesPerFile  int           // Max entries before rotation (default 100000)
	CompressionEnabled bool          // Enable compression for archived logs
	ImmutableStorage   bool          // Prevent deletion of logs (archival only)
	mu                 sync.RWMutex
}

// NewRetentionPolicy creates a default retention policy.
func NewRetentionPolicy() *RetentionPolicy {
	return &RetentionPolicy{
		RetentionDays:     90,
		ArchiveAfterDays:  30,
		MaxEntriesPerFile: 100000,
		ImmutableStorage:  true,
		CompressionEnabled: false,
	}
}

// RetentionAuditLog tracks deletions and archival.
type RetentionAuditLog struct {
	Timestamp   time.Time `json:"timestamp"`
	Action      string    `json:"action"` // "delete", "archive", "rotate", "purge"
	FilePath    string    `json:"file_path"`
	EntriesCount int64    `json:"entries_count"`
	Reason      string    `json:"reason"`
	ActorID     string    `json:"actor_id"`
}

// LogManager manages log files and retention.
type LogManager struct {
	dir            string
	policy         *RetentionPolicy
	retentionLog   []RetentionAuditLog
	logFilePath    string
	retentionPath  string
	mu             sync.RWMutex
	lastRotation   time.Time
	currentEntries int64
}

// NewLogManager creates a new log manager.
func NewLogManager(dir string, policy *RetentionPolicy) *LogManager {
	if policy == nil {
		policy = NewRetentionPolicy()
	}

	return &LogManager{
		dir:           dir,
		policy:        policy,
		retentionLog:  make([]RetentionAuditLog, 0),
		logFilePath:   filepath.Join(dir, "audit.log"),
		retentionPath: filepath.Join(dir, "audit.retention.log"),
		lastRotation:  time.Now(),
	}
}

// SaveLedger saves a ledger to disk with rotation support.
func (m *LogManager) SaveLedger(ledger *Ledger, sourceNodeID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if rotation is needed
	if m.currentEntries > int64(m.policy.MaxEntriesPerFile) {
		if err := m.rotateLedger(); err != nil {
			return "", fmt.Errorf("failed to rotate ledger: %w", err)
		}
	}

	// Serialize ledger
	data, err := json.MarshalIndent(ledger, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal ledger: %w", err)
	}

	// Write atomically
	tmpPath := m.logFilePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return "", fmt.Errorf("failed to write log file: %w", err)
	}

	if err := os.Rename(tmpPath, m.logFilePath); err != nil {
		return "", fmt.Errorf("failed to finalize log file: %w", err)
	}

	m.currentEntries = int64(len(ledger.Entries))

	return m.logFilePath, nil
}

// LoadLedger loads a ledger from disk.
func (m *LogManager) LoadLedger() (*Ledger, error) {
	m.mu.RLock()
	logPath := m.logFilePath
	m.mu.RUnlock()

	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Ledger{Entries: make([]Entry, 0)}, nil
		}
		return nil, err
	}

	var ledger Ledger
	if err := json.Unmarshal(data, &ledger); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ledger: %w", err)
	}

	return &ledger, nil
}

// rotateLedger rotates the current log file and archives it.
func (m *LogManager) rotateLedger() error {
	// Generate timestamped backup
	timestamp := time.Now().Format("2006-01-02-150405")
	backupPath := fmt.Sprintf("%s.%s", m.logFilePath, timestamp)

	if err := os.Rename(m.logFilePath, backupPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Record rotation
	m.recordRetentionAction("rotate", backupPath, m.currentEntries, "log file rotated due to size limit")

	// Check if archival is needed
	if m.shouldArchive(backupPath) {
		return m.archiveLog(backupPath)
	}

	return nil
}

// shouldArchive checks if a log file should be archived.
func (m *LogManager) shouldArchive(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	age := time.Since(info.ModTime())
	return age > time.Duration(m.policy.ArchiveAfterDays)*24*time.Hour
}

// archiveLog archives a log file.
func (m *LogManager) archiveLog(filePath string) error {
	archivePath := filePath + ".archived"

	if err := os.Rename(filePath, archivePath); err != nil {
		return err
	}

	// Record archival
	m.recordRetentionAction("archive", archivePath, m.currentEntries, "log file archived")

	return nil
}

// EnforceRetention applies the retention policy.
func (m *LogManager) EnforceRetention() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return err
	}

	cutoffTime := time.Now().AddDate(0, 0, -m.policy.RetentionDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check for old log files
		if isLogFile(entry.Name()) {
			fullPath := filepath.Join(m.dir, entry.Name())
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoffTime) {
				if m.policy.ImmutableStorage {
					// Archive instead of delete
					archivePath := fullPath + ".immutable"
					if err := os.Rename(fullPath, archivePath); err == nil {
						m.recordRetentionAction("archive", archivePath, 0, "automatic retention enforcement")
					}
				} else {
					// Delete old log
					if err := os.Remove(fullPath); err == nil {
						m.recordRetentionAction("delete", fullPath, 0, "automatic retention enforcement")
					}
				}
			}
		}
	}

	// Save retention audit log
	return m.saveRetentionLog()
}

// recordRetentionAction records a retention action.
func (m *LogManager) recordRetentionAction(action string, filePath string, count int64, reason string) {
	entry := RetentionAuditLog{
		Timestamp:    time.Now(),
		Action:       action,
		FilePath:     filePath,
		EntriesCount: count,
		Reason:       reason,
	}

	m.retentionLog = append(m.retentionLog, entry)
}

// saveRetentionLog persists the retention audit log.
func (m *LogManager) saveRetentionLog() error {
	data, err := json.MarshalIndent(m.retentionLog, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.retentionPath, data, 0o600)
}

// LoadRetentionLog loads the retention audit log.
func (m *LogManager) LoadRetentionLog() ([]RetentionAuditLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := os.ReadFile(m.retentionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []RetentionAuditLog{}, nil
		}
		return nil, err
	}

	var logs []RetentionAuditLog
	if err := json.Unmarshal(data, &logs); err != nil {
		return nil, err
	}

	return logs, nil
}

// VerifyIntegrity verifies ledger integrity without loading everything.
func (m *LogManager) VerifyIntegrity() error {
	ledger, err := m.LoadLedger()
	if err != nil {
		return fmt.Errorf("failed to load ledger: %w", err)
	}

	if err := VerifyIntegrity(ledger.Entries); err != nil {
		return err
	}

	return nil
}

// VerifyIntegrity checks a ledger for integrity.
func VerifyIntegrity(entries []Entry) error {
	if brk := Verify(entries, 0, Genesis); brk != nil {
		return brk
	}
	return nil
}

// ListLogFiles returns all log files (newest first).
func (m *LogManager) ListLogFiles() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, err
	}

	var files []struct {
		name  string
		time  time.Time
		path  string
	}

	for _, entry := range entries {
		if !entry.IsDir() && isLogFile(entry.Name()) {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			files = append(files, struct {
				name  string
				time  time.Time
				path  string
			}{entry.Name(), info.ModTime(), filepath.Join(m.dir, entry.Name())})
		}
	}

	// Sort by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].time.After(files[j].time)
	})

	result := make([]string, len(files))
	for i, f := range files {
		result[i] = f.path
	}

	return result, nil
}

// GetRetentionStats returns retention policy statistics.
func (m *LogManager) GetRetentionStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalRetentionDays := m.policy.RetentionDays
	archiveAfterDays := m.policy.ArchiveAfterDays

	return map[string]interface{}{
		"retention_days":       totalRetentionDays,
		"archive_after_days":   archiveAfterDays,
		"max_entries_per_file": m.policy.MaxEntriesPerFile,
		"immutable_storage":    m.policy.ImmutableStorage,
		"compression_enabled":  m.policy.CompressionEnabled,
		"current_entries":      m.currentEntries,
		"retention_log_entries": len(m.retentionLog),
	}
}

// Helper function to identify log files
func isLogFile(name string) bool {
	return (name == "audit.log" ||
		(len(name) > len("audit.log.") && name[:len("audit.log.")] == "audit.log.")) &&
		!isRetentionLog(name)
}

func isRetentionLog(name string) bool {
	return name == "audit.retention.log"
}

// CheckpointWithRetention creates a checkpoint and records it with retention info.
type CheckpointWithRetention struct {
	Checkpoint *Checkpoint              `json:"checkpoint"`
	Recorded   time.Time                `json:"recorded_at"`
	RetentionInfo map[string]interface{} `json:"retention_info"`
}
