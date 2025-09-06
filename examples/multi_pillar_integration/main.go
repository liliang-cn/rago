// Package main demonstrates complete four-pillar integration in RAGO v3.
// This example shows how LLM, RAG, MCP, and Agent pillars work together
// to create a powerful AI foundation.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents"
	"github.com/liliang-cn/rago/v2/pkg/agents/scheduler"
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/rag"
)

func main() {
	fmt.Println("=== RAGO V3 Multi-Pillar Integration Demo ===")
	fmt.Println()
	
	// Initialize configuration
	config := &core.Config{
		DataDir:  "/tmp/rago",
		LogLevel: "info",
		
		LLM: core.LLMConfig{
			DefaultProvider: "ollama",
			LoadBalancing: core.LoadBalancingConfig{
				Strategy:      "round_robin",
				HealthCheck:   true,
				CheckInterval: 30 * time.Second,
			},
			Providers: map[string]core.ProviderConfig{
				"ollama": {
					Type:    "ollama",
					BaseURL: "http://localhost:11434",
					Model:   "llama3.2",
					Weight:  10,
					Timeout: 30 * time.Second,
				},
			},
		},
		
		RAG: core.RAGConfig{
			StorageBackend: "sqvect",
			ChunkingStrategy: core.ChunkingConfig{
				Strategy:     "recursive",
				ChunkSize:    1000,
				ChunkOverlap: 100,
				MinChunkSize: 100,
			},
			VectorStore: core.VectorStoreConfig{
				Backend:    "sqvect",
				Dimensions: 1536,
				Metric:     "cosine",
			},
			Search: core.SearchConfig{
				DefaultLimit:     10,
				MaxLimit:         100,
				DefaultThreshold: 0.7,
			},
		},
		
		MCP: core.MCPConfig{
			ServersPath: "mcpServers.json",
			HealthCheckInterval: 1 * time.Minute,
			CacheSize: 100,
			CacheTTL: 5 * time.Minute,
		},
		
		Agents: core.AgentsConfig{
			WorkflowEngine: core.WorkflowEngineConfig{
				MaxConcurrentWorkflows: 10,
				MaxConcurrentSteps:     20,
				DefaultTimeout:         30 * time.Minute,
				EnableParallelism:      true,
			},
			Scheduling: core.SchedulingConfig{
				MaxConcurrentJobs: 5,
				RetryPolicy: core.RetryPolicy{
					MaxRetries:    3,
					RetryDelay:    5 * time.Second,
					BackoffFactor: 2.0,
				},
			},
			ReasoningChains: core.ReasoningChainsConfig{
				MaxDepth:       10,
				MemoryCapacity: 1000,
				EnableLearning: true,
			},
		},
	}
	
	ctx := context.Background()
	
	// === PILLAR 1: LLM Service ===
	fmt.Println("1. Initializing LLM Pillar...")
	llmService, err := llm.NewService(config.LLM)
	if err != nil {
		log.Fatalf("Failed to initialize LLM service: %v", err)
	}
	defer llmService.Close()
	
	// Demonstrate LLM capability
	fmt.Println("   - Testing LLM generation...")
	llmResponse, err := llmService.Generate(ctx, core.GenerationRequest{
		Prompt:      "What is the purpose of RAGO's four-pillar architecture?",
		MaxTokens:   100,
		Temperature: 0.7,
	})
	if err != nil {
		fmt.Printf("   LLM generation error: %v\n", err)
	} else {
		fmt.Printf("   LLM Response: %s\n", truncate(llmResponse.Content, 100))
	}
	fmt.Println()
	
	// === PILLAR 2: RAG Service ===
	fmt.Println("2. Initializing RAG Pillar...")
	// Note: RAG service requires embedder - using nil for demo
	ragService, err := rag.NewService(nil, nil)
	if err != nil {
		log.Fatalf("Failed to initialize RAG service: %v", err)
	}
	defer ragService.Close()
	
	// Demonstrate RAG capability
	fmt.Println("   - Ingesting sample document...")
	_, err = ragService.IngestDocument(ctx, core.IngestRequest{
		DocumentID: "doc1",
		Content: `RAGO is a local-first, privacy-first Go AI foundation that provides 
		four core pillars: LLM for language models, RAG for retrieval-augmented generation,
		MCP for tool integration, and Agents for workflow orchestration. Each pillar can 
		work independently or together to create powerful AI applications.`,
		ContentType: "text/plain",
		Metadata: map[string]interface{}{
			"source": "documentation",
			"topic":  "architecture",
		},
	})
	if err != nil {
		fmt.Printf("   RAG ingestion error: %v\n", err)
	} else {
		fmt.Println("   Document ingested successfully")
	}
	
	fmt.Println("   - Searching knowledge base...")
	searchResult, err := ragService.Search(ctx, core.SearchRequest{
		Query: "What are RAGO's pillars?",
		Limit: 5,
	})
	if err != nil {
		fmt.Printf("   RAG search error: %v\n", err)
	} else {
		fmt.Printf("   Found %d relevant chunks\n", len(searchResult.Chunks))
	}
	fmt.Println()
	
	// === PILLAR 3: MCP Service ===
	fmt.Println("3. Initializing MCP Pillar...")
	mcpService, err := mcp.NewService(config.MCP)
	if err != nil {
		log.Fatalf("Failed to initialize MCP service: %v", err)
	}
	defer mcpService.Close()
	
	// Demonstrate MCP capability
	fmt.Println("   - Listing available tools...")
	tools := mcpService.ListTools()
	fmt.Printf("   Available tools: %d\n", len(tools))
	for i, tool := range tools {
		if i < 3 { // Show first 3 tools
			fmt.Printf("     - %s: %s\n", tool.Name, tool.Description)
		}
	}
	fmt.Println()
	
	// === PILLAR 4: Agent Service ===
	fmt.Println("4. Initializing Agent Pillar...")
	agentService, err := agents.NewService(
		config.Agents,
		agents.WithLLMService(llmService),
		agents.WithRAGService(ragService),
		agents.WithMCPService(mcpService),
	)
	if err != nil {
		log.Fatalf("Failed to initialize Agent service: %v", err)
	}
	defer agentService.Close()
	
	// Create a research agent
	fmt.Println("   - Creating research agent...")
	err = agentService.CreateAgent(core.AgentDefinition{
		Name:        "research_agent",
		Type:        "research",
		Description: "Agent that performs research using RAG and LLM",
		Instructions: "Use RAG to find relevant information and LLM to analyze it",
		Tools:       []string{"rag_search", "llm_analyze"},
	})
	if err != nil {
		fmt.Printf("   Agent creation error: %v\n", err)
	} else {
		fmt.Println("   Research agent created successfully")
	}
	
	// Create a workflow
	fmt.Println("   - Creating multi-pillar workflow...")
	err = agentService.CreateWorkflow(core.WorkflowDefinition{
		Name:        "document_analysis",
		Description: "Analyze documents using all four pillars",
		Steps: []core.WorkflowStep{
			{
				ID:   "step1",
				Name: "Retrieve Context",
				Type: "rag",
				Parameters: map[string]interface{}{
					"query": "Find relevant documents",
				},
			},
			{
				ID:   "step2",
				Name: "Analyze Content",
				Type: "llm",
				Parameters: map[string]interface{}{
					"prompt": "Analyze the retrieved content",
				},
				Dependencies: []string{"step1"},
			},
			{
				ID:   "step3",
				Name: "Execute Tools",
				Type: "mcp",
				Parameters: map[string]interface{}{
					"tool": "filesystem",
					"action": "save_results",
				},
				Dependencies: []string{"step2"},
			},
		},
		Inputs: []core.WorkflowInput{
			{
				Name:     "document",
				Type:     "string",
				Required: true,
			},
		},
		Outputs: []core.WorkflowOutput{
			{
				Name: "analysis",
				Type: "object",
			},
		},
	})
	if err != nil {
		fmt.Printf("   Workflow creation error: %v\n", err)
	} else {
		fmt.Println("   Multi-pillar workflow created successfully")
	}
	
	// === DEMONSTRATION: Multi-Pillar Collaboration ===
	fmt.Println("\n=== Demonstrating Multi-Pillar Collaboration ===")
	
	// Execute agent with task
	fmt.Println("\n5. Executing Research Agent...")
	agentResult, err := agentService.ExecuteAgent(ctx, core.AgentRequest{
		AgentName: "research_agent",
		Task:      "Research RAGO's four-pillar architecture and explain how they work together",
		MaxSteps:  5,
	})
	if err != nil {
		fmt.Printf("   Agent execution error: %v\n", err)
	} else {
		fmt.Printf("   Agent Status: %s\n", agentResult.Status)
		fmt.Printf("   Steps Executed: %d\n", len(agentResult.Steps))
		for i, step := range agentResult.Steps {
			fmt.Printf("     Step %d: %s (%.2fs)\n", 
				step.StepNumber, step.Action, step.Duration.Seconds())
		}
		fmt.Printf("   Result: %s\n", truncate(agentResult.Result, 200))
	}
	
	// Execute workflow
	fmt.Println("\n6. Executing Multi-Pillar Workflow...")
	workflowResult, err := agentService.ExecuteWorkflow(ctx, core.WorkflowRequest{
		WorkflowName: "document_analysis",
		Inputs: map[string]interface{}{
			"document": "Sample document about RAGO architecture",
		},
	})
	if err != nil {
		fmt.Printf("   Workflow execution error: %v\n", err)
	} else {
		fmt.Printf("   Workflow Status: %s\n", workflowResult.Status)
		fmt.Printf("   Duration: %.2fs\n", workflowResult.Duration.Seconds())
		fmt.Printf("   Steps Completed: %d\n", len(workflowResult.Steps))
		for _, step := range workflowResult.Steps {
			fmt.Printf("     - %s: %s (%.2fs)\n", 
				step.StepID, step.Status, step.Duration.Seconds())
		}
	}
	
	// Schedule a workflow
	fmt.Println("\n7. Scheduling Workflow for Periodic Execution...")
	err = agentService.ScheduleWorkflow("document_analysis", core.ScheduleConfig{
		Type:       "interval",
		Expression: "5m", // Every 5 minutes
		Timezone:   "UTC",
	})
	if err != nil {
		fmt.Printf("   Scheduling error: %v\n", err)
	} else {
		fmt.Println("   Workflow scheduled successfully")
		
		// List scheduled tasks
		tasks := agentService.GetScheduledTasks()
		fmt.Printf("   Scheduled Tasks: %d\n", len(tasks))
		for _, task := range tasks {
			fmt.Printf("     - %s: Next run at %s\n", 
				task.WorkflowName, task.NextRun.Format(time.RFC3339))
		}
	}
	
	// === SUMMARY ===
	fmt.Println("\n=== Summary ===")
	fmt.Println("✓ LLM Pillar: Provides intelligent text generation and analysis")
	fmt.Println("✓ RAG Pillar: Enables knowledge retrieval and context augmentation")
	fmt.Println("✓ MCP Pillar: Integrates external tools and services")
	fmt.Println("✓ Agent Pillar: Orchestrates complex workflows using all pillars")
	fmt.Println()
	fmt.Println("The four pillars work together seamlessly:")
	fmt.Println("• Agents use LLM for reasoning and decision making")
	fmt.Println("• Agents query RAG for relevant knowledge and context")
	fmt.Println("• Agents execute MCP tools for external interactions")
	fmt.Println("• Workflows combine all pillars for complex tasks")
	fmt.Println()
	fmt.Println("This demonstrates RAGO as a complete AI foundation where")
	fmt.Println("each pillar can work independently or together to create")
	fmt.Println("powerful, privacy-first AI applications.")
}

// truncate truncates a string to the specified length
func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}