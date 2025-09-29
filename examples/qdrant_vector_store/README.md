# Qdrant Vector Store Example

This example demonstrates how to use Qdrant as a vector store backend for RAGO.

## Prerequisites

1. Qdrant server running locally or in Docker
2. Go 1.21 or later

## Quick Start

### Option 1: Using Docker Compose (Recommended)

```bash
# Start Qdrant with daocloud mirror
docker-compose up -d

# Wait for Qdrant to be ready
sleep 5

# Run the example
go run main.go
```

### Option 2: Using Docker directly

```bash
# Run Qdrant container with daocloud mirror
docker run -d \
  -p 6333:6333 \
  -p 6334:6334 \
  -v $(pwd)/qdrant_storage:/qdrant/storage:z \
  --name qdrant \
  m.daocloud.io/docker.io/qdrant/qdrant:latest

# Run the example
go run main.go
```

### Option 3: Using existing Qdrant instance

If you already have Qdrant running, update the URL in `main.go`:

```go
config := store.StoreConfig{
    Type: "qdrant",
    Parameters: map[string]interface{}{
        "url":        "your-qdrant-host:6334", // Update this
        "collection": "test_documents",
    },
}
```

## Features Demonstrated

1. **Store documents** - Store chunks with embeddings in Qdrant
2. **Vector search** - Find similar documents using cosine similarity
3. **Filtered search** - Search with metadata filters
4. **Delete documents** - Remove specific documents from the collection
5. **Reset collection** - Clear all data from the collection

## Qdrant Web UI

Once Qdrant is running, you can access the web UI at:
- http://localhost:6333

## Configuration for RAGO

To use Qdrant as your default vector store in RAGO, update your configuration:

### Using config file (rago.toml):

```toml
[vector_store]
type = "qdrant"

[vector_store.parameters]
url = "localhost:6334"
collection = "rago_documents"
```

### Using environment variables:

```bash
export RAGO_VECTOR_STORE_TYPE=qdrant
export RAGO_VECTOR_STORE_URL=localhost:6334
export RAGO_VECTOR_STORE_COLLECTION=rago_documents
```

### Using command-line flags:

```bash
rago query "your question" \
  --vector-store-type qdrant \
  --vector-store-url localhost:6334 \
  --vector-store-collection rago_documents
```

## Cleanup

```bash
# Stop and remove Qdrant container
docker-compose down

# Remove storage volume (WARNING: This deletes all data)
rm -rf qdrant_storage
```

## Troubleshooting

1. **Connection refused**: Ensure Qdrant is running and accessible on port 6334
2. **Collection not found**: The example will automatically create the collection
3. **Memory issues**: Qdrant may require significant RAM for large datasets

## Performance Tips

1. Use batch operations when storing multiple documents
2. Configure appropriate index settings for your use case
3. Monitor memory usage for large collections
4. Use filters to reduce search space when possible

## Learn More

- [Qdrant Documentation](https://qdrant.tech/documentation/)
- [RAGO Documentation](https://github.com/liliang-cn/rago)
- [Vector Search Concepts](https://qdrant.tech/documentation/concepts/)