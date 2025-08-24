# RAGO 库使用指南

RAGO 不仅可以作为独立的 CLI 工具使用，还可以作为 Go 库集成到您的项目中，为您的应用程序提供强大的 RAG（检索增强生成）和工具调用能力。

## 🚀 快速开始

### 安装

```bash
go get github.com/liliang-cn/rago/lib
```

### 基础使用

```go
package main

import (
    "fmt"
    "log"

    rago "github.com/liliang-cn/rago/lib"
)

func main() {
    // 创建客户端
    client, err := rago.New("config.toml")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 基础查询
    response, err := client.Query("什么是机器学习？")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("答案: %s\n", response.Answer)
}
```

## 📚 完整 API 文档

### 客户端初始化

```go
// 从配置文件创建客户端
client, err := rago.New("path/to/config.toml")

// 从配置对象创建客户端
config := &config.Config{...}
client, err := rago.NewWithConfig(config)

// 记住关闭客户端以释放资源
defer client.Close()
```

### 📝 文档管理

```go
// 导入文本内容
err := client.IngestText("这是文档内容", "document-id")

// 导入文件
err := client.IngestFile("/path/to/document.md")

// 列出所有文档
documents, err := client.ListDocuments()

// 删除文档
err := client.DeleteDocument("document-id")

// 重置所有数据
err := client.Reset()
```

### 🔍 查询功能

```go
// 基础查询
response, err := client.Query("你的问题")

// 带过滤器的查询
filters := map[string]interface{}{
    "category": "tech",
    "author": "john",
}
response, err := client.QueryWithFilters("问题", filters)

// 流式查询
err := client.StreamQuery("问题", func(chunk string) {
    fmt.Print(chunk)
})

// 带过滤器的流式查询
err := client.StreamQueryWithFilters("问题", filters, func(chunk string) {
    fmt.Print(chunk)
})
```

### ⚙️ 工具调用功能

```go
// 启用工具的查询
response, err := client.QueryWithTools(
    "现在几点了？",
    []string{"datetime"},  // 允许的工具列表，空表示允许所有
    5,                     // 最大工具调用次数
)

// 直接执行工具
result, err := client.ExecuteTool("datetime", map[string]interface{}{
    "action": "now",
})

// 列出可用工具
tools := client.ListAvailableTools()  // 所有工具
enabled := client.ListEnabledTools()  // 仅启用的工具

// 获取工具统计信息
stats := client.GetToolStats()
```

### 🔧 系统管理

```go
// 检查系统状态
status := client.CheckStatus()
fmt.Printf("Ollama 可用: %v\n", status.OllamaAvailable)
fmt.Printf("模型: %s\n", status.LLMModel)

// 获取配置
config := client.GetConfig()
```

## 🛠️ 可用工具

RAGO 内置了多个强大的工具：

### 1. DateTime Tool (datetime)

- **功能**: 时间日期操作
- **用法**:
  ```go
  client.ExecuteTool("datetime", map[string]interface{}{
      "action": "now",
  })
  ```

### 2. File Operations Tool (file_operations)

- **功能**: 安全的文件系统操作
- **用法**:

  ```go
  // 读取文件
  client.ExecuteTool("file_operations", map[string]interface{}{
      "action": "read",
      "path":   "./README.md",
  })

  // 列出目录
  client.ExecuteTool("file_operations", map[string]interface{}{
      "action": "list",
      "path":   "./",
  })
  ```

### 3. RAG Search Tool (rag_search)

- **功能**: 知识库搜索
- **用法**:
  ```go
  client.ExecuteTool("rag_search", map[string]interface{}{
      "query": "机器学习",
      "top_k": 5,
  })
  ```

### 4. Document Info Tool (document_info)

- **功能**: 文档管理
- **用法**:
  ```go
  // 获取文档数量
  client.ExecuteTool("document_info", map[string]interface{}{
      "action": "count",
  })
  ```

### 5. SQL Query Tool (sql_query)

- **功能**: 安全的数据库查询
- **用法**:
  ```go
  client.ExecuteTool("sql_query", map[string]interface{}{
      "action":   "query",
      "database": "main",
      "sql":      "SELECT * FROM documents LIMIT 5",
  })
  ```

## 📋 响应格式

### QueryResponse

```go
type QueryResponse struct {
    Answer    string                 // 生成的答案
    Sources   []Chunk               // 相关的文档片段
    Elapsed   string                // 查询耗时
    ToolCalls []ExecutedToolCall   // 执行的工具调用（如果使用工具）
    ToolsUsed []string             // 使用的工具名称列表
}
```

### ToolResult

```go
type ToolResult struct {
    Success bool        // 是否执行成功
    Data    interface{} // 结果数据
    Error   string      // 错误信息（如果失败）
}
```

## ⚙️ 配置

创建 `config.toml` 文件：

```toml
[ollama]
base_url = "http://localhost:11434"
llm_model = "qwen3"
embedding_model = "nomic-embed-text"

[tools]
enabled = true

[tools.builtin.datetime]
enabled = true

[tools.builtin.file_operations]
enabled = true
[tools.builtin.file_operations.parameters]
allowed_paths = "./knowledge,./data,./examples"
max_file_size = "10485760"

[tools.builtin.rag_search]
enabled = true

[tools.builtin.document_info]
enabled = true
```

## 🔒 安全特性

- **路径限制**: 文件操作仅限于配置的允许路径
- **SQL 安全**: 仅允许 SELECT 查询，防止 SQL 注入
- **速率限制**: 可配置的工具调用频率限制
- **文件大小限制**: 防止处理过大的文件

## 📱 完整示例

查看 [examples/library_usage.go](examples/library_usage.go) 获取完整的使用示例。

## 🎯 集成场景

RAGO 库非常适合以下场景：

1. **智能客服系统**: 基于企业知识库回答用户问题
2. **文档问答应用**: 对大量文档进行智能检索和问答
3. **AI 助手**: 具备文件操作、时间查询等能力的智能助手
4. **知识管理系统**: 企业内部知识库的智能化管理
5. **自动化工具**: 结合 AI 和工具调用的自动化脚本

## 📞 支持

如有问题或建议，请访问 [GitHub Issues](https://github.com/liliang-cn/rago/issues)。
