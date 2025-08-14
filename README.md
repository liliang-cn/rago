# RAGO - Local RAG System

[中文文档](README_zh-CN.md)

RAGO (Retrieval-Augmented Generation Offline) is a fully local RAG system written in Go, integrating SQLite vector database (sqvect) and local LLM client (ollama-go), supporting document ingestion, semantic search, and context-enhanced Q&A.

## 🎯 Features

- **Fully Offline** - No external APIs required, protecting data privacy
- **Multi-format Support** - Supports TXT and Markdown formats
- **Local Vector Database** - SQLite-based sqvect vector storage
- **Local LLM** - Call local models through Ollama
- **Web UI Interface** - Built-in web interface for easy interaction
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
   ollama pull qwen3           # Generation model
   ```

### Install RAGO

#### Option 1: Install from source

```bash
git clone https://github.com/liliang-cn/rago.git
cd rago
make setup
make build
```

#### Option 2: Install using go install

```bash
go install github.com/liliang-cn/rago/cmd/rago-cli@latest

# The binary will be named 'rago-cli'
rago-cli --help
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
   ./build/rago serve --port 7127
   ```

4. **Start with Web UI**

   ```bash
   ./build/rago serve --port 7127 --ui
   # Access web interface at http://localhost:7127
   ```

5. **List Imported Documents**
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

# Query with filters (requires documents with metadata)
rago query "machine learning concepts" --filter "source=textbook" --filter "category=ai"
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
rago serve --port 7127 --host 0.0.0.0
```

Start with Web UI:

```bash
rago serve --port 7127 --ui
# Access web interface at http://localhost:7127
```

The `--ui` flag enables the built-in web interface that provides:

- **Document Upload** - Drag and drop files or paste text content
- **Interactive Chat** - Real-time conversation with your documents
- **Document Management** - View, search, and delete documents
- **Advanced Options** - Control streaming, AI thinking process, and filters
- **Responsive Design** - Works on desktop and mobile devices

#### Web UI Features

**Chat Interface:**

- Real-time streaming responses
- Toggle AI thinking process visibility
- Document source citations
- Filter by document metadata
- Advanced settings panel

**Document Management:**

- Upload files (TXT, MD, PDF support planned)
- Paste text content directly
- View all ingested documents
- Search through documents
- Delete individual documents
- Bulk reset functionality

**Configuration:**

- Streaming response control
- Show/hide AI reasoning
- Custom metadata filters
- Adjustable response parameters

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
  "stream": false,
  "show_thinking": false,
  "filters": {
    "source": "textbook",
    "category": "ai"
  }
}
```

**Streaming Query**

```bash
POST /api/query-stream
Content-Type: application/json

{
  "query": "Explain machine learning",
  "show_thinking": true,
  "filters": {
    "category": "ml"
  }
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
  "top_k": 5,
  "filters": {
    "source": "textbook",
    "category": "ai"
  }
}
```

#### Filtering Support

RAGO supports filtering search results based on document metadata. This allows you to search within specific subsets of your knowledge base:

**CLI Usage:**

```bash
# Query with filters
rago query "machine learning" --filter "source=textbook" --filter "author=John Doe"

# Search only (no generation) with filters
rago search "neural networks" --filter "category=deep-learning" --filter "year=2023"
```

**API Usage:**

```bash
# Query with filters
curl -X POST http://localhost:7127/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "What is machine learning?",
    "filters": {
      "source": "textbook",
      "category": "ai"
    }
  }'

# Search with filters
curl -X POST http://localhost:7127/api/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "neural networks",
    "filters": {
      "category": "deep-learning"
    }
  }'
```

**Note:** Documents must have appropriate metadata fields set during ingestion for filtering to work effectively.

### Library Usage

RAGO can be used as a Go library in your projects. This allows you to integrate RAGO's RAG functionality directly into your applications without running it as a separate CLI tool.

#### Installation

```bash
go get github.com/liliang-cn/rago
```

#### Import the library

```go
import "github.com/liliang-cn/rago/lib"
```

#### Create a client

```go
// Using default config file (config.toml in current directory)
client, err := rago.New("config.toml")
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Or using a custom config
cfg := &config.Config{
    // ... your config here
}
client, err := rago.NewWithConfig(cfg)
```

#### Basic operations

```go
// Ingest text content
err = client.IngestText("Your text content here", "source_name")

// Ingest a file
err = client.IngestFile("/path/to/your/file.txt")

// Query the knowledge base
response, err := client.Query("Your question here")
fmt.Println("Answer:", response.Answer)

// Query with filters
filters := map[string]interface{}{
    "source": "textbook",
    "category": "ai",
}
response, err := client.QueryWithFilters("Your filtered question", filters)
fmt.Println("Filtered Answer:", response.Answer)

// Stream query with callback
err = client.StreamQuery("Your question", func(chunk string) {
    fmt.Print(chunk)
})

// Stream query with filters
err = client.StreamQueryWithFilters("Your filtered question", filters, func(chunk string) {
    fmt.Print(chunk)
})

// List documents
docs, err := client.ListDocuments()

// Delete a document
err = client.DeleteDocument(documentID)

// Reset the database
err = client.Reset()
```

#### Library Configuration

The library uses the same configuration format as the CLI tool. You can either:

1. Pass a config file path to `rago.New(configPath)`
2. Load config yourself and pass it to `rago.NewWithConfig(config)`

The library will read configuration from:

- Specified config file path
- `./config.toml` (current directory)
- `./config/config.toml`
- `$HOME/.rago/config.toml`

#### Example

See `examples/library_usage.go` for a complete example of how to use RAGO as a library.

```bash
cd examples
go run library_usage.go
```

#### API Reference

**Client Methods**

- `New(configPath string) (*Client, error)` - Create client with config file
- `NewWithConfig(cfg *config.Config) (*Client, error)` - Create client with config struct
- `IngestFile(filePath string) error` - Ingest a file
- `IngestText(text, source string) error` - Ingest text content
- `Query(query string) (domain.QueryResponse, error)` - Query knowledge base
- `QueryWithFilters(query string, filters map[string]interface{}) (domain.QueryResponse, error)` - Query with metadata filters
- `StreamQuery(query string, callback func(string)) error` - Stream query response
- `StreamQueryWithFilters(query string, filters map[string]interface{}, callback func(string)) error` - Stream query with filters
- `ListDocuments() ([]domain.Document, error)` - List all documents
- `DeleteDocument(documentID string) error` - Delete a document
- `Reset() error` - Reset the database
- `Close() error` - Close the client and cleanup
- `GetConfig() *config.Config` - Get current configuration

## ⚙️ Configuration

### Configuration File

Create `config.toml`:

```toml
[server]
port = 7127
host = "localhost"
enable_ui = true
cors_origins = ["*"]

[ollama]
embedding_model = "nomic-embed-text"
llm_model = "qwen3"
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
export RAGO_SERVER_PORT=7127
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
  -p 7127:7127 \
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
      - "7127:7127"
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
- Email: liliang.imut@gmail.com

---

⭐ If this project helps you, please give it a Star!
