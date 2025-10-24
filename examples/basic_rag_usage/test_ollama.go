package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

func main() {
	ctx := context.Background()

	// 配置使用你的 Ollama 模型
	cfg := &config.Config{
		Sqvect: config.SqvectConfig{
			DBPath:    "./test_rag.db",
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
		Providers: config.ProvidersConfig{
			DefaultLLM:     "openai",
			DefaultEmbedder: "openai",
			ProviderConfigs: domain.ProviderConfig{
				OpenAI: &domain.OpenAIProviderConfig{
					BaseProviderConfig: domain.BaseProviderConfig{
						Type:    domain.ProviderOpenAI,
						Timeout: 30 * 1000000000,
					},
					BaseURL:        "http://localhost:11434/v1",
					APIKey:         "ollama",
					LLMModel:       "qwen3",
					EmbeddingModel: "nomic-embed-text",
				},
			},
		},
	}

	// 创建提供商
	factory := providers.NewFactory()
	
	fmt.Println("🔧 Creating LLM provider...")
	llm, err := factory.CreateLLMProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	fmt.Println("🔧 Creating embedder provider...")
	embedder, err := factory.CreateEmbedderProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	// 测试健康检查
	fmt.Println("🏥 Testing LLM health...")
	if err := llm.Health(ctx); err != nil {
		fmt.Printf("LLM health check failed: %v\n", err)
	} else {
		fmt.Println("✅ LLM is healthy!")
	}

	fmt.Println("🏥 Testing embedder health...")
	if err := embedder.Health(ctx); err != nil {
		fmt.Printf("Embedder health check failed: %v\n", err)
	} else {
		fmt.Println("✅ Embedder is healthy!")
	}

	// 测试嵌入生成
	fmt.Println("🔍 Testing embedding generation...")
	testText := "RAGO is a local RAG system"
	embeddings, err := embedder.Embed(ctx, testText)
	if err != nil {
		fmt.Printf("Embedding failed: %v\n", err)
	} else {
		fmt.Printf("✅ Embedding generated successfully! Size: %d\n", len(embeddings))
	}

	// 测试 LLM 生成
	fmt.Println("💬 Testing LLM generation...")
	response, err := llm.Generate(ctx, "Say hello in one sentence", &domain.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   50,
	})
	if err != nil {
		fmt.Printf("LLM generation failed: %v\n", err)
	} else {
		fmt.Printf("✅ LLM response: %s\n", response)
	}

	fmt.Println("\n🎉 All tests completed!")
}
