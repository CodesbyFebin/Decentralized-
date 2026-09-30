package providers

import (
	"context"
	"database/sql"
	"fmt"
)

// DatabaseDriver defines database backend implementations.
type DatabaseDriver interface {
	Name() string
	Connect(ctx context.Context, connString string) (*sql.DB, error)
	CreateTablesSQL() []string
	Close() error
}

// DatabaseConfig holds configuration for database connections.
type DatabaseConfig struct {
	Driver     string
	ConnString string
	MaxOpen    int
	MaxIdle    int
}

// NewDatabaseConnection creates a database connection based on config.
func NewDatabaseConnection(ctx context.Context, cfg *DatabaseConfig) (*sql.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database config required")
	}

	var driver DatabaseDriver

	switch cfg.Driver {
	case "sqlite":
		driver = &SQLiteDriver{}
	case "postgres":
		driver = &PostgresDriver{}
	case "mysql":
		driver = &MySQLDriver{}
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	db, err := driver.Connect(ctx, cfg.ConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", driver.Name(), err)
	}

	if cfg.MaxOpen > 0 {
		db.SetMaxOpenConns(cfg.MaxOpen)
	}
	if cfg.MaxIdle > 0 {
		db.SetMaxIdleConns(cfg.MaxIdle)
	}

	return db, nil
}

// SQLiteDriver implements SQLite backend.
type SQLiteDriver struct{}

func (d *SQLiteDriver) Name() string {
	return "SQLite"
}

func (d *SQLiteDriver) Connect(ctx context.Context, connString string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", connString)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func (d *SQLiteDriver) CreateTablesSQL() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS qualification_campaigns (
			id TEXT PRIMARY KEY,
			resource_id TEXT NOT NULL,
			start_time INTEGER NOT NULL,
			end_time INTEGER,
			source_sha TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'RUNNING',
			qualification_level TEXT,
			evidence_signature TEXT,
			signer_id TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_campaigns_resource ON qualification_campaigns(resource_id)`,
		`CREATE INDEX IF NOT EXISTS idx_campaigns_status ON qualification_campaigns(status)`,
		`CREATE INDEX IF NOT EXISTS idx_campaigns_level ON qualification_campaigns(qualification_level)`,
		`CREATE INDEX IF NOT EXISTS idx_campaigns_start_time ON qualification_campaigns(start_time DESC)`,
		`CREATE TABLE IF NOT EXISTS gate_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			campaign_id TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			passed INTEGER NOT NULL,
			evidence TEXT,
			timestamp INTEGER NOT NULL,
			FOREIGN KEY(campaign_id) REFERENCES qualification_campaigns(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS backend_specific_gates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			campaign_id TEXT NOT NULL,
			backend_profile TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			passed INTEGER NOT NULL,
			evidence TEXT,
			timestamp INTEGER NOT NULL,
			FOREIGN KEY(campaign_id) REFERENCES qualification_campaigns(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS campaign_evidence (
			campaign_id TEXT PRIMARY KEY,
			content_hash TEXT NOT NULL,
			state_snapshot TEXT,
			signature TEXT NOT NULL,
			signer_pubkey TEXT,
			signer_id TEXT,
			os_boundary TEXT,
			filesystem_boundary TEXT,
			physical_boundary TEXT,
			operator_boundary TEXT,
			p1_core_passed INTEGER,
			p1_qemu_passed INTEGER,
			p1_k8s_passed INTEGER,
			p2_multi_passed INTEGER,
			FOREIGN KEY(campaign_id) REFERENCES qualification_campaigns(id) ON DELETE CASCADE
		)`,
	}
}

func (d *SQLiteDriver) Close() error {
	return nil
}

// PostgresDriver implements PostgreSQL backend.
type PostgresDriver struct{}

func (d *PostgresDriver) Name() string {
	return "PostgreSQL"
}

func (d *PostgresDriver) Connect(ctx context.Context, connString string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func (d *PostgresDriver) CreateTablesSQL() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS qualification_campaigns (
			id TEXT PRIMARY KEY,
			resource_id TEXT NOT NULL,
			start_time BIGINT NOT NULL,
			end_time BIGINT,
			source_sha TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'RUNNING',
			qualification_level TEXT,
			evidence_signature TEXT,
			signer_id TEXT,
			created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_campaigns_resource ON qualification_campaigns(resource_id)`,
		`CREATE INDEX IF NOT EXISTS idx_campaigns_status ON qualification_campaigns(status)`,
		`CREATE INDEX IF NOT EXISTS idx_campaigns_level ON qualification_campaigns(qualification_level)`,
		`CREATE INDEX IF NOT EXISTS idx_campaigns_start_time ON qualification_campaigns(start_time DESC)`,
		`CREATE TABLE IF NOT EXISTS gate_results (
			id SERIAL PRIMARY KEY,
			campaign_id TEXT NOT NULL REFERENCES qualification_campaigns(id) ON DELETE CASCADE,
			sequence INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			passed INTEGER NOT NULL,
			evidence TEXT,
			timestamp BIGINT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS backend_specific_gates (
			id SERIAL PRIMARY KEY,
			campaign_id TEXT NOT NULL REFERENCES qualification_campaigns(id) ON DELETE CASCADE,
			backend_profile TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			passed INTEGER NOT NULL,
			evidence TEXT,
			timestamp BIGINT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS campaign_evidence (
			campaign_id TEXT PRIMARY KEY REFERENCES qualification_campaigns(id) ON DELETE CASCADE,
			content_hash TEXT NOT NULL,
			state_snapshot TEXT,
			signature TEXT NOT NULL,
			signer_pubkey TEXT,
			signer_id TEXT,
			os_boundary TEXT,
			filesystem_boundary TEXT,
			physical_boundary TEXT,
			operator_boundary TEXT,
			p1_core_passed INTEGER,
			p1_qemu_passed INTEGER,
			p1_k8s_passed INTEGER,
			p2_multi_passed INTEGER
		)`,
	}
}

func (d *PostgresDriver) Close() error {
	return nil
}

// MySQLDriver implements MySQL backend.
type MySQLDriver struct{}

func (d *MySQLDriver) Name() string {
	return "MySQL"
}

func (d *MySQLDriver) Connect(ctx context.Context, connString string) (*sql.DB, error) {
	db, err := sql.Open("mysql", connString)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func (d *MySQLDriver) CreateTablesSQL() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS qualification_campaigns (
			id VARCHAR(255) PRIMARY KEY,
			resource_id VARCHAR(255) NOT NULL,
			start_time BIGINT NOT NULL,
			end_time BIGINT,
			source_sha VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'RUNNING',
			qualification_level VARCHAR(50),
			evidence_signature TEXT,
			signer_id VARCHAR(255),
			created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL,
			INDEX idx_campaigns_resource (resource_id),
			INDEX idx_campaigns_status (status),
			INDEX idx_campaigns_level (qualification_level),
			INDEX idx_campaigns_start_time (start_time DESC)
		)`,
		`CREATE TABLE IF NOT EXISTS gate_results (
			id INT AUTO_INCREMENT PRIMARY KEY,
			campaign_id VARCHAR(255) NOT NULL,
			sequence INT NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			passed INT NOT NULL,
			evidence TEXT,
			timestamp BIGINT NOT NULL,
			FOREIGN KEY(campaign_id) REFERENCES qualification_campaigns(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS backend_specific_gates (
			id INT AUTO_INCREMENT PRIMARY KEY,
			campaign_id VARCHAR(255) NOT NULL,
			backend_profile VARCHAR(50) NOT NULL,
			sequence INT NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			passed INT NOT NULL,
			evidence TEXT,
			timestamp BIGINT NOT NULL,
			FOREIGN KEY(campaign_id) REFERENCES qualification_campaigns(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS campaign_evidence (
			campaign_id VARCHAR(255) PRIMARY KEY,
			content_hash VARCHAR(255) NOT NULL,
			state_snapshot LONGTEXT,
			signature TEXT NOT NULL,
			signer_pubkey TEXT,
			signer_id VARCHAR(255),
			os_boundary VARCHAR(50),
			filesystem_boundary VARCHAR(50),
			physical_boundary VARCHAR(50),
			operator_boundary VARCHAR(50),
			p1_core_passed INT,
			p1_qemu_passed INT,
			p1_k8s_passed INT,
			p2_multi_passed INT,
			FOREIGN KEY(campaign_id) REFERENCES qualification_campaigns(id) ON DELETE CASCADE
		)`,
	}
}

func (d *MySQLDriver) Close() error {
	return nil
}
