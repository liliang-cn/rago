package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/rago"
)

func main() {
	fmt.Println("🚀 RAGO Simple MCP Client Example")
	fmt.Println("=================================\n")

	// Create a simple client with MCP enabled by default
	client, err := rago.NewSimpleClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Wait for servers to start
	fmt.Println("⏳ Starting MCP servers...")
	time.Sleep(2 * time.Second)

	// Check server status
	fmt.Println("\n📊 MCP Server Status:")
	serverStatus := client.GetServerStatus()
	for server, connected := range serverStatus {
		status := "❌ Stopped"
		if connected {
			status = "✅ Running"
		}
		fmt.Printf("  %s: %s\n", server, status)
	}

	// List available tools
	fmt.Println("\n🔧 Available MCP Tools:")
	tools := client.ListTools()
	
	if len(tools) == 0 {
		fmt.Println("  No tools available (servers may still be starting)")
	} else {
		for name, tool := range tools {
			fmt.Printf("  • %s - %s\n", name, tool.Description())
		}
	}

	// Example tool calls
	ctx := context.Background()
	
	fmt.Println("\n📝 Example Tool Calls:")
	fmt.Println("---------------------")

	// Example 1: Get current time
	fmt.Println("\n1. Getting current time...")
	result, err := client.CallTool(ctx, "time_get_current_time", map[string]interface{}{
		"timezone": "UTC",
	})
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
	} else {
		fmt.Printf("   ✅ Current time: %v\n", result)
	}

	// Example 2: Store and retrieve from memory
	fmt.Println("\n2. Using memory storage...")
	
	// Store a value
	_, err = client.CallTool(ctx, "memory_store", map[string]interface{}{
		"key":   "example_key",
		"value": "Hello from RAGO MCP Client!",
	})
	if err != nil {
		fmt.Printf("   ❌ Failed to store: %v\n", err)
	} else {
		fmt.Println("   ✅ Value stored successfully")
	}
	
	// Retrieve the value
	result, err = client.CallTool(ctx, "memory_retrieve", map[string]interface{}{
		"key": "example_key",
	})
	if err != nil {
		fmt.Printf("   ❌ Failed to retrieve: %v\n", err)
	} else {
		fmt.Printf("   ✅ Retrieved value: %v\n", result)
	}

	// Example 3: List files (if filesystem server is running)
	fmt.Println("\n3. Listing files in current directory...")
	result, err = client.CallTool(ctx, "filesystem_list", map[string]interface{}{
		"path": "./",
	})
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
		fmt.Println("   (Filesystem server may not be running)")
	} else {
		fmt.Printf("   ✅ Files: %v\n", result)
	}

	fmt.Println("\n✨ Example completed!")
	fmt.Println("\nThis example demonstrated:")
	fmt.Println("  • Automatic MCP server startup")
	fmt.Println("  • Server status checking")
	fmt.Println("  • Tool discovery")
	fmt.Println("  • Direct tool execution")
}