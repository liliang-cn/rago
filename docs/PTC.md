# PTC (Programmatic Tool Calling) 技术文档

PTC 是 AgentGo 的创新功能，允许 LLM 生成 JavaScript 代码来编排工具调用，而非传统的 JSON 参数方式。

## 架构总览

```
┌─────────────────────────────────────────────────────────────────────┐
│                           用户请求                                   │
│   "Which team members exceeded their Q3 travel budget?"             │
└──────────────────────────────┬──────────────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Agent Service (ChatWithPTC)                      │
│  1. 检测 PTC 是否启用                                                │
│  2. 构建 PTC 专用 System Prompt (含可用工具列表)                      │
│  3. 调用 LLM                                                         │
└──────────────────────────────┬──────────────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      LLM 生成响应                                    │
│  不返回 JSON 参数，而是生成 JavaScript 代码：                          │
│                                                                     │
│  <code>                                                             │
│  const team = callTool('get_team_members', { dept: 'eng' });       │
│  const budget = callTool('get_budget', { quarter: 'Q3' });         │
│  const perPerson = budget.travel / team.count;                     │
│  return team.members.filter(m => m.spent > perPerson);             │
│  </code>                                                            │
└──────────────────────────────┬──────────────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    PTCIntegration 处理                               │
│  1. IsCodeResponse() - 检测是否包含代码                              │
│  2. ExtractCode() - 提取 <code> 标签内的代码                         │
│  3. ExecuteCode() - 在沙箱中执行                                     │
└──────────────────────────────┬──────────────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      PTC Service                                     │
│  - 管理沙箱运行时 (Goja/Wazero)                                      │
│  - 安全验证代码                                                       │
│  - 注册可用工具                                                       │
│  - 执行超时控制                                                       │
└──────────────────────────────┬──────────────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                  JavaScript Sandbox (Goja Runtime)                  │
│                                                                     │
│  内置函数:                                                           │
│  - callTool(name, args) → 调用工具并返回结果                          │
│  - console.log(...) → 调试日志                                       │
│  - JSON.parse/stringify → JSON 处理                                 │
│  - sleep(ms) → 延迟                                                  │
│                                                                     │
│  执行过程:                                                           │
│  vm.RunString(code) → 解析 JS → 遇到 callTool → 调用 ToolHandler     │
└──────────────────────────────┬──────────────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      AgentGoRouter                                      │
│  路由工具调用到实际处理器:                                            │
│                                                                     │
│  1. MCP Tools → mcpService.CallTool()                               │
│  2. Skills → skillsService.Execute()                                │
│  3. RAG → ragProcessor.Query()                                      │
│  4. Custom → 注册的自定义 Handler                                    │
└──────────────────────────────┬──────────────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     实际工具执行                                      │
│  get_team_members → 返回 { members: [...], count: 3 }               │
│  get_budget → 返回 { travel: 8000, total: 50000 }                   │
│  get_expenses → 返回 { expenses: [...] }                            │
└──────────────────────────────┬──────────────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    ExecutionResult                                   │
│  {                                                                  │
│    "success": true,                                                 │
│    "return_value": [{ "name": "Carol", "spent": 3200, ... }],      │
│    "tool_calls": [                                                  │
│      { "tool_name": "get_team_members", "duration": "5ms" },       │
│      { "tool_name": "get_budget", "duration": "3ms" },             │
│      { "tool_name": "get_expenses", "duration": "4ms" }            │
│    ]                                                                │
│  }                                                                  │
└──────────────────────────────┬──────────────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    格式化返回给 LLM                                   │
│  PTCResult.FormatForLLM():                                          │
│  "Code execution completed.                                         │
│   **Status:** Success                                               │
│   **Return Value:** [{ name: Carol, spent: 3200, ... }]            │
│   **Tool Calls (5):**                                               │
│   - get_team_members ✓                                              │
│   - get_budget ✓                                                    │
│   - get_expenses ✓ ..."                                             │
└─────────────────────────────────────────────────────────────────────┘
```

## 核心组件

### 1. PTCIntegration (`pkg/agent/ptc_integration.go`)

Agent 与 PTC 的桥梁：

```go
type PTCIntegration struct {
    service *ptc.Service      // PTC 核心服务
    config  *PTCConfig        // 配置 (超时、工具白名单等)
    router  *ptc.AgentGoRouter   // 工具路由器
}
```

**关键方法：**

| 方法 | 说明 |
|------|------|
| `IsCodeResponse()` | 检测 LLM 响应是否包含可执行代码 |
| `ExtractCode()` | 从响应中提取 JavaScript 代码 |
| `ExecuteCode()` | 在沙箱中执行代码 |
| `GetPTCSystemPrompt()` | 生成 PTC 专用 System Prompt |
| `GetAvailableCallTools()` | 获取 callTool() 可调用的工具列表 |
| `GetPTCTools()` | 返回 `execute_javascript` 工具定义 |

### 2. PTC Service (`pkg/ptc/service.go`)

核心执行服务：

```go
func (s *Service) Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResult, error) {
    // 1. 验证请求
    s.validateRequest(req)

    // 2. 安全检查代码
    s.validateCode(req.Code)

    // 3. 过滤允许的工具
    filteredTools := s.filterTools(req.Tools)

    // 4. 注册工具处理器
    runtime.RegisterTool(toolName, handler)

    // 5. 在沙箱中执行
    result := runtime.Execute(ctx, req)

    return result, nil
}
```

### 3. Goja Runtime (`pkg/ptc/runtime/goja/runtime.go`)

JavaScript 沙箱实现，基于 [Goja](https://github.com/dop251/goja) 解释器：

```go
func (r *Runtime) Execute(ctx context.Context, req *ptc.ExecutionRequest) {
    vm := goja.New()  // 创建 JS 虚拟机

    // 注册内置函数
    vm.Set("callTool", func(call goja.FunctionCall) goja.Value {
        // 1. 检查调用次数限制
        if state.callCount > state.maxCalls {
            panic("maximum tool calls exceeded")
        }

        // 2. 获取工具处理器
        handler := state.tools[toolName]

        // 3. 调用实际工具
        result, err := handler(ctx, args)

        // 4. 记录调用
        *state.toolCalls = append(*state.toolCalls, toolRecord)

        // 5. 返回结果给 JS
        return vm.ToValue(result)
    })

    // 执行代码
    vm.RunString(wrappedCode)
}
```

**内置函数：**

| 函数 | 说明 |
|------|------|
| `callTool(name, args)` | 调用工具并返回结果 |
| `console.log(...)` | 调试日志输出 |
| `console.error(...)` | 错误日志输出 |
| `JSON.parse(s)` | 解析 JSON 字符串 |
| `JSON.stringify(v)` | 序列化为 JSON |
| `sleep(ms)` | 延迟执行 |
| `toolExists(name)` | 检查工具是否存在 |
| `listTools()` | 列出可用工具 |

### 4. AgentGoRouter (`pkg/ptc/router.go`)

工具路由器，将 `callTool()` 调用路由到实际服务：

```go
func (r *AgentGoRouter) Route(ctx context.Context, toolName string, args map[string]interface{}) {
    // 1. 尝试已注册的处理器
    if handler, ok := r.handlers[toolName]; ok {
        return handler(ctx, args)
    }

    // 2. 尝试 MCP 服务
    if svc, ok := r.mcpService.(callTooler); ok {
        return svc.CallTool(ctx, toolName, args)
    }

    // 3. 尝试 Skills 服务
    return r.callSkill(ctx, skillID, args)
}
```

**支持的工具来源：**

- **MCP Tools** - 通过 MCP 协议连接的外部工具
- **Skills** - AgentGo 内置的技能系统
- **RAG** - 知识库查询工具
- **Custom** - 用户注册的自定义处理器

## 与传统 Tool Calling 的对比

| 特性 | 传统 Tool Calling | PTC |
|------|------------------|-----|
| LLM 输出 | JSON 参数 | JavaScript 代码 |
| 并行调用 | 需要多次 LLM 轮次 | 单次执行多次调用 |
| 数据处理 | LLM 处理 | 沙箱内处理 |
| 复杂逻辑 | 难以表达 | 完整编程能力 |
| 上下文占用 | 每次返回完整数据 | 可预处理后返回 |
| 循环/条件 | 不支持 | 完全支持 |
| 错误处理 | 需要重试 | 代码内处理 |

## 使用示例

### 基本使用

```go
// 创建 PTC 集成
ptcConfig := agent.PTCConfig{
    Enabled:      true,
    MaxToolCalls: 20,
    Timeout:      30 * time.Second,
    Runtime:      "goja", // 或 "wazero"
}

ptcIntegration, _ := agent.NewPTCIntegration(ptcConfig, router)
agentService.SetPTC(ptcIntegration)

// 使用 PTC 模式聊天
result, _ := agentService.ChatWithPTC(ctx, "比较三个城市的旅行预算")
```

### CLI 使用

```bash
# 启用 PTC 模式运行 agent
./agentgo agent run "分析 Q3 预算" --ptc
```

### 代码示例

用户请求：
```
"比较东京、大阪、曼谷的 3 天旅行预算"
```

LLM 生成的代码：
```javascript
<code>
const cities = ['Tokyo', 'Osaka', 'Bangkok'];
return cities.map(city => {
  const weather = callTool('get_weather', { city });
  const flights = callTool('search_flights', { to: city });
  const hotels = callTool('search_hotels', { city, max_price: 150 });

  return {
    city,
    weather: weather.forecast.slice(0, 3),
    cheapest_flight: flights[0].price,
    avg_hotel: hotels.reduce((s, h) => s + h.price, 0) / hotels.length,
    total: flights[0].price * 2 + (hotels[0].price * 3) + 150
  };
});
</code>
```

执行过程：
```
[callTool] get_weather { city: 'Tokyo' } → 5ms
[callTool] search_flights { to: 'Tokyo' } → 8ms
[callTool] search_hotels { city: 'Tokyo', max_price: 150 } → 6ms
[callTool] get_weather { city: 'Osaka' } → 4ms
...
```

返回结果：
```json
{
  "success": true,
  "return_value": [
    { "city": "Tokyo", "weather": [...], "total": 967 },
    { "city": "Osaka", "weather": [...], "total": 892 },
    { "city": "Bangkok", "weather": [...], "total": 654 }
  ],
  "tool_calls": 12
}
```

## 安全机制

### 1. 调用次数限制

防止无限循环或恶意代码消耗资源：

```go
PTCConfig{
    MaxToolCalls: 20,  // 默认最多 20 次 callTool()
}
```

### 2. 执行超时

限制代码执行时间：

```go
PTCConfig{
    Timeout: 30 * time.Second,  // 默认 30 秒
}
```

### 3. 工具访问控制

白名单/黑名单机制：

```go
PTCConfig{
    AllowedTools: []string{"get_weather", "search_flights"},  // 只允许这些工具
    BlockedTools: []string{"delete_file", "execute_command"}, // 禁止这些工具
}
```

### 4. 沙箱隔离

- 无法访问文件系统
- 无法访问网络（除了通过工具）
- 无法访问环境变量
- 无法导入外部模块

### 5. 代码验证

在执行前进行静态分析，检测危险模式。

## 配置选项

```go
type PTCConfig struct {
    // 是否启用 PTC
    Enabled bool `json:"enabled"`

    // 单次执行最大工具调用次数
    MaxToolCalls int `json:"max_tool_calls"`

    // 执行超时时间
    Timeout time.Duration `json:"timeout"`

    // 允许的工具白名单（空=全部允许）
    AllowedTools []string `json:"allowed_tools"`

    // 禁止的工具黑名单
    BlockedTools []string `json:"blocked_tools"`

    // 运行时类型: "goja" 或 "wazero"
    Runtime string `json:"runtime"`
}
```

## 运行时选择

### Goja (推荐)

- 纯 Go 实现的 JavaScript 解释器
- 启动快，内存占用小
- 适合大多数场景

```go
PTCConfig{Runtime: "goja"}
```

### Wazero

- WebAssembly 运行时
- 更强的隔离性
- 适合需要更高安全性的场景

```go
PTCConfig{Runtime: "wazero"}
```

## 相关文件

```
pkg/
├── agent/
│   ├── ptc_integration.go    # Agent 与 PTC 集成
│   └── ptc_sanitise.go       # 代码清理和安全处理
├── ptc/
│   ├── types.go              # 核心类型定义
│   ├── service.go            # PTC 服务
│   ├── router.go             # 工具路由器
│   ├── config.go             # 配置
│   ├── errors.go             # 错误类型
│   ├── runtime/
│   │   ├── goja/             # Goja 运行时
│   │   │   ├── runtime.go    # 运行时实现
│   │   │   └── bindings.go   # JS 绑定
│   │   └── wazero/           # Wazero 运行时
│   ├── security/             # 安全验证
│   └── store/                # 执行历史存储
```

## 示例代码

完整示例请参考：

- `examples/ptc/custom_tools/main.go` - 自定义工具示例
- `examples/ptc/project_tracker/main.go` - 项目追踪示例
- `examples/ptc/weather_planner/main.go` - 旅行规划示例
