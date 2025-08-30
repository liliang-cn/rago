package rago

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/liliang-cn/rago/internal/config"
	"github.com/liliang-cn/rago/internal/mcp"
)

var mcpCmd = &cobra.Command{
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

func init() {
	RootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpStatusCmd)
	mcpCmd.AddCommand(mcpStartCmd)
	mcpCmd.AddCommand(mcpStopCmd)
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpCallCmd)

	// Add flags
	mcpListCmd.Flags().StringP("server", "s", "", "Filter tools by server name")
	mcpListCmd.Flags().BoolP("json", "j", false, "Output in JSON format")
	
	mcpCallCmd.Flags().StringP("timeout", "t", "30s", "Call timeout duration")
	mcpCallCmd.Flags().BoolP("json", "j", false, "Output result in JSON format")
}

func runMCPStatus(cmd *cobra.Command, args []string) error {
	if cfg == nil {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	if !cfg.MCP.Enabled {
		fmt.Println("❌ MCP is disabled in configuration")
		return nil
	}

	fmt.Println("🔧 MCP Status")
	fmt.Printf("   Enabled: %v\n", cfg.MCP.Enabled)
	fmt.Printf("   Log Level: %s\n", cfg.MCP.LogLevel)
	fmt.Printf("   Default Timeout: %v\n", cfg.MCP.DefaultTimeout)
	fmt.Printf("   Max Concurrent: %d\n", cfg.MCP.MaxConcurrentRequests)
	fmt.Printf("   Health Check Interval: %v\n", cfg.MCP.HealthCheckInterval)
	
	// Create tool manager
	toolManager := mcp.NewMCPToolManager(&cfg.MCP)
	defer toolManager.Close()
	
	// Get server status
	serverStatus := toolManager.GetServerStatus()
	fmt.Printf("\n📊 Servers (%d configured):\n", len(cfg.MCP.Servers))
	
	for _, serverConfig := range cfg.MCP.Servers {
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
	if cfg == nil {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	if !cfg.MCP.Enabled {
		return fmt.Errorf("MCP is disabled in configuration")
	}

	toolManager := mcp.NewMCPToolManager(&cfg.MCP)
	defer toolManager.Close()
	
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
	if cfg == nil {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	toolManager := mcp.NewMCPToolManager(&cfg.MCP)
	defer toolManager.Close()

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
	if cfg == nil {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	if !cfg.MCP.Enabled {
		return fmt.Errorf("MCP is disabled in configuration")
	}

	toolManager := mcp.NewMCPToolManager(&cfg.MCP)
	defer toolManager.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start servers to get tools
	if err := toolManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start MCP servers: %w", err)
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
	if cfg == nil {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	if !cfg.MCP.Enabled {
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

	toolManager := mcp.NewMCPToolManager(&cfg.MCP)
	defer toolManager.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout+10*time.Second)
	defer cancel()

	// Start servers
	if err := toolManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start MCP servers: %w", err)
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