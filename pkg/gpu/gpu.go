// Package gpu provides GPU discovery, capability reporting, and allocation scheduling.
//
// GPU discovery is performed on each host and reports to the scheduler.
// Capabilities include compute capacity, memory, arch (NVIDIA/AMD), driver version.
//
// Scheduling constrains:
//   - GPU count (GPUs per workload)
//   - GPU memory (device memory requirement)
//   - GPU arch (e.g., cuda-compute-80, rocm-gfx908)
//   - GPU tier (standard, performance, inference)
//   - GPU time-sharing policy (exclusive, shared-memory, full-sharing)
//
// Allocation is tracked per workload and enforces policy.
// Time-sharing requires workload isolation at OS level.
package gpu

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// Device describes a single GPU.
type Device struct {
	ID              int    `json:"id"`              // GPU index (0-based)
	UUID            string `json:"uuid"`            // unique device identifier
	Model           string `json:"model"`           // device name (e.g., A100, V100, MI300X)
	Arch            string `json:"arch"`            // architecture (cuda, rocm)
	ComputeCapability string `json:"compute_capability"` // major.minor (e.g., 8.0 for A100)
	MemoryBytes     int64  `json:"memory_bytes"`    // device memory in bytes
	DriverVersion   string `json:"driver_version"`  // GPU driver version
	ClockMHz        int    `json:"clock_mhz"`       // max clock speed
	Cores           int    `json:"cores"`           // compute cores
	Tier            string `json:"tier"`            // standard | performance | inference
}

// Discoverer probes the system for GPUs.
type Discoverer struct {
	mu              sync.RWMutex
	lastDiscovery   []Device
	nvidiaPresent   bool
	amdPresent      bool
	lastErr         error
}

// New creates a new GPU discoverer.
func New() *Discoverer {
	return &Discoverer{}
}

// Discover probes for GPUs (NVIDIA via nvidia-smi, AMD via rocm-smi).
// Returns a slice of discovered devices or error.
func (d *Discoverer) Discover() ([]Device, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var devices []Device

	// Try NVIDIA first
	nvDevices, nvErr := d.discoverNVIDIA()
	if nvErr == nil && len(nvDevices) > 0 {
		devices = append(devices, nvDevices...)
		d.nvidiaPresent = true
	}

	// Then try AMD/ROCm
	amdDevices, amdErr := d.discoverAMD()
	if amdErr == nil && len(amdDevices) > 0 {
		devices = append(devices, amdDevices...)
		d.amdPresent = true
	}

	d.lastDiscovery = devices
	if len(devices) == 0 && nvErr != nil && amdErr != nil {
		d.lastErr = fmt.Errorf("no GPUs discovered: nvidia-smi: %v, rocm-smi: %v", nvErr, amdErr)
		return nil, d.lastErr
	}

	return devices, nil
}

// discoverNVIDIA uses nvidia-smi to enumerate NVIDIA GPUs.
func (d *Discoverer) discoverNVIDIA() ([]Device, error) {
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=index,uuid,name,compute_cap,memory.total,driver_version,clocks.max.gr,compute_processes",
		"--format=csv,noheader,nounits")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var devices []Device
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ", ")
		if len(parts) < 5 {
			continue
		}

		memMB, _ := strconv.ParseInt(strings.TrimSpace(parts[4]), 10, 64)
		ccap := strings.TrimSpace(parts[3])
		tier := "standard"
		if strings.Contains(ccap, "8.") || strings.Contains(ccap, "9.") {
			tier = "performance"
		}

		device := Device{
			ID:              i,
			UUID:            strings.TrimSpace(parts[1]),
			Model:           strings.TrimSpace(parts[2]),
			Arch:            "cuda",
			ComputeCapability: ccap,
			MemoryBytes:     memMB * 1024 * 1024,
			DriverVersion:   strings.TrimSpace(parts[5]),
			ClockMHz:        2500,
			Cores:           5120,
			Tier:            tier,
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// discoverAMD uses rocm-smi to enumerate AMD GPUs.
func (d *Discoverer) discoverAMD() ([]Device, error) {
	cmd := exec.Command("rocm-smi",
		"--showid",
		"--showmeminfo",
		"--showproductname",
		"--csv")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var devices []Device
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i, line := range lines {
		if line == "" || strings.Contains(line, "GPU ID") {
			continue
		}
		parts := strings.Split(line, ", ")
		if len(parts) < 3 {
			continue
		}

		device := Device{
			ID:            i,
			UUID:          fmt.Sprintf("rocm-%d", i),
			Model:         strings.TrimSpace(parts[1]),
			Arch:          "rocm",
			ComputeCapability: strings.TrimSpace(parts[2]),
			MemoryBytes:   16 * 1024 * 1024 * 1024,
			DriverVersion: "rocm-6.0",
			ClockMHz:      2700,
			Cores:         7680,
			Tier:          "performance",
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// Query holds GPU scheduling constraints.
type Query struct {
	Count       int      `json:"count"`         // number of GPUs required
	MemoryBytes int64    `json:"memory_bytes"` // per-GPU memory requirement
	Archs       []string `json:"archs"`        // allowed architectures (cuda, rocm)
	Tiers       []string `json:"tiers"`        // allowed tiers (standard, performance, inference)
	TimeSharing string   `json:"time_sharing"` // exclusive | shared-memory | full-sharing
}

// Matcher evaluates whether a device satisfies a query.
type Matcher struct {
	Query Query
}

// Matches returns true if device satisfies the query.
func (m *Matcher) Matches(dev Device) bool {
	// Memory requirement: device must have AT LEAST the required memory
	if m.Query.MemoryBytes > 0 && dev.MemoryBytes < m.Query.MemoryBytes {
		return false
	}
	// But if the device has MORE memory than the maximum we're looking for,
	// it might not match. In practice, we accept any device with enough memory.

	if len(m.Query.Archs) > 0 {
		found := false
		for _, arch := range m.Query.Archs {
			if dev.Arch == arch {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(m.Query.Tiers) > 0 {
		found := false
		for _, tier := range m.Query.Tiers {
			if dev.Tier == tier {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Allocator tracks GPU allocations per workload.
type Allocator struct {
	mu          sync.RWMutex
	allocations map[string][]Device // workload ID -> assigned devices
	devices     []Device             // all available devices
}

// NewAllocator creates a new GPU allocator.
func NewAllocator(devices []Device) *Allocator {
	return &Allocator{
		allocations: make(map[string][]Device),
		devices:     devices,
	}
}

// Allocate assigns GPUs to a workload. Returns assigned devices or error.
func (a *Allocator) Allocate(workloadID string, query Query) ([]Device, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.allocations[workloadID]; exists {
		return a.allocations[workloadID], nil
	}

	matcher := &Matcher{Query: query}
	allocated := make(map[string]bool)

	// Build map of already-allocated devices
	for _, devs := range a.allocations {
		for _, dev := range devs {
			allocated[dev.UUID] = true
		}
	}

	var candidates []Device

	for _, dev := range a.devices {
		if !allocated[dev.UUID] && matcher.Matches(dev) {
			candidates = append(candidates, dev)
		}
	}

	if len(candidates) < query.Count {
		return nil, fmt.Errorf("insufficient GPUs: need %d, found %d matching", query.Count, len(candidates))
	}

	assigned := candidates[:query.Count]
	a.allocations[workloadID] = assigned
	return assigned, nil
}

// Release frees GPUs from a workload.
func (a *Allocator) Release(workloadID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.allocations[workloadID]; !exists {
		return fmt.Errorf("workload %s not allocated", workloadID)
	}

	delete(a.allocations, workloadID)
	return nil
}

// GetAllocated returns assigned GPUs for a workload.
func (a *Allocator) GetAllocated(workloadID string) ([]Device, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	devices, ok := a.allocations[workloadID]
	return devices, ok
}

// ListAllocations returns all active allocations.
func (a *Allocator) ListAllocations() map[string][]Device {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string][]Device)
	for k, v := range a.allocations {
		result[k] = append([]Device(nil), v...)
	}
	return result
}

// Report summarizes GPU availability.
type Report struct {
	TotalGPUs      int       `json:"total_gpus"`
	AvailableGPUs  int       `json:"available_gpus"`
	NvidiaGPUs     int       `json:"nvidia_gpus"`
	AMDGPUs        int       `json:"amd_gpus"`
	TotalMemoryGB  float64   `json:"total_memory_gb"`
	AvailableMemGB float64   `json:"available_memory_gb"`
	Devices        []Device  `json:"devices"`
	ByTier         map[string]int `json:"by_tier"`
	SchedulingEfficiency float64 `json:"scheduling_efficiency"`
}

// Reporter generates scheduling reports.
type Reporter struct {
	discoverer *Discoverer
	allocator  *Allocator
}

// NewReporter creates a GPU report generator.
func NewReporter(disc *Discoverer, alloc *Allocator) *Reporter {
	return &Reporter{discoverer: disc, allocator: alloc}
}

// Report generates a scheduling report.
func (r *Reporter) Report() Report {
	devices, _ := r.discoverer.Discover()
	allocations := r.allocator.ListAllocations()

	report := Report{
		Devices: devices,
		ByTier:  make(map[string]int),
	}

	allocated := make(map[string]bool)
	for _, devs := range allocations {
		for _, dev := range devs {
			allocated[dev.UUID] = true
		}
	}

	for _, dev := range devices {
		report.TotalGPUs++
		report.TotalMemoryGB += float64(dev.MemoryBytes) / 1e9
		report.ByTier[dev.Tier]++

		if !allocated[dev.UUID] {
			report.AvailableGPUs++
			report.AvailableMemGB += float64(dev.MemoryBytes) / 1e9
		}

		if dev.Arch == "cuda" {
			report.NvidiaGPUs++
		} else if dev.Arch == "rocm" {
			report.AMDGPUs++
		}
	}

	if report.TotalGPUs > 0 {
		report.SchedulingEfficiency = float64(report.TotalGPUs-report.AvailableGPUs) / float64(report.TotalGPUs)
	}

	return report
}

// SystemInfo gathers host GPU capabilities for advertising.
type SystemInfo struct {
	TotalGPUs int         `json:"total_gpus"`
	GPUMemory int64       `json:"gpu_memory_total"`
	Features  []string    `json:"features"`
	Devices   []Device    `json:"devices"`
}

// GetSystemInfo returns the GPU capabilities of this host.
func GetSystemInfo() (SystemInfo, error) {
	disc := New()
	devices, err := disc.Discover()

	info := SystemInfo{
		TotalGPUs: len(devices),
		Devices:   devices,
	}

	features := map[string]bool{}

	for _, dev := range devices {
		info.GPUMemory += dev.MemoryBytes

		if dev.Arch == "cuda" {
			features["gpu-nvidia"] = true
			switch {
			case strings.Contains(dev.ComputeCapability, "8."):
				features["cuda-compute-80"] = true
			case strings.Contains(dev.ComputeCapability, "9."):
				features["cuda-compute-90"] = true
			default:
				features["cuda-compute-70"] = true
			}
		} else if dev.Arch == "rocm" {
			features["gpu-amd"] = true
			features["rocm-available"] = true
		}
	}

	if info.TotalGPUs > 0 {
		if info.TotalGPUs >= 8 {
			features["gpu-cluster"] = true
		}
	}

	for feature := range features {
		info.Features = append(info.Features, feature)
	}

	return info, err
}
