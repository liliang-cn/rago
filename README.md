# RAGO - Local RAG System with Agent Automation

[中文文档](README_zh-CN.md)

RAGO (Retrieval-Augmented Generation Offline) is a fully local RAG system written in Go, integrating SQLite vector database and multi-provider LLM support for document ingestion, semantic search, and context-enhanced Q&A.

## 🌟 Core Features

### 📚 **RAG System (Core)**
- **Document Ingestion** - Import text, markdown, PDF files with automatic chunking
- **Vector Database** - SQLite-based vector storage with sqvect for high-performance search  
- **Semantic Search** - Find relevant documents using embedding similarity
- **Hybrid Search** - Combine vector similarity with keyword matching for better results
- **Smart Chunking** - Configurable text splitting (sentence, paragraph, token methods)
- **Q&A Generation** - Context-enhanced answers using retrieved documents
- **Metadata Extraction** - Automatic keywords and summary generation for documents

### 🔧 **Multi-Provider LLM Support**
- **Ollama Integration** - Local LLM inference with ollama-go client
- **OpenAI Compatible** - Support for OpenAI API and compatible services
- **LM Studio** - Local model serving with LM Studio integration
- **Provider Switching** - Easy configuration switching between providers
- **Streaming Support** - Real-time token streaming for better UX
- **Structured Generation** - Generate JSON outputs matching specific schemas

### 🛠️ **MCP Tools Integration**
- **Model Context Protocol** - Standard tool integration framework
- **Built-in Tools** - filesystem, fetch, memory, time, sequential-thinking
- **External Servers** - Connect to any MCP-compatible tool server
- **Query Enhancement** - Use tools during RAG queries for richer answers
- **Batch Operations** - Execute multiple tool calls in parallel

### 🤖 **Agent Automation**
- **Natural Language Workflows** - Generate workflows from plain text descriptions
- **MCP Tool Orchestration** - Coordinate multiple tools in automated workflows
- **Async Execution** - Parallel step execution with dependency resolution
- **Intent Recognition** - Automatically detect user intent for smarter responses

### 💻 **Developer Experience**
- **Simplified Client API** - Clean, intuitive client package for all operations
- **Comprehensive Examples** - Ready-to-run examples for common use cases
- **Interactive Mode** - Built-in REPL for testing and exploration
- **Chat History Management** - Full conversation tracking and context retention
- **Advanced Search Options** - Fine-tune search with scores, filters, and metadata

### 🏢 **Production Ready**
- **100% Local** - Complete offline operation with local providers
- **HTTP API** - RESTful endpoints for all operations
- **High Performance** - Optimized Go implementation
- **Configurable** - Extensive configuration options via TOML
- **Zero-Config Mode** - Works out of the box with smart defaults

## 🚀 Quick Start (Zero Config!)

**✨ NEW: RAGO works without ANY configuration!**

### 30-Second Setup

```bash
# 1. Install RAGO
go install github.com/liliang-cn/rago/v2@latest

# 2. Install Ollama (if not already installed)
curl -fsSL https://ollama.com/install.sh | sh

# 3. Start using RAGO immediately!
rago status  # Works without any config file!
```

That's it! No configuration needed. RAGO uses smart defaults.

### Installation Options

```bash
# Clone and build
git clone https://github.com/liliang-cn/rago.git
cd rago
go build -o rago ./cmd/rago

# Optional: Create config (only if you need custom settings)
./rago init  # Interactive - choose "Skip" for zero-config
```

### 🎯 Zero-Config Usage

```bash
# Pull default models
ollama pull qwen3              # Default LLM
ollama pull nomic-embed-text   # Default embedder

# Everything works without config!
./rago status                  # Check provider status
./rago ingest document.pdf     # Import documents
./rago query "What is this about?"  # Query knowledge base

# Interactive mode
./rago query -i

# With MCP tools
./rago query "Analyze this data and save results" --mcp
```

### 🤖 Agent Examples 

```bash
# Natural language workflows
./rago agent run "get current time and tell me if it's morning or evening"
./rago agent run "fetch weather for San Francisco and analyze conditions"

# Save workflows for reuse
./rago agent run "monitor github.com/golang/go for new releases" --save
```

## 📖 Library Usage

### Simplified Client API (Recommended)

The new client package provides a streamlined interface for all RAGO features:

```go
import "github.com/liliang-cn/rago/v2/client"

// Create client with default config
client, err := client.New("")
defer client.Close()

// Basic RAG operations
err = client.IngestText("Your content here", "doc-id")
err = client.IngestFile("document.pdf")

response, err := client.Query("What is this about?")
fmt.Println(response.Answer)

// Query with sources
resp, err := client.QueryWithSources("Tell me more", true)
for _, source := range resp.Sources {
    fmt.Printf("Source: %s (score: %.2f)\n", source.ID, source.Score)
}

// MCP tools integration
client.EnableMCP(ctx)
result, err := client.CallMCPTool(ctx, "filesystem_read", map[string]interface{}{
    "path": "README.md",
})

// Chat with history
chatResp, err := client.ChatWithHistory(ctx, "Hello", conversation)

// LLM operations
llmResp, err := client.LLMGenerate(ctx, client.LLMGenerateRequest{
    Prompt:      "Write a haiku",
    Temperature: 0.9,
})
```

### Advanced Usage Examples

Comprehensive examples demonstrating all client features:

- **[Basic RAG Operations](./examples/client_basic_rag)** - Document ingestion, queries, metadata extraction
- **[MCP Tool Integration](./examples/client_mcp_tools)** - Tool calls, batch operations, MCP-enhanced chat
- **[Interactive Chat](./examples/client_chat_history)** - Conversation history, streaming, interactive mode
- **[Advanced Search](./examples/client_advanced_search)** - Semantic/hybrid search, filtering, performance tuning
- **[LLM Operations](./examples/client_llm_operations)** - Generation, chat, streaming, structured output

### Direct Package Usage (Advanced)

For fine-grained control, use the underlying packages directly:

```go
import (
    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/rag/processor"
    "github.com/liliang-cn/rago/v2/pkg/store"
)

// Initialize components
cfg, _ := config.Load("rago.toml")
store, _ := store.NewSQLiteStore(cfg.Sqvect.DBPath)
processor := processor.New(cfg, store)

// Direct RAG operations
doc := domain.Document{
    ID:      "doc1",
    Content: "Your document content",
}
err := processor.IngestDocument(ctx, doc)

// Query with custom parameters
req := domain.QueryRequest{
    Query:       "What is this about?",
    TopK:        5,
    Temperature: 0.7,
}
response, _ := processor.Query(ctx, req)
```

## 🛠️ MCP Tools

### Built-in Tools

- **filesystem** - File operations (read, write, list, execute)
- **fetch** - HTTP/HTTPS requests
- **memory** - Temporary key-value storage  
- **time** - Date/time operations
- **sequential-thinking** - LLM analysis and reasoning
- **playwright** - Browser automation 

### Tool Configuration

Configure MCP servers in `mcpServers.json`:

```json
{
  "filesystem": {
    "command": "npx",
    "args": ["@modelcontextprotocol/server-filesystem"],
    "description": "File system operations"
  },
  "fetch": {
    "command": "npx",
    "args": ["@modelcontextprotocol/server-fetch"],
    "description": "HTTP/HTTPS operations"
  }
}
```

## 📊 HTTP API

Start the API server:

```bash
./rago serve --port 7127
```

### Core Endpoints

#### RAG Operations
- `POST /api/ingest` - Ingest documents into vector database
- `POST /api/query` - Perform RAG query with context retrieval
- `GET /api/list` - List indexed documents
- `DELETE /api/reset` - Clear vector database

#### MCP Tools
- `GET /api/mcp/tools` - List available MCP tools
- `POST /api/mcp/tools/call` - Execute MCP tool
- `GET /api/mcp/status` - Check MCP server status

#### Agent Automation 
- `POST /api/agent/run` - Generate and execute workflows
- `GET /api/agent/list` - List saved agents
- `POST /api/agent/create` - Create new agent

## ⚙️ Configuration

Configure providers in `rago.toml`:

```toml
[providers]
default_llm = "lmstudio"  # or "ollama", "openai"
default_embedder = "lmstudio"

[providers.lmstudio]
type = "lmstudio"
base_url = "http://localhost:1234"
llm_model = "qwen/qwen3-4b-2507"
embedding_model = "text-embedding-qwen3-embedding-4b"
timeout = "120s"

[providers.ollama]
type = "ollama"
base_url = "http://localhost:11434"
llm_model = "qwen3"
embedding_model = "nomic-embed-text"
timeout = "120s"

# Vector database configuration
[sqvect]
db_path = "~/.rago/rag.db"
top_k = 5
threshold = 0.0

# Text chunking configuration
[chunker]
chunk_size = 500
overlap = 50
method = "sentence"

# MCP tools configuration
[mcp]
enabled = true
servers_config_path = "mcpServers.json"
```

## 📚 Documentation

### Examples
- [Client Usage Examples](./examples/) - Comprehensive client library examples
  - [Basic RAG](./examples/client_basic_rag) - Getting started with RAG operations
  - [MCP Tools](./examples/client_mcp_tools) - Tool integration patterns
  - [Chat & History](./examples/client_chat_history) - Interactive conversations
  - [Advanced Search](./examples/client_advanced_search) - Search optimization
  - [LLM Operations](./examples/client_llm_operations) - Direct LLM usage
- [Agent Examples](./examples/agent_usage/) - Agent automation patterns

### References
- [API Reference](./docs/api.md) - HTTP API documentation
- [Configuration Guide](./rago.example.toml) - Full configuration options
- [中文文档](./README_zh-CN.md) - Chinese documentation

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

MIT License - see [LICENSE](LICENSE) for details

## 🔗 Links

- [GitHub Repository](https://github.com/liliang-cn/rago)
- [Issue Tracker](https://github.com/liliang-cn/rago/issues)
- [Discussions](https://github.com/liliang-cn/rago/discussions)