package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/liliang-cn/rago/v2/pkg/agent"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/pool"
)

func main() {
	ctx := context.Background()

	// 1. Configure RAGO Agent
	cfg := &config.Config{
		Cortexdb: config.CortexdbConfig{
			DBPath:    "/tmp/rago-example-combined-rag.db",
			TopK:      3,
			Threshold: 0.1,
		},
		Memory: config.MemoryConfig{
			StoreType:  "vector",
			MemoryPath: "/tmp/rago-example-combined-mem.db",
		},
		Chunker: config.ChunkerConfig{
			ChunkSize: 500,
			Overlap:   50,
			Method:    "sentence",
		},
	}

	provider := pool.Provider{
		Name:           "openai",
		BaseURL:        "http://localhost:11434/v1",
		Key:            "ollama",
		ModelName:      "qwen3.5:latest",
		MaxConcurrency: 10,
	}
	embedderProvider := pool.Provider{
		Name:           "openai",
		BaseURL:        "http://localhost:11434/v1",
		Key:            "ollama",
		ModelName:      "qwen3-embedding:8b",
		MaxConcurrency: 10,
	}

	cfg.LLMPool.Providers = []pool.Provider{provider}
	cfg.LLMPool.Enabled = true
	cfg.EmbeddingPool.Providers = []pool.Provider{embedderProvider}
	cfg.EmbeddingPool.Enabled = true

	// 2. Build Agent with Memory and RAG
	fmt.Println("=== Building Agent ===")
	
	agentSvc, err := agent.New("Combined Agent").
		WithDebug(true).
		WithConfig(cfg).
		WithRAG().
		WithMemory().
		Build()

	if err != nil {
		log.Fatalf("Failed to build agent: %v", err)
	}

	// 3. Inject Data into RAG
	fmt.Println("\n=== Ingesting Document into RAG ===")
	ragProcessor := agentSvc.RAG
	docContent := `RAGO Corp 是一家于 2025 年由一群 AI 爱好者创立的科技公司。他们的旗舰产品是一款名为 RAGO 的本地优先 AI 助手框架。该公司的总部位于中国杭州。`
	_, err = ragProcessor.Ingest(ctx, domain.IngestRequest{
		Content:   docContent,
		ChunkSize: 500,
		Overlap:   50,
		Metadata:  map[string]interface{}{"source": "company_info"},
	})
	if err != nil {
		log.Fatalf("Failed to ingest document: %v", err)
	}
	fmt.Println("✓ RAG Document ingested.")

	// 4. Add User Memory
	fmt.Println("\n=== Adding User Memories ===")
	memorySvc := agentSvc.Memory
	mem := &domain.Memory{
		ID:         uuid.New().String(),
		Type:       domain.MemoryTypePreference,
		Content:    "我非常喜欢 RAGO Corp 的产品，并且我很想去他们总部所在的城市旅游。",
		Importance: 0.9,
		SessionID:  "user-test",
		CreatedAt:  time.Now(),
	}
	if err := memorySvc.Add(ctx, mem); err != nil {
		log.Fatalf("Failed to add memory: %v", err)
	}
	fmt.Println("✓ User Memory added.")

	// 5. Query Agent requiring both facts
	fmt.Println("\n=== Querying Agent ===")
	query := "根据我的喜好，我喜欢的公司是哪一年成立的？他们总部在哪个城市？我为什么想去那个城市？"
	
	fmt.Printf("User: %s\n\n", query)
	
	resp, err := agentSvc.Chat(ctx, query)
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}
	
	fmt.Printf("\nAgent Response:\n%s\n", resp.Text())
	
	if len(resp.Sources) > 0 {
		fmt.Printf("\n--- RAG Sources Used ---\n")
		for i, src := range resp.Sources {
			fmt.Printf("%d. %s (Score: %.2f)\n", i+1, src.Content, src.Score)
		}
	}

	if len(resp.Memories) > 0 {
		fmt.Printf("\n--- User Memories Used ---\n")
		for i, mem := range resp.Memories {
			fmt.Printf("%d. [%s] %s (Score: %.2f)\n", i+1, mem.Type, mem.Content, mem.Score)
		}
	}
}
