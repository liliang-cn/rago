// Package gateway provides a multi-agent gateway library for AI agent management.
//
// Each agent runs in its own goroutine, allowing concurrent processing
// without blocking the main application.
//
// Example:
//
//	gw, err := gateway.New(ctx, cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create agents
//	gw.CreateAgent("assistant", "You are a helpful assistant.")
//	gw.CreateAgent("coder", "You are a code expert.")
//
//	// Switch to agent
//	gw.SetCurrent("assistant")
//
//	// Send query (non-blocking)
//	respCh, _ := gw.Query(ctx, "Hello!")
//
//	// Or wait for response
//	resp := <-respCh
//	fmt.Println(resp.Content)
package gateway

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liliang-cn/rago/v2/pkg/agent"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/memory"
	"github.com/liliang-cn/rago/v2/pkg/router"
	"github.com/liliang-cn/rago/v2/pkg/services"
	"github.com/liliang-cn/rago/v2/pkg/store"
)

// Option configures a Gateway
type Option func(*Gateway)

// WithAgentDBPath sets a custom database path for agents
func WithAgentDBPath(path string) Option {
	return func(g *Gateway) {
		g.agentDBPath = path
	}
}

// WithMemoryService sets a custom memory service
func WithMemoryService(memSvc domain.MemoryService) Option {
	return func(g *Gateway) {
		g.memorySvc = memSvc
	}
}

// WithRouter sets a custom router service
func WithRouter(routerSvc *router.Service) Option {
	return func(g *Gateway) {
		g.routerSvc = routerSvc
	}
}

// WithResponseCallback sets a callback for agent responses
func WithResponseCallback(cb func(*Response)) Option {
	return func(g *Gateway) {
		g.responseCallback = cb
	}
}

// ========================================
// Query
// ========================================

// Query represents a request to an agent
type Query struct {
	ID        string
	Content   string
	Ctx       context.Context
	cancel    context.CancelFunc
	Timestamp time.Time
}

// ========================================
// Response
// ========================================

// Response represents an agent's response
type Response struct {
	QueryID    string
	AgentName  string
	Content    string
	Error      error
	Done       bool
	Timestamp  time.Time
	Duration   time.Duration
}

// ========================================
// Agent
// ========================================

// Agent represents an AI agent running in its own goroutine
type Agent struct {
	id               string
	name             string
	systemPrompt     string
	service          *agent.Service
	queryCh          chan *Query
	responseChannels map[string]chan *Response
	running          atomic.Bool
	currentQuery     atomic.Value // *Query
	gw               *Gateway
	mu               sync.RWMutex
}

// ID returns the agent's unique ID
func (a *Agent) ID() string {
	return a.id
}

// Name returns the agent's name
func (a *Agent) Name() string {
	return a.name
}

// SystemPrompt returns the agent's system prompt
func (a *Agent) SystemPrompt() string {
	return a.systemPrompt
}

// IsBusy returns true if the agent is currently processing a query
func (a *Agent) IsBusy() bool {
	return a.currentQuery.Load() != nil
}

// Query sends a query to this agent (non-blocking)
// Returns a channel that will receive the response
func (a *Agent) Query(ctx context.Context, content string) (<-chan *Response, error) {
	queryCtx, cancel := context.WithCancel(ctx)

	query := &Query{
		ID:        fmt.Sprintf("qry_%d", time.Now().UnixNano()),
		Content:   content,
		Ctx:       queryCtx,
		cancel:    cancel,
		Timestamp: time.Now(),
	}

	respCh := make(chan *Response, 1)

	// Send query to agent's goroutine
	select {
	case a.queryCh <- query:
		// Register response channel
		a.mu.Lock()
		if a.responseChannels == nil {
			a.responseChannels = make(map[string]chan *Response)
		}
		a.responseChannels[query.ID] = respCh
		a.mu.Unlock()
		return respCh, nil
	case <-time.After(5 * time.Second):
		close(respCh)
		return nil, fmt.Errorf("agent %s: query channel full", a.name)
	}
}

// Ask sends a query and waits for the response (blocking)
func (a *Agent) Ask(ctx context.Context, content string) (*Response, error) {
	respCh, err := a.Query(ctx, content)
	if err != nil {
		return nil, err
	}

	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Cancel cancels the current query if any
func (a *Agent) Cancel() bool {
	if q := a.currentQuery.Load(); q != nil {
		if query, ok := q.(*Query); ok && query != nil && query.cancel != nil {
			query.cancel()
			return true
		}
	}
	return false
}

// Internal fields
type agentInternal struct {
	responseChannels map[string]chan *Response
}

func (a *Agent) responseChannelsMap() map[string]chan *Response {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.responseChannels == nil {
		return nil
	}
	// Return a copy for iteration
	m := make(map[string]chan *Response)
	for k, v := range a.responseChannels {
		m[k] = v
	}
	return m
}

func (a *Agent) getResponseChannel(queryID string) (chan *Response, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.responseChannels == nil {
		return nil, false
	}
	ch, ok := a.responseChannels[queryID]
	return ch, ok
}

func (a *Agent) removeResponseChannel(queryID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.responseChannels != nil {
		delete(a.responseChannels, queryID)
	}
}

func (a *Agent) start(ctx context.Context) {
	a.running.Store(true)
	go a.run(ctx)
}

func (a *Agent) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case query, ok := <-a.queryCh:
			if !ok {
				return
			}
			if query == nil {
				continue
			}

			a.currentQuery.Store(query)
			startTime := time.Now()

			// Run agent - blocks only this goroutine
			result, err := a.service.Run(query.Ctx, query.Content)

			resp := &Response{
				QueryID:   query.ID,
				AgentName: a.name,
				Done:      true,
				Timestamp: time.Now(),
				Duration:  time.Since(startTime),
			}

			if err != nil {
				resp.Error = err
			} else if result.FinalResult != nil {
				if str, ok := result.FinalResult.(string); ok {
					resp.Content = str
				} else {
					resp.Content = fmt.Sprintf("%v", result.FinalResult)
				}
			}

			// Send response
			if ch, ok := a.getResponseChannel(query.ID); ok {
				select {
				case ch <- resp:
				default:
				}
				a.removeResponseChannel(query.ID)
			}

			// Also send to gateway callback
			if a.gw != nil && a.gw.responseCallback != nil {
				go a.gw.responseCallback(resp)
			}

			a.currentQuery.Store((*Query)(nil))
		}
	}
}

// ========================================
// Gateway
// ========================================

// Gateway manages multiple AI agents
type Gateway struct {
	agents           map[string]*Agent
	currentID        atomic.Value // string
	llmPool          *services.GlobalPoolService
	mcpService       *mcp.Service
	embedService     domain.Embedder
	memorySvc        domain.MemoryService
	routerSvc        *router.Service
	responseCallback func(*Response)
	agentDBPath      string
	mu               sync.RWMutex
	ctx              context.Context
	cancelFunc       context.CancelFunc
}

// New creates a new Gateway with shared services
func New(ctx context.Context, cfg *config.Config, opts ...Option) (*Gateway, error) {
	globalPool := services.GetGlobalPoolService()

	llmSvc, err := globalPool.GetLLMService()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM: %w", err)
	}

	embedSvc, err := globalPool.GetEmbeddingService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedder: %w", err)
	}

	// MCP service (shared)
	mcpSvc, err := mcp.NewService(&cfg.MCP, llmSvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP: %w", err)
	}

	if err := mcpSvc.StartServers(ctx, nil); err != nil {
		// Continue anyway
	}

	// Memory service (shared)
	var memSvc domain.MemoryService
	homeDir, _ := os.UserHomeDir()
	memDBPath := filepath.Join(homeDir, ".rago", "data", "memory.db")
	memStore, err := store.NewMemoryStore(memDBPath)
	if err == nil {
		_ = memStore.InitSchema(ctx)
		memSvc = memory.NewService(memStore, llmSvc, embedSvc, memory.DefaultConfig())
	}

	// Router (shared)
	var routerSvc *router.Service
	if rs, err := router.NewService(embedSvc, router.DefaultConfig()); err == nil {
		_ = rs.RegisterDefaultIntents()
		routerSvc = rs
	}

	gwCtx, gwCancel := context.WithCancel(ctx)

	gw := &Gateway{
		agents:       make(map[string]*Agent),
		llmPool:      globalPool,
		mcpService:   mcpSvc,
		embedService: embedSvc,
		memorySvc:    memSvc,
		routerSvc:    routerSvc,
		agentDBPath:  filepath.Join(homeDir, ".rago", "data"),
		ctx:          gwCtx,
		cancelFunc:   gwCancel,
	}

	// Apply options
	for _, opt := range opts {
		opt(gw)
	}

	return gw, nil
}

// CreateAgent creates a new agent with its own goroutine
func (gw *Gateway) CreateAgent(name, systemPrompt string) (*Agent, error) {
	gw.mu.Lock()
	defer gw.mu.Unlock()

	if _, exists := gw.agents[name]; exists {
		return nil, fmt.Errorf("agent %s already exists", name)
	}

	llmSvc, err := gw.llmPool.GetLLMService()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM: %w", err)
	}

	mcpAdapter := &mcpToolAdapter{service: gw.mcpService}

	agentDBPath := filepath.Join(gw.agentDBPath, fmt.Sprintf("agent_%s.db", name))

	agentSvc, err := agent.NewService(llmSvc, mcpAdapter, nil, agentDBPath, gw.memorySvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent service: %w", err)
	}

	if gw.routerSvc != nil {
		agentSvc.SetRouter(gw.routerSvc)
	}

	a := &Agent{
		id:           fmt.Sprintf("agent_%s_%d", name, time.Now().Unix()),
		name:         name,
		systemPrompt: systemPrompt,
		service:      agentSvc,
		queryCh:      make(chan *Query, 10),
		gw:           gw,
	}

	gw.agents[name] = a

	// Start agent goroutine
	a.start(gw.ctx)

	return a, nil
}

// GetAgent returns an agent by name
func (gw *Gateway) GetAgent(name string) (*Agent, bool) {
	gw.mu.RLock()
	defer gw.mu.RUnlock()
	a, ok := gw.agents[name]
	return a, ok
}

// ListAgents returns all agents
func (gw *Gateway) ListAgents() []*Agent {
	gw.mu.RLock()
	defer gw.mu.RUnlock()

	result := make([]*Agent, 0, len(gw.agents))
	for _, a := range gw.agents {
		result = append(result, a)
	}
	return result
}

// SetCurrent sets the current active agent
func (gw *Gateway) SetCurrent(name string) error {
	gw.mu.RLock()
	_, ok := gw.agents[name]
	gw.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent %s not found", name)
	}

	gw.currentID.Store(name)
	return nil
}

// GetCurrent returns the current active agent
func (gw *Gateway) GetCurrent() *Agent {
	name := gw.currentID.Load()
	if name == nil || name == "" {
		return nil
	}

	agentName := name.(string)
	if a, ok := gw.GetAgent(agentName); ok {
		return a
	}
	return nil
}

// Query sends a query to the current agent
func (gw *Gateway) Query(ctx context.Context, content string) (<-chan *Response, error) {
	current := gw.GetCurrent()
	if current == nil {
		return nil, fmt.Errorf("no current agent")
	}
	return current.Query(ctx, content)
}

// Ask sends a query and waits for response
func (gw *Gateway) Ask(ctx context.Context, content string) (*Response, error) {
	current := gw.GetCurrent()
	if current == nil {
		return nil, fmt.Errorf("no current agent")
	}
	return current.Ask(ctx, content)
}

// Close closes the gateway and stops all agents
func (gw *Gateway) Close() error {
	gw.cancelFunc()
	gw.mu.Lock()
	defer gw.mu.Unlock()

	for _, a := range gw.agents {
		close(a.queryCh)
	}

	if gw.mcpService != nil {
		return gw.mcpService.Close()
	}
	return nil
}

// ========================================
// MCP Tool Adapter
// ========================================

type mcpToolAdapter struct {
	service *mcp.Service
}

func (a *mcpToolAdapter) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	result, err := a.service.CallTool(ctx, toolName, args)
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, fmt.Errorf("MCP tool error: %s", result.Error)
	}
	return result.Data, nil
}

func (a *mcpToolAdapter) ListTools() []domain.ToolDefinition {
	tools := a.service.GetAvailableTools(context.Background())
	result := make([]domain.ToolDefinition, 0, len(tools))

	for _, t := range tools {
		var parameters map[string]interface{}
		if t.InputSchema != nil && len(t.InputSchema) > 0 {
			parameters = t.InputSchema
		} else {
			parameters = map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"arguments": map[string]interface{}{
						"type":        "object",
						"description": "Tool arguments",
					},
				},
			}
		}

		result = append(result, domain.ToolDefinition{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  parameters,
			},
		})
	}
	return result
}
