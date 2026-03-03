# Rago Architecture

## Overview

Rago is a local-first RAG + Agent system. Core capability priority:

```
RAG System → Multi-Provider LLM → MCP Tools → Agent Automation → HTTP API
```

## Layer Diagram

```
┌────────────────────────────────────────────────┐
│                  Agent (public API)             │
│  Ask() / Chat() / Stream() / Run() / Plan()    │
└──────────────────┬─────────────────────────────┘
                   │
┌──────────────────▼─────────────────────────────┐
│           ExecutionEngine (single path)         │
│  - Collect context (RAG / Memory / Router)      │
│  - Build system prompt                          │
│  - Call LLM (streaming / non-streaming)         │
│  - Dispatch tool calls                          │
│  PTC is a "tool call transport", not a branch   │
└──────────────────┬─────────────────────────────┘
                   │
┌──────────────────▼─────────────────────────────┐
│              ToolRegistry (unified)             │
│  Register(name, info, handler)                  │
│  List() → []ToolDefinition                      │
│  Call(name, args) → result                      │
│  Source tags: mcp / skill / rag / memory / custom│
└──────────────────┬──────────────────────────────┘
                   │
     ┌─────────────┼──────────────┬─────────────┐
     ▼             ▼              ▼             ▼
  LLMProvider  MemoryModule  RAGModule     MCPModule
  (interface)  (self-reg)    (self-reg)    (self-reg)
```

## Key Packages

| Package | Responsibility |
|---------|---------------|
| `pkg/agent` | Agent builder, execution engine, session management |
| `pkg/rag/processor` | Document ingestion, chunking, vector search |
| `pkg/providers` | Unified LLM + Embedder interfaces with provider pool |
| `pkg/ptc` | PTC JS sandbox (goja), router, tool call transport |
| `pkg/memory` | Memory store + search |
| `pkg/domain` | Core interfaces and types |

## Key Files in pkg/agent

| File | Responsibility |
|------|---------------|
| `service.go` | Core struct, `NewService`, `Run/RunStream/Plan`, `runWithConfig` |
| `builder.go` | Fluent builder, module assembly, PTC sync |
| `tool_registry.go` | Unified ToolRegistry (Register/Call/ListForLLM/SyncToPTCRouter) |
| `module.go` | Module interface + RAGModule + MemoryModule |
| `service_execution.go` | `executeWithLLM`, `executeToolCalls`, `finalizeExecution` |
| `service_session.go` | `Chat/Ask/Stream/ChatStream/RunStream` |
| `service_ptc.go` | `runPTCExecution`, `ChatWithPTC` (thin wrapper) |
| `service_prompt.go` | System prompt + enriched prompt builders |
| `service_intent.go` | Intent recognition, RAG routing |
| `service_config.go` | Config setters/getters |
| `service_mcp.go` | MCP tool adapter |
| `service_helpers.go` | Tool collection helpers |
| `runtime.go` | Event loop powering RunStream |
| `events.go` | Event types for streaming |
| `hooks.go` | HookRegistry (per-service, not global) |

## ToolRegistry

All tools — regardless of source — register through a single `ToolRegistry`:

```go
registry.Register(ToolRegistration{
    Name:     "my_tool",
    Category: CategoryCustom,  // CategoryRAG / CategoryMemory / CategoryMCP / CategorySkill
    Info:     domain.ToolInfo{...},
    Handler:  func(ctx, args) (interface{}, error) { ... },
})
```

`ListForLLM(ptcEnabled bool)`:
- `ptcEnabled=false` → returns all tool schemas for native function calling
- `ptcEnabled=true` → returns nil (tools hidden; JS sandbox accesses them via `callTool()`)

`SyncToPTCRouter(router)` copies all registry entries into the PTC goja sandbox so `callTool("name", args)` works in JS code.

## Module Interface

Each capability module self-registers its tools:

```go
type Module interface {
    RegisterTools(registry *ToolRegistry) error
}
```

Built-in modules:
- `RAGModule` — registers `rag_query` + `rag_ingest`
- `MemoryModule` — registers `memory_save` + `memory_recall`

Custom modules can be added via `builder.WithModule(m)`.

## PTC (Programmatic Tool Calling)

PTC is a **transport mode**, not a separate execution path:

```
runWithConfig()
  ├── if PTC enabled → runPTCExecution()   [LLM generates JS; goja runs it]
  └── else           → executeWithLLM()    [LLM generates tool calls; Go runs them]
```

Both paths share: context collection, memory hooks, session save, `ExecutionResult` construction.

`ChatWithPTC()` is a thin backward-compat wrapper over `Chat()`.

## Streaming

```
RunStream()                  ← full Event channel (tool calls, partial text, errors)
Stream() / ChatStream()      ← text-only <-chan string (filter EventTypePartial)
```

## Memory Systems

Rago has **two complementary memory layers**, both accessible through a unified interface:

| Layer | Storage | Purpose | API |
|-------|---------|---------|-----|
| **DB Memory** | SQLite (`pkg/memory`) | Conversation history, auto-learned facts, semantic search | `MemoryModule` via `WithMemory()` |
| **File Memory** | Markdown files | Human-editable persona: SOUL.md, AGENTS.md, TOOLS.md | `MemoryManager` in LongRun |

### DB Memory (agent + LongRun share the same store)

When the agent is built with `WithMemory()`, **LongRun automatically uses the same DB**:
- Task results saved via `memSvc.Add()` (vector-indexed, searchable)
- Context built via `memSvc.RetrieveAndInject()` (semantic recall)
- Falls back to MEMORY.md if agent has no DB memory

### File Memory (persona config, human-editable)

LongRun reads `~/.rago/longrun/`:
- `SOUL.md` — personality
- `AGENTS.md` — agent identity & constraints
- `TOOLS.md` — available tools documentation
- `HEARTBEAT.md` — checklist for autonomous operation (runtime state)

`MEMORY.md` is only used as fallback when DB memory is unavailable.

### Access

```go
// From LongRunService:
lr.GetMemory()        // *MemoryManager (file-based persona files)
lr.GetMemoryService() // domain.MemoryService (DB, same as agent — may be nil)

// From agent Service:
svc.MemoryService()   // domain.MemoryService
```

Each `Service` instance has its own `HookRegistry` (created in `NewService`).
No global state in the hot path — tests can run isolated services without hook cross-contamination.

Hook lifecycle:
```
PreRun → (per tool call: PreToolUse → PostToolUse) → PostRun
```

Auto-memory hook (`RegisterAutoMemoryHook`) saves conversation to memory after each run
when `WithMemory()` is enabled.

## Session Management

- Sessions use **UUID** (not sequential IDs)
- `Chat()` auto-generates a UUID on first call and reuses it
- `Ask()` / `Run()` without `WithSessionID` create ephemeral sessions
- `CompactSession()` summarizes long histories to reduce token usage
