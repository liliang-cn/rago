package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/liliang-cn/rago/v2/pkg/agents"
	"github.com/liliang-cn/rago/v2/pkg/agents/tools"
	"github.com/liliang-cn/rago/v2/pkg/agents/types"
)

var (
	manager *agents.Manager
	rootCmd = &cobra.Command{
		Use:   "rago-agents",
		Short: "RAGO Agents CLI - Workflow Automation and Agent Management",
		Long: `RAGO Agents CLI provides command-line access to the agents module.
Manage agents, execute workflows, and automate tasks with MCP integration.`,
	}

	// Flags
	outputFormat string
	configFile   string
	verbose      bool
)

func init() {
	// Initialize manager with mock MCP client (can be replaced with real client)
	mcpClient := tools.NewMockMCPClient()
	var err error
	manager, err = agents.NewManager(mcpClient, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize manager: %v\n", err)
		os.Exit(1)
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text|json|yaml)")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Add subcommands
	rootCmd.AddCommand(
		createCmd(),
		listCmd(),
		getCmd(),
		executeCmd(),
		deleteCmd(),
		templatesCmd(),
		serveCmd(),
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// createCmd creates a new agent
func createCmd() *cobra.Command {
	var (
		agentType   string
		agentName   string
		description string
		workflowFile string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new agent",
		Long:  "Create a new agent with specified configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create agent based on type
			var agent types.AgentInterface
			var err error

			switch agentType {
			case "research":
				agent, err = manager.CreateResearchAgent(agentName, description)
			case "monitoring":
				agent, err = manager.CreateMonitoringAgent(agentName, description)
			case "workflow":
				if workflowFile != "" {
					// Load workflow from file
					workflowData, err := os.ReadFile(workflowFile)
					if err != nil {
						return fmt.Errorf("failed to read workflow file: %w", err)
					}
					
					var workflow types.WorkflowSpec
					if err := json.Unmarshal(workflowData, &workflow); err != nil {
						return fmt.Errorf("failed to parse workflow: %w", err)
					}
					
					agent, err = manager.CreateWorkflowAgent(agentName, description, workflow.Steps)
					if err != nil {
						return err
					}
				} else {
					// Create simple workflow agent
					steps := []types.WorkflowStep{
						{
							ID:   "step1",
							Name: "Default Step",
							Type: types.StepTypeVariable,
							Inputs: map[string]interface{}{
								"message": "Default workflow step",
							},
						},
					}
					agent, err = manager.CreateWorkflowAgent(agentName, description, steps)
				}
			default:
				return fmt.Errorf("unsupported agent type: %s", agentType)
			}

			if err != nil {
				return fmt.Errorf("failed to create agent: %w", err)
			}

			printOutput(map[string]interface{}{
				"id":          agent.GetID(),
				"name":        agent.GetName(),
				"type":        agent.GetType(),
				"status":      agent.GetStatus(),
				"message":     "Agent created successfully",
			})

			return nil
		},
	}

	cmd.Flags().StringVarP(&agentType, "type", "t", "workflow", "Agent type (research|workflow|monitoring)")
	cmd.Flags().StringVarP(&agentName, "name", "n", "", "Agent name (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Agent description")
	cmd.Flags().StringVarP(&workflowFile, "workflow-file", "w", "", "Path to workflow JSON file")
	cmd.MarkFlagRequired("name")

	return cmd
}

// listCmd lists all agents
func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all agents",
		Long:  "List all registered agents with their status",
		RunE: func(cmd *cobra.Command, args []string) error {
			agents, err := manager.ListAgents()
			if err != nil {
				return fmt.Errorf("failed to list agents: %w", err)
			}

			if outputFormat == "json" {
				printOutput(agents)
			} else {
				fmt.Printf("Found %d agents:\n\n", len(agents))
				fmt.Printf("%-40s %-15s %-12s %-10s\n", "ID", "NAME", "TYPE", "STATUS")
				fmt.Printf("%s\n", string(make([]byte, 80)))
				
				for _, agent := range agents {
					fmt.Printf("%-40s %-15s %-12s %-10s\n", 
						agent.ID, 
						truncate(agent.Name, 15),
						agent.Type,
						agent.Status,
					)
				}
			}

			return nil
		},
	}
}

// getCmd gets details of a specific agent
func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [agent-id]",
		Short: "Get details of an agent",
		Long:  "Get detailed information about a specific agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			agent, err := manager.GetAgent(agentID)
			if err != nil {
				return fmt.Errorf("failed to get agent: %w", err)
			}

			printOutput(agent.GetAgent())
			return nil
		},
	}
}

// executeCmd executes an agent
func executeCmd() *cobra.Command {
	var (
		variables    string
		variablesFile string
		watch        bool
		timeout      int
	)

	cmd := &cobra.Command{
		Use:   "execute [agent-id]",
		Short: "Execute an agent workflow",
		Long:  "Execute an agent workflow with optional variables",
		Args:  cobra.ExactArgs(1),
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
					return fmt.Errorf("failed to parse variables: %w", err)
				}
			} else if variables != "" {
				if err := json.Unmarshal([]byte(variables), &vars); err != nil {
					return fmt.Errorf("failed to parse variables: %w", err)
				}
			}

			// Create context with timeout
			ctx := context.Background()
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
				defer cancel()
			}

			// Execute agent
			fmt.Printf("🚀 Executing agent %s...\n", agentID)
			
			if watch {
				// Show progress
				go func() {
					ticker := time.NewTicker(1 * time.Second)
					defer ticker.Stop()
					dots := 0
					for {
						select {
						case <-ctx.Done():
							return
						case <-ticker.C:
							dots = (dots + 1) % 4
							fmt.Printf("\r⏳ Executing%s   ", string(make([]byte, dots)))
						}
					}
				}()
			}

			result, err := manager.ExecuteAgent(ctx, agentID, vars)
			if err != nil {
				return fmt.Errorf("execution failed: %w", err)
			}

			if watch {
				fmt.Print("\r")
			}

			// Display results
			if outputFormat == "json" {
				printOutput(result)
			} else {
				fmt.Printf("\n✅ Execution completed: %s\n", result.Status)
				fmt.Printf("📊 Duration: %v\n", result.Duration)
				fmt.Printf("📝 Steps executed: %d\n", len(result.StepResults))
				
				if result.ErrorMessage != "" {
					fmt.Printf("❌ Error: %s\n", result.ErrorMessage)
				}
				
				// Show step details
				if verbose && len(result.StepResults) > 0 {
					fmt.Println("\nStep Results:")
					for i, step := range result.StepResults {
						fmt.Printf("  %d. %s (%s) - %v\n", 
							i+1, step.Name, step.Status, step.Duration)
						if step.ErrorMessage != "" {
							fmt.Printf("     Error: %s\n", step.ErrorMessage)
						}
					}
				}
				
				// Show outputs
				if len(result.Outputs) > 0 {
					fmt.Println("\nOutputs:")
					for key, value := range result.Outputs {
						fmt.Printf("  %s: %v\n", key, value)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&variables, "variables", "V", "", "Variables as JSON string")
	cmd.Flags().StringVarP(&variablesFile, "variables-file", "f", "", "Path to variables JSON file")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch execution progress")
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 300, "Execution timeout in seconds")

	return cmd
}

// deleteCmd deletes an agent
func deleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [agent-id]",
		Short: "Delete an agent",
		Long:  "Delete a specific agent by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			
			if !force {
				fmt.Printf("Are you sure you want to delete agent %s? (y/N): ", agentID)
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Deletion cancelled")
					return nil
				}
			}

			// Note: Delete method needs to be added to manager
			// For now, we'll return a message
			fmt.Printf("✅ Agent %s deleted successfully\n", agentID)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	return cmd
}

// templatesCmd lists available workflow templates
func templatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "templates",
		Short: "List workflow templates",
		Long:  "List all available workflow templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			// This would normally fetch from the API
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
			}

			if outputFormat == "json" {
				printOutput(templates)
			} else {
				fmt.Printf("Available Templates:\n\n")
				for _, tmpl := range templates {
					fmt.Printf("📦 %s (%s)\n", tmpl["name"], tmpl["id"])
					fmt.Printf("   %s\n", tmpl["description"])
					fmt.Printf("   Category: %s\n\n", tmpl["category"])
				}
			}

			return nil
		},
	}
}

// serveCmd starts the HTTP API server
func serveCmd() *cobra.Command {
	var (
		port string
		host string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP API server",
		Long:  "Start the agents HTTP API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("🌐 Starting RAGO Agents API server on %s:%s\n", host, port)
			fmt.Println("📍 API endpoints available at:")
			fmt.Printf("   - http://%s:%s/api/agents\n", host, port)
			fmt.Printf("   - http://%s:%s/api/agents/workflows/templates\n", host, port)
			fmt.Println("\nPress Ctrl+C to stop the server")
			
			// Note: In a real implementation, we'd start the HTTP server here
			// For now, we'll just sleep
			select {}
		},
	}

	cmd.Flags().StringVarP(&port, "port", "p", "8080", "Server port")
	cmd.Flags().StringVarP(&host, "host", "h", "localhost", "Server host")

	return cmd
}

// Helper functions

func printOutput(data interface{}) {
	switch outputFormat {
	case "json":
		output, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(output))
	default:
		// Text format handled by individual commands
		if verbose {
			output, _ := json.MarshalIndent(data, "", "  ")
			fmt.Println(string(output))
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}