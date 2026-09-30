package providers

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// CampaignArchive manages archiving and compression of old campaigns.
type CampaignArchive struct {
	store       *CampaignStore
	archiveDir  string
	retentionDays int
}

// NewCampaignArchive creates a campaign archive manager.
func NewCampaignArchive(store *CampaignStore, archiveDir string, retentionDays int) (*CampaignArchive, error) {
	if store == nil {
		return nil, fmt.Errorf("campaign store required")
	}
	if archiveDir == "" {
		return nil, fmt.Errorf("archive directory required")
	}
	if retentionDays <= 0 {
		return nil, fmt.Errorf("retention days must be positive")
	}

	if err := os.MkdirAll(archiveDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create archive directory: %w", err)
	}

	return &CampaignArchive{
		store:         store,
		archiveDir:    archiveDir,
		retentionDays: retentionDays,
	}, nil
}

// ArchiveCampaigns archives campaigns older than retention period.
func (ca *CampaignArchive) ArchiveCampaigns(ctx context.Context) (*ArchiveResult, error) {
	cutoffTime := time.Now().AddDate(0, 0, -ca.retentionDays)

	campaigns, err := ca.store.QueryCampaigns(ctx, "", "", "", 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to query campaigns: %w", err)
	}

	result := &ArchiveResult{
		StartTime: time.Now(),
	}

	for _, campaign := range campaigns {
		if campaign.StartTime.After(cutoffTime) {
			continue
		}

		if err := ca.archiveCampaign(ctx, campaign); err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, err.Error())
			continue
		}

		if err := ca.store.DeleteCampaign(ctx, campaign.ID); err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("archive succeeded but delete failed for %s: %v", campaign.ID, err))
			continue
		}

		result.ArchivedCount++
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// archiveCampaign archives a single campaign to compressed tar file.
func (ca *CampaignArchive) archiveCampaign(ctx context.Context, campaign *QualificationCampaign) error {
	fileName := fmt.Sprintf("campaign-%s-%d.tar.gz", campaign.ID, campaign.StartTime)
	filePath := filepath.Join(ca.archiveDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	campaignJSON, err := json.MarshalIndent(campaign, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal campaign: %w", err)
	}

	header := &tar.Header{
		Name: fmt.Sprintf("campaign-%s.json", campaign.ID),
		Size: int64(len(campaignJSON)),
		Mode: 0644,
		ModTime: time.Now(),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}

	if _, err := tarWriter.Write(campaignJSON); err != nil {
		return fmt.Errorf("failed to write campaign data: %w", err)
	}

	return nil
}

// RestoreCampaigns restores campaigns from archive files.
func (ca *CampaignArchive) RestoreCampaigns(ctx context.Context, archiveFile string) (*RestoreResult, error) {
	if archiveFile == "" {
		return nil, fmt.Errorf("archive file path required")
	}

	file, err := os.Open(archiveFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive file: %w", err)
	}
	defer file.Close()

	result := &RestoreResult{
		StartTime: time.Now(),
	}

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("tar read error: %v", err))
			continue
		}

		if err := ca.restoreCampaignFromTar(ctx, tarReader, header, result); err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, err.Error())
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// restoreCampaignFromTar restores a single campaign from tar entry.
func (ca *CampaignArchive) restoreCampaignFromTar(ctx context.Context, tarReader *tar.Reader, header *tar.Header, result *RestoreResult) error {
	campaignData := make([]byte, header.Size)
	if _, err := io.ReadFull(tarReader, campaignData); err != nil {
		return fmt.Errorf("failed to read campaign data from tar: %w", err)
	}

	var campaign QualificationCampaign
	if err := json.Unmarshal(campaignData, &campaign); err != nil {
		return fmt.Errorf("failed to unmarshal campaign: %w", err)
	}

	if err := ca.store.StoreCampaign(ctx, &campaign); err != nil {
		return fmt.Errorf("failed to store restored campaign: %w", err)
	}

	result.RestoredCount++
	return nil
}

// BackupDatabase creates a complete database backup.
func (ca *CampaignArchive) BackupDatabase(ctx context.Context, backupDir string) (*BackupResult, error) {
	if backupDir == "" {
		return nil, fmt.Errorf("backup directory required")
	}

	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	result := &BackupResult{
		StartTime: time.Now(),
		BackupDir: backupDir,
	}

	// Backup campaigns
	campaigns, err := ca.store.QueryCampaigns(ctx, "", "", "", 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to query campaigns: %w", err)
	}

	backupFile := filepath.Join(backupDir, fmt.Sprintf("campaigns-backup-%d.tar.gz", time.Now().Unix()))
	file, err := os.Create(backupFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, campaign := range campaigns {
		campaignJSON, err := json.MarshalIndent(campaign, "", "  ")
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to marshal campaign %s: %v", campaign.ID, err))
			continue
		}

		header := &tar.Header{
			Name:    fmt.Sprintf("campaign-%s.json", campaign.ID),
			Size:    int64(len(campaignJSON)),
			Mode:    0644,
			ModTime: time.Now(),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to write tar header: %v", err))
			continue
		}

		if _, err := tarWriter.Write(campaignJSON); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to write campaign data: %v", err))
			continue
		}

		result.CampaignsBackedUp++
	}

	// Calculate backup hash
	file.Seek(0, 0)
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate backup hash: %w", err)
	}

	result.BackupFile = backupFile
	result.BackupHash = fmt.Sprintf("%x", hash.Sum(nil))
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// ArchiveResult represents the result of archiving campaigns.
type ArchiveResult struct {
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	ArchivedCount int
	FailedCount  int
	Errors       []string
}

// RestoreResult represents the result of restoring campaigns.
type RestoreResult struct {
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	RestoredCount int
	FailedCount  int
	Errors       []string
}

// BackupResult represents the result of backing up the database.
type BackupResult struct {
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	BackupDir      string
	BackupFile     string
	BackupHash     string
	CampaignsBackedUp int
	Errors         []string
}
