# LLM Tool Calling Implementation in RAGO

## Overview

RAGO now supports comprehensive LLM tool calling, allowing language models to use MCP (Model Context Protocol) tools during generation. This enables LLMs to:
- Access file systems
- Fetch web content
- Query databases
- Execute system commands
- And more through MCP-compliant tools

## Architecture

### 1. Provider Level
Each LLM provider (Ollama, OpenAI, LMStudio) implements tool calling through:
- `GenerateWithTools()` - Single generation with tool support
- `StreamWithTools()` - Streaming generation with tool support

#### Ollama Implementation
Since Ollama doesn't natively support function calling, we implement it through prompt engineering:
```go
// Tool definitions are included in the system prompt
// LLM responds with JSON when it wants to call a tool
{
  "tool_calls": [
    {
      "id": "unique_call_id",
      "name": "tool_name",
      "parameters": {"param1": "value1"}
    }
  ]
}
```

#### OpenAI Implementation
OpenAI has native tool calling support, so we use their API directly:
```go
params.Tools = tools // Native OpenAI tools format
params.ToolChoice = "auto" // Let model decide when to use tools
```

### 2. Service Level
The LLM service (`pkg/llm/service.go`) orchestrates tool calling:
- Integrates with MCP service to fetch available tools
- Routes tool generation requests to appropriate providers
- Handles tool execution and result continuation

Key methods:
- `GenerateWithTools()` - Basic tool calling
- `GenerateWithToolExecution()` - Automatic tool execution and continuation
- `SetMCPService()` - Connects MCP tools to LLM service

### 3. Client Level
The unified client wires everything together:
```go
// During initialization, connect MCP to LLM
if llmService != nil && mcpService != nil {
    mcpAdapter := &mcpServiceAdapter{service: mcpService}
    llmService.SetMCPService(mcpAdapter)
}
```

## Usage Examples

### Basic Tool Calling
```go
// Get available tools from MCP
tools := client.MCP().ListTools()

// Create request with tools
req := core.ToolGenerationRequest{
    GenerationRequest: core.GenerationRequest{
        Prompt: "What is the current time?",
    },
    Tools: tools,
    ToolChoice: "auto", // Let LLM decide
}

// Generate with tools
response, err := client.LLM().GenerateWithTools(ctx, req)

// Check if tools were called
for _, call := range response.ToolCalls {
    fmt.Printf("Tool called: %s\n", call.Name)
}
```

### Automatic Tool Execution
```go
// Use GenerateWithToolExecution for automatic execution
response, err := llmService.GenerateWithToolExecution(ctx, req)
// Tools are automatically executed and results fed back to LLM
```

### Streaming with Tools
```go
err := client.LLM().StreamWithTools(ctx, req, func(chunk core.ToolStreamChunk) error {
    // Handle streaming chunks
    fmt.Print(chunk.Delta)
    
    // Tool calls appear in the final chunk
    if chunk.Finished && len(chunk.ToolCalls) > 0 {
        // Process tool calls
    }
    return nil
})
```

## Tool Definition Format

Tools are defined in `mcpServers.json`:
```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["@modelcontextprotocol/server-filesystem"],
      "description": "File system operations"
    }
  }
}
```

These are automatically converted to LLM-compatible format with:
- Name prefixed with "mcp_" for identification
- Description for the LLM to understand purpose
- Parameter schema for proper invocation

## Implementation Details

### Tool Call Parsing (Ollama)
The Ollama provider uses regex to extract JSON tool calls from responses:
```go
func (p *OllamaProvider) parseToolCalls(content string) ([]ToolCall, string) {
    // Look for JSON blocks with tool_calls
    // Extract and parse tool call data
    // Return cleaned content without JSON
}
```

### Tool Result Handling
Tool results are added as messages with role "tool":
```go
toolResults = append(toolResults, core.Message{
    Role:       "tool",
    Content:    string(resultJSON),
    ToolCallID: toolCall.ID,
})
```

### Cross-Pillar Integration
The implementation showcases RAGO's four-pillar architecture:
- **LLM Pillar**: Handles generation and tool call parsing
- **MCP Pillar**: Provides and executes tools
- **RAG Pillar**: Can be used alongside tools for context
- **Agent Pillar**: Can orchestrate complex tool-using workflows

## Testing

Run the test program:
```bash
go run examples/test-llm-tool-integration/main.go
```

This tests:
1. Tool discovery
2. Simple tool calling
3. Automatic tool execution
4. Multiple tool calls
5. Streaming with tools

## Configuration

Enable tool calling in your configuration:
```toml
[providers.list]
[[providers.list]]
name = "ollama-llama"
type = "ollama"
model = "llama3.2"
enabled = true

[mcp]
servers_path = "mcpServers.json"
```

## Best Practices

1. **Tool Selection**: Provide only relevant tools to reduce token usage
2. **Tool Naming**: Use clear, descriptive names for tools
3. **Error Handling**: Always handle tool execution failures gracefully
4. **Token Limits**: Be aware of context limits when using many tools
5. **Streaming**: Tool calls typically appear at the end of streaming

## Future Enhancements

- [ ] Native Ollama function calling when available
- [ ] Tool result caching for repeated calls
- [ ] Parallel tool execution
- [ ] Tool dependency resolution
- [ ] Custom tool validation
- [ ] Tool usage analytics