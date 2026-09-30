package providers

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CampaignAnalytics provides analytics and metrics on qualification campaigns.
type CampaignAnalytics struct {
	db *sql.DB
}

// NewCampaignAnalytics creates an analytics engine.
func NewCampaignAnalytics(db *sql.DB) *CampaignAnalytics {
	return &CampaignAnalytics{db: db}
}

// QualificationRates represents pass/fail rates for a period.
type QualificationRates struct {
	TotalCampaigns  int64
	PassedCampaigns int64
	FailedCampaigns int64
	PassRate        float64 // 0.0 to 1.0
}

// GetQualificationRates returns pass rates for a time period.
func (ca *CampaignAnalytics) GetQualificationRates(ctx context.Context, since time.Time) (*QualificationRates, error) {
	sinceMs := since.UnixMilli()

	row := ca.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'PASSED' THEN 1 ELSE 0 END) as passed,
			SUM(CASE WHEN status = 'FAILED' THEN 1 ELSE 0 END) as failed
		FROM qualification_campaigns
		WHERE start_time >= ?`, sinceMs)

	rates := &QualificationRates{}
	var passed, failed sql.NullInt64

	if err := row.Scan(&rates.TotalCampaigns, &passed, &failed); err != nil {
		return nil, fmt.Errorf("failed to query qualification rates: %w", err)
	}

	if passed.Valid {
		rates.PassedCampaigns = passed.Int64
	}
	if failed.Valid {
		rates.FailedCampaigns = failed.Int64
	}

	if rates.TotalCampaigns > 0 {
		rates.PassRate = float64(rates.PassedCampaigns) / float64(rates.TotalCampaigns)
	}

	return rates, nil
}

// LevelDistribution represents campaign distribution across qualification levels.
type LevelDistribution struct {
	Level  string
	Count  int64
	Percent float64
}

// GetLevelDistribution returns distribution of campaigns by qualification level.
func (ca *CampaignAnalytics) GetLevelDistribution(ctx context.Context) ([]LevelDistribution, error) {
	rows, err := ca.db.QueryContext(ctx, `
		SELECT
			qualification_level,
			COUNT(*) as count
		FROM qualification_campaigns
		WHERE qualification_level IS NOT NULL
		GROUP BY qualification_level
		ORDER BY count DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query level distribution: %w", err)
	}
	defer rows.Close()

	// First pass: count totals
	var distributions []LevelDistribution
	var total int64

	for rows.Next() {
		var level string
		var count int64
		if err := rows.Scan(&level, &count); err != nil {
			return nil, err
		}
		distributions = append(distributions, LevelDistribution{
			Level: level,
			Count: count,
		})
		total += count
	}

	// Second pass: calculate percentages
	for i := range distributions {
		if total > 0 {
			distributions[i].Percent = float64(distributions[i].Count) / float64(total) * 100
		}
	}

	return distributions, rows.Err()
}

// GateMetrics represents performance metrics for a single gate.
type GateMetrics struct {
	Sequence   int
	Name       string
	TotalRuns  int64
	PassCount  int64
	FailCount  int64
	PassRate   float64
	AvgDuration time.Duration
}

// GetGateMetrics returns performance metrics for all gates.
func (ca *CampaignAnalytics) GetGateMetrics(ctx context.Context) ([]GateMetrics, error) {
	rows, err := ca.db.QueryContext(ctx, `
		SELECT
			sequence,
			name,
			COUNT(*) as total,
			SUM(CASE WHEN passed = 1 THEN 1 ELSE 0 END) as passed,
			SUM(CASE WHEN passed = 0 THEN 1 ELSE 0 END) as failed,
			AVG(CAST(strftime('%s', datetime(timestamp/1000, 'unixepoch')) AS REAL)) as avg_timestamp
		FROM gate_results
		GROUP BY sequence, name
		ORDER BY sequence`)
	if err != nil {
		return nil, fmt.Errorf("failed to query gate metrics: %w", err)
	}
	defer rows.Close()

	var metrics []GateMetrics
	for rows.Next() {
		var m GateMetrics
		var avgTs sql.NullFloat64

		if err := rows.Scan(&m.Sequence, &m.Name, &m.TotalRuns, &m.PassCount, &m.FailCount, &avgTs); err != nil {
			return nil, err
		}

		if m.TotalRuns > 0 {
			m.PassRate = float64(m.PassCount) / float64(m.TotalRuns)
		}

		metrics = append(metrics, m)
	}

	return metrics, rows.Err()
}

// BackendQualificationStats represents qualification statistics per backend.
type BackendQualificationStats struct {
	Backend        string
	TotalCampaigns int64
	PassedCount    int64
	PassRate       float64
	AvgGatesPassed int64
}

// GetBackendQualificationStats returns qualification rates by backend.
func (ca *CampaignAnalytics) GetBackendQualificationStats(ctx context.Context) ([]BackendQualificationStats, error) {
	rows, err := ca.db.QueryContext(ctx, `
		SELECT
			bsg.backend_profile,
			COUNT(DISTINCT bsg.campaign_id) as total_campaigns,
			SUM(CASE WHEN bsg.passed = 1 THEN 1 ELSE 0 END) as passed,
			AVG(CASE WHEN bsg.passed = 1 THEN 1 ELSE 0 END) as pass_rate,
			AVG(bsg.sequence) as avg_gates_passed
		FROM backend_specific_gates bsg
		GROUP BY bsg.backend_profile
		ORDER BY total_campaigns DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query backend stats: %w", err)
	}
	defer rows.Close()

	var stats []BackendQualificationStats
	for rows.Next() {
		var s BackendQualificationStats
		var passRate sql.NullFloat64
		var avgGates sql.NullFloat64

		if err := rows.Scan(&s.Backend, &s.TotalCampaigns, &s.PassedCount, &passRate, &avgGates); err != nil {
			return nil, err
		}

		if passRate.Valid {
			s.PassRate = passRate.Float64
		}
		if avgGates.Valid {
			s.AvgGatesPassed = int64(avgGates.Float64)
		}

		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// TrendPoint represents a data point in a trend.
type TrendPoint struct {
	Date     string  // YYYY-MM-DD
	Total    int64
	Passed   int64
	Failed   int64
	PassRate float64
}

// GetQualificationTrend returns daily pass rates over time.
func (ca *CampaignAnalytics) GetQualificationTrend(ctx context.Context, days int) ([]TrendPoint, error) {
	query := `
		SELECT
			DATE(datetime(start_time/1000, 'unixepoch')) as date,
			COUNT(*) as total,
			SUM(CASE WHEN status = 'PASSED' THEN 1 ELSE 0 END) as passed,
			SUM(CASE WHEN status = 'FAILED' THEN 1 ELSE 0 END) as failed
		FROM qualification_campaigns
		WHERE start_time >= (strftime('%s', 'now', '-' || ? || ' days') * 1000)
		GROUP BY DATE(datetime(start_time/1000, 'unixepoch'))
		ORDER BY date ASC`

	rows, err := ca.db.QueryContext(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query trend: %w", err)
	}
	defer rows.Close()

	var trends []TrendPoint
	for rows.Next() {
		var t TrendPoint
		if err := rows.Scan(&t.Date, &t.Total, &t.Passed, &t.Failed); err != nil {
			return nil, err
		}

		if t.Total > 0 {
			t.PassRate = float64(t.Passed) / float64(t.Total)
		}

		trends = append(trends, t)
	}

	return trends, rows.Err()
}

// ResourceStats represents qualification statistics for a single resource.
type ResourceStats struct {
	ResourceID     string
	TotalCampaigns int64
	PassedCount    int64
	FailedCount    int64
	PassRate       float64
	LastQualified  time.Time
	LastLevel      string
}

// GetResourceStats returns statistics for a specific resource.
func (ca *CampaignAnalytics) GetResourceStats(ctx context.Context, resourceID string) (*ResourceStats, error) {
	row := ca.db.QueryRowContext(ctx, `
		SELECT
			resource_id,
			COUNT(*) as total,
			SUM(CASE WHEN status = 'PASSED' THEN 1 ELSE 0 END) as passed,
			SUM(CASE WHEN status = 'FAILED' THEN 1 ELSE 0 END) as failed,
			MAX(end_time) as last_qualified,
			(SELECT qualification_level FROM qualification_campaigns
			 WHERE resource_id = ? ORDER BY end_time DESC LIMIT 1) as last_level
		FROM qualification_campaigns
		WHERE resource_id = ?
		GROUP BY resource_id`, resourceID, resourceID)

	stats := &ResourceStats{ResourceID: resourceID}
	var lastQualified sql.NullInt64
	var lastLevel sql.NullString

	if err := row.Scan(&stats.ResourceID, &stats.TotalCampaigns, &stats.PassedCount, &stats.FailedCount, &lastQualified, &lastLevel); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no campaigns found for resource: %s", resourceID)
		}
		return nil, err
	}

	if stats.TotalCampaigns > 0 {
		stats.PassRate = float64(stats.PassedCount) / float64(stats.TotalCampaigns)
	}

	if lastQualified.Valid {
		stats.LastQualified = time.UnixMilli(lastQualified.Int64)
	}

	if lastLevel.Valid {
		stats.LastLevel = lastLevel.String
	}

	return stats, nil
}

// ComplianceSummary represents overall compliance metrics.
type ComplianceSummary struct {
	TotalCampaigns       int64
	PassedCampaigns      int64
	FailedCampaigns      int64
	OverallPassRate      float64
	ResourcesCovered     int64
	AvgQualificationTime time.Duration
	LeastQualifiedLevel  string
	MostQualifiedLevel   string
}

// GetComplianceSummary returns overall compliance metrics.
func (ca *CampaignAnalytics) GetComplianceSummary(ctx context.Context) (*ComplianceSummary, error) {
	summary := &ComplianceSummary{}

	// Overall pass rate
	row := ca.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'PASSED' THEN 1 ELSE 0 END) as passed,
			SUM(CASE WHEN status = 'FAILED' THEN 1 ELSE 0 END) as failed
		FROM qualification_campaigns`)

	var passed, failed sql.NullInt64
	if err := row.Scan(&summary.TotalCampaigns, &passed, &failed); err != nil {
		return nil, err
	}

	if passed.Valid {
		summary.PassedCampaigns = passed.Int64
	}
	if failed.Valid {
		summary.FailedCampaigns = failed.Int64
	}

	if summary.TotalCampaigns > 0 {
		summary.OverallPassRate = float64(summary.PassedCampaigns) / float64(summary.TotalCampaigns)
	}

	// Resources covered
	row = ca.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT resource_id) FROM qualification_campaigns")
	row.Scan(&summary.ResourcesCovered)

	// Least and most qualified levels
	row = ca.db.QueryRowContext(ctx, `
		SELECT qualification_level FROM qualification_campaigns
		WHERE qualification_level IS NOT NULL
		GROUP BY qualification_level ORDER BY COUNT(*) ASC LIMIT 1`)
	row.Scan(&summary.LeastQualifiedLevel)

	row = ca.db.QueryRowContext(ctx, `
		SELECT qualification_level FROM qualification_campaigns
		WHERE qualification_level IS NOT NULL
		GROUP BY qualification_level ORDER BY COUNT(*) DESC LIMIT 1`)
	row.Scan(&summary.MostQualifiedLevel)

	return summary, nil
}
