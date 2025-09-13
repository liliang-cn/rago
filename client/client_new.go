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

// RagoClient represents the main thin client wrapper
type RagoClient struct {
	config        *config.Config
	ragClient     *rag.Client
	mcpService    *mcp.Service
	statusChecker *providers.StatusChecker
	llm           domain.Generator
	embedder      domain.Embedder
}

// NewRagoClient creates a new thin client wrapper
func NewRagoClient(configPath string) (*RagoClient, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return NewRagoClientWithConfig(cfg)
}

// NewRagoClientWithConfig creates a new client with provided configuration
func NewRagoClientWithConfig(cfg *config.Config) (*RagoClient, error) {
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
	mcpService, err := mcp.NewService(&cfg.MCP, llm)
	if err != nil {
		// MCP is optional, so we can continue without it
		fmt.Printf("Warning: MCP service not available: %v\n", err)
	}

	// Create status checker
	statusChecker := providers.NewStatusChecker(cfg, embedder, llm)

	return &RagoClient{
		config:        cfg,
		ragClient:     ragClient,
		mcpService:    mcpService,
		statusChecker: statusChecker,
		llm:           llm,
		embedder:      embedder,
	}, nil
}

// ========================================
// RAG Operations (delegated to RAG client)
// ========================================

// IngestFile ingests a file into the RAG system
func (c *RagoClient) IngestFile(ctx context.Context, filePath string) error {
	_, err := c.ragClient.IngestFile(ctx, filePath, nil)
	return err
}

// IngestFileWithOptions ingests a file with custom options
func (c *RagoClient) IngestFileWithOptions(ctx context.Context, filePath string, opts *rag.IngestOptions) error {
	_, err := c.ragClient.IngestFile(ctx, filePath, opts)
	return err
}

// IngestText ingests text content directly
func (c *RagoClient) IngestText(ctx context.Context, text, source string) error {
	_, err := c.ragClient.IngestText(ctx, text, source, nil)
	return err
}

// IngestURL ingests content from a URL
func (c *RagoClient) IngestURL(ctx context.Context, url string) error {
	_, err := c.ragClient.IngestURL(ctx, url, nil)
	return err
}

// Query performs a RAG query
func (c *RagoClient) Query(ctx context.Context, query string) (*domain.QueryResponse, error) {
	return c.ragClient.Query(ctx, query, nil)
}

// QueryWithOptions performs a RAG query with custom options
func (c *RagoClient) QueryWithOptions(ctx context.Context, query string, opts *rag.QueryOptions) (*domain.QueryResponse, error) {
	return c.ragClient.Query(ctx, query, opts)
}

// ListDocuments lists all documents in the RAG store
func (c *RagoClient) ListDocuments(ctx context.Context) ([]domain.Document, error) {
	return c.ragClient.ListDocuments(ctx)
}

// DeleteDocument deletes a document by ID
func (c *RagoClient) DeleteDocument(ctx context.Context, documentID string) error {
	return c.ragClient.DeleteDocument(ctx, documentID)
}

// ResetRAG clears all documents from the RAG store
func (c *RagoClient) ResetRAG(ctx context.Context) error {
	return c.ragClient.Reset(ctx)
}

// ========================================
// MCP Operations (delegated to MCP service)
// ========================================

// StartMCPServers starts MCP servers
func (c *RagoClient) StartMCPServers(ctx context.Context, serverNames ...string) error {
	if c.mcpService == nil {
		return fmt.Errorf("MCP service not available")
	}
	return c.mcpService.StartServers(ctx, serverNames)
}

// StopMCPServer stops a specific MCP server
func (c *RagoClient) StopMCPServer(serverName string) error {
	if c.mcpService == nil {
		return fmt.Errorf("MCP service not available")
	}
	return c.mcpService.StopServer(serverName)
}

// ListMCPServers returns a list of MCP servers and their status
func (c *RagoClient) ListMCPServers() []mcp.ServerStatus {
	if c.mcpService == nil {
		return []mcp.ServerStatus{}
	}
	return c.mcpService.ListServers()
}

// CallMCPTool calls an MCP tool by name
func (c *RagoClient) CallMCPTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*mcp.ToolResult, error) {
	if c.mcpService == nil {
		return nil, fmt.Errorf("MCP service not available")
	}
	return c.mcpService.CallTool(ctx, toolName, arguments)
}

// MCPChat performs an MCP-enabled chat interaction
func (c *RagoClient) MCPChat(ctx context.Context, message string) (*mcp.ChatResponse, error) {
	if c.mcpService == nil {
		return nil, fmt.Errorf("MCP service not available")
	}
	return c.mcpService.Chat(ctx, message, nil)
}

// ========================================
// Status Operations (delegated to status checker)
// ========================================

// CheckStatus checks the status of all providers
func (c *RagoClient) CheckStatus(ctx context.Context) (*providers.Status, error) {
	return c.statusChecker.CheckAll(ctx)
}

// CheckLLMHealth checks the health of the LLM provider
func (c *RagoClient) CheckLLMHealth(ctx context.Context) (bool, error) {
	return c.statusChecker.CheckLLM(ctx)
}

// CheckEmbedderHealth checks the health of the embedder provider
func (c *RagoClient) CheckEmbedderHealth(ctx context.Context) (bool, error) {
	return c.statusChecker.CheckEmbedder(ctx)
}

// ========================================
// Direct LLM Operations
// ========================================

// Generate generates text using the LLM
func (c *RagoClient) Generate(ctx context.Context, prompt string) (string, error) {
	return c.llm.Generate(ctx, prompt, &domain.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   2000,
	})
}

// GenerateWithOptions generates text with custom options
func (c *RagoClient) GenerateWithOptions(ctx context.Context, prompt string, opts *domain.GenerationOptions) (string, error) {
	return c.llm.Generate(ctx, prompt, opts)
}

// ========================================
// Utility Methods
// ========================================

// GetConfig returns the client configuration
func (c *RagoClient) GetConfig() *config.Config {
	return c.config
}

// GetRAGClient returns the underlying RAG client for advanced operations
func (c *RagoClient) GetRAGClient() *rag.Client {
	return c.ragClient
}

// GetMCPService returns the underlying MCP service for advanced operations
func (c *RagoClient) GetMCPService() *mcp.Service {
	return c.mcpService
}

// Close closes all services and releases resources
func (c *RagoClient) Close() error {
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

	if len(errs) > 0 {
		return fmt.Errorf("errors closing client: %v", errs)
	}

	return nil
}