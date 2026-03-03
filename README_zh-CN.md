# RAGO

**本地优先的 RAG + Agent Go 框架。**

[English](README.md) · [API 参考](references/API.md) · [架构文档](references/ARCHITECTURE.md)

RAGO 是一个 Go 库，用于构建数据保持本地的 AI 应用。从文档语义搜索开始，按需添加 Agent 自动化能力。

```bash
go get github.com/liliang-cn/rago/v2
```

---

## 核心能力

| 能力 | 说明 |
|---|---|
| **RAG** | 文档摄入 → 分块 → Embedding → SQLite 向量存储 → 混合检索 |
| **Agent** | 多轮推理循环，支持工具调用、规划和会话记忆 |
| **Memory** | SQLite 向量记忆 + 语义检索；无 Embedder 时自动降级为按时间列表检索 |
| **工具** | MCP（模型上下文协议）、Skills（YAML+Markdown）、自定义内联工具 |
| **PTC** | LLM 写 JavaScript，工具在 Goja 沙箱中运行 — 减少模型往返次数 |
| **Streaming** | 逐 Token 通道输出；完整事件流可见工具调用过程 |
| **多 Provider** | OpenAI、Anthropic、Azure、DeepSeek、Ollama — 运行时可切换 |

---

## 快速开始

### 简单问答

```go
svc, _ := agent.New("assistant").
    WithPrompt("你是一个智能助手。").
    Build()
defer svc.Close()

reply, _ := svc.Ask(ctx, "什么是 Go 语言？")
fmt.Println(reply)
```

### 接入 RAG（文档知识库）

```go
svc, _ := agent.New("assistant").
    WithPrompt("根据提供的文档回答问题。").
    WithRAG().
    WithDBPath("~/.rago/data/agent.db").
    Build()
defer svc.Close()

// 一次性摄入
svc.Run(ctx, "摄入 ./docs/ 目录")

// 查询
reply, _ := svc.Ask(ctx, "规范文档里关于错误处理是怎么说的？")
```

### 带记忆的多轮对话

```go
svc, _ := agent.New("assistant").
    WithMemory().
    Build()
defer svc.Close()

svc.Chat(ctx, "我叫 Alice，在支付团队工作。")
reply, _ := svc.Chat(ctx, "我在哪个团队？")
// → "你在支付团队，Alice。"
```

### 流式输出

```go
// Token 流
for token := range svc.Stream(ctx, "解释一下 Go 的接口") {
    fmt.Print(token)
}

// 完整事件流（工具调用、来源、错误）
events, _ := svc.RunStream(ctx, "搜索文档并总结")
for e := range events {
    switch e.Type {
    case agent.EventTypePartial:  fmt.Print(e.Content)
    case agent.EventTypeToolCall: fmt.Printf("[工具: %s]\n", e.ToolName)
    case agent.EventTypeComplete: fmt.Println("\n完成")
    }
}
```

---

## Builder

所有选项都通过 `agent.New()` 的方法链配置：

```go
svc, err := agent.New("my-agent").
    // 能力模块
    WithRAG().                                      // rag_query + rag_ingest 工具
    WithMemory().                                   // memory_save + memory_recall 工具
    WithMCP().                                      // MCP 工具服务器
    WithSkills(agent.WithSkillsPaths("./skills")).  // Skill 文件
    WithPTC().                                      // JS 沙箱工具传输
    // 配置
    WithPrompt("你是一个智能助手。").
    WithDBPath("~/.rago/data/agent.db").
    WithDebug(true).
    // 自定义工具
    WithTool(myDef, myHandler, "category").
    // 回调
    WithProgress(func(e *agent.ProgressEvent) { fmt.Println(e.Text) }).
    Build()
```

### Module 系统

能力模块通过 `Module` 接口自我注册工具：

```go
// 实现自己的 Module
type Module interface {
    ID() string
    RegisterTools(registry *ToolRegistry) error
}

svc, _ := agent.New("agent").
    WithModule(NewMyCustomModule()).
    Build()
```

---

## 调用 API

| 方法 | 返回值 | 会话 | 适用场景 |
|---|---|---|---|
| `Ask(ctx, prompt)` | `(string, error)` | 无 | 单次问答 |
| `Chat(ctx, prompt)` | `(*ExecutionResult, error)` | 有（自动 UUID） | 对话式交互 |
| `Run(ctx, goal, ...opts)` | `(*ExecutionResult, error)` | 可选 | 完整 Agent 循环 |
| `Stream(ctx, prompt)` | `<-chan string` | 无 | 实时 Token 输出 |
| `ChatStream(ctx, prompt)` | `<-chan string` | 有 | 对话式 + 实时输出 |
| `RunStream(ctx, goal)` | `(<-chan *Event, error)` | 可选 | 完整事件可见性 |

```go
result, _ := svc.Run(ctx, "目标",
    agent.WithMaxTurns(20),
    agent.WithTemperature(0.7),
    agent.WithSessionID("my-session"),
    agent.WithStoreHistory(true),
)

result.Text()        // 最终回答（字符串）
result.Err()         // 非 nil 表示 Agent 报告了错误
result.HasSources()  // true 表示使用了 RAG 检索结果
```

---

## 程序化工具调用（PTC）

使用 `WithPTC()` 时，LLM 生成 JavaScript 代码而非 JSON 工具调用。代码在 Goja 沙箱中运行，可使用 `callTool()`：

```go
svc, _ := agent.New("analyst").
    WithPTC().
    WithTool(teamDef, teamHandler, "data").
    WithTool(expenseDef, expenseHandler, "data").
    Build()

// LLM 可以这样写代码：
//   const team = callTool("get_team", { dept: "eng" });
//   return team.members.map(m => ({
//     name: m.name,
//     spend: callTool("get_expenses", { id: m.id }).total
//   }));
```

**适用场景：** 多个依赖工具调用一次完成、数据进入上下文前先转换、需要条件逻辑的工具调用。

---

## 记忆系统

记忆分为两层：

| 层级 | 存储 | 用途 |
|---|---|---|
| **DB 记忆** | SQLite + 向量 | 自动学习的事实、对话历史、语义检索 |
| **文件记忆** | Markdown 文件 | 人类可编辑的人设：`SOUL.md`、`AGENTS.md`、`HEARTBEAT.md` |

```go
// 启用 DB 记忆（自动从每次对话中学习）
svc, _ := agent.New("agent").WithMemory().Build()

// LongRun Agent 自动共享同一个 DB 记忆
lr, _ := agent.NewLongRun(svc).
    WithInterval(5 * time.Minute).
    WithWorkDir("~/.rago/longrun").
    Build()
```

无 Embedder 时优雅降级：自动切换为按时间倒序的列表检索。

---

## 自主 Agent（LongRun）

LongRun 按计划运行 Agent，带有持久化任务队列：

```go
lr, _ := agent.NewLongRun(svc).
    WithInterval(10 * time.Minute).
    WithMaxActions(5).
    Build()

lr.AddTask(ctx, "监控 RSS 订阅并总结新条目", nil)
lr.Start(ctx)
// ...
lr.Stop()
```

特性：SQLite 任务队列、心跳文件、定时调度、与父 Agent 共享 DB 记忆。

---

## 多 Agent 协作

```go
// Handoffs — 专业 Agent 路由
orchestrator.RegisterAgent(researchAgent)
orchestrator.RegisterAgent(writerAgent)
// LLM 通过 transfer_to_* 工具调用路由到对应 Agent

// SubAgents — 作用域委托
coordinator := agent.NewSubAgentCoordinator()
resultChan  := coordinator.RunAsync(ctx, subAgent)
results     := coordinator.WaitAll(ctx)
```

---

## 确定性规划（Plan-Review-Execute）

```go
plan, _   := svc.Plan(ctx, "部署新服务")
// 检查 plan.Steps，按需修改
result, _ := svc.Execute(ctx, plan.ID)
```

---

## 配置与存储

配置文件：`rago.toml`（自动发现路径：`./` → `~/.rago/` → `~/.rago/config/`）

### 目录结构（默认 `home = ~/.rago`）

```
~/.rago/
├── rago.toml              ← 配置文件
├── mcpServers.json        ← MCP 服务器定义
├── data/
│   ├── rago.db            ← RAG 向量库（sqlite-vec）；memory.store_type=vector 时共用
│   ├── agent.db           ← Agent 会话 + 执行计划
│   └── memories/          ← Memory 文件存储（每个 session 一个 JSON）
├── skills/                ← SKILL.md 技能文件
├── intents/               ← Intent 意图文件
└── workspace/             ← Agent 工作目录
```

### SQLite 文件说明

| 文件 | 默认路径 | 用途 |
|------|---------|------|
| `rago.db` | `$home/data/rago.db` | RAG 文档 + 向量索引；`memory.store_type=vector` 时同时作为 Memory 向量库 |
| `agent.db` | `$home/data/agent.db` | Agent 会话消息历史和执行计划 |
| `history.db` *(可选)* | 通过 `WithHistoryDBPath()` 指定 | 详细工具调用日志，仅在 `WithStoreHistory(true)` 时创建 |

### Memory 存储类型

| `store_type` | 存储方式 | 是否需要 Embedder |
|-------------|---------|-----------------|
| `file` *(默认)* | `data/memories/{session}.json` | 否 |
| `vector` | `data/rago.db`（共用） | 是 |
| `hybrid` | 文件为主 + `rago.db` 影子索引 | 是 |

### 核心配置字段

```toml
home = "~/.rago"             # 所有相对路径的基准目录

[sqvect]
db_path   = ""               # RAG 数据库，默认 $home/data/rago.db
                             # 环境变量：RAGO_SQVECT_DB_PATH

[memory]
store_type  = "file"         # file | vector | hybrid
memory_path = ""             # 默认 $home/data/memories

[chunker]
chunk_size = 512
overlap    = 64
method     = "sentence"

[skills]
enabled   = true
auto_load = true

[mcp]
servers = ["~/.rago/mcpServers.json"]
```

完整带注释的配置参见 [`references/CONFIG.md`](references/CONFIG.md)。

---

## Provider 配置

`rago.toml` 自动发现路径：`./`、`~/.rago/`、`~/.rago/config/`

```toml
[[llm_pool.providers]]
name     = "openai"
provider = "openai"
api_key  = "sk-..."
model    = "gpt-4o"

[[llm_pool.providers]]
name     = "local"
provider = "ollama"
base_url = "http://localhost:11434"
model    = "qwen2.5:14b"
```

支持：OpenAI · Anthropic · Azure OpenAI · DeepSeek · Ollama（本地）

---

## 示例

```
examples/
├── quickstart/               — 最简入门示例
├── agent/
│   ├── agent_usage/          — Builder 模式、工具注册
│   ├── multi_agent_orchestration/ — Handoffs + 流式输出
│   ├── longrun/              — 自主定时 Agent
│   └── realtime_chat/        — WebSocket 会话
├── rag/                      — 文档摄入 + 问答
├── memory/
│   ├── chat_with_memory/     — DB 记忆 + 对话
│   └── smart_fusion/         — 记忆合并
├── ptc/
│   ├── custom_tools/         — JS 沙箱工具编排
│   └── memory_chat/          — PTC + 记忆
├── skills/                   — Skill 文件
└── mcp/                      — MCP 工具服务器
```

---

## License

MIT — Copyright (c) 2024–2026 RAGO Authors
