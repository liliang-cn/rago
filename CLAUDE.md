# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Architecture Priority (Important!)

**RAGO's True Nature**: Primarily a **local RAG system** with **optional agent automation**. Core: document ingestion → chunking → vector storage → semantic search → Q&A. Agents are a secondary feature leveraging MCP tools.

**Feature Priority**: RAG System → Multi-Provider LLM → MCP Tools → Agent Automation → HTTP API → Local-First

**Evidence**: Core packages (`pkg/store/`, `pkg/chunker/`, `pkg/processor/`) vs optional agents (`pkg/agents/`). Most commands are RAG-focused (`ingest`, `query`) vs agent additions (`agent run`).

## Development Commands

### Build and Test
```bash
# Build main binary
make build

# Run all tests
make test

# Run linting and race condition tests  
make check

# Build from cmd directory
go build -o rago ./cmd/rago

# Run specific package tests
go test ./pkg/providers -v
go test ./pkg/mcp -race
```

### Running RAGO
```bash
# Initialize configuration
./rago init

# Test provider connectivity
./rago status

# Check MCP server status
./rago mcp status

# Start HTTP server
./rago serve --port 7127

# RAG operations
./rago ingest document.pdf
./rago query "What is this about?" --show-sources
```

### MCP Server Setup
```bash
# Install default MCP servers
./scripts/install-mcp-servers.sh

# Check server status
./rago mcp status --verbose
```

## Key Architecture Patterns

**RAG Pipeline**: `pkg/processor/` → `pkg/chunker/` → `pkg/embedder/` → `pkg/store/`  
Dual storage: SQLite vectors (sqvect) + keyword search (Bleve) with RRF fusion

**Provider Factory**: `pkg/providers/factory.go` - Creates LLM/embedder providers from config  
Supports Ollama, OpenAI, LM Studio with unified interfaces

**MCP-First Tools**: `pkg/mcp/` replaces built-in tools with standardized MCP servers  
Configuration via `mcpServers.json`, health checking, external tool integration

**Optional Agents**: `pkg/agents/` - Workflow automation layer on top of RAG  
Natural language → JSON workflow → MCP tool execution

## Configuration

**Config Loading**: `~/.rago/rago.toml` → `./rago.toml` → `RAGO_*` env vars → defaults  
**MCP Servers**: `mcpServers.json` (external tool definitions)  
**Provider Setup**: Must configure at least one LLM/embedder in `[providers]` section



## Key Files

- `pkg/config/config.go` - Configuration loading
- `pkg/providers/factory.go` - Provider creation  
- `pkg/processor/service.go` - RAG pipeline coordination
- `pkg/mcp/client.go` - MCP protocol client
- `client/client.go` - Simplified library interface


## Common Development Tasks

### Examples Generation
Use the `/examples` command to create runnable examples:
```bash
# This will:
# 1. Analyze the requested feature/task
# 2. Create appropriate folder structure in /examples/
# 3. Generate complete, runnable Go file with imports and error handling
# 4. Run the example with: go run examples/$feature/$filename.go
```

**Example Structure**:
```
examples/
├── basic_rag_usage/main.go          # Simple RAG operations
├── provider_switching/main.go       # Multi-provider examples  
├── mcp_integration/main.go          # MCP tool usage
├── agent_workflows/main.go          # Agent automation
├── custom_chunking/main.go          # Chunking strategies
└── http_api_client/main.go          # API usage examples
```

**Best Practices for Examples**:
- Each example in its own folder with `main.go`
- Include proper imports, error handling, and cleanup
- Add comments explaining key concepts
- Use realistic data and scenarios
- Follow Go best practices and project patterns

### Release Workflow
Use the `/release` command for automated releases:
```bash
# Custom slash command available: /release [optional commit message]
# This will:
# 1. Check git status and recent tags  
# 2. Add non-binary files to git
# 3. Analyze commit content to determine semantic version bump
# 4. Create commit without co-author
# 5. Create and push tag (auto-determined from changes)
# 6. Push changes to remote
```

**Custom Command**: The `/release` slash command is implemented in `.claude/commands/release.md`

**Tag Version Determination**:
- **MAJOR** (x.0.0): Breaking changes, API changes, architecture changes
- **MINOR** (x.y.0): New features, new commands, significant enhancements
- **PATCH** (x.y.z): Bug fixes, documentation updates, small improvements

**Manual Release Steps** (if needed):
```bash
# Check current status and analyze changes
git status
git tag --sort=-version:refname | head -5
git diff HEAD~1..HEAD --stat  # Review changes for version determination

# Add files (excluding binaries)
git add README.md README_zh-CN.md docs/index.html
git add examples/ pkg/ cmd/
git add *.json *.md mcpServers.json

# Create commit (message affects version determination)
git commit -m "feat: your commit message here"

# Create and push tag (version determined by content analysis)
# PATCH: "fix:", "docs:", "style:", "refactor:", "test:", "chore:"
# MINOR: "feat:", new functionality, new commands
# MAJOR: "BREAKING CHANGE:", API changes, architectural changes
git tag -a v2.x.x -m "Release v2.x.x: description"
git push origin main --tags
```

### Debugging Provider Issues
1. Check provider connectivity: `./rago status --verbose`
2. Test with minimal query: `./rago query "test" --verbose`
3. Verify model availability on provider (Ollama: `ollama list`)

### Debugging MCP Issues  
1. Check server status: `./rago mcp status`
2. Test tool directly: `./rago mcp tools call filesystem read_file '{"path": "test.txt"}'`
3. Review server logs in `~/.rago/logs/`

### Adding New Commands
1. Create command file in `cmd/rago/`
2. Add to root command in `cmd/rago/root.go`
3. Follow existing patterns for config loading and error handling
4. Add corresponding API endpoint in `api/handlers/` if needed

The codebase follows Go best practices with clear separation of concerns, comprehensive error handling, and extensive configuration options for different deployment scenarios.
- go run . to test it