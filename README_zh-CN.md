# RAGO - 本地化 RAG 系统

RAGO（Retrieval-Augmented Generation Offline）是一个完全本地运行的 RAG 系统，基于 Go 编写，集成 SQLite 向量库（sqvect）和本地 LLM 客户端（ollama-go），支持文档 ingest、语义搜索和上下文增强问答。

## 🎯 特性

- **完全离线运行** - 无需外部 API，保护数据隐私
- **多格式支持** - 支持 TXT、Markdown、PDF 等格式
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
   ollama pull gemma3           # 生成模型
   ```

### 安装 RAGO

```bash
git clone https://github.com/liliang-cn/rago.git
cd rago
make setup
make build
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
   ./build/rago serve --port 8080
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

# 从 URL 导入（规划中）
rago ingest --url "https://example.com/doc.html"
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
rago serve --port 8080 --host 0.0.0.0
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
  "stream": false
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
  "top_k": 5
}
```

## ⚙️ 配置

### 配置文件

创建 `config.toml`：

```toml
[server]
port = 8080
host = "localhost"
cors_origins = ["*"]

[ollama]
embedding_model = "nomic-embed-text"
llm_model = "gemma3"
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
export RAGO_SERVER_PORT=8080
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
  -p 8080:8080 \
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
      - "8080:8080"
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
