# RAGO - 本地 RAG 系统

[English Documentation](README.md)

RAGO (Retrieval-Augmented Generation Offline) 是一个完全本地的 RAG 系统，采用 Go 语言编写，集成 SQLite 向量数据库和多提供商 LLM 支持，用于文档摄取、语义搜索和上下文增强问答。

## 🌟 核心功能

### 📚 **RAG 系统（核心功能）**
- **文档摄取** - 导入文本、Markdown、PDF 文件并自动分块
- **向量数据库** - 基于 SQLite 的向量存储，使用 sqvect 实现高性能搜索
- **语义搜索** - 使用嵌入相似性查找相关文档
- **混合搜索** - 结合向量相似性和关键词匹配以获得更好的结果
- **智能分块** - 可配置的文本分割（句子、段落、词元方法）
- **问答生成** - 使用检索文档进行上下文增强回答
- **元数据提取** - 自动生成文档的关键词和摘要

### 🔧 **多提供商 LLM 支持**
- **Ollama 集成** - 使用 ollama-go 客户端进行本地 LLM 推理
- **OpenAI 兼容** - 支持 OpenAI API 和兼容服务
- **LM Studio** - 通过 LM Studio 集成进行本地模型服务
- **提供商切换** - 通过配置轻松切换不同提供商
- **流式支持** - 实时令牌流式传输以获得更好的用户体验
- **结构化生成** - 生成符合特定模式的 JSON 输出

### 🛠️ **MCP 工具集成**
- **模型上下文协议** - 标准工具集成框架
- **内置工具** - filesystem、fetch、memory、time、sequential-thinking
- **外部服务器** - 连接任何 MCP 兼容的工具服务器
- **查询增强** - 在 RAG 查询期间使用工具获得更丰富的答案
- **批量操作** - 并行执行多个工具调用


### 💻 **开发者体验**
- **简化的客户端 API** - 所有操作的清晰、直观的客户端包
- **全面的示例** - 常见用例的即用型示例
- **交互模式** - 内置 REPL 用于测试和探索
- **聊天历史管理** - 完整的对话跟踪和上下文保留
- **高级搜索选项** - 使用分数、过滤器和元数据微调搜索

### 🏢 **生产就绪**
- **100% 本地** - 使用本地提供商完全离线操作
- **HTTP API** - 所有操作的 RESTful 端点
- **高性能** - 优化的 Go 实现
- **可配置** - 通过 TOML 进行广泛配置
- **零配置模式** - 使用智能默认值开箱即用

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
go build -o rago ./cmd/rago-cli

# 可选：创建配置（仅在需要自定义设置时）
# 复制示例配置文件并根据需要修改：
cp rago.example.toml rago.toml
# 编辑 rago.toml 设置您的首选配置
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



## 📖 库使用

### 简化的客户端 API（推荐）

新的客户端包为所有 RAGO 功能提供了简洁的接口：

```go
import "github.com/liliang-cn/rago/v2/client"

// 创建客户端 - 现在只有两个入口点！
client, err := client.New("path/to/config.toml")  // 或空字符串使用默认值
// 或使用程序化配置
cfg := &config.Config{...}
client, err := client.NewWithConfig(cfg)
defer client.Close()

// 使用包装器的 LLM 操作
if client.LLM != nil {
    response, err := client.LLM.Generate("写一首俳句")
    
    // 带选项
    resp, err := client.LLM.GenerateWithOptions(ctx, "解释量子计算", 
        &client.GenerateOptions{Temperature: 0.7, MaxTokens: 200})
    
    // 流式处理
    err = client.LLM.Stream(ctx, "讲个故事", func(chunk string) {
        fmt.Print(chunk)
    })
}

// 使用包装器的 RAG 操作
if client.RAG != nil {
    err = client.RAG.Ingest("您的文档内容")
    answer, err := client.RAG.Query("这是关于什么的？")
    
    // 带来源
    resp, err := client.RAG.QueryWithOptions(ctx, "告诉我更多",
        &client.QueryOptions{TopK: 5, ShowSources: true})
}

// 使用包装器的 MCP 工具
if client.Tools != nil {
    tools, err := client.Tools.List()
    result, err := client.Tools.Call(ctx, "filesystem_read", 
        map[string]interface{}{"path": "README.md"})
}


// 也可以直接使用 BaseClient 方法
resp, err := client.Query(ctx, client.QueryRequest{Query: "测试"})
resp, err := client.RunTask(ctx, client.TaskRequest{Task: "分析数据"})
```

### 高级用法示例

展示所有客户端功能的综合示例：

- **[基本客户端初始化](./examples/01_basic_client)** - 初始化客户端的不同方式
- **[LLM 操作](./examples/02_llm_operations)** - 生成、流式传输、带历史的聊天
- **[RAG 操作](./examples/03_rag_operations)** - 文档摄取、查询、语义搜索
- **[MCP 工具集成](./examples/04_mcp_tools)** - 工具列表、执行、LLM 集成
- **[完整平台演示](./examples/06_complete_platform)** - 所有功能协同工作

### 直接包使用（高级）

如需精细控制，可直接使用底层包：

```go
import (
    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/rag/processor"
    "github.com/liliang-cn/rago/v2/pkg/store"
)

// 初始化组件
cfg, _ := config.Load("rago.toml")
store, _ := store.NewSQLiteStore(cfg.Sqvect.DBPath)
processor := processor.New(cfg, store)

// 直接 RAG 操作
doc := domain.Document{
    ID:      "doc1",
    Content: "您的文档内容",
}
err := processor.IngestDocument(ctx, doc)

// 使用自定义参数查询
req := domain.QueryRequest{
    Query:       "这是关于什么的？",
    TopK:        5,
    Temperature: 0.7,
}
response, _ := processor.Query(ctx, req)
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

### 示例
- [客户端使用示例](./examples/) - 全面的客户端库示例
  - [基本客户端](./examples/01_basic_client) - 客户端初始化方法
  - [LLM 操作](./examples/02_llm_operations) - 直接 LLM 使用
  - [RAG 操作](./examples/03_rag_operations) - 文档摄取和查询
  - [MCP 工具](./examples/04_mcp_tools) - 工具集成模式
    - [完整平台](./examples/06_complete_platform) - 完整集成示例

### 参考文档
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
