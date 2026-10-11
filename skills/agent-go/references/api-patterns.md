# AgentGo API Patterns

## Minimal Agent

```go
svc, err := agent.New("assistant").
	WithPrompt("You are a concise assistant.").
	WithLLM(llm). // llm from pool.NewClient(name, baseURL, apiKey, model)
	Build()
if err != nil {
	return err
}
defer svc.Close()

reply, err := svc.Ask(ctx, "What is AgentGo?")
```

Without `WithLLM`, providers come from `agentgo.db` under `AGENTGO_HOME`. With none configured, the first call fails with "no LLM provider is configured".

## Chat With Memory

```go
svc, err := agent.New("assistant").
	WithLLM(llm).
	WithMemory().
	Build()
if err != nil {
	return err
}
defer svc.Close()

_, _ = svc.Chat(ctx, "My name is Alice.")
result, err := svc.Chat(ctx, "What do you know about me?")
```

`WithMemory(agent.WithMemoryStoreType("file"))` needs no embedder and no server. To say whose memory a run uses, pass `agent.WithMemoryUser(id)` to `Run`.

## Custom Tools

The tool's parameters are a Go struct. The `json`, `desc`, `required` and `enum` tags generate the JSON Schema.

```go
type StatusArgs struct {
	Service string `json:"service" desc:"Service name" required:"true"`
}

status := agent.NewTool("fetch_status", "Fetch a service's status (read-only).",
	func(ctx context.Context, a *StatusArgs) (any, error) {
		return fetchStatus(ctx, a.Service)
	}).
	WithReadOnly(true)

svc, _ := agent.New("ops").WithLLM(llm).WithTool(status).Build()
```

`agent.BuildTool(name)` is the builder form for tools whose schema is built by hand. Declare `ReadOnly`, `ConcurrencySafe`, `Destructive` and `InterruptBehavior` honestly: batching, permission prompts and cancel depend on them.

## Sub-agents

```go
svc, _ := agent.New("lead").
	WithLLM(llm).
	WithSubagents(agent.SubagentSpec{
		Name:         "researcher",
		Description:  "Looks things up and reports back",
		Instructions: "Answer only from what you find.",
		Tools:        []string{"fs_read"},
	}).
	Build()
```

This registers one tool, `task(agent_name, prompt)`. The sub-agent runs through the same loop with its own session and returns only its final answer. `WithDelegation(false)` withholds the three generic delegation tools.

## Streaming a run

`RunStream` is the primary API; `Run` calls it and collects the events.

```go
events, err := svc.RunStream(ctx, "summarize the README in three bullets")
if err != nil {
	return err
}
for evt := range events {
	switch evt.Type {
	case agent.EventTypePartial:
		fmt.Print(evt.Content) // streaming token
	case agent.EventTypeToolCall:
		log.Printf("tool: %s", evt.ToolName)
	case agent.EventTypeComplete:
		// final text in evt.Content
	case agent.EventTypeBlocked, agent.EventTypeError, agent.EventTypeCancelled:
		// the run ended without a normal completion
	}
}
```

## Persistent tasks (Manager)

`agent.Manager` hosts named agents over a `Store` and exposes a task surface. It is an application-level host, not an orchestrator.

```go
store, err := agent.NewStore("agentgo.db")
if err != nil {
	return err
}
manager := agent.NewManager(store)

task, err := manager.Tasks().Submit(ctx, agent.TaskSubmitOptions{
	SessionID: "demo-session",
	AgentName: "assistant",
	Input:     "Check the repository status.",
})
if err != nil {
	return err
}
done, err := manager.Tasks().Await(ctx, task.ID)
```

Subscribe to a task's live events instead of blocking:

```go
events, unsub, _ := manager.SubscribeTask(task.ID)
defer unsub()
for evt := range events {
	if evt.Type == agent.TaskEventTypeRuntime && evt.Runtime != nil {
		// evt.Runtime is the same *agent.Event as the per-Service stream
	}
}
```

## Output lints

Register a deterministic check on the model's free-form final answer. A failure re-prompts the model with the reason.

```go
svc.RegisterOutputLint(agent.LintFunc{
	NameValue: "must_cite_source",
	Fn: func(text string, ctx agent.LintContext) (bool, string) {
		if !strings.Contains(text, "source:") {
			return false, "the answer must include a 'source:' citation"
		}
		return true, ""
	},
}, "researcher") // no agent names = global
```

Every service from `agent.New(...).Build()` already carries the built-in set (see `harness.md`). The feedback carries the reason only; never put the lint's name in it.

## Checkpoints, resume, and long runs

Every terminal state writes a checkpoint. `agent.New(...).Build()` wires a default sink over the service's own store; `Manager` wires itself in. Resume from the latest one:

```go
resumed, err := manager.Tasks().ResumeFromCheckpoint(ctx, taskID,
	agent.CheckpointResumeOptions{FollowUp: "and now also do X"})
```

For a task that takes many rounds, `RunSegments` calls `Run` again and again, each segment in a fresh session:

```go
res, err := svc.RunSegments(ctx, "port the module to the new API",
	agent.LongRunConfig{MaxSegments: 20, RoundsPerSegment: 30})
if res.Done() {
	// only LongRunStopFinished means the work is done
}
```

## Run options

Pass to `Run` to override defaults per call:

```go
result, err := svc.Run(ctx, "do the thing",
	agent.WithSessionID("custom-session-id"),
	agent.WithTaskID("custom-task-id"),
	agent.WithMaxTurns(8),
	agent.WithTemperature(0.1),
	agent.WithModel("a-small-model"),   // a preference, not a guarantee
	agent.WithMemoryUser("alice"),
	agent.WithResumeMessages(saved),    // start from a snapshot
)
```

`WithModel` is a preference: when nothing matches, the pool falls back to its strategy. `GenerationResult.Model` and `Provider` say what actually answered, and `ActiveRun.Model` shows it in the status snapshot.

## Watching a run

```go
svc, _ := agent.New("assistant").
	WithLLM(llm).
	WithObserver(agent.NewActivityLog(os.Stderr)).
	Build()

snap := svc.StatusSnapshot() // what is running now, from outside the run
```

## Where to look in the codebase

- Builder and Service public API: `pkg/agent/builder.go`, `pkg/agent/service.go`
- The loop: `pkg/agent/runtime.go`, with `loop_context.go`, `tool_prep.go`, `tool_round.go`
- Tools: `pkg/agent/tool.go`, `pkg/agent/tool_builder.go`
- Sub-agents: `pkg/agent/subagent_tool.go`, `pkg/agent/subagent.go`
- TaskService (Submit / Await / Resume / ResumeFromCheckpoint): `pkg/agent/task_service.go`, `pkg/agent/task_checkpoint_manager.go`
- Long runs: `pkg/agent/long_run.go`
- Output lints: `pkg/agent/output_lint.go`, `output_lints_builtin.go`
- Provider pool and compatibility fallbacks: `pkg/pool/`
- Eval runner: `eval/runner/`
