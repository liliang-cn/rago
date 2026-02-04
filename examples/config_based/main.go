package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

func main() {
	ctx := context.Background()

	// Load configuration from file (simulated by creating default config here)
	// In a real scenario, you would use: cfg, err := config.Load("path/to/config.toml")
	cfg := &config.Config{}

	fmt.Println("=== Config-Based RAGO Initialization ===")

	// Example 1: Accessing config values
	fmt.Printf("Database Path: %s\n", cfg.Sqvect.DBPath)
	fmt.Printf("Chunk Size: %d\n", cfg.Chunker.ChunkSize)
	fmt.Printf("Overlap: %d\n", cfg.Chunker.Overlap)

	// Example 2: Initialize providers from configuration
	// Note: Config structure changed, manual mapping required for examples using old structure
	provConfig := &domain.OpenAIProviderConfig{
		BaseProviderConfig: domain.BaseProviderConfig{
			Timeout: 30 * time.Second,
		},
		BaseURL:        "http://localhost:11434/v1",
		APIKey:         "ollama",
		LLMModel:       "qwen2.5-coder:14b",
		EmbeddingModel: "nomic-embed-text",
	}

	fmt.Printf("\nProviders Configuration:\n")
	fmt.Printf("  Default LLM: %s\n", "openai")
	fmt.Printf("  Default Embedder: %s\n", "openai")
	
	fmt.Printf("    Base URL: %s\n", provConfig.BaseURL)
	fmt.Printf("    LLM Model: %s\n", provConfig.LLMModel)
	fmt.Printf("    Embedding Model: %s\n", provConfig.EmbeddingModel)
	fmt.Printf("    Timeout: %v\n", provConfig.Timeout)

	// Create providers using factory
	factory := providers.NewFactory()

	_, err := factory.CreateLLMProvider(ctx, provConfig)
	if err != nil {
		log.Printf("Failed to create LLM provider: %v", err)
	} else {
		fmt.Printf("Successfully created LLM provider\n")
	}

	_, err = factory.CreateEmbedderProvider(ctx, provConfig)
	if err != nil {
		log.Printf("Failed to create Embedder provider: %v", err)
	} else {
		fmt.Printf("Successfully created Embedder provider\n")
	}

	// Note: Client initialization would follow similar patterns to other examples
	fmt.Println("\nConfiguration loading example completed successfully!")
}
