# 🗓️ Rago 任务调度系统

## 概述

Rago 任务调度系统提供了完整的定时任务管理功能，支持多种任务类型的创建、调度和执行。

## 🚀 功能特性

### 支持的任务类型

1. **脚本任务 (Script)** - 执行 Shell 脚本或命令
2. **查询任务 (Query)** - 执行 RAG 查询
3. **摄取任务 (Ingest)** - 批量导入文档
4. **MCP 任务 (MCP)** - 调用 MCP 工具

### 调度功能

- **Cron 表达式支持**: 标准 cron 格式 (分 时 日 月 周)
- **预定义表达式**: `@daily`, `@weekly`, `@monthly`, `@hourly`
- **一次性任务**: 立即执行或延迟执行
- **任务管理**: 启用/禁用、删除、更新

## 📋 CLI 命令

### 基本操作

```bash
# 创建脚本任务
rago task create --type script --desc "备份任务" --param "script=tar -czf backup.tar.gz /data" --schedule "@daily"

# 创建查询任务
rago task create --type query --desc "日报生成" --param "query=今日要点总结" --schedule "0 9 * * *"

# 创建摄取任务
rago task create --type ingest --desc "文档导入" --param "path=./docs" --schedule "0 0 * * 0"

# 创建MCP任务
rago task create --type mcp --desc "数据获取" --param "tool=fetch" --param "url=https://api.example.com" --schedule "0 */6 * * *"
```

### 任务管理

```bash
# 列出所有任务
rago task list

# 显示任务详情
rago task show <task-id>

# 立即运行任务
rago task run <task-id>

# 启用/禁用任务
rago task enable <task-id>
rago task disable <task-id>

# 删除任务
rago task delete <task-id>
```

## 📚 程序化 API

### Go 库使用

```go
package main

import (
    "context"
    "github.com/liliang-cn/rago/lib"
)

func main() {
    // 初始化客户端
    client, err := rago.New("config.toml")
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // 启用任务调度
    ctx := context.Background()
    err = client.EnableTasks(ctx)
    if err != nil {
        panic(err)
    }

    // 创建脚本任务
    taskID, err := client.CreateScriptTask(
        "echo 'Hello World'",
        "@daily",
        map[string]string{"workdir": "/tmp"},
    )

    // 创建查询任务
    queryTaskID, err := client.CreateQueryTask(
        "什么是人工智能？",
        "@daily",
        map[string]string{"top-k": "3"},
    )

    // 列出所有任务
    tasks, err := client.ListTasks(false)

    // 运行任务
    result, err := client.RunTask(taskID)
}
```

## 🏗️ 系统架构

### 核心组件

- **调度器 (Scheduler)**: 基于 robfig/cron/v3 的任务调度引擎
- **执行器 (Executors)**: 模块化的任务执行器
- **存储层 (Storage)**: SQLite 数据库持久化
- **CLI 接口**: 完整的命令行管理工具
- **库接口**: Go 程序化 API

### 执行器架构

```
scheduler.Executor (接口)
├── ScriptExecutor    - Shell 脚本执行
├── QueryExecutor     - RAG 查询执行
├── IngestExecutor    - 文档摄取执行
└── MCPExecutor       - MCP 工具调用
```

## 📊 任务输出格式

### 脚本任务输出

```json
{
  "script": "echo 'Hello World'",
  "shell": "/bin/sh",
  "workdir": "/tmp",
  "output": "Hello World\n",
  "duration": "4.5ms",
  "success": true
}
```

### 查询任务输出

```json
{
  "query": "什么是人工智能？",
  "response": "人工智能是...",
  "sources": ["doc1.pdf", "doc2.md"],
  "used_mcp": false
}
```

## 🔧 配置示例

```toml
# config.toml
[providers]
default_llm = "ollama"
default_embedder = "ollama"

[sqvect]
db_path = "./data/rag.db"

[keyword]
index_path = "./data/keyword.bleve"
```

## 📁 示例代码

查看 `examples/task_scheduling/main.go` 获取完整的使用示例。

## 🎯 使用场景

1. **自动化备份**: 定期备份重要数据
2. **内容生成**: 定时生成报告和摘要
3. **数据同步**: 定期同步外部数据源
4. **文档处理**: 批量处理和索引文档
5. **监控报警**: 定期检查系统状态

## ✅ 已完成功能

- ✅ 完整的任务调度系统
- ✅ 四种任务执行器 (Script, Query, Ingest, MCP)
- ✅ CLI 命令行界面
- ✅ Go 库 API
- ✅ SQLite 持久化存储
- ✅ Cron 表达式支持
- ✅ 并发执行控制
- ✅ 任务历史跟踪

## 🚀 快速开始

1. 构建项目: `go build -o rago main.go`
2. 创建任务: `./rago task create --type script --desc "测试" --param "script=echo hello"`
3. 查看任务: `./rago task list`
4. 运行示例: `cd examples/task_scheduling && go run main.go`

任务调度系统现已完全可用，支持生产环境部署！🎉
