package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/liliang-cn/agent-go/pkg/domain"
)

// Module is implemented by any component that can self-register tools into
// the agent's ToolRegistry. Modules are the authoritative source for both
// the tool schema (visible to the LLM) and the handler (called on invocation).
//
// Usage:
//
//	agent.New("bot").
//	    WithModule(NewRAGModule(proc, onSources)).
//	    WithModule(NewMemoryModule(memSvc, onSaved)).
//	    Build()
type Module interface {
	// ID returns a unique identifier for this module (e.g. "rag", "memory").
	ID() string
	// RegisterTools registers all tools this module provides into registry.
	RegisterTools(registry *ToolRegistry) error
}

// ── RAG Module ────────────────────────────────────────────────────────────────

type ragModule struct {
	proc      domain.Processor
	onSources func([]domain.Chunk) // called after rag_query with retrieved chunks
}

// NewRAGModule creates a Module that registers rag_query and rag_ingest tools.
// onSources is invoked after each successful query so the caller can surface
// source attribution (may be nil).
func NewRAGModule(proc domain.Processor, onSources func([]domain.Chunk)) Module {
	return &ragModule{proc: proc, onSources: onSources}
}

func (m *ragModule) ID() string { return "rag" }

func (m *ragModule) RegisterTools(registry *ToolRegistry) error {
	registry.Register(domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "rag_query",
			Description: "Search the knowledge base for information",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Search query"},
					"top_k": map[string]interface{}{"type": "integer", "description": "Number of results (default 5)"},
				},
				"required": []string{"query"},
			},
		},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		query, _ := args["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("rag_query: 'query' argument is required")
		}
		topK := 5
		if tk, ok := args["top_k"].(float64); ok {
			topK = int(tk)
		} else if tk, ok := args["top_k"].(int); ok {
			topK = tk
		}
		resp, err := m.proc.Query(ctx, domain.QueryRequest{Query: query, TopK: topK})
		if err != nil {
			return nil, err
		}
		if m.onSources != nil {
			m.onSources(resp.Sources)
		}
		return map[string]interface{}{"answer": resp.Answer, "sources": len(resp.Sources)}, nil
	}, CategoryRAG)

	registry.Register(domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "rag_ingest",
			Description: "Ingest a document into the RAG knowledge base",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content":   map[string]interface{}{"type": "string", "description": "Document content to ingest"},
					"file_path": map[string]interface{}{"type": "string", "description": "Path to document file"},
				},
			},
		},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		content, _ := args["content"].(string)
		filePath, _ := args["file_path"].(string)
		if content == "" && filePath == "" {
			return nil, fmt.Errorf("rag_ingest: 'content' or 'file_path' is required")
		}
		_, err := m.proc.Ingest(ctx, domain.IngestRequest{Content: content, FilePath: filePath})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "ingested"}, nil
	}, CategoryRAG)

	return nil
}

// ── Memory Module ─────────────────────────────────────────────────────────────

type memoryModule struct {
	svc     domain.MemoryService
	onSaved func() // called after memory_save so the run loop can note it
}

// NewMemoryModule creates a Module that registers memory_save, memory_recall,
// memory_update, and memory_delete tools.
// onSaved is invoked after each save (may be nil).
func NewMemoryModule(svc domain.MemoryService, onSaved func()) Module {
	return &memoryModule{svc: svc, onSaved: onSaved}
}

func (m *memoryModule) ID() string { return "memory" }

func (m *memoryModule) RegisterTools(registry *ToolRegistry) error {
	registry.Register(domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "memory_save",
			Description: "Save information to long-term memory for future reference",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{"type": "string", "description": "The information to remember"},
					"type":    map[string]interface{}{"type": "string", "description": "Type: fact, preference, skill, pattern, context"},
				},
				"required": []string{"content"},
			},
		},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		content, _ := args["content"].(string)
		if content == "" {
			return nil, fmt.Errorf("memory_save: 'content' argument is required")
		}
		memType := "fact"
		if t, ok := args["type"].(string); ok && t != "" {
			memType = t
		}
		if m.onSaved != nil {
			m.onSaved()
		}
		err := m.svc.Add(ctx, &domain.Memory{
			Type:       domain.MemoryType(memType),
			Content:    content,
			Importance: 0.8,
			Metadata:   map[string]interface{}{"source": "tool_call"},
		})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "saved", "content": content}, nil
	}, CategoryMemory)

	registry.Register(domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "memory_recall",
			Description: "Recall information from long-term memory",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Query to search memory for"},
				},
				"required": []string{"query"},
			},
		},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		query, _ := args["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("memory_recall: 'query' argument is required")
		}
		memories, err := m.svc.Search(ctx, query, 5)
		if err != nil {
			return nil, err
		}
		if len(memories) == 0 {
			allMems, _, listErr := m.svc.List(ctx, 10, 0)
			if listErr == nil && len(allMems) > 0 {
				var out []string
				for _, mem := range allMems {
					out = append(out, fmt.Sprintf("- [%s] %s", mem.Type, mem.Content))
				}
				return map[string]interface{}{"memories": strings.Join(out, "\n")}, nil
			}
			return map[string]interface{}{"memories": ""}, nil
		}
		var out []string
		for _, mem := range memories {
			out = append(out, fmt.Sprintf("- [%s: %.2f] %s", mem.Type, mem.Score, mem.Content))
		}
		return map[string]interface{}{"memories": strings.Join(out, "\n"), "count": len(memories)}, nil
	}, CategoryMemory)

	registry.Register(domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "memory_update",
			Description: "Update an existing memory entry by its ID",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":      map[string]interface{}{"type": "string", "description": "Memory ID to update"},
					"content": map[string]interface{}{"type": "string", "description": "New content"},
				},
				"required": []string{"id", "content"},
			},
		},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		id, _ := args["id"].(string)
		content, _ := args["content"].(string)
		if id == "" || content == "" {
			return nil, fmt.Errorf("memory_update: 'id' and 'content' are required")
		}
		if err := m.svc.Update(ctx, id, content); err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "updated", "id": id}, nil
	}, CategoryMemory)

	registry.Register(domain.ToolDefinition{
		Type: "function",
		Function: domain.ToolFunction{
			Name:        "memory_delete",
			Description: "Permanently remove a memory entry by its ID",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "string", "description": "Memory ID to delete"},
				},
				"required": []string{"id"},
			},
		},
	}, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		id, _ := args["id"].(string)
		if id == "" {
			return nil, fmt.Errorf("memory_delete: 'id' argument is required")
		}
		if err := m.svc.Delete(ctx, id); err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "deleted", "id": id}, nil
	}, CategoryMemory)

	return nil
}
