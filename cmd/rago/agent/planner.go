package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/liliang-cn/rago/v2/client"
	"github.com/liliang-cn/rago/v2/pkg/agents/planner"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/providers"
	"github.com/spf13/cobra"
)

var (
	// Agent planner flags
	plannerVerbose   bool
	plannerStorageDir string
	plannerResume    bool
)

// agentPlanCmd creates a new plan
var agentPlanCmd = &cobra.Command{
	Use:   "plan [goal]",
	Short: "Create an execution plan for a goal",
	Long: `Create a detailed execution plan for achieving a goal.
The plan will be saved to the filesystem with tracking enabled.

Examples:
  # Create a plan for code analysis
  rago agent plan "Analyze the codebase and identify performance bottlenecks"
  
  # Create a plan with custom storage
  rago agent plan "Refactor the authentication module" --storage /tmp/agent-plans
  
  # Create a plan with verbose output
  rago agent plan "Generate API documentation" -v`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAgentPlan,
}

// agentRunPlanCmd executes a plan
var agentRunPlanCmd = &cobra.Command{
	Use:   "run-plan [plan-id]",
	Short: "Execute a plan with tracking",
	Long: `Execute a previously created plan with progress tracking.
Use --resume to continue a paused or failed plan.

Examples:
  # Execute a plan
  rago agent run-plan abc123
  
  # Resume a failed plan
  rago agent run-plan abc123 --resume
  
  # Execute with verbose output
  rago agent run-plan abc123 -v`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentExecutePlan,
}

// agentListPlansCmd lists all plans
var agentListPlansCmd = &cobra.Command{
	Use:   "list-plans",
	Short: "List all execution plans",
	Long: `List all created plans with their status and progress.

Examples:
  # List all plans
  rago agent list-plans
  
  # List with custom storage location
  rago agent list-plans --storage /tmp/agent-plans`,
	RunE: runAgentListPlans,
}

// agentPlanStatusCmd shows plan status
var agentPlanStatusCmd = &cobra.Command{
	Use:   "plan-status [plan-id]",
	Short: "Show plan status and progress",
	Long: `Display detailed status and progress for a specific plan.

Examples:
  # Show plan status
  rago agent plan-status abc123
  
  # Show with verbose details
  rago agent plan-status abc123 -v`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentPlanStatus,
}

// agentAutoRunCmd creates and executes a plan in one command
var agentAutoRunCmd = &cobra.Command{
	Use:   "auto-run [goal]",
	Short: "Create and execute a plan automatically",
	Long: `Create a plan for the given goal and immediately execute it.
This combines 'agent plan' and 'agent run-plan' into a single command.

Examples:
  # Auto-run a simple task
  rago agent auto-run "Create a README file with project documentation"
  
  # Auto-run with verbose output
  rago agent auto-run "Analyze and optimize database queries" -v`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAgentAutoRun,
}

func init() {
	// Command registration moved to setupCommands
	
	// Add flags
	agentPlanCmd.Flags().BoolVarP(&plannerVerbose, "verbose", "v", false, "Enable verbose output")
	agentPlanCmd.Flags().StringVar(&plannerStorageDir, "storage", "", "Storage directory for plans (default: ~/.rago/agents)")
	
	agentRunPlanCmd.Flags().BoolVarP(&plannerVerbose, "verbose", "v", false, "Enable verbose output")
	agentRunPlanCmd.Flags().StringVar(&plannerStorageDir, "storage", "", "Storage directory for plans")
	agentRunPlanCmd.Flags().BoolVar(&plannerResume, "resume", false, "Resume a paused or failed plan")
	
	agentListPlansCmd.Flags().StringVar(&plannerStorageDir, "storage", "", "Storage directory for plans")
	
	agentPlanStatusCmd.Flags().BoolVarP(&plannerVerbose, "verbose", "v", false, "Show detailed progress")
	agentPlanStatusCmd.Flags().StringVar(&plannerStorageDir, "storage", "", "Storage directory for plans")
	
	agentAutoRunCmd.Flags().BoolVarP(&plannerVerbose, "verbose", "v", false, "Enable verbose output")
	agentAutoRunCmd.Flags().StringVar(&plannerStorageDir, "storage", "", "Storage directory for plans")
}

// runAgentPlan creates a new plan
func runAgentPlan(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	goal := strings.Join(args, " ")
	
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	// Set storage directory
	if plannerStorageDir == "" {
		home, _ := os.UserHomeDir()
		plannerStorageDir = filepath.Join(home, ".rago", "agents")
	}
	
	// Create LLM provider using factory
	fact := providers.NewFactory()
	_, err = providers.DetermineProviderType(&cfg.Providers.ProviderConfigs)
	if err != nil {
		return fmt.Errorf("failed to determine provider type: %w", err)
	}
	
	providerConfig, err := providers.GetProviderConfig(&cfg.Providers.ProviderConfigs)
	if err != nil {
		return fmt.Errorf("failed to get provider config: %w", err)
	}
	
	llmProvider, err := fact.CreateLLMProvider(ctx, providerConfig)
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}
	
	// Create planner
	agentPlanner := planner.NewAgentPlanner(llmProvider, plannerStorageDir)
	agentPlanner.SetVerbose(plannerVerbose)
	
	// Initialize MCP if available
	mcpClient, err := initializePlannerMCP(ctx, cfg)
	if err != nil {
		fmt.Printf("⚠️  Warning: MCP initialization failed: %v\n", err)
		fmt.Println("Continuing without MCP tools...")
	} else if mcpClient != nil {
		// Set available tools for planning
		// Convert MCP tools to domain.ToolDefinition format
		mcpTools := mcpClient.GetTools()
		tools := make([]domain.ToolDefinition, 0, len(mcpTools))
		for _, tool := range mcpTools {
			tools = append(tools, domain.ToolDefinition{
				Type: "function",
				Function: domain.ToolFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  make(map[string]interface{}), // TODO: convert InputSchema
				},
			})
		}
		agentPlanner.SetMCPTools(tools)
		if plannerVerbose {
			fmt.Printf("📦 Loaded %d MCP tools for planning\n", len(tools))
		}
	}
	
	// Create the plan
	fmt.Printf("🤔 Creating plan for: %s\n", goal)
	plan, err := agentPlanner.CreatePlan(ctx, goal)
	if err != nil {
		return fmt.Errorf("failed to create plan: %w", err)
	}
	
	// Display plan summary
	fmt.Printf("\n📋 Plan created: %s\n", plan.ID)
	fmt.Printf("📝 Summary: %s\n", plan.Summary)
	fmt.Printf("📊 Tasks: %d, Total steps: %d\n", len(plan.Tasks), plan.TotalSteps)
	
	if plannerVerbose {
		fmt.Println("\n📌 Tasks:")
		for i, task := range plan.Tasks {
			fmt.Printf("%d. %s (%d steps)\n", i+1, task.Name, len(task.Steps))
			if len(task.Dependencies) > 0 {
				fmt.Printf("   Dependencies: %v\n", task.Dependencies)
			}
			if len(task.Tools) > 0 {
				fmt.Printf("   Tools: %v\n", task.Tools)
			}
		}
	}
	
	fmt.Printf("\n✅ Plan saved. Execute with: rago agent run-plan %s\n", plan.ID)
	
	return nil
}

// runAgentExecutePlan executes a plan
func runAgentExecutePlan(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	planID := args[0]
	
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	// Set storage directory
	if plannerStorageDir == "" {
		home, _ := os.UserHomeDir()
		plannerStorageDir = filepath.Join(home, ".rago", "agents")
	}
	
	// Create LLM provider using factory
	fact := providers.NewFactory()
	_, err = providers.DetermineProviderType(&cfg.Providers.ProviderConfigs)
	if err != nil {
		return fmt.Errorf("failed to determine provider type: %w", err)
	}
	
	providerConfig, err := providers.GetProviderConfig(&cfg.Providers.ProviderConfigs)
	if err != nil {
		return fmt.Errorf("failed to get provider config: %w", err)
	}
	
	llmProvider, err := fact.CreateLLMProvider(ctx, providerConfig)
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}
	
	// Create planner
	agentPlanner := planner.NewAgentPlanner(llmProvider, plannerStorageDir)
	agentPlanner.SetVerbose(plannerVerbose)
	
	// Initialize MCP
	mcpClient, err := initializePlannerMCP(ctx, cfg)
	if err != nil {
		fmt.Printf("⚠️  Warning: MCP initialization failed: %v\n", err)
	}
	
	// Create executor
	executor := planner.NewPlanExecutor(agentPlanner, mcpClient)
	executor.SetVerbose(plannerVerbose)
	
	// Execute or resume the plan
	if plannerResume {
		fmt.Printf("🔄 Resuming plan %s...\n", planID)
		err = executor.ResumePlan(ctx, planID)
	} else {
		fmt.Printf("🚀 Executing plan %s...\n", planID)
		err = executor.ExecutePlan(ctx, planID)
	}
	
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}
	
	// Show final progress
	progress, err := executor.GetPlanProgress(planID)
	if err == nil {
		fmt.Printf("\n📊 Final Progress: %.1f%% (%d/%d steps)\n", 
			progress.PercentComplete, progress.CompletedSteps, progress.TotalSteps)
	}
	
	return nil
}

// runAgentListPlans lists all plans
func runAgentListPlans(cmd *cobra.Command, args []string) error {
	// Set storage directory
	if plannerStorageDir == "" {
		home, _ := os.UserHomeDir()
		plannerStorageDir = filepath.Join(home, ".rago", "agents")
	}
	
	// Create planner (just for loading plans)
	agentPlanner := planner.NewAgentPlanner(nil, plannerStorageDir)
	
	// List all plans
	plans, err := agentPlanner.ListPlans()
	if err != nil {
		return fmt.Errorf("failed to list plans: %w", err)
	}
	
	if len(plans) == 0 {
		fmt.Println("No plans found.")
		return nil
	}
	
	// Display plans in a table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tGOAL\tSTATUS\tPROGRESS\tCREATED")
	fmt.Fprintln(w, "──\t────\t──────\t────────\t───────")
	
	for _, plan := range plans {
		progress := fmt.Sprintf("%d/%d", plan.CompletedSteps, plan.TotalSteps)
		created := plan.CreatedAt.Format("2006-01-02 15:04")
		
		// Truncate goal if too long
		goal := plan.Goal
		if len(goal) > 40 {
			goal = goal[:37] + "..."
		}
		
		// Truncate ID for display
		shortID := plan.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", 
			shortID, goal, plan.Status, progress, created)
	}
	
	w.Flush()
	
	fmt.Printf("\nTotal plans: %d\n", len(plans))
	
	return nil
}

// runAgentPlanStatus shows plan status
func runAgentPlanStatus(cmd *cobra.Command, args []string) error {
	planID := args[0]
	
	// Set storage directory
	if plannerStorageDir == "" {
		home, _ := os.UserHomeDir()
		plannerStorageDir = filepath.Join(home, ".rago", "agents")
	}
	
	// Create planner
	agentPlanner := planner.NewAgentPlanner(nil, plannerStorageDir)
	
	// Create executor for getting progress
	executor := planner.NewPlanExecutor(agentPlanner, nil)
	
	// Get plan progress
	progress, err := executor.GetPlanProgress(planID)
	if err != nil {
		return fmt.Errorf("failed to get plan progress: %w", err)
	}
	
	// Display status
	fmt.Printf("📋 Plan: %s\n", progress.PlanID)
	fmt.Printf("🎯 Goal: %s\n", progress.Goal)
	fmt.Printf("📊 Status: %s\n", progress.Status)
	fmt.Printf("📈 Progress: %.1f%% (%d/%d steps)\n", 
		progress.PercentComplete, progress.CompletedSteps, progress.TotalSteps)
	fmt.Printf("📌 Tasks: %d/%d completed\n", 
		progress.CompletedTasks, progress.TotalTasks)
	
	if len(progress.TaskProgress) > 0 && plannerVerbose {
		fmt.Println("\n📝 Task Details:")
		for _, task := range progress.TaskProgress {
			status := "⏳"
			switch task.Status {
			case planner.TaskStatusCompleted:
				status = "✅"
			case planner.TaskStatusFailed:
				status = "❌"
			case planner.TaskStatusInProgress:
				status = "🔄"
			case planner.TaskStatusSkipped:
				status = "⏭️"
			}
			
			fmt.Printf("  %s %s (%d/%d steps)\n", 
				status, task.Name, task.CompletedSteps, task.TotalSteps)
		}
	}
	
	return nil
}

// runAgentAutoRun creates and executes a plan
func runAgentAutoRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	goal := strings.Join(args, " ")
	
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	// Set storage directory
	if plannerStorageDir == "" {
		home, _ := os.UserHomeDir()
		plannerStorageDir = filepath.Join(home, ".rago", "agents")
	}
	
	// Create LLM provider using factory
	fact := providers.NewFactory()
	_, err = providers.DetermineProviderType(&cfg.Providers.ProviderConfigs)
	if err != nil {
		return fmt.Errorf("failed to determine provider type: %w", err)
	}
	
	providerConfig, err := providers.GetProviderConfig(&cfg.Providers.ProviderConfigs)
	if err != nil {
		return fmt.Errorf("failed to get provider config: %w", err)
	}
	
	llmProvider, err := fact.CreateLLMProvider(ctx, providerConfig)
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}
	
	// Create planner
	agentPlanner := planner.NewAgentPlanner(llmProvider, plannerStorageDir)
	agentPlanner.SetVerbose(plannerVerbose)
	
	// Initialize MCP
	mcpClient, err := initializePlannerMCP(ctx, cfg)
	if err != nil {
		fmt.Printf("⚠️  Warning: MCP initialization failed: %v\n", err)
		fmt.Println("Continuing without MCP tools...")
	} else if mcpClient != nil {
		// Set available tools for planning
		// Convert MCP tools to domain.ToolDefinition format
		mcpTools := mcpClient.GetTools()
		tools := make([]domain.ToolDefinition, 0, len(mcpTools))
		for _, tool := range mcpTools {
			tools = append(tools, domain.ToolDefinition{
				Type: "function",
				Function: domain.ToolFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  make(map[string]interface{}), // TODO: convert InputSchema
				},
			})
		}
		agentPlanner.SetMCPTools(tools)
		if plannerVerbose {
			fmt.Printf("📦 Loaded %d MCP tools for planning\n", len(tools))
		}
	}
	
	// Create the plan
	fmt.Printf("🤔 Creating plan for: %s\n", goal)
	plan, err := agentPlanner.CreatePlan(ctx, goal)
	if err != nil {
		return fmt.Errorf("failed to create plan: %w", err)
	}
	
	// Display plan summary
	fmt.Printf("\n📋 Plan created: %s\n", plan.ID)
	fmt.Printf("📝 Summary: %s\n", plan.Summary)
	fmt.Printf("📊 Tasks: %d, Total steps: %d\n", len(plan.Tasks), plan.TotalSteps)
	
	// Create executor
	executor := planner.NewPlanExecutor(agentPlanner, mcpClient)
	executor.SetVerbose(plannerVerbose)
	
	// Execute the plan immediately
	fmt.Printf("\n🚀 Executing plan...\n")
	if err := executor.ExecutePlan(ctx, plan.ID); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}
	
	// Show final progress
	progress, err := executor.GetPlanProgress(plan.ID)
	if err == nil {
		fmt.Printf("\n📊 Final Progress: %.1f%% (%d/%d steps)\n", 
			progress.PercentComplete, progress.CompletedSteps, progress.TotalSteps)
	}
	
	return nil
}

// initializePlannerMCP initializes the MCP client if configured
func initializePlannerMCP(ctx context.Context, cfg *config.Config) (*mcp.Client, error) {
	// Check if MCP servers are configured
	if cfg.MCP.Servers == nil || len(cfg.MCP.Servers) == 0 {
		return nil, nil
	}
	
	// Create RAGO client to use its MCP initialization
	ragoClient, err := client.NewWithConfig(cfg)
	if err != nil {
		return nil, err
	}
	defer ragoClient.Close()
	
	// Enable MCP
	if err := ragoClient.EnableMCP(ctx); err != nil {
		return nil, err
	}
	
	// Get the MCP client from ragoClient  
	// We'll need to get it via the internal structure
	// For now, return nil until we can properly get the MCP client
	return nil, nil
}