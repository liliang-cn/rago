---
name: agent-go
description: Use AgentGo (github.com/liliang-cn/agent-go/v3) as a Go library to build AI agents. Use when the user is writing Go code that imports github.com/liliang-cn/agent-go/v3 and needs help with agent.New() / tools / sub-agents / the run loop / output lints / checkpoints and long runs / memory / MCP / skills / the LLM provider pool.
license: MIT
compatibility: Go 1.21+ project that imports github.com/liliang-cn/agent-go/v3. SQLite via pure-Go modernc.org/sqlite (no CGO). Optional Goja for PTC. Optional embedding model for RAG.
metadata:
  project: agent-go
  version: "3.0"
  homepage: https://github.com/liliang-cn/agent-go
  import: github.com/liliang-cn/agent-go/v3
---

# AgentGo: a Go library for building agents

Use this skill when the user wants to **embed AgentGo in a Go program**: build an agent, give it tools, keep memory, run long tasks, or watch a run. AgentGo is a library. Your program owns the UI, the server and the storage.

## Install

```bash
go get github.com/liliang-cn/agent-go/v3
```

SQLite comes from pure-Go `modernc.org/sqlite`, so `CGO_ENABLED=0` builds work.

## Three main features

1. **LLM provider pool.** A pool client is one provider with one model. A run can ask for a model (`WithModel`), and the pool falls back to its own strategy when nothing matches. Transient errors are retried, and an optional parameter a provider rejects (`tool_choice`, `response_format`) is dropped and the call retried once.
2. **One loop.** Every run goes through `Runtime.loop`: assemble context, call the model, run tools, check the answer, stop. `Run` is `RunStream` plus a collector.
3. **MCP and Skills.** MCP tools sit beside built-in tools in one list. A skill is a `SKILL.md` with `when_to_use` and `paths`; each turn only the relevant few are surfaced.

## Mental model: seven concepts

- **Agent**: a name, a system prompt, a model, a tool set, optional sub-agents. Build with `agent.New("name").With...().Build()`.
- **Loop**: the one state machine. `Service.Run`, `RunStream`, `Ask`, `Chat` all enter it through `startRun`.
- **Tool**: everything the agent can do. Built-ins, MCP, skills and sub-agents are all tools.
- **Context**: message assembly, history filtering, compaction, skill reminders.
- **Hooks + Lints**: the deterministic layer. Hooks bracket the run and each tool; lints reject a final answer and force a retry.
- **Session + Checkpoint**: a session UUID owns the conversation; every terminal state writes a checkpoint.
- **Events**: the single output channel.

There is no team, dispatcher, router or handoff. To compose agents, use `WithSubagents(...)`, which adds a `task(agent_name, prompt)` tool.

## Give it a model

```go
llm, _ := pool.NewClient("main", baseURL, apiKey, "deepseek-v4-flash")
svc, err := agent.New("assistant").
    WithPrompt("You are a concise Go assistant.").
    WithLLM(llm).
    Build()
if err != nil { return err }
defer svc.Close()

reply, err := svc.Ask(ctx, "What is AgentGo?")
```

Without `WithLLM`, providers come from `agentgo.db` under `AGENTGO_HOME` (default `~/.agentgo`). `config.Load()` reads no config file, and the `agentgo.toml` at the repo root is not read by anything. With no provider configured, the first call fails with "no LLM provider is configured".

## Give it a tool

```go
type RevenueArgs struct {
    Month string `json:"month" desc:"Month as YYYY-MM" required:"true"`
}

revenue := agent.NewTool("store_revenue", "Revenue of each store in a month",
    func(ctx context.Context, a *RevenueArgs) (any, error) {
        return db.RevenueByStore(ctx, a.Month)
    }).WithReadOnly(true)

svc, _ := agent.New("analyst").WithTool(revenue).Build()
```

The struct's `json`, `desc`, `required` and `enum` tags generate the JSON Schema. Declare `ReadOnly`, `ConcurrencySafe`, `Destructive` and `InterruptBehavior` honestly: batching, permissioning and cancel depend on them.

## Sub-agents

```go
svc, _ := agent.New("lead").
    WithSubagents(agent.SubagentSpec{
        Name:         "researcher",
        Description:  "Looks things up and reports back",
        Instructions: "Answer only from what you find.",
        Tools:        []string{"fs_read"},
    }).
    Build()
```

The sub-agent runs through the same loop with its own session and a narrower tool surface, and returns only its final answer. `WithDelegation(false)` withholds the three generic delegation tools when `task` should be the only route.

## Output lints: move "please don't" out of prompts

When the model keeps making the same mistake, register a lint instead of adding another sentence to the prompt:

```go
svc.RegisterOutputLint(agent.LintFunc{
    NameValue: "no_planning_only_finish",
    Fn: func(text string, ctx agent.LintContext) (bool, string) {
        if strings.HasSuffix(strings.TrimSpace(text), "Next steps:") {
            return false, "the reply reads like a plan; deliver the work"
        }
        return true, ""
    },
})
```

On a violation the runtime appends the reason and asks again, bounded by `defaultLintRetryBudget = 2`; after that the run blocks. Every service from `agent.New(...).Build()` already has the built-in set (`non_empty_final_answer`, `file_task_must_write`, `task_delivery_contract`, and others, listed in CLAUDE.md). Feedback to the model carries the reason only, never the lint's name.

## Long tasks, checkpoints, and watching a run

- `svc.RunSegments(ctx, goal, agent.LongRunConfig{...})` drives one task across many runs. Each segment gets a fresh session; the plan and the workspace carry continuity.
- Every terminal state writes a checkpoint. `manager.Tasks().ResumeFromCheckpoint(ctx, taskID, agent.CheckpointResumeOptions{FollowUp: "..."})` resumes from the latest one. `agent.New(...).Build()` wires a default sink over the service's own store.
- `agent.NewActivityLog(w)` is an Observer that narrates a run, one line per model turn, tool call and checkpoint. `svc.StatusSnapshot()` answers "what is it doing now" from outside the run.
- `svc.Cancel()`, `CancelRun(runID)` and `CancelSession(id)` stop runs. A stop is an outcome (`Cancelled`), not an error.

## Behavioral eval (optional)

```go
import evalrunner "github.com/liliang-cn/agent-go/v3/eval/runner"

results, _ := evalrunner.RunAll(ctx, "eval/scenarios", evalrunner.RunOptions{
    Live: yourLiveBuilder, // nil → mock-only
})
fmt.Print(evalrunner.FormatSummary(results))
```

Scenario format and the live-builder contract are in [references/harness.md](references/harness.md).

## Conventions to follow

- **Identity is the session UUID**, not a user id. Use `github.com/google/uuid`.
- **`task_id` is load-bearing.** Scope new state (history, memory, discovered tools) by task.
- **Never decide behavior by matching phrases in the user's request.** Extract a structured constraint, or add an explicit option.
- **Anything that reaches the prompt prefix must be byte-stable across a run.** Sort tool lists; keep timestamps coarse.
- **RAG is opt-in.** Do not gate basic features on embeddings being configured.
- **Use random high ports** (3000+, e.g. 3076, 6759, 43510) for any dev server.
- **The framework reports tokens, not money.** Multiply `ExecutionResult.Usage` by your own rates.

## Storage layout (default `~/.agentgo/`)

```
~/.agentgo/
├── data/
│   ├── agentgo.db   # providers, agents, sessions, tasks, plans, checkpoints
│   └── cortex.db    # optional memory/vector/graph
├── memories/        # file memory when enabled
├── skills/          # local skills (SKILL.md format)
└── workspace/       # agent working directory
```

Override with `AGENTGO_HOME=/path/to/home`. Tests should use `t.TempDir()`.

## Key references

- [Getting started in a real host](../../docs/getting-started.md): the right way to wire model, memory, search and plans
- [API patterns](references/api-patterns.md): code recipes for every public capability
- [Harness engineering](references/harness.md): lints, eval scenarios, checkpoints in depth
- [Testing your AgentGo app](references/testing.md): `*testing.T` helpers, mock LLM, fixtures
