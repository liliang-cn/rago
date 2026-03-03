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
| **🛠️ Tools** | Native support for **MCP (Model Context Protocol)**, **[Claude-compatible Skills](SKILLS.md)**, **PTC (Programmatic Tool Calling)**, and **WebSocket Realtime Sessions**. |
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

    svc, _ := agent.New("my-assistant").
        WithPrompt("You are a helpful assistant.").
        WithMCP().
        WithMemory().
        Build()
    defer svc.Close()

    // Simple Q&A — returns (string, error)
    reply, _ := svc.Ask(ctx, "What is Go?")
    fmt.Println(reply)

    // Full agentic run — planning, tool calls, RAG
    res, _ := svc.Run(ctx, "Research Go 1.24 features and save a summary to memory.")
    fmt.Println(res.Text())
}
```

---

## 🏗️ Architecture

### Layered Design

RAGO uses a **trimmable layered architecture** - use only what you need:

```
┌─────────────────────────────────────────────────────────────────┐
│                        Agent Layer                               │
│  (Planning, Execution, Handoffs, SubAgents, PTC)                │
├─────────────────────────────────────────────────────────────────┤
│                     Action Layer (Optional)                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  MCP Tools  │  │   Skills    │  │   Custom Tools          │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                     RAG Layer (Optional)                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Chunker   │  │   Embedder  │  │  Vector Store (SQLite)  │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                        LLM Layer                                 │
│  ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌─────────┐ ┌──────────┐  │
│  │ OpenAI  │ │ Ollama  │ │ DeepSeek │ │ Claude  │ │ Azure    │  │
│  └─────────┘ └─────────┘ └──────────┘ └─────────┘ └──────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Supported LLM Providers

| Provider | Type | Models |
|----------|------|--------|
| OpenAI | Cloud | GPT-4o, GPT-4-turbo, GPT-3.5-turbo |
| Azure OpenAI | Cloud | GPT-4, GPT-35-turbo |
| Ollama | Local | Llama 3, Mistral, Qwen, etc. |
| DeepSeek | Cloud | DeepSeek-V3, DeepSeek-Coder |
| Anthropic | Cloud | Claude 3.5 Sonnet, Claude 3 Opus |

---

## 🚀 Key Features

### 1. Multi-Agent Orchestration

**Handoffs**: Transfer control between specialized agents
```go
// Register specialized agents
svc.RegisterAgent(researchAgent)
svc.RegisterAgent(writerAgent)

// Agents can hand off tasks to each other
// The orchestrator routes to the best agent for each task
```

**SubAgents**: Delegate focused tasks with restricted tool access
```go
// Create a SubAgent with limited tools
sub := agent.NewSubAgent("data-collector", parentAgent).
    WithTools(allowlist).
    WithMaxTurns(5).
    Build()

// Delegate task execution
result := sub.Run(ctx, "Collect quarterly sales data")
```

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

### 6. Programmatic Tool Calling (PTC)

PTC allows the LLM to write **JavaScript code** that orchestrates multiple tool calls in a single execution, instead of requiring round-trips through the model for each tool invocation. This substantially reduces latency and token consumption.

**Benefits:**
- **Reduced Latency**: Execute multiple tools in one shot
- **Lower Token Usage**: Process large results before they hit the context window
- **Complex Logic**: Write code to filter, aggregate, and transform data

```go
// Enable PTC when creating an agent
svc, _ := agent.New("data-analyst").
    WithPTC().
    Build()

// The LLM can now respond with code like:
// <code>
// const team = callTool('get_team_members', { department: 'engineering' });
// const results = team.members.map(m => {
//     const expenses = callTool('get_expenses', { member_id: m.id });
//     return { name: m.name, total: expenses.total };
// });
// return results;
// </code>
```

See the [PTC examples](./examples/ptc/) for complete demos.

---

## 🎯 Invocation API

RAGO provides multiple invocation styles to match your use case:

```go
// Ask — one-shot Q&A, returns (string, error)
reply, err := svc.Ask(ctx, "What is the capital of France?")

// Chat — multi-turn with session memory, returns (string, error)
reply, err := svc.Chat(ctx, "Tell me more about it")

// Run — full agentic loop (tool calls, RAG, planning)
result, err := svc.Run(ctx, "goal", agent.WithMaxTurns(20))
fmt.Println(result.Text())       // final answer
fmt.Println(result.HasSources()) // true if RAG sources attached

// Stream — live token output, one-shot
for token := range svc.Stream(ctx, "Write a poem") {
    fmt.Print(token)
}

// ChatStream — multi-turn + live tokens
for token := range svc.ChatStream(ctx, "Continue the story") {
    fmt.Print(token)
}

// RunStream — full event stream (tool calls, sources, errors)
events, err := svc.RunStream(ctx, "Complex multi-step task")
for e := range events {
    fmt.Println(e.Type, e.Content)
}
```

---

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
*   **[quickstart](./examples/quickstart/)**: Simplest way to get started.
*   **[agent_usage](./examples/agent/agent_usage/)**: Complete agent usage patterns.
*   **[realtime_chat](./examples/agent/realtime_chat/)**: WebSocket realtime demo.
*   **[multi_agent_orchestration](./examples/agent/multi_agent_orchestration/)**: Comprehensive demo with Handoffs and streaming.
*   **[subagent](./examples/subagent/)**: SubAgent patterns for parallel execution.
*   **[ptc](./examples/ptc/)**: Programmatic Tool Calling examples.

## 📄 License
MIT License - Copyright (c) 2024-2026 RAGO Authors.
