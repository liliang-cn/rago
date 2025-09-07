package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/client"
	"github.com/liliang-cn/rago/v2/pkg/core"
)

func main() {
	fmt.Println("🤖 RAGO LLM with MCP Tools Example")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println()
	fmt.Println("This example demonstrates how LLMs use MCP tools for enhanced capabilities")
	fmt.Println()

	// Create RAGO client
	ragoClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create RAGO client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	// Check available MCP servers
	fmt.Println("📋 Step 1: Checking MCP servers")
	fmt.Println("-" + strings.Repeat("-", 40))
	
	servers := ragoClient.MCP().ListServers()
	fmt.Printf("Found %d MCP servers\n", len(servers))
	for _, server := range servers {
		fmt.Printf("  - %s: %s\n", server.Name, server.Description)
	}
	
	// Check available tools
	fmt.Println("\n🔧 Step 2: Available MCP tools")
	fmt.Println("-" + strings.Repeat("-", 40))
	
	tools := ragoClient.MCP().ListTools()
	fmt.Printf("Found %d tools\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}
	
	// Example: Ask LLM to use tools
	fmt.Println("\n💬 Step 3: LLM with tool calling")
	fmt.Println("-" + strings.Repeat("-", 40))
	
	// Create a request that would trigger tool usage
	toolReq := core.ToolGenerationRequest{
		GenerationRequest: core.GenerationRequest{
			Prompt: "What is the current time? Please use the available time tool if possible.",
			Temperature: 0.7,
			MaxTokens: 200,
		},
		Tools: convertMCPToolsToLLMTools(tools),
		ToolChoice: "auto", // Let LLM decide when to use tools
	}
	
	fmt.Println("Asking LLM: What is the current time?")
	fmt.Println("(The LLM should use the MCP time tool if available)")
	
	// Call LLM with tools
	response, err := ragoClient.LLM().GenerateWithTools(ctx, toolReq)
	if err != nil {
		log.Printf("Failed to generate with tools: %v", err)
	} else {
		fmt.Printf("\nLLM Response: %s\n", response.Content)
		
		// Check if any tools were called
		if len(response.ToolCalls) > 0 {
			fmt.Printf("\nTools called by LLM:\n")
			for _, call := range response.ToolCalls {
				fmt.Printf("  - Tool: %s\n", call.Name)
				fmt.Printf("    Arguments: %v\n", call.Arguments)
				
				// Execute the tool call via MCP
				if strings.HasPrefix(call.Name, "mcp_") {
					fmt.Printf("    Executing MCP tool...\n")
					
					toolCallReq := core.ToolCallRequest{
						ToolName:  call.Name,
						Arguments: call.Arguments,
					}
					
					result, err := ragoClient.MCP().CallTool(ctx, toolCallReq)
					if err != nil {
						fmt.Printf("    Error: %v\n", err)
					} else {
						fmt.Printf("    Result: %v\n", result.Result)
					}
				}
			}
		} else {
			fmt.Println("\nNo tools were called by the LLM")
		}
	}
	
	// Example 2: Direct tool calling (without LLM)
	fmt.Println("\n🎯 Step 4: Direct MCP tool calling")
	fmt.Println("-" + strings.Repeat("-", 40))
	
	if len(tools) > 0 {
		// Find a time tool if available
		var timeToolName string
		for _, tool := range tools {
			if strings.Contains(tool.Name, "time") || strings.Contains(tool.Description, "time") {
				timeToolName = tool.Name
				break
			}
		}
		
		if timeToolName != "" {
			fmt.Printf("Calling tool directly: %s\n", timeToolName)
			
			directReq := core.ToolCallRequest{
				ToolName:  timeToolName,
				Arguments: map[string]interface{}{},
			}
			
			result, err := ragoClient.MCP().CallTool(ctx, directReq)
			if err != nil {
				fmt.Printf("Error calling tool: %v\n", err)
			} else {
				fmt.Printf("Tool result: %v\n", result.Result)
			}
		} else {
			fmt.Println("No time tool found")
		}
	} else {
		fmt.Println("No tools available for direct calling")
	}
	
	fmt.Println("\n✨ Example completed!")
}

// Helper function to convert MCP tools to LLM tool format
func convertMCPToolsToLLMTools(mcpTools []core.ToolInfo) []core.Tool {
	tools := make([]core.Tool, 0, len(mcpTools))
	
	for _, mcpTool := range mcpTools {
		tool := core.Tool{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
			Parameters:  mcpTool.InputSchema,
		}
		tools = append(tools, tool)
	}
	
	return tools
}