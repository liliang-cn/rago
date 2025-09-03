# RAGO MCP Servers Documentation

## Overview

RAGO v2 includes built-in support for official Model Context Protocol (MCP) servers, providing powerful capabilities for file operations, web fetching, memory storage, enhanced reasoning, and time utilities.

## Default MCP Servers

### 1. **Filesystem Server** 🗂️
- **Package**: `@modelcontextprotocol/server-filesystem`
- **Description**: Secure file system operations with sandboxed directory access
- **Key Features**:
  - Read, write, and manage files
  - Directory traversal with security boundaries
  - File watching and monitoring
  - Configurable allowed directories
- **Default Config**:
  ```json
  {
    "command": "npx",
    "args": ["@modelcontextprotocol/server-filesystem", "--allowed-directories", "./", "/tmp"]
  }
  ```

### 2. **Fetch Server** 🌐
- **Package**: `@modelcontextprotocol/server-fetch`
- **Description**: HTTP/HTTPS operations for web content retrieval
- **Key Features**:
  - HTTP/HTTPS requests (GET, POST, PUT, DELETE)
  - Custom headers and authentication
  - Response parsing (JSON, HTML, text)
  - Proxy support
- **Use Cases**:
  - Web scraping
  - API integration
  - Content retrieval
  - Data fetching

### 3. **Memory Server** 💾
- **Package**: `@modelcontextprotocol/server-memory`
- **Description**: In-memory key-value store for temporary data
- **Key Features**:
  - Fast in-memory storage
  - Key-value operations
  - TTL support
  - Namespace isolation
- **Use Cases**:
  - Session storage
  - Caching
  - Temporary data persistence
  - Inter-agent communication

### 4. **Sequential Thinking Server** 🧠
- **Package**: `@modelcontextprotocol/server-sequential-thinking`
- **Description**: Enhanced reasoning through step-by-step problem decomposition
- **Key Features**:
  - Complex problem breakdown
  - Step-by-step reasoning chains
  - Intermediate result tracking
  - Thought process logging
- **Use Cases**:
  - Complex problem solving
  - Multi-step analysis
  - Logical reasoning tasks
  - Decision trees

### 5. **Time Server** ⏰
- **Package**: `@modelcontextprotocol/server-time`
- **Description**: Comprehensive time and date utilities
- **Key Features**:
  - Current time in any timezone
  - Date arithmetic
  - Format conversion
  - Scheduling utilities
- **Use Cases**:
  - Timestamp generation
  - Schedule management
  - Time zone conversion
  - Date calculations

## Installation

### Quick Setup

Run the installation script to set up all MCP servers:

```bash
./scripts/install-mcp-servers.sh
```

### Manual Installation

1. **Prerequisites**:
   - Node.js v18 or higher
   - npm or yarn

2. **Install MCP servers**:
   ```bash
   npm install -g @modelcontextprotocol/server-filesystem
   npm install -g @modelcontextprotocol/server-fetch
   npm install -g @modelcontextprotocol/server-memory
   npm install -g @modelcontextprotocol/server-sequential-thinking
   npm install -g @modelcontextprotocol/server-time
   ```

3. **Verify installation**:
   ```bash
   npx @modelcontextprotocol/server-filesystem --version
   ```

## Configuration

### Server Configuration File

MCP servers are configured in `mcpServers.json`:

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["@modelcontextprotocol/server-filesystem", "--allowed-directories", "./", "/tmp"],
      "description": "File system operations",
      "env": {
        "NODE_OPTIONS": "--max-old-space-size=4096"
      }
    },
    // ... other servers
  },
  "serverGroups": {
    "core": ["filesystem", "fetch", "memory"],
    "reasoning": ["sequential-thinking"],
    "utilities": ["time", "sqlite"]
  }
}
```

### Environment Variables

Each server supports environment variables for configuration:

- **Filesystem**: `ALLOWED_DIRS`, `MAX_FILE_SIZE`
- **Fetch**: `FETCH_USER_AGENT`, `PROXY_URL`
- **Memory**: `MEMORY_MAX_SIZE`, `MEMORY_TTL`
- **Sequential Thinking**: `THINKING_MAX_DEPTH`, `THINKING_TIMEOUT`
- **Time**: `TZ` (timezone)

## Usage

### With RAGO CLI

```bash
# Check MCP server status
rago mcp status

# List available tools
rago mcp list-tools

# Start specific servers
rago mcp start filesystem fetch memory

# Stop all servers
rago mcp stop-all
```

### With Agents

MCP servers integrate seamlessly with the agents module:

```go
// In agent workflow definition
{
  "steps": [
    {
      "id": "read_file",
      "type": "tool",
      "tool": "filesystem.read",
      "inputs": {
        "path": "{{file_path}}"
      }
    },
    {
      "id": "fetch_data",
      "type": "tool", 
      "tool": "fetch.get",
      "inputs": {
        "url": "{{api_endpoint}}"
      }
    }
  ]
}
```

### Programmatic Access

```go
import "github.com/liliang-cn/rago/v2/pkg/mcp"

// Initialize server manager
manager, err := mcp.NewServerManager(config)

// Start servers
err = manager.StartAllServers(ctx)

// Get available tools
tools, err := manager.ListAvailableTools()

// Call a tool
result, err := manager.CallTool(ctx, "filesystem.read", map[string]interface{}{
    "path": "/path/to/file.txt",
})
```

## Server Groups

Servers are organized into logical groups:

- **Core**: Essential servers (filesystem, fetch, memory)
- **Reasoning**: Advanced AI capabilities (sequential-thinking)
- **Utilities**: Helper services (time, sqlite)

Start a group:
```bash
rago mcp start-group core
```

## Security Considerations

1. **Filesystem Server**:
   - Always configure allowed directories
   - Use absolute paths for security
   - Avoid exposing sensitive directories

2. **Fetch Server**:
   - Be cautious with credentials
   - Use environment variables for secrets
   - Configure appropriate timeouts

3. **Memory Server**:
   - Set memory limits
   - Clear sensitive data after use
   - Use namespaces for isolation

## Troubleshooting

### Common Issues

1. **Server fails to start**:
   - Check Node.js version (requires v18+)
   - Verify npm packages are installed
   - Check port conflicts

2. **Tools not available**:
   - Ensure server is running (`rago mcp status`)
   - Check server logs for errors
   - Verify network connectivity

3. **Performance issues**:
   - Adjust NODE_OPTIONS memory limits
   - Configure appropriate timeouts
   - Use server groups to manage resources

### Debug Mode

Enable debug logging:
```bash
export MCP_DEBUG=true
rago mcp start --verbose
```

## Advanced Configuration

### Custom Server Implementation

Create custom MCP servers:

```javascript
// custom-server.js
import { Server } from '@modelcontextprotocol/sdk';

const server = new Server({
  name: 'custom-server',
  version: '1.0.0',
});

server.setRequestHandler('tools/list', async () => {
  return {
    tools: [
      {
        name: 'custom_tool',
        description: 'My custom tool',
        inputSchema: { /* ... */ }
      }
    ]
  };
});

server.start();
```

Add to configuration:
```json
{
  "mcpServers": {
    "custom": {
      "command": "node",
      "args": ["./custom-server.js"]
    }
  }
}
```

## Performance Optimization

1. **Start only needed servers**:
   ```bash
   rago mcp start filesystem fetch  # Start only what you need
   ```

2. **Configure memory limits**:
   ```json
   {
     "env": {
       "NODE_OPTIONS": "--max-old-space-size=2048"
     }
   }
   ```

3. **Use connection pooling** for fetch server
4. **Enable caching** where appropriate
5. **Set appropriate timeouts** for long-running operations

## Integration Examples

### Document Processing Pipeline
```yaml
workflow:
  - filesystem.read → text extraction
  - sequential-thinking → analysis
  - memory.store → cache results
  - fetch.post → send to API
```

### Web Monitoring Agent
```yaml
workflow:
  - time.get → timestamp
  - fetch.get → retrieve webpage
  - memory.compare → check changes
  - filesystem.write → log changes
```

## Contributing

To contribute new MCP servers:
1. Follow the MCP protocol specification
2. Implement required tool handlers
3. Add comprehensive tests
4. Update documentation
5. Submit PR to the MCP servers repository

## Resources

- [MCP Protocol Specification](https://modelcontextprotocol.org)
- [Official MCP Servers Repository](https://github.com/modelcontextprotocol/servers)
- [RAGO Documentation](https://github.com/liliang-cn/rago)
- [Node.js Installation Guide](https://nodejs.org)