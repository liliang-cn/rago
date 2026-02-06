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

	// Create agent service
	homeDir, _ := os.UserHomeDir()
	agentDBPath := filepath.Join(homeDir, ".rago", "data", "skills_integration.db")
	os.MkdirAll(filepath.Dir(agentDBPath), 0755)

	svc, err := agent.New(&agent.AgentConfig{
		Name:         "skills-integration-agent",
		DBPath:       agentDBPath,
		EnableSkills: true,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer svc.Close()

	fmt.Println("--- Skills Integration Demo ---")
	// List available skills
	if svc.Skills != nil {
		skillsList, _ := svc.Skills.ListSkills(ctx, skills.SkillFilter{})
		fmt.Printf("Found %d skills:\n", len(skillsList))
		for _, sk := range skillsList {
			fmt.Printf("- %s: %s\n", sk.ID, sk.Description)
		}
	}
	fmt.Println()

	// Run a goal
	fmt.Println("--- Running Goal: What can you help me with? ---")
	res, err := svc.Run(ctx, "What skills do you have available?")
	if err != nil {
		log.Fatalf("Run failed: %v", err)
	}

	fmt.Printf("\nFinal Result: %v\n", res.FinalResult)

	fmt.Println("\nSkills integration example completed successfully!")
}
