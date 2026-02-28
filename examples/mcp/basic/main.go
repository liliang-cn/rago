// Package main demonstrates MCP (Model Context Protocol) tools integration.
//
// MCP allows RAGO to use external tools from MCP servers like filesystem,
// web search, databases, etc.
//
// Prerequisites:
//   - Configure mcpServers.json with MCP servers
//
// Usage:
//
//	go run examples/mcp/basic/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("=== MCP Tools Integration Example ===")
	fmt.Println()

	svc, err := agent.NewBuilder("mcp-demo").
		WithMCP().
		WithDebug(true).
		Build()
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer svc.Close()

	result, err := svc.Run(ctx, "List the files in the current directory using available tools")
	if err != nil {
		log.Fatalf("Run failed: %v", err)
	}

	fmt.Printf("\nResult: %v\n", result.FinalResult)
	fmt.Println("\n✅ MCP integration example completed!")
}
