package providers

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// TraceSpan represents a distributed trace span.
type TraceSpan struct {
	TraceID   string
	SpanID    string
	ParentID  string
	Operation string
	Status    string // "pending", "success", "error"
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Tags      map[string]interface{}
	Logs      []map[string]interface{}
	Error     error
}

// MemoryMetrics captures memory usage statistics.
type MemoryMetrics struct {
	Timestamp       time.Time
	Alloc           uint64
	TotalAlloc      uint64
	Sys             uint64
	NumGC           uint32
	GCPausedTotal   time.Duration
	Goroutines      int
	HeapAlloc       uint64
	HeapSys         uint64
	HeapObjects     uint64
}

// CPUMetrics captures CPU usage statistics.
type CPUMetrics struct {
	Timestamp     time.Time
	UserTime      time.Duration
	SystemTime    time.Duration
	NumGoroutines int
	GCRuns        uint32
}

// ProfileSnapshot captures a performance profile snapshot.
type ProfileSnapshot struct {
	Timestamp    time.Time
	Memory       *MemoryMetrics
	CPU          *CPUMetrics
	Goroutines   map[string]int
	RequestCount int64
	ErrorCount   int64
}

// DistributedTracer manages request tracing.
type DistributedTracer struct {
	spans map[string]*TraceSpan
	mutex sync.RWMutex
}

// NewDistributedTracer creates a distributed tracer.
func NewDistributedTracer() *DistributedTracer {
	return &DistributedTracer{
		spans: make(map[string]*TraceSpan),
	}
}

// StartSpan creates and starts a new trace span.
func (dt *DistributedTracer) StartSpan(traceID, spanID, parentID, operation string) *TraceSpan {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	span := &TraceSpan{
		TraceID:   traceID,
		SpanID:    spanID,
		ParentID:  parentID,
		Operation: operation,
		Status:    "pending",
		StartTime: time.Now(),
		Tags:      make(map[string]interface{}),
		Logs:      make([]map[string]interface{}, 0),
	}

	dt.spans[spanID] = span
	return span
}

// EndSpan completes a trace span.
func (dt *DistributedTracer) EndSpan(spanID string, success bool, err error) {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	span, exists := dt.spans[spanID]
	if !exists {
		return
	}

	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)

	if success {
		span.Status = "success"
	} else {
		span.Status = "error"
		span.Error = err
	}
}

// AddTag adds a key-value tag to a span.
func (dt *DistributedTracer) AddTag(spanID, key string, value interface{}) {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	span, exists := dt.spans[spanID]
	if !exists {
		return
	}

	span.Tags[key] = value
}

// AddLog adds a log entry to a span.
func (dt *DistributedTracer) AddLog(spanID string, message string, fields map[string]interface{}) {
	dt.mutex.Lock()
	defer dt.mutex.Unlock()

	span, exists := dt.spans[spanID]
	if !exists {
		return
	}

	logEntry := map[string]interface{}{
		"timestamp": time.Now(),
		"message":   message,
	}
	for k, v := range fields {
		logEntry[k] = v
	}
	span.Logs = append(span.Logs, logEntry)
}

// GetSpan retrieves a span by ID.
func (dt *DistributedTracer) GetSpan(spanID string) *TraceSpan {
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()
	return dt.spans[spanID]
}

// GetTrace returns all spans for a trace.
func (dt *DistributedTracer) GetTrace(traceID string) []*TraceSpan {
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()

	trace := []*TraceSpan{}
	for _, span := range dt.spans {
		if span.TraceID == traceID {
			trace = append(trace, span)
		}
	}
	return trace
}

// ProfileMonitor tracks system performance metrics.
type ProfileMonitor struct {
	snapshots     []*ProfileSnapshot
	maxSnapshots  int
	lastMemStats  runtime.MemStats
	requestCount  int64
	errorCount    int64
	startTime     time.Time
	mutex         sync.RWMutex
}

// NewProfileMonitor creates a profile monitor.
func NewProfileMonitor(maxSnapshots int) *ProfileMonitor {
	return &ProfileMonitor{
		snapshots:    make([]*ProfileSnapshot, 0, maxSnapshots),
		maxSnapshots: maxSnapshots,
		startTime:    time.Now(),
	}
}

// CaptureMemoryMetrics captures current memory statistics.
func (pm *ProfileMonitor) CaptureMemoryMetrics() *MemoryMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &MemoryMetrics{
		Timestamp:     time.Now(),
		Alloc:         m.Alloc,
		TotalAlloc:    m.TotalAlloc,
		Sys:           m.Sys,
		NumGC:         m.NumGC,
		GCPausedTotal: time.Duration(m.PauseNs[(m.NumGC+255)%256]),
		Goroutines:    runtime.NumGoroutine(),
		HeapAlloc:     m.HeapAlloc,
		HeapSys:       m.HeapSys,
		HeapObjects:   m.HeapObjects,
	}
}

// CaptureCPUMetrics captures current CPU statistics.
func (pm *ProfileMonitor) CaptureCPUMetrics() *CPUMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &CPUMetrics{
		Timestamp:     time.Now(),
		NumGoroutines: runtime.NumGoroutine(),
		GCRuns:        m.NumGC,
	}
}

// CaptureProfile captures a full performance snapshot.
func (pm *ProfileMonitor) CaptureProfile() *ProfileSnapshot {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	snapshot := &ProfileSnapshot{
		Timestamp:    time.Now(),
		Memory:       pm.CaptureMemoryMetrics(),
		CPU:          pm.CaptureCPUMetrics(),
		Goroutines:   pm.getGoroutineStats(),
		RequestCount: pm.requestCount,
		ErrorCount:   pm.errorCount,
	}

	pm.snapshots = append(pm.snapshots, snapshot)
	if len(pm.snapshots) > pm.maxSnapshots {
		pm.snapshots = pm.snapshots[1:]
	}

	return snapshot
}

func (pm *ProfileMonitor) getGoroutineStats() map[string]int {
	// Simplified goroutine classification
	return map[string]int{
		"total":      runtime.NumGoroutine(),
		"active":     runtime.NumGoroutine() - 1, // Approximate
		"idle":       1,
		"max_threads": runtime.NumCPU(),
	}
}

// RecordRequest increments request counter.
func (pm *ProfileMonitor) RecordRequest(success bool) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.requestCount++
	if !success {
		pm.errorCount++
	}
}

// GetProfileSnapshots returns captured snapshots.
func (pm *ProfileMonitor) GetProfileSnapshots(limit int) []*ProfileSnapshot {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if limit > len(pm.snapshots) {
		limit = len(pm.snapshots)
	}

	snapshots := make([]*ProfileSnapshot, limit)
	copy(snapshots, pm.snapshots[len(pm.snapshots)-limit:])
	return snapshots
}

// GetMemoryTrend analyzes memory usage trend.
func (pm *ProfileMonitor) GetMemoryTrend() map[string]interface{} {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if len(pm.snapshots) == 0 {
		return map[string]interface{}{}
	}

	first := pm.snapshots[0]
	last := pm.snapshots[len(pm.snapshots)-1]

	allocGrowth := int64(last.Memory.Alloc) - int64(first.Memory.Alloc)
	goroutineGrowth := last.Goroutines["total"] - first.Goroutines["total"]

	return map[string]interface{}{
		"alloc_growth_bytes":    allocGrowth,
		"goroutine_growth":      goroutineGrowth,
		"current_alloc_bytes":   last.Memory.Alloc,
		"current_goroutines":    last.Goroutines["total"],
		"gc_runs":               last.CPU.GCRuns,
		"gc_paused_total_ms":    last.Memory.GCPausedTotal.Milliseconds(),
	}
}

// DebugInfo contains debug information.
type DebugInfo struct {
	Timestamp        time.Time
	Uptime           time.Duration
	Goroutines       int
	MemoryAlloc      uint64
	MemorySys        uint64
	HeapAlloc        uint64
	HeapObjects      uint64
	GCRuns           uint32
	RequestCount     int64
	ErrorCount       int64
	ErrorRate        float64
	TopSpans         []*TraceSpan
	RecentSnapshots  []*ProfileSnapshot
}

// Debugger provides debugging utilities.
type Debugger struct {
	tracer         *DistributedTracer
	monitor        *ProfileMonitor
	enableTracing  bool
	enableProfiling bool
	mutex          sync.RWMutex
}

// NewDebugger creates a debugger instance.
func NewDebugger() *Debugger {
	return &Debugger{
		tracer:          NewDistributedTracer(),
		monitor:         NewProfileMonitor(100),
		enableTracing:   true,
		enableProfiling: true,
	}
}

// EnableTracing enables/disables tracing.
func (d *Debugger) EnableTracing(enable bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.enableTracing = enable
}

// EnableProfiling enables/disables profiling.
func (d *Debugger) EnableProfiling(enable bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.enableProfiling = enable
}

// StartTrace creates a new trace.
func (d *Debugger) StartTrace(traceID, spanID, parentID, operation string) *TraceSpan {
	if !d.enableTracing {
		return nil
	}
	return d.tracer.StartSpan(traceID, spanID, parentID, operation)
}

// EndTrace completes a trace.
func (d *Debugger) EndTrace(spanID string, success bool, err error) {
	if !d.enableTracing {
		return
	}
	d.tracer.EndSpan(spanID, success, err)
}

// GetDebugInfo returns comprehensive debug information.
func (d *Debugger) GetDebugInfo() *DebugInfo {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	profile := d.monitor.CaptureProfile()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	errorRate := float64(0)
	if profile.RequestCount > 0 {
		errorRate = float64(profile.ErrorCount) / float64(profile.RequestCount) * 100
	}

	return &DebugInfo{
		Timestamp:       time.Now(),
		Uptime:          time.Since(d.monitor.startTime),
		Goroutines:      runtime.NumGoroutine(),
		MemoryAlloc:     m.Alloc,
		MemorySys:       m.Sys,
		HeapAlloc:       m.HeapAlloc,
		HeapObjects:     m.HeapObjects,
		GCRuns:          m.NumGC,
		RequestCount:    profile.RequestCount,
		ErrorCount:      profile.ErrorCount,
		ErrorRate:       errorRate,
		TopSpans:        d.getTopSlowSpans(),
		RecentSnapshots: d.monitor.GetProfileSnapshots(5),
	}
}

func (d *Debugger) getTopSlowSpans() []*TraceSpan {
	// Return up to 10 slowest spans
	spans := d.tracer.spans
	if len(spans) <= 10 {
		result := make([]*TraceSpan, 0, len(spans))
		for _, s := range spans {
			result = append(result, s)
		}
		return result
	}

	// Simplified top-10 (not actually sorted, just returns first 10)
	result := make([]*TraceSpan, 0, 10)
	count := 0
	for _, s := range spans {
		if count >= 10 {
			break
		}
		result = append(result, s)
		count++
	}
	return result
}

// HealthCheck performs a runtime health check.
type HealthCheck struct {
	Status           string
	Timestamp        time.Time
	MemoryOK         bool
	GoroutineOK      bool
	ErrorRateOK      bool
	MemoryUsagePercent float64
	GoroutineCount   int
	ErrorRate        float64
	Warnings         []string
}

// HealthChecker performs health checks.
type HealthChecker struct {
	debugger         *Debugger
	memoryThreshold  uint64 // in bytes
	goroutineLimit   int
	errorRateLimit   float64 // as percentage
	mutex            sync.RWMutex
}

// NewHealthChecker creates a health checker.
func NewHealthChecker(debugger *Debugger) *HealthChecker {
	return &HealthChecker{
		debugger:       debugger,
		memoryThreshold: 1024 * 1024 * 1024, // 1GB
		goroutineLimit: 10000,
		errorRateLimit: 5.0, // 5%
	}
}

// SetThresholds configures health check thresholds.
func (hc *HealthChecker) SetThresholds(memoryBytes uint64, goroutineLimit int, errorRatePercent float64) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	hc.memoryThreshold = memoryBytes
	hc.goroutineLimit = goroutineLimit
	hc.errorRateLimit = errorRatePercent
}

// Check performs a health check.
func (hc *HealthChecker) Check() *HealthCheck {
	info := hc.debugger.GetDebugInfo()

	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	check := &HealthCheck{
		Status:           "healthy",
		Timestamp:        time.Now(),
		MemoryOK:         true,
		GoroutineOK:      true,
		ErrorRateOK:      true,
		MemoryUsagePercent: float64(info.MemoryAlloc) / float64(hc.memoryThreshold) * 100,
		GoroutineCount:   info.Goroutines,
		ErrorRate:        info.ErrorRate,
		Warnings:         []string{},
	}

	if info.MemoryAlloc > hc.memoryThreshold {
		check.MemoryOK = false
		check.Status = "degraded"
		check.Warnings = append(check.Warnings, fmt.Sprintf("Memory usage %.1f%% of limit", check.MemoryUsagePercent))
	}

	if info.Goroutines > hc.goroutineLimit {
		check.GoroutineOK = false
		check.Status = "degraded"
		check.Warnings = append(check.Warnings, fmt.Sprintf("Goroutine count %d exceeds limit %d", info.Goroutines, hc.goroutineLimit))
	}

	if info.ErrorRate > hc.errorRateLimit {
		check.ErrorRateOK = false
		check.Status = "degraded"
		check.Warnings = append(check.Warnings, fmt.Sprintf("Error rate %.2f%% exceeds limit %.2f%%", info.ErrorRate, hc.errorRateLimit))
	}

	if !check.MemoryOK && !check.GoroutineOK && !check.ErrorRateOK {
		check.Status = "critical"
	}

	return check
}

// RequestTracer traces individual requests.
type RequestTracer struct {
	debugger *Debugger
	traceID  string
	startTime time.Time
	endTime   time.Time
	spans    []*TraceSpan
	mutex    sync.RWMutex
}

// NewRequestTracer creates a request tracer.
func NewRequestTracer(debugger *Debugger, traceID string) *RequestTracer {
	return &RequestTracer{
		debugger:  debugger,
		traceID:   traceID,
		startTime: time.Now(),
		spans:     make([]*TraceSpan, 0),
	}
}

// Start starts a named operation span.
func (rt *RequestTracer) Start(spanID, parentID, operation string) *TraceSpan {
	span := rt.debugger.StartTrace(rt.traceID, spanID, parentID, operation)
	rt.mutex.Lock()
	defer rt.mutex.Unlock()
	if span != nil {
		rt.spans = append(rt.spans, span)
	}
	return span
}

// End completes the trace.
func (rt *RequestTracer) End(spanID string, success bool, err error) {
	rt.debugger.EndTrace(spanID, success, err)
	rt.endTime = time.Now()
}

// GetTrace returns the complete trace.
func (rt *RequestTracer) GetTrace() map[string]interface{} {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()

	return map[string]interface{}{
		"trace_id":   rt.traceID,
		"start_time": rt.startTime,
		"end_time":   rt.endTime,
		"duration":   rt.endTime.Sub(rt.startTime),
		"span_count": len(rt.spans),
	}
}
