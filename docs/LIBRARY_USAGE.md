# RAGO: Industrial-Grade AI Library for Go

RAGO is not just a CLI tool; it is a high-performance, concurrency-safe **AI SDK** designed to be embedded into your Go applications. It empowers your software with private Knowledge Base (RAG), Semantic Chat Memory, and Agentic capabilities (MCP).

## 🚀 Why RAGO as a Library?

*   **Zero Infrastructure**: Powered by embedded `sqvect` (SQLite), requires no external vector DB deployment (Milvus/Qdrant optional).
*   **Unified Interface**: One client for Documents, Knowledge Graph, and Chat History.
*   **Standards Compliant**: Native support for OpenAI API formats and Model Context Protocol (MCP).
*   **Thread-Safe**: Built for high-concurrency web services (Goroutine-safe).

## 📦 Installation

```bash
go get github.com/liliang-cn/rago/v2
```

## ⚡️ Quick Integration

Embed a complete RAG system into your app in under 30 lines of code.

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/domain"
    "github.com/liliang-cn/rago/v2/pkg/providers"
    "github.com/liliang-cn/rago/v2/pkg/rag"
)

func main() {
    ctx := context.Background()

    // 1. Configure (Minimal setup for local LLM)
    cfg := &config.Config{
        Sqvect: config.SqvectConfig{
            DBPath: "./data/app.db", // Your app's private DB
        },
        Providers: config.ProvidersConfig{
            DefaultLLM: "openai",
            ProviderConfigs: domain.ProviderConfig{
                OpenAI: &domain.OpenAIProviderConfig{
                    BaseURL: "http://localhost:11434/v1", // Ollama
                    APIKey:  "ollama",
                    LLMModel: "qwen2.5:7b",
                    EmbeddingModel: "nomic-embed-text",
                },
            },
        },
    }

    // 2. Initialize Providers
    factory := providers.NewFactory()
    llm, _ := factory.CreateLLMProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)
    embedder, _ := factory.CreateEmbedderProvider(ctx, cfg.Providers.ProviderConfigs.OpenAI)

    // 3. Create RAG Client
    // Pass 'llm' as metadata extractor for GraphRAG capabilities
    client, err := rag.NewClient(cfg, embedder, llm, llm.(domain.MetadataExtractor))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 4. Use it!
    // Ingest a document
    client.IngestFile(ctx, "manual.pdf", nil)

    // Query with Hybrid Search (Vector + Graph)
    resp, _ := client.Query(ctx, "How do I reset the system?", nil)
    log.Println(resp.Answer)
}
```

## 🧠 Core Capabilities

### 1. Hybrid Ingestion & GraphRAG

RAGO doesn't just chunk text; it builds a **Knowledge Graph** automatically.

```go
opts := &rag.IngestOptions{
    ChunkSize: 500,
    Overlap:   50,
    // Enable automatic entity extraction (GraphRAG)
    EnhancedExtraction: true,
}

// Ingest user-uploaded file
resp, err := client.IngestFile(ctx, userFilePath, opts)
fmt.Printf("Ingested doc %s with %d chunks\n", resp.DocumentID, resp.ChunkCount)
```

### 2. Stateful Chat with Memory

Build chatbots that remember context. RAGO handles the complexity of **Semantic Recall**.

```go
// 1. Start a persistent session for a user
session, _ := client.StartChat(ctx, "user-123", map[string]interface{}{
    "tier": "pro",
})

// 2. Chat loop
// RAGO automatically:
// - Embeds the query
// - Searches past history (Long-term Memory)
// - Retrieves relevant documents (RAG)
// - Generates response
// - Saves the interaction
resp, err := client.Chat(ctx, session.ID, "My server is down", &rag.QueryOptions{
    Temperature: 0.3,
})

fmt.Println("AI:", resp.Answer)
```

### 3. Agentic Tools (MCP)

Give your AI hands to perform actions using the Model Context Protocol.

```go
// Enable tools in query options
opts := &rag.QueryOptions{
    ToolsEnabled: true,
    // Optional: Restrict specific tools
    // AllowedTools: []string{"filesystem", "fetch"},
}

// User asks to perform an action
// If configured with MCP servers, RAGO will execute the tool
resp, err := client.Query(ctx, "Check the status of order #999 from the database", opts)

// Check what tools were used
if len(resp.ToolsUsed) > 0 {
    log.Printf("AI used tools: %v", resp.ToolsUsed)
}
```

## 🏗️ Architecture for Integrators

RAGO is designed to be the **Knowledge Layer** of your application.

```mermaid
[Your App / Web Server]
       │
       ▼
[ RAGO Library ] ◄─── [ Your Config ]
       │
       ├─ [ LLM Provider ] ──► (OpenAI / Ollama / Azure)
       │
       └─ [ Storage Engine ] ──► (Local SQLite File)
```

### Best Practices

1.  **Singleton Client**: Create one `rag.Client` at startup and reuse it. It is thread-safe and manages connection pools efficiently.
2.  **Context Management**: Always pass `context.Context` to handle timeouts and cancellations gracefully.
3.  **Error Handling**: RAGO returns typed errors (in `pkg/domain`) allowing you to distinguish between "not found" vs "provider error".

## 📚 API Reference

See the full [GoDoc](https://pkg.go.dev/github.com/liliang-cn/rago/v2) for detailed API documentation.

```
