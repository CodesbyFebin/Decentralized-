package stake

import (
	"testing"
	"time"
)

func TestDepositStake(t *testing.T) {
	sm := NewStakingManager()

	tests := []struct {
		name      string
		operatorID string
		amount    uint64
		wantErr   bool
	}{
		{
			name:       "valid stake",
			operatorID: "op_001",
			amount:     MinimumStake,
			wantErr:    false,
		},
		{
			name:       "insufficient stake",
			operatorID: "op_002",
			amount:     MinimumStake / 2,
			wantErr:    true,
		},
		{
			name:       "empty operator ID",
			operatorID: "",
			amount:     MinimumStake,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sm.DepositStake(tt.operatorID, tt.amount)
			if (err != nil) != tt.wantErr {
				t.Errorf("DepositStake() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnbondStake(t *testing.T) {
	sm := NewStakingManager()
	operatorID := "op_001"

	sm.DepositStake(operatorID, MinimumStake)

	// Unbond
	err := sm.UnbondStake(operatorID)
	if err != nil {
		t.Fatalf("UnbondStake() error = %v", err)
	}

	account, _ := sm.GetStakeAccount(operatorID)
	if account.Status != UNBONDING {
		t.Errorf("expected UNBONDING status, got %s", account.Status)
	}

	// Try to unbond again (should fail)
	err = sm.UnbondStake(operatorID)
	if err == nil {
		t.Error("expected error unbonding from UNBONDING status")
	}
}

func TestCompleteUnbonding(t *testing.T) {
	sm := NewStakingManager()
	operatorID := "op_001"

	sm.DepositStake(operatorID, MinimumStake)
	sm.UnbondStake(operatorID)

	// Try to complete before period elapsed (should fail)
	err := sm.CompleteUnbonding(operatorID)
	if err == nil {
		t.Error("expected error completing unbonding before period")
	}

	// Manually move release time to past
	account, _ := sm.GetStakeAccount(operatorID)
	pastTime := time.Now().Add(-1 * time.Hour)
	account.ReleaseAt = &pastTime

	// Now complete should work
	err = sm.CompleteUnbonding(operatorID)
	if err != nil {
		t.Fatalf("CompleteUnbonding() error = %v", err)
	}

	account, _ = sm.GetStakeAccount(operatorID)
	if account.Status != RELEASED {
		t.Errorf("expected RELEASED status, got %s", account.Status)
	}
}

func TestSlashStake(t *testing.T) {
	sm := NewStakingManager()
	operatorID := "op_001"

	sm.DepositStake(operatorID, MinimumStake)

	// Slash 5%
	err := sm.SlashStake(operatorID, "SLA_VIOLATION", 5, "test violation")
	if err != nil {
		t.Fatalf("SlashStake() error = %v", err)
	}

	account, _ := sm.GetStakeAccount(operatorID)
	expectedAmount := MinimumStake - (MinimumStake / 20)
	if account.StakedAmount != expectedAmount {
		t.Errorf("expected stake %d, got %d", expectedAmount, account.StakedAmount)
	}

	if len(account.SlashHistory) == 0 {
		t.Error("expected slash to be recorded in history")
	}
}

func TestValidateStake(t *testing.T) {
	sm := NewStakingManager()
	operatorID := "op_001"

	// No stake
	err := sm.ValidateStake(operatorID)
	if err == nil {
		t.Error("expected error validating non-existent stake")
	}

	// Valid stake
	sm.DepositStake(operatorID, MinimumStake)
	err = sm.ValidateStake(operatorID)
	if err != nil {
		t.Errorf("ValidateStake() error = %v", err)
	}

	// Slash to insufficient
	sm.SlashStake(operatorID, "TEST", 100, "test")
	err = sm.ValidateStake(operatorID)
	if err == nil {
		t.Error("expected error validating insufficient stake")
	}
}

func TestGetStakingStatus(t *testing.T) {
	sm := NewStakingManager()

	for i := 0; i < 10; i++ {
		sm.DepositStake("op_"+string(rune(i)), MinimumStake)
	}

	status := sm.GetStakingStatus()
	if status.TotalOperators != 10 {
		t.Errorf("expected 10 operators, got %d", status.TotalOperators)
	}

	if status.ActiveOperators != 10 {
		t.Errorf("expected 10 active operators, got %d", status.ActiveOperators)
	}

	expectedTotal := MinimumStake * 10
	if status.TotalStaked != expectedTotal {
		t.Errorf("expected total staked %d, got %d", expectedTotal, status.TotalStaked)
	}
}

func TestAutoSlashSLAViolation(t *testing.T) {
	sm := NewStakingManager()
	operatorID := "op_001"

	sm.DepositStake(operatorID, MinimumStake)

	// Auto slash (default)
	err := sm.AutoSlashSLAViolation(operatorID, "NORMAL", "test")
	if err != nil {
		t.Fatalf("AutoSlashSLAViolation() error = %v", err)
	}

	account, _ := sm.GetStakeAccount(operatorID)
	expectedAmount := MinimumStake - (MinimumStake * uint64(DefaultSlashPercentage) / 100)
	if account.StakedAmount != expectedAmount {
		t.Errorf("expected stake %d, got %d", expectedAmount, account.StakedAmount)
	}

	// Auto slash critical
	sm.DepositStake("op_002", MinimumStake)
	err = sm.AutoSlashSLAViolation("op_002", "CRITICAL", "critical")
	if err != nil {
		t.Fatalf("AutoSlashSLAViolation() error = %v", err)
	}

	account, _ = sm.GetStakeAccount("op_002")
	expectedAmount = MinimumStake - (MinimumStake * uint64(SevereSlashPercentage) / 100)
	if account.StakedAmount != expectedAmount {
		t.Errorf("expected stake %d (critical slash), got %d", expectedAmount, account.StakedAmount)
	}
}

func TestListAccounts(t *testing.T) {
	sm := NewStakingManager()
	count := 5

	for i := 0; i < count; i++ {
		sm.DepositStake("op_"+string(rune(i)), MinimumStake)
	}

	accounts := sm.ListAccounts()
	if len(accounts) != count {
		t.Errorf("expected %d accounts, got %d", count, len(accounts))
	}
}
