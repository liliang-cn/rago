# AgentGo Architecture

## Overview

AgentGo is a local-first RAG + Agent framework. Core capability priority:

```
RAG System → Cognitive Memory → Multi-Provider LLM → MCP Tools → Agent Automation
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
│  - Build system prompt (Mission/Directives)     │
│  - Hybrid Retrieval (Vector + Navigator)        │
│  - Reflect Engine (Facts → Observations)        │
└──────────────────┬─────────────────────────────┘
                   │
┌──────────────────▼─────────────────────────────┐
│              ToolRegistry (unified)             │
│  Register(name, info, handler)                  │
│  List() → []ToolDefinition                      │
│  Call(name, args) → result                      │
└──────────────────┬──────────────────────────────┘
                   │
     ┌─────────────┼──────────────┬─────────────┐
     ▼             ▼              ▼             ▼
  LLMProvider  CognitiveMem  RAGModule     MCPModule
  (interface)  (Reflect)     (self-reg)    (self-reg)
```

## Cognitive Memory Model

AgentGo's memory system is inspired by **Hindsight** and **PageIndex**, providing an evolving, reasoning-driven cognitive layer.

### Three-Layer Memory Hierarchy
| Level | Name | Description |
|---|---|---|
| **L3** | **Directive / Model** | Hard rules and curated mental models. |
| **L2** | **Observation** | LLM-synthesized insights derived from multiple facts via `Reflect()`. |
| **L1** | **Fact** | Atomic data points extracted from interactions (provenance: `user`, `inferred`). |

### Hybrid Retrieval Strategy
AgentGo uses a parallel search approach to ensure both high recall and high precision:
- **Vector Search (Similarity)**: Uses cosine similarity in SQLite-vec to find semantically related chunks.
- **Index Navigator (Reasoning)**: PageIndex-style search where the LLM reads structural summaries in `_index/` to logically select relevant memories.
- **RRF Fusion**: Results are merged via Reciprocal Rank Fusion, giving higher weight to memories that appear in both tracks.

### Memory Evolution (Reflect Engine)
1. **Facts Intake**: Raw facts are saved as Markdown files.
2. **Periodic Reflection**: When a fact threshold is met, a background process consolidates facts into **Observations**.
3. **Evidence Tracking**: Observations store `EvidenceIDs`. Every belief the agent holds is backed by specific raw facts.
4. **Stale Management**: When a new fact contradicts an old one, the old memory is marked as `stale` and linked via `SupersededBy`.

## Key Packages

| Package | Responsibility |
|---------|---------------|
| `pkg/agent` | Agent builder, execution engine, session management |
| `pkg/rag/processor` | Document ingestion, chunking, vector search |
| `pkg/memory` | Cognitive memory: Reflect engine, Navigator, evolution tracking |
| `pkg/store` | File-based memory store with YAML/MD and indexing logic |
| `pkg/providers` | Unified LLM + Embedder interfaces with provider pool |
| `pkg/ptc` | PTC JS sandbox (goja), tool call transport |

## Programmatic Tool Calling (PTC)

PTC allows the LLM to write JavaScript logic to orchestrate tools, reducing model round-trips:

```
runWithConfig()
  ├── if PTC enabled → runPTCExecution()   [LLM generates JS; goja runs it]
  └── else           → executeWithLLM()    [LLM generates tool calls; Go runs them]
```

## Memory System Implementation

AgentGo uses a **Shadow Index** architecture for memory:
- **Truth Store (File-based)**: Human-editable Markdown/YAML files in `data/memories/`. This is the source of truth for all metadata and evidence.
- **Shadow Index (Vector-based)**: A vector index in `data/agentgo.db` used for fast similarity recall.

## Memory vs Cache

- **Memory** stores durable knowledge. It carries semantics such as `Importance`, `EvidenceIDs`, `SupersededBy`, and revision history.
- **Cache** stores disposable acceleration artifacts. It carries operational metadata such as `ExpiresAt`, `AccessedAt`, and `HitCount`.
- Both may use the filesystem, but they should not be treated as the same subsystem:
  - file memory is the source of truth for local-first cognition
  - file cache is only a restart-friendly performance layer for query/vector/LLM/chunk reuse

### Hierarchy Tracking
`MemoryService.GetEvolution(id)` allows tracing the life of a memory:
`Raw Fact` $\rightarrow$ `Observation` $\rightarrow$ `Superseded Observation`.

## Session Management
- Sessions use **UUIDs** for identification.
- Context is enriched using both conversation history and cognitive memory recall.
- `CompactSession()` summarizes long histories while preserving the cognitive observations.
