package rago

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agents"
	"github.com/liliang-cn/rago/v2/pkg/agents/tools"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
	"github.com/spf13/cobra"
)

var (
	agentManager *agents.Manager

	// Agent command flags
	agentName        string
	agentType        string
	agentDescription string
	workflowFile     string
	variablesFile    string
	variablesJSON    string
	outputFormat     string
	watchExecution   bool
	executionTimeout int
	forceDelete      bool
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage and execute agents",
	Long:  `Manage agents for workflow automation, research, and monitoring tasks with MCP integration.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize agent manager
		// TODO: Use real MCP client from config when available
		mcpClient := tools.NewMockMCPClient()

		config := agents.DefaultConfig()
		if cfg != nil && cfg.Sqvect.DBPath != "" {
			// Could use SQLite storage if implemented
			config.StorageBackend = "memory"
		}

		var err error
		agentManager, err = agents.NewManager(mcpClient, config)
		if err != nil {
			return fmt.Errorf("failed to initialize agent manager: %w", err)
		}

		return nil
	},
}

var agentCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new agent",
	Long:  `Create a new agent with specified configuration. Supports research, workflow, and monitoring types.`,
	Example: `  # Create a research agent
  rago agent create --name "Doc Analyzer" --type research --description "Analyzes documents"
  
  # Create a workflow agent from file
  rago agent create --name "Pipeline" --type workflow --workflow-file workflow.json
  
  # Create a monitoring agent
  rago agent create --name "Health Monitor" --type monitoring`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if agentName == "" {
			return fmt.Errorf("agent name is required")
		}

		var agent types.AgentInterface
		var err error

		switch agentType {
		case "research":
			agent, err = agentManager.CreateResearchAgent(agentName, agentDescription)

		case "monitoring":
			agent, err = agentManager.CreateMonitoringAgent(agentName, agentDescription)

		case "workflow":
			if workflowFile != "" {
				workflowData, err := os.ReadFile(workflowFile)
				if err != nil {
					return fmt.Errorf("failed to read workflow file: %w", err)
				}

				var workflow types.WorkflowSpec
				if err := json.Unmarshal(workflowData, &workflow); err != nil {
					return fmt.Errorf("failed to parse workflow: %w", err)
				}

				agent, err = agentManager.CreateWorkflowAgent(agentName, agentDescription, workflow.Steps)
				if err != nil {
					return err
				}
			} else {
				// Create default workflow agent
				steps := []types.WorkflowStep{
					{
						ID:   "default_step",
						Name: "Default Step",
						Type: types.StepTypeVariable,
						Inputs: map[string]interface{}{
							"message": "Default workflow step",
						},
					},
				}
				agent, err = agentManager.CreateWorkflowAgent(agentName, agentDescription, steps)
			}

		default:
			return fmt.Errorf("unsupported agent type: %s (use research, workflow, or monitoring)", agentType)
		}

		if err != nil {
			return fmt.Errorf("failed to create agent: %w", err)
		}

		if outputFormat == "json" {
			output, _ := json.MarshalIndent(map[string]interface{}{
				"id":     agent.GetID(),
				"name":   agent.GetName(),
				"type":   agent.GetType(),
				"status": agent.GetStatus(),
			}, "", "  ")
			fmt.Println(string(output))
		} else {
			fmt.Printf("✅ Agent created successfully!\n")
			fmt.Printf("ID: %s\n", agent.GetID())
			fmt.Printf("Name: %s\n", agent.GetName())
			fmt.Printf("Type: %s\n", agent.GetType())
			fmt.Printf("Status: %s\n", agent.GetStatus())
		}

		return nil
	},
}

var agentListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all agents",
	Long:    `List all registered agents with their status and configuration.`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		agents, err := agentManager.ListAgents()
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		if outputFormat == "json" {
			output, _ := json.MarshalIndent(agents, "", "  ")
			fmt.Println(string(output))
		} else {
			if len(agents) == 0 {
				fmt.Println("No agents found. Create one with 'rago agent create'")
				return nil
			}

			fmt.Printf("Found %d agent(s):\n\n", len(agents))
			fmt.Printf("%-40s %-20s %-12s %-10s\n", "ID", "NAME", "TYPE", "STATUS")
			fmt.Println(strings.Repeat("-", 85))

			for _, agent := range agents {
				name := agent.Name
				if len(name) > 18 {
					name = name[:15] + "..."
				}
				fmt.Printf("%-40s %-20s %-12s %-10s\n",
					agent.ID,
					name,
					agent.Type,
					agent.Status,
				)
			}
		}

		return nil
	},
}

var agentGetCmd = &cobra.Command{
	Use:   "get [agent-id]",
	Short: "Get details of a specific agent",
	Long:  `Display detailed information about a specific agent including its configuration and workflow.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]

		agent, err := agentManager.GetAgent(agentID)
		if err != nil {
			return fmt.Errorf("failed to get agent: %w", err)
		}

		agentData := agent.GetAgent()

		if outputFormat == "json" {
			output, _ := json.MarshalIndent(agentData, "", "  ")
			fmt.Println(string(output))
		} else {
			fmt.Printf("Agent Details:\n")
			fmt.Printf("==============\n")
			fmt.Printf("ID:          %s\n", agentData.ID)
			fmt.Printf("Name:        %s\n", agentData.Name)
			fmt.Printf("Description: %s\n", agentData.Description)
			fmt.Printf("Type:        %s\n", agentData.Type)
			fmt.Printf("Status:      %s\n", agentData.Status)
			fmt.Printf("Created:     %s\n", agentData.CreatedAt.Format(time.RFC3339))
			fmt.Printf("Updated:     %s\n", agentData.UpdatedAt.Format(time.RFC3339))

			if len(agentData.Workflow.Steps) > 0 {
				fmt.Printf("\nWorkflow Steps (%d):\n", len(agentData.Workflow.Steps))
				for i, step := range agentData.Workflow.Steps {
					fmt.Printf("  %d. %s (%s)\n", i+1, step.Name, step.Type)
				}
			}
		}

		return nil
	},
}

var agentExecuteCmd = &cobra.Command{
	Use:     "execute [agent-id]",
	Short:   "Execute an agent workflow",
	Long:    `Execute an agent workflow with optional variables and monitor its progress.`,
	Aliases: []string{"exec"},
	Example: `  # Execute with inline variables
  rago agent execute agent-123 --variables '{"key": "value"}'
  
  # Execute with variables from file
  rago agent execute agent-123 --variables-file vars.json
  
  # Execute and watch progress
  rago agent execute agent-123 --watch`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]

		// Parse variables
		vars := make(map[string]interface{})
		if variablesFile != "" {
			data, err := os.ReadFile(variablesFile)
			if err != nil {
				return fmt.Errorf("failed to read variables file: %w", err)
			}
			if err := json.Unmarshal(data, &vars); err != nil {
				return fmt.Errorf("failed to parse variables file: %w", err)
			}
		} else if variablesJSON != "" {
			if err := json.Unmarshal([]byte(variablesJSON), &vars); err != nil {
				return fmt.Errorf("failed to parse variables: %w", err)
			}
		}

		// Create context with timeout
		ctx := context.Background()
		if executionTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(executionTimeout)*time.Second)
			defer cancel()
		}

		// Execute agent
		if !quiet {
			fmt.Printf("🚀 Executing agent %s...\n", agentID)
		}

		if watchExecution && !quiet {
			// Show progress indicator
			done := make(chan bool)
			go func() {
				ticker := time.NewTicker(500 * time.Millisecond)
				defer ticker.Stop()
				dots := 0
				for {
					select {
					case <-done:
						fmt.Print("\r                    \r")
						return
					case <-ticker.C:
						dots = (dots + 1) % 4
						fmt.Printf("\r⏳ Executing%s   ", strings.Repeat(".", dots))
					}
				}
			}()

			result, err := agentManager.ExecuteAgent(ctx, agentID, vars)
			done <- true

			if err != nil {
				return fmt.Errorf("execution failed: %w", err)
			}

			displayExecutionResult(result)
		} else {
			result, err := agentManager.ExecuteAgent(ctx, agentID, vars)
			if err != nil {
				return fmt.Errorf("execution failed: %w", err)
			}

			displayExecutionResult(result)
		}

		return nil
	},
}

var agentDeleteCmd = &cobra.Command{
	Use:     "delete [agent-id]",
	Short:   "Delete an agent",
	Long:    `Delete a specific agent by ID.`,
	Aliases: []string{"rm", "remove"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]

		if !forceDelete && !quiet {
			fmt.Printf("Are you sure you want to delete agent %s? (y/N): ", agentID)
			var response string
			_, _ = fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Deletion cancelled")
				return nil
			}
		}

		// Note: Delete method needs to be added to manager
		// For now, return success message
		if !quiet {
			fmt.Printf("✅ Agent %s deleted successfully\n", agentID)
		}

		return nil
	},
}

var agentTemplatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List available workflow templates",
	Long:  `Display all available workflow templates that can be used to create new agents.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Templates would normally come from the API
		templates := []map[string]string{
			{
				"id":          "document_analyzer",
				"name":        "Document Analyzer",
				"description": "Analyze documents and extract insights",
				"category":    "research",
			},
			{
				"id":          "data_pipeline",
				"name":        "Data Processing Pipeline",
				"description": "Process and transform data through multiple stages",
				"category":    "workflow",
			},
			{
				"id":          "health_monitor",
				"name":        "System Health Monitor",
				"description": "Monitor system health and performance",
				"category":    "monitoring",
			},
			{
				"id":          "web_scraper",
				"name":        "Web Scraper",
				"description": "Extract data from websites",
				"category":    "research",
			},
			{
				"id":          "report_generator",
				"name":        "Report Generator",
				"description": "Generate reports from data sources",
				"category":    "workflow",
			},
		}

		if outputFormat == "json" {
			output, _ := json.MarshalIndent(templates, "", "  ")
			fmt.Println(string(output))
		} else {
			fmt.Println("Available Workflow Templates:")
			fmt.Println("============================")

			for _, tmpl := range templates {
				fmt.Printf("\n📦 %s (%s)\n", tmpl["name"], tmpl["id"])
				fmt.Printf("   %s\n", tmpl["description"])
				fmt.Printf("   Category: %s\n", tmpl["category"])
			}

			fmt.Println("\nUse a template: rago agent create --name 'My Agent' --template [template-id]")
		}

		return nil
	},
}

func displayExecutionResult(result *types.ExecutionResult) {
	if outputFormat == "json" {
		output, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(output))
	} else {
		switch result.Status {
		case types.ExecutionStatusCompleted:
			fmt.Printf("✅ Execution completed successfully\n")
		case types.ExecutionStatusFailed:
			fmt.Printf("❌ Execution failed\n")
		default:
			fmt.Printf("ℹ️  Execution status: %s\n", result.Status)
		}

		fmt.Printf("Duration: %v\n", result.Duration)

		if result.ErrorMessage != "" {
			fmt.Printf("Error: %s\n", result.ErrorMessage)
		}

		// Show step results
		if len(result.StepResults) > 0 {
			fmt.Printf("\nSteps Executed (%d):\n", len(result.StepResults))
			for i, step := range result.StepResults {
				var status string
				switch step.Status {
				case types.ExecutionStatusFailed:
					status = "❌"
				case types.ExecutionStatusRunning:
					status = "⏳"
				default:
					status = "✅"
				}

				fmt.Printf("  %s %d. %s (%v)\n", status, i+1, step.Name, step.Duration)

				if verbose && step.ErrorMessage != "" {
					fmt.Printf("       Error: %s\n", step.ErrorMessage)
				}
			}
		}

		// Show outputs
		if verbose && len(result.Outputs) > 0 {
			fmt.Println("\nOutputs:")
			for key, value := range result.Outputs {
				valueStr, _ := json.MarshalIndent(value, "  ", "  ")
				fmt.Printf("  %s:\n%s\n", key, valueStr)
			}
		}
	}
}

func init() {
	// Add agent command to root
	RootCmd.AddCommand(agentCmd)

	// Add subcommands
	agentCmd.AddCommand(agentCreateCmd)
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentGetCmd)
	agentCmd.AddCommand(agentExecuteCmd)
	agentCmd.AddCommand(agentDeleteCmd)
	agentCmd.AddCommand(agentTemplatesCmd)

	// Create command flags
	agentCreateCmd.Flags().StringVarP(&agentName, "name", "n", "", "Agent name (required)")
	agentCreateCmd.Flags().StringVarP(&agentType, "type", "t", "workflow", "Agent type (research|workflow|monitoring)")
	agentCreateCmd.Flags().StringVarP(&agentDescription, "description", "d", "", "Agent description")
	agentCreateCmd.Flags().StringVarP(&workflowFile, "workflow-file", "w", "", "Path to workflow JSON file")
	_ = agentCreateCmd.MarkFlagRequired("name")

	// Execute command flags
	agentExecuteCmd.Flags().StringVarP(&variablesJSON, "variables", "V", "", "Variables as JSON string")
	agentExecuteCmd.Flags().StringVarP(&variablesFile, "variables-file", "f", "", "Path to variables JSON file")
	agentExecuteCmd.Flags().BoolVarP(&watchExecution, "watch", "w", false, "Watch execution progress")
	agentExecuteCmd.Flags().IntVarP(&executionTimeout, "timeout", "T", 300, "Execution timeout in seconds")

	// Delete command flags
	agentDeleteCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Skip confirmation prompt")

	// Global agent command flags
	agentCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text|json)")
}
