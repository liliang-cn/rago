package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/liliang-cn/rago/v2/client"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
)

func main() {
	fmt.Println("RAGO Client Migration Guide")
	fmt.Println("===========================")
	fmt.Println("This example shows how to migrate existing client code to use the new settings system")

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "rago-migration-demo")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config
	cfg := &config.Config{
		Sqvect: config.SqvectConfig{
			DBPath: filepath.Join(tempDir, "rag.db"),
		},
		Providers: config.ProvidersConfig{
			DefaultLLM:      "ollama",
			DefaultEmbedder: "ollama",
			ProviderConfigs: domain.ProviderConfig{
				Ollama: &domain.OllamaProviderConfig{
					BaseURL:        "http://localhost:11434",
					LLMModel:       "qwen3",
					EmbeddingModel: "nomic-embed-text",
				},
			},
		},
	}

	// Initialize client
	ragoClient, err := client.NewWithConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	fmt.Println("\n=== BEFORE: Traditional Client Usage ===")
	
	// OLD WAY: Direct LLM usage without profile settings
	fmt.Println("\n1. Old Way - Direct LLM Generation")
	fmt.Println("-----------------------------------")
	
	oldResponse, err := ragoClient.LLM.GenerateWithOptions(ctx, "Explain Go channels", &client.GenerateOptions{
		Temperature: 0.7,
		MaxTokens:   500,
	})
	if err != nil {
		log.Printf("Old method failed: %v", err)
	} else {
		fmt.Printf("Old method response (first 100 chars): %s...\n", oldResponse[:min(100, len(oldResponse))])
	}

	// OLD WAY: Chat without profile settings
	fmt.Println("\n2. Old Way - Chat Completion")
	fmt.Println("-----------------------------")
	
	oldChatResponse, err := ragoClient.LLM.ChatWithOptions(ctx, []client.ChatMessage{
		{Role: "user", Content: "What are Go slices?"},
	}, &client.GenerateOptions{
		Temperature: 0.8,
		MaxTokens:   400,
	})
	if err != nil {
		log.Printf("Old chat method failed: %v", err)
	} else {
		fmt.Printf("Old chat response (first 100 chars): %s...\n", oldChatResponse[:min(100, len(oldChatResponse))])
	}

	fmt.Println("\n=== AFTER: Profile-Aware Client Usage ===")

	// Set up a profile for better responses
	fmt.Println("\n3. Setting Up Profile")
	fmt.Println("---------------------")
	
	err = ragoClient.Settings.SetSystemPrompt("ollama", "You are a Go programming expert. Provide concise, accurate explanations with code examples when helpful.")
	if err != nil {
		log.Fatalf("Failed to set system prompt: %v", err)
	}
	
	temperature := 0.6
	maxTokens := 600
	_, err = ragoClient.Settings.SetLLMSettings("ollama", "", &temperature, &maxTokens)
	if err != nil {
		log.Fatalf("Failed to set LLM settings: %v", err)
	}
	
	fmt.Println("✓ Profile configured with custom system prompt and parameters")

	// NEW WAY: Profile-aware generation
	fmt.Println("\n4. New Way - Profile-Aware Generation")
	fmt.Println("-------------------------------------")
	
	newResponse, err := ragoClient.GenerateWithProfile(ctx, "Explain Go channels", &client.GenerateOptions{
		Temperature: 0.9, // This will be overridden by profile settings (0.6)
		MaxTokens:   300, // This will be overridden by profile settings (600)
	})
	if err != nil {
		log.Fatalf("New method failed: %v", err)
	}
	
	fmt.Printf("New method response (first 200 chars): %s...\n", newResponse[:min(200, len(newResponse))])

	// NEW WAY: Profile-aware chat
	fmt.Println("\n5. New Way - Profile-Aware Chat")
	fmt.Println("-------------------------------")
	
	newChatResponse, err := ragoClient.ChatWithProfile(ctx, []client.ChatMessage{
		{Role: "user", Content: "What are Go slices?"},
	}, nil) // nil options means use profile defaults
	if err != nil {
		log.Fatalf("New chat method failed: %v", err)
	}
	
	fmt.Printf("New chat response (first 200 chars): %s...\n", newChatResponse[:min(200, len(newChatResponse))])

	fmt.Println("\n=== MIGRATION BENEFITS ===")
	fmt.Println("✅ Backward Compatibility: All existing methods still work")
	fmt.Println("✅ Profile Settings: Automatic application of user preferences")
	fmt.Println("✅ Persistent Configuration: Settings survive across sessions")
	fmt.Println("✅ Easy Migration: Just add .Settings calls for enhanced functionality")
	fmt.Println("✅ Multiple Profiles: Switch contexts easily")

	fmt.Println("\n=== MIGRATION STEPS ===")
	fmt.Println("1. Your existing code continues to work unchanged")
	fmt.Println("2. Add profile setup calls (once per application)")
	fmt.Println("3. Replace generate/chat calls with profile-aware versions")
	fmt.Println("4. Enjoy persistent, personalized LLM interactions")

	// Show migration examples
	fmt.Println("\n=== MIGRATION EXAMPLES ===")
	
	fmt.Println("\nBEFORE:")
	fmt.Println("  response, err := client.LLM.GenerateWithOptions(ctx, prompt, &GenerateOptions{...})")
	
	fmt.Println("\nAFTER:")
	fmt.Println("  // One-time setup")
	fmt.Println("  client.Settings.SetSystemPrompt(\"ollama\", \"Custom prompt...\")")
	fmt.Println("  // Then use profile-aware methods")
	fmt.Println("  response, err := client.GenerateWithProfile(ctx, prompt, options)")

	fmt.Println("\nBEFORE:")
	fmt.Println("  response, err := client.LLM.ChatWithOptions(ctx, messages, &GenerateOptions{...})")
	
	fmt.Println("\nAFTER:")
	fmt.Println("  response, err := client.ChatWithProfile(ctx, messages, options)")

	fmt.Printf("\nDemo data stored in: %s\n", tempDir)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}