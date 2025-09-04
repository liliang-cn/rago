# RAGO Library Usage Examples

This directory contains examples showing how to use RAGO as a Go library in your own applications.

## Prerequisites

Make sure you have:
1. Go 1.21 or later installed
2. RAGO configured with a valid `rago.toml` file
3. Required AI services running (LMStudio, Ollama, or OpenAI)

## Running Examples

Each example is in its own directory and can be run independently:

```bash
# Navigate to any example directory
cd basic_workflow

# Run the example
go run main.go
```

## Library Usage Examples

### 1. [Basic Workflow](./basic_workflow/)
**File**: `basic_workflow/main.go`

Basic workflow execution example showing:
- Loading configuration from rago.toml
- Initializing providers (LLM, embedder)
- Defining workflows programmatically
- Executing workflows and processing results

```bash
go run examples/basic_workflow/main.go
```

### 2. [Natural Language to Workflow](./nl_to_workflow/)
**File**: `nl_to_workflow/main.go`

Natural language to workflow conversion:
- Converting natural language requests to workflows
- Using LLM to generate workflow JSON
- Executing dynamically generated workflows
- Processing complex user requests

```bash
go run examples/nl_to_workflow/main.go
```

### 3. [Async Workflow](./async_workflow/)
**File**: `async_workflow/main.go`

Async workflow execution with parallel steps:
- Parallel execution of independent steps
- Dependency resolution for sequential steps
- Performance optimization through parallelism
- Complex workflow orchestration

```bash
go run examples/async_workflow/main.go
```

### 4. [Agents Integration](./agents_integration/)
**File**: `agents_integration/config_based_demo.go`

Configuration-based provider usage:
- Using different LLM providers (Ollama, LMStudio, OpenAI)
- Provider configuration from rago.toml
- Seamless provider switching

```bash
go run examples/agents_integration/config_based_demo.go
```

### 5. [Other Examples](./)

Additional examples in this directory:
- **basic_usage**: Simple client usage patterns
- **document_ingestion**: Document ingestion and RAG
- **mcp_integration**: MCP protocol integration
- **streaming_chat**: Streaming responses
- **task_scheduling**: Task management

## Quick Start

1. **Ensure you have a valid `rago.toml` configuration:**
```toml
[providers]
default_llm = "lmstudio"  # or "ollama", "openai"

[providers.lmstudio]
base_url = "http://localhost:1234/v1"
llm_model = "qwen/qwen3-4b-2507"
```

2. **Install dependencies:**
```bash
go mod tidy
```

3. **Run any example:**
```bash
# Basic workflow
go run examples/basic_workflow/main.go

# Natural language to workflow
go run examples/nl_to_workflow/main.go

# Async workflow
go run examples/async_workflow/main.go
```

## Using RAGO in Your Project

### Import the packages:
```go
import (
    "github.com/liliang-cn/rago/v2/pkg/agents/execution"
    "github.com/liliang-cn/rago/v2/pkg/agents/types"
    "github.com/liliang-cn/rago/v2/pkg/config"
    "github.com/liliang-cn/rago/v2/pkg/utils"
)
```

### Basic workflow execution:
```go
// Load config
cfg, err := config.Load("")

// Initialize providers
ctx := context.Background()
_, llmService, _, err := utils.InitializeProviders(ctx, cfg)

// Create workflow
workflow := &types.WorkflowSpec{
    Steps: []types.WorkflowStep{
        // Define your steps
    },
}

// Execute
executor := execution.NewWorkflowExecutor(cfg, llmService)
result, err := executor.Execute(ctx, workflow)
```

## Available Tools

- **filesystem**: File operations (read, write, list, execute, move, copy, delete, mkdir)
- **fetch**: HTTP/HTTPS requests for APIs and websites
- **memory**: Temporary storage (store, retrieve, delete, append)
- **time**: Date/time operations (now, format, parse)
- **sequential-thinking**: LLM analysis and reasoning
- **playwright**: Browser automation (if configured in mcpServers.json)

## Data Flow

Use `{{variableName}}` syntax to pass data between workflow steps:

```go
Steps: []types.WorkflowStep{
    {
        ID: "fetch_data",
        Outputs: map[string]string{"data": "fetched_data"},
    },
    {
        ID: "analyze",
        Inputs: map[string]interface{}{
            "data": "{{fetched_data}}", // Reference previous output
        },
    },
}
```

## Learn More

- [Main Documentation](../../README.md)
- [Client Library Documentation](../../client/README.md)
- [Configuration Guide](../../rago.example.toml)