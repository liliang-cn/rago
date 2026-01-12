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
)

func main() {
	ctx := context.Background()

	// Load configuration (can be empty for defaults)
	cfg, err := config.Load("")
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		// Create minimal config for demo
		apiKey := getEnvOrDefault("RAGO_OPENAI_API_KEY", "")  // Empty for local services
		cfg = &config.Config{
			Server: config.ServerConfig{
				Port: 7127,
				Host: "0.0.0.0",
			},
			Sqvect: config.SqvectConfig{
				DBPath:    "./test_rag.db",  // Use local file for demo
				MaxConns:  10,
				BatchSize: 100,
				TopK:      5,
				Threshold: 0.0,
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
			Providers: config.ProvidersConfig{
				DefaultLLM:     "openai",
				DefaultEmbedder: "openai",
				ProviderConfigs: domain.ProviderConfig{
					OpenAI: &domain.OpenAIProviderConfig{
						BaseProviderConfig: domain.BaseProviderConfig{
							Type:    domain.ProviderOpenAI,
							Timeout: 30 * time.Second,
						},
						BaseURL:        getEnvOrDefault("RAGO_OPENAI_BASE_URL", "http://localhost:11434/v1"),
						APIKey:         apiKey,
						LLMModel:       getEnvOrDefault("RAGO_OPENAI_LLM_MODEL", "qwen3"),
						EmbeddingModel: getEnvOrDefault("RAGO_OPENAI_EMBEDDING_MODEL", "nomic-embed-text"),
					},
				},
			},
		}
	}

	// The configuration is already set up above, either from file or fallback

	// Create providers using factory
	factory := providers.NewFactory()

	llm, err := factory.CreateLLMProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	embedder, err := factory.CreateEmbedderProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}
	
	// Cast LLM to MetadataExtractor for GraphRAG
	var extractor domain.MetadataExtractor
	if e, ok := llm.(domain.MetadataExtractor); ok {
		extractor = e
	}

	// Create RAG client
	client, err := rag.NewClient(cfg, embedder, llm, extractor)
	if err != nil {
		log.Fatalf("Failed to create RAG client: %v", err)
	}
	defer client.Close()

	// Example 1: Ingest a text file
	fmt.Println("=== Example 1: Ingesting text ===")
	if len(os.Args) > 1 {
		filePath := os.Args[1]
		resp, err := client.IngestFile(ctx, filePath, rag.DefaultIngestOptions())
		if err != nil {
			log.Printf("Failed to ingest file: %v", err)
		} else {
			fmt.Printf("Successfully ingested file: %s\n", filePath)
			fmt.Printf("Document ID: %s\n", resp.DocumentID)
			fmt.Printf("Chunks created: %d\n", resp.ChunkCount)
		}
	} else {
		// Ingest some sample text if no file provided
		sampleText := `RAGO (Retrieval-Augmented Generation Offline) is a local RAG system.
It provides document ingestion, semantic search, and context-enhanced Q&A.
The system uses SQLite for vector storage and supports multiple LLM providers.
Key features include smart chunking, metadata extraction, and MCP tool integration.`

		resp, err := client.IngestText(ctx, sampleText, "sample-document", rag.DefaultIngestOptions())
		if err != nil {
			log.Printf("Failed to ingest text: %v", err)
		} else {
			fmt.Printf("Successfully ingested sample text\n")
			fmt.Printf("Document ID: %s\n", resp.DocumentID)
			fmt.Printf("Chunks created: %d\n", resp.ChunkCount)
		}
	}

	// Example 2: Query the knowledge base
	fmt.Println("\n=== Example 2: Querying knowledge base ===")
	queries := []string{
		"What is RAGO?",
		"What are the key features of RAGO?",
		"How does RAGO work?",
	}

	for _, query := range queries {
		fmt.Printf("\nQuery: %s\n", query)
		resp, err := client.Query(ctx, query, rag.DefaultQueryOptions())
		if err != nil {
			log.Printf("Failed to query: %v", err)
			continue
		}

		fmt.Printf("Answer: %s\n", resp.Answer)
		fmt.Printf("Sources found: %d\n", len(resp.Sources))

		// Show source information
		for i, source := range resp.Sources {
			fmt.Printf("  Source %d: Score=%.2f, Content preview: %.100s...\n",
				i+1, source.Score, source.Content)
		}
	}

	// Example 3: List documents and get stats
	fmt.Println("\n=== Example 3: Document management ===")

	// List all documents
	docs, err := client.ListDocuments(ctx)
	if err != nil {
		log.Printf("Failed to list documents: %v", err)
	} else {
		fmt.Printf("Total documents in store: %d\n", len(docs))
		for i, doc := range docs {
			fmt.Printf("  Document %d: ID=%s, Path=%s, Created=%s\n",
				i+1, doc.ID, doc.Path, doc.Created.Format("2006-01-02 15:04:05"))
		}
	}

	// Get statistics
	stats, err := client.GetStats(ctx)
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		fmt.Printf("\nStatistics:\n")
		fmt.Printf("  Total documents: %d\n", stats.TotalDocuments)
		fmt.Printf("  Total chunks: %d\n", stats.TotalChunks)
	}

	// Example 4: Check MCP Status
	fmt.Println("\n=== Example 4: MCP Status ===")
	mcpStatus, err := client.GetMCPStatus(ctx)
	if err != nil {
		log.Printf("Failed to get MCP status: %v", err)
	} else {
		if statusMap, ok := mcpStatus.(map[string]interface{}); ok {
			fmt.Printf("MCP Enabled: %v\n", statusMap["enabled"])
			fmt.Printf("Message: %s\n", statusMap["message"])

			if servers, ok := statusMap["servers"].([]interface{}); ok {
				fmt.Printf("MCP Servers (%d):\n", len(servers))
				for i, server := range servers {
					if serverMap, ok := server.(map[string]interface{}); ok {
						name := serverMap["name"]
						description := serverMap["description"]
						running := serverMap["running"]
						toolCount := serverMap["tool_count"]

						fmt.Printf("  %d. %s: %v (%d tools)\n", i+1, name, running, toolCount)
						fmt.Printf("     Description: %s\n", description)
					}
				}
			}
		}
	}

	fmt.Println("\n=== Examples completed successfully! ===")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}