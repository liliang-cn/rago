package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/services"
	"github.com/spf13/cobra"
)

// MCPCmd is the parent command for MCP operations - exported for use in root.go
var MCPCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP (Model Context Protocol) management and tool usage",
	Long: `Manage MCP servers and use MCP tools.

MCP enables rago to connect to external tools and services through the Model Context Protocol.
Use these commands to start/stop MCP servers, list available tools, and call tools directly.`,
}

var mcpStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of MCP servers and tools",
	RunE:  runMCPStatus,
}

var mcpStartCmd = &cobra.Command{
	Use:   "start [server-name]",
	Short: "Start MCP server(s)",
	Long:  "Start one or all MCP servers. If no server name provided, starts all auto-start servers.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runMCPStart,
}

var mcpStopCmd = &cobra.Command{
	Use:   "stop [server-name]",
	Short: "Stop MCP server(s)",
	Long:  "Stop one or all MCP servers. If no server name provided, stops all servers.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runMCPStop,
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available MCP tools",
	RunE:  runMCPList,
}

var mcpCallCmd = &cobra.Command{
	Use:   "call <tool-name> [args...]",
	Short: "Call an MCP tool",
	Long: `Call an MCP tool with JSON arguments.

Examples:
  rago mcp call mcp_sqlite_query '{"query": "SELECT * FROM users LIMIT 5"}'
  rago mcp call mcp_filesystem_read '{"path": "./README.md"}'
  rago mcp call mcp_git_status '{}'`,
	Args: cobra.MinimumNArgs(1),
	RunE: runMCPCall,
}

var mcpChatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "Chat with MCP tools directly (no RAG)",
	Long: `Direct chat with MCP tools bypassing RAG search.

This command allows you to interact with MCP tools without any document search.
The AI will only use MCP tools to answer your questions.

Examples:
  # Single message
  rago mcp chat "Create a table called users with id, name, email columns"
  
  # Interactive mode (no message provided)
  rago mcp chat`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMCPChat,
}

func init() {
	// Add subcommands to MCP parent command
	MCPCmd.AddCommand(mcpStatusCmd)
	MCPCmd.AddCommand(mcpStartCmd)
	MCPCmd.AddCommand(mcpStopCmd)
	MCPCmd.AddCommand(mcpListCmd)
	MCPCmd.AddCommand(mcpCallCmd)
	MCPCmd.AddCommand(mcpChatCmd)
	// Note: mcpChatAdvancedCmd is commented out in mcp_chat.go to avoid duplicate

	// Add flags
	mcpListCmd.Flags().StringP("server", "s", "", "Filter tools by server name")
	mcpListCmd.Flags().BoolP("json", "j", false, "Output in JSON format")
	mcpListCmd.Flags().BoolP("skip-failed", "k", true, "Continue even if some servers fail to start")

	mcpCallCmd.Flags().StringP("timeout", "t", "30s", "Call timeout duration")
	mcpCallCmd.Flags().BoolP("json", "j", false, "Output result in JSON format")

	// Chat command flags
	mcpChatCmd.Flags().Float64P("temperature", "T", 0.7, "Generation temperature")
	mcpChatCmd.Flags().IntP("max-tokens", "m", 30000, "Maximum generation length")
	mcpChatCmd.Flags().BoolP("show-thinking", "t", true, "Show AI thinking process")
	mcpChatCmd.Flags().StringSliceP("allowed-tools", "a", []string{}, "Comma-separated list of allowed tools")
}

func runMCPChat(cmd *cobra.Command, args []string) error {
	// Get flags
	temperature, _ := cmd.Flags().GetFloat64("temperature")
	maxTokens, _ := cmd.Flags().GetInt("max-tokens")
	showThinking, _ := cmd.Flags().GetBool("show-thinking")
	allowedTools, _ := cmd.Flags().GetStringSlice("allowed-tools")

	// Check if interactive mode (no arguments provided)
	if len(args) == 0 {
		return fmt.Errorf("interactive mode not available - please provide a message")
	}

	message := strings.Join(args, " ")

	if Cfg == nil {
		var err error
		Cfg, err = config.Load("")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	ctx := context.Background()

	// Get global LLM service
	llmService, err := services.GetGlobalLLM()
	if err != nil {
		return fmt.Errorf("failed to get global LLM service: %w", err)
	}

	// Create MCP tool manager
	mcpManager := mcp.NewMCPToolManager(&Cfg.MCP)

	// Ensure MCP servers are started (continue even if some fail)
	succeeded, failed := mcpManager.StartWithFailures(ctx)
	if len(failed) > 0 {
		fmt.Printf("⚠️  Warning: Failed to start %d server(s): %s\n", len(failed), strings.Join(failed, ", "))
	}
	if len(succeeded) == 0 {
		return fmt.Errorf("no MCP servers could be started - please check MCP server status")
	}

	// Wait a moment for servers to initialize
	time.Sleep(time.Second)

	// Get available tools
	toolsMap := mcpManager.ListTools()

	if len(toolsMap) == 0 {
		return fmt.Errorf("no MCP tools available - please check MCP server status")
	}

	fmt.Printf("🔧 Found %d MCP tools available\n", len(toolsMap))

	// Convert to slice for filtering
	var tools []*mcp.MCPToolWrapper
	for _, tool := range toolsMap {
		tools = append(tools, tool)
	}

	// Filter tools if allowed-tools is specified
	if len(allowedTools) > 0 {
		allowedSet := make(map[string]bool)
		for _, tool := range allowedTools {
			allowedSet[tool] = true
		}

		var filteredTools []*mcp.MCPToolWrapper
		for _, tool := range tools {
			if allowedSet[tool.Name()] {
				filteredTools = append(filteredTools, tool)
			}
		}
		tools = filteredTools
	}

	// Build tool definitions for LLM
	var toolDefinitions []domain.ToolDefinition
	for _, tool := range tools {
		definition := domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.Schema(),
			},
		}
		toolDefinitions = append(toolDefinitions, definition)
	}

	// Prepare messages
	messages := []domain.Message{
		{
			Role:    "user",
			Content: message,
		},
	}

	// Generation options
	think := showThinking
	opts := &domain.GenerationOptions{
		Temperature: temperature,
		MaxTokens:   int(maxTokens),
		Think:       &think,
	}

	// Call LLM with tools
	result, err := llmService.GenerateWithTools(ctx, messages, toolDefinitions, opts)
	if err != nil {
		return fmt.Errorf("failed to generate response: %w", err)
	}

	// Show thinking if enabled
	if showThinking && result.Content != "" {
		// The thinking content might be included in the content
		fmt.Printf("🤔 **Thinking included in response**\n\n")
	}

	// Handle tool calls
	if len(result.ToolCalls) > 0 {
		fmt.Printf("🔧 **Tool Calls:**\n")

		var toolResults []domain.Message
		for _, toolCall := range result.ToolCalls {
			fmt.Printf("- Calling `%s`\n", toolCall.Function.Name)

			// Execute tool call via MCP
			result, err := mcpManager.CallTool(ctx, toolCall.Function.Name, toolCall.Function.Arguments)
			if err != nil {
				fmt.Printf("  ❌ Error: %v\n", err)
				continue
			}

			// Format result
			var resultStr string
			if result.Data != nil {
				resultStr = fmt.Sprintf("%v", result.Data)
			} else if result.Success {
				resultStr = "Tool executed successfully"
			} else {
				resultStr = fmt.Sprintf("Tool execution failed: %s", result.Error)
			}

			fmt.Printf("  ✅ Result: %s\n", resultStr)

			// Add tool result for potential follow-up
			toolResults = append(toolResults, domain.Message{
				Role:       "tool",
				Content:    resultStr,
				ToolCallID: toolCall.ID,
			})
		}

		// If we have tool results, send follow-up request for final response
		if len(toolResults) > 0 {
			// Append assistant message with tool calls
			messages = append(messages, domain.Message{
				Role:      "assistant",
				Content:   result.Content,
				ToolCalls: result.ToolCalls,
			})

			// Append tool results
			messages = append(messages, toolResults...)

			followUpResult, err := llmService.GenerateWithTools(ctx, messages, toolDefinitions, opts)
			if err == nil {
				fmt.Printf("\n💬 **Final Response:**\n%s\n", followUpResult.Content)
			}
		}
	} else {
		// Direct response without tools
		fmt.Printf("💬 **Response:**\n%s\n", result.Content)
	}

	return nil
}

func runMCPStatus(cmd *cobra.Command, args []string) error {
	// Check runtime environments first
	envStatus := CheckMCPEnvironment()
	PrintMCPEnvironmentStatus(envStatus)

	if Cfg == nil {
		var err error
		Cfg, err = config.Load("")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	if !Cfg.MCP.Enabled {
		fmt.Println("\n❌ MCP is disabled in configuration")
		fmt.Println("   To enable: Set mcp.enabled = true in rago.toml")
		return nil
	}

	fmt.Println("\n⚙️  MCP Configuration")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Printf("   Enabled: %v\n", Cfg.MCP.Enabled)
	fmt.Printf("   Log Level: %s\n", Cfg.MCP.LogLevel)
	fmt.Printf("   Default Timeout: %v\n", Cfg.MCP.DefaultTimeout)
	fmt.Printf("   Max Concurrent: %d\n", Cfg.MCP.MaxConcurrentRequests)
	fmt.Printf("   Health Check Interval: %v\n", Cfg.MCP.HealthCheckInterval)

	// Create tool manager
	toolManager := mcp.NewMCPToolManager(&Cfg.MCP)
	defer func() {
		if err := toolManager.Close(); err != nil {
			// Only print error if it's not a signal-related termination
			if !strings.Contains(err.Error(), "signal: killed") {
				fmt.Printf("Warning: failed to clean up tool manager: %v\n", err)
			}
		}
	}()

	// Get server status
	serverStatus := toolManager.GetServerStatus()
	fmt.Printf("\n📊 Servers (%d configured):\n", len(Cfg.MCP.Servers))

	// Load server configurations first
	if err := Cfg.MCP.LoadServersFromJSON(); err != nil {
		return fmt.Errorf("failed to load server configurations: %w", err)
	}

	for _, serverConfig := range Cfg.MCP.LoadedServers {
		status := "❌ Stopped"
		if connected, exists := serverStatus[serverConfig.Name]; exists && connected {
			status = "✅ Running"
		}

		fmt.Printf("   - %s: %s\n", serverConfig.Name, status)
		fmt.Printf("     Description: %s\n", serverConfig.Description)
		fmt.Printf("     Command: %v\n", serverConfig.Command)
		fmt.Printf("     Auto-start: %v\n", serverConfig.AutoStart)

		if connected, exists := serverStatus[serverConfig.Name]; exists && connected {
			tools := toolManager.ListToolsByServer(serverConfig.Name)
			fmt.Printf("     Tools: %d available\n", len(tools))
		}
		fmt.Println()
	}

	return nil
}

func runMCPStart(cmd *cobra.Command, args []string) error {
	if Cfg == nil {
		var err error
		Cfg, err = config.Load("")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	if !Cfg.MCP.Enabled {
		return fmt.Errorf("MCP is disabled in configuration")
	}

	toolManager := mcp.NewMCPToolManager(&Cfg.MCP)
	defer func() {
		if err := toolManager.Close(); err != nil {
			// Only print error if it's not a signal-related termination
			if !strings.Contains(err.Error(), "signal: killed") {
				fmt.Printf("Warning: failed to clean up tool manager: %v\n", err)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if len(args) == 0 {
		// Start all auto-start servers
		fmt.Println("🚀 Starting MCP servers...")
		if err := toolManager.Start(ctx); err != nil {
			return fmt.Errorf("failed to start MCP servers: %w", err)
		}
		fmt.Println("✅ Auto-start MCP servers started successfully")
	} else {
		// Start specific server
		serverName := args[0]
		fmt.Printf("🚀 Starting MCP server: %s\n", serverName)
		if err := toolManager.StartServer(ctx, serverName); err != nil {
			return fmt.Errorf("failed to start server %s: %w", serverName, err)
		}

		// Show available tools
		tools := toolManager.ListToolsByServer(serverName)
		fmt.Printf("✅ Server %s started with %d tools\n", serverName, len(tools))
	}

	return nil
}

func runMCPStop(cmd *cobra.Command, args []string) error {
	if Cfg == nil {
		var err error
		Cfg, err = config.Load("")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	toolManager := mcp.NewMCPToolManager(&Cfg.MCP)
	defer func() {
		if err := toolManager.Close(); err != nil {
			// Only print error if it's not a signal-related termination
			if !strings.Contains(err.Error(), "signal: killed") {
				fmt.Printf("Warning: failed to clean up tool manager: %v\n", err)
			}
		}
	}()

	if len(args) == 0 {
		// Stop all servers
		fmt.Println("🛑 Stopping all MCP servers...")
		if err := toolManager.Close(); err != nil {
			return fmt.Errorf("failed to stop MCP servers: %w", err)
		}
		fmt.Println("✅ All MCP servers stopped")
	} else {
		// Stop specific server
		serverName := args[0]
		fmt.Printf("🛑 Stopping MCP server: %s\n", serverName)
		if err := toolManager.StopServer(serverName); err != nil {
			return fmt.Errorf("failed to stop server %s: %w", serverName, err)
		}
		fmt.Printf("✅ Server %s stopped\n", serverName)
	}

	return nil
}

func runMCPList(cmd *cobra.Command, args []string) error {
	if Cfg == nil {
		var err error
		Cfg, err = config.Load("")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	if !Cfg.MCP.Enabled {
		return fmt.Errorf("MCP is disabled in configuration")
	}

	toolManager := mcp.NewMCPToolManager(&Cfg.MCP)
	defer func() {
		if err := toolManager.Close(); err != nil {
			// Only print error if it's not a signal-related termination
			if !strings.Contains(err.Error(), "signal: killed") {
				fmt.Printf("Warning: failed to clean up tool manager: %v\n", err)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	skipFailed, _ := cmd.Flags().GetBool("skip-failed")

	// Start servers to get tools
	if skipFailed {
		succeeded, failed := toolManager.StartWithFailures(ctx)
		if len(failed) > 0 {
			fmt.Printf("⚠️  Warning: Failed to start %d server(s): %s\n", len(failed), strings.Join(failed, ", "))
		}
		if len(succeeded) > 0 {
			fmt.Printf("✅ Started %d server(s) successfully\n\n", len(succeeded))
		}
	} else {
		if err := toolManager.Start(ctx); err != nil {
			return fmt.Errorf("failed to start MCP servers: %w", err)
		}
	}

	serverFilter, _ := cmd.Flags().GetString("server")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	var tools map[string]*mcp.MCPToolWrapper
	if serverFilter != "" {
		tools = toolManager.ListToolsByServer(serverFilter)
	} else {
		tools = toolManager.ListTools()
	}

	if jsonOutput {
		llmTools := toolManager.GetToolsForLLM()
		output, err := json.MarshalIndent(llmTools, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal tools: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Human-readable output
	fmt.Printf("🔧 Available MCP Tools (%d total):\n\n", len(tools))

	serverGroups := make(map[string][]*mcp.MCPToolWrapper)
	for _, tool := range tools {
		serverName := tool.ServerName()
		serverGroups[serverName] = append(serverGroups[serverName], tool)
	}

	for serverName, serverTools := range serverGroups {
		fmt.Printf("📦 Server: %s (%d tools)\n", serverName, len(serverTools))
		for _, tool := range serverTools {
			fmt.Printf("   - %s\n", tool.Name())
			fmt.Printf("     %s\n", tool.Description())

			// Show schema if available
			schema := tool.Schema()
			if props, ok := schema["properties"].(map[string]interface{}); ok && len(props) > 0 {
				fmt.Printf("     Parameters: %s\n", formatSchemaParams(props))
			}
		}
		fmt.Println()
	}

	return nil
}

func runMCPCall(cmd *cobra.Command, args []string) error {
	if Cfg == nil {
		var err error
		Cfg, err = config.Load("")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	if !Cfg.MCP.Enabled {
		return fmt.Errorf("MCP is disabled in configuration")
	}

	toolName := args[0]
	timeoutStr, _ := cmd.Flags().GetString("timeout")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return fmt.Errorf("invalid timeout: %w", err)
	}

	// Parse arguments
	var toolArgs map[string]interface{}
	if len(args) > 1 {
		argsStr := strings.Join(args[1:], " ")
		if err := json.Unmarshal([]byte(argsStr), &toolArgs); err != nil {
			return fmt.Errorf("failed to parse arguments as JSON: %w", err)
		}
	} else {
		toolArgs = make(map[string]interface{})
	}

	toolManager := mcp.NewMCPToolManager(&Cfg.MCP)
	defer func() {
		if err := toolManager.Close(); err != nil {
			// Only print error if it's not a signal-related termination
			if !strings.Contains(err.Error(), "signal: killed") {
				fmt.Printf("Warning: failed to clean up tool manager: %v\n", err)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout+10*time.Second)
	defer cancel()

	// Start servers (continue even if some fail)
	succeeded, failed := toolManager.StartWithFailures(ctx)
	if len(failed) > 0 {
		fmt.Printf("⚠️  Warning: Failed to start %d server(s): %s\n", len(failed), strings.Join(failed, ", "))
		fmt.Printf("   Run 'rago mcp status --verbose' for more details\n")
	}
	if len(succeeded) == 0 {
		fmt.Printf("\n❌ No MCP servers could be started.\n\n")
		fmt.Printf("This usually happens when MCP server packages are not installed.\n")
		fmt.Printf("To install MCP servers via npx (they will be downloaded on first use):\n")
		fmt.Printf("  - The servers will be automatically downloaded when you run them\n")
		fmt.Printf("  - Make sure you have Node.js and npm installed\n\n")
		fmt.Printf("To check server status: rago mcp status --verbose\n")
		return fmt.Errorf("no MCP servers available")
	}

	// Call the tool
	fmt.Printf("🔍 Calling tool: %s\n", toolName)
	if len(toolArgs) > 0 {
		argsJSON, _ := json.MarshalIndent(toolArgs, "", "  ")
		fmt.Printf("📝 Arguments:\n%s\n\n", string(argsJSON))
	}

	callCtx, callCancel := context.WithTimeout(ctx, timeout)
	defer callCancel()

	result, err := toolManager.CallTool(callCtx, toolName, toolArgs)
	if err != nil {
		return fmt.Errorf("failed to call tool: %w", err)
	}

	if jsonOutput {
		output, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		fmt.Println(string(output))
		return nil
	}

	// Human-readable output
	if result.Success {
		fmt.Printf("✅ Tool call succeeded (took %v)\n", result.Duration)
		fmt.Printf("📊 Result:\n")

		if result.Data != nil {
			if dataJSON, err := json.MarshalIndent(result.Data, "", "  "); err == nil {
				fmt.Println(string(dataJSON))
			} else {
				fmt.Printf("%v\n", result.Data)
			}
		}
	} else {
		fmt.Printf("❌ Tool call failed (took %v)\n", result.Duration)
		fmt.Printf("💥 Error: %s\n", result.Error)
	}

	return nil
}

func formatSchemaParams(props map[string]interface{}) string {
	var params []string
	for name, prop := range props {
		if propMap, ok := prop.(map[string]interface{}); ok {
			paramType := "any"
			if t, exists := propMap["type"]; exists {
				paramType = fmt.Sprintf("%v", t)
			}
			params = append(params, fmt.Sprintf("%s:%s", name, paramType))
		}
	}
	return strings.Join(params, ", ")
}
