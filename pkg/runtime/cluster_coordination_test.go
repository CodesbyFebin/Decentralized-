package runtime

import (
	"testing"
	"time"
)

// TestNodeRegistration verifies node registration in cluster
func TestNodeRegistration(t *testing.T) {
	t.Log("Testing node registration")

	cc := NewClusterCoordinator("node-1", 1*time.Second, 3*time.Second)

	// Register nodes
	if err := cc.RegisterNode("node-2", "192.168.1.2", 5000); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	if err := cc.RegisterNode("node-3", "192.168.1.3", 5000); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// Verify registration
	node2, err := cc.GetNodeStatus("node-2")
	if err != nil {
		t.Fatalf("GetNodeStatus failed: %v", err)
	}

	if node2.NodeID != "node-2" {
		t.Errorf("Expected node-2, got %s", node2.NodeID)
	}

	if node2.Status != "HEALTHY" {
		t.Errorf("Expected HEALTHY status, got %s", node2.Status)
	}

	t.Logf("PASS: %d nodes registered successfully", 2)
}

// TestLeaderElection verifies leader election process
func TestLeaderElection(t *testing.T) {
	t.Log("Testing leader election")

	cc := NewClusterCoordinator("node-1", 1*time.Second, 3*time.Second)
	cc.RegisterNode("node-2", "192.168.1.2", 5000)
	cc.RegisterNode("node-3", "192.168.1.3", 5000)

	// Start election
	if err := cc.StartLeaderElection(); err != nil {
		t.Fatalf("StartLeaderElection failed: %v", err)
	}

	// Verify leader state
	if !cc.IsLeader() {
		t.Error("Expected to be leader after election")
	}

	leaderID := cc.GetLeaderID()
	if leaderID != "node-1" {
		t.Errorf("Expected leader node-1, got %s", leaderID)
	}

	t.Logf("PASS: Leader elected successfully")
}

// TestVoteAcceptance verifies vote processing
func TestVoteAcceptance(t *testing.T) {
	t.Log("Testing vote acceptance")

	cc := NewClusterCoordinator("node-1", 1*time.Second, 3*time.Second)

	// Accept vote from candidate
	accepted := cc.AcceptLeadershipVote("node-2", 1)
	if !accepted {
		t.Error("Expected to accept vote")
	}

	// Verify voted for
	term := cc.GetTerm()
	if term != 1 {
		t.Errorf("Expected term 1, got %d", term)
	}

	// Try to vote for different candidate in same term
	accepted = cc.AcceptLeadershipVote("node-3", 1)
	if accepted {
		t.Error("Should not accept vote for different candidate in same term")
	}

	t.Logf("PASS: Vote acceptance working correctly")
}

// TestFollowerAcknowledgement verifies follower acknowledgement
func TestFollowerAcknowledgement(t *testing.T) {
	t.Log("Testing follower acknowledgement")

	cc := NewClusterCoordinator("node-1", 1*time.Second, 3*time.Second)
	cc.RegisterNode("node-2", "192.168.1.2", 5000)
	cc.RegisterNode("node-3", "192.168.1.3", 5000)

	// Become leader
	cc.StartLeaderElection()

	// Record follower acknowledgement
	if err := cc.FollowerAcknowledgesLeader("node-2", 5); err != nil {
		t.Fatalf("FollowerAcknowledgesLeader failed: %v", err)
	}

	if err := cc.FollowerAcknowledgesLeader("node-3", 5); err != nil {
		t.Fatalf("FollowerAcknowledgesLeader failed: %v", err)
	}

	t.Logf("PASS: Follower acknowledgements recorded")
}

// TestQuorumCalculation verifies quorum size
func TestQuorumCalculation(t *testing.T) {
	t.Log("Testing quorum calculation")

	cc := NewClusterCoordinator("node-1", 1*time.Second, 3*time.Second)
	cc.RegisterNode("node-2", "192.168.1.2", 5000)
	cc.RegisterNode("node-3", "192.168.1.3", 5000)

	quorum := cc.GetClusterQuorum()
	if quorum != 2 {
		t.Errorf("Expected quorum 2 for 3 nodes, got %d", quorum)
	}

	cc.RegisterNode("node-4", "192.168.1.4", 5000)
	quorum = cc.GetClusterQuorum()
	if quorum != 3 {
		t.Errorf("Expected quorum 3 for 4 nodes, got %d", quorum)
	}

	t.Logf("PASS: Quorum calculation correct")
}

// TestCommitIndexAdvance verifies commit index progression
func TestCommitIndexAdvance(t *testing.T) {
	t.Log("Testing commit index advancement")

	cc := NewClusterCoordinator("node-1", 1*time.Second, 3*time.Second)
	cc.RegisterNode("node-2", "192.168.1.2", 5000)
	cc.RegisterNode("node-3", "192.168.1.3", 5000)

	// Become leader
	cc.StartLeaderElection()

	// Get initial commit index
	status := cc.GetClusterStatus()
	initialCommit := status["commit_index"].(int64)

	// Record follower acknowledgements
	cc.FollowerAcknowledgesLeader("node-2", 1)
	cc.FollowerAcknowledgesLeader("node-3", 1)

	// Advance commit index
	if err := cc.AdvanceCommitIndex(); err != nil {
		t.Fatalf("AdvanceCommitIndex failed: %v", err)
	}

	status = cc.GetClusterStatus()
	newCommit := status["commit_index"].(int64)

	if newCommit <= initialCommit {
		t.Errorf("Commit index should advance: %d -> %d", initialCommit, newCommit)
	}

	t.Logf("PASS: Commit index advanced to %d", newCommit)
}

// TestHeartbeatHandling verifies heartbeat processing
func TestHeartbeatHandling(t *testing.T) {
	t.Log("Testing heartbeat handling")

	cc := NewClusterCoordinator("node-1", 1*time.Second, 3*time.Second)

	// Accept heartbeat from leader
	if err := cc.HandleHeartbeat("node-2", 1, 0); err != nil {
		t.Fatalf("HandleHeartbeat failed: %v", err)
	}

	term := cc.GetTerm()
	if term != 1 {
		t.Errorf("Expected term 1 after heartbeat, got %d", term)
	}

	t.Logf("PASS: Heartbeat processed successfully")
}

// TestNodeStateSynchronization verifies state sync
func TestNodeStateSynchronization(t *testing.T) {
	t.Log("Testing node state synchronization")

	cc := NewClusterCoordinator("node-1", 1*time.Second, 3*time.Second)
	cc.RegisterNode("node-2", "192.168.1.2", 5000)

	// Synchronize node state
	if err := cc.SynchronizeNodeState("node-2", 10, "HEALTHY"); err != nil {
		t.Fatalf("SynchronizeNodeState failed: %v", err)
	}

	// Verify synchronization
	node, _ := cc.GetNodeStatus("node-2")
	if node.Generation != 10 {
		t.Errorf("Expected generation 10, got %d", node.Generation)
	}

	if node.Status != "HEALTHY" {
		t.Errorf("Expected HEALTHY status, got %s", node.Status)
	}

	t.Logf("PASS: Node state synchronized")
}

// TestClusterStatus verifies cluster status reporting
func TestClusterStatus(t *testing.T) {
	t.Log("Testing cluster status")

	cc := NewClusterCoordinator("node-1", 1*time.Second, 3*time.Second)
	cc.RegisterNode("node-2", "192.168.1.2", 5000)
	cc.RegisterNode("node-3", "192.168.1.3", 5000)

	cc.StartLeaderElection()

	status := cc.GetClusterStatus()

	if status["node_id"] != "node-1" {
		t.Errorf("Expected node-1, got %v", status["node_id"])
	}

	if status["total_nodes"] != 3 {
		t.Errorf("Expected 3 nodes, got %v", status["total_nodes"])
	}

	if !status["is_leader"].(bool) {
		t.Error("Expected to be leader")
	}

	t.Logf("PASS: Cluster status: %v", status)
}

// TestBroadcastStateUpdate verifies state broadcast
func TestBroadcastStateUpdate(t *testing.T) {
	t.Log("Testing broadcast state update")

	cc := NewClusterCoordinator("node-1", 1*time.Second, 3*time.Second)
	cc.RegisterNode("node-2", "192.168.1.2", 5000)
	cc.RegisterNode("node-3", "192.168.1.3", 5000)

	cc.StartLeaderElection()

	// Broadcast update
	data := map[string]interface{}{"key": "value"}
	messages, err := cc.BroadcastStateUpdate(data)
	if err != nil {
		t.Fatalf("BroadcastStateUpdate failed: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages for 2 followers, got %d", len(messages))
	}

	for _, msg := range messages {
		if msg.MessageType != "STATE_SYNC" {
			t.Errorf("Expected STATE_SYNC message type, got %s", msg.MessageType)
		}
	}

	t.Logf("PASS: Broadcast %d state messages", len(messages))
}

// TestNodeMetadata verifies metadata storage
func TestNodeMetadata(t *testing.T) {
	t.Log("Testing node metadata")

	cc := NewClusterCoordinator("node-1", 1*time.Second, 3*time.Second)
	cc.RegisterNode("node-2", "192.168.1.2", 5000)

	// Set metadata
	if err := cc.SetNodeMetadata("node-2", "region", "us-east"); err != nil {
		t.Fatalf("SetNodeMetadata failed: %v", err)
	}

	// Get metadata
	value, err := cc.GetNodeMetadata("node-2", "region")
	if err != nil {
		t.Fatalf("GetNodeMetadata failed: %v", err)
	}

	if value != "us-east" {
		t.Errorf("Expected us-east, got %s", value)
	}

	t.Logf("PASS: Metadata stored and retrieved")
}

// TestNodeUnregistration verifies node removal
func TestNodeUnregistration(t *testing.T) {
	t.Log("Testing node unregistration")

	cc := NewClusterCoordinator("node-1", 1*time.Second, 3*time.Second)
	cc.RegisterNode("node-2", "192.168.1.2", 5000)

	// Verify exists
	_, err := cc.GetNodeStatus("node-2")
	if err != nil {
		t.Fatalf("Node should exist before unregistration")
	}

	// Unregister
	if err := cc.UnregisterNode("node-2"); err != nil {
		t.Fatalf("UnregisterNode failed: %v", err)
	}

	// Verify removed
	_, err = cc.GetNodeStatus("node-2")
	if err == nil {
		t.Error("Node should not exist after unregistration")
	}

	t.Logf("PASS: Node unregistered successfully")
}

// TestStepDown verifies leader step down
func TestStepDown(t *testing.T) {
	t.Log("Testing leader step down")

	cc := NewClusterCoordinator("node-1", 1*time.Second, 3*time.Second)
	cc.RegisterNode("node-2", "192.168.1.2", 5000)

	// Become leader
	cc.StartLeaderElection()
	if !cc.IsLeader() {
		t.Fatalf("Expected to be leader")
	}

	// Step down
	if err := cc.StepDown(); err != nil {
		t.Fatalf("StepDown failed: %v", err)
	}

	// Verify no longer leader
	if cc.IsLeader() {
		t.Error("Should not be leader after step down")
	}

	leaderID := cc.GetLeaderID()
	if leaderID != "" {
		t.Errorf("Expected empty leader ID, got %s", leaderID)
	}

	t.Logf("PASS: Leader stepped down")
}

// TestGetAllNodes verifies node listing
func TestGetAllNodes(t *testing.T) {
	t.Log("Testing get all nodes")

	cc := NewClusterCoordinator("node-1", 1*time.Second, 3*time.Second)
	cc.RegisterNode("node-2", "192.168.1.2", 5000)
	cc.RegisterNode("node-3", "192.168.1.3", 5000)

	nodes := cc.GetAllNodes()
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes (including self), got %d", len(nodes))
	}

	nodeMap := make(map[string]bool)
	for _, node := range nodes {
		nodeMap[node.NodeID] = true
	}

	if !nodeMap["node-2"] || !nodeMap["node-3"] {
		t.Error("Expected node-2 and node-3 in list")
	}

	t.Logf("PASS: Retrieved %d nodes", len(nodes))
}
