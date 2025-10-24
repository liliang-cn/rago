# RAGO v2 Library Usage Guide

This guide demonstrates how to use RAGO v2 as a library in your Go projects. All the core RAG functionality is now available through a clean, intuitive API.

## Table of Contents

1. [Basic Setup](#basic-setup)
2. [Configuration](#configuration)
3. [Core RAG Operations](#core-rag-operations)
4. [Profile Management](#profile-management)
5. [Advanced Features](#advanced-features)
6. [Complete Examples](#complete-examples)

## Basic Setup

### Installation

```bash
go get github.com/liliang-cn/rago/v2
```

### Minimal Working Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/liliang-cn/rago/v2/pkg/rag"
    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/providers"
)

func main() {
    ctx := context.Background()

    // 1. Initialize configuration
    cfg, _ := config.Load("")  // Uses defaults

    // 2. Configure for Ollama (local LLM)
    cfg.Providers.DefaultLLM = "openai"
    cfg.Providers.OpenAI.BaseURL = "http://localhost:11434/v1"
    cfg.Providers.OpenAI.LLMModel = "qwen3"
    cfg.Providers.OpenAI.EmbeddingModel = "nomic-embed-text"

    // 3. Create providers
    embedder, _ := providers.CreateEmbedderProvider(ctx, cfg.Providers.OpenAI)
    llm, _ := providers.CreateLLMProvider(ctx, cfg.Providers.OpenAI)

    // 4. Create RAG client
    client, _ := rag.NewClient(cfg, embedder, llm, nil)
    defer client.Close()

    // 5. Use RAG functionality
    resp, _ := client.IngestText(ctx, "RAGO is a RAG system", "doc1.txt", rag.DefaultIngestOptions())
    fmt.Printf("Ingested document ID: %s\n", resp.DocumentID)

    queryResp, _ := client.Query(ctx, "What is RAGO?", rag.DefaultQueryOptions())
    fmt.Printf("Answer: %s\n", queryResp.Answer)
}
```

## Configuration

### Using Configuration Files

Create a `rago.toml` file:

```toml
[providers]
default_llm = "openai"
default_embedder = "openai"

[providers.openai]
type = "openai"
base_url = "http://localhost:11434/v1"  # Ollama endpoint
api_key = "ollama"  # Required even for local
llm_model = "qwen3"
embedding_model = "nomic-embed-text"
timeout = "30s"

[sqvect]
db_path = "./data/rag.db"
top_k = 5
threshold = 0.0

[chunker]
chunk_size = 500
overlap = 50
method = "sentence"
```

Then load it in your code:

```go
cfg, err := config.Load("rago.toml")
if err != nil {
    log.Fatal("Failed to load config:", err)
}
```

### Environment Variables

You can also use environment variables:

```bash
export RAGO_OPENAI_BASE_URL="http://localhost:11434/v1"
export RAGO_OPENAI_LLM_MODEL="qwen3"
export RAGO_OPENAI_EMBEDDING_MODEL="nomic-embed-text"
```

## Core RAG Operations

### Document Ingestion

```go
// Ingest from file
resp, err := client.IngestFile(ctx, "document.pdf", rag.DefaultIngestOptions())
if err != nil {
    log.Fatal("Ingest failed:", err)
}
fmt.Printf("Ingested %d chunks\n", resp.ChunkCount)

// Ingest text directly
textResp, err := client.IngestText(ctx, "Your text content", "source.txt", rag.DefaultIngestOptions())
if err != nil {
    log.Fatal("Text ingest failed:", err)
}
fmt.Printf("Document ID: %s\n", textResp.DocumentID)

// Ingest from URL
urlResp, err := client.IngestURL(ctx, "https://example.com", rag.DefaultIngestOptions())
if err != nil {
    log.Fatal("URL ingest failed:", err)
}
fmt.Printf("URL content ingested\n")
```

### Custom Ingestion Options

```go
opts := &rag.IngestOptions{
    ChunkSize:          1000,  // Size of text chunks
    Overlap:            200,   // Overlap between chunks
    EnhancedExtraction: true,  // Enable enhanced metadata extraction
    Metadata: map[string]interface{}{
        "category": "documentation",
        "author":   "RAGO Team",
    },
}

resp, err := client.IngestFile(ctx, "manual.pdf", opts)
```

### Querying the Knowledge Base

```go
// Basic query
queryResp, err := client.Query(ctx, "What is this document about?", rag.DefaultQueryOptions())
if err != nil {
    log.Fatal("Query failed:", err)
}

fmt.Printf("Answer: %s\n", queryResp.Answer)
fmt.Printf("Sources found: %d\n", len(queryResp.Sources))

for i, source := range queryResp.Sources {
    fmt.Printf("Source %d: Score=%.2f, Content preview: %s\n",
        i+1, source.Score, source.Content[:min(100, len(source.Content))])
}
```

### Advanced Query Options

```go
opts := &rag.QueryOptions{
    TopK:         10,     // Number of documents to retrieve
    Temperature:  0.7,    // LLM temperature
    MaxTokens:    2000,   // Maximum tokens in response
    ShowSources:  true,   // Include source documents
    ShowThinking: false,  // Show reasoning process
    Stream:       false,  // Enable streaming response
    ToolsEnabled: false,  // Enable tool calling
    Filters: map[string]interface{}{
        "category": "documentation",
    },
}

queryResp, err := client.Query(ctx, "How does the authentication system work?", opts)
```

### Streaming Queries

```go
opts := &rag.QueryOptions{
    Stream: true,
}

// For streaming, you'll need to implement the streaming callback
// This is a simplified example - actual streaming requires more setup
```

### Document Management

```go
// List all documents
docs, err := client.ListDocuments(ctx)
if err != nil {
    log.Fatal("List failed:", err)
}

for _, doc := range docs {
    fmt.Printf("Document: %s (Created: %s)\n", doc.ID, doc.CreatedAt)
}

// Get statistics
stats, err := client.GetStats(ctx)
if err != nil {
    log.Fatal("Stats failed:", err)
}

fmt.Printf("Total documents: %d\n", stats.TotalDocuments)
fmt.Printf("Total chunks: %d\n", stats.TotalChunks)

// Delete a document
err = client.DeleteDocument(ctx, "document-id")
if err != nil {
    log.Fatal("Delete failed:", err)
}

// Reset entire database
err = client.Reset(ctx)
if err != nil {
    log.Fatal("Reset failed:", err)
}
```

## Profile Management

RAGO supports user profiles to manage different settings and configurations.

### Creating and Managing Profiles

```go
// Create a new profile
profile, err := client.CreateProfile("research", "Profile for research work")
if err != nil {
    log.Fatal("Create profile failed:", err)
}
fmt.Printf("Created profile: %s\n", profile.ID)

// List all profiles
profiles, err := client.ListProfiles()
if err != nil {
    log.Fatal("List profiles failed:", err)
}

for _, profile := range profiles {
    fmt.Printf("Profile: %s - %s (%s)\n", profile.ID, profile.Name, profile.Description)
}

// Get a specific profile
profile, err = client.GetProfile(profile.ID)
if err != nil {
    log.Fatal("Get profile failed:", err)
}

// Update a profile
err = client.UpdateProfile(profile.ID, map[string]interface{}{
    "description": "Updated description",
    "metadata": map[string]string{
        "department": "research",
        "project":    "ai-assistant",
    },
})
if err != nil {
    log.Fatal("Update profile failed:", err)
}

// Set active profile
err = client.SetActiveProfile(profile.ID)
if err != nil {
    log.Fatal("Set active profile failed:", err)
}

// Get current active profile
activeProfile, err := client.GetActiveProfile()
if err != nil {
    log.Fatal("Get active profile failed:", err)
}
fmt.Printf("Active profile: %s\n", activeProfile.Name)
```

### LLM Settings Management

```go
// Get current LLM model
currentModel, err := client.GetLLMModel()
if err != nil {
    log.Fatal("Get LLM model failed:", err)
}
fmt.Printf("Current LLM model: %s\n", currentModel)

// Update LLM model (placeholder - not yet fully implemented)
err = client.UpdateLLMModel("qwen3")
if err != nil {
    fmt.Printf("LLM model update not yet available: %v\n", err)
}
```

## Advanced Features

### Enhanced Generation with Context

```go
// Get some documents manually
docs, err := client.ListDocuments(ctx)
if err != nil {
    log.Fatal("List documents failed:", err)
}

// Create chunks for context (simplified)
var contextDocs []domain.Chunk
// In practice, you would retrieve chunks based on similarity search

// Generate with specific context
response, err := client.GenerateWithContext(ctx,
    "Summarize the key points",
    contextDocs,
    &domain.GenerationOptions{
        Temperature: 0.5,
        MaxTokens:   500,
    })
if err != nil {
    log.Fatal("Generation failed:", err)
}
fmt.Printf("Generated response: %s\n", response)
```

### MCP Tools Integration

MCP (Model Context Protocol) tools integration is planned but not yet fully available in the library API.

```go
// List available tools (placeholder)
tools, err := client.ListTools(ctx)
if err != nil {
    fmt.Printf("MCP tools not yet available: %v\n", err)
}

// Call a tool (placeholder)
result, err := client.CallTool(ctx, "tool_name", map[string]interface{}{
    "param": "value",
})
if err != nil {
    fmt.Printf("MCP tool calls not yet available: %v\n", err)
}

// Get MCP status (placeholder)
status, err := client.GetMCPStatus(ctx)
if err != nil {
    fmt.Printf("MCP status not yet available: %v\n", err)
}
```

## Complete Examples

### Example 1: Basic Document Q&A

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/liliang-cn/rago/v2/pkg/rag"
    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/providers"
)

func main() {
    ctx := context.Background()

    // Setup configuration for Ollama
    cfg := &config.Config{}
    cfg.Providers.DefaultLLM = "openai"
    cfg.Providers.OpenAI = &domain.OpenAIProviderConfig{
        BaseURL:        "http://localhost:11434/v1",
        APIKey:         "ollama",  // Required for OpenAI client lib
        LLMModel:       "qwen3",
        EmbeddingModel: "nomic-embed-text",
    }
    cfg.Sqvect.DBPath = "./data/rag.db"

    // Create providers
    embedder, err := providers.CreateEmbedderProvider(ctx, cfg.Providers.OpenAI)
    if err != nil {
        log.Fatal("Failed to create embedder:", err)
    }

    llm, err := providers.CreateLLMProvider(ctx, cfg.Providers.OpenAI)
    if err != nil {
        log.Fatal("Failed to create LLM:", err)
    }

    // Create client
    client, err := rag.NewClient(cfg, embedder, llm, nil)
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }
    defer client.Close()

    // Ingest some documents
    documents := []string{
        "RAGO is a Retrieval-Augmented Generation system for local use.",
        "It supports document ingestion, semantic search, and Q&A.",
        "RAGO uses SQLite for vector storage and supports multiple LLM providers.",
    }

    for i, doc := range documents {
        resp, err := client.IngestText(ctx, doc, fmt.Sprintf("doc%d.txt", i+1), rag.DefaultIngestOptions())
        if err != nil {
            log.Printf("Failed to ingest document %d: %v", i+1, err)
            continue
        }
        fmt.Printf("Ingested document %d: %s\n", i+1, resp.DocumentID)
    }

    // Query the knowledge base
    questions := []string{
        "What is RAGO?",
        "What are the main features of RAGO?",
        "How does RAGO store vectors?",
    }

    for _, question := range questions {
        resp, err := client.Query(ctx, question, rag.DefaultQueryOptions())
        if err != nil {
            log.Printf("Failed to query '%s': %v", question, err)
            continue
        }

        fmt.Printf("\nQ: %s\nA: %s\n", question, resp.Answer)
        fmt.Printf("Sources: %d\n", len(resp.Sources))
    }

    // Show statistics
    stats, err := client.GetStats(ctx)
    if err != nil {
        log.Printf("Failed to get stats: %v", err)
    } else {
        fmt.Printf("\nDatabase Statistics:\n")
        fmt.Printf("Total Documents: %d\n", stats.TotalDocuments)
        fmt.Printf("Total Chunks: %d\n", stats.TotalChunks)
    }
}
```

### Example 2: Multi-Profile System

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/liliang-cn/rago/v2/pkg/rag"
    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/providers"
)

func main() {
    ctx := context.Background()

    // Setup client (same as Example 1)
    client := setupClient(ctx)
    defer client.Close()

    // Create different profiles for different use cases
    profiles := []struct {
        name        string
        description string
    }{
        {"general", "General purpose Q&A"},
        {"technical", "Technical documentation"},
        {"research", "Research and analysis"},
    }

    // Create profiles
    for _, profile := range profiles {
        _, err := client.CreateProfile(profile.name, profile.description)
        if err != nil {
            log.Printf("Failed to create profile '%s': %v", profile.name, err)
            continue
        }
        fmt.Printf("Created profile: %s\n", profile.name)
    }

    // List all profiles
    allProfiles, err := client.ListProfiles()
    if err != nil {
        log.Fatal("Failed to list profiles:", err)
    }

    fmt.Printf("\nAvailable Profiles:\n")
    for _, profile := range allProfiles {
        activeStatus := ""
        if profile.IsActive {
            activeStatus = " [ACTIVE]"
        }
        fmt.Printf("- %s: %s%s\n", profile.Name, profile.Description, activeStatus)
    }

    // Switch between profiles (placeholder implementation)
    fmt.Printf("\nProfile switching will be available in future versions.\n")
}

func setupClient(ctx context.Context) *rag.Client {
    // Configuration setup (same as Example 1)
    cfg := &config.Config{}
    cfg.Providers.DefaultLLM = "openai"
    cfg.Providers.OpenAI = &domain.OpenAIProviderConfig{
        BaseURL:        "http://localhost:11434/v1",
        APIKey:         "ollama",
        LLMModel:       "qwen3",
        EmbeddingModel: "nomic-embed-text",
    }
    cfg.Sqvect.DBPath = "./data/rag.db"

    embedder, _ := providers.CreateEmbedderProvider(ctx, cfg.Providers.OpenAI)
    llm, _ := providers.CreateLLMProvider(ctx, cfg.Providers.OpenAI)

    client, _ := rag.NewClient(cfg, embedder, llm, nil)
    return client
}
```

## Error Handling

Always handle errors appropriately in your production code:

```go
resp, err := client.Query(ctx, "What is RAGO?", rag.DefaultQueryOptions())
if err != nil {
    // Handle different types of errors
    switch {
    case errors.Is(err, context.Canceled):
        log.Println("Query was canceled")
    case errors.Is(err, rag.ErrDocumentNotFound):
        log.Println("No documents found for query")
    default:
        log.Printf("Query failed: %v", err)
    }
    return
}
```

## Best Practices

1. **Resource Management**: Always call `client.Close()` to release resources
2. **Context Usage**: Use proper context with timeout for production applications
3. **Error Handling**: Implement comprehensive error handling
4. **Configuration**: Use configuration files for production deployments
5. **Testing**: Test with different document types and queries

## Current Limitations

- MCP tools integration is not yet available in library API
- Profile-based configuration switching is partially implemented
- LLM settings management through profiles needs implementation
- Streaming queries require additional setup

These features are planned for future releases.

## Support

For more examples and support, check the [RAGO GitHub repository](https://github.com/liliang-cn/rago).