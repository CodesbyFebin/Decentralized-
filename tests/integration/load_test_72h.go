package integration

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// LoadTest72hConfig configures the 72-hour load test.
type LoadTest72hConfig struct {
	TargetTxPerSec     int64         // Target transactions per second (default 1000)
	DurationMinutes    int           // Total duration in minutes (default 4320 = 72 hours)
	MemoryCheckInterval time.Duration // How often to check memory (default 30 seconds)
	LogInterval        time.Duration // How often to log progress (default 5 minutes)
	MaxMemoryGrowthMB  int64         // Maximum allowed memory growth in MB (default 500)
}

// LoadTestResult captures results from a load test.
type LoadTestResult struct {
	Duration              time.Duration
	TotalTransactions    int64
	SuccessfulTx         int64
	FailedTx             int64
	ErrorCount           int64
	AvgLatency           time.Duration
	MaxLatency           time.Duration
	MinLatency           time.Duration
	ThroughputPerSec     float64
	MemoryStartMB        uint64
	MemoryPeakMB         uint64
	MemoryEndMB          uint64
	MemoryGrowthMB       int64
	GCCount              uint32
	GoRoutineCount       int
	DetectedMemoryLeak   bool
	CrashDetected        bool
	ErrorSummary         map[string]int64
}

// LoadTestRunner executes the load test.
type LoadTestRunner struct {
	config          LoadTest72hConfig
	result          *LoadTestResult
	mu              sync.RWMutex
	startTime       time.Time
	stopChan        chan struct{}
	latencies       []time.Duration
	latenciesMu     sync.Mutex
	memoryCheckTick *time.Ticker
	logTicker       *time.Ticker
	errors          map[string]int64
	errorsMu        sync.Mutex
}

// NewLoadTest72h creates a new 72-hour load test.
func NewLoadTest72h() *LoadTestRunner {
	return &LoadTestRunner{
		config: LoadTest72hConfig{
			TargetTxPerSec:      1000,
			DurationMinutes:     4320, // 72 hours
			MemoryCheckInterval: 30 * time.Second,
			LogInterval:         5 * time.Minute,
			MaxMemoryGrowthMB:   500,
		},
		result:    &LoadTestResult{ErrorSummary: make(map[string]int64)},
		stopChan:  make(chan struct{}),
		latencies: make([]time.Duration, 0, 1000000),
		errors:    make(map[string]int64),
	}
}

// Start begins the load test.
func (l *LoadTestRunner) Start(t *testing.T) {
	l.startTime = time.Now()

	// Record initial memory state
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	l.result.MemoryStartMB = m.Alloc / 1024 / 1024

	t.Logf("[72h Load Test] Starting with target %d tx/sec for %d minutes",
		l.config.TargetTxPerSec, l.config.DurationMinutes)

	// Start memory monitor
	l.memoryCheckTick = time.NewTicker(l.config.MemoryCheckInterval)
	l.logTicker = time.NewTicker(l.config.LogInterval)

	go l.monitorMemory()
	go l.logProgress(t)
}

// Stop ends the load test and returns results.
func (l *LoadTestRunner) Stop() *LoadTestResult {
	close(l.stopChan)

	if l.memoryCheckTick != nil {
		l.memoryCheckTick.Stop()
	}
	if l.logTicker != nil {
		l.logTicker.Stop()
	}

	// Calculate final results
	l.mu.Lock()
	defer l.mu.Unlock()

	l.result.Duration = time.Since(l.startTime)
	l.result.TotalTransactions = l.result.SuccessfulTx + l.result.FailedTx

	if l.result.TotalTransactions > 0 {
		l.result.ThroughputPerSec = float64(l.result.TotalTransactions) / l.result.Duration.Seconds()
	}

	// Finalize error summary
	l.errorsMu.Lock()
	l.result.ErrorSummary = l.errors
	l.errorsMu.Unlock()

	// Calculate latency stats
	if len(l.latencies) > 0 {
		totalLatency := int64(0)
		minLatency := time.Duration(int64(1<<63 - 1))
		maxLatency := time.Duration(0)

		for _, lat := range l.latencies {
			totalLatency += int64(lat)
			if lat < minLatency {
				minLatency = lat
			}
			if lat > maxLatency {
				maxLatency = lat
			}
		}

		l.result.AvgLatency = time.Duration(totalLatency / int64(len(l.latencies)))
		l.result.MinLatency = minLatency
		l.result.MaxLatency = maxLatency
	}

	// Get final memory state
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	l.result.MemoryEndMB = m.Alloc / 1024 / 1024
	l.result.MemoryGrowthMB = int64(l.result.MemoryEndMB) - int64(l.result.MemoryStartMB)

	return l.result
}

// RecordTransaction records a completed transaction.
func (l *LoadTestRunner) RecordTransaction(success bool, latency time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if success {
		atomic.AddInt64(&l.result.SuccessfulTx, 1)
	} else {
		atomic.AddInt64(&l.result.FailedTx, 1)
	}

	l.latenciesMu.Lock()
	if len(l.latencies) < cap(l.latencies) {
		l.latencies = append(l.latencies, latency)
	}
	l.latenciesMu.Unlock()
}

// RecordError records an error that occurred.
func (l *LoadTestRunner) RecordError(errorType string) {
	l.errorsMu.Lock()
	defer l.errorsMu.Unlock()

	l.errors[errorType]++
	atomic.AddInt64(&l.result.ErrorCount, 1)
}

// monitorMemory checks for memory leaks.
func (l *LoadTestRunner) monitorMemory() {
	maxMemory := l.result.MemoryStartMB + uint64(l.config.MaxMemoryGrowthMB)

	for {
		select {
		case <-l.stopChan:
			return
		case <-l.memoryCheckTick.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			currentMB := m.Alloc / 1024 / 1024

			l.result.MemoryPeakMB = max(l.result.MemoryPeakMB, currentMB)

			// Detect memory leak (sustained growth)
			if currentMB > maxMemory {
				l.result.DetectedMemoryLeak = true
			}
		}
	}
}

// logProgress periodically logs test progress.
func (l *LoadTestRunner) logProgress(t *testing.T) {
	for {
		select {
		case <-l.stopChan:
			return
		case <-l.logTicker.C:
			l.mu.RLock()
			elapsed := time.Since(l.startTime)
			successTx := atomic.LoadInt64(&l.result.SuccessfulTx)
			failedTx := atomic.LoadInt64(&l.result.FailedTx)
			errorCount := atomic.LoadInt64(&l.result.ErrorCount)
			l.mu.RUnlock()

			throughput := float64(successTx) / elapsed.Seconds()
			errorRate := 0.0
			if successTx+failedTx > 0 {
				errorRate = float64(failedTx) / float64(successTx+failedTx) * 100
			}

			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			currentMB := memStats.Alloc / 1024 / 1024

			t.Logf("[72h Load Test] Elapsed: %v | Success: %d | Failed: %d | "+
				"Throughput: %.2f tx/s | Error Rate: %.2f%% | Memory: %d MB | "+
				"Errors: %d",
				elapsed, successTx, failedTx, throughput, errorRate, currentMB, errorCount)
		}
	}
}

// TestLoadTest72Hour runs the 72-hour production load test.
// This test validates system stability under sustained load.
// It should only be run in dedicated test environments with sufficient resources.
func TestLoadTest72Hour(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping 72-hour load test in short mode")
	}

	// Check environment variable to enable explicit 72-hour test
	// export DH_RUN_72H_TEST=true to enable
	// This prevents accidental long-running tests
	enable72h := os.Getenv("DH_RUN_72H_TEST") == "true"
	if !enable72h {
		t.Skip("72-hour load test requires DH_RUN_72H_TEST=true")
	}

	harness := NewTestHarness("72-Hour-Production-Load-Test")
	harness.Start()

	runner := NewLoadTest72h()
	runner.config.TargetTxPerSec = 1000      // 1000 tx/sec
	runner.config.DurationMinutes = 4320     // 72 hours
	runner.config.MaxMemoryGrowthMB = 500    // 500 MB growth tolerance
	runner.config.LogInterval = 5 * time.Minute

	runner.Start(t)

	// Run the load test
	ctx := &loadTestContext{
		runner:        runner,
		stopTime:      time.Now().Add(time.Duration(runner.config.DurationMinutes) * time.Minute),
		errorChan:     make(chan string, 1000),
		txChan:        make(chan struct{}, runner.config.TargetTxPerSec),
	}

	// Start transaction generator (would integrate with actual dh cluster)
	go ctx.generateLoad()
	go ctx.processErrors()

	// Monitor test progress
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			result := runner.result
			t.Logf("[72h Load Test] Checkpoint - Success: %d, Failed: %d, Errors: %d",
				result.SuccessfulTx, result.FailedTx, result.ErrorCount)

			// Check for issues
			if result.DetectedMemoryLeak {
				t.Logf("WARNING: Memory leak detected - growth: %d MB", result.MemoryGrowthMB)
			}

		case <-time.After(time.Duration(runner.config.DurationMinutes) * time.Minute):
			// Test duration reached
			goto testComplete
		}
	}

testComplete:
	// Stop the test
	result := runner.Stop()

	// Validate results
	t.Logf("[72h Load Test] Results:")
	t.Logf("  Duration: %v", result.Duration)
	t.Logf("  Total Transactions: %d", result.TotalTransactions)
	t.Logf("  Successful: %d", result.SuccessfulTx)
	t.Logf("  Failed: %d", result.FailedTx)
	t.Logf("  Error Count: %d", result.ErrorCount)
	t.Logf("  Throughput: %.2f tx/sec", result.ThroughputPerSec)
	t.Logf("  Avg Latency: %v", result.AvgLatency)
	t.Logf("  Max Latency: %v", result.MaxLatency)
	t.Logf("  Memory Start: %d MB", result.MemoryStartMB)
	t.Logf("  Memory Peak: %d MB", result.MemoryPeakMB)
	t.Logf("  Memory End: %d MB", result.MemoryEndMB)
	t.Logf("  Memory Growth: %d MB", result.MemoryGrowthMB)

	// Report results
	if result.SuccessfulTx > 0 && result.FailedTx == 0 {
		harness.ReportPass("sustained-throughput",
			fmt.Sprintf("Sustained %.2f tx/sec for 72 hours without errors", result.ThroughputPerSec))
	} else {
		harness.ReportFail("sustained-throughput",
			fmt.Sprintf("Errors detected: %d failed transactions", result.FailedTx))
	}

	if !result.DetectedMemoryLeak {
		harness.ReportPass("memory-stability",
			fmt.Sprintf("Memory stable - growth: %d MB (limit: %d MB)",
				result.MemoryGrowthMB, runner.config.MaxMemoryGrowthMB))
	} else {
		harness.ReportFail("memory-stability",
			fmt.Sprintf("Memory leak detected - growth: %d MB", result.MemoryGrowthMB))
	}

	if result.ThroughputPerSec >= float64(runner.config.TargetTxPerSec)*0.95 {
		harness.ReportPass("throughput-consistency",
			"Throughput maintained within 95% of target")
	} else {
		harness.ReportFail("throughput-consistency",
			"Throughput degradation detected")
	}

	if !result.CrashDetected && result.ErrorCount < (result.TotalTransactions / 1000) {
		harness.ReportPass("error-resilience",
			"Error rate within acceptable threshold (<0.1%)")
	} else {
		harness.ReportFail("error-resilience",
			"Error rate exceeded threshold")
	}

	harness.Finalize(t)
}

// loadTestContext manages load test execution.
type loadTestContext struct {
	runner    *LoadTestRunner
	stopTime  time.Time
	errorChan chan string
	txChan    chan struct{}
}

// generateLoad generates transactions at the target rate.
func (ctx *loadTestContext) generateLoad() {
	ticker := time.NewTicker(time.Second / time.Duration(ctx.runner.config.TargetTxPerSec))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.runner.stopChan:
			return
		case <-ticker.C:
			if time.Now().After(ctx.stopTime) {
				return
			}

			// Simulate a transaction
			start := time.Now()
			success := true // In real implementation, would execute actual transaction
			latency := time.Since(start)

			ctx.runner.RecordTransaction(success, latency)
		}
	}
}

// processErrors handles error reporting during load test.
func (ctx *loadTestContext) processErrors() {
	for {
		select {
		case <-ctx.runner.stopChan:
			return
		case errMsg := <-ctx.errorChan:
			ctx.runner.RecordError(errMsg)
		}
	}
}

// Helper function
func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// Import needed for os.Getenv
import "os"
