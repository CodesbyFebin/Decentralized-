// Command line tool for operator qualification management
package main

import (
	"flag"
	"fmt"
	"os"

	"decentralized.host/pkg/operator/backup"
	"decentralized.host/pkg/operator/nodes"
	"decentralized.host/pkg/operator/qualification"
	"decentralized.host/pkg/operator/security"
	"decentralized.host/pkg/operator/sla"
	"decentralized.host/pkg/operator/stake"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "register-operator":
		handleRegisterOperator()
	case "stake":
		handleStake()
	case "nodes":
		handleNodes()
	case "sla":
		handleSLA()
	case "security":
		handleSecurity()
	case "backup":
		handleBackup()
	case "status":
		handleStatus()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleRegisterOperator() {
	fs := flag.NewFlagSet("register-operator", flag.ExitOnError)
	operatorID := fs.String("id", "", "Operator ID")
	nodeID := fs.String("node", "", "Node ID (dh1...)")
	fs.Parse(os.Args[2:])

	if *operatorID == "" || *nodeID == "" {
		fmt.Fprintf(os.Stderr, "Usage: operator-qualification register-operator -id <id> -node <nodeID>\n")
		os.Exit(1)
	}

	qm := qualification.NewQualificationManager()
	profile, err := qm.RegisterOperator(*operatorID, *nodeID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Operator registered: %s\n", profile.OperatorID)
	fmt.Printf("  Tier: %s\n", profile.CurrentTier)
	fmt.Printf("  Reputation: %d/1000\n", profile.ReputationScore)
}

func handleStake() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: operator-qualification stake <action> [options]\n")
		os.Exit(1)
	}

	action := os.Args[2]
	sm := stake.NewStakingManager()

	switch action {
	case "deposit":
		fs := flag.NewFlagSet("stake deposit", flag.ExitOnError)
		operatorID := fs.String("id", "", "Operator ID")
		amount := fs.Uint64("amount", 0, "Stake amount in uWork")
		fs.Parse(os.Args[3:])

		if *operatorID == "" || *amount == 0 {
			fmt.Fprintf(os.Stderr, "Usage: operator-qualification stake deposit -id <id> -amount <uwork>\n")
			os.Exit(1)
		}

		account, err := sm.DepositStake(*operatorID, *amount)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Stake deposited for %s\n", account.OperatorID)
		fmt.Printf("  Amount: %d uWork\n", account.StakedAmount)
		fmt.Printf("  Status: %s\n", account.Status)

	case "validate":
		fs := flag.NewFlagSet("stake validate", flag.ExitOnError)
		operatorID := fs.String("id", "", "Operator ID")
		fs.Parse(os.Args[3:])

		if *operatorID == "" {
			fmt.Fprintf(os.Stderr, "Usage: operator-qualification stake validate -id <id>\n")
			os.Exit(1)
		}

		err := sm.ValidateStake(*operatorID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Stake validation passed for %s\n", *operatorID)

	default:
		fmt.Fprintf(os.Stderr, "Unknown stake action: %s\n", action)
		os.Exit(1)
	}
}

func handleNodes() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: operator-qualification nodes <action> [options]\n")
		os.Exit(1)
	}

	action := os.Args[2]

	switch action {
	case "info":
		fs := flag.NewFlagSet("nodes info", flag.ExitOnError)
		operatorID := fs.String("id", "", "Operator ID")
		fs.Parse(os.Args[3:])

		if *operatorID == "" {
			fmt.Fprintf(os.Stderr, "Usage: operator-qualification nodes info -id <id>\n")
			os.Exit(1)
		}

		nr := nodes.NewNodeRegistry()
		status := nr.GetOperatorNodeStatus(*operatorID)

		fmt.Printf("Node Status for %s\n", status.OperatorID)
		fmt.Printf("  Total Nodes: %d\n", status.NodeCount)
		fmt.Printf("  Min Required: %d\n", nodes.MinimumNodesForReadiness)
		fmt.Printf("  Ready: %v\n", status.IsReadyNodeCount)
		fmt.Printf("  Regions: %d\n", status.RegionCount)
		fmt.Printf("  Average Uptime: %.2f%%\n", status.AverageUptime)

	default:
		fmt.Fprintf(os.Stderr, "Unknown nodes action: %s\n", action)
		os.Exit(1)
	}
}

func handleSLA() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: operator-qualification sla <action> [options]\n")
		os.Exit(1)
	}

	action := os.Args[2]
	sm := sla.NewSLAManager()

	switch action {
	case "register":
		fs := flag.NewFlagSet("sla register", flag.ExitOnError)
		operatorID := fs.String("id", "", "Operator ID")
		fs.Parse(os.Args[3:])

		if *operatorID == "" {
			fmt.Fprintf(os.Stderr, "Usage: operator-qualification sla register -id <id>\n")
			os.Exit(1)
		}

		_, err := sm.RegisterOperator(*operatorID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("SLA tracking registered for %s\n", *operatorID)
		fmt.Printf("  Uptime Target: %.0f%%\n", sla.UptimeTarget)
		fmt.Printf("  P99 Latency Target: %v\n", sla.P99LatencyTarget)

	case "status":
		fs := flag.NewFlagSet("sla status", flag.ExitOnError)
		operatorID := fs.String("id", "", "Operator ID")
		fs.Parse(os.Args[3:])

		if *operatorID == "" {
			fmt.Fprintf(os.Stderr, "Usage: operator-qualification sla status -id <id>\n")
			os.Exit(1)
		}

		metrics, err := sm.GetMetrics(*operatorID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("SLA Status for %s\n", *operatorID)
		fmt.Printf("  Compliance: %s\n", metrics.Status)
		fmt.Printf("  Uptime: %.2f%%\n", metrics.CurrentUptime)
		fmt.Printf("  P99 Latency: %v\n", metrics.P99Latency)
		fmt.Printf("  Alert Level: %s\n", metrics.AlertLevel)

	default:
		fmt.Fprintf(os.Stderr, "Unknown sla action: %s\n", action)
		os.Exit(1)
	}
}

func handleSecurity() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: operator-qualification security <action> [options]\n")
		os.Exit(1)
	}

	action := os.Args[2]

	switch action {
	case "audit":
		fs := flag.NewFlagSet("security audit", flag.ExitOnError)
		operatorID := fs.String("id", "", "Operator ID")
		nodeID := fs.String("node", "", "Node ID")
		fs.Parse(os.Args[3:])

		if *operatorID == "" || *nodeID == "" {
			fmt.Fprintf(os.Stderr, "Usage: operator-qualification security audit -id <id> -node <nodeID>\n")
			os.Exit(1)
		}

		sm := security.NewSecurityManager()
		_, err := sm.InitiateAudit(*operatorID, *nodeID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Security audit initiated for %s\n", *operatorID)

	default:
		fmt.Fprintf(os.Stderr, "Unknown security action: %s\n", action)
		os.Exit(1)
	}
}

func handleBackup() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: operator-qualification backup <action> [options]\n")
		os.Exit(1)
	}

	action := os.Args[2]

	switch action {
	case "configure":
		fs := flag.NewFlagSet("backup configure", flag.ExitOnError)
		operatorID := fs.String("id", "", "Operator ID")
		schedule := fs.String("schedule", "daily", "Backup schedule")
		retention := fs.Int("retention", 30, "Retention days")
		fs.Parse(os.Args[3:])

		if *operatorID == "" {
			fmt.Fprintf(os.Stderr, "Usage: operator-qualification backup configure -id <id> -schedule <schedule> -retention <days>\n")
			os.Exit(1)
		}

		bm := backup.NewBackupManager()
		config, err := bm.CreateConfiguration(*operatorID, *schedule, *retention)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Backup configuration created for %s\n", config.OperatorID)
		fmt.Printf("  Schedule: %s\n", config.BackupSchedule)
		fmt.Printf("  Retention: %d days\n", config.RetentionDays)
		fmt.Printf("  RTO Target: %v\n", backup.RTOTarget)
		fmt.Printf("  RPO Target: %v\n", backup.RPOTarget)

	default:
		fmt.Fprintf(os.Stderr, "Unknown backup action: %s\n", action)
		os.Exit(1)
	}
}

func handleStatus() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: operator-qualification status\n")
		os.Exit(1)
	}

	qm := qualification.NewQualificationManager()
	status := qm.GetQualificationStatus()

	fmt.Printf("=== Operator Qualification Program Status ===\n")
	fmt.Printf("Total Operators: %d\n", status.TotalOperators)
	fmt.Printf("Certified Operators: %d\n", status.CertifiedOperators)
	fmt.Printf("Ready for Mainnet: %d\n", status.OperatorsReadyForMain)
	fmt.Printf("Average Reputation: %d/1000\n", status.AverageReputation)
	fmt.Println("\nBy Tier:")
	for tier, count := range status.OperatorsByTier {
		fmt.Printf("  %s: %d\n", tier, count)
	}

	// Staking status
	sm := stake.NewStakingManager()
	stakingStatus := sm.GetStakingStatus()
	fmt.Printf("\n=== Staking Status ===\n")
	fmt.Printf("Total Staked: %d uWork\n", stakingStatus.TotalStaked)
	fmt.Printf("Active Operators: %d\n", stakingStatus.ActiveOperators)
	fmt.Printf("Slashed Operators: %d\n", stakingStatus.SlashedOperators)

	// SLA status
	slaManager := sla.NewSLAManager()
	slaSummary := slaManager.GetStatusSummary()
	fmt.Printf("\n=== SLA Compliance ===\n")
	fmt.Printf("Compliant Operators: %d\n", slaSummary.CompliantOperators)
	fmt.Printf("At Risk: %d\n", slaSummary.AtRiskOperators)
	fmt.Printf("Violated: %d\n", slaSummary.ViolatedOperators)
	fmt.Printf("Average Uptime: %.2f%%\n", slaSummary.AverageUptime)

	// Security status
	secManager := security.NewSecurityManager()
	secSummary := secManager.GetSecuritySummary()
	fmt.Printf("\n=== Security Compliance ===\n")
	fmt.Printf("Certified: %d\n", secSummary.CertifiedOperators)
	fmt.Printf("Advanced: %d\n", secSummary.AdvancedOperators)
	fmt.Printf("Standard: %d\n", secSummary.StandardOperators)
	fmt.Printf("Critical Findings: %d\n", secSummary.CriticalFindings)

	// Backup status
	backupManager := backup.NewBackupManager()
	backupSummary := backupManager.GetStatusSummary()
	fmt.Printf("\n=== Backup Status ===\n")
	fmt.Printf("Configured: %d/%d\n", backupSummary.ConfiguredOperators, backupSummary.TotalOperators)
	fmt.Printf("Ready for Production: %d\n", backupSummary.ReadyForProductionCount)
	fmt.Printf("Average RTO: %v\n", backupSummary.AverageRTO)
	fmt.Printf("Average RPO: %v\n", backupSummary.AverageRPO)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Operator Qualification Program CLI

Usage: operator-qualification <command> [options]

Commands:
  register-operator    Register a new operator
  stake               Manage operator stakes
  nodes               Manage operator nodes
  sla                 Monitor SLA compliance
  security            Manage security audits
  backup              Configure backup/failover
  status              Display overall qualification status

Run 'operator-qualification <command> -h' for more details on each command.
`)
}
