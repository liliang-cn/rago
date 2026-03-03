---
name: rago
description: Use Rago for local RAG (Retrieval Augmented Generation) operations including document ingestion, semantic search, and Q&A. Supports multi-provider LLM, MCP tools, Skills, and autonomous LongRun mode. Use when building local knowledge bases, semantic search, or AI assistants with memory.
license: MIT
metadata:
  author: liliang-cn
  version: "2.47.0"
  github: https://github.com/liliang-cn/rago
---

# Rago - Local RAG System with Agent Automation

Rago is primarily a **local RAG system** with optional agent automation. Core features: document ingestion → chunking → vector storage → semantic search → Q&A.

## When to Use

- Build local knowledge base from documents
- Semantic search over document corpus
- Q&A systems backed by your own data
- AI agents with MCP tools and Skills
- Multi-provider LLM (OpenAI, Anthropic, Ollama)

## Quick Start

### As a Library

```go
package main

import (
    "context"
    "fmt"
    "github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
    ctx := context.Background()

    svc, _ := agent.New("my-agent").
        WithPrompt("You are a helpful assistant.").
        WithRAG().
        WithMCP().
        WithSkills().
        WithMemory().
        WithDebug(false).
        Build()
    defer svc.Close()

    // Simple one-shot Q&A
    reply, _ := svc.Ask(ctx, "Summarize my documents")
    fmt.Println(reply)

    // Full agentic run (tool calls, RAG, planning)
    result, _ := svc.Run(ctx, "Analyze and summarize")
    fmt.Println(result.Text())  // or result.Err() to check errors
}
```

### CLI Usage

```bash
# Ingest documents
rago ingest ./documents/ --collection my-docs

# Query
rago query "What is the main topic?" --collection my-docs

# Run agent
rago agent run "Analyze and summarize"
```

## Core Features

### 1. RAG Operations

```go
// Ingest
ragProcessor.Ingest(ctx, "./docs/", domain.IngestOptions{
    Collection: "my-docs",
    ChunkSize:  512,
})

// Query
result, _ := ragProcessor.Query(ctx, domain.QueryRequest{
    Query:   "search query",
    TopK:    5,
})
```

### 2. Multi-Provider LLM

OpenAI, Anthropic, Ollama, and custom providers via `~/.rago/config.yaml`.

### 3. MCP Tools

```go
svc, _ := agent.New("agent").
    WithMCP(agent.WithMCPConfigPaths("mcpServers.json")).
    Build()
```

### 4. Skills Integration

```go
svc, _ := agent.New("agent").
    WithSkills(agent.WithSkillsPaths("~/.agents/skills")).
    Build()
```

### 5. LongRun (Autonomous Mode)

```go
longRun, _ := agent.NewLongRun(svc).
    WithInterval(10 * time.Minute).
    WithMaxActions(5).
    Build()

longRun.AddTask(ctx, "Analyze documents", nil)
longRun.Start(ctx)
```

## Architecture

```
┌──────────────────────────────────────────┐
│              Rago System                 │
├──────────────────────────────────────────┤
│  RAG Store │ LLM Pool │ MCP Tools        │
│       ┌────┴──────────┴────┐             │
│       │    Agent Service   │             │
│       │  ToolRegistry      │ ← Module    │
│       │  HookRegistry      │   interface │
│       │  MemoryService     │             │
│       └────────────────────┘             │
└──────────────────────────────────────────┘
```

Tools are self-registered via the `Module` interface (`RAGModule`, `MemoryModule`). Each `Service` has its own isolated `HookRegistry`.

## Builder Pattern

```go
// Basic
svc, _ := agent.New("name").Build()

// Full features
svc, _ := agent.New("name").
    WithPrompt("You are a helpful assistant."). // system prompt
    WithRAG().
    WithMCP().
    WithSkills().
    WithMemory().
    WithDBPath("~/.rago/data/agent.db").
    WithDebug(true).
    WithTool(myDef, myHandler, "category").     // single tool
    WithTools(agent.ToolRegistration{...}).     // bulk tools
    WithProgress(func(e *agent.ProgressEvent) { fmt.Println(e) }).
    Build()
```

## Invocation API

```go
// Simple Q&A — returns (string, error)
reply, err := svc.Ask(ctx, "What is Go?")

// Multi-turn chat — session memory, returns (string, error)
reply, err := svc.Chat(ctx, "Tell me more")

// Full agent run — tool calls, RAG, planning
result, err := svc.Run(ctx, "goal", agent.WithMaxTurns(20))
fmt.Println(result.Text())      // final answer
fmt.Println(result.Err())       // first tool/LLM error
fmt.Println(result.HasSources()) // true if RAG sources attached

// Streaming — live token output
for token := range svc.Stream(ctx, "Write a poem") {
    fmt.Print(token)
}

// Streaming chat — session memory + live tokens
for token := range svc.ChatStream(ctx, "Continue the story") {
    fmt.Print(token)
}

// Full event stream — tool calls, sources, errors
events, err := svc.RunStream(ctx, "Complex task")
for e := range events {
    fmt.Println(e.Type, e.Content)
}
```

## Run Options

```go
result, _ := svc.Run(ctx, "goal",
    agent.WithMaxTurns(20),
    agent.WithTemperature(0.7),
    agent.WithStoreHistory(true),
)
```

## See Also

- [API Reference](references/API.md) - Detailed API
- [Architecture](references/ARCHITECTURE.md) - Design decisions
- [Configuration](references/CONFIG.md) - Config options
