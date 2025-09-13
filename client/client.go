package client

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/providers"
	"github.com/liliang-cn/rago/v2/pkg/rag"
)

// Client represents the main rago client with advanced features
type Client struct {
	config        *config.Config
	ragClient     *rag.Client         // RAG operations
	mcpService    *mcp.Service        // MCP operations
	statusChecker *providers.StatusChecker
	llm           domain.Generator
	embedder      domain.Embedder
	
	// Advanced features
	mcpClient   *MCPClient  // Advanced MCP functionality
	taskClient  *TaskClient // Task scheduling functionality
}

// New creates a new rago client with the specified config file path
func New(configPath string) (*Client, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return NewWithConfig(cfg)
}

// NewWithConfig creates a new rago client with the provided configuration
func NewWithConfig(cfg *config.Config) (*Client, error) {
	ctx := context.Background()

	// Initialize providers using the centralized initialization
	embedder, llm, metadataExtractor, err := providers.InitializeProviders(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	// Create RAG client
	ragClient, err := rag.NewClient(cfg, embedder, llm, metadataExtractor)
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

	return &Client{
		config:        cfg,
		ragClient:     ragClient,
		mcpService:    mcpService,
		statusChecker: statusChecker,
		llm:           llm,
		embedder:      embedder,
	}, nil
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	var errs []error

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
func (c *Client) GetConfig() *config.Config {
	return c.config
}
