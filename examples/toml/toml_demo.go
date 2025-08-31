package main

import (
	"fmt"
	"log"
	
	"github.com/liliang-cn/rago/internal/config"
)

func main() {
	fmt.Println("🔧 Testing TOML Configuration Loading...")
	
	// 尝试加载示例配置文件
	cfg, err := config.Load("config.example.toml")
	if err != nil {
		log.Fatalf("Failed to load config.example.toml: %v", err)
	}
	
	fmt.Println("✅ Successfully loaded config.example.toml!")
	fmt.Printf("   MCP Enabled: %v\n", cfg.MCP.Enabled)
	fmt.Printf("   MCP Log Level: %s\n", cfg.MCP.LogLevel)
	fmt.Printf("   MCP Default Timeout: %v\n", cfg.MCP.DefaultTimeout)
	fmt.Printf("   MCP Max Concurrent: %d\n", cfg.MCP.MaxConcurrentRequests)
	fmt.Printf("   MCP Health Check Interval: %v\n", cfg.MCP.HealthCheckInterval)
	fmt.Printf("   MCP Servers Count: %d\n", len(cfg.MCP.Servers))
	
	if len(cfg.MCP.Servers) > 0 {
		fmt.Println("\n🔧 Configured MCP Servers:")
		for _, server := range cfg.MCP.Servers {
			fmt.Printf("   - %s: %s\n", server.Name, server.Description)
			fmt.Printf("     Command: %v\n", server.Command)
			fmt.Printf("     Args: %v\n", server.Args)
			fmt.Printf("     Auto-start: %v\n", server.AutoStart)
		}
	}
	
	fmt.Println("\n🎉 TOML Configuration Support Verified!")
}