# RAGO - Local RAG System

[中文文档](README_zh-CN.md)

RAGO (Retrieval-Augmented Generation Offline) is a fully local RAG system written in Go, integrating SQLite vector database (sqvect) and multiple LLM providers (OpenAI, Ollama), supporting document ingestion, semantic search, tool calling, and context-enhanced Q&A.

## 🎯 Features

- **Multiple LLM Providers** - Support for OpenAI, Ollama, and other compatible providers
- **Flexible Configuration** - Modern provider-based architecture for easy switching between services
- **Tool Calling Support** - Built-in tools for web search, file operations, datetime, and more
- **Fully Offline Option** - Use with Ollama for complete data privacy
- **Multi-format Support** - Supports PDF, TXT, and Markdown formats
- **Local Vector Database** - SQLite-based sqvect vector storage
- **Web UI Interface** - Built-in web interface for easy interaction
- **Dual Interface Design** - Both CLI tool and HTTP API usage modes
- **High Performance** - Go implementation with low memory usage and fast response
- **Extensible** - Modular design, easy to extend new features

## 🚀 Quick Start

### Prerequisites

**Option 1: For Fully Local Setup (Ollama)**

1. **Install Go** (≥ 1.21)
2. **Install Ollama**
   ```bash
   curl -fsSL https://ollama.com/install.sh | sh
   ```
3. **Download Models**
   ```bash
   ollama pull nomic-embed-text  # Embedding model
   ollama pull qwen2.5          # Generation model (or qwen3)
   ```

**Option 2: For OpenAI Setup**

1. **Install Go** (≥ 1.21)
2. **Get OpenAI API Key** from [platform.openai.com](https://platform.openai.com)

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
go install github.com/liliang-cn/rago@latest

# The binary will be named 'rago'
rago --help
```

### Basic Usage

After building the project with `make build`, you can use the `rago` binary in the `build` directory.

1. **Initialize Configuration**

   ```bash
   ./build/rago init                    # Create config.toml with Ollama defaults
   ./build/rago init --force            # Overwrite existing config file
   ./build/rago init -o custom.toml     # Create config at custom path
   ```

   The `init` command creates:

   - Modern provider-based configuration with Ollama as default
   - Complete directory structure (./data/)
   - Tool calling enabled by default
   - Web UI enabled by default
   - Example OpenAI configuration (commented out)

2. **Configure Providers** (if using OpenAI)

   Edit the generated `config.toml` and uncomment the OpenAI section:

   ```toml
   [providers]
   default_llm = "openai"           # Change from "ollama" to "openai"
   default_embedder = "openai"      # Change from "ollama" to "openai"

   [providers.openai]
   type = "openai"
   api_key = "your-openai-api-key-here"
   # ... other settings
   ```

3. **Ingest Documents**

   ```bash
   ./build/rago ingest ./docs/sample.md
   ./build/rago ingest ./docs/ --recursive  # Process directory recursively
   ```

4. **Query Knowledge Base**

   ```bash
   ./build/rago query "What is RAG?"
   ./build/rago query --interactive         # Interactive mode
   ```

5. **Start API Service with Web UI**

   ```bash
   ./build/rago serve --port 7127
   # Access web interface at http://localhost:7127
   ```

6. **Check Status**

   ```bash
   ./build/rago status                      # Check provider connections
   ```

7. **Tool Calling Examples**

   When using the query command or web interface, RAGO can automatically use built-in tools:

   ```bash
   ./build/rago query "What's the current time in Tokyo?"        # Uses datetime tool
   ./build/rago query "Search for recent AI news"               # Uses web search
   ./build/rago query "What documents do I have about Python?"  # Uses rag_search tool
   ```

## 📖 Detailed Usage

### CLI Commands

#### Configuration Management

```bash
# Initialize configuration with default settings
rago init

# Overwrite existing configuration file
rago init --force

# Create configuration at custom location
rago init --output ./config/rago.toml

# View help for init command
rago init --help
```

#### Document Management

```bash
# Single file ingestion
rago ingest ./document.txt

# Ingest a directory and automatically extract metadata
rago ingest ./docs/ --recursive --extract-metadata

# Batch ingestion with custom chunking
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

### Initialize Configuration

RAGO provides an `init` command to quickly generate a modern configuration file with provider-based architecture:

```bash
# Create config.toml with Ollama defaults and directory structure
rago init

# Overwrite existing configuration file
rago init --force

# Create configuration at custom path
rago init --output /path/to/config.toml
```

The `init` command automatically:

- Creates a modern provider-based configuration
- Sets up complete directory structure (./data/, ./data/documents/, etc.)
- Enables tool calling by default
- Enables Web UI by default
- Includes commented OpenAI configuration for easy switching

### Provider Configuration

RAGO uses a flexible provider system supporting multiple LLM and embedding services:

#### Ollama Configuration (Default)

```toml
[providers]
default_llm = "ollama"
default_embedder = "ollama"

[providers.ollama]
type = "ollama"
base_url = "http://localhost:11434"
llm_model = "qwen2.5"
embedding_model = "nomic-embed-text"
timeout = "120s"
```

#### OpenAI Configuration

```toml
[providers]
default_llm = "openai"
default_embedder = "openai"

[providers.openai]
type = "openai"
api_key = "your-openai-api-key-here"
base_url = "https://api.openai.com/v1"          # Optional: for custom endpoints
llm_model = "gpt-4o-mini"
embedding_model = "text-embedding-3-small"
timeout = "60s"
```

#### Mixed Provider Setup

You can use different providers for LLM and embeddings:

```toml
[providers]
default_llm = "openai"      # Use OpenAI for generation
default_embedder = "ollama" # Use Ollama for embeddings

# Configure both providers
[providers.openai]
type = "openai"
api_key = "your-api-key"
llm_model = "gpt-4o-mini"

[providers.ollama]
type = "ollama"
base_url = "http://localhost:11434"
embedding_model = "nomic-embed-text"
```

### Tool Configuration

RAGO includes built-in tools that can be enabled/disabled:

```toml
[tools]
enabled = true                           # Enable tool calling
max_concurrent_calls = 5                 # Max parallel tool calls
call_timeout = "30s"                     # Timeout per tool call
security_level = "normal"                # strict, normal, relaxed
log_level = "info"                       # debug, info, warn, error

# Available built-in tools
enabled_tools = [
    "datetime",                          # Date and time operations
    "rag_search",                        # Search in RAG database
    "document_info",                     # Document information queries
    "open_url",                       # HTTP web requests
    "web_search",                     # Google search functionality
    "file_operations"                    # File system operations
]

# Tool-specific configurations
[tools.builtin.web_search]
enabled = true
api_key = "your-google-api-key"         # Optional
search_engine_id = "your-cse-id"        # Optional

[tools.builtin.open_url]
enabled = true
timeout = "10s"
max_redirects = 5
user_agent = "RAGO/1.3.1"

[tools.builtin.file_operations]
enabled = true
max_file_size = "10MB"
allowed_extensions = [".txt", ".md", ".json", ".csv", ".log"]
base_directory = "./data"
```

### Complete Configuration Example

```toml
[server]
port = 7127
host = "0.0.0.0"
enable_ui = true
cors_origins = ["*"]

[providers]
default_llm = "ollama"
default_embedder = "ollama"

[providers.ollama]
type = "ollama"
base_url = "http://localhost:11434"
llm_model = "qwen2.5"
embedding_model = "nomic-embed-text"
timeout = "120s"

[sqvect]
db_path = "./data/rag.db"
vector_dim = 768                         # 768 for nomic-embed-text, 1536 for OpenAI
max_conns = 10
batch_size = 100
top_k = 5
threshold = 0.0

[keyword]
index_path = "./data/keyword.bleve"

[chunker]
chunk_size = 500
overlap = 50
method = "sentence"                      # sentence, paragraph, token

[ingest.metadata_extraction]
enable = false                           # Enable automatic metadata extraction
llm_model = "qwen2.5"                   # Model for metadata extraction

[tools]
enabled = true
enabled_tools = ["datetime", "rag_search", "open_url", "web_search"]
```

### Environment Variables

```bash
export RAGO_SERVER_PORT=7127
export RAGO_PROVIDERS_DEFAULT_LLM=openai
export RAGO_PROVIDERS_OPENAI_API_KEY=your-key-here
export RAGO_SQVECT_DB_PATH=./data/custom.sqlite
```

## 🛠️ Tool Calling

RAGO includes a comprehensive tool system that allows the AI to perform actions and retrieve real-time information:

### Built-in Tools

#### 🕐 DateTime Tool

- Get current date and time
- Convert between timezones
- Calculate time differences

**Examples:**

```bash
"What time is it now?"
"What's the current date in Tokyo?"
"How many days until Christmas?"
```

#### 🔍 RAG Search Tool

- Search through your ingested documents
- Retrieve specific information from knowledge base
- Cross-reference document sources

**Examples:**

```bash
"What documents mention Python programming?"
"Search my notes for information about machine learning"
"Find all references to API documentation"
```

#### 📄 Document Info Tool

- Get metadata about ingested documents
- List available documents
- Check document statistics

**Examples:**

```bash
"How many documents do I have?"
"What are the latest documents I added?"
"Show me document statistics"
```

#### 🌐 Web Request Tool

- Make HTTP requests to APIs
- Fetch web content
- Access real-time data

**Examples:**

```bash
"Get the latest news from example.com/api"
"Fetch data from this REST API endpoint"
"Check the status of this website"
```

#### 🔎 Google Search Tool

- Search the internet via Google
- Get recent information
- Find specific resources

**Examples:**

```bash
"Search for recent developments in AI"
"What are the latest news about quantum computing?"
"Find documentation for React hooks"
```

#### 📁 File Operations Tool

- Read local files
- List directory contents
- Check file information

**Examples:**

```bash
"Read the contents of my todo.txt file"
"List files in the ./projects directory"
"What's in my configuration files?"
```

### Tool Configuration

Tools can be enabled/disabled and configured individually:

```toml
[tools]
enabled = true
max_concurrent_calls = 5
call_timeout = "30s"
security_level = "normal"  # Controls tool access levels

# Enable specific tools
enabled_tools = [
    "datetime",
    "rag_search",
    "document_info",
    "open_url",
    "web_search",
    "file_operations"
]

# Tool-specific settings
[tools.builtin.file_operations]
enabled = true
base_directory = "./data"                           # Restrict file access to this directory
allowed_extensions = [".txt", ".md", ".json"]       # Only allow these file types
max_file_size = "10MB"                              # Maximum file size to read

[tools.builtin.open_url]
enabled = true
timeout = "10s"
max_redirects = 5
user_agent = "RAGO/1.3.1"

[tools.builtin.web_search]
enabled = true
# api_key = "your-google-api-key"        # Optional: for better rate limits
# search_engine_id = "your-cse-id"       # Optional: for custom search engine
```

### Security Levels

- **strict**: Very limited tool access, safe for production
- **normal**: Balanced security and functionality (default)
- **relaxed**: Full tool access, use with caution

### Tool Usage in API

Tools are automatically invoked when using the API:

```bash
# Query that will trigger tool usage
curl -X POST http://localhost:7127/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "What time is it and search for recent AI news?",
    "stream": false
  }'
```

The response will include both the tool results and the AI's synthesized answer.

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
│   ├── domain/        # Domain models and interfaces
│   ├── providers/     # LLM and embedder providers (OpenAI, Ollama)
│   ├── chunker/       # Text chunking
│   ├── embedder/      # Embedding service layer
│   ├── llm/           # Generation service layer
│   ├── store/         # Storage layer (SQLite, Bleve)
│   ├── processor/     # Core processor
│   ├── tools/         # Tool calling system
│   └── utils/         # Utility functions
├── api/handlers/       # API handlers
├── ui/                # Web UI assets
├── test/              # Integration tests
├── docs/              # Documentation
├── examples/          # Usage examples
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
