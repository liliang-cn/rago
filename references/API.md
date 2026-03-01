# Rago API Reference

## Agent Builder

```go
import "github.com/liliang-cn/rago/v2/pkg/agent"

svc, err := agent.New("my-agent").
    WithRAG().                           // Enable RAG
    WithMCP().                           // Enable MCP
    WithSkills().                        // Enable Skills
    WithMemory().                        // Enable Memory
    WithDBPath("~/.rago/data/agent.db"). // Storage
    WithDebug(true).                     // Debug mode
    Build()
```

## Agent.Run

```go
result, err := svc.Run(ctx, "Your goal",
    agent.WithMaxTurns(20),
    agent.WithTemperature(0.7),
    agent.WithStoreHistory(true),
    agent.WithSessionID("resume-session-id"),
)
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
