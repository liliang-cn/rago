# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## RAGO Vision: Local-First AI Foundation

**RAGO's True Nature**: A **local-first, privacy-first Go AI foundation** that other applications can use as their AI infrastructure. Built as a library-first architecture with four equal core components that can work independently or together.

**Mission**: Provide a comprehensive AI foundation that prioritizes privacy, local execution, and developer freedom. Other applications should be able to integrate RAGO as their AI backbone without compromising user data or requiring cloud dependencies.

## Four-Pillar Architecture

### 1. **LLM (Language Model Layer)** 🧠
**Purpose**: Unified interface for multiple LLM providers with intelligent pooling, scheduling, and capability management.

**Features**:
- **Provider Pool**: Support for Ollama, LM Studio, OpenAI-compatible providers
- **Capability Detection**: Automatic discovery of model capabilities (chat, generation, embedding, streaming)
- **Priority Scheduling**: Smart routing based on model performance, availability, and cost
- **Load Balancing**: Distribute requests across multiple providers
- **Streaming Support**: Real-time response streaming for all providers
- **Embedding Pipeline**: Unified embedding generation across providers

**Key Packages**: `pkg/providers/`, `pkg/llm/`, `pkg/embedder/`

### 2. **RAG (Retrieval-Augmented Generation)** 📚
**Purpose**: Complete local document storage, chunking, vectorization, and semantic retrieval system.

**Features**:
- **Local Storage**: SQLite-based vector storage (sqvect) with privacy guarantee  
- **Hybrid Search**: Vector similarity + keyword search with RRF (Reciprocal Rank Fusion)
- **Smart Chunking**: Configurable document chunking strategies
- **Document Processing**: Support for PDF, text, markdown, and other formats
- **Metadata Preservation**: Rich metadata storage and filtering capabilities
- **Semantic Search**: Advanced vector similarity with configurable thresholds

**Key Packages**: `pkg/store/`, `pkg/chunker/`, `pkg/processor/`

### 3. **MCP (Model Context Protocol)** 🔧
**Purpose**: Standardized tool integration layer that can operate independently or enhance RAG/Agent capabilities.

**Features**:
- **Tool Ecosystem**: Integration with MCP-compliant tools (filesystem, web, databases)
- **Standalone Operation**: MCP tools can be used without RAG or Agents
- **Health Monitoring**: Automatic tool health checking and failover
- **External Integration**: Connect to any MCP-compatible service
- **Protocol Compliance**: Full MCP specification support

**Key Packages**: `pkg/mcp/`, `pkg/tools/`

### 4. **AGENT (Autonomous Workflows)** 🤖
**Purpose**: Intelligent automation layer that combines LLM reasoning with RAG knowledge and MCP tools.

**Features**:
- **Workflow Automation**: Natural language to structured workflow execution
- **RAG Integration**: Agents can query and reason over local knowledge bases
- **MCP Tool Usage**: Intelligent tool selection and execution
- **Multi-step Reasoning**: Complex task decomposition and execution
- **Scheduling**: Time-based and event-driven agent execution

**Key Packages**: `pkg/agents/`, `pkg/workflow/`, `pkg/scheduler/`

## Layered Architecture Design

```
┌─────────────────────────────────────────────────────────┐
│                    APPLICATION LAYER                     │
│              (Your App Using RAGO as Lib)               │
├─────────────────────────────────────────────────────────┤
│                      AGENT LAYER                        │
│         Workflow • Automation • Multi-step Logic       │
├─────────────────────────────────────────────────────────┤
│          MCP LAYER          │         RAG LAYER         │
│    Tools • Integrations     │   Knowledge • Retrieval   │
├─────────────────────────────┼─────────────────────────────┤
│                      LLM LAYER                          │
│           Provider Pool • Scheduling • Routing         │
├─────────────────────────────────────────────────────────┤
│                    FOUNDATION LAYER                     │
│         Config • Storage • Interfaces • Utils          │
└─────────────────────────────────────────────────────────┘
```

**Layer Independence**:
- **LLM**: Can be used standalone for chat, generation, embeddings
- **RAG**: Can operate without agents, just for knowledge retrieval
- **MCP**: Can work independently for tool integration
- **AGENT**: Leverages all lower layers but optional for simpler use cases

## Library-First Usage

### Primary Interface: `client/client.go`
```go
import "github.com/liliang-cn/rago/v2/client"

// Create RAGO client - handles all four pillars
client, err := client.New(&client.Config{
    ConfigPath: "~/.rago/rago.toml",
})

// Use individual components
llmResponse := client.LLM().Chat(ctx, "Hello world")
ragResults := client.RAG().Query(ctx, "search query") 
mcpResult := client.MCP().CallTool(ctx, "filesystem", "read_file", params)
agentResult := client.Agent().Execute(ctx, "complex workflow task")
```

### Component-Specific Packages
Applications can also use individual components directly:

```go
// Just LLM functionality
import "github.com/liliang-cn/rago/v2/pkg/providers"
provider := providers.NewOllama(config)

// Just RAG functionality  
import "github.com/liliang-cn/rago/v2/pkg/processor"
processor := processor.NewService(config)

// Just MCP functionality
import "github.com/liliang-cn/rago/v2/pkg/mcp"
mcpClient := mcp.NewClient(config)

// Just Agent functionality
import "github.com/liliang-cn/rago/v2/pkg/agents"
agent := agents.NewService(config)
```

## Development Commands

### Build and Test
```bash
# Build main binary and library
make build

# Run all tests across all four pillars
make test

# Run linting and race condition tests  
make check

# Build library-focused binary
go build -o rago ./cmd/rago

# Test individual pillars
go test ./pkg/providers -v  # LLM
go test ./pkg/store -v      # RAG  
go test ./pkg/mcp -v        # MCP
go test ./pkg/agents -v     # AGENT
```

### Running RAGO (CLI/Server Mode)
```bash
# Initialize configuration for all pillars
./rago init

# Test LLM provider connectivity  
./rago status

# RAG operations
./rago ingest document.pdf
./rago query "What is this about?" --show-sources

# MCP server management
./rago mcp status
./rago mcp tools list

# Agent workflow execution
./rago agent run "analyze documents and create summary"

# Start HTTP API server (for web applications)
./rago serve --port 7127
```

## Key Architecture Principles

### 1. **Local-First Privacy**
- All data processing happens locally by default
- No cloud dependencies required for core functionality  
- User data never leaves the local environment unless explicitly configured
- Full control over model selection and data flow

### 2. **Provider Flexibility**
- Support multiple LLM providers simultaneously
- Automatic fallback and load balancing
- Easy provider switching without code changes
- Support for local (Ollama) and cloud (OpenAI) providers

### 3. **Modular Design**
- Each pillar can be used independently
- Clear interfaces between components
- Minimal dependencies between layers
- Easy to integrate partial functionality

### 4. **Library-Optimized**
- Clean, simple Go interfaces
- Extensive examples and documentation
- Minimal configuration required for basic usage
- Production-ready error handling and logging

## Configuration Architecture

**Config Loading Hierarchy**: `~/.rago/rago.toml` → `./rago.toml` → `RAGO_*` env vars → defaults

**Core Configuration Sections**:
```toml
# LLM Provider Configuration
[providers]
default_llm = "ollama"
default_embedder = "ollama"

[[providers.ollama]]
name = "local-llama"
base_url = "http://localhost:11434"
models = ["llama3.2", "qwen2.5:7b"]
priority = 1

[[providers.openai]]  
name = "openai-backup"
api_key = "sk-..."
models = ["gpt-4o-mini"]
priority = 2

# RAG Configuration
[sqvect]
db_path = "~/.rago/vectors.db"
auto_dimension = true

[keyword]
index_path = "~/.rago/keyword_index"

# MCP Configuration  
[mcp]
servers_config = "mcpServers.json"
auto_start = true

# Agent Configuration
[agents]
max_concurrent = 3
timeout = "10m"
```

## Key Files and Interfaces

### Library Interface
- `client/client.go` - Main library interface for external applications
- `pkg/interfaces/interfaces.go` - Core interfaces and contracts

### LLM Pillar
- `pkg/providers/factory.go` - Provider creation and management
- `pkg/providers/ollama/` - Ollama provider implementation
- `pkg/providers/openai/` - OpenAI-compatible provider
- `pkg/llm/` - LLM abstraction layer
- `pkg/embedder/` - Embedding generation

### RAG Pillar  
- `pkg/processor/service.go` - RAG pipeline coordination
- `pkg/store/sqvect.go` - Vector storage implementation
- `pkg/store/keyword.go` - Keyword search implementation
- `pkg/chunker/` - Document chunking strategies

### MCP Pillar
- `pkg/mcp/client.go` - MCP protocol client
- `pkg/mcp/server.go` - MCP server management
- `pkg/tools/` - Tool integration layer

### Agent Pillar
- `pkg/agents/service.go` - Agent orchestration
- `pkg/workflow/` - Workflow execution engine
- `pkg/scheduler/` - Task scheduling system

## Library Integration Examples

### Example Structure for Documentation
```
examples/
├── library_usage/
│   ├── basic_llm/main.go           # LLM-only usage
│   ├── basic_rag/main.go           # RAG-only usage  
│   ├── basic_mcp/main.go           # MCP-only usage
│   ├── basic_agent/main.go         # Agent-only usage
│   └── full_integration/main.go    # All four pillars
├── use_cases/
│   ├── chatbot_app/main.go         # Building a chatbot
│   ├── document_qa/main.go         # Document Q&A system
│   ├── automation_tool/main.go     # Task automation
│   └── knowledge_base/main.go      # Knowledge management
└── advanced/
    ├── custom_provider/main.go     # Adding custom LLM provider
    ├── custom_chunking/main.go     # Custom RAG chunking
    ├── custom_mcp_tool/main.go     # Custom MCP tool
    └── multi_agent/main.go         # Multi-agent workflows
```

## Production Deployment Patterns

### 1. **Embedded Library Mode** (Recommended)
Application includes RAGO as a Go module dependency and uses it internally.

### 2. **Sidecar Service Mode**  
RAGO runs as a separate service with HTTP API, application communicates via REST.

### 3. **CLI Integration Mode**
Application shells out to RAGO CLI commands for specific operations.

## Development Guidelines

### Adding New LLM Providers
1. Implement `pkg/interfaces/LLMProvider` interface
2. Add provider to `pkg/providers/factory.go`  
3. Update configuration schema
4. Add comprehensive tests and examples

### Extending RAG Capabilities
1. Follow `pkg/processor/` patterns for pipeline stages
2. Implement `pkg/interfaces/VectorStore` for new storage backends
3. Add chunking strategies to `pkg/chunker/`

### Creating MCP Tools
1. Follow MCP specification for tool definitions
2. Add tool configurations to `mcpServers.json`
3. Test with `./rago mcp tools test <tool-name>`

### Building Agent Workflows  
1. Use `pkg/workflow/` workflow definition format
2. Implement workflow executors in `pkg/agents/`
3. Add scheduling configurations for automated execution

The codebase prioritizes library usability, clear separation of concerns, comprehensive error handling, and extensive configuration options for different integration patterns. Each pillar can be used independently or as part of the complete AI foundation.