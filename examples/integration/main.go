package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/liliang-cn/rago/internal/config"
	"github.com/liliang-cn/rago/internal/domain"
	"github.com/liliang-cn/rago/internal/embedder"
	"github.com/liliang-cn/rago/internal/store"
)

func main() {
	// Set up configuration
	cfg := &config.Config{
		Sqvect: config.SqvectConfig{
			DBPath:    "./data/integration_test.db",
			VectorDim: 768,
			MaxConns:  10,
			BatchSize: 100,
			TopK:      5,
			Threshold: 0.0,
		},
		Ollama: config.OllamaConfig{
			BaseURL:        "http://localhost:11434",
			EmbeddingModel: "nomic-embed-text",
		},
	}

	// Create data directory if it doesn't exist
	dataDir := filepath.Dir(cfg.Sqvect.DBPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize the vector store
	fmt.Println("Initializing vector store...")
	vectorStore, err := store.NewSQLiteStore(
		cfg.Sqvect.DBPath,
		cfg.Sqvect.VectorDim,
		cfg.Sqvect.MaxConns,
		cfg.Sqvect.BatchSize,
	)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}
	defer vectorStore.Close()

	// Initialize document store
	docStore := store.NewDocumentStore(vectorStore.GetSqvectStore())

	// Initialize embedder
	fmt.Println("Initializing embedder...")
	embedder, err := embedder.NewOllamaService(
		cfg.Ollama.BaseURL,
		cfg.Ollama.EmbeddingModel,
	)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	ctx := context.Background()

	// Reset the store to start fresh
	fmt.Println("Resetting store...")
	if err := vectorStore.Reset(ctx); err != nil {
		log.Fatalf("Failed to reset store: %v", err)
	}

	// Sample documents to ingest
	documents := []domain.Document{
		{
			ID:      "doc1",
			Content: "Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.",
			Path:    "/docs/go/intro.md",
			Created: time.Now(),
			Metadata: map[string]interface{}{
				"title": "Introduction to Go",
				"tags":  []string{"go", "programming", "intro"},
			},
		},
		{
			ID:      "doc2",
			Content: "Concurrency is the composition of independently executing processes, while parallelism is the simultaneous execution of (possibly related) computations. Concurrency is about dealing with lots of things at once. Parallelism is about doing lots of things at once.",
			Path:    "/docs/go/concurrency.md",
			Created: time.Now(),
			Metadata: map[string]interface{}{
				"title": "Go Concurrency",
				"tags":  []string{"go", "concurrency", "parallelism"},
			},
		},
		{
			ID:      "doc3",
			Content: "Interfaces in Go provide a way to specify the behavior of an object. If something can do this, then it can be used here. Interfaces are implemented implicitly. A type never declares that it implements an interface, it just implements the methods.",
			Path:    "/docs/go/interfaces.md",
			Created: time.Now(),
			Metadata: map[string]interface{}{
				"title": "Go Interfaces",
				"tags":  []string{"go", "interfaces", "types"},
			},
		},
	}

	// Ingest documents
	fmt.Println("Ingesting documents...")
	for _, doc := range documents {
		fmt.Printf("  Storing document: %s\n", doc.ID)
		if err := docStore.Store(ctx, doc); err != nil {
			log.Printf("Warning: failed to store document %s: %v", doc.ID, err)
			continue
		}

		// Create chunks for the document
		chunk := domain.Chunk{
			ID:         fmt.Sprintf("%s-chunk1", doc.ID),
			DocumentID: doc.ID,
			Content:    doc.Content,
			Metadata: map[string]interface{}{
				"chunk_number": 1,
			},
		}

		// Generate embedding for the chunk
		fmt.Printf("  Generating embedding for chunk: %s\n", chunk.ID)
		vector, err := embedder.Embed(ctx, chunk.Content)
		if err != nil {
			log.Printf("Warning: failed to embed chunk %s: %v", chunk.ID, err)
			continue
		}
		chunk.Vector = vector

		// Store the chunk
		fmt.Printf("  Storing chunk: %s\n", chunk.ID)
		if err := vectorStore.Store(ctx, []domain.Chunk{chunk}); err != nil {
			log.Printf("Warning: failed to store chunk %s: %v", chunk.ID, err)
			continue
		}
	}

	fmt.Println("Document ingestion completed.")

	// List documents
	fmt.Println("\nListing documents...")
	docs, err := docStore.List(ctx)
	if err != nil {
		log.Fatalf("Failed to list documents: %v", err)
	}
	for _, doc := range docs {
		fmt.Printf("  Document: %s (%s)\n", doc.ID, doc.Path)
	}

	// Perform a search
	fmt.Println("\nPerforming search...")
	query := "What is Go programming language?"
	fmt.Printf("Query: %s\n", query)

	// Generate embedding for the query
	queryVector, err := embedder.Embed(ctx, query)
	if err != nil {
		log.Fatalf("Failed to embed query: %v", err)
	}

	// Search for similar chunks
	results, err := vectorStore.Search(ctx, queryVector, cfg.Sqvect.TopK)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}

	fmt.Printf("Found %d results:\n", len(results))
	for i, result := range results {
		fmt.Printf("  %d. Document: %s, Score: %.4f\n", i+1, result.DocumentID, result.Score)
		fmt.Printf("     Content: %.100s...\n", result.Content)
	}

	fmt.Println("\nIntegration test completed successfully!")
}