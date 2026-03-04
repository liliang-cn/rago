# Rago API Reference

## Agent Builder

```go
import "github.com/liliang-cn/rago/v2/pkg/agent"

svc, err := agent.New("my-agent").
    WithRAG().                            // Enable RAG
    WithMCP().                            // Enable MCP tools
    WithSkills().                         // Enable Skills
    WithMemory().                         // Enable Cognitive Memory
    WithMemoryReflect(5).                 // Reflect facts into observations after 5 entries
    WithMemoryHybrid().                  // Enable Parallel Vector + Index Reasoning
    WithMemoryBank("You are an expert", []string{"Never lie"}). // Mission/Directives
    WithDBPath("~/.rago/data/agent.db").  // Storage path
    Build()
```

### Inline tools in the builder chain

```go
svc, err := agent.New("my-agent").
    WithTool(agent.NewTool("get_time", "Returns current time",
        map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
        func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
            return time.Now().Format(time.RFC3339), nil
        },
    )).
    Build()
```

## Querying the Agent

### Ask — simplest API (no session, returns plain string)

```go
reply, err := svc.Ask(ctx, "What is the capital of France?")
```

### Chat — multi-turn conversation with cognitive memory

```go
result, err := svc.Chat(ctx, "My name is Alice and I work on the Go team.")
result, err  = svc.Chat(ctx, "What team am I on?")

fmt.Println(result.Text())        // "You're on the Go team, Alice."
```

## ExecutionResult

```go
type ExecutionResult struct {
    FinalResult interface{}  
    Error       string
    SessionID   string
    Sources     []domain.Chunk         // RAG sources
    Memories    []*domain.MemoryWithScore // Retrieved Cognitive Memories
    MemoryLogic string                 // Reasoning from Index Navigator
    PTCResult   *PTCResult             // PTC execution details
}

result.Text()        // string — preferred accessor
result.Err()         // error — nil if no error
result.HasSources()  // bool  — true if RAG returned chunks
```

## Memory Management (Cognitive Layer)

Beyond simple retrieval, the `MemoryService` allows managing the cognition path.

### Trigger Reflection
```go
// Manually reflect facts in a session into higher-level Observations.
summary, err := svc.TriggerReflection(ctx, sessionID)
```

### Trace Memory Evolution
```go
// Get the evolution graph: Fact -> Observation -> SupersededBy.
node, err := svc.ExplainMemory(ctx, memoryID)
// node.Memory contains the memory at this node
// node.Children contains memories that superseded this one
// node.EvidenceOf contains the observation this fact supports
```

### Memory API (domain.MemoryService)
| Method | Description |
|---|---|
| `Reflect(ctx, sessionID)` | LLM-driven consolidation of raw facts. |
| `GetEvolution(ctx, id)` | Trace the cognitive path of a memory. |
| `AddMentalModel(ctx, m)` | Insert human-curated rules/summaries. |

## Streaming

```go
// Stream — yields tokens as they arrive
for token := range svc.Stream(ctx, "Write a poem") {
    fmt.Print(token)
}

// RunStream — full event stream (tool calls, RAG sources, errors)
events, err := svc.RunStream(ctx, "Complex task")
for evt := range events {
    switch evt.Type {
    case agent.EventTypePartial:   fmt.Print(evt.Content)   // token
    case agent.EventTypeToolCall:  fmt.Println(evt.ToolName) // tool invoked
    }
}
```

## CLI Commands

### Chat with Memory Visibility
```bash
# Show retrieved memories, source types, and navigator reasoning
rago chat "What's my tech stack?" --show-memory
```

### Enable PTC (JS Sandbox)
```bash
# Run agent in Programmatic Tool Calling mode
rago chat "Compare these three cities" --with-ptc
```

## Memory Storage Layout

| Location | Content | Format |
|---|---|---|
| `data/memories/` | Raw facts and observations | Markdown + YAML |
| `data/memories/_index/` | PageIndex-style summaries | Markdown |
| `data/rago.db` | Vector index for acceleration | SQLite-vec |
