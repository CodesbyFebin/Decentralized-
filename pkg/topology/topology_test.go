package topology

import (
	"testing"
	"time"
)

func TestTopologyManager(t *testing.T) {
	m := NewManager()

	// Test adding regions
	err := m.AddRegion("us-west", "US West", "us-west-1")
	if err != nil {
		t.Fatalf("AddRegion failed: %v", err)
	}

	err = m.AddRegion("us-east", "US East", "us-east-1")
	if err != nil {
		t.Fatalf("AddRegion failed: %v", err)
	}

	// Test adding zones
	err = m.AddZone("us-west", "us-west-1a", "Zone A")
	if err != nil {
		t.Fatalf("AddZone failed: %v", err)
	}

	err = m.AddZone("us-west", "us-west-1b", "Zone B")
	if err != nil {
		t.Fatalf("AddZone failed: %v", err)
	}

	err = m.AddZone("us-east", "us-east-1a", "Zone A")
	if err != nil {
		t.Fatalf("AddZone failed: %v", err)
	}

	// Test adding nodes
	err = m.AddNode("node-1", "Node 1", "us-west", "us-west-1a", 1000)
	if err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}

	err = m.AddNode("node-2", "Node 2", "us-west", "us-west-1b", 1000)
	if err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}

	err = m.AddNode("node-3", "Node 3", "us-east", "us-east-1a", 1000)
	if err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}

	// Test getting nodes in region
	nodesWest := m.GetNodesInRegion("us-west")
	if len(nodesWest) != 2 {
		t.Fatalf("Expected 2 nodes in us-west, got %d", len(nodesWest))
	}

	nodesEast := m.GetNodesInRegion("us-east")
	if len(nodesEast) != 1 {
		t.Fatalf("Expected 1 node in us-east, got %d", len(nodesEast))
	}

	// Test getting nodes in zone
	nodesZone := m.GetNodesInZone("us-west", "us-west-1a")
	if len(nodesZone) != 1 {
		t.Fatalf("Expected 1 node in zone, got %d", len(nodesZone))
	}

	// Test recording latency
	err = m.RecordLatency("us-west", "us-east", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("RecordLatency failed: %v", err)
	}

	latency := m.GetLatency("us-west", "us-east")
	if latency != 50*time.Millisecond {
		t.Fatalf("Expected latency 50ms, got %v", latency)
	}

	// Test updating node status
	err = m.UpdateNodeStatus("node-1", "ready")
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	node := m.GetNode("node-1")
	if node.Status != "ready" {
		t.Fatalf("Expected node status 'ready', got %s", node.Status)
	}

	// Test getting snapshot
	snapshot := m.GetSnapshot()
	if snapshot.Version < 1 {
		t.Fatalf("Expected version >= 1, got %d", snapshot.Version)
	}

	if len(snapshot.Regions) != 2 {
		t.Fatalf("Expected 2 regions in snapshot, got %d", len(snapshot.Regions))
	}

	if len(snapshot.NodeIndex) != 3 {
		t.Fatalf("Expected 3 nodes in snapshot, got %d", len(snapshot.NodeIndex))
	}

	// Test getting stats
	stats := m.GetStats()
	if stats.TotalNodes != 3 {
		t.Fatalf("Expected 3 total nodes, got %d", stats.TotalNodes)
	}

	if stats.RegionCount != 2 {
		t.Fatalf("Expected 2 regions, got %d", stats.RegionCount)
	}

	if stats.TotalCapacity != 3000 {
		t.Fatalf("Expected 3000 total capacity, got %d", stats.TotalCapacity)
	}

	// Test removing node
	err = m.RemoveNode("node-1")
	if err != nil {
		t.Fatalf("RemoveNode failed: %v", err)
	}

	node = m.GetNode("node-1")
	if node != nil {
		t.Fatalf("Expected node to be removed, but found %v", node)
	}

	stats = m.GetStats()
	if stats.TotalNodes != 2 {
		t.Fatalf("Expected 2 total nodes after removal, got %d", stats.TotalNodes)
	}

	// Test getting ready nodes
	m.UpdateNodeStatus("node-2", "ready")
	m.UpdateNodeStatus("node-3", "ready")

	readyNodes := m.GetReadyNodes()
	if len(readyNodes) != 2 {
		t.Fatalf("Expected 2 ready nodes, got %d", len(readyNodes))
	}

	// Test change log
	changes := m.GetChanges(0)
	if len(changes) < 1 {
		t.Fatalf("Expected at least 1 change in log")
	}

	// Test average latency
	m.RecordLatency("us-west", "us-east", 55*time.Millisecond)
	m.RecordLatency("us-west", "us-east", 45*time.Millisecond)

	avgLatency := m.GetAverageLatency("us-west", "us-east")
	expectedAvg := time.Duration(50 * time.Millisecond)
	if avgLatency != expectedAvg {
		t.Logf("Average latency: %v (expected ~50ms)", avgLatency)
	}
}

func TestTopologyErrors(t *testing.T) {
	m := NewManager()

	// Test error cases
	err := m.AddZone("nonexistent-region", "zone-1", "Zone")
	if err == nil {
		t.Fatalf("Expected error adding zone to nonexistent region")
	}

	err = m.AddNode("node", "Node", "nonexistent-region", "zone-1", 1000)
	if err == nil {
		t.Fatalf("Expected error adding node to nonexistent region")
	}

	err = m.RemoveNode("nonexistent-node")
	if err == nil {
		t.Fatalf("Expected error removing nonexistent node")
	}

	err = m.UpdateNodeStatus("nonexistent-node", "ready")
	if err == nil {
		t.Fatalf("Expected error updating nonexistent node")
	}
}

func TestLargeTopology(t *testing.T) {
	m := NewManager()

	// Create a large topology with 450 nodes across 3 regions
	regions := []RegionID{"us-west", "us-east", "eu-west"}
	regionNames := []string{"US West", "US East", "EU West"}

	for i, region := range regions {
		m.AddRegion(region, regionNames[i], "location-"+string(region))

		// Each region has 5 zones with 30 nodes each = 150 nodes
		for z := 0; z < 5; z++ {
			zoneID := ZoneID(string(region) + "-" + string(rune('a'+z)))
			m.AddZone(region, zoneID, "Zone "+string(rune('a'+z)))

			for n := 0; n < 30; n++ {
				nodeID := NodeID(string(region) + "-" + string(rune('a'+z)) + "-" + string(rune(n)))
				m.AddNode(nodeID, "Node", region, zoneID, 1000)
				m.UpdateNodeStatus(nodeID, "ready")
			}
		}
	}

	// Verify the topology
	stats := m.GetStats()
	if stats.TotalNodes != 450 {
		t.Fatalf("Expected 450 nodes, got %d", stats.TotalNodes)
	}

	if stats.RegionCount != 3 {
		t.Fatalf("Expected 3 regions, got %d", stats.RegionCount)
	}

	if stats.ZoneCount != 15 {
		t.Fatalf("Expected 15 zones, got %d", stats.ZoneCount)
	}

	// Record latencies between regions
	m.RecordLatency("us-west", "us-east", 50*time.Millisecond)
	m.RecordLatency("us-west", "eu-west", 150*time.Millisecond)
	m.RecordLatency("us-east", "eu-west", 180*time.Millisecond)

	// Test getting all regions
	allRegions := m.GetRegions()
	if len(allRegions) != 3 {
		t.Fatalf("Expected 3 regions, got %d", len(allRegions))
	}

	// Test getting snapshot
	snapshot := m.GetSnapshot()
	if len(snapshot.Regions) != 3 {
		t.Fatalf("Expected 3 regions in snapshot, got %d", len(snapshot.Regions))
	}

	if len(snapshot.NodeIndex) != 450 {
		t.Fatalf("Expected 450 nodes in snapshot, got %d", len(snapshot.NodeIndex))
	}
}
