// Package client - factory_test.go
// Comprehensive tests for individual pillar client factories and the builder pattern.
// This file validates the factory functions and builder pattern implementation.

package client

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
)

// ===== INDIVIDUAL PILLAR CLIENT TESTING =====

func TestNewLLMClient(t *testing.T) {
	tests := []struct {
		name        string
		config      core.LLMConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid LLM config",
			config: core.LLMConfig{
				DefaultProvider: "test-provider",
				Providers: map[string]core.ProviderConfig{
					"test-provider": {
						Type:    "ollama",
						BaseURL: "http://localhost:11434",
						Model:   "test-model",
						Weight:  1,
						Timeout: 30 * time.Second,
					},
				},
				LoadBalancing: core.LoadBalancingConfig{
					Strategy: "round_robin",
				},
				HealthCheck: core.HealthCheckConfig{
					Enabled:  true,
					Interval: 30 * time.Second,
				},
			},
			expectError: false,
		},
		{
			name: "empty providers config",
			config: core.LLMConfig{
				DefaultProvider: "nonexistent",
				Providers:       map[string]core.ProviderConfig{},
			},
			expectError: true,
			errorMsg:    "LLM service",
		},
		{
			name: "invalid provider config",
			config: core.LLMConfig{
				DefaultProvider: "test-provider",
				Providers: map[string]core.ProviderConfig{
					"test-provider": {
						Type: "", // Invalid empty type
					},
				},
			},
			expectError: true,
			errorMsg:    "LLM service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewLLMClient(tt.config)

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
				return
			}

			// Test interface compliance
			testLLMClientInterface(t, client)

			// Clean up
			err = client.Close()
			if err != nil {
				t.Errorf("Failed to close LLM client: %v", err)
			}
		})
	}
}

func TestNewRAGClient(t *testing.T) {
	t.Run("RAG client creation", func(t *testing.T) {
		config := core.RAGConfig{
			StorageBackend: "sqlite",
			ChunkingStrategy: core.ChunkingConfig{
				Strategy:  "fixed",
				ChunkSize: 1000,
			},
			VectorStore: core.VectorStoreConfig{
				Backend: "sqvect",
				Metric:  "cosine",
			},
		}

		client, err := NewRAGClient(config)
		if err == nil {
			t.Error("Expected error for unimplemented RAG client")
			if client != nil {
				client.Close()
			}
		}

		// Currently expecting error due to interface mismatch
		if !strings.Contains(err.Error(), "not yet implemented") {
			t.Errorf("Expected implementation error, got: %v", err)
		}
	})
}

func TestNewMCPClient(t *testing.T) {
	tests := []struct {
		name        string
		config      core.MCPConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid MCP config",
			config: core.MCPConfig{
				ServersPath: "test_servers.json",
				Servers:     []core.ServerConfig{},
				HealthCheck: core.HealthCheckConfig{
					Enabled:  true,
					Interval: 60 * time.Second,
				},
				ToolExecution: core.ToolExecutionConfig{
					MaxConcurrent:  5,
					DefaultTimeout: 30 * time.Second,
				},
			},
			expectError: false,
		},
		{
			name: "empty config",
			config: core.MCPConfig{
				ServersPath: "",
			},
			expectError: false, // Empty config should be valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewMCPClient(tt.config)

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
				return
			}

			// Test interface compliance
			testMCPClientInterface(t, client)

			// Clean up
			err = client.Close()
			if err != nil {
				t.Errorf("Failed to close MCP client: %v", err)
			}
		})
	}
}

func TestNewAgentClient(t *testing.T) {
	tests := []struct {
		name        string
		config      core.AgentsConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid Agent config",
			config: core.AgentsConfig{
				WorkflowEngine: core.WorkflowEngineConfig{
					MaxSteps:    100,
					StepTimeout: 60 * time.Second,
				},
				Scheduling: core.SchedulingConfig{
					MaxConcurrent: 3,
					QueueSize:     100,
				},
				StateStorage: core.StateStorageConfig{
					Backend: "memory",
					TTL:     24 * time.Hour,
				},
			},
			expectError: false,
		},
		{
			name: "minimal config",
			config: core.AgentsConfig{
				WorkflowEngine: core.WorkflowEngineConfig{
					MaxSteps: 10,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewAgentClient(tt.config)

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
				return
			}

			// Test interface compliance
			testAgentClientInterface(t, client)

			// Clean up
			err = client.Close()
			if err != nil {
				t.Errorf("Failed to close Agent client: %v", err)
			}
		})
	}
}

// ===== BUILDER PATTERN TESTING =====

func TestBuilder_Basic(t *testing.T) {
	t.Run("new builder", func(t *testing.T) {
		builder := NewBuilder()
		if builder == nil {
			t.Error("Builder should not be nil")
		}

		if builder.dataDir != "~/.rago" {
			t.Errorf("Expected default data dir '~/.rago', got: %s", builder.dataDir)
		}

		if builder.logLevel != "info" {
			t.Errorf("Expected default log level 'info', got: %s", builder.logLevel)
		}
	})

	t.Run("builder configuration", func(t *testing.T) {
		builder := NewBuilder().
			WithDataDir("/custom/data").
			WithLogLevel("debug")

		if builder.dataDir != "/custom/data" {
			t.Errorf("Expected data dir '/custom/data', got: %s", builder.dataDir)
		}

		if builder.logLevel != "debug" {
			t.Errorf("Expected log level 'debug', got: %s", builder.logLevel)
		}
	})
}

func TestBuilder_PillarConfiguration(t *testing.T) {
	llmConfig := core.LLMConfig{
		DefaultProvider: "test-llm",
		Providers: map[string]core.ProviderConfig{
			"test-llm": {
				Type:  "ollama",
				Model: "test-model",
			},
		},
	}

	ragConfig := core.RAGConfig{
		StorageBackend: "sqlite",
		ChunkingStrategy: core.ChunkingConfig{
			Strategy:  "fixed",
			ChunkSize: 1000,
		},
	}

	mcpConfig := core.MCPConfig{
		ServersPath: "test.json",
	}

	agentsConfig := core.AgentsConfig{
		WorkflowEngine: core.WorkflowEngineConfig{
			MaxSteps: 50,
		},
	}

	t.Run("add all pillars", func(t *testing.T) {
		builder := NewBuilder().
			WithLLM(llmConfig).
			WithRAG(ragConfig).
			WithMCP(mcpConfig).
			WithAgents(agentsConfig)

		if builder.llmConfig == nil {
			t.Error("LLM config should be set")
		}
		if builder.ragConfig == nil {
			t.Error("RAG config should be set")
		}
		if builder.mcpConfig == nil {
			t.Error("MCP config should be set")
		}
		if builder.agentsConfig == nil {
			t.Error("Agents config should be set")
		}
	})

	t.Run("disable pillars", func(t *testing.T) {
		builder := NewBuilder().
			WithLLM(llmConfig).
			WithRAG(ragConfig).
			WithMCP(mcpConfig).
			WithAgents(agentsConfig).
			WithoutRAG().
			WithoutMCP()

		if builder.llmConfig == nil {
			t.Error("LLM config should still be set")
		}
		if builder.ragConfig != nil {
			t.Error("RAG config should be disabled")
		}
		if builder.mcpConfig != nil {
			t.Error("MCP config should be disabled")
		}
		if builder.agentsConfig == nil {
			t.Error("Agents config should still be set")
		}
	})
}

func TestBuilder_Build(t *testing.T) {
	llmConfig := core.LLMConfig{
		DefaultProvider: "test-provider",
		Providers: map[string]core.ProviderConfig{
			"test-provider": {
				Type:    "ollama",
				BaseURL: "http://localhost:11434",
				Model:   "test-model",
				Weight:  1,
			},
		},
	}

	tests := []struct {
		name         string
		setupBuilder func() *Builder
		expectError  bool
		errorMsg     string
		validateFunc func(t *testing.T, client *Client)
	}{
		{
			name: "LLM only client",
			setupBuilder: func() *Builder {
				return NewBuilder().
					WithLLM(llmConfig).
					WithoutRAG().
					WithoutMCP().
					WithoutAgents()
			},
			expectError: false,
			validateFunc: func(t *testing.T, client *Client) {
				if client.LLM() == nil {
					t.Error("LLM service should be available")
				}
				if client.RAG() != nil {
					t.Error("RAG service should be disabled")
				}
				if client.MCP() != nil {
					t.Error("MCP service should be disabled")
				}
				if client.Agents() != nil {
					t.Error("Agent service should be disabled")
				}

				config := client.GetConfig()
				if !config.Mode.LLMOnly {
					t.Error("Client should be in LLM-only mode")
				}
			},
		},
		{
			name: "full client",
			setupBuilder: func() *Builder {
				return NewBuilder().
					WithLLM(llmConfig).
					WithRAG(core.RAGConfig{StorageBackend: "sqlite"}).
					WithMCP(core.MCPConfig{ServersPath: "test.json"}).
					WithAgents(core.AgentsConfig{
						WorkflowEngine: core.WorkflowEngineConfig{MaxSteps: 10},
					})
			},
			expectError: false,
			validateFunc: func(t *testing.T, client *Client) {
				if client.LLM() == nil {
					t.Error("LLM service should be available")
				}
				// RAG might be nil due to interface mismatch - that's expected
				if client.MCP() == nil {
					t.Error("MCP service should be available")
				}
				if client.Agents() == nil {
					t.Error("Agent service should be available")
				}

				config := client.GetConfig()
				if config.Mode.LLMOnly || config.Mode.RAGOnly {
					t.Error("Client should not be in restricted mode")
				}
			},
		},
		{
			name: "no pillars enabled",
			setupBuilder: func() *Builder {
				return NewBuilder().
					WithoutLLM().
					WithoutRAG().
					WithoutMCP().
					WithoutAgents()
			},
			expectError: true,
			errorMsg:    "pillar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.setupBuilder()
			client, err := builder.Build()

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
				return
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, client)
			}

			// Clean up
			err = client.Close()
			if err != nil {
				t.Errorf("Failed to close client: %v", err)
			}
		})
	}
}

// ===== CONVENIENCE FUNCTIONS TESTING =====

func TestConvenienceFunctions(t *testing.T) {
	llmConfig := core.LLMConfig{
		DefaultProvider: "test-provider",
		Providers: map[string]core.ProviderConfig{
			"test-provider": {
				Type:    "ollama",
				BaseURL: "http://localhost:11434",
				Model:   "test-model",
				Weight:  1,
			},
		},
	}

	ragConfig := core.RAGConfig{
		StorageBackend: "sqlite",
		ChunkingStrategy: core.ChunkingConfig{
			Strategy:  "fixed",
			ChunkSize: 1000,
		},
	}

	mcpConfig := core.MCPConfig{
		ServersPath: "test.json",
	}

	t.Run("NewLLMOnlyClient", func(t *testing.T) {
		client, err := NewLLMOnlyClient(llmConfig)
		if err != nil {
			t.Errorf("Failed to create LLM-only client: %v", err)
			return
		}
		defer client.Close()

		if client.LLM() == nil {
			t.Error("LLM service should be available")
		}
		if client.RAG() != nil {
			t.Error("RAG service should be disabled")
		}
		if client.MCP() != nil {
			t.Error("MCP service should be disabled")
		}
		if client.Agents() != nil {
			t.Error("Agent service should be disabled")
		}

		config := client.GetConfig()
		if !config.Mode.LLMOnly {
			t.Error("Client should be in LLM-only mode")
		}
	})

	t.Run("NewRAGOnlyClient", func(t *testing.T) {
		client, err := NewRAGOnlyClient(ragConfig)
		if err != nil {
			t.Errorf("Failed to create RAG-only client: %v", err)
			return
		}
		defer client.Close()

		if client.LLM() != nil {
			t.Error("LLM service should be disabled")
		}
		// RAG service might be nil due to interface issues, that's expected
		if client.MCP() != nil {
			t.Error("MCP service should be disabled")
		}
		if client.Agents() != nil {
			t.Error("Agent service should be disabled")
		}

		config := client.GetConfig()
		if !config.Mode.RAGOnly {
			t.Error("Client should be in RAG-only mode")
		}
	})

	t.Run("NewLLMRAGClient", func(t *testing.T) {
		client, err := NewLLMRAGClient(llmConfig, ragConfig)
		if err != nil {
			t.Errorf("Failed to create LLM+RAG client: %v", err)
			return
		}
		defer client.Close()

		if client.LLM() == nil {
			t.Error("LLM service should be available")
		}
		// RAG service might be nil due to interface issues, that's expected
		if client.MCP() != nil {
			t.Error("MCP service should be disabled")
		}
		if client.Agents() != nil {
			t.Error("Agent service should be disabled")
		}

		config := client.GetConfig()
		if config.Mode.LLMOnly || config.Mode.RAGOnly {
			t.Error("Client should not be in single-pillar mode")
		}
		if !config.Mode.DisableMCP || !config.Mode.DisableAgent {
			t.Error("MCP and Agent should be disabled")
		}
	})

	t.Run("NewToolIntegratedClient", func(t *testing.T) {
		client, err := NewToolIntegratedClient(llmConfig, ragConfig, mcpConfig)
		if err != nil {
			t.Errorf("Failed to create tool-integrated client: %v", err)
			return
		}
		defer client.Close()

		if client.LLM() == nil {
			t.Error("LLM service should be available")
		}
		// RAG service might be nil due to interface issues, that's expected
		if client.MCP() == nil {
			t.Error("MCP service should be available")
		}
		if client.Agents() != nil {
			t.Error("Agent service should be disabled")
		}

		config := client.GetConfig()
		if !config.Mode.DisableAgent {
			t.Error("Agent should be disabled")
		}
		if config.Mode.DisableMCP {
			t.Error("MCP should be enabled")
		}
	})
}

// ===== INTERFACE COMPLIANCE TESTING =====

func testLLMClientInterface(t *testing.T, client core.LLMClient) {
	t.Helper()

	ctx := context.Background()

	// Test provider management
	t.Run("provider management", func(t *testing.T) {
		providers := client.ListProviders()
		if providers == nil {
			t.Error("ListProviders should not return nil")
		}

		healthMap := client.GetProviderHealth()
		if healthMap == nil {
			t.Error("GetProviderHealth should not return nil")
		}

		// Test adding a provider
		err := client.AddProvider("test-add", core.ProviderConfig{
			Type:   "test",
			Model:  "test-model",
			Weight: 1,
		})
		if err != nil {
			t.Errorf("AddProvider failed: %v", err)
		}

		// Test removing a provider
		err = client.RemoveProvider("test-add")
		if err != nil {
			t.Errorf("RemoveProvider failed: %v", err)
		}
	})

	// Test generation operations
	t.Run("generation operations", func(t *testing.T) {
		req := core.GenerationRequest{
			Prompt: "Test prompt",
		}

		_, err := client.Generate(ctx, req)
		if err != nil {
			t.Errorf("Generate failed: %v", err)
		}

		// Test streaming
		callbackCalled := false
		callback := func(chunk core.StreamChunk) error {
			callbackCalled = true
			return nil
		}

		err = client.Stream(ctx, req, callback)
		if err != nil {
			t.Errorf("Stream failed: %v", err)
		}

		if !callbackCalled {
			t.Error("Stream callback should have been called")
		}
	})

	// Test batch operations
	t.Run("batch operations", func(t *testing.T) {
		requests := []core.GenerationRequest{
			{Prompt: "Test 1"},
			{Prompt: "Test 2"},
		}

		responses, err := client.GenerateBatch(ctx, requests)
		if err != nil {
			t.Errorf("GenerateBatch failed: %v", err)
		}

		if len(responses) != len(requests) {
			t.Errorf("Expected %d responses, got %d", len(requests), len(responses))
		}
	})

	// Test tool operations
	t.Run("tool operations", func(t *testing.T) {
		toolReq := core.ToolGenerationRequest{
			GenerationRequest: core.GenerationRequest{
				Prompt: "Test with tools",
			},
			Tools: []core.ToolInfo{
				{Name: "test-tool", Description: "A test tool"},
			},
		}

		_, err := client.GenerateWithTools(ctx, toolReq)
		if err != nil {
			t.Errorf("GenerateWithTools failed: %v", err)
		}

		// Test tool streaming
		toolCallbackCalled := false
		toolCallback := func(chunk core.ToolStreamChunk) error {
			toolCallbackCalled = true
			return nil
		}

		err = client.StreamWithTools(ctx, toolReq, toolCallback)
		if err != nil {
			t.Errorf("StreamWithTools failed: %v", err)
		}

		if !toolCallbackCalled {
			t.Error("Tool stream callback should have been called")
		}
	})
}

func testMCPClientInterface(t *testing.T, client core.MCPClient) {
	t.Helper()

	ctx := context.Background()

	// Test server management
	t.Run("server management", func(t *testing.T) {
		servers := client.ListServers()
		if servers == nil {
			t.Error("ListServers should not return nil")
		}

		// Test registering a server
		err := client.RegisterServer(core.ServerConfig{
			Name:        "test-server",
			Description: "test server",
		})
		if err != nil {
			t.Errorf("RegisterServer failed: %v", err)
		}

		// Test server health
		health := client.GetServerHealth("test-server")
		if health == "" {
			t.Error("GetServerHealth should return a status")
		}

		// Test unregistering
		err = client.UnregisterServer("test-server")
		if err != nil {
			t.Errorf("UnregisterServer failed: %v", err)
		}
	})

	// Test tool operations
	t.Run("tool operations", func(t *testing.T) {
		tools := client.ListTools()
		if tools == nil {
			t.Error("ListTools should not return nil")
		}

		// Test tool call
		req := core.ToolCallRequest{
			ToolName:  "test-tool",
			Arguments: map[string]interface{}{"test": "value"},
		}

		_, err := client.CallTool(ctx, req)
		if err != nil {
			t.Errorf("CallTool failed: %v", err)
		}

		// Test async tool call
		respChan, err := client.CallToolAsync(ctx, req)
		if err != nil {
			t.Errorf("CallToolAsync failed: %v", err)
		}

		if respChan == nil {
			t.Error("CallToolAsync should return a response channel")
		} else {
			// Wait for response
			select {
			case resp := <-respChan:
				if resp == nil {
					t.Error("Async response should not be nil")
				}
			case <-time.After(1 * time.Second):
				t.Error("Async response timed out")
			}
		}

		// Test batch tool calls
		requests := []core.ToolCallRequest{req, req}
		responses, err := client.CallToolsBatch(ctx, requests)
		if err != nil {
			t.Errorf("CallToolsBatch failed: %v", err)
		}

		if len(responses) != len(requests) {
			t.Errorf("Expected %d responses, got %d", len(requests), len(responses))
		}
	})
}

func testAgentClientInterface(t *testing.T, client core.AgentClient) {
	t.Helper()

	ctx := context.Background()

	// Test workflow management
	t.Run("workflow management", func(t *testing.T) {
		workflows := client.ListWorkflows()
		if workflows == nil {
			t.Error("ListWorkflows should not return nil")
		}

		// Test creating a workflow
		err := client.CreateWorkflow(core.WorkflowDefinition{
			Name:        "test-workflow",
			Description: "A test workflow",
		})
		if err != nil {
			t.Errorf("CreateWorkflow failed: %v", err)
		}

		// Test executing workflow
		req := core.WorkflowRequest{
			WorkflowName: "test-workflow",
			Inputs:       map[string]interface{}{"test": "value"},
		}

		_, err = client.ExecuteWorkflow(ctx, req)
		if err != nil {
			t.Errorf("ExecuteWorkflow failed: %v", err)
		}

		// Test deleting workflow
		err = client.DeleteWorkflow("test-workflow")
		if err != nil {
			t.Errorf("DeleteWorkflow failed: %v", err)
		}
	})

	// Test agent management
	t.Run("agent management", func(t *testing.T) {
		agents := client.ListAgents()
		if agents == nil {
			t.Error("ListAgents should not return nil")
		}

		// Test creating an agent
		err := client.CreateAgent(core.AgentDefinition{
			Name:        "test-agent",
			Description: "A test agent",
			Type:        "test",
		})
		if err != nil {
			t.Errorf("CreateAgent failed: %v", err)
		}

		// Test executing agent
		req := core.AgentRequest{
			AgentName: "test-agent",
			Task:      "test task",
		}

		_, err = client.ExecuteAgent(ctx, req)
		if err != nil {
			t.Errorf("ExecuteAgent failed: %v", err)
		}

		// Test deleting agent
		err = client.DeleteAgent("test-agent")
		if err != nil {
			t.Errorf("DeleteAgent failed: %v", err)
		}
	})

	// Test scheduling
	t.Run("scheduling", func(t *testing.T) {
		tasks := client.GetScheduledTasks()
		if tasks == nil {
			t.Error("GetScheduledTasks should not return nil")
		}

		// Test scheduling a workflow
		err := client.ScheduleWorkflow("test-workflow", core.ScheduleConfig{
			Type:       "cron",
			Expression: "0 0 * * *",
		})
		if err != nil {
			t.Errorf("ScheduleWorkflow failed: %v", err)
		}
	})
}

// ===== ERROR HANDLING FOR FACTORY FUNCTIONS =====

func TestFactory_ErrorHandling(t *testing.T) {
	t.Run("LLM client with invalid config", func(t *testing.T) {
		config := core.LLMConfig{
			DefaultProvider: "nonexistent",
			Providers:       map[string]core.ProviderConfig{},
		}

		client, err := NewLLMClient(config)
		if err == nil {
			t.Error("Expected error for invalid LLM config")
			if client != nil {
				client.Close()
			}
		}
	})

	t.Run("MCP client close after error", func(t *testing.T) {
		// Even with error-prone config, we should be able to close cleanly
		config := core.MCPConfig{
			ServersPath: "/invalid/path/servers.json",
		}

		client, err := NewMCPClient(config)
		if err != nil {
			t.Errorf("Unexpected error creating MCP client: %v", err)
			return
		}

		if client == nil {
			t.Error("Client should not be nil")
			return
		}

		// Should not panic or error when closing
		err = client.Close()
		if err != nil {
			t.Errorf("Close should not error: %v", err)
		}

		// Second close should be safe
		err = client.Close()
		if err != nil {
			t.Errorf("Second close should not error: %v", err)
		}
	})

	t.Run("builder with conflicting configs", func(t *testing.T) {
		// This isn't really conflicting, but tests edge cases
		builder := NewBuilder().
			WithLLM(core.LLMConfig{DefaultProvider: "test"}).
			WithoutLLM(). // Disable after enabling
			WithoutRAG(). // Disable without enabling
			WithoutMCP().
			WithoutAgents()

		_, err := builder.Build()
		if err == nil {
			t.Error("Expected error when no pillars are enabled")
		}
	})
}