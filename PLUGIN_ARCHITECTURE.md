# RAGO 工具调用系统 - 插件化设计

## 概述

RAGO的工具调用系统采用插件化架构，支持：

- **内置工具**: 核心功能工具，编译时集成
- **插件工具**: 动态加载的外部工具，运行时集成
- **灵活配置**: 支持工具的启用/禁用和精细配置
- **安全控制**: 插件白名单/黑名单，权限控制

## 插件化架构优势

### ✅ **扩展性**
- 无需修改核心代码即可添加新工具
- 支持第三方开发者贡献工具
- 热插拔式的工具管理

### ✅ **安全性**  
- 插件沙箱隔离
- 白名单/黑名单控制
- 权限粒度管控

### ✅ **维护性**
- 模块化设计，问题隔离
- 独立的插件版本管理
- 便于调试和测试

## 配置示例

### 完整配置 (`config.toml`)

```toml
[tools]
enabled = true
max_concurrent_calls = 3
call_timeout = "30s"
security_level = "normal"  # strict, normal, permissive
enabled_tools = ["datetime", "sql_query", "rag_search"]

# 内置工具配置
[tools.builtin]
datetime = { enabled = true }
sql_query = { 
    enabled = true,
    max_connections = 5,
    allowed_drivers = ["postgres", "mysql", "sqlite"]
}
rag_search = { enabled = true }
file_read = { 
    enabled = false,  # 默认禁用文件读取
    allowed_paths = ["./documents/*", "/safe/data/*"]
}

# 速率限制
[tools.rate_limit]
calls_per_minute = 30
calls_per_hour = 300
burst_size = 5

# 插件系统配置
[tools.plugins]
enabled = true
auto_load = true
plugin_paths = ["./plugins", "./tools/plugins", "/usr/local/rago/plugins"]

# 插件安全控制
whitelist = []  # 空表示允许所有（除黑名单外）
blacklist = ["dangerous_plugin"]  # 明确禁止的插件

# 插件特定配置
[tools.plugins.configs]

[tools.plugins.configs.weather_plugin]
api_key = "${WEATHER_API_KEY}"
base_url = "https://api.openweathermap.org/data/2.5"
timeout = "10s"

[tools.plugins.configs.database_plugin]
connection_string = "${DATABASE_URL}"
max_connections = 10
query_timeout = "30s"
```

## 插件开发指南

### 1. 创建插件结构

```go
// your_plugin/main.go
package main

import (
    "context"
    "fmt"
    "github.com/liliang-cn/rago/internal/tools"
)

// 实现你的工具
type WeatherTool struct {
    apiKey  string
    baseURL string
}

func (t *WeatherTool) Name() string {
    return "weather_query"
}

func (t *WeatherTool) Description() string {
    return "Get current weather information for a city"
}

func (t *WeatherTool) Parameters() tools.ToolParameters {
    return tools.ToolParameters{
        Type: "object",
        Properties: map[string]tools.ToolParameter{
            "city": {
                Type:        "string",
                Description: "City name to get weather for",
            },
            "units": {
                Type:        "string",
                Description: "Temperature units (celsius/fahrenheit)",
                Enum:        []string{"celsius", "fahrenheit"},
                Default:     "celsius",
            },
        },
        Required: []string{"city"},
    }
}

func (t *WeatherTool) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
    city, ok := args["city"].(string)
    if !ok {
        return &tools.ToolResult{
            Success: false,
            Error:   "city parameter is required",
        }, nil
    }
    
    // 实现你的天气查询逻辑
    weather := fmt.Sprintf("Weather in %s: 22°C, Sunny", city)
    
    return &tools.ToolResult{
        Success: true,
        Data:    weather,
    }, nil
}

func (t *WeatherTool) Validate(args map[string]interface{}) error {
    if _, ok := args["city"]; !ok {
        return fmt.Errorf("city parameter is required")
    }
    return nil
}

// 插件主体
type WeatherPlugin struct {
    tools []tools.Tool
}

func (p *WeatherPlugin) Name() string {
    return "weather-plugin"
}

func (p *WeatherPlugin) Version() string {
    return "1.0.0"
}

func (p *WeatherPlugin) Description() string {
    return "Weather information plugin"
}

func (p *WeatherPlugin) Tools() []tools.Tool {
    return p.tools
}

func (p *WeatherPlugin) Initialize(config map[string]interface{}) error {
    apiKey, _ := config["api_key"].(string)
    baseURL, _ := config["base_url"].(string)
    
    if baseURL == "" {
        baseURL = "https://api.openweathermap.org/data/2.5"
    }
    
    weatherTool := &WeatherTool{
        apiKey:  apiKey,
        baseURL: baseURL,
    }
    
    p.tools = []tools.Tool{weatherTool}
    return nil
}

func (p *WeatherPlugin) Cleanup() error {
    p.tools = nil
    return nil
}

// 导出插件实例
var Plugin tools.ToolPlugin = &WeatherPlugin{}
```

### 2. 编译插件

```bash
# 编译为共享库
go build -buildmode=plugin -o weather_plugin.so main.go

# 放置到插件目录
cp weather_plugin.so ./plugins/
```

### 3. 配置插件

```toml
# config.toml
[tools.plugins.configs.weather_plugin]
api_key = "your_weather_api_key"
base_url = "https://api.openweathermap.org/data/2.5"
```

## 使用示例

### CLI 命令

```bash
# 列出所有工具（包括插件）
rago tools list

# 查看特定工具信息
rago tools info weather_query

# 执行插件工具
rago tools execute weather_query '{"city": "Shanghai"}'

# 带工具调用的查询
rago query --tools "今天上海的天气怎么样？"
```

### API 调用

```bash
# 列出插件
GET /api/plugins

# 插件详情
GET /api/plugins/weather-plugin

# 工具调用查询
POST /api/query/with-tools
{
  "query": "What's the weather like in Tokyo?",
  "tools_enabled": true,
  "allowed_tools": ["weather_query", "datetime"]
}
```

## 安全考虑

### 插件验证
- 数字签名验证（计划中）
- 插件来源验证
- 恶意代码检测

### 权限控制
- 文件系统访问限制
- 网络访问控制  
- 系统资源限制

### 运行时安全
- 插件沙箱隔离
- 执行超时控制
- 内存使用限制

## 内置工具 vs 插件工具

| 特性 | 内置工具 | 插件工具 |
|------|----------|----------|
| 性能 | 最优 | 良好 |
| 安全性 | 最高 | 可控 |
| 扩展性 | 需重编译 | 动态加载 |
| 维护性 | 集中 | 分散 |
| 适用场景 | 核心功能 | 专用功能 |

## 开发最佳实践

1. **错误处理**: 插件应优雅处理所有错误情况
2. **资源管理**: 正确实现 Cleanup 方法释放资源
3. **配置验证**: 在 Initialize 中验证必需的配置参数
4. **文档完善**: 提供清晰的工具描述和参数说明
5. **测试覆盖**: 为插件工具编写完整的单元测试

## 故障排除

### 常见问题

1. **插件加载失败**
   - 检查 .so 文件是否存在于配置的路径中
   - 确认插件导出了 `Plugin` 符号
   - 查看日志中的详细错误信息

2. **工具注册失败**
   - 检查工具名称是否唯一
   - 验证 Parameters 结构是否正确
   - 确认必需的参数已定义

3. **权限被拒绝**
   - 检查插件是否在白名单中
   - 确认插件未在黑名单中
   - 验证安全等级配置

### 调试技巧

```bash
# 启用详细日志
export RAGO_TOOLS_LOG_LEVEL=debug

# 检查插件状态
rago tools plugins list

# 测试特定工具
rago tools execute your_tool '{"param": "value"}' --debug
```

---

通过这个插件化设计，RAGO的工具系统具备了强大的扩展能力，既保证了核心功能的稳定性，又为社区贡献和定制化需求提供了灵活的解决方案。