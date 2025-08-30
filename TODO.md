# MCP Integration Roadmap for Rago (Updated)

## Overview

集成 MCP (Model Context Protocol) 到 rago 系统，使用**官方 MCP Go SDK v0.3.1**，以 [mcp-sqlite-server](https://github.com/liliang-cn/mcp-sqlite-server) 为起点，构建可扩展的外部工具生态系统。

## 技术栈更新

### 🔧 基于官方 SDK 的优势
- **官方支持**: 使用 [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) v0.3.1
- **标准实现**: 遵循官方协议规范，确保兼容性
- **维护保障**: Google 协作维护，持续更新
- **完整功能**: 包含 `mcp`、`jsonschema`、`jsonrpc` 三个核心包
- **传输抽象**: 支持 stdin/stdout、WebSocket 等多种传输方式

### 📦 依赖管理
```go
module github.com/liliang-cn/rago

require (
    github.com/modelcontextprotocol/go-sdk v0.3.1
    // ... existing dependencies
)
```

## Phase 1: MCP 客户端基础架构 (Week 1-2)

### 1.1 MCP SDK 集成
```
internal/mcp/
├── client.go          # 基于官方 SDK 的客户端封装
├── manager.go         # MCP 服务器管理器
├── transport.go       # 传输层抽象 (stdio/websocket)
├── registry.go        # MCP 工具注册表
├── types.go           # rago 特定的 MCP 类型
└── examples/          # 示例和测试
    ├── sqlite_test.go # SQLite 服务器测试
    └── basic_test.go  # 基础功能测试
```

**关键实现:**
```go
// 基于官方 SDK 的客户端封装
type Client struct {
    mcpClient  *mcp.Client
    serverInfo *mcp.ServerInfo
    transport  jsonrpc.Transport
    tools      map[string]*mcp.Tool
    logger     Logger
}

// 服务器管理器
type Manager struct {
    clients    map[string]*Client
    configs    map[string]*ServerConfig
    mutex      sync.RWMutex
}
```

**关键任务:**
- [ ] 集成官方 MCP Go SDK v0.3.1
- [ ] 实现基于 SDK 的客户端封装层
- [ ] 创建服务器生命周期管理（启动/停止/监控）
- [ ] 实现 stdio 和 WebSocket 传输支持
- [ ] 添加连接重试和错误恢复机制
- [ ] 实现工具发现和动态注册

### 1.2 配置系统设计
```toml
[mcp]
enabled = true
log_level = "info"
default_timeout = "30s"
max_concurrent_requests = 10
health_check_interval = "60s"

# SQLite MCP Server 配置
[[mcp.servers]]
name = "sqlite"
description = "SQLite database operations"
command = ["node", "./mcp-sqlite-server/dist/index.js"]
args = ["--allowed-dir", "./data", "--readonly", "false"]
working_dir = "./data"
env = { "DEBUG" = "mcp:*" }
auto_start = true
restart_on_failure = true
max_restarts = 5
restart_delay = "5s"

# 文件系统 MCP Server 示例
[[mcp.servers]]
name = "filesystem"
description = "File system operations"  
command = ["python", "-m", "mcp.server.filesystem"]
args = ["--root", "./workspace"]
auto_start = false
capabilities = ["read", "write", "list"]
```

### 1.3 工具系统集成设计
```go
// 扩展现有工具接口以支持 MCP
type MCPTool struct {
    client     *Client
    serverName string
    toolName   string
    schema     *jsonschema.Schema
}

func (t *MCPTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
    // 调用 MCP 服务器工具
    result, err := t.client.CallTool(ctx, t.toolName, args)
    if err != nil {
        return nil, fmt.Errorf("MCP tool call failed: %w", err)
    }
    
    return &ToolResult{
        Success: true,
        Data:    result,
    }, nil
}
```

**关键任务:**
- [ ] 扩展 `internal/tools/registry.go` 支持 MCP 工具
- [ ] 实现 MCP 工具到 rago 工具接口的适配器
- [ ] 添加工具参数验证和类型转换
- [ ] 实现工具调用的超时和重试机制
- [ ] 添加工具调用监控和日志

## Phase 2: SQLite Server 集成验证 (Week 2-3)

### 2.1 SQLite MCP 服务器集成
**基于官方 SDK 的实现:**
```go
func NewSQLiteClient(config *ServerConfig) (*Client, error) {
    // 使用官方 SDK 创建客户端
    transport := jsonrpc.NewStdioClientTransport(
        exec.Command(config.Command[0], config.Args...),
    )
    
    client := mcp.NewClient(transport)
    
    // 初始化连接并获取工具列表
    serverInfo, err := client.Initialize(ctx, &mcp.InitializeRequest{
        ProtocolVersion: "2024-11-05",
        Capabilities: &mcp.ClientCapabilities{
            Tools: &mcp.ToolsCapability{},
        },
        ClientInfo: &mcp.Implementation{
            Name:    "rago",
            Version: "1.0.0",
        },
    })
    
    return &Client{
        mcpClient:  client,
        serverInfo: serverInfo,
        transport:  transport,
    }, nil
}
```

**SQLite 工具映射:**
- [ ] `sqlite_query(sql: string, database?: string)` - 执行查询
- [ ] `sqlite_execute(sql: string, database?: string)` - 执行语句
- [ ] `sqlite_list_databases()` - 列出数据库
- [ ] `sqlite_create_database(name: string)` - 创建数据库
- [ ] `sqlite_list_tables(database?: string)` - 列出表
- [ ] `sqlite_describe_table(table: string, database?: string)` - 描述表结构

### 2.2 实际集成测试
**创建端到端测试:**
```go
func TestSQLiteMCPIntegration(t *testing.T) {
    // 启动 mcp-sqlite-server
    manager := NewManager(testConfig)
    client, err := manager.StartServer("sqlite")
    require.NoError(t, err)
    
    // 测试工具调用
    result, err := client.CallTool(ctx, "sqlite_query", map[string]interface{}{
        "sql": "SELECT name FROM sqlite_master WHERE type='table'",
    })
    require.NoError(t, err)
    
    // 验证结果
    assert.Contains(t, result.Content[0].Text, "table")
}
```

## Phase 3: 工具生态扩展 (Week 4-5)

### 3.1 更多 MCP 工具集成
**基于社区生态的工具:**
- [ ] **[mcp-server-filesystem](https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem)**: 文件系统操作
- [ ] **[mcp-server-git](https://github.com/modelcontextprotocol/servers/tree/main/src/git)**: Git 版本控制
- [ ] **[mcp-server-fetch](https://github.com/modelcontextprotocol/servers/tree/main/src/fetch)**: HTTP 请求
- [ ] **[mcp-server-postgres](https://github.com/modelcontextprotocol/servers/tree/main/src/postgres)**: PostgreSQL 数据库
- [ ] **[mcp-server-brave-search](https://github.com/modelcontextprotocol/servers/tree/main/src/brave-search)**: Web 搜索

### 3.2 工具发现和管理系统
```go
type ToolRegistry struct {
    servers map[string]*ServerInfo
    tools   map[string]*ToolInfo
    mutex   sync.RWMutex
}

type ToolInfo struct {
    ServerName  string                 `json:"server_name"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    InputSchema *jsonschema.Schema     `json:"input_schema"`
    Capabilities []string              `json:"capabilities"`
    LastUsed    time.Time              `json:"last_used"`
    UsageCount  int64                  `json:"usage_count"`
}
```

## Phase 4: 高级特性开发 (Week 6-7)

### 4.1 智能工具编排
```go
type ToolChain struct {
    Steps []ToolStep `json:"steps"`
    Name  string     `json:"name"`
}

type ToolStep struct {
    ServerName   string                 `json:"server_name"`
    ToolName     string                 `json:"tool_name"`
    Arguments    map[string]interface{} `json:"arguments"`
    Condition    string                 `json:"condition,omitempty"`    // 条件执行
    OnError      string                 `json:"on_error,omitempty"`     // 错误处理
    Parallel     bool                   `json:"parallel,omitempty"`     // 并行执行
}
```

### 4.2 监控和可观测性
```go
type MCPMetrics struct {
    ToolCalls       Counter   `metrics:"mcp_tool_calls_total"`
    ToolErrors      Counter   `metrics:"mcp_tool_errors_total"`
    ToolLatency     Histogram `metrics:"mcp_tool_duration_seconds"`
    ActiveServers   Gauge     `metrics:"mcp_active_servers"`
    ServerRestarts  Counter   `metrics:"mcp_server_restarts_total"`
}
```

## 实现细节和最佳实践

### 错误处理策略
```go
type MCPError struct {
    ServerName string `json:"server_name"`
    ToolName   string `json:"tool_name,omitempty"`
    Code       int    `json:"code"`
    Message    string `json:"message"`
    Details    string `json:"details,omitempty"`
}

func (e *MCPError) Error() string {
    if e.ToolName != "" {
        return fmt.Sprintf("MCP tool error [%s:%s]: %s", e.ServerName, e.ToolName, e.Message)
    }
    return fmt.Sprintf("MCP server error [%s]: %s", e.ServerName, e.Message)
}
```

### 安全考虑
```go
type SecurityConfig struct {
    AllowedCommands     []string          `toml:"allowed_commands"`
    AllowedDirectories  []string          `toml:"allowed_directories"`
    ResourceLimits      *ResourceLimits   `toml:"resource_limits"`
    SandboxEnabled      bool              `toml:"sandbox_enabled"`
    NetworkAccess       bool              `toml:"network_access"`
}

type ResourceLimits struct {
    MaxMemoryMB    int           `toml:"max_memory_mb"`
    MaxCPUPercent  float64       `toml:"max_cpu_percent"`
    MaxExecutionTime time.Duration `toml:"max_execution_time"`
}
```

## 技术风险和缓解策略

### 1. SDK 版本兼容性
- **风险**: 官方 SDK 仍处于 pre-1.0，可能有破坏性变更
- **缓解**: 
  - 锁定特定版本 (v0.3.1)
  - 创建适配器层隔离版本变更
  - 定期跟踪上游更新

### 2. 进程管理复杂性
- **风险**: 多个外部进程的生命周期管理
- **缓解**:
  - 实现健壮的进程监控
  - 添加自动重启和故障恢复
  - 使用 context 管理超时和取消

### 3. 性能开销
- **风险**: JSON-RPC 序列化和进程间通信延迟
- **缓解**:
  - 实现工具调用结果缓存
  - 添加连接池和复用
  - 监控性能指标和优化热点

## 成功指标 (Updated)

### 技术指标
- [ ] 与官方 SDK 兼容性 100%
- [ ] MCP 工具调用成功率 > 99%
- [ ] 平均工具调用延迟 < 200ms
- [ ] 支持至少 5 种不同 MCP 服务器
- [ ] 服务器崩溃后自动恢复时间 < 30s

### 用户体验指标
- [ ] 工具调用错误率 < 1%
- [ ] 新工具集成时间 < 30 分钟
- [ ] 文档完整性评分 > 90%
- [ ] 开发者满意度 > 4.5/5

## 实施路线图 (Updated)

### 立即开始 (本周)
1. **环境搭建**: 集成官方 MCP Go SDK v0.3.1
2. **原型开发**: 创建基础的 MCP 客户端封装
3. **SQLite 测试**: 与 mcp-sqlite-server 建立连接

### 短期目标 (1个月)
1. **核心实现**: 完成 Phase 1-2 的所有功能
2. **测试验证**: 确保与多个 MCP 服务器稳定集成
3. **文档完善**: 提供详细的使用文档和示例

### 中期目标 (3个月)
1. **生态扩展**: 集成 5+ 不同类型的 MCP 服务器
2. **高级特性**: 实现工具编排和智能选择
3. **生产就绪**: 添加监控、日志、性能优化

### 长期愿景 (6个月)
1. **AI Agent 平台**: 完整的 AI Agent 能力
2. **社区贡献**: 开源 rago 的 MCP 集成经验
3. **标准制定**: 参与 MCP 协议的发展

---

**总结**: 基于官方 MCP Go SDK 的集成方案更加稳定可靠，降低了技术风险，同时保持了 rago 系统的灵活性和扩展性。这将使 rago 成为一个真正具有 AI Agent 能力的平台。