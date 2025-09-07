package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/liliang-cn/rago/v2/pkg/client"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP pillar operations",
	Long:  "Manage MCP tools and definitions",
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available MCP tools",
	RunE:  runMCPList,
}

var mcpSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for MCP tools",
	Args:  cobra.ExactArgs(1),
	RunE:  runMCPSearch,
}

var mcpInfoCmd = &cobra.Command{
	Use:   "info [tool-name]",
	Short: "Show detailed information about a specific tool",
	Args:  cobra.ExactArgs(1),
	RunE:  runMCPInfo,
}

func init() {
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpSearchCmd)
	mcpCmd.AddCommand(mcpInfoCmd)
	
	// Add flags
	mcpListCmd.Flags().BoolP("json", "j", false, "Output in JSON format")
	mcpListCmd.Flags().StringP("category", "c", "", "Filter by category")
	mcpInfoCmd.Flags().BoolP("json", "j", false, "Output in JSON format")
}

func runMCPList(cmd *cobra.Command, args []string) error {
	// Check if JSON output is requested early
	jsonOutput, _ := cmd.Flags().GetBool("json")
	
	if !jsonOutput {
		fmt.Println("🔧 RAGO MCP Tools")
		fmt.Println("==================")
	}

	coreConfig, err := loadCoreConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	client, err := client.NewWithConfig(coreConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Get all tools
	tools := client.MCP().GetTools()
	
	// Check for category filter
	category, _ := cmd.Flags().GetString("category")
	if category != "" {
		tools = client.MCP().GetToolsByCategory(category)
		if !jsonOutput {
			fmt.Printf("\nFiltered by category: %s\n", category)
		}
	}
	
	if !jsonOutput {
		fmt.Printf("\nFound %d tools:\n\n", len(tools))
	}
	if jsonOutput {
		output, err := json.MarshalIndent(tools, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal tools: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Group tools by server
	toolsByServer := make(map[string][]string)
	for _, tool := range tools {
		serverName := tool.ServerName
		if serverName == "" {
			serverName = "default"
		}
		toolsByServer[serverName] = append(toolsByServer[serverName], tool.Name)
	}

	// Display tools grouped by server
	for server, serverTools := range toolsByServer {
		fmt.Printf("📦 %s:\n", server)
		for _, toolName := range serverTools {
			// Find the tool to get its description
			for _, tool := range tools {
				if tool.Name == toolName {
					fmt.Printf("  • %s - %s\n", tool.Name, tool.Description)
					break
				}
			}
		}
		fmt.Println()
	}

	// Show summary
	fmt.Printf("Total: %d tools from %d sources\n", len(tools), len(toolsByServer))
	
	return nil
}

func runMCPSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	
	fmt.Printf("🔍 Searching for MCP tools matching: %s\n", query)
	fmt.Println(strings.Repeat("=", 50))

	coreConfig, err := loadCoreConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	client, err := client.NewWithConfig(coreConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Search for tools
	tools := client.MCP().SearchTools(query)
	
	if len(tools) == 0 {
		fmt.Printf("\nNo tools found matching '%s'\n", query)
		return nil
	}
	
	fmt.Printf("\nFound %d matching tools:\n\n", len(tools))
	
	for _, tool := range tools {
		fmt.Printf("• %s (%s)\n", tool.Name, tool.ServerName)
		fmt.Printf("  %s\n\n", tool.Description)
	}
	
	return nil
}

func runMCPInfo(cmd *cobra.Command, args []string) error {
	toolName := args[0]
	
	fmt.Printf("🔧 Tool Information: %s\n", toolName)
	fmt.Println(strings.Repeat("=", 50))

	coreConfig, err := loadCoreConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	client, err := client.NewWithConfig(coreConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	// Find the tool
	tool, found := client.MCP().FindTool(toolName)
	if !found {
		return fmt.Errorf("tool '%s' not found", toolName)
	}
	
	// Check if JSON output is requested
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		output, err := json.MarshalIndent(tool, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal tool: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}
	
	// Display tool information
	fmt.Printf("\nName:        %s\n", tool.Name)
	fmt.Printf("Server:      %s\n", tool.ServerName)
	fmt.Printf("Description: %s\n", tool.Description)
	if tool.Category != "" {
		fmt.Printf("Category:    %s\n", tool.Category)
	}
	
	// Display input schema
	fmt.Printf("\nInput Schema:\n")
	schemaJSON, err := json.MarshalIndent(tool.InputSchema, "  ", "  ")
	if err != nil {
		fmt.Printf("  Error formatting schema: %v\n", err)
	} else {
		fmt.Printf("  %s\n", string(schemaJSON))
	}
	
	return nil
}