# RAGO Integration Test

This directory contains a simple integration test for the RAGO project that demonstrates:

1. Initializing the vector store
2. Ingesting documents with metadata
3. Generating embeddings for document chunks
4. Storing documents and chunks in the database
5. Listing stored documents
6. Performing semantic search queries

## Running the Test

To run the integration test:

```bash
cd /path/to/rago
go run examples/integration/main.go
```

## What the Test Does

1. Sets up a configuration for the SQLite vector store and Ollama embedder
2. Creates a fresh database instance
3. Ingests three sample documents about Go programming language
4. Generates embeddings for each document chunk
5. Stores the documents and chunks in the database
6. Lists all stored documents
7. Performs a semantic search query and displays results

The test demonstrates the core functionality of RAGO in a single, self-contained example.