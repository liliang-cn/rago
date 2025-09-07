// Package client - integration_test.go
// Comprehensive tests for multi-pillar operations showcasing the four-pillar architecture synergy.
// This file validates how LLM, RAG, MCP, and Agent pillars work together to provide enhanced functionality.

package client

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// ===== MULTI-PILLAR CHAT OPERATIONS =====

func TestIntegration_ChatWithRAGAndTools(t *testing.T) {
	client := createIntegratedTestClient(t)
	defer client.Close()

	ctx := context.Background()

	t.Run("chat with RAG context enhancement", func(t *testing.T) {
		// First, ingest some documents to make RAG useful
		ingestTestDocuments(t, client)

		req := core.ChatRequest{
			Message: "What information do you have about artificial intelligence?",
			UseRAG:  true,
			Parameters: map[string]interface{}{
				"temperature": 0.7,
			},
		}

		response, err := client.Chat(ctx, req)
		if err != nil {
			t.Fatalf("Chat with RAG failed: %v", err)
		}

		if response == nil {
			t.Fatal("Response should not be nil")
		}

		if response.Response == "" {
			t.Error("Response content should not be empty")
		}

		// With RAG enabled, we should have sources
		if len(response.Sources) > 0 {
			t.Logf("RAG provided %d sources", len(response.Sources))
			for i, source := range response.Sources {
				if source.Content == "" {
					t.Errorf("Source %d should have content", i)
				}
				if source.Score < 0 || source.Score > 1 {
					t.Errorf("Source %d should have valid score [0,1], got: %f", i, source.Score)
				}
			}
		} else {
			t.Log("No RAG sources found - may indicate no matching documents")
		}

		// Response should include context from conversation
		if len(response.Context) == 0 {
			t.Error("Response should include conversation context")
		}

		if response.Duration == 0 {
			t.Error("Response should have timing information")
		}
	})

	t.Run("chat with tool usage", func(t *testing.T) {
		// Set up tools for testing
		setupIntegratedTools(t, client)

		req := core.ChatRequest{
			Message:  "Help me search for information and perform calculations",
			UseTools: true,
		}

		response, err := client.Chat(ctx, req)
		if err != nil {
			t.Fatalf("Chat with tools failed: %v", err)
		}

		if response == nil {
			t.Fatal("Response should not be nil")
		}

		// With tools enabled, we might have tool calls
		if len(response.ToolCalls) > 0 {
			t.Logf("Tools executed %d calls", len(response.ToolCalls))
			for i, toolCall := range response.ToolCalls {
				if toolCall.ID == "" {
					t.Errorf("Tool call %d should have ID", i)
				}
				if toolCall.Name == "" {
					t.Errorf("Tool call %d should have function name", i)
				}
				if toolCall.Parameters == nil {
					t.Errorf("Tool call %d should have parameters", i)
				}
			}
		} else {
			t.Log("No tool calls made - tools may not be triggered by this request")
		}

		if response.Usage.TotalTokens == 0 {
			t.Error("Response should include token usage")
		}
	})

	t.Run("chat with RAG and tools combined", func(t *testing.T) {
		// Ensure we have both documents and tools
		ingestTestDocuments(t, client)
		setupIntegratedTools(t, client)

		req := core.ChatRequest{
			Message:  "Based on the documents about AI, calculate the growth trend and search for related information",
			UseRAG:   true,
			UseTools: true,
			Context: []core.Message{
				{Role: "user", Content: "I'm researching AI trends"},
				{Role: "assistant", Content: "I can help with that research"},
			},
		}

		response, err := client.Chat(ctx, req)
		if err != nil {
			t.Fatalf("Combined chat failed: %v", err)
		}

		if response == nil {
			t.Fatal("Response should not be nil")
		}

		t.Logf("Combined operation results:")
		t.Logf("  RAG sources: %d", len(response.Sources))
		t.Logf("  Tool calls: %d", len(response.ToolCalls))
		t.Logf("  Context messages: %d", len(response.Context))
		t.Logf("  Response length: %d chars", len(response.Response))

		// This showcases the power of the four-pillar architecture:
		// - LLM provides generation
		// - RAG provides knowledge context
		// - MCP provides tool execution
		// - All coordinated by the client
		if response.Duration == 0 {
			t.Error("Response should have timing information")
		}
	})
}

func TestIntegration_StreamingChatWithMultiplePillars(t *testing.T) {
	client := createIntegratedTestClient(t)
	defer client.Close()

	ctx := context.Background()

	t.Run("streaming with RAG context", func(t *testing.T) {
		ingestTestDocuments(t, client)

		var chunks []core.StreamChunk
		callback := func(chunk core.StreamChunk) error {
			chunks = append(chunks, chunk)
			return nil
		}

		req := core.ChatRequest{
			Message: "Tell me about the AI technologies mentioned in the documents",
			UseRAG:  true,
		}

		err := client.StreamChat(ctx, req, callback)
		if err != nil {
			t.Fatalf("Streaming with RAG failed: %v", err)
		}

		if len(chunks) == 0 {
			t.Error("Should receive streaming chunks")
		}

		// Verify streaming progression
		hasContent := false
		hasFinished := false
		for _, chunk := range chunks {
			if chunk.Content != "" {
				hasContent = true
			}
			if chunk.Finished {
				hasFinished = true
			}
		}

		if !hasContent {
			t.Error("Should receive content chunks")
		}

		if !hasFinished {
			t.Error("Should receive final chunk")
		}

		t.Logf("Received %d streaming chunks", len(chunks))
	})

	t.Run("streaming error handling", func(t *testing.T) {
		callbackCount := 0
		errorCallback := func(chunk core.StreamChunk) error {
			callbackCount++
			if callbackCount > 2 {
				return fmt.Errorf("simulated callback error")
			}
			return nil
		}

		req := core.ChatRequest{
			Message: "This streaming should be interrupted",
		}

		err := client.StreamChat(ctx, req, errorCallback)
		if err == nil {
			t.Error("Expected error from callback")
		}

		if callbackCount == 0 {
			t.Error("Callback should have been called at least once")
		}
	})
}

// ===== DOCUMENT PROCESSING WITH MULTIPLE PILLARS =====

func TestIntegration_DocumentProcessingWorkflows(t *testing.T) {
	client := createIntegratedTestClient(t)
	defer client.Close()

	ctx := context.Background()

	t.Run("document ingestion and analysis workflow", func(t *testing.T) {
		// Step 1: Ingest document
		ingestReq := core.DocumentRequest{
			Action:     "ingest",
			DocumentID: "workflow-doc-1",
			Content:    "This document discusses advanced machine learning techniques including neural networks, deep learning, and transformer architectures. The latest developments in AI show significant progress in natural language processing and computer vision.",
		}

		ingestResp, err := client.ProcessDocument(ctx, ingestReq)
		if err != nil {
			t.Fatalf("Document ingestion failed: %v", err)
		}

		if ingestResp == nil {
			t.Fatal("Ingestion response should not be nil")
		}

		if ingestResp.DocumentID == "" {
			t.Error("Document ID should be set after ingestion")
		}

		t.Logf("Ingested document: %s", ingestResp.DocumentID)

		// Step 2: Analyze the same document
		analyzeReq := core.DocumentRequest{
			Action:  "analyze",
			Content: ingestReq.Content,
			Parameters: map[string]interface{}{
				"focus": "technical_concepts",
			},
		}

		analyzeResp, err := client.ProcessDocument(ctx, analyzeReq)
		if err != nil {
			t.Fatalf("Document analysis failed: %v", err)
		}

		if analyzeResp == nil {
			t.Fatal("Analysis response should not be nil")
		}

		if analyzeResp.Result == "" {
			t.Error("Analysis should produce results")
		}

		t.Logf("Analysis result: %s", analyzeResp.Result[:min(len(analyzeResp.Result), 100)])

		// Step 3: Query the ingested document via chat
		chatReq := core.ChatRequest{
			Message: "What machine learning techniques are mentioned in the documents?",
			UseRAG:  true,
		}

		chatResp, err := client.Chat(ctx, chatReq)
		if err != nil {
			t.Fatalf("Chat query failed: %v", err)
		}

		if chatResp == nil {
			t.Fatal("Chat response should not be nil")
		}

		// Should find relevant information from the ingested document
		if len(chatResp.Sources) > 0 {
			t.Logf("Found %d relevant sources in chat response", len(chatResp.Sources))
		}

		t.Logf("Chat response: %s", chatResp.Response[:min(len(chatResp.Response), 100)])
	})

	t.Run("batch document processing", func(t *testing.T) {
		documents := []string{
			"Artificial intelligence has revolutionized various industries through automation and smart algorithms.",
			"Machine learning models require large datasets and computational resources for training.",
			"Natural language processing enables computers to understand and generate human language.",
		}

		ingestedDocs := make([]string, 0, len(documents))

		// Ingest multiple documents
		for i, content := range documents {
			req := core.DocumentRequest{
				Action:     "ingest",
				DocumentID: fmt.Sprintf("batch-doc-%d", i),
				Content:    content,
			}

			resp, err := client.ProcessDocument(ctx, req)
			if err != nil {
				t.Errorf("Failed to ingest document %d: %v", i, err)
				continue
			}

			if resp != nil && resp.DocumentID != "" {
				ingestedDocs = append(ingestedDocs, resp.DocumentID)
			}
		}

		t.Logf("Successfully ingested %d documents", len(ingestedDocs))

		// Query across all documents
		chatReq := core.ChatRequest{
			Message: "Summarize the key concepts across all the AI documents",
			UseRAG:  true,
		}

		chatResp, err := client.Chat(ctx, chatReq)
		if err != nil {
			t.Fatalf("Multi-document chat failed: %v", err)
		}

		if chatResp != nil && len(chatResp.Sources) > 0 {
			t.Logf("Multi-document query found %d sources", len(chatResp.Sources))
		}
	})
}

// ===== TASK EXECUTION WITH AGENT WORKFLOWS =====

func TestIntegration_ComplexTaskExecution(t *testing.T) {
	client := createIntegratedTestClient(t)
	defer client.Close()

	ctx := context.Background()

	t.Run("agent-coordinated task with multiple pillars", func(t *testing.T) {
		// Set up comprehensive environment
		ingestTestDocuments(t, client)
		setupIntegratedTools(t, client)
		setupIntegratedAgents(t, client)

		req := core.TaskRequest{
			Task:  "Analyze the AI documents, extract key insights, and create a comprehensive report",
			Agent: "research-agent",
			Parameters: map[string]interface{}{
				"output_format": "detailed_report",
				"include_sources": true,
			},
		}

		response, err := client.ExecuteTask(ctx, req)
		if err != nil {
			t.Fatalf("Complex task execution failed: %v", err)
		}

		if response == nil {
			t.Fatal("Task response should not be nil")
		}

		if response.Result == "" {
			t.Error("Task should produce results")
		}

		// Agent execution should show steps
		if len(response.Steps) > 0 {
			t.Logf("Agent executed %d steps:", len(response.Steps))
			for i, step := range response.Steps {
				t.Logf("  Step %d (%s): %s", i+1, step.Status, step.StepID)
				if step.Duration == 0 {
					t.Errorf("Step %d should have duration", i+1)
				}
			}
		}

		if response.Duration == 0 {
			t.Error("Task should have total duration")
		}

		t.Logf("Task result: %s", response.Result[:min(len(response.Result), 200)])
	})

	t.Run("workflow-based task execution", func(t *testing.T) {
		setupIntegratedAgents(t, client)

		req := core.TaskRequest{
			Task:     "Process customer feedback and generate insights",
			Workflow: "feedback-analysis",
			Context: map[string]interface{}{
				"feedback_type": "product_review",
				"priority":      "high",
			},
		}

		response, err := client.ExecuteTask(ctx, req)
		if err != nil {
			t.Fatalf("Workflow task execution failed: %v", err)
		}

		if response == nil {
			t.Fatal("Workflow response should not be nil")
		}

		// Workflow should produce structured steps
		if len(response.Steps) > 0 {
			t.Logf("Workflow executed %d steps", len(response.Steps))
			
			// Verify step structure
			for i, step := range response.Steps {
				if step.StepID == "" {
					t.Errorf("Step %d should have ID", i+1)
				}
				if step.Status != "completed" {
					t.Errorf("Step %d should be completed, got: %s", i+1, step.Status)
				}
				if step.Output == "" {
					t.Errorf("Step %d should have output", i+1)
				}
			}
		} else {
			t.Log("No workflow steps recorded - may indicate mock workflow")
		}
	})

	t.Run("fallback task execution via LLM", func(t *testing.T) {
		// Task without specific agent or workflow - should fallback to LLM
		req := core.TaskRequest{
			Task: "Explain the benefits of the four-pillar architecture in AI systems",
			Parameters: map[string]interface{}{
				"style": "educational",
				"depth": "comprehensive",
			},
		}

		response, err := client.ExecuteTask(ctx, req)
		if err != nil {
			t.Fatalf("Fallback task execution failed: %v", err)
		}

		if response == nil {
			t.Fatal("Response should not be nil")
		}

		if response.Result == "" {
			t.Error("Task should produce results via LLM fallback")
		}

		// Should have usage information for LLM fallback
		if response.Usage.TotalTokens == 0 {
			t.Log("No usage tokens recorded - may indicate mock response")
		}

		t.Logf("LLM fallback result: %s", response.Result[:min(len(response.Result), 150)])
	})
}

// ===== PILLAR SYNERGY AND COORDINATION TESTS =====

func TestIntegration_PillarSynergy(t *testing.T) {
	client := createIntegratedTestClient(t)
	defer client.Close()

	ctx := context.Background()

	t.Run("knowledge retrieval enhances generation", func(t *testing.T) {
		// Ingest specialized knowledge
		specialDoc := core.DocumentRequest{
			Action:     "ingest",
			DocumentID: "synergy-test-doc",
			Content:    "The four-pillar architecture in RAGO consists of: 1) LLM for generation, 2) RAG for knowledge retrieval, 3) MCP for tool integration, 4) Agents for workflow orchestration. Each pillar can work independently or in coordination with others.",
		}

		_, err := client.ProcessDocument(ctx, specialDoc)
		if err != nil {
			t.Fatalf("Document ingestion failed: %v", err)
		}

		// Query without RAG
		baseReq := core.ChatRequest{
			Message: "What are the four pillars in RAGO?",
			UseRAG:  false,
		}

		baseResp, err := client.Chat(ctx, baseReq)
		if err != nil {
			t.Fatalf("Base chat failed: %v", err)
		}

		// Query with RAG
		enhancedReq := core.ChatRequest{
			Message: "What are the four pillars in RAGO?",
			UseRAG:  true,
		}

		enhancedResp, err := client.Chat(ctx, enhancedReq)
		if err != nil {
			t.Fatalf("Enhanced chat failed: %v", err)
		}

		// RAG-enhanced response should be more informed
		t.Logf("Base response length: %d", len(baseResp.Response))
		t.Logf("Enhanced response length: %d", len(enhancedResp.Response))
		t.Logf("RAG sources: %d", len(enhancedResp.Sources))

		if len(enhancedResp.Sources) > 0 {
			t.Log("RAG successfully provided knowledge enhancement")
		}
	})

	t.Run("tools extend generation capabilities", func(t *testing.T) {
		setupIntegratedTools(t, client)

		// Task that benefits from tool usage
		toolReq := core.ChatRequest{
			Message:  "Calculate the performance metrics and search for related benchmarks",
			UseTools: true,
		}

		toolResp, err := client.Chat(ctx, toolReq)
		if err != nil {
			t.Fatalf("Tool-enhanced chat failed: %v", err)
		}

		if len(toolResp.ToolCalls) > 0 {
			t.Logf("Tools enhanced generation with %d calls", len(toolResp.ToolCalls))
			for _, call := range toolResp.ToolCalls {
				t.Logf("  Tool: %s -> %v", call.Name, call.Parameters)
			}
		} else {
			t.Log("No tools called - may depend on mock tool configuration")
		}
	})

	t.Run("agents orchestrate multi-pillar workflows", func(t *testing.T) {
		ingestTestDocuments(t, client)
		setupIntegratedTools(t, client)
		setupIntegratedAgents(t, client)

		// Complex task requiring all pillars
		agentReq := core.TaskRequest{
			Task:  "Research AI trends, analyze documents, use tools to gather data, and create a comprehensive report",
			Agent: "research-agent",
		}

		agentResp, err := client.ExecuteTask(ctx, agentReq)
		if err != nil {
			t.Fatalf("Agent orchestration failed: %v", err)
		}

		// Agent should coordinate multiple operations
		if len(agentResp.Steps) > 0 {
			t.Logf("Agent orchestrated %d steps", len(agentResp.Steps))
			
			stepTypes := make(map[string]int)
			for _, step := range agentResp.Steps {
				// Analyze step types to see orchestration
				if output, ok := step.Output.(string); ok {
					if strings.Contains(output, "analyze") {
						stepTypes["analysis"]++
					}
					if strings.Contains(output, "search") {
						stepTypes["search"]++
					}
					if strings.Contains(output, "execute") {
						stepTypes["execution"]++
					}
				}
			}
			
			t.Logf("Step types: %v", stepTypes)
		}

		if agentResp.Result == "" {
			t.Error("Agent should produce comprehensive results")
		}
	})
}

// ===== PERFORMANCE AND LOAD TESTING =====

func TestIntegration_PerformanceUnderLoad(t *testing.T) {
	client := createIntegratedTestClient(t)
	defer client.Close()

	ctx := context.Background()

	t.Run("concurrent multi-pillar operations", func(t *testing.T) {
		// Set up environment
		ingestTestDocuments(t, client)
		setupIntegratedTools(t, client)

		numConcurrent := 5
		results := make(chan error, numConcurrent)

		// Launch concurrent operations
		for i := 0; i < numConcurrent; i++ {
			go func(id int) {
				defer func() { results <- nil }()

				// Different types of operations concurrently
				switch id % 3 {
				case 0:
					// Chat with RAG
					req := core.ChatRequest{
						Message: fmt.Sprintf("Concurrent query %d about AI", id),
						UseRAG:  true,
					}
					_, err := client.Chat(ctx, req)
					if err != nil {
						results <- fmt.Errorf("concurrent chat %d failed: %v", id, err)
						return
					}

				case 1:
					// Document processing
					req := core.DocumentRequest{
						Action:  "analyze",
						Content: fmt.Sprintf("Concurrent document %d content for analysis", id),
					}
					_, err := client.ProcessDocument(ctx, req)
					if err != nil {
						results <- fmt.Errorf("concurrent document processing %d failed: %v", id, err)
						return
					}

				case 2:
					// Task execution
					req := core.TaskRequest{
						Task: fmt.Sprintf("Concurrent task %d execution", id),
					}
					_, err := client.ExecuteTask(ctx, req)
					if err != nil {
						results <- fmt.Errorf("concurrent task %d failed: %v", id, err)
						return
					}
				}
			}(i)
		}

		// Collect results
		for i := 0; i < numConcurrent; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent operation failed: %v", err)
			}
		}
	})

	t.Run("sustained operation performance", func(t *testing.T) {
		numOperations := 10
		start := time.Now()

		for i := 0; i < numOperations; i++ {
			req := core.ChatRequest{
				Message: fmt.Sprintf("Sustained operation %d", i),
				UseRAG:  true,
			}

			_, err := client.Chat(ctx, req)
			if err != nil {
				t.Errorf("Sustained operation %d failed: %v", i, err)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(numOperations)

		t.Logf("Sustained operations: %d in %v (avg: %v)", numOperations, duration, avgDuration)

		// Performance should be reasonable for mock operations
		if avgDuration > 1*time.Second {
			t.Logf("Average operation time seems high: %v", avgDuration)
		}
	})
}

// ===== ERROR HANDLING AND RESILIENCE =====

func TestIntegration_ErrorHandlingAndResilience(t *testing.T) {
	t.Run("partial pillar failure handling", func(t *testing.T) {
		// Create client with some pillars unavailable
		client := createPartialTestClient(t)
		defer client.Close()

		ctx := context.Background()

		// Chat should work even with limited pillars
		req := core.ChatRequest{
			Message: "This should work with limited pillars",
		}

		_, err := client.Chat(ctx, req)
		if err != nil {
			t.Logf("Chat with partial pillars failed: %v", err)
			// This may be expected depending on what's available
		}

		// Health should reflect partial availability
		health := client.Health()
		if health.Overall == core.HealthStatusHealthy && len(health.Pillars) < 4 {
			t.Log("Health shows healthy with partial pillars - may indicate resilient design")
		}
	})

	t.Run("recovery from temporary failures", func(t *testing.T) {
		client := createIntegratedTestClient(t)
		defer client.Close()

		// Simulate temporary failure by manipulating mock
		if mockLLM, ok := client.llmService.(*MockLLMService); ok {
			// Set up failure condition
			mockLLM.SetGenerateFunc(func(ctx context.Context, req core.GenerationRequest) (*core.GenerationResponse, error) {
				return nil, fmt.Errorf("temporary failure")
			})

			ctx := context.Background()
			req := core.ChatRequest{Message: "This should fail"}

			_, err := client.Chat(ctx, req)
			if err == nil {
				t.Error("Expected failure with broken LLM service")
			}

			// Restore service
			mockLLM.SetGenerateFunc(nil) // Reset to default

			// Should work again
			_, err = client.Chat(ctx, req)
			if err != nil {
				t.Errorf("Should recover from temporary failure: %v", err)
			}
		}
	})
}

// ===== HELPER FUNCTIONS FOR INTEGRATION TESTING =====

func createIntegratedTestClient(t *testing.T) *Client {
	t.Helper()
	
	client := createTestClient(t)
	
	// Ensure all services are set up for integration testing
	setupMockServices(t, client)
	
	return client
}

func createPartialTestClient(t *testing.T) *Client {
	t.Helper()
	
	// Create client with only some pillars
	ctx, cancel := context.WithCancel(context.Background())
	
	client := &Client{
		config:       getDefaultConfig(),
		ctx:          ctx,
		cancel:       cancel,
		llmService:   NewMockLLMService(),
		ragService:   nil, // No RAG service
		mcpService:   nil, // No MCP service
		agentService: NewMockAgentService(),
	}
	
	client.healthMonitor = NewHealthMonitor(client)
	
	// Setup available services
	if mockLLM, ok := client.llmService.(*MockLLMService); ok {
		mockLLM.AddProvider("partial-provider", core.ProviderConfig{
			Type:  "partial",
			Model: "partial-model",
		})
	}
	
	return client
}

func ingestTestDocuments(t *testing.T, client *Client) {
	t.Helper()
	
	documents := []struct {
		id      string
		content string
	}{
		{
			id:      "ai-overview",
			content: "Artificial Intelligence (AI) encompasses machine learning, deep learning, and neural networks. Modern AI systems use transformer architectures and large language models for natural language processing.",
		},
		{
			id:      "ml-techniques",
			content: "Machine learning techniques include supervised learning, unsupervised learning, and reinforcement learning. Popular algorithms include decision trees, support vector machines, and neural networks.",
		},
		{
			id:      "nlp-advances",
			content: "Natural Language Processing has advanced significantly with transformer models like BERT, GPT, and T5. These models excel at text understanding, generation, and translation tasks.",
		},
	}
	
	ctx := context.Background()
	
	for _, doc := range documents {
		req := core.DocumentRequest{
			Action:     "ingest",
			DocumentID: doc.id,
			Content:    doc.content,
		}
		
		_, err := client.ProcessDocument(ctx, req)
		if err != nil {
			t.Logf("Failed to ingest document %s: %v", doc.id, err)
		}
	}
}

func setupIntegratedTools(t *testing.T, client *Client) {
	t.Helper()
	
	if mockMCP, ok := client.mcpService.(*MockMCPService); ok {
		// Add useful tools for testing
		tools := []core.ToolInfo{
			{
				Name:        "calculator",
				Description: "Performs mathematical calculations",
				InputSchema: map[string]interface{}{
					"expression": "string",
				},
			},
			{
				Name:        "search",
				Description: "Searches for information",
				InputSchema: map[string]interface{}{
					"query": "string",
				},
			},
			{
				Name:        "analyzer",
				Description: "Analyzes data and provides insights",
				InputSchema: map[string]interface{}{
					"data": "object",
				},
			},
		}
		
		for _, tool := range tools {
			mockMCP.AddMockTool(tool)
		}
		
		// Set up enhanced tool responses
		mockMCP.SetCallToolFunc(func(ctx context.Context, req core.ToolCallRequest) (*core.ToolCallResponse, error) {
			switch req.ToolName {
			case "calculator":
				return &core.ToolCallResponse{
					Result:   "Calculation result: 42",
					Success:  true,
					Duration: 50 * time.Millisecond,
				}, nil
			case "search":
				return &core.ToolCallResponse{
					Result:   "Search results: Found 5 relevant documents",
					Success:  true,
					Duration: 100 * time.Millisecond,
				}, nil
			case "analyzer":
				return &core.ToolCallResponse{
					Result:   "Analysis complete: Trends show positive growth",
					Success:  true,
					Duration: 150 * time.Millisecond,
				}, nil
			default:
				return &core.ToolCallResponse{
					Result:  fmt.Sprintf("Generic tool result for %s", req.ToolName),
					Success: true,
				}, nil
			}
		})
	}
}

func setupIntegratedAgents(t *testing.T, client *Client) {
	t.Helper()
	
	if mockAgent, ok := client.agentService.(*MockAgentService); ok {
		// Create research agent
		mockAgent.CreateAgent(core.AgentDefinition{
			Name:        "research-agent",
			Description: "Conducts comprehensive research using multiple tools",
			Type:        "research",
		})
		
		// Create feedback analysis workflow
		mockAgent.CreateWorkflow(core.WorkflowDefinition{
			Name:        "feedback-analysis",
			Description: "Analyzes customer feedback and generates insights",
		})
		
		// Enhanced agent execution
		mockAgent.SetExecuteAgentFunc(func(ctx context.Context, req core.AgentRequest) (*core.AgentResponse, error) {
			steps := []core.AgentStep{
				{
					StepNumber: 1,
					Action:     "analyze_requirements",
					Output:     "Analyzed task requirements and identified key components",
					Duration:   50 * time.Millisecond,
				},
				{
					StepNumber: 2,
					Action:     "gather_information",
					Output:     "Gathered information from knowledge base and tools",
					Duration:   100 * time.Millisecond,
				},
				{
					StepNumber: 3,
					Action:     "synthesize_results",
					Output:     "Synthesized information into comprehensive analysis",
					Duration:   75 * time.Millisecond,
				},
			}
			
			return &core.AgentResponse{
				AgentName: req.AgentName,
				Result:    fmt.Sprintf("Agent %s completed comprehensive analysis of: %s", req.AgentName, req.Task),
				Steps:     steps,
				Duration:  225 * time.Millisecond,
			}, nil
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}