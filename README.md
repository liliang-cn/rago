# RAGO: Unified AI Agent Framework for Go

[中文文档](README_zh-CN.md)

RAGO is a production-grade **AI Agent framework** built for Go developers. It provides a complete runtime environment integrating **Hybrid RAG (Vector + Graph)**, **Multi-Agent Orchestration**, **MCP Tool Protocol**, and a **Transparent File-based Memory System**.

It enables developers to build anything from simple chatbots to complex, autonomous agents with long-term memory and tool-using capabilities.

## 🌟 Why RAGO?

RAGO solves the core pain points of building complex AI applications with a **Layered Architecture** where everything is controllable.

| Core Pillars | Key Capabilities |
| :--- | :--- |
| **🧠 Reasoning** | **Layered Design**: `LLM (Base)` → `RAG (Optional)` → `Skills/MCP (Optional)` → `Agent (Interface)`. |
| **📚 Knowledge** | **Hybrid RAG**: Fast vector search combined with **SQLite-based GraphRAG**. Supports massive **Batch Embedding**. |
| **🛠️ Tools** | Native support for **MCP (Model Context Protocol)**, **[Claude-compatible Skills](SKILLS.md)**, and **WebSocket Realtime Sessions**. |
| **💾 Memory** | **Hybrid Storage**: High-performance SQLite or **Transparent Markdown files**. Features self-reflection and Smart Fusion. |
| **⚡ Runtime** | **Deterministic Workflow**: Unique **Plan -> Review -> Execute** loop to eliminate black-box AI behavior. |
| **🔒 Local-First** | Run entirely offline (**Ollama**) or connect to the cloud. Your data is physically isolated and protected. |

---

## 📦 Installation

```bash
go get github.com/liliang-cn/rago/v2
```

## 🚀 Quick Start: Hello World Agent

Create an agent that can plan, think, and execute tasks.

```go
package main

import (
    "context"
    "fmt"
    "github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
    ctx := context.Background()

    // 1. Initialize service (Runtime Environment)
    svc, _ := agent.New(&agent.AgentConfig{
        Name:            "my-assistant",
        EnableMCP:       true, 
        EnableMemory:    true,
        MemoryStoreType: "file", // Use transparent Markdown-based memory
    })
    defer svc.Close()

    // 2. Run a task
    res, _ := svc.Run(ctx, "Research Go 1.24 features and save a summary to memory.")
    fmt.Println(res.FinalResult)
}
```

---

## 🏗️ Architecture & Features

### 1. Layered Logic & Optional Components
RAGO uses strict hierarchical dependencies, allowing you to "trim" the system as needed:
*   **LLM Layer**: Unified wrapper for OpenAI, Ollama, DeepSeek, etc.
*   **RAG Layer (Optional)**: Injects domain knowledge.
*   **Action Layer (Optional)**: Extends capabilities via MCP/Skills.
*   **Agent Layer**: The brain handling intent and orchestration.

### 2. Transparent Memory & Smart Fusion
Memory is no longer a black box. RAGO stores long-term facts as human-readable **Markdown + YAML** files.
*   **Zero-Embedding Routing**: Automated "Memory Maps" for precise fact-finding without embedding costs.
*   **Smart Fusion**: Agents automatically merge new insights into existing files, ensuring knowledge evolves continuously.

### 3. Native Realtime Interaction (WebSocket)
Built-in support for WebSockets via OpenAI's latest Responses API.
*   **Sub-second Latency**: Persistent connections for low-overhead multi-turn tool calls.
*   **Stateful Sessions**: Server-side context management to save bandwidth.

### 4. Hybrid RAG (Vector + Knowledge Graph)
*   **Vector Search**: For semantic similarity.
*   **GraphRAG**: Discover implicit relationships between entities.
*   **Batching**: High-concurrency embedding injection for massive datasets.

### 5. Deterministic Planning (Plan-Review-Execute)
```go
// 1. Generate Plan (Readable steps)
plan, _ := svc.Plan(ctx, "Deploy a new web service")

// 2. Human reviews the steps in CLI or UI...

// 3. Execute after confirmation
result, _ := svc.Execute(ctx, plan.ID)
```

---

## 💻 CLI Usage

```bash
# Run a task
rago agent run "Clean up duplicate files in current directory"

# Manage RAG knowledge base
rago rag ingest ./docs/ --recursive
rago rag query "How to configure server ports?"
```

## ⚙️ Configuration

RAGO looks for `rago.toml` in `./`, `~/.rago/`, or `~/.rago/config/`.

```toml
[server]
port = 7127

[llm_pool]
enabled = true
strategy = "round_robin"

[[llm_pool.providers]]
name = "openai"
provider = "openai"
api_key = "sk-..."
model = "gpt-4o"
```

## 📚 Examples

Check the `examples/` directory for deep dives:
*   **[04_file_memory_test](./examples/agent_usage/04_file_memory_test/)**: Full workflow for transparent memory.
*   **[realtime_chat](./examples/realtime_chat/)**: WebSocket realtime demo.
*   **[multi_agent_orchestration](./examples/multi_agent_orchestration/)**: Comprehensive demo with Handoffs and streaming.

## 📄 License
MIT License - Copyright (c) 2024-2026 RAGO Authors.
