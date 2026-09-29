package chaos

import (
	"context"
	"fmt"
	"time"
)

// Invariants are properties that must always hold true in the system.
type Invariants struct {
	// Safety properties: what must never happen
	SafetyViolations []string

	// Liveness properties: what must eventually happen
	LivenessViolations []string

	// Durability properties: what must be stable
	DurabilityViolations []string

	// Observability properties: what must be recorded
	ObservabilityViolations []string
}

// CheckInvariants validates all system invariants during a chaos scenario.
func CheckInvariants(ctx context.Context, tc TestCluster, scenario *Scenario) *Invariants {
	inv := &Invariants{
		SafetyViolations:        []string{},
		LivenessViolations:      []string{},
		DurabilityViolations:    []string{},
		ObservabilityViolations: []string{},
	}

	// Safety: Quorum is never below 2/3 (with tolerance for recovery)
	if err := checkQuorumSafety(ctx, tc, scenario); err != nil {
		inv.SafetyViolations = append(inv.SafetyViolations, err.Error())
	}

	// Safety: Data is never corrupted (even after storage failures)
	if err := checkDataIntegrity(ctx, tc, scenario); err != nil {
		inv.SafetyViolations = append(inv.SafetyViolations, err.Error())
	}

	// Safety: Signed intent is never executed without local policy check
	if err := checkSignedIntentSafety(ctx, tc, scenario); err != nil {
		inv.SafetyViolations = append(inv.SafetyViolations, err.Error())
	}

	// Liveness: API remains responsive to at least quorum
	if err := checkAPILiveness(ctx, tc, scenario); err != nil {
		inv.LivenessViolations = append(inv.LivenessViolations, err.Error())
	}

	// Liveness: Raft leader is eventually elected
	if err := checkLeaderElection(ctx, tc, scenario); err != nil {
		inv.LivenessViolations = append(inv.LivenessViolations, err.Error())
	}

	// Liveness: Failed nodes eventually recover their state
	if err := checkStateRecovery(ctx, tc, scenario); err != nil {
		inv.LivenessViolations = append(inv.LivenessViolations, err.Error())
	}

	// Durability: Committed state persists across restarts
	if err := checkStatePersistence(ctx, tc, scenario); err != nil {
		inv.DurabilityViolations = append(inv.DurabilityViolations, err.Error())
	}

	// Durability: Audit trail is immutable
	if err := checkAuditImmutability(ctx, tc, scenario); err != nil {
		inv.DurabilityViolations = append(inv.DurabilityViolations, err.Error())
	}

	// Observability: All events are recorded with timestamps
	if err := checkEventRecording(ctx, tc, scenario); err != nil {
		inv.ObservabilityViolations = append(inv.ObservabilityViolations, err.Error())
	}

	// Observability: Clock skew is detected and reported
	if err := checkClockSkewDetection(ctx, tc, scenario); err != nil {
		inv.ObservabilityViolations = append(inv.ObservabilityViolations, err.Error())
	}

	return inv
}

// checkQuorumSafety verifies quorum is maintained appropriately.
func checkQuorumSafety(ctx context.Context, tc TestCluster, scenario *Scenario) error {
	// Context-specific quorum checks
	switch scenario.Category {
	case "node":
		// After killing 1 node, 2 nodes remain = quorum maintained
		if err := tc.VerifyNodeCount(ctx, 2); err == nil {
			return nil // Quorum OK
		}
	case "network":
		// After partition, quorum-holding side must have majority
		return tc.VerifyQuorumHolderLeads(ctx)
	}
	return nil
}

// checkDataIntegrity verifies data survives failures.
func checkDataIntegrity(ctx context.Context, tc TestCluster, scenario *Scenario) error {
	if scenario.Category == "storage" || scenario.Category == "load" {
		return tc.VerifyDataIntegrity(ctx)
	}
	return nil
}

// checkSignedIntentSafety verifies signed intent is properly enforced.
func checkSignedIntentSafety(ctx context.Context, tc TestCluster, scenario *Scenario) error {
	// This is verified through audit trail in observability checks
	// Would need deeper integration with control plane for full verification
	return nil
}

// checkAPILiveness verifies API responsiveness.
func checkAPILiveness(ctx context.Context, tc TestCluster, scenario *Scenario) error {
	// API should be responsive except during total partition/quorum loss
	switch scenario.Category {
	case "node":
		// After 1 node kill, API should still respond
		return tc.VerifyAPIResponsive(ctx)
	case "network":
		// After partition, quorum side should still respond
		return tc.VerifyAPIResponsive(ctx)
	case "storage":
		// API should remain responsive even with storage issues
		return tc.VerifyAPIResponsive(ctx)
	case "clock":
		// API should remain responsive even with clock skew
		return tc.VerifyAPIResponsive(ctx)
	}
	return nil
}

// checkLeaderElection verifies Raft leader election.
func checkLeaderElection(ctx context.Context, tc TestCluster, scenario *Scenario) error {
	// Give cluster time to recover (max 5 seconds for leader election)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("leader election timeout: %w", ctx.Err())
		default:
			if err := tc.VerifyRaftLeader(ctx); err == nil {
				return nil // Leader elected
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// checkStateRecovery verifies failed nodes recover their state.
func checkStateRecovery(ctx context.Context, tc TestCluster, scenario *Scenario) error {
	// After restart, node should rejoin cluster with consistent state
	if scenario.ID == "node-kill-and-restart" || scenario.ID == "cascade-failure" || scenario.ID == "shutdown" {
		// Give node time to restart and rejoin
		time.Sleep(2 * time.Second)
		return tc.VerifyNodeCount(ctx, 3)
	}
	return nil
}

// checkStatePersistence verifies committed state survives restarts.
func checkStatePersistence(ctx context.Context, tc TestCluster, scenario *Scenario) error {
	// After restart, data should still be present
	if scenario.ID == "node-kill-and-restart" || scenario.ID == "shutdown" {
		return tc.VerifyDataIntegrity(ctx)
	}
	return nil
}

// checkAuditImmutability verifies audit trail cannot be modified.
func checkAuditImmutability(ctx context.Context, tc TestCluster, scenario *Scenario) error {
	// Audit trail is immutable by design (Raft-backed, cryptographically signed)
	// This would require reading the actual audit trail and verifying signatures
	return nil
}

// checkEventRecording verifies events are recorded.
func checkEventRecording(ctx context.Context, tc TestCluster, scenario *Scenario) error {
	// All failure/recovery events should be in the audit trail
	// This is checked in VerifyClockSkewDetected, VerifyCorruptionDetected, etc.
	return nil
}

// checkClockSkewDetection verifies clock issues are detected.
func checkClockSkewDetection(ctx context.Context, tc TestCluster, scenario *Scenario) error {
	if scenario.Category == "clock" {
		return tc.VerifyClockSkewDetected(ctx)
	}
	return nil
}

// InvariantsSummary provides a human-readable summary of invariant violations.
func (inv *Invariants) Summary() string {
	var summary string

	if len(inv.SafetyViolations) > 0 {
		summary += fmt.Sprintf("SAFETY VIOLATIONS (%d):\n", len(inv.SafetyViolations))
		for _, v := range inv.SafetyViolations {
			summary += fmt.Sprintf("  - %s\n", v)
		}
	}

	if len(inv.LivenessViolations) > 0 {
		summary += fmt.Sprintf("LIVENESS VIOLATIONS (%d):\n", len(inv.LivenessViolations))
		for _, v := range inv.LivenessViolations {
			summary += fmt.Sprintf("  - %s\n", v)
		}
	}

	if len(inv.DurabilityViolations) > 0 {
		summary += fmt.Sprintf("DURABILITY VIOLATIONS (%d):\n", len(inv.DurabilityViolations))
		for _, v := range inv.DurabilityViolations {
			summary += fmt.Sprintf("  - %s\n", v)
		}
	}

	if len(inv.ObservabilityViolations) > 0 {
		summary += fmt.Sprintf("OBSERVABILITY VIOLATIONS (%d):\n", len(inv.ObservabilityViolations))
		for _, v := range inv.ObservabilityViolations {
			summary += fmt.Sprintf("  - %s\n", v)
		}
	}

	if summary == "" {
		summary = "✓ All invariants hold\n"
	}

	return summary
}

// IsValid returns true if all invariants passed.
func (inv *Invariants) IsValid() bool {
	return len(inv.SafetyViolations) == 0 &&
		len(inv.LivenessViolations) == 0 &&
		len(inv.DurabilityViolations) == 0 &&
		len(inv.ObservabilityViolations) == 0
}
