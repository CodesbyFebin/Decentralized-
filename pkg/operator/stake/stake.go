// Package stake implements operator stake validation, bonding, and slashing
// for the Decentralized.Host operator qualification program.
//
// Each operator must maintain a minimum 100M uWork stake. Stake is locked
// for 14 days after unbonding is initiated. Slashing penalties are applied
// for SLA violations or security breaches.
package stake

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	// MinimumStake is the minimum uWork required to participate
	MinimumStake uint64 = 100_000_000

	// UnbondingPeriod is the time before stake is released after unbonding
	UnbondingPeriod = 14 * 24 * time.Hour

	// DefaultSlashPercentage is the default slash amount (percentage of stake)
	DefaultSlashPercentage = 5 // 5%

	// SevereSlashPercentage for critical violations
	SevereSlashPercentage = 20 // 20%
)

// StakeStatus represents the status of staked tokens
type StakeStatus string

const (
	ACTIVE       StakeStatus = "ACTIVE"
	UNBONDING    StakeStatus = "UNBONDING"
	RELEASED     StakeStatus = "RELEASED"
	SLASHED      StakeStatus = "SLASHED"
	INSUFFICIENT StakeStatus = "INSUFFICIENT"
)

// StakeAccount tracks an operator's stake and bonding
type StakeAccount struct {
	OperatorID   string        `json:"operator_id"`
	StakedAmount uint64        `json:"staked_amount"`
	Status       StakeStatus   `json:"status"`
	LockedAt     time.Time     `json:"locked_at"`
	UnbondAt     *time.Time    `json:"unbond_at,omitempty"`
	ReleaseAt    *time.Time    `json:"release_at,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	SlashHistory []SlashEvent  `json:"slash_history,omitempty"`
	TotalSlashed uint64        `json:"total_slashed"`
}

// SlashEvent records a slashing action
type SlashEvent struct {
	Reason      string    `json:"reason"`
	Amount      uint64    `json:"amount"`
	Percentage  int       `json:"percentage"`
	SlashedAt   time.Time `json:"slashed_at"`
	ExecID      string    `json:"exec_id,omitempty"` // Reference to failing execution
	Description string    `json:"description,omitempty"`
}

// StakingManager manages operator stakes and bonding
type StakingManager struct {
	mu    sync.RWMutex
	stake map[string]*StakeAccount
}

// NewStakingManager creates a new staking manager
func NewStakingManager() *StakingManager {
	return &StakingManager{
		stake: make(map[string]*StakeAccount),
	}
}

// DepositStake deposits stake for an operator
func (sm *StakingManager) DepositStake(operatorID string, amount uint64) (*StakeAccount, error) {
	if operatorID == "" {
		return nil, errors.New("operator ID cannot be empty")
	}

	if amount < MinimumStake {
		return nil, fmt.Errorf("stake amount %d is below minimum %d", amount, MinimumStake)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()

	// If account doesn't exist, create it
	account, exists := sm.stake[operatorID]
	if !exists {
		account = &StakeAccount{
			OperatorID:   operatorID,
			StakedAmount: amount,
			Status:       ACTIVE,
			LockedAt:     now,
			CreatedAt:    now,
			UpdatedAt:    now,
			SlashHistory: []SlashEvent{},
		}
	} else {
		// Top-up existing stake
		account.StakedAmount += amount
		account.Status = ACTIVE
		account.UpdatedAt = now
	}

	sm.stake[operatorID] = account
	return account, nil
}

// GetStakeAccount retrieves an operator's stake account
func (sm *StakingManager) GetStakeAccount(operatorID string) (*StakeAccount, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	account, exists := sm.stake[operatorID]
	if !exists {
		return nil, fmt.Errorf("no stake account for operator %s", operatorID)
	}
	return account, nil
}

// UnbondStake initiates unbonding for an operator's stake
func (sm *StakingManager) UnbondStake(operatorID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	account, exists := sm.stake[operatorID]
	if !exists {
		return fmt.Errorf("no stake account for operator %s", operatorID)
	}

	if account.Status != ACTIVE {
		return fmt.Errorf("cannot unbond from status %s", account.Status)
	}

	now := time.Now()
	unbondAt := now
	releaseAt := now.Add(UnbondingPeriod)
	account.Status = UNBONDING
	account.UnbondAt = &unbondAt
	account.ReleaseAt = &releaseAt
	account.UpdatedAt = now

	return nil
}

// CompleteUnbonding completes the unbonding process if the period has elapsed
func (sm *StakingManager) CompleteUnbonding(operatorID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	account, exists := sm.stake[operatorID]
	if !exists {
		return fmt.Errorf("no stake account for operator %s", operatorID)
	}

	if account.Status != UNBONDING {
		return fmt.Errorf("account is not in UNBONDING status, current: %s", account.Status)
	}

	if account.ReleaseAt == nil || time.Now().Before(*account.ReleaseAt) {
		return fmt.Errorf("unbonding period not yet elapsed, release at %v", account.ReleaseAt)
	}

	account.Status = RELEASED
	account.UpdatedAt = time.Now()
	return nil
}

// SlashStake applies a slashing penalty to an operator's stake
func (sm *StakingManager) SlashStake(operatorID string, reason string, percentage int, description string) error {
	if percentage < 0 || percentage > 100 {
		return fmt.Errorf("invalid slash percentage: %d (must be 0-100)", percentage)
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	account, exists := sm.stake[operatorID]
	if !exists {
		return fmt.Errorf("no stake account for operator %s", operatorID)
	}

	slashAmount := (account.StakedAmount * uint64(percentage)) / 100
	if slashAmount == 0 && percentage > 0 {
		slashAmount = 1 // Ensure at least 1 unit is slashed if percentage > 0
	}

	if slashAmount > account.StakedAmount {
		slashAmount = account.StakedAmount
	}

	// Record slash event
	event := SlashEvent{
		Reason:      reason,
		Amount:      slashAmount,
		Percentage:  percentage,
		SlashedAt:   time.Now(),
		Description: description,
	}

	account.SlashHistory = append(account.SlashHistory, event)
	account.StakedAmount -= slashAmount
	account.TotalSlashed += slashAmount
	account.UpdatedAt = time.Now()

	// Update status if stake falls below minimum
	if account.StakedAmount < MinimumStake && account.Status == ACTIVE {
		account.Status = INSUFFICIENT
	}

	return nil
}

// GetStakeStatus returns the current status of an operator's stake
type StakeStatus2 struct {
	OperatorID   string
	StakedAmount uint64
	Status       StakeStatus
	IsActive     bool
	IsSufficient bool
	SlashCount   int
	TotalSlashed uint64
}

// GetStatus returns comprehensive stake status
func (sm *StakingManager) GetStatus(operatorID string) (*StakeStatus2, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	account, exists := sm.stake[operatorID]
	if !exists {
		return nil, fmt.Errorf("no stake account for operator %s", operatorID)
	}

	return &StakeStatus2{
		OperatorID:   operatorID,
		StakedAmount: account.StakedAmount,
		Status:       account.Status,
		IsActive:     account.Status == ACTIVE,
		IsSufficient: account.StakedAmount >= MinimumStake,
		SlashCount:   len(account.SlashHistory),
		TotalSlashed: account.TotalSlashed,
	}, nil
}

// ValidateStake checks if an operator has sufficient stake
func (sm *StakingManager) ValidateStake(operatorID string) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	account, exists := sm.stake[operatorID]
	if !exists {
		return fmt.Errorf("no stake account for operator %s", operatorID)
	}

	if account.Status != ACTIVE {
		return fmt.Errorf("stake not active, status: %s", account.Status)
	}

	if account.StakedAmount < MinimumStake {
		return fmt.Errorf("insufficient stake: %d (required: %d)", account.StakedAmount, MinimumStake)
	}

	return nil
}

// ListAccounts returns all stake accounts
func (sm *StakingManager) ListAccounts() []*StakeAccount {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	accounts := make([]*StakeAccount, 0, len(sm.stake))
	for _, acc := range sm.stake {
		accounts = append(accounts, acc)
	}
	return accounts
}

// StakingStatus provides aggregate staking information
type StakingStatus struct {
	TotalStaked      uint64
	TotalOperators   int
	ActiveOperators  int
	SlashedOperators int
	AverageStake     uint64
}

// GetStakingStatus returns overall staking statistics
func (sm *StakingManager) GetStakingStatus() StakingStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	status := StakingStatus{
		TotalOperators: len(sm.stake),
	}

	for _, acc := range sm.stake {
		status.TotalStaked += acc.StakedAmount
		if acc.Status == ACTIVE {
			status.ActiveOperators++
		}
		if len(acc.SlashHistory) > 0 {
			status.SlashedOperators++
		}
	}

	if status.TotalOperators > 0 {
		status.AverageStake = status.TotalStaked / uint64(status.TotalOperators)
	}

	return status
}

// AutoSlashSLAViolation applies automatic slashing for SLA violations
func (sm *StakingManager) AutoSlashSLAViolation(operatorID string, severity string, description string) error {
	slashPercentage := DefaultSlashPercentage

	if severity == "CRITICAL" {
		slashPercentage = SevereSlashPercentage
	}

	return sm.SlashStake(operatorID, "SLA_VIOLATION", slashPercentage, description)
}
