# RAGO 工具调用系统优化任务清单

## ✅ 已完成

### 阶段1: 基础架构 (完成)
- [x] **定义工具接口和基础类型** (`internal/tools/types.go`)
- [x] **实现工具注册管理器** (`internal/tools/registry.go`)
- [x] **创建基础的工具执行引擎** (`internal/tools/executor.go`)
- [x] **扩展配置系统** (`internal/config/config.go`)

### 插件化架构 (完成) 🎯
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

### LLM集成和工具实现 (完成)
- [x] **扩展Generator接口支持工具调用** (`internal/domain/types.go`)
- [x] **更新OllamaService实现工具调用** (`internal/llm/ollama.go`)
- [x] **实现工具协调器** (`internal/tools/coordinator.go`)
- [x] **实现5个内置工具**:
  - `datetime` - 日期时间查询工具
  - `file_operations` - 文件系统操作工具
  - `rag_search` - RAG搜索工具
  - `document_info` - 文档信息查询工具
  - `sql_query` - SQL查询工具 (支持 modernc.org/sqlite)
- [x] **API集成** (`api/handlers/tools.go`, `internal/processor/service.go`)
- [x] **CLI集成** (`cmd/rago/tools.go`, `cmd/rago/query.go`)
- [x] **库接口支持** (`lib/rago.go`)

### 测试和修复 (完成)
- [x] **单元测试覆盖**
- [x] **修复SQLite驱动问题** (modernc.org/sqlite)
- [x] **修复工具响应生成问题**
- [x] **版本发布** (v1.0.0, v1.0.1)

## 🔥 Critical Priority (进行中)

### 1. 修复流式工具调用的多轮对话问题 (In Progress)
- **问题**: 当前的 `StreamToolCallingConversation` 只支持单轮对话，无法处理工具执行结果后的后续对话
- **位置**: `internal/tools/coordinator.go:272-326`
- **解决方案**: 重新设计流式处理，支持多轮对话循环，类似非流式版本的实现

## 🎯 High Priority

### 2. 实现完整的插件加载系统(.so文件)
- **目标**: 完成 `.so` 文件动态加载功能
- **当前状态**: 基础框架已完成，需要完善加载逻辑
- **位置**: `internal/tools/plugin.go`
- **任务**:
  - 完善插件文件扫描和加载逻辑
  - 实现插件热重载功能
  - 添加插件依赖管理
  - 完善错误处理和日志记录

### 3. 增加更多实用工具
- **HTTP 请求工具** (GET/POST/PUT/DELETE)
- **邮件发送工具** (SMTP支持)
- **文件压缩/解压工具** (zip/tar.gz)
- **系统信息查询工具** (CPU、内存、磁盘)
- **网络测试工具** (ping、端口检测)

### 4. 增强安全机制
- **工具权限控制和白名单机制**
- **审计日志记录所有工具调用**
- **敏感操作确认机制**
- **资源使用限制和监控**
- **输入参数验证和净化**

## 🚀 Medium Priority

### 5. 添加监控和可观测性
- **Prometheus 指标收集**
- **OpenTelemetry 链路追踪**
- **工具执行性能监控**
- **错误率和成功率统计**
- **资源使用情况监控**

### 6. 性能优化
- **结果缓存机制**
- **数据库连接池管理**
- **并发执行优化**
- **内存使用优化**
- **工具调用去重**

### 7. 实现对话记忆和上下文管理
- **长期对话历史存储**
- **上下文相关性分析**
- **智能上下文剪枝**
- **会话恢复功能**
- **多轮对话状态管理**

## 📋 Standard Priority

### 8. 增强Web UI支持工具调用界面
- **工具执行状态可视化**
- **交互式工具参数配置**
- **执行结果展示优化**
- **实时执行监控**
- **工具使用统计图表**

### 9. 添加端到端集成测试
- **完整工具调用流程测试**
- **多工具协作场景测试**
- **错误处理和恢复测试**
- **性能基准测试**
- **安全边界测试**

### 10. 完善API文档和使用指南
- **完整的工具开发指南**
- **API 参考文档**
- **最佳实践文档**
- **故障排除指南**
- **性能调优指南**

## 📊 当前状态总结
- ✅ **核心工具调用框架** (已完成)
- ✅ **5个内置工具** (datetime, file_operations, rag_search, document_info, sql_query)
- ✅ **插件架构基础框架** (已完成)
- ✅ **CLI 和 API 集成** (已完成)
- ✅ **库接口支持** (已完成)
- 🔄 **流式工具调用修复** (进行中)
- 📝 **插件系统完善** (待开始)

## 🎯 下一步行动计划
1. **立即**: 修复流式工具调用的多轮对话问题
2. **接下来**: 完善插件系统的.so文件动态加载
3. **然后**: 添加HTTP请求、邮件等实用工具
4. **最后**: 增强安全机制和监控系统

## 关键文件结构

```
internal/
├── tools/
│   ├── types.go              ✅ 工具接口和基础类型
│   ├── registry.go           ✅ 工具注册管理器
│   ├── executor.go           ✅ 工具执行引擎
│   ├── coordinator.go        ✅ 工具协调器 (🔄 流式修复中)
│   ├── plugin_manager.go     ✅ 插件管理器
│   ├── plugin_example.go     ✅ 插件开发示例
│   └── builtin/              ✅ 内置工具实现
│       ├── datetime.go       ✅ 日期时间工具
│       ├── sql_tool.go       ✅ SQL查询工具
│       ├── file_tools.go     ✅ 文件操作工具
│       └── rag_tools.go      ✅ RAG专用工具
├── config/
│   └── config.go             ✅ 扩展配置支持工具和插件
├── domain/
│   └── types.go              ✅ 扩展领域类型
├── llm/
│   └── ollama.go             ✅ 添加工具调用支持
└── processor/
    └── service.go            ✅ 集成工具调用到查询处理

api/handlers/
└── tools.go                  ✅ 工具相关API端点

cmd/rago/
├── tools.go                  ✅ 工具管理CLI命令
└── query.go                  ✅ 支持工具调用的查询命令

lib/
└── rago.go                   ✅ 库接口支持工具调用

plugins/                      📝 插件目录 (待完善)
└── example_plugin.so         📝 示例插件

PLUGIN_ARCHITECTURE.md        ✅ 插件开发完整指南
```

## 配置示例

当前系统支持的配置结构：

```toml
[tools]
enabled = true
max_concurrent_calls = 3
call_timeout = "30s"
security_level = "normal"
enabled_tools = ["datetime", "sql_query", "rag_search", "file_operations", "document_info"]

[tools.builtin]
datetime = { enabled = true }
sql_query = { 
    enabled = true, 
    max_connections = 5,
    allowed_drivers = ["postgres", "mysql", "sqlite"]
}
rag_search = { enabled = true }
file_operations = { 
    enabled = true,
    allowed_paths = ["/safe/path/*", "./documents/*", "."]
}
document_info = { enabled = true }

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

当前系统支持的API接口：

```bash
# 工具管理
GET /api/tools                    # 获取所有工具列表
GET /api/tools/datetime           # 获取特定工具详情
POST /api/tools/datetime/execute  # 执行特定工具

# 插件管理
GET /api/plugins                  # 获取所有插件列表
POST /api/plugins/reload          # 重载插件

# 带工具调用的查询
POST /api/query/with-tools        # 执行带工具的查询
POST /api/query/with-tools/stream # 流式执行带工具的查询
```

## CLI示例

当前系统支持的CLI命令：

```bash
# 工具管理
rago tools list                   # 列出所有可用工具
rago tools info datetime          # 查看工具详情
rago tools execute datetime       # 执行工具

# 带工具调用的查询
rago query "现在几点？" --tools
rago query "查询文档数量" --tools --allowed-tools="document_info,rag_search"
rago query "检查当前目录" --tools --max-tool-calls=5
```

## 测试计划

- [x] **单元测试所有工具实现**
- [x] **集成测试工具调用流程** 
- [x] **修复测试失败问题**
- [ ] **性能测试并发执行**
- [ ] **安全测试权限控制**
- [ ] **端到端集成测试**

---
**更新时间**: 2025-01-23  
**完成度**: 85% (17/20 个主要功能模块)

## 🎯 插件化架构亮点

通过插件化设计，RAGO的工具调用系统实现了：

- **动态扩展**: 无需重编译即可添加新工具
- **社区生态**: 支持第三方开发者贡献工具插件  
- **安全隔离**: 插件沙箱和权限控制
- **热更新**: 支持插件的动态加载/卸载/重载
- **配置灵活**: 细粒度的插件配置管理

这使得RAGO不仅是一个RAG系统，更是一个可扩展的AI工具平台。

## 📈 版本历史

- **v1.0.1** (Current) - 修复工具调用响应生成问题，增强稳定性
- **v1.0.0** - 首个完整的工具调用系统版本
- **v0.x.x** - 基础RAG功能版本