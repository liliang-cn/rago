package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents/execution"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/utils"
)

// AsyncWorkflowExample demonstrates async workflow execution with parallel steps
func main() {
	fmt.Println("⚡ RAGO Library Usage - Async Workflow Execution")
	fmt.Println("================================================")
	fmt.Println()
	fmt.Println("This example shows parallel execution of independent steps")
	fmt.Println("and dependency resolution for sequential steps.")

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize providers
	ctx := context.Background()
	_, llmService, _, err := utils.InitializeProviders(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize providers: %v", err)
	}

	// Define a workflow with parallel and dependent steps
	workflow := &types.WorkflowSpec{
		Steps: []types.WorkflowStep{
			// These three steps can run in parallel (no dependencies)
			{
				ID:   "fetch_github",
				Name: "Fetch GitHub Data",
				Type: types.StepType("tool"),
				Tool: "fetch",
				Inputs: map[string]interface{}{
					"url": "https://api.github.com/repos/golang/go",
					"headers": map[string]interface{}{
						"Accept": "application/vnd.github.v3+json",
					},
				},
				Outputs: map[string]string{
					"data": "github_data",
				},
			},
			{
				ID:   "fetch_weather",
				Name: "Fetch Weather Data",
				Type: types.StepType("tool"),
				Tool: "fetch",
				Inputs: map[string]interface{}{
					"url": "https://wttr.in/London?format=j1",
				},
				Outputs: map[string]string{
					"data": "weather_data",
				},
			},
			{
				ID:   "get_time",
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
			// This step depends on the GitHub fetch
			{
				ID:        "analyze_github",
				Name:      "Analyze GitHub Repository",
				Type:      types.StepType("tool"),
				Tool:      "sequential-thinking",
				DependsOn: []string{"fetch_github"}, // Explicit dependency
				Inputs: map[string]interface{}{
					"prompt": "Analyze this GitHub repository data and provide key statistics",
					"data":   "{{github_data}}", // Uses data from fetch_github
				},
				Outputs: map[string]string{
					"analysis": "github_analysis",
				},
			},
			// This step depends on the weather fetch
			{
				ID:        "analyze_weather",
				Name:      "Analyze Weather",
				Type:      types.StepType("tool"),
				Tool:      "sequential-thinking",
				DependsOn: []string{"fetch_weather"}, // Explicit dependency
				Inputs: map[string]interface{}{
					"prompt": "Analyze this weather data and provide a brief forecast",
					"data":   "{{weather_data}}", // Uses data from fetch_weather
				},
				Outputs: map[string]string{
					"analysis": "weather_analysis",
				},
			},
			// Final step depends on all analyses
			{
				ID:        "create_report",
				Name:      "Create Combined Report",
				Type:      types.StepType("tool"),
				Tool:      "sequential-thinking",
				DependsOn: []string{"analyze_github", "analyze_weather", "get_time"},
				Inputs: map[string]interface{}{
					"prompt": `Create a brief daily report combining:
					1. Time: {{current_time}}
					2. GitHub Analysis: {{github_analysis}}
					3. Weather Analysis: {{weather_analysis}}
					
					Format it as a concise daily briefing.`,
				},
				Outputs: map[string]string{
					"report": "daily_report",
				},
			},
			// Save the report (depends on report creation)
			{
				ID:        "save_report",
				Name:      "Save Report to File",
				Type:      types.StepType("tool"),
				Tool:      "filesystem",
				DependsOn: []string{"create_report"},
				Inputs: map[string]interface{}{
					"action": "write",
					"path":   fmt.Sprintf("/tmp/daily_report_%d.txt", time.Now().Unix()),
					"content": `Daily Briefing
==============
Generated: {{current_time}}

{{daily_report}}`,
				},
				Outputs: map[string]string{
					"file": "report_file",
				},
			},
		},
		Variables: make(map[string]interface{}),
	}

	// Use the V2 executor for async execution
	fmt.Println("📊 Workflow Structure:")
	fmt.Println("   • 3 parallel fetch operations (GitHub, Weather, Time)")
	fmt.Println("   • 2 parallel analyses (after their respective fetches)")
	fmt.Println("   • 1 report generation (after all analyses)")
	fmt.Println("   • 1 file save (after report)")
	fmt.Println()

	executor := execution.NewWorkflowExecutorV2(cfg, llmService)
	executor.SetVerbose(true)

	fmt.Println("⚙️  Starting async workflow execution...")
	fmt.Println("   Watch for parallel step execution!")

	_ = time.Now() // startTime - removed unused variable
	result, err := executor.Execute(ctx, workflow)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	// Display results
	fmt.Println("\n✅ Workflow completed!")
	fmt.Printf("   Total Duration: %v\n", result.Duration)
	fmt.Printf("   Execution ID: %s\n", result.ExecutionID)
	fmt.Printf("   Status: %s\n", result.Status)

	// Show execution timeline
	fmt.Println("\n⏱️  Execution Timeline:")
	for _, stepResult := range result.StepResults {
		fmt.Printf("   • %s: %v\n", stepResult.StepID, stepResult.Duration)
	}

	// Display the final report
	fmt.Println("\n📄 Generated Report:")
	if report, ok := result.Outputs["daily_report"]; ok {
		fmt.Println("---")
		fmt.Printf("%v\n", report)
		fmt.Println("---")
	}

	if file, ok := result.Outputs["report_file"]; ok {
		fmt.Printf("\n💾 Report saved to: %v\n", file)
	}

	fmt.Printf("\n🚀 Async execution saved approximately %.2f seconds compared to sequential!\n",
		calculateTimeSaved(result))
}

// calculateTimeSaved estimates time saved by parallel execution
func calculateTimeSaved(result *types.ExecutionResult) float64 {
	// In sequential execution, all steps would run one after another
	// In parallel execution, independent steps run simultaneously
	// This is a rough estimate based on the workflow structure

	// Assume each fetch takes ~1 second, analyses take ~2 seconds
	sequentialEstimate := 3.0 + 2.0 + 2.0 + 1.0 + 0.5 // All steps sequential
	actualTime := result.Duration.Seconds()

	saved := sequentialEstimate - actualTime
	if saved < 0 {
		return 0
	}
	return saved
}
