# RAGO v2 命令参考

## 📋 概览

RAGO v2 使用子命令结构来组织不同功能。RAG 相关操作都在 `rag` 子命令下。

## 🚀 基本命令结构

```bash
./rago-cli [全局选项] <命令> [子命令] [选项] [参数]
```

## 🔧 全局选项

- `--config string`: 配置文件路径 (默认: ~/.rago/rago.toml 或 ./rago.toml)
- `--db-path string`: 数据库路径 (默认: ./.rago/data/rag.db)
- `-q, --quiet`: 静默模式
- `-v, --verbose`: 详细日志输出
- `-h, --help`: 显示帮助信息

## 📊 主要命令

### `status` - 系统状态检查
检查 LLM 提供商连接状态

```bash
./rago-cli status
```

### `llm` - 语言模型操作
与各种提供商的语言模型进行交互

#### 子命令
- `chat`: 与 LLM 聊天
- `list`: 列出可用的 LLM 模型

```bash
# LLM 聊天
./rago-cli llm chat "请解释什么是 RAG"

# 列出可用模型
./rago-cli llm list
```

### `rag` - RAG (检索增强生成) 操作
文档摄取、查询和知识库管理

#### 子命令
- `ingest`: 导入文档到向量数据库
- `query`: 查询知识库
- `list`: 列出已索引的文档
- `reset`: 清空向量数据库
- `collections`: 列出所有集合
- `import`: 导入知识库数据
- `export`: 导出知识库数据

```bash
# 导入文档
./rago-cli rag ingest document.pdf
./rago-cli rag ingest "path/to/text/file.txt"
./rago-cli rag ingest --text "直接文本内容" --source "文档名称"

# 查询知识库
./rago-cli rag query "这个文档是关于什么的？"

# 列出文档
./rago-cli rag list

# 查看集合
./rago-cli rag collections

# 清空数据库
./rago-cli rag reset
```

#### `rag ingest` 选项
- `-b, --batch-size int`: 批处理大小 (默认 10)
- `-c, --chunk-size int`: 文本块大小 (默认 300)
- `-e, --enhanced`: 启用增强元数据提取
- `-o, --overlap int`: 文本块重叠大小 (默认 50)
- `-r, --recursive`: 递归处理目录
- `--source string`: 文本输入的源名称 (默认: text-input)
- `--text string`: 直接摄取文本而不是从文件

#### `rag query` 选项
- `-e, --enhanced`: 启用增强查询
- `-m, --mcp`: 启用 MCP 工具集成
- `-s, --show-sources`: 显示来源文档

### `mcp` - MCP (模型上下文协议) 管理
管理 MCP 服务器和工具

```bash
# 检查 MCP 状态
./rago-cli mcp status

# 列出可用工具
./rago-cli mcp tools

# 调用工具
./rago-cli mcp tools call <tool-name> '{"arg": "value"}'
```

### `serve` - 启动 HTTP API 服务
启动 RESTful API 服务器

```bash
# 启动 API 服务
./rago-cli serve --port 7127

# 启动带 UI 的服务 (如果支持)
./rago-cli serve --ui --port 7127
```

### `profile` - 用户配置管理
管理用户配置文件和 LLM 设置（v2.17.0 完全功能）

```bash
# 显示当前配置
./rago-cli profile show

# 创建新配置
./rago-cli profile create "research" "Profile for academic research"

# 列出所有配置
./rago-cli profile list

# 设置活跃配置
./rago-cli profile set-active research

# 更新配置
./rago-cli profile update research --description "Updated research profile"

# 删除配置
./rago-cli profile delete research

# 配置 LLM 设置
./rago-cli profile llm-settings research --temperature 0.3 --max-tokens 3000 --system-prompt "You are a research assistant"
```

### `examples` - 运行示例程序
运行 RAGO v2 库使用示例（v2.17.0 新增）

```bash
# 基础 RAG 使用示例
./rago-cli examples basic

# 高级功能示例（Profile + MCP）
./rago-cli examples advanced

# 快速入门演示（所有功能）
./rago-cli examples quickstart
```

### `usage` - 使用统计
查看 RAGO 使用情况和统计信息

```bash
# 显示使用统计
./rago-cli usage

# 显示详细统计
./rago-cli usage --verbose
```

## 🌐 HTTP API 端点

当使用 `serve` 命令启动服务时，可用的 API 端点：

### RAG 操作
- `POST /api/rag/ingest` - 摄取文档到向量数据库
- `POST /api/rag/query` - 执行 RAG 查询
- `GET /api/rag/list` - 列出已索引文档
- `DELETE /api/rag/reset` - 清空向量数据库
- `GET /api/rag/collections` - 列出所有集合

### LLM 操作
- `POST /api/llm/chat` - LLM 聊天
- `POST /api/llm/generate` - 文本生成
- `GET /api/llm/models` - 列出可用模型

### MCP 工具
- `GET /api/mcp/tools` - 列出可用 MCP 工具
- `POST /api/mcp/tools/call` - 执行 MCP 工具
- `GET /api/mcp/status` - 检查 MCP 服务器状态

### 系统信息
- `GET /api/status` - 系统状态
- `GET /api/version` - 版本信息

## 📝 使用示例

### 完整的 RAG 工作流

```bash
# 1. 检查系统状态
./rago-cli status

# 2. 导入文档
./rago-cli rag ingest README.md

# 3. 查看导入的文档
./rago-cli rag list

# 4. 查询知识库
./rago-cli rag query "这个项目的主要功能是什么？"

# 5. 直接与 LLM 对话
./rago-cli llm chat "你能帮我总结一下这个项目吗？"
```

### 批量处理文档

```bash
# 递归导入目录
./rago-cli rag ingest ./docs --recursive

# 自定义块大小
./rago-cli rag ingest document.pdf --chunk-size 1000

# 启用增强元数据提取
./rago-cli rag ingest document.pdf --enhanced
```

## 🔧 故障排除

### 常见问题

1. **"embedder service does not implement MetadataExtractor interface"**
   - 使用 `--enhanced` 标志时确保配置正确
   - 或者不使用 `--enhanced` 标志

2. **"model not found" 错误**
   - 确保 Ollama 中已安装所需模型
   - 检查模型名称是否正确

3. **连接超时**
   - 检查 Ollama 服务是否运行
   - 使用 `./rago-cli status --verbose` 查看详细错误

### 调试技巧

```bash
# 使用详细日志
./rago-cli --verbose rag query "测试问题"

# 检查配置
./rago-cli --config ./custom-config.toml status

# 测试 LLM 连接
./rago-cli llm chat "测试连接"
```

## 📚 更多信息

- [主文档](README.md) - 完整的项目文档
- [配置示例](rago.example.toml) - 详细的配置选项
- [中文文档](README_zh-CN.md) - 中文版本文档
- [库使用指南](docs/LIBRARY_USAGE.md) - 完整的库 API 文档
- [示例代码](examples/) - 使用示例
  - [基础 RAG 示例](examples/basic_rag_usage/) - 基础库使用
  - [高级功能示例](examples/advanced_features/) - Profile + MCP
  - [快速入门演示](examples/quickstart/) - 所有功能演示

## 🆕 v2.17.0 新功能

### Profile Management (完全功能)
- ✅ **多用户支持** - 创建和管理不同配置
- ✅ **LLM 设置** - 每个配置独立的 LLM 参数
- ✅ **配置切换** - 无缝切换不同环境
- ✅ **设置持久化** - 自动保存和加载用户偏好

### MCP Integration (完全功能)
- ✅ **工具管理** - 列出和调用 MCP 工具
- ✅ **服务状态** - 实时监控 MCP 服务器
- ✅ **工具调用** - 程序化工具执行
- ✅ **配置支持** - 灵活的 MCP 服务器配置

### Enhanced Library API
- ✅ **完整客户端** - 600+ 行的完整实现
- ✅ **类型安全** - 所有方法都有正确类型
- ✅ **错误处理** - 全面的错误处理机制
- ✅ **向后兼容** - 保持 API 稳定性