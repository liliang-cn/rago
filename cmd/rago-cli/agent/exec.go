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

var agentExecCmd = &cobra.Command{
	Use:   "exec [plan-id]",
	Short: "Execute a saved plan by ID",
	Long: `Execute a previously generated plan from the database.
This command executes the steps defined in a plan that was created
using the 'agent run --plan-only' command.

Examples:
  rago agent exec abc123-def456-789012
  rago agent exec 1b95bd5d-7455-4290-9765-5a5455cf8879`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentExec,
}

func init() {
	// Flags for exec command
	agentExecCmd.Flags().BoolP("verbose", "v", false, "Show detailed execution output")
}

func runAgentExec(cmd *cobra.Command, args []string) error {
	planID := args[0]
	verbose, _ := cmd.Flags().GetBool("verbose")

	fmt.Printf("📄 Executing plan ID: %s\n", planID)
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

	// Create MCP manager
	var mcpManager *mcp.Manager

	// Try to initialize with MCP if available
	if Cfg.MCP.Servers != nil && len(Cfg.MCP.Servers) > 0 {
		// Use the actual MCP config from loaded configuration
		mcpManager = mcp.NewManager(&Cfg.MCP)

		// Start essential MCP servers for execution
		if !quiet {
			fmt.Println("   🔧 Starting MCP servers for execution...")
		}

		// Start filesystem server (essential for file operations)
		if _, err := mcpManager.StartServer(ctx, "filesystem"); err != nil {
			fmt.Printf("   ⚠️  Warning: filesystem server failed to start: %v\n", err)
		}

		// Start memory server (useful for data storage)
		if _, err := mcpManager.StartServer(ctx, "memory"); err != nil {
			fmt.Printf("   ⚠️  Warning: memory server failed to start: %v\n", err)
		}

		// No need to start other servers - they'll be started on demand

		if !quiet {
			fmt.Println("   ✅ MCP servers ready")
		}
	} else {
		if !quiet {
			fmt.Println("   🔧 MCP not configured - limited functionality")
		}
	}

	// Create agent
	agent := agents.NewAgent(Cfg, llmService, mcpManager)
	agent.SetVerbose(verbose || !quiet)

	// Execute the plan
	fmt.Println("\n⚡ Executing plan...")
	startTime := time.Now()

	results, err := agent.ExecuteOnly(ctx, planID)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	duration := time.Since(startTime)

	// Display results
	fmt.Println("\n✅ Plan executed successfully!")
	fmt.Printf("⏱️  Execution time: %v\n", duration)

	// Display outputs
	fmt.Println("\n📊 Results:")
	if len(results) == 0 {
		fmt.Println("   (No outputs generated)")
	} else {
		for key, value := range results {
			// Skip internal result keys
			if strings.HasPrefix(key, "step_") || key == "last_result" {
				continue
			}

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

	// Show step results if verbose
	if verbose {
		fmt.Println("\n📝 Step Results:")
		for key, value := range results {
			if strings.HasPrefix(key, "step_") {
				fmt.Printf("\n%s:\n", key)
				fmt.Printf("%v\n", value)
			}
		}
	}

	fmt.Println("\n🎉 Task completed! The plan has been executed.")

	return nil
}
