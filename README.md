# RAGO - Local RAG System

[中文文档](README_zh-CN.md)

RAGO (Retrieval-Augmented Generation Offline) is a fully local RAG system written in Go, integrating SQLite vector database (sqvect) and local LLM client (ollama-go), supporting document ingestion, semantic search, and context-enhanced Q&A.

## 🎯 Features

- **Fully Offline** - No external APIs required, protecting data privacy
- **Multi-format Support** - Supports TXT, Markdown, PDF and other formats
- **Local Vector Database** - SQLite-based sqvect vector storage
- **Local LLM** - Call local models through Ollama
- **Dual Interface Design** - Both CLI tool and HTTP API usage modes
- **High Performance** - Go implementation with low memory usage and fast response
- **Extensible** - Modular design, easy to extend new features

## 🚀 Quick Start

### Prerequisites

1. **Install Go** (≥ 1.21)
2. **Install Ollama**
   ```bash
   curl -fsSL https://ollama.com/install.sh | sh
   ```
3. **Download Models**
   ```bash
   ollama pull nomic-embed-text  # Embedding model
   ollama pull gemma3           # Generation model
   ```

### Install RAGO

```bash
git clone https://github.com/liliang-cn/rago.git
cd rago
make setup
make build
```

### Basic Usage

1. **Ingest Documents**

   ```bash
   ./build/rago ingest ./docs/sample.md
   ./build/rago ingest ./docs/ --recursive  # Process directory recursively
   ```

2. **Query Knowledge Base**

   ```bash
   ./build/rago query "What is RAG?"
   ./build/rago query --interactive         # Interactive mode
   ```

3. **Start API Service**

   ```bash
   ./build/rago serve --port 8080
   ```

4. **List Imported Documents**
   ```bash
   ./build/rago list
   ```

## 📖 Detailed Usage

### CLI Commands

#### Document Management

```bash
# Single file ingestion
rago ingest ./document.txt

# Batch ingestion
rago ingest ./docs/ --recursive --chunk-size 500 --overlap 100

# Import from URL (planned)
rago ingest --url "https://example.com/doc.html"
```

#### Query Functions

```bash
# Direct query
rago query "Hello world"

# Interactive mode
rago query --interactive

# Streaming output
rago query "Explain machine learning" --stream

# Batch query
rago query --file questions.txt

# Adjust parameters
rago query "What is deep learning" --top-k 10 --temperature 0.3 --max-tokens 1000
```

#### Data Management

```bash
# List all documents
rago list

# Reset database
rago reset --force

# Export data (planned)
rago export ./backup.json

# Import data (planned)
rago import ./backup.json
```

### HTTP API

Start the server:

```bash
rago serve --port 8080 --host 0.0.0.0
```

#### API Endpoints

**Health Check**

```bash
GET /api/health
```

**Document Ingestion**

```bash
POST /api/ingest
Content-Type: application/json

{
  "content": "This is the text content to ingest",
  "chunk_size": 300,
  "overlap": 50,
  "metadata": {
    "source": "manual_input"
  }
}
```

**Query**

```bash
POST /api/query
Content-Type: application/json

{
  "query": "What is artificial intelligence?",
  "top_k": 5,
  "temperature": 0.7,
  "max_tokens": 500,
  "stream": false
}
```

**Document Management**

```bash
# Get document list
GET /api/documents

# Delete document
DELETE /api/documents/{document_id}
```

**Search (Retrieval Only)**

```bash
POST /api/search
Content-Type: application/json

{
  "query": "artificial intelligence",
  "top_k": 5
}
```

## ⚙️ Configuration

### Configuration File

Create `config.toml`:

```toml
[server]
port = 8080
host = "localhost"
cors_origins = ["*"]

[ollama]
embedding_model = "nomic-embed-text"
llm_model = "gemma3"
base_url = "http://localhost:11434"
timeout = "30s"

[sqvect]
db_path = "./data/rag.db"
top_k = 5

[chunker]
chunk_size = 300
overlap = 50
method = "sentence"  # sentence, paragraph, token

[ui]
title = "RAGO - Local RAG System"
theme = "light"
max_file_size = "10MB"
```

### Environment Variables

```bash
export RAGO_SERVER_PORT=8080
export RAGO_OLLAMA_BASE_URL=http://localhost:11434
export RAGO_SQVECT_DB_PATH=./data/custom.sqlite
```

## 🐳 Docker Deployment

### Build Image

```bash
make docker-build
```

### Run Container

```bash
docker run -d \
  --name rago \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config.toml:/app/config.toml \
  rago:latest
```

### Docker Compose

```yaml
version: "3.8"
services:
  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama_data:/root/.ollama

  rago:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./config.toml:/app/config.toml
    depends_on:
      - ollama
    environment:
      - RAGO_OLLAMA_BASE_URL=http://ollama:11434

volumes:
  ollama_data:
```

## 🧪 Development

### Build and Test

```bash
# Install dependencies
make deps

# Format code
make fmt

# Run tests
make test

# Code check
make check

# Development mode
make dev
```

### Project Structure

```
rago/
├── cmd/rago/           # CLI commands
├── internal/           # Internal modules
│   ├── config/        # Configuration management
│   ├── domain/        # Domain models
│   ├── chunker/       # Text chunking
│   ├── embedder/      # Embedding service
│   ├── llm/           # Generation service
│   ├── store/         # Storage layer
│   └── processor/     # Core processor
├── api/handlers/       # API handlers
├── test/              # Integration tests
├── docs/              # Documentation
└── Makefile           # Build scripts
```

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the project
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Ollama](https://ollama.com/) - Local LLM runtime
- [SQLite](https://sqlite.org/) - Embedded database
- [Gin](https://gin-gonic.com/) - HTTP web framework
- [Cobra](https://cobra.dev/) - CLI application framework

## 📞 Contact

For questions or suggestions, please contact through:

- GitHub Issues: [https://github.com/liliang-cn/rago/issues](https://github.com/liliang-cn/rago/issues)
- Email: your.email@example.com

---

⭐ If this project helps you, please give it a Star!
