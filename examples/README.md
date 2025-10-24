# RAGO v2 Examples

This directory contains comprehensive examples demonstrating how to use RAGO v2 as a library. The examples showcase the simplified, streamlined API introduced in v2.

## 📁 Example Structure

### [basic_rag_usage/](./basic_rag_usage)
**Purpose**: Demonstrates core RAG functionality with the simplified client API.

**Features shown**:
- RAG client initialization
- Document ingestion (file and text)
- Querying the knowledge base
- Document management and statistics

**Run**:
```bash
cd examples/basic_rag_usage
go run main.go
# Or with a specific file
go run main.go path/to/your/document.pdf
```

### [llm_service/](./llm_service)
**Purpose**: Shows direct LLM operations using the LLM service API.

**Features shown**:
- Simple text generation
- Streaming generation
- Tool calling (if supported)
- Structured JSON generation
- Intent recognition
- Health checks

**Run**:
```bash
cd examples/llm_service
go run main.go
```

### [config_based/](./config_based)
**Purpose**: Demonstrates configuration-based initialization and management.

**Features shown**:
- Loading configuration from TOML files
- Provider initialization from config
- RAG client creation with configuration
- Configuration validation
- Example configuration file generation

**Run**:
```bash
cd examples/config_based
go run main.go rago.toml
```

## 🔧 Environment Variables

All examples support these environment variables for configuration:

```bash
# LLM Provider Configuration
export RAGO_OPENAI_BASE_URL="http://localhost:11434/v1"
export RAGO_OPENAI_API_KEY="your-api-key"
export RAGO_OPENAI_LLM_MODEL="qwen3"
export RAGO_OPENAI_EMBEDDING_MODEL="nomic-embed-text"
```

## 📋 Prerequisites

1. **Go 1.24+**: Required to build and run examples
2. **LLM Provider**: An OpenAI-compatible service such as:
   - [Ollama](https://ollama.com/) (recommended for local use)
   - [LM Studio](https://lmstudio.ai/)
   - OpenAI API
   - Any OpenAI-compatible endpoint

3. **Models**: For Ollama, pull the default models:
```bash
ollama pull qwen3
ollama pull nomic-embed-text
```

## 🚀 Quick Start

1. **Clone the repository**:
```bash
git clone https://github.com/liliang-cn/rago.git
cd rago
```

2. **Set up environment variables** (optional):
```bash
export RAGO_OPENAI_BASE_URL="http://localhost:11434/v1"
export RAGO_OPENAI_API_KEY="ollama"
```

3. **Run any example**:
```bash
cd examples/basic_rag_usage
go run main.go
```

## 🎯 Example Focus Areas

### RAG Operations
- Document ingestion (files and text)
- Semantic search and retrieval
- Context-enhanced question answering
- Source attribution and scoring

### LLM Integration
- Direct text generation
- Streaming responses
- Tool calling capabilities
- Structured output generation

### Configuration Management
- TOML-based configuration
- Environment variable overrides
- Provider switching and management
- Validation and error handling

## 📚 Related Documentation

- [RAGO Client API](../pkg/rag/) - Core RAG client documentation
- [LLM Service API](../pkg/llm/) - LLM service documentation
- [Configuration Guide](../pkg/config/) - Full configuration options
- [Main README](../README.md) - Complete project documentation

## 🤝 Contributing

Found an issue or want to improve the examples? Please:

1. Check existing [issues](https://github.com/liliang-cn/rago/issues)
2. Create a new issue with detailed description
3. Submit a pull request with your improvements

## 📄 License

These examples are part of the RAGO project and licensed under the MIT License - see the main [LICENSE](../LICENSE) file for details.