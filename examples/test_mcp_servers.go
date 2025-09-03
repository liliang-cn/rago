package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
)

func main() {
	fmt.Println("🧪 Testing MCP Server Integration")
	fmt.Println("==================================")
	fmt.Println()

	// Create MCP configuration
	mcpConfig := &config.MCPConfig{
		ServersConfigPath: "mcpServers.json",
		DefaultTimeout:    30 * time.Second,
	}

	// Initialize server manager
	manager, err := mcp.NewServerManager(mcpConfig)
	if err != nil {
		log.Fatalf("Failed to create server manager: %v", err)
	}

	// Get all server status
	fmt.Println("📊 Server Configuration Status:")
	fmt.Println("------------------------------")
	allStatus := manager.GetAllServerStatus()
	
	for name, status := range allStatus {
		icon := "⚪"
		switch status {
		case mcp.ServerStatusRunning:
			icon = "🟢"
		case mcp.ServerStatusStopped:
			icon = "🔴"
		case mcp.ServerStatusError:
			icon = "🟡"
		}
		
		info := manager.GetServerInfo(name)
		desc := ""
		if d, ok := info["description"].(string); ok {
			desc = d
		}
		
		fmt.Printf("%s %-20s %s\n", icon, name, desc)
	}

	fmt.Println()
	fmt.Println("🚀 Starting MCP Servers...")
	fmt.Println("---------------------------")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// List of servers to test
	servers := []string{"filesystem", "fetch", "memory", "time"}

	for _, serverName := range servers {
		fmt.Printf("Starting %s... ", serverName)
		
		err := manager.StartServer(ctx, serverName)
		if err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			continue
		}
		
		// Give server time to initialize
		time.Sleep(2 * time.Second)
		
		// Check status
		status, _ := manager.GetServerStatus(serverName)
		if status == mcp.ServerStatusRunning {
			fmt.Printf("✅ Running\n")
			
			// Try to get tools from the server
			client, err := manager.GetServerClient(serverName)
			if err == nil {
				tools, err := client.ListTools()
				if err == nil && len(tools) > 0 {
					fmt.Printf("  Found %d tools:\n", len(tools))
					for i, tool := range tools {
						if i < 3 { // Show first 3 tools
							fmt.Printf("    - %s: %s\n", tool.Name, tool.Description)
						}
					}
					if len(tools) > 3 {
						fmt.Printf("    ... and %d more\n", len(tools)-3)
					}
				}
			}
		} else {
			fmt.Printf("⚠️  Status: %s\n", status)
		}
	}

	fmt.Println()
	fmt.Println("🛠️  Testing Tool Execution...")
	fmt.Println("----------------------------")

	// Test time server
	fmt.Println("Testing time server:")
	client, err := manager.GetServerClient("time")
	if err != nil {
		fmt.Printf("  ❌ Failed to get time client: %v\n", err)
	} else {
		// Call a time tool
		result, err := client.CallTool(ctx, "get_current_time", map[string]interface{}{
			"timezone": "UTC",
		})
		if err != nil {
			fmt.Printf("  ❌ Failed to get time: %v\n", err)
		} else {
			fmt.Printf("  ✅ Current UTC time: %v\n", result.Content)
		}
	}

	// Test memory server
	fmt.Println("\nTesting memory server:")
	client, err = manager.GetServerClient("memory")
	if err != nil {
		fmt.Printf("  ❌ Failed to get memory client: %v\n", err)
	} else {
		// Store a value
		_, err = client.CallTool(ctx, "store", map[string]interface{}{
			"key":   "test_key",
			"value": "Hello from RAGO!",
		})
		if err != nil {
			fmt.Printf("  ❌ Failed to store value: %v\n", err)
		} else {
			fmt.Println("  ✅ Stored value successfully")
			
			// Retrieve the value
			result, err := client.CallTool(ctx, "retrieve", map[string]interface{}{
				"key": "test_key",
			})
			if err != nil {
				fmt.Printf("  ❌ Failed to retrieve value: %v\n", err)
			} else {
				fmt.Printf("  ✅ Retrieved value: %v\n", result.Content)
			}
		}
	}

	fmt.Println()
	fmt.Println("🔧 Listing All Available Tools...")
	fmt.Println("---------------------------------")
	
	allTools, err := manager.ListAvailableTools()
	if err != nil {
		fmt.Printf("❌ Failed to list tools: %v\n", err)
	} else {
		fmt.Printf("Found %d total tools across all servers:\n", len(allTools))
		
		// Group tools by server
		toolsByServer := make(map[string][]string)
		for _, tool := range allTools {
			server := "unknown"
			if s, ok := tool.Metadata["server"].(string); ok {
				server = s
			}
			toolsByServer[server] = append(toolsByServer[server], tool.Name)
		}
		
		for server, tools := range toolsByServer {
			fmt.Printf("\n  %s (%d tools):\n", server, len(tools))
			for i, tool := range tools {
				if i < 5 {
					fmt.Printf("    • %s\n", tool)
				}
			}
			if len(tools) > 5 {
				fmt.Printf("    ... and %d more\n", len(tools)-5)
			}
		}
	}

	fmt.Println()
	fmt.Println("🛑 Stopping All Servers...")
	fmt.Println("-------------------------")
	
	if err := manager.StopAllServers(); err != nil {
		fmt.Printf("⚠️  Some servers failed to stop: %v\n", err)
	} else {
		fmt.Println("✅ All servers stopped successfully")
	}

	fmt.Println()
	fmt.Println("✨ MCP Server Integration Test Complete!")
}