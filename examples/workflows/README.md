# Workflow Examples

This directory contains workflow examples for the RAGO agent system, demonstrating how to create and use workflow automation with MCP tools.

## What are Workflows?

Workflows are JSON-defined sequences of steps that agents can execute to automate complex tasks. They leverage MCP (Model Context Protocol) tools to interact with various systems.

## Directory Contents

### Pre-built Workflow Templates

- **`website_monitor.json`** - Monitor a website for changes and log differences
- **`file_organizer.json`** - Organize files in directories based on type
- **`repo_analyzer.json`** - Analyze Git repository activity and generate reports

### Workflow Generator

- **`generator.go`** - Template-based workflow generator that creates workflows from natural language descriptions

## Using Pre-built Workflows

1. Create an agent with a workflow:
```bash
rago agent create --name "Website Monitor" --type workflow --workflow-file website_monitor.json
```

2. Execute the agent:
```bash
rago agent execute [agent-id]
```

3. Schedule the agent (optional):
```bash
rago agent schedule [agent-id] --cron "*/30 * * * *"
```

## Workflow Generation Options

RAGO provides two ways to generate workflows from descriptions:

### 1. Template-Based Generation (No LLM Required)

Use pre-defined templates for common workflow patterns:

```bash
# List available templates
rago agent generate-template --list

# Generate from templates
rago agent generate-template "monitor https://example.com for changes"
rago agent generate-template "process all JSON files and create a report"
rago agent generate-template "backup important files daily"

# Generate and execute immediately
rago agent generate-template -e "fetch data from api"

# Specify output file
rago agent generate-template -o my_workflow.json "organize files by type"
```

**Supported Templates:**
- **Website Monitoring** - Track changes on websites
- **File Processing** - Batch process files in directories
- **Data Fetching** - Retrieve and save data from APIs
- **Report Generation** - Analyze data and create reports
- **Backup Automation** - Create automated backups

### 2. LLM-Based Generation (Advanced)

For more complex or custom workflows, use the LLM-powered generator:

```bash
# Generate from description
rago agent generate "Monitor RSS feeds and create daily summaries"

# Interactive mode with refinement
rago agent generate -i "Complex data pipeline with error handling"

# Generate and execute immediately
rago agent generate -e "Fetch weather data and send alerts"
```

## Workflow Structure

Workflows consist of:

```json
{
  "steps": [
    {
      "id": "unique_id",
      "name": "Human readable name",
      "type": "tool|condition|loop|variable",
      "tool": "tool_name",
      "inputs": {},
      "outputs": {}
    }
  ],
  "triggers": [],
  "variables": {}
}
```

### Step Types

- **`tool`** - Execute an MCP tool
- **`condition`** - Conditional branching
- **`loop`** - Iterate over items
- **`variable`** - Set or transform variables

### Available MCP Tools

- **`filesystem`** - File operations (read, write, list, execute)
- **`fetch`** - HTTP/HTTPS requests
- **`memory`** - Temporary storage
- **`time`** - Date/time operations
- **`sequential-thinking`** - Complex reasoning tasks

## Creating Custom Workflows

1. Start with a template or generate one
2. Modify the JSON structure to fit your needs
3. Test with `rago agent validate --workflow-file your_workflow.json`
4. Create and execute the agent

## Best Practices

1. **Use meaningful step IDs** - Makes workflows easier to understand
2. **Store intermediate results** - Use outputs for data flow between steps
3. **Add error handling** - Use conditions to handle failures
4. **Test incrementally** - Start simple and add complexity
5. **Use variables** - Make workflows reusable with input variables

## Examples of Variable Usage

```json
{
  "variables": {
    "website_url": "https://example.com",
    "check_interval": 1800
  },
  "steps": [
    {
      "inputs": {
        "url": "{{website_url}}"
      }
    }
  ]
}
```

## Debugging Workflows

1. Use verbose mode: `rago agent execute [id] -v`
2. Check logs: `rago agent logs [id]`
3. Validate JSON: `rago agent validate --workflow-file workflow.json`

## Contributing

To add new workflow examples:
1. Create a descriptive JSON file
2. Test thoroughly
3. Document the use case
4. Submit a pull request