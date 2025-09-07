package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

func main() {
	// Test native tool calling with both Ollama and LMStudio
	ctx := context.Background()

	// Create client - use local config
	ragoClient, err := client.New("./rago.toml")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragoClient.Close()

	// Get available MCP tools
	tools := ragoClient.MCP().GetToolsForLLM()
	fmt.Printf("📦 Available tools: %d\n", len(tools))

	// Filter for a few key tools for testing
	var testTools []core.ToolInfo
	for _, tool := range tools {
		if tool.Name == "list_database_files" || 
		   tool.Name == "list_allowed_directories" || 
		   tool.Name == "get_current_time" {
			testTools = append(testTools, tool)
			fmt.Printf("  • %s: %s\n", tool.Name, tool.Description)
		}
	}

	if len(testTools) == 0 {
		fmt.Println("⚠️  No test tools found. Make sure MCP servers are configured.")
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	
	// Test query
	query := "What time is it and what databases are available?"
	
	// Test with Ollama first
	fmt.Println("\n🧪 Testing Native Tool Calling with Ollama:")
	fmt.Println(strings.Repeat("-", 40))
	testProvider(ctx, ragoClient, "ollama", query, testTools)
	
	// Test with LMStudio
	fmt.Println("\n🧪 Testing Native Tool Calling with LMStudio:")
	fmt.Println(strings.Repeat("-", 40))
	testProvider(ctx, ragoClient, "lmstudio", query, testTools)
}

func testProvider(ctx context.Context, client *client.Client, provider, query string, tools []core.ToolInfo) {
	// Build request with tools
	request := core.ToolGenerationRequest{
		GenerationRequest: core.GenerationRequest{
			Prompt:      query,
			Temperature: 0.7,
			MaxTokens:   1000,
			Context: []core.Message{
				{
					Role:    "system",
					Content: "You are a helpful assistant with access to tools. Use them to answer questions accurately.",
				},
				{
					Role:    "user",
					Content: query,
				},
			},
		},
		Tools:      tools,
		ToolChoice: "auto", // Let the LLM decide when to use tools
	}

	// If provider is specified, set it in the request
	if provider == "lmstudio" {
		// Try to use LMStudio model
		request.Model = "llama3.2" // Adjust based on what's loaded in LMStudio
	}

	// Generate response with native tool calling
	response, err := client.LLM().GenerateWithTools(ctx, request)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	// Display results
	fmt.Printf("✅ Provider: %s\n", response.Provider)
	fmt.Printf("📊 Model: %s\n", response.Model)
	
	if len(response.ToolCalls) > 0 {
		fmt.Printf("🔧 Tool Calls: %d\n", len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
			fmt.Printf("  → %s (ID: %s)\n", tc.Name, tc.ID)
			if tc.Parameters != nil {
				params, _ := json.MarshalIndent(tc.Parameters, "    ", "  ")
				fmt.Printf("    Parameters: %s\n", string(params))
			}
		}
	} else {
		fmt.Println("ℹ️  No tool calls made")
	}
	
	if response.Content != "" {
		fmt.Printf("💬 Response: %s\n", response.Content)
	}
	
	// Show metadata
	if response.Metadata != nil {
		if finishReason, ok := response.Metadata["finish_reason"].(string); ok && finishReason != "" {
			fmt.Printf("🏁 Finish Reason: %s\n", finishReason)
		}
		if hasTools, ok := response.Metadata["has_tools"].(bool); ok {
			fmt.Printf("🛠️  Has Tools: %v\n", hasTools)
		}
	}
}