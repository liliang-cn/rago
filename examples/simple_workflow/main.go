package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/agents/execution"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/liliang-cn/rago/v2/pkg/config"
)

// SimpleWorkflowExample demonstrates a simple workflow without LLM dependency
func main() {
	fmt.Println("📚 RAGO Library Usage - Simple Workflow Example")
	fmt.Println("===============================================")
	fmt.Println("This example runs without requiring LLM service")

	// Create minimal config (no LLM required for this workflow)
	cfg := &config.Config{}

	// Define a simple workflow using only basic tools
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
				Name: "Store in Memory",
				Type: types.StepType("tool"),
				Tool: "memory",
				Inputs: map[string]interface{}{
					"action": "store",
					"key":    "last_run",
					"value":  "{{current_time}}",
				},
				Outputs: map[string]string{
					"result": "stored_time",
				},
			},
			{
				ID:   "step3",
				Name: "Create Report",
				Type: types.StepType("tool"),
				Tool: "filesystem",
				Inputs: map[string]interface{}{
					"action":  "write",
					"path":    "/tmp/simple_workflow_report.txt",
					"content": "Simple Workflow Report\n======================\n\nExecution Time: {{current_time}}\nStatus: Success\n\nThis workflow demonstrates:\n- Time retrieval\n- Memory storage\n- File writing\n\nAll operations completed successfully.",
				},
				Outputs: map[string]string{
					"file": "report_file",
				},
			},
		},
		Variables: map[string]interface{}{
			"demo": "simple",
		},
	}

	// Create workflow executor (nil for LLM since we don't need it)
	executor := execution.NewWorkflowExecutor(cfg, nil)
	executor.SetVerbose(true)

	// Execute the workflow
	fmt.Println("⚡ Executing workflow...")
	ctx := context.Background()
	result, err := executor.Execute(ctx, workflow)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	// Display results
	fmt.Println("\n✅ Workflow completed successfully!")
	fmt.Printf("   Execution ID: %s\n", result.ExecutionID)
	fmt.Printf("   Duration: %v\n", result.Duration)
	fmt.Printf("   Status: %s\n", result.Status)

	// Show outputs
	fmt.Println("\n📊 Results:")
	if time, ok := result.Outputs["current_time"]; ok {
		fmt.Printf("   Current Time: %v\n", time)
	}
	if stored, ok := result.Outputs["stored_time"]; ok {
		fmt.Printf("   Stored Value: %v\n", stored)
	}
	if file, ok := result.Outputs["report_file"]; ok {
		fmt.Printf("   Report File: %v\n", file)
		fmt.Println("\n📝 You can view the report with:")
		fmt.Printf("   cat %v\n", file)
	}
}
