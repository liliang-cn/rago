package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/liliang-cn/rago/v2/pkg/agents"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

func main() {
	// Change to project root if running from examples directory
	if _, err := os.Stat("../../mcpServers.json"); err == nil {
		os.Chdir("../..")
	}

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()

	// Initialize LLM provider
	factory := providers.NewFactory()
	providerConfig, err := providers.GetProviderConfig(&cfg.Providers.ProviderConfigs)
	if err != nil {
		log.Fatalf("Failed to get provider config: %v", err)
	}

	llmService, err := factory.CreateLLMProvider(ctx, providerConfig)
	if err != nil {
		log.Fatalf("Failed to initialize LLM service: %v", err)
	}

	// Initialize MCP manager
	var mcpManager *mcp.Manager
	if cfg.MCP.Servers != nil && len(cfg.MCP.Servers) > 0 {
		mcpManager = mcp.NewManager(&cfg.MCP)
		
		// Start essential servers
		fmt.Println("Starting MCP servers...")
		if _, err := mcpManager.StartServer(ctx, "filesystem"); err != nil {
			fmt.Printf("Warning: filesystem server failed to start: %v\n", err)
		}
		if _, err := mcpManager.StartServer(ctx, "memory"); err != nil {
			fmt.Printf("Warning: memory server failed to start: %v\n", err)
		}
		fmt.Println("MCP servers ready")
	}

	// Create the agent
	agent := agents.NewAgent(cfg, llmService, mcpManager)
	agent.SetVerbose(true)

	// Example 1: Plan and Execute in one go
	fmt.Println("\n=== Example 1: Plan and Execute ===")
	result, err := agent.PlanAndExecute(ctx, "list all files in the current directory")
	if err != nil {
		log.Printf("Failed to plan and execute: %v", err)
	} else {
		fmt.Printf("Success! Plan saved to: %s\n", result.PlanFile)
		fmt.Printf("Results: %v\n", result.Results)
	}

	// Example 2: Plan only (for review)
	fmt.Println("\n=== Example 2: Plan Only ===")
	planFile, err := agent.PlanOnly(ctx, "count the number of Go files in the project")
	if err != nil {
		log.Printf("Failed to create plan: %v", err)
	} else {
		fmt.Printf("Plan saved to: %s\n", planFile)
		
		// Read and display the plan
		plan, err := agent.GetPlan(planFile)
		if err == nil {
			fmt.Printf("Goal: %s\n", plan.Goal)
			fmt.Printf("Steps: %d\n", len(plan.Steps))
			for _, step := range plan.Steps {
				fmt.Printf("  Step %d: %s - %s\n", step.StepNumber, step.Tool, step.Description)
			}
		}
	}

	// Example 3: Execute a saved plan
	fmt.Println("\n=== Example 3: Execute Saved Plan ===")
	if planFile != "" {
		results, err := agent.ExecuteOnly(ctx, planFile)
		if err != nil {
			log.Printf("Failed to execute plan: %v", err)
		} else {
			fmt.Printf("Execution successful!\n")
			for key, value := range results {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}
	}

	// Example 4: List all saved plans
	fmt.Println("\n=== Example 4: List Saved Plans ===")
	plans, err := agent.ListPlans()
	if err != nil {
		log.Printf("Failed to list plans: %v", err)
	} else {
		fmt.Printf("Found %d saved plans:\n", len(plans))
		for _, plan := range plans {
			fmt.Printf("  - %s (%.2f KB, modified: %s)\n", 
				plan.Filename, 
				float64(plan.Size)/1024, 
				plan.Modified.Format("2006-01-02 15:04:05"))
		}
	}

	// Example 5: Delete old plans (optional)
	fmt.Println("\n=== Example 5: Cleanup Old Plans ===")
	if len(plans) > 5 {
		// Delete the oldest plan as an example
		oldestPlan := plans[0]
		for _, p := range plans {
			if p.Modified.Before(oldestPlan.Modified) {
				oldestPlan = p
			}
		}
		
		err := agent.DeletePlan(oldestPlan.Filename)
		if err != nil {
			log.Printf("Failed to delete plan: %v", err)
		} else {
			fmt.Printf("Deleted old plan: %s\n", oldestPlan.Filename)
		}
	}
}

// Helper function to demonstrate error handling
func handleAgentError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}