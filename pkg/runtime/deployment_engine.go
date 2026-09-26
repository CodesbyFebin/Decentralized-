package runtime

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// SchedulingStrategy determines node selection algorithm
type SchedulingStrategy string

const (
	StrategyFirstFit    SchedulingStrategy = "FIRST_FIT"
	StrategyBestFit     SchedulingStrategy = "BEST_FIT"
	StrategyRoundRobin  SchedulingStrategy = "ROUND_ROBIN"
	StrategySpreadOut   SchedulingStrategy = "SPREAD_OUT"
	StrategyPackDense   SchedulingStrategy = "PACK_DENSE"
)

// ResourceConstraints specifies workload resource requirements
type ResourceConstraints struct {
	MemoryBytes int64              // Memory requirement in bytes
	CPUShares   int                // CPU shares (1000 = 1 CPU)
	DiskBytes   int64              // Disk requirement in bytes
	Labels      map[string]string  // Workload placement labels
}

// NodeCapacity tracks available resources on a node
type NodeCapacity struct {
	NodeID         string
	TotalMemory    int64
	AllocatedMem   int64
	AvailableMemory int64
	TotalCPU       int
	AllocatedCPU   int
	AvailableCPU   int
	TotalDisk      int64
	AllocatedDisk  int64
	AvailableDisk  int64
	WorkloadCount  int
	MaxWorkloads   int
}

// AvailableCapacity returns remaining allocatable resources
func (nc *NodeCapacity) AvailableCapacity() (int64, int, int64) {
	return nc.AvailableMemory, nc.AvailableCPU, nc.AvailableDisk
}

// CanFit checks if node has capacity for constraints
func (nc *NodeCapacity) CanFit(constraints *ResourceConstraints) bool {
	if nc.WorkloadCount >= nc.MaxWorkloads {
		return false
	}

	if constraints.MemoryBytes > nc.AvailableMemory {
		return false
	}

	if constraints.CPUShares > nc.AvailableCPU {
		return false
	}

	if constraints.DiskBytes > nc.AvailableDisk {
		return false
	}

	return true
}

// AllocateResources reserves resources on node
func (nc *NodeCapacity) AllocateResources(constraints *ResourceConstraints) error {
	if !nc.CanFit(constraints) {
		return fmt.Errorf("insufficient capacity: mem=%d/%d, cpu=%d/%d, disk=%d/%d",
			constraints.MemoryBytes, nc.AvailableMemory,
			constraints.CPUShares, nc.AvailableCPU,
			constraints.DiskBytes, nc.AvailableDisk)
	}

	nc.AllocatedMem += constraints.MemoryBytes
	nc.AvailableMemory -= constraints.MemoryBytes

	nc.AllocatedCPU += constraints.CPUShares
	nc.AvailableCPU -= constraints.CPUShares

	nc.AllocatedDisk += constraints.DiskBytes
	nc.AvailableDisk -= constraints.DiskBytes

	nc.WorkloadCount++

	return nil
}

// ReleaseResources frees reserved resources
func (nc *NodeCapacity) ReleaseResources(constraints *ResourceConstraints) error {
	if nc.AllocatedMem < constraints.MemoryBytes {
		return fmt.Errorf("cannot release more memory than allocated")
	}

	nc.AllocatedMem -= constraints.MemoryBytes
	nc.AvailableMemory += constraints.MemoryBytes

	nc.AllocatedCPU -= constraints.CPUShares
	nc.AvailableCPU += constraints.CPUShares

	nc.AllocatedDisk -= constraints.DiskBytes
	nc.AvailableDisk += constraints.DiskBytes

	if nc.WorkloadCount > 0 {
		nc.WorkloadCount--
	}

	return nil
}

// SchedulingDecision tracks placement decision
type SchedulingDecision struct {
	WorkloadID     string
	SelectedNodes  []string
	AlternateNodes []string
	Strategy       SchedulingStrategy
	DecisionTime   int64
	Reason         string
}

// DeploymentEngine orchestrates workload scheduling
type DeploymentEngine struct {
	contract      *DeploymentContract
	nlm           *NodeLifecycleManager
	wlm           *WorkloadLifecycleManager
	capacities    map[string]*NodeCapacity
	decisions     map[string]*SchedulingDecision
	strategy      SchedulingStrategy
	mu            sync.RWMutex
}

// NewDeploymentEngine creates a new deployment engine
func NewDeploymentEngine(contract *DeploymentContract, strategy SchedulingStrategy) *DeploymentEngine {
	return &DeploymentEngine{
		contract:   contract,
		nlm:        contract.validator.nlm,
		wlm:        contract.validator.wlm,
		capacities: make(map[string]*NodeCapacity),
		decisions:  make(map[string]*SchedulingDecision),
		strategy:   strategy,
	}
}

// RegisterNodeCapacity registers capacity for a node
func (de *DeploymentEngine) RegisterNodeCapacity(ctx context.Context, nodeID string, capacity *NodeCapacity) error {
	de.mu.Lock()
	defer de.mu.Unlock()

	if capacity.MaxWorkloads == 0 {
		capacity.MaxWorkloads = 100
	}

	capacity.AvailableMemory = capacity.TotalMemory
	capacity.AvailableCPU = capacity.TotalCPU
	capacity.AvailableDisk = capacity.TotalDisk

	de.capacities[nodeID] = capacity
	return nil
}

// GetNodeCapacity retrieves node capacity
func (de *DeploymentEngine) GetNodeCapacity(ctx context.Context, nodeID string) (*NodeCapacity, error) {
	de.mu.RLock()
	defer de.mu.RUnlock()

	cap, ok := de.capacities[nodeID]
	if !ok {
		return nil, fmt.Errorf("node %s capacity not registered", nodeID)
	}

	return cap, nil
}

// SelectNodesWithConstraints selects nodes based on resource constraints
func (de *DeploymentEngine) SelectNodesWithConstraints(ctx context.Context,
	replicas int, constraints *ResourceConstraints) ([]string, []string, error) {

	de.mu.RLock()
	defer de.mu.RUnlock()

	if replicas < 1 {
		return nil, nil, fmt.Errorf("replicas must be at least 1")
	}

	// Get eligible nodes (not cordoned, offline, etc.)
	eligible := []string{}
	de.nlm.mu.RLock()
	for nodeID := range de.nlm.nodes {
		de.nlm.mu.RUnlock()

		nodeState, err := de.nlm.GetNodeState(ctx, nodeID)
		if err != nil {
			de.nlm.mu.RLock()
			continue
		}

		// Filter: only ACTIVE nodes initially
		if nodeState.State != NodeActive {
			de.nlm.mu.RLock()
			continue
		}

		eligible = append(eligible, nodeID)
		de.nlm.mu.RLock()
	}
	de.nlm.mu.RUnlock()

	// Filter by capacity
	capable := []string{}
	for _, nodeID := range eligible {
		cap, ok := de.capacities[nodeID]
		if !ok {
			continue
		}

		if cap.CanFit(constraints) {
			capable = append(capable, nodeID)
		}
	}

	selected := []string{}
	alternates := []string{}

	// Apply scheduling strategy
	switch de.strategy {
	case StrategyFirstFit:
		selected, alternates = de.selectFirstFit(capable, replicas, constraints)

	case StrategyBestFit:
		selected, alternates = de.selectBestFit(capable, replicas, constraints)

	case StrategySpreadOut:
		selected, alternates = de.selectSpreadOut(capable, replicas, constraints)

	case StrategyPackDense:
		selected, alternates = de.selectPackDense(capable, replicas, constraints)

	case StrategyRoundRobin:
		fallthrough
	default:
		selected, alternates = de.selectFirstFit(capable, replicas, constraints)
	}

	if len(selected) < replicas {
		return nil, nil, fmt.Errorf("insufficient nodes: found %d, need %d", len(selected), replicas)
	}

	return selected[:replicas], alternates, nil
}

// selectFirstFit: select first nodes with capacity
func (de *DeploymentEngine) selectFirstFit(nodes []string, replicas int,
	constraints *ResourceConstraints) ([]string, []string) {

	selected := []string{}
	alternates := []string{}

	for _, nodeID := range nodes {
		cap, ok := de.capacities[nodeID]
		if !ok {
			continue
		}

		if cap.CanFit(constraints) {
			selected = append(selected, nodeID)
		} else {
			alternates = append(alternates, nodeID)
		}

		if len(selected) >= replicas {
			alternates = append(alternates, nodes[len(selected):]...)
			break
		}
	}

	return selected, alternates
}

// selectBestFit: select nodes with least waste
func (de *DeploymentEngine) selectBestFit(nodes []string, replicas int,
	constraints *ResourceConstraints) ([]string, []string) {

	type nodeUtilization struct {
		nodeID string
		waste  int64
	}

	candidates := []nodeUtilization{}
	for _, nodeID := range nodes {
		cap, ok := de.capacities[nodeID]
		if !ok {
			continue
		}

		if !cap.CanFit(constraints) {
			continue
		}

		// Calculate waste (unused after allocation)
		remainMem := cap.AvailableMemory - constraints.MemoryBytes
		remainCPU := cap.AvailableCPU - constraints.CPUShares
		remainDisk := cap.AvailableDisk - constraints.DiskBytes

		waste := remainMem + (int64(remainCPU) * 1000) + remainDisk
		candidates = append(candidates, nodeUtilization{nodeID, waste})
	}

	// Sort by least waste (best fit)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].waste < candidates[j].waste
	})

	selected := []string{}
	for _, cu := range candidates {
		selected = append(selected, cu.nodeID)
		if len(selected) >= replicas {
			break
		}
	}

	alternates := []string{}
	for i := len(selected); i < len(candidates); i++ {
		alternates = append(alternates, candidates[i].nodeID)
	}

	return selected, alternates
}

// selectSpreadOut: spread workloads across nodes
func (de *DeploymentEngine) selectSpreadOut(nodes []string, replicas int,
	constraints *ResourceConstraints) ([]string, []string) {

	type nodeLoad struct {
		nodeID string
		load   int
	}

	candidates := []nodeLoad{}
	for _, nodeID := range nodes {
		cap, ok := de.capacities[nodeID]
		if !ok {
			continue
		}

		if !cap.CanFit(constraints) {
			continue
		}

		candidates = append(candidates, nodeLoad{nodeID, cap.WorkloadCount})
	}

	// Sort by least load (spread out)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].load < candidates[j].load
	})

	selected := []string{}
	for _, cl := range candidates {
		selected = append(selected, cl.nodeID)
		if len(selected) >= replicas {
			break
		}
	}

	alternates := []string{}
	for i := len(selected); i < len(candidates); i++ {
		alternates = append(alternates, candidates[i].nodeID)
	}

	return selected, alternates
}

// selectPackDense: pack workloads densely
func (de *DeploymentEngine) selectPackDense(nodes []string, replicas int,
	constraints *ResourceConstraints) ([]string, []string) {

	type nodeLoad struct {
		nodeID string
		load   int
	}

	candidates := []nodeLoad{}
	for _, nodeID := range nodes {
		cap, ok := de.capacities[nodeID]
		if !ok {
			continue
		}

		if !cap.CanFit(constraints) {
			continue
		}

		candidates = append(candidates, nodeLoad{nodeID, cap.WorkloadCount})
	}

	// Sort by most load (pack dense)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].load > candidates[j].load
	})

	selected := []string{}
	for _, cl := range candidates {
		selected = append(selected, cl.nodeID)
		if len(selected) >= replicas {
			break
		}
	}

	alternates := []string{}
	for i := len(selected); i < len(candidates); i++ {
		alternates = append(alternates, candidates[i].nodeID)
	}

	return selected, alternates
}

// ScheduleWorkload schedules a workload with resource constraints
func (de *DeploymentEngine) ScheduleWorkload(ctx context.Context,
	workloadID string, replicas int, constraints *ResourceConstraints) (*SchedulingDecision, error) {

	if workloadID == "" {
		return nil, fmt.Errorf("workload ID cannot be empty")
	}

	if constraints == nil {
		constraints = &ResourceConstraints{
			MemoryBytes: 1024 * 1024,      // 1MB default
			CPUShares:   100,               // 0.1 CPU default
			DiskBytes:   10 * 1024 * 1024, // 10MB default
			Labels:      make(map[string]string),
		}
	}

	// Select nodes with constraints
	selected, alternates, err := de.SelectNodesWithConstraints(ctx, replicas, constraints)
	if err != nil {
		return nil, fmt.Errorf("node selection failed: %v", err)
	}

	// Allocate resources on selected nodes
	de.mu.Lock()
	for _, nodeID := range selected {
		cap := de.capacities[nodeID]
		if err := cap.AllocateResources(constraints); err != nil {
			de.mu.Unlock()
			return nil, fmt.Errorf("resource allocation failed on %s: %v", nodeID, err)
		}
	}
	de.mu.Unlock()

	// Create workload in lifecycle manager
	if err := de.wlm.CreateWorkload(ctx, workloadID); err != nil {
		// Rollback allocations
		de.mu.Lock()
		for _, nodeID := range selected {
			cap := de.capacities[nodeID]
			cap.ReleaseResources(constraints)
		}
		de.mu.Unlock()

		return nil, fmt.Errorf("workload creation failed: %v", err)
	}

	// Record scheduling decision
	decision := &SchedulingDecision{
		WorkloadID:     workloadID,
		SelectedNodes:  selected,
		AlternateNodes: alternates,
		Strategy:       de.strategy,
		DecisionTime:   time.Now().UnixNano(),
		Reason:         fmt.Sprintf("Scheduled via %s strategy", de.strategy),
	}

	de.mu.Lock()
	de.decisions[workloadID] = decision
	de.mu.Unlock()

	return decision, nil
}

// GetSchedulingDecision retrieves a scheduling decision
func (de *DeploymentEngine) GetSchedulingDecision(ctx context.Context, workloadID string) (*SchedulingDecision, error) {
	de.mu.RLock()
	defer de.mu.RUnlock()

	decision, ok := de.decisions[workloadID]
	if !ok {
		return nil, fmt.Errorf("no scheduling decision for workload %s", workloadID)
	}

	return decision, nil
}

// RescheduleWorkload moves workload to different nodes
func (de *DeploymentEngine) RescheduleWorkload(ctx context.Context,
	workloadID string, constraints *ResourceConstraints) (*SchedulingDecision, error) {

	de.mu.RLock()
	oldDecision, ok := de.decisions[workloadID]
	de.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no previous scheduling decision for workload %s", workloadID)
	}

	// Release old allocations
	de.mu.Lock()
	for _, nodeID := range oldDecision.SelectedNodes {
		if cap, ok := de.capacities[nodeID]; ok {
			cap.ReleaseResources(constraints)
		}
	}
	de.mu.Unlock()

	// Schedule on new nodes
	return de.ScheduleWorkload(ctx, workloadID, len(oldDecision.SelectedNodes), constraints)
}

// GetAllocationSummary returns current resource allocation across nodes
func (de *DeploymentEngine) GetAllocationSummary(ctx context.Context) map[string]*NodeCapacity {
	de.mu.RLock()
	defer de.mu.RUnlock()

	summary := make(map[string]*NodeCapacity)
	for nodeID, cap := range de.capacities {
		// Return copy to prevent external modification
		capCopy := *cap
		summary[nodeID] = &capCopy
	}

	return summary
}
