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
	// We'll create a default config instead of loading from file for this example
	cfg := &config.Config{}

	// Configure providers via pool settings
	// Note: In a real app, you'd use config.LLMPool.Providers list
	provider := pool.Provider{
		Name:           "openai",
		BaseURL:        "http://localhost:11434/v1", // Ollama
		Key:            "ollama",
		ModelName:      "qwen2.5-coder:14b",
		MaxConcurrency: 10,
	}

	cfg.LLMPool.Providers = []pool.Provider{provider}
	cfg.EmbeddingPool.Providers = []pool.Provider{provider}

	cfg.Sqvect.DBPath = "./data/rag.db"
	cfg.Chunker.ChunkSize = 500
	cfg.Chunker.Overlap = 50

	// 2. Initialize Components
	// Create providers using factory
	factory := providers.NewFactory()
	
	// Manually map to provider config for factory
	provConfig := &domain.OpenAIProviderConfig{
		BaseProviderConfig: domain.BaseProviderConfig{
			Timeout: 30 * time.Second,
		},
		BaseURL:        provider.BaseURL,
		APIKey:         provider.Key,
		LLMModel:       provider.ModelName,
		EmbeddingModel: "nomic-embed-text", // Hardcoded default for example
	}
	
	llm, err := factory.CreateLLMProvider(ctx, provConfig)
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}

	embedder, err := factory.CreateEmbedderProvider(ctx, provConfig)
	if err != nil {
		log.Fatalf("Failed to create Embedder provider: %v", err)
	}

	// 3. Initialize RAG Client
	client, err := rag.NewClient(cfg, embedder, llm, nil)
	if err != nil {
		log.Fatalf("Failed to create RAG client: %v", err)
	}
	defer client.Close()

	// 4. Ingest Documents
	fmt.Println("Ingesting documents...")
	
	// Create sample documents
	docs := []struct {
		Name    string
		Content string
	}{
		{
			Name: "rago_intro.txt",
			Content: `RAGO is a lightweight RAG (Retrieval-Augmented Generation) system written in Go.
It provides a modular architecture for building LLM applications with context.
Key features include support for multiple vector stores (SQLite, Qdrant), 
various LLM providers (OpenAI, Ollama), and an MCP (Model Context Protocol) implementation.`,
		},
		{
			Name: "go_concurrency.txt",
			Content: `Go's concurrency model is based on goroutines and channels.
Goroutines are lightweight threads managed by the Go runtime.
Channels provide a way for goroutines to communicate and synchronize execution.
The 'select' statement allows waiting on multiple channel operations.`,
		},
	}

	for _, doc := range docs {
		result, err := client.IngestText(ctx, doc.Content, doc.Name, rag.DefaultIngestOptions())
		if err != nil {
			log.Printf("Failed to ingest %s: %v", doc.Name, err)
			continue
		}
		fmt.Printf("Ingested %s: %d chunks, document ID: %s\n", doc.Name, result.ChunkCount, result.DocumentID)
	}

	// 5. Query Knowledge Base
	fmt.Println("\nQuerying knowledge base...")
	
	query := "What are the key features of RAGO?"
	answer, err := client.Query(ctx, query, rag.DefaultQueryOptions())
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Printf("Question: %s\n", query)
	fmt.Printf("Answer: %s\n", answer.Answer)
	fmt.Printf("Sources used: %d\n", len(answer.Sources))

	fmt.Println("\nBasic RAG usage example completed successfully!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
