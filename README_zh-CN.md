# RAGO - 高级 RAG 系统与智能代理自动化

[English Documentation](README.md)

RAGO (Retrieval-Augmented Generation Offline) 是一个强大的本地 RAG 系统，具备智能代理自动化能力，支持自然语言工作流生成、MCP 工具集成和多提供商 LLM 支持。

## 🌟 核心功能

### 🤖 **智能代理自动化**
- **自然语言 → 工作流** - 将自然语言请求转换为可执行工作流
- **异步执行** - 支持并行步骤执行和依赖解析
- **MCP 工具集成** - 内置文件系统、网络、内存、时间和 LLM 推理工具

### 📚 **高级 RAG 系统**
- **多提供商支持** - 无缝切换 Ollama、OpenAI 和 LM Studio
- **向量搜索** - 基于 SQLite 向量数据库的高性能语义搜索
- **智能分块** - 可配置策略的智能文档处理

### ⚡ **工作流自动化**
- **JSON 工作流规范** - 以编程方式定义复杂工作流
- **变量传递** - 使用 `{{variable}}` 语法在步骤间传递数据
- **工具编排** - 协调多个 MCP 工具

### 🔧 **企业就绪**
- **HTTP APIs** - 完整的 REST API
- **100% 本地选项** - 使用本地 LLM 提供商实现完全隐私
- **高性能** - 优化的 Go 实现

## 🚀 快速开始

### 先决条件

1. **安装 Go** (≥ 1.21)
2. **选择你的 LLM 提供商**：
   - **Ollama** (本地): `curl -fsSL https://ollama.com/install.sh | sh`
   - **LM Studio** (本地): 从 [lmstudio.ai](https://lmstudio.ai) 下载
   - **OpenAI** (云端): 从 [platform.openai.com](https://platform.openai.com) 获取 API 密钥

### 安装

```bash
# 克隆并构建
git clone https://github.com/liliang-cn/rago.git
cd rago
go build -o rago ./cmd/rago

# 初始化配置
./rago init
```

### 🎯 代理示例

```bash
# 自然语言转工作流
./rago agent run "获取当前时间并告诉我是早上还是晚上"

# GitHub 集成
./rago agent run "获取 golang/go 仓库的信息"

# 复杂工作流
./rago agent run "获取旧金山的天气并分析是否适合户外活动"

# 保存工作流
./rago agent run "监控 github.com/golang/go 的新版本发布" --save
```

## 📖 库使用

在你的应用程序中使用 RAGO 作为 Go 库：

```go
import (
    "github.com/liliang-cn/rago/v2/pkg/agents/execution"
    "github.com/liliang-cn/rago/v2/pkg/agents/types"
    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/utils"
)

// 加载配置并初始化
cfg, _ := config.Load("")
ctx := context.Background()
_, llmService, _, _ := utils.InitializeProviders(ctx, cfg)

// 定义工作流
workflow := &types.WorkflowSpec{
    Steps: []types.WorkflowStep{
        {
            ID:   "fetch",
            Tool: "fetch",
            Inputs: map[string]interface{}{
                "url": "https://api.github.com/repos/golang/go",
            },
            Outputs: map[string]string{"data": "result"},
        },
        {
            ID:   "analyze",
            Tool: "sequential-thinking",
            Inputs: map[string]interface{}{
                "prompt": "分析这些数据",
                "data":   "{{result}}",
            },
        },
    },
}

// 执行
executor := execution.NewWorkflowExecutor(cfg, llmService)
result, _ := executor.Execute(ctx, workflow)
```

## 🛠️ MCP 工具

### 内置工具

- **filesystem** - 文件操作（读、写、列表、执行）
- **fetch** - HTTP/HTTPS 请求
- **memory** - 临时键值存储
- **time** - 日期/时间操作
- **sequential-thinking** - LLM 分析和推理
- **playwright** - 浏览器自动化（可选）

### 工具配置

在 `mcpServers.json` 中配置 MCP 服务器：

```json
{
  "filesystem": {
    "command": "npx",
    "args": ["@modelcontextprotocol/server-filesystem"],
    "description": "文件系统操作"
  },
  "fetch": {
    "command": "npx",
    "args": ["@modelcontextprotocol/server-fetch"],
    "description": "HTTP/HTTPS 操作"
  }
}
```

## 📊 HTTP API

启动 API 服务器：

```bash
./rago serve --port 7127
```

### 核心端点

- `POST /api/ingest` - 摄取文档
- `POST /api/query` - RAG 查询
- `GET /api/mcp/tools` - 列出 MCP 工具
- `POST /api/mcp/tools/call` - 执行 MCP 工具
- `POST /api/agent/run` - 运行自然语言工作流


## ⚙️ 配置

在 `rago.toml` 中配置提供商：

```toml
[providers]
default_llm = "lmstudio"  # 或 "ollama", "openai"
default_embedder = "lmstudio"

[providers.lmstudio]
type = "lmstudio"
base_url = "http://localhost:1234/v1"
llm_model = "qwen/qwen3-4b"
embedding_model = "nomic-embed-text"

[providers.ollama]
type = "ollama"
base_url = "http://localhost:11434"
llm_model = "llama3"
embedding_model = "nomic-embed-text"
```

## 📚 文档

- [示例](./examples/) - 代码示例和用例
- [API 参考](./docs/api.md) - HTTP API 文档
- [配置指南](./rago.example.toml) - 完整配置选项
- [English Docs](./README.md) - 英文文档

## 🤝 贡献

欢迎贡献！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解指南。

## 📄 许可证

MIT 许可证 - 详情请见 [LICENSE](LICENSE)

## 🔗 链接

- [GitHub 仓库](https://github.com/liliang-cn/rago)
- [问题跟踪](https://github.com/liliang-cn/rago/issues)
- [讨论区](https://github.com/liliang-cn/rago/discussions)
