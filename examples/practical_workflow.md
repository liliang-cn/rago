# Practical Workflow Examples with RAGO

## Example 1: Daily News Digest

Create a workflow that fetches news from multiple sources and creates a daily digest.

### Step 1: Generate the Workflow

```bash
# Use the template generator for a quick start
rago agent generate-template "fetch news from multiple sources and create daily summary"

# Or use the LLM generator for more customization
rago agent generate "Create a workflow that fetches news from RSS feeds, filters for tech topics, and creates a markdown summary"
```

### Step 2: Customize the Generated Workflow

Edit the generated JSON to add your specific news sources:

```json
{
  "steps": [
    {
      "id": "fetch_hackernews",
      "name": "Fetch HackerNews Top Stories",
      "type": "tool",
      "tool": "fetch",
      "inputs": {
        "url": "https://hacker-news.firebaseio.com/v0/topstories.json"
      },
      "outputs": {
        "body": "hn_stories"
      }
    },
    {
      "id": "fetch_techcrunch",
      "name": "Fetch TechCrunch RSS",
      "type": "tool",
      "tool": "fetch",
      "inputs": {
        "url": "https://techcrunch.com/feed/"
      },
      "outputs": {
        "body": "tc_feed"
      }
    },
    {
      "id": "analyze_content",
      "name": "Analyze and Summarize",
      "type": "tool",
      "tool": "sequential-thinking",
      "inputs": {
        "task": "Create a digest of the top 5 most interesting tech stories from these sources. Include title, summary, and why it matters.",
        "hackernews": "{{hn_stories}}",
        "techcrunch": "{{tc_feed}}"
      },
      "outputs": {
        "result": "news_digest"
      }
    },
    {
      "id": "save_digest",
      "name": "Save Daily Digest",
      "type": "tool",
      "tool": "filesystem",
      "inputs": {
        "action": "write",
        "path": "./digests/tech_news_{{date}}.md",
        "content": "# Tech News Digest\n\n{{news_digest}}"
      }
    }
  ]
}
```

### Step 3: Create and Schedule the Agent

```bash
# Create the agent
rago agent create --name "Daily Tech Digest" --type workflow --workflow-file news_digest.json

# Schedule it to run every morning at 8 AM
rago agent schedule agent_xyz123 --cron "0 8 * * *"

# Or run it manually whenever you want
rago agent execute agent_xyz123
```

## Example 2: Project Documentation Generator

Automatically generate and update project documentation.

### Step 1: Generate Base Workflow

```bash
rago agent generate-template "analyze code files and generate documentation"
```

### Step 2: The Generated Workflow Will:

1. **Scan project files** - List all source code files
2. **Extract documentation** - Parse comments and docstrings
3. **Generate structure** - Create a project structure diagram
4. **Build README** - Compile everything into documentation
5. **Update timestamps** - Track when docs were last generated

### Step 3: Execute

```bash
# One-time documentation generation
rago agent execute agent_abc456

# Or integrate into your CI/CD pipeline
rago agent execute agent_abc456 --output docs/
```

## Example 3: Website Change Monitor

Monitor competitor websites or important pages for changes.

### Quick Setup:

```bash
# Generate a monitoring workflow
rago agent generate-template "monitor https://example.com/pricing for changes every hour"

# The workflow will:
# - Fetch the webpage
# - Compare with previous version
# - Detect changes
# - Log changes with timestamps
# - Create alerts for significant changes

# Create and schedule the agent
rago agent create --name "Pricing Monitor" --workflow-file testdata/website_monitor_workflow.json
rago agent schedule agent_monitor123 --cron "0 * * * *"  # Every hour
```

### Check Results:

```bash
# View recent executions
rago agent logs agent_monitor123

# Check the change log
cat website_changes.log

# Look for alerts
ls PRICE_ALERT_*.txt
```

## Example 4: Data Backup Automation

Create automated backups with verification.

```bash
# Generate backup workflow
rago agent generate-template "backup important files daily with verification"

# The workflow will:
# 1. List files in source directory
# 2. Create timestamped backup folder
# 3. Copy files to backup
# 4. Verify backup integrity
# 5. Log backup completion
# 6. Clean up old backups (optional)

# Schedule daily backups at 2 AM
rago agent schedule agent_backup --cron "0 2 * * *"
```

## Example 5: API Data Collector

Collect data from APIs and build datasets.

```bash
# Generate data collection workflow
rago agent generate-template "fetch weather data from api every hour and save to csv"

# Customize with your API endpoints
# Add data transformation steps
# Set up aggregation rules

# Execute on demand or schedule
rago agent execute agent_weather --params '{"city": "San Francisco"}'
```

## Tips for Creating Effective Workflows

1. **Start with templates** - Use `generate-template` for common patterns
2. **Use LLM for complex logic** - Use `generate` when you need custom business logic
3. **Test incrementally** - Run workflows manually before scheduling
4. **Monitor outputs** - Check logs regularly: `rago agent logs [id]`
5. **Version control** - Keep workflow JSON files in git
6. **Use variables** - Make workflows reusable with input parameters
7. **Add error handling** - Use conditions to handle failures gracefully

## Debugging Workflows

```bash
# Run in verbose mode to see each step
rago agent execute agent_id -v

# Check agent status
rago agent status agent_id

# View execution history
rago agent history agent_id

# Inspect workflow definition
rago agent get agent_id --format json
```

## Advanced: Chaining Workflows

You can create workflows that trigger other workflows:

```json
{
  "steps": [
    {
      "id": "trigger_analysis",
      "name": "Trigger Analysis Workflow",
      "type": "tool",
      "tool": "filesystem",
      "inputs": {
        "action": "execute",
        "command": "rago agent execute agent_analysis"
      }
    }
  ]
}
```

This enables complex automation pipelines where workflows can orchestrate other workflows based on conditions.