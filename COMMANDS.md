# RAGO CLI Command Guide

RAGO (Retrieval-Augmented Generation Offline) provides a powerful command-line interface for managing your local knowledge base, chatting with AI, and utilizing MCP tools.

## 🚀 Basic Usage

The fastest way to get started is using environment variables:

```bash
export RAGO_OPENAI_API_KEY="your-key" # or "ollama" for local
export RAGO_OPENAI_BASE_URL="https://api.openai.com/v1" # or "http://localhost:11434/v1"
```

## 📚 1. Knowledge Base (`rago rag`)

Manage your documents and perform RAG queries.

### Ingest Documents
Import documents into your vector database. Supports PDF, Markdown, Text, and URLs.

```bash
# Ingest a single file
rago rag ingest document.pdf

# Ingest a directory (recursive)
rago rag ingest ./docs/ --recursive

# Ingest with specific collection
rago rag ingest paper.pdf --collection research

# Enable GraphRAG (Knowledge Graph Extraction) during ingest
# Note: This uses more tokens but enables smarter entity-aware retrieval
rago rag ingest complex_doc.pdf --graph
```

### Query Knowledge Base
Ask questions based on your ingested documents.

```bash
# Basic query
rago rag query "What is the main conclusion of the paper?"

# Enable streaming output
rago rag query "Summarize the document" -s

# Show reasoning process (for R1/O1 models)
rago rag query "Analyze the findings" --thinking

# Enable MCP tools (Agentic RAG)
rago rag query "Read the file and summarize it" --mcp
```

### Maintenance
```bash
# List all indexed documents
rago rag list

# Delete a specific document
rago rag delete <doc_id>

# Reset the entire database (Danger!)
rago rag reset
```

## 💬 2. Chat with Memory (`rago chat`)

Start a stateful conversation where the AI remembers previous context and can recall information from history.

```bash
# Start a new chat session (returns session ID)
rago chat start

# Chat in a specific session
rago chat query --session <session_id> "Hello, my name is Alice."

# Ask follow-up (AI remembers your name via Semantic Recall)
rago chat query --session <session_id> "Do you remember who I am?"

# View chat history
rago chat history --session <session_id>
```

## 🛠️ 3. MCP Tools / Skills (`rago mcp`)

Inspect and test available tools (Skills) connected via the Model Context Protocol.

```bash
# List all available tools
rago mcp list

# Call a tool manually (for debugging)
rago mcp call filesystem read_file path=README.md
```

## 👤 4. Profiles (`rago profile`)

Manage different AI personas or configurations (e.g., "Coder", "Writer").

```bash
# Create a new profile
rago profile create "coder" --description "Expert Go developer"

# List all profiles
rago profile list

# Switch active profile
rago profile switch <profile_id>
```

## ⚙️ 5. System & Config (`rago`)

### Server
Start the REST API server and Web UI.

```bash
# Start server on default port (7127)
rago serve

# Start with Web UI enabled
rago serve --ui
```

### Configuration
```bash
# Generate a default configuration file
rago config init

# View current effective configuration
rago config view
```

### Diagnostics
Check system health, database connection, and LLM availability.

```bash
rago status
```

---

## 📝 Configuration Reference (`rago.toml`)

Minimal required configuration for local LLMs (Ollama):

```toml
[providers.openai]
base_url = "http://localhost:11434/v1"
api_key = "ollama"
llm_model = "qwen2.5:7b"
embedding_model = "nomic-embed-text"
```
