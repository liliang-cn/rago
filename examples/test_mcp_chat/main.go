package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	rago "github.com/liliang-cn/rago/client"
)

func main() {
	// Get absolute path to config file
	configPath, err := filepath.Abs("../../config.toml")
	if err != nil {
		log.Fatal("Failed to get config path:", err)
	}

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatal("Config file not found:", configPath)
	}

	// Create a rago client
	client, err := rago.New(configPath)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer client.Close()

	// Test ChatWithMCP
	fmt.Println("🧪 Testing ChatWithMCP library function...")

	opts := &rago.MCPChatOptions{
		Temperature:  0.7,
		MaxTokens:    500,
		ShowThinking: false,
	}

	response, err := client.ChatWithMCP("列出当前数据库中的所有表", opts)
	if err != nil {
		log.Fatal("ChatWithMCP failed:", err)
	}

	fmt.Printf("💬 Response: %s\n", response.Content)

	if response.FinalResponse != "" {
		fmt.Printf("🎯 Final Response: %s\n", response.FinalResponse)
	}

	if len(response.ToolCalls) > 0 {
		fmt.Printf("🔧 Tool calls made: %d\n", len(response.ToolCalls))
		for i, toolCall := range response.ToolCalls {
			fmt.Printf("  %d. %s: %v\n", i+1, toolCall.ToolName, toolCall.Success)
		}
	}

	fmt.Println("\n✅ ChatWithMCP test completed!")
}
