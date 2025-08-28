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
	return c.IngestTextWithMetadata(text, source, nil)
}

func (c *Client) IngestTextWithMetadata(text, source string, additionalMetadata map[string]interface{}) error {
	ctx := context.Background()
	
	// Start with base metadata
	metadata := map[string]interface{}{
		"source": source,
	}
	
	// Merge additional metadata if provided
	if additionalMetadata != nil {
		for key, value := range additionalMetadata {
			metadata[key] = value
		}
	}
	
	req := domain.IngestRequest{
		Content:   text,
		ChunkSize: c.config.Chunker.ChunkSize,
		Overlap:   c.config.Chunker.Overlap,
		Metadata:  metadata,
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
		ShowSources:  false, // Default to false for backward compatibility
	}

	return c.processor.Query(ctx, req)
}

// QueryWithSources performs a query and returns sources information
func (c *Client) QueryWithSources(query string, showSources bool) (domain.QueryResponse, error) {
	ctx := context.Background()
	req := domain.QueryRequest{
		Query:        query,
		TopK:         c.config.Sqvect.TopK,
		Temperature:  0.7,
		MaxTokens:    25000,
		Stream:       false,
		ShowThinking: true,
		ShowSources:  showSources,
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
		ShowSources:  false, // Default to false for backward compatibility
	}

	return c.processor.StreamQuery(ctx, req, callback)
}

// StreamQueryWithSources performs a streaming query and returns sources at the end
func (c *Client) StreamQueryWithSources(query string, callback func(string), showSources bool) ([]domain.Chunk, error) {
	if !showSources {
		// If sources not requested, use regular streaming
		return nil, c.StreamQuery(query, callback)
	}

	// First get the sources by doing a non-streaming query
	ctx := context.Background()
	req := domain.QueryRequest{
		Query:        query,
		TopK:         c.config.Sqvect.TopK,
		Temperature:  0.7,
		MaxTokens:    25000,
		Stream:       false,
		ShowThinking: true,
		ShowSources:  true,
	}

	// Get sources first
	resp, err := c.processor.Query(ctx, req)
	if err != nil {
		return nil, err
	}

	// Now do streaming with the same query
	req.Stream = true
	req.ShowSources = false // Don't need sources in streaming response
	err = c.processor.StreamQuery(ctx, req, callback)
	if err != nil {
		return resp.Sources, err
	}

	return resp.Sources, nil
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

// LLMGenerateRequest defines the request for a direct LLM generation
type LLMGenerateRequest struct {
	Prompt      string
	Temperature float64
	MaxTokens   int
}

// LLMGenerateResponse defines the response from a direct LLM generation
type LLMGenerateResponse struct {
	Content string
}

// LLMGenerate performs a direct generation using the configured LLM
func (c *Client) LLMGenerate(ctx context.Context, req LLMGenerateRequest) (LLMGenerateResponse, error) {
	if c.llm == nil {
		return LLMGenerateResponse{}, fmt.Errorf("LLM service not initialized")
	}

	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	content, err := c.llm.Generate(ctx, req.Prompt, opts)
	if err != nil {
		return LLMGenerateResponse{}, fmt.Errorf("LLM generation failed: %w", err)
	}

	return LLMGenerateResponse{Content: content}, nil
}

// LLMGenerateStream performs a direct streaming generation using the configured LLM
func (c *Client) LLMGenerateStream(ctx context.Context, req LLMGenerateRequest, callback func(string)) error {
	if c.llm == nil {
		return fmt.Errorf("LLM service not initialized")
	}

	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	return c.llm.Stream(ctx, req.Prompt, opts, callback)
}

// ChatMessage defines a single message in a chat conversation
type ChatMessage struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// LLMChatRequest defines the request for a direct LLM chat
type LLMChatRequest struct {
	Messages    []ChatMessage
	Temperature float64
	MaxTokens   int
}

// LLMChat performs a direct chat using the configured LLM
func (c *Client) LLMChat(ctx context.Context, req LLMChatRequest) (LLMGenerateResponse, error) {
	if c.llm == nil {
		return LLMGenerateResponse{}, fmt.Errorf("LLM service not initialized")
	}

	// Convert messages to internal domain format
	domainMessages := make([]domain.Message, len(req.Messages))
	for i, msg := range req.Messages {
		domainMessages[i] = domain.Message{Role: msg.Role, Content: msg.Content}
	}

	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	// Use GenerateWithTools with no tools for chat
	result, err := c.llm.GenerateWithTools(ctx, domainMessages, nil, opts)
	if err != nil {
		return LLMGenerateResponse{}, fmt.Errorf("LLM chat failed: %w", err)
	}

	return LLMGenerateResponse{Content: result.Content}, nil
}

// LLMChatStream performs a direct streaming chat using the configured LLM
func (c *Client) LLMChatStream(ctx context.Context, req LLMChatRequest, callback func(string)) error {
	if c.llm == nil {
		return fmt.Errorf("LLM service not initialized")
	}

	// Convert messages to internal domain format
	domainMessages := make([]domain.Message, len(req.Messages))
	for i, msg := range req.Messages {
		domainMessages[i] = domain.Message{Role: msg.Role, Content: msg.Content}
	}

	opts := &domain.GenerationOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	// Use StreamWithTools with no tools for chat streaming
	return c.llm.StreamWithTools(ctx, domainMessages, nil, opts, func(chunk string, toolCalls []domain.ToolCall) error {
		callback(chunk)
		return nil
	})
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
