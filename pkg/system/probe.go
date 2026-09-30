package system

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"decentralized/pkg/providers"
)

// HardwareProfile represents the detected hardware capabilities of a node
type HardwareProfile struct {
	CPU      CPUInfo       `json:"cpu"`
	Memory   MemoryInfo    `json:"memory"`
	GPU      []GPUInfo     `json:"gpu"`
	Disk     DiskInfo      `json:"disk"`
	Network  []NetworkInfo `json:"network"`
	Hostname string        `json:"hostname"`
	Arch     string        `json:"arch"`
	OS       string        `json:"os"`
	Probed   time.Time     `json:"probed"`
}

// CPUInfo represents CPU capabilities
type CPUInfo struct {
	Cores       int      `json:"cores"`
	Model       string   `json:"model"`
	Frequency   float64  `json:"frequency_ghz"`
	Flags       []string `json:"flags"`
	Threads     int      `json:"threads"`
	TurboBoost  bool     `json:"turbo_boost"`
}

// MemoryInfo represents memory configuration
type MemoryInfo struct {
	TotalBytes     int64 `json:"total_bytes"`
	AvailableBytes int64 `json:"available_bytes"`
	UsedBytes      int64 `json:"used_bytes"`
}

// GPUInfo represents GPU capabilities
type GPUInfo struct {
	Name       string `json:"name"`
	MemoryBytes int64  `json:"memory_bytes"`
	Type       string `json:"type"` // nvidia, amd, intel, etc.
	Index      int    `json:"index"`
	Status     string `json:"status"`
}

// DiskInfo represents disk configuration
type DiskInfo struct {
	TotalBytes     int64  `json:"total_bytes"`
	AvailableBytes int64  `json:"available_bytes"`
	UsedBytes      int64  `json:"used_bytes"`
	Filesystem     string `json:"filesystem"`
	MountPoint     string `json:"mount_point"`
}

// NetworkInfo represents network interface details
type NetworkInfo struct {
	Name       string   `json:"name"`
	MacAddress string   `json:"mac_address"`
	IPv4       []string `json:"ipv4"`
	IPv6       []string `json:"ipv6"`
	Speed      int      `json:"speed_mbps"`
	Status     string   `json:"status"`
	MTU        int      `json:"mtu"`
}

// Prober handles hardware detection
type Prober struct{}

// NewProber creates a new hardware prober
func NewProber() *Prober {
	return &Prober{}
}

// Probe detects the hardware profile of the current node
func (p *Prober) Probe(ctx context.Context) (*HardwareProfile, error) {
	profile := &HardwareProfile{
		Probed: time.Now(),
	}

	// Detect OS and architecture
	profile.OS = getOS()
	profile.Arch = getArch()

	// Detect hostname
	hostname, _ := os.Hostname()
	profile.Hostname = hostname

	// Detect CPU
	if cpuInfo, err := p.probeCPU(); err == nil {
		profile.CPU = cpuInfo
	}

	// Detect memory
	if memInfo, err := p.probeMemory(); err == nil {
		profile.Memory = memInfo
	}

	// Detect GPUs
	if gpus, err := p.probeGPU(); err == nil {
		profile.GPU = gpus
	}

	// Detect disk
	if diskInfo, err := p.probeDisk(); err == nil {
		profile.Disk = diskInfo
	}

	// Detect network
	if netInfo, err := p.probeNetwork(); err == nil {
		profile.Network = netInfo
	}

	return profile, nil
}

// probeCPU detects CPU information
func (p *Prober) probeCPU() (CPUInfo, error) {
	info := CPUInfo{}

	// Read /proc/cpuinfo
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return info, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	seenProcessors := make(map[int]bool)
	flags := make(map[string]bool)
	var model, freq string

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "processor":
			if procID, err := strconv.Atoi(value); err == nil {
				seenProcessors[procID] = true
			}
		case "model name":
			model = value
		case "cpu MHz":
			freq = value
		case "flags":
			for _, flag := range strings.Fields(value) {
				flags[flag] = true
			}
		}
	}

	info.Cores = len(seenProcessors)
	info.Model = model

	// Parse frequency
	if freq != "" {
		if f, err := strconv.ParseFloat(freq, 64); err == nil {
			info.Frequency = f / 1000.0 // Convert MHz to GHz
		}
	}

	// Check for turbo boost
	info.TurboBoost = flags["lm"] && flags["ht"]

	// Extract flags
	for flag := range flags {
		info.Flags = append(info.Flags, flag)
	}

	// Threads (usually 2x cores for hyperthreaded CPUs)
	info.Threads = info.Cores * 2
	if !flags["ht"] {
		info.Threads = info.Cores
	}

	return info, nil
}

// probeMemory detects memory information
func (p *Prober) probeMemory() (MemoryInfo, error) {
	info := MemoryInfo{}

	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return info, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := parts[1]

		// Parse value in KB and convert to bytes
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			bytes := v * 1024

			switch key {
			case "MemTotal:":
				info.TotalBytes = bytes
			case "MemAvailable:":
				info.AvailableBytes = bytes
			}
		}
	}

	info.UsedBytes = info.TotalBytes - info.AvailableBytes
	return info, nil
}

// probeGPU attempts to detect NVIDIA, AMD, or Intel GPUs
func (p *Prober) probeGPU() ([]GPUInfo, error) {
	var gpus []GPUInfo

	// Try NVIDIA first
	nvidiaGPUs := p.probeNvidiaGPU()
	gpus = append(gpus, nvidiaGPUs...)

	// Try AMD
	amdGPUs := p.probeAMDGPU()
	gpus = append(gpus, amdGPUs...)

	// Try Intel Arc
	intelGPUs := p.probeIntelGPU()
	gpus = append(gpus, intelGPUs...)

	return gpus, nil
}

// probeNvidiaGPU detects NVIDIA GPUs via nvidia-smi
func (p *Prober) probeNvidiaGPU() []GPUInfo {
	var gpus []GPUInfo

	cmd := exec.Command("nvidia-smi", "--query-gpu=index,name,memory.total", "--format=csv,noheader")
	output, err := cmd.Output()
	if err != nil {
		return gpus
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

		idx, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		name := strings.TrimSpace(parts[1])
		memStr := strings.TrimSpace(parts[2])

		// Parse memory (usually in MiB)
		memBytes := int64(0)
		if mem, err := strconv.ParseInt(strings.Fields(memStr)[0], 10, 64); err == nil {
			memBytes = mem * 1024 * 1024 // Convert MiB to bytes
		}

		gpus = append(gpus, GPUInfo{
			Name:        name,
			MemoryBytes: memBytes,
			Type:        "nvidia",
			Index:       idx,
			Status:      "available",
		})
	}

	return gpus
}

// probeAMDGPU detects AMD GPUs via rocm-smi
func (p *Prober) probeAMDGPU() []GPUInfo {
	var gpus []GPUInfo

	cmd := exec.Command("rocm-smi", "--showid", "--showmeminfo")
	output, err := cmd.Output()
	if err != nil {
		return gpus
	}

	// Parse rocm-smi output
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "GPU") {
			continue
		}

		gpus = append(gpus, GPUInfo{
			Name:   "AMD Radeon GPU",
			Type:   "amd",
			Status: "available",
		})
	}

	return gpus
}

// probeIntelGPU detects Intel Arc GPUs
func (p *Prober) probeIntelGPU() []GPUInfo {
	var gpus []GPUInfo

	cmd := exec.Command("clinfo")
	output, err := cmd.Output()
	if err != nil {
		return gpus
	}

	if strings.Contains(string(output), "Intel") {
		gpus = append(gpus, GPUInfo{
			Name:   "Intel Arc GPU",
			Type:   "intel",
			Status: "available",
		})
	}

	return gpus
}

// probeDisk detects disk information
func (p *Prober) probeDisk() (DiskInfo, error) {
	info := DiskInfo{
		MountPoint: "/",
	}

	cmd := exec.Command("df", "-B1", "/")
	output, err := cmd.Output()
	if err != nil {
		return info, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum == 1 {
			// Header line
			continue
		}

		parts := strings.Fields(scanner.Text())
		if len(parts) < 4 {
			continue
		}

		info.Filesystem = parts[0]
		if total, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
			info.TotalBytes = total
		}
		if used, err := strconv.ParseInt(parts[2], 10, 64); err == nil {
			info.UsedBytes = used
		}
		if avail, err := strconv.ParseInt(parts[3], 10, 64); err == nil {
			info.AvailableBytes = avail
		}
	}

	return info, nil
}

// probeNetwork detects network interfaces
func (p *Prober) probeNetwork() ([]NetworkInfo, error) {
	var networks []NetworkInfo

	// Parse /proc/net/dev for basic interface info
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return networks, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			// Skip headers
			continue
		}

		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		if name == "lo" {
			continue // Skip loopback
		}

		networks = append(networks, NetworkInfo{
			Name:   name,
			Status: "up",
		})
	}

	// Try to get detailed interface info via ip command
	for i := range networks {
		p.enrichNetworkInfo(&networks[i])
	}

	return networks, nil
}

// enrichNetworkInfo adds detailed information to a network interface
func (p *Prober) enrichNetworkInfo(net *NetworkInfo) {
	// Get MAC address
	if macCmd, err := exec.Command("ip", "link", "show", net.Name).Output(); err == nil {
		for _, line := range strings.Split(string(macCmd), "\n") {
			if strings.Contains(line, "link/ether") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					net.MacAddress = parts[1]
				}
			}
		}
	}

	// Get IP addresses
	if addrCmd, err := exec.Command("ip", "addr", "show", net.Name).Output(); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(addrCmd)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "inet ") && !strings.Contains(line, "inet6") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					net.IPv4 = append(net.IPv4, parts[1])
				}
			} else if strings.Contains(line, "inet6") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					net.IPv6 = append(net.IPv6, parts[1])
				}
			}
		}
	}
}

// ConvertToCapabilities converts a hardware profile to provider capabilities
func (p *Prober) ConvertToCapabilities(profile *HardwareProfile) []providers.Capability {
	var caps []providers.Capability

	// CPU capabilities
	if profile.CPU.Cores > 0 {
		caps = append(caps, providers.Capability{
			Name:  fmt.Sprintf("cpu-%d-core", profile.CPU.Cores),
			Level: providers.CapabilityLevelFull,
			Description: fmt.Sprintf("%d-core CPU (%s, %.2f GHz)",
				profile.CPU.Cores, profile.CPU.Model, profile.CPU.Frequency),
		})
	}

	// Memory capability
	if profile.Memory.TotalBytes > 0 {
		gbTotal := profile.Memory.TotalBytes / (1024 * 1024 * 1024)
		caps = append(caps, providers.Capability{
			Name:        fmt.Sprintf("memory-%dgb", gbTotal),
			Level:       providers.CapabilityLevelFull,
			Description: fmt.Sprintf("%d GB RAM", gbTotal),
		})
	}

	// GPU capabilities
	for _, gpu := range profile.GPU {
		gpuName := strings.ToLower(strings.ReplaceAll(gpu.Name, " ", "-"))
		caps = append(caps, providers.Capability{
			Name:        fmt.Sprintf("gpu-%s", gpuName),
			Level:       providers.CapabilityLevelFull,
			Description: fmt.Sprintf("%s (%s, %d GB)", gpu.Name, gpu.Type, gpu.MemoryBytes/(1024*1024*1024)),
		})
	}

	// Storage capability
	if profile.Disk.TotalBytes > 0 {
		gbTotal := profile.Disk.TotalBytes / (1024 * 1024 * 1024)
		caps = append(caps, providers.Capability{
			Name:        fmt.Sprintf("storage-%dgb", gbTotal),
			Level:       providers.CapabilityLevelFull,
			Description: fmt.Sprintf("%d GB storage available", gbTotal),
		})
	}

	// Network capabilities
	if len(profile.Network) > 0 {
		caps = append(caps, providers.Capability{
			Name:        fmt.Sprintf("network-%d-interfaces", len(profile.Network)),
			Level:       providers.CapabilityLevelFull,
			Description: fmt.Sprintf("%d network interfaces", len(profile.Network)),
		})
	}

	return caps
}

// getOS returns the operating system
func getOS() string {
	switch {
	case strings.Contains(os.Getenv("OSTYPE"), "linux"):
		return "linux"
	case strings.Contains(os.Getenv("OSTYPE"), "darwin"):
		return "darwin"
	default:
		return "unknown"
	}
}

// getArch returns the system architecture
func getArch() string {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "unknown"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "flags") {
			if strings.Contains(line, "lm") {
				return "x86_64"
			}
			return "x86"
		}
	}

	return "unknown"
}
