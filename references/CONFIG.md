# Rago Configuration

Config file: `rago.toml`  
Auto-discovered in: `./` → `~/.rago/` → `~/.rago/config/`

## Full Example

```toml
home = "~/.rago"   # base directory

[llm_pool]
enabled  = true
strategy = "round_robin"

[[llm_pool.providers]]
name            = "openai"
base_url        = "https://api.openai.com/v1"
key             = "sk-..."
model_name      = "gpt-4o"

# ── RAG Vector Store ───────────────────────────────────
[cortexdb]
db_path    = ""      # default: $home/data/rago.db
index_type = "hnsw"

# ── Cognitive Memory ────────────────────────────────────
[memory]
store_type   = "file"   # "file" (Truth Store) | "vector" | "hybrid"
memory_path  = ""       # default: $home/data/memories
reflect_threshold = 5   # Auto-reflect facts into observations after 5 new entries
min_score    = 0.0
max_memories = 5

  [memory.hybrid]
  enabled        = false  # Parallel Vector + Index reasoning
  rrf_k          = 60.0   # Reciprocal Rank Fusion constant

  # Hindsight Evolution
  [memory.hindsight]
  auto_reflect   = true
  evidence_tracking = true
  stale_management = true

  [memory.scoring]
  enabled            = true
  recency_weight     = 0.3
  importance_weight  = 0.4
  enable_recency     = true
  enable_importance  = true

# ── Skills ──────────────────────────────────────────────
[skills]
enabled   = true
paths     = ["~/.rago/skills"]
auto_load = true

# ── MCP ─────────────────────────────────────────────────
[mcp]
servers = ["~/.rago/mcpServers.json"]
```

---

## Directory Layout (default `home = ~/.rago`)

```
~/.rago/
├── rago.toml              ← config file
├── mcpServers.json        ← MCP server definitions
├── data/
│   ├── rago.db            ← RAG + Memory Shadow Index (sqlite-vec)
│   ├── agent.db           ← Agent sessions + execution plans
│   └── memories/          ← Cognitive Memory Store (Truth)
│       ├── entities/      ← Fact and Observation files (.md)
│       ├── streams/       ← Context streams
│       └── _index/        ← PageIndex summaries (.md)
├── skills/                ← SKILL.md files
└── workspace/             ← Agent working directory
```

---

## SQLite Files

| File | Config Key | Tables | Purpose |
|------|-----------|--------|---------|
| `data/rago.db` | `cortexdb.db_path` | `chunks`, `embeddings`, `memories` | RAG vector index + Memory vector index (shadow) |
| `data/agent.db` | `builder.WithDBPath()` | `agent_sessions`, `agent_plans` | Multi-turn chat history and plan state |

---

## Memory Store Types

| `store_type` | Storage | Retrieval | Mode |
|-------------|---------|-----------|------|
| `file` *(def)* | `data/memories/` | Index Navigator (Reasoning) | PageIndex |
| `vector` | `data/rago.db` | Vector Similarity | Semantic |
| `hybrid` | Both | RRF Fusion (Vector + Navigator) | Cognitive |

## Environment Variables

| Variable | Config equivalent | Description |
|----------|------------------|-------------|
| `RAGO_HOME` | `home` | Override base home directory |
| `RAGO_CORTEXDB_DB_PATH` | `cortexdb.db_path` | Override RAG/Shadow database path |

## Path Resolution Priority

```
Explicit Builder Code > rago.toml > Environment variable > Default
```
