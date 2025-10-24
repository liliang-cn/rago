package main

import (
	"context"
	"fmt"
	"log"
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

	// Configure for specialized agents
	if cfg.Agents == nil {
		cfg.Agents = &config.AgentsConfig{}
	}
	cfg.Agents.Enabled = true
	cfg.Agents.MaxAgents = 10      // Larger pool for specialized agents
	cfg.Agents.MaxConcurrent = 5   // Allow 5 concurrent specialists

	// Initialize LLM provider
	factory := providers.NewFactory()
	provider, err := providers.InitializeLLM(context.Background(), cfg, factory)
	if err != nil {
		log.Fatalf("Failed to initialize LLM provider: %v", err)
	}

	// Initialize MCP manager for tool access
	var mcpManager *mcp.Manager
	if cfg.MCP.Enabled {
		mcpManager = mcp.NewManager(&cfg.MCP)
		ctx := context.Background()
		// MCP manager is already initialized
		defer mcpManager.Close()
	}

	// Create commander
	commander := agents.NewCommander(cfg, provider, mcpManager)
	commander.SetVerbose(true)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Example 1: Software Development Team
	fmt.Println("\n" + "="*60)
	fmt.Println("EXAMPLE 1: Software Development Team Simulation")
	fmt.Println("="*60 + "\n")

	devTeamGoal := `Develop a simple web application feature:
1. Product Manager: Define requirements for a user authentication feature
2. Backend Developer: Design the API endpoints and database schema
3. Frontend Developer: Design the user interface components
4. QA Engineer: Create test cases and validation criteria
5. DevOps Engineer: Plan deployment and monitoring strategy
Coordinate all agents to deliver a complete feature specification.`

	mission1, err := commander.ExecuteMission(ctx, devTeamGoal)
	if err != nil {
		fmt.Printf("Development team mission failed: %v\n", err)
	} else {
		fmt.Printf("\n✅ Feature specification completed!\n")
		displayMissionResults(mission1)
	}

	// Example 2: Research Team
	fmt.Println("\n" + "="*60)
	fmt.Println("EXAMPLE 2: Research Team Analysis")
	fmt.Println("="*60 + "\n")

	researchTeamGoal := `Conduct comprehensive research on "Impact of AI on Education":
1. Literature Reviewer: Survey existing academic papers and studies
2. Data Analyst: Analyze trends and statistics in AI adoption
3. Field Expert: Provide domain expertise on educational methodologies
4. Synthesizer: Combine all findings into cohesive insights
5. Critic: Identify gaps and potential issues in the analysis
Work collaboratively to produce a thorough research report.`

	mission2, err := commander.ExecuteMission(ctx, researchTeamGoal)
	if err != nil {
		fmt.Printf("Research team mission failed: %v\n", err)
	} else {
		fmt.Printf("\n✅ Research analysis completed!\n")
		displayMissionResults(mission2)
	}

	// Example 3: Business Strategy Team
	fmt.Println("\n" + "="*60)
	fmt.Println("EXAMPLE 3: Business Strategy Consultation")
	fmt.Println("="*60 + "\n")

	strategyTeamGoal := `Develop a market entry strategy for a new product:
1. Market Analyst: Analyze target market size and demographics
2. Competitive Intelligence: Research competitor products and strategies
3. Financial Analyst: Project revenue and cost estimates
4. Risk Assessor: Identify potential risks and mitigation strategies
5. Strategy Consultant: Synthesize findings into actionable recommendations
Create a comprehensive market entry plan.`

	mission3, err := commander.ExecuteMission(ctx, strategyTeamGoal)
	if err != nil {
		fmt.Printf("Strategy team mission failed: %v\n", err)
	} else {
		fmt.Printf("\n✅ Market strategy completed!\n")
		displayMissionResults(mission3)
	}

	// Example 4: Creative Content Team
	fmt.Println("\n" + "="*60)
	fmt.Println("EXAMPLE 4: Creative Content Production")
	fmt.Println("="*60 + "\n")

	creativeTeamGoal := `Create a marketing campaign for an eco-friendly product:
1. Creative Director: Develop the overall campaign concept and theme
2. Copywriter: Write compelling ad copy and slogans
3. Content Strategist: Plan content distribution across channels
4. Social Media Expert: Design social media engagement strategy
5. Analytics Specialist: Define success metrics and tracking plan
Deliver an integrated marketing campaign proposal.`

	mission4, err := commander.ExecuteMission(ctx, creativeTeamGoal)
	if err != nil {
		fmt.Printf("Creative team mission failed: %v\n", err)
	} else {
		fmt.Printf("\n✅ Marketing campaign completed!\n")
		displayMissionResults(mission4)
	}

	// Example 5: Crisis Response Team
	fmt.Println("\n" + "="*60)
	fmt.Println("EXAMPLE 5: Crisis Response Coordination")
	fmt.Println("="*60 + "\n")

	crisisTeamGoal := `Respond to a hypothetical data breach incident:
1. Incident Commander: Assess the situation and coordinate response
2. Security Analyst: Investigate the breach and identify vulnerabilities
3. Legal Advisor: Evaluate compliance and legal implications
4. Communications Lead: Draft internal and external communications
5. Recovery Specialist: Plan system restoration and data recovery
Execute a coordinated incident response plan.`

	mission5, err := commander.ExecuteMission(ctx, crisisTeamGoal)
	if err != nil {
		fmt.Printf("Crisis response mission failed: %v\n", err)
	} else {
		fmt.Printf("\n✅ Crisis response plan completed!\n")
		displayMissionResults(mission5)
	}

	// Display overall performance
	fmt.Println("\n" + "="*60)
	fmt.Println("SPECIALIZED AGENT PERFORMANCE ANALYSIS")
	fmt.Println("="*60 + "\n")

	metrics := commander.GetMetrics()
	
	fmt.Println("📊 Mission Statistics:")
	fmt.Printf("  Total missions: %v\n", metrics["total_missions"])
	fmt.Printf("  Successful: %v\n", metrics["completed"])
	fmt.Printf("  Failed: %v\n", metrics["failed"])
	fmt.Printf("  Success rate: %.2f%%\n", metrics["success_rate"].(float64)*100)
	
	fmt.Println("\n🤖 Agent Pool Statistics:")
	fmt.Printf("  Pool capacity: %v agents\n", metrics["agent_pool_size"])
	fmt.Printf("  Currently available: %v agents\n", metrics["agents_available"])
	
	// List all missions with details
	fmt.Println("\n📋 Mission Summary:")
	missions := commander.ListMissions()
	for i, mission := range missions {
		fmt.Printf("\n  Mission %d: %s\n", i+1, mission.ID[:8])
		fmt.Printf("    Goal: %.50s...\n", mission.Goal)
		fmt.Printf("    Strategy: %s\n", mission.Strategy.Type)
		fmt.Printf("    Tasks: %d\n", len(mission.Strategy.Decomposition))
		fmt.Printf("    Status: %s\n", mission.Status)
		if mission.EndTime != nil {
			fmt.Printf("    Duration: %v\n", mission.EndTime.Sub(mission.StartTime))
		}
		
		// Show task specializations
		fmt.Println("    Specializations used:")
		taskTypes := make(map[agents.TaskType]int)
		for _, task := range mission.Strategy.Decomposition {
			taskTypes[task.Type]++
		}
		for taskType, count := range taskTypes {
			fmt.Printf("      - %s: %d tasks\n", taskType, count)
		}
	}

	// Demonstrate agent efficiency over time
	fmt.Println("\n📈 Agent Learning and Efficiency:")
	fmt.Println("  (Agents improve their efficiency with each successful task)")
	fmt.Println("  This demonstrates the adaptive nature of the multi-agent system")
	fmt.Println("  where agents learn from experience and become more effective.")
}

// Helper function to display mission results
func displayMissionResults(mission *agents.Mission) {
	fmt.Printf("Mission ID: %s\n", mission.ID[:8])
	fmt.Printf("Strategy: %s\n", mission.Strategy.Type)
	fmt.Printf("Tasks completed: %d\n", len(mission.Agents))
	
	if mission.EndTime != nil {
		fmt.Printf("Time taken: %v\n", mission.EndTime.Sub(mission.StartTime))
	}
	
	// Show task breakdown
	fmt.Println("\nTask Execution Summary:")
	for _, task := range mission.Strategy.Decomposition {
		agentTask, exists := mission.Agents[task.ID]
		if exists {
			status := "✅"
			if agentTask.Status != agents.AgentStatusCompleted {
				status = "❌"
			}
			fmt.Printf("  %s [%s] %s\n", status, task.Type, task.Description)
		}
	}
	
	// Show final output if available
	if finalResult, exists := mission.Results["final"]; exists {
		fmt.Printf("\nFinal Output:\n")
		switch v := finalResult.(type) {
		case string:
			// Truncate long strings
			if len(v) > 200 {
				fmt.Printf("  %.200s...\n", v)
			} else {
				fmt.Printf("  %s\n", v)
			}
		default:
			fmt.Printf("  %v\n", v)
		}
	}
	
	// Show any errors
	if len(mission.Errors) > 0 {
		fmt.Println("\n⚠️  Errors encountered:")
		for _, err := range mission.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}
}