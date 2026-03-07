package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/liliang-cn/agent-go/pkg/agent"
	"github.com/liliang-cn/agent-go/pkg/config"
	"github.com/liliang-cn/agent-go/pkg/pool"
)

func main() {
	ctx := context.Background()

	// 1. Load configuration (from agentgo.toml or environment)
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println("=== Building Agent with PTC and Tool Search ===")
	builder := agent.New("PTCSearchAgent").
		WithPTC(). // Enable Programmatic Tool Calling
		WithConfig(cfg)

	// Tool 1: Weather (Deferred)
	weatherTool := agent.BuildTool("check_climate_status").
		Description("Get the current weather for a specific location. Use this to check weather conditions.").
		Param("location", agent.TypeString, "City name, e.g., 'Tokyo'").
		Deferred(true).
		Handler(func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			loc := args["location"].(string)
			return fmt.Sprintf("The climate in %s is snowy and -2°C.", loc), nil
		}).
		Build()

	// Tool 2: Expense Calculator (Deferred)
	expenseTool := agent.BuildTool("calculate_expenses").
		Description("Calculate business travel expenses. Use this for accounting.").
		Param("employee_id", agent.TypeString, "Employee ID").
		Deferred(true).
		Handler(func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			empID := args["employee_id"].(string)
			return fmt.Sprintf("Total expenses for employee %s: $1,250.00", empID), nil
		}).
		Build()

	agentSvc, err := builder.
		WithTools(weatherTool, expenseTool).
		Build()

	if err != nil {
		log.Fatalf("Failed to build agent: %v", err)
	}

	fmt.Println("\n=== Starting Query ===")
	query := "Write a PTC script that uses `searchAndCallTool` to calculate the expenses for employee 'EMP123', then uses it again to check the climate in 'Hokkaido'. Return the combined results object."

	fmt.Printf("User: %s\n\n", query)

	agentSvc.SetSessionID("ptc-test-session")
	resp, err := agentSvc.ChatWithPTC(ctx, query)
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}

	fmt.Printf("\n=== Final Response ===\n")
	if resp.PTCUsed && resp.PTCResult.ExecutionResult != nil {
		fmt.Println("Code Executed:")
		fmt.Println(resp.PTCResult.Code)
		fmt.Println("\nLogs:")
		fmt.Println(strings.Join(resp.PTCResult.ExecutionResult.Logs, "\n"))
		fmt.Println("\nReturn Value:")
		fmt.Println(resp.PTCResult.ExecutionResult.ReturnValue)
		if resp.PTCResult.ExecutionResult.Error != "" {
			fmt.Println("\nError:", resp.PTCResult.ExecutionResult.Error)
		}
	} else {
		fmt.Println(resp.ExecutionResult.Text())
	}
}
