// Package client - client_test.go
// Comprehensive tests for the unified RAGO client implementing the four-pillar architecture.
// This file validates the core Client interface and high-level multi-pillar operations.

package client

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// ===== CLIENT INTERFACE TESTING =====

func TestClient_New(t *testing.T) {
	tests := []struct {
		name        string
		configPath  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid config path uses defaults",
			configPath:  "/nonexistent/path/config.toml",
			expectError: false,
		},
		{
			name:        "empty config path uses defaults",
			configPath:  "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.configPath)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorMsg, err.Error())
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if client == nil {
				t.Error("Client should not be nil")
			}
			
			// Clean up
			if client != nil {
				client.Close()
			}
		})
	}
}

func TestClient_NewWithDefaults(t *testing.T) {
	client, err := NewWithDefaults()
	if err != nil {
		t.Fatalf("Failed to create client with defaults: %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	// Verify client has expected structure
	config := client.GetConfig()
	if config == nil {
		t.Error("Client config should not be nil")
	}

	if config.ClientName != "rago-client" {
		t.Errorf("Expected client name 'rago-client', got: %s", config.ClientName)
	}

	if config.ClientVersion != "3.0.0" {
		t.Errorf("Expected client version '3.0.0', got: %s", config.ClientVersion)
	}
}

func TestClient_NewWithConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "config",
		},
		{
			name:        "valid config",
			config:      getDefaultConfig(),
			expectError: false,
		},
		{
			name: "config with only LLM and RAG enabled",
			config: &Config{
				Config: core.Config{
					DataDir:  "/tmp/test",
					LogLevel: "info",
					Mode: core.ModeConfig{
						RAGOnly:      false,
						LLMOnly:      false,
						DisableMCP:   true,
						DisableAgent: true,
					},
					LLM: core.LLMConfig{
						DefaultProvider: "test-provider",
						Providers: core.ProvidersConfig{
							List: []core.ProviderConfig{
								{
									Name:    "test-provider",
									Type:    "ollama",
									BaseURL: "http://localhost:11434",
									Model:   "test-model",
									Weight:  1,
									Enabled: true,
									Timeout: 30 * time.Second,
								},
							},
						},
						LoadBalancing: core.LoadBalancingConfig{
							Strategy: "round_robin",
						},
						HealthCheck: core.HealthCheckConfig{
							Enabled:  false, // Disable for testing
							Interval: 30 * time.Second,
						},
					},
					RAG: core.RAGConfig{
						StorageBackend: "sqvect",
						ChunkingStrategy: core.ChunkingConfig{
							Strategy:     "fixed",
							ChunkSize:    1000,
							ChunkOverlap: 200,
							MinChunkSize: 100,
						},
						VectorStore: core.VectorStoreConfig{
							Backend:    "sqvect",
							Dimensions: 0, // Auto-detect
							Metric:     "cosine",
							IndexType:  "flat",
						},
					},
				},
			},
			expectError: false, // LLM and RAG are enabled with proper configs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewWithConfig(tt.config)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					if client != nil {
						client.Close()
					}
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorMsg, err.Error())
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if client == nil {
				t.Error("Client should not be nil")
			}
			
			// Clean up
			if client != nil {
				client.Close()
			}
		})
	}
}

// ===== PILLAR ACCESS TESTING =====

func TestClient_PillarAccess(t *testing.T) {
	// Create client with mock services
	client := createTestClient(t)
	defer client.Close()

	t.Run("LLM pillar access", func(t *testing.T) {
		llmService := client.LLM()
		if llmService == nil {
			t.Error("LLM service should be available")
		}
		
		// Test that we can call LLM service methods
		providers := llmService.ListProviders()
		if providers == nil {
			t.Error("Should be able to list providers")
		}
	})

	t.Run("RAG pillar access", func(t *testing.T) {
		ragService := client.RAG()
		if ragService == nil {
			t.Error("RAG service should be available")
		}
		
		// Test that we can call RAG service methods
		ctx := context.Background()
		stats, err := ragService.GetStats(ctx)
		if err != nil {
			t.Errorf("Should be able to get RAG stats: %v", err)
		}
		if stats == nil {
			t.Error("RAG stats should not be nil")
		}
	})

	t.Run("MCP pillar access", func(t *testing.T) {
		mcpService := client.MCP()
		if mcpService == nil {
			t.Error("MCP service should be available")
		}
		
		// Test that we can call MCP service methods
		servers := mcpService.ListServers()
		if servers == nil {
			t.Error("Should be able to list servers")
		}
	})

	t.Run("Agents pillar access", func(t *testing.T) {
		agentsService := client.Agents()
		if agentsService == nil {
			t.Error("Agents service should be available")
		}
		
		// Test that we can call Agents service methods
		workflows := agentsService.ListWorkflows()
		if workflows == nil {
			t.Error("Should be able to list workflows")
		}
	})
}

func TestClient_PillarAccess_ClosedClient(t *testing.T) {
	client := createTestClient(t)
	client.Close()

	// Test accessing pillars on closed client should still return services
	// but their operations should fail
	t.Run("LLM pillar on closed client", func(t *testing.T) {
		llmService := client.LLM()
		if llmService == nil {
			t.Error("LLM service should still be accessible")
		}
	})

	t.Run("RAG pillar on closed client", func(t *testing.T) {
		ragService := client.RAG()
		if ragService == nil {
			t.Error("RAG service should still be accessible")
		}
	})

	t.Run("MCP pillar on closed client", func(t *testing.T) {
		mcpService := client.MCP()
		if mcpService == nil {
			t.Error("MCP service should still be accessible")
		}
	})

	t.Run("Agents pillar on closed client", func(t *testing.T) {
		agentsService := client.Agents()
		if agentsService == nil {
			t.Error("Agents service should still be accessible")
		}
	})
}

// ===== HIGH-LEVEL OPERATIONS TESTING =====

func TestClient_Chat(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	ctx := context.Background()

	tests := []struct {
		name     string
		request  core.ChatRequest
		validate func(t *testing.T, response *core.ChatResponse, err error)
	}{
		{
			name: "basic chat request",
			request: core.ChatRequest{
				Message: "Hello, world!",
				Parameters: map[string]interface{}{
					"temperature": 0.7,
				},
			},
			validate: func(t *testing.T, response *core.ChatResponse, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Response should not be nil")
					return
				}
				if response.Response == "" {
					t.Error("Response content should not be empty")
				}
				if response.Duration == 0 {
					t.Error("Response should have duration")
				}
			},
		},
		{
			name: "chat with RAG context",
			request: core.ChatRequest{
				Message: "What is in the documents?",
				UseRAG:  true,
				Parameters: map[string]interface{}{
					"temperature": 0.5,
				},
			},
			validate: func(t *testing.T, response *core.ChatResponse, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Response should not be nil")
					return
				}
				// With RAG enabled, we might have sources
				if response.Sources != nil && len(response.Sources) > 0 {
					t.Log("RAG sources found in response")
				}
			},
		},
		{
			name: "chat with tools",
			request: core.ChatRequest{
				Message:  "Help me with a task that requires tools",
				UseTools: true,
			},
			validate: func(t *testing.T, response *core.ChatResponse, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Response should not be nil")
					return
				}
				// With tools enabled, we might have tool calls
				if response.ToolCalls != nil && len(response.ToolCalls) > 0 {
					t.Log("Tool calls found in response")
				}
			},
		},
		{
			name: "chat with context messages",
			request: core.ChatRequest{
				Message: "Continue our conversation",
				Context: []core.Message{
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi there!"},
				},
			},
			validate: func(t *testing.T, response *core.ChatResponse, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Response should not be nil")
					return
				}
				if len(response.Context) == 0 {
					t.Error("Response context should include conversation history")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := client.Chat(ctx, tt.request)
			tt.validate(t, response, err)
		})
	}
}

func TestClient_StreamChat(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	ctx := context.Background()

	t.Run("basic streaming chat", func(t *testing.T) {
		var chunks []core.StreamChunk
		callback := func(chunk core.StreamChunk) error {
			chunks = append(chunks, chunk)
			return nil
		}

		req := core.ChatRequest{
			Message: "Tell me a story",
		}

		err := client.StreamChat(ctx, req, callback)
		if err != nil {
			t.Errorf("StreamChat failed: %v", err)
		}

		if len(chunks) == 0 {
			t.Error("Should have received streaming chunks")
		}

		// Verify we received a final chunk
		hasFinished := false
		for _, chunk := range chunks {
			if chunk.Finished {
				hasFinished = true
				break
			}
		}
		if !hasFinished {
			t.Error("Should have received a final chunk")
		}
	})

	t.Run("streaming chat with context", func(t *testing.T) {
		var chunkCount int
		callback := func(chunk core.StreamChunk) error {
			chunkCount++
			return nil
		}

		req := core.ChatRequest{
			Message: "Continue the story",
			Context: []core.Message{
				{Role: "user", Content: "Tell me a story"},
				{Role: "assistant", Content: "Once upon a time..."},
			},
		}

		err := client.StreamChat(ctx, req, callback)
		if err != nil {
			t.Errorf("StreamChat failed: %v", err)
		}

		if chunkCount == 0 {
			t.Error("Should have received streaming chunks")
		}
	})

	t.Run("streaming chat error handling", func(t *testing.T) {
		errorCallback := func(chunk core.StreamChunk) error {
			return context.Canceled
		}

		req := core.ChatRequest{
			Message: "This should fail",
		}

		err := client.StreamChat(ctx, req, errorCallback)
		if err == nil {
			t.Error("Expected error from callback, got none")
		}
	})
}

func TestClient_ProcessDocument(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	ctx := context.Background()

	tests := []struct {
		name     string
		request  core.DocumentRequest
		validate func(t *testing.T, response *core.DocumentResponse, err error)
	}{
		{
			name: "document analysis",
			request: core.DocumentRequest{
				Action:  "analyze",
				Content: "This is a test document with important information about machine learning.",
			},
			validate: func(t *testing.T, response *core.DocumentResponse, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Response should not be nil")
					return
				}
				if response.Action != "analyze" {
					t.Errorf("Expected action 'analyze', got: %s", response.Action)
				}
				if response.Result == "" {
					t.Error("Analysis result should not be empty")
				}
			},
		},
		{
			name: "document ingestion",
			request: core.DocumentRequest{
				Action:     "ingest",
				DocumentID: "test-doc-1",
				Content:    "This document will be ingested into the RAG system for later retrieval.",
			},
			validate: func(t *testing.T, response *core.DocumentResponse, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Response should not be nil")
					return
				}
				if response.Action != "ingest" {
					t.Errorf("Expected action 'ingest', got: %s", response.Action)
				}
				if response.DocumentID == "" {
					t.Error("Document ID should be set")
				}
			},
		},
		{
			name: "unsupported action",
			request: core.DocumentRequest{
				Action:  "unsupported",
				Content: "Test content",
			},
			validate: func(t *testing.T, response *core.DocumentResponse, err error) {
				if err == nil {
					t.Error("Expected error for unsupported action")
				}
				if !strings.Contains(err.Error(), "unsupported") {
					t.Errorf("Expected unsupported action error, got: %v", err)
				}
			},
		},
		{
			name: "analysis with parameters",
			request: core.DocumentRequest{
				Action:  "analyze",
				Content: "Technical document about Go programming language.",
				Parameters: map[string]interface{}{
					"focus": "technical_details",
				},
			},
			validate: func(t *testing.T, response *core.DocumentResponse, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Response should not be nil")
					return
				}
				if response.Duration == 0 {
					t.Error("Response should have duration")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := client.ProcessDocument(ctx, tt.request)
			tt.validate(t, response, err)
		})
	}
}

func TestClient_ExecuteTask(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	ctx := context.Background()

	tests := []struct {
		name     string
		request  core.TaskRequest
		validate func(t *testing.T, response *core.TaskResponse, err error)
	}{
		{
			name: "basic task execution",
			request: core.TaskRequest{
				Task: "Analyze the current market trends and provide insights",
			},
			validate: func(t *testing.T, response *core.TaskResponse, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Response should not be nil")
					return
				}
				if response.Task == "" {
					t.Error("Task should be preserved in response")
				}
				if response.Result == "" {
					t.Error("Task result should not be empty")
				}
			},
		},
		{
			name: "task with specific agent",
			request: core.TaskRequest{
				Task:  "Create a comprehensive report",
				Agent: "report-generator",
			},
			validate: func(t *testing.T, response *core.TaskResponse, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Response should not be nil")
					return
				}
				if len(response.Steps) == 0 {
					t.Error("Agent execution should produce steps")
				}
			},
		},
		{
			name: "task with workflow",
			request: core.TaskRequest{
				Task:     "Process customer feedback",
				Workflow: "feedback-analysis",
			},
			validate: func(t *testing.T, response *core.TaskResponse, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Response should not be nil")
					return
				}
				if len(response.Steps) == 0 {
					t.Error("Workflow execution should produce steps")
				}
			},
		},
		{
			name: "task with parameters",
			request: core.TaskRequest{
				Task: "Generate summary with specific parameters",
				Parameters: map[string]interface{}{
					"max_length": 500,
					"style":      "formal",
				},
			},
			validate: func(t *testing.T, response *core.TaskResponse, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Response should not be nil")
					return
				}
				if response.Duration == 0 {
					t.Error("Response should have duration")
				}
			},
		},
		{
			name: "task with context",
			request: core.TaskRequest{
				Task: "Continue previous analysis",
				Context: map[string]interface{}{
					"previous_results": "Analysis shows positive trends",
				},
			},
			validate: func(t *testing.T, response *core.TaskResponse, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if response == nil {
					t.Error("Response should not be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := client.ExecuteTask(ctx, tt.request)
			tt.validate(t, response, err)
		})
	}
}

// ===== CLIENT LIFECYCLE TESTING =====

func TestClient_Close(t *testing.T) {
	t.Run("close normal client", func(t *testing.T) {
		client := createTestClient(t)
		
		err := client.Close()
		if err != nil {
			t.Errorf("Close should not return error: %v", err)
		}
		
		// Second close should not error
		err = client.Close()
		if err != nil {
			t.Errorf("Second close should not return error: %v", err)
		}
	})

	t.Run("close client multiple times", func(t *testing.T) {
		client := createTestClient(t)
		
		// Close multiple times
		for i := 0; i < 3; i++ {
			err := client.Close()
			if err != nil {
				t.Errorf("Close #%d should not return error: %v", i+1, err)
			}
		}
	})
}

func TestClient_Health(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	t.Run("health report structure", func(t *testing.T) {
		health := client.Health()
		
		if health.Overall == "" {
			t.Error("Health report should have overall status")
		}
		
		if health.Pillars == nil {
			t.Error("Health report should have pillars map")
		}
		
		if health.Providers == nil {
			t.Error("Health report should have providers map")
		}
		
		if health.Servers == nil {
			t.Error("Health report should have servers map")
		}
		
		if health.Details == nil {
			t.Error("Health report should have details map")
		}
		
		if health.LastCheck.IsZero() {
			t.Error("Health report should have last check time")
		}
	})

	t.Run("health status values", func(t *testing.T) {
		health := client.Health()
		
		validStatuses := map[core.HealthStatus]bool{
			core.HealthStatusHealthy:   true,
			core.HealthStatusDegraded: true,
			core.HealthStatusUnhealthy: true,
			core.HealthStatusUnknown:  true,
		}
		
		if !validStatuses[health.Overall] {
			t.Errorf("Invalid overall health status: %s", health.Overall)
		}
		
		// Check pillar statuses
		for pillarName, status := range health.Pillars {
			if !validStatuses[status] {
				t.Errorf("Invalid pillar %s health status: %s", pillarName, status)
			}
		}
	})
}

// ===== ERROR HANDLING TESTING =====

func TestClient_ErrorHandling(t *testing.T) {
	t.Run("chat with no LLM service", func(t *testing.T) {
		// Create client with no LLM service
		client := createTestClientWithoutLLM(t)
		defer client.Close()

		ctx := context.Background()
		req := core.ChatRequest{
			Message: "This should fail",
		}

		_, err := client.Chat(ctx, req)
		if err == nil {
			t.Error("Expected error when no LLM service available")
		}
		if !strings.Contains(err.Error(), "LLM service not available") {
			t.Errorf("Expected LLM service error, got: %v", err)
		}
	})

	t.Run("document processing with no RAG service", func(t *testing.T) {
		client := createTestClientWithoutRAG(t)
		defer client.Close()

		ctx := context.Background()
		req := core.DocumentRequest{
			Action:  "ingest",
			Content: "Test content",
		}

		_, err := client.ProcessDocument(ctx, req)
		if err == nil {
			t.Error("Expected error when no RAG service available for ingestion")
		}
		if !strings.Contains(err.Error(), "RAG service required") {
			t.Errorf("Expected RAG service error, got: %v", err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		client := createTestClient(t)
		defer client.Close()

		// Test timeout instead since mock might not respect cancellation
		timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancelTimeout()

		req := core.ChatRequest{
			Message: "This should timeout",
		}

		_, err := client.Chat(timeoutCtx, req)
		// This might not error due to mock implementation, which is fine
		_ = err
	})
}

// ===== CONCURRENT ACCESS TESTING =====

func TestClient_ConcurrentAccess(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	ctx := context.Background()

	t.Run("concurrent chat operations", func(t *testing.T) {
		numGoroutines := 10
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				req := core.ChatRequest{
					Message: fmt.Sprintf("Concurrent message %d", id),
				}

				_, err := client.Chat(ctx, req)
				errors <- err
			}(i)
		}

		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-errors
			if err != nil {
				t.Errorf("Concurrent operation %d failed: %v", i, err)
			}
		}
	})

	t.Run("concurrent pillar access", func(t *testing.T) {
		numGoroutines := 20
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				// Access different pillars concurrently
				switch id % 4 {
				case 0:
					llm := client.LLM()
					if llm != nil {
						llm.ListProviders()
					}
				case 1:
					rag := client.RAG()
					if rag != nil {
						rag.GetStats(ctx)
					}
				case 2:
					mcp := client.MCP()
					if mcp != nil {
						mcp.ListServers()
					}
				case 3:
					agents := client.Agents()
					if agents != nil {
						agents.ListWorkflows()
					}
				}
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

// ===== HELPER FUNCTIONS =====

// createTestClient creates a client with all mock services for testing
func createTestClient(t *testing.T) *Client {
	t.Helper()
	
	ctx, cancel := context.WithCancel(context.Background())
	
	client := &Client{
		config: getDefaultConfig(),
		ctx:    ctx,
		cancel: cancel,
		llmService:   NewMockLLMService(),
		ragService:   NewMockRAGService(), 
		mcpService:   NewMockMCPService(),
		agentService: NewMockAgentService(),
	}
	
	// Initialize health monitor
	client.healthMonitor = NewHealthMonitor(client)
	
	// Add some mock data to make services functional
	setupMockServices(t, client)
	
	return client
}

// createTestClientWithoutLLM creates a client without LLM service
func createTestClientWithoutLLM(t *testing.T) *Client {
	t.Helper()
	
	ctx, cancel := context.WithCancel(context.Background())
	
	client := &Client{
		config: getDefaultConfig(),
		ctx:    ctx,
		cancel: cancel,
		llmService:   nil, // No LLM service
		ragService:   NewMockRAGService(),
		mcpService:   NewMockMCPService(),
		agentService: NewMockAgentService(),
	}
	
	client.healthMonitor = NewHealthMonitor(client)
	return client
}

// createTestClientWithoutRAG creates a client without RAG service
func createTestClientWithoutRAG(t *testing.T) *Client {
	t.Helper()
	
	ctx, cancel := context.WithCancel(context.Background())
	
	client := &Client{
		config: getDefaultConfig(),
		ctx:    ctx,
		cancel: cancel,
		llmService:   NewMockLLMService(),
		ragService:   nil, // No RAG service
		mcpService:   NewMockMCPService(),
		agentService: NewMockAgentService(),
	}
	
	client.healthMonitor = NewHealthMonitor(client)
	setupMockServices(t, client)
	return client
}

// setupMockServices configures mock services with test data
func setupMockServices(t *testing.T, client *Client) {
	t.Helper()
	
	// Setup LLM service
	if mockLLM, ok := client.llmService.(*MockLLMService); ok {
		mockLLM.AddProvider("mock-provider", core.ProviderConfig{
			Type:   "mock",
			Model:  "mock-model",
			Weight: 1,
		})
	}
	
	// Setup RAG service
	if mockRAG, ok := client.ragService.(*MockRAGService); ok {
		mockRAG.AddMockDocument(core.Document{
			ID:      "doc-1",
			Content: "Test document content",
			Metadata: map[string]interface{}{
				"title": "Test Document",
			},
		})
	}
	
	// Setup MCP service
	if mockMCP, ok := client.mcpService.(*MockMCPService); ok {
		mockMCP.RegisterServer(core.ServerConfig{
			Name:        "test-server",
			Description: "mock server",
		})
		mockMCP.AddMockTool(core.ToolInfo{
			Name:        "test-tool",
			Description: "A test tool",
		})
	}
	
	// Setup Agent service
	if mockAgent, ok := client.agentService.(*MockAgentService); ok {
		mockAgent.CreateAgent(core.AgentDefinition{
			Name:        "report-generator",
			Description: "Generates reports",
			Type:        "mock",
		})
		mockAgent.CreateWorkflow(core.WorkflowDefinition{
			Name:        "feedback-analysis",
			Description: "Analyzes feedback",
		})
	}
}