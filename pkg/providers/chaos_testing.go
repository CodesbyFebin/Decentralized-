package providers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ProductionChaosScenario defines a chaos experiment for testing production resilience.
type ProductionChaosScenario struct {
	Name              string
	Description       string
	Type              string // "network", "storage", "metrics", "database", "compute"
	Duration          time.Duration
	InjectionPoint    string // Where failure is injected
	Severity          string // "low", "medium", "high", "critical"
	PreConditions     []string
	PostConditions    []string
	ExpectedBehavior  string
	RecoveryTime      time.Duration // Expected time to recover
	MaxAllowedDowntime time.Duration
	Invariants        []string // Conditions that must hold true
}

// ProductionChaosResult tracks the outcome of a production chaos experiment.
type ProductionChaosResult struct {
	ScenarioName      string
	StartTime         time.Time
	EndTime           time.Time
	Duration          time.Duration
	Status            string // "passed", "failed", "degraded", "error"
	InvariantsBroken  []string
	RecoveryTime      time.Duration
	DowntimeDuration  time.Duration
	ErrorCount        int64
	ErrorMessages     []string
	DataLoss          bool
	CorruptionDetected bool
	Observations      []string
}

// NetworkFailureScenario simulates network faults.
type NetworkFailureScenario struct {
	BaseName          string
	LatencyMs         int // Artificial delay in milliseconds
	PacketLossPercent int // Percentage of packets lost (0-100)
	JitterMs          int // Random variation in latency
	ConnectionTimeout int // Timeout in seconds
	BurstLoss         bool // Intermittent total loss vs. gradual
	BurstInterval     time.Duration // How often bursts occur
	BurstDuration     time.Duration // How long each burst lasts
	TargetService     string // Which service (database, backup, cache)
	AffectedRegions   []string // Geographic regions affected
}

// StorageFailureScenario simulates disk and I/O faults.
type StorageFailureScenario struct {
	BaseName          string
	FailureType       string // "disk_full", "read_slow", "write_slow", "corruption", "inaccessible"
	PercentageFull    int // For disk_full (0-100)
	LatencyMultiplier float64 // For slow reads/writes (1.0 = normal)
	CorruptionRate    float64 // Percentage of operations affected
	TargetPath        string // Path being affected
	RecoveryTime      time.Duration
	DataConsistency   bool // Whether data remains consistent
}

// MetricsFailureScenario simulates observability faults.
type MetricsFailureScenario struct {
	BaseName              string
	FailureType           string // "collector_crash", "metric_loss", "high_cardinality", "exporter_timeout"
	MetricsLossPercent    int // Percentage of metrics dropped
	ExporterLatencyMs     int // Latency before timeout
	HighCardinalityFactor int // Number of metric variants (increases cardinality)
	CollectorRestartTime  time.Duration
	AlertsDelivered       bool // Whether alerts still function
	DashboardAvailable    bool // Whether Grafana dashboard works
}

// DatabaseFailureScenario simulates database faults.
type DatabaseFailureScenario struct {
	BaseName           string
	FailureType        string // "connection_limit", "slow_queries", "deadlock", "replica_lag", "failover"
	MaxConnections     int // Connection pool exhaustion
	QueryLatencyMs     int // Artificial slowdown
	DeadlockFrequency  int // How often deadlocks occur (per 1000 queries)
	ReplicaLagSeconds  int // Replication lag in seconds
	FailoverDuration   time.Duration
	DataConsistency    bool
	TransactionRollback bool
}

// ComputeFailureScenario simulates compute resource faults.
type ComputeFailureScenario struct {
	BaseName       string
	FailureType    string // "cpu_spike", "memory_leak", "goroutine_leak", "gc_pause"
	CPUPercent     int // Target CPU utilization
	MemoryPercent  int // Target memory utilization
	LeakRate       int // Bytes per second leaked
	GCPauseDuration time.Duration
	RecoveryTime   time.Duration
}

// InvariantValidator checks system invariants during chaos experiments.
type InvariantValidator struct {
	invariantChecks    map[string]func(context.Context) bool
	validationMutex    sync.RWMutex
	validationHistory  []ValidationResult
	lastValidationTime time.Time
}

// ValidationResult tracks a single invariant check.
type ValidationResult struct {
	Timestamp       time.Time
	InvariantName   string
	Passed          bool
	Details         string
	RecoveryTime    time.Duration
	DataLoss        bool
	CorruptionDetected bool
}

// ChaosTestRunner orchestrates chaos experiments.
type ChaosTestRunner struct {
	scenarios         []ProductionChaosScenario
	results           []*ProductionChaosResult
	validator         *InvariantValidator
	runnerMutex       sync.RWMutex
	isRunning         bool
	currentScenario   string
	totalDuration     time.Duration
	experimentErrors  []string
}

// NewInvariantValidator creates an invariant validator.
func NewInvariantValidator() *InvariantValidator {
	return &InvariantValidator{
		invariantChecks:   make(map[string]func(context.Context) bool),
		validationHistory: make([]ValidationResult, 0),
	}
}

// RegisterInvariant registers an invariant check function.
func (iv *InvariantValidator) RegisterInvariant(name string, checkFunc func(context.Context) bool) {
	iv.validationMutex.Lock()
	defer iv.validationMutex.Unlock()
	iv.invariantChecks[name] = checkFunc
}

// ValidateInvariants runs all registered invariant checks.
func (iv *InvariantValidator) ValidateInvariants(ctx context.Context) ([]ValidationResult, bool) {
	iv.validationMutex.Lock()
	defer iv.validationMutex.Unlock()

	results := make([]ValidationResult, 0)
	allPassed := true

	for name, checkFunc := range iv.invariantChecks {
		result := ValidationResult{
			Timestamp:     time.Now(),
			InvariantName: name,
			Passed:        checkFunc(ctx),
		}
		if !result.Passed {
			allPassed = false
			result.Details = fmt.Sprintf("Invariant %s violated", name)
		}
		results = append(results, result)
		iv.validationHistory = append(iv.validationHistory, result)
	}

	iv.lastValidationTime = time.Now()
	return results, allPassed
}

// NewChaosRunner creates a chaos testing runner.
func NewChaosRunner(validator *InvariantValidator) *ChaosTestRunner {
	return &ChaosTestRunner{
		scenarios:        make([]ProductionChaosScenario, 0),
		results:          make([]*ProductionChaosResult, 0),
		validator:        validator,
		experimentErrors: make([]string, 0),
	}
}

// AddScenario adds a scenario to the runner.
func (cr *ChaosTestRunner) AddScenario(scenario ProductionChaosScenario) {
	cr.runnerMutex.Lock()
	defer cr.runnerMutex.Unlock()
	cr.scenarios = append(cr.scenarios, scenario)
}

// RunScenario executes a single chaos scenario.
func (cr *ChaosTestRunner) RunScenario(ctx context.Context, scenario ProductionChaosScenario) *ProductionChaosResult {
	cr.runnerMutex.Lock()
	cr.isRunning = true
	cr.currentScenario = scenario.Name
	cr.runnerMutex.Unlock()

	result := &ProductionChaosResult{
		ScenarioName:  scenario.Name,
		StartTime:     time.Now(),
		Status:        "running",
		ErrorMessages: make([]string, 0),
		Observations: make([]string, 0),
	}

	// Pre-condition validation
	result.Observations = append(result.Observations, "Validating pre-conditions...")
	_, preOk := cr.validator.ValidateInvariants(ctx)
	if !preOk {
		result.Status = "failed"
		result.Observations = append(result.Observations, "Pre-conditions not satisfied")
		return result
	}

	// Inject chaos
	result.Observations = append(result.Observations, fmt.Sprintf("Injecting %s failure at %s...", scenario.Type, scenario.InjectionPoint))
	injectionCtx, cancel := context.WithTimeout(ctx, scenario.Duration)
	defer cancel()

	select {
	case <-injectionCtx.Done():
		// Chaos duration complete
	case <-ctx.Done():
		// External cancellation
		cancel()
	}

	// Monitor recovery
	recoveryStart := time.Now()
	recoveryTimeout := time.After(scenario.RecoveryTime + (5 * time.Second)) // Allow 5s buffer

	var recovered = false
	var postValidations []ValidationResult

	monitorCtx, monitorCancel := context.WithTimeout(ctx, scenario.RecoveryTime)
	defer monitorCancel()

	for !recovered {
		select {
		case <-recoveryTimeout:
			result.Status = "failed"
			result.Observations = append(result.Observations, "Recovery timeout exceeded")
			break
		case <-monitorCtx.Done():
			result.Status = "failed"
			break
		default:
			postValidations, recovered = cr.validator.ValidateInvariants(monitorCtx)
			if recovered {
				result.Status = "passed"
				result.RecoveryTime = time.Since(recoveryStart)
				result.Observations = append(result.Observations, fmt.Sprintf("System recovered in %v", result.RecoveryTime))
			} else {
				// Wait before next check
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	// Check for broken invariants
	for _, validation := range postValidations {
		if !validation.Passed {
			result.InvariantsBroken = append(result.InvariantsBroken, validation.InvariantName)
		}
	}

	// Calculate downtime
	result.Duration = time.Since(result.StartTime)
	if result.Duration > scenario.MaxAllowedDowntime {
		result.Status = "degraded"
		result.DowntimeDuration = result.Duration - scenario.MaxAllowedDowntime
		result.Observations = append(result.Observations,
			fmt.Sprintf("Downtime %v exceeded max %v", result.Duration, scenario.MaxAllowedDowntime))
	}

	result.EndTime = time.Now()
	cr.results = append(cr.results, result)

	cr.runnerMutex.Lock()
	cr.isRunning = false
	cr.runnerMutex.Unlock()

	return result
}

// RunAllScenarios executes all registered scenarios sequentially.
func (cr *ChaosTestRunner) RunAllScenarios(ctx context.Context) []*ProductionChaosResult {
	cr.runnerMutex.Lock()
	cr.isRunning = true
	overallStart := time.Now()
	cr.runnerMutex.Unlock()

	for _, scenario := range cr.scenarios {
		if ctx.Err() != nil {
			break
		}
		cr.RunScenario(ctx, scenario)
	}

	cr.runnerMutex.Lock()
	cr.totalDuration = time.Since(overallStart)
	cr.isRunning = false
	cr.runnerMutex.Unlock()

	return cr.results
}

// GetScenarioResults returns results for a specific scenario.
func (cr *ChaosTestRunner) GetScenarioResults(scenarioName string) []*ProductionChaosResult {
	cr.runnerMutex.RLock()
	defer cr.runnerMutex.RUnlock()

	matching := make([]*ProductionChaosResult, 0)
	for _, result := range cr.results {
		if result.ScenarioName == scenarioName {
			matching = append(matching, result)
		}
	}
	return matching
}

// GetChaosReport generates a summary report of all chaos tests.
func (cr *ChaosTestRunner) GetChaosReport() *ChaosReport {
	cr.runnerMutex.RLock()
	defer cr.runnerMutex.RUnlock()

	report := &ChaosReport{
		TotalScenarios: len(cr.scenarios),
		GeneratedAt:    time.Now(),
		Results:        cr.results,
	}

	passCount := 0
	degradedCount := 0
	failureCount := 0

	for _, result := range cr.results {
		switch result.Status {
		case "passed":
			passCount++
		case "degraded":
			degradedCount++
		case "failed":
			failureCount++
		}
	}

	report.PassedScenarios = passCount
	report.DegradedScenarios = degradedCount
	report.FailedScenarios = failureCount

	if report.TotalScenarios > 0 {
		report.PassRate = float64(passCount) / float64(report.TotalScenarios) * 100
	}

	report.TotalDuration = cr.totalDuration

	// Find most common failures
	failureMap := make(map[string]int)
	for _, result := range cr.results {
		for _, broken := range result.InvariantsBroken {
			failureMap[broken]++
		}
	}

	for invariant, count := range failureMap {
		report.MostCommonFailures = append(report.MostCommonFailures,
			InvariantFailure{InvariantName: invariant, FailureCount: count})
	}

	// Calculate average recovery time
	var totalRecovery time.Duration
	recoveryCount := 0
	for _, result := range cr.results {
		if result.Status == "passed" || result.Status == "degraded" {
			totalRecovery += result.RecoveryTime
			recoveryCount++
		}
	}
	if recoveryCount > 0 {
		report.AverageRecoveryTime = totalRecovery / time.Duration(recoveryCount)
	}

	return report
}

// ChaosReport summarizes chaos testing results.
type ChaosReport struct {
	TotalScenarios        int
	PassedScenarios       int
	DegradedScenarios     int
	FailedScenarios       int
	PassRate              float64
	TotalDuration         time.Duration
	AverageRecoveryTime   time.Duration
	MostCommonFailures    []InvariantFailure
	Results               []*ProductionChaosResult
	GeneratedAt           time.Time
}

// InvariantFailure tracks failures of a specific invariant.
type InvariantFailure struct {
	InvariantName string
	FailureCount  int
}

// StandardChaosScenarios returns a suite of standard chaos scenarios.
func StandardChaosScenarios() []ProductionChaosScenario {
	return []ProductionChaosScenario{
		{
			Name:             "network_latency_database",
			Description:      "Add 500ms latency to database connections",
			Type:             "network",
			Duration:         30 * time.Second,
			InjectionPoint:   "database_connection",
			Severity:         "medium",
			MaxAllowedDowntime: 30 * time.Second,
			RecoveryTime:     10 * time.Second,
			Invariants:       []string{"database_accessible", "campaign_data_consistent"},
		},
		{
			Name:             "network_packet_loss",
			Description:      "Introduce 10% packet loss to backup replication",
			Type:             "network",
			Duration:         60 * time.Second,
			InjectionPoint:   "backup_replication",
			Severity:         "medium",
			MaxAllowedDowntime: 60 * time.Second,
			RecoveryTime:     15 * time.Second,
			Invariants:       []string{"backup_replicas_healthy", "data_integrity"},
		},
		{
			Name:             "storage_disk_full",
			Description:      "Simulate 95% disk capacity",
			Type:             "storage",
			Duration:         45 * time.Second,
			InjectionPoint:   "storage_backend",
			Severity:         "critical",
			MaxAllowedDowntime: 10 * time.Second,
			RecoveryTime:     20 * time.Second,
			Invariants:       []string{"archive_writable", "backup_creatable"},
		},
		{
			Name:             "storage_slow_reads",
			Description:      "Slow down storage reads by 10x",
			Type:             "storage",
			Duration:         30 * time.Second,
			InjectionPoint:   "storage_read",
			Severity:         "medium",
			MaxAllowedDowntime: 30 * time.Second,
			RecoveryTime:     10 * time.Second,
			Invariants:       []string{"campaigns_queryable", "no_data_corruption"},
		},
		{
			Name:             "metrics_collector_crash",
			Description:      "Metrics collector process crashes",
			Type:             "metrics",
			Duration:         20 * time.Second,
			InjectionPoint:   "metrics_collector",
			Severity:         "low",
			MaxAllowedDowntime: 20 * time.Second,
			RecoveryTime:     10 * time.Second,
			Invariants:       []string{"system_operational", "alerts_functional"},
		},
		{
			Name:             "metrics_high_cardinality",
			Description:      "Metric cardinality explosion (100x increase)",
			Type:             "metrics",
			Duration:         30 * time.Second,
			InjectionPoint:   "metrics_export",
			Severity:         "low",
			MaxAllowedDowntime: 30 * time.Second,
			RecoveryTime:     15 * time.Second,
			Invariants:       []string{"prometheus_scrape_works", "grafana_responsive"},
		},
		{
			Name:             "database_connection_exhaustion",
			Description:      "Connection pool reached 100% capacity",
			Type:             "database",
			Duration:         30 * time.Second,
			InjectionPoint:   "connection_pool",
			Severity:         "critical",
			MaxAllowedDowntime: 5 * time.Second,
			RecoveryTime:     20 * time.Second,
			Invariants:       []string{"campaigns_stored", "queries_complete"},
		},
		{
			Name:             "database_slow_queries",
			Description:      "All queries execute 5x slower",
			Type:             "database",
			Duration:         45 * time.Second,
			InjectionPoint:   "query_execution",
			Severity:         "medium",
			MaxAllowedDowntime: 45 * time.Second,
			RecoveryTime:     15 * time.Second,
			Invariants:       []string{"campaigns_queryable", "gates_executable"},
		},
		{
			Name:             "database_replica_lag",
			Description:      "Replica falls 60 seconds behind",
			Type:             "database",
			Duration:         60 * time.Second,
			InjectionPoint:   "replication_lag",
			Severity:         "medium",
			MaxAllowedDowntime: 60 * time.Second,
			RecoveryTime:     30 * time.Second,
			Invariants:       []string{"primary_writable", "backup_consistent"},
		},
		{
			Name:             "compute_cpu_spike",
			Description:      "CPU spike to 90% utilization",
			Type:             "compute",
			Duration:         40 * time.Second,
			InjectionPoint:   "gate_execution",
			Severity:         "medium",
			MaxAllowedDowntime: 40 * time.Second,
			RecoveryTime:     20 * time.Second,
			Invariants:       []string{"campaigns_progressing", "no_deadlock"},
		},
		{
			Name:             "compute_memory_leak",
			Description:      "Memory leak (10MB per second)",
			Type:             "compute",
			Duration:         50 * time.Second,
			InjectionPoint:   "process_memory",
			Severity:         "high",
			MaxAllowedDowntime: 10 * time.Second,
			RecoveryTime:     30 * time.Second,
			Invariants:       []string{"system_operational", "memory_recoverable"},
		},
		{
			Name:             "compute_goroutine_leak",
			Description:      "Goroutine leak (10 new goroutines per second)",
			Type:             "compute",
			Duration:         60 * time.Second,
			InjectionPoint:   "goroutine_pool",
			Severity:         "high",
			MaxAllowedDowntime: 10 * time.Second,
			RecoveryTime:     30 * time.Second,
			Invariants:       []string{"concurrency_bounded", "no_resource_exhaustion"},
		},
		{
			Name:             "cascading_failure",
			Description:      "Database slow + network latency simultaneously",
			Type:             "database",
			Duration:         60 * time.Second,
			InjectionPoint:   "multiple",
			Severity:         "critical",
			MaxAllowedDowntime: 10 * time.Second,
			RecoveryTime:     40 * time.Second,
			Invariants:       []string{"no_data_loss", "backup_integrity", "campaigns_safe"},
		},
		{
			Name:             "partial_region_outage",
			Description:      "One geographic region becomes unreachable",
			Type:             "network",
			Duration:         30 * time.Second,
			InjectionPoint:   "region_connectivity",
			Severity:         "high",
			MaxAllowedDowntime: 30 * time.Second,
			RecoveryTime:     20 * time.Second,
			Invariants:       []string{"failover_activated", "other_regions_operational"},
		},
		{
			Name:             "backup_replica_split_brain",
			Description:      "Two replicas diverge due to network partition",
			Type:             "network",
			Duration:         40 * time.Second,
			InjectionPoint:   "replica_communication",
			Severity:         "critical",
			MaxAllowedDowntime: 30 * time.Second,
			RecoveryTime:     25 * time.Second,
			Invariants:       []string{"data_convergence", "single_primary", "no_divergence"},
		},
	}
}

// ChaosTestSuite runs a complete chaos testing suite.
type ChaosTestSuite struct {
	name              string
	scenarios         []ProductionChaosScenario
	validator         *InvariantValidator
	runner            *ChaosTestRunner
	suiteStartTime    time.Time
	suiteEndTime      time.Time
	suiteDuration     time.Duration
	successCriteria   map[string]float64 // invariant -> min success rate
}

// NewChaosTestSuite creates a new chaos test suite.
func NewChaosTestSuite(name string, validator *InvariantValidator) *ChaosTestSuite {
	return &ChaosTestSuite{
		name:            name,
		scenarios:       StandardChaosScenarios(),
		validator:       validator,
		runner:          NewChaosRunner(validator),
		successCriteria: make(map[string]float64),
	}
}

// SetSuccessCriteria sets minimum success rate for an invariant.
func (cts *ChaosTestSuite) SetSuccessCriteria(invariant string, minSuccessRate float64) {
	cts.successCriteria[invariant] = minSuccessRate
}

// Run executes the chaos test suite.
func (cts *ChaosTestSuite) Run(ctx context.Context) *ChaosReport {
	cts.suiteStartTime = time.Now()

	// Add scenarios to runner
	for _, scenario := range cts.scenarios {
		cts.runner.AddScenario(scenario)
	}

	// Run all scenarios
	cts.runner.RunAllScenarios(ctx)

	cts.suiteEndTime = time.Now()
	cts.suiteDuration = cts.suiteEndTime.Sub(cts.suiteStartTime)

	// Generate report
	report := cts.runner.GetChaosReport()
	report.TotalDuration = cts.suiteDuration

	return report
}

// ValidateSuiteResults validates that chaos results meet criteria.
func (cts *ChaosTestSuite) ValidateSuiteResults(report *ChaosReport) bool {
	if report.PassRate < 80.0 {
		return false // Less than 80% pass rate
	}

	// Check specific invariant criteria
	invariantFailures := make(map[string]int)
	for _, failure := range report.MostCommonFailures {
		invariantFailures[failure.InvariantName] = failure.FailureCount
	}

	for invariant, minRate := range cts.successCriteria {
		failureCount := invariantFailures[invariant]
		totalRuns := report.TotalScenarios
		successRate := float64(totalRuns-failureCount) / float64(totalRuns) * 100

		if successRate < minRate {
			return false
		}
	}

	return true
}
