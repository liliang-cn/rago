package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/pool"
	"github.com/liliang-cn/rago/v2/pkg/providers"
	"github.com/liliang-cn/rago/v2/pkg/rag"
)

func main() {
	ctx := context.Background()

	// 1. Initialize Configuration
	cfg := &config.Config{}

	provider := pool.Provider{
		Name:           "openai",
		BaseURL:        "http://localhost:11434/v1", // Ollama
		Key:            "ollama",
		ModelName:      "qwen2.5-coder:14b",
		MaxConcurrency: 10,
	}

	cfg.Sqvect.DBPath = "./data/rag_advanced.db"
	cfg.Chunker.ChunkSize = 500
	cfg.Chunker.Overlap = 50

	// 2. Initialize Components
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

	// Create providers using factory
	embedder, err := factory.CreateEmbedderProvider(ctx, provConfig)
	if err != nil {
		log.Fatalf("Failed to create Embedder provider: %v", err)
	}

	llm, err := factory.CreateLLMProvider(ctx, provConfig)
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}

	// 3. Initialize Client with Advanced Options
	// (Assuming we might want to customize the metadata extractor later)
	client, err := rag.NewClient(cfg, embedder, llm, nil)
	if err != nil {
		log.Fatalf("Failed to create RAG client: %v", err)
	}
	defer client.Close()

	// 4. Advanced Ingestion: Adding Metadata
	fmt.Println("Ingesting with metadata...")

	docContent := `RAGO supports advanced metadata filtering. 
This allows users to query specific subsets of documents based on tags, dates, or other attributes.
Metadata is stored alongside vectors in the SQLite database.`
	
	metadata := map[string]interface{}{
		"category": "documentation",
		"topic":    "metadata",
		"version":  "2.0",
		"author":   "RAGO Team",
	}

	ingestOpts := &rag.IngestOptions{
		Metadata:  metadata,
	}

	result, err := client.IngestText(ctx, docContent, "metadata_guide.md", ingestOpts)
	if err != nil {
		log.Fatalf("Ingestion failed: %v", err)
	}
	fmt.Printf("Ingested document ID: %s with %d chunks\n", result.DocumentID, result.ChunkCount)

	// 5. Advanced Query: Using Filters
	fmt.Println("\nQuerying with filters...")

	query := "How does metadata filtering work?"
	
	// Create query options with filters
	queryOpts := &rag.QueryOptions{
		TopK:        3,
		Temperature: 0.1,
		ShowSources: true,
		Filters: map[string]interface{}{
			"category": "documentation",
			"topic":    "metadata",
		},
	}

	answer, err := client.Query(ctx, query, queryOpts)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Printf("Question: %s\n", query)
	fmt.Printf("Answer: %s\n", answer.Answer)
	
	fmt.Println("\nSources found (filtered):")
	for i, src := range answer.Sources {
		source := "unknown"
		if src.Metadata != nil {
			if s, ok := src.Metadata["source"].(string); ok {
				source = s
			}
		}
		fmt.Printf("[%d] %s (Score: %.4f)\n    Metadata: %v\n", i+1, source, src.Score, src.Metadata)
	}

	// 6. Demonstrate Stats
	stats, err := client.GetStats(ctx)
	if err == nil {
		fmt.Printf("\nSystem Stats:\n  Documents: %d\n  Chunks: %d\n", stats.TotalDocuments, stats.TotalChunks)
	}

	fmt.Println("\nAdvanced features example completed successfully!")
}
