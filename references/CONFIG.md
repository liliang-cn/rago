# Rago Configuration

Config file: `rago.toml`  
Auto-discovered in: `./` → `~/.rago/` → `~/.rago/config/`

## Full Example

```toml
# All paths are relative to `home` unless absolute.
home = "~/.rago"   # base directory; defaults to directory of the config file

[server]
port        = 7127
host        = "0.0.0.0"
enable_ui   = true
cors_origins = []

# ── LLM Pool ────────────────────────────────────────────────────────────────
[llm_pool]
enabled  = true
strategy = "round_robin"   # round_robin | random | least_load | capability | failover

[[llm_pool.providers]]
name            = "openai"
base_url        = "https://api.openai.com/v1"
key             = "sk-..."
model_name      = "gpt-4o"
max_concurrency = 5
capability      = 5

[[llm_pool.providers]]
name            = "local"
base_url        = "http://localhost:11434/v1"
key             = ""
model_name      = "qwen2.5:14b"
max_concurrency = 3
capability      = 3

# ── Embedding Pool ───────────────────────────────────────────────────────────
[embedding_pool]
enabled  = true
strategy = "round_robin"

[[embedding_pool.providers]]
name            = "local-embed"
base_url        = "http://localhost:11434/v1"
key             = ""
model_name      = "nomic-embed-text"
max_concurrency = 10
capability      = 3

# ── RAG Vector Store (sqvect / sqlite-vec) ───────────────────────────────────
[sqvect]
db_path    = ""      # default: $home/data/rago.db
max_conns  = 10
batch_size = 100
top_k      = 5
threshold  = 0.7
index_type = "hnsw"

# ── Document Chunking ────────────────────────────────────────────────────────
[chunker]
chunk_size = 512
overlap    = 64
method     = "sentence"   # sentence | fixed | paragraph

[ingest]
  [ingest.metadata_extraction]
  enable = false

# ── Memory ───────────────────────────────────────────────────────────────────
[memory]
store_type   = "file"   # "file" | "vector" | "hybrid"
memory_path  = ""       # default: $home/data/memories  (file/hybrid)
                        #          $home/data/rago.db   (vector, shared with RAG)
min_score    = 0.0
max_memories = 10

  [memory.scoring]
  enabled            = true
  recency_weight     = 0.3
  half_life_days     = 30.0
  enable_recency     = true
  importance_weight  = 0.4
  min_importance     = 0.3
  enable_importance  = true
  length_norm_weight = 0.1
  anchor_length      = 200
  enable_length_norm = true
  access_boost_weight = 0.2
  enable_access_boost = true

  [memory.noise_filter]
  enabled           = true
  min_content_length = 20
  filter_refusals   = true
  filter_meta       = true
  filter_duplicates = true

  [memory.adaptive]
  enabled         = true
  min_query_length = 10

  [memory.hybrid]
  enabled        = false
  rrf_k          = 60.0
  vector_weight  = 0.6
  bm25_weight    = 0.4

# ── Skills ───────────────────────────────────────────────────────────────────
[skills]
enabled                = true
paths                  = []     # extra SKILL.md search paths
auto_load              = true
allow_command_injection = false
require_confirmation   = false

# ── MCP ──────────────────────────────────────────────────────────────────────
[mcp]
servers = ["~/.rago/mcpServers.json"]   # list of mcpServers.json paths
```

---

## Directory Layout (default `home = ~/.rago`)

```
~/.rago/
├── rago.toml              ← config file
├── mcpServers.json        ← MCP server definitions
├── data/
│   ├── rago.db            ← RAG vector store (sqlite-vec)
│   │                        also used as Memory vector store when store_type=vector
│   ├── agent.db           ← Agent sessions + execution plans
│   └── memories/          ← Memory file store (store_type=file, one JSON per session)
├── skills/                ← SKILL.md files
├── intents/               ← Intent YAML files
└── workspace/             ← Agent working directory
```

---

## SQLite Files

| File | Config Key | Tables | Notes |
|------|-----------|--------|-------|
| `data/rago.db` | `sqvect.db_path` | `documents`, `chunks`, `embeddings` | RAG vector store; shared as Memory vector store when `memory.store_type = "vector"` |
| `data/agent.db` | `builder.WithDBPath()` | `agent_sessions`, `agent_plans` | Agent conversation history and plan state |
| `data/history.db` *(opt)* | `WithHistoryDBPath()` | `execution_history`, `tool_results` | Detailed execution log; only created when `WithStoreHistory(true)` is set at runtime |

> **Memory store_type behaviour:**
> - `file`   → `data/memories/{session_id}.json` (no SQLite, no embedder required)
> - `vector` → `data/rago.db` (shared file, requires embedder)
> - `hybrid` → file store primary + `data/rago.db` shadow index for vector recall

---

## Environment Variables

| Variable | Config equivalent | Description |
|----------|------------------|-------------|
| `RAGO_SQVECT_DB_PATH` | `sqvect.db_path` | Override RAG database path |
| `RAGO_DEBUG` | `debug = true` | Enable debug logging |

---

## MCP Servers (`mcpServers.json`)

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/home/user/docs"]
    },
    "brave-search": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-brave-search"],
      "env": { "BRAVE_API_KEY": "BSA..." }
    }
  }
}
```

---

## Path Resolution Priority

```
Explicit code (WithDBPath) > rago.toml > Environment variable > Default
```

Default paths when `home = ~/.rago`:

| Resource | Default path |
|----------|-------------|
| RAG DB | `~/.rago/data/rago.db` |
| Agent DB | `~/.rago/data/agent.db` |
| Memory (file) | `~/.rago/data/memories/` |
| Memory (vector) | `~/.rago/data/rago.db` |
| Skills | `~/.rago/skills/` |
| Intents | `~/.rago/intents/` |
| Workspace | `~/.rago/workspace/` |
| MCP config | `~/.rago/mcpServers.json` |
