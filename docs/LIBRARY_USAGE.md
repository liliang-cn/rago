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

MCP (Model Context Protocol) tools integration is fully functional! You can manage MCP servers and call tools programmatically.

```go
// Get MCP service status
status, err := client.GetMCPStatus(ctx)
if err != nil {
    log.Printf("Failed to get MCP status: %v", err)
} else {
    fmt.Printf("MCP Status: %+v\n", status)
    // Output: map[enabled:true message:MCP service operational servers:[...]]
}

// List available MCP tools
tools, err := client.ListTools(ctx)
if err != nil {
    log.Printf("Failed to list tools: %v", err)
} else {
    fmt.Printf("Available tools: %d\n", len(tools))
    for _, tool := range tools {
        if toolMap, ok := tool.(map[string]interface{}); ok {
            fmt.Printf("- %s: %s (from %s)\n",
                toolMap["name"], toolMap["description"], toolMap["server"])
        }
    }
}

// Call an MCP tool
result, err := client.CallTool(ctx, "filesystem_read_file", map[string]interface{}{
    "path": "/path/to/file.txt",
})
if err != nil {
    log.Printf("Tool call failed: %v", err)
} else {
    fmt.Printf("Tool result: %+v\n", result)
    // Output: map[success:true data:file_content error:]
}
```

#### MCP Configuration

Configure MCP servers in `mcpServers.json`:

```json
{
  "filesystem": {
    "command": "npx",
    "args": ["@modelcontextprotocol/server-filesystem", "/allowed/path"],
    "description": "File system operations",
    "auto_start": true
  },
  "fetch": {
    "command": "npx",
    "args": ["@modelcontextprotocol/server-fetch"],
    "description": "HTTP/HTTPS requests",
    "auto_start": true
  }
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

### Example 2: Complete Profile Management

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/liliang-cn/rago/v2/pkg/rag"
    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/domain"
    "github.com/liliang-cn/rago/v2/pkg/providers"
    "github.com/liliang-cn/rago/v2/pkg/settings"
)

func main() {
    ctx := context.Background()

    // Setup client
    client := setupClient(ctx)
    defer client.Close()

    fmt.Println("=== Profile Management Demo ===")

    // 1. List existing profiles
    profiles, err := client.ListProfiles()
    if err != nil {
        log.Fatal("Failed to list profiles:", err)
    }

    fmt.Printf("Existing profiles: %d\n", len(profiles))
    for _, profile := range profiles {
        status := ""
        if profile.IsActive {
            status = " [ACTIVE]"
        }
        fmt.Printf("  - %s: %s%s\n", profile.Name, profile.Description, status)
    }

    // 2. Create specialized profiles
    researchProfile, err := client.CreateProfile(
        "research",
        "Profile for academic research and analysis",
    )
    if err != nil {
        log.Printf("Failed to create research profile: %v", err)
    } else {
        fmt.Printf("Created research profile: %s\n", researchProfile.ID)
    }

    devProfile, err := client.CreateProfile(
        "development",
        "Profile for software development tasks",
    )
    if err != nil {
        log.Printf("Failed to create dev profile: %v", err)
    } else {
        fmt.Printf("Created dev profile: %s\n", devProfile.ID)
    }

    // 3. Configure LLM settings for profiles
    configureLLMSettings(ctx, client, researchProfile, "research")
    configureLLMSettings(ctx, client, devProfile, "development")

    // 4. Switch between profiles and test
    fmt.Println("\n=== Testing Profile Switching ===")

    // Switch to research profile
    err = client.SetActiveProfile(researchProfile.ID)
    if err != nil {
        log.Printf("Failed to set active profile: %v", err)
    } else {
        fmt.Printf("Switched to research profile\n")
        testProfileQuery(ctx, client, "What are the latest research trends in AI?")
    }

    // Switch to development profile
    err = client.SetActiveProfile(devProfile.ID)
    if err != nil {
        log.Printf("Failed to set active profile: %v", err)
    } else {
        fmt.Printf("Switched to development profile\n")
        testProfileQuery(ctx, client, "How do I implement a REST API in Go?")
    }

    // Switch back to default profile
    if len(profiles) > 0 {
        for _, profile := range profiles {
            if profile.IsActive {
                err = client.SetActiveProfile(profile.ID)
                if err == nil {
                    fmt.Printf("Switched back to default profile: %s\n", profile.Name)
                }
                break
            }
        }
    }
}

func configureLLMSettings(ctx context.Context, client *rag.Client, profile *rag.UserProfile, useCase string) {
    // Create LLM settings request
    settingsReq := settings.CreateLLMSettingsRequest{
        ProfileID:    profile.ID,
        ProviderName: "openai",
        SystemPrompt: getSystemPrompt(useCase),
        Temperature:  getTemperature(useCase),
        MaxTokens:    getMaxTokens(useCase),
        Settings: map[string]interface{}{
            "model":       "qwen3",
            "use_case":    useCase,
            "created_by":  "rago-client",
        },
    }

    // Update LLM settings using the settings service
    settingsService := client.GetSettings()
    llmSettings, err := settingsService.CreateOrUpdateLLMSettings(settingsReq)
    if err != nil {
        log.Printf("Failed to configure LLM settings for %s: %v", profile.Name, err)
    } else {
        fmt.Printf("Configured LLM settings for %s profile\n", profile.Name)
        fmt.Printf("  System Prompt: %s\n", llmSettings.SystemPrompt[:min(50, len(llmSettings.SystemPrompt))] + "...")
        fmt.Printf("  Temperature: %.1f\n", *llmSettings.Temperature)
    }
}

func getSystemPrompt(useCase string) string {
    switch useCase {
    case "research":
        return "You are a research assistant. Provide detailed, accurate, and well-cited responses. Focus on academic rigor and evidence-based answers."
    case "development":
        return "You are a software development assistant. Provide clear, practical code examples and best practices. Focus on production-ready solutions."
    default:
        return "You are a helpful assistant. Provide accurate and useful information."
    }
}

func getTemperature(useCase string) *float64 {
    switch useCase {
    case "research":
        temp := 0.3
        return &temp
    case "development":
        temp := 0.1
        return &temp
    default:
        temp := 0.7
        return &temp
    }
}

func getMaxTokens(useCase string) *int {
    switch useCase {
    case "research":
        maxTokens := 3000
        return &maxTokens
    case "development":
        maxTokens := 2000
        return &maxTokens
    default:
        maxTokens := 1500
        return &maxTokens
    }
}

func testProfileQuery(ctx context.Context, client *rag.Client, query string) {
    // Ingest some test content
    testContent := fmt.Sprintf("Test content for query: %s", query)
    _, err := client.IngestText(ctx, testContent, "test.txt", rag.DefaultIngestOptions())
    if err != nil {
        log.Printf("Failed to ingest test content: %v", err)
        return
    }

    // Query with current profile
    resp, err := client.Query(ctx, query, rag.DefaultQueryOptions())
    if err != nil {
        log.Printf("Failed to query: %v", err)
        return
    }

    fmt.Printf("Query: %s\n", query)
    fmt.Printf("Answer: %s\n", resp.Answer[:min(200, len(resp.Answer))] + "...")
    fmt.Printf("Sources: %d\n\n", len(resp.Sources))
}

func setupClient(ctx context.Context) *rag.Client {
    cfg := &config.Config{}
    cfg.Providers.DefaultLLM = "openai"
    cfg.Providers.ProviderConfigs = domain.ProviderConfig{
        OpenAI: &domain.OpenAIProviderConfig{
            BaseURL:        "http://localhost:11434/v1",
            APIKey:         "ollama",
            LLMModel:       "qwen3",
            EmbeddingModel: "nomic-embed-text",
        },
    }
    cfg.Sqvect.DBPath = "./data/rag.db"
    cfg.MCP.Enabled = true

    factory := providers.NewFactory()
    embedder, _ := factory.CreateEmbedderProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)
    llm, _ := factory.CreateLLMProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)

    client, _ := rag.NewClient(cfg, embedder, llm, nil)
    return client
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

### Example 3: MCP Tools Integration

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/liliang-cn/rago/v2/pkg/rag"
)

func main() {
    ctx := context.Background()

    // Setup client with MCP enabled
    client := setupClient(ctx)
    defer client.Close()

    fmt.Println("=== MCP Integration Demo ===")

    // 1. Check MCP service status
    status, err := client.GetMCPStatus(ctx)
    if err != nil {
        log.Printf("Failed to get MCP status: %v", err)
        return
    }

    fmt.Printf("MCP Status: %+v\n", status)

    if enabled, ok := status["enabled"].(bool); ok && enabled {
        // 2. List available tools
        tools, err := client.ListTools(ctx)
        if err != nil {
            log.Printf("Failed to list tools: %v", err)
            return
        }

        fmt.Printf("Available MCP tools: %d\n", len(tools))
        for i, tool := range tools {
            if toolMap, ok := tool.(map[string]interface{}); ok {
                fmt.Printf("%d. %s: %s (from %s)\n",
                    i+1,
                    toolMap["name"],
                    toolMap["description"],
                    toolMap["server"])
            }
        }

        // 3. Test tool calls (if tools are available)
        if len(tools) > 0 {
            fmt.Println("\n=== Testing Tool Calls ===")
            testMCPTools(ctx, client, tools)
        } else {
            fmt.Println("\nNo MCP tools available. Configure MCP servers in mcpServers.json")
        }
    } else {
        fmt.Println("MCP is not enabled. Set MCP.Enabled = true in configuration.")
    }
}

func testMCPTools(ctx context.Context, client *rag.Client, tools []interface{}) {
    for i, tool := range tools {
        if i >= 3 { // Test only first 3 tools
            break
        }

        if toolMap, ok := tool.(map[string]interface{}); ok {
            toolName := toolMap["name"].(string)
            fmt.Printf("Testing tool: %s\n", toolName)

            // Try different tool calls based on tool type
            var arguments map[string]interface{}
            switch {
            case contains(toolName, "filesystem"):
                arguments = map[string]interface{}{
                    "path": "/tmp/test.txt",
                }
            case contains(toolName, "fetch"):
                arguments = map[string]interface{}{
                    "url": "https://httpbin.org/json",
                }
            case contains(toolName, "memory"):
                arguments = map[string]interface{}{
                    "key":   "test_key",
                    "value": "test_value",
                }
            default:
                arguments = map[string]interface{}{
                    "test": "value",
                }
            }

            result, err := client.CallTool(ctx, toolName, arguments)
            if err != nil {
                log.Printf("Failed to call tool %s: %v", toolName, err)
            } else {
                fmt.Printf("Tool %s result: %+v\n", toolName, result)
            }
        }
    }
}

func contains(s, substr string) bool {
    return len(s) >= len(substr) && (s == substr ||
           (len(s) > len(substr) &&
            (s[:len(substr)] == substr ||
             s[len(s)-len(substr):] == substr ||
             indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return i
        }
    }
    return -1
}

func setupClient(ctx context.Context) *rag.Client {
    // Same setup as previous examples with MCP enabled
    cfg := &config.Config{}
    cfg.Providers.DefaultLLM = "openai"
    cfg.Providers.ProviderConfigs = domain.ProviderConfig{
        OpenAI: &domain.OpenAIProviderConfig{
            BaseURL:        "http://localhost:11434/v1",
            APIKey:         "ollama",
            LLMModel:       "qwen3",
            EmbeddingModel: "nomic-embed-text",
        },
    }
    cfg.Sqvect.DBPath = "./data/rag.db"
    cfg.MCP.Enabled = true
    cfg.MCP.ServersConfigPath = "mcpServers.json"

    factory := providers.NewFactory()
    embedder, _ := factory.CreateEmbedderProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)
    llm, _ := factory.CreateLLMProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)

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

- Streaming queries require additional setup
- MCP tools require proper server configuration in `mcpServers.json`

All other features including Profile Management and LLM Settings are fully functional!

## Support

For more examples and support, check the [RAGO GitHub repository](https://github.com/liliang-cn/rago).