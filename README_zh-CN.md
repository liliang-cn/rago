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

RAGO 由四大支柱构成，共同支撑起应用程序的智能层。

### 1. 统一 LLM 支持
为多个供应商（Ollama, OpenAI, DeepSeek 等）提供统一接口，内置负载均衡与容错机制。

### 2. 混合 RAG (向量 + 知识图谱)
结合了极速向量相似度匹配与基于 SQLite 的 **知识图谱 (GraphRAG)**，实现深度的关联检索。

```go
// 开启 GraphRAG 以提取复杂的实体关系
opts := &rag.IngestOptions{ EnhancedExtraction: true }
client.IngestFile(ctx, "data.pdf", opts)

// 使用混合搜索 (Hybrid Search) 进行查询
resp, _ := client.Query(ctx, "分析数据中的潜在关联", nil)
```

### 3. MCP 与 Claude 兼容的技能
通过 **Model Context Protocol** 和 **Claude 兼容的技能** 极大地扩展 Agent 的能力边界。

```go
// 通过 Markdown 技能和 MCP 服务器添加专家能力
svc, _ := agent.New(&agent.AgentConfig{
    EnableMCP:    true, // 连接外部工具
    EnableSkills: true, // 从 .skills/ 加载 Claude 技能
})
```

### 4. Hindsight：自验证与反思
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

## 📚 文档与示例

*   **[Agent 技能集成](./examples/skills_integration/)**: 连接自定义工具。
*   **[会话压缩](./examples/compact_session/)**: 管理 Agent 的长期上下文。
*   **[混合 RAG 系统](./examples/advanced_library_usage/)**: 构建知识库。
*   **[快速入门指南](./examples/quickstart/)**: Go 应用的基本配置。

## 📄 许可证
MIT License - Copyright (c) 2024-2025 RAGO Authors.