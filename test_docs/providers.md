# RAGO Provider Configuration Guide

RAGO 现在支持多种 LLM 提供商，你可以选择使用本地的 Ollama 或 OpenAI 兼容的服务。

## 支持的提供商

1. **Ollama** - 本地 LLM 服务
2. **OpenAI** - OpenAI API 或兼容服务（如 vLLM、LocalAI 等）

## 配置方式

### 使用新的提供商配置格式（推荐）

在 `config.toml` 中使用新的 `[providers]` 配置：

```toml
# 设置默认提供商
[providers]
default_llm = "ollama"      # 或 "openai"  
default_embedder = "ollama" # 或 "openai"

# Ollama 提供商配置
[providers.ollama]
type = "ollama"
embedding_model = "nomic-embed-text"
llm_model = "qwen3"
base_url = "http://localhost:11434"
timeout = "30s"

# OpenAI 提供商配置（可选）
[providers.openai]
type = "openai"
api_key = "your-api-key"
base_url = "https://api.openai.com/v1" 
embedding_model = "text-embedding-3-small"
llm_model = "gpt-4"
timeout = "60s"
```

### 环境变量支持

可以通过环境变量覆盖配置：

```bash
export RAGO_PROVIDERS_DEFAULT_LLM=openai
export RAGO_PROVIDERS_DEFAULT_EMBEDDER=ollama
export RAGO_OPENAI_API_KEY=sk-your-key
export RAGO_OLLAMA_BASE_URL=http://localhost:11434
```

## 使用场景

### 1. 纯 Ollama（本地）

```toml
[providers]
default_llm = "ollama"
default_embedder = "ollama"

[providers.ollama]
type = "ollama"
embedding_model = "nomic-embed-text"
llm_model = "qwen3"
base_url = "http://localhost:11434"
timeout = "30s"
```

### 2. 纯 OpenAI

```toml
[providers]
default_llm = "openai"
default_embedder = "openai"

[providers.openai]
type = "openai"
api_key = "sk-your-key"
base_url = "https://api.openai.com/v1"
embedding_model = "text-embedding-3-small"
llm_model = "gpt-4"
timeout = "60s"
```

**注意**：使用 OpenAI embeddings 时，需要调整向量维度：
```toml
[sqvect]
vector_dim = 1536  # text-embedding-3-small
# 或
vector_dim = 3072  # text-embedding-3-large
```

### 3. 混合配置

使用 OpenAI 进行文本生成，Ollama 进行向量化（节省成本）：

```toml
[providers]
default_llm = "openai"      # 使用 OpenAI 生成文本
default_embedder = "ollama" # 使用 Ollama 生成向量

[providers.ollama]
type = "ollama"
embedding_model = "nomic-embed-text"
base_url = "http://localhost:11434"
timeout = "30s"

[providers.openai]
type = "openai"
api_key = "sk-your-key"
llm_model = "gpt-4"
timeout = "60s"

[sqvect]
vector_dim = 768  # 匹配 nomic-embed-text
```

### 4. 本地 OpenAI 兼容服务

使用如 vLLM、LocalAI、Ollama OpenAI API 等：

```toml
[providers]
default_llm = "openai"
default_embedder = "openai"

[providers.openai]
type = "openai"
api_key = "dummy-key"  # 本地服务通常不需要真实 key
base_url = "http://localhost:8000/v1"
embedding_model = "BAAI/bge-large-en-v1.5"
llm_model = "meta-llama/Llama-2-7b-chat-hf"
timeout = "120s"
```

## 兼容性

### 后向兼容

旧的 `[ollama]` 配置格式仍然支持，但建议迁移到新格式：

```toml
# 旧格式（仍支持）
[ollama]
embedding_model = "nomic-embed-text"
llm_model = "qwen3"
base_url = "http://localhost:11434"
timeout = "30s"
```

### 迁移指南

1. **从旧配置迁移到新配置**：
   ```toml
   # 旧配置
   [ollama]
   embedding_model = "nomic-embed-text"
   llm_model = "qwen3"
   
   # 新配置
   [providers]
   default_llm = "ollama"
   default_embedder = "ollama"
   
   [providers.ollama]
   type = "ollama" 
   embedding_model = "nomic-embed-text"
   llm_model = "qwen3"
   ```

2. **添加 OpenAI 支持**：只需添加 `[providers.openai]` 配置块并更新默认提供商。

## 故障排除

### 常见问题

1. **OpenAI API 密钥错误**
   - 确保 `api_key` 正确
   - 检查 API 密钥权限

2. **向量维度不匹配**
   - 确保 `sqvect.vector_dim` 与嵌入模型维度匹配
   - Ollama nomic-embed-text: 768
   - OpenAI text-embedding-3-small: 1536

3. **连接超时**
   - 调整 `timeout` 设置
   - 检查网络连接或本地服务状态

4. **模型不存在**
   - 确保指定的模型在提供商中可用
   - Ollama: 使用 `ollama list` 查看已拉取模型

### 调试

启用调试日志：
```bash
./rago serve --log-level debug
```

检查提供商健康状态：
```bash
curl http://localhost:7127/api/health
```

## 配置示例文件

项目提供了多个配置示例：

- `config.toml` - 默认配置（Ollama + 后向兼容）
- `examples/config-openai.toml` - 纯 OpenAI 配置
- `examples/config-mixed.toml` - 混合提供商配置  
- `examples/config-local-openai.toml` - 本地 OpenAI 兼容服务配置