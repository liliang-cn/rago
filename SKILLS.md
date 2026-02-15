# RAGO Skills

Skills are a lightweight, open format for extending AI agent capabilities with specialized knowledge and workflows. In RAGO, skills follow the [Agent Skills](https://agentskills.io) standard, enabling agents to discover and execute complex tasks efficiently.

## Core Concept

A **Skill** is a folder containing a `SKILL.md` file. This file uses YAML frontmatter to define metadata and variables, followed by Markdown instructions that guide the LLM or agent through a specific process.

### Discovery & Activation
- **Discovery:** RAGO loads only the metadata (name, description, tags) of available skills to keep the context window clean.
- **Activation:** When an agent determines a skill is relevant to a user request, it "activates" the skill by loading its full instructions into the conversation context.

## Skill Structure

A typical skill directory looks like this:

```text
my-skill/
├── SKILL.md          # Required: Metadata and instructions
├── scripts/          # Optional: Executable scripts or code
├── references/       # Optional: Documentation or data files
└── assets/           # Optional: Templates or static resources
```

### SKILL.md Format

The `SKILL.md` file must contain a YAML frontmatter block:

```markdown
---
name: code-reviewer
description: "Analyze code quality and provide refactoring suggestions."
version: 1.0.0
category: development
tags: [code, review, quality]
variables:
  - name: code
    type: string
    required: true
    description: "The source code to review"
---

# Code Reviewer Instructions

You are an expert software architect. When this skill is activated:

## Step 1: Analyze Logic
Check for edge cases and logical errors...

## Step 2: Review Style
Ensure the code follows project conventions...
```

## Creating a Skill

1. Create a new directory in your skills path (default: `~/.rago/skills/`).
2. Create a `SKILL.md` file with the required frontmatter.
3. Define `variables` if your skill needs input parameters.
4. Write clear, step-by-step instructions in Markdown.

## Managing Skills via CLI

RAGO provides a built-in command for skill management:

```bash
# List all loaded skills
rago skills list

# Show details of a specific skill
rago skills show code-reviewer

# Execute a skill manually
rago skills run code-reviewer --var code="$(cat main.go)"

# Reload skills from disk
rago skills load
```

## Integration with LLMs

When RAGO's agentic planner is used, it can automatically select and apply skills based on their descriptions. Developers can also programmatically trigger skills via the `pkg/skills` service.

### Example: Programmatic Usage

```go
// Get the skills service
skillsService, _ := services.GetSkillsService()

// Execute a skill
result, err := skillsService.Execute(ctx, &skills.ExecutionRequest{
    SkillID: "code-reviewer",
    Variables: map[string]interface{}{
        "code": myCode,
    },
})
```

## Best Practices

- **Atomic Skills:** Keep skills focused on a single, well-defined task.
- **Clear Descriptions:** Use precise descriptions to help the discovery mechanism match the right skill.
- **Variable Documentation:** Always include descriptions for variables to guide the agent in providing the correct inputs.
- **Step-by-Step:** Use `## Step N: ...` headers to structure complex workflows, as RAGO's loader can parse these into individual execution steps.
