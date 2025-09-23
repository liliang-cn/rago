// Example: LLM Operations
// This example demonstrates various LLM operations using the RAGO client

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/liliang-cn/rago/v2/client"
	"github.com/liliang-cn/rago/v2/pkg/config"
)

func main() {
	// Initialize client with config
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			DefaultLLM: "ollama", // Change to your provider
		},
	}

	c, err := client.NewWithConfig(cfg)
	if err != nil {
		// Try with config file
		configPath := filepath.Join(os.Getenv("HOME"), ".rago", "rago.toml")
		c, err = client.New(configPath)
		if err != nil {
			log.Fatalf("Failed to initialize client: %v", err)
		}
	}
	defer c.Close()

	ctx := context.Background()

	// Example 1: Simple Generation
	fmt.Println("=== Example 1: Simple Generation ===")
	if c.LLM != nil {
		response, err := c.LLM.Generate("Write a haiku about programming")
		if err != nil {
			log.Printf("Generation error: %v\n", err)
		} else {
			fmt.Printf("Response:\n%s\n", response)
		}
	} else {
		fmt.Println("LLM not initialized")
	}

	// Example 2: Generation with Options
	fmt.Println("\n=== Example 2: Generation with Options ===")
	if c.LLM != nil {
		opts := &client.GenerateOptions{
			Temperature: 0.9,
			MaxTokens:   100,
		}

		response, err := c.LLM.GenerateWithOptions(
			ctx,
			"Explain quantum computing in one sentence",
			opts,
		)
		if err != nil {
			log.Printf("Generation error: %v\n", err)
		} else {
			fmt.Printf("Response:\n%s\n", response)
		}
	}

	// Example 3: Streaming Generation
	fmt.Println("\n=== Example 3: Streaming Generation ===")
	if c.LLM != nil {
		fmt.Print("Response: ")
		opts := &client.GenerateOptions{
			Temperature: 0.7,
			MaxTokens:   150,
		}

		err := c.LLM.StreamWithOptions(
			ctx,
			"Tell me a very short story about a robot",
			func(chunk string) {
				fmt.Print(chunk)
			},
			opts,
		)
		fmt.Println() // New line after streaming

		if err != nil {
			log.Printf("Streaming error: %v\n", err)
		}
	}

	// Example 4: Chat Conversation
	fmt.Println("\n=== Example 4: Chat Conversation ===")
	if c.LLM != nil {
		messages := []client.ChatMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "What is the capital of France?"},
			{Role: "assistant", Content: "The capital of France is Paris."},
			{Role: "user", Content: "What is its population?"},
		}

		opts := &client.GenerateOptions{
			Temperature: 0.5,
			MaxTokens:   100,
		}

		response, err := c.LLM.ChatWithOptions(ctx, messages, opts)
		if err != nil {
			log.Printf("Chat error: %v\n", err)
		} else {
			fmt.Printf("Response:\n%s\n", response)
		}
	}

	// Example 5: Streaming Chat
	fmt.Println("\n=== Example 5: Streaming Chat ===")
	if c.LLM != nil {
		messages := []client.ChatMessage{
			{Role: "user", Content: "Count from 1 to 5 slowly"},
		}

		fmt.Print("Response: ")
		err := c.LLM.ChatStreamWithOptions(
			ctx,
			messages,
			func(chunk string) {
				fmt.Print(chunk)
			},
			&client.GenerateOptions{Temperature: 0.3},
		)
		fmt.Println()

		if err != nil {
			log.Printf("Streaming chat error: %v\n", err)
		}
	}

	// Example 6: Using BaseClient's LLM methods directly
	fmt.Println("\n=== Example 6: Direct LLM Methods ===")
	req := client.LLMGenerateRequest{
		Prompt:      "What is 2+2?",
		Temperature: 0.1,
		MaxTokens:   50,
	}

	resp, err := c.LLMGenerate(ctx, req)
	if err != nil {
		log.Printf("Direct generation error: %v\n", err)
	} else {
		fmt.Printf("Response: %s\n", resp.Content)
	}

	// Example 7: Chat with History
	fmt.Println("\n=== Example 7: Chat with History ===")
	history := client.NewConversationHistory("You are a math tutor", 10)

	// First message
	response, err := c.ChatWithHistory(ctx, "What is calculus?", history, nil)
	if err != nil {
		log.Printf("Chat error: %v\n", err)
	} else {
		fmt.Printf("Q: What is calculus?\nA: %s\n", response)
	}

	// Follow-up question (history is maintained)
	response, err = c.ChatWithHistory(ctx, "Can you give an example?", history, nil)
	if err != nil {
		log.Printf("Chat error: %v\n", err)
	} else {
		fmt.Printf("\nQ: Can you give an example?\nA: %s\n", response)
	}

	fmt.Println("\n=== LLM Operations Complete ===")
}
