# RAGO: 专为 Go 开发者设计的自主 Agent 与 RAG 库

[English Documentation](README.md)

RAGO 是一个 **AI Agent SDK**，专为 Go 开发者设计。它赋予您的应用程序“双手”（MCP 工具与技能）、“大脑”（规划与推理）以及“记忆”（向量 RAG 与图谱 RAG）。

## 🤖 构建自主 Agent

RAGO 的 Agent 系统是整个架构的核心“大脑”，它通过协调 LLM、RAG 和 MCP 工具来动态解决复杂任务。

### 零配置 Agent
仅需几行代码即可创建一个集成全方位能力的智能 Agent：

```go
// 创建具备完整能力的 Agent 服务
svc, _ := agent.New(&agent.AgentConfig{
    Name:         "my-agent",
    EnableMCP:    true, // 赋予 Agent “双手” (MCP 工具)
    EnableSkills: true, // 赋予 Agent “专业知识” (Claude Skills)
    EnableMemory: true, // 赋予 Agent “经验” (Hindsight)
    EnableRouter: true, // 赋予 Agent “直觉” (意图识别)
})

// 运行目标 - Agent 将自动规划并执行所有步骤
result, _ := svc.Run(ctx, "研究最新的 Go 语言特性并写一份报告")
```

## 🧠 核心能力

RAGO 由三大支柱构成，共同支撑起应用程序的智能层。

### 1. 混合 RAG (向量 + 知识图谱)
结合了极速向量相似度匹配与基于 SQLite 的 **知识图谱 (GraphRAG)**，实现深度的关联检索。

```go
// 开启 GraphRAG 以提取复杂的实体关系
opts := &rag.IngestOptions{ EnhancedExtraction: true }
client.IngestFile(ctx, "data.pdf", opts)

// 使用混合搜索 (Hybrid Search) 进行查询
resp, _ := client.Query(ctx, "分析数据中的潜在关联", nil)
```

### 2. MCP 与 Claude 兼容的技能
通过 **Model Context Protocol** 和 **Claude 兼容的技能** 极大地扩展 Agent 的能力边界。

```go
// 通过 Markdown 技能和 MCP 服务器添加专家能力
svc, _ := agent.New(&agent.AgentConfig{
    EnableMCP:    true, // 连接外部工具
    EnableSkills: true, // 从 .skills/ 加载 Claude 技能
})
```

### 3. Hindsight：自验证与反思
由 **Hindsight** 系统驱动，Agent 会反思自身表现以确保结果的准确性。

*   **自动纠偏**：通过多轮验证循环自动检测并修复执行过程中的错误。
*   **智能观察**：仅将真正有价值的观察结果和见解存入长期记忆。

## 🧠 核心支柱

| 特性 | 描述 |
| :--- | :--- |
| **自主 Agent** | 动态任务拆解（Planner）和多轮工具执行（Executor）。 |
| **意图识别** | 高速语义路由和基于 LLM 的目标分类。 |
| **Hindsight 记忆** | 自反思记忆系统，存储经验证的见解并自动纠错。 |
| **工具集成** | 原生支持 **MCP (Model Context Protocol)** 和 **Claude 兼容的技能**。 |
| **混合 RAG** | 向量搜索 + 基于 SQLite 的 **知识图谱 (GraphRAG)**。 |
| **本地优先** | 完全支持离线运行（搭配 Ollama/LM Studio），也可连接 OpenAI/DeepSeek。 |

## 📦 安装

```bash
go get github.com/liliang-cn/rago/v2
```

## ⚙️ 配置

RAGO 会按以下顺序自动查找配置文件：
1.  `./rago.toml` (当前目录)
2.  `~/.rago/rago.toml`
3.  `~/.rago/config/rago.toml` (推荐)

您可以从 `rago.toml.example` 复制模板开始配置：
```bash
mkdir -p ~/.rago/config
cp rago.toml.example ~/.rago/config/rago.toml
```

或者使用 init 命令：
```bash
rago init              # 在 ~/.rago/ 初始化
rago init -d ~/my-rago  # 自定义目录
```

## 🔌 扩展 RAGO

### MCP 服务器

通过 Model Context Protocol 添加外部工具：

```bash
# 添加服务器
rago mcp add websearch mcp-websearch-server
rago mcp add filesystem mcp-filesystem-server ~/.rago/

# 列出可用工具
rago mcp list

# 直接调用工具
rago mcp call mcp_websearch_websearch_basic '{"query": "golang news", "max_results": 5}'
```

服务器配置存储在 `~/.rago/mcpServers.json`：

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

### Skills 技能

Skills 是定义特定领域能力的 Markdown 文件。将它们放在 `~/.rago/.skills/` 目录：

```markdown
<!-- ~/.rago/.skills/weather.md -->
---
description: 获取当前天气和预报
args:
  - name: location
    description: 城市名称
    type: string
    required: true
---

# 天气助手

你是一个天气助手。使用 mcp_websearch 工具查找 {{location}} 的当前天气信息。

请提供温度、天气状况和预报的简明信息。
```

```bash
# 列出已加载的技能
rago skills list

# 测试技能
rago skills call weather '{"location": "北京"}'
```

### Intents 意图

Intents 实现语义路由 - 将用户目标自动匹配到合适的工具。将它们放在 `~/.rago/.intents/` 目录：

```markdown
<!-- ~/.rago/.intents/filesystem.md -->
---
label: filesystem
description: 文件系统操作
examples:
  - "列出当前目录的文件"
  - "读取 README.md"
  - "创建一个新文件"
  - "删除临时文件"
tools:
  - mcp_filesystem_list_directory
  - mcp_filesystem_read_file
  - mcp_filesystem_write_file
  - mcp_filesystem_delete_file
priority: 0.8
---

此意图使用 MCP 文件系统工具处理文件系统操作。
```

```bash
# 列出已注册的意图
rago agent plan list

# 测试意图识别
rago agent intents recognize "显示所有 go 文件"
```

## 🏗️ 架构设计

RAGO 旨在成为您应用程序的 **智能层（Intelligence Layer）**：

- **`pkg/agent`**: 核心 Agent 循环（规划器/执行器/会话）。
- **`pkg/skills`**: 垂直领域能力的插件系统。
- **`pkg/mcp`**: 标准化外部工具的连接器。
- **`pkg/rag`**: 知识检索引擎。

## 📊 CLI vs Library

RAGO 提供强大的 CLI 用于管理，但它已针对库使用进行了深度优化：
- **CLI**: `./rago agent run "任务目标"`
- **Library**: `agentSvc.Run(ctx, "任务目标")`

## 🔌 库 API：MCP、Skills 与 Intents

### 在代码中使用 MCP

```go
import (
    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/mcp"
    "github.com/liliang-cn/rago/v2/pkg/services"
)

// 加载配置并初始化 LLM
cfg, _ := config.Load("")
globalPool := services.GetGlobalPoolService()
globalPool.Initialize(ctx, cfg)
llmSvc, _ := globalPool.GetLLMService()

// 创建 MCP 服务
mcpSvc, _ := mcp.NewService(&cfg.MCP, llmSvc)
mcpSvc.StartServers(ctx, nil)

// 列出可用工具
tools := mcpSvc.GetAvailableTools(ctx)
for _, tool := range tools {
    fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
}

// 调用工具
result, _ := mcpSvc.CallTool(ctx, "mcp_websearch_websearch_basic", map[string]interface{}{
    "query": "golang 新闻",
    "max_results": 5,
})
```

### 在代码中使用 Skills

```go
import "github.com/liliang-cn/rago/v2/pkg/skills"

// 创建技能服务
skillsCfg := skills.DefaultConfig()
skillsCfg.Paths = []string{cfg.SkillsDir()} // ~/.rago/.skills
skillsSvc, _ := skills.NewService(skillsCfg)

// 从目录加载所有技能
skillsSvc.LoadAll(ctx)

// 列出可用技能
allSkills, _ := skillsSvc.ListSkills(ctx, skills.SkillFilter{})
for _, skill := range allSkills {
    fmt.Printf("- %s: %s\n", skill.ID, skill.Description)
}

// 直接调用技能
result, _ := skillsSvc.Call(ctx, "weather", map[string]interface{}{
    "location": "北京",
})
```

### 在代码中使用 Intents (路由)

```go
import "github.com/liliang-cn/rago/v2/pkg/router"

// 创建路由服务
routerCfg := router.DefaultConfig()
routerCfg.Threshold = 0.75
routerSvc, _ := router.NewService(embedSvc, routerCfg)

// 注册默认意图
routerSvc.RegisterDefaultIntents()

// 或从目录注册
routerSvc.RegisterIntentsFrom(cfg.IntentsDir()) // ~/.rago/.intents

// 测试意图识别
result, _ := routerSvc.Route(ctx, "今天天气怎么样？")
if result.Matched {
    fmt.Printf("匹配: %s (得分: %.2f)\n", result.IntentName, result.Score)
    fmt.Printf("工具: %s\n", result.ToolName)
}

// 列出所有已注册的意图
intents := routerSvc.ListIntents()
for _, intent := range intents {
    fmt.Printf("- %s: %s\n", intent.Name, intent.Description)
}
```

### 完整 Agent 示例

```go
import "github.com/liliang-cn/rago/v2/pkg/agent"

// 创建启用所有功能的 Agent
svc, _ := agent.New(&agent.AgentConfig{
    Name:         "my-agent",
    EnableMCP:    true,  // MCP 工具
    EnableSkills: true,  // 技能
    EnableRouter: true,  // 意图路由
    EnableMemory: true,  // 长期记忆
    EnableRAG:    true,  // RAG 功能
    RouterThreshold: 0.75,
})
defer svc.Close()

// 运行目标 - Agent 将自动规划并执行
result, _ := svc.Run(ctx, "研究最新的 Go 特性并总结")
fmt.Println(result.FinalResult)
```

## 📚 文档与示例

*   **[Agent 技能集成](./examples/skills_integration/)**: 连接自定义工具。
*   **[会话压缩](./examples/compact_session/)**: 管理 Agent 的长期上下文。
*   **[混合 RAG 系统](./examples/advanced_library_usage/)**: 构建知识库。
*   **[快速入门指南](./examples/quickstart/)**: Go 应用的基本配置。

## 📄 许可证
MIT License - Copyright (c) 2024-2025 RAGO Authors.