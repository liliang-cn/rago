# Swagger MCP Server Integration in RAGO

RAGO now has built-in support for [mcp-swagger-server v0.4.0](https://github.com/liliang-cn/mcp-swagger-server), allowing you to automatically generate MCP tools from any Swagger/OpenAPI specification.

## Features

- **Automatic Tool Generation**: Convert Swagger operations into MCP tools
- **Multiple Transport Options**: Support for both stdio and HTTP transports
- **Flexible Input Sources**: Load from URLs, files, or raw data
- **API Filtering**: Include/exclude specific operations
- **Authentication Support**: Built-in support for API keys and tokens
- **HTTP Server Mode**: Run as an HTTP server with health, tools, and MCP endpoints

## Usage

### 1. Using the CLI Command

RAGO provides a `swagger` command to manage Swagger-based MCP servers:

```bash
# Start a server from a Swagger URL
./rago swagger start --url https://petstore.swagger.io/v2/swagger.json

# Start from a local file
./rago swagger start --file ./api-spec.yaml

# Start with HTTP transport
./rago swagger start --url https://petstore.swagger.io/v2/swagger.json \
  --transport http --port 8080

# List available tools
./rago swagger list --url https://petstore.swagger.io/v2/swagger.json

# Call a specific tool
./rago swagger call getPetById --url https://petstore.swagger.io/v2/swagger.json \
  --args '{"petId": 1}'

# With authentication
./rago swagger start --url https://api.example.com/swagger.json \
  --auth-type bearer --auth-value YOUR_TOKEN
```

### 2. Programmatic Integration

You can integrate Swagger MCP servers directly in your Go code:

```go
import mcpswagger "github.com/liliang-cn/mcp-swagger-server/mcp"

// Create server from URL
server, err := mcpswagger.NewFromSwaggerURL(
    "https://petstore.swagger.io/v2/swagger.json",
    "", // apiBaseURL (will be inferred)
    "", // apiKey
)

// Create server from file
server, err := mcpswagger.NewFromSwaggerFile(
    "/path/to/swagger.json",
    "https://api.example.com",
    "YOUR_API_KEY",
)

// Create server from raw data
data := []byte(`{"swagger": "2.0", ...}`)
server, err := mcpswagger.NewFromSwaggerData(
    data,
    "https://api.example.com",
    "YOUR_API_KEY",
)
```

### 3. Using Filters

Filter operations to expose only what you need:

```go
filter := &mcpswagger.APIFilter{
    // Exclude specific HTTP methods
    ExcludeMethods: []string{"DELETE", "PATCH"},
    
    // Exclude specific paths
    ExcludePaths: []string{"/admin/*"},
    
    // Exclude by operation ID
    ExcludeOperationIDs: []string{"deleteUser"},
    
    // Exclude by tags
    ExcludeTags: []string{"internal", "deprecated"},
    
    // Or include only specific operations
    IncludeOnlyOperationIDs: []string{"getUser", "listUsers"},
}

config := mcpswagger.DefaultConfig()
config.Filter = filter
server, err := mcpswagger.New(config)
```

### 4. HTTP Transport

Run the MCP server as an HTTP server:

```go
// Configure HTTP transport
httpTransport := &mcpswagger.HTTPTransport{
    Port: 8080,
    Host: "localhost",
    Path: "/mcp",
}

config := mcpswagger.DefaultConfig()
config.Transport = httpTransport

server, err := mcpswagger.New(config)

// Start HTTP server
ctx := context.Background()
err = server.RunHTTP(ctx, 8080)
```

Available HTTP endpoints:
- `/health` - Health check endpoint
- `/tools` - List all available tools
- `/mcp` - MCP protocol endpoint

## Configuration Options

### SwaggerConfig

Used for the RAGO wrapper:

```go
type SwaggerConfig struct {
    // Name of the server
    Name string
    
    // Swagger source (one of these required)
    SwaggerURL  string  // URL to fetch Swagger spec
    SwaggerFile string  // Local file path
    SwaggerData []byte  // Raw Swagger data
    
    // Transport configuration
    Transport string // "stdio" or "http"
    HTTPConfig *HTTPTransportConfig
    
    // API configuration
    BaseURL string            // Override base URL
    Headers map[string]string // Custom headers
    Auth    *SwaggerAuthConfig // Authentication
    
    // Operation timeout
    Timeout time.Duration
}
```

### Authentication

```go
type SwaggerAuthConfig struct {
    Type   string // "bearer", "basic", or "apikey"
    Value  string // Token or key value
    Header string // Header name for API key
}
```

## Examples

### Complete Example: Petstore API

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    mcpswagger "github.com/liliang-cn/mcp-swagger-server/mcp"
)

func main() {
    // Create server from Petstore API
    server, err := mcpswagger.NewFromSwaggerURL(
        "https://petstore.swagger.io/v2/swagger.json",
        "", // Will use URL from swagger spec
        "", // No API key needed
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Run with stdio transport
    ctx := context.Background()
    if err := server.RunStdio(ctx); err != nil {
        log.Fatal(err)
    }
}
```

### Example with Authentication

```go
// Create config with API key
config := mcpswagger.DefaultConfig()
config.APIKey = "sk-abc123..."
config.APIBaseURL = "https://api.openai.com/v1"

// Load Swagger spec
data, _ := mcpswagger.FetchSwaggerFromURL("https://api.openai.com/swagger.json")
config.SwaggerData = data

// Create and run server
server, _ := mcpswagger.New(config)
server.Run(context.Background())
```

### Running Multiple Swagger Servers

```go
manager := mcp.NewSwaggerManager()

// Add Petstore API
manager.AddServer("petstore", &mcp.SwaggerConfig{
    SwaggerURL: "https://petstore.swagger.io/v2/swagger.json",
})

// Add GitHub API
manager.AddServer("github", &mcp.SwaggerConfig{
    SwaggerURL: "https://api.github.com/swagger.json",
    Auth: &mcp.SwaggerAuthConfig{
        Type:  "bearer",
        Value: "ghp_xxxx",
    },
})

// Start all servers
ctx := context.Background()
manager.StartAll(ctx)
```

## Benefits

1. **Automatic Tool Discovery**: No manual tool definition needed
2. **Type Safety**: Input/output schemas derived from Swagger
3. **Authentication Handling**: Built-in support for common auth patterns
4. **Flexible Filtering**: Control exactly which operations to expose
5. **Multiple Transports**: Use stdio for CLI or HTTP for web integrations
6. **Error Handling**: Proper error propagation and logging

## Common Use Cases

- **API Integration**: Quickly integrate any REST API with Swagger docs
- **Microservices**: Expose internal services as MCP tools
- **Testing**: Create mock servers from Swagger specs
- **Development**: Rapid prototyping with existing APIs
- **Gateway**: Unified interface to multiple APIs

## Troubleshooting

### Server doesn't start
- Check that the Swagger spec is valid (use swagger.io validator)
- Ensure the URL is accessible or file exists
- Verify authentication credentials if required

### Tools not appearing
- Check filter configuration isn't excluding operations
- Verify operations have operationId defined
- Ensure paths are properly formatted in the spec

### HTTP transport issues
- Check port is not already in use
- Verify firewall allows the port
- Test with curl: `curl http://localhost:PORT/health`

## References

- [mcp-swagger-server GitHub](https://github.com/liliang-cn/mcp-swagger-server)
- [MCP Protocol Specification](https://github.com/modelcontextprotocol/spec)
- [Swagger/OpenAPI Specification](https://swagger.io/specification/)
- [RAGO Documentation](../README.md)