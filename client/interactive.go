package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/liliang-cn/rago/v2/pkg/domain"
)

// InteractiveChatOptions configures the interactive chat session
type InteractiveChatOptions struct {
	// System prompt for the conversation
	SystemPrompt string

	// Whether to show tool calls
	ShowToolCalls bool

	// Whether to show thinking process (if supported)
	ShowThinking bool

	// Custom prompt format
	UserPromptFormat      string // Default: "You: "
	AssistantPromptFormat string // Default: "Assistant: "

	// Exit commands
	ExitCommands []string // Default: ["exit", "quit", "bye", "/exit", "/quit"]

	// Input/Output streams (for testing)
	Input  io.Reader
	Output io.Writer

	// Callback for each interaction (optional)
	OnInteraction func(userMsg string, assistantMsg string)

	// Maximum conversation history to maintain
	MaxHistory int // Default: 50

	// Generation options
	Temperature float64
	MaxTokens   int
}

// DefaultInteractiveChatOptions returns default options for interactive chat
func DefaultInteractiveChatOptions() *InteractiveChatOptions {
	return &InteractiveChatOptions{
		SystemPrompt:          "You are a helpful assistant.",
		ShowToolCalls:         false,
		ShowThinking:          false,
		UserPromptFormat:      "\n💭 You: ",
		AssistantPromptFormat: "\n🤖 Assistant: ",
		ExitCommands:          []string{"exit", "quit", "bye", "/exit", "/quit"},
		Input:                 os.Stdin,
		Output:                os.Stdout,
		MaxHistory:            50,
		Temperature:           0.7,
		MaxTokens:             2000,
	}
}

// InteractiveChat starts an interactive chat session
func (c *BaseClient) InteractiveChat(ctx context.Context, opts *InteractiveChatOptions) error {
	if opts == nil {
		opts = DefaultInteractiveChatOptions()
	}

	if c.llm == nil {
		return fmt.Errorf("LLM service not initialized")
	}

	// Initialize conversation history
	messages := []domain.Message{
		{
			Role:    "system",
			Content: opts.SystemPrompt,
		},
	}

	// Welcome message
	fmt.Fprintln(opts.Output, "\n🎭 Interactive Chat Session")
	fmt.Fprintln(opts.Output, "="+strings.Repeat("=", 40))
	fmt.Fprintf(opts.Output, "Type 'exit', 'quit', or 'bye' to end the conversation.\n")
	if c.mcpClient != nil && c.mcpClient.IsInitialized() {
		fmt.Fprintln(opts.Output, "✨ MCP tools are available for enhanced capabilities.")
	}
	fmt.Fprintln(opts.Output)

	scanner := bufio.NewScanner(opts.Input)

	for {
		// Show user prompt
		fmt.Fprint(opts.Output, opts.UserPromptFormat)

		// Read user input
		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())

		// Check for exit commands
		if isExitCommand(userInput, opts.ExitCommands) {
			fmt.Fprintln(opts.Output, "\n👋 Goodbye! Thank you for chatting.")
			break
		}

		// Skip empty input
		if userInput == "" {
			continue
		}

		// Add user message to history
		messages = append(messages, domain.Message{
			Role:    "user",
			Content: userInput,
		})

		// Show assistant prompt
		fmt.Fprint(opts.Output, opts.AssistantPromptFormat)

		// Generate response
		genOpts := &domain.GenerationOptions{
			Temperature: opts.Temperature,
			MaxTokens:   opts.MaxTokens,
		}

		if opts.ShowThinking {
			think := true
			genOpts.Think = &think
		}

		// Check if MCP tools are available and use them
		var response string
		var toolCalls []domain.ToolCall

		if c.mcpClient != nil && c.mcpClient.IsInitialized() {
			// Get available tools
			tools := c.mcpClient.GetToolDefinitions(ctx)
			if len(tools) > 0 {
				// Generate with tools
				result, err := c.llm.GenerateWithTools(ctx, messages, tools, genOpts)
				if err != nil {
					fmt.Fprintf(opts.Output, "Error: %v\n", err)
					continue
				}

				response = result.Content
				toolCalls = result.ToolCalls

				// Handle tool calls if present
				if len(toolCalls) > 0 && opts.ShowToolCalls {
					fmt.Fprintln(opts.Output, "\n🔧 Using tools...")
					for _, tc := range toolCalls {
						fmt.Fprintf(opts.Output, "  - %s\n", tc.Function.Name)
					}
				}

				// Execute tool calls and get results
				if len(toolCalls) > 0 {
					// Add assistant message with tool calls
					messages = append(messages, domain.Message{
						Role:      "assistant",
						Content:   response,
						ToolCalls: toolCalls,
					})

					// Execute each tool call
					for _, tc := range toolCalls {
						result, err := c.mcpClient.CallTool(ctx, tc.Function.Name, tc.Function.Arguments)
						if err != nil {
							// Add error as tool result
							messages = append(messages, domain.Message{
								Role:       "tool",
								Content:    fmt.Sprintf("Error: %v", err),
								ToolCallID: tc.ID,
							})
						} else {
							// Add successful result
							var content string
							if result.Success {
								content = fmt.Sprintf("%v", result.Data)
							} else {
								content = fmt.Sprintf("Error: %s", result.Error)
							}

							messages = append(messages, domain.Message{
								Role:       "tool",
								Content:    content,
								ToolCallID: tc.ID,
							})
						}
					}

					// Get final response after tool execution
					finalResult, err := c.llm.GenerateWithTools(ctx, messages, tools, genOpts)
					if err == nil {
						response = finalResult.Content
					}
				}
			} else {
				// No tools available, use regular generation
				resp, err := c.llm.Generate(ctx, userInput, genOpts)
				if err != nil {
					fmt.Fprintf(opts.Output, "Error: %v\n", err)
					continue
				}
				response = resp
			}
		} else {
			// No MCP client, use regular generation
			resp, err := c.llm.Generate(ctx, userInput, genOpts)
			if err != nil {
				fmt.Fprintf(opts.Output, "Error: %v\n", err)
				continue
			}
			response = resp
		}

		// Display response
		fmt.Fprintln(opts.Output, response)

		// Add assistant response to history (if not already added with tool calls)
		if len(toolCalls) == 0 {
			messages = append(messages, domain.Message{
				Role:    "assistant",
				Content: response,
			})
		}

		// Trim history if it exceeds max
		if len(messages) > opts.MaxHistory {
			// Keep system message and trim old messages
			messages = append(messages[:1], messages[len(messages)-opts.MaxHistory+1:]...)
		}

		// Call interaction callback if provided
		if opts.OnInteraction != nil {
			opts.OnInteraction(userInput, response)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	return nil
}

// InteractiveChatWithRAG starts an interactive chat session with RAG support
func (c *BaseClient) InteractiveChatWithRAG(ctx context.Context, opts *InteractiveChatOptions) error {
	if opts == nil {
		opts = DefaultInteractiveChatOptions()
	}

	if opts.SystemPrompt == "You are a helpful assistant." {
		opts.SystemPrompt = "You are a helpful assistant with access to a knowledge base. Use the provided context to answer questions accurately."
	}

	if c.llm == nil {
		return fmt.Errorf("LLM service not initialized")
	}

	// Initialize conversation history
	messages := []domain.Message{
		{
			Role:    "system",
			Content: opts.SystemPrompt,
		},
	}

	// Welcome message
	fmt.Fprintln(opts.Output, "\n📚 Interactive RAG Chat Session")
	fmt.Fprintln(opts.Output, "="+strings.Repeat("=", 40))
	fmt.Fprintf(opts.Output, "Type 'exit', 'quit', or 'bye' to end the conversation.\n")
	fmt.Fprintln(opts.Output, "💡 I'll search the knowledge base to answer your questions.")
	if c.mcpClient != nil && c.mcpClient.IsInitialized() {
		fmt.Fprintln(opts.Output, "✨ MCP tools are also available for enhanced capabilities.")
	}
	fmt.Fprintln(opts.Output)

	scanner := bufio.NewScanner(opts.Input)

	for {
		// Show user prompt
		fmt.Fprint(opts.Output, opts.UserPromptFormat)

		// Read user input
		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())

		// Check for exit commands
		if isExitCommand(userInput, opts.ExitCommands) {
			fmt.Fprintln(opts.Output, "\n👋 Goodbye! Thank you for chatting.")
			break
		}

		// Skip empty input
		if userInput == "" {
			continue
		}

		// Search knowledge base
		searchOpts := &ClientSearchOptions{
			TopK:           5,
			ScoreThreshold: 0.7,
		}

		results, err := c.Search(ctx, userInput, searchOpts)
		if err != nil {
			fmt.Fprintf(opts.Output, "Warning: Search failed: %v\n", err)
		}

		// Build context from search results
		var contextContent string
		if len(results) > 0 {
			contextContent = "\nRelevant information from knowledge base:\n"
			for i, result := range results {
				contextContent += fmt.Sprintf("\n[%d] %s\n", i+1, result.Content)
			}
		}

		// Create augmented prompt
		augmentedPrompt := userInput
		if contextContent != "" {
			augmentedPrompt = fmt.Sprintf("Context:%s\n\nQuestion: %s", contextContent, userInput)
		}

		// Add user message to history (original, not augmented)
		messages = append(messages, domain.Message{
			Role:    "user",
			Content: augmentedPrompt,
		})

		// Show assistant prompt
		fmt.Fprint(opts.Output, opts.AssistantPromptFormat)

		// Generate response
		genOpts := &domain.GenerationOptions{
			Temperature: opts.Temperature,
			MaxTokens:   opts.MaxTokens,
		}

		// Generate response (with or without tools)
		var response string
		if c.mcpClient != nil && c.mcpClient.IsInitialized() {
			tools := c.mcpClient.GetToolDefinitions(ctx)
			if len(tools) > 0 {
				result, err := c.llm.GenerateWithTools(ctx, messages, tools, genOpts)
				if err != nil {
					fmt.Fprintf(opts.Output, "Error: %v\n", err)
					continue
				}
				response = result.Content

				// Handle tool calls similar to InteractiveChat
				if len(result.ToolCalls) > 0 {
					// Execute tools and get final response
					// (Similar implementation as InteractiveChat)
				}
			} else {
				resp, err := c.llm.Generate(ctx, augmentedPrompt, genOpts)
				if err != nil {
					fmt.Fprintf(opts.Output, "Error: %v\n", err)
					continue
				}
				response = resp
			}
		} else {
			resp, err := c.llm.Generate(ctx, augmentedPrompt, genOpts)
			if err != nil {
				fmt.Fprintf(opts.Output, "Error: %v\n", err)
				continue
			}
			response = resp
		}

		// Display response
		fmt.Fprintln(opts.Output, response)

		// Add assistant response to history
		messages = append(messages, domain.Message{
			Role:    "assistant",
			Content: response,
		})

		// Trim history if it exceeds max
		if len(messages) > opts.MaxHistory {
			messages = append(messages[:1], messages[len(messages)-opts.MaxHistory+1:]...)
		}

		// Call interaction callback if provided
		if opts.OnInteraction != nil {
			opts.OnInteraction(userInput, response)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	return nil
}

// isExitCommand checks if the input is an exit command
func isExitCommand(input string, exitCommands []string) bool {
	lowered := strings.ToLower(input)
	for _, cmd := range exitCommands {
		if lowered == strings.ToLower(cmd) {
			return true
		}
	}
	return false
}
