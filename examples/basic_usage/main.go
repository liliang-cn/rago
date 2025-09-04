package main

import (
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/client"
)

func main() {
	fmt.Println("🚀 RAGO Basic Usage Example")

	// Create a client with default configuration
	// This will look for rago.toml in the current directory or ~/.rago/rago.toml
	c, err := client.New("")
	if err != nil {
		log.Fatal("Failed to create RAGO client:", err)
	}
	defer func() {
		if err := c.Close(); err != nil {
			log.Printf("Failed to close client: %v", err)
		}
	}()

	fmt.Println("✅ Client created successfully!")

	// Test basic query functionality
	fmt.Println("\n📝 Testing basic query...")
	response, err := c.Query("Hello, what can you help me with?")
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	fmt.Println("🤖 Response:", response.Answer)
	fmt.Printf("⚡ Processing took: %v\n", response.Elapsed)

	// Show retrieved sources if any
	if len(response.Sources) > 0 {
		fmt.Printf("📚 Used %d source chunks for the response\n", len(response.Sources))
	}

	// Show tool usage if any
	if len(response.ToolsUsed) > 0 {
		fmt.Printf("🛠️  Tools used: %v\n", response.ToolsUsed)
	}

	fmt.Println("\n✨ Basic usage example completed!")
}
