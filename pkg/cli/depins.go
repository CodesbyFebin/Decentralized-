package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

	// Build requirement for the profile
	req := profileRequirement(profileName)
	if req == nil {
		return fmt.Errorf("unknown profile '%s'", profileName)
	}

	// Convert to simple profile format
	simpleProfile := &providers.SimpleHardwareProfile{
		CPUCores:     hwProfile.CPU.Cores,
		CPUModel:     hwProfile.CPU.Model,
		MemoryBytes:  hwProfile.Memory.TotalBytes,
		StorageBytes: hwProfile.Disk.TotalBytes,
		NetworkCount: len(hwProfile.Network),
	}
	for _, gpu := range hwProfile.GPU {
		simpleProfile.GPUs = append(simpleProfile.GPUs, gpu.Type)
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

// ListAvailableProfiles shows all available DePIN hardware profiles from control plane
func (dc *DePINCommand) ListAvailableProfiles() error {
	if dc.operator == nil || len(dc.operator.Cfg.Endpoints) == 0 {
		return fmt.Errorf("control plane not configured")
	}

	url := fmt.Sprintf("%s://%s/api/v1/depins/profiles", dc.operator.scheme, dc.operator.Cfg.Endpoints[0])
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch profiles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to fetch profiles (status %d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("Available DePIN Hardware Profiles:")
	fmt.Println("==================================\n")
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))

	return nil
}

// GetNodeHardware retrieves a node's hardware profile from the control plane
func (dc *DePINCommand) GetNodeHardware(nodeID string) error {
	if dc.operator == nil || len(dc.operator.Cfg.Endpoints) == 0 {
		return fmt.Errorf("control plane not configured")
	}

	url := fmt.Sprintf("%s://%s/api/v1/nodes/%s/hardware", dc.operator.scheme, dc.operator.Cfg.Endpoints[0], nodeID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch hardware: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to fetch hardware (status %d): %s", resp.StatusCode, string(body))
	}

	var hwResp interface{}
	if err := json.NewDecoder(resp.Body).Decode(&hwResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Hardware Profile for Node %s:\n", nodeID)
	data, _ := json.MarshalIndent(hwResp, "", "  ")
	fmt.Println(string(data))

	return nil
}

// CheckNodeCompatibility checks a node's compatibility with a profile via the control plane
func (dc *DePINCommand) CheckNodeCompatibility(nodeID, profileName string) error {
	if dc.operator == nil || len(dc.operator.Cfg.Endpoints) == 0 {
		return fmt.Errorf("control plane not configured")
	}

	url := fmt.Sprintf("%s://%s/api/v1/nodes/%s/compatibility?profile=%s", dc.operator.scheme, dc.operator.Cfg.Endpoints[0], nodeID, profileName)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to check compatibility: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to check compatibility (status %d): %s", resp.StatusCode, string(body))
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Compatibility Report for Node %s (Profile: %s):\n", nodeID, profileName)
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))

	return nil
}

// CreateOwnerReserve creates an owner reserve for a node via the control plane
func (dc *DePINCommand) CreateOwnerReserve(nodeID, ownerID string) error {
	if dc.operator == nil || len(dc.operator.Cfg.Endpoints) == 0 {
		return fmt.Errorf("control plane not configured")
	}

	reserve := map[string]string{
		"owner_id": ownerID,
		"host":     nodeID,
	}

	body, _ := json.Marshal(reserve)
	url := fmt.Sprintf("%s://%s/api/v1/nodes/%s/reserve", dc.operator.scheme, dc.operator.Cfg.Endpoints[0], nodeID)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create reserve: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create reserve (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Owner Reserve for %s:\n", nodeID)
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))

	return nil
}

// GetNodeReserve retrieves the owner reserve status for a node
func (dc *DePINCommand) GetNodeReserve(nodeID string) error {
	if dc.operator == nil || len(dc.operator.Cfg.Endpoints) == 0 {
		return fmt.Errorf("control plane not configured")
	}

	url := fmt.Sprintf("%s://%s/api/v1/nodes/%s/reserve", dc.operator.scheme, dc.operator.Cfg.Endpoints[0], nodeID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch reserve: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to fetch reserve (status %d): %s", resp.StatusCode, string(body))
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Owner Reserve Status for Node %s:\n", nodeID)
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))

	return nil
}

// Helper functions

func profileRequirement(name string) *providers.HardwareRequirement {
	switch name {
	case "compute-light":
		return &providers.HardwareRequirement{
			MinCPUCores:  2,
			MinMemoryGB:  4,
			MinStorageGB: 50,
		}
	case "compute-standard":
		return &providers.HardwareRequirement{
			MinCPUCores:  4,
			MinMemoryGB:  8,
			MinStorageGB: 100,
		}
	case "compute-heavy":
		return &providers.HardwareRequirement{
			MinCPUCores:    8,
			MinMemoryGB:    16,
			MinStorageGB:   500,
			RequiredGPU:    "nvidia",
			MinGPUMemoryGB: 6,
		}
	case "gpu-optimized":
		return &providers.HardwareRequirement{
			MinCPUCores:    16,
			MinMemoryGB:    32,
			MinStorageGB:   1000,
			RequiredGPU:    "nvidia",
			MinGPUMemoryGB: 24,
		}
	case "storage-node":
		return &providers.HardwareRequirement{
			MinCPUCores:  2,
			MinMemoryGB:  8,
			MinStorageGB: 5120,
		}
	}
	return nil
}

func extractGPUTypes(gpus []system.GPUInfo) []string {
	var types []string
	for _, gpu := range gpus {
		types = append(types, gpu.Type)
	}
	return types
}
