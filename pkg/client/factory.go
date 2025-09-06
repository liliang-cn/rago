package client

import (
	"time"
	
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	"github.com/liliang-cn/rago/v2/pkg/rag"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/agents"
)

// ===== INDIVIDUAL PILLAR CLIENTS =====

// LLMClient implements the individual LLM pillar client.
type LLMClient struct {
	*llm.Service
}

// NewLLMClient creates a new standalone LLM client.
func NewLLMClient(config core.LLMConfig) (core.LLMClient, error) {
	service, err := llm.NewService(config)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "factory", "NewLLMClient", "failed to create LLM service")
	}
	
	return &LLMClient{Service: service}, nil
}

// Close closes the LLM client and cleans up resources.
func (c *LLMClient) Close() error {
	return c.Service.Close()
}

// RAGClient implements the individual RAG pillar client.
type RAGClient struct {
	*rag.Service
}

// NewRAGClient creates a new standalone RAG client.
func NewRAGClient(config core.RAGConfig) (core.RAGClient, error) {
	// Use backward compatibility adapter with default embedder
	embedder := &rag.DefaultEmbedder{}
	service, err := rag.NewServiceFromCoreConfig(config, embedder)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "factory", "NewRAGClient", "failed to create RAG service")
	}
	
	return &RAGClient{Service: service}, nil
}

// Close closes the RAG client and cleans up resources.
func (c *RAGClient) Close() error {
	return c.Service.Close()
}

// MCPClient implements the individual MCP pillar client.
type MCPClient struct {
	*mcp.Service
}

// NewMCPClient creates a new standalone MCP client.
func NewMCPClient(config core.MCPConfig) (core.MCPClient, error) {
	service, err := mcp.NewService(config)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "factory", "NewMCPClient", "failed to create MCP service")
	}
	
	return &MCPClient{Service: service}, nil
}

// Close closes the MCP client and cleans up resources.
func (c *MCPClient) Close() error {
	return c.Service.Close()
}

// AgentClient implements the individual Agent pillar client.
type AgentClient struct {
	*agents.Service
}

// NewAgentClient creates a new standalone Agent client.
func NewAgentClient(config core.AgentsConfig) (core.AgentClient, error) {
	service, err := agents.NewService(config)
	if err != nil {
		return nil, core.WrapErrorWithContext(err, "factory", "NewAgentClient", "failed to create Agent service")
	}
	
	return &AgentClient{Service: service}, nil
}

// Close closes the Agent client and cleans up resources.
func (c *AgentClient) Close() error {
	return c.Service.Close()
}

// ===== FACTORY FUNCTIONS =====

// NewFromConfig creates the appropriate client based on configuration mode.
func NewFromConfig(config core.Config) (core.Client, error) {
	// Check for single-pillar modes
	if config.Mode.LLMOnly {
		return NewLLMOnlyClient(config)
	}
	
	if config.Mode.RAGOnly {
		return NewRAGOnlyClient(config)
	}
	
	// Create full unified client
	return New(config)
}

// NewLLMOnlyClient creates a client with only LLM pillar enabled.
func NewLLMOnlyClient(config core.Config) (core.Client, error) {
	// Disable other pillars
	config.Mode.DisableMCP = true
	config.Mode.DisableAgent = true
	
	return New(config)
}

// NewRAGOnlyClient creates a client with only RAG pillar enabled.
func NewRAGOnlyClient(config core.Config) (core.Client, error) {
	// Disable other pillars
	config.Mode.DisableMCP = true
	config.Mode.DisableAgent = true
	
	return New(config)
}

// NewMinimalClient creates a client with minimal pillars (LLM + RAG only).
func NewMinimalClient(config core.Config) (core.Client, error) {
	// Disable optional pillars
	config.Mode.DisableMCP = true
	config.Mode.DisableAgent = true
	
	return New(config)
}

// NewFullClient creates a client with all four pillars enabled.
func NewFullClient(config core.Config) (core.Client, error) {
	// Ensure all pillars are enabled
	config.Mode.LLMOnly = false
	config.Mode.RAGOnly = false
	config.Mode.DisableMCP = false
	config.Mode.DisableAgent = false
	
	return New(config)
}

// NewChatClient creates a client optimized for chat operations (LLM + RAG).
func NewChatClient(config core.Config) (core.Client, error) {
	// Enable essential pillars for chat
	config.Mode.LLMOnly = false
	config.Mode.RAGOnly = false
	config.Mode.DisableMCP = true  // MCP optional for chat
	config.Mode.DisableAgent = true // Agents optional for chat
	
	return New(config)
}

// NewAutomationClient creates a client optimized for automation (Agents + MCP + LLM).
func NewAutomationClient(config core.Config) (core.Client, error) {
	// Enable pillars needed for automation
	config.Mode.LLMOnly = false
	config.Mode.DisableAgent = false
	config.Mode.DisableMCP = false
	config.Mode.RAGOnly = false  // RAG can be disabled for pure automation
	
	return New(config)
}

// NewKnowledgeClient creates a client optimized for knowledge management (RAG + LLM).
func NewKnowledgeClient(config core.Config) (core.Client, error) {
	// Enable pillars needed for knowledge management
	config.Mode.RAGOnly = false
	config.Mode.LLMOnly = false
	config.Mode.DisableMCP = true
	config.Mode.DisableAgent = true
	
	return New(config)
}

// ===== BUILDER PATTERN =====

// ClientBuilder provides a fluent interface for building clients.
type ClientBuilder struct {
	config core.Config
	err    error
}

// NewBuilder creates a new client builder with default configuration.
func NewBuilder() *ClientBuilder {
	return &ClientBuilder{
		config: core.Config{
			DataDir:  "~/.rago",
			LogLevel: "info",
			Mode:     core.ModeConfig{},
		},
	}
}

// WithConfig sets the base configuration.
func (b *ClientBuilder) WithConfig(config core.Config) *ClientBuilder {
	b.config = config
	return b
}

// WithConfigPath loads configuration from a file.
func (b *ClientBuilder) WithConfigPath(path string) *ClientBuilder {
	config, err := LoadCoreConfigFromPath(path)
	if err != nil {
		b.err = err
		return b
	}
	b.config = *config
	return b
}

// WithLLM enables and configures the LLM pillar.
func (b *ClientBuilder) WithLLM(config core.LLMConfig) *ClientBuilder {
	b.config.LLM = config
	b.config.Mode.LLMOnly = false
	return b
}

// WithRAG enables and configures the RAG pillar.
func (b *ClientBuilder) WithRAG(config core.RAGConfig) *ClientBuilder {
	b.config.RAG = config
	b.config.Mode.RAGOnly = false
	return b
}

// WithMCP enables and configures the MCP pillar.
func (b *ClientBuilder) WithMCP(config core.MCPConfig) *ClientBuilder {
	b.config.MCP = config
	b.config.Mode.DisableMCP = false
	return b
}

// WithAgents enables and configures the Agent pillar.
func (b *ClientBuilder) WithAgents(config core.AgentsConfig) *ClientBuilder {
	b.config.Agents = config
	b.config.Mode.DisableAgent = false
	return b
}

// WithoutMCP disables the MCP pillar.
func (b *ClientBuilder) WithoutMCP() *ClientBuilder {
	b.config.Mode.DisableMCP = true
	return b
}

// WithoutAgents disables the Agent pillar.
func (b *ClientBuilder) WithoutAgents() *ClientBuilder {
	b.config.Mode.DisableAgent = true
	return b
}

// WithDataDir sets the data directory.
func (b *ClientBuilder) WithDataDir(dir string) *ClientBuilder {
	b.config.DataDir = dir
	return b
}

// WithLogLevel sets the log level.
func (b *ClientBuilder) WithLogLevel(level string) *ClientBuilder {
	b.config.LogLevel = level
	return b
}

// Build creates the client with the configured settings.
func (b *ClientBuilder) Build() (core.Client, error) {
	if b.err != nil {
		return nil, b.err
	}
	return New(b.config)
}

// ===== CONVENIENCE CONSTRUCTORS FOR COMMON PATTERNS =====

// NewWithDefaults creates a client with sensible defaults for all pillars.
func NewWithDefaults() (core.Client, error) {
	config, err := LoadDefaultCoreConfig()
	if err != nil {
		// Create minimal default configuration
		config = &core.Config{
			DataDir:  "~/.rago",
			LogLevel: "info",
			LLM: core.LLMConfig{
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
				HealthCheck: core.HealthCheckConfig{
					Enabled:  true,
					Interval: 30 * time.Second,
					Timeout:  10 * time.Second,
					Retries:  3,
				},
			},
			RAG: core.RAGConfig{
				StorageBackend: "dual",
				ChunkingStrategy: core.ChunkingConfig{
					Strategy:     "recursive",
					ChunkSize:    500,
					ChunkOverlap: 50,
				},
			},
			MCP: core.MCPConfig{
				ServersPath: "~/.rago/mcpServers.json",
			},
			Agents: core.AgentsConfig{
				WorkflowEngine: core.WorkflowEngineConfig{
					MaxSteps:    100,
					StateBackend: "memory",
				},
			},
		}
	}
	
	return NewFullClient(*config)
}

// NewForTesting creates a client suitable for testing with in-memory backends.
func NewForTesting() (core.Client, error) {
	config := core.Config{
		DataDir:  "/tmp/rago-test",
		LogLevel: "debug",
		LLM: core.LLMConfig{
			DefaultProvider: "mock",
			Providers: map[string]core.ProviderConfig{
				"mock": {
					Type: "mock",
					Parameters: map[string]interface{}{
						"test_mode": true,
					},
				},
			},
		},
		RAG: core.RAGConfig{
			StorageBackend: "memory",
			ChunkingStrategy: core.ChunkingConfig{
				Strategy:  "fixed",
				ChunkSize: 100,
			},
		},
		MCP: core.MCPConfig{
			ToolExecution: core.ToolExecutionConfig{
				MaxConcurrent: 1,
				EnableCache:   false,
			},
		},
		Agents: core.AgentsConfig{
			WorkflowEngine: core.WorkflowEngineConfig{
				StateBackend: "memory",
			},
			StateStorage: core.StateStorageConfig{
				Backend:    "memory",
				Persistent: false,
			},
		},
	}
	
	return NewFullClient(config)
}