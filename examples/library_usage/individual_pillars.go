// This example demonstrates using individual RAGO pillars independently.
package main

import (
	"context"
	"fmt"
	"log"
	"time"
	
	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

func main() {
	// Example 1: Using only the LLM pillar
	fmt.Println("=== Example 1: LLM-Only Client ===")
	llmOnlyExample()
	
	// Example 2: Using only the RAG pillar
	fmt.Println("\n=== Example 2: RAG-Only Client ===")
	ragOnlyExample()
	
	// Example 3: Using only the MCP pillar
	fmt.Println("\n=== Example 3: MCP-Only Client ===")
	mcpOnlyExample()
	
	// Example 4: Using only the Agent pillar
	fmt.Println("\n=== Example 4: Agent-Only Client ===")
	agentOnlyExample()
	
	// Example 5: Custom combination of pillars
	fmt.Println("\n=== Example 5: Custom Pillar Combination ===")
	customCombinationExample()
}

func llmOnlyExample() {
	// Create LLM-only client
	config := core.LLMConfig{
		DefaultProvider: "ollama",
		Providers: map[string]core.ProviderConfig{
			"ollama": {
				Type:    "ollama",
				BaseURL: "http://localhost:11434",
				Model:   "llama3.2",
				Weight:  1,
				Timeout: 30 * time.Second,
			},
		},
		LoadBalancing: core.LoadBalancingConfig{
			Strategy:      "round_robin",
			HealthCheck:   true,
			CheckInterval: 30 * time.Second,
		},
	}
	
	llmClient, err := client.NewLLMClient(config)
	if err != nil {
		log.Printf("Failed to create LLM client: %v", err)
		return
	}
	defer llmClient.Close()
	
	ctx := context.Background()
	
	// Generate text
	req := core.GenerateRequest{
		Prompt:      "Write a haiku about programming",
		Temperature: 0.9,
		MaxTokens:   50,
	}
	
	resp, err := llmClient.Generate(ctx, req)
	if err != nil {
		log.Printf("Generation failed: %v", err)
		return
	}
	
	fmt.Printf("Generated Haiku:\n%s\n", resp.Text)
	
	// Stream generation
	streamReq := core.StreamRequest{
		Prompt:      "List 3 benefits of Go programming",
		Temperature: 0.7,
		MaxTokens:   150,
	}
	
	fmt.Println("\nStreaming response:")
	err = llmClient.Stream(ctx, streamReq, func(chunk string) error {
		fmt.Print(chunk)
		return nil
	})
	fmt.Println()
	
	if err != nil {
		log.Printf("Streaming failed: %v", err)
	}
}

func ragOnlyExample() {
	// Create RAG-only client
	config := core.RAGConfig{
		StorageBackend: "dual",
		ChunkingStrategy: core.ChunkingConfig{
			Strategy:     "recursive",
			ChunkSize:    300,
			ChunkOverlap: 50,
		},
		VectorStore: core.VectorStoreConfig{
			Backend:   "sqvect",
			Metric:    "cosine",
			IndexType: "hnsw",
		},
		Search: core.SearchConfig{
			DefaultLimit:     10,
			MaxLimit:        100,
			DefaultThreshold: 0.7,
		},
		Embedding: core.EmbeddingConfig{
			Provider:  "ollama",
			Model:     "nomic-embed-text",
			BatchSize: 32,
		},
	}
	
	ragClient, err := client.NewRAGClient(config)
	if err != nil {
		log.Printf("Failed to create RAG client: %v", err)
		return
	}
	defer ragClient.Close()
	
	ctx := context.Background()
	
	// Ingest documents
	documents := []struct {
		id      string
		content string
	}{
		{
			id: "go-concurrency",
			content: `Go's concurrency model is based on goroutines and channels. 
			Goroutines are lightweight threads managed by the Go runtime. 
			Channels are the pipes that connect concurrent goroutines.`,
		},
		{
			id: "go-interfaces",
			content: `Interfaces in Go provide a way to specify the behavior of an object. 
			A type implements an interface by implementing its methods. 
			Go's interfaces are satisfied implicitly.`,
		},
	}
	
	for _, doc := range documents {
		req := core.IngestRequest{
			DocumentID: doc.id,
			Content:    doc.content,
			Metadata: map[string]interface{}{
				"topic": "go-programming",
			},
		}
		
		resp, err := ragClient.Ingest(ctx, req)
		if err != nil {
			log.Printf("Failed to ingest %s: %v", doc.id, err)
			continue
		}
		
		fmt.Printf("Ingested %s: %d chunks created\n", doc.id, resp.ChunkCount)
	}
	
	// Search for relevant content
	searchReq := core.SearchRequest{
		Query: "How do goroutines work?",
		Limit: 5,
	}
	
	searchResp, err := ragClient.Search(ctx, searchReq)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}
	
	fmt.Printf("\nSearch Results for '%s':\n", searchReq.Query)
	for i, result := range searchResp.Results {
		fmt.Printf("%d. Score: %.2f - %s...\n", i+1, result.Score, result.Content[:50])
	}
	
	// Hybrid search (vector + keyword)
	hybridReq := core.HybridSearchRequest{
		Query:         "interface methods",
		VectorWeight:  0.6,
		KeywordWeight: 0.4,
		Limit:         5,
	}
	
	hybridResp, err := ragClient.HybridSearch(ctx, hybridReq)
	if err != nil {
		log.Printf("Hybrid search failed: %v", err)
		return
	}
	
	fmt.Printf("\nHybrid Search Results:\n")
	for i, result := range hybridResp.Results {
		fmt.Printf("%d. Combined Score: %.2f\n", i+1, result.Score)
	}
}

func mcpOnlyExample() {
	// Create MCP-only client
	config := core.MCPConfig{
		ServersPath: "~/.rago/mcpServers.json",
		HealthCheck: core.HealthCheckConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
		},
		ToolExecution: core.ToolExecutionConfig{
			MaxConcurrent:  5,
			DefaultTimeout: 30 * time.Second,
			EnableCache:    true,
			CacheTTL:       5 * time.Minute,
		},
	}
	
	mcpClient, err := client.NewMCPClient(config)
	if err != nil {
		log.Printf("Failed to create MCP client: %v", err)
		return
	}
	defer mcpClient.Close()
	
	ctx := context.Background()
	
	// List available tools
	toolsResp, err := mcpClient.ListTools(ctx)
	if err != nil {
		log.Printf("Failed to list tools: %v", err)
		return
	}
	
	fmt.Printf("Available MCP Tools:\n")
	for _, tool := range toolsResp.Tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}
	
	// Call a specific tool (example: filesystem read)
	toolReq := core.ToolRequest{
		Name: "filesystem_read",
		Parameters: map[string]interface{}{
			"path": "/tmp/example.txt",
		},
	}
	
	toolResp, err := mcpClient.CallTool(ctx, toolReq)
	if err != nil {
		// Tool might not exist or file might not exist - that's okay for this example
		fmt.Printf("Tool call note: %v\n", err)
	} else {
		fmt.Printf("\nTool Result: %v\n", toolResp.Result)
	}
	
	// Execute multiple tools concurrently
	toolRequests := []core.ToolRequest{
		{
			Name: "calculator",
			Parameters: map[string]interface{}{
				"expression": "2 + 2",
			},
		},
		{
			Name: "web_fetch",
			Parameters: map[string]interface{}{
				"url": "https://example.com",
			},
		},
	}
	
	results, err := mcpClient.CallToolsConcurrent(ctx, toolRequests)
	if err != nil {
		log.Printf("Concurrent tool execution failed: %v", err)
		return
	}
	
	fmt.Printf("\nConcurrent Tool Results:\n")
	for tool, result := range results {
		fmt.Printf("  %s: %v\n", tool, result)
	}
}

func agentOnlyExample() {
	// Create Agent-only client
	config := core.AgentsConfig{
		WorkflowEngine: core.WorkflowEngineConfig{
			MaxSteps:       50,
			StepTimeout:    2 * time.Minute,
			StateBackend:   "memory",
			EnableRecovery: true,
		},
		Scheduling: core.SchedulingConfig{
			Backend:       "memory",
			MaxConcurrent: 5,
			QueueSize:     100,
		},
		ReasoningChains: core.ReasoningChainsConfig{
			MaxSteps:      20,
			MaxMemorySize: 5000,
			StepTimeout:   30 * time.Second,
		},
	}
	
	agentClient, err := client.NewAgentClient(config)
	if err != nil {
		log.Printf("Failed to create Agent client: %v", err)
		return
	}
	defer agentClient.Close()
	
	ctx := context.Background()
	
	// Create an agent
	createReq := core.CreateAgentRequest{
		Name:         "task_planner",
		Instructions: "You are a task planning agent that breaks down complex tasks into steps",
		Capabilities: []string{"planning", "reasoning"},
	}
	
	agent, err := agentClient.CreateAgent(ctx, createReq)
	if err != nil {
		log.Printf("Failed to create agent: %v", err)
		return
	}
	
	fmt.Printf("Created Agent: %s (ID: %s)\n", agent.Name, agent.ID)
	
	// Execute a task with the agent
	execReq := core.AgentExecuteRequest{
		AgentID: agent.ID,
		Task:    "Plan a simple web application deployment",
		Context: map[string]interface{}{
			"platform": "kubernetes",
			"services": []string{"frontend", "backend", "database"},
		},
	}
	
	execResp, err := agentClient.Execute(ctx, execReq)
	if err != nil {
		log.Printf("Agent execution failed: %v", err)
		return
	}
	
	fmt.Printf("\nAgent Execution Results:\n")
	fmt.Printf("Status: %s\n", execResp.Status)
	fmt.Printf("Steps executed: %d\n", len(execResp.Steps))
	for i, step := range execResp.Steps {
		fmt.Printf("  %d. %s - %s\n", i+1, step.Name, step.Status)
	}
	
	// Create and execute a workflow
	workflowReq := core.WorkflowRequest{
		Name: "data_processing",
		Input: map[string]interface{}{
			"data_source": "api",
			"format":      "json",
		},
	}
	
	workflowResp, err := agentClient.ExecuteWorkflow(ctx, workflowReq)
	if err != nil {
		log.Printf("Workflow execution failed: %v", err)
		return
	}
	
	fmt.Printf("\nWorkflow Execution:\n")
	fmt.Printf("Steps executed: %d\n", workflowResp.StepsExecuted)
	fmt.Printf("Duration: %v\n", workflowResp.Duration)
	fmt.Printf("Output: %v\n", workflowResp.Output)
	
	// Clean up
	err = agentClient.DeleteAgent(ctx, agent.ID)
	if err != nil {
		log.Printf("Failed to delete agent: %v", err)
	}
}

func customCombinationExample() {
	// Create a custom client with specific pillars using the builder pattern
	client, err := client.NewBuilder().
		WithLLM(core.LLMConfig{
			DefaultProvider: "ollama",
			Providers: map[string]core.ProviderConfig{
				"ollama": {
					Type:    "ollama",
					BaseURL: "http://localhost:11434",
					Model:   "llama3.2",
					Weight:  1,
					Timeout: 30 * time.Second,
				},
			},
		}).
		WithRAG(core.RAGConfig{
			StorageBackend: "dual",
			ChunkingStrategy: core.ChunkingConfig{
				Strategy:  "fixed",
				ChunkSize: 500,
			},
		}).
		WithoutMCP().    // Explicitly disable MCP
		WithoutAgents(). // Explicitly disable Agents
		WithDataDir("/tmp/rago-custom").
		WithLogLevel("debug").
		Build()
	
	if err != nil {
		log.Fatalf("Failed to build custom client: %v", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	
	// Use the custom client with only LLM and RAG
	fmt.Println("Custom Client Configuration:")
	health := client.Health()
	
	for pillar, status := range health.Pillars {
		fmt.Printf("  %s: %s\n", pillar, status)
	}
	
	// Demonstrate usage with available pillars
	if client.LLM() != nil {
		req := core.GenerateRequest{
			Prompt:      "What is RAGO?",
			Temperature: 0.5,
			MaxTokens:   100,
		}
		
		resp, err := client.LLM().Generate(ctx, req)
		if err != nil {
			log.Printf("Generation failed: %v", err)
		} else {
			fmt.Printf("\nLLM Response: %s\n", resp.Text)
		}
	}
	
	if client.RAG() != nil {
		// RAG operations available
		fmt.Println("RAG pillar is available for use")
	}
	
	if client.MCP() == nil {
		fmt.Println("MCP pillar is disabled as configured")
	}
	
	if client.Agents() == nil {
		fmt.Println("Agent pillar is disabled as configured")
	}
}