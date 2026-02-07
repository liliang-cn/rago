# RAGO: The All-in-One AI Agent Framework for Go

[中文文档](README_zh-CN.md)

RAGO is a production-grade **AI Agent Framework** built natively for Go. It bridges the gap between static LLM calls and autonomous, stateful, and tool-using agents.

Unlike simple wrappers, RAGO provides a complete runtime environment with **Hybrid RAG (Vector + Graph)**, **Multi-Agent Orchestration**, **Model Context Protocol (MCP)** support, and **Self-Reflecting Memory**.

## 🌟 Why RAGO?

RAGO is designed for developers who want to build complex AI applications that run locally or in the cloud, without the bloat of Python dependencies.

| Core Pillar | Key Capabilities |
| :--- | :--- |
| **🧠 Reasoning Engine** | **Planner/Executor**, **Intent Recognition**, and **Multi-Agent Handoffs**. Capable of solving multi-step complex tasks. |
| **📚 Knowledge Engine** | **Hybrid RAG**: Combines high-speed Vector Search with **SQLite-based GraphRAG** for deep relationship discovery. |
| **🛠️ Tooling Engine** | Native support for **MCP (Model Context Protocol)**, **Claude-compatible Skills**, and **Dynamic Go Functions**. |
| **💾 Memory System** | **Hindsight Architecture**: Features Short-term context, Long-term factual memory, Entity tracking, and Reflection. |
| **⚡ Runtime** | **Event-Driven Loop**: Real-time streaming (token-by-token), state management, and async execution. |
| **🔒 Local-First** | Designed to work offline with **Ollama** or online with **OpenAI/DeepSeek**. Data stays in your control. |

---

## 📦 Installation

```bash
go get github.com/liliang-cn/rago/v2
```

## 🚀 Quick Start: The "Hello World" Agent

Create an agent that can plan, think, and execute.

```go
package main

import (
    "context"
    "fmt"
    "github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
    ctx := context.Background()

    // 1. Initialize the Service (The Runtime)
    svc, _ := agent.New(&agent.AgentConfig{
        Name:         "my-assistant",
        EnableMCP:    true, // Enable external tools
        EnableMemory: true, // Enable long-term memory
    })
    defer svc.Close()

    // 2. Run a task (Streamed for real-time feedback)
    events, _ := svc.RunStream(ctx, "Research the latest Go 1.24 features and summarize them.")

    // 3. Consume the Event Loop
    for evt := range events {
        switch evt.Type {
        case agent.EventTypeThinking:
            fmt.Println("🤖 Thinking...")
        case agent.EventTypeToolCall:
            fmt.Printf("🛠️  Calling Tool: %s\n", evt.ToolName)
        case agent.EventTypePartial:
            fmt.Print(evt.Content) // Real-time typewriter effect
        }
    }
}
```

---

## 🏗️ Architecture & Features

### 1. Hybrid RAG (Vector + Graph)
RAGO doesn't just store embeddings; it builds a **Knowledge Graph** automatically.

*   **Vector Search**: For semantic similarity.
*   **GraphRAG**: For "connecting the dots" between entities across documents.

```go
// Ingest a document with enhanced Graph extraction
client.IngestFile(ctx, "manual.pdf", &rag.IngestOptions{ EnhancedExtraction: true })

// Query using Hybrid Search
resp, _ := client.Query(ctx, "What are the relationships between Module A and Module B?", nil)
```

### 2. Multi-Agent Orchestration (Handoffs)
Build complex systems where specialized agents collaborate. A "Triage" agent can route tasks to a "Security Expert" or a "Writer".

```go
// Define a Specialist
mathAgent := agent.NewAgent("MathExpert")
mathAgent.SetInstructions("You solve complex math problems.")

// Define a Triage Agent with a Handoff
triageAgent := agent.NewAgent("Receptionist")
triageAgent.AddHandoff(agent.NewHandoff(mathAgent, 
    agent.WithHandoffToolDescription("Transfer calculation tasks to the math expert."),
))

// The runtime handles the switching automatically
svc.RegisterAgent(mathAgent)
svc.RegisterAgent(triageAgent)
```

### 3. Universal Tooling (MCP & Code)
RAGO unifies all tools under one interface.

*   **MCP Servers**: Connect to filesystem, GitHub, databases, or browsers via standard protocol.
*   **Go Functions**: Register your own code as tools dynamically.

```go
// 1. Add a standard MCP Server (e.g., Brave Search)
svc.AddMCPServer(ctx, "brave", "npx", []string{"-y", "@modelcontextprotocol/server-brave-search"})

// 2. Add a native Go Function
agent.AddTool("check_status", "Check system status", 
    map[string]interface{}{"type": "object", "properties": {...}},
    func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
        return "System All Green", nil
    },
)
```

### 4. Hindsight Memory (Reflection)
The memory system isn't just a log. It **reflects** on interactions to distill "Observations" and "Mental Models".

```go
// Configure the "Personality" of the memory bank
svc.ConfigureMemory(ctx, &domain.MemoryBankConfig{
    Mission:    "You are a strict auditor.",
    Skepticism: 5, // High doubt, requires verification
})

// Trigger a reflection cycle to consolidate insights
summary, _ := svc.ReflectMemory(ctx)
```

---

## 💻 CLI Usage

RAGO comes with a powerful CLI for managing the entire lifecycle.

```bash
# 1. Start a task (with streaming output)
rago agent run "Find distinct files in this folder and summarize them"

# 2. Manage RAG Knowledge Base
rago rag ingest ./docs/ --recursive
rago rag query "How do I configure the server?"

# 3. Manage MCP Tools
rago mcp list
rago mcp install filesystem
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
model = "gpt-4-turbo"

[mcp]
enabled = true
# MCP Servers are defined in mcpServers.json
```

## 📚 Examples

Check out the `examples/` directory for deep dives:

*   **[multi_agent_orchestration](./examples/multi_agent_orchestration/)**: Complete demo of Handoffs, Dynamic Tools, and Streaming.
*   **[advanced_rag](./examples/advanced_rag/)**: Building a knowledge base with metadata filters.
*   **[skills_integration](./examples/skills_integration/)**: Using Claude-compatible Markdown skills.

## 📄 License
MIT License - Copyright (c) 2024-2025 RAGO Authors.