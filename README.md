# AgentGo

**Local-first RAG + Agent framework for Go.**

> “AgentGo? It's useless and it consumes a lot of tokens.” -- some guy on the internet

[中文文档](README_zh-CN.md) · [API Reference](references/API.md) · [Architecture](references/ARCHITECTURE.md)

AgentGo is a Go library for building AI applications that keep your data local. Start with semantic search over your documents, add agent automation when you need it.

```bash
go get github.com/liliang-cn/agent-go
```

---

## What AgentGo does

| Capability    | Details                                                                                                    |
| ------------- | ---------------------------------------------------------------------------------------------------------- |
| **RAG**       | Ingest docs → chunk → embed → SQLite vector store → hybrid search                                          |
| **Agent**     | Multi-turn reasoning loop with tool calls, planning, and session memory                                    |
| **Memory**    | **Cognitive Layer**: Hindsight-style evolution (Fact → Observation) + PageIndex-style reasoning navigation |
| **Tools**     | MCP (Model Context Protocol), Skills (YAML+Markdown), custom inline tools                                  |
| **PTC**       | LLM writes JavaScript; tools run in a Goja sandbox — cuts round-trips                                      |
| **Streaming** | Token-by-token via channel; full event stream with tool call visibility                                    |
| **Providers** | OpenAI, Anthropic, Azure, DeepSeek, Ollama — switchable at runtime                                         |

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
    WithDBPath("~/.agentgo/data/agent.db").
    Build()
defer svc.Close()

// Ingest once
svc.Run(ctx, "Ingest ./docs/")

// Query
reply, _ := svc.Ask(ctx, "What does the spec say about error handling?")
```

### Multi-turn chat with cognitive memory

```go
svc, _ := agent.New("assistant").
    WithMemory().
    Build()
defer svc.Close()

svc.Chat(ctx, "My name is Alice and I work on the Go team.")
reply, _ := svc.Chat(ctx, "What team am I on?")
// → "You're on the Go team, Alice." (Recall via hybrid vector/index search)
```

### CLI Interface

Run the interactive chat with memory visibility:

```bash
# Start interactive chat showing retrieved memories and reasoning
go run ./cmd/agentgo-cli chat --show-memory

# Enable JavaScript sandbox for complex logic
go run ./cmd/agentgo-cli chat --with-ptc
```

Run squad workflows from the CLI:

```bash
# Create a standalone agent
agentgo agent add Scout --description "Independent field agent" \
  --instructions "Work independently, answer directly, and use tools when needed."

# Inspect or update that agent
agentgo agent show Scout
agentgo agent update Scout --model openai/gpt-5-mini

# Run a stored agent directly
agentgo agent run --agent Scout "Summarize the current repo structure"

# Create a squad
agentgo squad add "Docs Squad" --description "Documentation and release notes"

# Join the standalone agent to a squad
agentgo agent join Scout --squad "Docs Squad" --role specialist

# Run a task through the default captain and a specialist
agentgo squad go "@Captain @Scout summarize the UI/backend relationship and write workspace/ui_backend_overview.md"

# Inspect runtime task state; follows while tasks are still running or queued
agentgo squad status "Docs Squad"

# Leave the squad again
agentgo agent leave Scout

# Delete the squad when you're done
agentgo squad delete "Docs Squad"
```

---

## Cognitive Memory (Hindsight & PageIndex)

AgentGo implements an evolving memory layer inspired by **Hindsight** (Cognitive Hierarchy) and **PageIndex** (Structural Navigation).

| Concept                | Description                                                                                                     |
| ---------------------- | --------------------------------------------------------------------------------------------------------------- |
| **Facts**              | Raw atomic data points extracted from conversations (e.g., "User likes Go").                                    |
| **Observations**       | LLM-consolidated insights synthesized from multiple facts via **Reflect()**.                                    |
| **Hierarchical Index** | A `_index/` directory with Markdown summaries for lightning-fast reasoning navigation.                          |
| **Hybrid Search**      | Parallel **Vector Search** (similarity) + **Index Navigator** (reasoning) fused via RRF.                        |
| **Traceability**       | Every observation tracks its **EvidenceIDs**, providing a clear audit trail of why the agent "knows" something. |

### Memory Evolution

1. **Extraction**: Agent identifies a fact during chat.
2. **Indexing**: Fact is stored in a Markdown file with YAML metadata (Confidence, SourceType).
3. **Reflection**: Periodically (e.g., every 5 facts), a background worker triggers `Reflect()` to merge facts into high-level Observations.
4. **Superseded**: When information changes, old memories are marked as `stale` and linked to new ones via `SupersededBy`.

---

## Builder

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

| Method                    | Returns                     | Session         | Use case              |
| ------------------------- | --------------------------- | --------------- | --------------------- |
| `Ask(ctx, prompt)`        | `(string, error)`           | no              | one-shot Q&A          |
| `Chat(ctx, prompt)`       | `(*ExecutionResult, error)` | yes (auto UUID) | conversational        |
| `Run(ctx, goal, ...opts)` | `(*ExecutionResult, error)` | optional        | full agent loop       |
| `Stream(ctx, prompt)`     | `<-chan string`             | no              | live token output     |
| `ChatStream(ctx, prompt)` | `<-chan string`             | yes             | conversational + live |
| `RunStream(ctx, goal)`    | `(<-chan *Event, error)`    | optional        | full event visibility |

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

Memory and cache are different subsystems:

| Subsystem  | Storage                                 | What for                                                           |
| ---------- | --------------------------------------- | ------------------------------------------------------------------ |
| **Memory** | Markdown/YAML files or SQLite + vectors | Durable facts, observations, preferences, and reasoning context    |
| **Cache**  | In-memory or file-backed JSON entries   | Disposable acceleration artifacts for query/vector/LLM/chunk reuse |

```go
// Enable cognitive memory
svc, _ := agent.New("agent").WithMemory().Build()

// LongRun agents share the same memory automatically
lr, _ := agent.NewLongRun(svc).
    WithInterval(5 * time.Minute).
    WithWorkDir("~/.agentgo/longrun").
    Build()
```

Memory degrades gracefully:

- no embedder -> file memory still works
- file-backed memory uses Markdown + YAML frontmatter and PageIndex-style retrieval
- `remember:` prompts can be written directly to memory
- ordinary dialogue can also be extracted into memory via `StoreIfWorthwhile`

Cache is separate from memory:

- use `agentgo cache status|put|get|delete|clear`
- configure `cache.store_type = "memory"` or `cache.store_type = "file"`

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

## Squad API

AgentGo exposes a squad-oriented manager API for standalone agents and squad agents. A `captain` is just an agent role inside a squad.

```go
store, err := agent.NewStore(filepath.Join(cfg.DataDir(), "agent.db"))
if err != nil {
    panic(err)
}

manager := agent.NewSquadManager(store)
if err := manager.SeedDefaultMembers(); err != nil {
    panic(err)
}

scout, err := manager.CreateAgent(ctx, &agent.AgentModel{
    Name:         "Scout",
    Kind:         agent.AgentKindAgent,
    Description:  "Independent field agent",
    Instructions: "Work independently and answer directly.",
})
if err != nil {
    panic(err)
}

docsSquad, err := manager.CreateSquad(ctx, &agent.Squad{
    Name:        "Docs Squad",
    Description: "Documentation and release notes",
})
if err != nil {
    panic(err)
}

writer, err := manager.JoinSquad(ctx, scout.Name, docsSquad.ID, agent.AgentKindSpecialist)
if err != nil {
    panic(err)
}

result, err := manager.DispatchTask(ctx, writer.Name, "Write workspace/ui_backend_overview.md")
if err != nil {
    panic(err)
}
fmt.Println(result)
```

Useful squad-manager entry points:

- `CreateAgent`, `UpdateAgent`, `DeleteAgent`, `GetAgentByName`, `ListAgents`, `ListStandaloneAgents`
- `JoinSquad`, `LeaveSquad`, `GetAgentService`
- `CreateSquad`, `ListSquads`, `GetSquadByName`
- `AddSquadAgent`, `CreateSquadAgent`, `ListSquadAgents`, `GetSquadAgentByName`
- `AddCaptain`, `AddSpecialist`, `ListCaptains`, `ListSpecialists` (role-specific helpers)
- `DispatchTask`, `DispatchTaskStream`
- `EnqueueSharedTask`, `ListSharedTasks`

---

## Planning (deterministic workflow)

```go
plan, _   := svc.Plan(ctx, "Deploy the new service")
// inspect plan.Steps, edit if needed
result, _ := svc.Execute(ctx, plan.ID)
```

---

## Configuration & Storage

Config file: `agentgo.toml` (auto-discovered in `./` → `~/.agentgo/` → `~/.agentgo/config/`).

### Directory layout (default `home = ~/.agentgo`)

```
~/.agentgo/
├── agentgo.toml              ← config file
├── mcpServers.json        ← MCP server definitions
├── data/
│   ├── agentgo.db            ← RAG vector store (sqlite-vec); also Memory vector store
│   ├── agent.db           ← Agent sessions + execution plans
│   └── memories/          ← Memory file store (Markdown + YAML frontmatter)
├── skills/                ← SKILL.md files
├── intents/               ← Intent YAML files
└── workspace/             ← Agent working directory
```

### SQLite files

| File                 | Default path              | Purpose                                                                                         |
| -------------------- | ------------------------- | ----------------------------------------------------------------------------------------------- |
| `agentgo.db`         | `$home/data/agentgo.db`   | RAG documents + vector index; shared as Memory vector store when `memory.store_type = "vector"` |
| `agent.db`           | `$home/data/agent.db`     | Agent sessions and plan state                                                                   |
| `history.db` _(opt)_ | via `WithHistoryDBPath()` | Detailed tool-call logs — only created when `WithStoreHistory(true)`                            |

### Memory store types

| `store_type`       | Storage                                                        | Requires embedder |
| ------------------ | -------------------------------------------------------------- | ----------------- |
| `file` _(default)_ | `data/memories/entities/*.md` and `data/memories/streams/*.md` | No                |
| `vector`           | `data/agentgo.db` (shared)                                     | Yes               |
| `hybrid`           | file primary + `agentgo.db` shadow index                       | Yes               |

### Cache store types

| `store_type`         | Storage                         | Purpose                            |
| -------------------- | ------------------------------- | ---------------------------------- |
| `memory` _(default)_ | in-process memory               | Fast ephemeral cache               |
| `file`               | `data/cache/<namespace>/*.json` | Restart-friendly cache persistence |

### Key config fields

```toml
home = "~/.agentgo"             # base for all relative paths

[memory]
store_type  = "file"         # file | vector | hybrid

[cache]
store_type = "memory"        # memory | file
max_size   = 1000
query_ttl  = "15m"
vector_ttl = "24h"
llm_ttl    = "1h"
chunk_ttl  = "24h"

[rag.chunker]
chunk_size = 512
overlap    = 64
method     = "sentence"

[skills]
enabled  = true
auto_load = true

[mcp]
servers = ["~/.agentgo/mcpServers.json"]
```

AgentGo derives the runtime storage layout automatically from `home`:

- workspace: `$home/workspace`
- MCP filesystem allowlist: `$home/workspace`
- RAG database: `$home/data/agentgo.db`
- memory store: `$home/data/memories` or `$home/data/agentgo.db` when `memory.store_type = "vector"`
- cache directory: `$home/data/cache`

### Cache CLI

```bash
agentgo cache status
agentgo cache put query my-key my-value --ttl 5m
agentgo cache get query my-key
agentgo cache clear query
```

See [`references/CONFIG.md`](references/CONFIG.md) for the full annotated config.

---

## Providers

Configure in `agentgo.toml` (auto-discovered in `./`, `~/.agentgo/`, `~/.agentgo/config/`):

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

MIT — Copyright (c) 2024–2026 AgentGo Authors
