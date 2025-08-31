package main

import (
	"fmt"
	"log"

	"github.com/liliang-cn/rago/client"
)

func main() {
	fmt.Println("🚀 RAGO MCP Integration Example")
	
	// Create client
	c, err := client.New("")
	if err != nil {
		log.Fatal("Failed to create RAGO client:", err)
	}
	defer c.Close()

	fmt.Println("✅ Client created successfully!")

	// Example: Query with potential MCP integration
	fmt.Println("\n💬 Testing query that may use MCP tools...")
	
	response, err := c.Query("What tools are available to help me with my tasks?")
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	fmt.Println("🤖 Response:", response.Answer)
	
	// Show tool usage if any
	if len(response.ToolsUsed) > 0 {
		fmt.Printf("🛠️  Tools used: %v\n", response.ToolsUsed)
	}
	
	// Show tool calls if any
	if len(response.ToolCalls) > 0 {
		fmt.Printf("🔧 Tool calls executed: %d\n", len(response.ToolCalls))
		for _, toolCall := range response.ToolCalls {
			fmt.Printf("  • %s: %v\n", toolCall.Function.Name, toolCall.Result)
		}
	}

	fmt.Println("\n💡 Note: MCP (Model Context Protocol) integration allows RAGO to")
	fmt.Println("   connect with external tools and services. Configure MCP servers")
	fmt.Println("   in your rago.toml file to enable additional capabilities.")
	fmt.Println("\n✨ MCP integration example completed!")
}