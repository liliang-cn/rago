package client

import (
	"context"
	"fmt"

	"github.com/liliang-cn/rago/v2/pkg/chunker"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/processor"
	"github.com/liliang-cn/rago/v2/pkg/store"
	"github.com/liliang-cn/rago/v2/pkg/utils"
)

// Client represents the main rago client
type Client struct {
	config       *config.Config
	processor    *processor.Service
	vectorStore  *store.SQLiteStore
	keywordStore *store.KeywordStore
	embedder     domain.Embedder
	llm          domain.Generator
	mcpClient    *MCPClient  // MCP functionality
	taskClient   *TaskClient // Task scheduling functionality
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

	keywordStore, err := store.NewKeywordStore(cfg.Keyword.IndexPath)
	if err != nil {
		if err := vectorStore.Close(); err != nil {
			fmt.Printf("Warning: failed to close vector store during cleanup: %v\n", err)
		}
		return nil, fmt.Errorf("failed to create keyword store: %w", err)
	}

	docStore := store.NewDocumentStore(vectorStore.GetSqvectStore())

	// Initialize services using provider system
	ctx := context.Background()
	embedService, llmService, metadataExtractor, err := utils.InitializeProviders(ctx, cfg)
	if err != nil {
		if err := vectorStore.Close(); err != nil {
			fmt.Printf("Warning: failed to close vector store during cleanup: %v\n", err)
		}
		if err := keywordStore.Close(); err != nil {
			fmt.Printf("Warning: failed to close keyword store during cleanup: %v\n", err)
		}
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

// Close closes the client and releases resources
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
		return fmt.Errorf("errors closing client: %v", errs)
	}

	return nil
}

// GetConfig returns the client configuration
func (c *Client) GetConfig() *config.Config {
	return c.config
}
