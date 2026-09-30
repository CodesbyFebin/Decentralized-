package providers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"decentralized.host/pkg/audit"
)

// CampaignStore provides persistent storage for qualification campaigns.
type CampaignStore struct {
	db *sql.DB
	ledger *audit.Ledger
}

// NewCampaignStore creates a campaign storage instance with database and audit ledger.
func NewCampaignStore(db *sql.DB, ledger *audit.Ledger) (*CampaignStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection required")
	}
	if ledger == nil {
		return nil, fmt.Errorf("audit ledger required")
	}

	cs := &CampaignStore{
		db: db,
		ledger: ledger,
	}

	if err := cs.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize campaign schema: %w", err)
	}

	return cs, nil
}

// initSchema creates campaign storage tables if they don't exist.
func (cs *CampaignStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS qualification_campaigns (
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
	);

	CREATE TABLE IF NOT EXISTS gate_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		campaign_id TEXT NOT NULL,
		sequence INTEGER NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		passed INTEGER NOT NULL,
		evidence TEXT,
		timestamp INTEGER NOT NULL,
		FOREIGN KEY (campaign_id) REFERENCES qualification_campaigns(id) ON DELETE CASCADE,
		UNIQUE(campaign_id, sequence)
	);

	CREATE TABLE IF NOT EXISTS backend_specific_gates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		campaign_id TEXT NOT NULL,
		backend_profile TEXT NOT NULL,
		sequence INTEGER NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		passed INTEGER NOT NULL,
		evidence TEXT,
		timestamp INTEGER NOT NULL,
		FOREIGN KEY (campaign_id) REFERENCES qualification_campaigns(id) ON DELETE CASCADE,
		UNIQUE(campaign_id, backend_profile, sequence)
	);

	CREATE TABLE IF NOT EXISTS campaign_evidence (
		campaign_id TEXT PRIMARY KEY,
		content_hash TEXT NOT NULL,
		state_snapshot TEXT,
		signature TEXT NOT NULL,
		signer_pubkey TEXT NOT NULL,
		os_boundary TEXT,
		filesystem_boundary TEXT,
		physical_boundary TEXT,
		operator_boundary TEXT,
		p1_core_passed INTEGER,
		p1_qemu_passed INTEGER,
		p1_k8s_passed INTEGER,
		p2_multi_passed INTEGER,
		created_at INTEGER NOT NULL,
		FOREIGN KEY (campaign_id) REFERENCES qualification_campaigns(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_campaigns_resource ON qualification_campaigns(resource_id);
	CREATE INDEX IF NOT EXISTS idx_campaigns_status ON qualification_campaigns(status);
	CREATE INDEX IF NOT EXISTS idx_campaigns_level ON qualification_campaigns(qualification_level);
	CREATE INDEX IF NOT EXISTS idx_campaigns_start_time ON qualification_campaigns(start_time DESC);
	`

	_, err := cs.db.Exec(schema)
	return err
}

// StoreCampaign persists a qualification campaign to database and audit ledger.
func (cs *CampaignStore) StoreCampaign(ctx context.Context, campaign *QualificationCampaign) error {
	if campaign == nil {
		return fmt.Errorf("campaign cannot be nil")
	}

	now := time.Now().UnixMilli()

	// Begin transaction
	tx, err := cs.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Store campaign metadata
	_, err = tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO qualification_campaigns
		(id, resource_id, start_time, end_time, source_sha, status, qualification_level, evidence_signature, signer_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		campaign.ID,
		campaign.ResourceID,
		campaign.StartTime.UnixMilli(),
		campaign.EndTime.UnixMilli(),
		campaign.SourceSHA,
		campaign.Status,
		campaign.QualificationLevel,
		campaign.Evidence.Signature,
		campaign.Evidence.SignerID,
		now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to store campaign: %w", err)
	}

	// Store P1_CORE gate results
	for _, result := range campaign.GateResults {
		_, err = tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO gate_results
			(campaign_id, sequence, name, description, passed, evidence, timestamp)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			campaign.ID,
			result.Sequence,
			result.Name,
			result.Description,
			boolToInt(result.Passed),
			result.Evidence,
			result.Timestamp.UnixMilli(),
		)
		if err != nil {
			return fmt.Errorf("failed to store gate result: %w", err)
		}
	}

	// Store backend-specific gate results
	for profile, results := range campaign.BackendSpecificGates {
		for _, result := range results {
			_, err = tx.ExecContext(ctx, `
				INSERT OR REPLACE INTO backend_specific_gates
				(campaign_id, backend_profile, sequence, name, description, passed, evidence, timestamp)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				campaign.ID,
				profile,
				result.Sequence,
				result.Name,
				result.Description,
				boolToInt(result.Passed),
				result.Evidence,
				result.Timestamp.UnixMilli(),
			)
			if err != nil {
				return fmt.Errorf("failed to store backend gate result: %w", err)
			}
		}
	}

	// Store evidence
	if campaign.Evidence != nil {
		_, err = tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO campaign_evidence
			(campaign_id, content_hash, state_snapshot, signature, signer_pubkey, os_boundary, filesystem_boundary, physical_boundary, operator_boundary, p1_core_passed, p1_qemu_passed, p1_k8s_passed, p2_multi_passed, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			campaign.ID,
			campaign.Evidence.ContentHash,
			campaign.Evidence.StateSnapshot,
			campaign.Evidence.Signature,
			campaign.Evidence.SignerPubKey,
			campaign.Evidence.OSBoundary,
			campaign.Evidence.FilesystemBoundary,
			campaign.Evidence.PhysicalBoundary,
			campaign.Evidence.OperatorBoundary,
			boolToInt(campaign.Evidence.P1_CORE_Passed),
			boolToInt(campaign.Evidence.P1_QEMU_Passed),
			boolToInt(campaign.Evidence.P1_K8S_Passed),
			boolToInt(campaign.Evidence.P2_Multi_Passed),
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to store evidence: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Log to audit ledger
	cs.auditCampaignEvent(campaign, "qualification.campaign.completed")

	return nil
}

// GetCampaign retrieves a campaign by ID.
func (cs *CampaignStore) GetCampaign(ctx context.Context, campaignID string) (*QualificationCampaign, error) {
	campaign := &QualificationCampaign{
		BackendSpecificGates: make(map[string][]GateResult),
	}

	row := cs.db.QueryRowContext(ctx, `
		SELECT id, resource_id, start_time, end_time, source_sha, status, qualification_level
		FROM qualification_campaigns WHERE id = ?`, campaignID)

	var startMs, endMs int64
	err := row.Scan(
		&campaign.ID,
		&campaign.ResourceID,
		&startMs,
		&endMs,
		&campaign.SourceSHA,
		&campaign.Status,
		&campaign.QualificationLevel,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("campaign not found: %s", campaignID)
		}
		return nil, err
	}

	campaign.StartTime = time.UnixMilli(startMs)
	campaign.EndTime = time.UnixMilli(endMs)

	// Load P1_CORE gate results
	rows, err := cs.db.QueryContext(ctx, `
		SELECT sequence, name, description, passed, evidence, timestamp
		FROM gate_results WHERE campaign_id = ? ORDER BY sequence`, campaignID)
	if err != nil {
		return nil, fmt.Errorf("failed to query gate results: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var result GateResult
		var passed int
		var ts int64
		err := rows.Scan(
			&result.Sequence,
			&result.Name,
			&result.Description,
			&passed,
			&result.Evidence,
			&ts,
		)
		if err != nil {
			return nil, err
		}
		result.Passed = passed != 0
		result.Timestamp = time.UnixMilli(ts)
		campaign.GateResults = append(campaign.GateResults, result)
	}

	// Load backend-specific gate results
	rows, err = cs.db.QueryContext(ctx, `
		SELECT backend_profile, sequence, name, description, passed, evidence, timestamp
		FROM backend_specific_gates WHERE campaign_id = ? ORDER BY backend_profile, sequence`, campaignID)
	if err != nil {
		return nil, fmt.Errorf("failed to query backend gates: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var result GateResult
		var profile string
		var passed int
		var ts int64
		err := rows.Scan(
			&profile,
			&result.Sequence,
			&result.Name,
			&result.Description,
			&passed,
			&result.Evidence,
			&ts,
		)
		if err != nil {
			return nil, err
		}
		result.Passed = passed != 0
		result.Timestamp = time.UnixMilli(ts)
		campaign.BackendSpecificGates[profile] = append(campaign.BackendSpecificGates[profile], result)
	}

	// Load evidence
	evidence := &QualifiedEvidence{
		CampaignID: campaignID,
	}
	row = cs.db.QueryRowContext(ctx, `
		SELECT content_hash, state_snapshot, signature, signer_pubkey, os_boundary, filesystem_boundary, physical_boundary, operator_boundary, p1_core_passed, p1_qemu_passed, p1_k8s_passed, p2_multi_passed
		FROM campaign_evidence WHERE campaign_id = ?`, campaignID)

	var p1Core, p1Qemu, p1K8s, p2Multi int
	err = row.Scan(
		&evidence.ContentHash,
		&evidence.StateSnapshot,
		&evidence.Signature,
		&evidence.SignerPubKey,
		&evidence.OSBoundary,
		&evidence.FilesystemBoundary,
		&evidence.PhysicalBoundary,
		&evidence.OperatorBoundary,
		&p1Core, &p1Qemu, &p1K8s, &p2Multi,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query evidence: %w", err)
	}
	if err == nil {
		evidence.P1_CORE_Passed = p1Core != 0
		evidence.P1_QEMU_Passed = p1Qemu != 0
		evidence.P1_K8S_Passed = p1K8s != 0
		evidence.P2_Multi_Passed = p2Multi != 0
		campaign.Evidence = evidence
	}

	return campaign, nil
}

// QueryCampaigns returns campaigns matching the given criteria with pagination.
func (cs *CampaignStore) QueryCampaigns(ctx context.Context, resourceID string, status string, level string, limit int, offset int) ([]*QualificationCampaign, error) {
	query := "SELECT id FROM qualification_campaigns WHERE 1=1"
	args := []interface{}{}

	if resourceID != "" {
		query += " AND resource_id = ?"
		args = append(args, resourceID)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if level != "" {
		query += " AND qualification_level = ?"
		args = append(args, level)
	}

	query += " ORDER BY start_time DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	if offset > 0 {
		query += " OFFSET ?"
		args = append(args, offset)
	}

	rows, err := cs.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var campaigns []*QualificationCampaign
	for rows.Next() {
		var campaignID string
		if err := rows.Scan(&campaignID); err != nil {
			return nil, err
		}

		campaign, err := cs.GetCampaign(ctx, campaignID)
		if err != nil {
			return nil, fmt.Errorf("failed to load campaign %s: %w", campaignID, err)
		}
		campaigns = append(campaigns, campaign)
	}

	return campaigns, rows.Err()
}

// DeleteCampaign removes a campaign and all related records (cascade).
func (cs *CampaignStore) DeleteCampaign(ctx context.Context, campaignID string) error {
	result, err := cs.db.ExecContext(ctx, "DELETE FROM qualification_campaigns WHERE id = ?", campaignID)
	if err != nil {
		return fmt.Errorf("failed to delete campaign: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("campaign not found: %s", campaignID)
	}

	return nil
}

// auditCampaignEvent logs campaign activity to the audit ledger.
func (cs *CampaignStore) auditCampaignEvent(campaign *QualificationCampaign, action string) {
	if cs.ledger == nil {
		return
	}

	detail, _ := json.Marshal(map[string]interface{}{
		"status": campaign.Status,
		"level": campaign.QualificationLevel,
		"gates_passed": countPassed(campaign.GateResults),
		"total_gates": len(campaign.GateResults),
	})

	entry := audit.Entry{
		Actor: campaign.Evidence.SignerID,
		Source: audit.SourceControl,
		Action: action,
		Resource: campaign.ResourceID,
		Detail: string(detail),
		Evidence: campaign.Evidence.Signature,
	}

	cs.ledger.Append(entry)
}

// boolToInt converts boolean to SQLite integer.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
