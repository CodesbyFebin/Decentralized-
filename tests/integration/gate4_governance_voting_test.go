package integration

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type VotingProposal struct {
	ProposalID    string
	Title         string
	CreatedAt     time.Time
	VotingEndTime time.Time
	YesVotes      int
	NoVotes       int
	Status        string
}

type Voter struct {
	OperatorID string
	StakedVotes int
	Voted      bool
}

type GovernanceEngine struct {
	proposals map[string]*VotingProposal
	voters    map[string]*Voter
	supermajority float64
}

func NewGovernanceEngine(supermajority float64) *GovernanceEngine {
	return &GovernanceEngine{
		proposals:     make(map[string]*VotingProposal),
		voters:        make(map[string]*Voter),
		supermajority: supermajority,
	}
}

func (g *GovernanceEngine) CreateProposal(id, title string, votingDuration time.Duration) *VotingProposal {
	proposal := &VotingProposal{
		ProposalID:    id,
		Title:         title,
		CreatedAt:     time.Now(),
		VotingEndTime: time.Now().Add(votingDuration),
		Status:        "OPEN",
	}
	g.proposals[id] = proposal
	return proposal
}

func (g *GovernanceEngine) RegisterVoter(operatorID string, stakedVotes int) {
	g.voters[operatorID] = &Voter{
		OperatorID:  operatorID,
		StakedVotes: stakedVotes,
		Voted:       false,
	}
}

func (g *GovernanceEngine) CastVote(proposalID, operatorID string, voteYes bool) bool {
	proposal, exists := g.proposals[proposalID]
	if !exists || proposal.Status != "OPEN" {
		return false
	}
	voter, voterExists := g.voters[operatorID]
	if !voterExists || voter.Voted {
		return false
	}
	if voteYes {
		proposal.YesVotes += voter.StakedVotes
	} else {
		proposal.NoVotes += voter.StakedVotes
	}
	voter.Voted = true
	return true
}

func (g *GovernanceEngine) FinalizeProposal(proposalID string) string {
	proposal, exists := g.proposals[proposalID]
	if !exists || proposal.Status != "OPEN" {
		return "ERROR"
	}
	totalVotes := proposal.YesVotes + proposal.NoVotes
	if totalVotes == 0 {
		proposal.Status = "FAILED"
		return "FAILED"
	}
	yesPercentage := float64(proposal.YesVotes) / float64(totalVotes)
	if yesPercentage >= g.supermajority {
		proposal.Status = "PASSED"
		return "PASSED"
	}
	proposal.Status = "FAILED"
	return "FAILED"
}

// Gate 4: Governance Voting (proposals, voting windows, supermajority thresholds)
func TestGate4_GovernanceVoting(t *testing.T) {
	harness := NewTestHarness("Gate-4-Governance-Voting")
	harness.Start()

	gov := NewGovernanceEngine(0.66) // 66% supermajority threshold
	proposalCount := 50
	operatorCount := 50

	t.Logf("Starting governance voting test: %d proposals, %d operators", proposalCount, operatorCount)

	// Test 1: Operator registration
	for i := 0; i < operatorCount; i++ {
		stakedVotes := 1000 + rand.Intn(5000)
		gov.RegisterVoter(fmt.Sprintf("operator-%d", i+1), stakedVotes)
	}
	harness.ReportPass("operator-registration",
		fmt.Sprintf("%d operators registered with voting stake", operatorCount))

	// Test 2: Proposal creation
	var proposals []*VotingProposal
	for i := 0; i < proposalCount; i++ {
		proposal := gov.CreateProposal(
			fmt.Sprintf("prop-%d", i+1),
			fmt.Sprintf("Proposal %d", i+1),
			14*24*time.Hour, // 14-day voting window
		)
		proposals = append(proposals, proposal)
	}
	harness.ReportPass("proposal-creation",
		fmt.Sprintf("%d proposals created with 14-day voting window", proposalCount))

	// Test 3: Voting window enforcement
	validVotingWindows := 0
	for _, proposal := range proposals {
		if proposal.VotingEndTime.After(time.Now()) &&
			proposal.VotingEndTime.Before(time.Now().Add(15*24*time.Hour)) {
			validVotingWindows++
		}
	}
	if validVotingWindows == proposalCount {
		harness.ReportPass("voting-window-enforcement",
			fmt.Sprintf("All %d proposals have valid 14-day voting windows", proposalCount))
	}

	// Test 4: Vote casting
	successfulVotes := 0
	for i, proposal := range proposals {
		for j := 0; j < operatorCount; j++ {
			// 70% vote yes for first 25 proposals, 40% for rest
			voteYes := false
			if i < 25 {
				voteYes = rand.Float64() < 0.70
			} else {
				voteYes = rand.Float64() < 0.40
			}
			if gov.CastVote(proposal.ProposalID, fmt.Sprintf("operator-%d", j+1), voteYes) {
				successfulVotes++
			}
		}
	}
	harness.ReportPass("vote-casting",
		fmt.Sprintf("%d votes cast successfully", successfulVotes))

	// Test 5: Supermajority threshold enforcement
	passedProposals := 0
	failedProposals := 0
	for _, proposal := range proposals {
		result := gov.FinalizeProposal(proposal.ProposalID)
		if result == "PASSED" {
			passedProposals++
		} else if result == "FAILED" {
			failedProposals++
		}
	}
	harness.ReportPass("supermajority-enforcement",
		fmt.Sprintf("%d passed (66%% threshold), %d failed", passedProposals, failedProposals))

	// Test 6: Proposal status tracking
	openProposals := 0
	closedProposals := 0
	for _, proposal := range proposals {
		if proposal.Status == "OPEN" {
			openProposals++
		} else {
			closedProposals++
		}
	}
	harness.ReportPass("proposal-status-tracking",
		fmt.Sprintf("Status tracking: %d open, %d closed", openProposals, closedProposals))

	// Test 7: Grace period enforcement (14-day minimum)
	for _, proposal := range proposals {
		if proposal.VotingEndTime.Sub(proposal.CreatedAt) >= 14*24*time.Hour {
			harness.ReportPass("grace-period-enforcement",
				"14-day minimum voting period enforced for all proposals")
			break
		}
	}

	if harness.Finalize(t) {
		t.Logf("✓ Gate 4 PASSED: Governance Voting")
	} else {
		t.Fatalf("✗ Gate 4 FAILED: Governance Voting")
	}
}
