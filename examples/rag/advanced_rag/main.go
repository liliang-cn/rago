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

	// 1. Configure RAGO
	cfg := &config.Config{
		Sqvect: config.SqvectConfig{
			DBPath:    "./data/advanced_rag.db",
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

	provider := pool.Provider{
		Name:           "openai",
		BaseURL:        "http://localhost:11434/v1",
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
	client, err := rag.NewClient(cfg, embedder, llm, nil)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 4. Advanced Ingestion with Custom Options
	fmt.Println("Ingesting sample documents...")

	docs := []struct {
		name    string
		content string
	}{
		{
			name: "rago_intro.txt",
			content: `RAGO is an Autonomous Agent & RAG Library for Go.
It enables you to build autonomous agents with "hands" (MCP Tools & Skills),
"brains" (Planning & Reasoning), and "memory" (Vector RAG & GraphRAG).`,
		},
		{
			name: "features.txt",
			content: `RAGO supports multiple LLM providers (OpenAI, Ollama, DeepSeek),
hybrid RAG (Vector + Knowledge Graph), and MCP tools integration.
It's designed to be the intelligence layer of your application.`,
		},
	}

	for _, doc := range docs {
		opts := &rag.IngestOptions{
			Metadata: map[string]interface{}{
				"source":   "example",
				"filetype": "text",
				"priority": "high",
			},
		}

		resp, err := client.IngestText(ctx, doc.content, doc.name, opts)
		if err != nil {
			log.Printf("Failed to ingest %s: %v", doc.name, err)
			continue
		}
		fmt.Printf("Ingested %s: %d chunks, document ID: %s\n", doc.name, resp.ChunkCount, resp.DocumentID)
	}

	// 5. Advanced Querying with Filters
	fmt.Println("\nQuerying knowledge base...")

	query := "What are the main features of RAGO?"

	queryOpts := &rag.QueryOptions{
		TopK:        3,
		Temperature: 0.1,
		MaxTokens:   500,
		ShowSources: true,
		Filters: map[string]interface{}{
			"priority": "high",
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

	fmt.Println("\nAdvanced RAG example completed successfully!")
}
