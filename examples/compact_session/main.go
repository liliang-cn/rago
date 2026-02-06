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

	// Load config
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize global pool
	globalPool := services.GetGlobalPoolService()
	if err := globalPool.Initialize(ctx, cfg); err != nil {
		log.Fatalf("Failed to initialize pool: %v", err)
	}

	// Create agent service with simplified API
	homeDir, _ := os.UserHomeDir()
	agentDBPath := filepath.Join(homeDir, ".rago", "data", "test_compact.db")
	
	// Ensure directory exists
	os.MkdirAll(filepath.Dir(agentDBPath), 0755)

	svc, err := agent.New(&agent.AgentConfig{
		Name:   "compact-test-agent",
		DBPath: agentDBPath,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer svc.Close()

	// Start a chat to create a session
	fmt.Println("--- Starting Conversation ---")
	res1, err := svc.Chat(ctx, "Hello, my name is Alice. I'm a software engineer from Shanghai.")
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}
	fmt.Printf("Agent: %v\n\n", res1.FinalResult)

	res2, err := svc.Chat(ctx, "I like hiking and reading science fiction. I'm currently learning Go programming.")
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}
	fmt.Printf("Agent: %v\n\n", res2.FinalResult)

	sessionID := svc.CurrentSessionID()
	fmt.Printf("Current Session ID: %s\n\n", sessionID)

	// Compact the session
	fmt.Println("--- Compacting Session ---")
	summary, err := svc.CompactSession(ctx, sessionID)
	if err != nil {
		log.Fatalf("CompactSession failed: %v", err)
	}
	fmt.Printf("Summary: %s\n\n", summary)

	// Verify the summary is used in a new prompt
	fmt.Println("--- Verifying Summary Usage ---")
	
	res3, err := svc.RunWithSession(ctx, "Based on what we've discussed, what do you know about me?", sessionID)
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}
	fmt.Printf("Agent: %v\n", res3.FinalResult)
}