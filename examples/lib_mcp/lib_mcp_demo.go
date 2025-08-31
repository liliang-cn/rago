package main

import (
	"context"
	"fmt"
	"log"
	"time"

	rago "github.com/liliang-cn/rago/client"
)

func main() {
	fmt.Println("🚀 Rago Library with MCP Integration Demo")
	
	// 创建rago客户端
	client, err := rago.New("config.example.toml")
	if err != nil {
		log.Fatalf("Failed to create rago client: %v", err)
	}
	defer client.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	
	// 启用MCP功能
	fmt.Println("\n🔧 Enabling MCP functionality...")
	if err := client.EnableMCP(ctx); err != nil {
		log.Printf("MCP not available: %v", err)
		fmt.Println("💡 Make sure MCP is enabled in config and servers are accessible")
		return
	}
	defer client.DisableMCP()
	
	// 演示1: 基本MCP工具使用
	fmt.Println("\n📋 Demo 1: Basic MCP Tool Usage")
	demonstrateBasicMCP(client)
	
	// 演示2: 数据库操作快捷方法
	fmt.Println("\n🗄️ Demo 2: Database Quick Operations")
	demonstrateDatabaseOps(client)
	
	// 演示3: LLM集成
	fmt.Println("\n🤖 Demo 3: LLM Integration")
	demonstrateLLMIntegration(client)
	
	// 演示4: 批量工具调用
	fmt.Println("\n⚡ Demo 4: Batch Tool Calls")
	demonstrateBatchCalls(client, ctx)
	
	// 演示5: 结合RAG和MCP
	fmt.Println("\n🔗 Demo 5: Combining RAG and MCP")
	demonstrateRAGWithMCP(client, ctx)
}

func demonstrateBasicMCP(client *rago.Client) {
	// 检查MCP状态
	if !client.IsMCPEnabled() {
		fmt.Println("   ❌ MCP is not enabled")
		return
	}
	
	// 列出MCP工具
	tools, err := client.ListMCPTools()
	if err != nil {
		fmt.Printf("   ❌ Failed to list tools: %v\n", err)
		return
	}
	
	fmt.Printf("   📦 Available MCP Tools: %d\n", len(tools))
	for i, tool := range tools[:min(3, len(tools))] {
		fmt.Printf("   %d. %s (%s): %s\n", i+1, tool.Name, tool.ServerName, tool.Description)
	}
	
	// 获取服务器状态
	status, err := client.GetMCPServerStatus()
	if err != nil {
		fmt.Printf("   ❌ Failed to get server status: %v\n", err)
		return
	}
	
	fmt.Printf("   🔄 Server Status:\n")
	for server, connected := range status {
		status := "❌ Disconnected"
		if connected {
			status = "✅ Connected"
		}
		fmt.Printf("      - %s: %s\n", server, status)
	}
}

func demonstrateDatabaseOps(client *rago.Client) {
	// 使用通用的MCP工具调用，而不是快捷方法
	fmt.Println("   📊 Listing database tables...")
	result, err := client.CallMCPToolWithTimeout("mcp_sqlite_list_tables", map[string]interface{}{}, 10*time.Second)
	if err != nil {
		fmt.Printf("   ❌ Failed to list tables: %v\n", err)
		return
	}
	
	if result.Success {
		fmt.Printf("   ✅ Tables: %v\n", result.Data)
	} else {
		fmt.Printf("   ❌ List tables failed: %s\n", result.Error)
		return
	}
	
	// 执行查询
	fmt.Println("   🔍 Executing sample query...")
	queryResult, err := client.CallMCPToolWithTimeout("mcp_sqlite_query", map[string]interface{}{
		"query": "SELECT name FROM sqlite_master WHERE type='table' LIMIT 3",
	}, 15*time.Second)
	if err != nil {
		fmt.Printf("   ❌ Query failed: %v\n", err)
		return
	}
	
	if queryResult.Success {
		fmt.Printf("   ✅ Query result: %v\n", queryResult.Data)
	} else {
		fmt.Printf("   ❌ Query failed: %s\n", queryResult.Error)
	}
	
	// 描述表结构
	if result.Success {
		fmt.Println("   📋 Describing table structure...")
		descResult, err := client.CallMCPToolWithTimeout("mcp_sqlite_describe_table", map[string]interface{}{
			"table": "test_table",
		}, 10*time.Second)
		if err != nil {
			fmt.Printf("   ❌ Describe failed: %v\n", err)
		} else if descResult.Success {
			fmt.Printf("   ✅ Table structure: %v\n", descResult.Data)
		}
	}
}

func demonstrateLLMIntegration(client *rago.Client) {
	// 获取LLM兼容的工具定义
	llmTools, err := client.GetMCPToolsForLLM()
	if err != nil {
		fmt.Printf("   ❌ Failed to get LLM tools: %v\n", err)
		return
	}
	
	fmt.Printf("   🤖 LLM-compatible tools: %d\n", len(llmTools))
	fmt.Println("   📝 Example tool definition for LLM:")
	
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
		
		fmt.Println("   💡 These tools can be passed directly to OpenAI, Claude, or other LLMs")
		fmt.Println("   💡 When LLM calls a tool, use client.CallMCPTool() to execute it")
	}
}

func demonstrateBatchCalls(client *rago.Client, ctx context.Context) {
	// 并行调用多个工具
	fmt.Println("   ⚡ Making parallel tool calls...")
	
	calls := []rago.ToolCall{
		{
			ToolName: "mcp_sqlite_list_tables",
			Args:     map[string]interface{}{},
		},
		{
			ToolName: "mcp_sqlite_current_database", 
			Args:     map[string]interface{}{},
		},
	}
	
	results, err := client.BatchCallMCPTools(ctx, calls)
	if err != nil {
		fmt.Printf("   ❌ Batch call failed: %v\n", err)
		return
	}
	
	fmt.Printf("   ✅ Batch call completed: %d results\n", len(results))
	for i, result := range results {
		fmt.Printf("   %d. Success: %v, Duration: %v\n", i+1, result.Success, result.Duration)
	}
}

func demonstrateRAGWithMCP(client *rago.Client, ctx context.Context) {
	fmt.Println("   🔗 Combining RAG search with MCP database queries...")
	
	// 1. 执行RAG搜索（假设有文档已索引）
	fmt.Println("   📚 Step 1: RAG search for documents...")
	ragResults, err := client.Query("rago configuration")
	if err != nil {
		fmt.Printf("   ⚠️  RAG search failed (may not have indexed documents): %v\n", err)
	} else {
		fmt.Printf("   ✅ Found relevant documents (answer length: %d chars)\n", len(ragResults.Answer))
	}
	
	// 2. 使用MCP查询相关的数据库信息
	fmt.Println("   🗄️ Step 2: Query database for additional context...")
	dbResult, err := client.CallMCPToolWithTimeout("mcp_sqlite_query", map[string]interface{}{
		"query": "SELECT COUNT(*) as total_tables FROM sqlite_master WHERE type='table'",
	}, 10*time.Second)
	if err != nil {
		fmt.Printf("   ❌ Database query failed: %v\n", err)
		return
	}
	
	if dbResult.Success {
		fmt.Printf("   ✅ Database context: %v\n", dbResult.Data)
	}
	
	// 3. 组合结果
	fmt.Println("   🎯 Step 3: Combining RAG and MCP results...")
	fmt.Println("   💡 In a real application, you would:")
	fmt.Println("      - Use RAG results as context for questions")
	fmt.Println("      - Use MCP tools to get live data from databases/APIs")
	fmt.Println("      - Combine both in LLM prompts for comprehensive answers")
	fmt.Println("      - Use MCP tools to perform actions based on RAG insights")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}