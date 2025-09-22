package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/liliang-cn/rago/v2/client"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/spf13/cobra"
)

var (
	chatStream      bool
	chatTemperature float64
	chatMaxTokens   int
	chatSystem      string
	chatMultiline   bool
	chatInteractive bool
)

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "Chat directly with the LLM",
	Long: `Start a chat session with the configured LLM provider.
	
Examples:
  # Single message
  rago chat "What is the capital of France?"
  
  # Streaming response
  rago chat --stream "Explain quantum computing"
  
  # With system prompt
  rago chat --system "You are a helpful assistant" "Hello!"
  
  # Interactive mode (no message provided)
  rago chat
  
  # Multi-line input mode
  rago chat --multiline`,
	RunE: runChat,
}

func init() {
	chatCmd.Flags().BoolVarP(&chatStream, "stream", "s", false, "Stream the response")
	chatCmd.Flags().Float64VarP(&chatTemperature, "temperature", "t", 0.7, "Generation temperature (0.0-1.0)")
	chatCmd.Flags().IntVarP(&chatMaxTokens, "max-tokens", "m", 2000, "Maximum tokens to generate")
	chatCmd.Flags().StringVar(&chatSystem, "system", "", "System prompt to set context")
	chatCmd.Flags().BoolVar(&chatMultiline, "multiline", false, "Enable multi-line input (end with empty line)")
	chatCmd.Flags().BoolVarP(&chatInteractive, "interactive", "i", false, "Start interactive chat session with history")
}

func runChat(cmd *cobra.Command, args []string) error {
	// Initialize client with configuration
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	ctx := context.Background()

	// Create client
	ragoClient, err := client.NewWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer ragoClient.Close()

	// Check if interactive mode or no arguments provided
	if chatInteractive || len(args) == 0 {
		return runInteractiveChat(ctx, ragoClient)
	}

	// Single message mode
	var prompt string

	if len(args) > 0 {
		// Message provided as argument
		prompt = strings.Join(args, " ")
	} else if chatMultiline {
		// Multi-line input mode
		fmt.Println("📝 Multi-line input mode (press Enter twice to send):")
		prompt = readMultilineInput()
	}

	if prompt == "" {
		return fmt.Errorf("no message provided")
	}

	// Build messages with optional system prompt
	messages := []client.ChatMessage{}
	if chatSystem != "" {
		messages = append(messages, client.ChatMessage{
			Role:    "system",
			Content: chatSystem,
		})
	}
	messages = append(messages, client.ChatMessage{
		Role:    "user",
		Content: prompt,
	})

	// Prepare request
	chatReq := client.LLMChatRequest{
		Messages:    messages,
		Temperature: chatTemperature,
		MaxTokens:   chatMaxTokens,
	}

	fmt.Println("\n🤖 Assistant:")
	fmt.Println("─────────────")

	// Execute based on streaming preference
	if chatStream {
		// Streaming mode
		err = ragoClient.LLMChatStream(ctx, chatReq, func(chunk string) {
			fmt.Print(chunk)
		})
		if err != nil {
			return fmt.Errorf("streaming chat failed: %w", err)
		}
		fmt.Println() // Add newline after streaming
	} else {
		// Non-streaming mode
		resp, err := ragoClient.LLMChat(ctx, chatReq)
		if err != nil {
			return fmt.Errorf("chat failed: %w", err)
		}
		fmt.Println(resp.Content)
	}

	return nil
}

// readMultilineInput reads multiple lines until an empty line is entered
func readMultilineInput() string {
	reader := bufio.NewReader(os.Stdin)
	var lines []string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimRight(line, "\n\r")

		// Empty line signals end of input
		if line == "" {
			break
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// runInteractiveChat runs an interactive chat session with conversation history
func runInteractiveChat(ctx context.Context, ragoClient *client.BaseClient) error {
	// Create conversation history
	systemPrompt := "You are a helpful assistant."
	if chatSystem != "" {
		systemPrompt = chatSystem
	}
	history := client.NewConversationHistory(systemPrompt, 50)

	// Welcome message
	fmt.Println("\n🎭 Interactive Chat Session")
	fmt.Println("=" + strings.Repeat("=", 40))
	fmt.Println("Type 'exit', 'quit', or 'bye' to end the conversation.")
	fmt.Println("Type 'clear' to reset the conversation history.")
	if chatMultiline {
		fmt.Println("Multi-line mode: Press Enter twice to send your message.")
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("💭 You: ")

		var input string
		if chatMultiline {
			input = readMultilineInput()
		} else {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			input = strings.TrimSpace(line)
		}

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

		// Generate response
		opts := &domain.GenerationOptions{
			Temperature: chatTemperature,
			MaxTokens:   chatMaxTokens,
		}

		fmt.Print("\n🤖 Assistant: ")

		if chatStream {
			// Use streaming for better UX
			err := ragoClient.StreamChatWithHistory(ctx, input, history, opts, func(chunk string) {
				fmt.Print(chunk)
			})
			if err != nil {
				fmt.Printf("\nError: %v\n", err)
				continue
			}
			fmt.Println() // End the line after streaming
		} else {
			response, err := ragoClient.ChatWithHistory(ctx, input, history, opts)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			fmt.Println(response)
		}

		fmt.Println()
	}

	return nil
}
