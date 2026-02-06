package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/liliang-cn/rago/v2/pkg/agent"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/services"
)

func main() {
	ctx := context.Background()

	// 1. Load config
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Initialize global pool
	globalPool := services.GetGlobalPoolService()
	if err := globalPool.Initialize(ctx, cfg); err != nil {
		log.Fatalf("Failed to initialize pool: %v", err)
	}

	// 3. Create agent service with Router enabled
	homeDir, _ := os.UserHomeDir()
	agentDBPath := filepath.Join(homeDir, ".rago", "data", "test_intents.db")
	os.MkdirAll(filepath.Dir(agentDBPath), 0755)

	svc, err := agent.New(&agent.AgentConfig{
		Name:            "intent-test-agent",
		DBPath:          agentDBPath,
		EnableRouter:    true, // This will trigger loading from .intents/
		RouterThreshold: 0.5,  // Lower threshold for testing
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer svc.Close()

	fmt.Println("--- Testing Custom Intent Recognition ---")
	
	// Check if our custom intent is loaded
	intents := svc.Router.ListIntents()
	found := false
	for _, intent := range intents {
		if intent.Name == "weather_lookup" {
			found = true
			fmt.Printf("✅ Success: Custom intent '%s' loaded from Markdown!\n", intent.Name)
			break
		}
	}
	if !found {
		log.Fatalf("❌ Error: Custom intent 'weather_lookup' not found in router.")
	}

	// 4. Test semantic routing
	queries := []string{
		"深圳天气如何？",
		"查一下明天的天气",
		"我想看看广州的天气预报",
	}

	for _, query := range queries {
		fmt.Printf("\nQuery: %s\n", query)
		result, err := svc.Router.Route(ctx, query)
		if err != nil {
			log.Fatalf("Routing failed: %v", err)
		}

		if result.Matched {
			fmt.Printf("🎯 Matched Intent: %s (Score: %.2f)\n", result.IntentName, result.Score)
			fmt.Printf("🛠️  Mapped Tool: %s\n", result.ToolName)
		} else {
			fmt.Println("⚠️  No match found.")
		}
	}
}