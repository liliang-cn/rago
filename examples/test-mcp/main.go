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
	fmt.Println("🔧 Testing MCP (Model Context Protocol) Functionality")
	fmt.Println("=" + strings.Repeat("=", 50))

	// Create RAGO client with default config
	ragoClient, err := client.New("") // Uses default config search path
	if err != nil {
		log.Fatalf("Failed to create RAGO client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	// Check if MCP service is available
	if ragoClient.MCP() == nil {
		log.Fatal("MCP service is not available")
	}

	fmt.Println("\n📋 Step 1: Listing available MCP servers")
	fmt.Println("-" + strings.Repeat("-", 40))

	// List available servers
	servers := ragoClient.MCP().ListServers()
	fmt.Printf("Found %d MCP servers\n", len(servers))
	
	for _, server := range servers {
		fmt.Printf("\n  Server: %s\n", server.Name)
		fmt.Printf("    Version: %s\n", server.Version)
		if server.Description != "" {
			fmt.Printf("    Description: %s\n", server.Description)
		}
		// Get individual server health
		health := ragoClient.MCP().GetServerHealth(server.Name)
		fmt.Printf("    Health: %v\n", health)
	}

	// Try to start the MCP service if it has a Start method
	fmt.Println("\n🚀 Step 2: Starting MCP service")
	fmt.Println("-" + strings.Repeat("-", 40))

	// The Start method should be called internally, but let's check
	// Note: This might need adjustment based on the actual interface
	
	// Try to discover available tools
	fmt.Println("\n🔍 Step 3: Discovering available tools")
	fmt.Println("-" + strings.Repeat("-", 40))

	tools := ragoClient.MCP().ListTools()
	fmt.Printf("Found %d tools\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("\n  Tool: %s\n", tool.Name)
		fmt.Printf("    Description: %s\n", tool.Description)
		if tool.InputSchema != nil {
			fmt.Printf("    Input Schema: %v\n", tool.InputSchema)
		}
	}

	// Try to call a simple tool if available
	fmt.Println("\n⚡ Step 4: Testing tool execution (if tools available)")
	fmt.Println("-" + strings.Repeat("-", 40))

	if len(tools) > 0 {
		// Try to use the time tool if available
		for _, tool := range tools {
			if tool.Name == "get_current_time" || strings.Contains(tool.Name, "time") {
				fmt.Printf("\nTrying to execute tool: %s\n", tool.Name)
				
				callReq := core.ToolCallRequest{
					ToolName:  tool.Name,
					Arguments: map[string]interface{}{},
				}

				result, err := ragoClient.MCP().CallTool(ctx, callReq)
				if err != nil {
					fmt.Printf("Failed to call tool: %v\n", err)
				} else {
					fmt.Printf("Tool result: %v\n", result.Result)
				}
				break
			}
		}
	}

	// Test server health for each server
	fmt.Println("\n💓 Step 5: Checking individual server health")
	fmt.Println("-" + strings.Repeat("-", 40))

	for _, server := range servers {
		health := ragoClient.MCP().GetServerHealth(server.Name)
		fmt.Printf("Server %s health: %v\n", server.Name, health)
	}

	// Try batch tool calls if we have multiple tools
	fmt.Println("\n🚀 Step 6: Testing batch tool calls (if multiple tools available)")
	fmt.Println("-" + strings.Repeat("-", 40))

	if len(tools) > 1 {
		var batchRequests []core.ToolCallRequest
		for i, tool := range tools {
			if i >= 2 {
				break // Just test with 2 tools
			}
			batchRequests = append(batchRequests, core.ToolCallRequest{
				ToolName:  tool.Name,
				Arguments: map[string]interface{}{},
			})
		}
		
		results, err := ragoClient.MCP().CallToolsBatch(ctx, batchRequests)
		if err != nil {
			fmt.Printf("Failed batch call: %v\n", err)
		} else {
			fmt.Printf("Batch results: %d responses\n", len(results))
			for i, result := range results {
				fmt.Printf("  Result %d: %v\n", i+1, result.Result)
			}
		}
	} else {
		fmt.Println("Not enough tools for batch testing")
	}

	fmt.Println("\n✨ MCP testing completed!")
}