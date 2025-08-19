package rago

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/liliang-cn/rago/internal/chunker"
	"github.com/liliang-cn/rago/internal/config"
	"github.com/liliang-cn/rago/internal/domain"
	"github.com/liliang-cn/rago/internal/embedder"
	"github.com/liliang-cn/rago/internal/llm"
	"github.com/liliang-cn/rago/internal/processor"
	"github.com/liliang-cn/rago/internal/store"
)

type Client struct {
	config    *config.Config
	processor *processor.Service
	store     *store.SQLiteStore
	embedder  *embedder.OllamaService
	llm       *llm.OllamaService
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

	docStore := store.NewDocumentStore(vectorStore.GetSqvectStore())

	embedService, err := embedder.NewOllamaService(
		cfg.Ollama.BaseURL,
		cfg.Ollama.EmbeddingModel,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	llmService, err := llm.NewOllamaService(
		cfg.Ollama.BaseURL,
		cfg.Ollama.LLMModel,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM service: %w", err)
	}

	chunkerService := chunker.New()

	processor := processor.New(
		embedService,
		llmService,
		chunkerService,
		vectorStore,
		docStore,
		cfg,
		llmService,
	)

	return &Client{
		config:    cfg,
		processor: processor,
		store:     vectorStore,
		embedder:  embedService,
		llm:       llmService,
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
		MaxTokens:    500,
		Stream:       true,
		ShowThinking: true,
	}

	return c.processor.Query(ctx, req)
}

func (c *Client) QueryWithFilters(query string, filters map[string]interface{}) (domain.QueryResponse, error) {
	ctx := context.Background()
	req := domain.QueryRequest{
		Query:        query,
		TopK:         c.config.Sqvect.TopK,
		Temperature:  0.7,
		MaxTokens:    500,
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
		MaxTokens:    500,
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
		MaxTokens:    500,
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
	if c.store != nil {
		return c.store.Close()
	}
	return nil
}

func (c *Client) GetConfig() *config.Config {
	return c.config
}

type StatusResult struct {
	OllamaAvailable bool
	BaseURL         string
	LLMModel        string
	EmbeddingModel  string
	Timeout         time.Duration
	Error           error
}

func (c *Client) CheckStatus() StatusResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := StatusResult{
		BaseURL:        c.config.Ollama.BaseURL,
		LLMModel:       c.config.Ollama.LLMModel,
		EmbeddingModel: c.config.Ollama.EmbeddingModel,
		Timeout:        c.config.Ollama.Timeout,
	}

	if err := c.llm.Health(ctx); err != nil {
		result.OllamaAvailable = false
		result.Error = err
	} else {
		result.OllamaAvailable = true
	}

	return result
}