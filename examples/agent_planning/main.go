package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/liliang-cn/rago/v2/pkg/agents/planner"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Create LLM provider
	factory := providers.NewFactory()
	providerConfig, err := providers.GetProviderConfig(&cfg.Providers.ProviderConfigs)
	if err != nil {
		log.Fatal("Failed to get provider config:", err)
	}

	llmProvider, err := factory.CreateLLMProvider(ctx, providerConfig)
	if err != nil {
		log.Fatal("Failed to create LLM provider:", err)
	}

	// Set storage directory
	home, _ := os.UserHomeDir()
	storageDir := filepath.Join(home, ".rago", "agents")

	// Create planner
	agentPlanner := planner.NewAgentPlanner(llmProvider, storageDir)
	agentPlanner.SetVerbose(true)

	// Example 1: Create a simple plan
	fmt.Println("=== Example 1: Creating a Plan ===")
	goal := "Write a Python script that fetches weather data and saves it to a JSON file"

	plan, err := agentPlanner.CreatePlan(ctx, goal)
	if err != nil {
		log.Fatal("Failed to create plan:", err)
	}

	fmt.Printf("✅ Plan created: %s\n", plan.ID)
	fmt.Printf("📝 Summary: %s\n", plan.Summary)
	fmt.Printf("📊 Tasks: %d, Steps: %d\n", len(plan.Tasks), plan.TotalSteps)

	// Display tasks
	fmt.Println("\n📌 Tasks:")
	for i, task := range plan.Tasks {
		fmt.Printf("%d. %s\n", i+1, task.Name)
		fmt.Printf("   Description: %s\n", task.Description)
		fmt.Printf("   Steps: %d\n", len(task.Steps))
		for j, step := range task.Steps {
			fmt.Printf("      %d.%d %s\n", i+1, j+1, step.Description)
		}
	}

	// Example 2: Execute the plan
	fmt.Println("\n=== Example 2: Executing the Plan ===")

	executor := planner.NewPlanExecutor(agentPlanner, nil)
	executor.SetVerbose(true)

	// Note: In real usage, you'd have MCP tools available
	// For this example, we'll use generic actions
	err = executor.ExecutePlan(ctx, plan.ID)
	if err != nil {
		fmt.Printf("⚠️  Execution had issues: %v\n", err)
	}

	// Example 3: Check progress
	fmt.Println("\n=== Example 3: Checking Progress ===")

	progress, err := executor.GetPlanProgress(plan.ID)
	if err != nil {
		log.Fatal("Failed to get progress:", err)
	}

	fmt.Printf("📊 Progress: %.1f%% complete\n", progress.PercentComplete)
	fmt.Printf("✅ Completed: %d/%d steps\n", progress.CompletedSteps, progress.TotalSteps)

	for _, taskProg := range progress.TaskProgress {
		status := "⏳"
		if taskProg.Status == planner.TaskStatusCompleted {
			status = "✅"
		} else if taskProg.Status == planner.TaskStatusFailed {
			status = "❌"
		}
		fmt.Printf("%s %s: %d/%d steps\n", status, taskProg.Name,
			taskProg.CompletedSteps, taskProg.TotalSteps)
	}

	// Example 4: List all plans
	fmt.Println("\n=== Example 4: Listing All Plans ===")

	plans, err := agentPlanner.ListPlans()
	if err != nil {
		log.Fatal("Failed to list plans:", err)
	}

	for _, p := range plans {
		fmt.Printf("📋 %s: %s (Status: %s, Progress: %d/%d)\n",
			p.ID[:8], p.Goal, p.Status, p.CompletedSteps, p.TotalSteps)
	}
}
