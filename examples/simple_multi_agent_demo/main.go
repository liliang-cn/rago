package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

func main() {
	fmt.Println("🚀 Multi-Agent System Demo")
	fmt.Println(strings.Repeat("=", 50))

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Configure agents
	if cfg.Agents == nil {
		cfg.Agents = &config.AgentsConfig{}
	}
	cfg.Agents.Enabled = true
	cfg.Agents.MaxAgents = 3

	// Initialize LLM provider
	factory := providers.NewFactory()
	provider, err := providers.InitializeLLM(context.Background(), cfg, factory)
	if err != nil {
		log.Fatalf("Failed to initialize LLM: %v", err)
	}

	// Create commander (no MCP for this simple demo)
	commander := agents.NewCommander(cfg, provider, nil)
	commander.SetVerbose(true)

	// Demo 1: Simple Parallel Tasks
	fmt.Println("\n📋 Demo 1: Parallel Task Execution")
	fmt.Println(strings.Repeat("-", 40))
	
	goal1 := `Complete these three independent tasks in parallel:
1. Write a haiku about technology
2. List 5 benefits of cloud computing
3. Explain what machine learning is in one sentence`

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	mission1, err := commander.ExecuteMission(ctx, goal1)
	if err != nil {
		fmt.Printf("❌ Mission failed: %v\n", err)
	} else {
		fmt.Printf("✅ Mission completed in %v\n", mission1.EndTime.Sub(mission1.StartTime))
		fmt.Printf("Strategy used: %s\n", mission1.Strategy.Type)
		fmt.Printf("Tasks executed: %d\n", len(mission1.Strategy.Decomposition))
	}

	// Demo 2: Sequential Processing
	fmt.Println("\n📋 Demo 2: Sequential Task Processing")
	fmt.Println(strings.Repeat("-", 40))
	
	goal2 := `Process this information step by step:
1. First, generate a random number between 1 and 100
2. Then, determine if it's prime
3. Finally, write a short fact about that number`

	mission2, err := commander.ExecuteMission(ctx, goal2)
	if err != nil {
		fmt.Printf("❌ Mission failed: %v\n", err)
	} else {
		fmt.Printf("✅ Mission completed in %v\n", mission2.EndTime.Sub(mission2.StartTime))
		fmt.Printf("Strategy used: %s\n", mission2.Strategy.Type)
	}

	// Show final metrics
	fmt.Println("\n📊 Commander Metrics")
	fmt.Println(strings.Repeat("-", 40))
	metrics := commander.GetMetrics()
	fmt.Printf("Total missions: %v\n", metrics["total_missions"])
	fmt.Printf("Success rate: %.2f%%\n", metrics["success_rate"].(float64)*100)
	fmt.Printf("Agent pool size: %v\n", metrics["agent_pool_size"])

	fmt.Println("\n✨ Multi-Agent Demo Complete!")
}