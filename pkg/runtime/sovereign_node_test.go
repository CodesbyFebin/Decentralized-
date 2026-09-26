package runtime

import (
	"context"
	"testing"
	"time"
)

// TestP0SovereignFoundation verifies P0 foundation works on single node
func TestP0SovereignFoundation(t *testing.T) {
	t.Log("Testing P0 foundation on sovereign node")

	ctx := context.Background()

	// 1. Node Lifecycle Management
	t.Log("Phase 1: Node lifecycle management")
	nlm := NewNodeLifecycleManager()
	nodeID := "sovereign-node"

	if err := nlm.RegisterNode(ctx, nodeID); err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	// 2. Transition to verified (manually for test)
	t.Log("Phase 2: Transition to verified")
	nlm.mu.Lock()
	if node, ok := nlm.nodes[nodeID]; ok {
		node.State = NodeVerified
	}
	nlm.mu.Unlock()

	// 3. Transition to active
	t.Log("Phase 3: Transition to active")
	if err := nlm.TransitionToActive(ctx, nodeID); err != nil {
		t.Fatalf("TransitionToActive failed: %v", err)
	}

	state, _ := nlm.GetNodeState(ctx, nodeID)
	if state == nil || state.State != "ACTIVE" {
		t.Errorf("Expected ACTIVE state, got %v", state)
	}

	// 4. Service Registry
	t.Log("Phase 4: Service registry setup")
	nc := NewNetworkConfig()
	registry := nc.GetRegistry()

	address := &NetworkAddress{
		Host:     "127.0.0.1",
		Port:     8080,
		Protocol: "HTTP",
		TLS:      false,
	}

	workloadID := "workload-1"
	endpoint, err := registry.RegisterService(workloadID, workloadID, nodeID, address)
	if err != nil {
		t.Fatalf("RegisterService failed: %v", err)
	}

	if endpoint.Status != "REGISTERED" {
		t.Errorf("Expected REGISTERED, got %s", endpoint.Status)
	}

	// 5. Network Policies
	t.Log("Phase 5: Network policy enforcement")
	policyEngine := nc.GetPolicyEngine()

	_, err = policyEngine.CreatePolicy("allow-http", "Allow HTTP",
		workloadID, "*", 8080, "ALLOW")
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	allowed, _ := policyEngine.EvaluatePolicy(workloadID, workloadID, 8080)
	if !allowed {
		t.Error("Expected traffic to be allowed")
	}

	// 6. Scheduling Storage
	t.Log("Phase 6: Scheduling decision storage")
	ss := NewSchedulingStore()

	decision := &SchedulingDecision{
		WorkloadID:    workloadID,
		SelectedNodes: []string{nodeID},
		Strategy:      StrategyFirstFit,
	}

	constraints := &ResourceConstraints{
		MemoryBytes: 256 * 1024 * 1024,
		CPUShares:   200,
		DiskBytes:   2 * 1024 * 1024,
	}

	recordID, err := ss.StorePlacement(ctx, workloadID, decision, constraints)
	if err != nil {
		t.Fatalf("StorePlacement failed: %v", err)
	}

	if recordID == "" {
		t.Error("Expected valid record ID")
	}

	// 7. Status Updates
	t.Log("Phase 7: Status tracking")
	if err := ss.UpdatePlacementStatus(ctx, recordID, "RUNNING", "Started"); err != nil {
		t.Fatalf("UpdatePlacementStatus failed: %v", err)
	}

	record, _ := ss.GetPlacementRecord(ctx, recordID)
	if record.Status != "RUNNING" {
		t.Errorf("Expected RUNNING, got %s", record.Status)
	}

	// 8. Audit Trail
	t.Log("Phase 8: Audit trail verification")
	auditTrail, err := ss.GetAuditTrail(ctx, recordID)
	if err != nil {
		t.Fatalf("GetAuditTrail failed: %v", err)
	}

	if len(auditTrail) < 2 {
		t.Errorf("Expected at least 2 audit entries, got %d", len(auditTrail))
	}

	// 9. Cluster State
	t.Log("Phase 9: Cluster coordination")
	cc := NewClusterCoordinator(nodeID, 1*time.Second, 3*time.Second)

	// Single node coordinator tracks itself
	status := cc.GetClusterStatus()
	if status["total_nodes"] != 1 {
		t.Errorf("Expected 1 node, got %v", status["total_nodes"])
	}

	// Node can start election
	if err := cc.StartLeaderElection(); err != nil {
		t.Fatalf("StartLeaderElection failed: %v", err)
	}

	if !cc.IsLeader() {
		t.Error("Expected to be leader after election")
	}

	// 10. Graceful Shutdown
	t.Log("Phase 10: Graceful shutdown sequence")
	if err := nlm.CordonNode(ctx, nodeID); err != nil {
		t.Fatalf("CordonNode failed: %v", err)
	}

	state, _ = nlm.GetNodeState(ctx, nodeID)
	if state.State != "CORDONED" {
		t.Errorf("Expected CORDONED state, got %s", state.State)
	}

	// 11. Drain node
	t.Log("Phase 11: Draining workloads")
	if err := nlm.DrainNode(ctx, nodeID, 1); err != nil {
		t.Fatalf("DrainNode failed: %v", err)
	}

	state, _ = nlm.GetNodeState(ctx, nodeID)
	if state.State != "DRAINING" {
		t.Errorf("Expected DRAINING state, got %s", state.State)
	}

	t.Logf("PASS: P0 foundation operational on sovereign node")
	t.Logf("  ✓ Node lifecycle: ACTIVE → CORDONED → DRAINING")
	t.Logf("  ✓ Service registration and policy enforcement")
	t.Logf("  ✓ Scheduling placement tracking and audit trail")
	t.Logf("  ✓ Cluster coordination (single-node leader)")
	t.Logf("  ✓ Graceful shutdown sequence")
}

// TestP0StatePersistence verifies state persistence across operations
func TestP0StatePersistence(t *testing.T) {
	t.Log("Testing state persistence on sovereign node")

	ctx := context.Background()
	ss := NewSchedulingStore()

	workloadID := "persistent-workload"
	nodeID := "node-1"

	// Store initial decision
	decision := &SchedulingDecision{
		WorkloadID:    workloadID,
		SelectedNodes: []string{nodeID},
		Strategy:      StrategyBestFit,
	}

	recordID, err := ss.StorePlacement(ctx, workloadID, decision, nil)
	if err != nil {
		t.Fatalf("StorePlacement failed: %v", err)
	}

	// Perform multiple state transitions
	statuses := []string{"SCHEDULED", "RUNNING", "UPDATING", "RUNNING", "TERMINATED"}
	for _, status := range statuses {
		if err := ss.UpdatePlacementStatus(ctx, recordID, status, "State change"); err != nil {
			t.Fatalf("UpdatePlacementStatus failed: %v", err)
		}
	}

	// Verify complete audit trail
	auditTrail, _ := ss.GetAuditTrail(ctx, recordID)
	if len(auditTrail) != len(statuses)+1 { // +1 for initial SCHEDULED
		t.Errorf("Expected %d audit entries, got %d", len(statuses)+1, len(auditTrail))
	}

	// Verify chronological order
	for i := 0; i < len(auditTrail)-1; i++ {
		if auditTrail[i].Timestamp >= auditTrail[i+1].Timestamp {
			t.Error("Audit entries not in chronological order")
		}
	}

	t.Logf("PASS: State persistence verified with %d transitions", len(statuses))
}

// TestP0MultipleServices verifies multiple services on single node
func TestP0MultipleServices(t *testing.T) {
	t.Log("Testing multiple services on sovereign node")

	nc := NewNetworkConfig()
	registry := nc.GetRegistry()
	policyEngine := nc.GetPolicyEngine()
	nodeID := "single-node"

	// Register multiple services
	services := []string{"api-service", "cache-service", "db-service"}
	ports := []int{8080, 6379, 5432}

	for i, svc := range services {
		address := &NetworkAddress{
			Host:     "127.0.0.1",
			Port:     ports[i],
			Protocol: "TCP",
			TLS:      false,
		}

		_, err := registry.RegisterService(svc, svc, nodeID, address)
		if err != nil {
			t.Fatalf("RegisterService %s failed: %v", svc, err)
		}

		// Create inter-service policy
		if i > 0 {
			_, err = policyEngine.CreatePolicy(
				"policy-"+svc,
				"Allow "+svc,
				services[i-1], svc, ports[i], "ALLOW")
			if err != nil {
				t.Fatalf("CreatePolicy failed: %v", err)
			}
		}
	}

	// Verify all services registered
	allServices := registry.GetAllServices()
	if len(allServices) != len(services) {
		t.Errorf("Expected %d services, got %d", len(services), len(allServices))
	}

	// Verify policies
	allPolicies := policyEngine.GetAllPolicies()
	if len(allPolicies) != len(services)-1 {
		t.Errorf("Expected %d policies, got %d", len(services)-1, len(allPolicies))
	}

	t.Logf("PASS: %d services and %d policies verified", len(services), len(allPolicies))
}

// TestP0LoadBalancerConfiguration verifies load balancing setup
func TestP0LoadBalancerConfiguration(t *testing.T) {
	t.Log("Testing load balancer configuration on sovereign node")

	nc := NewNetworkConfig()

	// Check default strategy
	config := nc.GetLoadBalancerConfig()
	if config.Strategy != "ROUND_ROBIN" {
		t.Errorf("Expected ROUND_ROBIN, got %s", config.Strategy)
	}

	// Change strategy
	strategies := []string{"LEAST_LOAD", "RANDOM", "ROUND_ROBIN"}
	for _, strategy := range strategies {
		if err := nc.SetLoadBalancingStrategy(strategy); err != nil {
			t.Fatalf("SetLoadBalancingStrategy failed: %v", err)
		}

		config = nc.GetLoadBalancerConfig()
		if config.Strategy != strategy {
			t.Errorf("Expected %s, got %s", strategy, config.Strategy)
		}
	}

	t.Logf("PASS: Load balancer configuration verified")
}
