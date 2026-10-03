// Package alerts implements alert management and escalation for Decentralized.Host.
//
// This package provides:
//   - Alert rule definitions with threshold-based triggering
//   - Multi-level severity (CRITICAL, HIGH, MEDIUM, LOW)
//   - Alert routing and escalation paths
//   - On-call scheduling integration
//   - Alert deduplication and state management
//   - Incident tracking and resolution
//
// Alerts are triggered when metrics exceed defined thresholds and can be
// routed to multiple escalation channels based on severity.
package alerts

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Severity defines the alert severity level.
type Severity string

const (
	// SeverityCritical indicates a critical alert requiring immediate action.
	SeverityCritical Severity = "CRITICAL"

	// SeverityHigh indicates a high-priority alert.
	SeverityHigh Severity = "HIGH"

	// SeverityMedium indicates a medium-priority alert.
	SeverityMedium Severity = "MEDIUM"

	// SeverityLow indicates a low-priority alert.
	SeverityLow Severity = "LOW"
)

// AlertState represents the current state of an alert.
type AlertState string

const (
	// StateActive indicates the alert is currently firing.
	StateActive AlertState = "ACTIVE"

	// StatePending indicates the alert threshold is being met.
	StatePending AlertState = "PENDING"

	// StateResolved indicates the alert has been resolved.
	StateResolved AlertState = "RESOLVED"

	// StateSuppressed indicates the alert has been suppressed.
	StateSuppressed AlertState = "SUPPRESSED"
)

// Rule defines an alert rule with thresholds and actions.
type Rule struct {
	// ID uniquely identifies this rule
	ID string

	// Name is a human-readable rule name
	Name string

	// Description explains what the alert means
	Description string

	// Severity of the alert when triggered
	Severity Severity

	// MetricName is the Prometheus metric to evaluate
	MetricName string

	// Condition defines the threshold (e.g., "> 0.9" for CPU > 90%)
	Condition string

	// Threshold is the numeric threshold value
	Threshold float64

	// EvaluationInterval is how often to evaluate this rule
	EvaluationInterval time.Duration

	// ForDuration is how long the condition must be true before alerting
	ForDuration time.Duration

	// Routes specify where this alert should be sent
	Routes []string

	// Enabled controls whether this rule is active
	Enabled bool

	// AnnotationTemplates provide additional context for the alert
	AnnotationTemplates map[string]string
}

// Alert represents a fired alert instance.
type Alert struct {
	// ID uniquely identifies this alert
	ID string

	// RuleID is the ID of the rule that triggered this alert
	RuleID string

	// State is the current state of the alert
	State AlertState

	// Severity is the alert severity
	Severity Severity

	// Summary is a brief description of the alert
	Summary string

	// Description is a detailed description
	Description string

	// FiredAt is when the alert was first fired
	FiredAt time.Time

	// ResolvedAt is when the alert was resolved (if resolved)
	ResolvedAt *time.Time

	// Labels contains metric labels
	Labels map[string]string

	// Annotations contains additional context
	Annotations map[string]string

	// GeneratorURL is the Prometheus URL that generated this alert
	GeneratorURL string

	// LastEvaluationTime is when the rule was last evaluated
	LastEvaluationTime time.Time

	// ConsecutiveFires is how many consecutive evaluations have triggered this alert
	ConsecutiveFires int
}

// Manager manages alert rules and instances.
type Manager struct {
	mu sync.RWMutex

	// rules maps rule ID to rule
	rules map[string]*Rule

	// alerts maps alert ID to alert instance
	alerts map[string]*Alert

	// routes maps route name to handler
	routes map[string]AlertHandler

	// evaluationTicker triggers periodic rule evaluation
	evaluationTicker *time.Ticker

	// config stores manager configuration
	config Config

	// done signals shutdown
	done chan struct{}

	// dedupCache maps (ruleID, labels) to last alert time for deduplication
	dedupCache map[string]time.Time

	// statistics
	firedCount     uint64
	resolvedCount  uint64
	suppressedCount uint64
}

// AlertHandler is called when an alert should be sent to a route.
type AlertHandler func(context.Context, *Alert) error

// Config defines alert manager configuration.
type Config struct {
	// EvaluationInterval is the default evaluation interval for rules
	EvaluationInterval time.Duration

	// DeduplicationWindow prevents duplicate alerts within this duration
	DeduplicationWindow time.Duration

	// MaxConcurrentEvaluations limits concurrent rule evaluations
	MaxConcurrentEvaluations int

	// HistorySize is the maximum number of historical alerts to keep
	HistorySize int
}

// DefaultConfig returns a default alert manager configuration.
func DefaultConfig() Config {
	return Config{
		EvaluationInterval:       time.Minute,
		DeduplicationWindow:      5 * time.Minute,
		MaxConcurrentEvaluations: 10,
		HistorySize:              10000,
	}
}

// NewManager creates a new alert manager.
func NewManager(cfg Config) *Manager {
	return &Manager{
		rules:      make(map[string]*Rule),
		alerts:     make(map[string]*Alert),
		routes:     make(map[string]AlertHandler),
		config:     cfg,
		done:       make(chan struct{}),
		dedupCache: make(map[string]time.Time),
	}
}

// Start starts the alert manager.
func (m *Manager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.evaluationTicker = time.NewTicker(m.config.EvaluationInterval)

	go func() {
		for {
			select {
			case <-m.evaluationTicker.C:
				m.evaluateRules(context.Background())
			case <-m.done:
				return
			}
		}
	}()
}

// Stop stops the alert manager.
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	close(m.done)
	if m.evaluationTicker != nil {
		m.evaluationTicker.Stop()
	}

	return nil
}

// AddRule adds a new alert rule.
func (m *Manager) AddRule(rule *Rule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rule.ID == "" {
		return fmt.Errorf("alerts: rule ID required")
	}

	if rule.ForDuration == 0 {
		rule.ForDuration = m.config.EvaluationInterval * 3
	}

	m.rules[rule.ID] = rule
	return nil
}

// RemoveRule removes an alert rule.
func (m *Manager) RemoveRule(ruleID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.rules, ruleID)
}

// GetRule returns a rule by ID.
func (m *Manager) GetRule(ruleID string) (*Rule, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rule, ok := m.rules[ruleID]
	return rule, ok
}

// RegisterRoute registers an alert handler for a route.
func (m *Manager) RegisterRoute(name string, handler AlertHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.routes[name] = handler
}

// FireAlert fires an alert.
func (m *Manager) FireAlert(ctx context.Context, ruleID string, alert *Alert) error {
	m.mu.Lock()

	rule, ok := m.rules[ruleID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("alerts: rule not found: %s", ruleID)
	}

	// Check deduplication
	dedupKey := fmt.Sprintf("%s:%v", ruleID, alert.Labels)
	if last, exists := m.dedupCache[dedupKey]; exists {
		if time.Since(last) < m.config.DeduplicationWindow {
			m.mu.Unlock()
			return nil // Deduplicated
		}
	}

	m.dedupCache[dedupKey] = time.Now()
	m.firedCount++

	alert.ID = fmt.Sprintf("alert-%d", len(m.alerts))
	alert.RuleID = ruleID
	alert.State = StateActive
	alert.FiredAt = time.Now()
	alert.Severity = rule.Severity

	m.alerts[alert.ID] = alert

	routes := append([]string{}, rule.Routes...)
	m.mu.Unlock()

	// Send to routes
	for _, routeName := range routes {
		if handler, ok := m.routes[routeName]; ok {
			if err := handler(ctx, alert); err != nil {
				fmt.Printf("alerts: route %s error: %v\n", routeName, err)
			}
		}
	}

	return nil
}

// ResolveAlert resolves an active alert.
func (m *Manager) ResolveAlert(alertID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	alert, ok := m.alerts[alertID]
	if !ok {
		return fmt.Errorf("alerts: alert not found: %s", alertID)
	}

	now := time.Now()
	alert.ResolvedAt = &now
	alert.State = StateResolved
	m.resolvedCount++

	return nil
}

// SuppressAlert suppresses an alert.
func (m *Manager) SuppressAlert(alertID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	alert, ok := m.alerts[alertID]
	if !ok {
		return fmt.Errorf("alerts: alert not found: %s", alertID)
	}

	alert.State = StateSuppressed
	m.suppressedCount++

	return nil
}

// GetAlert returns an alert by ID.
func (m *Manager) GetAlert(alertID string) (*Alert, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	alert, ok := m.alerts[alertID]
	return alert, ok
}

// ListActiveAlerts returns all active alerts.
func (m *Manager) ListActiveAlerts() []*Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []*Alert
	for _, alert := range m.alerts {
		if alert.State == StateActive {
			active = append(active, alert)
		}
	}
	return active
}

// ListAlerts returns all alerts.
func (m *Manager) ListAlerts() []*Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	alerts := make([]*Alert, 0, len(m.alerts))
	for _, alert := range m.alerts {
		alerts = append(alerts, alert)
	}
	return alerts
}

// evaluateRules evaluates all enabled rules.
func (m *Manager) evaluateRules(ctx context.Context) {
	m.mu.RLock()
	rules := make([]*Rule, 0, len(m.rules))
	for _, rule := range m.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	m.mu.RUnlock()

	// Evaluate rules (in a production system, this would evaluate actual metrics)
	for _, rule := range rules {
		// This is a placeholder; in production, metrics would be evaluated
		_ = rule
	}
}

// Stats returns alert manager statistics.
type Stats struct {
	TotalRules    int
	ActiveAlerts  int
	FiredCount    uint64
	ResolvedCount uint64
}

// GetStats returns alert manager statistics.
func (m *Manager) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	active := 0
	for _, alert := range m.alerts {
		if alert.State == StateActive {
			active++
		}
	}

	return Stats{
		TotalRules:    len(m.rules),
		ActiveAlerts:  active,
		FiredCount:    m.firedCount,
		ResolvedCount: m.resolvedCount,
	}
}
