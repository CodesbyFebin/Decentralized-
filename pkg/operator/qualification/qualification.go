// Package qualification implements operator tier progression and reputation tracking
// for the Decentralized.Host operator qualification program.
//
// The qualification system tracks operator progression through four tiers:
//   - BOOTSTRAP (0 uWork): Initial onboarding
//   - NOVICE (100K uWork): Basic operations
//   - JOURNEYMAN (1M uWork): Advanced operations
//   - MASTER (10M uWork): Full capabilities
//
// Reputation is tracked on a 0-1000 point scale, with penalties applied
// for SLA violations and rewards for compliance.
package qualification

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"decentralized.host/pkg/identity"
)

// Tier represents an operator's certification tier.
type Tier string

const (
	BOOTSTRAP  Tier = "BOOTSTRAP"
	NOVICE     Tier = "NOVICE"
	JOURNEYMAN Tier = "JOURNEYMAN"
	MASTER     Tier = "MASTER"
)

// TierThresholds define uWork requirements for each tier
var TierThresholds = map[Tier]uint64{
	BOOTSTRAP:  0,
	NOVICE:     100_000,
	JOURNEYMAN: 1_000_000,
	MASTER:     10_000_000,
}

// OperatorProfile tracks an operator's qualification status and metrics.
type OperatorProfile struct {
	OperatorID      string          `json:"operator_id"`
	NodeID          string          `json:"node_id"` // dh1 identity
	CurrentTier     Tier            `json:"current_tier"`
	AccumulatedWork uint64          `json:"accumulated_work"` // Total work completed (uWork)
	ReputationScore int16           `json:"reputation_score"` // 0-1000 points
	Certified       bool            `json:"certified"`
	CertifiedAt     *time.Time      `json:"certified_at,omitempty"`
	RevokedAt       *time.Time      `json:"revoked_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	TierHistory     []TierAdvance   `json:"tier_history,omitempty"`
	MetricsSnapshot OperatorMetrics `json:"metrics_snapshot"`
}

// TierAdvance records a tier progression event
type TierAdvance struct {
	FromTier   Tier      `json:"from_tier"`
	ToTier     Tier      `json:"to_tier"`
	WorkCount  uint64    `json:"work_count"`
	AdvancedAt time.Time `json:"advanced_at"`
	Reason     string    `json:"reason"`
}

// OperatorMetrics captures operational performance data
type OperatorMetrics struct {
	TotalWorkCount        uint64        `json:"total_work_count"`
	SuccessfulWorkCount   uint64        `json:"successful_work_count"`
	FailedWorkCount       uint64        `json:"failed_work_count"`
	AverageLatency        time.Duration `json:"average_latency"`
	UptimePercentage      float64       `json:"uptime_percentage"`
	SLAViolationCount     int           `json:"sla_violation_count"`
	LastHealthCheckAt     time.Time     `json:"last_health_check_at"`
	NodeCount             int           `json:"node_count"`
	GeographicRegions     int           `json:"geographic_regions"`
	FailoverNodesCount    int           `json:"failover_nodes_count"`
	LastMetricsUpdateTime time.Time     `json:"last_metrics_update_time"`
}

// QualificationManager manages operator qualification state
type QualificationManager struct {
	mu        sync.RWMutex
	operators map[string]*OperatorProfile
}

// NewQualificationManager creates a new qualification manager
func NewQualificationManager() *QualificationManager {
	return &QualificationManager{
		operators: make(map[string]*OperatorProfile),
	}
}

// RegisterOperator registers a new operator for qualification
func (qm *QualificationManager) RegisterOperator(operatorID string, nodeID string) (*OperatorProfile, error) {
	if operatorID == "" {
		return nil, errors.New("operator ID cannot be empty")
	}
	if !identity.IDPattern.MatchString(nodeID) {
		return nil, fmt.Errorf("invalid node ID format: %s", nodeID)
	}

	qm.mu.Lock()
	defer qm.mu.Unlock()

	if _, exists := qm.operators[operatorID]; exists {
		return nil, fmt.Errorf("operator %s already registered", operatorID)
	}

	now := time.Now()
	profile := &OperatorProfile{
		OperatorID:      operatorID,
		NodeID:          nodeID,
		CurrentTier:     BOOTSTRAP,
		AccumulatedWork: 0,
		ReputationScore: 500, // Start at neutral (500/1000)
		Certified:       false,
		CreatedAt:       now,
		UpdatedAt:       now,
		TierHistory:     []TierAdvance{},
		MetricsSnapshot: OperatorMetrics{
			LastHealthCheckAt:     now,
			LastMetricsUpdateTime: now,
		},
	}

	qm.operators[operatorID] = profile
	return profile, nil
}

// GetOperator retrieves an operator profile
func (qm *QualificationManager) GetOperator(operatorID string) (*OperatorProfile, error) {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	profile, exists := qm.operators[operatorID]
	if !exists {
		return nil, fmt.Errorf("operator %s not found", operatorID)
	}
	return profile, nil
}

// RecordWork records successful work completion and updates tier if needed
func (qm *QualificationManager) RecordWork(operatorID string, workCount uint64, latency time.Duration) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	profile, exists := qm.operators[operatorID]
	if !exists {
		return fmt.Errorf("operator %s not found", operatorID)
	}

	profile.AccumulatedWork += workCount
	profile.MetricsSnapshot.TotalWorkCount += workCount
	profile.MetricsSnapshot.SuccessfulWorkCount += workCount
	profile.UpdatedAt = time.Now()

	// Update average latency
	if profile.MetricsSnapshot.TotalWorkCount > 0 {
		currentAvg := profile.MetricsSnapshot.AverageLatency
		newAvg := (currentAvg*time.Duration(profile.MetricsSnapshot.TotalWorkCount-workCount) + latency*time.Duration(workCount)) /
			time.Duration(profile.MetricsSnapshot.TotalWorkCount)
		profile.MetricsSnapshot.AverageLatency = newAvg
	} else {
		profile.MetricsSnapshot.AverageLatency = latency
	}

	// Check for tier advancement
	return qm.checkAndAdvanceTier(profile)
}

// checkAndAdvanceTier checks if operator should advance to next tier
func (qm *QualificationManager) checkAndAdvanceTier(profile *OperatorProfile) error {
	var nextTier Tier
	shouldAdvance := false

	switch profile.CurrentTier {
	case BOOTSTRAP:
		if profile.AccumulatedWork >= TierThresholds[NOVICE] && profile.ReputationScore >= 400 {
			nextTier = NOVICE
			shouldAdvance = true
		}
	case NOVICE:
		if profile.AccumulatedWork >= TierThresholds[JOURNEYMAN] && profile.ReputationScore >= 600 {
			nextTier = JOURNEYMAN
			shouldAdvance = true
		}
	case JOURNEYMAN:
		if profile.AccumulatedWork >= TierThresholds[MASTER] && profile.ReputationScore >= 800 {
			nextTier = MASTER
			shouldAdvance = true
		}
	}

	if shouldAdvance {
		advance := TierAdvance{
			FromTier:   profile.CurrentTier,
			ToTier:     nextTier,
			WorkCount:  profile.AccumulatedWork,
			AdvancedAt: time.Now(),
			Reason:     fmt.Sprintf("Automatic advancement: %d work units, reputation %d", profile.AccumulatedWork, profile.ReputationScore),
		}
		profile.TierHistory = append(profile.TierHistory, advance)
		profile.CurrentTier = nextTier
	}

	return nil
}

// UpdateReputationScore updates an operator's reputation score
func (qm *QualificationManager) UpdateReputationScore(operatorID string, delta int16) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	profile, exists := qm.operators[operatorID]
	if !exists {
		return fmt.Errorf("operator %s not found", operatorID)
	}

	newScore := int32(profile.ReputationScore) + int32(delta)
	if newScore < 0 {
		profile.ReputationScore = 0
	} else if newScore > 1000 {
		profile.ReputationScore = 1000
	} else {
		profile.ReputationScore = int16(newScore)
	}

	profile.UpdatedAt = time.Now()
	return nil
}

// Certify marks an operator as certified for mainnet
func (qm *QualificationManager) Certify(operatorID string) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	profile, exists := qm.operators[operatorID]
	if !exists {
		return fmt.Errorf("operator %s not found", operatorID)
	}

	if profile.CurrentTier != MASTER {
		return fmt.Errorf("operator must be in MASTER tier to be certified, current: %s", profile.CurrentTier)
	}

	if profile.ReputationScore < 800 {
		return fmt.Errorf("operator reputation must be >= 800, current: %d", profile.ReputationScore)
	}

	now := time.Now()
	profile.Certified = true
	profile.CertifiedAt = &now
	profile.UpdatedAt = now

	return nil
}

// Revoke removes an operator's certification
func (qm *QualificationManager) Revoke(operatorID string, reason string) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	profile, exists := qm.operators[operatorID]
	if !exists {
		return fmt.Errorf("operator %s not found", operatorID)
	}

	now := time.Now()
	profile.Certified = false
	profile.RevokedAt = &now
	profile.UpdatedAt = now

	return nil
}

// ListOperators returns all registered operators (RLock)
func (qm *QualificationManager) ListOperators() []*OperatorProfile {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	operators := make([]*OperatorProfile, 0, len(qm.operators))
	for _, op := range qm.operators {
		operators = append(operators, op)
	}
	return operators
}

// CertifiedOperators returns only certified operators
func (qm *QualificationManager) CertifiedOperators() []*OperatorProfile {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	var certified []*OperatorProfile
	for _, op := range qm.operators {
		if op.Certified {
			certified = append(certified, op)
		}
	}
	return certified
}

// OperatorsByTier returns operators filtered by tier
func (qm *QualificationManager) OperatorsByTier(tier Tier) []*OperatorProfile {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	var filtered []*OperatorProfile
	for _, op := range qm.operators {
		if op.CurrentTier == tier {
			filtered = append(filtered, op)
		}
	}
	return filtered
}

// QualificationStatus returns summary statistics
type QualificationStatus struct {
	TotalOperators        int
	CertifiedOperators    int
	OperatorsReadyForMain int
	OperatorsByTier       map[Tier]int
	AverageReputation     int16
}

// GetQualificationStatus returns overall qualification program status
func (qm *QualificationManager) GetQualificationStatus() QualificationStatus {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	status := QualificationStatus{
		TotalOperators:    len(qm.operators),
		OperatorsByTier:   make(map[Tier]int),
		AverageReputation: 0,
	}

	var totalReputation int32
	for _, op := range qm.operators {
		if op.Certified && op.CurrentTier == MASTER {
			status.OperatorsReadyForMain++
		}
		if op.Certified {
			status.CertifiedOperators++
		}
		status.OperatorsByTier[op.CurrentTier]++
		totalReputation += int32(op.ReputationScore)
	}

	if len(qm.operators) > 0 {
		status.AverageReputation = int16(totalReputation / int32(len(qm.operators)))
	}

	return status
}

// SignOperatorID returns a deterministic hash of an operator identifier
// useful for operator identity verification in audit logs
func SignOperatorID(operatorID string, pub ed25519.PublicKey) string {
	h := sha256.New()
	h.Write([]byte(operatorID))
	h.Write(pub)
	return "op_" + hex.EncodeToString(h.Sum(nil))[:16]
}
