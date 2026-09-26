package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TruthState represents the truth state of an observation
type TruthState string

const (
	TruthLive      TruthState = "LIVE"
	TruthStale     TruthState = "STALE"
	TruthUnknown   TruthState = "UNKNOWN"
	TruthUnavailable TruthState = "UNAVAILABLE"
	TruthPlanned   TruthState = "PLANNED"
)

// Freshness represents how current an observation is
type Freshness string

const (
	FreshnessFresh      Freshness = "FRESH"
	FreshnessStale      Freshness = "STALE"
	FreshnessExpired    Freshness = "EXPIRED"
	FreshnessUnreachable Freshness = "UNREACHABLE"
	FreshnessUnknown    Freshness = "UNKNOWN"
)

// TruthEnvelope wraps an observed value with metadata about when and where it was observed
type TruthEnvelope struct {
	Source     string      // node ID or "control-plane" identity
	ObservedAt int64       // Unix nanoseconds
	Freshness  Freshness   // How recent is this data?
	Value      interface{} // The observed state (can be string, struct, etc.)
	TruthState TruthState  // State of the observation
}

// NodeStateView represents a node's current state as observed and tracked by the control plane
type NodeStateView struct {
	NodeID            string
	State             string // DISCOVERED, ENROLLING, VERIFIED, ACTIVE, CORDONED, DRAINING, IDLE, OFFLINE, REVOKED, DEGRADED
	DesiredState      string
	LastObserved      int64       // Unix nanoseconds when last observation was made
	Freshness         Freshness   // FRESH, STALE, EXPIRED, UNREACHABLE, UNKNOWN
	ObservedBy        string      // Which control plane or agent made this observation
	Generation        int64       // Optimistic versioning for conflict detection
	Convergence       string      // CONVERGED, DEGRADED, CONVERGING, DIVERGED, STOPPED, UNKNOWN
	TruthEnvelope     *TruthEnvelope // Raw observation metadata
}

// CommandCentreBackend provides state queries and operations for the control plane UI
type CommandCentreBackend struct {
	nlm              *NodeLifecycleManager      // Reference to node lifecycle manager
	sr               *ServiceRegistry           // Reference to service registry
	npe              *NetworkPolicyEngine       // Reference to network policy engine
	staleThreshold   time.Duration              // How old can data be before it's considered stale?
	expiryThreshold  time.Duration              // How old before it's considered expired?
	mu               sync.RWMutex
	observations     map[string]*TruthEnvelope  // nodeID -> most recent observation
}

// NewCommandCentreBackend creates a new Command Centre backend
func NewCommandCentreBackend(nlm *NodeLifecycleManager, sr *ServiceRegistry, npe *NetworkPolicyEngine) *CommandCentreBackend {
	return &CommandCentreBackend{
		nlm:             nlm,
		sr:              sr,
		npe:             npe,
		staleThreshold:  30 * time.Second,  // Data > 30s old is stale
		expiryThreshold: 5 * time.Minute,   // Data > 5min old is expired
		observations:    make(map[string]*TruthEnvelope),
	}
}

// RecordObservation records an observation of a node's state
func (cc *CommandCentreBackend) RecordObservation(ctx context.Context, nodeID string, state string, source string) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	envelope := &TruthEnvelope{
		Source:     source,
		ObservedAt: time.Now().UnixNano(),
		Freshness:  FreshnessFresh,
		Value:      state,
		TruthState: TruthLive,
	}

	cc.observations[nodeID] = envelope
	return nil
}

// calculateFreshness determines if an observation is fresh, stale, expired, etc.
func (cc *CommandCentreBackend) calculateFreshness(observedAt int64) Freshness {
	age := time.Now().UnixNano() - observedAt
	ageSeconds := time.Duration(age)

	if ageSeconds > cc.expiryThreshold {
		return FreshnessExpired
	}
	if ageSeconds > cc.staleThreshold {
		return FreshnessStale
	}
	return FreshnessFresh
}

// ListNodes returns all nodes with their current state as TruthEnvelopes
func (cc *CommandCentreBackend) ListNodes(ctx context.Context) ([]*NodeStateView, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	var nodes []*NodeStateView

	cc.nlm.mu.RLock()
	for nodeID, nodeState := range cc.nlm.nodes {
		freshness := cc.calculateFreshness(nodeState.LastHeartbeat)

		envelope := &TruthEnvelope{
			Source:     nodeID,
			ObservedAt: nodeState.LastHeartbeat,
			Freshness:  freshness,
			Value:      string(nodeState.State),
			TruthState: TruthLive,
		}

		// Determine convergence based on desired vs observed state
		convergence := "CONVERGED"
		if nodeState.DesiredState != nodeState.State {
			convergence = "DIVERGED"
		}

		view := &NodeStateView{
			NodeID:        nodeID,
			State:         string(nodeState.State),
			DesiredState:  string(nodeState.DesiredState),
			LastObserved:  nodeState.LastHeartbeat,
			Freshness:     freshness,
			ObservedBy:    "control-plane",
			Generation:    nodeState.Generation,
			Convergence:   convergence,
			TruthEnvelope: envelope,
		}
		nodes = append(nodes, view)
	}
	cc.nlm.mu.RUnlock()

	return nodes, nil
}

// GetNodeState returns a specific node's state with freshness information
func (cc *CommandCentreBackend) GetNodeState(ctx context.Context, nodeID string) (*NodeStateView, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	cc.nlm.mu.RLock()
	nodeState, exists := cc.nlm.nodes[nodeID]
	cc.nlm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	freshness := cc.calculateFreshness(nodeState.LastHeartbeat)

	envelope := &TruthEnvelope{
		Source:     nodeID,
		ObservedAt: nodeState.LastHeartbeat,
		Freshness:  freshness,
		Value:      string(nodeState.State),
		TruthState: TruthLive,
	}

	// Determine convergence
	convergence := "CONVERGED"
	if nodeState.DesiredState != nodeState.State {
		convergence = "DIVERGED"
	}

	return &NodeStateView{
		NodeID:        nodeID,
		State:         string(nodeState.State),
		DesiredState:  string(nodeState.DesiredState),
		LastObserved:  nodeState.LastHeartbeat,
		Freshness:     freshness,
		ObservedBy:    "control-plane",
		Generation:    nodeState.Generation,
		Convergence:   convergence,
		TruthEnvelope: envelope,
	}, nil
}

// UpdateDesiredState updates what state a node should be in
func (cc *CommandCentreBackend) UpdateDesiredState(ctx context.Context, nodeID string, desiredState string) error {
	cc.nlm.mu.Lock()
	if node, exists := cc.nlm.nodes[nodeID]; exists {
		node.DesiredState = NodeLifecycleState(desiredState)
	}
	cc.nlm.mu.Unlock()

	return nil
}

// DrainNode initiates graceful node draining
func (cc *CommandCentreBackend) DrainNode(ctx context.Context, nodeID string) error {
	return cc.nlm.DrainNode(ctx, nodeID, 0)
}

// RevokeNode revokes a node (permanent state)
func (cc *CommandCentreBackend) RevokeNode(ctx context.Context, nodeID string) error {
	return cc.nlm.RevokeNode(ctx, nodeID)
}

// GetNodeServices returns all services running on a node
func (cc *CommandCentreBackend) GetNodeServices(ctx context.Context, nodeID string) ([]*ServiceEndpoint, error) {
	cc.sr.mu.RLock()
	defer cc.sr.mu.RUnlock()

	endpoints := cc.sr.byNode[nodeID]
	return endpoints, nil
}

// GetNodePolicies returns all network policies affecting a node's services
func (cc *CommandCentreBackend) GetNodePolicies(ctx context.Context, nodeID string) ([]*NetworkPolicy, error) {
	cc.npe.mu.RLock()
	defer cc.npe.mu.RUnlock()

	var policies []*NetworkPolicy
	for _, policy := range cc.npe.policies {
		policies = append(policies, policy)
	}
	return policies, nil
}

// EvidenceRecord documents an operation
type EvidenceRecord struct {
	Timestamp   int64  // When the operation occurred
	NodeID      string // Affected node
	Operation   string // RegisterNode, TransitionToActive, DrainNode, etc.
	RequestData string // JSON-serialized request
	Response    string // JSON-serialized response
	Signer      string // Who authorized this (cert DN or control-plane ID)
	Status      string // SUCCESS, FAILED
	ErrorMsg    string // If failed, why?
}

// AuditLog stores operation evidence
type AuditLog struct {
	records []*EvidenceRecord
	mu      sync.RWMutex
}

// LogOperation records an operation in the audit log
func (al *AuditLog) LogOperation(record *EvidenceRecord) {
	al.mu.Lock()
	defer al.mu.Unlock()
	record.Timestamp = time.Now().UnixNano()
	al.records = append(al.records, record)
}

// GetOperationHistory returns all recorded operations for a node
func (al *AuditLog) GetOperationHistory(nodeID string) []*EvidenceRecord {
	al.mu.RLock()
	defer al.mu.RUnlock()

	var history []*EvidenceRecord
	for _, record := range al.records {
		if record.NodeID == nodeID {
			history = append(history, record)
		}
	}
	return history
}

// NewAuditLog creates a new audit log
func NewAuditLog() *AuditLog {
	return &AuditLog{
		records: make([]*EvidenceRecord, 0),
	}
}
