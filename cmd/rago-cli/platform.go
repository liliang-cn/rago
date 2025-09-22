package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/liliang-cn/rago/v2/client"
	"github.com/spf13/cobra"
)

var (
	platformCmd = &cobra.Command{
		Use:   "platform",
		Short: "RAGO AI Platform - Unified AI operations",
		Long: `The RAGO platform provides unified access to all AI capabilities:
  • LLM - Language model operations  
  • RAG - Retrieval-augmented generation
  • Tools - External tool integration (MCP)
  • Agent - Autonomous task execution`,
		Aliases: []string{"p"},
	}

	// Platform-wide flags
	temperature float64
	maxTokens   int
	topK        int
)

// llmGenerateCmd - Generate text using the unified API
var llmGenerateCmd = &cobra.Command{
	Use:   "generate [prompt]",
	Short: "Generate text using LLM",
	Long:  `Generate text from a prompt using the configured language model.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := strings.Join(args, " ")
		
		// Initialize RAGO platform
		rago, err := client.New(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to initialize RAGO: %w", err)
		}
		defer rago.Close()

		// Use the unified LLM API
		if verbose {
			fmt.Printf("Generating with prompt: %s\n", prompt)
		}

		var response string
		if llmStream {
			fmt.Print("Response: ")
			err = rago.LLM.StreamWithOptions(
				context.Background(),
				prompt,
				func(token string) {
					fmt.Print(token)
				},
				&client.GenerateOptions{
					Temperature: temperature,
					MaxTokens:   maxTokens,
				},
			)
			fmt.Println()
		} else {
			response, err = rago.LLM.GenerateWithOptions(
				context.Background(),
				prompt,
				&client.GenerateOptions{
					Temperature: temperature,
					MaxTokens:   maxTokens,
				},
			)
			if err == nil {
				fmt.Println(response)
			}
		}

		return err
	},
}

// ragQueryCmd - Query using RAG
var ragQueryCmd = &cobra.Command{
	Use:   "query [question]",
	Short: "Query knowledge base using RAG",
	Long:  `Query the knowledge base using retrieval-augmented generation.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")
		
		// Initialize RAGO platform
		rago, err := client.New(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to initialize RAGO: %w", err)
		}
		defer rago.Close()

		// Use the unified RAG API
		response, err := rago.RAG.QueryWithOptions(
			context.Background(),
			query,
			&client.QueryOptions{
				TopK:        topK,
				Temperature: temperature,
				MaxTokens:   maxTokens,
				ShowSources: verbose,
			},
		)
		if err != nil {
			return err
		}

		fmt.Printf("Answer: %s\n", response.Answer)
		
		if verbose && len(response.Sources) > 0 {
			fmt.Println("\nSources:")
			for i, source := range response.Sources {
				fmt.Printf("  [%d] %s (score: %.2f)\n", 
					i+1, source.ID, source.Score)
			}
		}

		return nil
	},
}

// toolsListCmd - List available tools
var toolsListCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available MCP tools",
	Long:  `List all available tools from configured MCP servers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize RAGO platform
		rago, err := client.New(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to initialize RAGO: %w", err)
		}
		defer rago.Close()

		// Enable MCP if needed
		if err := rago.EnableMCP(context.Background()); err != nil {
			fmt.Printf("Warning: MCP not fully enabled: %v\n", err)
		}

		// Use the unified Tools API
		tools, err := rago.Tools.List()
		if err != nil {
			return err
		}

		if len(tools) == 0 {
			fmt.Println("No tools available. Configure MCP servers in mcpServers.json")
			return nil
		}

		fmt.Printf("Available tools (%d):\n", len(tools))
		for _, tool := range tools {
			fmt.Printf("  • %s: %s\n", tool.Name, tool.Description)
		}

		return nil
	},
}

// agentRunCmd - Run an agent task
var agentRunCmd = &cobra.Command{
	Use:   "run [task]",
	Short: "Run a task using the AI agent",
	Long:  `Execute a task using the autonomous agent with natural language instructions.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		task := strings.Join(args, " ")
		
		// Initialize RAGO platform
		rago, err := client.New(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to initialize RAGO: %w", err)
		}
		defer rago.Close()

		// Enable MCP for agent tools
		if err := rago.EnableMCP(context.Background()); err != nil {
			fmt.Printf("Warning: MCP not fully enabled: %v\n", err)
		}

		fmt.Printf("Task: %s\n", task)
		fmt.Println("Executing...")

		// Use the unified Agent API
		result, err := rago.Agent.RunWithOptions(
			context.Background(),
			task,
			&client.AgentOptions{
				Verbose: verbose,
			},
		)
		if err != nil {
			return err
		}

		if result.Success {
			fmt.Println("✓ Task completed successfully")
			if result.Output != nil {
				fmt.Printf("Output: %v\n", result.Output)
			}
		} else {
			fmt.Printf("✗ Task failed: %s\n", result.Error)
		}

		// Show execution steps if verbose
		if verbose && len(result.Steps) > 0 {
			fmt.Println("\nExecution steps:")
			for i, step := range result.Steps {
				status := "✓"
				if !step.Success {
					status = "✗"
				}
				fmt.Printf("  %s Step %d: %s\n", status, i+1, step.Name)
			}
		}

		return nil
	},
}

// demoCmd - Run platform demo
var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Run a demonstration of all platform capabilities",
	Long:  `Run an interactive demonstration showcasing all four RAGO platform components.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize RAGO platform
		rago, err := client.New(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to initialize RAGO: %w", err)
		}
		defer rago.Close()

		fmt.Println("=== RAGO Platform Demo ===")
		fmt.Println("Demonstrating unified AI capabilities\n")

		// 1. LLM Demo
		fmt.Println("1. LLM Generation")
		fmt.Println("-----------------")
		response, err := rago.LLM.Generate("What is Go programming language in one sentence?")
		if err != nil {
			fmt.Printf("LLM error: %v\n", err)
		} else {
			fmt.Printf("Response: %s\n\n", response)
		}

		// 2. RAG Demo (if documents exist)
		fmt.Println("2. RAG Knowledge")
		fmt.Println("----------------")
		
		// Try to ingest README if it exists
		if _, err := os.Stat("README.md"); err == nil {
			err = rago.RAG.Ingest("README.md")
			if err != nil {
				fmt.Printf("Ingestion error: %v\n", err)
			} else {
				answer, err := rago.RAG.Query("What is RAGO?")
				if err != nil {
					fmt.Printf("Query error: %v\n", err)
				} else {
					fmt.Printf("Answer: %s\n\n", answer)
				}
			}
		} else {
			fmt.Println("No documents to ingest\n")
		}

		// 3. Tools Demo
		fmt.Println("3. MCP Tools")
		fmt.Println("------------")
		
		// Enable MCP
		if err := rago.EnableMCP(context.Background()); err != nil {
			fmt.Printf("MCP not available: %v\n\n", err)
		} else {
			tools, err := rago.Tools.List()
			if err != nil {
				fmt.Printf("Tools error: %v\n", err)
			} else {
				fmt.Printf("Available tools: %d\n", len(tools))
				for i, tool := range tools {
					if i >= 3 {
						fmt.Println("  ...")
						break
					}
					fmt.Printf("  • %s\n", tool.Name)
				}
				fmt.Println()
			}
		}

		// 4. Agent Demo
		fmt.Println("4. Agent Automation")
		fmt.Println("-------------------")
		
		task := "List files in the current directory"
		fmt.Printf("Task: %s\n", task)
		
		result, err := rago.Agent.Run(task)
		if err != nil {
			fmt.Printf("Agent error: %v\n", err)
		} else {
			if result.Success {
				fmt.Println("✓ Task completed")
				fmt.Printf("Output: %v\n", result.Output)
			} else {
				fmt.Printf("✗ Task failed: %s\n", result.Error)
			}
		}

		fmt.Println("\n=== Demo Complete ===")
		fmt.Println("Use 'rago platform --help' to explore more commands")

		return nil
	},
}

func init() {
	// Add platform command to root
	RootCmd.AddCommand(platformCmd)

	// Add subcommands
	platformCmd.AddCommand(llmGenerateCmd)
	platformCmd.AddCommand(ragQueryCmd)
	platformCmd.AddCommand(toolsListCmd)
	platformCmd.AddCommand(agentRunCmd)
	platformCmd.AddCommand(demoCmd)

	// LLM generate flags
	llmGenerateCmd.Flags().Float64VarP(&temperature, "temperature", "t", 0.7, "generation temperature (0.0-1.0)")
	llmGenerateCmd.Flags().IntVarP(&maxTokens, "max-tokens", "m", 0, "maximum tokens to generate")
	llmGenerateCmd.Flags().BoolVarP(&llmStream, "stream", "s", false, "stream the response")

	// RAG query flags
	ragQueryCmd.Flags().IntVarP(&topK, "top-k", "k", 5, "number of documents to retrieve")
	ragQueryCmd.Flags().Float64VarP(&temperature, "temperature", "t", 0.7, "generation temperature")
	ragQueryCmd.Flags().IntVarP(&maxTokens, "max-tokens", "m", 0, "maximum tokens to generate")

	// Agent run flags are inherited from global verbose flag
}