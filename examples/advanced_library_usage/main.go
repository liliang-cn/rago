package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/providers"
	"github.com/liliang-cn/rago/v2/pkg/rag"
	"github.com/liliang-cn/rago/v2/pkg/pool"
)

func main() {
	ctx := context.Background()

	// 1. Configure RAGO
	// Using struct initialization for complete control
	cfg := &config.Config{
		Sqvect: config.SqvectConfig{
			DBPath:    "./data/library_usage.db",
			TopK:      5,
			Threshold: 0.3,
		},
		Chunker: config.ChunkerConfig{
			ChunkSize: 500,
			Overlap:   50,
			Method:    "sentence",
		},
		Ingest: config.IngestConfig{
			MetadataExtraction: config.MetadataExtractionConfig{
				Enable: true,
			},
		},
	}
	
	// Create provider config for pool
	provider := pool.Provider{
		Name:           "openai",
		BaseURL:        "http://localhost:11434/v1", // Ollama
		Key:            "ollama",
		ModelName:      "qwen2.5-coder:14b",
		MaxConcurrency: 10,
	}

	cfg.LLMPool.Providers = []pool.Provider{provider}
	cfg.EmbeddingPool.Providers = []pool.Provider{provider}

	// 2. Initialize Providers
	factory := providers.NewFactory()

	provConfig := &domain.OpenAIProviderConfig{
		BaseProviderConfig: domain.BaseProviderConfig{
			Timeout: 30 * time.Second,
		},
		BaseURL:        provider.BaseURL,
		APIKey:         provider.Key,
		LLMModel:       provider.ModelName,
		EmbeddingModel: "nomic-embed-text",
	}

	llm, _ := factory.CreateLLMProvider(ctx, provConfig)
	embedder, _ := factory.CreateEmbedderProvider(ctx, provConfig)

	// 3. Initialize Client
	// For advanced usage, we might want custom metadata extractor or other services
	client, err := rag.NewClient(cfg, embedder, llm, nil) // Using default metadata extractor (LLM-based)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 4. Advanced Ingestion with Custom Options
	fmt.Println("Ingesting documents...")
	
	files := []string{"docs/LIBRARY_USAGE.md", "README.md"}
	for _, file := range files {
		// Read file content (simulated)
		content, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Skipping %s: %v", file, err)
			continue
		}

		// Use advanced options
		opts := &rag.IngestOptions{
			Metadata: map[string]interface{}{
				"source":   "repo",
				"filetype": "markdown",
				"priority": "high",
			},
		}

		resp, err := client.IngestText(ctx, string(content), file, opts)
		if err != nil {
			log.Printf("Failed to ingest %s: %v", file, err)
			continue
		}
		fmt.Printf("Ingested %s: %d chunks, document ID: %s\n", file, resp.ChunkCount, resp.DocumentID)
	}

	// 5. Advanced Querying with Filters
	fmt.Println("\nQuerying knowledge base...")

	query := "How do I use the library in my Go project?"
	
	// Advanced query options
	queryOpts := &rag.QueryOptions{
		TopK:        3,
		Temperature: 0.1, // More deterministic
		MaxTokens:   500,
		ShowSources: true,
		Filters: map[string]interface{}{
			"filetype": "markdown", // Filter by metadata
		},
	}

	result, err := client.Query(ctx, query, queryOpts)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Printf("\nQuestion: %s\n", query)
	fmt.Printf("Answer: %s\n", result.Answer)
	
	fmt.Println("\nSources:")
	for i, source := range result.Sources {
		src := "unknown"
		if source.Metadata != nil {
			if s, ok := source.Metadata["source"].(string); ok {
				src = s
			}
		}
		fmt.Printf("[%d] %s (Score: %.4f)\n", i+1, src, source.Score)
		// Print snippet if needed
		if len(source.Content) > 100 {
			fmt.Printf("    %s...\n", source.Content[:100])
		} else {
			fmt.Printf("    %s\n", source.Content)
		}
	}

	// 6. Get statistics
	stats, err := client.GetStats(ctx)
	if err == nil {
		fmt.Printf("\nTotal chunks in system: %d\n", stats.TotalChunks)
	}
}