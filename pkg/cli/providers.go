package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"decentralized.host/pkg/providers"
)

func init() {
	reg("connect", "add provider credentials: PROVIDER --token TOKEN [--endpoint ENDPOINT]", true, cmdConnect)
	reg("graph", "build unified resource graph from all providers", true, cmdGraph)
	reg("doctor", "check health of all resources and providers", true, cmdDoctor)
	reg("migrate", "generate and show migration plan for a resource: RESOURCE_ID [--target LOCATION]", true, cmdMigrate)
}

// cmdConnect adds provider credentials to the control plane
func cmdConnect(op *Operator, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: dh connect PROVIDER --token TOKEN [--endpoint ENDPOINT]")
	}

	provider := args[0]
	fs := flag.NewFlagSet("connect", flag.ContinueOnError)
	token := fs.String("token", "", "authentication token for provider")
	endpoint := fs.String("endpoint", "", "API endpoint (if different from default)")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	if *token == "" {
		return fmt.Errorf("--token is required")
	}

	config := providers.ProviderConfig{
		Name:     provider,
		Provider: provider,
		Token:    *token,
		Endpoint: *endpoint,
	}

	payload, err := json.Marshal(config)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := op.Do("POST", "/api/v1/providers/connect", payload, &result); err != nil {
		return err
	}

	fmt.Printf("Connected to %s provider\n", provider)
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))

	return nil
}

// cmdGraph builds the unified resource graph from all configured providers
func cmdGraph(op *Operator, args []string) error {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	output := fs.String("o", "table", "output format: table, json")

	if err := fs.Parse(args); err != nil {
		return err
	}

	var graph providers.Graph
	if err := op.Do("GET", "/api/v1/providers/graph", nil, &graph); err != nil {
		return err
	}

	if *output == "json" {
		data, _ := json.MarshalIndent(graph, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Table output
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPROVIDER\tTYPE\tTRUST_DOMAIN\tLOCATION\tSTATE")

	for _, resource := range graph.Resources {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			resource.ID,
			resource.Provider,
			resource.Type,
			resource.TrustDomain,
			resource.Location,
			resource.ObservedState,
		)
	}

	w.Flush()
	fmt.Printf("\nTotal resources: %d\n", len(graph.Resources))

	return nil
}

// cmdDoctor checks health of all providers and resources
func cmdDoctor(op *Operator, args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	var result map[string]interface{}
	if err := op.Do("GET", "/api/v1/providers/health", nil, &result); err != nil {
		return err
	}

	// Print results
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "PROVIDER\tSTATUS\tMESSAGE")

	if providers, ok := result["providers"].(map[string]interface{}); ok {
		for name, status := range providers {
			if statusMap, ok := status.(map[string]interface{}); ok {
				healthy := statusMap["healthy"]
				message := statusMap["message"]
				statusStr := "✗ UNHEALTHY"
				if h, ok := healthy.(bool); ok && h {
					statusStr = "✓ HEALTHY"
				}
				fmt.Fprintf(w, "%s\t%s\t%v\n", name, statusStr, message)
			}
		}
	}

	w.Flush()

	return nil
}

// cmdMigrate generates a migration plan for a resource
func cmdMigrate(op *Operator, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: dh migrate RESOURCE_ID [--target LOCATION]")
	}

	resourceID := args[0]
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	target := fs.String("target", "owned-infrastructure", "target location for migration")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	payload, err := json.Marshal(map[string]string{
		"resource_id":      resourceID,
		"target_location": *target,
	})
	if err != nil {
		return err
	}

	var plan providers.MigrationPlan
	if err := op.Do("POST", "/api/v1/providers/migrate", payload, &plan); err != nil {
		return err
	}

	// Print migration plan
	fmt.Printf("Migration Plan for %s\n", resourceID)
	fmt.Printf("Source Provider:  %s\n", plan.SourceProvider)
	fmt.Printf("Target Location:  %s\n", plan.TargetLocation)
	fmt.Printf("Estimated Cost:   %s\n", plan.EstimatedCost)
	fmt.Printf("Estimated Time:   %v\n", plan.EstimatedTime)
	fmt.Printf("Downtime Window:  %v\n", plan.DowntimeWindow)

	if len(plan.PrerequisiteIDs) > 0 {
		fmt.Printf("Prerequisites:    %v\n", plan.PrerequisiteIDs)
	}

	fmt.Println("\nMigration Steps:")
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SEQ\tNAME\tACTION\tTIMEOUT")

	for _, step := range plan.Steps {
		fmt.Fprintf(w, "%d\t%s\t%s\t%v\n",
			step.Sequence,
			step.Name,
			step.Action,
			step.Timeout,
		)
	}

	w.Flush()

	if plan.RollbackPlan != "" {
		fmt.Printf("\nRollback Plan: %s\n", plan.RollbackPlan)
	}

	return nil
}
