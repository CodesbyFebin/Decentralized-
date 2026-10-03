// Package nodes implements operator node registration, validation, and health monitoring
// for the Decentralized.Host operator qualification program.
//
// Operators must register 50+ nodes for readiness, with hardware validation,
// network connectivity tests, and geographic diversity tracking.
package nodes

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	// MinimumNodesForReadiness is the required node count
	MinimumNodesForReadiness = 50

	// MinimumGeographicRegions is the preferred number of regions
	MinimumGeographicRegions = 3

	// HealthCheckInterval is the recommended health check interval
	HealthCheckInterval = 5 * time.Minute

	// NodeHealthyThreshold is the percentage needed for healthy status
	NodeHealthyThreshold = 95.0
)

// NodeStatus represents the operational status of a node
type NodeStatus string

const (
	HEALTHY     NodeStatus = "HEALTHY"
	DEGRADED    NodeStatus = "DEGRADED"
	UNHEALTHY   NodeStatus = "UNHEALTHY"
	UNREACHABLE NodeStatus = "UNREACHABLE"
	OFFLINE     NodeStatus = "OFFLINE"
)

// NodeInfo contains registration and validation data for a node
type NodeInfo struct {
	NodeID               string                 `json:"node_id"` // dh1 identifier
	OperatorID           string                 `json:"operator_id"`
	Status               NodeStatus             `json:"status"`
	HardwareProfile      HardwareProfile        `json:"hardware_profile"`
	NetworkProfile       NetworkProfile         `json:"network_profile"`
	GeographicLocation   GeographicLocation     `json:"geographic_location"`
	RegistrationTime     time.Time              `json:"registration_time"`
	LastHealthCheckTime  time.Time              `json:"last_health_check_time"`
	HealthCheckInterval  time.Duration          `json:"health_check_interval"`
	UptimePercentage     float64                `json:"uptime_percentage"` // 0-100
	SuccessfulChecks     int                    `json:"successful_checks"`
	FailedChecks         int                    `json:"failed_checks"`
	NetworkLatency       time.Duration          `json:"network_latency"`
	ConnectionQuality    string                 `json:"connection_quality"` // EXCELLENT, GOOD, FAIR, POOR
	Tags                 map[string]string      `json:"tags,omitempty"`
}

// HardwareProfile contains hardware specifications
type HardwareProfile struct {
	CPUCores      int    `json:"cpu_cores"`
	CPUSpeedGHz   float64 `json:"cpu_speed_ghz"`
	MemoryGB      int    `json:"memory_gb"`
	DiskGB        int    `json:"disk_gb"`
	DiskType      string `json:"disk_type"` // SSD, HDD, NVME
	ValidatedAt   time.Time `json:"validated_at"`
	ValidationErr string `json:"validation_err,omitempty"`
}

// NetworkProfile contains network connectivity data
type NetworkProfile struct {
	IPv4Address      string        `json:"ipv4_address"`
	IPv6Address      string        `json:"ipv6_address,omitempty"`
	Port             int           `json:"port"`
	Bandwidth        int64         `json:"bandwidth"` // Mbps
	Latency          time.Duration `json:"latency"`
	PacketLoss       float64       `json:"packet_loss"` // percentage
	ConnectionType   string        `json:"connection_type"` // public, NAT, VPN
	TestedAt         time.Time     `json:"tested_at"`
	TestErr          string        `json:"test_err,omitempty"`
}

// GeographicLocation contains node location data
type GeographicLocation struct {
	Region          string    `json:"region"` // e.g., "us-east", "eu-west", "ap-southeast"
	Country         string    `json:"country"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	Timezone        string    `json:"timezone"`
	DataCenterName  string    `json:"data_center_name,omitempty"`
}

// NodeRegistry manages node registration and validation
type NodeRegistry struct {
	mu    sync.RWMutex
	nodes map[string]*NodeInfo
	ops   map[string][]string // operator -> node IDs
}

// NewNodeRegistry creates a new node registry
func NewNodeRegistry() *NodeRegistry {
	return &NodeRegistry{
		nodes: make(map[string]*NodeInfo),
		ops:   make(map[string][]string),
	}
}

// RegisterNode registers a new node
func (nr *NodeRegistry) RegisterNode(nodeID string, operatorID string, hardware HardwareProfile, network NetworkProfile, location GeographicLocation) (*NodeInfo, error) {
	if nodeID == "" || operatorID == "" {
		return nil, errors.New("node ID and operator ID cannot be empty")
	}

	// Validate hardware
	if err := validateHardware(hardware); err != nil {
		return nil, fmt.Errorf("hardware validation failed: %w", err)
	}

	// Validate network
	if err := validateNetwork(network); err != nil {
		return nil, fmt.Errorf("network validation failed: %w", err)
	}

	nr.mu.Lock()
	defer nr.mu.Unlock()

	if _, exists := nr.nodes[nodeID]; exists {
		return nil, fmt.Errorf("node %s already registered", nodeID)
	}

	now := time.Now()
	node := &NodeInfo{
		NodeID:              nodeID,
		OperatorID:          operatorID,
		Status:              HEALTHY,
		HardwareProfile:     hardware,
		NetworkProfile:      network,
		GeographicLocation:  location,
		RegistrationTime:    now,
		LastHealthCheckTime: now,
		HealthCheckInterval: HealthCheckInterval,
		UptimePercentage:    100.0,
		SuccessfulChecks:    1,
		FailedChecks:        0,
		NetworkLatency:      network.Latency,
		ConnectionQuality:   assessConnectionQuality(network.Latency, network.PacketLoss),
		Tags:                make(map[string]string),
	}

	nr.nodes[nodeID] = node
	nr.ops[operatorID] = append(nr.ops[operatorID], nodeID)

	return node, nil
}

// GetNode retrieves a node's information
func (nr *NodeRegistry) GetNode(nodeID string) (*NodeInfo, error) {
	nr.mu.RLock()
	defer nr.mu.RUnlock()

	node, exists := nr.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}
	return node, nil
}

// RecordHealthCheck records a health check result
func (nr *NodeRegistry) RecordHealthCheck(nodeID string, healthy bool, latency time.Duration) error {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	node, exists := nr.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	now := time.Now()
	if healthy {
		node.SuccessfulChecks++
		node.Status = HEALTHY
	} else {
		node.FailedChecks++
		node.Status = UNHEALTHY
	}

	node.LastHealthCheckTime = now
	node.NetworkLatency = latency
	node.ConnectionQuality = assessConnectionQuality(latency, node.NetworkProfile.PacketLoss)

	// Calculate uptime percentage
	total := node.SuccessfulChecks + node.FailedChecks
	if total > 0 {
		node.UptimePercentage = float64(node.SuccessfulChecks) / float64(total) * 100.0
	}

	// Update status based on uptime
	if node.UptimePercentage >= NodeHealthyThreshold {
		node.Status = HEALTHY
	} else if node.UptimePercentage >= 80.0 {
		node.Status = DEGRADED
	} else {
		node.Status = UNHEALTHY
	}

	return nil
}

// OperatorNodeCount returns number of nodes for an operator
func (nr *NodeRegistry) OperatorNodeCount(operatorID string) int {
	nr.mu.RLock()
	defer nr.mu.RUnlock()

	return len(nr.ops[operatorID])
}

// OperatorNodes returns all nodes for an operator
func (nr *NodeRegistry) OperatorNodes(operatorID string) []*NodeInfo {
	nr.mu.RLock()
	defer nr.mu.RUnlock()

	nodeIDs := nr.ops[operatorID]
	nodes := make([]*NodeInfo, 0, len(nodeIDs))
	for _, id := range nodeIDs {
		if node, ok := nr.nodes[id]; ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// OperatorReadiness checks if operator has minimum node requirements
type OperatorNodeStatus struct {
	OperatorID         string
	NodeCount          int
	IsReadyNodeCount   bool
	RegionCount        int
	IsReadyRegions     bool
	AverageUptime      float64
	HealthyNodeCount   int
	UnhealthyNodeCount int
	DegradedNodeCount  int
}

// GetOperatorNodeStatus returns comprehensive node status for operator
func (nr *NodeRegistry) GetOperatorNodeStatus(operatorID string) OperatorNodeStatus {
	nr.mu.RLock()
	defer nr.mu.RUnlock()

	nodeIDs := nr.ops[operatorID]
	status := OperatorNodeStatus{
		OperatorID:       operatorID,
		NodeCount:        len(nodeIDs),
		IsReadyNodeCount: len(nodeIDs) >= MinimumNodesForReadiness,
	}

	regions := make(map[string]bool)
	var totalUptime float64

	for _, id := range nodeIDs {
		if node, ok := nr.nodes[id]; ok {
			regions[node.GeographicLocation.Region] = true
			totalUptime += node.UptimePercentage

			switch node.Status {
			case HEALTHY:
				status.HealthyNodeCount++
			case UNHEALTHY, OFFLINE:
				status.UnhealthyNodeCount++
			case DEGRADED:
				status.DegradedNodeCount++
			}
		}
	}

	status.RegionCount = len(regions)
	status.IsReadyRegions = status.RegionCount >= MinimumGeographicRegions

	if len(nodeIDs) > 0 {
		status.AverageUptime = totalUptime / float64(len(nodeIDs))
	}

	return status
}

// validateHardware validates hardware specifications
func validateHardware(hw HardwareProfile) error {
	if hw.CPUCores < 2 {
		return errors.New("at least 2 CPU cores required")
	}
	if hw.MemoryGB < 4 {
		return errors.New("at least 4 GB memory required")
	}
	if hw.DiskGB < 100 {
		return errors.New("at least 100 GB disk required")
	}
	return nil
}

// validateNetwork validates network specifications
func validateNetwork(net NetworkProfile) error {
	if net.Port < 1024 || net.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", net.Port)
	}
	if net.Bandwidth < 10 {
		return errors.New("minimum 10 Mbps bandwidth required")
	}
	if net.PacketLoss > 5.0 {
		return fmt.Errorf("packet loss %.1f%% exceeds 5%% threshold", net.PacketLoss)
	}
	return nil
}

// assessConnectionQuality evaluates connection quality
func assessConnectionQuality(latency time.Duration, packetLoss float64) string {
	if latency < 50*time.Millisecond && packetLoss < 0.1 {
		return "EXCELLENT"
	}
	if latency < 100*time.Millisecond && packetLoss < 0.5 {
		return "GOOD"
	}
	if latency < 200*time.Millisecond && packetLoss < 2.0 {
		return "FAIR"
	}
	return "POOR"
}

// ListNodes returns all registered nodes
func (nr *NodeRegistry) ListNodes() []*NodeInfo {
	nr.mu.RLock()
	defer nr.mu.RUnlock()

	nodes := make([]*NodeInfo, 0, len(nr.nodes))
	for _, node := range nr.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// NodeRegistryStatus returns aggregate statistics
type NodeRegistryStatus struct {
	TotalNodes       int
	HealthyNodes     int
	UnhealthyNodes   int
	DegradedNodes    int
	AverageUptime    float64
	UniqueRegions    int
	RegistrarsCount  int
}

// GetStatus returns overall node registry statistics
func (nr *NodeRegistry) GetStatus() NodeRegistryStatus {
	nr.mu.RLock()
	defer nr.mu.RUnlock()

	status := NodeRegistryStatus{
		TotalNodes:      len(nr.nodes),
		RegistrarsCount: len(nr.ops),
	}

	regions := make(map[string]bool)
	var totalUptime float64

	for _, node := range nr.nodes {
		regions[node.GeographicLocation.Region] = true
		totalUptime += node.UptimePercentage

		switch node.Status {
		case HEALTHY:
			status.HealthyNodes++
		case UNHEALTHY, OFFLINE:
			status.UnhealthyNodes++
		case DEGRADED:
			status.DegradedNodes++
		}
	}

	status.UniqueRegions = len(regions)

	if len(nr.nodes) > 0 {
		status.AverageUptime = totalUptime / float64(len(nr.nodes))
	}

	return status
}

// SetTag sets a metadata tag on a node
func (nr *NodeRegistry) SetTag(nodeID string, key string, value string) error {
	nr.mu.Lock()
	defer nr.mu.Unlock()

	node, exists := nr.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.Tags[key] = value
	return nil
}
