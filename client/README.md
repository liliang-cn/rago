# RAGO Library Usage Guide

RAGO can be used not only as a standalone CLI tool but also as a Go library integrated into your projects, providing powerful RAG (Retrieval-Augmented Generation) and MCP (Model Context Protocol) tool integration for your applications.

## 🚀 Quick Start

### Installation

```bash
go get github.com/liliang-cn/rago/v2/client
```

### Basic Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/liliang-cn/rago/v2/client"
)

func main() {
    // Create a client
    client, err := client.New("rago.toml")
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
client, err := client.New("path/to/rago.toml")

// Create a client from a configuration object
config := &config.Config{...}
client, err := client.NewWithConfig(config)

// Remember to close the client to release resources
defer client.Close()
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

### 🛠️ MCP Tool Integration

RAGO now uses MCP (Model Context Protocol) for tool integration, providing enhanced functionality and extensibility.

```go
// Enable MCP functionality
ctx := context.Background()
err := client.EnableMCP(ctx)
if err != nil {
    log.Fatal(err)
}

// Check if MCP is enabled
if client.IsMCPEnabled() {
    fmt.Println("MCP is ready!")
}

// List available MCP tools
tools, err := client.ListMCPTools()
if err != nil {
    log.Fatal(err)
}

// Call an MCP tool directly
result, err := client.CallMCPTool(ctx, "filesystem_read", map[string]interface{}{
    "path": "./README.md",
})

// Query with MCP tools enabled (automatic tool calling)
response, err := client.QueryWithMCP("List files in the current directory")

// Chat with MCP tools (bypassing RAG)
mcpResponse, err := client.ChatWithMCP("What's the current time?", &client.MCPChatOptions{
    Temperature:  0.7,
    MaxTokens:    1000,
    AllowedTools: []string{"datetime", "filesystem"},
})
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

## 🛠️ MCP Server Configuration

RAGO uses MCP servers to provide tool functionality. Configure your MCP servers in your configuration file:

### Setting up MCP Servers

Create or update your `mcpServers.json` file:

```json
{
  "filesystem": {
    "command": "npx",
    "args": [
      "-y",
      "@modelcontextprotocol/server-filesystem", 
      "/path/to/allowed/directory"
    ]
  },
  "sqlite": {
    "command": "npx",
    "args": [
      "-y",
      "@modelcontextprotocol/server-sqlite",
      "/path/to/database.db"
    ]
  },
  "brave-search": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-brave-search"],
    "env": {
      "BRAVE_API_KEY": "your-api-key"
    }
  }
}
```

### Available MCP Servers

Popular MCP servers you can use:

1. **Filesystem Server** - File operations
   ```bash
   npx -y @modelcontextprotocol/server-filesystem /allowed/path
   ```

2. **SQLite Server** - Database queries
   ```bash
   npx -y @modelcontextprotocol/server-sqlite /path/to/db.sqlite
   ```

3. **Brave Search** - Web search
   ```bash
   npx -y @modelcontextprotocol/server-brave-search
   ```

4. **GitHub Server** - Repository operations
   ```bash
   npx -y @modelcontextprotocol/server-github
   ```

For a complete list of available MCP servers, visit the [MCP Servers Registry](https://github.com/modelcontextprotocol/servers).

## 📋 Response Formats

### QueryResponse

```go
type QueryResponse struct {
    Answer    string                 // Generated answer
    Sources   []Chunk               // Relevant document chunks  
    Elapsed   string                // Query duration
    ToolCalls []ExecutedToolCall   // Executed MCP tool calls (if tools are used)
    ToolsUsed []string             // List of MCP tool names used
}
```

### MCPChatResponse

```go
type MCPChatResponse struct {
    Content       string              // Initial LLM response
    FinalResponse string              // Final response after tool execution
    ToolCalls     []MCPToolCallResult // MCP tool call results
    Thinking      string              // Reasoning process (if enabled)
    HasThinking   bool                // Whether thinking was enabled
}
```

### MCPToolResult

```go
type MCPToolResult struct {
    Success bool        // Whether the execution was successful
    Data    interface{} // Result data from MCP server
    Error   string      // Error message (if failed)
}
```

## ⚙️ Configuration

Create a `rago.toml` file:

```toml
[providers]
default_llm = "ollama"
default_embedder = "ollama"

[providers.ollama]
base_url = "http://localhost:11434"
llm_model = "qwen3"
embedding_model = "nomic-embed-text"

[mcp]
enabled = true
servers_config_path = "./mcpServers.json"
log_level = "info"
default_timeout = "30s"
max_concurrent_requests = 5

# Built-in tools are deprecated - use MCP servers instead
[tools]
enabled = false  # Disable built-in tools - use MCP instead
```

## 🔒 Security Features

- **MCP Server Isolation**: Each MCP server runs in its own isolated process
- **Configurable Timeouts**: Prevent long-running tool calls from hanging
- **Rate Limiting**: Built-in rate limiting for tool execution
- **Access Control**: Granular control over which MCP servers are enabled
- **Sandboxed Execution**: MCP servers operate in controlled environments

## 📱 Complete Example

Check out [examples/library_usage.go](examples/library_usage.go) for a complete usage example demonstrating MCP integration.

## 🎯 Integration Scenarios

The RAGO library with MCP integration is ideal for:

1. **Intelligent Customer Service Systems**: Answer questions using both knowledge base and external APIs
2. **Development Assistant Tools**: Combine code documentation with real-time system information  
3. **Data Analysis Applications**: Query databases while providing contextual analysis
4. **Process Automation**: Integrate AI reasoning with system operations
5. **Knowledge Management Platforms**: Enhanced search with live data integration

## 📞 Support

For questions or suggestions, please visit [GitHub Issues](https://github.com/liliang-cn/rago/issues).
