# RAGO 工具调用功能设计建议

## 概述

本文档提供在RAGO项目中集成工具调用(Tool/Function Calling)功能的架构设计建议。工具调用将使LLM能够调用外部函数来获取实时信息、执行计算或与外部系统交互，大大增强RAG系统的能力。

## 当前架构分析

### 现有优势
- **模块化设计**：`internal/domain`、`internal/processor`等清晰的模块划分
- **接口抽象**：`Generator`、`Embedder`等接口便于扩展
- **双重接口**：CLI和HTTP API支持
- **流式支持**：已支持流式生成，便于集成工具调用的实时反馈

### 集成点
- **LLM接口扩展**：`internal/llm/ollama.go`需要支持工具调用
- **处理器增强**：`internal/processor/service.go`需要处理工具调用逻辑
- **领域模型扩展**：`internal/domain/types.go`需要新的工具相关类型

## 架构设计

### 1. 核心组件设计

#### 1.1 工具定义系统
```go
// internal/tools/types.go
type Tool interface {
    Name() string
    Description() string
    Parameters() ToolParameters
    Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error)
    Validate(args map[string]interface{}) error
}

type ToolParameters struct {
    Type       string                 `json:"type"`
    Properties map[string]ToolParam   `json:"properties"`
    Required   []string               `json:"required"`
}

type ToolParam struct {
    Type        string      `json:"type"`
    Description string      `json:"description"`
    Enum        []string    `json:"enum,omitempty"`
    Default     interface{} `json:"default,omitempty"`
}

type ToolResult struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}
```

#### 1.2 工具注册管理器
```go
// internal/tools/registry.go
type Registry struct {
    tools   map[string]Tool
    mu      sync.RWMutex
    config  *ToolConfig
}

type ToolConfig struct {
    EnabledTools    []string          `toml:"enabled_tools"`
    ToolConfigs     map[string]Config `toml:"tool_configs"`
    SecurityLevel   string            `toml:"security_level"` // strict, normal, permissive
    MaxConcurrency  int               `toml:"max_concurrency"`
    Timeout         time.Duration     `toml:"timeout"`
}

func (r *Registry) Register(tool Tool) error
func (r *Registry) Get(name string) (Tool, bool)
func (r *Registry) List() []ToolInfo
func (r *Registry) IsEnabled(name string) bool
```

#### 1.3 工具执行引擎
```go
// internal/tools/executor.go
type Executor struct {
    registry    *Registry
    limiter     *rate.Limiter
    semaphore   chan struct{}
}

type ExecutionContext struct {
    RequestID   string
    UserID      string
    SessionID   string
    Permissions []string
}

func (e *Executor) Execute(ctx context.Context, execCtx *ExecutionContext, 
                          toolName string, args map[string]interface{}) (*ToolResult, error)
```

### 2. 内置工具建议

#### 2.1 基础工具集
```go
// internal/tools/builtin/

// 1. 搜索工具
type SearchTool struct{} // 搜索知识库
type WebSearchTool struct{} // 网页搜索

// 2. 计算工具  
type CalculatorTool struct{} // 数学计算
type DateTimeTool struct{} // 日期时间处理

// 3. 文件操作工具
type FileReadTool struct{} // 读取文件
type FileListTool struct{} // 列出文件

// 4. 网络工具
type HTTPRequestTool struct{} // HTTP请求
type PingTool struct{} // 网络连通性测试

// 5. 文档操作工具
type DocumentCountTool struct{} // 统计文档数量
type DocumentInfoTool struct{} // 获取文档信息
```

#### 2.2 RAG特定工具
```go
// RAG系统专用工具
type RAGSearchTool struct {
    processor *processor.Service
}

type DocumentManagementTool struct {
    processor *processor.Service
}

type MetadataQueryTool struct {
    processor *processor.Service
}
```

### 3. LLM接口扩展

#### 3.1 Generator接口扩展
```go
// internal/domain/types.go
type ToolCallGenerator interface {
    Generator
    GenerateWithTools(ctx context.Context, prompt string, tools []ToolDefinition, 
                     opts *GenerationOptions) (*GenerationResult, error)
    StreamWithTools(ctx context.Context, prompt string, tools []ToolDefinition,
                   opts *GenerationOptions, callback ToolCallCallback) error
}

type ToolDefinition struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    Parameters  ToolParameters `json:"parameters"`
}

type ToolCall struct {
    ID       string                 `json:"id"`
    Type     string                 `json:"type"` // "function"
    Function FunctionCall           `json:"function"`
}

type FunctionCall struct {
    Name      string                 `json:"name"`
    Arguments map[string]interface{} `json:"arguments"`
}

type GenerationResult struct {
    Content   string     `json:"content"`
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`
    Finished  bool       `json:"finished"`
}
```

#### 3.2 Ollama服务扩展
```go
// internal/llm/ollama.go
func (s *OllamaService) GenerateWithTools(ctx context.Context, prompt string, 
                                         tools []ToolDefinition, opts *GenerationOptions) (*GenerationResult, error) {
    // 构建包含工具定义的请求
    // 处理LLM返回的工具调用
    // 返回结构化结果
}

func (s *OllamaService) StreamWithTools(ctx context.Context, prompt string,
                                       tools []ToolDefinition, opts *GenerationOptions,
                                       callback ToolCallCallback) error {
    // 流式处理工具调用
    // 实时反馈工具执行结果
}
```

### 4. 处理器增强

#### 4.1 查询处理器扩展
```go
// internal/processor/service.go
type Service struct {
    // 现有字段...
    toolRegistry *tools.Registry
    toolExecutor *tools.Executor
}

func (s *Service) QueryWithTools(ctx context.Context, req QueryWithToolsRequest) (QueryWithToolsResponse, error) {
    // 1. 执行混合搜索获取相关文档
    // 2. 构建包含工具定义的提示
    // 3. 调用LLM生成回答和工具调用
    // 4. 执行工具调用
    // 5. 将工具结果反馈给LLM
    // 6. 生成最终回答
}

type QueryWithToolsRequest struct {
    QueryRequest
    ToolsEnabled  bool     `json:"tools_enabled"`
    AllowedTools  []string `json:"allowed_tools,omitempty"`
    MaxToolCalls  int      `json:"max_tool_calls"`
}

type QueryWithToolsResponse struct {
    QueryResponse
    ToolCalls     []ExecutedToolCall `json:"tool_calls,omitempty"`
    ToolsUsed     []string           `json:"tools_used,omitempty"`
}

type ExecutedToolCall struct {
    ToolCall
    Result    interface{} `json:"result"`
    Elapsed   string      `json:"elapsed"`
    Success   bool        `json:"success"`
    Error     string      `json:"error,omitempty"`
}
```

### 5. API接口设计

#### 5.1 HTTP API端点
```go
// api/handlers/tools.go
type ToolsHandler struct {
    registry *tools.Registry
}

// GET /api/tools - 列出可用工具
func (h *ToolsHandler) ListTools(c *gin.Context) {}

// GET /api/tools/{name} - 获取工具详情
func (h *ToolsHandler) GetTool(c *gin.Context) {}

// POST /api/tools/{name}/execute - 直接执行工具
func (h *ToolsHandler) ExecuteTool(c *gin.Context) {}

// POST /api/query/with-tools - 带工具调用的查询
func (h *QueryHandler) QueryWithTools(c *gin.Context) {}

// POST /api/query/with-tools/stream - 带工具调用的流式查询
func (h *QueryHandler) StreamQueryWithTools(c *gin.Context) {}
```

#### 5.2 CLI命令扩展
```bash
# 工具管理命令
rago tools list                    # 列出可用工具
rago tools info <tool-name>       # 查看工具详情
rago tools execute <tool-name>    # 执行工具

# 带工具调用的查询
rago query --tools "问今天天气如何？"
rago query --tools --allowed-tools "weather,search" "查询并计算..."
rago query --interactive --tools  # 交互模式启用工具
```

### 6. 配置管理

#### 6.1 配置文件扩展
```toml
# config.toml
[tools]
enabled = true
max_concurrent_calls = 3
call_timeout = "30s"
security_level = "normal" # strict, normal, permissive

[tools.builtin]
search = { enabled = true }
calculator = { enabled = true }
datetime = { enabled = true }
web_search = { enabled = false, api_key = "" }

[tools.custom]
# 自定义工具配置
weather = { 
    enabled = true, 
    api_key = "${WEATHER_API_KEY}",
    base_url = "https://api.openweathermap.org"
}
```

#### 6.2 环境变量
```bash
export RAGO_TOOLS_ENABLED=true
export RAGO_TOOLS_SECURITY_LEVEL=normal
export WEATHER_API_KEY=your_api_key
```

### 7. 安全考虑

#### 7.1 权限控制
```go
type ToolPermission struct {
    ToolName    string   `json:"tool_name"`
    Actions     []string `json:"actions"`     // read, write, execute
    Resources   []string `json:"resources"`   // file paths, URLs等
}

type SecurityPolicy struct {
    DefaultDeny     bool             `json:"default_deny"`
    Permissions     []ToolPermission `json:"permissions"`
    RateLimits      map[string]int   `json:"rate_limits"`
    TimeoutSeconds  int              `json:"timeout_seconds"`
}
```

#### 7.2 执行沙箱
```go
type SandboxConfig struct {
    AllowNetworkAccess  bool     `json:"allow_network_access"`
    AllowFileAccess     bool     `json:"allow_file_access"`
    AllowedDirectories  []string `json:"allowed_directories"`
    AllowedDomains      []string `json:"allowed_domains"`
    MaxMemoryMB         int      `json:"max_memory_mb"`
    MaxExecutionTime    int      `json:"max_execution_time"`
}
```

### 8. 实现步骤建议

#### 阶段1: 基础架构
1. 定义工具接口和基础类型
2. 实现工具注册管理器
3. 创建基础的工具执行引擎
4. 扩展配置系统

#### 阶段2: LLM集成
1. 扩展Generator接口支持工具调用
2. 更新OllamaService实现工具调用
3. 实现工具调用的请求/响应处理
4. 添加流式工具调用支持

#### 阶段3: 内置工具
1. 实现基础工具集(计算器、日期时间等)
2. 实现RAG专用工具
3. 添加网络工具(可选)
4. 添加文件操作工具

#### 阶段4: API集成
1. 扩展查询处理器支持工具调用
2. 添加新的API端点
3. 更新CLI命令支持工具调用
4. 实现流式工具调用API

#### 阶段5: 安全和优化
1. 实现权限控制系统
2. 添加执行沙箱
3. 实现速率限制
4. 性能优化和监控

### 9. 使用示例

#### 9.1 基础查询 + 工具调用
```json
POST /api/query/with-tools
{
  "query": "今天是几月几号？上海的天气如何？",
  "tools_enabled": true,
  "allowed_tools": ["datetime", "weather"],
  "max_tool_calls": 3,
  "temperature": 0.7
}
```

响应:
```json
{
  "answer": "今天是2024年1月15日。上海今天多云，气温8-12°C，湿度65%。",
  "sources": [...],
  "tool_calls": [
    {
      "id": "call_1",
      "function": {
        "name": "datetime",
        "arguments": {"action": "current_date"}
      },
      "result": "2024-01-15",
      "success": true,
      "elapsed": "2ms"
    },
    {
      "id": "call_2", 
      "function": {
        "name": "weather",
        "arguments": {"city": "Shanghai"}
      },
      "result": {"temperature": "8-12°C", "condition": "cloudy"},
      "success": true,
      "elapsed": "150ms"
    }
  ]
}
```

#### 9.2 知识库增强查询
```json
POST /api/query/with-tools
{
  "query": "根据我的文档计算一下总共有多少个PDF文件，然后搜索关于'机器学习'的内容",
  "tools_enabled": true,
  "allowed_tools": ["document_count", "rag_search"],
  "temperature": 0.3
}
```

### 10. 测试策略

#### 10.1 单元测试
- 工具接口实现测试
- 工具执行引擎测试
- 权限控制测试
- LLM集成测试

#### 10.2 集成测试
- 端到端工具调用流程测试
- 多工具协作测试
- 错误处理和恢复测试
- 性能和并发测试

#### 10.3 安全测试
- 权限绕过测试
- 注入攻击测试
- 资源消耗测试
- 网络隔离测试

### 11. 监控和日志

#### 11.1 关键指标
- 工具调用频率和成功率
- 工具执行时间分布
- 错误类型统计
- 资源使用情况

#### 11.2 日志设计
```go
type ToolCallLog struct {
    RequestID    string                 `json:"request_id"`
    ToolName     string                 `json:"tool_name"`
    Arguments    map[string]interface{} `json:"arguments"`
    Result       interface{}            `json:"result,omitempty"`
    Success      bool                   `json:"success"`
    Error        string                 `json:"error,omitempty"`
    Elapsed      time.Duration          `json:"elapsed"`
    Timestamp    time.Time              `json:"timestamp"`
}
```

## 总结

这个设计提供了一个完整、可扩展的工具调用架构，能够：

1. **无缝集成**到现有RAGO架构中
2. **保持模块化**设计原则
3. **支持多种工具类型**和执行模式
4. **提供完整的安全控制**
5. **兼容现有API**和CLI接口
6. **支持流式和批量**处理
7. **便于扩展**新工具和功能

通过分阶段实施，可以逐步引入工具调用功能，在每个阶段都能提供价值，同时保持系统的稳定性和可维护性。