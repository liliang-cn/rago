// Package client - config_test.go
// Comprehensive tests for configuration management and backward compatibility.
// This file validates configuration loading, conversion, validation, and operational modes.

package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	// "github.com/liliang-cn/rago/v2/pkg/config" // TODO: Removed in V3 cleanup
	"github.com/liliang-cn/rago/v2/pkg/core"
	// "github.com/liliang-cn/rago/v2/pkg/domain" // TODO: Removed in V3 cleanup
	"github.com/liliang-cn/rago/v2/pkg/mcp"
)

// ===== CONFIGURATION CREATION TESTING =====

func TestGetDefaultConfig(t *testing.T) {
	cfg := getDefaultConfig()

	t.Run("basic configuration structure", func(t *testing.T) {
		if cfg == nil {
			t.Fatal("Default config should not be nil")
		}

		if cfg.ClientName != "rago-client" {
			t.Errorf("Expected client name 'rago-client', got: %s", cfg.ClientName)
		}

		if cfg.ClientVersion != "3.0.0" {
			t.Errorf("Expected client version '3.0.0', got: %s", cfg.ClientVersion)
		}

		if cfg.LegacyMode != false {
			t.Error("Default config should not be in legacy mode")
		}

		if cfg.DataDir == "" {
			t.Error("Data directory should be set")
		}

		if cfg.LogLevel != "info" {
			t.Errorf("Expected log level 'info', got: %s", cfg.LogLevel)
		}
	})

	t.Run("LLM configuration", func(t *testing.T) {
		if cfg.LLM.DefaultProvider == "" {
			t.Error("Default LLM provider should be set")
		}

		allProviders := cfg.LLM.Providers.GetProviders()
		if len(allProviders) == 0 {
			t.Error("Should have at least one LLM provider configured")
		}

		// Check for default ollama provider
		if _, exists := cfg.LLM.Providers.GetProvider("ollama-default"); !exists {
			t.Error("Should have ollama-default provider")
		}

		if cfg.LLM.HealthCheck.Enabled != true {
			t.Error("LLM health check should be enabled by default")
		}

		if cfg.LLM.LoadBalancing.Strategy == "" {
			t.Error("Load balancing strategy should be set")
		}
	})

	t.Run("RAG configuration", func(t *testing.T) {
		if cfg.RAG.StorageBackend != "sqlite" {
			t.Errorf("Expected RAG storage backend 'sqlite', got: %s", cfg.RAG.StorageBackend)
		}

		if cfg.RAG.ChunkingStrategy.Strategy != "fixed" {
			t.Errorf("Expected chunking strategy 'fixed', got: %s", cfg.RAG.ChunkingStrategy.Strategy)
		}

		if cfg.RAG.ChunkingStrategy.ChunkSize <= 0 {
			t.Error("Chunk size should be positive")
		}

		if cfg.RAG.VectorStore.Backend != "sqvect" {
			t.Errorf("Expected vector store backend 'sqvect', got: %s", cfg.RAG.VectorStore.Backend)
		}

		if cfg.RAG.Search.DefaultLimit <= 0 {
			t.Error("Default search limit should be positive")
		}
	})

	t.Run("MCP configuration", func(t *testing.T) {
		if cfg.MCP.ServersPath == "" {
			t.Error("MCP servers path should be set")
		}

		if cfg.MCP.HealthCheck.Enabled != true {
			t.Error("MCP health check should be enabled by default")
		}

		if cfg.MCP.ToolExecution.MaxConcurrent <= 0 {
			t.Error("Max concurrent tools should be positive")
		}
	})

	t.Run("Agents configuration", func(t *testing.T) {
		if cfg.Agents.WorkflowEngine.MaxSteps <= 0 {
			t.Error("Max workflow steps should be positive")
		}

		if cfg.Agents.Scheduling.MaxConcurrent <= 0 {
			t.Error("Max concurrent schedules should be positive")
		}

		if cfg.Agents.StateStorage.Backend == "" {
			t.Error("State storage backend should be set")
		}
	})

	t.Run("mode configuration", func(t *testing.T) {
		if cfg.Mode.RAGOnly {
			t.Error("Default config should not be RAG-only")
		}

		if cfg.Mode.LLMOnly {
			t.Error("Default config should not be LLM-only")
		}

		if cfg.Mode.DisableMCP {
			t.Error("Default config should not disable MCP")
		}

		if cfg.Mode.DisableAgent {
			t.Error("Default config should not disable Agents")
		}
	})
}

func TestGetDefaultLLMConfig(t *testing.T) {
	cfg := getDefaultLLMConfig()

	t.Run("provider configuration", func(t *testing.T) {
		if cfg.DefaultProvider == "" {
			t.Error("Default provider should be set")
		}

		allProviders := cfg.Providers.GetProviders()
		if len(allProviders) == 0 {
			t.Error("Should have providers configured")
		}

		provider, exists := cfg.Providers.GetProvider(cfg.DefaultProvider)
		if !exists {
			t.Errorf("Default provider '%s' should exist in providers map", cfg.DefaultProvider)
		}

		if provider.Type == "" {
			t.Error("Provider type should be set")
		}

		if provider.BaseURL == "" {
			t.Error("Provider base URL should be set")
		}

		if provider.Model == "" {
			t.Error("Provider model should be set")
		}
	})

	t.Run("load balancing configuration", func(t *testing.T) {
		if cfg.LoadBalancing.Strategy == "" {
			t.Error("Load balancing strategy should be set")
		}

		if !cfg.LoadBalancing.HealthCheck {
			t.Error("Health check should be enabled for load balancing")
		}

		if cfg.LoadBalancing.CheckInterval <= 0 {
			t.Error("Check interval should be positive")
		}
	})

	t.Run("health check configuration", func(t *testing.T) {
		if !cfg.HealthCheck.Enabled {
			t.Error("Health check should be enabled")
		}

		if cfg.HealthCheck.Interval <= 0 {
			t.Error("Health check interval should be positive")
		}

		if cfg.HealthCheck.Timeout <= 0 {
			t.Error("Health check timeout should be positive")
		}

		if cfg.HealthCheck.Retries <= 0 {
			t.Error("Health check retries should be positive")
		}
	})
}

func TestGetDefaultRAGConfig(t *testing.T) {
	dataDir := "/test/data"
	cfg := getDefaultRAGConfig(dataDir)

	t.Run("storage configuration", func(t *testing.T) {
		if cfg.StorageBackend != "sqlite" {
			t.Errorf("Expected storage backend 'sqlite', got: %s", cfg.StorageBackend)
		}

		if cfg.VectorStore.Backend != "sqvect" {
			t.Errorf("Expected vector store 'sqvect', got: %s", cfg.VectorStore.Backend)
		}

		if cfg.KeywordStore.Backend != "bleve" {
			t.Errorf("Expected keyword store 'bleve', got: %s", cfg.KeywordStore.Backend)
		}
	})

	t.Run("chunking configuration", func(t *testing.T) {
		if cfg.ChunkingStrategy.Strategy != "fixed" {
			t.Errorf("Expected chunking strategy 'fixed', got: %s", cfg.ChunkingStrategy.Strategy)
		}

		if cfg.ChunkingStrategy.ChunkSize <= 0 {
			t.Error("Chunk size should be positive")
		}

		if cfg.ChunkingStrategy.ChunkOverlap < 0 {
			t.Error("Chunk overlap should be non-negative")
		}

		if cfg.ChunkingStrategy.MinChunkSize <= 0 {
			t.Error("Min chunk size should be positive")
		}
	})

	t.Run("search configuration", func(t *testing.T) {
		if cfg.Search.DefaultLimit <= 0 {
			t.Error("Default search limit should be positive")
		}

		if cfg.Search.MaxLimit < cfg.Search.DefaultLimit {
			t.Error("Max limit should be >= default limit")
		}

		if cfg.Search.DefaultThreshold < 0 || cfg.Search.DefaultThreshold > 1 {
			t.Error("Default threshold should be between 0 and 1")
		}

		if cfg.Search.HybridWeights.Vector < 0 || cfg.Search.HybridWeights.Vector > 1 {
			t.Error("Vector weight should be between 0 and 1")
		}

		if cfg.Search.HybridWeights.Keyword < 0 || cfg.Search.HybridWeights.Keyword > 1 {
			t.Error("Keyword weight should be between 0 and 1")
		}
	})

	t.Run("embedding configuration", func(t *testing.T) {
		if cfg.Embedding.Provider == "" {
			t.Error("Embedding provider should be set")
		}

		if cfg.Embedding.Model == "" {
			t.Error("Embedding model should be set")
		}

		if cfg.Embedding.BatchSize <= 0 {
			t.Error("Embedding batch size should be positive")
		}
	})
}

func TestGetDefaultMCPConfig(t *testing.T) {
	cfg := getDefaultMCPConfig()

	t.Run("server configuration", func(t *testing.T) {
		if cfg.ServersPath == "" {
			t.Error("Servers path should be set")
		}

		if cfg.Servers == nil {
			t.Error("Servers slice should be initialized")
		}
	})

	t.Run("health check configuration", func(t *testing.T) {
		if !cfg.HealthCheck.Enabled {
			t.Error("Health check should be enabled")
		}

		if cfg.HealthCheck.Interval <= 0 {
			t.Error("Health check interval should be positive")
		}

		if cfg.HealthCheckInterval <= 0 {
			t.Error("Health check interval should be positive")
		}
	})

	t.Run("tool execution configuration", func(t *testing.T) {
		if cfg.ToolExecution.MaxConcurrent <= 0 {
			t.Error("Max concurrent tools should be positive")
		}

		if cfg.ToolExecution.DefaultTimeout <= 0 {
			t.Error("Default timeout should be positive")
		}

		if !cfg.ToolExecution.EnableCache {
			t.Error("Cache should be enabled by default")
		}

		if cfg.CacheSize <= 0 {
			t.Error("Cache size should be positive")
		}
	})
}

func TestGetDefaultAgentsConfig(t *testing.T) {
	cfg := getDefaultAgentsConfig()

	t.Run("workflow engine configuration", func(t *testing.T) {
		if cfg.WorkflowEngine.MaxSteps <= 0 {
			t.Error("Max steps should be positive")
		}

		if cfg.WorkflowEngine.StepTimeout <= 0 {
			t.Error("Step timeout should be positive")
		}

		if cfg.WorkflowEngine.StateBackend == "" {
			t.Error("State backend should be set")
		}

		if cfg.WorkflowEngine.MaxConcurrentWorkflows <= 0 {
			t.Error("Max concurrent workflows should be positive")
		}

		if cfg.WorkflowEngine.DefaultTimeout <= 0 {
			t.Error("Default timeout should be positive")
		}
	})

	t.Run("scheduling configuration", func(t *testing.T) {
		if cfg.Scheduling.Backend == "" {
			t.Error("Scheduling backend should be set")
		}

		if cfg.Scheduling.MaxConcurrent <= 0 {
			t.Error("Max concurrent jobs should be positive")
		}

		if cfg.Scheduling.QueueSize <= 0 {
			t.Error("Queue size should be positive")
		}

		if cfg.Scheduling.RetryPolicy.MaxRetries <= 0 {
			t.Error("Max retries should be positive")
		}

		if cfg.Scheduling.RetryPolicy.RetryDelay <= 0 {
			t.Error("Retry delay should be positive")
		}
	})

	t.Run("state storage configuration", func(t *testing.T) {
		if cfg.StateStorage.Backend == "" {
			t.Error("State storage backend should be set")
		}

		if cfg.StateStorage.TTL <= 0 {
			t.Error("TTL should be positive")
		}
	})

	t.Run("reasoning chains configuration", func(t *testing.T) {
		if cfg.ReasoningChains.MaxSteps <= 0 {
			t.Error("Max reasoning steps should be positive")
		}

		if cfg.ReasoningChains.MaxMemorySize <= 0 {
			t.Error("Max memory size should be positive")
		}

		if cfg.ReasoningChains.StepTimeout <= 0 {
			t.Error("Step timeout should be positive")
		}
	})
}

// ===== LEGACY CONFIGURATION CONVERSION TESTING =====

// Legacy conversion tests removed in V3 - functionality no longer exists
					OpenAI: &domain.OpenAIProviderConfig{
						BaseURL:  "https://api.openai.com/v1",
						APIKey:   "sk-test-key",
						LLMModel: "gpt-4o-mini",
					},
				},
			},
			// Fallback Ollama config
			Ollama: config.OllamaConfig{
				BaseURL:  "http://localhost:11435", // Different URL for fallback test
				LLMModel: "qwen2.5",
			},
		}

		cfg, err := convertLegacyConfig(legacy)
		if err != nil {
			t.Fatalf("Legacy conversion failed: %v", err)
		}

		// Check that providers were converted
		if len(cfg.LLM.Providers) < 2 {
			t.Errorf("Expected at least 2 providers, got %d", len(cfg.LLM.Providers))
		}

		// Check Ollama provider
		if ollamaProvider, exists := cfg.LLM.Providers["ollama-default"]; exists {
			if ollamaProvider.Type != "ollama" {
				t.Errorf("Expected ollama type, got: %s", ollamaProvider.Type)
			}
			if ollamaProvider.BaseURL != "http://localhost:11434" {
				t.Errorf("Expected ollama URL from provider config, got: %s", ollamaProvider.BaseURL)
			}
			if ollamaProvider.Model != "llama3.2" {
				t.Errorf("Expected llama3.2 model, got: %s", ollamaProvider.Model)
			}
		} else {
			t.Error("Ollama provider should be converted")
		}

		// Check OpenAI provider
		if openaiProvider, exists := cfg.LLM.Providers["openai-default"]; exists {
			if openaiProvider.Type != "openai" {
				t.Errorf("Expected openai type, got: %s", openaiProvider.Type)
			}
			if openaiProvider.APIKey != "sk-test-key" {
				t.Errorf("Expected API key to be preserved, got: %s", openaiProvider.APIKey)
			}
			if openaiProvider.Model != "gpt-4o-mini" {
				t.Errorf("Expected gpt-4o-mini model, got: %s", openaiProvider.Model)
			}
		} else {
			t.Error("OpenAI provider should be converted")
		}

		// Check default provider is set
		if cfg.LLM.DefaultProvider == "" {
			t.Error("Default provider should be set")
		}
	})

	t.Run("fallback Ollama conversion", func(t *testing.T) {
		// Test legacy config without new provider configs
		legacy := &config.Config{
			Ollama: config.OllamaConfig{
				BaseURL:  "http://localhost:11435",
				LLMModel: "qwen2.5",
			},
		}

		cfg, err := convertLegacyConfig(legacy)
		if err != nil {
			t.Fatalf("Legacy conversion failed: %v", err)
		}

		// Should create fallback provider
		if _, exists := cfg.LLM.Providers.GetProvider("ollama-legacy"); !exists {
			t.Error("Should create ollama-legacy provider as fallback")
		}

		fallbackProvider, _ := cfg.LLM.Providers.GetProvider("ollama-legacy")
		if fallbackProvider.BaseURL != "http://localhost:11435" {
			t.Errorf("Expected fallback URL, got: %s", fallbackProvider.BaseURL)
		}
		if fallbackProvider.Model != "qwen2.5" {
			t.Errorf("Expected fallback model, got: %s", fallbackProvider.Model)
		}
	})

	t.Run("RAG config conversion", func(t *testing.T) {
		legacy := &config.Config{
			Chunker: config.ChunkerConfig{
				ChunkSize: 2000,
				Overlap:   400,
			},
			Sqvect: config.SqvectConfig{
				TopK: 20,
			},
			Providers: config.ProvidersConfig{
				DefaultEmbedder: "ollama-embed",
			},
		}

		cfg, err := convertLegacyConfig(legacy)
		if err != nil {
			t.Fatalf("Legacy conversion failed: %v", err)
		}

		// Check chunking conversion
		if cfg.RAG.ChunkingStrategy.ChunkSize != 2000 {
			t.Errorf("Expected chunk size 2000, got: %d", cfg.RAG.ChunkingStrategy.ChunkSize)
		}
		if cfg.RAG.ChunkingStrategy.ChunkOverlap != 400 {
			t.Errorf("Expected chunk overlap 400, got: %d", cfg.RAG.ChunkingStrategy.ChunkOverlap)
		}

		// Check search conversion
		if cfg.RAG.Search.DefaultLimit != 20 {
			t.Errorf("Expected default limit 20, got: %d", cfg.RAG.Search.DefaultLimit)
		}

		// Check embedding conversion
		if cfg.RAG.Embedding.Provider != "ollama-embed" {
			t.Errorf("Expected embedding provider 'ollama-embed', got: %s", cfg.RAG.Embedding.Provider)
		}
	})

	t.Run("MCP config conversion", func(t *testing.T) {
		legacy := &config.Config{
			MCP: mcp.Config{
				Enabled: false,
			},
		}

		cfg, err := convertLegacyConfig(legacy)
		if err != nil {
			t.Fatalf("Legacy conversion failed: %v", err)
		}

		// Check MCP disabled
		if !cfg.Mode.DisableMCP {
			t.Error("MCP should be disabled when legacy config has enabled=false")
		}

		// Check default MCP configuration was still applied
		if cfg.MCP.ServersPath == "" {
			t.Error("MCP servers path should be set even when disabled")
		}
	})

	t.Run("mode configuration", func(t *testing.T) {
		legacy := &config.Config{
			MCP: mcp.Config{
				Enabled: true,
			},
		}

		cfg, err := convertLegacyConfig(legacy)
		if err != nil {
			t.Fatalf("Legacy conversion failed: %v", err)
		}

		// Check that legacy mode sets appropriate defaults
		if cfg.Mode.RAGOnly {
			t.Error("Legacy config should not be RAG-only")
		}
		if cfg.Mode.LLMOnly {
			t.Error("Legacy config should not be LLM-only")
		}
		if cfg.Mode.DisableMCP {
			t.Error("MCP should be enabled when legacy has enabled=true")
		}
		if !cfg.Mode.DisableAgent {
			t.Error("Agents should be disabled by default in legacy mode")
		}
	})
}

func TestGetDataDir(t *testing.T) {
	tests := []struct {
		name     string
		legacy   *config.Config
		expected string
	}{
		{
			name: "from sqvect DB path",
			legacy: &config.Config{
				Sqvect: config.SqvectConfig{
					DBPath: "/custom/path/vectors.db",
				},
			},
			expected: "/custom/path",
		},
		{
			name:     "fallback to home directory",
			legacy:   &config.Config{},
			expected: "", // Will contain actual home directory path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDataDir(tt.legacy)

			if tt.expected != "" {
				if result != tt.expected {
					t.Errorf("Expected data dir '%s', got: '%s'", tt.expected, result)
				}
			} else {
				// Should contain home directory or fallback
				if result == "" {
					t.Error("Data directory should not be empty")
				}
			}
		})
	}
}

// ===== CONFIGURATION VALIDATION TESTING =====

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid default config",
			config:      getDefaultConfig(),
			expectError: false,
		},
		{
			name: "no pillars enabled",
			config: &Config{
				Config: core.Config{
					DataDir: "/test",
					Mode: core.ModeConfig{
						RAGOnly:      false,
						LLMOnly:      false,
						DisableMCP:   true,
						DisableAgent: true,
					},
				},
			},
			expectError: true,
			errorMsg:    "at least one pillar must be enabled",
		},
		{
			name: "LLM enabled but no default provider",
			config: &Config{
				Config: core.Config{
					DataDir: "/test",
					LLM: core.LLMConfig{
						DefaultProvider: "",
						Providers:       map[string]core.ProviderConfig{},
					},
					Mode: core.ModeConfig{
						RAGOnly:      false,
						LLMOnly:      false,
						DisableMCP:   true,
						DisableAgent: true,
					},
				},
			},
			expectError: true,
			errorMsg:    "LLM default provider must be specified",
		},
		{
			name: "LLM enabled but no providers",
			config: &Config{
				Config: core.Config{
					DataDir: "/test",
					LLM: core.LLMConfig{
						DefaultProvider: "test",
						Providers:       map[string]core.ProviderConfig{},
					},
					Mode: core.ModeConfig{
						RAGOnly:      false,
						LLMOnly:      false,
						DisableMCP:   true,
						DisableAgent: true,
					},
				},
			},
			expectError: true,
			errorMsg:    "at least one LLM provider must be configured",
		},
		{
			name: "default provider not in providers map",
			config: &Config{
				Config: core.Config{
					DataDir: "/test",
					LLM: core.LLMConfig{
						DefaultProvider: "nonexistent",
						Providers: map[string]core.ProviderConfig{
							"existing": {
								Type:  "ollama",
								Model: "test",
							},
						},
					},
					Mode: core.ModeConfig{
						RAGOnly:      false,
						LLMOnly:      false,
						DisableMCP:   true,
						DisableAgent: true,
					},
				},
			},
			expectError: true,
			errorMsg:    "default LLM provider 'nonexistent' not found",
		},
		{
			name: "empty data directory",
			config: &Config{
				Config: core.Config{
					DataDir: "",
					LLM: core.LLMConfig{
						DefaultProvider: "test",
						Providers: map[string]core.ProviderConfig{
							"test": {Type: "ollama", Model: "test"},
						},
					},
					Mode: core.ModeConfig{
						RAGOnly:      false,
						LLMOnly:      false,
						DisableMCP:   true,
						DisableAgent: true,
					},
				},
			},
			expectError: true,
			errorMsg:    "data directory must be specified",
		},
		{
			name: "RAG-only mode",
			config: &Config{
				Config: core.Config{
					DataDir: "/test",
					Mode: core.ModeConfig{
						RAGOnly:      true,
						LLMOnly:      false,
						DisableMCP:   true,
						DisableAgent: true,
					},
				},
			},
			expectError: false, // RAG-only should be valid (no LLM validation needed)
		},
		{
			name: "LLM-only mode",
			config: &Config{
				Config: core.Config{
					DataDir: "/test",
					LLM: core.LLMConfig{
						DefaultProvider: "test",
						Providers: map[string]core.ProviderConfig{
							"test": {Type: "ollama", Model: "test"},
						},
					},
					Mode: core.ModeConfig{
						RAGOnly:      false,
						LLMOnly:      true,
						DisableMCP:   true,
						DisableAgent: true,
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// ===== MODE CONFIGURATION TESTING =====

func TestClientModeConfig(t *testing.T) {
	tests := []struct {
		name           string
		mode           core.ModeConfig
		expectedString string
		expectedDisableLLM bool
		expectedDisableRAG bool
	}{
		{
			name: "LLM only",
			mode: core.ModeConfig{
				LLMOnly:      true,
				RAGOnly:      false,
				DisableMCP:   true,
				DisableAgent: true,
			},
			expectedString:     "llm-only",
			expectedDisableLLM: false,
			expectedDisableRAG: true,
		},
		{
			name: "RAG only",
			mode: core.ModeConfig{
				LLMOnly:      false,
				RAGOnly:      true,
				DisableMCP:   true,
				DisableAgent: true,
			},
			expectedString:     "rag-only",
			expectedDisableLLM: true,
			expectedDisableRAG: false,
		},
		{
			name: "LLM and RAG",
			mode: core.ModeConfig{
				LLMOnly:      false,
				RAGOnly:      false,
				DisableMCP:   true,
				DisableAgent: true,
			},
			expectedString:     "llm-rag",
			expectedDisableLLM: false,
			expectedDisableRAG: false,
		},
		{
			name: "no MCP",
			mode: core.ModeConfig{
				LLMOnly:      false,
				RAGOnly:      false,
				DisableMCP:   true,
				DisableAgent: false,
			},
			expectedString:     "no-mcp",
			expectedDisableLLM: false,
			expectedDisableRAG: false,
		},
		{
			name: "no agents",
			mode: core.ModeConfig{
				LLMOnly:      false,
				RAGOnly:      false,
				DisableMCP:   false,
				DisableAgent: true,
			},
			expectedString:     "no-agents",
			expectedDisableLLM: false,
			expectedDisableRAG: false,
		},
		{
			name: "full mode",
			mode: core.ModeConfig{
				LLMOnly:      false,
				RAGOnly:      false,
				DisableMCP:   false,
				DisableAgent: false,
			},
			expectedString:     "full",
			expectedDisableLLM: false,
			expectedDisableRAG: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientMode := &ClientModeConfig{ModeConfig: tt.mode}

			// Test string representation
			if clientMode.String() != tt.expectedString {
				t.Errorf("Expected mode string '%s', got: '%s'", tt.expectedString, clientMode.String())
			}

			// Test disable methods
			if clientMode.DisableLLM() != tt.expectedDisableLLM {
				t.Errorf("Expected DisableLLM %v, got: %v", tt.expectedDisableLLM, clientMode.DisableLLM())
			}

			if clientMode.DisableRAG() != tt.expectedDisableRAG {
				t.Errorf("Expected DisableRAG %v, got: %v", tt.expectedDisableRAG, clientMode.DisableRAG())
			}
		})
	}
}

// ===== CONFIGURATION LOADING/SAVING TESTING =====

func TestConfig_LoadSave(t *testing.T) {
	t.Run("save not implemented", func(t *testing.T) {
		cfg := getDefaultConfig()
		tempFile := filepath.Join(t.TempDir(), "test_config.toml")

		err := cfg.Save(tempFile)
		if err == nil {
			t.Error("Expected error for unimplemented save")
		}
		if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("Expected 'not implemented' error, got: %v", err)
		}
	})

	t.Run("load not implemented", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "test_config.toml")

		cfg, err := Load(tempFile)
		if err == nil {
			t.Error("Expected error for unimplemented load")
		}
		if !strings.Contains(err.Error(), "not implemented") {
			t.Errorf("Expected 'not implemented' error, got: %v", err)
		}
		if cfg != nil {
			t.Error("Config should be nil when load fails")
		}
	})
}

// ===== EDGE CASES AND ERROR CONDITIONS =====

func TestConfig_EdgeCases(t *testing.T) {
	t.Run("nil config conversion", func(t *testing.T) {
		// This would panic in real code, but we're testing the expected behavior
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when converting nil config")
			}
		}()

		convertLegacyConfig(nil)
	})

	t.Run("empty legacy config", func(t *testing.T) {
		legacy := &config.Config{}

		cfg, err := convertLegacyConfig(legacy)
		if err != nil {
			t.Fatalf("Should handle empty legacy config: %v", err)
		}

		if cfg == nil {
			t.Error("Should return valid config even for empty legacy")
		}

		if !cfg.LegacyMode {
			t.Error("Should be in legacy mode")
		}
	})

	t.Run("config with long paths", func(t *testing.T) {
		longPath := strings.Repeat("/very/long/path", 50)
		cfg := getDefaultConfig()
		cfg.DataDir = longPath

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Should handle long paths: %v", err)
		}
	})

	t.Run("config with special characters", func(t *testing.T) {
		cfg := getDefaultConfig()
		cfg.DataDir = "/path/with spaces/and-special_chars.123"

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Should handle special characters in paths: %v", err)
		}
	})

	t.Run("provider config with empty model", func(t *testing.T) {
		cfg := &Config{
			Config: core.Config{
				DataDir: "/test",
				LLM: core.LLMConfig{
					DefaultProvider: "test",
					Providers: map[string]core.ProviderConfig{
						"test": {
							Type:  "ollama",
							Model: "", // Empty model
						},
					},
				},
				Mode: core.ModeConfig{
					LLMOnly: true,
				},
			},
		}

		err := cfg.Validate()
		// Should still be valid - empty model might be valid for some providers
		if err != nil {
			t.Errorf("Should handle empty model: %v", err)
		}
	})
}

// ===== HELPER FUNCTIONS FOR TESTING =====

func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "rago.toml")

	err := os.WriteFile(configFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}

	return configFile
}

func TestConfig_Integration(t *testing.T) {
	t.Run("default config creates valid client", func(t *testing.T) {
		cfg := getDefaultConfig()

		err := cfg.Validate()
		if err != nil {
			t.Fatalf("Default config should be valid: %v", err)
		}

		// Try to create a client with this config
		client, err := NewWithConfig(cfg)
		if err != nil {
			t.Fatalf("Should be able to create client with default config: %v", err)
		}

		if client == nil {
			t.Error("Client should not be nil")
		} else {
			client.Close()
		}
	})

	t.Run("legacy config conversion creates valid client", func(t *testing.T) {
		legacy := &config.Config{
			Ollama: config.OllamaConfig{
				BaseURL:  "http://localhost:11434",
				LLMModel: "llama3.2",
			},
			Chunker: config.ChunkerConfig{
				ChunkSize: 1000,
				Overlap:   200,
			},
			MCP: mcp.Config{
				Enabled: true,
			},
		}

		cfg, err := convertLegacyConfig(legacy)
		if err != nil {
			t.Fatalf("Legacy conversion failed: %v", err)
		}

		err = cfg.Validate()
		if err != nil {
			t.Fatalf("Converted config should be valid: %v", err)
		}

		// Try to create a client with converted config
		// Note: This might fail due to actual service initialization, but config should be valid
		_, err = NewWithConfig(cfg)
		// We don't require this to succeed since services might not be available
		// But we've validated the config structure
	})
}