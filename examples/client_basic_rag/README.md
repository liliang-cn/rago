# Basic RAG Operations Example

This example demonstrates basic Retrieval-Augmented Generation (RAG) operations using the RAGO client.

## Features Demonstrated

- Text ingestion
- File ingestion
- Simple queries
- Queries with source documents
- Document listing
- Enhanced metadata extraction

## Usage

```bash
# Run with just text ingestion
go run main.go

# Run with file ingestion
go run main.go /path/to/document.pdf
```

## What It Does

1. **Ingests Text**: Adds a sample text about Go programming to the knowledge base
2. **Ingests File** (optional): If you provide a file path, it will ingest that file
3. **Simple Query**: Asks a question and gets an answer
4. **Query with Sources**: Gets an answer along with the source documents used
5. **Lists Documents**: Shows all documents in the knowledge base
6. **Enhanced Metadata**: Displays documents with extracted metadata like summaries and keywords

## Configuration

The example uses the default configuration. To use a custom config:

```go
ragClient, err := client.New("/path/to/config.toml")
```

## Prerequisites

- RAGO must be initialized: `rago init`
- LLM provider (e.g., Ollama) must be running
- Embedder provider must be available