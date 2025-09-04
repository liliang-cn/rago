# RAGO - Advanced RAG System with Agent Automation

[中文文档](README_zh-CN.md)

RAGO (Retrieval-Augmented Generation Offline) is a powerful local RAG system with agent automation capabilities, supporting natural language workflow generation, MCP tools integration, and multi-provider LLM support.

## 🌟 Core Features

### 🤖 **Agent Automation**
- **Natural Language → Workflow** - Convert plain text requests into executable workflows
- **Async Execution** - Parallel step execution with dependency resolution
- **MCP Tools Integration** - Built-in tools for filesystem, web, memory, time, and LLM reasoning

### 📚 **Advanced RAG System**
- **Multi-Provider Support** - Seamlessly switch between Ollama, OpenAI, and LM Studio
- **Vector Search** - High-performance semantic search with SQLite vector database
- **Smart Chunking** - Intelligent document processing with configurable strategies

### ⚡ **Workflow Automation**
- **JSON Workflow Spec** - Define complex workflows programmatically
- **Variable Passing** - Data flow between steps using `{{variable}}` syntax
- **Tool Orchestration** - Coordinate multiple MCP tools in workflows

### 🔧 **Enterprise Ready**
- **HTTP APIs** - Complete REST API for all operations
- **100% Local Option** - Full privacy with local LLM providers
- **High Performance** - Optimized Go implementation

## 🚀 Quick Start

### Prerequisites

1. **Install Go** (≥ 1.21)
2. **Choose Your LLM Provider**:
   - **Ollama** (Local): `curl -fsSL https://ollama.com/install.sh | sh`
   - **LM Studio** (Local): Download from [lmstudio.ai](https://lmstudio.ai)
   - **OpenAI** (Cloud): Get API key from [platform.openai.com](https://platform.openai.com)

### Installation

```bash
# Clone and build
git clone https://github.com/liliang-cn/rago.git
cd rago
go build -o rago ./cmd/rago

# Initialize configuration
./rago init
```

### 🎯 Agent Examples

```bash
# Natural language to workflow
./rago agent run "get current time and tell me if it's morning or evening"

# GitHub integration
./rago agent run "get information about golang/go repository"

# Complex workflows
./rago agent run "fetch weather for San Francisco and analyze if it's good for outdoor activities"

# Save workflows
./rago agent run "monitor github.com/golang/go for new releases" --save
```

## 📖 Library Usage

Use RAGO as a Go library in your applications:

```go
import (
    "github.com/liliang-cn/rago/v2/pkg/agents/execution"
    "github.com/liliang-cn/rago/v2/pkg/agents/types"
    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/utils"
)

// Load config and initialize
cfg, _ := config.Load("")
ctx := context.Background()
_, llmService, _, _ := utils.InitializeProviders(ctx, cfg)

// Define workflow
workflow := &types.WorkflowSpec{
    Steps: []types.WorkflowStep{
        {
            ID:   "fetch",
            Tool: "fetch",
            Inputs: map[string]interface{}{
                "url": "https://api.github.com/repos/golang/go",
            },
            Outputs: map[string]string{"data": "result"},
        },
        {
            ID:   "analyze",
            Tool: "sequential-thinking",
            Inputs: map[string]interface{}{
                "prompt": "Analyze this data",
                "data":   "{{result}}",
            },
        },
    },
}

// Execute
executor := execution.NewWorkflowExecutor(cfg, llmService)
result, _ := executor.Execute(ctx, workflow)
```

## 🛠️ MCP Tools

### Built-in Tools

- **filesystem** - File operations (read, write, list, execute)
- **fetch** - HTTP/HTTPS requests
- **memory** - Temporary key-value storage  
- **time** - Date/time operations
- **sequential-thinking** - LLM analysis and reasoning
- **playwright** - Browser automation (optional)

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

- `POST /api/ingest` - Ingest documents
- `POST /api/query` - Query with RAG
- `GET /api/mcp/tools` - List MCP tools
- `POST /api/mcp/tools/call` - Execute MCP tool
- `POST /api/agent/run` - Run natural language workflow

## ⚙️ Configuration

Configure providers in `rago.toml`:

```toml
[providers]
default_llm = "lmstudio"  # or "ollama", "openai"
default_embedder = "lmstudio"

[providers.lmstudio]
type = "lmstudio"
base_url = "http://localhost:1234/v1"
llm_model = "qwen/qwen3-4b"
embedding_model = "nomic-embed-text"

[providers.ollama]
type = "ollama"
base_url = "http://localhost:11434"
llm_model = "llama3"
embedding_model = "nomic-embed-text"
```

## 📚 Documentation

- [Examples](./examples/) - Code examples and use cases
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