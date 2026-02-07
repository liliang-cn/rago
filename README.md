# RAGO: Autonomous Agent & RAG Library for Go

[中文文档](README_zh-CN.md)

RAGO is an **AI Agent SDK** designed for Go developers. It enables you to build autonomous agents with "hands" (MCP Tools & Skills), "brains" (Planning & Reasoning), and "memory" (Vector RAG & GraphRAG).

## 🤖 Building Autonomous Agents

RAGO's agent system acts as the central brain, orchestrating all other components (LLM, RAG, MCP) to solve complex tasks dynamically.

### Autonomous Agent
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

RAGO is organized into three powerful pillars that form the intelligence layer of your application.

### 1. Hybrid RAG (Vector + Knowledge Graph)
Combines high-speed vector similarity with **GraphRAG** for deep relationship discovery and context retrieval.

```go
// Enable GraphRAG for complex entity extraction
opts := &rag.IngestOptions{ EnhancedExtraction: true }
client.IngestFile(ctx, "data.pdf", opts)

// Query using Hybrid Search (Vector + Graph)
resp, _ := client.Query(ctx, "Analyze relationships in the data", nil)
```

### 2. MCP & Claude-compatible Skills
Extend your agent with **Model Context Protocol** tools and **Claude-compatible Skills**.

```go
// Add expert capabilities via Markdown skills and MCP servers
svc, _ := agent.New(&agent.AgentConfig{
    EnableMCP:    true, // Connect to external tools
    EnableSkills: true, // Loads Claude skills from .skills/
})
```

### 3. Hindsight: Self-Verification & Reflection
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

## ⚙️ Configuration

RAGO automatically looks for a configuration file in the following order:
1.  `./rago.toml` (Current directory)
2.  `~/.rago/rago.toml`
3.  `~/.rago/config/rago.toml` (Recommended)

You can copy the template from `rago.toml.example` to get started:
```bash
mkdir -p ~/.rago/config
cp rago.toml.example ~/.rago/config/rago.toml
```

Or use the init command:
```bash
rago init              # Initialize in ~/.rago/
rago init -d ~/my-rago  # Custom directory
```

## 🔌 Extending RAGO

### MCP Servers

Add external tools via Model Context Protocol:

```bash
# Add a server
rago mcp add websearch mcp-websearch-server
rago mcp add filesystem mcp-filesystem-server ~/.rago/

# List available tools
rago mcp list

# Call a tool directly
rago mcp call mcp_websearch_websearch_basic '{"query": "golang news", "max_results": 5}'
```

Servers are stored in `~/.rago/mcpServers.json`:

```json
{
  "mcpServers": {
    "websearch": {
      "type": "stdio",
      "command": "mcp-websearch-server"
    },
    "filesystem": {
      "type": "stdio",
      "command": "mcp-filesystem-server",
      "args": ["/Users/liliang/.rago/"]
    }
  }
}
```

### Skills

Skills are Markdown files that define domain-specific capabilities. Place them in `~/.rago/.skills/`:

```markdown
<!-- ~/.rago/.skills/weather.md -->
---
description: Get current weather and forecasts
args:
  - name: location
    description: City name
    type: string
    required: true
---

# Weather Skill

You are a weather assistant. Use the mcp_websearch tool to find current weather information for {{location}}.

Provide temperature, conditions, and forecast in a concise format.
```

```bash
# List loaded skills
rago skills list

# Test a skill
rago skills call weather '{"location": "Beijing"}'
```

### Intents

Intents enable semantic routing - matching user goals to appropriate tools automatically. Place them in `~/.rago/.intents/`:

```markdown
<!-- ~/.rago/.intents/filesystem.md -->
---
label: filesystem
description: File system operations
examples:
  - "list files in current directory"
  - "read README.md"
  - "create a new file"
  - "delete temporary files"
tools:
  - mcp_filesystem_list_directory
  - mcp_filesystem_read_file
  - mcp_filesystem_write_file
  - mcp_filesystem_delete_file
priority: 0.8
---

This intent handles file system operations using MCP filesystem tools.
```

```bash
# List registered intents
rago agent intents list

# Test intent recognition
rago agent intents recognize "show me all go files"
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

## 🔌 Library API: MCP, Skills & Intents

### Using MCP in Your Code

```go
import (
    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/mcp"
    "github.com/liliang-cn/rago/v2/pkg/services"
)

// Load config and initialize LLM
cfg, _ := config.Load("")
globalPool := services.GetGlobalPoolService()
globalPool.Initialize(ctx, cfg)
llmSvc, _ := globalPool.GetLLMService()

// Create MCP service
mcpSvc, _ := mcp.NewService(&cfg.MCP, llmSvc)
mcpSvc.StartServers(ctx, nil)

// List available tools
tools := mcpSvc.GetAvailableTools(ctx)
for _, tool := range tools {
    fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
}

// Call a tool
result, _ := mcpSvc.CallTool(ctx, "mcp_websearch_websearch_basic", map[string]interface{}{
    "query": "golang news",
    "max_results": 5,
})
```

### Using Skills in Your Code

```go
import "github.com/liliang-cn/rago/v2/pkg/skills"

// Create skills service
skillsCfg := skills.DefaultConfig()
skillsCfg.Paths = []string{cfg.SkillsDir()} // ~/.rago/.skills
skillsSvc, _ := skills.NewService(skillsCfg)

// Load all skills from directory
skillsSvc.LoadAll(ctx)

// List available skills
allSkills, _ := skillsSvc.ListSkills(ctx, skills.SkillFilter{})
for _, skill := range allSkills {
    fmt.Printf("- %s: %s\n", skill.ID, skill.Description)
}

// Call a skill directly
result, _ := skillsSvc.Call(ctx, "weather", map[string]interface{}{
    "location": "Beijing",
})
```

### Using Intents (Router) in Your Code

```go
import "github.com/liliang-cn/rago/v2/pkg/router"

// Create router service
routerCfg := router.DefaultConfig()
routerCfg.Threshold = 0.75
routerSvc, _ := router.NewService(embedSvc, routerCfg)

// Register default intents
routerSvc.RegisterDefaultIntents()

// Or register from directory
routerSvc.RegisterIntentsFrom(cfg.IntentsDir()) // ~/.rago/.intents

// Test intent recognition
result, _ := routerSvc.Route(ctx, "What's the weather today?")
if result.Matched {
    fmt.Printf("Matched: %s (Score: %.2f)\n", result.IntentName, result.Score)
    fmt.Printf("Tool: %s\n", result.ToolName)
}

// List all registered intents
intents := routerSvc.ListIntents()
for _, intent := range intents {
    fmt.Printf("- %s: %s\n", intent.Name, intent.Description)
}
```

### Complete Agent Example

```go
import "github.com/liliang-cn/rago/v2/pkg/agent"

// Create agent with all features enabled
svc, _ := agent.New(&agent.AgentConfig{
    Name:         "my-agent",
    EnableMCP:    true,  // MCP tools
    EnableSkills: true,  // Skills
    EnableRouter: true,  // Intent routing
    EnableMemory: true,  // Long-term memory
    EnableRAG:    true,  // RAG features
    RouterThreshold: 0.75,
})
defer svc.Close()

// Run a goal - agent will plan and execute automatically
result, _ := svc.Run(ctx, "Research latest Go features and summarize")
fmt.Println(result.FinalResult)
```

## 📚 Documentation & Examples

*   **[Quickstart Guide](./examples/quickstart/)**: Basic setup for Go apps.
*   **[Advanced RAG](./examples/advanced_rag/)**: Building the knowledge base with metadata and filters.
*   **[Agent Usage](./examples/agent_usage/)**: Creating autonomous agents with planning and execution.
*   **[Skills Integration](./examples/skills_integration/)**: Connecting custom tools and Claude skills.
*   **[Session Compaction](./examples/compact_session/)**: Managing long-term agent context.
*   **[Intent Routing](./examples/intent_routing/)**: Semantic routing and intent recognition.

## 📄 License
MIT License - Copyright (c) 2024-2025 RAGO Authors.