# RAGO: Go 语言原生的一站式 AI 智能体框架

[English Documentation](README.md)

RAGO 是一个专为 Go 开发者打造的生产级 **AI Agent 框架**。它不仅仅是一个简单的 LLM 包装器，而是提供了一个完整的运行时环境，集成了 **混合 RAG（向量+图谱）**、**多智能体协作**、**MCP 工具协议** 以及 **极致透明的文件记忆系统**。

它旨在帮助开发者构建从简单的聊天机器人到复杂的、拥有长期记忆和工具使用能力的自主智能体。

## 🌟 为什么选择 RAGO？

RAGO 解决了构建复杂 AI 应用时的核心痛点，且采用 **分层架构** 设计，一切皆可控。

| 核心支柱 | 关键能力 |
| :--- | :--- |
| **🧠 推理引擎** | **分层设计**: `LLM (基础)` → `RAG (可选知识)` → `Skills/MCP (可选工具)` → `Agent (统一接口)`。 |
| **📚 知识引擎** | **混合 RAG**: 结合了极速向量搜索与 **基于 SQLite 的 GraphRAG**，支持大规模 **批量 Embedding** 注入。 |
| **🛠️ 工具引擎** | 原生支持 **MCP (Model Context Protocol)**、**[Claude 兼容技能](SKILLS.md)**，以及 **WebSocket 实时有状态交互**。 |
| **💾 记忆系统** | **混合存储**: 支持高性能 SQLite 或 **极致透明的 Markdown 文件**。具备自反思 (Reflection) 和智能融合能力。 |
| **⚡ 运行时** | **确定性工作流**: 独创 **规划 (Plan) -> 审核 (Review) -> 执行 (Execute)**，彻底消除 Agent 的黑盒行为。 |
| **🔒 本地优先** | 设计为既可离线运行（**Ollama**），也可连接云端。数据完全受物理隔离保护。 |

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
        Name:            "my-assistant",
        EnableMCP:       true, 
        EnableMemory:    true,
        MemoryStoreType: "file", // 使用透明的 Markdown 文件记忆
    })
    defer svc.Close()

    // 2. 运行任务
    res, _ := svc.Run(ctx, "研究 Go 1.24 的最新特性并写一份总结存入记忆。")
    fmt.Println(res.FinalResult)
}
```

---

## 🏗️ 架构与特性深度解析

### 1. 分层化设计与可选组件
RAGO 采用严格的层级依赖，允许开发者根据需求“修剪”系统：
*   **LLM 层**: 统一封装 OpenAI, Ollama, DeepSeek 等模型。
*   **RAG 层 (可选)**: 注入领域知识。
*   **Skills/MCP 层 (可选)**: 赋予执行动作的能力。
*   **Agent 层**: 作为大脑和入口，负责意图识别与任务编排。

### 2. 透明文件记忆与智能融合
记忆不再是黑盒。RAGO 允许将长期记忆存储为人类可读的 **Markdown + YAML** 文件。
*   **零向量路由**: 自动生成“记忆地图”，无需 Embedding 即可精准定位事实。
*   **智能融合 (Fusion)**: 当记忆更新时，Agent 会自动合并新旧信息，保证知识的连续生长。

### 3. 原生实时交互 (WebSocket)
基于 OpenAI 最新的 Responses API，RAGO 内置了 WebSocket 会话支持。
*   **毫秒级响应**: 维持长连接，显著降低多轮对话和工具调用的延迟。
*   **有状态会话**: 服务器端维持上下文，极大地节省带宽。

### 4. 混合 RAG (向量 + 知识图谱)
RAGO 不仅仅存储向量，它还会自动构建 **知识图谱**。
*   **向量搜索**: 用于语义相似度匹配。
*   **GraphRAG**: 用于发现实体间的隐式关系。
*   **批量处理**: 支持海量文档的高并发批量 Embedding 注入。

### 5. 确定性规划流 (Plan-Review-Execute)
为了生产环境的安全性，RAGO 支持显式的规划流程。
```go
// 1. 生成计划 (Agent 输出可读的步骤)
plan, _ := svc.Plan(ctx, "部署一个新的 Web 服务")

// 2. 人类在 CLI 或 UI 进行审核...

// 3. 确认后执行
result, _ := svc.Execute(ctx, plan.ID)
```

---

## 💻 CLI 使用

RAGO 提供了一个强大的 CLI 来管理整个生命周期。

```bash
# 启动任务
rago agent run "查找当前目录下的重复文件并清理"

# 管理 RAG 知识库
rago rag ingest ./docs/ --recursive
rago rag query "如何配置服务器端口？"

# 管理 MCP 工具
rago mcp list
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
model = "gpt-4o"
```

## 📚 示例代码

请查看 `examples/` 目录获取深入的示例：
*   **[04_file_memory_test](./examples/agent_usage/04_file_memory_test/)**: 演示透明文件记忆的全流程。
*   **[realtime_chat](./examples/realtime_chat/)**: WebSocket 实时对话演示。
*   **[multi_agent_orchestration](./examples/multi_agent_orchestration/)**: 包含 Handoffs 和流式输出的完整演示。

## 📄 许可证
MIT License - Copyright (c) 2024-2026 RAGO Authors.
