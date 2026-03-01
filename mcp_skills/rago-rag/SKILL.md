---
name: rago-rag
description: Use Rago for local RAG (Retrieval Augmented Generation) operations including document ingestion, semantic search, and Q&A. Supports multi-provider LLM, MCP tools, Skills, and autonomous LongRun mode. Use when building local knowledge bases, semantic search, or AI assistants with memory.
license: MIT
metadata:
  author: liliang-cn
  version: "2.46.0"
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
    "github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
    ctx := context.Background()

    // Create agent with RAG, MCP, and Skills
    svc, _ := agent.New("my-agent").
        WithRAG().
        WithMCP().
        WithSkills().
        WithMemory().
        Build()
    defer svc.Close()

    // Run a query
    result, _ := svc.Run(ctx, "Summarize my documents")
    println(result.FinalResult)
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
┌─────────────────────────────────────┐
│           Rago System               │
├─────────────────────────────────────┤
│  RAG Store │ LLM Pool │ MCP Tools   │
│       ┌────┴──────────┴────┐        │
│       │    Agent Service   │        │
│       │    Skills Service  │        │
│       └────────────────────┘        │
└─────────────────────────────────────┘
```

## Builder Pattern

```go
// Basic
svc, _ := agent.New("name").Build()

// Full features
svc, _ := agent.New("name").
    WithRAG().
    WithMCP().
    WithSkills().
    WithMemory().
    WithDBPath("~/.rago/data/agent.db").
    WithDebug(true).
    Build()
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
- [Configuration](references/CONFIG.md) - Config options
