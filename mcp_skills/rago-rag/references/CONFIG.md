# Rago Configuration

Location: `~/.rago/config.yaml`

## Example

```yaml
llm:
  default_provider: openai
  providers:
    openai:
      api_key: ${OPENAI_API_KEY}
      model: gpt-4
    anthropic:
      api_key: ${ANTHROPIC_API_KEY}
      model: claude-3-opus-20240229
    ollama:
      base_url: http://localhost:11434
      model: llama2

embedding:
  default_provider: openai
  providers:
    openai:
      model: text-embedding-3-small

rag:
  collection: default
  chunk_size: 512
  overlap: 50

mcp:
  config_paths:
    - ~/.rago/mcpServers.json

skills:
  paths:
    - ~/.agents/skills

memory:
  enabled: true
  db_path: ~/.rago/data/memory.db
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | OpenAI API key |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `RAGO_DEBUG` | Debug mode |
| `RAGO_CONFIG` | Custom config path |

## MCP Servers (mcpServers.json)

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "mcp-filesystem",
      "args": ["--root", "/home/user/docs"]
    }
  }
}
```

## Database Paths

| DB | Path |
|----|------|
| RAG | `~/.rago/data/rag.db` |
| Memory | `~/.rago/data/memory.db` |
| Tasks | `~/.rago/longrun/tasks.db` |
