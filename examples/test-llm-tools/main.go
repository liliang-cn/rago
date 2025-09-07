package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

func main() {
	fmt.Println("🤖 Testing LLM Tool Registration from MCP")
	fmt.Println("==========================================\n")

	// Create RAGO client
	ragoClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create RAGO client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	// Step 1: Check what tools are available from MCP
	fmt.Println("📋 Available MCP Tools:")
	fmt.Println("-----------------------")
	
	tools := ragoClient.MCP().ListTools()
	if len(tools) == 0 {
		fmt.Println("No tools registered from MCP servers")
		fmt.Println("\nThis means the tool definitions in mcpServers.json")
		fmt.Println("are not being loaded as LLM-callable functions.")
	} else {
		for _, tool := range tools {
			fmt.Printf("\n✓ Tool: %s\n", tool.Name)
			fmt.Printf("  Description: %s\n", tool.Description)
			if tool.InputSchema != nil {
				schemaJSON, _ := json.MarshalIndent(tool.InputSchema, "  ", "  ")
				fmt.Printf("  Schema: %s\n", schemaJSON)
			}
		}
	}

	// Step 2: Test if LLM can use tools
	fmt.Println("\n\n💬 Testing LLM with Tool Calling:")
	fmt.Println("----------------------------------")
	
	// Create a prompt that would naturally trigger tool use
	prompts := []string{
		"What files are in the current directory?",
		"What's the current time?",
		"Can you fetch the content from https://example.com?",
		"Store the value 'test123' in memory with key 'mykey'",
	}

	for _, prompt := range prompts {
		fmt.Printf("\n🔹 Prompt: %s\n", prompt)
		
		// Try with tool support
		toolReq := core.ToolGenerationRequest{
			GenerationRequest: core.GenerationRequest{
				Prompt:      prompt,
				Temperature: 0.3,
				MaxTokens:   500,
			},
			Tools: tools,
			ToolChoice: "auto",
		}

		response, err := ragoClient.LLM().GenerateWithTools(ctx, toolReq)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
			continue
		}

		fmt.Printf("   Response: %s\n", response.Content)
		
		if len(response.ToolCalls) > 0 {
			fmt.Printf("   🔧 Tool Calls Made:\n")
			for _, call := range response.ToolCalls {
				fmt.Printf("      - %s with args: %v\n", call.Name, call.Parameters)
			}
		} else {
			fmt.Printf("   ℹ️  No tools were called\n")
		}
	}

	// Step 3: Show how tools SHOULD be registered
	fmt.Println("\n\n📚 How MCP Tools Should Work:")
	fmt.Println("------------------------------")
	fmt.Println("1. mcpServers.json defines TOOL SPECIFICATIONS")
	fmt.Println("2. RAGO reads these and creates tool definitions")
	fmt.Println("3. Tool definitions are passed to LLM as available functions")
	fmt.Println("4. LLM decides when to use tools based on user prompts")
	fmt.Println("5. LLM returns tool call requests in its response")
	fmt.Println("6. RAGO or LLM executes the tool and gets results")
	fmt.Println("7. Results are used to generate the final response")

	fmt.Println("\n✨ Test completed!")
}

