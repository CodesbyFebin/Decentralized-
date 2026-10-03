// Package topology provides region-aware node discovery and topology management for Decentralized.Host.
//
// Multi-region support includes:
//   - Region-aware node discovery and labeling
//   - Cross-region latency measurement and recording
//   - Region affinity constraints for placement
//   - Topology change detection and propagation
//
// Topology serves as the foundation for scheduling optimization and cross-region consensus.
package topology

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// RegionID uniquely identifies a geographic region.
type RegionID string

// ZoneID uniquely identifies an availability zone within a region.
type ZoneID string

// NodeID uniquely identifies a node in the topology.
type NodeID string

// Region represents a geographic region with its zones and nodes.
type Region struct {
	ID       RegionID       `json:"id"`
	Name     string         `json:"name"`
	Location string         `json:"location"` // Human-readable location (e.g., "us-west-1")
	Zones    map[ZoneID]*Zone `json:"zones"`
	Latency  map[RegionID]time.Duration `json:"latency"` // Measured latencies to other regions
	Updated  time.Time      `json:"updated"`
}

// Zone represents an availability zone within a region.
type Zone struct {
	ID       ZoneID         `json:"id"`
	Name     string         `json:"name"`
	Region   RegionID       `json:"region"`
	Nodes    map[NodeID]*TopologyNode `json:"nodes"`
	Updated  time.Time      `json:"updated"`
}

// TopologyNode represents a node in the topology with placement constraints.
type TopologyNode struct {
	ID           NodeID      `json:"id"`
	Name         string      `json:"name"`
	Zone         ZoneID      `json:"zone"`
	Region       RegionID    `json:"region"`
	Status       string      `json:"status"` // ready, pending, revoked, draining, lost
	Capacity     int64       `json:"capacity"` // Total capacity in arbitrary units (CPU millicores, etc.)
	Available    int64       `json:"available"` // Available capacity
	Workloads    int64       `json:"workloads"` // Number of active workloads
	LocalLatency time.Duration `json:"local_latency"` // Latency to zone leader
	Updated      time.Time   `json:"updated"`
}

// TopologySnapshot is an immutable view of the cluster topology at a point in time.
type TopologySnapshot struct {
	Timestamp  time.Time                    `json:"timestamp"`
	Regions    map[RegionID]*Region         `json:"regions"`
	NodeIndex  map[NodeID]*TopologyNode     `json:"node_index"` // Flat index for fast lookup
	Version    int64                        `json:"version"`
	ChangeLog  []TopologyChange             `json:"change_log"`
}

// TopologyChange represents a single topology change event.
type TopologyChange struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // node_joined, node_left, node_updated, latency_changed, zone_changed
	NodeID    NodeID    `json:"node_id,omitempty"`
	RegionID  RegionID  `json:"region_id,omitempty"`
	ZoneID    ZoneID    `json:"zone_id,omitempty"`
	Details   string    `json:"details,omitempty"`
	Version   int64     `json:"version"`
}

// Manager maintains and updates the topology state.
type Manager struct {
	mu              sync.RWMutex
	regions         map[RegionID]*Region
	nodeIndex       map[NodeID]*TopologyNode
	version         int64
	changeLog       []TopologyChange
	maxChangeLog    int
	latencyHistory  map[string][]time.Duration // region-pair -> latency samples
	maxHistorySamples int
}

// NewManager creates a new topology manager.
func NewManager() *Manager {
	return &Manager{
		regions:           make(map[RegionID]*Region),
		nodeIndex:         make(map[NodeID]*TopologyNode),
		changeLog:         make([]TopologyChange, 0),
		maxChangeLog:      1000,
		latencyHistory:    make(map[string][]time.Duration),
		maxHistorySamples: 100,
	}
}

// AddRegion adds or updates a region in the topology.
func (m *Manager) AddRegion(regionID RegionID, name, location string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.regions[regionID]; !exists {
		m.regions[regionID] = &Region{
			ID:       regionID,
			Name:     name,
			Location: location,
			Zones:    make(map[ZoneID]*Zone),
			Latency:  make(map[RegionID]time.Duration),
			Updated:  time.Now(),
		}
		m.version++
		m.logChange(TopologyChange{
			Timestamp: time.Now(),
			Type:      "region_added",
			RegionID:  regionID,
			Details:   fmt.Sprintf("Added region %s (%s)", name, location),
			Version:   m.version,
		})
	}
	return nil
}

// AddZone adds or updates a zone within a region.
func (m *Manager) AddZone(regionID RegionID, zoneID ZoneID, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	region, ok := m.regions[regionID]
	if !ok {
		return fmt.Errorf("region %s not found", regionID)
	}

	if _, exists := region.Zones[zoneID]; !exists {
		region.Zones[zoneID] = &Zone{
			ID:       zoneID,
			Name:     name,
			Region:   regionID,
			Nodes:    make(map[NodeID]*TopologyNode),
			Updated:  time.Now(),
		}
		region.Updated = time.Now()
		m.version++
		m.logChange(TopologyChange{
			Timestamp: time.Now(),
			Type:      "zone_added",
			RegionID:  regionID,
			ZoneID:    zoneID,
			Details:   fmt.Sprintf("Added zone %s in region %s", name, regionID),
			Version:   m.version,
		})
	}
	return nil
}

// AddNode adds or updates a node in the topology.
func (m *Manager) AddNode(nodeID NodeID, name string, regionID RegionID, zoneID ZoneID, capacity int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	region, ok := m.regions[regionID]
	if !ok {
		return fmt.Errorf("region %s not found", regionID)
	}

	zone, ok := region.Zones[zoneID]
	if !ok {
		return fmt.Errorf("zone %s not found in region %s", zoneID, regionID)
	}

	isNew := false
	node, ok := m.nodeIndex[nodeID]
	if !ok {
		isNew = true
		node = &TopologyNode{
			ID:        nodeID,
			Name:      name,
			Zone:      zoneID,
			Region:    regionID,
			Status:    "pending",
			Capacity:  capacity,
			Available: capacity,
			Updated:   time.Now(),
		}
		m.nodeIndex[nodeID] = node
		zone.Nodes[nodeID] = node
	} else {
		node.Capacity = capacity
		node.Available = capacity
		node.Updated = time.Now()
	}

	m.version++
	changeType := "node_joined"
	if !isNew {
		changeType = "node_updated"
	}
	m.logChange(TopologyChange{
		Timestamp: time.Now(),
		Type:      changeType,
		NodeID:    nodeID,
		RegionID:  regionID,
		ZoneID:    zoneID,
		Details:   fmt.Sprintf("Node %s capacity %d", name, capacity),
		Version:   m.version,
	})

	return nil
}

// RemoveNode removes a node from the topology.
func (m *Manager) RemoveNode(nodeID NodeID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, ok := m.nodeIndex[nodeID]
	if !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}

	region, ok := m.regions[node.Region]
	if ok {
		if zone, ok := region.Zones[node.Zone]; ok {
			delete(zone.Nodes, nodeID)
		}
	}
	delete(m.nodeIndex, nodeID)

	m.version++
	m.logChange(TopologyChange{
		Timestamp: time.Now(),
		Type:      "node_left",
		NodeID:    nodeID,
		RegionID:  node.Region,
		ZoneID:    node.Zone,
		Details:   fmt.Sprintf("Removed node %s", node.Name),
		Version:   m.version,
	})

	return nil
}

// UpdateNodeStatus updates a node's operational status.
func (m *Manager) UpdateNodeStatus(nodeID NodeID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, ok := m.nodeIndex[nodeID]
	if !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}

	oldStatus := node.Status
	node.Status = status
	node.Updated = time.Now()

	if oldStatus != status {
		m.version++
		m.logChange(TopologyChange{
			Timestamp: time.Now(),
			Type:      "node_updated",
			NodeID:    nodeID,
			Details:   fmt.Sprintf("Status changed: %s -> %s", oldStatus, status),
			Version:   m.version,
		})
	}

	return nil
}

// UpdateNodeCapacity updates a node's available capacity.
func (m *Manager) UpdateNodeCapacity(nodeID NodeID, available int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	node, ok := m.nodeIndex[nodeID]
	if !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.Available = available
	node.Updated = time.Now()
	return nil
}

// RecordLatency records measured latency between two regions.
func (m *Manager) RecordLatency(from, to RegionID, latency time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fromRegion, ok := m.regions[from]
	if !ok {
		return fmt.Errorf("region %s not found", from)
	}

	toRegion, ok := m.regions[to]
	if !ok {
		return fmt.Errorf("region %s not found", to)
	}

	// Update the latency map
	oldLatency := fromRegion.Latency[to]
	fromRegion.Latency[to] = latency
	fromRegion.Updated = time.Now()

	// Also update reverse direction
	toRegion.Latency[from] = latency
	toRegion.Updated = time.Now()

	// Keep history for analysis
	key := string(from) + "->" + string(to)
	if _, ok := m.latencyHistory[key]; !ok {
		m.latencyHistory[key] = make([]time.Duration, 0)
	}
	m.latencyHistory[key] = append(m.latencyHistory[key], latency)
	if len(m.latencyHistory[key]) > m.maxHistorySamples {
		m.latencyHistory[key] = m.latencyHistory[key][1:]
	}

	if oldLatency != latency {
		m.version++
		m.logChange(TopologyChange{
			Timestamp: time.Now(),
			Type:      "latency_changed",
			RegionID:  from,
			Details:   fmt.Sprintf("Latency to %s: %v", to, latency),
			Version:   m.version,
		})
	}

	return nil
}

// GetSnapshot returns an immutable snapshot of the current topology.
func (m *Manager) GetSnapshot() *TopologySnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Deep copy regions and node index
	regions := make(map[RegionID]*Region)
	for rID, r := range m.regions {
		regionCopy := &Region{
			ID:       r.ID,
			Name:     r.Name,
			Location: r.Location,
			Zones:    make(map[ZoneID]*Zone),
			Latency:  make(map[RegionID]time.Duration),
			Updated:  r.Updated,
		}
		for latencyRegion, latency := range r.Latency {
			regionCopy.Latency[latencyRegion] = latency
		}
		for zID, z := range r.Zones {
			zoneCopy := &Zone{
				ID:      z.ID,
				Name:    z.Name,
				Region:  z.Region,
				Nodes:   make(map[NodeID]*TopologyNode),
				Updated: z.Updated,
			}
			for nID, n := range z.Nodes {
				nodeCopy := &TopologyNode{
					ID:           n.ID,
					Name:         n.Name,
					Zone:         n.Zone,
					Region:       n.Region,
					Status:       n.Status,
					Capacity:     n.Capacity,
					Available:    n.Available,
					Workloads:    n.Workloads,
					LocalLatency: n.LocalLatency,
					Updated:      n.Updated,
				}
				zoneCopy.Nodes[nID] = nodeCopy
			}
			regionCopy.Zones[zID] = zoneCopy
		}
		regions[rID] = regionCopy
	}

	nodeIndex := make(map[NodeID]*TopologyNode)
	for nID, n := range m.nodeIndex {
		nodeCopy := &TopologyNode{
			ID:           n.ID,
			Name:         n.Name,
			Zone:         n.Zone,
			Region:       n.Region,
			Status:       n.Status,
			Capacity:     n.Capacity,
			Available:    n.Available,
			Workloads:    n.Workloads,
			LocalLatency: n.LocalLatency,
			Updated:      n.Updated,
		}
		nodeIndex[nID] = nodeCopy
	}

	changeLogCopy := make([]TopologyChange, len(m.changeLog))
	copy(changeLogCopy, m.changeLog)

	return &TopologySnapshot{
		Timestamp: time.Now(),
		Regions:   regions,
		NodeIndex: nodeIndex,
		Version:   m.version,
		ChangeLog: changeLogCopy,
	}
}

// GetNodesInRegion returns all nodes in a region.
func (m *Manager) GetNodesInRegion(regionID RegionID) []*TopologyNode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	region, ok := m.regions[regionID]
	if !ok {
		return nil
	}

	nodes := make([]*TopologyNode, 0)
	for _, zone := range region.Zones {
		for _, node := range zone.Nodes {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// GetNodesInZone returns all nodes in a zone.
func (m *Manager) GetNodesInZone(regionID RegionID, zoneID ZoneID) []*TopologyNode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	region, ok := m.regions[regionID]
	if !ok {
		return nil
	}

	zone, ok := region.Zones[zoneID]
	if !ok {
		return nil
	}

	nodes := make([]*TopologyNode, 0)
	for _, node := range zone.Nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetNode returns a specific node.
func (m *Manager) GetNode(nodeID NodeID) *TopologyNode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.nodeIndex[nodeID]
}

// GetLatency returns the measured latency between two regions.
func (m *Manager) GetLatency(from, to RegionID) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	region, ok := m.regions[from]
	if !ok {
		return 0
	}

	return region.Latency[to]
}

// GetAverageLatency returns the average measured latency between two regions.
func (m *Manager) GetAverageLatency(from, to RegionID) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := string(from) + "->" + string(to)
	history, ok := m.latencyHistory[key]
	if !ok || len(history) == 0 {
		return 0
	}

	var total time.Duration
	for _, l := range history {
		total += l
	}
	return total / time.Duration(len(history))
}

// GetVersion returns the current topology version.
func (m *Manager) GetVersion() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.version
}

// GetChanges returns topology changes since a given version.
func (m *Manager) GetChanges(since int64) []TopologyChange {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var changes []TopologyChange
	for _, change := range m.changeLog {
		if change.Version > since {
			changes = append(changes, change)
		}
	}
	return changes
}

// GetReadyNodes returns all nodes with status "ready".
func (m *Manager) GetReadyNodes() []*TopologyNode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var nodes []*TopologyNode
	for _, node := range m.nodeIndex {
		if node.Status == "ready" {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// GetRegions returns all regions.
func (m *Manager) GetRegions() []*Region {
	m.mu.RLock()
	defer m.mu.RUnlock()

	regions := make([]*Region, 0, len(m.regions))
	for _, r := range m.regions {
		regions = append(regions, r)
	}
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].ID < regions[j].ID
	})
	return regions
}

// logChange records a topology change in the change log.
func (m *Manager) logChange(change TopologyChange) {
	m.changeLog = append(m.changeLog, change)
	if len(m.changeLog) > m.maxChangeLog {
		m.changeLog = m.changeLog[len(m.changeLog)-m.maxChangeLog:]
	}
}

// Stats provides statistics about the topology.
type Stats struct {
	TotalNodes      int64
	ReadyNodes      int64
	RegionCount     int64
	ZoneCount       int64
	TotalCapacity   int64
	AvailableCapacity int64
	AverageWorkload float64
}

// GetStats returns topology statistics.
func (m *Manager) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var stats Stats
	stats.RegionCount = int64(len(m.regions))

	zoneSet := make(map[string]bool)
	for _, node := range m.nodeIndex {
		stats.TotalNodes++
		stats.TotalCapacity += node.Capacity
		stats.AvailableCapacity += node.Available
		if node.Status == "ready" {
			stats.ReadyNodes++
		}
		zoneSet[string(node.Region)+"|"+string(node.Zone)] = true
	}

	stats.ZoneCount = int64(len(zoneSet))

	if stats.TotalNodes > 0 {
		totalWorkloads := int64(0)
		for _, node := range m.nodeIndex {
			totalWorkloads += node.Workloads
		}
		stats.AverageWorkload = float64(totalWorkloads) / float64(stats.TotalNodes)
	}

	return stats
}
