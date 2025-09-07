# RAGO V3 - Local-First AI Foundation

A privacy-first, local AI foundation built in Go with a four-pillar architecture that provides comprehensive AI capabilities without compromising user data.

## 🏗️ Four-Pillar Architecture

RAGO V3 is built on four independent yet integrated pillars:

### 1. **LLM Pillar** 🧠
Unified interface for multiple language model providers with intelligent pooling and load balancing.

### 2. **RAG Pillar** 📚
Complete local document storage, chunking, vectorization, and semantic retrieval system.

### 3. **MCP Pillar** 🔧
Model Context Protocol implementation for standardized tool integration.

### 4. **Agent Pillar** 🤖
Intelligent automation layer combining LLM reasoning with RAG knowledge and MCP tools.

## 🚀 Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/liliang-cn/rago.git
cd rago

# Build the project
go build -o rago ./cmd/rago

# Or run directly
go run ./cmd/rago
```

### Configuration

RAGO uses a TOML configuration file. Create `rago.toml` in your project directory:

```toml
# Global settings
data_dir = "~/.rago"
log_level = "info"

# LLM Configuration
[llm]
default_provider = "ollama"

[[llm.providers.list]]
name = "ollama"
type = "ollama"
base_url = "http://localhost:11434"
model = "llama3.2"
enabled = true
weight = 10
timeout = "30s"

[[llm.providers.list]]
name = "lmstudio"
type = "openai"
base_url = "http://localhost:1234/v1"
api_key = "not-needed"
model = "local-model"
enabled = true
weight = 5
timeout = "30s"

[llm.load_balancing]
strategy = "weighted_round_robin"
max_retries = 3
```

### Usage

#### As a Library

```go
import "github.com/liliang-cn/rago/v2/pkg/client"

// Load configuration and create client
ragoClient, err := client.New("rago.toml")
if err != nil {
    log.Fatal(err)
}
defer ragoClient.Close()

// Use LLM pillar
response, err := ragoClient.LLM().Generate(ctx, core.GenerationRequest{
    Prompt: "Hello, world!",
    MaxTokens: 100,
})

// Use RAG pillar
results, err := ragoClient.RAG().Search(ctx, core.SearchRequest{
    Query: "How does this work?",
    TopK: 5,
})
```

#### CLI Usage

```bash
# Simple generation
go run examples/rago_cli.go "What is the meaning of life?"

# With options
go run examples/rago_cli.go -verbose -stream "Tell me a story"

# List providers
go run examples/rago_cli.go -list

# Interactive mode
go run examples/rago_cli.go
```

## 📝 Configuration Guide

### Provider Configuration

RAGO supports multiple provider types:

- **Ollama**: Local models with full privacy
- **LM Studio**: OpenAI-compatible local models
- **OpenAI**: Cloud-based GPT models (when needed)

Example multi-provider setup:

```toml
[llm]
default_provider = "primary"

# Primary local model
[[llm.providers.list]]
name = "primary"
type = "ollama"
base_url = "http://localhost:11434"
model = "llama3.2"
weight = 10  # Gets 2/3 of requests

# Secondary model
[[llm.providers.list]]
name = "secondary"
type = "openai"
base_url = "http://localhost:1234/v1"
model = "gpt-4o-mini"
weight = 5   # Gets 1/3 of requests

# Embedding model
[[llm.providers.list]]
name = "embeddings"
type = "ollama"
base_url = "http://localhost:11434"
model = "nomic-embed-text"
[llm.providers.list.parameters]
is_embedding_model = true
```

### Load Balancing Strategies

```toml
[llm.load_balancing]
strategy = "weighted_round_robin"  # Options: round_robin, weighted_round_robin, least_connections
max_retries = 3
retry_delay = "1s"
circuit_breaker_threshold = 5
circuit_breaker_timeout = "30s"
```

### RAG Configuration

```toml
[rag]
storage_backend = "hybrid"

[rag.embedding]
provider = "embeddings"
model = "nomic-embed-text"

[rag.vector_store]
backend = "sqvect"
dimensions = 768
metric = "cosine"

[rag.chunking_strategy]
chunk_size = 1000
chunk_overlap = 200
strategy = "recursive"
```

## 🔧 Advanced Features

### Health Monitoring

```go
// Check system health
health := ragoClient.Health()
fmt.Printf("System: %s\n", health.Overall)

// Check individual providers
for name, status := range health.Providers {
    fmt.Printf("%s: %s\n", name, status)
}
```

### Streaming Responses

```go
err := ragoClient.LLM().Stream(ctx, request, func(chunk core.StreamChunk) error {
    fmt.Print(chunk.Delta)
    return nil
})
```

### Batch Processing

```go
requests := []core.GenerationRequest{
    {Prompt: "Question 1"},
    {Prompt: "Question 2"},
    {Prompt: "Question 3"},
}
responses, err := ragoClient.LLM().GenerateBatch(ctx, requests)
```

## 🛠️ Development

### Building from Source

```bash
# Build all components
make build

# Run tests
make test

# Run with race detection
make check
```

### Running Examples

```bash
# Test configuration
go run examples/test_with_config.go rago.toml

# Dual provider demo
go run examples/dual_provider_demo.go

# Interactive CLI
go run examples/rago_cli.go
```

## 📚 Examples

### Simple Generation

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/liliang-cn/rago/v2/pkg/client"
    "github.com/liliang-cn/rago/v2/pkg/core"
)

func main() {
    // Create client from config
    ragoClient, err := client.New("rago.toml")
    if err != nil {
        log.Fatal(err)
    }
    defer ragoClient.Close()
    
    // Generate response
    response, err := ragoClient.LLM().Generate(
        context.Background(),
        core.GenerationRequest{
            Prompt: "Explain quantum computing in simple terms",
            MaxTokens: 200,
            Temperature: 0.7,
        },
    )
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response.Content)
    fmt.Printf("Provider: %s, Model: %s\n", response.Provider, response.Model)
}
```

### Document Ingestion and RAG

```go
// Ingest a document
err := ragoClient.RAG().IngestDocument(ctx, core.IngestRequest{
    DocumentID: "doc1",
    Content: "Your document content here",
    Metadata: map[string]interface{}{
        "source": "manual",
        "type": "documentation",
    },
})

// Search documents
results, err := ragoClient.RAG().Search(ctx, core.SearchRequest{
    Query: "How does authentication work?",
    TopK: 5,
    Threshold: 0.7,
})

// Use results for RAG-enhanced generation
context := buildContextFromResults(results)
response, err := ragoClient.LLM().Generate(ctx, core.GenerationRequest{
    Prompt: fmt.Sprintf("Based on: %s\n\nAnswer: %s", context, query),
})
```

## 🔒 Privacy & Security

RAGO prioritizes privacy:

- **Local-First**: All processing happens locally by default
- **No Cloud Dependencies**: Core functionality works offline
- **Data Control**: Your data never leaves your environment
- **Configurable**: Choose when and if to use cloud services

## 📋 System Requirements

- Go 1.21 or higher
- For Ollama provider: Ollama installed and running
- For LM Studio: LM Studio server running
- Minimum 8GB RAM recommended
- SSD storage for vector database

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🆘 Support

- **Documentation**: [docs.rago.ai](https://docs.rago.ai)
- **Issues**: [GitHub Issues](https://github.com/liliang-cn/rago/issues)
- **Discussions**: [GitHub Discussions](https://github.com/liliang-cn/rago/discussions)

## 🗺️ Roadmap

- [ ] Web UI Dashboard
- [ ] Additional vector store backends
- [ ] More MCP tool integrations
- [ ] Agent marketplace
- [ ] Distributed processing support
- [ ] Mobile SDK

---

Built with ❤️ for the local AI community