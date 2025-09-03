# RAGO Agents Module - Implementation Plan

## 🤖 **Overview**

Transform RAGO into an autonomous AI platform with workflow automation, tool chaining, and agent-like behaviors. This module can be used both as part of RAGO or as a standalone Go library.

## **Module Structure**

```
pkg/agents/
├── core/
│   ├── agent.go           # Agent interface & base types
│   ├── executor.go        # Agent execution engine
│   ├── workflow.go        # Workflow definition & execution
│   └── context.go         # Agent execution context
├── tools/
│   ├── chain.go           # Tool chaining logic
│   ├── conditional.go     # Conditional execution
│   └── parallel.go        # Parallel tool execution
├── storage/
│   ├── agent_store.go     # Agent definition storage
│   └── execution_log.go   # Execution history
├── api/
│   ├── handlers.go        # HTTP API endpoints
│   └── websocket.go       # Real-time execution updates
└── types/
    ├── agent_types.go     # Core data structures
    └── workflow_types.go  # Workflow definitions
```

## **Core Agent Types**

```go
type Agent struct {
    ID           string            `json:"id"`
    Name         string            `json:"name"`
    Description  string            `json:"description"`
    Type         AgentType         `json:"type"`
    Config       AgentConfig       `json:"config"`
    Workflow     WorkflowSpec      `json:"workflow"`
    Status       AgentStatus       `json:"status"`
    CreatedAt    time.Time         `json:"created_at"`
    UpdatedAt    time.Time         `json:"updated_at"`
}

type WorkflowSpec struct {
    Steps        []WorkflowStep    `json:"steps"`
    Triggers     []Trigger         `json:"triggers"`
    Variables    map[string]any    `json:"variables"`
    ErrorPolicy  ErrorPolicy       `json:"error_policy"`
}
```

## **Agent Categories**

### Research Agents
- Document analysis and summarization
- Multi-source information gathering
- Fact-checking and verification
- Knowledge synthesis

### Workflow Agents
- Task automation and scheduling
- Multi-step process execution
- Data pipeline management
- Integration orchestration

### Monitoring Agents
- System health monitoring
- Alert generation and escalation
- Performance tracking
- Anomaly detection

## **Autonomous Behaviors**

```go
type AutonomyLevel int

const (
    Manual     AutonomyLevel = iota // User-triggered only
    Scheduled                       // Time-based execution
    Reactive                        // Event-driven execution
    Proactive                       // Goal-seeking behavior
    Adaptive                        // Learning and optimization
)
```

## **Workflow Definition Language**

```yaml
# Example: Document Analysis Workflow
name: "document_analyzer"
description: "Analyze uploaded documents and extract insights"
triggers:
  - type: "document_upload"
    conditions:
      - file_type: ["pdf", "docx", "txt"]
variables:
  analysis_depth: "detailed"
  output_format: "json"
steps:
  - name: "extract_text"
    tool: "mcp_pdf_extract"
    inputs:
      file_path: "{{trigger.file_path}}"
    outputs:
      text_content: "text"
  
  - name: "analyze_content"
    tool: "llm_analyze"
    inputs:
      content: "{{steps.extract_text.text_content}}"
      prompt: "Analyze this document and extract key insights"
    outputs:
      insights: "analysis"
  
  - name: "store_results"
    tool: "mcp_sqlite_insert"
    inputs:
      table: "document_analysis"
      data: "{{steps.analyze_content.insights}}"
```

## **Tool Chaining Engine**

```go
type ToolChain struct {
    Steps       []ChainStep       `json:"steps"`
    Conditions  []Condition       `json:"conditions"`
    Loops       []LoopDefinition  `json:"loops"`
    Parallels   []ParallelGroup   `json:"parallels"`
}

type ChainStep struct {
    ID          string            `json:"id"`
    ToolName    string            `json:"tool_name"`
    Inputs      map[string]string `json:"inputs"`     // Template expressions
    Outputs     map[string]string `json:"outputs"`    // Variable mappings
    Conditions  []string          `json:"conditions"` // Execution conditions
    Retry       RetryPolicy       `json:"retry"`
    Timeout     time.Duration     `json:"timeout"`
}
```

## **Web Interface Extensions**

### New UI Components
- **Agent Builder**: Visual workflow designer
- **Execution Dashboard**: Real-time agent monitoring  
- **Template Library**: Pre-built agent templates
- **Analytics Panel**: Usage metrics and performance

### Agent Management Tab
```typescript
// New AgentsTab component features:
- Agent creation wizard
- Workflow visual editor  
- Execution history viewer
- Performance analytics
- Template marketplace
```

## **API Extensions**

### New HTTP Endpoints
```
# Agent Management
POST   /api/agents                    # Create agent
GET    /api/agents                    # List agents
GET    /api/agents/:id                # Get agent details
PUT    /api/agents/:id                # Update agent
DELETE /api/agents/:id                # Delete agent

# Execution Control
POST   /api/agents/:id/execute        # Trigger agent execution
GET    /api/agents/:id/executions     # Get execution history
GET    /api/executions/:exec_id       # Get execution details
POST   /api/executions/:exec_id/stop  # Stop running execution

# Workflow Management
GET    /api/workflows/templates       # Get workflow templates
POST   /api/workflows/validate        # Validate workflow definition
```

### WebSocket Integration
```go
// Real-time execution updates
type ExecutionUpdate struct {
    ExecutionID  string              `json:"execution_id"`
    AgentID      string              `json:"agent_id"`
    Status       ExecutionStatus     `json:"status"`
    CurrentStep  string              `json:"current_step"`
    Progress     float64             `json:"progress"`
    Logs         []ExecutionLog      `json:"logs"`
    Results      map[string]any      `json:"results"`
}
```

## **Implementation Priority**

### Phase 1: Core Foundation (Week 1-2)
1. Basic agent types and storage
2. Simple workflow execution engine
3. MCP tool integration layer

### Phase 2: Workflow Engine (Week 3-4)
1. Advanced workflow features (conditions, loops, parallel)
2. Template system
3. Error handling and retry logic

### Phase 3: Advanced Features (Week 5-6)
1. Autonomous behaviors
2. Learning and adaptation
3. Performance optimization

### Phase 4: UI/UX Polish (Week 7-8)
1. Visual workflow builder
2. Real-time monitoring dashboard
3. Analytics and reporting

## **📚 Library Usage**

### Standalone Library Usage

The agents module can be used as a standalone Go library:

```go
package main

import (
    "github.com/liliang-cn/rago/v2/pkg/agents/core"
    "github.com/liliang-cn/rago/v2/pkg/agents/storage"
    "github.com/liliang-cn/rago/v2/pkg/mcp"
)

func main() {
    // Initialize MCP client
    mcpConfig := &mcp.Config{
        Enabled: true,
        Servers: map[string]mcp.ServerConfig{
            "sqlite": {
                Command: "mcp-sqlite-server",
                Args:    []string{"./data"},
            },
        },
    }
    
    mcpClient := mcp.NewMCPService(mcpConfig)
    
    // Initialize agent storage
    store := storage.NewMemoryAgentStore()
    
    // Create agent executor
    executor := core.NewAgentExecutor(mcpClient, store)
    
    // Define and run agent
    agent := &core.Agent{
        Name: "document_processor",
        Workflow: core.WorkflowSpec{
            Steps: []core.WorkflowStep{
                {
                    Name: "extract",
                    Tool: "mcp_sqlite_query",
                    Inputs: map[string]string{
                        "query": "SELECT * FROM documents",
                    },
                },
            },
        },
    }
    
    // Execute agent
    result, err := executor.Execute(context.Background(), agent)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Agent execution result: %v\n", result)
}
```

### Integration with Existing Applications

```go
// Add to existing RAG application
import "github.com/liliang-cn/rago/v2/pkg/agents"

// Initialize agents module
agentManager := agents.NewManager(mcpService, database)

// Register agents with HTTP server
router.PathPrefix("/api/agents").Handler(agentManager.HTTPHandler())

// Use in business logic
agent := agentManager.GetAgent("document-analyzer")
result := agent.ExecuteWorkflow(ctx, input)
```

### Configuration Options

```go
type AgentConfig struct {
    MaxConcurrentExecutions int           `yaml:"max_concurrent_executions"`
    DefaultTimeout          time.Duration `yaml:"default_timeout"`
    EnableMetrics           bool          `yaml:"enable_metrics"`
    StorageBackend          string        `yaml:"storage_backend"` // memory, sqlite, postgres
    MCPIntegration          bool          `yaml:"mcp_integration"`
}
```

### Key Benefits as Library
- **Modular**: Use only the components you need
- **Extensible**: Plugin architecture for custom tools
- **Portable**: Works with any Go application
- **Lightweight**: Minimal dependencies outside MCP integration
- **Observable**: Built-in metrics and logging support