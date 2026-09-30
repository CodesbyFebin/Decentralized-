package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RuntimeBackend identifies the infrastructure layer executing resources.
type RuntimeBackend string

const (
	BACKEND_UNKNOWN    RuntimeBackend = "UNKNOWN"
	BACKEND_NATIVE     RuntimeBackend = "NATIVE"      // Bare metal or direct process execution
	BACKEND_CONTAINER  RuntimeBackend = "CONTAINER"   // Docker, containerd, cri-o
	BACKEND_QEMU_VM    RuntimeBackend = "QEMU_VM"     // QEMU/KVM hypervisor
	BACKEND_HYPERV     RuntimeBackend = "HYPERV"      // Hyper-V
	BACKEND_VMWARE     RuntimeBackend = "VMWARE"      // VMware vSphere
	BACKEND_KUBERNETES RuntimeBackend = "KUBERNETES" // Kubernetes pods
	BACKEND_CLOUD      RuntimeBackend = "CLOUD"       // AWS EC2, GCP Compute, Azure VMs
)

// RuntimeNode represents an identifiable runtime instance.
type RuntimeNode struct {
	ID               string         // Unique node identifier
	Hostname         string         // Observable hostname
	Backend          RuntimeBackend // Infrastructure layer
	IsolationLevel   string         // PHYSICAL, VM, CONTAINER, PROCESS
	OSBoundary       string         // DISTINCT, SAME, UNKNOWN
	FilesystemBound  string         // DISTINCT, SAME, UNKNOWN
	PhysicalBound    string         // DISTINCT, SAME, UNKNOWN
	OperatorBound    string         // DISTINCT, SAME, UNKNOWN
	FirstObserved    int64          // UnixNano timestamp
	LastHeartbeat    int64          // UnixNano timestamp
}

// RuntimeTopology describes observed infrastructure layout.
type RuntimeTopology struct {
	NodeCount        int
	Nodes            []*RuntimeNode
	PhysicalHosts    int // Count of distinct physical machines
	VMHosts          int // Count of distinct VM hosts
	Containers       int // Count of distinct container runtimes
	OperatorDomains  int // Count of distinct operators
	DetectionMethod  string
	DetectionTime    int64
	Confidence       string // HIGH, MEDIUM, LOW
}

// DetectRuntime discovers the runtime backend and node topology.
func DetectRuntime(ctx context.Context) (*RuntimeTopology, error) {
	// Try to detect Kubernetes environment first (highest specificity)
	if isKubernetes() {
		return detectKubernetesTopology(ctx)
	}

	// Try to detect Docker environment
	if isDockerized() {
		return detectDockerTopology(ctx)
	}

	// Try to detect QEMU/KVM environment
	if isQEMUVM() {
		return detectQEMUTopology(ctx)
	}

	// Fall back to native runtime detection
	return detectNativeTopology(ctx)
}

// isKubernetes detects Kubernetes environment.
func isKubernetes() bool {
	// Check for Kubernetes API service environment variable
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}
	// Check for service account token
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		return true
	}
	return false
}

// isDockerized detects container runtime environment.
func isDockerized() bool {
	// Check for /.dockerenv (Docker marker)
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	// Check /proc/self/cgroup for docker/container references
	if data, err := os.ReadFile("/proc/self/cgroup"); err == nil {
		return bytes.Contains(data, []byte("docker")) || bytes.Contains(data, []byte("lxc"))
	}
	return false
}

// isQEMUVM detects QEMU/KVM hypervisor environment.
func isQEMUVM() bool {
	// Check for QEMU/KVM CPU flags in cpuinfo
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		return bytes.Contains(data, []byte("hypervisor")) && bytes.Contains(data, []byte("qemu"))
	}
	// Check dmesg for hypervisor detection
	cmd := exec.Command("dmesg")
	if output, err := cmd.CombinedOutput(); err == nil {
		return bytes.Contains(output, []byte("KVM"))
	}
	return false
}

// detectKubernetesTopology discovers Kubernetes pod topology.
func detectKubernetesTopology(ctx context.Context) (*RuntimeTopology, error) {
	hostname, _ := os.Hostname()

	topo := &RuntimeTopology{
		NodeCount:       1, // Single pod observed from inside
		Nodes:           make([]*RuntimeNode, 0),
		DetectionMethod: "kubernetes",
		Confidence:      "HIGH",
	}

	node := &RuntimeNode{
		ID:            hostname,
		Hostname:      hostname,
		Backend:       BACKEND_KUBERNETES,
		IsolationLevel: "CONTAINER",
		OSBoundary:    "SAME",    // All pods share node OS (possibly)
		FilesystemBound: "DISTINCT", // Each pod gets own filesystem
		PhysicalBound:   "UNKNOWN",  // Can't determine from inside pod
		OperatorBound:   "UNKNOWN",  // Can't determine from inside pod
		FirstObserved:   0,
		LastHeartbeat:   0,
	}

	topo.Nodes = append(topo.Nodes, node)
	return topo, nil
}

// detectDockerTopology discovers container topology.
func detectDockerTopology(ctx context.Context) (*RuntimeTopology, error) {
	hostname, _ := os.Hostname()
	containerID := hostname // Container hostname is often its ID

	topo := &RuntimeTopology{
		NodeCount:       1, // Single container observed from inside
		Nodes:           make([]*RuntimeNode, 0),
		Containers:      1,
		DetectionMethod: "docker",
		Confidence:      "HIGH",
	}

	node := &RuntimeNode{
		ID:            containerID,
		Hostname:      hostname,
		Backend:       BACKEND_CONTAINER,
		IsolationLevel: "CONTAINER",
		OSBoundary:    "SAME",       // Shares kernel with host
		FilesystemBound: "DISTINCT", // Separate filesystem namespace
		PhysicalBound:   "UNKNOWN",  // Unknown if multiple containers on same host
		OperatorBound:   "UNKNOWN",  // Can't determine from inside
		FirstObserved:   0,
		LastHeartbeat:   0,
	}

	topo.Nodes = append(topo.Nodes, node)
	return topo, nil
}

// detectQEMUTopology discovers QEMU/KVM VM topology.
func detectQEMUTopology(ctx context.Context) (*RuntimeTopology, error) {
	hostname, _ := os.Hostname()

	topo := &RuntimeTopology{
		NodeCount:       1, // Single VM observed from inside
		Nodes:           make([]*RuntimeNode, 0),
		VMHosts:         1,
		DetectionMethod: "qemu",
		Confidence:      "MEDIUM",
	}

	node := &RuntimeNode{
		ID:            hostname,
		Hostname:      hostname,
		Backend:       BACKEND_QEMU_VM,
		IsolationLevel: "VM",
		OSBoundary:    "DISTINCT", // VM has separate kernel
		FilesystemBound: "DISTINCT", // VM has separate filesystem
		PhysicalBound:   "UNKNOWN",  // Can't determine if multiple VMs on same host
		OperatorBound:   "UNKNOWN",  // Can't determine operator separation
		FirstObserved:   0,
		LastHeartbeat:   0,
	}

	topo.Nodes = append(topo.Nodes, node)
	return topo, nil
}

// detectNativeTopology discovers native runtime topology.
func detectNativeTopology(ctx context.Context) (*RuntimeTopology, error) {
	hostname, _ := os.Hostname()

	topo := &RuntimeTopology{
		NodeCount:       1,
		Nodes:           make([]*RuntimeNode, 0),
		PhysicalHosts:   1,
		DetectionMethod: "native",
		Confidence:      "HIGH",
	}

	node := &RuntimeNode{
		ID:            hostname,
		Hostname:      hostname,
		Backend:       BACKEND_NATIVE,
		IsolationLevel: "PROCESS",
		OSBoundary:    "SAME",       // Same OS kernel
		FilesystemBound: "SAME",     // Shared filesystem
		PhysicalBound:   "DISTINCT", // On a physical machine (assume)
		OperatorBound:   "UNKNOWN",  // Operator assignment unknown
		FirstObserved:   0,
		LastHeartbeat:   0,
	}

	topo.Nodes = append(topo.Nodes, node)
	return topo, nil
}

// ClassifyFailureDomain determines isolation boundaries between nodes.
func ClassifyFailureDomain(nodes []*RuntimeNode, boundaryType string) string {
	if len(nodes) == 0 {
		return "UNKNOWN"
	}

	switch boundaryType {
	case "OS":
		// Check if all nodes have distinct OS kernels
		osBoundaries := make(map[string]bool)
		for _, n := range nodes {
			if n.OSBoundary == "DISTINCT" {
				osBoundaries[n.Hostname] = true
			}
		}
		if len(osBoundaries) == len(nodes) {
			return "DISTINCT"
		}
		if len(osBoundaries) == 0 {
			return "SAME"
		}
		return "UNKNOWN"

	case "Filesystem":
		// Check if all nodes have distinct filesystems
		fsBoundaries := make(map[string]bool)
		for _, n := range nodes {
			if n.FilesystemBound == "DISTINCT" {
				fsBoundaries[n.Hostname] = true
			}
		}
		if len(fsBoundaries) == len(nodes) {
			return "DISTINCT"
		}
		if len(fsBoundaries) == 0 {
			return "SAME"
		}
		return "UNKNOWN"

	case "Physical":
		// Check if all nodes have distinct physical boundaries
		physicalBoundaries := make(map[string]bool)
		for _, n := range nodes {
			if n.PhysicalBound == "DISTINCT" {
				physicalBoundaries[n.Hostname] = true
			}
		}
		if len(physicalBoundaries) == len(nodes) {
			return "DISTINCT"
		}
		if len(physicalBoundaries) == 0 {
			return "SAME"
		}
		return "UNKNOWN"

	case "Operator":
		// Check if all nodes have distinct operators
		opBoundaries := make(map[string]bool)
		for _, n := range nodes {
			if n.OperatorBound == "DISTINCT" {
				opBoundaries[n.Hostname] = true
			}
		}
		if len(opBoundaries) == len(nodes) {
			return "DISTINCT"
		}
		if len(opBoundaries) == 0 {
			return "SAME"
		}
		return "UNKNOWN"

	default:
		return "UNKNOWN"
	}
}

// QualificationReport summarizes runtime qualification results.
type QualificationReport struct {
	CampaignID       string
	NodeCount        int
	BackendDetected  RuntimeBackend
	P1_CORE_Passed   bool
	P1_QEMU_Passed   bool
	P1_K8S_Passed    bool
	P2_Multi_Passed  bool
	OSBoundary       string
	FilesystemBound  string
	PhysicalBound    string
	OperatorBound    string
	SourceSHA        string
	Timestamp        int64
	Evidence         *QualifiedEvidence
	Summary          string
}

// GenerateQualificationReport creates a comprehensive qualification summary.
func GenerateQualificationReport(campaign *QualificationCampaign, topology *RuntimeTopology) *QualificationReport {
	report := &QualificationReport{
		CampaignID:      campaign.ID,
		NodeCount:       topology.NodeCount,
		P1_CORE_Passed:  campaign.Evidence != nil && campaign.Evidence.P1_CORE_Passed,
		SourceSHA:       campaign.SourceSHA,
		Timestamp:       campaign.EndTime.UnixNano(),
		Evidence:        campaign.Evidence,
	}

	// Classify failure domains based on topology
	if len(topology.Nodes) > 0 {
		report.OSBoundary = ClassifyFailureDomain(topology.Nodes, "OS")
		report.FilesystemBound = ClassifyFailureDomain(topology.Nodes, "Filesystem")
		report.PhysicalBound = ClassifyFailureDomain(topology.Nodes, "Physical")
		report.OperatorBound = ClassifyFailureDomain(topology.Nodes, "Operator")

		// Set backend qualification flags based on detected environment
		firstNode := topology.Nodes[0]
		switch firstNode.Backend {
		case BACKEND_QEMU_VM:
			report.P1_QEMU_Passed = report.P1_CORE_Passed
		case BACKEND_KUBERNETES:
			report.P1_K8S_Passed = report.P1_CORE_Passed
		}

		// P2_Multi requires multiple physical hosts
		if topology.PhysicalHosts >= 3 && report.P1_CORE_Passed {
			report.P2_Multi_Passed = true
		}
	}

	// Generate summary
	qualifications := []string{}
	if report.P1_CORE_Passed {
		qualifications = append(qualifications, "P1_CORE")
	}
	if report.P1_QEMU_Passed {
		qualifications = append(qualifications, "P1_QEMU")
	}
	if report.P1_K8S_Passed {
		qualifications = append(qualifications, "P1_K8S")
	}
	if report.P2_Multi_Passed {
		qualifications = append(qualifications, "P2_MULTI")
	}

	if len(qualifications) == 0 {
		report.Summary = fmt.Sprintf(
			"Qualification FAILED: %d nodes, backend %s, boundaries OS:%s FS:%s PHYS:%s OP:%s",
			report.NodeCount, topology.Nodes[0].Backend,
			report.OSBoundary, report.FilesystemBound, report.PhysicalBound, report.OperatorBound,
		)
	} else {
		report.Summary = fmt.Sprintf(
			"Qualification PASSED [%s]: %d nodes, backend %s, boundaries OS:%s FS:%s PHYS:%s OP:%s",
			strings.Join(qualifications, " + "), report.NodeCount, topology.Nodes[0].Backend,
			report.OSBoundary, report.FilesystemBound, report.PhysicalBound, report.OperatorBound,
		)
	}

	return report
}

// CandidateNode represents a potential runtime node for qualification.
type CandidateNode struct {
	ID        string
	Hostname  string
	Backend   RuntimeBackend
	Available bool
	Reason    string // Why this node was selected/rejected
}

// SelectQualificationNodes chooses optimal nodes for qualification testing.
func SelectQualificationNodes(ctx context.Context, count int) ([]*CandidateNode, error) {
	nodes := make([]*CandidateNode, 0)

	// v0.1: Return current node for single-node qualification
	// v0.2: Will implement multi-node discovery and selection
	hostname, _ := os.Hostname()

	node := &CandidateNode{
		ID:        hostname,
		Hostname:  hostname,
		Backend:   BACKEND_UNKNOWN,
		Available: true,
		Reason:    "self-qualification on current node",
	}

	nodes = append(nodes, node)
	return nodes, nil
}

// RuntimeQualificationConfig holds settings for qualification runs.
type RuntimeQualificationConfig struct {
	MinNodes       int
	MaxNodes       int
	TimeoutSeconds int
	RetryCount     int
	TestMode       string // "comprehensive" or "quick"
}

// DefaultQualificationConfig returns standard settings for P1_CORE qualification.
func DefaultQualificationConfig() *RuntimeQualificationConfig {
	return &RuntimeQualificationConfig{
		MinNodes:       1,
		MaxNodes:       32,
		TimeoutSeconds: 3600,
		RetryCount:     3,
		TestMode:       "comprehensive",
	}
}

// ToJSON marshals configuration to JSON.
func (c *RuntimeQualificationConfig) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON unmarshals configuration from JSON.
func (c *RuntimeQualificationConfig) FromJSON(data string) error {
	return json.Unmarshal([]byte(data), c)
}

// String returns configuration summary.
func (c *RuntimeQualificationConfig) String() string {
	return fmt.Sprintf(
		"RuntimeQualification{nodes:%d-%d, timeout:%ds, retries:%d, mode:%s}",
		c.MinNodes, c.MaxNodes, c.TimeoutSeconds, c.RetryCount, c.TestMode,
	)
}
