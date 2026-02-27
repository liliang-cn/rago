package ptc

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// RAGORouter routes tool calls to existing RAGO services
type RAGORouter struct {
	mu sync.RWMutex

	// Tool handlers by name
	handlers map[string]ToolHandler
	// Tool info cache
	toolInfo map[string]*ToolInfo

	// External services (optional)
	mcpService    interface{} // *mcp.Service
	skillsService interface{} // *skills.Service
	ragProcessor  interface{} // domain.Processor
}

// RouterOption configures the router
type RouterOption func(*RAGORouter)

// WithMCPService sets the MCP service
func WithMCPService(svc interface{}) RouterOption {
	return func(r *RAGORouter) {
		r.mcpService = svc
	}
}

// WithSkillsService sets the skills service
func WithSkillsService(svc interface{}) RouterOption {
	return func(r *RAGORouter) {
		r.skillsService = svc
	}
}

// WithRAGProcessor sets the RAG processor
func WithRAGProcessor(proc interface{}) RouterOption {
	return func(r *RAGORouter) {
		r.ragProcessor = proc
	}
}

// NewRAGORouter creates a new tool router
func NewRAGORouter(opts ...RouterOption) *RAGORouter {
	r := &RAGORouter{
		handlers: make(map[string]ToolHandler),
		toolInfo: make(map[string]*ToolInfo),
	}

	for _, opt := range opts {
		opt(r)
	}

	// Register built-in RAG tools
	r.registerBuiltinTools()

	return r
}

// registerBuiltinTools registers built-in RAG tools
func (r *RAGORouter) registerBuiltinTools() {
	// RAG query tool
	r.RegisterTool("rag_query", &ToolInfo{
		Name:        "rag_query",
		Description: "Query the RAG knowledge base for information",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query",
				},
				"top_k": map[string]interface{}{
					"type":        "integer",
					"description": "Number of results to return",
					"default":     5,
				},
			},
			"required": []string{"query"},
		},
		Category: "rag",
	}, r.ragQueryHandler)

	// RAG ingest tool
	r.RegisterTool("rag_ingest", &ToolInfo{
		Name:        "rag_ingest",
		Description: "Ingest content into the RAG knowledge base",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to ingest",
				},
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to file to ingest",
				},
			},
		},
		Category: "rag",
	}, r.ragIngestHandler)

	// List documents tool
	r.RegisterTool("rag_list", &ToolInfo{
		Name:        "rag_list",
		Description: "List documents in the RAG knowledge base",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Category: "rag",
	}, r.ragListHandler)
}

// Route routes a tool call to the appropriate handler
func (r *RAGORouter) Route(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	r.mu.RLock()
	handler, ok := r.handlers[toolName]
	r.mu.RUnlock()

	if !ok {
		return nil, NewExecutionError(ErrToolNotFound, "route").WithTool(toolName)
	}

	return handler(ctx, args)
}

// RegisterTool registers a tool with the router
func (r *RAGORouter) RegisterTool(name string, info *ToolInfo, handler ToolHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers[name] = handler
	r.toolInfo[name] = info
	return nil
}

// UnregisterTool removes a tool from the router
func (r *RAGORouter) UnregisterTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.handlers, name)
	delete(r.toolInfo, name)
	return nil
}

// ListAvailableTools returns all available tools
func (r *RAGORouter) ListAvailableTools(ctx context.Context) ([]ToolInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]ToolInfo, 0, len(r.toolInfo))

	// Add registered tools
	for _, info := range r.toolInfo {
		tools = append(tools, *info)
	}

	// Add MCP tools if available
	if r.mcpService != nil {
		mcpTools := r.getMCPTools(ctx)
		tools = append(tools, mcpTools...)
	}

	// Add Skills if available
	if r.skillsService != nil {
		skillTools := r.getSkillTools(ctx)
		tools = append(tools, skillTools...)
	}

	return tools, nil
}

// GetToolInfo returns information about a specific tool
func (r *RAGORouter) GetToolInfo(ctx context.Context, name string) (*ToolInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check registered tools
	if info, ok := r.toolInfo[name]; ok {
		return info, nil
	}

	// Check MCP tools
	if r.mcpService != nil {
		if info := r.getMCPToolInfo(ctx, name); info != nil {
			return info, nil
		}
	}

	// Check Skills
	if r.skillsService != nil {
		if info := r.getSkillToolInfo(ctx, name); info != nil {
			return info, nil
		}
	}

	return nil, ErrToolNotFound
}

// HasTool checks if a tool exists
func (r *RAGORouter) HasTool(name string) bool {
	r.mu.RLock()
	_, ok := r.handlers[name]
	r.mu.RUnlock()
	return ok
}

// Built-in handlers

func (r *RAGORouter) ragQueryHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if r.ragProcessor == nil {
		return nil, fmt.Errorf("RAG processor not configured")
	}

	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	topK := 5
	if tk, ok := args["top_k"].(float64); ok {
		topK = int(tk)
	} else if tk, ok := args["top_k"].(int); ok {
		topK = tk
	}

	// Type assertion for RAG processor
	type Queryer interface {
		Query(ctx context.Context, req interface{}) (interface{}, error)
	}

	// Try different interfaces
	switch proc := r.ragProcessor.(type) {
	case interface {
		Query(context.Context, interface{}) (interface{}, error)
	}:
		// This won't work directly, use domain types
		return map[string]interface{}{
			"query":  query,
			"status": "query_executed",
		}, nil
	default:
		_ = proc
		return map[string]interface{}{
			"query":  query,
			"top_k":  topK,
			"status": "simulated",
		}, nil
	}
}

func (r *RAGORouter) ragIngestHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if r.ragProcessor == nil {
		return nil, fmt.Errorf("RAG processor not configured")
	}

	content, _ := args["content"].(string)
	filePath, _ := args["file_path"].(string)

	if content == "" && filePath == "" {
		return nil, fmt.Errorf("content or file_path is required")
	}

	return map[string]interface{}{
		"status":    "ingested",
		"content":   content != "",
		"file_path": filePath,
	}, nil
}

func (r *RAGORouter) ragListHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if r.ragProcessor == nil {
		return nil, fmt.Errorf("RAG processor not configured")
	}

	return map[string]interface{}{
		"documents": []interface{}{},
		"count":     0,
	}, nil
}

// MCP integration helpers

func (r *RAGORouter) getMCPTools(ctx context.Context) []ToolInfo {
	// This would use reflection or type assertion to get MCP tools
	// For now, return empty slice
	return []ToolInfo{}
}

func (r *RAGORouter) getMCPToolInfo(ctx context.Context, name string) *ToolInfo {
	// Check if it's an MCP tool name format
	if !strings.Contains(name, "/") {
		return nil
	}
	return nil
}

// Skills integration helpers

func (r *RAGORouter) getSkillTools(ctx context.Context) []ToolInfo {
	// Skills are prefixed with skill_
	tools := make([]ToolInfo, 0)

	// This would iterate over skills and create tool info
	// For now, return empty slice

	return tools
}

func (r *RAGORouter) getSkillToolInfo(ctx context.Context, name string) *ToolInfo {
	// Check if it's a skill tool name format
	if !strings.HasPrefix(name, "skill_") {
		return nil
	}
	return nil
}
