package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// NodeLifecycleState represents the observed state of a node
type NodeLifecycleState string

const (
	NodeDiscovered  NodeLifecycleState = "DISCOVERED"
	NodeEnrolling   NodeLifecycleState = "ENROLLING"
	NodeVerified    NodeLifecycleState = "VERIFIED"
	NodeActive      NodeLifecycleState = "ACTIVE"
	NodeCordoned    NodeLifecycleState = "CORDONED"
	NodeDraining    NodeLifecycleState = "DRAINING"
	NodeIdle        NodeLifecycleState = "IDLE"
	NodeDegraded    NodeLifecycleState = "DEGRADED"
	NodeOffline     NodeLifecycleState = "OFFLINE"
	NodeRevoked     NodeLifecycleState = "REVOKED"
	NodeUnknown     NodeLifecycleState = "UNKNOWN"
)

// NodeStateValue represents the complete observed state with provenance
type NodeStateValue struct {
	NodeID        string
	State         NodeLifecycleState
	DesiredState  NodeLifecycleState // Desired state for reconciliation
	ObservedAt    int64              // Unix nanoseconds
	SourceID      string             // control-plane, agent, audit
	Freshness     string             // FRESH, STALE, EXPIRED, UNREACHABLE
	Cordoned      bool               // admin cordon flag
	DrainTarget   int                // target workload count during drain (0 = fully drained)
	LastHeartbeat int64
	Generation    int64 // Version counter for optimistic updates
	Reason        string // human-readable state reason
}

// WorkloadLifecycleState represents observed workload execution state
type WorkloadLifecycleState string

const (
	WorkloadDesired    WorkloadLifecycleState = "DESIRED"
	WorkloadScheduled  WorkloadLifecycleState = "SCHEDULED"
	WorkloadStarting   WorkloadLifecycleState = "STARTING"
	WorkloadRunning    WorkloadLifecycleState = "RUNNING"
	WorkloadHealthy    WorkloadLifecycleState = "HEALTHY"
	WorkloadUnhealthy  WorkloadLifecycleState = "UNHEALTHY"
	WorkloadStopping   WorkloadLifecycleState = "STOPPING"
	WorkloadStopped    WorkloadLifecycleState = "STOPPED"
	WorkloadFailed     WorkloadLifecycleState = "FAILED"
	WorkloadUnknown    WorkloadLifecycleState = "UNKNOWN"
)

// WorkloadStateValue represents complete observed workload state with provenance
type WorkloadStateValue struct {
	WorkloadID   string
	DesiredState WorkloadLifecycleState
	ObservedState WorkloadLifecycleState
	ObservedAt   int64 // Unix nanoseconds
	SourceID     string // agent, control-plane
	Freshness    string // FRESH, STALE, EXPIRED, UNREACHABLE
	ExitCode     int
	LastError    string
	HealthStatus string // from health checks
	Reason       string
}

// NodeLifecycleManager manages node state transitions with persistent storage
type NodeLifecycleManager struct {
	nodes map[string]*NodeStateValue
	mu    sync.RWMutex

	heartbeatTimeout  time.Duration
	staleThreshold    time.Duration
	stateStore        *NodeStateStore        // Persistent storage
	reconStore        *ReconciliationStore   // Reconciliation tracking
}

// NewNodeLifecycleManager creates a new node lifecycle manager
func NewNodeLifecycleManager() *NodeLifecycleManager {
	return NewNodeLifecycleManagerWithStore("", "")
}

// NewNodeLifecycleManagerWithStore creates a lifecycle manager with persistent storage
func NewNodeLifecycleManagerWithStore(stateStorePath, reconStorePath string) *NodeLifecycleManager {
	stateStore, _ := NewNodeStateStore(stateStorePath)
	reconStore, _ := NewReconciliationStore(reconStorePath)

	return &NodeLifecycleManager{
		nodes:             make(map[string]*NodeStateValue),
		heartbeatTimeout:  30 * time.Second,
		staleThreshold:    60 * time.Second,
		stateStore:        stateStore,
		reconStore:        reconStore,
	}
}

// RegisterNode adds a node to the lifecycle manager with persistent storage
func (nlm *NodeLifecycleManager) RegisterNode(ctx context.Context, nodeID string) error {
	nlm.mu.Lock()
	defer nlm.mu.Unlock()

	if _, exists := nlm.nodes[nodeID]; exists {
		return fmt.Errorf("node %s already registered", nodeID)
	}

	nodeState := &NodeStateValue{
		NodeID:        nodeID,
		State:         NodeDiscovered,
		DesiredState:  NodeDiscovered,
		ObservedAt:    time.Now().UnixNano(),
		Generation:    1,
		SourceID:      "agent",
		Freshness:     "FRESH",
		Cordoned:      false,
		LastHeartbeat: time.Now().UnixNano(),
		Reason:        "node discovered",
	}

	nlm.nodes[nodeID] = nodeState

	// Persist to disk
	if nlm.stateStore != nil {
		persistState := &PersistentNodeState{
			NodeID:        nodeState.NodeID,
			State:         string(nodeState.State),
			DesiredState:  string(nodeState.DesiredState),
			ObservedAt:    nodeState.ObservedAt,
			SourceID:      nodeState.SourceID,
			Freshness:     nodeState.Freshness,
			Cordoned:      nodeState.Cordoned,
			DrainTarget:   nodeState.DrainTarget,
			LastHeartbeat: nodeState.LastHeartbeat,
			Generation:    nodeState.Generation,
			Reason:        nodeState.Reason,
		}
		if err := nlm.stateStore.SaveNodeState(nodeID, persistState); err != nil {
			return fmt.Errorf("failed to persist node state: %v", err)
		}
	}

	return nil
}

// UpdateHeartbeat marks a node as having heartbeated
func (nlm *NodeLifecycleManager) UpdateHeartbeat(ctx context.Context, nodeID string) error {
	nlm.mu.Lock()
	defer nlm.mu.Unlock()

	node, exists := nlm.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.LastHeartbeat = time.Now().UnixNano()
	node.Freshness = "FRESH"

	// If was offline, bring back to active
	if node.State == NodeOffline {
		node.State = NodeActive
		node.Reason = "reconnected after offline"
	}

	return nil
}

// RecoverNodeStatesFromDisk loads previously persisted node states
func (nlm *NodeLifecycleManager) RecoverNodeStatesFromDisk(ctx context.Context) error {
	nlm.mu.Lock()
	defer nlm.mu.Unlock()

	if nlm.stateStore == nil {
		return fmt.Errorf("state store not configured")
	}

	states, err := nlm.stateStore.LoadAllNodeStates()
	if err != nil {
		return err
	}

	for nodeID, persistedState := range states {
		nlm.nodes[nodeID] = &NodeStateValue{
			NodeID:        persistedState.NodeID,
			State:         NodeLifecycleState(persistedState.State),
			DesiredState:  NodeLifecycleState(persistedState.DesiredState),
			ObservedAt:    persistedState.ObservedAt,
			SourceID:      persistedState.SourceID,
			Freshness:     persistedState.Freshness,
			Cordoned:      persistedState.Cordoned,
			DrainTarget:   persistedState.DrainTarget,
			LastHeartbeat: persistedState.LastHeartbeat,
			Generation:    persistedState.Generation,
			Reason:        persistedState.Reason,
		}
	}

	return nil
}

// persistNodeState saves node state to disk
func (nlm *NodeLifecycleManager) persistNodeState(node *NodeStateValue) error {
	if nlm.stateStore == nil {
		return nil
	}

	persistState := &PersistentNodeState{
		NodeID:        node.NodeID,
		State:         string(node.State),
		DesiredState:  string(node.DesiredState),
		ObservedAt:    node.ObservedAt,
		SourceID:      node.SourceID,
		Freshness:     node.Freshness,
		Cordoned:      node.Cordoned,
		DrainTarget:   node.DrainTarget,
		LastHeartbeat: node.LastHeartbeat,
		Generation:    node.Generation,
		Reason:        node.Reason,
	}
	return nlm.stateStore.SaveNodeState(node.NodeID, persistState)
}

// ReconcileNodeState tracks divergence between desired and observed state
func (nlm *NodeLifecycleManager) ReconcileNodeState(ctx context.Context, nodeID string) error {
	nlm.mu.RLock()
	node, exists := nlm.nodes[nodeID]
	nlm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	if nlm.reconStore != nil {
		return nlm.reconStore.RecordReconciliation(nodeID, string(node.DesiredState), string(node.State))
	}
	return nil
}

// TransitionToActive moves a node to ACTIVE state
func (nlm *NodeLifecycleManager) TransitionToActive(ctx context.Context, nodeID string) error {
	nlm.mu.Lock()
	defer nlm.mu.Unlock()

	node, exists := nlm.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	if node.State == NodeVerified || node.State == NodeIdle {
		node.State = NodeActive
		node.DesiredState = NodeActive
		node.ObservedAt = time.Now().UnixNano()
		node.Generation++
		node.Reason = "transitioned to ACTIVE"
		return nlm.persistNodeState(node)
	}

	return fmt.Errorf("cannot transition from %s to ACTIVE", node.State)
}

// CordonNode prevents new workloads from being scheduled
func (nlm *NodeLifecycleManager) CordonNode(ctx context.Context, nodeID string) error {
	nlm.mu.Lock()
	defer nlm.mu.Unlock()

	node, exists := nlm.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.Cordoned = true
	node.State = NodeCordoned
	node.DesiredState = NodeCordoned
	node.ObservedAt = time.Now().UnixNano()
	node.Generation++
	node.Reason = "cordoned by operator"

	return nlm.persistNodeState(node)
}

// DrainNode starts graceful workload drainage
func (nlm *NodeLifecycleManager) DrainNode(ctx context.Context, nodeID string, currentWorkloadCount int) error {
	nlm.mu.Lock()
	defer nlm.mu.Unlock()

	node, exists := nlm.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.State = NodeDraining
	node.DesiredState = NodeDraining
	node.DrainTarget = currentWorkloadCount
	node.ObservedAt = time.Now().UnixNano()
	node.Generation++
	node.Reason = fmt.Sprintf("draining %d workloads", currentWorkloadCount)

	return nlm.persistNodeState(node)
}

// CompleteDrain marks drainage as complete
func (nlm *NodeLifecycleManager) CompleteDrain(ctx context.Context, nodeID string) error {
	nlm.mu.Lock()
	defer nlm.mu.Unlock()

	node, exists := nlm.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	if node.State != NodeDraining {
		return fmt.Errorf("node is not draining")
	}

	node.State = NodeIdle
	node.DesiredState = NodeIdle
	node.DrainTarget = 0
	node.ObservedAt = time.Now().UnixNano()
	node.Generation++
	node.Reason = "drain complete"

	return nlm.persistNodeState(node)
}

// RevokeNode prevents any further operations on a node
func (nlm *NodeLifecycleManager) RevokeNode(ctx context.Context, nodeID string) error {
	nlm.mu.Lock()
	defer nlm.mu.Unlock()

	node, exists := nlm.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.State = NodeRevoked
	node.DesiredState = NodeRevoked
	node.ObservedAt = time.Now().UnixNano()
	node.Generation++
	node.Reason = "revoked by authority"

	return nlm.persistNodeState(node)
}

// GetNodeState returns the current observed state of a node
func (nlm *NodeLifecycleManager) GetNodeState(ctx context.Context, nodeID string) (*NodeStateValue, error) {
	nlm.mu.RLock()
	defer nlm.mu.RUnlock()

	node, exists := nlm.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeID)
	}

	// Make a copy to avoid external mutation
	copy := *node
	return &copy, nil
}

// DetectOfflineNodes marks nodes with expired heartbeats as OFFLINE
func (nlm *NodeLifecycleManager) DetectOfflineNodes(ctx context.Context) []string {
	nlm.mu.Lock()
	defer nlm.mu.Unlock()

	offline := []string{}
	now := time.Now()

	for nodeID, node := range nlm.nodes {
		if node.State == NodeRevoked || node.State == NodeOffline {
			continue
		}

		lastBeat := time.Unix(0, node.LastHeartbeat)
		if now.Sub(lastBeat) > nlm.staleThreshold {
			node.State = NodeOffline
			node.Freshness = "UNREACHABLE"
			node.Reason = fmt.Sprintf("heartbeat timeout (no update for %v)", nlm.staleThreshold)
			offline = append(offline, nodeID)
		} else if now.Sub(lastBeat) > nlm.heartbeatTimeout {
			node.Freshness = "STALE"
		}
	}

	return offline
}

// WorkloadLifecycleManager manages workload state transitions
type WorkloadLifecycleManager struct {
	workloads map[string]*WorkloadStateValue
	mu        sync.RWMutex
}

// NewWorkloadLifecycleManager creates a new workload lifecycle manager
func NewWorkloadLifecycleManager() *WorkloadLifecycleManager {
	return &WorkloadLifecycleManager{
		workloads: make(map[string]*WorkloadStateValue),
	}
}

// CreateWorkload registers a new workload with DESIRED state
func (wlm *WorkloadLifecycleManager) CreateWorkload(ctx context.Context, workloadID string) error {
	wlm.mu.Lock()
	defer wlm.mu.Unlock()

	if _, exists := wlm.workloads[workloadID]; exists {
		return fmt.Errorf("workload %s already exists", workloadID)
	}

	wlm.workloads[workloadID] = &WorkloadStateValue{
		WorkloadID:    workloadID,
		DesiredState:  WorkloadDesired,
		ObservedState: WorkloadUnknown,
		ObservedAt:    time.Now().UnixNano(),
		SourceID:      "control-plane",
		Freshness:     "FRESH",
	}

	return nil
}

// UpdateWorkloadObservedState updates the agent-observed workload state
func (wlm *WorkloadLifecycleManager) UpdateWorkloadObservedState(ctx context.Context, workloadID string, observed WorkloadLifecycleState, details map[string]string) error {
	wlm.mu.Lock()
	defer wlm.mu.Unlock()

	wl, exists := wlm.workloads[workloadID]
	if !exists {
		return fmt.Errorf("workload %s not found", workloadID)
	}

	wl.ObservedState = observed
	wl.ObservedAt = time.Now().UnixNano()
	wl.SourceID = "agent"
	wl.Freshness = "FRESH"

	if exitCode, ok := details["exitCode"]; ok {
		fmt.Sscanf(exitCode, "%d", &wl.ExitCode)
	}
	if err, ok := details["error"]; ok {
		wl.LastError = err
	}
	if health, ok := details["health"]; ok {
		wl.HealthStatus = health
	}

	return nil
}

// GetWorkloadState returns complete workload state
func (wlm *WorkloadLifecycleManager) GetWorkloadState(ctx context.Context, workloadID string) (*WorkloadStateValue, error) {
	wlm.mu.RLock()
	defer wlm.mu.RUnlock()

	wl, exists := wlm.workloads[workloadID]
	if !exists {
		return nil, fmt.Errorf("workload %s not found", workloadID)
	}

	copy := *wl
	return &copy, nil
}

// StopWorkload transitions workload to STOPPING
func (wlm *WorkloadLifecycleManager) StopWorkload(ctx context.Context, workloadID string) error {
	wlm.mu.Lock()
	defer wlm.mu.Unlock()

	wl, exists := wlm.workloads[workloadID]
	if !exists {
		return fmt.Errorf("workload %s not found", workloadID)
	}

	if wl.ObservedState == WorkloadStopped || wl.ObservedState == WorkloadFailed {
		return nil // already stopped
	}

	wl.DesiredState = WorkloadStopped
	wl.ObservedState = WorkloadStopping
	wl.ObservedAt = time.Now().UnixNano()

	return nil
}

// MarkWorkloadStopped marks workload as fully stopped
func (wlm *WorkloadLifecycleManager) MarkWorkloadStopped(ctx context.Context, workloadID string) error {
	wlm.mu.Lock()
	defer wlm.mu.Unlock()

	wl, exists := wlm.workloads[workloadID]
	if !exists {
		return fmt.Errorf("workload %s not found", workloadID)
	}

	wl.ObservedState = WorkloadStopped
	wl.DesiredState = WorkloadStopped
	wl.ObservedAt = time.Now().UnixNano()

	return nil
}

// DeleteWorkload removes a workload from tracking
func (wlm *WorkloadLifecycleManager) DeleteWorkload(ctx context.Context, workloadID string) error {
	wlm.mu.Lock()
	defer wlm.mu.Unlock()

	if _, exists := wlm.workloads[workloadID]; !exists {
		return fmt.Errorf("workload %s not found", workloadID)
	}

	delete(wlm.workloads, workloadID)
	return nil
}

// GetAllWorkloads returns all tracked workloads on a node
func (wlm *WorkloadLifecycleManager) GetAllWorkloads(ctx context.Context) []*WorkloadStateValue {
	wlm.mu.RLock()
	defer wlm.mu.RUnlock()

	result := make([]*WorkloadStateValue, 0, len(wlm.workloads))
	for _, wl := range wlm.workloads {
		copy := *wl
		result = append(result, &copy)
	}

	return result
}

// CountWorkloadsByState returns count of workloads in each state
func (wlm *WorkloadLifecycleManager) CountWorkloadsByState(ctx context.Context) map[WorkloadLifecycleState]int {
	wlm.mu.RLock()
	defer wlm.mu.RUnlock()

	counts := make(map[WorkloadLifecycleState]int)
	for _, wl := range wlm.workloads {
		counts[wl.ObservedState]++
	}

	return counts
}
