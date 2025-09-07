package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
)

func main() {
	fmt.Println("🤖 RAGO MCP Tools Test")
	fmt.Println("==========================")
	fmt.Println("Loading tool definitions from mcpServers.json for LLM tool calling\n")

	// Create RAGO client
	ragoClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create RAGO client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	// Load tools from mcpServers.json
	// These are TOOL DEFINITIONS for the LLM, NOT running servers!
	tools, err := mcp.LoadToolsFromMCP("mcpServers.json")
	if err != nil {
		log.Fatalf("Failed to load tools from mcpServers.json: %v", err)
	}

	fmt.Printf("📋 Loaded %d tools from mcpServers.json:\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s (%s): %s\n", tool.Name, tool.ServerName, tool.Description)
	}
	fmt.Println()

	// Test cases using the loaded MCP tools
	testCases := []struct {
		prompt string
		desc   string
	}{
		{
			prompt: "List the files in the current directory",
			desc:   "Filesystem Tool",
		},
		{
			prompt: "What's the current time in New York?",
			desc:   "Time Tool",
		},
		{
			prompt: "Store the value 'test data' with key 'example' in memory",
			desc:   "Memory Tool",
		},
		{
			prompt: "Fetch the content from https://example.com",
			desc:   "Fetch Tool",
		},
		{
			prompt: "Read the README.md file and tell me the current time",
			desc:   "Multi-Tool",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n📝 Test: %s\n", tc.desc)
		fmt.Printf("Prompt: %s\n", tc.prompt)
		fmt.Println(strings.Repeat("-", 60))

		// Create request with MCP tools
		req := core.ToolGenerationRequest{
			GenerationRequest: core.GenerationRequest{
				Prompt:      tc.prompt,
				Temperature: 0.2,
				MaxTokens:   500,
			},
			Tools:      tools, // Tools loaded from mcpServers.json
			ToolChoice: "auto",
		}

		// Call LLM with tools
		response, err := ragoClient.LLM().GenerateWithTools(ctx, req)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		// Check if LLM recognized and called the tools
		if len(response.ToolCalls) > 0 {
			fmt.Printf("✅ LLM recognized MCP tools!\n")
			fmt.Printf("🔧 Tool Calls:\n")
			for _, call := range response.ToolCalls {
				fmt.Printf("  - Tool: %s\n", call.Name)
				params, _ := json.MarshalIndent(call.Parameters, "    ", "  ")
				fmt.Printf("    Parameters: %s\n", params)
				
				// The LLM knows how to call these tools!
				// In a real implementation, the LLM would handle execution
				fmt.Printf("    → LLM would execute: %s with given parameters\n", call.Name)
			}
		} else {
			fmt.Printf("ℹ️  No tools called\n")
			fmt.Printf("Response: %s\n", response.Content)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("💡 Key Points:")
	fmt.Println("- Tools loaded from mcpServers.json")
	fmt.Println("- These are TOOL DEFINITIONS, not running servers")
	fmt.Println("- LLM recognizes and calls appropriate tools")
	fmt.Println("- Tool execution is handled by the LLM's tool calling")
	fmt.Println("\n✨ MCP tools test completed!")
}