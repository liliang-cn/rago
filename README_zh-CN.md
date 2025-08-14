# RAGO - 本地化 RAG 系统

RAGO（Retrieval-Augmented Generation Offline）是一个完全本地运行的 RAG 系统，基于 Go 编写，集成 SQLite 向量库（sqvect）和本地 LLM 客户端（ollama-go），支持文档 ingest、语义搜索和上下文增强问答。

## 🎯 特性

- **完全离线运行** - 无需外部 API，保护数据隐私
- **多格式支持** - 支持 TXT、Markdown 等文本格式
- **本地向量数据库** - 基于 SQLite 的 sqvect 向量存储
- **本地 LLM** - 通过 Ollama 调用本地模型
- **双接口设计** - CLI 工具和 HTTP API 两种使用方式
- **高性能** - Go 语言实现，内存占用低，响应速度快
- **可扩展** - 模块化设计，易于扩展新功能

## 🚀 快速开始

### 前置条件

1. **安装 Go** (≥ 1.21)
2. **安装 Ollama**
   ```bash
   curl -fsSL https://ollama.com/install.sh | sh
   ```
3. **下载模型**
   ```bash
   ollama pull nomic-embed-text  # 嵌入模型
   ollama pull qwen3           # 生成模型
   ```

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
go install github.com/liliang-cn/rago/cmd/rago-cli@latest

# 二进制文件名为 'rago-cli'
rago-cli --help
```

### 基本使用

1. **导入文档**

   ```bash
   ./build/rago ingest ./docs/sample.md
   ./build/rago ingest ./docs/ --recursive  # 递归处理目录
   ```

2. **查询知识库**

   ```bash
   ./build/rago query "什么是 RAG？"
   ./build/rago query --interactive         # 交互模式
   ```

3. **启动 API 服务**

   ```bash
   ./build/rago serve --port 7127
   ```

4. **查看已导入文档**
   ```bash
   ./build/rago list
   ```

## 📖 详细使用

### CLI 命令

#### 文档管理

```bash
# 单文件导入
rago ingest ./document.txt

# 批量导入
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

### 配置文件

创建 `config.toml`：

```toml
[server]
port = 7127
host = "localhost"
cors_origins = ["*"]

[ollama]
embedding_model = "nomic-embed-text"
llm_model = "qwen3"
base_url = "http://localhost:11434"
timeout = "30s"

[sqvect]
db_path = "./data/rag.db"
top_k = 5

[chunker]
chunk_size = 300
overlap = 50
method = "sentence"  # sentence, paragraph, token

[ui]
title = "RAGO - 本地 RAG 系统"
theme = "light"
max_file_size = "10MB"
```

### 环境变量

```bash
export RAGO_SERVER_PORT=7127
export RAGO_OLLAMA_BASE_URL=http://localhost:11434
export RAGO_SQVECT_DB_PATH=./data/custom.sqlite
```

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
