package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/rag"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== RAGO v2 Advanced Features Test ===")

	// Setup client (same as basic example)
	client := setupClient(ctx)
	defer client.Close()

	// Test 1: Profile Management
	fmt.Println("\n--- Test 1: Profile Management ---")
	testProfileManagement(ctx, client)

	// Test 2: LLM Settings Management
	fmt.Println("\n--- Test 2: LLM Settings Management ---")
	testLLMSettings(ctx, client)

	// Test 3: MCP Integration
	fmt.Println("\n--- Test 3: MCP Integration ---")
	testMCPIntegration(ctx, client)

	// Test 4: Enhanced RAG Operations
	fmt.Println("\n--- Test 4: Enhanced RAG Operations ---")
	testEnhancedRAG(ctx, client)

	fmt.Println("\n=== Advanced Features Test Completed ===")
}

func setupClient(ctx context.Context) *rag.Client {
	// Configuration setup
	cfg := &config.Config{}
	cfg.Providers.DefaultLLM = "openai"
	cfg.Providers.ProviderConfigs = domain.ProviderConfig{
		OpenAI: &domain.OpenAIProviderConfig{
			BaseURL:        "http://localhost:11434/v1",
			APIKey:         "ollama",
			LLMModel:       "qwen3",
			EmbeddingModel: "nomic-embed-text",
		},
	}
	cfg.Sqvect.DBPath = "./data/rag.db"
	cfg.Sqvect.TopK = 5
	cfg.Sqvect.Threshold = 0.0
	cfg.MCP.Enabled = true
	cfg.MCP.ServersConfigPath = "mcpServers.json"

	// Create provider factory
	factory := providers.NewFactory()

	// Create providers using factory
	embedder, err := factory.CreateEmbedderProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	llm, err := factory.CreateLLMProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	client, err := rag.NewClient(cfg, embedder, llm, nil)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	return client
}

func testProfileManagement(ctx context.Context, client *rag.Client) {
	// List existing profiles
	profiles, err := client.ListProfiles()
	if err != nil {
		log.Printf("Failed to list profiles: %v", err)
		return
	}

	fmt.Printf("Existing profiles: %d\n", len(profiles))
	for _, profile := range profiles {
		status := ""
		if profile.IsActive {
			status = " [ACTIVE]"
		}
		fmt.Printf("  - %s: %s%s\n", profile.Name, profile.Description, status)
	}

	// Create a new profile
	newProfile, err := client.CreateProfile("test-profile", "Test profile for advanced features")
	if err != nil {
		log.Printf("Failed to create profile: %v", err)
		return
	}
	fmt.Printf("Created new profile: %s (ID: %s)\n", newProfile.Name, newProfile.ID)

	// Update the profile
	err = client.UpdateProfile(newProfile.ID, map[string]interface{}{
		"description": "Updated test profile",
		"metadata": map[string]string{
			"environment": "test",
			"version":     "1.0",
		},
	})
	if err != nil {
		log.Printf("Failed to update profile: %v", err)
		return
	}
	fmt.Printf("Updated profile: %s\n", newProfile.ID)

	// Get the updated profile
	updatedProfile, err := client.GetProfile(newProfile.ID)
	if err != nil {
		log.Printf("Failed to get profile: %v", err)
		return
	}
	fmt.Printf("Retrieved profile: %s - %s\n", updatedProfile.Name, updatedProfile.Description)
}

func testLLMSettings(ctx context.Context, client *rag.Client) {
	// Get current LLM model
	currentModel, err := client.GetLLMModel()
	if err != nil {
		log.Printf("Failed to get current LLM model: %v", err)
		return
	}
	fmt.Printf("Current LLM model: %s\n", currentModel)

	// Get LLM settings
	llmSettings, err := client.GetLLMSettings()
	if err != nil {
		log.Printf("LLM settings not available (expected for new profile): %v", err)
	} else {
		fmt.Printf("LLM settings found: %+v\n", llmSettings)
	}

	// Try to update LLM model (this might not work if no settings exist)
	err = client.UpdateLLMModel("qwen3")
	if err != nil {
		log.Printf("Failed to update LLM model (expected): %v", err)
	} else {
		fmt.Printf("Successfully updated LLM model\n")
	}
}

func testMCPIntegration(ctx context.Context, client *rag.Client) {
	// Get MCP status
	status, err := client.GetMCPStatus(ctx)
	if err != nil {
		log.Printf("Failed to get MCP status: %v", err)
		return
	}

	// Display MCP status in a readable format
	if statusMap, ok := status.(map[string]interface{}); ok {
		fmt.Printf("MCP Enabled: %v\n", statusMap["enabled"])
		fmt.Printf("Message: %s\n", statusMap["message"])

		if servers, ok := statusMap["servers"].([]interface{}); ok {
			fmt.Printf("MCP Servers (%d):\n", len(servers))
			for i, server := range servers {
				if serverMap, ok := server.(map[string]interface{}); ok {
					name := serverMap["name"]
					description := serverMap["description"]
					running := serverMap["running"]
					toolCount := serverMap["tool_count"]

					statusStr := "Stopped"
					if running.(bool) {
						statusStr = "Running"
					}

					fmt.Printf("  %d. %s: %s (%d tools)\n", i+1, name, statusStr, toolCount)
					fmt.Printf("     Description: %s\n", description)
				}
			}
		}
	} else {
		fmt.Printf("MCP Status: %+v\n", status)
	}

	// List available tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		log.Printf("Failed to list MCP tools: %v", err)
		return
	}

	fmt.Printf("Available MCP tools: %d\n", len(tools))
	for i, tool := range tools {
		if toolMap, ok := tool.(map[string]interface{}); ok {
			fmt.Printf("  %d. %s: %s (from %s)\n",
				i+1,
				toolMap["name"],
				toolMap["description"],
				toolMap["server"])
		}
	}

	// Try to call a simple tool (if available)
	if len(tools) > 0 {
		if toolMap, ok := tools[0].(map[string]interface{}); ok {
			toolName := toolMap["name"].(string)
			fmt.Printf("Attempting to call tool: %s\n", toolName)

			result, err := client.CallTool(ctx, toolName, map[string]interface{}{
				"test": "value",
			})
			if err != nil {
				log.Printf("Failed to call tool %s: %v", toolName, err)
			} else {
				fmt.Printf("Tool result: %+v\n", result)
			}
		}
	}
}

func testEnhancedRAG(ctx context.Context, client *rag.Client) {
	// Ingest text with metadata
	text := "RAGO supports advanced features like profile management and MCP integration."
	metadata := map[string]interface{}{
		"type":     "documentation",
		"section":  "advanced-features",
		"priority": "high",
		"tags":     []string{"rago", "features", "advanced"},
	}

	resp, err := client.IngestTextWithMetadata(ctx, text, "advanced-features.txt", metadata, rag.DefaultIngestOptions())
	if err != nil {
		log.Printf("Failed to ingest text with metadata: %v", err)
		return
	}
	fmt.Printf("Ingested text with metadata: %s\n", resp.DocumentID)

	// Query with enhanced options
	opts := &rag.QueryOptions{
		TopK:         3,
		Temperature:  0.5,
		MaxTokens:    500,
		ShowSources:  true,
		ShowThinking: true,
		Filters: map[string]interface{}{
			"type": "documentation",
		},
	}

	queryResp, err := client.Query(ctx, "What advanced features does RAGO support?", opts)
	if err != nil {
		log.Printf("Failed to query: %v", err)
		return
	}

	fmt.Printf("Query Response:\n")
	fmt.Printf("Answer: %s\n", queryResp.Answer)
	fmt.Printf("Sources: %d\n", len(queryResp.Sources))

	for i, source := range queryResp.Sources {
		fmt.Printf("  Source %d: Score=%.2f\n", i+1, source.Score)
		if len(source.Content) > 100 {
			fmt.Printf("    Content: %s...\n", source.Content[:100])
		} else {
			fmt.Printf("    Content: %s\n", source.Content)
		}
	}

	// Test profile-based operations
	profiles, _ := client.ListProfiles()
	if len(profiles) > 1 {
		// Try using a different profile (simplified implementation)
		otherProfile := profiles[0]
		if otherProfile.IsActive {
			if len(profiles) > 1 {
				otherProfile = profiles[1]
			}
		}

		fmt.Printf("Querying with profile: %s\n", otherProfile.Name)
		profileResp, err := client.QueryWithProfile(ctx, otherProfile.ID, "What is RAGO?", rag.DefaultQueryOptions())
		if err != nil {
			log.Printf("Profile-based query not fully implemented: %v", err)
		} else {
			fmt.Printf("Profile query result: %s\n", profileResp.Answer)
		}
	}
}