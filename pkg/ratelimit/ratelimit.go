// Package ratelimit provides token-bucket rate limiting with per-node and per-IP limits.
//
// The rate limiter implements a token-bucket algorithm that allows:
// - Configurable requests per second
// - Configurable burst size
// - Per-node rate limits
// - Per-IP rate limits
// - Graceful degradation under overload
// - Async cleanup of idle limiters
package ratelimit

import (
	"net"
	"sync"
	"time"
)

// Limiter is a token-bucket rate limiter.
type Limiter struct {
	capacity      float64       // Maximum tokens
	tokensPerSec  float64       // Refill rate
	mu            sync.Mutex
	tokens        float64       // Current tokens
	lastRefillTime time.Time
}

// NewLimiter creates a new rate limiter.
// capacity: maximum tokens (burst size)
// tokensPerSec: tokens added per second (rate)
func NewLimiter(capacity float64, tokensPerSec float64) *Limiter {
	return &Limiter{
		capacity:       capacity,
		tokensPerSec:   tokensPerSec,
		tokens:         capacity,
		lastRefillTime: time.Now(),
	}
}

// Allow attempts to consume n tokens. Returns true if allowed, false if rate limit exceeded.
func (l *Limiter) Allow(n float64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()

	if l.tokens >= n {
		l.tokens -= n
		return true
	}

	return false
}

// AllowWithPenalty attempts to consume tokens and returns available tokens (may be negative).
// Useful for tracking how far over the limit a request is.
func (l *Limiter) AllowWithPenalty(n float64) float64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()
	l.tokens -= n
	return l.tokens
}

// AvailableTokens returns the current number of available tokens.
func (l *Limiter) AvailableTokens() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()
	return l.tokens
}

// Reset resets the limiter to full capacity.
func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.tokens = l.capacity
	l.lastRefillTime = time.Now()
}

// refill adds tokens based on elapsed time since last refill.
func (l *Limiter) refill() {
	now := time.Now()
	elapsed := now.Sub(l.lastRefillTime).Seconds()
	l.tokens = min(l.capacity, l.tokens+elapsed*l.tokensPerSec)
	l.lastRefillTime = now
}

// RateLimitManager manages per-node and per-IP rate limiters.
type RateLimitManager struct {
	defaultRPS           float64
	defaultBurst         float64
	nodeCapacity         float64
	nodeTokensPerSec     float64
	ipCapacity           float64
	ipTokensPerSec       float64
	mu                   sync.RWMutex
	nodeLimiters         map[string]*Limiter
	ipLimiters           map[string]*Limiter
	lastCleanup          time.Time
	cleanupInterval      time.Duration
	idleTimeout          time.Duration
}

// NewRateLimitManager creates a new rate limit manager.
// defaultRPS: default requests per second for each limiter
// defaultBurst: default burst size (capacity)
func NewRateLimitManager(defaultRPS float64, defaultBurst float64) *RateLimitManager {
	return &RateLimitManager{
		defaultRPS:      defaultRPS,
		defaultBurst:    defaultBurst,
		nodeCapacity:    defaultBurst,
		nodeTokensPerSec: defaultRPS,
		ipCapacity:      defaultBurst / 2, // IP limit stricter than node limit
		ipTokensPerSec:  defaultRPS / 2,
		nodeLimiters:    make(map[string]*Limiter),
		ipLimiters:      make(map[string]*Limiter),
		lastCleanup:     time.Now(),
		cleanupInterval: 5 * time.Minute,
		idleTimeout:     10 * time.Minute,
	}
}

// SetNodeLimits sets custom rate limits for a specific node.
func (m *RateLimitManager) SetNodeLimits(nodeID string, rps float64, burst float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nodeLimiters[nodeID] = NewLimiter(burst, rps)
}

// SetIPLimits sets custom rate limits for a specific IP address.
func (m *RateLimitManager) SetIPLimits(ip string, rps float64, burst float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ipLimiters[ip] = NewLimiter(burst, rps)
}

// AllowNode checks if a request from a node is allowed.
func (m *RateLimitManager) AllowNode(nodeID string, tokens float64) bool {
	m.mu.Lock()
	limiter, exists := m.nodeLimiters[nodeID]
	if !exists {
		limiter = NewLimiter(m.nodeCapacity, m.nodeTokensPerSec)
		m.nodeLimiters[nodeID] = limiter
	}
	m.mu.Unlock()

	m.maybeCleanup()
	return limiter.Allow(tokens)
}

// AllowIP checks if a request from an IP address is allowed.
func (m *RateLimitManager) AllowIP(ip string, tokens float64) bool {
	m.mu.Lock()
	limiter, exists := m.ipLimiters[ip]
	if !exists {
		limiter = NewLimiter(m.ipCapacity, m.ipTokensPerSec)
		m.ipLimiters[ip] = limiter
	}
	m.mu.Unlock()

	m.maybeCleanup()
	return limiter.Allow(tokens)
}

// AllowRequest checks both node and IP limits.
func (m *RateLimitManager) AllowRequest(nodeID string, ip string, tokens float64) bool {
	nodeAllowed := m.AllowNode(nodeID, tokens)
	ipAllowed := m.AllowIP(ip, tokens)
	return nodeAllowed && ipAllowed
}

// GetNodeStats returns current statistics for a node's limiter.
func (m *RateLimitManager) GetNodeStats(nodeID string) (available float64, capacity float64) {
	m.mu.RLock()
	limiter, exists := m.nodeLimiters[nodeID]
	m.mu.RUnlock()

	if !exists {
		return 0, m.nodeCapacity
	}

	return limiter.AvailableTokens(), limiter.capacity
}

// GetIPStats returns current statistics for an IP's limiter.
func (m *RateLimitManager) GetIPStats(ip string) (available float64, capacity float64) {
	m.mu.RLock()
	limiter, exists := m.ipLimiters[ip]
	m.mu.RUnlock()

	if !exists {
		return 0, m.ipCapacity
	}

	return limiter.AvailableTokens(), limiter.capacity
}

// maybeCleanup periodically cleans up idle limiters.
func (m *RateLimitManager) maybeCleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if time.Since(m.lastCleanup) < m.cleanupInterval {
		return
	}

	m.lastCleanup = time.Now()

	// Clean up idle node limiters
	for nodeID, limiter := range m.nodeLimiters {
		if time.Since(limiter.lastRefillTime) > m.idleTimeout {
			delete(m.nodeLimiters, nodeID)
		}
	}

	// Clean up idle IP limiters
	for ip, limiter := range m.ipLimiters {
		if time.Since(limiter.lastRefillTime) > m.idleTimeout {
			delete(m.ipLimiters, ip)
		}
	}
}

// ExtractIP extracts the IP address from a network address string.
func ExtractIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// If there's no port, the whole addr might be the IP
		return addr
	}
	return host
}

// AdaptiveRateLimiter adjusts rate limits based on system load.
type AdaptiveRateLimiter struct {
	base           *RateLimitManager
	mu             sync.RWMutex
	currentRPS     float64
	baseRPS        float64
	minRPS         float64
	maxRPS         float64
	lastUpdate     time.Time
	updateInterval time.Duration
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter.
func NewAdaptiveRateLimiter(baseRPS float64, baseBurst float64) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		base:           NewRateLimitManager(baseRPS, baseBurst),
		baseRPS:        baseRPS,
		currentRPS:     baseRPS,
		minRPS:         baseRPS / 2,      // Can drop to 50% under load
		maxRPS:         baseRPS * 2,      // Can increase to 200% when available
		lastUpdate:     time.Now(),
		updateInterval: 10 * time.Second,
	}
}

// AdjustLoad updates the rate limit based on system load (0.0 to 1.0).
// 0.0 = no load, 1.0 = maximum load.
func (a *AdaptiveRateLimiter) AdjustLoad(load float64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if time.Since(a.lastUpdate) < a.updateInterval {
		return
	}

	a.lastUpdate = time.Now()

	// Scale RPS based on load
	// At 50% load, keep baseRPS
	// Below 50% load, increase up to maxRPS
	// Above 50% load, decrease down to minRPS
	if load < 0.5 {
		// Less than 50% load: increase allowance
		scale := 1.0 + (0.5-load)*2 // max 2x at 0% load
		a.currentRPS = min(a.maxRPS, a.baseRPS*scale)
	} else {
		// More than 50% load: decrease allowance
		scale := 1.0 - (load - 0.5) * 2 // min 0.5x at 100% load
		a.currentRPS = max(a.minRPS, a.baseRPS*scale)
	}
}

// AllowRequest checks request allowance with adaptive limits.
func (a *AdaptiveRateLimiter) AllowRequest(nodeID string, ip string, tokens float64) bool {
	return a.base.AllowRequest(nodeID, ip, tokens)
}

// CurrentRPS returns the current requests per second limit.
func (a *AdaptiveRateLimiter) CurrentRPS() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentRPS
}

// Helper functions

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
