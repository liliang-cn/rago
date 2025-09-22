package rag

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/liliang-cn/rago/v2/client"
	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// runInteractiveRAGQuery runs an interactive RAG query session with history
func runInteractiveRAGQuery(ctx context.Context) error {
	// Create rago client
	ragoClient, err := client.NewWithConfig(Cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer ragoClient.Close()

	// Enable MCP if requested
	if useMCP || enableTools {
		if err := ragoClient.EnableMCP(ctx); err != nil {
			fmt.Printf("⚠️  Warning: Failed to enable MCP: %v\n", err)
			fmt.Println("Continuing without MCP tools...")
		}
	}

	// Create conversation history
	systemPrompt := "You are a helpful assistant with access to a knowledge base. Use the provided context to answer questions accurately."
	if useMCP || enableTools {
		systemPrompt = "You are a helpful assistant with access to both a knowledge base and MCP tools. Use the provided context and available tools to answer questions accurately and complete tasks."
	}
	history := client.NewConversationHistory(systemPrompt, 50)

	// Welcome message
	fmt.Println("\n📚 Interactive RAG Query Session")
	fmt.Println("=" + strings.Repeat("=", 40))
	fmt.Println("Type 'exit', 'quit', or 'bye' to end the conversation.")
	fmt.Println("Type 'clear' to reset the conversation history.")
	fmt.Println("I'll search the knowledge base to answer your questions.")
	if useMCP || enableTools {
		fmt.Println("MCP tools are also available for enhanced capabilities.")
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("💭 You: ")

		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		input := strings.TrimSpace(line)

		// Check for exit commands
		lowered := strings.ToLower(input)
		if lowered == "exit" || lowered == "quit" || lowered == "bye" {
			fmt.Println("\n👋 Goodbye! Thank you for your questions.")
			break
		}

		// Check for clear command
		if lowered == "clear" {
			history.Clear()
			fmt.Println("🔄 Conversation history cleared.")
			continue
		}

		// Skip empty input
		if input == "" {
			continue
		}

		// Generate response with RAG (and optionally MCP)
		opts := &domain.GenerationOptions{
			Temperature: temperature,
			MaxTokens:   maxTokens,
			ToolChoice:  "auto",
		}

		if showThinking {
			think := true
			opts.Think = &think
		}

		fmt.Print("\n🤖 Assistant: ")

		if useMCP || enableTools {
			// RAG + MCP with streaming
			sources, toolCalls, err := ragoClient.StreamChatWithRAGAndMCPHistory(ctx, input, history, opts, func(chunk string) {
				fmt.Print(chunk)
			})
			if err != nil {
				fmt.Printf("\nError: %v\n", err)
				continue
			}

			// Show sources if requested and available
			if showSources && len(sources) > 0 {
				fmt.Println("\n\n📚 Sources:")
				for i, source := range sources {
					fmt.Printf("  [%d] %.100s... (score: %.2f)\n", i+1, source.Content, source.Score)
					if source.Source != "" {
						fmt.Printf("      From: %s\n", source.Source)
					}
				}
			}

			// Show tool calls if any
			if len(toolCalls) > 0 {
				fmt.Println("\n🔧 Tools used:")
				for _, tc := range toolCalls {
					fmt.Printf("  - %s\n", tc.Function.Name)
				}
			}

			fmt.Println() // End the line after streaming
		} else {
			// RAG only with streaming
			sources, err := ragoClient.StreamChatWithRAGHistory(ctx, input, history, opts, func(chunk string) {
				fmt.Print(chunk)
			})
			if err != nil {
				fmt.Printf("\nError: %v\n", err)
				continue
			}

			// Show sources if requested and available
			if showSources && len(sources) > 0 {
				fmt.Println("\n\n📚 Sources:")
				for i, source := range sources {
					fmt.Printf("  [%d] %.100s... (score: %.2f)\n", i+1, source.Content, source.Score)
					if source.Source != "" {
						fmt.Printf("      From: %s\n", source.Source)
					}
				}
			}

			fmt.Println() // End the line after streaming
		}

		fmt.Println()
	}

	return nil
}
