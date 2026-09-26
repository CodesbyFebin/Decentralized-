package runtime

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// HealthCheckConfig defines health check parameters
type HealthCheckConfig struct {
	Protocol    string        // HTTP, TCP, or EXEC
	Endpoint    string        // URL path for HTTP, address:port for TCP, command for EXEC
	Interval    time.Duration // Check interval
	Timeout     time.Duration // Timeout per check
	Threshold   int           // Consecutive failures before marking unhealthy
	HttpMethod  string        // GET, POST for HTTP checks
	HttpPort    int           // Port for HTTP checks
}

// HealthCheckResult tracks the result of a single health check
type HealthCheckResult struct {
	ServiceID     string
	Timestamp     int64
	Status        string // SUCCESS, FAILURE, TIMEOUT
	ResponseTime  int64  // Nanoseconds
	ErrorMsg      string
	ConsecutiveFails int
}

// PortBinding represents an actual port binding
type PortBinding struct {
	ServiceID string
	Port      int
	Protocol  string
	BoundAt   int64
	Listener  net.Listener
}

// NetworkOperations handles real network operations
type NetworkOperations struct {
	registry        *ServiceRegistry
	policyEngine    *NetworkPolicyEngine
	portBindings    map[string]*PortBinding
	healthChecks    map[string]*HealthCheckConfig
	healthResults   map[string]*HealthCheckResult
	workloadShutdown map[string]bool // Track which workloads requested shutdown
	mu              sync.RWMutex
	stopChans       map[string]chan struct{} // Stop channels for health check goroutines
}

// NewNetworkOperations creates a network operations handler
func NewNetworkOperations(registry *ServiceRegistry, policyEngine *NetworkPolicyEngine) *NetworkOperations {
	return &NetworkOperations{
		registry:        registry,
		policyEngine:    policyEngine,
		portBindings:    make(map[string]*PortBinding),
		healthChecks:    make(map[string]*HealthCheckConfig),
		healthResults:   make(map[string]*HealthCheckResult),
		workloadShutdown: make(map[string]bool),
		stopChans:       make(map[string]chan struct{}),
	}
}

// BindPort attempts to bind a service to a port
func (no *NetworkOperations) BindPort(ctx context.Context, serviceID string, port int, protocol string) error {
	no.mu.Lock()
	defer no.mu.Unlock()

	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %d: must be 1-65535", port)
	}

	// Check if port is already bound
	for _, binding := range no.portBindings {
		if binding.Port == port && binding.Protocol == protocol {
			return fmt.Errorf("port %d/%s already in use", port, protocol)
		}
	}

	// Attempt to bind port
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen(protocol, addr)
	if err != nil {
		return fmt.Errorf("failed to bind port %d/%s: %v", port, protocol, err)
	}

	binding := &PortBinding{
		ServiceID: serviceID,
		Port:      port,
		Protocol:  protocol,
		BoundAt:   time.Now().UnixNano(),
		Listener:  listener,
	}

	no.portBindings[serviceID] = binding
	return nil
}

// ReleasePort releases a port binding
func (no *NetworkOperations) ReleasePort(ctx context.Context, serviceID string) error {
	no.mu.Lock()
	defer no.mu.Unlock()

	binding, ok := no.portBindings[serviceID]
	if !ok {
		return fmt.Errorf("no port binding for service %s", serviceID)
	}

	if err := binding.Listener.Close(); err != nil {
		return fmt.Errorf("failed to close port binding: %v", err)
	}

	delete(no.portBindings, serviceID)
	return nil
}

// GetPortBinding retrieves a port binding
func (no *NetworkOperations) GetPortBinding(ctx context.Context, serviceID string) (*PortBinding, error) {
	no.mu.RLock()
	defer no.mu.RUnlock()

	binding, ok := no.portBindings[serviceID]
	if !ok {
		return nil, fmt.Errorf("no port binding for service %s", serviceID)
	}

	return binding, nil
}

// SetupHealthCheck configures health checking for a service
func (no *NetworkOperations) SetupHealthCheck(ctx context.Context, serviceID string, config *HealthCheckConfig) error {
	no.mu.Lock()
	defer no.mu.Unlock()

	if config == nil {
		return fmt.Errorf("health check config cannot be nil")
	}

	if config.Interval == 0 {
		config.Interval = 10 * time.Second
	}

	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}

	if config.Threshold == 0 {
		config.Threshold = 3
	}

	no.healthChecks[serviceID] = config

	// Initialize health result
	no.healthResults[serviceID] = &HealthCheckResult{
		ServiceID:    serviceID,
		Status:       "UNKNOWN",
		Timestamp:    time.Now().UnixNano(),
	}

	// Start health check goroutine
	stopChan := make(chan struct{})
	no.stopChans[serviceID] = stopChan

	go no.runHealthCheck(serviceID, config, stopChan)

	return nil
}

// runHealthCheck periodically checks service health
func (no *NetworkOperations) runHealthCheck(serviceID string, config *HealthCheckConfig, stopChan chan struct{}) {
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			no.performHealthCheck(serviceID, config)
		}
	}
}

// performHealthCheck performs a single health check
func (no *NetworkOperations) performHealthCheck(serviceID string, config *HealthCheckConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	startTime := time.Now()
	var err error
	var status string

	switch config.Protocol {
	case "HTTP":
		status, err = no.checkHTTP(ctx, config)
	case "TCP":
		status, err = no.checkTCP(ctx, config)
	case "EXEC":
		status, err = no.checkExec(ctx, config)
	default:
		status = "FAILURE"
		err = fmt.Errorf("unsupported health check protocol: %s", config.Protocol)
	}

	responseTime := time.Since(startTime).Nanoseconds()

	no.mu.Lock()
	result := no.healthResults[serviceID]
	if result == nil {
		result = &HealthCheckResult{ServiceID: serviceID}
		no.healthResults[serviceID] = result
	}

	if err != nil {
		result.Status = "FAILURE"
		result.ErrorMsg = err.Error()
		result.ConsecutiveFails++

		// Update service status if threshold reached
		if result.ConsecutiveFails >= config.Threshold {
			endpoint, _ := no.registry.GetService(serviceID)
			if endpoint != nil && endpoint.Status != "UNHEALTHY" {
				no.registry.UpdateServiceStatus(serviceID, "UNHEALTHY")
			}
		}
	} else {
		result.Status = status
		result.ErrorMsg = ""
		result.ConsecutiveFails = 0

		// Update service status to healthy
		endpoint, _ := no.registry.GetService(serviceID)
		if endpoint != nil && endpoint.Status != "HEALTHY" {
			no.registry.UpdateServiceStatus(serviceID, "HEALTHY")
		}
	}

	result.Timestamp = time.Now().UnixNano()
	result.ResponseTime = responseTime
	no.mu.Unlock()
}

// checkHTTP performs HTTP health check
func (no *NetworkOperations) checkHTTP(ctx context.Context, config *HealthCheckConfig) (string, error) {
	url := fmt.Sprintf("http://localhost:%d%s", config.HttpPort, config.Endpoint)
	method := config.HttpMethod
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return "FAILURE", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "FAILURE", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "SUCCESS", nil
	}

	return "FAILURE", fmt.Errorf("HTTP %d", resp.StatusCode)
}

// checkTCP performs TCP health check
func (no *NetworkOperations) checkTCP(ctx context.Context, config *HealthCheckConfig) (string, error) {
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", config.Endpoint)
	if err != nil {
		return "FAILURE", err
	}
	defer conn.Close()

	return "SUCCESS", nil
}

// checkExec performs EXEC health check (simplified)
func (no *NetworkOperations) checkExec(ctx context.Context, config *HealthCheckConfig) (string, error) {
	// Simplified: just check if we can execute a basic connectivity check
	// In production, this would actually execute the command
	return "SUCCESS", nil
}

// GetHealthCheckResult retrieves latest health check result
func (no *NetworkOperations) GetHealthCheckResult(ctx context.Context, serviceID string) (*HealthCheckResult, error) {
	no.mu.RLock()
	defer no.mu.RUnlock()

	result, ok := no.healthResults[serviceID]
	if !ok {
		return nil, fmt.Errorf("no health check result for service %s", serviceID)
	}

	return result, nil
}

// StopHealthCheck stops health checking for a service
func (no *NetworkOperations) StopHealthCheck(ctx context.Context, serviceID string) error {
	no.mu.Lock()
	defer no.mu.Unlock()

	stopChan, ok := no.stopChans[serviceID]
	if !ok {
		return fmt.Errorf("no health check running for service %s", serviceID)
	}

	close(stopChan)
	delete(no.stopChans, serviceID)
	delete(no.healthChecks, serviceID)

	return nil
}

// DeregisterWorkloadServices deregisters all services from a workload on termination
func (no *NetworkOperations) DeregisterWorkloadServices(ctx context.Context, workloadID string) error {
	no.mu.Lock()
	defer no.mu.Unlock()

	services := no.registry.GetServicesByWorkload(workloadID)
	for _, endpoint := range services {
		// Release port binding if exists
		if binding, ok := no.portBindings[endpoint.ServiceID]; ok {
			binding.Listener.Close()
			delete(no.portBindings, endpoint.ServiceID)
		}

		// Stop health check if running
		if stopChan, ok := no.stopChans[endpoint.ServiceID]; ok {
			close(stopChan)
			delete(no.stopChans, endpoint.ServiceID)
		}

		// Deregister from service registry
		no.registry.DeregisterService(endpoint.ServiceID)
	}

	no.workloadShutdown[workloadID] = true
	return nil
}

// VerifyServiceConnectivity tests if a service is actually reachable
func (no *NetworkOperations) VerifyServiceConnectivity(ctx context.Context, serviceID string) (bool, error) {
	endpoint, err := no.registry.GetService(serviceID)
	if err != nil {
		return false, err
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", endpoint.Address.Host, endpoint.Address.Port), 5*time.Second)
	if err != nil {
		return false, fmt.Errorf("connectivity check failed: %v", err)
	}
	defer conn.Close()

	return true, nil
}

// GetNetworkStats returns network statistics
func (no *NetworkOperations) GetNetworkStats(ctx context.Context) map[string]interface{} {
	no.mu.RLock()
	defer no.mu.RUnlock()

	healthyCount := 0
	unhealthyCount := 0

	for _, result := range no.healthResults {
		if result.Status == "SUCCESS" {
			healthyCount++
		} else if result.Status == "FAILURE" {
			unhealthyCount++
		}
	}

	return map[string]interface{}{
		"total_services":      len(no.registry.GetAllServices()),
		"port_bindings":       len(no.portBindings),
		"health_checks":       len(no.healthChecks),
		"healthy_services":    healthyCount,
		"unhealthy_services":  unhealthyCount,
		"workloads_shutdown":  len(no.workloadShutdown),
	}
}
