// Package main demonstrates how to use Claude and Gemini providers with RAGO
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

func main() {
	ctx := context.Background()

	// Example 1: Claude Provider
	fmt.Println("=== Claude Provider Example ===")
	claudeConfig := &domain.ClaudeProviderConfig{
		APIKey:           "your-claude-api-key-here",
		LLMModel:         "claude-3-sonnet-20240229",
		MaxTokens:        1000,
		AnthropicVersion: "2023-06-01",
	}

	claudeProvider, err := providers.NewClaudeProvider(claudeConfig)
	if err != nil {
		log.Printf("Failed to create Claude provider: %v", err)
	} else {
		fmt.Printf("✅ Claude provider created successfully\n")
		fmt.Printf("   Provider Type: %s\n", claudeProvider.ProviderType())
		fmt.Printf("   Model: %s\n", claudeConfig.LLMModel)
		
		// Example generation (commented out since it requires real API key)
		/*
		response, err := claudeProvider.Generate(ctx, "What is the meaning of life?", &domain.GenerationOptions{
			MaxTokens:   100,
			Temperature: 0.7,
		})
		if err != nil {
			log.Printf("Claude generation failed: %v", err)
		} else {
			fmt.Printf("   Response: %s\n", response)
		}
		*/
	}

	fmt.Println()

	// Example 2: Gemini Provider
	fmt.Println("=== Gemini Provider Example ===")
	geminiConfig := &domain.GeminiProviderConfig{
		APIKey:         "your-gemini-api-key-here",
		LLMModel:       "gemini-pro",
		EmbeddingModel: "embedding-001",
		ProjectID:      "your-project-id",
		Location:       "us-central1",
	}

	geminiProvider, err := providers.NewGeminiProvider(geminiConfig)
	if err != nil {
		log.Printf("Failed to create Gemini provider: %v", err)
	} else {
		fmt.Printf("✅ Gemini provider created successfully\n")
		fmt.Printf("   Provider Type: %s\n", geminiProvider.ProviderType())
		fmt.Printf("   Model: %s\n", geminiConfig.LLMModel)
		
		// Example generation (commented out since it requires real API key)
		/*
		response, err := geminiProvider.Generate(ctx, "Explain quantum computing", &domain.GenerationOptions{
			MaxTokens:   100,
			Temperature: 0.8,
		})
		if err != nil {
			log.Printf("Gemini generation failed: %v", err)
		} else {
			fmt.Printf("   Response: %s\n", response)
		}
		*/
	}

	fmt.Println()

	// Example 3: Using Factory Pattern
	fmt.Println("=== Factory Pattern Example ===")
	factory := providers.NewFactory()

	// Create Claude provider through factory
	claudeFromFactory, err := factory.CreateLLMProvider(ctx, claudeConfig)
	if err != nil {
		log.Printf("Failed to create Claude provider via factory: %v", err)
	} else {
		fmt.Printf("✅ Claude provider created via factory\n")
		fmt.Printf("   Type: %s\n", claudeFromFactory.ProviderType())
	}

	// Create Gemini provider through factory
	geminiFromFactory, err := factory.CreateLLMProvider(ctx, geminiConfig)
	if err != nil {
		log.Printf("Failed to create Gemini provider via factory: %v", err)
	} else {
		fmt.Printf("✅ Gemini provider created via factory\n")
		fmt.Printf("   Type: %s\n", geminiFromFactory.ProviderType())
	}

	fmt.Println()

	// Example 4: Configuration Map (for dynamic configuration)
	fmt.Println("=== Dynamic Configuration Example ===")
	
	claudeConfigMap := map[string]interface{}{
		"type":              "claude",
		"api_key":           "your-claude-api-key",
		"llm_model":         "claude-3-haiku-20240307",
		"max_tokens":        500,
		"anthropic_version": "2023-06-01",
	}

	claudeDynamic, err := factory.CreateLLMProviderFromMap(ctx, claudeConfigMap)
	if err != nil {
		log.Printf("Failed to create Claude provider from map: %v", err)
	} else {
		fmt.Printf("✅ Claude provider created from config map\n")
		fmt.Printf("   Type: %s\n", claudeDynamic.ProviderType())
	}

	geminiConfigMap := map[string]interface{}{
		"type":            "gemini",
		"api_key":         "your-gemini-api-key",
		"llm_model":       "gemini-1.5-pro",
		"embedding_model": "text-embedding-004",
		"project_id":      "my-project",
	}

	geminiDynamic, err := factory.CreateLLMProviderFromMap(ctx, geminiConfigMap)
	if err != nil {
		log.Printf("Failed to create Gemini provider from map: %v", err)
	} else {
		fmt.Printf("✅ Gemini provider created from config map\n")
		fmt.Printf("   Type: %s\n", geminiDynamic.ProviderType())
	}

	fmt.Println()

	// Example 5: Tool Calling (Advanced Usage)
	fmt.Println("=== Tool Calling Example ===")
	
	// Define a simple tool
	tools := []domain.ToolDefinition{
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
							"description": "City name",
						},
					},
					"required": []string{"location"},
				},
			},
		},
	}

	messages := []domain.Message{
		{
			Role:    "user",
			Content: "What's the weather like in San Francisco?",
		},
	}

	if claudeProvider != nil {
		fmt.Printf("Claude tool calling example (would need real API key to execute)\n")
		fmt.Printf("   Tools defined: %d\n", len(tools))
		fmt.Printf("   Messages: %d\n", len(messages))
		
		// Example tool calling (commented out since it requires real API key)
		/*
		result, err := claudeProvider.GenerateWithTools(ctx, messages, tools, &domain.GenerationOptions{
			MaxTokens: 200,
		})
		if err != nil {
			log.Printf("Claude tool calling failed: %v", err)
		} else {
			fmt.Printf("   Content: %s\n", result.Content)
			fmt.Printf("   Tool Calls: %d\n", len(result.ToolCalls))
		}
		*/
	}

	fmt.Println("\n🎉 All provider examples completed!")
	fmt.Println("\nTo use these providers with real API keys:")
	fmt.Println("1. Claude: Get API key from https://console.anthropic.com/")
	fmt.Println("2. Gemini: Get API key from https://makersuite.google.com/")
	fmt.Println("3. Update your rago.toml configuration file with the new providers")
	fmt.Println("4. Use the providers in your RAG queries and agent workflows")
}