# RAGO

**Radical Agentic Go** — A modular, local-first RAG and Agent framework built for high-performance, transparent AI workflows.

[![Go Report Card](https://goreportcard.com/badge/github.com/liliang-cn/rago)](https://goreportcard.com/report/github.com/liliang-cn/rago)
[![Go Reference](https://pkg.go.dev/badge/github.com/liliang-cn/rago/v2.svg)](https://pkg.go.dev/github.com/liliang-cn/rago/v2)

## Why RAGO?

RAGO is designed with a **Layered Architecture** where everything is optional except the LLM. It prioritizes human-readability and local control.

### 🚀 Key Features

- **Layered Logic**: `LLM (Base)` → `RAG (Optional)` → `Skills/MCP (Optional)` → `Agent (Interface)`. Use only what you need.
- **Transparent Memory**: Persistent long-term memory stored as human-readable **Markdown files**. Zero-Embedding search support via automated Memory Maps.
- **Realtime Native**: Built-in support for WebSockets (OpenAI Responses API) for sub-second latency and stateful sessions.
- **Human-in-the-loop**: Explicit **Plan -> Review -> Execute** workflow. No more black-box AI actions.
- **Local-First**: Powered by SQLite (`sqvect`) and local file systems. Your data stays on your machine.

---

## Quick Start

### 1. Simple Agent with Memory
```go
svc, _ := agent.New(&agent.AgentConfig{
    Name:            "Alice",
    EnableMemory:    true,
    MemoryStoreType: "file", // Transparent Markdown memory
})

// Run a task
res, _ := svc.Run(ctx, "Research RAGO architecture and save to memory.")
fmt.Println(res.FinalResult)
```

### 2. Planning Workflow
```go
// 1. Generate Plan
plan, _ := svc.Plan(ctx, "Build a CLI tool in Go")

// 2. Human reviews the steps in CLI/UI...

// 3. Execute
result, _ := svc.Execute(ctx, plan.ID)
```

## Installation

```bash
go get github.com/liliang-cn/rago/v2
```

## Architecture

1.  **LLM Layer**: Unified interface for OpenAI, Ollama, and more.
2.  **Knowledge (RAG)**: High-performance vector + graph retrieval via `sqvect`.
3.  **Action (MCP/Skills)**: Extensible tool calling via Model Context Protocol.
4.  **Agent**: The brain. Handles intent, orchestration, and persistence.

---
License: MIT
