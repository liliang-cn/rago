package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/domain"
	mcppkg "github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/providers"
	"github.com/liliang-cn/rago/v2/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	mcpChatModel   string
	mcpChatVerbose bool
	mcpChatTimeout int
)

// mcpChatAdvancedCmd allows LLM to use MCP tools to complete tasks
var mcpChatAdvancedCmd = &cobra.Command{
	Use:   "chat [prompt]",
	Short: "Chat with LLM using MCP tools",
	Long: `Execute a chat prompt with access to MCP tools.
The LLM will automatically select and use appropriate MCP tools to complete the task.

Examples:
  rago mcp chat "count go files in current directory and save it as go.json"
  rago mcp chat "read the README.md file and summarize it"
  rago mcp chat "list all files in the current directory"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Join all arguments as the prompt
		prompt := strings.Join(args, " ")

		// Use the global Cfg
		if Cfg == nil {
			return fmt.Errorf("configuration not loaded")
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(mcpChatTimeout)*time.Second)
		defer cancel()

		// Initialize LLM service
		factory := providers.NewFactory()
		llmSvc, err := utils.InitializeLLM(ctx, Cfg, factory)
		if err != nil {
			return fmt.Errorf("failed to initialize LLM service: %w", err)
		}

		// Initialize MCP service with actual implementation
		mcpConfig := &mcppkg.Config{
			Enabled:           true,
			ServersConfigPath: Cfg.MCP.ServersConfigPath,
		}

		// Load servers from JSON file
		if err := mcpConfig.LoadServersFromJSON(); err != nil {
			if mcpChatVerbose {
				fmt.Printf("⚠️  Warning: Failed to load MCP servers: %v\n", err)
			}
		}

		// Load server configurations
		if err := mcpConfig.LoadServersFromJSON(); err != nil {
			return fmt.Errorf("failed to load server configurations: %w", err)
		}

		if mcpChatVerbose {
			fmt.Printf("📦 Loaded %d MCP servers\n", len(mcpConfig.LoadedServers))
			for _, s := range mcpConfig.LoadedServers {
				fmt.Printf("  - %s (autostart: %v)\n", s.Name, s.AutoStart)
			}
		}

		// Create MCP API which has the CallTool method
		mcpAPI := mcppkg.NewMCPLibraryAPI(mcpConfig)

		// Start MCP service
		if err := mcpAPI.Start(ctx); err != nil {
			return fmt.Errorf("failed to start MCP service: %w", err)
		}
		defer mcpAPI.Stop()

		// Get tools for LLM
		llmTools := mcpAPI.GetToolsForLLMIntegration()

		if mcpChatVerbose {
			fmt.Printf("🔧 Available MCP tools: %d\n", len(llmTools))
			for _, tool := range llmTools {
				// Debug: print the structure
				if len(llmTools) > 0 && len(llmTools) < 5 {
					fmt.Printf("  Tool structure: %+v\n", tool)
				}
				if funcMap, ok := tool["function"].(map[string]interface{}); ok {
					if nameField, ok := funcMap["name"].(string); ok {
						fmt.Printf("  - %s\n", nameField)
					}
				}
			}
			fmt.Println()
		}

		// Convert to domain.ToolDefinition format
		var toolDefs []domain.ToolDefinition
		for _, tool := range llmTools {
			// Tools are in OpenAI format: {"type": "function", "function": {...}}
			funcMap, ok := tool["function"].(map[string]interface{})
			if !ok {
				continue
			}

			// Get name from function
			name, ok := funcMap["name"].(string)
			if !ok || name == "" {
				continue
			}

			toolFunc := domain.ToolFunction{
				Name:        name,
				Description: "",
				Parameters:  make(map[string]interface{}),
			}

			if desc, ok := funcMap["description"].(string); ok {
				toolFunc.Description = desc
			}

			if params, ok := funcMap["parameters"].(map[string]interface{}); ok {
				toolFunc.Parameters = params
			}

			toolDef := domain.ToolDefinition{
				Type:     "function",
				Function: toolFunc,
			}

			toolDefs = append(toolDefs, toolDef)
		}

		fmt.Println("🤖 MCP Chat")
		fmt.Println("===========")
		fmt.Printf("📝 Task: %s\n", prompt)
		fmt.Printf("🧠 Model: %s\n", mcpChatModel)
		fmt.Printf("🔧 Available tools: %d\n", len(toolDefs))
		fmt.Println("\n⏳ Processing...")

		// Prepare messages for LLM with tools
		messages := []domain.Message{
			{
				Role:    "system",
				Content: "You are a helpful assistant with access to MCP tools. Use the available tools to complete the user's task. Be specific when calling tools and provide all required parameters.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		}

		opts := &domain.GenerationOptions{
			MaxTokens:   2000,
			Temperature: 0.7,
			ToolChoice:  "auto",
		}

		// Call LLM with tools
		toolResp, err := llmSvc.GenerateWithTools(ctx, messages, toolDefs, opts)
		if err != nil {
			return fmt.Errorf("failed to generate with tools: %w", err)
		}

		// Execute any tool calls made by the LLM
		if len(toolResp.ToolCalls) > 0 {
			fmt.Printf("\n🔨 Executing %d tool call(s)...\n", len(toolResp.ToolCalls))

			// Update messages with assistant response
			messages = append(messages, domain.Message{
				Role:      "assistant",
				Content:   toolResp.Content,
				ToolCalls: toolResp.ToolCalls,
			})

			// Execute each tool call and collect results
			for _, toolCall := range toolResp.ToolCalls {
				if mcpChatVerbose {
					fmt.Printf("  📞 Calling: %s\n", toolCall.Function.Name)
					if toolCall.Function.Arguments != nil {
						paramsJSON, _ := json.MarshalIndent(toolCall.Function.Arguments, "    ", "  ")
						fmt.Printf("    Parameters: %s\n", string(paramsJSON))
					}
				}

				// Execute the MCP tool
				result, err := mcpAPI.CallTool(ctx, toolCall.Function.Name, toolCall.Function.Arguments)

				// Add tool result to conversation
				var toolResultContent string
				if err != nil {
					toolResultContent = fmt.Sprintf("Error: %v", err)
					fmt.Printf("    ❌ Error: %v\n", err)
				} else {
					if result.Success {
						if dataStr, ok := result.Data.(string); ok {
							toolResultContent = dataStr
						} else {
							dataBytes, _ := json.Marshal(result.Data)
							toolResultContent = string(dataBytes)
						}
						if mcpChatVerbose {
							fmt.Printf("    ✅ Success: %s\n", toolResultContent)
						}
					} else {
						toolResultContent = fmt.Sprintf("Error: %s", result.Error)
						fmt.Printf("    ❌ Failed: %s\n", result.Error)
					}
				}

				// Add tool result as a tool message
				messages = append(messages, domain.Message{
					Role:       "tool",
					Content:    toolResultContent,
					ToolCallID: toolCall.ID,
				})
			}

			// Get final response from LLM with tool results in conversation
			finalOpts := &domain.GenerationOptions{
				MaxTokens:   2000,
				Temperature: 0.7,
			}

			// Convert messages to a single prompt for final generation
			var finalPrompt string
			for _, msg := range messages {
				finalPrompt += fmt.Sprintf("%s: %s\n\n", msg.Role, msg.Content)
			}

			finalResp, err := llmSvc.Generate(ctx, finalPrompt, finalOpts)
			if err != nil {
				return fmt.Errorf("failed to generate final response: %w", err)
			}

			fmt.Println("\n✅ Result:")
			fmt.Println("----------")
			fmt.Println(finalResp)
		} else {
			// No tool calls, just show the response
			fmt.Println("\n📄 Response:")
			fmt.Println("------------")
			fmt.Println(toolResp.Content)
		}

		return nil
	},
}

// NOTE: Commenting out to avoid duplicate command registration
// The mcp.go file already has an mcpChatCmd registered
// This file contains an alternative implementation that can be enabled if needed
/*
func init() {
	// Add chat command to mcp command
	mcpCmd.AddCommand(mcpChatAdvancedCmd)

	// Add flags
	mcpChatAdvancedCmd.Flags().StringVarP(&mcpChatModel, "model", "m", "", "LLM model to use (defaults to configured model)")
	mcpChatAdvancedCmd.Flags().BoolVarP(&mcpChatVerbose, "verbose", "v", false, "Show detailed execution steps")
	mcpChatAdvancedCmd.Flags().IntVarP(&mcpChatTimeout, "timeout", "t", 60, "Timeout in seconds")
}
*/
