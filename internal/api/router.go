package api

import (
	"log"
	"net/http"
	"time"

	"decentralized.host/pkg/control"
)

type Router struct {
	server *control.Server
	mux    *http.ServeMux
	logger *log.Logger
}

func NewRouter(server *control.Server, logger *log.Logger) *Router {
	r := &Router{
		server: server,
		mux:    http.NewServeMux(),
		logger: logger,
	}
	r.setupRoutes()
	return r
}

func (r *Router) setupRoutes() {
	// Health check
	r.mux.HandleFunc("/api/v1/health", r.middleware(r.handleHealth))

	// Dashboard endpoints
	r.mux.HandleFunc("/api/v1/dashboard/metrics", r.middleware(r.handleDashboardMetrics))
	r.mux.HandleFunc("/api/v1/dashboard/activity", r.middleware(r.handleDashboardActivity))

	// Nodes endpoints
	r.mux.HandleFunc("/api/v1/nodes", r.middleware(r.handleNodesList))
	r.mux.HandleFunc("/api/v1/nodes/", r.middleware(r.handleNodeDetail))

	// Deployments endpoints
	r.mux.HandleFunc("/api/v1/deployments", r.middleware(r.handleDeploymentsList))
	r.mux.HandleFunc("/api/v1/deployments/", r.middleware(r.handleDeploymentDetail))

	// Storage endpoints
	r.mux.HandleFunc("/api/v1/storage/buckets", r.middleware(r.handleStorageBuckets))
	r.mux.HandleFunc("/api/v1/storage/buckets/", r.middleware(r.handleBucketDetail))

	// Domains endpoints
	r.mux.HandleFunc("/api/v1/domains", r.middleware(r.handleDomainsList))
	r.mux.HandleFunc("/api/v1/domains/", r.middleware(r.handleDomainDetail))

	// Security endpoints
	r.mux.HandleFunc("/api/v1/security/certificates", r.middleware(r.handleCertificatesList))
	r.mux.HandleFunc("/api/v1/security/policies", r.middleware(r.handlePoliciesList))
	r.mux.HandleFunc("/api/v1/security/audit", r.middleware(r.handleAuditLog))

	// Analytics endpoints
	r.mux.HandleFunc("/api/v1/analytics/traffic", r.middleware(r.handleTrafficMetrics))
	r.mux.HandleFunc("/api/v1/analytics/errors", r.middleware(r.handleErrorAnalytics))
	r.mux.HandleFunc("/api/v1/analytics/latency", r.middleware(r.handleLatencyMetrics))

	// Team endpoints
	r.mux.HandleFunc("/api/v1/team/members", r.middleware(r.handleTeamMembers))

	// Settings endpoints
	r.mux.HandleFunc("/api/v1/settings", r.middleware(r.handleSettings))
	r.mux.HandleFunc("/api/v1/api-keys", r.middleware(r.handleApiKeys))

	// Evidence endpoints
	r.mux.HandleFunc("/api/v1/evidence", r.middleware(r.handleEvidenceList))
	r.mux.HandleFunc("/api/v1/evidence/", r.middleware(r.handleEvidenceDetail))
}

func (r *Router) middleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Trace-ID")
		w.Header().Set("Content-Type", "application/json")

		// Handle preflight
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Request logging and tracing
		start := time.Now()
		traceID := req.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = generateTraceID()
		}
		w.Header().Set("X-Trace-ID", traceID)

		// Call handler
		handler(w, req)

		// Log request
		duration := time.Since(start)
		r.logger.Printf("[%s] %s %s %d (%v)", traceID, req.Method, req.URL.Path, http.StatusOK, duration)
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}
