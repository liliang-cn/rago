# RAGO: Go 语言原生的一站式 AI 智能体框架

[English Documentation](README.md)

RAGO 是一个专为 Go 开发者打造的生产级 **AI Agent 框架**。它不仅仅是一个简单的 LLM 包装器，而是提供了一个完整的运行时环境，集成了 **混合 RAG（向量+图谱）**、**多智能体协作**、**MCP 工具协议** 以及 **具备自反思能力的记忆系统**。

它旨在帮助开发者构建从简单的聊天机器人到复杂的、拥有长期记忆和工具使用能力的自主智能体。

## 🌟 为什么选择 RAGO？

RAGO 解决了构建复杂 AI 应用时的核心痛点，且完全本地化，无需依赖 Python 生态。

| 核心支柱 | 关键能力 |
| :--- | :--- |
| **🧠 推理引擎** | **规划与执行 (Planner/Executor)**、**意图识别** 以及 **多智能体协作 (Handoffs)**。能够自动拆解并解决多步骤复杂任务。 |
| **📚 知识引擎** | **混合 RAG**: 结合了极速向量搜索与 **基于 SQLite 的 GraphRAG**，能够发现数据间深层的关联。 |
| **🛠️ 工具引擎** | 原生支持 **MCP (Model Context Protocol)**、**Claude 兼容技能**，以及 **动态 Go 函数注册**。 |
| **💾 记忆系统** | **Hindsight 架构**: 包含短期上下文、长期事实记忆、实体追踪以及 **反思 (Reflection)** 机制。 |
| **⚡ 运行时** | **事件驱动循环 (Event-Driven Loop)**: 支持实时流式输出 (Token-by-token)、状态管理和全异步执行。 |
| **🔒 本地优先** | 设计为既可离线运行（**Ollama**），也可连接云端（**OpenAI/DeepSeek**）。数据完全掌控。 |

---

## 📦 安装

```bash
go get github.com/liliang-cn/rago/v2
```

## 🚀 快速开始：Hello World Agent

创建一个能够规划、思考并执行任务的 Agent。

```go
package main

import (
    "context"
    "fmt"
    "github.com/liliang-cn/rago/v2/pkg/agent"
)

func main() {
    ctx := context.Background()

    // 1. 初始化服务 (运行时环境)
    svc, _ := agent.New(&agent.AgentConfig{
        Name:         "my-assistant",
        EnableMCP:    true, // 启用外部工具
        EnableMemory: true, // 启用长期记忆
    })
    defer svc.Close()

    // 2. 运行任务 (使用流式接口获取实时反馈)
    events, _ := svc.RunStream(ctx, "研究 Go 1.24 的最新特性并写一份总结。")

    // 3. 消费事件循环
    for evt := range events {
        switch evt.Type {
        case agent.EventTypeThinking:
            fmt.Println("🤖 正在思考...")
        case agent.EventTypeToolCall:
            fmt.Printf("🛠️  调用工具: %s\n", evt.ToolName)
        case agent.EventTypePartial:
            fmt.Print(evt.Content) // 实时打字机效果
        }
    }
}
```

---

## 🏗️ 架构与特性深度解析

### 1. 混合 RAG (向量 + 知识图谱)
RAGO 不仅仅存储向量，它还会自动构建 **知识图谱**。

*   **向量搜索**: 用于语义相似度匹配。
*   **GraphRAG**: 用于“连接点与点”，发现实体间的隐式关系。

```go
// 导入文档并开启增强图谱提取
client.IngestFile(ctx, "manual.pdf", &rag.IngestOptions{ EnhancedExtraction: true })

// 使用混合搜索进行查询
resp, _ := client.Query(ctx, "分析模块 A 和模块 B 之间的潜在关系", nil)
```

### 2. 多智能体协作 (Handoffs)
构建复杂的协作系统。你可以定义一个“分诊” Agent，将任务自动路由给“安全专家”或“数学专家”。

```go
// 定义专家 Agent
mathAgent := agent.NewAgent("MathExpert")
mathAgent.SetInstructions("你负责解决复杂的数学计算问题。")

// 定义分诊 Agent 并添加转接能力
triageAgent := agent.NewAgent("Receptionist")
triageAgent.AddHandoff(agent.NewHandoff(mathAgent, 
    agent.WithHandoffToolDescription("将计算类任务转交给数学专家。"),
))

// 运行时会自动处理 Agent 切换
svc.RegisterAgent(mathAgent)
svc.RegisterAgent(triageAgent)
```

### 3. 通用工具接口 (MCP & Code)
RAGO 统一了所有工具的调用接口。

*   **MCP 服务器**: 通过标准协议连接文件系统、GitHub、数据库或浏览器。
*   **Go 函数**: 直接在代码中注册你自己的业务逻辑函数。

```go
// 1. 添加标准 MCP 服务器 (例如 Brave 搜索)
svc.AddMCPServer(ctx, "brave", "npx", []string{"-y", "@modelcontextprotocol/server-brave-search"})

// 2. 添加原生 Go 函数作为工具
agent.AddTool("check_status", "检查系统状态", 
    map[string]interface{}{"type": "object", "properties": {...}},
    func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
        return "系统运行正常", nil
    },
)
```

### 4. Hindsight 记忆系统 (自反思)
记忆系统不仅仅是一个日志。它会 **反思 (Reflect)** 交互过程，提炼出“观察 (Observations)”和“思维模型 (Mental Models)”。

```go
// 配置记忆库的“性格”
svc.ConfigureMemory(ctx, &domain.MemoryBankConfig{
    Mission:    "你是一个极其严谨的审计员。",
    Skepticism: 5, // 高度怀疑，不轻信未验证的信息
})

// 触发反思循环以整合见解
summary, _ := svc.ReflectMemory(ctx)
```

---

## 💻 CLI 使用

RAGO 提供了一个强大的 CLI 来管理整个生命周期。

```bash
# 1. 启动任务 (支持实时流式输出)
rago agent run "查找当前目录下的重复文件并清理"

# 2. 管理 RAG 知识库
rago rag ingest ./docs/ --recursive
rago rag query "如何配置服务器端口？"

# 3. 管理 MCP 工具
rago mcp list
rago mcp install filesystem
```

## ⚙️ 配置指南

RAGO 会在 `./`, `~/.rago/`, 或 `~/.rago/config/` 查找 `rago.toml`。

```toml
[server]
port = 7127

[llm_pool]
enabled = true
strategy = "round_robin"

[[llm_pool.providers]]
name = "openai"
provider = "openai"
api_key = "sk-..."
model = "gpt-4-turbo"

[mcp]
enabled = true
# MCP 服务器定义在 mcpServers.json 中
```

## 📚 示例代码

请查看 `examples/` 目录获取深入的示例：

*   **[multi_agent_orchestration](./examples/multi_agent_orchestration/)**: 包含 Handoffs、动态工具和流式输出的完整演示。
*   **[advanced_rag](./examples/advanced_rag/)**: 构建带有元数据过滤的知识库。
*   **[skills_integration](./examples/skills_integration/)**: 使用 Claude 兼容的 Markdown 技能。

## 📄 许可证
MIT License - Copyright (c) 2024-2025 RAGO Authors.