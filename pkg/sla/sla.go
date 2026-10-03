// Package sla implements SLA monitoring and enforcement for Decentralized.Host.
//
// This package provides:
//   - SLA definition and tracking
//   - SLA violation detection and reporting
//   - SLA compliance dashboards and reporting
//   - SLA-based auto-remediation triggers
//   - Operator SLA compliance tracking
//   - Monthly and quarterly SLA windows
//
// SLAs are defined per service component with targets for uptime,
// response time, and error rates. Violations are tracked and can
// trigger automatic remediation actions.
package sla

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SLALevel defines the SLA target level.
type SLALevel string

const (
	// SLALevel99p9 targets 99.9% uptime (8.76 hours downtime/year)
	SLALevel99p9 SLALevel = "99.9"

	// SLALevel99p99 targets 99.99% uptime (52.56 minutes downtime/year)
	SLALevel99p99 SLALevel = "99.99"

	// SLALevel99p999 targets 99.999% uptime (5.26 minutes downtime/year)
	SLALevel99p999 SLALevel = "99.999"
)

// SLADefinition defines an SLA for a service component.
type SLADefinition struct {
	// ID uniquely identifies this SLA
	ID string

	// ServiceName is the service this SLA applies to
	ServiceName string

	// Level is the target uptime level
	Level SLALevel

	// UptimeTarget is the target uptime percentage (99.9, 99.99, 99.999)
	UptimeTarget float64

	// ResponseTimeP99 is the target P99 response time
	ResponseTimeP99 time.Duration

	// ErrorRateTarget is the target error rate (0.0-1.0)
	ErrorRateTarget float64

	// Window is the measurement window (MONTHLY, QUARTERLY, YEARLY)
	Window string

	// RemediationAction is called when SLA is violated
	RemediationAction RemediationFunc

	// Enabled controls whether this SLA is active
	Enabled bool

	// CreatedAt is when this SLA was created
	CreatedAt time.Time
}

// RemediationFunc is called when an SLA is violated.
type RemediationFunc func(context.Context, *SLAViolation) error

// SLAWindow represents a measurement window.
type SLAWindow struct {
	// ID uniquely identifies this window
	ID string

	// StartTime is when the window began
	StartTime time.Time

	// EndTime is when the window ends
	EndTime time.Time

	// SLADefinitionID is the SLA this window measures
	SLADefinitionID string

	// TotalTime is the total time in this window
	TotalTime time.Duration

	// UptimeTime is the time the service was up
	UptimeTime time.Duration

	// ActualUptime is the calculated uptime percentage
	ActualUptime float64

	// Violations is the number of SLA violations in this window
	Violations int

	// Closed indicates whether this window is closed
	Closed bool
}

// SLAViolation represents an SLA violation event.
type SLAViolation struct {
	// ID uniquely identifies this violation
	ID string

	// SLADefinitionID is the SLA that was violated
	SLADefinitionID string

	// ViolationType indicates what was violated (UPTIME, LATENCY, ERROR_RATE)
	ViolationType string

	// Details contains violation-specific details
	Details map[string]interface{}

	// OccurredAt is when the violation was detected
	OccurredAt time.Time

	// ResolvedAt is when the violation was remedied (if remedied)
	ResolvedAt *time.Time

	// RemediationAttempts is how many remediation attempts were made
	RemediationAttempts int

	// LastRemediationError is the last error from remediation
	LastRemediationError error

	// WindowID is the SLA window this violation belongs to
	WindowID string
}

// Manager manages SLAs and their monitoring.
type Manager struct {
	mu sync.RWMutex

	// definitions maps SLA ID to definition
	definitions map[string]*SLADefinition

	// windows maps window ID to window
	windows map[string]*SLAWindow

	// violations maps violation ID to violation
	violations map[string]*SLAViolation

	// currentWindows maps SLA ID to current window
	currentWindows map[string]*SLAWindow

	// windowTicker triggers periodic window checks
	windowTicker *time.Ticker

	// config stores configuration
	config Config

	// done signals shutdown
	done chan struct{}

	// statistics
	violationCount uint64
	remediationAttempts uint64
}

// Config defines SLA manager configuration.
type Config struct {
	// WindowDuration is the default SLA window duration
	WindowDuration time.Duration

	// CheckInterval is how often to check SLA compliance
	CheckInterval time.Duration

	// MaxRemediationRetries is the maximum number of remediation retries
	MaxRemediationRetries int

	// HistorySize is the maximum number of historical violations to keep
	HistorySize int
}

// DefaultConfig returns a default SLA manager configuration.
func DefaultConfig() Config {
	return Config{
		WindowDuration:        24 * time.Hour, // Daily windows for testing; use 30*24*time.Hour for monthly
		CheckInterval:         time.Minute,
		MaxRemediationRetries: 3,
		HistorySize:           10000,
	}
}

// NewManager creates a new SLA manager.
func NewManager(cfg Config) *Manager {
	return &Manager{
		definitions:    make(map[string]*SLADefinition),
		windows:        make(map[string]*SLAWindow),
		violations:     make(map[string]*SLAViolation),
		currentWindows: make(map[string]*SLAWindow),
		config:         cfg,
		done:           make(chan struct{}),
	}
}

// Start starts the SLA manager.
func (m *Manager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.windowTicker = time.NewTicker(m.config.CheckInterval)

	go func() {
		for {
			select {
			case <-m.windowTicker.C:
				m.checkCompliance(context.Background())
			case <-m.done:
				return
			}
		}
	}()
}

// Stop stops the SLA manager.
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	close(m.done)
	if m.windowTicker != nil {
		m.windowTicker.Stop()
	}

	return nil
}

// DefineSLA defines a new SLA.
func (m *Manager) DefineSLA(def *SLADefinition) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if def.ID == "" {
		return fmt.Errorf("sla: SLA ID required")
	}

	if def.UptimeTarget == 0 {
		// Parse level to uptime target
		switch def.Level {
		case SLALevel99p9:
			def.UptimeTarget = 99.9
		case SLALevel99p99:
			def.UptimeTarget = 99.99
		case SLALevel99p999:
			def.UptimeTarget = 99.999
		default:
			return fmt.Errorf("sla: invalid SLA level: %s", def.Level)
		}
	}

	def.CreatedAt = time.Now()

	m.definitions[def.ID] = def

	// Create initial window
	now := time.Now()
	window := &SLAWindow{
		ID:              fmt.Sprintf("window-%s-%d", def.ID, now.Unix()),
		StartTime:       now,
		EndTime:         now.Add(m.config.WindowDuration),
		SLADefinitionID: def.ID,
		TotalTime:       m.config.WindowDuration,
	}

	m.windows[window.ID] = window
	m.currentWindows[def.ID] = window

	return nil
}

// GetSLA returns an SLA definition by ID.
func (m *Manager) GetSLA(slaID string) (*SLADefinition, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	def, ok := m.definitions[slaID]
	return def, ok
}

// UpdateUptime updates the uptime for a service in its current SLA window.
func (m *Manager) UpdateUptime(slaID string, uptime float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	def, ok := m.definitions[slaID]
	if !ok {
		return fmt.Errorf("sla: SLA not found: %s", slaID)
	}

	if !def.Enabled {
		return nil
	}

	window, ok := m.currentWindows[slaID]
	if !ok {
		return fmt.Errorf("sla: no active window for SLA: %s", slaID)
	}

	window.ActualUptime = uptime

	// Check for violation
	if uptime < def.UptimeTarget {
		violation := &SLAViolation{
			ID:                  fmt.Sprintf("violation-%s-%d", slaID, time.Now().UnixNano()),
			SLADefinitionID:     slaID,
			ViolationType:       "UPTIME",
			OccurredAt:          time.Now(),
			WindowID:            window.ID,
			Details: map[string]interface{}{
				"target":   def.UptimeTarget,
				"actual":   uptime,
				"deficit":  def.UptimeTarget - uptime,
			},
		}

		m.violations[violation.ID] = violation
		window.Violations++
		m.violationCount++

		// Trigger remediation if available
		if def.RemediationAction != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				for attempt := 0; attempt < m.config.MaxRemediationRetries; attempt++ {
					m.remediationAttempts++
					violation.RemediationAttempts++

					if err := def.RemediationAction(ctx, violation); err != nil {
						violation.LastRemediationError = err
						time.Sleep(time.Duration((attempt+1)*100) * time.Millisecond)
					} else {
						now := time.Now()
						violation.ResolvedAt = &now
						break
					}
				}
			}()
		}
	}

	return nil
}

// UpdateLatency updates the P99 latency for a service.
func (m *Manager) UpdateLatency(slaID string, latencyP99 time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	def, ok := m.definitions[slaID]
	if !ok {
		return fmt.Errorf("sla: SLA not found: %s", slaID)
	}

	if !def.Enabled || def.ResponseTimeP99 == 0 {
		return nil
	}

	window, ok := m.currentWindows[slaID]
	if !ok {
		return fmt.Errorf("sla: no active window for SLA: %s", slaID)
	}

	// Check for violation
	if latencyP99 > def.ResponseTimeP99 {
		violation := &SLAViolation{
			ID:              fmt.Sprintf("violation-%s-%d", slaID, time.Now().UnixNano()),
			SLADefinitionID: slaID,
			ViolationType:   "LATENCY",
			OccurredAt:      time.Now(),
			WindowID:        window.ID,
			Details: map[string]interface{}{
				"target": def.ResponseTimeP99.Seconds(),
				"actual": latencyP99.Seconds(),
			},
		}

		m.violations[violation.ID] = violation
		window.Violations++
		m.violationCount++
	}

	return nil
}

// UpdateErrorRate updates the error rate for a service.
func (m *Manager) UpdateErrorRate(slaID string, errorRate float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	def, ok := m.definitions[slaID]
	if !ok {
		return fmt.Errorf("sla: SLA not found: %s", slaID)
	}

	if !def.Enabled {
		return nil
	}

	window, ok := m.currentWindows[slaID]
	if !ok {
		return fmt.Errorf("sla: no active window for SLA: %s", slaID)
	}

	// Check for violation
	if errorRate > def.ErrorRateTarget {
		violation := &SLAViolation{
			ID:              fmt.Sprintf("violation-%s-%d", slaID, time.Now().UnixNano()),
			SLADefinitionID: slaID,
			ViolationType:   "ERROR_RATE",
			OccurredAt:      time.Now(),
			WindowID:        window.ID,
			Details: map[string]interface{}{
				"target": def.ErrorRateTarget,
				"actual": errorRate,
				"excess": errorRate - def.ErrorRateTarget,
			},
		}

		m.violations[violation.ID] = violation
		window.Violations++
		m.violationCount++
	}

	return nil
}

// GetWindow returns an SLA window by ID.
func (m *Manager) GetWindow(windowID string) (*SLAWindow, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	window, ok := m.windows[windowID]
	return window, ok
}

// GetViolations returns all violations for an SLA.
func (m *Manager) GetViolations(slaID string) []*SLAViolation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var violations []*SLAViolation
	for _, v := range m.violations {
		if v.SLADefinitionID == slaID {
			violations = append(violations, v)
		}
	}
	return violations
}

// checkCompliance checks SLA compliance for all active windows.
func (m *Manager) checkCompliance(ctx context.Context) {
	m.mu.RLock()
	windows := make([]*SLAWindow, 0, len(m.currentWindows))
	for _, w := range m.currentWindows {
		windows = append(windows, w)
	}
	m.mu.RUnlock()

	// Check each window for expiration
	now := time.Now()
	for _, window := range windows {
		if now.After(window.EndTime) {
			m.closeWindow(window)
		}
	}
}

// closeWindow closes an SLA window and opens a new one.
func (m *Manager) closeWindow(window *SLAWindow) {
	m.mu.Lock()
	defer m.mu.Unlock()

	window.Closed = true

	// Create new window
	now := time.Now()
	newWindow := &SLAWindow{
		ID:              fmt.Sprintf("window-%s-%d", window.SLADefinitionID, now.Unix()),
		StartTime:       now,
		EndTime:         now.Add(m.config.WindowDuration),
		SLADefinitionID: window.SLADefinitionID,
		TotalTime:       m.config.WindowDuration,
	}

	m.windows[newWindow.ID] = newWindow
	m.currentWindows[window.SLADefinitionID] = newWindow
}

// Report represents a compliance report.
type Report struct {
	SLAID         string
	ServiceName   string
	WindowCount   int
	TotalViolations int
	ComplianceRate float64
	NextWindowAt  time.Time
}

// GetComplianceReport returns a compliance report for an SLA.
func (m *Manager) GetComplianceReport(slaID string) (Report, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	def, ok := m.definitions[slaID]
	if !ok {
		return Report{}, fmt.Errorf("sla: SLA not found: %s", slaID)
	}

	windowCount := 0
	totalViolations := 0
	for _, w := range m.windows {
		if w.SLADefinitionID == slaID {
			windowCount++
			totalViolations += w.Violations
		}
	}

	currentWindow, ok := m.currentWindows[slaID]
	nextWindowAt := time.Now().Add(time.Hour)
	if ok && currentWindow != nil {
		nextWindowAt = currentWindow.EndTime
	}

	complianceRate := 100.0
	if windowCount > 0 && totalViolations > 0 {
		complianceRate = 100.0 - (float64(totalViolations) / float64(windowCount) * 100.0)
	}

	return Report{
		SLAID:           slaID,
		ServiceName:     def.ServiceName,
		WindowCount:     windowCount,
		TotalViolations: totalViolations,
		ComplianceRate:  complianceRate,
		NextWindowAt:    nextWindowAt,
	}, nil
}

// Stats returns SLA manager statistics.
type Stats struct {
	TotalSLAs           int
	ViolationCount      uint64
	RemediationAttempts uint64
}

// GetStats returns SLA manager statistics.
func (m *Manager) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Stats{
		TotalSLAs:           len(m.definitions),
		ViolationCount:      m.violationCount,
		RemediationAttempts: m.remediationAttempts,
	}
}
