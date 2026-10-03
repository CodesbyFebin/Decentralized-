package qualification

import (
	"testing"
	"time"
)

func TestRegisterOperator(t *testing.T) {
	qm := NewQualificationManager()

	tests := []struct {
		name      string
		operatorID string
		nodeID     string
		wantErr   bool
	}{
		{
			name:       "valid registration",
			operatorID: "op_001",
			nodeID:     "dh1abcdefghijklmnopqrstuvwxyz",
			wantErr:    false,
		},
		{
			name:       "empty operator ID",
			operatorID: "",
			nodeID:     "dh1abcdefghijklmnopqrstuvwxyz",
			wantErr:    true,
		},
		{
			name:       "invalid node ID",
			operatorID: "op_001",
			nodeID:     "invalid",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := qm.RegisterOperator(tt.operatorID, tt.nodeID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterOperator() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && profile.CurrentTier != BOOTSTRAP {
				t.Errorf("expected BOOTSTRAP tier, got %s", profile.CurrentTier)
			}
		})
	}
}

func TestRecordWork(t *testing.T) {
	qm := NewQualificationManager()
	operatorID := "op_001"
	qm.RegisterOperator(operatorID, "dh1abcdefghijklmnopqrstuvwxyz")

	// Record work to reach NOVICE
	err := qm.RecordWork(operatorID, 100_000, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("RecordWork() error = %v", err)
	}

	profile, _ := qm.GetOperator(operatorID)
	if profile.CurrentTier != NOVICE {
		t.Errorf("expected NOVICE tier after 100k work, got %s", profile.CurrentTier)
	}

	if len(profile.TierHistory) == 0 {
		t.Error("expected tier history to be recorded")
	}
}

func TestReputationScore(t *testing.T) {
	qm := NewQualificationManager()
	operatorID := "op_001"
	qm.RegisterOperator(operatorID, "dh1abcdefghijklmnopqrstuvwxyz")

	// Decrease reputation
	qm.UpdateReputationScore(operatorID, -100)
	profile, _ := qm.GetOperator(operatorID)
	if profile.ReputationScore != 400 {
		t.Errorf("expected reputation 400, got %d", profile.ReputationScore)
	}

	// Test bounds
	qm.UpdateReputationScore(operatorID, 1000)
	profile, _ = qm.GetOperator(operatorID)
	if profile.ReputationScore != 1000 {
		t.Errorf("expected max reputation 1000, got %d", profile.ReputationScore)
	}
}

func TestCertification(t *testing.T) {
	qm := NewQualificationManager()
	operatorID := "op_001"
	qm.RegisterOperator(operatorID, "dh1abcdefghijklmnopqrstuvwxyz")

	// Try to certify BOOTSTRAP operator (should fail)
	err := qm.Certify(operatorID)
	if err == nil {
		t.Error("expected error certifying BOOTSTRAP operator")
	}

	// Advance to MASTER
	profile, _ := qm.GetOperator(operatorID)
	profile.CurrentTier = MASTER
	profile.ReputationScore = 850

	// Now certify should work
	err = qm.Certify(operatorID)
	if err != nil {
		t.Errorf("Certify() error = %v", err)
	}

	profile, _ = qm.GetOperator(operatorID)
	if !profile.Certified {
		t.Error("expected certified flag to be true")
	}
	if profile.CertifiedAt == nil {
		t.Error("expected CertifiedAt to be set")
	}
}

func TestQualificationStatus(t *testing.T) {
	qm := NewQualificationManager()

	for i := 0; i < 25; i++ {
		qm.RegisterOperator("op_"+string(rune(i)), "dh1abcdefghijklmnopqrstuvwxyz")
	}

	status := qm.GetQualificationStatus()
	if status.TotalOperators != 25 {
		t.Errorf("expected 25 operators, got %d", status.TotalOperators)
	}
}

func TestListOperators(t *testing.T) {
	qm := NewQualificationManager()
	count := 10

	for i := 0; i < count; i++ {
		qm.RegisterOperator("op_"+string(rune(i)), "dh1abcdefghijklmnopqrstuvwxyz")
	}

	operators := qm.ListOperators()
	if len(operators) != count {
		t.Errorf("expected %d operators, got %d", count, len(operators))
	}
}

func TestOperatorsByTier(t *testing.T) {
	qm := NewQualificationManager()

	for i := 0; i < 5; i++ {
		qm.RegisterOperator("op_"+string(rune(i)), "dh1abcdefghijklmnopqrstuvwxyz")
	}

	bootstrapOps := qm.OperatorsByTier(BOOTSTRAP)
	if len(bootstrapOps) != 5 {
		t.Errorf("expected 5 BOOTSTRAP operators, got %d", len(bootstrapOps))
	}
}
