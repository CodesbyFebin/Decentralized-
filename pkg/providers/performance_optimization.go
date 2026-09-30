package providers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CacheEntry represents a single cached item with TTL.
type CacheEntry struct {
	Key       string
	Value     interface{}
	ExpiresAt time.Time
	CreatedAt time.Time
	Hits      int64
	Size      int64
}

// CacheConfig defines caching behavior.
type CacheConfig struct {
	MaxSize       int64         // Maximum cache size in bytes
	MaxEntries    int           // Maximum number of entries
	DefaultTTL    time.Duration // Default time-to-live
	EvictionPolicy string        // "lru" or "lfu"
	EnableMetrics bool          // Track cache statistics
}

// CacheMetrics tracks cache performance.
type CacheMetrics struct {
	Hits              int64
	Misses            int64
	Evictions         int64
	CurrentSize       int64
	CurrentEntries    int
	AverageHitLatency time.Duration
	LatencyValues     []time.Duration
	mutex             sync.RWMutex
}

// RecordHit increments hit counter.
func (cm *CacheMetrics) RecordHit(latency time.Duration) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.Hits++
	cm.LatencyValues = append(cm.LatencyValues, latency)
	if len(cm.LatencyValues) > 1000 {
		cm.LatencyValues = cm.LatencyValues[1:]
	}
	cm.calculateAverageLatency()
}

// RecordMiss increments miss counter.
func (cm *CacheMetrics) RecordMiss() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.Misses++
}

// RecordEviction increments eviction counter.
func (cm *CacheMetrics) RecordEviction() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.Evictions++
}

func (cm *CacheMetrics) calculateAverageLatency() {
	if len(cm.LatencyValues) == 0 {
		cm.AverageHitLatency = 0
		return
	}
	sum := time.Duration(0)
	for _, v := range cm.LatencyValues {
		sum += v
	}
	cm.AverageHitLatency = sum / time.Duration(len(cm.LatencyValues))
}

// GetMetrics returns cache statistics.
func (cm *CacheMetrics) GetMetrics() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	total := cm.Hits + cm.Misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(cm.Hits) / float64(total) * 100
	}
	return map[string]interface{}{
		"hits":                 cm.Hits,
		"misses":               cm.Misses,
		"hit_rate_percent":     hitRate,
		"evictions":            cm.Evictions,
		"current_size_bytes":   cm.CurrentSize,
		"current_entries":      cm.CurrentEntries,
		"average_hit_latency":  cm.AverageHitLatency.String(),
	}
}

// MemoryCache implements in-memory caching with LRU eviction.
type MemoryCache struct {
	entries map[string]*CacheEntry
	config  *CacheConfig
	metrics *CacheMetrics
	mutex   sync.RWMutex
	ticker  *time.Ticker
}

// NewMemoryCache creates a memory cache with configuration.
func NewMemoryCache(config *CacheConfig) (*MemoryCache, error) {
	if config == nil {
		return nil, fmt.Errorf("cache config required")
	}
	if config.MaxSize < 1 {
		return nil, fmt.Errorf("max cache size must be >= 1")
	}
	if config.MaxEntries < 1 {
		return nil, fmt.Errorf("max entries must be >= 1")
	}
	mc := &MemoryCache{
		entries: make(map[string]*CacheEntry),
		config:  config,
		metrics: &CacheMetrics{},
		ticker:  time.NewTicker(1 * time.Minute),
	}
	go mc.expireEntries()
	return mc, nil
}

// Set stores a value in cache with default TTL.
func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if ttl == 0 {
		ttl = mc.config.DefaultTTL
	}

	entry := &CacheEntry{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}

	if len(mc.entries) >= mc.config.MaxEntries {
		mc.evictOne()
	}

	mc.entries[key] = entry
	mc.metrics.CurrentEntries = len(mc.entries)
	return nil
}

// Get retrieves a value from cache.
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	start := time.Now()
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	entry, exists := mc.entries[key]
	if !exists {
		mc.metrics.RecordMiss()
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		mc.metrics.RecordMiss()
		return nil, false
	}

	entry.Hits++
	latency := time.Since(start)
	mc.metrics.RecordHit(latency)
	return entry.Value, true
}

// Delete removes a key from cache.
func (mc *MemoryCache) Delete(key string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	delete(mc.entries, key)
	mc.metrics.CurrentEntries = len(mc.entries)
}

// Clear removes all entries.
func (mc *MemoryCache) Clear() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.entries = make(map[string]*CacheEntry)
	mc.metrics.CurrentEntries = 0
}

func (mc *MemoryCache) evictOne() {
	if mc.config.EvictionPolicy == "lfu" {
		mc.evictLFU()
	} else {
		mc.evictLRU()
	}
	mc.metrics.RecordEviction()
}

func (mc *MemoryCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	for key, entry := range mc.entries {
		if oldestTime.IsZero() || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}
	if oldestKey != "" {
		delete(mc.entries, oldestKey)
	}
}

func (mc *MemoryCache) evictLFU() {
	var leastKey string
	var leastHits int64 = -1
	for key, entry := range mc.entries {
		if leastHits == -1 || entry.Hits < leastHits {
			leastKey = key
			leastHits = entry.Hits
		}
	}
	if leastKey != "" {
		delete(mc.entries, leastKey)
	}
}

func (mc *MemoryCache) expireEntries() {
	for range mc.ticker.C {
		mc.mutex.Lock()
		now := time.Now()
		for key, entry := range mc.entries {
			if now.After(entry.ExpiresAt) {
				delete(mc.entries, key)
			}
		}
		mc.metrics.CurrentEntries = len(mc.entries)
		mc.mutex.Unlock()
	}
}

// Stop terminates expiration goroutine.
func (mc *MemoryCache) Stop() {
	mc.ticker.Stop()
}

// GetMetrics returns cache statistics.
func (mc *MemoryCache) GetMetrics() map[string]interface{} {
	return mc.metrics.GetMetrics()
}

// QueryOptimizer suggests database optimizations.
type QueryOptimizer struct {
	slowQueryThreshold time.Duration
	queryPatterns      map[string]*QueryPattern
	mutex              sync.RWMutex
}

// QueryPattern tracks query execution patterns.
type QueryPattern struct {
	Query           string
	Count           int64
	TotalLatency    time.Duration
	AverageLatency  time.Duration
	MaxLatency      time.Duration
	Suggestions     []string
}

// NewQueryOptimizer creates a query optimizer.
func NewQueryOptimizer(slowQueryThreshold time.Duration) *QueryOptimizer {
	return &QueryOptimizer{
		slowQueryThreshold: slowQueryThreshold,
		queryPatterns:      make(map[string]*QueryPattern),
	}
}

// RecordQuery records query execution for analysis.
func (qo *QueryOptimizer) RecordQuery(query string, duration time.Duration) {
	qo.mutex.Lock()
	defer qo.mutex.Unlock()

	pattern, exists := qo.queryPatterns[query]
	if !exists {
		pattern = &QueryPattern{Query: query, Suggestions: []string{}}
		qo.queryPatterns[query] = pattern
	}

	pattern.Count++
	pattern.TotalLatency += duration
	pattern.AverageLatency = pattern.TotalLatency / time.Duration(pattern.Count)

	if duration > pattern.MaxLatency {
		pattern.MaxLatency = duration
	}

	if duration > qo.slowQueryThreshold && len(pattern.Suggestions) == 0 {
		pattern.Suggestions = qo.generateSuggestions(query, duration)
	}
}

func (qo *QueryOptimizer) generateSuggestions(query string, duration time.Duration) []string {
	suggestions := []string{}
	if duration > qo.slowQueryThreshold {
		suggestions = append(suggestions, "Consider adding indexes on frequently queried columns")
		suggestions = append(suggestions, "Review query plan for full table scans")
		suggestions = append(suggestions, "Consider query parameterization for prepared statements")
		suggestions = append(suggestions, "Analyze join conditions for efficiency")
	}
	return suggestions
}

// GetSlowQueries returns queries exceeding threshold.
func (qo *QueryOptimizer) GetSlowQueries() []*QueryPattern {
	qo.mutex.RLock()
	defer qo.mutex.RUnlock()

	slow := []*QueryPattern{}
	for _, pattern := range qo.queryPatterns {
		if pattern.AverageLatency > qo.slowQueryThreshold {
			slow = append(slow, pattern)
		}
	}
	return slow
}

// IndexRecommendation suggests a missing index.
type IndexRecommendation struct {
	Table      string
	Columns    []string
	Reason     string
	EstimatedGain string
}

// IndexOptimizer identifies missing database indexes.
type IndexOptimizer struct {
	existingIndexes map[string][]string
	queryAnalysis   map[string]int
	recommendations []*IndexRecommendation
	mutex           sync.RWMutex
}

// NewIndexOptimizer creates index optimizer.
func NewIndexOptimizer() *IndexOptimizer {
	return &IndexOptimizer{
		existingIndexes: make(map[string][]string),
		queryAnalysis:   make(map[string]int),
	}
}

// RegisterIndex registers an existing index.
func (io *IndexOptimizer) RegisterIndex(table string, columns []string) {
	io.mutex.Lock()
	defer io.mutex.Unlock()
	key := fmt.Sprintf("%s:%s", table, fmt.Sprint(columns))
	io.existingIndexes[key] = columns
}

// AnalyzeQuery analyzes query for index recommendations.
func (io *IndexOptimizer) AnalyzeQuery(query string, table string, columns []string) {
	io.mutex.Lock()
	defer io.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", table, fmt.Sprint(columns))
	io.queryAnalysis[key]++

	// Check if index exists
	if _, exists := io.existingIndexes[key]; !exists && io.queryAnalysis[key] > 5 {
		rec := &IndexRecommendation{
			Table:   table,
			Columns: columns,
			Reason:  fmt.Sprintf("Frequently queried columns (%d queries)", io.queryAnalysis[key]),
			EstimatedGain: "30-50% query latency reduction",
		}
		io.recommendations = append(io.recommendations, rec)
	}
}

// GetRecommendations returns suggested indexes.
func (io *IndexOptimizer) GetRecommendations() []*IndexRecommendation {
	io.mutex.RLock()
	defer io.mutex.RUnlock()
	recs := make([]*IndexRecommendation, len(io.recommendations))
	copy(recs, io.recommendations)
	return recs
}

// ConnectionPool manages database connections.
type ConnectionPool struct {
	maxConnections    int
	activeConnections int
	idleConnections   int
	totalCreated      int64
	totalClosed       int64
	acquireLatencies  []time.Duration
	mutex             sync.RWMutex
}

// NewConnectionPool creates a connection pool.
func NewConnectionPool(maxConnections int) *ConnectionPool {
	return &ConnectionPool{
		maxConnections:   maxConnections,
		acquireLatencies: make([]time.Duration, 0, 1000),
	}
}

// AcquireConnection acquires a connection (simulated).
func (cp *ConnectionPool) AcquireConnection(ctx context.Context) error {
	start := time.Now()
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if cp.activeConnections >= cp.maxConnections {
		return fmt.Errorf("connection pool exhausted: %d/%d", cp.activeConnections, cp.maxConnections)
	}

	cp.activeConnections++
	cp.totalCreated++
	latency := time.Since(start)
	cp.acquireLatencies = append(cp.acquireLatencies, latency)
	if len(cp.acquireLatencies) > 1000 {
		cp.acquireLatencies = cp.acquireLatencies[1:]
	}
	return nil
}

// ReleaseConnection releases a connection.
func (cp *ConnectionPool) ReleaseConnection() {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	if cp.activeConnections > 0 {
		cp.activeConnections--
		cp.idleConnections++
		cp.totalClosed++
	}
}

// GetStats returns pool statistics.
func (cp *ConnectionPool) GetStats() map[string]interface{} {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	avgLatency := time.Duration(0)
	if len(cp.acquireLatencies) > 0 {
		sum := time.Duration(0)
		for _, l := range cp.acquireLatencies {
			sum += l
		}
		avgLatency = sum / time.Duration(len(cp.acquireLatencies))
	}

	return map[string]interface{}{
		"active_connections": cp.activeConnections,
		"idle_connections":   cp.idleConnections,
		"max_connections":    cp.maxConnections,
		"total_created":      cp.totalCreated,
		"total_closed":       cp.totalClosed,
		"average_acquire_latency": avgLatency.String(),
	}
}

// PerformanceProfile captures performance metrics.
type PerformanceProfile struct {
	Timestamp           time.Time
	CacheMetrics        map[string]interface{}
	SlowQueries         []*QueryPattern
	IndexRecommendations []*IndexRecommendation
	ConnectionPoolStats map[string]interface{}
}

// PerformanceManager orchestrates performance optimization.
type PerformanceManager struct {
	cache            *MemoryCache
	queryOptimizer   *QueryOptimizer
	indexOptimizer   *IndexOptimizer
	connectionPool   *ConnectionPool
	profiles         []*PerformanceProfile
	maxProfiles      int
	mutex            sync.RWMutex
}

// NewPerformanceManager creates performance manager.
func NewPerformanceManager(cacheConfig *CacheConfig, maxConnections int) (*PerformanceManager, error) {
	cache, err := NewMemoryCache(cacheConfig)
	if err != nil {
		return nil, err
	}

	return &PerformanceManager{
		cache:          cache,
		queryOptimizer: NewQueryOptimizer(100 * time.Millisecond),
		indexOptimizer: NewIndexOptimizer(),
		connectionPool: NewConnectionPool(maxConnections),
		profiles:       make([]*PerformanceProfile, 0),
		maxProfiles:    1000,
	}, nil
}

// RecordQueryPerformance records query execution metrics.
func (pm *PerformanceManager) RecordQueryPerformance(query string, duration time.Duration) {
	pm.queryOptimizer.RecordQuery(query, duration)
	if duration > 100*time.Millisecond {
		pm.indexOptimizer.AnalyzeQuery(query, "campaigns", []string{"id"})
	}
}

// GetPerformanceProfile captures current performance state.
func (pm *PerformanceManager) GetPerformanceProfile() *PerformanceProfile {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	profile := &PerformanceProfile{
		Timestamp:           time.Now(),
		CacheMetrics:        pm.cache.GetMetrics(),
		SlowQueries:         pm.queryOptimizer.GetSlowQueries(),
		IndexRecommendations: pm.indexOptimizer.GetRecommendations(),
		ConnectionPoolStats: pm.connectionPool.GetStats(),
	}

	pm.profiles = append(pm.profiles, profile)
	if len(pm.profiles) > pm.maxProfiles {
		pm.profiles = pm.profiles[1:]
	}

	return profile
}

// GetCacheHitRate returns cache hit rate percentage.
func (pm *PerformanceManager) GetCacheHitRate() float64 {
	metrics := pm.cache.GetMetrics()
	hits := metrics["hits"].(int64)
	misses := metrics["misses"].(int64)
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

// Close stops performance manager.
func (pm *PerformanceManager) Close() {
	pm.cache.Stop()
}
