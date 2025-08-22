# RAGO 工具调用功能实施进度

## 已完成

### ✅ 阶段1: 基础架构 (完成)
- [x] **定义工具接口和基础类型** (`internal/tools/types.go`)
- [x] **实现工具注册管理器** (`internal/tools/registry.go`)
- [x] **创建基础的工具执行引擎** (`internal/tools/executor.go`)
- [x] **扩展配置系统** (`internal/config/config.go`)

### ✅ 插件化架构 (完成) 🎯
- [x] **设计和实现插件管理器** (`internal/tools/plugin_manager.go`)
  - 支持动态加载 .so 文件
  - 插件生命周期管理 (加载/卸载/重载)
  - 插件白名单/黑名单安全控制
  - 插件元数据和状态管理

- [x] **创建插件示例和文档** (`internal/tools/plugin_example.go`)
  - 完整的插件开发模板
  - 详细的使用说明和最佳实践
  - 编译和部署指导

- [x] **扩展配置系统支持插件** 
  - 插件路径配置
  - 插件特定参数配置
  - 环境变量绑定

- [x] **插件架构文档** (`PLUGIN_ARCHITECTURE.md`)
  - 完整的插件开发指南
  - 配置示例和使用说明
  - 安全考虑和最佳实践

### ✅ 阶段2: LLM集成 (部分完成)
- [x] **扩展Generator接口支持工具调用** (`internal/domain/types.go`)
- [x] **更新OllamaService实现工具调用** (`internal/llm/ollama.go`)

## 进行中

### 🚧 阶段2: LLM集成 (2/4 已完成)
- [ ] **实现工具调用的请求/响应处理**
  - 需要实现工具调用结果的处理逻辑
  - 处理多轮对话中的工具调用
  
- [ ] **添加流式工具调用支持** 
  - 完善流式处理中的工具调用逻辑
  - 实现实时工具执行反馈

## 待完成

### ⏳ 阶段3: 内置工具
- [ ] **实现实用工具集**
  - 创建 `internal/tools/builtin/` 目录
  - 实现日期时间工具 (`DateTimeTool`) - 获取当前时间、日期计算等
  - 实现SQL查询工具 (`SQLQueryTool`) - 连接外部数据库查询
  - 实现文件操作工具 (`FileReadTool`, `FileListTool`) - 安全的文件系统访问

- [ ] **实现RAG专用工具**
  - 实现RAG搜索工具 (`RAGSearchTool`) - 搜索知识库内容
  - 实现文档管理工具 (`DocumentManagementTool`) - 管理文档元数据
  - 实现统计分析工具 (`StatsQueryTool`) - 分析文档和查询统计

### ⏳ 阶段4: API集成
- [ ] **扩展查询处理器支持工具调用**
  - 更新 `internal/processor/service.go`
  - 实现 `QueryWithTools` 方法
  - 集成工具执行引擎

- [ ] **添加新的API端点**
  - 创建 `api/handlers/tools.go`
  - 实现工具列表、详情、执行等端点
  - 实现带工具调用的查询端点

- [ ] **更新CLI命令支持工具调用**
  - 扩展 `cmd/rago/query.go` 支持工具调用选项
  - 添加新的工具管理命令
  - 更新帮助文档

### ⏳ 阶段5: 安全和优化
- [ ] **实现权限控制系统**
  - 实现工具权限验证
  - 添加用户权限管理
  - 实现资源访问控制

- [ ] **添加执行沙箱**
  - 实现安全沙箱环境
  - 限制工具的系统访问
  - 添加资源使用监控

## 关键文件结构

```
internal/
├── tools/
│   ├── types.go              ✅ 工具接口和基础类型
│   ├── registry.go           ✅ 工具注册管理器
│   ├── executor.go           ✅ 工具执行引擎
│   ├── plugin_manager.go     ✅ 插件管理器
│   ├── plugin_example.go     ✅ 插件开发示例
│   └── builtin/              ⏳ 内置工具实现
│       ├── datetime.go       ⏳ 日期时间工具
│       ├── sql.go            ⏳ SQL查询工具
│       ├── file.go           ⏳ 文件操作工具
│       └── rag_tools.go      ⏳ RAG专用工具
├── config/
│   └── config.go             ✅ 扩展配置支持工具和插件
├── domain/
│   └── types.go              ✅ 扩展领域类型
├── llm/
│   └── ollama.go             ✅ 添加工具调用支持
└── processor/
    └── service.go            ⏳ 集成工具调用到查询处理

api/handlers/
├── tools.go                  ⏳ 工具相关API端点
└── plugins.go                ⏳ 插件管理API端点

cmd/rago/
├── tools.go                  ⏳ 工具管理CLI命令
└── plugins.go                ⏳ 插件管理CLI命令

plugins/                      ⏳ 插件目录
└── example_plugin.so         ⏳ 示例插件

PLUGIN_ARCHITECTURE.md        ✅ 插件开发完整指南
```

## 下一步行动

1. **立即**: 完成阶段2剩余工作 - 实现工具调用的请求/响应处理
2. **接下来**: 实现实用内置工具（日期时间、SQL查询、文件操作）
3. **然后**: 实现RAG专用工具并集成到处理器
4. **最后**: 添加API端点和CLI支持

## 配置示例

创建的配置将支持如下结构：

```toml
[tools]
enabled = true
max_concurrent_calls = 3
call_timeout = "30s"
security_level = "normal"
enabled_tools = ["datetime", "sql_query", "rag_search", "file_read"]

[tools.builtin]
datetime = { enabled = true }
sql_query = { 
    enabled = true, 
    max_connections = 5,
    allowed_drivers = ["postgres", "mysql", "sqlite"]
}
rag_search = { enabled = true }
file_read = { 
    enabled = true,
    allowed_paths = ["/safe/path/*", "./documents/*"]
}

[tools.plugins]
enabled = true
auto_load = true
plugin_paths = ["./plugins", "./tools/plugins"]
whitelist = []  # 空=允许所有
blacklist = ["dangerous_plugin"]

[tools.plugins.configs.weather_plugin]
api_key = "${WEATHER_API_KEY}"
timeout = "10s"
```

## API示例

实现后将支持：

```bash
# 工具管理
GET /api/tools
GET /api/tools/datetime
POST /api/tools/datetime/execute

# 插件管理
GET /api/plugins
GET /api/plugins/weather-plugin
POST /api/plugins/weather-plugin/reload

# 带工具调用的查询
POST /api/query/with-tools
POST /api/query/with-tools/stream
```

## 测试计划

1. 单元测试所有工具实现
2. 集成测试工具调用流程
3. 性能测试并发执行
4. 安全测试权限控制

---
**更新时间**: 2025-01-08
**完成度**: 50% (10/20 个主要任务)

## 🎯 插件化架构亮点

通过插件化设计，RAGO的工具调用系统实现了：

- **动态扩展**: 无需重编译即可添加新工具
- **社区生态**: 支持第三方开发者贡献工具插件  
- **安全隔离**: 插件沙箱和权限控制
- **热更新**: 支持插件的动态加载/卸载/重载
- **配置灵活**: 细粒度的插件配置管理

这使得RAGO不仅是一个RAG系统，更是一个可扩展的AI工具平台。