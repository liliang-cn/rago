# RAGO - 本地 RAG 系统与智能代理自动化

[English Documentation](README.md)

RAGO (Retrieval-Augmented Generation Offline) 是一个完全本地的 RAG 系统，采用 Go 语言编写，集成 SQLite 向量数据库和多提供商 LLM 支持，用于文档摄取、语义搜索和上下文增强问答。

## 🌟 核心功能

### 📚 **RAG 系统（核心功能）**
- **文档摄取** - 导入文本、Markdown、PDF 文件并自动分块
- **向量数据库** - 基于 SQLite 的向量存储，使用 sqvect 实现高性能搜索
- **语义搜索** - 使用嵌入相似性查找相关文档
- **智能分块** - 可配置的文本分割（句子、段落、词元方法）
- **问答生成** - 使用检索文档进行上下文增强回答

### 🔧 **多提供商 LLM 支持**
- **Ollama 集成** - 使用 ollama-go 客户端进行本地 LLM 推理
- **OpenAI 兼容** - 支持 OpenAI API 和兼容服务
- **LM Studio** - 通过 LM Studio 集成进行本地模型服务
- **提供商切换** - 通过配置轻松切换不同提供商

### 🛠️ **MCP 工具集成**
- **模型上下文协议** - 标准工具集成框架
- **内置工具** - filesystem、fetch、memory、time、sequential-thinking
- **外部服务器** - 连接任何 MCP 兼容的工具服务器
- **查询增强** - 在 RAG 查询期间使用工具获得更丰富的答案

### 🤖 **智能代理自动化**
- **自然语言工作流** - 从纯文本描述生成工作流
- **MCP 工具编排** - 在自动化工作流中协调多个工具
- **异步执行** - 支持依赖解析的并行步骤执行

### 🏢 **生产就绪**
- **100% 本地** - 使用本地提供商完全离线操作
- **HTTP API** - 所有操作的 RESTful 端点
- **高性能** - 优化的 Go 实现
- **可配置** - 通过 TOML 进行广泛配置

## 🚀 快速开始（零配置！）

**✨ 新功能：RAGO 无需任何配置即可运行！**

### 30秒快速设置

```bash
# 1. 安装 RAGO
go install github.com/liliang-cn/rago/v2@latest

# 2. 安装 Ollama（如果尚未安装）
curl -fsSL https://ollama.com/install.sh | sh

# 3. 立即开始使用 RAGO！
rago status  # 无需配置文件即可工作！
```

就是这样！无需配置。RAGO 使用智能默认设置。

### 安装选项

```bash
# 克隆并构建
git clone https://github.com/liliang-cn/rago.git
cd rago
go build -o rago ./cmd/rago

# 可选：创建配置（仅在需要自定义设置时）
./rago init  # 交互式 - 选择"跳过"以零配置
```

### 🎯 零配置使用

```bash
# 拉取默认模型
ollama pull qwen3              # 默认 LLM
ollama pull nomic-embed-text   # 默认嵌入器

# 无需配置即可工作！
./rago status                  # 检查提供商状态
./rago ingest document.pdf     # 导入文档
./rago query "这是关于什么的？"  # 查询知识库
```

### 🎯 RAG 示例

```bash
# 导入更多文档
./rago ingest ./docs --recursive

# 查询您的文档
./rago query "主要概念是什么？"
./rago query "如何配置系统？" --show-sources

# 交互模式
./rago query -i

# 使用 MCP 工具
./rago query "分析这些数据并保存结果" --mcp
```

### 🤖 代理示例

```bash
# 自然语言工作流
./rago agent run "获取当前时间并告诉我是早上还是晚上"
./rago agent run "获取旧金山的天气并分析条件"

# 保存工作流以便重用
./rago agent run "监控 github.com/golang/go 的新版本发布" --save
```

## 📖 库使用

使用 RAGO 作为 Go 库进行 RAG 操作：

```go
import (
    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/store"
    "github.com/liliang-cn/rago/v2/pkg/processor"
)

// 初始化 RAGO
cfg, _ := config.Load("rago.toml")
store, _ := store.NewSQLiteStore(cfg.Sqvect.DBPath)
processor := processor.New(cfg, store)

// 摄取文档
doc := domain.Document{
    ID:      "doc1",
    Content: "您的文档内容",
    Path:    "/path/to/doc.txt",
}

err := processor.IngestDocument(ctx, doc)

// 查询文档
req := domain.QueryRequest{
    Query:       "这是关于什么的？",
    TopK:        5,
    Temperature: 0.7,
    MaxTokens:   500,
}

response, _ := processor.Query(ctx, req)
fmt.Println(response.Answer)
```

### 代理库使用

```go
import (
    "github.com/liliang-cn/rago/v2/pkg/agents/execution"
    "github.com/liliang-cn/rago/v2/pkg/agents/types"
)

// 定义工作流
workflow := &types.WorkflowSpec{
    Steps: []types.WorkflowStep{
        {
            ID:   "fetch",
            Tool: "fetch",
            Inputs: map[string]interface{}{
                "url": "https://api.github.com/repos/golang/go",
            },
        },
    },
}

// 执行工作流
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
- **playwright** - 浏览器自动化

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

#### RAG 操作
- `POST /api/ingest` - 将文档摄取到向量数据库
- `POST /api/query` - 执行带上下文检索的 RAG 查询
- `GET /api/list` - 列出已索引的文档
- `DELETE /api/reset` - 清空向量数据库

#### MCP 工具
- `GET /api/mcp/tools` - 列出可用的 MCP 工具
- `POST /api/mcp/tools/call` - 执行 MCP 工具
- `GET /api/mcp/status` - 检查 MCP 服务器状态

#### 智能代理自动化
- `POST /api/agent/run` - 生成并执行工作流
- `GET /api/agent/list` - 列出已保存的代理
- `POST /api/agent/create` - 创建新代理


## ⚙️ 配置

在 `rago.toml` 中配置提供商：

```toml
[providers]
default_llm = "lmstudio"  # 或 "ollama", "openai"
default_embedder = "lmstudio"

[providers.lmstudio]
type = "lmstudio"
base_url = "http://localhost:1234"
llm_model = "qwen/qwen3-4b-2507"
embedding_model = "text-embedding-qwen3-embedding-4b"
timeout = "120s"

[providers.ollama]
type = "ollama"
base_url = "http://localhost:11434"
llm_model = "qwen3"
embedding_model = "nomic-embed-text"
timeout = "120s"

# 向量数据库配置
[sqvect]
db_path = "~/.rago/rag.db"
top_k = 5
threshold = 0.0

# 文本分块配置
[chunker]
chunk_size = 500
overlap = 50
method = "sentence"

# MCP 工具配置
[mcp]
enabled = true
servers_config_path = "mcpServers.json"
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
