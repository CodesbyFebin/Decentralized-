package providers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// EvolutionVersion represents a database schema version with metadata and change tracking
type EvolutionVersion struct {
	Version          string                    // e.g., "1.0", "1.1", "2.0"
	Backend          string                    // "sqlite", "postgres", "mysql"
	CreatedAt        time.Time                 // When version was created
	AppliedAt        time.Time                 // When applied to database
	Description      string                    // Human-readable description
	Tables           []string                  // List of table names
	SchemaHash       string                    // SHA256 of schema definition
	MigrationID      string                    // Reference to migration that created this
	ParentVersion    string                    // Previous version (for rollback chain)
	BreakingChanges  []string                  // List of breaking changes
	DataMigration    bool                      // Whether data migration required
	RollbackPath     string                    // Path to rollback migration
	VerifiedAt       time.Time                 // Last verification timestamp
	CompatibleWith   []string                  // Versions this is compatible with
}

// EvolutionStep defines a single schema change operation
type EvolutionStep struct {
	StepID         string                    // Unique step identifier
	StepNumber     int                       // Sequence within migration
	Type           string                    // "add_column", "drop_column", "modify_column", "create_table", "drop_table", "add_index", "drop_index", "add_constraint", "drop_constraint"
	TargetTable    string                    // Table affected (if applicable)
	TargetColumn   string                    // Column affected (if applicable)
	Definition     string                    // SQL or structured definition
	DataMigration  string                    // SQL for data transformation (if needed)
	Rollback       string                    // SQL to revert this step
	PreCondition   func(ctx context.Context) bool // Validation before step
	PostCondition  func(ctx context.Context) bool // Validation after step
	Description    string                    // Human-readable description
	ExecutedAt     time.Time                 // When executed
	Duration       time.Duration             // Execution time
	Status         string                    // "pending", "in_progress", "completed", "failed", "rolled_back"
	Error          string                    // Error message if failed
}

// EvolutionMigration represents a complete versioned schema migration
type EvolutionMigration struct {
	ID              string                    // Unique migration identifier
	FromVersion     string                    // Starting schema version
	ToVersion       string                    // Target schema version
	Backend         string                    // Database backend
	Steps           []EvolutionStep     // Ordered list of changes
	Description     string                    // Purpose of migration
	CreatedAt       time.Time                 // When plan created
	ApprovedAt      time.Time                 // When approved for execution
	ApprovedBy      string                    // User who approved
	BreakingChanges []string                  // List of breaking changes
	DataImpact      string                    // "none", "read_only", "write_delay", "downtime_required"
	EstimatedTime   time.Duration             // Expected execution time
	RollbackPlan    *EvolutionMigration      // Reverse migration plan
	CompatibilityMatrix map[string]bool       // Compatibility with versions
}

// EvolutionResult tracks execution outcome
type EvolutionResult struct {
	ID              string                    // Migration ID
	Version         string                    // Target version
	Backend         string                    // Database backend
	StartTime       time.Time                 // Execution start
	EndTime         time.Time                 // Execution end
	Duration        time.Duration             // Total time
	Status          string                    // "completed", "failed", "rolled_back", "partially_completed"
	StepsCompleted  int                       // Number of steps that succeeded
	StepsFailed     int                       // Number of steps that failed
	StepDetails     []EvolutionStep     // Detailed step results
	Errors          []string                  // Error messages
	SchemaHash      string                    // SHA256 of resulting schema
	RowsAffected    int64                     // Total rows modified
	DataValidated   bool                      // Whether data consistency verified
	RolledBack      bool                      // Whether rollback was performed
	RollbackReason  string                    // Why rollback occurred
}

// SchemaChange represents a single tracked schema modification
type SchemaChange struct {
	ID              string                    // Unique change ID
	Version         string                    // Version that introduced change
	ChangeType      string                    // "add", "remove", "modify"
	ObjectType      string                    // "table", "column", "index", "constraint"
	ObjectName      string                    // Name of affected object
	ParentTable     string                    // Table (if column/index/constraint)
	OldDefinition   string                    // Previous definition
	NewDefinition   string                    // New definition
	BreakingFor     []string                  // Versions/apps this breaks
	Timestamp       time.Time                 // When change was recorded
	ChangeHash      string                    // SHA256 of change record
}

// SchemaValidator handles schema compatibility and validation
type SchemaValidator struct {
	schemas         map[string]*EvolutionVersion // Known versions
	changes         []SchemaChange            // Change history
	compatibility   map[string][]string       // Compatibility matrix
	validatorMutex  sync.RWMutex
	lastValidation  time.Time
	validationCache map[string]bool
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		schemas:         make(map[string]*EvolutionVersion),
		changes:         make([]SchemaChange, 0),
		compatibility:   make(map[string][]string),
		validationCache: make(map[string]bool),
	}
}

// RegisterVersion registers a known schema version
func (sv *SchemaValidator) RegisterVersion(version *EvolutionVersion) error {
	sv.validatorMutex.Lock()
	defer sv.validatorMutex.Unlock()

	if _, exists := sv.schemas[version.Version]; exists {
		return fmt.Errorf("version %s already registered", version.Version)
	}

	// Calculate schema hash
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", version.Tables)))
	version.SchemaHash = fmt.Sprintf("%x", h.Sum(nil))

	sv.schemas[version.Version] = version
	return nil
}

// ValidateCompatibility checks if two versions are compatible
func (sv *SchemaValidator) ValidateCompatibility(fromVersion, toVersion string) (bool, error) {
	sv.validatorMutex.RLock()
	defer sv.validatorMutex.RUnlock()

	_, ok := sv.schemas[fromVersion]
	if !ok {
		return false, fmt.Errorf("unknown version: %s", fromVersion)
	}

	to, ok := sv.schemas[toVersion]
	if !ok {
		return false, fmt.Errorf("unknown version: %s", toVersion)
	}

	// Check compatibility matrix
	if compat, ok := sv.compatibility[fromVersion]; ok {
		for _, v := range compat {
			if v == toVersion {
				return true, nil
			}
		}
	}

	// If no breaking changes in target, versions are compatible
	if len(to.BreakingChanges) == 0 {
		return true, nil
	}

	return false, fmt.Errorf("breaking changes from %s to %s", fromVersion, toVersion)
}

// RecordChange records a schema change for history tracking
func (sv *SchemaValidator) RecordChange(change SchemaChange) error {
	sv.validatorMutex.Lock()
	defer sv.validatorMutex.Unlock()

	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s:%s:%s:%s", change.ChangeType, change.ObjectType, change.ObjectName, change.NewDefinition)))
	change.ChangeHash = fmt.Sprintf("%x", h.Sum(nil))

	sv.changes = append(sv.changes, change)
	return nil
}

// GetChangeHistory returns changes for a given version
func (sv *SchemaValidator) GetChangeHistory(version string) []SchemaChange {
	sv.validatorMutex.RLock()
	defer sv.validatorMutex.RUnlock()

	var result []SchemaChange
	for _, change := range sv.changes {
		if change.Version == version {
			result = append(result, change)
		}
	}
	return result
}

// SchemaEvolver orchestrates schema migrations and version management
type SchemaEvolver struct {
	driver          DatabaseDriver           // Database driver
	db              *sql.DB                  // Database connection
	validator       *SchemaValidator         // Schema validator
	migrations      map[string]*EvolutionMigration // Registered migrations
	results         []EvolutionResult  // Migration execution history
	currentVersion  string                   // Current schema version
	evolverMutex    sync.RWMutex
	isApplying      bool                     // Migration in progress
}

// NewSchemaEvolver creates a new schema evolution manager
func NewSchemaEvolver(driver DatabaseDriver, db *sql.DB, validator *SchemaValidator) *SchemaEvolver {
	return &SchemaEvolver{
		driver:     driver,
		db:         db,
		validator:  validator,
		migrations: make(map[string]*EvolutionMigration),
		results:    make([]EvolutionResult, 0),
	}
}

// RegisterMigration registers a named migration plan
func (se *SchemaEvolver) RegisterMigration(plan *EvolutionMigration) error {
	se.evolverMutex.Lock()
	defer se.evolverMutex.Unlock()

	if _, exists := se.migrations[plan.ID]; exists {
		return fmt.Errorf("migration %s already registered", plan.ID)
	}

	se.migrations[plan.ID] = plan
	return nil
}

// GetCurrentVersion returns the current schema version
func (se *SchemaEvolver) GetCurrentVersion(ctx context.Context) (string, error) {
	se.evolverMutex.RLock()
	defer se.evolverMutex.RUnlock()

	if se.currentVersion != "" {
		return se.currentVersion, nil
	}

	// Query schema_versions table if it exists
	row := se.db.QueryRowContext(ctx, "SELECT version FROM schema_versions ORDER BY applied_at DESC LIMIT 1")
	var version string
	err := row.Scan(&version)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	se.currentVersion = version
	return version, nil
}

// PrepareMigration validates and prepares a migration for execution
func (se *SchemaEvolver) PrepareMigration(ctx context.Context, migrationID string) (*EvolutionMigration, error) {
	se.evolverMutex.RLock()
	plan, ok := se.migrations[migrationID]
	se.evolverMutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("migration not found: %s", migrationID)
	}

	// Validate current version
	current, err := se.GetCurrentVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot determine current version: %v", err)
	}

	if current != plan.FromVersion {
		return nil, fmt.Errorf("version mismatch: current=%s, expected=%s", current, plan.FromVersion)
	}

	// Validate compatibility
	compatible, err := se.validator.ValidateCompatibility(current, plan.ToVersion)
	if err != nil {
		return nil, fmt.Errorf("compatibility check failed: %v", err)
	}

	if !compatible {
		return nil, fmt.Errorf("migration would introduce breaking changes")
	}

	// Pre-condition checks
	for i, step := range plan.Steps {
		if step.PreCondition != nil {
			if !step.PreCondition(ctx) {
				return nil, fmt.Errorf("step %d pre-condition failed: %s", i, step.Description)
			}
		}
	}

	return plan, nil
}

// ApplyMigration executes a migration plan with step-by-step execution and error handling
func (se *SchemaEvolver) ApplyMigration(ctx context.Context, migrationID string) (*EvolutionResult, error) {
	se.evolverMutex.Lock()
	if se.isApplying {
		se.evolverMutex.Unlock()
		return nil, fmt.Errorf("migration already in progress")
	}
	se.isApplying = true
	se.evolverMutex.Unlock()
	defer func() {
		se.evolverMutex.Lock()
		se.isApplying = false
		se.evolverMutex.Unlock()
	}()

	plan, err := se.PrepareMigration(ctx, migrationID)
	if err != nil {
		return nil, err
	}

	result := &EvolutionResult{
		ID:          migrationID,
		Version:     plan.ToVersion,
		Backend:     plan.Backend,
		StartTime:   time.Now(),
		Status:      "in_progress",
		StepDetails: make([]EvolutionStep, 0),
		Errors:      make([]string, 0),
	}

	tx, err := se.db.BeginTx(ctx, nil)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, err.Error())
		return result, err
	}

	// Execute each step
	for i, step := range plan.Steps {
		step.StepNumber = i + 1
		step.Status = "in_progress"
		step.ExecutedAt = time.Now()

		// Execute step SQL
		if step.Definition != "" {
			_, err := tx.ExecContext(ctx, step.Definition)
			if err != nil {
				step.Status = "failed"
				step.Error = err.Error()
				result.StepDetails = append(result.StepDetails, step)
				result.StepsFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("step %d failed: %v", i+1, err))

				// Rollback on error
				tx.Rollback()
				result.Status = "failed"
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				return result, err
			}
		}

		// Execute data migration if needed
		if step.DataMigration != "" {
			_, err := tx.ExecContext(ctx, step.DataMigration)
			if err != nil {
				step.Status = "failed"
				step.Error = err.Error()
				result.StepDetails = append(result.StepDetails, step)
				result.StepsFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("step %d data migration failed: %v", i+1, err))

				tx.Rollback()
				result.Status = "failed"
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				return result, err
			}
		}

		// Verify post-condition
		if step.PostCondition != nil {
			if !step.PostCondition(ctx) {
				step.Status = "failed"
				step.Error = "post-condition check failed"
				result.StepDetails = append(result.StepDetails, step)
				result.StepsFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("step %d post-condition failed", i+1))

				tx.Rollback()
				result.Status = "failed"
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				return result, fmt.Errorf("step %d post-condition failed", i+1)
			}
		}

		step.Status = "completed"
		step.Duration = time.Since(step.ExecutedAt)
		result.StepDetails = append(result.StepDetails, step)
		result.StepsCompleted++
	}

	// Record version in schema_versions table
	versionSQL := fmt.Sprintf(
		"INSERT INTO schema_versions (version, backend, created_at, applied_at, description) VALUES ('%s', '%s', '%s', '%s', '%s')",
		plan.ToVersion, plan.Backend, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339), plan.Description,
	)
	_, err = tx.ExecContext(ctx, versionSQL)
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("failed to record version: %v", err))
		tx.Rollback()
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, fmt.Sprintf("transaction commit failed: %v", err))
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	result.Status = "completed"
	result.DataValidated = true
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Store result
	se.evolverMutex.Lock()
	se.results = append(se.results, *result)
	se.currentVersion = plan.ToVersion
	se.evolverMutex.Unlock()

	return result, nil
}

// RollbackMigration reverts a migration to previous version
func (se *SchemaEvolver) RollbackMigration(ctx context.Context, migrationID string) (*EvolutionResult, error) {
	se.evolverMutex.RLock()
	plan, ok := se.migrations[migrationID]
	se.evolverMutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("migration not found: %s", migrationID)
	}

	if plan.RollbackPlan == nil {
		return nil, fmt.Errorf("no rollback plan available for migration %s", migrationID)
	}

	// Execute rollback plan
	result, err := se.ApplyMigration(ctx, plan.RollbackPlan.ID)
	if err != nil {
		return nil, err
	}

	result.RolledBack = true
	result.RollbackReason = "manual rollback"

	return result, nil
}

// GetMigrationHistory returns all migration results
func (se *SchemaEvolver) GetMigrationHistory(ctx context.Context) []EvolutionResult {
	se.evolverMutex.RLock()
	defer se.evolverMutex.RUnlock()

	results := make([]EvolutionResult, len(se.results))
	copy(results, se.results)
	return results
}

// ValidateSchema checks schema integrity after migration
func (se *SchemaEvolver) ValidateSchema(ctx context.Context, version string) (bool, error) {
	se.evolverMutex.RLock()
	defer se.evolverMutex.RUnlock()

	// Query current schema version from database
	row := se.db.QueryRowContext(ctx, "SELECT version FROM schema_versions WHERE version = $1 ORDER BY applied_at DESC LIMIT 1", version)
	var dbVersion string
	err := row.Scan(&dbVersion)
	if err != nil {
		return false, fmt.Errorf("version not found in database: %s", version)
	}

	return dbVersion == version, nil
}

// CreateMigrationPlan creates a new migration plan from version to version
func CreateMigrationPlan(fromVersion, toVersion, backend, description string) *EvolutionMigration {
	return &EvolutionMigration{
		ID:              fmt.Sprintf("migration-%d", time.Now().UnixNano()),
		FromVersion:     fromVersion,
		ToVersion:       toVersion,
		Backend:         backend,
		Description:     description,
		CreatedAt:       time.Now(),
		Steps:           make([]EvolutionStep, 0),
		BreakingChanges: make([]string, 0),
		DataImpact:      "none",
		CompatibilityMatrix: make(map[string]bool),
	}
}

// AddStep adds a migration step to a plan
func (plan *EvolutionMigration) AddStep(step EvolutionStep) {
	step.StepID = fmt.Sprintf("%s-step-%d", plan.ID, len(plan.Steps)+1)
	plan.Steps = append(plan.Steps, step)
}

// SetRollbackPlan sets the reverse migration for this plan
func (plan *EvolutionMigration) SetRollbackPlan(rollbackPlan *EvolutionMigration) {
	plan.RollbackPlan = rollbackPlan
}

// GetVersionReport generates a report of version changes
func (se *SchemaEvolver) GetVersionReport(ctx context.Context) map[string]interface{} {
	se.evolverMutex.RLock()
	defer se.evolverMutex.RUnlock()

	current, _ := se.GetCurrentVersion(ctx)

	totalMigrations := len(se.results)
	successfulMigrations := 0
	failedMigrations := 0

	for _, result := range se.results {
		if result.Status == "completed" {
			successfulMigrations++
		} else if result.Status == "failed" {
			failedMigrations++
		}
	}

	return map[string]interface{}{
		"current_version":        current,
		"total_migrations":       totalMigrations,
		"successful_migrations":  successfulMigrations,
		"failed_migrations":      failedMigrations,
		"registered_migrations":  len(se.migrations),
		"last_migration_at":      func() interface{} {
			if len(se.results) > 0 {
				return se.results[len(se.results)-1].EndTime
			}
			return nil
		}(),
	}
}
