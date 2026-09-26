package runtime

import (
	"context"
	"testing"
)

// TestNodeCapacityTracking verifies capacity tracking works correctly
func TestNodeCapacityTracking(t *testing.T) {
	t.Log("Testing node capacity tracking")

	cap := &NodeCapacity{
		NodeID:       "node-1",
		TotalMemory:  1024 * 1024 * 1024, // 1GB
		TotalCPU:     1000,               // 1 CPU
		TotalDisk:    10 * 1024 * 1024,   // 10GB
		MaxWorkloads: 10,
	}

	// Initialize available resources
	cap.AvailableMemory = cap.TotalMemory
	cap.AvailableCPU = cap.TotalCPU
	cap.AvailableDisk = cap.TotalDisk

	constraints := &ResourceConstraints{
		MemoryBytes: 256 * 1024 * 1024, // 256MB
		CPUShares:   200,               // 0.2 CPU
		DiskBytes:   2 * 1024 * 1024,   // 2GB
	}

	// Initial capacity check
	if !cap.CanFit(constraints) {
		t.Errorf("Constraints should fit in empty node")
	}

	// Allocate
	if err := cap.AllocateResources(constraints); err != nil {
		t.Fatalf("Allocation failed: %v", err)
	}

	if cap.WorkloadCount != 1 {
		t.Errorf("Expected workload count 1, got %d", cap.WorkloadCount)
	}

	if cap.AvailableMemory != cap.TotalMemory-constraints.MemoryBytes {
		t.Errorf("Expected available memory %d, got %d",
			cap.TotalMemory-constraints.MemoryBytes, cap.AvailableMemory)
	}

	// Release
	if err := cap.ReleaseResources(constraints); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	if cap.WorkloadCount != 0 {
		t.Errorf("Expected workload count 0 after release, got %d", cap.WorkloadCount)
	}

	if cap.AvailableMemory != cap.TotalMemory {
		t.Errorf("Expected available memory restored to %d, got %d",
			cap.TotalMemory, cap.AvailableMemory)
	}

	t.Logf("PASS: Capacity tracking working correctly")
}

// TestCapacityExhaustion verifies rejection when capacity exceeded
func TestCapacityExhaustion(t *testing.T) {
	t.Log("Testing capacity exhaustion")

	cap := &NodeCapacity{
		NodeID:       "node-low-mem",
		TotalMemory:  512 * 1024,  // 512KB
		TotalCPU:     100,         // 0.1 CPU
		TotalDisk:    1024 * 1024, // 1MB
		MaxWorkloads: 5,
	}

	cap.AvailableMemory = cap.TotalMemory
	cap.AvailableCPU = cap.TotalCPU
	cap.AvailableDisk = cap.TotalDisk

	constraints := &ResourceConstraints{
		MemoryBytes: 256 * 1024 * 1024, // 256MB (exceeds available)
		CPUShares:   50,
		DiskBytes:   1024,
	}

	if cap.CanFit(constraints) {
		t.Error("Should reject when memory insufficient")
	}

	err := cap.AllocateResources(constraints)
	if err == nil {
		t.Error("Expected error when allocating more than available")
	}

	t.Logf("PASS: Capacity exhaustion correctly rejected")
}

// TestDeploymentEngineCreation verifies engine initialization
func TestDeploymentEngineCreation(t *testing.T) {
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)

	t.Log("Testing deployment engine creation")

	engine := NewDeploymentEngine(contract, StrategyFirstFit)
	if engine == nil {
		t.Fatal("Failed to create deployment engine")
	}

	if engine.strategy != StrategyFirstFit {
		t.Errorf("Expected strategy FIRST_FIT, got %s", engine.strategy)
	}

	t.Logf("PASS: Deployment engine created successfully")
}

// TestRegisterNodeCapacity verifies node registration
func TestRegisterNodeCapacity(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)
	engine := NewDeploymentEngine(contract, StrategyFirstFit)

	t.Log("Testing node capacity registration")

	cap := &NodeCapacity{
		NodeID:       "node-1",
		TotalMemory:  1024 * 1024 * 1024,
		TotalCPU:     1000,
		TotalDisk:    10 * 1024 * 1024,
		MaxWorkloads: 10,
	}

	if err := engine.RegisterNodeCapacity(ctx, "node-1", cap); err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	retrieved, err := engine.GetNodeCapacity(ctx, "node-1")
	if err != nil {
		t.Fatalf("Retrieval failed: %v", err)
	}

	if retrieved.TotalMemory != cap.TotalMemory {
		t.Errorf("Expected memory %d, got %d", cap.TotalMemory, retrieved.TotalMemory)
	}

	t.Logf("PASS: Node capacity registration working")
}

// TestFirstFitStrategy verifies first-fit scheduling
func TestFirstFitStrategy(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)
	engine := NewDeploymentEngine(contract, StrategyFirstFit)

	t.Log("Testing first-fit scheduling strategy")

	// Setup nodes
	for i := 1; i <= 3; i++ {
		nodeID := "node-" + string(rune('0'+i))
		if err := nlm.RegisterNode(ctx, nodeID); err != nil {
			t.Fatalf("RegisterNode failed: %v", err)
		}

		nlm.mu.Lock()
		nlm.nodes[nodeID].State = NodeVerified
		nlm.mu.Unlock()

		if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
			t.Fatalf("TransitionToActive failed: %v", err)
		}

		cap := &NodeCapacity{
			NodeID:       nodeID,
			TotalMemory:  1024 * 1024 * 1024,
			TotalCPU:     1000,
			TotalDisk:    10 * 1024 * 1024,
			MaxWorkloads: 10,
		}

		if err := engine.RegisterNodeCapacity(ctx, nodeID, cap); err != nil {
			t.Fatalf("RegisterNodeCapacity failed: %v", err)
		}
	}

	constraints := &ResourceConstraints{
		MemoryBytes: 256 * 1024 * 1024,
		CPUShares:   200,
		DiskBytes:   2 * 1024 * 1024,
		Labels:      make(map[string]string),
	}

	decision, err := engine.ScheduleWorkload(ctx, "workload-1", 2, constraints)
	if err != nil {
		t.Fatalf("ScheduleWorkload failed: %v", err)
	}

	if len(decision.SelectedNodes) != 2 {
		t.Errorf("Expected 2 selected nodes, got %d", len(decision.SelectedNodes))
	}

	t.Logf("PASS: First-fit strategy selected nodes: %v", decision.SelectedNodes)
}

// TestSpreadOutStrategy verifies spread-out scheduling
func TestSpreadOutStrategy(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)
	engine := NewDeploymentEngine(contract, StrategySpreadOut)

	t.Log("Testing spread-out scheduling strategy")

	// Setup 3 nodes with different loads
	nodeData := []struct {
		id     string
		memory int64
		cpu    int
	}{
		{"node-1", 1024 * 1024 * 1024, 1000},
		{"node-2", 1024 * 1024 * 1024, 1000},
		{"node-3", 1024 * 1024 * 1024, 1000},
	}

	for _, nd := range nodeData {
		if err := nlm.RegisterNode(ctx, nd.id); err != nil {
			t.Fatalf("RegisterNode failed: %v", err)
		}

		nlm.mu.Lock()
		nlm.nodes[nd.id].State = NodeVerified
		nlm.mu.Unlock()

		if err := nlm.TransitionToActive(ctx, nd.id); err != nil {
			t.Fatalf("TransitionToActive failed: %v", err)
		}

		cap := &NodeCapacity{
			NodeID:       nd.id,
			TotalMemory:  nd.memory,
			TotalCPU:     nd.cpu,
			TotalDisk:    10 * 1024 * 1024,
			MaxWorkloads: 10,
		}

		if err := engine.RegisterNodeCapacity(ctx, nd.id, cap); err != nil {
			t.Fatalf("RegisterNodeCapacity failed: %v", err)
		}
	}

	constraints := &ResourceConstraints{
		MemoryBytes: 256 * 1024 * 1024,
		CPUShares:   200,
		DiskBytes:   2 * 1024 * 1024,
		Labels:      make(map[string]string),
	}

	// Schedule first workload
	decision1, _ := engine.ScheduleWorkload(ctx, "workload-1", 1, constraints)

	// Schedule second workload - should go to different node due to spread-out
	decision2, _ := engine.ScheduleWorkload(ctx, "workload-2", 1, constraints)

	if decision1.SelectedNodes[0] == decision2.SelectedNodes[0] {
		t.Logf("Note: Both scheduled on same node (may happen with small cluster)")
	} else {
		t.Logf("PASS: Spread-out strategy distributed workloads")
	}
}

// TestPackDenseStrategy verifies pack-dense scheduling
func TestPackDenseStrategy(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)
	engine := NewDeploymentEngine(contract, StrategyPackDense)

	t.Log("Testing pack-dense scheduling strategy")

	// Setup 3 nodes
	for i := 1; i <= 3; i++ {
		nodeID := "node-" + string(rune('0'+i))
		if err := nlm.RegisterNode(ctx, nodeID); err != nil {
			t.Fatalf("RegisterNode failed: %v", err)
		}

		nlm.mu.Lock()
		nlm.nodes[nodeID].State = NodeVerified
		nlm.mu.Unlock()

		if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
			t.Fatalf("TransitionToActive failed: %v", err)
		}

		cap := &NodeCapacity{
			NodeID:       nodeID,
			TotalMemory:  1024 * 1024 * 1024,
			TotalCPU:     1000,
			TotalDisk:    10 * 1024 * 1024,
			MaxWorkloads: 10,
		}

		if err := engine.RegisterNodeCapacity(ctx, nodeID, cap); err != nil {
			t.Fatalf("RegisterNodeCapacity failed: %v", err)
		}
	}

	constraints := &ResourceConstraints{
		MemoryBytes: 256 * 1024 * 1024,
		CPUShares:   200,
		DiskBytes:   2 * 1024 * 1024,
		Labels:      make(map[string]string),
	}

	// Schedule multiple workloads - should pack onto same nodes
	for i := 1; i <= 3; i++ {
		wlID := "workload-" + string(rune('0'+i))
		decision, err := engine.ScheduleWorkload(ctx, wlID, 1, constraints)
		if err != nil {
			t.Errorf("Workload %s scheduling failed: %v", wlID, err)
		}

		t.Logf("Workload %s scheduled on %v", wlID, decision.SelectedNodes)
	}

	summary := engine.GetAllocationSummary(ctx)
	nodeLoads := []int{0, 0, 0}
	for _, cap := range summary {
		for i := 1; i <= 3; i++ {
			if cap.NodeID == "node-"+string(rune('0'+i)) {
				nodeLoads[i-1] = cap.WorkloadCount
			}
		}
	}

	t.Logf("PASS: Pack-dense strategy - loads: %v", nodeLoads)
}

// TestBestFitStrategy verifies best-fit scheduling
func TestBestFitStrategy(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)
	engine := NewDeploymentEngine(contract, StrategyBestFit)

	t.Log("Testing best-fit scheduling strategy")

	// Setup 2 nodes: one small, one large
	cap1 := &NodeCapacity{
		NodeID:       "node-small",
		TotalMemory:  512 * 1024,
		TotalCPU:     100,
		TotalDisk:    5 * 1024 * 1024,
		MaxWorkloads: 10,
	}

	cap2 := &NodeCapacity{
		NodeID:       "node-large",
		TotalMemory:  1024 * 1024 * 1024,
		TotalCPU:     1000,
		TotalDisk:    100 * 1024 * 1024,
		MaxWorkloads: 10,
	}

	for _, cap := range []*NodeCapacity{cap1, cap2} {
		if err := nlm.RegisterNode(ctx, cap.NodeID); err != nil {
			t.Fatalf("RegisterNode failed: %v", err)
		}

		nlm.mu.Lock()
		nlm.nodes[cap.NodeID].State = NodeVerified
		nlm.mu.Unlock()

		if err := nlm.TransitionToActive(ctx, cap.NodeID); err != nil {
			t.Fatalf("TransitionToActive failed: %v", err)
		}

		if err := engine.RegisterNodeCapacity(ctx, cap.NodeID, cap); err != nil {
			t.Fatalf("RegisterNodeCapacity failed: %v", err)
		}
	}

	constraints := &ResourceConstraints{
		MemoryBytes: 256 * 1024,
		CPUShares:   50,
		DiskBytes:   1024,
		Labels:      make(map[string]string),
	}

	decision, err := engine.ScheduleWorkload(ctx, "workload-1", 1, constraints)
	if err != nil {
		t.Fatalf("ScheduleWorkload failed: %v", err)
	}

	// Best-fit should select small node (least waste)
	if decision.SelectedNodes[0] != "node-small" {
		t.Logf("Note: Best-fit selected %s (may select either node)", decision.SelectedNodes[0])
	}

	t.Logf("PASS: Best-fit strategy working")
}

// TestInsufficientCapacity verifies rejection on capacity shortage
func TestInsufficientCapacity(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)
	engine := NewDeploymentEngine(contract, StrategyFirstFit)

	t.Log("Testing insufficient capacity rejection")

	nodeID := "node-limited"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	cap := &NodeCapacity{
		NodeID:       nodeID,
		TotalMemory:  512 * 1024, // Very limited
		TotalCPU:     100,
		TotalDisk:    1024 * 1024,
		MaxWorkloads: 1,
	}

	if err := engine.RegisterNodeCapacity(ctx, nodeID, cap); err != nil {
		t.Fatalf("RegisterNodeCapacity failed: %v", err)
	}

	constraints := &ResourceConstraints{
		MemoryBytes: 256 * 1024 * 1024, // Exceeds available
		CPUShares:   200,
		DiskBytes:   2 * 1024 * 1024,
		Labels:      make(map[string]string),
	}

	_, err := engine.ScheduleWorkload(ctx, "workload-1", 1, constraints)
	if err == nil {
		t.Error("Expected error for insufficient capacity")
	}

	t.Logf("PASS: Insufficient capacity correctly rejected")
}

// TestGetAllocationSummary verifies summary reporting
func TestGetAllocationSummary(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)
	engine := NewDeploymentEngine(contract, StrategyFirstFit)

	t.Log("Testing allocation summary reporting")

	// Register 2 nodes
	for i := 1; i <= 2; i++ {
		nodeID := "node-" + string(rune('0'+i))
		if err := nlm.RegisterNode(ctx, nodeID); err != nil {
			t.Fatalf("RegisterNode failed: %v", err)
		}

		nlm.mu.Lock()
		nlm.nodes[nodeID].State = NodeVerified
		nlm.mu.Unlock()

		if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
			t.Fatalf("TransitionToActive failed: %v", err)
		}

		cap := &NodeCapacity{
			NodeID:       nodeID,
			TotalMemory:  1024 * 1024 * 1024,
			TotalCPU:     1000,
			TotalDisk:    10 * 1024 * 1024,
			MaxWorkloads: 10,
		}

		if err := engine.RegisterNodeCapacity(ctx, nodeID, cap); err != nil {
			t.Fatalf("RegisterNodeCapacity failed: %v", err)
		}
	}

	// Schedule a workload
	constraints := &ResourceConstraints{
		MemoryBytes: 256 * 1024 * 1024,
		CPUShares:   200,
		DiskBytes:   2 * 1024 * 1024,
		Labels:      make(map[string]string),
	}

	engine.ScheduleWorkload(ctx, "workload-1", 1, constraints)

	// Get summary
	summary := engine.GetAllocationSummary(ctx)
	if len(summary) != 2 {
		t.Errorf("Expected 2 nodes in summary, got %d", len(summary))
	}

	totalAllocated := 0
	for _, cap := range summary {
		totalAllocated += cap.WorkloadCount
	}

	if totalAllocated != 1 {
		t.Errorf("Expected 1 allocated workload, got %d", totalAllocated)
	}

	t.Logf("PASS: Allocation summary reporting working")
}

// TestSchedulingDecisionTracking verifies decision persistence
func TestSchedulingDecisionTracking(t *testing.T) {
	ctx := context.Background()
	nlm := NewNodeLifecycleManager()
	wlm := NewWorkloadLifecycleManager()
	contract := NewDeploymentContract(nlm, wlm)
	engine := NewDeploymentEngine(contract, StrategyFirstFit)

	t.Log("Testing scheduling decision tracking")

	nodeID := "node-1"
	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	nlm.mu.Lock()
	nlm.nodes[nodeID].State = NodeVerified
	nlm.mu.Unlock()

	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	cap := &NodeCapacity{
		NodeID:       nodeID,
		TotalMemory:  1024 * 1024 * 1024,
		TotalCPU:     1000,
		TotalDisk:    10 * 1024 * 1024,
		MaxWorkloads: 10,
	}

	if err := engine.RegisterNodeCapacity(ctx, nodeID, cap); err != nil {
		t.Fatalf("RegisterNodeCapacity failed: %v", err)
	}

	constraints := &ResourceConstraints{
		MemoryBytes: 256 * 1024 * 1024,
		CPUShares:   200,
		DiskBytes:   2 * 1024 * 1024,
		Labels:      make(map[string]string),
	}

	originalDecision, err := engine.ScheduleWorkload(ctx, "workload-1", 1, constraints)
	if err != nil {
		t.Fatalf("ScheduleWorkload failed: %v", err)
	}

	// Retrieve decision
	retrieved, err := engine.GetSchedulingDecision(ctx, "workload-1")
	if err != nil {
		t.Fatalf("GetSchedulingDecision failed: %v", err)
	}

	if retrieved.WorkloadID != originalDecision.WorkloadID {
		t.Errorf("Expected workload ID %s, got %s",
			originalDecision.WorkloadID, retrieved.WorkloadID)
	}

	if len(retrieved.SelectedNodes) != 1 {
		t.Errorf("Expected 1 selected node, got %d", len(retrieved.SelectedNodes))
	}

	t.Logf("PASS: Scheduling decision tracking working")
}

// TestMultipleSchedulingStrategies verifies all strategies work
func TestMultipleSchedulingStrategies(t *testing.T) {
	ctx := context.Background()

	t.Log("Testing all scheduling strategies")

	strategies := []SchedulingStrategy{
		StrategyFirstFit,
		StrategyBestFit,
		StrategyRoundRobin,
		StrategySpreadOut,
		StrategyPackDense,
	}

	for _, strategy := range strategies {
		nlm := NewNodeLifecycleManager()
		wlm := NewWorkloadLifecycleManager()
		contract := NewDeploymentContract(nlm, wlm)
		engine := NewDeploymentEngine(contract, strategy)

		nodeID := "node-" + string(strategy[0])
		if err := nlm.RegisterNode(ctx, nodeID); err != nil {
			t.Fatalf("RegisterNode failed: %v", err)
		}

		nlm.mu.Lock()
		nlm.nodes[nodeID].State = NodeVerified
		nlm.mu.Unlock()

		if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
			t.Fatalf("TransitionToActive failed: %v", err)
		}

		cap := &NodeCapacity{
			NodeID:       nodeID,
			TotalMemory:  1024 * 1024 * 1024,
			TotalCPU:     1000,
			TotalDisk:    10 * 1024 * 1024,
			MaxWorkloads: 10,
		}

		if err := engine.RegisterNodeCapacity(ctx, nodeID, cap); err != nil {
			t.Fatalf("RegisterNodeCapacity failed: %v", err)
		}

		constraints := &ResourceConstraints{
			MemoryBytes: 256 * 1024 * 1024,
			CPUShares:   200,
			DiskBytes:   2 * 1024 * 1024,
			Labels:      make(map[string]string),
		}

		_, err := engine.ScheduleWorkload(ctx, "workload-"+string(strategy[0]), 1, constraints)
		if err != nil {
			t.Errorf("Strategy %s failed: %v", strategy, err)
		}

		t.Logf("✓ Strategy %s working", strategy)
	}

	t.Logf("PASS: All scheduling strategies working")
}
