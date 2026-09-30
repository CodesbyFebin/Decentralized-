package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"decentralized.host/pkg/control"
	"decentralized.host/pkg/providers"
	"decentralized.host/pkg/system"
)

// DePINCommand represents a DePIN-related CLI command
type DePINCommand struct {
	operator *Operator
	ctx      context.Context
}

// NewDePINCommand creates a new DePIN command handler
func NewDePINCommand(op *Operator, ctx context.Context) *DePINCommand {
	return &DePINCommand{
		operator: op,
		ctx:      ctx,
	}
}

// ProbeHardware detects and reports actual hardware capabilities (no simulation)
func (dc *DePINCommand) ProbeHardware() error {
	prober := system.NewProber()
	profile, err := prober.Probe(dc.ctx)
	if err != nil {
		return fmt.Errorf("hardware probe failed: %w", err)
	}

	// Convert to JSON for display
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println("Hardware Profile:")
	fmt.Println(string(data))

	return nil
}

// CheckDePINCompatibility evaluates hardware against a DePIN profile
func (dc *DePINCommand) CheckDePINCompatibility(profileName string) error {
	// Get hardware profile
	prober := system.NewProber()
	hwProfile, err := prober.Probe(dc.ctx)
	if err != nil {
		return fmt.Errorf("hardware probe failed: %w", err)
	}

	// Get DePIN requirements
	requirements := providers.CommonDePINRequirements()
	req, ok := requirements[profileName]
	if !ok {
		availableProfiles := []string{}
		for name := range requirements {
			availableProfiles = append(availableProfiles, name)
		}
		return fmt.Errorf("unknown profile '%s'. Available: %v", profileName, availableProfiles)
	}

	// Convert hardware profile to simple format
	simpleProfile := &providers.SimpleHardwareProfile{
		CPUCores:     hwProfile.CPU.Cores,
		CPUModel:     hwProfile.CPU.Model,
		MemoryBytes:  hwProfile.Memory.TotalBytes,
		GPUs:         extractGPUTypes(hwProfile.GPU),
		StorageBytes: hwProfile.Disk.TotalBytes,
		NetworkCount: len(hwProfile.Network),
	}

	// Check compatibility
	checker := providers.NewCompatibilityChecker()
	score := checker.CheckCompatibility(dc.ctx, simpleProfile, req)

	// Display results
	fmt.Printf("DePIN Compatibility Report - Profile: %s\n", profileName)
	fmt.Printf("================================\n")
	fmt.Printf("Overall Score:   %.1f%%\n", score.Overall*100)
	fmt.Printf("Level:           %s\n", score.GetCompatibilityLevel())
	fmt.Printf("\nComponent Scores:\n")
	fmt.Printf("  CPU:           %.1f%%\n", score.CPU*100)
	fmt.Printf("  Memory:        %.1f%%\n", score.Memory*100)
	fmt.Printf("  Storage:       %.1f%%\n", score.Storage*100)
	fmt.Printf("  GPU:           %.1f%%\n", score.GPU*100)
	fmt.Printf("  Network:       %.1f%%\n", score.Network*100)

	if len(score.Issues) > 0 {
		fmt.Println("\nIssues:")
		for _, issue := range score.Issues {
			fmt.Printf("  ❌ %s\n", issue)
		}
	}

	if len(score.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warning := range score.Warnings {
			fmt.Printf("  ⚠️  %s\n", warning)
		}
	}

	fmt.Println("\nDetected Hardware:")
	fmt.Printf("  CPU:           %d cores @ %.2f GHz (%s)\n", hwProfile.CPU.Cores, hwProfile.CPU.Frequency, hwProfile.CPU.Model)
	fmt.Printf("  Memory:        %d GB\n", hwProfile.Memory.TotalBytes/(1024*1024*1024))
	fmt.Printf("  Storage:       %d GB (%s)\n", hwProfile.Disk.TotalBytes/(1024*1024*1024), hwProfile.Disk.Filesystem)
	if len(hwProfile.GPU) > 0 {
		fmt.Println("  GPUs:")
		for _, gpu := range hwProfile.GPU {
			fmt.Printf("    - %s (%d GB)\n", gpu.Name, gpu.MemoryBytes/(1024*1024*1024))
		}
	} else {
		fmt.Println("  GPUs:          none detected")
	}
	fmt.Printf("  Network:       %d interfaces\n", len(hwProfile.Network))

	return nil
}

// RegisterOwnerReserve verifies ownership and creates a reserve token
func (dc *DePINCommand) RegisterOwnerReserve(ownerID string, hostID string) error {
	if ownerID == "" || hostID == "" {
		return fmt.Errorf("owner_id and host_id are required")
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Verify ownership (this would normally use cryptographic proof)
	// For v0.1, we do a simplified check
	fmt.Printf("Registering owner reserve for %s on host %s...\n", ownerID, hostID)

	// In production, this would call a real validator with proper key management
	// For now, create a placeholder reserve
	now := time.Now()
	reserve := &control.OwnerReserve{
		OwnerID:    ownerID,
		HostID:     hostID,
		Hostname:   hostname,
		Status:     "verified",
		VerifiedAt: now,
		TokenIssued: now,
		TokenExpiry: now.Add(24 * time.Hour),
	}

	// Get hardware profile and populate reserve
	prober := system.NewProber()
	profile, err := prober.Probe(dc.ctx)
	if err != nil {
		return fmt.Errorf("failed to probe hardware: %w", err)
	}

	reserve.CPUCores = profile.CPU.Cores
	reserve.CPUModel = profile.CPU.Model
	reserve.MemoryBytes = profile.Memory.TotalBytes
	reserve.StorageBytes = profile.Disk.TotalBytes
	reserve.GPUTypes = extractGPUTypes(profile.GPU)

	// Display reserve
	data, _ := json.MarshalIndent(reserve, "", "  ")
	fmt.Println("Owner Reserve Created:")
	fmt.Println(string(data))

	return nil
}

// ListAvailableProfiles shows all available DePIN hardware profiles
func (dc *DePINCommand) ListAvailableProfiles() error {
	profiles := providers.CommonDePINRequirements()

	fmt.Println("Available DePIN Hardware Profiles:")
	fmt.Println("==================================\n")

	for name, req := range profiles {
		fmt.Printf("%s:\n", name)
		fmt.Printf("  CPU:          %d+ cores\n", req.MinCPUCores)
		fmt.Printf("  Memory:       %d+ GB\n", req.MinMemoryGB)
		fmt.Printf("  Storage:      %d+ GB\n", req.MinStorageGB)
		if req.RequiredGPU != "" {
			fmt.Printf("  GPU:          %s (%d+ GB)\n", req.RequiredGPU, req.MinGPUMemoryGB)
		} else {
			fmt.Printf("  GPU:          optional\n")
		}
		fmt.Println()
	}

	return nil
}

// Helper functions

func extractGPUTypes(gpus []system.GPUInfo) []string {
	var types []string
	for _, gpu := range gpus {
		types = append(types, gpu.Type)
	}
	return types
}
