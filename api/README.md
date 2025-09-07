# RAGO API Layer

The RAGO API layer provides a comprehensive HTTP/WebSocket interface for all four pillars of the RAGO architecture: LLM, RAG, MCP, and Agents.

## Architecture

The API layer follows a modular design with clear separation of concerns:

```
api/
├── server.go           # Main HTTP server
├── router.go           # Route definitions
├── middleware/         # Common middleware
│   ├── auth.go        # Authentication (Bearer, API Key, Basic)
│   ├── cors.go        # CORS handling
│   ├── ratelimit.go   # Rate limiting
│   └── logging.go     # Request/Response logging
├── handlers/           # HTTP handlers for each pillar
│   ├── llm/           # LLM operations
│   ├── rag/           # RAG operations
│   ├── mcp/           # MCP tool operations
│   ├── agents/        # Agent workflows
│   └── unified/       # Multi-pillar operations
├── websocket/         # WebSocket support
│   ├── hub.go         # WebSocket connection management
│   ├── stream.go      # Streaming operations
│   └── events.go      # Event broadcasting
└── docs/
    └── openapi.yaml   # OpenAPI 3.0 specification

## Features

### Security
- **Multiple Auth Types**: Bearer tokens (JWT), API keys, Basic auth
- **Rate Limiting**: Per-IP and per-user rate limiting with configurable limits
- **CORS Support**: Configurable CORS with wildcard subdomain support
- **TLS Support**: HTTPS with configurable certificates

### Performance
- **Streaming**: Server-Sent Events (SSE) and WebSocket support for real-time streaming
- **Batch Operations**: Batch endpoints for bulk processing
- **Connection Pooling**: Efficient resource management
- **Metrics**: Prometheus-compatible metrics endpoint

### Developer Experience
- **OpenAPI Documentation**: Complete OpenAPI 3.0 specification
- **Swagger UI**: Interactive API documentation at `/swagger`
- **Request IDs**: Automatic request ID generation and tracking
- **Structured Logging**: JSON and console logging with request context

## Quick Start

### Running the Server

```go
package main

import (
    "log"
    "github.com/liliang-cn/rago/v2/api"
)

func main() {
    config := &api.Config{
        Host: "0.0.0.0",
        Port: 7127,
        EnableAuth: true,
        AuthType: "bearer",
        AuthSecret: "your-secret-key",
        EnableRateLimit: true,
        RateLimit: 100, // requests per minute
        EnableSwagger: true,
        ClientConfig: "~/.rago/rago.toml",
    }

    server, err := api.NewServer(config)
    if err != nil {
        log.Fatal(err)
    }

    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### Configuration

```toml
# api.toml
[server]
host = "0.0.0.0"
port = 7127
read_timeout = "30s"
write_timeout = "30s"

[auth]
enabled = true
type = "bearer" # bearer, api_key, basic
secret = "${RAGO_API_SECRET}"

[rate_limit]
enabled = true
requests_per_minute = 100
burst_size = 20

[cors]
allowed_origins = ["http://localhost:3000", "https://*.example.com"]
allow_credentials = true

[features]
swagger = true
metrics = true
websocket = true
```

## API Endpoints

### Health & Monitoring

- `GET /health` - Health check for all components
- `GET /ready` - Readiness probe
- `GET /metrics` - Prometheus metrics

### LLM Operations

- `POST /api/v1/llm/generate` - Generate text
- `POST /api/v1/llm/stream` - Stream text generation (SSE)
- `GET /api/v1/llm/providers` - List providers
- `POST /api/v1/llm/providers` - Add provider
- `DELETE /api/v1/llm/providers/:name` - Remove provider
- `GET /api/v1/llm/providers/health` - Provider health status
- `POST /api/v1/llm/batch` - Batch generation
- `POST /api/v1/llm/tools/generate` - Generate with tools
- `POST /api/v1/llm/tools/stream` - Stream with tools

### RAG Operations

- `POST /api/v1/rag/ingest` - Ingest document
- `POST /api/v1/rag/ingest/batch` - Batch ingestion
- `GET /api/v1/rag/documents` - List documents
- `DELETE /api/v1/rag/documents/:id` - Delete document
- `POST /api/v1/rag/search` - Vector search
- `POST /api/v1/rag/search/hybrid` - Hybrid search
- `GET /api/v1/rag/stats` - RAG statistics
- `POST /api/v1/rag/optimize` - Optimize indices
- `POST /api/v1/rag/reset` - Reset RAG system

### MCP Operations

- `GET /api/v1/mcp/servers` - List MCP servers
- `POST /api/v1/mcp/servers` - Register server
- `DELETE /api/v1/mcp/servers/:name` - Unregister server
- `GET /api/v1/mcp/servers/:name/health` - Server health
- `GET /api/v1/mcp/tools` - List tools
- `GET /api/v1/mcp/tools/:name` - Get tool info
- `POST /api/v1/mcp/tools/:name/call` - Call tool
- `POST /api/v1/mcp/tools/:name/call-async` - Async tool call
- `POST /api/v1/mcp/tools/batch` - Batch tool execution

### Agent Operations

- `GET /api/v1/agents/workflows` - List workflows
- `POST /api/v1/agents/workflows` - Create workflow
- `DELETE /api/v1/agents/workflows/:name` - Delete workflow
- `POST /api/v1/agents/workflows/:name/execute` - Execute workflow
- `POST /api/v1/agents/workflows/:name/schedule` - Schedule workflow
- `GET /api/v1/agents/agents` - List agents
- `POST /api/v1/agents/agents` - Create agent
- `DELETE /api/v1/agents/agents/:name` - Delete agent
- `POST /api/v1/agents/agents/:name/execute` - Execute agent
- `GET /api/v1/agents/scheduled` - Get scheduled tasks

### Unified Operations

- `POST /api/v1/chat` - Chat with RAG and tools
- `POST /api/v1/chat/stream` - Stream chat (SSE)
- `POST /api/v1/process` - Process document
- `POST /api/v1/task` - Execute complex task
- `POST /api/v1/query` - Intelligent query
- `POST /api/v1/analyze` - Content analysis

### WebSocket Endpoints

- `WS /ws/stream` - Real-time streaming for all operations
- `WS /ws/events` - System event notifications

## Authentication

### Bearer Token (JWT)

```bash
curl -H "Authorization: Bearer ${JWT_TOKEN}" \
     -X POST http://localhost:7127/api/v1/llm/generate \
     -d '{"prompt": "Hello, world!"}'
```

### API Key

```bash
curl -H "X-API-Key: ${API_KEY}" \
     -X POST http://localhost:7127/api/v1/llm/generate \
     -d '{"prompt": "Hello, world!"}'
```

### Basic Auth

```bash
curl -u username:password \
     -X POST http://localhost:7127/api/v1/llm/generate \
     -d '{"prompt": "Hello, world!"}'
```

## WebSocket Usage

### Streaming Connection

```javascript
const ws = new WebSocket('ws://localhost:7127/ws/stream');

ws.onopen = () => {
    // Send generation request
    ws.send(JSON.stringify({
        id: '123',
        type: 'generate',
        data: {
            prompt: 'Tell me about AI',
            max_tokens: 500
        }
    }));
};

ws.onmessage = (event) => {
    const response = JSON.parse(event.data);
    if (response.type === 'chunk') {
        console.log('Received chunk:', response.data);
    } else if (response.type === 'complete') {
        console.log('Generation complete');
    }
};
```

### Event Subscription

```javascript
const ws = new WebSocket('ws://localhost:7127/ws/events');

ws.onopen = () => {
    // Subscribe to events
    ws.send(JSON.stringify({
        action: 'subscribe',
        events: ['health', 'tool_executed', 'workflow_start']
    }));
};

ws.onmessage = (event) => {
    const evt = JSON.parse(event.data);
    console.log(`Event ${evt.type}:`, evt.data);
};
```

## Testing

### Unit Tests

```bash
go test ./api/... -v -cover
```

### Integration Tests

```bash
go test ./api/... -tags=integration -v
```

### Load Testing

```bash
# Using Apache Bench
ab -n 1000 -c 10 -H "X-API-Key: test-key" \
   http://localhost:7127/api/v1/llm/generate

# Using hey
hey -n 1000 -c 10 -H "X-API-Key: test-key" \
    -m POST -d '{"prompt":"test"}' \
    http://localhost:7127/api/v1/llm/generate
```

## Monitoring

### Prometheus Metrics

The API exposes Prometheus metrics at `/metrics`:

- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - Request latency
- `http_request_size_bytes` - Request size
- `http_response_size_bytes` - Response size
- `active_websocket_connections` - Active WebSocket connections
- `llm_generation_duration_seconds` - LLM generation time
- `rag_search_duration_seconds` - RAG search time
- `mcp_tool_execution_duration_seconds` - Tool execution time
- `agent_workflow_duration_seconds` - Workflow execution time

### Health Checks

```bash
# Simple health check
curl http://localhost:7127/health

# Detailed component health
curl http://localhost:7127/health | jq .
```

## Error Handling

The API uses standard HTTP status codes and structured error responses:

```json
{
    "error": "Invalid request",
    "message": "The 'prompt' field is required",
    "request_id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2024-01-01T12:00:00Z"
}
```

Status codes:
- `200` - Success
- `201` - Created
- `202` - Accepted (async operations)
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `429` - Too Many Requests
- `500` - Internal Server Error
- `503` - Service Unavailable

## Production Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o rago-api ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/rago-api .
EXPOSE 7127
CMD ["./rago-api"]
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rago-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: rago-api
  template:
    metadata:
      labels:
        app: rago-api
    spec:
      containers:
      - name: rago-api
        image: rago:latest
        ports:
        - containerPort: 7127
        env:
        - name: RAGO_API_AUTH_SECRET
          valueFrom:
            secretKeyRef:
              name: rago-secret
              key: auth-secret
        livenessProbe:
          httpGet:
            path: /health
            port: 7127
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 7127
          initialDelaySeconds: 5
          periodSeconds: 5
```

### TLS/HTTPS

```go
config := &api.Config{
    // ... other config
    TLSEnabled: true,
    TLSCertFile: "/path/to/cert.pem",
    TLSKeyFile: "/path/to/key.pem",
}
```

## Performance Tuning

### Connection Pooling

```go
config := &api.Config{
    MaxConnections: 1000,
    MaxIdleConnections: 100,
    ConnectionTimeout: 30 * time.Second,
}
```

### Rate Limiting

```go
config := &api.Config{
    RateLimit: 1000,  // requests per minute
    RateBurst: 100,   // burst size
    // Adaptive rate limiting based on load
    AdaptiveRateLimit: true,
}
```

### Caching

```go
config := &api.Config{
    EnableCache: true,
    CacheTTL: 5 * time.Minute,
    CacheSize: 100 * 1024 * 1024, // 100MB
}
```

## License

MIT License - see LICENSE file for details.