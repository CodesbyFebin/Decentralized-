package runtime

import (
	"fmt"
	"sync"
	"time"
)

// NetworkAddress represents workload network endpoint
type NetworkAddress struct {
	Host     string
	Port     int
	Protocol string // TCP, UDP, HTTP, gRPC, etc.
	TLS      bool
}

// ServiceEndpoint represents a registered service instance
type ServiceEndpoint struct {
	ServiceID     string
	WorkloadID    string
	NodeID        string
	Address       *NetworkAddress
	Status        string // REGISTERED, HEALTHY, UNHEALTHY, DEREGISTERED
	RegisteredAt  int64
	LastHeartbeat int64
}

// NetworkPolicy defines access control rules
type NetworkPolicy struct {
	PolicyID    string
	Name        string
	Source      string // Service or workload ID, or * for any
	Destination string // Service or workload ID, or * for any
	Port        int    // Destination port, 0 for all
	Action      string // ALLOW or DENY
	CreatedAt   int64
}

// ServiceRegistry maintains service discovery information
type ServiceRegistry struct {
	services   map[string]*ServiceEndpoint   // serviceID -> endpoint
	byWorkload map[string][]*ServiceEndpoint // workloadID -> endpoints
	byNode     map[string][]*ServiceEndpoint // nodeID -> endpoints
	mu         sync.RWMutex
}

// NetworkPolicyEngine enforces network access control
type NetworkPolicyEngine struct {
	policies map[string]*NetworkPolicy
	mu       sync.RWMutex
}

// NewServiceRegistry creates a service registry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services:   make(map[string]*ServiceEndpoint),
		byWorkload: make(map[string][]*ServiceEndpoint),
		byNode:     make(map[string][]*ServiceEndpoint),
	}
}

// NewNetworkPolicyEngine creates a policy engine
func NewNetworkPolicyEngine() *NetworkPolicyEngine {
	return &NetworkPolicyEngine{
		policies: make(map[string]*NetworkPolicy),
	}
}

// RegisterService registers a workload as a service
func (sr *ServiceRegistry) RegisterService(serviceID string, workloadID string,
	nodeID string, address *NetworkAddress) (*ServiceEndpoint, error) {

	if serviceID == "" {
		return nil, fmt.Errorf("service ID cannot be empty")
	}

	if address == nil {
		return nil, fmt.Errorf("network address cannot be nil")
	}

	endpoint := &ServiceEndpoint{
		ServiceID:     serviceID,
		WorkloadID:    workloadID,
		NodeID:        nodeID,
		Address:       address,
		Status:        "REGISTERED",
		RegisteredAt:  time.Now().UnixNano(),
		LastHeartbeat: time.Now().UnixNano(),
	}

	sr.mu.Lock()
	sr.services[serviceID] = endpoint
	sr.byWorkload[workloadID] = append(sr.byWorkload[workloadID], endpoint)
	sr.byNode[nodeID] = append(sr.byNode[nodeID], endpoint)
	sr.mu.Unlock()

	return endpoint, nil
}

// DeregisterService removes a service from registry
func (sr *ServiceRegistry) DeregisterService(serviceID string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	endpoint, ok := sr.services[serviceID]
	if !ok {
		return fmt.Errorf("service not found: %s", serviceID)
	}

	// Remove from services
	delete(sr.services, serviceID)

	// Remove from byWorkload
	workloadID := endpoint.WorkloadID
	endpoints := sr.byWorkload[workloadID]
	newEndpoints := []*ServiceEndpoint{}
	for _, e := range endpoints {
		if e.ServiceID != serviceID {
			newEndpoints = append(newEndpoints, e)
		}
	}
	if len(newEndpoints) == 0 {
		delete(sr.byWorkload, workloadID)
	} else {
		sr.byWorkload[workloadID] = newEndpoints
	}

	// Remove from byNode
	nodeID := endpoint.NodeID
	endpoints = sr.byNode[nodeID]
	newEndpoints = []*ServiceEndpoint{}
	for _, e := range endpoints {
		if e.ServiceID != serviceID {
			newEndpoints = append(newEndpoints, e)
		}
	}
	if len(newEndpoints) == 0 {
		delete(sr.byNode, nodeID)
	} else {
		sr.byNode[nodeID] = newEndpoints
	}

	return nil
}

// UpdateServiceStatus changes service health status
func (sr *ServiceRegistry) UpdateServiceStatus(serviceID string, status string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	endpoint, ok := sr.services[serviceID]
	if !ok {
		return fmt.Errorf("service not found: %s", serviceID)
	}

	endpoint.Status = status
	endpoint.LastHeartbeat = time.Now().UnixNano()
	return nil
}

// GetService retrieves a specific service
func (sr *ServiceRegistry) GetService(serviceID string) (*ServiceEndpoint, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	endpoint, ok := sr.services[serviceID]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	return endpoint, nil
}

// GetServicesByWorkload retrieves all services for a workload
func (sr *ServiceRegistry) GetServicesByWorkload(workloadID string) []*ServiceEndpoint {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	endpoints, ok := sr.byWorkload[workloadID]
	if !ok {
		return []*ServiceEndpoint{}
	}

	result := make([]*ServiceEndpoint, len(endpoints))
	copy(result, endpoints)
	return result
}

// GetServicesByNode retrieves all services on a node
func (sr *ServiceRegistry) GetServicesByNode(nodeID string) []*ServiceEndpoint {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	endpoints, ok := sr.byNode[nodeID]
	if !ok {
		return []*ServiceEndpoint{}
	}

	result := make([]*ServiceEndpoint, len(endpoints))
	copy(result, endpoints)
	return result
}

// GetAllServices retrieves all registered services
func (sr *ServiceRegistry) GetAllServices() []*ServiceEndpoint {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	endpoints := []*ServiceEndpoint{}
	for _, endpoint := range sr.services {
		endpoints = append(endpoints, endpoint)
	}

	return endpoints
}

// CreatePolicy creates a new network policy
func (npe *NetworkPolicyEngine) CreatePolicy(policyID string, name string,
	source string, destination string, port int, action string) (*NetworkPolicy, error) {

	if policyID == "" {
		return nil, fmt.Errorf("policy ID cannot be empty")
	}

	if action != "ALLOW" && action != "DENY" {
		return nil, fmt.Errorf("action must be ALLOW or DENY")
	}

	policy := &NetworkPolicy{
		PolicyID:    policyID,
		Name:        name,
		Source:      source,
		Destination: destination,
		Port:        port,
		Action:      action,
		CreatedAt:   time.Now().UnixNano(),
	}

	npe.mu.Lock()
	npe.policies[policyID] = policy
	npe.mu.Unlock()

	return policy, nil
}

// GetPolicy retrieves a specific policy
func (npe *NetworkPolicyEngine) GetPolicy(policyID string) (*NetworkPolicy, error) {
	npe.mu.RLock()
	defer npe.mu.RUnlock()

	policy, ok := npe.policies[policyID]
	if !ok {
		return nil, fmt.Errorf("policy not found: %s", policyID)
	}

	return policy, nil
}

// GetAllPolicies retrieves all policies
func (npe *NetworkPolicyEngine) GetAllPolicies() []*NetworkPolicy {
	npe.mu.RLock()
	defer npe.mu.RUnlock()

	policies := []*NetworkPolicy{}
	for _, policy := range npe.policies {
		policies = append(policies, policy)
	}

	return policies
}

// DeletePolicy removes a policy
func (npe *NetworkPolicyEngine) DeletePolicy(policyID string) error {
	npe.mu.Lock()
	defer npe.mu.Unlock()

	_, ok := npe.policies[policyID]
	if !ok {
		return fmt.Errorf("policy not found: %s", policyID)
	}

	delete(npe.policies, policyID)
	return nil
}

// EvaluatePolicy checks if traffic is allowed
func (npe *NetworkPolicyEngine) EvaluatePolicy(source string, destination string, port int) (bool, string) {
	npe.mu.RLock()
	defer npe.mu.RUnlock()

	// Check matching policies
	for _, policy := range npe.policies {
		// Check if source matches
		sourceMatch := policy.Source == "*" || policy.Source == source

		// Check if destination matches
		destMatch := policy.Destination == "*" || policy.Destination == destination

		// Check if port matches (0 means all ports)
		portMatch := policy.Port == 0 || policy.Port == port

		if sourceMatch && destMatch && portMatch {
			if policy.Action == "ALLOW" {
				return true, fmt.Sprintf("Allowed by policy %s", policy.Name)
			} else {
				return false, fmt.Sprintf("Denied by policy %s", policy.Name)
			}
		}
	}

	// Default policy: allow if no deny rule matches
	return true, "Default allow (no matching deny rule)"
}

// GetPoliciesForService retrieves all policies affecting a service
func (npe *NetworkPolicyEngine) GetPoliciesForService(serviceID string) []*NetworkPolicy {
	npe.mu.RLock()
	defer npe.mu.RUnlock()

	policies := []*NetworkPolicy{}
	for _, policy := range npe.policies {
		// Check if service is source or destination
		if policy.Source == serviceID || policy.Source == "*" ||
			policy.Destination == serviceID || policy.Destination == "*" {
			policies = append(policies, policy)
		}
	}

	return policies
}

// NetworkConfig aggregates network configuration
type NetworkConfig struct {
	registry     *ServiceRegistry
	policyEngine *NetworkPolicyEngine
	loadBalancer *LoadBalancerConfig
	mu           sync.RWMutex
}

// LoadBalancerConfig manages load balancing
type LoadBalancerConfig struct {
	Strategy  string // ROUND_ROBIN, LEAST_LOAD, RANDOM
	Algorithm string
}

// NewNetworkConfig creates network configuration
func NewNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		registry:     NewServiceRegistry(),
		policyEngine: NewNetworkPolicyEngine(),
		loadBalancer: &LoadBalancerConfig{
			Strategy:  "ROUND_ROBIN",
			Algorithm: "RANDOM_SELECTION_FALLBACK",
		},
	}
}

// GetRegistry returns the service registry
func (nc *NetworkConfig) GetRegistry() *ServiceRegistry {
	return nc.registry
}

// GetPolicyEngine returns the policy engine
func (nc *NetworkConfig) GetPolicyEngine() *NetworkPolicyEngine {
	return nc.policyEngine
}

// GetLoadBalancerConfig returns load balancer configuration
func (nc *NetworkConfig) GetLoadBalancerConfig() *LoadBalancerConfig {
	return nc.loadBalancer
}

// SetLoadBalancingStrategy changes the strategy
func (nc *NetworkConfig) SetLoadBalancingStrategy(strategy string) error {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	if strategy != "ROUND_ROBIN" && strategy != "LEAST_LOAD" && strategy != "RANDOM" {
		return fmt.Errorf("unknown strategy: %s", strategy)
	}

	nc.loadBalancer.Strategy = strategy
	return nil
}
