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

// This example demonstrates the "Industrial-Grade" capabilities of the RAGO library:
// 1. Concurrent Batch Ingestion
// 2. GraphRAG (Automatic Entity Extraction)
// 3. Stateful Chat Memory with Semantic Recall
// 4. Advanced Search (Reranking & Diversity)

func main() {
	// 1. Setup Context with Timeout for safety
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 2. Configuration: Minimal but Powerful
	// We configure it to use a local DB and OpenAI-compatible provider (e.g., Ollama)
	cfg := &config.Config{
		Sqvect: config.SqvectConfig{
			DBPath:    "./industrial_demo.db",
			IndexType: "hnsw", // Use Hierarchical Navigable Small World index for speed
			TopK:      10,
		},
		Chunker: config.ChunkerConfig{
			ChunkSize: 500,
			Overlap:   50,
			Method:    "sentence",
		},
		Ingest: config.IngestConfig{
			MetadataExtraction: config.MetadataExtractionConfig{
				Enable:   true, // Enable GraphRAG entity extraction
				LLMModel: "qwen3",
			},
		},
		Providers: config.ProvidersConfig{
			DefaultLLM: "openai",
			ProviderConfigs: domain.ProviderConfig{
				OpenAI: &domain.OpenAIProviderConfig{
					BaseProviderConfig: domain.BaseProviderConfig{
						Type:    domain.ProviderOpenAI,
						Timeout: 60 * time.Second,
					},
					// Adjust these for your environment (e.g., real OpenAI key or local Ollama)
					BaseURL:        getEnvOrDefault("RAGO_OPENAI_BASE_URL", "http://localhost:11434/v1"),
					APIKey:         getEnvOrDefault("RAGO_OPENAI_API_KEY", "ollama"),
					LLMModel:       getEnvOrDefault("RAGO_OPENAI_LLM_MODEL", "qwen3"),
					EmbeddingModel: getEnvOrDefault("RAGO_OPENAI_EMBEDDING_MODEL", "nomic-embed-text"),
				},
			},
		},
	}

	// 3. Initialize Providers
	factory := providers.NewFactory()
	llm, err := factory.CreateLLMProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}
	embedder, err := factory.CreateEmbedderProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)
	if err != nil {
		log.Fatalf("Failed to create Embedder: %v", err)
	}

	// 4. Create the Industrial RAG Client
	// Note: We pass 'llm' as the MetadataExtractor for GraphRAG
	client, err := rag.NewClient(cfg, embedder, llm, llm.(domain.MetadataExtractor))
	if err != nil {
		log.Fatalf("Failed to create RAG client: %v", err)
	}
	defer client.Close()

	fmt.Println("🚀 RAGO Client Initialized")

	// =========================================================================
	// Feature 1: High-Throughput Batch Ingestion with GraphRAG
	// =========================================================================
	fmt.Println("\n📚 Ingesting Documents (Batch + Graph Extraction)...")
	
	// Simulate multiple documents
	docs := []domain.IngestRequest{
		{
			Content:   "Project Alpha is led by Alice. It focuses on quantum computing research.",
			ChunkSize: 500,
			Metadata:  map[string]interface{}{"source": "internal_memo_1.txt", "category": "research"},
		},
		{
			Content:   "Bob manages the budget for Project Alpha. The deadline is Q4 2025.",
			ChunkSize: 500,
			Metadata:  map[string]interface{}{"source": "email_archive.txt", "category": "finance"},
		},
		{
			Content:   "Charlie is the lead engineer for the new Quantum Chip prototype.",
			ChunkSize: 500,
			Metadata:  map[string]interface{}{"source": "hr_records.txt", "category": "personnel"},
		},
	}

	// This runs concurrently with semaphore control to prevent overloading the LLM
	ingestResponses, err := client.IngestBatch(ctx, docs)
	if err != nil {
		log.Printf("Batch ingestion warning: %v", err)
	}
	fmt.Printf("✅ Processed %d documents.\n", len(ingestResponses))

	// =========================================================================
	// Feature 2: Advanced Search (Reranking & Diversity)
	// =========================================================================
	fmt.Println("\n🔍 Performing Advanced Search (Hybrid + Rerank + Diversity)...")

	query := "Who is working on Project Alpha?"
	
	// Option A: Keyword Reranking (Boost results containing specific keywords)
	rerankOpts := &rag.QueryOptions{
		TopK:           5,
		RerankStrategy: "keyword", // or "rrf"
		RerankBoost:    2.0,       // Boost factor
	}
	resp, _ := client.Query(ctx, query, rerankOpts)
	fmt.Printf("🔎 Keyword Rerank Answer: %s\n", resp.Answer)

	// Option B: Diversity Search (MMR) to avoid repetitive results
	divOpts := &rag.QueryOptions{
		TopK:            5,
		DiversityLambda: 0.7, // 0.7 = balance between relevance and diversity
	}
	respDiv, _ := client.Query(ctx, query, divOpts)
	fmt.Printf("🌈 Diversity Search Answer: %s\n", respDiv.Answer)

	// =========================================================================
	// Feature 3: Stateful Chat with Semantic Memory
	// =========================================================================
	fmt.Println("\n💬 Starting Stateful Chat Session...")

	// Create a persistent session
	session, err := client.StartChat(ctx, "user_007", map[string]interface{}{
		"department": "security",
	})
	if err != nil {
		log.Fatalf("Failed to start chat: %v", err)
	}
	fmt.Printf("🔹 Session ID: %s\n", session.ID)

	// Turn 1: Ask a question
	// RAGO automatically embeds this query and stores it in history
	fmt.Println("👤 User: Who manages the budget?")
	chatResp1, _ := client.Chat(ctx, session.ID, "Who manages the budget?", rag.DefaultQueryOptions())
	fmt.Printf("🤖 AI: %s\n", chatResp1.Answer)

	// Turn 2: Ask a follow-up (Implicit Context)
	// "When is it due?" refers to the budget/project mentioned in history
	// RAGO retrieves recent history + semantic matches from long-term memory
	fmt.Println("👤 User: When is it due?")
	chatResp2, _ := client.Chat(ctx, session.ID, "When is it due?", rag.DefaultQueryOptions())
	fmt.Printf("🤖 AI: %s\n", chatResp2.Answer)

	// Cleanup
	fmt.Println("\n🧹 Cleaning up...")
	client.Reset(ctx)
	os.Remove("./industrial_demo.db")
	fmt.Println("✨ Done.")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != ""
	{
		return value
	}
	return defaultValue
}
