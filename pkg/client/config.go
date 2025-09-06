package client

import (
	"time"

	"github.com/liliang-cn/rago/v2/pkg/core"
	// Import the existing config package for backward compatibility
	existingConfig "github.com/liliang-cn/rago/v2/pkg/config"
)

// ConfigAdapter provides methods to load and convert configurations
type ConfigAdapter struct {
	configPath string
}

// NewConfigAdapter creates a new configuration adapter
func NewConfigAdapter(configPath string) *ConfigAdapter {
	return &ConfigAdapter{
		configPath: configPath,
	}
}

// LoadCoreConfig loads the configuration and converts it to the new four-pillar format
func (adapter *ConfigAdapter) LoadCoreConfig() (*core.Config, error) {
	// Load existing configuration
	existingCfg, err := existingConfig.Load(adapter.configPath)
	if err != nil {
		return nil, core.WrapError(err, "failed to load existing configuration")
	}

	// Convert to new format
	return adapter.convertConfig(existingCfg)
}

// convertConfig converts existing configuration to new four-pillar format
func (adapter *ConfigAdapter) convertConfig(existing *existingConfig.Config) (*core.Config, error) {
	config := &core.Config{
		DataDir:  "~/.rago", // Default data directory
		LogLevel: "info",    // Default log level
	}

	// Convert mode configuration
	config.Mode = core.ModeConfig{
		RAGOnly:      existing.Mode.RAGOnly,
		LLMOnly:      false, // New field, default false
		DisableMCP:   existing.Mode.DisableMCP,
		DisableAgent: !existing.Agents.Enabled, // Convert from enabled to disabled
	}

	// Convert LLM configuration
	llmConfig, err := adapter.convertLLMConfig(existing)
	if err != nil {
		return nil, core.WrapError(err, "failed to convert LLM configuration")
	}
	config.LLM = llmConfig

	// Convert RAG configuration
	ragConfig, err := adapter.convertRAGConfig(existing)
	if err != nil {
		return nil, core.WrapError(err, "failed to convert RAG configuration")
	}
	config.RAG = ragConfig

	// Convert MCP configuration
	mcpConfig, err := adapter.convertMCPConfig(existing)
	if err != nil {
		return nil, core.WrapError(err, "failed to convert MCP configuration")
	}
	config.MCP = mcpConfig

	// Convert Agent configuration
	agentConfig, err := adapter.convertAgentConfig(existing)
	if err != nil {
		return nil, core.WrapError(err, "failed to convert Agent configuration")
	}
	config.Agents = agentConfig

	return config, nil
}

// convertLLMConfig converts provider configuration to LLM pillar configuration
func (adapter *ConfigAdapter) convertLLMConfig(existing *existingConfig.Config) (core.LLMConfig, error) {
	config := core.LLMConfig{
		DefaultProvider: existing.Providers.DefaultLLM,
		LoadBalancing: core.LoadBalancingConfig{
			Strategy:      "round_robin",
			HealthCheck:   true,
			CheckInterval: 30 * time.Second,
		},
		Providers:   make(map[string]core.ProviderConfig),
		HealthCheck: core.HealthCheckConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
			Retries:  3,
		},
	}

	// Convert Ollama provider if configured
	if existing.Providers.ProviderConfigs.Ollama != nil {
		ollamaConfig := existing.Providers.ProviderConfigs.Ollama
		config.Providers["ollama"] = core.ProviderConfig{
			Type:    "ollama",
			BaseURL: ollamaConfig.BaseURL,
			Model:   ollamaConfig.LLMModel,
			Weight:  1,
			Timeout: ollamaConfig.Timeout,
			Parameters: map[string]interface{}{
				"embedding_model": ollamaConfig.EmbeddingModel,
			},
		}
	}

	// Convert OpenAI provider if configured
	if existing.Providers.ProviderConfigs.OpenAI != nil {
		openaiConfig := existing.Providers.ProviderConfigs.OpenAI
		config.Providers["openai"] = core.ProviderConfig{
			Type:    "openai",
			BaseURL: openaiConfig.BaseURL,
			APIKey:  openaiConfig.APIKey,
			Model:   openaiConfig.LLMModel,
			Weight:  1,
			Timeout: openaiConfig.Timeout,
			Parameters: map[string]interface{}{
				"embedding_model": openaiConfig.EmbeddingModel,
				"organization":    openaiConfig.Organization,
				"project":         openaiConfig.Project,
			},
		}
	}

	// Convert LM Studio provider if configured
	if existing.Providers.ProviderConfigs.LMStudio != nil {
		lmstudioConfig := existing.Providers.ProviderConfigs.LMStudio
		config.Providers["lmstudio"] = core.ProviderConfig{
			Type:    "lmstudio",
			BaseURL: lmstudioConfig.BaseURL,
			Model:   lmstudioConfig.LLMModel,
			Weight:  1,
			Timeout: lmstudioConfig.Timeout,
			Parameters: map[string]interface{}{
				"embedding_model": lmstudioConfig.EmbeddingModel,
			},
		}
	}

	// Backward compatibility: convert legacy Ollama config if no new providers
	if len(config.Providers) == 0 {
		config.Providers["ollama"] = core.ProviderConfig{
			Type:    "ollama",
			BaseURL: existing.Ollama.BaseURL,
			Model:   existing.Ollama.LLMModel,
			Weight:  1,
			Timeout: existing.Ollama.Timeout,
			Parameters: map[string]interface{}{
				"embedding_model": existing.Ollama.EmbeddingModel,
			},
		}
		config.DefaultProvider = "ollama"
	}

	return config, nil
}

// convertRAGConfig converts existing RAG-related configuration to RAG pillar configuration
func (adapter *ConfigAdapter) convertRAGConfig(existing *existingConfig.Config) (core.RAGConfig, error) {
	config := core.RAGConfig{
		StorageBackend: "dual", // sqvect + bleve
		ChunkingStrategy: core.ChunkingConfig{
			Strategy:     existing.Chunker.Method,
			ChunkSize:    existing.Chunker.ChunkSize,
			ChunkOverlap: existing.Chunker.Overlap,
			MinChunkSize: 100, // Default minimum
		},
		VectorStore: core.VectorStoreConfig{
			Backend:    "sqvect",
			Dimensions: 0, // Will be determined by model
			Metric:     "cosine",
			IndexType:  "hnsw",
		},
		KeywordStore: core.KeywordStoreConfig{
			Backend:   "bleve",
			Analyzer:  "standard",
			Languages: []string{"en"},
			Stemming:  true,
		},
		Search: core.SearchConfig{
			DefaultLimit:     existing.Sqvect.TopK,
			MaxLimit:         100,
			DefaultThreshold: float32(existing.Sqvect.Threshold),
			HybridWeights: struct {
				Vector  float32 `toml:"vector"`
				Keyword float32 `toml:"keyword"`
			}{
				Vector:  0.7,
				Keyword: 0.3,
			},
		},
		Embedding: core.EmbeddingConfig{
			Provider:   existing.Providers.DefaultEmbedder,
			Dimensions: 0, // Will be determined by model
			BatchSize:  existing.Sqvect.BatchSize,
		},
	}

	// Set model based on provider configuration
	if existing.Providers.ProviderConfigs.Ollama != nil {
		config.Embedding.Model = existing.Providers.ProviderConfigs.Ollama.EmbeddingModel
	} else if existing.Providers.ProviderConfigs.OpenAI != nil {
		config.Embedding.Model = existing.Providers.ProviderConfigs.OpenAI.EmbeddingModel
	} else if existing.Providers.ProviderConfigs.LMStudio != nil {
		config.Embedding.Model = existing.Providers.ProviderConfigs.LMStudio.EmbeddingModel
	} else {
		// Fallback to legacy config
		config.Embedding.Model = existing.Ollama.EmbeddingModel
	}

	return config, nil
}

// convertMCPConfig converts existing MCP configuration to MCP pillar configuration
func (adapter *ConfigAdapter) convertMCPConfig(existing *existingConfig.Config) (core.MCPConfig, error) {
	config := core.MCPConfig{
		ServersPath: existing.MCP.ServersConfigPath,
		Servers:     make(map[string]core.ServerConfig),
		HealthCheck: core.HealthCheckConfig{
			Enabled:  existing.MCP.Enabled,
			Interval: existing.MCP.HealthCheckInterval,
			Timeout:  existing.MCP.DefaultTimeout,
			Retries:  3,
		},
		ToolExecution: core.ToolExecutionConfig{
			MaxConcurrent:  existing.MCP.MaxConcurrentRequests,
			DefaultTimeout: existing.MCP.DefaultTimeout,
			EnableCache:    true,
			CacheTTL:       5 * time.Minute,
		},
	}

	// Convert server configurations
	for _, server := range existing.MCP.Servers {
		config.Servers[server.Name] = core.ServerConfig{
			Name:       server.Name,
			Command:    server.Command,
			Args:       server.Args,
			Environment: server.Env,
			WorkingDir: server.WorkingDir,
			Timeout:    existing.MCP.DefaultTimeout,
			Retries:    server.MaxRestarts,
		}
	}

	return config, nil
}

// convertAgentConfig converts existing agent configuration to Agent pillar configuration
func (adapter *ConfigAdapter) convertAgentConfig(existing *existingConfig.Config) (core.AgentsConfig, error) {
	config := core.AgentsConfig{
		WorkflowEngine: core.WorkflowEngineConfig{
			MaxSteps:       100,
			StepTimeout:    5 * time.Minute,
			StateBackend:   "memory",
			EnableRecovery: true,
		},
		Scheduling: core.SchedulingConfig{
			Backend:       "memory",
			MaxConcurrent: 10,
			QueueSize:     1000,
		},
		StateStorage: core.StateStorageConfig{
			Backend:    "memory",
			Persistent: false,
			TTL:        24 * time.Hour,
		},
		ReasoningChains: core.ReasoningChainsConfig{
			MaxSteps:      50,
			MaxMemorySize: 10000,
			StepTimeout:   30 * time.Second,
		},
	}

	return config, nil
}

// LoadCoreConfigFromPath is a convenience function to load configuration from a path
func LoadCoreConfigFromPath(configPath string) (*core.Config, error) {
	adapter := NewConfigAdapter(configPath)
	return adapter.LoadCoreConfig()
}

// LoadDefaultCoreConfig loads configuration from default locations
func LoadDefaultCoreConfig() (*core.Config, error) {
	adapter := NewConfigAdapter("")
	return adapter.LoadCoreConfig()
}