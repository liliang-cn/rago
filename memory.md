现在我对整个架构有了清晰的理解。让我设计详细的实现方案。

---

Agent Memory 模块详细设计方案

一、架构概览

┌─────────────────────────────────────────────────────────────────────────┐
│ rago │
├─────────────────────────────────────────────────────────────────────────┤
│ │
│ ┌───────────────────────────────────────────────────────────────────┐ │
│ │ pkg/memory/ (NEW) │ │
│ │ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐ │ │
│ │ │ Store │ │ Retriever │ │ Summarizer │ │ │
│ │ │ │ │ │ │ │ │ │
│ │ │ - Add() │ │ - Retrieve()│ │ - SummarizeTask() │ │ │
│ │ │ - Search() │ │ - GetScore()│ │ - ShouldStore() │ │ │
│ │ │ - Update() │ │ │ │ - ExtractInsights() │ │ │
│ │ └──────┬──────┘ └──────┬──────┘ └──────────┬──────────────┘ │ │
│ │ │ │ │ │ │
│ └─────────┼──────────────────┼─────────────────────┼──────────────────┘ │
│ │ │ │ │
│ ┌─────────▼──────────────────▼─────────────────────▼──────────────────┐ │
│ │ MemoryService (Facade) │ │
│ │ - RetrieveAndInject() // 检索并注入到 LLM context │ │
│ │ - StoreIfWorthwhile() // 任务完成后判断是否存储 │ │
│ └────────────────────────────────────────────────────────────────────┘ │
│ │ │
│ │ ┌─────────────────────────────────────────────────┐ │
│ │ │ Integration Points │ │
│ │ │ ┌──────────────┐ ┌──────────────────────┐ │ │
│ └────┼─▶ Agent │ │ LLM Service │ │ │
│ │ │ Executor │ │ (prompt composition) │ │ │
│ │ └──────┬───────┘ └──────────────────────┘ │ │
│ │ │ │ │
│ │ 1. ExecutePlan() 完成后调用 StoreIfWorthwhile() │ │
│ │ 2. Query 时调用 RetrieveAndInject() 补充 context │ │
│ └────────────────────────────────────────────────────┘ │
│ │
├─────────────────────────────────────────────────────────────────────────┤
│ sqvect (依赖) │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ agent_memory 表 (NEW) │ │
│ │ - id TEXT PRIMARY KEY │ │
│ │ - session_id TEXT (关联 conversation) │ │
│ │ - memory_type TEXT ('fact', 'skill', 'pattern', 'context') │ │
│ │ - content TEXT NOT NULL │ │
│ │ - vector BLOB (语义检索) │ │
│ │ - importance FLOAT (0-1, 用于排序/优先级) │ │
│ │ - access_count INTEGER (访问频率) │ │
│ │ - last_accessed DATETIME │ │
│ │ - metadata TEXT (JSON) │ │
│ │ - created_at DATETIME │ │
│ │ - updated_at DATETIME │ │
│ └─────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘

二、数据模型设计

2.1 sqvect 新增表结构

// sqvect/pkg/core/memory.go (NEW)

CREATE TABLE IF NOT EXISTS agent_memory (
id TEXT PRIMARY KEY,
session_id TEXT, -- 关联的 conversation，NULL 表示全局记忆
memory_type TEXT NOT NULL, -- 'fact', 'skill', 'pattern', 'context', 'preference'
content TEXT NOT NULL, -- 记忆内容
vector BLOB, -- 向量嵌入，用于语义检索
importance REAL DEFAULT 0.5, -- 重要性 0-1
access_count INTEGER DEFAULT 0, -- 访问次数
last_accessed DATETIME, -- 最后访问时间
metadata TEXT, -- JSON: {"tags": [], "source_task": "...", "entities": []}
created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agent_memory_session ON agent_memory(session_id);
CREATE INDEX IF NOT EXISTS idx_agent_memory_type ON agent_memory(memory_type);
CREATE INDEX IF NOT EXISTS idx_agent_memory_importance ON agent_memory(importance DESC);

-- FTS5 for keyword search
CREATE VIRTUAL TABLE IF NOT EXISTS agent_memory_fts USING fts5(
content,
content='agent_memory',
content_rowid='rowid'
);

2.2 Memory Types 定义
┌────────────┬────────────┬────────────────────────────────────┐
│ Type │ 说明 │ 示例 │
├────────────┼────────────┼────────────────────────────────────┤
│ fact │ 事实性知识 │ "用户偏好使用 Go 语言开发" │
├────────────┼────────────┼────────────────────────────────────┤
│ skill │ 技能/流程 │ "如何使用 Git rebase 解决冲突" │
├────────────┼────────────┼────────────────────────────────────┤
│ pattern │ 模式/规律 │ "用户常在下午请求代码审查" │
├────────────┼────────────┼────────────────────────────────────┤
│ context │ 上下文信息 │ "项目 X 正在重构认证模块" │
├────────────┼────────────┼────────────────────────────────────┤
│ preference │ 用户偏好 │ "代码风格：不使用 emoji，tab 缩进" │
└────────────┴────────────┴────────────────────────────────────┘
2.3 Domain 类型定义

// rago/pkg/domain/memory.go (NEW)

package domain

import "time"

type MemoryType string

const (
MemoryTypeFact MemoryType = "fact"
MemoryTypeSkill MemoryType = "skill"
MemoryTypePattern MemoryType = "pattern"
MemoryTypeContext MemoryType = "context"
MemoryTypePreference MemoryType = "preference"
)

// Memory 表示一条长期记忆
type Memory struct {
ID string `json:"id"`
SessionID string `json:"session_id,omitempty"`
Type MemoryType `json:"type"`
Content string `json:"content"`
Vector []float64 `json:"vector,omitempty"`
Importance float64 `json:"importance"`
AccessCount int `json:"access_count"`
LastAccessed time.Time `json:"last_accessed"`
Metadata map[string]interface{} `json:"metadata,omitempty"`
CreatedAt time.Time `json:"created_at"`
UpdatedAt time.Time `json:"updated_at"`
}

// MemoryRetrieveResult 检索结果
type MemoryRetrieveResult struct {
Memories []\*MemoryWithScore `json:"memories"`
Query string `json:"query"`
Threshold float64 `json:"threshold"` // 最低相关分数
HasRelevant bool `json:"has_relevant"` // 是否有相关记忆
}

type MemoryWithScore struct {
\*Memory
Score float64 `json:"score"`
}

// MemoryStoreRequest 写入请求
type MemoryStoreRequest struct {
SessionID string `json:"session_id"`
TaskGoal string `json:"task_goal"`
TaskResult string `json:"task_result"`
ExecutionLog string `json:"execution_log,omitempty"`
Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// MemorySummaryResult LLM 总结结果
type MemorySummaryResult struct {
ShouldStore bool `json:"should_store"`
Memories []MemoryItem `json:"memories"`
Reasoning string `json:"reasoning"`
}

type MemoryItem struct {
Type MemoryType `json:"type"`
Content string `json:"content"`
Importance float64 `json:"importance"`
Tags []string `json:"tags,omitempty"`
Entities []string `json:"entities,omitempty"`
}

三、核心接口设计

3.1 MemoryStore Interface (sqvect)

// sqvect/pkg/core/memory.go

// MemoryStore 管理长期记忆存储
type MemoryStore interface {
// Store 存储一条新记忆
Store(ctx context.Context, memory \*Memory) error

      // Search 向量搜索相关记忆
      Search(ctx context.Context, vector []float64, topK int, minScore float64) ([]*MemoryWithScore, error)

      // SearchBySession 搜索特定 session 的记忆
      SearchBySession(ctx context.Context, sessionID string, vector []float64, topK int) ([]*MemoryWithScore, error)

      // Update 更新记忆内容和访问统计
      Update(ctx context.Context, id string, memory *Memory) error

      // IncrementAccess 增加访问计数
      IncrementAccess(ctx context.Context, id string) error

      // GetByType 按类型获取记忆
      GetByType(ctx context.Context, memoryType MemoryType, limit int) ([]*Memory, error)

      // Delete 删除记忆
      Delete(ctx context.Context, id string) error

      // InitMemorySchema 初始化表结构
      InitMemorySchema(ctx context.Context) error

}

3.2 MemoryService Interface (rago)

// rago/pkg/memory/service.go

import (
"context"
"github.com/liliang-cn/rago/v2/pkg/domain"
)

type MemoryService interface {
// RetrieveAndInject 检索相关记忆并格式化为 LLM context
RetrieveAndInject(ctx context.Context, query string, sessionID string) (string, []\*domain.MemoryWithScore, error)

      // StoreIfWorthwhile 任务完成后判断是否值得存储，由 LLM 决定
      StoreIfWorthwhile(ctx context.Context, req *domain.MemoryStoreRequest) error

      // Add 直接添加记忆
      Add(ctx context.Context, memory *domain.Memory) error

      // Update 由 LLM 更新现有记忆
      Update(ctx context.Context, id string, content string) error

      // Search 搜索记忆
      Search(ctx context.Context, query string, topK int) ([]*domain.MemoryWithScore, error)

}

type memoryService struct {
store core.MemoryStore // sqvect
llm domain.Generator // LLM
embedder domain.Embedder // Embedding
minScore float64 // 相关性阈值，默认 0.7
maxMemories int // 最多注入记忆数，默认 5
}

四、核心实现逻辑

4.1 检索流程

// RetrieveAndInject 实现
func (s *memoryService) RetrieveAndInject(ctx context.Context, query string, sessionID string) (string, []*domain.MemoryWithScore, error) {
// 1. 生成 query 向量
vector, err := s.embedder.Embed(ctx, query)
if err != nil {
return "", nil, err
}

      // 2. 检索：session 记忆 + 全局记忆
      sessionMems, _ := s.store.SearchBySession(ctx, sessionID, vector, s.maxMemories/2)
      globalMems, _ := s.store.Search(ctx, vector, s.maxMemories/2, s.minScore)

      // 3. 合并去重，按分数排序
      memories := s.mergeAndRank(sessionMems, globalMems)

      // 4. 更新访问统计
      for _, m := range memories {
          _ = s.store.IncrementAccess(ctx, m.ID)
      }

      // 5. 格式化为 LLM context
      if len(memories) == 0 {
          return "", memories, nil
      }

      contextStr := s.formatMemories(memories)
      return contextStr, memories, nil

}

func (s *memoryService) formatMemories(memories []*domain.MemoryWithScore) string {
var sb strings.Builder
sb.WriteString("## Relevant Memory\n\n")
for i, m := range memories {
sb.WriteString(fmt.Sprintf("[%d] [%s] (score: %.2f)\n%s\n\n",
i+1, m.Type, m.Score, m.Content))
}
return sb.String()
}

4.2 存储流程 (LLM 驱动)

// StoreIfWorthwhile 实现
func (s *memoryService) StoreIfWorthwhile(ctx context.Context, req *domain.MemoryStoreRequest) error {
// 1. 构造 LLM prompt，让 LLM 判断并提取记忆
prompt := s.buildSummaryPrompt(req)

      // 2. 使用 structured generation 获取结果
      schema := map[string]interface{}{
          "type": "object",
          "properties": map[string]interface{}{
              "should_store":   {"type": "boolean"},
              "reasoning":      {"type": "string"},
              "memories": {
                  "type": "array",
                  "items": map[string]interface{}{
                      "type": "object",
                      "properties": map[string]interface{}{
                          "type":       {"type": "string", "enum": []string{"fact", "skill", "pattern", "context", "preference"}},
                          "content":    {"type": "string"},
                          "importance": {"type": "number"},
                          "tags":       {"type": "array", "items": map[string]string{"type": "string"}},
                          "entities":   {"type": "array", "items": map[string]string{"type": "string"}},
                      },
                      "required": []string{"type", "content", "importance"},
                  },
              },
          },
          "required": []string{"should_store", "memories"},
      }

      result, err := s.llm.GenerateStructured(ctx, prompt, schema, &domain.GenerationOptions{
          Temperature: 0.3, // 低温度保证稳定输出
          MaxTokens:   1000,
      })
      if err != nil {
          return err
      }

      // 3. 解析结果
      var summary domain.MemorySummaryResult
      if err := json.Unmarshal([]byte(result.Raw), &summary); err != nil {
          return err
      }

      if !summary.ShouldStore || len(summary.Memories) == 0 {
          return nil // 不值得存储
      }

      // 4. 存储每条记忆
      for _, item := range summary.Memories {
          vector, _ := s.embedder.Embed(ctx, item.Content)

          memory := &domain.Memory{
              ID:         uuid.New().String(),
              SessionID:  req.SessionID,
              Type:       item.Type,
              Content:    item.Content,
              Vector:     vector,
              Importance: item.Importance,
              Metadata: map[string]interface{}{
                  "tags":     item.Tags,
                  "entities": item.Entities,
                  "source":   "task_completion",
              },
          }

          if err := s.store.Store(ctx, memory); err != nil {
              // log but continue
              continue
          }
      }

      return nil

}

func (s *memoryService) buildSummaryPrompt(req *domain.MemoryStoreRequest) string {
return fmt.Sprintf(`Analyze the completed task and extract any information worth storing in long-term memory.

Task Goal: %s

Task Result: %s

%s

Guidelines:

- Extract facts, skills, patterns, or preferences that could be useful for future tasks
- Only store information that is likely to be referenced again
- Importance score (0-1): >0.8 for critical info, >0.5 for useful info, <0.5 for trivial
- Tags: short keywords for categorization
- Entities: named entities (people, projects, concepts)

Return JSON with: should_store (boolean), reasoning, and memories array.
`, req.TaskGoal, req.TaskResult, s.formatExecutionLog(req.ExecutionLog))
}

4.3 LLM 更新记忆

// Update 实现 - 由 LLM 决定如何更新
func (s \*memoryService) Update(ctx context.Context, id string, updateInstruction string) error {
// 1. 获取现有记忆
memory, err := s.store.Get(ctx, id)
if err != nil {
return err
}

      // 2. 让 LLM 根据 instruction 更新
      prompt := fmt.Sprintf(`Update the following memory based on the instruction.

Current Memory:
Type: %s
Content: %s
Importance: %.2f

Update Instruction: %s

Return updated JSON with: content, importance (if changed).
`, memory.Type, memory.Content, memory.Importance, updateInstruction)

      schema := map[string]interface{}{
          "type": "object",
          "properties": map[string]interface{}{
              "content":    {"type": "string"},
              "importance": {"type": "number"},
          },
          "required": []string{"content"},
      }

      result, err := s.llm.GenerateStructured(ctx, prompt, schema, &domain.GenerationOptions{
          Temperature: 0.2,
          MaxTokens:   500,
      })
      if err != nil {
          return err
      }

      var update struct {
          Content    string  `json:"content"`
          Importance float64 `json:"importance"`
      }
      json.Unmarshal([]byte(result.Raw), &update)

      // 3. 更新存储
      memory.Content = update.Content
      if update.Importance > 0 {
          memory.Importance = update.Importance
      }
      memory.UpdatedAt = time.Now()

      return s.store.Update(ctx, id, memory)

}

五、集成点

5.1 Agent Executor 集成

// rago/pkg/agent/executor.go 修改

type Executor struct {
llmService domain.Generator
toolExecutor ToolExecutor
mcpService MCPToolExecutor
ragProcessor domain.Processor
memoryService domain.MemoryService // NEW
}

// ExecutePlan 结束时调用
func (e *Executor) ExecutePlan(ctx context.Context, plan *Plan) (\*ExecutionResult, error) {
// ... 原有执行逻辑 ...

      // NEW: 任务完成后，尝试提取并存储记忆
      if e.memoryService != nil && plan.Status == PlanStatusCompleted {
          go func() {  // 异步执行，不阻塞结果返回
              _ = e.memoryService.StoreIfWorthwhile(context.Background(), &domain.MemoryStoreRequest{
                  SessionID:    plan.SessionID,
                  TaskGoal:     plan.Goal,
                  TaskResult:   formatResultForContent(finalResult),
                  ExecutionLog: e.buildExecutionLog(plan),
              })
          }()
      }

      return &ExecutionResult{...}, nil

}

5.2 Query 集成

// rago/pkg/processor/service.go 修改

func (s *ragoService) Query(ctx context.Context, req domain.QueryRequest) (domain.QueryResponse, error) {
var memoryContext string
var memoryMems []*domain.MemoryWithScore

      // NEW: 检索相关记忆
      if s.memoryService != nil {
          memoryContext, memoryMems, _ = s.memoryService.RetrieveAndInject(
              ctx, req.Query, req.ConversationID,
          )
      }

      // 原有 RAG 检索...
      chunks := s.vectorStore.Search(...)

      // 组装 prompt，注入记忆 context
      prompt := s.ComposePrompt(chunks, memoryContext, req.Query)

      // LLM 生成...
      answer := s.llmService.Generate(ctx, prompt, ...)

      return domain.QueryResponse{
          Answer:  answer,
          Sources: chunks,
          Memories: memoryMems,  // NEW: 返回使用的记忆
      }, nil

}

5.3 MCP Tool 暴露

// rago/pkg/mcp/memory_tools.go (NEW)

// RegisterMemoryTools 注册 memory 相关 MCP 工具
func RegisterMemoryTools(svc \*MCPService, memorySvc domain.MemoryService) {
svc.RegisterTool(domain.ToolDefinition{
Type: "function",
Function: domain.ToolFunction{
Name: "memory*search",
Description: "Search long-term memory for relevant information",
Parameters: map[string]interface{}{
"type": "object",
"properties": map[string]interface{}{
"query": map[string]interface{}{
"type": "string",
"description": "Search query",
},
"limit": map[string]interface{}{
"type": "integer",
"description": "Max results",
"default": 5,
},
},
"required": []string{"query"},
},
},
}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
query, * := args["query"].(string)
limit, \_ := args["limit"].(int)
if limit == 0 {
limit = 5
}
return memorySvc.Search(ctx, query, limit)
})

      svc.RegisterTool(domain.ToolDefinition{
          Type: "function",
          Function: domain.ToolFunction{
              Name:        "memory_update",
              Description: "Update existing memory with new information",
              Parameters: map[string]interface{}{
                  "type": "object",
                  "properties": map[string]interface{}{
                      "memory_id": map[string]interface{}{
                          "type": "string",
                      },
                      "instruction": map[string]interface{}{
                          "type": "string",
                          "description": "How to update the memory",
                      },
                  },
                  "required": []string{"memory_id", "instruction"},
              },
          },
      }, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
          id, _ := args["memory_id"].(string)
          instruction, _ := args["instruction"].(string)
          return nil, memorySvc.Update(ctx, id, instruction)
      })

}

六、配置参数

// rago/pkg/config/memory.go (NEW)

type MemoryConfig struct {
Enabled bool `toml:"enabled" json:"enabled"`
MinScore float64 `toml:"min_score" json:"min_score"` // 默认 0.7
MaxMemories int `toml:"max_memories" json:"max_memories"` // 默认 5
AutoStore bool `toml:"auto_store" json:"auto_store"` // 任务完成后自动存储
StoreThreshold float64 `toml:"store_threshold" json:"store_threshold"` // 存储阈值，默认 0.5
}

// 默认配置
func DefaultMemoryConfig() \*MemoryConfig {
return &MemoryConfig{
Enabled: true,
MinScore: 0.7,
MaxMemories: 5,
AutoStore: true,
StoreThreshold: 0.5,
}
}

七、文件结构

sqvect/
├── pkg/core/
│ ├── memory.go # NEW: MemoryStore interface + impl
│ └── store_init.go # MODIFY: Add agent_memory table

rago/
├── pkg/domain/
│ └── memory.go # NEW: Memory domain types
├── pkg/memory/
│ ├── service.go # MemoryService 实现
│ ├── retriever.go # 检索逻辑
│ ├── summarizer.go # LLM 总结逻辑
│ └── formatter.go # Context 格式化
├── pkg/agent/
│ └── executor.go # MODIFY: 集成 memoryService
├── pkg/processor/
│ └── service.go # MODIFY: Query 时注入 memory
├── pkg/mcp/
│ └── memory_tools.go # NEW: Memory MCP tools
└── cmd/rago/
└── memory.go # NEW: CLI 命令

八、CLI 命令 (可选)

# 搜索记忆

./rago memory search "user preferences" --limit 5

# 查看记忆详情

./rago memory get <memory-id>

# 手动添加记忆

./rago memory add --type fact --content "User prefers Go" --importance 0.8

# 更新记忆

./rago memory update <memory-id> "User now prefers Rust for systems programming"

# 列出记忆

./rago memory list --type skill --limit 10

# 删除记忆

./rago memory delete <memory-id>

---

实施步骤建议

1. Phase 1: sqvect 层 - 添加 agent_memory 表和 MemoryStore 接口
2. Phase 2: rago domain - 添加 pkg/domain/memory.go 类型定义
3. Phase 3: Memory Service - 实现 pkg/memory/ 核心逻辑
4. Phase 4: Agent 集成 - 修改 executor.go 添加任务完成处理
5. Phase 5: Query 集成 - 修改 processor.go 注入 memory context
6. Phase 6: MCP Tools - 添加 memory 相关工具供 LLM 调用
7. Phase 7: CLI 命令 - 添加调试和管理命令
