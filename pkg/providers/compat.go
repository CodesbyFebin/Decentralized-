package providers

import (
	"context"
	"fmt"
	"strings"
)

// HardwareRequirement represents a hardware requirement for a DePIN project
type HardwareRequirement struct {
	MinCPUCores    int
	MinMemoryGB    int
	MinStorageGB   int
	RequiredGPU    string // nvidia, amd, intel, or empty for none
	MinGPUMemoryGB int
	NetworkSpeed   int    // Mbps
}

// CompatibilityScore represents the result of a hardware compatibility check
type CompatibilityScore struct {
	Overall      float64                    `json:"overall"`
	CPU          float64                    `json:"cpu"`
	Memory       float64                    `json:"memory"`
	Storage      float64                    `json:"storage"`
	GPU          float64                    `json:"gpu"`
	Network      float64                    `json:"network"`
	Issues       []string                   `json:"issues"`
	Warnings     []string                   `json:"warnings"`
	Metadata     map[string]interface{}     `json:"metadata"`
}

// HardwareProfile mirrors the structure from pkg/system for compatibility checking
type SimpleHardwareProfile struct {
	CPUCores    int
	CPUModel    string
	MemoryBytes int64
	GPUs        []string // GPU types detected
	StorageBytes int64
	NetworkCount int
}

// CompatibilityChecker validates hardware against requirements
type CompatibilityChecker struct{}

// NewCompatibilityChecker creates a new compatibility checker
func NewCompatibilityChecker() *CompatibilityChecker {
	return &CompatibilityChecker{}
}

// CheckCompatibility evaluates hardware against project requirements
func (cc *CompatibilityChecker) CheckCompatibility(
	ctx context.Context,
	hardware *SimpleHardwareProfile,
	requirement *HardwareRequirement,
) *CompatibilityScore {
	score := &CompatibilityScore{
		Issues:   []string{},
		Warnings: []string{},
		Metadata: make(map[string]interface{}),
	}

	// Check CPU
	score.CPU = cc.scoreCPU(hardware.CPUCores, requirement.MinCPUCores, score)

	// Check Memory
	memoryGB := hardware.MemoryBytes / (1024 * 1024 * 1024)
	score.Memory = cc.scoreMemory(memoryGB, int64(requirement.MinMemoryGB), score)

	// Check Storage
	storageGB := hardware.StorageBytes / (1024 * 1024 * 1024)
	score.Storage = cc.scoreStorage(storageGB, int64(requirement.MinStorageGB), score)

	// Check GPU
	score.GPU = cc.scoreGPU(hardware.GPUs, requirement.RequiredGPU, requirement.MinGPUMemoryGB, score)

	// Check Network
	score.Network = cc.scoreNetwork(hardware.NetworkCount, score)

	// Calculate overall score (average of all components, weighted by availability)
	weights := 0.0
	total := 0.0

	if requirement.MinCPUCores > 0 {
		total += score.CPU
		weights += 1.0
	}
	if requirement.MinMemoryGB > 0 {
		total += score.Memory
		weights += 1.0
	}
	if requirement.MinStorageGB > 0 {
		total += score.Storage
		weights += 1.0
	}
	if requirement.RequiredGPU != "" {
		total += score.GPU
		weights += 1.0
	}
	if requirement.NetworkSpeed > 0 {
		total += score.Network
		weights += 1.0
	}

	if weights > 0 {
		score.Overall = total / weights
	} else {
		score.Overall = 0.5 // Default neutral score if no requirements specified
	}

	// Store hardware details in metadata
	score.Metadata["detected_cpu_cores"] = hardware.CPUCores
	score.Metadata["detected_memory_gb"] = memoryGB
	score.Metadata["detected_storage_gb"] = storageGB
	score.Metadata["detected_gpus"] = hardware.GPUs
	score.Metadata["detected_networks"] = hardware.NetworkCount

	return score
}

// scoreCPU evaluates CPU compatibility
func (cc *CompatibilityChecker) scoreCPU(detected, required int, score *CompatibilityScore) float64 {
	if required == 0 {
		return 1.0 // No requirement
	}

	if detected < required {
		score.Issues = append(score.Issues,
			fmt.Sprintf("CPU: only %d cores available, %d required", detected, required))
		return float64(detected) / float64(required)
	}

	if detected == required {
		return 1.0
	}

	// More cores than required is fine
	return 1.0
}

// scoreMemory evaluates memory compatibility
func (cc *CompatibilityChecker) scoreMemory(detected, required int64, score *CompatibilityScore) float64 {
	if required == 0 {
		return 1.0 // No requirement
	}

	if detected < required {
		score.Issues = append(score.Issues,
			fmt.Sprintf("Memory: only %d GB available, %d GB required", detected, required))
		return float64(detected) / float64(required)
	}

	if detected == required {
		return 1.0
	}

	// More memory than required is fine
	return 1.0
}

// scoreStorage evaluates storage compatibility
func (cc *CompatibilityChecker) scoreStorage(detected, required int64, score *CompatibilityScore) float64 {
	if required == 0 {
		return 1.0 // No requirement
	}

	if detected < required {
		score.Issues = append(score.Issues,
			fmt.Sprintf("Storage: only %d GB available, %d GB required", detected, required))
		return float64(detected) / float64(required)
	}

	if detected == required {
		return 1.0
	}

	// More storage than required is fine
	return 1.0
}

// scoreGPU evaluates GPU compatibility
func (cc *CompatibilityChecker) scoreGPU(detected []string, required string, minMemory int, score *CompatibilityScore) float64 {
	if required == "" {
		return 1.0 // No GPU requirement
	}

	if len(detected) == 0 {
		score.Issues = append(score.Issues,
			fmt.Sprintf("GPU: %s GPU required but none detected", required))
		return 0.0
	}

	// Check if required GPU type is present
	found := false
	for _, gpu := range detected {
		if strings.Contains(strings.ToLower(gpu), strings.ToLower(required)) {
			found = true
			break
		}
	}

	if !found {
		score.Warnings = append(score.Warnings,
			fmt.Sprintf("GPU: %s GPU required but detected %v", required, detected))
		return 0.5 // Partial compatibility
	}

	// If memory requirement specified, note it (full check would need actual GPU memory detection)
	if minMemory > 0 {
		score.Metadata["gpu_memory_check_required"] = true
	}

	return 1.0
}

// scoreNetwork evaluates network compatibility
func (cc *CompatibilityChecker) scoreNetwork(detected int, score *CompatibilityScore) float64 {
	if detected == 0 {
		score.Issues = append(score.Issues, "Network: no network interfaces detected")
		return 0.0
	}

	if detected < 1 {
		score.Warnings = append(score.Warnings, "Network: only 1 interface, may limit redundancy")
		return 0.7
	}

	// Multiple interfaces is good
	return 1.0
}

// GetCompatibilityLevel returns a human-readable compatibility level
func (score *CompatibilityScore) GetCompatibilityLevel() string {
	switch {
	case score.Overall >= 0.95:
		return "EXCELLENT"
	case score.Overall >= 0.85:
		return "GOOD"
	case score.Overall >= 0.7:
		return "ADEQUATE"
	case score.Overall >= 0.5:
		return "MARGINAL"
	case len(score.Issues) > 0:
		return "INCOMPATIBLE"
	default:
		return "UNKNOWN"
	}
}

// ConvertProbeToSimpleProfile converts a full hardware profile to a simple one for compatibility checking
// This bridges between pkg/system probed data and compatibility scoring
type ProbeHardwareInfo struct {
	CPUCores     int
	CPUModel     string
	MemoryBytes  int64
	GPUTypes     []string
	StorageBytes int64
	NetworkIfaces int
}

// ToSimpleProfile converts probe info to compatibility checker format
func (p *ProbeHardwareInfo) ToSimpleProfile() *SimpleHardwareProfile {
	return &SimpleHardwareProfile{
		CPUCores:     p.CPUCores,
		CPUModel:     p.CPUModel,
		MemoryBytes:  p.MemoryBytes,
		GPUs:         p.GPUTypes,
		StorageBytes: p.StorageBytes,
		NetworkCount: p.NetworkIfaces,
	}
}

// CommonDePINRequirements returns typical hardware requirements for DePIN projects
func CommonDePINRequirements() map[string]*HardwareRequirement {
	return map[string]*HardwareRequirement{
		"compute-light": {
			MinCPUCores:  2,
			MinMemoryGB:  4,
			MinStorageGB: 50,
		},
		"compute-standard": {
			MinCPUCores:  4,
			MinMemoryGB:  8,
			MinStorageGB: 100,
		},
		"compute-heavy": {
			MinCPUCores:    8,
			MinMemoryGB:    16,
			MinStorageGB:   500,
			RequiredGPU:    "nvidia",
			MinGPUMemoryGB: 6,
		},
		"gpu-optimized": {
			MinCPUCores:    16,
			MinMemoryGB:    32,
			MinStorageGB:   1000,
			RequiredGPU:    "nvidia",
			MinGPUMemoryGB: 24,
		},
		"storage-node": {
			MinCPUCores:  2,
			MinMemoryGB:  8,
			MinStorageGB: 5000,
		},
	}
}
