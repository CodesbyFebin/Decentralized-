package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// APIRequest represents an incoming API request
type APIRequest struct {
	Method      string            // HTTP method
	Path        string            // URL path
	Headers     map[string]string // Request headers
	Query       map[string]string // Query parameters
	Body        interface{}       // Request body
	ContentType string            // Content-Type header
	ClientID    string            // Authenticated client
	Timestamp   time.Time         // Request time
	TraceID     string            // Request trace identifier
}

// APIResponse represents an outgoing API response
type APIResponse struct {
	StatusCode int         // HTTP status code
	Headers    map[string]string // Response headers
	Body       interface{} // Response body
	Error      *APIError   // Error details (if any)
	Duration   time.Duration // Response time
	Timestamp  time.Time   // Response time
}

// APIError represents a structured API error
type APIError struct {
	Code      string // Error code (e.g., "CAMPAIGN_NOT_FOUND")
	Message   string // Human-readable message
	Details   string // Additional details
	Status    int    // HTTP status code
	Timestamp time.Time
}

// APIEndpoint defines a single API endpoint
type APIEndpoint struct {
	Method      string                    // HTTP method (GET, POST, PUT, DELETE)
	Path        string                    // URL path pattern
	Handler     func(*APIRequest) *APIResponse // Handler function
	Description string                    // Endpoint description
	RequireAuth bool                      // Authentication required
	RateLimit   int                       // Requests per minute
	Version     string                    // API version (e.g., "v1")
}

// CampaignCreateRequest represents a campaign creation request
type CampaignCreateRequest struct {
	ResourceID        string `json:"resource_id"`
	QualificationLevel string `json:"qualification_level"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

// CampaignResponse represents campaign data in API response
type CampaignResponse struct {
	ID                 string            `json:"id"`
	ResourceID         string            `json:"resource_id"`
	QualificationLevel string            `json:"qualification_level"`
	Status             string            `json:"status"`
	StartTime          time.Time         `json:"start_time"`
	EndTime            *time.Time        `json:"end_time,omitempty"`
	ExecutionTime      int               `json:"execution_time_seconds"`
	GatePassed         int               `json:"gates_passed"`
	GateFailed         int               `json:"gates_failed"`
	PassRate           float64           `json:"pass_rate"`
	Metadata           map[string]string `json:"metadata,omitempty"`
}

// MetricsResponse represents metrics snapshot in API response
type MetricsResponse struct {
	Timestamp            time.Time `json:"timestamp"`
	TotalCampaigns       int       `json:"total_campaigns"`
	ActiveCampaigns      int       `json:"active_campaigns"`
	CompletedCampaigns   int       `json:"completed_campaigns"`
	SuccessRate          float64   `json:"success_rate"`
	AverageDuration      float64   `json:"average_duration_seconds"`
	DatabaseConnections  int       `json:"database_connections"`
	StorageUsage         int64     `json:"storage_usage_bytes"`
	MemoryUsage          int64     `json:"memory_usage_bytes"`
	CacheHitRate         float64   `json:"cache_hit_rate"`
}

// VersionResponse represents schema version info in API response
type VersionResponse struct {
	Current            string   `json:"current"`
	Previous           string   `json:"previous"`
	Available          []string `json:"available"`
	LastMigrationTime  time.Time `json:"last_migration_time,omitempty"`
	LastMigrationStatus string   `json:"last_migration_status,omitempty"`
}

// MigrationStartRequest represents a migration start request
type MigrationStartRequest struct {
	MigrationID string `json:"migration_id"`
	FromVersion string `json:"from_version"`
	ToVersion   string `json:"to_version"`
}

// MigrationStatusResponse represents migration status in API response
type MigrationStatusResponse struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	FromVersion     string `json:"from_version"`
	ToVersion       string `json:"to_version"`
	Progress        int    `json:"progress_percent"`
	StepsCompleted  int    `json:"steps_completed"`
	TotalSteps      int    `json:"total_steps"`
	StartTime       time.Time `json:"start_time"`
	EstimatedEndTime *time.Time `json:"estimated_end_time,omitempty"`
}

// ChaosTestRequest represents a chaos test execution request
type ChaosTestRequest struct {
	ScenarioName string        `json:"scenario_name"`
	Duration     time.Duration `json:"duration_seconds"`
	Severity     string        `json:"severity"`
}

// ChaosTestResponse represents chaos test results in API response
type ChaosTestResponse struct {
	ScenarioName       string `json:"scenario_name"`
	Status             string `json:"status"`
	RecoveryTime       int    `json:"recovery_time_seconds"`
	MaxAllowedDowntime int    `json:"max_allowed_downtime_seconds"`
	InvariantsBroken   []string `json:"invariants_broken,omitempty"`
	ErrorCount         int    `json:"error_count"`
	DataLoss           bool   `json:"data_loss"`
	ExecutionTime      int    `json:"execution_time_seconds"`
}

// RateLimiter tracks rate limiting per client
type RateLimiter struct {
	requests    map[string][]time.Time
	limiterLock sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
	}
}

// Allow checks if request is within rate limit
func (rl *RateLimiter) Allow(clientID string, limit int) bool {
	rl.limiterLock.Lock()
	defer rl.limiterLock.Unlock()

	now := time.Now()
	oneMinuteAgo := now.Add(-time.Minute)

	// Clean old requests
	if reqs, ok := rl.requests[clientID]; ok {
		filtered := make([]time.Time, 0)
		for _, t := range reqs {
			if t.After(oneMinuteAgo) {
				filtered = append(filtered, t)
			}
		}
		rl.requests[clientID] = filtered
	}

	// Check limit
	if len(rl.requests[clientID]) >= limit {
		return false
	}

	// Record request
	rl.requests[clientID] = append(rl.requests[clientID], now)
	return true
}

// APIGateway orchestrates API request routing and handling
type APIGateway struct {
	endpoints    map[string][]*APIEndpoint // Map of path → endpoints
	metricsCltr  *MetricsCollector         // Metrics collector dependency (optional)
	evolver      *SchemaEvolver            // Schema evolver dependency (optional)
	chaosRunner  *ChaosTestRunner          // Chaos runner dependency (optional)
	rateLimiter  *RateLimiter
	gatewayLock  sync.RWMutex
	requestCount int64                     // Total requests handled
	errorCount   int64                     // Total errors
	startTime    time.Time                 // Gateway start time
}

// NewAPIGateway creates a new API gateway with optional dependencies
func NewAPIGateway(metricsCltr *MetricsCollector, evolver *SchemaEvolver,
	chaosRunner *ChaosTestRunner) *APIGateway {
	gateway := &APIGateway{
		endpoints:   make(map[string][]*APIEndpoint),
		metricsCltr: metricsCltr,
		evolver:     evolver,
		chaosRunner: chaosRunner,
		rateLimiter: NewRateLimiter(),
		startTime:   time.Now(),
	}

	// Register default endpoints
	gateway.registerDefaultEndpoints()

	return gateway
}

// registerDefaultEndpoints registers standard API endpoints
func (ag *APIGateway) registerDefaultEndpoints() {
	// Campaign endpoints
	ag.RegisterEndpoint(&APIEndpoint{
		Method:      "GET",
		Path:        "/api/v1/campaigns",
		Description: "List all campaigns",
		RequireAuth: true,
		RateLimit:   100,
		Handler:     ag.handleListCampaigns,
	})

	ag.RegisterEndpoint(&APIEndpoint{
		Method:      "POST",
		Path:        "/api/v1/campaigns",
		Description: "Create new campaign",
		RequireAuth: true,
		RateLimit:   50,
		Handler:     ag.handleCreateCampaign,
	})

	ag.RegisterEndpoint(&APIEndpoint{
		Method:      "GET",
		Path:        "/api/v1/campaigns/{id}",
		Description: "Get campaign details",
		RequireAuth: true,
		RateLimit:   100,
		Handler:     ag.handleGetCampaign,
	})

	ag.RegisterEndpoint(&APIEndpoint{
		Method:      "DELETE",
		Path:        "/api/v1/campaigns/{id}",
		Description: "Delete campaign",
		RequireAuth: true,
		RateLimit:   20,
		Handler:     ag.handleDeleteCampaign,
	})

	// Metrics endpoints
	ag.RegisterEndpoint(&APIEndpoint{
		Method:      "GET",
		Path:        "/api/v1/metrics",
		Description: "Get current metrics snapshot",
		RequireAuth: false,
		RateLimit:   200,
		Handler:     ag.handleGetMetrics,
	})

	ag.RegisterEndpoint(&APIEndpoint{
		Method:      "GET",
		Path:        "/api/v1/metrics/history",
		Description: "Get metrics history",
		RequireAuth: true,
		RateLimit:   50,
		Handler:     ag.handleGetMetricsHistory,
	})

	// Schema version endpoints
	ag.RegisterEndpoint(&APIEndpoint{
		Method:      "GET",
		Path:        "/api/v1/schema/version",
		Description: "Get current schema version",
		RequireAuth: false,
		RateLimit:   200,
		Handler:     ag.handleGetSchemaVersion,
	})

	ag.RegisterEndpoint(&APIEndpoint{
		Method:      "POST",
		Path:        "/api/v1/schema/migrate",
		Description: "Start schema migration",
		RequireAuth: true,
		RateLimit:   5,
		Handler:     ag.handleStartMigration,
	})

	ag.RegisterEndpoint(&APIEndpoint{
		Method:      "GET",
		Path:        "/api/v1/schema/migrate/{id}",
		Description: "Get migration status",
		RequireAuth: true,
		RateLimit:   100,
		Handler:     ag.handleGetMigrationStatus,
	})

	// Chaos testing endpoints
	ag.RegisterEndpoint(&APIEndpoint{
		Method:      "POST",
		Path:        "/api/v1/chaos/test",
		Description: "Start chaos test",
		RequireAuth: true,
		RateLimit:   10,
		Handler:     ag.handleStartChaosTest,
	})

	ag.RegisterEndpoint(&APIEndpoint{
		Method:      "GET",
		Path:        "/api/v1/chaos/test/{id}",
		Description: "Get chaos test results",
		RequireAuth: true,
		RateLimit:   100,
		Handler:     ag.handleGetChaosTestResults,
	})

	// Health check endpoint
	ag.RegisterEndpoint(&APIEndpoint{
		Method:      "GET",
		Path:        "/api/v1/health",
		Description: "Health check",
		RequireAuth: false,
		RateLimit:   1000,
		Handler:     ag.handleHealthCheck,
	})
}

// RegisterEndpoint registers a new API endpoint
func (ag *APIGateway) RegisterEndpoint(endpoint *APIEndpoint) error {
	ag.gatewayLock.Lock()
	defer ag.gatewayLock.Unlock()

	if endpoint.Method == "" || endpoint.Path == "" || endpoint.Handler == nil {
		return fmt.Errorf("invalid endpoint configuration")
	}

	key := fmt.Sprintf("%s %s", endpoint.Method, endpoint.Path)
	ag.endpoints[key] = append(ag.endpoints[key], endpoint)

	return nil
}

// HandleRequest processes an incoming API request
func (ag *APIGateway) HandleRequest(req *APIRequest) *APIResponse {
	req.Timestamp = time.Now()
	startTime := req.Timestamp

	defer func() {
		ag.gatewayLock.Lock()
		ag.requestCount++
		ag.gatewayLock.Unlock()
	}()

	// Check rate limiting
	clientID := req.ClientID
	if clientID == "" {
		clientID = "anonymous"
	}

	// Find matching endpoint
	key := fmt.Sprintf("%s %s", req.Method, req.Path)
	ag.gatewayLock.RLock()
	endpoints, ok := ag.endpoints[key]
	ag.gatewayLock.RUnlock()

	if !ok || len(endpoints) == 0 {
		ag.gatewayLock.Lock()
		ag.errorCount++
		ag.gatewayLock.Unlock()

		return &APIResponse{
			StatusCode: http.StatusNotFound,
			Error: &APIError{
				Code:      "ENDPOINT_NOT_FOUND",
				Message:   fmt.Sprintf("No endpoint found for %s %s", req.Method, req.Path),
				Status:    http.StatusNotFound,
				Timestamp: time.Now(),
			},
			Timestamp: time.Now(),
		}
	}

	endpoint := endpoints[0]

	// Check authentication if required
	if endpoint.RequireAuth && req.ClientID == "" {
		ag.gatewayLock.Lock()
		ag.errorCount++
		ag.gatewayLock.Unlock()

		return &APIResponse{
			StatusCode: http.StatusUnauthorized,
			Error: &APIError{
				Code:      "UNAUTHORIZED",
				Message:   "Authentication required",
				Status:    http.StatusUnauthorized,
				Timestamp: time.Now(),
			},
			Timestamp: time.Now(),
		}
	}

	// Check rate limit
	if !ag.rateLimiter.Allow(clientID, endpoint.RateLimit) {
		ag.gatewayLock.Lock()
		ag.errorCount++
		ag.gatewayLock.Unlock()

		return &APIResponse{
			StatusCode: http.StatusTooManyRequests,
			Error: &APIError{
				Code:      "RATE_LIMIT_EXCEEDED",
				Message:   fmt.Sprintf("Rate limit of %d requests/minute exceeded", endpoint.RateLimit),
				Status:    http.StatusTooManyRequests,
				Timestamp: time.Now(),
			},
			Timestamp: time.Now(),
		}
	}

	// Execute handler
	resp := endpoint.Handler(req)
	resp.Duration = time.Since(startTime)
	resp.Timestamp = time.Now()

	// Track errors
	if resp.Error != nil {
		ag.gatewayLock.Lock()
		ag.errorCount++
		ag.gatewayLock.Unlock()
	}

	return resp
}

// Handler functions for each endpoint

func (ag *APIGateway) handleListCampaigns(req *APIRequest) *APIResponse {
	// Placeholder - would call campaignMgr to list campaigns
	return &APIResponse{
		StatusCode: http.StatusOK,
		Body: map[string]interface{}{
			"campaigns": []CampaignResponse{},
			"total":     0,
		},
	}
}

func (ag *APIGateway) handleCreateCampaign(req *APIRequest) *APIResponse {
	var createReq CampaignCreateRequest
	if body, ok := req.Body.(map[string]interface{}); ok {
		data, _ := json.Marshal(body)
		json.Unmarshal(data, &createReq)
	}

	if createReq.ResourceID == "" {
		return &APIResponse{
			StatusCode: http.StatusBadRequest,
			Error: &APIError{
				Code:      "INVALID_REQUEST",
				Message:   "resource_id is required",
				Status:    http.StatusBadRequest,
				Timestamp: time.Now(),
			},
		}
	}

	// Placeholder - would call campaignMgr to create campaign
	return &APIResponse{
		StatusCode: http.StatusCreated,
		Body: CampaignResponse{
			ID:         "campaign-" + fmt.Sprintf("%d", time.Now().UnixNano()),
			ResourceID: createReq.ResourceID,
			Status:     "pending",
			StartTime:  time.Now(),
		},
	}
}

func (ag *APIGateway) handleGetCampaign(req *APIRequest) *APIResponse {
	campaignID := extractPathParam(req.Path, "{id}")

	if campaignID == "" {
		return &APIResponse{
			StatusCode: http.StatusBadRequest,
			Error: &APIError{
				Code:      "INVALID_REQUEST",
				Message:   "Campaign ID is required",
				Status:    http.StatusBadRequest,
				Timestamp: time.Now(),
			},
		}
	}

	return &APIResponse{
		StatusCode: http.StatusOK,
		Body: CampaignResponse{
			ID:     campaignID,
			Status: "completed",
		},
	}
}

func (ag *APIGateway) handleDeleteCampaign(req *APIRequest) *APIResponse {
	return &APIResponse{
		StatusCode: http.StatusNoContent,
	}
}

func (ag *APIGateway) handleGetMetrics(req *APIRequest) *APIResponse {
	// Placeholder - would call metricsCltr to collect metrics
	return &APIResponse{
		StatusCode: http.StatusOK,
		Body: MetricsResponse{
			Timestamp:       time.Now(),
			TotalCampaigns:  0,
			ActiveCampaigns: 0,
			SuccessRate:     100.0,
		},
	}
}

func (ag *APIGateway) handleGetMetricsHistory(req *APIRequest) *APIResponse {
	return &APIResponse{
		StatusCode: http.StatusOK,
		Body: map[string]interface{}{
			"snapshots": []MetricsResponse{},
		},
	}
}

func (ag *APIGateway) handleGetSchemaVersion(req *APIRequest) *APIResponse {
	// Placeholder - would call evolver to get version
	return &APIResponse{
		StatusCode: http.StatusOK,
		Body: VersionResponse{
			Current:   "1.0",
			Available: []string{"1.0", "1.1"},
		},
	}
}

func (ag *APIGateway) handleStartMigration(req *APIRequest) *APIResponse {
	var migReq MigrationStartRequest
	if body, ok := req.Body.(map[string]interface{}); ok {
		data, _ := json.Marshal(body)
		json.Unmarshal(data, &migReq)
	}

	if migReq.MigrationID == "" {
		return &APIResponse{
			StatusCode: http.StatusBadRequest,
			Error: &APIError{
				Code:      "INVALID_REQUEST",
				Message:   "migration_id is required",
				Status:    http.StatusBadRequest,
				Timestamp: time.Now(),
			},
		}
	}

	return &APIResponse{
		StatusCode: http.StatusAccepted,
		Body: MigrationStatusResponse{
			ID:     migReq.MigrationID,
			Status: "in_progress",
		},
	}
}

func (ag *APIGateway) handleGetMigrationStatus(req *APIRequest) *APIResponse {
	migrationID := extractPathParam(req.Path, "{id}")

	return &APIResponse{
		StatusCode: http.StatusOK,
		Body: MigrationStatusResponse{
			ID:             migrationID,
			Status:         "completed",
			Progress:       100,
			StepsCompleted: 5,
			TotalSteps:     5,
		},
	}
}

func (ag *APIGateway) handleStartChaosTest(req *APIRequest) *APIResponse {
	var chaosReq ChaosTestRequest
	if body, ok := req.Body.(map[string]interface{}); ok {
		data, _ := json.Marshal(body)
		json.Unmarshal(data, &chaosReq)
	}

	if chaosReq.ScenarioName == "" {
		return &APIResponse{
			StatusCode: http.StatusBadRequest,
			Error: &APIError{
				Code:      "INVALID_REQUEST",
				Message:   "scenario_name is required",
				Status:    http.StatusBadRequest,
				Timestamp: time.Now(),
			},
		}
	}

	return &APIResponse{
		StatusCode: http.StatusAccepted,
		Body: ChaosTestResponse{
			ScenarioName: chaosReq.ScenarioName,
			Status:       "running",
		},
	}
}

func (ag *APIGateway) handleGetChaosTestResults(req *APIRequest) *APIResponse {
	testID := extractPathParam(req.Path, "{id}")

	return &APIResponse{
		StatusCode: http.StatusOK,
		Body: ChaosTestResponse{
			ScenarioName:       testID,
			Status:             "completed",
			RecoveryTime:       5,
			MaxAllowedDowntime: 10,
			ErrorCount:         0,
			DataLoss:           false,
		},
	}
}

func (ag *APIGateway) handleHealthCheck(req *APIRequest) *APIResponse {
	uptime := time.Since(ag.startTime)

	return &APIResponse{
		StatusCode: http.StatusOK,
		Body: map[string]interface{}{
			"status":         "healthy",
			"uptime_seconds": uptime.Seconds(),
			"requests":       ag.requestCount,
			"errors":         ag.errorCount,
		},
	}
}

// GetStats returns gateway statistics
func (ag *APIGateway) GetStats() map[string]interface{} {
	ag.gatewayLock.RLock()
	defer ag.gatewayLock.RUnlock()

	uptime := time.Since(ag.startTime)
	errorRate := 0.0
	if ag.requestCount > 0 {
		errorRate = float64(ag.errorCount) / float64(ag.requestCount) * 100
	}

	return map[string]interface{}{
		"total_requests":  ag.requestCount,
		"total_errors":    ag.errorCount,
		"error_rate":      errorRate,
		"uptime_seconds":  uptime.Seconds(),
		"endpoints":       len(ag.endpoints),
	}
}

// Helper function to extract path parameters
func extractPathParam(path string, paramName string) string {
	// Simple parameter extraction - in production would use proper routing
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
