package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Response wraps all API responses
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	TraceID string      `json:"traceId"`
}

// Pagination helper
type PaginationParams struct {
	Offset int
	Limit  int
}

func parsePaginationParams(limit, offset int) PaginationParams {
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return PaginationParams{Offset: offset, Limit: limit}
}

// Response helpers
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}, traceID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := Response{
		Success: statusCode >= 200 && statusCode < 300,
		Data:    data,
		TraceID: traceID,
	}

	json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, statusCode int, message string, traceID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := Response{
		Success: false,
		Error:   message,
		TraceID: traceID,
	}

	json.NewEncoder(w).Encode(resp)
}

// Trace ID generation
func generateTraceID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Time helpers
func nowMs() int64 {
	return time.Now().UnixMilli()
}

func formatTime(ts int64) string {
	return time.UnixMilli(ts).UTC().Format(time.RFC3339)
}

func parseQueryInt(r *http.Request, key string, defaultValue int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}
	var i int
	fmt.Sscanf(val, "%d", &i)
	return i
}

func parseQueryString(r *http.Request, key string, defaultValue string) string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}
	return val
}
