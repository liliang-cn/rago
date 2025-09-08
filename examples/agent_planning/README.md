# Agent Planning System Usage Guide

The agent planning system allows you to create detailed execution plans from natural language goals and execute them with progress tracking.

## CLI Usage

### 1. Create a Plan

Create a detailed execution plan for any goal:

```bash
# Basic plan creation
./rago agent plan "Create a REST API for user management"

# With verbose output to see planning details
./rago agent plan "Analyze the codebase and identify performance bottlenecks" -v

# With custom storage location
./rago agent plan "Generate API documentation" --storage /tmp/my-plans
```

The system will:
- Use LLM to break down your goal into tasks and steps
- Save the plan to `~/.rago/agents/plans/[plan-id]/`
- Display the plan ID for execution

### 2. Execute a Plan

Execute a previously created plan:

```bash
# Execute a plan
./rago agent run-plan abc123def4

# Resume a failed/paused plan
./rago agent run-plan abc123def4 --resume

# With verbose output to see execution details
./rago agent run-plan abc123def4 -v
```

### 3. Check Plan Status

Monitor the progress of your plans:

```bash
# Show detailed status of a specific plan
./rago agent plan-status abc123def4

# List all plans
./rago agent list-plans
```

### 4. Auto-Run (Plan + Execute)

Create and immediately execute a plan:

```bash
# One-step planning and execution
./rago agent auto-run "Create a README file with project documentation"

# With verbose output
./rago agent auto-run "Optimize database queries" -v
```

## How It Works

### Plan Structure

Each plan contains:
- **Goal**: The high-level objective
- **Tasks**: Major work items to achieve the goal
- **Steps**: Specific actions within each task
- **Dependencies**: Task ordering requirements
- **Tools**: MCP tools needed for execution

Example plan structure:
```json
{
  "id": "uuid-here",
  "goal": "Create a REST API",
  "tasks": [
    {
      "id": "task_1",
      "name": "Setup project structure",
      "steps": [
        {
          "id": "step_1_1",
          "action": "Create directory structure",
          "tool": "filesystem"
        }
      ]
    }
  ]
}
```

### Filesystem Storage

Plans are saved to:
```
~/.rago/agents/plans/
├── plan-id-1/
│   ├── plan.json       # Complete plan details
│   └── tracking.json   # Quick status info
├── plan-id-2/
│   ├── plan.json
│   └── tracking.json
```

### Task Execution Flow

1. **Planning Phase**:
   - LLM analyzes your goal
   - Breaks it into tasks and steps
   - Identifies required tools
   - Saves plan to filesystem

2. **Execution Phase**:
   - Loads plan from filesystem
   - Executes tasks respecting dependencies
   - Updates status after each step
   - Saves progress to tracking file

3. **Tracking**:
   - Real-time status updates
   - Persistent state for resume capability
   - Progress percentage calculation

## Examples

### Example 1: Code Analysis Task

```bash
# Create a plan for code analysis
./rago agent plan "Analyze the Go codebase for security vulnerabilities and generate a report"

# Output:
# 🤔 Creating plan for: Analyze the Go codebase...
# 📋 Plan created: abc123def4
# 📝 Summary: Comprehensive security analysis of Go codebase
# 📊 Tasks: 4, Total steps: 12
# ✅ Plan saved. Execute with: rago agent run-plan abc123def4

# Execute the plan
./rago agent run-plan abc123def4

# Check progress
./rago agent plan-status abc123def4
```

### Example 2: Documentation Generation

```bash
# Auto-run for immediate execution
./rago agent auto-run "Generate API documentation from code comments" -v

# This will:
# 1. Create a plan with tasks like:
#    - Scan codebase for API endpoints
#    - Extract comments and annotations
#    - Generate markdown documentation
#    - Create examples
# 2. Execute each task immediately
# 3. Show progress in real-time
```

### Example 3: Database Optimization

```bash
# Create plan
PLAN_ID=$(./rago agent plan "Optimize database queries in the application" | grep "Plan created:" | awk '{print $3}')

# Execute with monitoring
./rago agent run-plan $PLAN_ID -v

# If it fails, resume from where it stopped
./rago agent run-plan $PLAN_ID --resume
```

## With MCP Tools

When MCP servers are configured, the planner will:
1. Discover available tools
2. Include them in planning
3. Use them during execution

Example with MCP tools:
```bash
# The planner will use filesystem, github, and other MCP tools
./rago agent auto-run "Create a new feature branch and add unit tests"
```

## Library Usage

```go
import "github.com/liliang-cn/rago/v2/pkg/agents/planner"

// Create planner
agentPlanner := planner.NewAgentPlanner(llmProvider, storageDir)

// Create a plan
plan, err := agentPlanner.CreatePlan(ctx, "Your goal here")

// Execute the plan
executor := planner.NewPlanExecutor(agentPlanner, mcpClient)
err = executor.ExecutePlan(ctx, plan.ID)

// Check progress
progress, err := executor.GetPlanProgress(plan.ID)
fmt.Printf("Progress: %.1f%%\n", progress.PercentComplete)
```

## Tips

1. **Start Simple**: Begin with simple, well-defined goals
2. **Use Verbose Mode**: Add `-v` to understand what's happening
3. **Check Status**: Use `plan-status` to monitor long-running plans
4. **Resume on Failure**: Plans are persistent, use `--resume` to continue
5. **Review Plans**: Check the plan before execution with `list-plans`

## Troubleshooting

- **Plan Creation Timeout**: Ensure your LLM provider is running (e.g., Ollama)
- **Execution Failures**: Check MCP server status with `./rago mcp status`
- **Storage Issues**: Ensure write permissions to `~/.rago/agents/`