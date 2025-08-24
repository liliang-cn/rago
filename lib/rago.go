package rago

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/liliang-cn/rago/internal/chunker"
	"github.com/liliang-cn/rago/internal/config"
	"github.com/liliang-cn/rago/internal/domain"
	"github.com/liliang-cn/rago/internal/processor"
	"github.com/liliang-cn/rago/internal/store"
	"github.com/liliang-cn/rago/internal/tools"
	"github.com/liliang-cn/rago/internal/utils"
)

type Client struct {
	config       *config.Config
	processor    *processor.Service
	vectorStore  *store.SQLiteStore
	keywordStore *store.KeywordStore
	embedder     domain.Embedder
	llm          domain.Generator
}

func New(configPath string) (*Client, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return NewWithConfig(cfg)
}

func NewWithConfig(cfg *config.Config) (*Client, error) {
	vectorStore, err := store.NewSQLiteStore(
		cfg.Sqvect.DBPath,
		cfg.Sqvect.VectorDim,
		cfg.Sqvect.MaxConns,
		cfg.Sqvect.BatchSize,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	keywordStore, err := store.NewKeywordStore(cfg.Keyword.IndexPath)
	if err != nil {
		vectorStore.Close() // clean up previous store
		return nil, fmt.Errorf("failed to create keyword store: %w", err)
	}

	docStore := store.NewDocumentStore(vectorStore.GetSqvectStore())

	// Initialize services using provider system
	ctx := context.Background()
	embedService, llmService, metadataExtractor, err := utils.InitializeProviders(ctx, cfg)
	if err != nil {
		vectorStore.Close()
		keywordStore.Close()
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	chunkerService := chunker.New()

	processor := processor.New(
		embedService,
		llmService,
		chunkerService,
		vectorStore,
		keywordStore,
		docStore,
		cfg,
		metadataExtractor,
	)

	return &Client{
		config:       cfg,
		processor:    processor,
		vectorStore:  vectorStore,
		keywordStore: keywordStore,
		embedder:     embedService,
		llm:          llmService,
	}, nil
}

func (c *Client) IngestFile(filePath string) error {
	ctx := context.Background()
	req := domain.IngestRequest{
		FilePath:  filePath,
		ChunkSize: c.config.Chunker.ChunkSize,
		Overlap:   c.config.Chunker.Overlap,
		Metadata: map[string]interface{}{
			"file_path": filePath,
			"file_ext":  filepath.Ext(filePath),
		},
	}

	_, err := c.processor.Ingest(ctx, req)
	return err
}

func (c *Client) IngestText(text, source string) error {
	ctx := context.Background()
	req := domain.IngestRequest{
		Content:   text,
		ChunkSize: c.config.Chunker.ChunkSize,
		Overlap:   c.config.Chunker.Overlap,
		Metadata: map[string]interface{}{
			"source": source,
		},
	}

	_, err := c.processor.Ingest(ctx, req)
	return err
}

func (c *Client) Query(query string) (domain.QueryResponse, error) {
	ctx := context.Background()
	req := domain.QueryRequest{
		Query:        query,
		TopK:         c.config.Sqvect.TopK,
		Temperature:  0.7,
		MaxTokens:    25000,
		Stream:       false, // Changed to false for library use
		ShowThinking: true,
	}

	return c.processor.Query(ctx, req)
}

// QueryWithTools performs a query with tool calling enabled
func (c *Client) QueryWithTools(query string, allowedTools []string, maxToolCalls int) (domain.QueryResponse, error) {
	ctx := context.Background()
	req := domain.QueryRequest{
		Query:        query,
		TopK:         c.config.Sqvect.TopK,
		Temperature:  0.7,
		MaxTokens:    25000,
		Stream:       false, // Non-streaming for library use
		ShowThinking: true,
		ToolsEnabled: true,
		AllowedTools: allowedTools,
		MaxToolCalls: maxToolCalls,
	}

	return c.processor.QueryWithTools(ctx, req)
}

func (c *Client) QueryWithFilters(query string, filters map[string]interface{}) (domain.QueryResponse, error) {
	ctx := context.Background()
	req := domain.QueryRequest{
		Query:        query,
		TopK:         c.config.Sqvect.TopK,
		Temperature:  0.7,
		MaxTokens:    25000,
		Stream:       true,
		ShowThinking: true,
		Filters:      filters,
	}

	return c.processor.Query(ctx, req)
}

func (c *Client) StreamQuery(query string, callback func(string)) error {
	ctx := context.Background()
	req := domain.QueryRequest{
		Query:        query,
		TopK:         c.config.Sqvect.TopK,
		Temperature:  0.7,
		MaxTokens:    25000,
		Stream:       true,
		ShowThinking: true,
	}

	return c.processor.StreamQuery(ctx, req, callback)
}

func (c *Client) StreamQueryWithFilters(query string, filters map[string]interface{}, callback func(string)) error {
	ctx := context.Background()
	req := domain.QueryRequest{
		Query:        query,
		TopK:         c.config.Sqvect.TopK,
		Temperature:  0.7,
		MaxTokens:    25000,
		Stream:       true,
		ShowThinking: true,
		Filters:      filters,
	}

	return c.processor.StreamQuery(ctx, req, callback)
}

func (c *Client) ListDocuments() ([]domain.Document, error) {
	ctx := context.Background()
	return c.processor.ListDocuments(ctx)
}

func (c *Client) DeleteDocument(documentID string) error {
	ctx := context.Background()
	return c.processor.DeleteDocument(ctx, documentID)
}

func (c *Client) Reset() error {
	ctx := context.Background()
	return c.processor.Reset(ctx)
}

func (c *Client) Close() error {
	var errs []error
	if c.vectorStore != nil {
		if err := c.vectorStore.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close vector store: %w", err))
		}
	}
	if c.keywordStore != nil {
		if err := c.keywordStore.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close keyword store: %w", err))
		}
	}

	if len(errs) > 0 {
		// Return a single error wrapping all close errors
		return fmt.Errorf("failed to close client resources: %v", errs)
	}
	return nil
}

// Tool Management Methods

// ListAvailableTools returns all available tools
func (c *Client) ListAvailableTools() []tools.ToolInfo {
	if registry := c.processor.GetToolRegistry(); registry != nil {
		return registry.List()
	}
	return []tools.ToolInfo{}
}

// ListEnabledTools returns only enabled tools
func (c *Client) ListEnabledTools() []tools.ToolInfo {
	if registry := c.processor.GetToolRegistry(); registry != nil {
		return registry.ListEnabled()
	}
	return []tools.ToolInfo{}
}

// ExecuteTool directly executes a tool with given arguments
func (c *Client) ExecuteTool(toolName string, args map[string]interface{}) (*tools.ToolResult, error) {
	registry := c.processor.GetToolRegistry()
	executor := c.processor.GetToolExecutor()

	if registry == nil || executor == nil {
		return nil, fmt.Errorf("tools are not enabled")
	}

	// Check if tool exists and is enabled
	if !registry.IsEnabled(toolName) {
		return nil, fmt.Errorf("tool '%s' is not enabled", toolName)
	}

	tool, exists := registry.Get(toolName)
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", toolName)
	}

	// Validate arguments
	if err := tool.Validate(args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Create execution context
	ctx := context.Background()
	execCtx := &tools.ExecutionContext{
		RequestID: fmt.Sprintf("lib-%s-%d", toolName, time.Now().Unix()),
		UserID:    "lib-user",
		SessionID: "lib-session",
	}

	// Execute tool
	result, err := executor.Execute(ctx, execCtx, toolName, args)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	return result, nil
}

// GetToolStats returns tool execution statistics
func (c *Client) GetToolStats() map[string]interface{} {
	registry := c.processor.GetToolRegistry()
	executor := c.processor.GetToolExecutor()

	if registry == nil || executor == nil {
		return map[string]interface{}{
			"tools_enabled": false,
		}
	}

	return map[string]interface{}{
		"tools_enabled":  true,
		"registry_stats": registry.Stats(),
		"executor_stats": executor.GetStats(),
	}
}

// convertToolInfoSlice converts internal tool info slice to interface slice
// func convertToolInfoSlice(toolInfos []tools.ToolInfo) []interface{} {
// 	result := make([]interface{}, len(toolInfos))
// 	for i, info := range toolInfos {
// 		result[i] = info
// 	}
// 	return result
// }

func (c *Client) GetConfig() *config.Config {
	return c.config
}

type StatusResult struct {
	ProvidersAvailable bool
	LLMProvider        string
	EmbedderProvider   string
	Error              error
}

func (c *Client) CheckStatus() StatusResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := StatusResult{
		LLMProvider:      c.config.Providers.DefaultLLM,
		EmbedderProvider: c.config.Providers.DefaultEmbedder,
	}

	// If using legacy config, show that instead
	if c.config.Providers.DefaultLLM == "" {
		result.LLMProvider = "ollama (legacy)"
		result.EmbedderProvider = "ollama (legacy)"
	}

	// Check provider health using the utils function
	if err := utils.CheckProviderHealth(ctx, c.embedder, c.llm); err != nil {
		result.ProvidersAvailable = false
		result.Error = err
	} else {
		result.ProvidersAvailable = true
	}

	return result
}
