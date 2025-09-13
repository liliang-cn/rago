package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/client"
)

func main() {
	// Create a new client
	ragClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragClient.Close()

	ctx := context.Background()

	// Enable MCP functionality
	fmt.Println("🔧 Enabling MCP tools...")
	err = ragClient.EnableMCP(ctx)
	if err != nil {
		// MCP is optional, continue even if it fails
		fmt.Printf("⚠️  Warning: MCP not fully available: %v\n", err)
	} else {
		fmt.Println("✅ MCP enabled successfully")
	}

	// List available MCP tools
	fmt.Println("\n📋 Available MCP Tools:")
	tools, err := ragClient.ListMCPTools()
	if err != nil {
		log.Printf("Failed to list tools: %v", err)
	} else {
		for _, tool := range tools {
			fmt.Printf("  🔧 %s - %s\n", tool.Name, tool.Description)
		}
	}

	// Example 1: Call a specific MCP tool (filesystem read)
	fmt.Println("\n📁 Reading a file using MCP filesystem tool...")
	result, err := ragClient.CallMCPTool(ctx, "mcp_filesystem_read_file", map[string]interface{}{
		"path": "README.md",
	})
	if err != nil {
		fmt.Printf("Failed to read file: %v\n", err)
	} else if result.Success {
		fmt.Println("✅ File read successfully")
		if content, ok := result.Data.(string); ok {
			fmt.Printf("📄 Content preview: %s...\n", content[:min(200, len(content))])
		}
	}

	// Example 2: Call with timeout
	fmt.Println("\n⏱️  Calling tool with timeout...")
	timeoutResult, err := ragClient.CallMCPToolWithTimeout(
		"mcp_filesystem_list_directory",
		map[string]interface{}{"path": "."},
		5*time.Second,
	)
	if err != nil {
		fmt.Printf("Failed to list directory: %v\n", err)
	} else if timeoutResult.Success {
		fmt.Println("✅ Directory listed successfully")
	}

	// Example 3: Batch tool calls
	fmt.Println("\n🔄 Executing batch tool calls...")
	batchCalls := []client.ToolCall{
		{
			ToolName: "mcp_filesystem_get_file_info",
			Args:     map[string]interface{}{"path": "go.mod"},
		},
		{
			ToolName: "mcp_filesystem_get_file_info",
			Args:     map[string]interface{}{"path": "README.md"},
		},
	}
	
	batchResults, err := ragClient.BatchCallMCPTools(ctx, batchCalls)
	if err != nil {
		fmt.Printf("Batch call failed: %v\n", err)
	} else {
		fmt.Printf("✅ Batch calls completed: %d results\n", len(batchResults))
		for i, res := range batchResults {
			fmt.Printf("  [%d] Success: %v\n", i+1, res.Success)
		}
	}

	// Example 4: Chat with MCP tools
	fmt.Println("\n💬 Chat with MCP tool assistance...")
	chatOpts := &client.MCPChatOptions{
		Temperature:  0.7,
		MaxTokens:    500,
		ShowThinking: false,
		AllowedTools: []string{"mcp_filesystem_read_file", "mcp_filesystem_list_directory"},
	}
	
	chatResponse, err := ragClient.ChatWithMCP(
		"What files are in the current directory and what's in the README?",
		chatOpts,
	)
	if err != nil {
		fmt.Printf("MCP chat failed: %v\n", err)
	} else {
		fmt.Printf("🤖 Response: %s\n", chatResponse.Content)
		if len(chatResponse.ToolCalls) > 0 {
			fmt.Println("🔧 Tools used:")
			for _, call := range chatResponse.ToolCalls {
				fmt.Printf("  - %s (Success: %v)\n", call.ToolName, call.Success)
			}
		}
	}

	// Example 5: Query with MCP tools
	fmt.Println("\n🔍 RAG Query with MCP tools...")
	
	// First ingest some content
	ragClient.IngestText(
		"RAGO supports MCP (Model Context Protocol) for external tool integration. "+
			"You can use filesystem tools to read files, web search tools to search online, "+
			"and many other tools through the MCP protocol.",
		"mcp-guide",
	)
	
	// Query with MCP tools enabled
	mcpResponse, err := ragClient.QueryWithMCP("How can I read files in RAGO?")
	if err != nil {
		fmt.Printf("Query with MCP failed: %v\n", err)
	} else {
		fmt.Printf("💡 Answer: %s\n", mcpResponse.Answer)
	}

	// Get server status
	fmt.Println("\n📊 MCP Server Status:")
	serverStatus, err := ragClient.GetMCPServerStatus()
	if err != nil {
		fmt.Printf("Failed to get server status: %v\n", err)
	} else {
		for name, running := range serverStatus {
			status := "❌ Stopped"
			if running {
				status = "✅ Running"
			}
			fmt.Printf("  %s: %s\n", name, status)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}