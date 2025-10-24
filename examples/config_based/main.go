package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/providers"
	"github.com/liliang-cn/rago/v2/pkg/rag"
)

func main() {
	ctx := context.Background()

	// Example 1: Load configuration from file
	fmt.Println("=== Example 1: Configuration from File ===")

	configPath := "rago.toml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Printf("Failed to load config from %s: %v", configPath, err)
		fmt.Println("Creating example configuration file...")
		createExampleConfig()
		return
	}

	fmt.Printf("Successfully loaded configuration from: %s\n", configPath)
	printConfiguration(cfg)

	// Example 2: Initialize providers from configuration
	fmt.Println("\n=== Example 2: Provider Initialization ===")

	// Create providers using factory
	factory := providers.NewFactory()

	llmProvider, err := factory.CreateLLMProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}
	defer func() {
		if closer, ok := llmProvider.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()

	embedderProvider, err := factory.CreateEmbedderProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)
	if err != nil {
		log.Fatalf("Failed to create embedder provider: %v", err)
	}
	defer func() {
		if closer, ok := embedderProvider.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()

	fmt.Printf("Successfully created providers:\n")
	fmt.Printf("  LLM Provider: %s\n", llmProvider.ProviderType())
	fmt.Printf("  Embedder Provider: %s\n", embedderProvider.ProviderType())

	// Example 3: Create RAG client with configuration
	fmt.Println("\n=== Example 3: RAG Client with Configuration ===")

	// Create metadata extractor (optional, can be nil)
	var metadataExtractor interface{} = nil

	// Create RAG client
	client, err := rag.NewClient(cfg, embedderProvider, llmProvider, metadataExtractor)
	if err != nil {
		log.Fatalf("Failed to create RAG client: %v", err)
	}
	defer client.Close()

	fmt.Printf("Successfully created RAG client\n")

	// Example 4: Test the configuration
	fmt.Println("\n=== Example 4: Testing Configuration ===")

	// Test with sample content
	sampleContent := `This is a sample document to test the RAGO configuration.
It contains multiple sentences that will be split into chunks.
The system should be able to search and retrieve relevant information.
Configuration-based initialization allows flexible deployment.`

	// Ingest content
	resp, err := client.IngestText(ctx, sampleContent, "config-test-doc", rag.DefaultIngestOptions())
	if err != nil {
		log.Printf("Failed to ingest content: %v", err)
	} else {
		fmt.Printf("Successfully ingested test content\n")
		fmt.Printf("Document ID: %s\n", resp.DocumentID)
		fmt.Printf("Chunks created: %d\n", resp.ChunkCount)
	}

	// Test query
	queryResp, err := client.Query(ctx, "What is this document about?", rag.DefaultQueryOptions())
	if err != nil {
		log.Printf("Failed to query: %v", err)
	} else {
		fmt.Printf("Successfully queried test content\n")
		fmt.Printf("Answer: %s\n", queryResp.Answer)
		fmt.Printf("Sources: %d\n", len(queryResp.Sources))
	}

	// Example 5: Configuration validation
	fmt.Println("\n=== Example 5: Configuration Validation ===")

	if err := cfg.Validate(); err != nil {
		fmt.Printf("Configuration validation failed: %v\n", err)
	} else {
		fmt.Printf("Configuration validation passed!\n")
	}

	fmt.Println("\n=== Configuration-based Examples completed successfully! ===")
}

func printConfiguration(cfg *config.Config) {
	fmt.Printf("Server Configuration:\n")
	fmt.Printf("  Port: %d\n", cfg.Server.Port)
	fmt.Printf("  Host: %s\n", cfg.Server.Host)
	fmt.Printf("  UI Enabled: %t\n", cfg.Server.EnableUI)

	fmt.Printf("\nProviders Configuration:\n")
	fmt.Printf("  Default LLM: %s\n", cfg.Providers.DefaultLLM)
	fmt.Printf("  Default Embedder: %s\n", cfg.Providers.DefaultEmbedder)

	if cfg.Providers.ProviderConfigs.OpenAI != nil {
		fmt.Printf("  OpenAI Provider:\n")
		fmt.Printf("    Base URL: %s\n", cfg.Providers.ProviderConfigs.OpenAI.BaseURL)
		fmt.Printf("    LLM Model: %s\n", cfg.Providers.ProviderConfigs.OpenAI.LLMModel)
		fmt.Printf("    Embedding Model: %s\n", cfg.Providers.ProviderConfigs.OpenAI.EmbeddingModel)
		fmt.Printf("    Timeout: %v\n", cfg.Providers.ProviderConfigs.OpenAI.Timeout)
	}

	fmt.Printf("\nVector Store Configuration:\n")
	fmt.Printf("  Database Path: %s\n", cfg.Sqvect.DBPath)
	fmt.Printf("  Top K: %d\n", cfg.Sqvect.TopK)
	fmt.Printf("  Threshold: %.2f\n", cfg.Sqvect.Threshold)

	fmt.Printf("\nChunker Configuration:\n")
	fmt.Printf("  Chunk Size: %d\n", cfg.Chunker.ChunkSize)
	fmt.Printf("  Overlap: %d\n", cfg.Chunker.Overlap)
	fmt.Printf("  Method: %s\n", cfg.Chunker.Method)

	fmt.Printf("\nMCP Configuration:\n")
	fmt.Printf("  Enabled: %t\n", cfg.MCP.Enabled)
	fmt.Printf("  Servers Config Path: %s\n", cfg.MCP.ServersConfigPath)
}

func createExampleConfig() {
	exampleConfig := `# RAGO v2 Configuration Example
# This file demonstrates all available configuration options

[server]
port = 7127
host = "0.0.0.0"
enable_ui = false

[providers]
default_llm = "openai"
default_embedder = "openai"

[providers.openai]
type = "openai"
base_url = "http://localhost:11434/v1"  # Change this to your LLM endpoint
api_key = "ollama"                      # Change this to your API key
llm_model = "qwen3"                     # Change this to your preferred model
embedding_model = "nomic-embed-text"   # Change this to your embedding model
timeout = "30s"

[sqvect]
db_path = "~/.rago/data/rag.db"
top_k = 5
threshold = 0.0

[chunker]
chunk_size = 500
overlap = 50
method = "sentence"

[mcp]
enabled = true
servers_config_path = "mcpServers.json"
log_level = "info"
default_timeout = "30s"
max_concurrent_requests = 5
health_check_interval = "60s"
`

	err := os.WriteFile("rago.toml", []byte(exampleConfig), 0644)
	if err != nil {
		log.Printf("Failed to create example config: %v", err)
		return
	}

	fmt.Printf("Created example configuration file: rago.toml\n")
	fmt.Printf("Please edit the file with your settings and run again.\n")
}