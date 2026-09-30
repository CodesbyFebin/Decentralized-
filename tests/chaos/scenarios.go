// Package chaos defines M7 chaos validation scenarios.
//
// 17 defined scenarios validate system behavior under real failure modes:
// - Process crashes and recovery
// - Network partitions and recovery
// - Storage failures (corruption, full disk)
// - Clock skew and synchronization
// - Cascading failures
// - Load and stress conditions
//
// Each scenario:
// 1. Injects a specific failure
// 2. Verifies system observes and responds
// 3. Validates recovery/resilience
// 4. Confirms invariants hold
package chaos

import (
	"context"
	"fmt"
	"time"
)

// Scenario represents one chaos test scenario.
type Scenario struct {
	ID          string        // Unique identifier (e.g., "node-kill-1")
	Title       string        // Human-readable title
	Description string        // What this scenario tests
	Category    string        // Category: node | network | storage | clock | load | recovery
	Duration    time.Duration // How long to run the scenario
	Setup       func(ctx context.Context, tc TestCluster) error
	Inject      func(ctx context.Context, tc TestCluster) error
	Verify      func(ctx context.Context, tc TestCluster) error
	Cleanup     func(ctx context.Context, tc TestCluster) error
}

// AllScenarios returns all 17 defined chaos scenarios.
func AllScenarios() []*Scenario {
	return []*Scenario{
		// Node failure scenarios (4 scenarios)
		ScenarioNodeKillSingleProvider(),
		ScenarioNodeKillMultipleProviders(),
		ScenarioNodeKillAndRestart(),
		ScenarioNodeGracefulShutdown(),

		// Network partition scenarios (3 scenarios)
		ScenarioNetworkPartitionSingleNode(),
		ScenarioNetworkPartitionHalf(),
		ScenarioNetworkPartitionRecovery(),

		// Storage failure scenarios (3 scenarios)
		ScenarioStorageCorruption(),
		ScenarioStorageFull(),
		ScenarioStorageRecovery(),

		// Clock and timing scenarios (2 scenarios)
		ScenarioClockSkew(),
		ScenarioClockJump(),

		// Cascading and compound failures (3 scenarios)
		ScenarioCascadingNodeFailure(),
		ScenarioSimultaneousNetworkAndNodeFailure(),
		ScenarioStorageFailureUnderLoad(),

		// Load and stress scenarios (2 scenarios)
		ScenarioSustainedQueryLoad(),
		ScenarioShutdown(),
	}
}

// Node failure scenarios

// ScenarioNodeKillSingleProvider: Kill one provider node, verify cluster remains operational.
func ScenarioNodeKillSingleProvider() *Scenario {
	return &Scenario{
		ID:          "node-kill-1",
		Title:       "Single Provider Node Termination",
		Description: "Kill one provider node; cluster must remain operational and elect new leader if needed",
		Category:    "node",
		Duration:    30 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			// Verify all 3 nodes are running
			return tc.VerifyNodeCount(ctx, 3)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			// Kill provider node 1 (leaving 2 providers + 1 edge)
			return tc.KillNode(ctx, "provider-1")
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			// Cluster should remain responsive
			if err := tc.VerifyNodeCount(ctx, 2); err != nil {
				return fmt.Errorf("node count after kill: %w", err)
			}
			// API should still respond
			if err := tc.VerifyAPIResponsive(ctx); err != nil {
				return fmt.Errorf("API responsive after node kill: %w", err)
			}
			// Raft should have elected a new leader (or maintained if still leader)
			if err := tc.VerifyRaftLeader(ctx); err != nil {
				return fmt.Errorf("Raft leader after node kill: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return tc.RestartNode(ctx, "provider-1")
		},
	}
}

// ScenarioNodeKillMultipleProviders: Kill 2 of 3 providers, cluster loses quorum.
func ScenarioNodeKillMultipleProviders() *Scenario {
	return &Scenario{
		ID:          "node-kill-2",
		Title:       "Multiple Provider Node Termination (Quorum Loss)",
		Description: "Kill 2 of 3 providers; cluster loses Raft quorum, remaining node isolated",
		Category:    "node",
		Duration:    30 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyNodeCount(ctx, 3)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			if err := tc.KillNode(ctx, "provider-1"); err != nil {
				return err
			}
			if err := tc.KillNode(ctx, "provider-2"); err != nil {
				return err
			}
			return nil
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			// Only 1 provider + 1 edge left
			if err := tc.VerifyNodeCount(ctx, 2); err != nil {
				return fmt.Errorf("node count after quorum loss: %w", err)
			}
			// Remaining provider should detect quorum lost
			if err := tc.VerifyRaftQuorumLost(ctx); err != nil {
				return fmt.Errorf("Raft quorum detection: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			if err := tc.RestartNode(ctx, "provider-1"); err != nil {
				return err
			}
			return tc.RestartNode(ctx, "provider-2")
		},
	}
}

// ScenarioNodeKillAndRestart: Kill node and verify recovery.
func ScenarioNodeKillAndRestart() *Scenario {
	return &Scenario{
		ID:          "node-kill-3",
		Title:       "Node Kill and Restart Recovery",
		Description: "Kill node and restart; verify re-integration with cluster",
		Category:    "node",
		Duration:    45 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyNodeCount(ctx, 3)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			if err := tc.KillNode(ctx, "provider-2"); err != nil {
				return err
			}
			time.Sleep(5 * time.Second)
			return tc.RestartNode(ctx, "provider-2")
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			// Verify node restarted and rejoined
			if err := tc.VerifyNodeCount(ctx, 3); err != nil {
				return fmt.Errorf("node count after restart: %w", err)
			}
			// Verify Raft consensus
			if err := tc.VerifyRaftConsensus(ctx); err != nil {
				return fmt.Errorf("Raft consensus after restart: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return nil
		},
	}
}

// ScenarioNodeGracefulShutdown: Graceful shutdown of node.
func ScenarioNodeGracefulShutdown() *Scenario {
	return &Scenario{
		ID:          "node-shutdown-1",
		Title:       "Graceful Node Shutdown",
		Description: "Gracefully shutdown node; cluster should handle cleanly without disruption",
		Category:    "node",
		Duration:    30 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyNodeCount(ctx, 3)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			return tc.GracefulShutdown(ctx, "edge-1")
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			if err := tc.VerifyNodeCount(ctx, 2); err != nil {
				return fmt.Errorf("node count after shutdown: %w", err)
			}
			if err := tc.VerifyAPIResponsive(ctx); err != nil {
				return fmt.Errorf("API responsive after shutdown: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return tc.RestartNode(ctx, "edge-1")
		},
	}
}

// Network partition scenarios

// ScenarioNetworkPartitionSingleNode: Isolate one node from network.
func ScenarioNetworkPartitionSingleNode() *Scenario {
	return &Scenario{
		ID:          "network-partition-1",
		Title:       "Single Node Network Partition",
		Description: "Isolate single node from cluster; verify detection and recovery",
		Category:    "network",
		Duration:    30 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyNetworkConnectivity(ctx)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			return tc.PartitionNode(ctx, "provider-2")
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			// Isolated node should detect partition
			if err := tc.VerifyNodePartitioned(ctx, "provider-2"); err != nil {
				return fmt.Errorf("partition detection: %w", err)
			}
			// Other nodes should remain connected
			if err := tc.VerifyNetworkConnectivity(ctx); err != nil {
				return fmt.Errorf("cluster connectivity: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return tc.HealPartition(ctx, "provider-2")
		},
	}
}

// ScenarioNetworkPartitionHalf: Split cluster in half (2 vs 1).
func ScenarioNetworkPartitionHalf() *Scenario {
	return &Scenario{
		ID:          "network-partition-2",
		Title:       "Cluster Network Partition (Split Brain)",
		Description: "Split cluster 2 vs 1; verify quorum holder is authoritative",
		Category:    "network",
		Duration:    30 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyNetworkConnectivity(ctx)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			// Partition providers 1,2 from provider 3
			return tc.PartitionNodes(ctx, []string{"provider-1", "provider-2"}, []string{"provider-3"})
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			// Partition should be detected
			if err := tc.VerifyPartitionDetected(ctx); err != nil {
				return fmt.Errorf("partition detection: %w", err)
			}
			// Quorum holder (2/3) should maintain leadership
			if err := tc.VerifyQuorumHolderLeads(ctx); err != nil {
				return fmt.Errorf("quorum holder leadership: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return tc.HealPartition(ctx, "provider-3")
		},
	}
}

// ScenarioNetworkPartitionRecovery: Partition and recovery.
func ScenarioNetworkPartitionRecovery() *Scenario {
	return &Scenario{
		ID:          "network-partition-3",
		Title:       "Network Partition Recovery",
		Description: "Partition cluster and recover; verify converged state",
		Category:    "network",
		Duration:    45 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyNetworkConnectivity(ctx)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			if err := tc.PartitionNode(ctx, "provider-1"); err != nil {
				return err
			}
			time.Sleep(10 * time.Second)
			return tc.HealPartition(ctx, "provider-1")
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			// Network should be healed
			if err := tc.VerifyNetworkConnectivity(ctx); err != nil {
				return fmt.Errorf("network healed: %w", err)
			}
			// Nodes should converge
			if err := tc.VerifyRaftConsensus(ctx); err != nil {
				return fmt.Errorf("Raft converged: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return nil
		},
	}
}

// Storage failure scenarios

// ScenarioStorageCorruption: Corrupt storage chunk and verify detection.
func ScenarioStorageCorruption() *Scenario {
	return &Scenario{
		ID:          "storage-corrupt-1",
		Title:       "Storage Corruption Detection",
		Description: "Corrupt storage chunk; verify system detects and handles gracefully",
		Category:    "storage",
		Duration:    30 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyStorageHealthy(ctx)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			return tc.InjectStorageCorruption(ctx, "provider-1")
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			if err := tc.VerifyCorruptionDetected(ctx); err != nil {
				return fmt.Errorf("corruption detection: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return tc.RepairStorage(ctx, "provider-1")
		},
	}
}

// ScenarioStorageFull: Inject disk full condition.
func ScenarioStorageFull() *Scenario {
	return &Scenario{
		ID:          "storage-full-1",
		Title:       "Disk Full Condition",
		Description: "Simulate disk full; verify graceful degradation and recovery",
		Category:    "storage",
		Duration:    30 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyStorageHealthy(ctx)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			return tc.InjectStorageFull(ctx, "provider-2")
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			if err := tc.VerifyStorageFullDetected(ctx); err != nil {
				return fmt.Errorf("disk full detection: %w", err)
			}
			// System should remain operational (no crash)
			if err := tc.VerifyAPIResponsive(ctx); err != nil {
				return fmt.Errorf("API responsive during disk full: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return tc.ClearStorageFull(ctx, "provider-2")
		},
	}
}

// ScenarioStorageRecovery: Storage failure and recovery.
func ScenarioStorageRecovery() *Scenario {
	return &Scenario{
		ID:          "storage-recovery-1",
		Title:       "Storage Failure Recovery",
		Description: "Storage error and recovery; verify data integrity",
		Category:    "storage",
		Duration:    45 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyStorageHealthy(ctx)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			if err := tc.InjectStorageError(ctx, "provider-3"); err != nil {
				return err
			}
			time.Sleep(5 * time.Second)
			return tc.RepairStorage(ctx, "provider-3")
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			if err := tc.VerifyStorageHealthy(ctx); err != nil {
				return fmt.Errorf("storage recovered: %w", err)
			}
			if err := tc.VerifyDataIntegrity(ctx); err != nil {
				return fmt.Errorf("data integrity: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return nil
		},
	}
}

// Clock and timing scenarios

// ScenarioClockSkew: Simulate clock offset on one node.
func ScenarioClockSkew() *Scenario {
	return &Scenario{
		ID:          "clock-skew-1",
		Title:       "Clock Skew Detection",
		Description: "Introduce clock offset; verify detection and adaptation",
		Category:    "clock",
		Duration:    30 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyClockSynchronized(ctx)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			return tc.OffsetClock(ctx, "provider-1", 5*time.Second)
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			if err := tc.VerifyClockSkewDetected(ctx); err != nil {
				return fmt.Errorf("clock skew detection: %w", err)
			}
			// System should tolerate ±5s skew
			if err := tc.VerifyAPIResponsive(ctx); err != nil {
				return fmt.Errorf("API responsive with clock skew: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return tc.ResetClock(ctx, "provider-1")
		},
	}
}

// ScenarioClockJump: Simulate large clock jump.
func ScenarioClockJump() *Scenario {
	return &Scenario{
		ID:          "clock-jump-1",
		Title:       "Large Clock Jump",
		Description: "Jump clock backward; verify system doesn't break",
		Category:    "clock",
		Duration:    30 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyClockSynchronized(ctx)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			return tc.OffsetClock(ctx, "provider-2", -30*time.Second)
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			// System should handle large clock jumps gracefully
			if err := tc.VerifyAPIResponsive(ctx); err != nil {
				return fmt.Errorf("API responsive after clock jump: %w", err)
			}
			// Timestamps should monotonically increase (no negative time)
			if err := tc.VerifyMonotonicTimestamps(ctx); err != nil {
				return fmt.Errorf("monotonic timestamps: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return tc.ResetClock(ctx, "provider-2")
		},
	}
}

// Cascading and compound failure scenarios

// ScenarioCascadingNodeFailure: Rapid sequential node failures.
func ScenarioCascadingNodeFailure() *Scenario {
	return &Scenario{
		ID:          "cascading-1",
		Title:       "Cascading Node Failures",
		Description: "Rapid sequential failures; verify system survives",
		Category:    "recovery",
		Duration:    60 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyNodeCount(ctx, 3)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			// Kill provider-1
			if err := tc.KillNode(ctx, "provider-1"); err != nil {
				return err
			}
			time.Sleep(5 * time.Second)
			// Kill provider-2
			if err := tc.KillNode(ctx, "provider-2"); err != nil {
				return err
			}
			time.Sleep(10 * time.Second)
			// Restart both
			if err := tc.RestartNode(ctx, "provider-1"); err != nil {
				return err
			}
			time.Sleep(5 * time.Second)
			return tc.RestartNode(ctx, "provider-2")
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			if err := tc.VerifyNodeCount(ctx, 3); err != nil {
				return fmt.Errorf("node count after cascading recovery: %w", err)
			}
			if err := tc.VerifyRaftConsensus(ctx); err != nil {
				return fmt.Errorf("Raft consensus after cascading: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return nil
		},
	}
}

// ScenarioSimultaneousNetworkAndNodeFailure: Network partition + node failure.
func ScenarioSimultaneousNetworkAndNodeFailure() *Scenario {
	return &Scenario{
		ID:          "compound-1",
		Title:       "Simultaneous Network and Node Failure",
		Description: "Network partition while node fails; verify resilience",
		Category:    "recovery",
		Duration:    45 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyNetworkConnectivity(ctx)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			if err := tc.PartitionNode(ctx, "provider-1"); err != nil {
				return err
			}
			time.Sleep(2 * time.Second)
			if err := tc.KillNode(ctx, "provider-2"); err != nil {
				return err
			}
			time.Sleep(5 * time.Second)
			// Heal partition
			if err := tc.HealPartition(ctx, "provider-1"); err != nil {
				return err
			}
			// Restart node
			return tc.RestartNode(ctx, "provider-2")
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			if err := tc.VerifyNetworkConnectivity(ctx); err != nil {
				return fmt.Errorf("network connectivity: %w", err)
			}
			if err := tc.VerifyNodeCount(ctx, 3); err != nil {
				return fmt.Errorf("node count: %w", err)
			}
			if err := tc.VerifyRaftConsensus(ctx); err != nil {
				return fmt.Errorf("Raft consensus: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return nil
		},
	}
}

// ScenarioStorageFailureUnderLoad: Storage failure while load is high.
func ScenarioStorageFailureUnderLoad() *Scenario {
	return &Scenario{
		ID:          "storage-load-1",
		Title:       "Storage Failure Under Load",
		Description: "Storage error during sustained load; verify system doesn't lose data",
		Category:    "recovery",
		Duration:    60 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyStorageHealthy(ctx)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			// Start sustained query load
			if err := tc.StartLoadGenerator(ctx, 100); err != nil {
				return err
			}
			time.Sleep(10 * time.Second)
			// Inject storage error during load
			if err := tc.InjectStorageError(ctx, "provider-1"); err != nil {
				return err
			}
			time.Sleep(15 * time.Second)
			// Repair
			if err := tc.RepairStorage(ctx, "provider-1"); err != nil {
				return err
			}
			// Stop load
			return tc.StopLoadGenerator(ctx)
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			if err := tc.VerifyStorageHealthy(ctx); err != nil {
				return fmt.Errorf("storage recovered: %w", err)
			}
			if err := tc.VerifyDataIntegrity(ctx); err != nil {
				return fmt.Errorf("data integrity under load: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return nil
		},
	}
}

// Load and stress scenarios

// ScenarioSustainedQueryLoad: High sustained query load.
func ScenarioSustainedQueryLoad() *Scenario {
	return &Scenario{
		ID:          "load-1",
		Title:       "Sustained Query Load",
		Description: "High sustained query load; verify performance and reliability",
		Category:    "load",
		Duration:    60 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyAPIResponsive(ctx)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			return tc.StartLoadGenerator(ctx, 1000) // 1000 req/s
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			// API should remain responsive
			if err := tc.VerifyAPIResponsive(ctx); err != nil {
				return fmt.Errorf("API responsive under load: %w", err)
			}
			// Latency should be acceptable
			if err := tc.VerifyLatencyAcceptable(ctx, 500*time.Millisecond); err != nil {
				return fmt.Errorf("latency under load: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return tc.StopLoadGenerator(ctx)
		},
	}
}

// ScenarioShutdown: Graceful cluster shutdown and recovery.
func ScenarioShutdown() *Scenario {
	return &Scenario{
		ID:          "shutdown-1",
		Title:       "Graceful Cluster Shutdown",
		Description: "Gracefully shutdown entire cluster; verify recovery on restart",
		Category:    "load",
		Duration:    60 * time.Second,
		Setup: func(ctx context.Context, tc TestCluster) error {
			return tc.VerifyNodeCount(ctx, 3)
		},
		Inject: func(ctx context.Context, tc TestCluster) error {
			// Gracefully shutdown all nodes
			if err := tc.GracefulShutdown(ctx, "provider-1"); err != nil {
				return err
			}
			if err := tc.GracefulShutdown(ctx, "provider-2"); err != nil {
				return err
			}
			if err := tc.GracefulShutdown(ctx, "provider-3"); err != nil {
				return err
			}
			time.Sleep(5 * time.Second)
			// Restart all
			if err := tc.RestartNode(ctx, "provider-1"); err != nil {
				return err
			}
			if err := tc.RestartNode(ctx, "provider-2"); err != nil {
				return err
			}
			return tc.RestartNode(ctx, "provider-3")
		},
		Verify: func(ctx context.Context, tc TestCluster) error {
			// All nodes should be back
			if err := tc.VerifyNodeCount(ctx, 3); err != nil {
				return fmt.Errorf("node count after restart: %w", err)
			}
			// Cluster should be operational
			if err := tc.VerifyAPIResponsive(ctx); err != nil {
				return fmt.Errorf("API responsive after full restart: %w", err)
			}
			// Raft should have consensus
			if err := tc.VerifyRaftConsensus(ctx); err != nil {
				return fmt.Errorf("Raft consensus: %w", err)
			}
			return nil
		},
		Cleanup: func(ctx context.Context, tc TestCluster) error {
			return nil
		},
	}
}
