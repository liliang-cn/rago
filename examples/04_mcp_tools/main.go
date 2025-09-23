// Example: MCP Tools Integration
// This example demonstrates how to use external tools via Model Context Protocol

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/liliang-cn/rago/v2/client"
)

func main() {
	// Initialize client
	configPath := filepath.Join(os.Getenv("HOME"), ".rago", "rago.toml")
	c, err := client.New(configPath)
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Example 1: Enable MCP
	fmt.Println("=== Example 1: Enable MCP ===")
	err = c.EnableMCP(ctx)
	if err != nil {
		fmt.Printf("Note: MCP not available: %v\n", err)
		fmt.Println("Make sure MCP servers are configured in mcpServers.json")
	} else {
		fmt.Println("✓ MCP enabled successfully")
	}

	// Example 2: List available tools
	fmt.Println("\n=== Example 2: List Available Tools ===")
	if c.Tools != nil {
		tools, err := c.Tools.List()
		if err != nil {
			log.Printf("Failed to list tools: %v\n", err)
		} else {
			fmt.Printf("Found %d tools:\n", len(tools))
			for i, tool := range tools {
				fmt.Printf("  [%d] %s\n", i+1, tool.Name)
				fmt.Printf("      Description: %s\n", tool.Description)
				if tool.Parameters != nil {
					fmt.Printf("      Parameters: %v\n", tool.Parameters)
				}
			}
		}

		// If no tools found, provide guidance
		if len(tools) == 0 {
			fmt.Println("\nNo tools available. To add tools:")
			fmt.Println("1. Install MCP servers (e.g., npm install -g @modelcontextprotocol/server-filesystem)")
			fmt.Println("2. Configure them in ~/.rago/mcpServers.json")
			fmt.Println("3. Restart the application")
		}
	} else {
		fmt.Println("Tools wrapper not initialized")
	}

	// Example 3: Call a specific tool (if available)
	fmt.Println("\n=== Example 3: Call a Tool ===")
	if c.Tools != nil {
		// Example: filesystem read_file tool
		// This assumes you have the filesystem MCP server configured
		toolName := "mcp_filesystem_read_file"
		args := map[string]interface{}{
			"path": "README.md",
		}

		fmt.Printf("Attempting to call tool: %s\n", toolName)
		result, err := c.Tools.CallWithOptions(ctx, toolName, args)
		if err != nil {
			fmt.Printf("Tool call failed (this is normal if the tool is not configured): %v\n", err)

			// Try a simpler tool that might be available
			fmt.Println("\nTrying to call a different tool...")
			toolName = "mcp_brave_search_web_search" // Brave search example
			args = map[string]interface{}{
				"query": "RAGO AI platform",
			}

			result, err = c.Tools.CallWithOptions(ctx, toolName, args)
			if err != nil {
				fmt.Printf("Alternative tool also not available: %v\n", err)
			} else {
				fmt.Printf("✓ Tool result: %v\n", result)
			}
		} else {
			fmt.Printf("✓ Tool result: %v\n", result)
		}
	}

	// Example 4: Using MCP client directly for advanced operations
	fmt.Println("\n=== Example 4: Advanced MCP Operations ===")
	mcpClient := c.GetMCPClient()
	if mcpClient != nil {
		fmt.Println("✓ MCP client is available")

		// Note: The MCP client provides low-level access to MCP servers
		// For most use cases, the Tools wrapper is more convenient
		fmt.Println("  MCP client can be used for advanced server management")
		fmt.Println("  Configuration is in ~/.rago/mcpServers.json")
	} else {
		fmt.Println("MCP client not enabled")
	}

	// Example 5: Tool integration with LLM
	fmt.Println("\n=== Example 5: LLM with Tools ===")
	if c.Tools != nil && c.LLM != nil {
		// This simulates how an LLM might use tools
		fmt.Println("Demonstrating tool-augmented generation:")

		// First, get available tools
		tools, err := c.Tools.ListWithOptions(ctx)
		if err == nil && len(tools) > 0 {
			fmt.Printf("LLM has access to %d tools\n", len(tools))

			// Generate a response that might use tools
			prompt := "What files are in the current directory? (use tools if available)"
			response, err := c.LLM.Generate(prompt)
			if err != nil {
				log.Printf("Generation error: %v\n", err)
			} else {
				fmt.Printf("Response: %s\n", response)
			}
		} else {
			fmt.Println("No tools available for LLM integration")
		}
	}

	// Example 6: Disable MCP
	fmt.Println("\n=== Example 6: Disable MCP ===")
	err = c.DisableMCP()
	if err != nil {
		fmt.Printf("Failed to disable MCP: %v\n", err)
	} else {
		fmt.Println("✓ MCP disabled")
	}

	fmt.Println("\n=== MCP Tools Example Complete ===")
	fmt.Println("\nNote: For full MCP functionality:")
	fmt.Println("1. Install MCP servers:")
	fmt.Println("   npm install -g @modelcontextprotocol/server-filesystem")
	fmt.Println("   npm install -g @modelcontextprotocol/server-brave-search")
	fmt.Println("2. Configure in ~/.rago/mcpServers.json:")
	fmt.Println(`{
  "filesystem": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
  }
}`)
}
