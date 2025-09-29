package rag

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/rag/chunker"
	"github.com/liliang-cn/rago/v2/pkg/rag/processor"
	"github.com/liliang-cn/rago/v2/pkg/rag/store"
)

// Client provides high-level RAG operations
type Client struct {
	processor   *processor.Service
	vectorStore domain.VectorStore
	docStore    *store.DocumentStore
	embedder    domain.Embedder
	llm         domain.Generator
	config      *config.Config
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

	return &Client{
		processor:   proc,
		vectorStore: vectorStore,
		docStore:    docStore,
		embedder:    embedder,
		llm:         llm,
		config:      cfg,
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