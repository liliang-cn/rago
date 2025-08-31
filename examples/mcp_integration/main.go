package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/pkg/config"
	"github.com/liliang-cn/rago/pkg/mcp"
)

// 演示如何在CLI和lib中使用MCP工具
func main() {
	fmt.Println("🤖 MCP Tool Integration Demo")
	
	// 加载配置
	cfg, err := config.Load("config.example.toml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	if !cfg.MCP.Enabled {
		fmt.Println("❌ MCP is disabled in config.example.toml")
		fmt.Println("💡 Enable MCP by setting enabled = true in the [mcp] section")
		return
	}
	
	// 演示1: 直接使用MCP API
	fmt.Println("\n🔧 Demo 1: Direct MCP API Usage")
	if err := demonstrateDirectAPI(&cfg.MCP); err != nil {
		log.Printf("Direct API demo failed: %v", err)
	}
	
	// 演示2: 使用简化的Library API
	fmt.Println("\n📚 Demo 2: Simplified Library API Usage")  
	if err := demonstrateLibraryAPI(&cfg.MCP); err != nil {
		log.Printf("Library API demo failed: %v", err)
	}
	
	// 演示3: LLM集成格式
	fmt.Println("\n🤖 Demo 3: LLM Integration Format")
	if err := demonstrateLLMIntegration(&cfg.MCP); err != nil {
		log.Printf("LLM integration demo failed: %v", err)
	}
	
	// 演示4: CLI命令等效操作
	fmt.Println("\n💻 Demo 4: CLI Command Equivalents")
	demonstrateCLIEquivalents()
}

func demonstrateDirectAPI(config *mcp.Config) error {
	// 创建MCP工具管理器
	toolManager := mcp.NewMCPToolManager(config)
	defer toolManager.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// 启动MCP服务器
	if err := toolManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start MCP servers: %w", err)
	}
	
	// 列出所有工具
	tools := toolManager.ListTools()
	fmt.Printf("   Found %d MCP tools\n", len(tools))
	
	// 显示每个服务器的工具
	serverGroups := make(map[string]int)
	for _, tool := range tools {
		serverGroups[tool.ServerName()]++
	}
	
	for server, count := range serverGroups {
		fmt.Printf("   - %s: %d tools\n", server, count)
	}
	
	// 尝试调用SQLite工具
	if sqliteTool, exists := tools["mcp_sqlite_query"]; exists {
		fmt.Printf("   Calling %s...\n", sqliteTool.Name())
		
		result, err := sqliteTool.Call(ctx, map[string]interface{}{
			"query": "SELECT name FROM sqlite_master WHERE type='table' LIMIT 3",
		})
		
		if err != nil {
			return fmt.Errorf("tool call failed: %w", err)
		}
		
		if result.Success {
			fmt.Printf("   ✅ Success: %v\n", result.Data)
		} else {
			fmt.Printf("   ❌ Failed: %s\n", result.Error)
		}
	}
	
	return nil
}

func demonstrateLibraryAPI(config *mcp.Config) error {
	// 使用简化的Library API
	api := mcp.NewMCPLibraryAPI(config)
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// 启动服务
	if err := api.Start(ctx); err != nil {
		return fmt.Errorf("failed to start MCP API: %w", err)
	}
	defer api.Stop()
	
	// 列出工具
	tools := api.ListTools()
	fmt.Printf("   Available tools: %d\n", len(tools))
	
	for _, tool := range tools[:min(3, len(tools))] {
		fmt.Printf("   - %s (%s): %s\n", tool.Name, tool.ServerName, tool.Description)
	}
	
	// 快速调用示例
	if len(tools) > 0 {
		firstTool := tools[0]
		fmt.Printf("   Trying quick call to %s...\n", firstTool.Name)
		
		// 使用QuickCall方法
		result, err := api.QuickCall(firstTool.Name, mcp.QuickCallOptions{
			Timeout: 10 * time.Second,
			Args:    map[string]interface{}{},
		})
		
		if err != nil {
			fmt.Printf("   ❌ Quick call failed: %v\n", err)
		} else {
			fmt.Printf("   ✅ Quick call result: success=%v\n", result.Success)
		}
	}
	
	return nil
}

func demonstrateLLMIntegration(config *mcp.Config) error {
	api := mcp.NewMCPLibraryAPI(config)
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := api.Start(ctx); err != nil {
		return fmt.Errorf("failed to start MCP API: %w", err)
	}
	defer api.Stop()
	
	// 获取LLM格式的工具定义
	llmTools := api.GetToolsForLLMIntegration()
	
	fmt.Printf("   LLM-compatible tools: %d\n", len(llmTools))
	fmt.Println("   Example LLM tool definition:")
	
	if len(llmTools) > 0 {
		tool := llmTools[0]
		fmt.Printf("   {\n")
		fmt.Printf("     \"type\": \"%v\",\n", tool["type"])
		if fn, ok := tool["function"].(map[string]interface{}); ok {
			fmt.Printf("     \"function\": {\n")
			fmt.Printf("       \"name\": \"%v\",\n", fn["name"])
			fmt.Printf("       \"description\": \"%v\"\n", fn["description"])
			fmt.Printf("     }\n")
		}
		fmt.Printf("   }\n")
	}
	
	return nil
}

func demonstrateCLIEquivalents() {
	fmt.Println("   CLI commands you can use:")
	fmt.Println("   rago mcp status              # Show MCP server status")
	fmt.Println("   rago mcp start               # Start all auto-start servers")
	fmt.Println("   rago mcp start sqlite        # Start specific server")
	fmt.Println("   rago mcp list                # List all available tools")
	fmt.Println("   rago mcp list -s sqlite      # List tools from sqlite server")
	fmt.Println("   rago mcp list --json         # List tools in JSON format")
	fmt.Println("   rago mcp call mcp_sqlite_query '{\"query\": \"SELECT 1\"}'")
	fmt.Println("   rago mcp stop               # Stop all servers")
	fmt.Println("   rago mcp stop sqlite        # Stop specific server")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}