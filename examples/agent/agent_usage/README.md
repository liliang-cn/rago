# Agent 使用示例

这些示例展示了 RAGO Agent 的不同使用场景和层级能力。

## 示例列表

1. **[01_simple_chat](./01_simple_chat/main.go)**: 基础对话与持久化。
   - 展示 `Chat()` API。
   - 展示如何通过 SQLite 自动维护会话历史和记忆。
   - 演示 Agent 如何在不手动管理 ID 的情况下记住用户信息。

2. **[02_with_tools](./02_with_tools/main.go)**: 工具化 Agent (Tool-use)。
   - 展示 `Run()` API。
   - 集成 MCP、Skills 和 RAG。
   - 演示进度回调 (Progress Callback) 以监控 Agent 的工具调用过程。

3. **[03_planning_flow](./03_planning_flow/main.go)**: 显式规划与执行 (Planning)。
   - 展示 `Plan()`, `RevisePlan()` 和 `ExecutePlan()` API。
   - 演示如何先让 Agent 生成可读的步骤，经过人工确认（或修改）后再执行。
   - 适用于需要高可靠性和人工干预的复杂工作流。

## 层级结构说明

RAGO 的设计遵循以下层级，Agent 是最上层的统一接口：

1. **LLM (基础)**: 处理文本生成和推理。
2. **RAG (可选部件)**: 为 Agent 提供领域知识检索。
3. **Skills/MCP (可选部件)**: 为 Agent 提供执行动作的能力。
4. **Agent (入口)**: 负责意图识别、任务规划、工具调度和状态持久化。

## 如何运行

确保你已经根据项目根目录的 `rago.toml.example` 配置好了环境变量（如 `OPENAI_API_KEY`）。

```bash
# 运行示例 01
go run examples/agent_usage/01_simple_chat/main.go

# 运行示例 02
go run examples/agent_usage/02_with_tools/main.go

# 运行示例 03
go run examples/agent_usage/03_planning_flow/main.go
```
