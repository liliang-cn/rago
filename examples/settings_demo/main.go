package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/settings"
)

func main() {
	fmt.Println("RAGO Settings Demo")
	fmt.Println("==================")

	// Create a temporary directory for this demo
	tempDir, err := os.MkdirTemp("", "rago-settings-demo")
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

	// Initialize settings service
	fmt.Println("1. Initializing Settings Service")
	fmt.Println("---------------------------------")

	configWithSettings, err := settings.NewConfigWithSettings(cfg)
	if err != nil {
		log.Fatalf("Failed to create config with settings: %v", err)
	}
	defer configWithSettings.Close()

	// List profiles (should show default profile)
	fmt.Println("Initial profiles:")
	profiles, err := configWithSettings.SettingsService.ListProfiles()
	if err != nil {
		log.Fatalf("Failed to list profiles: %v", err)
	}

	for _, profile := range profiles {
		status := ""
		if profile.IsActive {
			status = " (ACTIVE)"
		}
		fmt.Printf("  - %s: %s%s\n", profile.Name, profile.Description, status)
	}

	// Create a new profile
	fmt.Println("\n2. Creating Custom Profile")
	fmt.Println("--------------------------")

	createReq := settings.CreateProfileRequest{
		Name:                "developer",
		Description:         "Profile for software development tasks",
		DefaultSystemPrompt: "You are an expert software engineer. Provide concise, accurate, and practical solutions.",
		Metadata: map[string]string{
			"role":    "developer",
			"created": "demo",
		},
	}

	devProfile, err := configWithSettings.SettingsService.CreateProfile(createReq)
	if err != nil {
		log.Fatalf("Failed to create profile: %v", err)
	}

	fmt.Printf("Created profile: %s (%s)\n", devProfile.Name, devProfile.ID[:8])

	// Set LLM settings for the profile
	fmt.Println("\n3. Configuring LLM Settings")
	fmt.Println("----------------------------")

	llmSettingsReq := settings.CreateLLMSettingsRequest{
		ProfileID:    devProfile.ID,
		ProviderName: "ollama",
		SystemPrompt: "You are a helpful coding assistant. Always provide working code examples with explanations.",
		Temperature:  func() *float64 { t := 0.7; return &t }(),
		MaxTokens:    func() *int { m := 2048; return &m }(),
		Settings: map[string]interface{}{
			"context_length": 4096,
			"num_predict":    -1,
		},
	}

	_, err = configWithSettings.SettingsService.CreateOrUpdateLLMSettings(llmSettingsReq)
	if err != nil {
		log.Fatalf("Failed to create LLM settings: %v", err)
	}

	fmt.Printf("✓ Set LLM settings for %s provider\n", llmSettingsReq.ProviderName)

	// Switch to the new profile
	fmt.Println("\n4. Switching Active Profile")
	fmt.Println("----------------------------")

	err = configWithSettings.SettingsService.SetActiveProfile(devProfile.ID)
	if err != nil {
		log.Fatalf("Failed to switch profile: %v", err)
	}

	fmt.Printf("✓ Switched to profile: %s\n", devProfile.Name)

	// Show profile with settings
	fmt.Println("\n5. Current Profile Configuration")
	fmt.Println("--------------------------------")

	currentProfile, allSettings, err := configWithSettings.GetActiveProfileSettings()
	if err != nil {
		log.Fatalf("Failed to get active profile settings: %v", err)
	}

	fmt.Printf("Active Profile: %s\n", currentProfile.Name)
	fmt.Printf("Description: %s\n", currentProfile.Description)
	fmt.Printf("Default System Prompt: %s\n", currentProfile.DefaultSystemPrompt)

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
		if len(setting.Settings) > 0 {
			fmt.Printf("    Additional Settings: %v\n", setting.Settings)
		}
		fmt.Println()
	}

	// Demonstrate provider integration
	fmt.Println("6. Testing Provider Integration")
	fmt.Println("-------------------------------")

	// Get system prompt for the provider
	systemPrompt, err := configWithSettings.GetSystemPromptForProvider("ollama")
	if err != nil {
		log.Fatalf("Failed to get system prompt: %v", err)
	}

	fmt.Printf("System prompt for ollama provider: %s\n", systemPrompt)

	// Demonstrate conversation context
	fmt.Println("\n7. Conversation Context Management")
	fmt.Println("----------------------------------")

	contextData := map[string]interface{}{
		"conversation_id": "demo-conversation-001",
		"topic":          "golang development",
		"last_exchange":  "Discussing RAGO settings architecture",
		"user_intent":    "learning system implementation",
	}

	savedContext, err := configWithSettings.SettingsService.SaveConversationContext(contextData)
	if err != nil {
		log.Fatalf("Failed to save conversation context: %v", err)
	}

	fmt.Printf("✓ Saved conversation context (ID: %s)\n", savedContext.ID[:8])

	// Retrieve context
	retrievedContext, err := configWithSettings.SettingsService.GetConversationContext()
	if err != nil {
		log.Fatalf("Failed to get conversation context: %v", err)
	}

	fmt.Printf("✓ Retrieved conversation context:\n")
	for key, value := range retrievedContext.ContextData {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// Create another profile to demonstrate persistence
	fmt.Println("\n8. Profile Persistence Test")
	fmt.Println("---------------------------")

	createReq2 := settings.CreateProfileRequest{
		Name:        "qa-tester",
		Description: "Profile for QA and testing tasks",
		Metadata: map[string]string{
			"role": "qa",
		},
	}

	qaProfile, err := configWithSettings.SettingsService.CreateProfile(createReq2)
	if err != nil {
		log.Fatalf("Failed to create QA profile: %v", err)
	}

	fmt.Printf("✓ Created second profile: %s\n", qaProfile.Name)

	// List all profiles
	fmt.Println("\nAll profiles:")
	profiles, err = configWithSettings.SettingsService.ListProfiles()
	if err != nil {
		log.Fatalf("Failed to list profiles: %v", err)
	}

	for _, profile := range profiles {
		status := ""
		if profile.IsActive {
			status = " (ACTIVE)"
		}
		fmt.Printf("  - %s: %s%s\n", profile.Name, profile.Description, status)
	}

	fmt.Println("\n✅ Demo completed successfully!")
	fmt.Println("The settings system is working correctly with:")
	fmt.Println("  - Profile management")
	fmt.Println("  - LLM settings persistence")
	fmt.Println("  - Provider integration")
	fmt.Println("  - Conversation context storage")
	fmt.Printf("\nNote: Demo data stored in: %s\n", tempDir)
}