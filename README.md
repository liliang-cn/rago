# RAGO - Simplified Local RAG System

[中文文档](README_zh-CN.md)

RAGO (Retrieval-Augmented Generation Offline) v2 is a streamlined, production-ready RAG system written in Go. It provides clean APIs for document ingestion, semantic search, and context-enhanced Q&A with a focus on simplicity and reliability.

## 🌟 Core Features (v2 Simplified)

### 📚 **RAG System (Core)**
- **Document Ingestion** - Import text, markdown, PDF files with automatic chunking
- **Vector Database** - SQLite-based vector storage with sqvect for high-performance search
- **Semantic Search** - Find relevant documents using embedding similarity
- **Smart Chunking** - Configurable text splitting (sentence, paragraph, token methods)
- **Q&A Generation** - Context-enhanced answers using retrieved documents
- **Metadata Extraction** - Automatic keywords and summary generation for documents

### 🔧 **OpenAI-Compatible LLM Support**
- **Unified Provider Interface** - Single OpenAI-compatible API for all LLM services
- **Local First** - Works with Ollama, LM Studio, and any OpenAI-compatible server
- **Streaming Support** - Real-time token streaming for better UX
- **Structured Generation** - Generate JSON outputs matching specific schemas
- **Health Monitoring** - Built-in provider health checks

### 🛠️ **MCP Tools Integration**
- **Model Context Protocol** - Standard tool integration framework
- **Built-in Tools** - filesystem, fetch, memory, time, sequential-thinking
- **External Servers** - Connect to any MCP-compatible tool server
- **Query Enhancement** - Use tools during RAG queries for richer answers

### 👥 **Profile Management**
- **Multi-User Support** - Create and manage user profiles with different settings
- **Profile Switching** - Seamlessly switch between different configurations
- **Settings Persistence** - Store and retrieve user preferences and LLM settings
- **Isolated Environments** - Each profile maintains its own configuration and data

### 💻 **Developer Experience**
- **Clean Library API** - Simple, intuitive interfaces for all operations
- **Zero-Config Mode** - Works out of the box with smart defaults
- **HTTP API** - RESTful endpoints for all operations
- **High Performance** - Optimized Go implementation with minimal dependencies

## 🚀 Quick Start (Zero Config!)

**✨ RAGO v2 works without ANY configuration!**

### Installation

```bash
# Option 1: Install directly
go install github.com/liliang-cn/rago/v2@latest

# Option 2: Clone and build
git clone https://github.com/liliang-cn/rago.git
cd rago
go build -o rago ./cmd/rago-cli

# Option 3: Use Makefile
make build
```

### 🎯 Zero-Config Usage

RAGO v2 works with OpenAI-compatible providers out of the box:

```bash
# Check system status (works without config!)
./rago status

# Import documents into RAG knowledge base
./rago rag ingest document.pdf
./rago rag ingest "path/to/text/file.txt"
./rago rag ingest --text "Direct text content" --source "my-document"

# Query your knowledge base
./rago rag query "What is this document about?"

# List all indexed documents
./rago rag list

# Interactive mode (if available)
./rago rag query -i

# With MCP tools enabled (if available)
./rago rag query "Analyze this data and save results" --mcp
```

### Environment Variables (Optional)

```bash
# For OpenAI-compatible services
# API key is optional - provider handles authentication
export RAGO_OPENAI_API_KEY="your-api-key"  # Optional
export RAGO_OPENAI_BASE_URL="http://localhost:11434/v1"  # Ollama
export RAGO_OPENAI_LLM_MODEL="qwen3"
export RAGO_OPENAI_EMBEDDING_MODEL="nomic-embed-text"
```


## 📖 Library Usage

### RAG Client API (Recommended)

The simplified RAG client provides clean interfaces for all operations:

```go
import (
    "context"
    "fmt"
    "github.com/liliang-cn/rago/v2/pkg/rag"
    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/providers"
)

// Initialize with default configuration
cfg, _ := config.Load("")  // Empty string for defaults
cfg.Providers.DefaultLLM = "openai"
cfg.Providers.OpenAI.BaseURL = "http://localhost:11434/v1"  // Ollama
cfg.Providers.OpenAI.LLMModel = "qwen3"
cfg.Providers.OpenAI.EmbeddingModel = "nomic-embed-text"

// Create providers
embedder, _ := providers.CreateEmbedderProvider(context.Background(), cfg.Providers.OpenAI)
llm, _ := providers.CreateLLMProvider(context.Background(), cfg.Providers.OpenAI)

// Create RAG client
client, _ := rag.NewClient(cfg, embedder, llm, nil)
defer client.Close()

// Ingest documents
ctx := context.Background()
resp, err := client.IngestFile(ctx, "document.pdf", rag.DefaultIngestOptions())
fmt.Printf("Ingested %d chunks\n", resp.ChunkCount)

// Query your knowledge base
queryResp, err := client.Query(ctx, "What is this document about?", rag.DefaultQueryOptions())
fmt.Printf("Answer: %s\n", queryResp.Answer)
fmt.Printf("Sources: %d\n", len(queryResp.Sources))

// Ingest text directly
textResp, err := client.IngestText(ctx, "Your text content", "source.txt", rag.DefaultIngestOptions())
fmt.Printf("Text ingested with ID: %s\n", textResp.DocumentID)
```

### LLM Service API

For direct LLM operations:

```go
import (
    "context"
    "github.com/liliang-cn/rago/v2/pkg/llm"
    "github.com/liliang-cn/rago/v2/pkg/domain"
)

// Create LLM service
llmService := llm.NewService(llmProvider)

// Simple generation
response, err := llmService.Generate(ctx, "Write a haiku", &domain.GenerationOptions{
    Temperature: 0.7,
    MaxTokens:   100,
})

// Streaming generation
err = llmService.Stream(ctx, "Tell me a story", &domain.GenerationOptions{
    Temperature: 0.8,
    MaxTokens:   500,
}, func(chunk string) {
    fmt.Print(chunk)
})

// Tool calling
messages := []domain.Message{
    {Role: "user", Content: "What's the weather like today?"},
}
tools := []domain.ToolDefinition{
    // Define your tools here
}
result, err := llmService.GenerateWithTools(ctx, messages, tools, &domain.GenerationOptions{})
```

### Configuration-Based Usage

Create a `rago.toml` configuration file:

```toml
[providers]
default_llm = "openai"
default_embedder = "openai"

[providers.openai]
type = "openai"
base_url = "http://localhost:11434/v1"  # Ollama endpoint
api_key = "ollama"  # Required even for local
llm_model = "qwen3"
embedding_model = "nomic-embed-text"
timeout = "30s"

[sqvect]
db_path = "~/.rago/data/rag.db"
top_k = 5
threshold = 0.0

[chunker]
chunk_size = 500
overlap = 50
method = "sentence"

[mcp]
enabled = true
```

Then use it in your code:

```go
cfg, _ := config.Load("rago.toml")
// ... rest of the initialization code
```

## 🛠️ MCP Tools

### Built-in Tools

- **filesystem** - File operations (read, write, list, execute)
- **fetch** - HTTP/HTTPS requests
- **memory** - Temporary key-value storage
- **time** - Date/time operations
- **sequential-thinking** - LLM analysis and reasoning

### Tool Configuration

Configure MCP servers in `mcpServers.json`:

```json
{
  "filesystem": {
    "command": "npx",
    "args": ["@modelcontextprotocol/server-filesystem", "/path/to/allowed/directory"],
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
- `POST /api/rag/ingest` - Ingest documents into vector database
- `POST /api/rag/query` - Perform RAG query with context retrieval
- `GET /api/rag/list` - List indexed documents
- `DELETE /api/rag/reset` - Clear vector database
- `GET /api/rag/collections` - List all collections

#### MCP Tools
- `GET /api/mcp/tools` - List available MCP tools
- `POST /api/mcp/tools/call` - Execute MCP tool
- `GET /api/mcp/status` - Check MCP server status

## ⚙️ Configuration

### Environment Variables (Simple)

```bash
# Basic OpenAI-compatible configuration
export RAGO_OPENAI_API_KEY="your-api-key"
export RAGO_OPENAI_BASE_URL="http://localhost:11434/v1"
export RAGO_OPENAI_LLM_MODEL="qwen3"
export RAGO_OPENAI_EMBEDDING_MODEL="nomic-embed-text"

# Server settings
export RAGO_SERVER_PORT="7127"
export RAGO_SERVER_HOST="0.0.0.0"
```

### Configuration File (Advanced)

Create `rago.toml` for full control:

```toml
[server]
port = 7127
host = "0.0.0.0"
enable_ui = false

[providers]
default_llm = "openai"
default_embedder = "openai"

[providers.openai]
type = "openai"
base_url = "http://localhost:11434/v1"  # Ollama endpoint
api_key = "ollama"
llm_model = "qwen3"
embedding_model = "nomic-embed-text"
timeout = "30s"

[sqvect]
db_path = "~/.rago/data/rag.db"
top_k = 5
threshold = 0.0

[chunker]
chunk_size = 500
overlap = 50
method = "sentence"

[mcp]
enabled = true
servers_config_path = "mcpServers.json"
```

## 📚 Documentation

### API References
- **[RAG Client API](./pkg/rag/)** - Core RAG client documentation
- **[LLM Service API](./pkg/llm/)** - LLM service documentation
- **[Configuration Guide](./pkg/config/)** - Full configuration options
- **[中文文档](./README_zh-CN.md)** - Chinese documentation

### Examples (Coming Soon)
We're updating examples for the simplified v2 API. Check back soon for:
- Basic RAG client usage
- LLM service examples
- MCP tools integration
- Configuration patterns

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

MIT License - see [LICENSE](LICENSE) for details

## 🔗 Links

- [GitHub Repository](https://github.com/liliang-cn/rago)
- [Issue Tracker](https://github.com/liliang-cn/rago/issues)
- [Discussions](https://github.com/liliang-cn/rago/discussions)