package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/internal/mcp"
)

// MCP 集成概念验证 - 使用 rago 的 MCP 客户端
func main() {
	// 创建 MCP 配置
	config := &mcp.Config{
		Enabled:                true,
		LogLevel:               "info",
		DefaultTimeout:         30 * time.Second,
		MaxConcurrentRequests:  10,
		HealthCheckInterval:    60 * time.Second,
		Servers: []mcp.ServerConfig{
			{
				Name:             "sqlite",
				Description:      "SQLite database operations",
				Command:          []string{"node", "./mcp-sqlite-server/dist/index.js"},
				Args:             []string{"--allowed-dir", "./data"},
				WorkingDir:       "./data",
				Env:              map[string]string{"DEBUG": "mcp:*"},
				AutoStart:        true,
				RestartOnFailure: true,
				MaxRestarts:      5,
				RestartDelay:     5 * time.Second,
				Capabilities:     []string{"query", "execute", "list"},
			},
		},
	}

	// 创建 MCP 管理器
	manager := mcp.NewManager(config)
	defer manager.Close()

	fmt.Println("🚀 Starting MCP integration test...")

	// 启动 SQLite MCP 服务器
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("📡 Connecting to SQLite MCP server...")
	client, err := manager.StartServer(ctx, "sqlite")
	if err != nil {
		log.Printf("❌ Failed to start SQLite MCP server: %v", err)
		fmt.Println("\n💡 Make sure mcp-sqlite-server is installed and available:")
		fmt.Println("   npm install -g @modelcontextprotocol/server-sqlite")
		fmt.Println("   Or clone: https://github.com/liliang-cn/mcp-sqlite-server")
		return
	}

	// 获取服务器信息
	if serverInfo := client.GetServerInfo(); serverInfo != nil {
		fmt.Printf("✅ Connected to server: %s v%s\n", 
			serverInfo.Name, 
			serverInfo.Version,
		)
	}

	// 获取可用工具列表
	tools := client.GetTools()
	fmt.Printf("🔧 Available tools (%d):\n", len(tools))
	for name, tool := range tools {
		fmt.Printf("   - %s: %s\n", name, tool.Description)
	}

	// 测试工具调用
	if len(tools) > 0 {
		// 找一个查询工具
		var queryTool string
		for name := range tools {
			if name == "sqlite_query" || name == "query" {
				queryTool = name
				break
			}
		}

		if queryTool == "" {
			// 使用第一个可用工具
			for name := range tools {
				queryTool = name
				break
			}
		}

		fmt.Printf("\n🔍 Testing tool call: %s\n", queryTool)
		result, err := client.CallTool(ctx, queryTool, map[string]interface{}{
			"sql": "SELECT name FROM sqlite_master WHERE type='table' LIMIT 5",
		})

		if err != nil {
			log.Printf("❌ Tool call failed: %v", err)
		} else if !result.Success {
			log.Printf("❌ Tool returned error: %s", result.Error)
		} else {
			fmt.Printf("✅ Tool result:\n%v\n", result.Data)
		}
	}

	// 测试管理器功能
	fmt.Printf("\n📊 Manager status:\n")
	clients := manager.ListClients()
	fmt.Printf("   Active clients: %d\n", len(clients))
	for name, client := range clients {
		fmt.Printf("   - %s: connected=%v\n", name, client.IsConnected())
	}

	fmt.Println("\n✨ MCP integration test completed!")
}