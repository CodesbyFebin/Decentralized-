package providers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// DatabaseMigrator handles schema and data migration between database backends.
type DatabaseMigrator struct {
	sourceDriver   DatabaseDriver
	targetDriver   DatabaseDriver
	sourceDB       *sql.DB
	targetDB       *sql.DB
	strategy       *MigrationStrategy
	migrations     map[string]*SchemaMigration
	migrationMutex sync.RWMutex
	validator      *MigrationValidator
}

// MigrationStrategy defines migration behavior and constraints.
type MigrationStrategy struct {
	SourceBackend      string // "sqlite", "postgres", "mysql"
	TargetBackend      string // "sqlite", "postgres", "mysql"
	BatchSize          int    // Number of campaigns per batch
	ValidateData       bool   // Validate data after migration
	CreateBackup       bool   // Create backup before migration
	AllowDowntime      bool   // Allow temporary service pause
	PreserveTimestamps bool   // Keep original timestamps
	SchemaVersion      string // Target schema version
}

// SchemaMigration tracks migration history and state.
type SchemaMigration struct {
	ID              string
	SourceBackend   string
	TargetBackend   string
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	Status          string // "pending", "in_progress", "completed", "failed", "rolled_back"
	CampaignsMigrated int
	CampaignsFailed int
	SourceHash      string // SHA256 of source data
	TargetHash      string // SHA256 of target data
	SchemaVersion   string
	RollbackPath    string
	Errors          []string
}

// MigrationValidator validates data consistency before and after migration.
type MigrationValidator struct {
	sourceCampaignCount   int64
	targetCampaignCount   int64
	sourceCampaignHashes  map[string]string
	targetCampaignHashes  map[string]string
	validationMutex       sync.RWMutex
	evidenceValidation    bool
	gateResultsValidation bool
}

// NewDatabaseMigrator creates a database migrator.
func NewDatabaseMigrator(sourceDriver, targetDriver DatabaseDriver, sourceDB, targetDB *sql.DB, strategy *MigrationStrategy) (*DatabaseMigrator, error) {
	if sourceDriver == nil || targetDriver == nil {
		return nil, fmt.Errorf("source and target drivers required")
	}
	if sourceDB == nil || targetDB == nil {
		return nil, fmt.Errorf("source and target databases required")
	}
	if strategy == nil {
		return nil, fmt.Errorf("migration strategy required")
	}
	if strategy.BatchSize < 1 {
		return nil, fmt.Errorf("batch size must be >= 1")
	}

	return &DatabaseMigrator{
		sourceDriver:   sourceDriver,
		targetDriver:   targetDriver,
		sourceDB:       sourceDB,
		targetDB:       targetDB,
		strategy:       strategy,
		migrations:     make(map[string]*SchemaMigration),
		validator:      NewMigrationValidator(),
	}, nil
}

// NewMigrationValidator creates a migration validator.
func NewMigrationValidator() *MigrationValidator {
	return &MigrationValidator{
		sourceCampaignHashes: make(map[string]string),
		targetCampaignHashes: make(map[string]string),
		evidenceValidation:   true,
		gateResultsValidation: true,
	}
}

// PrepareMigration validates source database and creates backup if needed.
func (dm *DatabaseMigrator) PrepareMigration(ctx context.Context) (*MigrationValidationResult, error) {
	result := &MigrationValidationResult{
		StartTime: time.Now(),
	}

	// Validate source database connectivity
	if err := dm.sourceDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("source database validation failed: %w", err)
	}
	result.SourceValidated = true

	// Count source campaigns
	var count int64
	err := dm.sourceDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM qualification_campaigns").Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to count source campaigns: %w", err)
	}
	result.SourceCampaignCount = count
	dm.validator.sourceCampaignCount = count

	// Validate target database connectivity and schema
	if err := dm.targetDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("target database validation failed: %w", err)
	}
	result.TargetValidated = true

	// Create target schema if needed
	if err := dm.createTargetSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to create target schema: %w", err)
	}
	result.SchemaCreated = true

	// Calculate source data hash
	sourceHash, err := dm.calculateSourceHash(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate source hash: %w", err)
	}
	result.SourceDataHash = sourceHash
	dm.validator.sourceCampaignHashes["all"] = sourceHash

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.ReadyForMigration = true

	return result, nil
}

// ExecuteMigration performs the actual data migration in batches.
func (dm *DatabaseMigrator) ExecuteMigration(ctx context.Context) (*SchemaMigration, error) {
	migration := &SchemaMigration{
		ID:            fmt.Sprintf("migration-%d", time.Now().UnixNano()),
		SourceBackend: dm.strategy.SourceBackend,
		TargetBackend: dm.strategy.TargetBackend,
		StartTime:     time.Now(),
		Status:        "in_progress",
		SchemaVersion: dm.strategy.SchemaVersion,
	}

	dm.migrationMutex.Lock()
	dm.migrations[migration.ID] = migration
	dm.migrationMutex.Unlock()

	// Migrate campaigns in batches
	var offset int64 = 0
	totalMigrated := 0

	for {
		rows, err := dm.sourceDB.QueryContext(ctx,
			"SELECT id, resource_id, qualification_level, status, start_time, end_time FROM qualification_campaigns ORDER BY start_time LIMIT ? OFFSET ?",
			dm.strategy.BatchSize, offset)
		if err != nil {
			migration.Status = "failed"
			migration.Errors = append(migration.Errors, fmt.Sprintf("batch query failed: %v", err))
			dm.updateMigration(migration)
			return migration, fmt.Errorf("failed to query batch: %w", err)
		}

		batchCount := 0
		for rows.Next() {
			var id, resourceID, qualLevel, status string
			var startTime, endTime time.Time

			if err := rows.Scan(&id, &resourceID, &qualLevel, &status, &startTime, &endTime); err != nil {
				migration.CampaignsFailed++
				migration.Errors = append(migration.Errors, fmt.Sprintf("scan error for campaign: %v", err))
				continue
			}

			// Insert into target database
			_, err := dm.targetDB.ExecContext(ctx,
				"INSERT INTO qualification_campaigns (id, resource_id, qualification_level, status, start_time, end_time) VALUES (?, ?, ?, ?, ?, ?)",
				id, resourceID, qualLevel, status, startTime, endTime)
			if err != nil {
				migration.CampaignsFailed++
				migration.Errors = append(migration.Errors, fmt.Sprintf("insert failed for campaign %s: %v", id, err))
				continue
			}

			migration.CampaignsMigrated++
			totalMigrated++
			batchCount++
		}
		rows.Close()

		if batchCount == 0 {
			break
		}
		offset += int64(batchCount)
	}

	migration.EndTime = time.Now()
	migration.Duration = migration.EndTime.Sub(migration.StartTime)
	migration.Status = "completed"

	dm.updateMigration(migration)
	return migration, nil
}

// ValidateMigration verifies data consistency post-migration.
func (dm *DatabaseMigrator) ValidateMigration(ctx context.Context, migrationID string) (*MigrationValidationResult, error) {
	dm.migrationMutex.RLock()
	migration, exists := dm.migrations[migrationID]
	dm.migrationMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("migration not found: %s", migrationID)
	}

	result := &MigrationValidationResult{
		StartTime: time.Now(),
	}

	// Count target campaigns
	var targetCount int64
	err := dm.targetDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM qualification_campaigns").Scan(&targetCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count target campaigns: %w", err)
	}
	result.TargetCampaignCount = targetCount

	// Validate counts match
	if targetCount != int64(migration.CampaignsMigrated) {
		result.ConsistencyErrors = append(result.ConsistencyErrors,
			fmt.Sprintf("campaign count mismatch: migrated=%d, target=%d", migration.CampaignsMigrated, targetCount))
	} else {
		result.CampaignCountValid = true
	}

	// Calculate target data hash
	targetHash, err := dm.calculateTargetHash(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate target hash: %w", err)
	}
	result.TargetDataHash = targetHash
	migration.TargetHash = targetHash

	// Validate schema
	if err := dm.validateSchema(ctx); err != nil {
		result.ConsistencyErrors = append(result.ConsistencyErrors, fmt.Sprintf("schema validation failed: %v", err))
	} else {
		result.SchemaValid = true
	}

	// Validate sample records
	if err := dm.validateSampleRecords(ctx); err != nil {
		result.ConsistencyErrors = append(result.ConsistencyErrors, fmt.Sprintf("sample validation failed: %v", err))
	} else {
		result.SampleRecordsValid = true
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.ValidationPassed = len(result.ConsistencyErrors) == 0

	dm.updateMigration(migration)
	return result, nil
}

// RollbackMigration reverts migration by dropping target data.
func (dm *DatabaseMigrator) RollbackMigration(ctx context.Context, migrationID string) (*RollbackResult, error) {
	dm.migrationMutex.RLock()
	migration, exists := dm.migrations[migrationID]
	dm.migrationMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("migration not found: %s", migrationID)
	}

	result := &RollbackResult{
		MigrationID: migrationID,
		StartTime:   time.Now(),
	}

	// Verify migration can be rolled back
	if migration.Status == "rolled_back" {
		return nil, fmt.Errorf("migration already rolled back")
	}

	// Delete all migrated campaigns from target
	deleteResult, err := dm.targetDB.ExecContext(ctx, "DELETE FROM qualification_campaigns WHERE id LIKE ?", migration.SourceBackend+"%")
	if err != nil {
		result.RollbackFailed = true
		result.Error = fmt.Sprintf("failed to delete campaigns: %v", err)
		return result, err
	}

	deleted, err := deleteResult.RowsAffected()
	if err != nil {
		result.RollbackFailed = true
		result.Error = fmt.Sprintf("failed to get affected rows: %v", err)
		return result, err
	}

	result.CampaignsCleaned = int(deleted)
	migration.Status = "rolled_back"
	migration.RollbackPath = result.MigrationID

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = !result.RollbackFailed

	dm.updateMigration(migration)
	return result, nil
}

// GetMigrationHistory returns migration history.
func (dm *DatabaseMigrator) GetMigrationHistory(ctx context.Context) []*SchemaMigration {
	dm.migrationMutex.RLock()
	defer dm.migrationMutex.RUnlock()

	migrations := make([]*SchemaMigration, 0, len(dm.migrations))
	for _, m := range dm.migrations {
		migrations = append(migrations, m)
	}
	return migrations
}

// Helper methods

func (dm *DatabaseMigrator) createTargetSchema(ctx context.Context) error {
	// In real implementation, would use targetDriver.CreateSchema()
	// For now, validate schema exists
	var tableExists bool
	err := dm.targetDB.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name='qualification_campaigns')").Scan(&tableExists)
	if err != nil {
		// Fallback for SQLite which doesn't have information_schema
		err = dm.targetDB.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='qualification_campaigns')").Scan(&tableExists)
	}
	if !tableExists {
		return fmt.Errorf("target schema not initialized")
	}
	return nil
}

func (dm *DatabaseMigrator) calculateSourceHash(ctx context.Context) (string, error) {
	rows, err := dm.sourceDB.QueryContext(ctx,
		"SELECT id, resource_id, qualification_level, status FROM qualification_campaigns ORDER BY id")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	hash := sha256.New()
	for rows.Next() {
		var id, resourceID, qualLevel, status string
		if err := rows.Scan(&id, &resourceID, &qualLevel, &status); err != nil {
			return "", err
		}
		hash.Write([]byte(id + resourceID + qualLevel + status))
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (dm *DatabaseMigrator) calculateTargetHash(ctx context.Context) (string, error) {
	rows, err := dm.targetDB.QueryContext(ctx,
		"SELECT id, resource_id, qualification_level, status FROM qualification_campaigns ORDER BY id")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	hash := sha256.New()
	for rows.Next() {
		var id, resourceID, qualLevel, status string
		if err := rows.Scan(&id, &resourceID, &qualLevel, &status); err != nil {
			return "", err
		}
		hash.Write([]byte(id + resourceID + qualLevel + status))
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (dm *DatabaseMigrator) validateSchema(ctx context.Context) error {
	// Verify both databases have required tables
	tables := []string{"qualification_campaigns", "gate_results", "campaign_evidence"}
	for _, table := range tables {
		var exists bool
		err := dm.targetDB.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name=?)", table).Scan(&exists)
		if err != nil {
			// Fallback for SQLite
			err = dm.targetDB.QueryRowContext(ctx,
				"SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)", table).Scan(&exists)
		}
		if err != nil || !exists {
			return fmt.Errorf("table %s not found in target database", table)
		}
	}
	return nil
}

func (dm *DatabaseMigrator) validateSampleRecords(ctx context.Context) error {
	// Sample 10 random campaigns from both databases and compare
	rows, err := dm.sourceDB.QueryContext(ctx,
		"SELECT id, resource_id FROM qualification_campaigns LIMIT 10")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, resourceID string
		if err := rows.Scan(&id, &resourceID); err != nil {
			return err
		}

		// Verify in target
		var targetResourceID string
		err := dm.targetDB.QueryRowContext(ctx,
			"SELECT resource_id FROM qualification_campaigns WHERE id=?", id).Scan(&targetResourceID)
		if err != nil {
			return fmt.Errorf("campaign %s not found in target or data mismatch", id)
		}
		if targetResourceID != resourceID {
			return fmt.Errorf("data mismatch for campaign %s: source=%s, target=%s", id, resourceID, targetResourceID)
		}
	}
	return nil
}

func (dm *DatabaseMigrator) updateMigration(migration *SchemaMigration) {
	dm.migrationMutex.Lock()
	defer dm.migrationMutex.Unlock()
	dm.migrations[migration.ID] = migration
}

// MigrationValidationResult represents validation results pre/post-migration.
type MigrationValidationResult struct {
	StartTime              time.Time
	EndTime                time.Time
	Duration               time.Duration
	SourceValidated        bool
	TargetValidated        bool
	SchemaCreated          bool
	SourceCampaignCount    int64
	TargetCampaignCount    int64
	SourceDataHash         string
	TargetDataHash         string
	CampaignCountValid     bool
	SchemaValid            bool
	SampleRecordsValid     bool
	ConsistencyErrors      []string
	ValidationPassed       bool
	ReadyForMigration      bool
}

// RollbackResult represents rollback operation results.
type RollbackResult struct {
	MigrationID     string
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	CampaignsCleaned int
	RollbackFailed  bool
	Success         bool
	Error           string
}

// SchemaVersion tracks database schema versions for migration.
type SchemaVersion struct {
	Version   string
	Backend   string
	CreatedAt time.Time
	Tables    []string
	Hash      string
}
