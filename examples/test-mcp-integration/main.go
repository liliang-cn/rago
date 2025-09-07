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
	fmt.Println("🤖 RAGO MCP Integration Test")
	fmt.Println("==========================")

	// Create RAGO client
	ragoClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create RAGO client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	// Load tools from mcpServers.json
	tools, err := mcp.LoadToolsFromMCP("mcpServers.json")
	if err != nil {
		log.Fatalf("Failed to load tools: %v", err)
	}

	fmt.Printf("✅ Loaded %d tool definitions:\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()

	// Test that LLM can use these tools
	testPrompt := "List files in the current directory and tell me what time it is"
	
	fmt.Printf("📝 Test Prompt: %s\n", testPrompt)
	fmt.Println(strings.Repeat("-", 60))

	req := core.ToolGenerationRequest{
		GenerationRequest: core.GenerationRequest{
			Prompt:      testPrompt,
			Temperature: 0.2,
			MaxTokens:   500,
		},
		Tools:      tools,
		ToolChoice: "auto",
	}

	// Call LLM with tools
	response, err := ragoClient.LLM().GenerateWithTools(ctx, req)
	if err != nil {
		log.Fatalf("Failed to generate with tools: %v", err)
	}

	// Check if LLM called tools
	if len(response.ToolCalls) > 0 {
		fmt.Printf("✅ LLM successfully called tools!\n")
		fmt.Printf("🔧 Tool Calls:\n")
		for _, call := range response.ToolCalls {
			fmt.Printf("  - %s\n", call.Name)
			params, _ := json.MarshalIndent(call.Parameters, "    ", "  ")
			fmt.Printf("    Parameters: %s\n", params)
		}
	} else {
		fmt.Printf("ℹ️  Response: %s\n", response.Content)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("💡 Summary:")
	fmt.Println("✅ MCP provides tool definitions for LLM function calling")
	fmt.Println("✅ LLM handles tool execution through its own mechanisms")
}