package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/providers"
	"github.com/spf13/cobra"
)

var agentDoCmd = &cobra.Command{
	Use:   "do [request]",
	Short: "Intelligently handle requests using RAG, planning, and execution",
	Long: `The 'do' command combines RAG retrieval, intelligent planning, and execution.
It first searches the knowledge base for relevant context, determines if tools
are needed, and either answers directly or creates and executes a plan.

This is the most intelligent mode that combines:
- RAG (Retrieval Augmented Generation) from your knowledge base
- Smart decision making about tool usage
- Automatic planning and execution when needed
- Comprehensive answer synthesis

Examples:
  rago agent do "what is the architecture of this project?"
  rago agent do "create a backup of all configuration files"
  rago agent do "explain how the MCP integration works"
  rago agent do "analyze the performance of the system"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAgentDo,
}

func init() {
	agentDoCmd.Flags().BoolP("verbose", "v", false, "Show detailed progress")
	agentDoCmd.Flags().BoolP("show-context", "c", false, "Show RAG context used")
	agentDoCmd.Flags().BoolP("show-plan", "p", false, "Show execution plan if created")
}

func runAgentDo(cmd *cobra.Command, args []string) error {
	request := strings.Join(args, " ")
	verbose, _ := cmd.Flags().GetBool("verbose")
	showContext, _ := cmd.Flags().GetBool("show-context")
	showPlan, _ := cmd.Flags().GetBool("show-plan")

	fmt.Printf("🤖 Intelligent Request: %s\n", request)
	fmt.Println("=" + strings.Repeat("=", 50))

	// Load config if not already loaded
	if Cfg == nil {
		var err error
		Cfg, err = config.Load("")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Initialize providers
	ctx := context.Background()
	factory := providers.NewFactory()
	providerConfig, err := providers.GetProviderConfig(&Cfg.Providers.ProviderConfigs)
	if err != nil {
		return fmt.Errorf("failed to get provider config: %w", err)
	}

	llmService, err := factory.CreateLLMProvider(ctx, providerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM service: %w", err)
	}

	// Initialize embedder for RAG
	embedderService, err := factory.CreateEmbedderProvider(ctx, providerConfig)
	if err != nil && verbose {
		fmt.Printf("⚠️  Warning: Failed to initialize embedder: %v\n", err)
	}

	// Create MCP manager
	var mcpManager *mcp.Manager

	// Try to initialize with MCP if available
	if Cfg.MCP.Servers != nil && len(Cfg.MCP.Servers) > 0 {
		mcpManager = mcp.NewManager(&Cfg.MCP)

		if verbose {
			fmt.Println("   🔧 Starting MCP servers...")
		}

		// Start essential MCP servers
		if _, err := mcpManager.StartServer(ctx, "filesystem"); err != nil && verbose {
			fmt.Printf("   ⚠️  Warning: filesystem server failed to start: %v\n", err)
		}

		if _, err := mcpManager.StartServer(ctx, "memory"); err != nil && verbose {
			fmt.Printf("   ⚠️  Warning: memory server failed to start: %v\n", err)
		}

		if verbose {
			fmt.Println("   ✅ MCP servers ready")
		}
	}

	// Create the unified agent with embedder for RAG
	agent := agents.NewAgentWithEmbedder(Cfg, llmService, embedderService, mcpManager)
	agent.SetVerbose(verbose || !quiet)
	defer agent.Close()

	// Execute the intelligent Do operation
	fmt.Println("\n🧠 Processing your request intelligently...")
	startTime := time.Now()

	result, err := agent.Do(ctx, request)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	duration := time.Since(startTime)

	// Display results based on what happened
	fmt.Println("\n" + strings.Repeat("=", 50))

	// Show RAG context if requested
	if showContext && result.RAGContext != "" {
		fmt.Println("\n📚 Knowledge Base Context:")
		fmt.Println(result.RAGContext)
		fmt.Println("\n" + strings.Repeat("-", 40))
	}

	// Show enhanced request if different
	if result.EnhancedRequest != "" && result.EnhancedRequest != request {
		fmt.Println("\n🔍 Enhanced Request:")
		fmt.Println(result.EnhancedRequest)
	}

	// Show approach decision
	if result.NeedsTools {
		fmt.Println("\n🔧 Approach: Tool execution required")

		if result.PlanID != "" {
			fmt.Printf("📋 Plan ID: %s\n", result.PlanID)

			if showPlan {
				// Optionally show the plan details
				if plan, err := agent.GetPlan(result.PlanID); err == nil {
					fmt.Printf("\nGoal: %s\n", plan.Goal)
					fmt.Printf("Steps: %d\n", len(plan.Steps))
					for _, step := range plan.Steps {
						fmt.Printf("  %d. %s - %s\n", step.StepNumber, step.Tool, step.Description)
					}
				}
			}
		}
	} else {
		fmt.Println("\n💡 Approach: Direct answer from knowledge base")
	}

	// Show the answer
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📝 Answer:")
	fmt.Println()

	if result.DirectAnswer != "" {
		fmt.Println(result.DirectAnswer)
	} else if result.FinalAnswer != "" {
		fmt.Println(result.FinalAnswer)
	} else if result.ExecutionResults != nil {
		// Fallback to showing execution results
		fmt.Println("Execution Results:")
		for key, value := range result.ExecutionResults {
			if !strings.HasPrefix(key, "step_") && key != "last_result" {
				fmt.Printf("\n%s:\n%v\n", key, value)
			}
		}
	}

	// Show timing and status
	fmt.Println("\n" + strings.Repeat("=", 50))
	if result.Success {
		fmt.Println("✅ Request completed successfully!")
	} else {
		fmt.Println("⚠️  Request completed with warnings")
	}
	fmt.Printf("⏱️  Total time: %v\n", duration)

	// Show statistics
	if result.RAGContext != "" {
		fmt.Println("📊 Used knowledge base context")
	}
	if result.PlanID != "" {
		fmt.Println("📊 Created and executed plan")
	}

	return nil
}
