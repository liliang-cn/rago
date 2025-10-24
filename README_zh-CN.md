# RAGO - 简化本地 RAG 系统

[English Documentation](README.md)

RAGO (Retrieval-Augmented Generation Offline) v2 是一个流线型、生产就绪的 RAG 系统，采用 Go 语言编写。它提供简洁的 API 用于文档摄取、语义搜索和上下文增强问答，专注于简单性和可靠性。

## 🌟 核心功能（v2 简化版）

### 📚 **RAG 系统（核心功能）**
- **文档摄取** - 导入文本、Markdown、PDF 文件并自动分块
- **向量数据库** - 基于 SQLite 的向量存储，使用 sqvect 实现高性能搜索
- **语义搜索** - 使用嵌入相似性查找相关文档
- **智能分块** - 可配置的文本分割（句子、段落、词元方法）
- **问答生成** - 使用检索文档进行上下文增强回答
- **元数据提取** - 自动生成文档的关键词和摘要

### 🔧 **OpenAI 兼容 LLM 支持**
- **统一提供商接口** - 所有 LLM 服务使用单一 OpenAI 兼容 API
- **本地优先** - 支持 Ollama、LM Studio 和任何 OpenAI 兼容服务器
- **流式支持** - 实时令牌流式传输以获得更好的用户体验
- **结构化生成** - 生成符合特定模式的 JSON 输出
- **健康监控** - 内置提供商健康检查

### 🛠️ **MCP 工具集成**
- **模型上下文协议** - 标准工具集成框架
- **内置工具** - filesystem、fetch、memory、time、sequential-thinking
- **外部服务器** - 连接任何 MCP 兼容的工具服务器
- **查询增强** - 在 RAG 查询期间使用工具获得更丰富的答案

### 💻 **开发者体验**
- **简洁库 API** - 所有操作的简单、直观接口
- **零配置模式** - 使用智能默认值开箱即用
- **HTTP API** - 所有操作的 RESTful 端点
- **高性能** - 优化的 Go 实现，依赖最少

## 🚀 快速开始（零配置！）

**✨ RAGO v2 无需任何配置即可运行！**

### 安装

```bash
# 选项 1：直接安装
go install github.com/liliang-cn/rago/v2@latest

# 选项 2：克隆并构建
git clone https://github.com/liliang-cn/rago.git
cd rago
go build -o rago ./cmd/rago-cli

# 选项 3：使用 Makefile
make build
```

### 🎯 零配置使用

RAGO v2 开箱即用地支持 OpenAI 兼容的提供商：

```bash
# 检查系统状态（无需配置！）
./rago status

# 将文档导入 RAG 知识库
./rago rag ingest document.pdf
./rago rag ingest "path/to/text/file.txt"
./rago rag ingest --text "直接文本内容" --source "我的文档"

# 查询您的知识库
./rago rag query "这个文档是关于什么的？"

# 列出所有已索引的文档
./rago rag list

# 交互模式（如果可用）
./rago rag query -i

# 启用 MCP 工具（如果可用）
./rago rag query "分析这些数据并保存结果" --mcp
```

### 环境变量（可选）

```bash
# 用于 OpenAI 兼容服务
# API 密钥是可选的 - 由提供商处理认证
export RAGO_OPENAI_API_KEY="your-api-key"  # 可选
export RAGO_OPENAI_BASE_URL="http://localhost:11434/v1"  # Ollama
export RAGO_OPENAI_LLM_MODEL="qwen3"
export RAGO_OPENAI_EMBEDDING_MODEL="nomic-embed-text"
```


## 📖 库使用

### RAG 客户端 API（推荐）

简化的 RAG 客户端为所有操作提供清晰的接口：

```go
import (
    "context"
    "fmt"
    "github.com/liliang-cn/rago/v2/pkg/rag"
    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/providers"
)

// 使用默认配置初始化
cfg, _ := config.Load("")  // 空字符串表示默认值
cfg.Providers.DefaultLLM = "openai"
cfg.Providers.OpenAI.BaseURL = "http://localhost:11434/v1"  // Ollama
cfg.Providers.OpenAI.LLMModel = "qwen3"
cfg.Providers.OpenAI.EmbeddingModel = "nomic-embed-text"

// 创建提供商
embedder, _ := providers.CreateEmbedderProvider(context.Background(), cfg.Providers.OpenAI)
llm, _ := providers.CreateLLMProvider(context.Background(), cfg.Providers.OpenAI)

// 创建 RAG 客户端
client, _ := rag.NewClient(cfg, embedder, llm, nil)
defer client.Close()

// 摄取文档
ctx := context.Background()
resp, err := client.IngestFile(ctx, "document.pdf", rag.DefaultIngestOptions())
fmt.Printf("已摄取 %d 个块\n", resp.ChunkCount)

// 查询知识库
queryResp, err := client.Query(ctx, "这个文档是关于什么的？", rag.DefaultQueryOptions())
fmt.Printf("回答: %s\n", queryResp.Answer)
fmt.Printf("来源: %d\n", len(queryResp.Sources))

// 直接摄取文本
textResp, err := client.IngestText(ctx, "您的文本内容", "source.txt", rag.DefaultIngestOptions())
fmt.Printf("文本已摄取，ID: %s\n", textResp.DocumentID)
```

### LLM 服务 API

用于直接 LLM 操作：

```go
import (
    "context"
    "github.com/liliang-cn/rago/v2/pkg/llm"
    "github.com/liliang-cn/rago/v2/pkg/domain"
)

// 创建 LLM 服务
llmService := llm.NewService(llmProvider)

// 简单生成
response, err := llmService.Generate(ctx, "写一首俳句", &domain.GenerationOptions{
    Temperature: 0.7,
    MaxTokens:   100,
})

// 流式生成
err = llmService.Stream(ctx, "给我讲个故事", &domain.GenerationOptions{
    Temperature: 0.8,
    MaxTokens:   500,
}, func(chunk string) {
    fmt.Print(chunk)
})

// 工具调用
messages := []domain.Message{
    {Role: "user", Content: "今天天气怎么样？"},
}
tools := []domain.ToolDefinition{
    // 在这里定义您的工具
}
result, err := llmService.GenerateWithTools(ctx, messages, tools, &domain.GenerationOptions{})
```

### 基于配置的使用

创建 `rago.toml` 配置文件：

```toml
[providers]
default_llm = "openai"
default_embedder = "openai"

[providers.openai]
type = "openai"
base_url = "http://localhost:11434/v1"  # Ollama 端点
api_key = "ollama"  # 即使本地也需要
llm_model = "qwen3"
embedding_model = "nomic-embed-text"
timeout = "30s"

[sqvect]
db_path = "~/.rago/data/rag.db"
top_k = 5
threshold = 0.0

[chunker]
chunk_size = 500
overlap = 50
method = "sentence"

[mcp]
enabled = true
```

然后在代码中使用：

```go
cfg, _ := config.Load("rago.toml")
// ... 其余初始化代码
```

## 🛠️ MCP 工具

### 内置工具

- **filesystem** - 文件操作（读、写、列表、执行）
- **fetch** - HTTP/HTTPS 请求
- **memory** - 临时键值存储
- **time** - 日期/时间操作
- **sequential-thinking** - LLM 分析和推理

### 工具配置

在 `mcpServers.json` 中配置 MCP 服务器：

```json
{
  "filesystem": {
    "command": "npx",
    "args": ["@modelcontextprotocol/server-filesystem", "/path/to/allowed/directory"],
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
- `POST /api/rag/ingest` - 将文档摄取到向量数据库
- `POST /api/rag/query` - 执行带上下文检索的 RAG 查询
- `GET /api/rag/list` - 列出已索引的文档
- `DELETE /api/rag/reset` - 清空向量数据库
- `GET /api/rag/collections` - 列出所有集合

#### MCP 工具
- `GET /api/mcp/tools` - 列出可用的 MCP 工具
- `POST /api/mcp/tools/call` - 执行 MCP 工具
- `GET /api/mcp/status` - 检查 MCP 服务器状态

## ⚙️ 配置

### 环境变量（简单）

```bash
# 基本 OpenAI 兼容配置
export RAGO_OPENAI_API_KEY="your-api-key"
export RAGO_OPENAI_BASE_URL="http://localhost:11434/v1"
export RAGO_OPENAI_LLM_MODEL="qwen3"
export RAGO_OPENAI_EMBEDDING_MODEL="nomic-embed-text"

# 服务器设置
export RAGO_SERVER_PORT="7127"
export RAGO_SERVER_HOST="0.0.0.0"
```

### 配置文件（高级）

创建 `rago.toml` 进行完全控制：

```toml
[server]
port = 7127
host = "0.0.0.0"
enable_ui = false

[providers]
default_llm = "openai"
default_embedder = "openai"

[providers.openai]
type = "openai"
base_url = "http://localhost:11434/v1"  # Ollama 端点
api_key = "ollama"
llm_model = "qwen3"
embedding_model = "nomic-embed-text"
timeout = "30s"

[sqvect]
db_path = "~/.rago/data/rag.db"
top_k = 5
threshold = 0.0

[chunker]
chunk_size = 500
overlap = 50
method = "sentence"

[mcp]
enabled = true
servers_config_path = "mcpServers.json"
```

## 📚 文档

### API 参考
- **[RAG 客户端 API](./pkg/rag/)** - 核心 RAG 客户端文档
- **[LLM 服务 API](./pkg/llm/)** - LLM 服务文档
- **[配置指南](./pkg/config/)** - 完整配置选项
- **[English Docs](./README.md)** - 英文文档

### 示例（即将推出）
我们正在更新简化 v2 API 的示例。敬请期待：
- 基本 RAG 客户端使用
- LLM 服务示例
- MCP 工具集成
- 配置模式

## 🤝 贡献

欢迎贡献！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解指南。

## 📄 许可证

MIT 许可证 - 详情请见 [LICENSE](LICENSE)

## 🔗 链接

- [GitHub 仓库](https://github.com/liliang-cn/rago)
- [问题跟踪](https://github.com/liliang-cn/rago/issues)
- [讨论区](https://github.com/liliang-cn/rago/discussions)
