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
	// Create a new rago client
	ragoClient, err := client.New("")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer ragoClient.Close()

	ctx := context.Background()

	// Enable MCP if configured
	if err := ragoClient.EnableMCP(ctx); err != nil {
		fmt.Printf("Note: MCP not enabled: %v\n", err)
	}

	// Create conversation history
	var history *client.ConversationHistory

	// Determine chat mode from command line
	chatMode := "normal"
	if len(os.Args) > 1 {
		chatMode = os.Args[1]
	}

	switch chatMode {
	case "rag":
		fmt.Println("🔍 RAG-Enhanced Chat Mode")
		fmt.Println("I'll search the knowledge base to answer your questions.")
		history = client.NewConversationHistory(
			"You are a helpful assistant with access to a knowledge base. Use the provided context to answer questions accurately.",
			50,
		)
	case "mcp":
		fmt.Println("🔧 MCP Tools Chat Mode")
		fmt.Println("I have access to MCP tools to help you.")
		history = client.NewConversationHistory(
			"You are a helpful assistant with access to MCP tools. Use the available tools to help answer questions and complete tasks.",
			50,
		)
	case "ragmcp":
		fmt.Println("🔍🔧 RAG + MCP Chat Mode")
		fmt.Println("I'll use both knowledge base and MCP tools to help you.")
		history = client.NewConversationHistory(
			"You are a helpful assistant with access to both a knowledge base and MCP tools. Use the provided context and available tools to answer questions accurately and complete tasks.",
			50,
		)
	default:
		fmt.Println("💬 Normal Chat Mode")
		history = client.NewConversationHistory(
			"You are a helpful assistant.",
			50,
		)
	}

	fmt.Println("Type 'exit', 'quit', or 'bye' to end the conversation.")
	fmt.Println("Type 'clear' to reset the conversation history.")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		// Check for exit commands
		if input == "exit" || input == "quit" || input == "bye" {
			fmt.Println("\n👋 Goodbye!")
			break
		}

		// Check for clear command
		if input == "clear" {
			history.Clear()
			fmt.Println("🔄 Conversation history cleared.")
			continue
		}

		// Skip empty input
		if input == "" {
			continue
		}

		// Generate response based on mode
		opts := &domain.GenerationOptions{
			Temperature: 0.7,
			MaxTokens:   2000,
		}

		fmt.Print("\nAssistant: ")

		switch chatMode {
		case "rag":
			response, sources, err := ragoClient.ChatWithRAGHistory(ctx, input, history, opts)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			fmt.Println(response)

			// Show sources if available
			if len(sources) > 0 {
				fmt.Println("\n📚 Sources:")
				for i, source := range sources {
					fmt.Printf("  [%d] %.100s... (score: %.2f)\n", i+1, source.Content, source.Score)
				}
			}

		case "mcp":
			response, toolCalls, err := ragoClient.ChatWithMCPHistory(ctx, input, history, opts)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			// Show tool calls if any
			if len(toolCalls) > 0 {
				fmt.Println("\n🔧 Tools used:")
				for _, tc := range toolCalls {
					fmt.Printf("  - %s\n", tc.Function.Name)
				}
				fmt.Println()
			}

			fmt.Println(response)

		case "ragmcp":
			response, sources, toolCalls, err := ragoClient.ChatWithRAGAndMCPHistory(ctx, input, history, opts)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			// Show sources if available
			if len(sources) > 0 {
				fmt.Println("\n📚 Sources:")
				for i, source := range sources {
					fmt.Printf("  [%d] %.100s... (score: %.2f)\n", i+1, source.Content, source.Score)
				}
			}

			// Show tool calls if any
			if len(toolCalls) > 0 {
				fmt.Println("\n🔧 Tools used:")
				for _, tc := range toolCalls {
					fmt.Printf("  - %s\n", tc.Function.Name)
				}
			}

			if len(sources) > 0 || len(toolCalls) > 0 {
				fmt.Println()
			}

			fmt.Println(response)

		default:
			response, err := ragoClient.ChatWithHistory(ctx, input, history, opts)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			fmt.Println(response)
		}

		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading input: %v", err)
	}
}
