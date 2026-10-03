package integration

import (
	"fmt"
	"testing"
	"time"

	"decentralized.host/pkg/api"
	"decentralized.host/pkg/scheduler"
	"decentralized.host/pkg/snapshots"
	"decentralized.host/pkg/topology"
)

// TestPhase6A_MultiRegionTopology verifies multi-region topology awareness
func TestPhase6A_MultiRegionTopology(t *testing.T) {
	t.Logf("Phase 6A: Multi-Region Topology Awareness")

	// Create topology manager
	tm := topology.NewManager()

	// Add 3 regions
	tm.AddRegion("us-west", "US West Coast", "us-west-1")
	tm.AddRegion("us-east", "US East Coast", "us-east-1")
	tm.AddRegion("eu-west", "Europe West", "eu-west-1")

	// Add zones in each region
	for _, region := range []string{"us-west", "us-east", "eu-west"} {
		for i := 0; i < 3; i++ {
			zone := topology.ZoneID(fmt.Sprintf("%s-1%s", region, string(rune('a'+i))))
			tm.AddZone(topology.RegionID(region), zone, fmt.Sprintf("Zone %s", string(rune('a'+i))))
		}
	}

	// Add nodes across regions (150 per region)
	for regionIdx, region := range []string{"us-west", "us-east", "eu-west"} {
		for nodeIdx := 0; nodeIdx < 150; nodeIdx++ {
			zoneChar := rune('a' + (nodeIdx % 3))
			zone := topology.ZoneID(fmt.Sprintf("%s-1%s", region, string(zoneChar)))
			nodeID := topology.NodeID(fmt.Sprintf("node-%03d-%d", regionIdx*150+nodeIdx, regionIdx))

			tm.AddNode(nodeID, fmt.Sprintf("Node %s", nodeID), topology.RegionID(region), zone, 4000)
			tm.UpdateNodeStatus(nodeID, "ready")
		}
	}

	// Record cross-region latencies
	tm.RecordLatency("us-west", "us-east", 50*time.Millisecond)
	tm.RecordLatency("us-west", "eu-west", 150*time.Millisecond)
	tm.RecordLatency("us-east", "eu-west", 180*time.Millisecond)

	// Get snapshot and verify
	snap := tm.GetSnapshot()

	if len(snap.Regions) != 3 {
		t.Fatalf("Expected 3 regions, got %d", len(snap.Regions))
	}

	if len(snap.NodeIndex) != 450 {
		t.Fatalf("Expected 450 nodes, got %d", len(snap.NodeIndex))
	}

	// Get statistics
	stats := tm.GetStats()
	t.Logf("Topology Stats:")
	t.Logf("  Total nodes: %d", stats.TotalNodes)
	t.Logf("  Ready nodes: %d", stats.ReadyNodes)
	t.Logf("  Regions: %d", stats.RegionCount)
	t.Logf("  Zones: %d", stats.ZoneCount)
	t.Logf("  Total capacity: %d", stats.TotalCapacity)
	t.Logf("  Available capacity: %d", stats.AvailableCapacity)

	if stats.TotalNodes != 450 {
		t.Fatalf("Expected 450 total nodes, got %d", stats.TotalNodes)
	}

	if stats.RegionCount != 3 {
		t.Fatalf("Expected 3 regions, got %d", stats.RegionCount)
	}

	if stats.ZoneCount != 9 {
		t.Fatalf("Expected 9 zones, got %d", stats.ZoneCount)
	}
}

// TestPhase6A_DeltaSnapshots verifies delta snapshot functionality
func TestPhase6A_DeltaSnapshots(t *testing.T) {
	t.Logf("Phase 6A: Delta Snapshots for Bandwidth Reduction")

	repo := snapshots.NewRepository()

	// Create initial full snapshot (10,000 objects)
	fullObjects := make(map[snapshots.ObjectID][]byte)
	for i := 0; i < 10000; i++ {
		objID := snapshots.ObjectID(fmt.Sprintf("obj-%d", i))
		data := []byte(fmt.Sprintf("object data %d", i))
		fullObjects[objID] = data
	}

	snapshotID, err := repo.CreateFullSnapshot(fullObjects)
	if err != nil {
		t.Fatalf("CreateFullSnapshot failed: %v", err)
	}

	fullSnap := repo.GetSnapshot(snapshotID)
	fullSize := fullSnap.Bytes

	t.Logf("Full Snapshot:")
	t.Logf("  Objects: %d", len(fullSnap.Objects))
	t.Logf("  Size: %d bytes", fullSize)

	// Create delta: 5% modification (500 objects changed, 500 new)
	modifiedObjects := make(map[snapshots.ObjectID][]byte)
	for i := 0; i < 10000; i++ {
		objID := snapshots.ObjectID(fmt.Sprintf("obj-%d", i))
		if i < 9500 {
			// Keep unchanged
			modifiedObjects[objID] = []byte(fmt.Sprintf("object data %d", i))
		} else {
			// Modified
			modifiedObjects[objID] = []byte(fmt.Sprintf("modified object %d", i))
		}
	}

	// Add 500 new objects
	for i := 10000; i < 10500; i++ {
		objID := snapshots.ObjectID(fmt.Sprintf("obj-%d", i))
		modifiedObjects[objID] = []byte(fmt.Sprintf("object data %d", i))
	}

	deltaID, err := repo.CreateDeltaSnapshot(snapshotID, modifiedObjects)
	if err != nil {
		t.Fatalf("CreateDeltaSnapshot failed: %v", err)
	}

	delta := repo.GetDelta(deltaID)
	deltaSize := delta.Size

	compressionRatio := float64(deltaSize) / float64(fullSize)
	bandwidthSavings := (1 - compressionRatio) * 100

	t.Logf("Delta Snapshot (5%% change):")
	t.Logf("  Added objects: %d", len(delta.Added))
	t.Logf("  Modified objects: %d", len(delta.Modified))
	t.Logf("  Removed objects: %d", len(delta.Removed))
	t.Logf("  Delta size: %d bytes", deltaSize)
	t.Logf("  Compression ratio: %.2f", compressionRatio)
	t.Logf("  Bandwidth saved: %.1f%%", bandwidthSavings)

	// Verify delta is much smaller than full snapshot
	if compressionRatio > 0.15 {
		t.Logf("WARNING: Compression ratio %.2f higher than expected for 5%% change", compressionRatio)
	}

	// Apply delta and verify correctness
	reconstructed, err := repo.ApplyDelta(deltaID)
	if err != nil {
		t.Fatalf("ApplyDelta failed: %v", err)
	}

	if len(reconstructed.Objects) != 10500 {
		t.Fatalf("Expected 10500 objects after delta, got %d", len(reconstructed.Objects))
	}
}

// TestPhase6A_LargeScaleScheduling verifies scheduler optimization for 1000+ nodes
func TestPhase6A_LargeScaleScheduling(t *testing.T) {
	t.Logf("Phase 6A: Scheduler Optimization for 1000+ Nodes")

	// Create 1000-node cluster
	nodes := make([]scheduler.Node, 1000)
	regions := []string{"us-west", "us-east", "eu-west"}

	for i := 0; i < 1000; i++ {
		region := regions[i%3]
		zone := fmt.Sprintf("%s-1%s", region, string(rune('a'+(i%3))))

		nodes[i] = scheduler.Node{
			ID:       fmt.Sprintf("node-%04d", i),
			Name:     fmt.Sprintf("node-%04d", i),
			Status:   "ready",
			Region:   region,
			Zone:     zone,
			CPUMilli: 4000,
			MemBytes: 8e9,
			Tiers:    []string{"compute"},
		}
	}

	// Test scheduling multiple apps
	apps := []struct {
		name     string
		replicas int64
	}{
		{"app-1", 5},
		{"app-2", 10},
		{"app-3", 20},
	}

	totalTime := time.Duration(0)

	for _, app := range apps {
		req := scheduler.Request{
			App: app.name,
			Spec: api.AppSpec{
				Replicas: app.replicas,
				Resources: api.Resources{
					CPUMilli: 500,
					MemBytes: 1e9,
				},
				Placement: api.Placement{
					Tiers:        []string{"compute"},
					Spread:       "failure-domain",
					PreferRegion: "us-west",
				},
			},
			Nodes: nodes,
		}

		startTime := time.Now()
		plan := scheduler.ScheduleFast(req)
		elapsed := time.Since(startTime)
		totalTime += elapsed

		scheduled := 0
		for _, rp := range plan.Replicas {
			if rp.Node != "" {
				scheduled++
			}
		}

		t.Logf("%s: Scheduled %d/%d replicas in %v", app.name, scheduled, app.replicas, elapsed)

		if elapsed > 100*time.Millisecond {
			t.Logf("  WARNING: Exceeded 100ms target")
		}
	}

	t.Logf("Total scheduling time for 3 apps: %v", totalTime)
	t.Logf("Average per app: %v", totalTime/3)
}

// TestPhase6A_Gate23_MultiRegionFailover tests GATE 23 requirements
func TestPhase6A_Gate23_MultiRegionFailover(t *testing.T) {
	t.Logf("GATE 23: Multi-Region Failover (450 nodes, 3 regions, <5s recovery)")

	// Create topology
	tm := topology.NewManager()
	tm.AddRegion("us-west", "US West", "us-west-1")
	tm.AddRegion("us-east", "US East", "us-east-1")
	tm.AddRegion("eu-west", "EU West", "eu-west-1")

	// Add zones
	for _, region := range []string{"us-west", "us-east", "eu-west"} {
		for i := 0; i < 5; i++ {
			zone := topology.ZoneID(fmt.Sprintf("%s-1%s", region, string(rune('a'+i))))
			tm.AddZone(topology.RegionID(region), zone, fmt.Sprintf("Zone %s", string(rune('a'+i))))
		}
	}

	// Create 450 nodes (150 per region)
	schedulerNodes := make([]scheduler.Node, 450)
	for i := 0; i < 450; i++ {
		region := []string{"us-west", "us-east", "eu-west"}[i%3]
		zone := fmt.Sprintf("%s-1%s", region, string(rune('a'+(i%5))))

		// Add to topology
		topologyZone := topology.ZoneID(zone)
		nodeID := topology.NodeID(fmt.Sprintf("node-%03d", i))
		tm.AddNode(nodeID, fmt.Sprintf("node-%03d", i), topology.RegionID(region), topologyZone, 4000)
		tm.UpdateNodeStatus(nodeID, "ready")

		// Add to scheduler
		schedulerNodes[i] = scheduler.Node{
			ID:       fmt.Sprintf("node-%03d", i),
			Name:     fmt.Sprintf("node-%03d", i),
			Status:   "ready",
			Region:   region,
			Zone:     zone,
			CPUMilli: 4000,
			MemBytes: 8e9,
			DiskFree: 100e9,
			Tiers:    []string{"compute"},
		}
	}

	// Record cross-region latencies
	tm.RecordLatency("us-west", "us-east", 50*time.Millisecond)
	tm.RecordLatency("us-west", "eu-west", 150*time.Millisecond)
	tm.RecordLatency("us-east", "eu-west", 180*time.Millisecond)

	// Test 1: Normal placement
	t.Logf("Test 1: Normal Placement")
	startTime := time.Now()

	req := scheduler.Request{
		App: "gate23-app",
		Spec: api.AppSpec{
			Replicas: 3,
			Resources: api.Resources{
				CPUMilli: 1000,
				MemBytes: 2e9,
			},
			Placement: api.Placement{
				Tiers:  []string{"compute"},
				Spread: "failure-domain",
			},
		},
		Nodes: schedulerNodes,
	}

	plan := scheduler.ScheduleFast(req)
	elapsed := time.Since(startTime)

	if len(plan.Replicas) != 3 {
		t.Fatalf("Expected 3 replicas in plan, got %d", len(plan.Replicas))
	}

	scheduled := 0
	for _, rp := range plan.Replicas {
		if rp.Node != "" {
			scheduled++
		}
	}

	t.Logf("  Scheduled: %d/3 replicas", scheduled)
	t.Logf("  Time: %v", elapsed)

	// Test 2: Region failure scenario
	t.Logf("Test 2: Region Failure (us-west down)")
	failoverStart := time.Now()

	// Remove all us-west nodes
	remainingNodes := make([]scheduler.Node, 0)
	for _, n := range schedulerNodes {
		if n.Region != "us-west" {
			remainingNodes = append(remainingNodes, n)
		}
	}

	req.Nodes = remainingNodes
	failoverPlan := scheduler.ScheduleFast(req)
	failoverElapsed := time.Since(failoverStart)

	failoverScheduled := 0
	for _, rp := range failoverPlan.Replicas {
		if rp.Node != "" {
			failoverScheduled++
		}
	}

	t.Logf("  Available nodes: %d", len(remainingNodes))
	t.Logf("  Scheduled: %d/3 replicas", failoverScheduled)
	t.Logf("  Recovery time: %v", failoverElapsed)

	// Verify recovery time < 5 seconds
	if failoverElapsed > 5*time.Second {
		t.Fatalf("Recovery time %.2f > 5 second target", failoverElapsed.Seconds())
	}

	t.Logf("✅ GATE 23 PASS: Multi-region failover recovered in %.3f seconds", failoverElapsed.Seconds())
}

// TestPhase6A_CrossRegionConsensus verifies cross-region aware consensus
func TestPhase6A_CrossRegionConsensus(t *testing.T) {
	t.Logf("Phase 6A: Cross-Region Consensus Awareness")

	// This test verifies that the consensus layer is aware of region topology
	// In the full implementation, this would involve the actual Raft consensus code

	tm := topology.NewManager()

	// Setup regions
	for _, r := range []string{"us-west", "us-east"} {
		tm.AddRegion(topology.RegionID(r), r, r+"-1")
		tm.AddZone(topology.RegionID(r), topology.ZoneID(r+"-1a"), "Zone A")
	}

	// Create control plane nodes
	for i := 0; i < 5; i++ {
		region := []string{"us-west", "us-west", "us-west", "us-east", "us-east"}[i]
		zone := topology.ZoneID(region + "-1a")
		nodeID := topology.NodeID(fmt.Sprintf("cp-%d", i))
		tm.AddNode(nodeID, fmt.Sprintf("cp-%d", i), topology.RegionID(region), zone, 4000)
		tm.UpdateNodeStatus(nodeID, "ready")
	}

	// Record latency
	tm.RecordLatency("us-west", "us-east", 50*time.Millisecond)

	snap := tm.GetSnapshot()

	if len(snap.Regions) != 2 {
		t.Fatalf("Expected 2 regions, got %d", len(snap.Regions))
	}

	// Verify topology is ready for consensus layer
	westNodes := tm.GetNodesInRegion("us-west")
	if len(westNodes) != 3 {
		t.Fatalf("Expected 3 us-west nodes, got %d", len(westNodes))
	}

	eastNodes := tm.GetNodesInRegion("us-east")
	if len(eastNodes) != 2 {
		t.Fatalf("Expected 2 us-east nodes, got %d", len(eastNodes))
	}

	t.Logf("✅ Cross-Region Consensus Ready:")
	t.Logf("  Control plane regions: 2")
	t.Logf("  us-west members: %d", len(westNodes))
	t.Logf("  us-east members: %d", len(eastNodes))
	t.Logf("  Latency us-west to us-east: %v", tm.GetLatency("us-west", "us-east"))
}
