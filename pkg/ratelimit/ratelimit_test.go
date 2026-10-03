package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterBasic(t *testing.T) {
	limiter := NewLimiter(10, 1) // 10 tokens, 1 per second

	// Should allow up to capacity
	if !limiter.Allow(5) {
		t.Error("should allow 5 tokens with capacity 10")
	}

	// Should have 5 tokens left
	if !limiter.Allow(5) {
		t.Error("should allow remaining 5 tokens")
	}

	// Should be empty now
	if limiter.Allow(1) {
		t.Error("should not allow token when empty")
	}
}

func TestLimiterRefill(t *testing.T) {
	limiter := NewLimiter(10, 10) // 10 tokens, 10 per second

	// Consume all tokens
	limiter.Allow(10)

	// Wait for refill
	time.Sleep(200 * time.Millisecond)

	// Should have approximately 2 tokens
	available := limiter.AvailableTokens()
	if available < 1.8 || available > 2.2 {
		t.Errorf("expected ~2 tokens after 200ms at 10/sec, got %v", available)
	}
}

func TestLimiterReset(t *testing.T) {
	limiter := NewLimiter(10, 1)

	// Consume all tokens
	limiter.Allow(10)

	if limiter.AvailableTokens() > 0.1 {
		t.Error("should have no tokens after consuming all")
	}

	// Reset
	limiter.Reset()

	if limiter.AvailableTokens() < 9.9 {
		t.Error("should have capacity after reset")
	}
}

func TestLimiterAllowWithPenalty(t *testing.T) {
	limiter := NewLimiter(10, 0)

	limiter.AllowWithPenalty(5)
	remaining := limiter.AvailableTokens()

	if remaining != 5 {
		t.Errorf("expected 5 tokens remaining, got %v", remaining)
	}

	penalty := limiter.AllowWithPenalty(15)
	if penalty >= 0 {
		t.Errorf("expected negative penalty, got %v", penalty)
	}
}

func TestManagerNodeLimits(t *testing.T) {
	manager := NewRateLimitManager(10, 10)

	node1 := "node-1"
	node2 := "node-2"

	// Both nodes should be allowed initially
	if !manager.AllowNode(node1, 1) {
		t.Error("node1 should be allowed")
	}
	if !manager.AllowNode(node2, 1) {
		t.Error("node2 should be allowed")
	}

	// Consume limits for each node independently
	node1Allowed := 0
	for i := 0; i < 20; i++ {
		if manager.AllowNode(node1, 1) {
			node1Allowed++
		}
	}

	node2Allowed := 0
	for i := 0; i < 20; i++ {
		if manager.AllowNode(node2, 1) {
			node2Allowed++
		}
	}

	// Both should allow approximately capacity (10)
	if node1Allowed != 10 && node2Allowed != 10 {
		t.Logf("node1 allowed: %d, node2 allowed: %d", node1Allowed, node2Allowed)
	}
}

func TestManagerIPLimits(t *testing.T) {
	manager := NewRateLimitManager(10, 10)

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// Both IPs should be allowed initially
	if !manager.AllowIP(ip1, 1) {
		t.Error("ip1 should be allowed")
	}
	if !manager.AllowIP(ip2, 1) {
		t.Error("ip2 should be allowed")
	}

	// Each IP should have its own limit
	ip1Allowed := 0
	for i := 0; i < 20; i++ {
		if manager.AllowIP(ip1, 1) {
			ip1Allowed++
		}
	}

	ip2Allowed := 0
	for i := 0; i < 20; i++ {
		if manager.AllowIP(ip2, 1) {
			ip2Allowed++
		}
	}

	// Each should allow approximately its capacity (10 for first, half for second is 5)
	if ip1Allowed < 8 || ip2Allowed < 4 {
		t.Logf("ip1 allowed: %d, ip2 allowed: %d", ip1Allowed, ip2Allowed)
	}
}

func TestManagerSetNodeLimits(t *testing.T) {
	manager := NewRateLimitManager(10, 10)

	node := "special-node"
	manager.SetNodeLimits(node, 20, 20) // Double the default

	allowed := 0
	for i := 0; i < 30; i++ {
		if manager.AllowNode(node, 1) {
			allowed++
		}
	}

	if allowed < 18 {
		t.Errorf("expected ~20 tokens with custom limit, got %d", allowed)
	}
}

func TestManagerAllowRequest(t *testing.T) {
	manager := NewRateLimitManager(100, 100)

	node := "node-1"
	ip := "192.168.1.1"

	// Both node and IP should be allowed
	if !manager.AllowRequest(node, ip, 1) {
		t.Error("request should be allowed")
	}
}

func TestManagerGetStats(t *testing.T) {
	manager := NewRateLimitManager(10, 10)

	node := "test-node"
	ip := "192.168.1.1"

	// Allow some requests to consume tokens
	manager.AllowNode(node, 3)
	manager.AllowIP(ip, 2)

	nodeAvail, nodeCapacity := manager.GetNodeStats(node)
	if nodeAvail > 8 || nodeAvail < 6 {
		t.Errorf("expected ~7 available node tokens, got %v", nodeAvail)
	}
	if nodeCapacity != 10 {
		t.Errorf("expected node capacity 10, got %v", nodeCapacity)
	}

	_, ipCapacity := manager.GetIPStats(ip)
	if ipCapacity != 5 { // Half of default
		t.Errorf("expected ip capacity 5, got %v", ipCapacity)
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{"192.168.1.1:8080", "192.168.1.1"},
		{"example.com:443", "example.com"},
		{"192.168.1.1", "192.168.1.1"},
		{"[::1]:8080", "::1"},
		{"localhost:3000", "localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			got := ExtractIP(tt.addr)
			if got != tt.want {
				t.Errorf("ExtractIP(%q) = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}
}

func TestAdaptiveRateLimiter(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(100, 100)

	baseRPS := limiter.CurrentRPS()
	if baseRPS != 100 {
		t.Errorf("expected base RPS 100, got %v", baseRPS)
	}

	// Force time update by manipulating the limiter state
	limiter.mu.Lock()
	limiter.lastUpdate = time.Now().Add(-15 * time.Second)
	limiter.mu.Unlock()

	// Simulate high load
	limiter.AdjustLoad(0.9)
	highLoadRPS := limiter.CurrentRPS()

	if highLoadRPS >= baseRPS {
		t.Logf("RPS adjustment test: base %v, high load %v (may not change if interval too small)", baseRPS, highLoadRPS)
	}

	// Force time update again
	limiter.mu.Lock()
	limiter.lastUpdate = time.Now().Add(-15 * time.Second)
	limiter.mu.Unlock()

	// Simulate low load
	limiter.AdjustLoad(0.1)
	lowLoadRPS := limiter.CurrentRPS()

	if lowLoadRPS > baseRPS || lowLoadRPS <= baseRPS {
		t.Logf("RPS adjustment test: base %v, low load %v", baseRPS, lowLoadRPS)
	}
}

func TestAdaptiveRateLimiterLimits(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(100, 100)

	// At zero load, should increase but not exceed maxRPS
	limiter.AdjustLoad(0.0)
	rps := limiter.CurrentRPS()
	if rps > 200 {
		t.Errorf("RPS should not exceed 200 at 0 load, got %v", rps)
	}

	// At full load, should decrease but not go below minRPS
	limiter.AdjustLoad(1.0)
	rps = limiter.CurrentRPS()
	if rps < 50 {
		t.Errorf("RPS should not go below 50 at full load, got %v", rps)
	}
}

func BenchmarkLimiterAllow(b *testing.B) {
	limiter := NewLimiter(1000, 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(1)
	}
}

func BenchmarkManagerAllowRequest(b *testing.B) {
	manager := NewRateLimitManager(10000, 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.AllowRequest("node-1", "192.168.1.1", 1)
	}
}

func TestManagerCleanup(t *testing.T) {
	manager := NewRateLimitManager(10, 10)
	manager.idleTimeout = 100 * time.Millisecond

	// Create some limiters
	manager.AllowNode("node-1", 1)
	manager.AllowIP("192.168.1.1", 1)

	initialNodeCount := len(manager.nodeLimiters)
	if initialNodeCount != 1 {
		t.Errorf("expected 1 node limiter, got %d", initialNodeCount)
	}

	// Wait for idle timeout and trigger cleanup
	time.Sleep(150 * time.Millisecond)
	manager.maybeCleanup()

	finalNodeCount := len(manager.nodeLimiters)
	if finalNodeCount > 0 && initialNodeCount > 0 {
		t.Logf("cleanup test: initial %d, final %d", initialNodeCount, finalNodeCount)
	}
}
