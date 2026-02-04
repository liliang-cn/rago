package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/llm"
	"github.com/liliang-cn/rago/v2/pkg/pool"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

func main() {
	ctx := context.Background()

	// 1. Initialize Configuration

	// Create provider config
	provider := pool.Provider{
		Name:           "openai",
		BaseURL:        "http://localhost:11434/v1", // Ollama
		Key:            "ollama",
		ModelName:      "qwen2.5-coder:14b",
		MaxConcurrency: 10,
	}

	fmt.Printf("Provider: %s\n", "openai")
	fmt.Printf("Model: %s\n", provider.ModelName)

	// 2. Initialize Provider
	factory := providers.NewFactory()
	llmProvider, err := factory.CreateLLMProvider(ctx, &domain.OpenAIProviderConfig{
		BaseProviderConfig: domain.BaseProviderConfig{
			Timeout: 30_000_000_000, // 30 seconds in nanoseconds
		},
		BaseURL:        provider.BaseURL,
		APIKey:         provider.Key,
		LLMModel:       provider.ModelName,
		EmbeddingModel: "nomic-embed-text",
	})
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}

	// Create LLM service
	llmService := llm.NewService(llmProvider)

	// Example 1: Simple text generation
	fmt.Println("=== Example 1: Simple Generation ===")
	prompt := "Write a haiku about artificial intelligence"

	response, err := llmService.Generate(ctx, prompt, &domain.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   100,
	})
	if err != nil {
		log.Printf("Failed to generate: %v", err)
	} else {
		fmt.Printf("Prompt: %s\n", prompt)
		fmt.Printf("Response: %s\n", response)
	}

	// Example 2: Streaming generation
	fmt.Println("\n=== Example 2: Streaming Generation ===")
	streamPrompt := "Tell me a short story about a robot learning to paint"

	fmt.Printf("Prompt: %s\n", streamPrompt)
	fmt.Print("Response: ")

	err = llmService.Stream(ctx, streamPrompt, &domain.GenerationOptions{
		Temperature: 0.8,
		MaxTokens:   300,
	}, func(chunk string) {
		fmt.Print(chunk)
	})
	if err != nil {
		log.Printf("Failed to stream: %v", err)
	} else {
		fmt.Printf("\n\nStreaming completed successfully!\n")
	}

	// Example 3: Tool calling (if supported)
	fmt.Println("\n=== Example 3: Tool Calling ===")

	// Define messages for tool calling
	messages := []domain.Message{
		{Role: "user", Content: "What's the current time and weather? Use appropriate tools to find out."},
	}

	// Define available tools (simplified example)
	tools := []domain.ToolDefinition{
		{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "get_current_time",
				Description: "Get the current time",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"format": map[string]interface{}{
							"type":        "string",
							"description": "Time format (e.g., 'RFC3339', 'unix')",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "get_weather",
				Description: "Get current weather for a location",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "City name or location",
						},
					},
					"required": []string{"location"},
				},
			},
		},
	}

	result, err := llmService.GenerateWithTools(ctx, messages, tools, &domain.GenerationOptions{
		Temperature: 0.3,
		MaxTokens:   200,
	})
	if err != nil {
		log.Printf("Tool calling not supported or failed: %v", err)
		fmt.Println("Note: Tool calling may not be supported by all providers")
	} else {
		fmt.Printf("Tool calling response:\n")
		fmt.Printf("Content: %s\n", result.Content)
		fmt.Printf("Tool calls: %d\n", len(result.ToolCalls))

		for i, toolCall := range result.ToolCalls {
			fmt.Printf("  Tool Call %d: %s\n", i+1, toolCall.Function.Name)
			fmt.Printf("    Arguments: %v\n", toolCall.Function.Arguments)
		}
	}

	// Example 4: Structured generation (JSON output)
	fmt.Println("\n=== Example 4: Structured Generation ===")

	jsonPrompt := "Generate a JSON object describing a fictional character with name, age, occupation, and hobbies"

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Character's name",
			},
			"age": map[string]interface{}{
				"type":        "integer",
				"description": "Character's age",
			},
			"occupation": map[string]interface{}{
				"type":        "string",
				"description": "Character's occupation",
			},
			"hobbies": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "Character's hobbies",
			},
		},
		"required": []string{"name", "age", "occupation", "hobbies"},
	}

	structuredResult, err := llmService.GenerateStructured(ctx, jsonPrompt, schema, &domain.GenerationOptions{
		Temperature: 0.7,
		MaxTokens:   200,
	})
	if err != nil {
		log.Printf("Structured generation not supported or failed: %v", err)
		fmt.Println("Note: Structured generation may not be supported by all providers")
	} else {
		fmt.Printf("Structured generation result:\n")
		fmt.Printf("Raw JSON: %s\n", structuredResult.Raw)
		fmt.Printf("Valid: %t\n", structuredResult.Valid)
		fmt.Printf("Data: %+v\n", structuredResult.Data)
	}

	// Example 5: Intent recognition
	fmt.Println("\n=== Example 5: Intent Recognition ===")

	requests := []string{
		"Tell me about quantum computing",
		"Calculate 15% of 250",
		"What's the weather like today?",
		"Help me write a business plan",
	}

	for _, request := range requests {
		intent, err := llmService.RecognizeIntent(ctx, request)
		if err != nil {
			log.Printf("Failed to recognize intent: %v", err)
			continue
		}

		fmt.Printf("Request: %s\n", request)
		fmt.Printf("Intent: %s (confidence: %.2f)\n", intent.Intent, intent.Confidence)
		fmt.Printf("Needs tools: %t\n", intent.NeedsTools)
		if intent.Reasoning != "" {
			fmt.Printf("Reasoning: %s\n", intent.Reasoning)
		}
		fmt.Println()
	}

	// Example 6: Health check
	fmt.Println("=== Example 6: Health Check ===")

	err = llmService.Health(ctx)
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
	} else {
		fmt.Printf("LLM provider is healthy!\n")
	}

	fmt.Printf("Provider type: %s\n", llmService.ProviderType())

	fmt.Println("\n=== LLM Service Examples completed successfully! ===")
}
