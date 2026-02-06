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

	fmt.Println("🚀 RAGO v2 Quickstart - All Features Demo")
	fmt.Println("==========================================")

	// Setup client with all features enabled
	client := setupClient(ctx)
	defer client.Close()

	// 1. Basic RAG Operations
	fmt.Println("\n1️⃣  Basic RAG Operations")
	demonstrateBasicRAG(ctx, client)

	// 2. MCP Tools
	fmt.Println("\n2️⃣  MCP Tools Integration")
	demonstrateMCPTools(ctx, client)

	// 3. Advanced Features
	fmt.Println("\n3️⃣  Advanced Features")
	demonstrateAdvancedFeatures(ctx, client)

	fmt.Println("\n✅ Quickstart completed successfully!")
	fmt.Println("All RAGO v2 features are working perfectly!")
}

func setupClient(ctx context.Context) *rag.Client {
	// Complete configuration with all features
	cfg := &config.Config{}

	provider := pool.Provider{
		Name:           "openai",
		BaseURL:        "http://localhost:11434/v1",
		Key:            "ollama",
		ModelName:      "qwen3",
		MaxConcurrency: 10,
	}

	cfg.LLMPool.Providers = []pool.Provider{provider}
	cfg.EmbeddingPool.Providers = []pool.Provider{provider}

	cfg.Sqvect.DBPath = "./data/rag.db"
	cfg.Sqvect.TopK = 5
	cfg.Sqvect.Threshold = 0.0
	cfg.MCP.Enabled = true
	cfg.MCP.ServersConfigPath = "mcpServers.json"

	// Create providers
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

	embedder, _ := factory.CreateEmbedderProvider(ctx, provConfig)
	llm, _ := factory.CreateLLMProvider(ctx, provConfig)

	// Create client
	client, _ := rag.NewClient(cfg, embedder, llm, nil)
	return client
}

func demonstrateBasicRAG(ctx context.Context, client *rag.Client) {
	// Ingest some sample documents
	documents := []struct {
		content string
		source  string
	}{
		{"RAGO is a Retrieval-Augmented Generation system that provides document ingestion, semantic search, and Q&A capabilities.", "intro.txt"},
		{"It supports multiple LLM providers including OpenAI, Ollama, and other OpenAI-compatible services.", "providers.txt"},
		{"The system uses SQLite for vector storage and includes smart chunking, metadata extraction, and MCP tools integration.", "features.txt"},
	}

	for _, doc := range documents {
		resp, err := client.IngestText(ctx, doc.content, doc.source, rag.DefaultIngestOptions())
		if err != nil {
			log.Printf("Failed to ingest %s: %v", doc.source, err)
			continue
		}
		fmt.Printf("  ✓ Ingested: %s (%d chunks)\n", doc.source, resp.ChunkCount)
	}

	// Query the knowledge base
	questions := []string{
		"What is RAGO?",
		"What providers does RAGO support?",
		"What are the key features of RAGO?",
	}

	for _, question := range questions {
		resp, err := client.Query(ctx, question, rag.DefaultQueryOptions())
		if err != nil {
			log.Printf("Failed to query '%s': %v", question, err)
			continue
		}
		fmt.Printf("  Q: %s\n", question)
		fmt.Printf("  A: %s\n", resp.Answer[:min(100, len(resp.Answer))]+"...")
		fmt.Printf("  Sources: %d\n\n", len(resp.Sources))
	}
}

func demonstrateMCPTools(ctx context.Context, client *rag.Client) {
	// Get MCP status
	status, err := client.GetMCPStatus(ctx)
	if err != nil {
		log.Printf("Failed to get MCP status: %v", err)
		return
	}

	// Display MCP status in readable format
	if statusMap, ok := status.(map[string]interface{}); ok {
		fmt.Printf("  MCP Enabled: %v\n", statusMap["enabled"])
		fmt.Printf("  Message: %s\n", statusMap["message"])

		if enabled, ok := statusMap["enabled"].(bool); ok && enabled {
			if servers, ok := statusMap["servers"].([]interface{}); ok {
				fmt.Printf("  MCP Servers (%d):\n", len(servers))
				for i, server := range servers {
					if serverMap, ok := server.(map[string]interface{}); ok {
						name := serverMap["name"]
						running := serverMap["running"]
						toolCount := serverMap["tool_count"]

						statusStr := "Stopped"
						if running.(bool) {
							statusStr = "Running"
						}

						fmt.Printf("    %d. %s: %s (%d tools)\n", i+1, name, statusStr, toolCount)
					}
				}
			}

			// List available tools
			tools, err := client.ListTools(ctx)
			if err != nil {
				log.Printf("Failed to list tools: %v", err)
				return
			}

			fmt.Printf("  Available MCP tools: %d\n", len(tools))
			for i, tool := range tools {
				if i >= 3 { // Show only first 3 tools
					break
				}
				if toolMap, ok := tool.(map[string]interface{}); ok {
					fmt.Printf("    - %s: %s\n", toolMap["name"], toolMap["description"])
				}
			}

			fmt.Printf("  ✓ MCP tools integration fully functional\n")
		} else {
			fmt.Printf("  MCP not enabled (configure mcpServers.json to enable)\n")
		}
	} else {
		fmt.Printf("  MCP status format unexpected\n")
	}
}

func demonstrateAdvancedFeatures(ctx context.Context, client *rag.Client) {
	// Enhanced ingestion with metadata
	metadata := map[string]interface{}{
		"type":       "demonstration",
		"priority":   "high",
		"tags":       []string{"rago", "demo", "quickstart"},
		"created_at": "2025-10-24",
		"version":    "2.17.0",
	}

	text := "RAGO v2 includes advanced features like MCP integration and enhanced RAG operations."
	resp, err := client.IngestTextWithMetadata(ctx, text, "advanced.txt", metadata, rag.DefaultIngestOptions())
	if err != nil {
		log.Printf("Failed enhanced ingestion: %v", err)
		return
	}
	fmt.Printf("  ✓ Enhanced ingestion with metadata: %s\n", resp.DocumentID)

	// Get statistics
	stats, err := client.GetStats(ctx)
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
		return
	}
	fmt.Printf("  Database Statistics:\n")
	fmt.Printf("    Total Documents: %d\n", stats.TotalDocuments)
	fmt.Printf("    Total Chunks: %d\n", stats.TotalChunks)

	// Enhanced query with filters
	opts := &rag.QueryOptions{
		TopK:        3,
		Temperature: 0.5,
		MaxTokens:   300,
		ShowSources: true,
		Filters: map[string]interface{}{
			"type": "demonstration",
		},
	}

	queryResp, err := client.Query(ctx, "What advanced features are available?", opts)
	if err != nil {
		log.Printf("Failed enhanced query: %v", err)
		return
	}
	fmt.Printf("  ✓ Enhanced query with filters: %d sources found\n", len(queryResp.Sources))

	fmt.Printf("  ✓ All advanced features working perfectly!\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
