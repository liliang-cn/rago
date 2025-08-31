# RAGO Library Usage Guide

RAGO can be used not only as a standalone CLI tool but also as a Go library integrated into your projects, providing powerful RAG (Retrieval-Augmented Generation) and tool-calling capabilities for your applications.

## 🚀 Quick Start

### Installation

```bash
go get github.com/liliang-cn/rago/lib
```

### Basic Usage

```go
package main

import (
    "fmt"
    "log"

    rago "github.com/liliang-cn/rago/lib"
)

func main() {
    // Create a client
    client, err := rago.New("rago.toml")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Basic query
    response, err := client.Query("What is machine learning?")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Answer: %s\n", response.Answer)
}
```

## 📚 Full API Documentation

### Client Initialization

```go
// Create a client from a configuration file
client, err := rago.New("path/to/rago.toml")

// Create a client from a configuration object
config := &config.Config{...}
client, err := rago.NewWithConfig(config)

// Remember to close the client to release resources
def client.Close()
```

### 📝 Document Management

```go
// Ingest text content
err := client.IngestText("This is the document content", "document-id")

// Ingest a file
err := client.IngestFile("/path/to/document.md")

// List all documents
documents, err := client.ListDocuments()

// Delete a document
err := client.DeleteDocument("document-id")

// Reset all data
err := client.Reset()
```

### 🔍 Query Functions

```go
// Basic query
response, err := client.Query("Your question")

// Query with filters
filters := map[string]interface{}{
    "category": "tech",
    "author": "john",
}
response, err := client.QueryWithFilters("Question", filters)

// Streaming query
err := client.StreamQuery("Question", func(chunk string) {
    fmt.Print(chunk)
})

// Streaming query with filters
err := client.StreamQueryWithFilters("Question", filters, func(chunk string) {
    fmt.Print(chunk)
})
```

### 🧠 Direct LLM Functions

```go
// Direct LLM generation
req := rago.LLMGenerateRequest{
	Prompt:      "Hello, world!",
	Temperature: 0.7,
	MaxTokens:   100,
}
resp, err := client.LLMGenerate(context.Background(), req)

// Streaming LLM generation
streamReq := rago.LLMGenerateRequest{...}
err := client.LLMGenerateStream(context.Background(), streamReq, func(chunk string) {
    fmt.Print(chunk)
})

// LLM chat
chatReq := rago.LLMChatRequest{
	Messages: []rago.ChatMessage{
		{Role: "user", Content: "Hello!"},
	},
}
chatResp, err := client.LLMChat(context.Background(), chatReq)

// Streaming LLM chat
streamChatReq := rago.LLMChatRequest{...}
err := client.LLMChatStream(context.Background(), streamChatReq, func(chunk string) {
    fmt.Print(chunk)
})
```

### ⚙️ Tool Calling Functions

```go
// Query with tools enabled
response, err := client.QueryWithTools(
    "What time is it?",
    []string{"datetime"},  // List of allowed tools, empty means all are allowed
    5,                     // Maximum number of tool calls
)

// Execute a tool directly
result, err := client.ExecuteTool("datetime", map[string]interface{}{
    "action": "now",
})

// List available tools
tools := client.ListAvailableTools()  // All tools
enabled := client.ListEnabledTools()  // Only enabled tools

// Get tool statistics
stats := client.GetToolStats()
```

### 🔧 System Management

```go
// Check system status
status := client.CheckStatus()
fmt.Printf("Providers Available: %v\n", status.ProvidersAvailable)
fmt.Printf("LLM Provider: %s\n", status.LLMProvider)

// Get configuration
config := client.GetConfig()
```

## 🛠️ Available Tools

RAGO comes with several powerful built-in tools:

### 1. DateTime Tool (datetime)

- **Functionality**: Date and time operations
- **Usage**:
  ```go
  client.ExecuteTool("datetime", map[string]interface{}{
      "action": "now",
  })
  ```

### 2. File Operations Tool (file_operations)

- **Functionality**: Secure file system operations
- **Usage**:

  ```go
  // Read a file
  client.ExecuteTool("file_operations", map[string]interface{}{
      "action": "read",
      "path":   "./README.md",
  })

  // List a directory
  client.ExecuteTool("file_operations", map[string]interface{}{
      "action": "list",
      "path":   "./",
  })
  ```

### 3. RAG Search Tool (rag_search)

- **Functionality**: Knowledge base search
- **Usage**:
  ```go
  client.ExecuteTool("rag_search", map[string]interface{}{
      "query": "machine learning",
      "top_k": 5,
  })
  ```

### 4. Document Info Tool (document_info)

- **Functionality**: Document management
- **Usage**:
  ```go
  // Get document count
  client.ExecuteTool("document_info", map[string]interface{}{
      "action": "count",
  })
  ```

### 5. SQL Query Tool (sql_query)

- **Functionality**: Secure database queries
- **Usage**:
  ```go
  client.ExecuteTool("sql_query", map[string]interface{}{
      "action":   "query",
      "database": "main",
      "sql":      "SELECT * FROM documents LIMIT 5",
  })
  ```

## 📋 Response Formats

### QueryResponse

```go
type QueryResponse struct {
    Answer    string                 // Generated answer
    Sources   []Chunk               // Relevant document chunks
    Elapsed   string                // Query duration
    ToolCalls []ExecutedToolCall   // Executed tool calls (if tools are used)
    ToolsUsed []string             // List of tool names used
}
```

### ToolResult

```go
type ToolResult struct {
    Success bool        // Whether the execution was successful
    Data    interface{} // Result data
    Error   string      // Error message (if failed)
}
```

## ⚙️ Configuration

Create a `rago.toml` file:

```toml
[providers]
default_llm = "ollama"
default_embedder = "ollama"

[llm.ollama]
base_url = "http://localhost:11434"
model = "qwen3"

[embedder.ollama]
base_url = "http://localhost:11434"
model = "nomic-embed-text"

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

## 🔒 Security Features

- **Path Restriction**: File operations are restricted to configured allowed paths.
- **SQL Safety**: Only SELECT queries are allowed to prevent SQL injection.
- **Rate Limiting**: Configurable rate limiting for tool calls.
- **File Size Limit**: Prevents processing of overly large files.

## 📱 Complete Example

Check out [examples/library_usage.go](examples/library_usage.go) for a complete usage example.

## 🎯 Integration Scenarios

The RAGO library is ideal for the following scenarios:

1. **Intelligent Customer Service Systems**: Answering user questions based on a corporate knowledge base.
2. **Document Q&A Applications**: Intelligent search and Q&A for large volumes of documents.
3. **AI Assistants**: Smart assistants with capabilities like file operations and time queries.
4. **Knowledge Management Systems**: Intelligent management of internal corporate knowledge bases.
5. **Automation Tools**: Automation scripts combining AI and tool calls.

## 📞 Support

For questions or suggestions, please visit [GitHub Issues](https://github.com/liliang-cn/rago/issues).
