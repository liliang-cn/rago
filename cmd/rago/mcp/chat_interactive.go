package mcp

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/liliang-cn/rago/v2/client"
	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// runInteractiveMCPChat runs an interactive MCP chat session with history
func runInteractiveMCPChat(ctx context.Context, temperature float64, maxTokens int) error {
	// Create rago client
	ragoClient, err := client.NewWithConfig(Cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer ragoClient.Close()

	// Start MCP servers with failure tolerance
	fmt.Println("Starting MCP servers...")
	if err := ragoClient.EnableMCP(ctx); err != nil {
		// Try to continue anyway, some servers might still work
		fmt.Printf("⚠️  Note: Some MCP servers may not be available: %v\n", err)
	}

	// Create conversation history
	systemPrompt := "You are a helpful assistant with access to MCP tools. Use the available tools to help answer questions and complete tasks."
	history := client.NewConversationHistory(systemPrompt, 50)

	// Welcome message
	fmt.Println("\n🔧 Interactive MCP Chat Session")
	fmt.Println("=" + strings.Repeat("=", 40))
	fmt.Println("Type 'exit', 'quit', or 'bye' to end the conversation.")
	fmt.Println("Type 'clear' to reset the conversation history.")
	fmt.Println("MCP tools are available for enhanced capabilities.")
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
			fmt.Println("\n👋 Goodbye! Thank you for chatting.")
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

		// Generate response with MCP tools
		opts := &domain.GenerationOptions{
			Temperature: temperature,
			MaxTokens:   maxTokens,
			ToolChoice:  "auto",
		}

		fmt.Print("\n🤖 Assistant: ")

		// Use streaming for better UX
		var receivedToolCalls []domain.ToolCall
		toolCalls, err := ragoClient.StreamChatWithMCPHistory(ctx, input, history, opts, func(chunk string) {
			fmt.Print(chunk)
		})
		if err != nil {
			fmt.Printf("\nError: %v\n", err)
			continue
		}
		receivedToolCalls = toolCalls

		// Show tool calls if any
		if len(receivedToolCalls) > 0 {
			fmt.Println("\n\n🔧 Tools used:")
			for _, tc := range receivedToolCalls {
				fmt.Printf("  - %s\n", tc.Function.Name)
			}
		}

		fmt.Println() // End the line after streaming
		fmt.Println()
	}

	return nil
}
