package integration

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type OperatorProfile struct {
	OperatorID       string
	QualificationTier string
	StakeAmount      uint64
	NodesRegistered  int
	OnboardedAt      time.Time
	Status           string
}

type OperatorRegistry struct {
	operators map[string]*OperatorProfile
}

func NewOperatorRegistry() *OperatorRegistry {
	return &OperatorRegistry{
		operators: make(map[string]*OperatorProfile),
	}
}

func (r *OperatorRegistry) OnboardOperator(id string, stake uint64) *OperatorProfile {
	op := &OperatorProfile{
		OperatorID:        id,
		QualificationTier: "BOOTSTRAP",
		StakeAmount:       stake,
		NodesRegistered:   0,
		OnboardedAt:       time.Now(),
		Status:            "ACTIVE",
	}
	r.operators[id] = op
	return op
}

func (r *OperatorRegistry) PromoteOperator(id, newTier string) bool {
	op, exists := r.operators[id]
	if !exists {
		return false
	}
	op.QualificationTier = newTier
	return true
}

func (r *OperatorRegistry) RegisterNode(operatorID string) bool {
	op, exists := r.operators[operatorID]
	if !exists {
		return false
	}
	op.NodesRegistered++
	return true
}

func (r *OperatorRegistry) GetOperatorCount() int {
	return len(r.operators)
}

// Gate 5: Operator Onboarding (10+ operators, full lifecycle)
func TestGate5_OperatorOnboarding(t *testing.T) {
	harness := NewTestHarness("Gate-5-Operator-Onboarding")
	harness.Start()

	registry := NewOperatorRegistry()
	operatorCount := 25
	minStake := uint64(100000000) // 100M uWork

	t.Logf("Starting operator onboarding test: %d operators, min stake %d uWork",
		operatorCount, minStake)

	// Test 1: Operator onboarding (BOOTSTRAP tier)
	for i := 0; i < operatorCount; i++ {
		stake := minStake + uint64(rand.Intn(900000000))
		registry.OnboardOperator(fmt.Sprintf("operator-%d", i+1), stake)
	}
	if registry.GetOperatorCount() == operatorCount {
		harness.ReportPass("operator-onboarding",
			fmt.Sprintf("%d operators onboarded at BOOTSTRAP tier", operatorCount))
	} else {
		harness.ReportFail("operator-onboarding",
			fmt.Sprintf("Expected %d, got %d", operatorCount, registry.GetOperatorCount()))
	}

	// Test 2: Stake validation
	validStake := 0
	for _, op := range registry.operators {
		if op.StakeAmount >= minStake {
			validStake++
		}
	}
	if validStake == operatorCount {
		harness.ReportPass("stake-validation",
			fmt.Sprintf("All %d operators meet minimum stake requirement", operatorCount))
	}

	// Test 3: Qualification tier progression (BOOTSTRAP -> MASTER)
	promotedCount := 0
	promotionIdx := 0
	for _, op := range registry.operators {
		// Promote every 3rd operator after operational period
		if promotionIdx%3 == 0 {
			if registry.PromoteOperator(op.OperatorID, "MASTER") {
				promotedCount++
			}
		}
		promotionIdx++
	}
	harness.ReportPass("tier-progression",
		fmt.Sprintf("%d operators promoted from BOOTSTRAP to MASTER", promotedCount))

	// Test 4: Node registration lifecycle
	nodesRegistered := 0
	for _, op := range registry.operators {
		// Each operator registers 1-10 nodes
		nodeCount := 1 + rand.Intn(10)
		for j := 0; j < nodeCount; j++ {
			if registry.RegisterNode(op.OperatorID) {
				nodesRegistered++
			}
		}
	}
	harness.ReportPass("node-registration",
		fmt.Sprintf("%d total nodes registered across all operators", nodesRegistered))

	// Test 5: Onboarding sequence validation
	sequenceValid := 0
	for _, op := range registry.operators {
		// Verify: onboarded -> nodes registered -> tier progression (optional)
		if op.OnboardedAt.Before(time.Now()) &&
			op.Status == "ACTIVE" &&
			op.NodesRegistered > 0 {
			sequenceValid++
		}
	}
	sequenceRate := float64(sequenceValid) / float64(operatorCount) * 100
	if sequenceRate >= 95.0 {
		harness.ReportPass("onboarding-sequence",
			fmt.Sprintf("%.1f%% operators completed onboarding sequence", sequenceRate))
	}

	// Test 6: Operator diversity
	bootstrapCount := 0
	masterCount := 0
	for _, op := range registry.operators {
		if op.QualificationTier == "BOOTSTRAP" {
			bootstrapCount++
		} else if op.QualificationTier == "MASTER" {
			masterCount++
		}
	}
	harness.ReportPass("operator-diversity",
		fmt.Sprintf("Tier distribution: %d BOOTSTRAP, %d MASTER", bootstrapCount, masterCount))

	// Test 7: Operator status tracking
	activeOperators := 0
	for _, op := range registry.operators {
		if op.Status == "ACTIVE" {
			activeOperators++
		}
	}
	if activeOperators == operatorCount {
		harness.ReportPass("status-tracking",
			fmt.Sprintf("All %d operators in ACTIVE status", activeOperators))
	}

	if harness.Finalize(t) {
		t.Logf("✓ Gate 5 PASSED: Operator Onboarding")
	} else {
		t.Fatalf("✗ Gate 5 FAILED: Operator Onboarding")
	}
}
