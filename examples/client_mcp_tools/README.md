# MCP Tools Integration Example

This example demonstrates how to use MCP (Model Context Protocol) tools with the RAGO client.

## Features Demonstrated

- Enabling MCP functionality
- Listing available MCP tools
- Calling individual tools
- Calling tools with timeout
- Batch tool execution
- Chat with MCP tool assistance
- RAG queries enhanced with MCP tools
- Server status monitoring

## Usage

```bash
# Make sure MCP servers are configured in mcpServers.json
go run main.go
```

## What It Does

1. **Enables MCP**: Activates MCP functionality and starts configured servers
2. **Lists Tools**: Shows all available MCP tools from running servers
3. **Direct Tool Calls**: Demonstrates calling filesystem tools directly
4. **Timeout Control**: Shows how to set timeouts for tool calls
5. **Batch Operations**: Executes multiple tool calls in parallel
6. **MCP-Enhanced Chat**: Uses natural language with automatic tool selection
7. **RAG + MCP**: Combines RAG knowledge with MCP tool capabilities
8. **Status Monitoring**: Checks the status of MCP servers

## Available MCP Tools (Examples)

- `mcp_filesystem_read_file` - Read file contents
- `mcp_filesystem_write_file` - Write to files
- `mcp_filesystem_list_directory` - List directory contents
- `mcp_filesystem_get_file_info` - Get file metadata
- `mcp_websearch_search` - Search the web (if configured)

## Configuration

MCP servers are configured in `mcpServers.json`. Example:

```json
{
  "filesystem": {
    "command": ["npx", "@modelcontextprotocol/server-filesystem"],
    "args": ["--allowed-directories", "."],
    "autoStart": true
  }
}
```

## Prerequisites

- MCP servers must be installed (e.g., `npm install -g @modelcontextprotocol/server-filesystem`)
- `mcpServers.json` must be configured
- Node.js and npx must be available for Node-based MCP servers
- Python and uvx must be available for Python-based MCP servers