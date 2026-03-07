package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/liliang-cn/agent-go/pkg/domain"
	"github.com/liliang-cn/agent-go/pkg/ptc"
)

// ToolHandler executes a tool call synchronously.
type ToolHandler func(ctx context.Context, args map[string]interface{}) (interface{}, error)

// Tool categories used by ToolRegistry.
const (
	CategoryCustom = "custom" // user-registered via AddTool()
	CategoryRAG    = "rag"    // rag_query, rag_ingest
	CategoryMemory = "memory" // memory_save/recall/update/delete
	CategorySkill  = "skill"  // skill tools (currently managed via ptcRouter)
	CategoryMCP    = "mcp"    // MCP tools (dynamically managed via ptcRouter)
)

type registeredTool struct {
	def      domain.ToolDefinition
	handler  ToolHandler
	category string
}

// ToolRegistry is the single source of truth for tool definitions and handlers.
//
// All modules (custom, RAG, Memory) register here. PTC's callTool() dispatches
// through this registry (via SyncToPTCRouter). This eliminates the dual-registration
// pattern where tools had to be registered both on agent.tools and ptcRouter separately.
type ToolRegistry struct {
	mu               sync.RWMutex
	tools            map[string]*registeredTool
	sessionActivated map[string]map[string]bool // sessionID -> toolName -> bool
}

// NewToolRegistry creates an empty ToolRegistry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:            make(map[string]*registeredTool),
		sessionActivated: make(map[string]map[string]bool),
	}
}

// ActivateForSession marks a deferred tool as active for the given session.
func (r *ToolRegistry) ActivateForSession(sessionID, toolName string) {
	if sessionID == "" || toolName == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sessionActivated[sessionID]; !ok {
		r.sessionActivated[sessionID] = make(map[string]bool)
	}
	r.sessionActivated[sessionID][toolName] = true
}

// SearchDeferredTools searches for deferred tools matching the query.
func (r *ToolRegistry) SearchDeferredTools(query string) []domain.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = strings.ToLower(query)
	keywords := strings.Fields(query)
	var matches []domain.ToolDefinition

	for _, t := range r.tools {
		if t.def.DeferLoading {
			name := strings.ToLower(t.def.Function.Name)
			desc := strings.ToLower(t.def.Function.Description)

			matched := false
			for _, kw := range keywords {
				if strings.Contains(name, kw) || strings.Contains(desc, kw) {
					matched = true
					break
				}
			}

			if matched {
				matches = append(matches, t.def)
			}
		}
	}
	return matches
}

// Register adds (or replaces) a tool. Tools registered here are:
//   - Visible to the LLM in non-PTC mode
//   - Accessible via callTool() inside the PTC JavaScript sandbox

// SearchAllTools searches ALL registered tools (not just deferred ones)
func (r *ToolRegistry) SearchAllTools(query string) []domain.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = strings.ToLower(query)
	keywords := strings.Fields(query)
	var matches []domain.ToolDefinition

	for _, t := range r.tools {
		name := strings.ToLower(t.def.Function.Name)
		desc := strings.ToLower(t.def.Function.Description)

		matched := false
		for _, kw := range keywords {
			if strings.Contains(name, kw) || strings.Contains(desc, kw) {
				matched = true
				break
			}
		}

		if matched {
			matches = append(matches, t.def)
		}
	}
	return matches
}

//
// Register adds (or replaces) a tool. The tool will be:
//   - returned by ListForLLM(false) for native function calling
//   - accessible via callTool() in the PTC JavaScript sandbox after SyncToPTCRouter
func (r *ToolRegistry) Register(def domain.ToolDefinition, handler ToolHandler, category string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[def.Function.Name] = &registeredTool{def: def, handler: handler, category: category}
}

// Unregister removes a tool from the registry.
func (r *ToolRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// Has reports whether a tool is registered.
func (r *ToolRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[name]
	return ok
}

// CategoryOf returns the category of a registered tool, or "" if not found.
func (r *ToolRegistry) CategoryOf(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if t, ok := r.tools[name]; ok {
		return t.category
	}
	return ""
}

// ListForLLM returns the tool definitions that should be passed to the LLM.
//
//   - ptcEnabled=false: all registered tools (they appear as direct function calls)
//   - ptcEnabled=true:  none (tools are hidden; only execute_javascript is shown,
//     which is added separately by PTCIntegration)
func (r *ToolRegistry) ListForLLM(ptcEnabled bool, sessionID string) []domain.ToolDefinition {
	if ptcEnabled {
		// In PTC mode all module tools are hidden from direct LLM function calls;
		// they are only accessible via callTool() inside execute_javascript.
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	activeMap := r.sessionActivated[sessionID]

	out := make([]domain.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		// Include if not deferred, or if explicitly activated for this session
		if !t.def.DeferLoading || (activeMap != nil && activeMap[t.def.Function.Name]) {
			out = append(out, t.def)
		}
	}
	return out
}

// ListForCallTool returns ToolInfos for all tools accessible via callTool() in the
// PTC JavaScript sandbox. Used by GetAvailableCallTools() to build the system prompt.
func (r *ToolRegistry) ListForCallTool() []ptc.ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ptc.ToolInfo, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, ptc.ToolInfo{
			Name:        t.def.Function.Name,
			Description: t.def.Function.Description,
			Parameters:  t.def.Function.Parameters,
			Category:    t.category,
		})
	}
	return out
}

// Call dispatches a tool call to the registered handler.
func (r *ToolRegistry) Call(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	r.mu.RLock()
	tool, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("tool %q not found in registry", name)
	}
	if tool.handler == nil {
		return nil, fmt.Errorf("tool %q has no handler", name)
	}
	return tool.handler(ctx, args)
}

// SyncToPTCRouter registers all tools from this registry into the given PTC router
// so that callTool() inside the JavaScript sandbox can find and execute them.
// This is called once during PTC setup after all module tools have been registered.
func (r *ToolRegistry) SyncToPTCRouter(router *ptc.AgentGoRouter) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for name, t := range r.tools {
		toolName := name
		handler := t.handler
		info := &ptc.ToolInfo{
			Name:        t.def.Function.Name,
			Description: t.def.Function.Description,
			Parameters:  t.def.Function.Parameters,
			Category:    t.category,
		}
		// Ignore errors: tool may already be registered (idempotent)
		_ = router.RegisterTool(toolName, info, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return handler(ctx, args)
		})
	}
}
