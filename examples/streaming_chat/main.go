package main

import (
	"context"
	"fmt"
	"log"

	"github.com/liliang-cn/rago/v2/client"
)

func main() {
	fmt.Println("🚀 RAGO Streaming Chat Example")
	
	// Create client
	c, err := client.New("")
	if err != nil {
		log.Fatal("Failed to create RAGO client:", err)
	}
	defer c.Close()

	fmt.Println("✅ Client created successfully!")

	// Example 1: Simple streaming query
	fmt.Println("\n💬 Streaming Query Example:")
	fmt.Print("🤖 Response: ")
	
	err = c.StreamQuery("Tell me about artificial intelligence in simple terms", func(chunk string) {
		fmt.Print(chunk)
	})
	if err != nil {
		log.Printf("\nStreaming query failed: %v", err)
		return
	}
	fmt.Println()

	// Example 2: LLM Chat streaming
	fmt.Println("\n💬 LLM Chat Streaming Example:")
	fmt.Print("🤖 Chat Response: ")
	
	req := client.LLMChatRequest{
		Messages: []client.ChatMessage{
			{Role: "user", Content: "What are the benefits of using Go programming language?"},
		},
		Temperature: 0.7,
		MaxTokens:   500,
	}
	
	err = c.LLMChatStream(context.Background(), req, func(chunk string) {
		fmt.Print(chunk)
	})
	if err != nil {
		log.Printf("\nLLM chat streaming failed: %v", err)
		return
	}
	
	fmt.Println("\n\n✨ Streaming examples completed!")
}