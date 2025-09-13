# RAGO Client Examples

This directory contains comprehensive examples demonstrating the capabilities of the RAGO client library after the recent refactoring that removed redundancy while preserving advanced features.

## Examples Overview

### 1. [Basic RAG Operations](./client_basic_rag/)
Demonstrates fundamental RAG functionality:
- Text and file ingestion
- Simple queries
- Queries with source documents
- Document management
- Metadata extraction

```bash
cd client_basic_rag
go run main.go
```

### 2. [MCP Tool Integration](./client_mcp_tools/)
Shows how to leverage MCP (Model Context Protocol) tools:
- Enabling MCP servers
- Direct tool calls
- Batch operations
- MCP-enhanced chat
- RAG + MCP integration

```bash
cd client_mcp_tools
go run main.go
```

### 3. [Interactive Chat with History](./client_chat_history/)
Advanced conversational AI features:
- Conversation history management
- Multi-turn conversations
- RAG-enhanced chat
- Interactive mode with commands
- Streaming with context

```bash
cd client_chat_history
go run main.go
```

### 4. [Advanced Search and Filtering](./client_advanced_search/)
Sophisticated search capabilities:
- Semantic search
- Hybrid search (vector + keyword)
- Score thresholds
- Metadata filtering
- Performance comparisons

```bash
cd client_advanced_search
go run main.go
```

### 5. [Complete LLM Operations](./client_llm_operations/)
Direct LLM operations without RAG:
- Simple generation
- Multi-turn chat
- Streaming responses
- Structured JSON output
- Temperature control
- Code generation

```bash
cd client_llm_operations
go run main.go
```

## Architecture Benefits

These examples showcase the refactored client architecture:

1. **Thin Client Layer**: The client acts as a convenient orchestration layer
2. **Service Delegation**: Basic operations delegate to specialized services:
   - RAG operations → `pkg/rag/client`
   - MCP operations → `pkg/mcp/service`
   - Status checks → `pkg/providers/status`
3. **Advanced Features**: Complex features remain in the client package:
   - Chat history management
   - Interactive modes
   - Task scheduling
   - Search coordination

## Quick Start

1. **Initialize RAGO** (if not already done):
```bash
rago init
```

2. **Ensure providers are running**:
```bash
# For Ollama
ollama serve
ollama pull qwen3:8b
ollama pull nomic-embed-text

# Check status
rago status
```

3. **Run any example**:
```bash
cd examples/client_basic_rag
go run main.go
```

## Common Patterns

### Creating a Client
```go
// Default configuration
client, err := client.New("")

// Custom configuration
client, err := client.New("/path/to/config.toml")

// Always close when done
defer client.Close()
```

### Error Handling
```go
resp, err := client.Query("question")
if err != nil {
    log.Printf("Query failed: %v", err)
    // Handle error appropriately
}
```

### Context Usage
```go
ctx := context.Background()
// Or with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

## Configuration

Each example can be customized through:

1. **Configuration file** (`~/.rago/rago.toml`)
2. **Environment variables** (`RAGO_*`)
3. **Programmatic options** (per-operation options)

## Prerequisites

- Go 1.21 or later
- RAGO initialized (`rago init`)
- LLM provider running (Ollama, OpenAI, etc.)
- For MCP examples: Node.js/Python for MCP servers

## Troubleshooting

### Provider Issues
```bash
# Check provider status
rago status

# Verify models are available
ollama list
```

### MCP Issues
```bash
# Check MCP server status
rago mcp status

# Install MCP servers
npm install -g @modelcontextprotocol/server-filesystem
```

### Database Issues
```bash
# Reset RAG database
rago rag reset

# Check database location
ls ~/.rago/data/
```

## Contributing

Feel free to add more examples! Follow these guidelines:
1. Create a new directory under `examples/`
2. Include a `main.go` with clear comments
3. Add a `README.md` explaining the example
4. Focus on demonstrating specific features
5. Include error handling and cleanup

## License

These examples are part of the RAGO project and follow the same license.