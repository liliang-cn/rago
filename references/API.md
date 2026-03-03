# Rago API Reference

## Agent Builder

```go
import "github.com/liliang-cn/rago/v2/pkg/agent"

svc, err := agent.New("my-agent").
    WithRAG().                            // Enable RAG (self-registers rag_query tool)
    WithMCP().                            // Enable MCP tools
    WithSkills().                         // Enable Skills
    WithMemory().                         // Enable Memory (self-registers memory_save/recall)
    WithDBPath("~/.rago/data/agent.db").  // Storage path
    WithSystem("You are a helpful assistant."). // System prompt
    WithDebug().                          // Debug mode (variadic: WithDebug(false) to disable)
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
    WithTools(tool1, tool2, tool3). // multiple at once
    Build()
```

### Progress callback

```go
svc, err := agent.New("my-agent").
    WithProgress(func(msg string) { fmt.Print(msg) }).
    Build()
```

### PTC (Programmatic Tool Calling)

```go
svc, err := agent.New("my-agent").
    WithPTC().   // PTC is a transport mode, not a separate API
    Build()

// API is identical — PTC executes tools via JS sandbox internally
result, err := svc.Chat(ctx, "Write and run some code")
```

## Querying the Agent

### Ask — simplest API (no session, returns plain string)

```go
reply, err := svc.Ask(ctx, "What is the capital of France?")
// reply == "The capital of France is Paris."
```

### Chat — multi-turn conversation

```go
// Session UUID auto-generated on first call; preserved across Chat() calls
result, err := svc.Chat(ctx, "My name is Alice")
result, err  = svc.Chat(ctx, "What's my name?")  // remembers "Alice"

fmt.Println(result.Text())        // plain string reply
fmt.Println(result.Err())         // non-nil if agent reported an error
fmt.Println(result.HasSources())  // true if RAG sources were used
```

### Run — single call with full options

```go
result, err := svc.Run(ctx, "Your goal",
    agent.WithMaxTurns(20),
    agent.WithTemperature(0.7),
    agent.WithStoreHistory(true),
    agent.WithSessionID("resume-session-id"),
)
fmt.Println(result.Text())
```

## Streaming

```go
// Stream — simplest streaming, yields tokens as they arrive (one-shot, no session)
for token := range svc.Stream(ctx, "Write a poem") {
    fmt.Print(token)
}

// ChatStream — multi-turn streaming with session memory
for token := range svc.ChatStream(ctx, "Tell me more") {
    fmt.Print(token)
}

// RunStream — full event stream (tool calls, RAG sources, errors)
events, err := svc.RunStream(ctx, "Complex task")
for evt := range events {
    switch evt.Type {
    case agent.EventTypePartial:   fmt.Print(evt.Content)   // token
    case agent.EventTypeToolCall:  fmt.Println(evt.ToolName) // tool invoked
    case agent.EventTypeComplete:  // done
    case agent.EventTypeError:     // error
    }
}
```

### Streaming API comparison

| Method | Returns | Session | Use case |
|--------|---------|---------|----------|
| `Stream()` | `<-chan string` | no | live output, one-off |
| `ChatStream()` | `<-chan string` | yes | live output, conversational |
| `RunStream()` | `(<-chan *Event, error)` | optional | full control, tool hooks |

## ExecutionResult

```go
type ExecutionResult struct {
    FinalResult interface{}  // always a string in practice; use Text()
    Error       string
    SessionID   string
    Sources     []domain.Chunk  // RAG sources (non-nil when HasSources() == true)
    PTCResult   *PTCResult       // PTC execution details (nil when PTC disabled)
    // ...
}

result.Text()        // string — preferred accessor
result.Err()         // error — nil if no error
result.HasSources()  // bool  — true if RAG returned chunks
```

## RAG Operations

### Ingest

```go
err := ragProcessor.Ingest(ctx, "./docs/", domain.IngestOptions{
    Collection:   "my-docs",
    ChunkSize:    512,
    Overlap:      50,
    FilePatterns: []string{"*.pdf", "*.md"},
})
```

### Query

```go
result, err := ragProcessor.Query(ctx, domain.QueryRequest{
    Query:       "search",
    Collection:  "my-docs",
    TopK:        5,
    ShowSources: true,
})
```

## LongRun

```go
longRun, _ := agent.NewLongRun(svc).
    WithInterval(10 * time.Minute).
    WithWorkDir("~/.rago/longrun").
    WithMaxActions(5).
    Build()

longRun.AddTask(ctx, "Task goal", nil)
longRun.Start(ctx)
longRun.Stop()
```

## SubAgent Coordinator

```go
coordinator := agent.NewSubAgentCoordinator()
resultChan := coordinator.RunAsync(ctx, subAgent)
results := coordinator.WaitAll(ctx)
coordinator.CancelAll()
```

## Memory Files

| File | Purpose |
|------|---------|
| `MEMORY.md` | Long-term memory |
| `AGENTS.md` | Agent config |
| `SOUL.md` | Personality |
| `HEARTBEAT.md` | Autonomous checklist |
