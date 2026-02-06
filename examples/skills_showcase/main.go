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

	// 3. Create agent service with full capabilities
	homeDir, _ := os.UserHomeDir()
	agentDBPath := filepath.Join(homeDir, ".rago", "data", "showcase.db")
	os.MkdirAll(filepath.Dir(agentDBPath), 0755)

	svc, err := agent.New(&agent.AgentConfig{
		Name:         "rago-showcase-agent",
		DBPath:       agentDBPath,
		EnableRouter: true,
		EnableMCP:    true,
		EnableSkills: true,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer svc.Close()

	fmt.Println("--- RAGO Skills & MCP Showcase ---")
	
	// Showcase 1: Code Review Skill
	codeFile := "pkg/router/loader.go"
	codeContent, _ := os.ReadFile(codeFile)
	
	fmt.Printf("\nGoal: Review the code in %s\n", codeFile)
	res1, err := svc.Run(ctx, fmt.Sprintf("Please use the code-reviewer skill to analyze this code:\n\n%s", string(codeContent)))
	if err != nil {
		fmt.Printf("Run failed: %v\n", err)
	} else {
		fmt.Printf("\n--- Code Review Result ---\n%v\n", res1.FinalResult)
	}

	// Showcase 2: Web Research + MCP Fetch
	url := "https://go.dev/blog/go1.24"
	fmt.Printf("\nGoal: Research and summarize %s\n", url)
	res2, err := svc.Run(ctx, fmt.Sprintf("Use the web-researcher skill to summarize this page: %s", url))
	if err != nil {
		fmt.Printf("Run failed: %v\n", err)
	} else {
		fmt.Printf("\n--- Web Research Result ---\n%v\n", res2.FinalResult)
	}
}