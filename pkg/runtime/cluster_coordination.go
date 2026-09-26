package runtime

import (
	"fmt"
	"sync"
	"time"
)

// NodeInfo represents metadata about a cluster node
type NodeInfo struct {
	NodeID       string
	Address      string
	Port         int
	Generation   int64 // Monotonic version for state comparison
	Status       string // HEALTHY, UNREACHABLE, SUSPECTED
	LastHeartbeat int64
	Metadata     map[string]string
}

// ClusterState represents distributed cluster-wide state
type ClusterState struct {
	StateID      string            // Unique state identifier
	Nodes        map[string]*NodeInfo
	Generation   int64             // Global generation counter
	Timestamp    int64
	CommittedSeq int64             // Last committed sequence
}

// ConsensusMessage represents inter-node communication
type ConsensusMessage struct {
	MessageID   string
	Source      string              // Originating node ID
	MessageType string              // HEARTBEAT, STATE_SYNC, VOTE, ACK
	Term        int64               // Logical clock for ordering
	Data        map[string]interface{}
	Timestamp   int64
}

// LeaderState tracks leadership information
type LeaderState struct {
	LeaderID    string
	Term        int64
	ElectedAt   int64
	Majority    int
	Followers   map[string]bool // followerID -> acknowledged
}

// ClusterCoordinator manages multi-node consensus and state distribution
type ClusterCoordinator struct {
	nodeID           string
	nodes            map[string]*NodeInfo // All known nodes in cluster
	clusterState     *ClusterState
	leaderState      *LeaderState
	term             int64
	votedFor         string // Voted for in current term
	commitIndex      int64
	lastApplied      int64
	nextIndex        map[string]int64 // For each node
	matchIndex       map[string]int64 // For each node
	heartbeatTimeout time.Duration
	electionTimeout  time.Duration
	lastHeartbeat    int64
	mu               sync.RWMutex
}

// NewClusterCoordinator creates a new cluster coordinator
func NewClusterCoordinator(nodeID string, heartbeatTimeout, electionTimeout time.Duration) *ClusterCoordinator {
	nodes := make(map[string]*NodeInfo)
	clusterStateNodes := make(map[string]*NodeInfo)

	// Register this node itself
	selfNode := &NodeInfo{
		NodeID:        nodeID,
		Address:       "127.0.0.1",
		Port:          0,
		Generation:    0,
		Status:        "HEALTHY",
		LastHeartbeat: time.Now().UnixNano(),
		Metadata:      make(map[string]string),
	}
	nodes[nodeID] = selfNode
	clusterStateNodes[nodeID] = selfNode

	return &ClusterCoordinator{
		nodeID:           nodeID,
		nodes:            nodes,
		leaderState:      nil,
		term:             0,
		votedFor:         "",
		commitIndex:      0,
		lastApplied:      0,
		nextIndex:        make(map[string]int64),
		matchIndex:       make(map[string]int64),
		heartbeatTimeout: heartbeatTimeout,
		electionTimeout:  electionTimeout,
		lastHeartbeat:    time.Now().UnixNano(),
		clusterState: &ClusterState{
			StateID:      "state-1",
			Nodes:        clusterStateNodes,
			Generation:   0,
			CommittedSeq: 0,
		},
	}
}

// RegisterNode adds a node to the cluster
func (cc *ClusterCoordinator) RegisterNode(nodeID string, address string, port int) error {
	if nodeID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	cc.mu.Lock()
	defer cc.mu.Unlock()

	nodeInfo := &NodeInfo{
		NodeID:        nodeID,
		Address:       address,
		Port:          port,
		Generation:    0,
		Status:        "HEALTHY",
		LastHeartbeat: time.Now().UnixNano(),
		Metadata:      make(map[string]string),
	}

	cc.nodes[nodeID] = nodeInfo
	cc.clusterState.Nodes[nodeID] = nodeInfo
	cc.nextIndex[nodeID] = cc.commitIndex + 1
	cc.matchIndex[nodeID] = 0

	return nil
}

// UnregisterNode removes a node from cluster
func (cc *ClusterCoordinator) UnregisterNode(nodeID string) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	_, exists := cc.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	delete(cc.nodes, nodeID)
	delete(cc.clusterState.Nodes, nodeID)
	delete(cc.nextIndex, nodeID)
	delete(cc.matchIndex, nodeID)

	return nil
}

// StartLeaderElection initiates Raft-style leader election
func (cc *ClusterCoordinator) StartLeaderElection() error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.term++
	cc.votedFor = cc.nodeID

	majority := (len(cc.nodes) / 2) + 1

	cc.leaderState = &LeaderState{
		LeaderID:  cc.nodeID,
		Term:      cc.term,
		ElectedAt: time.Now().UnixNano(),
		Majority:  majority,
		Followers: make(map[string]bool),
	}

	for nodeID := range cc.nodes {
		if nodeID != cc.nodeID {
			cc.leaderState.Followers[nodeID] = false
		}
	}

	return nil
}

// AcceptLeadershipVote processes a vote from another node
func (cc *ClusterCoordinator) AcceptLeadershipVote(candidateID string, term int64) bool {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if term < cc.term {
		return false
	}

	if term > cc.term {
		cc.term = term
		cc.votedFor = ""
		cc.leaderState = nil
	}

	if cc.votedFor == "" || cc.votedFor == candidateID {
		cc.votedFor = candidateID
		return true
	}

	return false
}

// FollowerAcknowledgesLeader records acknowledgement from follower
func (cc *ClusterCoordinator) FollowerAcknowledgesLeader(followerID string, matchedIndex int64) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.leaderState == nil {
		return fmt.Errorf("not a leader")
	}

	if _, exists := cc.leaderState.Followers[followerID]; !exists {
		return fmt.Errorf("unknown follower: %s", followerID)
	}

	cc.leaderState.Followers[followerID] = true
	cc.matchIndex[followerID] = matchedIndex

	return nil
}

// GetClusterQuorum returns the quorum size for this cluster
func (cc *ClusterCoordinator) GetClusterQuorum() int {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return (len(cc.nodes) / 2) + 1
}

// AdvanceCommitIndex moves commit index when majority acknowledged
func (cc *ClusterCoordinator) AdvanceCommitIndex() error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.leaderState == nil {
		return fmt.Errorf("not a leader")
	}

	ackCount := 1 // Leader acks itself
	for _, acked := range cc.leaderState.Followers {
		if acked {
			ackCount++
		}
	}

	if ackCount >= cc.leaderState.Majority {
		cc.commitIndex++
		cc.clusterState.CommittedSeq = cc.commitIndex
		cc.clusterState.Generation++
		return nil
	}

	return fmt.Errorf("insufficient acknowledgements: %d < %d", ackCount, cc.leaderState.Majority)
}

// HandleHeartbeat processes heartbeat from leader
func (cc *ClusterCoordinator) HandleHeartbeat(leaderID string, term int64, commitIndex int64) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if term < cc.term {
		return fmt.Errorf("stale term: %d < %d", term, cc.term)
	}

	if term > cc.term {
		cc.term = term
		cc.votedFor = ""
		cc.leaderState = nil
	}

	cc.lastHeartbeat = time.Now().UnixNano()

	// Update commit index from leader
	if commitIndex > cc.commitIndex {
		cc.commitIndex = commitIndex
	}

	return nil
}

// SynchronizeNodeState updates node state in cluster
func (cc *ClusterCoordinator) SynchronizeNodeState(nodeID string, generation int64, status string) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	node, exists := cc.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	node.Generation = generation
	node.Status = status
	node.LastHeartbeat = time.Now().UnixNano()

	return nil
}

// GetNodeStatus retrieves status of specific node
func (cc *ClusterCoordinator) GetNodeStatus(nodeID string) (*NodeInfo, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	node, exists := cc.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	return node, nil
}

// GetClusterStatus returns aggregated cluster status
func (cc *ClusterCoordinator) GetClusterStatus() map[string]interface{} {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	healthyCount := 0
	unreachableCount := 0
	suspectedCount := 0

	for _, node := range cc.nodes {
		switch node.Status {
		case "HEALTHY":
			healthyCount++
		case "UNREACHABLE":
			unreachableCount++
		case "SUSPECTED":
			suspectedCount++
		}
	}

	return map[string]interface{}{
		"node_id":         cc.nodeID,
		"term":            cc.term,
		"leader_id":       func() string { if cc.leaderState != nil { return cc.leaderState.LeaderID } ; return "" }(),
		"is_leader":       cc.leaderState != nil && cc.leaderState.LeaderID == cc.nodeID,
		"total_nodes":     len(cc.nodes),
		"healthy_nodes":   healthyCount,
		"unreachable":     unreachableCount,
		"suspected":       suspectedCount,
		"commit_index":    cc.commitIndex,
		"generation":      cc.clusterState.Generation,
		"committed_seq":   cc.clusterState.CommittedSeq,
	}
}

// GetAllNodes returns all known nodes
func (cc *ClusterCoordinator) GetAllNodes() []*NodeInfo {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	nodes := []*NodeInfo{}
	for _, node := range cc.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

// BroadcastStateUpdate sends state to all followers
func (cc *ClusterCoordinator) BroadcastStateUpdate(data map[string]interface{}) ([]*ConsensusMessage, error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.leaderState == nil {
		return nil, fmt.Errorf("not a leader")
	}

	messages := []*ConsensusMessage{}
	for _ = range cc.leaderState.Followers {
		msg := &ConsensusMessage{
			MessageID:   fmt.Sprintf("msg-%d", time.Now().UnixNano()),
			Source:      cc.nodeID,
			MessageType: "STATE_SYNC",
			Term:        cc.term,
			Data:        data,
			Timestamp:   time.Now().UnixNano(),
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// IsLeader checks if this node is the leader
func (cc *ClusterCoordinator) IsLeader() bool {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.leaderState != nil && cc.leaderState.LeaderID == cc.nodeID
}

// GetLeaderID returns current leader ID or empty string
func (cc *ClusterCoordinator) GetLeaderID() string {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	if cc.leaderState != nil {
		return cc.leaderState.LeaderID
	}
	return ""
}

// GetTerm returns current term
func (cc *ClusterCoordinator) GetTerm() int64 {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.term
}

// SetNodeMetadata stores metadata for a node
func (cc *ClusterCoordinator) SetNodeMetadata(nodeID string, key string, value string) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	node, exists := cc.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	node.Metadata[key] = value
	return nil
}

// GetNodeMetadata retrieves metadata for a node
func (cc *ClusterCoordinator) GetNodeMetadata(nodeID string, key string) (string, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	node, exists := cc.nodes[nodeID]
	if !exists {
		return "", fmt.Errorf("node not found: %s", nodeID)
	}

	value, ok := node.Metadata[key]
	if !ok {
		return "", fmt.Errorf("metadata key not found: %s", key)
	}

	return value, nil
}

// StepDown removes leadership
func (cc *ClusterCoordinator) StepDown() error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.leaderState == nil {
		return fmt.Errorf("not a leader")
	}

	cc.leaderState = nil
	return nil
}
