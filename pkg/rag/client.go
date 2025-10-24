package rag

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/rag/chunker"
	"github.com/liliang-cn/rago/v2/pkg/rag/processor"
	"github.com/liliang-cn/rago/v2/pkg/rag/store"
	"github.com/liliang-cn/rago/v2/pkg/settings"
)

// Client provides high-level RAG operations
type Client struct {
	processor   *processor.Service
	vectorStore domain.VectorStore
	docStore    *store.DocumentStore
	embedder    domain.Embedder
	llm         domain.Generator
	config      *config.Config
	settings    *settings.Service
	mcpService  *mcp.Service
}

// NewClient creates a new RAG client
func NewClient(cfg *config.Config, embedder domain.Embedder, llm domain.Generator, metadataExtractor domain.MetadataExtractor) (*Client, error) {
	// Initialize vector store based on configuration
	var vectorStore domain.VectorStore
	var docStore *store.DocumentStore
	var err error

	if cfg.VectorStore != nil && cfg.VectorStore.Type != "" {
		// Use configured vector store
		storeConfig := store.StoreConfig{
			Type:       cfg.VectorStore.Type,
			Parameters: cfg.VectorStore.Parameters,
		}
		vectorStore, err = store.NewVectorStore(storeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create vector store: %w", err)
		}

		// For document store, use SQLite alongside vector stores that don't provide document storage
		if cfg.VectorStore.Type == "qdrant" {
			// Qdrant doesn't store full documents, so use SQLite for document storage
			sqliteStore, err := store.NewSQLiteStore(cfg.Sqvect.DBPath)
			if err != nil {
				return nil, fmt.Errorf("failed to create document store: %w", err)
			}
			docStore = store.NewDocumentStore(sqliteStore.GetSqvectStore())
		}
	} else {
		// Default to SQLite for backward compatibility
		sqliteStore, err := store.NewSQLiteStore(cfg.Sqvect.DBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create vector store: %w", err)
		}
		vectorStore = sqliteStore
		docStore = store.NewDocumentStore(sqliteStore.GetSqvectStore())
	}

	// If docStore is still nil (for SQLite vector store), create it
	if docStore == nil {
		if sqliteStore, ok := vectorStore.(*store.SQLiteStore); ok {
			docStore = store.NewDocumentStore(sqliteStore.GetSqvectStore())
		}
	}

	// Initialize chunker
	chunkerService := chunker.New()

	// Initialize processor
	proc := processor.New(
		embedder,
		llm,
		chunkerService,
		vectorStore,
		docStore,
		cfg,
		metadataExtractor,
	)

	// Initialize settings service
	settingsService, err := settings.NewService(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create settings service: %w", err)
	}

	// Initialize MCP service
	// Note: MCP service requires an LLM generator, but we'll initialize it later when needed
	var mcpService *mcp.Service
	// mcpService will be initialized lazily when needed

	return &Client{
		processor:   proc,
		vectorStore: vectorStore,
		docStore:    docStore,
		embedder:    embedder,
		llm:         llm,
		config:      cfg,
		settings:    settingsService,
		mcpService:  mcpService,
	}, nil
}

// IngestOptions configures how content is ingested
type IngestOptions struct {
	ChunkSize          int                    // Size of text chunks
	Overlap            int                    // Overlap between chunks
	EnhancedExtraction bool                   // Enable enhanced metadata extraction
	Metadata           map[string]interface{} // Additional metadata
}

// DefaultIngestOptions returns default ingest options
func DefaultIngestOptions() *IngestOptions {
	return &IngestOptions{
		ChunkSize: 1000,
		Overlap:   200,
	}
}

// IngestFile ingests a file from the local filesystem
func (c *Client) IngestFile(ctx context.Context, filePath string, opts *IngestOptions) (*domain.IngestResponse, error) {
	if opts == nil {
		opts = DefaultIngestOptions()
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	req := domain.IngestRequest{
		FilePath:  absPath,
		ChunkSize: opts.ChunkSize,
		Overlap:   opts.Overlap,
		Metadata:  opts.Metadata,
	}

	// Handle enhanced extraction
	if opts.EnhancedExtraction {
		origConfig := c.config.Ingest.MetadataExtraction.Enable
		c.config.Ingest.MetadataExtraction.Enable = true
		defer func() {
			c.config.Ingest.MetadataExtraction.Enable = origConfig
		}()
	}

	resp, err := c.processor.Ingest(ctx, req)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// IngestText ingests text content directly
func (c *Client) IngestText(ctx context.Context, text, source string, opts *IngestOptions) (*domain.IngestResponse, error) {
	if opts == nil {
		opts = DefaultIngestOptions()
	}

	metadata := make(map[string]interface{})
	if opts.Metadata != nil {
		metadata = opts.Metadata
	}
	metadata["source"] = source
	metadata["type"] = "text"
	metadata["ingested_at"] = time.Now().Format(time.RFC3339)

	req := domain.IngestRequest{
		Content:   text,
		ChunkSize: opts.ChunkSize,
		Overlap:   opts.Overlap,
		Metadata:  metadata,
	}

	resp, err := c.processor.Ingest(ctx, req)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// IngestURL ingests content from a URL
func (c *Client) IngestURL(ctx context.Context, url string, opts *IngestOptions) (*domain.IngestResponse, error) {
	if opts == nil {
		opts = DefaultIngestOptions()
	}

	req := domain.IngestRequest{
		URL:       url,
		ChunkSize: opts.ChunkSize,
		Overlap:   opts.Overlap,
		Metadata:  opts.Metadata,
	}

	resp, err := c.processor.Ingest(ctx, req)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// QueryOptions configures how queries are executed
type QueryOptions struct {
	TopK         int                    // Number of documents to retrieve
	Temperature  float64                // LLM temperature
	MaxTokens    int                    // Maximum tokens in response
	ShowSources  bool                   // Include source documents in response
	ShowThinking bool                   // Show reasoning process
	Stream       bool                   // Enable streaming response
	ToolsEnabled bool                   // Enable tool calling
	AllowedTools []string               // Specific tools to allow
	MaxToolCalls int                    // Maximum tool calls
	Filters      map[string]interface{} // Metadata filters for document retrieval
}

// DefaultQueryOptions returns default query options
func DefaultQueryOptions() *QueryOptions {
	return &QueryOptions{
		TopK:        5,
		Temperature: 0.7,
		MaxTokens:   2000,
		ShowSources: true,
	}
}

// Query performs a RAG query
func (c *Client) Query(ctx context.Context, query string, opts *QueryOptions) (*domain.QueryResponse, error) {
	if opts == nil {
		opts = DefaultQueryOptions()
	}

	req := domain.QueryRequest{
		Query:        query,
		TopK:         opts.TopK,
		Temperature:  opts.Temperature,
		MaxTokens:    opts.MaxTokens,
		ShowSources:  opts.ShowSources,
		ShowThinking: opts.ShowThinking,
		Stream:       opts.Stream,
		ToolsEnabled: opts.ToolsEnabled,
		AllowedTools: opts.AllowedTools,
		MaxToolCalls: opts.MaxToolCalls,
		Filters:      opts.Filters,
	}

	resp, err := c.processor.Query(ctx, req)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// ListDocuments lists all documents in the store
func (c *Client) ListDocuments(ctx context.Context) ([]domain.Document, error) {
	if c.docStore != nil {
		return c.docStore.List(ctx)
	}
	// Fallback to vector store if it supports listing (like SQLite)
	return c.vectorStore.List(ctx)
}

// DeleteDocument deletes a document by ID
func (c *Client) DeleteDocument(ctx context.Context, documentID string) error {
	return c.vectorStore.Delete(ctx, documentID)
}

// Reset clears all documents from the store
func (c *Client) Reset(ctx context.Context) error {
	return c.vectorStore.Reset(ctx)
}

// GetStats returns statistics about the RAG store
func (c *Client) GetStats(ctx context.Context) (*domain.Stats, error) {
	var docs []domain.Document
	var err error
	
	if c.docStore != nil {
		docs, err = c.docStore.List(ctx)
	} else {
		docs, err = c.vectorStore.List(ctx)
	}
	
	if err != nil {
		return nil, err
	}

	// Count chunks (approximate based on documents)
	totalChunks := 0
	for range docs {
		// Estimate chunks per document
		totalChunks += 5 // This is an approximation
	}

	return &domain.Stats{
		TotalDocuments: len(docs),
		TotalChunks:    totalChunks,
	}, nil
}

// Close closes the RAG client and releases resources
func (c *Client) Close() error {
	var errs []error
	
	if c.vectorStore != nil {
		if closer, ok := c.vectorStore.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	
	if c.docStore != nil {
		// Document store typically uses same connection as SQLite vector store
		// so it's already closed above
	}
	
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// GetProcessor returns the underlying processor for advanced operations
func (c *Client) GetProcessor() *processor.Service {
	return c.processor
}

// GetVectorStore returns the underlying vector store for advanced operations
func (c *Client) GetVectorStore() domain.VectorStore {
	return c.vectorStore
}

// Profile management methods

// CreateProfile creates a new user profile
func (c *Client) CreateProfile(name, description string) (*settings.UserProfile, error) {
	req := settings.CreateProfileRequest{
		Name:        name,
		Description: description,
	}
	return c.settings.CreateProfile(req)
}

// ListProfiles returns all available profiles
func (c *Client) ListProfiles() ([]*settings.UserProfile, error) {
	return c.settings.ListProfiles()
}

// GetProfile retrieves a specific profile by ID
func (c *Client) GetProfile(profileID string) (*settings.UserProfile, error) {
	return c.settings.GetProfile(profileID)
}

// UpdateProfile updates an existing profile
func (c *Client) UpdateProfile(profileID string, updates map[string]interface{}) error {
	req := settings.UpdateProfileRequest{}
	// Convert updates to the request structure
	if name, ok := updates["name"].(string); ok {
		req.Name = &name
	}
	if desc, ok := updates["description"].(string); ok {
		req.Description = &desc
	}
	if prompt, ok := updates["default_system_prompt"].(string); ok {
		req.DefaultSystemPrompt = &prompt
	}
	if metadata, ok := updates["metadata"].(map[string]string); ok {
		req.Metadata = &metadata
	}
	_, err := c.settings.UpdateProfile(profileID, req)
	return err
}

// DeleteProfile deletes a profile
func (c *Client) DeleteProfile(profileID string) error {
	return c.settings.DeleteProfile(profileID)
}

// SetActiveProfile sets the active profile
func (c *Client) SetActiveProfile(profileID string) error {
	if err := c.settings.SetActiveProfile(profileID); err != nil {
		return err
	}
	// Update config to reflect active profile settings
	return c.updateConfigFromActiveProfile()
}

// GetActiveProfile returns the currently active profile
func (c *Client) GetActiveProfile() (*settings.UserProfile, error) {
	return c.settings.GetActiveProfile()
}

// updateConfigFromActiveProfile updates client config based on active profile settings
func (c *Client) updateConfigFromActiveProfile() error {
	profile, err := c.settings.GetActiveProfile()
	if err != nil {
		return err
	}

	// Note: Profile model settings are stored in LLMSettings, not directly in UserProfile
	// This is a placeholder for future profile-based configuration updates
	_ = profile

	return nil
}

// LLM settings management

// UpdateLLMSettings updates LLM settings in the active profile
func (c *Client) UpdateLLMSettings(settings *settings.LLMSettings) error {
	// TODO: Implement LLM settings update when settings service supports it
	return fmt.Errorf("LLM settings management not yet available")
}

// GetLLMSettings retrieves LLM settings from the active profile
func (c *Client) GetLLMSettings() (*settings.LLMSettings, error) {
	// TODO: Implement LLM settings retrieval when settings service supports it
	return nil, fmt.Errorf("LLM settings management not yet available")
}

// UpdateLLMModel updates the LLM model in the active profile
func (c *Client) UpdateLLMModel(modelName string) error {
	// TODO: Implement LLM model update when settings service supports it
	return fmt.Errorf("LLM model management not yet available")
}

// GetLLMModel retrieves the current LLM model
func (c *Client) GetLLMModel() (string, error) {
	// For now, return the model from config
	if c.config.Providers.ProviderConfigs.OpenAI != nil {
		return c.config.Providers.ProviderConfigs.OpenAI.LLMModel, nil
	}
	return "", fmt.Errorf("no LLM model configured")
}

// MCP/Tools integration methods

// ListTools lists available MCP tools
func (c *Client) ListTools(ctx context.Context) ([]interface{}, error) {
	// TODO: Implement MCP tools integration when MCP service is properly initialized
	return []interface{}{}, fmt.Errorf("MCP tools integration not yet available")
}

// CallTool executes an MCP tool
func (c *Client) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error) {
	// TODO: Implement MCP tools integration when MCP service is properly initialized
	return nil, fmt.Errorf("MCP tools integration not yet available")
}

// GetMCPStatus returns MCP service status
func (c *Client) GetMCPStatus(ctx context.Context) (interface{}, error) {
	// TODO: Implement MCP status when MCP service is properly initialized
	return map[string]interface{}{
		"enabled": false,
		"message": "MCP service not yet available",
	}, nil
}

// Enhanced RAG methods with profiles

// IngestWithProfile ingests content using a specific profile
func (c *Client) IngestWithProfile(ctx context.Context, profileID string, filePath string, opts *IngestOptions) (*domain.IngestResponse, error) {
	// TODO: Implement profile-based ingestion when profile switching is available
	// For now, just use regular ingestion
	return c.IngestFile(ctx, filePath, opts)
}

// QueryWithProfile queries using a specific profile
func (c *Client) QueryWithProfile(ctx context.Context, profileID string, query string, opts *QueryOptions) (*domain.QueryResponse, error) {
	// TODO: Implement profile-based query when profile switching is available
	// For now, just use regular query
	return c.Query(ctx, query, opts)
}

// IngestTextWithMetadata ingests text with custom metadata
func (c *Client) IngestTextWithMetadata(ctx context.Context, text, source string, metadata map[string]interface{}, opts *IngestOptions) (*domain.IngestResponse, error) {
	if opts == nil {
		opts = DefaultIngestOptions()
	}

	// Merge custom metadata with default metadata
	mergedMetadata := make(map[string]interface{})
	if opts.Metadata != nil {
		for k, v := range opts.Metadata {
			mergedMetadata[k] = v
		}
	}
	for k, v := range metadata {
		mergedMetadata[k] = v
	}

	// Create modified options
	modifiedOpts := *opts
	modifiedOpts.Metadata = mergedMetadata

	req := domain.IngestRequest{
	Content:   text,
		ChunkSize: opts.ChunkSize,
		Overlap:   opts.Overlap,
		Metadata:  modifiedOpts.Metadata,
	}

	resp, err := c.processor.Ingest(ctx, req)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Enhanced generation methods with profiles

// GenerateWithContext generates text with context awareness
func (c *Client) GenerateWithContext(ctx context.Context, prompt string, contextDocs []domain.Chunk, opts *domain.GenerationOptions) (string, error) {
	if c.llm == nil {
		return "", fmt.Errorf("LLM not available")
	}

	// Build context string
	var contextStr string
	if len(contextDocs) > 0 {
		var contexts []string
		for _, chunk := range contextDocs {
			contexts = append(contexts, fmt.Sprintf("[Context: %s]\n%s", chunk.Metadata["source"], chunk.Content))
		}
		contextStr = strings.Join(contexts, "\n\n---\n\n")
	}

	// Combine context and prompt
	fullPrompt := prompt
	if contextStr != "" {
		fullPrompt = fmt.Sprintf("Context:\n%s\n\nQuestion:\n%s", contextStr, prompt)
	}

	return c.llm.Generate(ctx, fullPrompt, opts)
}

// StreamWithContext streams generation with context awareness
func (c *Client) StreamWithContext(ctx context.Context, prompt string, contextDocs []domain.Chunk, opts *domain.GenerationOptions, callback func(string)) error {
	if c.llm == nil {
		return fmt.Errorf("LLM not available")
	}

	// Build context string
	var contextStr string
	if len(contextDocs) > 0 {
		var contexts []string
		for _, chunk := range contextDocs {
			contexts = append(contexts, fmt.Sprintf("[Context: %s]\n%s", chunk.Metadata["source"], chunk.Content))
		}
		contextStr = strings.Join(contexts, "\n\n---\n\n")
	}

	// Combine context and prompt
	fullPrompt := prompt
	if contextStr != "" {
		fullPrompt = fmt.Sprintf("Context:\n%s\n\nQuestion:\n%s", contextStr, prompt)
	}

	return c.llm.Stream(ctx, fullPrompt, opts, callback)
}