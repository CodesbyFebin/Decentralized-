// Package integration provides multi-operator coordination tests for dh/v1.
// These tests validate behavior when 50+ operators interact with conflicting intentions.
package integration

import (
	cryptoRand "crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================================
// Multi-Operator Testing Infrastructure
// ============================================================================

type StakeAllocation struct {
	Operator string
	Amount   uint64
	LockTime time.Time
	Released bool
}

type ConsensusVote struct {
	OperatorID  string
	ProposalID  string
	VoteValue   int // -1: no, 0: abstain, 1: yes
	Timestamp   time.Time
	SignatureOK bool
}

type OperatorCoordinator struct {
	Operators       map[string]*Operator
	Proposals       map[string]*Proposal
	Stakes          []StakeAllocation
	Votes           []ConsensusVote
	AuditLog        []AuditEvent
	lock            sync.RWMutex
	TotalStaked     uint64
	TotalDisputed   uint64
	DisputesClosed  int64
}

type Operator struct {
	ID              string
	PublicKey       string
	Stake           uint64
	LockedStake     uint64
	Tier            string // BOOTSTRAP, QUALIFIED, TRUSTED
	OnboardedAt     time.Time
	ComplianceScore float64
	LastAction      time.Time
	Disputed        bool
}

type Proposal struct {
	ID              string
	InitiatorID     string
	Description     string
	Type            string // STAKE_SLASH, POLICY_CHANGE, VALIDATOR_ROTATION
	VotesYes        uint64
	VotesNo         uint64
	VotesAbstain    uint64
	Threshold       float64 // e.g., 0.67 for 2/3 majority
	Status          string  // OPEN, PASSED, FAILED, DISPUTED
	CreatedAt       time.Time
	ExpiresAt       time.Time
	ResolutionTime  time.Time
}

type AuditEvent struct {
	Timestamp  time.Time
	Actor      string
	Action     string
	TargetOP   string
	Details    string
	Evidence   map[string]string
}

func NewOperatorCoordinator() *OperatorCoordinator {
	return &OperatorCoordinator{
		Operators:   make(map[string]*Operator),
		Proposals:   make(map[string]*Proposal),
		Stakes:      make([]StakeAllocation, 0),
		Votes:       make([]ConsensusVote, 0),
		AuditLog:    make([]AuditEvent, 0),
		TotalStaked: 0,
	}
}

func (oc *OperatorCoordinator) RegisterOperator(id string, stake uint64) error {
	oc.lock.Lock()
	defer oc.lock.Unlock()

	if _, exists := oc.Operators[id]; exists {
		return fmt.Errorf("operator %s already registered", id)
	}

	pk := make([]byte, 32)
	cryptoRand.Read(pk)

	op := &Operator{
		ID:          id,
		PublicKey:   hex.EncodeToString(pk),
		Stake:       stake,
		Tier:        "BOOTSTRAP",
		OnboardedAt: time.Now(),
		LastAction:  time.Now(),
	}

	oc.Operators[id] = op
	oc.TotalStaked += stake

	oc.logEvent(AuditEvent{
		Timestamp: time.Now(),
		Actor:     "system",
		Action:    "operator-registered",
		TargetOP:  id,
		Details:   fmt.Sprintf("Stake: %d", stake),
	})

	return nil
}

func (oc *OperatorCoordinator) CreateProposal(id, initiatorID, description, pType string, threshold float64) (*Proposal, error) {
	oc.lock.Lock()
	defer oc.lock.Unlock()

	if _, exists := oc.Operators[initiatorID]; !exists {
		return nil, fmt.Errorf("initiator %s not found", initiatorID)
	}

	p := &Proposal{
		ID:          id,
		InitiatorID: initiatorID,
		Description: description,
		Type:        pType,
		Threshold:   threshold,
		Status:      "OPEN",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	oc.Proposals[id] = p

	oc.logEvent(AuditEvent{
		Timestamp: time.Now(),
		Actor:     initiatorID,
		Action:    "proposal-created",
		Details:   fmt.Sprintf("Type: %s, Threshold: %.2f", pType, threshold),
		Evidence: map[string]string{
			"proposal_id": id,
		},
	})

	return p, nil
}

func (oc *OperatorCoordinator) CastVote(proposalID, operatorID string, vote int) error {
	oc.lock.Lock()
	defer oc.lock.Unlock()

	p, pExists := oc.Proposals[proposalID]
	if !pExists || p.Status != "OPEN" {
		return fmt.Errorf("proposal %s not open for voting", proposalID)
	}

	op, opExists := oc.Operators[operatorID]
	if !opExists {
		return fmt.Errorf("operator %s not found", operatorID)
	}

	// Check for duplicate vote
	for _, v := range oc.Votes {
		if v.ProposalID == proposalID && v.OperatorID == operatorID {
			return fmt.Errorf("operator %s already voted", operatorID)
		}
	}

	// Weight votes by stake
	stakeWeight := op.Stake

	switch vote {
	case 1:
		p.VotesYes += stakeWeight
	case -1:
		p.VotesNo += stakeWeight
	case 0:
		p.VotesAbstain += stakeWeight
	default:
		return fmt.Errorf("invalid vote value: %d", vote)
	}

	cv := ConsensusVote{
		OperatorID:  operatorID,
		ProposalID:  proposalID,
		VoteValue:   vote,
		Timestamp:   time.Now(),
		SignatureOK: true,
	}

	oc.Votes = append(oc.Votes, cv)
	op.LastAction = time.Now()

	oc.logEvent(AuditEvent{
		Timestamp: time.Now(),
		Actor:     operatorID,
		Action:    "vote-cast",
		Details:   fmt.Sprintf("Proposal: %s, Vote: %d, Weight: %d", proposalID, vote, stakeWeight),
	})

	return nil
}

func (oc *OperatorCoordinator) FinalizeProposal(proposalID string) error {
	oc.lock.Lock()
	defer oc.lock.Unlock()

	p, exists := oc.Proposals[proposalID]
	if !exists {
		return fmt.Errorf("proposal %s not found", proposalID)
	}

	if p.Status != "OPEN" {
		return fmt.Errorf("proposal %s already finalized", proposalID)
	}

	totalVotes := p.VotesYes + p.VotesNo + p.VotesAbstain

	if totalVotes == 0 {
		p.Status = "FAILED"
		return nil
	}

	yesRatio := float64(p.VotesYes) / float64(p.VotesYes + p.VotesNo)

	if yesRatio >= p.Threshold {
		p.Status = "PASSED"
	} else {
		p.Status = "FAILED"
	}

	p.ResolutionTime = time.Now()

	oc.logEvent(AuditEvent{
		Timestamp: time.Now(),
		Actor:     "system",
		Action:    "proposal-finalized",
		Details:   fmt.Sprintf("Status: %s, Yes: %d, No: %d, Abstain: %d, Ratio: %.2f%%",
			p.Status, p.VotesYes, p.VotesNo, p.VotesAbstain, yesRatio*100),
		Evidence: map[string]string{
			"proposal_id": proposalID,
		},
	})

	return nil
}

func (oc *OperatorCoordinator) SlashStake(operatorID string, slashPercentage float64) error {
	oc.lock.Lock()
	defer oc.lock.Unlock()

	op, exists := oc.Operators[operatorID]
	if !exists {
		return fmt.Errorf("operator %s not found", operatorID)
	}

	slashAmount := uint64(float64(op.Stake) * slashPercentage / 100)
	if slashAmount > op.Stake {
		slashAmount = op.Stake
	}

	op.Stake -= slashAmount
	oc.TotalStaked -= slashAmount
	oc.TotalDisputed += slashAmount

	oc.logEvent(AuditEvent{
		Timestamp: time.Now(),
		Actor:     "system",
		Action:    "stake-slashed",
		TargetOP:  operatorID,
		Details:   fmt.Sprintf("Slashed: %d (%.2f%%), Remaining: %d",
			slashAmount, slashPercentage, op.Stake),
	})

	return nil
}

func (oc *OperatorCoordinator) logEvent(e AuditEvent) {
	oc.AuditLog = append(oc.AuditLog, e)
}

// ============================================================================
// Test: 50+ Concurrent Operators
// ============================================================================

// TestMultiOperator_50ConcurrentOperators tests system with 50+ operators
func TestMultiOperator_50ConcurrentOperators(t *testing.T) {
	harness := NewTestHarness("MultiOp-50Concurrent")
	harness.Start()

	const numOperators = 50
	const minStake = uint64(100000000) // 100M uWork

	coord := NewOperatorCoordinator()

	// Register 50 operators
	wg := sync.WaitGroup{}
	errCount := atomic.Int32{}

	for i := 0; i < numOperators; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			opID := fmt.Sprintf("op-%d", idx)
			stake := minStake + uint64(idx)*1000000

			if err := coord.RegisterOperator(opID, stake); err != nil {
				errCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	coord.lock.RLock()
	registeredOps := len(coord.Operators)
	totalStaked := coord.TotalStaked
	coord.lock.RUnlock()

	if registeredOps == numOperators && errCount.Load() == 0 {
		harness.ReportPass("50-operators-registration",
			fmt.Sprintf("Registered %d operators, Total stake: %d", registeredOps, totalStaked))
	} else {
		harness.ReportFail("50-operators-registration",
			fmt.Sprintf("Expected %d, got %d (errors: %d)", numOperators, registeredOps, errCount.Load()))
	}

	harness.Finalize(t)
}

// ============================================================================
// Test: Consensus Under Disagreement
// ============================================================================

// TestMultiOperator_ConsensusDisagreement tests voting with conflicting positions
func TestMultiOperator_ConsensusDisagreement(t *testing.T) {
	harness := NewTestHarness("MultiOp-ConsensusDisagreement")
	harness.Start()

	const numOperators = 30
	const minStake = uint64(100000000)

	coord := NewOperatorCoordinator()

	// Register operators with varying stake
	for i := 0; i < numOperators; i++ {
		opID := fmt.Sprintf("op-%d", i)
		stake := minStake + uint64(i%10)*10000000
		coord.RegisterOperator(opID, stake)
	}

	// Create a proposal
	_, _ = coord.CreateProposal("proposal-1", "op-0", "Policy change", "POLICY_CHANGE", 0.67)

	// Operators vote with disagreement
	// Group 1: 40% vote YES (smaller stake)
	// Group 2: 40% vote NO (larger stake)
	// Group 3: 20% abstain
	coord.lock.RLock()
	operators := make([]*Operator, 0, len(coord.Operators))
	for _, op := range coord.Operators {
		operators = append(operators, op)
	}
	coord.lock.RUnlock()

	voteGroup1Idx := numOperators * 40 / 100
	voteGroup2Idx := numOperators * 80 / 100

	voteCount := atomic.Int64{}
	voteFailCount := atomic.Int64{}

	for i, op := range operators {
		var vote int
		if i < voteGroup1Idx {
			vote = 1 // YES
		} else if i < voteGroup2Idx {
			vote = -1 // NO
		} else {
			vote = 0 // ABSTAIN
		}

		if err := coord.CastVote("proposal-1", op.ID, vote); err != nil {
			voteFailCount.Add(1)
		} else {
			voteCount.Add(1)
		}
	}

	coord.lock.RLock()
	p := coord.Proposals["proposal-1"]
	coord.lock.RUnlock()

	totalVotes := p.VotesYes + p.VotesNo + p.VotesAbstain
	yesPercentage := float64(p.VotesYes) / float64(totalVotes) * 100

	if voteCount.Load() == int64(numOperators) {
		harness.ReportPass("consensus-vote-aggregation",
			fmt.Sprintf("Votes: YES %.2f%% (%d), NO %.2f%% (%d), ABSTAIN: %d, Total: %d",
				yesPercentage, p.VotesYes, 100-yesPercentage, p.VotesNo, p.VotesAbstain, totalVotes))
	}

	// Finalize and check outcome
	coord.FinalizeProposal("proposal-1")

	coord.lock.RLock()
	status := coord.Proposals["proposal-1"].Status
	coord.lock.RUnlock()

	if status == "FAILED" { // Should fail because disagreement
		harness.ReportPass("consensus-disagreement-resolution",
			fmt.Sprintf("Proposal correctly FAILED under disagreement"))
	}

	harness.Finalize(t)
}

// ============================================================================
// Test: Stake Redistribution Scenarios
// ============================================================================

// TestMultiOperator_StakeRedistribution tests stake slashing and redistribution
func TestMultiOperator_StakeRedistribution(t *testing.T) {
	harness := NewTestHarness("MultiOp-StakeRedistribution")
	harness.Start()

	const numOperators = 20
	const minStake = uint64(100000000)

	coord := NewOperatorCoordinator()

	// Register operators
	for i := 0; i < numOperators; i++ {
		opID := fmt.Sprintf("op-%d", i)
		stake := minStake
		coord.RegisterOperator(opID, stake)
	}

	coord.lock.RLock()
	initialTotalStake := coord.TotalStaked
	coord.lock.RUnlock()

	// Scenario 1: Slash a single operator
	coord.SlashStake("op-0", 10.0) // 10% slash

	coord.lock.RLock()
	slashedOp := coord.Operators["op-0"]
	totalDisputed := coord.TotalDisputed
	coord.lock.RUnlock()

	expectedSlash := minStake / 10
	if slashedOp.Stake == minStake-expectedSlash && totalDisputed == expectedSlash {
		harness.ReportPass("stake-slashing-single",
			fmt.Sprintf("Slashed operator-0: %.0f%%, Remaining stake: %d, Disputed pool: %d",
				10.0, slashedOp.Stake, totalDisputed))
	}

	// Scenario 2: Multiple slashings
	slashCount := 0
	for i := 1; i < 5; i++ {
		opID := fmt.Sprintf("op-%d", i)
		coord.SlashStake(opID, 5.0)
		slashCount++
	}

	coord.lock.RLock()
	finalDisputed := coord.TotalDisputed
	coord.lock.RUnlock()

	if finalDisputed > totalDisputed {
		harness.ReportPass("stake-slashing-multiple",
			fmt.Sprintf("Slashed %d operators, Total disputed: %d", slashCount, finalDisputed))
	}

	// Scenario 3: Verify stake conservation
	coord.lock.RLock()
	sumOperatorStakes := uint64(0)
	for _, op := range coord.Operators {
		sumOperatorStakes += op.Stake
	}
	finalTotal := coord.TotalStaked
	coord.lock.RUnlock()

	totalAccounted := finalTotal + finalDisputed
	if totalAccounted == initialTotalStake {
		harness.ReportPass("stake-conservation",
			fmt.Sprintf("Stake conserved: Active %d + Disputed %d = Initial %d",
				finalTotal, finalDisputed, initialTotalStake))
	} else {
		harness.ReportFail("stake-conservation",
			fmt.Sprintf("Mismatch: %d vs initial %d", totalAccounted, initialTotalStake))
	}

	harness.Finalize(t)
}

// ============================================================================
// Test: Dispute Resolution
// ============================================================================

// TestMultiOperator_DisputeResolution tests dispute handling and operator slashing
func TestMultiOperator_DisputeResolution(t *testing.T) {
	harness := NewTestHarness("MultiOp-DisputeResolution")
	harness.Start()

	const numOperators = 40
	const minStake = uint64(100000000)

	coord := NewOperatorCoordinator()

	// Register operators
	for i := 0; i < numOperators; i++ {
		opID := fmt.Sprintf("op-%d", i)
		stake := minStake
		coord.RegisterOperator(opID, stake)
	}

	// Create dispute proposal against operator-1
	_, _ = coord.CreateProposal("dispute-1", "op-0", "Operator-1 breach", "STAKE_SLASH", 0.51)

	// Simulate jury voting on dispute (simple majority)
	coord.lock.RLock()
	allOps := make([]*Operator, 0, len(coord.Operators))
	for _, op := range coord.Operators {
		if op.ID != "op-1" { // Disputed operator can't vote
			allOps = append(allOps, op)
		}
	}
	coord.lock.RUnlock()

	// 70% vote to slash
	slashVoteIdx := int(float64(len(allOps)) * 0.7)

	for i, op := range allOps {
		vote := -1 // NO (innocent)
		if i < slashVoteIdx {
			vote = 1 // YES (guilty)
		}
		coord.CastVote("dispute-1", op.ID, vote)
	}

	// Finalize dispute
	coord.FinalizeProposal("dispute-1")

	coord.lock.RLock()
	dpStatus := coord.Proposals["dispute-1"].Status
	yesVotes := coord.Proposals["dispute-1"].VotesYes
	coord.lock.RUnlock()

	if dpStatus == "PASSED" {
		// Slash the disputed operator
		coord.SlashStake("op-1", 50.0) // 50% slash for breach

		coord.lock.RLock()
		op1 := coord.Operators["op-1"]
		coord.lock.RUnlock()

		expectedSlash := minStake / 2
		if op1.Stake == minStake-expectedSlash {
			harness.ReportPass("dispute-resolution-execution",
				fmt.Sprintf("Dispute PASSED (%.0f votes), Operator slashed 50%%, Remaining: %d",
					float64(yesVotes), op1.Stake))
		}

		atomic.AddInt64(&coord.DisputesClosed, 1)
	}

	if coord.DisputesClosed > 0 {
		harness.ReportPass("dispute-resolution-count",
			fmt.Sprintf("Disputes resolved: %d", coord.DisputesClosed))
	}

	harness.Finalize(t)
}

// ============================================================================
// Test: Cross-Operator Coordination
// ============================================================================

// TestMultiOperator_CrossOperatorActions tests coordination across multiple operators
func TestMultiOperator_CrossOperatorActions(t *testing.T) {
	harness := NewTestHarness("MultiOp-CrossOperatorActions")
	harness.Start()

	const numOperators = 25
	const minStake = uint64(100000000)

	coord := NewOperatorCoordinator()

	// Register operators
	for i := 0; i < numOperators; i++ {
		opID := fmt.Sprintf("op-%d", i)
		stake := minStake
		coord.RegisterOperator(opID, stake)
	}

	// Simulate multiple concurrent proposals and voting
	proposalCount := atomic.Int32{}
	voteCount := atomic.Int64{}
	voteFailCount := atomic.Int64{}

	wg := sync.WaitGroup{}

	// Creator goroutines
	for p := 0; p < 5; p++ {
		wg.Add(1)
		go func(pIdx int) {
			defer wg.Done()

			proposalID := fmt.Sprintf("proposal-%d", pIdx)
			coord.CreateProposal(proposalID, fmt.Sprintf("op-%d", pIdx),
				"Cross-operator change", "POLICY_CHANGE", 0.66)
			proposalCount.Add(1)
		}(p)
	}

	// Voter goroutines
	for v := 0; v < numOperators; v++ {
		wg.Add(1)
		go func(vIdx int) {
			defer wg.Done()

			for p := 0; p < 5; p++ {
				proposalID := fmt.Sprintf("proposal-%d", p)
				vote := 1 // YES
				if vIdx%3 == 0 {
					vote = -1 // NO
				} else if vIdx%5 == 0 {
					vote = 0 // ABSTAIN
				}

				if err := coord.CastVote(proposalID, fmt.Sprintf("op-%d", vIdx), vote); err == nil {
					voteCount.Add(1)
				} else {
					voteFailCount.Add(1)
				}
			}
		}(v)
	}

	wg.Wait()

	// Finalize all proposals
	for p := 0; p < 5; p++ {
		coord.FinalizeProposal(fmt.Sprintf("proposal-%d", p))
	}

	coord.lock.RLock()
	auditLogSize := len(coord.AuditLog)
	coord.lock.RUnlock()

	expectedVotes := int64(5 * numOperators)
	if voteCount.Load() == expectedVotes {
		harness.ReportPass("cross-operator-coordination",
			fmt.Sprintf("Proposals: %d, Votes: %d, Audit entries: %d",
				proposalCount.Load(), voteCount.Load(), auditLogSize))
	} else {
		harness.ReportFail("cross-operator-coordination",
			fmt.Sprintf("Votes: %d/%d, Failures: %d", voteCount.Load(), expectedVotes, voteFailCount.Load()))
	}

	harness.Finalize(t)
}

// TestMultiOperator_OperatorChurn tests system resilience with operator join/leave
func TestMultiOperator_OperatorChurn(t *testing.T) {
	harness := NewTestHarness("MultiOp-OperatorChurn")
	harness.Start()

	const initialOps = 30
	const churnDuration = 30 * time.Second
	const minStake = uint64(100000000)

	coord := NewOperatorCoordinator()

	// Register initial operators
	for i := 0; i < initialOps; i++ {
		opID := fmt.Sprintf("op-%d", i)
		coord.RegisterOperator(opID, minStake)
	}

	wg := sync.WaitGroup{}
	stopChan := make(chan struct{})
	joinCount := atomic.Int32{}
	endTime := time.Now().Add(churnDuration)

	// Churn generator - operators joining and leaving
	go func() {
		opIDCounter := initialOps
		for time.Now().Before(endTime) {
			select {
			case <-stopChan:
				return
			default:
				// Random operator joins
				newOpID := fmt.Sprintf("op-%d", opIDCounter)
				if err := coord.RegisterOperator(newOpID, minStake); err == nil {
					joinCount.Add(1)
				}
				opIDCounter++

				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// Normal operations during churn
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func(wID int) {
			defer wg.Done()

			for time.Now().Before(endTime) {
				coord.lock.RLock()
				ops := make([]*Operator, 0, len(coord.Operators))
				for _, op := range coord.Operators {
					ops = append(ops, op)
				}
				numOps := len(ops)
				coord.lock.RUnlock()

				if numOps > 0 {
					// Pick random operator and simulate activity
					idx := wID % numOps
					op := ops[idx]
					op.LastAction = time.Now()
				}

				time.Sleep(50 * time.Millisecond)
			}
		}(w)
	}

	// Wait for test duration
	<-time.After(churnDuration)
	close(stopChan)
	wg.Wait()

	coord.lock.RLock()
	finalOpCount := len(coord.Operators)
	coord.lock.RUnlock()

	if finalOpCount > initialOps {
		harness.ReportPass("operator-churn-resilience",
			fmt.Sprintf("Initial: %d, Final: %d, Joins: %d, System remained operational",
				initialOps, finalOpCount, joinCount.Load()))
	}

	harness.Finalize(t)
}
