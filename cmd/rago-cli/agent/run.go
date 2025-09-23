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

var agentRunCmd = &cobra.Command{
	Use:   "run [natural language request]",
	Short: "Generate and execute a workflow from natural language",
	Long: `Generate a workflow from natural language and execute it immediately.
This command uses LLM to understand your request, create an appropriate workflow,
and execute it - all in one step.

Examples:
  rago agent run "check the latest iPhone price and analyze if it's worth buying"
  rago agent run "monitor github.com/golang/go for new releases"
  rago agent run "fetch weather data for San Francisco and create a summary"
  rago agent run "analyze all JSON files in current directory and generate report"`,
	Aliases: []string{"nl"},
	Args:    cobra.MinimumNArgs(1),
	RunE:    runNaturalLanguageAgent,
}

func init() {
	// Command registration moved to setupCommands

	agentRunCmd.Flags().BoolP("dry-run", "d", false, "Analyze request but don't execute")
	agentRunCmd.Flags().BoolP("interactive", "i", false, "Review request before execution")
	agentRunCmd.Flags().BoolP("plan-only", "p", false, "Only create plan without executing")
}

func runNaturalLanguageAgent(cmd *cobra.Command, args []string) error {
	request := strings.Join(args, " ")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	interactive, _ := cmd.Flags().GetBool("interactive")
	planOnly, _ := cmd.Flags().GetBool("plan-only")

	fmt.Printf("🤖 Natural Language Request: %s\n", request)
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
	// Create LLM provider using providers factory
	factory := providers.NewFactory()
	providerConfig, err := providers.GetProviderConfig(&Cfg.Providers.ProviderConfigs)
	if err != nil {
		return fmt.Errorf("failed to get provider config: %w", err)
	}

	llmService, err := factory.CreateLLMProvider(ctx, providerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM service: %w", err)
	}

	// Interactive review
	if interactive && !dryRun {
		fmt.Print("\n❓ Execute this request? (y/n): ")
		var response string
		_, _ = fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("❌ Execution cancelled")
			return nil
		}
	}

	if dryRun {
		fmt.Println("\n🔍 Dry run mode - request analyzed but not executed")
		return nil
	}

	// Create MCP manager
	var mcpManager *mcp.Manager

	// Try to initialize with MCP if available
	if Cfg.MCP.Servers != nil && len(Cfg.MCP.Servers) > 0 {
		// Use the actual MCP config from loaded configuration
		mcpManager = mcp.NewManager(&Cfg.MCP)

		// Start essential MCP servers for agent execution
		if !quiet {
			fmt.Println("   🔧 Starting MCP servers for planning...")
		}

		// Start filesystem server (essential for file operations)
		if _, err := mcpManager.StartServer(ctx, "filesystem"); err != nil {
			fmt.Printf("   ⚠️  Warning: filesystem server failed to start: %v\n", err)
		}

		// Start memory server (useful for data storage)
		if _, err := mcpManager.StartServer(ctx, "memory"); err != nil {
			fmt.Printf("   ⚠️  Warning: memory server failed to start: %v\n", err)
		}

		if !quiet {
			fmt.Println("   ✅ MCP servers ready")
		}
	} else {
		if !quiet {
			fmt.Println("   🔧 MCP not configured - limited functionality")
		}
	}

	// Create the unified agent
	agent := agents.NewAgent(Cfg, llmService, mcpManager)
	agent.SetVerbose(!quiet)

	// If plan-only flag is set, just create and save the plan
	if planOnly {
		fmt.Println("\n📝 Creating execution plan...")
		startTime := time.Now()

		planID, err := agent.PlanOnly(ctx, request)
		if err != nil {
			return fmt.Errorf("planning failed: %w", err)
		}

		duration := time.Since(startTime)

		fmt.Println("\n✅ Plan created successfully!")
		fmt.Printf("💾 Plan ID: %s\n", planID)
		fmt.Printf("⏱️  Planning time: %v\n", duration)

		return nil
	}

	// Plan and execute the request
	fmt.Println("\n⚡ Planning and executing your request...")
	startTime := time.Now()

	result, err := agent.PlanAndExecute(ctx, request)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	duration := time.Since(startTime)
	results := result.Results

	// Display results
	fmt.Println("\n✅ Request completed successfully!")
	fmt.Printf("💾 Plan ID: %s\n", result.PlanID)
	fmt.Printf("⏱️  Execution time: %v\n", duration)

	// Display outputs
	fmt.Println("\n📊 Results:")
	if len(results) == 0 {
		fmt.Println("   (No outputs generated)")
	} else {
		for key, value := range results {
			fmt.Printf("\n--- %s ---\n", key)
			if str, ok := value.(string); ok {
				if len(str) > 500 {
					fmt.Println(str[:500] + "...\n[truncated]")
				} else {
					fmt.Println(str)
				}
			} else {
				valueStr := fmt.Sprintf("%v", value)
				if len(valueStr) > 500 {
					fmt.Println(valueStr[:500] + "...\n[truncated]")
				} else {
					fmt.Println(valueStr)
				}
			}
		}
	}

	fmt.Println("\n🎉 Task completed! Your request has been processed.")

	return nil
}
