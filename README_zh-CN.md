# RAGO (Radical Agentic Go)

RAGO 是一个极简、模块化的 Go 语言 Agent 框架。它专为本地优先、高性能且极致透明的 AI 工作流而设计。

## 为什么选择 RAGO?

RAGO 采用 **分层架构** 设计，除了 LLM 基础层外，所有组件（RAG、工具、记忆）均为可选。我们拒绝黑盒，主张一切皆可控。

### 🌟 核心特质

- **层级架构**: `LLM (基础)` → `RAG (可选知识)` → `Skills/MCP (可选工具)` → `Agent (统一接口)`。按需组合，灵活轻量。
- **透明文件记忆**: 长期记忆以 **Markdown + YAML** 格式存储在本地。看得见、摸得着、手动能改。支持**零向量 (Zero-Embedding)** 检索，通过记忆地图自动路由。
- **原生实时交互**: 内置 WebSocket 支持（基于 OpenAI 最新 Responses API），实现毫秒级响应和有状态的长连接对话。
- **确定性流程**: 独创 **规划 (Plan) -> 审核 (Review) -> 执行 (Execute)** 工作流，确保 Agent 的每一步动作都在人类掌控之中。
- **本地优先**: 基于 SQLite (`sqvect`) 和本地文件系统，数据隐私受物理隔离保护。

---

## 快速开始

### 1. 创建一个带文件记忆的 Agent
```go
svc, _ := agent.New(&agent.AgentConfig{
    Name:            "小智",
    EnableMemory:    true,
    MemoryStoreType: "file", // 开启透明文件存储
})

// 执行任务并存入记忆
res, _ := svc.Run(ctx, "研究 RAGO 的架构并总结存入记忆。")
fmt.Println(res.FinalResult)
```

### 2. 规划与执行流
```go
// 1. 生成计划（不立即执行）
plan, _ := svc.Plan(ctx, "用 Go 语言写一个文件加密工具")

// 2. 人类可以在 UI 或终端查看步骤...

// 3. 确认无误后执行
result, _ := svc.Execute(ctx, plan.ID)
```

## 安装

```bash
go get github.com/liliang-cn/rago/v2
```

## 架构层级

1.  **LLM 层**: 统一封装 OpenAI, Ollama 等模型。
2.  **知识层 (RAG)**: 基于 `sqvect` 的高性能向量 + 知识图谱检索。
3.  **能力层 (MCP/Skills)**: 通过 Model Context Protocol 扩展 Agent 的动作边界。
4.  **大脑 (Agent)**: 负责意图识别、任务编排、状态持久化。

---
License: MIT
