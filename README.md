# RAGO

**Local-first RAG + Agent framework for Go.**

[中文文档](README_zh-CN.md) · [API Reference](references/API.md) · [Architecture](references/ARCHITECTURE.md)

RAGO is a Go library for building AI applications that keep your data local. Start with semantic search over your documents, add agent automation when you need it.

```bash
go get github.com/liliang-cn/rago/v2
```

---

## What RAGO does

| Capability | Details |
|---|---|
| **RAG** | Ingest docs → chunk → embed → SQLite vector store → hybrid search |
| **Agent** | Multi-turn reasoning loop with tool calls, planning, and session memory |
| **Memory** | SQLite-backed with semantic recall; degrades gracefully to list-based without an embedder |
| **Tools** | MCP (Model Context Protocol), Skills (YAML+Markdown), custom inline tools |
| **PTC** | LLM writes JavaScript; tools run in a Goja sandbox — cuts round-trips |
| **Streaming** | Token-by-token via channel; full event stream with tool call visibility |
| **Providers** | OpenAI, Anthropic, Azure, DeepSeek, Ollama — switchable at runtime |

---

## Quick Start

### Simple Q&A

```go
svc, _ := agent.New("assistant").
    WithPrompt("You are a helpful assistant.").
    Build()
defer svc.Close()

reply, _ := svc.Ask(ctx, "What is Go?")
fmt.Println(reply)
```

### With RAG (document knowledge base)

```go
svc, _ := agent.New("assistant").
    WithPrompt("Answer questions based on the provided documents.").
    WithRAG().
    WithDBPath("~/.rago/data/agent.db").
    Build()
defer svc.Close()

// Ingest once
svc.Run(ctx, "Ingest ./docs/")

// Query
reply, _ := svc.Ask(ctx, "What does the spec say about error handling?")
```

### Multi-turn chat with memory

```go
svc, _ := agent.New("assistant").
    WithMemory().
    Build()
defer svc.Close()

svc.Chat(ctx, "My name is Alice and I work on the payments team.")
reply, _ := svc.Chat(ctx, "What team am I on?")
// → "You're on the payments team, Alice."
```

### Streaming

```go
// Token stream
for token := range svc.Stream(ctx, "Explain Go interfaces") {
    fmt.Print(token)
}

// Full event stream (tool calls, sources, errors)
events, _ := svc.RunStream(ctx, "Search the docs and summarize")
for e := range events {
    switch e.Type {
    case agent.EventTypePartial:  fmt.Print(e.Content)
    case agent.EventTypeToolCall: fmt.Printf("[tool: %s]\n", e.ToolName)
    case agent.EventTypeComplete: fmt.Println("\ndone")
    }
}
```

---

## Builder

Every option is a method chain on `agent.New()`:

```go
svc, err := agent.New("my-agent").
    // Capabilities
    WithRAG().                                      // rag_query + rag_ingest tools
    WithMemory().                                   // memory_save + memory_recall tools
    WithMCP().                                      // MCP tool servers
    WithSkills(agent.WithSkillsPaths("./skills")).  // Skill files
    WithPTC().                                      // JS sandbox tool transport
    // Config
    WithPrompt("You are a helpful assistant.").
    WithDBPath("~/.rago/data/agent.db").
    WithDebug(true).
    // Custom tools
    WithTool(myDef, myHandler, "category").
    // Callbacks
    WithProgress(func(e *agent.ProgressEvent) { fmt.Println(e.Text) }).
    Build()
```

### Module system

Capabilities self-register their tools via the `Module` interface:

```go
// Implement your own module
type Module interface {
    ID() string
    RegisterTools(registry *ToolRegistry) error
}

svc, _ := agent.New("agent").
    WithModule(NewMyCustomModule()).
    Build()
```

---

## Invocation API

| Method | Returns | Session | Use case |
|---|---|---|---|
| `Ask(ctx, prompt)` | `(string, error)` | no | one-shot Q&A |
| `Chat(ctx, prompt)` | `(*ExecutionResult, error)` | yes (auto UUID) | conversational |
| `Run(ctx, goal, ...opts)` | `(*ExecutionResult, error)` | optional | full agent loop |
| `Stream(ctx, prompt)` | `<-chan string` | no | live token output |
| `ChatStream(ctx, prompt)` | `<-chan string` | yes | conversational + live |
| `RunStream(ctx, goal)` | `(<-chan *Event, error)` | optional | full event visibility |

```go
result, _ := svc.Run(ctx, "goal",
    agent.WithMaxTurns(20),
    agent.WithTemperature(0.7),
    agent.WithSessionID("my-session"),
    agent.WithStoreHistory(true),
)

result.Text()        // final answer as string
result.Err()         // non-nil if agent reported an error
result.HasSources()  // true when RAG chunks were used
```

---

## Programmatic Tool Calling (PTC)

With `WithPTC()`, the LLM generates JavaScript instead of JSON tool calls. The code runs in a Goja sandbox where `callTool()` is available:

```go
svc, _ := agent.New("analyst").
    WithPTC().
    WithTool(teamDef, teamHandler, "data").
    WithTool(expenseDef, expenseHandler, "data").
    Build()

// The LLM can now write:
//   const team = callTool("get_team", { dept: "eng" });
//   return team.members.map(m => ({
//     name: m.name,
//     spend: callTool("get_expenses", { id: m.id }).total
//   }));
```

**When to use PTC:** multiple dependent tool calls in one shot, data transformation before it hits the context window, conditional tool logic.

---

## Memory

Memory has two layers:

| Layer | Storage | What for |
|---|---|---|
| **DB Memory** | SQLite + vectors | Auto-learned facts, conversation history, semantic recall |
| **File Memory** | Markdown files | Human-editable persona: `SOUL.md`, `AGENTS.md`, `HEARTBEAT.md` |

```go
// Enable DB memory (auto-learns from every conversation)
svc, _ := agent.New("agent").WithMemory().Build()

// LongRun agents share the same DB memory automatically
lr, _ := agent.NewLongRun(svc).
    WithInterval(5 * time.Minute).
    WithWorkDir("~/.rago/longrun").
    Build()
```

Memory degrades gracefully: no embedder → falls back to recency-based list retrieval.

---

## Autonomous Agents (LongRun)

LongRun runs an agent on a schedule with a persistent task queue:

```go
lr, _ := agent.NewLongRun(svc).
    WithInterval(10 * time.Minute).
    WithMaxActions(5).
    Build()

lr.AddTask(ctx, "Monitor RSS feeds and summarize new entries", nil)
lr.Start(ctx)
// ...
lr.Stop()
```

Features: SQLite task queue, heartbeat file, cron-style scheduling, shared DB memory with the parent agent.

---

## Multi-Agent Orchestration

```go
// Handoffs — specialist agents
orchestrator.RegisterAgent(researchAgent)
orchestrator.RegisterAgent(writerAgent)
// The LLM routes to the right agent via transfer_to_* tool calls

// SubAgents — scoped delegation
coordinator := agent.NewSubAgentCoordinator()
resultChan  := coordinator.RunAsync(ctx, subAgent)
results     := coordinator.WaitAll(ctx)
```

---

## Planning (deterministic workflow)

```go
plan, _   := svc.Plan(ctx, "Deploy the new service")
// inspect plan.Steps, edit if needed
result, _ := svc.Execute(ctx, plan.ID)
```

---

## Providers

Configure in `rago.toml` (auto-discovered in `./`, `~/.rago/`, `~/.rago/config/`):

```toml
[[llm_pool.providers]]
name     = "openai"
provider = "openai"
api_key  = "sk-..."
model    = "gpt-4o"

[[llm_pool.providers]]
name     = "local"
provider = "ollama"
base_url = "http://localhost:11434"
model    = "qwen2.5:14b"
```

Supported: OpenAI · Anthropic · Azure OpenAI · DeepSeek · Ollama (local)

---

## Examples

```
examples/
├── quickstart/               — simplest possible agent
├── agent/
│   ├── agent_usage/          — builder patterns, tool registration
│   ├── multi_agent_orchestration/ — handoffs + streaming
│   ├── longrun/              — autonomous scheduled agent
│   └── realtime_chat/        — WebSocket session
├── rag/                      — document ingestion + Q&A
├── memory/
│   ├── chat_with_memory/     — DB memory + chat
│   └── smart_fusion/         — memory merging
├── ptc/
│   ├── custom_tools/         — JS sandbox tool orchestration
│   └── memory_chat/          — PTC + memory
├── skills/                   — Skill files
└── mcp/                      — MCP tool servers
```

---

## License

MIT — Copyright (c) 2024–2026 RAGO Authors

