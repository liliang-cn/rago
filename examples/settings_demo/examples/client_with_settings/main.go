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
	fmt.Println("RAGO Client with Settings Demo")
	fmt.Println("==============================")

	// Create a temporary directory for this demo
	tempDir, err := os.MkdirTemp("", "rago-client-settings-demo")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a basic config
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

	fmt.Printf("Using temp directory: %s\n\n", tempDir)

	// Initialize RAGO client with settings
	fmt.Println("1. Initializing RAGO Client with Settings")
	fmt.Println("-----------------------------------------")

	ragoClient, err := client.NewWithConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create RAGO client: %v", err)
	}
	defer ragoClient.Close()

	fmt.Println("✓ RAGO client initialized with settings support")

	// Show current profile
	fmt.Println("\n2. Current Profile Information")
	fmt.Println("------------------------------")

	activeProfile, err := ragoClient.Settings.GetActiveProfile()
	if err != nil {
		log.Fatalf("Failed to get active profile: %v", err)
	}

	fmt.Printf("Active Profile: %s (%s)\n", activeProfile.Name, activeProfile.ID[:8])
	fmt.Printf("Description: %s\n", activeProfile.Description)

	// Create a new profile for this demo
	fmt.Println("\n3. Creating Custom Profile")
	fmt.Println("--------------------------")

	devProfile, err := ragoClient.Settings.CreateProfile(
		"ai-developer",
		"Profile for AI development assistance",
		"You are an expert AI developer. Provide clear, practical code examples and explanations.",
	)
	if err != nil {
		log.Fatalf("Failed to create profile: %v", err)
	}

	fmt.Printf("✓ Created profile: %s (%s)\n", devProfile.Name, devProfile.ID[:8])

	// Switch to the new profile
	fmt.Println("\n4. Switching Profile")
	fmt.Println("--------------------")

	err = ragoClient.Settings.SwitchProfile(devProfile.ID)
	if err != nil {
		log.Fatalf("Failed to switch profile: %v", err)
	}

	fmt.Printf("✓ Switched to profile: %s\n", devProfile.Name)

	// Set LLM settings for the profile
	fmt.Println("\n5. Configuring LLM Settings")
	fmt.Println("----------------------------")

	temperature := 0.8
	maxTokens := 1500

	llmSettings, err := ragoClient.Settings.SetLLMSettings(
		"ollama",
		"You are a helpful AI programming assistant. Always provide working code with detailed explanations.",
		&temperature,
		&maxTokens,
	)
	if err != nil {
		log.Fatalf("Failed to set LLM settings: %v", err)
	}

	fmt.Printf("✓ Set LLM settings for ollama provider\n")
	fmt.Printf("  System Prompt: %s\n", llmSettings.SystemPrompt)
	fmt.Printf("  Temperature: %.2f\n", *llmSettings.Temperature)
	fmt.Printf("  Max Tokens: %d\n", *llmSettings.MaxTokens)

	// Test generation with profile settings
	fmt.Println("\n6. Testing Generation with Profile Settings")
	fmt.Println("-------------------------------------------")

	ctx := context.Background()
	prompt := "Write a simple Go function that calculates the factorial of a number"

	// Generate using profile settings
	response, err := ragoClient.GenerateWithProfile(ctx, prompt, &client.GenerateOptions{
		Temperature: 0.7, // This will be overridden by profile settings
		MaxTokens:   1000, // This will be overridden by profile settings
	})
	if err != nil {
		log.Fatalf("Failed to generate with profile: %v", err)
	}

	fmt.Printf("Generated response with profile settings:\n%s\n", response)

	// Test chat with profile settings
	fmt.Println("\n7. Testing Chat with Profile Settings")
	fmt.Println("-------------------------------------")

	chatMessages := []client.ChatMessage{
		{Role: "user", Content: "Explain how Go interfaces work with a simple example"},
	}

	chatResponse, err := ragoClient.ChatWithProfile(ctx, chatMessages, nil)
	if err != nil {
		log.Fatalf("Failed to chat with profile: %v", err)
	}

	fmt.Printf("Chat response with profile settings:\n%s\n", chatResponse)

	// Show all profile settings
	fmt.Println("\n8. Profile Settings Summary")
	fmt.Println("---------------------------")

	profile, allSettings, err := ragoClient.Settings.GetProfileWithSettings()
	if err != nil {
		log.Fatalf("Failed to get profile with settings: %v", err)
	}

	fmt.Printf("Profile: %s\n", profile.Name)
	fmt.Printf("Default System Prompt: %s\n", profile.DefaultSystemPrompt)
	fmt.Println("\nLLM Settings:")
	for _, setting := range allSettings {
		fmt.Printf("  Provider: %s\n", setting.ProviderName)
		if setting.SystemPrompt != "" {
			fmt.Printf("    System Prompt: %s\n", setting.SystemPrompt)
		}
		if setting.Temperature != nil {
			fmt.Printf("    Temperature: %.2f\n", *setting.Temperature)
		}
		if setting.MaxTokens != nil {
			fmt.Printf("    Max Tokens: %d\n", *setting.MaxTokens)
		}
	}

	// Test conversation context
	fmt.Println("\n9. Conversation Context Management")
	fmt.Println("----------------------------------")

	contextData := map[string]interface{}{
		"conversation_type": "technical_discussion",
		"programming_language": "go",
		"topic": "interfaces_and_methods",
		"user_level": "intermediate",
	}

	savedContext, err := ragoClient.Settings.SaveConversationContext(contextData)
	if err != nil {
		log.Fatalf("Failed to save conversation context: %v", err)
	}

	fmt.Printf("✓ Saved conversation context (ID: %s)\n", savedContext.ID[:8])

	// Retrieve context
	retrievedContext, err := ragoClient.Settings.GetConversationContext()
	if err != nil {
		log.Fatalf("Failed to get conversation context: %v", err)
	}

	fmt.Printf("✓ Retrieved conversation context:\n")
	for key, value := range retrievedContext.ContextData {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// List all profiles
	fmt.Println("\n10. All Profiles")
	fmt.Println("----------------")

	allProfiles, err := ragoClient.Settings.ListProfiles()
	if err != nil {
		log.Fatalf("Failed to list profiles: %v", err)
	}

	for _, p := range allProfiles {
		status := ""
		if p.IsActive {
			status = " (ACTIVE)"
		}
		fmt.Printf("  - %s: %s%s\n", p.Name, p.Description, status)
	}

	// Direct provider access with settings
	fmt.Println("\n11. Direct Provider Access with Settings")
	fmt.Println("----------------------------------------")

	ollamaProvider, err := ragoClient.GetProviderForProfile("ollama")
	if err != nil {
		log.Fatalf("Failed to get provider for profile: %v", err)
	}

	directResponse, err := ollamaProvider.Generate(ctx, "What is dependency injection?", &domain.GenerationOptions{
		Temperature: 0.5,
		MaxTokens:   500,
	})
	if err != nil {
		log.Fatalf("Failed to generate with direct provider: %v", err)
	}

	fmt.Printf("Direct provider response:\n%s\n", directResponse)

	// Show system prompt
	fmt.Println("\n12. System Prompt Verification")
	fmt.Println("-------------------------------")

	systemPrompt, err := ragoClient.GetSystemPromptForProvider("ollama")
	if err != nil {
		log.Fatalf("Failed to get system prompt: %v", err)
	}

	fmt.Printf("Current system prompt for ollama: %s\n", systemPrompt)

	fmt.Println("\n✅ Demo completed successfully!")
	fmt.Println("The client now automatically applies profile settings to all LLM operations:")
	fmt.Println("  - Profile-specific system prompts")
	fmt.Println("  - Temperature and token limits")
	fmt.Println("  - Conversation context persistence")
	fmt.Println("  - Cross-session settings persistence")
	fmt.Printf("\nNote: Demo data stored in: %s\n", tempDir)
}