# RAGO: Autonomous Agent & RAG Library for Go

[中文文档](README_zh-CN.md)

RAGO is an **AI Agent SDK** designed for Go developers. It enables you to build autonomous agents with "hands" (MCP Tools & Skills), "brains" (Planning & Reasoning), and "memory" (Vector RAG & GraphRAG).

## 🤖 Building Autonomous Agents

RAGO's agent system acts as the central brain, orchestrating all other components (LLM, RAG, MCP) to solve complex tasks dynamically.

### Zero-Config Agent
Create an intelligent agent that can reason, plan, and use tools with just a few lines:

```go
// Create agent service with full capabilities
svc, _ := agent.New(&agent.AgentConfig{
    Name:         "my-agent",
    EnableMCP:    true, // Give it "hands" (MCP Tools)
    EnableSkills: true, // Give it "expertise" (Claude Skills)
    EnableMemory: true, // Give it "experience" (Hindsight)
    EnableRouter: true, // Give it "intuition" (Intent Recognition)
})

// Run a goal - The agent will plan and execute steps automatically
result, _ := svc.Run(ctx, "Research the latest Go features and write a summary")
```

## 🧠 Core Capabilities

RAGO is organized into four powerful pillars that form the intelligence layer of your application.

### 1. Unified LLM Support
Single API for multiple providers (Ollama, OpenAI, DeepSeek, etc.) with built-in resilience and load balancing.

```go
// Initialize LLM Pool with multiple providers via agent config
svc, _ := agent.New(&agent.AgentConfig{
    Name: "my-agent",
    EnableRouter: true, // Auto-select best strategy
})
```

### 2. Hybrid RAG (Vector + Knowledge Graph)
Combines high-speed vector similarity with **GraphRAG** for deep relationship discovery and context retrieval.

```go
// Enable GraphRAG for complex entity extraction
opts := &rag.IngestOptions{ EnhancedExtraction: true }
client.IngestFile(ctx, "data.pdf", opts)

// Query using Hybrid Search (Vector + Graph)
resp, _ := client.Query(ctx, "Analyze relationships in the data", nil)
```

### 3. MCP & Claude-compatible Skills
Extend your agent with **Model Context Protocol** tools and **Claude-compatible Skills**.

```go
// Add expert capabilities via Markdown skills and MCP servers
svc, _ := agent.New(&agent.AgentConfig{
    EnableMCP:    true, // Connect to external tools
    EnableSkills: true, // Loads Claude skills from .skills/
})
```

### 4. Hindsight: Self-Verification & Reflection
Powered by the **Hindsight** system, the agent reflects on its own performance to ensure accuracy.

*   **Self-Correction**: Automatically detects and fixes errors via multi-turn verification loops.
*   **Smart Observations**: Extracts and stores only worthwhile insights into long-term memory.

## 🧠 Core Pillars

| Feature | Description |
| :--- | :--- |
| **Autonomous Agent** | Dynamic task decomposition (Planner) and multi-round tool execution (Executor). |
| **Intent Recognition** | High-speed semantic routing and LLM-based goal classification. |
| **Hindsight Memory** | Self-reflecting memory system that stores verified insights and corrects errors. |
| **Tool Integration** | Native **MCP (Model Context Protocol)** support and **Claude-compatible Skills**. |
| **Hybrid RAG** | Vector search + **Knowledge Graph (GraphRAG)** via SQLite. |
| **Smart Memory** | Long-term factual memory and semantic conversation recall. |
| **Local-First** | Runs entirely offline with Ollama/LM Studio or connects to OpenAI/DeepSeek. |

## 📦 Installation

```bash
go get github.com/liliang-cn/rago/v2
```

## 🏗️ Architecture for Integrators

RAGO is designed to be the **Intelligence Layer** of your application:

- **`pkg/agent`**: The core Agentic loop (Planner/Executor/Sessions).
- **`pkg/skills`**: Plugin system for vertical domain capabilities.
- **`pkg/mcp`**: Connector for standardized external tools.
- **`pkg/rag`**: Knowledge retrieval engine.

## 📊 CLI vs Library

RAGO provides a powerful CLI for management, but is optimized for library usage:
- **CLI**: `./rago agent run "Task"`
- **Library**: `agentSvc.Run(ctx, "Task")`

## 📚 Documentation & Examples

*   **[Quickstart Guide](./examples/quickstart/)**: Basic setup for Go apps.
*   **[Advanced RAG](./examples/advanced_rag/)**: Building the knowledge base with metadata and filters.
*   **[Agent Usage](./examples/agent_usage/)**: Creating autonomous agents with planning and execution.
*   **[Skills Integration](./examples/skills_integration/)**: Connecting custom tools and Claude skills.
*   **[Session Compaction](./examples/compact_session/)**: Managing long-term agent context.
*   **[Intent Routing](./examples/intent_routing/)**: Semantic routing and intent recognition.

## 📄 License
MIT License - Copyright (c) 2024-2025 RAGO Authors.