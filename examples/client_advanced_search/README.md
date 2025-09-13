# Advanced Search and Filtering Example

This example demonstrates sophisticated search capabilities and filtering options in RAGO.

## Features Demonstrated

- Basic semantic search
- Search with context generation
- Similarity search with score thresholds
- Hybrid search (vector + keyword)
- Metadata-based filtering
- Time-based search patterns
- Performance comparisons
- Search result ranking

## Usage

```bash
go run main.go
```

## What It Does

1. **Prepares Knowledge Base**: Ingests diverse content with rich metadata
2. **Basic Search**: Simple semantic search with top-K results
3. **Context Generation**: Creates formatted context from search results
4. **Score Filtering**: Returns only high-confidence matches
5. **Hybrid Search**: Combines vector similarity and keyword matching
6. **Metadata Filtering**: Filters results based on metadata fields
7. **Time-based Search**: Finds recent or date-specific content
8. **Performance Analysis**: Compares different search strategies

## Search Options

```go
type SearchOptions struct {
    TopK            int     // Number of results to return
    ScoreThreshold  float64 // Minimum similarity score
    IncludeMetadata bool    // Include metadata in results
    HybridSearch    bool    // Enable hybrid search
    VectorWeight    float64 // Weight for vector search (0-1)
}
```

## Search Strategies

### Pure Vector Search
- Best for semantic similarity
- Understands context and meaning
- Language-agnostic

### Hybrid Search
- Combines vector and keyword matching
- Better for specific terms and names
- Adjustable weighting between methods

### Filtered Search
- Pre-filter by metadata
- Reduce search space
- Improve relevance for specific domains

## Metadata Schema Example

```go
metadata := map[string]interface{}{
    "category": "programming",
    "language": "go",
    "topic":    "concurrency",
    "year":     2024,
    "level":    "intermediate",
}
```

## Use Cases

- **Technical Documentation**: Search across multiple programming languages
- **Knowledge Management**: Find relevant articles by topic and date
- **Content Discovery**: Discover related content through semantic similarity
- **Filtered Queries**: Search within specific categories or time ranges
- **Multi-modal Search**: Combine different search strategies for best results

## Performance Tips

1. **Hybrid Search**: Use for queries with specific keywords
2. **Pure Vector**: Use for conceptual/semantic queries
3. **Score Threshold**: Set to 0.7+ for high precision
4. **TopK**: Balance between coverage and performance
5. **Metadata**: Pre-index metadata for faster filtering

## Prerequisites

- Embedder provider must be running
- Vector store must be initialized
- For hybrid search: Both vector and keyword indices needed