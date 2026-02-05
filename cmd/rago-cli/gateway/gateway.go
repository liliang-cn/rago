package gateway

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/liliang-cn/rago/v2/pkg/agent"
	"github.com/liliang-cn/rago/v2/pkg/config"
	"github.com/liliang-cn/rago/v2/pkg/domain"
	"github.com/liliang-cn/rago/v2/pkg/mcp"
	"github.com/liliang-cn/rago/v2/pkg/memory"
	"github.com/liliang-cn/rago/v2/pkg/router"
	"github.com/liliang-cn/rago/v2/pkg/services"
	"github.com/liliang-cn/rago/v2/pkg/store"
	"github.com/spf13/cobra"
)

var (
	cfg     *config.Config
	verbose bool
)

// Styles
var styles = struct {
	title    lipgloss.Style
	header   lipgloss.Style
	prompt   lipgloss.Style
	response lipgloss.Style
	success  lipgloss.Style
	error    lipgloss.Style
	warning  lipgloss.Style
	dim      lipgloss.Style
	muted    lipgloss.Style
	agent    lipgloss.Style
}{
	title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
	header:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("38")),
	prompt:   lipgloss.NewStyle().Foreground(lipgloss.Color("99")),
	response: lipgloss.NewStyle().Foreground(lipgloss.Color("43")),
	success:  lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
	error:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
	warning:  lipgloss.NewStyle().Foreground(lipgloss.Color("226")),
	dim:      lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	muted:    lipgloss.NewStyle().Faint(true),
	agent:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213")),
}

// GatewayCmd represents the gateway command
var GatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Run an interactive AI gateway with multi-agent support",
	Long: `Gateway mode runs an async AI reception desk with multi-agent support.

Features:
  - Multiple named agents sharing LLM pool
  - Concurrent request processing
  - Independent conversation history per agent
  - Agent switch/create commands

Commands:
  /agent list           - List all agents
  /agent new <name>     - Create new agent
  /agent switch <name>  - Switch to agent
  /agent current        - Show current agent
  /status               - Show status
  /cancel               - Cancel current task
  /quit                 - Exit`,
	RunE: runGateway,
}

func init() {
	GatewayCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
}

func SetSharedVariables(c *config.Config, v bool) {
	cfg = c
	verbose = v
}

// ========================================
// Request/Response System
// ========================================

type Request struct {
	ID      string
	AgentID string
	Query   string
	Ctx     context.Context
	Cancel  context.CancelFunc
	ReplyCh chan *Response
}

type Response struct {
	AgentID string
	Content string
	Error   error
	Done    bool
}

// ========================================
// Agent Instance
// ========================================

type AgentInstance struct {
	ID         string
	Name       string
	SystemPrompt string
	Service    *agent.Service
	History    []domain.Message
	CreatedAt  time.Time
	mu         sync.RWMutex
}

func (a *AgentInstance) AddMessage(msg domain.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.History = append(a.History, msg)
}

func (a *AgentInstance) GetHistory() []domain.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]domain.Message{}, a.History...)
}

func (a *AgentInstance) SetHistory(history []domain.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.History = history
}

// ========================================
// Agent Manager
// ========================================

type AgentManager struct {
	agents      map[string]*AgentInstance
	currentID   atomic.Value // string
	llmPool     *services.GlobalPoolService
	mcpService  *mcp.Service
	embedService domain.Embedder
	memoryService domain.MemoryService
	routerService *router.Service
	mu          sync.RWMutex
}

func NewAgentManager(ctx context.Context, cfg *config.Config) (*AgentManager, error) {
	globalPool := services.GetGlobalPoolService()

	llmSvc, err := globalPool.GetLLMService()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM: %w", err)
	}

	embedSvc, err := globalPool.GetEmbeddingService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedder: %w", err)
	}

	// MCP service (shared by all agents)
	mcpSvc, err := mcp.NewService(&cfg.MCP, llmSvc)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP service: %w", err)
	}

	if err := mcpSvc.StartServers(ctx, nil); err != nil {
		// Continue anyway
	}

	// Memory service (shared)
	homeDir, _ := os.UserHomeDir()
	memDBPath := filepath.Join(homeDir, ".rago", "data", "memory.db")
	var memSvc domain.MemoryService
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

	am := &AgentManager{
		agents:       make(map[string]*AgentInstance),
		llmPool:      globalPool,
		mcpService:   mcpSvc,
		embedService: embedSvc,
		memoryService: memSvc,
		routerService: routerSvc,
	}

	// Create default agent
	if _, err := am.CreateAgent(ctx, "default", "You are a helpful AI assistant."); err != nil {
		return nil, err
	}
	// Set default as current
	am.currentID.Store("default")

	return am, nil
}

func (am *AgentManager) CreateAgent(ctx context.Context, name, systemPrompt string) (*AgentInstance, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.agents[name]; exists {
		return nil, fmt.Errorf("agent %s already exists", name)
	}

	// Create agent service (each gets its own instance)
	llmSvc, err := am.llmPool.GetLLMService()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM: %w", err)
	}

	mcpAdapter := &mcpToolAdapter{service: am.mcpService}

	homeDir, _ := os.UserHomeDir()
	agentDBPath := filepath.Join(homeDir, ".rago", "data", fmt.Sprintf("agent_%s.db", name))

	agentSvc, err := agent.NewService(llmSvc, mcpAdapter, nil, agentDBPath, am.memoryService)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	// Set router
	if am.routerService != nil {
		agentSvc.SetRouter(am.routerService)
	}

	inst := &AgentInstance{
		ID:           fmt.Sprintf("agent_%s_%d", name, time.Now().Unix()),
		Name:         name,
		SystemPrompt: systemPrompt,
		Service:      agentSvc,
		History:      []domain.Message{},
		CreatedAt:    time.Now(),
	}

	am.agents[name] = inst
	return inst, nil
}

func (am *AgentManager) GetAgent(name string) (*AgentInstance, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	a, ok := am.agents[name]
	return a, ok
}

func (am *AgentManager) ListAgents() []*AgentInstance {
	am.mu.RLock()
	defer am.mu.RUnlock()

	result := make([]*AgentInstance, 0, len(am.agents))
	for _, a := range am.agents {
		result = append(result, a)
	}
	return result
}

func (am *AgentManager) SetCurrent(name string) error {
	am.mu.RLock()
	_, ok := am.agents[name]
	am.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent %s not found", name)
	}

	am.currentID.Store(name)
	return nil
}

func (am *AgentManager) GetCurrent() *AgentInstance {
	name := am.currentID.Load()
	if name == nil || name == "" {
		return nil
	}

	agentName := name.(string)
	if a, ok := am.GetAgent(agentName); ok {
		return a
	}
	return nil
}

// ========================================
// Gateway Service
// ========================================

type Gateway struct {
	agentMgr    *AgentManager
	requestCh   chan *Request
	activeReqs  atomic.Value // map[string]*Request
	running     atomic.Bool
	workerCount int
}

func NewGateway(ctx context.Context, cfg *config.Config) (*Gateway, error) {
	agentMgr, err := NewAgentManager(ctx, cfg)
	if err != nil {
		return nil, err
	}

	workerCount := 3 // Allow concurrent processing
	gw := &Gateway{
		agentMgr:    agentMgr,
		requestCh:   make(chan *Request, 50),
		workerCount: workerCount,
	}
	gw.running.Store(true)
	gw.activeReqs.Store(make(map[string]*Request))

	// Start worker pool
	for i := 0; i < workerCount; i++ {
		go gw.worker(ctx, i)
	}
	// Start response listener
	go gw.responseListener(ctx)

	return gw, nil
}

func (gw *Gateway) worker(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-gw.requestCh:
			if req == nil {
				return
			}
			gw.processRequest(req, workerID)
		}
	}
}

func (gw *Gateway) processRequest(req *Request, workerID int) {
	agentInst := gw.agentMgr.GetCurrent()
	if agentInst == nil {
		resp := &Response{
			Error: fmt.Errorf("no active agent"),
			Done:  true,
		}
		req.ReplyCh <- resp
		return
	}

	req.AgentID = agentInst.ID
	fmt.Println(styles.dim.Render(fmt.Sprintf("[Worker %d → %s: %s]", workerID, agentInst.Name, req.ID)))

	result, err := agentInst.Service.Run(req.Ctx, req.Query)

	resp := &Response{AgentID: agentInst.ID}
	if err != nil {
		resp.Error = err
	} else if result.FinalResult != nil {
		if str, ok := result.FinalResult.(string); ok {
			resp.Content = str
		} else {
			resp.Content = fmt.Sprintf("%v", result.FinalResult)
		}
	} else {
		resp.Content = "Done"
	}
	resp.Done = true

	select {
	case req.ReplyCh <- resp:
	case <-req.Ctx.Done():
	}
}

func (gw *Gateway) Submit(ctx context.Context, query string) (*Request, error) {
	reqCtx, cancel := context.WithCancel(ctx)

	req := &Request{
		ID:      fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Query:   query,
		Ctx:     reqCtx,
		Cancel:  cancel,
		ReplyCh: make(chan *Response, 1),
	}

	// Add to active requests
	active := gw.activeReqs.Load().(map[string]*Request)
	newActive := make(map[string]*Request)
	for k, v := range active {
		newActive[k] = v
	}
	newActive[req.ID] = req
	gw.activeReqs.Store(newActive)

	select {
	case gw.requestCh <- req:
		return req, nil
	case <-ctx.Done():
		cancel()
		return nil, ctx.Err()
	}
}

func (gw *Gateway) CancelCurrent() bool {
	active := gw.activeReqs.Load().(map[string]*Request)
	cancelled := false
	for _, req := range active {
		req.Cancel()
		cancelled = true
	}
	// Clear active requests
	gw.activeReqs.Store(make(map[string]*Request))
	return cancelled
}

func (gw *Gateway) GetActiveCount() int {
	active := gw.activeReqs.Load().(map[string]*Request)
	return len(active)
}

// ========================================
// Interactive Loop
// ========================================

func (gw *Gateway) Run(ctx context.Context) error {
	fmt.Println(styles.header.Render("╔══════════════════════════════════════════════════╗"))
	fmt.Println(styles.header.Render("║" + styles.title.Render("           RAGO GATEWAY - MULTI-AGENT            ") + "║"))
	fmt.Println(styles.header.Render("╚══════════════════════════════════════════════════╝"))
	fmt.Println()
	gw.printAgentInfo()
	fmt.Println(styles.muted.Render("Type /help for commands"))

	// Input loop
	scanner := bufio.NewScanner(os.Stdin)

	for gw.running.Load() {
		select {
		case <-ctx.Done():
			return nil
		default:
			prompt := gw.buildPrompt()
			fmt.Print(prompt)

			if !scanner.Scan() {
				return nil
			}

			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				continue
			}

			if gw.handleCommand(ctx, input) {
				continue
			}

			// Submit request
			req, err := gw.Submit(ctx, input)
			if err != nil {
				fmt.Println(styles.error.Render(fmt.Sprintf("Error: %v", err)))
			} else {
				fmt.Println(styles.dim.Render(fmt.Sprintf("[→ %s queued]", req.ID)))
			}
		}
	}
	return nil
}

func (gw *Gateway) buildPrompt() string {
	current := gw.agentMgr.GetCurrent()
	if current == nil {
		return styles.prompt.Render("> ")
	}
	activeCount := gw.GetActiveCount()
	status := ""
	if activeCount > 0 {
		status = styles.dim.Render(fmt.Sprintf("[%d running]", activeCount))
	}
	return fmt.Sprintf("%s %s> ", styles.agent.Render(current.Name+":"), status)
}

func (gw *Gateway) printAgentInfo() {
	current := gw.agentMgr.GetCurrent()
	if current != nil {
		fmt.Printf("Current: %s | Workers: %d\n", styles.agent.Render(current.Name), gw.workerCount)
	}
}

func (gw *Gateway) responseListener(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			active := gw.activeReqs.Load().(map[string]*Request)

			for id, req := range active {
				select {
				case resp := <-req.ReplyCh:
					gw.printResponse(resp)
					// Remove from active
					newActive := make(map[string]*Request)
					for k, v := range active {
						if k != id {
							newActive[k] = v
						}
					}
					gw.activeReqs.Store(newActive)
					return // One per tick
				default:
				}
			}
		}
	}
}

func (gw *Gateway) printResponse(resp *Response) {
	agentName := "unknown"
	if resp.AgentID != "" {
		for _, a := range gw.agentMgr.ListAgents() {
			if a.ID == resp.AgentID {
				agentName = a.Name
				break
			}
		}
	}

	if resp.Error != nil {
		fmt.Println(styles.error.Render(fmt.Sprintf("[%s] Error: %v", agentName, resp.Error)))
	} else if resp.Content != "" {
		fmt.Printf("%s %s\n", styles.agent.Render("["+agentName+"]:"), styles.response.Render(resp.Content))
	}
}

func (gw *Gateway) handleCommand(ctx context.Context, input string) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/quit", "/exit", "/q":
		gw.running.Store(false)
		fmt.Println(styles.success.Render("Goodbye!"))
		os.Exit(0)
		return true

	case "/cancel", "/c":
		if gw.CancelCurrent() {
			fmt.Println(styles.warning.Render("[All requests cancelled]"))
		} else {
			fmt.Println(styles.dim.Render("[No active requests]"))
		}
		return true

	case "/status", "/s":
		activeCount := gw.GetActiveCount()
		if activeCount > 0 {
			fmt.Println(styles.warning.Render(fmt.Sprintf("[%d requests processing]", activeCount)))
		} else {
			fmt.Println(styles.success.Render("[Idle]"))
		}
		gw.printAgentInfo()
		return true

	case "/agent", "/a":
		gw.handleAgentCommand(ctx, args)
		return true

	case "/help", "/h", "?":
		printHelp()
		return true

	default:
		return false
	}
}

func (gw *Gateway) handleAgentCommand(ctx context.Context, args []string) {
	if len(args) == 0 {
		gw.printAgentList()
		return
	}

	subCmd := args[0]

	switch subCmd {
	case "list", "ls":
		gw.printAgentList()

	case "new", "create", "n":
		if len(args) < 2 {
			fmt.Println(styles.error.Render("Usage: /agent new <name> [prompt]"))
			return
		}
		name := args[1]
		prompt := "You are a helpful AI assistant."
		if len(args) > 2 {
			prompt = strings.Join(args[2:], " ")
		}
		inst, err := gw.agentMgr.CreateAgent(ctx, name, prompt)
		if err != nil {
			fmt.Println(styles.error.Render(fmt.Sprintf("Failed: %v", err)))
			return
		}
		fmt.Println(styles.success.Render(fmt.Sprintf("[Agent '%s' created: %s]", name, inst.ID)))

	case "switch", "use", "s":
		if len(args) < 2 {
			fmt.Println(styles.error.Render("Usage: /agent switch <name>"))
			return
		}
		name := args[1]
		if err := gw.agentMgr.SetCurrent(name); err != nil {
			fmt.Println(styles.error.Render(fmt.Sprintf("Failed: %v", err)))
			return
		}
		fmt.Println(styles.success.Render(fmt.Sprintf("[Switched to agent '%s']", name)))

	case "current", "curr":
		current := gw.agentMgr.GetCurrent()
		if current == nil {
			fmt.Println(styles.dim.Render("[No current agent]"))
		} else {
			fmt.Printf("Current: %s (ID: %s, Messages: %d)\n",
				styles.agent.Render(current.Name),
				current.ID,
				len(current.History))
		}

	default:
		fmt.Println(styles.error.Render(fmt.Sprintf("Unknown agent command: %s", subCmd)))
	}
}

func (gw *Gateway) printAgentList() {
	agents := gw.agentMgr.ListAgents()
	current := gw.agentMgr.GetCurrent()

	fmt.Println(styles.header.Render("\nAgents:"))
	for _, a := range agents {
		prefix := "  "
		if current != nil && current.ID == a.ID {
			prefix = styles.success.Render("* ")
		}
		fmt.Printf("%s%s (%s) - %d messages\n",
			prefix,
			styles.agent.Render(a.Name),
			a.ID[:8]+"...",
			len(a.History))
	}
	fmt.Println()
}

func printHelp() {
	fmt.Println(styles.header.Render("\nCommands:"))
	fmt.Println("  /help, /h, ?              - Show this help")
	fmt.Println("  /status, /s                - Show status")
	fmt.Println("  /cancel, /c                - Cancel all requests")
	fmt.Println()
	fmt.Println(styles.header.Render("Agent Commands:"))
	fmt.Println("  /agent list, /a ls         - List all agents")
	fmt.Println("  /agent new <name> [prompt] - Create new agent")
	fmt.Println("  /agent switch <name>       - Switch to agent")
	fmt.Println("  /agent current             - Show current agent")
	fmt.Println()
	fmt.Println(styles.header.Render("Other:"))
	fmt.Println("  /quit, /q                  - Exit gateway")
	fmt.Println()
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

// ========================================
// Command Entry Point
// ========================================

func runGateway(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	gw, err := NewGateway(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create gateway: %w", err)
	}

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		gw.running.Store(false)
		fmt.Println("\n" + styles.success.Render("Shutting down..."))
		os.Exit(0)
	}()

	return gw.Run(ctx)
}
