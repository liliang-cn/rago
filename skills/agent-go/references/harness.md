# Harness Engineering With AgentGo

This file is the in-depth companion to the Harness section in `SKILL.md`.
Read it when the user wants reliability — lints to catch recurring agent
mistakes, checkpoints to recover crashed runs, behavioral eval to measure
"is my change actually better".

## Output lints

### Why they exist

The model keeps making the same mistake (an agent ending with "Next
steps:..." instead of finishing; a run that ends with no text at all; a
task that asked for a file finishing without writing one). The reflexive fix is to add another sentence to
the system prompt. That doesn't scale: prompts grow, instructions get
ignored, and you have no signal when the rule fires.

A lint is the alternative: a deterministic check that runs on the
model's free-form final answer, before the runtime emits the
`EventTypeComplete` event. On violation, the runtime appends structured
feedback as a system message and re-prompts the model — bounded by
`defaultLintRetryBudget = 2`. Exhaust the budget → the task blocks
(the lint's name goes to observers and logs, never to the model).

### Registering one

```go
import "github.com/liliang-cn/agent-go/v3/pkg/agent"

svc.RegisterOutputLint(agent.LintFunc{
    NameValue: "must_cite_source",
    Fn: func(text string, ctx agent.LintContext) (bool, string) {
        if !strings.Contains(text, "source:") {
            return false, "response must include 'source:' citation"
        }
        return true, ""
    },
}, "researcher")  // no agent names → global
```

`LintContext` carries `AgentName / TaskID / SessionID / TurnIndex /
ToolCalls / IsRetry / RetryCount` so a lint can be context-aware.

### Built-ins

Every service from `agent.New(...).Build()` gets these automatically. There
are no agent-scoped or role-scoped lints in v3.

| Lint                              | What it rejects                                                                  |
| --------------------------------- | -------------------------------------------------------------------------------- |
| `no_planning_only_finish`         | a final text that reads like "Next steps:" / "I will ..."                        |
| `file_task_must_write`            | a task that asked for a file finishing without producing one                     |
| `non_empty_final_answer`          | a run terminating with no text at all (an image output counts as an answer)      |
| `task_delivery_contract`          | a goal naming a delivery action finishing with no matching tool actually called  |
| `requested_action_contract`       | a requested action (reminder, calendar entry, note) finishing with its tool unused |
| `no_tool_scaffolding_answer`      | the framework's own plumbing ("No tools found matching the query.") as the answer |
| `deliverable_block_must_carry_work` | a run blocked on a missing delivery tool that reports none of the work it could do |
| `plan_ready_steps_done`           | finishing while the plan still has ready, unchecked steps                        |

`LintContext` carries `ToolCalls` (what ran) and `AvailableTools` (what could
have run), so a lint can tell "the agent skipped a capability it had" from
"the agent never had it".

### Discipline

When you observe a recurring mistake in a real run:

1. Reproduce with a mock-LLM eval scenario that fails today.
2. Write the lint in `pkg/agent/output_lints_builtin.go` (or in your
   own app code).
3. Wire the lint, re-run the scenario; it should self-heal on retry.
4. Optionally: cut the corresponding sentence from the system prompt
   now that the runtime enforces it.

## Task checkpoint + replay

### What gets saved

`pkg/agent/runtime.go` calls `persistTerminalCheckpoint(...)` from both
`completeRun` and `blockRun`. The snapshot captures the entire message
history at that moment plus `final_text`, `agent_name`, `session_id`,
`task_id`, `round`. Stored in the `task_checkpoints` SQLite table.

Snapshots beyond `MaxCheckpointsPerTask = 32` are pruned by the
in-process `checkpointWriter` after each write.

### Resume API

```go
resumed, err := manager.Tasks().ResumeFromCheckpoint(ctx, taskID,
    agent.CheckpointResumeOptions{
        // CheckpointID: "<id>",  // optional; default = latest
        FollowUp: "and now also confirm with 'replay confirmed'",
    })
```

`ResumeFromCheckpoint` looks up the latest snapshot for `taskID`,
rebuilds the agent's message history with `WithResumeMessages(...)`,
and re-runs through the normal streaming pipeline. The original
`taskID` is reused so existing `SubscribeTask` clients keep working.

### Wiring for custom services

`Service` exposes `SetCheckpointSink(sink CheckpointSink)`. `Manager`
wires itself in when it builds a service, and `agent.New(...).Build()`
wires a default sink over the service's own store whenever it has one.
Call `svc.SetCheckpointSink(myImpl)` only to persist somewhere else.

A minimal sink:

```go
type checkpointSink struct{ store *agent.Store }

func (s checkpointSink) WriteCheckpoint(taskID string, reason agent.CheckpointReason, round int, sessionID, agentName, finalText, afterTool string, messages []domain.Message) error {
    return s.store.SaveTaskCheckpoint(&agent.TaskCheckpoint{
        TaskID: taskID, Round: round, AfterTool: afterTool,
        SessionID: sessionID, AgentName: agentName,
        Messages: messages, FinalText: finalText, CreatedAt: time.Now(),
    })
}
```

### Listing and replaying from code

```go
cps, _ := manager.Tasks().ListCheckpoints(ctx, taskID, 10)       // list snapshots
_, _ = manager.Tasks().ResumeFromCheckpoint(ctx, taskID,         // rerun from latest
    agent.CheckpointResumeOptions{FollowUp: "..."})              // append a user instruction
_, _ = manager.Tasks().ResumeFromCheckpoint(ctx, taskID,         // or a specific snapshot
    agent.CheckpointResumeOptions{CheckpointID: cps[0].ID})
```

## Behavioral eval

### Scenario YAML

```yaml
# eval/scenarios/gate_empty_final_answer_is_rejected.yaml
name: gate_empty_final_answer_is_rejected
description: a run that ends with no text is rejected; the second reply carries an answer.
agent: Assistant
register_lints:
  - non_empty_final_answer
input: "what did you find?"
llm_replies:
  - ""
  - "I could not find a matching tool, so here is what I know directly: nothing was scheduled."
expect:
  status: completed
  final_text_match: "nothing was scheduled"
  llm_calls: 2
  lint_violations:
    - lint: non_empty_final_answer
      count: 1
```

Required fields: `name`, `description`, `input`, `llm_replies` (mock
mode) **or** `mode: live` (real provider). Optional: `runs: N` for
live, `register_lints`, `max_lint_violations` (loose ceilings for
live), `expect.final_text_match` / `expect.final_text_must_not_match`
(prefix `re:` for regex), `expect.llm_calls`.

### Profiles

- **mock** (default, CI-safe): `MockLLM` plays back `llm_replies` in
  order. Deterministic. `make eval` runs every mock scenario.
- **live**: scenarios with `mode: live` use the configured provider
  pool (`make eval-live`, gated on `AGENTGO_EVAL_LIVE=1`). Use `runs: 2` or `runs: 3`
  to amortize non-determinism.

### Library API

```go
import evalrunner "github.com/liliang-cn/agent-go/v3/eval/runner"

opts := evalrunner.RunOptions{
    Live: func(scenarioName, agentName string, lints []string, home string) (*agent.Service, error) {
        return buildLiveService(home, agentName) // your code, returns a real-LLM Service
    },
}
results, err := evalrunner.RunAll(ctx, "eval/scenarios", opts)
fmt.Print(evalrunner.FormatSummary(results))
_, _ = evalrunner.SaveResults(results, "live", "eval/results")
```

`results` is `[]*RunResult`; each has `PassCount / FailCount /
AvgLLMCalls / AvgDurationMs / LintViolations` so you can diff between
runs.

### Discipline (the harness loop)

1. Make a runtime / lint / prompt change.
2. `make eval` (mock) — must stay green.
3. `make eval-live` — runs against the real provider.
4. `git diff eval/results/<prev>.json eval/results/<latest>.json` —
   look at `pass_count` per scenario, `avg_llm_calls`,
   `avg_duration_ms`, `lint_violations` deltas.
5. If pass rate dropped or LLM calls per scenario rose: revert or investigate
   before merging.

This is the loop OpenAI's "harness engineering" piece argues for —
the test isn't whether the code compiles, the test is whether the
agent's behavior actually improved.
