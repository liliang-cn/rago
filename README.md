# RAGO - Local RAG System

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
go build -o rago ./cmd/rago-cli

# Optional: Create config (only if you need custom settings)
# Copy the example config and modify as needed:
cp rago.example.toml rago.toml
# Edit rago.toml with your preferred settings
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


## 📖 Library Usage

### Simplified Client API (Recommended)

The new client package provides a streamlined interface for all RAGO features:

```go
import "github.com/liliang-cn/rago/v2/client"

// Create client - now only two entry points!
client, err := client.New("path/to/config.toml")  // Or empty string for defaults
// Or with programmatic config
cfg := &config.Config{...}
client, err := client.NewWithConfig(cfg)
defer client.Close()

// LLM operations with wrapper
if client.LLM != nil {
    response, err := client.LLM.Generate("Write a haiku")
    
    // With options
    resp, err := client.LLM.GenerateWithOptions(ctx, "Explain quantum computing", 
        &client.GenerateOptions{Temperature: 0.7, MaxTokens: 200})
    
    // Streaming
    err = client.LLM.Stream(ctx, "Tell a story", func(chunk string) {
        fmt.Print(chunk)
    })
}

// RAG operations with wrapper  
if client.RAG != nil {
    err = client.RAG.Ingest("Your document content")
    answer, err := client.RAG.Query("What is this about?")
    
    // With sources
    resp, err := client.RAG.QueryWithOptions(ctx, "Tell me more",
        &client.QueryOptions{TopK: 5, ShowSources: true})
}

// MCP tools with wrapper
if client.Tools != nil {
    tools, err := client.Tools.List()
    result, err := client.Tools.Call(ctx, "filesystem_read", 
        map[string]interface{}{"path": "README.md"})
}


// Direct BaseClient methods also available
resp, err := client.Query(ctx, client.QueryRequest{Query: "test"})
resp, err := client.RunTask(ctx, client.TaskRequest{Task: "analyze data"})
```

### Advanced Usage Examples

Comprehensive examples demonstrating all client features:

- **[Basic Client Initialization](./examples/01_basic_client)** - Different ways to initialize the client
- **[LLM Operations](./examples/02_llm_operations)** - Generation, streaming, chat with history
- **[RAG Operations](./examples/03_rag_operations)** - Document ingestion, queries, semantic search
- **[MCP Tools Integration](./examples/04_mcp_tools)** - Tool listing, execution, LLM integration
- **[Complete Platform Demo](./examples/06_complete_platform)** - All features working together

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
  - [Basic Client](./examples/01_basic_client) - Client initialization methods
  - [LLM Operations](./examples/02_llm_operations) - Direct LLM usage
  - [RAG Operations](./examples/03_rag_operations) - Document ingestion and queries
  - [MCP Tools](./examples/04_mcp_tools) - Tool integration patterns
    - [Complete Platform](./examples/06_complete_platform) - Full integration example

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