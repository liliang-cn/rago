package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/providers"
)

func main() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Enable agents and set multi-agent parameters
	if cfg.Agents == nil {
		cfg.Agents = &config.AgentsConfig{}
	}
	cfg.Agents.Enabled = true
	cfg.Agents.MaxAgents = 5       // Allow up to 5 concurrent agents
	cfg.Agents.MaxConcurrent = 3   // Max 3 agents working simultaneously
	cfg.Agents.DefaultTimeout = 300 // 5 minutes timeout

	// Initialize LLM provider
	factory := providers.NewFactory()
	provider, err := providers.InitializeLLM(context.Background(), cfg, factory)
	if err != nil {
		log.Fatalf("Failed to initialize LLM provider: %v", err)
	}

	// Initialize MCP manager if enabled
	var mcpManager *mcp.Manager
	if cfg.MCP.Enabled {
		mcpManager = mcp.NewManager(&cfg.MCP)
		ctx := context.Background()
		// MCP manager is already initialized
		defer mcpManager.Close()
	}

	// Create commander for multi-agent orchestration
	commander := agents.NewCommander(cfg, provider, mcpManager)
	commander.SetVerbose(true)

	// Example 1: Parallel Research Task
	fmt.Println("\n" + "="*60)
	fmt.Println("EXAMPLE 1: Parallel Research on Multiple Topics")
	fmt.Println("="*60 + "\n")

	researchGoal := `Research and analyze the following topics in parallel:
1. Latest developments in quantum computing
2. Current state of renewable energy technologies
3. Recent breakthroughs in artificial intelligence
Provide a comprehensive summary of each topic.`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	mission1, err := commander.ExecuteMission(ctx, researchGoal)
	if err != nil {
		fmt.Printf("Research mission failed: %v\n", err)
	} else {
		fmt.Printf("\n✅ Research completed!\n")
		fmt.Printf("Mission ID: %s\n", mission1.ID[:8])
		fmt.Printf("Strategy used: %s\n", mission1.Strategy.Type)
		fmt.Printf("Tasks executed: %d\n", len(mission1.Agents))
		
		// Display results
		if finalResult, exists := mission1.Results["final"]; exists {
			fmt.Printf("\nFinal Research Summary:\n%v\n", finalResult)
		}
	}

	// Example 2: Pipeline Processing Task
	fmt.Println("\n" + "="*60)
	fmt.Println("EXAMPLE 2: Pipeline Data Processing")
	fmt.Println("="*60 + "\n")

	pipelineGoal := `Process this data through a pipeline:
1. First, analyze the text: "The quick brown fox jumps over the lazy dog"
2. Then, extract key entities and concepts
3. Finally, generate a semantic representation`

	mission2, err := commander.ExecuteMission(ctx, pipelineGoal)
	if err != nil {
		fmt.Printf("Pipeline mission failed: %v\n", err)
	} else {
		fmt.Printf("\n✅ Pipeline completed!\n")
		fmt.Printf("Mission ID: %s\n", mission2.ID[:8])
		fmt.Printf("Strategy used: %s\n", mission2.Strategy.Type)
		
		// Show pipeline stages
		fmt.Println("\nPipeline stages completed:")
		for taskID, result := range mission2.Results {
			fmt.Printf("  - %s: %v\n", taskID, result)
		}
	}

	// Example 3: Map-Reduce Analysis
	fmt.Println("\n" + "="*60)
	fmt.Println("EXAMPLE 3: Map-Reduce Document Analysis")
	fmt.Println("="*60 + "\n")

	mapReduceGoal := `Analyze these documents using map-reduce:
1. Document A: "Machine learning is transforming industries"
2. Document B: "Deep learning enables new AI capabilities"
3. Document C: "Neural networks power modern AI systems"
Map phase: Extract key concepts from each document
Reduce phase: Synthesize a unified understanding`

	mission3, err := commander.ExecuteMission(ctx, mapReduceGoal)
	if err != nil {
		fmt.Printf("Map-Reduce mission failed: %v\n", err)
	} else {
		fmt.Printf("\n✅ Map-Reduce completed!\n")
		fmt.Printf("Mission ID: %s\n", mission3.ID[:8])
		
		// Show map and reduce results
		if mapResults, exists := mission3.Results["map_results"]; exists {
			fmt.Printf("\nMap phase results:\n%v\n", mapResults)
		}
		if finalResult, exists := mission3.Results["final"]; exists {
			fmt.Printf("\nReduced result:\n%v\n", finalResult)
		}
	}

	// Example 4: Complex Multi-Stage Task
	fmt.Println("\n" + "="*60)
	fmt.Println("EXAMPLE 4: Complex Multi-Agent Collaboration")
	fmt.Println("="*60 + "\n")

	complexGoal := `Build a comprehensive market analysis:
1. Research current trends in the tech industry
2. Analyze competitor strategies for top 3 companies
3. Identify emerging opportunities and threats
4. Synthesize findings into strategic recommendations
Use multiple agents working in parallel where possible.`

	mission4, err := commander.ExecuteMission(ctx, complexGoal)
	if err != nil {
		fmt.Printf("Complex mission failed: %v\n", err)
	} else {
		fmt.Printf("\n✅ Complex analysis completed!\n")
		fmt.Printf("Mission ID: %s\n", mission4.ID[:8])
		fmt.Printf("Execution time: %v\n", mission4.EndTime.Sub(mission4.StartTime))
		
		// Show task breakdown
		fmt.Println("\nTasks executed:")
		for _, task := range mission4.Strategy.Decomposition {
			fmt.Printf("  - [%s] %s\n", task.Type, task.Description)
		}
		
		// Show final recommendations
		if finalResult, exists := mission4.Results["final"]; exists {
			fmt.Printf("\nStrategic Recommendations:\n%v\n", finalResult)
		}
	}

	// Display commander metrics
	fmt.Println("\n" + "="*60)
	fmt.Println("COMMANDER PERFORMANCE METRICS")
	fmt.Println("="*60 + "\n")

	metrics := commander.GetMetrics()
	fmt.Printf("Total missions executed: %v\n", metrics["total_missions"])
	fmt.Printf("Success rate: %.2f%%\n", metrics["success_rate"].(float64)*100)
	fmt.Printf("Average execution time: %v\n", metrics["average_time"])
	fmt.Printf("Agent pool utilization: %v/%v\n", 
		metrics["agents_available"], metrics["agent_pool_size"])

	// Example 5: Dynamic Agent Allocation
	fmt.Println("\n" + "="*60)
	fmt.Println("EXAMPLE 5: Dynamic Agent Allocation")
	fmt.Println("="*60 + "\n")

	// Create multiple concurrent missions to test agent pool management
	missions := make(chan *agents.Mission, 3)
	errors := make(chan error, 3)

	goals := []string{
		"Analyze the impact of AI on healthcare",
		"Research blockchain applications in finance",
		"Study quantum computing use cases",
	}

	// Launch missions concurrently
	for i, goal := range goals {
		go func(idx int, g string) {
			fmt.Printf("🚀 Launching mission %d: %s\n", idx+1, g)
			mission, err := commander.ExecuteMission(ctx, g)
			if err != nil {
				errors <- err
			} else {
				missions <- mission
			}
		}(i, goal)
	}

	// Collect results
	completedMissions := 0
	timeout := time.After(5 * time.Minute)

	for completedMissions < len(goals) {
		select {
		case mission := <-missions:
			completedMissions++
			fmt.Printf("✅ Mission %s completed\n", mission.ID[:8])
		case err := <-errors:
			completedMissions++
			fmt.Printf("❌ Mission failed: %v\n", err)
		case <-timeout:
			fmt.Println("⏰ Timeout reached")
			os.Exit(1)
		}
	}

	fmt.Printf("\n🎯 All concurrent missions completed!\n")
	
	// Final metrics
	finalMetrics := commander.GetMetrics()
	fmt.Printf("\nFinal Statistics:\n")
	fmt.Printf("  Total missions: %v\n", finalMetrics["total_missions"])
	fmt.Printf("  Completed: %v\n", finalMetrics["completed"])
	fmt.Printf("  Failed: %v\n", finalMetrics["failed"])
	fmt.Printf("  Success rate: %.2f%%\n", finalMetrics["success_rate"].(float64)*100)
}