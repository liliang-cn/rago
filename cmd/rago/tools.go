package rago

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liliang-cn/rago/internal/chunker"
	"github.com/liliang-cn/rago/internal/processor"
	"github.com/liliang-cn/rago/internal/store"
	"github.com/liliang-cn/rago/internal/tools"
	"github.com/spf13/cobra"
)

var (
	toolsListEnabled bool
	toolsOutputJSON  bool
	toolExecuteArgs  []string
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Manage and interact with tools",
	Long:  `List available tools, get tool information, and execute tools directly.`,
}

var toolsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available tools",
	Long:  `Display all available tools with their descriptions and parameters.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		processor, cleanup, err := initializeProcessor()
		if err != nil {
			return err
		}
		defer cleanup()

		registry := processor.GetToolRegistry()
		if registry == nil {
			fmt.Println("Tools are not enabled in the configuration")
			return nil
		}

		var toolInfos []tools.ToolInfo
		if toolsListEnabled {
			toolInfos = registry.ListEnabled()
		} else {
			toolInfos = registry.List()
		}

		if toolsOutputJSON {
			data, err := json.MarshalIndent(toolInfos, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal tools to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Available tools (%d):\n\n", len(toolInfos))
		for _, toolInfo := range toolInfos {
			status := "✗ Disabled"
			if toolInfo.Enabled {
				status = "✓ Enabled"
			}

			fmt.Printf("• %s [%s]\n", toolInfo.Name, status)
			fmt.Printf("  %s\n\n", toolInfo.Description)
		}

		return nil
	},
}

var toolsDescribeCmd = &cobra.Command{
	Use:   "describe <tool-name>",
	Short: "Get detailed information about a specific tool",
	Long:  `Display detailed information about a tool including its parameters and usage.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		processor, cleanup, err := initializeProcessor()
		if err != nil {
			return err
		}
		defer cleanup()

		registry := processor.GetToolRegistry()
		if registry == nil {
			return fmt.Errorf("tools are not enabled in the configuration")
		}

		toolName := args[0]
		tool, exists := registry.Get(toolName)
		if !exists {
			return fmt.Errorf("tool '%s' not found", toolName)
		}

		toolInfo := map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters":  tool.Parameters(),
			"enabled":     registry.IsEnabled(toolName),
		}

		if toolsOutputJSON {
			data, err := json.MarshalIndent(toolInfo, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal tool info to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Tool: %s\n", tool.Name())
		fmt.Printf("Description: %s\n", tool.Description())
		fmt.Printf("Enabled: %v\n\n", registry.IsEnabled(toolName))

		params := tool.Parameters()
		fmt.Println("Parameters:")
		fmt.Printf("  Type: %s\n", params.Type)
		
		if len(params.Required) > 0 {
			fmt.Printf("  Required: %s\n", strings.Join(params.Required, ", "))
		}

		if len(params.Properties) > 0 {
			fmt.Println("  Properties:")
			for name, param := range params.Properties {
				fmt.Printf("    %s (%s): %s\n", name, param.Type, param.Description)
				if param.Default != nil {
					fmt.Printf("      Default: %v\n", param.Default)
				}
				if param.Enum != nil && len(param.Enum) > 0 {
					fmt.Printf("      Allowed values: %v\n", param.Enum)
				}
			}
		}

		return nil
	},
}

var toolsExecuteCmd = &cobra.Command{
	Use:   "execute <tool-name>",
	Short: "Execute a tool with specified arguments",
	Long: `Execute a tool directly with the provided arguments.
Arguments should be provided in key=value format.

Example:
  rago tools execute datetime action=now
  rago tools execute file_operations action=read path=./README.md
  rago tools execute rag_search query="machine learning" top_k=3`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		processor, cleanup, err := initializeProcessor()
		if err != nil {
			return err
		}
		defer cleanup()

		registry := processor.GetToolRegistry()
		executor := processor.GetToolExecutor()
		if registry == nil || executor == nil {
			return fmt.Errorf("tools are not enabled in the configuration")
		}

		toolName := args[0]
		tool, exists := registry.Get(toolName)
		if !exists {
			return fmt.Errorf("tool '%s' not found", toolName)
		}

		if !registry.IsEnabled(toolName) {
			return fmt.Errorf("tool '%s' is disabled", toolName)
		}

		// Parse arguments
		toolArgs := make(map[string]interface{})
		for _, arg := range args[1:] {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid argument format: %s (expected key=value)", arg)
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			
			// Try to parse as JSON first, then fallback to string
			var parsedValue interface{}
			if err := json.Unmarshal([]byte(value), &parsedValue); err != nil {
				parsedValue = value
			}
			toolArgs[key] = parsedValue
		}

		// Parse additional args from flag
		for _, arg := range toolExecuteArgs {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				
				var parsedValue interface{}
				if err := json.Unmarshal([]byte(value), &parsedValue); err != nil {
					parsedValue = value
				}
				toolArgs[key] = parsedValue
			}
		}

		// Validate arguments
		if err := tool.Validate(toolArgs); err != nil {
			return fmt.Errorf("invalid arguments: %w", err)
		}

		// Execute tool
		ctx := context.Background()
		execCtx := &tools.ExecutionContext{
			RequestID: "cli-" + toolName,
			UserID:    "cli-user",
			SessionID: "cli-session",
		}

		fmt.Printf("Executing tool '%s'...\n", toolName)
		result, err := executor.Execute(ctx, execCtx, toolName, toolArgs)
		if err != nil {
			return fmt.Errorf("tool execution failed: %w", err)
		}

		if toolsOutputJSON {
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("\nResult:\n")
		fmt.Printf("Success: %v\n", result.Success)
		if result.Error != "" {
			fmt.Printf("Error: %s\n", result.Error)
		}
		if result.Data != nil {
			fmt.Printf("Data: %v\n", result.Data)
		}

		return nil
	},
}

var toolsStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show tool execution statistics",
	Long:  `Display statistics about tool usage and performance.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		processor, cleanup, err := initializeProcessor()
		if err != nil {
			return err
		}
		defer cleanup()

		registry := processor.GetToolRegistry()
		executor := processor.GetToolExecutor()
		if registry == nil || executor == nil {
			return fmt.Errorf("tools are not enabled in the configuration")
		}

		registryStats := registry.Stats()
		executorStats := executor.GetStats()

		if toolsOutputJSON {
			stats := map[string]interface{}{
				"registry": registryStats,
				"executor": executorStats,
			}
			data, err := json.MarshalIndent(stats, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal stats to JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Println("Tool Registry Statistics:")
		fmt.Printf("  Total tools: %v\n", registryStats["total_tools"])
		fmt.Printf("  Enabled tools: %v\n", registryStats["enabled_tools"])
		fmt.Printf("  Disabled tools: %v\n", registryStats["disabled_tools"])

		fmt.Println("\nTool Executor Statistics:")
		fmt.Printf("  Total executions: %v\n", executorStats["total_executions"])
		fmt.Printf("  Successful executions: %v\n", executorStats["successful_executions"])
		fmt.Printf("  Failed executions: %v\n", executorStats["failed_executions"])
		fmt.Printf("  Currently running: %v\n", executorStats["running_executions"])

		return nil
	},
}

func initializeProcessor() (*processor.Service, func(), error) {
	// Initialize stores
	vectorStore, err := store.NewSQLiteStore(
		cfg.Sqvect.DBPath,
		cfg.Sqvect.VectorDim,
		cfg.Sqvect.MaxConns,
		cfg.Sqvect.BatchSize,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	keywordStore, err := store.NewKeywordStore(cfg.Keyword.IndexPath)
	if err != nil {
		vectorStore.Close()
		return nil, nil, fmt.Errorf("failed to create keyword store: %w", err)
	}

	docStore := store.NewDocumentStore(vectorStore.GetSqvectStore())

	// Initialize services using shared provider system
	ctx := context.Background()
	embedService, llmService, metadataExtractor, err := initializeProviders(ctx, cfg)
	if err != nil {
		vectorStore.Close()
		keywordStore.Close()
		return nil, nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	chunkerService := chunker.New()

	processor := processor.New(
		embedService,
		llmService,
		chunkerService,
		vectorStore,
		keywordStore,
		docStore,
		cfg,
		metadataExtractor,
	)

	cleanup := func() {
		vectorStore.Close()
		keywordStore.Close()
	}

	return processor, cleanup, nil
}

func init() {
	// Add subcommands
	toolsCmd.AddCommand(toolsListCmd)
	toolsCmd.AddCommand(toolsDescribeCmd)
	toolsCmd.AddCommand(toolsExecuteCmd)
	toolsCmd.AddCommand(toolsStatsCmd)

	// Global flags
	toolsCmd.PersistentFlags().BoolVar(&toolsOutputJSON, "json", false, "output in JSON format")

	// List command flags
	toolsListCmd.Flags().BoolVar(&toolsListEnabled, "enabled-only", false, "show only enabled tools")

	// Execute command flags
	toolsExecuteCmd.Flags().StringSliceVar(&toolExecuteArgs, "arg", []string{}, "tool arguments in key=value format (can be used multiple times)")
}