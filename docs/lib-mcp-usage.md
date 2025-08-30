# 📚 Rago Library MCP 使用指南

## 🚀 快速开始

```go
package main

import (
    "context"
    "fmt" 
    "time"
    
    rago "github.com/liliang-cn/rago/lib"
)

func main() {
    // 1. 创建rago客户端
    client, err := rago.New("config.toml")
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    // 2. 启用MCP功能
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := client.EnableMCP(ctx); err != nil {
        panic(err)
    }
    defer client.DisableMCP()
    
    // 3. 使用MCP工具
    result, err := client.MCPQuickQuery("SELECT * FROM users LIMIT 5")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Query result: %v\n", result.Data)
}
```

## 🔧 主要API方法

### 基础方法

```go
// 启用/禁用MCP
client.EnableMCP(ctx)                    // 启用MCP功能
client.DisableMCP()                      // 禁用MCP功能
client.IsMCPEnabled()                    // 检查MCP是否启用

// 工具管理
tools, _ := client.ListMCPTools()        // 列出所有MCP工具
status, _ := client.GetMCPServerStatus() // 获取服务器状态
```

### 工具调用

```go
// 基本调用
result, err := client.CallMCPTool(ctx, "mcp_sqlite_query", map[string]interface{}{
    "query": "SELECT * FROM users WHERE active = 1",
})

// 带超时调用  
result, err := client.CallMCPToolWithTimeout("mcp_sqlite_query", args, 10*time.Second)

// 批量调用
calls := []rago.ToolCall{
    {ToolName: "mcp_sqlite_list_tables", Args: map[string]interface{}{}},
    {ToolName: "mcp_sqlite_current_database", Args: map[string]interface{}{}},
}
results, err := client.BatchCallMCPTools(ctx, calls)
```

### 数据库快捷方法

```go
// SQLite操作快捷方法
result, _ := client.MCPQuickQuery("SELECT COUNT(*) FROM users")
result, _ := client.MCPQuickExecute("INSERT INTO users (name) VALUES ('Alice')")
result, _ := client.MCPListTables()
result, _ := client.MCPDescribeTable("users")
```

### LLM集成

```go
// 获取LLM兼容的工具定义
llmTools, err := client.GetMCPToolsForLLM()
// 返回OpenAI function calling格式

// 示例: 与LLM集成使用
for _, tool := range llmTools {
    fmt.Printf("Tool: %s\n", tool["function"].(map[string]interface{})["name"])
}

// LLM调用工具后执行
result, err := client.CallMCPTool(ctx, toolName, llmArgs)
```

## 🎯 使用场景

### 1. 数据库查询增强RAG

```go
// 结合RAG搜索和数据库查询
ragResult, _ := client.Query("用户管理相关问题")
dbResult, _ := client.MCPQuickQuery("SELECT COUNT(*) FROM users WHERE created_at > DATE('now', '-7 days')")

// 组合结果提供更完整的答案
```

### 2. 动态工具调用

```go
// 根据用户需求动态调用不同工具
switch userRequest {
case "list_files":
    result, _ := client.CallMCPTool(ctx, "mcp_filesystem_list", args)
case "git_status":
    result, _ := client.CallMCPTool(ctx, "mcp_git_status", args) 
case "web_search":
    result, _ := client.CallMCPTool(ctx, "mcp_brave_search", args)
}
```

### 3. 批量数据处理

```go
// 并行执行多个数据库操作
calls := []rago.ToolCall{
    {ToolName: "mcp_sqlite_query", Args: map[string]interface{}{"query": "SELECT * FROM users"}},
    {ToolName: "mcp_sqlite_query", Args: map[string]interface{}{"query": "SELECT * FROM orders"}},
    {ToolName: "mcp_sqlite_query", Args: map[string]interface{}{"query": "SELECT * FROM products"}},
}

results, _ := client.BatchCallMCPTools(ctx, calls)
```

## ⚙️ 配置

在 `config.toml` 中启用MCP:

```toml
[mcp]
enabled = true
log_level = "info"
default_timeout = "30s"

[[mcp.servers]]
name = "sqlite"
description = "SQLite database operations"
command = ["mcp-sqlite-server"]
auto_start = true
```

## 🎉 特性

✅ **简单易用** - 一行代码启用MCP功能  
✅ **完整集成** - 与现有RAG功能无缝结合  
✅ **异步支持** - 支持批量和并发工具调用  
✅ **LLM就绪** - 工具定义直接兼容OpenAI等LLM  
✅ **类型安全** - 完整的Go类型定义  
✅ **错误处理** - 健壮的错误处理和超时机制  

现在rago不仅是RAG系统，更是完整的AI Agent平台！🤖