package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/pool"
	"github.com/liliang-cn/rago/v2/pkg/providers"
	"github.com/liliang-cn/rago/v2/pkg/rag/graphrag"
)

func main() {
	ctx := context.Background()

	// Setup provider
	provider := pool.Provider{
		Name:           "ollama",
		BaseURL:        "http://localhost:11434/v1",
		Key:            "ollama",
		ModelName:      "qwen3.5:latest",
		MaxConcurrency: 10,
	}

	factory := providers.NewFactory()

	provConfig := &domain.OpenAIProviderConfig{
		BaseProviderConfig: domain.BaseProviderConfig{
			Timeout: 60,
		},
		BaseURL:  provider.BaseURL,
		APIKey:   provider.Key,
		LLMModel: provider.ModelName,
	}

	llm, err := factory.CreateLLMProvider(ctx, provConfig)
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}

	// Test: Using extractor
	fmt.Println("=== Test: Using extractor ===")
	extractor := graphrag.NewEntityExtractor(llm, []string{"person", "organization", "location"})

	testText := "Apple Inc. founded by Steve Jobs."

	log.Println("Extracting entities...")
	entities, relations, err := extractor.Extract(ctx, testText)
	if err != nil {
		log.Printf("Extraction error: %v", err)
	} else {
		fmt.Printf("Extracted %d entities:\n", len(entities))
		for _, e := range entities {
			fmt.Printf("  - %s (%s): %s\n", e.Name, e.Type, e.Description)
		}
		fmt.Printf("Extracted %d relations:\n", len(relations))
		for _, r := range relations {
			fmt.Printf("  - %s --[%s]--> %s\n", r.Source, r.Type, r.Target)
		}
	}

	fmt.Println("\n=== Tests completed ===")
}
