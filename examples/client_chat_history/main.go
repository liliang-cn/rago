package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/liliang-cn/rago/v2/client"
	"github.com/liliang-cn/rago/v2/pkg/domain"
)

func main() {
	// Create a new client
	ragClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragClient.Close()

	ctx := context.Background()

	// Example 1: Simple chat with history
	fmt.Println("💬 Chat with Conversation History Demo")
	fmt.Println("=====================================")
	
	// Create conversation history
	history := client.NewConversationHistory(
		"You are a helpful AI assistant with knowledge about programming and technology.",
		20, // Keep last 20 messages
	)

	// Add some initial context
	history.AddUserMessage("I'm learning about Go programming.")
	history.AddAssistantMessage("That's great! Go is an excellent language for building scalable and efficient applications. What aspect of Go would you like to learn about?", nil)

	// Chat with history
	response, err := ragClient.ChatWithHistory(
		ctx,
		"Tell me about goroutines",
		history,
		&domain.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   500,
		},
	)
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}
	fmt.Printf("🤖 Assistant: %s\n\n", response)

	// Example 2: Chat with RAG and history
	fmt.Println("📚 Chat with RAG Context and History")
	fmt.Println("====================================")
	
	// Ingest some Go-related content
	ragClient.IngestText(
		"Goroutines are lightweight threads managed by the Go runtime. They are created using the 'go' keyword. "+
			"Channels are the pipes that connect concurrent goroutines, allowing them to communicate safely. "+
			"The select statement lets a goroutine wait on multiple communication operations.",
		"go-concurrency",
	)

	// Create a new conversation for RAG-enhanced chat
	ragHistory := client.NewConversationHistory(
		"You are a Go programming expert. Use the knowledge base to provide accurate information.",
		20,
	)

	// Chat with RAG context
	ragResponse, sources, err := ragClient.ChatWithRAGHistory(
		ctx,
		"How do goroutines communicate?",
		ragHistory,
		&domain.GenerationOptions{
			Temperature: 0.5,
			MaxTokens:   500,
		},
	)
	if err != nil {
		log.Fatalf("RAG chat failed: %v", err)
	}
	fmt.Printf("🤖 Assistant (with RAG): %s\n", ragResponse)
	if len(sources) > 0 {
		fmt.Println("📚 Sources used:")
		for i, source := range sources {
			fmt.Printf("  [%d] %s\n", i+1, source.Content[:min(100, len(source.Content))])
		}
	}

	// Example 3: Interactive chat mode
	fmt.Println("\n🎮 Interactive Chat Mode")
	fmt.Println("========================")
	fmt.Println("Type 'quit' to exit, 'history' to show conversation history")
	fmt.Println("Type 'clear' to clear history, 'rag' to toggle RAG mode")
	fmt.Println()

	interactiveHistory := client.NewConversationHistory(
		"You are a helpful assistant. Be concise and friendly.",
		50,
	)
	
	useRAG := false
	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		
		// Handle commands
		switch strings.ToLower(input) {
		case "quit", "exit":
			fmt.Println("👋 Goodbye!")
			return
		case "history":
			fmt.Println("\n📜 Conversation History:")
			for i, msg := range interactiveHistory.Messages {
				role := "System"
				if msg.Role == "user" {
					role = "You"
				} else if msg.Role == "assistant" {
					role = "Assistant"
				}
				fmt.Printf("[%d] %s: %s\n", i, role, msg.Content[:min(100, len(msg.Content))])
			}
			fmt.Println()
			continue
		case "clear":
			interactiveHistory = client.NewConversationHistory(
				"You are a helpful assistant. Be concise and friendly.",
				50,
			)
			fmt.Println("✨ History cleared")
			continue
		case "rag":
			useRAG = !useRAG
			status := "disabled"
			if useRAG {
				status = "enabled"
			}
			fmt.Printf("🔄 RAG mode %s\n", status)
			continue
		}
		
		// Add user message to history
		interactiveHistory.AddUserMessage(input)
		
		// Generate response
		var response string
		if useRAG {
			response, _, err = ragClient.ChatWithRAGHistory(
				ctx,
				input,
				interactiveHistory,
				&domain.GenerationOptions{
					Temperature: 0.7,
					MaxTokens:   500,
				},
			)
		} else {
			response, err = ragClient.ChatWithHistory(
				ctx,
				input,
				interactiveHistory,
				&domain.GenerationOptions{
					Temperature: 0.7,
					MaxTokens:   500,
				},
			)
		}
		
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}
		
		// Add assistant response to history
		interactiveHistory.AddAssistantMessage(response, nil)
		
		fmt.Printf("🤖 Assistant: %s\n\n", response)
	}

	// Example 4: Streaming chat with history
	fmt.Println("\n🌊 Streaming Chat Example")
	fmt.Println("=========================")
	
	streamHistory := client.NewConversationHistory(
		"You are a storyteller. Be creative and engaging.",
		10,
	)
	
	fmt.Print("🤖 Assistant (streaming): ")
	err = ragClient.StreamChatWithHistory(
		ctx,
		"Tell me a very short story about a programmer",
		streamHistory,
		&domain.GenerationOptions{
			Temperature: 0.9,
			MaxTokens:   200,
		},
		func(chunk string) {
			fmt.Print(chunk)
		},
	)
	fmt.Println()
	
	if err != nil {
		log.Printf("Streaming failed: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}