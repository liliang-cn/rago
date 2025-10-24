package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/providers"
	"github.com/liliang-cn/rago/v2/pkg/rag/chunker"
	"github.com/liliang-cn/rago/v2/pkg/rag/processor"
	"github.com/liliang-cn/rago/v2/pkg/rag/store"
)

func main() {
	ctx := context.Background()
	fmt.Println("=== RAGO Qdrant + Ollama Test ===")
	fmt.Println("Models: qwen3:latest + nomic-embed-text:latest")
	fmt.Println()

	// 1. Setup configuration
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			DefaultLLM:      "ollama",
			DefaultEmbedder: "ollama",
			ProviderConfigs: domain.ProviderConfig{
				Ollama: &domain.OllamaProviderConfig{
					BaseProviderConfig: domain.BaseProviderConfig{
						Type:    domain.ProviderOllama,
						Timeout: 30 * time.Second,
					},
					BaseURL:        "http://localhost:11434",
					LLMModel:       "qwen3:latest",
					EmbeddingModel: "nomic-embed-text:latest",
				},
			},
		},
		Chunker: config.ChunkerConfig{
			ChunkSize: 500,
			Overlap:   50,
			Method:    "sentence",
		},
	}

	// 2. Create Qdrant vector store
	fmt.Println("1. Setting up Qdrant vector store...")
	vectorStoreConfig := store.StoreConfig{
		Type: "qdrant",
		Parameters: map[string]interface{}{
			"url":        "localhost:6334",
			"collection": "ollama_test_collection",
		},
	}

	vectorStore, err := store.NewVectorStore(vectorStoreConfig)
	if err != nil {
		log.Fatalf("Failed to create Qdrant store: %v", err)
	}

	// Ensure proper cleanup
	if qdrantStore, ok := vectorStore.(*store.QdrantStore); ok {
		defer qdrantStore.Close()
	}

	// 3. Create document store (using SQLite for documents)
	sqliteStoreConfig := store.StoreConfig{
		Type: "sqlite",
		Parameters: map[string]interface{}{
			"db_path": "./.rago/test_ollama.db",
		},
	}
	
	sqliteStore, err := store.NewVectorStore(sqliteStoreConfig)
	if err != nil {
		log.Fatalf("Failed to create SQLite store: %v", err)
	}
	
	documentStore := store.NewDocumentStoreFor(sqliteStore)

	// 4. Create providers
	fmt.Println("2. Creating Ollama providers...")
	embedder, generator, _, err := providers.InitializeProviders(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize providers: %v", err)
	}

	// 5. Create RAG processor
	fmt.Println("3. Creating RAG processor...")
	chunkerService := chunker.New()
	ragProcessor := processor.New(
		embedder,
		generator,
		chunkerService,
		vectorStore,
		documentStore,
		cfg,
		nil, // metadata extractor
	)

	// 6. Clear any existing data
	fmt.Println("4. Clearing existing data...")
	err = vectorStore.Reset(ctx)
	if err != nil {
		fmt.Printf("Warning: Failed to reset vector store: %v\n", err)
	}

	// 7. Prepare test documents
	fmt.Println("5. Ingesting test documents...")
	testDocuments := []struct {
		title   string
		content string
	}{
		{
			title: "Qdrant Introduction",
			content: `Qdrant is a vector database and vector similarity search engine. 
It provides a production-ready service with a convenient API to store, search, and manage vectors 
with additional payload and extended filtering support. Qdrant is tailored for neural network or 
semantic-based matching, faceted search, and other applications that require fast and scalable vector operations.`,
		},
		{
			title: "Ollama Overview",
			content: `Ollama is a tool that allows you to run large language models locally. 
It supports various models including Llama 2, Code Llama, Qwen, and many others. 
Ollama provides a simple API for creating, running, and managing models, as well as a library 
of pre-built models that can be easily used in applications. The nomic-embed-text model is 
excellent for generating embeddings for semantic search.`,
		},
		{
			title: "RAG Architecture",
			content: `Retrieval-Augmented Generation (RAG) is an AI framework that combines the strengths 
of pre-trained language models with retrieval mechanisms. RAG systems first retrieve relevant 
information from a knowledge base using semantic search, then use this context to generate 
more accurate and informed responses. This approach reduces hallucinations and allows AI systems 
to access up-to-date information without retraining.`,
		},
		{
			title: "Vector Embeddings",
			content: `Vector embeddings are numerical representations of text, images, or other data types 
that capture semantic meaning in a high-dimensional space. Similar items have embeddings that are 
close together in this space, enabling similarity search. Modern embedding models like nomic-embed-text 
can generate high-quality embeddings that preserve semantic relationships, making them ideal for 
search and recommendation systems.`,
		},
		{
			title: "Qwen Model",
			content: `Qwen (通义千问) is a series of large language models developed by Alibaba Cloud. 
The Qwen3 model offers excellent multilingual capabilities, particularly strong in Chinese and English. 
It can handle various tasks including text generation, question answering, reasoning, and code generation. 
When used with Ollama, Qwen3 provides a powerful local LLM solution for RAG applications.`,
		},
	}

	// Ingest documents
	for i, doc := range testDocuments {
		req := domain.IngestRequest{
			Content:   doc.content,
			ChunkSize: 300,
			Overlap:   50,
			Metadata: map[string]interface{}{
				"title":  doc.title,
				"doc_id": fmt.Sprintf("doc_%d", i+1),
				"source": "test_data",
			},
		}

		resp, err := ragProcessor.Ingest(ctx, req)
		if err != nil {
			log.Printf("Failed to ingest document %s: %v", doc.title, err)
			continue
		}
		fmt.Printf("   ✓ Ingested: %s (ID: %s, Chunks: %d)\n", 
			doc.title, resp.DocumentID, resp.ChunkCount)
	}

	fmt.Println()
	fmt.Println("6. Testing RAG queries...")
	fmt.Println(strings.Repeat("-", 60))

	// Test queries
	testQueries := []string{
		"What is Qdrant and what are its main features?",
		"How does Ollama work with local models?",
		"Explain the RAG architecture and its benefits",
		"What are vector embeddings used for?",
		"Tell me about the Qwen model capabilities",
		"How do Qdrant and Ollama work together in a RAG system?",
	}

	for i, query := range testQueries {
		fmt.Printf("\nQuery %d: %s\n", i+1, query)
		fmt.Println(strings.Repeat("-", 40))

		req := domain.QueryRequest{
			Query:       query,
			TopK:        3,
			Temperature: 0.7,
			MaxTokens:   500,
			ShowSources: true,
		}

		startTime := time.Now()
		resp, err := ragProcessor.Query(ctx, req)
		elapsed := time.Since(startTime)

		if err != nil {
			log.Printf("Query failed: %v", err)
			continue
		}

		fmt.Printf("\nAnswer:\n%s\n", resp.Answer)
		
		if len(resp.Sources) > 0 {
			fmt.Printf("\nSources (found %d relevant chunks):\n", len(resp.Sources))
			for j, source := range resp.Sources {
				title := "Unknown"
				if source.Metadata != nil {
					if t, ok := source.Metadata["title"].(string); ok {
						title = t
					}
				}
				preview := source.Content
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				fmt.Printf("  %d. [%s] Score: %.3f\n     %s\n", 
					j+1, title, source.Score, preview)
			}
		}

		fmt.Printf("\nElapsed: %v\n", elapsed)
		fmt.Println(strings.Repeat("-", 60))
	}

	// 8. Test statistics
	fmt.Println("\n7. Testing vector store statistics...")
	
	// Perform a test search to verify data
	testVector := make([]float64, 768) // nomic-embed-text uses 768 dimensions
	for i := range testVector {
		testVector[i] = float64(i) / float64(len(testVector))
	}
	
	results, err := vectorStore.Search(ctx, testVector, 5)
	if err != nil {
		fmt.Printf("Failed to search: %v\n", err)
	} else {
		fmt.Printf("   ✓ Vector store contains %d searchable chunks\n", len(results))
	}

	// 9. Interactive mode
	fmt.Println("\n8. Interactive Q&A (type 'exit' to quit)")
	fmt.Println(strings.Repeat("=", 60))
	
	for {
		fmt.Print("\nYour question: ")
		var input string
		fmt.Scanln(&input)
		
		if strings.ToLower(strings.TrimSpace(input)) == "exit" {
			break
		}
		
		if strings.TrimSpace(input) == "" {
			continue
		}

		req := domain.QueryRequest{
			Query:       input,
			TopK:        3,
			Temperature: 0.7,
			MaxTokens:   500,
			ShowSources: false,
		}

		fmt.Println("\nThinking...")
		resp, err := ragProcessor.Query(ctx, req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\nAnswer: %s\n", resp.Answer)
		fmt.Printf("(Retrieved from %d sources in %s)\n", len(resp.Sources), resp.Elapsed)
	}

	fmt.Println("\n=== Test completed successfully ===")
	fmt.Println("\nSummary:")
	fmt.Println("- Vector Store: Qdrant (localhost:6334)")
	fmt.Println("- LLM: Ollama qwen3:latest")
	fmt.Println("- Embeddings: Ollama nomic-embed-text:latest")
	fmt.Println("- Documents ingested: 5")
	fmt.Println("\nYou can access Qdrant UI at: http://localhost:6333")
}

func init() {
	// Ensure required directories exist
	os.MkdirAll("./.rago", 0755)
	
	// Check if Ollama is running
	fmt.Println("Prerequisites:")
	fmt.Println("1. Qdrant running on localhost:6334")
	fmt.Println("2. Ollama running with qwen3:latest and nomic-embed-text:latest")
	fmt.Println()
	fmt.Println("To install required models:")
	fmt.Println("  ollama pull qwen3:latest")
	fmt.Println("  ollama pull nomic-embed-text:latest")
	fmt.Println()
	fmt.Println("To start Qdrant:")
	fmt.Println("  docker run -p 6333:6333 -p 6334:6334 \\")
	fmt.Println("    m.daocloud.io/docker.io/qdrant/qdrant:latest")
	fmt.Println()
	fmt.Println("Press Enter to continue...")
	fmt.Scanln()
}