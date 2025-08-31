# RAGO Client Examples

This directory contains various examples demonstrating how to use RAGO as a Go library.

## Prerequisites

Make sure you have:
1. Go 1.21 or later installed
2. RAGO configured with a valid `rago.toml` file
3. Required AI services running (Ollama, OpenAI, etc.)

## Running Examples

Each example is in its own directory and can be run independently:

```bash
# Navigate to any example directory
cd basic_usage

# Run the example
go run main.go
```

## Available Examples

### 1. [Basic Usage](./basic_usage/)
**File**: `basic_usage/main.go`

Demonstrates the most basic RAGO client usage:
- Creating a client
- Performing simple queries
- Handling responses

```bash
cd basic_usage && go run main.go
```

### 2. [Streaming Chat](./streaming_chat/)
**File**: `streaming_chat/main.go`  

Shows how to use RAGO's streaming capabilities:
- Streaming query responses
- Direct LLM chat streaming
- Real-time response processing

```bash
cd streaming_chat && go run main.go
```

### 3. [Document Ingestion](./document_ingestion/)
**File**: `document_ingestion/main.go`

Demonstrates document ingestion and management:
- Ingesting text content directly
- Ingesting files from filesystem
- Listing ingested documents
- Querying ingested content

```bash
cd document_ingestion && go run main.go
```

### 4. [MCP Integration](./mcp_integration/)
**File**: `mcp_integration/main.go`

Shows Model Context Protocol (MCP) integration:
- Tool calling capabilities
- External service integration
- MCP server communication

```bash
cd mcp_integration && go run main.go
```

### 5. [Task Scheduling](./task_scheduling/)
**File**: `task_scheduling/main.go`

Demonstrates workflow and task management:
- Task-oriented queries  
- Workflow processing
- Data organization

```bash
cd task_scheduling && go run main.go
```

## Code Structure

Each example follows this pattern:

```go
package main

import (
    "github.com/liliang-cn/rago/client"
)

func main() {
    // Create client
    c, err := client.New("")
    if err != nil {
        log.Fatal("Failed to create RAGO client:", err)
    }
    defer c.Close()

    // Use client features
    // ...
}
```

## Configuration

All examples use the default configuration loading:

```go
c, err := client.New("") // Uses default config path
```

This will look for configuration in:
- `./rago.toml` (current directory)
- `~/.rago/rago.toml` (home directory)

You can also specify a custom config path:

```go
c, err := client.New("/path/to/your/rago.toml")
```

## Error Handling

All examples include proper error handling patterns. In production code, you should handle errors appropriately for your use case.

## Integration

These examples can be easily integrated into your own applications. Simply copy the relevant patterns and adapt them to your needs.

## Learn More

- [Main Documentation](../../README.md)
- [Client Library Documentation](../../client/README.md)
- [Configuration Guide](../../rago.example.toml)