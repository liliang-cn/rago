package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents"
	"github.com/liliang-cn/rago/v2/pkg/agents/tools"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
)

func main() {
	fmt.Println("🤖 RAGO Agents Module - Example Usage")
	fmt.Println("=====================================")

	// Example 1: Basic standalone usage
	if err := basicStandaloneUsage(); err != nil {
		log.Printf("Basic usage example failed: %v", err)
	}

	fmt.Println()

	// Example 2: Research agent creation
	if err := researchAgentExample(); err != nil {
		log.Printf("Research agent example failed: %v", err)
	}

	fmt.Println()

	// Example 3: Workflow automation agent
	if err := workflowAutomationExample(); err != nil {
		log.Printf("Workflow automation example failed: %v", err)
	}

	fmt.Println()

	// Example 4: MCP integration
	if err := mcpIntegrationExample(); err != nil {
		log.Printf("MCP integration example failed: %v", err)
	}
}

// basicStandaloneUsage demonstrates basic usage as a standalone library
func basicStandaloneUsage() error {
	fmt.Println("📚 Example 1: Basic Standalone Usage")
	fmt.Println("------------------------------------")

	// Initialize MCP client (using mock for example)
	mcpClient := tools.NewMockMCPClient()

	// Initialize agents manager
	config := agents.DefaultConfig()
	config.StorageBackend = "memory"
	config.MaxConcurrentExecutions = 5

	manager, err := agents.NewManager(mcpClient, config)
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	// Create a simple agent
	agent := &types.Agent{
		ID:          "simple_agent_001",
		Name:        "Simple Task Agent",
		Description: "A basic agent for demonstration",
		Type:        types.AgentTypeWorkflow,
		Config: types.AgentConfig{
			MaxConcurrentExecutions: 1,
			DefaultTimeout:          5 * time.Minute,
			EnableMetrics:           true,
			AutonomyLevel:           types.AutonomyManual,
		},
		Workflow: types.WorkflowSpec{
			Steps: []types.WorkflowStep{
				{
					ID:   "greet",
					Name: "Greeting Step",
					Type: types.StepTypeVariable,
					Inputs: map[string]interface{}{
						"message":   "Hello from RAGO Agents!",
						"timestamp": time.Now().Format(time.RFC3339),
					},
					Outputs: map[string]string{
						"message":   "greeting",
						"timestamp": "execution_time",
					},
				},
			},
		},
		Status: types.AgentStatusActive,
	}

	// Create the agent
	agentImpl, err := manager.CreateAgent(agent)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	fmt.Printf("✅ Created agent: %s (%s)\n", agentImpl.GetName(), agentImpl.GetID())

	// Execute the agent
	ctx := context.Background()
	result, err := manager.ExecuteAgent(ctx, agent.ID, map[string]interface{}{
		"user_input": "Testing the agents module",
	})

	if err != nil {
		return fmt.Errorf("failed to execute agent: %w", err)
	}

	fmt.Printf("🚀 Execution completed: %s\n", result.Status)
	fmt.Printf("⏱️  Duration: %v\n", result.Duration)
	fmt.Printf("📊 Results: %v\n", result.Results)

	return nil
}

// researchAgentExample demonstrates creating a research agent
func researchAgentExample() error {
	fmt.Println("🔬 Example 2: Research Agent")
	fmt.Println("----------------------------")

	mcpClient := tools.NewMockMCPClient()
	manager, err := agents.NewManager(mcpClient, nil)
	if err != nil {
		return err
	}

	// Use the convenience method to create a research agent
	agent, err := manager.CreateResearchAgent(
		"Document Research Assistant",
		"Analyzes documents and extracts key insights for research purposes",
	)
	if err != nil {
		return fmt.Errorf("failed to create research agent: %w", err)
	}

	fmt.Printf("✅ Created research agent: %s\n", agent.GetName())
	fmt.Printf("🔍 Agent type: %s\n", agent.GetType())

	// Execute with research-specific variables
	ctx := context.Background()
	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"document": "Sample research document content here...",
		},
		"research_focus": "key_findings",
		"output_format":  "structured_summary",
	}

	result, err := manager.ExecuteWorkflow(ctx, agent, variables)
	if err != nil {
		return fmt.Errorf("failed to execute research agent: %w", err)
	}

	fmt.Printf("🎯 Research completed in %v\n", result.Duration)
	if result.Status == types.ExecutionStatusCompleted {
		fmt.Printf("📈 Analysis results: %v\n", result.Outputs)
	}

	return nil
}

// workflowAutomationExample demonstrates workflow automation
func workflowAutomationExample() error {
	fmt.Println("⚙️  Example 3: Workflow Automation")
	fmt.Println("----------------------------------")

	mcpClient := tools.NewMockMCPClient()
	manager, err := agents.NewManager(mcpClient, nil)
	if err != nil {
		return err
	}

	// Create a multi-step workflow
	workflowSteps := []types.WorkflowStep{
		{
			ID:   "fetch_data",
			Name: "Fetch Data",
			Type: types.StepTypeTool,
			Tool: "sqlite_query",
			Inputs: map[string]interface{}{
				"query":    "SELECT * FROM users WHERE active = 1",
				"database": "./data/app.db",
			},
			Outputs: map[string]string{
				"result": "active_users",
			},
		},
		{
			ID:   "process_users",
			Name: "Process User Data",
			Type: types.StepTypeVariable,
			Inputs: map[string]interface{}{
				"users_data":      "{{active_users}}",
				"processing_date": time.Now().Format("2006-01-02"),
			},
			Outputs: map[string]string{
				"users_data":      "processed_users",
				"processing_date": "last_processed",
			},
		},
		{
			ID:   "generate_report",
			Name: "Generate Report",
			Type: types.StepTypeTool,
			Tool: "file_write",
			Inputs: map[string]interface{}{
				"path":    "./reports/user_report_{{last_processed}}.json",
				"content": "{{processed_users}}",
			},
			Outputs: map[string]string{
				"path": "report_path",
			},
		},
	}

	agent, err := manager.CreateWorkflowAgent(
		"User Data Processing Pipeline",
		"Automated pipeline for processing user data and generating reports",
		workflowSteps,
	)
	if err != nil {
		return fmt.Errorf("failed to create workflow agent: %w", err)
	}

	fmt.Printf("✅ Created workflow agent: %s\n", agent.GetName())
	fmt.Printf("📝 Steps configured: %d\n", len(workflowSteps))

	// Execute the workflow
	ctx := context.Background()
	result, err := manager.ExecuteWorkflow(ctx, agent, map[string]interface{}{
		"batch_id":    "batch_001",
		"environment": "production",
	})

	if err != nil {
		return fmt.Errorf("failed to execute workflow: %w", err)
	}

	fmt.Printf("🔄 Workflow completed: %s\n", result.Status)
	fmt.Printf("📊 Steps executed: %d\n", len(result.StepResults))
	for i, step := range result.StepResults {
		fmt.Printf("   %d. %s: %s (%v)\n", i+1, step.Name, step.Status, step.Duration)
	}

	return nil
}

// mcpIntegrationExample demonstrates MCP tool integration
func mcpIntegrationExample() error {
	fmt.Println("🔗 Example 4: MCP Tool Integration")
	fmt.Println("----------------------------------")

	// Create mock MCP client with custom results
	mcpClient := tools.NewMockMCPClient()
	mcpClient.SetMockResult("web_search", map[string]interface{}{
		"results": []map[string]interface{}{
			{
				"title":   "RAGO Documentation",
				"url":     "https://github.com/liliang-cn/rago",
				"snippet": "Advanced RAG system with MCP integration",
			},
		},
		"count": 1,
	})

	// Create MCP tool executor
	toolExecutor := tools.NewMCPToolExecutor(mcpClient)

	// List available tools
	ctx := context.Background()
	tools_list, err := toolExecutor.ListAvailableTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to list MCP tools: %w", err)
	}

	fmt.Printf("🛠️  Available MCP tools: %d\n", len(tools_list))
	for _, tool := range tools_list {
		fmt.Printf("   - %s: %s\n", tool.Name, tool.Description)
	}

	// Execute a web search tool
	searchInputs := map[string]interface{}{
		"query": "RAGO agents workflow automation",
		"limit": 5,
	}

	fmt.Printf("\n🔍 Executing web search with query: %s\n", searchInputs["query"])

	searchResult, err := toolExecutor.ExecuteTool(ctx, "web_search", searchInputs)
	if err != nil {
		return fmt.Errorf("failed to execute web search: %w", err)
	}

	fmt.Printf("✅ Search completed successfully in %v\n", searchResult.Duration)

	// Pretty print the result
	if searchResult.Result != nil {
		resultJSON, _ := json.MarshalIndent(searchResult.Result, "", "  ")
		fmt.Printf("📋 Search results:\n%s\n", string(resultJSON))
	}

	// Demonstrate SQLite query tool
	fmt.Printf("\n💾 Executing SQLite query\n")
	queryInputs := map[string]interface{}{
		"query":    "SELECT name, description FROM agents WHERE type = 'research'",
		"database": "./agents.db",
	}

	queryResult, err := toolExecutor.ExecuteTool(ctx, "sqlite_query", queryInputs)
	if err != nil {
		return fmt.Errorf("failed to execute SQLite query: %w", err)
	}

	fmt.Printf("✅ Query completed in %v\n", queryResult.Duration)
	if queryResult.Result != nil {
		resultJSON, _ := json.MarshalIndent(queryResult.Result, "", "  ")
		fmt.Printf("📋 Query results:\n%s\n", string(resultJSON))
	}

	return nil
}
