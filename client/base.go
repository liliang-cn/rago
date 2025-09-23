package client

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/providers"
	"github.com/liliang-cn/rago/v2/pkg/rag"
	"github.com/liliang-cn/rago/v2/pkg/settings"
)

// BaseClient represents the main rago client with advanced features
type BaseClient struct {
	config         *config.Config
	configWithSettings *settings.ConfigWithSettings // Settings-aware configuration
	ragClient      *rag.Client                      // RAG operations
	mcpService     *mcp.Service                     // MCP operations
	statusChecker  *providers.StatusChecker
	llm            domain.Generator
	embedder       domain.Embedder

	// Advanced features
	mcpClient  *MCPClient  // Advanced MCP functionality
	taskClient *TaskClient // Task scheduling functionality

	// Public properties for backward compatibility
	LLM      *LLMWrapper
	RAG      *RAGWrapper
	Tools    *ToolsWrapper
	Agent    *AgentWrapper
	Settings *SettingsWrapper // New settings wrapper
}

// New creates a new rago client with the specified config file path
func New(configPath string) (*BaseClient, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return NewWithConfig(cfg)
}

// NewWithConfig creates a new rago client with the provided configuration
func NewWithConfig(cfg *config.Config) (*BaseClient, error) {
	ctx := context.Background()

	// Initialize settings-aware configuration
	configWithSettings, err := settings.NewConfigWithSettings(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize settings: %w", err)
	}

	// Initialize LLM provider with settings integration
	llm, err := configWithSettings.CreateDefaultLLMProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM provider: %w", err)
	}

	// Initialize embedder provider (no settings integration needed for embedders)
	embedder, err := configWithSettings.CreateDefaultEmbedderProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embedder provider: %w", err)
	}

	// Initialize metadata extractor (use LLM provider)
	metadataExtractor, ok := llm.(domain.LLMProvider)
	if !ok {
		return nil, fmt.Errorf("LLM provider does not support metadata extraction")
	}

	// Create RAG client
	ragClient, err := rag.NewClient(cfg, embedder, metadataExtractor, metadataExtractor)
	if err != nil {
		return nil, fmt.Errorf("failed to create RAG client: %w", err)
	}

	// Create MCP service
	var mcpService *mcp.Service
	if cfg.MCP.Enabled {
		mcpService, err = mcp.NewService(&cfg.MCP, llm)
		if err != nil {
			// MCP is optional, so we can continue without it
			fmt.Printf("Warning: MCP service not available: %v\n", err)
		}
	}

	// Create status checker
	statusChecker := providers.NewStatusChecker(cfg, embedder, llm)

	client := &BaseClient{
		config:             cfg,
		configWithSettings: configWithSettings,
		ragClient:          ragClient,
		mcpService:         mcpService,
		statusChecker:      statusChecker,
		llm:                llm,
		embedder:           embedder,
		LLM:                NewLLMWrapper(llm), // Public property for backward compatibility
		RAG:                NewRAGWrapper(ragClient),
		Tools:              nil, // Will be set if MCP is available
		Settings:           NewSettingsWrapper(configWithSettings.SettingsService),
	}

	// Set Tools wrapper if MCP service is available
	if mcpService != nil {
		client.Tools = NewToolsWrapper(mcpService)
	}

	// Set Agent wrapper
	client.Agent = NewAgentWrapper(client)

	return client, nil
}

// Close closes the client and releases resources
func (c *BaseClient) Close() error {
	var errs []error

	// Close settings service
	if c.configWithSettings != nil {
		if err := c.configWithSettings.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close settings service: %w", err))
		}
	}

	// Close RAG client
	if c.ragClient != nil {
		if err := c.ragClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close RAG client: %w", err))
		}
	}

	// Close MCP service
	if c.mcpService != nil {
		if err := c.mcpService.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close MCP service: %w", err))
		}
	}

	// Close advanced MCP client if initialized
	if c.mcpClient != nil && c.mcpClient.enabled {
		if err := c.DisableMCP(); err != nil {
			errs = append(errs, fmt.Errorf("failed to disable MCP client: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing client: %v", errs)
	}

	return nil
}

// GetConfig returns the client configuration
func (c *BaseClient) GetConfig() *config.Config {
	return c.config
}

// Ingest ingests documents into the knowledge base
func (c *BaseClient) Ingest(ctx context.Context, req IngestRequest) (*IngestResponse, error) {
	if c.ragClient == nil {
		return nil, fmt.Errorf("RAG client not initialized")
	}

	// Delegate to RAG client
	opts := &rag.IngestOptions{
		ChunkSize: req.ChunkSize,
		Overlap:   req.Overlap,
		Metadata:  make(map[string]interface{}),
	}

	// Convert metadata if provided
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			opts.Metadata[k] = v
		}
	}

	_, err := c.ragClient.IngestFile(ctx, req.Path, opts)
	if err != nil {
		return nil, err
	}

	return &IngestResponse{
		Success: true,
		Message: "Documents ingested successfully",
	}, nil
}

// Query performs a RAG query
func (c *BaseClient) Query(ctx context.Context, req QueryRequest) (*QueryResponse, error) {
	if c.ragClient == nil {
		return nil, fmt.Errorf("RAG client not initialized")
	}

	// Prepare options
	opts := &rag.QueryOptions{
		TopK:        req.TopK,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		ShowSources: req.ShowSources,
	}

	// Perform query
	result, err := c.ragClient.Query(ctx, req.Query, opts)
	if err != nil {
		return nil, err
	}

	// Convert to response
	resp := &QueryResponse{
		Answer: result.Answer,
	}

	if req.ShowSources && result.Sources != nil {
		for _, src := range result.Sources {
			resp.Sources = append(resp.Sources, SearchResult{
				Score:    src.Score,
				Content:  src.Content,
				Metadata: src.Metadata,
			})
		}
	}

	return resp, nil
}

// GetMCPClient returns the MCP client if available
func (c *BaseClient) GetMCPClient() *MCPClient {
	return c.mcpClient
}

// EnableMCP enables MCP functionality
func (c *BaseClient) EnableMCP(ctx context.Context) error {
	if c.mcpClient != nil && c.mcpClient.enabled {
		return nil // Already enabled
	}

	// Initialize MCP client
	if c.mcpService != nil {
		c.mcpClient = &MCPClient{
			service: c.mcpService,
			enabled: true,
		}
		return nil
	}

	return fmt.Errorf("MCP service not available")
}

// RunTask executes a task using the agent
func (c *BaseClient) RunTask(ctx context.Context, req TaskRequest) (*TaskResponse, error) {
	// For now, return a simple implementation
	// This would be replaced with actual agent execution
	return &TaskResponse{
		Success: true,
		Output:  map[string]interface{}{"result": "Task completed"},
		Steps: []StepResult{
			{Name: "Execute", Success: true, Output: "Done"},
		},
	}, nil
}
