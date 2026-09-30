package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"decentralized.host/pkg/audit"
)

// ComplianceReport represents an exportable compliance audit report.
type ComplianceReport struct {
	ReportID          string                  `json:"report_id"`
	Generated         time.Time               `json:"generated"`
	Period            string                  `json:"period"` // e.g., "2026-01-01 to 2026-12-31"
	OrganizationName  string                  `json:"organization_name"`
	AuditorName       string                  `json:"auditor_name"`
	Summary           ComplianceSummary       `json:"summary"`
	CampaignSnapshots []CampaignSnapshot      `json:"campaigns"`
	TrendAnalysis     []TrendPoint            `json:"trends"`
	BackendStats      []BackendQualificationStats `json:"backend_stats"`
	LevelDistribution []LevelDistribution     `json:"level_distribution"`
	GateMetrics       []GateMetrics           `json:"gate_metrics"`
	ResourceSamples   []ResourceStats         `json:"resource_samples"`
	Integrity         IntegrityCheckResult    `json:"integrity_check"`
	SignatureProof    string                  `json:"signature_proof"` // Ed25519 signature
}

// CampaignSnapshot represents a single campaign in a report.
type CampaignSnapshot struct {
	CampaignID         string    `json:"campaign_id"`
	ResourceID         string    `json:"resource_id"`
	StartTime          time.Time `json:"start_time"`
	EndTime            time.Time `json:"end_time"`
	Status             string    `json:"status"`
	QualificationLevel string    `json:"qualification_level"`
	GatesPassed        int       `json:"gates_passed"`
	TotalGates         int       `json:"total_gates"`
	Evidence           EvidenceSnapshot `json:"evidence"`
}

// EvidenceSnapshot represents cryptographic evidence in a report.
type EvidenceSnapshot struct {
	ContentHash    string `json:"content_hash"`
	Signature      string `json:"signature"`
	SignerID       string `json:"signer_id"`
	OSBoundary     string `json:"os_boundary"`
	FilesystemBoundary string `json:"filesystem_boundary"`
	PhysicalBoundary string `json:"physical_boundary"`
	OperatorBoundary string `json:"operator_boundary"`
}

// IntegrityCheckResult represents verification of report integrity.
type IntegrityCheckResult struct {
	Verified          bool      `json:"verified"`
	CheckTime         time.Time `json:"check_time"`
	CampaignsChecked  int64     `json:"campaigns_checked"`
	CampaignsValid    int64     `json:"campaigns_valid"`
	CampaignsMissing  int64     `json:"campaigns_missing"`
	SignaturesValid   int64     `json:"signatures_valid"`
	SignaturesInvalid int64     `json:"signatures_invalid"`
	TamperingDetected bool      `json:"tampering_detected"`
	Details           string    `json:"details"`
}

// ComplianceReporter generates compliance reports.
type ComplianceReporter struct {
	store     *CampaignStore
	analytics *CampaignAnalytics
	ledger    *audit.Ledger
	signer    *EvidenceQualifier
}

// NewComplianceReporter creates a compliance reporter.
func NewComplianceReporter(store *CampaignStore, analytics *CampaignAnalytics,
	ledger *audit.Ledger, signer *EvidenceQualifier) *ComplianceReporter {
	return &ComplianceReporter{
		store:     store,
		analytics: analytics,
		ledger:    ledger,
		signer:    signer,
	}
}

// GenerateReport creates a comprehensive compliance report.
func (cr *ComplianceReporter) GenerateReport(ctx context.Context,
	orgName string, auditorName string, fromDate time.Time, toDate time.Time) (*ComplianceReport, error) {

	report := &ComplianceReport{
		ReportID:         fmt.Sprintf("compliance-%d", time.Now().UnixNano()),
		Generated:        time.Now(),
		Period:           fmt.Sprintf("%s to %s", fromDate.Format("2006-01-02"), toDate.Format("2006-01-02")),
		OrganizationName: orgName,
		AuditorName:      auditorName,
	}

	// Get summary statistics
	summary, err := cr.analytics.GetComplianceSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get compliance summary: %w", err)
	}
	report.Summary = *summary

	// Get trend analysis
	days := int(toDate.Sub(fromDate).Hours() / 24)
	trends, err := cr.analytics.GetQualificationTrend(ctx, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get trend analysis: %w", err)
	}
	report.TrendAnalysis = trends

	// Get backend statistics
	backendStats, err := cr.analytics.GetBackendQualificationStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get backend stats: %w", err)
	}
	report.BackendStats = backendStats

	// Get level distribution
	levelDist, err := cr.analytics.GetLevelDistribution(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get level distribution: %w", err)
	}
	report.LevelDistribution = levelDist

	// Get gate metrics
	gateMetrics, err := cr.analytics.GetGateMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gate metrics: %w", err)
	}
	report.GateMetrics = gateMetrics

	// Verify integrity of campaigns in database
	integrityResult, err := cr.verifyCampaignIntegrity(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to verify integrity: %w", err)
	}
	report.Integrity = *integrityResult

	// Sign the report
	reportJSON, _ := json.Marshal(report)
	reportHash := cr.signer.HashResourceState(string(reportJSON))

	// Create a QualifiedEvidence for the report itself
	reportEvidence := &QualifiedEvidence{
		CampaignID: report.ReportID,
		ResourceID: fmt.Sprintf("compliance-report:%s", report.ReportID),
		ContentHash: reportHash,
		StateSnapshot: fmt.Sprintf("Compliance report with %d campaigns, %d resources, %d gates",
			report.Summary.TotalCampaigns, report.Summary.ResourcesCovered, len(report.GateMetrics)),
	}

	if err := cr.signer.SignEvidence(reportEvidence); err != nil {
		return nil, fmt.Errorf("failed to sign report: %w", err)
	}

	report.SignatureProof = reportEvidence.Signature

	return report, nil
}

// verifyCampaignIntegrity checks cryptographic integrity of all campaigns.
func (cr *ComplianceReporter) verifyCampaignIntegrity(ctx context.Context) (*IntegrityCheckResult, error) {
	result := &IntegrityCheckResult{
		CheckTime: time.Now(),
	}

	// Get all campaigns from database
	campaigns, err := cr.store.QueryCampaigns(ctx, "", "", "", 10000, 0)
	if err != nil {
		result.Details = fmt.Sprintf("Query failed: %v", err)
		return result, nil
	}

	result.CampaignsChecked = int64(len(campaigns))
	result.Verified = true

	// Verify each campaign's evidence
	for _, campaign := range campaigns {
		if campaign.Evidence == nil {
			result.CampaignsMissing++
			result.Verified = false
			continue
		}

		if err := VerifyEvidence(campaign.Evidence); err != nil {
			result.SignaturesInvalid++
			result.Verified = false
		} else {
			result.SignaturesValid++
			result.CampaignsValid++
		}
	}

	// Check for tampering patterns
	if result.SignaturesInvalid > 0 || result.CampaignsMissing > 0 {
		result.TamperingDetected = true
		result.Details = fmt.Sprintf("Found %d invalid signatures and %d missing evidence records",
			result.SignaturesInvalid, result.CampaignsMissing)
	} else {
		result.Details = fmt.Sprintf("All %d campaigns verified successfully", result.CampaignsValid)
	}

	return result, nil
}

// ExportJSON exports the report as JSON.
func (cr *ComplianceReporter) ExportJSON(report *ComplianceReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// ExportCSV exports key metrics as CSV.
func (cr *ComplianceReporter) ExportCSV(report *ComplianceReport) (string, error) {
	csv := "Date,Total,Passed,Failed,PassRate\n"

	for _, trend := range report.TrendAnalysis {
		line := fmt.Sprintf("%s,%d,%d,%d,%.2f%%\n",
			trend.Date, trend.Total, trend.Passed, trend.Failed, trend.PassRate*100)
		csv += line
	}

	return csv, nil
}

// QualificationBadge represents a digital qualification badge.
type QualificationBadge struct {
	ResourceID         string    `json:"resource_id"`
	QualificationLevel string    `json:"qualification_level"`
	IssuedAt           time.Time `json:"issued_at"`
	ExpiresAt          time.Time `json:"expires_at"`
	SignedProof        string    `json:"signed_proof"` // Ed25519 signature
	CampaignID         string    `json:"campaign_id"`
}

// GenerateBadge creates a signed qualification badge for a resource.
func (cr *ComplianceReporter) GenerateBadge(ctx context.Context, resourceID string, validity time.Duration) (*QualificationBadge, error) {
	// Get resource statistics
	stats, err := cr.analytics.GetResourceStats(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource stats: %w", err)
	}

	if stats.LastLevel == "" {
		return nil, fmt.Errorf("resource has no qualification level")
	}

	badge := &QualificationBadge{
		ResourceID:         resourceID,
		QualificationLevel: stats.LastLevel,
		IssuedAt:           time.Now(),
		ExpiresAt:          time.Now().Add(validity),
	}

	// Sign the badge
	badgeData := fmt.Sprintf("%s|%s|%d|%d",
		resourceID, stats.LastLevel,
		badge.IssuedAt.Unix(), badge.ExpiresAt.Unix())

	badgeEvidence := &QualifiedEvidence{
		ResourceID: resourceID,
		ContentHash: cr.signer.HashResourceState(badgeData),
	}

	if err := cr.signer.SignEvidence(badgeEvidence); err != nil {
		return nil, fmt.Errorf("failed to sign badge: %w", err)
	}

	badge.SignedProof = badgeEvidence.Signature

	return badge, nil
}

// VerifyBadge verifies a qualification badge.
func (cr *ComplianceReporter) VerifyBadge(badge *QualificationBadge) (bool, error) {
	if time.Now().After(badge.ExpiresAt) {
		return false, fmt.Errorf("badge has expired")
	}

	// In a real implementation, would verify the signature
	// For now, return true if signature is present
	return badge.SignedProof != "", nil
}
