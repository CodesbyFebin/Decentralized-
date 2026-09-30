package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// CampaignCLI provides command-line interface for campaign management.
type CampaignCLI struct {
	store   *CampaignStore
	archive *CampaignArchive
	analytics *CampaignAnalytics
	reporter *ComplianceReporter
}

// NewCampaignCLI creates a CLI instance.
func NewCampaignCLI(store *CampaignStore, archive *CampaignArchive, analytics *CampaignAnalytics, reporter *ComplianceReporter) *CampaignCLI {
	return &CampaignCLI{
		store:     store,
		archive:   archive,
		analytics: analytics,
		reporter:  reporter,
	}
}

// ExecuteCommand executes a CLI command.
func (cli *CampaignCLI) ExecuteCommand(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("command required")
	}

	command := args[0]

	switch command {
	case "list":
		return cli.ListCampaigns(ctx, args[1:])
	case "get":
		return cli.GetCampaign(ctx, args[1:])
	case "stats":
		return cli.GetStats(ctx, args[1:])
	case "export":
		return cli.ExportCampaign(ctx, args[1:])
	case "archive":
		return cli.ArchiveCampaigns(ctx, args[1:])
	case "restore":
		return cli.RestoreCampaigns(ctx, args[1:])
	case "backup":
		return cli.BackupDatabase(ctx, args[1:])
	case "report":
		return cli.GenerateReport(ctx, args[1:])
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// ListCampaigns lists campaigns with optional filters.
func (cli *CampaignCLI) ListCampaigns(ctx context.Context, args []string) error {
	resourceID := ""
	status := ""
	level := ""
	limit := 100

	campaigns, err := cli.store.QueryCampaigns(ctx, resourceID, status, level, limit, 0)
	if err != nil {
		return fmt.Errorf("failed to query campaigns: %w", err)
	}

	if len(campaigns) == 0 {
		fmt.Println("No campaigns found")
		return nil
	}

	fmt.Printf("Found %d campaigns:\n", len(campaigns))
	fmt.Printf("%-40s %-20s %-15s %-20s\n", "ID", "Resource", "Status", "Level")
	fmt.Println("------------------------------------------------------------------------------------")

	for _, campaign := range campaigns {
		level := "UNKNOWN"
		if campaign.QualificationLevel != "" {
			level = campaign.QualificationLevel
		}
		fmt.Printf("%-40s %-20s %-15s %-20s\n", campaign.ID, campaign.ResourceID, campaign.Status, level)
	}

	return nil
}

// GetCampaign retrieves and displays a campaign.
func (cli *CampaignCLI) GetCampaign(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("campaign ID required")
	}

	campaignID := args[0]

	campaign, err := cli.store.GetCampaign(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("failed to get campaign: %w", err)
	}

	jsonData, err := json.MarshalIndent(campaign, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal campaign: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

// GetStats retrieves and displays compliance statistics.
func (cli *CampaignCLI) GetStats(ctx context.Context, args []string) error {
	summary, err := cli.analytics.GetComplianceSummary(ctx)
	if err != nil {
		return fmt.Errorf("failed to get compliance summary: %w", err)
	}

	fmt.Printf("Compliance Statistics\n")
	fmt.Printf("======================\n")
	fmt.Printf("Total Campaigns:        %d\n", summary.TotalCampaigns)
	fmt.Printf("Passed Campaigns:       %d\n", summary.PassedCampaigns)
	fmt.Printf("Failed Campaigns:       %d\n", summary.FailedCampaigns)
	fmt.Printf("Overall Pass Rate:      %.2f%%\n", summary.OverallPassRate*100)
	fmt.Printf("Resources Covered:      %d\n", summary.ResourcesCovered)
	fmt.Printf("Least Qualified Level:  %s\n", summary.LeastQualifiedLevel)
	fmt.Printf("Most Qualified Level:   %s\n", summary.MostQualifiedLevel)

	return nil
}

// ExportCampaign exports a campaign to JSON file.
func (cli *CampaignCLI) ExportCampaign(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("campaign ID and output file required")
	}

	campaignID := args[0]
	outputFile := args[1]

	campaign, err := cli.store.GetCampaign(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("failed to get campaign: %w", err)
	}

	jsonData, err := json.MarshalIndent(campaign, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal campaign: %w", err)
	}

	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	fmt.Printf("Campaign exported to %s\n", outputFile)
	return nil
}

// ArchiveCampaigns archives old campaigns.
func (cli *CampaignCLI) ArchiveCampaigns(ctx context.Context, args []string) error {
	result, err := cli.archive.ArchiveCampaigns(ctx)
	if err != nil {
		return fmt.Errorf("failed to archive campaigns: %w", err)
	}

	fmt.Printf("Archive Complete\n")
	fmt.Printf("=================\n")
	fmt.Printf("Archived:       %d\n", result.ArchivedCount)
	fmt.Printf("Failed:         %d\n", result.FailedCount)
	fmt.Printf("Duration:       %v\n", result.Duration)

	if len(result.Errors) > 0 {
		fmt.Printf("Errors:\n")
		for _, e := range result.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	return nil
}

// RestoreCampaigns restores campaigns from archive.
func (cli *CampaignCLI) RestoreCampaigns(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("archive file required")
	}

	archiveFile := args[0]

	result, err := cli.archive.RestoreCampaigns(ctx, archiveFile)
	if err != nil {
		return fmt.Errorf("failed to restore campaigns: %w", err)
	}

	fmt.Printf("Restore Complete\n")
	fmt.Printf("=================\n")
	fmt.Printf("Restored:       %d\n", result.RestoredCount)
	fmt.Printf("Failed:         %d\n", result.FailedCount)
	fmt.Printf("Duration:       %v\n", result.Duration)

	if len(result.Errors) > 0 {
		fmt.Printf("Errors:\n")
		for _, e := range result.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	return nil
}

// BackupDatabase backs up the entire database.
func (cli *CampaignCLI) BackupDatabase(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("backup directory required")
	}

	backupDir := args[0]

	result, err := cli.archive.BackupDatabase(ctx, backupDir)
	if err != nil {
		return fmt.Errorf("failed to backup database: %w", err)
	}

	fmt.Printf("Backup Complete\n")
	fmt.Printf("================\n")
	fmt.Printf("Backed Up:      %d campaigns\n", result.CampaignsBackedUp)
	fmt.Printf("Backup File:    %s\n", result.BackupFile)
	fmt.Printf("Backup Hash:    %s\n", result.BackupHash)
	fmt.Printf("Duration:       %v\n", result.Duration)

	if len(result.Errors) > 0 {
		fmt.Printf("Errors:\n")
		for _, e := range result.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	return nil
}

// GenerateReport generates a compliance report.
func (cli *CampaignCLI) GenerateReport(ctx context.Context, args []string) error {
	if cli.reporter == nil {
		return fmt.Errorf("compliance reporter not configured")
	}

	report, err := cli.reporter.GenerateReport(ctx, "Decentralized.Host", "CLI",
		cli.analytics.GetCampaignStartDate(ctx),
		cli.analytics.GetCampaignEndDate(ctx))
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	jsonData, err := cli.reporter.ExportJSON(report)
	if err != nil {
		return fmt.Errorf("failed to export report: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}
