package runtime

import (
	"testing"
)

// TestServiceRegistry verifies service registration and discovery
func TestServiceRegistry(t *testing.T) {
	t.Log("Testing service registry")

	registry := NewServiceRegistry()

	// Register a service
	address := &NetworkAddress{
		Host:     "10.0.1.5",
		Port:     8080,
		Protocol: "HTTP",
		TLS:      false,
	}

	endpoint, err := registry.RegisterService("svc-1", "workload-1", "node-1", address)
	if err != nil {
		t.Fatalf("RegisterService failed: %v", err)
	}

	if endpoint.ServiceID != "svc-1" {
		t.Errorf("Expected service ID svc-1, got %s", endpoint.ServiceID)
	}

	if endpoint.Status != "REGISTERED" {
		t.Errorf("Expected status REGISTERED, got %s", endpoint.Status)
	}

	// Retrieve service
	retrieved, err := registry.GetService("svc-1")
	if err != nil {
		t.Fatalf("GetService failed: %v", err)
	}

	if retrieved.WorkloadID != "workload-1" {
		t.Errorf("Expected workload-1, got %s", retrieved.WorkloadID)
	}

	t.Logf("PASS: Service registered and retrieved successfully")
}

// TestServicesByWorkload queries services by workload
func TestServicesByWorkload(t *testing.T) {
	t.Log("Testing services by workload query")

	registry := NewServiceRegistry()

	// Register multiple services for same workload
	for i := 1; i <= 3; i++ {
		address := &NetworkAddress{
			Host:     "10.0.1.5",
			Port:     8080 + i,
			Protocol: "HTTP",
			TLS:      false,
		}

		registry.RegisterService("svc-"+string(rune('0'+i)), "workload-1", "node-1", address)
	}

	// Query by workload
	services := registry.GetServicesByWorkload("workload-1")
	if len(services) != 3 {
		t.Errorf("Expected 3 services, got %d", len(services))
	}

	t.Logf("PASS: Retrieved %d services for workload", len(services))
}

// TestServicesByNode queries services by node
func TestServicesByNode(t *testing.T) {
	t.Log("Testing services by node query")

	registry := NewServiceRegistry()

	// Register services on different nodes
	for i := 1; i <= 2; i++ {
		for j := 1; j <= 2; j++ {
			address := &NetworkAddress{
				Host:     "10.0.1.5",
				Port:     8080 + (i-1)*100 + j,
				Protocol: "HTTP",
				TLS:      false,
			}

			nodeID := "node-" + string(rune('0'+i))
			wlID := "workload-" + string(rune('0'+(i-1)*2+j))
			registry.RegisterService(wlID, wlID, nodeID, address)
		}
	}

	// Query by node
	services := registry.GetServicesByNode("node-1")
	if len(services) != 2 {
		t.Errorf("Expected 2 services on node-1, got %d", len(services))
	}

	t.Logf("PASS: Retrieved services by node successfully")
}

// TestUpdateServiceStatus verifies status updates
func TestUpdateServiceStatus(t *testing.T) {
	t.Log("Testing service status updates")

	registry := NewServiceRegistry()

	address := &NetworkAddress{
		Host:     "10.0.1.5",
		Port:     8080,
		Protocol: "HTTP",
		TLS:      false,
	}

	registry.RegisterService("svc-1", "workload-1", "node-1", address)

	// Update status
	if err := registry.UpdateServiceStatus("svc-1", "HEALTHY"); err != nil {
		t.Fatalf("UpdateServiceStatus failed: %v", err)
	}

	// Verify update
	service, _ := registry.GetService("svc-1")
	if service.Status != "HEALTHY" {
		t.Errorf("Expected status HEALTHY, got %s", service.Status)
	}

	t.Logf("PASS: Service status updated successfully")
}

// TestDeregisterService removes services
func TestDeregisterService(t *testing.T) {
	t.Log("Testing service deregistration")

	registry := NewServiceRegistry()

	address := &NetworkAddress{
		Host:     "10.0.1.5",
		Port:     8080,
		Protocol: "HTTP",
		TLS:      false,
	}

	registry.RegisterService("svc-1", "workload-1", "node-1", address)

	// Verify exists
	_, err := registry.GetService("svc-1")
	if err != nil {
		t.Fatalf("Service should exist before deregistration")
	}

	// Deregister
	if err := registry.DeregisterService("svc-1"); err != nil {
		t.Fatalf("DeregisterService failed: %v", err)
	}

	// Verify removed
	_, err = registry.GetService("svc-1")
	if err == nil {
		t.Error("Service should not exist after deregistration")
	}

	t.Logf("PASS: Service deregistered successfully")
}

// TestNetworkPolicy verifies policy creation and management
func TestNetworkPolicy(t *testing.T) {
	t.Log("Testing network policies")

	policyEngine := NewNetworkPolicyEngine()

	// Create allow policy
	policy, err := policyEngine.CreatePolicy("policy-1", "Allow HTTP",
		"workload-1", "workload-2", 8080, "ALLOW")
	if err != nil {
		t.Fatalf("CreatePolicy failed: %v", err)
	}

	if policy.PolicyID != "policy-1" {
		t.Errorf("Expected policy ID policy-1, got %s", policy.PolicyID)
	}

	if policy.Action != "ALLOW" {
		t.Errorf("Expected action ALLOW, got %s", policy.Action)
	}

	// Retrieve policy
	retrieved, err := policyEngine.GetPolicy("policy-1")
	if err != nil {
		t.Fatalf("GetPolicy failed: %v", err)
	}

	if retrieved.Name != "Allow HTTP" {
		t.Errorf("Expected name 'Allow HTTP', got %s", retrieved.Name)
	}

	t.Logf("PASS: Network policy created and retrieved successfully")
}

// TestEvaluatePolicy checks policy enforcement
func TestEvaluatePolicy(t *testing.T) {
	t.Log("Testing policy evaluation")

	policyEngine := NewNetworkPolicyEngine()

	// Create allow policy
	policyEngine.CreatePolicy("policy-allow", "Allow HTTP",
		"workload-1", "workload-2", 8080, "ALLOW")

	// Create deny policy
	policyEngine.CreatePolicy("policy-deny", "Deny SSH",
		"*", "workload-2", 22, "DENY")

	// Test allowed traffic
	allowed, reason := policyEngine.EvaluatePolicy("workload-1", "workload-2", 8080)
	if !allowed {
		t.Errorf("Traffic should be allowed: %s", reason)
	}

	// Test denied traffic
	allowed, reason = policyEngine.EvaluatePolicy("workload-1", "workload-2", 22)
	if allowed {
		t.Errorf("Traffic should be denied: %s", reason)
	}

	// Test default (not explicitly denied)
	allowed, reason = policyEngine.EvaluatePolicy("workload-3", "workload-4", 9000)
	if !allowed {
		t.Errorf("Traffic should be allowed by default: %s", reason)
	}

	t.Logf("PASS: Policy evaluation working correctly")
}

// TestGetAllPolicies retrieves all policies
func TestGetAllPolicies(t *testing.T) {
	t.Log("Testing policy retrieval")

	policyEngine := NewNetworkPolicyEngine()

	// Create multiple policies
	for i := 1; i <= 3; i++ {
		policyEngine.CreatePolicy("policy-"+string(rune('0'+i)),
			"Policy "+string(rune('0'+i)),
			"workload-1", "workload-2", 8080, "ALLOW")
	}

	// Retrieve all
	policies := policyEngine.GetAllPolicies()
	if len(policies) != 3 {
		t.Errorf("Expected 3 policies, got %d", len(policies))
	}

	t.Logf("PASS: Retrieved %d policies", len(policies))
}

// TestNetworkConfig aggregates network components
func TestNetworkConfig(t *testing.T) {
	t.Log("Testing network configuration")

	nc := NewNetworkConfig()

	// Register service
	address := &NetworkAddress{
		Host:     "10.0.1.5",
		Port:     8080,
		Protocol: "HTTP",
		TLS:      false,
	}

	registry := nc.GetRegistry()
	registry.RegisterService("svc-1", "workload-1", "node-1", address)

	// Create policy
	policyEngine := nc.GetPolicyEngine()
	policyEngine.CreatePolicy("policy-1", "Allow",
		"workload-1", "workload-2", 8080, "ALLOW")

	// Verify integration
	service, _ := registry.GetService("svc-1")
	if service == nil {
		t.Error("Service should be registered")
	}

	policy, _ := policyEngine.GetPolicy("policy-1")
	if policy == nil {
		t.Error("Policy should be created")
	}

	t.Logf("PASS: Network configuration working")
}

// TestLoadBalancingStrategy verifies strategy configuration
func TestLoadBalancingStrategy(t *testing.T) {
	t.Log("Testing load balancing strategy")

	nc := NewNetworkConfig()

	// Check default strategy
	config := nc.GetLoadBalancerConfig()
	if config.Strategy != "ROUND_ROBIN" {
		t.Errorf("Expected default strategy ROUND_ROBIN, got %s", config.Strategy)
	}

	// Change strategy
	if err := nc.SetLoadBalancingStrategy("LEAST_LOAD"); err != nil {
		t.Fatalf("SetLoadBalancingStrategy failed: %v", err)
	}

	config = nc.GetLoadBalancerConfig()
	if config.Strategy != "LEAST_LOAD" {
		t.Errorf("Expected updated strategy LEAST_LOAD, got %s", config.Strategy)
	}

	t.Logf("PASS: Load balancing strategy configuration working")
}

// TestDeletePolicy removes policies
func TestDeletePolicy(t *testing.T) {
	t.Log("Testing policy deletion")

	policyEngine := NewNetworkPolicyEngine()

	policyEngine.CreatePolicy("policy-1", "Test Policy",
		"workload-1", "workload-2", 8080, "ALLOW")

	// Verify exists
	_, err := policyEngine.GetPolicy("policy-1")
	if err != nil {
		t.Fatalf("Policy should exist before deletion")
	}

	// Delete
	if err := policyEngine.DeletePolicy("policy-1"); err != nil {
		t.Fatalf("DeletePolicy failed: %v", err)
	}

	// Verify removed
	_, err = policyEngine.GetPolicy("policy-1")
	if err == nil {
		t.Error("Policy should not exist after deletion")
	}

	t.Logf("PASS: Policy deleted successfully")
}

// TestGetPoliciesForService retrieves policies for service
func TestGetPoliciesForService(t *testing.T) {
	t.Log("Testing policies for service")

	policyEngine := NewNetworkPolicyEngine()

	// Create policies affecting workload-1
	policyEngine.CreatePolicy("policy-1", "Source policy",
		"workload-1", "workload-2", 8080, "ALLOW")
	policyEngine.CreatePolicy("policy-2", "Destination policy",
		"workload-3", "workload-1", 9000, "DENY")
	policyEngine.CreatePolicy("policy-3", "Other policy",
		"workload-3", "workload-4", 8000, "ALLOW")

	// Get policies for workload-1
	policies := policyEngine.GetPoliciesForService("workload-1")
	if len(policies) != 2 {
		t.Errorf("Expected 2 policies affecting workload-1, got %d", len(policies))
	}

	t.Logf("PASS: Retrieved %d policies for service", len(policies))
}
