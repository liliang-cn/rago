# RAGO - 本地化 RAG 系统

RAGO（Retrieval-Augmented Generation Offline）是一个完全本地运行的 RAG 系统，基于 Go 编写，集成 SQLite 向量库（sqvect）和多种 LLM 提供商（OpenAI、Ollama），支持文档导入、语义搜索、工具调用和上下文增强问答。

## 🎯 特性

- **多种 LLM 提供商** - 支持 OpenAI、Ollama 和其他兼容提供商
- **灵活配置架构** - 现代化的 provider 架构，易于切换服务
- **工具调用支持** - 内置工具：网络搜索、文件操作、时间查询等
- **完全离线选项** - 使用 Ollama 实现完整数据隐私保护
- **多格式支持** - 支持 PDF、TXT、Markdown 等文本格式
- **本地向量数据库** - 基于 SQLite 的 sqvect 向量存储
- **Web UI 界面** - 内置 Web 界面，易于交互
- **双接口设计** - CLI 工具和 HTTP API 两种使用方式
- **高性能** - Go 语言实现，内存占用低，响应速度快
- **可扩展** - 模块化设计，易于扩展新功能

## 🚀 快速开始

### 前置条件

**方式一：完全本地部署（Ollama）**

1. **安装 Go** (≥ 1.21)
2. **安装 Ollama**
   ```bash
   curl -fsSL https://ollama.com/install.sh | sh
   ```
3. **下载模型**
   ```bash
   ollama pull nomic-embed-text  # 嵌入模型
   ollama pull qwen2.5          # 生成模型（或 qwen3）
   ```

**方式二：使用 OpenAI**

1. **安装 Go** (≥ 1.21)
2. **获取 OpenAI API 密钥** - 访问 [platform.openai.com](https://platform.openai.com)

### 安装 RAGO

#### 方式一：从源码安装

```bash
git clone https://github.com/liliang-cn/rago.git
cd rago
make setup
make build
```

#### 方式二：使用 go install 安装

```bash
go install github.com/liliang-cn/rago@latest

# 二进制文件名为 'rago'
rago --help
```

### 基本使用

使用 `make build` 构建项目后，你可以在 `build` 目录中使用 `rago` 二进制文件。

1. **初始化配置**

   ```bash
   ./build/rago init                    # 创建默认配置（Ollama）
   ./build/rago init --force            # 强制覆盖现有配置文件
   ./build/rago init -o custom.toml     # 在自定义路径创建配置文件
   ```

   `init` 命令自动创建：

   - 基于 provider 架构的现代配置，默认使用 Ollama
   - 完整目录结构（./data/）
   - 默认启用工具调用功能
   - 默认启用 Web UI
   - 包含 OpenAI 配置示例（已注释）

2. **配置提供商**（如果使用 OpenAI）

   编辑生成的 `config.toml` 文件，取消注释 OpenAI 部分：

   ```toml
   [providers]
   default_llm = "openai"           # 从 "ollama" 改为 "openai"
   default_embedder = "openai"      # 从 "ollama" 改为 "openai"

   [providers.openai]
   type = "openai"
   api_key = "your-openai-api-key-here"
   # ... 其他设置
   ```

3. **导入文档**

   ```bash
   ./build/rago ingest ./docs/sample.md
   ./build/rago ingest ./docs/ --recursive  # 递归处理目录
   ```

4. **查询知识库**

   ```bash
   ./build/rago query "什么是 RAG？"
   ./build/rago query --interactive         # 交互模式
   ```

5. **启动 Web UI 服务**

   ```bash
   ./build/rago serve --port 7127
   # 在浏览器中访问 http://localhost:7127
   ```

6. **检查状态**

   ```bash
   ./build/rago status                      # 检查提供商连接状态
   ```

7. **工具调用示例**

   在使用查询命令或 Web 界面时，RAGO 会自动使用内置工具：

   ```bash
   ./build/rago query "东京现在几点了？"                    # 使用时间工具
   ./build/rago query "搜索最新的 AI 新闻"                  # 使用网络搜索
   ./build/rago query "我有哪些关于 Python 的文档？"         # 使用 rag_search 工具
   ```

## 📖 详细使用

### CLI 命令

#### 配置管理

```bash
# 使用默认设置初始化配置
rago init

# 覆盖现有配置文件
rago init --force

# 在自定义位置创建配置
rago init --output ./config/rago.toml

# 查看 init 命令帮助
rago init --help
```

#### 文档管理

```bash
# 单文件导入
rago ingest ./document.txt

# 批量导入并自动提取元数据
rago ingest ./docs/ --recursive --extract-metadata

# 调整分块参数
rago ingest ./docs/ --recursive --chunk-size 500 --overlap 100
```

#### 查询功能

```bash
# 直接查询
rago query "你好世界"

# 交互模式
rago query --interactive

# 流式输出
rago query "解释一下机器学习" --stream

# 批量查询
rago query --file questions.txt

# 调整参数
rago query "什么是深度学习" --top-k 10 --temperature 0.3 --max-tokens 1000

# 使用过滤器查询（需要文档包含元数据）
rago query "机器学习概念" --filter "source=textbook" --filter "category=ai"
```

#### 数据管理

```bash
# 列出所有文档
rago list

# 重置数据库
rago reset --force

# 导出数据（规划中）
rago export ./backup.json

# 导入数据（规划中）
rago import ./backup.json
```

### HTTP API

启动服务器：

```bash
rago serve --port 7127 --host 0.0.0.0
```

#### API 端点

**健康检查**

```bash
GET /api/health
```

**文档导入**

```bash
POST /api/ingest
Content-Type: application/json

{
  "content": "这是要导入的文本内容",
  "chunk_size": 300,
  "overlap": 50,
  "metadata": {
    "source": "manual_input"
  }
}
```

**查询**

```bash
POST /api/query
Content-Type: application/json

{
  "query": "什么是人工智能？",
  "top_k": 5,
  "temperature": 0.7,
  "max_tokens": 500,
  "stream": false,
  "filters": {
    "source": "textbook",
    "category": "ai"
  }
}
```

**文档管理**

```bash
# 获取文档列表
GET /api/documents

# 删除文档
DELETE /api/documents/{document_id}
```

**搜索（仅检索）**

```bash
POST /api/search
Content-Type: application/json

{
  "query": "人工智能",
  "top_k": 5,
  "filters": {
    "source": "textbook",
    "category": "ai"
  }
}
```

#### 过滤器支持

RAGO 支持基于文档元数据的过滤搜索结果。这允许您在知识库的特定子集中进行搜索：

**CLI 使用：**

```bash
# 使用过滤器查询
rago query "机器学习" --filter "source=textbook" --filter "author=张三"

# 仅搜索（无生成）使用过滤器
rago search "神经网络" --filter "category=deep-learning" --filter "year=2023"
```

**API 使用：**

```bash
# 使用过滤器查询
curl -X POST http://localhost:7127/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "什么是机器学习？",
    "filters": {
      "source": "textbook",
      "category": "ai"
    }
  }'

# 使用过滤器搜索
curl -X POST http://localhost:7127/api/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "神经网络",
    "filters": {
      "category": "deep-learning"
    }
  }'
```

**注意：** 为了使过滤功能有效工作，文档在导入时必须设置适当的元数据字段。

### 作为库使用

RAGO 可以作为 Go 库在您的项目中使用。这允许您将 RAGO 的 RAG 功能直接集成到您的应用程序中，而无需将其作为单独的 CLI 工具运行。

#### 安装

```bash
go get github.com/liliang-cn/rago
```

#### 导入库

```go
import "github.com/liliang-cn/rago/lib"
```

#### 创建客户端

```go
// 使用默认配置文件（当前目录下的 config.toml）
client, err := rago.New("config.toml")
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// 或使用自定义配置
cfg := &config.Config{
    // ... 您的配置
}
client, err := rago.NewWithConfig(cfg)
```

#### 基本操作

```go
// 导入文本内容
err = client.IngestText("您的文本内容", "来源名称")

// 导入文件
err = client.IngestFile("/path/to/your/file.txt")

// 查询知识库
response, err := client.Query("您的问题")
fmt.Println("答案:", response.Answer)

// 使用过滤器查询
filters := map[string]interface{}{
    "source": "textbook",
    "category": "ai",
}
response, err := client.QueryWithFilters("您的过滤问题", filters)
fmt.Println("过滤答案:", response.Answer)

// 流式查询
err = client.StreamQuery("您的问题", func(chunk string) {
    fmt.Print(chunk)
})

// 使用过滤器的流式查询
err = client.StreamQueryWithFilters("您的过滤问题", filters, func(chunk string) {
    fmt.Print(chunk)
})

// 列出文档
docs, err := client.ListDocuments()

// 删除文档
err = client.DeleteDocument(documentID)

// 重置数据库
err = client.Reset()
```

#### 库配置

库使用与 CLI 工具相同的配置格式。您可以：

1. 将配置文件路径传递给 `rago.New(configPath)`
2. 自己加载配置并传递给 `rago.NewWithConfig(config)`

库将从以下位置读取配置：

- 指定的配置文件路径
- `./config.toml`（当前目录）
- `./config/config.toml`
- `$HOME/.rago/config.toml`

#### 示例

查看 `examples/library_usage.go` 以获取如何将 RAGO 用作库的完整示例。

```bash
cd examples
go run library_usage.go
```

#### API 参考

**客户端方法**

- `New(configPath string) (*Client, error)` - 使用配置文件创建客户端
- `NewWithConfig(cfg *config.Config) (*Client, error)` - 使用配置结构创建客户端
- `IngestFile(filePath string) error` - 导入文件
- `IngestText(text, source string) error` - 导入文本内容
- `Query(query string) (domain.QueryResponse, error)` - 查询知识库
- `QueryWithFilters(query string, filters map[string]interface{}) (domain.QueryResponse, error)` - 使用过滤器查询
- `StreamQuery(query string, callback func(string)) error` - 流式查询响应
- `StreamQueryWithFilters(query string, filters map[string]interface{}, callback func(string)) error` - 使用过滤器的流式查询
- `ListDocuments() ([]domain.Document, error)` - 列出所有文档
- `DeleteDocument(documentID string) error` - 删除文档
- `Reset() error` - 重置数据库
- `Close() error` - 关闭客户端并清理
- `GetConfig() *config.Config` - 获取当前配置

## ⚙️ 配置

### 初始化配置

RAGO 提供 `init` 命令来快速生成基于 provider 架构的现代配置文件：

```bash
# 创建默认设置的 config.toml（使用 Ollama 和目录结构）
rago init

# 覆盖现有配置文件
rago init --force

# 在自定义路径创建配置文件
rago init --output /path/to/config.toml
```

`init` 命令自动：

- 创建现代化的基于 provider 的配置
- 建立完整的目录结构（./data/、./data/documents/ 等）
- 默认启用工具调用功能
- 默认启用 Web UI
- 包含注释的 OpenAI 配置示例，便于切换

### Provider 配置

RAGO 使用灵活的 provider 系统，支持多种 LLM 和嵌入服务：

#### Ollama 配置（默认）

```toml
[providers]
default_llm = "ollama"
default_embedder = "ollama"

[providers.ollama]
type = "ollama"
base_url = "http://localhost:11434"
llm_model = "qwen2.5"
embedding_model = "nomic-embed-text"
timeout = "120s"
```

#### OpenAI 配置

```toml
[providers]
default_llm = "openai"
default_embedder = "openai"

[providers.openai]
type = "openai"
api_key = "your-openai-api-key-here"
base_url = "https://api.openai.com/v1"          # 可选：用于自定义端点
llm_model = "gpt-4o-mini"
embedding_model = "text-embedding-3-small"
timeout = "60s"
```

#### 混合 Provider 设置

您可以为 LLM 和嵌入使用不同的提供商：

```toml
[providers]
default_llm = "openai"      # 使用 OpenAI 进行生成
default_embedder = "ollama" # 使用 Ollama 进行嵌入

# 配置两个提供商
[providers.openai]
type = "openai"
api_key = "your-api-key"
llm_model = "gpt-4o-mini"

[providers.ollama]
type = "ollama"
base_url = "http://localhost:11434"
embedding_model = "nomic-embed-text"
```

### 工具配置

RAGO 包含可以启用/禁用的内置工具：

```toml
[tools]
enabled = true                           # 启用工具调用
max_concurrent_calls = 5                 # 最大并行工具调用数
call_timeout = "30s"                     # 每个工具调用的超时时间
security_level = "normal"                # 安全级别：strict、normal、relaxed
log_level = "info"                       # 日志级别：debug、info、warn、error

# 可用的内置工具
enabled_tools = [
    "datetime",                          # 日期时间操作
    "rag_search",                        # 在 RAG 数据库中搜索
    "document_info",                     # 文档信息查询
    "open_url",                       # HTTP 网络请求
    "web_search",                     # Google 搜索功能
    "file_operations"                    # 文件系统操作
]

# 工具特定配置
[tools.builtin.web_search]
enabled = true
api_key = "your-google-api-key"         # 可选：获得更好的速率限制
search_engine_id = "your-cse-id"        # 可选：自定义搜索引擎

[tools.builtin.open_url]
enabled = true
timeout = "10s"
max_redirects = 5
user_agent = "RAGO/1.3.1"

[tools.builtin.file_operations]
enabled = true
max_file_size = "10MB"
allowed_extensions = [".txt", ".md", ".json", ".csv", ".log"]
base_directory = "./data"
```

### 完整配置示例

```toml
[server]
port = 7127
host = "0.0.0.0"
enable_ui = true
cors_origins = ["*"]

[providers]
default_llm = "ollama"
default_embedder = "ollama"

[providers.ollama]
type = "ollama"
base_url = "http://localhost:11434"
llm_model = "qwen2.5"
embedding_model = "nomic-embed-text"
timeout = "120s"

[sqvect]
db_path = "./data/rag.db"
vector_dim = 768                         # nomic-embed-text 为 768，OpenAI 为 1536
max_conns = 10
batch_size = 100
top_k = 5
threshold = 0.0

[keyword]
index_path = "./data/keyword.bleve"

[chunker]
chunk_size = 500
overlap = 50
method = "sentence"                      # sentence、paragraph、token

[ingest.metadata_extraction]
enable = false                           # 启用自动元数据提取
llm_model = "qwen2.5"                   # 用于元数据提取的模型

[tools]
enabled = true
enabled_tools = ["datetime", "rag_search", "open_url", "web_search"]
```

### 环境变量

```bash
export RAGO_SERVER_PORT=7127
export RAGO_PROVIDERS_DEFAULT_LLM=openai
export RAGO_PROVIDERS_OPENAI_API_KEY=your-key-here
export RAGO_SQVECT_DB_PATH=./data/custom.sqlite
```

## 🛠️ 工具调用

RAGO 包含全面的工具系统，允许 AI 执行操作并检索实时信息：

### 内置工具

#### 🕐 时间工具

- 获取当前日期和时间
- 时区之间的转换
- 计算时间差

**示例：**

```bash
"现在几点了？"
"东京的当前日期是什么？"
"距离圣诞节还有多少天？"
```

#### 🔍 RAG 搜索工具

- 搜索您的已导入文档
- 从知识库中检索特定信息
- 交叉引用文档来源

**示例：**

```bash
"哪些文档提到了 Python 编程？"
"在我的笔记中搜索机器学习相关信息"
"找到所有 API 文档的引用"
```

#### 📄 文档信息工具

- 获取已导入文档的元数据
- 列出可用文档
- 检查文档统计信息

**示例：**

```bash
"我有多少个文档？"
"最新添加的文档有哪些？"
"显示文档统计信息"
```

#### 🌐 网络请求工具

- 向 API 发出 HTTP 请求
- 获取网页内容
- 访问实时数据

**示例：**

```bash
"从 example.com/api 获取最新新闻"
"从这个 REST API 端点获取数据"
"检查这个网站的状态"
```

#### 🔎 Google 搜索工具

- 通过 Google 搜索互联网
- 获取最新信息
- 查找特定资源

**示例：**

```bash
"搜索 AI 的最新发展"
"量子计算的最新新闻是什么？"
"找到 React hooks 的文档"
```

#### 📁 文件操作工具

- 读取本地文件
- 列出目录内容
- 检查文件信息

**示例：**

```bash
"读取我的 todo.txt 文件内容"
"列出 ./projects 目录中的文件"
"我的配置文件里有什么？"
```

### 工具配置

工具可以单独启用/禁用和配置：

```toml
[tools]
enabled = true
max_concurrent_calls = 5
call_timeout = "30s"
security_level = "normal"  # 控制工具访问级别

# 启用特定工具
enabled_tools = [
    "datetime",
    "rag_search",
    "document_info",
    "open_url",
    "web_search",
    "file_operations"
]

# 工具特定设置
[tools.builtin.file_operations]
enabled = true
base_directory = "./data"                           # 限制文件访问到这个目录
allowed_extensions = [".txt", ".md", ".json"]       # 只允许这些文件类型
max_file_size = "10MB"                              # 最大读取文件大小

[tools.builtin.open_url]
enabled = true
timeout = "10s"
max_redirects = 5
user_agent = "RAGO/1.3.1"

[tools.builtin.web_search]
enabled = true
# api_key = "your-google-api-key"        # 可选：获得更好的速率限制
# search_engine_id = "your-cse-id"       # 可选：自定义搜索引擎
```

### 安全级别

- **strict**: 非常有限的工具访问，适合生产环境
- **normal**: 平衡的安全性和功能性（默认）
- **relaxed**: 完全的工具访问，请谨慎使用

### API 中的工具使用

使用 API 时工具会自动调用：

```bash
# 会触发工具使用的查询
curl -X POST http://localhost:7127/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "现在几点了，搜索最新的 AI 新闻？",
    "stream": false
  }'
```

响应将包含工具结果和 AI 的综合回答。

## 🐳 Docker 部署

### 构建镜像

```bash
make docker-build
```

### 运行容器

```bash
docker run -d \
  --name rago \
  -p 7127:7127 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config.toml:/app/config.toml \
  rago:latest
```

### Docker Compose

```yaml
version: "3.8"
services:
  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama_data:/root/.ollama

  rago:
    build: .
    ports:
      - "7127:7127"
    volumes:
      - ./data:/app/data
      - ./config.toml:/app/config.toml
    depends_on:
      - ollama
    environment:
      - RAGO_OLLAMA_BASE_URL=http://ollama:11434

volumes:
  ollama_data:
```

## 🧪 开发

### 构建和测试

```bash
# 安装依赖
make deps

# 格式化代码
make fmt

# 运行测试
make test

# 代码检查
make check

# 开发模式运行
make dev
```

### 项目结构

```
rago/
├── cmd/rago/           # CLI 命令
├── internal/           # 内部模块
│   ├── config/        # 配置管理
│   ├── domain/        # 领域模型
│   ├── chunker/       # 文本分块
│   ├── embedder/      # 嵌入服务
│   ├── llm/           # 生成服务
│   ├── store/         # 存储层
│   └── processor/     # 核心处理器
├── api/handlers/       # API 处理器
├── test/              # 集成测试
├── docs/              # 文档
└── Makefile           # 构建脚本
```

## 🤝 贡献

欢迎贡献代码！请：

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 🙏 致谢

- [Ollama](https://ollama.com/) - 本地 LLM 运行时
- [SQLite](https://sqlite.org/) - 嵌入式数据库
- [Gin](https://gin-gonic.com/) - HTTP Web 框架
- [Cobra](https://cobra.dev/) - CLI 应用框架

## 📞 联系

如有问题或建议，请通过以下方式联系：

- GitHub Issues: [https://github.com/liliang-cn/rago/issues](https://github.com/liliang-cn/rago/issues)
- Email: your.email@example.com

---

⭐ 如果这个项目对您有帮助，请给个 Star！
