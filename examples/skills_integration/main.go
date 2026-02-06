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
	"github.com/liliang-cn/rago/v2/pkg/skills"
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
	agentDBPath := filepath.Join(homeDir, ".rago", "data", "test_skills.db")
	os.MkdirAll(filepath.Dir(agentDBPath), 0755)

	// NEW: Simply set EnableSkills to true
	svc, err := agent.New(&agent.AgentConfig{
		Name:         "skills-test-agent",
		DBPath:       agentDBPath,
		EnableSkills: true,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer svc.Close()

	fmt.Println("--- Available Skills ---")
	// You can still access the service via svc.Skills if needed
	if svc.Skills != nil {
		skillsList, _ := svc.Skills.ListSkills(ctx, skills.SkillFilter{})
		for _, sk := range skillsList {
			fmt.Printf("- %s: %s\n", sk.ID, sk.Description)
		}
	}
	fmt.Println()

	// Run a goal that should trigger the test-skill
	fmt.Println("--- Running Goal: Greet Bob ---")
	res, err := svc.Run(ctx, "Please use the test-skill to greet Bob.")
	if err != nil {
		log.Fatalf("Run failed: %v", err)
	}

	fmt.Printf("\nFinal Result: %v\n", res.FinalResult)
}
