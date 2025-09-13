package client

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/rag/chunker"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/rag/processor"
	"github.com/liliang-cn/rago/v2/pkg/rag/store"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

// Client represents the main rago client
type Client struct {
	config      *config.Config
	processor   *processor.Service
	vectorStore *store.SQLiteStore
	embedder    domain.Embedder
	llm         domain.Generator
	mcpClient   *MCPClient  // MCP functionality
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
	vectorStore, err := store.NewSQLiteStore(
		cfg.Sqvect.DBPath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	docStore := store.NewDocumentStore(vectorStore.GetSqvectStore())

	// Initialize services using provider system
	ctx := context.Background()
	embedService, llmService, metadataExtractor, err := providers.InitializeProviders(ctx, cfg)
	if err != nil {
		if err := vectorStore.Close(); err != nil {
			fmt.Printf("Warning: failed to close vector store during cleanup: %v\n", err)
		}
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	chunkerService := chunker.New()

	processor := processor.New(
		embedService,
		llmService,
		chunkerService,
		vectorStore,
		docStore,
		cfg,
		metadataExtractor,
	)

	return &Client{
		config:      cfg,
		processor:   processor,
		vectorStore: vectorStore,
		embedder:    embedService,
		llm:         llmService,
	}, nil
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	var errs []error

	if c.vectorStore != nil {
		if err := c.vectorStore.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close vector store: %w", err))
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
