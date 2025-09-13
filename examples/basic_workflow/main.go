package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/agents/execution"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

// BasicWorkflowExample demonstrates using RAGO as a library for workflow execution
func main() {
	fmt.Println("📚 RAGO Library Usage - Basic Workflow Example")
	fmt.Println("===============================================")
	fmt.Println()

	// Step 1: Load configuration
	cfg, err := config.Load("") // Uses rago.toml from current dir or ~/.rago/
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Step 2: Initialize providers (LLM, embedder, etc.)
	ctx := context.Background()
	_, llmService, _, err := utils.InitializeProviders(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize providers: %v", err)
	}

	// Step 3: Define a workflow programmatically
	workflow := &types.WorkflowSpec{
		Steps: []types.WorkflowStep{
			{
				ID:   "step1",
				Name: "Get Current Time",
				Type: types.StepType("tool"),
				Tool: "time",
				Inputs: map[string]interface{}{
					"action": "now",
					"format": "2006-01-02 15:04:05",
				},
				Outputs: map[string]string{
					"time": "current_time",
				},
			},
			{
				ID:   "step2",
				Name: "Fetch Weather Data",
				Type: types.StepType("tool"),
				Tool: "fetch",
				Inputs: map[string]interface{}{
					"url": "https://wttr.in/San+Francisco?format=j1",
				},
				Outputs: map[string]string{
					"data": "weather_data",
				},
			},
			{
				ID:   "step3",
				Name: "Analyze with LLM",
				Type: types.StepType("tool"),
				Tool: "sequential-thinking",
				Inputs: map[string]interface{}{
					"prompt": "Current time: {{current_time}}. Analyze this weather data and provide a brief summary",
					"data":   "{{weather_data}}",
				},
				Outputs: map[string]string{
					"analysis": "weather_analysis",
				},
			},
			{
				ID:   "step4",
				Name: "Save Results",
				Type: types.StepType("tool"),
				Tool: "filesystem",
				Inputs: map[string]interface{}{
					"action":  "write",
					"path":    "/tmp/weather_report.txt",
					"content": "Weather Report\n==============\nTime: {{current_time}}\n\n{{weather_analysis}}",
				},
				Outputs: map[string]string{
					"file": "report_file",
				},
			},
		},
		Variables: map[string]interface{}{
			"location": "San Francisco",
		},
	}

	// Step 4: Create and configure workflow executor
	executor := execution.NewWorkflowExecutorV2(cfg, llmService)
	executor.SetVerbose(true) // Enable verbose output

	// Step 5: Execute the workflow
	fmt.Println("⚡ Executing workflow...")
	result, err := executor.Execute(ctx, workflow)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	// Step 6: Process results
	fmt.Println("\n✅ Workflow completed!")
	fmt.Printf("   Execution ID: %s\n", result.ExecutionID)
	fmt.Printf("   Duration: %v\n", result.Duration)
	fmt.Printf("   Status: %s\n", result.Status)

	// Display outputs
	fmt.Println("\n📊 Results:")
	if analysis, ok := result.Outputs["weather_analysis"]; ok {
		fmt.Printf("\nWeather Analysis:\n%v\n", analysis)
	}
	if file, ok := result.Outputs["report_file"]; ok {
		fmt.Printf("\nReport saved to: %v\n", file)
	}
}
